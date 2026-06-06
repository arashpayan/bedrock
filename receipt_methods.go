package bedrock

import (
	"fmt"
	"time"
)

// CreateReceipt creates a new receipt for a contribution. memo is the
// treasurer's free-form note; pass an empty string when no note is wanted.
func (db *DB) CreateReceipt(customerID ID, soldAt time.Time, memo string) (*Receipt, error) {
	// Get assembly timezone to generate HumanID
	var assemblyTimezone string
	err := db.conn.Get(&assemblyTimezone, "SELECT timezone FROM assembly LIMIT 1")
	if err != nil {
		return nil, fmt.Errorf("failed to get assembly timezone: %w", err)
	}

	// Parse the timezone
	location, err := time.LoadLocation(assemblyTimezone)
	if err != nil {
		return nil, fmt.Errorf("failed to parse assembly timezone %s: %w", assemblyTimezone, err)
	}

	// Generate a unique HumanID using current system time in the assembly timezone.
	humanID, err := db.nextReceiptHumanID(location)
	if err != nil {
		return nil, err
	}

	// Verify customer exists
	_, err = db.Party(customerID)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	query, args := db.sq.Insert("receipts").
		SetMap(map[string]any{
			"human_id":    humanID,
			"customer_id": customerID,
			"sold_at":     soldAt.Round(0),
			"memo":        memo,
		}).
		Suffix("RETURNING *").
		MustSql()

	receipt := Receipt{}
	if err := db.conn.Get(&receipt, query, args...); err != nil {
		return nil, fmt.Errorf("failed to insert receipt: %w", err)
	}

	return &receipt, nil
}

// CreateReceiptWithItems creates a receipt together with all of its line items
// in a single database transaction. If any item fails to insert, the receipt
// is rolled back and no state is left behind. memo is the treasurer's
// free-form note; pass an empty string when no note is wanted.
func (db *DB) CreateReceiptWithItems(customerID ID, soldAt time.Time, memo string, items []ReceiptItemInput) (*Receipt, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("at least one receipt item is required")
	}

	// Validate all items have positive prices in a consistent currency
	currency := items[0].Price.Currency
	for i, item := range items {
		if item.Price.Amount <= 0 {
			return nil, fmt.Errorf("item %d: price must be positive, got %d cents", i, item.Price.Amount)
		}
		if item.Price.Currency != currency {
			return nil, fmt.Errorf("item %d: currency %s does not match %s", i, item.Price.Currency, currency)
		}
	}

	// Get assembly timezone to generate HumanID
	var assemblyTimezone string
	if err := db.conn.Get(&assemblyTimezone, "SELECT timezone FROM assembly LIMIT 1"); err != nil {
		return nil, fmt.Errorf("failed to get assembly timezone: %w", err)
	}
	location, err := time.LoadLocation(assemblyTimezone)
	if err != nil {
		return nil, fmt.Errorf("failed to parse assembly timezone %s: %w", assemblyTimezone, err)
	}
	humanID, err := db.nextReceiptHumanID(location)
	if err != nil {
		return nil, err
	}

	tx, err := db.conn.Beginx()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	receiptQuery, receiptArgs := db.sq.Insert("receipts").
		SetMap(map[string]any{
			"human_id":    humanID,
			"customer_id": customerID,
			"sold_at":     soldAt.Round(0),
			"memo":        memo,
		}).
		Suffix("RETURNING *").
		MustSql()

	receipt := Receipt{}
	if err := tx.Get(&receipt, receiptQuery, receiptArgs...); err != nil {
		return nil, fmt.Errorf("failed to insert receipt: %w", err)
	}

	for i, item := range items {
		itemQuery, itemArgs := db.sq.Insert("receipt_items").
			SetMap(map[string]any{
				"receipt_id":  receipt.ID,
				"item_id":     item.ItemID,
				"description": item.Description,
				"price":       item.Price.Amount,
				"currency":    item.Price.Currency,
			}).
			MustSql()
		if _, err := tx.Exec(itemQuery, itemArgs...); err != nil {
			return nil, fmt.Errorf("failed to insert receipt item %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &receipt, nil
}

// nextReceiptHumanID returns a human-readable receipt ID — the current time in
// the assembly's timezone formatted as 20060102150405.000 — that is not already
// taken. Because that format has only millisecond resolution, two receipts
// created within the same millisecond would otherwise collide on the
// receipts.human_id UNIQUE constraint. The common (no-collision) path costs a
// single COUNT; on the rare collision it waits a millisecond for the clock to
// advance and tries again.
func (db *DB) nextReceiptHumanID(location *time.Location) (string, error) {
	for range 1000 {
		humanID := time.Now().In(location).Format("20060102150405.000")
		var count int
		if err := db.conn.Get(&count, "SELECT COUNT(*) FROM receipts WHERE human_id = ?", humanID); err != nil {
			return "", fmt.Errorf("failed to check receipt id uniqueness: %w", err)
		}
		if count == 0 {
			return humanID, nil
		}
		time.Sleep(time.Millisecond)
	}
	return "", fmt.Errorf("failed to generate a unique receipt id")
}

// Receipt retrieves a receipt by ID
func (db *DB) Receipt(id ID) (*Receipt, error) {
	var receipt Receipt

	query, args := db.sq.Select("id", "human_id", "customer_id", "sold_at", "transaction_id", "memo", "created_at", "modified_at").
		From("receipts").
		Where("id = ?", id).
		MustSql()

	err := db.conn.Get(&receipt, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt: %w", err)
	}

	return &receipt, nil
}

// FullReceipt loads a receipt together with its customer, resolved line items,
// and computed total — everything required to render a receipt PDF without
// further database access. The total's currency is taken from the receipt's
// line items, falling back to the assembly's default currency for an itemless
// receipt.
func (db *DB) FullReceipt(receiptID ID) (*FullReceipt, error) {
	receipt, err := db.Receipt(receiptID)
	if err != nil {
		return nil, err
	}

	customer, err := db.Party(receipt.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to load receipt customer: %w", err)
	}

	rawItems, err := db.ReceiptItems(receiptID)
	if err != nil {
		return nil, err
	}

	var currency Currency
	var totalAmount int64
	items := make([]FullReceiptItem, 0, len(rawItems))
	for i, ri := range rawItems {
		if i == 0 {
			currency = ri.Price.Currency
		}
		totalAmount += ri.Price.Amount
		item, err := db.Item(ri.ItemID)
		if err != nil {
			return nil, fmt.Errorf("failed to load item %d for receipt: %w", ri.ItemID, err)
		}
		items = append(items, FullReceiptItem{Item: *item, ReceiptItem: ri})
	}

	if currency == "" {
		assembly, err := db.Assembly()
		if err != nil {
			return nil, fmt.Errorf("failed to load assembly for receipt currency: %w", err)
		}
		currency = assembly.DefaultCurrency
	}

	return &FullReceipt{
		Receipt:  *receipt,
		Customer: *customer,
		Items:    items,
		Total:    NewMoney(totalAmount, currency),
	}, nil
}

// ReceiptByHumanID retrieves a receipt by its human-readable ID
func (db *DB) ReceiptByHumanID(humanID string) (*Receipt, error) {
	var receipt Receipt

	query, args := db.sq.Select("id", "human_id", "customer_id", "sold_at", "transaction_id", "memo", "created_at", "modified_at").
		From("receipts").
		Where("human_id = ?", humanID).
		MustSql()

	err := db.conn.Get(&receipt, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt by human ID: %w", err)
	}

	return &receipt, nil
}

// ReceiptsByCustomer retrieves all receipts for a specific customer
func (db *DB) ReceiptsByCustomer(customerID ID) ([]Receipt, error) {
	var receipts []Receipt

	query, args := db.sq.Select("id", "human_id", "customer_id", "sold_at", "transaction_id", "memo", "created_at", "modified_at").
		From("receipts").
		Where("customer_id = ?", customerID).
		OrderBy("sold_at DESC").
		MustSql()

	err := db.conn.Select(&receipts, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipts by customer: %w", err)
	}

	return receipts, nil
}

// ReceiptsByTransaction retrieves all receipts for a specific transaction
func (db *DB) ReceiptsByTransaction(transactionID ID) ([]Receipt, error) {
	var receipts []Receipt

	query, args := db.sq.Select("id", "human_id", "customer_id", "sold_at", "transaction_id", "memo", "created_at", "modified_at").
		From("receipts").
		Where("transaction_id = ?", transactionID).
		OrderBy("sold_at DESC").
		MustSql()

	err := db.conn.Select(&receipts, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipts by transaction: %w", err)
	}

	return receipts, nil
}

// UndepositedReceipts retrieves all receipts that haven't been deposited yet
func (db *DB) UndepositedReceipts() ([]Receipt, error) {
	var receipts []Receipt

	query, args := db.sq.Select("id", "human_id", "customer_id", "sold_at", "transaction_id", "memo", "created_at", "modified_at").
		From("receipts").
		Where("transaction_id IS NULL").
		OrderBy("sold_at DESC").
		MustSql()

	err := db.conn.Select(&receipts, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get undeposited receipts: %w", err)
	}

	return receipts, nil
}

// UnsentReceipts returns receipts that have no successful email delivery,
// optionally filtered by sold-at date range and/or customer. Results are
// ordered by customer then sold date so callers can group a contributor's
// outstanding receipts into a single email.
func (db *DB) UnsentReceipts(opts UnsentReceiptsOptions) ([]Receipt, error) {
	q := db.sq.Select("id", "human_id", "customer_id", "sold_at", "transaction_id", "memo", "created_at", "modified_at").
		From("receipts").
		Where("NOT EXISTS (SELECT 1 FROM receipt_deliveries d WHERE d.receipt_id = receipts.id AND d.status = ?)", DeliveryStatusSuccess)

	if opts.StartDate != nil {
		q = q.Where("sold_at >= ?", *opts.StartDate)
	}
	if opts.EndDate != nil {
		q = q.Where("sold_at <= ?", *opts.EndDate)
	}
	if opts.CustomerID != nil {
		q = q.Where("customer_id = ?", *opts.CustomerID)
	}

	query, args := q.OrderBy("customer_id ASC", "sold_at ASC").MustSql()

	var receipts []Receipt
	if err := db.conn.Select(&receipts, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get unsent receipts: %w", err)
	}
	return receipts, nil
}

// Receipts retrieves all receipts
func (db *DB) Receipts() ([]Receipt, error) {
	var receipts []Receipt

	query, args := db.sq.Select("id", "human_id", "customer_id", "sold_at", "transaction_id", "memo", "created_at", "modified_at").
		From("receipts").
		OrderBy("sold_at DESC").
		MustSql()

	err := db.conn.Select(&receipts, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list receipts: %w", err)
	}

	return receipts, nil
}

// AssignReceiptToTransaction assigns a receipt to a deposit transaction
func (db *DB) AssignReceiptToTransaction(receiptID, transactionID ID) (*Receipt, error) {
	// Verify transaction exists and is a deposit (positive amount)
	var amount int64
	err := db.conn.Get(&amount, "SELECT amount FROM transactions WHERE id = ?", transactionID)
	if err != nil {
		return nil, fmt.Errorf("transaction not found: %w", err)
	}
	if amount <= 0 {
		return nil, fmt.Errorf("transaction %d is not a deposit (amount: %d)", transactionID, amount)
	}

	query, args := db.sq.Update("receipts").
		SetMap(map[string]any{
			"transaction_id": transactionID,
		}).
		Where("id = ?", receiptID).
		Suffix("RETURNING *").
		MustSql()

	var receipt Receipt
	if err := db.conn.Get(&receipt, query, args...); err != nil {
		return nil, fmt.Errorf("failed to assign receipt to transaction: %w", err)
	}

	return &receipt, nil
}

// UnassignReceiptFromTransaction removes a receipt's transaction assignment
func (db *DB) UnassignReceiptFromTransaction(receiptID ID) (*Receipt, error) {
	query, args := db.sq.Update("receipts").
		SetMap(map[string]any{
			"transaction_id": nil,
		}).
		Where("id = ?", receiptID).
		Suffix("RETURNING *").
		MustSql()

	var receipt Receipt
	if err := db.conn.Get(&receipt, query, args...); err != nil {
		return nil, fmt.Errorf("failed to unassign receipt from transaction: %w", err)
	}

	return &receipt, nil
}

// DeleteReceipt deletes a receipt by ID
func (db *DB) DeleteReceipt(id ID) error {
	// Check if receipt has associated receipt items
	var itemCount int
	err := db.conn.Get(&itemCount, "SELECT COUNT(*) FROM receipt_items WHERE receipt_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to check receipt items: %w", err)
	}
	if itemCount > 0 {
		return fmt.Errorf("cannot delete receipt with %d associated items", itemCount)
	}

	query, args := db.sq.Delete("receipts").
		Where("id = ?", id).
		MustSql()

	result, err := db.conn.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete receipt: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("receipt with id %d not found", id)
	}

	return nil
}

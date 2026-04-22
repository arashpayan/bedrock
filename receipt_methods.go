package bedrock

import (
	"fmt"
	"time"
)

// CreateReceipt creates a new receipt for a contribution
func (db *DB) CreateReceipt(customerID ID, soldAt time.Time) (*Receipt, error) {
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

	// Generate HumanID using current system time in assembly timezone format 20060102150405.000
	humanID := time.Now().In(location).Format("20060102150405.000")

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
// is rolled back and no state is left behind.
func (db *DB) CreateReceiptWithItems(customerID ID, soldAt time.Time, items []ReceiptItemInput) (*Receipt, error) {
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
	humanID := time.Now().In(location).Format("20060102150405.000")

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

// Receipt retrieves a receipt by ID
func (db *DB) Receipt(id ID) (*Receipt, error) {
	var receipt Receipt

	query, args := db.sq.Select("id", "human_id", "customer_id", "sold_at", "transaction_id", "created_at", "modified_at").
		From("receipts").
		Where("id = ?", id).
		MustSql()

	err := db.conn.Get(&receipt, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt: %w", err)
	}

	return &receipt, nil
}

// ReceiptByHumanID retrieves a receipt by its human-readable ID
func (db *DB) ReceiptByHumanID(humanID string) (*Receipt, error) {
	var receipt Receipt

	query, args := db.sq.Select("id", "human_id", "customer_id", "sold_at", "transaction_id", "created_at", "modified_at").
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

	query, args := db.sq.Select("id", "human_id", "customer_id", "sold_at", "transaction_id", "created_at", "modified_at").
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

	query, args := db.sq.Select("id", "human_id", "customer_id", "sold_at", "transaction_id", "created_at", "modified_at").
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

	query, args := db.sq.Select("id", "human_id", "customer_id", "sold_at", "transaction_id", "created_at", "modified_at").
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

// Receipts retrieves all receipts
func (db *DB) Receipts() ([]Receipt, error) {
	var receipts []Receipt

	query, args := db.sq.Select("id", "human_id", "customer_id", "sold_at", "transaction_id", "created_at", "modified_at").
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

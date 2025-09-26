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

	// Generate HumanID using the format 20060102150405.999 in assembly timezone
	humanID := soldAt.In(location).Format("20060102150405.000")

	// Verify customer exists
	_, err = db.Party(customerID)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	receipt := &Receipt{
		HumanID:    humanID,
		CustomerID: customerID,
		SoldAt:     soldAt,
	}

	query, args := db.sq.Insert("receipts").
		SetMap(map[string]interface{}{
			"human_id":    humanID,
			"customer_id": customerID,
			"sold_at":     soldAt,
		}).
		Suffix("RETURNING id, created_at, modified_at").
		MustSql()

	err = db.conn.QueryRow(query, args...).Scan(&receipt.ID, &receipt.CreatedAt, &receipt.ModifiedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert receipt: %w", err)
	}

	return receipt, nil
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
		SetMap(map[string]interface{}{
			"transaction_id": transactionID,
		}).
		Where("id = ?", receiptID).
		Suffix("RETURNING id, human_id, customer_id, sold_at, transaction_id, created_at, modified_at").
		MustSql()

	var receipt Receipt
	err = db.conn.QueryRow(query, args...).Scan(&receipt.ID, &receipt.HumanID, &receipt.CustomerID, &receipt.SoldAt, &receipt.TransactionID, &receipt.CreatedAt, &receipt.ModifiedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to assign receipt to transaction: %w", err)
	}

	return &receipt, nil
}

// UnassignReceiptFromTransaction removes a receipt's transaction assignment
func (db *DB) UnassignReceiptFromTransaction(receiptID ID) (*Receipt, error) {
	query, args := db.sq.Update("receipts").
		SetMap(map[string]interface{}{
			"transaction_id": nil,
		}).
		Where("id = ?", receiptID).
		Suffix("RETURNING id, human_id, customer_id, sold_at, transaction_id, created_at, modified_at").
		MustSql()

	var receipt Receipt
	err := db.conn.QueryRow(query, args...).Scan(&receipt.ID, &receipt.HumanID, &receipt.CustomerID, &receipt.SoldAt, &receipt.TransactionID, &receipt.CreatedAt, &receipt.ModifiedAt)
	if err != nil {
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

package bedrock

import (
	"fmt"
)

// CreateReceiptItem creates a new receipt item for a receipt
func (db *DB) CreateReceiptItem(receiptID ID, itemID ID, description string, price Money) (*ReceiptItem, error) {
	// Verify receipt exists
	_, err := db.Receipt(receiptID)
	if err != nil {
		return nil, fmt.Errorf("receipt not found: %w", err)
	}

	// Verify item exists
	_, err = db.Item(itemID)
	if err != nil {
		return nil, fmt.Errorf("item not found: %w", err)
	}

	// Validate price
	if price.Amount <= 0 {
		return nil, fmt.Errorf("price must be positive")
	}

	query, args := db.sq.Insert("receipt_items").
		SetMap(map[string]any{
			"receipt_id":  receiptID,
			"item_id":     itemID,
			"description": description,
			"price":       price.Amount,
			"currency":    price.Currency,
		}).
		Suffix("RETURNING id, receipt_id, item_id, description, price, currency, created_at, modified_at").
		MustSql()

	var ri ReceiptItem
	var priceAmount int64
	var currency Currency
	if err := db.conn.QueryRow(query, args...).Scan(
		&ri.ID,
		&ri.ReceiptID,
		&ri.ItemID,
		&ri.Description,
		&priceAmount,
		&currency,
		&ri.CreatedAt,
		&ri.ModifiedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to insert receipt item: %w", err)
	}

	ri.Price = Money{Amount: priceAmount, Currency: currency}

	return &ri, nil
}

// DeleteReceiptItem deletes a receipt item by ID
func (db *DB) DeleteReceiptItem(id ID) error {
	query, args := db.sq.Delete("receipt_items").
		Where("id = ?", id).
		MustSql()

	result, err := db.conn.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete receipt item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("receipt item with id %d not found", id)
	}

	return nil
}

// ReceiptItem retrieves a receipt item by ID
func (db *DB) ReceiptItem(id ID) (*ReceiptItem, error) {
	query, args := db.sq.Select("id", "receipt_id", "item_id", "description", "price", "currency", "created_at", "modified_at").
		From("receipt_items").
		Where("id = ?", id).
		MustSql()

	var ri ReceiptItem
	var priceAmount int64
	var currency Currency
	if err := db.conn.QueryRow(query, args...).Scan(
		&ri.ID,
		&ri.ReceiptID,
		&ri.ItemID,
		&ri.Description,
		&priceAmount,
		&currency,
		&ri.CreatedAt,
		&ri.ModifiedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to get receipt item: %w", err)
	}

	ri.Price = Money{Amount: priceAmount, Currency: currency}

	return &ri, nil
}

// ReceiptItems retrieves all receipt items for a receipt
func (db *DB) ReceiptItems(receiptID ID) ([]ReceiptItem, error) {
	query, args := db.sq.Select("id", "receipt_id", "item_id", "description", "price", "currency", "created_at", "modified_at").
		From("receipt_items").
		Where("receipt_id = ?", receiptID).
		OrderBy("id ASC").
		MustSql()

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query receipt items: %w", err)
	}
	defer rows.Close()

	var items []ReceiptItem
	for rows.Next() {
		var ri ReceiptItem
		var priceAmount int64
		var currency Currency
		if err := rows.Scan(
			&ri.ID,
			&ri.ReceiptID,
			&ri.ItemID,
			&ri.Description,
			&priceAmount,
			&currency,
			&ri.CreatedAt,
			&ri.ModifiedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan receipt item: %w", err)
		}
		ri.Price = Money{Amount: priceAmount, Currency: currency}
		items = append(items, ri)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating receipt items: %w", err)
	}

	return items, nil
}

// UpdateReceiptItem updates a receipt item
func (db *DB) UpdateReceiptItem(id ID, description string, price Money) (*ReceiptItem, error) {
	// Validate price
	if price.Amount <= 0 {
		return nil, fmt.Errorf("price must be positive")
	}

	query, args := db.sq.Update("receipt_items").
		SetMap(map[string]any{
			"description": description,
			"price":       price.Amount,
			"currency":    price.Currency,
		}).
		Where("id = ?", id).
		Suffix("RETURNING id, receipt_id, item_id, description, price, currency, created_at, modified_at").
		MustSql()

	var ri ReceiptItem
	var priceAmount int64
	var currency Currency
	if err := db.conn.QueryRow(query, args...).Scan(
		&ri.ID,
		&ri.ReceiptID,
		&ri.ItemID,
		&ri.Description,
		&priceAmount,
		&currency,
		&ri.CreatedAt,
		&ri.ModifiedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to update receipt item: %w", err)
	}

	ri.Price = Money{Amount: priceAmount, Currency: currency}

	return &ri, nil
}

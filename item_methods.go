package bedrock

import (
	"fmt"
)

// CreateItem creates a new item (contribution category)
func (db *DB) CreateItem(name string) (*Item, error) {
	if name == "" {
		return nil, fmt.Errorf("item name cannot be empty")
	}

	query, args := db.sq.Insert("items").
		SetMap(map[string]any{
			"name": name,
		}).
		Suffix("RETURNING *").
		MustSql()

	item := Item{}
	if err := db.conn.Get(&item, query, args...); err != nil {
		return nil, fmt.Errorf("failed to insert item: %w", err)
	}

	return &item, nil
}

// Item retrieves an item by ID
func (db *DB) Item(id ID) (*Item, error) {
	var item Item

	query, args := db.sq.Select("id", "name", "created_at", "modified_at").
		From("items").
		Where("id = ?", id).
		MustSql()

	err := db.conn.Get(&item, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	return &item, nil
}

// ItemByName retrieves an item by name
func (db *DB) ItemByName(name string) (*Item, error) {
	var item Item

	query, args := db.sq.Select("id", "name", "created_at", "modified_at").
		From("items").
		Where("name = ?", name).
		MustSql()

	err := db.conn.Get(&item, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get item by name: %w", err)
	}

	return &item, nil
}

// ListItems retrieves all items
func (db *DB) ListItems() ([]Item, error) {
	var items []Item

	query, args := db.sq.Select("id", "name", "created_at", "modified_at").
		From("items").
		OrderBy("name ASC").
		MustSql()

	err := db.conn.Select(&items, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}

	return items, nil
}

// UpdateItem updates an item's name
func (db *DB) UpdateItem(id ID, name string) (*Item, error) {
	if name == "" {
		return nil, fmt.Errorf("item name cannot be empty")
	}

	query, args := db.sq.Update("items").
		SetMap(map[string]any{
			"name": name,
		}).
		Where("id = ?", id).
		Suffix("RETURNING *").
		MustSql()

	var item Item
	if err := db.conn.Get(&item, query, args...); err != nil {
		return nil, fmt.Errorf("failed to update item: %w", err)
	}

	return &item, nil
}

// DeleteItem deletes an item by ID
func (db *DB) DeleteItem(id ID) error {
	// Check for dependencies
	var count int
	if err := db.conn.Get(&count, "SELECT COUNT(*) FROM receipt_items WHERE item_id = ?", id); err != nil {
		return fmt.Errorf("failed to check item dependencies: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete item: %d receipt items are using this item", count)
	}

	query, args := db.sq.Delete("items").
		Where("id = ?", id).
		MustSql()

	result, err := db.conn.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("item with id %d not found", id)
	}

	return nil
}

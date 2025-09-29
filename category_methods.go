package bedrock

import (
	"fmt"
)

// CreateCategory creates a new expense category
func (db *DB) CreateCategory(name string) (*Category, error) {
	query, args := db.sq.Insert("categories").
		SetMap(map[string]interface{}{
			"name": name,
		}).
		Suffix("RETURNING *").
		MustSql()

	category := Category{}
	if err := db.conn.Get(&category, query, args...); err != nil {
		return nil, fmt.Errorf("failed to insert category: %w", err)
	}

	return &category, nil
}

// Category retrieves a category by ID
func (db *DB) Category(id ID) (*Category, error) {
	query, args := db.sq.Select("*").
		From("categories").
		Where("id = ?", id).
		MustSql()

	category := Category{}
	if err := db.conn.Get(&category, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	return &category, nil
}

// CategoryByName retrieves a category by name
func (db *DB) CategoryByName(name string) (*Category, error) {
	query, args := db.sq.Select("*").
		From("categories").
		Where("name = ?", name).
		MustSql()

	category := Category{}
	if err := db.conn.Get(&category, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get category by name: %w", err)
	}

	return &category, nil
}

// Categories retrieves all categories
func (db *DB) Categories() ([]Category, error) {
	query, args := db.sq.Select("*").
		From("categories").
		OrderBy("name").
		MustSql()

	categories := []Category{}
	if err := db.conn.Select(&categories, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	return categories, nil
}

// UpdateCategory updates an existing category
func (db *DB) UpdateCategory(id ID, name string) (*Category, error) {
	query, args := db.sq.Update("categories").
		SetMap(map[string]interface{}{
			"name": name,
		}).
		Where("id = ?", id).
		Suffix("RETURNING *").
		MustSql()

	category := Category{}
	if err := db.conn.Get(&category, query, args...); err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	return &category, nil
}

// DeleteCategory deletes a category if it has no associated expenses
func (db *DB) DeleteCategory(id ID) error {
	// Check for dependencies
	var count int
	if err := db.conn.Get(&count, "SELECT COUNT(*) FROM expenses WHERE category_id = ?", id); err != nil {
		return fmt.Errorf("failed to check category dependencies: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete category: %d expenses are using this category", count)
	}

	query, args := db.sq.Delete("categories").
		Where("id = ?", id).
		MustSql()

	result, err := db.conn.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("category not found")
	}

	return nil
}

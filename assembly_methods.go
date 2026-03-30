package bedrock

import (
	"fmt"
	"time"
)

// createAssembly creates a new Spiritual Assembly in the database (internal use only)
func (db *DB) createAssembly(name string, timezone *time.Location, defaultCurrency Currency) (*Assembly, error) {
	if name == "" {
		return nil, fmt.Errorf("assembly name cannot be empty")
	}
	if timezone == nil {
		return nil, fmt.Errorf("timezone cannot be nil")
	}

	// Check if assembly already exists (only one assembly per database)
	var count int
	if err := db.conn.Get(&count, "SELECT COUNT(*) FROM assembly"); err != nil {
		return nil, fmt.Errorf("failed to check existing assembly: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("assembly already exists in this database")
	}

	query, args := db.sq.Insert("assembly").
		SetMap(map[string]any{
			"name":             name,
			"timezone":         timezone.String(),
			"default_currency": defaultCurrency,
		}).
		Suffix("RETURNING *").
		MustSql()

	var assembly Assembly
	var timezoneStr string
	if err := db.conn.QueryRow(query, args...).Scan(
		&assembly.ID,
		&assembly.Name,
		&timezoneStr,
		&assembly.DefaultCurrency,
		&assembly.CreatedAt,
		&assembly.ModifiedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to insert assembly: %w", err)
	}

	// Parse timezone back from string
	parsedTimezone, err := time.LoadLocation(timezoneStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timezone: %w", err)
	}
	assembly.Timezone = parsedTimezone

	return &assembly, nil
}

// Assembly retrieves the Spiritual Assembly from the database
func (db *DB) Assembly() (*Assembly, error) {
	var assembly Assembly
	var timezoneStr string

	if err := db.conn.QueryRow("SELECT id, name, timezone, default_currency, created_at, modified_at FROM assembly LIMIT 1").Scan(
		&assembly.ID,
		&assembly.Name,
		&timezoneStr,
		&assembly.DefaultCurrency,
		&assembly.CreatedAt,
		&assembly.ModifiedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to get assembly: %w", err)
	}

	// Parse timezone from string
	parsedTimezone, err := time.LoadLocation(timezoneStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timezone: %w", err)
	}
	assembly.Timezone = parsedTimezone

	return &assembly, nil
}

// UpdateAssembly updates the assembly information
func (db *DB) UpdateAssembly(name string, timezone *time.Location, defaultCurrency Currency) (*Assembly, error) {
	if name == "" {
		return nil, fmt.Errorf("assembly name cannot be empty")
	}
	if timezone == nil {
		return nil, fmt.Errorf("timezone cannot be nil")
	}
	if defaultCurrency != CurrencyUSD && defaultCurrency != CurrencyCAD {
		return nil, fmt.Errorf("invalid currency: %s", defaultCurrency)
	}

	query, args := db.sq.Update("assembly").
		SetMap(map[string]any{
			"name":             name,
			"timezone":         timezone.String(),
			"default_currency": defaultCurrency,
		}).
		Suffix("RETURNING *").
		MustSql()

	var assembly Assembly
	var timezoneStr string
	if err := db.conn.QueryRow(query, args...).Scan(
		&assembly.ID,
		&assembly.Name,
		&timezoneStr,
		&assembly.DefaultCurrency,
		&assembly.CreatedAt,
		&assembly.ModifiedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to update assembly: %w", err)
	}

	// Parse timezone back from string
	parsedTimezone, err := time.LoadLocation(timezoneStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timezone: %w", err)
	}
	assembly.Timezone = parsedTimezone

	return &assembly, nil
}

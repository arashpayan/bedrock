package bedrock

import (
	"fmt"
	"time"
)

// assemblyColumns is the canonical column list for reading an assembly row.
// The order must match the destinations in scanAssembly.
const assemblyColumns = "id, name, timezone, default_currency, " +
	"mailing_address, charitable_reg_number, contact_email, contact_phone, " +
	"receipt_disclaimer, in_kind_receipt_disclaimer, created_at, modified_at"

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
		Suffix("RETURNING " + assemblyColumns).
		MustSql()

	assembly, err := scanAssembly(db.conn.QueryRow(query, args...))
	if err != nil {
		return nil, fmt.Errorf("failed to insert assembly: %w", err)
	}
	return assembly, nil
}

// Assembly retrieves the Spiritual Assembly from the database
func (db *DB) Assembly() (*Assembly, error) {
	assembly, err := scanAssembly(db.conn.QueryRow("SELECT " + assemblyColumns + " FROM assembly LIMIT 1"))
	if err != nil {
		return nil, fmt.Errorf("failed to get assembly: %w", err)
	}
	return assembly, nil
}

// UpdateAssembly updates the assembly's core information (name, timezone, and
// default currency). Issuer details are updated separately via
// UpdateAssemblyDetails.
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
		Suffix("RETURNING " + assemblyColumns).
		MustSql()

	assembly, err := scanAssembly(db.conn.QueryRow(query, args...))
	if err != nil {
		return nil, fmt.Errorf("failed to update assembly: %w", err)
	}
	return assembly, nil
}

// UpdateAssemblyDetails updates the issuer fields printed on contribution
// receipts. All fields are optional and may be empty strings. inKindDisclaimer
// is the wording printed on gift-in-kind receipts.
func (db *DB) UpdateAssemblyDetails(mailingAddress, charitableRegNumber, contactEmail, contactPhone, receiptDisclaimer, inKindDisclaimer string) (*Assembly, error) {
	query, args := db.sq.Update("assembly").
		SetMap(map[string]any{
			"mailing_address":            mailingAddress,
			"charitable_reg_number":      charitableRegNumber,
			"contact_email":              contactEmail,
			"contact_phone":              contactPhone,
			"receipt_disclaimer":         receiptDisclaimer,
			"in_kind_receipt_disclaimer": inKindDisclaimer,
		}).
		Suffix("RETURNING " + assemblyColumns).
		MustSql()

	assembly, err := scanAssembly(db.conn.QueryRow(query, args...))
	if err != nil {
		return nil, fmt.Errorf("failed to update assembly details: %w", err)
	}
	return assembly, nil
}

// scanAssembly reads a single assembly row whose columns are in assemblyColumns
// order, parsing the stored timezone string back into a *time.Location.
func scanAssembly(row interface{ Scan(...any) error }) (*Assembly, error) {
	var assembly Assembly
	var timezoneStr string
	if err := row.Scan(
		&assembly.ID,
		&assembly.Name,
		&timezoneStr,
		&assembly.DefaultCurrency,
		&assembly.MailingAddress,
		&assembly.CharitableRegNumber,
		&assembly.ContactEmail,
		&assembly.ContactPhone,
		&assembly.ReceiptDisclaimer,
		&assembly.InKindDisclaimer,
		&assembly.CreatedAt,
		&assembly.ModifiedAt,
	); err != nil {
		return nil, err
	}

	parsedTimezone, err := time.LoadLocation(timezoneStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timezone: %w", err)
	}
	assembly.Timezone = parsedTimezone

	return &assembly, nil
}

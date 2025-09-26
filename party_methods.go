package bedrock

import (
	"fmt"
)

// CreateParty creates a new party (contributor or vendor)
func (db *DB) CreateParty(name string, emailAddress, bahaiIDNumber, address, telephoneNumber *string) (*Party, error) {
	if name == "" {
		return nil, fmt.Errorf("party name cannot be empty")
	}

	party := &Party{
		Name:            name,
		EmailAddress:    emailAddress,
		BahaiIDNumber:   bahaiIDNumber,
		Address:         address,
		TelephoneNumber: telephoneNumber,
	}

	query, args := db.sq.Insert("parties").
		SetMap(map[string]interface{}{
			"name":             name,
			"email_address":    emailAddress,
			"bahai_id_number":  bahaiIDNumber,
			"address":          address,
			"telephone_number": telephoneNumber,
		}).
		Suffix("RETURNING id, created_at, modified_at").
		MustSql()

	err := db.conn.QueryRow(query, args...).Scan(&party.ID, &party.CreatedAt, &party.ModifiedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert party: %w", err)
	}

	return party, nil
}

// Party retrieves a party by ID
func (db *DB) Party(id ID) (*Party, error) {
	var party Party

	query, args := db.sq.Select("id", "name", "email_address", "bahai_id_number", "address", "telephone_number", "created_at", "modified_at").
		From("parties").
		Where("id = ?", id).
		MustSql()

	err := db.conn.Get(&party, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get party: %w", err)
	}

	return &party, nil
}

// PartyByName retrieves a party by name
func (db *DB) PartyByName(name string) (*Party, error) {
	var party Party

	query, args := db.sq.Select("id", "name", "email_address", "bahai_id_number", "address", "telephone_number", "created_at", "modified_at").
		From("parties").
		Where("name = ?", name).
		MustSql()

	err := db.conn.Get(&party, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get party by name: %w", err)
	}

	return &party, nil
}

// PartyByEmail retrieves a party by email address
func (db *DB) PartyByEmail(email string) (*Party, error) {
	var party Party

	query, args := db.sq.Select("id", "name", "email_address", "bahai_id_number", "address", "telephone_number", "created_at", "modified_at").
		From("parties").
		Where("email_address = ?", email).
		MustSql()

	err := db.conn.Get(&party, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get party by email: %w", err)
	}

	return &party, nil
}

// PartyByBahaiID retrieves a party by Bahá'í ID number
func (db *DB) PartyByBahaiID(bahaiID string) (*Party, error) {
	var party Party

	query, args := db.sq.Select("id", "name", "email_address", "bahai_id_number", "address", "telephone_number", "created_at", "modified_at").
		From("parties").
		Where("bahai_id_number = ?", bahaiID).
		MustSql()

	err := db.conn.Get(&party, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get party by Bahai ID: %w", err)
	}

	return &party, nil
}

// ListParties retrieves all parties
func (db *DB) ListParties() ([]Party, error) {
	var parties []Party

	query, args := db.sq.Select("id", "name", "email_address", "bahai_id_number", "address", "telephone_number", "created_at", "modified_at").
		From("parties").
		OrderBy("name ASC").
		MustSql()

	err := db.conn.Select(&parties, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list parties: %w", err)
	}

	return parties, nil
}

// SearchParties searches parties by name (partial match)
func (db *DB) SearchParties(namePattern string) ([]Party, error) {
	var parties []Party

	query, args := db.sq.Select("id", "name", "email_address", "bahai_id_number", "address", "telephone_number", "created_at", "modified_at").
		From("parties").
		Where("name LIKE ?", "%"+namePattern+"%").
		OrderBy("name ASC").
		MustSql()

	err := db.conn.Select(&parties, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search parties: %w", err)
	}

	return parties, nil
}

// UpdateParty updates a party's information
func (db *DB) UpdateParty(id ID, name string, emailAddress, bahaiIDNumber, address, telephoneNumber *string) (*Party, error) {
	if name == "" {
		return nil, fmt.Errorf("party name cannot be empty")
	}

	query, args := db.sq.Update("parties").
		SetMap(map[string]interface{}{
			"name":             name,
			"email_address":    emailAddress,
			"bahai_id_number":  bahaiIDNumber,
			"address":          address,
			"telephone_number": telephoneNumber,
		}).
		Where("id = ?", id).
		Suffix("RETURNING id, name, email_address, bahai_id_number, address, telephone_number, created_at, modified_at").
		MustSql()

	var party Party
	err := db.conn.QueryRow(query, args...).Scan(&party.ID, &party.Name, &party.EmailAddress, &party.BahaiIDNumber, &party.Address, &party.TelephoneNumber, &party.CreatedAt, &party.ModifiedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update party: %w", err)
	}

	return &party, nil
}

// DeleteParty deletes a party by ID
func (db *DB) DeleteParty(id ID) error {
	query, args := db.sq.Delete("parties").
		Where("id = ?", id).
		MustSql()

	result, err := db.conn.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete party: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("party with id %d not found", id)
	}

	return nil
}

package bedrock

import (
	"fmt"
)

// CreateBankAccount creates a new bank account.
// Only earmark accounts can have a parent. Earmarks must have a checking or savings parent.
// If openingBalance has a positive amount, a deposit transaction is created with memo "Opening Balance".
func (db *DB) CreateBankAccount(name string, accountType AccountType, currency Currency, parentID *ID, description string, isActive bool, openingBalance Money) (*BankAccount, error) {
	if name == "" {
		return nil, fmt.Errorf("account name cannot be empty")
	}

	// Only earmark accounts can have a parent
	if parentID != nil && accountType != AccountTypeEarmark {
		return nil, fmt.Errorf("only earmark accounts can have a parent account")
	}

	// Validate opening balance
	if openingBalance.Amount < 0 {
		return nil, fmt.Errorf("opening balance cannot be negative")
	}
	if openingBalance.Amount > 0 && openingBalance.Currency != currency {
		return nil, fmt.Errorf("opening balance currency %s does not match account currency %s", openingBalance.Currency, currency)
	}

	// Earmark accounts must have a parent that is checking or savings
	var parent *BankAccount
	if accountType == AccountTypeEarmark {
		if parentID == nil {
			return nil, fmt.Errorf("earmark accounts must have a parent account")
		}
		var err error
		parent, err = db.BankAccount(*parentID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve parent account: %w", err)
		}
		if parent.AccountType != AccountTypeChecking && parent.AccountType != AccountTypeSavings {
			return nil, fmt.Errorf("earmark accounts must have a checking or savings parent, not %s", parent.AccountType)
		}
		// Earmarks must inherit parent's currency
		if currency != parent.Currency {
			return nil, fmt.Errorf("earmark accounts must use the same currency as their parent (%s)", parent.Currency)
		}
		if err := db.validateParentAccount(*parentID, nil); err != nil {
			return nil, err
		}
	}

	query, args := db.sq.Insert("bank_accounts").
		SetMap(map[string]any{
			"parent_id":    parentID,
			"name":         name,
			"account_type": accountType,
			"currency":     currency,
			"description":  description,
			"is_active":    isActive,
		}).
		Suffix("RETURNING *").
		MustSql()

	account := BankAccount{}
	if err := db.conn.Get(&account, query, args...); err != nil {
		return nil, fmt.Errorf("failed to insert bank account: %w", err)
	}

	// Create opening balance transaction if amount is positive
	if openingBalance.Amount > 0 {
		_, err := db.CreateDeposit(account.ID, openingBalance, TransactionMethodElectronicTransfer, "Opening Balance", account.CreatedAt)
		if err != nil {
			// Rollback by deleting the account
			_ = db.DeleteBankAccount(account.ID)
			return nil, fmt.Errorf("failed to create opening balance transaction: %w", err)
		}
	}

	return &account, nil
}

// BankAccount retrieves a bank account by ID
func (db *DB) BankAccount(id ID) (*BankAccount, error) {
	var account BankAccount

	query, args := db.sq.Select("id", "parent_id", "name", "account_type", "currency", "description", "is_active", "created_at", "modified_at").
		From("bank_accounts").
		Where("id = ?", id).
		MustSql()

	err := db.conn.Get(&account, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get bank account: %w", err)
	}

	return &account, nil
}

// BankAccountByName retrieves a bank account by name
func (db *DB) BankAccountByName(name string) (*BankAccount, error) {
	var account BankAccount

	query, args := db.sq.Select("id", "parent_id", "name", "account_type", "currency", "description", "is_active", "created_at", "modified_at").
		From("bank_accounts").
		Where("name = ?", name).
		MustSql()

	err := db.conn.Get(&account, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get bank account by name: %w", err)
	}

	return &account, nil
}

// RootBankAccounts retrieves all top-level bank accounts (no parent)
func (db *DB) RootBankAccounts() ([]BankAccount, error) {
	var accounts []BankAccount

	query, args := db.sq.Select("id", "parent_id", "name", "account_type", "currency", "description", "is_active", "created_at", "modified_at").
		From("bank_accounts").
		Where("parent_id IS NULL").
		OrderBy("name ASC").
		MustSql()

	err := db.conn.Select(&accounts, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get root bank accounts: %w", err)
	}

	return accounts, nil
}

// ChildBankAccounts retrieves all child accounts of a parent account
func (db *DB) ChildBankAccounts(parentID ID) ([]BankAccount, error) {
	var accounts []BankAccount

	query, args := db.sq.Select("id", "parent_id", "name", "account_type", "currency", "description", "is_active", "created_at", "modified_at").
		From("bank_accounts").
		Where("parent_id = ?", parentID).
		OrderBy("name ASC").
		MustSql()

	err := db.conn.Select(&accounts, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get child bank accounts: %w", err)
	}

	return accounts, nil
}

// BankAccounts retrieves all bank accounts
func (db *DB) BankAccounts() ([]BankAccount, error) {
	var accounts []BankAccount

	query, args := db.sq.Select("id", "parent_id", "name", "account_type", "currency", "description", "is_active", "created_at", "modified_at").
		From("bank_accounts").
		OrderBy("name ASC").
		MustSql()

	err := db.conn.Select(&accounts, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list bank accounts: %w", err)
	}

	return accounts, nil
}

// ActiveBankAccounts retrieves all active bank accounts
func (db *DB) ActiveBankAccounts() ([]BankAccount, error) {
	var accounts []BankAccount

	query, args := db.sq.Select("id", "parent_id", "name", "account_type", "currency", "description", "is_active", "created_at", "modified_at").
		From("bank_accounts").
		Where("is_active = ?", true).
		OrderBy("name ASC").
		MustSql()

	err := db.conn.Select(&accounts, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get active bank accounts: %w", err)
	}

	return accounts, nil
}

// UpdateBankAccount updates a bank account's information
func (db *DB) UpdateBankAccount(id ID, name string, accountType AccountType, currency Currency, parentID *ID, description string, isActive bool) (*BankAccount, error) {
	if name == "" {
		return nil, fmt.Errorf("account name cannot be empty")
	}

	// If parentID is provided, validate it exists and check for cycles
	if parentID != nil {
		if err := db.validateParentAccount(*parentID, &id); err != nil {
			return nil, err
		}
	}

	query, args := db.sq.Update("bank_accounts").
		SetMap(map[string]interface{}{
			"parent_id":    parentID,
			"name":         name,
			"account_type": accountType,
			"currency":     currency,
			"description":  description,
			"is_active":    isActive,
		}).
		Where("id = ?", id).
		Suffix("RETURNING *").
		MustSql()

	var account BankAccount
	if err := db.conn.Get(&account, query, args...); err != nil {
		return nil, fmt.Errorf("failed to update bank account: %w", err)
	}

	return &account, nil
}

// DeactivateBankAccount marks a bank account as inactive
func (db *DB) DeactivateBankAccount(id ID) (*BankAccount, error) {
	query, args := db.sq.Update("bank_accounts").
		SetMap(map[string]interface{}{
			"is_active": false,
		}).
		Where("id = ?", id).
		Suffix("RETURNING *").
		MustSql()

	var account BankAccount
	if err := db.conn.Get(&account, query, args...); err != nil {
		return nil, fmt.Errorf("failed to deactivate bank account: %w", err)
	}

	return &account, nil
}

// DeleteBankAccount deletes a bank account by ID
func (db *DB) DeleteBankAccount(id ID) error {
	// Check if account has child accounts
	childCount, err := db.getChildAccountCount(id)
	if err != nil {
		return fmt.Errorf("failed to check child accounts: %w", err)
	}
	if childCount > 0 {
		return fmt.Errorf("cannot delete account with %d child accounts", childCount)
	}

	// Check if account has transactions
	transactionCount, err := db.getAccountTransactionCount(id)
	if err != nil {
		return fmt.Errorf("failed to check transactions: %w", err)
	}
	if transactionCount > 0 {
		return fmt.Errorf("cannot delete account with %d transactions", transactionCount)
	}

	query, args := db.sq.Delete("bank_accounts").
		Where("id = ?", id).
		MustSql()

	result, err := db.conn.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete bank account: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("bank account with id %d not found", id)
	}

	return nil
}

// validateParentAccount checks if the parent account exists and prevents cycles
func (db *DB) validateParentAccount(parentID ID, excludeID *ID) error {
	// Check if parent account exists
	_, err := db.BankAccount(parentID)
	if err != nil {
		return fmt.Errorf("parent account not found: %w", err)
	}

	// If we're updating an existing account, check for cycles
	if excludeID != nil {
		if parentID == *excludeID {
			return fmt.Errorf("account cannot be its own parent")
		}

		// Walk up the parent chain to detect cycles
		currentParentID := &parentID
		for currentParentID != nil {
			if *currentParentID == *excludeID {
				return fmt.Errorf("setting parent would create a cycle in account hierarchy")
			}

			// Get the next parent in the chain
			var nextParentID *ID
			err := db.conn.Get(&nextParentID, "SELECT parent_id FROM bank_accounts WHERE id = ?", *currentParentID)
			if err != nil {
				return fmt.Errorf("failed to check parent chain: %w", err)
			}
			currentParentID = nextParentID
		}
	}

	return nil
}

// getChildAccountCount returns the number of child accounts for a given account
func (db *DB) getChildAccountCount(accountID ID) (int, error) {
	var count int
	err := db.conn.Get(&count, "SELECT COUNT(*) FROM bank_accounts WHERE parent_id = ?", accountID)
	return count, err
}

// getAccountTransactionCount returns the number of transactions for a given account
func (db *DB) getAccountTransactionCount(accountID ID) (int, error) {
	var count int
	err := db.conn.Get(&count, "SELECT COUNT(*) FROM transactions WHERE account_id = ?", accountID)
	return count, err
}

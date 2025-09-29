package bedrock

import (
	"fmt"
	"time"
)

// CreateDeposit creates a new deposit transaction
func (db *DB) CreateDeposit(accountID ID, amount Money, method TransactionMethod, memo string, transactedAt time.Time) (*Transaction, error) {
	if amount.Amount <= 0 {
		return nil, fmt.Errorf("deposit amount must be positive, got %d cents", amount.Amount)
	}

	// Validate account currency matches amount currency
	var accountCurrency Currency
	err := db.conn.Get(&accountCurrency, "SELECT currency FROM bank_accounts WHERE id = ?", accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account currency: %w", err)
	}
	if accountCurrency != amount.Currency {
		return nil, fmt.Errorf("amount currency %s does not match account currency %s", amount.Currency, accountCurrency)
	}

	query, args := db.sq.Insert("transactions").
		SetMap(map[string]any{
			"account_id":    accountID,
			"amount":        amount.Amount,
			"memo":          memo,
			"method":        method,
			"transacted_at": transactedAt,
		}).
		Suffix("RETURNING *").
		MustSql()

	transaction := Transaction{}
	if err := db.conn.Get(&transaction, query, args...); err != nil {
		return nil, fmt.Errorf("failed to insert deposit transaction: %w", err)
	}

	return &transaction, nil
}

// CreateWithdrawal creates a new withdrawal transaction with associated expenses
func (db *DB) CreateWithdrawal(accountID ID, payeeID ID, method TransactionMethod, memo string, transactedAt time.Time, checkNumber *string, expenses []ExpenseItem) (*Transaction, error) {
	if len(expenses) == 0 {
		return nil, fmt.Errorf("at least one expense item is required for withdrawal")
	}

	// Calculate total amount from expenses and validate currencies
	var totalAmount int64
	var currency Currency
	for i, expense := range expenses {
		if expense.Amount.Amount <= 0 {
			return nil, fmt.Errorf("expense %d amount must be positive, got %d cents", i, expense.Amount.Amount)
		}
		if i == 0 {
			currency = expense.Amount.Currency
		} else if expense.Amount.Currency != currency {
			return nil, fmt.Errorf("all expenses must have the same currency, got %s and %s", currency, expense.Amount.Currency)
		}
		totalAmount += expense.Amount.Amount
	}

	// Validate account currency matches expense currency
	var accountCurrency Currency
	err := db.conn.Get(&accountCurrency, "SELECT currency FROM bank_accounts WHERE id = ?", accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account currency: %w", err)
	}
	if accountCurrency != currency {
		return nil, fmt.Errorf("expense currency %s does not match account currency %s", currency, accountCurrency)
	}

	// Start transaction
	tx, err := db.conn.Beginx()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Create the withdrawal transaction
	query, args := db.sq.Insert("transactions").
		SetMap(map[string]interface{}{
			"account_id":    accountID,
			"amount":        -totalAmount,
			"payee_id":      payeeID,
			"memo":          memo,
			"method":        method,
			"transacted_at": transactedAt,
			"check_number":  checkNumber,
		}).
		Suffix("RETURNING *").
		MustSql()

	transaction := Transaction{}
	if err := tx.Get(&transaction, query, args...); err != nil {
		return nil, fmt.Errorf("failed to insert withdrawal transaction: %w", err)
	}

	// Create expense records
	for _, expense := range expenses {
		expenseQuery, expenseArgs := db.sq.Insert("expenses").
			SetMap(map[string]interface{}{
				"transaction_id": transaction.ID,
				"category_id":    expense.CategoryID,
				"description":    expense.Description,
				"amount":         expense.Amount.Amount,
				"currency":       expense.Amount.Currency,
			}).
			MustSql()

		if _, err := tx.Exec(expenseQuery, expenseArgs...); err != nil {
			return nil, fmt.Errorf("failed to insert expense: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &transaction, nil
}

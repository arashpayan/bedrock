package bedrock

import (
	"fmt"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
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

// CreateDepositWithReceipts creates a deposit transaction and assigns the
// given receipts to it in a single database transaction. All receipts must be
// currently undeposited (transaction_id IS NULL). If any step fails the entire
// operation is rolled back.
func (db *DB) CreateDepositWithReceipts(accountID ID, amount Money, method TransactionMethod, memo string, transactedAt time.Time, receiptIDs []ID) (*Transaction, error) {
	if amount.Amount <= 0 {
		return nil, fmt.Errorf("deposit amount must be positive, got %d cents", amount.Amount)
	}
	if len(receiptIDs) == 0 {
		return nil, fmt.Errorf("at least one receipt is required")
	}

	// Validate account currency matches amount currency
	var accountCurrency Currency
	if err := db.conn.Get(&accountCurrency, "SELECT currency FROM bank_accounts WHERE id = ?", accountID); err != nil {
		return nil, fmt.Errorf("failed to get account currency: %w", err)
	}
	if accountCurrency != amount.Currency {
		return nil, fmt.Errorf("amount currency %s does not match account currency %s", amount.Currency, accountCurrency)
	}

	tx, err := db.conn.Beginx()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Ensure every requested receipt exists and is undeposited. Any receipt
	// already assigned to a transaction is rejected.
	existsQuery, existsArgs, err := sqlx.In("SELECT id FROM receipts WHERE id IN (?) AND transaction_id IS NULL", receiptIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to build receipt lookup: %w", err)
	}
	var availableIDs []ID
	if err := tx.Select(&availableIDs, existsQuery, existsArgs...); err != nil {
		return nil, fmt.Errorf("failed to check receipts: %w", err)
	}
	if len(availableIDs) != len(receiptIDs) {
		return nil, fmt.Errorf("expected %d undeposited receipts, found %d", len(receiptIDs), len(availableIDs))
	}

	// Insert the deposit transaction
	depositQuery, depositArgs := db.sq.Insert("transactions").
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
	if err := tx.Get(&transaction, depositQuery, depositArgs...); err != nil {
		return nil, fmt.Errorf("failed to insert deposit transaction: %w", err)
	}

	// Assign all receipts in one UPDATE
	assignQuery, assignArgs, err := sqlx.In("UPDATE receipts SET transaction_id = ? WHERE id IN (?)", transaction.ID, receiptIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to build receipt assignment: %w", err)
	}
	if _, err := tx.Exec(assignQuery, assignArgs...); err != nil {
		return nil, fmt.Errorf("failed to assign receipts: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
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
		SetMap(map[string]any{
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
			SetMap(map[string]any{
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

// HighestCheckNumber returns the highest numeric check number used for an account.
// For earmark accounts, it looks at the root account and all its sub-accounts.
// Returns 0 if no checks have been written.
func (db *DB) HighestCheckNumber(accountID ID) (int, error) {
	// Find the root account (traverse up if this is an earmark)
	rootID := accountID
	for {
		var parentID *ID
		err := db.conn.Get(&parentID, "SELECT parent_id FROM bank_accounts WHERE id = ?", rootID)
		if err != nil {
			return 0, fmt.Errorf("failed to get account: %w", err)
		}
		if parentID == nil {
			break // This is the root
		}
		rootID = *parentID
	}

	// Get all check numbers from root account and its children
	query := `
		WITH RECURSIVE account_tree AS (
			SELECT id FROM bank_accounts WHERE id = ?
			UNION ALL
			SELECT ba.id FROM bank_accounts ba
			INNER JOIN account_tree at ON ba.parent_id = at.id
		)
		SELECT check_number FROM transactions
		WHERE account_id IN (SELECT id FROM account_tree)
		AND check_number IS NOT NULL
	`

	var checkNumbers []string
	if err := db.conn.Select(&checkNumbers, query, rootID); err != nil {
		return 0, fmt.Errorf("failed to get check numbers: %w", err)
	}

	// Find the highest numeric check number
	highest := 0
	for _, cn := range checkNumbers {
		if num, err := strconv.Atoi(cn); err == nil && num > highest {
			highest = num
		}
	}

	return highest, nil
}

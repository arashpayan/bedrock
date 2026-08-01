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
	// In-kind contributions (is_in_kind = 1) are non-cash and can never be part
	// of a deposit, so they are excluded here; passing one yields a count
	// mismatch below and the whole deposit is rejected.
	existsQuery, existsArgs, err := sqlx.In("SELECT id FROM receipts WHERE id IN (?) AND transaction_id IS NULL AND is_in_kind = 0", receiptIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to build receipt lookup: %w", err)
	}
	var availableIDs []ID
	if err := tx.Select(&availableIDs, existsQuery, existsArgs...); err != nil {
		return nil, fmt.Errorf("failed to check receipts: %w", err)
	}
	if len(availableIDs) != len(receiptIDs) {
		return nil, fmt.Errorf("expected %d depositable (cash, undeposited) receipts, found %d", len(receiptIDs), len(availableIDs))
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

// DeleteWithdrawal deletes a withdrawal transaction together with all of its
// expense line items in a single database transaction.
//
// It refuses to delete a check that has been reconciled (reconciliation_id IS
// NOT NULL), since removing it would desync that reconciliation's cleared
// balance; the transaction must be uncleared first. It also refuses anything
// that is not a withdrawal (non-negative amount), such as a deposit.
func (db *DB) DeleteWithdrawal(id ID) error {
	transaction, err := db.Transaction(id)
	if err != nil {
		return err
	}
	if transaction.Amount >= 0 {
		return fmt.Errorf("transaction %d is not a withdrawal", id)
	}
	if transaction.ReconciliationID != nil {
		return fmt.Errorf("cannot delete a reconciled check; unclear it from its reconciliation first")
	}

	tx, err := db.conn.Beginx()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	expensesQuery, expensesArgs := db.sq.Delete("expenses").
		Where("transaction_id = ?", id).
		MustSql()
	if _, err := tx.Exec(expensesQuery, expensesArgs...); err != nil {
		return fmt.Errorf("failed to delete expenses: %w", err)
	}

	txQuery, txArgs := db.sq.Delete("transactions").
		Where("id = ?", id).
		MustSql()
	if _, err := tx.Exec(txQuery, txArgs...); err != nil {
		return fmt.Errorf("failed to delete withdrawal: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ExpensesForTransaction returns the expense line items of a withdrawal
// transaction, in insertion order.
func (db *DB) ExpensesForTransaction(transactionID ID) ([]Expense, error) {
	query, args := db.sq.Select("id", "transaction_id", "category_id", "description", "amount", "currency", "created_at", "modified_at").
		From("expenses").
		Where("transaction_id = ?", transactionID).
		OrderBy("id ASC").
		MustSql()

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query expenses: %w", err)
	}
	defer rows.Close()

	var expenses []Expense
	for rows.Next() {
		var e Expense
		var amount int64
		var currency Currency
		if err := rows.Scan(
			&e.ID,
			&e.TransactionID,
			&e.CategoryID,
			&e.Description,
			&amount,
			&currency,
			&e.CreatedAt,
			&e.ModifiedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan expense: %w", err)
		}
		e.Amount = Money{Amount: amount, Currency: currency}
		expenses = append(expenses, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating expenses: %w", err)
	}
	return expenses, nil
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

// Transaction retrieves a single transaction by ID.
func (db *DB) Transaction(id ID) (*Transaction, error) {
	query, args := db.sq.Select("*").
		From("transactions").
		Where("id = ?", id).
		MustSql()

	var transaction Transaction
	if err := db.conn.Get(&transaction, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &transaction, nil
}

// TransactionsByPayee retrieves all withdrawal transactions where the given
// party is the payee, newest first. Deposits are not included because the
// payee_id column is only set on withdrawals.
func (db *DB) TransactionsByPayee(payeeID ID) ([]Transaction, error) {
	query, args := db.sq.Select("*").
		From("transactions").
		Where("payee_id = ?", payeeID).
		OrderBy("transacted_at DESC", "id DESC").
		MustSql()

	var transactions []Transaction
	if err := db.conn.Select(&transactions, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get transactions by payee: %w", err)
	}

	return transactions, nil
}

// UpdateWithdrawal edits an existing withdrawal transaction (a check) in place:
// it updates the header fields (payee, method, memo, date, check number) and
// replaces the entire set of expense line items, recomputing the transaction
// amount from the new expenses. Callers pass the complete desired expense set
// rather than a diff. The bank account is not changed.
//
// It refuses to update a check that has been reconciled (reconciliation_id IS
// NOT NULL), since changing its amount would desync that reconciliation's
// cleared balance; the transaction must be uncleared first. It also refuses
// anything that is not a withdrawal (non-negative amount).
func (db *DB) UpdateWithdrawal(id ID, payeeID ID, method TransactionMethod, memo string, transactedAt time.Time, checkNumber *string, expenses []ExpenseItem) (*Transaction, error) {
	if len(expenses) == 0 {
		return nil, fmt.Errorf("at least one expense item is required for withdrawal")
	}

	// Calculate total amount from expenses and validate currencies.
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

	existing, err := db.Transaction(id)
	if err != nil {
		return nil, err
	}
	if existing.Amount >= 0 {
		return nil, fmt.Errorf("transaction %d is not a withdrawal", id)
	}
	if existing.ReconciliationID != nil {
		return nil, fmt.Errorf("cannot edit a reconciled check; unclear it from its reconciliation first")
	}

	// Validate the (unchanged) account currency matches the expense currency.
	var accountCurrency Currency
	if err := db.conn.Get(&accountCurrency, "SELECT currency FROM bank_accounts WHERE id = ?", existing.AccountID); err != nil {
		return nil, fmt.Errorf("failed to get account currency: %w", err)
	}
	if accountCurrency != currency {
		return nil, fmt.Errorf("expense currency %s does not match account currency %s", currency, accountCurrency)
	}

	// Verify the payee exists before mutating anything.
	if _, err := db.Party(payeeID); err != nil {
		return nil, fmt.Errorf("payee not found: %w", err)
	}

	tx, err := db.conn.Beginx()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	headerQuery, headerArgs := db.sq.Update("transactions").
		SetMap(map[string]any{
			"amount":        -totalAmount,
			"payee_id":      payeeID,
			"memo":          memo,
			"method":        method,
			"transacted_at": transactedAt.Round(0),
			"check_number":  checkNumber,
		}).
		Where("id = ?", id).
		Suffix("RETURNING *").
		MustSql()

	updated := Transaction{}
	if err := tx.Get(&updated, headerQuery, headerArgs...); err != nil {
		return nil, fmt.Errorf("failed to update withdrawal transaction: %w", err)
	}

	deleteQuery, deleteArgs := db.sq.Delete("expenses").
		Where("transaction_id = ?", id).
		MustSql()
	if _, err := tx.Exec(deleteQuery, deleteArgs...); err != nil {
		return nil, fmt.Errorf("failed to clear existing expenses: %w", err)
	}

	for _, expense := range expenses {
		expenseQuery, expenseArgs := db.sq.Insert("expenses").
			SetMap(map[string]any{
				"transaction_id": id,
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

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &updated, nil
}

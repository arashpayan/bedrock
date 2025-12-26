package bedrock

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// CancelReconciliation cancels an in-progress reconciliation session
func (db *DB) CancelReconciliation(reconciliationID ID) (*Reconciliation, error) {
	// Get the reconciliation
	reconciliation, err := db.Reconciliation(reconciliationID)
	if err != nil {
		return nil, err
	}

	// Only in-progress reconciliations can be cancelled
	if reconciliation.Status != ReconciliationStatusInProgress {
		return nil, fmt.Errorf("can only cancel in-progress reconciliations, current status is %s", reconciliation.Status)
	}

	// Start database transaction
	tx, err := db.conn.Beginx()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Unclear all transactions associated with this reconciliation
	unclearQuery := `UPDATE transactions SET reconciliation_id = NULL WHERE reconciliation_id = ?`
	if _, err := tx.Exec(unclearQuery, reconciliationID); err != nil {
		return nil, fmt.Errorf("failed to unclear transactions: %w", err)
	}

	// Update reconciliation status
	updateQuery, updateArgs := db.sq.Update("reconciliations").
		SetMap(map[string]any{
			"status": ReconciliationStatusCancelled,
		}).
		Where("id = ?", reconciliationID).
		Suffix("RETURNING id, account_id, statement_date, statement_balance, statement_balance_currency, status, completed_at, created_at, modified_at").
		MustSql()

	var row reconciliationRow
	if err := tx.Get(&row, updateQuery, updateArgs...); err != nil {
		return nil, fmt.Errorf("failed to update reconciliation status: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	result := row.toReconciliation()
	return &result, nil
}

// ClearTransaction marks a transaction as cleared in a reconciliation
func (db *DB) ClearTransaction(reconciliationID ID, transactionID ID) error {
	// Get the reconciliation
	reconciliation, err := db.Reconciliation(reconciliationID)
	if err != nil {
		return err
	}

	// Only in-progress reconciliations can have transactions cleared
	if reconciliation.Status != ReconciliationStatusInProgress {
		return fmt.Errorf("can only clear transactions for in-progress reconciliations, current status is %s", reconciliation.Status)
	}

	// Get the transaction
	var transaction Transaction
	query := `SELECT * FROM transactions WHERE id = ?`
	if err := db.conn.Get(&transaction, query, transactionID); err != nil {
		return fmt.Errorf("failed to get transaction: %w", err)
	}

	// Verify transaction belongs to the same account (including subaccounts)
	accountIDs, err := db.getAccountAndDescendantIDs(reconciliation.AccountID)
	if err != nil {
		return fmt.Errorf("failed to get account hierarchy: %w", err)
	}

	validAccount := false
	for _, id := range accountIDs {
		if transaction.AccountID == id {
			validAccount = true
			break
		}
	}
	if !validAccount {
		return fmt.Errorf("transaction does not belong to the reconciliation account or its subaccounts")
	}

	// Check if transaction is already cleared
	if transaction.ReconciliationID != nil {
		if *transaction.ReconciliationID == reconciliationID {
			return nil // Already cleared for this reconciliation
		}
		return fmt.Errorf("transaction is already cleared in a different reconciliation")
	}

	// Verify transaction date is on or before statement date
	if transaction.TransactedAt.After(reconciliation.StatementDate) {
		return fmt.Errorf("cannot clear transaction dated after the statement date")
	}

	// Clear the transaction
	updateQuery, updateArgs := db.sq.Update("transactions").
		SetMap(map[string]any{
			"reconciliation_id": reconciliationID,
		}).
		Where("id = ?", transactionID).
		MustSql()

	if _, err := db.conn.Exec(updateQuery, updateArgs...); err != nil {
		return fmt.Errorf("failed to clear transaction: %w", err)
	}

	return nil
}

// ClearedBalance calculates the sum of all cleared transactions for a reconciliation
func (db *DB) ClearedBalance(reconciliationID ID) (Money, error) {
	reconciliation, err := db.Reconciliation(reconciliationID)
	if err != nil {
		return Money{}, err
	}

	var totalAmount int64
	query := `SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE reconciliation_id = ?`
	if err := db.conn.Get(&totalAmount, query, reconciliationID); err != nil {
		return Money{}, fmt.Errorf("failed to calculate cleared balance: %w", err)
	}

	// Get account currency
	accountCurrency := db.getAccountCurrency(reconciliation.AccountID)

	return Money{
		Amount:   totalAmount,
		Currency: accountCurrency,
	}, nil
}

// ClearedTransactions returns all transactions cleared in a reconciliation
func (db *DB) ClearedTransactions(reconciliationID ID) ([]Transaction, error) {
	var transactions []Transaction

	query := `SELECT * FROM transactions WHERE reconciliation_id = ? ORDER BY transacted_at ASC, id ASC`
	if err := db.conn.Select(&transactions, query, reconciliationID); err != nil {
		return nil, fmt.Errorf("failed to get cleared transactions: %w", err)
	}

	return transactions, nil
}

// CompleteReconciliation finalizes a reconciliation if the balance matches
func (db *DB) CompleteReconciliation(reconciliationID ID) (*Reconciliation, error) {
	// Get the reconciliation
	reconciliation, err := db.Reconciliation(reconciliationID)
	if err != nil {
		return nil, err
	}

	// Only in-progress reconciliations can be completed
	if reconciliation.Status != ReconciliationStatusInProgress {
		return nil, fmt.Errorf("can only complete in-progress reconciliations, current status is %s", reconciliation.Status)
	}

	// Calculate cleared balance
	clearedBalance, err := db.ClearedBalance(reconciliationID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate cleared balance: %w", err)
	}

	// Verify balance matches statement balance
	if clearedBalance.Amount != reconciliation.StatementBalance.Amount {
		difference := reconciliation.StatementBalance.Amount - clearedBalance.Amount
		return nil, fmt.Errorf("cleared balance (%s) does not match statement balance (%s), difference: %d cents",
			clearedBalance.String(), reconciliation.StatementBalance.String(), difference)
	}

	// Update reconciliation status
	now := time.Now()
	updateQuery, updateArgs := db.sq.Update("reconciliations").
		SetMap(map[string]any{
			"status":       ReconciliationStatusCompleted,
			"completed_at": now,
		}).
		Where("id = ?", reconciliationID).
		Suffix("RETURNING id, account_id, statement_date, statement_balance, statement_balance_currency, status, completed_at, created_at, modified_at").
		MustSql()

	var row reconciliationRow
	if err := db.conn.Get(&row, updateQuery, updateArgs...); err != nil {
		return nil, fmt.Errorf("failed to update reconciliation status: %w", err)
	}

	result := row.toReconciliation()
	return &result, nil
}

// InProgressReconciliation returns the in-progress reconciliation for an account, if any
func (db *DB) InProgressReconciliation(accountID ID) (*Reconciliation, error) {
	query := `SELECT id, account_id, statement_date, statement_balance, statement_balance_currency, status, completed_at, created_at, modified_at
			  FROM reconciliations WHERE account_id = ? AND status = ? LIMIT 1`

	var row reconciliationRow
	err := db.conn.Get(&row, query, accountID, ReconciliationStatusInProgress)
	if err != nil {
		return nil, fmt.Errorf("failed to get in-progress reconciliation: %w", err)
	}

	result := row.toReconciliation()
	return &result, nil
}

// LastCompletedReconciliation returns the most recent completed reconciliation for an account
func (db *DB) LastCompletedReconciliation(accountID ID) (*Reconciliation, error) {
	query := `SELECT id, account_id, statement_date, statement_balance, statement_balance_currency, status, completed_at, created_at, modified_at
			  FROM reconciliations WHERE account_id = ? AND status = ?
			  ORDER BY statement_date DESC LIMIT 1`

	var row reconciliationRow
	err := db.conn.Get(&row, query, accountID, ReconciliationStatusCompleted)
	if err != nil {
		return nil, fmt.Errorf("failed to get last completed reconciliation: %w", err)
	}

	result := row.toReconciliation()
	return &result, nil
}

// Reconciliation retrieves a reconciliation by ID
func (db *DB) Reconciliation(id ID) (*Reconciliation, error) {
	query := `SELECT id, account_id, statement_date, statement_balance, statement_balance_currency, status, completed_at, created_at, modified_at
			  FROM reconciliations WHERE id = ?`

	var row reconciliationRow
	err := db.conn.Get(&row, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get reconciliation: %w", err)
	}

	result := row.toReconciliation()
	return &result, nil
}

// Reconciliations returns all reconciliations for an account
func (db *DB) Reconciliations(accountID ID) ([]Reconciliation, error) {
	query := `SELECT id, account_id, statement_date, statement_balance, statement_balance_currency, status, completed_at, created_at, modified_at
			  FROM reconciliations WHERE account_id = ?
			  ORDER BY statement_date DESC`

	var rows []reconciliationRow
	if err := db.conn.Select(&rows, query, accountID); err != nil {
		return nil, fmt.Errorf("failed to get reconciliations: %w", err)
	}

	reconciliations := make([]Reconciliation, len(rows))
	for i, row := range rows {
		reconciliations[i] = row.toReconciliation()
	}

	return reconciliations, nil
}

// StartReconciliation begins a new reconciliation session for an account
func (db *DB) StartReconciliation(accountID ID, statementDate time.Time, statementBalance Money) (*Reconciliation, error) {
	// Validate account exists and is a root account
	account, err := db.BankAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if account.ParentID != nil {
		return nil, fmt.Errorf("reconciliation is only allowed for root-level accounts")
	}

	// Validate currency matches
	if account.Currency != statementBalance.Currency {
		return nil, fmt.Errorf("statement balance currency %s does not match account currency %s",
			statementBalance.Currency, account.Currency)
	}

	// Check for existing in-progress reconciliation
	existing, err := db.InProgressReconciliation(accountID)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("account already has an in-progress reconciliation (ID: %d)", existing.ID)
	}

	// Insert the reconciliation
	query, args := db.sq.Insert("reconciliations").
		SetMap(map[string]any{
			"account_id":                 accountID,
			"statement_date":             statementDate,
			"statement_balance":          statementBalance.Amount,
			"statement_balance_currency": statementBalance.Currency,
			"status":                     ReconciliationStatusInProgress,
		}).
		Suffix("RETURNING id, account_id, statement_date, statement_balance, statement_balance_currency, status, completed_at, created_at, modified_at").
		MustSql()

	var row reconciliationRow
	if err := db.conn.Get(&row, query, args...); err != nil {
		return nil, fmt.Errorf("failed to insert reconciliation: %w", err)
	}

	result := row.toReconciliation()
	return &result, nil
}

// UnclearTransaction removes the cleared status from a transaction
func (db *DB) UnclearTransaction(transactionID ID) error {
	// Get the transaction
	var transaction Transaction
	query := `SELECT * FROM transactions WHERE id = ?`
	if err := db.conn.Get(&transaction, query, transactionID); err != nil {
		return fmt.Errorf("failed to get transaction: %w", err)
	}

	// Check if transaction is cleared
	if transaction.ReconciliationID == nil {
		return nil // Already not cleared
	}

	// Get the reconciliation
	reconciliation, err := db.Reconciliation(*transaction.ReconciliationID)
	if err != nil {
		return fmt.Errorf("failed to get reconciliation: %w", err)
	}

	// Only transactions in in-progress reconciliations can be uncleared
	if reconciliation.Status != ReconciliationStatusInProgress {
		return fmt.Errorf("cannot unclear transaction from a %s reconciliation", reconciliation.Status)
	}

	// Unclear the transaction
	updateQuery, updateArgs := db.sq.Update("transactions").
		SetMap(map[string]any{
			"reconciliation_id": nil,
		}).
		Where("id = ?", transactionID).
		MustSql()

	if _, err := db.conn.Exec(updateQuery, updateArgs...); err != nil {
		return fmt.Errorf("failed to unclear transaction: %w", err)
	}

	return nil
}

// UnclearedTransactions returns all uncleared transactions for an account up to the statement date
func (db *DB) UnclearedTransactions(accountID ID, statementDate time.Time) ([]Transaction, error) {
	// Get account and all descendant account IDs
	accountIDs, err := db.getAccountAndDescendantIDs(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account hierarchy: %w", err)
	}

	// Build query with IN clause for account IDs
	query := `SELECT * FROM transactions
			  WHERE account_id IN (?)
			  AND reconciliation_id IS NULL
			  AND transacted_at <= ?
			  ORDER BY transacted_at ASC, id ASC`

	// Use sqlx.In to expand the slice
	query, args, err := sqlxIn(query, accountIDs, statementDate)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var transactions []Transaction
	if err := db.conn.Select(&transactions, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get uncleared transactions: %w", err)
	}

	return transactions, nil
}

// UndoReconciliation reverts the most recent completed reconciliation for an account
func (db *DB) UndoReconciliation(reconciliationID ID) error {
	// Get the reconciliation
	reconciliation, err := db.Reconciliation(reconciliationID)
	if err != nil {
		return err
	}

	// Only completed reconciliations can be undone
	if reconciliation.Status != ReconciliationStatusCompleted {
		return fmt.Errorf("can only undo completed reconciliations, current status is %s", reconciliation.Status)
	}

	// Verify this is the most recent completed reconciliation
	lastCompleted, err := db.LastCompletedReconciliation(reconciliation.AccountID)
	if err != nil {
		return fmt.Errorf("failed to get last completed reconciliation: %w", err)
	}

	if lastCompleted.ID != reconciliationID {
		return fmt.Errorf("can only undo the most recent completed reconciliation")
	}

	// Start database transaction
	tx, err := db.conn.Beginx()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Unclear all transactions associated with this reconciliation
	unclearQuery := `UPDATE transactions SET reconciliation_id = NULL WHERE reconciliation_id = ?`
	if _, err := tx.Exec(unclearQuery, reconciliationID); err != nil {
		return fmt.Errorf("failed to unclear transactions: %w", err)
	}

	// Update reconciliation status to cancelled (we keep the record for history)
	updateQuery, updateArgs := db.sq.Update("reconciliations").
		SetMap(map[string]any{
			"status":       ReconciliationStatusCancelled,
			"completed_at": nil,
		}).
		Where("id = ?", reconciliationID).
		MustSql()

	if _, err := tx.Exec(updateQuery, updateArgs...); err != nil {
		return fmt.Errorf("failed to update reconciliation status: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// getAccountAndDescendantIDs returns the account ID and all descendant account IDs
func (db *DB) getAccountAndDescendantIDs(accountID ID) ([]ID, error) {
	query := `
		WITH RECURSIVE account_tree AS (
			SELECT id FROM bank_accounts WHERE id = ?
			UNION ALL
			SELECT ba.id FROM bank_accounts ba
			INNER JOIN account_tree at ON ba.parent_id = at.id
		)
		SELECT id FROM account_tree`

	var ids []ID
	if err := db.conn.Select(&ids, query, accountID); err != nil {
		return nil, fmt.Errorf("failed to get account tree: %w", err)
	}

	return ids, nil
}

// sqlxIn is a helper to expand IN clause placeholders
func sqlxIn(query string, args ...interface{}) (string, []interface{}, error) {
	// Use sqlx.In for proper expansion
	return sqlx.In(query, args...)
}

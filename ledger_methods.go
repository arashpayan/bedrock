package bedrock

import (
	"fmt"
	"time"
)

// AccountLedger returns the complete ledger for an account with running balances
func (db *DB) AccountLedger(accountID ID, options *LedgerOptions) ([]LedgerEntry, error) {
	if options == nil {
		options = &LedgerOptions{}
	}

	// Get all transactions for the account (and subaccounts if requested)
	transactions, err := db.TransactionsForAccount(accountID, options.IncludeSubaccounts, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	// Calculate running balances
	ledgerEntries := make([]LedgerEntry, len(transactions))
	var runningBalance int64 = 0

	for i, transaction := range transactions {
		// Add transaction amount to running balance
		runningBalance += transaction.Amount

		// Create ledger entry
		entry := LedgerEntry{
			Transaction: transaction,
			RunningBalance: Money{
				Amount:   runningBalance,
				Currency: db.getAccountCurrency(transaction.AccountID),
			},
		}

		// Enrich with additional data
		if err := db.enrichLedgerEntry(&entry); err != nil {
			return nil, fmt.Errorf("failed to enrich ledger entry: %w", err)
		}

		ledgerEntries[i] = entry
	}

	return ledgerEntries, nil
}

// AccountBalance calculates the current balance of an account
func (db *DB) AccountBalance(accountID ID, includeSubaccounts bool) (Money, error) {
	accountCurrency := db.getAccountCurrency(accountID)

	var totalAmount int64
	var err error

	if includeSubaccounts {
		// Get balance including all subaccounts
		query := `
			WITH RECURSIVE account_tree AS (
				SELECT id FROM bank_accounts WHERE id = ?
				UNION ALL
				SELECT ba.id FROM bank_accounts ba
				INNER JOIN account_tree at ON ba.parent_id = at.id
			)
			SELECT COALESCE(SUM(amount), 0) FROM transactions
			WHERE account_id IN (SELECT id FROM account_tree)`

		err = db.conn.Get(&totalAmount, query, accountID)
	} else {
		// Get balance for just this account
		query := `SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE account_id = ?`
		err = db.conn.Get(&totalAmount, query, accountID)
	}

	if err != nil {
		return Money{}, fmt.Errorf("failed to calculate account balance: %w", err)
	}

	return Money{
		Amount:   totalAmount,
		Currency: accountCurrency,
	}, nil
}

// AccountBalanceAsOf calculates the balance as of a specific date/time
func (db *DB) AccountBalanceAsOf(accountID ID, asOfDate time.Time, includeSubaccounts bool) (Money, error) {
	accountCurrency := db.getAccountCurrency(accountID)

	var totalAmount int64
	var err error

	if includeSubaccounts {
		// Get balance including all subaccounts up to the specified date
		query := `
			WITH RECURSIVE account_tree AS (
				SELECT id FROM bank_accounts WHERE id = ?
				UNION ALL
				SELECT ba.id FROM bank_accounts ba
				INNER JOIN account_tree at ON ba.parent_id = at.id
			)
			SELECT COALESCE(SUM(amount), 0) FROM transactions
			WHERE account_id IN (SELECT id FROM account_tree)
			AND transacted_at <= ?`

		err = db.conn.Get(&totalAmount, query, accountID, asOfDate)
	} else {
		// Get balance for just this account up to the specified date
		query := `SELECT COALESCE(SUM(amount), 0) FROM transactions
				  WHERE account_id = ? AND transacted_at <= ?`
		err = db.conn.Get(&totalAmount, query, accountID, asOfDate)
	}

	if err != nil {
		return Money{}, fmt.Errorf("failed to calculate account balance as of date: %w", err)
	}

	return Money{
		Amount:   totalAmount,
		Currency: accountCurrency,
	}, nil
}

// TransactionsForAccount returns transactions for an account with optional subaccounts
func (db *DB) TransactionsForAccount(accountID ID, includeSubaccounts bool, options *LedgerOptions) ([]Transaction, error) {
	if options == nil {
		options = &LedgerOptions{}
	}

	// Build the base query
	var query string
	var args []interface{}

	if includeSubaccounts {
		query = `
			WITH RECURSIVE account_tree AS (
				SELECT id FROM bank_accounts WHERE id = ?
				UNION ALL
				SELECT ba.id FROM bank_accounts ba
				INNER JOIN account_tree at ON ba.parent_id = at.id
			)
			SELECT t.* FROM transactions t
			WHERE t.account_id IN (SELECT id FROM account_tree)`
		args = append(args, accountID)
	} else {
		query = `SELECT * FROM transactions WHERE account_id = ?`
		args = append(args, accountID)
	}

	// Add date filters
	if options.StartDate != nil {
		query += ` AND transacted_at >= ?`
		args = append(args, *options.StartDate)
	}
	if options.EndDate != nil {
		query += ` AND transacted_at <= ?`
		args = append(args, *options.EndDate)
	}

	// Order by transaction date, then by ID for consistency
	query += ` ORDER BY transacted_at ASC, id ASC`

	// Add pagination
	if options.Limit != nil {
		query += ` LIMIT ?`
		args = append(args, *options.Limit)
	}
	if options.Offset != nil {
		query += ` OFFSET ?`
		args = append(args, *options.Offset)
	}

	var transactions []Transaction
	if err := db.conn.Select(&transactions, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get transactions for account: %w", err)
	}

	return transactions, nil
}

// AccountTransactionCount returns the total number of transactions for pagination
func (db *DB) AccountTransactionCount(accountID ID, includeSubaccounts bool, options *LedgerOptions) (int, error) {
	if options == nil {
		options = &LedgerOptions{}
	}

	var query string
	var args []interface{}

	if includeSubaccounts {
		query = `
			WITH RECURSIVE account_tree AS (
				SELECT id FROM bank_accounts WHERE id = ?
				UNION ALL
				SELECT ba.id FROM bank_accounts ba
				INNER JOIN account_tree at ON ba.parent_id = at.id
			)
			SELECT COUNT(*) FROM transactions t
			WHERE t.account_id IN (SELECT id FROM account_tree)`
		args = append(args, accountID)
	} else {
		query = `SELECT COUNT(*) FROM transactions WHERE account_id = ?`
		args = append(args, accountID)
	}

	// Add date filters
	if options.StartDate != nil {
		query += ` AND transacted_at >= ?`
		args = append(args, *options.StartDate)
	}
	if options.EndDate != nil {
		query += ` AND transacted_at <= ?`
		args = append(args, *options.EndDate)
	}

	var count int
	if err := db.conn.Get(&count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count transactions for account: %w", err)
	}

	return count, nil
}

// AllAccountBalances returns current balances for all active accounts
func (db *DB) AllAccountBalances() (map[ID]Money, error) {
	// Get all active accounts with their currencies
	accounts, err := db.ActiveBankAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to get active accounts: %w", err)
	}

	balances := make(map[ID]Money)

	// Calculate balance for each account (without subaccounts to avoid double-counting)
	for _, account := range accounts {
		var totalAmount int64
		query := `SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE account_id = ?`
		if err := db.conn.Get(&totalAmount, query, account.ID); err != nil {
			return nil, fmt.Errorf("failed to calculate balance for account %d: %w", account.ID, err)
		}

		balances[account.ID] = Money{
			Amount:   totalAmount,
			Currency: account.Currency,
		}
	}

	return balances, nil
}

// LastTransactionDate returns the most recent transaction date for an account
func (db *DB) LastTransactionDate(accountID ID, includeSubaccounts bool) (*time.Time, error) {
	var query string
	var args []interface{}

	if includeSubaccounts {
		query = `
			WITH RECURSIVE account_tree AS (
				SELECT id FROM bank_accounts WHERE id = ?
				UNION ALL
				SELECT ba.id FROM bank_accounts ba
				INNER JOIN account_tree at ON ba.parent_id = at.id
			)
			SELECT MAX(transacted_at) FROM transactions t
			WHERE t.account_id IN (SELECT id FROM account_tree)`
		args = append(args, accountID)
	} else {
		query = `SELECT MAX(transacted_at) FROM transactions WHERE account_id = ?`
		args = append(args, accountID)
	}

	var lastDateStr *string
	if err := db.conn.Get(&lastDateStr, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get last transaction date: %w", err)
	}

	if lastDateStr == nil {
		return nil, nil
	}

	// Parse the date string returned by SQLite
	lastDate, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", *lastDateStr)
	if err != nil {
		// Try alternative format
		lastDate, err = time.Parse("2006-01-02 15:04:05", *lastDateStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse last transaction date: %w", err)
		}
	}

	return &lastDate, nil
}

// Helper methods

// getAccountCurrency retrieves the currency for an account (cached or from DB)
func (db *DB) getAccountCurrency(accountID ID) Currency {
	// Simple implementation - in production, this could be cached
	var currency Currency
	query := `SELECT currency FROM bank_accounts WHERE id = ?`
	if err := db.conn.Get(&currency, query, accountID); err != nil {
		// Default to USD if we can't find the account
		return CurrencyUSD
	}
	return currency
}

// enrichLedgerEntry adds additional contextual data to a ledger entry
func (db *DB) enrichLedgerEntry(entry *LedgerEntry) error {
	transaction := &entry.Transaction

	// For deposits, get customer name and receipt count
	if transaction.Amount > 0 {
		// Get receipts for this transaction
		receipts, err := db.ReceiptsByTransaction(transaction.ID)
		if err != nil {
			return fmt.Errorf("failed to get receipts for transaction: %w", err)
		}

		entry.ReceiptCount = len(receipts)

		// If there's exactly one receipt, get the customer name
		if len(receipts) == 1 {
			party, err := db.Party(receipts[0].CustomerID)
			if err == nil {
				entry.CustomerName = &party.Name
			}
		}
	}

	// For withdrawals, get payee name and expense count
	if transaction.Amount < 0 && transaction.PayeeID != nil {
		// Get payee name
		party, err := db.Party(*transaction.PayeeID)
		if err == nil {
			entry.PayeeName = &party.Name
		}

		// Get expense count
		var expenseCount int
		query := `SELECT COUNT(*) FROM expenses WHERE transaction_id = ?`
		if err := db.conn.Get(&expenseCount, query, transaction.ID); err == nil {
			entry.ExpenseCount = expenseCount
		}
	}

	return nil
}

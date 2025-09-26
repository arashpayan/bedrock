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

	transaction := &Transaction{
		AccountID:    accountID,
		Amount:       amount.Amount,
		Memo:         memo,
		Method:       &method,
		TransactedAt: transactedAt,
	}

	query, args := db.sq.Insert("transactions").
		SetMap(map[string]interface{}{
			"account_id":    accountID,
			"amount":        amount.Amount,
			"memo":          memo,
			"method":        method,
			"transacted_at": transactedAt,
		}).
		Suffix("RETURNING id, created_at, modified_at").
		MustSql()

	err = db.conn.QueryRow(query, args...).Scan(&transaction.ID, &transaction.CreatedAt, &transaction.ModifiedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert deposit transaction: %w", err)
	}

	return transaction, nil
}

// CreateWithdrawal creates a new withdrawal transaction
func (db *DB) CreateWithdrawal(accountID ID, amount Money, payeeID ID, method TransactionMethod, memo string, transactedAt time.Time, checkNumber *string) (*Transaction, error) {
	if amount.Amount <= 0 {
		return nil, fmt.Errorf("withdrawal amount must be positive, got %d cents (it will be stored as negative)", amount.Amount)
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

	transaction := &Transaction{
		AccountID:    accountID,
		Amount:       -amount.Amount, // Store withdrawals as negative
		CheckNumber:  checkNumber,
		Memo:         memo,
		Method:       &method,
		PayeeID:      &payeeID,
		TransactedAt: transactedAt,
	}

	query, args := db.sq.Insert("transactions").
		SetMap(map[string]interface{}{
			"account_id":    accountID,
			"amount":        -amount.Amount,
			"payee_id":      payeeID,
			"memo":          memo,
			"method":        method,
			"transacted_at": transactedAt,
			"check_number":  checkNumber,
		}).
		Suffix("RETURNING id, created_at, modified_at").
		MustSql()

	err = db.conn.QueryRow(query, args...).Scan(&transaction.ID, &transaction.CreatedAt, &transaction.ModifiedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert withdrawal transaction: %w", err)
	}

	return transaction, nil
}

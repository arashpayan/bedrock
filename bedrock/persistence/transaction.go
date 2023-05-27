package persistence

import (
	"context"
	"errors"

	"ara.sh/iabdaccounting/bedrock/datetime"
	"ara.sh/iabdaccounting/bedrock/model"
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

func (db *Database) createDepositTx(tx *sqlx.Tx, now datetime.DateTime, in model.CreateDepositTransactionInput) (*model.Transaction, error) {
	// make sure it's a well-formed transaction
	if !in.Amount.IsPositive() {
		return nil, errors.New("deposits require a positive amount")
	}

	query, args := sq.Insert(tableTransactions).SetMap(map[string]any{
		"created_at":  now,
		"modified_at": now,

		"account_id":    in.AccountID,
		"amount":        in.Amount,
		"deposit_id":    in.DepositID,
		"memo":          in.Memo,
		"method":        in.Method,
		"transacted_at": in.TransactedAt,
	}).Suffix("RETURNING *").MustSql()
	brTx := model.Transaction{}
	if err := tx.Get(&brTx, query, args...); err != nil {
		return nil, err
	}

	return &brTx, nil
}

func (db *Database) CreateWithdrawal(ctx context.Context, in model.CreateWithdrawalTransactionInput) (*model.Transaction, error) {
	if !in.Amount.IsNegative() {
		return nil, errors.New("withdrawals require a negative amount")
	}

	tx := db.dbx.MustBeginTx(ctx, nil)
	defer tx.Rollback()

	now := datetime.Now()
	query, args := sq.Insert(tableTransactions).SetMap(map[string]any{
		"created_at":  now,
		"modified_at": now,

		"account_id":    in.AccountID,
		"amount":        in.Amount,
		"check_number":  in.CheckNumber,
		"memo":          in.Memo,
		"method":        in.Method,
		"payee_id":      in.PayeeID,
		"transacted_at": in.TransactedAt,
	}).Suffix("RETURNING *").MustSql()
	brTx := model.Transaction{}
	if err := tx.Get(&brTx, query, args...); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &brTx, nil
}

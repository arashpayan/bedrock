package persistence

import (
	"context"
	"errors"
	"fmt"

	"ara.sh/iabdaccounting/bedrock/datetime"
	"ara.sh/iabdaccounting/bedrock/model"
	sq "github.com/Masterminds/squirrel"
)

func (db *Database) CreateDeposit(ctx context.Context, input model.CreateDepositInput) (*model.Deposit, error) {
	tx := db.dbx.MustBeginTx(ctx, nil)
	defer tx.Rollback()

	if len(input.UndepositedReceiptIDs) == 0 {
		return nil, errors.New("no receipts specified")
	}

	now := datetime.Now()
	query, args := sq.Insert(tableDeposits).SetMap(map[string]any{
		"created_at":  now,
		"modified_at": now,

		"account_id": input.AccountID,
	}).Suffix("RETURNING *").MustSql()
	deposit := model.Deposit{}
	if err := tx.Get(&deposit, query, args...); err != nil {
		return nil, err
	}

	var depositTotal model.Money
	for _, rcptID := range input.UndepositedReceiptIDs {
		_, err := sq.Insert(tableDepositReceipts).SetMap(map[string]any{
			"receipt_id": rcptID,
			"deposit_id": deposit.ID,
		}).RunWith(tx.Tx).Exec()
		if err != nil {
			return nil, fmt.Errorf("inserting receipt: %v", err)
		}

		// obtain the price for each receipt
		var price model.Money
		query, args := sq.Select("total").From(tableReceipts).Where(sq.Eq{"id": rcptID}).MustSql()
		if err := tx.Get(&price, query, args...); err != nil {
			return nil, fmt.Errorf("checking for receipt total: %v", err)
		}
		depositTotal += price
	}

	// create a corresponding transaction for the account
	_, err := db.createDepositTx(tx, now, model.CreateDepositTransactionInput{
		AccountID:    input.AccountID,
		Amount:       depositTotal,
		DepositID:    deposit.ID,
		Memo:         input.Memo,
		TransactedAt: input.DepositedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("creating transaction: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &deposit, nil
}

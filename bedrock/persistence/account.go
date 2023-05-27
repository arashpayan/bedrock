package persistence

import (
	"context"

	"ara.sh/iabdaccounting/bedrock/datetime"
	"ara.sh/iabdaccounting/bedrock/model"
	sq "github.com/Masterminds/squirrel"
)

func (db *Database) CreateAccount(ctx context.Context, in model.CreateAccountInput) (*model.Account, error) {
	tx := db.dbx.MustBeginTx(ctx, nil)
	defer tx.Rollback()

	now := datetime.Now()
	query, args := sq.Insert(tableAccounts).SetMap(map[string]any{
		"created_at":  now,
		"modified_at": now,

		"type":             in.Type,
		"name":             in.Name,
		"description":      in.Description,
		"denomination":     in.Denomination,
		"starting_balance": in.StartingBalance,
		"starting_date":    in.StartingDate,
		"parent_id":        in.ParentID,
	}).Suffix("RETURNING *").MustSql()
	acct := model.Account{}
	if err := tx.Get(&acct, query, args...); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &acct, nil
}

package persistence

import (
	"context"

	"ara.sh/iabdaccounting/bedrock/datetime"
	"ara.sh/iabdaccounting/bedrock/model"
	sq "github.com/Masterminds/squirrel"
)

func (db *Database) CreateItem(ctx context.Context, item model.CreateItemInput) (*model.Item, error) {
	tx := db.dbx.MustBeginTx(ctx, nil)
	defer tx.Rollback()

	now := datetime.Now()
	query, args := sq.Insert(tableItems).SetMap(map[string]any{
		"created_at":  now,
		"modified_at": now,

		"name":     item.Name,
		"shortcut": item.Shortcut,
	}).Suffix("RETURNING *").MustSql()

	var inserted model.Item
	if err := tx.Get(&inserted, query, args...); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &inserted, nil
}

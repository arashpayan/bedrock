package persistence

import (
	"context"

	"ara.sh/iabdaccounting/bedrock/datetime"
	"ara.sh/iabdaccounting/bedrock/model"
	sq "github.com/Masterminds/squirrel"
)

func (db *Database) CreateParty(ctx context.Context, p model.CreatePartyInput) (*model.Party, error) {
	tx := db.dbx.MustBegin()
	defer tx.Rollback()

	now := datetime.Now()
	query, args := sq.Insert(tableParties).SetMap(map[string]any{
		"created_at":  now,
		"modified_at": now,

		"name":             p.Name,
		"email_address":    p.EmailAddress,
		"bahai_id_number":  p.BahaiIDNumber,
		"address":          p.Address,
		"telephone_number": p.TelephoneNumber,
	}).Suffix("RETURNING *").MustSql()

	var inserted model.Party
	if err := tx.Get(&inserted, query, args...); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &inserted, nil
}

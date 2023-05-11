package persistence

import (
	"ara.sh/iabdaccounting/bedrock/datetime"
	"ara.sh/iabdaccounting/bedrock/model"
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

func (db *Database) account(tx *sqlx.Tx, id int64) (*model.Account, error) {
	query, args := sq.Select("*").From(tableAccounts).Where(sq.Eq{"id": id}).MustSql()
	var acct model.Account
	if err := tx.Get(&acct, query, args...); err != nil {
		return nil, err
	}
	return &acct, nil
}

func (db *Database) InsertAccount(in model.CreateAccountInput) (*model.Account, error) {
	tx, err := db.dbx.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := datetime.Now()
	result, err := sq.Insert(tableAccounts).SetMap(map[string]any{
		"created_at": now,
	}).RunWith(tx.Tx).Exec()
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return db.account(tx, id)
}

package persistence

import (
	"context"
	"fmt"
	"time"

	"ara.sh/iabdaccounting/bedrock/model"
	sq "github.com/Masterminds/squirrel"
)

type assemblyKey string

const (
	assemblyKeyName     assemblyKey = "name"
	assemblyKeyTimezone assemblyKey = "timezone"
)

func (db *Database) AssemblyInfo(ctx context.Context) (*model.Assembly, error) {
	tx := db.dbx.MustBeginTx(ctx, nil)
	defer tx.Rollback()

	kvs := []struct {
		Key   assemblyKey `db:"key_name"`
		Value string      `db:"value"`
	}{}
	query, args := sq.Select("*").From(tableAssemblyInfo).MustSql()
	if err := tx.Select(&kvs, query, args...); err != nil {
		return nil, err
	}

	a := model.Assembly{}
	for _, kv := range kvs {
		switch kv.Key {
		case assemblyKeyName:
			a.Name = kv.Value
		case assemblyKeyTimezone:
			tz, err := time.LoadLocation(kv.Value)
			if err != nil {
				return nil, fmt.Errorf("parsing location: %v", err)
			}
			a.Timezone = *tz
		}
	}

	return &a, nil
}

func (db *Database) SetAssemblyInfo(ctx context.Context, asbly model.Assembly) error {
	tx := db.dbx.MustBeginTx(ctx, nil)
	defer tx.Rollback()

	insertKV := func(key assemblyKey, value string) error {
		_, err := sq.Insert(tableAssemblyInfo).SetMap(map[string]any{
			"key_name": key,
			"value":    value,
		}).RunWith(tx.Tx).Exec()
		return err
	}

	if err := insertKV(assemblyKeyName, asbly.Name); err != nil {
		return err
	}
	if err := insertKV(assemblyKeyTimezone, asbly.Timezone.String()); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	db.Assembly = asbly

	return nil
}

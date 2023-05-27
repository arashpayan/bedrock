package persistence

import (
	"context"
	_ "embed"
	"fmt"

	"ara.sh/iabdaccounting/bedrock/model"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

const InMemoryDSN = ":memory:"
const requiredVersion = 1

//go:embed 0001.sql
var migrationsSQL []byte

type Database struct {
	dbx      *sqlx.DB
	Assembly model.Assembly
}

func Open(path string) (*Database, error) {
	dsn := `file://` + path
	dbx, err := sqlx.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}
	dbx.SetMaxOpenConns(1)
	db := &Database{dbx: dbx}
	dbVer, err := db.schemaVersion()
	if err != nil {
		return nil, fmt.Errorf("obtaining schema version: %v", err)
	}
	if dbVer > requiredVersion {
		return nil, ErrAssemblyFileTooNew
	}
	if dbVer != requiredVersion {
		log.Info().Uint("fromVer", dbVer).Uint("toVer", requiredVersion).Msg("Migrating assembly file")
		if err := db.migrate(dbVer); err != nil {
			return nil, fmt.Errorf("assembly file migration failed: %v", err)
		}
	}

	asbly, err := db.AssemblyInfo(context.Background())
	if err != nil {
		return nil, fmt.Errorf("loading assembly info")
	}
	db.Assembly = *asbly

	return db, nil
}

func (db *Database) migrate(currVer uint) error {
	tx, err := db.dbx.Beginx()
	if err != nil {
		return fmt.Errorf("beginning transaction: %v", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(string(migrationsSQL)); err != nil {
		return fmt.Errorf("executing migration: %v", err)
	}

	if err := db.setSchemaVersion(tx, requiredVersion); err != nil {
		return fmt.Errorf("setting assembly file version: %v", err)
	}

	if _, err := tx.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return fmt.Errorf("enabling foreign keys: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %v", err)
	}

	return nil
}

func (db *Database) schemaVersion() (uint, error) {
	var v uint
	err := db.dbx.QueryRow(`PRAGMA user_version;`).Scan(&v)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func (db *Database) setSchemaVersion(tx *sqlx.Tx, ver uint) error {
	query := fmt.Sprintf(`PRAGMA user_version = %d`, ver)
	_, err := tx.Exec(query)
	return err
}

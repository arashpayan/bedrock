package bedrock

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/jmoiron/sqlx"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// currentSchemaVersion is the schema version this binary knows about. Bump by
// one whenever a new migrations/000N_*.sql file is added. The version is
// stored in each database file as PRAGMA user_version.
const currentSchemaVersion = 6

// migrate brings the database up to currentSchemaVersion. The work happens
// inside a single BEGIN IMMEDIATE transaction so:
//
//   - a concurrent open of the same file blocks here until this one commits,
//     instead of racing on which process applies the migrations;
//   - if any individual migration fails, or the user_version write fails,
//     the whole upgrade rolls back and the file is left at the prior version.
//
// If user_version is already at currentSchemaVersion the transaction commits
// without doing any work. If it's greater, migrate returns an error and the
// caller must refuse to open the file.
func (db *DB) migrate(ctx context.Context) error {
	conn, err := db.conn.Connx(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("begin immediate: %w", err)
	}

	if err := applyPendingMigrations(ctx, conn); err != nil {
		if _, rbErr := conn.ExecContext(ctx, "ROLLBACK"); rbErr != nil {
			return fmt.Errorf("%w (rollback also failed: %v)", err, rbErr)
		}
		return err
	}

	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("commit migration: %w", err)
	}
	return nil
}

func applyPendingMigrations(ctx context.Context, conn *sqlx.Conn) error {
	var v int
	if err := conn.GetContext(ctx, &v, "PRAGMA user_version"); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}

	if v > currentSchemaVersion {
		return fmt.Errorf("database is at schema v%d but this build only understands up to v%d; upgrade the application", v, currentSchemaVersion)
	}
	if v == currentSchemaVersion {
		return nil
	}

	for next := v + 1; next <= currentSchemaVersion; next++ {
		sql, err := loadMigration(next)
		if err != nil {
			return err
		}
		if _, err := conn.ExecContext(ctx, sql); err != nil {
			return fmt.Errorf("apply migration v%d: %w", next, err)
		}
	}

	// PRAGMA user_version doesn't accept bound parameters, but
	// currentSchemaVersion is a compile-time constant so there's no injection
	// surface.
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", currentSchemaVersion)); err != nil {
		return fmt.Errorf("set user_version: %w", err)
	}
	return nil
}

// loadMigration reads migrations/000N_*.sql for the given version. Exactly
// one file must match.
func loadMigration(version int) (string, error) {
	pattern := fmt.Sprintf("migrations/%04d_*.sql", version)
	matches, err := fs.Glob(migrationsFS, pattern)
	if err != nil {
		return "", fmt.Errorf("glob %s: %w", pattern, err)
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no migration file matches %s", pattern)
	case 1:
		b, err := migrationsFS.ReadFile(matches[0])
		if err != nil {
			return "", fmt.Errorf("read %s: %w", matches[0], err)
		}
		return string(b), nil
	default:
		return "", fmt.Errorf("multiple migration files match %s: %v", pattern, matches)
	}
}

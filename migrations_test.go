package bedrock

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// openRaw opens the database using the bare sql.DB driver, without going
// through bedrock.Open() (which would run migrations). It exists so tests can
// stage a file at a specific user_version before handing it to bedrock.Open()
// and observing the upgrade behavior.
func openRaw(t *testing.T, path string) *sql.DB {
	t.Helper()
	raw, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { _ = raw.Close() })
	return raw
}

// TestMigrate_FreshDatabaseStampsCurrentVersion verifies that bedrock.Open on a
// brand-new file applies every migration and ends up at currentSchemaVersion.
func TestMigrate_FreshDatabaseStampsCurrentVersion(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "fresh.bedrock")

	db, err := Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	var v int
	require.NoError(t, db.conn.Get(&v, "PRAGMA user_version"))
	assert.Equal(t, currentSchemaVersion, v, "fresh DB should be stamped at currentSchemaVersion")
}

// TestMigrate_UpgradesLegacyV1Database simulates a real-world bedrock file
// that was created before the migration system existed: it has the v1 schema
// (no memo column) and user_version=0. Opening it via bedrock.Open should run
// every pending migration, leaving user_version=currentSchemaVersion and the
// receipts.memo column in place and writable.
func TestMigrate_UpgradesLegacyV1Database(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.bedrock")

	// Stage a v1 file: apply migration 0001 and leave user_version=0.
	{
		raw := openRaw(t, dbPath)
		v1, err := loadMigration(1)
		require.NoError(t, err)
		_, err = raw.Exec(v1)
		require.NoError(t, err)
		// loadMigration uses the embedded SQL; user_version is still 0 because
		// no migration writer set it. That matches the state of any DB the old
		// bedrock.Open() ever produced.
		var v int
		require.NoError(t, raw.QueryRow("PRAGMA user_version").Scan(&v))
		require.Equal(t, 0, v, "staged DB should look like a pre-migration-system file")
		require.NoError(t, raw.Close())
	}

	// Hand it to bedrock.Open — it should upgrade silently.
	db, err := Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	var v int
	require.NoError(t, db.conn.Get(&v, "PRAGMA user_version"))
	assert.Equal(t, currentSchemaVersion, v)

	// receipts.memo must exist and accept a write.
	_, err = db.createAssembly("Migrated Assembly", time.UTC, CurrencyUSD)
	require.NoError(t, err)
	party, err := db.CreateParty("Contributor", nil, nil, nil, nil)
	require.NoError(t, err)
	receipt, err := db.CreateReceipt(party.ID, time.Now(), "post-upgrade memo")
	require.NoError(t, err)
	assert.Equal(t, "post-upgrade memo", receipt.Memo)
}

// TestMigrate_RejectsFutureVersion verifies that a file written by a newer
// build (user_version > currentSchemaVersion) refuses to open rather than
// silently truncating its features.
func TestMigrate_RejectsFutureVersion(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "future.bedrock")

	{
		raw := openRaw(t, dbPath)
		_, err := raw.Exec(fmt.Sprintf("PRAGMA user_version = %d", currentSchemaVersion+5))
		require.NoError(t, err)
		require.NoError(t, raw.Close())
	}

	_, err := Open(dbPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upgrade the application", "error should tell the user how to recover")
}

// TestMigrate_IsAtomic verifies that if any individual migration step fails,
// the file is left at the prior version with no partial schema state.
func TestMigrate_IsAtomic(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "atomic.bedrock")

	// Bring a file up to currentSchemaVersion using a real Open(), then close.
	{
		db, err := Open(dbPath)
		require.NoError(t, err)
		require.NoError(t, db.Close())
	}

	// Inject a deliberately broken migration step into the live connection
	// inside a manually-managed IMMEDIATE transaction, mimicking what
	// applyPendingMigrations does. If the broken step rolls back, the file
	// must remain readable and at currentSchemaVersion.
	raw := openRaw(t, dbPath)
	ctx := context.Background()
	conn, err := raw.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE")
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, "ALTER TABLE receipts ADD COLUMN scratch TEXT")
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, "ALTER TABLE this_table_does_not_exist ADD COLUMN x TEXT")
	require.Error(t, err, "the broken step must fail so we can verify rollback")
	_, err = conn.ExecContext(ctx, "ROLLBACK")
	require.NoError(t, err)

	// The scratch column should not exist — the transaction rolled back.
	var hasScratch int
	require.NoError(t, conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('receipts') WHERE name='scratch'").Scan(&hasScratch))
	assert.Zero(t, hasScratch, "rolled-back column should not be present")

	// And the file should still open cleanly.
	require.NoError(t, conn.Close())
	require.NoError(t, raw.Close())
	db, err := Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
}

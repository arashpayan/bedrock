package persistence

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"ara.sh/iabdaccounting/bedrock/model"
	"github.com/stretchr/testify/require"
)

func assertValidBase(t *testing.T, b model.Base) {
	now := time.Now()
	require.Greater(t, int64(b.ID), int64(0))
	require.WithinDuration(t, now, b.CreatedAt.Time(), time.Second)
	require.WithinDuration(t, now, b.ModifiedAt.Time(), time.Second)

}

func newDB(t *testing.T) *Database {
	t.Helper()

	// d := t.TempDir()
	d := "/tmp"
	path := filepath.Join(d, fmt.Sprintf("%d.bedrock", time.Now().UnixNano()))
	db, err := Open(path)
	require.NoError(t, err)
	return db
}

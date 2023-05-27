package persistence

import (
	"context"
	"testing"

	"ara.sh/iabdaccounting/bedrock/model"
	"github.com/stretchr/testify/require"
)

func TestInsertItem(t *testing.T) {
	t.Parallel()

	in := model.CreateItemInput{
		Name:     "Local Bahá'í Fund",
		Shortcut: "LBF",
	}
	db := newDB(t)
	actual, err := db.CreateItem(context.Background(), in)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assertValidBase(t, actual.Base)
}

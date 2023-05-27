package persistence

import (
	"context"
	"testing"
	"time"

	"ara.sh/iabdaccounting/bedrock/datetime"
	"ara.sh/iabdaccounting/bedrock/model"
	"github.com/stretchr/testify/require"
)

func TestInsertAccount(t *testing.T) {
	t.Parallel()

	db := newDB(t)
	in := model.CreateAccountInput{
		Type:            model.AccountBank,
		Name:            "Local Fund",
		Description:     "Checking @ BofA",
		StartingBalance: 3000_00,
		Denomination:    model.USD,
		StartingDate:    datetime.FromTime(time.Date(2023, time.May, 1, 0, 0, 0, 0, time.Local)),
	}
	actual, err := db.CreateAccount(context.Background(), in)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assertValidBase(t, actual.Base)
}

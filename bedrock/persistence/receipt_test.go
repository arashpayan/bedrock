package persistence

import (
	"context"
	"testing"
	"time"

	"ara.sh/iabdaccounting/bedrock/datetime"
	"ara.sh/iabdaccounting/bedrock/model"
	"github.com/stretchr/testify/require"
)

func TestInsertReceipt(t *testing.T) {
	t.Parallel()

	db := newDB(t)
	ctx := context.Background()
	party, err := db.CreateParty(ctx, model.CreatePartyInput{
		Name: "Happy Pappy",
	})
	require.NoError(t, err)

	lbf, err := db.CreateItem(ctx, model.CreateItemInput{Name: "Local Fund", Shortcut: "LBF"})
	require.NoError(t, err)
	hf, err := db.CreateItem(ctx, model.CreateItemInput{Name: "Humanitarian Fund", Shortcut: "HF"})
	require.NoError(t, err)

	input := model.CreateReceiptInput{
		CustomerID: party.ID,
		SoldAt:     datetime.FromTime(time.Now().Add(-24 * time.Hour)),
		Items: []model.CreateReceiptItem{
			{
				ItemID:      lbf.ID,
				Description: "Local Fund",
				Price:       30_00,
			},
			{
				ItemID:      hf.ID,
				Description: "Humanitarian Fund",
				Price:       10_00,
			},
		},
	}

	actual, err := db.CreateReceipt(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assertValidBase(t, actual.Base)
	// make sure a human id was generated
	require.NotEmpty(t, actual.HumanID)
}

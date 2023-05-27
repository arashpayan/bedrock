package persistence

import (
	"context"
	"testing"
	"time"

	"ara.sh/iabdaccounting/bedrock/datetime"
	"ara.sh/iabdaccounting/bedrock/model"
	"ara.sh/iabdaccounting/bedrock/ptr"
	"github.com/stretchr/testify/require"
)

func TestCreateDeposit(t *testing.T) {
	t.Parallel()

	db := newDB(t)
	ctx := context.Background()
	acct, err := db.CreateAccount(context.Background(), model.CreateAccountInput{
		Type:            model.AccountBank,
		Name:            "Local Fund",
		Description:     "The Bank Account",
		Denomination:    model.USD,
		StartingBalance: 1500_00,
		StartingDate:    datetime.FromTime(time.Now().Add(-96 * time.Hour)),
	})
	require.NoError(t, err)

	lbf, err := db.CreateItem(ctx, model.CreateItemInput{
		Name:     "Local Bahá'í Fund",
		Shortcut: "LBF",
	})
	require.NoError(t, err)

	party, err := db.CreateParty(ctx, model.CreatePartyInput{
		Name:         "Shawn Spencer",
		EmailAddress: ptr.Of("shawn.spencer@psyche.com"),
	})
	require.NoError(t, err)

	rcpt, err := db.CreateReceipt(ctx, model.CreateReceiptInput{
		CustomerID: party.ID,
		SoldAt:     datetime.FromTime(time.Now().Add(-72 * time.Hour)),
		Items: []model.CreateReceiptItem{
			{
				ItemID:      lbf.ID,
				Description: "Local Fund",
				Price:       19_00,
			},
		},
	})
	require.NoError(t, err)

	deposit, err := db.CreateDeposit(ctx, model.CreateDepositInput{
		DepositedAt: datetime.FromTime(time.Now().Add(-48 * time.Hour)),
		AccountID:   acct.ID,
		Memo:        "Deposit at bank",
		UndepositedReceiptIDs: []model.ID{
			rcpt.ID,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, deposit)
}

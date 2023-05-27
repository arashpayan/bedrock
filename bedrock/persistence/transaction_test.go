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

func TestCreateWithDrawal(t *testing.T) {
	t.Parallel()

	db := newDB(t)
	ctx := context.Background()
	lfAcct, err := db.CreateAccount(ctx, model.CreateAccountInput{
		Type:            model.AccountBank,
		Name:            "Local Fund",
		Description:     "Community bank account",
		Denomination:    model.USD,
		StartingBalance: 2000_00,
		StartingDate:    datetime.FromTime(time.Now().Add(-96 * time.Hour)),
	})
	require.NoError(t, err)

	hfAcct, err := db.CreateAccount(ctx, model.CreateAccountInput{
		Type:            model.AccountBank,
		Name:            "Humanitarian Fund",
		Description:     "Sub account of Local Fund",
		Denomination:    model.USD,
		StartingBalance: 0,
		StartingDate:    datetime.FromTime(time.Now().Add(-96 * time.Hour)),
		ParentID:        &lfAcct.ID,
	})
	require.NoError(t, err)

	johnDoe, err := db.CreateParty(ctx, model.CreatePartyInput{
		Name: "John Doe",
	})
	require.NoError(t, err)

	wdIn := model.CreateWithdrawalTransactionInput{
		AccountID:    hfAcct.ID,
		Amount:       -20_00,
		CheckNumber:  "1",
		Memo:         "Backpack project",
		Method:       ptr.Of(model.Check),
		PayeeID:      johnDoe.ID,
		TransactedAt: datetime.FromTime(time.Now().Add(-72 * time.Hour)),
	}
	withdrawal, err := db.CreateWithdrawal(ctx, wdIn)
	require.NoError(t, err)
	assertValidBase(t, withdrawal.Base)
	require.Equal(t, wdIn.AccountID, withdrawal.AccountID)
	require.Equal(t, wdIn.Amount, withdrawal.Amount)
	require.NotNil(t, withdrawal.CheckNumber)
	require.Equal(t, wdIn.CheckNumber, *withdrawal.CheckNumber)
	require.Equal(t, wdIn.Memo, withdrawal.Memo)
	require.NotNil(t, withdrawal.Method)
	require.Equal(t, wdIn.Method, withdrawal.Method)
	require.NotNil(t, withdrawal.PayeeID)
	require.Equal(t, wdIn.PayeeID, *withdrawal.PayeeID)
	require.Equal(t, wdIn.TransactedAt, withdrawal.TransactedAt)
}

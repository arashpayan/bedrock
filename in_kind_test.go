package bedrock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// inKindItems builds a one-line in-kind input referencing the given fund and
// expense category.
func inKindItems(fundID, categoryID ID, cents int64) []ReceiptItemInput {
	return []ReceiptItemInput{
		{ItemID: fundID, CategoryID: &categoryID, Description: "Catering", Price: NewMoney(cents, CurrencyUSD)},
	}
}

func TestInKindReceiptRoundTrip(t *testing.T) {
	db := testDB(t)
	_, _, fund, party, category := setupTestData(t, db)

	receipt, err := db.CreateReceiptWithItems(party.ID, time.Now(), "donated food", true, inKindItems(fund.ID, category.ID, 800_00))
	require.NoError(t, err)
	assert.True(t, receipt.IsInKind)

	// Reloading the receipt preserves the in-kind flag.
	reloaded, err := db.Receipt(receipt.ID)
	require.NoError(t, err)
	assert.True(t, reloaded.IsInKind)

	// The line item carries its category.
	items, err := db.ReceiptItems(receipt.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.NotNil(t, items[0].CategoryID)
	assert.Equal(t, category.ID, *items[0].CategoryID)

	// FullReceipt resolves both the fund and the expense category.
	full, err := db.FullReceipt(receipt.ID)
	require.NoError(t, err)
	require.Len(t, full.Items, 1)
	require.NotNil(t, full.Items[0].Category)
	assert.Equal(t, category.ID, full.Items[0].Category.ID)
	assert.Equal(t, int64(800_00), full.Total.Amount)
}

func TestCashReceiptRejectsCategory(t *testing.T) {
	db := testDB(t)
	_, _, fund, party, category := setupTestData(t, db)

	// A category on a non-in-kind contribution is rejected.
	_, err := db.CreateReceiptWithItems(party.ID, time.Now(), "", false, inKindItems(fund.ID, category.ID, 100_00))
	assert.Error(t, err)
}

func TestInKindExcludedFromCashWorkflows(t *testing.T) {
	db := testDB(t)
	_, account, fund, party, category := setupTestData(t, db)

	inKind, err := db.CreateReceiptWithItems(party.ID, time.Now(), "", true, inKindItems(fund.ID, category.ID, 500_00))
	require.NoError(t, err)

	// A normal cash receipt, for contrast.
	cash, err := db.CreateReceiptWithItems(party.ID, time.Now(), "", false, []ReceiptItemInput{
		{ItemID: fund.ID, Price: NewMoney(250_00, CurrencyUSD)},
	})
	require.NoError(t, err)

	// UndepositedReceipts lists the cash one but never the in-kind one.
	undeposited, err := db.UndepositedReceipts()
	require.NoError(t, err)
	ids := make(map[ID]bool, len(undeposited))
	for _, r := range undeposited {
		ids[r.ID] = true
	}
	assert.True(t, ids[cash.ID], "cash receipt should be undeposited")
	assert.False(t, ids[inKind.ID], "in-kind receipt must not appear as undeposited")

	// Direct assignment to a deposit is rejected.
	deposit, err := db.CreateDeposit(account.ID, NewMoney(500_00, CurrencyUSD), TransactionMethodInBranch, "", time.Now())
	require.NoError(t, err)
	_, err = db.AssignReceiptToTransaction(inKind.ID, deposit.ID)
	assert.Error(t, err, "in-kind receipt cannot be assigned to a deposit")

	// Creating a deposit that includes the in-kind receipt is rejected.
	_, err = db.CreateDepositWithReceipts(account.ID, NewMoney(500_00, CurrencyUSD), TransactionMethodInBranch, "", time.Now(), []ID{inKind.ID})
	assert.Error(t, err, "deposit including an in-kind receipt must be rejected")
}

func TestInKindCountsTowardGoal(t *testing.T) {
	db := testDB(t)
	_, _, fund, party, category := setupTestData(t, db)
	// setupTestData's fund counts toward the goal by default.

	inYear := time.Date(2026, time.June, 1, 12, 0, 0, 0, time.UTC)
	_, err := db.CreateReceiptWithItems(party.ID, inYear, "", true, inKindItems(fund.ID, category.ID, 900_00))
	require.NoError(t, err)

	progress, err := db.FundraisingProgress(2026)
	require.NoError(t, err)
	assert.Equal(t, int64(900_00), progress.Raised.Amount, "in-kind contributions count toward the goal")
}

func TestUpdateReceiptTogglesInKind(t *testing.T) {
	db := testDB(t)
	_, _, fund, party, category := setupTestData(t, db)

	// Start as a cash receipt.
	receipt, err := db.CreateReceiptWithItems(party.ID, time.Now(), "", false, []ReceiptItemInput{
		{ItemID: fund.ID, Price: NewMoney(100_00, CurrencyUSD)},
	})
	require.NoError(t, err)
	assert.False(t, receipt.IsInKind)

	// Convert it to in-kind with a category.
	updated, err := db.UpdateReceiptWithItems(receipt.ID, party.ID, time.Now(), "", true, inKindItems(fund.ID, category.ID, 100_00))
	require.NoError(t, err)
	assert.True(t, updated.IsInKind)

	items, err := db.ReceiptItems(receipt.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.NotNil(t, items[0].CategoryID)
}

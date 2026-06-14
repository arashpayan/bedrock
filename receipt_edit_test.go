package bedrock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateReceiptWithItems verifies that a receipt's header and line items can
// be edited together, replacing the previous item set.
func TestUpdateReceiptWithItems(t *testing.T) {
	db := testDB(t)
	_, _, item, party, _ := setupTestData(t, db)

	other, err := db.CreateParty("Other Contributor", nil, nil, nil, nil)
	require.NoError(t, err)
	fund2, err := db.CreateItem("Humanitarian Fund")
	require.NoError(t, err)

	soldAt := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	receipt, err := db.CreateReceiptWithItems(party.ID, soldAt, "original", []ReceiptItemInput{
		{ItemID: item.ID, Price: NewMoney(1000, CurrencyUSD)},
	})
	require.NoError(t, err)

	newSoldAt := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	updated, err := db.UpdateReceiptWithItems(receipt.ID, other.ID, newSoldAt, "edited", []ReceiptItemInput{
		{ItemID: item.ID, Price: NewMoney(2500, CurrencyUSD)},
		{ItemID: fund2.ID, Price: NewMoney(500, CurrencyUSD)},
	})
	require.NoError(t, err)
	assert.Equal(t, other.ID, updated.CustomerID)
	assert.Equal(t, "edited", updated.Memo)
	assert.Equal(t, newSoldAt.Unix(), updated.SoldAt.Unix())
	assert.Equal(t, receipt.HumanID, updated.HumanID, "human ID must be preserved across edits")

	// The item set is fully replaced, not appended.
	items, err := db.ReceiptItems(receipt.ID)
	require.NoError(t, err)
	require.Len(t, items, 2)
	var total int64
	for _, ri := range items {
		total += ri.Price.Amount
	}
	assert.Equal(t, int64(3000), total)
}

// TestUpdateReceiptWithItems_Validation covers the rejected inputs.
func TestUpdateReceiptWithItems_Validation(t *testing.T) {
	db := testDB(t)
	_, _, item, party, _ := setupTestData(t, db)

	receipt, err := db.CreateReceiptWithItems(party.ID, time.Now(), "", []ReceiptItemInput{
		{ItemID: item.ID, Price: NewMoney(1000, CurrencyUSD)},
	})
	require.NoError(t, err)

	t.Run("NoItems", func(t *testing.T) {
		_, err := db.UpdateReceiptWithItems(receipt.ID, party.ID, time.Now(), "", nil)
		require.Error(t, err)
	})

	t.Run("NonPositivePrice", func(t *testing.T) {
		_, err := db.UpdateReceiptWithItems(receipt.ID, party.ID, time.Now(), "", []ReceiptItemInput{
			{ItemID: item.ID, Price: NewMoney(0, CurrencyUSD)},
		})
		require.Error(t, err)
	})

	t.Run("MixedCurrency", func(t *testing.T) {
		_, err := db.UpdateReceiptWithItems(receipt.ID, party.ID, time.Now(), "", []ReceiptItemInput{
			{ItemID: item.ID, Price: NewMoney(1000, CurrencyUSD)},
			{ItemID: item.ID, Price: NewMoney(1000, CurrencyCAD)},
		})
		require.Error(t, err)
	})

	t.Run("MissingCustomer", func(t *testing.T) {
		_, err := db.UpdateReceiptWithItems(receipt.ID, ID(99999), time.Now(), "", []ReceiptItemInput{
			{ItemID: item.ID, Price: NewMoney(1000, CurrencyUSD)},
		})
		require.Error(t, err)
	})

	// A failed update must leave the original items untouched.
	items, err := db.ReceiptItems(receipt.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, int64(1000), items[0].Price.Amount)
}

// TestUpdateReceiptWithItems_Deposited verifies a deposited receipt cannot be
// edited.
func TestUpdateReceiptWithItems_Deposited(t *testing.T) {
	db := testDB(t)
	_, account, item, party, _ := setupTestData(t, db)

	receipt, err := db.CreateReceiptWithItems(party.ID, time.Now(), "", []ReceiptItemInput{
		{ItemID: item.ID, Price: NewMoney(1000, CurrencyUSD)},
	})
	require.NoError(t, err)

	_, err = db.CreateDepositWithReceipts(account.ID, NewMoney(1000, CurrencyUSD),
		TransactionMethodCheck, "deposit", time.Now(), []ID{receipt.ID})
	require.NoError(t, err)

	_, err = db.UpdateReceiptWithItems(receipt.ID, party.ID, time.Now(), "edited", []ReceiptItemInput{
		{ItemID: item.ID, Price: NewMoney(2000, CurrencyUSD)},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deposited")
}

// TestDeleteReceiptWithItems verifies a receipt and its items are removed.
func TestDeleteReceiptWithItems(t *testing.T) {
	db := testDB(t)
	_, _, item, party, _ := setupTestData(t, db)

	receipt, err := db.CreateReceiptWithItems(party.ID, time.Now(), "", []ReceiptItemInput{
		{ItemID: item.ID, Price: NewMoney(1000, CurrencyUSD)},
		{ItemID: item.ID, Price: NewMoney(500, CurrencyUSD)},
	})
	require.NoError(t, err)

	require.NoError(t, db.DeleteReceiptWithItems(receipt.ID))

	_, err = db.Receipt(receipt.ID)
	require.Error(t, err, "receipt should be gone")
	items, err := db.ReceiptItems(receipt.ID)
	require.NoError(t, err)
	assert.Empty(t, items, "items should be gone")
}

// TestDeleteReceiptWithItems_Deposited verifies a deposited receipt cannot be
// deleted.
func TestDeleteReceiptWithItems_Deposited(t *testing.T) {
	db := testDB(t)
	_, account, item, party, _ := setupTestData(t, db)

	receipt, err := db.CreateReceiptWithItems(party.ID, time.Now(), "", []ReceiptItemInput{
		{ItemID: item.ID, Price: NewMoney(1000, CurrencyUSD)},
	})
	require.NoError(t, err)

	_, err = db.CreateDepositWithReceipts(account.ID, NewMoney(1000, CurrencyUSD),
		TransactionMethodCheck, "deposit", time.Now(), []ID{receipt.ID})
	require.NoError(t, err)

	err = db.DeleteReceiptWithItems(receipt.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deposited")

	// The receipt must still exist.
	_, err = db.Receipt(receipt.ID)
	require.NoError(t, err)
}

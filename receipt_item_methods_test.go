package bedrock

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReceiptItemCRUD(t *testing.T) {
	// Setup: Create a test database with necessary data
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "receipt_item_test.bedrock")

	db, err := New(dbPath, "Test Assembly", time.UTC, CurrencyUSD)
	require.NoError(t, err, "Failed to create database")
	defer db.Close()

	// Create a party for receipts
	party, err := db.CreateParty("Test Contributor", nil, nil, nil, nil)
	require.NoError(t, err, "Failed to create party")

	// Create items for receipt items
	item1, err := db.CreateItem("Local Fund")
	require.NoError(t, err, "Failed to create item 1")

	item2, err := db.CreateItem("Humanitarian Fund")
	require.NoError(t, err, "Failed to create item 2")

	// Create a receipt
	receipt, err := db.CreateReceipt(party.ID, time.Now())
	require.NoError(t, err, "Failed to create receipt")

	t.Run("CreateReceiptItem", func(t *testing.T) {
		ri, err := db.CreateReceiptItem(
			receipt.ID,
			item1.ID,
			"Monthly contribution",
			NewMoney(5000, CurrencyUSD),
		)
		require.NoError(t, err, "CreateReceiptItem should succeed")

		assert.Equal(t, receipt.ID, ri.ReceiptID)
		assert.Equal(t, item1.ID, ri.ItemID)
		assert.Equal(t, "Monthly contribution", ri.Description)
		assert.Equal(t, int64(5000), ri.Price.Amount)
		assert.Equal(t, CurrencyUSD, ri.Price.Currency)
	})

	t.Run("CreateReceiptItemInvalidReceipt", func(t *testing.T) {
		_, err := db.CreateReceiptItem(
			99999, // Non-existent receipt
			item1.ID,
			"Test",
			NewMoney(1000, CurrencyUSD),
		)
		assert.Error(t, err, "Should fail with invalid receipt")
	})

	t.Run("CreateReceiptItemInvalidItem", func(t *testing.T) {
		_, err := db.CreateReceiptItem(
			receipt.ID,
			99999, // Non-existent item
			"Test",
			NewMoney(1000, CurrencyUSD),
		)
		assert.Error(t, err, "Should fail with invalid item")
	})

	t.Run("CreateReceiptItemZeroPrice", func(t *testing.T) {
		_, err := db.CreateReceiptItem(
			receipt.ID,
			item1.ID,
			"Test",
			NewMoney(0, CurrencyUSD),
		)
		assert.Error(t, err, "Should fail with zero price")
	})

	t.Run("CreateReceiptItemNegativePrice", func(t *testing.T) {
		_, err := db.CreateReceiptItem(
			receipt.ID,
			item1.ID,
			"Test",
			NewMoney(-100, CurrencyUSD),
		)
		assert.Error(t, err, "Should fail with negative price")
	})

	t.Run("ReceiptItem", func(t *testing.T) {
		// Create a new receipt item to test retrieval
		created, err := db.CreateReceiptItem(
			receipt.ID,
			item2.ID,
			"Special contribution",
			NewMoney(2500, CurrencyCAD),
		)
		require.NoError(t, err, "Failed to create receipt item")

		retrieved, err := db.ReceiptItem(created.ID)
		require.NoError(t, err, "ReceiptItem should succeed")

		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, item2.ID, retrieved.ItemID)
		assert.Equal(t, "Special contribution", retrieved.Description)
		assert.Equal(t, int64(2500), retrieved.Price.Amount)
		assert.Equal(t, CurrencyCAD, retrieved.Price.Currency)
	})

	t.Run("ReceiptItemNotFound", func(t *testing.T) {
		_, err := db.ReceiptItem(99999)
		assert.Error(t, err, "Should fail with non-existent ID")
	})

	t.Run("ReceiptItems", func(t *testing.T) {
		// Create a new receipt with multiple items
		receipt2, err := db.CreateReceipt(party.ID, time.Now())
		require.NoError(t, err, "Failed to create receipt 2")

		_, err = db.CreateReceiptItem(receipt2.ID, item1.ID, "Item A", NewMoney(1000, CurrencyUSD))
		require.NoError(t, err)

		_, err = db.CreateReceiptItem(receipt2.ID, item2.ID, "Item B", NewMoney(2000, CurrencyUSD))
		require.NoError(t, err)

		_, err = db.CreateReceiptItem(receipt2.ID, item1.ID, "Item C", NewMoney(3000, CurrencyUSD))
		require.NoError(t, err)

		items, err := db.ReceiptItems(receipt2.ID)
		require.NoError(t, err, "ReceiptItems should succeed")

		assert.Len(t, items, 3)
		assert.Equal(t, "Item A", items[0].Description)
		assert.Equal(t, "Item B", items[1].Description)
		assert.Equal(t, "Item C", items[2].Description)
	})

	t.Run("ReceiptItemsEmpty", func(t *testing.T) {
		// Create a receipt with no items
		receipt3, err := db.CreateReceipt(party.ID, time.Now())
		require.NoError(t, err, "Failed to create receipt 3")

		items, err := db.ReceiptItems(receipt3.ID)
		require.NoError(t, err, "ReceiptItems should succeed for empty receipt")

		assert.Len(t, items, 0)
	})

	t.Run("UpdateReceiptItem", func(t *testing.T) {
		// Create a receipt item to update
		ri, err := db.CreateReceiptItem(
			receipt.ID,
			item1.ID,
			"Original description",
			NewMoney(1000, CurrencyUSD),
		)
		require.NoError(t, err, "Failed to create receipt item")

		// Update the receipt item
		updated, err := db.UpdateReceiptItem(ri.ID, "Updated description", NewMoney(1500, CurrencyCAD))
		require.NoError(t, err, "UpdateReceiptItem should succeed")

		assert.Equal(t, ri.ID, updated.ID)
		assert.Equal(t, "Updated description", updated.Description)
		assert.Equal(t, int64(1500), updated.Price.Amount)
		assert.Equal(t, CurrencyCAD, updated.Price.Currency)

		// Verify update persisted
		retrieved, err := db.ReceiptItem(ri.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated description", retrieved.Description)
		assert.Equal(t, int64(1500), retrieved.Price.Amount)
	})

	t.Run("UpdateReceiptItemZeroPrice", func(t *testing.T) {
		ri, err := db.CreateReceiptItem(
			receipt.ID,
			item1.ID,
			"Test",
			NewMoney(1000, CurrencyUSD),
		)
		require.NoError(t, err)

		_, err = db.UpdateReceiptItem(ri.ID, "Test", NewMoney(0, CurrencyUSD))
		assert.Error(t, err, "Should fail with zero price")
	})

	t.Run("DeleteReceiptItem", func(t *testing.T) {
		// Create a receipt item to delete
		ri, err := db.CreateReceiptItem(
			receipt.ID,
			item1.ID,
			"To be deleted",
			NewMoney(500, CurrencyUSD),
		)
		require.NoError(t, err, "Failed to create receipt item")

		// Delete it
		err = db.DeleteReceiptItem(ri.ID)
		require.NoError(t, err, "DeleteReceiptItem should succeed")

		// Verify deletion
		_, err = db.ReceiptItem(ri.ID)
		assert.Error(t, err, "Should fail to retrieve deleted item")
	})

	t.Run("DeleteReceiptItemNotFound", func(t *testing.T) {
		err := db.DeleteReceiptItem(99999)
		assert.Error(t, err, "Should fail with non-existent ID")
	})
}

package bedrock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateDeposit verifies that a deposit's header and receipt set can be
// edited together, replacing the previous receipt set and recomputing the
// transaction amount.
func TestUpdateDeposit(t *testing.T) {
	db := testDB(t)
	_, account, item, party, _ := setupTestData(t, db)

	soldAt := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	first := createTestReceipt(t, db, party.ID, item.ID, soldAt, NewMoney(1000, CurrencyUSD))
	second := createTestReceipt(t, db, party.ID, item.ID, soldAt, NewMoney(2500, CurrencyUSD))

	depositedAt := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)
	deposit, err := db.CreateDepositWithReceipts(account.ID, NewMoney(1000, CurrencyUSD), TransactionMethodInBranch, "original", depositedAt, []ID{first.ID})
	require.NoError(t, err)
	require.Equal(t, int64(1000), deposit.Amount)

	// The date was wrong and a second receipt belongs in the same deposit.
	correctedAt := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC)
	updated, err := db.UpdateDeposit(deposit.ID, TransactionMethodMobileApp, "corrected", correctedAt, []ID{first.ID, second.ID})
	require.NoError(t, err)
	assert.Equal(t, "corrected", updated.Memo)
	assert.Equal(t, TransactionMethodMobileApp, *updated.Method)
	assert.Equal(t, correctedAt.Unix(), updated.TransactedAt.Unix())
	assert.Equal(t, int64(3500), updated.Amount, "amount must be recomputed from the new receipts")
	assert.Equal(t, account.ID, updated.AccountID, "account must not change")

	receipts, err := db.ReceiptsByTransaction(deposit.ID)
	require.NoError(t, err)
	assert.Len(t, receipts, 2)
}

// TestUpdateDeposit_ReleasesDroppedReceipts verifies that a receipt removed
// from a deposit returns to the undeposited pool.
func TestUpdateDeposit_ReleasesDroppedReceipts(t *testing.T) {
	db := testDB(t)
	_, account, item, party, _ := setupTestData(t, db)

	soldAt := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	kept := createTestReceipt(t, db, party.ID, item.ID, soldAt, NewMoney(1000, CurrencyUSD))
	dropped := createTestReceipt(t, db, party.ID, item.ID, soldAt, NewMoney(2500, CurrencyUSD))

	deposit, err := db.CreateDepositWithReceipts(account.ID, NewMoney(3500, CurrencyUSD), TransactionMethodInBranch, "", soldAt, []ID{kept.ID, dropped.ID})
	require.NoError(t, err)

	updated, err := db.UpdateDeposit(deposit.ID, TransactionMethodInBranch, "", soldAt, []ID{kept.ID})
	require.NoError(t, err)
	assert.Equal(t, int64(1000), updated.Amount)

	reloaded, err := db.Receipt(dropped.ID)
	require.NoError(t, err)
	assert.Nil(t, reloaded.TransactionID, "a dropped receipt must become undeposited again")

	undeposited, err := db.UndepositedReceipts()
	require.NoError(t, err)
	require.Len(t, undeposited, 1)
	assert.Equal(t, dropped.ID, undeposited[0].ID)
}

// TestUpdateDeposit_Validation covers the rejected inputs.
func TestUpdateDeposit_Validation(t *testing.T) {
	db := testDB(t)
	_, account, item, party, category := setupTestData(t, db)

	soldAt := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	receipt := createTestReceipt(t, db, party.ID, item.ID, soldAt, NewMoney(1000, CurrencyUSD))
	deposit, err := db.CreateDepositWithReceipts(account.ID, NewMoney(1000, CurrencyUSD), TransactionMethodInBranch, "", soldAt, []ID{receipt.ID})
	require.NoError(t, err)

	t.Run("NoReceipts", func(t *testing.T) {
		_, err := db.UpdateDeposit(deposit.ID, TransactionMethodInBranch, "", soldAt, nil)
		assert.Error(t, err)
	})

	t.Run("ReceiptInAnotherDeposit", func(t *testing.T) {
		other := createTestReceipt(t, db, party.ID, item.ID, soldAt, NewMoney(500, CurrencyUSD))
		otherDeposit, err := db.CreateDepositWithReceipts(account.ID, NewMoney(500, CurrencyUSD), TransactionMethodInBranch, "", soldAt, []ID{other.ID})
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, db.DeleteDeposit(otherDeposit.ID)) })

		_, err = db.UpdateDeposit(deposit.ID, TransactionMethodInBranch, "", soldAt, []ID{receipt.ID, other.ID})
		assert.Error(t, err)
	})

	t.Run("InKindReceipt", func(t *testing.T) {
		inKind, err := db.CreateReceiptWithItems(party.ID, soldAt, "", true, []ReceiptItemInput{
			{ItemID: item.ID, CategoryID: &category.ID, Price: NewMoney(750, CurrencyUSD)},
		})
		require.NoError(t, err)

		_, err = db.UpdateDeposit(deposit.ID, TransactionMethodInBranch, "", soldAt, []ID{receipt.ID, inKind.ID})
		assert.Error(t, err)
	})

	t.Run("CurrencyMismatchWithAccount", func(t *testing.T) {
		cad := createTestReceipt(t, db, party.ID, item.ID, soldAt, NewMoney(1000, CurrencyCAD))
		_, err := db.UpdateDeposit(deposit.ID, TransactionMethodInBranch, "", soldAt, []ID{cad.ID})
		assert.Error(t, err)
	})

	t.Run("NotADeposit", func(t *testing.T) {
		check, err := db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "", soldAt, nil, []ExpenseItem{
			{CategoryID: category.ID, Amount: NewMoney(1000, CurrencyUSD)},
		})
		require.NoError(t, err)

		_, err = db.UpdateDeposit(check.ID, TransactionMethodInBranch, "", soldAt, []ID{receipt.ID})
		assert.Error(t, err)
	})

	t.Run("OpeningBalance", func(t *testing.T) {
		openingDate := time.Date(2025, 12, 1, 12, 0, 0, 0, time.UTC)
		opened, err := db.CreateBankAccount("Opened Checking", AccountTypeChecking, CurrencyUSD, nil, "", true, NewMoney(50000, CurrencyUSD), openingDate)
		require.NoError(t, err)
		openingTx, err := db.OpeningBalanceTransaction(opened.ID)
		require.NoError(t, err)
		require.NotNil(t, openingTx)

		_, err = db.UpdateDeposit(openingTx.ID, TransactionMethodInBranch, "", soldAt, []ID{receipt.ID})
		assert.Error(t, err, "an opening balance is edited through its account")
		assert.Error(t, db.DeleteDeposit(openingTx.ID))
	})

	// None of the rejections above may have disturbed the deposit.
	reloaded, err := db.Transaction(deposit.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), reloaded.Amount)
	held, err := db.ReceiptsByTransaction(deposit.ID)
	require.NoError(t, err)
	require.Len(t, held, 1)
	assert.Equal(t, receipt.ID, held[0].ID)
}

// TestUpdateDeposit_Reconciled verifies that a reconciled deposit can be
// neither edited nor deleted, preserving the reconciliation's cleared balance.
func TestUpdateDeposit_Reconciled(t *testing.T) {
	db := testDB(t)
	_, account, item, party, _ := setupTestData(t, db)

	soldAt := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	receipt := createTestReceipt(t, db, party.ID, item.ID, soldAt, NewMoney(1000, CurrencyUSD))
	deposit, err := db.CreateDepositWithReceipts(account.ID, NewMoney(1000, CurrencyUSD), TransactionMethodInBranch, "", soldAt, []ID{receipt.ID})
	require.NoError(t, err)

	statementDate := time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC)
	rec, err := db.StartReconciliation(account.ID, statementDate, NewMoney(1000, CurrencyUSD))
	require.NoError(t, err)
	require.NoError(t, db.ClearTransaction(rec.ID, deposit.ID))

	_, err = db.UpdateDeposit(deposit.ID, TransactionMethodInBranch, "edited", soldAt, []ID{receipt.ID})
	assert.Error(t, err, "a reconciled deposit must not be editable")

	assert.Error(t, db.DeleteDeposit(deposit.ID), "a reconciled deposit must not be deletable")
}

// TestDeleteDeposit verifies that deleting a deposit removes the transaction
// and returns its receipts to the undeposited pool.
func TestDeleteDeposit(t *testing.T) {
	db := testDB(t)
	_, account, item, party, _ := setupTestData(t, db)

	soldAt := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	receipt := createTestReceipt(t, db, party.ID, item.ID, soldAt, NewMoney(1000, CurrencyUSD))
	deposit, err := db.CreateDepositWithReceipts(account.ID, NewMoney(1000, CurrencyUSD), TransactionMethodInBranch, "", soldAt, []ID{receipt.ID})
	require.NoError(t, err)

	require.NoError(t, db.DeleteDeposit(deposit.ID))

	_, err = db.Transaction(deposit.ID)
	assert.Error(t, err, "the transaction should be gone")

	reloaded, err := db.Receipt(receipt.ID)
	require.NoError(t, err, "the contribution itself must survive")
	assert.Nil(t, reloaded.TransactionID, "its receipt must become undeposited again")
}

// TestDeleteDeposit_NotADeposit verifies that a check cannot be deleted via
// DeleteDeposit.
func TestDeleteDeposit_NotADeposit(t *testing.T) {
	db := testDB(t)
	_, account, _, payee, category := setupTestData(t, db)

	check, err := db.CreateWithdrawal(account.ID, payee.ID, TransactionMethodCheck, "", time.Now(), nil, []ExpenseItem{
		{CategoryID: category.ID, Amount: NewMoney(1000, CurrencyUSD)},
	})
	require.NoError(t, err)

	assert.Error(t, db.DeleteDeposit(check.ID))
}

// createTestReceipt creates a single-line cash receipt for the deposit tests.
func createTestReceipt(t *testing.T, db *DB, customerID, itemID ID, soldAt time.Time, price Money) *Receipt {
	t.Helper()

	receipt, err := db.CreateReceiptWithItems(customerID, soldAt, "", false, []ReceiptItemInput{
		{ItemID: itemID, Price: price},
	})
	require.NoError(t, err, "failed to create test receipt")
	return receipt
}

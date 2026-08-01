package bedrock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// checkNum returns the address of a check-number literal for the test calls.
func checkNum(s string) *string { return &s }

// TestUpdateWithdrawal verifies that a check's header and expense lines can be
// edited together, replacing the previous expense set and recomputing the
// transaction amount.
func TestUpdateWithdrawal(t *testing.T) {
	db := testDB(t)
	_, account, _, payee, category := setupTestData(t, db)

	otherPayee, err := db.CreateParty("Other Vendor", nil, nil, nil, nil)
	require.NoError(t, err)
	cat2, err := db.CreateCategory("Travel")
	require.NoError(t, err)

	date := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	check, err := db.CreateWithdrawal(account.ID, payee.ID, TransactionMethodCheck, "original", date, checkNum("101"), []ExpenseItem{
		{CategoryID: category.ID, Amount: NewMoney(1000, CurrencyUSD)},
	})
	require.NoError(t, err)
	require.Equal(t, int64(-1000), check.Amount)

	newDate := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	updated, err := db.UpdateWithdrawal(check.ID, otherPayee.ID, TransactionMethodCheck, "edited", newDate, checkNum("102"), []ExpenseItem{
		{CategoryID: category.ID, Amount: NewMoney(2500, CurrencyUSD)},
		{CategoryID: cat2.ID, Amount: NewMoney(500, CurrencyUSD)},
	})
	require.NoError(t, err)
	assert.Equal(t, otherPayee.ID, *updated.PayeeID)
	assert.Equal(t, "edited", updated.Memo)
	assert.Equal(t, "102", *updated.CheckNumber)
	assert.Equal(t, newDate.Unix(), updated.TransactedAt.Unix())
	assert.Equal(t, int64(-3000), updated.Amount, "amount must be recomputed from the new expenses")
	assert.Equal(t, account.ID, updated.AccountID, "account must not change")

	// The expense set is fully replaced, not appended.
	expenses, err := db.ExpensesForTransaction(check.ID)
	require.NoError(t, err)
	require.Len(t, expenses, 2)
	var total int64
	for _, e := range expenses {
		total += e.Amount.Amount
	}
	assert.Equal(t, int64(3000), total)
}

// TestUpdateWithdrawal_Validation covers the rejected inputs.
func TestUpdateWithdrawal_Validation(t *testing.T) {
	db := testDB(t)
	_, account, _, payee, category := setupTestData(t, db)

	check, err := db.CreateWithdrawal(account.ID, payee.ID, TransactionMethodCheck, "", time.Now(), nil, []ExpenseItem{
		{CategoryID: category.ID, Amount: NewMoney(1000, CurrencyUSD)},
	})
	require.NoError(t, err)

	t.Run("NoExpenses", func(t *testing.T) {
		_, err := db.UpdateWithdrawal(check.ID, payee.ID, TransactionMethodCheck, "", time.Now(), nil, nil)
		assert.Error(t, err)
	})

	t.Run("NonPositiveAmount", func(t *testing.T) {
		_, err := db.UpdateWithdrawal(check.ID, payee.ID, TransactionMethodCheck, "", time.Now(), nil, []ExpenseItem{
			{CategoryID: category.ID, Amount: NewMoney(0, CurrencyUSD)},
		})
		assert.Error(t, err)
	})

	t.Run("MixedCurrencies", func(t *testing.T) {
		_, err := db.UpdateWithdrawal(check.ID, payee.ID, TransactionMethodCheck, "", time.Now(), nil, []ExpenseItem{
			{CategoryID: category.ID, Amount: NewMoney(1000, CurrencyUSD)},
			{CategoryID: category.ID, Amount: NewMoney(1000, CurrencyCAD)},
		})
		assert.Error(t, err)
	})

	t.Run("CurrencyMismatchWithAccount", func(t *testing.T) {
		_, err := db.UpdateWithdrawal(check.ID, payee.ID, TransactionMethodCheck, "", time.Now(), nil, []ExpenseItem{
			{CategoryID: category.ID, Amount: NewMoney(1000, CurrencyCAD)},
		})
		assert.Error(t, err)
	})

	t.Run("NotAWithdrawal", func(t *testing.T) {
		deposit, err := db.CreateDeposit(account.ID, NewMoney(5000, CurrencyUSD), TransactionMethodInBranch, "", time.Now())
		require.NoError(t, err)
		_, err = db.UpdateWithdrawal(deposit.ID, payee.ID, TransactionMethodCheck, "", time.Now(), nil, []ExpenseItem{
			{CategoryID: category.ID, Amount: NewMoney(1000, CurrencyUSD)},
		})
		assert.Error(t, err)
	})
}

// TestUpdateWithdrawal_Reconciled verifies that a reconciled check cannot be
// edited, preserving the integrity of the reconciliation's cleared balance.
func TestUpdateWithdrawal_Reconciled(t *testing.T) {
	db := testDB(t)
	_, account, _, payee, category := setupTestData(t, db)

	date := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	check, err := db.CreateWithdrawal(account.ID, payee.ID, TransactionMethodCheck, "", date, checkNum("101"), []ExpenseItem{
		{CategoryID: category.ID, Amount: NewMoney(1000, CurrencyUSD)},
	})
	require.NoError(t, err)

	statementDate := time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC)
	rec, err := db.StartReconciliation(account.ID, statementDate, NewMoney(-1000, CurrencyUSD))
	require.NoError(t, err)
	require.NoError(t, db.ClearTransaction(rec.ID, check.ID))

	_, err = db.UpdateWithdrawal(check.ID, payee.ID, TransactionMethodCheck, "edited", date, checkNum("101"), []ExpenseItem{
		{CategoryID: category.ID, Amount: NewMoney(2000, CurrencyUSD)},
	})
	assert.Error(t, err, "a reconciled check must not be editable")

	err = db.DeleteWithdrawal(check.ID)
	assert.Error(t, err, "a reconciled check must not be deletable")
}

// TestDeleteWithdrawal verifies that a check and its expense lines are removed
// together.
func TestDeleteWithdrawal(t *testing.T) {
	db := testDB(t)
	_, account, _, payee, category := setupTestData(t, db)

	check, err := db.CreateWithdrawal(account.ID, payee.ID, TransactionMethodCheck, "", time.Now(), nil, []ExpenseItem{
		{CategoryID: category.ID, Amount: NewMoney(1000, CurrencyUSD)},
	})
	require.NoError(t, err)

	require.NoError(t, db.DeleteWithdrawal(check.ID))

	_, err = db.Transaction(check.ID)
	assert.Error(t, err, "the transaction should be gone")

	expenses, err := db.ExpensesForTransaction(check.ID)
	require.NoError(t, err)
	assert.Empty(t, expenses, "expense lines should be deleted with the check")
}

// TestDeleteWithdrawal_NotAWithdrawal verifies that a deposit cannot be deleted
// via DeleteWithdrawal.
func TestDeleteWithdrawal_NotAWithdrawal(t *testing.T) {
	db := testDB(t)
	_, account, _, _, _ := setupTestData(t, db)

	deposit, err := db.CreateDeposit(account.ID, NewMoney(5000, CurrencyUSD), TransactionMethodInBranch, "", time.Now())
	require.NoError(t, err)

	assert.Error(t, db.DeleteWithdrawal(deposit.ID))
}

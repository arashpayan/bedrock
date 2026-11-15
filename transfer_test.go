package bedrock

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var transferDate = time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

// setupTransferAccounts creates a checking account holding a $1,000.00 opening
// balance and an empty earmark beneath it.
func setupTransferAccounts(t *testing.T, db *DB) (parent, earmark *BankAccount) {
	t.Helper()
	_, err := db.createAssembly("Test Assembly", time.UTC, CurrencyUSD)
	require.NoError(t, err)
	parent, err = db.CreateBankAccount("General Checking", AccountTypeChecking, CurrencyUSD, nil, "", true, NewMoney(100000, CurrencyUSD), transferDate)
	require.NoError(t, err)
	earmark, err = db.CreateBankAccount("Building Fund", AccountTypeEarmark, CurrencyUSD, &parent.ID, "", true, Money{}, time.Time{})
	require.NoError(t, err)
	return parent, earmark
}

// ownBalance returns an account's balance excluding its sub-accounts.
func ownBalance(t *testing.T, db *DB, accountID ID) int64 {
	t.Helper()
	balance, err := db.AccountBalance(accountID, false)
	require.NoError(t, err)
	return balance.Amount
}

// countRows returns the number of rows in table.
func countRows(t *testing.T, db *DB, table string) int {
	t.Helper()
	var n int
	require.NoError(t, db.conn.Get(&n, "SELECT COUNT(*) FROM "+table))
	return n
}

// transferLegs returns a transfer's two legs, source first.
func transferLegs(t *testing.T, db *DB, transferID ID) []Transaction {
	t.Helper()
	var legs []Transaction
	require.NoError(t, db.conn.Select(&legs, "SELECT * FROM transactions WHERE transfer_id = ? ORDER BY id", transferID))
	require.Len(t, legs, 2)
	return legs
}

// spendFrom writes a check for cents from accountID.
func spendFrom(t *testing.T, db *DB, accountID ID, cents int64) *Transaction {
	t.Helper()
	payee, err := db.CreateParty("Vendor", nil, nil, nil, nil)
	require.NoError(t, err)
	category, err := db.CreateCategory("Supplies")
	require.NoError(t, err)
	check, err := db.CreateWithdrawal(accountID, payee.ID, TransactionMethodCheck, "", transferDate, nil, []ExpenseItem{
		{CategoryID: category.ID, Amount: NewMoney(cents, CurrencyUSD)},
	})
	require.NoError(t, err)
	return check
}

// TestCreateTransfer_ParentToEarmark verifies that funding an earmark moves
// each account's own balance while the parent's combined balance stays put.
func TestCreateTransfer_ParentToEarmark(t *testing.T) {
	db := testDB(t)
	parent, earmark := setupTransferAccounts(t, db)

	transfer, err := db.CreateTransfer(parent.ID, earmark.ID, NewMoney(30000, CurrencyUSD), "Set aside for roof", transferDate)
	require.NoError(t, err)
	assert.Equal(t, parent.ID, transfer.FromAccountID)
	assert.Equal(t, earmark.ID, transfer.ToAccountID)
	assert.Equal(t, NewMoney(30000, CurrencyUSD), transfer.Amount)
	assert.Equal(t, "Set aside for roof", transfer.Memo)
	assert.True(t, transfer.TransferredAt.Equal(transferDate))

	assert.Equal(t, int64(70000), ownBalance(t, db, parent.ID))
	assert.Equal(t, int64(30000), ownBalance(t, db, earmark.ID))
	combined, err := db.AccountBalance(parent.ID, true)
	require.NoError(t, err)
	assert.Equal(t, int64(100000), combined.Amount, "no money left the bank")

	legs := transferLegs(t, db, transfer.ID)
	assert.Equal(t, parent.ID, legs[0].AccountID)
	assert.Equal(t, int64(-30000), legs[0].Amount)
	assert.Equal(t, earmark.ID, legs[1].AccountID)
	assert.Equal(t, int64(30000), legs[1].Amount)
	for _, leg := range legs {
		assert.Nil(t, leg.Method, "nothing happened at the bank, so a leg has no method")
		assert.Nil(t, leg.PayeeID)
		assert.Nil(t, leg.CheckNumber)
		assert.Equal(t, "Set aside for roof", leg.Memo)
		assert.True(t, leg.TransactedAt.Equal(transferDate))
	}

	loaded, err := db.Transfer(transfer.ID)
	require.NoError(t, err)
	assert.Equal(t, transfer, loaded)
}

// TestCreateTransfer_EarmarkToParent verifies that money can be released from
// an earmark back to its parent.
func TestCreateTransfer_EarmarkToParent(t *testing.T) {
	db := testDB(t)
	parent, earmark := setupTransferAccounts(t, db)

	_, err := db.CreateTransfer(parent.ID, earmark.ID, NewMoney(30000, CurrencyUSD), "", transferDate)
	require.NoError(t, err)
	_, err = db.CreateTransfer(earmark.ID, parent.ID, NewMoney(10000, CurrencyUSD), "Released", transferDate)
	require.NoError(t, err)

	assert.Equal(t, int64(80000), ownBalance(t, db, parent.ID))
	assert.Equal(t, int64(20000), ownBalance(t, db, earmark.ID))
}

// TestCreateTransfer_ExactBalance verifies that a transfer may empty the
// source account completely.
func TestCreateTransfer_ExactBalance(t *testing.T) {
	db := testDB(t)
	parent, earmark := setupTransferAccounts(t, db)

	_, err := db.CreateTransfer(parent.ID, earmark.ID, NewMoney(100000, CurrencyUSD), "", transferDate)
	require.NoError(t, err)
	assert.Zero(t, ownBalance(t, db, parent.ID))
}

// TestCreateTransfer_Validation covers the rejected inputs, and checks that a
// rejected transfer writes nothing.
func TestCreateTransfer_Validation(t *testing.T) {
	db := testDB(t)
	parent, earmark := setupTransferAccounts(t, db)

	sibling, err := db.CreateBankAccount("Youth Fund", AccountTypeEarmark, CurrencyUSD, &parent.ID, "", true, NewMoney(5000, CurrencyUSD), transferDate)
	require.NoError(t, err)
	otherRoot, err := db.CreateBankAccount("Savings", AccountTypeSavings, CurrencyUSD, nil, "", true, NewMoney(5000, CurrencyUSD), transferDate)
	require.NoError(t, err)
	cousin, err := db.CreateBankAccount("Reserve", AccountTypeEarmark, CurrencyUSD, &otherRoot.ID, "", true, Money{}, time.Time{})
	require.NoError(t, err)
	retired, err := db.CreateBankAccount("Retired Fund", AccountTypeEarmark, CurrencyUSD, &parent.ID, "", false, Money{}, time.Time{})
	require.NoError(t, err)

	transactionsBefore := countRows(t, db, "transactions")
	usd := func(cents int64) Money { return NewMoney(cents, CurrencyUSD) }

	cases := []struct {
		name     string
		from, to ID
		amount   Money
		date     time.Time
	}{
		{"SameAccount", parent.ID, parent.ID, usd(100), transferDate},
		{"Siblings", sibling.ID, earmark.ID, usd(100), transferDate},
		{"UnrelatedRoots", parent.ID, otherRoot.ID, usd(100), transferDate},
		{"EarmarkOfAnotherRoot", parent.ID, cousin.ID, usd(100), transferDate},
		{"ZeroAmount", parent.ID, earmark.ID, usd(0), transferDate},
		{"NegativeAmount", parent.ID, earmark.ID, usd(-100), transferDate},
		{"WrongCurrency", parent.ID, earmark.ID, NewMoney(100, CurrencyCAD), transferDate},
		{"ZeroDate", parent.ID, earmark.ID, usd(100), time.Time{}},
		{"InactiveDestination", parent.ID, retired.ID, usd(100), transferDate},
		{"MissingAccount", parent.ID, 9999, usd(100), transferDate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := db.CreateTransfer(tc.from, tc.to, tc.amount, "", tc.date)
			assert.Error(t, err)
		})
	}

	t.Run("InactiveSource", func(t *testing.T) {
		_, err := db.UpdateBankAccount(earmark.ID, BankAccountDelta{IsActive: new(false)})
		require.NoError(t, err)
		t.Cleanup(func() {
			_, err := db.UpdateBankAccount(earmark.ID, BankAccountDelta{IsActive: new(true)})
			require.NoError(t, err)
		})
		_, err = db.CreateTransfer(earmark.ID, parent.ID, usd(100), "", transferDate)
		assert.Error(t, err)
	})

	t.Run("Overdraft", func(t *testing.T) {
		_, err := db.CreateTransfer(parent.ID, earmark.ID, usd(100001), "", transferDate)
		require.ErrorIs(t, err, ErrInsufficientFunds)
		assert.Contains(t, err.Error(), "General Checking")
		assert.Contains(t, err.Error(), "$0.01")
	})

	t.Run("OverdraftCountsOnlyTheSourcesOwnBalance", func(t *testing.T) {
		// The parent's combined balance includes the sibling's $50.00, but
		// that money is already set aside and is not the parent's to move.
		_, err := db.CreateTransfer(parent.ID, earmark.ID, usd(100000+5000), "", transferDate)
		assert.ErrorIs(t, err, ErrInsufficientFunds)
	})

	t.Run("OverdraftFromEmptyEarmark", func(t *testing.T) {
		_, err := db.CreateTransfer(earmark.ID, parent.ID, usd(1), "", transferDate)
		assert.ErrorIs(t, err, ErrInsufficientFunds)
	})

	assert.Zero(t, countRows(t, db, "transfers"), "a rejected transfer must write no header")
	assert.Equal(t, transactionsBefore, countRows(t, db, "transactions"), "a rejected transfer must write no legs")
}

// TestUpdateTransfer verifies that an edit rewrites the header and both legs.
func TestUpdateTransfer(t *testing.T) {
	db := testDB(t)
	parent, earmark := setupTransferAccounts(t, db)

	transfer, err := db.CreateTransfer(parent.ID, earmark.ID, NewMoney(30000, CurrencyUSD), "original", transferDate)
	require.NoError(t, err)

	newDate := transferDate.AddDate(0, 1, 0)
	updated, err := db.UpdateTransfer(transfer.ID, NewMoney(45000, CurrencyUSD), "edited", newDate)
	require.NoError(t, err)
	assert.Equal(t, transfer.ID, updated.ID)
	assert.Equal(t, parent.ID, updated.FromAccountID, "accounts must not change")
	assert.Equal(t, earmark.ID, updated.ToAccountID, "accounts must not change")
	assert.Equal(t, NewMoney(45000, CurrencyUSD), updated.Amount)
	assert.Equal(t, "edited", updated.Memo)
	assert.True(t, updated.TransferredAt.Equal(newDate))

	legs := transferLegs(t, db, transfer.ID)
	assert.Equal(t, int64(-45000), legs[0].Amount)
	assert.Equal(t, int64(45000), legs[1].Amount)
	for _, leg := range legs {
		assert.Equal(t, "edited", leg.Memo)
		assert.True(t, leg.TransactedAt.Equal(newDate))
	}
	assert.Equal(t, int64(55000), ownBalance(t, db, parent.ID))
	assert.Equal(t, int64(45000), ownBalance(t, db, earmark.ID))

	loaded, err := db.Transfer(transfer.ID)
	require.NoError(t, err)
	assert.Equal(t, updated.Amount, loaded.Amount)
}

// TestUpdateTransfer_Overdraft verifies that an edit may not overdraw the
// source (by raising the amount) or the destination (by lowering it after the
// money was spent).
func TestUpdateTransfer_Overdraft(t *testing.T) {
	db := testDB(t)
	parent, earmark := setupTransferAccounts(t, db)

	transfer, err := db.CreateTransfer(parent.ID, earmark.ID, NewMoney(30000, CurrencyUSD), "", transferDate)
	require.NoError(t, err)

	t.Run("RaiseUpToTheSourcesBalance", func(t *testing.T) {
		// The parent holds $700.00 more, so the transfer can grow to $1,000.00.
		_, err := db.UpdateTransfer(transfer.ID, NewMoney(100000, CurrencyUSD), "", transferDate)
		require.NoError(t, err)
		_, err = db.UpdateTransfer(transfer.ID, NewMoney(30000, CurrencyUSD), "", transferDate)
		require.NoError(t, err)
	})

	t.Run("RaisePastTheSourcesBalance", func(t *testing.T) {
		_, err := db.UpdateTransfer(transfer.ID, NewMoney(100001, CurrencyUSD), "", transferDate)
		assert.ErrorIs(t, err, ErrInsufficientFunds)
	})

	t.Run("LowerBelowWhatTheDestinationSpent", func(t *testing.T) {
		spendFrom(t, db, earmark.ID, 20000)
		_, err := db.UpdateTransfer(transfer.ID, NewMoney(19999, CurrencyUSD), "", transferDate)
		require.ErrorIs(t, err, ErrInsufficientFunds)
		assert.Contains(t, err.Error(), "Building Fund")

		_, err = db.UpdateTransfer(transfer.ID, NewMoney(20000, CurrencyUSD), "", transferDate)
		assert.NoError(t, err, "lowering to exactly what was spent leaves the earmark at zero")
	})

	legs := transferLegs(t, db, transfer.ID)
	assert.Equal(t, int64(-20000), legs[0].Amount, "rejected edits must leave the legs alone")
}

// TestUpdateTransfer_Validation covers the rejected inputs.
func TestUpdateTransfer_Validation(t *testing.T) {
	db := testDB(t)
	parent, earmark := setupTransferAccounts(t, db)

	transfer, err := db.CreateTransfer(parent.ID, earmark.ID, NewMoney(30000, CurrencyUSD), "", transferDate)
	require.NoError(t, err)

	_, err = db.UpdateTransfer(transfer.ID, NewMoney(0, CurrencyUSD), "", transferDate)
	assert.Error(t, err)
	_, err = db.UpdateTransfer(transfer.ID, NewMoney(100, CurrencyCAD), "", transferDate)
	assert.Error(t, err)
	_, err = db.UpdateTransfer(transfer.ID, NewMoney(100, CurrencyUSD), "", time.Time{})
	assert.Error(t, err)
	_, err = db.UpdateTransfer(9999, NewMoney(100, CurrencyUSD), "", transferDate)
	assert.Error(t, err)
}

// TestDeleteTransfer verifies that a delete removes the header and both legs,
// and is allowed even when it leaves the destination overdrawn.
func TestDeleteTransfer(t *testing.T) {
	db := testDB(t)
	parent, earmark := setupTransferAccounts(t, db)

	transfer, err := db.CreateTransfer(parent.ID, earmark.ID, NewMoney(30000, CurrencyUSD), "", transferDate)
	require.NoError(t, err)
	spendFrom(t, db, earmark.ID, 20000)

	require.NoError(t, db.DeleteTransfer(transfer.ID))

	assert.Zero(t, countRows(t, db, "transfers"))
	var legs int
	require.NoError(t, db.conn.Get(&legs, "SELECT COUNT(*) FROM transactions WHERE transfer_id IS NOT NULL"))
	assert.Zero(t, legs)
	assert.Equal(t, int64(100000), ownBalance(t, db, parent.ID))
	assert.Equal(t, int64(-20000), ownBalance(t, db, earmark.ID), "deleting is how a mistake is undone, even when the money was spent")

	_, err = db.Transfer(transfer.ID)
	assert.Error(t, err)
	assert.Error(t, db.DeleteTransfer(transfer.ID), "deleting twice must report the transfer missing")
}

// TestTransferLegsRejectedByEditors verifies that the deposit, withdrawal, and
// receipt-assignment paths refuse a transfer leg, since changing one leg alone
// would unbalance the pair.
func TestTransferLegsRejectedByEditors(t *testing.T) {
	db := testDB(t)
	parent, earmark := setupTransferAccounts(t, db)

	transfer, err := db.CreateTransfer(parent.ID, earmark.ID, NewMoney(30000, CurrencyUSD), "", transferDate)
	require.NoError(t, err)
	legs := transferLegs(t, db, transfer.ID)
	out, in := legs[0], legs[1]

	payee, err := db.CreateParty("Vendor", nil, nil, nil, nil)
	require.NoError(t, err)
	category, err := db.CreateCategory("Supplies")
	require.NoError(t, err)
	item, err := db.CreateItem("Local Fund", true)
	require.NoError(t, err)
	receipt := createTestReceipt(t, db, payee.ID, item.ID, transferDate, NewMoney(500, CurrencyUSD))

	_, err = db.UpdateWithdrawal(out.ID, payee.ID, TransactionMethodCheck, "", transferDate, nil, []ExpenseItem{
		{CategoryID: category.ID, Amount: NewMoney(30000, CurrencyUSD)},
	})
	assert.ErrorContains(t, err, "transfer")
	assert.ErrorContains(t, db.DeleteWithdrawal(out.ID), "transfer")
	_, err = db.UpdateDeposit(in.ID, TransactionMethodInBranch, "", transferDate, []ID{receipt.ID})
	assert.ErrorContains(t, err, "transfer")
	assert.ErrorContains(t, db.DeleteDeposit(in.ID), "transfer")
	_, err = db.AssignReceiptToTransaction(receipt.ID, in.ID)
	assert.ErrorContains(t, err, "transfer")

	// Both legs survive untouched.
	transferLegs(t, db, transfer.ID)
	assert.Equal(t, int64(30000), ownBalance(t, db, earmark.ID))
}

// TestTransferLegsExcludedFromReconciliation verifies that transfer legs never
// show up as uncleared, cannot be cleared, and do not get in the way of
// balancing a statement.
func TestTransferLegsExcludedFromReconciliation(t *testing.T) {
	db := testDB(t)
	parent, earmark := setupTransferAccounts(t, db)

	_, err := db.CreateTransfer(parent.ID, earmark.ID, NewMoney(30000, CurrencyUSD), "", transferDate)
	require.NoError(t, err)

	statementDate := transferDate.AddDate(0, 0, 30)
	uncleared, err := db.UnclearedTransactions(parent.ID, statementDate)
	require.NoError(t, err)
	require.Len(t, uncleared, 1, "only the opening balance is a bank transaction")
	assert.Nil(t, uncleared[0].TransferID)

	reconciliation, err := db.StartReconciliation(parent.ID, statementDate, NewMoney(100000, CurrencyUSD))
	require.NoError(t, err)

	var leg Transaction
	require.NoError(t, db.conn.Get(&leg, "SELECT * FROM transactions WHERE transfer_id IS NOT NULL LIMIT 1"))
	assert.Error(t, db.ClearTransaction(reconciliation.ID, leg.ID))

	require.NoError(t, db.ClearTransaction(reconciliation.ID, uncleared[0].ID))
	_, err = db.CompleteReconciliation(reconciliation.ID)
	assert.NoError(t, err, "the bank statement balances without the transfer")
}

// TestOpeningBalanceIgnoresTransferLegs verifies that a transfer whose memo
// reads "Opening Balance" is not mistaken for an account's opening balance:
// both have a NULL method.
func TestOpeningBalanceIgnoresTransferLegs(t *testing.T) {
	db := testDB(t)
	parent, earmark := setupTransferAccounts(t, db)

	transfer, err := db.CreateTransfer(parent.ID, earmark.ID, NewMoney(30000, CurrencyUSD), openingBalanceMemo, transferDate)
	require.NoError(t, err)

	tx, err := db.OpeningBalanceTransaction(earmark.ID)
	require.NoError(t, err)
	assert.Nil(t, tx, "the earmark has no opening balance")
	assert.Error(t, db.UpdateOpeningBalanceDate(earmark.ID, transferDate.AddDate(0, 1, 0)))

	tx, err = db.OpeningBalanceTransaction(parent.ID)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.Nil(t, tx.TransferID, "the parent's real opening balance, not its transfer leg")

	newDate := transferDate.AddDate(0, -1, 0)
	require.NoError(t, db.UpdateOpeningBalanceDate(parent.ID, newDate))
	for _, leg := range transferLegs(t, db, transfer.ID) {
		assert.True(t, leg.TransactedAt.Equal(transferDate), "moving the opening balance must not move a transfer leg")
	}
}

// TestLedgerNamesTransferAccount verifies that each ledger row for a transfer
// leg names the account on the other side.
func TestLedgerNamesTransferAccount(t *testing.T) {
	db := testDB(t)
	parent, earmark := setupTransferAccounts(t, db)

	_, err := db.CreateTransfer(parent.ID, earmark.ID, NewMoney(30000, CurrencyUSD), "", transferDate)
	require.NoError(t, err)

	entries, err := db.AccountLedger(parent.ID, &LedgerOptions{IncludeSubaccounts: true})
	require.NoError(t, err)
	require.Len(t, entries, 3)
	assert.Nil(t, entries[0].TransferAccountName, "the opening balance is not a transfer")

	require.NotNil(t, entries[1].TransferAccountName)
	assert.Equal(t, "Building Fund", *entries[1].TransferAccountName, "the parent's leg names the earmark")
	require.NotNil(t, entries[2].TransferAccountName)
	assert.Equal(t, "General Checking", *entries[2].TransferAccountName, "the earmark's leg names the parent")
	assert.Zero(t, entries[2].ReceiptCount, "a receiving leg is not a deposit")
	assert.Equal(t, int64(100000), entries[2].RunningBalance.Amount)
}

// TestCreateEarmarkFundedFromParent verifies that a new earmark can be funded
// from money its parent already holds, in one step.
func TestCreateEarmarkFundedFromParent(t *testing.T) {
	db := testDB(t)
	parent, _ := setupTransferAccounts(t, db)

	earmark, err := db.CreateEarmarkFundedFromParent("Roof Fund", parent.ID, "New roof", NewMoney(25000, CurrencyUSD), transferDate)
	require.NoError(t, err)
	assert.Equal(t, AccountTypeEarmark, earmark.AccountType)
	assert.Equal(t, CurrencyUSD, earmark.Currency)
	require.NotNil(t, earmark.ParentID)
	assert.Equal(t, parent.ID, *earmark.ParentID)
	assert.True(t, earmark.IsActive)
	assert.Equal(t, "New roof", earmark.Description)

	assert.Equal(t, int64(75000), ownBalance(t, db, parent.ID))
	assert.Equal(t, int64(25000), ownBalance(t, db, earmark.ID))
	combined, err := db.AccountBalance(parent.ID, true)
	require.NoError(t, err)
	assert.Equal(t, int64(100000), combined.Amount, "funding from the parent adds no new money")

	var transfer transferRow
	require.NoError(t, db.conn.Get(&transfer, "SELECT "+transferColumns+" FROM transfers WHERE to_account_id = ?", earmark.ID))
	assert.Equal(t, parent.ID, transfer.FromAccountID)
	assert.Equal(t, earmarkFundingMemo, transfer.Memo)

	opening, err := db.OpeningBalanceTransaction(earmark.ID)
	require.NoError(t, err)
	assert.Nil(t, opening, "a funded earmark has no opening balance")
}

// TestCreateEarmarkFundedFromParent_Validation verifies that nothing is
// created when the account or its funding is rejected.
func TestCreateEarmarkFundedFromParent_Validation(t *testing.T) {
	db := testDB(t)
	parent, earmark := setupTransferAccounts(t, db)
	accountsBefore := countRows(t, db, "bank_accounts")

	t.Run("InsufficientFunds", func(t *testing.T) {
		_, err := db.CreateEarmarkFundedFromParent("Roof Fund", parent.ID, "", NewMoney(100001, CurrencyUSD), transferDate)
		assert.ErrorIs(t, err, ErrInsufficientFunds)
	})
	t.Run("ZeroFunding", func(t *testing.T) {
		_, err := db.CreateEarmarkFundedFromParent("Roof Fund", parent.ID, "", NewMoney(0, CurrencyUSD), transferDate)
		assert.Error(t, err)
	})
	t.Run("EmptyName", func(t *testing.T) {
		_, err := db.CreateEarmarkFundedFromParent("", parent.ID, "", NewMoney(100, CurrencyUSD), transferDate)
		assert.Error(t, err)
	})
	t.Run("EarmarkParent", func(t *testing.T) {
		_, err := db.CreateEarmarkFundedFromParent("Nested", earmark.ID, "", NewMoney(100, CurrencyUSD), transferDate)
		assert.Error(t, err)
	})

	assert.Equal(t, accountsBefore, countRows(t, db, "bank_accounts"), "a rejected earmark must not be created")
	assert.Zero(t, countRows(t, db, "transfers"))
}

// TestMigrate_V7ToV8AddsTransfers verifies that upgrading a v7 file leaves its
// existing transactions readable, with no transfer.
func TestMigrate_V7ToV8AddsTransfers(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "v7.bedrock")

	{
		raw := openRaw(t, dbPath)
		for v := 1; v <= 7; v++ {
			migration, err := loadMigration(v)
			require.NoError(t, err)
			_, err = raw.Exec(migration)
			require.NoError(t, err)
		}
		_, err := raw.Exec("PRAGMA user_version = 7")
		require.NoError(t, err)
		_, err = raw.Exec("INSERT INTO bank_accounts (name, account_type, currency) VALUES ('Checking', 'checking', 'USD')")
		require.NoError(t, err)
		_, err = raw.Exec("INSERT INTO transactions (account_id, amount, memo, transacted_at) VALUES (1, 5000, 'Opening Balance', ?)", transferDate)
		require.NoError(t, err)
		require.NoError(t, raw.Close())
	}

	db, err := Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	tx, err := db.Transaction(1)
	require.NoError(t, err)
	assert.Nil(t, tx.TransferID)
	assert.Zero(t, countRows(t, db, "transfers"))
}

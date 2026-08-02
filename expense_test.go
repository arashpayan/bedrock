package bedrock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// day is a convenience for the fixed dates these tests range over.
func day(year int, month time.Month, d int) time.Time {
	return time.Date(year, month, d, 12, 0, 0, 0, time.UTC)
}

func usd(cents int64) Money {
	return NewMoney(cents, CurrencyUSD)
}

func TestExpensesInRange(t *testing.T) {
	db := testDB(t)
	_, account, _, party, category := setupTestData(t, db)

	food, err := db.CreateCategory("Food")
	require.NoError(t, err)

	checkNo := "1042"
	inRange, err := db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "Feast supplies",
		day(2025, time.June, 10), &checkNo, []ExpenseItem{
			{CategoryID: category.ID, Description: new("Paper"), Amount: usd(2500)},
			{CategoryID: food.ID, Description: new("Refreshments"), Amount: usd(7500)},
		})
	require.NoError(t, err)

	// One withdrawal on each side of the range, which must not appear.
	_, err = db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "Before",
		day(2025, time.May, 31), nil, []ExpenseItem{{CategoryID: category.ID, Amount: usd(1000)}})
	require.NoError(t, err)
	_, err = db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "After",
		day(2025, time.July, 1), nil, []ExpenseItem{{CategoryID: category.ID, Amount: usd(1000)}})
	require.NoError(t, err)

	start := time.Date(2025, time.June, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, time.July, 1, 0, 0, 0, 0, time.UTC)
	expenses, err := db.ExpensesInRange(start, end)
	require.NoError(t, err)
	require.Len(t, expenses, 2, "only the June withdrawal's lines belong to the range")

	// Lines of one withdrawal keep insertion order.
	assert.Equal(t, "Office Supplies", expenses[0].CategoryName)
	assert.Equal(t, "Food", expenses[1].CategoryName)

	first := expenses[0]
	assert.Equal(t, inRange.ID, first.TransactionID)
	assert.Equal(t, usd(2500), first.Amount)
	assert.Equal(t, "Paper", *first.Description)
	assert.Equal(t, "Test Checking", first.AccountName)
	assert.Equal(t, "Test Contributor", first.PayeeName)
	assert.Equal(t, "Feast supplies", first.Memo)
	require.NotNil(t, first.CheckNumber)
	assert.Equal(t, "1042", *first.CheckNumber)
	assert.Equal(t, day(2025, time.June, 10).UTC(), first.TransactedAt.UTC())
}

func TestExpensesInRangeBoundaries(t *testing.T) {
	db := testDB(t)
	_, account, _, party, category := setupTestData(t, db)

	start := time.Date(2025, time.June, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, time.July, 1, 0, 0, 0, 0, time.UTC)

	// Exactly on each bound: the range is half-open, so start is in and end is out.
	_, err := db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "On start",
		start, nil, []ExpenseItem{{CategoryID: category.ID, Amount: usd(1100)}})
	require.NoError(t, err)
	_, err = db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "On end",
		end, nil, []ExpenseItem{{CategoryID: category.ID, Amount: usd(2200)}})
	require.NoError(t, err)

	expenses, err := db.ExpensesInRange(start, end)
	require.NoError(t, err)
	require.Len(t, expenses, 1)
	assert.Equal(t, usd(1100), expenses[0].Amount)
	assert.Equal(t, "On start", expenses[0].Memo)
}

func TestExpensesInRangeOrdersByDate(t *testing.T) {
	db := testDB(t)
	_, account, _, party, category := setupTestData(t, db)

	// Created out of chronological order to prove the query sorts by date.
	for _, d := range []int{20, 5, 12} {
		_, err := db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "",
			day(2025, time.June, d), nil, []ExpenseItem{{CategoryID: category.ID, Amount: usd(int64(d) * 100)}})
		require.NoError(t, err)
	}

	expenses, err := db.ExpensesInRange(
		time.Date(2025, time.June, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, time.July, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, expenses, 3)

	assert.Equal(t, usd(500), expenses[0].Amount)
	assert.Equal(t, usd(1200), expenses[1].Amount)
	assert.Equal(t, usd(2000), expenses[2].Amount)
}

func TestExpensesInRangeEmpty(t *testing.T) {
	db := testDB(t)
	_, account, _, party, category := setupTestData(t, db)

	_, err := db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "",
		day(2025, time.June, 10), nil, []ExpenseItem{{CategoryID: category.ID, Amount: usd(1000)}})
	require.NoError(t, err)

	expenses, err := db.ExpensesInRange(
		time.Date(2025, time.August, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, time.September, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Nil(t, expenses)

	totals, err := db.ExpensesByCategory(
		time.Date(2025, time.August, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, time.September, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Nil(t, totals)
}

// Deposits carry no payee and no expense lines, so they must never surface.
func TestExpensesInRangeIgnoresDeposits(t *testing.T) {
	db := testDB(t)
	_, account, _, _, _ := setupTestData(t, db)

	_, err := db.CreateDeposit(account.ID, usd(50000), TransactionMethodInBranch, "Sunday deposit", day(2025, time.June, 10))
	require.NoError(t, err)

	expenses, err := db.ExpensesInRange(
		time.Date(2025, time.June, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, time.July, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Empty(t, expenses)
}

// An in-kind receipt line may carry an expense category, but no money moved, so
// it is not an expense.
func TestExpensesInRangeExcludesInKindContributions(t *testing.T) {
	db := testDB(t)
	_, _, item, party, category := setupTestData(t, db)

	_, err := db.CreateReceiptWithItems(party.ID, day(2025, time.June, 10), "Donated food", true,
		[]ReceiptItemInput{{ItemID: item.ID, CategoryID: &category.ID, Description: "Feast food", Price: usd(9000)}})
	require.NoError(t, err)

	start := time.Date(2025, time.June, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, time.July, 1, 0, 0, 0, 0, time.UTC)

	expenses, err := db.ExpensesInRange(start, end)
	require.NoError(t, err)
	assert.Empty(t, expenses, "in-kind contributions are not cash expenses")

	totals, err := db.ExpensesByCategory(start, end)
	require.NoError(t, err)
	assert.Empty(t, totals)
}

func TestExpensesByCategory(t *testing.T) {
	db := testDB(t)
	_, account, _, party, supplies := setupTestData(t, db)

	food, err := db.CreateCategory("Food")
	require.NoError(t, err)
	travel, err := db.CreateCategory("Travel")
	require.NoError(t, err)

	// Two withdrawals, with Food spread across both.
	_, err = db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "", day(2025, time.June, 5), nil,
		[]ExpenseItem{
			{CategoryID: food.ID, Amount: usd(3000)},
			{CategoryID: supplies.ID, Amount: usd(1500)},
		})
	require.NoError(t, err)
	_, err = db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "", day(2025, time.June, 20), nil,
		[]ExpenseItem{
			{CategoryID: food.ID, Amount: usd(2500)},
			{CategoryID: travel.ID, Amount: usd(8000)},
		})
	require.NoError(t, err)

	// Outside the range, so excluded from the totals.
	_, err = db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "", day(2025, time.July, 5), nil,
		[]ExpenseItem{{CategoryID: food.ID, Amount: usd(9900)}})
	require.NoError(t, err)

	totals, err := db.ExpensesByCategory(
		time.Date(2025, time.June, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, time.July, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, totals, 3, "categories with no spending in the range are omitted")

	// Ordered by category name.
	assert.Equal(t, "Food", totals[0].CategoryName)
	assert.Equal(t, food.ID, totals[0].CategoryID)
	assert.Equal(t, usd(5500), totals[0].Total, "Food is summed across both withdrawals")

	assert.Equal(t, "Office Supplies", totals[1].CategoryName)
	assert.Equal(t, usd(1500), totals[1].Total)

	assert.Equal(t, "Travel", totals[2].CategoryName)
	assert.Equal(t, usd(8000), totals[2].Total)
}

func TestExpensesByCategorySplitsCurrencies(t *testing.T) {
	db := testDB(t)
	_, usdAccount, _, party, category := setupTestData(t, db)

	cadAccount, err := db.CreateBankAccount("Canadian Checking", AccountTypeChecking, CurrencyCAD, nil, "", true, Money{}, time.Time{})
	require.NoError(t, err)

	_, err = db.CreateWithdrawal(usdAccount.ID, party.ID, TransactionMethodCheck, "", day(2025, time.June, 5), nil,
		[]ExpenseItem{{CategoryID: category.ID, Amount: usd(4000)}})
	require.NoError(t, err)
	_, err = db.CreateWithdrawal(cadAccount.ID, party.ID, TransactionMethodCheck, "", day(2025, time.June, 6), nil,
		[]ExpenseItem{{CategoryID: category.ID, Amount: NewMoney(6000, CurrencyCAD)}})
	require.NoError(t, err)

	totals, err := db.ExpensesByCategory(
		time.Date(2025, time.June, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, time.July, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, totals, 2, "one category spent in two currencies yields one row per currency")

	assert.Equal(t, NewMoney(6000, CurrencyCAD), totals[0].Total)
	assert.Equal(t, usd(4000), totals[1].Total)
	assert.Equal(t, totals[0].CategoryID, totals[1].CategoryID)

	// Both currencies also survive intact as detail lines.
	expenses, err := db.ExpensesInRange(
		time.Date(2025, time.June, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, time.July, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, expenses, 2)
	assert.Equal(t, "Test Checking", expenses[0].AccountName)
	assert.Equal(t, "Canadian Checking", expenses[1].AccountName)
}

func TestExpensesInRangeReflectsEdits(t *testing.T) {
	db := testDB(t)
	_, account, _, party, category := setupTestData(t, db)

	food, err := db.CreateCategory("Food")
	require.NoError(t, err)

	withdrawal, err := db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "Original",
		day(2025, time.June, 10), nil, []ExpenseItem{{CategoryID: category.ID, Amount: usd(2000)}})
	require.NoError(t, err)

	start := time.Date(2025, time.June, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, time.July, 1, 0, 0, 0, 0, time.UTC)

	// Moving the check out of the range takes its expense lines with it.
	_, err = db.UpdateWithdrawal(withdrawal.ID, party.ID, TransactionMethodCheck, "Moved",
		day(2025, time.July, 10), nil, []ExpenseItem{{CategoryID: food.ID, Amount: usd(2000)}})
	require.NoError(t, err)

	expenses, err := db.ExpensesInRange(start, end)
	require.NoError(t, err)
	assert.Empty(t, expenses)

	expenses, err = db.ExpensesInRange(end, time.Date(2025, time.August, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, expenses, 1)
	assert.Equal(t, "Food", expenses[0].CategoryName, "the replaced expense set is what gets reported")
	assert.Equal(t, "Moved", expenses[0].Memo)
}

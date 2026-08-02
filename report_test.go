package bedrock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reportTestDB builds a database whose Assembly keeps time in tz, plus the
// account, fund, contributor, and expense category the report tests spend.
func reportTestDB(t *testing.T, tz *time.Location) (db *DB, account *BankAccount, item *Item, party *Party, category *Category) {
	t.Helper()

	db = testDB(t)
	_, err := db.createAssembly("Test Assembly", tz, CurrencyUSD)
	require.NoError(t, err)

	account, err = db.CreateBankAccount("Checking", AccountTypeChecking, CurrencyUSD, nil, "", true, Money{}, time.Time{})
	require.NoError(t, err)
	item, err = db.CreateItem("Local Fund", true)
	require.NoError(t, err)
	party, err = db.CreateParty("Test Contributor", nil, nil, nil, nil)
	require.NoError(t, err)
	category, err = db.CreateCategory("Office Supplies")
	require.NoError(t, err)

	return db, account, item, party, category
}

// contribute records a contribution of amount to fund, received on soldAt.
func contribute(t *testing.T, db *DB, party *Party, fund *Item, soldAt time.Time, amount Money) {
	t.Helper()

	_, err := db.CreateReceiptWithItems(party.ID, soldAt, "", false,
		[]ReceiptItemInput{{ItemID: fund.ID, Price: amount}})
	require.NoError(t, err)
}

// ilm182 is ‘Ilm 182 B.E., 15 October – 2 November 2025. Most tests report on
// it: it sits well inside fiscal year 2025 and leaves 11 Bahá'í months in the
// year, which makes the pacing arithmetic easy to check by hand.
func ilm182(t *testing.T) BadiPeriod {
	t.Helper()

	period, err := BadiPeriodForDate(date(2025, time.October, 20), time.UTC)
	require.NoError(t, err)
	require.Equal(t, 12, period.Month)
	return period
}

func TestMonthlyReportContributionsAndExpenses(t *testing.T) {
	db, account, item, party, category := reportTestDB(t, time.UTC)

	earmarked, err := db.CreateItem("National Fund", false)
	require.NoError(t, err)

	// Earlier in the fiscal year, so it counts toward the year but not the month.
	contribute(t, db, party, item, day(2025, time.June, 10), usd(600_000))
	// Inside the period.
	contribute(t, db, party, item, day(2025, time.October, 20), usd(400_000))
	// Inside the period but earmarked, so it counts toward neither.
	contribute(t, db, party, earmarked, day(2025, time.October, 21), usd(500_000))
	// After the period, so it counts toward neither.
	contribute(t, db, party, item, day(2025, time.November, 10), usd(999_900))

	_, err = db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "Supplies",
		day(2025, time.October, 20), nil, []ExpenseItem{
			{CategoryID: category.ID, Amount: usd(25_000)},
			{CategoryID: category.ID, Amount: usd(75_000)},
		})
	require.NoError(t, err)
	// Outside the period.
	_, err = db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "Later",
		day(2025, time.November, 10), nil, []ExpenseItem{{CategoryID: category.ID, Amount: usd(11_100)}})
	require.NoError(t, err)

	report, err := db.MonthlyReport(ilm182(t))
	require.NoError(t, err)

	assert.Equal(t, 2025, report.FiscalYear)
	assert.Equal(t, "‘Ilm 182 B.E.", report.Period.String())
	assert.False(t, report.StraddlesFiscalYear)

	assert.Equal(t, usd(400_000), report.PeriodContributions, "only the month's counted funds")
	assert.Equal(t, usd(1_000_000), report.RaisedToDate, "the fiscal year through the end of the month")

	require.Len(t, report.Expenses, 2)
	assert.Equal(t, usd(100_000), report.ExpensesTotal)
	assert.Empty(t, report.ExpensesInOtherCurrencies)
	require.Len(t, report.ExpensesByCategory, 1)
	assert.Equal(t, "Office Supplies", report.ExpensesByCategory[0].CategoryName)
	assert.Equal(t, usd(100_000), report.ExpensesByCategory[0].Total)
}

func TestMonthlyReportPacing(t *testing.T) {
	db, _, item, party, _ := reportTestDB(t, time.UTC)

	// $36,500 over the 365 days of fiscal year 2025 is exactly $100 a day.
	_, err := db.SetFundraisingGoal(2025, usd(3_650_000))
	require.NoError(t, err)

	contribute(t, db, party, item, day(2025, time.June, 10), usd(600_000))
	contribute(t, db, party, item, day(2025, time.October, 20), usd(400_000))

	report, err := db.MonthlyReport(ilm182(t))
	require.NoError(t, err)

	require.NotNil(t, report.Goal)
	assert.Equal(t, usd(3_650_000), *report.Goal)
	assert.Equal(t, usd(1_000_000), report.RaisedToDate)

	// 1 May through 3 November 2025 is 186 days, so $18,600 was due.
	assert.Equal(t, usd(1_860_000), report.ExpectedToDate)
	assert.Equal(t, usd(-860_000), report.Variance, "behind schedule reads as a negative variance")
	assert.Equal(t, usd(2_650_000), report.RemainingGoal)

	// ‘Ilm ends 2 November; 11 Bahá'í months begin between then and 1 May 2026.
	assert.Equal(t, 11, report.PeriodsRemaining)
	// $26,500 over 11 months is $2,409.0909…, rounded up so 11 months cover it.
	assert.Equal(t, usd(240_910), report.AdjustedPeriodGoal)
	assert.GreaterOrEqual(t, report.AdjustedPeriodGoal.Amount*11, report.RemainingGoal.Amount,
		"the adjusted goal must never undershoot the remaining goal")
}

func TestMonthlyReportWithoutGoal(t *testing.T) {
	db, _, item, party, _ := reportTestDB(t, time.UTC)

	contribute(t, db, party, item, day(2025, time.October, 20), usd(400_000))

	report, err := db.MonthlyReport(ilm182(t))
	require.NoError(t, err)

	assert.Nil(t, report.Goal)
	assert.Equal(t, usd(400_000), report.RaisedToDate, "contributions are still reported")
	assert.Equal(t, usd(0), report.ExpectedToDate)
	assert.Equal(t, usd(0), report.Variance)
	assert.Equal(t, usd(0), report.RemainingGoal)
	assert.Equal(t, usd(0), report.AdjustedPeriodGoal)
	assert.Equal(t, 11, report.PeriodsRemaining, "the calendar does not depend on a goal")
}

func TestMonthlyReportGoalMet(t *testing.T) {
	db, _, item, party, _ := reportTestDB(t, time.UTC)

	_, err := db.SetFundraisingGoal(2025, usd(100_000))
	require.NoError(t, err)
	contribute(t, db, party, item, day(2025, time.October, 20), usd(150_000))

	report, err := db.MonthlyReport(ilm182(t))
	require.NoError(t, err)

	assert.Equal(t, usd(0), report.RemainingGoal, "an exceeded goal clamps at zero")
	assert.Equal(t, usd(0), report.AdjustedPeriodGoal, "nothing left to raise")
	assert.Equal(t, usd(99_041), report.Variance, "ahead of schedule reads as a positive variance")
}

// The one month a year that crosses 1 May reports its own contributions whole,
// while the fiscal-year figures start at 1 May.
func TestMonthlyReportStraddlingFiscalYear(t *testing.T) {
	db, _, item, party, _ := reportTestDB(t, time.UTC)

	// Jamál 183 B.E. runs 28 April – 16 May 2026.
	period, err := BadiPeriodForDate(date(2026, time.May, 5), time.UTC)
	require.NoError(t, err)
	require.Equal(t, date(2026, time.April, 28), period.Start)

	contribute(t, db, party, item, day(2026, time.April, 29), usd(100_000))
	contribute(t, db, party, item, day(2026, time.May, 5), usd(200_000))

	report, err := db.MonthlyReport(period)
	require.NoError(t, err)

	assert.True(t, report.StraddlesFiscalYear)
	assert.Equal(t, 2026, report.FiscalYear, "the fiscal year holding the month's last day")
	assert.Equal(t, usd(300_000), report.PeriodContributions, "the whole Bahá'í month")
	assert.Equal(t, usd(200_000), report.RaisedToDate, "the fiscal year, which began on 1 May")
}

// A month wholly inside April belongs to the fiscal year that is ending, not
// the one about to start.
func TestMonthlyReportLastMonthOfFiscalYear(t *testing.T) {
	db, _, item, party, _ := reportTestDB(t, time.UTC)

	// Jalál 183 B.E. runs 9 – 27 April 2026.
	period, err := BadiPeriodForDate(date(2026, time.April, 15), time.UTC)
	require.NoError(t, err)
	require.Equal(t, date(2026, time.April, 9), period.Start)

	contribute(t, db, party, item, day(2025, time.December, 1), usd(500_000))

	report, err := db.MonthlyReport(period)
	require.NoError(t, err)

	assert.Equal(t, 2025, report.FiscalYear)
	assert.False(t, report.StraddlesFiscalYear)
	assert.Equal(t, usd(500_000), report.RaisedToDate)
	assert.Equal(t, 1, report.PeriodsRemaining, "only Jamál begins before 1 May")
}

func TestMonthlyReportSeparatesOtherCurrencies(t *testing.T) {
	db, account, _, party, category := reportTestDB(t, time.UTC)

	cad, err := db.CreateBankAccount("Canadian Checking", AccountTypeChecking, CurrencyCAD, nil, "", true, Money{}, time.Time{})
	require.NoError(t, err)

	_, err = db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "", day(2025, time.October, 20), nil,
		[]ExpenseItem{{CategoryID: category.ID, Amount: usd(40_000)}})
	require.NoError(t, err)
	_, err = db.CreateWithdrawal(cad.ID, party.ID, TransactionMethodCheck, "", day(2025, time.October, 21), nil,
		[]ExpenseItem{{CategoryID: category.ID, Amount: NewMoney(60_000, CurrencyCAD)}})
	require.NoError(t, err)

	report, err := db.MonthlyReport(ilm182(t))
	require.NoError(t, err)

	assert.Equal(t, usd(40_000), report.ExpensesTotal, "the grand total stays in the Assembly's currency")
	require.Len(t, report.ExpensesInOtherCurrencies, 1)
	assert.Equal(t, NewMoney(60_000, CurrencyCAD), report.ExpensesInOtherCurrencies[0])
	assert.Len(t, report.Expenses, 2, "both lines are still listed")
}

// A period built in one timezone must still name the same Bahá'í month once the
// report resolves it in the Assembly's.
func TestMonthlyReportNormalizesPeriodTimezone(t *testing.T) {
	eastern, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	db, _, item, party, _ := reportTestDB(t, eastern)
	contribute(t, db, party, item, day(2025, time.October, 20), usd(400_000))

	utcPeriod := ilm182(t)
	report, err := db.MonthlyReport(utcPeriod)
	require.NoError(t, err)

	assert.Equal(t, 12, report.Period.Month)
	assert.Equal(t, 182, report.Period.Year)
	assert.Equal(t, time.Date(2025, time.October, 15, 0, 0, 0, 0, eastern), report.Period.Start,
		"the period is re-anchored to the Assembly's wall clock")
	assert.Equal(t, time.Date(2025, time.November, 3, 0, 0, 0, 0, eastern), report.Period.End)
	assert.Equal(t, usd(400_000), report.PeriodContributions)
}

func TestMonthlyReportInvalidPeriod(t *testing.T) {
	db, _, _, _, _ := reportTestDB(t, time.UTC)

	_, err := db.MonthlyReport(BadiPeriod{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid Badí' period")
}

func TestCurrentAndMostRecentlyCompletedBadiPeriod(t *testing.T) {
	db, _, _, _, _ := reportTestDB(t, time.UTC)

	current, err := db.CurrentBadiPeriod()
	require.NoError(t, err)
	assert.False(t, current.Start.After(time.Now()))
	assert.True(t, current.End.After(time.Now()))

	completed, err := db.MostRecentlyCompletedBadiPeriod()
	require.NoError(t, err)
	assert.Equal(t, current.Start, completed.End, "the completed month ends where the current one begins")

	next, err := NextBadiPeriod(completed)
	require.NoError(t, err)
	assert.Equal(t, current, next)
}

func TestProrate(t *testing.T) {
	cases := []struct {
		name           string
		amount         int64
		elapsed, total int
		want           int64
	}{
		{"nothing elapsed", 3_650_000, 0, 365, 0},
		{"one day", 3_650_000, 1, 365, 10_000},
		{"half a year", 3_650_000, 182, 365, 1_820_000},
		{"whole year", 3_650_000, 365, 365, 3_650_000},
		{"past the end clamps", 3_650_000, 400, 365, 3_650_000},
		{"negative elapsed", 3_650_000, -5, 365, 0},
		{"zero total", 3_650_000, 10, 0, 0},
		{"rounds to nearest", 100, 1, 3, 33},
		{"rounds half away from zero", 100, 1, 8, 13},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, prorate(tc.amount, tc.elapsed, tc.total))
		})
	}
}

func TestDivideRoundingUp(t *testing.T) {
	cases := []struct {
		name   string
		amount int64
		n      int
		want   int64
	}{
		{"exact", 1000, 4, 250},
		{"rounds up", 1000, 3, 334},
		{"one period", 2_650_000, 1, 2_650_000},
		{"no periods left", 2_650_000, 0, 0},
		{"negative periods", 100, -2, 0},
		{"nothing to raise", 0, 11, 0},
		{"goal already met", -500, 11, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := divideRoundingUp(tc.amount, tc.n)
			assert.Equal(t, tc.want, got)
			if tc.n > 0 && tc.amount > 0 {
				assert.GreaterOrEqual(t, got*int64(tc.n), tc.amount, "n parts must cover the whole")
			}
		})
	}
}

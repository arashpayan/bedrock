package bedrock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFiscalYearForDate(t *testing.T) {
	cases := []struct {
		name string
		date time.Time
		want int
	}{
		{"may 1 starts new fiscal year", time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC), 2026},
		{"mid fiscal year (autumn)", time.Date(2026, time.October, 15, 12, 0, 0, 0, time.UTC), 2026},
		{"january belongs to prior year", time.Date(2027, time.January, 10, 12, 0, 0, 0, time.UTC), 2026},
		{"april 30 is last day of prior year", time.Date(2027, time.April, 30, 23, 59, 59, 0, time.UTC), 2026},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, FiscalYearForDate(tc.date, time.UTC))
		})
	}
}

func TestFiscalYearForDateRespectsTimezone(t *testing.T) {
	eastern, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	// May 1, 2026 00:30 UTC is still April 30, 2026 (20:30) in New York, so the
	// fiscal year differs depending on the location used.
	instant := time.Date(2026, time.May, 1, 0, 30, 0, 0, time.UTC)
	assert.Equal(t, 2026, FiscalYearForDate(instant, time.UTC))
	assert.Equal(t, 2025, FiscalYearForDate(instant, eastern))
}

func TestFiscalYearRange(t *testing.T) {
	start, end := FiscalYearRange(2026, time.UTC)
	assert.Equal(t, time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC), start)
	assert.Equal(t, time.Date(2027, time.May, 1, 0, 0, 0, 0, time.UTC), end)
}

func TestFundraisingGoalCRUD(t *testing.T) {
	db := testDB(t)
	_, _, _, _, _ = setupTestData(t, db)

	// No goal yet -> (nil, nil).
	goal, err := db.FundraisingGoal(2026)
	require.NoError(t, err)
	assert.Nil(t, goal)

	// Set a goal.
	created, err := db.SetFundraisingGoal(2026, NewMoney(5_000_00, CurrencyUSD))
	require.NoError(t, err)
	assert.Equal(t, 2026, created.FiscalYear)
	assert.Equal(t, int64(5_000_00), created.Amount.Amount)

	// Read it back.
	got, err := db.FundraisingGoal(2026)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int64(5_000_00), got.Amount.Amount)

	// Setting again for the same year overwrites rather than duplicating.
	updated, err := db.SetFundraisingGoal(2026, NewMoney(7_500_00, CurrencyUSD))
	require.NoError(t, err)
	assert.Equal(t, created.ID, updated.ID)
	assert.Equal(t, int64(7_500_00), updated.Amount.Amount)

	goals, err := db.FundraisingGoals()
	require.NoError(t, err)
	assert.Len(t, goals, 1)

	// Delete it (idempotent).
	require.NoError(t, db.DeleteFundraisingGoal(2026))
	require.NoError(t, db.DeleteFundraisingGoal(2026))
	gone, err := db.FundraisingGoal(2026)
	require.NoError(t, err)
	assert.Nil(t, gone)
}

func TestSetFundraisingGoalValidation(t *testing.T) {
	db := testDB(t)
	_, _, _, _, _ = setupTestData(t, db)

	_, err := db.SetFundraisingGoal(2026, NewMoney(0, CurrencyUSD))
	assert.Error(t, err, "zero goal should be rejected")

	_, err = db.SetFundraisingGoal(2026, NewMoney(-100, CurrencyUSD))
	assert.Error(t, err, "negative goal should be rejected")

	// Assembly default currency is USD; a CAD goal must be rejected.
	_, err = db.SetFundraisingGoal(2026, NewMoney(1_000_00, CurrencyCAD))
	assert.Error(t, err, "currency mismatch should be rejected")
}

func TestFundraisingProgress(t *testing.T) {
	db := testDB(t)
	_, _, localFund, party, _ := setupTestData(t, db)

	// An earmarked fund that must not count toward the goal.
	earmarked, err := db.CreateItem("Humanitarian Fund", false)
	require.NoError(t, err)

	inYear := time.Date(2026, time.June, 1, 12, 0, 0, 0, time.UTC)
	beforeYear := time.Date(2026, time.April, 15, 12, 0, 0, 0, time.UTC) // prior fiscal year
	afterYear := time.Date(2027, time.May, 2, 12, 0, 0, 0, time.UTC)     // next fiscal year

	// Counted contribution inside the fiscal year.
	_, err = db.CreateReceiptWithItems(party.ID, inYear, "", false, []ReceiptItemInput{
		{ItemID: localFund.ID, Price: NewMoney(1_000_00, CurrencyUSD)},
	})
	require.NoError(t, err)

	// Earmarked contribution inside the year -> excluded.
	_, err = db.CreateReceiptWithItems(party.ID, inYear, "", false, []ReceiptItemInput{
		{ItemID: earmarked.ID, Price: NewMoney(500_00, CurrencyUSD)},
	})
	require.NoError(t, err)

	// Counted fund but outside the fiscal year -> excluded (both sides).
	_, err = db.CreateReceiptWithItems(party.ID, beforeYear, "", false, []ReceiptItemInput{
		{ItemID: localFund.ID, Price: NewMoney(999_00, CurrencyUSD)},
	})
	require.NoError(t, err)
	_, err = db.CreateReceiptWithItems(party.ID, afterYear, "", false, []ReceiptItemInput{
		{ItemID: localFund.ID, Price: NewMoney(999_00, CurrencyUSD)},
	})
	require.NoError(t, err)

	// A second counted contribution inside the year, to verify summation.
	_, err = db.CreateReceiptWithItems(party.ID, inYear, "", false, []ReceiptItemInput{
		{ItemID: localFund.ID, Price: NewMoney(250_00, CurrencyUSD)},
	})
	require.NoError(t, err)

	// No goal set yet.
	progress, err := db.FundraisingProgress(2026)
	require.NoError(t, err)
	assert.Equal(t, int64(1_250_00), progress.Raised.Amount, "only counted, in-year contributions")
	assert.Nil(t, progress.Goal)
	assert.Equal(t, 0.0, progress.Percent())
	assert.Equal(t, int64(0), progress.Remaining().Amount)

	// Now set a goal and re-check derived figures.
	_, err = db.SetFundraisingGoal(2026, NewMoney(5_000_00, CurrencyUSD))
	require.NoError(t, err)

	progress, err = db.FundraisingProgress(2026)
	require.NoError(t, err)
	require.NotNil(t, progress.Goal)
	assert.Equal(t, int64(5_000_00), progress.Goal.Amount)
	assert.InDelta(t, 25.0, progress.Percent(), 0.001)
	assert.Equal(t, int64(3_750_00), progress.Remaining().Amount)
}

func TestFundraisingProgressExceededGoalClampsRemaining(t *testing.T) {
	db := testDB(t)
	_, _, localFund, party, _ := setupTestData(t, db)

	inYear := time.Date(2026, time.June, 1, 12, 0, 0, 0, time.UTC)
	_, err := db.CreateReceiptWithItems(party.ID, inYear, "", false, []ReceiptItemInput{
		{ItemID: localFund.ID, Price: NewMoney(6_000_00, CurrencyUSD)},
	})
	require.NoError(t, err)

	_, err = db.SetFundraisingGoal(2026, NewMoney(5_000_00, CurrencyUSD))
	require.NoError(t, err)

	progress, err := db.FundraisingProgress(2026)
	require.NoError(t, err)
	assert.Equal(t, int64(0), progress.Remaining().Amount, "remaining clamps at zero when exceeded")
	assert.Greater(t, progress.Percent(), 100.0)
}

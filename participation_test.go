package bedrock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// period resolves a Bahá'í month from a date, for tests that need a real one.
func period(t *testing.T, year int, month time.Month, d int) BadiPeriod {
	t.Helper()

	p, err := BadiPeriodForDate(date(year, month, d), time.UTC)
	require.NoError(t, err)
	return p
}

func sampleCounts() ParticipationCounts {
	return ParticipationCounts{
		AdultsContributed: 34,
		AdultsActive:      120,
		Youth:             6,
		JuniorYouth:       4,
		Children:          9,
	}
}

func TestSetAndGetParticipation(t *testing.T) {
	db := testDB(t)
	sharaf := period(t, 2026, time.January, 20)

	saved, err := db.SetParticipation(sharaf, sampleCounts())
	require.NoError(t, err)
	assert.Equal(t, sharaf.Year, saved.BadiYear)
	assert.Equal(t, sharaf.Month, saved.BadiMonth)
	assert.Equal(t, 34, saved.AdultsContributed)
	assert.Equal(t, 120, saved.AdultsActive)

	got, err := db.Participation(sharaf)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, saved.ID, got.ID)
	assert.Equal(t, 6, got.Youth)
	assert.Equal(t, 4, got.JuniorYouth)
	assert.Equal(t, 9, got.Children)
}

func TestParticipationUnrecorded(t *testing.T) {
	db := testDB(t)

	got, err := db.Participation(period(t, 2026, time.January, 20))
	require.NoError(t, err)
	assert.Nil(t, got, "an unrecorded month is not an error")
}

func TestSetParticipationReplaces(t *testing.T) {
	db := testDB(t)
	sharaf := period(t, 2026, time.January, 20)

	first, err := db.SetParticipation(sharaf, sampleCounts())
	require.NoError(t, err)

	updated := sampleCounts()
	updated.AdultsContributed = 40
	updated.Youth = 8
	second, err := db.SetParticipation(sharaf, updated)
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID, "re-entering a month updates its row")
	assert.Equal(t, 40, second.AdultsContributed)
	assert.Equal(t, 8, second.Youth)

	// And there is still only one row for the month.
	var count int
	require.NoError(t, db.conn.Get(&count, "SELECT COUNT(*) FROM participation"))
	assert.Equal(t, 1, count)
}

func TestSetParticipationValidation(t *testing.T) {
	db := testDB(t)
	sharaf := period(t, 2026, time.January, 20)

	cases := []struct {
		name   string
		counts ParticipationCounts
		errMsg string
	}{
		{
			"more contributors than active adults",
			ParticipationCounts{AdultsContributed: 121, AdultsActive: 120},
			"cannot exceed active adults",
		},
		{"negative adults", ParticipationCounts{AdultsContributed: -1, AdultsActive: 10}, "cannot be negative"},
		{"negative active adults", ParticipationCounts{AdultsActive: -5}, "cannot be negative"},
		{"negative youth", ParticipationCounts{Youth: -1}, "cannot be negative"},
		{"negative junior youth", ParticipationCounts{JuniorYouth: -1}, "cannot be negative"},
		{"negative children", ParticipationCounts{Children: -1}, "cannot be negative"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := db.SetParticipation(sharaf, tc.counts)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errMsg)
		})
	}
}

// Zero contributors out of zero active adults is a legitimate month.
func TestSetParticipationAcceptsZeroes(t *testing.T) {
	db := testDB(t)

	saved, err := db.SetParticipation(period(t, 2026, time.January, 20), ParticipationCounts{})
	require.NoError(t, err)
	assert.Equal(t, 0, saved.Contributors())
	assert.False(t, saved.HasAdultRate(), "no active adults means no rate to show")
	assert.Equal(t, 0.0, saved.AdultRate())
}

// Ayyám-i-Há is counted with Mulk, so it cannot hold a record of its own.
func TestParticipationRejectsAyyamiHa(t *testing.T) {
	db := testDB(t)
	ayyamiHa := period(t, 2026, time.February, 27)
	require.True(t, ayyamiHa.IsAyyamiHa())

	_, err := db.SetParticipation(ayyamiHa, sampleCounts())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "recorded with Mulk")

	_, err = db.Participation(ayyamiHa)
	assert.Error(t, err)

	assert.Error(t, db.DeleteParticipation(ayyamiHa))
}

func TestDeleteParticipation(t *testing.T) {
	db := testDB(t)
	sharaf := period(t, 2026, time.January, 20)

	_, err := db.SetParticipation(sharaf, sampleCounts())
	require.NoError(t, err)
	require.NoError(t, db.DeleteParticipation(sharaf))

	got, err := db.Participation(sharaf)
	require.NoError(t, err)
	assert.Nil(t, got)

	// Deleting a month that has no record is not an error.
	assert.NoError(t, db.DeleteParticipation(sharaf))
}

func TestAdultRate(t *testing.T) {
	cases := []struct {
		name        string
		contributed int
		active      int
		want        float64
		hasRate     bool
	}{
		{"typical", 34, 120, 28.333333333333332, true},
		{"everyone", 120, 120, 100, true},
		{"nobody", 0, 120, 0, true},
		{"no active adults recorded", 0, 0, 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := Participation{AdultsContributed: tc.contributed, AdultsActive: tc.active}
			assert.InDelta(t, tc.want, p.AdultRate(), 0.0001)
			assert.Equal(t, tc.hasRate, p.HasAdultRate())
		})
	}
}

func TestContributors(t *testing.T) {
	p := Participation{AdultsContributed: 34, Youth: 6, JuniorYouth: 4, Children: 9}
	assert.Equal(t, 53, p.Contributors())
}

func TestParticipationForPeriods(t *testing.T) {
	db := testDB(t)

	// Three consecutive months, with only the first and third recorded.
	first := period(t, 2025, time.October, 20)
	second, err := NextBadiPeriod(first)
	require.NoError(t, err)
	third, err := NextBadiPeriod(second)
	require.NoError(t, err)

	_, err = db.SetParticipation(first, ParticipationCounts{AdultsContributed: 10, AdultsActive: 100})
	require.NoError(t, err)
	_, err = db.SetParticipation(third, ParticipationCounts{AdultsContributed: 30, AdultsActive: 100})
	require.NoError(t, err)

	got, err := db.ParticipationForPeriods([]BadiPeriod{first, second, third})
	require.NoError(t, err)
	require.Len(t, got, 3, "the result is aligned with the input")

	require.NotNil(t, got[0])
	assert.Equal(t, 10, got[0].AdultsContributed)
	assert.Nil(t, got[1], "an unrecorded month comes back nil, not zero")
	require.NotNil(t, got[2])
	assert.Equal(t, 30, got[2].AdultsContributed)
}

// Ordering must come from the calendar, not from badi_month: Ayyám-i-Há is
// month 0 but falls between months 18 and 19, so a naive sort would misplace
// the tail of every year.
func TestParticipationForPeriodsSpansAyyamiHaAndYearEnd(t *testing.T) {
	db := testDB(t)

	mulk := period(t, 2026, time.February, 10)
	require.Equal(t, 18, mulk.Month)
	ayyamiHa, err := NextBadiPeriod(mulk)
	require.NoError(t, err)
	require.True(t, ayyamiHa.IsAyyamiHa())
	ala, err := NextBadiPeriod(ayyamiHa)
	require.NoError(t, err)
	require.Equal(t, 19, ala.Month)
	baha, err := NextBadiPeriod(ala)
	require.NoError(t, err)
	require.Equal(t, 1, baha.Month, "the next year opens with Bahá")

	_, err = db.SetParticipation(mulk, ParticipationCounts{AdultsContributed: 18, AdultsActive: 100})
	require.NoError(t, err)
	_, err = db.SetParticipation(ala, ParticipationCounts{AdultsContributed: 19, AdultsActive: 100})
	require.NoError(t, err)
	_, err = db.SetParticipation(baha, ParticipationCounts{AdultsContributed: 1, AdultsActive: 100})
	require.NoError(t, err)

	got, err := db.ParticipationForPeriods([]BadiPeriod{mulk, ayyamiHa, ala, baha})
	require.NoError(t, err)
	require.Len(t, got, 4)

	require.NotNil(t, got[0])
	assert.Equal(t, 18, got[0].AdultsContributed)
	assert.Nil(t, got[1], "Ayyám-i-Há never has its own record")
	require.NotNil(t, got[2])
	assert.Equal(t, 19, got[2].AdultsContributed, "‘Alá' follows Ayyám-i-Há, not month 1")
	require.NotNil(t, got[3])
	assert.Equal(t, 1, got[3].AdultsContributed, "the next Badí' year is a different key")
}

func TestParticipationForPeriodsEmpty(t *testing.T) {
	db := testDB(t)

	got, err := db.ParticipationForPeriods(nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestMonthlyReportParticipation(t *testing.T) {
	db, _, _, _, _ := reportTestDB(t, time.UTC)
	ilm := ilm182(t)

	_, err := db.SetParticipation(ilm, sampleCounts())
	require.NoError(t, err)

	report, err := db.MonthlyReport(ilm)
	require.NoError(t, err)

	require.NotNil(t, report.Participation)
	assert.Equal(t, 34, report.Participation.AdultsContributed)
	assert.InDelta(t, 28.33, report.Participation.AdultRate(), 0.01)
	assert.Empty(t, report.ParticipationBelongsTo)
}

func TestMonthlyReportWithoutParticipation(t *testing.T) {
	db, _, _, _, _ := reportTestDB(t, time.UTC)

	report, err := db.MonthlyReport(ilm182(t))
	require.NoError(t, err)
	assert.Nil(t, report.Participation)
}

// The fiscal year 2025 history runs from Jamál (the month holding 1 May) to the
// reported month, and never charts Ayyám-i-Há.
func TestMonthlyReportParticipationHistory(t *testing.T) {
	db, _, _, _, _ := reportTestDB(t, time.UTC)
	ilm := ilm182(t)

	_, err := db.SetParticipation(ilm, sampleCounts())
	require.NoError(t, err)

	report, err := db.MonthlyReport(ilm)
	require.NoError(t, err)
	require.NotEmpty(t, report.ParticipationHistory)

	first := report.ParticipationHistory[0]
	assert.Equal(t, 3, first.Period.Month, "the fiscal year opens mid-Jamál")
	assert.Equal(t, "Jamál", first.Period.Name)
	assert.Nil(t, first.Data)

	last := report.ParticipationHistory[len(report.ParticipationHistory)-1]
	assert.Equal(t, ilm.Month, last.Period.Month, "the history ends at the reported month")
	require.NotNil(t, last.Data)
	assert.Equal(t, 34, last.Data.AdultsContributed)

	// Jamál (3) through ‘Ilm (12) inclusive, with no Ayyám-i-Há among them.
	assert.Len(t, report.ParticipationHistory, 10)
	for _, point := range report.ParticipationHistory {
		assert.False(t, point.Period.IsAyyamiHa(), "Ayyám-i-Há is never charted")
	}
}

// Fiscal year 2022 opens on 1 May 2022, before the Naw-Rúz table starts. The
// report must still build, charting only the months the calendar can supply.
func TestMonthlyReportHistoryClampsToTheCalendarStart(t *testing.T) {
	db, _, _, _, _ := reportTestDB(t, time.UTC)

	first, err := BadiPeriodForDate(date(2023, time.March, 21), time.UTC)
	require.NoError(t, err)
	require.Equal(t, firstBadiYear, first.Year)

	report, err := db.MonthlyReport(first)
	require.NoError(t, err, "a month at the very start of the table must still report")

	require.Len(t, report.ParticipationHistory, 1)
	assert.Equal(t, first, report.ParticipationHistory[0].Period)
	assert.True(t, report.FiscalYearStart.Before(first.Start),
		"the fiscal year really does start before the calendar table")
}

// An Ayyám-i-Há report has no participation of its own and says where it lives.
func TestMonthlyReportAyyamiHaPointsAtMulk(t *testing.T) {
	db, _, _, _, _ := reportTestDB(t, time.UTC)

	ayyamiHa, err := BadiPeriodForDate(date(2026, time.February, 27), time.UTC)
	require.NoError(t, err)
	require.True(t, ayyamiHa.IsAyyamiHa())

	mulk, err := PreviousBadiPeriod(ayyamiHa)
	require.NoError(t, err)
	_, err = db.SetParticipation(mulk, sampleCounts())
	require.NoError(t, err)

	report, err := db.MonthlyReport(ayyamiHa)
	require.NoError(t, err)

	assert.Nil(t, report.Participation)
	assert.Equal(t, mulk.String(), report.ParticipationBelongsTo)

	// Mulk still appears in the history with its own record.
	var found bool
	for _, point := range report.ParticipationHistory {
		assert.False(t, point.Period.IsAyyamiHa())
		if point.Period.Month == 18 {
			found = true
			require.NotNil(t, point.Data)
			assert.Equal(t, 34, point.Data.AdultsContributed)
		}
	}
	assert.True(t, found, "Mulk should be charted")
}

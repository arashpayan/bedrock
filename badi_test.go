package bedrock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ayyamiHaTable is the Ayyám-i-Há column of the published table of Bahá'í dates
// 180–221 B.E. ("Guidelines for Local Spiritual Assemblies" §8.4.1.1),
// transcribed independently of nawRuzDay. badi.go derives these days from
// consecutive Naw-Rúz dates alone, so checking every row against the source
// verifies the whole Naw-Rúz transcription: a single mistyped Naw-Rúz shifts
// the derived Ayyám-i-Há of the year before it, the year itself, or both.
var ayyamiHaTable = []struct {
	year  int       // Badí' year
	start time.Time // first day of Ayyám-i-Há
	days  int
}{
	{180, date(2024, time.February, 26), 4},
	{181, date(2025, time.February, 25), 4},
	{182, date(2026, time.February, 25), 5},
	{183, date(2027, time.February, 26), 4},
	{184, date(2028, time.February, 26), 4},
	{185, date(2029, time.February, 25), 4},
	{186, date(2030, time.February, 25), 4},
	{187, date(2031, time.February, 25), 5},
	{188, date(2032, time.February, 26), 4},
	{189, date(2033, time.February, 25), 4},
	{190, date(2034, time.February, 25), 4},
	{191, date(2035, time.February, 25), 5},
	{192, date(2036, time.February, 26), 4},
	{193, date(2037, time.February, 25), 4},
	{194, date(2038, time.February, 25), 4},
	{195, date(2039, time.February, 25), 5},
	{196, date(2040, time.February, 26), 4},
	{197, date(2041, time.February, 25), 4},
	{198, date(2042, time.February, 25), 4},
	{199, date(2043, time.February, 25), 5},
	{200, date(2044, time.February, 26), 4},
	{201, date(2045, time.February, 25), 4},
	{202, date(2046, time.February, 25), 4},
	{203, date(2047, time.February, 25), 5},
	{204, date(2048, time.February, 26), 4},
	{205, date(2049, time.February, 25), 4},
	{206, date(2050, time.February, 25), 4},
	{207, date(2051, time.February, 25), 5},
	{208, date(2052, time.February, 26), 4},
	{209, date(2053, time.February, 25), 4},
	{210, date(2054, time.February, 25), 4},
	{211, date(2055, time.February, 25), 5},
	{212, date(2056, time.February, 26), 4},
	{213, date(2057, time.February, 25), 4},
	{214, date(2058, time.February, 25), 4},
	{215, date(2059, time.February, 25), 4},
	{216, date(2060, time.February, 25), 5},
	{217, date(2061, time.February, 25), 4},
	{218, date(2062, time.February, 25), 4},
	{219, date(2063, time.February, 25), 4},
	{220, date(2064, time.February, 25), 5},
	{221, date(2065, time.February, 25), 4},
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestAyyamiHaMatchesPublishedTable(t *testing.T) {
	for _, want := range ayyamiHaTable {
		period, err := BadiPeriodForDate(want.start, time.UTC)
		require.NoError(t, err, "%d B.E.", want.year)

		assert.True(t, period.IsAyyamiHa(), "%d B.E.: %s is not Ayyám-i-Há", want.year, want.start.Format("2 Jan 2006"))
		assert.Equal(t, want.year, period.Year, "%d B.E.: wrong year", want.year)
		assert.Equal(t, want.start, period.Start, "%d B.E.: wrong first day", want.year)
		assert.Equal(t, want.days, period.Days(), "%d B.E.: wrong number of days", want.year)
		assert.Equal(t, "Ayyám-i-Há", period.Name, "%d B.E.", want.year)

		// The day before Ayyám-i-Há closes Mulk, and the day after opens ‘Alá'.
		before, err := BadiPeriodForDate(want.start.AddDate(0, 0, -1), time.UTC)
		require.NoError(t, err)
		assert.Equal(t, 18, before.Month, "%d B.E.: Ayyám-i-Há must follow Mulk", want.year)

		after, err := BadiPeriodForDate(want.start.AddDate(0, 0, want.days), time.UTC)
		require.NoError(t, err)
		assert.Equal(t, 19, after.Month, "%d B.E.: ‘Alá' must follow Ayyám-i-Há", want.year)
	}
}

// TestYearStructure walks every supported year and checks that the months tile
// it exactly: 19 days each, starting at Naw-Rúz, with ‘Alá' ending the day
// before the next Naw-Rúz.
func TestYearStructure(t *testing.T) {
	for year := firstBadiYear; year <= firstBadiYear+len(nawRuzDay)-2; year++ {
		yearStart, err := nawRuz(year)
		require.NoError(t, err)
		nextYearStart, err := nawRuz(year + 1)
		require.NoError(t, err)

		for month := 1; month <= 18; month++ {
			want := yearStart.AddDate(0, 0, (month-1)*badiMonthDays)
			period, err := BadiPeriodForDate(want, time.UTC)
			require.NoError(t, err, "%d B.E. month %d", year, month)

			assert.Equal(t, year, period.Year)
			assert.Equal(t, month, period.Month, "%d B.E.: %s should open month %d", year, want.Format("2 Jan 2006"), month)
			assert.Equal(t, want, period.Start)
			assert.Equal(t, badiMonthDays, period.Days())

			// The last day of the month still resolves to the same period.
			last, err := BadiPeriodForDate(want.AddDate(0, 0, badiMonthDays-1), time.UTC)
			require.NoError(t, err)
			assert.Equal(t, period, last, "%d B.E. month %d: last day fell outside the month", year, month)
		}

		// ‘Alá' is the final 19 days, ending the day before the next Naw-Rúz.
		ala, err := BadiPeriodForDate(nextYearStart.AddDate(0, 0, -1), time.UTC)
		require.NoError(t, err)
		assert.Equal(t, year, ala.Year, "%d B.E.: year ends too early", year)
		assert.Equal(t, 19, ala.Month, "%d B.E.: last day of year is not in ‘Alá'", year)
		assert.Equal(t, badiMonthDays, ala.Days())
		assert.Equal(t, nextYearStart, ala.End, "%d B.E.: ‘Alá' must end at the next Naw-Rúz", year)
	}
}

func TestBadiPeriodForDate(t *testing.T) {
	cases := []struct {
		name  string
		date  time.Time
		year  int
		month int
		label string
		start time.Time
		days  int
	}{
		{"naw-rúz opens Bahá", date(2025, time.March, 20), 182, 1, "Bahá", date(2025, time.March, 20), 19},
		{"day before naw-rúz closes the prior year", date(2025, time.March, 19), 181, 19, "‘Alá’", date(2025, time.March, 1), 19},
		{"mid-year month", date(2025, time.October, 20), 182, 12, "‘Ilm", date(2025, time.October, 15), 19},
		{"first day of a month", date(2025, time.April, 27), 182, 3, "Jamál", date(2025, time.April, 27), 19},
		{"last day of a month", date(2025, time.May, 15), 182, 3, "Jamál", date(2025, time.April, 27), 19},
		{"five-day ayyám-i-há", date(2026, time.February, 27), 182, BadiMonthAyyamiHa, "Ayyám-i-Há", date(2026, time.February, 25), 5},
		{"four-day ayyám-i-há", date(2027, time.February, 28), 183, BadiMonthAyyamiHa, "Ayyám-i-Há", date(2027, time.February, 26), 4},
		{"first supported day", date(2023, time.March, 21), 180, 1, "Bahá", date(2023, time.March, 21), 19},
		{"last supported day", date(2065, time.March, 19), 221, 19, "‘Alá’", date(2065, time.March, 1), 19},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			period, err := BadiPeriodForDate(tc.date, time.UTC)
			require.NoError(t, err)
			assert.Equal(t, tc.year, period.Year)
			assert.Equal(t, tc.month, period.Month)
			assert.Equal(t, tc.label, period.Name)
			assert.Equal(t, tc.start, period.Start)
			assert.Equal(t, tc.days, period.Days())
			assert.Equal(t, tc.start.AddDate(0, 0, tc.days), period.End)
		})
	}
}

func TestBadiPeriodForDateOutOfRange(t *testing.T) {
	cases := []struct {
		name string
		date time.Time
	}{
		{"day before the table starts", date(2023, time.March, 20)},
		{"well before the table", date(2019, time.July, 4)},
		{"day after the table ends", date(2065, time.March, 20)},
		{"well after the table", date(2100, time.January, 1)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BadiPeriodForDate(tc.date, time.UTC)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrBadiDateOutOfRange)
		})
	}
}

func TestBadiPeriodUsesLocationWallClock(t *testing.T) {
	eastern, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	// ‘Ilm 182 runs 15 Oct – 2 Nov 2025 and straddles the end of daylight
	// saving time on 2 November, so its bounds span a UTC offset change.
	period, err := BadiPeriodForDate(time.Date(2025, time.October, 20, 9, 30, 0, 0, eastern), eastern)
	require.NoError(t, err)

	assert.Equal(t, 12, period.Month)
	assert.Equal(t, time.Date(2025, time.October, 15, 0, 0, 0, 0, eastern), period.Start)
	assert.Equal(t, time.Date(2025, time.November, 3, 0, 0, 0, 0, eastern), period.End)
	assert.Equal(t, 19, period.Days(), "a DST transition must not change the day count")

	// An instant late on the final day still belongs to the period, and the
	// first instant of the next day does not.
	last, err := BadiPeriodForDate(time.Date(2025, time.November, 2, 23, 59, 59, 0, eastern), eastern)
	require.NoError(t, err)
	assert.Equal(t, period, last)

	next, err := BadiPeriodForDate(time.Date(2025, time.November, 3, 0, 0, 0, 0, eastern), eastern)
	require.NoError(t, err)
	assert.Equal(t, 13, next.Month)
}

func TestBadiPeriodForDateRespectsTimezone(t *testing.T) {
	eastern, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	// 03:00 UTC on 15 October is still 14 October in New York, which is the
	// last day of Mashíyyat rather than the first of ‘Ilm.
	instant := time.Date(2025, time.October, 15, 3, 0, 0, 0, time.UTC)

	utcPeriod, err := BadiPeriodForDate(instant, time.UTC)
	require.NoError(t, err)
	assert.Equal(t, 12, utcPeriod.Month)

	easternPeriod, err := BadiPeriodForDate(instant, eastern)
	require.NoError(t, err)
	assert.Equal(t, 11, easternPeriod.Month)
}

func TestNextAndPreviousBadiPeriod(t *testing.T) {
	// Walk a full year, including Ayyám-i-Há and the year boundary.
	period, err := BadiPeriodForDate(date(2025, time.March, 20), time.UTC)
	require.NoError(t, err)

	seen := []BadiPeriod{period}
	for range 20 {
		next, err := NextBadiPeriod(period)
		require.NoError(t, err)
		assert.Equal(t, period.End, next.Start, "periods must be contiguous")
		period = next
		seen = append(seen, period)
	}

	// 19 months plus Ayyám-i-Há brings us to Bahá of the following year.
	assert.Equal(t, 182, seen[19].Year)
	assert.Equal(t, 19, seen[19].Month, "the twentieth period of a year is ‘Alá'")
	assert.Equal(t, 183, seen[20].Year)
	assert.Equal(t, 1, seen[20].Month)
	assert.Equal(t, date(2026, time.March, 21), seen[20].Start)

	// Ayyám-i-Há sits between Mulk and ‘Alá'.
	assert.Equal(t, 18, seen[17].Month)
	assert.True(t, seen[18].IsAyyamiHa())

	// Walking back retraces the same path.
	for i := len(seen) - 1; i > 0; i-- {
		prev, err := PreviousBadiPeriod(seen[i])
		require.NoError(t, err)
		assert.Equal(t, seen[i-1], prev, "step %d back", i)
	}
}

func TestNextBadiPeriodPastTableEnd(t *testing.T) {
	last, err := BadiPeriodForDate(date(2065, time.March, 19), time.UTC)
	require.NoError(t, err)

	_, err = NextBadiPeriod(last)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBadiDateOutOfRange)
}

func TestPreviousBadiPeriodBeforeTableStart(t *testing.T) {
	first, err := BadiPeriodForDate(date(2023, time.March, 21), time.UTC)
	require.NoError(t, err)

	_, err = PreviousBadiPeriod(first)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBadiDateOutOfRange)
}

func TestBadiPeriodsInRange(t *testing.T) {
	// Fiscal year 2025 runs 1 May 2025 through 30 April 2026.
	start, end := FiscalYearRange(2025, time.UTC)
	periods, err := BadiPeriodsInRange(start, end, time.UTC)
	require.NoError(t, err)
	require.NotEmpty(t, periods)

	// The range opens mid-Jamál and closes mid-Jamál of the following year;
	// periods are returned whole, so both overhang the fiscal year.
	assert.Equal(t, date(2025, time.April, 27), periods[0].Start)
	assert.Equal(t, 3, periods[0].Month)
	assert.True(t, periods[0].Start.Before(start), "first period should overhang the start")

	last := periods[len(periods)-1]
	assert.Equal(t, date(2026, time.April, 28), last.Start)
	assert.True(t, last.Start.Before(end), "last period must begin within the range")
	assert.True(t, last.End.After(end), "last period should overhang the end")
	assert.Len(t, periods, 21)

	for i := 1; i < len(periods); i++ {
		assert.Equal(t, periods[i-1].End, periods[i].Start, "gap before period %d", i)
	}
}

func TestBadiPeriodsInRangeEmpty(t *testing.T) {
	day := date(2025, time.June, 1)

	periods, err := BadiPeriodsInRange(day, day, time.UTC)
	require.NoError(t, err)
	assert.Nil(t, periods, "an empty range yields no periods")

	periods, err = BadiPeriodsInRange(day, day.AddDate(0, 0, -1), time.UTC)
	require.NoError(t, err)
	assert.Nil(t, periods, "a reversed range yields no periods")
}

func TestBadiPeriodsInRangeSingleDay(t *testing.T) {
	day := date(2025, time.June, 1)

	periods, err := BadiPeriodsInRange(day, day.AddDate(0, 0, 1), time.UTC)
	require.NoError(t, err)
	require.Len(t, periods, 1)
	assert.Equal(t, 4, periods[0].Month, "1 June 2025 falls in ‘Aẓamat (16 May – 3 June)")
}

func TestBadiPeriodString(t *testing.T) {
	period, err := BadiPeriodForDate(date(2025, time.October, 20), time.UTC)
	require.NoError(t, err)
	assert.Equal(t, "‘Ilm 182 B.E.", period.String())

	ayyamiHa, err := BadiPeriodForDate(date(2026, time.February, 27), time.UTC)
	require.NoError(t, err)
	assert.Equal(t, "Ayyám-i-Há 182 B.E.", ayyamiHa.String())
}

func TestNilLocationDefaultsToUTC(t *testing.T) {
	period, err := BadiPeriodForDate(date(2025, time.October, 20), nil)
	require.NoError(t, err)
	assert.Equal(t, time.UTC, period.Start.Location())
	assert.Equal(t, 12, period.Month)
}

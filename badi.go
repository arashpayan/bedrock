package bedrock

import (
	"errors"
	"fmt"
	"time"
)

// This file implements the Badí' (Bahá'í) calendar to the extent the Assembly's
// reporting needs it: locating the Bahá'í month that contains a given date, and
// enumerating the months that overlap a range.
//
// A Badí' year begins at Naw-Rúz and holds 19 months of 19 days. The
// intercalary days of Ayyám-i-Há fall between the eighteenth month (Mulk) and
// the nineteenth (‘Alá'), absorbing whatever days remain so that the following
// year again begins at Naw-Rúz; there are four of them in most years and five
// otherwise.
//
// Since 172 B.E. the date of Naw-Rúz has been fixed astronomically (by the
// vernal equinox at Tehran) rather than arithmetically, so it cannot be
// computed from a formula. The Gregorian dates in nawRuzDay are therefore
// transcribed from the published table of Bahá'í dates 180–221 B.E. in
// "Guidelines for Local Spiritual Assemblies" §8.4.1.1. Everything else —
// month boundaries and the length of Ayyám-i-Há — is derived from consecutive
// Naw-Rúz dates, and badi_test.go checks those derivations against the
// Ayyám-i-Há column of the same table.
//
// Bahá'í days begin at sunset. For bookkeeping purposes a period here starts at
// the first instant of its Gregorian start date in the Assembly's timezone,
// which keeps report ranges aligned with the transaction dates the treasurer
// enters.

// ErrBadiDateOutOfRange reports a date outside the published Naw-Rúz table.
// The supported span runs from Naw-Rúz 180 B.E. (21 March 2023) through the end
// of 221 B.E. (19 March 2065).
var ErrBadiDateOutOfRange = errors.New("date is outside the supported Badí' calendar range")

const (
	// BadiMonthAyyamiHa is the BadiPeriod.Month value for Ayyám-i-Há. The named
	// months use their ordinal, 1 (Bahá) through 19 (‘Alá').
	BadiMonthAyyamiHa = 0

	badiMonthDays   = 19 // days in each named month
	badiNamedMonths = 19 // named months per year, excluding Ayyám-i-Há

	// firstBadiYear is the first Badí' year in nawRuzDay, and firstNawRuzYear
	// the Gregorian year its Naw-Rúz falls in.
	firstBadiYear   = 180
	firstNawRuzYear = 2023
)

// badiMonthNames holds the nineteen month names in ordinal order.
var badiMonthNames = [badiNamedMonths]string{
	"Bahá", "Jalál", "Jamál", "‘Aẓamat", "Núr",
	"Raḥmat", "Kalimát", "Kamál", "Asmá’", "‘Izzat",
	"Mashíyyat", "‘Ilm", "Qudrat", "Qawl", "Masá’il",
	"Sharaf", "Sulṭán", "Mulk", "‘Alá’",
}

// nawRuzDay is the day of March on which Naw-Rúz falls, for consecutive Badí'
// years beginning at firstBadiYear. The Gregorian year of entry i is
// firstNawRuzYear+i.
//
// The final entry (222 B.E.) is not in the published table; it is derived from
// that table's Ayyám-i-Há column for 221 B.E. (25–28 February 2065, so ‘Alá'
// runs 1–19 March 2065) and exists only to close out 221. Dates on or after it
// are out of range.
var nawRuzDay = [...]int{
	21, // 180 B.E. — 21 Mar 2023
	20, // 181 B.E. — 20 Mar 2024
	20, // 182 B.E. — 20 Mar 2025
	21, // 183 B.E. — 21 Mar 2026
	21, // 184 B.E. — 21 Mar 2027
	20, // 185 B.E. — 20 Mar 2028
	20, // 186 B.E. — 20 Mar 2029
	20, // 187 B.E. — 20 Mar 2030
	21, // 188 B.E. — 21 Mar 2031
	20, // 189 B.E. — 20 Mar 2032
	20, // 190 B.E. — 20 Mar 2033
	20, // 191 B.E. — 20 Mar 2034
	21, // 192 B.E. — 21 Mar 2035
	20, // 193 B.E. — 20 Mar 2036
	20, // 194 B.E. — 20 Mar 2037
	20, // 195 B.E. — 20 Mar 2038
	21, // 196 B.E. — 21 Mar 2039
	20, // 197 B.E. — 20 Mar 2040
	20, // 198 B.E. — 20 Mar 2041
	20, // 199 B.E. — 20 Mar 2042
	21, // 200 B.E. — 21 Mar 2043
	20, // 201 B.E. — 20 Mar 2044
	20, // 202 B.E. — 20 Mar 2045
	20, // 203 B.E. — 20 Mar 2046
	21, // 204 B.E. — 21 Mar 2047
	20, // 205 B.E. — 20 Mar 2048
	20, // 206 B.E. — 20 Mar 2049
	20, // 207 B.E. — 20 Mar 2050
	21, // 208 B.E. — 21 Mar 2051
	20, // 209 B.E. — 20 Mar 2052
	20, // 210 B.E. — 20 Mar 2053
	20, // 211 B.E. — 20 Mar 2054
	21, // 212 B.E. — 21 Mar 2055
	20, // 213 B.E. — 20 Mar 2056
	20, // 214 B.E. — 20 Mar 2057
	20, // 215 B.E. — 20 Mar 2058
	20, // 216 B.E. — 20 Mar 2059
	20, // 217 B.E. — 20 Mar 2060
	20, // 218 B.E. — 20 Mar 2061
	20, // 219 B.E. — 20 Mar 2062
	20, // 220 B.E. — 20 Mar 2063
	20, // 221 B.E. — 20 Mar 2064
	20, // 222 B.E. — 20 Mar 2065 (derived; see above)
}

// BadiPeriod is one month of the Badí' calendar, or the run of Ayyám-i-Há days
// that precedes ‘Alá'. Start and End bound it as a half-open range in the
// timezone the period was resolved in.
type BadiPeriod struct {
	Year  int       // Badí' year, e.g. 182
	Month int       // 1–19, or BadiMonthAyyamiHa for the intercalary days
	Name  string    // "Sharaf", "Ayyám-i-Há"
	Start time.Time // first instant of the period
	End   time.Time // first instant of the following period (exclusive)
}

// BadiPeriodForDate returns the Badí' month containing t, evaluated in loc. A
// nil loc is treated as UTC.
func BadiPeriodForDate(t time.Time, loc *time.Location) (BadiPeriod, error) {
	if loc == nil {
		loc = time.UTC
	}
	local := t.In(loc)
	return badiPeriodForCivil(civilDate(local.Year(), local.Month(), local.Day()), loc)
}

// BadiPeriodsInRange returns every Badí' month overlapping the half-open range
// [start, end), in chronological order. Periods are returned whole: the first
// and last may extend beyond the range. It returns nil when end is not after
// start.
func BadiPeriodsInRange(start, end time.Time, loc *time.Location) ([]BadiPeriod, error) {
	if !end.After(start) {
		return nil, nil
	}
	if loc == nil {
		loc = time.UTC
	}

	period, err := BadiPeriodForDate(start, loc)
	if err != nil {
		return nil, err
	}

	var periods []BadiPeriod
	for period.Start.Before(end) {
		periods = append(periods, period)
		next, err := NextBadiPeriod(period)
		if err != nil {
			return nil, err
		}
		period = next
	}
	return periods, nil
}

// NextBadiPeriod returns the period immediately following p, in p's timezone.
func NextBadiPeriod(p BadiPeriod) (BadiPeriod, error) {
	loc := p.Start.Location()
	end := p.End.In(loc)
	return badiPeriodForCivil(civilDate(end.Year(), end.Month(), end.Day()), loc)
}

// PreviousBadiPeriod returns the period immediately preceding p, in p's
// timezone.
func PreviousBadiPeriod(p BadiPeriod) (BadiPeriod, error) {
	start := p.Start
	dayBefore := civilDate(start.Year(), start.Month(), start.Day()).AddDate(0, 0, -1)
	return badiPeriodForCivil(dayBefore, start.Location())
}

// Days returns the number of days in the period: 19 for a named month, 4 or 5
// for Ayyám-i-Há.
func (p BadiPeriod) Days() int {
	from := civilDate(p.Start.Year(), p.Start.Month(), p.Start.Day())
	end := p.End.In(p.Start.Location())
	return daysBetween(from, civilDate(end.Year(), end.Month(), end.Day()))
}

// IsAyyamiHa reports whether the period is the intercalary days rather than a
// named month.
func (p BadiPeriod) IsAyyamiHa() bool {
	return p.Month == BadiMonthAyyamiHa
}

// String returns a label such as "Sharaf 181 B.E.".
func (p BadiPeriod) String() string {
	return fmt.Sprintf("%s %d B.E.", p.Name, p.Year)
}

// badiPeriodForCivil resolves the period containing the civil date c and
// materializes its bounds in loc.
func badiPeriodForCivil(c time.Time, loc *time.Location) (BadiPeriod, error) {
	year, err := badiYearForCivil(c)
	if err != nil {
		return BadiPeriod{}, err
	}
	yearStart, err := nawRuz(year)
	if err != nil {
		return BadiPeriod{}, err
	}
	nextYearStart, err := nawRuz(year + 1)
	if err != nil {
		return BadiPeriod{}, err
	}

	// Ayyám-i-Há absorbs whatever the 19 named months do not cover.
	ayyamiHaDays := daysBetween(yearStart, nextYearStart) - badiNamedMonths*badiMonthDays
	ayyamiHaStart := badiNamedMonths - 1 // months 1–18 precede it
	offset := daysBetween(yearStart, c)

	var (
		month     int
		dayInYear int
		length    int
	)
	switch {
	case offset < ayyamiHaStart*badiMonthDays:
		month = offset/badiMonthDays + 1
		dayInYear = (month - 1) * badiMonthDays
		length = badiMonthDays
	case offset < ayyamiHaStart*badiMonthDays+ayyamiHaDays:
		month = BadiMonthAyyamiHa
		dayInYear = ayyamiHaStart * badiMonthDays
		length = ayyamiHaDays
	default:
		month = badiNamedMonths
		dayInYear = ayyamiHaStart*badiMonthDays + ayyamiHaDays
		length = badiMonthDays
	}

	periodStart := yearStart.AddDate(0, 0, dayInYear)
	return BadiPeriod{
		Year:  year,
		Month: month,
		Name:  badiPeriodName(month),
		Start: startOfDay(periodStart, loc),
		End:   startOfDay(periodStart.AddDate(0, 0, length), loc),
	}, nil
}

// earliestBadiInstant returns the first instant the Naw-Rúz table covers,
// evaluated in loc. Callers enumerating a range that may start earlier clamp to
// it rather than failing outright.
func earliestBadiInstant(loc *time.Location) time.Time {
	// firstBadiYear is always present in the table, so this cannot fail.
	start, _ := nawRuz(firstBadiYear)
	return startOfDay(start, loc)
}

// badiPeriodName returns the display name for a month ordinal, or "Ayyám-i-Há"
// for BadiMonthAyyamiHa.
func badiPeriodName(month int) string {
	if month == BadiMonthAyyamiHa {
		return "Ayyám-i-Há"
	}
	return badiMonthNames[month-1]
}

// badiYearForCivil returns the Badí' year containing the civil date c. Both the
// year's own Naw-Rúz and the following one must be in the table, since the
// latter fixes the length of Ayyám-i-Há.
func badiYearForCivil(c time.Time) (int, error) {
	year := firstBadiYear + c.Year() - firstNawRuzYear
	start, err := nawRuz(year)
	if err != nil {
		return 0, err
	}
	if c.Before(start) {
		year--
		if _, err := nawRuz(year); err != nil {
			return 0, err
		}
	}
	if _, err := nawRuz(year + 1); err != nil {
		return 0, err
	}
	return year, nil
}

// civilDate returns a date free of timezone and DST effects, for calendar
// arithmetic only. Use startOfDay to place the result in a real location.
func civilDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// daysBetween returns the number of whole days from civil date a to b.
func daysBetween(a, b time.Time) int {
	return int(b.Sub(a) / (24 * time.Hour))
}

// nawRuz returns the civil date of Naw-Rúz for a Badí' year.
func nawRuz(badiYear int) (time.Time, error) {
	i := badiYear - firstBadiYear
	if i < 0 || i >= len(nawRuzDay) {
		return time.Time{}, fmt.Errorf("%w: Badí' year %d", ErrBadiDateOutOfRange, badiYear)
	}
	return civilDate(firstNawRuzYear+i, time.March, nawRuzDay[i]), nil
}

// startOfDay materializes a civil date as the first instant of that day in loc.
func startOfDay(c time.Time, loc *time.Location) time.Time {
	return time.Date(c.Year(), c.Month(), c.Day(), 0, 0, 0, 0, loc)
}

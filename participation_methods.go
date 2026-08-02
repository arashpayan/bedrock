package bedrock

import (
	"database/sql"
	"errors"
	"fmt"
)

// Participation records how much of the community contributed to the Fund
// during one Bahá'í month. The numbers are gathered by hand rather than derived
// from receipts — a contribution can come from a household, arrive anonymously,
// or be given outside the Assembly's records — so any month may simply have no
// record. A missing month means "not known", never "nobody contributed".
//
// Only adults carry a denominator, so only adults yield a rate. The other
// categories are counts of contributors with nothing to divide by.
type Participation struct {
	Base
	BadiYear          int `db:"badi_year"`
	BadiMonth         int `db:"badi_month"` // 1–19; never BadiMonthAyyamiHa
	AdultsContributed int `db:"adults_contributed"`
	AdultsActive      int `db:"adults_active"`
	Youth             int `db:"youth"`
	JuniorYouth       int `db:"junior_youth"`
	Children          int `db:"children"`
}

// ParticipationCounts is the set of figures recorded for one month.
type ParticipationCounts struct {
	AdultsContributed int
	AdultsActive      int
	Youth             int
	JuniorYouth       int
	Children          int
}

// AdultRate returns adult participation as a percentage in [0, 100]. It returns
// 0 when no active adults are recorded, which callers should present as "no
// rate" rather than as zero participation.
func (p Participation) AdultRate() float64 {
	if p.AdultsActive <= 0 {
		return 0
	}
	return float64(p.AdultsContributed) / float64(p.AdultsActive) * 100
}

// Contributors returns everyone counted as having contributed, across all four
// categories.
func (p Participation) Contributors() int {
	return p.AdultsContributed + p.Youth + p.JuniorYouth + p.Children
}

// HasAdultRate reports whether a meaningful adult rate can be shown.
func (p Participation) HasAdultRate() bool {
	return p.AdultsActive > 0
}

// DeleteParticipation removes the record for a month. Deleting a month that has
// no record is not an error.
func (db *DB) DeleteParticipation(period BadiPeriod) error {
	if err := validateParticipationPeriod(period); err != nil {
		return err
	}

	query, args := db.sq.Delete("participation").
		Where("badi_year = ? AND badi_month = ?", period.Year, period.Month).
		MustSql()
	if _, err := db.conn.Exec(query, args...); err != nil {
		return fmt.Errorf("failed to delete participation: %w", err)
	}
	return nil
}

// Participation returns the record for a Bahá'í month, or (nil, nil) when none
// has been entered.
func (db *DB) Participation(period BadiPeriod) (*Participation, error) {
	if err := validateParticipationPeriod(period); err != nil {
		return nil, err
	}

	query, args := db.sq.Select(participationColumns...).
		From("participation").
		Where("badi_year = ? AND badi_month = ?", period.Year, period.Month).
		MustSql()

	var p Participation
	if err := db.conn.Get(&p, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get participation: %w", err)
	}
	return &p, nil
}

// ParticipationForPeriods returns the record for each of the given periods, as
// a slice aligned by index with nil for any month that has none.
//
// Callers pass periods in calendar order (from BadiPeriodsInRange) and this
// preserves that order. It deliberately does not sort or range-scan on
// badi_month: Ayyám-i-Há is month 0 but falls between months 18 and 19, so
// ordering by that column would put the year's periods in the wrong sequence.
// Any Ayyám-i-Há period in the input yields nil, since those days are counted
// with Mulk.
func (db *DB) ParticipationForPeriods(periods []BadiPeriod) ([]*Participation, error) {
	results := make([]*Participation, len(periods))
	if len(periods) == 0 {
		return results, nil
	}

	// One query for the whole span, then matched back onto the input order.
	years := make(map[int]bool)
	for _, period := range periods {
		years[period.Year] = true
	}
	yearList := make([]any, 0, len(years))
	for year := range years {
		yearList = append(yearList, year)
	}

	query, args := db.sq.Select(participationColumns...).
		From("participation").
		Where(squirrelIn("badi_year", len(yearList)), yearList...).
		MustSql()

	var rows []Participation
	if err := db.conn.Select(&rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list participation: %w", err)
	}

	type key struct{ year, month int }
	byMonth := make(map[key]*Participation, len(rows))
	for i := range rows {
		byMonth[key{rows[i].BadiYear, rows[i].BadiMonth}] = &rows[i]
	}
	for i, period := range periods {
		if period.IsAyyamiHa() {
			continue
		}
		results[i] = byMonth[key{period.Year, period.Month}]
	}
	return results, nil
}

// SetParticipation creates or replaces the record for a Bahá'í month.
//
// Ayyám-i-Há cannot be recorded on its own: those days are counted with Mulk,
// the month they follow, so a record for Mulk covers 23 or 24 days.
func (db *DB) SetParticipation(period BadiPeriod, counts ParticipationCounts) (*Participation, error) {
	if err := validateParticipationPeriod(period); err != nil {
		return nil, err
	}
	if err := validateParticipationCounts(counts); err != nil {
		return nil, err
	}

	query, args := db.sq.Insert("participation").
		SetMap(map[string]any{
			"badi_year":          period.Year,
			"badi_month":         period.Month,
			"adults_contributed": counts.AdultsContributed,
			"adults_active":      counts.AdultsActive,
			"youth":              counts.Youth,
			"junior_youth":       counts.JuniorYouth,
			"children":           counts.Children,
		}).
		Suffix(`ON CONFLICT(badi_year, badi_month) DO UPDATE SET
			adults_contributed = excluded.adults_contributed,
			adults_active = excluded.adults_active,
			youth = excluded.youth,
			junior_youth = excluded.junior_youth,
			children = excluded.children`).
		Suffix("RETURNING " + participationColumnList).
		MustSql()

	var p Participation
	if err := db.conn.Get(&p, query, args...); err != nil {
		return nil, fmt.Errorf("failed to set participation: %w", err)
	}
	return &p, nil
}

// participationColumns is the explicit column list used for every read, so a
// future column addition surfaces here rather than silently changing scans.
var participationColumns = []string{
	"id", "badi_year", "badi_month", "adults_contributed", "adults_active",
	"youth", "junior_youth", "children", "created_at", "modified_at",
}

const participationColumnList = "id, badi_year, badi_month, adults_contributed, adults_active, " +
	"youth, junior_youth, children, created_at, modified_at"

// squirrelIn builds an `column IN (?, ?, …)` clause for n placeholders.
func squirrelIn(column string, n int) string {
	placeholders := make([]byte, 0, 2*n)
	for i := range n {
		if i > 0 {
			placeholders = append(placeholders, ',')
		}
		placeholders = append(placeholders, '?')
	}
	return column + " IN (" + string(placeholders) + ")"
}

// validateParticipationCounts rejects figures that cannot describe a real month.
func validateParticipationCounts(c ParticipationCounts) error {
	for _, f := range []struct {
		name  string
		value int
	}{
		{"adults who contributed", c.AdultsContributed},
		{"active adults", c.AdultsActive},
		{"youth", c.Youth},
		{"junior youth", c.JuniorYouth},
		{"children", c.Children},
	} {
		if f.value < 0 {
			return fmt.Errorf("%s cannot be negative, got %d", f.name, f.value)
		}
	}

	if c.AdultsContributed > c.AdultsActive {
		return fmt.Errorf("adults who contributed (%d) cannot exceed active adults (%d)",
			c.AdultsContributed, c.AdultsActive)
	}
	return nil
}

// validateParticipationPeriod rejects periods that cannot hold a record.
func validateParticipationPeriod(period BadiPeriod) error {
	if period.IsAyyamiHa() {
		return errors.New("participation for Ayyám-i-Há is recorded with Mulk, the month it follows")
	}
	if period.Month < 1 || period.Month > badiNamedMonths || period.Year < firstBadiYear {
		return fmt.Errorf("invalid Badí' period %q", period.String())
	}
	return nil
}

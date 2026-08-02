package bedrock

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// The Assembly fiscal year runs from May 1 through April 30 and is identified
// by its starting calendar year: fiscal year 2026 spans May 1, 2026 through
// April 30, 2027.
const (
	fiscalYearStartMonth = time.May
	fiscalYearStartDay   = 1
)

// FundraisingGoal is the editable fundraising target for a single fiscal year.
type FundraisingGoal struct {
	Base
	FiscalYear int   `db:"fiscal_year"` // starting calendar year (see fiscalYearStartMonth)
	Amount     Money `db:"-"`           // target amount; scanned from amount + currency columns
}

// fundraisingGoalRow mirrors the fundraising_goals table for scanning, with the
// money split into its stored columns.
type fundraisingGoalRow struct {
	Base
	FiscalYear int      `db:"fiscal_year"`
	Amount     int64    `db:"amount"`
	Currency   Currency `db:"currency"`
}

func (r fundraisingGoalRow) toGoal() FundraisingGoal {
	return FundraisingGoal{
		Base:       r.Base,
		FiscalYear: r.FiscalYear,
		Amount:     Money{Amount: r.Amount, Currency: r.Currency},
	}
}

// FundraisingProgress summarizes contributions toward the goal for one fiscal
// year. Only contributions to non-earmarked funds (Item.CountsTowardGoal) with
// a SoldAt within the fiscal year and in the Assembly's default currency are
// counted. Goal is nil when no goal has been set for the year.
type FundraisingProgress struct {
	FiscalYear int       // starting calendar year
	Start      time.Time // inclusive start of the fiscal year (Assembly timezone)
	End        time.Time // exclusive end (start of the next fiscal year)
	Goal       *Money    // target, or nil if unset
	Raised     Money     // total counted contributions
}

// Percent returns progress toward the goal as a value in [0, 100], or 0 when no
// goal is set or the goal is zero.
func (p FundraisingProgress) Percent() float64 {
	if p.Goal == nil || p.Goal.Amount <= 0 {
		return 0
	}
	pct := float64(p.Raised.Amount) / float64(p.Goal.Amount) * 100
	return max(0, pct)
}

// Remaining returns how much is still needed to reach the goal, clamped at zero
// (an exceeded goal returns zero). It returns a zero Money in the Raised
// currency when no goal is set.
func (p FundraisingProgress) Remaining() Money {
	if p.Goal == nil {
		return Money{Amount: 0, Currency: p.Raised.Currency}
	}
	return Money{Amount: max(0, p.Goal.Amount-p.Raised.Amount), Currency: p.Goal.Currency}
}

// CurrentFiscalYear returns the fiscal year containing the present moment in the
// Assembly's timezone.
func (db *DB) CurrentFiscalYear() (int, error) {
	assembly, err := db.Assembly()
	if err != nil {
		return 0, err
	}
	return FiscalYearForDate(time.Now(), assembly.Timezone), nil
}

// DeleteFundraisingGoal removes the goal for a fiscal year. It is not an error
// to delete a year that has no goal.
func (db *DB) DeleteFundraisingGoal(fiscalYear int) error {
	query, args := db.sq.Delete("fundraising_goals").
		Where("fiscal_year = ?", fiscalYear).
		MustSql()
	if _, err := db.conn.Exec(query, args...); err != nil {
		return fmt.Errorf("failed to delete fundraising goal: %w", err)
	}
	return nil
}

// FundraisingGoal returns the goal for the given fiscal year, or (nil, nil) when
// none has been set.
func (db *DB) FundraisingGoal(fiscalYear int) (*FundraisingGoal, error) {
	query, args := db.sq.Select("id", "fiscal_year", "amount", "currency", "created_at", "modified_at").
		From("fundraising_goals").
		Where("fiscal_year = ?", fiscalYear).
		MustSql()

	var row fundraisingGoalRow
	if err := db.conn.Get(&row, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get fundraising goal: %w", err)
	}
	goal := row.toGoal()
	return &goal, nil
}

// FundraisingGoals returns every fiscal-year goal, most recent year first.
func (db *DB) FundraisingGoals() ([]FundraisingGoal, error) {
	query, args := db.sq.Select("id", "fiscal_year", "amount", "currency", "created_at", "modified_at").
		From("fundraising_goals").
		OrderBy("fiscal_year DESC").
		MustSql()

	var rows []fundraisingGoalRow
	if err := db.conn.Select(&rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list fundraising goals: %w", err)
	}
	goals := make([]FundraisingGoal, len(rows))
	for i, r := range rows {
		goals[i] = r.toGoal()
	}
	return goals, nil
}

// FundraisingProgress computes contributions raised toward the goal for the
// given fiscal year. Only contributions to non-earmarked funds, received within
// the fiscal year, and in the Assembly's default currency are counted.
func (db *DB) FundraisingProgress(fiscalYear int) (*FundraisingProgress, error) {
	assembly, err := db.Assembly()
	if err != nil {
		return nil, err
	}
	start, end := FiscalYearRange(fiscalYear, assembly.Timezone)

	raised, err := db.countedContributions(start, end, assembly.DefaultCurrency)
	if err != nil {
		return nil, err
	}

	goal, err := db.FundraisingGoal(fiscalYear)
	if err != nil {
		return nil, err
	}

	progress := &FundraisingProgress{
		FiscalYear: fiscalYear,
		Start:      start,
		End:        end,
		Raised:     Money{Amount: raised, Currency: assembly.DefaultCurrency},
	}
	if goal != nil {
		progress.Goal = &goal.Amount
	}
	return progress, nil
}

// SetFundraisingGoal creates or replaces the goal for a fiscal year. The amount
// must be positive and in the Assembly's default currency. Setting a goal again
// for the same year overwrites the previous amount.
func (db *DB) SetFundraisingGoal(fiscalYear int, amount Money) (*FundraisingGoal, error) {
	if amount.Amount <= 0 {
		return nil, fmt.Errorf("fundraising goal must be positive, got %d cents", amount.Amount)
	}

	assembly, err := db.Assembly()
	if err != nil {
		return nil, err
	}
	if amount.Currency != assembly.DefaultCurrency {
		return nil, fmt.Errorf("goal currency %s does not match the Assembly's default currency %s", amount.Currency, assembly.DefaultCurrency)
	}

	query, args := db.sq.Insert("fundraising_goals").
		SetMap(map[string]any{
			"fiscal_year": fiscalYear,
			"amount":      amount.Amount,
			"currency":    amount.Currency,
		}).
		Suffix("ON CONFLICT(fiscal_year) DO UPDATE SET amount = excluded.amount, currency = excluded.currency").
		Suffix("RETURNING id, fiscal_year, amount, currency, created_at, modified_at").
		MustSql()

	var row fundraisingGoalRow
	if err := db.conn.Get(&row, query, args...); err != nil {
		return nil, fmt.Errorf("failed to set fundraising goal: %w", err)
	}
	goal := row.toGoal()
	return &goal, nil
}

// countedContributions sums the contributions that count toward the fundraising
// goal with a SoldAt in the half-open range [start, end): those to non-earmarked
// funds (Item.CountsTowardGoal) priced in the given currency. Deposit status is
// deliberately ignored — a contribution counts when it is received.
//
// This is the single definition of "counts toward the goal"; FundraisingProgress
// and MonthlyReport both go through it so the fiscal-year total and any
// shorter-range total can never drift apart.
func (db *DB) countedContributions(start, end time.Time, currency Currency) (int64, error) {
	query := `
		SELECT COALESCE(SUM(ri.price), 0)
		FROM receipt_items ri
		JOIN receipts r ON r.id = ri.receipt_id
		JOIN items i ON i.id = ri.item_id
		WHERE i.counts_toward_goal = 1
		  AND ri.currency = ?
		  AND r.sold_at >= ? AND r.sold_at < ?`

	var total int64
	if err := db.conn.Get(&total, query, currency, start, end); err != nil {
		return 0, fmt.Errorf("failed to sum counted contributions: %w", err)
	}
	return total, nil
}

// FiscalYearForDate returns the starting calendar year of the fiscal year that
// contains t, evaluated in loc. Dates in or after May belong to that calendar
// year's fiscal year; January through April belong to the prior year's.
func FiscalYearForDate(t time.Time, loc *time.Location) int {
	if loc == nil {
		loc = time.UTC
	}
	local := t.In(loc)
	if local.Month() >= fiscalYearStartMonth {
		return local.Year()
	}
	return local.Year() - 1
}

// FiscalYearRange returns the half-open range [start, end) for the fiscal year
// identified by its starting calendar year, evaluated in loc. start is midnight
// on May 1 of fiscalYear; end is midnight on May 1 of the following year.
func FiscalYearRange(fiscalYear int, loc *time.Location) (start, end time.Time) {
	if loc == nil {
		loc = time.UTC
	}
	start = time.Date(fiscalYear, fiscalYearStartMonth, fiscalYearStartDay, 0, 0, 0, 0, loc)
	end = time.Date(fiscalYear+1, fiscalYearStartMonth, fiscalYearStartDay, 0, 0, 0, 0, loc)
	return start, end
}

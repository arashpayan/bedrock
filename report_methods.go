package bedrock

import (
	"fmt"
	"maps"
	"slices"
	"time"
)

// MonthlyReport is the Assembly's report for one Bahá'í month: what came in
// toward the annual goal, what went out, and how the fiscal year is pacing
// against its fundraising goal. It is built by DB.MonthlyReport.
//
// Two ranges are in play and they are deliberately different. The period
// figures (PeriodContributions, Expenses) cover the whole Bahá'í month, even
// the one month a year that straddles the 1 May fiscal-year boundary — a report
// headed "Jamál" should account for all of Jamál. The fiscal-year figures
// (RaisedToDate onward) always run from the start of the fiscal year to the end
// of the period, clipped at the fiscal year's end. StraddlesFiscalYear marks
// the report where the two disagree.
type MonthlyReport struct {
	// Period is the Bahá'í month covered, resolved in the Assembly timezone.
	Period BadiPeriod

	FiscalYear          int       // starting calendar year, e.g. 2026
	FiscalYearStart     time.Time // inclusive
	FiscalYearEnd       time.Time // exclusive
	StraddlesFiscalYear bool      // the period began before this fiscal year

	// PeriodContributions is what was received toward the annual goal during
	// the period: contributions to non-earmarked funds, in the Assembly's
	// default currency. Deposit status is irrelevant.
	PeriodContributions Money

	// Expenses lists every expense line paid during the period, oldest first,
	// with ExpensesByCategory holding the same spending grouped by category.
	Expenses           []ExpenseDetail
	ExpensesByCategory []ExpenseCategoryTotal

	// ExpensesTotal is the grand total in the Assembly's default currency.
	// Spending in any other currency is summed separately into
	// ExpensesInOtherCurrencies (ordered by currency, normally empty) rather
	// than added in, since currencies cannot be summed.
	ExpensesTotal             Money
	ExpensesInOtherCurrencies []Money

	// Goal is the fiscal year's fundraising target, nil when none is set. The
	// remaining fields are zero in that case, except PeriodsRemaining, which is
	// pure calendar arithmetic.
	Goal *Money

	// RaisedToDate is everything counted toward the goal from the start of the
	// fiscal year through the end of the period.
	RaisedToDate Money

	// ExpectedToDate is the share of the goal that should have been raised by
	// the end of the period, pro-rated by days elapsed in the fiscal year.
	ExpectedToDate Money

	// Variance is RaisedToDate minus ExpectedToDate: positive is ahead of
	// schedule, negative behind. It is the only signed Money here.
	Variance Money

	// RemainingGoal is what is still needed, clamped at zero once the goal is met.
	RemainingGoal Money

	// PeriodsRemaining counts the Bahá'í months that begin between the end of
	// this period and the end of the fiscal year, including the month now
	// starting. AdjustedPeriodGoal spreads RemainingGoal evenly across them,
	// rounded up to the cent so the schedule never undershoots; it is zero when
	// no months remain.
	PeriodsRemaining   int
	AdjustedPeriodGoal Money

	// Participation is how much of the community contributed during the period,
	// nil when it has not been recorded. On an Ayyám-i-Há report it is always
	// nil: those days are counted with Mulk, and ParticipationBelongsTo names
	// the month that holds them.
	Participation          *Participation
	ParticipationBelongsTo string

	// ParticipationHistory covers the named Bahá'í months of the fiscal year up
	// to and including this period, oldest first, for charting the adult rate
	// over the year. Months with no record carry a nil Data — distinct from a
	// month in which nobody contributed.
	ParticipationHistory []ParticipationPoint
}

// ParticipationPoint pairs one Bahá'í month with its participation record, if
// any. Data is nil when the month has not been recorded.
type ParticipationPoint struct {
	Period BadiPeriod
	Data   *Participation
}

// CurrentBadiPeriod returns the Bahá'í month containing the present moment in
// the Assembly's timezone.
func (db *DB) CurrentBadiPeriod() (BadiPeriod, error) {
	assembly, err := db.Assembly()
	if err != nil {
		return BadiPeriod{}, err
	}
	return BadiPeriodForDate(time.Now(), assembly.Timezone)
}

// MonthlyReport builds the report for a Bahá'í month. The period is re-resolved
// in the Assembly's timezone, so one built in another location still names the
// same Bahá'í month.
func (db *DB) MonthlyReport(period BadiPeriod) (*MonthlyReport, error) {
	if period.Start.IsZero() || !period.End.After(period.Start) {
		return nil, fmt.Errorf("invalid Badí' period %q", period.String())
	}

	assembly, err := db.Assembly()
	if err != nil {
		return nil, err
	}
	timezone := assembly.Timezone
	if timezone == nil {
		timezone = time.UTC
	}

	// Resolving from an instant in the middle of the period keeps the answer
	// stable no matter which timezone the caller used for the bounds.
	midpoint := period.Start.Add(time.Duration(period.Days()) * 12 * time.Hour)
	period, err = BadiPeriodForDate(midpoint, timezone)
	if err != nil {
		return nil, err
	}

	// The fiscal year is the one holding the period's last day, so the month
	// straddling 1 May reports against the fiscal year it ends in — the one
	// whose pacing the treasurer is about to act on.
	fiscalYear := FiscalYearForDate(period.End.Add(-time.Nanosecond), timezone)
	fyStart, fyEnd := FiscalYearRange(fiscalYear, timezone)

	reportEnd := period.End
	if reportEnd.After(fyEnd) {
		reportEnd = fyEnd
	}

	report := &MonthlyReport{
		Period:              period,
		FiscalYear:          fiscalYear,
		FiscalYearStart:     fyStart,
		FiscalYearEnd:       fyEnd,
		StraddlesFiscalYear: period.Start.Before(fyStart),
	}

	periodTotal, err := db.countedContributions(period.Start, period.End, assembly.DefaultCurrency)
	if err != nil {
		return nil, err
	}
	report.PeriodContributions = Money{Amount: periodTotal, Currency: assembly.DefaultCurrency}

	raised, err := db.countedContributions(fyStart, reportEnd, assembly.DefaultCurrency)
	if err != nil {
		return nil, err
	}
	report.RaisedToDate = Money{Amount: raised, Currency: assembly.DefaultCurrency}

	if report.Expenses, err = db.ExpensesInRange(period.Start, period.End); err != nil {
		return nil, err
	}
	if report.ExpensesByCategory, err = db.ExpensesByCategory(period.Start, period.End); err != nil {
		return nil, err
	}
	report.ExpensesTotal, report.ExpensesInOtherCurrencies = totalExpenses(report.ExpensesByCategory, assembly.DefaultCurrency)

	periods, err := BadiPeriodsInRange(reportEnd, fyEnd, timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to count the Bahá'í months left in fiscal year %d: %w", fiscalYear, err)
	}
	report.PeriodsRemaining = len(periods)

	if err := db.addParticipation(report, timezone); err != nil {
		return nil, err
	}

	goal, err := db.FundraisingGoal(fiscalYear)
	if err != nil {
		return nil, err
	}
	if goal == nil {
		zero := Money{Currency: assembly.DefaultCurrency}
		report.ExpectedToDate, report.Variance = zero, zero
		report.RemainingGoal, report.AdjustedPeriodGoal = zero, zero
		return report, nil
	}
	report.Goal = &goal.Amount

	elapsedDays := daysBetween(civilDateOf(fyStart), civilDateOf(reportEnd))
	fiscalYearDays := daysBetween(civilDateOf(fyStart), civilDateOf(fyEnd))
	expected := prorate(goal.Amount.Amount, elapsedDays, fiscalYearDays)
	remaining := max(0, goal.Amount.Amount-raised)

	report.ExpectedToDate = Money{Amount: expected, Currency: goal.Amount.Currency}
	report.Variance = Money{Amount: raised - expected, Currency: goal.Amount.Currency}
	report.RemainingGoal = Money{Amount: remaining, Currency: goal.Amount.Currency}
	report.AdjustedPeriodGoal = Money{
		Amount:   divideRoundingUp(remaining, report.PeriodsRemaining),
		Currency: goal.Amount.Currency,
	}
	return report, nil
}

// addParticipation fills in the report's participation figures: the reported
// month's own record, and the fiscal year to date for charting.
//
// Ayyám-i-Há holds no record of its own — those days are counted with Mulk — so
// an Ayyám-i-Há report points at Mulk instead, and the history skips it rather
// than charting a gap that only reflects how the calendar works.
func (db *DB) addParticipation(report *MonthlyReport, timezone *time.Location) error {
	if report.Period.IsAyyamiHa() {
		mulk, err := PreviousBadiPeriod(report.Period)
		if err != nil {
			return err
		}
		report.ParticipationBelongsTo = mulk.String()
	} else {
		participation, err := db.Participation(report.Period)
		if err != nil {
			return err
		}
		report.Participation = participation
	}

	// A fiscal year that opens before the Naw-Rúz table begins can only be
	// charted from where the table does. The report itself is still valid, so
	// the history is clamped rather than failing the whole thing.
	historyStart := report.FiscalYearStart
	if earliest := earliestBadiInstant(timezone); historyStart.Before(earliest) {
		historyStart = earliest
	}

	elapsed, err := BadiPeriodsInRange(historyStart, report.Period.End, timezone)
	if err != nil {
		return fmt.Errorf("failed to list the Bahá'í months of fiscal year %d: %w", report.FiscalYear, err)
	}

	named := make([]BadiPeriod, 0, len(elapsed))
	for _, period := range elapsed {
		if !period.IsAyyamiHa() {
			named = append(named, period)
		}
	}

	records, err := db.ParticipationForPeriods(named)
	if err != nil {
		return err
	}

	report.ParticipationHistory = make([]ParticipationPoint, len(named))
	for i, period := range named {
		report.ParticipationHistory[i] = ParticipationPoint{Period: period, Data: records[i]}
	}
	return nil
}

// MostRecentlyCompletedBadiPeriod returns the Bahá'í month that ended most
// recently in the Assembly's timezone — the month a report generated today
// would cover.
func (db *DB) MostRecentlyCompletedBadiPeriod() (BadiPeriod, error) {
	current, err := db.CurrentBadiPeriod()
	if err != nil {
		return BadiPeriod{}, err
	}
	return PreviousBadiPeriod(current)
}

// civilDateOf strips t to the civil date of its own location, for day counting
// that DST transitions cannot skew.
func civilDateOf(t time.Time) time.Time {
	return civilDate(t.Year(), t.Month(), t.Day())
}

// divideRoundingUp splits amount into n equal parts, rounding up so that n
// parts always cover the whole. It returns 0 when there is nothing to divide
// into.
func divideRoundingUp(amount int64, n int) int64 {
	if n <= 0 || amount <= 0 {
		return 0
	}
	return (amount + int64(n) - 1) / int64(n)
}

// prorate returns the share of amount corresponding to elapsed out of total,
// rounded to the nearest unit. It clamps to [0, amount].
func prorate(amount int64, elapsed, total int) int64 {
	if total <= 0 || elapsed <= 0 {
		return 0
	}
	if elapsed >= total {
		return amount
	}
	return (amount*int64(elapsed) + int64(total)/2) / int64(total)
}

// totalExpenses sums category totals into a grand total in the primary
// currency, keeping any other currencies apart rather than adding them in.
func totalExpenses(totals []ExpenseCategoryTotal, primary Currency) (Money, []Money) {
	total := Money{Currency: primary}
	others := make(map[Currency]int64)
	for _, t := range totals {
		if t.Total.Currency == primary {
			total.Amount += t.Total.Amount
			continue
		}
		others[t.Total.Currency] += t.Total.Amount
	}

	var rest []Money
	for _, currency := range slices.Sorted(maps.Keys(others)) {
		rest = append(rest, Money{Amount: others[currency], Currency: currency})
	}
	return total, rest
}

package bedrock

import (
	"fmt"
	"time"
)

// Expense queries spanning more than one withdrawal. These read only the
// `expenses` table, so they cover cash spending exclusively: an in-kind receipt
// line may carry a CategoryID recording what the donated value was spent on,
// but no money left a bank account, and it is never reported here.
//
// Both queries take a half-open range [start, end) over the withdrawal's
// transacted_at, matching how FundraisingProgress ranges over receipts.

// ExpenseCategoryTotal is the sum of one category's expenses over a range. A
// category that was spent in more than one currency yields one row per
// currency, since amounts in different currencies cannot be added.
type ExpenseCategoryTotal struct {
	CategoryID   ID
	CategoryName string
	Total        Money
}

// ExpenseDetail is one expense line together with the withdrawal it belongs to,
// carrying everything a report or ledger view needs without further lookups.
type ExpenseDetail struct {
	Expense
	CategoryName string
	AccountName  string
	PayeeName    string    // empty when the withdrawal has no payee
	Memo         string    // the withdrawal's memo, empty when unset
	CheckNumber  *string   // nil unless paid by check
	TransactedAt time.Time // when the withdrawal occurred
}

// ExpensesByCategory totals expenses per category over the half-open range
// [start, end), ordered by category name. Categories with no spending in the
// range are omitted. See ExpenseCategoryTotal for how mixed currencies are
// reported.
func (db *DB) ExpensesByCategory(start, end time.Time) ([]ExpenseCategoryTotal, error) {
	query := `
		SELECT c.id, c.name, e.currency, SUM(e.amount)
		FROM expenses e
		JOIN transactions t ON t.id = e.transaction_id
		JOIN categories c ON c.id = e.category_id
		WHERE t.transacted_at >= ? AND t.transacted_at < ?
		GROUP BY c.id, c.name, e.currency
		ORDER BY c.name ASC, e.currency ASC`

	rows, err := db.conn.Query(query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query expense totals: %w", err)
	}
	defer rows.Close()

	var totals []ExpenseCategoryTotal
	for rows.Next() {
		var (
			total    ExpenseCategoryTotal
			amount   int64
			currency Currency
		)
		if err := rows.Scan(&total.CategoryID, &total.CategoryName, &currency, &amount); err != nil {
			return nil, fmt.Errorf("failed to scan expense total: %w", err)
		}
		total.Total = Money{Amount: amount, Currency: currency}
		totals = append(totals, total)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating expense totals: %w", err)
	}
	return totals, nil
}

// ExpensesInRange returns every expense line whose withdrawal falls in the
// half-open range [start, end), oldest first. Lines of the same withdrawal keep
// their insertion order.
func (db *DB) ExpensesInRange(start, end time.Time) ([]ExpenseDetail, error) {
	query := `
		SELECT e.id, e.transaction_id, e.category_id, e.description, e.amount,
		       e.currency, e.created_at, e.modified_at,
		       c.name, ba.name, COALESCE(p.name, ''), COALESCE(t.memo, ''),
		       t.check_number, t.transacted_at
		FROM expenses e
		JOIN transactions t ON t.id = e.transaction_id
		JOIN categories c ON c.id = e.category_id
		JOIN bank_accounts ba ON ba.id = t.account_id
		LEFT JOIN parties p ON p.id = t.payee_id
		WHERE t.transacted_at >= ? AND t.transacted_at < ?
		ORDER BY t.transacted_at ASC, t.id ASC, e.id ASC`

	rows, err := db.conn.Query(query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query expenses: %w", err)
	}
	defer rows.Close()

	var expenses []ExpenseDetail
	for rows.Next() {
		var (
			detail   ExpenseDetail
			amount   int64
			currency Currency
		)
		if err := rows.Scan(
			&detail.ID,
			&detail.TransactionID,
			&detail.CategoryID,
			&detail.Description,
			&amount,
			&currency,
			&detail.CreatedAt,
			&detail.ModifiedAt,
			&detail.CategoryName,
			&detail.AccountName,
			&detail.PayeeName,
			&detail.Memo,
			&detail.CheckNumber,
			&detail.TransactedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan expense: %w", err)
		}
		detail.Amount = Money{Amount: amount, Currency: currency}
		expenses = append(expenses, detail)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating expenses: %w", err)
	}
	return expenses, nil
}

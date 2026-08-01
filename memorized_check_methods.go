package bedrock

import (
	"database/sql"
	"errors"
	"fmt"
)

// MemorizedCheck is a reusable template for a recurring check. It stores the
// parts that stay the same between instances (account, payee, memo, and the
// expense lines); the date and check number are entered fresh each time it is
// used and are not stored here.
type MemorizedCheck struct {
	Base
	Name      string `db:"name"`
	AccountID ID     `db:"account_id"`
	PayeeID   ID     `db:"payee_id"`
	Memo      string `db:"memo"`
}

// MemorizedCheckExpense is one template expense line on a memorized check.
type MemorizedCheckExpense struct {
	Base
	MemorizedCheckID ID      `db:"memorized_check_id"`
	CategoryID       ID      `db:"category_id"`
	Description      *string `db:"description"`
	Amount           Money   `db:"-"`
}

// CreateMemorizedCheck stores a new memorized check together with its expense
// lines in a single transaction. name must be non-empty and there must be at
// least one expense; all expenses must share one currency that matches the
// account's currency (mirroring CreateWithdrawal).
func (db *DB) CreateMemorizedCheck(name string, accountID, payeeID ID, memo string, expenses []ExpenseItem) (*MemorizedCheck, error) {
	if name == "" {
		return nil, fmt.Errorf("memorized check name cannot be empty")
	}
	if len(expenses) == 0 {
		return nil, fmt.Errorf("at least one expense item is required")
	}

	var currency Currency
	for i, expense := range expenses {
		if expense.Amount.Amount <= 0 {
			return nil, fmt.Errorf("expense %d amount must be positive, got %d cents", i, expense.Amount.Amount)
		}
		if i == 0 {
			currency = expense.Amount.Currency
		} else if expense.Amount.Currency != currency {
			return nil, fmt.Errorf("all expenses must have the same currency, got %s and %s", currency, expense.Amount.Currency)
		}
	}

	var accountCurrency Currency
	if err := db.conn.Get(&accountCurrency, "SELECT currency FROM bank_accounts WHERE id = ?", accountID); err != nil {
		return nil, fmt.Errorf("failed to get account currency: %w", err)
	}
	if accountCurrency != currency {
		return nil, fmt.Errorf("expense currency %s does not match account currency %s", currency, accountCurrency)
	}

	tx, err := db.conn.Beginx()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	headerQuery, headerArgs := db.sq.Insert("memorized_checks").
		SetMap(map[string]any{
			"name":       name,
			"account_id": accountID,
			"payee_id":   payeeID,
			"memo":       memo,
		}).
		Suffix("RETURNING *").
		MustSql()

	var mc MemorizedCheck
	if err := tx.Get(&mc, headerQuery, headerArgs...); err != nil {
		return nil, fmt.Errorf("failed to insert memorized check: %w", err)
	}

	for _, expense := range expenses {
		expenseQuery, expenseArgs := db.sq.Insert("memorized_check_expenses").
			SetMap(map[string]any{
				"memorized_check_id": mc.ID,
				"category_id":        expense.CategoryID,
				"description":        expense.Description,
				"amount":             expense.Amount.Amount,
				"currency":           expense.Amount.Currency,
			}).
			MustSql()
		if _, err := tx.Exec(expenseQuery, expenseArgs...); err != nil {
			return nil, fmt.Errorf("failed to insert memorized check expense: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &mc, nil
}

// DeleteMemorizedCheck removes a memorized check and its expense lines in a
// single transaction.
func (db *DB) DeleteMemorizedCheck(id ID) error {
	tx, err := db.conn.Beginx()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	expensesQuery, expensesArgs := db.sq.Delete("memorized_check_expenses").
		Where("memorized_check_id = ?", id).
		MustSql()
	if _, err := tx.Exec(expensesQuery, expensesArgs...); err != nil {
		return fmt.Errorf("failed to delete memorized check expenses: %w", err)
	}

	headerQuery, headerArgs := db.sq.Delete("memorized_checks").
		Where("id = ?", id).
		MustSql()
	result, err := tx.Exec(headerQuery, headerArgs...)
	if err != nil {
		return fmt.Errorf("failed to delete memorized check: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("memorized check with id %d not found", id)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// MemorizedCheck retrieves a memorized check by ID.
func (db *DB) MemorizedCheck(id ID) (*MemorizedCheck, error) {
	query, args := db.sq.Select("id", "name", "account_id", "payee_id", "memo", "created_at", "modified_at").
		From("memorized_checks").
		Where("id = ?", id).
		MustSql()

	var mc MemorizedCheck
	if err := db.conn.Get(&mc, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get memorized check: %w", err)
	}
	return &mc, nil
}

// MemorizedCheckExpenses returns the expense lines for a memorized check, in
// insertion order.
func (db *DB) MemorizedCheckExpenses(memorizedCheckID ID) ([]MemorizedCheckExpense, error) {
	query, args := db.sq.Select("id", "memorized_check_id", "category_id", "description", "amount", "currency", "created_at", "modified_at").
		From("memorized_check_expenses").
		Where("memorized_check_id = ?", memorizedCheckID).
		OrderBy("id ASC").
		MustSql()

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query memorized check expenses: %w", err)
	}
	defer rows.Close()

	var expenses []MemorizedCheckExpense
	for rows.Next() {
		var e MemorizedCheckExpense
		var amount int64
		var currency Currency
		if err := rows.Scan(
			&e.ID,
			&e.MemorizedCheckID,
			&e.CategoryID,
			&e.Description,
			&amount,
			&currency,
			&e.CreatedAt,
			&e.ModifiedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan memorized check expense: %w", err)
		}
		e.Amount = Money{Amount: amount, Currency: currency}
		expenses = append(expenses, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating memorized check expenses: %w", err)
	}
	return expenses, nil
}

// MemorizedChecks lists all memorized checks, ordered by name (case-insensitive).
func (db *DB) MemorizedChecks() ([]MemorizedCheck, error) {
	query, args := db.sq.Select("id", "name", "account_id", "payee_id", "memo", "created_at", "modified_at").
		From("memorized_checks").
		OrderBy("name COLLATE NOCASE ASC").
		MustSql()

	var checks []MemorizedCheck
	if err := db.conn.Select(&checks, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list memorized checks: %w", err)
	}
	return checks, nil
}

// RenameMemorizedCheck changes a memorized check's name.
func (db *DB) RenameMemorizedCheck(id ID, name string) (*MemorizedCheck, error) {
	if name == "" {
		return nil, fmt.Errorf("memorized check name cannot be empty")
	}

	query, args := db.sq.Update("memorized_checks").
		SetMap(map[string]any{"name": name}).
		Where("id = ?", id).
		Suffix("RETURNING id, name, account_id, payee_id, memo, created_at, modified_at").
		MustSql()

	var mc MemorizedCheck
	if err := db.conn.Get(&mc, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("memorized check with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to rename memorized check: %w", err)
	}
	return &mc, nil
}

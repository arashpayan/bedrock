package bedrock

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// ErrInsufficientFunds is wrapped by the error a transfer returns when it would
// take an account's own balance (excluding its sub-accounts) below zero.
var ErrInsufficientFunds = errors.New("insufficient funds")

// earmarkFundingMemo is the memo on the transfer that
// CreateEarmarkFundedFromParent records.
const earmarkFundingMemo = "Initial funding"

// transferColumns lists the columns of the transfers table in the order
// transferRow scans them.
const transferColumns = "id, from_account_id, to_account_id, amount, currency, memo, transferred_at, created_at, modified_at"

// Transfer moves money between a bank account and one of its direct
// sub-accounts, such as setting funds aside in an earmark or releasing them
// back to the parent. No money moves at the bank, so a transfer is stored as
// this header plus two ordinary transaction legs that reference it through
// Transaction.TransferID: a negative leg on FromAccountID and a positive leg on
// ToAccountID. Balance and ledger queries count the legs like any other
// transaction, so each account's own balance moves while the parent's combined
// balance is unchanged.
//
// Transfer legs never appear on a bank statement, so they are excluded from
// reconciliation, and the deposit and withdrawal editors refuse them; a
// transfer is changed only through UpdateTransfer and DeleteTransfer.
type Transfer struct {
	Base
	FromAccountID ID
	ToAccountID   ID
	Amount        Money // always positive
	Memo          string
	TransferredAt time.Time
}

// transferRow is used for database scanning with flat fields.
type transferRow struct {
	Base
	FromAccountID ID        `db:"from_account_id"`
	ToAccountID   ID        `db:"to_account_id"`
	Amount        int64     `db:"amount"`
	Currency      Currency  `db:"currency"`
	Memo          string    `db:"memo"`
	TransferredAt time.Time `db:"transferred_at"`
}

func (r transferRow) toTransfer() Transfer {
	return Transfer{
		Base:          r.Base,
		FromAccountID: r.FromAccountID,
		ToAccountID:   r.ToAccountID,
		Amount:        Money{Amount: r.Amount, Currency: r.Currency},
		Memo:          r.Memo,
		TransferredAt: r.TransferredAt,
	}
}

// CreateTransfer moves amount from one bank account to another, dated
// transferredAt, writing the transfer and both of its legs in a single
// database transaction.
//
// One account must be the direct parent of the other, both must be active, and
// amount must be positive and in the accounts' currency. The source account
// may not be overdrawn: its own balance, excluding its sub-accounts, must cover
// amount, or the error wraps ErrInsufficientFunds.
func (db *DB) CreateTransfer(fromAccountID, toAccountID ID, amount Money, memo string, transferredAt time.Time) (*Transfer, error) {
	from, err := db.BankAccount(fromAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source account: %w", err)
	}
	to, err := db.BankAccount(toAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get destination account: %w", err)
	}

	tx, err := db.conn.Beginx()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	transfer, err := db.insertTransfer(tx, from, to, amount, memo, transferredAt)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return transfer, nil
}

// DeleteTransfer removes a transfer and both of its legs in a single database
// transaction. Unlike creating or editing a transfer, deleting one is allowed
// even when it leaves the destination overdrawn (because its funds were spent
// in the meantime): deleting is how a mistaken transfer is undone.
func (db *DB) DeleteTransfer(id ID) error {
	tx, err := db.conn.Beginx()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	legsQuery, legsArgs := db.sq.Delete("transactions").
		Where("transfer_id = ?", id).
		MustSql()
	if _, err := tx.Exec(legsQuery, legsArgs...); err != nil {
		return fmt.Errorf("failed to delete transfer legs: %w", err)
	}

	headerQuery, headerArgs := db.sq.Delete("transfers").
		Where("id = ?", id).
		MustSql()
	result, err := tx.Exec(headerQuery, headerArgs...)
	if err != nil {
		return fmt.Errorf("failed to delete transfer: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("transfer with id %d not found", id)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// Transfer retrieves a transfer by ID.
func (db *DB) Transfer(id ID) (*Transfer, error) {
	return getTransfer(db.conn, id)
}

// UpdateTransfer edits a transfer's amount, memo, and date, rewriting the
// transfer and both of its legs in a single database transaction. The accounts
// and the direction cannot change; delete the transfer and create a new one
// instead.
//
// An edit may not overdraw either account: raising the amount takes more from
// the source, and lowering it takes some back from the destination. Either
// account whose own balance would end below zero makes the error wrap
// ErrInsufficientFunds.
func (db *DB) UpdateTransfer(id ID, amount Money, memo string, transferredAt time.Time) (*Transfer, error) {
	existing, err := db.Transfer(id)
	if err != nil {
		return nil, err
	}
	if err := validateTransferAmount(amount, existing.Amount.Currency, transferredAt); err != nil {
		return nil, err
	}
	from, err := db.BankAccount(existing.FromAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source account: %w", err)
	}
	to, err := db.BankAccount(existing.ToAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get destination account: %w", err)
	}

	tx, err := db.conn.Beginx()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	delta := amount.Amount - existing.Amount.Amount
	if err := requireNoOverdraft(tx, from, -delta); err != nil {
		return nil, err
	}
	if err := requireNoOverdraft(tx, to, delta); err != nil {
		return nil, err
	}

	at := transferredAt.Round(0)
	headerQuery, headerArgs := db.sq.Update("transfers").
		SetMap(map[string]any{
			"amount":         amount.Amount,
			"memo":           memo,
			"transferred_at": at,
		}).
		Where("id = ?", id).
		Suffix("RETURNING " + transferColumns).
		MustSql()
	var row transferRow
	if err := tx.Get(&row, headerQuery, headerArgs...); err != nil {
		return nil, fmt.Errorf("failed to update transfer: %w", err)
	}

	legs := []struct {
		accountID ID
		amount    int64
	}{
		{from.ID, -amount.Amount},
		{to.ID, amount.Amount},
	}
	for _, leg := range legs {
		legQuery, legArgs := db.sq.Update("transactions").
			SetMap(map[string]any{
				"amount":        leg.amount,
				"memo":          memo,
				"transacted_at": at,
			}).
			Where("transfer_id = ?", id).
			Where("account_id = ?", leg.accountID).
			MustSql()
		result, err := tx.Exec(legQuery, legArgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to update transfer leg: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("failed to get rows affected: %w", err)
		}
		if rows != 1 {
			return nil, fmt.Errorf("transfer %d has %d legs on account %d, expected 1", id, rows, leg.accountID)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	updated := row.toTransfer()
	return &updated, nil
}

// insertTransfer validates a new transfer and writes it with both of its legs
// inside tx. The source leg is inserted first, so a ledger that shows both legs
// lists the money leaving before it arrives.
func (db *DB) insertTransfer(tx *sqlx.Tx, from, to *BankAccount, amount Money, memo string, transferredAt time.Time) (*Transfer, error) {
	if err := validateTransferAccounts(from, to); err != nil {
		return nil, err
	}
	if err := validateTransferAmount(amount, from.Currency, transferredAt); err != nil {
		return nil, err
	}
	if err := requireNoOverdraft(tx, from, -amount.Amount); err != nil {
		return nil, err
	}

	at := transferredAt.Round(0)
	headerQuery, headerArgs := db.sq.Insert("transfers").
		SetMap(map[string]any{
			"from_account_id": from.ID,
			"to_account_id":   to.ID,
			"amount":          amount.Amount,
			"currency":        amount.Currency,
			"memo":            memo,
			"transferred_at":  at,
		}).
		Suffix("RETURNING " + transferColumns).
		MustSql()
	var row transferRow
	if err := tx.Get(&row, headerQuery, headerArgs...); err != nil {
		return nil, fmt.Errorf("failed to insert transfer: %w", err)
	}

	// A leg has no method, payee, or check number: nothing happened at the bank.
	legs := []struct {
		accountID ID
		amount    int64
	}{
		{from.ID, -amount.Amount},
		{to.ID, amount.Amount},
	}
	for _, leg := range legs {
		legQuery, legArgs := db.sq.Insert("transactions").
			SetMap(map[string]any{
				"account_id":    leg.accountID,
				"amount":        leg.amount,
				"memo":          memo,
				"transacted_at": at,
				"transfer_id":   row.ID,
			}).
			MustSql()
		if _, err := tx.Exec(legQuery, legArgs...); err != nil {
			return nil, fmt.Errorf("failed to insert transfer leg: %w", err)
		}
	}

	transfer := row.toTransfer()
	return &transfer, nil
}

// getTransfer retrieves a transfer by ID through q, which may be the database
// or an open transaction.
func getTransfer(q sqlx.Queryer, id ID) (*Transfer, error) {
	var row transferRow
	if err := sqlx.Get(q, &row, "SELECT "+transferColumns+" FROM transfers WHERE id = ?", id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("transfer with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get transfer: %w", err)
	}
	transfer := row.toTransfer()
	return &transfer, nil
}

// requireNoOverdraft returns an error wrapping ErrInsufficientFunds if
// changing account's own balance (excluding its sub-accounts) by delta cents
// would leave it below zero. A non-negative delta never overdraws, so an
// account that is already negative can still receive money.
func requireNoOverdraft(q sqlx.Queryer, account *BankAccount, delta int64) error {
	if delta >= 0 {
		return nil
	}
	var balance int64
	if err := sqlx.Get(q, &balance, "SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE account_id = ?", account.ID); err != nil {
		return fmt.Errorf("failed to calculate balance of %s: %w", account.Name, err)
	}
	if after := balance + delta; after < 0 {
		shortfall := Money{Amount: -after, Currency: account.Currency}
		return fmt.Errorf("%w: this would overdraw %s by %s", ErrInsufficientFunds, account.Name, shortfall)
	}
	return nil
}

// validateTransferAccounts checks that a transfer between from and to is
// allowed: they must be distinct, one must be the direct parent of the other,
// they must share a currency, and both must be active.
func validateTransferAccounts(from, to *BankAccount) error {
	if from.ID == to.ID {
		return fmt.Errorf("cannot transfer from an account to itself")
	}
	toIsChild := to.ParentID != nil && *to.ParentID == from.ID
	fromIsChild := from.ParentID != nil && *from.ParentID == to.ID
	if !toIsChild && !fromIsChild {
		return fmt.Errorf("transfers must be between an account and one of its sub-accounts; %s and %s are not", from.Name, to.Name)
	}
	if from.Currency != to.Currency {
		return fmt.Errorf("account currencies differ: %s is %s, %s is %s", from.Name, from.Currency, to.Name, to.Currency)
	}
	if !from.IsActive {
		return fmt.Errorf("cannot transfer from inactive account %s", from.Name)
	}
	if !to.IsActive {
		return fmt.Errorf("cannot transfer to inactive account %s", to.Name)
	}
	return nil
}

// validateTransferAmount checks that a transfer moves a positive amount in the
// accounts' currency on a specified date.
func validateTransferAmount(amount Money, currency Currency, transferredAt time.Time) error {
	if amount.Amount <= 0 {
		return fmt.Errorf("transfer amount must be positive, got %d cents", amount.Amount)
	}
	if amount.Currency != currency {
		return fmt.Errorf("transfer currency %s does not match account currency %s", amount.Currency, currency)
	}
	if transferredAt.IsZero() {
		return fmt.Errorf("transfer date must be specified")
	}
	return nil
}

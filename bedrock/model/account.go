package model

type Account struct {
	Base
	Type            AccountType  `db:"type"`
	Name            string       `db:"name"`
	Description     string       `db:"description"`
	Denomination    Denomination `db:"denomination"`
	StartingBalance int64        `db:"starting_balance"`
	StartingDate    int64        `db:"starting_date"`
	ParentID        *int64       `db:"parent_id"`
}

type AccountType string

const (
	AccountBank AccountType = "bank"
)

type CreateAccountInput struct {
	Type            AccountType
	Name            string
	Description     string
	Denomination    Denomination
	StartingBalance int64
	StartingDate    int64
	ParentID        *int64
}

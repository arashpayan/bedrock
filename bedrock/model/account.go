package model

import (
	"ara.sh/iabdaccounting/bedrock/datetime"
)

type Account struct {
	Base
	Type            AccountType       `db:"type"`
	Name            string            `db:"name"`
	Description     string            `db:"description"`
	Denomination    Denomination      `db:"denomination"`
	StartingBalance Money             `db:"starting_balance"`
	StartingDate    datetime.DateTime `db:"starting_date"`
	ParentID        *ID               `db:"parent_id"`
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
	StartingBalance Money
	StartingDate    datetime.DateTime
	ParentID        *ID
}

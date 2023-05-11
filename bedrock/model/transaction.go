package model

import "ara.sh/iabdaccounting/bedrock/datetime"

type CreateTransactionInput struct {
	Date        datetime.DateTime
	Amount      int64
	Memo        string
	CheckNumber string
	PayeeID     *int64
	DepositID   *int64
	// Categories
}

type Transaction struct {
	Base
	TransactedAt datetime.DateTime
	Amount       int64
	Memo         string
	CheckNumber  string
	PayeeID      *int64
	DepositID    *int64
}

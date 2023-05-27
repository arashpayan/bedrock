package model

import "ara.sh/iabdaccounting/bedrock/datetime"

type CreateDepositTransactionInput struct {
	AccountID    ID
	Amount       Money
	DepositID    ID
	Memo         string
	Method       *TransactionMethod
	TransactedAt datetime.DateTime
}

type CreateWithdrawalTransactionInput struct {
	AccountID    ID
	Amount       Money
	CheckNumber  string
	Memo         string
	Method       *TransactionMethod
	PayeeID      ID
	TransactedAt datetime.DateTime
}

type Transaction struct {
	Base
	AccountID    ID                 `db:"account_id"`
	Amount       Money              `db:"amount"`
	CheckNumber  *string            `db:"check_number"`
	DepositID    *ID                `db:"deposit_id"`
	Memo         string             `db:"memo"`
	Method       *TransactionMethod `db:"method"`
	PayeeID      *ID                `db:"payee_id"`
	TransactedAt datetime.DateTime  `db:"transacted_at"`
}

type TransactionMethod string

const (
	ATM                TransactionMethod = "atm"
	AutoPay            TransactionMethod = "auto-pay"
	ElectronicTransfer TransactionMethod = "electronic-transfer"
	Check              TransactionMethod = "check"
)

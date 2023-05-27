package model

import "ara.sh/iabdaccounting/bedrock/datetime"

type Deposit struct {
	Base
	AccountID ID `db:"account_id"`
}

type CreateDepositInput struct {
	DepositedAt           datetime.DateTime
	AccountID             ID
	Memo                  string
	UndepositedReceiptIDs []ID
}

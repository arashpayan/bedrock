package model

import "ara.sh/iabdaccounting/bedrock/datetime"

type Receipt struct {
	Base
	HumanID    string            `db:"human_id"`
	CustomerID ID                `db:"customer_id"`
	SoldAt     datetime.DateTime `db:"sold_at"`
	Total      Money             `db:"total"`
}

type ReceiptItem struct {
	Base

	ReceiptID   ID     `db:"receipt_id"`
	ItemID      ID     `db:"item_id"`
	Description string `db:"description"`
	Price       Money  `db:"amount"`
}

type CreateReceiptItem struct {
	ItemID      ID
	Description string
	Price       Money
}

type CreateReceiptInput struct {
	CustomerID ID
	SoldAt     datetime.DateTime

	Items []CreateReceiptItem
}

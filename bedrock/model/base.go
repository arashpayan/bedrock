package model

import "ara.sh/iabdaccounting/bedrock/datetime"

type ID int64

type Base struct {
	ID         ID                `db:"id" json:"id"`
	CreatedAt  datetime.DateTime `db:"created_at" json:"created_at"`
	ModifiedAt datetime.DateTime `db:"modified_at" json:"modified_at"`
}

package model

import "ara.sh/iabdaccounting/bedrock/datetime"

type Base struct {
	ID         int64             `db:"id" json:"id"`
	CreatedAt  datetime.DateTime `db:"created_at" json:"created_at"`
	ModifiedAt datetime.DateTime `db:"modified_at" json:"modified_at"`
}

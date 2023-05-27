package model

type Party struct {
	Base

	Name            string  `db:"name"`
	EmailAddress    *string `db:"email_address"`
	BahaiIDNumber   *string `db:"bahai_id_number"`
	Address         *string `db:"address"`
	TelephoneNumber *string `db:"telephone_number"`
}

type CreatePartyInput struct {
	Name            string
	EmailAddress    *string
	BahaiIDNumber   *string
	Address         *string
	TelephoneNumber *string
}

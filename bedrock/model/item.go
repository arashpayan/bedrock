package model

type Item struct {
	Base

	Name     string `db:"name"`
	Shortcut string `db:"shortcut"`
}

type CreateItemInput struct {
	Name     string
	Shortcut string
}

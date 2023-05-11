package model

type CategoryType string

const (
	CategoryExpense CategoryType = "expense"
	CategoryIncome  CategoryType = "income"
)

type Category struct {
	Type        CategoryType
	Name        string
	Description string
	ParentID    *int64
}

type CategoryEntry struct {
	TransactionID int64
}

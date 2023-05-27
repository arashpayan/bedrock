package persistence

import (
	"context"
	"fmt"
	"time"

	"ara.sh/iabdaccounting/bedrock/model"
	sq "github.com/Masterminds/squirrel"
)

type assemblyKey string

const (
	assemblyKeyName     assemblyKey = "name"
	assemblyKeyTimezone assemblyKey = "timezone"
)

func (db *Database) AssemblyInfo(ctx context.Context) (*model.Assembly, error) {
	tx := db.dbx.MustBeginTx(ctx, nil)
	defer tx.Rollback()

	kvs := []struct {
		Key   assemblyKey `db:"key_name"`
		Value string      `db:"value"`
	}{}
	query, args := sq.Select("*").From(tableAssemblyInfo).MustSql()
	if err := tx.Select(&kvs, query, args...); err != nil {
		return nil, err
	}

	a := model.Assembly{}
	for _, kv := range kvs {
		switch kv.Key {
		case assemblyKeyName:
			a.Name = kv.Value
		case assemblyKeyTimezone:
			tz, err := time.LoadLocation(kv.Value)
			if err != nil {
				return nil, fmt.Errorf("parsing location: %v", err)
			}
			a.Timezone = *tz
		}
	}

	return &a, nil
}

func (db *Database) SetAssemblyInfo(ctx context.Context, asbly model.Assembly) error {
	tx := db.dbx.MustBeginTx(ctx, nil)
	defer tx.Rollback()

	insertKV := func(key assemblyKey, value string) error {
		_, err := sq.Insert(tableAssemblyInfo).SetMap(map[string]any{
			"key_name": key,
			"value":    value,
		}).RunWith(tx.Tx).Exec()
		return err
	}

	if err := insertKV(assemblyKeyName, asbly.Name); err != nil {
		return err
	}
	if err := insertKV(assemblyKeyTimezone, asbly.Timezone.String()); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	db.Assembly = asbly

	return nil
}

/*
assembly, err := bedrock.OpenAssembly("/home/arash/Documents/to.fund")

type Currency string // USD | CAD

type Money struct {
	Currency Currency
	Amount int64
}

type Account struct {
	Type AccountType // AccountBank | AccountExpense | AccountIncome
	Name string
	Denomination Currency
	StartingBalance Money
}

assembly.CreateItem({Name: "Local Bahá'í Fund", Shortcut:"LBF"})
assembly.CreateItem({Name: "Earmark - Shrine of Abdulbaha", Shortcut: "SOA"})
assembly.CreateItem({Name: "Earmark - International Bahá'í Fund", Shortcut: "IBF"})

acct, err := assembly.AddBankAccount(Account{Bank: "Bank of America", Number: "3991890419801", StartingBalance: "3940", Name: "Local Bahá'í Fund"})

hfAcct, err := acct.AddSubAccount(Account{Name: "Humanitarian Fund", StartingBalance: "100"})
emIBF, err := acct.AddSubAccount(Account{Name: "Earmarks - International Bahá'í Fund", StartingBalance:""})
emSOA, err := acct.AddSubAccount(Account{Name: "Earmarks - Shrine of Abdulbaha", StartingBalance: ""})

timmy, err := assembly.CreatePerson("Little Timmy")

income, err := assembly.RecordIncome(Income{
	From: timmy,
	To: Undeposited,
	Date: "2023-05-01",
	Amount: "$30",
	Type: Contribution|LoanRepayment,
	Items: []Item{
		{ID: LBFID, Amount: "$10"},
		{ID: EarmarkIBFID, Amount: "$10"},
		{ID: EarmarkSOAID, Amount: "$10"},
	}
	Form: CHECK|CASH|ACH|DWOLLA,
	CheckNumber: ptr.Of("381"),
})

acct.CreateTransaction(Transaction{
	Date: "2023-05-02",
	Sources: []IncomeIDs{income.ID},
})

administrative, err := assembly.CreateAccount("Expenses:Administrative")
holyDays, err := assembly.CreateAccount("Expenses:Holy Days")

acct.WriteCheck(Check{
	Date: "2023-05-03",
	Number: "",
	Account: acct.ID,
	Amount: "$40",
	Memo: "Reimbursement for mailers and holy day supplies",
	ExpenseAccounts: []AccountAmount{
		{
			ID: Administrative.ID,
			Amount: "$35"
		}
		{
			ID: HolyDays.ID,
			Amount: "$5"
		},
	}
})

acct.CreateTransaction(Transaction{
	Date: "2023-05-03",
	Sources: []
})

tx, err := assembly.CreateTransaction(Transaction{
	From: timmy,
	To: acct,
	Amount: "$30",
})



acct.CreateTransaction(Transaction{Amount: "$30",

*/

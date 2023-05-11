package bedrock

/*
type Currency string // USD | CAD

type DateTime int64

type Account struct {
	Type AccountType // AccountBank | AccountNonCash
	Name string
	Denomination Currency
	StartingBalance int64
	StartingDate
}

type Category struct {
	Type CategoryType // CategoryIncome | CategoryExpense
	Name string
	Description string
	ParentID ID	// another category of the same type
}

type CategoryEntry struct {
	TransactionID ID!
}

type Transaction struct {
	Date DateTime!
	Method MethodType // Check | ACH | ATM | Teller
	Amount int64!	// signed
	Memo string!
	PayeeID ID	// for debits
	CheckNumber // for debits
	DepositID ID
	Categories []Category
}

// Bank Account and sub accounts
checkingAcct, _ := assembly.CreateAccount(Account{
	Type: AccountBank,
	Name: "Checking Account",
	Denomination: USD,
	StartingBalance: 300000, // $3000
	Description string
})
ucAcct, _ := assembly.CreateAccount(Account{
	Type: AccountBank,
	Name: "Checking Account:Unit Convention",
	Parent: &checkingAccount.ID
})

nonCashAcct, _ := assembly.CreateAccount(Account{
	Type: AccountNonCash,
	Name: "NonCash",
	Denomination: USD,
})

// Expense Accounts
expenseAcct, _ := assembly.CreateAccount(Account{
	Type: AccountExpense,
	Name: "Expense",
	Denomination: USD,
})
assembly.CreateAccount(Account{
	Type: AccountExpense,
	Name: "Expense:Holy Days",
	Denomination: USD,
	Parent: &expenseAccount.ID,
})
assembly.CreateAccount(Account{
	Type: AccountExpense,
	Name: "Expense:Administrative",
	Denomination: USD,
	Parent: &expenseAccount.ID,
})
assembly.CreateAccount(Account{
	Type: AccountExpense,
	Name: "Expense:Holy Days",
	Denomination: USD,
	Parent: &expenseAccount.ID,
})

assembly.CreateCategory(Category{
	Name: "Expense",
	Denomination: USD,
})int64

// Income Accounts
incomeAcct, err := assembly.CreateAccount(Account{
	Type: AccountIncome,
	Name: "Income",
	Denomination: USD,
})
incomeLBF, _ := assembly.CreateAccount(Account{
	Type: AccountIncome
	Name: "Income:Local Bahá'í Fund",
	Denomination: USD,
	Parent: &incomeAcct.ID,
})

timmy, _ := assembly.CreatePerson("Little Timmy", "little.timmy@gmail.com", "214657")
nbf, _ := assembly.CreatePerson("National Bahá'í Fund")
lbfItem, _ := assembly.CreateItem({"Local Bahá'í Fund", "LBF"})
emSOAItem, _ := assembly.CreateItem({"Earmark - Shrine of Abdulbaha", "SOA"})
emIBFItem, _ := assembly.CreateItem({"Earmark - International Bahá'í Fund", "IBF"})

timmyRcpt, _ := assembly.CreateContribution(Contribution{
	For: timmy,
	Receipt: // will autogenerate "YYYYMMDDHHMMSS.Nanoseconds" in UTC,
	Date: "2023-05-01",
	Sold: []ItemSale{
		{
			ItemID: lbfItem.ID,
			Amount: $19,
		},
		{
			ItemID: smSOAItem.ID,
			Amount: $29
		},
		{
			ItemID: emIBFItem.ID,
			Amount: $39,
		},
	}
	// Total: $87,	// derived then persisted
})
deposit, _ := assembly.Deposit(CreateDeposit{
	To: checkingAcct.ID,
	Date: "2023-05-02",
	Sources: []ReceiptIDs{
		timmyRcpt.ID,
	},
	Account: checkingAcct.ID,
})
// creates a transaction in the account
checkingAcct.CreateTransaction(Transaction{
	Date: "2023-05-02",	// date from the deposit
	Deposit: &deposit.ID,	// just used for provenance
	Amount: $87,
	Memo: "Deposit to bank"
})


checkingAcct.Debit(Debit{
	Payee: nbf,
	CheckNumber: ptr.Of("179"),
	Amount: -$100,
	Date: "2023-05-04",
	Memo: "Monthly contribution",
	Categories: []CategoryAmount{
		{ ID: "Expense:National Bahá'í Fund", Amount: $50 }
		{ ID: "Expense:SOA", Amount: $50 }
	}
})


check, err := checkingAcct.WriteCheck(Check{
	Payee: nbf,
	Number: "179"
	Amount: $100
	Date: "2023-05-04",
	Memo: "Monthly contribution",
	Account: checkingAcct.ID,
})

*/

# Bedrock Library - Financial Management for Spiritual Assemblies

## Project Overview
Bedrock is a Go library that provides simplified QuickBooks-like functionality specifically designed for Spiritual Assembly organizations. It handles monetary and in-kind contributions, vendor payments, and bank account management with SQLite persistence.

## Code Style Guidelines
**All new code must follow idiomatic Go practices and the [Google Go Style Guide](https://google.github.io/styleguide/go/) unless there is a strong reason to deviate (e.g., performance-sensitive code).** Key principles:
- Use clear, descriptive names without unnecessary prefixes (e.g., `Item()` not `GetItem()`)
- Follow Go naming conventions for exported vs unexported identifiers
- Prefer composition over inheritance
- Handle errors explicitly
- Use `gofumpt` for consistent formatting (a stricter superset of `gofmt`). Run `gofumpt -w .` before committing; do not use plain `gofmt`, which will leave formatting that `gofumpt` rejects.
- Write clear, concise documentation for all exported functions
- Prefer `max()`/`min()` builtins over manual if-then clamping patterns
- Prefer `for i := range N` over `for i := 0; i < N; i++`

### File Structure
Functions, types, and methods should be organized alphabetically within files to improve discoverability and maintainability.

**Ordering Rules:**
1. **Type definitions** come first, in alphabetical order
2. **Constructor functions** for a type come immediately after the type definition (in alphabetical order)
3. **Methods** of a type come after constructors (in alphabetical order)
4. **Package-level functions** come after all types and their methods (in alphabetical order)

**Example:**
```go
type Cat struct {
    Name string
    Age  int
}

// Constructors come after type definition, alphabetically
func catFromProto(pb *proto.Cat) Cat {
    return Cat{Name: pb.Name, Age: int(pb.Age)}
}

func newCat(name string, age int) Cat {
    return Cat{Name: name, Age: age}
}

// Methods come after constructors, alphabetically
func (c Cat) Greet() string {
    return "Meow, I'm " + c.Name
}

func (c Cat) Speak() string {
    return "Meow!"
}

type Dog struct {
    Name string
}

// Dog's methods immediately follow Dog type
func (d Dog) Jump() {
    // implementation
}

func (d Dog) String() string {
    return d.Name
}

// Package-level functions come after all types, alphabetically
func AnimalFunction() {
    // implementation
}

func SomeOtherFunction() {
    // implementation
}
```

**Benefits:**
- Easy to locate functions and methods
- Reduces merge conflicts
- Consistent across the codebase
- No arbitrary or chronological ordering

## Key Design Decisions
- Each `.bedrock` file contains data for a single Assembly (no cross-Assembly foreign keys needed)
- Uses SQLite for persistence with file-based databases
- Hierarchical bank accounts support earmarked funds through parent/child relationships
- Type-safe account types (checking, savings, earmark)
- Earmark accounts must have a checking or savings parent (cannot be top-level or nested under another earmark)
- **Only earmark accounts can have a parent** (checking/savings accounts are always top-level)
- **Sub-accounts must inherit currency from their parent** (enforced in `CreateBankAccount`)
- **Opening balance support**: Accounts can be created with an initial balance, which creates a deposit transaction with memo "Opening Balance"
- Integer-based monetary amounts to avoid floating-point precision issues

## Dependencies
- `github.com/jmoiron/sqlx` - Database operations with reduced boilerplate
- `github.com/Masterminds/squirrel` - SQL query building with SetMap() and MustSql()
- `modernc.org/sqlite` - Pure Go SQLite driver (no CGO required)
- `github.com/stretchr/testify` - Enhanced testing assertions and test utilities

## Core Types Implemented

### Financial Types
- `ID` - int64 unique identifier
- `Base` - embedded struct with ID, CreatedAt, ModifiedAt for all entities
- `Currency` - type-safe constants (USD, CAD)
- `Money` - integer-based monetary amounts with currency (stored as cents to avoid floating-point precision issues)

### Organization & Accounts
- `Assembly` - represents the organization (Name, Timezone, DefaultCurrency)
- `AccountType` - type-safe constants (checking, savings, earmark)
- `BankAccount` - supports hierarchical structure with ParentID for sub-accounts and currency per account

### Contribution Management
- `Item` - represents contribution categories/funds (e.g., "Local Fund", "Humanitarian Fund")
- `Receipt` - issued for contributions with HumanID (auto-generated), CustomerID (Party FK), SoldAt timestamp, and TransactionID foreign key
- `ReceiptItem` - line items on receipts linking to Items with Price (Money type)

### Expense Management
- `Category` - represents expense categories for withdrawal transactions (e.g., "Office Supplies", "Food", "Travel")
- `Expense` - line items for withdrawal transactions linking to Categories with Amount (Money type) and optional Description
- `ExpenseItem` - input type for creating withdrawals with expense breakdowns

### Ledger and Reporting
- `LedgerEntry` - represents a single row in a ledger view with transaction data and running balance
- `LedgerOptions` - provides filtering and pagination options for ledger queries

### Reconciliation
- `ReconciliationStatus` - type-safe constants (in-progress, completed). A reconciliation is either in-flight or done; cancel and undo delete the record entirely.
- `Reconciliation` - represents a bank account reconciliation session with statement date, balance, and status

### People & Entities
- `Party` - represents both contributors and vendors with optional contact information (email, Bahá'í ID, address, phone)

### Transactions
- `TransactionMethod` - type-safe constants (ATM, auto-pay, electronic-transfer, in-branch, check)
- `Transaction` - unified deposits and withdrawals with amount (positive/negative), account, method, payee, etc.

### Database Connection
- `DB` - database connection with sqlx.DB and squirrel builder

## Key Relationships
- **Assembly**: One per database file (no foreign keys needed)
- **BankAccount**: Hierarchical (parent/child) via ParentID for sub-accounts and earmarking
- **Item**: Independent contribution categories
- **Category**: Independent expense categories
- **Party**: Independent entities (contributors and vendors)
- **Receipt → Party**: Many-to-One (receipts belong to customers)
- **Receipt → Transaction**: Many-to-One (each receipt belongs to exactly one deposit transaction, or none if not yet deposited)
- **Receipt → ReceiptItem**: One-to-Many (receipts contain multiple line items)
- **ReceiptItem → Item**: Many-to-One (items can be referenced by multiple receipt items)
- **Transaction → BankAccount**: Many-to-One (transactions belong to specific accounts)
- **Transaction → Party**: Many-to-One (withdrawal transactions have payees)
- **Transaction → Expense**: One-to-Many (withdrawal transactions contain multiple expense line items)
- **Transaction → Reconciliation**: Many-to-One (cleared transactions belong to a reconciliation)
- **Expense → Category**: Many-to-One (expenses belong to specific categories)
- **Reconciliation → BankAccount**: Many-to-One (reconciliations belong to root-level accounts only)

## Database Schema
Schema is defined in `schema.sql` and embedded into the Go binary using `//go:embed`. Complete schema includes:
- `assembly` table - single row per database file with name, timezone, and default_currency
- `bank_accounts` table - hierarchical accounts with parent_id foreign key, currency, and account type
- `items` table - contribution categories/funds
- `categories` table - expense categories for withdrawal transactions
- `parties` table - contributors and vendors with optional contact fields (email, Bahá'í ID, address, phone)
- `receipts` table - contribution receipts with customer_id (Party FK) and transaction_id foreign key (nullable)
- `receipt_items` table - line items with receipt_id and item_id foreign keys, price and currency
- `reconciliations` table - reconciliation sessions with account_id, statement date/balance, status, and completion timestamp
- `transactions` table - all deposits and withdrawals with account_id, optional payee_id, and optional reconciliation_id foreign keys
- `expenses` table - expense line items with transaction_id and category_id foreign keys, amount and optional description
- Automatic timestamp triggers for modified_at updates on all tables
- Foreign key constraints enabled for data integrity

## API Usage Patterns

### Creating a New Bedrock Database
```go
// Create a new .bedrock database with Assembly initialization
eastern, _ := time.LoadLocation("America/New_York")
db, err := bedrock.New("/path/to/finances.bedrock", "Local Spiritual Assembly of New York", eastern, bedrock.CurrencyUSD)
if err != nil {
    // handle error
}
defer db.Close()

// Assembly is automatically created and ready to use
assembly, err := db.Assembly()
```

### Opening an Existing Bedrock Database
```go
db, err := bedrock.Open("/path/to/finances.bedrock")
if err != nil {
    // handle error
}
defer db.Close()

// All operations are methods on the db variable
```

## Current Status
✅ Basic types and structures defined
✅ Financial types (Money, Currency) with integer precision
✅ Contribution workflow types (Item, Receipt, ReceiptItem, removed redundant Deposit type)
✅ Database initialization with schema creation from embedded SQL file
✅ Open/Close functionality implemented
✅ Proper one-to-many relationships established
✅ **Complete database schema implemented** in `schema.sql` with all required tables and triggers
✅ **CRUD Operations Completed:**
  - **Assembly Operations**: Assembly, UpdateAssembly (creation only via `bedrock.New()`, one assembly per database)
  - **Item CRUD**: CreateItem, Item, ItemByName, ListItems, UpdateItem, DeleteItem
  - **Category CRUD**: CreateCategory, Category, CategoryByName, Categories, UpdateCategory, DeleteCategory
  - **Party CRUD**: CreateParty, Party, PartyByName, PartyByEmail, PartyByBahaiID, ListParties, SearchParties, UpdateParty, DeleteParty
  - **Receipt CRUD**: CreateReceipt (auto HumanID with current time), Receipt, ReceiptByHumanID, ReceiptsByCustomer, ReceiptsByTransaction, UndepositedReceipts, Receipts, AssignReceiptToTransaction, UnassignReceiptFromTransaction, DeleteReceipt
  - **ReceiptItem CRUD**: CreateReceiptItem, ReceiptItem, ReceiptItems, UpdateReceiptItem, DeleteReceiptItem
  - **BankAccount CRUD**: CreateBankAccount, BankAccount, BankAccountByName, RootBankAccounts, ChildBankAccounts, BankAccounts, ActiveBankAccounts, UpdateBankAccount, DeactivateBankAccount, DeleteBankAccount
  - **Transaction CRUD**: CreateDeposit, CreateWithdrawal (with expense categorization and currency validation)
  - **Ledger Operations**: AccountLedger, AccountBalance, AccountBalanceAsOf, TransactionsForAccount, AccountTransactionCount, AllAccountBalances, LastTransactionDate
  - **Reconciliation Operations**: StartReconciliation, Reconciliation, Reconciliations, InProgressReconciliation, LastCompletedReconciliation, ClearTransaction, UnclearTransaction, ClearedTransactions, ClearedBalance, UnclearedTransactions, CompleteReconciliation, CancelReconciliation, UndoReconciliation

## Advanced Features Implemented
- **Database Initialization**: `bedrock.New()` creates new databases with automatic Assembly setup and timezone configuration
- **Timezone-Aware Receipt IDs**: HumanID automatically generated using current system time in Assembly timezone format `20060102150405.000` (ensures uniqueness)
- **Cycle Prevention**: BankAccount parent relationships prevent circular references with comprehensive validation
- **Currency Inheritance**: Sub-accounts must use the same currency as their parent account (validated in `CreateBankAccount`)
- **Currency Validation**: Transactions validated against account currency to prevent currency mismatches
- **Expense Categorization**: Withdrawal transactions require one or more expense items with automatic total calculation and validation
- **Ledger Views**: Complete account ledgers with running balances, hierarchical account support, and enriched transaction data
- **Balance Calculations**: Current and historical balance calculations with subaccount aggregation
- **Referential Integrity**: Prevents deletion of entities with dependencies (accounts with children/transactions, receipts with items, categories with expenses)
- **Single Assembly Per Database**: Each `.bedrock` file contains exactly one Assembly with proper validation
- **Workflow Support**: Undeposited receipts tracking, transaction assignment, hierarchical account queries, expense breakdowns
- **Bank Reconciliation**: Complete reconciliation workflow with transaction clearing, balance verification, undo support, and historical tracking
- **Performance Optimization**: Efficient SQL queries with pagination, date filtering, and recursive account tree traversal
- **Idiomatic Go API**: All method names follow Go conventions (e.g., `Item()` instead of `GetItem()`) per Google Go Style Guide
- **Embedded Schema**: Database schema stored in `schema.sql` and embedded using `//go:embed` for clean separation of concerns

## Contribution Workflow
1. **Create Items** - Define contribution categories (Local Fund, Humanitarian Fund, etc.)
2. **Create Parties** - Add contributors and vendors to the system
3. **Issue Receipts** - When contributions are received, create Receipt with ReceiptItems
4. **Make Deposits** - Group receipts together and create deposit Transaction when money is banked
5. **Track Relationships** - Each receipt can only belong to one deposit transaction (immutable once assigned)

## Expense Workflow
1. **Create Categories** - Define expense categories (Office Supplies, Food, Travel, etc.)
2. **Create Parties** - Add vendors and payees to the system
3. **Make Withdrawals** - Create withdrawal transactions with one or more expense items categorizing the spending
4. **Automatic Validation** - System ensures all expenses use consistent currency and sum to the withdrawal total

## Reconciliation Workflow
Reconciliation matches transactions in Bedrock with bank statements to guard against bookkeeping mistakes and fraudulent activity.

1. **Start Reconciliation** - Begin a reconciliation session for a root-level account with the statement date and ending balance
2. **Clear Transactions** - Mark transactions that appear on the bank statement as cleared
3. **Verify Balance** - The system validates that cleared transactions sum to the statement balance
4. **Complete or Cancel** - Finalize the reconciliation when balanced, or cancel to abandon. **Cancel deletes the reconciliation record entirely**, unclears its transactions, and leaves no trace — a cancelled reconciliation is indistinguishable from one that never happened.
5. **Undo if Needed** - Completed reconciliations can be undone if a mistake is found. **Undo also deletes the reconciliation record**; it is not marked cancelled. Only the most recent completed reconciliation per account may be undone, to preserve the integrity of older history.

**Key Constraints:**
- Reconciliation is only allowed for root-level accounts (not subaccounts)
- Transactions from subaccounts can be cleared against the parent account's reconciliation
- Only one in-progress reconciliation per account at a time
- Transactions dated after the statement date cannot be cleared
- Only the most recent completed reconciliation can be undone
- Cancel and undo both delete the record and unclear its transactions in a single transaction. `CancelReconciliation` returns only `error`; the record is gone on success.

## Testing
✅ **Comprehensive test suite implemented** using `github.com/stretchr/testify` for enhanced assertions
- **180+ test cases** covering all CRUD operations and edge cases
- **Database isolation**: Each test uses temporary databases for clean state
- **Error case testing**: Validation errors, constraint violations, referential integrity
- **Workflow testing**: Receipt-to-transaction assignment, hierarchical accounts, currency validation, expense categorization
- **Expense validation testing**: Currency mismatches, mixed currencies, zero amounts, foreign key constraints
- **Ledger functionality testing**: Running balances, hierarchical account aggregation, date filtering, pagination
- **Reconciliation testing**: Start/complete/cancel/undo workflows, transaction clearing, balance verification, constraint validation
- **Type safety testing**: Money type precision, currency handling, string formatting

## Usage Examples

### Creating Expense Categories and Withdrawals
```go
// Create expense categories
officeCategory, err := db.CreateCategory("Office Supplies")
foodCategory, err := db.CreateCategory("Food & Refreshments")

// Create withdrawal with multiple expense categories
expenses := []ExpenseItem{
    {
        CategoryID:  officeCategory.ID,
        Description: &[]string{"Printer paper and pens"}[0],
        Amount:      NewMoney(2000, CurrencyUSD), // $20.00
    },
    {
        CategoryID:  foodCategory.ID,
        Description: &[]string{"Meeting refreshments"}[0],
        Amount:      NewMoney(8000, CurrencyUSD), // $80.00
    },
}

transaction, err := db.CreateWithdrawal(
    accountID, payeeID, TransactionMethodCheck, 
    "Office supplies and meeting food", time.Now(), 
    &checkNumber, expenses,
)
// Total withdrawal: $100.00 automatically calculated
```

### Ledger Views and Balance Calculations
```go
// Get current account balance
balance, err := db.AccountBalance(accountID, true) // include subaccounts
fmt.Printf("Current balance: %s\n", balance.String())

// Get account ledger with running balances
ledger, err := db.AccountLedger(accountID, &LedgerOptions{
    IncludeSubaccounts: true,
    Limit:              &[]int{50}[0], // Last 50 transactions
})

// Display ledger entries
for _, entry := range ledger {
    fmt.Printf("%s | %s | %s | %s\n", 
        entry.Transaction.TransactedAt.Format("2006-01-02"),
        entry.Transaction.Memo,
        entry.Transaction.Amount,
        entry.RunningBalance.String())
}

// Get balance as of specific date
asOfDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
yearEndBalance, err := db.AccountBalanceAsOf(accountID, asOfDate, true)

// Get all account balances for financial statements
allBalances, err := db.AllAccountBalances()
for accountID, balance := range allBalances {
    fmt.Printf("Account %d: %s\n", accountID, balance.String())
}
```

### Bank Account Reconciliation
```go
// Start a reconciliation session with the bank statement details
statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
statementBalance := NewMoney(150000, CurrencyUSD) // $1,500.00

reconciliation, err := db.StartReconciliation(accountID, statementDate, statementBalance)
if err != nil {
    // handle error (e.g., account is a subaccount, or reconciliation already in progress)
}

// Get uncleared transactions up to the statement date
uncleared, err := db.UnclearedTransactions(accountID, statementDate)
for _, tx := range uncleared {
    fmt.Printf("%s | %s | %d cents\n", 
        tx.TransactedAt.Format("2006-01-02"), tx.Memo, tx.Amount)
}

// Clear transactions that appear on the bank statement
for _, tx := range transactionsOnStatement {
    err := db.ClearTransaction(reconciliation.ID, tx.ID)
    if err != nil {
        // handle error
    }
}

// Check current cleared balance
clearedBalance, err := db.ClearedBalance(reconciliation.ID)
fmt.Printf("Cleared balance: %s (statement: %s)\n", 
    clearedBalance.String(), statementBalance.String())

// Complete the reconciliation (validates balance matches)
completed, err := db.CompleteReconciliation(reconciliation.ID)
if err != nil {
    // Balance mismatch - investigate discrepancy
    fmt.Printf("Reconciliation failed: %v\n", err)
    // Cancel deletes the in-progress record and unclears its transactions:
    //   _ = db.CancelReconciliation(reconciliation.ID)
}

// If a mistake is discovered, undo the most recent reconciliation.
// Undo deletes the record — the reconciliation is gone, not merely marked cancelled.
err = db.UndoReconciliation(completed.ID)
```

### Creating Bank Accounts with Opening Balances
```go
// Create a checking account with $1,500.00 opening balance
openingBalance := NewMoney(150000, CurrencyUSD) // Amount in cents
account, err := db.CreateBankAccount(
    "Main Checking",
    AccountTypeChecking,
    CurrencyUSD,
    nil,           // no parent (top-level account)
    "Primary checking account",
    true,          // isActive
    openingBalance,
)
// This creates the account AND a deposit transaction with memo "Opening Balance"

// Create an account with no opening balance
account, err := db.CreateBankAccount(
    "Savings",
    AccountTypeSavings,
    CurrencyUSD,
    nil,
    "",
    true,
    Money{}, // Zero balance - no transaction created
)

// Earmark accounts can also have opening balances
// They inherit currency from their parent
earmark, err := db.CreateBankAccount(
    "Emergency Fund",
    AccountTypeEarmark,
    parent.Currency, // Must match parent
    &parent.ID,
    "",
    true,
    NewMoney(50000, parent.Currency), // $500.00
)
```

## Next Steps (Suggested)
- Expense querying and reporting methods (ExpensesByCategory, ExpensesByTransaction, etc.)
- Transaction querying and reporting methods
- Account balance calculations with expense breakdowns
- Reporting functionality (contribution summaries, expense reports, financial statements)
- Budget tracking and variance analysis

## Testing Commands
- `go mod tidy` - Clean up dependencies
- `go build` - Build the library
- `go test` - Run comprehensive test suite (all tests passing)
- `go test -v` - Run tests with verbose output
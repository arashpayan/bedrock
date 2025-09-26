# Bedrock Library - Financial Management for Spiritual Assemblies

## Project Overview
Bedrock is a Go library that provides simplified QuickBooks-like functionality specifically designed for Spiritual Assembly organizations. It handles monetary and in-kind contributions, vendor payments, and bank account management with SQLite persistence.

## Code Style Guidelines
**All new code must follow idiomatic Go practices and the [Google Go Style Guide](https://google.github.io/styleguide/go/) unless there is a strong reason to deviate (e.g., performance-sensitive code).** Key principles:
- Use clear, descriptive names without unnecessary prefixes (e.g., `Item()` not `GetItem()`)
- Follow Go naming conventions for exported vs unexported identifiers
- Prefer composition over inheritance
- Handle errors explicitly
- Use `go fmt` for consistent formatting
- Write clear, concise documentation for all exported functions

## Key Design Decisions
- Each `.bedrock` file contains data for a single Assembly (no cross-Assembly foreign keys needed)
- Uses SQLite for persistence with file-based databases
- Hierarchical bank accounts support earmarked funds through parent/child relationships
- Type-safe account types (checking, savings, earmark)
- Integer-based monetary amounts to avoid floating-point precision issues

## Dependencies
- `github.com/jmoiron/sqlx` - Database operations with reduced boilerplate
- `github.com/Masterminds/squirrel` - SQL query building with SetMap() and MustSql()
- `github.com/mattn/go-sqlite3` - SQLite driver

## Core Types Implemented

### Financial Types
- `ID` - int64 unique identifier
- `Base` - embedded struct with ID, CreatedAt, ModifiedAt for all entities
- `Currency` - type-safe constants (USD, CAD)
- `Money` - integer-based monetary amounts with currency (stored as cents to avoid floating-point precision issues)

### Organization & Accounts
- `Assembly` - represents the organization (Name, Timezone)
- `AccountType` - type-safe constants (checking, savings, earmark)
- `BankAccount` - supports hierarchical structure with ParentID for sub-accounts and currency per account

### Contribution Management
- `Item` - represents contribution categories/funds (e.g., "Local Fund", "Humanitarian Fund")
- `Receipt` - issued for contributions with HumanID (auto-generated), CustomerID (Party FK), SoldAt timestamp, and TransactionID foreign key
- `ReceiptItem` - line items on receipts linking to Items with Price (Money type)

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
- **Party**: Independent entities (contributors and vendors)
- **Receipt → Party**: Many-to-One (receipts belong to customers)
- **Receipt → Transaction**: Many-to-One (each receipt belongs to exactly one deposit transaction, or none if not yet deposited)
- **Receipt → ReceiptItem**: One-to-Many (receipts contain multiple line items)
- **ReceiptItem → Item**: Many-to-One (items can be referenced by multiple receipt items)
- **Transaction → BankAccount**: Many-to-One (transactions belong to specific accounts)
- **Transaction → Party**: Many-to-One (withdrawal transactions have payees)

## Database Schema (Planned)
- `assembly` table - single row per database file
- `bank_accounts` table - hierarchical accounts with parent_id foreign key and currency
- `items` table - contribution categories/funds
- `parties` table - contributors and vendors with optional contact fields
- `receipts` table - contribution receipts with customer_id (Party FK) and transaction_id foreign key (nullable)
- `receipt_items` table - line items with receipt_id and item_id foreign keys
- `transactions` table - all deposits and withdrawals with account_id and optional payee_id foreign keys
- Automatic timestamp triggers for modified_at updates

## API Usage Pattern
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
✅ Database initialization with schema creation
✅ Open/Close functionality implemented
✅ Proper one-to-many relationships established
✅ **CRUD Operations Completed:**
  - **Item CRUD**: CreateItem, Item, ItemByName, Items, UpdateItem, DeleteItem
  - **Party CRUD**: CreateParty, Party, PartyByName, PartyByEmail, PartyByBahaiID, Parties, SearchParties, UpdateParty, DeleteParty
  - **Receipt CRUD**: CreateReceipt (auto HumanID), Receipt, ReceiptByHumanID, ReceiptsByCustomer, ReceiptsByTransaction, UndepositedReceipts, Receipts, AssignReceiptToTransaction, UnassignReceiptFromTransaction, DeleteReceipt
  - **BankAccount CRUD**: CreateBankAccount, BankAccount, BankAccountByName, RootBankAccounts, ChildBankAccounts, BankAccounts, ActiveBankAccounts, UpdateBankAccount, DeactivateBankAccount, DeleteBankAccount
  - **Transaction CRUD**: CreateDeposit, CreateWithdrawal (with currency validation)

## Advanced Features Implemented
- **Timezone-Aware Receipt IDs**: HumanID automatically generated using Assembly timezone in format `20060102150405.000`
- **Cycle Prevention**: BankAccount parent relationships prevent circular references with comprehensive validation
- **Currency Validation**: Transactions validated against account currency to prevent currency mismatches
- **Referential Integrity**: Prevents deletion of entities with dependencies (accounts with children/transactions, receipts with items)
- **Workflow Support**: Undeposited receipts tracking, transaction assignment, hierarchical account queries
- **Idiomatic Go API**: All method names follow Go conventions (e.g., `Item()` instead of `GetItem()`) per Google Go Style Guide

## Contribution Workflow
1. **Create Items** - Define contribution categories (Local Fund, Humanitarian Fund, etc.)
2. **Create Parties** - Add contributors and vendors to the system
3. **Issue Receipts** - When contributions are received, create Receipt with ReceiptItems
4. **Make Deposits** - Group receipts together and create deposit Transaction when money is banked
5. **Track Relationships** - Each receipt can only belong to one deposit transaction (immutable once assigned)

## Next Steps (Suggested)
- Update database schema with new tables in initSchema()
- Assembly creation/retrieval methods
- ReceiptItem CRUD operations
- Transaction querying and reporting methods
- Account balance calculations
- Reporting functionality (contribution summaries, financial statements)
- Data validation rules and constraints

## Testing Commands
- `go mod tidy` - Clean up dependencies
- `go build` - Build the library (currently compiles successfully)
- `go test` - Run tests (when implemented)
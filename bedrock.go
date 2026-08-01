// Package bedrock is a light-weight accounting system for Spiritual Assemblies.
package bedrock

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// ID represents a unique identifier for entities in the system
type ID int64

// Currency represents supported currencies
type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyCAD Currency = "CAD"
)

// Money represents a monetary amount with currency
type Money struct {
	Amount   int64    `db:"amount"` // stored in smallest currency unit (cents)
	Currency Currency `db:"currency"`
}

// NewMoney creates a new Money instance from an amount in cents
func NewMoney(amountCents int64, currency Currency) Money {
	return Money{
		Amount:   amountCents,
		Currency: currency,
	}
}

// Float64 returns the monetary amount as a float64 for display purposes only
func (m Money) Float64() float64 {
	return float64(m.Amount) / 100
}

// String returns a formatted string representation of the money
func (m Money) String() string {
	return fmt.Sprintf("%s%.2f", m.CurrencySymbol(), m.Float64())
}

// CurrencySymbol returns the symbol for the currency (e.g., "$" for USD)
func (m Money) CurrencySymbol() string {
	switch m.Currency {
	case CurrencyUSD:
		return "$"
	case CurrencyCAD:
		return "$"
	default:
		return "$"
	}
}

// AccountType represents the type of bank account
type AccountType string

const (
	AccountTypeChecking AccountType = "checking"
	AccountTypeSavings  AccountType = "savings"
	AccountTypeEarmark  AccountType = "earmark"
)

// Base provides common fields for all persisted entities
type Base struct {
	ID         ID        `db:"id"`
	CreatedAt  time.Time `db:"created_at"`
	ModifiedAt time.Time `db:"modified_at"`
}

// Assembly represents a Spiritual Assembly organization. The issuer fields
// (mailing address, charitable registration number, contact details, and the
// receipt disclaimer) appear on emailed/printed contribution receipts and are
// edited in the Settings screen. They default to the empty string.
type Assembly struct {
	Base
	CharitableRegNumber string         `db:"charitable_reg_number"`
	ContactEmail        string         `db:"contact_email"`
	ContactPhone        string         `db:"contact_phone"`
	DefaultCurrency     Currency       `db:"default_currency"`
	InKindDisclaimer    string         `db:"in_kind_receipt_disclaimer"`
	MailingAddress      string         `db:"mailing_address"`
	Name                string         `db:"name"`
	ReceiptDisclaimer   string         `db:"receipt_disclaimer"`
	Timezone            *time.Location `db:"timezone"`
}

// BankAccount represents a bank account with optional parent for sub-accounts
type BankAccount struct {
	Base
	ParentID    *ID         `db:"parent_id"` // nil for main accounts, set for sub-accounts
	Name        string      `db:"name"`
	AccountType AccountType `db:"account_type"`
	Currency    Currency    `db:"currency"` // account currency (USD or CAD)
	Description string      `db:"description"`
	IsActive    bool        `db:"is_active"`
}

// BankAccountDelta specifies the fields to update on a bank account.
// Fields left nil are not modified. AccountType, Currency, and ParentID are
// immutable after creation and are deliberately absent.
type BankAccountDelta struct {
	Name        *string
	Description *string
	IsActive    *bool
}

// Item represents a fund or category that can receive contributions.
// CountsTowardGoal reports whether contributions to this fund count toward the
// Assembly's annual fundraising goal; earmarked funds (e.g. National Fund,
// Huqúqu'lláh) set it to false.
type Item struct {
	Base
	Name             string `db:"name"`
	CountsTowardGoal bool   `db:"counts_toward_goal"`
}

// Category represents an expense category for withdrawal transactions
type Category struct {
	Base
	Name string `db:"name"`
}

// Receipt represents a receipt issued for contributions
type Receipt struct {
	Base
	HumanID       string    `db:"human_id"`       // human-readable receipt ID
	CustomerID    ID        `db:"customer_id"`    // foreign key to Party (contributor)
	SoldAt        time.Time `db:"sold_at"`        // when contribution was received
	TransactionID *ID       `db:"transaction_id"` // foreign key to deposit transaction (nil if not yet deposited)
	Memo          string    `db:"memo"`           // treasurer's free-form note; empty string when unset
	IsInKind      bool      `db:"is_in_kind"`     // true for non-cash (in-kind) contributions, which never get a deposit
	Total         Money     `db:"-"`              // computed from receipt items, not stored directly
}

// ReceiptItem represents a line item on a receipt
type ReceiptItem struct {
	Base
	ReceiptID   ID     `db:"receipt_id"`  // foreign key to receipt
	ItemID      ID     `db:"item_id"`     // foreign key to item (fund)
	CategoryID  *ID    `db:"category_id"` // optional expense category (in-kind only); nil for cash
	Description string `db:"description"` // description of the contribution
	Price       Money  `db:"price"`       // amount contributed (fair market value for in-kind)
}

// ReceiptItemInput represents a line item when creating a receipt with
// CreateReceiptWithItems. CategoryID is optional and only meaningful for
// in-kind contributions (it records what the donated value was spent on).
type ReceiptItemInput struct {
	ItemID      ID
	CategoryID  *ID
	Description string
	Price       Money
}

// DeliveryStatus records whether a receipt email delivery attempt succeeded.
type DeliveryStatus string

const (
	DeliveryStatusSuccess DeliveryStatus = "success"
	DeliveryStatusFailure DeliveryStatus = "failure"
)

// EmailMethod selects how receipt emails are delivered.
type EmailMethod string

const (
	EmailMethodNone       EmailMethod = ""            // not configured
	EmailMethodSMTP       EmailMethod = "smtp"        // generic SMTP / Gmail app-password
	EmailMethodGmailOAuth EmailMethod = "gmail_oauth" // Gmail API with the gmail.send scope
)

// SMTPSecurity selects the transport security for an SMTP connection.
type SMTPSecurity string

const (
	SMTPSecurityNone     SMTPSecurity = "none"     // plaintext (not recommended)
	SMTPSecuritySTARTTLS SMTPSecurity = "starttls" // upgrade on the submission port (587)
	SMTPSecurityTLS      SMTPSecurity = "tls"      // implicit TLS (465)
)

// EmailSettings holds the per-assembly email delivery configuration. Exactly
// one row exists per database file. Secret fields (SMTPPassword, OAuth client
// secret and tokens) are stored unencrypted in the .bedrock file.
type EmailSettings struct {
	Base
	Method            EmailMethod  `db:"method"`
	FromName          string       `db:"from_name"`
	FromAddress       string       `db:"from_address"`
	ReplyTo           string       `db:"reply_to"`
	SMTPHost          string       `db:"smtp_host"`
	SMTPPort          int          `db:"smtp_port"`
	SMTPUsername      string       `db:"smtp_username"`
	SMTPPassword      string       `db:"smtp_password"`
	SMTPSecurity      SMTPSecurity `db:"smtp_security"`
	OAuthClientID     string       `db:"oauth_client_id"`
	OAuthClientSecret string       `db:"oauth_client_secret"`
	OAuthRefreshToken string       `db:"oauth_refresh_token"`
	OAuthAccessToken  string       `db:"oauth_access_token"`
	OAuthTokenExpiry  *time.Time   `db:"oauth_token_expiry"`
}

// FullReceipt bundles a receipt with its customer, resolved line items, and
// computed total — everything needed to render a receipt PDF without further
// database access.
type FullReceipt struct {
	Receipt  Receipt
	Customer Party
	Items    []FullReceiptItem
	Total    Money
}

// FullReceiptItem pairs a receipt line item with its resolved Item (fund) and,
// for in-kind lines that carry one, the resolved expense Category.
type FullReceiptItem struct {
	Item        Item
	Category    *Category // resolved expense category, or nil
	ReceiptItem ReceiptItem
}

// ReceiptDelivery is one row in the append-only receipt email delivery log.
type ReceiptDelivery struct {
	Base
	ReceiptID        ID             `db:"receipt_id"`
	SentAt           time.Time      `db:"sent_at"`
	Method           string         `db:"method"`
	RecipientAddress string         `db:"recipient_address"`
	Status           DeliveryStatus `db:"status"`
	ErrorMessage     string         `db:"error_message"`
}

// UnsentReceiptsOptions filters the receipts returned by UnsentReceipts. A nil
// field imposes no constraint on that dimension.
type UnsentReceiptsOptions struct {
	StartDate  *time.Time // include receipts sold on or after this date
	EndDate    *time.Time // include receipts sold on or before this date
	CustomerID *ID        // restrict to a single contributor
}

// Expense represents a line item for a withdrawal transaction
type Expense struct {
	Base
	TransactionID ID      `db:"transaction_id"` // foreign key to withdrawal transaction
	CategoryID    ID      `db:"category_id"`    // foreign key to expense category
	Description   *string `db:"description"`    // optional description of the expense
	Amount        Money   `db:"amount"`         // expense amount
}

// ExpenseItem represents an expense item when creating a withdrawal
type ExpenseItem struct {
	CategoryID  ID
	Description *string
	Amount      Money
}

// LedgerEntry represents a single row in the ledger view
type LedgerEntry struct {
	Transaction    Transaction `json:"transaction"`
	RunningBalance Money       `json:"running_balance"`

	// Optional enriched data
	CustomerName *string `json:"customer_name,omitempty"` // For receipts
	PayeeName    *string `json:"payee_name,omitempty"`    // For withdrawals
	ReceiptCount int     `json:"receipt_count,omitempty"` // Number of receipts in deposit
	ExpenseCount int     `json:"expense_count,omitempty"` // Number of expenses in withdrawal
}

// LedgerOptions provides filtering and pagination options
type LedgerOptions struct {
	StartDate          *time.Time `json:"start_date,omitempty"`
	EndDate            *time.Time `json:"end_date,omitempty"`
	Limit              *int       `json:"limit,omitempty"`     // For pagination
	Offset             *int       `json:"offset,omitempty"`    // For pagination
	IncludeSubaccounts bool       `json:"include_subaccounts"` // Include child accounts
}

// Party represents contributors and vendors
type Party struct {
	Base
	Name            string  `db:"name"`
	EmailAddress    *string `db:"email_address"`    // optional email address
	BahaiIDNumber   *string `db:"bahai_id_number"`  // optional Bahá'í ID number
	Address         *string `db:"address"`          // optional physical mailing address
	TelephoneNumber *string `db:"telephone_number"` // optional telephone number
}

// ReconciliationStatus represents the state of a reconciliation session.
// A reconciliation is either in-progress or completed; cancel and undo delete
// the record entirely rather than mark it as cancelled.
type ReconciliationStatus string

const (
	ReconciliationStatusInProgress ReconciliationStatus = "in-progress"
	ReconciliationStatusCompleted  ReconciliationStatus = "completed"
)

// Reconciliation represents a bank account reconciliation session
type Reconciliation struct {
	Base
	AccountID        ID                   `db:"account_id"`     // bank account being reconciled
	StatementDate    time.Time            `db:"statement_date"` // ending date of the bank statement
	StatementBalance Money                `db:"-"`              // ending balance per the bank statement
	Status           ReconciliationStatus `db:"status"`         // in-progress, completed, or cancelled
	CompletedAt      *time.Time           `db:"completed_at"`   // when reconciliation was finalized
}

// reconciliationRow is used for database scanning with flat fields
type reconciliationRow struct {
	Base
	AccountID                ID                   `db:"account_id"`
	StatementDate            time.Time            `db:"statement_date"`
	StatementBalance         int64                `db:"statement_balance"`
	StatementBalanceCurrency Currency             `db:"statement_balance_currency"`
	Status                   ReconciliationStatus `db:"status"`
	CompletedAt              *time.Time           `db:"completed_at"`
}

func (r reconciliationRow) toReconciliation() Reconciliation {
	return Reconciliation{
		Base:          r.Base,
		AccountID:     r.AccountID,
		StatementDate: r.StatementDate,
		StatementBalance: Money{
			Amount:   r.StatementBalance,
			Currency: r.StatementBalanceCurrency,
		},
		Status:      r.Status,
		CompletedAt: r.CompletedAt,
	}
}

// TransactionMethod represents how a transaction was performed
type TransactionMethod string

const (
	TransactionMethodATM                TransactionMethod = "atm"
	TransactionMethodAutoPay            TransactionMethod = "auto-pay"
	TransactionMethodElectronicTransfer TransactionMethod = "electronic-transfer"
	TransactionMethodInBranch           TransactionMethod = "in-branch"
	TransactionMethodMobileApp          TransactionMethod = "mobile-app"
	TransactionMethodCheck              TransactionMethod = "check"
)

// Label returns a human-readable label for the transaction method, suitable
// for display in the UI. Unknown values fall back to their string form.
func (m TransactionMethod) Label() string {
	switch m {
	case TransactionMethodATM:
		return "ATM"
	case TransactionMethodAutoPay:
		return "Auto-pay"
	case TransactionMethodCheck:
		return "Check"
	case TransactionMethodElectronicTransfer:
		return "Electronic Transfer"
	case TransactionMethodInBranch:
		return "In-Branch"
	case TransactionMethodMobileApp:
		return "Mobile App"
	default:
		return string(m)
	}
}

// Transaction represents a deposit or withdrawal in a bank account
type Transaction struct {
	Base
	AccountID        ID                 `db:"account_id"`        // bank account for this transaction
	Amount           int64              `db:"amount"`            // positive for deposits, negative for withdrawals (in cents)
	CheckNumber      *string            `db:"check_number"`      // check number if withdrawal by check
	Memo             string             `db:"memo"`              // treasurer's notes
	Method           *TransactionMethod `db:"method"`            // how the transaction was performed
	PayeeID          *ID                `db:"payee_id"`          // foreign key to Party if this is a payment
	ReconciliationID *ID                `db:"reconciliation_id"` // foreign key to Reconciliation if cleared
	TransactedAt     time.Time          `db:"transacted_at"`     // when the transaction occurred
}

// DB represents the database connection and operations
type DB struct {
	conn *sqlx.DB
	sq   squirrel.StatementBuilderType
}

// New creates a new bedrock database file with an initialized Spiritual Assembly
func New(filepath string, assemblyName string, timezone *time.Location, defaultCurrency Currency) (*DB, error) {
	if assemblyName == "" {
		return nil, fmt.Errorf("assembly name cannot be empty")
	}
	if timezone == nil {
		return nil, fmt.Errorf("timezone cannot be nil")
	}
	if defaultCurrency != CurrencyUSD && defaultCurrency != CurrencyCAD {
		return nil, fmt.Errorf("invalid currency: %s", defaultCurrency)
	}

	// Create the database using Open
	db, err := Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Create the assembly
	_, err = db.createAssembly(assemblyName, timezone, defaultCurrency)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create assembly: %w", err)
	}

	return db, nil
}

// Open opens or creates a bedrock database file. On open it runs any pending
// schema migrations to bring the file up to currentSchemaVersion. If the file
// is at a version newer than this binary knows about, Open returns an error.
func Open(filepath string) (*DB, error) {
	conn, err := sqlx.Connect("sqlite", filepath+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{
		conn: conn,
		sq:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
	}

	if err := db.migrate(context.Background()); err != nil {
		conn.Close()
		return nil, err
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

package bedrock

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrate_V2ToV3Receipting stages a v2 database (the schema before the
// receipting migration), opens it through bedrock.Open, and verifies the
// upgrade adds the assembly issuer columns, email_settings, and
// receipt_deliveries, all writable.
func TestMigrate_V2ToV3Receipting(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "v2.bedrock")

	// Stage a v2 file: apply migrations 0001 and 0002, stamp user_version=2.
	{
		raw := openRaw(t, dbPath)
		for _, v := range []int{1, 2} {
			sql, err := loadMigration(v)
			require.NoError(t, err)
			_, err = raw.Exec(sql)
			require.NoError(t, err)
		}
		_, err := raw.Exec("PRAGMA user_version = 2")
		require.NoError(t, err)
		require.NoError(t, raw.Close())
	}

	db, err := Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	var v int
	require.NoError(t, db.conn.Get(&v, "PRAGMA user_version"))
	assert.Equal(t, currentSchemaVersion, v)

	// Assembly issuer columns exist and are writable.
	_, err = db.createAssembly("Migrated Assembly", time.UTC, CurrencyUSD)
	require.NoError(t, err)
	updated, err := db.UpdateAssemblyDetails("1 Main St", "REG-1", "t@lsa.org", "555-0100", "Custom statement")
	require.NoError(t, err)
	assert.Equal(t, "1 Main St", updated.MailingAddress)

	// New tables exist and accept writes.
	party, err := db.CreateParty("Contributor", nil, nil, nil, nil)
	require.NoError(t, err)
	item, err := db.CreateItem("Local Fund")
	require.NoError(t, err)
	receipt, err := db.CreateReceiptWithItems(party.ID, time.Now(), "", []ReceiptItemInput{
		{ItemID: item.ID, Price: NewMoney(1000, CurrencyUSD)},
	})
	require.NoError(t, err)

	_, err = db.UpdateEmailSettings(EmailSettings{Method: EmailMethodSMTP, FromAddress: "t@lsa.org"})
	require.NoError(t, err)
	_, err = db.RecordReceiptDelivery(receipt.ID, time.Now(), "smtp", "c@lsa.org", DeliveryStatusSuccess, "")
	require.NoError(t, err)
}

// TestAssemblyIssuerDetails verifies issuer-field defaults and round-tripping.
func TestAssemblyIssuerDetails(t *testing.T) {
	db := testDB(t)

	assembly, err := db.createAssembly("Test Assembly", time.UTC, CurrencyUSD)
	require.NoError(t, err)

	// A fresh assembly has empty issuer fields except the seeded disclaimer.
	assert.Empty(t, assembly.MailingAddress)
	assert.Empty(t, assembly.CharitableRegNumber)
	assert.Equal(t, "No goods or services were provided in exchange for this contribution.", assembly.ReceiptDisclaimer)

	updated, err := db.UpdateAssemblyDetails("123 Main St\nSpringfield", "CRA-12345", "treasurer@lsa.org", "555-1234", "Thank you for your gift.")
	require.NoError(t, err)
	assert.Equal(t, "123 Main St\nSpringfield", updated.MailingAddress)
	assert.Equal(t, "CRA-12345", updated.CharitableRegNumber)
	assert.Equal(t, "treasurer@lsa.org", updated.ContactEmail)
	assert.Equal(t, "555-1234", updated.ContactPhone)
	assert.Equal(t, "Thank you for your gift.", updated.ReceiptDisclaimer)

	// Core fields are untouched and the change persists.
	got, err := db.Assembly()
	require.NoError(t, err)
	assert.Equal(t, "Test Assembly", got.Name)
	assert.Equal(t, CurrencyUSD, got.DefaultCurrency)
	assert.Equal(t, time.UTC.String(), got.Timezone.String())
	assert.Equal(t, "123 Main St\nSpringfield", got.MailingAddress)
}

// TestEmailSettings verifies default settings, upsert behavior, and that only a
// single settings row is ever maintained.
func TestEmailSettings(t *testing.T) {
	db := testDB(t)

	// No row yet: sensible defaults, zero ID.
	s, err := db.EmailSettings()
	require.NoError(t, err)
	assert.Equal(t, EmailMethodNone, s.Method)
	assert.Equal(t, 587, s.SMTPPort)
	assert.Equal(t, SMTPSecuritySTARTTLS, s.SMTPSecurity)
	assert.Zero(t, s.ID)

	// First save inserts the row.
	s.Method = EmailMethodSMTP
	s.FromName = "Treasurer"
	s.FromAddress = "treasurer@lsa.org"
	s.SMTPHost = "smtp.example.com"
	s.SMTPUsername = "user"
	s.SMTPPassword = "secret"
	saved, err := db.UpdateEmailSettings(*s)
	require.NoError(t, err)
	assert.NotZero(t, saved.ID)
	assert.Equal(t, EmailMethodSMTP, saved.Method)
	assert.Equal(t, "secret", saved.SMTPPassword)

	got, err := db.EmailSettings()
	require.NoError(t, err)
	assert.Equal(t, saved.ID, got.ID)
	assert.Equal(t, "Treasurer", got.FromName)

	// Second save updates the same row rather than inserting a new one.
	got.FromName = "New Treasurer"
	saved2, err := db.UpdateEmailSettings(*got)
	require.NoError(t, err)
	assert.Equal(t, saved.ID, saved2.ID)
	assert.Equal(t, "New Treasurer", saved2.FromName)

	var count int
	require.NoError(t, db.conn.Get(&count, "SELECT COUNT(*) FROM email_settings"))
	assert.Equal(t, 1, count)
}

// TestUpdateOAuthTokens verifies token persistence, refresh-token preservation,
// and that connecting before configuring settings is rejected.
func TestUpdateOAuthTokens(t *testing.T) {
	db := testDB(t)

	// No settings yet: must reject.
	_, err := db.UpdateOAuthTokens("refresh", "access", time.Now().Add(time.Hour))
	assert.Error(t, err)

	_, err = db.UpdateEmailSettings(EmailSettings{
		Method:            EmailMethodGmailOAuth,
		OAuthClientID:     "client-id",
		OAuthClientSecret: "client-secret",
	})
	require.NoError(t, err)

	expiry := time.Now().Add(time.Hour)
	saved, err := db.UpdateOAuthTokens("refresh-1", "access-1", expiry)
	require.NoError(t, err)
	assert.Equal(t, "refresh-1", saved.OAuthRefreshToken)
	assert.Equal(t, "access-1", saved.OAuthAccessToken)
	require.NotNil(t, saved.OAuthTokenExpiry)
	assert.WithinDuration(t, expiry, *saved.OAuthTokenExpiry, time.Second)

	// A later refresh with an empty refresh token must not clobber the stored one.
	saved2, err := db.UpdateOAuthTokens("", "access-2", expiry.Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, "refresh-1", saved2.OAuthRefreshToken)
	assert.Equal(t, "access-2", saved2.OAuthAccessToken)
	// Unrelated settings survive token updates.
	assert.Equal(t, "client-id", saved2.OAuthClientID)
	assert.Equal(t, EmailMethodGmailOAuth, saved2.Method)
}

// TestReceiptDeliveries verifies the delivery log and the "last successful"
// lookup that drives the unsent check.
func TestReceiptDeliveries(t *testing.T) {
	db := testDB(t)
	_, _, item, party, _ := setupTestData(t, db)

	receipt, err := db.CreateReceiptWithItems(party.ID, time.Now(), "", []ReceiptItemInput{
		{ItemID: item.ID, Price: NewMoney(1000, CurrencyUSD)},
	})
	require.NoError(t, err)

	last, err := db.LastSuccessfulDelivery(receipt.ID)
	require.NoError(t, err)
	assert.Nil(t, last)

	history, err := db.ReceiptDeliveries(receipt.ID)
	require.NoError(t, err)
	assert.Empty(t, history)

	// A failure does not count as sent.
	failAt := time.Now()
	_, err = db.RecordReceiptDelivery(receipt.ID, failAt, "smtp", "c@lsa.org", DeliveryStatusFailure, "connection refused")
	require.NoError(t, err)
	last, err = db.LastSuccessfulDelivery(receipt.ID)
	require.NoError(t, err)
	assert.Nil(t, last)

	// A success does.
	successAt := failAt.Add(time.Minute)
	success, err := db.RecordReceiptDelivery(receipt.ID, successAt, "smtp", "c@lsa.org", DeliveryStatusSuccess, "")
	require.NoError(t, err)
	last, err = db.LastSuccessfulDelivery(receipt.ID)
	require.NoError(t, err)
	require.NotNil(t, last)
	assert.Equal(t, success.ID, last.ID)

	// History is most-recent-first.
	history, err = db.ReceiptDeliveries(receipt.ID)
	require.NoError(t, err)
	require.Len(t, history, 2)
	assert.Equal(t, DeliveryStatusSuccess, history[0].Status)
	assert.Equal(t, DeliveryStatusFailure, history[1].Status)
	assert.Equal(t, "connection refused", history[1].ErrorMessage)
}

// TestUnsentReceipts verifies that the unsent query excludes successfully
// delivered receipts, keeps ones with only failures, and honors the filters.
func TestUnsentReceipts(t *testing.T) {
	db := testDB(t)
	_, _, item, party, _ := setupTestData(t, db)

	other, err := db.CreateParty("Second Contributor", nil, nil, nil, nil)
	require.NoError(t, err)

	mkReceipt := func(customerID ID, soldAt time.Time) *Receipt {
		r, err := db.CreateReceiptWithItems(customerID, soldAt, "", []ReceiptItemInput{
			{ItemID: item.ID, Price: NewMoney(1000, CurrencyUSD)},
		})
		require.NoError(t, err)
		return r
	}

	jan := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	feb := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	mar := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	r1 := mkReceipt(party.ID, jan)
	r2 := mkReceipt(party.ID, feb)
	r3 := mkReceipt(other.ID, mar)

	all, err := db.UnsentReceipts(UnsentReceiptsOptions{})
	require.NoError(t, err)
	assert.Len(t, all, 3)

	// r1 delivered -> excluded; r2 failed-only -> still unsent.
	_, err = db.RecordReceiptDelivery(r1.ID, time.Now(), "smtp", "x@y.com", DeliveryStatusSuccess, "")
	require.NoError(t, err)
	_, err = db.RecordReceiptDelivery(r2.ID, time.Now(), "smtp", "x@y.com", DeliveryStatusFailure, "oops")
	require.NoError(t, err)

	remaining, err := db.UnsentReceipts(UnsentReceiptsOptions{})
	require.NoError(t, err)
	ids := receiptIDSet(remaining)
	assert.NotContains(t, ids, r1.ID)
	assert.Contains(t, ids, r2.ID)
	assert.Contains(t, ids, r3.ID)

	// Filter by customer.
	byCustomer, err := db.UnsentReceipts(UnsentReceiptsOptions{CustomerID: &party.ID})
	require.NoError(t, err)
	require.Len(t, byCustomer, 1)
	assert.Equal(t, r2.ID, byCustomer[0].ID)

	// Filter by date range (Feb–Mar) keeps r2 and r3.
	start := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)
	byDate, err := db.UnsentReceipts(UnsentReceiptsOptions{StartDate: &start, EndDate: &end})
	require.NoError(t, err)
	dateIDs := receiptIDSet(byDate)
	assert.Contains(t, dateIDs, r2.ID)
	assert.Contains(t, dateIDs, r3.ID)
}

// TestFullReceipt verifies the bundled receipt used for PDF rendering.
func TestFullReceipt(t *testing.T) {
	db := testDB(t)
	_, _, item, party, _ := setupTestData(t, db)

	item2, err := db.CreateItem("Humanitarian Fund")
	require.NoError(t, err)

	receipt, err := db.CreateReceiptWithItems(party.ID, time.Now(), "thanks", []ReceiptItemInput{
		{ItemID: item.ID, Description: "January", Price: NewMoney(1000, CurrencyUSD)},
		{ItemID: item2.ID, Description: "February", Price: NewMoney(2500, CurrencyUSD)},
	})
	require.NoError(t, err)

	full, err := db.FullReceipt(receipt.ID)
	require.NoError(t, err)
	assert.Equal(t, receipt.ID, full.Receipt.ID)
	assert.Equal(t, party.Name, full.Customer.Name)
	require.Len(t, full.Items, 2)
	assert.Equal(t, item.Name, full.Items[0].Item.Name)
	assert.Equal(t, item2.Name, full.Items[1].Item.Name)
	assert.Equal(t, int64(3500), full.Total.Amount)
	assert.Equal(t, CurrencyUSD, full.Total.Currency)

	// An itemless receipt totals zero in the assembly's default currency.
	bare, err := db.CreateReceipt(party.ID, time.Now(), "")
	require.NoError(t, err)
	fullBare, err := db.FullReceipt(bare.ID)
	require.NoError(t, err)
	assert.Empty(t, fullBare.Items)
	assert.Equal(t, int64(0), fullBare.Total.Amount)
	assert.Equal(t, CurrencyUSD, fullBare.Total.Currency)
}

// receiptIDSet collects receipt IDs into a set for membership assertions.
func receiptIDSet(receipts []Receipt) map[ID]bool {
	set := make(map[ID]bool, len(receipts))
	for _, r := range receipts {
		set[r.ID] = true
	}
	return set
}

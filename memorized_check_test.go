package bedrock

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemorizedCheckCRUD(t *testing.T) {
	db := testDB(t)
	_, account, _, payee, category := setupTestData(t, db)

	// A second category to exercise multi-line templates.
	travel, err := db.CreateCategory("Travel")
	require.NoError(t, err)

	desc := "Monthly office rent"
	expenses := []ExpenseItem{
		{CategoryID: category.ID, Description: &desc, Amount: NewMoney(120_000, CurrencyUSD)},
		{CategoryID: travel.ID, Amount: NewMoney(5_000, CurrencyUSD)},
	}

	mc, err := db.CreateMemorizedCheck("Monthly rent + travel", account.ID, payee.ID, "rent & travel", expenses)
	require.NoError(t, err)
	assert.Equal(t, "Monthly rent + travel", mc.Name)
	assert.Equal(t, account.ID, mc.AccountID)
	assert.Equal(t, payee.ID, mc.PayeeID)
	assert.Equal(t, "rent & travel", mc.Memo)

	// Read back the header.
	got, err := db.MemorizedCheck(mc.ID)
	require.NoError(t, err)
	assert.Equal(t, mc.ID, got.ID)
	assert.Equal(t, "Monthly rent + travel", got.Name)

	// Read back the expense lines (order preserved, money + description intact).
	lines, err := db.MemorizedCheckExpenses(mc.ID)
	require.NoError(t, err)
	require.Len(t, lines, 2)
	assert.Equal(t, category.ID, lines[0].CategoryID)
	require.NotNil(t, lines[0].Description)
	assert.Equal(t, "Monthly office rent", *lines[0].Description)
	assert.Equal(t, int64(120_000), lines[0].Amount.Amount)
	assert.Equal(t, travel.ID, lines[1].CategoryID)
	assert.Nil(t, lines[1].Description)
	assert.Equal(t, int64(5_000), lines[1].Amount.Amount)

	// List.
	all, err := db.MemorizedChecks()
	require.NoError(t, err)
	assert.Len(t, all, 1)

	// Rename.
	renamed, err := db.RenameMemorizedCheck(mc.ID, "Rent")
	require.NoError(t, err)
	assert.Equal(t, "Rent", renamed.Name)

	// Delete removes the header and its expense lines.
	require.NoError(t, db.DeleteMemorizedCheck(mc.ID))
	all, err = db.MemorizedChecks()
	require.NoError(t, err)
	assert.Empty(t, all)
	lines, err = db.MemorizedCheckExpenses(mc.ID)
	require.NoError(t, err)
	assert.Empty(t, lines)
}

func TestMemorizedChecksOrderedByName(t *testing.T) {
	db := testDB(t)
	_, account, _, payee, category := setupTestData(t, db)

	line := []ExpenseItem{{CategoryID: category.ID, Amount: NewMoney(1_000, CurrencyUSD)}}
	_, err := db.CreateMemorizedCheck("zebra", account.ID, payee.ID, "", line)
	require.NoError(t, err)
	_, err = db.CreateMemorizedCheck("apple", account.ID, payee.ID, "", line)
	require.NoError(t, err)

	all, err := db.MemorizedChecks()
	require.NoError(t, err)
	require.Len(t, all, 2)
	assert.Equal(t, "apple", all[0].Name)
	assert.Equal(t, "zebra", all[1].Name)
}

func TestCreateMemorizedCheckValidation(t *testing.T) {
	db := testDB(t)
	_, account, _, payee, category := setupTestData(t, db)
	line := []ExpenseItem{{CategoryID: category.ID, Amount: NewMoney(1_000, CurrencyUSD)}}

	_, err := db.CreateMemorizedCheck("", account.ID, payee.ID, "", line)
	assert.Error(t, err, "empty name rejected")

	_, err = db.CreateMemorizedCheck("No lines", account.ID, payee.ID, "", nil)
	assert.Error(t, err, "no expenses rejected")

	_, err = db.CreateMemorizedCheck("Bad amount", account.ID, payee.ID, "",
		[]ExpenseItem{{CategoryID: category.ID, Amount: NewMoney(0, CurrencyUSD)}})
	assert.Error(t, err, "non-positive amount rejected")

	// Account is USD; a CAD expense line must be rejected.
	_, err = db.CreateMemorizedCheck("Wrong currency", account.ID, payee.ID, "",
		[]ExpenseItem{{CategoryID: category.ID, Amount: NewMoney(1_000, CurrencyCAD)}})
	assert.Error(t, err, "currency mismatch rejected")

	_, err = db.RenameMemorizedCheck(99999, "x")
	assert.Error(t, err, "rename of missing check errors")
}

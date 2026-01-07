package bedrock

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// strPtr returns a pointer to a string (helper for tests)
func strPtr(s string) *string {
	return &s
}

// testDB creates a temporary test database
func testDB(t *testing.T) *DB {
	t.Helper()

	// Create a temporary file for the test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.bedrock")

	db, err := Open(dbPath)
	require.NoError(t, err, "Failed to open test database")

	// Cleanup function to close the database
	t.Cleanup(func() {
		err := db.Close()
		assert.NoError(t, err, "Failed to close test database")
	})

	return db
}

// setupTestData creates common test data
func setupTestData(t *testing.T, db *DB) (assembly *Assembly, account *BankAccount, item *Item, party *Party, category *Category) {
	t.Helper()

	// Create assembly
	assembly, err := db.createAssembly("Test Assembly", time.UTC, CurrencyUSD)
	require.NoError(t, err, "Failed to create test assembly")

	// Create test bank account
	account, err = db.CreateBankAccount("Test Checking", AccountTypeChecking, CurrencyUSD, nil, "Test account", true, Money{})
	require.NoError(t, err, "Failed to create test bank account")

	// Create test item
	item, err = db.CreateItem("Local Fund")
	require.NoError(t, err, "Failed to create test item")

	// Create test party
	email := "test@example.com"
	party, err = db.CreateParty("Test Contributor", &email, nil, nil, nil)
	require.NoError(t, err, "Failed to create test party")

	// Create test category
	category, err = db.CreateCategory("Office Supplies")
	require.NoError(t, err, "Failed to create test category")

	return assembly, account, item, party, category
}

func TestDatabaseOpenClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.bedrock")

	// Test opening a new database
	db, err := Open(dbPath)
	require.NoError(t, err, "Failed to open database")

	// Verify the database connection works
	assert.NotNil(t, db.conn, "Database connection should not be nil")

	// Test closing the database
	err = db.Close()
	require.NoError(t, err, "Failed to close database")

	// Test opening an existing database
	db2, err := Open(dbPath)
	require.NoError(t, err, "Failed to reopen existing database")
	defer db2.Close()
}

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()

	// Test creating a new bedrock database
	t.Run("NewDatabase", func(t *testing.T) {
		dbPath := filepath.Join(tmpDir, "new.bedrock")
		eastern, err := time.LoadLocation("America/New_York")
		require.NoError(t, err, "Failed to load timezone")

		db, err := New(dbPath, "Test Spiritual Assembly", eastern, CurrencyUSD)
		require.NoError(t, err, "New should succeed")
		defer db.Close()

		// Verify assembly was created
		assembly, err := db.Assembly()
		require.NoError(t, err, "Should be able to retrieve assembly")
		assert.Equal(t, "Test Spiritual Assembly", assembly.Name)
		assert.Equal(t, "America/New_York", assembly.Timezone.String())
		assert.Equal(t, CurrencyUSD, assembly.DefaultCurrency)
		assert.NotZero(t, assembly.ID)
	})

	// Test with empty assembly name
	t.Run("EmptyAssemblyName", func(t *testing.T) {
		dbPath := filepath.Join(tmpDir, "empty_name.bedrock")
		_, err := New(dbPath, "", time.UTC, CurrencyUSD)
		assert.Error(t, err, "Expected error for empty assembly name")
	})

	// Test with nil timezone
	t.Run("NilTimezone", func(t *testing.T) {
		dbPath := filepath.Join(tmpDir, "nil_timezone.bedrock")
		_, err := New(dbPath, "Test Assembly", nil, CurrencyUSD)
		assert.Error(t, err, "Expected error for nil timezone")
	})

	// Test with invalid currency
	t.Run("InvalidCurrency", func(t *testing.T) {
		dbPath := filepath.Join(tmpDir, "invalid_currency.bedrock")
		_, err := New(dbPath, "Test Assembly", time.UTC, Currency("EUR"))
		assert.Error(t, err, "Expected error for invalid currency")
	})
}

func TestAssemblyCRUD(t *testing.T) {
	pacific, err := time.LoadLocation("America/Los_Angeles")
	require.NoError(t, err, "Failed to load timezone")

	// Test assembly creation through bedrock.New
	t.Run("AssemblyCreationViaNew", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.bedrock")

		db, err := New(dbPath, "Local Spiritual Assembly of Test City", pacific, CurrencyCAD)
		require.NoError(t, err, "bedrock.New should succeed")
		defer db.Close()

		assembly, err := db.Assembly()
		require.NoError(t, err, "Should be able to retrieve assembly")

		assert.NotZero(t, assembly.ID, "Assembly ID should not be zero")
		assert.Equal(t, "Local Spiritual Assembly of Test City", assembly.Name)
		assert.Equal(t, "America/Los_Angeles", assembly.Timezone.String())
		assert.Equal(t, CurrencyCAD, assembly.DefaultCurrency)
		assert.NotZero(t, assembly.CreatedAt)
		assert.NotZero(t, assembly.ModifiedAt)
	})

	// Test duplicate assembly prevention (already tested in TestNew, but testing internal behavior)
	t.Run("DuplicateAssemblyPrevention", func(t *testing.T) {
		freshDB := testDB(t)
		defer freshDB.Close()

		// First assembly should succeed (internal method for testing)
		_, err := freshDB.createAssembly("First Assembly", pacific, CurrencyUSD)
		require.NoError(t, err, "First assembly creation should succeed")

		// Second assembly should fail
		_, err = freshDB.createAssembly("Second Assembly", pacific, CurrencyUSD)
		assert.Error(t, err, "Expected error for duplicate assembly creation")
	})

	// Test Assembly retrieval
	t.Run("AssemblyRetrieval", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "retrieval_test.bedrock")

		eastern, err := time.LoadLocation("America/New_York")
		require.NoError(t, err, "Failed to load timezone")

		// Create database with assembly via New
		db, err := New(dbPath, "Test Assembly for Retrieval", eastern, CurrencyUSD)
		require.NoError(t, err, "Failed to create database")
		defer db.Close()

		// Retrieve assembly
		retrieved, err := db.Assembly()
		require.NoError(t, err, "Assembly retrieval should succeed")

		assert.Equal(t, "Test Assembly for Retrieval", retrieved.Name)
		assert.Equal(t, "America/New_York", retrieved.Timezone.String())
		assert.Equal(t, CurrencyUSD, retrieved.DefaultCurrency)
		assert.NotZero(t, retrieved.ID)
	})

	// Test Assembly retrieval with no assembly
	t.Run("NoAssemblyFound", func(t *testing.T) {
		emptyDB := testDB(t)
		defer emptyDB.Close()

		_, err := emptyDB.Assembly()
		assert.Error(t, err, "Expected error when no assembly exists")
	})

	// Test UpdateAssembly
	t.Run("UpdateAssembly", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "update_test.bedrock")

		central, err := time.LoadLocation("America/Chicago")
		require.NoError(t, err, "Failed to load timezone")

		// Create database with assembly via New
		db, err := New(dbPath, "Original Name", time.UTC, CurrencyUSD)
		require.NoError(t, err, "Failed to create database")
		defer db.Close()

		// Small delay to ensure timestamp difference
		time.Sleep(time.Millisecond)

		// Update assembly
		updated, err := db.UpdateAssembly("Updated Name", central, CurrencyCAD)
		require.NoError(t, err, "UpdateAssembly should succeed")

		assert.Equal(t, "Updated Name", updated.Name)
		assert.Equal(t, "America/Chicago", updated.Timezone.String())
		assert.Equal(t, CurrencyCAD, updated.DefaultCurrency)

		// Verify the update persisted
		retrieved, err := db.Assembly()
		require.NoError(t, err, "Should be able to retrieve updated assembly")
		assert.Equal(t, "Updated Name", retrieved.Name)
		assert.Equal(t, "America/Chicago", retrieved.Timezone.String())
		assert.Equal(t, CurrencyCAD, retrieved.DefaultCurrency)
	})
}

func TestMoneyType(t *testing.T) {
	tests := []struct {
		name          string
		amountCents   int64
		currency      Currency
		expectedStr   string
		expectedFloat float64
	}{
		{"USD dollars", 1000, CurrencyUSD, "$10.00", 10.00},
		{"USD cents", 199, CurrencyUSD, "$1.99", 1.99},
		{"CAD", 2550, CurrencyCAD, "$25.50", 25.50},
		{"Zero", 0, CurrencyUSD, "$0.00", 0.00},
		{"Large amount", 1234567, CurrencyUSD, "$12345.67", 12345.67},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			money := NewMoney(tt.amountCents, tt.currency)

			assert.Equal(t, tt.amountCents, money.Amount)
			assert.Equal(t, tt.currency, money.Currency)
			assert.Equal(t, tt.expectedStr, money.String())
			assert.Equal(t, tt.expectedFloat, money.Float64())
		})
	}
}

func TestItemCRUD(t *testing.T) {
	db := testDB(t)

	// Test CreateItem
	t.Run("CreateItem", func(t *testing.T) {
		item, err := db.CreateItem("Test Fund")
		require.NoError(t, err, "CreateItem should succeed")

		assert.Equal(t, "Test Fund", item.Name)
		assert.NotZero(t, item.ID, "ID should not be zero")
		assert.False(t, item.CreatedAt.IsZero(), "CreatedAt should not be zero")
		assert.False(t, item.ModifiedAt.IsZero(), "ModifiedAt should not be zero")
	})

	// Test CreateItem with empty name
	t.Run("CreateItemEmptyName", func(t *testing.T) {
		_, err := db.CreateItem("")
		assert.Error(t, err, "Expected error for empty name")
	})

	// Create an item for other tests
	item, err := db.CreateItem("Humanitarian Fund")
	require.NoError(t, err, "Failed to create test item")

	// Test Item retrieval
	t.Run("Item", func(t *testing.T) {
		retrieved, err := db.Item(item.ID)
		require.NoError(t, err, "Item retrieval should succeed")

		assert.Equal(t, item.ID, retrieved.ID)
		assert.Equal(t, item.Name, retrieved.Name)
	})

	// Test ItemByName
	t.Run("ItemByName", func(t *testing.T) {
		retrieved, err := db.ItemByName("Humanitarian Fund")
		require.NoError(t, err, "ItemByName should succeed")

		assert.Equal(t, item.ID, retrieved.ID)
	})

	// Test ListItems
	t.Run("ListItems", func(t *testing.T) {
		items, err := db.ListItems()
		require.NoError(t, err, "ListItems should succeed")

		assert.NotEmpty(t, items, "Expected at least one item")

		// Should be sorted by name
		for i := 1; i < len(items); i++ {
			assert.LessOrEqual(t, items[i-1].Name, items[i].Name, "Items should be sorted by name")
		}
	})

	// Test UpdateItem
	t.Run("UpdateItem", func(t *testing.T) {
		updated, err := db.UpdateItem(item.ID, "Updated Fund Name")
		require.NoError(t, err, "UpdateItem should succeed")

		assert.Equal(t, "Updated Fund Name", updated.Name)
		assert.Equal(t, item.ID, updated.ID)
	})

	// Test UpdateItem with empty name
	t.Run("UpdateItemEmptyName", func(t *testing.T) {
		_, err := db.UpdateItem(item.ID, "")
		assert.Error(t, err, "Expected error for empty name")
	})

	// Test DeleteItem
	t.Run("DeleteItem", func(t *testing.T) {
		err := db.DeleteItem(item.ID)
		require.NoError(t, err, "DeleteItem should succeed")

		// Verify item is deleted
		_, err = db.Item(item.ID)
		assert.Error(t, err, "Expected error when retrieving deleted item")
	})

	// Test DeleteItem non-existent
	t.Run("DeleteItemNonExistent", func(t *testing.T) {
		err := db.DeleteItem(99999)
		assert.Error(t, err, "Expected error when deleting non-existent item")
	})
}

func TestCategoryCRUD(t *testing.T) {
	db := testDB(t)

	// Test CreateCategory
	t.Run("CreateCategory", func(t *testing.T) {
		category, err := db.CreateCategory("Office Supplies")
		require.NoError(t, err, "CreateCategory should succeed")

		assert.NotZero(t, category.ID, "Category ID should not be zero")
		assert.Equal(t, "Office Supplies", category.Name)
		assert.NotZero(t, category.CreatedAt)
		assert.NotZero(t, category.ModifiedAt)
	})

	// Test Category retrieval
	t.Run("CategoryRetrieval", func(t *testing.T) {
		created, err := db.CreateCategory("Food")
		require.NoError(t, err, "Failed to create category")

		// Test Category by ID
		category, err := db.Category(created.ID)
		require.NoError(t, err, "Category retrieval should succeed")
		assert.Equal(t, created.ID, category.ID)
		assert.Equal(t, "Food", category.Name)

		// Test CategoryByName
		categoryByName, err := db.CategoryByName("Food")
		require.NoError(t, err, "CategoryByName should succeed")
		assert.Equal(t, created.ID, categoryByName.ID)
		assert.Equal(t, "Food", categoryByName.Name)

		// Test CategoryByName with non-existent name
		_, err = db.CategoryByName("NonExistent")
		assert.Error(t, err, "Expected error for non-existent category")
	})

	// Test Categories listing
	t.Run("CategoriesListing", func(t *testing.T) {
		// Create multiple categories
		_, err := db.CreateCategory("Supplies")
		require.NoError(t, err, "Failed to create category")
		_, err = db.CreateCategory("Equipment")
		require.NoError(t, err, "Failed to create category")

		categories, err := db.Categories()
		require.NoError(t, err, "Categories should succeed")
		assert.GreaterOrEqual(t, len(categories), 2, "Should have at least 2 categories")

		// Verify ordering by name
		for i := 1; i < len(categories); i++ {
			assert.LessOrEqual(t, categories[i-1].Name, categories[i].Name, "Categories should be ordered by name")
		}
	})

	// Test UpdateCategory
	t.Run("UpdateCategory", func(t *testing.T) {
		category, err := db.CreateCategory("Old Name")
		require.NoError(t, err, "Failed to create category")

		// Small delay to ensure timestamp difference
		time.Sleep(time.Millisecond)

		updated, err := db.UpdateCategory(category.ID, "New Name")
		require.NoError(t, err, "UpdateCategory should succeed")

		assert.Equal(t, category.ID, updated.ID)
		assert.Equal(t, "New Name", updated.Name)
		assert.True(t, updated.ModifiedAt.After(category.ModifiedAt) || updated.ModifiedAt.Equal(category.ModifiedAt), "ModifiedAt should be updated or at least equal")
	})

	// Test DeleteCategory
	t.Run("DeleteCategory", func(t *testing.T) {
		category, err := db.CreateCategory("To Delete")
		require.NoError(t, err, "Failed to create category")

		err = db.DeleteCategory(category.ID)
		require.NoError(t, err, "DeleteCategory should succeed")

		// Verify category is deleted
		_, err = db.Category(category.ID)
		assert.Error(t, err, "Expected error when retrieving deleted category")
	})

	// Test DeleteCategory with dependencies
	t.Run("DeleteCategoryWithDependencies", func(t *testing.T) {
		assembly, account, _, party, category := setupTestData(t, db)
		_ = assembly

		// Create a withdrawal with expenses using this category
		expenses := []ExpenseItem{
			{
				CategoryID:  category.ID,
				Description: strPtr("Test expense"),
				Amount:      NewMoney(1000, CurrencyUSD),
			},
		}
		_, err := db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "Test", time.Now(), nil, expenses)
		require.NoError(t, err, "Failed to create withdrawal")

		// Try to delete category with dependencies
		err = db.DeleteCategory(category.ID)
		assert.Error(t, err, "Expected error when deleting category with dependencies")
	})
}

func TestPartyCRUD(t *testing.T) {
	db := testDB(t)

	// Test CreateParty
	t.Run("CreateParty", func(t *testing.T) {
		email := "john@example.com"
		bahaiID := "12345"
		address := "123 Main St"
		phone := "555-1234"

		party, err := db.CreateParty("John Doe", &email, &bahaiID, &address, &phone)
		require.NoError(t, err, "CreateParty should succeed")

		assert.Equal(t, "John Doe", party.Name)
		require.NotNil(t, party.EmailAddress)
		assert.Equal(t, email, *party.EmailAddress)
		require.NotNil(t, party.BahaiIDNumber)
		assert.Equal(t, bahaiID, *party.BahaiIDNumber)
		assert.NotZero(t, party.ID, "ID should not be zero")
	})

	// Test CreateParty with minimal data
	t.Run("CreatePartyMinimal", func(t *testing.T) {
		party, err := db.CreateParty("Jane Doe", nil, nil, nil, nil)
		require.NoError(t, err, "CreateParty should succeed")

		assert.Equal(t, "Jane Doe", party.Name)
		assert.Nil(t, party.EmailAddress)
	})

	// Test CreateParty with empty name
	t.Run("CreatePartyEmptyName", func(t *testing.T) {
		_, err := db.CreateParty("", nil, nil, nil, nil)
		assert.Error(t, err, "Expected error for empty name")
	})

	// Create a party for other tests
	email := "test@example.com"
	party, err := db.CreateParty("Test Party", &email, nil, nil, nil)
	require.NoError(t, err, "Failed to create test party")

	// Test Party retrieval
	t.Run("Party", func(t *testing.T) {
		retrieved, err := db.Party(party.ID)
		require.NoError(t, err, "Party retrieval should succeed")

		assert.Equal(t, party.ID, retrieved.ID)
		assert.Equal(t, party.Name, retrieved.Name)
	})

	// Test PartyByName
	t.Run("PartyByName", func(t *testing.T) {
		retrieved, err := db.PartyByName("Test Party")
		require.NoError(t, err, "PartyByName should succeed")

		assert.Equal(t, party.ID, retrieved.ID)
	})

	// Test PartyByEmail
	t.Run("PartyByEmail", func(t *testing.T) {
		retrieved, err := db.PartyByEmail("test@example.com")
		require.NoError(t, err, "PartyByEmail should succeed")

		assert.Equal(t, party.ID, retrieved.ID)
	})

	// Test ListParties
	t.Run("ListParties", func(t *testing.T) {
		parties, err := db.ListParties()
		require.NoError(t, err, "ListParties should succeed")

		assert.NotEmpty(t, parties, "Expected at least one party")
	})

	// Test SearchParties
	t.Run("SearchParties", func(t *testing.T) {
		parties, err := db.SearchParties("Test")
		require.NoError(t, err, "SearchParties should succeed")

		assert.NotEmpty(t, parties, "Expected at least one matching party")

		found := false
		for _, p := range parties {
			if p.ID == party.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "Test party should be found in search results")
	})

	// Test UpdateParty
	t.Run("UpdateParty", func(t *testing.T) {
		newEmail := "updated@example.com"
		updated, err := db.UpdateParty(party.ID, "Updated Party", &newEmail, nil, nil, nil)
		require.NoError(t, err, "UpdateParty should succeed")

		assert.Equal(t, "Updated Party", updated.Name)
		require.NotNil(t, updated.EmailAddress)
		assert.Equal(t, newEmail, *updated.EmailAddress)
	})

	// Test DeleteParty
	t.Run("DeleteParty", func(t *testing.T) {
		err := db.DeleteParty(party.ID)
		require.NoError(t, err, "DeleteParty should succeed")

		// Verify party is deleted
		_, err = db.Party(party.ID)
		assert.Error(t, err, "Expected error when retrieving deleted party")
	})
}

func TestBankAccountCRUD(t *testing.T) {
	db := testDB(t)

	// Test CreateBankAccount
	t.Run("CreateBankAccount", func(t *testing.T) {
		account, err := db.CreateBankAccount("Main Checking", AccountTypeChecking, CurrencyUSD, nil, "Primary checking account", true, Money{})
		require.NoError(t, err, "CreateBankAccount should succeed")

		assert.Equal(t, "Main Checking", account.Name)
		assert.Equal(t, AccountTypeChecking, account.AccountType)
		assert.Equal(t, CurrencyUSD, account.Currency)
		assert.Nil(t, account.ParentID)
		assert.True(t, account.IsActive)
		assert.NotZero(t, account.ID, "ID should not be zero")
	})

	// Test CreateBankAccount with empty name
	t.Run("CreateBankAccountEmptyName", func(t *testing.T) {
		_, err := db.CreateBankAccount("", AccountTypeChecking, CurrencyUSD, nil, "", true, Money{})
		assert.Error(t, err, "Expected error for empty name")
	})

	// Test CreateBankAccount earmark without parent
	t.Run("CreateEarmarkWithoutParent", func(t *testing.T) {
		_, err := db.CreateBankAccount("Orphan Earmark", AccountTypeEarmark, CurrencyUSD, nil, "", true, Money{})
		assert.Error(t, err, "Expected error for earmark without parent")
		assert.Contains(t, err.Error(), "must have a parent")
	})

	// Create accounts for hierarchy tests
	parentAccount, err := db.CreateBankAccount("Parent Account", AccountTypeChecking, CurrencyUSD, nil, "Parent account", true, Money{})
	require.NoError(t, err, "Failed to create parent account")

	// Test CreateBankAccount with parent
	t.Run("CreateBankAccountWithParent", func(t *testing.T) {
		childAccount, err := db.CreateBankAccount("Child Account", AccountTypeEarmark, CurrencyUSD, &parentAccount.ID, "Child account", true, Money{})
		require.NoError(t, err, "CreateBankAccount with parent should succeed")

		require.NotNil(t, childAccount.ParentID)
		assert.Equal(t, parentAccount.ID, *childAccount.ParentID)
	})

	// Test CreateBankAccount earmark with earmark parent (should fail)
	t.Run("CreateEarmarkWithEarmarkParent", func(t *testing.T) {
		// First create an earmark under the checking account
		earmark1, err := db.CreateBankAccount("Earmark 1", AccountTypeEarmark, CurrencyUSD, &parentAccount.ID, "", true, Money{})
		require.NoError(t, err)

		// Try to create another earmark under the first earmark (should fail)
		_, err = db.CreateBankAccount("Earmark 2", AccountTypeEarmark, CurrencyUSD, &earmark1.ID, "", true, Money{})
		assert.Error(t, err, "Expected error for earmark with earmark parent")
		assert.Contains(t, err.Error(), "checking or savings parent")
	})

	// Test CreateBankAccount sub-account with mismatched currency (should fail)
	t.Run("CreateSubAccountCurrencyMismatch", func(t *testing.T) {
		// Parent is USD, try to create CAD sub-account
		_, err := db.CreateBankAccount("CAD Earmark", AccountTypeEarmark, CurrencyCAD, &parentAccount.ID, "", true, Money{})
		assert.Error(t, err, "Expected error for currency mismatch")
		assert.Contains(t, err.Error(), "same currency as their parent")
	})

	// Test non-earmark account with parent (should fail)
	t.Run("CreateNonEarmarkWithParent", func(t *testing.T) {
		// Try to create a checking account with a parent (should fail)
		_, err := db.CreateBankAccount("Sub Checking", AccountTypeChecking, CurrencyUSD, &parentAccount.ID, "", true, Money{})
		assert.Error(t, err, "Expected error for non-earmark with parent")
		assert.Contains(t, err.Error(), "only earmark accounts can have a parent")

		// Try to create a savings account with a parent (should fail)
		_, err = db.CreateBankAccount("Sub Savings", AccountTypeSavings, CurrencyUSD, &parentAccount.ID, "", true, Money{})
		assert.Error(t, err, "Expected error for non-earmark with parent")
		assert.Contains(t, err.Error(), "only earmark accounts can have a parent")
	})

	// Test CreateBankAccount with invalid parent
	t.Run("CreateBankAccountInvalidParent", func(t *testing.T) {
		invalidParentID := ID(99999)
		_, err := db.CreateBankAccount("Invalid Parent", AccountTypeEarmark, CurrencyUSD, &invalidParentID, "", true, Money{})
		assert.Error(t, err, "Expected error for invalid parent ID")
	})

	// Test CreateBankAccount with opening balance
	t.Run("CreateBankAccountWithOpeningBalance", func(t *testing.T) {
		openingBalance := NewMoney(150000, CurrencyUSD) // $1,500.00
		account, err := db.CreateBankAccount("Opening Balance Account", AccountTypeChecking, CurrencyUSD, nil, "", true, openingBalance)
		require.NoError(t, err, "CreateBankAccount with opening balance should succeed")

		// Verify the opening balance transaction was created
		balance, err := db.AccountBalance(account.ID, false)
		require.NoError(t, err, "AccountBalance should succeed")
		assert.Equal(t, int64(150000), balance.Amount, "Balance should equal opening balance")
		assert.Equal(t, CurrencyUSD, balance.Currency, "Currency should match")

		// Verify the transaction exists with correct memo
		transactions, err := db.TransactionsForAccount(account.ID, false, nil)
		require.NoError(t, err, "TransactionsForAccount should succeed")
		require.Len(t, transactions, 1, "Should have exactly one transaction")
		assert.Equal(t, "Opening Balance", transactions[0].Memo, "Transaction memo should be 'Opening Balance'")
		assert.Equal(t, int64(150000), transactions[0].Amount, "Transaction amount should match opening balance")
	})

	// Test CreateBankAccount with opening balance currency mismatch
	t.Run("CreateBankAccountOpeningBalanceCurrencyMismatch", func(t *testing.T) {
		openingBalance := NewMoney(10000, CurrencyCAD) // CAD balance for USD account
		_, err := db.CreateBankAccount("Currency Mismatch", AccountTypeChecking, CurrencyUSD, nil, "", true, openingBalance)
		assert.Error(t, err, "Expected error for opening balance currency mismatch")
		assert.Contains(t, err.Error(), "does not match account currency")
	})

	// Test CreateBankAccount with negative opening balance
	t.Run("CreateBankAccountNegativeOpeningBalance", func(t *testing.T) {
		openingBalance := Money{Amount: -10000, Currency: CurrencyUSD}
		_, err := db.CreateBankAccount("Negative Balance", AccountTypeChecking, CurrencyUSD, nil, "", true, openingBalance)
		assert.Error(t, err, "Expected error for negative opening balance")
		assert.Contains(t, err.Error(), "cannot be negative")
	})

	// Test CreateBankAccount earmark with opening balance
	t.Run("CreateEarmarkWithOpeningBalance", func(t *testing.T) {
		openingBalance := NewMoney(50000, CurrencyUSD) // $500.00
		earmark, err := db.CreateBankAccount("Earmark With Balance", AccountTypeEarmark, CurrencyUSD, &parentAccount.ID, "", true, openingBalance)
		require.NoError(t, err, "CreateBankAccount earmark with opening balance should succeed")

		// Verify the opening balance transaction was created
		balance, err := db.AccountBalance(earmark.ID, false)
		require.NoError(t, err, "AccountBalance should succeed")
		assert.Equal(t, int64(50000), balance.Amount, "Balance should equal opening balance")
	})

	// Test BankAccount retrieval
	t.Run("BankAccount", func(t *testing.T) {
		retrieved, err := db.BankAccount(parentAccount.ID)
		require.NoError(t, err, "BankAccount retrieval should succeed")

		assert.Equal(t, parentAccount.ID, retrieved.ID)
		assert.Equal(t, parentAccount.Name, retrieved.Name)
	})

	// Test BankAccountByName
	t.Run("BankAccountByName", func(t *testing.T) {
		retrieved, err := db.BankAccountByName("Parent Account")
		require.NoError(t, err, "BankAccountByName should succeed")

		assert.Equal(t, parentAccount.ID, retrieved.ID)
	})

	// Test RootBankAccounts
	t.Run("RootBankAccounts", func(t *testing.T) {
		accounts, err := db.RootBankAccounts()
		require.NoError(t, err, "RootBankAccounts should succeed")

		assert.NotEmpty(t, accounts, "Expected at least one root account")

		// All accounts should have nil ParentID
		for _, account := range accounts {
			assert.Nil(t, account.ParentID, "Root account %s should have nil ParentID", account.Name)
		}
	})

	// Test ChildBankAccounts
	t.Run("ChildBankAccounts", func(t *testing.T) {
		children, err := db.ChildBankAccounts(parentAccount.ID)
		require.NoError(t, err, "ChildBankAccounts should succeed")

		assert.NotEmpty(t, children, "Expected at least one child account")

		// All children should have the correct ParentID
		for _, child := range children {
			require.NotNil(t, child.ParentID, "Child account %s should have ParentID", child.Name)
			assert.Equal(t, parentAccount.ID, *child.ParentID)
		}
	})

	// Test BankAccounts
	t.Run("BankAccounts", func(t *testing.T) {
		accounts, err := db.BankAccounts()
		require.NoError(t, err, "BankAccounts should succeed")

		assert.NotEmpty(t, accounts, "Expected at least one account")
	})

	// Test ActiveBankAccounts
	t.Run("ActiveBankAccounts", func(t *testing.T) {
		accounts, err := db.ActiveBankAccounts()
		require.NoError(t, err, "ActiveBankAccounts should succeed")

		// All accounts should be active
		for _, account := range accounts {
			assert.True(t, account.IsActive, "Account %s should be active", account.Name)
		}
	})

	// Test UpdateBankAccount
	t.Run("UpdateBankAccount", func(t *testing.T) {
		updated, err := db.UpdateBankAccount(parentAccount.ID, "Updated Account", AccountTypeSavings, CurrencyCAD, nil, "Updated description", false)
		require.NoError(t, err, "UpdateBankAccount should succeed")

		assert.Equal(t, "Updated Account", updated.Name)
		assert.Equal(t, AccountTypeSavings, updated.AccountType)
		assert.Equal(t, CurrencyCAD, updated.Currency)
		assert.False(t, updated.IsActive)
	})

	// Test DeactivateBankAccount
	t.Run("DeactivateBankAccount", func(t *testing.T) {
		// Create a new account to deactivate
		account, err := db.CreateBankAccount("To Deactivate", AccountTypeChecking, CurrencyUSD, nil, "", true, Money{})
		require.NoError(t, err, "Failed to create account")

		deactivated, err := db.DeactivateBankAccount(account.ID)
		require.NoError(t, err, "DeactivateBankAccount should succeed")

		assert.False(t, deactivated.IsActive, "Account should be inactive after deactivation")
	})

	// Test cycle prevention
	t.Run("CyclePrevention", func(t *testing.T) {
		// Try to make parent account a child of itself
		_, err := db.UpdateBankAccount(parentAccount.ID, "Cyclic", AccountTypeChecking, CurrencyUSD, &parentAccount.ID, "", true)
		assert.Error(t, err, "Expected error when making account its own parent")
	})
}

func TestReceiptCRUD(t *testing.T) {
	db := testDB(t)
	assembly, account, item, party, category := setupTestData(t, db)
	_ = assembly // assembly is set up but may not be used directly in this test
	_ = account
	_ = item
	_ = category

	soldAt := time.Now()

	// Test CreateReceipt
	t.Run("CreateReceipt", func(t *testing.T) {
		receipt, err := db.CreateReceipt(party.ID, soldAt)
		require.NoError(t, err, "CreateReceipt should succeed")

		assert.Equal(t, party.ID, receipt.CustomerID)
		assert.Equal(t, soldAt.Round(0), receipt.SoldAt)
		assert.NotEmpty(t, receipt.HumanID, "HumanID should not be empty")
		assert.Nil(t, receipt.TransactionID)
		assert.NotZero(t, receipt.ID, "ID should not be zero")
	})

	// Test CreateReceipt with invalid customer
	t.Run("CreateReceiptInvalidCustomer", func(t *testing.T) {
		_, err := db.CreateReceipt(99999, soldAt)
		assert.Error(t, err, "Expected error for invalid customer ID")
	})

	// Create a receipt for other tests with a slightly different timestamp
	receipt, err := db.CreateReceipt(party.ID, soldAt.Add(time.Millisecond))
	require.NoError(t, err, "Failed to create test receipt")

	// Test Receipt retrieval
	t.Run("Receipt", func(t *testing.T) {
		retrieved, err := db.Receipt(receipt.ID)
		require.NoError(t, err, "Receipt retrieval should succeed")

		assert.Equal(t, receipt.ID, retrieved.ID)
		assert.Equal(t, receipt.HumanID, retrieved.HumanID)
	})

	// Test ReceiptByHumanID
	t.Run("ReceiptByHumanID", func(t *testing.T) {
		retrieved, err := db.ReceiptByHumanID(receipt.HumanID)
		require.NoError(t, err, "ReceiptByHumanID should succeed")

		assert.Equal(t, receipt.ID, retrieved.ID)
	})

	// Test ReceiptsByCustomer
	t.Run("ReceiptsByCustomer", func(t *testing.T) {
		receipts, err := db.ReceiptsByCustomer(party.ID)
		require.NoError(t, err, "ReceiptsByCustomer should succeed")

		assert.NotEmpty(t, receipts, "Expected at least one receipt")

		found := false
		for _, r := range receipts {
			if r.ID == receipt.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "Test receipt should be found in customer receipts")
	})

	// Test UndepositedReceipts
	t.Run("UndepositedReceipts", func(t *testing.T) {
		receipts, err := db.UndepositedReceipts()
		require.NoError(t, err, "UndepositedReceipts should succeed")

		found := false
		for _, r := range receipts {
			if r.ID == receipt.ID {
				found = true
				assert.Nil(t, r.TransactionID, "Undeposited receipt should have nil TransactionID")
				break
			}
		}
		assert.True(t, found, "Test receipt should be found in undeposited receipts")
	})

	// Test Receipts
	t.Run("Receipts", func(t *testing.T) {
		receipts, err := db.Receipts()
		require.NoError(t, err, "Receipts should succeed")

		assert.NotEmpty(t, receipts, "Expected at least one receipt")
	})

	// Test UnassignReceiptFromTransaction (on unassigned receipt)
	t.Run("UnassignReceiptFromTransaction", func(t *testing.T) {
		unassigned, err := db.UnassignReceiptFromTransaction(receipt.ID)
		require.NoError(t, err, "UnassignReceiptFromTransaction should succeed")

		assert.Nil(t, unassigned.TransactionID)
	})

	// Test DeleteReceipt
	t.Run("DeleteReceipt", func(t *testing.T) {
		err := db.DeleteReceipt(receipt.ID)
		require.NoError(t, err, "DeleteReceipt should succeed")

		// Verify receipt is deleted
		_, err = db.Receipt(receipt.ID)
		assert.Error(t, err, "Expected error when retrieving deleted receipt")
	})
}

func TestTransactionCRUD(t *testing.T) {
	db := testDB(t)
	assembly, account, item, party, category := setupTestData(t, db)
	_ = assembly
	_ = item
	_ = category

	amount := NewMoney(10000, CurrencyUSD) // $100.00
	memo := "Test transaction"
	transactedAt := time.Now()

	// Test CreateDeposit
	t.Run("CreateDeposit", func(t *testing.T) {
		transaction, err := db.CreateDeposit(account.ID, amount, TransactionMethodElectronicTransfer, memo, transactedAt)
		require.NoError(t, err, "CreateDeposit should succeed")

		assert.Equal(t, account.ID, transaction.AccountID)
		assert.Equal(t, amount.Amount, transaction.Amount)
		assert.Equal(t, memo, transaction.Memo)
		require.NotNil(t, transaction.Method)
		assert.Equal(t, TransactionMethodElectronicTransfer, *transaction.Method)
		assert.Nil(t, transaction.PayeeID, "Deposit should have nil PayeeID")
		assert.NotZero(t, transaction.ID, "ID should not be zero")
	})

	// Test CreateDeposit with negative amount
	t.Run("CreateDepositNegativeAmount", func(t *testing.T) {
		negativeAmount := NewMoney(-1000, CurrencyUSD)
		_, err := db.CreateDeposit(account.ID, negativeAmount, TransactionMethodElectronicTransfer, memo, transactedAt)
		assert.Error(t, err, "Expected error for negative deposit amount")
	})

	// Test CreateDeposit with currency mismatch
	t.Run("CreateDepositCurrencyMismatch", func(t *testing.T) {
		cadAmount := NewMoney(1000, CurrencyCAD)
		_, err := db.CreateDeposit(account.ID, cadAmount, TransactionMethodElectronicTransfer, memo, transactedAt)
		assert.Error(t, err, "Expected error for currency mismatch")
	})

	// Test CreateWithdrawal
	t.Run("CreateWithdrawal", func(t *testing.T) {
		checkNumber := "1001"
		expenses := []ExpenseItem{
			{
				CategoryID:  category.ID,
				Description: strPtr("Office supplies"),
				Amount:      NewMoney(6000, CurrencyUSD), // $60.00
			},
			{
				CategoryID:  category.ID,
				Description: nil,
				Amount:      NewMoney(4000, CurrencyUSD), // $40.00
			},
		}
		transaction, err := db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, memo, transactedAt, &checkNumber, expenses)
		require.NoError(t, err, "CreateWithdrawal should succeed")

		assert.Equal(t, account.ID, transaction.AccountID)
		assert.Equal(t, int64(-10000), transaction.Amount, "Withdrawal amount should be negative sum of expenses")
		require.NotNil(t, transaction.PayeeID)
		assert.Equal(t, party.ID, *transaction.PayeeID)
		require.NotNil(t, transaction.CheckNumber)
		assert.Equal(t, checkNumber, *transaction.CheckNumber)
		require.NotNil(t, transaction.Method)
		assert.Equal(t, TransactionMethodCheck, *transaction.Method)
	})

	// Test CreateWithdrawal with negative amount
	t.Run("CreateWithdrawalNegativeAmount", func(t *testing.T) {
		negativeExpenses := []ExpenseItem{
			{
				CategoryID:  category.ID,
				Description: strPtr("Invalid expense"),
				Amount:      NewMoney(-1000, CurrencyUSD), // Negative amount should fail
			},
		}
		_, err := db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, memo, transactedAt, nil, negativeExpenses)
		assert.Error(t, err, "Expected error for negative expense amount")
	})

	// Test CreateWithdrawal with no expenses
	t.Run("CreateWithdrawalNoExpenses", func(t *testing.T) {
		_, err := db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, memo, transactedAt, nil, []ExpenseItem{})
		assert.Error(t, err, "Expected error for no expense items")
	})

	// Test receipt assignment workflow
	t.Run("ReceiptAssignmentWorkflow", func(t *testing.T) {
		// Create a receipt
		receipt, err := db.CreateReceipt(party.ID, transactedAt)
		require.NoError(t, err, "Failed to create receipt")

		// Create a deposit transaction
		deposit, err := db.CreateDeposit(account.ID, amount, TransactionMethodElectronicTransfer, "Deposit for receipt", transactedAt)
		require.NoError(t, err, "Failed to create deposit")

		// Assign receipt to transaction
		assigned, err := db.AssignReceiptToTransaction(receipt.ID, deposit.ID)
		require.NoError(t, err, "AssignReceiptToTransaction should succeed")

		require.NotNil(t, assigned.TransactionID)
		assert.Equal(t, deposit.ID, *assigned.TransactionID)

		// Test ReceiptsByTransaction
		receipts, err := db.ReceiptsByTransaction(deposit.ID)
		require.NoError(t, err, "ReceiptsByTransaction should succeed")

		require.Len(t, receipts, 1, "Expected exactly one receipt assigned to transaction")
		assert.Equal(t, receipt.ID, receipts[0].ID)

		// Try to assign receipt to withdrawal (should fail)
		withdrawalExpenses := []ExpenseItem{
			{
				CategoryID:  category.ID,
				Description: strPtr("Test expense"),
				Amount:      amount,
			},
		}
		withdrawal, err := db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "Test withdrawal", transactedAt, nil, withdrawalExpenses)
		require.NoError(t, err, "Failed to create withdrawal")

		_, err = db.AssignReceiptToTransaction(receipt.ID, withdrawal.ID)
		assert.Error(t, err, "Expected error when assigning receipt to withdrawal transaction")
	})
}

func TestExpenseValidation(t *testing.T) {
	db := testDB(t)
	assembly, account, _, party, category := setupTestData(t, db)
	_ = assembly

	transactedAt := time.Now()

	// Test currency mismatch between expenses and account
	t.Run("ExpenseCurrencyMismatch", func(t *testing.T) {
		// Create CAD account
		cadAccount, err := db.CreateBankAccount("CAD Account", AccountTypeChecking, CurrencyCAD, nil, "CAD test account", true, Money{})
		require.NoError(t, err, "Failed to create CAD account")

		// Try to create withdrawal with USD expenses on CAD account
		usdExpenses := []ExpenseItem{
			{
				CategoryID:  category.ID,
				Description: strPtr("USD expense on CAD account"),
				Amount:      NewMoney(1000, CurrencyUSD),
			},
		}
		_, err = db.CreateWithdrawal(cadAccount.ID, party.ID, TransactionMethodCheck, "Test", transactedAt, nil, usdExpenses)
		assert.Error(t, err, "Expected error for currency mismatch")
	})

	// Test mixed currencies in expenses
	t.Run("MixedCurrenciesInExpenses", func(t *testing.T) {
		category2, err := db.CreateCategory("Food")
		require.NoError(t, err, "Failed to create second category")

		mixedExpenses := []ExpenseItem{
			{
				CategoryID:  category.ID,
				Description: strPtr("USD expense"),
				Amount:      NewMoney(1000, CurrencyUSD),
			},
			{
				CategoryID:  category2.ID,
				Description: strPtr("CAD expense"),
				Amount:      NewMoney(1000, CurrencyCAD),
			},
		}
		_, err = db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "Test", transactedAt, nil, mixedExpenses)
		assert.Error(t, err, "Expected error for mixed currencies in expenses")
	})

	// Test valid multi-expense withdrawal
	t.Run("ValidMultiExpenseWithdrawal", func(t *testing.T) {
		category2, err := db.CreateCategory("Travel")
		require.NoError(t, err, "Failed to create category")

		validExpenses := []ExpenseItem{
			{
				CategoryID:  category.ID,
				Description: strPtr("Office supplies"),
				Amount:      NewMoney(2500, CurrencyUSD), // $25.00
			},
			{
				CategoryID:  category2.ID,
				Description: strPtr("Flight tickets"),
				Amount:      NewMoney(15000, CurrencyUSD), // $150.00
			},
			{
				CategoryID:  category.ID,
				Description: nil,                        // Test nil description
				Amount:      NewMoney(750, CurrencyUSD), // $7.50
			},
		}

		transaction, err := db.CreateWithdrawal(account.ID, party.ID, TransactionMethodElectronicTransfer, "Multi-category expense", transactedAt, nil, validExpenses)
		require.NoError(t, err, "Valid multi-expense withdrawal should succeed")

		// Verify total amount
		expectedTotal := int64(2500 + 15000 + 750)
		assert.Equal(t, -expectedTotal, transaction.Amount, "Transaction amount should be negative sum of all expenses")

		// Verify expenses were created in database
		var expenseCount int
		err = db.conn.Get(&expenseCount, "SELECT COUNT(*) FROM expenses WHERE transaction_id = ?", transaction.ID)
		require.NoError(t, err, "Failed to count expenses")
		assert.Equal(t, 3, expenseCount, "Should have created 3 expense records")

		// Verify specific expense data
		var expenses []struct {
			CategoryID  ID      `db:"category_id"`
			Description *string `db:"description"`
			Amount      int64   `db:"amount"`
			Currency    string  `db:"currency"`
		}
		err = db.conn.Select(&expenses, "SELECT category_id, description, amount, currency FROM expenses WHERE transaction_id = ? ORDER BY amount", transaction.ID)
		require.NoError(t, err, "Failed to retrieve expenses")

		assert.Equal(t, int64(750), expenses[0].Amount)
		assert.Nil(t, expenses[0].Description)
		assert.Equal(t, int64(2500), expenses[1].Amount)
		assert.Equal(t, "Office supplies", *expenses[1].Description)
		assert.Equal(t, int64(15000), expenses[2].Amount)
		assert.Equal(t, "Flight tickets", *expenses[2].Description)
	})

	// Test zero amount expense
	t.Run("ZeroAmountExpense", func(t *testing.T) {
		zeroExpenses := []ExpenseItem{
			{
				CategoryID:  category.ID,
				Description: strPtr("Zero amount"),
				Amount:      NewMoney(0, CurrencyUSD),
			},
		}
		_, err := db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "Test", transactedAt, nil, zeroExpenses)
		assert.Error(t, err, "Expected error for zero amount expense")
	})

	// Test non-existent category
	t.Run("NonExistentCategory", func(t *testing.T) {
		invalidExpenses := []ExpenseItem{
			{
				CategoryID:  ID(99999), // Non-existent category
				Description: strPtr("Invalid category"),
				Amount:      NewMoney(1000, CurrencyUSD),
			},
		}
		_, err := db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "Test", transactedAt, nil, invalidExpenses)
		assert.Error(t, err, "Expected error for non-existent category")
	})
}

func TestLedgerFunctionality(t *testing.T) {
	db := testDB(t)
	assembly, account, item, party, category := setupTestData(t, db)
	_ = assembly
	_ = item

	// Create a child account for testing hierarchical features
	childAccount, err := db.CreateBankAccount("Child Account", AccountTypeEarmark, CurrencyUSD, &account.ID, "Child of main account", true, Money{})
	require.NoError(t, err, "Failed to create child account")

	// Create some test transactions with specific dates for testing
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// Transaction 1: Deposit $100 on Jan 1
	deposit1, err := db.CreateDeposit(account.ID, NewMoney(10000, CurrencyUSD), TransactionMethodElectronicTransfer, "Initial deposit", baseTime)
	require.NoError(t, err, "Failed to create deposit1")

	// Transaction 2: Withdrawal $30 on Jan 2
	expenses1 := []ExpenseItem{
		{CategoryID: category.ID, Description: strPtr("Office supplies"), Amount: NewMoney(3000, CurrencyUSD)},
	}
	withdrawal1, err := db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "Office supplies", baseTime.Add(24*time.Hour), strPtr("001"), expenses1)
	require.NoError(t, err, "Failed to create withdrawal1")

	// Transaction 3: Deposit $50 to child account on Jan 3
	_, err = db.CreateDeposit(childAccount.ID, NewMoney(5000, CurrencyUSD), TransactionMethodElectronicTransfer, "Child account deposit", baseTime.Add(48*time.Hour))
	require.NoError(t, err, "Failed to create deposit2")

	// Transaction 4: Withdrawal $20 from main account on Jan 4
	expenses2 := []ExpenseItem{
		{CategoryID: category.ID, Description: strPtr("Food"), Amount: NewMoney(2000, CurrencyUSD)},
	}
	withdrawal2, err := db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "Food", baseTime.Add(72*time.Hour), strPtr("002"), expenses2)
	require.NoError(t, err, "Failed to create withdrawal2")

	// Test AccountBalance
	t.Run("AccountBalance", func(t *testing.T) {
		// Main account balance (excluding subaccounts)
		balance, err := db.AccountBalance(account.ID, false)
		require.NoError(t, err, "AccountBalance should succeed")
		assert.Equal(t, int64(5000), balance.Amount, "Main account balance should be $50 (100 - 30 - 20)")

		// Main account balance (including subaccounts)
		balanceWithSub, err := db.AccountBalance(account.ID, true)
		require.NoError(t, err, "AccountBalance with subaccounts should succeed")
		assert.Equal(t, int64(10000), balanceWithSub.Amount, "Total balance should be $100 (50 + 50)")

		// Child account balance
		childBalance, err := db.AccountBalance(childAccount.ID, false)
		require.NoError(t, err, "Child AccountBalance should succeed")
		assert.Equal(t, int64(5000), childBalance.Amount, "Child account balance should be $50")
	})

	// Test AccountBalanceAsOf
	t.Run("AccountBalanceAsOf", func(t *testing.T) {
		// Balance as of Jan 1 (after first deposit)
		balanceJan1, err := db.AccountBalanceAsOf(account.ID, baseTime.Add(1*time.Hour), false)
		require.NoError(t, err, "AccountBalanceAsOf should succeed")
		assert.Equal(t, int64(10000), balanceJan1.Amount, "Balance on Jan 1 should be $100")

		// Balance as of Jan 2 (after first withdrawal)
		balanceJan2, err := db.AccountBalanceAsOf(account.ID, baseTime.Add(25*time.Hour), false)
		require.NoError(t, err, "AccountBalanceAsOf should succeed")
		assert.Equal(t, int64(7000), balanceJan2.Amount, "Balance on Jan 2 should be $70")

		// Balance as of Jan 3 including subaccounts
		balanceJan3, err := db.AccountBalanceAsOf(account.ID, baseTime.Add(49*time.Hour), true)
		require.NoError(t, err, "AccountBalanceAsOf with subaccounts should succeed")
		assert.Equal(t, int64(12000), balanceJan3.Amount, "Balance on Jan 3 with subaccounts should be $120")
	})

	// Test TransactionsForAccount
	t.Run("TransactionsForAccount", func(t *testing.T) {
		// Get all transactions for main account only
		transactions, err := db.TransactionsForAccount(account.ID, false, nil)
		require.NoError(t, err, "TransactionsForAccount should succeed")
		assert.Len(t, transactions, 3, "Main account should have 3 transactions")

		// Verify transaction order (oldest first)
		assert.Equal(t, deposit1.ID, transactions[0].ID, "First transaction should be the initial deposit")
		assert.Equal(t, withdrawal1.ID, transactions[1].ID, "Second transaction should be the first withdrawal")
		assert.Equal(t, withdrawal2.ID, transactions[2].ID, "Third transaction should be the second withdrawal")

		// Get all transactions including subaccounts
		allTransactions, err := db.TransactionsForAccount(account.ID, true, nil)
		require.NoError(t, err, "TransactionsForAccount with subaccounts should succeed")
		assert.Len(t, allTransactions, 4, "All transactions should be 4")

		// Test with date filtering
		startDate := baseTime.Add(24 * time.Hour)
		endDate := baseTime.Add(49 * time.Hour)
		options := &LedgerOptions{
			StartDate: &startDate,
			EndDate:   &endDate,
		}
		filteredTransactions, err := db.TransactionsForAccount(account.ID, true, options)
		require.NoError(t, err, "TransactionsForAccount with date filter should succeed")
		assert.Len(t, filteredTransactions, 2, "Should have 2 transactions in date range")
	})

	// Test AccountLedger
	t.Run("AccountLedger", func(t *testing.T) {
		// Get ledger for main account only
		ledger, err := db.AccountLedger(account.ID, &LedgerOptions{IncludeSubaccounts: false})
		require.NoError(t, err, "AccountLedger should succeed")
		assert.Len(t, ledger, 3, "Ledger should have 3 entries")

		// Verify running balances
		assert.Equal(t, int64(10000), ledger[0].RunningBalance.Amount, "First entry balance should be $100")
		assert.Equal(t, int64(7000), ledger[1].RunningBalance.Amount, "Second entry balance should be $70")
		assert.Equal(t, int64(5000), ledger[2].RunningBalance.Amount, "Third entry balance should be $50")

		// Verify enriched data
		assert.Equal(t, 1, ledger[1].ExpenseCount, "Withdrawal should have 1 expense")
		assert.Equal(t, party.Name, *ledger[1].PayeeName, "Payee name should be set")

		// Get ledger including subaccounts
		fullLedger, err := db.AccountLedger(account.ID, &LedgerOptions{IncludeSubaccounts: true})
		require.NoError(t, err, "AccountLedger with subaccounts should succeed")
		assert.Len(t, fullLedger, 4, "Full ledger should have 4 entries")

		// Verify final running balance includes subaccounts
		assert.Equal(t, int64(10000), fullLedger[3].RunningBalance.Amount, "Final balance should be $100")
	})

	// Test AccountTransactionCount
	t.Run("AccountTransactionCount", func(t *testing.T) {
		count, err := db.AccountTransactionCount(account.ID, false, nil)
		require.NoError(t, err, "AccountTransactionCount should succeed")
		assert.Equal(t, 3, count, "Main account should have 3 transactions")

		countWithSub, err := db.AccountTransactionCount(account.ID, true, nil)
		require.NoError(t, err, "AccountTransactionCount with subaccounts should succeed")
		assert.Equal(t, 4, countWithSub, "Total transactions should be 4")

		// Test with date filtering
		startDate := baseTime.Add(24 * time.Hour)
		options := &LedgerOptions{
			StartDate: &startDate,
		}
		filteredCount, err := db.AccountTransactionCount(account.ID, true, options)
		require.NoError(t, err, "AccountTransactionCount with date filter should succeed")
		assert.Equal(t, 3, filteredCount, "Should have 3 transactions after Jan 1")
	})

	// Test AllAccountBalances
	t.Run("AllAccountBalances", func(t *testing.T) {
		balances, err := db.AllAccountBalances()
		require.NoError(t, err, "AllAccountBalances should succeed")

		assert.Contains(t, balances, account.ID, "Should include main account")
		assert.Contains(t, balances, childAccount.ID, "Should include child account")

		assert.Equal(t, int64(5000), balances[account.ID].Amount, "Main account balance should be $50")
		assert.Equal(t, int64(5000), balances[childAccount.ID].Amount, "Child account balance should be $50")
	})

	// Test LastTransactionDate
	t.Run("LastTransactionDate", func(t *testing.T) {
		lastDate, err := db.LastTransactionDate(account.ID, false)
		require.NoError(t, err, "LastTransactionDate should succeed")
		require.NotNil(t, lastDate, "Last transaction date should not be nil")
		assert.Equal(t, baseTime.Add(72*time.Hour).Unix(), lastDate.Unix(), "Last transaction should be on Jan 4")

		lastDateWithSub, err := db.LastTransactionDate(account.ID, true)
		require.NoError(t, err, "LastTransactionDate with subaccounts should succeed")
		require.NotNil(t, lastDateWithSub, "Last transaction date with subaccounts should not be nil")
		assert.Equal(t, baseTime.Add(72*time.Hour).Unix(), lastDateWithSub.Unix(), "Last transaction should still be on Jan 4")
	})

	// Test pagination
	t.Run("LedgerPagination", func(t *testing.T) {
		// Get first 2 transactions
		options := &LedgerOptions{
			IncludeSubaccounts: true,
			Limit:              &[]int{2}[0],
			Offset:             &[]int{0}[0],
		}
		page1, err := db.AccountLedger(account.ID, options)
		require.NoError(t, err, "First page should succeed")
		assert.Len(t, page1, 2, "First page should have 2 entries")

		// Get next 2 transactions
		options.Offset = &[]int{2}[0]
		page2, err := db.AccountLedger(account.ID, options)
		require.NoError(t, err, "Second page should succeed")
		assert.Len(t, page2, 2, "Second page should have 2 entries")

		// Verify no overlap
		assert.NotEqual(t, page1[0].Transaction.ID, page2[0].Transaction.ID, "Pages should not overlap")
	})
}

func TestLedgerEdgeCases(t *testing.T) {
	db := testDB(t)
	assembly, account, _, party, _ := setupTestData(t, db)
	_ = assembly

	// Test with no transactions
	t.Run("EmptyAccount", func(t *testing.T) {
		balance, err := db.AccountBalance(account.ID, false)
		require.NoError(t, err, "Empty account balance should succeed")
		assert.Equal(t, int64(0), balance.Amount, "Empty account balance should be 0")

		ledger, err := db.AccountLedger(account.ID, nil)
		require.NoError(t, err, "Empty account ledger should succeed")
		assert.Len(t, ledger, 0, "Empty ledger should have no entries")

		lastDate, err := db.LastTransactionDate(account.ID, false)
		require.NoError(t, err, "LastTransactionDate for empty account should succeed")
		assert.Nil(t, lastDate, "Last transaction date should be nil for empty account")
	})

	// Test with receipts for enrichment
	t.Run("LedgerEnrichmentWithReceipts", func(t *testing.T) {
		// Create a receipt and deposit
		receipt, err := db.CreateReceipt(party.ID, time.Now())
		require.NoError(t, err, "Failed to create receipt")

		deposit, err := db.CreateDeposit(account.ID, NewMoney(5000, CurrencyUSD), TransactionMethodElectronicTransfer, "Test deposit", time.Now())
		require.NoError(t, err, "Failed to create deposit")

		// Assign receipt to deposit
		_, err = db.AssignReceiptToTransaction(receipt.ID, deposit.ID)
		require.NoError(t, err, "Failed to assign receipt to transaction")

		// Get ledger and verify enrichment
		ledger, err := db.AccountLedger(account.ID, nil)
		require.NoError(t, err, "AccountLedger should succeed")
		require.Len(t, ledger, 1, "Should have one ledger entry")

		assert.Equal(t, 1, ledger[0].ReceiptCount, "Should have 1 receipt")
		assert.Equal(t, party.Name, *ledger[0].CustomerName, "Customer name should be set")
	})

	// Test currency consistency
	t.Run("CurrencyConsistency", func(t *testing.T) {
		// Create CAD account
		cadAccount, err := db.CreateBankAccount("CAD Account", AccountTypeChecking, CurrencyCAD, nil, "CAD account", true, Money{})
		require.NoError(t, err, "Failed to create CAD account")

		// Create CAD transaction
		_, err = db.CreateDeposit(cadAccount.ID, NewMoney(10000, CurrencyCAD), TransactionMethodElectronicTransfer, "CAD deposit", time.Now())
		require.NoError(t, err, "Failed to create CAD deposit")

		// Verify balance has correct currency
		balance, err := db.AccountBalance(cadAccount.ID, false)
		require.NoError(t, err, "CAD account balance should succeed")
		assert.Equal(t, CurrencyCAD, balance.Currency, "Balance should have CAD currency")

		// Verify ledger has correct currency
		ledger, err := db.AccountLedger(cadAccount.ID, nil)
		require.NoError(t, err, "CAD account ledger should succeed")
		require.Len(t, ledger, 1, "Should have one ledger entry")
		assert.Equal(t, CurrencyCAD, ledger[0].RunningBalance.Currency, "Running balance should have CAD currency")
	})
}

func TestReconciliationCRUD(t *testing.T) {
	// Test StartReconciliation
	t.Run("StartReconciliation", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
		statementBalance := NewMoney(10000, CurrencyUSD)

		reconciliation, err := db.StartReconciliation(account.ID, statementDate, statementBalance)
		require.NoError(t, err, "StartReconciliation should succeed")

		assert.Equal(t, account.ID, reconciliation.AccountID)
		assert.Equal(t, statementDate.Unix(), reconciliation.StatementDate.Unix())
		assert.Equal(t, statementBalance.Amount, reconciliation.StatementBalance.Amount)
		assert.Equal(t, statementBalance.Currency, reconciliation.StatementBalance.Currency)
		assert.Equal(t, ReconciliationStatusInProgress, reconciliation.Status)
		assert.Nil(t, reconciliation.CompletedAt)
		assert.NotZero(t, reconciliation.ID)
	})

	// Test StartReconciliation on subaccount (should fail)
	t.Run("StartReconciliationOnSubaccount", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
		statementBalance := NewMoney(10000, CurrencyUSD)

		childAccount, err := db.CreateBankAccount("Child Account", AccountTypeEarmark, CurrencyUSD, &account.ID, "Child", true, Money{})
		require.NoError(t, err, "Failed to create child account")

		_, err = db.StartReconciliation(childAccount.ID, statementDate, statementBalance)
		assert.Error(t, err, "Expected error for reconciliation on subaccount")
		assert.Contains(t, err.Error(), "root-level")
	})

	// Test StartReconciliation with currency mismatch
	t.Run("StartReconciliationCurrencyMismatch", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
		cadBalance := NewMoney(10000, CurrencyCAD)

		_, err := db.StartReconciliation(account.ID, statementDate, cadBalance)
		assert.Error(t, err, "Expected error for currency mismatch")
	})

	// Test duplicate in-progress reconciliation
	t.Run("DuplicateInProgressReconciliation", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
		statementBalance := NewMoney(10000, CurrencyUSD)

		_, err := db.StartReconciliation(account.ID, statementDate, statementBalance)
		require.NoError(t, err, "First reconciliation should succeed")

		_, err = db.StartReconciliation(account.ID, statementDate, statementBalance)
		assert.Error(t, err, "Expected error for duplicate in-progress reconciliation")
		assert.Contains(t, err.Error(), "in-progress")
	})

	// Test Reconciliation retrieval
	t.Run("ReconciliationRetrieval", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
		statementBalance := NewMoney(10000, CurrencyUSD)

		reconciliation, err := db.StartReconciliation(account.ID, statementDate, statementBalance)
		require.NoError(t, err, "Failed to start reconciliation")

		retrieved, err := db.Reconciliation(reconciliation.ID)
		require.NoError(t, err, "Reconciliation retrieval should succeed")

		assert.Equal(t, reconciliation.ID, retrieved.ID)
		assert.Equal(t, reconciliation.AccountID, retrieved.AccountID)
		assert.Equal(t, reconciliation.StatementBalance.Amount, retrieved.StatementBalance.Amount)
	})

	// Test InProgressReconciliation
	t.Run("InProgressReconciliation", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
		statementBalance := NewMoney(10000, CurrencyUSD)

		reconciliation, err := db.StartReconciliation(account.ID, statementDate, statementBalance)
		require.NoError(t, err, "Failed to start reconciliation")

		inProgress, err := db.InProgressReconciliation(account.ID)
		require.NoError(t, err, "InProgressReconciliation should succeed")

		assert.Equal(t, reconciliation.ID, inProgress.ID)
	})

	// Test Reconciliations list
	t.Run("ReconciliationsList", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
		statementBalance := NewMoney(10000, CurrencyUSD)

		// Create and cancel a reconciliation
		reconciliation1, err := db.StartReconciliation(account.ID, statementDate, statementBalance)
		require.NoError(t, err, "Failed to start reconciliation")
		_, err = db.CancelReconciliation(reconciliation1.ID)
		require.NoError(t, err, "Failed to cancel reconciliation")

		// Create another one
		_, err = db.StartReconciliation(account.ID, statementDate.AddDate(0, 1, 0), statementBalance)
		require.NoError(t, err, "Failed to start second reconciliation")

		reconciliations, err := db.Reconciliations(account.ID)
		require.NoError(t, err, "Reconciliations should succeed")

		assert.Equal(t, 2, len(reconciliations), "Should have 2 reconciliations")
	})

	// Test CancelReconciliation
	t.Run("CancelReconciliation", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
		statementBalance := NewMoney(10000, CurrencyUSD)

		reconciliation, err := db.StartReconciliation(account.ID, statementDate, statementBalance)
		require.NoError(t, err, "Failed to start reconciliation")

		cancelled, err := db.CancelReconciliation(reconciliation.ID)
		require.NoError(t, err, "CancelReconciliation should succeed")

		assert.Equal(t, ReconciliationStatusCancelled, cancelled.Status)

		// Verify can start a new one after cancellation
		_, err = db.StartReconciliation(account.ID, statementDate, statementBalance)
		require.NoError(t, err, "Should be able to start new reconciliation after cancellation")
	})

	// Test CancelReconciliation on completed (should fail)
	t.Run("CancelCompletedReconciliation", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
		statementBalance := NewMoney(10000, CurrencyUSD)

		// Create transaction that matches statement balance
		_, err := db.CreateDeposit(account.ID, statementBalance, TransactionMethodElectronicTransfer, "Test deposit", statementDate.Add(-24*time.Hour))
		require.NoError(t, err, "Failed to create deposit")

		reconciliation, err := db.StartReconciliation(account.ID, statementDate, statementBalance)
		require.NoError(t, err, "Failed to start reconciliation")

		// Clear all transactions
		uncleared, err := db.UnclearedTransactions(account.ID, statementDate)
		require.NoError(t, err, "Failed to get uncleared transactions")
		for _, tx := range uncleared {
			err = db.ClearTransaction(reconciliation.ID, tx.ID)
			require.NoError(t, err, "Failed to clear transaction")
		}

		// Complete the reconciliation
		completed, err := db.CompleteReconciliation(reconciliation.ID)
		require.NoError(t, err, "CompleteReconciliation should succeed")
		assert.Equal(t, ReconciliationStatusCompleted, completed.Status)

		// Try to cancel it
		_, err = db.CancelReconciliation(completed.ID)
		assert.Error(t, err, "Expected error when cancelling completed reconciliation")
	})

	// Test ClearTransaction and UnclearTransaction
	t.Run("ClearAndUnclearTransaction", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		// Create a deposit
		deposit, err := db.CreateDeposit(account.ID, NewMoney(5000, CurrencyUSD), TransactionMethodElectronicTransfer, "Test deposit", statementDate.Add(-48*time.Hour))
		require.NoError(t, err, "Failed to create deposit")

		reconciliation, err := db.StartReconciliation(account.ID, statementDate, NewMoney(5000, CurrencyUSD))
		require.NoError(t, err, "Failed to start reconciliation")

		// Clear the transaction
		err = db.ClearTransaction(reconciliation.ID, deposit.ID)
		require.NoError(t, err, "ClearTransaction should succeed")

		// Verify it's cleared
		clearedTxs, err := db.ClearedTransactions(reconciliation.ID)
		require.NoError(t, err, "ClearedTransactions should succeed")
		assert.Len(t, clearedTxs, 1, "Should have 1 cleared transaction")
		assert.Equal(t, deposit.ID, clearedTxs[0].ID)

		// Unclear it
		err = db.UnclearTransaction(deposit.ID)
		require.NoError(t, err, "UnclearTransaction should succeed")

		// Verify it's uncleared
		clearedTxs, err = db.ClearedTransactions(reconciliation.ID)
		require.NoError(t, err, "ClearedTransactions should succeed")
		assert.Len(t, clearedTxs, 0, "Should have no cleared transactions")
	})

	// Test ClearTransaction validation
	t.Run("ClearTransactionValidation", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		// Create account2 (different root account)
		account2, err := db.CreateBankAccount("Account 2", AccountTypeChecking, CurrencyUSD, nil, "", true, Money{})
		require.NoError(t, err, "Failed to create account2")

		// Create deposit on account2
		deposit, err := db.CreateDeposit(account2.ID, NewMoney(5000, CurrencyUSD), TransactionMethodElectronicTransfer, "Test", statementDate.Add(-24*time.Hour))
		require.NoError(t, err, "Failed to create deposit")

		reconciliation, err := db.StartReconciliation(account.ID, statementDate, NewMoney(5000, CurrencyUSD))
		require.NoError(t, err, "Failed to start reconciliation")

		// Try to clear transaction from different account
		err = db.ClearTransaction(reconciliation.ID, deposit.ID)
		assert.Error(t, err, "Expected error when clearing transaction from different account")

		// Create transaction after statement date
		futureDeposit, err := db.CreateDeposit(account.ID, NewMoney(1000, CurrencyUSD), TransactionMethodElectronicTransfer, "Future", statementDate.Add(24*time.Hour))
		require.NoError(t, err, "Failed to create future deposit")

		err = db.ClearTransaction(reconciliation.ID, futureDeposit.ID)
		assert.Error(t, err, "Expected error when clearing transaction after statement date")
	})

	// Test ClearTransaction on child account
	t.Run("ClearTransactionOnChildAccount", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		childAccount, err := db.CreateBankAccount("Earmark", AccountTypeEarmark, CurrencyUSD, &account.ID, "Earmark fund", true, Money{})
		require.NoError(t, err, "Failed to create child account")

		// Create deposit on child account
		childDeposit, err := db.CreateDeposit(childAccount.ID, NewMoney(3000, CurrencyUSD), TransactionMethodElectronicTransfer, "Child deposit", statementDate.Add(-24*time.Hour))
		require.NoError(t, err, "Failed to create child deposit")

		reconciliation, err := db.StartReconciliation(account.ID, statementDate, NewMoney(3000, CurrencyUSD))
		require.NoError(t, err, "Failed to start reconciliation")

		// Should be able to clear transaction from child account
		err = db.ClearTransaction(reconciliation.ID, childDeposit.ID)
		require.NoError(t, err, "Should be able to clear transaction from child account")
	})

	// Test ClearedBalance
	t.Run("ClearedBalance", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		// Create deposits
		deposit1, err := db.CreateDeposit(account.ID, NewMoney(6000, CurrencyUSD), TransactionMethodElectronicTransfer, "Deposit 1", statementDate.Add(-72*time.Hour))
		require.NoError(t, err, "Failed to create deposit")

		deposit2, err := db.CreateDeposit(account.ID, NewMoney(4000, CurrencyUSD), TransactionMethodElectronicTransfer, "Deposit 2", statementDate.Add(-48*time.Hour))
		require.NoError(t, err, "Failed to create deposit")

		reconciliation, err := db.StartReconciliation(account.ID, statementDate, NewMoney(10000, CurrencyUSD))
		require.NoError(t, err, "Failed to start reconciliation")

		// Clear both transactions
		err = db.ClearTransaction(reconciliation.ID, deposit1.ID)
		require.NoError(t, err, "Failed to clear deposit1")
		err = db.ClearTransaction(reconciliation.ID, deposit2.ID)
		require.NoError(t, err, "Failed to clear deposit2")

		// Check cleared balance
		clearedBalance, err := db.ClearedBalance(reconciliation.ID)
		require.NoError(t, err, "ClearedBalance should succeed")
		assert.Equal(t, int64(10000), clearedBalance.Amount, "Cleared balance should be $100")
		assert.Equal(t, CurrencyUSD, clearedBalance.Currency)
	})

	// Test UnclearedTransactions
	t.Run("UnclearedTransactions", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		// Create deposits
		deposit1, err := db.CreateDeposit(account.ID, NewMoney(2000, CurrencyUSD), TransactionMethodElectronicTransfer, "Uncleared 1", statementDate.Add(-24*time.Hour))
		require.NoError(t, err, "Failed to create deposit")

		deposit2, err := db.CreateDeposit(account.ID, NewMoney(3000, CurrencyUSD), TransactionMethodElectronicTransfer, "Uncleared 2", statementDate.Add(-12*time.Hour))
		require.NoError(t, err, "Failed to create deposit")

		// Create deposit after statement date (should not appear)
		_, err = db.CreateDeposit(account.ID, NewMoney(1000, CurrencyUSD), TransactionMethodElectronicTransfer, "After statement", statementDate.Add(24*time.Hour))
		require.NoError(t, err, "Failed to create future deposit")

		uncleared, err := db.UnclearedTransactions(account.ID, statementDate)
		require.NoError(t, err, "UnclearedTransactions should succeed")

		assert.Len(t, uncleared, 2, "Should have 2 uncleared transactions")

		// Find our test transactions
		found1, found2 := false, false
		for _, tx := range uncleared {
			if tx.ID == deposit1.ID {
				found1 = true
			}
			if tx.ID == deposit2.ID {
				found2 = true
			}
		}
		assert.True(t, found1, "Should include deposit1")
		assert.True(t, found2, "Should include deposit2")
	})

	// Test CompleteReconciliation
	t.Run("CompleteReconciliation", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		// Create deposit
		deposit, err := db.CreateDeposit(account.ID, NewMoney(7500, CurrencyUSD), TransactionMethodElectronicTransfer, "For completion", statementDate.Add(-24*time.Hour))
		require.NoError(t, err, "Failed to create deposit")

		reconciliation, err := db.StartReconciliation(account.ID, statementDate, NewMoney(7500, CurrencyUSD))
		require.NoError(t, err, "Failed to start reconciliation")

		// Clear the deposit
		err = db.ClearTransaction(reconciliation.ID, deposit.ID)
		require.NoError(t, err, "Failed to clear transaction")

		// Complete the reconciliation
		completed, err := db.CompleteReconciliation(reconciliation.ID)
		require.NoError(t, err, "CompleteReconciliation should succeed")

		assert.Equal(t, ReconciliationStatusCompleted, completed.Status)
		assert.NotNil(t, completed.CompletedAt)

		// Verify LastCompletedReconciliation
		last, err := db.LastCompletedReconciliation(account.ID)
		require.NoError(t, err, "LastCompletedReconciliation should succeed")
		assert.Equal(t, completed.ID, last.ID)
	})

	// Test CompleteReconciliation with balance mismatch
	t.Run("CompleteReconciliationBalanceMismatch", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		// Create deposit
		deposit, err := db.CreateDeposit(account.ID, NewMoney(5000, CurrencyUSD), TransactionMethodElectronicTransfer, "Mismatch test", statementDate.Add(-24*time.Hour))
		require.NoError(t, err, "Failed to create deposit")

		// Start reconciliation expecting $100 but we only have $50
		reconciliation, err := db.StartReconciliation(account.ID, statementDate, NewMoney(10000, CurrencyUSD))
		require.NoError(t, err, "Failed to start reconciliation")

		// Clear transaction
		err = db.ClearTransaction(reconciliation.ID, deposit.ID)
		require.NoError(t, err, "Failed to clear transaction")

		// Try to complete - should fail due to balance mismatch
		_, err = db.CompleteReconciliation(reconciliation.ID)
		assert.Error(t, err, "Expected error for balance mismatch")
		assert.Contains(t, err.Error(), "does not match")
	})

	// Test UndoReconciliation
	t.Run("UndoReconciliation", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		// Create and complete a reconciliation
		deposit, err := db.CreateDeposit(account.ID, NewMoney(8000, CurrencyUSD), TransactionMethodElectronicTransfer, "For undo", statementDate.Add(-24*time.Hour))
		require.NoError(t, err, "Failed to create deposit")

		reconciliation, err := db.StartReconciliation(account.ID, statementDate, NewMoney(8000, CurrencyUSD))
		require.NoError(t, err, "Failed to start reconciliation")

		err = db.ClearTransaction(reconciliation.ID, deposit.ID)
		require.NoError(t, err, "Failed to clear transaction")

		completed, err := db.CompleteReconciliation(reconciliation.ID)
		require.NoError(t, err, "Failed to complete reconciliation")

		// Undo the reconciliation
		err = db.UndoReconciliation(completed.ID)
		require.NoError(t, err, "UndoReconciliation should succeed")

		// Verify status changed to cancelled
		undone, err := db.Reconciliation(completed.ID)
		require.NoError(t, err, "Failed to get undone reconciliation")
		assert.Equal(t, ReconciliationStatusCancelled, undone.Status)

		// Verify transaction is uncleared
		var tx Transaction
		err = db.conn.Get(&tx, "SELECT * FROM transactions WHERE id = ?", deposit.ID)
		require.NoError(t, err, "Failed to get transaction")
		assert.Nil(t, tx.ReconciliationID, "Transaction should be uncleared after undo")
	})

	// Test UndoReconciliation only works on most recent
	t.Run("UndoReconciliationOnlyMostRecent", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		// Create and complete first reconciliation
		deposit1, err := db.CreateDeposit(account.ID, NewMoney(1000, CurrencyUSD), TransactionMethodElectronicTransfer, "First", statementDate.Add(-48*time.Hour))
		require.NoError(t, err, "Failed to create deposit")

		recon1, err := db.StartReconciliation(account.ID, statementDate.Add(-24*time.Hour), NewMoney(1000, CurrencyUSD))
		require.NoError(t, err, "Failed to start reconciliation")

		err = db.ClearTransaction(recon1.ID, deposit1.ID)
		require.NoError(t, err, "Failed to clear transaction")

		completed1, err := db.CompleteReconciliation(recon1.ID)
		require.NoError(t, err, "Failed to complete reconciliation")

		// Create and complete second reconciliation
		deposit2, err := db.CreateDeposit(account.ID, NewMoney(2000, CurrencyUSD), TransactionMethodElectronicTransfer, "Second", statementDate.Add(-12*time.Hour))
		require.NoError(t, err, "Failed to create deposit")

		recon2, err := db.StartReconciliation(account.ID, statementDate, NewMoney(2000, CurrencyUSD))
		require.NoError(t, err, "Failed to start reconciliation")

		err = db.ClearTransaction(recon2.ID, deposit2.ID)
		require.NoError(t, err, "Failed to clear transaction")

		completed2, err := db.CompleteReconciliation(recon2.ID)
		require.NoError(t, err, "Failed to complete reconciliation")

		// Try to undo the first reconciliation (not most recent) - should fail
		err = db.UndoReconciliation(completed1.ID)
		assert.Error(t, err, "Expected error when undoing non-most-recent reconciliation")
		assert.Contains(t, err.Error(), "most recent")

		// Undo second (most recent) should work
		err = db.UndoReconciliation(completed2.ID)
		require.NoError(t, err, "Undo most recent should succeed")

		// Now first becomes most recent, undo should work
		err = db.UndoReconciliation(completed1.ID)
		require.NoError(t, err, "Undo first should succeed after second is undone")
	})

	// Test UnclearTransaction from completed reconciliation (should fail)
	t.Run("UnclearFromCompletedReconciliation", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		deposit, err := db.CreateDeposit(account.ID, NewMoney(1500, CurrencyUSD), TransactionMethodElectronicTransfer, "Test", statementDate.Add(-24*time.Hour))
		require.NoError(t, err, "Failed to create deposit")

		reconciliation, err := db.StartReconciliation(account.ID, statementDate, NewMoney(1500, CurrencyUSD))
		require.NoError(t, err, "Failed to start reconciliation")

		err = db.ClearTransaction(reconciliation.ID, deposit.ID)
		require.NoError(t, err, "Failed to clear transaction")

		_, err = db.CompleteReconciliation(reconciliation.ID)
		require.NoError(t, err, "Failed to complete reconciliation")

		// Try to unclear transaction from completed reconciliation
		err = db.UnclearTransaction(deposit.ID)
		assert.Error(t, err, "Expected error when unclearing from completed reconciliation")
	})

	// Test clearing already cleared transaction
	t.Run("ClearAlreadyClearedTransaction", func(t *testing.T) {
		db := testDB(t)
		_, account, _, _, _ := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		deposit, err := db.CreateDeposit(account.ID, NewMoney(500, CurrencyUSD), TransactionMethodElectronicTransfer, "Test", statementDate.Add(-24*time.Hour))
		require.NoError(t, err, "Failed to create deposit")

		reconciliation, err := db.StartReconciliation(account.ID, statementDate, NewMoney(500, CurrencyUSD))
		require.NoError(t, err, "Failed to start reconciliation")

		// Clear once
		err = db.ClearTransaction(reconciliation.ID, deposit.ID)
		require.NoError(t, err, "First clear should succeed")

		// Clear again (same reconciliation) - should be idempotent
		err = db.ClearTransaction(reconciliation.ID, deposit.ID)
		require.NoError(t, err, "Clearing already cleared transaction in same reconciliation should be idempotent")

		// Cancel and start new reconciliation
		_, err = db.CancelReconciliation(reconciliation.ID)
		require.NoError(t, err, "Failed to cancel reconciliation")

		// After cancellation, transaction is uncleared
		reconciliation2, err := db.StartReconciliation(account.ID, statementDate, NewMoney(500, CurrencyUSD))
		require.NoError(t, err, "Failed to start second reconciliation")

		// Clear it for the new reconciliation
		err = db.ClearTransaction(reconciliation2.ID, deposit.ID)
		require.NoError(t, err, "Clear should succeed after previous reconciliation cancelled")

		// Complete reconciliation
		_, err = db.CompleteReconciliation(reconciliation2.ID)
		require.NoError(t, err, "Failed to complete reconciliation")

		// Start new reconciliation and try to clear the already-reconciled transaction
		_, err = db.CreateDeposit(account.ID, NewMoney(300, CurrencyUSD), TransactionMethodElectronicTransfer, "New", statementDate.Add(24*time.Hour))
		require.NoError(t, err, "Failed to create deposit")

		reconciliation3, err := db.StartReconciliation(account.ID, statementDate.AddDate(0, 1, 0), NewMoney(300, CurrencyUSD))
		require.NoError(t, err, "Failed to start third reconciliation")

		// Try to clear the already reconciled transaction
		err = db.ClearTransaction(reconciliation3.ID, deposit.ID)
		assert.Error(t, err, "Expected error when clearing transaction from completed reconciliation")
	})

	// Test with withdrawals
	t.Run("ReconciliationWithWithdrawals", func(t *testing.T) {
		db := testDB(t)
		_, account, _, party, category := setupTestData(t, db)

		statementDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		// Create a deposit
		deposit, err := db.CreateDeposit(account.ID, NewMoney(20000, CurrencyUSD), TransactionMethodElectronicTransfer, "Big deposit", statementDate.Add(-72*time.Hour))
		require.NoError(t, err, "Failed to create deposit")

		expenses := []ExpenseItem{
			{CategoryID: category.ID, Description: strPtr("Expense"), Amount: NewMoney(5000, CurrencyUSD)},
		}
		withdrawal, err := db.CreateWithdrawal(account.ID, party.ID, TransactionMethodCheck, "Payment", statementDate.Add(-48*time.Hour), strPtr("100"), expenses)
		require.NoError(t, err, "Failed to create withdrawal")

		// Expected balance: 20000 - 5000 = 15000
		reconciliation, err := db.StartReconciliation(account.ID, statementDate, NewMoney(15000, CurrencyUSD))
		require.NoError(t, err, "Failed to start reconciliation")

		// Clear both transactions
		err = db.ClearTransaction(reconciliation.ID, deposit.ID)
		require.NoError(t, err, "Failed to clear deposit")

		err = db.ClearTransaction(reconciliation.ID, withdrawal.ID)
		require.NoError(t, err, "Failed to clear withdrawal")

		// Complete
		completed, err := db.CompleteReconciliation(reconciliation.ID)
		require.NoError(t, err, "CompleteReconciliation should succeed")
		assert.Equal(t, ReconciliationStatusCompleted, completed.Status)

		// Verify cleared balance
		clearedBalance, err := db.ClearedBalance(completed.ID)
		require.NoError(t, err, "ClearedBalance should succeed")
		assert.Equal(t, int64(15000), clearedBalance.Amount, "Cleared balance should be $150 (200 - 50)")
	})
}

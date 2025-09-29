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
	assembly, err := db.createAssembly("Test Assembly", time.UTC)
	require.NoError(t, err, "Failed to create test assembly")

	// Create test bank account
	account, err = db.CreateBankAccount("Test Checking", AccountTypeChecking, CurrencyUSD, nil, "Test account", true)
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

		db, err := New(dbPath, "Test Spiritual Assembly", eastern)
		require.NoError(t, err, "New should succeed")
		defer db.Close()

		// Verify assembly was created
		assembly, err := db.Assembly()
		require.NoError(t, err, "Should be able to retrieve assembly")
		assert.Equal(t, "Test Spiritual Assembly", assembly.Name)
		assert.Equal(t, "America/New_York", assembly.Timezone.String())
		assert.NotZero(t, assembly.ID)
	})

	// Test with empty assembly name
	t.Run("EmptyAssemblyName", func(t *testing.T) {
		dbPath := filepath.Join(tmpDir, "empty_name.bedrock")
		_, err := New(dbPath, "", time.UTC)
		assert.Error(t, err, "Expected error for empty assembly name")
	})

	// Test with nil timezone
	t.Run("NilTimezone", func(t *testing.T) {
		dbPath := filepath.Join(tmpDir, "nil_timezone.bedrock")
		_, err := New(dbPath, "Test Assembly", nil)
		assert.Error(t, err, "Expected error for nil timezone")
	})
}

func TestAssemblyCRUD(t *testing.T) {
	pacific, err := time.LoadLocation("America/Los_Angeles")
	require.NoError(t, err, "Failed to load timezone")

	// Test assembly creation through bedrock.New
	t.Run("AssemblyCreationViaNew", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.bedrock")

		db, err := New(dbPath, "Local Spiritual Assembly of Test City", pacific)
		require.NoError(t, err, "bedrock.New should succeed")
		defer db.Close()

		assembly, err := db.Assembly()
		require.NoError(t, err, "Should be able to retrieve assembly")

		assert.NotZero(t, assembly.ID, "Assembly ID should not be zero")
		assert.Equal(t, "Local Spiritual Assembly of Test City", assembly.Name)
		assert.Equal(t, "America/Los_Angeles", assembly.Timezone.String())
		assert.NotZero(t, assembly.CreatedAt)
		assert.NotZero(t, assembly.ModifiedAt)
	})

	// Test duplicate assembly prevention (already tested in TestNew, but testing internal behavior)
	t.Run("DuplicateAssemblyPrevention", func(t *testing.T) {
		freshDb := testDB(t)
		defer freshDb.Close()

		// First assembly should succeed (internal method for testing)
		_, err := freshDb.createAssembly("First Assembly", pacific)
		require.NoError(t, err, "First assembly creation should succeed")

		// Second assembly should fail
		_, err = freshDb.createAssembly("Second Assembly", pacific)
		assert.Error(t, err, "Expected error for duplicate assembly creation")
	})

	// Test Assembly retrieval
	t.Run("AssemblyRetrieval", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "retrieval_test.bedrock")

		eastern, err := time.LoadLocation("America/New_York")
		require.NoError(t, err, "Failed to load timezone")

		// Create database with assembly via New
		db, err := New(dbPath, "Test Assembly for Retrieval", eastern)
		require.NoError(t, err, "Failed to create database")
		defer db.Close()

		// Retrieve assembly
		retrieved, err := db.Assembly()
		require.NoError(t, err, "Assembly retrieval should succeed")

		assert.Equal(t, "Test Assembly for Retrieval", retrieved.Name)
		assert.Equal(t, "America/New_York", retrieved.Timezone.String())
		assert.NotZero(t, retrieved.ID)
	})

	// Test Assembly retrieval with no assembly
	t.Run("NoAssemblyFound", func(t *testing.T) {
		emptyDb := testDB(t)
		defer emptyDb.Close()

		_, err := emptyDb.Assembly()
		assert.Error(t, err, "Expected error when no assembly exists")
	})

	// Test UpdateAssembly
	t.Run("UpdateAssembly", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "update_test.bedrock")

		central, err := time.LoadLocation("America/Chicago")
		require.NoError(t, err, "Failed to load timezone")

		// Create database with assembly via New
		db, err := New(dbPath, "Original Name", time.UTC)
		require.NoError(t, err, "Failed to create database")
		defer db.Close()

		// Small delay to ensure timestamp difference
		time.Sleep(time.Millisecond)

		// Update assembly
		updated, err := db.UpdateAssembly("Updated Name", central)
		require.NoError(t, err, "UpdateAssembly should succeed")

		assert.Equal(t, "Updated Name", updated.Name)
		assert.Equal(t, "America/Chicago", updated.Timezone.String())

		// Verify the update persisted
		retrieved, err := db.Assembly()
		require.NoError(t, err, "Should be able to retrieve updated assembly")
		assert.Equal(t, "Updated Name", retrieved.Name)
		assert.Equal(t, "America/Chicago", retrieved.Timezone.String())
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
		{"USD dollars", 1000, CurrencyUSD, "10.00 USD", 10.00},
		{"USD cents", 199, CurrencyUSD, "1.99 USD", 1.99},
		{"CAD", 2550, CurrencyCAD, "25.50 CAD", 25.50},
		{"Zero", 0, CurrencyUSD, "0.00 USD", 0.00},
		{"Large amount", 1234567, CurrencyUSD, "12345.67 USD", 12345.67},
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
		account, err := db.CreateBankAccount("Main Checking", AccountTypeChecking, CurrencyUSD, nil, "Primary checking account", true)
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
		_, err := db.CreateBankAccount("", AccountTypeChecking, CurrencyUSD, nil, "", true)
		assert.Error(t, err, "Expected error for empty name")
	})

	// Create accounts for hierarchy tests
	parentAccount, err := db.CreateBankAccount("Parent Account", AccountTypeChecking, CurrencyUSD, nil, "Parent account", true)
	require.NoError(t, err, "Failed to create parent account")

	// Test CreateBankAccount with parent
	t.Run("CreateBankAccountWithParent", func(t *testing.T) {
		childAccount, err := db.CreateBankAccount("Child Account", AccountTypeEarmark, CurrencyUSD, &parentAccount.ID, "Child account", true)
		require.NoError(t, err, "CreateBankAccount with parent should succeed")

		require.NotNil(t, childAccount.ParentID)
		assert.Equal(t, parentAccount.ID, *childAccount.ParentID)
	})

	// Test CreateBankAccount with invalid parent
	t.Run("CreateBankAccountInvalidParent", func(t *testing.T) {
		invalidParentID := ID(99999)
		_, err := db.CreateBankAccount("Invalid Parent", AccountTypeEarmark, CurrencyUSD, &invalidParentID, "", true)
		assert.Error(t, err, "Expected error for invalid parent ID")
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
		account, err := db.CreateBankAccount("To Deactivate", AccountTypeChecking, CurrencyUSD, nil, "", true)
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
		cadAccount, err := db.CreateBankAccount("CAD Account", AccountTypeChecking, CurrencyCAD, nil, "CAD test account", true)
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

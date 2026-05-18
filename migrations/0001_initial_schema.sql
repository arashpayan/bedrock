-- Bedrock Database Schema
-- Financial management system for Spiritual Assemblies

CREATE TABLE IF NOT EXISTS assembly (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    timezone TEXT NOT NULL,
    default_currency TEXT NOT NULL DEFAULT 'USD',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS bank_accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_id INTEGER,
    name TEXT NOT NULL,
    account_type TEXT NOT NULL,
    currency TEXT NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (parent_id) REFERENCES bank_accounts(id)
);

CREATE TABLE IF NOT EXISTS items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS parties (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email_address TEXT,
    bahai_id_number TEXT,
    address TEXT,
    telephone_number TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS receipts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    human_id TEXT NOT NULL UNIQUE,
    customer_id INTEGER NOT NULL,
    sold_at DATETIME NOT NULL,
    transaction_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (customer_id) REFERENCES parties(id),
    FOREIGN KEY (transaction_id) REFERENCES transactions(id)
);

CREATE TABLE IF NOT EXISTS receipt_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    receipt_id INTEGER NOT NULL,
    item_id INTEGER NOT NULL,
    description TEXT,
    price INTEGER NOT NULL,
    currency TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (receipt_id) REFERENCES receipts(id),
    FOREIGN KEY (item_id) REFERENCES items(id)
);

CREATE TABLE IF NOT EXISTS reconciliations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,
    statement_date DATETIME NOT NULL,
    statement_balance INTEGER NOT NULL,
    statement_balance_currency TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'in-progress',
    completed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (account_id) REFERENCES bank_accounts(id)
);

CREATE TABLE IF NOT EXISTS transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,
    amount INTEGER NOT NULL,
    check_number TEXT,
    memo TEXT,
    method TEXT,
    payee_id INTEGER,
    reconciliation_id INTEGER,
    transacted_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (account_id) REFERENCES bank_accounts(id),
    FOREIGN KEY (payee_id) REFERENCES parties(id),
    FOREIGN KEY (reconciliation_id) REFERENCES reconciliations(id)
);

CREATE TABLE IF NOT EXISTS expenses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    transaction_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL,
    description TEXT,
    amount INTEGER NOT NULL,
    currency TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (transaction_id) REFERENCES transactions(id),
    FOREIGN KEY (category_id) REFERENCES categories(id)
);

-- Triggers to update modified_at timestamp automatically
CREATE TRIGGER IF NOT EXISTS update_assembly_modified_at
    AFTER UPDATE ON assembly
BEGIN
    UPDATE assembly SET modified_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_bank_accounts_modified_at
    AFTER UPDATE ON bank_accounts
BEGIN
    UPDATE bank_accounts SET modified_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_items_modified_at
    AFTER UPDATE ON items
BEGIN
    UPDATE items SET modified_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_parties_modified_at
    AFTER UPDATE ON parties
BEGIN
    UPDATE parties SET modified_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_receipts_modified_at
    AFTER UPDATE ON receipts
BEGIN
    UPDATE receipts SET modified_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_receipt_items_modified_at
    AFTER UPDATE ON receipt_items
BEGIN
    UPDATE receipt_items SET modified_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_transactions_modified_at
    AFTER UPDATE ON transactions
BEGIN
    UPDATE transactions SET modified_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_categories_modified_at
    AFTER UPDATE ON categories
BEGIN
    UPDATE categories SET modified_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_expenses_modified_at
    AFTER UPDATE ON expenses
BEGIN
    UPDATE expenses SET modified_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_reconciliations_modified_at
    AFTER UPDATE ON reconciliations
BEGIN
    UPDATE reconciliations SET modified_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

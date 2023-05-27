-- used as a key-value store for information about the assembly
CREATE TABLE assembly_info (
    key_name TEXT PRIMARY KEY,
    value TEXT
);

CREATE TABLE accounts (
    id                  INTEGER PRIMARY KEY,
    created_at          INTEGER CHECK (created_at > 0) NOT NULL,
    modified_at         INTEGER CHECK (modified_at > 0) NOT NULL,

    type                TEXT CHECK (type IN ('bank')) NOT NULL,
    name                TEXT NOT NULL,
    description         TEXT NOT NULL,
    denomination        TEXT CHECK (denomination IN ('USD', 'CAD')) NOT NULL,
    starting_balance    INTEGER NOT NULL DEFAULT 0,
    starting_date       INTEGER NOT NULL,
    parent_id           INTEGER,

    CONSTRAINT fk_parent_id FOREIGN KEY(parent_id) REFERENCES accounts(id)
);

CREATE TABLE parties (
    id                  INTEGER PRIMARY KEY,
    created_at          INTEGER CHECK (created_at > 0) NOT NULL,
    modified_at         INTEGER CHECK (modified_at > 0) NOT NULL,

    name                TEXT NOT NULL,
    email_address       TEXT,
    bahai_id_number     TEXT,
    address             TEXT,
    telephone_number    TEXT
);

CREATE TABLE items (
    id                  INTEGER PRIMARY KEY,
    created_at          INTEGER CHECK (created_at > 0) NOT NULL,
    modified_at         INTEGER CHECK (modified_at > 0) NOT NULL,

    name TEXT NOT NULL,
    shortcut TEXT NOT NULL
);

CREATE TABLE receipts (
    id                  INTEGER PRIMARY KEY,
    created_at          INTEGER CHECK (created_at > 0) NOT NULL,
    modified_at         INTEGER CHECK (modified_at > 0) NOT NULL,

    human_id TEXT CHECK (length(human_id) != 0) NOT NULL,
    customer_id INTEGER NOT NULL,
    sold_at INTEGER CHECK (sold_at > 0) NOT NULL,
    total INTEGER CHECK (total > 0) NOT NULL,

    CONSTRAINT fk_customer_id FOREIGN KEY(customer_id) REFERENCES parties(id)
);

CREATE TABLE receipt_items (
    id                  INTEGER PRIMARY KEY,
    created_at          INTEGER CHECK (created_at > 0) NOT NULL,
    modified_at         INTEGER CHECK (modified_at > 0) NOT NULL,

    receipt_id INTEGER NOT NULL,
    item_id INTEGER NOT NULL,
    description TEXT,
    price INTEGER NOT NULL,

    CONSTRAINT fk_receipt_id FOREIGN KEY(receipt_id) REFERENCES receipts(id),
    CONSTRAINT fk_item_id FOREIGN KEY(item_id) REFERENCES items(id)
);

CREATE TABLE deposits (
    id                  INTEGER PRIMARY KEY,
    created_at          INTEGER CHECK (created_at > 0) NOT NULL,
    modified_at         INTEGER CHECK (modified_at > 0) NOT NULL,

    account_id          INTEGER NOT NULL,

    CONSTRAINT fk_account_id FOREIGN KEY(account_id) REFERENCES accounts(id)
);

CREATE TABLE deposit_receipts (
    receipt_id          INTEGER PRIMARY KEY,
    deposit_id          INTEGER NOT NULL,

    CONSTRAINT fk_deposit_id FOREIGN KEY(deposit_id) REFERENCES deposits(id),
    CONSTRAINT fk_receipt_id FOREIGN KEY(receipt_id) REFERENCES receipts(id)
);

CREATE TABLE transactions (
    id                  INTEGER PRIMARY KEY,
    created_at          INTEGER CHECK (created_at > 0) NOT NULL,
    modified_at         INTEGER CHECK (modified_at > 0) NOT NULL,

    account_id          INTEGER NOT NULL,
    amount              INTEGER NOT NULL,
    check_number        TEXT,
    deposit_id          INTEGER,
    memo                TEXT NOT NULL,
    method              TEXT CHECK (method IN ('atm', 'auto-pay', 'electronic-transfer', 'check')),
    payee_id            INTEGER,
    transacted_at       INTEGER CHECK (transacted_at > 0) NOT NULL,

    CONSTRAINT fk_account_id FOREIGN KEY(account_id) REFERENCES accounts(id),
    CONSTRAINT fk_payee_id FOREIGN KEY(payee_id) REFERENCES parties(id),
    -- a deposit should have exactly 1 transaction representing it in the transactions table
    CONSTRAINT unique_deposit_id UNIQUE (deposit_id)
);

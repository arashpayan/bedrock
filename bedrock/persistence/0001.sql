CREATE TABLE accounts (
    id                  INTEGER                             PRIMARY KEY,
    created_at          INTEGER                             NOT NULL,
    modified_at         INTEGER                             NOT NULL,

    type                TEXT CHECK (type IN ('bank'))       NOT NULL,
    name                TEXT                                NOT NULL,
    description         TEXT                                NOT NULL,
    denomination        TEXT CHECK (denomination IN ('USD', 'CAD')) NOT NULL,
    starting_balance    INTEGER                             NOT NULL DEFAULT 0,
    starting_date       INTEGER                             NOT NULL,
    parent_id           INTEGER                             NOT NULL,

    CONSTRAINT fk_parent_id FOREIGN KEY(parent_id) REFERENCES accounts(id)
);

CREATE TABLE deposits (
    id                  INTEGER                             PRIMARY KEY,
    created_at          INTEGER                             NOT NULL,
    modified_at         INTEGER                             NOT NULL,

    deposited_at        INTEGER                             NOT NULL,
    account_id          INTEGER                             NOT NULL,

    CONSTRAINT fk_account_id FOREIGN KEY(account_id) REFERENCES accounts(id)
);

CREATE TABLE deposit_receipts (
    id                  INTEGER                             PRIMARY KEY,

    deposit_id          INTEGER                             NOT NULL,
    receipt_id          INTEGER                             NOT NULL,
)

CREATE TABLE transactions (
    id                  INTEGER                             PRIMARY KEY,
    created_at          INTEGER                             NOT NULL,
    modified_at         INTEGER                             NOT NULL,

    transacted_at       INTEGER                             NOT NULL,
    account_id          INTEGER                             NOT NULL,
    memo                TEXT                                NOT NULL,
    check_number        TEXT                                NOT NULL,
    payee_id            INTEGER,
    deposit_id          INTEGER,

    CONSTRAINT fk_account_id FOREIGN KEY(account_id) REFERENCES accounts(id),
    -- CONSTRAINT fk_payee_ 
);

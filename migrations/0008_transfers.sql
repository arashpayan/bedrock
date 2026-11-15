-- Transfers between a bank account and one of its sub-accounts.
--
-- A transfer reallocates money inside one real bank account: setting funds
-- aside in an earmark, or releasing them back to the parent. No money moves at
-- the bank, so a transfer is recorded as a header row here plus two ordinary
-- transaction legs that reference it: a negative leg on the source account and
-- a positive leg on the destination. Every balance and ledger query already
-- sums transactions, so the legs are counted without special cases, and the
-- parent's combined balance (which includes its sub-accounts) nets to zero.
--
-- The header holds the user-facing record (accounts, amount, memo, date); the
-- legs mirror its amount, memo, and date. bedrock writes all three rows in one
-- database transaction, so they cannot drift apart.
CREATE TABLE IF NOT EXISTS transfers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_account_id INTEGER NOT NULL,
    to_account_id INTEGER NOT NULL,
    amount INTEGER NOT NULL CHECK (amount > 0),
    currency TEXT NOT NULL,
    memo TEXT NOT NULL DEFAULT '',
    transferred_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_account_id) REFERENCES bank_accounts(id),
    FOREIGN KEY (to_account_id) REFERENCES bank_accounts(id),
    CHECK (from_account_id <> to_account_id)
);

-- The transfer a transaction is a leg of; NULL for every ordinary deposit and
-- withdrawal. This, not the (NULL) method, is what marks a transfer leg.
ALTER TABLE transactions ADD COLUMN transfer_id INTEGER REFERENCES transfers(id);

CREATE INDEX IF NOT EXISTS idx_transactions_transfer_id ON transactions(transfer_id);

CREATE TRIGGER IF NOT EXISTS update_transfers_modified_at
    AFTER UPDATE ON transfers
BEGIN
    UPDATE transfers SET modified_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

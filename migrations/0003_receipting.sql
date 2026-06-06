-- Receipting: issuer details on the assembly, per-assembly email delivery
-- settings, and an append-only delivery log for emailed receipts.

-- Issuer details printed on receipt PDFs. All default to the empty string so
-- they map to plain Go string fields with no pointer indirection. The
-- disclaimer seeds the standard tax-deductible-donation statement; treasurers
-- can edit or clear it in Settings.
ALTER TABLE assembly ADD COLUMN mailing_address TEXT NOT NULL DEFAULT '';
ALTER TABLE assembly ADD COLUMN charitable_reg_number TEXT NOT NULL DEFAULT '';
ALTER TABLE assembly ADD COLUMN contact_email TEXT NOT NULL DEFAULT '';
ALTER TABLE assembly ADD COLUMN contact_phone TEXT NOT NULL DEFAULT '';
ALTER TABLE assembly ADD COLUMN receipt_disclaimer TEXT NOT NULL DEFAULT 'No goods or services were provided in exchange for this contribution.';

-- Per-assembly email delivery configuration. A single row, like assembly.
-- Secrets (smtp_password, oauth_client_secret, oauth_refresh_token,
-- oauth_access_token) are stored unencrypted in this file by design; protect
-- the .bedrock file with filesystem permissions and avoid sharing it.
CREATE TABLE IF NOT EXISTS email_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    method TEXT NOT NULL DEFAULT '',                 -- '', 'smtp', or 'gmail_oauth'
    from_name TEXT NOT NULL DEFAULT '',
    from_address TEXT NOT NULL DEFAULT '',
    reply_to TEXT NOT NULL DEFAULT '',
    smtp_host TEXT NOT NULL DEFAULT '',
    smtp_port INTEGER NOT NULL DEFAULT 587,
    smtp_username TEXT NOT NULL DEFAULT '',
    smtp_password TEXT NOT NULL DEFAULT '',
    smtp_security TEXT NOT NULL DEFAULT 'starttls',  -- 'none', 'starttls', or 'tls'
    oauth_client_id TEXT NOT NULL DEFAULT '',
    oauth_client_secret TEXT NOT NULL DEFAULT '',
    oauth_refresh_token TEXT NOT NULL DEFAULT '',
    oauth_access_token TEXT NOT NULL DEFAULT '',
    oauth_token_expiry DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Append-only log of receipt email delivery attempts. A receipt counts as
-- "sent" when it has at least one row here with status='success'.
CREATE TABLE IF NOT EXISTS receipt_deliveries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    receipt_id INTEGER NOT NULL,
    sent_at DATETIME NOT NULL,
    method TEXT NOT NULL,
    recipient_address TEXT NOT NULL,
    status TEXT NOT NULL,                            -- 'success' or 'failure'
    error_message TEXT NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (receipt_id) REFERENCES receipts(id)
);

CREATE INDEX IF NOT EXISTS idx_receipt_deliveries_receipt_id
    ON receipt_deliveries(receipt_id);

CREATE TRIGGER IF NOT EXISTS update_email_settings_modified_at
    AFTER UPDATE ON email_settings
BEGIN
    UPDATE email_settings SET modified_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_receipt_deliveries_modified_at
    AFTER UPDATE ON receipt_deliveries
BEGIN
    UPDATE receipt_deliveries SET modified_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

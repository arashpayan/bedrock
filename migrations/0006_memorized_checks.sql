-- Memorized checks: reusable templates for recurring checks.
--
-- A memorized check stores the parts of a check that stay the same from one
-- instance to the next (account, payee, memo, and the categorized expense
-- lines) so the treasurer can recall it, adjust the date and check number, and
-- save a new withdrawal. The date and check number are deliberately NOT stored;
-- they are entered fresh each time.
CREATE TABLE IF NOT EXISTS memorized_checks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    account_id INTEGER NOT NULL,
    payee_id INTEGER NOT NULL,
    memo TEXT NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (account_id) REFERENCES bank_accounts(id),
    FOREIGN KEY (payee_id) REFERENCES parties(id)
);

-- Template expense lines, mirroring the `expenses` table but attached to a
-- memorized check instead of a transaction.
CREATE TABLE IF NOT EXISTS memorized_check_expenses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    memorized_check_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL,
    description TEXT,
    amount INTEGER NOT NULL,
    currency TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (memorized_check_id) REFERENCES memorized_checks(id),
    FOREIGN KEY (category_id) REFERENCES categories(id)
);

CREATE INDEX IF NOT EXISTS idx_memorized_check_expenses_check_id
    ON memorized_check_expenses(memorized_check_id);

CREATE TRIGGER IF NOT EXISTS update_memorized_checks_modified_at
    AFTER UPDATE ON memorized_checks
BEGIN
    UPDATE memorized_checks SET modified_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_memorized_check_expenses_modified_at
    AFTER UPDATE ON memorized_check_expenses
BEGIN
    UPDATE memorized_check_expenses SET modified_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

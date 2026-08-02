-- Community participation in contributing to the Fund, recorded per Bahá'í month.
--
-- These counts cannot be derived from receipts: a contribution may come from a
-- household, anonymously, or outside the Assembly's records, so the treasurer
-- gathers them by hand and enters them when known. Every month is therefore
-- optional, and a month with no row simply has no data — which is not the same
-- as a month in which nobody contributed.
--
-- Only adults have a denominator (adults_active), so only adults yield a rate;
-- the other categories are raw counts of contributors.
--
-- Rows are keyed by the Bahá'í month itself rather than by date, so the key is
-- stable regardless of timezone. badi_month is 1–19. Ayyám-i-Há is deliberately
-- NOT a valid key: those intercalary days are counted with Mulk (month 18), the
-- month they follow.
CREATE TABLE IF NOT EXISTS participation (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    badi_year INTEGER NOT NULL,
    badi_month INTEGER NOT NULL CHECK (badi_month BETWEEN 1 AND 19),
    adults_contributed INTEGER NOT NULL CHECK (adults_contributed >= 0),
    adults_active INTEGER NOT NULL CHECK (adults_active >= 0),
    youth INTEGER NOT NULL CHECK (youth >= 0),
    junior_youth INTEGER NOT NULL CHECK (junior_youth >= 0),
    children INTEGER NOT NULL CHECK (children >= 0),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK (adults_contributed <= adults_active),
    UNIQUE (badi_year, badi_month)
);

CREATE TRIGGER IF NOT EXISTS update_participation_modified_at
    AFTER UPDATE ON participation
BEGIN
    UPDATE participation SET modified_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

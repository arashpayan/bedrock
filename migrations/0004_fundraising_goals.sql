-- Fundraising-goal tracking.
--
-- Each fund (item) is either counted toward the Assembly's annual fundraising
-- goal or excluded ("earmarked"). Existing funds default to counting; the
-- treasurer marks earmarked funds (National Fund, Huqúqu'lláh, Humanitarian
-- Fund, etc.) as not counting.
ALTER TABLE items ADD COLUMN counts_toward_goal BOOLEAN NOT NULL DEFAULT 1;

-- One editable fundraising goal per fiscal year. The fiscal year runs May 1
-- through April 30 and is identified by its starting calendar year: fiscal_year
-- = 2026 means May 1, 2026 .. April 30, 2027. amount is stored in the smallest
-- currency unit (cents), matching the rest of the schema.
CREATE TABLE IF NOT EXISTS fundraising_goals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fiscal_year INTEGER NOT NULL UNIQUE,
    amount INTEGER NOT NULL,
    currency TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER IF NOT EXISTS update_fundraising_goals_modified_at
    AFTER UPDATE ON fundraising_goals
BEGIN
    UPDATE fundraising_goals SET modified_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

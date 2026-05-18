-- Add an optional treasurer's memo to receipts. Existing rows default to the
-- empty string so the column maps cleanly to a Go string field with no
-- pointer indirection.
ALTER TABLE receipts ADD COLUMN memo TEXT NOT NULL DEFAULT '';

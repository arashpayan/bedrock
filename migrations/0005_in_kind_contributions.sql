-- In-kind (non-cash) contributions.
--
-- An in-kind contribution records value a contributor donated directly (e.g.
-- paying a vendor for an event's food) without the money passing through the
-- Assembly. It is a receipt that counts toward fundraising but never produces a
-- bank deposit, so it is flagged here and kept out of every cash workflow
-- (deposits, the undeposited list, reconciliation).
ALTER TABLE receipts ADD COLUMN is_in_kind BOOLEAN NOT NULL DEFAULT 0;

-- Optional expense category for a receipt line. Only meaningful for in-kind
-- contributions, where it records what the donated value was spent on (e.g.
-- "Food"), so the gift can later be surfaced in expense reporting. NULL for
-- ordinary cash contributions.
ALTER TABLE receipt_items ADD COLUMN category_id INTEGER REFERENCES categories(id);

-- Gift-in-kind receipts need different wording than cash receipts (fair market
-- value of donated property, no cash received). Editable per Assembly, seeded
-- with a sensible default.
ALTER TABLE assembly ADD COLUMN in_kind_receipt_disclaimer TEXT NOT NULL
    DEFAULT 'This is an acknowledgement of a gift in kind. The amount shown is the fair market value of the donated property; no cash was received by the Assembly. No goods or services were provided in exchange for this gift.';

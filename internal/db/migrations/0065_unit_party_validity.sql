-- HAUSV-794: date bounds per unit party (email -> valid_from / valid_to).
-- Empty bounds are unbounded. Existing JSON records and parties stay intact.
-- The column shares the unit's tenant scope; no new tenant table is introduced.
ALTER TABLE units ADD COLUMN party_validity TEXT NOT NULL DEFAULT '{}';

-- HAUSV-794: additive dates on the existing FORCE ROW LEVEL SECURITY units table.
-- dbmove transfers this text JSON column unchanged, including unbounded parties.
ALTER TABLE units ADD COLUMN party_validity text NOT NULL DEFAULT '{}';

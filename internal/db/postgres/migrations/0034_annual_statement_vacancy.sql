-- HAUSV-785: inclusive vacancy range per unit and period. Empty strings mean
-- the unit is not flagged vacant. Existing rows stay occupied.
ALTER TABLE annual_statement_period_unit_bases ADD COLUMN vacant_from text NOT NULL DEFAULT '';
ALTER TABLE annual_statement_period_unit_bases ADD COLUMN vacant_to text NOT NULL DEFAULT '';

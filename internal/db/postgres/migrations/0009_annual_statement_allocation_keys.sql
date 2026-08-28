-- HAUSV-577: one Verteilerschlüssel per allocatable cost type. Existing
-- allocatable rows fall back to Nutzwert, the § 32 WEG default that reuses the
-- Miteigentumsanteil already recorded on each unit.
ALTER TABLE annual_statement_cost_types ADD COLUMN IF NOT EXISTS allocation_key text NOT NULL DEFAULT '';
UPDATE annual_statement_cost_types SET allocation_key = 'nutzwert' WHERE allocatable AND allocation_key = '';

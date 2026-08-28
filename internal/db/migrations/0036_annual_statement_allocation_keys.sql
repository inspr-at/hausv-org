-- HAUSV-577: one Verteilerschlüssel per allocatable cost type. Existing
-- allocatable rows fall back to Nutzwert, the § 32 WEG default that reuses the
-- Miteigentumsanteil already recorded on each unit. Non-allocatable rows keep
-- an empty key.
ALTER TABLE annual_statement_cost_types ADD COLUMN allocation_key TEXT NOT NULL DEFAULT '';
UPDATE annual_statement_cost_types SET allocation_key = 'nutzwert' WHERE allocatable = 1 AND allocation_key = '';

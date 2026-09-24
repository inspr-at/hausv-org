-- HAUSV-787: JSON text keeps SQLite/dbmove parity; existing clause RLS and
-- immutable tenant_id guards protect this column as well.
ALTER TABLE index_clauses ADD COLUMN staffel_steps text NOT NULL DEFAULT '[]'
    CHECK (jsonb_typeof(staffel_steps::jsonb) = 'array');

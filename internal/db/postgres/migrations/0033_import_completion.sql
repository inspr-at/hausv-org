-- Preserve the existing unique dedupe key and all historical evidence. Its
-- PRIMARY KEY already prohibits duplicates; no deduplication rewrite is needed.
-- Existing ENABLE/FORCE ROW LEVEL SECURITY and tenant policies cover these
-- additive columns as well. Defaults preserve the tenant-only RLS smoke seed.
ALTER TABLE integration_imports ADD COLUMN status text NOT NULL DEFAULT 'complete' CHECK (status IN ('pending', 'complete'));
ALTER TABLE integration_imports ADD COLUMN blob_key text NOT NULL DEFAULT '';
ALTER TABLE integration_imports ADD COLUMN document_data text NOT NULL DEFAULT '';

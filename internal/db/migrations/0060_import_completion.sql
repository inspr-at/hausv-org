-- The existing PRIMARY KEY (tenant_slug, format, file_digest) is already the
-- import dedupe key: duplicates of that key cannot exist. Keep it unchanged;
-- adding a tenant_id index would needlessly merge historical slug identities.
-- Old records describe completed imports and retain all original evidence.
ALTER TABLE integration_imports ADD COLUMN status TEXT NOT NULL DEFAULT 'complete' CHECK (status IN ('pending', 'complete'));
ALTER TABLE integration_imports ADD COLUMN blob_key TEXT NOT NULL DEFAULT '';
ALTER TABLE integration_imports ADD COLUMN document_data TEXT NOT NULL DEFAULT '';

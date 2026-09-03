ALTER TABLE textbausteine ADD COLUMN active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1));
ALTER TABLE textbausteine ADD COLUMN updated_at TEXT NOT NULL DEFAULT '';

UPDATE textbausteine
SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE updated_at = '';

CREATE INDEX idx_textbausteine_org_active ON textbausteine(org_key, active);

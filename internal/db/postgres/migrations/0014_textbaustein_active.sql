ALTER TABLE textbausteine ADD COLUMN active boolean NOT NULL DEFAULT true;
ALTER TABLE textbausteine ADD COLUMN updated_at text NOT NULL DEFAULT '';

UPDATE textbausteine
SET updated_at = to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
WHERE updated_at = '';

CREATE INDEX idx_textbausteine_org_active ON textbausteine(org_key, active);

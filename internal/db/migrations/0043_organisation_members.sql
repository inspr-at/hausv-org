-- Employees of a Hausverwaltung. The house roles they receive stay in the
-- membership tables, where authorization reads them; this table records who
-- belongs to the organisation and — in granted — exactly which house roles the
-- membership handed out, so leaving can put them back the way they were.
CREATE TABLE organisation_members (
    org_key TEXT NOT NULL,
    email TEXT NOT NULL,
    role TEXT NOT NULL,
    -- JSON object: tenant slug -> the role that house had before this
    -- membership granted one ("" when the person had no membership at all).
    granted TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (org_key, email)
);
CREATE INDEX idx_organisation_members_email ON organisation_members(email);

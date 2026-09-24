-- HAUSV-782: provenance and complete undo state for bundled house grants.
ALTER TABLE house_memberships ADD COLUMN grant_origin TEXT NOT NULL DEFAULT '';
ALTER TABLE organisation_members ADD COLUMN undo TEXT NOT NULL DEFAULT '{}';

-- The existing JSONL audit is a presentation mirror. This record commits in
-- the same transaction as the privileges it describes and survives removal.
CREATE TABLE organisation_membership_audit (
    org_key TEXT NOT NULL,
    id TEXT NOT NULL,
    events TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (org_key, id)
);

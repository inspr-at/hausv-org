-- HAUSV-782: additive counterpart of SQLite 0059. Existing table RLS also
-- protects the new columns. Old role-only undo data is retained unchanged.
ALTER TABLE house_memberships ADD COLUMN grant_origin text NOT NULL DEFAULT '';
ALTER TABLE organisation_members ADD COLUMN undo text NOT NULL DEFAULT '{}';
CREATE TABLE organisation_membership_audit (
    org_key text NOT NULL,
    id text NOT NULL,
    events text NOT NULL,
    created_at text NOT NULL,
    PRIMARY KEY (org_key, id)
);
ALTER TABLE organisation_membership_audit ENABLE ROW LEVEL SECURITY;
ALTER TABLE organisation_membership_audit FORCE ROW LEVEL SECURITY;
CREATE POLICY organisation_isolation ON organisation_membership_audit
    USING (coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR org_key = coalesce(current_setting('app.org_key', true), ''))
    WITH CHECK (coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR org_key = coalesce(current_setting('app.org_key', true), ''));

-- Mirror of migrations/0043_organisation_members.sql for PostgreSQL, with the
-- same organisation isolation the other org-keyed tables carry.
CREATE TABLE organisation_members (
    org_key text NOT NULL,
    email text NOT NULL,
    role text NOT NULL,
    granted text NOT NULL DEFAULT '{}',
    created_at text NOT NULL,
    updated_at text NOT NULL,
    PRIMARY KEY (org_key, email)
);
CREATE INDEX idx_organisation_members_email ON organisation_members(email);

DO $$
BEGIN
    EXECUTE 'ALTER TABLE organisation_members ENABLE ROW LEVEL SECURITY';
    EXECUTE 'ALTER TABLE organisation_members FORCE ROW LEVEL SECURITY';
    EXECUTE
        'CREATE POLICY organisation_isolation ON organisation_members USING (' ||
        'coalesce(current_setting(''hausv.cross_tenant'', true), '''') = ''on'' ' ||
        'OR org_key = coalesce(current_setting(''app.org_key'', true), '''')) ' ||
        'WITH CHECK (coalesce(current_setting(''hausv.cross_tenant'', true), '''') = ''on'' ' ||
        'OR org_key = coalesce(current_setting(''app.org_key'', true), ''''))';
END;
$$;

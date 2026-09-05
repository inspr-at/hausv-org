-- Mirror of migrations/0042_organisation.sql for PostgreSQL, with the same
-- organisation isolation the other org-keyed tables carry.
CREATE TABLE organisations (
    org_key text PRIMARY KEY,
    name text NOT NULL,
    contact_name text NOT NULL DEFAULT '',
    contact_email text NOT NULL DEFAULT '',
    contact_phone text NOT NULL DEFAULT '',
    updated_at text NOT NULL
);

CREATE TABLE organisation_houses (
    org_key text NOT NULL,
    tenant_slug text NOT NULL,
    added_at text NOT NULL,
    PRIMARY KEY (org_key, tenant_slug)
);
CREATE INDEX idx_organisation_houses_tenant ON organisation_houses(tenant_slug);

DO $$
DECLARE table_name text;
BEGIN
    FOREACH table_name IN ARRAY ARRAY['organisations', 'organisation_houses'] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', table_name);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', table_name);
        EXECUTE format(
            'CREATE POLICY organisation_isolation ON %I USING (' ||
            'coalesce(current_setting(''hausv.cross_tenant'', true), '''') = ''on'' ' ||
            'OR org_key = coalesce(current_setting(''app.org_key'', true), '''')) ' ||
            'WITH CHECK (coalesce(current_setting(''hausv.cross_tenant'', true), '''') = ''on'' ' ||
            'OR org_key = coalesce(current_setting(''app.org_key'', true), ''''))',
            table_name
        );
    END LOOP;
END;
$$;

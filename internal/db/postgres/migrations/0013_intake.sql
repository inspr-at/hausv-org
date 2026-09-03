CREATE TABLE intake_items (
    org_key text NOT NULL,
    id text NOT NULL,
    status text NOT NULL,
    source text NOT NULL,
    tenant_slug text NOT NULL DEFAULT '',
    received_at text NOT NULL,
    due_at text NOT NULL DEFAULT '',
    data text NOT NULL,
    PRIMARY KEY (org_key, id)
);
CREATE INDEX idx_intake_items_org_status_received ON intake_items(org_key, status, received_at);
CREATE INDEX idx_intake_items_org_tenant ON intake_items(org_key, tenant_slug);

CREATE TABLE org_settings (
    org_key text PRIMARY KEY,
    data text NOT NULL
);

CREATE TABLE textbausteine (
    org_key text NOT NULL,
    key text NOT NULL,
    data text NOT NULL,
    PRIMARY KEY (org_key, key)
);

DO $$
DECLARE table_name text;
BEGIN
    FOREACH table_name IN ARRAY ARRAY['intake_items', 'org_settings', 'textbausteine'] LOOP
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

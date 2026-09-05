-- Mirror of migrations/0044_intake_mail_seen.sql for PostgreSQL, with the same
-- organisation isolation the other org-keyed tables carry.
CREATE TABLE intake_mail_seen (
    org_key text NOT NULL,
    message_id text NOT NULL,
    intake_id text NOT NULL DEFAULT '',
    seen_at text NOT NULL,
    PRIMARY KEY (org_key, message_id)
);

DO $$
BEGIN
    EXECUTE 'ALTER TABLE intake_mail_seen ENABLE ROW LEVEL SECURITY';
    EXECUTE 'ALTER TABLE intake_mail_seen FORCE ROW LEVEL SECURITY';
    EXECUTE
        'CREATE POLICY organisation_isolation ON intake_mail_seen USING (' ||
        'coalesce(current_setting(''hausv.cross_tenant'', true), '''') = ''on'' ' ||
        'OR org_key = coalesce(current_setting(''app.org_key'', true), '''')) ' ||
        'WITH CHECK (coalesce(current_setting(''hausv.cross_tenant'', true), '''') = ''on'' ' ||
        'OR org_key = coalesce(current_setting(''app.org_key'', true), ''''))';
END;
$$;

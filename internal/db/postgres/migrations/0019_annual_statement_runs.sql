CREATE TABLE annual_statement_runs (
    tenant_id varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    tenant_slug text NOT NULL DEFAULT '',
    id text NOT NULL DEFAULT '',
    period_year bigint NOT NULL DEFAULT 0,
    revision bigint NOT NULL DEFAULT 0,
    data text NOT NULL DEFAULT '{}',
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, period_year, revision)
);
ALTER TABLE annual_statement_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE annual_statement_runs FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON annual_statement_runs
    USING (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    )
    WITH CHECK (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    );
CREATE TRIGGER tenant_id_immutable
    BEFORE UPDATE OF tenant_id ON annual_statement_runs
    FOR EACH ROW EXECUTE FUNCTION hausv_reject_tenant_id_change();

-- Stored calculations and their snapshots must remain immutable, even when a
-- future caller attempts a direct write rather than going through the store.
CREATE FUNCTION hausv_reject_annual_statement_run_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'annual_statement_runs are immutable' USING ERRCODE = '23514';
END;
$$;
CREATE TRIGGER annual_statement_run_immutable
    BEFORE UPDATE OR DELETE ON annual_statement_runs
    FOR EACH ROW EXECUTE FUNCTION hausv_reject_annual_statement_run_change();

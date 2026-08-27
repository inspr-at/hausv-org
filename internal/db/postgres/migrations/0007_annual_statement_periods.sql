-- Reusable basic periods for Austrian annual operating-cost statements.
CREATE TABLE IF NOT EXISTS annual_statement_periods (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    year bigint NOT NULL DEFAULT 0,
    starts_on text NOT NULL DEFAULT '',
    ends_on text NOT NULL DEFAULT '',
    updated_at text NOT NULL DEFAULT '',
    updated_by text NOT NULL DEFAULT '',
    PRIMARY KEY (tenant_slug, year)
);
CREATE INDEX IF NOT EXISTS idx_annual_statement_periods_tenant_id
    ON annual_statement_periods(tenant_id);

ALTER TABLE annual_statement_periods ENABLE ROW LEVEL SECURITY;
ALTER TABLE annual_statement_periods FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON annual_statement_periods
    USING (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    )
    WITH CHECK (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    );
CREATE TRIGGER tenant_id_immutable
    BEFORE UPDATE OF tenant_id ON annual_statement_periods
    FOR EACH ROW EXECUTE FUNCTION hausv_reject_tenant_id_change();

CREATE TABLE IF NOT EXISTS annual_statement_prepayments (
    tenant_slug  text NOT NULL DEFAULT '',
    tenant_id    varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    period_year  bigint NOT NULL DEFAULT 0,
    unit_id      text NOT NULL DEFAULT '',
    amount_cents bigint NOT NULL DEFAULT 0 CHECK (amount_cents >= 0),
    updated_at   text NOT NULL DEFAULT '',
    updated_by   text NOT NULL DEFAULT '',
    PRIMARY KEY (tenant_slug, period_year, unit_id)
);
CREATE INDEX IF NOT EXISTS idx_annual_statement_prepayments_tenant_id
    ON annual_statement_prepayments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_annual_statement_prepayments_period
    ON annual_statement_prepayments(tenant_slug, period_year);

ALTER TABLE annual_statement_prepayments ENABLE ROW LEVEL SECURITY;
ALTER TABLE annual_statement_prepayments FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON annual_statement_prepayments
    USING (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    )
    WITH CHECK (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    );
CREATE TRIGGER tenant_id_immutable
    BEFORE UPDATE OF tenant_id ON annual_statement_prepayments
    FOR EACH ROW EXECUTE FUNCTION hausv_reject_tenant_id_change();

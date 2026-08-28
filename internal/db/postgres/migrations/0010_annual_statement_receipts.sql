CREATE TABLE IF NOT EXISTS annual_statement_receipts (
    tenant_slug   text NOT NULL DEFAULT '',
    tenant_id     varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    id            text NOT NULL DEFAULT '',
    document_id   text NOT NULL DEFAULT '',
    period_year   bigint NOT NULL DEFAULT 0,
    cost_type_key text NOT NULL DEFAULT '',
    amount_cents  bigint NOT NULL DEFAULT 0,
    invoice_date  text NOT NULL DEFAULT '',
    created_at    text NOT NULL DEFAULT '',
    created_by    text NOT NULL DEFAULT '',
    updated_at    text NOT NULL DEFAULT '',
    updated_by    text NOT NULL DEFAULT '',
    PRIMARY KEY (tenant_slug, id),
    UNIQUE (tenant_slug, document_id)
);
CREATE INDEX IF NOT EXISTS idx_annual_statement_receipts_tenant_id
    ON annual_statement_receipts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_annual_statement_receipts_period
    ON annual_statement_receipts(tenant_slug, period_year);

ALTER TABLE annual_statement_receipts ENABLE ROW LEVEL SECURITY;
ALTER TABLE annual_statement_receipts FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON annual_statement_receipts
    USING (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    )
    WITH CHECK (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    );
CREATE TRIGGER tenant_id_immutable
    BEFORE UPDATE OF tenant_id ON annual_statement_receipts
    FOR EACH ROW EXECUTE FUNCTION hausv_reject_tenant_id_change();

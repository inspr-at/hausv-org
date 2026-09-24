CREATE TABLE IF NOT EXISTS annual_statement_reserve_entries (
    tenant_slug   text NOT NULL DEFAULT '',
    tenant_id     varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    id            text NOT NULL DEFAULT '',
    period_year   bigint NOT NULL DEFAULT 0,
    kind          text NOT NULL DEFAULT 'opening',
    entry_date    text NOT NULL DEFAULT '',
    amount_cents  bigint NOT NULL DEFAULT 0,
    document_id   text NOT NULL DEFAULT '',
    note          text NOT NULL DEFAULT '',
    created_at    text NOT NULL DEFAULT '',
    created_by    text NOT NULL DEFAULT '',
    PRIMARY KEY (tenant_slug, id),
    CONSTRAINT annual_statement_reserve_entries_kind CHECK (kind IN ('opening', 'contribution', 'withdrawal', 'interest', 'closing_check'))
);
CREATE INDEX IF NOT EXISTS idx_annual_statement_reserve_entries_tenant_id
    ON annual_statement_reserve_entries(tenant_id);
CREATE INDEX IF NOT EXISTS idx_annual_statement_reserve_entries_period
    ON annual_statement_reserve_entries(tenant_slug, period_year);

ALTER TABLE annual_statement_reserve_entries ENABLE ROW LEVEL SECURITY;
ALTER TABLE annual_statement_reserve_entries FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON annual_statement_reserve_entries
    USING (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    )
    WITH CHECK (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    );
CREATE TRIGGER tenant_id_immutable
    BEFORE UPDATE OF tenant_id ON annual_statement_reserve_entries
    FOR EACH ROW EXECUTE FUNCTION hausv_reject_tenant_id_change();

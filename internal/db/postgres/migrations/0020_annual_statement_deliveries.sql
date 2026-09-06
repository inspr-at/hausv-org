CREATE TABLE annual_statement_deliveries (
    tenant_id varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    tenant_slug TEXT NOT NULL,
    id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    revision bigint NOT NULL CHECK (revision > 0),
    party_id TEXT NOT NULL,
    unit_id TEXT NOT NULL,
    document_id TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    recipient TEXT NOT NULL,
    sent_at TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('sent', 'failed')),
    error TEXT NOT NULL,
    actor TEXT NOT NULL,
    attempt bigint NOT NULL CHECK (attempt > 0),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, run_id, revision, unit_id, party_id, attempt)
);
CREATE UNIQUE INDEX annual_statement_delivery_sent ON annual_statement_deliveries
    (tenant_id, run_id, revision, unit_id, party_id) WHERE status = 'sent';

ALTER TABLE annual_statement_deliveries ENABLE ROW LEVEL SECURITY;
ALTER TABLE annual_statement_deliveries FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON annual_statement_deliveries
    USING (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    )
    WITH CHECK (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    );
CREATE TRIGGER tenant_id_immutable
    BEFORE UPDATE OF tenant_id ON annual_statement_deliveries
    FOR EACH ROW EXECUTE FUNCTION hausv_reject_tenant_id_change();


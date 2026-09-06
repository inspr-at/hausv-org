CREATE TABLE annual_statement_deliveries (
    tenant_id TEXT NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    tenant_slug TEXT NOT NULL,
    id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    revision INTEGER NOT NULL CHECK (revision > 0),
    party_id TEXT NOT NULL,
    unit_id TEXT NOT NULL,
    document_id TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    recipient TEXT NOT NULL,
    sent_at TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('sent', 'failed')),
    error TEXT NOT NULL,
    actor TEXT NOT NULL,
    attempt INTEGER NOT NULL CHECK (attempt > 0),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, run_id, revision, unit_id, party_id, attempt)
);
CREATE UNIQUE INDEX annual_statement_delivery_sent ON annual_statement_deliveries
    (tenant_id, run_id, revision, unit_id, party_id) WHERE status = 'sent';

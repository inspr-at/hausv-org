-- HAUSV-642: a delivery is reserved (status pending) in a short transaction that
-- commits before the mail exchange, so a slow mail server never holds SQLite's
-- process-wide write lock. SQLite cannot widen a CHECK in place; the table is
-- rebuilt with the same columns, keys and partial unique index.
CREATE TABLE annual_statement_deliveries_next (
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
    status TEXT NOT NULL CHECK (status IN ('pending', 'sent', 'failed')),
    error TEXT NOT NULL,
    actor TEXT NOT NULL,
    attempt INTEGER NOT NULL CHECK (attempt > 0),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, run_id, revision, unit_id, party_id, attempt)
);
INSERT INTO annual_statement_deliveries_next
    (tenant_id,tenant_slug,id,run_id,revision,party_id,unit_id,document_id,sha256,recipient,sent_at,status,error,actor,attempt)
SELECT tenant_id,tenant_slug,id,run_id,revision,party_id,unit_id,document_id,sha256,recipient,sent_at,status,error,actor,attempt
FROM annual_statement_deliveries;
DROP TABLE annual_statement_deliveries;
ALTER TABLE annual_statement_deliveries_next RENAME TO annual_statement_deliveries;
CREATE UNIQUE INDEX annual_statement_delivery_sent ON annual_statement_deliveries
    (tenant_id, run_id, revision, unit_id, party_id) WHERE status = 'sent';

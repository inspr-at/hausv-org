CREATE TABLE IF NOT EXISTS annual_statement_receipts (
    tenant_slug   TEXT NOT NULL,
    tenant_id     TEXT NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    id            TEXT NOT NULL,
    document_id   TEXT NOT NULL,
    period_year   INTEGER NOT NULL,
    cost_type_key TEXT NOT NULL,
    amount_cents  INTEGER NOT NULL,
    invoice_date  TEXT NOT NULL,
    created_at    TEXT NOT NULL,
    created_by    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    updated_by    TEXT NOT NULL,
    PRIMARY KEY (tenant_slug, id),
    UNIQUE (tenant_slug, document_id)
);
CREATE INDEX IF NOT EXISTS idx_annual_statement_receipts_tenant_id
    ON annual_statement_receipts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_annual_statement_receipts_period
    ON annual_statement_receipts(tenant_slug, period_year);

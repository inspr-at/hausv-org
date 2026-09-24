CREATE TABLE IF NOT EXISTS annual_statement_reserve_entries (
    tenant_slug   TEXT NOT NULL,
    tenant_id     TEXT NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    id            TEXT NOT NULL,
    period_year   INTEGER NOT NULL,
    kind          TEXT NOT NULL,
    entry_date    TEXT NOT NULL,
    amount_cents  INTEGER NOT NULL,
    document_id   TEXT NOT NULL DEFAULT '',
    note          TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL,
    created_by    TEXT NOT NULL,
    PRIMARY KEY (tenant_slug, id)
);
CREATE INDEX IF NOT EXISTS idx_annual_statement_reserve_entries_tenant_id
    ON annual_statement_reserve_entries(tenant_id);
CREATE INDEX IF NOT EXISTS idx_annual_statement_reserve_entries_period
    ON annual_statement_reserve_entries(tenant_slug, period_year);

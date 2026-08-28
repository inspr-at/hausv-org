CREATE TABLE IF NOT EXISTS annual_statement_prepayments (
    tenant_slug  TEXT NOT NULL,
    tenant_id    TEXT NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    period_year  INTEGER NOT NULL,
    unit_id      TEXT NOT NULL,
    amount_cents INTEGER NOT NULL CHECK (amount_cents >= 0),
    updated_at   TEXT NOT NULL,
    updated_by   TEXT NOT NULL,
    PRIMARY KEY (tenant_slug, period_year, unit_id)
);
CREATE INDEX IF NOT EXISTS idx_annual_statement_prepayments_tenant_id
    ON annual_statement_prepayments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_annual_statement_prepayments_period
    ON annual_statement_prepayments(tenant_slug, period_year);

-- Reusable basic periods for Austrian annual operating-cost statements.
-- Parties remain on the existing units record; later statement slices may
-- refer to this period but must not smuggle calculations into these basics.
CREATE TABLE IF NOT EXISTS annual_statement_periods (
    tenant_slug TEXT NOT NULL,
    tenant_id   TEXT NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    year        INTEGER NOT NULL,
    starts_on   TEXT NOT NULL,
    ends_on     TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    updated_by  TEXT NOT NULL,
    PRIMARY KEY (tenant_slug, year)
);
CREATE INDEX IF NOT EXISTS idx_annual_statement_periods_tenant_id
    ON annual_statement_periods(tenant_id);

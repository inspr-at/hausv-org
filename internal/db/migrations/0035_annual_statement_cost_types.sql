-- Tenant-owned cost-type catalogue for annual operating-cost statements.
-- Allocation keys, receipts and calculations belong to later slices.
CREATE TABLE IF NOT EXISTS annual_statement_cost_types (
    tenant_slug TEXT NOT NULL,
    tenant_id   TEXT NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    key         TEXT NOT NULL,
    name        TEXT NOT NULL,
    allocatable INTEGER NOT NULL,
    updated_at  TEXT NOT NULL,
    updated_by  TEXT NOT NULL,
    PRIMARY KEY (tenant_slug, key)
);
CREATE INDEX IF NOT EXISTS idx_annual_statement_cost_types_tenant_id
    ON annual_statement_cost_types(tenant_id);

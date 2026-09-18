-- Parking and charging state as one JSON document per tenant (HAUSV-758).
CREATE TABLE IF NOT EXISTS parking (
    tenant_slug TEXT NOT NULL DEFAULT '',
    tenant_id TEXT NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    data TEXT NOT NULL DEFAULT '{}',
    PRIMARY KEY (tenant_slug)
);
CREATE INDEX IF NOT EXISTS idx_parking_tenant_id ON parking(tenant_id);

-- Wohneinheiten per tenant (HAUSV-168/170). Replaces /data/units.json. Full
-- record as JSON; key (tenant, id). The primary key enforces the same
-- uniqueness NormalizeUnits deduplicates in memory.
CREATE TABLE IF NOT EXISTS units (
    tenant_slug TEXT NOT NULL,
    id          TEXT NOT NULL,
    data        TEXT NOT NULL,
    PRIMARY KEY (tenant_slug, id)
);

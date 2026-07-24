-- Wohnungsübergaben (move-in/out protocols) per tenant (HAUSV-168/170).
-- Replaces /data/handovers.json. Full record as JSON; key (tenant, id).
-- Confirmation tokens are matched by scanning all rows (low volume).
CREATE TABLE IF NOT EXISTS handovers (
    tenant_slug TEXT NOT NULL,
    id          TEXT NOT NULL,
    data        TEXT NOT NULL,
    PRIMARY KEY (tenant_slug, id)
);

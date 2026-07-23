-- Termine / Kalender entries per tenant (HAUSV-168/170). Replaces
-- /data/events.json. Full record as JSON; key (tenant, id).
CREATE TABLE IF NOT EXISTS events (
    tenant_slug TEXT NOT NULL,
    id          TEXT NOT NULL,
    data        TEXT NOT NULL,
    PRIMARY KEY (tenant_slug, id)
);

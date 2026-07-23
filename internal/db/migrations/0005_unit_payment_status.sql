-- Manual per-unit payment-status marker (HAUSV-168/170). Replaces
-- /data/unit_payment_status.json. One row per tenant+unit. This is only a
-- transparency marker (offen/bezahlt/teilbezahlt/ueberfaellig), not accounting.
CREATE TABLE IF NOT EXISTS unit_payment_status (
    tenant_slug TEXT NOT NULL,
    unit_id     TEXT NOT NULL,
    status      TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    updated_by  TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (tenant_slug, unit_id)
);

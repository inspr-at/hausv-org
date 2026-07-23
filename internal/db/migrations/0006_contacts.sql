-- Managed contacts (Dienstleister, Hausmeister, Notdienste) per tenant
-- (HAUSV-168/170). Replaces /data/contacts.json. The full record is a JSON
-- document; tenant_slug and active are columns because the store filters by
-- them (ListTenant with/without inactive). Key is (tenant, id).
CREATE TABLE IF NOT EXISTS contacts (
    tenant_slug TEXT NOT NULL,
    id          TEXT NOT NULL,
    active      INTEGER NOT NULL DEFAULT 1,
    data        TEXT NOT NULL,
    PRIMARY KEY (tenant_slug, id)
);

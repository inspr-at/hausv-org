-- The Hausverwaltung becomes a stored entity instead of a name assembled from
-- environment configuration. Houses stay the unit of authorization; this is the
-- administrative level above them, and the table the employee roster will hang
-- off next.
CREATE TABLE organisations (
    org_key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    contact_name TEXT NOT NULL DEFAULT '',
    contact_email TEXT NOT NULL DEFAULT '',
    contact_phone TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL
);

CREATE TABLE organisation_houses (
    org_key TEXT NOT NULL,
    tenant_slug TEXT NOT NULL,
    added_at TEXT NOT NULL,
    PRIMARY KEY (org_key, tenant_slug)
);
CREATE INDEX idx_organisation_houses_tenant ON organisation_houses(tenant_slug);

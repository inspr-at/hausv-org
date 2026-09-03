CREATE TABLE intake_items (
    org_key TEXT NOT NULL,
    id TEXT NOT NULL,
    status TEXT NOT NULL,
    source TEXT NOT NULL,
    tenant_slug TEXT NOT NULL DEFAULT '',
    received_at TEXT NOT NULL,
    due_at TEXT NOT NULL DEFAULT '',
    data TEXT NOT NULL,
    PRIMARY KEY (org_key, id)
);
CREATE INDEX idx_intake_items_org_status_received ON intake_items(org_key, status, received_at);
CREATE INDEX idx_intake_items_org_tenant ON intake_items(org_key, tenant_slug);

CREATE TABLE org_settings (
    org_key TEXT PRIMARY KEY,
    data TEXT NOT NULL
);

CREATE TABLE textbausteine (
    org_key TEXT NOT NULL,
    key TEXT NOT NULL,
    data TEXT NOT NULL,
    PRIMARY KEY (org_key, key)
);

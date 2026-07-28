-- HAUSV Zuhause: private house profile, vendor-neutral energy inventory and
-- read-only measurement mappings. Secrets and Home Assistant endpoints do not
-- belong in this database; they stay in the host configuration.
CREATE TABLE IF NOT EXISTS home_profiles (
    tenant_slug         TEXT PRIMARY KEY,
    home_type           TEXT NOT NULL DEFAULT 'apartment',
    household_name      TEXT NOT NULL DEFAULT '',
    operating_mode      TEXT NOT NULL DEFAULT 'observe'
                        CHECK (operating_mode IN ('observe', 'active')),
    automation_stage    TEXT NOT NULL DEFAULT 'observe'
                        CHECK (automation_stage IN ('observe', 'recommend', 'shadow', 'active')),
    onboarding_step     INTEGER NOT NULL DEFAULT 1 CHECK (onboarding_step >= 1),
    onboarding_complete INTEGER NOT NULL DEFAULT 0 CHECK (onboarding_complete IN (0, 1)),
    target_peak_kw      REAL,
    recommendation_id  TEXT NOT NULL DEFAULT '',
    recommendation_status TEXT NOT NULL DEFAULT '',
    free_started_at     TEXT,
    created_at          TEXT NOT NULL,
    updated_at          TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS energy_assets (
    id              TEXT PRIMARY KEY,
    tenant_slug     TEXT NOT NULL,
    kind            TEXT NOT NULL,
    name            TEXT NOT NULL,
    rated_power_kw  REAL,
    flexibility     TEXT NOT NULL DEFAULT 'unknown',
    source          TEXT NOT NULL DEFAULT 'manual',
    confirmed       INTEGER NOT NULL DEFAULT 0 CHECK (confirmed IN (0, 1)),
    metadata_json   TEXT NOT NULL DEFAULT '{}',
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    FOREIGN KEY (tenant_slug) REFERENCES home_profiles(tenant_slug) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS energy_assets_tenant_kind
    ON energy_assets(tenant_slug, kind, name);

CREATE TABLE IF NOT EXISTS energy_entity_mappings (
    id            TEXT PRIMARY KEY,
    tenant_slug   TEXT NOT NULL,
    entity_id     TEXT NOT NULL,
    metric        TEXT NOT NULL,
    display_name  TEXT NOT NULL DEFAULT '',
    unit          TEXT NOT NULL DEFAULT '',
    device_class  TEXT NOT NULL DEFAULT '',
    confirmed     INTEGER NOT NULL DEFAULT 0 CHECK (confirmed IN (0, 1)),
    last_seen_at  TEXT,
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    UNIQUE (tenant_slug, entity_id),
    FOREIGN KEY (tenant_slug) REFERENCES home_profiles(tenant_slug) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS energy_entity_mappings_tenant_metric
    ON energy_entity_mappings(tenant_slug, metric, confirmed);

CREATE TABLE IF NOT EXISTS energy_intervals (
    tenant_slug      TEXT NOT NULL,
    starts_at        TEXT NOT NULL,
    duration_minutes INTEGER NOT NULL DEFAULT 15 CHECK (duration_minutes > 0),
    import_kwh       REAL NOT NULL,
    average_kw       REAL NOT NULL,
    quality          TEXT NOT NULL DEFAULT 'measured',
    source           TEXT NOT NULL DEFAULT 'home-assistant',
    created_at       TEXT NOT NULL,
    PRIMARY KEY (tenant_slug, starts_at, source),
    FOREIGN KEY (tenant_slug) REFERENCES home_profiles(tenant_slug) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS energy_intervals_tenant_time
    ON energy_intervals(tenant_slug, starts_at DESC);

CREATE TABLE IF NOT EXISTS energy_imports (
    id          TEXT PRIMARY KEY,
    tenant_slug TEXT NOT NULL,
    filename    TEXT NOT NULL,
    sha256      TEXT NOT NULL,
    format      TEXT NOT NULL,
    payload     BLOB NOT NULL,
    imported_at TEXT NOT NULL,
    UNIQUE (tenant_slug, sha256),
    FOREIGN KEY (tenant_slug) REFERENCES home_profiles(tenant_slug) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS energy_imports_tenant_time
    ON energy_imports(tenant_slug, imported_at DESC);

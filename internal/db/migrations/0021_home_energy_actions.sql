-- Durable follow-up actions for HAUSV Zuhause. These tables deliberately keep
-- house maintenance, tariff snapshots and curated measures tenant-scoped.
-- External service-provider access remains governed by the existing launch
-- gate; storing a known contact here never grants portal access.
-- The first energy schema used deterministic asset IDs without the tenant.
-- Prefix managed seed/onboarding assets before more homes are added.
UPDATE energy_assets
SET id = 'asset-' || tenant_slug || '-' || kind
WHERE source IN ('onboarding', 'profile-seed')
  AND id = 'asset-' || kind;

CREATE TABLE IF NOT EXISTS energy_maintenance_plans (
    id                TEXT PRIMARY KEY,
    tenant_slug       TEXT NOT NULL,
    asset_id          TEXT NOT NULL,
    title             TEXT NOT NULL,
    interval_months   INTEGER NOT NULL CHECK (interval_months BETWEEN 1 AND 120),
    last_completed_at TEXT,
    next_due_at       TEXT NOT NULL,
    contact_id        TEXT NOT NULL DEFAULT '',
    document_id       TEXT NOT NULL DEFAULT '',
    issue_id          TEXT NOT NULL DEFAULT '',
    evidence_note     TEXT NOT NULL DEFAULT '',
    active            INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL,
    UNIQUE (tenant_slug, asset_id),
    FOREIGN KEY (tenant_slug) REFERENCES home_profiles(tenant_slug) ON DELETE CASCADE,
    FOREIGN KEY (asset_id) REFERENCES energy_assets(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS energy_maintenance_tenant_due
    ON energy_maintenance_plans(tenant_slug, active, next_due_at);

CREATE TABLE IF NOT EXISTS energy_tariff_assessments (
    id               TEXT PRIMARY KEY,
    tenant_slug      TEXT NOT NULL,
    assessment_month TEXT NOT NULL,
    profile_id       TEXT NOT NULL,
    profile_version  TEXT NOT NULL,
    profile_status   TEXT NOT NULL,
    source_url       TEXT NOT NULL,
    peak_kw          REAL NOT NULL,
    billed_kw        REAL NOT NULL,
    annual_power_eur REAL NOT NULL,
    data_quality     TEXT NOT NULL,
    created_at       TEXT NOT NULL,
    FOREIGN KEY (tenant_slug) REFERENCES home_profiles(tenant_slug) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS energy_tariff_assessments_tenant_time
    ON energy_tariff_assessments(tenant_slug, created_at DESC);

CREATE TABLE IF NOT EXISTS energy_measures (
    id                 TEXT PRIMARY KEY,
    tenant_slug        TEXT NOT NULL,
    issue_id           TEXT NOT NULL,
    recommendation_id  TEXT NOT NULL,
    title              TEXT NOT NULL,
    status             TEXT NOT NULL DEFAULT 'draft'
                       CHECK (status IN ('draft','requested','assigned','scheduled','completed','cancelled')),
    contact_id         TEXT NOT NULL DEFAULT '',
    shared_fields_json TEXT NOT NULL DEFAULT '[]',
    offer_note         TEXT NOT NULL DEFAULT '',
    appointment_at     TEXT,
    work_note          TEXT NOT NULL DEFAULT '',
    completed_at       TEXT,
    evidence_note      TEXT NOT NULL DEFAULT '',
    before_from        TEXT,
    before_to          TEXT,
    after_from         TEXT,
    after_to           TEXT,
    before_peak_kw     REAL,
    after_peak_kw      REAL,
    before_quality     TEXT NOT NULL DEFAULT '',
    after_quality      TEXT NOT NULL DEFAULT '',
    created_at         TEXT NOT NULL,
    updated_at         TEXT NOT NULL,
    UNIQUE (tenant_slug, issue_id),
    FOREIGN KEY (tenant_slug) REFERENCES home_profiles(tenant_slug) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS energy_measures_tenant_time
    ON energy_measures(tenant_slug, updated_at DESC);

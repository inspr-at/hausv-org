-- Isolate HAUSV Zuhause data below a tenant by a stable home_key. Existing
-- installations remain the implicit "default" home; no operator config or
-- data rewrite is required for the single-home case.

CREATE TABLE home_profiles_scoped (
    tenant_slug         TEXT NOT NULL,
    home_key            TEXT NOT NULL DEFAULT 'default',
    unit_id             TEXT NOT NULL DEFAULT '',
    home_type           TEXT NOT NULL DEFAULT 'apartment',
    household_name      TEXT NOT NULL DEFAULT '',
    operating_mode      TEXT NOT NULL DEFAULT 'observe' CHECK (operating_mode IN ('observe', 'active')),
    automation_stage    TEXT NOT NULL DEFAULT 'observe' CHECK (automation_stage IN ('observe', 'recommend', 'shadow', 'active')),
    onboarding_step     INTEGER NOT NULL DEFAULT 1 CHECK (onboarding_step >= 1),
    onboarding_complete INTEGER NOT NULL DEFAULT 0 CHECK (onboarding_complete IN (0, 1)),
    target_peak_kw      REAL,
    agreed_power_kw     REAL,
    recommendation_id  TEXT NOT NULL DEFAULT '',
    recommendation_status TEXT NOT NULL DEFAULT '',
    free_started_at     TEXT,
    created_at          TEXT NOT NULL,
    updated_at          TEXT NOT NULL,
    PRIMARY KEY (tenant_slug, home_key)
);
INSERT INTO home_profiles_scoped
    (tenant_slug,home_key,unit_id,home_type,household_name,operating_mode,automation_stage,onboarding_step,onboarding_complete,target_peak_kw,agreed_power_kw,recommendation_id,recommendation_status,free_started_at,created_at,updated_at)
SELECT tenant_slug,'default',unit_id,home_type,household_name,operating_mode,automation_stage,onboarding_step,onboarding_complete,target_peak_kw,agreed_power_kw,recommendation_id,recommendation_status,free_started_at,created_at,updated_at
FROM home_profiles;

CREATE TABLE energy_assets_scoped (
    id TEXT PRIMARY KEY, tenant_slug TEXT NOT NULL, home_key TEXT NOT NULL DEFAULT 'default',
    kind TEXT NOT NULL, name TEXT NOT NULL, rated_power_kw REAL,
    flexibility TEXT NOT NULL DEFAULT 'unknown', source TEXT NOT NULL DEFAULT 'manual',
    confirmed INTEGER NOT NULL DEFAULT 0 CHECK (confirmed IN (0,1)),
    metadata_json TEXT NOT NULL DEFAULT '{}', created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
    FOREIGN KEY (tenant_slug,home_key) REFERENCES home_profiles_scoped(tenant_slug,home_key) ON DELETE CASCADE
);
INSERT INTO energy_assets_scoped SELECT id,tenant_slug,'default',kind,name,rated_power_kw,flexibility,source,confirmed,metadata_json,created_at,updated_at FROM energy_assets;

CREATE TABLE energy_entity_mappings_scoped (
    id TEXT PRIMARY KEY, tenant_slug TEXT NOT NULL, home_key TEXT NOT NULL DEFAULT 'default',
    entity_id TEXT NOT NULL, asset_id TEXT NOT NULL DEFAULT '', metric TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '', unit TEXT NOT NULL DEFAULT '', device_class TEXT NOT NULL DEFAULT '',
    confirmed INTEGER NOT NULL DEFAULT 0 CHECK (confirmed IN (0,1)), last_seen_at TEXT,
    created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
    UNIQUE (tenant_slug,home_key,entity_id),
    FOREIGN KEY (tenant_slug,home_key) REFERENCES home_profiles_scoped(tenant_slug,home_key) ON DELETE CASCADE
);
INSERT INTO energy_entity_mappings_scoped SELECT id,tenant_slug,'default',entity_id,asset_id,metric,display_name,unit,device_class,confirmed,last_seen_at,created_at,updated_at FROM energy_entity_mappings;

CREATE TABLE energy_intervals_scoped (
    tenant_slug TEXT NOT NULL, home_key TEXT NOT NULL DEFAULT 'default', starts_at TEXT NOT NULL,
    duration_minutes INTEGER NOT NULL DEFAULT 15 CHECK (duration_minutes > 0), import_kwh REAL NOT NULL,
    average_kw REAL NOT NULL, quality TEXT NOT NULL DEFAULT 'measured', source TEXT NOT NULL DEFAULT 'home-assistant',
    created_at TEXT NOT NULL, PRIMARY KEY (tenant_slug,home_key,starts_at,source),
    FOREIGN KEY (tenant_slug,home_key) REFERENCES home_profiles_scoped(tenant_slug,home_key) ON DELETE CASCADE
);
INSERT INTO energy_intervals_scoped SELECT tenant_slug,'default',starts_at,duration_minutes,import_kwh,average_kw,quality,source,created_at FROM energy_intervals;

CREATE TABLE energy_imports_scoped (
    id TEXT PRIMARY KEY, tenant_slug TEXT NOT NULL, home_key TEXT NOT NULL DEFAULT 'default', filename TEXT NOT NULL,
    sha256 TEXT NOT NULL, format TEXT NOT NULL, payload BLOB NOT NULL, imported_at TEXT NOT NULL,
    UNIQUE (tenant_slug,home_key,sha256),
    FOREIGN KEY (tenant_slug,home_key) REFERENCES home_profiles_scoped(tenant_slug,home_key) ON DELETE CASCADE
);
INSERT INTO energy_imports_scoped SELECT id,tenant_slug,'default',filename,sha256,format,payload,imported_at FROM energy_imports;

CREATE TABLE energy_maintenance_plans_scoped (
    id TEXT PRIMARY KEY, tenant_slug TEXT NOT NULL, home_key TEXT NOT NULL DEFAULT 'default', asset_id TEXT NOT NULL,
    title TEXT NOT NULL, interval_months INTEGER NOT NULL CHECK (interval_months BETWEEN 1 AND 120),
    last_completed_at TEXT, next_due_at TEXT NOT NULL, contact_id TEXT NOT NULL DEFAULT '',
    document_id TEXT NOT NULL DEFAULT '', issue_id TEXT NOT NULL DEFAULT '', evidence_note TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)), created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
    UNIQUE (tenant_slug,home_key,asset_id),
    FOREIGN KEY (tenant_slug,home_key) REFERENCES home_profiles_scoped(tenant_slug,home_key) ON DELETE CASCADE,
    FOREIGN KEY (asset_id) REFERENCES energy_assets_scoped(id) ON DELETE CASCADE
);
INSERT INTO energy_maintenance_plans_scoped SELECT id,tenant_slug,'default',asset_id,title,interval_months,last_completed_at,next_due_at,contact_id,document_id,issue_id,evidence_note,active,created_at,updated_at FROM energy_maintenance_plans;

CREATE TABLE energy_tariff_assessments_scoped (
    id TEXT PRIMARY KEY, tenant_slug TEXT NOT NULL, home_key TEXT NOT NULL DEFAULT 'default', assessment_month TEXT NOT NULL,
    profile_id TEXT NOT NULL, profile_version TEXT NOT NULL, profile_status TEXT NOT NULL, source_url TEXT NOT NULL,
    peak_kw REAL NOT NULL, billed_kw REAL NOT NULL, annual_power_eur REAL NOT NULL, data_quality TEXT NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY (tenant_slug,home_key) REFERENCES home_profiles_scoped(tenant_slug,home_key) ON DELETE CASCADE
);
INSERT INTO energy_tariff_assessments_scoped SELECT id,tenant_slug,'default',assessment_month,profile_id,profile_version,profile_status,source_url,peak_kw,billed_kw,annual_power_eur,data_quality,created_at FROM energy_tariff_assessments;

CREATE TABLE energy_measures_scoped (
    id TEXT PRIMARY KEY, tenant_slug TEXT NOT NULL, home_key TEXT NOT NULL DEFAULT 'default', issue_id TEXT NOT NULL,
    recommendation_id TEXT NOT NULL, title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','requested','assigned','scheduled','completed','cancelled')),
    contact_id TEXT NOT NULL DEFAULT '', shared_fields_json TEXT NOT NULL DEFAULT '[]', offer_note TEXT NOT NULL DEFAULT '',
    appointment_at TEXT, work_note TEXT NOT NULL DEFAULT '', completed_at TEXT, evidence_note TEXT NOT NULL DEFAULT '',
    before_from TEXT, before_to TEXT, after_from TEXT, after_to TEXT, before_peak_kw REAL, after_peak_kw REAL,
    before_quality TEXT NOT NULL DEFAULT '', after_quality TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
    UNIQUE (tenant_slug,home_key,issue_id),
    FOREIGN KEY (tenant_slug,home_key) REFERENCES home_profiles_scoped(tenant_slug,home_key) ON DELETE CASCADE
);
INSERT INTO energy_measures_scoped SELECT id,tenant_slug,'default',issue_id,recommendation_id,title,status,contact_id,shared_fields_json,offer_note,appointment_at,work_note,completed_at,evidence_note,before_from,before_to,after_from,after_to,before_peak_kw,after_peak_kw,before_quality,after_quality,created_at,updated_at FROM energy_measures;

DROP TABLE energy_maintenance_plans;
DROP TABLE energy_entity_mappings;
DROP TABLE energy_intervals;
DROP TABLE energy_imports;
DROP TABLE energy_tariff_assessments;
DROP TABLE energy_measures;
DROP TABLE energy_assets;
DROP TABLE home_profiles;

ALTER TABLE home_profiles_scoped RENAME TO home_profiles;
ALTER TABLE energy_assets_scoped RENAME TO energy_assets;
ALTER TABLE energy_entity_mappings_scoped RENAME TO energy_entity_mappings;
ALTER TABLE energy_intervals_scoped RENAME TO energy_intervals;
ALTER TABLE energy_imports_scoped RENAME TO energy_imports;
ALTER TABLE energy_maintenance_plans_scoped RENAME TO energy_maintenance_plans;
ALTER TABLE energy_tariff_assessments_scoped RENAME TO energy_tariff_assessments;
ALTER TABLE energy_measures_scoped RENAME TO energy_measures;

CREATE INDEX energy_assets_tenant_home_kind ON energy_assets(tenant_slug,home_key,kind,name);
CREATE INDEX energy_entity_mappings_tenant_home_metric ON energy_entity_mappings(tenant_slug,home_key,metric,confirmed);
CREATE INDEX energy_entity_mappings_tenant_home_asset ON energy_entity_mappings(tenant_slug,home_key,asset_id,confirmed);
CREATE INDEX energy_intervals_tenant_home_time ON energy_intervals(tenant_slug,home_key,starts_at DESC);
CREATE INDEX energy_imports_tenant_home_time ON energy_imports(tenant_slug,home_key,imported_at DESC);
CREATE INDEX energy_maintenance_tenant_home_due ON energy_maintenance_plans(tenant_slug,home_key,active,next_due_at);
CREATE INDEX energy_tariff_assessments_tenant_home_time ON energy_tariff_assessments(tenant_slug,home_key,created_at DESC);
CREATE INDEX energy_measures_tenant_home_time ON energy_measures(tenant_slug,home_key,updated_at DESC);

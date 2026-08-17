-- PostgreSQL 1.x target schema. Phase 0 creates this schema beside the live
-- SQLite schema; no production store uses it yet.

CREATE TABLE IF NOT EXISTS tenant (
    tenant_id varchar(26) PRIMARY KEY
        CHECK (tenant_id ~ '^[0-7][0-9A-HJKMNP-TV-Z]{25}$'),
    slug text NOT NULL UNIQUE,
    name text NOT NULL DEFAULT '',
    created_at text NOT NULL DEFAULT '',
    updated_at text NOT NULL DEFAULT ''
);

CREATE OR REPLACE FUNCTION hausv_reject_tenant_id_change()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.tenant_id IS DISTINCT FROM OLD.tenant_id THEN
        RAISE EXCEPTION 'tenant_id is immutable';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS tenant_tenant_id_immutable ON tenant;
CREATE TRIGGER tenant_tenant_id_immutable
BEFORE UPDATE OF tenant_id ON tenant
FOR EACH ROW EXECUTE FUNCTION hausv_reject_tenant_id_change();

CREATE TABLE IF NOT EXISTS app_meta (
    key text PRIMARY KEY,
    value text NOT NULL
);
CREATE TABLE IF NOT EXISTS login_activity (
    email text PRIMARY KEY,
    last_login text NOT NULL DEFAULT '',
    auth_method text NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS profile_overlays (
    email text PRIMARY KEY,
    title text NOT NULL DEFAULT '', first_name text NOT NULL DEFAULT '',
    last_name text NOT NULL DEFAULT '', phone text NOT NULL DEFAULT '',
    directory_opt_in boolean NOT NULL DEFAULT false,
    updated_at text NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS notification_prefs (
    email text PRIMARY KEY,
    prefs text NOT NULL DEFAULT '{}'
);
CREATE TABLE IF NOT EXISTS persons (
    id text PRIMARY KEY,
    email text NOT NULL UNIQUE,
    title text NOT NULL DEFAULT '', first_name text NOT NULL DEFAULT '',
    last_name text NOT NULL DEFAULT '', auth_methods text NOT NULL DEFAULT '',
    deactivated boolean NOT NULL DEFAULT false,
    adopted boolean NOT NULL DEFAULT false,
    created_at text NOT NULL DEFAULT '',
    updated_at text NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS telegram_state (
    key text PRIMARY KEY, value text NOT NULL
);
CREATE TABLE IF NOT EXISTS telegram_links (
    chat_id bigint PRIMARY KEY, email text NOT NULL,
    name text NOT NULL DEFAULT '', linked_at text NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS telegram_link_codes (
    code text PRIMARY KEY, email text NOT NULL, created_by text NOT NULL DEFAULT '',
    expires_at text NOT NULL
);

-- The nine document-row tables deliberately retain their data text column in
-- Phase 0. Their relational decomposition belongs to Phase 2.
CREATE TABLE IF NOT EXISTS unit_payment_status (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    unit_id text NOT NULL DEFAULT '', status text NOT NULL DEFAULT '',
    updated_at text NOT NULL DEFAULT '', updated_by text NOT NULL DEFAULT '',
    PRIMARY KEY (tenant_slug, unit_id)
);
CREATE TABLE IF NOT EXISTS contacts (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    id text NOT NULL DEFAULT '', active boolean NOT NULL DEFAULT true,
    data text NOT NULL DEFAULT '{}', PRIMARY KEY (tenant_slug, id)
);
CREATE TABLE IF NOT EXISTS announcement_reads (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    email text NOT NULL DEFAULT '', seen_at text NOT NULL DEFAULT '',
    PRIMARY KEY (tenant_slug, email)
);
CREATE TABLE IF NOT EXISTS announcements (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    id text NOT NULL DEFAULT '', data text NOT NULL DEFAULT '{}', PRIMARY KEY (tenant_slug, id)
);
CREATE TABLE IF NOT EXISTS events (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    id text NOT NULL DEFAULT '', data text NOT NULL DEFAULT '{}', PRIMARY KEY (tenant_slug, id)
);
CREATE TABLE IF NOT EXISTS handovers (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    id text NOT NULL DEFAULT '', data text NOT NULL DEFAULT '{}', PRIMARY KEY (tenant_slug, id)
);
CREATE TABLE IF NOT EXISTS documents (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    id text NOT NULL DEFAULT '', data text NOT NULL DEFAULT '{}', PRIMARY KEY (tenant_slug, id)
);
CREATE TABLE IF NOT EXISTS attachments (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    id text NOT NULL DEFAULT '', data text NOT NULL DEFAULT '{}', PRIMARY KEY (tenant_slug, id)
);
CREATE TABLE IF NOT EXISTS units (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    id text NOT NULL DEFAULT '', data text NOT NULL DEFAULT '{}', PRIMARY KEY (tenant_slug, id)
);
CREATE TABLE IF NOT EXISTS ballots (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    id text NOT NULL DEFAULT '', data text NOT NULL DEFAULT '{}', PRIMARY KEY (tenant_slug, id)
);
CREATE TABLE IF NOT EXISTS issues (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    id text NOT NULL DEFAULT '', data text NOT NULL DEFAULT '{}', PRIMARY KEY (tenant_slug, id)
);

CREATE TABLE IF NOT EXISTS house_memberships (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    person_id text NOT NULL DEFAULT '', role text NOT NULL DEFAULT '', permissions text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT '', directory_opt_in boolean,
    created_at text NOT NULL DEFAULT '', updated_at text NOT NULL DEFAULT '',
    PRIMARY KEY (tenant_slug, person_id),
    FOREIGN KEY (person_id) REFERENCES persons(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_house_memberships_tenant ON house_memberships(tenant_slug);
CREATE TABLE IF NOT EXISTS integration_imports (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    format text NOT NULL DEFAULT '', file_digest text NOT NULL DEFAULT '', source_version text NOT NULL DEFAULT '',
    applied_at text NOT NULL DEFAULT '', applied_by text NOT NULL DEFAULT '',
    assigned integer NOT NULL DEFAULT 0, changed integer NOT NULL DEFAULT 0,
    unclear integer NOT NULL DEFAULT 0, rejected integer NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_slug, format, file_digest)
);

CREATE TABLE IF NOT EXISTS home_profiles (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    home_key text NOT NULL DEFAULT 'default', unit_id text NOT NULL DEFAULT '',
    home_type text NOT NULL DEFAULT 'apartment', household_name text NOT NULL DEFAULT '',
    operating_mode text NOT NULL DEFAULT 'observe' CHECK (operating_mode IN ('observe','active')),
    automation_stage text NOT NULL DEFAULT 'observe' CHECK (automation_stage IN ('observe','recommend','shadow','active')),
    onboarding_step integer NOT NULL DEFAULT 1 CHECK (onboarding_step >= 1),
    onboarding_complete boolean NOT NULL DEFAULT false,
    target_peak_kw double precision, agreed_power_kw double precision,
    recommendation_id text NOT NULL DEFAULT '', recommendation_status text NOT NULL DEFAULT '',
    free_started_at text, free_until_at text,
    created_at text NOT NULL DEFAULT '', updated_at text NOT NULL DEFAULT '',
    PRIMARY KEY (tenant_slug, home_key)
);
CREATE TABLE IF NOT EXISTS energy_assets (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    home_key text NOT NULL DEFAULT 'default', id text NOT NULL DEFAULT '', kind text NOT NULL DEFAULT '',
    name text NOT NULL DEFAULT '', rated_power_kw double precision, flexibility text NOT NULL DEFAULT 'unknown',
    source text NOT NULL DEFAULT 'manual', confirmed boolean NOT NULL DEFAULT false,
    metadata_json text NOT NULL DEFAULT '{}', created_at text NOT NULL DEFAULT '',
    updated_at text NOT NULL DEFAULT '', PRIMARY KEY (tenant_slug, id),
    FOREIGN KEY (tenant_slug, home_key) REFERENCES home_profiles(tenant_slug, home_key) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS energy_assets_tenant_home_kind ON energy_assets(tenant_slug,home_key,kind,name);
CREATE TABLE IF NOT EXISTS energy_entity_mappings (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    home_key text NOT NULL DEFAULT 'default', id text NOT NULL DEFAULT '', entity_id text NOT NULL DEFAULT '',
    asset_id text NOT NULL DEFAULT '', metric text NOT NULL DEFAULT '', display_name text NOT NULL DEFAULT '',
    unit text NOT NULL DEFAULT '', device_class text NOT NULL DEFAULT '', confirmed boolean NOT NULL DEFAULT false,
    last_seen_at text, created_at text NOT NULL DEFAULT '',
    updated_at text NOT NULL DEFAULT '', PRIMARY KEY (tenant_slug, id),
    UNIQUE (tenant_slug,home_key,entity_id),
    FOREIGN KEY (tenant_slug,home_key) REFERENCES home_profiles(tenant_slug,home_key) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS energy_entity_mappings_tenant_home_metric ON energy_entity_mappings(tenant_slug,home_key,metric,confirmed);
CREATE INDEX IF NOT EXISTS energy_entity_mappings_tenant_home_asset ON energy_entity_mappings(tenant_slug,home_key,asset_id,confirmed);
CREATE TABLE IF NOT EXISTS energy_intervals (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    home_key text NOT NULL DEFAULT 'default', starts_at text NOT NULL DEFAULT '',
    duration_minutes integer NOT NULL DEFAULT 15 CHECK (duration_minutes > 0),
    import_kwh double precision NOT NULL DEFAULT 0, average_kw double precision NOT NULL DEFAULT 0,
    quality text NOT NULL DEFAULT 'measured', source text NOT NULL DEFAULT 'home-assistant',
    created_at text NOT NULL DEFAULT '', PRIMARY KEY (tenant_slug,home_key,starts_at,source),
    FOREIGN KEY (tenant_slug,home_key) REFERENCES home_profiles(tenant_slug,home_key) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS energy_intervals_tenant_home_time ON energy_intervals(tenant_slug,home_key,starts_at DESC);
CREATE TABLE IF NOT EXISTS energy_imports (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    home_key text NOT NULL DEFAULT 'default', id text NOT NULL DEFAULT '', filename text NOT NULL DEFAULT '',
    sha256 text NOT NULL DEFAULT '', format text NOT NULL DEFAULT '', payload bytea NOT NULL DEFAULT ''::bytea,
    imported_at text NOT NULL DEFAULT '', PRIMARY KEY (tenant_slug,id), UNIQUE (tenant_slug,home_key,sha256),
    FOREIGN KEY (tenant_slug,home_key) REFERENCES home_profiles(tenant_slug,home_key) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS energy_imports_tenant_home_time ON energy_imports(tenant_slug,home_key,imported_at DESC);
CREATE TABLE IF NOT EXISTS energy_maintenance_plans (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    home_key text NOT NULL DEFAULT 'default', id text NOT NULL DEFAULT '', asset_id text NOT NULL DEFAULT '',
    title text NOT NULL DEFAULT '', interval_months integer NOT NULL DEFAULT 12 CHECK (interval_months BETWEEN 1 AND 120),
    last_completed_at text, next_due_at text NOT NULL DEFAULT '', contact_id text NOT NULL DEFAULT '',
    document_id text NOT NULL DEFAULT '', issue_id text NOT NULL DEFAULT '', evidence_note text NOT NULL DEFAULT '',
    active boolean NOT NULL DEFAULT true, created_at text NOT NULL DEFAULT '',
    updated_at text NOT NULL DEFAULT '', PRIMARY KEY (tenant_slug,id), UNIQUE (tenant_slug,home_key,asset_id),
    FOREIGN KEY (tenant_slug,home_key) REFERENCES home_profiles(tenant_slug,home_key) ON DELETE CASCADE,
    FOREIGN KEY (tenant_slug,asset_id) REFERENCES energy_assets(tenant_slug,id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS energy_maintenance_tenant_home_due ON energy_maintenance_plans(tenant_slug,home_key,active,next_due_at);
CREATE TABLE IF NOT EXISTS energy_tariff_assessments (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    home_key text NOT NULL DEFAULT 'default', id text NOT NULL DEFAULT '', assessment_month text NOT NULL DEFAULT '',
    profile_id text NOT NULL DEFAULT '', profile_version text NOT NULL DEFAULT '', profile_status text NOT NULL DEFAULT '',
    source_url text NOT NULL DEFAULT '', peak_kw double precision NOT NULL DEFAULT 0,
    billed_kw double precision NOT NULL DEFAULT 0, annual_power_eur double precision NOT NULL DEFAULT 0,
    data_quality text NOT NULL DEFAULT '', created_at text NOT NULL DEFAULT '', PRIMARY KEY (tenant_slug,id),
    FOREIGN KEY (tenant_slug,home_key) REFERENCES home_profiles(tenant_slug,home_key) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS energy_tariff_assessments_tenant_home_time ON energy_tariff_assessments(tenant_slug,home_key,created_at DESC);
CREATE TABLE IF NOT EXISTS energy_measures (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    home_key text NOT NULL DEFAULT 'default', id text NOT NULL DEFAULT '', issue_id text NOT NULL DEFAULT '',
    recommendation_id text NOT NULL DEFAULT '', title text NOT NULL DEFAULT '', status text NOT NULL DEFAULT 'draft',
    contact_id text NOT NULL DEFAULT '', shared_fields_json text NOT NULL DEFAULT '[]', offer_note text NOT NULL DEFAULT '',
    appointment_at text, work_note text NOT NULL DEFAULT '', completed_at text,
    evidence_note text NOT NULL DEFAULT '', before_from text, before_to text,
    after_from text, after_to text, before_peak_kw double precision, after_peak_kw double precision,
    before_quality text NOT NULL DEFAULT '', after_quality text NOT NULL DEFAULT '',
    created_at text NOT NULL DEFAULT '', updated_at text NOT NULL DEFAULT '',
    PRIMARY KEY (tenant_slug,id), UNIQUE (tenant_slug,home_key,issue_id),
    FOREIGN KEY (tenant_slug,home_key) REFERENCES home_profiles(tenant_slug,home_key) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS energy_measures_tenant_home_time ON energy_measures(tenant_slug,home_key,updated_at DESC);

CREATE TABLE IF NOT EXISTS home_reservations (
    slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    household_name text NOT NULL DEFAULT '', owner_email text NOT NULL DEFAULT '',
    authorization_confirmed boolean NOT NULL DEFAULT false, status text NOT NULL DEFAULT '',
    created_at text NOT NULL DEFAULT '', updated_at text NOT NULL DEFAULT '',
    confirmed_at text, PRIMARY KEY (slug)
);
CREATE INDEX IF NOT EXISTS idx_home_reservations_owner ON home_reservations(owner_email);
CREATE TABLE IF NOT EXISTS home_connectors (
    slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    status text NOT NULL DEFAULT '', credential_hash bytea, generation integer NOT NULL DEFAULT 0,
    pairing_hash bytea, pairing_expires_at text, connector_version text NOT NULL DEFAULT '',
    ha_version text NOT NULL DEFAULT '', entity_count integer NOT NULL DEFAULT 0,
    created_at text NOT NULL DEFAULT '', updated_at text NOT NULL DEFAULT '',
    paired_at text, last_seen_at text, PRIMARY KEY (slug),
    FOREIGN KEY (slug) REFERENCES home_reservations(slug) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_home_connectors_credential ON home_connectors(credential_hash) WHERE credential_hash IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_home_connectors_pairing ON home_connectors(pairing_hash) WHERE pairing_hash IS NOT NULL;
CREATE TABLE IF NOT EXISTS home_portals (
    slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    household_name text NOT NULL DEFAULT '', owner_email text NOT NULL DEFAULT '',
    activated_at text NOT NULL DEFAULT '', updated_at text NOT NULL DEFAULT '',
    PRIMARY KEY (slug),
    FOREIGN KEY (slug) REFERENCES home_reservations(slug) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_home_portals_owner_email ON home_portals(owner_email);
CREATE TABLE IF NOT EXISTS home_connector_readings (
    slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    entity_id text NOT NULL DEFAULT '', state text NOT NULL DEFAULT '', display_name text NOT NULL DEFAULT '',
    unit text NOT NULL DEFAULT '', device_class text NOT NULL DEFAULT '', state_class text NOT NULL DEFAULT '',
    last_updated text NOT NULL DEFAULT '', received_at text NOT NULL DEFAULT '',
    PRIMARY KEY (slug,entity_id),
    FOREIGN KEY (slug) REFERENCES home_connectors(slug) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_home_connector_readings_slug_received ON home_connector_readings(slug,received_at DESC);

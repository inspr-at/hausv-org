-- Organisationsweite Rollenregeln, persönliche Ausnahmen und Vorlagen (HAUSV-699).
CREATE TABLE organisation_capability_overrides (
    org_key TEXT NOT NULL CHECK (org_key <> ''),
    role_family TEXT NOT NULL CHECK (role_family IN ('verwaltung','eigentuemer','bewohner')),
    capability TEXT NOT NULL CHECK (capability NOT IN ('manage-users','platform-admin')),
    allowed BOOLEAN NOT NULL CHECK (allowed IN (0,1)),
    updated_at TEXT NOT NULL,
    updated_by TEXT NOT NULL,
    PRIMARY KEY (org_key, role_family, capability)
);
CREATE TABLE user_capability_grants (
    org_key TEXT NOT NULL CHECK (org_key <> ''),
    email TEXT NOT NULL,
    capability TEXT NOT NULL CHECK (capability NOT IN ('manage-users','platform-admin')),
    effect TEXT NOT NULL CHECK (effect IN ('grant','deny')),
    tenant_slug TEXT,
    updated_at TEXT NOT NULL,
    updated_by TEXT NOT NULL,
    CHECK (tenant_slug IS NULL OR tenant_slug <> '')
);
CREATE UNIQUE INDEX user_capability_grants_scope ON user_capability_grants(org_key,email,capability,COALESCE(tenant_slug,''));
CREATE TABLE capability_profiles (
    org_key TEXT NOT NULL CHECK (org_key <> ''),
    name TEXT NOT NULL,
    capabilities TEXT NOT NULL,
    PRIMARY KEY (org_key,name)
);

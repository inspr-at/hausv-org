-- HAUSV-582: freeze the editable annual-statement structure per period.
-- Units and parties remain canonical; only cost types, allocation keys and
-- allocation bases are snapshotted. Monetary rows stay in their existing
-- period tables and are deliberately never cloned with this structure.
CREATE TABLE annual_statement_period_cost_types (
    tenant_slug    TEXT NOT NULL,
    tenant_id      TEXT NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    period_year    INTEGER NOT NULL,
    key            TEXT NOT NULL,
    name           TEXT NOT NULL,
    allocatable    INTEGER NOT NULL,
    allocation_key TEXT NOT NULL DEFAULT '',
    updated_at     TEXT NOT NULL,
    updated_by     TEXT NOT NULL,
    PRIMARY KEY (tenant_slug, period_year, key),
    FOREIGN KEY (tenant_slug, period_year) REFERENCES annual_statement_periods(tenant_slug, year) ON DELETE CASCADE
);

CREATE INDEX idx_annual_statement_period_cost_types_tenant_id
    ON annual_statement_period_cost_types(tenant_id);

CREATE TABLE annual_statement_period_unit_bases (
    tenant_slug                  TEXT NOT NULL,
    tenant_id                    TEXT NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    period_year                  INTEGER NOT NULL,
    unit_id                      TEXT NOT NULL,
    miteigentumsanteil_ppm       INTEGER NOT NULL DEFAULT 0,
    usable_area_m2_hundredths    INTEGER NOT NULL DEFAULT 0,
    usable_area_recorded         INTEGER NOT NULL DEFAULT 0,
    persons                      INTEGER NOT NULL DEFAULT 0,
    persons_recorded             INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_slug, period_year, unit_id),
    FOREIGN KEY (tenant_slug, period_year) REFERENCES annual_statement_periods(tenant_slug, year) ON DELETE CASCADE
);

CREATE INDEX idx_annual_statement_period_unit_bases_tenant_id
    ON annual_statement_period_unit_bases(tenant_id);

-- Freeze every pre-existing period at migration time. This is intentionally a
-- migration write, never a page-render side effect. The JSON unit row is the
-- only historical source available before this migration; identities and
-- parties remain canonical and are not copied.
INSERT INTO annual_statement_period_cost_types(
    tenant_slug, tenant_id, period_year, key, name, allocatable,
    allocation_key, updated_at, updated_by
)
SELECT p.tenant_slug, p.tenant_id, p.year, c.key, c.name, c.allocatable,
       c.allocation_key, c.updated_at, c.updated_by
FROM annual_statement_periods p
JOIN annual_statement_cost_types c ON c.tenant_id = p.tenant_id;

INSERT INTO annual_statement_period_unit_bases(
    tenant_slug, tenant_id, period_year, unit_id, miteigentumsanteil_ppm,
    usable_area_m2_hundredths, usable_area_recorded, persons, persons_recorded
)
SELECT p.tenant_slug, p.tenant_id, p.year, u.id,
       COALESCE(CAST(json_extract(u.data, '$.miteigentumsanteil') AS INTEGER), 0),
       COALESCE(CAST(json_extract(u.data, '$.usable_area_m2_hundredths') AS INTEGER), 0),
       COALESCE(CAST(json_extract(u.data, '$.usable_area_recorded') AS INTEGER), 0),
       COALESCE(CAST(json_extract(u.data, '$.persons') AS INTEGER), 0),
       COALESCE(CAST(json_extract(u.data, '$.persons_recorded') AS INTEGER), 0)
FROM annual_statement_periods p
JOIN units u ON u.tenant_id = p.tenant_id;

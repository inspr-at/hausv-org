-- HAUSV-582: period-scoped structure snapshots. Units/parties stay canonical;
-- monetary rows remain in their existing period tables and are never cloned.
-- The migration runner is deliberately unscoped and existing tables are
-- fail-closed under RLS. Declare the maintenance scope transaction-locally so
-- the one-time backfill can see all tenants without leaking onto the pool.
SET LOCAL hausv.cross_tenant = 'on';

CREATE TABLE annual_statement_period_cost_types (
    tenant_slug    text NOT NULL DEFAULT '',
    tenant_id      varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    period_year    bigint NOT NULL DEFAULT 0,
    key            text NOT NULL DEFAULT '',
    name           text NOT NULL DEFAULT '',
    allocatable    boolean NOT NULL DEFAULT false,
    allocation_key text NOT NULL DEFAULT '',
    updated_at     text NOT NULL DEFAULT '',
    updated_by     text NOT NULL DEFAULT '',
    PRIMARY KEY (tenant_slug, period_year, key),
    FOREIGN KEY (tenant_slug, period_year) REFERENCES annual_statement_periods(tenant_slug, year) ON DELETE CASCADE
);
CREATE INDEX idx_annual_statement_period_cost_types_tenant_id
    ON annual_statement_period_cost_types(tenant_id);

ALTER TABLE annual_statement_period_cost_types ENABLE ROW LEVEL SECURITY;
ALTER TABLE annual_statement_period_cost_types FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON annual_statement_period_cost_types
    USING (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    )
    WITH CHECK (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    );
CREATE TRIGGER tenant_id_immutable
    BEFORE UPDATE OF tenant_id ON annual_statement_period_cost_types
    FOR EACH ROW EXECUTE FUNCTION hausv_reject_tenant_id_change();

CREATE TABLE annual_statement_period_unit_bases (
    tenant_slug                  text NOT NULL DEFAULT '',
    tenant_id                    varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    period_year                  bigint NOT NULL DEFAULT 0,
    unit_id                      text NOT NULL DEFAULT '',
    miteigentumsanteil_ppm       bigint NOT NULL DEFAULT 0,
    usable_area_m2_hundredths    bigint NOT NULL DEFAULT 0,
    usable_area_recorded         boolean NOT NULL DEFAULT false,
    persons                      bigint NOT NULL DEFAULT 0,
    persons_recorded             boolean NOT NULL DEFAULT false,
    PRIMARY KEY (tenant_slug, period_year, unit_id),
    FOREIGN KEY (tenant_slug, period_year) REFERENCES annual_statement_periods(tenant_slug, year) ON DELETE CASCADE
);
CREATE INDEX idx_annual_statement_period_unit_bases_tenant_id
    ON annual_statement_period_unit_bases(tenant_id);

ALTER TABLE annual_statement_period_unit_bases ENABLE ROW LEVEL SECURITY;
ALTER TABLE annual_statement_period_unit_bases FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON annual_statement_period_unit_bases
    USING (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    )
    WITH CHECK (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    );
CREATE TRIGGER tenant_id_immutable
    BEFORE UPDATE OF tenant_id ON annual_statement_period_unit_bases
    FOR EACH ROW EXECUTE FUNCTION hausv_reject_tenant_id_change();

-- Backfill pre-existing periods once, during migration. Rendering remains
-- read-only. Unit identities/parties are deliberately not snapshotted.
INSERT INTO annual_statement_period_cost_types(
    tenant_slug, tenant_id, period_year, key, name, allocatable,
    allocation_key, updated_at, updated_by
)
SELECT p.tenant_slug, p.tenant_id, p.year, c.key, c.name, c.allocatable,
       c.allocation_key, c.updated_at, c.updated_by
FROM annual_statement_periods p
JOIN annual_statement_cost_types c ON c.tenant_id = p.tenant_id;

-- Legacy pages displayed the starter catalogue without writing it. Freeze the
-- exact five starter rows only when the tenant's mutable catalogue is wholly
-- empty. A single customized row makes that catalogue authoritative, so it is
-- copied exactly above and is neither overwritten nor augmented here.
INSERT INTO annual_statement_period_cost_types(
    tenant_slug, tenant_id, period_year, key, name, allocatable,
    allocation_key, updated_at, updated_by
)
SELECT p.tenant_slug, p.tenant_id, p.year, defaults.key, defaults.name, true,
       'nutzwert', p.updated_at, p.updated_by
FROM annual_statement_periods p
CROSS JOIN (VALUES
    ('grundsteuer', 'Grundsteuer'),
    ('muellabfuhr', 'Müllabfuhr'),
    ('hausbetreuung', 'Hausbetreuung'),
    ('gebaeudeversicherung', 'Gebäudeversicherung'),
    ('gartenpflege', 'Gartenpflege')
) AS defaults(key, name)
WHERE NOT EXISTS (
    SELECT 1 FROM annual_statement_cost_types c WHERE c.tenant_id = p.tenant_id
);

INSERT INTO annual_statement_period_unit_bases(
    tenant_slug, tenant_id, period_year, unit_id, miteigentumsanteil_ppm,
    usable_area_m2_hundredths, usable_area_recorded, persons, persons_recorded
)
SELECT p.tenant_slug, p.tenant_id, p.year, u.id,
       COALESCE((u.data::jsonb ->> 'miteigentumsanteil')::bigint, 0),
       COALESCE((u.data::jsonb ->> 'usable_area_m2_hundredths')::bigint, 0),
       COALESCE((u.data::jsonb ->> 'usable_area_recorded')::boolean, false),
       COALESCE((u.data::jsonb ->> 'persons')::bigint, 0),
       COALESCE((u.data::jsonb ->> 'persons_recorded')::boolean, false)
FROM annual_statement_periods p
JOIN units u ON u.tenant_id = p.tenant_id;

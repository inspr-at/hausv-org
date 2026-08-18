-- Energy ids are globally unique on SQLite and were not on PostgreSQL.
--
-- Every energy table that carries an `id` declares it as `id TEXT PRIMARY KEY`
-- in SQLite migration 0026, so one id can belong to exactly one house. The
-- PostgreSQL target schema keys the same tables on (tenant_slug, id), which is
-- the right shape for tenancy but silently dropped that guarantee: two houses
-- could hold the same asset id, and nothing in the repository contradicted it
-- because the schema-agreement oracle compares table and column SETS only and
-- says so in its own header.
--
-- One place it is not merely a divergence but a hard failure:
-- energy.SQLStore.UpsertAsset arbitrates on ON CONFLICT(id), which is what makes
-- "this id already belongs to another house" a COLLISION it can detect and
-- refuse. Without a unique constraint on id alone, PostgreSQL rejects the
-- statement outright (SQLSTATE 42P10) — so UpsertAsset could not run at all —
-- and re-pointing the clause at (tenant_slug, id) would have made the insert
-- succeed instead, deleting the cross-tenant guard rather than porting it.
--
-- These indexes are additive: they refuse exactly what SQLite already refuses,
-- so any data or test that is legal on SQLite stays legal here. A pre-existing
-- duplicate would fail this migration loudly, which is the correct outcome —
-- such a row means two houses already disagree about who owns an id.
CREATE UNIQUE INDEX IF NOT EXISTS energy_assets_id_unique ON energy_assets(id);
CREATE UNIQUE INDEX IF NOT EXISTS energy_entity_mappings_id_unique ON energy_entity_mappings(id);
CREATE UNIQUE INDEX IF NOT EXISTS energy_imports_id_unique ON energy_imports(id);
CREATE UNIQUE INDEX IF NOT EXISTS energy_maintenance_plans_id_unique ON energy_maintenance_plans(id);
CREATE UNIQUE INDEX IF NOT EXISTS energy_tariff_assessments_id_unique ON energy_tariff_assessments(id);
CREATE UNIQUE INDEX IF NOT EXISTS energy_measures_id_unique ON energy_measures(id);

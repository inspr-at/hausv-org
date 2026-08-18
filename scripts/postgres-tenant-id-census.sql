-- Tenant-id census: is this database ready for PostgreSQL migration 0006?
--
-- Migration 0006 (internal/db/postgres/migrations/0006_rls_fail_closed.sql)
-- makes tenant_id NOT NULL on every RLS-governed table in the same transaction
-- that flips the row-level policy to fail-closed. ALTER ... SET NOT NULL refuses
-- if a single row has no identity, and the whole migration rolls back with it —
-- so running that release against a database that still holds an orphan FAILS
-- THE DEPLOY. That is the intended outcome. This file is how to know beforehand.
--
-- READ-ONLY. It issues no ALTER, UPDATE, INSERT or DELETE. It is safe to run
-- against production, and running it there is the owner's step, by hand:
--
--     psql "$DATABASE_URL" -f scripts/postgres-tenant-id-census.sql
--
-- as the APPLICATION role (NOSUPERUSER NOBYPASSRLS — the same role the migration
-- runs as). The SET below declares the maintenance lane exactly the way
-- internal/db/scoped.go's Unscoped does, in-session, so the counts are complete
-- under BOTH policy postures: under 0003 an undeclared session already sees
-- everything, and under 0006 only a declared one does. It is a session setting,
-- not a table write, and it ends with the session.
--
-- One row per governed table — the SAME catalog selection 0002/0003/0006 make:
-- every table with a tenant_id column except the registry itself — with:
--
--     rows_total      all rows the table holds
--     rows_no_tenant  rows whose tenant_id IS NULL: the orphans 0006 will refuse
--     not_null_now    whether the column is already NOT NULL (true after 0006)
--     verdict         READY            rows_no_tenant = 0
--                     BLOCKS-0006      rows_no_tenant > 0 on a table 0006 constrains
--                     EXEMPT (pre-tenant)  home_reservations: a reservation
--                                      precedes the house and its identity is
--                                      minted at activation; 0006 leaves it
--                                      nullable, so its NULLs never block
--
-- 0006 may run only when no table reads BLOCKS-0006. A blocking row is repaired
-- by the running product's boot pass (store.BackfillTenantIDs, which links a row
-- to the tenant its slug names and mints one if none exists) — the release
-- BEFORE 0006 must have booted against this database at least once with every
-- table READY. If it did and a row still shows here, its slug resolves to no
-- tenant (empty or whitespace); the boot pass names such rows in its error and
-- they need a human decision, not a script.
--
-- The dynamic count uses query_to_xml, which is the one way a single read-only
-- SQL statement can COUNT(*) over table names it reads from the catalog. It is
-- deliberately not a DO block or a function: nothing here creates anything.

SET hausv.cross_tenant = 'on';

SELECT
    c.table_name,
    (xpath('/row/n/text()', query_to_xml(
        format('SELECT count(*) AS n FROM %I', c.table_name), false, true, '')))[1]::text::bigint
        AS rows_total,
    (xpath('/row/n/text()', query_to_xml(
        format('SELECT count(*) AS n FROM %I WHERE tenant_id IS NULL', c.table_name), false, true, '')))[1]::text::bigint
        AS rows_no_tenant,
    (c.is_nullable = 'NO') AS not_null_now,
    CASE
        WHEN c.table_name = 'home_reservations' THEN 'EXEMPT (pre-tenant)'
        WHEN (xpath('/row/n/text()', query_to_xml(
            format('SELECT count(*) AS n FROM %I WHERE tenant_id IS NULL', c.table_name), false, true, '')))[1]::text::bigint > 0
            THEN 'BLOCKS-0006'
        ELSE 'READY'
    END AS verdict
FROM information_schema.columns c
JOIN pg_catalog.pg_tables t
  ON t.schemaname = c.table_schema AND t.tablename = c.table_name
WHERE c.table_schema = current_schema()
  AND c.column_name = 'tenant_id'
  AND c.table_name <> 'tenant'
ORDER BY c.table_name;

-- Flip the tenant isolation contract to fail-closed, and make tenant_id NOT
-- NULL, in one migration.
--
-- Migration 0003 wrote the policy as:
--
--     coalesce(current_setting('hausv.tenant_id', true), '') = ''
--         OR tenant_id = current_setting('hausv.tenant_id', true)
--
-- and said of itself: "it is not fail-closed for an unscoped connection. It
-- cannot be yet." The reason it gave was structural: every store issued its
-- queries on the pooled *sql.DB, which carries no scope, so a policy that hid
-- rows from an unscoped session would have hidden the entire database from the
-- application.
--
-- That reason is gone. Every SQL surface on the serve path now runs on a lane
-- (internal/db/scoped.go): store.TenantDB.For(tenant) opens connections whose
-- startup packet carries hausv.tenant_id = <ulid>, and Unscoped(reason) opens
-- connections whose startup packet carries hausv.cross_tenant = 'on'. The
-- process pool — the one handle that declares neither — no longer touches a
-- governed table on the serve path; a raw *sql.DB field is a test failure in
-- store, server and energy. So the escape can move again, this time from "the
-- session declared nothing" to "the session declared, in the call that opened
-- it, that it crosses tenants":
--
--     an unscoped session (neither GUC) sees NOTHING and can write NOTHING;
--     a tenant lane (hausv.tenant_id) sees exactly its own tenant;
--     the maintenance lane (hausv.cross_tenant = 'on') sees everything.
--
-- The GUC names and values match what internal/db/scoped.go sends, character
-- for character: tenantParam / crossTenantParam / crossTenantOn. A tenant lane
-- built from an unusable id carries the sentinel '-', which no row can hold
-- (tenant_id is a foreign key into tenant, whose CHECK admits only ULIDs), so
-- such a lane matches nothing rather than everything.
--
-- NOT NULL ships in the same file, on purpose. ALTER ... SET NOT NULL is the
-- loud check: it scans the whole table outside RLS and REFUSES if a single row
-- has no identity, and the whole migration — this transaction — rolls back with
-- it. Running this against a database that still holds an orphan therefore
-- FAILS THE DEPLOY, which is the correct outcome: under the fail-closed policy
-- an orphan row is unreachable from every tenant lane, and the boot-time
-- backfill (store.BackfillTenantIDs, still on the process pool) can no longer
-- see it to repair it. The census that shows, before anyone runs this, whether
-- a database would pass is scripts/postgres-tenant-id-census.sql — read-only,
-- runnable with psql as the application role, one row per governed table.
-- Running it against production is the owner's step, not this file's.
--
-- ONE table keeps a nullable tenant_id: home_reservations. A reservation is the
-- request for a house, made before the house exists; its identity is minted at
-- activation (store.tenantIDTables marks it preTenant, and the boot-time
-- backfill excludes it from the completeness check for the same reason). Its
-- store already runs every statement on the maintenance lane, so the policy
-- flip covers it fully — an unscoped session sees none of it — and the only
-- thing it is exempt from is the NOT NULL. That exemption is spelled out here
-- and in the census rather than derived, so it cannot widen silently.
--
-- The immutability trigger from 0003 is kept as written: with tenant_id NOT
-- NULL there is no NULL -> value transition left on any table but
-- home_reservations, where activation is exactly that transition.
--
-- What this does NOT do: it does not create the backup role, and it does not
-- assume superuser or BYPASSRLS anywhere — the migration runner is the
-- application role, NOSUPERUSER NOBYPASSRLS, verified at open.
--
-- Reversal is this one file: dropping it and recreating the 0003 policy is a
-- complete rollback, and 0003 is left byte-identical because it is history.
--
-- coalesce(...,'') on both settings is deliberate, for the reason 0003 gave:
-- once a session has run SET LOCAL for a custom GUC, resetting it can leave an
-- empty string rather than NULL. Here that only matters for the maintenance
-- test — an empty string must not read as 'on' — but reading NULL and '' as the
-- same thing keeps the two branches symmetrical and unsurprising.
DO $$
DECLARE
    table_name text;
BEGIN
    FOR table_name IN
        SELECT c.table_name
        FROM information_schema.columns c
        JOIN pg_catalog.pg_tables t
          ON t.schemaname = c.table_schema AND t.tablename = c.table_name
        WHERE c.table_schema = current_schema()
          AND c.column_name = 'tenant_id'
          AND c.table_name <> 'tenant'
        ORDER BY c.table_name
    LOOP
        IF table_name <> 'home_reservations' THEN
            EXECUTE format('ALTER TABLE %I ALTER COLUMN tenant_id SET NOT NULL', table_name);
        END IF;
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', table_name);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', table_name);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', table_name);
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON %I '
            'USING (coalesce(current_setting(''hausv.cross_tenant'', true), '''') = ''on'' '
            '       OR tenant_id = coalesce(current_setting(''hausv.tenant_id'', true), '''')) '
            'WITH CHECK (coalesce(current_setting(''hausv.cross_tenant'', true), '''') = ''on'' '
            '            OR tenant_id = coalesce(current_setting(''hausv.tenant_id'', true), ''''))',
            table_name
        );
        EXECUTE format('DROP TRIGGER IF EXISTS tenant_id_immutable ON %I', table_name);
        EXECUTE format(
            'CREATE TRIGGER tenant_id_immutable BEFORE UPDATE OF tenant_id ON %I FOR EACH ROW EXECUTE FUNCTION hausv_reject_tenant_id_change()',
            table_name
        );
    END LOOP;
END;
$$;

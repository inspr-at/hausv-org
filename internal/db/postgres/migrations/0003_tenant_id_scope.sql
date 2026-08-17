-- Re-base the tenant isolation contract onto tenant_id itself.
--
-- Migration 0002 wrote the policy as:
--
--     tenant_id IS NULL OR tenant_id = current_setting('hausv.tenant_id', true)
--
-- Both halves of that were wrong in the same way: they made a row's visibility
-- depend on tenant_id being EMPTY. Two consequences, both demonstrated rather
-- than argued:
--
--   1. A row with a NULL tenant_id was readable by EVERY tenant. A session
--      scoped to tenant B saw tenant A's un-backfilled rows. That is a
--      cross-tenant leak, not merely a visibility bug.
--   2. Because no writer populated tenant_id, the application only worked at
--      all thanks to that escape. The moment writes started carrying a real
--      tenant_id — which is what makes the query switch possible — every
--      unscoped INSERT was rejected by WITH CHECK and every unscoped SELECT
--      returned nothing.
--
-- The replacement below keys the escape on the SESSION instead of on the ROW:
--
--     an unscoped session (no hausv.tenant_id) sees everything;
--     a scoped session sees exactly its own tenant and nothing else.
--
-- What that tightens, relative to 0002:
--   * a NULL tenant_id row is now visible to NO scoped tenant, where before it
--     was visible to all of them;
--   * a scoped session is now strictly single-tenant.
--
-- What it does NOT do, stated plainly rather than left to be discovered: it is
-- not fail-closed for an unscoped connection. It cannot be yet. db.BeginTenantTx
-- still has zero production callers — every store issues its queries on the
-- pooled *sql.DB — and several operations are legitimately cross-tenant by
-- design (a person's houses, the attachment tombstone purge, connector lookup by
-- credential hash, retention purges). Making the unscoped case fail closed
-- requires routing every read and write through a scoped transaction AND giving
-- those cross-tenant operations an explicit way through. Until then the
-- application's own WHERE tenant_id = $1 is the primary control and this policy
-- is the second line, which is the honest description of where it stands.
--
-- coalesce(...,'') is deliberate: once a session has run SET LOCAL for a custom
-- GUC, resetting it can leave an empty string rather than NULL, and a policy
-- that only tested IS NULL would blank out every subsequent unscoped query on
-- that pooled connection.

-- The immutability trigger has to allow the one transition the repair needs.
-- Unconditional rejection made a missed backfill unrepairable: NULL IS DISTINCT
-- FROM '<ulid>' is TRUE, so assigning an identity to an orphan row raised
-- "tenant_id is immutable". Giving an unowned row its identity once is now
-- allowed; re-pointing an owned row at a different tenant, or erasing its
-- identity, still is not.
CREATE OR REPLACE FUNCTION hausv_reject_tenant_id_change()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.tenant_id IS NOT NULL AND NEW.tenant_id IS DISTINCT FROM OLD.tenant_id THEN
        RAISE EXCEPTION 'tenant_id is immutable';
    END IF;
    RETURN NEW;
END;
$$;

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
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', table_name);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', table_name);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', table_name);
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON %I '
            'USING (coalesce(current_setting(''hausv.tenant_id'', true), '''') = '''' '
            '       OR tenant_id = current_setting(''hausv.tenant_id'', true)) '
            'WITH CHECK (coalesce(current_setting(''hausv.tenant_id'', true), '''') = '''' '
            '            OR tenant_id = current_setting(''hausv.tenant_id'', true))',
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

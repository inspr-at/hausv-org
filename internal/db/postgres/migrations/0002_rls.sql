-- Install the same fail-closed RLS contract on every tenant_id table. This is
-- intentionally catalog-driven: adding a tenant-bound table without RLS is a
-- migration-time impossibility rather than a review convention.
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
            'CREATE POLICY tenant_isolation ON %I USING (tenant_id = current_setting(''hausv.tenant_id'', true)) WITH CHECK (tenant_id = current_setting(''hausv.tenant_id'', true))',
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

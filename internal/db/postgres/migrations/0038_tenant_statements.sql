-- HAUSV-798. JSON snapshots retain the exact lease, mandate and source run.
CREATE TABLE rental_management (
 tenant_id varchar(26) NOT NULL REFERENCES tenant(tenant_id),
 tenant_slug text NOT NULL,
 unit_id text NOT NULL,
 data text NOT NULL,
 PRIMARY KEY (tenant_id, unit_id)
);
CREATE TABLE tenant_statements (
 tenant_id varchar(26) NOT NULL REFERENCES tenant(tenant_id),
 tenant_slug text NOT NULL,
 id text NOT NULL,
 run_id text NOT NULL,
 unit_id text NOT NULL,
 data text NOT NULL,
 PRIMARY KEY (tenant_id, id)
);
CREATE TABLE tenant_statement_approvals (
 tenant_id varchar(26) NOT NULL REFERENCES tenant(tenant_id),
 tenant_slug text NOT NULL,
 statement_id text NOT NULL,
 data text NOT NULL,
 PRIMARY KEY (tenant_id, statement_id),
 FOREIGN KEY (tenant_id, statement_id) REFERENCES tenant_statements(tenant_id,id)
);
DO $$
DECLARE table_name text;
BEGIN
 FOREACH table_name IN ARRAY ARRAY['rental_management','tenant_statements','tenant_statement_approvals'] LOOP
  EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', table_name);
  EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', table_name);
  EXECUTE format('CREATE POLICY tenant_isolation ON %I USING (coalesce(current_setting(''hausv.cross_tenant'', true), '''') = ''on'' OR tenant_id = coalesce(current_setting(''hausv.tenant_id'', true), '''')) WITH CHECK (coalesce(current_setting(''hausv.cross_tenant'', true), '''') = ''on'' OR tenant_id = coalesce(current_setting(''hausv.tenant_id'', true), ''''))', table_name);
  EXECUTE format('CREATE TRIGGER tenant_id_immutable BEFORE UPDATE OF tenant_id ON %I FOR EACH ROW EXECUTE FUNCTION hausv_reject_tenant_id_change()', table_name);
 END LOOP;
END;
$$;
CREATE FUNCTION hausv_reject_tenant_statement_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'tenant statements and approvals are immutable' USING ERRCODE = '23514'; END;
$$;
CREATE TRIGGER tenant_statement_immutable BEFORE UPDATE OR DELETE ON tenant_statements FOR EACH ROW EXECUTE FUNCTION hausv_reject_tenant_statement_change();
CREATE TRIGGER tenant_statement_approval_immutable BEFORE UPDATE OR DELETE ON tenant_statement_approvals FOR EACH ROW EXECUTE FUNCTION hausv_reject_tenant_statement_change();

CREATE TABLE annual_statement_run_approvals (
 tenant_id varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
 tenant_slug text NOT NULL DEFAULT '',
 run_id text NOT NULL,
 data text NOT NULL,
 PRIMARY KEY (tenant_id, run_id)
);
ALTER TABLE annual_statement_run_approvals ENABLE ROW LEVEL SECURITY;
ALTER TABLE annual_statement_run_approvals FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON annual_statement_run_approvals
 USING (coalesce(current_setting('hausv.cross_tenant', true), '') = 'on' OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), ''))
 WITH CHECK (coalesce(current_setting('hausv.cross_tenant', true), '') = 'on' OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), ''));
CREATE TRIGGER annual_statement_approval_immutable BEFORE UPDATE OR DELETE ON annual_statement_run_approvals
 FOR EACH ROW EXECUTE FUNCTION hausv_reject_annual_statement_run_change();

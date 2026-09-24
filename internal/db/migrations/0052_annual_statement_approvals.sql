CREATE TABLE annual_statement_run_approvals (
 tenant_id text NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
 tenant_slug text NOT NULL DEFAULT '',
 run_id text NOT NULL,
 data text NOT NULL,
 PRIMARY KEY (tenant_id, run_id)
);

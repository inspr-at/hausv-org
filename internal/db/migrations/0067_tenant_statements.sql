-- HAUSV-798: optional rental management and immutable derived statements.
CREATE TABLE rental_management (
 tenant_id TEXT NOT NULL REFERENCES tenant(tenant_id),
 tenant_slug TEXT NOT NULL,
 unit_id TEXT NOT NULL,
 data TEXT NOT NULL,
 PRIMARY KEY (tenant_id, unit_id)
);
CREATE TABLE tenant_statements (
 tenant_id TEXT NOT NULL REFERENCES tenant(tenant_id),
 tenant_slug TEXT NOT NULL,
 id TEXT NOT NULL,
 run_id TEXT NOT NULL,
 unit_id TEXT NOT NULL,
 data TEXT NOT NULL,
 PRIMARY KEY (tenant_id, id)
);
CREATE TABLE tenant_statement_approvals (
 tenant_id TEXT NOT NULL REFERENCES tenant(tenant_id),
 tenant_slug TEXT NOT NULL,
 statement_id TEXT NOT NULL,
 data TEXT NOT NULL,
 PRIMARY KEY (tenant_id, statement_id),
 FOREIGN KEY (tenant_id, statement_id) REFERENCES tenant_statements(tenant_id,id)
);
CREATE TRIGGER tenant_statement_immutable_update BEFORE UPDATE ON tenant_statements
BEGIN SELECT RAISE(ABORT, 'tenant_statements are immutable'); END;
CREATE TRIGGER tenant_statement_immutable_delete BEFORE DELETE ON tenant_statements
BEGIN SELECT RAISE(ABORT, 'tenant_statements are immutable'); END;
CREATE TRIGGER tenant_statement_approval_immutable_update BEFORE UPDATE ON tenant_statement_approvals
BEGIN SELECT RAISE(ABORT, 'tenant_statement_approvals are immutable'); END;
CREATE TRIGGER tenant_statement_approval_immutable_delete BEFORE DELETE ON tenant_statement_approvals
BEGIN SELECT RAISE(ABORT, 'tenant_statement_approvals are immutable'); END;

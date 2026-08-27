-- Tenant-owned cost-type catalogue for annual operating-cost statements.
CREATE TABLE IF NOT EXISTS annual_statement_cost_types (
    tenant_slug text NOT NULL DEFAULT '',
    tenant_id varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    key text NOT NULL DEFAULT '',
    name text NOT NULL DEFAULT '',
    allocatable boolean NOT NULL DEFAULT false,
    updated_at text NOT NULL DEFAULT '',
    updated_by text NOT NULL DEFAULT '',
    PRIMARY KEY (tenant_slug, key)
);
CREATE INDEX IF NOT EXISTS idx_annual_statement_cost_types_tenant_id
    ON annual_statement_cost_types(tenant_id);

ALTER TABLE annual_statement_cost_types ENABLE ROW LEVEL SECURITY;
ALTER TABLE annual_statement_cost_types FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON annual_statement_cost_types
    USING (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    )
    WITH CHECK (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    );
CREATE TRIGGER tenant_id_immutable
    BEFORE UPDATE OF tenant_id ON annual_statement_cost_types
    FOR EACH ROW EXECUTE FUNCTION hausv_reject_tenant_id_change();

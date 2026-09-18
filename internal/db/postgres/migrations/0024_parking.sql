-- Parking and charging state as one JSON document per tenant (HAUSV-758).
CREATE TABLE parking (
    tenant_id varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    tenant_slug text NOT NULL DEFAULT '',
    data text NOT NULL DEFAULT '{}',
    PRIMARY KEY (tenant_slug)
);
CREATE INDEX idx_parking_tenant_id ON parking(tenant_id);
ALTER TABLE parking ENABLE ROW LEVEL SECURITY;
ALTER TABLE parking FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON parking
    USING (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    )
    WITH CHECK (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    );
CREATE TRIGGER tenant_id_immutable
    BEFORE UPDATE OF tenant_id ON parking
    FOR EACH ROW EXECUTE FUNCTION hausv_reject_tenant_id_change();

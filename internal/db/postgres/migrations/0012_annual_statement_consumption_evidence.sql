-- Append-only cumulative consumption evidence for period-bound annual-
-- statement inputs. Defaults keep the generic RLS schema probe able to seed a
-- row while real writes always supply every field.
CREATE TABLE IF NOT EXISTS annual_statement_consumption_evidence (
    tenant_id       varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    tenant_slug     text NOT NULL DEFAULT '',
    source_key      text NOT NULL DEFAULT '',
    unit_id         text NOT NULL DEFAULT '',
    cost_type_key   text NOT NULL DEFAULT '',
    source_kind     text NOT NULL DEFAULT 'entity' CHECK (source_kind IN ('entity', 'asset')),
    source_id       text NOT NULL DEFAULT '',
    measured_at_ns  bigint NOT NULL DEFAULT 0,
    value_micros    bigint NOT NULL DEFAULT 0 CHECK (value_micros >= 0),
    measurement_unit text NOT NULL DEFAULT '',
    received_at_ns  bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, source_key)
);

CREATE INDEX IF NOT EXISTS idx_annual_statement_consumption_evidence_tenant_period
    ON annual_statement_consumption_evidence(tenant_id, cost_type_key, measured_at_ns);

ALTER TABLE annual_statement_consumption_evidence ENABLE ROW LEVEL SECURITY;
ALTER TABLE annual_statement_consumption_evidence FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON annual_statement_consumption_evidence
    USING (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    )
    WITH CHECK (
        coalesce(current_setting('hausv.cross_tenant', true), '') = 'on'
        OR tenant_id = coalesce(current_setting('hausv.tenant_id', true), '')
    );
CREATE TRIGGER tenant_id_immutable
    BEFORE UPDATE OF tenant_id ON annual_statement_consumption_evidence
    FOR EACH ROW EXECUTE FUNCTION hausv_reject_tenant_id_change();

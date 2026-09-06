-- Append-only cumulative consumption evidence for period-bound annual-
-- statement inputs. This is deliberately separate from
-- home_connector_readings, whose latest-per-entity semantics remain unchanged.
CREATE TABLE IF NOT EXISTS annual_statement_consumption_evidence (
    tenant_id       TEXT NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    tenant_slug     TEXT NOT NULL,
    source_key      TEXT NOT NULL,
    unit_id         TEXT NOT NULL,
    cost_type_key   TEXT NOT NULL,
    source_kind     TEXT NOT NULL CHECK (source_kind IN ('entity', 'asset')),
    source_id       TEXT NOT NULL,
    measured_at_ns  INTEGER NOT NULL,
    value_micros    INTEGER NOT NULL CHECK (value_micros >= 0),
    measurement_unit TEXT NOT NULL,
    received_at_ns  INTEGER NOT NULL,
    PRIMARY KEY (tenant_id, source_key)
);

CREATE INDEX IF NOT EXISTS idx_annual_statement_consumption_evidence_tenant_period
    ON annual_statement_consumption_evidence(tenant_id, cost_type_key, measured_at_ns);

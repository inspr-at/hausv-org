-- Lease model per unit (HAUSV-768). Money is integer cents. Index bases stay
-- decimal text exactly as published. Rent components are versioned by
-- valid_from and are never updated in place. valorisation_state is derived
-- and stays empty until a later slice fills it.
CREATE TABLE leases (
    id TEXT NOT NULL PRIMARY KEY,
    tenant_slug TEXT NOT NULL,
    tenant_id TEXT NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    unit_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('draft','active','ended')),
    concluded_on TEXT NOT NULL,
    starts_on TEXT NOT NULL,
    ends_on TEXT,
    lease_kind TEXT NOT NULL CHECK (lease_kind IN ('hauptmiete','untermiete')),
    use_kind TEXT NOT NULL CHECK (use_kind IN ('wohnung','geschaeft','garage','sonstiges')),
    mrg_scope TEXT NOT NULL CHECK (mrg_scope IN ('voll','teil','ausnahme','wgg')),
    rent_regime TEXT NOT NULL CHECK (rent_regime IN ('richtwert','kategorie','angemessen','frei','sonstig')),
    price_restricted INTEGER NOT NULL CHECK (price_restricted IN (0,1)),
    max_hmz_cents INTEGER,
    landlord_is_business INTEGER NOT NULL CHECK (landlord_is_business IN (0,1)),
    tenant_is_consumer INTEGER NOT NULL CHECK (tenant_is_consumer IN (0,1)),
    zinstermin_day INTEGER NOT NULL DEFAULT 5 CHECK (zinstermin_day BETWEEN 1 AND 28),
    vat_opted INTEGER NOT NULL CHECK (vat_opted IN (0,1)),
    notes TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL,
    updated_by TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_leases_tenant_id ON leases(tenant_id);
CREATE INDEX idx_leases_tenant_unit ON leases(tenant_id, unit_id);

CREATE TABLE lease_parties (
    id TEXT NOT NULL PRIMARY KEY,
    lease_id TEXT NOT NULL REFERENCES leases(id) ON UPDATE RESTRICT ON DELETE CASCADE,
    tenant_slug TEXT NOT NULL,
    tenant_id TEXT NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    name TEXT NOT NULL,
    address TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL CHECK (role IN ('hauptmieter','mitmieter')),
    valid_from TEXT NOT NULL,
    valid_to TEXT
);
CREATE INDEX idx_lease_parties_tenant_id ON lease_parties(tenant_id);
CREATE INDEX idx_lease_parties_lease ON lease_parties(lease_id);

CREATE TABLE rent_components (
    id TEXT NOT NULL PRIMARY KEY,
    lease_id TEXT NOT NULL REFERENCES leases(id) ON UPDATE RESTRICT ON DELETE CASCADE,
    tenant_slug TEXT NOT NULL,
    tenant_id TEXT NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('hmz','bk_akonto','heiz_akonto','lift','moebel','stellplatz','sonstiges')),
    net_cents INTEGER NOT NULL,
    vat_rate_bp INTEGER NOT NULL CHECK (vat_rate_bp >= 0),
    valid_from TEXT NOT NULL,
    origin TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_rent_components_tenant_id ON rent_components(tenant_id);
CREATE INDEX idx_rent_components_lease ON rent_components(lease_id);

CREATE TABLE index_clauses (
    id TEXT NOT NULL PRIMARY KEY,
    lease_id TEXT NOT NULL REFERENCES leases(id) ON UPDATE RESTRICT ON DELETE CASCADE,
    tenant_slug TEXT NOT NULL,
    tenant_id TEXT NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    component_kind TEXT NOT NULL DEFAULT 'hmz',
    clause_type TEXT NOT NULL CHECK (clause_type IN ('mieweg_model','vpi_threshold','vpi_periodic','staffel','none')),
    series TEXT NOT NULL DEFAULT '',
    base_period TEXT NOT NULL DEFAULT '',
    base_value TEXT NOT NULL DEFAULT '',
    threshold_kind TEXT,
    threshold_value TEXT,
    threshold_inclusive INTEGER NOT NULL DEFAULT 0 CHECK (threshold_inclusive IN (0,1)),
    full_change_on_trigger INTEGER NOT NULL DEFAULT 1 CHECK (full_change_on_trigger IN (0,1)),
    two_way INTEGER NOT NULL DEFAULT 1 CHECK (two_way IN (0,1)),
    pct_rounding TEXT NOT NULL DEFAULT 'none' CHECK (pct_rounding IN ('none','one_decimal')),
    periodic_month INTEGER,
    reference_month_offset INTEGER,
    clause_text TEXT NOT NULL DEFAULT '',
    review_status TEXT NOT NULL DEFAULT 'unreviewed' CHECK (review_status IN ('unreviewed','ok','doubtful','invalid')),
    review_note TEXT NOT NULL DEFAULT '',
    valid_from TEXT NOT NULL
);
CREATE INDEX idx_index_clauses_tenant_id ON index_clauses(tenant_id);
CREATE INDEX idx_index_clauses_lease ON index_clauses(lease_id);

CREATE TABLE valorisation_state (
    clause_id TEXT NOT NULL PRIMARY KEY REFERENCES index_clauses(id) ON UPDATE RESTRICT ON DELETE CASCADE,
    tenant_slug TEXT NOT NULL,
    tenant_id TEXT NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    contract_value TEXT NOT NULL DEFAULT '',
    contract_base_period TEXT NOT NULL DEFAULT '',
    contract_base_value TEXT NOT NULL DEFAULT '',
    cap_value TEXT NOT NULL DEFAULT '',
    cap_anchor_period TEXT NOT NULL DEFAULT '',
    cap_last_year INTEGER,
    last_effective_on TEXT,
    last_run_item_id TEXT
);
CREATE INDEX idx_valorisation_state_tenant_id ON valorisation_state(tenant_id);

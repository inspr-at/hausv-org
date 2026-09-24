-- Lease model per unit (HAUSV-768). Money is integer cents. Index bases stay
-- decimal text exactly as published. Rent components are versioned by
-- valid_from and are never updated in place.
CREATE TABLE leases (
    id text NOT NULL PRIMARY KEY,
    tenant_slug text NOT NULL,
    tenant_id varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    unit_id text NOT NULL,
    status text NOT NULL CHECK (status IN ('draft','active','ended')),
    concluded_on text NOT NULL,
    starts_on text NOT NULL,
    ends_on text,
    lease_kind text NOT NULL CHECK (lease_kind IN ('hauptmiete','untermiete')),
    use_kind text NOT NULL CHECK (use_kind IN ('wohnung','geschaeft','garage','sonstiges')),
    mrg_scope text NOT NULL CHECK (mrg_scope IN ('voll','teil','ausnahme','wgg')),
    rent_regime text NOT NULL CHECK (rent_regime IN ('richtwert','kategorie','angemessen','frei','sonstig')),
    price_restricted boolean NOT NULL,
    max_hmz_cents bigint,
    landlord_is_business boolean NOT NULL,
    tenant_is_consumer boolean NOT NULL,
    zinstermin_day integer NOT NULL DEFAULT 5 CHECK (zinstermin_day BETWEEN 1 AND 28),
    vat_opted boolean NOT NULL,
    notes text NOT NULL DEFAULT '',
    updated_at text NOT NULL,
    updated_by text NOT NULL DEFAULT ''
);
CREATE INDEX idx_leases_tenant_id ON leases(tenant_id);
CREATE INDEX idx_leases_tenant_unit ON leases(tenant_id, unit_id);

CREATE TABLE lease_parties (
    id text NOT NULL PRIMARY KEY,
    lease_id text NOT NULL REFERENCES leases(id) ON UPDATE RESTRICT ON DELETE CASCADE,
    tenant_slug text NOT NULL,
    tenant_id varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    name text NOT NULL,
    address text NOT NULL DEFAULT '',
    email text NOT NULL DEFAULT '',
    role text NOT NULL CHECK (role IN ('hauptmieter','mitmieter')),
    valid_from text NOT NULL,
    valid_to text
);
CREATE INDEX idx_lease_parties_tenant_id ON lease_parties(tenant_id);
CREATE INDEX idx_lease_parties_lease ON lease_parties(lease_id);

CREATE TABLE rent_components (
    id text NOT NULL PRIMARY KEY,
    lease_id text NOT NULL REFERENCES leases(id) ON UPDATE RESTRICT ON DELETE CASCADE,
    tenant_slug text NOT NULL,
    tenant_id varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('hmz','bk_akonto','heiz_akonto','lift','moebel','stellplatz','sonstiges')),
    net_cents bigint NOT NULL,
    vat_rate_bp integer NOT NULL CHECK (vat_rate_bp >= 0),
    valid_from text NOT NULL,
    origin text NOT NULL,
    created_at text NOT NULL
);
CREATE INDEX idx_rent_components_tenant_id ON rent_components(tenant_id);
CREATE INDEX idx_rent_components_lease ON rent_components(lease_id);

CREATE TABLE index_clauses (
    id text NOT NULL PRIMARY KEY,
    lease_id text NOT NULL REFERENCES leases(id) ON UPDATE RESTRICT ON DELETE CASCADE,
    tenant_slug text NOT NULL,
    tenant_id varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    component_kind text NOT NULL DEFAULT 'hmz',
    clause_type text NOT NULL CHECK (clause_type IN ('mieweg_model','vpi_threshold','vpi_periodic','staffel','none')),
    series text NOT NULL DEFAULT '',
    base_period text NOT NULL DEFAULT '',
    base_value text NOT NULL DEFAULT '',
    threshold_kind text,
    threshold_value text,
    threshold_inclusive boolean NOT NULL DEFAULT false,
    full_change_on_trigger boolean NOT NULL DEFAULT true,
    two_way boolean NOT NULL DEFAULT true,
    pct_rounding text NOT NULL DEFAULT 'none' CHECK (pct_rounding IN ('none','one_decimal')),
    periodic_month integer,
    reference_month_offset integer,
    clause_text text NOT NULL DEFAULT '',
    review_status text NOT NULL DEFAULT 'unreviewed' CHECK (review_status IN ('unreviewed','ok','doubtful','invalid')),
    review_note text NOT NULL DEFAULT '',
    valid_from text NOT NULL
);
CREATE INDEX idx_index_clauses_tenant_id ON index_clauses(tenant_id);
CREATE INDEX idx_index_clauses_lease ON index_clauses(lease_id);

CREATE TABLE valorisation_state (
    clause_id text NOT NULL PRIMARY KEY REFERENCES index_clauses(id) ON UPDATE RESTRICT ON DELETE CASCADE,
    tenant_slug text NOT NULL,
    tenant_id varchar(26) NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    contract_value text NOT NULL DEFAULT '',
    contract_base_period text NOT NULL DEFAULT '',
    contract_base_value text NOT NULL DEFAULT '',
    cap_value text NOT NULL DEFAULT '',
    cap_anchor_period text NOT NULL DEFAULT '',
    cap_last_year integer,
    last_effective_on text,
    last_run_item_id text
);
CREATE INDEX idx_valorisation_state_tenant_id ON valorisation_state(tenant_id);

DO $$
DECLARE table_name text;
BEGIN
    FOREACH table_name IN ARRAY ARRAY['leases','lease_parties','rent_components','index_clauses','valorisation_state'] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', table_name);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', table_name);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', table_name);
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON %I USING (' ||
            'coalesce(current_setting(''hausv.cross_tenant'', true), '''') = ''on'' ' ||
            'OR tenant_id = coalesce(current_setting(''hausv.tenant_id'', true), '''')) ' ||
            'WITH CHECK (coalesce(current_setting(''hausv.cross_tenant'', true), '''') = ''on'' ' ||
            'OR tenant_id = coalesce(current_setting(''hausv.tenant_id'', true), ''''))', table_name
        );
        EXECUTE format(
            'CREATE TRIGGER tenant_id_immutable BEFORE UPDATE OF tenant_id ON %I ' ||
            'FOR EACH ROW EXECUTE FUNCTION hausv_reject_tenant_id_change()', table_name
        );
    END LOOP;
END;
$$;

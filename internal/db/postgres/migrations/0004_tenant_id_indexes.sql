-- Index the column every store read now filters on.
--
-- SQLite migration 0033 gave all 25 tenant-scoped tables a tenant_id column AND
-- an index on it. The PostgreSQL side got the column and nothing else: nine
-- index references to tenant_slug, zero to tenant_id. The moment the query layer
-- switched from the renameable label to the immutable identity, every one of
-- those reads became a sequential scan on PostgreSQL — measured on announcements
-- as Seq Scan cost 394 against Bitmap Index Scan cost 143 for the same predicate
-- before the switch.
--
-- This is not a live regression: production runs SQLite, where the indexes have
-- existed since 0033. It is a cutover blocker, which is why it lands with the
-- switch rather than after it.
--
-- Names match the SQLite side exactly, so a reader comparing the two schemas
-- sees the same objects rather than having to map them.
--
-- The composite tenant_slug indexes from 0001 are left in place: the rollback
-- window is still open and the previous release addresses rows by the label.
--
-- Deliberately NOT touched: the RLS policies and the tenant_id_immutable trigger
-- from 0003. Re-basing a security contract is a separate decision.

CREATE INDEX IF NOT EXISTS idx_unit_payment_status_tenant_id ON unit_payment_status(tenant_id);
CREATE INDEX IF NOT EXISTS idx_contacts_tenant_id ON contacts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_announcement_reads_tenant_id ON announcement_reads(tenant_id);
CREATE INDEX IF NOT EXISTS idx_announcements_tenant_id ON announcements(tenant_id);
CREATE INDEX IF NOT EXISTS idx_events_tenant_id ON events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_handovers_tenant_id ON handovers(tenant_id);
CREATE INDEX IF NOT EXISTS idx_documents_tenant_id ON documents(tenant_id);
CREATE INDEX IF NOT EXISTS idx_attachments_tenant_id ON attachments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_units_tenant_id ON units(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ballots_tenant_id ON ballots(tenant_id);
CREATE INDEX IF NOT EXISTS idx_issues_tenant_id ON issues(tenant_id);
CREATE INDEX IF NOT EXISTS idx_house_memberships_tenant_id ON house_memberships(tenant_id);
CREATE INDEX IF NOT EXISTS idx_integration_imports_tenant_id ON integration_imports(tenant_id);
CREATE INDEX IF NOT EXISTS idx_home_profiles_tenant_id ON home_profiles(tenant_id);
CREATE INDEX IF NOT EXISTS idx_energy_assets_tenant_id ON energy_assets(tenant_id);
CREATE INDEX IF NOT EXISTS idx_energy_entity_mappings_tenant_id ON energy_entity_mappings(tenant_id);
CREATE INDEX IF NOT EXISTS idx_energy_intervals_tenant_id ON energy_intervals(tenant_id);
CREATE INDEX IF NOT EXISTS idx_energy_imports_tenant_id ON energy_imports(tenant_id);
CREATE INDEX IF NOT EXISTS idx_energy_maintenance_plans_tenant_id ON energy_maintenance_plans(tenant_id);
CREATE INDEX IF NOT EXISTS idx_energy_tariff_assessments_tenant_id ON energy_tariff_assessments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_energy_measures_tenant_id ON energy_measures(tenant_id);
CREATE INDEX IF NOT EXISTS idx_home_reservations_tenant_id ON home_reservations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_home_portals_tenant_id ON home_portals(tenant_id);
CREATE INDEX IF NOT EXISTS idx_home_connectors_tenant_id ON home_connectors(tenant_id);
CREATE INDEX IF NOT EXISTS idx_home_connector_readings_tenant_id ON home_connector_readings(tenant_id);

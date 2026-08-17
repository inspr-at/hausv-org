-- Add the immutable tenant identity beside the legacy slug. The slug remains
-- the operational key during the compatibility window; tenant_id is nullable
-- because migration 0032 deliberately did not invent identities for tenants
-- that are still configured outside the database.

ALTER TABLE unit_payment_status ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE contacts ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE announcement_reads ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE announcements ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE events ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE handovers ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE documents ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE attachments ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE units ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE ballots ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE issues ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE house_memberships ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE integration_imports ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE home_profiles ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE energy_assets ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE energy_entity_mappings ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE energy_intervals ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE energy_imports ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE energy_maintenance_plans ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE energy_tariff_assessments ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE energy_measures ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE home_reservations ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE home_portals ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE home_connectors ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE home_connector_readings ADD COLUMN tenant_id TEXT REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE;

UPDATE unit_payment_status SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=unit_payment_status.tenant_slug);
UPDATE contacts SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=contacts.tenant_slug);
UPDATE announcement_reads SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=announcement_reads.tenant_slug);
UPDATE announcements SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=announcements.tenant_slug);
UPDATE events SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=events.tenant_slug);
UPDATE handovers SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=handovers.tenant_slug);
UPDATE documents SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=documents.tenant_slug);
UPDATE attachments SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=attachments.tenant_slug);
UPDATE units SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=units.tenant_slug);
UPDATE ballots SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=ballots.tenant_slug);
UPDATE issues SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=issues.tenant_slug);
UPDATE house_memberships SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=house_memberships.tenant_slug);
UPDATE integration_imports SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=integration_imports.tenant_slug);
UPDATE home_profiles SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=home_profiles.tenant_slug);
UPDATE energy_assets SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=energy_assets.tenant_slug);
UPDATE energy_entity_mappings SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=energy_entity_mappings.tenant_slug);
UPDATE energy_intervals SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=energy_intervals.tenant_slug);
UPDATE energy_imports SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=energy_imports.tenant_slug);
UPDATE energy_maintenance_plans SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=energy_maintenance_plans.tenant_slug);
UPDATE energy_tariff_assessments SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=energy_tariff_assessments.tenant_slug);
UPDATE energy_measures SET tenant_id=(SELECT tenant_id FROM tenant WHERE slug=energy_measures.tenant_slug);

CREATE INDEX idx_unit_payment_status_tenant_id ON unit_payment_status(tenant_id);
CREATE INDEX idx_contacts_tenant_id ON contacts(tenant_id);
CREATE INDEX idx_announcement_reads_tenant_id ON announcement_reads(tenant_id);
CREATE INDEX idx_announcements_tenant_id ON announcements(tenant_id);
CREATE INDEX idx_events_tenant_id ON events(tenant_id);
CREATE INDEX idx_handovers_tenant_id ON handovers(tenant_id);
CREATE INDEX idx_documents_tenant_id ON documents(tenant_id);
CREATE INDEX idx_attachments_tenant_id ON attachments(tenant_id);
CREATE INDEX idx_units_tenant_id ON units(tenant_id);
CREATE INDEX idx_ballots_tenant_id ON ballots(tenant_id);
CREATE INDEX idx_issues_tenant_id ON issues(tenant_id);
CREATE INDEX idx_house_memberships_tenant_id ON house_memberships(tenant_id);
CREATE INDEX idx_integration_imports_tenant_id ON integration_imports(tenant_id);
CREATE INDEX idx_home_profiles_tenant_id ON home_profiles(tenant_id);
CREATE INDEX idx_energy_assets_tenant_id ON energy_assets(tenant_id);
CREATE INDEX idx_energy_entity_mappings_tenant_id ON energy_entity_mappings(tenant_id);
CREATE INDEX idx_energy_intervals_tenant_id ON energy_intervals(tenant_id);
CREATE INDEX idx_energy_imports_tenant_id ON energy_imports(tenant_id);
CREATE INDEX idx_energy_maintenance_plans_tenant_id ON energy_maintenance_plans(tenant_id);
CREATE INDEX idx_energy_tariff_assessments_tenant_id ON energy_tariff_assessments(tenant_id);
CREATE INDEX idx_energy_measures_tenant_id ON energy_measures(tenant_id);
CREATE INDEX idx_home_reservations_tenant_id ON home_reservations(tenant_id);
CREATE INDEX idx_home_portals_tenant_id ON home_portals(tenant_id);
CREATE INDEX idx_home_connectors_tenant_id ON home_connectors(tenant_id);
CREATE INDEX idx_home_connector_readings_tenant_id ON home_connector_readings(tenant_id);

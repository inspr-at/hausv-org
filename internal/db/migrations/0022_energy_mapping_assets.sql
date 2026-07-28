ALTER TABLE energy_entity_mappings
    ADD COLUMN asset_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS energy_entity_mappings_tenant_asset
    ON energy_entity_mappings(tenant_slug, asset_id, confirmed);

-- Optionally scope a home-energy profile to one administrative unit.
ALTER TABLE home_profiles
    ADD COLUMN unit_id TEXT NOT NULL DEFAULT '';

-- This is the only safe moment to infer a legacy apartment: the inventory is
-- stable inside the migration transaction, and exactly one residential unit is
-- unambiguous. Runtime inventory changes must never silently move historical
-- energy data between homes.
UPDATE home_profiles
SET unit_id = (
    SELECT units.id
    FROM units
    WHERE units.tenant_slug = home_profiles.tenant_slug
      AND COALESCE(
            NULLIF(LOWER(TRIM(json_extract(units.data, '$.unit_type'))), ''),
            'residential'
          ) IN ('residential', 'wohnung', 'wohneinheit')
    LIMIT 1
)
WHERE home_profiles.home_type = 'apartment'
  AND home_profiles.unit_id = ''
  AND (
      SELECT COUNT(*)
      FROM units
      WHERE units.tenant_slug = home_profiles.tenant_slug
        AND COALESCE(
              NULLIF(LOWER(TRIM(json_extract(units.data, '$.unit_type'))), ''),
              'residential'
            ) IN ('residential', 'wohnung', 'wohneinheit')
  ) = 1;

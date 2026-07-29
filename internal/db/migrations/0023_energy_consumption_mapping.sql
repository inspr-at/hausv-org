-- Early energy discovery treated a few battery-prefixed house-consumption
-- sensors as battery flow. Their names are unambiguous, so normalize only this
-- narrow legacy case and keep every other confirmed/manual mapping untouched.
UPDATE energy_entity_mappings
SET metric = 'load-power',
    asset_id = '',
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE metric = 'battery-power'
  AND (
    lower(entity_id) LIKE '%home_consumption%'
    OR lower(entity_id) LIKE '%consumption_current%'
    OR lower(display_name) LIKE '%home current consumption%'
    OR lower(display_name) LIKE '%hausverbrauch%'
  );

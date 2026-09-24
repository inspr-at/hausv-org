-- HAUSV-801: organisation readings and an optional lease override.
ALTER TABLE leases ADD COLUMN wirksamwerden_mode TEXT NOT NULL DEFAULT '' CHECK(wirksamwerden_mode IN ('','wko','oevi','contract'));
SET LOCAL hausv.cross_tenant = 'on';
UPDATE org_settings SET data=jsonb_set(data::jsonb, '{valorisation,wirksamwerden_mode}',
 to_jsonb(CASE data::jsonb #>> '{valorisation,wirksamwerden_mode}' WHEN 'contractual' THEN 'contract' ELSE 'wko' END))::text
 WHERE data::jsonb #>> '{valorisation,wirksamwerden_mode}' IN ('cautious','contractual');

-- Delivery evidence stays separate from the frozen calculation.
ALTER TABLE valorisation_deliveries ADD COLUMN received_on TEXT NOT NULL DEFAULT '';
ALTER TABLE valorisation_deliveries ADD COLUMN receipt_due_on TEXT NOT NULL DEFAULT '';

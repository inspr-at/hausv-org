ALTER TABLE home_profiles ADD COLUMN free_until_at TEXT;

-- Existing HAUSV-Home pilots keep the three-year end that the product already
-- displayed. New profiles write their explicit twelve-month end in the app.
UPDATE home_profiles
SET free_until_at = strftime('%Y-%m-%dT%H:%M:%SZ', free_started_at, '+3 years')
WHERE free_started_at IS NOT NULL
  AND trim(free_started_at) <> ''
  AND free_until_at IS NULL;

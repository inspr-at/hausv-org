-- HAUSV-786: VAT rate per period cost type. The period switch lives in
-- legal_settings JSON and stays off until a manager turns it on.
ALTER TABLE annual_statement_period_cost_types ADD COLUMN vat_rate_percent INTEGER NOT NULL DEFAULT 10;

UPDATE annual_statement_period_cost_types
SET vat_rate_percent = 20
WHERE key IN ('heizung', 'warmwasser');

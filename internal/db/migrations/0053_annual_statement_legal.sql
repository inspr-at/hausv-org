ALTER TABLE annual_statement_periods ADD COLUMN legal_settings text NOT NULL DEFAULT '{"regime":"weg","heizkg_applies":false}';

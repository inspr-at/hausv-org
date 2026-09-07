-- HAUSV-642: a delivery is reserved (status pending) in a short transaction that
-- commits before the mail exchange. Same end state as the SQLite rebuild in
-- 0048; PostgreSQL can widen the CHECK in place.
ALTER TABLE annual_statement_deliveries DROP CONSTRAINT annual_statement_deliveries_status_check;
ALTER TABLE annual_statement_deliveries ADD CONSTRAINT annual_statement_deliveries_status_check
    CHECK (status IN ('pending', 'sent', 'failed'));

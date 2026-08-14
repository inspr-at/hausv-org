CREATE TABLE IF NOT EXISTS home_reservations (
    slug                   TEXT PRIMARY KEY,
    household_name         TEXT NOT NULL,
    owner_email            TEXT NOT NULL,
    authorization_confirmed INTEGER NOT NULL DEFAULT 0,
    status                 TEXT NOT NULL,
    created_at             TEXT NOT NULL,
    updated_at             TEXT NOT NULL,
    confirmed_at           TEXT
);

CREATE INDEX IF NOT EXISTS idx_home_reservations_owner
    ON home_reservations(owner_email);

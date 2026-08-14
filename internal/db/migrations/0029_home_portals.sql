CREATE TABLE IF NOT EXISTS home_portals (
    slug TEXT PRIMARY KEY,
    household_name TEXT NOT NULL,
    owner_email TEXT NOT NULL,
    activated_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (slug) REFERENCES home_reservations(slug) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_home_portals_owner_email ON home_portals(owner_email);

CREATE TABLE IF NOT EXISTS home_connectors (
    slug                 TEXT PRIMARY KEY REFERENCES home_reservations(slug) ON DELETE CASCADE,
    status               TEXT NOT NULL,
    credential_hash      BLOB,
    generation           INTEGER NOT NULL DEFAULT 0,
    pairing_hash         BLOB,
    pairing_expires_at   TEXT,
    connector_version    TEXT NOT NULL DEFAULT '',
    ha_version           TEXT NOT NULL DEFAULT '',
    entity_count         INTEGER NOT NULL DEFAULT 0,
    created_at           TEXT NOT NULL,
    updated_at           TEXT NOT NULL,
    paired_at            TEXT,
    last_seen_at         TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_home_connectors_credential
    ON home_connectors(credential_hash)
    WHERE credential_hash IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_home_connectors_pairing
    ON home_connectors(pairing_hash)
    WHERE pairing_hash IS NOT NULL;

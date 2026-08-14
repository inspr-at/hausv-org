CREATE TABLE home_connector_readings (
    slug          TEXT NOT NULL,
    entity_id     TEXT NOT NULL,
    state         TEXT NOT NULL,
    display_name  TEXT NOT NULL DEFAULT '',
    unit          TEXT NOT NULL DEFAULT '',
    device_class  TEXT NOT NULL DEFAULT '',
    state_class   TEXT NOT NULL DEFAULT '',
    last_updated  TEXT NOT NULL,
    received_at   TEXT NOT NULL,
    PRIMARY KEY (slug, entity_id),
    FOREIGN KEY (slug) REFERENCES home_connectors(slug) ON DELETE CASCADE
);

CREATE INDEX idx_home_connector_readings_slug_received
    ON home_connector_readings(slug, received_at DESC);

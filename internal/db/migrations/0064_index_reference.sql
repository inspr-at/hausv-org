-- Statistics Austria observations are shared reference data, never tenant data.
-- Only the maintenance importer writes them; calculations read the active rows.
CREATE TABLE index_imports (
    id TEXT PRIMARY KEY,
    url TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    data_sha256 TEXT NOT NULL,
    status_sha256 TEXT NOT NULL,
    actor TEXT NOT NULL,
    fetched_at TEXT NOT NULL,
    rows_added INTEGER NOT NULL,
    rows_revised INTEGER NOT NULL,
    rows_flagged INTEGER NOT NULL,
    changes TEXT NOT NULL,
    UNIQUE(url, sha256)
);
CREATE TABLE index_values (
    id TEXT PRIMARY KEY,
    series TEXT NOT NULL,
    period TEXT NOT NULL,
    value_millionths BIGINT NOT NULL CHECK(value_millionths > 0),
    status TEXT NOT NULL CHECK(status IN ('preliminary','final')),
    source TEXT NOT NULL,
    fetched_at TEXT NOT NULL,
    import_id TEXT NOT NULL REFERENCES index_imports(id),
    superseded_by TEXT REFERENCES index_values(id) DEFERRABLE INITIALLY DEFERRED,
    CHECK(superseded_by IS NULL OR superseded_by <> id)
);
CREATE UNIQUE INDEX index_values_active ON index_values(series,period) WHERE superseded_by IS NULL;
CREATE INDEX index_values_import ON index_values(import_id);

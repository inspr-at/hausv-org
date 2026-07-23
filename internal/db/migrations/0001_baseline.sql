-- Baseline schema. Real store tables arrive one migration per store as the JSON
-- stores are moved onto SQLite (HAUSV-166). app_meta both proves the migration
-- pipeline and gives a place to record app-level markers (e.g. a data-format
-- version) later.
CREATE TABLE IF NOT EXISTS app_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

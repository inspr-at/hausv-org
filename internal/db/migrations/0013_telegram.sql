-- Telegram bot state (HAUSV-168/170). Replaces /data/telegram.json. Not
-- tenant-scoped. Three distinct kinds get three proper tables rather than one
-- JSON blob: the getUpdates offset, chat<->user links, and pending one-time
-- link codes. Chat IDs live only here, never in env or git.
CREATE TABLE IF NOT EXISTS telegram_state (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS telegram_links (
    chat_id   INTEGER PRIMARY KEY,
    email     TEXT NOT NULL,
    name      TEXT NOT NULL DEFAULT '',
    linked_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS telegram_link_codes (
    code       TEXT PRIMARY KEY,
    email      TEXT NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    expires_at TEXT NOT NULL
);

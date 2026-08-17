-- Give every tenant an immutable identity, so the slug can become a renameable
-- URL label instead of a primary key.
--
-- Until now the slug WAS the identity: it is part of the primary key of 29
-- tables and simultaneously the URL segment, which makes renaming a house a
-- migration across all of them. This table is the one place that mapping lives.
--
-- The CHECK mirrors the PostgreSQL target schema exactly. If the two ever drift,
-- rows valid in one engine are rejected by the other at insert time.
CREATE TABLE IF NOT EXISTS tenant (
    tenant_id  TEXT PRIMARY KEY
        CHECK (length(tenant_id) = 26
               AND tenant_id GLOB '[0-7]*'
               AND tenant_id NOT GLOB '*[^0-9A-HJKMNP-TV-Z]*'),
    slug       TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Durable idempotency ledger for product-facing integration imports. Raw
-- source files and personal details are deliberately not stored here.
CREATE TABLE IF NOT EXISTS integration_imports (
    tenant_slug    TEXT NOT NULL,
    format         TEXT NOT NULL,
    file_digest    TEXT NOT NULL,
    source_version TEXT NOT NULL DEFAULT '',
    applied_at     TEXT NOT NULL,
    applied_by     TEXT NOT NULL DEFAULT '',
    assigned       INTEGER NOT NULL DEFAULT 0,
    changed        INTEGER NOT NULL DEFAULT 0,
    unclear        INTEGER NOT NULL DEFAULT 0,
    rejected       INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_slug, format, file_digest)
);

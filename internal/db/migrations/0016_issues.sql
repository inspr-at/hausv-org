-- Anliegen per tenant (HAUSV-168/170). Replaces /data/issues.json. The whole
-- issue — comments and status history included — is one JSON document keyed by
-- (tenant, id), so adding a comment is a single-row read-modify-write inside a
-- transaction. Photo files stay on disk under ISSUE_ATTACHMENT_DIR.
CREATE TABLE IF NOT EXISTS issues (
    tenant_slug TEXT NOT NULL,
    id          TEXT NOT NULL,
    data        TEXT NOT NULL,
    PRIMARY KEY (tenant_slug, id)
);

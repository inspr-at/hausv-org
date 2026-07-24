-- Dokumente (metadata only) per tenant (HAUSV-168/170). Replaces
-- /data/documents.json. The file bytes stay on disk under DOC_FILE_DIR; only
-- the metadata record moves here, keyed by (tenant, id). Versioning uses
-- series_id/version inside the JSON document.
CREATE TABLE IF NOT EXISTS documents (
    tenant_slug TEXT NOT NULL,
    id          TEXT NOT NULL,
    data        TEXT NOT NULL,
    PRIMARY KEY (tenant_slug, id)
);

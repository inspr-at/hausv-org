-- Anhänge (metadata only) per tenant (HAUSV-168/170). Replaces
-- /data/attachments.json. The file and its image variants stay on disk under
-- ATTACHMENT_FILE_DIR; only the metadata record moves here, keyed by
-- (tenant, id). Deletion is a soft delete (deleted_at inside the JSON document)
-- plus a hard delete of the files.
CREATE TABLE IF NOT EXISTS attachments (
    tenant_slug TEXT NOT NULL,
    id          TEXT NOT NULL,
    data        TEXT NOT NULL,
    PRIMARY KEY (tenant_slug, id)
);

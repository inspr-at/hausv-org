-- Per-user last-seen announcement time (HAUSV-168/170). Replaces
-- /data/announcement_reads.json. Drives the "neu" badges. Key (tenant, email).
CREATE TABLE IF NOT EXISTS announcement_reads (
    tenant_slug TEXT NOT NULL,
    email       TEXT NOT NULL,
    seen_at     TEXT NOT NULL,
    PRIMARY KEY (tenant_slug, email)
);

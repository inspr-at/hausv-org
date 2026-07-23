-- Self-service profile display/contact overlays (HAUSV-168/170).
-- Replaces /data/profile_overlays.json. One row per normalized email. These
-- overlays never carry role/permission/tenant/auth fields.
CREATE TABLE IF NOT EXISTS profile_overlays (
    email            TEXT PRIMARY KEY,
    title            TEXT NOT NULL DEFAULT '',
    first_name       TEXT NOT NULL DEFAULT '',
    last_name        TEXT NOT NULL DEFAULT '',
    phone            TEXT NOT NULL DEFAULT '',
    directory_opt_in INTEGER NOT NULL DEFAULT 0,
    updated_at       TEXT NOT NULL
);

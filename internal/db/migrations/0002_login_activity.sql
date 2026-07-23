-- Login activity: last successful login per email (HAUSV-168/170).
-- Replaces /data/activity.json. One row per normalized login email.
CREATE TABLE IF NOT EXISTS login_activity (
    email       TEXT PRIMARY KEY,
    last_login  TEXT NOT NULL,
    auth_method TEXT NOT NULL DEFAULT ''
);

-- Notification preferences per user (HAUSV-168/170). Replaces
-- /data/notification_prefs.json. The value is a small JSON document (per-event
-- email flags + unsubscribed) stored in a TEXT column; it is only ever read by
-- email, never queried per event, so a JSON column preserves exact semantics.
CREATE TABLE IF NOT EXISTS notification_prefs (
    email TEXT PRIMARY KEY,
    prefs TEXT NOT NULL
);

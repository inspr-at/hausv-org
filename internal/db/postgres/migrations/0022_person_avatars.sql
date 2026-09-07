-- Spiegel von migrations/0049_person_avatars.sql für PostgreSQL (HAUSV-675).
--
-- Ohne tenant_id und damit bewusst ohne tenant_isolation-Policy: das Profilbild
-- gehört zur Person, nicht zu einem Haus. Es teilt diese Eigenschaft mit
-- persons, house_memberships, profile_overlays und notification_prefs, die aus
-- demselben Grund keine Policy tragen. Der Zugriff wird in der Anwendung
-- entschieden (angemeldet und mindestens ein gemeinsames Haus).
CREATE TABLE IF NOT EXISTS person_avatars (
    email        text PRIMARY KEY,
    avatar_id    text NOT NULL UNIQUE,
    content_type text NOT NULL DEFAULT 'image/jpeg',
    image        bytea NOT NULL DEFAULT ''::bytea,
    byte_size    bigint NOT NULL DEFAULT 0,
    sha256       text NOT NULL DEFAULT '',
    updated_at   text NOT NULL DEFAULT ''
);

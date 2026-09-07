-- Profilbild einer Person (HAUSV-675).
--
-- Eine Zeile je normalisierter Login-E-Mail, genau wie profile_overlays: das
-- Profilbild gehört zur Person und folgt ihr in jedes Haus. Deshalb hat die
-- Tabelle KEINE tenant_id und keine RLS-Policy — dieselbe bewusste Entscheidung
-- wie bei persons, house_memberships und profile_overlays (HAUSV-169).
--
-- Gespeichert wird ausschliesslich das serverseitig neu kodierte Bild: maximal
-- 512x512, JPEG. Das Neukodieren entfernt jede Metadatenspur (EXIF, GPS) aus der
-- hochgeladenen Datei.
--
-- avatar_id ist der öffentliche, nicht erratbare Teil der Bild-URL (ULID). Er
-- wird bei jedem neuen Bild neu vergeben, damit eine ausgelieferte URL immer
-- genau einen Bildinhalt bezeichnet und Zwischenspeicher nicht von Hand
-- entwertet werden müssen.
CREATE TABLE IF NOT EXISTS person_avatars (
    email        TEXT PRIMARY KEY,
    avatar_id    TEXT NOT NULL UNIQUE,
    content_type TEXT NOT NULL DEFAULT 'image/jpeg',
    image        BLOB NOT NULL,
    byte_size    INTEGER NOT NULL DEFAULT 0,
    sha256       TEXT NOT NULL DEFAULT '',
    updated_at   TEXT NOT NULL DEFAULT ''
);

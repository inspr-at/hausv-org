-- Person <-> Haus als N:N (HAUSV-169, absorbiert HAUSV-135).
--
-- Bindende Domänenentscheidung vom 18.07.2026: Person und Haus sind eigene
-- Aggregate. Globale Identität (E-Mail, Titel, Name, Anmeldewege, Sperre) gehört
-- zur Person; Rolle, Rechte und Mitgliedschaftsstatus gehören ausschliesslich
-- an die Verbindung Person<->Haus.
--
-- Damit entfernt eine Hausverwaltung beim "Löschen" nur ihre eigene
-- Mitgliedschaft: die Person und alle anderen Mitgliedschaften bleiben bestehen.
CREATE TABLE IF NOT EXISTS persons (
    id           TEXT PRIMARY KEY,
    -- normalisierte Login-E-Mail; global eindeutig
    email        TEXT NOT NULL UNIQUE,
    title        TEXT NOT NULL DEFAULT '',
    first_name   TEXT NOT NULL DEFAULT '',
    last_name    TEXT NOT NULL DEFAULT '',
    -- JSON-Array; leer = Standard-Anmeldewege
    auth_methods TEXT NOT NULL DEFAULT '',
    -- globale Anmeldesperre (nicht: Status in einem einzelnen Haus)
    deactivated  INTEGER NOT NULL DEFAULT 0,
    adopted      INTEGER NOT NULL DEFAULT 0,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS house_memberships (
    person_id   TEXT NOT NULL,
    tenant_slug TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT '',
    -- JSON-Array
    permissions TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    PRIMARY KEY (person_id, tenant_slug),
    FOREIGN KEY (person_id) REFERENCES persons(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_house_memberships_tenant ON house_memberships(tenant_slug);

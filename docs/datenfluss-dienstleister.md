# Datenfluss Dienstleister-Zugriff (JHW22)

**Zweck dieses Dokuments:** eine vollständige, sachliche Beschreibung, welche Daten
beim Dienstleister-Zugriff entstehen, wozu sie verarbeitet werden, wer sie
empfängt und wo sie gespeichert werden. Grundlage für die externe Prüfung nach
HAUSV-86.

**Dieses Dokument trifft ausdrücklich keine rechtliche Bewertung.** Es enthält
keine Aussage zu Verantwortlichkeit, Rechtsgrundlage, Erforderlichkeit eines
Auftragsverarbeitungsvertrags oder Zulässigkeit. Diese Festlegungen sind
Gegenstand der externen Prüfung. Offene Punkte sind am Ende gesammelt.

Stand: 24.07.2026, Anwendungsversion 0.17.7.

---

## 1. Aktueller Betriebszustand

Der Dienstleister-Zugriff ist **technisch implementiert, aber standardmäßig
geschlossen**. `SERVICE_PROVIDER_ACCESS_ENABLED` ist `false`; in dieser
Einstellung werden Zuordnungen, Dienstleister-Kontakte und -Einladungen,
Anmeldung per Magic-Link und OIDC, bestehende Sitzungen, Benachrichtigungen und
Parkplatzrechte **abgewiesen, bevor** Daten-, Mail-, Audit- oder
Sitzungsänderungen entstehen. Es wird kein unzugänglicher Entwurf gespeichert.

**In der Produktion bei JHW22 existiert derzeit kein Dienstleister-Zugang.** Die
folgende Beschreibung gilt für den Zustand nach einer etwaigen Freigabe.

## 2. Speicherorte

Alle Anwendungsdaten liegen auf dem Host `csb1` im Verzeichnis
`/var/lib/csb1-docker/hausv-org` (im Container `/data`), das ausschließlich
diesem Dienst zugeordnet ist.

| Ort | Inhalt |
|---|---|
| `hausv.db` (SQLite) | Personen, Haus-Mitgliedschaften, Anliegen inkl. Kommentaren und Statusverlauf, Anhang-**Metadaten**, Dokument-**Metadaten**, Anmeldeverlauf, Profilangaben, weitere Fachdaten |
| `attachments/<haus>/` | Anhang-**Dateien** und daraus erzeugte Vorschau-/Miniaturbilder |
| `documents/<haus>/` | Dokument-**Dateien** |
| `audit.jsonl` (+ Archive) | Protokoll sicherheitsrelevanter Vorgänge |
| `parking.json` | Parkplatz-/Lade-Messreihen (kein Dienstleister-Bezug) |

Dateien liegen bewusst im Dateisystem, nicht in der Datenbank; die Datenbank hält
nur die zugehörigen Metadaten.

## 3. Datenflüsse je Vorgang

Angegeben sind jeweils Zweck, verarbeitete Datenfelder, Empfänger und
Speicherort.

### 3.1 Einladung eines Dienstleisters

- **Zweck:** einer externen Firma Zugang zu einem konkreten Anliegen ermöglichen.
- **Daten:** E-Mail-Adresse, optional Titel/Vor-/Nachname, Rolle
  („Dienstleister“), Haus-Zugehörigkeit, Einladungsstatus.
- **Erhoben von:** der Hausverwaltung (manuelle Eingabe im Portal).
- **Empfänger:** der eingeladene Dienstleister (per E-Mail, siehe 4.1).
- **Speicherort:** `hausv.db`, Tabellen `persons` (globale Identität: E-Mail,
  Titel, Name) und `house_memberships` (Rolle, Rechte, Status je Haus).
- **Aufbewahrung:** bis zum Entzug der Haus-Zugehörigkeit. Der Entzug entfernt
  die Mitgliedschaft; die Person bleibt nur bestehen, solange sie noch einem
  anderen Haus zugeordnet ist.

### 3.2 Anmeldung

- **Zweck:** Authentifizierung.
- **Wege:** Magic-Link per E-Mail und/oder OIDC (Single Sign-on); je Zugang
  einstellbar.
- **Daten:** E-Mail-Adresse; bei OIDC die vom Anbieter zurückgelieferte Identität;
  Zeitpunkt und Ergebnis des Anmeldeversuchs.
- **Empfänger:** E-Mail-Versanddienst bzw. OIDC-Anbieter (siehe 4).
- **Speicherort:** Anmeldeverlauf in `hausv.db`; Sitzungen als signiertes Cookie
  beim Nutzer, nicht serverseitig gespeichert.

### 3.3 Anliegen (Sichtbarkeit)

- **Zweck:** Bearbeitung des beauftragten Mangels.
- **Sichtbarkeitsregel (technisch durchgesetzt):** eine Person mit
  Dienstleister-Rolle sieht ein Anliegen **ausschließlich dann**, wenn es
  **offen** und **ihr zugewiesen** ist. Andere Anliegen desselben Hauses,
  Gemeinschaftsanliegen und geschlossene Anliegen sind nicht sichtbar. Der Entzug
  der Zuweisung wirkt sofort.
- **Daten im Zugriff:** Titel, Beschreibung, Kategorie, Ort (Einheit oder
  Gemeinschaftsbereich), Status, Priorität, Zeitstempel, Name/E-Mail der
  meldenden Person.
- **Speicherort:** `hausv.db`, Tabelle `issues` (vollständiger Datensatz je
  Anliegen).

### 3.4 Kommentare

- **Zweck:** Abstimmung zum Anliegen zwischen Hausverwaltung, meldender Person
  und Dienstleister.
- **Daten:** Text (max. 3000 Zeichen), Name und E-Mail der verfassenden Person,
  Zeitpunkt.
- **Empfänger:** alle Personen, die das Anliegen sehen dürfen (siehe 3.3), sowie
  E-Mail-Benachrichtigungen an Abonnenten.
- **Speicherort:** innerhalb des Anliegen-Datensatzes in `hausv.db`.
- **Hinweis:** Kommentare sind Freitext. Welche personenbezogenen Daten darin
  entstehen, bestimmen die schreibenden Personen; eine technische Beschränkung
  besteht nicht.

### 3.5 Fotos und Dateianhänge

- **Zweck:** Dokumentation des Mangels und des Arbeitsfortschritts.
- **Daten:** Bild-/Dateiinhalt, Originaldateiname, Größe, Inhaltstyp,
  hochladende Person, Zeitpunkt; automatisch erzeugte Vorschau- und
  Miniaturbilder.
- **Empfänger:** dieselben Personen wie beim zugehörigen Anliegen (3.3). Die
  Auslieferungsrouten prüfen die Berechtigung; Negativtests belegen, dass ein
  Dienstleister Dateien fremder Anliegen nicht abrufen kann.
- **Speicherort:** Dateien unter `attachments/<haus>/`, Metadaten in `hausv.db`.
- **Löschung:** beim Löschen werden die Dateien sofort entfernt; der Metadaten-
  Eintrag bleibt als Markierung bestehen und wird nach einem Jahr endgültig
  gelöscht.
- **Hinweis:** Fotos können unbeabsichtigt weitere personenbezogene Daten
  enthalten (z. B. abgebildete Personen, Einrichtung, Dokumente im Bild).
  Eingebettete Aufnahme-Metadaten werden derzeit nicht entfernt.

### 3.6 Kostenvoranschläge und Terminvorschläge

- **Zweck:** Angebot und Terminabstimmung durch den Dienstleister.
- **Daten:** Betrag, Freitext-Notiz, vorgeschlagener Zeitraum, Person und
  Zeitpunkt der Eingabe.
- **Empfänger:** wie 3.3.
- **Speicherort:** innerhalb des Anliegen-Datensatzes in `hausv.db`; zugehörige
  Dateien als Anhänge (3.5).

### 3.7 Auditdaten

- **Zweck:** Nachvollziehbarkeit sicherheitsrelevanter Vorgänge (Zugänge,
  Rechteänderungen, Zuweisungen, Löschungen).
- **Daten:** Zeitpunkt, handelnde Person (E-Mail) und Rolle, Aktion, betroffenes
  Objekt, Kurzbeschreibung, ausgewählte Vorher-/Nachher-Werte (z. B. Rolle,
  Rechte, Anmeldewege).
- **Empfänger:** Personen mit Einsichtsrecht in das Protokoll innerhalb der
  Verwaltung.
- **Speicherort:** `audit.jsonl`; oberhalb von 20 000 Einträgen wird die
  laufende Datei in ein Archiv daneben verschoben und mit den jüngsten 5 000
  Einträgen neu geschrieben. **Es wird nichts gelöscht**; die Archive wachsen und
  werden derzeit nicht automatisch entfernt.

## 4. Externe Empfänger

| Empfänger | Wofür | Übermittelte Daten |
|---|---|---|
| **Resend** (`smtp.resend.com`, Absender `noreply@notify.hausv.org`) | Versand von Einladungen, Anmeldelinks und Benachrichtigungen | Empfänger-E-Mail-Adresse, Betreff und Inhalt der Nachricht (kann Anliegen-Titel enthalten) |
| **Zitadel** (`https://auth.inspr.at`) | Anmeldung per Single Sign-on, sofern für den Zugang aktiviert | Identitätsdaten im Rahmen des OIDC-Ablaufs |
| **Home Assistant** (`100.64.0.7`, internes Netz) | Parkplatz-/Ladesteuerung | **kein Dienstleister-Bezug**; hier werden keine Dienstleisterdaten übermittelt |

Weitere Übermittlungen an Dritte finden nicht statt. Die Anwendung bindet keine
externen Skripte, Schriftarten oder Analysedienste ein.

## 5. Technische Datenminimierung (Ist-Zustand)

- Dienstleister sehen ausschließlich offene, ihnen zugewiesene Anliegen (3.3).
- Der Entzug einer Zuweisung wirkt sofort, auch für bestehende Sitzungen.
- Datei- und Foto-Routen prüfen die Berechtigung je Abruf.
- Rolle und Rechte gelten je Haus; eine Änderung in einem Haus wirkt nicht in
  einem anderen.
- Das Verwaltungs-Formular gibt globale Identitätsfelder (E-Mail, Titel, Name)
  nur der Plattform-Administration zur Änderung frei.
- Genannte Punkte sind durch Negativtests abgedeckt.

## 6. Offene Punkte für die externe Prüfung

Bewusst **nicht** in diesem Dokument entschieden:

1. Verantwortlichkeit je Verarbeitung (Eigentümergemeinschaft, Verwaltung,
   Betreiber der Anwendung, Dienstleister).
2. Rechtsgrundlage je Verarbeitung.
3. Erforderlichkeit eines Auftragsverarbeitungsvertrags oder eines anderen
   Vertrags-/Informationsinstruments — insbesondere gegenüber Resend und Zitadel
   sowie gegenüber dem Dienstleister.
4. Inhalt und Ort der Informationspflichten gegenüber Bewohnern und
   Dienstleistern.
5. Aufbewahrungs- und Löschfristen für Anliegen, Kommentare, Fotos und
   Auditdaten — heute technisch unbegrenzt, mit Ausnahme der Anhang-Markierungen
   (ein Jahr) und der Parkplatzmesswerte (rund 13 Monate).
6. Umgang mit personenbezogenen Daten in Freitext und in Bildinhalten
   (3.4, 3.5), einschließlich der Frage, ob Aufnahme-Metadaten vor der Speicherung
   entfernt werden sollen.
7. Umgang mit den Audit-Archiven, die derzeit unbegrenzt aufbewahrt werden.

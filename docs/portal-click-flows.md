# Klickwege des Portals

Stand: 1. August 2026

Diese Landkarte beschreibt die konkreten Haupt- und Unterwege des Portals:
Einstieg, Routen, Rollen, Abzweigungen und sichtbare Abschlusssignale. Sie ist
eine versionierte, code-nahe Spezifikation. Produktprinzipien, Begründungen und
Lieferstatus bleiben in PPM Knowledge beziehungsweise im PPM-Backlog.

## Gemeinsamer Vertrag

Jeder Hauptweg folgt denselben Regeln:

- Die erste Ansicht zeigt genau die jetzt relevante Hauptaktion. Seltener
  benötigte Aktionen sind als ruhige Links, Ghost Buttons oder Details
  nachgeordnet.
- Zurück, Abbrechen, Browser-Zurück und Browser-Vorwärts verlieren keine
  bereits erfassten Angaben.
- Ein Schreibvorgang endet auf einer Ansicht, die Ergebnis, aktuellen Stand und
  nächsten Schritt eindeutig bestätigt.
- Rollenbeschränkungen gelten serverseitig. Ein ausgeblendetes Bedienelement ist
  kein Berechtigungsnachweis.
- Primäre mobile Aktionen besitzen eine mindestens 44 Pixel große Klickfläche;
  alle bedienbaren Ziele erreichen mindestens 40 Pixel oder besitzen eine
  nachweislich vergrößerte Trefferfläche. Es gibt keinen horizontalen Überlauf
  oder verdeckte Aktionen.
- Tastaturfokus, reduzierte Bewegung und der sinnvolle Weg ohne JavaScript
  werden dort geprüft, wo JavaScript den Ablauf verdichtet.

## Hauptwege

| Nr. | Hauptweg | Rollen | Klickfolge und Unterwege | Sichtbarer Abschluss |
| --- | --- | --- | --- | --- |
| 1 | Öffentlicher Einstieg und Anmeldung | Öffentlich, danach alle Rollen | Marketing-Host: `/` → HAUSV Home „starten“ → `/start` → Pfad und Eigentümer-E-Mail vormerken → Einmal-Link `/start/verify` → geschützter Einrichtungsbereich `/start/connector` → „Portal jetzt aktivieren“ → direkt `/<slug>/app`. Optional und unabhängig: Einmal-Code erzeugen → Linux-Connector laden und lokal koppeln → im Energie-Onboarding Messwerte auswählen. Unterwege: aktives Portal erneut öffnen, Bestätigung erneut senden, Code erneuern, Zugang rotieren oder widerrufen, ungültiger/abgelaufener/verbrauchter Link, Datenschutz, Abmelden und Browser-Zurück. Tenant-Pfad: `/<slug>` → E-Mail-Formular `POST /auth/request` → Anmeldelink `GET /auth/verify` → `/<slug>/app`. Alternative: OIDC-Start und Callback. | Portalpfad und Eigentümerzugriff werden gemeinsam aktiv; die Eigentümerin oder der Eigentümer ist direkt angemeldet. Der lokale Connector liefert ausschließlich ausgewählte Energiewerte in den persönlichen Bereich. Home-Assistant-Adresse und Token bleiben lokal; Gerätesteuerung ist nicht aktiv. |
| 2 | Hausüberblick | Mieter, Bewohner, Eigentümer, Beirat, Verwalter, Admin | `/app` → genau eine heutige Aufgabe oder ruhiger Alles-erledigt-Zustand → passende Fachroute. Unterwege: nächster Termin, aktueller Aushang, offene Anliegen, Zuhause/Energie und die leise Aktion „Anliegen melden“. | Die gewählte Fachansicht öffnet direkt am relevanten Inhalt oder Arbeitsschritt. |
| 3 | Aushang | Lesen: Rollen mit Bewohnerbereich; Schreiben: Verwalter/Admin | `/app/announcements` → Suche/Kategorie → Beitrag lesen. Verwaltung: „Aushang erstellen“ → Dialog → speichern; anschließend bearbeiten oder löschen. | Neuer oder geänderter Beitrag steht in der richtigen Gruppe; Filter und lange Titel bleiben bedienbar. |
| 4 | Termine | Lesen: Rollen mit Bewohnerbereich; Schreiben: Verwalter/Admin | `/app/events` → zeitliche Gruppe → Termin lesen oder Kalender abonnieren. Verwaltung: Termin erstellen → Datum/Ort → speichern; anschließend bearbeiten oder löschen. | Der Termin erscheint chronologisch und mit eindeutigem Datum/Ort; der Kalender-Link liefert den persönlichen Feed. |
| 5 | Kontakte | Rollen mit Bewohnerbereich; Pflege: Verwalter/Admin | `/app/kontakte` → dringenden Hauskontakt direkt anrufen/anschreiben oder Adressbuch durchsuchen. Unterwege: freiwilliges Bewohnerverzeichnis und eigener Opt-in unter `/app/settings/profile`. Verwaltung: Kontakt anlegen → optionale Qualifikation/Fähigkeiten → speichern; bearbeiten oder deaktivieren. | Der Kontakt ist erreichbar beziehungsweise im passenden Verzeichnis sichtbar; deaktivierte Einträge verschwinden für Bewohner. |
| 6 | Dokumente | Lesen: Rollen mit Bewohnerbereich nach Sichtbarkeit; Pflege: Verwalter/Admin | `/app/dokumente` → Kategorie/Dokument → Vorschau oder Download. Verwaltung: hochladen → Sichtbarkeit/Kategorie → speichern; ersetzen. Unterweg: `/app/dokumente/rechnungen/import` → Vorschau → Ablage. | Vorschau/Download liefert nur berechtigte Inhalte; ein Upload erscheint in der gewählten Sichtbarkeit. |
| 7 | Anliegen melden und verfolgen | Meldung: Mieter/Bewohner/Eigentümer; Aufsicht: Beirat; Triage: Verwalter/Admin; Ausführung: zugewiesener Dienstleister | `/app` oder `/app/anliegen` → „Anliegen melden“ → Schritt 1 Beschreibung/Art/Ort/Dateien → Schritt 2 prüfen → senden → `/app/anliegen/{id}?created=1`. Unterwege im Detail: Rückfrage beantworten, Lösung bestätigen oder offen halten, Anhänge ansehen/entfernen. Verwaltung: `/app/anliegen/board` → Filter → `/app/anliegen/board/{id}` → Priorität/Zuständigkeit/Status. Dienstleister: annehmen → Termin → Arbeit → Erledigung/Kostenschätzung. | Direkte Bestätigung, aktueller Stand und Originalmeldung sind sichtbar; spätere Neuigkeiten erscheinen erst, wenn es welche gibt. |
| 8 | Abstimmungen | Lesen: Rollen mit Bewohnerbereich; Stimme: fachlich Berechtigte; Steuerung: Verwalter/Admin | `/app/abstimmungen` → offene Abstimmung → Auswahl → Stimme senden → Ergebnis verstehen. Verwaltung: öffnen, schließen und `/app/abstimmungen/{id}/protokoll` abrufen. | Stimme oder Schließung wird bestätigt; Ergebnis und Protokoll bleiben nachvollziehbar. |
| 9 | Übergabe | Verwaltung/Admin; Bestätigung: eingeladene Parteien | `/app/uebergaben` → „Übergabe anlegen“ → Termin → Räume/Zustand → Parteien/Zähler/Schlüssel/Dateien → speichern → persönlicher Link `/handover/{token}` → prüfen/bestätigen → zweite Bestätigung → Ablage. Unterwege: tokengebundene Anhänge, bestätigten Link schreibgeschützt erneut öffnen, PDF und Dateivorschau. | Beide Seiten sehen den bestätigten Zustand; der Zwischenstand ist unveränderlich und das abgelegte Protokoll bleibt abrufbar. |
| 10 | Zuhause einrichten | Eigentümer, Verwalter, Admin; freigegebene technische Hilfe im erlaubten Umfang | Self-Service: `/start` → Reservierung → zugestellten E-Mail-Link öffnen → Portal aktivieren → optional Energieverbindung vorbereiten. Im Portal: `/app/zuhause/onboarding` → Verständnis → Zuhause-Art/Name → Verbraucher → Home Assistant → Messwerte bestätigen → Abschluss. Unterwege: Unterbrechung/Fortsetzen, keine Verbindung, unpassende Entitäten, Wohnung/Einfamilienhaus/Hausgemeinschaft. | Das private Portal ist dauerhaft erreichbar; die optionale Verbindung bleibt nur lesend. Danach sind Zuhause und Messquellen verständlich zugeordnet. |
| 11 | Energie verstehen und sicher erproben | Steuerung: Eigentümer, Verwalter, Admin; lesend weitere Bewohnerrollen oder freigegebene Hilfe | `/app/energie` → permanenter Sicherheitsmodus → Live-Zustand → genau nächster Schritt → Tarifdetails → Verlauf/Fahrplan. Unterwege: Testlauf bewusst öffnen/bestätigen und sofort zu „Nur beobachten“ zurückkehren; Messwerte zuordnen; Smart-Meter-Referenz; Verbraucher, Wartung, Hausaufgabe, technische Vertrauensperson und Tarifstand. | Wirkung und Datenbasis sind sichtbar; Testlauf behauptet keine Gerätewirkung; Rückkehr beendet ihn sofort. |
| 12 | Energie- und Profildaten | Eigentümer, Verwalter, Admin | `/app/settings/home` → Anzeigename/offizielle Einheit pflegen. `/app/settings/energy-data` → Datenumfang verstehen → Export; getrennte Löschwege für Messhistorie oder Profil. | Speicherung, Export oder Löschung wird konkret bestätigt; Haus- und Personengrenzen bleiben erhalten. |
| 13 | Parkplatz und Laden | Portal: ausdrücklich berechtigte Nutzer oder Admin; Monat/Zugang/Erinnerungen: Verwalter/Admin; Technik: Admin | `/app/parking` → aktueller Monat/Status → Monatsdetail → Export. Unterwege: Laden ein/aus/automatisch. Verwaltung: `/app/settings/parking-access`, Monatswerte und Erinnerungen. Admin: `/app/parking/settings`, Laderegeln und Telegram-Verknüpfung. | Monat und Ladezustand sind eindeutig; jede Steueraktion zeigt den tatsächlich erreichten Zustand. |
| 14 | Persönliche Einstellungen | Rollen mit Bewohnerbereich | `/app/settings` → `/app/settings/profile` für Namen/Verzeichnissichtbarkeit oder `/app/settings/notifications` für Kanäle/Ereignisse. Rollenabhängig zusätzlich Zuhause, Energiedaten und Parkplatz-Zugang. | Gespeicherte Werte und Freigaben werden unmittelbar bestätigt. |
| 15 | Gebäude, Einheiten, Zahlungen und Personen | Verwalter/Admin | `/app/settings/building` → Hausdaten/Einheiten → Einheit anlegen oder zuordnen → Zahlungsstatus. Unterwege: Bankdatei `/app/settings/payments/import` → Vorschau → anwenden; `/app/settings/data-export` → Vorschau → Download; `/app/settings/users` → Einladung → Rechte bearbeiten oder Zugang entfernen. | Vorschau benennt den Umfang vor der Änderung; Ergebnis, Rollen und Hauszuordnung sind danach sichtbar. |
| 16 | Verlauf und Nachweis | Eigener Verlauf: Rollen mit Bewohnerbereich; Hausverlauf: Verwalter/Admin | `/app/audit` → Zeit/Vorgang/Person/Objekt prüfen und bei vorhandenem Umfang filtern. Fachprotokolle bleiben bei Abstimmung, Übergabe und Export verlinkt. | Ein Vorgang ist ohne interne Rohdaten verständlich zuordenbar; der Leerzustand erklärt, was künftig erscheint. |
| 17 | Mitarbeiter der Verwaltung | Admin der gewählten Verwaltung | Eine Person zuerst unter `/app/settings/users` in einer Liegenschaft einladen; danach `/app/verwaltung/einstellungen` → Mitarbeiter aufnehmen, Rolle ändern oder entfernen. | Hausrechte, Mitarbeiterdatensatz, Sicherung der vorherigen Rolle samt Einzelrechten und SQL-Nachweis werden gemeinsam gespeichert. Spätere manuelle Hausrechte bleiben beim Entfernen erhalten, auch bei unveränderter Rolle. Alte Datensätze mit bloßer Rollensicherung verlangen vor einer Änderung einen Abgleich. |

## Zielvertrag je Hauptweg

Die folgende Tabelle ist der gewünschte Abnahmevertrag. Sie behauptet nicht,
dass jeder Zustand jedes Unterwegs bereits automatisiert ist. Welche Teile der
Klickwege Playwright tatsächlich ausführt, steht getrennt im Abschnitt
„Automatisierter Beleg“ und im technischen QA-Dokument.

| Zustand | Erwartung |
| --- | --- |
| Leer | Ein kurzer Satz erklärt die Fläche; eine berechtigte Hauptaktion ist sichtbar, sonst keine erfundene Aktion. |
| Befüllt | Wichtigstes Element und nächster Schritt sind ohne Aufklappen sichtbar. |
| Langer Inhalt | Titel, Dateinamen, Adressen und Hinweise bleiben innerhalb ihrer Komponente; Aktionen werden nicht verdrängt. |
| Validierungsfehler | Fehler steht auf Deutsch direkt am Feld, Fokus springt sinnvoll, Eingaben bleiben erhalten. |
| Erfolg | Ergebnis, aktueller Zustand und nächster Schritt stehen zusammen; ein Listenumweg ist nicht nötig. |
| Nicht berechtigt | Direkter Request scheitert serverseitig und führt verständlich zurück. |
| Unterbrochen | Browser-Zurück/Vorwärts, Neuladen oder erneute Anmeldung erzeugen weder Doppelaktion noch stillen Datenverlust. |
| Langsam/offline | Aktion sperrt sich gegen Doppelklick, zeigt Beschäftigung und endet mit einer hilfreichen Rückmeldung. |

## Automatisierter Beleg

Die Orchestrierung liegt in `scripts/qa-main-flows.sh`. Sie startet eine
isolierte Portalinstanz und führt mehrere spezialisierte Playwright-Harnesses
nacheinander aus. Die Zuordnung ist bewusst präzise: Ein grüner Screenshot
allein gilt nicht als Nachweis eines Klickwegs.

| Wege | Browsernachweis | Tatsächlich ausgeführte Interaktionen |
| --- | --- | --- |
| 1 | `qa-public-auth.mjs` | Tenant-Anmeldung per lokalem Link, vorbereiteter Link, ungültiger Link, Datenschutz, Karte, Browser-Zurück/Vorwärts, No-JS, reduzierte Bewegung und Abmelden. OIDC bleibt zusätzlich serverseitig getestet. |
| 2, 7, 10, 11 | `qa-main-flows.mjs`, `qa-home-setup.mjs` | Hausüberblick, Anliegen in zwei Schritten samt Verlauf/No-JS, Energie-Onboarding mit Unterbrechung und mehrere Zuhause-Archetypen, Cockpit, Testlauf und Sofort-Rückkehr. Der Self-Service-Lauf verarbeitet zusätzlich die echte lokale E-Mail, aktiviert Portal und Connector und prüft beides samt Mandantentrennung nach Neustart. |
| 3, 4, 5, 8, 16 | `qa-main-flows.mjs` | Suchen/Lesen, Kalender, Kontaktaktion, Abstimmen/Ändern/Schließen/Ergebnis/Protokoll und Verlaufsfilter; Verwaltungs-Fixtures werden über die echten Formulare angelegt. |
| 6 | `qa-document-flow.mjs` | Upload, Sichtbarkeit, Vorschau, Download, Ersetzen/Version, Bildvorschau und E-Rechnung von XML-Prüfung bis geschützter Ablage; direkte Rollenverbote werden gesendet. |
| 9 | `qa-handover-flow.mjs` | Anlegen, tokengebundene öffentliche Prüfung/Anhänge, erste und zweite Bestätigung, idempotentes erneutes Senden, gesperrter Zwischenstand, Ablage, PDF-Download und Lightbox. |
| 12–15 | `qa-settings-parking.mjs` | Profil und Benachrichtigungen speichern, Einheit/Zahlungsstatus, CAMT-Vorschau, Datenexport, Einladung anlegen/bearbeiten/entfernen, Parkplatz-Zugriff erteilen/entziehen sowie leere und befüllte Parkplatzmonate. |

Go-Tests ergänzen diese Browserwege um Mieter, Beirat, Dienstleister,
hausfremde IDs, unzulässige Direktrequests und seltene Fehlerzustände. Der
separate `capture.mjs`-Pfad plant 4 × 19 Rollen-/Routenaufnahmen plus drei
Cockpit-Aufnahmen bei 1440 × 900; er ist eine visuelle Stichprobe und kein Ersatz
für die oben genannten Klickwege. Messbereiche, blinde Stellen und lokale
Aufrufe stehen unter [Orakel und CI-Gates](playwright-main-flow-qa.md#orakel-und-ci-gates).

## Rollen- und Fixture-Matrix

Die deterministischen QA-Fixtures enthalten vier Häuser und sieben
ausschließlich erfundene statische Konten. Der Rollenlauf bedient davon vier
Hauptpersonen aktiv:

- `resident@example.com`: Bewohner
- `owner@example.com`: Eigentümer
- `verwalter@example.com`: Verwalter
- `admin@example.com`: Admin
- zusätzliche Eigentümer-Fixtures für Haus A-, Haus B- und
  Energie-Cockpit-Zuhause; `cockpit-owner@example.com` hält den fertigen
  Cockpit-Zustand vor, ist aber keine eigene Playwright-Hauptperson

Home Assistant, Kartenkacheln, Dokumente, Anhänge und E-Mail-Links stammen aus
lokalen Fixtures. Drei getrennte Home-Assistant-Datensätze modellieren DEMO,
Haus A und Haus B; das Cockpit-Haus verwendet bewusst den
umfangreichen DEMO-Datensatz. Mieter, Beirat und Dienstleister sind in
serverseitigen Rollenprüfungen abgedeckt, derzeit aber keine eigenen
Playwright-Hauptpersonen. Reale Home-Assistant-Daten sind für diese Regression
nicht erforderlich und werden nicht geschrieben.

## Browsermatrix und Ausführung

Die private Grundmatrix öffnet die Rollen/Routen auf 1440 × 900 und 390 × 844.
Öffentlicher Einstieg, Anmeldung und Karte werden zusätzlich bei 320, 390,
768, 1024 und 1440 Pixeln geprüft; Übergabe und Dokumente verwenden dieselben
fünf Breiten, Einstellungen/Parkplatz 320, 390, 1024 und 1440. Die Breiten 360,
430, 900/901, 1119/1120/1121 und 1280 sowie kurze Fensterhöhen gehören zu
gezielten Geometrieprüfungen kritischer Shell-, Dialog-, Energie- und
Fachflächen; sie laufen nicht pauschal für jede Route. Der vollständige Lauf:

```fish
scripts/qa-main-flows.sh
```

Das vollständige CI-Gate „Full browser flow suite“ wird lokal so reproduziert:

```sh
direnv exec . npm --prefix scripts/snapshot exec -- playwright install chromium
CI=true HV_QA_HEADLESS=true HV_QA_ARTIFACT_DIR=/tmp/hausv-oracles/full \
  direnv exec . bash scripts/qa-main-flows.sh
```

`CI=true` verhindert dabei den automatischen lokalen Fallback auf ein
installiertes Chrome. Ohne explizites `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH`
wird das zur festgeschriebenen Playwright-Version gehörende Chromium verwendet.
Chromium läuft automatisiert ausdrücklich headless. Screenshots
unterstützen die visuelle Prüfung; DOM-Reihenfolge, Geometrie, Fokus,
Rollenverbote, Touch-Ziele, Überlauf und strukturierte Logs werden zusätzlich
maschinell geprüft.

# Playwright-QA der Hauptwege

Die konkrete Landkarte aller Haupt- und Unterwege steht in
[`portal-click-flows.md`](portal-click-flows.md). Dieses Dokument beschreibt
die Ausführung und die technischen Gates des Browserlaufs.

Der lokale QA-Lauf startet das aktuelle Arbeitsverzeichnis mit ausschließlich
erfundenen Daten, mehreren Rollen und strikt getrennten Hausprofilen. E-Mails
gehen ausschließlich an einen lokalen SMTP-Empfänger im temporären
Testverzeichnis. Der Lauf greift nicht auf Produktivdaten zu und lädt auch keine
Kartenkacheln von einem öffentlichen Dienst.

Der vollständige lokale Lauf ist zugleich das Browser-Regressionsgate:

```fish
scripts/qa-main-flows.sh
```

Der Lauf:

- meldet Bewohner, Eigentümer, Verwalter und Admin über den lokalen
  Entwicklungszugang an;
- legt einen Aushang, einen Termin, einen Kontakt, ein Dokument und ein Anliegen
  an;
- bedient die Bewohnerwege für Suche, Kalender, Telefon, Download, Verlauf und
  Abstimmung einschließlich Änderung, Schließung, Ergebnis und Protokoll;
- prüft das private Haus-Cockpit für Bewohner und Eigentümer, einschließlich
  dauerhaft sichtbarem Beobachtungsmodus, bewusstem Shadow-Testlauf,
  Sofort-Rückkehr und idempotentem Smart-Meter-Import;
- spielt die Haus A-Konstellation mit PV, E-Auto, Warmwasser-Wärmepumpe und
  bewusst fehlendem Speicher sowie die Haus B-Konstellation mit PV,
  Speicher, E-Auto und eigener technischer Vertrauensperson durch;
- wechselt im Onboarding zwischen Wohnung, Einfamilienhaus und
  Hausgemeinschaft und prüft, dass Umfang, unveränderte Rechte und der
  Beobachtungsmodus direkt verständlich bleiben;
- prüft an drei simulierten, ausschließlich lesenden Home-Assistant-Instanzen,
  dass Messwerte hausbezogen bleiben, höchstens fünf ruhige Vorschläge
  erscheinen und Messlücken ohne Entity-IDs erklärt werden;
- bedient den festen Kartenausschnitt aus einer lokalen gültigen PNG-Fixture und
  prüft sowohl Dekodierung als auch Content-Type/PNG-Signatur;
- prüft den öffentlichen Tenant-Einstieg mit Anmeldung, vorbereiteten und
  ungültigen Links, Datenschutz, Browser-Historie, No-JS und Abmelden;
- spielt eine Übergabe vollständig von der Anlage über tokengebundene Anhänge
  und zwei Bestätigungen bis zur unveränderlichen Ablage, PDF und Lightbox;
- spielt Dokument-Upload, Sichtbarkeit, Vorschau, Download, Ersetzen/Version
  und E-Rechnung bis zum unveränderten, geschützten XML-Original durch;
- speichert Profil und Benachrichtigungen, legt eine Einheit und eine
  Testeinladung an, prüft Export/CAMT-Vorschau und erteilt beziehungsweise
  entzieht einen hausbezogenen Parkplatz-Zugriff;
- prüft den Parkplatz zuerst ehrlich leer, startet danach dieselbe isolierte
  Anwendung mit einer rollierenden Zwei-Monats-Fixture neu und prüft Monats-,
  Zahlungs- und Exportzustände;
- startet zwischen den fachlichen Harnesses nur den isolierten Prozess neu,
  behält dabei die gemeinsame Fake-Datenbasis und setzt so ausschließlich
  kurzlebige Anmeldelimits zurück;
- reserviert ein HAUSV Home im Browser, verarbeitet die tatsächlich über SMTP
  zugestellte Bestätigungs-E-Mail, aktiviert das private Portal, koppelt einen
  simulierten lokalen Nur-Lese-Helfer und prüft Portal, Verbindung sowie
  Mandantentrennung erneut nach einem Prozessneustart;
- öffnet die private Grundmatrix aus vier Rollen und sieben Routen auf
  1440 × 900 und 390 × 844;
- prüft öffentlichen Einstieg, Übergabe und Dokument-Lifecycle bei 320, 390,
  768, 1024 und 1440 Pixeln; Einstellungen/Parkplatz laufen bei 320, 390, 1024
  und 1440 Pixeln;
- prüft kritische Shell-, Dialog-, Verwaltungs- und Energiegeometrie zusätzlich
  an 320, 360, 430, 768, 900/901, 1024, 1119/1120/1121 und 1280 Pixeln sowie
  mit kurzen Fensterhöhen;
- prüft Überschriften, zentrale Aktionen, Rollenverbote, die Größenhierarchie
  des Hauszeichens und horizontalen Überlauf;
- baut und speichert alles in einem temporären Verzeichnis und räumt es beim
  Beenden wieder auf.

Voraussetzungen sind Go, Node.js, npm und ein lokales Chromium oder Google
Chrome. Falls die Playwright-Abhängigkeiten fehlen, installiert der Runner die
in `scripts/snapshot/package-lock.json` festgeschriebene Version ohne einen
Browser herunterzuladen. In CI wird das zur festgeschriebenen
Playwright-Version gehörende Chromium installiert. Alle automatisierten Läufe
starten Chromium ausdrücklich headless; das hängt nicht von Playwrights
Standardwert ab. Nur der breite Basis-Rollenlauf kann zur lokalen Fehlersuche
mit `HV_QA_HEADLESS=false` sichtbar gestartet werden; die spezialisierten
Lebenszyklen bleiben headless. In CI bleibt auch der Basislauf zwingend
headless.

## Required CI gate

The parallel Blacksmith job `Full browser flow suite` runs this complete
orchestrator headlessly on every pull request and every push to `main`. The job
does not set the focused energy, fast, landing-only, or reduced role-matrix
flags. A failure therefore makes every role/route pass and every specialist
public, resident, handover, document, settings, parking, and HAUSV Home
lifecycle fail closed in CI.

The suite uses only invented local data and verifies:

- login for residents, owners, managers, and admins;
- the home overview, issues, and resident content on desktop and mobile;
- guided energy onboarding and HAUSV Home activation through a persistent
  local connector;
- logout followed by browser Back without protected content reappearing;
- public login/map, handover, document/e-invoice, settings, and parking flows
  across their targeted viewport matrices;
- real writes and downloads, role denials, primary actions, touch targets,
  browser errors, and horizontal overflow.

Das private Repository bietet im aktuellen GitHub-Tarif weder Branch Protection
noch Rulesets; Push oder Merge werden daher nicht von GitHub selbst gesperrt.
Verbindlich ist das Gate trotzdem für Produktion: `scripts/deploy.sh`
akzeptiert ausschließlich einen vollständig grünen `CI`-Push-Lauf auf
Blacksmith für exakt den auszurollenden Commit. Ein roter Browserjob verhindert
damit fail-closed das Deployment. Der vollständige Releasevertrag steht in
`docs/production-deploy.md`.

Run the same gate locally with:

```fish
npm --prefix scripts/snapshot exec -- playwright install chromium
set -lx CI true
set -lx HV_QA_HEADLESS true
set -lx HV_QA_ARTIFACT_DIR ./tmp/browser-role-qa
scripts/qa-main-flows.sh
```

With `CI=true`, the local run uses Playwright's pinned Chromium, as Blacksmith
does, instead of a system browser.

Das Artefaktverzeichnis enthält Build-, Fake-Home-Assistant-, Fake-SMTP-, App-, Basis-,
Public/Auth-, Übergabe-, Dokument-, Einstellungs-/Parkplatz- und
Strukturprüfungs-Logs. Die Fachharnesses legen ihre Berichte und
Vollseiten-Screenshots in getrennten Unterordnern ab; der Basislauf ergänzt bei
Fehlern Playwright-Traces, Fehlermeldung und Browserkonsolen-Ereignisse. CI lädt
diese Belege nur bei einem Fehlschlag für sieben Tage hoch. URLs in
Browser-Logs werden auf ihren Pfad gekürzt; die Fixtures enthalten keine
produktiven Geheimnisse oder Konten.

Der Fehlerpfad der Artefakterzeugung kann lokal bewusst ausgelöst werden:

```fish
set -lx HV_QA_CI_CORE true
set -lx HV_QA_FAILURE_PROBE true
set -lx HV_QA_ARTIFACT_DIR ./tmp/browser-role-qa-failure
scripts/qa-main-flows.sh
```

Dieser Prüflauf muss fehlschlagen. `HV_QA_FAILURE_PROBE` ist ausschließlich für
die Wartung des QA-Harness gedacht und wird in CI nicht gesetzt.

Die automatisierte UX-Selbstprüfung orientiert sich an WCAG 2.2: sichtbarer
Tastaturfokus, sprechende Beschriftungen, kein horizontaler Überlauf und für
primäre mobile Bedienelemente mindestens 44 Pixel Höhe (bewusst strenger als
das AA-Minimum von 24 × 24 CSS-Pixeln). Sie ersetzt keine spätere Beobachtung
mit realen Bewohnern; diese bleibt in PPM als eigene Pilotabnahme sichtbar.

Die Autorisierungsprüfung folgt dem serverseitigen Fail-closed-Prinzip: Ein
ausgeblendeter Schalter allein ist kein Schutz. Playwright und Go-Tests senden
deshalb auch direkte unzulässige Requests und prüfen fremde Häuser. Logs werden
anschließend strukturell validiert und auf versehentlich ausgegebene
Zugangsdaten beziehungsweise personenbezogene Inhalte stichprobenartig geprüft.

Die fachliche Zuordnung der 16 Hauptwege zu den einzelnen Harnesses steht in
[`portal-click-flows.md`](portal-click-flows.md#automatisierter-beleg). Der
separate `scripts/snapshot/capture.mjs`-Pfad erzeugt 4 Rollen × 14 Routen als
56 HTML-/PNG-Aufnahmen. Er dient der visuellen Breitenprüfung und wird nicht als
Klick- oder Berechtigungsnachweis gezählt.

Referenzen:

- <https://www.w3.org/TR/WCAG22/>
- <https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum.html>
- <https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html>
- <https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck>

Für einen abweichenden Port:

```fish
set -lx HV_QA_PORT 8125
scripts/qa-main-flows.sh
```

Wenn Go nicht im `PATH` liegt, findet der Runner unter Nix automatisch die in
`go.mod` angegebene Version. Alternativ kann `HV_GO` auf das gewünschte
Go-Binary zeigen.

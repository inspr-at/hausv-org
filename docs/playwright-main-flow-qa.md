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

## Verbindliches CI-Gate

Der parallele Blacksmith-Job `Full browser flow suite` führt den vollständigen
Orchestrator bei jedem Pull Request und jedem Push auf `main` headless aus.
Er setzt weder `HV_QA_ENERGY_ONLY`, `HV_QA_FAST`, `HV_QA_LANDING_ONLY` noch
`HV_QA_CI_CORE`; ein reduzierter lokaler Lauf entspricht diesem Gate nicht.
Vorher startet `verify-versions.mjs --strict` Chromium und prüft die Toolchain.

Die Reihenfolge in `scripts/qa-main-flows.sh` ist:

1. `qa-main-flows.mjs`: Rollen, Bewohnerwege, Energie und Geometrie.
2. `qa-public-auth.mjs`: öffentlicher Einstieg, Anmeldung und Karte.
3. `qa-handover-flow.mjs`: Übergabe bis zur Ablage.
4. `qa-document-flow.mjs`: Dokumente und E-Rechnung.
5. `qa-settings-parking.mjs`: zuerst leer, danach mit befüllter Parkplatz-Fixture.
6. `qa-home-setup.mjs`: `create`, Prozessneustart, `verify`.
7. Nachweis der lokalen Karten-Fixture und strukturierte Log-Prüfung.

Zwischen den Fachharnesses wird der Portalprozess neu gestartet; die Datenbasis
bleibt erhalten. Jeder fehlgeschlagene Schritt beendet den Lauf.

Das private Repository bietet im aktuellen GitHub-Tarif weder Branch Protection
noch Rulesets; Push oder Merge werden daher nicht von GitHub selbst gesperrt.
Verbindlich ist das Gate trotzdem für Produktion: `scripts/deploy.sh`
akzeptiert ausschließlich einen vollständig grünen `CI`-Push-Lauf auf
Blacksmith für exakt den auszurollenden Commit. Ein roter Browserjob verhindert
damit fail-closed das Deployment. Der vollständige Releasevertrag steht in
`docs/production-deploy.md`.

Dasselbe Gate startet lokal aus dem Worktree-Root so:

```sh
direnv exec . npm --prefix scripts/snapshot exec -- playwright install chromium
CI=true HV_QA_HEADLESS=true HV_QA_ARTIFACT_DIR=/tmp/hausv-oracles/full \
  direnv exec . bash scripts/qa-main-flows.sh
```

`CI=true` unterbindet die automatische Suche nach einem Systembrowser. Ohne
explizites `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH` wird das zu Playwright gehörende
Chromium verwendet.

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
separate `scripts/snapshot/capture.mjs`-Pfad erfasst 4 Rollen × 19 Routen plus
3 Cockpit-Routen bei 1440 × 900, insgesamt 79 geplante HTML-/PNG-/Vertragsaufnahmen.
Er bricht bei unerwarteten Weiterleitungen ab und wird nicht als Klick- oder
Berechtigungsnachweis gezählt.

## Orakel und CI-Gates

Stand: 10. September 2026, Arbeitsbaum auf Basis von 1.9.26. Maßgeblich sind die
Aufrufe in `.github/workflows/ci.yml` und die ausgeführten Prüfungen, nicht alte
Zählungen in Kopfkommentaren. `Portal parity and responsive shell` startet
nacheinander `shell-widths.mjs` (8299), `qa-viewport-actions.mjs` (8499) und
`templ-coverage.mjs` (8399). Trotz des Jobnamens gibt es keinen Vergleich zweier
Renderer mehr. Beide Browserjobs laden Fehlerartefakte für sieben Tage hoch.

Alle folgenden Dateien liegen unter `scripts/snapshot/`. „Fachlauf“ bezeichnet
den vollständigen lokalen Aufruf oben; dessen Portalport ist standardmäßig
8121. „Lokal“ bedeutet: kein Aufruf dieses Skripts in `ci.yml` oder im
Fachlauf. Port 8099 ist dafür nur ein lokales Beispiel.

Für die Tabellenaufrufe aus dem Worktree-Root diese Shell-Funktion verwenden:

```sh
qa_orakel() {
  CI=true HV_QA_HEADLESS=true HV_CAPTURE="$1" \
    direnv exec . bash scripts/snapshot/run.sh WORKTREE "$2" "$3"
}
direnv exec . mkdir -p /tmp/hausv-oracles
```

Das entspricht dem Einzelaufruf
`CI=true HV_CAPTURE=<orakel>.mjs direnv exec . bash scripts/snapshot/run.sh WORKTREE <ausgabe> <port>`.
Die drei Portal-Gates schreiben JSON-Dateien; ihr Elternverzeichnis muss schon
existieren. Die übrigen `qa_orakel`-Aufrufe erhalten Artefaktverzeichnisse. Die
Versionsprüfung startet ohne App-Port.

| Datei | Misst | Blind für | CI-Job / Port | Lokaler Start |
| --- | --- | --- | --- | --- |
| `qa-main-flows.mjs` | Rollen-/Routenmatrix, echte Bewohneraktionen, Onboarding, Energie-Sicherheit, Geometrie und Abmelden/Zurück. | Nicht ausgeführte Rollen, Unterwege und reale Gerätewirkung; keine lückenlose Paint-Messung. | `Full browser flow suite` / 8121 | Fachlauf |
| `qa-public-auth.mjs` | Tenant-Einstieg, lokale Anmeldelinks, Karte samt Fehlerzustand, Historie, No-JS und Abmelden. | Echter Mailtransport, externer OIDC-Anbieter und reale Kartendienste. | `Full browser flow suite` / 8121 | Fachlauf |
| `qa-handover-flow.mjs` | Anlage, Anhänge, beide Bestätigungen, Sperren, Ablage, PDF und Lightbox. | Reale Zustellung der Bestätigungslinks; deren Token-Hashes werden in der Fixture ersetzt. | `Full browser flow suite` / 8121 | Fachlauf |
| `qa-document-flow.mjs` | Upload, Sichtbarkeit, Vorschau, Download, Ersetzen und E-Rechnung samt direkten Rollenverboten. | Beliebige Dateiformate und Inhalte außerhalb der Fixtures. | `Full browser flow suite` / 8121 | Fachlauf |
| `qa-settings-parking.mjs` | Einstellungen, Einladungen, Export/CAMT, Parkplatzrechte sowie leere und befüllte Monate. | Reale Banken, Ladegeräte und nicht modellierte Monatszustände. | `Full browser flow suite` / 8121, zweimal | Fachlauf; Fixture-Wechsel und Neustart gehören dazu |
| `qa-home-setup.mjs` | Reservierung, lokale SMTP-Mail, Aktivierung, Connector und Mandantentrennung nach Neustart. | Externe Mailzustellung und reale Connector-Installation. | `Full browser flow suite` / 8121 | Fachlauf; benötigt SMTP, `create`/`verify` und Zustandsdatei |
| `shell-widths.mjs` | Einheitliche sichtbare Navigation und Shell-Geometrie an 19 Breiten; Hausüberblick, Tablet-Spalten, Breiten-Sweep und Posteingangs-Aktionsbereich. | Vollständige Klickwege, alle Rollen und Paint-Flackern; fehlende Aktionsleiste allein ist kein Fehler. | `Portal parity and responsive shell` / 8299 | `qa_orakel shell-widths.mjs /tmp/hausv-oracles/widths.json 8299` |
| `qa-viewport-actions.mjs` | Erreichbarkeit und Trefferpunkt ausgewählter Aktionen, scrollender Panels und Dialogabschlüsse bei 1512 × 982, 1440 × 900 und 1280 × 720. | Fachlicher Erfolg nach Absenden, andere Rollen/Breiten, übersprungene HTTP-Fehler, nicht erfasste Selektoren und ausgelassene Dialoge. | `Portal parity and responsive shell` / 8499 | `qa_orakel qa-viewport-actions.mjs /tmp/hausv-oracles/viewport-actions.json 8499` |
| `templ-coverage.mjs` | 22 gepflegte Routen: HTTP 200, exaktes Routenziel, templ-Marker; Unterlängen kompakter Titel bei 1440 und 390 Pixeln. | Vollständiger Inhalt, Schreibaktionen und Unterrouten außerhalb der Liste; kein Vergleich mit dem entfernten Renderer. | `Portal parity and responsive shell` / 8399 | `qa_orakel templ-coverage.mjs /tmp/hausv-oracles/renderer.json 8399` |
| `qa-nav-flicker.mjs` | Helligkeit tatsächlicher CDP-Frames bei 18 Admin-Seitenwechseln auf 1440 × 900; helle Sidebar oder dunkler Inhalt. | Nicht gelieferte Frames, kleine lokale Farbfehler unter den Flächenmittelwerten und andere Viewports. | Lokal / 8099 | `qa_orakel qa-nav-flicker.mjs /tmp/hausv-oracles/nav-flicker 8099` |
| `qa-nav-paint.mjs` | Frühe Sidebar-Pixel und gespeicherte Breite vor absichtlich verzögertem Shell-CSS, danach Geometrie, Hover und Fokus auf fünf Verwaltungsrouten. | Unverändertes natürliches Ladeverhalten, andere Pixelbereiche und mobile Drawer; HTML wird für die Probe abgefangen. | Lokal / 8099 | `qa_orakel qa-nav-paint.mjs /tmp/hausv-oracles/nav-paint 8099` |
| `qa-navigation.mjs` | Echte Menülinks, gemeinsame Blöcke/Abstände, aktive Route, Footer-Erreichbarkeit und Scrollposition über Seitenwechsel für Admin, Eigentümer und Bewohner. | Fachaktionen der Zielseiten, weitere Rollen und Pixelhelligkeit zwischen den DOM-Messungen. | Lokal / 8099 | `qa_orakel qa-navigation.mjs /tmp/hausv-oracles/navigation 8099` |
| `qa-switcher.mjs` | Kontext-/Hauswechsel, Suche per Tastatur und No-JS, Rollen-Vorschau, Popover-Geometrie, Sidebar-Breite und reduzierte Bewegung. | Vollständige Berechtigungsprüfung; Rollen-Vorschau ist kein Login als diese Rolle. | Lokal / 8099 | `qa_orakel qa-switcher.mjs /tmp/hausv-oracles/switcher 8099` |
| `qa-font-stability.mjs` | Source Serif 4, Cache/Preload/CSP und Geometrie von Identität/Falltitel beim Kaltstart und J/K-Navigieren an sechs Breiten. | Andere Texte/Schriften; Kaltstart-Geometriesprünge werden nicht wie warme Besuche verglichen, erster Messframe kann nach FCP liegen. | Lokal / 8099 | `qa_orakel qa-font-stability.mjs /tmp/hausv-oracles/font 8099` |
| `qa-issue-board.mjs` | Fünf Spalten, Detailpanel, Tastatur/Fokus, Status/Zuständigkeit/Termin, Abbruch, Netzwerk-Rollback, Touch und No-JS. | Andere Rollen und echte Touchgeräte; Desktop-Drag verwendet synthetische DragEvents, Touch CDP-Ereignisse. | Lokal / 8099 | `qa_orakel qa-issue-board.mjs /tmp/hausv-oracles/issue-board 8099` |
| `qa-inbox-suggest.mjs` | Tenant-Präfix und stabiler Vorschlagsbereich beim Polling, Erfolg, Verzögerung, Anbieter-/HTTP-/Netzfehler, Retry und Abbruch. | Qualität echter KI-Antworten und externer Anbieterbetrieb; lokaler Stub und gezielt abgefangene Antworten. | Lokal / 8099 | `qa_orakel qa-inbox-suggest.mjs /tmp/hausv-oracles/inbox-suggest 8099` |
| `qa-legacy-routes.mjs` | Migrierte Detail-/Einstellungsseiten und echte Import-/Exportvorschauen: HTTP/Zielroute, templ-Marker, Shell, Überschrift, Touch-Ziele und Überlauf an drei Breiten. | Vollständiger fachlicher Abschluss aller gezeigten Formulare und allgemeine Rollenverbote. | Lokal / 8099 | `qa_orakel qa-legacy-routes.mjs /tmp/hausv-oracles/legacy-routes 8099` |
| `energy-redesign-contract.mjs` | Über aufgerufene Exports: Energie-Kopf, Verbraucher-Dialoge, Offenlegungen, Flussgeometrie und Grenze zum Verlauf. | Allein gestartet keinerlei Prüfung; Verlauf hinter der Grenze und reale Energie-/Gerätewirkung. | Indirekt über `qa-main-flows.mjs` in `Full browser flow suite` / 8121 | Fachlauf; kein eigenes `HV_CAPTURE` |
| `contract.mjs` | `EXTRACT_CONTRACT` erfasst Scripts, Body-Attribute, Formziele, Bestätigungs-/Pflichtfelder und Daten-Hooks aus dem DOM. | Verhalten der Hooks, Serverberechtigungen und Sollvergleich; Export allein führt nichts aus. | Lokal, über `capture.mjs` / 8099 | `qa_orakel capture.mjs /tmp/hausv-oracles/capture 8099` |
| `capture.mjs` | Normalisierte HTML-, PNG- und DOM-Vertragsaufnahmen für 79 geplante Rollen-/Routenkombinationen; Weiterleitungen brechen ab. | Automatischer Sollvergleich, Klick-/Berechtigungsnachweis und mobile Breiten; HTTP-Status wird nur aufgezeichnet. | Lokal / 8099 | `qa_orakel capture.mjs /tmp/hausv-oracles/capture 8099` |
| `verify-versions.mjs` | Laufende Node- und installierte Playwright-Version gegen deklarierte Pins; startet headless Chromium und protokolliert dessen Version. | Produktverhalten und separater Chromium-Sollversionsvergleich; ohne `--strict` bleiben Abweichungen reine Meldungen. | `Full browser flow suite`, vor Fachlauf / kein App-Port | `direnv exec . node scripts/snapshot/verify-versions.mjs --strict` |

`run.sh WORKTREE` baut den aktuellen Baum einschließlich uncommitteter Änderungen
mit Go; es legt keinen zusätzlichen Git-Worktree an und verwendet keinen Stash.
Der Runner startet frische temporäre Daten, lokale HA-/Karten-Fixtures und die
App, wartet auf `/healthz` und übergibt Basis-URL sowie Ausgabe an `HV_CAPTURE`
(Standard: `capture.mjs`). Der normale Abschluss beendet die Prozesse und gibt
den Exitcode des Orakels zurück. Die ausgegebenen Artefakte liegen außerhalb
des temporären Datenverzeichnisses.

Worktrees brauchen `scripts/snapshot/node_modules` als Symlink auf das
`scripts/snapshot/node_modules` des Hauptcheckouts mit den installierten
Playwright-Abhängigkeiten; `run.sh` installiert sie nicht. Zusätzlich muss
Chromium installiert sein. `qa-inbox-suggest.mjs` erhält durch den Runner einen
lokalen KI-Stub mit Port `App-Port + 200` und `haus-b` als abweichenden
Standardmandanten. `qa-legacy-routes.mjs` erhält eine befüllte Parkplatz-Fixture.
Die HA-Fixture verwendet standardmäßig `App-Port + 100`; diese Ports müssen
neben dem App-Port frei sein.

`qa-viewport-actions.mjs` führt eine derzeit leere `KNOWN`-Allowlist mit dem
Schlüssel Route, Viewport, Fehlerart und Beschriftung. Bekannte Treffer bleiben
im Bericht, lassen den Lauf aber nicht scheitern; `page-scroll` ist nur eine
Information. Höchstens vier sichtbare Dialogauslöser je Route werden versucht;
fehlgeschlagene Öffnungen können ohne Befund ausfallen. Deshalb auch `seen`,
übersprungene Routen und die tatsächlich gemessenen Aktionen prüfen.

Das Anmeldelimit beträgt fünf Anmeldungen je 15 Minuten je Demo-Konto.
`qa-switcher.mjs` verwendet gespeicherte Sessions je Konto und für die gesamte
Vorschau-Matrix eine Admin-Session; `qa-font-stability.mjs` übernimmt eine
Admin-Session in alle Breitenkontexte. Nicht für jede Breite neu anmelden.
`energy-redesign-contract.mjs` enthält nur Exports: ein direkter Node-Aufruf
beendet sich ohne Messung. Auch `contract.mjs` braucht einen aufrufenden Harness.

Eine Prüfung, die man nie hat scheitern sehen, misst nichts. Ein grüner Exitcode
allein belegt weder erreichte Prüfpunkte noch Empfindlichkeit gegen Regressionen.
Zum Orakel gehören ausgeführte Fälle und ein bekannter Gegenbeleg; die bloße
Existenz von Assertions im Code ist noch kein beobachteter Fehlschlag.

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

## Runbook-Kurzfassung

1. `Full browser flow suite` führt den vollständigen Fachlauf ohne Reduktionsflags aus.
2. Reihenfolge: Basis → Public/Auth → Übergabe → Dokumente → Parkplatz leer/befüllt → Home create/verify → Karten-/Log-Prüfung.
3. `Portal parity and responsive shell`: Shell 8299 → Viewport-Aktionen 8499 → templ-Abdeckung 8399.
4. Lokal aus dem Worktree-Root: `CI=true HV_CAPTURE=<orakel>.mjs direnv exec . bash scripts/snapshot/run.sh WORKTREE <ausgabe> <port>`.
5. `WORKTREE` baut den aktuellen Baum ohne Stash mit frischen lokalen Fixtures; Zusatzports freihalten.
6. Voraussetzung: `scripts/snapshot/node_modules` als Worktree-Symlink, installiertes Chromium und vorhandener JSON-Elternordner.
7. Viewport-`KNOWN` gilt je Route/Breite/Fehler/Beschriftung; Allowlist, übersprungene Routen und Dialogmessungen mitlesen.
8. Fünf Logins je 15 Minuten/Konto: Switcher-Vorschau und Schriftprüfung verwenden eine Admin-Session weiter.
9. Exportmodule allein prüfen nichts; Capture ersetzt keinen Klickweg, lokale Zusatzorakel sind keine eigenen CI-Gates.
10. Messbereich und blinde Stellen nennen; eine Prüfung ohne beobachteten Fehlschlag ist kein belastbarer Nachweis.

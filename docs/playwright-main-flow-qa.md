# Playwright-QA der Hauptwege

Der lokale QA-Lauf startet das aktuelle Arbeitsverzeichnis mit ausschließlich
erfundenen Daten, mehreren Rollen und drei strikt getrennten Hausprofilen. Er
sendet keine E-Mails, greift nicht auf Produktivdaten zu und lädt auch keine
Kartenkacheln von einem öffentlichen Dienst.

Der vollständige lokale Lauf bleibt das umfassende Browser-Regressionsgate:

```fish
scripts/qa-main-flows.sh
```

Der Lauf:

- meldet Bewohner, Eigentümer, Verwalter und Admin über den lokalen
  Entwicklungszugang an;
- legt einen Aushang, einen Termin, einen Kontakt, ein Dokument und ein Anliegen
  an;
- prüft das private Haus-Cockpit für Bewohner und Eigentümer, einschließlich
  dauerhaft sichtbarem Beobachtungsmodus, bewusstem Shadow-Testlauf,
  Sofort-Rückkehr und idempotentem Smart-Meter-Import;
- spielt die Eltern-Konstellation mit PV, E-Auto, Warmwasser-Wärmepumpe und
  bewusst fehlendem Speicher sowie die Schwiegereltern-Konstellation mit PV,
  Speicher, E-Auto und eigener technischer Vertrauensperson durch;
- wechselt im Onboarding zwischen Wohnung, Einfamilienhaus und
  Hausgemeinschaft und prüft, dass Umfang, unveränderte Rechte und der
  Beobachtungsmodus direkt verständlich bleiben;
- prüft an drei simulierten, ausschließlich lesenden Home-Assistant-Instanzen,
  dass Messwerte hausbezogen bleiben, höchstens fünf ruhige Vorschläge
  erscheinen und Messlücken ohne Entity-IDs erklärt werden;
- bedient den festen Kartenausschnitt aus einer lokalen gültigen PNG-Fixture und
  prüft sowohl Dekodierung als auch Content-Type/PNG-Signatur;
- öffnet die Hauptwege auf 1440 × 900 und 390 × 844;
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
Standardwert ab. Nur zur lokalen Fehlersuche kann mit
`HV_QA_HEADLESS=false` ein sichtbares Browserfenster angefordert werden. In CI
bleibt der Lauf auch dann zwingend headless.

## Verpflichtendes CI-Gate

Der parallele Blacksmith-Job `Browser roles + mobile` führt bei jedem Push auf
`main` und bei jedem Pull Request einen bewusst kompakten Kernlauf aus. Ein
Fehler macht den Workflow rot. Der Kernlauf prüft mit vollständig erfundenen,
lokalen Daten:

- Anmeldung als Bewohner und Admin;
- Hausüberblick und Anliegen auf Desktop und Mobil;
- das geführte Energie-Onboarding auf Desktop und Mobil;
- Abmelden mit anschließendem Browser-Zurück ohne wieder sichtbare
  Portal-Inhalte;
- Rollenverbote, primäre Aktionen, Touch-Ziele und horizontalen Überlauf.

Das private Repository bietet im aktuellen GitHub-Tarif weder Branch Protection
noch Rulesets; Push oder Merge werden daher nicht von GitHub selbst gesperrt.
Verbindlich ist das Gate trotzdem für Produktion: `scripts/deploy.sh`
akzeptiert ausschließlich einen vollständig grünen `CI`-Push-Lauf auf
Blacksmith für exakt den auszurollenden Commit. Ein roter Browserjob verhindert
damit fail-closed das Deployment. Der vollständige Releasevertrag steht in
`docs/csb1-deploy.md`.

Der gleiche Lauf lässt sich lokal so reproduzieren:

```fish
set -lx HV_QA_CI_CORE true
set -lx HV_QA_ARTIFACT_DIR ./tmp/browser-role-qa
scripts/qa-main-flows.sh
```

Das Artefaktverzeichnis enthält Build-, Fake-Home-Assistant-, App-,
Playwright- und Strukturprüfungs-Logs. Bei einem Browserfehler kommen
Vollseiten-Screenshots, Playwright-Traces, eine Fehlermeldung und relevante
Browserkonsolen-Ereignisse hinzu. CI lädt diese Belege nur bei einem Fehlschlag
für sieben Tage hoch. URLs in Browser-Logs werden auf ihren Pfad gekürzt; die
Fixtures enthalten keine produktiven Geheimnisse oder Konten.

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

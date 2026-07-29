# Playwright-QA der Hauptwege

Der lokale QA-Lauf startet das aktuelle Arbeitsverzeichnis mit ausschließlich
erfundenen Daten, mehreren Rollen und drei strikt getrennten Hausprofilen. Er
sendet keine E-Mails und greift nicht auf Produktivdaten zu.

Dieser Lauf ist derzeit das umfassende lokale Browser-Regressionsgate. Er ist
noch kein verpflichtender CI-Schritt; die CI-Integration und ihr Schutz für
Änderungen auf `main` bleiben ausdrücklich in HAUSV-404 offen.

```fish
scripts/qa-main-flows.fish
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
- öffnet die Hauptwege auf 1440 × 900 und 390 × 844;
- prüft Überschriften, zentrale Aktionen, Rollenverbote, die Größenhierarchie
  des Hauszeichens und horizontalen Überlauf;
- baut und speichert alles in einem temporären Verzeichnis und räumt es beim
  Beenden wieder auf.

Voraussetzungen sind Go, Node.js, npm und ein lokales Chromium oder Google
Chrome. Falls die Playwright-Abhängigkeiten fehlen, installiert der Runner die
in `scripts/snapshot/package-lock.json` festgeschriebene Version ohne einen
Browser herunterzuladen.

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
scripts/qa-main-flows.fish
```

Wenn Go nicht im `PATH` liegt, findet der Runner unter Nix automatisch die in
`go.mod` angegebene Version. Alternativ kann `HV_GO` auf das gewünschte
Go-Binary zeigen.

# Playwright-QA der Hauptwege

Der lokale QA-Lauf startet das aktuelle Arbeitsverzeichnis mit ausschließlich
erfundenen Daten und vier Rollen. Er sendet keine E-Mails und greift nicht auf
Produktivdaten zu.

```fish
scripts/qa-main-flows.fish
```

Der Lauf:

- meldet Bewohner, Eigentümer, Verwalter und Admin über den lokalen
  Entwicklungszugang an;
- legt einen Aushang, einen Termin, einen Kontakt, ein Dokument und ein Anliegen
  an;
- öffnet die Hauptwege auf 1440 × 900 und 390 × 844;
- prüft Überschriften, zentrale Aktionen, Rollenverbote und horizontalen
  Überlauf;
- baut und speichert alles in einem temporären Verzeichnis und räumt es beim
  Beenden wieder auf.

Voraussetzungen sind Go, Node.js, npm und ein lokales Chromium oder Google
Chrome. Falls die Playwright-Abhängigkeiten fehlen, installiert der Runner die
in `scripts/snapshot/package-lock.json` festgeschriebene Version ohne einen
Browser herunterzuladen.

Für einen abweichenden Port:

```fish
set -lx HV_QA_PORT 8125
scripts/qa-main-flows.fish
```

Wenn Go nicht im `PATH` liegt, findet der Runner unter Nix automatisch die in
`go.mod` angegebene Version. Alternativ kann `HV_GO` auf das gewünschte
Go-Binary zeigen.

# Supply-Chain-Pins: was festgelegt ist und wie man es aktualisiert

Die Release-Freigabe bindet ein Deployment an genau einen Commit. Diese Zusage
ist nur so viel wert wie die Eingaben dahinter: ein bewegliches
`actions/checkout@v6` oder ein `FROM golang:1.26.5-alpine` ohne Digest lässt
denselben Commit morgen aus anderen Bytes bauen.

Deshalb sind alle extern aufgelösten CI- und Build-Eingaben unveränderlich
festgelegt. `scripts/check-supply-chain-pins.py` erzwingt das bei jedem CI-Lauf
und lokal; es braucht kein Netz und meldet alle Funde auf einmal.

## Was festgelegt ist

| Eingabe | Form | Ort |
|---|---|---|
| GitHub Actions | vollständige Commit-SHA + lesbarer `# <tag>`-Kommentar | `.github/workflows/ci.yml` |
| Node | exakte Patchversion | `.github/workflows/ci.yml` |
| Playwright und Browser | exakte Version plus `package-lock.json`, Installation mit `npm ci` | `scripts/snapshot/` |
| Docker-Basisimages | lesbarer Tag **und** `@sha256:`-Digest | `Dockerfile` |
| govulncheck | exakte Version im `go install`-Aufruf | `.github/workflows/ci.yml` |
| Go | `go-version-file: go.mod` | `.github/workflows/ci.yml` |

Ein kurzer SHA gilt nicht: er ist mehrdeutig und wird abgelehnt.

## Aktualisieren

Pins altern absichtlich. Sie zu erneuern ist eine bewusste Handlung, kein
Nebeneffekt — Sicherheitsupdates sollen zeitnah übernommen werden, aber
nachvollziehbar.

### GitHub Action

```fish
gh api repos/actions/checkout/git/ref/tags/v6 --jq '.object.sha'
```

Löst das Ref auf ein annotiertes Tag auf (`.object.type` ist dann `tag`), muss
zusätzlich `gh api repos/actions/checkout/git/tags/<sha> --jq '.object.sha'`
den Commit auflösen. Den lesbaren Tag als Kommentar hinter der SHA behalten.

### Node

```fish
curl -s https://nodejs.org/dist/index.json | jq -r '[.[] | select(.version | startswith("v24."))][0].version'
```

### Docker-Basisimage

```fish
docker buildx imagetools inspect golang:1.26.5-alpine --format '{{.Manifest.Digest}}'
```

Tag und Digest gehören zusammen in dieselbe `FROM`-Zeile: der Tag erklärt, was
gemeint ist, der Digest legt fest, was gebaut wird.

### Playwright und Browser

Version in `scripts/snapshot/package.json` ändern, `npm install` im selben
Verzeichnis ausführen und `package-lock.json` mitcommitten. CI installiert mit
`npm ci`, nimmt also ausschließlich den Lockfile-Stand.

## Was tatsächlich gestartet wurde

Ein Pin in einer Konfigurationsdatei sagt, was laufen *soll*.
`scripts/snapshot/verify-versions.mjs` sagt, was tatsächlich lief: der Node-
Interpreter des Laufs, das aus dem Lockfile aufgelöste Playwright und die
Chromium-Version, die wirklich startet.

Die Sollwerte liest das Skript aus den Dateien, die sie ohnehin deklarieren —
`.github/workflows/ci.yml` für Node, `package.json` für Playwright. Es gibt
also keine zweite Stelle, die gepflegt werden müsste.

```fish
cd scripts/snapshot
node verify-versions.mjs            # nur Bericht
node verify-versions.mjs --strict   # Abweichung lässt den Lauf scheitern
```

CI führt die strikte Variante aus. Lokal berichtet das Skript nur, denn die
Node-Version der Entwicklungsumgebung stammt aus `flake.nix` (nixpkgs) und
deckt sich nicht zwangsläufig mit der in CI gepinnten Patchversion. Diese
Abweichung ist bekannt und bewusst: CI ist die maßgebliche Umgebung, und die
lokale Meldung macht den Unterschied sichtbar, statt ihn zu verbergen.

## Regeln für die Aktualisierung

1. Eine Änderung pro Commit-Absicht: Pins gemeinsam aktualisieren, nicht
   verstreut über mehrere Commits.
2. Den vollständigen CI-Lauf abwarten — Go, Browser-QA, govulncheck und
   Docker-Build. Ein grüner Teilbereich ist keine Freigabe.
3. Im Commit festhalten, *warum* aktualisiert wurde (Sicherheitsupdate,
   Funktionsbedarf, Routinepflege).

## Bekannte Lücke

`fish` wird in CI weiterhin über `apt-get install` aus dem Ubuntu-Repository des
Runner-Images bezogen und ist damit nicht festgelegt. Eine Versionsangabe an
`apt-get` hilft nicht dauerhaft: Ubuntu entfernt alte Paketstände, der Lauf
würde später brechen statt reproduzierbar zu bleiben.

Sinnvolle Wege, bewusst noch offen:

- Die Fish-Jobs in einem digest-gepinnten Container ausführen.
- `fish` über das bereits vorhandene `flake.nix` beziehen; `flake.lock` legt
  nixpkgs auf einen Commit fest und wäre damit die unveränderliche Quelle.

Der zweite Weg nutzt, was das Repository ohnehin für die lokale Entwicklung
mitbringt, kostet in CI aber die Installation von Nix. Die Entscheidung steht
noch aus; bis dahin ist dies die einzige bewegliche Eingabe im Lauf.

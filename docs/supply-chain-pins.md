# Supply-Chain-Pins: was festgelegt ist und wie man es aktualisiert

Die Release-Freigabe bindet ein Deployment an genau einen Commit. Diese Zusage
ist nur so viel wert wie die Eingaben dahinter: ein bewegliches
`actions/checkout@v6` oder ein `FROM golang:1.26.5-alpine` ohne Digest lässt
denselben Commit morgen aus anderen Bytes bauen.

Deshalb sind die extern aufgelösten CI- und Build-Eingaben festgelegt — bis auf
eine bewusste Ausnahme, den Go-Patchstand (siehe unten).

Durchgesetzt wird das an zwei Stellen, und es ist wichtig zu wissen, welche
was abdeckt:

- `scripts/check-supply-chain-pins.py` prüft textuell drei Dinge: Action-SHAs,
  die exakte Node-Patchversion und `sha256`-Digests der Basisimages. Es braucht
  kein Netz und meldet alle Funde auf einmal.
- `scripts/snapshot/verify-versions.mjs` prüft zur Laufzeit, was tatsächlich
  gestartet ist: Node, Playwright und Chromium.

Nicht maschinell geprüft sind die govulncheck-Version, der lesbare
`# <tag>`-Kommentar hinter einer Action-SHA und der Go-Pin. Wer diese ändert,
muss selbst hinsehen.

## Was festgelegt ist

| Eingabe | Form | Ort |
|---|---|---|
| GitHub Actions | vollständige Commit-SHA + lesbarer `# <tag>`-Kommentar | `.github/workflows/ci.yml` |
| Node | exakte Patchversion | `.github/workflows/ci.yml` |
| Playwright und Browser | exakte Version plus `package-lock.json`, Installation mit `npm ci` | `scripts/snapshot/` |
| Docker-Basisimages | lesbarer Tag **und** `@sha256:`-Digest | `Dockerfile` |
| govulncheck | exakte Version im `go install`-Aufruf | `.github/workflows/ci.yml` |
| Go | Minor aus `go.mod`, Patch **bewusst beweglich** | `.github/workflows/ci.yml` |

Ein kurzer SHA gilt nicht: er ist mehrdeutig und wird abgelehnt.

Die Go-Zeile ist die eine bewusste Ausnahme: alle `setup-go`-Schritte setzen
`check-latest: true` und holen damit den neuesten Patch der in `go.mod`
festgelegten Minor-Version aus dem Netz. Das ist Absicht — so landen
Stdlib-CVE-Fixes ohne Zutun im Build (HAUSV-139: der `crypto/tls`-Fix kam in
einem Patch-Release). Der Preis ist, dass derselbe Commit morgen mit einem
anderen Go-Patch gebaut werden kann.

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
curl -s https://nodejs.org/dist/index.json \
  | jq -r '[.[] | select(.version | startswith("v24."))][0].version | ltrimstr("v")'
```

Das `ltrimstr("v")` gehört dazu: `node-version` muss `24.18.1` lauten. Mit dem
führenden `v` schlägt `check-supply-chain-pins.py` fehl.

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
Chromium-Version, die wirklich headless startet.

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

## Geschlossene Lücke: fish

`fish` wurde in CI über `apt-get install` aus dem Ubuntu-Repository des
Runner-Images bezogen und war damit die einzige unbewusst bewegliche Eingabe.
Statt sie festzulegen, ist sie entfallen: alle Skripte laufen seit HAUSV-427
unter bash (`scripts/*.sh`).

Zielstand ist bewusst bash 3.2 — die Fassung, die macOS als `/bin/bash`
mitliefert. Nur damit verschwindet die Abhängigkeit wirklich. Gegen eine
neuere bash aus nix zu schreiben hätte fish nur gegen eine andere
Voraussetzung getauscht, und ein versehentlich genutztes bash-4-Merkmal wäre
in CI (Ubuntu, bash 5) und lokal (nix, bash 5) durchgelaufen und ausgerechnet
auf dem Mac gebrochen, der das Produktions-Deployment ausführt.

Wer die Skripte ändert, prüft sie deshalb gegen `/bin/bash` und `shellcheck`:

```fish
/bin/bash -n scripts/deploy.sh
nix run nixpkgs#shellcheck -- -x --shell=bash scripts/*.sh scripts/snapshot/*.sh
```

Damit bleibt genau eine bewegliche Eingabe im Lauf: der Go-Patchstand, und der
ist Absicht.

# csb1 Deploy & Rollback (HAUSV)

**Stand:** 25. August 2026  
**Live Version:** 0.99.9 (Commit `04e6801`)  
**Live URL:** https://hausv.org/jhw22/  
**Letzter Deploy-Nachweis:** GitHub Actions Deploy-Workflow Run 32447870655  

## Architektur

### Repositories

- **Produkt-Repository:** `hausv-org` (öffentlich, GitHub `inspr-at/hausv-org`)
  - Enthält den HAUSV-Produktcode, Deployment-Skripte und versionsnahe Dokumentation
  - `scripts/deploy.sh` (attended Mac-Pfad, break-glass)
  - `scripts/deploy-from-ci-image.sh` (Mac-less CD-Pfad, csb1-Runner)
  - `docs/production-deploy.md` (vollständige Deployment-Dokumentation)
  
- **Private Instanz-Repository:** `hausv-jhw22` (privat, nicht öffentlich)
  - Enthält die compose.yml und instanzspezifische Konfiguration
  - Liegt auf csb1 unter `/home/mba/Code/hausv-jhw22`
  - Der normale Einstiegspunkt aus diesem Repo ist `scripts/deploy-product.sh ../hausv-org`

### Deployment-Architektur

**Mac-less Continuous Deployment (Standardpfad):**

1. Howie öffnet PR gegen `hausv-org` main
2. CI (Blacksmith-Runner) baut und testet bei jedem Push
3. Bei grünem CI auf main wird das Image nach GHCR gepusht
4. Deploy-Workflow (`.github/workflows/deploy.yml`) triggert automatisch
5. Self-hosted Runner `csb1-hausv` auf csb1 führt `scripts/deploy-from-ci-image.sh` aus
6. Deployment läuft unter der Compose-Lock durch: Pull CI-Image → Swap Container
7. Bei Schema-Änderungen nimmt der Runner automatisch ein Snapshot vor dem Swap

**Attended Mac Path (manuell, break-glass):**

- `scripts/deploy.sh` bleibt für manuelle Verifikation oder Notfälle erreichbar
- Erfordert SSH-Zugang zu csb1 und die private Deployment-Umgebung (`~/.config/hausv/deploy.env`)
- Details in `hausv-org/docs/production-deploy.md`

Amy führt die CD-Deployments durch. Howie merged NICHT selbst.

## Vorbedingungen (Fail-Closed Gates)

Jedes Production-Deployment muss diese Bedingungen erfüllen, sonst wird es abgelehnt:

1. **VERSION bump:** Jedes Release muss `VERSION` im Repo erhöhen
2. **CHANGELOG:** `docs/CHANGELOG.md` aktualisiert (Deutsch, neuester Eintrag oben, positive Formulierung)
3. **Version Notes:** `internal/version/version.go` `Notes()` aktualisiert; neuester Eintrag muss `VERSION` entsprechen (Test erzwingt dies)
4. **Commit pushed:** Der exakte Commit muss auf `origin/main` liegen
5. **Green CI:** Blacksmith CI muss für den exakten Commit erfolgreich durchgelaufen sein
6. **Dirty Tree:** Keine uncommitted Changes (würde Verwirrung erzeugen, was deployed wurde)
7. **Live Health:** Production muss vor Deployment gesund sein (`/healthz` ok)
8. **Schema-Änderungen:** Wenn `internal/db/migrations` geändert wurde, nimmt das System automatisch ein transaktional konsistentes SQLite+Blob-Snapshot vor dem Container-Swap

## Deployment-Workflow

### Automatisches Mac-less Deployment (Standardpfad)

**Trigger:** Automatisch nach grünem CI auf `main`, oder manuell via GitHub Actions `workflow_dispatch`

**Schritte:**

1. Self-hosted Runner `csb1-hausv` checkt `hausv-org` am grünen Commit aus
2. Runner prüft ob Migrations geändert wurden
3. Bei Schema-Änderungen: Automatisches Pre-Deploy-Snapshot (unter Compose-Lock)
4. Pull des CI-Image von `ghcr.io/inspr-at/hausv-org:release-<version>-<commit>`
5. Swap des Production-Containers (unter Compose-Lock, atomic retag + recreate)
6. Health- und Versions-Verifikation (Container-Health, `/healthz`, sichtbare Version)
7. Critical-Log-Check (keine Panics, keine failed migrations)

**Runner-Setup:** Der `csb1-hausv` Runner läuft als `mba` auf csb1, hat Zugriff auf Host-Docker, GHCR-Token (`/run/agenix/csb1-hausv-ghcr-pull`), Compose-Lock und Compose-Directory.

**Dry-Run:** 
```bash
# Auf csb1 als mba, im hausv-org Checkout:
bash scripts/deploy-from-ci-image.sh <version> <commit> --dry-run
```

**Manuelle Invocation:**  
GitHub Actions → Deploy → Run workflow → Branch `main` → Version (z.B. `0.99.9`) + Commit (7-char SHA)

### Attended Mac Deployment (break-glass)

**Verwendung:** Nur für manuelle Verifikation oder wenn der Runner nicht verfügbar ist.

**Voraussetzung:** Private Deployment-Umgebung (`~/.config/hausv/deploy.env`) mit allen Umgebungsvariablen.

**Commands:**

```fish
# Fish shell (auf Mac):
direnv exec . bash -c 'set -a; . ~/.config/hausv/deploy.env; set +a; bash scripts/deploy.sh --dry-run'
direnv exec . bash -c 'set -a; . ~/.config/hausv/deploy.env; set +a; bash scripts/deploy.sh'
```

```bash
# Bash:
set -a
. ~/.config/hausv/deploy.env
set +a

scripts/deploy.sh --dry-run
scripts/deploy.sh
```

**Detaillierte Dokumentation:** `hausv-org/docs/production-deploy.md`

## Production-Smoke-Test

Nach jedem Deployment (automatisch oder manuell):

1. **Container-Health:** `docker inspect --format '{{.State.Health.Status}}' hausv-org` → `healthy`
2. **Public Health:** https://hausv.org/healthz → Status `ok`, Service `hausv-org`
3. **Sichtbare Version:** https://hausv.org/jhw22/ → Version und Commit sichtbar im Footer/Page
4. **Login-Test:** Als Resident und als Manager einloggen (beide Tenants wenn vorhanden)
5. **Critical Logs:** Start-Logs prüfen (keine Panics, keine failed migrations, "listening"-Marker vorhanden)

**Host-Logs einsehen:**
```bash
# Auf csb1:
docker logs hausv-org --since 10m
```

## Rollback-Prozedur

### Image-Rollback (keine Schema-Änderungen)

**Wann:** Wenn das Release KEINE Migrations-Änderungen enthält.

**Automatischer Rollback-Command:** Der Deploy-Skript gibt am Ende einen Rollback-Command aus.

**Manueller Rollback:**

```bash
# Auf csb1, unter Compose-Lock:
flock -w 300 /run/lock/compose-hausv.lock /bin/sh -eu

# Dann im Lock:
docker tag ghcr.io/inspr-at/hausv-org:prev-<prev-version>-<prev-commit> ghcr.io/inspr-at/hausv-org:latest
docker compose --project-directory /home/mba/Code/hausv-jhw22 -p hausv-jhw22 -f /home/mba/Code/hausv-jhw22/compose.yml up -d --force-recreate --no-deps hausv-org

# Health prüfen bevor Lock freigegeben wird
curl -fsS https://hausv.org/healthz
```

**Verifikation:** Wie Production-Smoke-Test oben.

### Schema-Rollback (mit Migrations-Änderungen)

**Wann:** Wenn das Release `internal/db/migrations` geändert hat und der Deploy fehlgeschlagen ist oder das Release zurückgerollt werden muss.

**WARNUNG:** Kein Image-only Rollback gegen eine bereits migrierte Datenbank! Die alte Version würde gegen das neue Schema laufen und könnte Daten korrumpieren.

**Rollback-Pfad:**

1. **Lock öffnen:**
   ```bash
   ssh -tt -p 2222 mba@csb1 \
     /run/current-system/sw/bin/flock -w 300 /run/lock/compose-hausv.lock /bin/sh -eu
   ```

2. **Service stoppen (im Lock):**
   ```bash
   docker compose --project-directory /home/mba/Code/hausv-jhw22 \
     -p hausv-jhw22 \
     -f /home/mba/Code/hausv-jhw22/compose.yml \
     stop -t 30 hausv-org
   ```

3. **Failed Data sichern:**
   ```bash
   # Live Data-Dir als timestamped Recovery-Copy sichern
   sudo cp -a /var/lib/csb1-docker/hausv-org /var/backups/hausv-failed-$(date -Iseconds)
   ```

4. **Snapshot verifizieren:**
   - Snapshot-Pfad: `/var/backups/hausv-predeploy/<version>-<commit>/`
   - Snapshot in separates Restore-Check-Directory kopieren
   - `PRAGMA quick_check` auf der wiederhergestellten SQLite-DB ausführen
   - Benötigte Blob-Files prüfen

5. **Restore (im Lock):**
   ```bash
   # Beispiel (Pfade anpassen):
   sudo rm -rf /var/lib/csb1-docker/hausv-org
   sudo cp -a /var/backups/hausv-predeploy/<version>-<commit>/ /var/lib/csb1-docker/hausv-org
   sudo chown -R 65532:65532 /var/lib/csb1-docker/hausv-org
   ```

6. **Image Rollback + Service recreate (im Lock):**
   ```bash
   docker tag ghcr.io/inspr-at/hausv-org:prev-<prev-version>-<prev-commit> ghcr.io/inspr-at/hausv-org:latest
   docker compose --project-directory /home/mba/Code/hausv-jhw22 \
     -p hausv-jhw22 \
     -f /home/mba/Code/hausv-jhw22/compose.yml \
     up -d --force-recreate --no-deps hausv-org
   ```

7. **Verifikation (im Lock):**
   - `/healthz` prüfen
   - Sichtbare Version prüfen
   - Application-Logs prüfen
   - Als User einloggen und Daten prüfen

8. **Lock freigeben:** Exit aus dem Lock-Shell

**Detaillierte Schema-Rollback-Prozedur:** `hausv-org/docs/production-deploy.md` → "Schema rollback procedure"

## Host-Konfiguration

**Host:** csb1 (NixOS)  
**SSH:** `mba@csb1` Port 2222 (nicht 22!)  
**Compose-Directory:** `/home/mba/Code/hausv-jhw22`  
**Compose-File:** `/home/mba/Code/hausv-jhw22/compose.yml`  
**Compose-Project:** `hausv-jhw22`  
**Container-Name:** `hausv-org`  
**Compose-Lock:** `/run/lock/compose-hausv.lock` (0660 root:users, shared mit Backup-Timer)  
**Data-Directory:** `/var/lib/csb1-docker/hausv-org`  
**Snapshot-Root:** `/var/backups/hausv-predeploy`  

**GHCR-Auth:**
- Token-File: `/run/agenix/csb1-hausv-ghcr-pull` (0440 root:users, agenix-managed)
- Username: `x-access-token`
- Image: `ghcr.io/inspr-at/hausv-org:latest` und `:release-<version>-<commit>`

**NixOS-spezifische Binaries:**
- `flock`: `/run/current-system/sw/bin/flock`
- `base64`: `/run/current-system/sw/bin/base64`
- `mktemp`: `/run/current-system/sw/bin/mktemp`

**WICHTIG:** Niemals `docker compose down` auf dem gesamten `hausv-jhw22` Projekt! Der Runner recreated nur den `hausv-org` Service mit `--force-recreate --no-deps hausv-org`.

## Weitere Dokumentation

- **Vollständige Deployment-Dokumentation:** `hausv-org/docs/production-deploy.md`
- **Host-spezifische Runbook:** `nixcfg` → `hosts/csb1/docs/RUNBOOK.md` (für Host-Operationen, nicht Application-Deployment)
- **Private Instanz-Konfiguration:** `hausv-jhw22` Repository (nicht öffentlich)
- **Deployment-Umgebung:** `~/.config/hausv/deploy.env` (für attended Mac path)

## Notizen

- **Keine Secrets in diesem Dokument:** Tokens, Agenix-Keys, Tenant-Daten bleiben in der privaten Infrastruktur
- **VERSION-Bumps:** Patch für Fixes, Minor für Features, Major für Breaking Changes
- **CHANGELOG-Sprache:** Deutsch, positive Formulierung ("Stabilität verbessert" statt "Bug behoben")
- **Test vor Deploy:** `VERSION` bump VOR dem finalen Test-Run, sonst schlägt der Test fehl
- **Kein Force-Push:** Deployment-Skript refused wenn `HEAD` nicht exakt `origin/main` ist
- **CI-Requirement:** Nur Blacksmith-Runs zählen; GitHub-hosted Runners werden abgelehnt

---

**Letzte Verifikation:**  
Version 0.99.9, Commit 04e6801, Deploy-Run 32447870655, erfolgreich deployed am 21. August 2026.

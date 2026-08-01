# csb1 deployment notes

The csb1 stack expects the production env at:

```fish
/run/agenix/csb1-hausv-org-env
```

Create the encrypted source file from `nixcfg`:

```fish
cd ~/Code/nixcfg
agenix -e secrets/csb1-hausv-org-env.age
```

Expected keys:

```text
SESSION_KEY=<base64url-or-long-random-string>
ADMIN_EMAILS=<comma-separated-admin-emails>
INVITE_EMAILS=<comma-separated-allowed-emails>
WEG_TENANTS_JSON=<json-array>
WEG_USERS_JSON=<json-array>
HA_TOKEN=<home-assistant-long-lived-access-token>
SMTP_USER=resend
SMTP_PASS=<resend-api-key>
OIDC_ISSUER=https://auth.inspr.at
OIDC_CLIENT_ID=<zitadel-web-app-client-id>
OIDC_PROVIDER_NAME=Zitadel
SESSION_TTL=720h
SERVICE_PROVIDER_ACCESS_ENABLED=false
# Only set together after the documented operator self-assessment:
# SERVICE_PROVIDER_ASSESSMENT_VERSION=2026-07-26
TELEGRAM_BOT_TOKEN=<botfather-token-for-the-hausv-bot>
```

Jeder Eintrag in `WEG_TENANTS_JSON` kann den festen OpenStreetMap-Ausschnitt der
Portalnavigation ohne Laufzeit-Geocoding exakt verorten:

```json
{
  "slug": "jhw22",
  "name": "Hausportal",
  "address": "Janischhofweg 22, 8043 Graz",
  "portal_type": "community",
  "host": "jhw22.hausv.org",
  "brand_icon": "community",
  "map_latitude": 47.1008592,
  "map_longitude": 15.4717681,
  "map_zoom": 17
}
```

`portal_type` beschreibt unabhängig vom Energieprofil die Portalform:
`community` für eine Hausgemeinschaft, `apartment` für eine einzelne Wohnung
oder `house` für ein privates Haus. JHW22 bleibt `community`, auch wenn „Mein
Zuhause“ darin als Wohnung eingerichtet ist.

`brand_icon` bleibt über die Gebäudeeinstellungen änderbar und erscheint im
Karten-Pin. Für neue Häuser werden Breiten- und Längengrad deklarativ gesetzt;
`map_zoom` ist aus Sicherheits- und Darstellungsgründen auf 15 bis 18 begrenzt.
Ohne gültige Koordinaten zeigt das Portal eine neutrale Kartenfläche und
verlinkt die konfigurierte Adresse weiterhin zu OpenStreetMap.
`MAP_TILE_BASE_URL` ist ein optionaler, nicht geheimer Upstream-Override für
Tests oder einen eigenen Tile-Proxy; ohne Wert verwendet Produktion
`https://tile.openstreetmap.org`. Die Browser-QA setzt ihn zwingend auf ihre
lokale PNG-Fixture.

`TELEGRAM_BOT_TOKEN` aktiviert den hausv-eigenen Telegram-Bot (Ladesteuerung
Parkplatz 20: Befehle + Benachrichtigungen). Ohne den Schlüssel bleibt der Bot
vollständig deaktiviert; die Ladesteuerung selbst läuft unabhängig davon.
Nicht-geheime Ladeparameter (`CHARGING_*`-Entities, Intervalle,
`TELEGRAM_DATA_PATH`) stehen im Compose-File in nixcfg, nicht in der env.

`SERVICE_PROVIDER_ACCESS_ENABLED` ist die technische Freigabesperre für externe
Dienstleister. Sie bleibt `false`, bis die versionierte Betreiber-Selbstprüfung
und Betreiberentscheidung in HAUSV-86 dokumentiert abgeschlossen sind. Selbst
bei `true` bleibt der Zugang geschlossen, solange
`SERVICE_PROVIDER_ASSESSMENT_VERSION` nicht exakt der vom Build verlangten
Prüfversion entspricht. Dadurch kann eine alte Freigabe eine geänderte
Datenschutzbewertung nicht stillschweigend weiterverwenden.

For Zitadel Web apps using PKCE, `OIDC_CLIENT_SECRET` is not needed. Only set
`OIDC_CLIENT_SECRET` if Zitadel creates a confidential client that explicitly
requires one.

Production mail is sent directly through Resend SMTP. Verify `notify.hausv.org` in
Resend first, add the DNS records it gives you in Cloudflare, then store the
API key only in agenix as `SMTP_PASS`. The production sender is:

```text
hausv.org <noreply@notify.hausv.org>
```

HAUSV limits SMTP connection setup to five seconds and the complete exchange
after connection to fifteen seconds. During shutdown, the login-mail queue gets
five seconds to drain and one further second to cancel. The declarative csb1
Compose service must therefore keep `stop_grace_period: 30s`: 15 seconds for
HTTP, 5+1 seconds for mail, and 9 seconds of host margin. After a recreate,
inspect only this non-secret lifecycle field:

```bash
docker inspect --format '{{.Config.StopTimeout}}' hausv-org
```

The expected value is `30`; do not print secret or environment values for this
check.

The service-provider access gate must stay closed until the controller has
approved the concrete Art. 28/TOM package in
[`jhw22-art28-tom-approval.md`](jhw22-art28-tom-approval.md). After approval,
record its date and assessment revision in PPM, then set both
`SERVICE_PROVIDER_ACCESS_ENABLED=true` and the exact matching
`SERVICE_PROVIDER_ASSESSMENT_VERSION`. A single value, an outdated revision, or
missing approval remains a No-Go.

## Zitadel SSO

Create a Web/OIDC application in Zitadel for the portal.

Use this redirect URI exactly:

```text
https://jhw22.hausv.org/auth/oidc/callback
```

Use this post-logout URI if Zitadel asks for one:

```text
https://jhw22.hausv.org/
```

The portal requests these scopes:

```text
openid email profile
```

Authentication and authorization are separate:

- Zitadel authenticates the person and provides a verified email address.
- The portal only logs the user in when that email is present in
  `WEG_USERS_JSON`, `ADMIN_EMAILS`, or `INVITE_EMAILS` for the active tenant.
- Per-user login methods can be restricted in `WEG_USERS_JSON` via
  `auth_methods`: `["email"]`, `["oidc"]`, or `["email","oidc"]`.

Example user snippet:

```json
{
  "email": "person@example.invalid",
  "title": "Dr.",
  "first_name": "Jörg",
  "last_name": "Lehner",
  "role": "resident",
  "tenants": ["jhw22"],
  "permissions": ["parking"],
  "auth_methods": ["oidc"]
}
```

For multi-building accounts, `tenant_memberships` can override role and
permissions per tenant while the top-level values remain the default:

```json
{
  "email": "person@example.invalid",
  "role": "resident",
  "tenants": ["jhw22", "haus-b"],
  "tenant_memberships": {
    "jhw22": {"role": "manager", "permissions": ["parking"]},
    "haus-b": {"role": "resident", "permissions": []}
  }
}
```

The compose service bind-mounts all durable state from:

```text
/var/lib/csb1-docker/hausv-org
```

## Durable data layout

Relational application data is authoritative in `/data/hausv.db`. The path is
derived from `PARKING_DATA_PATH` and can be overridden with `DB_PATH`. SQLite is
required: an unavailable database aborts startup instead of silently falling
back to stale JSON.

The database contains identities and house memberships, login activity,
preferences, tenants and units, payment-status markers, contacts,
announcements/read state, events, handovers, ballots, issues/comments/history,
Telegram state, home-energy profiles/assets/entity mappings, retained
Smart-Meter source files and normalized quarter-hour values, tariff snapshots,
maintenance/measure metadata, and document/attachment **metadata**. Schema
migrations run transactionally and idempotently at boot.

Files remain outside SQLite:

| Path | Purpose |
|---|---|
| `/data/documents/` | private document versions and generated protocol PDFs |
| `/data/attachments/` | private uploads, previews and thumbnails |
| `/data/tenant-heroes/` | uploaded house imagery |
| `/data/audit.jsonl` + archives | append-only audit; rotation by count, 10 MiB or 90 days; archives max. three years |
| `/data/parking.json` | parking/charging time series, tariffs, paid flags and controller state |

Downloads always use authenticated, object-scoped routes. Stored filenames are
server-generated and files use restrictive modes. Superseded document versions
are retained deliberately as house-document history. Deleted attachment files
are removed immediately and their tombstones after one year.

Energy retention is enforced at startup and every six hours while the process
is running:

- uploaded Smart-Meter originals: maximum 30 days;
- normalized quarter-hour values: maximum 13 months;
- persisted tariff assessments: maximum three years.

Selected Home Assistant live/history values are fetched transiently for the
cockpit and are not mirrored into SQLite as individual states. Confirmed entity
mappings are stored. When grid import is confirmed, the read-only sampler also
stores one completed quarter-hour average with source and quality; those
derived intervals follow the same 13-month retention as imported quarter-hour
values. Home Assistant URLs and tokens stay in agenix/host configuration and
are excluded from the application database and energy export. Deleting an
energy profile removes its energy data and delegated energy grants, while a
non-personal empty marker prevents declarative profile seeds from resurrecting
deleted household data. The commercial free-period start remains separate
contract metadata so re-onboarding cannot restart it.

The old `*_DATA_PATH` JSON values may still be present in nixcfg so a fresh
database can perform the idempotent historical import. They are not runtime
backends after the SQLite cutover. Frozen JSON files are rollback evidence, not
a second source of truth.

Env-config users (`WEG_USERS_JSON`, `ADMIN_EMAILS`, `INVITE_EMAILS`) remain
authoritative over persisted identity data and cannot be escalated by an old
invite record.

Historical accounting backfill uses Home Assistant recorder statistics from
`PARKING_HISTORY_START` onward. For the 2026 rollout this is set in compose as:

```text
PARKING_HISTORY_START=2026-01-01
```

Generate a session key without printing it to the terminal:

```fish
openssl rand -base64 48 | tr '+/' '-_' | tr -d '=' | pbcopy
```

After the secret exists, rebuild or switch csb1 so agenix materializes
`/run/agenix/csb1-hausv-org-env`.

## Guarded production release

Die Compose-Spezifikation wird seit OPS-127 deklarativ aus nixcfg gerendert.
Zur Laufzeit ist ausschließlich
`/etc/compose/csb1/docker-compose.yml` maßgeblich; im Git-Checkout liegt bewusst
keine zweite YAML-Kopie. Jeder schreibende Aufruf für das Projekt `csb1` wird
über `/run/lock/compose-csb1.lock` serialisiert, damit Deployment, deklarativer
Reconcile und andere Betriebsjobs keine parallelen Containerübergänge starten.

Before a release, run the deterministic local release fixtures:

```fish
cd ~/Code/hausv-org
scripts/test-deploy.sh
```

The fixtures replace `git`, `gh`, `curl`, `ssh`, and `sleep` with local command
doubles. They cover the successful dry-run, dirty/stale Git state, exact CI
query contract, missing or wrong CI, non-Blacksmith/reusable jobs, unreadable or
unhealthy production, running-image/tag mismatches, retries after partial
activation, snapshot/recovery failures, build/activation failures, post-deploy
health/version/log failures, and both schema/non-schema success. They cannot
contact or mutate production. A dedicated fixture also proves that an early
remote failure cannot be hidden by a later successful command. The same test
creates local SQLite/blob recovery points, proves their integrity and root-only
mode, and proves that neither a later release nor a same-release retry
overwrites an existing point.

Commit and push the release. Then run the read-only preflight before the actual
release:

```fish
scripts/deploy.sh --dry-run
scripts/deploy.sh
```

Both commands fail closed unless all of these statements are true:

- the worktree is clean and `HEAD` is exactly the freshly fetched
  `origin/main`;
- every job in the committed `CI` workflow declares a Blacksmith runner;
  reusable-workflow jobs are rejected because this script cannot prove their
  runner contract;
- a completed successful `CI` push run exists for that exact full commit SHA;
- the current live version/commit is readable and `/healthz` reports HAUSV
  healthy;
- the live commit exists locally, so migration changes can be compared with
  `HEAD`;
- csb1 reports the current container healthy, proves that its exact image ID is
  the image currently named by `:latest`, has the expected Compose service, and
  (for a schema release) non-interactive system-Python capability;
- the rendered runtime stack exists at
  `/etc/compose/csb1/docker-compose.yml`, resolves `hausv-org` in project
  `csb1`, and can be read while holding the shared Compose lock;
- an already existing rollback tag either names that exact running image or the
  release is refused. It is never silently retargeted.

`--dry-run` stops after those read-only gates. It does not create a snapshot,
tag or build an image, or recreate a container.

The actual release ships only `git archive HEAD`, builds a release-specific
image on csb1, preserves the current image under an immutable rollback tag, and
only then retags/recreates the Compose service from the rendered runtime stack
while holding the shared project lock. It never pushes an image to GHCR. A
retry after a partial activation is therefore rejected while
`:latest`, the running container and the immutable rollback tag disagree.
Because csb1 uses fish as its SSH login shell, every compound remote operation
is explicitly executed through `/bin/sh -eu -c`. The printed recovery,
containment, and image-rollback commands use the same fail-fast wrapper, so a
failed first command cannot be masked by a successful final command.

### Mandatory pre-schema recovery point

When any file under `internal/db/migrations` differs between the visible live
commit and `HEAD`, deployment cannot continue without a fresh recovery point.
The helper:

1. verifies that the current container is healthy;
2. stops only `hausv-org`;
3. copies the complete durable data tree (including private blobs);
4. creates `hausv.db` through Python
   `sqlite3.Connection.backup` and requires `PRAGMA integrity_check = ok`;
5. writes value-free source/target build metadata;
6. atomically publishes the directory, restarts HAUSV, and requires it to
   become healthy again.

The root-only base is:

```text
/var/lib/csb1-docker/hausv-org-predeploy/
```

Each release receives its own `0700` recovery directory:

```text
/var/lib/csb1-docker/hausv-org-predeploy/<target-semver>-<target-7-char-commit>/
```

The script validates the exact path, timestamp, SQLite-integrity proof, and
restarted service health before it preserves or builds an image. Recovery paths
are write-once: neither a later version nor a retry of the same target may
replace an existing point. After a failed attempt, keep its recovery directory
and prepare a newly versioned, committed, green release before retrying. This
local pre-deploy point is a fast release rollback boundary, not a replacement
for the daily quiesced/offsite backup.

### Hard post-deploy checks

A release is only successful when all four checks pass:

- Docker reports the new container healthy;
- public `/healthz` identifies `hausv-org` with status `ok`;
- the rendered app shows the exact target semver and seven-character commit;
- logs from the current container start contain the structured `listening`
  marker and none of the defined fatal, migration, SQLite, atomicity, or purge
  failures.

Any precondition or postcondition exits non-zero. A snapshot failure explicitly
rechecks current container health: when healthy, it confirms that no new image
or schema was activated; when unhealthy, it reports affected availability and
the mandatory recovery command. After activation, failures print schema-aware
containment and rollback guidance.

### Recurring operations alarm

The release checks above cover the deployment itself. Between releases,
`hausv-alerts.timer` on csb1 checks the daily snapshot timer and service result,
the timestamp of the latest coherent recovery point, the existing sanitized
Restic status, the container, the exact public `/healthz` contract and
privacy-safe categories of new critical application logs. Snapshot and Restic
success may be at most 30 hours old. The Restic signal remains owned by the
Pharos backup observation; the HAUSV watcher does not run a second backup or
retention process.

Alerts and recoveries are transition-based and use the existing declarative
csb1 operator channel. Raw log values, resident or object identifiers, URLs,
tokens and recipient identifiers are never copied into alert state or text.
The executable configuration and operator commands live in nixcfg under
`hosts/csb1`; the canonical operational procedure is the
`HAUSV Snapshot, Health And Application Alerts` section of its host runbook.

The visible app version is `SEMVER (git-hash)`. Semver is sourced from
`VERSION`; bump it before every production deployment and keep
`docs/CHANGELOG.md` in German, newest entry first.

### Git release tags

Git source tags are immutable, post-deployment evidence. Only after
`scripts/deploy.sh` has completed every production check, create an annotated
`v<VERSION>` tag on the exact deployed commit and push that tag. Never tag a
failed candidate, move or reuse an existing release tag, or backfill a
historical gap without reliable deployment evidence. Git tags are
authoritative; a GitHub Release is optional, and neither one triggers a
deployment.

## Rollback boundaries

The release output is authoritative for the exact previous image tag and, for
a schema release, the exact matching recovery directory.

| Release type | Safe rollback |
|---|---|
| No migration change | Run the printed image rollback command. No data restore is required. |
| Migration change | Open the printed locked recovery shell and keep it open through containment, verified data restore and image recreation. Do **not** run an old image against a possibly migrated database. |

### Schema rollback procedure

This procedure is self-contained and uses tools that are present on csb1.
There is no dependency on a `sqlite3` CLI.

1. Run the **locked schema recovery shell** command printed by the failed
   release. It opens one attended shell while holding
   `/run/lock/compose-csb1.lock`. Keep that shell open until the old image and
   matching data are healthy again; do not start a second Compose command in
   another terminal.
2. In that locked shell, set the following three values from the same release
   output.
   `snapshot_dir` must be the exact versioned recovery point, `restore_check`
   must be a new path, and `failed_live` must not already exist:

```bash
snapshot_dir=/var/lib/csb1-docker/hausv-org-predeploy/0.58.0-aaaaaaa
restore_check=/var/lib/csb1-docker/hausv-org-restore-check-0.58.0-aaaaaaa
failed_live=/var/lib/csb1-docker/hausv-org-failed-20260730T120000Z
```

The values above are examples only. Stop if the release output names a
different version/commit, or if either destination already exists.

3. Validate paths, make an isolated complete copy, and verify its SQLite
   database with system Python:

```bash
sudo test -d "$snapshot_dir"
sudo test ! -e "$restore_check"
sudo test ! -e "$failed_live"
case "$snapshot_dir" in
  /var/lib/csb1-docker/hausv-org-predeploy/*) ;;
  *) echo "unexpected snapshot path" >&2; exit 1 ;;
esac
sudo install -d -m 0700 "$restore_check"
sudo cp -a "$snapshot_dir"/. "$restore_check"/
sudo /run/current-system/sw/bin/python3 - "$restore_check/hausv.db" <<'PY'
import sqlite3
import sys

database = sqlite3.connect(f"file:{sys.argv[1]}?mode=ro", uri=True)
result = database.execute("PRAGMA integrity_check").fetchone()
print(result[0] if result else "missing result")
raise SystemExit(0 if result == ("ok",) else 1)
PY
```

The expected value-free output is only `ok`. Any other output or non-zero exit
stops the rollback; the live directory has not been touched.

4. Still inside the same locked shell, run the exact **inside that same locked
   shell, containment command** printed by the release. It uses the canonical
   rendered stack without trying to acquire the already-held lock. Then
   atomically preserve the failed live tree and install the checked copy:

```bash
# Paste the exact printed containment command here. Its shape is:
docker compose --project-directory /home/mba/Code/nixcfg/hosts/csb1/docker \
  -p csb1 -f /etc/compose/csb1/docker-compose.yml stop -t 30 hausv-org
sudo test -d /var/lib/csb1-docker/hausv-org
sudo test ! -e "$failed_live"
sudo mv /var/lib/csb1-docker/hausv-org "$failed_live"
sudo mv "$restore_check" /var/lib/csb1-docker/hausv-org
sudo chown -R 65532:65532 /var/lib/csb1-docker/hausv-org
```

The original versioned snapshot remains untouched. The failed data also
remains preserved at the explicit `failed_live` path.

5. Without leaving the locked shell, run the exact **inside that same locked
   shell after the matching data restore** command printed by the release. It
   retags the preserved image and recreates `hausv-org` from the rendered
   stack. Require all of the following before leaving the shell and declaring
   recovery complete:

```bash
docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' hausv-org
curl -fsS https://jhw22.hausv.org/healthz
```

Docker must print `healthy`; the endpoint must return
`{"service":"hausv-org","status":"ok"}`. Confirm the visible version/commit is
the previous build named by the rollback output. Do not remove `failed_live` or
the versioned snapshot during the incident.

For disaster/offsite recovery, the checked canonical source remains the
quiesced snapshot:

```text
/var/lib/csb1-docker/hausv-org-backup-snapshot
```

Its restic retrieval is documented in
`nixcfg/hosts/csb1/docs/RUNBOOK.md`, section **HAUSV Snapshot And Restore**.
After retrieval, use the system-Python integrity check and preserved-directory
procedure above. Image rollback and data/schema restore are separate
operations; neither one substitutes for the other after a migration.

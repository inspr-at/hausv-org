# Production deployment

## Architecture

**Product repository:** This repository (`hausv-org`) contains the HAUSV product
code, deployment scripts, and versioned documentation.

**Private instance repository:** The private repository `hausv-jhw22` contains the
compose configuration and instance-specific settings for the csb1 production
deployment. It lives at `/home/mba/Code/hausv-jhw22` on csb1. The normal entry
point from that repository is `scripts/deploy-product.sh ../hausv-org`, which
calls the fail-closed product deploy scripts in this repository.

**PPM runbook:** The operator index and deployment checklist for csb1 lives in PPM
Knowledge as runbook `csb1-deploy-rollback`. This document contains the detailed
commands and technical reference.

Production infrastructure hostnames, paths and operator details are supplied
through a private deployment environment. Do not commit a populated environment
file.

## Release contract

Every production release must:

1. bump `VERSION` and update `docs/CHANGELOG.md` plus `Notes()`;
2. be committed and pushed to `origin/main`;
3. have a completed green `CI` run on Blacksmith for the exact commit;
4. pass the read-only deployment preflight;
5. be the CI image that green run pushed to GHCR — both paths pull `release-<version>-<short sha>`, neither builds on the production host;
6. expose the expected version and commit after activation.

The script preserves the previous image. When migrations changed (in either
series, `internal/db/migrations` or `internal/db/postgres/migrations`), both
paths create one recovery point before the swap: a quiesced SQLite and blob
snapshot plus, because production runs PostgreSQL, a custom-format `pg_dump`
of the production database taken in the same stopped-service window — the
Mac-less path inside the green image under the project lock, the attended Mac
path on the host. A failed snapshot or dump refuses the release; production
stays on the previous image (HAUSV-562).

A merge to `main` that does not bump `VERSION` still triggers `Deploy`. The
unattended script reads the live build, finds this version already there and
ends with `nothing to release: VERSION <x> is already live` and exit code 3;
the workflow turns that into a green run whose summary says "Nothing to
release". Nothing is pulled, nothing is swapped, production keeps serving its
image.

The distinction is deliberate: a red `Deploy` run always means a release was
refused or failed, so it is worth reading. The check only ends the run when the
live build was actually read and reports exactly the candidate version — an
unreachable or unparsable live build still refuses through the fail-closed path,
and a release whose deploy failed is still picked up by the next merge, because
the live version then differs from `VERSION`. `scripts/deploy.sh`, the attended
path, keeps refusing an unchanged version with an error: there an operator asked
for a release explicitly.

The preflight proves the running container by its image: normally the image
`:latest` names. When `:latest` has gone missing on the host (it did once on
csb1, HAUSV-634), the live release's own tag `release-<live version>-<live
commit>` is the second witness — the running image must be exactly that image,
otherwise the release is refused as before. The preserve step then puts
`:latest` back onto the running image under the project lock before anything
else changes, and the preflight prints a `note:` line so the operator sees the
tag was missing. Neither witness present means an unproven host: refuse.

## Deployment paths

### Mac-less automatic deployment (recommended for all releases)

After a green CI run on `main`, the `.github/workflows/deploy.yml` workflow
automatically triggers on a self-hosted runner (`csb1-hausv` label) on csb1.
The runner:

1. Checks out the repository at the exact green CI commit;
2. Checks whether migrations changed vs. the live version (both series);
3. Preserves the live image and pulls the CI-built image from GHCR without
   changing the running container;
4. If migrations changed, stops the service under the project lock, dumps the
   PostgreSQL database inside its own container, and runs that green image's
   restricted snapshot command with read-only data and a dedicated writable
   backup bind mount, streaming the dump into the same recovery point;
5. Swaps the production container under the project lock.

**Schema support:** When either migration series changes, the unattended
script captures the same recovery scope as `scripts/deploy.sh`: SQLite + blobs,
and the PostgreSQL dump named by `HAUSV_DEPLOY_POSTGRES_CONTAINER`. The runner
needs no sudo access: Docker performs the two explicit host bind mounts, the
snapshot container has no network, and the live service is automatically
recovered before the script can fail. Snapshot or dump failure refuses the
release before swapping; production stays on the previous image.

**PostgreSQL in the recovery point (HAUSV-562):** `pg_dump -U <user>
--format=custom` runs inside the database container over its local socket, so
no credential leaves the host; on csb1 the user is the read-only `BYPASSRLS`
backup role, never the superuser (HAUSV-635). The archive is listed with `pg_restore --list`
inside that container (the `TABLE DATA` count is the restorability witness),
then streamed byte-exact into the snapshot command, which verifies the
announced length and the `PGDMP` header and records the SHA-256 in
`PREDEPLOY-SNAPSHOT.json`. The result is `postgres.pgdump` (mode 600) next to
`hausv.db` in the versioned, write-once recovery point. The dump is taken while
`hausv-org` is stopped, so files and database describe one moment. A host that
runs SQLite only states that with `HAUSV_DEPLOY_POSTGRES_CONTAINER=none`; an
unset value refuses every schema release before any change.

The versioned published snapshot is write-once. An interrupted, unpublished
`.staging` directory for that same version is removed by the next snapshot
attempt. Snapshot recovery allows up to 100 seconds by default for the old
container to pass Docker health checks; override the positive attempt count
with `HAUSV_DEPLOY_SNAPSHOT_VERIFY_ATTEMPTS` when the shared two-second polling
interval is changed.

**No SSH secrets:** The runner runs locally on csb1 with host docker and GHCR
credentials from `/run/agenix/csb1-hausv-ghcr-pull`. No SSH keys or GitHub
secrets are involved.

**Manual invocation:** The workflow also supports `workflow_dispatch` with
explicit `version` and `commit` inputs for manual or roll-forward deploys.

**Concurrency:** One deploy at a time via `concurrency: deploy-production`.

### Attended Mac deployment (for manual verification or break-glass)

`scripts/deploy.sh` remains the attended path for manual verification or
break-glass operations. It:

1. Runs preflight checks including migration diff;
2. Creates a quiesced SQLite snapshot plus the PostgreSQL dump if migrations
   changed (the root-only helper `scripts/create-predeploy-snapshot.py` runs
   `pg_dump` inside the database container);
3. Pulls the CI image and swaps the container over SSH;
4. Provides rollback commands.

Use this path when manual verification is required or when the unattended runner
is unavailable. Schema releases no longer require this path: the unattended
script now takes the same snapshot.

## Private deployment environment

These are host paths, a container name and public URLs — **not credentials**.
Access is by your own SSH key, so this file can live in a normal config
location; `~/.config/hausv/deploy.env` with mode `600` is the convention.

The values below are the **actual csb1 production target**, not placeholders.
Every one of them was wrong in an earlier version of this document, and each
wrong value cost a failed deploy attempt to discover:

```sh
HAUSV_DEPLOY_SSH_HOST=mba@csb1
# 2222, not 22: deploy.sh passes -p explicitly, which OVERRIDES the port your
# ~/.ssh/config sets for the csb1 alias. Getting this wrong times out.
HAUSV_DEPLOY_SSH_PORT=2222
HAUSV_DEPLOY_COMPOSE_DIR=/home/mba/Code/hausv-jhw22
HAUSV_DEPLOY_COMPOSE_FILE=/home/mba/Code/hausv-jhw22/compose.yml
HAUSV_DEPLOY_COMPOSE_PROJECT=hausv-jhw22
HAUSV_DEPLOY_CONTAINER=hausv-org
# Shared with the backup snapshot timer, 0660 root:users.
HAUSV_DEPLOY_COMPOSE_LOCK=/run/lock/compose-hausv.lock
HAUSV_DEPLOY_DATA_DIR=/var/lib/csb1-docker/hausv-org
HAUSV_DEPLOY_SNAPSHOT_ROOT=/var/backups/hausv-predeploy
# Production runs PostgreSQL in this container; a schema release dumps this
# database into the recovery point. "none" would declare a SQLite-only host.
HAUSV_DEPLOY_POSTGRES_CONTAINER=hausv-postgres
HAUSV_DEPLOY_POSTGRES_DB=hausv
# The role pg_dump connects as over the container's local socket: the
# read-only backup role with BYPASSRLS (HAUSV-559), which sees every row
# despite FORCE ROW LEVEL SECURITY without being a superuser. The scripts
# default to postgres only for hosts that have no such role.
HAUSV_DEPLOY_POSTGRES_USER=hausv_backup
# Only for the printed rollback command: restoring drops, recreates and owns
# objects, which the read-only dump role may not. Default postgres.
HAUSV_DEPLOY_POSTGRES_RESTORE_USER=postgres
# The live tenant is jhw22. https://hausv.org/demo/ returns 404.
HAUSV_DEPLOY_LIVE_URL=https://hausv.org/jhw22/
HAUSV_DEPLOY_HEALTH_URL=https://hausv.org/healthz
```

If you are ever unsure of a value, read it off the running host rather than
guessing — `docker inspect hausv-org` carries the compose project, working
directory, config file and mounts.

`HAUSV_DEPLOY_CONTAINER` identifies the unique container created by the
configured Compose service. It may differ from the stable service name
`hausv-org`, which allows multiple path-routed instances on one host.

Optional overrides:

The image override remains optional. csb1 is NixOS: **there is no `/usr/bin`**,
so the three executable-path overrides are required there. Their defaults are
FHS paths and fail with `No such file or directory`.

```sh
HAUSV_DEPLOY_IMAGE=ghcr.io/inspr-at/hausv-org:latest
HAUSV_DEPLOY_FLOCK_BIN=/run/current-system/sw/bin/flock
HAUSV_DEPLOY_BASE64_BIN=/run/current-system/sw/bin/base64
HAUSV_DEPLOY_MKTEMP_BIN=/run/current-system/sw/bin/mktemp
```

## GHCR authentication

The `hausv-org` GHCR package is private and inherits repository visibility.
Production deployment pulls the CI-built image from `ghcr.io/inspr-at/hausv-org`
and requires authentication before every pull.

Both `deploy.sh` (remote SSH pull) and `deploy-from-ci-image.sh` (runs on csb1)
read a token from a file on the target host and fail closed if the file is
missing or unreadable.

Default token file: `/run/agenix/csb1-hausv-ghcr-pull` (override via
`HAUSV_DEPLOY_GHCR_TOKEN_FILE`).

Default username: `x-access-token` (override via `HAUSV_DEPLOY_GHCR_USER`).
This username is valid for GHCR personal access tokens and fine-grained tokens.

### Setup procedure

1. Create a fine-grained GitHub PAT with `read:packages` scope limited to
   `inspr-at/hausv-org` only.
2. Store the PAT in 1Password with a descriptive name (e.g.,
   `csb1-hausv-ghcr-pull`).
3. Encrypt the token as `csb1-hausv-ghcr-pull.age` in the agenix secrets
   repository (nixcfg).
4. Deploy the agenix secret to csb1 so it appears at
   `/run/agenix/csb1-hausv-ghcr-pull` with mode `0440 root:users`, readable by
   the deployment user (mba).

The token is passed to `docker login` via stdin in a subshell, preventing
accidental disclosure through environment variables, command history, or
process listings. Never print, commit, or summarize the token value.

Janus will later project this capability; do not add a reveal path or manual
token distribution outside the agenix pipeline.

The application runtime must separately configure its public URL, operator
details, tenant directory, mail or OIDC login and secrets. Tenant URLs use
`https://hausv.org/<tenant>/...`; tenant-specific DNS entries are not required.

## Run

### Mac-less automatic path (self-hosted runner on csb1)

The `.github/workflows/deploy.yml` workflow runs automatically after green CI on
`main`. It requires a self-hosted runner with the `csb1-hausv` label registered
on csb1.

**Runner setup:**

The repository runner is declared in nixcfg at
`hosts/csb1/hausv-github-runner.nix` and runs as mba. It must have:

1. `csb1-hausv` label;
2. Access to host docker socket;
3. GHCR authentication via `/run/agenix/csb1-hausv-ghcr-pull` token file;
4. Access to the compose lock and compose directory;
5. Docker socket access, with the HAUSV data and pre-deploy snapshot bind-source
   directories already present on the host. The runner process does not need
   direct filesystem access to those root-only directories; the PostgreSQL
   dump also travels through `docker exec` and a container stdin, never
   through a runner-readable file.

**Operator checklist for one-time runner setup:**

1. Apply the reviewed declarative nixcfg configuration through the normal host
   change process.
2. Verify `github-runner-csb1-hausv.service` is active and the runner appears in
   GitHub repository Settings → Actions → Runners.
3. **Critical**: Never run `docker compose down` on the entire csb1 hausv-jhw22
   project. The runner only recreates the `hausv-org` service with
   `--force-recreate --no-deps hausv-org`.

Do not print or commit the runner registration token. Retrieve it from the
GitHub repository settings when needed.

**Manual dispatch:**

To deploy a specific version and commit via the workflow:

1. Go to Actions → Deploy → Run workflow
2. Select branch `main`
3. Enter `version` (e.g., `0.98.10`) and `commit` (7-char SHA)
4. Click "Run workflow"

### Attended Mac path (for schema releases)

`set -a` is **bash**. In fish it is a different builtin and fails with
`expected >= 1 arguments; got 0`, so wrap the whole thing in one `bash -c`:

```fish
direnv exec . bash -c 'set -a; . ~/.config/hausv/deploy.env; set +a; bash scripts/deploy.sh --dry-run'
direnv exec . bash -c 'set -a; . ~/.config/hausv/deploy.env; set +a; bash scripts/deploy.sh'
```

From bash the original form works:

```sh
set -a
. ~/.config/hausv/deploy.env
set +a

scripts/deploy.sh --dry-run
scripts/deploy.sh
```

The dry run performs all read-only checks and stops before snapshot, build,
retag or Compose mutation.

## Schema rollback procedure

When the script reports a schema-changing failure, do not start an old image
against the migrated live database.

1. Open the locked recovery shell printed by `scripts/deploy.sh`.
2. Stop the `hausv-org` service while holding that lock.
3. Keep the failed live data directory as a timestamped recovery copy, and take
   a `pg_dump` of the migrated PostgreSQL database the same way (a
   post-failure copy; never overwrite the pre-deploy dump).
4. Copy the matching snapshot into a separate restore-check directory.
5. Run `PRAGMA quick_check` on the restored SQLite database and verify required
   blob files before replacing the live data directory.
6. Restore the PostgreSQL database from `postgres.pgdump` in that snapshot,
   with the service still stopped:
   `docker exec -i hausv-postgres pg_restore -U postgres --clean --if-exists
   --exit-on-error -d hausv < <snapshot>/postgres.pgdump`. `--clean` drops and
   recreates every object the archive knows, including `schema_migrations`, so
   the ledger no longer names the migration that failed. Then
   `docker exec hausv-postgres psql -U postgres -d hausv -c "select version
   from schema_migrations order by 1 desc limit 3"` must end at the last
   migration of the previous release.
7. Restore ownership expected by the container.
8. Recreate the service with the previous image command printed by the script.
9. Verify `/healthz`, the visible version and the application logs before
   releasing the lock.

Exact storage paths are intentionally absent from this repository. Use only the
paths loaded in the private deployment environment and the snapshot path printed
for the failed release.

## PostgreSQL cutover procedure

Everything the cutover needs is merged: every store runs on a tenant lane, migration
`0006` makes row-level security fail-closed with `NOT NULL` on `tenant_id`, and
`hausv-org migrate-data` moves the rows and proves the copy before committing. Nothing
below runs by itself. Steps marked **operator** need a secret or an authorisation an
agent must not hold; the rest can be rehearsed by anyone with the private deployment
environment.

The order matters. `0006` refuses on any orphan row and the mover refuses a non-empty
target, so each step is a gate for the next.

1. **Census.** On the *source*, confirm every tenant-scoped row already carries an
   identity: `SELECT count(*) FROM <table> WHERE tenant_id IS NULL` per governed table
   must be 0. (Measured 2026-08-19 on production: 25 tables, 1,633 rows, 0 unlinked.)
   Against the *target* after `0006` has been applied, `psql -f
   scripts/postgres-tenant-id-census.sql` must report every table `READY`
   (`home_reservations` `EXEMPT`). Any `BLOCKS-0006` row means stop.
2. **operator — application role.** In the PostgreSQL container, an application role
   that is `NOSUPERUSER NOBYPASSRLS` and owns an empty database. `verifyPostgresRole`
   refuses anything else at boot.
3. **operator — `DATABASE_URL`.** Provide it to the container through the private
   deployment environment (agenix), *without* setting `DB_BACKEND` yet. It is read
   from the environment only; there is no flag for it anywhere.
4. **operator — backup role.** `FORCE ROW LEVEL SECURITY` binds the table owner too, so
   `pg_dump` errors under the application role. Create a dedicated `hausv_backup`
   role with `BYPASSRLS`, read-only, for backups; never the superuser. Then perform
   one dump-and-restore drill under the flipped policy before relying on it.
5. **Snapshot.** Take the pre-cutover snapshot of the SQLite data directory exactly
   as `scripts/deploy.sh` does for a schema change; run `PRAGMA quick_check` on the
   copy; record its path. This is the rollback.
6. **Stop.** Hold the compose lock and stop `hausv-org` so the SQLite file is
   quiescent. The mover opens it read-only and refuses to guess at a moving file.
7. **Rehearse.** `docker compose run --rm hausv-org migrate-data --dry-run` — everything
   runs, including verification, then rolls back on purpose. Expect `ROLLED BACK`, the
   full table count, a source row total matching the census, and no refusal.
8. **Move.** The same command without `--dry-run`. Expect `COMMITTED: N rows in M
   tables`. Then `migrate-data --verify-only` must exit 0.
9. **Switch.** Set `DB_BACKEND=postgres`, recreate the service, verify `/healthz`, the
   visible version, and log in as each tenant. The first PostgreSQL boot runs the
   identity backfill and the idempotent imports; watch the log for any import doing
   real work — none should.
10. **Verify as a user, not a health check.** Open the pages a resident and a manager
    actually use on both a desktop and a phone. Row-level security is now fail-closed:
    a page that renders empty where SQLite showed data is a lane bug, and the
    Unscoped inventory (`internal/store/testdata/unscoped_calls.golden`) is where to
    look first.

**Rollback** before step 9 is trivial: the SQLite file was never modified. After step
9, roll back by switching `DB_BACKEND` back and recreating the service — the SQLite
data directory is exactly as it was, and `0006` never touched it. A PostgreSQL
database that has already applied `0006` keeps `NOT NULL` and fail-closed until the
`0003` policy is recreated by hand and `NOT NULL` dropped; a fresh database is easier.

## Runtime legal and privacy configuration

Public operators must provide accurate deployment-specific values outside the
repository:

```sh
HAUSV_OPERATOR_NAME=Example Operator GmbH
HAUSV_OPERATOR_ADDRESS=Example Street 1, 1010 Vienna, Austria
HAUSV_PRIMARY_APP_URL=https://hausv.org/demo/
HAUSV_PROFESSIONAL_SERVICES_NOTICE=Professional service notice
HAUSV_IDENTITY_STORAGE_NOTICE=Accurate identity and hosting disclosure
HAUSV_BACKUP_STORAGE_NOTICE=Accurate encrypted backup disclosure
HAUSV_WEB_ACCESS_NOTICE=Accurate web access and protection disclosure
HAUSV_MAIL_DELIVERY_NOTICE=Accurate transactional mail disclosure
```

These values are public legal disclosures, not secrets, but remain deployment
configuration because they differ between self-hosted and managed instances.

## Demo-Instanz

`deploy/demo/` is a self-contained Docker Compose bundle for a **non-production
demo**. It runs only the committed Musterstadt fixture data on port `8098` by
default; do not use it for real homes, people, documents, or production data.

From the repository root, prepare the two host-side configuration files and
start the demo:

```sh
cp deploy/demo/demo.env.example deploy/demo/demo.env
cp deploy/demo/secrets.env.example deploy/demo/secrets.env
chmod 600 deploy/demo/secrets.env
docker compose -f deploy/demo/docker-compose.yml up -d --build
deploy/demo/seed.sh
```

**Anmeldung auf einem öffentlichen Host.** Für die Demo-Instanz setzt `demo.env`
`DEMO_LOGIN_ENABLED=true`, und `secrets.env` trägt `DEMO_LOGIN_ACCESS_CODE`.
Der Demo-Modus hat immer Vorrang vor `LOCAL_DEV_LOGIN`, auch bei einer
localhost-`BASE_URL`; ohne richtigen Zugangscode wird daher nie ein Anmeldelink
erzeugt.
Das Anmeldeformular fragt dann zusätzlich nach dem Zugangscode und zeigt den
Anmeldelink direkt an; ein falscher Code entwertet den Link. Das ist eine
Vorführ-Hilfe für Fixture-Daten, keine Zugangskontrolle: den Host zusätzlich
per Reverse-Proxy (Basic Auth oder IP-Liste) schützen und den Code nach dem
Termin ändern.

**Demodaten aus dem Portal initialisieren (HAUSV-613, HAUSV-636).** Auf einer
Demo-Instanz (`DEMO_LOGIN_ENABLED=true` und `DEMO_SEED_DIR`, im Bundle `/seed`)
zeigt Einstellungen die Karte „Demodaten initialisieren“. Der Link öffnet einen
Bestätigungsdialog („Ja, initialisieren“ / „Abbrechen“); ohne JavaScript führt
er auf eine einstufige Bestätigungsseite. Die Bestätigung spielt Anliegen,
Organisationsdaten, Textbausteine, Ankündigungen und Termine aus dem Seed neu
ein (Anker heute); Benutzer und Sitzungen bleiben, der Audit-Eintrag heißt
„Demodaten initialisiert“. Ohne Demo-Modus antwortet die Route mit 404.

Der Seed enthält pro Haus die Tops samt Eigentümer- und Mieterzuordnung sowie Stellplätze.

`deploy/demo/seed.sh` can be rerun safely to restore the fixture records. For a
fresh database and blobs, run `deploy/demo/reset.sh`; it stops the demo, removes
only its named data volume, starts it again, and seeds it. Confirm the running
instance with `deploy/demo/verify.sh`.

Organisationsadministratoren können die laufende Demo außerdem ohne Abmeldung
unter `/<tenant>/app/verwaltung/einstellungen` über die Karte **Demodaten
initialisieren** neu einspielen. Der Portal-Weg ist ein einziger
Bestätigungsdialog; anschließend werden die gebündelten Anliegen,
Organisationsdaten, Ankündigungen und Termine mit dem heutigen Datum als Anker
neu eingespielt.

Auf dem Host bleiben die CLI-Wege für Wartung und vollständige Neuinitialisierung
verfügbar. Der Compose-Projektname muss dabei ausdrücklich der Demo gehören:

```sh
COMPOSE_PROJECT_NAME=hausv-demo HAUSV_DEMO_SEED_ANCHOR=today deploy/demo/seed.sh
COMPOSE_PROJECT_NAME=hausv-demo HAUSV_DEMO_SEED_ANCHOR=today deploy/demo/reset.sh
```

Put the OpenRouter API key only in `deploy/demo/secrets.env` as `AI_API_KEY`;
that file is ignored by Git and must remain mode `600` on the Docker host. Set a
separate, high-entropy `SESSION_KEY` there too. `demo.env` contains the committed
fixture configuration and defaults to OpenRouter's inexpensive structured-output
model.

For a reverse proxy, point the upstream at `http://127.0.0.1:8098` (or the
chosen `HAUSV_DEMO_PORT`) and set `BASE_URL` in `demo.env` to the public HTTPS
URL before exposing it. Never expose the localhost dev-login flow as a
production authentication mechanism.

### Demo auf agm1 (hausv.agm.ng)

The public demo runs on the Augmentoring host `agm1`. Ownership is split:

- `agm-nixcfg` (module `agm-hausv-demo`) owns the declarative half: the agenix
  file `agm1-hausv-demo-env` (`AI_API_KEY`, `SESSION_KEY`,
  `DEMO_LOGIN_ACCESS_CODE`, mode `0440 root:users`), the Caddy vhost, the
  registry entry in `hostnames.json` (DNS + aliases), `/srv/hausv-demo` and a
  boot unit that runs `up -d` for the bundle. Rotating the access code is a
  secret edit plus deploy there; the AGM-16 trigger restarts the bundle.
- This repository owns the application half. The image is private and CI
  never pushes it, so the code travels exactly like production: `git archive
  HEAD` to the host, built there, run from `deploy/demo/`. Ship with

  ```sh
  HAUSV_DEMO_SSH_HOST=mba@<ip> HAUSV_DEMO_SSH_PORT=2222 HAUSV_DEMO_SSH_KEY=~/.ssh/agm_deploy \
  HAUSV_DEMO_BASE_URL=https://hausv.agm.ng \
  HAUSV_DEMO_SECRETS_FILE=/run/agenix/agm1-hausv-demo-env \
  deploy/demo/deploy-remote.sh --seed
  ```

  It refuses a dirty tree, keeps every release in
  `/srv/hausv-demo/releases/<sha>`, points `src` at the live one, rewrites
  `demo.env` from the example with `BASE_URL`/`ROOT_DOMAIN` set, builds the
  image as `hausv-demo:<sha>`, waits for `/healthz`, seeds when asked and
  prints the rollback line. `HAUSV_DEMO_SECRETS_FILE` is the host-side path
  that replaces `./secrets.env` in the compose `env_file` list;
  `HAUSV_DEMO_SEED_ANCHOR=today` shifts the fixture dates to the deploy day.
  The compose project is `hausv-demo`; a manual `seed.sh`/`reset.sh` on the
  host needs `COMPOSE_PROJECT_NAME=hausv-demo` (or `HAUSV_DEMO_PROJECT`).
- No `VERSION` bump and no CI gate: the demo version is `<VERSION>-demo.<sha>`
  and the data is disposable.
- Proxy trust: with a public `BASE_URL` the app refuses to start without
  `TRUSTED_PROXY_CIDRS`. The bundle pins its compose subnet
  (`HAUSV_DEMO_SUBNET`, default `172.30.98.0/24`) so the proxy's source address
  inside the container is the gateway `172.30.98.1`; the deploy script writes
  that `/32` into `demo.env`. The Caddy vhost must send `X-Real-IP`, otherwise
  every visitor shares one login rate-limit bucket. The container port binds to
  `127.0.0.1` only; Docker-published ports bypass the host firewall.

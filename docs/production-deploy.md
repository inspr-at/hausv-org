# Production deployment

Production infrastructure is not part of this repository. Hostnames, paths and
operator details are supplied through a private deployment environment. Do not
commit a populated environment file.

## Release contract

Every production release must:

1. bump `VERSION` and update `docs/CHANGELOG.md` plus `Notes()`;
2. be committed and pushed to `origin/main`;
3. have a completed green `CI` run on Blacksmith for the exact commit;
4. pass the read-only deployment preflight;
5. be built from the CI image pushed to GHCR (Mac-less path) or from `git archive HEAD` on the configured host (attended Mac path);
6. expose the expected version and commit after activation.

The script preserves the previous image. If migrations changed, deployment is
refused in the Mac-less path; the attended Mac path creates a quiesced SQLite
and blob snapshot before building the new image.

## Deployment paths

### Mac-less automatic deployment (recommended for non-schema releases)

After a green CI run on `main`, the `.github/workflows/deploy.yml` workflow
automatically triggers on a self-hosted runner (`csb1-hausv` label) on csb1.
The runner:

1. Checks out the repository at the exact green CI commit;
2. Verifies migrations did not change vs. the live version;
3. Pulls the CI-built image from GHCR;
4. Swaps the production container under the project lock.

**Migration refuse:** If `internal/db/migrations` changed, deployment is refused
before any pull or swap. Schema releases still require the attended Mac path.

**No SSH secrets:** The runner runs locally on csb1 with host docker and GHCR
credentials from `/run/agenix/csb1-hausv-ghcr-pull`. No SSH keys or GitHub
secrets are involved.

**Manual invocation:** The workflow also supports `workflow_dispatch` with
explicit `version` and `commit` inputs for manual or roll-forward deploys.

**Concurrency:** One deploy at a time via `concurrency: deploy-production`.

### Attended Mac deployment (required for schema releases)

`scripts/deploy.sh` remains the attended path for schema-changing releases. It:

1. Runs preflight checks including migration diff;
2. Creates a quiesced SQLite snapshot if migrations changed;
3. Pulls the CI image and swaps the container over SSH;
4. Provides rollback commands.

Use this path when `internal/db/migrations` changes or when manual verification
is required.

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
# /run/lock is root-owned and the deploy user cannot create there. The per-user
# runtime dir is right: /tmp would let any process on the box hold the lock.
HAUSV_DEPLOY_COMPOSE_LOCK=/run/user/1000/hausv-compose.lock
HAUSV_DEPLOY_DATA_DIR=/var/lib/csb1-docker/hausv-org
HAUSV_DEPLOY_SNAPSHOT_ROOT=/var/backups/hausv-predeploy
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

csb1 is NixOS: **there is no `/usr/bin`**, so the three binary overrides are
required there, not optional. The defaults below are FHS paths and fail with
`No such file or directory`.

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

**Runner setup (already configured on csb1):**

1. Runner registered with `csb1-hausv` label;
2. Runner has access to host docker socket;
3. GHCR authentication via `/run/agenix/csb1-hausv-ghcr-pull` token file;
4. Runner user has access to the compose lock and compose directory.

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
3. Keep the failed live data directory as a timestamped recovery copy.
4. Copy the matching snapshot into a separate restore-check directory.
5. Run `PRAGMA quick_check` on the restored SQLite database and verify required
   blob files before replacing the live data directory.
6. Restore ownership expected by the container.
7. Recreate the service with the previous image command printed by the script.
8. Verify `/healthz`, the visible version and the application logs before
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

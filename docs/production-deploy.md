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
5. be built from `git archive HEAD` on the configured host;
6. expose the expected version and commit after activation.

The script preserves the previous image. If migrations changed, it also creates
a quiesced SQLite and blob snapshot before building the new image.

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

The application runtime must separately configure its public URL, operator
details, tenant directory, mail or OIDC login and secrets. Tenant URLs use
`https://hausv.org/<tenant>/...`; tenant-specific DNS entries are not required.

## Run

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

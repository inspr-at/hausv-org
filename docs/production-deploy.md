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

Load these values from a password manager, secret mount or untracked local file:

```sh
HAUSV_DEPLOY_SSH_HOST=deployer@host.example
HAUSV_DEPLOY_SSH_PORT=22
HAUSV_DEPLOY_COMPOSE_DIR=/srv/hausv/compose
HAUSV_DEPLOY_COMPOSE_FILE=/srv/hausv/compose/docker-compose.yml
HAUSV_DEPLOY_COMPOSE_PROJECT=hausv
HAUSV_DEPLOY_COMPOSE_LOCK=/run/lock/hausv-compose.lock
HAUSV_DEPLOY_DATA_DIR=/var/lib/hausv
HAUSV_DEPLOY_SNAPSHOT_ROOT=/var/backups/hausv-predeploy
HAUSV_DEPLOY_LIVE_URL=https://hausv.org/demo/
HAUSV_DEPLOY_HEALTH_URL=https://hausv.org/healthz
```

Optional overrides:

```sh
HAUSV_DEPLOY_IMAGE=ghcr.io/inspr-at/hausv-org:latest
HAUSV_DEPLOY_FLOCK_BIN=/usr/bin/flock
HAUSV_DEPLOY_BASE64_BIN=/usr/bin/base64
HAUSV_DEPLOY_MKTEMP_BIN=/usr/bin/mktemp
```

The application runtime must separately configure its public URL, operator
details, tenant directory, mail or OIDC login and secrets. Tenant URLs use
`https://hausv.org/<tenant>/...`; tenant-specific DNS entries are not required.

## Run

```sh
set -a
. /secure/path/hausv-deploy.env
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

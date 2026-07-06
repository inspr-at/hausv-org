# csb1 deployment notes

The csb1 stack expects the production env at:

```fish
/run/agenix/csb1-weg-portal-env
```

Create the encrypted source file from `nixcfg`:

```fish
cd ~/Code/nixcfg
agenix -e secrets/csb1-weg-portal-env.age
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
```

For Zitadel Web apps using PKCE, `OIDC_CLIENT_SECRET` is not needed. Only set
`OIDC_CLIENT_SECRET` if Zitadel creates a confidential client that explicitly
requires one.

Production mail is sent directly through Resend SMTP. Verify `notify.hausv.org` in
Resend first, add the DNS records it gives you in Cloudflare, then store the
API key only in agenix as `SMTP_PASS`. The production sender is:

```text
WEG Portal <noreply@notify.hausv.org>
```

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
  "email": "joerg.lehner@gmx.at",
  "title": "Dr.",
  "first_name": "Jörg",
  "last_name": "Lehner",
  "role": "resident",
  "tenants": ["jhw22"],
  "permissions": ["parking"],
  "auth_methods": ["oidc"]
}
```

Parking accounting is stateful. On csb1 the container writes monthly paid flags,
grid/base surcharge settings, meter readings, and aWATTar price readings to:

```text
/data/parking.json
```

The compose service bind-mounts that path from:

```text
/var/lib/csb1-docker/weg-portal
```

Invited users (Benutzer & Rechte -> "Person einladen") persist to `/data/invites.json`
via `INVITE_DATA_PATH` in compose, on the same bind-mount, so they survive redeploys.
Env-config users (`WEG_USERS_JSON`/`ADMIN_EMAILS`/`INVITE_EMAILS`) stay authoritative;
a stored invite can never override or escalate an env-defined user.

Login activity (last-login per email) persists to `/data/activity.json` via
`ACTIVITY_DATA_PATH`. The roster derives status from it (a user who has logged in
shows `Aktiv` + "zuletzt angemeldet: <date>"; invited-but-never-logged-in shows
"noch nie angemeldet"), so it stays consistent for both env and invited users
without mutating their records.

Aushang/Hausjournal entries persist to `/data/announcements.json` via
`ANNOUNCE_DATA_PATH`. The store uses the same JSON-store pattern as invites and
parking data: mutexed writes, atomic temp-file replacement, and file mode `0600`.
Per-user read state persists to `/data/announcement_reads.json` via
`ANNOUNCE_READ_DATA_PATH`, so "neu" badges survive redeploys. The dashboard shows
only published, unexpired entries; admin/Verwalter authoring controls live under
`/app/announcements`.

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
`/run/agenix/csb1-weg-portal-env`, then deploy the container:

```fish
cd ~/Code/weg-portal
set version (git describe --tags --match 'v[0-9]*' --abbrev=0 | string replace -r '^v' '')
test -n "$version"; or set version 0.1.0
set commit (git rev-parse --short HEAD)
git archive --format=tar HEAD | ssh -p 2222 mba@cs1.barta.cm "bash -lc 'set -euo pipefail; rm -rf /tmp/weg-portal-deploy; mkdir -p /tmp/weg-portal-deploy; tar -xf - -C /tmp/weg-portal-deploy; cd /tmp/weg-portal-deploy; docker build --build-arg APP_VERSION=$version --build-arg GIT_COMMIT=$commit -t ghcr.io/markus-barta/weg-portal:latest .; cd /home/mba/Code/nixcfg/hosts/csb1/docker; docker compose up -d --no-deps weg-portal'"
```

The visible app version is `SEMVER (git-hash)`. Semver is sourced from the latest
Git tag matching `v*`; the rollout starts at `v0.1.0`.

Smoke checks:

```fish
curl -fsS https://jhw22.hausv.org/healthz
curl -I https://jhw22.hausv.org/
```

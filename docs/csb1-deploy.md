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
Telegram state, and document/attachment **metadata**. Schema migrations run
transactionally and idempotently at boot.

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
`/run/agenix/csb1-hausv-org-env`. Commit the release, then use the guarded deploy
script:

```fish
cd ~/Code/hausv-org
scripts/deploy.fish --dry-run
scripts/deploy.fish
```

The script refuses dirty trees and an unchanged live version, ships exactly
`git archive HEAD`, builds on csb1, tags the previous image for rollback,
recreates the compose service, verifies the visible version and prints the exact
rollback command. It never pushes the image to GHCR.

The visible app version is `SEMVER (git-hash)`. Semver is sourced from
`VERSION`; bump it before every production deployment and keep
`docs/CHANGELOG.md` in German, newest entry first.

Smoke checks:

```fish
curl -fsS https://jhw22.hausv.org/healthz
curl -I https://jhw22.hausv.org/
```

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

Parking accounting is stateful. On csb1 the container writes monthly paid flags,
grid/base surcharge settings, meter readings, and aWATTar price readings to:

```text
/data/parking.json
```

The compose service bind-mounts that path from:

```text
/var/lib/csb1-docker/hausv-org
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

Termine/Kalender entries persist to `/data/events.json` via `EVENT_DATA_PATH`.
The dashboard agenda renders upcoming entries; past entries roll off the
resident view while staying editable for the Verwaltung.

Das Adressbuch persistiert pro Tenant in `/data/contacts.json` via
`CONTACT_DATA_PATH`. Es enthält Dienstleister, Hausmeister, Notdienste und andere
wiederkehrende Kontakte; deaktivierte Einträge bleiben für die Verwaltung sichtbar,
werden aber Bewohnern und Auswahlhilfen nicht angeboten.

Dokumente metadata persists to `/data/documents.json` via `DOC_DATA_PATH`; uploaded
files are stored under `/data/documents/` via `DOC_FILE_DIR`. The document store
uses private generated filenames, content-type/size validation, `0600` file modes,
and does not serve files from `/assets`. Downloads go through the authenticated
`/app/dokumente/{id}/download` route, which enforces per-document visibility and
records successful downloads in the audit log.

Shared app attachments persist metadata to `/data/attachments.json` via
`ATTACHMENT_DATA_PATH`; generated files and previews are stored below
`/data/attachments` via `ATTACHMENT_FILE_DIR`. This covers Aushänge, Termine,
Abstimmungen, Kommentare and other non-document attachment strips that should
survive container replacement.

Abstimmungen persist to `/data/votes.json` via `VOTE_DATA_PATH`. Ballots store
options, type, weighting, quorum, open/close timestamps and per-owner votes. The
store uses the same mutexed atomic JSON pattern and `0600` file mode; vote weights
come from the Wohneinheiten ownership links for Miteigentumsanteil voting.

Übergabeprotokolle persist to `/data/handovers.json` via `HANDOVER_DATA_PATH`.
Fotos und andere Anhänge nutzen den gemeinsamen `ATTACHMENT_DATA_PATH` /
`ATTACHMENT_FILE_DIR`-Store; abgelegte Protokoll-PDFs werden als private
Dokumente im bestehenden `DOC_DATA_PATH` / `DOC_FILE_DIR`-Store gespeichert.
Bestätigungslinks speichern nur Token-Hashes im JSON-Store, keine Klartext-Tokens.

Notification preferences persist to `/data/notification_prefs.json` via
`NOTIFICATION_PREF_DATA_PATH`. Missing preferences default to enabled delivery;
explicit event opt-outs and the global unsubscribe flag are checked before
non-authentication emails are sent.

Self-service profile display/contact overlays persist to
`/data/profile_overlays.json` via `PROFILE_DATA_PATH`. These overlays never carry
role, permission, tenant or auth-method fields; env/invite records remain
authoritative for authorization.

Gebäude-/Tenant-Einstellungen persist to `/data/tenant_overrides.json` via
`TENANT_DATA_PATH`; uploaded tenant hero images are stored below
`/data/tenant-heroes` via `TENANT_HERO_DIR`. Overrides are layered over
`WEG_TENANTS_JSON` defaults and carry display/contact/emergency/hero fields plus
the curated sidebar brand icon and short abbreviation. They never carry roles,
permissions, auth methods or secret-bearing configuration.
The `/app/kontakte` page reads these fields plus Beirat role assignments;
resident directory entries appear only after the user explicitly opts in from
their profile.

Wohneinheiten and ownership/renter links persist to `/data/units.json` via
`UNIT_DATA_PATH`. Each unit stores tenant, id, label, type, billable weight,
Miteigentumsanteil and owner/renter email links; the file uses the same mutexed
atomic JSON-store pattern and `0600` file mode. Missing legacy unit types are
normalized as billable Wohnungen. Stellplätze, Keller/Lager and sonstige
Einheiten can be tracked without counting toward fair-use Wohnungseinheiten.

Manueller Zahlungsstatus pro Einheit persists to
`/data/unit_payment_status.json` via `UNIT_PAYMENT_STATUS_DATA_PATH`. The status
is only a transparency marker (`offen`, `bezahlt`, `teilbezahlt`,
`ueberfaellig`) and does not create receivables, bookings, reminders or
accounting records. Residents only see explicit status records for units linked
to their own email address.

Anliegen submitted by residents persist to `/data/issues.json` via
`ISSUE_DATA_PATH`. Optional uploaded photos are validated as JPG/PNG/WebP up to
5 MB and written under `/data/issue-attachments/` via `ISSUE_ATTACHMENT_DIR`;
the JSON file and attachment files use mode `0600`.

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
`/run/agenix/csb1-hausv-org-env`, then deploy the container:

```fish
cd ~/Code/hausv-org
set version (string trim < VERSION)
test -n "$version"; or set version (git describe --tags --match 'v[0-9]*' --abbrev=0 | string replace -r '^v' '')
test -n "$version"; or set version 0.8.0
set commit (git rev-parse --short HEAD)
git diff --quiet; or set commit "$commit-dirty"
git ls-files -co --exclude-standard -z | tar --null -T - -cf - | ssh -p 2222 mba@cs1.barta.cm "bash -lc 'set -euo pipefail; tmpdir=\$(mktemp -d /tmp/hausv-org-deploy.XXXXXX); trap \"rm -rf \\\"\$tmpdir\\\"\" EXIT; tar -xf - -C \"\$tmpdir\"; cd \"\$tmpdir\"; docker build --build-arg APP_VERSION=$version --build-arg GIT_COMMIT=$commit -t ghcr.io/markus-barta/hausv-org:latest .; cd /home/mba/Code/nixcfg/hosts/csb1/docker; docker compose up -d --no-deps hausv-org'"
```

The visible app version is `SEMVER (git-hash)`. Semver is sourced from
`VERSION`; bump it before every production deployment and keep
`docs/CHANGELOG.md` in German, newest entry first.

Smoke checks:

```fish
curl -fsS https://jhw22.hausv.org/healthz
curl -I https://jhw22.hausv.org/
```

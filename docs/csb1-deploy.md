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
ssh -p 2222 mba@cs1.barta.cm 'cd ~/Code/nixcfg/hosts/csb1/docker; docker compose pull weg-portal; docker compose up -d weg-portal'
```

Smoke checks:

```fish
curl -fsS https://jhw22.hausv.org/healthz
curl -I https://jhw22.hausv.org/
```

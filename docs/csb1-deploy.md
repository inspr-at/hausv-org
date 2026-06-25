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
```

Production mail is sent directly through Resend SMTP. Verify `notify.hausv.org` in
Resend first, add the DNS records it gives you in Cloudflare, then store the
API key only in agenix as `SMTP_PASS`. The production sender is:

```text
WEG Portal <noreply@notify.hausv.org>
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

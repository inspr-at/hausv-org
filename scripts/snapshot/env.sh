# Deterministic local config for snapshot runs.
#
# Every value here is fake and committed on purpose: the snapshot harness must
# produce byte-identical pages across two builds, so it cannot depend on
# .env.local (real HA tokens, real SMTP, real emails, and it drifts).
#
# loadLocalEnv() skips keys already present in the environment, so exporting
# these BEFORE running the binary makes them win over .env.local.

export BASE_URL="http://localhost:${HV_PORT:-}"
export ADDR=":${HV_PORT:-}"
# NOT "localhost": the app renders the public marketing page when the request
# host equals ROOT_DOMAIN. Keeping the root domain distinct from the tenant host
# ("localhost") makes / resolve to the tenant login instead.
export ROOT_DOMAIN=hausv.test
export DEFAULT_TENANT=jhw22
export LOCAL_DEV_LOGIN=true

export WEG_TENANTS_JSON='[{"slug":"jhw22","name":"JHW22-Portal","address":"Janischhofweg 22, 8043 Graz","host":"localhost","map_latitude":47.1008592,"map_longitude":15.4717681,"map_zoom":17},{"slug":"eltern","name":"Haus Eltern","address":"Pilot Eltern","host":"eltern.hausv.test"},{"slug":"schwiegereltern","name":"Haus Schwiegereltern","address":"Pilot Schwiegereltern","host":"schwiegereltern.hausv.test"}]'
export WEG_USERS_JSON='[{"email":"admin@example.com","first_name":"Ada","last_name":"Admin","role":"Admin","status":"Aktiv","tenants":["jhw22"],"auth_methods":["email"]},{"email":"verwalter@example.com","first_name":"Vera","last_name":"Verwalter","role":"Verwalter","status":"Aktiv","tenants":["jhw22"],"auth_methods":["email"]},{"email":"owner@example.com","first_name":"Otto","last_name":"Eigentuemer","role":"Eigentümer","status":"Aktiv","tenants":["jhw22"],"auth_methods":["email"]},{"email":"resident@example.com","first_name":"Rita","last_name":"Bewohnerin","role":"Bewohner","status":"Aktiv","tenants":["jhw22"],"auth_methods":["email"]},{"email":"parents-owner@example.com","first_name":"Erika","last_name":"Eigentuemerin","role":"Eigentümer","status":"Aktiv","tenants":["eltern"],"auth_methods":["email"]},{"email":"inlaws-owner@example.com","first_name":"Ilse","last_name":"Eigentuemerin","role":"Eigentümer","status":"Aktiv","tenants":["schwiegereltern"],"auth_methods":["email"]}]'
export HOME_PROFILE_SEEDS_JSON='[{"tenant_slug":"jhw22","household_name":"QA Zuhause","home_type":"apartment","assets":["pv","ev","wallbox"]},{"tenant_slug":"eltern","household_name":"Haus Eltern","home_type":"house","assets":["pv","ev","hot-water","heat-pump"]},{"tenant_slug":"schwiegereltern","household_name":"Haus Schwiegereltern","home_type":"house","assets":["pv","battery","ev"]}]'
export ADMIN_EMAILS=admin@example.com
export INVITE_EMAILS=admin@example.com,verwalter@example.com,owner@example.com,resident@example.com,parents-owner@example.com,inlaws-owner@example.com

# Fixed session key so cookies from the baseline run stay valid for the
# candidate run — otherwise every page would just be the login screen.
export SESSION_KEY=snapshot-harness-fixed-key-not-a-secret-000

# Deterministic, local-only Home Assistant fixture. It deliberately includes
# device noise so Playwright proves that onboarding stays calm and read-only.
export HV_QA_JHW_HA_TOKEN="qa-read-only-jhw-fixture"
export HV_QA_PARENTS_HA_TOKEN="qa-read-only-parents-fixture"
export HV_QA_INLAWS_HA_TOKEN="qa-read-only-inlaws-fixture"
export HA_CONNECTORS_JSON="[{\"tenant_slug\":\"jhw22\",\"base_url\":\"http://127.0.0.1:${HV_QA_HA_PORT:-}/jhw22\",\"token_env\":\"HV_QA_JHW_HA_TOKEN\"},{\"tenant_slug\":\"eltern\",\"base_url\":\"http://127.0.0.1:${HV_QA_HA_PORT:-}/eltern\",\"token_env\":\"HV_QA_PARENTS_HA_TOKEN\"},{\"tenant_slug\":\"schwiegereltern\",\"base_url\":\"http://127.0.0.1:${HV_QA_HA_PORT:-}/schwiegereltern\",\"token_env\":\"HV_QA_INLAWS_HA_TOKEN\"}]"
# The same local fixture serves a valid PNG for the sidebar map. Browser QA
# must never depend on or send traffic to the public OpenStreetMap tile service.
export MAP_TILE_BASE_URL="http://127.0.0.1:${HV_QA_HA_PORT:-}/map-tiles"

# Explicitly shadow every external integration and secret that may exist in a
# developer's .env.local. Empty exported values are intentional: LoadLocalEnv
# will not replace them, so this harness cannot contact real systems.
export HA_BASE_URL=""
export HA_TOKEN=""
export SMTP_HOST=""
export SMTP_PORT=587
export SMTP_USER=""
export SMTP_PASS=""
export MAIL_FROM="HAUSV QA <qa@example.invalid>"
export OIDC_ISSUER=""
export OIDC_CLIENT_ID=""
export OIDC_CLIENT_SECRET=""
export OIDC_REDIRECT_URL=""
export OIDC_PROVIDER_NAME="Lokaler QA-Zugang"
export TELEGRAM_API_BASE_URL=""
export TELEGRAM_BOT_TOKEN=""
export TELEGRAM_DATA_PATH="${HV_DATA:-}/telegram.json"
export SERVICE_PROVIDER_ACCESS_ENABLED=false
export SERVICE_PROVIDER_ASSESSMENT_VERSION=""

# All state under one dir, seeded identically per run.
export DB_PATH="${HV_DATA:-}/hausv.db"
export PARKING_DATA_PATH="${HV_DATA:-}/parking.json"
export ANNOUNCE_DATA_PATH="${HV_DATA:-}/announcements.json"
export ANNOUNCE_READ_DATA_PATH="${HV_DATA:-}/announcement_reads.json"
export EVENT_DATA_PATH="${HV_DATA:-}/events.json"
export DOC_DATA_PATH="${HV_DATA:-}/documents.json"
export DOC_FILE_DIR="${HV_DATA:-}/documents"
export VOTE_DATA_PATH="${HV_DATA:-}/votes.json"
export HANDOVER_DATA_PATH="${HV_DATA:-}/handovers.json"
export NOTIFICATION_PREF_DATA_PATH="${HV_DATA:-}/notification_prefs.json"
export PROFILE_DATA_PATH="${HV_DATA:-}/profile_overlays.json"
export TENANT_DATA_PATH="${HV_DATA:-}/tenant_overrides.json"
export TENANT_HERO_DIR="${HV_DATA:-}/tenant-heroes"
export INVITE_DATA_PATH="${HV_DATA:-}/invites.json"
export ACTIVITY_DATA_PATH="${HV_DATA:-}/activity.json"
export AUDIT_DATA_PATH="${HV_DATA:-}/audit.jsonl"
export UNIT_DATA_PATH="${HV_DATA:-}/units.json"
export UNIT_PAYMENT_STATUS_DATA_PATH="${HV_DATA:-}/unit_payment_status.json"
export CONTACT_DATA_PATH="${HV_DATA:-}/contacts.json"
export ATTACHMENT_DATA_PATH="${HV_DATA:-}/attachments.json"
export ATTACHMENT_FILE_DIR="${HV_DATA:-}/attachments"
export ISSUE_DATA_PATH="${HV_DATA:-}/issues.json"
export ISSUE_ATTACHMENT_DIR="${HV_DATA:-}/issue-attachments"

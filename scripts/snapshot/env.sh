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
# The root domain serves the public marketing page. Tenant portals are reached
# through /<slug>, so browser QA does not need tenant-specific DNS entries.
export ROOT_DOMAIN=hausv.test
export DEFAULT_TENANT=demo
export LOCAL_DEV_LOGIN=true

# "cockpit" ist bewusst ein VIERTER Mandant: die drei anderen fahren im QA-Lauf
# den Einrichtungsassistenten durch, deshalb darf keiner von ihnen vorab
# abgeschlossen sein. Ohne einen abgeschlossenen Haushalt hat aber nie jemand
# das Energie-Cockpit gesehen — jede Aufnahme von /app/energie zeigte den
# Assistenten. Dieser Mandant schließt genau diese Lücke.
export WEG_TENANTS_JSON='[{"slug":"demo","name":"Demohaus","address":"Musterweg 1, 1010 Wien","map_latitude":48.2082,"map_longitude":16.3738,"map_zoom":17},{"slug":"haus-a","name":"Haus A","address":"Beispielweg 2, 1020 Wien"},{"slug":"haus-b","name":"Haus B","address":"Beispielweg 3, 1030 Wien","map_latitude":47.0707,"map_longitude":15.4395,"map_zoom":17},{"slug":"cockpit","name":"Energiehaus","address":"Energiestraße 8, 1020 Wien"}]'
export WEG_USERS_JSON='[{"email":"admin@example.com","first_name":"Ada","last_name":"Admin","role":"Admin","status":"Aktiv","tenants":["demo"],"permissions":["support-view"],"auth_methods":["email"]},{"email":"multi@example.com","first_name":"Mara","last_name":"Mehrhaus","role":"Admin","status":"Aktiv","tenants":["demo","haus-b"],"auth_methods":["email"]},{"email":"verwalter@example.com","first_name":"Vera","last_name":"Verwalter","role":"Verwalter","status":"Aktiv","tenants":["demo"],"auth_methods":["email"]},{"email":"owner@example.com","first_name":"Otto","last_name":"Eigentuemer","role":"Eigentümer","status":"Aktiv","tenants":["demo"],"auth_methods":["email"]},{"email":"resident@example.com","first_name":"Rita","last_name":"Bewohnerin","role":"Bewohner","status":"Aktiv","tenants":["demo"],"auth_methods":["email"]},{"email":"house-a-owner@example.com","first_name":"Alex","last_name":"Eigentuemer","role":"Eigentümer","status":"Aktiv","tenants":["haus-a"],"auth_methods":["email"]},{"email":"house-b-owner@example.com","first_name":"Bianca","last_name":"Eigentuemerin","role":"Eigentümer","status":"Aktiv","tenants":["haus-b"],"auth_methods":["email"]},{"email":"cockpit-owner@example.com","first_name":"Clara","last_name":"Eigentuemerin","role":"Eigentümer","status":"Aktiv","tenants":["cockpit"],"auth_methods":["email"]}]'
# complete=true überspringt den Assistenten auf Schritt 5. Der Betriebsmodus
# bleibt trotzdem "Nur beobachten" — ApplyProfileSeeds erlaubt keine
# Freigabe per Konfiguration.
export HOME_PROFILE_SEEDS_JSON='[{"tenant_slug":"demo","household_name":"QA Zuhause","home_type":"apartment","assets":["pv","ev","wallbox"]},{"tenant_slug":"haus-a","household_name":"Haus A","home_type":"house","assets":["pv","ev","hot-water","heat-pump"]},{"tenant_slug":"haus-b","household_name":"Haus B","home_type":"house","assets":["pv","battery","ev"]},{"tenant_slug":"cockpit","household_name":"Energiehaus","home_type":"house","complete":true,"assets":["pv","battery","ev","wallbox","heat-pump","hot-water",{"kind":"sauna","name":"Sauna Keller","rated_power_kw":8,"flexibility":"shift"}]}]'
export ADMIN_EMAILS=admin@example.com
export INVITE_EMAILS=admin@example.com,multi@example.com,verwalter@example.com,owner@example.com,resident@example.com,house-a-owner@example.com,house-b-owner@example.com,cockpit-owner@example.com

# Fixed session key so cookies from the baseline run stay valid for the
# candidate run — otherwise every page would just be the login screen.
export SESSION_KEY=snapshot-harness-fixed-key-not-a-secret-000

# Deterministic, local-only Home Assistant fixture. It deliberately includes
# device noise so Playwright proves that onboarding stays calm and read-only.
export HV_QA_DEMO_HA_TOKEN="qa-read-only-demo-fixture"
export HV_QA_HOUSE_A_HA_TOKEN="qa-read-only-house-a-fixture"
export HV_QA_HOUSE_B_HA_TOKEN="qa-read-only-house-b-fixture"
# Der Mandant "cockpit" liest bewusst dieselbe /demo-Fixture: sie hat die
# vollständigste Sensorlage (Netz beide Richtungen, PV, Speicher, Verbrauch).
# Damit braucht der abgeschlossene Haushalt keine eigene Fixture-Kopie.
export HA_CONNECTORS_JSON="[{\"tenant_slug\":\"demo\",\"base_url\":\"http://127.0.0.1:${HV_QA_HA_PORT:-}/demo\",\"token_env\":\"HV_QA_DEMO_HA_TOKEN\"},{\"tenant_slug\":\"haus-a\",\"base_url\":\"http://127.0.0.1:${HV_QA_HA_PORT:-}/haus-a\",\"token_env\":\"HV_QA_HOUSE_A_HA_TOKEN\"},{\"tenant_slug\":\"haus-b\",\"base_url\":\"http://127.0.0.1:${HV_QA_HA_PORT:-}/haus-b\",\"token_env\":\"HV_QA_HOUSE_B_HA_TOKEN\"},{\"tenant_slug\":\"cockpit\",\"base_url\":\"http://127.0.0.1:${HV_QA_HA_PORT:-}/demo\",\"token_env\":\"HV_QA_DEMO_HA_TOKEN\"}]"
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

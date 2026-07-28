# Deterministic local config for snapshot runs.
#
# Every value here is fake and committed on purpose: the snapshot harness must
# produce byte-identical pages across two builds, so it cannot depend on
# .env.local (real HA tokens, real SMTP, real emails, and it drifts).
#
# loadLocalEnv() skips keys already present in the environment, so exporting
# these BEFORE running the binary makes them win over .env.local.

set -gx BASE_URL "http://localhost:$HV_PORT"
set -gx ADDR ":$HV_PORT"
# NOT "localhost": the app renders the public marketing page when the request
# host equals ROOT_DOMAIN. Keeping the root domain distinct from the tenant host
# ("localhost") makes / resolve to the tenant login instead.
set -gx ROOT_DOMAIN hausv.test
set -gx DEFAULT_TENANT jhw22
set -gx LOCAL_DEV_LOGIN true

set -gx WEG_TENANTS_JSON '[{"slug":"jhw22","name":"WEG Portal","address":"Janischhofweg 22","host":"localhost"}]'
set -gx WEG_USERS_JSON '[{"email":"admin@example.com","first_name":"Ada","last_name":"Admin","role":"Admin","status":"Aktiv","tenants":["jhw22"],"auth_methods":["email"]},{"email":"verwalter@example.com","first_name":"Vera","last_name":"Verwalter","role":"Verwalter","status":"Aktiv","tenants":["jhw22"],"auth_methods":["email"]},{"email":"owner@example.com","first_name":"Otto","last_name":"Eigentuemer","role":"Eigentümer","status":"Aktiv","tenants":["jhw22"],"auth_methods":["email"]},{"email":"resident@example.com","first_name":"Rita","last_name":"Bewohnerin","role":"Bewohner","status":"Aktiv","tenants":["jhw22"],"auth_methods":["email"]}]'
set -gx HOME_PROFILE_SEEDS_JSON '[{"tenant_slug":"jhw22","household_name":"QA Zuhause","home_type":"apartment","assets":["pv","ev","wallbox"],"complete":true}]'
set -gx ADMIN_EMAILS admin@example.com
set -gx INVITE_EMAILS admin@example.com,verwalter@example.com,owner@example.com,resident@example.com

# Fixed session key so cookies from the baseline run stay valid for the
# candidate run — otherwise every page would just be the login screen.
set -gx SESSION_KEY snapshot-harness-fixed-key-not-a-secret-000

# No Home Assistant, no SMTP: outbound calls would make runs non-deterministic.
set -gx HA_BASE_URL ""
set -gx HA_TOKEN ""
set -gx SMTP_HOST ""

# All state under one dir, seeded identically per run.
set -gx PARKING_DATA_PATH "$HV_DATA/parking.json"
set -gx ANNOUNCE_DATA_PATH "$HV_DATA/announcements.json"
set -gx ANNOUNCE_READ_DATA_PATH "$HV_DATA/announcement_reads.json"
set -gx EVENT_DATA_PATH "$HV_DATA/events.json"
set -gx DOC_DATA_PATH "$HV_DATA/documents.json"
set -gx DOC_FILE_DIR "$HV_DATA/documents"
set -gx VOTE_DATA_PATH "$HV_DATA/votes.json"
set -gx HANDOVER_DATA_PATH "$HV_DATA/handovers.json"
set -gx NOTIFICATION_PREF_DATA_PATH "$HV_DATA/notification_prefs.json"
set -gx PROFILE_DATA_PATH "$HV_DATA/profile_overlays.json"
set -gx TENANT_DATA_PATH "$HV_DATA/tenant_overrides.json"
set -gx TENANT_HERO_DIR "$HV_DATA/tenant-heroes"
set -gx INVITE_DATA_PATH "$HV_DATA/invites.json"
set -gx ACTIVITY_DATA_PATH "$HV_DATA/activity.json"
set -gx AUDIT_DATA_PATH "$HV_DATA/audit.jsonl"
set -gx UNIT_DATA_PATH "$HV_DATA/units.json"
set -gx UNIT_PAYMENT_STATUS_DATA_PATH "$HV_DATA/unit_payment_status.json"
set -gx CONTACT_DATA_PATH "$HV_DATA/contacts.json"
set -gx ATTACHMENT_DATA_PATH "$HV_DATA/attachments.json"
set -gx ATTACHMENT_FILE_DIR "$HV_DATA/attachments"
set -gx ISSUE_DATA_PATH "$HV_DATA/issues.json"
set -gx ISSUE_ATTACHMENT_DIR "$HV_DATA/issue-attachments"

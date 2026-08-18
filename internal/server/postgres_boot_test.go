package server

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
)

func TestBootsOnPostgres(t *testing.T) {
	if dbtest.Backend() != db.BackendPostgres {
		if strings.EqualFold(strings.TrimSpace(os.Getenv("HAUSV_TEST_POSTGRES_REQUIRED")), "true") {
			t.Fatal("PostgreSQL boot test is required but the PostgreSQL test backend is not configured")
		}
		t.Skip("PostgreSQL test backend is not configured")
	}

	// dbtest owns the isolated schema and its cleanup. newApp then opens its own
	// real process pool against that same database, exactly as production does.
	provisioned := dbtest.Open(t)
	var schema string
	if err := provisioned.QueryRow(`SELECT current_schema()`).Scan(&schema); err != nil {
		t.Fatalf("read isolated PostgreSQL test schema: %v", err)
	}
	dsn, err := url.Parse(strings.TrimSpace(os.Getenv("HAUSV_TEST_POSTGRES_DSN")))
	if err != nil || (dsn.Scheme != "postgres" && dsn.Scheme != "postgresql") {
		// Parsing errors can include the URL, which carries the role password.
		t.Fatal("HAUSV_TEST_POSTGRES_DSN must be a PostgreSQL URL")
	}
	query := dsn.Query()
	query.Set("search_path", schema)
	dsn.RawQuery = query.Encode()

	t.Setenv("BASE_URL", "http://localhost:8080")
	t.Setenv("DB_BACKEND", string(db.BackendPostgres))
	t.Setenv("DATABASE_URL", dsn.String())
	t.Setenv("DB_LANE_CAP", "2")
	t.Setenv("DB_LANE_MAX_CONNS", "1")
	t.Setenv("WEG_TENANTS_JSON", "")
	t.Setenv("WEG_USERS_JSON", "")
	t.Setenv("HOME_PROFILE_SEEDS_JSON", "")
	t.Setenv("HA_CONNECTORS_JSON", "")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("OIDC_ISSUER", "")
	t.Setenv("TELEGRAM_BOT_TOKEN", "")

	dataDir := t.TempDir()
	for key, name := range map[string]string{
		"ANNOUNCE_DATA_PATH":            "announcements.json",
		"ANNOUNCE_READ_DATA_PATH":       "announcement_reads.json",
		"EVENT_DATA_PATH":               "events.json",
		"NOTIFICATION_PREF_DATA_PATH":   "notification_prefs.json",
		"PROFILE_DATA_PATH":             "profile_overlays.json",
		"TENANT_DATA_PATH":              "tenant_overrides.json",
		"INVITE_DATA_PATH":              "invites.json",
		"ACTIVITY_DATA_PATH":            "activity.json",
		"UNIT_DATA_PATH":                "units.json",
		"UNIT_PAYMENT_STATUS_DATA_PATH": "unit_payment_status.json",
		"ISSUE_DATA_PATH":               "issues.json",
		"ATTACHMENT_DATA_PATH":          "attachments.json",
		"CONTACT_DATA_PATH":             "contacts.json",
		"AUDIT_DATA_PATH":               "audit.jsonl",
		"DOC_DATA_PATH":                 "documents.json",
		"HANDOVER_DATA_PATH":            "handovers.json",
		"VOTE_DATA_PATH":                "votes.json",
		"PARKING_DATA_PATH":             "parking.json",
		"TELEGRAM_DATA_PATH":            "telegram.json",
	} {
		t.Setenv(key, filepath.Join(dataDir, name))
	}

	a, err := newApp()
	if err != nil {
		t.Fatalf("boot real app on PostgreSQL: %v", err)
	}
	if a == nil {
		t.Fatal("PostgreSQL boot returned a nil app")
	}
	t.Cleanup(a.closeMagicLinkDelivery)
	if a.db == nil || a.energyStore == nil {
		t.Fatal("PostgreSQL boot returned an app without its database-backed energy store")
	}
	var appSchema string
	if err := a.db.QueryRow(`SELECT current_schema()`).Scan(&appSchema); err != nil {
		t.Fatalf("query through booted app database: %v", err)
	}
	if appSchema != schema {
		t.Fatalf("app booted in schema %q, want isolated dbtest schema %q", appSchema, schema)
	}
	if err := a.Close(); err != nil {
		t.Fatalf("close PostgreSQL app: %v", err)
	}
}

package demo

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

// HAUSV-663: after a day of demos the annual statement page listed "Lauf 1 …
// Lauf 6" right after a reset. Stored runs, the delivery log and the archive
// documents (rows and files) belong to the demo day and must go with it.
func TestResetClearsAnnualStatementRunsArchiveAndDeliveries(t *testing.T) {
	database := dbtest.Open(t)
	dir := t.TempDir()
	seedDir := "../../scripts/demo/seed"
	load := func(discard bool) {
		t.Helper()
		if _, err := Load(t.Context(), database, seedDir, SeedOptions{Reset: true, DiscardAnnualStatements: discard, DocumentDir: dir, Anchor: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	load(false)
	const slug = "janusbergweg-123"
	var tenantID string
	if err := database.QueryRow(`SELECT tenant_id FROM tenant WHERE slug=$1`, slug).Scan(&tenantID); err != nil {
		t.Fatalf("demo tenant: %v", err)
	}
	documentsBefore, err := countRows(t, database, "documents")
	if err != nil {
		t.Fatal(err)
	}
	// Plant what a demo day leaves behind: a stored run, one delivery and an
	// archived PDF with its file on disk.
	tx, err := database.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(fmt.Sprintf("%T", database.Driver()), "stdlib") {
		if _, err := tx.Exec(`SET LOCAL hausv.cross_tenant = 'on'`); err != nil {
			t.Fatal(err)
		}
	}
	for _, statement := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO annual_statement_runs(tenant_id,tenant_slug,id,period_year,revision,data) VALUES($1,$2,'run-663',2025,1,'{}')`, []any{tenantID, slug}},
		{`INSERT INTO annual_statement_deliveries(tenant_id,tenant_slug,id,run_id,revision,party_id,unit_id,document_id,sha256,recipient,sent_at,status,error,actor,attempt) VALUES($1,$2,'delivery-663','run-663',1,'a@example.com','top-1','annual-archive-663','abc','a@example.com','2026-09-07T08:00:00Z','sent','','vera@example.com',1)`, []any{tenantID, slug}},
		{`INSERT INTO documents(tenant_id,tenant_slug,id,data) VALUES($1,$2,'annual-archive-663',$3)`, []any{tenantID, slug, `{"id":"annual-archive-663","tenant":"janusbergweg-123","title":"Jahresabrechnung 2025 · Top 1","stored_filename":"annual-archive-663.pdf","version":1,"current":true}`}},
	} {
		if _, err := tx.Exec(statement.query, statement.args...); err != nil {
			t.Fatalf("plant %s: %v", statement.query[:40], err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, slug, "annual-archive-663.pdf")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("%PDF-1.4 demo"), 0o644); err != nil {
		t.Fatal(err)
	}
	for table, want := range map[string]int{"annual_statement_runs": 1, "annual_statement_deliveries": 1} {
		if got, err := countRows(t, database, table); err != nil || got != want {
			t.Fatalf("%s before reset = %d (%v), want %d — the fixture did not land", table, got, err, want)
		}
	}

	// A plain reset is an input operation and keeps the immutable records.
	load(false)
	for table, want := range map[string]int{"annual_statement_runs": 1, "annual_statement_deliveries": 1} {
		if got, err := countRows(t, database, table); err != nil || got != want {
			t.Fatalf("%s after plain reset = %d (%v), want %d", table, got, err, want)
		}
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("plain reset must keep the archive file: %v", err)
	}

	load(true)

	for _, table := range []string{"annual_statement_runs", "annual_statement_deliveries"} {
		if got, err := countRows(t, database, table); err != nil || got != 0 {
			t.Errorf("%s after reset = %d (%v), want 0", table, got, err)
		}
	}
	var archived int
	if err := database.QueryRow(`SELECT COUNT(*) FROM documents WHERE id LIKE 'annual-archive-%'`).Scan(&archived); err != nil || archived != 0 {
		t.Errorf("archive documents after reset = %d (%v), want 0", archived, err)
	}
	if documentsAfter, err := countRows(t, database, "documents"); err != nil || documentsAfter != documentsBefore {
		t.Errorf("regular documents changed: %d → %d (%v)", documentsBefore, documentsAfter, err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("archive file still on disk: %v", err)
	}
	// The immutability guard is back after the reset on PostgreSQL.
	if strings.Contains(fmt.Sprintf("%T", database.Driver()), "stdlib") {
		if _, err := database.Exec(`DELETE FROM annual_statement_runs WHERE id='none'`); err != nil {
			t.Errorf("trigger check query: %v", err)
		}
		var enabled sql.NullString
		if err := database.QueryRow(`SELECT tgenabled FROM pg_trigger WHERE tgname='annual_statement_run_immutable'`).Scan(&enabled); err != nil || enabled.String != "O" {
			t.Errorf("immutability trigger not re-enabled: %q %v", enabled.String, err)
		}
	}
}

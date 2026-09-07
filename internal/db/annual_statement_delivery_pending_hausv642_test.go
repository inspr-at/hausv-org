package db

import (
	"path/filepath"
	"testing"
)

// HAUSV-642 widens the delivery status CHECK to admit 'pending'. SQLite can only
// do that by rebuilding the table, so the migration must carry every row, both
// keys and the partial unique index across unchanged.
func TestAnnualStatementDeliveryPendingMigrationRebuildsTheTableWithItsRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "delivery-pending.db")
	database, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO tenant(tenant_id,slug,created_at,updated_at) VALUES('01ARZ3NDEKTSV4RRFFQ69G5FAV','demo','2026-09-06T12:00:00Z','2026-09-06T12:00:00Z')`); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	// Restore the 0047 shape: the same table with the narrow CHECK.
	for _, statement := range []string{
		`DROP TABLE annual_statement_deliveries`,
		`CREATE TABLE annual_statement_deliveries (
    tenant_id TEXT NOT NULL REFERENCES tenant(tenant_id) ON UPDATE RESTRICT ON DELETE CASCADE,
    tenant_slug TEXT NOT NULL, id TEXT NOT NULL, run_id TEXT NOT NULL,
    revision INTEGER NOT NULL CHECK (revision > 0), party_id TEXT NOT NULL, unit_id TEXT NOT NULL,
    document_id TEXT NOT NULL, sha256 TEXT NOT NULL, recipient TEXT NOT NULL, sent_at TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('sent', 'failed')), error TEXT NOT NULL, actor TEXT NOT NULL,
    attempt INTEGER NOT NULL CHECK (attempt > 0),
    PRIMARY KEY (tenant_id, id), UNIQUE (tenant_id, run_id, revision, unit_id, party_id, attempt))`,
		`CREATE UNIQUE INDEX annual_statement_delivery_sent ON annual_statement_deliveries (tenant_id, run_id, revision, unit_id, party_id) WHERE status = 'sent'`,
		`INSERT INTO annual_statement_deliveries(tenant_id,tenant_slug,id,run_id,revision,party_id,unit_id,document_id,sha256,recipient,sent_at,status,error,actor,attempt) VALUES
    ('01ARZ3NDEKTSV4RRFFQ69G5FAV','demo','d1','run',1,'a@example.com','top-1','archive','abc','a@example.com','2026-09-06T12:00:00Z','failed','Testfehler','manager@example.com',1),
    ('01ARZ3NDEKTSV4RRFFQ69G5FAV','demo','d2','run',1,'a@example.com','top-1','archive','abc','a@example.com','2026-09-06T12:01:00Z','sent','','manager@example.com',2)`,
		`DELETE FROM schema_migrations WHERE version='0048_annual_statement_delivery_pending.sql'`,
	} {
		if _, err := database.Exec(statement); err != nil {
			t.Fatalf("restore pre-migration state: %v\n%s", err, statement)
		}
	}
	if _, err := database.Exec(`INSERT INTO annual_statement_deliveries(tenant_id,tenant_slug,id,run_id,revision,party_id,unit_id,document_id,sha256,recipient,sent_at,status,error,actor,attempt) VALUES('01ARZ3NDEKTSV4RRFFQ69G5FAV','demo','d3','run',1,'a@example.com','top-1','archive','abc','a@example.com','2026-09-06T12:02:00Z','pending','','manager@example.com',3)`); err == nil {
		t.Fatal("the pre-migration CHECK must refuse pending, or this test proves nothing")
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	database, err = Open(path)
	if err != nil {
		t.Fatalf("reopen (migration 0048): %v", err)
	}
	defer database.Close()
	var count int
	var status, errText string
	if err := database.QueryRow(`SELECT COUNT(*) FROM annual_statement_deliveries`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("rows after rebuild: %d %v", count, err)
	}
	if err := database.QueryRow(`SELECT status,error FROM annual_statement_deliveries WHERE id='d1'`).Scan(&status, &errText); err != nil || status != "failed" || errText != "Testfehler" {
		t.Fatalf("row d1 after rebuild: %s %q %v", status, errText, err)
	}
	insert := `INSERT INTO annual_statement_deliveries(tenant_id,tenant_slug,id,run_id,revision,party_id,unit_id,document_id,sha256,recipient,sent_at,status,error,actor,attempt) VALUES('01ARZ3NDEKTSV4RRFFQ69G5FAV','demo',?,'run',1,'a@example.com','top-1','archive','abc','a@example.com','2026-09-06T12:02:00Z',?,'','manager@example.com',?)`
	if _, err := database.Exec(insert, "d3", "pending", 3); err != nil {
		t.Fatalf("pending must be admitted after the migration: %v", err)
	}
	if _, err := database.Exec(insert, "d4", "bogus", 4); err == nil {
		t.Fatal("the CHECK must still refuse unknown states")
	}
	if _, err := database.Exec(insert, "d5", "sent", 5); err == nil {
		t.Fatal("the partial unique index on sent deliveries was lost in the rebuild")
	}
	if _, err := database.Exec(insert, "d6", "failed", 3); err == nil {
		t.Fatal("the unique attempt key was lost in the rebuild")
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_list('annual_statement_deliveries') WHERE "table"='tenant'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("tenant foreign key after rebuild: %d %v", count, err)
	}
	if _, err := database.Exec(`DELETE FROM tenant WHERE tenant_id='01ARZ3NDEKTSV4RRFFQ69G5FAV'`); err != nil {
		t.Fatalf("cascade delete: %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM annual_statement_deliveries`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("ON DELETE CASCADE after rebuild: %d %v", count, err)
	}
}

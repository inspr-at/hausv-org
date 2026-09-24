package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestImportCompletionMigrationPreservesHistoricalLedger(t *testing.T) {
	database, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	legacy, err := migrationsFS.ReadFile("migrations/0019_integration_imports.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(string(legacy)); err != nil {
		t.Fatal(err)
	}
	insert := `INSERT INTO integration_imports(tenant_slug,format,file_digest,applied_at,applied_by,changed) VALUES('demo','camt.053','digest','original-time','first@example.com',3)`
	if _, err := database.Exec(insert); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(insert); err == nil {
		t.Fatal("historical primary key allowed duplicate import")
	}
	migration, err := migrationsFS.ReadFile("migrations/0060_import_completion.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(string(migration)); err != nil {
		t.Fatal(err)
	}
	var at, actor, status, key, metadata string
	var changed int
	if err := database.QueryRow(`SELECT applied_at,applied_by,changed,status,blob_key,document_data FROM integration_imports`).Scan(&at, &actor, &changed, &status, &key, &metadata); err != nil {
		t.Fatal(err)
	}
	if at != "original-time" || actor != "first@example.com" || changed != 3 || status != "complete" || key != "" || metadata != "" {
		t.Fatalf("historical evidence changed: %s %s %d %s %s %s", at, actor, changed, status, key, metadata)
	}
}

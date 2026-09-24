package db

import (
	"os"
	"strings"
	"testing"
)

func TestPostgresIndexReferenceGlobalReadMaintenanceWrite(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("HAUSV_TEST_POSTGRES_DSN"))
	if dsn == "" {
		if os.Getenv("HAUSV_TEST_POSTGRES_REQUIRED") == "true" {
			t.Fatal("PostgreSQL DSN required")
		}
		t.Skip("disposable PostgreSQL required")
	}
	cfg := Config{Backend: BackendPostgres, DSN: isolatedPostgresSchema(t, dsn)}
	database, err := OpenConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	scoped, err := NewScoped(cfg, database)
	if err != nil {
		t.Fatal(err)
	}
	defer scoped.Close()
	maintenance := scoped.Unscoped("test: import global index reference data")
	insertImport := `INSERT INTO index_imports(id,url,sha256,data_sha256,status_sha256,actor,fetched_at,rows_added,rows_revised,rows_flagged,changes) VALUES('test','official','hash','hash','hash','system','2026-09-24T00:00:00Z',1,0,0,'[]')`
	if _, err = database.Exec(insertImport); sqlState(err) != "42501" {
		t.Fatalf("process wrote import: %v", err)
	}
	if _, err = maintenance.Exec(insertImport); err != nil {
		t.Fatal(err)
	}
	insertValue := `INSERT INTO index_values(id,series,period,value_millionths,status,source,fetched_at,import_id) VALUES('value','VPI2020','2026-09',133200000,'preliminary','official','2026-09-24T00:00:00Z','test')`
	if _, err = maintenance.Exec(insertValue); err != nil {
		t.Fatal(err)
	}
	for _, lane := range []Handle{database, scoped.For(tenantA), scoped.For(tenantB)} {
		for _, table := range []string{"index_values", "index_imports"} {
			var n int
			if err = lane.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil || n != 1 {
				t.Fatalf("global read %s: %d %v", table, n, err)
			}
			for _, query := range []string{`UPDATE ` + table + ` SET id=id`, `DELETE FROM ` + table} {
				if _, err = lane.Exec(query); sqlState(err) != "42501" {
					t.Fatalf("tenant mutation %s: %v", query, err)
				}
			}
		}
		if _, err = lane.Exec(strings.Replace(insertValue, "'value'", "'denied'", 1)); sqlState(err) != "42501" {
			t.Fatalf("tenant inserted value: %v", err)
		}
	}
	// The existing RLS fixture discovers tenant tables by tenant_id. Confirm
	// these intentionally global tables neither enter that fixture nor use RLS.
	for _, table := range []string{"index_values", "index_imports"} {
		var rls bool
		var columns int
		if err = database.QueryRow(`SELECT relrowsecurity FROM pg_class WHERE oid=$1::regclass`, table).Scan(&rls); err != nil || rls {
			t.Fatalf("global RLS: %v %v", rls, err)
		}
		if err = database.QueryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name=$1 AND column_name='tenant_id'`, table).Scan(&columns); err != nil || columns != 0 {
			t.Fatalf("tenant column: %d %v", columns, err)
		}
	}
}

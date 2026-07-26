package server

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	appdb "github.com/markus-barta/hausv-org/internal/db"
)

func TestHealthChecksDatabaseAndWritableDataDir(t *testing.T) {
	dir := t.TempDir()
	database, err := appdb.Open(filepath.Join(dir, "hausv.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	a := &app{db: database, dataDir: dir}
	rr := httptest.NewRecorder()
	a.health(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != http.StatusOK || rr.Body.String() != `{"service":"hausv-org","status":"ok"}` {
		t.Fatalf("healthy response = %d %q", rr.Code, rr.Body.String())
	}
}

func TestHealthRejectsMissingDependency(t *testing.T) {
	for name, a := range map[string]*app{
		"database": {dataDir: t.TempDir()},
		"data-dir": {db: mustTestDB(t)},
	} {
		t.Run(name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			a.health(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
			if rr.Code != http.StatusServiceUnavailable || rr.Body.String() != `{"service":"hausv-org","status":"unhealthy"}` {
				t.Fatalf("unhealthy response = %d %q", rr.Code, rr.Body.String())
			}
		})
	}
}

func mustTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := appdb.Open(filepath.Join(t.TempDir(), "hausv.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

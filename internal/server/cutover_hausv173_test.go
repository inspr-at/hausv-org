package server

import (
	"strings"
	"testing"
)

// HAUSV-173: after the cutover there is no JSON fallback. A database that
// cannot be opened must ABORT the boot, because the alternative — quietly
// serving a stale second copy of the data — is the failure mode the cutover
// exists to remove.
func TestBootFailsWhenSQLiteCannotOpen(t *testing.T) {
	// A directory is not a usable database file.
	t.Setenv("DB_PATH", t.TempDir())
	t.Setenv("BASE_URL", "http://localhost:8080")

	_, err := newApp()
	if err == nil {
		t.Fatal("boot must fail when SQLite cannot be opened, not fall back to JSON")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "sqlite") {
		t.Fatalf("error should name sqlite so the cause is obvious, got: %v", err)
	}
}

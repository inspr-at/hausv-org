package store

import (
	"testing"
)

// This is intentionally an actual migrated boolean column, not a driver-only
// probe. It proves the one representation the stores use works unchanged on
// SQLite and PostgreSQL.
func TestSQLBoolRoundTrip(t *testing.T) {
	database := testDB(t)
	for _, want := range []bool{true, false} {
		email := "false@example.com"
		if want {
			email = "true@example.com"
		}
		if _, err := database.Exec(
			`INSERT INTO profile_overlays(email,directory_opt_in,updated_at) VALUES($1,$2,$3)`,
			email, want, "2026-08-17T00:00:00Z",
		); err != nil {
			t.Fatalf("write %t: %v", want, err)
		}
		var got bool
		if err := database.QueryRow(
			`SELECT directory_opt_in FROM profile_overlays WHERE email=$1`, email,
		).Scan(&got); err != nil {
			t.Fatalf("read %t: %v", want, err)
		}
		if got != want {
			t.Fatalf("round trip = %t, want %t", got, want)
		}
	}
}

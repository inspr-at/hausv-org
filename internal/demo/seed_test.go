package demo

import (
	"bytes"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestLoadIsIdempotentAndResettable(t *testing.T) {
	database := dbtest.Open(t)
	dir := filepath.Join("testdata")
	var out bytes.Buffer
	first, err := Load(t.Context(), database, dir, SeedOptions{Stats: true, Out: &out})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Load(t.Context(), database, dir, SeedOptions{})
	if err != nil {
		t.Fatal(err)
	}
	third, err := Load(t.Context(), database, dir, SeedOptions{Reset: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Statuses) != len(second.Statuses) || len(second.Statuses) != len(third.Statuses) || first.Statuses[store.IntakeStatusApproved] != 2 {
		t.Fatalf("unexpected results: %#v %#v %#v", first, second, third)
	}
	for table, want := range map[string]int{"intake_items": 6, "issues": 4, "events": 1, "announcements": 1, "units": 1, "textbausteine": 1, "org_settings": 1} {
		got, err := countRows(t, database, table)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s rows = %d, want %d", table, got, want)
		}
	}
	if out.Len() == 0 {
		t.Fatal("stats output is empty")
	}
}

// countRows counts through the maintenance lane on PostgreSQL, where row-level
// security hides every row from a connection without tenant or organisation
// context; SQLite has no such policy.
func countRows(t *testing.T, database *sql.DB, table string) (int, error) {
	t.Helper()
	tx, err := database.BeginTx(t.Context(), nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if strings.Contains(fmt.Sprintf("%T", database.Driver()), "stdlib") {
		if _, err := tx.ExecContext(t.Context(), `SET LOCAL hausv.cross_tenant = 'on'`); err != nil {
			return 0, err
		}
	}
	var got int
	err = tx.QueryRowContext(t.Context(), `SELECT count(*) FROM `+table).Scan(&got)
	return got, err
}

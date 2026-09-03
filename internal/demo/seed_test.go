package demo

import (
	"bytes"
	"path/filepath"
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
		var got int
		if err := database.QueryRow(`SELECT count(*) FROM ` + table).Scan(&got); err != nil {
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

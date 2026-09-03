package store

import (
	"encoding/json"
	"testing"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

func TestTextbausteinRepositoryListActiveAndInactiveUpsert(t *testing.T) {
	database := dbtest.Open(t)
	repo := BindTextbausteinRepository(database, "musterstadt")
	for _, item := range []Textbaustein{
		{Key: "aktiv", Category: IntakeCategoryRepair, Title: "Aktiv", Body: "Hallo {{Name}}", Active: true},
		{Key: "inaktiv", Category: IntakeCategoryOther, Title: "Inaktiv", Body: "Pause", Active: false},
	} {
		if err := repo.Upsert(t.Context(), item); err != nil {
			t.Fatal(err)
		}
	}

	all, err := repo.List(t.Context())
	if err != nil || len(all) != 2 {
		t.Fatalf("List() = %#v, %v", all, err)
	}
	active, err := repo.ListActive(t.Context())
	if err != nil || len(active) != 1 || active[0].Key != "aktiv" {
		t.Fatalf("ListActive() = %#v, %v", active, err)
	}
	inactive, err := repo.Get(t.Context(), "inaktiv")
	if err != nil || inactive.Active || inactive.UpdatedAt.IsZero() {
		t.Fatalf("inactive round trip = %#v, %v", inactive, err)
	}
}

func TestTextbausteinMissingActiveDefaultsToTrue(t *testing.T) {
	var item Textbaustein
	if err := json.Unmarshal([]byte(`{"key":"alt","category":"sonstiges","title":"Alt","body":"Text"}`), &item); err != nil {
		t.Fatal(err)
	}
	if !item.Active {
		t.Fatal("catalogue entry without active must default to active")
	}

	if err := json.Unmarshal([]byte(`{"key":"aus","category":"sonstiges","title":"Aus","body":"Text","active":false}`), &item); err != nil {
		t.Fatal(err)
	}
	if item.Active {
		t.Fatal("explicit inactive flag was lost")
	}
}

func TestTextbausteinPlaceholdersMatchStarterCatalogue(t *testing.T) {
	want := []string{"{{Anrede}}", "{{Name}}", "{{Haus}}", "{{Einheit}}", "{{Nummer}}", "{{Zuständig}}", "{{Handwerker}}", "{{Frist}}"}
	got := TextbausteinPlaceholders()
	if len(got) != len(want) {
		t.Fatalf("placeholders = %#v", got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("placeholder %d = %q, want %q", index, got[index], want[index])
		}
	}
}

package server

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

func TestArchivedDocumentsFollowRegisterOrder(t *testing.T) {
	register := []store.Unit{
		{ID: "parking", Label: "Stellplatz 1", UnitType: store.UnitTypeParking},
		{ID: "ten", Label: "Top 10"}, {ID: "two", Label: "Top 2"}, {ID: "one", Label: "Top 1"},
	}
	now := time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC)
	var items []documentRecord
	for _, unit := range register {
		items = append(items, documentRecord{ID: unit.ID, UnitID: unit.ID, Title: "Jahresabrechnung 2025 · " + unit.Label + " · Lauf 1", UploadedAt: now, AnnualStatementArchive: &store.AnnualStatementArchiveMetadata{RunID: "run", PeriodYear: 2025, Revision: 1}})
	}
	for _, mode := range []string{"", "newest", "oldest", "title"} {
		t.Run(fmt.Sprintf("sort=%s", mode), func(t *testing.T) {
			var got []string
			for _, item := range sortDocumentsForView(items, mode, documentSortContext{Units: register}) {
				got = append(got, item.UnitID)
			}
			if want := []string{"one", "two", "ten", "parking"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
			if items[0].ID != "parking" {
				t.Fatal("sort mutated source")
			}
		})
	}
	t.Run("date precedence", func(t *testing.T) {
		withNewer := append(append([]documentRecord(nil), items...), documentRecord{ID: "latest", Title: "Neu", UploadedAt: now.Add(time.Hour)})
		if got := sortDocumentsForView(withNewer, "newest", documentSortContext{Units: register})[0].ID; got != "latest" {
			t.Fatalf("newest=%s", got)
		}
		if got := sortDocumentsForView(withNewer, "oldest", documentSortContext{Units: register})[0].ID; got != "one" {
			t.Fatalf("oldest=%s", got)
		}
	})
	t.Run("immutable run order after rename and partial archive retry", func(t *testing.T) {
		run := store.AnnualStatementRun{}
		for _, unit := range register {
			run.Input.Units = append(run.Input.Units, store.AnnualStatementRunUnitIdentity{ID: unit.ID, Label: unit.Label, UnitType: unit.UnitType})
			run.Result.Units = append(run.Result.Units, store.AnnualStatementRunUnit{UnitID: unit.ID, Label: unit.Label})
		}
		changed := []store.Unit{{ID: "two", Label: "Top 99"}, {ID: "one", Label: "Top 100"}}
		partial := append([]documentRecord(nil), items...)
		partial[0].UploadedAt = now.Add(time.Hour)
		for _, mode := range []string{"newest", "oldest", "title"} {
			var got []string
			for _, item := range sortDocumentsForView(partial, mode, documentSortContext{Units: changed, Runs: map[string]store.AnnualStatementRun{"run": run}}) {
				got = append(got, item.UnitID)
			}
			if want := []string{"one", "two", "ten", "parking"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("%s got %v want %v", mode, got, want)
			}
		}
	})
	t.Run("legacy unit IDs without register", func(t *testing.T) {
		var legacy []documentRecord
		for _, id := range []string{"stellplatz-1", "top-10", "top-2", "top-1"} {
			legacy = append(legacy, documentRecord{ID: id, UnitID: id, AnnualStatementArchive: &store.AnnualStatementArchiveMetadata{RunID: "legacy"}})
		}
		var got []string
		for _, item := range sortDocumentsForView(legacy, "") {
			got = append(got, item.UnitID)
		}
		if want := []string{"top-1", "top-2", "top-10", "stellplatz-1"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v", got)
		}
	})
}

func TestArchivePartyNameAndAccessibleFallback(t *testing.T) {
	item := documentRecord{ID: "document", UnitID: "top-1", AnnualStatementArchive: &store.AnnualStatementArchiveMetadata{RunID: "run", PartyID: "anna@example.com"}}
	for _, tc := range []struct {
		name string
		run  store.AnnualStatementRun
		want string
	}{
		{"named", store.AnnualStatementRun{Input: store.AnnualStatementRunInput{Parties: []store.AnnualStatementRunParty{{UnitID: "top-1", ID: "anna@example.com", Name: "Anna Auer"}}}}, "Für Anna Auer"},
		{"nameless", store.AnnualStatementRun{Input: store.AnnualStatementRunInput{Parties: []store.AnnualStatementRunParty{{UnitID: "top-1", ID: "anna@example.com"}}}}, "Partei ohne gespeicherten Namen"},
		{"missing run", store.AnnualStatementRun{}, "Partei ohne gespeicherten Namen"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := documentViewWithArchiveSnapshot(item, tc.run)
			var body bytes.Buffer
			if err := web.DocumentRow(v, false).Render(t.Context(), &body); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(body.String(), tc.want) {
				t.Fatalf("missing %s", tc.want)
			}
			if tc.name != "named" && !strings.Contains(body.String(), `class="document-party-address">anna@example.com</span>`) {
				t.Fatal("fallback address must be visible for touch and keyboard")
			}
			if strings.Contains(body.String(), "Für anna@example.com") {
				t.Fatal("email must be subtitle rather than name chip")
			}
		})
	}
}

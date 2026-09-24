package demo

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/statementpdf"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestCommittedDemo2025CreatesRunAndEveryPartyPDF(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(filepath.Join(t.TempDir(), "demo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	scoped, err := db.NewScoped(db.Config{Backend: db.BackendSQLite}, database)
	if err != nil {
		t.Fatal(err)
	}
	defer scoped.Close()
	lanes := store.NewTenantDB(scoped)
	dir := "../../scripts/demo/seed"
	options := SeedOptions{Reset: true, DocumentDir: filepath.Join(t.TempDir(), "originals"), Anchor: time.Date(2030, 8, 1, 0, 0, 0, 0, time.UTC)}
	if _, err := Load(ctx, database, dir, options); err != nil {
		t.Fatal(err)
	}
	identities, err := store.EnsureTenantIdentities(ctx, database, []store.TenantIdentity{{Slug: "janusbergweg-123"}})
	if err != nil {
		t.Fatal(err)
	}
	tenant := identities["janusbergweg-123"].Ref()
	docs := store.NewSQLDocumentStore(lanes, options.DocumentDir)
	documents, _ := store.BindDocumentRepository(docs, tenant)
	repo, _ := store.BindAnnualStatementRunRepository(store.NewSQLAnnualStatementRunStore(lanes, docs), tenant)
	if runs, err := repo.List(2025); err != nil || len(runs) != 0 {
		t.Fatal("seed must not create a run", err)
	}
	originals := map[string][]byte{}
	for _, doc := range documents.List() {
		path, ok := documents.FilePath(doc)
		if !ok {
			t.Fatal(doc)
		}
		raw, err := os.ReadFile(path)
		if err != nil || !bytes.HasPrefix(raw, []byte("%PDF-")) {
			t.Fatal("missing original", err)
		}
		originals[doc.ID] = raw
	}
	// Five cost receipts, the Rücklage withdrawal receipt, plus six portal documents (HAUSV-711).
	if len(originals) != 12 {
		t.Fatalf("original count=%d", len(originals))
	}
	presentation := store.AnnualStatementRunPresentation{Organisation: "Hausverwaltung Musterstadt", EstateSlug: "janusbergweg-123", EstateName: "Janusbergweg 123", EstateAddress: "Janusbergweg 123, 8010 Graz", ContactName: "Vera Verwalter", ContactEmail: "vera.verwalter@musterstadt.example"}
	run, err := repo.Create(2025, "vera.verwalter@musterstadt.example", time.Date(2026, 1, 20, 9, 0, 0, 0, time.UTC), presentation)
	if err != nil {
		t.Fatal(err)
	}
	loaded, found, err := repo.Get(run.ID)
	if err != nil || !found {
		t.Fatal("load run", err)
	}
	if len(loaded.Result.Units) != 24 || len(loaded.Input.Receipts) != 5 || len(loaded.Input.Prepayments) != 24 || loaded.Result.TotalCents != 2070000 || loaded.Input.Period.StartsOn != "2025-01-01" || loaded.Input.Period.EndsOn != "2025-12-31" {
		t.Fatalf("incomplete fixture: %+v", loaded)
	}
	if !loaded.Input.Structure.Legal.HeizKGApplies || loaded.Input.Structure.Legal.HeatingConsumptionPercent != 70 || len(loaded.Result.Proposals) != 96 {
		t.Fatal("missing legal demo inputs")
	}
	if loaded.Result.Reserve == nil || loaded.Result.Reserve.ClosingCents != 3184328 || loaded.Result.Reserve.MinimumWarning || len(loaded.Input.Reserve) != 15 {
		t.Fatalf("reserve snapshot: %+v entries=%d", loaded.Result.Reserve, len(loaded.Input.Reserve))
	}
	ppm := 0
	keys := map[string]bool{}
	for _, basis := range loaded.Input.Structure.UnitBases {
		ppm += basis.MiteigentumsanteilPPM
		if !basis.UsableAreaRecorded || !basis.PersonsRecorded {
			t.Fatal("missing basis")
		}
	}
	for _, cost := range loaded.Input.Structure.CostTypes {
		keys[cost.AllocationKey] = true
	}
	if ppm != 1000000 || !keys[store.AllocationKeyNutzwert] || !keys[store.AllocationKeyFlaeche] || !keys[store.AllocationKeyPersonen] || !keys[store.AllocationKeyVerbrauch] {
		t.Fatal("invalid bases/keys", ppm, keys)
	}
	renderedUnits := map[string]bool{}
	pdfs := map[string][]byte{}
	var persons []seedPerson
	if err := readJSON(filepath.Join(dir, "persons.json"), &persons); err != nil {
		t.Fatal(err)
	}
	byEmail := map[string]seedPerson{}
	for _, p := range persons {
		byEmail[p.Email] = p
	}
	for _, party := range loaded.Input.Parties {
		person, ok := byEmail[party.ID]
		if !ok || party.Name != person.Name || party.Address != person.Address || party.Address == "" {
			t.Fatalf("invented/missing party: %+v", party)
		}
		raw, err := statementpdf.Render(loaded, party.UnitID, party.ID)
		if err != nil || !bytes.HasPrefix(raw, []byte("%PDF-")) {
			t.Fatal("party PDF", err)
		}
		renderedUnits[party.UnitID] = true
		pdfs[party.UnitID+"/"+party.ID] = raw
	}
	if len(renderedUnits) != 24 || len(pdfs) != 30 {
		t.Fatalf("units=%d documents=%d", len(renderedUnits), len(pdfs))
	}
	all, err := statementpdf.Render(loaded, "", "")
	if err != nil || !bytes.Contains(all, []byte("R\\374cklage")) || !bytes.Contains(all, []byte("Anfangsstand")) || !bytes.Contains(all, []byte("Anteil dieser Einheit")) {
		t.Fatal("party PDF missing Rücklage", err)
	}
	// A second normal seed is idempotent; reset also repairs edited inputs.
	options.Reset = false
	if _, err := Load(ctx, database, dir, options); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`UPDATE annual_statement_prepayments SET amount_cents=1 WHERE tenant_id=$1 AND period_year=2025`, tenant.ID); err != nil {
		t.Fatal(err)
	}
	options.Reset = true
	if _, err := Load(ctx, database, dir, options); err != nil {
		t.Fatal(err)
	}
	if runs, err := repo.List(2025); err != nil || len(runs) != 1 || !reflect.DeepEqual(runs[0], loaded) {
		t.Fatal("seed changed stored run", err)
	}
	for _, doc := range documents.List() {
		path, _ := documents.FilePath(doc)
		raw, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(raw, originals[doc.ID]) {
			t.Fatal("original changed", err)
		}
	}
	for table, want := range map[string]int{"annual_statement_periods": 1, "annual_statement_period_cost_types": 4, "annual_statement_period_unit_bases": 24, "annual_statement_receipts": 5, "annual_statement_prepayments": 24, "annual_statement_reserve_entries": 15, "annual_statement_runs": 1} {
		var count int
		if err := database.QueryRow(`SELECT count(*) FROM `+table+` WHERE tenant_id=$1`, tenant.ID).Scan(&count); err != nil || count != want {
			t.Fatalf("%s=%d want=%d err=%v", table, count, want, err)
		}
	}
	next, err := repo.Create(2025, "vera.verwalter@musterstadt.example", run.CreatedAt.Add(time.Hour), presentation)
	if err != nil || !reflect.DeepEqual(next.Result, loaded.Result) || next.InputHash != loaded.InputHash {
		t.Fatal("reset did not restore runnable fixture", err)
	}
	again, _ := statementpdf.Render(loaded, "", "")
	if !bytes.Equal(all, again) {
		t.Fatal("reseed changed PDF")
	}
	if out := os.Getenv("HAUSV_DEMO_PDF_TEST_OUTPUT"); out != "" {
		if err := os.WriteFile(out, all, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if dir := os.Getenv("HAUSV_ANNUAL_LEGAL_QA_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		for _, party := range loaded.Input.Parties {
			if party.UnitID != "top-1" {
				continue
			}
			draft, err := statementpdf.Render(loaded, party.UnitID, party.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "draft.pdf"), draft, 0600); err != nil {
				t.Fatal(err)
			}
			loaded.Approval = &store.AnnualStatementRunApproval{ApprovedAt: time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC), ApprovedBy: "vera.verwalter@musterstadt.example", Role: store.RoleManager}
			final, err := statementpdf.Render(loaded, party.UnitID, party.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "final.pdf"), final, 0600); err != nil {
				t.Fatal(err)
			}
			loaded.Input.Structure.Legal.Regime = "mrg_voll"
			notice, err := statementpdf.RenderAushang(loaded)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "aushang.pdf"), notice, 0600); err != nil {
				t.Fatal(err)
			}
			break
		}
		for _, u := range loaded.Result.Units {
			if u.UnitID == "top-1" {
				t.Logf("Top 1: %+v", u)
			}
		}
	}
	t.Logf("2025: %d units, %d party PDFs, %d originals, total %d cents", len(renderedUnits), len(pdfs), len(originals), loaded.Result.TotalCents)
}

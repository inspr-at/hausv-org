package store

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	appdb "github.com/inspr-at/hausv-org/internal/db"
)

func seedAnnualRunSources(t *testing.T, sources AnnualStatementRunSources, tenant TenantRef) (AnnualStatementReceipt, DocumentRecord) {
	t.Helper()
	return seedAnnualRunInput(t, sources, tenant, annualRunFixture())
}

func seedAnnualRunInput(t *testing.T, sources AnnualStatementRunSources, tenant TenantRef, in AnnualStatementRunInput) (AnnualStatementReceipt, DocumentRecord) {
	t.Helper()
	const actor = "manager@example.com"
	units, _ := BindUnitRepository(sources.Units, tenant)
	records := []Unit{{ID: "a", Label: "Top 1", MiteigentumsanteilPPM: 250000, Persons: 1, PersonsRecorded: true}, {ID: "b", Label: "Top 2", MiteigentumsanteilPPM: 750000, Persons: 1, PersonsRecorded: true}}
	if err := units.SetUnits(records); err != nil {
		t.Fatal(err)
	}
	periods, _ := BindAnnualStatementPeriodRepository(sources.Periods, tenant)
	in.Period.UpdatedBy = actor
	if _, err := periods.SaveWithStructure(in.Period, in.Structure.CostTypes, records); err != nil {
		t.Fatal(err)
	}
	docs, _ := BindDocumentRepository(sources.Documents, tenant)
	receipts, _ := BindAnnualStatementReceiptRepository(sources.Receipts, tenant)
	var first AnnualStatementReceipt
	var firstDoc DocumentRecord
	for i, item := range in.Receipts {
		doc, err := docs.CreateGenerated(DocumentRecord{Title: item.CostTypeKey, Category: DocumentCategoryBilling, Visibility: DocumentVisibilityManagerOnly, UploadedBy: actor}, item.CostTypeKey+".pdf", "application/pdf", []byte("%PDF-1.4 original"), time.Now())
		if err != nil {
			t.Fatal(err)
		}
		item.DocumentID = doc.ID
		item.CreatedBy = actor
		receipt, err := receipts.Create(item)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = receipt
			firstDoc = doc
		}
	}
	prepayments, _ := BindAnnualStatementPrepaymentRepository(sources.Prepayments, tenant)
	for _, item := range in.Prepayments {
		item.UpdatedBy = actor
		if _, _, err := prepayments.Save(item); err != nil {
			t.Fatal(err)
		}
	}
	return first, firstDoc
}

func TestAnnualStatementRunStorage(t *testing.T) {
	for _, kind := range []string{"memory", "sql"} {
		t.Run(kind, func(t *testing.T) {
			var sources AnnualStatementRunSources
			var storage AnnualStatementRunStorage
			if kind == "memory" {
				periods := NewMemoryAnnualStatementPeriodStore()
				units, err := NewUnitStore("")
				if err != nil {
					t.Fatal(err)
				}
				docs, err := NewDocumentStore("", filepath.Join(t.TempDir(), "docs"))
				if err != nil {
					t.Fatal(err)
				}
				sources = AnnualStatementRunSources{Periods: periods, Units: units, Documents: docs, Receipts: NewMemoryAnnualStatementReceiptStore(periods, NewMemoryAnnualStatementCostTypeStore(), docs), Prepayments: NewMemoryAnnualStatementPrepaymentStore(periods, units), Consumption: NewMemoryAnnualStatementConsumptionStore()}
				storage = NewMemoryAnnualStatementRunStore(sources)
			} else {
				_, lanes := testLanes(t)
				docs := NewSQLDocumentStore(lanes, filepath.Join(t.TempDir(), "docs"))
				sources = AnnualStatementRunSources{Periods: NewSQLAnnualStatementPeriodStore(lanes), Units: NewSQLUnitStore(lanes), Documents: docs, Receipts: NewSQLAnnualStatementReceiptStore(lanes), Prepayments: NewSQLAnnualStatementPrepaymentStore(lanes), Consumption: NewSQLAnnualStatementConsumptionStore(lanes)}
				storage = NewSQLAnnualStatementRunStore(lanes, docs)
			}
			tenant := testTenantRef("demo")
			receipt, document := seedAnnualRunSources(t, sources, tenant)
			units, _ := BindUnitRepository(sources.Units, tenant)
			contacts := []UnitPartyContact{{Email: "anna@example.com", Name: "Anna Groß", Address: "Gasse 2, 8010 Graz"}, {Email: "mieter@example.com", ValidFrom: "2025-07-01"}}
			if _, err := units.UpdateParties([]UnitPartyUpdate{{UnitID: "a", SetOwners: true, SetRenters: true, OwnerEmails: []string{"anna@example.com"}, RenterEmails: []string{"anna@example.com", "mieter@example.com"}, Contacts: contacts}}); err != nil {
				t.Fatal(err)
			}
			repo, ok := BindAnnualStatementRunRepository(storage, tenant)
			if !ok {
				t.Fatal("bind")
			}
			if _, ok := BindAnnualStatementRunRepository(storage, TenantRef{}); ok {
				t.Fatal("invalid tenant bound")
			}
			_, preview, err := repo.Preview(2025, nil)
			if err != nil || preview.TotalCents != 10102 {
				t.Fatalf("preview: %+v %v", preview, err)
			}
			before, _ := repo.List(2025)
			if len(before) != 0 {
				t.Fatal("preview wrote a run")
			}
			run, err := repo.Create(2025, "manager@example.com", time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if run.ID == "" || run.Revision != 1 || run.CalculationVersion != 2 || run.InputHash == "" || run.Result.Units[0].BalanceCents != -449 {
				t.Fatalf("run=%+v", run)
			}
			original := copyAnnualStatementRun(run)
			if len(run.Input.Parties) != 1 || run.Input.Parties[0].Address != "Gasse 2, 8010 Graz" || !run.Input.Parties[0].Owner || !run.Input.Parties[0].Renter {
				t.Fatalf("parties=%+v", run.Input.Parties)
			}
			if _, err := units.UpdateParties([]UnitPartyUpdate{{UnitID: "a", SetOwners: true, OwnerEmails: []string{"new@example.com"}, Contacts: []UnitPartyContact{{Email: "anna@example.com", Address: "Changed"}}}}); err != nil {
				t.Fatal(err)
			}
			loaded, found, err := repo.Get(run.ID)
			if err != nil || !found || !reflect.DeepEqual(loaded, original) {
				t.Fatal("stored snapshot changed", err)
			}
			loaded.Input.Parties[0].Address = "Mutated return value"
			again, _, _ := repo.Get(run.ID)
			if !reflect.DeepEqual(again, original) {
				t.Fatal("Get aliases snapshot")
			}
			run.Result.Units[0].BalanceCents = 0
			run.Input.Receipts[0].AmountCents = 1
			receipts, _ := BindAnnualStatementReceiptRepository(sources.Receipts, tenant)
			if _, err := receipts.UpdateAmount(receipt.ID, 20001, "manager@example.com"); err != nil {
				t.Fatal(err)
			}
			next, err := repo.Create(2025, "manager@example.com", time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if next.Revision != 2 || next.ID == original.ID || next.InputHash == original.InputHash || next.Result.TotalCents != 20102 {
				t.Fatalf("next=%+v", next)
			}
			runs, err := repo.List(2025)
			if err != nil || len(runs) != 2 || !reflect.DeepEqual(runs[1], original) {
				t.Fatalf("old run changed: %+v %v", runs, err)
			}
			foreign, _ := BindAnnualStatementRunRepository(storage, testTenantRef("other"))
			if _, found, err := foreign.Get(original.ID); err != nil || found {
				t.Fatal("foreign Get leaked run", err)
			}
			if _, found, err := repo.Get("unknown"); err != nil || found {
				t.Fatal("unknown Get", err)
			}
			if runs, err := foreign.List(2025); err != nil || len(runs) != 0 {
				t.Fatalf("foreign list=%v %v", runs, err)
			}
			if _, err := foreign.Create(2025, "manager@example.com", time.Now()); err == nil {
				t.Fatal("foreign creation accepted")
			}
			documents, _ := BindDocumentRepository(sources.Documents, tenant)
			path, _ := documents.FilePath(document)
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			_, err = repo.Create(2025, "manager@example.com", time.Now())
			var blocked *AnnualStatementRunBlockedError
			if !errors.As(err, &blocked) {
				t.Fatalf("missing original accepted: %v", err)
			}
			runs, _ = repo.List(2025)
			if len(runs) != 2 {
				t.Fatal("blocked run persisted")
			}
		})
	}
}

func TestAnnualStatementRunConsumptionSQL(t *testing.T) {
	_, lanes := testLanes(t)
	docs := NewSQLDocumentStore(lanes, filepath.Join(t.TempDir(), "docs"))
	sources := AnnualStatementRunSources{Periods: NewSQLAnnualStatementPeriodStore(lanes), Units: NewSQLUnitStore(lanes), Documents: docs, Receipts: NewSQLAnnualStatementReceiptStore(lanes), Prepayments: NewSQLAnnualStatementPrepaymentStore(lanes), Consumption: NewSQLAnnualStatementConsumptionStore(lanes)}
	tenant := testTenantRef("demo")
	in := annualRunFixture()
	in.Structure.CostTypes[0].Key = "heizung"
	in.Structure.CostTypes[0].AllocationKey = AllocationKeyVerbrauch
	in.Receipts[0].CostTypeKey = "heizung"
	seedAnnualRunInput(t, sources, tenant, in)
	repo, _ := BindAnnualStatementRunRepository(NewSQLAnnualStatementRunStore(lanes, docs), tenant)
	consumption, _ := BindAnnualStatementConsumptionRepository(sources.Consumption, tenant)
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, mustViennaLocation(t))
	end := start.AddDate(1, 0, 0)
	appendConsumption(t, consumption, consumptionEvidence("a", "heizung", "sensor.a", start, 100, "kWh"))
	appendConsumption(t, consumption, consumptionEvidence("a", "heizung", "sensor.a", end, 101, "kWh"))
	appendConsumption(t, consumption, consumptionEvidence("b", "heizung", "sensor.b", start, 100, "kWh"))
	assertBlocked := func(reason AnnualStatementConsumptionGapReason, wantRuns int) {
		t.Helper()
		input, _, err := repo.Preview(2025, nil)
		var blocked *AnnualStatementRunBlockedError
		if !errors.As(err, &blocked) || !reflect.DeepEqual(blocked.Issues, []AnnualStatementRunIssue{{Code: "consumption", CostTypeKey: "heizung"}}) {
			t.Fatalf("preview must block only consumption: %v", err)
		}
		gaps := input.Consumption["heizung"].Gaps
		if len(gaps) != 1 || gaps[0].Reason != reason {
			t.Fatalf("gaps=%+v want=%s", gaps, reason)
		}
		if _, err := repo.Create(2025, "manager@example.com", time.Now()); !errors.As(err, &blocked) {
			t.Fatalf("new run must be blocked: %v", err)
		}
		if runs, err := repo.List(2025); err != nil || len(runs) != wantRuns {
			t.Fatalf("blocked creation changed history: %d runs, %v", len(runs), err)
		}
	}
	assertBlocked(ConsumptionGapMissingEndEvidence, 0)
	appendConsumption(t, consumption, consumptionEvidence("b", "heizung", "sensor.b", end, 103, "kWh"))
	run, err := repo.Create(2025, "manager@example.com", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	runs, err := repo.List(2025)
	if err != nil || len(runs) != 1 || !reflect.DeepEqual(runs[0], run) {
		t.Fatalf("stored run=%+v err=%v", runs, err)
	}
	stored := runs[0]
	if stored.Result.Units[0].AllocatedCents != 2551 || len(stored.Input.Evidence) != 4 || len(stored.Input.Consumption["heizung"].Units) != 2 {
		t.Fatalf("unexpected stored result/snapshot: %+v", stored)
	}
	replay, issues := CalculateAnnualStatementRun(stored.Input)
	if len(issues) != 0 || !reflect.DeepEqual(replay, stored.Result) {
		t.Fatalf("snapshot replay=%+v issues=%+v", replay, issues)
	}
	// Both boundaries remain valid, but a later interior fact reveals a reset.
	appendConsumption(t, consumption, consumptionEvidence("a", "heizung", "sensor.a", start.Add(24*time.Hour), 1, "kWh"))
	// The display and its readiness preview use the same already-loaded vector;
	// Create must still re-read within its own transaction and refuse this run.
	if _, result, err := repo.Preview(2025, stored.Input.Consumption); err != nil || !reflect.DeepEqual(result, stored.Result) {
		t.Fatalf("preview reloaded its supplied consumption: %+v %v", result, err)
	}
	assertBlocked(ConsumptionGapCounterReset, 1)
	runs, err = repo.List(2025)
	if err != nil || len(runs) != 1 || !reflect.DeepEqual(runs[0], stored) {
		t.Fatalf("interior reset changed stored run: %+v %v", runs, err)
	}
	replay, issues = CalculateAnnualStatementRun(runs[0].Input)
	if len(issues) != 0 || !reflect.DeepEqual(replay, stored.Result) {
		t.Fatalf("historical replay changed: %+v %v", replay, issues)
	}
	// Complete boundary evidence is insufficient without the validated vector.
	withoutVector := runs[0].Input
	withoutVector.Consumption = nil
	if result, issues := CalculateAnnualStatementRun(withoutVector); len(issues) == 0 || len(result.Units) != 0 {
		t.Fatal("calculation rebuilt a vector from boundary-only evidence")
	}
}

func TestAnnualStatementRunPostgresImmutability(t *testing.T) {
	if strings.TrimSpace(os.Getenv("HAUSV_TEST_POSTGRES_DSN")) == "" {
		t.Skip("set HAUSV_TEST_POSTGRES_DSN to check PostgreSQL run immutability")
	}
	t.Setenv("HAUSV_STORE_TEST_POSTGRES", "1")
	database, lanes := testLanes(t)
	docs := NewSQLDocumentStore(lanes, filepath.Join(t.TempDir(), "docs"))
	sources := AnnualStatementRunSources{Periods: NewSQLAnnualStatementPeriodStore(lanes), Units: NewSQLUnitStore(lanes), Documents: docs, Receipts: NewSQLAnnualStatementReceiptStore(lanes), Prepayments: NewSQLAnnualStatementPrepaymentStore(lanes), Consumption: NewSQLAnnualStatementConsumptionStore(lanes)}
	tenant := testTenantRef("demo")
	seedAnnualRunSources(t, sources, tenant)
	repo, _ := BindAnnualStatementRunRepository(NewSQLAnnualStatementRunStore(lanes, docs), tenant)
	run, err := repo.Create(2025, "manager@example.com", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	// Even the maintenance lane must not change a stored calculation.
	for _, query := range []string{
		`UPDATE annual_statement_runs SET data='{}' WHERE tenant_id=$1 AND id=$2`,
		`DELETE FROM annual_statement_runs WHERE tenant_id=$1 AND id=$2`,
	} {
		_, err := database.Exec(query, tenant.ID, run.ID)
		var sqlErr interface{ SQLState() string }
		if !errors.As(err, &sqlErr) || sqlErr.SQLState() != "23514" || !strings.Contains(err.Error(), "annual_statement_runs are immutable") {
			t.Fatalf("immutable trigger did not reject mutation: %v", err)
		}
	}
	runs, err := repo.List(2025)
	if err != nil || len(runs) != 1 || !reflect.DeepEqual(runs[0], run) {
		t.Fatalf("stored run changed: %+v %v", runs, err)
	}
}

func TestAnnualStatementRunSQLiteRestartAndReadFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.db")
	docDir := filepath.Join(t.TempDir(), "docs")
	database, err := appdb.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	seedFixtureTenants(t, database)
	scoped, err := appdb.NewScoped(appdb.Config{Backend: appdb.BackendSQLite, DSN: path}, database)
	if err != nil {
		t.Fatal(err)
	}
	lanes := NewTenantDB(scoped)
	docs := NewSQLDocumentStore(lanes, docDir)
	sources := AnnualStatementRunSources{Periods: NewSQLAnnualStatementPeriodStore(lanes), Units: NewSQLUnitStore(lanes), Documents: docs, Receipts: NewSQLAnnualStatementReceiptStore(lanes), Prepayments: NewSQLAnnualStatementPrepaymentStore(lanes), Consumption: NewSQLAnnualStatementConsumptionStore(lanes)}
	tenant := testTenantRef("demo")
	seedAnnualRunSources(t, sources, tenant)
	repo, _ := BindAnnualStatementRunRepository(NewSQLAnnualStatementRunStore(lanes, docs), tenant)
	original, err := repo.Create(2025, "manager@example.com", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := scoped.Close(); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	database, err = appdb.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	scoped, err = appdb.NewScoped(appdb.Config{Backend: appdb.BackendSQLite, DSN: path}, database)
	if err != nil {
		t.Fatal(err)
	}
	defer scoped.Close()
	lanes = NewTenantDB(scoped)
	repo, _ = BindAnnualStatementRunRepository(NewSQLAnnualStatementRunStore(lanes, NewSQLDocumentStore(lanes, docDir)), tenant)
	runs, err := repo.List(2025)
	if err != nil || len(runs) != 1 || !reflect.DeepEqual(runs[0], original) {
		t.Fatalf("restart result=%+v err=%v", runs, err)
	}
	// A malformed unit row must not be silently dropped from a full-house run.
	if _, err := database.Exec(`UPDATE units SET data='{' WHERE tenant_id=$1 AND id='a'`, tenant.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create(2025, "manager@example.com", time.Now()); err == nil {
		t.Fatal("corrupt unit accepted")
	}
	runs, err = repo.List(2025)
	if err != nil || len(runs) != 1 {
		t.Fatalf("failed read wrote run: %+v %v", runs, err)
	}
}

// Count the repository's actual report load, including the run store's load
// path, rather than only counting calls made by the consumption panel.
type countedRunConsumptionStore struct {
	*MemoryAnnualStatementConsumptionStore
	reports map[string]int
}

func (s *countedRunConsumptionStore) reportAnnualStatementConsumption(tenant TenantRef, year int, cost string, expected []string, start, end time.Time) (AnnualStatementConsumptionReport, error) {
	s.reports[cost]++
	evidence, err := s.listAnnualStatementConsumption(tenant, cost, start, end)
	if err != nil {
		return AnnualStatementConsumptionReport{}, err
	}
	return buildAnnualStatementConsumptionReport(year, cost, expected, start, end, evidence), nil
}

func TestAnnualStatementConsumptionReadsEachUsedKindOnceIncludingRunLoad(t *testing.T) {
	periods := NewMemoryAnnualStatementPeriodStore()
	units, err := NewUnitStore("")
	if err != nil {
		t.Fatal(err)
	}
	docs, err := NewDocumentStore("", filepath.Join(t.TempDir(), "docs"))
	if err != nil {
		t.Fatal(err)
	}
	counted := &countedRunConsumptionStore{NewMemoryAnnualStatementConsumptionStore(), map[string]int{}}
	sources := AnnualStatementRunSources{Periods: periods, Units: units, Documents: docs, Receipts: NewMemoryAnnualStatementReceiptStore(periods, NewMemoryAnnualStatementCostTypeStore(), docs), Prepayments: NewMemoryAnnualStatementPrepaymentStore(periods, units), Consumption: counted}
	in := annualRunFixture()
	for i, key := range []string{"heizung", "warmwasser"} {
		in.Structure.CostTypes[i].Key = key
		in.Structure.CostTypes[i].AllocationKey = AllocationKeyVerbrauch
		in.Receipts[i].CostTypeKey = key
	}
	tenant := testTenantRef("demo")
	seedAnnualRunInput(t, sources, tenant, in)
	consumption, _ := BindAnnualStatementConsumptionRepository(counted, tenant)
	location := mustViennaLocation(t)
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, location)
	vectors := map[string]AnnualStatementConsumptionVector{}
	for _, key := range []string{"heizung", "warmwasser"} {
		for _, unit := range []string{"a", "b"} {
			appendConsumption(t, consumption, consumptionEvidence(unit, key, "sensor."+key+unit, start, 100, "kWh"))
			appendConsumption(t, consumption, consumptionEvidence(unit, key, "sensor."+key+unit, start.AddDate(1, 0, 0), 101, "kWh"))
		}
		// This is the page's display read, using the same source as the run.
		report, err := consumption.ConsumptionReport(in.Period, key, []string{"a", "b"}, location)
		if err != nil {
			t.Fatal(err)
		}
		vectors[key] = report.Vector
	}
	repo, _ := BindAnnualStatementRunRepository(NewMemoryAnnualStatementRunStore(sources), tenant)
	if _, _, err := repo.Preview(2025, vectors); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(counted.reports, map[string]int{"heizung": 1, "warmwasser": 1}) {
		t.Fatalf("panel + readiness load queried reports more than once: %+v", counted.reports)
	}
	if _, err := repo.Create(2025, "manager@example.com", time.Now()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(counted.reports, map[string]int{"heizung": 2, "warmwasser": 2}) {
		t.Fatalf("creation did not load each current report once: %+v", counted.reports)
	}
}

type countedAnnualRunQueries struct {
	*sql.Tx
	documents int
	reports   map[string]int
}

func (q *countedAnnualRunQueries) count(query string, args []any) {
	if strings.Contains(query, "FROM documents") {
		q.documents++
	}
	if strings.Contains(query, "FROM annual_statement_consumption_evidence") {
		q.reports[args[1].(string)]++
	}
}
func (q *countedAnnualRunQueries) Query(query string, args ...any) (*sql.Rows, error) {
	q.count(query, args)
	return q.Tx.Query(query, args...)
}
func (q *countedAnnualRunQueries) QueryRow(query string, args ...any) *sql.Row {
	q.count(query, args)
	return q.Tx.QueryRow(query, args...)
}

func TestAnnualStatementRunSQLBatchesReadinessInputs(t *testing.T) {
	_, lanes := testLanes(t)
	docs := NewSQLDocumentStore(lanes, filepath.Join(t.TempDir(), "docs"))
	sources := AnnualStatementRunSources{Periods: NewSQLAnnualStatementPeriodStore(lanes), Units: NewSQLUnitStore(lanes), Documents: docs, Receipts: NewSQLAnnualStatementReceiptStore(lanes), Prepayments: NewSQLAnnualStatementPrepaymentStore(lanes), Consumption: NewSQLAnnualStatementConsumptionStore(lanes)}
	tenant := testTenantRef("demo")
	in := annualRunFixture()
	for i, key := range []string{"heizung", "warmwasser"} {
		in.Structure.CostTypes[i].Key = key
		in.Structure.CostTypes[i].AllocationKey = AllocationKeyVerbrauch
		in.Receipts[i].CostTypeKey = key
	}
	seedAnnualRunInput(t, sources, tenant, in)
	consumption, _ := BindAnnualStatementConsumptionRepository(sources.Consumption, tenant)
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, mustViennaLocation(t))
	end := start.AddDate(1, 0, 0)
	for _, key := range []string{"heizung", "warmwasser"} {
		for _, unit := range []string{"a", "b"} {
			appendConsumption(t, consumption, consumptionEvidence(unit, key, "sensor."+key+unit, start, 100, "kWh"))
			appendConsumption(t, consumption, consumptionEvidence(unit, key, "sensor."+key+unit, end, 101, "kWh"))
		}
	}
	storage := NewSQLAnnualStatementRunStore(lanes, docs)
	tx, err := storage.begin(tenant, true)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	counted := &countedAnnualRunQueries{Tx: tx, reports: map[string]int{}}
	vectors := map[string]AnnualStatementConsumptionVector{}
	for _, key := range []string{"heizung", "warmwasser"} {
		// Count the panel query and the run loader against a real SQL database.
		report, err := queryAnnualStatementConsumptionReport(counted, tenant, 2025, key, []string{"a", "b"}, start, end)
		if err != nil {
			t.Fatal(err)
		}
		vectors[key] = report.Vector
	}
	input, err := storage.load(counted, tenant, 2025, vectors)
	if err != nil {
		t.Fatal(err)
	}
	if len(input.Documents) != 3 || counted.documents != 1 || !reflect.DeepEqual(counted.reports, map[string]int{"heizung": 1, "warmwasser": 1}) {
		t.Fatalf("documents=%d queries=%d report queries=%+v", len(input.Documents), counted.documents, counted.reports)
	}
	if _, issues := CalculateAnnualStatementRun(input); len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	// The authoritative load used by Create still queries each report once.
	if _, err := storage.load(counted, tenant, 2025, nil); err != nil {
		t.Fatal(err)
	}
	if counted.documents != 2 || !reflect.DeepEqual(counted.reports, map[string]int{"heizung": 2, "warmwasser": 2}) {
		t.Fatalf("fresh load queries: documents=%d reports=%+v", counted.documents, counted.reports)
	}
}

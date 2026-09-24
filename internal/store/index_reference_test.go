package store

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/indexation"
)

func indexRelease(t *testing.T, period, label, value string, at time.Time) indexation.FetchedOGD {
	t.Helper()
	snapshot, err := publishedIndices()
	if err != nil {
		t.Fatal(err)
	}
	var source indexation.SnapshotSource
	for _, v := range snapshot.Manifest.Sources {
		if v.Series == indexation.VPI2020 {
			source = v
		}
	}
	data := "C-VPIZR-0;C-VPICOICOP18_5-0;F-VPIMZBM\nVPIZR-" + period + ";VPICOICOP18-0;" + value + "\n"
	labels := "code;name\nVPIZR-" + period + ";" + label + "\n"
	fetched, err := indexation.ParseOGDRelease([]byte(data), []byte(labels), source, at)
	if err != nil {
		t.Fatal(err)
	}
	return fetched
}

func TestIndexRuntimeRevisionsConflictsAndPrecedence(t *testing.T) {
	database, lanes := testLanes(t)
	service := NewIndexReferenceStore(lanes)
	at := mustDate("2026-10-20")
	apply := func(period, label, value string) IndexImport {
		t.Helper()
		got, err := service.Apply(t.Context(), indexRelease(t, period, label, value, at), "admin@example.com")
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	first := apply("202609", "Sep.26 (vorl.)", "133,2")
	if first.RowsAdded != 1 || first.RowsRevised != 0 {
		t.Fatalf("new month: %+v", first)
	}
	if duplicate := apply("202609", "Sep.26 (vorl.)", "133,2"); !duplicate.Duplicate {
		t.Fatal("non-idempotent import")
	}
	revised := apply("202609", "Sep.26 (vorl.)", "133,3")
	if revised.RowsRevised != 1 {
		t.Fatalf("preliminary revision: %+v", revised)
	}
	final := apply("202609", "Sep.26", "133,3")
	if final.RowsRevised != 1 {
		t.Fatal("status-only revision lost")
	}
	conflict := apply("202609", "Sep.26", "133,4")
	if conflict.RowsFlagged != 1 || conflict.RowsRevised != 0 {
		t.Fatalf("final overwrite: %+v", conflict)
	}
	if downgrade := apply("202609", "Sep.26 (vorl.)", "133,4"); downgrade.RowsFlagged != 1 {
		t.Fatal("final downgraded")
	}
	// A changed embedded preliminary value becomes the first linked revision.
	if embedded := apply("202608", "Aug.26", "133,0"); embedded.RowsRevised != 1 {
		t.Fatalf("embedded revision: %+v", embedded)
	}
	// Already final embedded history is protected, including before first import.
	if embedded := apply("202607", "Jul.26", "1,0"); embedded.RowsFlagged != 1 {
		t.Fatal("embedded final overwritten")
	}
	snapshot, err := runtimeIndexSnapshot(lanes.For(testTenantRef("demo")))
	if err != nil {
		t.Fatal(err)
	}
	for month, want := range map[indexation.Month]string{"2026-09": "133.3", "2026-08": "133.0", "2024-09": "123.6"} {
		v, ok, e := snapshot.Data.Lookup(indexation.VPI2020, month)
		if e != nil || !ok || v.Value.String() != want || v.Preliminary {
			t.Fatalf("lookup %s: %+v %v", month, v, e)
		}
	}
	var total, active, linked int
	if err = database.QueryRow(`SELECT COUNT(*),COUNT(CASE WHEN superseded_by IS NULL THEN 1 END),COUNT(superseded_by) FROM index_values WHERE period='2026-09'`).Scan(&total, &active, &linked); err != nil {
		t.Fatal(err)
	}
	if total != 3 || active != 1 || linked != 2 {
		t.Fatalf("history overwritten: %d %d %d", total, active, linked)
	}
	var imports int
	if err = database.QueryRow(`SELECT COUNT(*) FROM index_imports`).Scan(&imports); err != nil {
		t.Fatal(err)
	}
	if imports != 7 {
		t.Fatalf("imports=%d", imports)
	}
	other, err := runtimeIndexSnapshot(lanes.For(testTenantRef("other")))
	if err != nil || other.Manifest.DataSHA256 != snapshot.Manifest.DataSHA256 {
		t.Fatal("reference data was tenant scoped", err)
	}
	// Overlay assembly never mutates the process-wide embedded fallback.
	embedded, _ := publishedIndices()
	v, _, _ := embedded.Data.Lookup(indexation.VPI2020, "2026-08")
	if !v.Preliminary || v.Value.String() != "132.9" {
		t.Fatal("embedded fallback mutated")
	}
}

func TestIndexImportConcurrentIdempotency(t *testing.T) {
	database, lanes := testLanes(t)
	service := NewIndexReferenceStore(lanes)
	release := indexRelease(t, "202609", "Sep.26 (vorl.)", "133,2", mustDate("2026-10-20"))
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Go(func() { _, err := service.Apply(t.Context(), release, "system:index-refresh"); errs <- err })
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var n int
	if err := database.QueryRow(`SELECT COUNT(*) FROM index_imports`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("duplicate imports=%d %v", n, err)
	}
}

func TestIndexRevisionIdentitySurvivesReturnToOriginalValue(t *testing.T) {
	_, lanes := testLanes(t)
	service := NewIndexReferenceStore(lanes)
	release := indexRelease(t, "202609", "Sep.26 (vorl.)", "133,2", mustDate("2026-10-20"))
	if _, err := service.Apply(t.Context(), release, "system:index-refresh"); err != nil {
		t.Fatal(err)
	}
	values, err := currentIndexObservations(lanes.For(testTenantRef("demo")))
	if err != nil {
		t.Fatal(err)
	}
	frozen := ValorisationRun{IndexSnapshot: []ValorisationIndex{{Series: "VPI2020", Period: "2026-09", Value: "133.2", Status: "preliminary", RevisionID: values["VPI2020/2026-09"].ID}}}
	if _, err = service.Apply(t.Context(), indexRelease(t, "202609", "Sep.26 (vorl.)", "133,3", mustDate("2026-10-21")), "system:index-refresh"); err != nil {
		t.Fatal(err)
	}
	// A new publication can restore a previous number. Its revision identity
	// remains different even when the value and status happen to match again.
	restored, err := indexation.ParseOGDRelease([]byte("C-VPIZR-0;C-VPICOICOP18_5-0;F-VPIMZBM\nVPIZR-202609;VPICOICOP18-0;133,2\n\n"), []byte("code;name\nVPIZR-202609;Sep.26 (vorl.)\n"), indexation.SnapshotSource{Series: indexation.VPI2020, URL: release.URL}, mustDate("2026-10-22"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Apply(t.Context(), restored, "system:index-refresh"); err != nil {
		t.Fatal(err)
	}
	if revised, err := indexRunRevised(lanes.For(testTenantRef("demo")), frozen); err != nil || !revised {
		t.Fatal("revision history lost", err)
	}
}

func TestIndexRevisionFlagsFrozenRunsWithoutChangingApproval(t *testing.T) {
	database, lanes := testLanes(t)
	tenant := testTenantRef("demo")
	if _, err := database.Exec(`INSERT INTO units(tenant_id,tenant_slug,id,data) VALUES($1,$2,'top-1','{}')`, tenant.ID, tenant.Slug); err != nil {
		t.Fatal(err)
	}
	leases, _ := BindLeaseRepository(NewSQLLeaseStore(lanes), tenant)
	if _, err := leases.Create(valorisationFixture()); err != nil {
		t.Fatal(err)
	}
	repo, _ := BindValorisationRepository(lanes, NewSQLDocumentStore(lanes, t.TempDir()), tenant)
	actor := ValorisationActor{Email: "admin@example.com", Manage: true, Approve: true}
	input := ValorisationInput{EffectiveOn: "2026-04-01", Settings: DefaultValorisationSettings()}
	draft, err := repo.Create(input, "org", actor, mustDate("2026-04-01"))
	if err != nil {
		t.Fatal(err)
	}
	approved, err := repo.Approve(draft.ID, actor, input.Settings, mustDate("2026-04-01"), func(ValorisationRun, ValorisationItem, time.Time) ([]byte, error) { return []byte("%PDF fixture"), nil })
	if err != nil {
		t.Fatal(err)
	}
	service := NewIndexReferenceStore(lanes)
	// Unrelated new month must not flag the frozen April run.
	if _, err = service.Apply(t.Context(), indexRelease(t, "202609", "Sep.26 (vorl.)", "133,2", mustDate("2026-10-20")), actor.Email); err != nil {
		t.Fatal(err)
	}
	unchanged, _, err := repo.Get(draft.ID)
	if err != nil || unchanged.IndexRevised {
		t.Fatal("unrelated month flagged", err)
	}
	// A final correction is held for review, and warns on its dependent run.
	if _, err = service.Apply(t.Context(), indexRelease(t, "202512", "Dez.25", "130,0", mustDate("2026-10-20")), actor.Email); err != nil {
		t.Fatal(err)
	}
	flagged, _, err := repo.Get(draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !flagged.IndexRevised || flagged.Status != "approved" || flagged.InputsSHA256 != approved.InputsSHA256 || flagged.Items[0].NewCents != approved.Items[0].NewCents || flagged.Items[0].LetterSHA256 != approved.Items[0].LetterSHA256 {
		t.Fatalf("frozen approval changed or warning absent: %+v", flagged)
	}
	// A run containing a preliminary input is affected by a status-only revision.
	observations, err := currentIndexObservations(lanes.For(tenant))
	if err != nil {
		t.Fatal(err)
	}
	before := ValorisationRun{IndexSnapshot: []ValorisationIndex{{RevisionID: observations["VPI2020/2026-09"].ID, Series: "VPI2020", Period: "2026-09", Value: "133.2", Status: "preliminary"}}}
	if revised, err := indexRunRevised(lanes.For(tenant), before); err != nil || revised {
		t.Fatal("unchanged preliminary flagged", err)
	}
	if _, err = service.Apply(t.Context(), indexRelease(t, "202609", "Sep.26", "133,2", mustDate("2026-11-20")), actor.Email); err != nil {
		t.Fatal(err)
	}
	if revised, err := indexRunRevised(lanes.For(tenant), before); err != nil || !revised {
		t.Fatal("status revision not flagged", err)
	}
}

func TestRuntimeFinalPublicationBeyondCalendar(t *testing.T) {
	_, lanes := testLanes(t)
	service := NewIndexReferenceStore(lanes)
	release := indexRelease(t, "202701", "Jän.27", "135,0", mustDate("2027-03-20"))
	if _, err := service.Apply(t.Context(), release, "system:index-refresh"); err != nil {
		t.Fatal(err)
	}
	snapshot, err := runtimeIndexSnapshot(lanes.For(testTenantRef("demo")))
	if err != nil {
		t.Fatal(err)
	}
	date, source, ok := snapshotIndexPublication(snapshot, indexation.VPI2020, "2027-01")
	if !ok || date.Format(time.DateOnly) != "2027-03-20" || !strings.HasSuffix(source, "_C-VPIZR-0.csv") {
		t.Fatal(fmt.Sprint(date, source, ok))
	}
}

func TestRuntimeValuesReachEngineAndFlagPersistedDraft(t *testing.T) {
	database, lanes := testLanes(t)
	tenant := testTenantRef("demo")
	if _, err := database.Exec(`INSERT INTO units(tenant_id,tenant_slug,id,data) VALUES($1,$2,'top-1','{}')`, tenant.ID, tenant.Slug); err != nil {
		t.Fatal(err)
	}
	lease := valorisationFixture()
	lease.UseKind, lease.MRGScope = UseKindGeschaeft, MRGAusnahme
	lease.Clauses[0].BasePeriod, lease.Clauses[0].BaseValue = "2026-08", "132.9"
	lease.Clauses[0].ThresholdValue = "1"
	leases, _ := BindLeaseRepository(NewSQLLeaseStore(lanes), tenant)
	if _, err := leases.Create(lease); err != nil {
		t.Fatal(err)
	}
	repo, _ := BindValorisationRepository(lanes, NewSQLDocumentStore(lanes, t.TempDir()), tenant)
	actor := ValorisationActor{Email: "admin@example.com", Manage: true, Approve: true}
	input := ValorisationInput{EffectiveOn: "2026-12-01", Settings: DefaultValorisationSettings()}
	old, err := repo.Create(input, "org", actor, mustDate("2026-12-01"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsValorisation(old.Items[0].Exceptions, "index_preliminary") {
		t.Fatal("expected preliminary base")
	}
	service := NewIndexReferenceStore(lanes)
	for _, release := range []indexation.FetchedOGD{
		indexRelease(t, "202608", "Aug.26", "132,9", mustDate("2026-11-20")),
		indexRelease(t, "202609", "Sep.26", "140,0", mustDate("2026-11-20")),
	} {
		if _, err = service.Apply(t.Context(), release, actor.Email); err != nil {
			t.Fatal(err)
		}
	}
	frozen, _, err := repo.Get(old.ID)
	if err != nil || !frozen.IndexRevised || frozen.InputsSHA256 != old.InputsSHA256 {
		t.Fatal("stored draft revision warning", err)
	}
	if _, err = repo.Approve(old.ID, actor, input.Settings, mustDate("2026-12-01"), func(ValorisationRun, ValorisationItem, time.Time) ([]byte, error) { return []byte("%PDF"), nil }); err == nil || !strings.Contains(err.Error(), "index_revised") {
		t.Fatal("stale draft approval", err)
	}
	newRun, err := repo.Create(input, "org", actor, mustDate("2026-12-01"))
	if err != nil {
		t.Fatal(err)
	}
	if newRun.Revision != old.Revision+1 || newRun.Items[0].Group != "ready" || newRun.Items[0].Contract.TriggerMonth != "2026-09" || newRun.Items[0].NewCents <= 100000 {
		t.Fatalf("runtime values did not reach engine: %+v", newRun.Items[0])
	}
}

func TestRuntimeAnnualAverageRevision(t *testing.T) {
	_, lanes := testLanes(t)
	snapshot, _ := publishedIndices()
	var source indexation.SnapshotSource
	for _, s := range snapshot.Manifest.Sources {
		if s.Series == indexation.VPI2020 {
			source = s
		}
	}
	service := NewIndexReferenceStore(lanes)
	for _, preliminary := range []bool{true, false} {
		label := "Jahresdurchschnitt 2026"
		if preliminary {
			label += " (vorl.)"
		}
		data := []byte("C-VPIZR-0;C-VPICOICOP18_5-0;F-VPIMZBM\nVPIZR-202612;VPICOICOP18-0;134,2\nVPIZR-2026;VPICOICOP18-0;132,4\n")
		labels := []byte("code;name\nVPIZR-202612;Dez.26 (vorl.)\nVPIZR-2026;" + label + "\n")
		fetched, err := indexation.ParseOGDRelease(data, labels, source, mustDate("2027-01-20"))
		if err != nil {
			t.Fatal(err)
		}
		result, err := service.Apply(t.Context(), fetched, "system:index-refresh")
		if err != nil {
			t.Fatal(err)
		}
		if !preliminary && result.RowsRevised != 1 {
			t.Fatal("annual finality lost")
		}
	}
	current, err := runtimeIndexSnapshot(lanes.For(testTenantRef("demo")))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, annual := range current.Annual {
		if annual.Series == indexation.VPI2020 && annual.Year == 2026 {
			count++
			if annual.Preliminary || annual.Value.String() != "132.4" {
				t.Fatal(annual)
			}
		}
	}
	if count != 1 {
		t.Fatal("annual average missing or duplicated")
	}
}

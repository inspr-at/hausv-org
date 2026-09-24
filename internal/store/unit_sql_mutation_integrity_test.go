package store

import (
	"errors"
	"strings"
	"sync"
	"testing"

	appdb "github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
)

func TestCorruptUnitSurvivesUnrelatedEdit(t *testing.T) {
	pool, lanes := testLanes(t)
	repo, ok := BindUnitRepository(NewSQLUnitStore(lanes), testTenantRef("demo"))
	if !ok {
		t.Fatal("bind demo repository")
	}
	ref := testTenantRef("demo")
	if err := repo.SetUnits([]Unit{
		{ID: "top-1", TenantSlug: "demo", Label: "Top 1", UnitType: UnitTypeResidential, MiteigentumsanteilPPM: 400000, UsableAreaRecorded: true, UsableAreaM2Hundredths: 5000},
		{ID: "top-2", TenantSlug: "demo", Label: "Top 2", UnitType: UnitTypeResidential, MiteigentumsanteilPPM: 600000, UsableAreaRecorded: true, UsableAreaM2Hundredths: 8000},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(`UPDATE units SET data='invalid-json' WHERE tenant_id=$1 AND id='top-2'`, ref.ID); err != nil {
		t.Fatal(err)
	}

	dup, err := repo.UpsertUnit("top-1", Unit{ID: "top-1", TenantSlug: "demo", Label: "Changed", UnitType: UnitTypeResidential, MiteigentumsanteilPPM: 400000})
	if dup {
		t.Fatal("unrelated edit reported a duplicate")
	}
	var raw string
	if scanErr := pool.QueryRow(`SELECT data FROM units WHERE tenant_id=$1 AND id='top-2'`, ref.ID).Scan(&raw); scanErr != nil {
		t.Fatalf("corrupt unit was deleted: %v", scanErr)
	}
	if raw != "invalid-json" {
		t.Fatalf("corrupt payload changed: %q", raw)
	}
	if err != nil {
		var dataErr *UnitDataError
		if !errors.As(err, &dataErr) {
			t.Fatalf("edit failed without a unit data error: %v", err)
		}
		var label string
		if scanErr := pool.QueryRow(`SELECT data FROM units WHERE tenant_id=$1 AND id='top-1'`, ref.ID).Scan(&label); scanErr != nil {
			t.Fatal(scanErr)
		}
		if label == "" || !strings.Contains(label, `"label":"Top 1"`) {
			t.Fatalf("failed edit still changed top-1: %s", label)
		}
		return
	}

	var changed string
	if scanErr := pool.QueryRow(`SELECT data FROM units WHERE tenant_id=$1 AND id='top-1'`, ref.ID).Scan(&changed); scanErr != nil {
		t.Fatal(scanErr)
	}
	if !strings.Contains(changed, `"label":"Changed"`) {
		t.Fatalf("narrow edit did not persist: %s", changed)
	}
	if !strings.Contains(changed, `"miteigentumsanteil":400000`) {
		t.Fatalf("allocation share changed: %s", changed)
	}
}

func TestSetUnitsRefusesPartialRead(t *testing.T) {
	pool, lanes := testLanes(t)
	repo, ok := BindUnitRepository(NewSQLUnitStore(lanes), testTenantRef("demo"))
	if !ok {
		t.Fatal("bind demo repository")
	}
	ref := testTenantRef("demo")
	if err := repo.SetUnits([]Unit{
		{ID: "top-1", Label: "Top 1"},
		{ID: "top-2", Label: "Top 2"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(`UPDATE units SET data='invalid-json' WHERE tenant_id=$1 AND id='top-2'`, ref.ID); err != nil {
		t.Fatal(err)
	}
	err := repo.SetUnits([]Unit{{ID: "top-1", Label: "Only one"}})
	var dataErr *UnitDataError
	if !errors.As(err, &dataErr) || dataErr.Op != "decode" || dataErr.UnitID != "top-2" {
		t.Fatalf("set units = %v, want decode error for top-2", err)
	}
	var count int
	if err := pool.QueryRow(`SELECT count(*) FROM units WHERE tenant_id=$1`, ref.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("refused replace deleted rows: count=%d", count)
	}
}

func TestStaleSnapshotEditsBothSurvive(t *testing.T) {
	_, lanes := testLanes(t)
	repo, ok := BindUnitRepository(NewSQLUnitStore(lanes), testTenantRef("demo"))
	if !ok {
		t.Fatal("bind demo repository")
	}
	if err := repo.SetUnits([]Unit{
		{ID: "top-1", Label: "Top 1", UsableAreaRecorded: true, UsableAreaM2Hundredths: 5000, PersonsRecorded: true, Persons: 2},
		{ID: "top-2", Label: "Top 2", UsableAreaRecorded: true, UsableAreaM2Hundredths: 8000, PersonsRecorded: true, Persons: 3},
	}); err != nil {
		t.Fatal(err)
	}
	// Each edit carries only its own unit, as a writer would after reading a
	// snapshot that does not include the other writer's change.
	if dup, err := repo.UpsertUnit("top-1", Unit{ID: "top-1", Label: "Top 1 edited", UsableAreaRecorded: true, UsableAreaM2Hundredths: 5000, PersonsRecorded: true, Persons: 2}); err != nil || dup {
		t.Fatalf("edit top-1: dup=%v err=%v", dup, err)
	}
	if dup, err := repo.UpsertUnit("top-2", Unit{ID: "top-2", Label: "Top 2 edited", UsableAreaRecorded: true, UsableAreaM2Hundredths: 8000, PersonsRecorded: true, Persons: 3}); err != nil || dup {
		t.Fatalf("edit top-2: dup=%v err=%v", dup, err)
	}
	got := map[string]Unit{}
	for _, item := range repo.List() {
		got[item.ID] = item
	}
	if got["top-1"].Label != "Top 1 edited" || got["top-1"].UsableAreaM2Hundredths != 5000 || got["top-1"].Persons != 2 {
		t.Fatalf("top-1 = %+v", got["top-1"])
	}
	if got["top-2"].Label != "Top 2 edited" || got["top-2"].UsableAreaM2Hundredths != 8000 || got["top-2"].Persons != 3 {
		t.Fatalf("top-2 = %+v", got["top-2"])
	}
}

func TestConcurrentUnitEditsBothSurvive(t *testing.T) {
	if dbtest.Backend() != appdb.BackendPostgres {
		t.Skip("concurrent lost-update is the PostgreSQL replacement race")
	}
	_, lanes := testLanes(t)
	repo, ok := BindUnitRepository(NewSQLUnitStore(lanes), testTenantRef("demo"))
	if !ok {
		t.Fatal("bind demo repository")
	}
	if err := repo.SetUnits([]Unit{
		{ID: "top-1", Label: "Top 1"},
		{ID: "top-2", Label: "Top 2"},
	}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		if _, err := repo.UpsertUnit("top-1", Unit{ID: "top-1", Label: "from A"}); err != nil {
			errCh <- err
		}
	}()
	go func() {
		defer wg.Done()
		if _, err := repo.UpsertUnit("top-2", Unit{ID: "top-2", Label: "from B"}); err != nil {
			errCh <- err
		}
	}()
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("concurrent edit: %v", err)
	}
	got := map[string]string{}
	for _, item := range repo.List() {
		got[item.ID] = item.Label
	}
	if got["top-1"] != "from A" || got["top-2"] != "from B" {
		t.Fatalf("labels = %v", got)
	}
}

func TestNarrowAllocationUpdateLeavesSibling(t *testing.T) {
	pool, lanes := testLanes(t)
	repo, ok := BindUnitRepository(NewSQLUnitStore(lanes), testTenantRef("demo"))
	if !ok {
		t.Fatal("bind demo repository")
	}
	ref := testTenantRef("demo")
	if err := repo.SetUnits([]Unit{
		{ID: "top-1", Label: "Top 1", MiteigentumsanteilPPM: 400000},
		{ID: "top-2", Label: "Top 2", MiteigentumsanteilPPM: 600000, UsableAreaRecorded: true, UsableAreaM2Hundredths: 8000},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(`UPDATE units SET data='invalid-json' WHERE tenant_id=$1 AND id='top-2'`, ref.ID); err != nil {
		t.Fatal(err)
	}
	unknown, err := repo.UpdateAllocationBases([]UnitAllocationBasisUpdate{{
		UnitID: "top-1", UsableAreaRecorded: true, UsableAreaM2Hundredths: 4200, PersonsRecorded: true, Persons: 1,
	}})
	if err != nil || unknown {
		t.Fatalf("allocation update: unknown=%v err=%v", unknown, err)
	}
	var raw string
	if err := pool.QueryRow(`SELECT data FROM units WHERE tenant_id=$1 AND id='top-2'`, ref.ID).Scan(&raw); err != nil {
		t.Fatalf("sibling missing: %v", err)
	}
	if raw != "invalid-json" {
		t.Fatalf("sibling payload = %q", raw)
	}
	updated := repo.MembersForUnit("top-1")
	if !updated.Found || updated.Unit.UsableAreaM2Hundredths != 4200 || updated.Unit.Persons != 1 || updated.Unit.MiteigentumsanteilPPM != 400000 {
		t.Fatalf("updated unit = %+v", updated.Unit)
	}
}

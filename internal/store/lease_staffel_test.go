package store

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/indexation"
)

func staffelFixture(t *testing.T) Lease {
	t.Helper()
	l := valorisationFixture()
	steps, err := ParseStaffel("2026-04-01:1100,00;2027-04-01:2%")
	if err != nil {
		t.Fatal(err)
	}
	l.Clauses[0] = IndexClause{ID: "clause-1", ComponentKind: ComponentHMZ, ClauseType: ClauseStaffel, TwoWay: true, ReviewStatus: ReviewOK, ValidFrom: l.StartsOn, StaffelSteps: steps}
	return l
}

func TestStaffelPreviewCapAndContractTiming(t *testing.T) {
	l := staffelFixture(t)
	got := previewTest(t, l, "2026-04-01")
	if got.Group != "ready" || got.ContractCents != 110000 || got.NewCents != 104028 || got.Decision.CappedCents != 5972 || got.CollectableFrom != "2026-04-05" {
		t.Fatalf("partial %+v", got)
	}
	if !strings.Contains(strings.Join(got.Explanation, " "), "Staffelmietzins laut Vertrag") {
		t.Fatal("missing German explanation")
	}
	l.MRGScope = MRGVoll
	l.PriceRestricted = true
	got = previewTest(t, l, "2026-04-01")
	if got.NewCents != 101735 || got.CollectableFrom != "2026-05-05" {
		t.Fatalf("full %+v", got)
	}
	l.MRGScope = MRGAusnahme
	l.PriceRestricted = false
	l.Clauses[0].StaffelSteps[0].EffectiveOn = "2026-02-12"
	got = previewTest(t, l, "2026-04-01")
	if got.NewCents != 110000 || got.WirksamOn != "2026-02-12" || len(got.Indices) != 0 {
		t.Fatalf("exempt %+v", got)
	}
	l.UseKind = UseKindGeschaeft
	l.MRGScope = MRGVoll
	got = previewTest(t, l, "2026-04-01")
	if got.NewCents != 110000 || got.WirksamOn != "2026-02-12" || got.CollectableFrom != "2026-03-05" {
		t.Fatalf("commercial %+v", got)
	}
}

func TestStaffelAprilCutoffAndLaterIndependentCurves(t *testing.T) {
	l := staffelFixture(t)
	snap, err := indexation.LoadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	preview := func(date string, prior map[string]ValorisationItem) ValorisationItem {
		t.Helper()
		run, err := PreviewValorisation(ValorisationInput{EffectiveOn: date, Leases: []Lease{l}, Prior: prior}, snap, mustDate(date))
		if err != nil {
			t.Fatal(err)
		}
		return run.Items[0]
	}
	l.Clauses[0].StaffelSteps[0].EffectiveOn = "2026-05-01"
	got := preview("2026-06-01", nil)
	if got.Group != "unchanged" || got.NewCents != 100000 {
		t.Fatalf("May leaked into April: %+v", got)
	}
	l.Clauses[0].StaffelSteps[0].EffectiveOn = "2026-02-01"
	got = preview("2026-03-01", nil)
	if got.NewCents != 100000 || got.Decision.DeferredCents != 4028 || got.WirksamOn != "2026-04-01" {
		t.Fatalf("deferred %+v", got)
	}
	first := preview("2026-04-01", nil)
	l.Components = append(l.Components, RentComponent{Kind: ComponentHMZ, NetCents: first.NewCents, ValidFrom: first.WirksamOn})
	l.Clauses[0].State = &ValorisationState{CapValue: "1040.28", CapAnchorPeriod: string(first.CapAnchor), LastEffectiveOn: first.WirksamOn}
	p, _ := indexation.ParseDecimal("130.0")
	snap.Annual = append(snap.Annual, indexation.AnnualValue{Series: indexation.VPI2020, Year: 2026, Value: p})
	second := preview("2027-04-01", map[string]ValorisationItem{l.Clauses[0].ID: first})
	if second.Group != "ready" || second.ContractCents != 112200 || second.CapStartCents != 100000 || second.NewCents <= first.NewCents || second.NewCents >= 112200 {
		t.Fatalf("independent curve lost %+v", second)
	}
	// No new scheduled step is needed to admit part of an existing capped curve.
	l.Clauses[0].StaffelSteps = l.Clauses[0].StaffelSteps[:1]
	second = preview("2027-04-01", map[string]ValorisationItem{l.Clauses[0].ID: first})
	if second.ContractCents != 110000 || second.NewCents <= first.NewCents {
		t.Fatalf("carry lost %+v", second)
	}
}

func TestStaffelCSVValidation(t *testing.T) {
	for _, raw := range []string{"", "Text aus Vertrag", "2026-02-30:1050,00", "2026-04-01:-1", "2026-04-01:0", "2026-04-01:1.001", "2026-04-01:1.050", "2026-04-01:999999999999999999", "2026-04-01:2%;2026-04-01:1100", "2027-04-01:2%;2026-04-01:1100", "2026-04-01:2%;"} {
		if _, err := ParseStaffel(raw); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
	csv := "einheit;vertragsabschluss;beginn;nutzung;mrg;regime;hmz_netto;klausel_typ;staffel\nTop 1;2025-01-01;2025-02-01;wohnung;teil;frei;1000;staffel;\"2026-04-01:1.050,00;2027-04-01:2,5%\"\n"
	drafts, err := ParseLeaseCSV([]byte(csv))
	if err != nil {
		t.Fatal(err)
	}
	l, row := leaseFromDraft(drafts[0], "top-1")
	if len(row.Errors) != 0 || len(l.Clauses[0].StaffelSteps) != 2 || *l.Clauses[0].StaffelSteps[0].NetCents != 105000 || l.Clauses[0].StaffelSteps[1].Percent != "2.5" {
		t.Fatalf("%+v %+v", l, row)
	}
	drafts[0].Staffel = "broken"
	_, row = leaseFromDraft(drafts[0], "top-1")
	if !strings.Contains(strings.Join(row.Errors, ","), "invalid_staffel") {
		t.Fatal(row)
	}
}

func TestStaffelPersistenceAndFrozenApproval(t *testing.T) {
	database, lanes := testLanes(t)
	tenant := testTenantRef("demo")
	if _, err := database.Exec(`INSERT INTO units(tenant_id,tenant_slug,id,data) VALUES($1,$2,'top-1','{}')`, tenant.ID, tenant.Slug); err != nil {
		t.Fatal(err)
	}
	leases, _ := BindLeaseRepository(NewSQLLeaseStore(lanes), tenant)
	l, err := leases.Create(staffelFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	loaded, _, err := leases.Get(l.ID)
	if err != nil || !reflect.DeepEqual(loaded.Clauses[0].StaffelSteps, l.Clauses[0].StaffelSteps) {
		t.Fatalf("roundtrip %v %+v", err, loaded)
	}
	foreign, _ := BindLeaseRepository(NewSQLLeaseStore(lanes), testTenantRef("other"))
	if _, found, err := foreign.Get(l.ID); err != nil || found {
		t.Fatal("tenant leak", err)
	}
	if _, err := foreign.SetClause(l.Clauses[0]); err == nil {
		t.Fatal("foreign write accepted")
	}
	repo, _ := BindValorisationRepository(lanes, NewSQLDocumentStore(lanes, t.TempDir()), tenant)
	actor := ValorisationActor{Email: "reviewer@example.com", Manage: true, Approve: true}
	in := ValorisationInput{EffectiveOn: "2026-04-01"}
	run, err := repo.Create(in, "org", actor, mustDate(in.EffectiveOn))
	if err != nil {
		t.Fatal(err)
	}
	approved, err := repo.Approve(run.ID, actor, DefaultValorisationSettings(), mustDate(in.EffectiveOn), func(ValorisationRun, ValorisationItem, time.Time) ([]byte, error) {
		return []byte("%PDF-1.4 staffel"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	item := approved.Items[0]
	loaded, _, err = leases.Get(l.ID)
	if err != nil || loaded.Clauses[0].State.LastRunItemID != item.ID || loaded.Clauses[0].State.ContractValue != "1100.00" {
		t.Fatalf("state %v %+v", err, loaded.Clauses[0].State)
	}
	stored, _, err := repo.Get(run.ID)
	if err != nil || stored.Items[0].Decision.CappedCents != 5972 || stored.Items[0].Contract.ExactAmountCents != "110000" {
		t.Fatal("lost capped contract", err)
	}
	// Approval persisted a capped HMZ and a rounded state value. Repeating the
	// same date must still use the original anchor from the approved snapshot.
	repeated, err := repo.Preview(in, mustDate(in.EffectiveOn))
	if err != nil || repeated.Items[0].Group != "unchanged" || repeated.Items[0].NewCents != 104028 || repeated.Items[0].CapStartCents != 100000 {
		t.Fatalf("repeated capped run: %+v %v", repeated, err)
	}
	updated := loaded.Clauses[0]
	updated.StaffelSteps = updated.StaffelSteps[:1]
	if _, err := leases.SetClause(updated); err != nil {
		t.Fatal(err)
	}
	stored, _, err = repo.Get(run.ID)
	if err != nil || len(stored.Items[0].Clause.StaffelSteps) != 2 {
		t.Fatal("approved schedule mutated", err)
	}
	var raw string
	if err := database.QueryRow(`SELECT staffel_steps FROM index_clauses WHERE id=$1`, updated.ID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var steps []indexation.StaffelStep
	if json.Unmarshal([]byte(raw), &steps) != nil || len(steps) != 1 {
		t.Fatal("row replacement failed", raw)
	}
}

func TestStaffelHistoricalAdjustmentsNeedReviewedAnchor(t *testing.T) {
	l := staffelFixture(t)
	l.Clauses[0].StaffelSteps, _ = ParseStaffel("2025-04-01:1100;2026-04-01:1200")
	l.Components = append(l.Components, RentComponent{Kind: ComponentHMZ, NetCents: 110000, ValidFrom: "2025-04-01"})
	got := previewTest(t, l, "2026-04-01")
	if got.Group != "exception" || !strings.Contains(strings.Join(got.Exceptions, ","), "missed_pre2026") {
		t.Fatalf("guessed historical cap anchor: %+v", got)
	}
	l.Clauses[0].State = &ValorisationState{CapAnchorPeriod: "2025-04", CapValue: "1100.00", LastEffectiveOn: "2025-04-01"}
	got = previewTest(t, l, "2026-04-01")
	if got.Group != "ready" || got.ContractCents != 120000 || got.CapStartCents != 110000 {
		t.Fatalf("reviewed anchor: %+v", got)
	}

	l = staffelFixture(t)
	l.Components[0].NetCents = 10001
	l.Components = append(l.Components, RentComponent{Kind: ComponentHMZ, NetCents: 10051, ValidFrom: "2025-04-01"})
	l.Clauses[0].StaffelSteps, _ = ParseStaffel("2025-04-01:0.5%")
	got = previewTest(t, l, "2026-04-01")
	if got.Group != "unchanged" {
		t.Fatalf("rounded legacy step: %+v", got)
	}
}

package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/indexation"
)

func valorisationFixture() Lease {
	return Lease{ID: "lease-e1", UnitID: "top-1", Status: LeaseStatusActive, ConcludedOn: "2024-09-01", StartsOn: "2024-09-01", LeaseKind: LeaseKindHauptmiete, UseKind: UseKindWohnung, MRGScope: MRGTeil, RentRegime: RentRegimeFrei, PriceRestrictedSet: true, TenantIsConsumer: true, ZinsterminDay: 5,
		Parties:    []LeaseParty{{ID: "party-1", Name: "Eva Huber", Email: "eva@example.com", Role: PartyHauptmieter, ValidFrom: "2024-09-01"}},
		Components: []RentComponent{{ID: "hmz-1", Kind: ComponentHMZ, NetCents: 100000, VATRateBP: 1000, ValidFrom: "2024-09-01"}},
		Clauses:    []IndexClause{{ID: "clause-1", ComponentKind: ComponentHMZ, ClauseType: ClauseVPIThreshold, Series: "vpi2020", BasePeriod: "2024-09", BaseValue: "123.6", ThresholdKind: "percent", ThresholdValue: "5", TwoWay: true, FullChangeOnTrigger: true, PctRounding: "none", ReviewStatus: ReviewOK, ValidFrom: "2024-09-01"}},
	}
}
func previewTest(t *testing.T, lease Lease, at string) ValorisationItem {
	t.Helper()
	snap, err := indexation.LoadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	run, err := PreviewValorisation(ValorisationInput{EffectiveOn: "2026-04-01", Leases: []Lease{lease}}, snap, mustDate(at))
	if err != nil {
		t.Fatal(err)
	}
	return run.Items[0]
}
func TestValorisationExamplesAndTiming(t *testing.T) {
	partial := valorisationFixture()
	e1 := previewTest(t, partial, "2026-04-01")
	if e1.NewCents != 104028 || e1.Group != "ready" || e1.WirksamOn != "2026-04-01" {
		t.Fatalf("E1: %+v", e1)
	}
	full := partial
	full.MRGScope = MRGVoll
	full.RentRegime = RentRegimeRichtwert
	full.PriceRestricted = true
	e2 := previewTest(t, full, "2026-04-01")
	if e2.NewCents != 101735 || e2.CollectableFrom != "2026-05-05" || e2.NoticeDeadline != "2026-04-21" {
		t.Fatalf("E2: %+v", e2)
	}
	for _, tt := range []struct{ date, want string }{{"2026-04-21", "2026-05-05"}, {"2026-04-22", "2026-06-05"}, {"2026-09-24", "2026-11-05"}} {
		timed, err := ValorisationLetterTiming(e2, mustDate(tt.date))
		if err != nil || timed.CollectableFrom != tt.want {
			t.Fatalf("%s: %s %v", tt.date, timed.CollectableFrom, err)
		}
	}
	if _, err := ValorisationLetterTiming(e2, mustDate("2026-03-31")); err == nil {
		t.Fatal("premature notice accepted")
	}
	full.PriceRestricted = false
	full.RentRegime = RentRegimeFrei
	if got := previewTest(t, full, "2026-04-01"); got.NewCents != 104028 {
		t.Fatalf("unrestricted full: %+v", got)
	}
	commercial := partial
	commercial.UseKind = UseKindGeschaeft
	commercial.MRGScope = MRGVoll
	got := previewTest(t, commercial, "2026-04-01")
	if got.MieWeG || got.NewCents != 105016 || got.WirksamOn != "2026-04-01" {
		t.Fatalf("commercial %+v", got)
	}
}
func TestValorisationExceptions(t *testing.T) {
	cases := []struct {
		code   string
		change func(*Lease)
	}{
		{"no_clause", func(l *Lease) { l.Clauses = nil }},
		{"clause_invalid", func(l *Lease) { l.Clauses[0].ReviewStatus = ReviewInvalid }},
		{"clause_unreviewed", func(l *Lease) { l.Clauses[0].ReviewStatus = ReviewUnreviewed }},
		{"one_way_clause_risk", func(l *Lease) { l.Clauses[0].TwoWay = false }},
		{"base_value_mismatch", func(l *Lease) { l.Clauses[0].BaseValue = "123.7" }},
		{"index_missing", func(l *Lease) { l.Clauses[0].Series = "vpi1" }},
		{"max_hmz_exceeded", func(l *Lease) { l.MRGScope = MRGVoll; l.PriceRestricted = true; n := int64(100500); l.MaxHMZCents = &n }},
		{"lease_ended", func(l *Lease) { l.EndsOn = "2026-04-01" }},
		{"lease_not_started", func(l *Lease) { l.StartsOn = "2026-04-02" }},
		{"mixed_use_check", func(l *Lease) { l.UseKind = UseKindSonstiges }},
		{"no_recipient", func(l *Lease) { l.Parties[0].Email = "" }},
		{"missed_pre2026", func(l *Lease) { l.Clauses[0].ThresholdValue = "0" }},
	}
	for _, tt := range cases {
		t.Run(tt.code, func(t *testing.T) {
			l := valorisationFixture()
			tt.change(&l)
			got := previewTest(t, l, "2026-04-01")
			if !containsValorisation(got.Exceptions, tt.code) || got.Group != "exception" {
				t.Fatalf("%+v", got)
			}
		})
	}
	early := previewTest(t, valorisationFixture(), "2026-03-25")
	if !containsValorisation(early.Exceptions, "letter_too_early") {
		t.Fatalf("early %+v", early)
	}
}
func containsValorisation(items []string, s string) bool {
	for _, item := range items {
		if item == s {
			return true
		}
	}
	return false
}
func TestValorisationIndexStatusesAndRounding(t *testing.T) {
	for _, code := range []string{"index_preliminary", "index_derived", "index_missing"} {
		t.Run(code, func(t *testing.T) {
			l := valorisationFixture()
			snapshot, _ := indexation.LoadSnapshot()
			values := snapshot.Data.Values()
			filtered := values[:0]
			for _, v := range values {
				if v.Series == indexation.VPI2020 && v.Month == "2025-12" {
					if code == "index_missing" {
						continue
					}
					if code == "index_preliminary" {
						v.Preliminary = true
					} else {
						v.ChainSource = "https://example.org/factor"
					}
				}
				filtered = append(filtered, v)
			}
			snapshot.Data, _ = indexation.NewDataset(filtered, nil)
			run, err := PreviewValorisation(ValorisationInput{EffectiveOn: "2026-04-01", Leases: []Lease{l}}, snapshot, mustDate("2026-04-01"))
			if err != nil {
				t.Fatal(err)
			}
			if !containsValorisation(run.Items[0].Exceptions, code) {
				t.Fatalf("%+v", run.Items[0])
			}
		})
	}
	l := valorisationFixture()
	l.Clauses[0].PctRounding = "one_decimal"
	got := previewTest(t, l, "2026-04-01")
	if got.Group != "unchanged" || got.NewCents != 100000 {
		t.Fatalf("exclusive rounded threshold %+v", got)
	}
	l.Clauses[0].ThresholdInclusive = true
	got = previewTest(t, l, "2026-04-01")
	if got.NewCents != 104028 {
		t.Fatalf("inclusive %+v", got)
	}
	if value, ok := PublishedIndexValue("vpi2020", "2024-09"); !ok || value != "123.6" {
		t.Fatalf("published %s %v", value, ok)
	}
	if _, ok := PublishedIndexValue("vpi2020", "2026-08"); ok {
		t.Fatal("preliminary is not final")
	}
}
func TestValorisationLifecycle(t *testing.T) {
	database, lanes := testLanes(t)
	tenant := testTenantRef("demo")
	if _, err := database.Exec(`INSERT INTO units(tenant_id,tenant_slug,id,data) VALUES($1,$2,'top-1','{}')`, tenant.ID, tenant.Slug); err != nil {
		t.Fatal(err)
	}
	leases, _ := BindLeaseRepository(NewSQLLeaseStore(lanes), tenant)
	if _, err := leases.Create(valorisationFixture()); err != nil {
		t.Fatal(err)
	}
	documents := NewSQLDocumentStore(lanes, t.TempDir())
	repo, _ := BindValorisationRepository(lanes, documents, tenant)
	author := ValorisationActor{Email: "creator@example.com", Manage: true, Approve: true}
	reviewer := ValorisationActor{Email: "reviewer@example.com", Manage: true, Approve: true}
	settings := DefaultValorisationSettings()
	settings.FourEyes = true
	run, err := repo.Create(ValorisationInput{EffectiveOn: "2026-04-01", Settings: settings}, "org", author, mustDate("2026-04-01"))
	if err != nil {
		t.Fatal(err)
	}
	if len(run.InputsSHA256) != 64 || len(run.IndexSnapshot) == 0 {
		t.Fatal("missing snapshot")
	}
	render := func(run ValorisationRun, item ValorisationItem, at time.Time) ([]byte, error) {
		if run.Status != "approved" {
			t.Fatal("draft render at approval")
		}
		return []byte("%PDF-1.4 test letter " + at.Format(time.DateOnly)), nil
	}
	if _, err = repo.Approve(run.ID, author, settings, mustDate("2026-04-01"), render); err == nil {
		t.Fatal("four eyes")
	}
	if _, err = repo.Approve(run.ID, ValorisationActor{Email: "manager@example.com", Manage: true}, settings, mustDate("2026-04-01"), render); !errors.Is(err, ErrValorisationDenied) {
		t.Fatal(err)
	}
	approved, err := repo.Approve(run.ID, reviewer, settings, mustDate("2026-04-01"), render)
	if err != nil {
		t.Fatal(err)
	}
	if approved.Items[0].LetterDocumentID == "" {
		t.Fatal("not archived")
	}
	lease, _, err := leases.Get("lease-e1")
	if err != nil {
		t.Fatal(err)
	}
	if len(lease.Components) != 2 || lease.Components[1].NetCents != 104028 || !strings.HasPrefix(lease.Components[1].Origin, "valorisation_item:") {
		t.Fatalf("components %+v", lease.Components)
	}
	if lease.Clauses[0].State.LastRunItemID != run.Items[0].ID {
		t.Fatal("state not advanced")
	}
	if _, err = repo.Approve(run.ID, reviewer, settings, mustDate("2026-04-01"), render); !errors.Is(err, ErrValorisationConflict) {
		t.Fatal("duplicate approve", err)
	}
	for _, q := range []string{`UPDATE valorisation_runs SET data='{}' WHERE id=$1`, `UPDATE valorisation_items SET data='{}' WHERE run_id=$1`, `DELETE FROM valorisation_items WHERE run_id=$1`, `DELETE FROM valorisation_runs WHERE id=$1`, `INSERT INTO valorisation_items(tenant_id,tenant_slug,id,run_id,lease_id,clause_id,data) SELECT tenant_id,tenant_slug,'extra',id,'extra','extra','{}' FROM valorisation_runs WHERE id=$1`} {
		if _, err = database.Exec(q, run.ID); err == nil {
			t.Fatal("immutable query accepted", q)
		}
	}
	foreign, _ := BindValorisationRepository(lanes, documents, testTenantRef("other"))
	if _, found, err := foreign.Get(run.ID); err != nil || found {
		t.Fatal("foreign read", found, err)
	}
	doc, err := repo.PrepareLetter(run.ID, run.Items[0].ID, reviewer, mustDate("2026-04-22"), render)
	if err != nil {
		t.Fatal(err)
	}
	deliveries, _ := BindValorisationDeliveryRepository(NewSQLValorisationDeliveryStore(lanes), tenant)
	delivery := ValorisationDelivery{RunID: run.ID, Revision: run.Revision, PartyID: "eva@example.com", UnitID: "top-1", DocumentID: doc.ID, SHA256: doc.ValorisationArchive.SHA256, Recipient: "eva@example.com", Actor: reviewer.Email}
	attempts := 0
	send := func(context.Context) error {
		attempts++
		if attempts == 1 {
			return errors.New("mail unavailable")
		}
		return nil
	}
	first, _, err := deliveries.Attempt(t.Context(), delivery, send)
	if err != nil || first.Status != "failed" {
		t.Fatal(first, err)
	}
	second, _, err := deliveries.Attempt(t.Context(), delivery, send)
	if err != nil || second.Status != "sent" {
		t.Fatal(second, err)
	}
	_, skip, err := deliveries.Attempt(t.Context(), delivery, send)
	if err != nil || !skip || attempts != 2 {
		t.Fatal("duplicate send", skip, attempts, err)
	}
	if err = repo.FinishSend(run.ID, reviewer, time.Now()); err != nil {
		t.Fatal(err)
	}
	sent, _, _ := repo.Get(run.ID)
	if sent.Status != "sent" {
		t.Fatal(sent.Status)
	}
	if err = repo.Cancel(run.ID, "", reviewer, time.Now()); err == nil {
		t.Fatal("cancel without reason")
	}
	if err = repo.Cancel(run.ID, "Korrektur gesondert veranlasst", reviewer, time.Now()); err != nil {
		t.Fatal(err)
	}
	var events int
	if err = database.QueryRow(`SELECT COUNT(*) FROM valorisation_events WHERE run_id=$1`, run.ID).Scan(&events); err != nil || events < 5 {
		t.Fatal("events", events, err)
	}
}

func TestValorisationPeriodicReference(t *testing.T) {
	l := valorisationFixture()
	l.ConcludedOn = "2025-09-01"
	l.Clauses[0].BasePeriod = "2025-09"
	l.Clauses[0].BaseValue = "128.5"
	l.Clauses[0].ClauseType = ClauseVPIPeriodic
	l.Clauses[0].PeriodicMonth = 4
	offset := -4
	l.Clauses[0].ReferenceMonthOffset = &offset
	got := previewTest(t, l, "2026-04-01")
	if got.Group != "ready" || got.Contract.TriggerMonth != "2025-12" {
		t.Fatalf("periodic %+v", got)
	}
}

func TestValorisationDeflationAndClassification(t *testing.T) {
	snapshot, _ := indexation.LoadSnapshot()
	// Synthetic isolated evidence tests the adapter, not the official data set.
	snapshot.Data, _ = indexation.NewDataset([]indexation.IndexValue{{Series: indexation.VPI2020, Month: "2025-10", Value: 100 * indexation.Unit}, {Series: indexation.VPI2020, Month: "2025-11", Value: 94 * indexation.Unit}}, nil)
	l := valorisationFixture()
	l.ConcludedOn = "2025-10-01"
	l.Clauses[0].BasePeriod, l.Clauses[0].BaseValue = "2025-10", "100"
	run, err := PreviewValorisation(ValorisationInput{EffectiveOn: "2026-02-01", Leases: []Lease{l}}, snapshot, mustDate("2026-02-01"))
	if err != nil {
		t.Fatal(err)
	}
	item := run.Items[0]
	if item.Outcome != "decrease" || item.NewCents != 94000 || item.WirksamOn != "2026-02-01" || item.RequiresMRGNotice {
		t.Fatalf("deflation %+v", item)
	}
	l.TenantIsConsumer = false
	l.Clauses[0].TwoWay = false
	run, err = PreviewValorisation(ValorisationInput{EffectiveOn: "2026-02-01", Leases: []Lease{l}}, snapshot, mustDate("2026-02-01"))
	if err != nil || run.Items[0].Group != "unchanged" || run.Items[0].NewCents != 100000 {
		t.Fatal("one-way business decrease", err)
	}
	l = valorisationFixture()
	l.LeaseKind = LeaseKindUntermiete
	l.MRGScope = MRGVoll
	item = previewTest(t, l, "2026-04-01")
	if !item.MieWeG || item.RequiresMRGNotice || item.NewCents != 104028 {
		t.Fatalf("sublease %+v", item)
	}
	l = valorisationFixture()
	l.UseKind = UseKindGarage
	l.MRGScope = MRGAusnahme
	item = previewTest(t, l, "2026-04-01")
	if item.MieWeG || item.NewCents != 105016 {
		t.Fatalf("garage %+v", item)
	}
	l = valorisationFixture()
	l.TenantIsConsumer = false
	l.Clauses[0].TwoWay = false
	if item = previewTest(t, l, "2026-04-01"); item.Group != "ready" {
		t.Fatal("B2B clause", item.Exceptions)
	}
	if got := ValorisationVAT(100005, 1000); got != 10001 {
		t.Fatal("VAT half cent", got)
	}
}

func TestValorisationConcurrentRevisionsPrescribeOnce(t *testing.T) {
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
	actor := ValorisationActor{Email: "review@example.com", Manage: true, Approve: true}
	now := mustDate("2026-04-01")
	var ids []string
	for range 2 {
		run, err := repo.Create(ValorisationInput{EffectiveOn: "2026-04-01"}, "org", actor, now)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, run.ID)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, id := range ids {
		go func() {
			<-start
			_, err := repo.Approve(id, actor, DefaultValorisationSettings(), now, func(ValorisationRun, ValorisationItem, time.Time) ([]byte, error) {
				return []byte("%PDF-1.4 test"), nil
			})
			results <- err
		}()
	}
	close(start)
	succeeded := 0
	for range 2 {
		if err := <-results; err == nil {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Fatal("successful competing approvals", succeeded)
	}
	lease, _, err := leases.Get("lease-e1")
	if err != nil || len(lease.Components) != 2 {
		t.Fatal("duplicate prescriptions", len(lease.Components), err)
	}
}

package store

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/indexation"
)

func aprilTimingFixture(t *testing.T) (Lease, indexation.Snapshot) {
	t.Helper()
	l := valorisationFixture()
	l.UseKind, l.MRGScope = UseKindGeschaeft, MRGVoll
	l.ConcludedOn, l.StartsOn = "2026-01-01", "2026-01-01"
	l.Clauses[0].BasePeriod, l.Clauses[0].BaseValue = "2026-01", "100"
	values := []indexation.IndexValue{}
	for _, v := range []struct {
		month string
		value int64
	}{{"2026-01", 100}, {"2026-02", 101}, {"2026-03", 102}, {"2026-04", 106}} {
		values = append(values, indexation.IndexValue{Series: indexation.VPI2020, Month: indexation.Month(v.month), Value: indexation.Decimal(v.value) * indexation.Unit})
	}
	data, err := indexation.NewDataset(values, nil)
	if err != nil {
		t.Fatal(err)
	}
	return l, indexation.Snapshot{Data: data}
}

func TestValorisationModesAndLeaseOverrides(t *testing.T) {
	for _, tc := range []struct{ org, override, date, effective, due string }{
		{"wko", "", "2026-08-01", "2026-08-01", "2026-09-05"},
		{"oevi", "", "2026-06-17", "2026-06-17", "2026-07-05"},
		{"wko", "oevi", "2026-07-01", "2026-06-17", "2026-07-05"},
		{"oevi", "wko", "2026-08-01", "2026-08-01", "2026-09-05"},
		{"wko", "contract", "2026-08-01", "2026-08-01", "2026-09-05"},
		{"oevi", "contract", "2026-07-01", "2026-07-01", "2026-08-05"},
		{"contract", "", "2026-09-01", "2026-09-01", "2026-10-05"},
	} {
		t.Run(tc.org+"/"+tc.override+"/"+tc.date, func(t *testing.T) {
			l, snap := aprilTimingFixture(t)
			l.WirksamwerdenMode = tc.override
			run, err := PreviewValorisation(ValorisationInput{EffectiveOn: tc.date, Settings: ValorisationSettings{WirksamwerdenMode: tc.org}, Leases: []Lease{l}}, snap, mustDate(tc.date))
			if err != nil {
				t.Fatal(err)
			}
			got := run.Items[0]
			if got.Group != "ready" || got.WirksamOn != tc.effective || got.CollectableFrom != tc.due || got.NewCents != 106000 {
				t.Fatalf("%+v", got)
			}
			if got.TimingInput.TriggerMonth != "2026-04" || got.TimingInput.FinalPublishedOn != mustDate("2026-06-17") {
				t.Fatal("missing date evidence")
			}
			if _, err := ValorisationLetterTiming(got, mustDate(tc.effective).AddDate(0, 0, -1)); err == nil || !strings.Contains(err.Error(), "gesperrt") {
				t.Fatalf("early letter: %v", err)
			}
			timed, err := ValorisationLetterTiming(got, mustDate(tc.effective))
			if err != nil || timed.TimingInput.NoticeIssuedOn.IsZero() || timed.TimingInput.NoticeReceivedOn.IsZero() {
				t.Fatal("notice dates not retained", err)
			}
			vals := snap.Data.Values()
			vals[len(vals)-1].Preliminary = true
			snap.Data, _ = indexation.NewDataset(vals, nil)
			run, err = PreviewValorisation(ValorisationInput{EffectiveOn: tc.date, Settings: ValorisationSettings{WirksamwerdenMode: tc.org}, Leases: []Lease{l}}, snap, mustDate(tc.date))
			if err != nil || !containsValorisation(run.Items[0].Exceptions, "index_preliminary") {
				t.Fatal("preliminary accepted", err)
			}
		})
	}
}

func TestLeaseTimingOverridePersistenceAndScope(t *testing.T) {
	database, lanes := testLanes(t)
	tenant := testTenantRef("demo")
	if _, err := database.Exec(`INSERT INTO units(tenant_id,tenant_slug,id,data) VALUES($1,$2,'top-1','{}')`, tenant.ID, tenant.Slug); err != nil {
		t.Fatal(err)
	}
	repo, _ := BindLeaseRepository(NewSQLLeaseStore(lanes), tenant)
	l := valorisationFixture()
	l.WirksamwerdenMode = "oevi"
	l.UpdatedAt = time.Now()
	saved, err := repo.Create(l)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"oevi", "wko", "contract", ""} {
		saved.WirksamwerdenMode = mode
		if _, err = repo.Update(saved); err != nil {
			t.Fatal(err)
		}
		got, found, err := repo.Get(saved.ID)
		if err != nil || !found || got.WirksamwerdenMode != mode {
			t.Fatalf("%+v %v", got, err)
		}
	}
	saved.WirksamwerdenMode = "unsafe"
	if _, err = repo.Update(saved); err == nil {
		t.Fatal("invalid override accepted")
	}
	for _, change := range []func(*Lease){func(l *Lease) { l.MRGScope = MRGTeil }, func(l *Lease) { l.MRGScope = MRGAusnahme }, func(l *Lease) { l.LeaseKind = LeaseKindUntermiete }, func(l *Lease) { l.UseKind = UseKindWohnung }} {
		l, snap := aprilTimingFixture(t)
		change(&l)
		l.WirksamwerdenMode = "wko"
		a, err := PreviewValorisation(ValorisationInput{EffectiveOn: "2026-08-01", Settings: ValorisationSettings{WirksamwerdenMode: "wko"}, Leases: []Lease{l}}, snap, mustDate("2026-08-01"))
		if err != nil {
			t.Fatal(err)
		}
		l.WirksamwerdenMode = "oevi"
		b, err := PreviewValorisation(ValorisationInput{EffectiveOn: "2026-08-01", Settings: ValorisationSettings{WirksamwerdenMode: "oevi"}, Leases: []Lease{l}}, snap, mustDate("2026-08-01"))
		if err != nil {
			t.Fatal(err)
		}
		if a.Items[0].WirksamOn != b.Items[0].WirksamOn || a.Items[0].NewCents != b.Items[0].NewCents || l.UsesValorisationTimingMode() {
			t.Fatal("mode leaks into excluded scope")
		}
	}
}

func TestEarlyApprovalAndReceiptEvidence(t *testing.T) {
	database, lanes := testLanes(t)
	tenant := testTenantRef("demo")
	if _, err := database.Exec(`INSERT INTO units(tenant_id,tenant_slug,id,data) VALUES($1,$2,'top-1','{}')`, tenant.ID, tenant.Slug); err != nil {
		t.Fatal(err)
	}
	leases, _ := BindLeaseRepository(NewSQLLeaseStore(lanes), tenant)
	l := valorisationFixture()
	l.MRGScope = MRGVoll
	if _, err := leases.Create(l); err != nil {
		t.Fatal(err)
	}
	repo, _ := BindValorisationRepository(lanes, NewSQLDocumentStore(lanes, t.TempDir()), tenant)
	actor := ValorisationActor{Email: "manager@example.com", Manage: true, Approve: true}
	settings := DefaultValorisationSettings()
	run, err := repo.Create(ValorisationInput{EffectiveOn: "2026-04-01", Settings: settings}, "org", actor, mustDate("2026-03-30"))
	if err != nil {
		t.Fatal(err)
	}
	rendered := 0
	render := func(run ValorisationRun, item ValorisationItem, at time.Time) ([]byte, error) {
		rendered++
		return []byte("%PDF-1.4 fixture"), nil
	}
	if _, err = repo.Approve(run.ID, actor, settings, mustDate("2026-03-31"), render); err == nil || !strings.Contains(err.Error(), "Ein Schreiben darf erst ab Wirksamkeit ausgestellt werden.") {
		t.Fatalf("approval not blocked: %v", err)
	}
	if rendered != 0 {
		t.Fatal("premature letter rendered")
	}
	approved, err := repo.Approve(run.ID, actor, settings, mustDate("2026-04-01"), render)
	if err != nil {
		t.Fatal(err)
	}
	stored, _, err := repo.Get(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Items[0].TimingInput.NoticeIssuedOn != mustDate("2026-04-01") || stored.Items[0].TimingInput.NoticeReceivedOn != mustDate("2026-04-01") {
		t.Fatal("notice assumptions not persisted")
	}
	if _, err = repo.PrepareLetter(run.ID, approved.Items[0].ID, actor, mustDate("2026-03-31"), render); err == nil {
		t.Fatal("premature reissue accepted")
	}
	item := approved.Items[0]
	deliveries, _ := BindValorisationDeliveryRepository(NewSQLValorisationDeliveryStore(lanes), tenant)
	delivery, _, err := deliveries.Attempt(context.Background(), ValorisationDelivery{RunID: run.ID, Revision: run.Revision, PartyID: "eva@example.com", UnitID: item.UnitID, DocumentID: item.LetterDocumentID, SHA256: item.LetterSHA256, Recipient: "eva@example.com", Actor: actor.Email}, func(context.Context) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	// Simulate the mail transport timestamp independently of this test's clock.
	if _, err = database.Exec(`UPDATE valorisation_deliveries SET sent_at='2026-04-20T08:00:00Z' WHERE tenant_id=$1 AND id=$2`, tenant.ID, delivery.ID); err != nil {
		t.Fatal(err)
	}
	if err = deliveries.RecordReceipt(run.ID, delivery.ID, "2026-04-19", actor, mustDate("2026-04-30")); err == nil {
		t.Fatal("receipt before send accepted")
	}
	if err = deliveries.RecordReceipt(run.ID, delivery.ID, "2026-05-01", actor, mustDate("2026-04-30")); err == nil {
		t.Fatal("future receipt accepted")
	}
	if err = deliveries.RecordReceipt(run.ID, delivery.ID, "2026-04-22", ValorisationActor{Email: actor.Email}, mustDate("2026-04-30")); err != ErrValorisationDenied {
		t.Fatal("receipt authorization", err)
	}
	if err = deliveries.RecordReceipt(run.ID, delivery.ID, "2026-04-22", actor, mustDate("2026-04-30")); err != nil {
		t.Fatal(err)
	}
	rows, err := deliveries.List(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	// A stored UTC timestamp must use the Austrian civil day for the notice.
	if _, err = database.Exec(`UPDATE valorisation_deliveries SET sent_at='2026-04-21T23:00:00Z' WHERE tenant_id=$1 AND id=$2`, tenant.ID, delivery.ID); err != nil {
		t.Fatal(err)
	}
	if err = deliveries.RecordReceipt(run.ID, delivery.ID, "2026-04-21", actor, mustDate("2026-04-30")); err == nil {
		t.Fatal("UTC date admitted receipt before Austrian send day")
	}
	if len(rows) != 1 || rows[0].ReceivedOn != "2026-04-22" || rows[0].ReceiptDueOn != "2026-06-05" {
		t.Fatalf("receipt dates: %+v", rows)
	}
	after, _, err := repo.Get(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stored.Items, after.Items) {
		t.Fatal("receipt rewrote frozen calculation")
	}
	foreign, _ := BindValorisationDeliveryRepository(NewSQLValorisationDeliveryStore(lanes), testTenantRef("other"))
	if err = foreign.RecordReceipt(run.ID, delivery.ID, "2026-04-22", actor, mustDate("2026-04-30")); err == nil {
		t.Fatal("foreign receipt accepted")
	}
}

func TestPeriodicTimingKeepsLaterContractDate(t *testing.T) {
	l, snap := aprilTimingFixture(t)
	l.Clauses[0].ClauseType = ClauseVPIPeriodic
	l.Clauses[0].PeriodicMonth = 9
	offset := -5
	l.Clauses[0].ReferenceMonthOffset = &offset
	l.WirksamwerdenMode = "oevi"
	run, err := PreviewValorisation(ValorisationInput{EffectiveOn: "2026-09-01", Leases: []Lease{l}}, snap, mustDate("2026-09-01"))
	if err != nil {
		t.Fatal(err)
	}
	if run.Items[0].WirksamOn != "2026-09-01" || run.Items[0].CollectableFrom != "2026-10-05" {
		t.Fatalf("%+v", run.Items[0])
	}
}

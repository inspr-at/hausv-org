package demo

import (
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestJanusbergwegLeaseFixture(t *testing.T) {
	database, config := dbtest.OpenWithConfig(t)
	if _, err := Load(t.Context(), database, "../../scripts/demo/seed", SeedOptions{DocumentDir: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	scoped, err := db.NewScoped(config, database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = scoped.Close() })
	identities, err := store.EnsureTenantIdentities(t.Context(), database, []store.TenantIdentity{{Slug: "janusbergweg-123"}})
	if err != nil {
		t.Fatal(err)
	}
	repo, ok := store.BindLeaseRepository(store.NewSQLLeaseStore(store.NewTenantDB(scoped)), identities["janusbergweg-123"].Ref())
	if !ok {
		t.Fatal("bind")
	}
	leases, err := repo.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(leases) != 12 {
		t.Fatalf("leases = %d, want 12", len(leases))
	}
	byID := map[string]store.Lease{}
	for _, lease := range leases {
		byID[lease.ID] = lease
	}
	partial := byID["lease-top-1"]
	full := byID["lease-top-2"]
	hmz := func(lease store.Lease) int64 {
		for _, component := range lease.Components {
			if component.Kind == store.ComponentHMZ {
				return component.NetCents
			}
		}
		return -1
	}
	if partial.MRGScope != store.MRGTeil || hmz(partial) != 100000 || partial.Clauses[0].BaseValue != "123.6" || partial.Clauses[0].State == nil || partial.Clauses[0].State.CapAnchorPeriod != "2024-09" {
		t.Fatalf("E1 = %+v state=%+v", partial, partial.Clauses[0].State)
	}
	if !full.PriceRestricted || full.RentRegime != store.RentRegimeRichtwert || full.Clauses[0].BaseValue != "123.6" {
		t.Fatalf("E2 = %+v", full)
	}
	if byID["lease-top-4"].Clauses[0].ClauseType != store.ClauseNone || byID["lease-top-5"].Clauses[0].ReviewStatus != store.ReviewUnreviewed || byID["lease-top-6"].Clauses[0].TwoWay || byID["lease-top-6"].TenantIsConsumer != true {
		t.Fatal("exception fixtures drifted")
	}
	if byID["lease-top-7"].UseKind != store.UseKindGeschaeft || store.ClassifyLease(byID["lease-top-7"]).MieWeG {
		t.Fatal("commercial lease should sit outside the MieWeG")
	}
}

func TestZinshausDemoLeasesAndValorisation(t *testing.T) {
	database, config := dbtest.OpenWithConfig(t)
	options := SeedOptions{DocumentDir: t.TempDir(), Reset: true}
	if _, err := Load(t.Context(), database, "../../scripts/demo/seed", options); err != nil {
		t.Fatal(err)
	}
	scoped, err := db.NewScoped(config, database)
	if err != nil {
		t.Fatal(err)
	}
	defer scoped.Close()
	identities, err := store.EnsureTenantIdentities(t.Context(), database, []store.TenantIdentity{{Slug: "musterstrasse-12"}})
	if err != nil {
		t.Fatal(err)
	}
	lanes := store.NewTenantDB(scoped)
	ref := identities["musterstrasse-12"].Ref()
	repo, ok := store.BindLeaseRepository(store.NewSQLLeaseStore(lanes), ref)
	if !ok {
		t.Fatal("bind")
	}
	leases, err := repo.List()
	if err != nil || len(leases) != 10 {
		t.Fatalf("leases = %d, %v", len(leases), err)
	}
	byID := map[string]store.Lease{}
	for _, lease := range leases {
		byID[lease.ID] = lease
	}
	shop := byID["m12-lease-geschaeft"]
	if shop.UseKind != store.UseKindGeschaeft || shop.Clauses[0].ThresholdValue != "3" || store.ClassifyLease(shop).MieWeG {
		t.Fatalf("shop = %+v", shop)
	}
	restricted := store.ClassifyLease(byID["m12-lease-top-1"])
	free := store.ClassifyLease(byID["m12-lease-top-7"])
	if !restricted.MieWeG || !restricted.SpecialCap || !free.MieWeG || free.SpecialCap || byID["m12-lease-top-8"].Clauses[0].ClauseType != store.ClauseStaffel {
		t.Fatal("cap mix drifted")
	}
	if byID["m12-lease-top-3"].RentRegime != store.RentRegimeAngemessen || byID["m12-lease-top-10"].RentRegime != store.RentRegimeKategorie {
		t.Fatal("rent regimes drifted")
	}
	runsRepo, _ := store.BindValorisationRepository(lanes, store.NewSQLDocumentStore(lanes, options.DocumentDir), ref)
	runs, err := runsRepo.List()
	if err != nil || len(runs) != 1 || runs[0].Status != "draft" || runs[0].EffectiveOn != "2026-04-01" || len(runs[0].Items) != 10 {
		t.Fatalf("draft %+v %v", runs, err)
	}
	groups := map[string]int{}
	for _, item := range runs[0].Items {
		groups[item.Group]++
	}
	if groups["ready"] != 7 || groups["unchanged"] != 1 || groups["exception"] != 2 {
		t.Fatalf("preview groups %+v", groups)
	}
	if _, err := Load(t.Context(), database, "../../scripts/demo/seed", options); err != nil {
		t.Fatal("reseed", err)
	}
	runs, err = runsRepo.List()
	if err != nil || len(runs) != 1 || runs[0].Status != "draft" {
		t.Fatal("reset draft", len(runs), err)
	}
}

func TestDemoValorisationDraft(t *testing.T) {
	database, config := dbtest.OpenWithConfig(t)
	options := SeedOptions{DocumentDir: t.TempDir(), Reset: true}
	if _, err := Load(t.Context(), database, "../../scripts/demo/seed", options); err != nil {
		t.Fatal(err)
	}
	scoped, err := db.NewScoped(config, database)
	if err != nil {
		t.Fatal(err)
	}
	defer scoped.Close()
	identities, err := store.EnsureTenantIdentities(t.Context(), database, []store.TenantIdentity{{Slug: "janusbergweg-123"}})
	if err != nil {
		t.Fatal(err)
	}
	lanes := store.NewTenantDB(scoped)
	repo, _ := store.BindValorisationRepository(lanes, store.NewSQLDocumentStore(lanes, options.DocumentDir), identities["janusbergweg-123"].Ref())
	runs, err := repo.List()
	if err != nil || len(runs) != 1 {
		t.Fatal(len(runs), err)
	}
	run := runs[0]
	if run.Status != "draft" || len(run.Items) != 12 {
		t.Fatal("demo run", run.Status, len(run.Items))
	}
	codes := map[string]bool{}
	groups := map[string]int{}
	for _, item := range run.Items {
		groups[item.Group]++
		t.Logf("%s: %s (%s), %d", item.Label(), item.Group, item.Reason, item.NewCents)
		if item.LeaseID == "lease-top-7" && (item.Group != "ready" || item.MieWeG) {
			t.Fatal("commercial lease must be ready outside MieWeG", item)
		}
		if item.LeaseID == "lease-top-1" && item.Label() != "Top 1 · Eva Huber" {
			t.Fatal("frozen unit/tenant label", item.Label())
		}
		for _, code := range item.Exceptions {
			codes[code] = true
		}
		if item.LeaseID == "lease-top-1" && item.NewCents != 104028 {
			t.Fatal("E1", item)
		}
		if item.LeaseID == "lease-top-2" && item.NewCents != 101735 {
			t.Fatal("E2", item)
		}
		// 105000 * (1 + (3 + ((128.2/123.8-1)*100-3)/2)/100*11/12), half-down.
		if item.LeaseID == "lease-top-9" && (item.Group != "ready" || !item.MieWeG || item.ContractCents != 115000 || item.NewCents != 108154 || item.Decision.CappedCents != 6846) {
			t.Fatal("Staffel with MieWeG cap", item)
		}
	}
	if groups["ready"] != 7 || groups["unchanged"] != 2 || groups["exception"] != 3 {
		t.Fatalf("demo mix: %+v", groups)
	}
	for _, code := range []string{"no_clause", "clause_unreviewed", "one_way_clause_risk"} {
		if !codes[code] {
			t.Fatalf("missing intended exception %s", code)
		}
	}
	if len(codes) != 3 {
		t.Fatal("unexpected exception", codes)
	}
	if _, err := Load(t.Context(), database, "../../scripts/demo/seed", options); err != nil {
		t.Fatal("reseed", err)
	}
	runs, err = repo.List()
	if err != nil || len(runs) != 1 {
		t.Fatal("reseed drafts", len(runs), err)
	}
	actor := store.ValorisationActor{Email: "demo-review@example.com", Manage: true, Approve: true}
	now := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	run, err = repo.Create(store.ValorisationInput{EffectiveOn: "2026-04-01"}, "musterstadt", actor, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range run.Items {
		if item.Group == "exception" {
			if err := repo.ItemAction(run.ID, item.ID, "exclude", "Gesonderte Prüfung", nil, actor, now); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := repo.Approve(run.ID, actor, store.DefaultValorisationSettings(), now, func(store.ValorisationRun, store.ValorisationItem, time.Time) ([]byte, error) {
		return []byte("%PDF-1.4 demo"), nil
	}); err != nil {
		t.Fatal(err)
	}
	options.DiscardAnnualStatements = true
	if _, err := Load(t.Context(), database, "../../scripts/demo/seed", options); err != nil {
		t.Fatal("clean reset", err)
	}
	runs, err = repo.List()
	if err != nil || len(runs) != 1 || runs[0].Status != "draft" || runs[0].ID == run.ID {
		t.Fatal("fresh draft after clean reset", err)
	}

}

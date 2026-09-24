package demo

import (
	"testing"

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

package energy_test

import (
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/store"
)

// TestEnergyWritesReuseTheBootIdentity is MUST-FIX 2, driven through the real
// boot entry point.
//
// store.EnsureTenantIdentities is what newApp calls; it mints the identity for
// every configured house under textutil.Slug. This package then used its own,
// stricter normalizer to ask for an identity — so for any legal slug the two
// normalizers disagree on, the first energy write minted a SECOND tenant row and
// stamped the wrong id onto the row it wrote. Nothing could see it: the
// completeness check counts NULL tenant_ids and a wrong one is not NULL.
//
// Where it is blind: it exercises SaveProfile, the write every onboarding path
// goes through, not all eight energy writers. Every one of them resolves its
// identity through the same tenantid.Ensure call, and
// TestEveryEnergyWriteRecordsATenantIdentity covers the rest for population.
func TestEnergyWritesReuseTheBootIdentity(t *testing.T) {
	for _, slug := range []string{"haus-a", "haus.a", "haus a", "haus--a", "haus-", "haus-grün", "demo"} {
		t.Run(slug, func(t *testing.T) {
			database, lanes := openEnergyLanes(t)
			identities, err := store.EnsureTenantIdentities(t.Context(), database,
				[]store.TenantIdentity{{Slug: slug, Name: "Haus"}})
			if err != nil {
				t.Fatalf("boot: %v", err)
			}
			booted := identities[slug].ID
			if booted == "" {
				t.Fatalf("boot minted no identity for %q", slug)
			}

			now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
			if err := energy.NewSQLStore(lanes).SaveProfile(energy.DefaultProfile(slug, now)); err != nil {
				t.Fatalf("save profile: %v", err)
			}

			var tenants int
			if err := database.QueryRow(`SELECT count(*) FROM tenant`).Scan(&tenants); err != nil {
				t.Fatalf("count tenants: %v", err)
			}
			if tenants != 1 {
				t.Errorf("one house, %d identities: an energy write minted its own", tenants)
			}
			var stored string
			if err := database.QueryRow(`SELECT tenant_id FROM home_profiles`).Scan(&stored); err != nil {
				t.Fatalf("read the profile's identity: %v", err)
			}
			if stored != booted {
				t.Errorf("the profile carries %q, the booted house owns %q", stored, booted)
			}
			var addressable int
			if err := database.QueryRow(`SELECT count(*) FROM home_profiles WHERE tenant_slug=$1`, slug).
				Scan(&addressable); err != nil {
				t.Fatalf("read the profile's slug: %v", err)
			}
			if addressable != 1 {
				t.Errorf("the profile is not addressable by its own slug %q — "+
					"a rollback to the previous release would not find it", slug)
			}
			// And the boot check agrees on a database the product wrote.
			if err := store.BackfillTenantIDs(t.Context(), database); err != nil {
				t.Fatalf("boot completeness check after the energy write: %v", err)
			}
		})
	}
}

// TestTwoHousesWhoseSlugsDifferOnlyOutsideTheAlphabetStaySeparate is the
// collision half of MUST-FIX 2. "haus-a" and "haus.a" are two different houses
// to the configuration layer and to the identity resolver. Under this package's
// old normalizer both collapsed to "haus-a", so home_profiles ended up with ONE
// row and the second house had no energy row of its own.
//
// Where it is blind: it proves the two rows exist and carry different
// identities. It does not prove every energy READ separates them — those still
// filter on tenant_slug, which is now the same string the identity was minted
// from, so they agree by construction rather than by test.
func TestTwoHousesWhoseSlugsDifferOnlyOutsideTheAlphabetStaySeparate(t *testing.T) {
	database, lanes := openEnergyLanes(t)
	identities, err := store.EnsureTenantIdentities(t.Context(), database, []store.TenantIdentity{
		{Slug: "haus-a", Name: "Haus A"},
		{Slug: "haus.a", Name: "Haus Punkt A"},
	})
	if err != nil {
		t.Fatalf("boot: %v", err)
	}
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	storage := energy.NewSQLStore(lanes)
	for _, slug := range []string{"haus-a", "haus.a"} {
		if err := storage.SaveProfile(energy.DefaultProfile(slug, now)); err != nil {
			t.Fatalf("save profile for %s: %v", slug, err)
		}
	}
	for _, slug := range []string{"haus-a", "haus.a"} {
		var id string
		if err := database.QueryRow(`SELECT tenant_id FROM home_profiles WHERE tenant_slug=$1`, slug).Scan(&id); err != nil {
			t.Fatalf("house %s has no energy profile of its own: %v", slug, err)
		}
		if want := identities[slug].ID; id != want {
			t.Errorf("house %s's profile carries %q, want %q", slug, id, want)
		}
	}
}

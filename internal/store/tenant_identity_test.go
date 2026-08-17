package store

import (
	"testing"

	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/ulid"
)

func TestTenantIdentityMintsOncePerSlug(t *testing.T) {
	database := dbtest.Open(t)
	configured := []TenantIdentity{
		{Slug: "demo", Name: "Demohaus"},
		{Slug: "haus-a", Name: "Haus A"},
	}

	first, err := EnsureTenantIdentities(t.Context(), database, configured)
	if err != nil {
		t.Fatalf("first boot: %v", err)
	}
	if len(first) != 2 {
		t.Fatalf("expected 2 identities, got %d", len(first))
	}
	for slug, identity := range first {
		if !ulid.Valid(identity.ID) {
			t.Errorf("%s: minted id %q is not a valid ULID", slug, identity.ID)
		}
	}
	if first["demo"].ID == first["haus-a"].ID {
		t.Fatal("two tenants share an id")
	}

	// The whole point: a second boot must not re-mint, or every row referencing
	// the old id is orphaned.
	second, err := EnsureTenantIdentities(t.Context(), database, configured)
	if err != nil {
		t.Fatalf("second boot: %v", err)
	}
	for slug, identity := range first {
		if second[slug].ID != identity.ID {
			t.Errorf("%s: id changed across boots: %q -> %q", slug, identity.ID, second[slug].ID)
		}
	}
}

func TestTenantIdentitySurvivesARename(t *testing.T) {
	database := dbtest.Open(t)
	before, err := EnsureTenantIdentities(t.Context(), database, []TenantIdentity{{Slug: "demo", Name: "Demohaus"}})
	if err != nil {
		t.Fatal(err)
	}
	// A changed display name must not disturb identity — that is what makes the
	// slug a label rather than a key.
	after, err := EnsureTenantIdentities(t.Context(), database, []TenantIdentity{{Slug: "demo", Name: "Haus am Park"}})
	if err != nil {
		t.Fatal(err)
	}
	if after["demo"].ID != before["demo"].ID {
		t.Fatalf("renaming changed the id: %q -> %q", before["demo"].ID, after["demo"].ID)
	}
	if after["demo"].Name != "Haus am Park" {
		t.Fatalf("name not refreshed: %q", after["demo"].Name)
	}
}

func TestTenantIdentityRejectsAnEmptySlug(t *testing.T) {
	database := dbtest.Open(t)
	if _, err := EnsureTenantIdentities(t.Context(), database, []TenantIdentity{{Slug: "", Name: "x"}}); err == nil {
		t.Fatal("an empty slug must fail closed; it would make identity unaddressable")
	}
}

func TestTenantIdentityConstraintRejectsMalformedIDsInTheDatabase(t *testing.T) {
	// Valid() and the schema CHECK are two statements of the same rule in two
	// languages. Asserting only the Go side would leave the database free to
	// accept what PostgreSQL later rejects, so exercise the constraint itself.
	database := dbtest.Open(t)
	for name, id := range map[string]string{
		"too short":     "0123456789ABCDEFGHJKMNPQR",
		"lowercase":     "0123456789abcdefghjkmnpqrs",
		"excluded I":    "0123456789ABCDEFGHIJKMNPQR",
		"leading digit": "80000000000000000000000000",
	} {
		_, err := database.ExecContext(t.Context(),
			`INSERT INTO tenant(tenant_id,slug,name,created_at,updated_at) VALUES($1,$2,'','x','x')`,
			id, "slug-"+name)
		if err == nil {
			t.Errorf("%s: the database accepted a malformed id %q", name, id)
		}
	}
}

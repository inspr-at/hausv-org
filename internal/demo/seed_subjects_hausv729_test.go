package demo

import (
	"path/filepath"
	"testing"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
)

// HAUSV-729: the open issues of one house (what the house overview lists as
// current work) must not share a subject. Loads the committed demo seed and
// checks the issues the loader actually opens, not the generator's intent.
func TestOpenIssuesOfAHouseHaveDistinctSubjectsHAUSV729(t *testing.T) {
	database, config := dbtest.OpenWithConfig(t)
	scoped, err := db.NewScoped(config, database)
	if err != nil {
		t.Fatal(err)
	}
	defer scoped.Close()
	const seedDir = "../../scripts/demo/seed"
	if _, err := Load(t.Context(), database, seedDir, SeedOptions{DocumentDir: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	var houses []seedHouse
	if err := readJSON(filepath.Join(seedDir, "houses.json"), &houses); err != nil {
		t.Fatal(err)
	}
	var tenants []store.TenantIdentity
	for _, house := range houses {
		tenants = append(tenants, store.TenantIdentity{Slug: house.Slug})
	}
	identities, err := store.EnsureTenantIdentities(t.Context(), database, tenants)
	if err != nil {
		t.Fatal(err)
	}
	storage := store.NewSQLIssueStore(store.NewTenantDB(scoped), t.TempDir())
	openIssues := 0
	for _, house := range houses {
		issues, ok := store.BindIssueRepository(storage, identities[house.Slug].Ref())
		if !ok {
			t.Fatalf("%s: issue repository unavailable", house.Slug)
		}
		seen := map[string]string{}
		for _, issue := range issues.List() {
			switch issue.Status {
			case store.IssueStatusNew, store.IssueStatusAccepted, store.IssueStatusScheduled, store.IssueStatusProgress:
			default:
				continue
			}
			openIssues++
			if other, dup := seen[issue.Title]; dup {
				t.Errorf("%s: open issues %s and %s share the subject %q", house.Slug, other, issue.ID, issue.Title)
			}
			seen[issue.Title] = issue.ID
		}
	}
	if openIssues < 40 {
		t.Fatalf("only %d open issues across the demo houses; the check would be vacuous", openIssues)
	}
}

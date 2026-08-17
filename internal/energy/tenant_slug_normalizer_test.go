package energy

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// TestTenantSlugNormalizersAgree is MUST-FIX 2 at its root.
//
// Two normalizers decided what a tenant slug is. This package's normalizeToken
// stripped everything outside [a-z0-9]; internal/tenantid resolves an identity
// with textutil.Slug, and internal/config accepts textutil.Slug output as the
// legal slug. So a configured house named "haus.a" already owned an identity
// under "haus.a", and every energy write asked for an identity for "haus-a" —
// minting a SECOND one and writing a WRONG tenant_id onto the row.
//
// A wrong tenant_id is worse than a missing one: the boot-time completeness
// check counts NULLs and can never see it.
//
// textutil.Slug wins because it is what the configuration layer and the identity
// resolver already use. This test fails the moment they diverge again.
//
// Where it is blind: it compares the two FUNCTIONS on a fixed table of inputs.
// It cannot prove agreement for an input nobody thought of, and it says nothing
// about a third normalizer appearing somewhere else in the codebase — only about
// these two.
func TestTenantSlugNormalizersAgree(t *testing.T) {
	for _, raw := range []string{
		"haus-a", "Haus_A", "haus.a", "haus a", "haus--a", "haus-", "-haus",
		"haus-grün", "HAUS-Ä", " demo ", "demo", "haus/a", "haus_a_1",
		"", "   ", "1", "a1-b2", "haus\ta", "Ö", "haus--a--b",
	} {
		got := normalizeSlug(raw)
		want := textutil.Slug(raw)
		if got != want {
			t.Errorf("normalizeSlug(%q) = %q but textutil.Slug(%q) = %q\n"+
				"two normalizers mint two identities for one house", raw, got, raw, want)
		}
	}
}

// TestHomeKeyNormalizationIsNotTheTenantNormalizer pins the deliberate other
// half of the fix. A home key is a tenant-LOCAL label that has never been a
// tenant slug and never resolves to an identity; routing it through textutil.Slug
// as well would silently change the stored key of every existing home whose key
// contains anything outside [a-z0-9-], making those rows unreachable. It keeps
// the strict token normalizer, and that is a decision rather than an oversight.
func TestHomeKeyNormalizationIsNotTheTenantNormalizer(t *testing.T) {
	for raw, want := range map[string]string{
		"Haus.A":     "haus-a",
		"Erdgeschoß": "erdgescho",
		"  ":         DefaultHomeKey,
		"":           DefaultHomeKey,
		"Wohnung 1":  "wohnung-1",
	} {
		if got := NormalizeHomeKey(raw); got != want {
			t.Errorf("NormalizeHomeKey(%q) = %q, want %q", raw, got, want)
		}
	}
}

// energyTenantTables are this package's tenant-scoped tables, as given a
// tenant_id column by SQLite migration 0033.
var energyTenantTables = map[string]bool{
	"home_profiles":             true,
	"energy_assets":             true,
	"energy_entity_mappings":    true,
	"energy_intervals":          true,
	"energy_imports":            true,
	"energy_maintenance_plans":  true,
	"energy_measures":           true,
	"energy_tariff_assessments": true,
}

var energyInsertedTable = regexp.MustCompile(`(?is)INSERT\s+INTO\s+([a-z_]+)`)

// TestEveryEnergyUpsertHealsTheIdentity is MUST-FIX 3 for this package: a row
// written by the PREVIOUS release carries a NULL tenant_id, and a write through
// the new binary must give it one instead of leaving it invisible to every
// tenant_id filter and to PostgreSQL's row-level-security policy.
//
// coalesce(<table>.tenant_id, excluded.tenant_id) is the required shape:
// PostgreSQL's tenant_id_immutable trigger rejects re-pointing an owned row, and
// only the coalesce form is guaranteed to be a no-op there.
//
// Where it is blind: it reads string literals, so a statement assembled at
// runtime is invisible; energy_imports uses ON CONFLICT DO NOTHING and therefore
// has no clause to carry a repair at all — such a row is healed by the boot pass
// or not at all.
func TestEveryEnergyUpsertHealsTheIdentity(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}
	checked := 0
	offenders := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			text := energyLiteralText(lit.Value)
			upper := strings.ToUpper(text)
			doUpdate := strings.Index(upper, "DO UPDATE")
			if !strings.Contains(upper, "ON CONFLICT") || doUpdate < 0 {
				return true
			}
			match := energyInsertedTable.FindStringSubmatch(text)
			if match == nil || !energyTenantTables[match[1]] {
				return true
			}
			checked++
			set := strings.ToLower(text[doUpdate:])
			if !strings.Contains(set, "coalesce("+match[1]+".tenant_id") {
				offenders = append(offenders, fset.Position(lit.Pos()).String()+
					": upsert on "+match[1]+" must assign coalesce("+match[1]+
					".tenant_id,excluded.tenant_id)\n    "+strings.Join(strings.Fields(text), " "))
			}
			return true
		})
	}
	if checked < 7 {
		t.Fatalf("only %d tenant-scoped upserts were found — the probe is broken, not the package", checked)
	}
	for _, offender := range offenders {
		t.Errorf("a write through the new binary must heal an unowned row\n  %s", offender)
	}
}

func energyLiteralText(raw string) string {
	if strings.HasPrefix(raw, "`") {
		return strings.Trim(raw, "`")
	}
	unquoted, err := strconv.Unquote(raw)
	if err != nil {
		return raw
	}
	return unquoted
}

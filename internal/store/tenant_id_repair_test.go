package store

import (
	"database/sql"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

// TestBootRepairsRowsWhoseStoredSlugIsNotCanonical is the boot brick, reproduced.
//
// mintMissingIdentities used to insert textutil.Slug(raw) into `tenant` while
// linkTenantIDs joined `tenant.slug = <table>.<slugColumn>` on the RAW column
// value. For any stored slug that is not already canonical — uppercase, an
// underscore, surrounding whitespace — the two could never meet: the mint
// created a canonical `tenant` row the join could not see, the row stayed NULL,
// verifyTenantIDsTx returned ErrTenantIDMissing, and EnsureTenantIdentities
// aborted newApp. Every restart repeated it, leaving another unreachable tenant
// row behind. The product was dead and the operator had no lead.
//
// This drives EnsureTenantIdentities — the call the boot actually makes — twice,
// because "converges" and "converges again" are different claims.
//
// Where it is blind: it seeds `announcements` only. The link is table-driven, so
// one table exercises the code path, but a table whose slugColumn is misdeclared
// in tenantIDTables would not be caught here (TestTenantIDTablesMatchTheMigratedSchema
// covers the table set, not the column). It also cannot prove that a slug form
// nobody listed here is safe; that the whole module agrees on what
// canonicalisation IS is asserted separately, by
// internal/textutil's TestSlugIsTheOnlyTenantSlugNormalizer.
func TestBootRepairsRowsWhoseStoredSlugIsNotCanonical(t *testing.T) {
	cases := []struct {
		stored    string
		canonical string
	}{
		{stored: "Haus_A", canonical: "haus-a"},
		{stored: "Demo", canonical: "demo"},
		{stored: "demo ", canonical: "demo"},
		{stored: " Haus_A ", canonical: "haus-a"},
		{stored: "haus.a", canonical: "haus.a"},
		{stored: "demo", canonical: "demo"},
	}
	for _, tc := range cases {
		t.Run(tc.stored, func(t *testing.T) {
			database := dbtest.Open(t)
			// On PostgreSQL the row this reproduces cannot exist since
			// migration 0006 made tenant_id NOT NULL; the helper proves that
			// refusal, and with it there is nothing left here to assert on
			// that engine. On SQLite — production, and the open rollback
			// window — it proves the row IS accepted and carries on.
			if dbtest.RollbackWindowClosed(t, database) {
				return
			}
			seedSlugOnlyRow(t, database, "announcements", tc.stored, "a1")
			configured := []TenantIdentity{{Slug: "demo", Name: "Demo"}}

			linked := ""
			for attempt := 1; attempt <= 2; attempt++ {
				if _, err := EnsureTenantIdentities(t.Context(), database, configured); err != nil {
					t.Fatalf("boot %d with stored slug %q: %v", attempt, tc.stored, err)
				}
				var got sql.NullString
				if err := database.QueryRowContext(t.Context(),
					`SELECT tenant_id FROM announcements WHERE id='a1'`).Scan(&got); err != nil {
					t.Fatalf("read row after boot %d: %v", attempt, err)
				}
				if !got.Valid {
					t.Fatalf("boot %d left the row without an identity", attempt)
				}
				if attempt == 1 {
					linked = got.String
				} else if got.String != linked {
					t.Fatalf("identity changed across boots: %q -> %q", linked, got.String)
				}
			}

			// The identity the row points at must be the one the CANONICAL slug
			// owns — the same one every configured tenant and every write path
			// resolves to. Linking to some other row would be invisible here
			// without this.
			var owner string
			if err := database.QueryRowContext(t.Context(),
				`SELECT slug FROM tenant WHERE tenant_id=$1`, linked).Scan(&owner); err != nil {
				t.Fatalf("read the tenant the row points at: %v", err)
			}
			if owner != tc.canonical {
				t.Fatalf("row linked to tenant %q, want %q", owner, tc.canonical)
			}

			// And no second, unreachable tenant row was left behind. Each failed
			// boot used to mint exactly one of those.
			wanted := map[string]bool{"demo": true, tc.canonical: true}
			assertTenantSlugs(t, database, wanted)
		})
	}
}

func assertTenantSlugs(t *testing.T, database *sql.DB, wanted map[string]bool) {
	t.Helper()
	rows, err := database.QueryContext(t.Context(), `SELECT slug FROM tenant ORDER BY slug`)
	if err != nil {
		t.Fatalf("list tenants: %v", err)
	}
	defer rows.Close()
	got := []string{}
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			t.Fatalf("scan tenant: %v", err)
		}
		got = append(got, slug)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate tenants: %v", err)
	}
	if len(got) != len(wanted) {
		t.Fatalf("tenant table = %v, want exactly the %d slugs %v", got, len(wanted), wanted)
	}
	for _, slug := range got {
		if !wanted[slug] {
			t.Fatalf("tenant table = %v, want exactly the slugs %v", got, wanted)
		}
	}
}

// TestTenantIDMissingErrorNamesTheOffendingSlugs: the count alone
// ("announcements=1") tells an operator that the boot refused and nothing about
// which row to fix. A boot-blocking error has to carry its own lead.
//
// Where it is blind: it caps how many slugs it reports, so a database with
// hundreds of distinct broken slugs names only the first few — deliberately, an
// error message is not a report — and it names the slug, not the primary key of
// the row.
func TestTenantIDMissingErrorNamesTheOffendingSlugs(t *testing.T) {
	database := dbtest.Open(t)
	if dbtest.RollbackWindowClosed(t, database) {
		return
	}
	// Whitespace-only and empty slugs canonicalise to nothing, so no identity
	// can be minted for them. They are the genuinely unrepairable case, and the
	// only one that should still refuse the boot.
	seedSlugOnlyRow(t, database, "announcements", "", "orphan-empty")
	seedSlugOnlyRow(t, database, "announcements", "   ", "orphan-blank")

	err := BackfillTenantIDs(t.Context(), database)
	if !errors.Is(err, ErrTenantIDMissing) {
		t.Fatalf("backfill error = %v, want ErrTenantIDMissing", err)
	}
	message := err.Error()
	if !strings.Contains(message, "announcements=2") {
		t.Errorf("the error must still name the table and the count, got: %s", message)
	}
	// %q so an operator can tell "" from "   ".
	for _, want := range []string{`""`, `"   "`} {
		if !strings.Contains(message, want) {
			t.Errorf("the error must name the offending slug %s, got: %s", want, message)
		}
	}
}

// TestWritesHealRowsLeftWithoutAnIdentity is MUST-FIX 3, reproduced.
//
// NOT NULL is genuinely blocked while the rollback window is open, so rows
// written by the PREVIOUS release carry a NULL tenant_id and are invisible to
// every tenant_id filter. Writing over such a row used to succeed, land the
// data, and leave it invisible: no error, no log, and the same repository's
// List() could not see what it had just written.
//
// The repair is the write itself: every upsert onto a tenant-scoped table now
// gives an unowned row its identity.
//
// Where it is blind: it exercises three tables, not fifteen, and only the ones
// whose rows are addressable by a natural key. A legacy announcement, event,
// document, issue, ballot, handover or attachment carries a RANDOM id, so no
// write ever collides with it and no upsert can heal it — those rows are
// repaired by the boot pass or not at all, which is exactly why MUST-FIX 1 had
// to be fixed as well. TestEveryTenantScopedUpsertHealsTheIdentity covers the
// remaining statements as SQL text rather than behaviour.
func TestWritesHealRowsLeftWithoutAnIdentity(t *testing.T) {
	database, lanes := testLanes(t)
	if dbtest.RollbackWindowClosed(t, database) {
		return
	}
	tenant := testTenantRef("demo")
	now := time.Now().UTC().Truncate(time.Second)

	// unit_payment_status: the reviewer's own reproduction — Set() returned nil,
	// the status landed, and List() could not see it.
	if _, err := database.ExecContext(t.Context(),
		`INSERT INTO unit_payment_status(tenant_slug, unit_id, status, updated_at, updated_by)
		 VALUES($1,$2,$3,$4,$5)`,
		"demo", "u1", UnitPaymentStatusOpen, now.Format(time.RFC3339Nano), "old@example.com"); err != nil {
		t.Fatalf("seed legacy unit payment status: %v", err)
	}
	payments, _ := BindUnitPaymentStatusRepository(NewSQLUnitPaymentStatusStore(lanes), tenant)
	if _, err := payments.Set(UnitPaymentStatus{
		UnitID: "u1", Status: UnitPaymentStatusPaid, UpdatedBy: "a@example.com",
	}); err != nil {
		t.Fatalf("set the legacy payment status: %v", err)
	}
	if got := payments.List(); len(got) != 1 {
		t.Errorf("unit_payment_status: the repository wrote a row it cannot see: List() = %d rows, want 1", len(got))
	}

	// announcement_reads: the per-person seen marker.
	if _, err := database.ExecContext(t.Context(),
		`INSERT INTO announcement_reads(tenant_slug, email, seen_at) VALUES($1,$2,$3)`,
		"demo", "a@example.com", now.Format(time.RFC3339Nano)); err != nil {
		t.Fatalf("seed legacy announcement read: %v", err)
	}
	reads, _ := BindAnnouncementReadRepository(NewSQLAnnouncementReadStore(lanes), tenant)
	if err := reads.MarkSeen("a@example.com", now.Add(time.Hour)); err != nil {
		t.Fatalf("mark seen: %v", err)
	}
	if seen := reads.LastSeen("a@example.com"); seen.IsZero() {
		t.Error("announcement_reads: the repository wrote a seen marker it cannot read back")
	}

	// units: a tenant lane neither sees nor adopts a row left with no tenant_id
	// (HAUSV-774). The write fails, the legacy row stays, and the boot backfill
	// below is what links it.
	if _, err := database.ExecContext(t.Context(),
		`INSERT INTO units(tenant_slug, id, data) VALUES($1,$2,'{"id":"u1","label":"alt"}')`,
		"demo", "u1"); err != nil {
		t.Fatalf("seed legacy unit: %v", err)
	}
	units, _ := BindUnitRepository(NewSQLUnitStore(lanes), tenant)
	if err := units.SetUnits([]Unit{{ID: "u1", Label: "Top 1"}}); err == nil {
		t.Fatal("set units adopted a row with no tenant_id")
	}
	if got := units.List(); len(got) != 0 {
		t.Errorf("units: tenant lane sees a row with no tenant_id: List() = %d rows", len(got))
	}
	var legacyID string
	if err := database.QueryRowContext(t.Context(),
		`SELECT id FROM units WHERE tenant_slug=$1 AND id=$2 AND tenant_id IS NULL`, "demo", "u1").Scan(&legacyID); err != nil {
		t.Fatalf("legacy unit was deleted or already linked: %v", err)
	}

	// And the boot check now agrees, on rows it previously refused to serve.
	if err := BackfillTenantIDs(t.Context(), database); err != nil {
		t.Fatalf("boot completeness check after the healing writes: %v", err)
	}
}

// insertedTable pulls the target table out of an INSERT statement.
var insertedTable = regexp.MustCompile(`(?is)INSERT\s+INTO\s+([a-z_]+)`)

// sqlLiteralsIn returns every string literal and literal concatenation in the
// non-test sources of one package directory, so a test can ask what SQL the
// package actually issues instead of what a hand-written list claims.
func sqlLiteralsIn(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}
	out := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.BinaryExpr:
				if node.Op == token.ADD {
					out = append(out, concatenatedText(node))
				}
			case *ast.BasicLit:
				if node.Kind == token.STRING {
					out = append(out, literalText(node.Value))
				}
			}
			return true
		})
	}
	return out
}

// TestEveryTenantScopedUpsertHealsTheIdentity reads the SQL this package
// actually issues and fails if an upsert onto a tenant-scoped table can leave a
// pre-existing row without its identity.
//
// It is the general form of TestWritesHealRowsLeftWithoutAnIdentity: every
// statement, no fixtures. coalesce(<table>.tenant_id, excluded.tenant_id) is the
// required shape rather than a bare excluded.tenant_id, because PostgreSQL's
// tenant_id_immutable trigger rejects re-pointing an OWNED row and only the
// coalesce form is a guaranteed no-op there.
//
// Where it is blind: it reads string literals and their concatenations, so an
// upsert assembled from a variable at runtime is invisible to it; it only reads
// internal/store (internal/energy has its own copy of this check); and it proves
// the clause is PRESENT, not that the value bound to it is the right tenant's.
func TestEveryTenantScopedUpsertHealsTheIdentity(t *testing.T) {
	scoped := map[string]bool{}
	for _, table := range tenantIDTables {
		scoped[table.name] = true
	}
	checked, offenders := auditUpsertHealing(t, ".", scoped)
	if checked < 15 {
		t.Fatalf("only %d tenant-scoped upserts were found — the probe is broken, not the package", checked)
	}
	for _, offender := range offenders {
		t.Errorf("a write through the new binary must heal an unowned row\n  %s", offender)
	}
}

// auditUpsertHealing is shared with internal/energy through a copy rather than an
// export: a test helper that becomes package API is a worse trade than the
// duplication.
func auditUpsertHealing(t *testing.T, dir string, scoped map[string]bool) (int, []string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
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
			var text string
			switch node := n.(type) {
			case *ast.BinaryExpr:
				if node.Op != token.ADD {
					return true
				}
				text = concatenatedText(node)
			case *ast.BasicLit:
				if node.Kind != token.STRING {
					return true
				}
				text = literalText(node.Value)
			default:
				return true
			}
			upper := strings.ToUpper(text)
			doUpdate := strings.Index(upper, "DO UPDATE")
			if !strings.Contains(upper, "ON CONFLICT") || doUpdate < 0 {
				return true
			}
			match := insertedTable.FindStringSubmatch(text)
			if match == nil || !scoped[match[1]] {
				// Either not a tenant-scoped table, or a fragment whose INSERT
				// half lives in another literal — the flattened concatenation is
				// visited as its own node and resolves there.
				return true
			}
			checked++
			set := strings.ToLower(text[doUpdate:])
			switch {
			case !strings.Contains(set, "tenant_id"):
				offenders = append(offenders,
					fset.Position(n.Pos()).String()+": upsert on "+match[1]+" never assigns tenant_id\n    "+collapse(text))
			case !strings.Contains(set, "coalesce("+match[1]+".tenant_id"):
				offenders = append(offenders,
					fset.Position(n.Pos()).String()+": upsert on "+match[1]+
						" must assign coalesce("+match[1]+".tenant_id,excluded.tenant_id)\n    "+collapse(text))
			}
			return false
		})
	}
	return checked, offenders
}

// concatenatedText flattens `"a" + ident + "b"` into a single string so a
// statement split across a shared column constant is read as one statement. A
// non-literal operand becomes a space rather than vanishing, so two fragments
// never accidentally join into a word that was never written.
func concatenatedText(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.BasicLit:
		if node.Kind == token.STRING {
			return literalText(node.Value)
		}
		return " "
	case *ast.BinaryExpr:
		if node.Op != token.ADD {
			return " "
		}
		return concatenatedText(node.X) + concatenatedText(node.Y)
	case *ast.ParenExpr:
		return concatenatedText(node.X)
	case *ast.Ident:
		if node.Name == "membershipColumns" {
			return membershipColumns
		}
		return " "
	default:
		return " "
	}
}

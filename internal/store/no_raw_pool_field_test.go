package store

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The lane conversion is enforced by a TYPE, not by discipline: a store holds a
// *TenantDB, which has no Query/QueryRow/Exec/Begin, so every statement site has
// to name a lane before it compiles. That enforcement has exactly one way to be
// undone, and it is not subtle — put a *sql.DB back in a store struct. The new
// store then compiles, its statements name no lane, and nothing else in the
// package notices, because the twenty converted stores keep working.
//
// So this reads the package's own source and asserts the absence. It is the only
// guard here that fires on code that has not been written yet: a store added
// next month with `db *sql.DB` fails this test on the first `go test`, without
// anyone remembering that lanes exist.
//
// WHERE IT IS BLIND, stated so nobody reads a pass as more than it is:
//
//   - It reads only THIS package. internal/energy still takes the pool and is
//     the one unconverted SQL surface; this test says nothing about it, and
//     will not notice a new package that takes pools either. internal/server
//     has its own sibling, TestServerReachesTheDatabaseOnlyThroughTheTenantSeam,
//     which additionally traces every statement call there to a lane.
//   - It checks struct FIELDS only. A function parameter, a return value, a
//     package-level var or a local that carries a *sql.DB is invisible to it,
//     and two such functions exist on purpose: BackfillTenantIDs and
//     EnsureTenantIdentities are boot-time maintenance that runs before there is
//     a request, let alone a tenant.
//   - It checks the TYPE NAME, not what flows through it. A field typed
//     db.Handle, any, or an interface the pool satisfies holds a pool just as
//     well and passes here. That hole is closed from the other side, by
//     TestTenantDBIsNotItselfADatabaseHandle and
//     TestTenantDBExposesOnlyTheTwoAccessors, not by this test.
//   - Test files are excluded deliberately. The fixtures hold the pool on
//     purpose: an assertion that read back through the same lane as the write
//     could not tell a correctly scoped row from a row the lane was hiding.
func TestNoStoreHoldsARawConnectionPool(t *testing.T) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read the store package directory: %v", err)
	}

	var offences []string
	filesParsed, structsSeen, fieldsSeen := 0, 0, 0

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		filesParsed++

		alias, dotImported := sqlPackageAlias(file)
		if dotImported {
			// A dot-import would make the pool spell itself `*DB`, which the
			// selector walk below cannot see. Refuse to report a pass this test
			// did not actually earn.
			t.Fatalf("%s dot-imports database/sql; this guard cannot see *sql.DB through a dot-import", name)
		}
		if alias == "" {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			structType, ok := n.(*ast.StructType)
			if !ok || structType.Fields == nil {
				return true
			}
			structsSeen++
			for _, field := range structType.Fields.List {
				fieldsSeen++
				if !mentionsType(field.Type, alias, "DB") {
					continue
				}
				offences = append(offences, describeField(fset, field))
			}
			return true
		})
	}

	// A source-reading test that walks nothing passes forever. These are the
	// numbers today; they only ever move down if the walk itself breaks.
	if filesParsed < 20 {
		t.Fatalf("only %d non-test files parsed in the store package; the walk is broken, not the package empty", filesParsed)
	}
	if structsSeen < 20 || fieldsSeen < 20 {
		t.Fatalf("walk found %d structs / %d fields; it is not reaching the declarations it claims to check", structsSeen, fieldsSeen)
	}

	if len(offences) > 0 {
		t.Fatalf("a store holds a raw *sql.DB, which lets its statements reach the database without naming a tenant lane:\n  %s\n\n"+
			"Take a *TenantDB instead. Retyping the field is the point: every statement site stops\n"+
			"compiling until it says For(tenant) or Unscoped(reason).",
			strings.Join(offences, "\n  "))
	}
}

// sqlPackageAlias reports the local name database/sql is imported under, and
// whether it was dot-imported.
func sqlPackageAlias(file *ast.File) (alias string, dotImported bool) {
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || path != "database/sql" {
			continue
		}
		if spec.Name == nil {
			return "sql", false
		}
		switch spec.Name.Name {
		case ".":
			return "", true
		case "_":
			return "", false
		default:
			return spec.Name.Name, false
		}
	}
	return "", false
}

// mentionsType looks for pkg.Name ANYWHERE inside a type expression, so
// []*sql.DB, map[string]*sql.DB and a bare embedded sql.DB are caught along with
// the plain pointer field.
func mentionsType(expr ast.Expr, pkg string, name string) bool {
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		selector, ok := n.(*ast.SelectorExpr)
		if !ok || selector.Sel == nil || selector.Sel.Name != name {
			return true
		}
		ident, ok := selector.X.(*ast.Ident)
		if ok && ident.Name == pkg {
			found = true
			return false
		}
		return true
	})
	return found
}

func describeField(fset *token.FileSet, field *ast.Field) string {
	position := fset.Position(field.Pos())
	names := make([]string, 0, len(field.Names))
	for _, ident := range field.Names {
		names = append(names, ident.Name)
	}
	label := "<embedded>"
	if len(names) > 0 {
		label = strings.Join(names, ", ")
	}
	return filepath.Base(position.Filename) + ":" + strconv.Itoa(position.Line) + " field " + label
}

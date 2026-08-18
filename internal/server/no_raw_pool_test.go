package server

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

// The lane conversion made every STORE name a lane per statement, and a guard in
// internal/store asserts no store can hold a pool. The server package had no
// such guard, and that is exactly where four statements hid: the import ledger
// ran a.db.QueryRow and a.db.Exec on the process pool from the app struct —
// outside every lane, invisible to the Unscoped inventory, and under a
// fail-closed policy they would have seen and written nothing. A review found
// them. A review is not a mechanism.
//
// This is the sibling of TestNoStoreHoldsARawConnectionPool for internal/server,
// and it asserts two things about the package's non-test source:
//
//  1. NO struct field is typed as a raw database surface: *sql.DB, *sql.Tx,
//     *sql.Conn or db.Handle. The app keeps the process pool for its lifecycle
//     only, behind a two-method interface (processPool) that has no Query, so
//     a handler cannot run a statement on it without first changing a type
//     that this test reads.
//  2. EVERY statement call in the package — Query with arguments, QueryRow,
//     Exec, their Context variants, Prepare, Begin, BeginTx — has a receiver
//     that this test can trace, inside the same function, to a lane:
//     <expr>.For(...) or <expr>.Unscoped(...), directly or through a local bound
//     to one (`h := a.tenantDB.For(t)`, `tx, err := h.Begin()`). Anything else
//     is an offence: a field, a parameter, a call to something that is not a
//     lane. The test does not try to guess what those are; it fails and names
//     the site.
//
// WHERE IT IS BLIND, so a pass is read for exactly what it is:
//
//   - It resolves receivers syntactically. A statement executor that arrives as
//     a function PARAMETER, or through a closure that captures newApp's boot-time
//     `database` local, is reported as unresolved rather than followed. That is
//     the fail-closed choice: today no such shape exists in this package, and
//     the day one appears the author extends this test with the trace, not the
//     allowlist.
//   - It checks the TYPE NAME of fields, not what flows through them. A field
//     typed `any` or a locally declared interface with a Query method would
//     pass rule 1; rule 2 still fires on the call, so the pair is what closes
//     the hole, not either half.
//   - Test files are excluded on purpose: fixtures and assertions read through
//     the pool deliberately, outside every lane, so they cannot be fooled by
//     the lane they are checking. testPool is the one place they get it.
func TestServerReachesTheDatabaseOnlyThroughTheTenantSeam(t *testing.T) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read the server package directory: %v", err)
	}

	var fieldOffences, statementOffences []string
	filesParsed, structsSeen, statementsSeen := 0, 0, 0

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

		aliases := rawSurfaceAliases(t, file, name)

		// Rule 1: no raw database surface as a struct field.
		ast.Inspect(file, func(n ast.Node) bool {
			structType, ok := n.(*ast.StructType)
			if !ok || structType.Fields == nil {
				return true
			}
			structsSeen++
			for _, field := range structType.Fields.List {
				for _, surface := range aliases {
					if mentionsQualifiedType(field.Type, surface.pkg, surface.name) {
						fieldOffences = append(fieldOffences, describeServerField(fset, field)+" is typed with "+surface.pkg+"."+surface.name)
					}
				}
			}
			return true
		})

		// Rule 2: every statement call names a lane.
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel == nil || !isStatementCall(selector.Sel.Name, len(call.Args)) {
					return true
				}
				statementsSeen++
				if problem := laneOfReceiver(fn, selector.X, 0); problem != "" {
					position := fset.Position(call.Pos())
					statementOffences = append(statementOffences,
						filepath.Base(position.Filename)+":"+strconv.Itoa(position.Line)+" "+fn.Name.Name+
							" calls ."+selector.Sel.Name+" on `"+exprText(selector.X)+"`: "+problem)
				}
				return true
			})
		}
	}

	// A source-reading test that walks nothing passes forever. The four ledger
	// statements are the floor: fewer means the walk stopped seeing calls, not
	// that the package stopped making them.
	if filesParsed < 30 {
		t.Fatalf("only %d non-test files parsed in the server package; the walk is broken, not the package empty", filesParsed)
	}
	if structsSeen < 30 {
		t.Fatalf("walk found %d structs; it is not reaching the declarations it claims to check", structsSeen)
	}
	if statementsSeen < 4 {
		t.Fatalf("walk found %d statement calls; the four import-ledger statements alone should be visible to it", statementsSeen)
	}

	if len(fieldOffences) > 0 {
		t.Errorf("a struct in the server package holds a raw database surface, which lets a handler reach the database without naming a tenant lane:\n  %s\n\n"+
			"Hold a *store.TenantDB and take For(tenant) or Unscoped(reason) at the statement; keep only the pool's LIFECYCLE (processPool) on the app.",
			strings.Join(fieldOffences, "\n  "))
	}
	if len(statementOffences) > 0 {
		t.Errorf("a statement in the server package runs on something this test cannot trace to a lane:\n  %s\n\n"+
			"Every Query/QueryRow/Exec/Begin in this package must sit on a.tenantDB.For(tenant) or a.tenantDB.Unscoped(reason), directly or through a local bound to one in the same function.\n"+
			"If the shape is legitimately new (a transaction handed in as a parameter, say), extend laneOfReceiver to trace it — do not allowlist the site.",
			strings.Join(statementOffences, "\n  "))
	}
}

// isStatementCall picks out the database/sql calls that run or prepare SQL, or
// open a transaction that will. Query is only counted with arguments so that
// url.URL.Query() — dozens of them in this package — is not mistaken for it.
func isStatementCall(name string, args int) bool {
	switch name {
	case "QueryRow", "Exec", "QueryContext", "QueryRowContext", "ExecContext", "Prepare", "PrepareContext", "Begin", "BeginTx":
		return true
	case "Query":
		return args > 0
	}
	return false
}

// laneOfReceiver reports "" when expr is a lane — X.For(...) or X.Unscoped(...)
// — or a local bound to one within fn, possibly through .Begin(); otherwise it
// reports why it could not be traced.
func laneOfReceiver(fn *ast.FuncDecl, expr ast.Expr, depth int) string {
	if depth > 8 {
		return "binding chain deeper than this test follows"
	}
	switch node := expr.(type) {
	case *ast.ParenExpr:
		return laneOfReceiver(fn, node.X, depth+1)
	case *ast.CallExpr:
		selector, ok := node.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel == nil {
			return "receiver is a call that is not a lane accessor"
		}
		switch selector.Sel.Name {
		case "For", "Unscoped":
			return ""
		case "Begin", "BeginTx":
			return laneOfReceiver(fn, selector.X, depth+1)
		}
		return "receiver is `" + exprText(node) + "`, which is not For(...) or Unscoped(...)"
	case *ast.Ident:
		bindings := localBindings(fn, node.Name)
		if len(bindings) == 0 {
			return "`" + node.Name + "` is not bound to a lane in this function (a parameter, a receiver, or a package-level value)"
		}
		for _, rhs := range bindings {
			if problem := laneOfReceiver(fn, rhs, depth+1); problem != "" {
				return "`" + node.Name + "` is bound to `" + exprText(rhs) + "`: " + problem
			}
		}
		return ""
	case *ast.SelectorExpr:
		return "receiver is a field or package value, not a lane"
	}
	return "receiver shape `" + exprText(expr) + "` is not one this test traces"
}

// localBindings returns every right-hand side assigned to name inside fn.
func localBindings(fn *ast.FuncDecl, name string) []ast.Expr {
	var out []ast.Expr
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			for i, lhs := range node.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok || ident.Name != name {
					continue
				}
				switch {
				case len(node.Rhs) == len(node.Lhs):
					out = append(out, node.Rhs[i])
				case len(node.Rhs) == 1:
					out = append(out, node.Rhs[0])
				}
			}
		case *ast.ValueSpec:
			for i, ident := range node.Names {
				if ident.Name != name {
					continue
				}
				switch {
				case len(node.Values) == len(node.Names):
					out = append(out, node.Values[i])
				case len(node.Values) == 1:
					out = append(out, node.Values[0])
				}
			}
		}
		return true
	})
	return out
}

type qualifiedType struct {
	pkg  string // local import alias
	name string
}

// rawSurfaceAliases lists, for one file, the raw database surfaces under the
// names that file imports them by. A dot-import would hide them from the
// selector walk, so it is refused rather than passed.
func rawSurfaceAliases(t *testing.T, file *ast.File, filename string) []qualifiedType {
	t.Helper()
	var out []qualifiedType
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		var names []string
		var defaultAlias string
		switch {
		case path == "database/sql":
			names, defaultAlias = []string{"DB", "Tx", "Conn"}, "sql"
		case strings.HasSuffix(path, "/internal/db"):
			names, defaultAlias = []string{"Handle"}, "db"
		default:
			continue
		}
		alias := defaultAlias
		if spec.Name != nil {
			switch spec.Name.Name {
			case ".":
				t.Fatalf("%s dot-imports %s; this guard cannot see its types through a dot-import", filename, path)
			case "_":
				continue
			default:
				alias = spec.Name.Name
			}
		}
		for _, name := range names {
			out = append(out, qualifiedType{pkg: alias, name: name})
		}
	}
	return out
}

// mentionsQualifiedType looks for pkg.Name ANYWHERE inside a type expression, so
// []*sql.DB, map[string]*sql.DB and an embedded sql.DB are caught along with the
// plain pointer field.
func mentionsQualifiedType(expr ast.Expr, pkg string, name string) bool {
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

func describeServerField(fset *token.FileSet, field *ast.Field) string {
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

// exprText renders a small expression for a failure message without needing
// the source: enough to name a receiver, not to reproduce a statement.
func exprText(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.Ident:
		return node.Name
	case *ast.SelectorExpr:
		return exprText(node.X) + "." + node.Sel.Name
	case *ast.CallExpr:
		return exprText(node.Fun) + "(...)"
	case *ast.ParenExpr:
		return "(" + exprText(node.X) + ")"
	case *ast.StarExpr:
		return "*" + exprText(node.X)
	case *ast.IndexExpr:
		return exprText(node.X) + "[...]"
	case *ast.BasicLit:
		return node.Value
	}
	return "<expr>"
}

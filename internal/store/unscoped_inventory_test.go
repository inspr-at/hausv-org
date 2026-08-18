package store

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Every Unscoped call is a decision to cross tenants. The lane seam makes that
// decision impossible to take by ACCIDENT — a store cannot reach the database
// without writing For or Unscoped — but it cannot make the decision impossible
// to take CASUALLY. Unscoped("...") compiles just as readily as For(tenant), and
// the reason string is not read by anything at runtime.
//
// So the cross-tenant surface is written down here, whole, as a golden file: 78
// declared cross-tenant call sites (65 in the stores, 10 in internal/energy, 2
// in the import ledger in internal/server, 1 in the data mover command in
// cmd/hausv-org, 1 in internal/dbtest — the maintenance view every test fixture
// reads and writes through since migration 0006 closed the pool) plus the seam's
// own forwarder, each with the reason its author typed. The value is in the diff.
// Adding a cross-tenant call is a two-line change in a store plus a line in this
// file, and that second line is what a reviewer sees without having to know the
// lane mechanism exists or think to grep for it.
//
// Regenerate deliberately, never reflexively:
//
//	HAUSV_UPDATE_UNSCOPED_GOLDEN=1 go test ./internal/store -run TestUnscopedCallSiteInventory
//
// WHERE IT IS BLIND — this is an inventory, not a proof:
//
//   - It records the reason, and has NO opinion on whether the reason is true.
//     A site that copies its neighbour's justification passes unchanged. The
//     golden makes the claim reviewable; a human still has to review it.
//   - It counts CALLS to Unscoped, not statements that run cross-tenant. Several
//     sites hoist one handle and run two or three statements on it — every
//     Import* does — so this undercounts the executed surface on purpose,
//     because the decision is what is being inventoried.
//   - Cross-tenant access that never calls Unscoped is invisible to it, and
//     some exists: the boot-time BackfillTenantIDs / EnsureTenantIdentities
//     take a *sql.DB directly. Those are outside the seam, not exceptions
//     inside it. (internal/energy used to be on that list; it is on lanes now
//     and its ten sites are in the golden.)
//   - A reason built at runtime is recorded as its SOURCE TEXT under kind
//     "dynamic", not as a value. Exactly one such site exists today and it is
//     the seam's own forwarder, TenantDB.Unscoped, passing its parameter
//     through. A second appearing is the signal that a reason has stopped being
//     greppable, which is precisely what should show up in review. A reason
//     named through THIS package's exported constants from another package
//     (store.HealOrphanReason in internal/server) is resolved to its text under
//     kind "const:store.<Name>", so it is not mistaken for a runtime value.
//   - Moving or renaming a function churns the golden without changing the
//     surface, because the entry is keyed on file and function rather than on a
//     line number that every unrelated edit would move.
func TestUnscopedCallSiteInventory(t *testing.T) {
	root := repositoryRoot(t)
	sites := collectUnscopedCallSites(t, root)

	// An inventory that finds nothing agrees with an empty golden forever.
	if len(sites) < 50 {
		t.Fatalf("only %d Unscoped call sites found; the scan is broken, not the surface gone", len(sites))
	}

	golden := filepath.Join("testdata", "unscoped_calls.golden")
	got := strings.Join(sites, "\n") + "\n"

	if os.Getenv("HAUSV_UPDATE_UNSCOPED_GOLDEN") != "" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("create testdata: %v", err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatalf("write %s: %v", golden, err)
		}
		t.Logf("rewrote %s with %d cross-tenant call sites", golden, len(sites))
		return
	}

	wantRaw, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read %s: %v\n\nRegenerate with HAUSV_UPDATE_UNSCOPED_GOLDEN=1", golden, err)
	}
	want := string(wantRaw)
	if got == want {
		return
	}

	added, removed := diffLines(strings.Split(strings.TrimRight(want, "\n"), "\n"), sites)
	var report strings.Builder
	report.WriteString("the cross-tenant surface changed.\n")
	if len(added) > 0 {
		report.WriteString("\nNEW cross-tenant call sites (each one needs a reviewer to agree it is genuinely cross-tenant):\n")
		for _, line := range added {
			report.WriteString("  + " + line + "\n")
		}
	}
	if len(removed) > 0 {
		report.WriteString("\nGONE from the cross-tenant surface:\n")
		for _, line := range removed {
			report.WriteString("  - " + line + "\n")
		}
	}
	report.WriteString("\nIf every change above is intended, record it:\n")
	report.WriteString("  HAUSV_UPDATE_UNSCOPED_GOLDEN=1 go test ./internal/store -run TestUnscopedCallSiteInventory\n")
	t.Fatal(report.String())
}

// collectUnscopedCallSites returns one sorted "path\tfunction\tkind\treason"
// line per Unscoped call in the non-test sources under internal/ and cmd/.
func collectUnscopedCallSites(t *testing.T, root string) []string {
	t.Helper()
	// This package's exported string constants, so a qualified reference from
	// another package resolves to the same text a bare one does here.
	storeConstants := map[string]string{}
	storeFiles, err := os.ReadDir(filepath.Join(root, "internal", "store"))
	if err != nil {
		t.Fatalf("read internal/store: %v", err)
	}
	for _, entry := range storeFiles {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "internal", "store", name), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse internal/store/%s: %v", name, err)
		}
		collectStringConstants(parsed, storeConstants)
	}
	for name := range storeConstants {
		if !ast.IsExported(name) {
			delete(storeConstants, name)
		}
	}
	var lines []string
	for _, tree := range []string{"internal", "cmd"} {
		base := filepath.Join(root, tree)
		if _, err := os.Stat(base); err != nil {
			continue
		}
		dirs := map[string]bool{}
		err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if entry.Name() == "testdata" || entry.Name() == "vendor" {
					return fs.SkipDir
				}
				dirs[path] = true
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", base, err)
		}
		for dir := range dirs {
			lines = append(lines, unscopedCallsInPackage(t, root, dir, storeConstants)...)
		}
	}
	slices.Sort(lines)
	return lines
}

func unscopedCallsInPackage(t *testing.T, root string, dir string, storeConstants map[string]string) []string {
	t.Helper()
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	files := map[string]*ast.File{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", filepath.Join(dir, name), err)
		}
		files[name] = parsed
	}
	if len(files) == 0 {
		return nil
	}

	// String constants of this package, so a reason hoisted into a named
	// constant — HealOrphanReason is one — lands in the golden as its TEXT.
	// A reason nobody can read is not a reason.
	constants := map[string]string{}
	for _, file := range files {
		collectStringConstants(file, constants)
	}

	var lines []string
	for name, file := range files {
		relative := filepath.ToSlash(mustRelative(t, root, filepath.Join(dir, name)))
		qualified := qualifiedStoreConstants(file, storeConstants)
		for _, decl := range file.Decls {
			owner := "<package-level>"
			if fn, ok := decl.(*ast.FuncDecl); ok {
				owner = functionLabel(fn)
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) != 1 {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel == nil || selector.Sel.Name != "Unscoped" {
					return true
				}
				kind, reason := describeReason(fset, call.Args[0], constants, qualified)
				lines = append(lines, strings.Join([]string{relative, owner, kind, reason}, "\t"))
				return true
			})
		}
	}
	return lines
}

func collectStringConstants(file *ast.File, into map[string]string) {
	for _, decl := range file.Decls {
		general, ok := decl.(*ast.GenDecl)
		if !ok || general.Tok != token.CONST {
			continue
		}
		for _, spec := range general.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || len(value.Names) != len(value.Values) {
				continue
			}
			for i, name := range value.Names {
				literal, ok := value.Values[i].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					continue
				}
				text, err := strconv.Unquote(literal.Value)
				if err != nil {
					continue
				}
				into[name.Name] = text
			}
		}
	}
}

// qualifiedStoreConstants maps "alias.Name" to text for every exported store
// constant, under the alias this file imports internal/store by. Empty when the
// file does not import it.
func qualifiedStoreConstants(file *ast.File, storeConstants map[string]string) map[string]string {
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || !strings.HasSuffix(path, "/internal/store") {
			continue
		}
		alias := "store"
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		out := make(map[string]string, len(storeConstants))
		for name, text := range storeConstants {
			out[alias+"."+name] = text
		}
		return out
	}
	return nil
}

// describeReason classifies the argument so the golden distinguishes a reason
// that can be read from source from one that only exists at runtime.
func describeReason(fset *token.FileSet, arg ast.Expr, constants map[string]string, qualified map[string]string) (kind string, reason string) {
	switch node := arg.(type) {
	case *ast.BasicLit:
		if node.Kind == token.STRING {
			if text, err := strconv.Unquote(node.Value); err == nil {
				return "literal", flatten(text)
			}
		}
	case *ast.Ident:
		if text, ok := constants[node.Name]; ok {
			return "const:" + node.Name, flatten(text)
		}
	case *ast.SelectorExpr:
		if pkg, ok := node.X.(*ast.Ident); ok && node.Sel != nil {
			if text, ok := qualified[pkg.Name+"."+node.Sel.Name]; ok {
				return "const:" + pkg.Name + "." + node.Sel.Name, flatten(text)
			}
		}
	}
	return "dynamic", flatten(nodeSource(fset, arg))
}

func nodeSource(fset *token.FileSet, node ast.Node) string {
	start := fset.Position(node.Pos())
	end := fset.Position(node.End())
	if start.Filename != end.Filename || start.Offset >= end.Offset {
		return "<unreadable>"
	}
	source, err := os.ReadFile(start.Filename)
	if err != nil || end.Offset > len(source) {
		return "<unreadable>"
	}
	return string(source[start.Offset:end.Offset])
}

func functionLabel(fn *ast.FuncDecl) string {
	name := fn.Name.Name
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return name
	}
	return "(" + flatten(typeLabel(fn.Recv.List[0].Type)) + ")." + name
}

func typeLabel(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.StarExpr:
		return "*" + typeLabel(node.X)
	case *ast.Ident:
		return node.Name
	case *ast.SelectorExpr:
		return typeLabel(node.X) + "." + node.Sel.Name
	case *ast.IndexExpr:
		return typeLabel(node.X)
	default:
		return "?"
	}
}

// flatten keeps every entry on exactly one tab-separated line, so the golden
// stays diffable no matter how a reason is wrapped in source.
func flatten(text string) string {
	text = strings.ReplaceAll(text, "\t", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	return strings.Join(strings.Fields(text), " ")
}

func mustRelative(t *testing.T, root string, path string) string {
	t.Helper()
	relative, err := filepath.Rel(root, path)
	if err != nil {
		t.Fatalf("relative path for %s: %v", path, err)
	}
	return relative
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the store package: cannot locate the repository root")
		}
		dir = parent
	}
}

// diffLines reports the multiset difference between two sorted line lists.
func diffLines(want []string, got []string) (added []string, removed []string) {
	counts := map[string]int{}
	for _, line := range want {
		counts[line]++
	}
	for _, line := range got {
		if counts[line] > 0 {
			counts[line]--
			continue
		}
		added = append(added, line)
	}
	for line, n := range counts {
		for range n {
			removed = append(removed, line)
		}
	}
	slices.Sort(removed)
	return added, removed
}

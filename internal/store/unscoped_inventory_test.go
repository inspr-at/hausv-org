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
// So the cross-tenant surface is written down here, whole, as a golden file: 65
// declared cross-tenant call sites plus the seam's own forwarder, each with the
// reason its author typed. The value is in the diff.
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
//     some exists: internal/energy still takes the process pool, and the
//     boot-time BackfillTenantIDs / EnsureTenantIdentities take a *sql.DB
//     directly. Those are outside the seam, not exceptions inside it.
//   - A reason built at runtime is recorded as its SOURCE TEXT under kind
//     "dynamic", not as a value. Exactly one such site exists today and it is
//     the seam's own forwarder, TenantDB.Unscoped, passing its parameter
//     through. A second appearing is the signal that a reason has stopped being
//     greppable, which is precisely what should show up in review.
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
			lines = append(lines, unscopedCallsInPackage(t, root, dir)...)
		}
	}
	slices.Sort(lines)
	return lines
}

func unscopedCallsInPackage(t *testing.T, root string, dir string) []string {
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
	// constant — healOrphanReason is one — lands in the golden as its TEXT.
	// A reason nobody can read is not a reason.
	constants := map[string]string{}
	for _, file := range files {
		collectStringConstants(file, constants)
	}

	var lines []string
	for name, file := range files {
		relative := filepath.ToSlash(mustRelative(t, root, filepath.Join(dir, name)))
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
				kind, reason := describeReason(fset, call.Args[0], constants)
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

// describeReason classifies the argument so the golden distinguishes a reason
// that can be read from source from one that only exists at runtime.
func describeReason(fset *token.FileSet, arg ast.Expr, constants map[string]string) (kind string, reason string) {
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

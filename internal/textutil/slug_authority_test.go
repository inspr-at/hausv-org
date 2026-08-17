package textutil_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// TestSlugIsTheOnlyTenantSlugNormalizer keeps one concept to one function.
//
// Two normalizers for "what is a tenant slug" is what MUST-FIX 2 was:
// internal/energy stripped everything outside [a-z0-9] while internal/config,
// internal/tenantid and internal/server used textutil.Slug. While slugs were
// only labels the divergence merely produced odd keys. Once every write carried
// a tenant_id resolved FROM the slug, it started minting a second identity for a
// house that already had one — writing a WRONG tenant_id, which the boot-time
// completeness check can never see because it counts NULLs.
//
// So textutil.Slug is the authority, and any function in this module that
// presents itself as a slug normalizer has to be a delegation to it, not a
// second opinion.
//
// Where it is blind, stated here rather than discovered later:
//   - It recognises normalizers BY NAME (normalizeSlug / NormalizeSlug /
//     slugify). A third normalizer called something else — the way
//     internal/energy's normalizeToken still exists for asset ids and home keys,
//     legitimately — is invisible to it. Name-matching is what makes it cheap;
//     it is a guard against the mistake recurring in the same shape, not a proof
//     that no other normalizer exists.
//   - It reads source, so a normalizer that delegates to textutil.Slug and then
//     post-processes the result in a helper would pass.
func TestSlugIsTheOnlyTenantSlugNormalizer(t *testing.T) {
	root := filepath.Join("..", "..")
	found := 0
	offenders := []string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "vendor", "doctrine", "doctrine-private":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil // not our file to police
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			switch fn.Name.Name {
			case "normalizeSlug", "NormalizeSlug", "slugify", "Slugify":
			default:
				continue
			}
			found++
			if !delegatesToTextutilSlug(fn) {
				offenders = append(offenders, fset.Position(fn.Pos()).String()+": "+fn.Name.Name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk the module: %v", err)
	}
	if found < 2 {
		// Without this the test would pass on a tree it failed to read, which is
		// the failure mode a grep script has.
		t.Fatalf("only %d slug normalizer(s) found — the probe is broken, not the module", found)
	}
	for _, offender := range offenders {
		t.Errorf("a tenant slug normalizer must be `return textutil.Slug(raw)` and nothing else;\n"+
			"  a second opinion here mints a second identity for one house\n  %s", offender)
	}
}

// delegatesToTextutilSlug reports whether the function body is exactly
// `return textutil.Slug(<arg>)`.
func delegatesToTextutilSlug(fn *ast.FuncDecl) bool {
	if len(fn.Body.List) != 1 {
		return false
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return false
	}
	call, ok := ret.Results[0].(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Slug" {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && pkg.Name == "textutil"
}

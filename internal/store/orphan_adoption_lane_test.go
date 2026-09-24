package store

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	appdb "github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
)

// HealOrphanReason documents a lane decision: an INSERT ... ON CONFLICT ... DO
// UPDATE on a table that migration 0003 governs can land on a row the previous
// release left with a NULL tenant_id, and from a tenant lane that row is neither
// visible nor writable, so the upsert has to run on the maintenance lane. The
// comment on the constant used to say that class was three tables. It was eight.
// Nothing noticed, because the only thing asserting which upserts sit on which
// lane was that comment.
//
// This test is the comment turned into a check. It ENUMERATES, from source,
// every ON CONFLICT ... DO UPDATE statement in the non-test tree, keeps the ones
// whose target table carries tenant_id — the table set is read from the migrated
// schema, not typed here, so a new tenant-bound table joins the check the day its
// migration lands — and for each one traces the executor back to the lane it was
// taken from: through a `tx` parameter to every caller that opened the
// transaction, through a hoisted `unscoped := ...` local, through
// `<lane>.Begin()`. Then it demands one of three things:
//
//   - the lane is Unscoped(HealOrphanReason), which is the default expectation;
//   - or the table is listed in orphanAdoptionExemptions with the lane class it
//     is allowed on and a written reason why that class cannot lose an orphan;
//   - or the test fails, naming the site, the lane it found and the exemption it
//     would need.
//
// An exemption nobody relies on fails the test too, so the list cannot rot.
//
// WHERE IT IS BLIND — read a pass for exactly this much:
//
//   - It resolves executors syntactically, within one package. A transaction
//     handed across a package boundary, or an executor reached through an
//     interface, comes back as "unresolved", which FAILS rather than passes; the
//     analysis refuses to guess. Every shape the tree uses today resolves.
//   - It matches upsert SQL by regular expression over the string literal (and
//     over literal concatenations such as `INSERT INTO t(`+columns+`) ...`). SQL
//     assembled at runtime from a non-literal is invisible to the statement
//     scan, so a separate pass fails on ANY string literal containing
//     ON CONFLICT ... DO UPDATE that is not the direct argument of a statement
//     call: an upsert this test cannot see is reported, not skipped.
//   - It says nothing about DO NOTHING upserts. Those cannot heal and cannot
//     fail on the orphan: the conflict path writes no row, so the RLS USING
//     check never runs. The import replays are all of that shape.
//   - The exemption REASONS are prose. The test checks that a reason exists and
//     is exercised, not that it is true; a reviewer still reads them.
func TestEveryOrphanAdoptingUpsertNamesTheMaintenanceLane(t *testing.T) {
	root := repositoryRoot(t)
	governed := rlsGovernedTables(t)
	if len(governed) < 20 {
		t.Fatalf("only %d tenant-bound tables read from the schema; the table walk is broken, not the schema small", len(governed))
	}

	sites, orphanLiterals := collectUpsertSites(t, root, governed)
	if len(sites) < 20 {
		t.Fatalf("only %d ON CONFLICT ... DO UPDATE sites on tenant-bound tables found; the scan is broken, not the surface gone", len(sites))
	}

	var failures []string
	for _, literal := range orphanLiterals {
		failures = append(failures, "  "+literal+"\n      an ON CONFLICT ... DO UPDATE literal that is not the direct SQL argument of a statement call;\n      this test cannot see its lane, so it refuses to pass it")
	}

	exercised := map[string]bool{}
	for _, site := range sites {
		exemption, exempted := orphanAdoptionExemptions[site.table]
		for _, unresolved := range site.unresolved {
			failures = append(failures, "  "+site.label()+"\n      executor could not be traced to a lane: "+unresolved)
		}
		for _, lane := range site.lanes {
			class := lane.class()
			if class == laneHeal {
				continue
			}
			if exempted && slices.Contains(exemption.allowed, class) {
				exercised[site.table] = true
				continue
			}
			failures = append(failures, "  "+site.label()+"\n      runs on "+lane.String()+
				"\n      an upsert on a tenant-bound table can collide with a row that has no tenant_id, and a tenant lane can neither see nor write that row."+
				"\n      Either take Unscoped(HealOrphanReason) for the statement (or the transaction it runs in),"+
				"\n      or add "+site.table+" to orphanAdoptionExemptions with the class "+string(class)+" and a reason it cannot lose an orphan.")
		}
	}
	for table, exemption := range orphanAdoptionExemptions {
		if !exercised[table] {
			failures = append(failures, "  exemption for "+table+" ("+strings.Join(classNames(exemption.allowed), ", ")+
				") is not relied on by any upsert: it is stale, delete it so the list stays true")
		}
	}

	if len(failures) > 0 {
		sort.Strings(failures)
		var report strings.Builder
		report.WriteString("the maintenance-lane rule for orphan-adopting upserts is broken:\n\n")
		report.WriteString(strings.Join(failures, "\n"))
		report.WriteString("\n\nEvery ON CONFLICT ... DO UPDATE on a tenant-bound table, with the lane it was traced to:\n")
		for _, site := range sites {
			report.WriteString("  " + site.label() + "\n")
			for _, lane := range site.lanes {
				report.WriteString("      " + lane.String() + "\n")
			}
			for _, unresolved := range site.unresolved {
				report.WriteString("      UNRESOLVED " + unresolved + "\n")
			}
		}
		t.Fatal(report.String())
	}
}

// laneClass is how a resolved lane is judged, not how it was spelled.
type laneClass string

const (
	// laneHeal is Unscoped(HealOrphanReason): the expected answer.
	laneHeal laneClass = "unscoped(HealOrphanReason)"
	// laneUnscopedOther is the maintenance lane taken for a reason that is not
	// the heal. The orphan is reachable, but the site will not revert to
	// For(tenant) when tenant_id goes NOT NULL, so it needs its own justification.
	laneUnscopedOther laneClass = "unscoped(other reason)"
	// laneFor is a tenant lane. An orphan is unreachable from it.
	laneFor laneClass = "for(tenant)"
	// laneRaw is an executor that is not a lane at all: a *sql.DB field or a
	// value the seam never produced. Outside every guard the seam provides.
	// Nothing in the tree is allowed on it any more; it stays a class so that
	// a raw executor is REPORTED as one rather than as "unresolved".
	laneRaw laneClass = "raw pool"
)

func classNames(classes []laneClass) []string {
	out := make([]string, 0, len(classes))
	for _, class := range classes {
		out = append(out, string(class))
	}
	return out
}

type orphanAdoptionExemption struct {
	allowed []laneClass
	why     string
}

// randomIDReason is the argument for the tables whose primary key is minted by
// the store or by its caller (handovers take theirs from the server, contacts
// and issues only when the caller left it empty), spelled out once because it
// covers eight tables and each of them has to be able to point at it.
const randomIDReason = "the conflict target is (tenant_slug, id) and id is minted by the store or its caller as a random token or ULID: a NEW row can never " +
	"collide with a row the previous release wrote, and an EDIT addresses a row the same tenant lane has just read, which an orphan by " +
	"definition is not. A caller inventing an id that happens to be an orphan's is refused by RLS with SQLSTATE 42501 — loudly, never silently — " +
	"so the tenant lane is the right lane and the flip changes nothing here."

// orphanAdoptionExemptions is every tenant-bound table whose DO UPDATE upserts
// are allowed on a lane other than Unscoped(HealOrphanReason), with the class
// they are allowed on and the reason. Keyed by table because the property is a
// property of the key shape, not of the function that writes it.
var orphanAdoptionExemptions = map[string]orphanAdoptionExemption{
	"announcements": {allowed: []laneClass{laneFor}, why: randomIDReason},
	"attachments":   {allowed: []laneClass{laneFor}, why: randomIDReason},
	"ballots":       {allowed: []laneClass{laneFor}, why: randomIDReason},
	"contacts":      {allowed: []laneClass{laneFor}, why: randomIDReason},
	"documents":     {allowed: []laneClass{laneFor}, why: randomIDReason},
	"events":        {allowed: []laneClass{laneFor}, why: randomIDReason},
	"handovers": {allowed: []laneClass{laneFor, laneUnscopedOther}, why: randomIDReason + " ConfirmByToken additionally " +
		"rewrites the handover from its confirmation link, which names a token and no tenant, so that one write is on the maintenance " +
		"lane for a structural reason: the orphan is reachable there, and the lane will not revert to For(tenant)."},
	"issues":  {allowed: []laneClass{laneFor}, why: randomIDReason},
	"parking": {allowed: []laneClass{laneFor}, why: "PRIMARY KEY is tenant_slug and tenant_id is NOT NULL from the first parking migration; there is no orphan generation, so For(tenant) is the live write lane."},

	"announcement_reads":       {allowed: []laneClass{laneFor}, why: notNullTenantReason},
	"home_connectors":          {allowed: []laneClass{laneFor}, why: notNullTenantReason},
	"home_connector_readings":  {allowed: []laneClass{laneFor}, why: notNullTenantReason},
	"home_profiles":            {allowed: []laneClass{laneFor}, why: notNullTenantReason},
	"energy_assets":            {allowed: []laneClass{laneFor}, why: notNullTenantReason},
	"energy_entity_mappings":   {allowed: []laneClass{laneFor}, why: notNullTenantReason},
	"energy_intervals":         {allowed: []laneClass{laneFor}, why: notNullTenantReason},
	"energy_maintenance_plans": {allowed: []laneClass{laneFor}, why: notNullTenantReason},
	"energy_measures":          {allowed: []laneClass{laneFor}, why: notNullTenantReason},
	"unit_payment_status":      {allowed: []laneClass{laneFor}, why: "PostgreSQL migration 0006 makes tenant_id NOT NULL; SQLite has no RLS and boot backfills legacy rows. HAUSV-783 payment batches and manual writes use the tenant lane."},

	"home_portals": {allowed: []laneClass{laneUnscopedOther}, why: "Activate is the moment the tenant identity is MINTED: there is no TenantRef " +
		"before it runs, so its transaction is on the maintenance lane for a structural reason that outlives the flip. The upsert still " +
		"coalesces tenant_id, so an orphan portal row is adopted — on the one lane that can reach it."},
	"house_memberships": {allowed: []laneClass{laneFor, laneUnscopedOther}, why: "PostgreSQL migration 0006 enforces tenant_id NOT NULL, so SetMembership resolves the registry identity and writes on For(tenant). " +
		"Home-portal activation, whole-profile writes and organisation membership transactions still span houses by design and declare the maintenance lane."},
}

// notNullTenantReason is why the ordinary writes HAUSV-779 moved back onto
// For(tenant) no longer need the heal lane. Migration 0006 made tenant_id NOT
// NULL on these tables, so the orphan the heal was introduced to reach cannot
// exist. coalesce(tenant_id) stays in the SQL so an owned row is not re-pointed.
const notNullTenantReason = "migration 0006 made tenant_id NOT NULL, so the orphan HealOrphanReason was introduced to reach cannot exist; the upsert is on For(tenant)"

// upsertLane is one lane an upsert was traced to, with where the decision was
// made so a failure names the function to change.
type upsertLane struct {
	kind   string // "for", "unscoped", "raw"
	reason string // unscoped: "const:Name" | "literal:<text>" | "dynamic:<src>"; raw: source text
	origin string // "path (function)" where the lane was chosen
}

func (l upsertLane) class() laneClass {
	switch l.kind {
	case "for":
		return laneFor
	case "unscoped":
		if l.reason == "const:HealOrphanReason" {
			return laneHeal
		}
		return laneUnscopedOther
	default:
		return laneRaw
	}
}

func (l upsertLane) String() string {
	switch l.kind {
	case "for":
		return "For(tenant) chosen in " + l.origin
	case "unscoped":
		return "Unscoped(" + l.reason + ") chosen in " + l.origin
	default:
		return "raw executor `" + l.reason + "` in " + l.origin
	}
}

type upsertSite struct {
	path       string
	function   string
	line       int
	table      string
	lanes      []upsertLane
	unresolved []string
}

func (s upsertSite) label() string {
	return s.path + ":" + strconv.Itoa(s.line) + " " + s.function + " -> " + s.table
}

// rlsGovernedTables reads, from the migrated schema, exactly the selection
// PostgreSQL migration 0003 makes when it enables row-level security: every
// table with a tenant_id column except the registry itself.
func rlsGovernedTables(t *testing.T) map[string]bool {
	t.Helper()
	database := dbtest.Open(t)
	query := `SELECT m.name FROM sqlite_master m JOIN pragma_table_info(m.name) p
		WHERE m.type='table' AND p.name='tenant_id' AND m.name <> 'tenant'`
	if dbtest.Backend() == appdb.BackendPostgres {
		query = `SELECT c.table_name FROM information_schema.columns c
			JOIN pg_catalog.pg_tables t ON t.schemaname=c.table_schema AND t.tablename=c.table_name
			WHERE c.table_schema=current_schema() AND c.column_name='tenant_id' AND c.table_name <> 'tenant'`
	}
	rows, err := database.QueryContext(t.Context(), query)
	if err != nil {
		t.Fatalf("list tenant-bound tables: %v", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}
		out[strings.ToLower(name)] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table names: %v", err)
	}
	return out
}

var (
	upsertPattern       = regexp.MustCompile(`(?is)\binsert\s+into\s+([A-Za-z_][A-Za-z0-9_]*)\b.*\bon\s+conflict\b.*\bdo\s+update\b`)
	upsertLiteralMarker = regexp.MustCompile(`(?is)\bon\s+conflict\b.*\bdo\s+update\b`)
)

// statementMethods maps the database/sql statement methods to the index of
// their SQL argument.
var statementMethods = map[string]int{
	"Exec": 0, "Query": 0, "QueryRow": 0,
	"ExecContext": 1, "QueryContext": 1, "QueryRowContext": 1,
}

// collectUpsertSites walks every non-test package under internal/ and cmd/,
// finds each ON CONFLICT ... DO UPDATE statement whose table is governed, and
// traces its executor to a lane. It also returns every DO UPDATE string literal
// it could NOT attach to a statement call, so those fail instead of hiding.
func collectUpsertSites(t *testing.T, root string, governed map[string]bool) ([]upsertSite, []string) {
	t.Helper()
	var sites []upsertSite
	var stray []string
	for _, dir := range packageDirs(t, root) {
		pkg := indexPackage(t, root, dir)
		if pkg == nil {
			continue
		}
		found, literals := pkg.upsertSites(governed)
		sites = append(sites, found...)
		stray = append(stray, literals...)
	}
	sort.Slice(sites, func(i, j int) bool {
		if sites[i].path != sites[j].path {
			return sites[i].path < sites[j].path
		}
		return sites[i].line < sites[j].line
	})
	sort.Strings(stray)
	return sites, stray
}

func packageDirs(t *testing.T, root string) []string {
	t.Helper()
	var dirs []string
	for _, tree := range []string{"internal", "cmd"} {
		base := filepath.Join(root, tree)
		if _, err := os.Stat(base); err != nil {
			continue
		}
		err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if entry.Name() == "testdata" || entry.Name() == "vendor" {
					return fs.SkipDir
				}
				dirs = append(dirs, path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", base, err)
		}
	}
	sort.Strings(dirs)
	return dirs
}

// packageIndex is one parsed package: enough structure to find a function's
// callers and a name's binding without a type checker.
type packageIndex struct {
	t      *testing.T
	root   string
	fset   *token.FileSet
	files  map[string]*ast.File
	funcs  []*ast.FuncDecl
	byName map[string][]*ast.FuncDecl
	consts map[string]string
	// fields maps a struct type name to its field names and their type labels,
	// so a call through a field (`f.documents.writeTx(...)`) resolves to the
	// method of the field's type rather than to any method of that name.
	fields map[string]map[string]string
	// storeAlias is, per file, the local name internal/store is imported under,
	// so a qualified store.HealOrphanReason in another package resolves to the
	// same constant a bare HealOrphanReason does here.
	storeAlias map[*ast.File]string
}

func indexPackage(t *testing.T, root string, dir string) *packageIndex {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	pkg := &packageIndex{
		t: t, root: root, fset: token.NewFileSet(),
		files: map[string]*ast.File{}, byName: map[string][]*ast.FuncDecl{},
		consts: map[string]string{}, fields: map[string]map[string]string{}, storeAlias: map[*ast.File]string{},
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		parsed, err := parser.ParseFile(pkg.fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		pkg.files[path] = parsed
		collectStringConstants(parsed, pkg.consts)
		for _, spec := range parsed.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil || !strings.HasSuffix(importPath, "/internal/store") {
				continue
			}
			alias := "store"
			if spec.Name != nil {
				alias = spec.Name.Name
			}
			pkg.storeAlias[parsed] = alias
		}
		for _, decl := range parsed.Decls {
			switch node := decl.(type) {
			case *ast.FuncDecl:
				pkg.funcs = append(pkg.funcs, node)
				pkg.byName[node.Name.Name] = append(pkg.byName[node.Name.Name], node)
			case *ast.GenDecl:
				for _, spec := range node.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					structType, ok := typeSpec.Type.(*ast.StructType)
					if !ok || structType.Fields == nil {
						continue
					}
					fields := map[string]string{}
					for _, field := range structType.Fields.List {
						for _, name := range field.Names {
							fields[name.Name] = typeLabel(field.Type)
						}
					}
					pkg.fields[typeSpec.Name.Name] = fields
				}
			}
		}
	}
	if len(pkg.files) == 0 {
		return nil
	}
	return pkg
}

func (p *packageIndex) fileOf(node ast.Node) *ast.File {
	name := p.fset.Position(node.Pos()).Filename
	return p.files[name]
}

func (p *packageIndex) where(node ast.Node) string {
	position := p.fset.Position(node.Pos())
	return filepath.ToSlash(mustRelative(p.t, p.root, position.Filename)) + ":" + strconv.Itoa(position.Line)
}

func (p *packageIndex) upsertSites(governed map[string]bool) ([]upsertSite, []string) {
	var sites []upsertSite
	// Every string literal that carries upsert SQL, keyed by position, so the
	// ones no statement call claims can be reported.
	claimed := map[token.Pos]bool{}
	var candidates []*ast.BasicLit

	for _, fn := range p.funcs {
		ast.Inspect(fn, func(n ast.Node) bool {
			if literal, ok := n.(*ast.BasicLit); ok && literal.Kind == token.STRING {
				if text, err := strconv.Unquote(literal.Value); err == nil && upsertLiteralMarker.MatchString(text) {
					candidates = append(candidates, literal)
				}
				return true
			}
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel == nil {
				return true
			}
			sqlIndex, ok := statementMethods[selector.Sel.Name]
			if !ok || len(call.Args) <= sqlIndex {
				return true
			}
			sqlArg := call.Args[sqlIndex]
			text := p.flattenSQL(sqlArg)
			match := upsertPattern.FindStringSubmatch(text)
			if match == nil {
				return true
			}
			// Claim every literal inside the SQL argument, concatenations included.
			ast.Inspect(sqlArg, func(inner ast.Node) bool {
				if literal, ok := inner.(*ast.BasicLit); ok {
					claimed[literal.Pos()] = true
				}
				return true
			})
			table := strings.ToLower(match[1])
			if !governed[table] {
				return true
			}
			site := upsertSite{
				path:     filepath.ToSlash(mustRelative(p.t, p.root, p.fset.Position(call.Pos()).Filename)),
				function: functionLabel(fn),
				line:     p.fset.Position(call.Pos()).Line,
				table:    table,
			}
			site.lanes, site.unresolved = p.resolveExecutor(fn, selector.X, map[string]bool{})
			site.lanes = dedupeLanes(site.lanes)
			site.unresolved = dedupeStrings(site.unresolved)
			sites = append(sites, site)
			return true
		})
	}

	var stray []string
	for _, literal := range candidates {
		if !claimed[literal.Pos()] {
			stray = append(stray, p.where(literal))
		}
	}
	return sites, stray
}

func dedupeLanes(lanes []upsertLane) []upsertLane {
	seen := map[string]bool{}
	out := lanes[:0]
	for _, lane := range lanes {
		key := lane.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, lane)
	}
	return out
}

func dedupeStrings(items []string) []string {
	seen := map[string]bool{}
	out := items[:0]
	for _, item := range items {
		if seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

// flattenSQL renders a SQL argument as text: literals as themselves, `+`
// concatenations joined, package constants substituted, anything else as a
// placeholder that keeps the surrounding text matchable.
func (p *packageIndex) flattenSQL(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.BasicLit:
		if node.Kind == token.STRING {
			if text, err := strconv.Unquote(node.Value); err == nil {
				return text
			}
		}
		return " ? "
	case *ast.BinaryExpr:
		if node.Op == token.ADD {
			return p.flattenSQL(node.X) + p.flattenSQL(node.Y)
		}
	case *ast.ParenExpr:
		return p.flattenSQL(node.X)
	case *ast.Ident:
		if text, ok := p.consts[node.Name]; ok {
			return text
		}
	}
	return " ? "
}

// resolveExecutor traces the receiver of a statement call back to the lane (or
// lanes) it came from. It follows locals to their assignments and parameters to
// every caller, and reports what it cannot follow instead of guessing.
func (p *packageIndex) resolveExecutor(fn *ast.FuncDecl, expr ast.Expr, visiting map[string]bool) ([]upsertLane, []string) {
	origin := p.where(fn) + " " + functionLabel(fn)
	switch node := expr.(type) {
	case *ast.ParenExpr:
		return p.resolveExecutor(fn, node.X, visiting)
	case *ast.CallExpr:
		selector, ok := node.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel == nil {
			return nil, []string{"call `" + flatten(nodeSource(p.fset, node)) + "` in " + origin}
		}
		switch selector.Sel.Name {
		case "Unscoped":
			if len(node.Args) != 1 {
				return nil, []string{"Unscoped call with " + strconv.Itoa(len(node.Args)) + " arguments in " + origin}
			}
			return []upsertLane{{kind: "unscoped", reason: p.describeLaneReason(fn, node.Args[0]), origin: origin}}, nil
		case "For":
			return []upsertLane{{kind: "for", origin: origin}}, nil
		case "Begin", "BeginTx":
			return p.resolveExecutor(fn, selector.X, visiting)
		}
		return nil, []string{"call `" + flatten(nodeSource(p.fset, node)) + "` in " + origin}
	case *ast.Ident:
		if index, isParam := paramIndex(fn, node.Name); isParam {
			return p.resolveThroughCallers(fn, index, visiting)
		}
		bindings := bindingsOf(fn, node.Name)
		if len(bindings) == 0 {
			return nil, []string{"`" + node.Name + "` is neither a parameter nor assigned in " + origin}
		}
		var lanes []upsertLane
		var unresolved []string
		for _, rhs := range bindings {
			found, missing := p.resolveExecutor(fn, rhs, visiting)
			lanes = append(lanes, found...)
			unresolved = append(unresolved, missing...)
		}
		return lanes, unresolved
	case *ast.SelectorExpr:
		// A field or a package-level value: nothing the seam handed out.
		return []upsertLane{{kind: "raw", reason: flatten(nodeSource(p.fset, node)), origin: origin}}, nil
	}
	return nil, []string{"`" + flatten(nodeSource(p.fset, expr)) + "` in " + origin}
}

// resolveThroughCallers follows a parameter of fn to the argument every caller
// passes for it, and resolves each of those in the caller's own scope.
func (p *packageIndex) resolveThroughCallers(fn *ast.FuncDecl, index int, visiting map[string]bool) ([]upsertLane, []string) {
	key := functionLabel(fn)
	if visiting[key] {
		return nil, nil
	}
	visiting[key] = true
	defer delete(visiting, key)

	callers, ambiguous := p.callersOf(fn)
	if len(callers) == 0 && len(ambiguous) == 0 {
		return nil, []string{"no caller in the package passes a value for parameter " + strconv.Itoa(index) + " of " + key}
	}
	var lanes []upsertLane
	unresolved := ambiguous
	for _, caller := range callers {
		if index >= len(caller.call.Args) {
			unresolved = append(unresolved, "call to "+key+" in "+p.where(caller.call)+" passes too few arguments to trace")
			continue
		}
		found, missing := p.resolveExecutor(caller.fn, caller.call.Args[index], visiting)
		lanes = append(lanes, found...)
		unresolved = append(unresolved, missing...)
	}
	return lanes, unresolved
}

type callSite struct {
	fn   *ast.FuncDecl
	call *ast.CallExpr
}

// callersOf finds the calls to fn within the package. Methods are matched by
// name AND receiver: `s.writeTx(...)` inside a method of the same receiver type
// is that type's writeTx, not one of the six others; a call through some other
// value resolves only if the name is unique in the package, and is otherwise
// reported as ambiguous rather than guessed at.
func (p *packageIndex) callersOf(fn *ast.FuncDecl) ([]callSite, []string) {
	var callers []callSite
	var ambiguous []string
	name := fn.Name.Name
	for _, candidate := range p.funcs {
		ast.Inspect(candidate, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch target := call.Fun.(type) {
			case *ast.Ident:
				if fn.Recv == nil && target.Name == name {
					callers = append(callers, callSite{fn: candidate, call: call})
				}
			case *ast.SelectorExpr:
				if fn.Recv == nil || target.Sel == nil || target.Sel.Name != name {
					return true
				}
				if receiver, ok := target.X.(*ast.Ident); ok && receiver.Name == receiverName(candidate) && receiverName(candidate) != "" {
					if receiverType(candidate) == receiverType(fn) {
						callers = append(callers, callSite{fn: candidate, call: call})
					}
					return true
				}
				if fieldType, ok := p.receiverFieldType(candidate, target.X); ok {
					if fieldType == receiverType(fn) {
						callers = append(callers, callSite{fn: candidate, call: call})
					}
					return true
				}
				methods := 0
				for _, decl := range p.byName[name] {
					if decl.Recv != nil {
						methods++
					}
				}
				if methods == 1 {
					callers = append(callers, callSite{fn: candidate, call: call})
				} else {
					ambiguous = append(ambiguous, "call `"+flatten(nodeSource(p.fset, call.Fun))+"` in "+p.where(call)+
						" could be any of "+strconv.Itoa(methods)+" methods named "+name+"; the analysis will not guess")
				}
			}
			return true
		})
	}
	return callers, ambiguous
}

// receiverFieldType resolves `recv.field` — the caller's receiver followed by
// one struct field — to that field's declared type label.
func (p *packageIndex) receiverFieldType(caller *ast.FuncDecl, expr ast.Expr) (string, bool) {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || selector.Sel == nil {
		return "", false
	}
	base, ok := selector.X.(*ast.Ident)
	if !ok || base.Name != receiverName(caller) || base.Name == "" {
		return "", false
	}
	fields, ok := p.fields[strings.TrimPrefix(receiverType(caller), "*")]
	if !ok {
		return "", false
	}
	fieldType, ok := fields[selector.Sel.Name]
	return fieldType, ok
}

func receiverName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 || len(fn.Recv.List[0].Names) == 0 {
		return ""
	}
	return fn.Recv.List[0].Names[0].Name
}

func receiverType(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	return typeLabel(fn.Recv.List[0].Type)
}

// paramIndex reports the flat position of a named parameter, counting grouped
// names (`a, b *sql.Tx`) individually, which is how call arguments line up.
func paramIndex(fn *ast.FuncDecl, name string) (int, bool) {
	if fn.Type == nil || fn.Type.Params == nil {
		return 0, false
	}
	index := 0
	for _, field := range fn.Type.Params.List {
		if len(field.Names) == 0 {
			index++
			continue
		}
		for _, ident := range field.Names {
			if ident.Name == name {
				return index, true
			}
			index++
		}
	}
	return 0, false
}

// bindingsOf returns every right-hand side assigned to name inside fn: plain
// and multi-value assignments (`tx, err := lane.Begin()` binds tx to the call)
// and var declarations with an initialiser.
func bindingsOf(fn *ast.FuncDecl, name string) []ast.Expr {
	var out []ast.Expr
	if fn.Body == nil {
		return nil
	}
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

// describeLaneReason classifies the argument of an Unscoped call the way the
// inventory golden does, plus the qualified form another package uses to name
// the store's constant.
func (p *packageIndex) describeLaneReason(fn *ast.FuncDecl, arg ast.Expr) string {
	switch node := arg.(type) {
	case *ast.BasicLit:
		if node.Kind == token.STRING {
			if text, err := strconv.Unquote(node.Value); err == nil {
				return "literal:" + flatten(text)
			}
		}
	case *ast.Ident:
		if _, ok := p.consts[node.Name]; ok {
			return "const:" + node.Name
		}
	case *ast.SelectorExpr:
		if pkgIdent, ok := node.X.(*ast.Ident); ok && node.Sel != nil {
			if alias := p.storeAlias[p.fileOf(fn)]; alias != "" && pkgIdent.Name == alias {
				return "const:" + node.Sel.Name
			}
		}
	}
	return "dynamic:" + flatten(nodeSource(p.fset, arg))
}

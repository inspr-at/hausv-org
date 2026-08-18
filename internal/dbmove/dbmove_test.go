package dbmove

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
)

func openMigratedSQLite(t *testing.T) *sql.DB {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "plan.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

// TableOrder is a hand-written list. Two things can go wrong with it: a table
// the schema has that the list lacks (or the reverse), and an order that
// violates a foreign key. Both are read from the real catalog here rather than
// from a second hand-written list.
func TestTableOrderCoversEverySQLiteTableExactlyOnce(t *testing.T) {
	database := openMigratedSQLite(t)
	got, err := sqliteSchema(context.Background(), database)
	if err != nil {
		t.Fatalf("sqlite schema: %v", err)
	}
	want := sortedKeys(got)
	planned := append([]string(nil), TableOrder...)
	sort.Strings(planned)
	if !reflect.DeepEqual(planned, want) {
		gotSet := toSet(want)
		missing, extra := diffKeys(gotSet, toSet(TableOrder))
		t.Fatalf("TableOrder disagrees with the SQLite schema\n  tables missing from TableOrder: %v\n  TableOrder entries with no table: %v", missing, extra)
	}
	seen := map[string]int{}
	for _, name := range TableOrder {
		seen[name]++
		if seen[name] > 1 {
			t.Errorf("TableOrder lists %q twice", name)
		}
	}
}

func toSet(names []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, name := range names {
		out[name] = struct{}{}
	}
	return out
}

func TestTableOrderRespectsForeignKeys(t *testing.T) {
	position := map[string]int{}
	for i, name := range TableOrder {
		position[name] = i
	}
	check := func(t *testing.T, edges [][2]string) {
		t.Helper()
		if len(edges) < 6 {
			t.Fatalf("only %d foreign keys found — the probe is broken, the schema has more", len(edges))
		}
		for _, edge := range edges {
			from, to := edge[0], edge[1]
			if from == to {
				continue
			}
			pf, okf := position[from]
			pt, okt := position[to]
			if !okf || !okt {
				t.Errorf("foreign key %s -> %s names a table outside TableOrder", from, to)
				continue
			}
			if pt >= pf {
				t.Errorf("TableOrder loads %s (position %d) before %s (position %d), which it references", from, pf, to, pt)
			}
		}
	}

	t.Run("sqlite", func(t *testing.T) {
		database := openMigratedSQLite(t)
		var edges [][2]string
		for _, name := range TableOrder {
			rows, err := database.Query(`PRAGMA foreign_key_list(` + quoteIdent(name) + `)`)
			if err != nil {
				t.Fatalf("foreign keys of %s: %v", name, err)
			}
			for rows.Next() {
				var (
					id, seq            int
					refTable, from, to string
					onUpdate, onDelete string
					match              string
				)
				if err := rows.Scan(&id, &seq, &refTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
					rows.Close()
					t.Fatalf("scan foreign key of %s: %v", name, err)
				}
				edges = append(edges, [2]string{name, refTable})
			}
			rows.Close()
		}
		check(t, edges)
	})

	t.Run("postgres", func(t *testing.T) {
		if dbtest.Backend() != db.BackendPostgres {
			t.Skip("PostgreSQL foreign keys need the PostgreSQL suite")
		}
		database, cfg := dbtest.OpenWithConfig(t)
		scoped, err := db.NewScoped(cfg, database)
		if err != nil {
			t.Fatal(err)
		}
		defer scoped.Close()
		rows, err := scoped.Unscoped("test: the mover proof reads foreign keys and rows across every tenant on the target").Query(`
			SELECT c.conrelid::regclass::text, c.confrelid::regclass::text
			  FROM pg_constraint c
			  JOIN pg_namespace n ON n.oid = c.connamespace
			 WHERE c.contype = 'f' AND n.nspname = current_schema()`)
		if err != nil {
			t.Fatalf("postgres foreign keys: %v", err)
		}
		defer rows.Close()
		var edges [][2]string
		for rows.Next() {
			var from, to string
			if err := rows.Scan(&from, &to); err != nil {
				t.Fatal(err)
			}
			// regclass renders schema-qualified names when the schema is not
			// first on the search_path; strip to the bare table name.
			edges = append(edges, [2]string{bareName(from), bareName(to)})
		}
		check(t, edges)
	})
}

func bareName(qualified string) string {
	if i := strings.LastIndex(qualified, "."); i >= 0 {
		qualified = qualified[i+1:]
	}
	return strings.Trim(qualified, `"`)
}

// The converters are the whole type story. Each must accept exactly what the
// SQLite driver produces for its column class and refuse the rest loudly — a
// converter that guessed would move a wrong value silently.
func TestConvertersAcceptTheirInputsAndRefuseTheRest(t *testing.T) {
	cases := []struct {
		name    string
		convert converter
		in      any
		want    any
		wantErr bool
	}{
		{"text nil", toText, nil, nil, false},
		{"text string", toText, "München", "München", false},
		{"text bytes", toText, []byte("x"), "x", false},
		{"text int refused", toText, int64(1), nil, true},
		{"text float refused", toText, 1.5, nil, true},
		{"bool nil", toBool, nil, nil, false},
		{"bool 0", toBool, int64(0), false, false},
		{"bool 1", toBool, int64(1), true, false},
		{"bool 2 refused", toBool, int64(2), nil, true},
		{"bool -1 refused", toBool, int64(-1), nil, true},
		{"bool string refused", toBool, "true", nil, true},
		{"bool bool", toBool, true, true, false},
		{"int nil", toInt, nil, nil, false},
		{"int int", toInt, int64(-42), int64(-42), false},
		{"int float refused", toInt, 1.0, nil, true},
		{"int string refused", toInt, "1", nil, true},
		{"float nil", toFloat, nil, nil, false},
		{"float float", toFloat, 8.25, 8.25, false},
		{"float int widened", toFloat, int64(8), 8.0, false},
		{"float string refused", toFloat, "8.25", nil, true},
		{"bytes nil", toBytes, nil, nil, false},
		{"bytes bytes", toBytes, []byte{1, 2}, []byte{1, 2}, false},
		{"bytes string", toBytes, "ab", []byte("ab"), false},
		{"bytes int refused", toBytes, int64(1), nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.convert(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected refusal, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
	// Every PostgreSQL type the schema uses has a converter, and the report's
	// notion of "same representation" agrees with the converter table.
	for pgType := range convertersByPostgresType {
		if sameRepresentation("blob", pgType) {
			t.Errorf("blob must never be reported as the same representation as %s", pgType)
		}
	}
	if !sameRepresentation("text", "text") || sameRepresentation("integer", "boolean") || sameRepresentation("blob", "bytea") {
		t.Fatal("sameRepresentation disagrees with the converters")
	}
}

// The canonical form must not depend on the driver: the SQLite integer 1 in a
// boolean column and the PostgreSQL boolean true converge, while genuinely
// different values never collide across kinds.
func TestCanonicalFormIsKindTaggedAndLengthSafe(t *testing.T) {
	one, _ := toBool(int64(1))
	a, _ := canonical(one)
	b, _ := canonical(true)
	if a != b {
		t.Fatalf("converted 1 (%s) and true (%s) must hash alike", a, b)
	}
	distinct := []any{nil, "", "1", int64(1), true, 1.0, []byte("1"), []byte{}}
	seen := map[string]any{}
	for _, v := range distinct {
		c, err := canonical(v)
		if err != nil {
			t.Fatal(err)
		}
		if prev, dup := seen[c]; dup {
			t.Fatalf("%#v and %#v share the canonical form %q", prev, v, c)
		}
		seen[c] = v
	}
	// A pair of strings that concatenate identically must not collide: the
	// length prefix is what prevents "ab"+"c" == "a"+"bc" across two columns.
	if x, _ := canonical("ab"); x == "Sab" {
		t.Fatal("string canonical form has no length prefix")
	}
	if _, err := canonical(struct{}{}); err == nil {
		t.Fatal("an unknown kind must be refused, not stringified")
	}
}

func TestCheckSourceMigrationsRefusesBehindAndAhead(t *testing.T) {
	ctx := context.Background()
	database := openMigratedSQLite(t)
	if err := CheckSourceMigrations(ctx, database); err != nil {
		t.Fatalf("a freshly migrated file must pass: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO schema_migrations(version) VALUES('9999_from_the_future.sql')`); err != nil {
		t.Fatal(err)
	}
	err := CheckSourceMigrations(ctx, database)
	if !errors.Is(err, ErrSourceMigrations) || !strings.Contains(err.Error(), "9999_from_the_future.sql") {
		t.Fatalf("ahead must be refused by name, got %v", err)
	}
	if _, err := database.Exec(`DELETE FROM schema_migrations WHERE version='9999_from_the_future.sql'`); err != nil {
		t.Fatal(err)
	}
	last := db.SQLiteMigrationNames()[len(db.SQLiteMigrationNames())-1]
	if _, err := database.Exec(`DELETE FROM schema_migrations WHERE version=$1`, last); err != nil {
		t.Fatal(err)
	}
	err = CheckSourceMigrations(ctx, database)
	if !errors.Is(err, ErrSourceMigrations) || !strings.Contains(err.Error(), last) {
		t.Fatalf("behind must be refused by name, got %v", err)
	}
}

func TestInsertStatementNumbersParametersAcrossRows(t *testing.T) {
	tbl := table{name: "contacts", columns: []column{{name: "a"}, {name: "b"}}}
	got := insertStatement(tbl, 3)
	want := `INSERT INTO "contacts" ("a", "b") VALUES ($1, $2), ($3, $4), ($5, $6)`
	if got != want {
		t.Fatalf("insert statement\n got %s\nwant %s", got, want)
	}
}

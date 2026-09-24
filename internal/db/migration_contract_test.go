package db

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestMigrationNumericPrefixesAreUnique(t *testing.T) {
	for _, tc := range []struct {
		name    string
		files   []string
		allowed map[string][]string
	}{
		{"sqlite", SQLiteMigrationNames(), map[string][]string{
			"0039": {"0039_annual_statement_consumption_evidence.sql", "0039_annual_statement_period_structure.sql"},
		}},
		{"postgres", PostgresMigrationNames(), map[string][]string{
			"0012": {"0012_annual_statement_consumption_evidence.sql", "0012_annual_statement_period_structure.sql"},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := checkMigrationPrefixes(tc.files, tc.allowed); err != nil {
				t.Fatal(err)
			}
			// These exact historical filenames remain identities. Removing or
			// renaming either would make the runner replay a published migration.
			for _, names := range tc.allowed {
				for _, name := range names {
					found := false
					for _, file := range tc.files {
						found = found || file == name
					}
					if !found {
						t.Errorf("historical migration %s was removed or renamed", name)
					}
				}
			}
		})
	}
}

func checkMigrationPrefixes(names []string, allowed map[string][]string) error {
	pattern := regexp.MustCompile(`^[0-9]{4}_[a-z0-9_]+\.sql$`)
	byPrefix := map[string][]string{}
	for _, name := range names {
		if !pattern.MatchString(name) {
			return fmt.Errorf("invalid numbered migration filename %q", name)
		}
		prefix, _, _ := strings.Cut(name, "_")
		byPrefix[prefix] = append(byPrefix[prefix], name)
	}
	for prefix, files := range byPrefix {
		if len(files) < 2 {
			continue
		}
		sort.Strings(files)
		if !reflect.DeepEqual(files, allowed[prefix]) {
			return fmt.Errorf("duplicate migration prefix %s: %s (only the exact historical pair is allowed)", prefix, strings.Join(files, ", "))
		}
	}
	return nil
}

func TestMigrationPrefixGuardRejectsNewCollisions(t *testing.T) {
	allowed := map[string][]string{"0012": {"0012_a.sql", "0012_b.sql"}}
	for _, files := range [][]string{
		{"0013_a.sql", "0013_b.sql"},
		{"0012_a.sql", "0012_b.sql", "0012_c.sql"},
		{"0012_a.sql", "0012_renamed.sql"},
	} {
		if err := checkMigrationPrefixes(files, allowed); err == nil {
			t.Errorf("guard allowed a new collision: %v", files)
		}
	}
}

func TestMigrationPairedSequenceStaysInStep(t *testing.T) {
	// PostgreSQL 0001-0006 consolidate SQLite 0001-0033 plus PG-only RLS
	// and indexes. Equal file counts or numeric prefixes would be false rules.
	// From annual_statement_periods onward (SQLite 0034 / PG 0007), both
	// directories use the same ordered suffixes. Compare those, preserving the
	// historical duplicate numbers. SchemaAgreement checks the resulting schema.
	suffixes := func(files []string, first string) []string {
		var names []string
		for _, file := range files {
			if file < first {
				continue
			}
			_, suffix, _ := strings.Cut(file, "_")
			names = append(names, suffix)
		}
		return names
	}
	lite := suffixes(SQLiteMigrationNames(), "0034_")
	pg := suffixes(PostgresMigrationNames(), "0007_")
	if len(lite) == 0 || !reflect.DeepEqual(lite, pg) {
		t.Fatalf("paired migration sequence differs:\nSQLite: %v\nPostgreSQL: %v", lite, pg)
	}
}

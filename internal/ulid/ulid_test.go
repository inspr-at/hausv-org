package ulid

import (
	"regexp"
	"testing"
	"time"
)

// The same expression the PostgreSQL target schema enforces. If Valid and this
// ever disagree, rows are rejected at insert time with no useful message, so the
// constraint is asserted here rather than trusted.
var schemaCheck = regexp.MustCompile(`^[0-7][0-9A-HJKMNP-TV-Z]{25}$`)

func TestNewSatisfiesTheSchemaConstraint(t *testing.T) {
	for i := 0; i < 500; i++ {
		id, err := New()
		if err != nil {
			t.Fatalf("mint: %v", err)
		}
		if !schemaCheck.MatchString(id) {
			t.Fatalf("id %q does not satisfy the schema CHECK", id)
		}
		if !Valid(id) {
			t.Fatalf("Valid rejected its own output %q", id)
		}
	}
}

func TestIDsAreUniqueAndOrderedByTime(t *testing.T) {
	seen := make(map[string]bool, 2000)
	for i := 0; i < 2000; i++ {
		id, err := New()
		if err != nil {
			t.Fatal(err)
		}
		if seen[id] {
			t.Fatalf("duplicate id %q after %d mints", id, i)
		}
		seen[id] = true
	}

	early, err := newAt(time.UnixMilli(1_600_000_000_000))
	if err != nil {
		t.Fatal(err)
	}
	late, err := newAt(time.UnixMilli(1_700_000_000_000))
	if err != nil {
		t.Fatal(err)
	}
	if !(early < late) {
		t.Fatalf("lexical order must follow creation order: %q !< %q", early, late)
	}
}

func TestValidRejectsMalformed(t *testing.T) {
	id, err := New()
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"empty":             "",
		"too short":         id[:25],
		"too long":          id + "0",
		"lowercase":         "0123456789abcdefghjkmnpqrs",
		"excluded letter I": "0123456789ABCDEFGHIJKMNPQR",
		"excluded letter L": "0123456789ABCDEFGHJKLMNPQR",
		"excluded letter O": "0123456789ABCDEFGHJKMNOPQR",
		"excluded letter U": "0123456789ABCDEFGHJKMNPQRU",
		"overflows 128 bit": "80000000000000000000000000",
		"hyphen":            "0123456789ABCDEFGHJKMNPQ-S",
	}
	for name, value := range cases {
		if Valid(value) {
			t.Errorf("%s: Valid accepted %q", name, value)
		}
	}
}

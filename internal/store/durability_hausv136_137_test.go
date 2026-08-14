package store

import (
	"os"
	"path/filepath"
	"testing"
)

// HAUSV-136: a torn trailing line (what a crash mid-append leaves) must not
// prevent the store — and therefore the whole app — from loading.
func TestAuditStoreToleratesTornTrailingLine(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "audit.jsonl")
	good := `{"at":"2026-07-13T10:00:00Z","tenant":"demo","actor_email":"a@b.c","action":"login","summary":"ok"}` + "\n"
	torn := `{"at":"2026-07-13T10:01:00Z","tenant":"demo","act` // no newline, truncated
	if err := os.WriteFile(p, []byte(good+torn), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := NewAuditStore(p)
	if err != nil {
		t.Fatalf("boot must survive a torn trailing line, got: %v", err)
	}
	got := s.List(AuditFilter{TenantSlug: "demo", Limit: 10})
	if len(got) != 1 {
		t.Fatalf("want 1 recovered entry, got %d", len(got))
	}
}

// A bad line in the MIDDLE is real corruption, not a torn append — still fatal.
func TestAuditStoreRejectsCorruptMiddleLine(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "audit.jsonl")
	body := `{"at":"2026-07-13T10:00:00Z","tenant":"demo","actor_email":"a@b.c","action":"login","summary":"ok"}` + "\n" +
		`GARBAGE` + "\n" +
		`{"at":"2026-07-13T10:02:00Z","tenant":"demo","actor_email":"a@b.c","action":"login","summary":"ok2"}` + "\n"
	os.WriteFile(p, []byte(body), 0o600)
	if _, err := NewAuditStore(p); err == nil {
		t.Fatal("a corrupt middle line should still fail loudly")
	}
}

// HAUSV-137: SaveJSONAtomic must land the real content (fsync path is exercised;
// we can at least assert the write is correct and complete).
func TestSaveJSONAtomicWritesCompleteFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.json")
	if err := SaveJSONAtomic(p, map[string]int{"a": 1, "b": 2}, "test"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 || b[len(b)-1] != '}' {
		t.Fatalf("incomplete write: %q", b)
	}
	if _, err := os.Stat(p + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("temp file should be gone after rename")
	}
}

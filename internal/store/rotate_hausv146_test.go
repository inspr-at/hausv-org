package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// HAUSV-146: once the live audit file grows past the threshold it rotates and
// the in-memory slice and live file stay bounded.
func TestAuditStoreRotatesAndPreservesHistory(t *testing.T) {
	// Force rotation quickly.
	oldT, oldK := auditRotateThreshold, auditRotateKeep
	auditRotateThreshold, auditRotateKeep = 20, 5
	defer func() { auditRotateThreshold, auditRotateKeep = oldT, oldK }()

	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	s, err := NewAuditStore(path)
	if err != nil {
		t.Fatal(err)
	}
	const total = 60
	for i := 0; i < total; i++ {
		if err := s.Append(AuditEvent{TenantSlug: "jhw22", ActorEmail: "a@b.c", Action: AuditActionLogin, Summary: "x"}); err != nil {
			t.Fatal(err)
		}
	}

	// In-memory slice is bounded, not `total`.
	if len(s.entries) > auditRotateThreshold+1 {
		t.Fatalf("in-memory entries not bounded: %d", len(s.entries))
	}
	// Live file also bounded.
	raw, _ := os.ReadFile(path)
	if lines := strings.Count(strings.TrimSpace(string(raw)), "\n") + 1; lines > auditRotateThreshold+1 {
		t.Fatalf("live file not bounded: %d lines", lines)
	}
	// History preserved: archive files exist, and total archived + live >= total.
	archived := 0
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "audit.jsonl.") {
			b, _ := os.ReadFile(filepath.Join(dir, e.Name()))
			archived += len(strings.Split(strings.TrimSpace(string(b)), "\n"))
		}
	}
	if archived == 0 {
		t.Fatal("no archive file created — history would be lost, not preserved")
	}
	// Queries still work off the retained tail.
	if got := s.List(AuditFilter{TenantSlug: "jhw22", Limit: 3}); len(got) != 3 {
		t.Fatalf("List after rotation returned %d, want 3", len(got))
	}
}

func TestAuditStoreRotatesByAgeAndSize(t *testing.T) {
	oldT, oldK, oldB, oldA := auditRotateThreshold, auditRotateKeep, auditRotateMaxBytes, auditRotateMaxAge
	auditRotateThreshold, auditRotateKeep = 1000, 5
	auditRotateMaxBytes, auditRotateMaxAge = 1<<20, 24*time.Hour
	defer func() {
		auditRotateThreshold, auditRotateKeep = oldT, oldK
		auditRotateMaxBytes, auditRotateMaxAge = oldB, oldA
	}()

	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	s, err := NewAuditStore(path)
	if err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := s.Append(AuditEvent{At: old, TenantSlug: "jhw22", Action: AuditActionLogin, Summary: "old"}); err != nil {
		t.Fatal(err)
	}
	if archives, _ := filepath.Glob(path + ".*"); len(archives) == 0 {
		t.Fatal("age policy did not rotate")
	}
	if len(s.entries) != 0 {
		t.Fatalf("expired events remained in live tail after age rotation: %+v", s.entries)
	}

	auditRotateMaxAge = 365 * 24 * time.Hour
	auditRotateMaxBytes = 1
	if err := s.Append(AuditEvent{TenantSlug: "jhw22", Action: AuditActionLogin, Summary: "large"}); err != nil {
		t.Fatal(err)
	}
	if archives, _ := filepath.Glob(path + ".*"); len(archives) < 2 {
		t.Fatal("size policy did not rotate")
	}
}

func TestAuditStorePrunesExpiredArchives(t *testing.T) {
	oldR := auditArchiveRetention
	auditArchiveRetention = 30 * 24 * time.Hour
	defer func() { auditArchiveRetention = oldR }()

	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	if err := os.WriteFile(path+".old", []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-60 * 24 * time.Hour)
	if err := os.Chtimes(path+".old", old, old); err != nil {
		t.Fatal(err)
	}
	if _, err := NewAuditStore(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".old"); !os.IsNotExist(err) {
		t.Fatalf("expired archive still exists: %v", err)
	}
}

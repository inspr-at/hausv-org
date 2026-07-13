package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// HAUSV-146: once the live audit file grows past the threshold it rotates —
// the in-memory slice and the live file stay bounded, and the full history is
// preserved in an archive (nothing deleted).
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

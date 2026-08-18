package web

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

// The approved energy flow spans the live rail, the recommendation dialog and
// the lower system inventory. This fingerprint records that deliberate new
// baseline so later chart/below changes still cannot slip in unnoticed.

func TestApprovedCompactSidebarFooterStaysStable(t *testing.T) {
	const (
		startMarker = `{{define "sidebar"}}`
		endMarker   = "\n{{define \"releaseHistoryDialog\"}}"
		wantSHA256  = "629aa25d472d8ee910644dcf70f63d905618995fe50496541489bc0b44746333"
	)

	start := strings.Index(PageTemplates, startMarker)
	if start < 0 {
		t.Fatal("shared sidebar template is missing")
	}
	endOffset := strings.Index(PageTemplates[start:], endMarker)
	if endOffset < 0 {
		t.Fatal("release-history boundary after the shared sidebar is missing")
	}

	protected := PageTemplates[start : start+endOffset]
	got := fmt.Sprintf("%x", sha256.Sum256([]byte(protected)))
	if got != wantSHA256 {
		t.Fatalf("approved compact sidebar footer changed: sha256=%s, want %s", got, wantSHA256)
	}
}

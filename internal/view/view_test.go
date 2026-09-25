package view

import (
	"strings"
	"testing"
)

func TestTruncateIssueDescription(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "short text unchanged",
			body: "Das Licht ist defekt.",
			want: "Das Licht ist defekt.",
		},
		{
			name: "exactly 100 runes unchanged",
			body: "0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789",
			want: "0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789",
		},
		{
			name: "truncates at word boundary near 100 chars",
			body: "Die Haustür fällt nicht richtig ins Schloss. Man muss sie kräftig zuziehen, sonst bleibt sie offen stehen.",
			want: "Die Haustür fällt nicht richtig ins Schloss. Man muss sie kräftig zuziehen, sonst bleibt sie offen…",
		},
		{
			name: "collapses multiple spaces",
			body: "Das    Licht    ist    defekt.",
			want: "Das Licht ist defekt.",
		},
		{
			name: "handles newlines",
			body: "Das Licht\nist\ndefekt.",
			want: "Das Licht ist defekt.",
		},
		{
			name: "truncates long text without spaces near boundary",
			body: "Das ist ein sehr langer Text ohne Leerzeichen am Ende: abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
			want: "Das ist ein sehr langer Text ohne Leerzeichen am Ende: abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrs…",
		},
		{
			name: "handles empty string",
			body: "",
			want: "",
		},
		{
			name: "handles whitespace-only string",
			body: "   \n\t   ",
			want: "",
		},
		{
			name: "long text with space at exactly 100",
			body: strings.Repeat("a", 99) + " " + strings.Repeat("b", 20),
			want: strings.Repeat("a", 99) + "…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateIssueDescription(tt.body)
			if got != tt.want {
				t.Errorf("TruncateIssueDescription(%q) = %q, want %q (len=%d)", tt.body, got, tt.want, len([]rune(got)))
			}
			// Ensure result is never longer than ~100 characters + ellipsis
			if len([]rune(got)) > 101 {
				t.Errorf("TruncateIssueDescription(%q) returned %d runes, want <= 101", tt.body, len([]rune(got)))
			}
		})
	}
}

func TestIssueLocationLabelForManagementDropsOwnUnitPrefix(t *testing.T) {
	if got := IssueLocationLabel("own-unit", "Top 7"); got != "Eigene Einheit · Top\u00a07" {
		t.Fatalf("resident label = %q", got)
	}
	if got := IssueLocationLabelForManagement("own-unit", "Top 7"); got != "Top\u00a07" {
		t.Fatalf("management label = %q", got)
	}
	if got := IssueLocationLabelForManagement("own-unit", ""); got != "Einheit" {
		t.Fatalf("management label without detail = %q", got)
	}
	if got := IssueLocationLabelForManagement("common", "Hof"); got != "Gemeinschaft · Hof" {
		t.Fatalf("management common label = %q", got)
	}
}

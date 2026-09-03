package view

import "testing"

func TestHausv615BreakAfterSlashes(t *testing.T) {
	if got, want := BreakAfterSlashes("Winterdienst/Garten/Reinigung"), "Winterdienst/\u200bGarten/\u200bReinigung"; got != want {
		t.Fatalf("BreakAfterSlashes() = %q, want %q", got, want)
	}
}

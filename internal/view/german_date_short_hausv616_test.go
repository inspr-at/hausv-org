package view

import (
	"testing"
	"time"
)

// The case view and the reply placeholder "Frist" printed a literal "Mo,"
// for every date because Go's layout has no German weekday token.
func TestGermanDateShortUsesTheRealWeekday(t *testing.T) {
	t.Parallel()
	cases := map[string]time.Time{
		"Do, 17.09.2026": time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC),
		"Mo, 07.09.2026": time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC),
		"So, 06.09.2026": time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC),
	}
	for want, at := range cases {
		if got := GermanDateShort(at); got != want {
			t.Fatalf("GermanDateShort(%s) = %q, want %q", at.Format(time.RFC3339), got, want)
		}
	}
}

package web

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestValorisationDateLockIsNotAReviewRequest(t *testing.T) {
	run := store.ValorisationRun{ID: "future", Status: "draft", Items: []store.ValorisationItem{{Outcome: "increase", Group: "exception", Exceptions: []string{"letter_too_early"}, WirksamOn: "2026-04-01"}}}
	for _, date := range []string{"2026-03-31", "2026-04-01"} {
		now, _ := time.Parse(time.DateOnly, date)
		view := ValorisationRunView{Run: run, URL: "/app/settings/valorisation", CanApprove: true, OpenExceptions: run.OpenExceptionCount(), ApprovalDate: run.ApprovalNotBefore(now)}
		var out bytes.Buffer
		if err := ValorisationRun(view, false).Render(t.Context(), &out); err != nil {
			t.Fatal(err)
		}
		body := out.String()
		locked := date < "2026-04-01"
		if got := regexp.MustCompile(`<button[^>]*disabled[^>]*>Freigeben und archivieren</button>`).MatchString(body); got != locked {
			t.Fatalf("%s: approval disabled = %v", date, got)
		}
		if strings.Contains(body, "Freigabe ab 01.04.2026 möglich.") != locked || strings.Contains(body, "Bitte zuerst prüfen oder mit Begründung ausschließen.") {
			t.Fatalf("%s: date lock must name its date without suggesting exclusion", date)
		}
		if locked && !strings.Contains(body, `aria-describedby="approval-date-future"`) {
			t.Fatal("date lock reason must be accessible")
		}
	}
}

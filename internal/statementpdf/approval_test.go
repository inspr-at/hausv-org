package statementpdf

import (
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestApprovedStatementHasFinalNotice(t *testing.T) {
	run := fixture()
	run.Approval = &store.AnnualStatementRunApproval{ApprovedAt: time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC), ApprovedBy: "manager@example.com", Role: store.RoleManager}
	docs, err := Documents(run, "a", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	for _, page := range docs[0].Pages() {
		text := strings.Join(page.Footer, " ")
		for _, line := range page.Lines {
			text += " " + line.Text
		}
		if strings.Contains(text, "Entwurf") || !strings.Contains(text, "02.03.2026") || !strings.Contains(text, "Verwaltung") {
			t.Fatal(text)
		}
	}
}

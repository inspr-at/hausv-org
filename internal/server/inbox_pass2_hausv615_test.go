package server

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/demo"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

func TestHausv615InboxLabelsAndProviderFootline(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	item := store.IntakeItem{
		ID: "case-1", TenantSlug: "demo", Source: store.IntakeSourceEmail, Subject: "Vorschreibung",
		Body: "Bitte prüfen.", ReceivedAt: time.Now(), Status: store.IntakeStatusProposed,
		Suggestion: &store.IntakeSuggestion{Category: store.IntakeCategoryOperatingCosts, Priority: store.IssuePriorityNorm, TenantSlug: "demo"},
	}
	got, err := a.inboxCaseView(t.Context(), "musterstadt", item, []web.InboxHouse{{Slug: "demo", Name: "Münzgrabenstraße 9"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.CategoryLabel != "Betriebskosten/\u200bVorschreibung" || got.AssigneeLabel != "Noch niemand" {
		t.Fatalf("case labels = category %q, assignee %q", got.CategoryLabel, got.AssigneeLabel)
	}
	for _, option := range got.Categories {
		if strings.Contains(option.Label, "/") && !strings.Contains(option.Label, "/\u200b") {
			t.Fatalf("category option lacks slash break opportunity: %q", option.Label)
		}
	}
	if got := a.inboxProviderFootline(); got != "Kein KI-Anbieter konfiguriert · jeder Vorschlag und jede Freigabe wird protokolliert" {
		t.Fatalf("provider footline = %q", got)
	}
}

func TestHausv615DemoResetIsNewestPortfolioActivity(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	a.recordAudit(store.AuditEvent{
		At: time.Now().Add(-time.Hour), TenantSlug: "demo", ActorEmail: "vera@example.com",
		Action: store.AuditActionLogin, TargetType: "session", TargetID: "older", Summary: "Anmeldung",
	})
	a.demoReset = func(context.Context, time.Time, io.Writer) (demo.SeedResult, error) {
		return demo.SeedResult{}, nil
	}
	reset := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen/demo", url.Values{})
	if reset.Code != http.StatusOK {
		t.Fatalf("reset status=%d body=%s", reset.Code, reset.Body.String())
	}
	page := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung")
	if page.Code != http.StatusOK {
		t.Fatalf("portfolio status=%d body=%s", page.Code, page.Body.String())
	}
	body := page.Body.String()
	recentStart := strings.Index(body, `id="portfolio-recent-title"`)
	if recentStart < 0 {
		t.Fatalf("portfolio recent section missing: %s", body)
	}
	recent := body[recentStart:]
	resetIndex, loginIndex := strings.Index(recent, "Demodaten initialisiert"), strings.Index(recent, "Anmeldung")
	if resetIndex < 0 || loginIndex < 0 || resetIndex > loginIndex || strings.Contains(recent[:resetIndex], "Noch keine Aktivitäten") {
		t.Fatalf("recent activity order is wrong: %s", recent)
	}
}

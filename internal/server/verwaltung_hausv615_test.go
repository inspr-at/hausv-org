package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestHausv615PortfolioUsesFullNamesGermanAuditLabelsAndAnchoredDueDates(t *testing.T) {
	now := time.Date(2026, time.September, 3, 12, 0, 0, 0, time.Local)
	data := buildPortfolio(now, "Hausverwaltung Musterstadt", "Vera", "need", []portfolioHouseInput{{
		Slug: "muenze", Name: "Münzgrabenstraße 12", Role: roleAdmin,
		Issues: []store.ResidentIssue{{
			Status: store.IssueStatusProgress, CreatedAt: now.Add(-8 * 24 * time.Hour),
			DueAt: now.Add(-24 * time.Hour), AssigneeEmail: "vera.verwalter@example.test",
		}},
		AuditEvents: []store.AuditEvent{{Action: store.AuditActionContextSwitch, ActorEmail: "vera.verwalter@example.test", At: now.Add(-time.Hour)}},
		PeopleNames: map[string]string{"vera.verwalter@example.test": "Vera Verwalter"},
	}})
	if data.OpenIssues != 1 || data.Overdue != 1 || data.Houses[0].Assignee != "Vera Verwalter" {
		t.Fatalf("portfolio aggregate = %+v", data)
	}
	if len(data.AuditEvents) != 1 || data.AuditEvents[0].Action != "Portal gewechselt" || strings.Contains(data.AuditEvents[0].Action, ".") {
		t.Fatalf("portfolio audit = %+v", data.AuditEvents)
	}
}

func TestHausv615EffectiveAIConfigUsesNeutralEnvironmentState(t *testing.T) {
	empty := effectiveAIConfig(func(string) string { return "" }, store.OrgSettings{})
	if empty.Provider != "environment" || empty.Configured || empty.Label != "Umgebung" || empty.Host != "" || empty.Model != "" {
		t.Fatalf("empty config = %#v", empty)
	}
	localEnvironment := map[string]string{"AI_BASE_URL": "http://localhost:11434/v1", "AI_MODEL": "local-model"}
	local := effectiveAIConfig(func(key string) string { return localEnvironment[key] }, store.OrgSettings{})
	if local.Provider != "environment" || !local.Configured {
		t.Fatalf("local environment config = %#v", local)
	}
}

func TestHausv615SettingsCanReturnToEnvironmentAndTextbausteineShowCount(t *testing.T) {
	a, _, settingsRepo := newInboxTestApp(t, roleAdmin)
	settings := store.DefaultOrgSettings("musterstadt")
	settings.AIProvider = "local"
	settings.AIBaseURL = "http://localhost:11434/v1"
	settings.AIModel = "local-model"
	if err := settingsRepo.Save(t.Context(), settings); err != nil {
		t.Fatal(err)
	}
	response := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen", url.Values{
		"threshold": {"90"}, "ai_provider": {"environment"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("settings status=%d body=%s", response.Code, response.Body.String())
	}
	saved, err := settingsRepo.Get(t.Context())
	if err != nil || saved.AIProvider != "" || saved.AIBaseURL != "" || saved.AIModel != "" {
		t.Fatalf("environment settings = %#v err=%v", saved, err)
	}

	repo := a.textbausteine("musterstadt")
	for _, item := range []store.Textbaustein{
		{Key: "eins", Category: store.IntakeCategoryOther, Title: "Eins", Body: "Text", Active: true},
		{Key: "zwei", Category: store.IntakeCategoryOther, Title: "Zwei", Body: "Text", Active: true},
	} {
		if err := repo.Upsert(t.Context(), item); err != nil {
			t.Fatal(err)
		}
	}
	page := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/textbausteine")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "2 Textbausteine") {
		t.Fatalf("textbausteine status=%d body=%s", page.Code, page.Body.String())
	}
}

package server

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/ai"
	"github.com/inspr-at/hausv-org/internal/store"
)

type fakeInboxSuggester struct {
	confidence float64
	err        error
}

func (f fakeInboxSuggester) Label() string { return "Cloud (OpenRouter)" }
func (f fakeInboxSuggester) Suggest(context.Context, ai.TriageInput) (ai.TriageSuggestion, error) {
	return ai.TriageSuggestion{Category: store.IntakeCategoryRepair, Priority: store.IssuePriorityHigh, HouseSlug: "demo", Unit: "Top 1", Assignee: "vera", Reply: "Erledigt.", Confidence: map[string]float64{"overall": f.confidence}, Model: "fake", PromptHash: "1234567890"}, f.err
}

func TestIntakePipelineTrustLevelsAndNilSuggesterFailClosed(t *testing.T) {
	for _, test := range []struct {
		name, level string
		confidence  float64
		enabled     bool
		nilAI       bool
		want        store.IntakeStatus
		wantIssues  int
	}{
		{name: "auto above threshold", level: "auto", confidence: .95, enabled: true, want: store.IntakeStatusAuto, wantIssues: 1},
		{name: "auto below threshold proposes", level: "auto", confidence: .70, enabled: true, want: store.IntakeStatusProposed},
		{name: "manual stays open", level: "manual", confidence: .95, enabled: true, want: store.IntakeStatusOpen},
		{name: "nil never handles", level: "auto", confidence: .95, enabled: true, nilAI: true, want: store.IntakeStatusOpen},
	} {
		t.Run(test.name, func(t *testing.T) {
			a, intake, settingsRepo := newInboxTestApp(t, roleManager)
			if !test.nilAI {
				a.triage = fakeInboxSuggester{confidence: test.confidence}
			}
			settings, _ := settingsRepo.Get(context.Background())
			settings.TrustLevels[store.IntakeCategoryRepair] = test.level
			settings.AutoEnabled = test.enabled
			settings.AutoThreshold = .9
			if err := settingsRepo.Save(context.Background(), settings); err != nil {
				t.Fatal(err)
			}
			now := time.Now()
			item := store.IntakeItem{ID: "pipeline", TenantSlug: "demo", Source: store.IntakeSourcePhone, ExternalRef: "vera@example.com|test", FromName: "Rita", Subject: "Rohr", Body: "Wasser", ReceivedAt: now, Status: store.IntakeStatusOpen}
			if err := intake.Create(context.Background(), item); err != nil {
				t.Fatal(err)
			}
			got, err := a.processIntake(context.Background(), "musterstadt", item, "vera@example.com")
			if test.nilAI && err != nil {
				t.Fatal(err)
			}
			if !test.nilAI && err != nil {
				t.Fatal(err)
			}
			if got.Status != test.want {
				t.Fatalf("status=%s want %s item=%#v", got.Status, test.want, got)
			}
			issues := testRequestRepositories(t, a, "demo").issues.List()
			if len(issues) != test.wantIssues {
				t.Fatalf("issues=%d want=%d", len(issues), test.wantIssues)
			}
			if test.want == store.IntakeStatusAuto {
				if got.Handling == nil || got.Handling.ByName != "System (KI)" {
					t.Fatalf("auto handling=%#v", got.Handling)
				}
				counters, _ := settingsRepo.Get(context.Background())
				if counters.Counters.Auto != 1 {
					t.Fatalf("auto counter=%d", counters.Counters.Auto)
				}
			}
		})
	}
}

func TestIntakePipelineUncertainSuggestionIsHumanReviewedAndOpenBatchIsIdempotent(t *testing.T) {
	a, intake, _ := newInboxTestApp(t, roleManager)
	a.triage = fakeInboxSuggester{confidence: .7, err: ai.ErrUncertain}
	now := time.Now()
	item := store.IntakeItem{ID: "uncertain", TenantSlug: "demo", Source: store.IntakeSourceEmail, FromEmail: "rita@example.com", Subject: "Frage", Body: "Text", ReceivedAt: now, Status: store.IntakeStatusOpen}
	if err := intake.Create(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	got, err := a.processIntake(context.Background(), "musterstadt", item, "vera@example.com")
	if err != nil && !errors.Is(err, ai.ErrUncertain) {
		t.Fatal(err)
	}
	if got.Status != store.IntakeStatusProposed || got.Suggestion == nil {
		t.Fatalf("uncertain=%#v", got)
	}
	n, err := a.processOpenIntake(context.Background(), "musterstadt", 10)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("processed existing suggestion=%d", n)
	}
}

func TestIntakePhoneNoteAutoHandlingAppearsInTodayGroup(t *testing.T) {
	a, intake, settingsRepo := newInboxTestApp(t, roleManager)
	a.triage = fakeInboxSuggester{confidence: .95}
	settings, _ := settingsRepo.Get(context.Background())
	settings.TrustLevels[store.IntakeCategoryRepair] = "auto"
	settings.AutoEnabled = true
	settings.AutoThreshold = .9
	if err := settingsRepo.Save(context.Background(), settings); err != nil {
		t.Fatal(err)
	}
	response := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/telefonnotiz", url.Values{"house": {"demo"}, "unit": {"Top 1"}, "from_name": {"Rita"}, "from_phone": {"+43 1 234"}, "subject": {"Lift steht"}, "body": {"Bitte prüfen"}})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("phone status=%d body=%s", response.Code, response.Body.String())
	}
	items, err := intake.List(context.Background(), store.IntakeFilter{Statuses: []store.IntakeStatus{store.IntakeStatusAuto}})
	if err != nil || len(items) != 1 {
		t.Fatalf("auto phone items=%#v err=%v", items, err)
	}
	page := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Heute automatisch erledigt · 1") || !strings.Contains(page.Body.String(), "Lift steht") {
		t.Fatalf("auto group status=%d body=%s", page.Code, page.Body.String())
	}
}

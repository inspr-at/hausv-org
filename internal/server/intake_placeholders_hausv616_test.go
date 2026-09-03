package server

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/ai"
	"github.com/inspr-at/hausv-org/internal/store"
)

type hausv616Suggester struct{ input ai.TriageInput }

func (s *hausv616Suggester) Label() string { return "test" }
func (s *hausv616Suggester) Suggest(_ context.Context, in ai.TriageInput) (ai.TriageSuggestion, error) {
	s.input = in
	return ai.TriageSuggestion{
		Category: store.IntakeCategoryMasterData, Priority: store.IssuePriorityNorm,
		Reply:      "Ihre Änderung für {{Haus}}, {{Einheit}}, wurde aufgenommen. {{Handwerker}}",
		Confidence: map[string]float64{"overall": .9},
	}, nil
}

func TestHausv616SuggestionFillsAssignedHouseAndMarksMissingValues(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleManager)
	tenant := a.tenants["demo"]
	tenant.Name = "Grazbachgasse 14"
	a.tenants["demo"] = tenant
	suggester := &hausv616Suggester{}
	a.triage = suggester
	item := store.IntakeItem{ID: "in-0002", TenantSlug: "demo", Unit: "Top 7", Source: store.IntakeSourceEmail, FromName: "Rita", Subject: "Name am Klingelschild", Body: "Bitte ändern", ReceivedAt: time.Now()}
	got, err := a.suggestIntake(context.Background(), "musterstadt", item)
	if err != nil {
		t.Fatal(err)
	}
	if suggester.input.AssignedHouseSlug != "demo" || suggester.input.AssignedHouseName != "Grazbachgasse 14" || suggester.input.AssignedUnit != "Top 7" {
		t.Fatalf("assigned input = %#v", suggester.input)
	}
	if !strings.Contains(got.Reply, "Grazbachgasse 14, Top 7") || strings.Contains(got.Reply, "{{") {
		t.Fatalf("reply = %q", got.Reply)
	}
	if len(got.Unfilled) != 1 || got.Unfilled[0] != "Handwerker" || got.Confidence["overall"] != .5 {
		t.Fatalf("suggestion = %#v", got)
	}
}

func TestHausv616TemplateActionUsesHouseName(t *testing.T) {
	a, intake, _ := newInboxTestApp(t, roleManager)
	tenant := a.tenants["demo"]
	tenant.Name = "Grazbachgasse 14"
	a.tenants["demo"] = tenant
	if err := a.textbausteine("musterstadt").Upsert(context.Background(), store.Textbaustein{Key: "stammdaten", Category: store.IntakeCategoryMasterData, Title: "Stammdaten", Body: "Ihre Änderung für {{Haus}}, {{Einheit}}, wurde aufgenommen.", Active: true}); err != nil {
		t.Fatal(err)
	}
	seedInboxItem(t, intake, "template-616", store.IntakeStatusProposed, &store.IntakeSuggestion{Category: store.IntakeCategoryMasterData, Priority: store.IssuePriorityNorm, TenantSlug: "demo", Unit: "Top 7", Confidence: map[string]float64{"overall": .9}})
	response := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang/template-616", url.Values{"action": {"template"}, "template": {"stammdaten"}})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("template status=%d body=%s", response.Code, response.Body.String())
	}
	got, err := intake.Get(context.Background(), "template-616")
	if err != nil {
		t.Fatal(err)
	}
	if got.Suggestion == nil || got.Suggestion.Reply != "Ihre Änderung für Grazbachgasse 14, Top 7 wurde aufgenommen." || len(got.Suggestion.Unfilled) != 0 {
		t.Fatalf("template suggestion = %#v", got.Suggestion)
	}
}

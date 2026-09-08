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

type blockingInboxSuggester struct {
	started chan struct{}
	release chan struct{}
	err     error
}

func (s *blockingInboxSuggester) Label() string { return "Cloud (OpenRouter)" }

func (s *blockingInboxSuggester) Suggest(ctx context.Context, _ ai.TriageInput) (ai.TriageSuggestion, error) {
	select {
	case <-s.started:
	default:
		close(s.started)
	}
	select {
	case <-ctx.Done():
		return ai.TriageSuggestion{}, ctx.Err()
	case <-s.release:
	}
	if s.err != nil {
		return ai.TriageSuggestion{}, s.err
	}
	return ai.TriageSuggestion{Category: store.IntakeCategoryRepair, Priority: store.IssuePriorityHigh, HouseSlug: "demo", Unit: "Top 1", Assignee: "vera", Reply: "Wir kümmern uns.", Confidence: map[string]float64{"overall": .91}, Model: "fake"}, nil
}

func waitInboxSuggestJob(t *testing.T, a *app, id string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		a.suggestJobsMu.Lock()
		job := a.suggestJobs[id]
		done := job != nil && job.done
		a.suggestJobsMu.Unlock()
		if done {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("suggestion job did not finish")
}

func TestInboxSuggestionRunningCancelAndAudit(t *testing.T) {
	a, intake, _ := newInboxTestApp(t, roleManager)
	a.inboxSuggestTimeout = time.Second
	suggester := &blockingInboxSuggester{started: make(chan struct{}), release: make(chan struct{})}
	a.triage = suggester
	seedInboxItem(t, intake, "cancel-me", store.IntakeStatusOpen, nil)

	start := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang/cancel-me", url.Values{"action": {"suggest"}})
	if start.Code != http.StatusSeeOther {
		t.Fatalf("start status=%d body=%s", start.Code, start.Body.String())
	}
	<-suggester.started
	for _, path := range []string{
		"/demo/app/verwaltung/posteingang?status=open",
		"/demo/app/verwaltung/posteingang/cancel-me?status=open",
		"/demo/app/verwaltung/posteingang/cancel-me/vorschlag?status=open",
	} {
		response := authedRequest(t, a, "vera@example.com", path)
		want := `hx-get="/demo/app/verwaltung/posteingang/cancel-me/vorschlag?status=open"`
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), want) {
			t.Fatalf("path-tenant polling URL missing on %s (status %d)", path, response.Code)
		}
		if strings.Contains(response.Body.String(), `hx-get="/app/`) {
			t.Fatalf("unprefixed polling URL on %s", path)
		}
	}
	running := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang/cancel-me/vorschlag")
	for _, want := range []string{"Vorschlag wird erstellt", "Cloud (OpenRouter)", "max. 1 s", "Abbrechen"} {
		if running.Code != http.StatusOK || !strings.Contains(running.Body.String(), want) {
			t.Fatalf("running missing %q status=%d body=%s", want, running.Code, running.Body.String())
		}
	}

	cancelled := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang/cancel-me", url.Values{"action": {"suggest_cancel"}})
	if cancelled.Code != http.StatusSeeOther {
		t.Fatalf("cancel status=%d", cancelled.Code)
	}
	waitInboxSuggestJob(t, a, "cancel-me")
	partial := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang/cancel-me/vorschlag")
	if !strings.Contains(partial.Body.String(), "Abgebrochen") || !strings.Contains(partial.Body.String(), "Vorschlag anfordern") {
		t.Fatalf("cancelled partial=%s", partial.Body.String())
	}
	item, _ := intake.Get(context.Background(), "cancel-me")
	if item.Status != store.IntakeStatusOpen || item.Suggestion != nil {
		t.Fatalf("cancelled item=%#v", item)
	}
	events := a.auditStore.List(store.AuditFilter{TenantSlug: "demo", Action: store.AuditActionIssueAICancel})
	if len(events) != 1 || events[0].TargetID != "cancel-me" {
		t.Fatalf("cancel audit=%#v", events)
	}
}

func TestInboxSuggestionArrivedAndFailedAreSanitised(t *testing.T) {
	t.Setenv("AI_API_KEY", "dummy-secret-never-render")

	t.Run("arrived", func(t *testing.T) {
		a, intake, _ := newInboxTestApp(t, roleManager)
		suggester := &blockingInboxSuggester{started: make(chan struct{}), release: make(chan struct{})}
		a.triage = suggester
		seedInboxItem(t, intake, "arrived", store.IntakeStatusOpen, nil)
		response := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang/arrived", url.Values{"action": {"suggest"}})
		if response.Code != http.StatusSeeOther {
			t.Fatalf("start=%d", response.Code)
		}
		<-suggester.started
		close(suggester.release)
		waitInboxSuggestJob(t, a, "arrived")
		partial := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang/arrived/vorschlag")
		body := partial.Body.String()
		for _, want := range []string{"Vorschlag eingetroffen", "Einordnung", "Übernehmen &amp; weiter", "hx-swap-oob=\"outerHTML\""} {
			if !strings.Contains(body, want) {
				t.Fatalf("arrived missing %q: %s", want, body)
			}
		}
		if strings.Contains(body, "dummy-secret-never-render") {
			t.Fatal("AI_API_KEY leaked into arrived partial")
		}
	})

	t.Run("failed", func(t *testing.T) {
		a, intake, _ := newInboxTestApp(t, roleManager)
		raw := "provider https://secret.invalid?key=dummy-secret-never-render failed"
		suggester := &blockingInboxSuggester{started: make(chan struct{}), release: make(chan struct{}), err: errors.New(raw)}
		a.triage = suggester
		seedInboxItem(t, intake, "failed", store.IntakeStatusOpen, nil)
		_ = authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang/failed", url.Values{"action": {"suggest"}})
		<-suggester.started
		close(suggester.release)
		waitInboxSuggestJob(t, a, "failed")
		partial := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang/failed/vorschlag")
		body := partial.Body.String()
		if !strings.Contains(body, "Vorschlag fehlgeschlagen") || !strings.Contains(body, "Anbieter nicht erreichbar") || !strings.Contains(body, "Erneut versuchen") {
			t.Fatalf("failed partial=%s", body)
		}
		if strings.Contains(body, raw) || strings.Contains(body, "dummy-secret-never-render") {
			t.Fatal("raw provider error or key leaked")
		}
	})
}

func TestInboxSuggestionTimeoutIsEnforcedAndShown(t *testing.T) {
	a, intake, _ := newInboxTestApp(t, roleManager)
	a.inboxSuggestTimeout = 10 * time.Millisecond
	suggester := &blockingInboxSuggester{started: make(chan struct{}), release: make(chan struct{})}
	a.triage = suggester
	seedInboxItem(t, intake, "timeout", store.IntakeStatusOpen, nil)
	response := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang/timeout", url.Values{"action": {"suggest"}})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("start=%d", response.Code)
	}
	<-suggester.started
	waitInboxSuggestJob(t, a, "timeout")
	partial := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang/timeout/vorschlag")
	if !strings.Contains(partial.Body.String(), "Zeitüberschreitung") || strings.Contains(partial.Body.String(), "dummy-secret-never-render") {
		t.Fatalf("timeout partial=%s", partial.Body.String())
	}
}

package server

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestInboxStatusLabels(t *testing.T) {
	for _, test := range []struct {
		name string
		item store.IntakeItem
		want string
	}{
		{name: "unassigned", item: store.IntakeItem{Status: store.IntakeStatusOpen}, want: "Nicht zugeordnet"},
		{name: "open", item: store.IntakeItem{TenantSlug: "demo", Status: store.IntakeStatusOpen}, want: "Offen"},
		{name: "proposed", item: store.IntakeItem{TenantSlug: "demo", Status: store.IntakeStatusProposed}, want: "Vorschlag liegt vor"},
		{name: "approved", item: store.IntakeItem{TenantSlug: "demo", Status: store.IntakeStatusApproved}, want: "Freigegeben"},
		{name: "automatic", item: store.IntakeItem{TenantSlug: "demo", Status: store.IntakeStatusAuto}, want: "Automatisch erledigt"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, _ := inboxStatus(test.item)
			if got != test.want {
				t.Fatalf("status=%q want %q", got, test.want)
			}
		})
	}
}

func TestInboxApproveAndNextFollowsVisibleQueueAcrossThreeItems(t *testing.T) {
	a, intake, _ := newInboxTestApp(t, roleManager)
	now := time.Now().UTC()
	for index, id := range []string{"first", "second", "third"} {
		item := store.IntakeItem{ID: id, TenantSlug: "demo", Unit: "Top 1", Source: store.IntakeSourceEmail, FromName: "Rita", FromEmail: "rita@example.com", Subject: id, Body: "Bitte prüfen.", ReceivedAt: now.Add(-time.Duration(index) * time.Minute), Status: store.IntakeStatusProposed, Suggestion: inboxSuggestion(.93), CreatedAt: now, UpdatedAt: now}
		if err := intake.Create(context.Background(), item); err != nil {
			t.Fatal(err)
		}
	}

	queueQuery := "?sort=received&status=proposed"
	for _, step := range []struct {
		id           string
		wantLocation string
	}{
		{id: "first", wantLocation: "/demo/app/verwaltung/posteingang/second?"},
		{id: "second", wantLocation: "/demo/app/verwaltung/posteingang/third?"},
		{id: "third", wantLocation: "/demo/app/verwaltung/posteingang?"},
	} {
		response := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang/"+step.id, url.Values{"action": {"approve_next"}, "queue_query": {queueQuery}})
		if response.Code != http.StatusSeeOther {
			t.Fatalf("approve_next %s status=%d body=%s", step.id, response.Code, response.Body.String())
		}
		location := response.Header().Get("Location")
		if !strings.HasPrefix(location, step.wantLocation) {
			t.Fatalf("approve_next %s location=%q want prefix %q", step.id, location, step.wantLocation)
		}
		if step.id == "third" && !strings.Contains(location, "Alle+offenen+Anliegen+erledigt") {
			t.Fatalf("last item location=%q", location)
		}
	}

	for _, id := range []string{"first", "second", "third"} {
		item, err := intake.Get(context.Background(), id)
		if err != nil || item.Status != store.IntakeStatusApproved {
			t.Fatalf("item %s=%#v err=%v", id, item, err)
		}
	}
}

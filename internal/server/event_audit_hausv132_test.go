package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestEventMutationsWritePrivacySafeAuditTrail(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "manager@example.com",
		Role:        roleManager,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	start := time.Now().Add(48 * time.Hour).In(time.Local).Format("2006-01-02T15:04")
	privateValues := []string{
		"Interne Eigentümerrunde",
		"Vertraulicher Tagesordnungspunkt",
		"Privatwohnung Top 4",
		"interne-notiz.png",
		"Geänderter vertraulicher Termin",
		"Nicht im Audit protokollieren",
		"Besprechungsraum 2",
	}

	create := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/events", map[string]string{
		"title":     privateValues[0],
		"body":      privateValues[1],
		"location":  privateValues[2],
		"category":  "Eigentümerversammlung",
		"starts_at": start,
	}, "attachments", privateValues[3], minimalPNG())
	if create.Code != http.StatusSeeOther {
		t.Fatalf("create status = %d, want redirect", create.Code)
	}
	items := a.eventStore.ListTenant("demo")
	if len(items) != 1 {
		t.Fatalf("created events = %+v", items)
	}
	eventID := items[0].ID

	edit := authedFormRequest(t, a, "manager@example.com", "/demo/app/events/edit", url.Values{
		"id":        {eventID},
		"title":     {privateValues[4]},
		"body":      {privateValues[5]},
		"location":  {privateValues[6]},
		"category":  {"Wartung"},
		"starts_at": {start},
	})
	if edit.Code != http.StatusSeeOther {
		t.Fatalf("edit status = %d, want redirect", edit.Code)
	}
	deleteResponse := authedFormRequest(t, a, "manager@example.com", "/demo/app/events/delete", url.Values{"id": {eventID}})
	if deleteResponse.Code != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want redirect", deleteResponse.Code)
	}

	events := a.auditStore.List(auditFilter{TenantSlug: "demo", Limit: 10})
	if len(events) != 3 {
		t.Fatalf("event audit count = %d, want 3: %+v", len(events), events)
	}
	byAction := map[string]auditEvent{}
	for _, event := range events {
		byAction[event.Action] = event
		if event.ActorEmail != "manager@example.com" || event.ActorRole != roleManager {
			t.Fatalf("event audit actor = %+v", event)
		}
		if event.TargetType != "event" || event.TargetID != eventID {
			t.Fatalf("event audit target = %+v", event)
		}
		haystack := strings.Join(append([]string{event.Summary}, auditDetailValues(event.Details)...), " ")
		for _, privateValue := range privateValues {
			if strings.Contains(haystack, privateValue) {
				t.Fatalf("event audit leaked %q in %+v", privateValue, event)
			}
		}
	}

	createEvent := byAction[auditActionEventCreate]
	if createEvent.Summary != "Kalendertermin angelegt" || createEvent.Details["category"] != "Eigentümerversammlung" || createEvent.Details["file_count"] != "1" {
		t.Fatalf("create audit = %+v", createEvent)
	}
	updateEvent := byAction[auditActionEventUpdate]
	if updateEvent.Summary != "Kalendertermin geändert" || updateEvent.Details["category"] != "Wartung" {
		t.Fatalf("update audit = %+v", updateEvent)
	}
	deleteEvent := byAction[auditActionEventDelete]
	if deleteEvent.Summary != "Kalendertermin gelöscht" || len(deleteEvent.Details) != 0 {
		t.Fatalf("delete audit = %+v", deleteEvent)
	}

	tones := map[string]string{
		auditActionEventCreate: "add",
		auditActionEventUpdate: "change",
		auditActionEventDelete: "danger",
	}
	for action, want := range tones {
		if got := auditEventViewFrom(byAction[action]).ActionTone; got != want {
			t.Fatalf("%s tone = %q, want %q", action, got, want)
		}
	}

	page := authedRequest(t, a, "manager@example.com", "/demo/app/audit")
	if page.Code != http.StatusOK {
		t.Fatalf("audit page status = %d", page.Code)
	}
	body := page.Body.String()
	for _, want := range []string{"Termin angelegt", "Termin geändert", "Termin gelöscht", "Kalendertermin angelegt", "Kalendertermin geändert", "Kalendertermin gelöscht"} {
		if !strings.Contains(body, want) {
			t.Fatalf("audit page missing %q:\n%s", want, body)
		}
	}
	for _, privateValue := range privateValues {
		if strings.Contains(body, privateValue) {
			t.Fatalf("audit page leaked %q", privateValue)
		}
	}
}

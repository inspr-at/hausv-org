package server

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

// HAUSV-655: a mailbox that cannot be reached must not leak the raw Go error
// into the status line; the line names the situation and the last good poll,
// the technical text sits in a disclosure for administrators.
func TestMailIntakeFailureShowsACustomerMessageAndKeepsTheRawErrorInTheDisclosure(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	const raw = "mail intake: connect hausv-demo-mailbox: dial tcp: lookup hausv-demo-mailbox: no such host"
	now := time.Date(2026, 9, 7, 21, 40, 0, 0, time.Local)
	a.mailIntake.set("musterstadt", func(s *mailIntakeStatus) {
		s.Configured, s.Mailbox, s.Interval = true, "eingang@musterstadt.example", 5*time.Minute
		s.LastRun, s.LastSuccess, s.LastError = now, now.Add(-2*time.Hour), raw
	})
	response := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	html := response.Body.String()
	for _, want := range []string{
		`data-ok="false"`,
		"Postfach derzeit nicht erreichbar · letzter erfolgreicher Abruf 07.09.2026 19:40",
		`<details class="settings-technical"><summary>Technische Details</summary>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("settings page misses %q", want)
		}
	}
	if strings.Contains(html, "Letzter Abruf fehlgeschlagen") {
		t.Errorf("the old raw status line is still rendered")
	}
	if strings.Count(html, raw) != 1 {
		t.Fatalf("raw error rendered %d times, want exactly once (inside the disclosure)", strings.Count(html, raw))
	}
	if strings.Index(html, raw) < strings.Index(html, `<details class="settings-technical">`) {
		t.Errorf("raw error appears before the disclosure")
	}
	status := html[strings.Index(html, `data-ok="false"`):]
	status = status[:strings.Index(status, "</p>")]
	if strings.Contains(status, "dial tcp") || strings.Contains(status, "mail intake") {
		t.Errorf("status line carries technical text: %s", status)
	}
}

func TestMailIntakeFailureWithoutAnySuccessSaysSo(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	a.mailIntake.set("musterstadt", func(s *mailIntakeStatus) {
		s.Configured, s.Mailbox, s.Interval = true, "eingang@musterstadt.example", 5*time.Minute
		s.LastRun, s.LastError = time.Now(), "mail intake: login failed"
	})
	html := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen").Body.String()
	if !strings.Contains(html, "Postfach derzeit nicht erreichbar · noch kein erfolgreicher Abruf seit dem Start") {
		t.Fatalf("missing the no-success wording")
	}
}

func TestMailIntakeHealthyStatusIsUnchanged(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	a.mailIntake.set("musterstadt", func(s *mailIntakeStatus) {
		s.Configured, s.Mailbox, s.Interval = true, "eingang@musterstadt.example", 5*time.Minute
		s.LastRun, s.LastSuccess, s.Total = time.Now(), time.Now(), 3
	})
	html := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen").Body.String()
	if !strings.Contains(html, `data-ok="true"`) || !strings.Contains(html, "Zuletzt abgerufen") || strings.Contains(html, `<details class="settings-technical">`) {
		t.Fatalf("healthy status changed: ok=%v abgerufen=%v technical=%v", strings.Contains(html, `data-ok="true"`), strings.Contains(html, "Zuletzt abgerufen"), strings.Contains(html, `<details class="settings-technical">`))
	}
}

// A poll that ends without error records the moment as the last success, so
// a later failure can still name it.
func TestSuccessfulPollRecordsTheLastSuccess(t *testing.T) {
	a, _ := mailIntakeTestApp(t)
	config := startTestMailbox(t, rawTestMail("first@example.com", "Unbekannt <niemand@example.com>", "Frage", "Text"))
	before := time.Now()
	a.pollMailIntake(context.Background(), "musterstadt", config)
	status, ok := a.mailIntake.get("musterstadt")
	if !ok || status.LastError != "" || status.LastSuccess.Before(before) || status.LastRun.IsZero() {
		t.Fatalf("status after a good poll = %+v (ok=%v)", status, ok)
	}
}

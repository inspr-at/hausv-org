package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/demo"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestVerwaltungDemoResetIsHiddenWithoutDemoReset(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	response := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen/demo")
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.Code)
	}
	post := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen/demo", url.Values{})
	if post.Code != http.StatusNotFound {
		t.Fatalf("post status = %d, want 404", post.Code)
	}
	settings := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen")
	for _, leaked := range []string{"Demodaten initialisieren", `id="demo-init-dialog"`, "data-demo-init-open", "/assets/demo-init.js"} {
		if strings.Contains(settings.Body.String(), leaked) {
			t.Fatalf("settings exposed %q without configured reset closure", leaked)
		}
	}
}

func TestVerwaltungDemoResetPageIsOneConfirmationWithoutJavaScript(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	calls := 0
	a.demoReset = func(context.Context, time.Time, io.Writer) (demo.SeedResult, error) {
		calls++
		return demo.SeedResult{}, nil
	}
	response := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen/demo")
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{"Demodaten initialisieren", `action="/demo/app/verwaltung/einstellungen/demo"`, "Ja, initialisieren", "Abbrechen"} {
		if !strings.Contains(body, want) {
			t.Fatalf("confirmation page missing %q: %s", want, body)
		}
	}
	for _, gone := range []string{"confirm_word", `name="step"`, "ZURÜCKSETZEN", "Demo zurücksetzen"} {
		if strings.Contains(body, gone) {
			t.Fatalf("confirmation page still carries the two-step flow (%q)", gone)
		}
	}
	if got := strings.Count(body, `action="/demo/app/verwaltung/einstellungen/demo"`); got != 1 {
		t.Fatalf("confirmation page must post to the demo route exactly once, got %d", got)
	}
	if calls != 0 {
		t.Fatalf("GET must not reseed, reset calls = %d", calls)
	}
}

func TestVerwaltungDemoResetRunsOnceAndRendersCounters(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	calls := 0
	a.demoReset = func(_ context.Context, anchor time.Time, out io.Writer) (demo.SeedResult, error) {
		calls++
		if anchor.IsZero() || out == nil {
			t.Fatalf("reset received anchor=%v out=%v", anchor, out)
		}
		if want := demoResetAnchor(time.Now()); !anchor.Equal(want) {
			t.Fatalf("anchor = %v, want today %v", anchor, want)
		}
		return demo.SeedResult{
			Statuses:   map[store.IntakeStatus]int{store.IntakeStatusOpen: 3, store.IntakeStatusApproved: 2},
			Categories: map[string]int{store.IntakeCategoryRepair: 4},
		}, nil
	}
	// The dialog posts nothing but the request itself: no step, no typed word.
	response := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen/demo", url.Values{})
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if calls != 1 {
		t.Fatalf("reset calls = %d, want 1", calls)
	}
	today := demoResetAnchor(time.Now()).Format("02.01.2006")
	for _, want := range []string{"Demodaten wurden initialisiert", "Ankerdatum: " + today, "Reparatur/Mangel", ">4<", ">3<", ">2<", "Zum Posteingang"} {
		if !strings.Contains(response.Body.String(), want) {
			t.Fatalf("result missing %q: %s", want, response.Body.String())
		}
	}
	events := a.auditStore.List(store.AuditFilter{TenantSlug: "demo", Action: store.AuditActionDemoReset})
	if len(events) != 1 || events[0].Summary != "Demodaten initialisiert" {
		t.Fatalf("demo reset audit events = %#v", events)
	}
}

func TestVerwaltungDemoResetRequiresOrganisationAdmin(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleManager)
	calls := 0
	a.demoReset = func(context.Context, time.Time, io.Writer) (demo.SeedResult, error) {
		calls++
		return demo.SeedResult{}, nil
	}
	response := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen/demo")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
	post := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen/demo", url.Values{})
	if post.Code != http.StatusForbidden {
		t.Fatalf("post status = %d, want 403", post.Code)
	}
	if calls != 0 {
		t.Fatalf("reset calls = %d, want 0", calls)
	}
	if events := a.auditStore.List(store.AuditFilter{TenantSlug: "demo", Action: store.AuditActionDemoReset}); len(events) != 0 {
		t.Fatalf("audit recorded a refused reset: %#v", events)
	}
}

func TestVerwaltungDemoResetRefusesOtherMethodsAndForeignOrigins(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	calls := 0
	a.demoReset = func(context.Context, time.Time, io.Writer) (demo.SeedResult, error) {
		calls++
		return demo.SeedResult{}, nil
	}
	token, _, err := a.sessions.Put("vera@example.com", "demo", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{http.MethodPut, http.MethodDelete, http.MethodPatch} {
		request := httptest.NewRequest(method, "http://hausv.org/demo/app/verwaltung/einstellungen/demo", nil)
		request.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
		response := httptest.NewRecorder()
		a.handler().ServeHTTP(response, request)
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s status = %d, want 405", method, response.Code)
		}
	}
	// A single POST is the whole confirmation now, so the same-origin check is
	// what keeps a foreign page from reseeding the demo.
	foreign := authedFormRequestWithOrigin(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen/demo", url.Values{}, "https://evil.example")
	if foreign.Code != http.StatusForbidden {
		t.Fatalf("foreign origin status = %d, want 403", foreign.Code)
	}
	if calls != 0 {
		t.Fatalf("reset calls = %d, want 0", calls)
	}
}

func TestVerwaltungSettingsOffersDemoInitDialogAndScript(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	a.demoReset = func(context.Context, time.Time, io.Writer) (demo.SeedResult, error) { return demo.SeedResult{}, nil }
	response := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen")
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{
		"<h2>Demodaten initialisieren</h2>",
		`href="/demo/app/verwaltung/einstellungen/demo" data-demo-init-open`,
		`<dialog class="demo-init-dialog" id="demo-init-dialog"`,
		`action="/demo/app/verwaltung/einstellungen/demo"`,
		"Ja, initialisieren", "data-close-dialog>Abbrechen",
		`/demo/assets/demo-init.js?v=`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("settings missing %q: %s", want, body)
		}
	}
	for _, gone := range []string{"Demo zurücksetzen", "ZURÜCKSETZEN", "<script>"} {
		if strings.Contains(body, gone) {
			t.Fatalf("settings still render %q", gone)
		}
	}
	if strings.Index(body, `/demo/assets/demo-init.js?v=`) < strings.Index(body, `/demo/assets/app.js?v=`) {
		t.Fatal("demo-init.js must load after app.js, which owns dialog closing and focus return")
	}
	asset := httptest.NewRecorder()
	a.handler().ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/assets/demo-init.js?v=test", nil))
	if asset.Code != http.StatusOK || !strings.Contains(asset.Body.String(), "data-demo-init-open") {
		t.Fatalf("demo-init.js not served: status=%d", asset.Code)
	}
}

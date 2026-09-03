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
)

func TestVerwaltungDemoResetIsHiddenWithoutDemoReset(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	response := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen/demo")
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.Code)
	}
	settings := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen")
	if strings.Contains(settings.Body.String(), "Demo zurücksetzen") {
		t.Fatalf("settings exposed demo reset without configured reset closure")
	}
}

func TestVerwaltungDemoResetRejectsWrongConfirmation(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	calls := 0
	a.demoReset = func(context.Context, time.Time, io.Writer) (demo.SeedResult, error) {
		calls++
		return demo.SeedResult{}, nil
	}
	response := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen/demo", url.Values{
		"step": {"1"}, "confirm_word": {"FALSCH"}, "understood": {"1"},
	})
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "ZURÜCKSETZEN") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if calls != 0 {
		t.Fatalf("reset calls = %d, want 0", calls)
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
		return demo.SeedResult{
			Statuses:   map[store.IntakeStatus]int{store.IntakeStatusOpen: 3, store.IntakeStatusApproved: 2},
			Categories: map[string]int{store.IntakeCategoryRepair: 4},
		}, nil
	}
	response := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen/demo", url.Values{"step": {"2"}})
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if calls != 1 {
		t.Fatalf("reset calls = %d, want 1", calls)
	}
	for _, want := range []string{"Demo wurde zurückgesetzt", "Reparatur/Mangel", ">4<", ">3<", ">2<"} {
		if !strings.Contains(response.Body.String(), want) {
			t.Fatalf("result missing %q: %s", want, response.Body.String())
		}
	}
	events := a.auditStore.List(store.AuditFilter{TenantSlug: "demo", Action: store.AuditActionDemoReset})
	if len(events) != 1 {
		t.Fatalf("demo reset audit events = %#v", events)
	}
}

func TestVerwaltungDemoResetRequiresOrganisationAdmin(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleManager)
	a.demoReset = func(context.Context, time.Time, io.Writer) (demo.SeedResult, error) { return demo.SeedResult{}, nil }
	response := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen/demo")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}

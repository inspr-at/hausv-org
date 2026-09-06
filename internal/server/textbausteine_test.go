package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestTextbausteinListRendersStarterCatalogue(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	repo := a.textbausteine("musterstadt")
	if err := repo.Upsert(t.Context(), store.Textbaustein{
		Key: "reparatur-beauftragt", Category: store.IntakeCategoryRepair,
		Title: "Reparatur – Fachbetrieb beauftragt", Body: "Sehr geehrte{{Anrede}} {{Name}}", Active: true,
	}); err != nil {
		t.Fatal(err)
	}

	page := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/textbausteine")
	body := page.Body.String()
	if page.Code != http.StatusOK || !strings.Contains(body, "Reparatur – Fachbetrieb beauftragt") || !strings.Contains(body, "Reparatur/Mangel") || !strings.Contains(body, "Aktiv") {
		t.Fatalf("list status=%d body=%s", page.Code, body)
	}
}

func TestTextbausteinSaveAuditsAndDeactivateHidesFromActiveList(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	repo := a.textbausteine("musterstadt")
	if err := repo.Upsert(t.Context(), store.Textbaustein{Key: "antwort", Category: store.IntakeCategoryOther, Title: "Antwort", Body: "Hallo {{Name}}", Active: true}); err != nil {
		t.Fatal(err)
	}

	saved := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/textbausteine/antwort", url.Values{
		"title": {"Persönliche Antwort"}, "category": {store.IntakeCategoryOther}, "body": {"Grüß Gott {{Name}}"}, "active": {"1"},
	})
	if saved.Code != http.StatusSeeOther {
		t.Fatalf("save status=%d body=%s", saved.Code, saved.Body.String())
	}
	item, err := repo.Get(t.Context(), "antwort")
	if err != nil || item.Title != "Persönliche Antwort" || !item.Active {
		t.Fatalf("saved item=%#v err=%v", item, err)
	}
	events := a.auditStore.List(store.AuditFilter{TenantSlug: "demo", Action: store.AuditActionTextbausteinChanged})
	if len(events) != 1 || events[0].TargetID != "antwort" {
		t.Fatalf("save audit=%#v", events)
	}

	deactivated := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/textbausteine/antwort/deaktivieren", url.Values{})
	if deactivated.Code != http.StatusSeeOther {
		t.Fatalf("deactivate status=%d body=%s", deactivated.Code, deactivated.Body.String())
	}
	active, err := repo.ListActive(t.Context())
	if err != nil || len(active) != 0 {
		t.Fatalf("active after deactivate=%#v err=%v", active, err)
	}
	events = a.auditStore.List(store.AuditFilter{TenantSlug: "demo", Action: store.AuditActionTextbausteinChanged})
	if len(events) != 2 || events[0].TargetID != "antwort" || events[0].Summary != "Textbaustein deaktiviert" {
		t.Fatalf("deactivate audit=%#v", events)
	}
}

func TestTextbausteinCreateDerivesUniqueKebabKey(t *testing.T) {
	a, _, _ := newInboxTestApp(t, roleAdmin)
	repo := a.textbausteine("musterstadt")
	if err := repo.Upsert(t.Context(), store.Textbaustein{Key: "neue-oel-pruefung", Category: store.IntakeCategoryOther, Title: "Schon da", Body: "Text", Active: true}); err != nil {
		t.Fatal(err)
	}
	created := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/textbausteine/neu", url.Values{
		"title": {"Neue Öl-Prüfung"}, "category": {store.IntakeCategoryOther}, "body": {"Bitte {{Nummer}} prüfen."}, "active": {"1"},
	})
	if created.Code != http.StatusSeeOther {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	item, err := repo.Get(t.Context(), "neue-oel-pruefung-2")
	if err != nil || item.Title != "Neue Öl-Prüfung" {
		t.Fatalf("created item=%#v err=%v", item, err)
	}
}

func TestTextbausteinPageRequiresOrganisationAdminAndNavIsRoleScoped(t *testing.T) {
	admin, _, _ := newInboxTestApp(t, roleAdmin)
	adminPage := authedRequest(t, admin, "vera@example.com", "/demo/app/verwaltung/textbausteine")
	adminBody := adminPage.Body.String()
	for _, want := range []string{`href="/demo/app/verwaltung/textbausteine"`, `href="/demo/app/verwaltung/rechte"`} {
		if adminPage.Code != http.StatusOK || !strings.Contains(adminBody, want) {
			t.Fatalf("admin navigation missing %q: status=%d body=%s", want, adminPage.Code, adminBody)
		}
	}

	manager, _, _ := newInboxTestApp(t, roleManager)
	if denied := authedRequest(t, manager, "vera@example.com", "/demo/app/verwaltung/textbausteine"); denied.Code != http.StatusForbidden {
		t.Fatalf("manager textbausteine status=%d, want 403", denied.Code)
	}
	owner, _, _ := newInboxTestApp(t, roleOwner)
	ownerPage := authedRequest(t, owner, "vera@example.com", "/demo/app")
	if strings.Contains(ownerPage.Body.String(), "/app/verwaltung/textbausteine") || strings.Contains(ownerPage.Body.String(), "/app/verwaltung/rechte") {
		t.Fatalf("Eigentümer navigation exposed Verwaltung admin links: %s", ownerPage.Body.String())
	}
}

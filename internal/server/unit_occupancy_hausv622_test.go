package server

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

func TestHausv622UnitOccupancyResolvesProfilesAndFallbacks(t *testing.T) {
	a := newHausv622App(t)
	occupancy := a.unitOccupancy("demo", "Top 3")
	if !occupancy.Found || len(occupancy.Renters) != 1 || occupancy.Renters[0].Name != "Matthias Dorn" {
		t.Fatalf("Top 3 occupancy = %+v", occupancy)
	}
	if got := occupancyLabel(occupancy); got != "Mieter Matthias Dorn" {
		t.Fatalf("occupancy label = %q", got)
	}
	owner := a.unitOccupancy("demo", "Top 1")
	if len(owner.Owners) != 1 || owner.Owners[0].Name != "Alina Auer" {
		t.Fatalf("Top 1 owner = %+v", owner)
	}
	fallback := a.occupancyPeople("demo", []string{"max.mustermann@example.com"})
	if len(fallback) != 1 || fallback[0].Name != "Max Mustermann" {
		t.Fatalf("fallback person = %+v", fallback)
	}
	if got := occupancyLabel(a.unitOccupancy("demo", "Top 99")); got != "nicht zugeordnet" {
		t.Fatalf("unknown occupancy label = %q", got)
	}
}

func TestHausv622InboxCaseUsesSuggestedUnitOccupancy(t *testing.T) {
	a := newHausv622App(t)
	item := store.IntakeItem{
		ID: "top-3-case", TenantSlug: "demo", Unit: "Top 1", Source: store.IntakeSourceEmail,
		Subject: "Heizung", Body: "Bitte prüfen.", ReceivedAt: time.Now(), Status: store.IntakeStatusProposed,
		Suggestion: &store.IntakeSuggestion{TenantSlug: "demo", Unit: "Top 3", Category: store.IntakeCategoryRepair, Priority: store.IssuePriorityHigh, Confidence: map[string]float64{"overall": .9}},
	}
	view, err := a.inboxCaseView(context.Background(), "musterstadt", item, []web.InboxHouse{{Slug: "demo", Name: "Janusbergweg 123"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if view.Unit != "Top 3" || view.Occupancy != "Mieter Matthias Dorn" {
		t.Fatalf("case view = %+v", view)
	}
	var rendered bytes.Buffer
	if err := web.InboxCaseWorkflowContent(view).Render(context.Background(), &rendered); err != nil {
		t.Fatal(err)
	}
	if body := rendered.String(); !strings.Contains(body, "Top 3") || !strings.Contains(body, "Mieter Matthias Dorn") {
		t.Fatalf("case occupancy was not rendered: %s", body)
	}
}

func TestHausv622BuildingAndContactsShowOccupancyByRole(t *testing.T) {
	a := newHausv622App(t)
	building := authedRequest(t, a, "vera.verwalter@musterstadt.example", "/demo/app/settings/building?section=units")
	if building.Code != 200 {
		t.Fatalf("building status = %d", building.Code)
	}
	buildingBody := building.Body.String()
	for _, want := range []string{"Alina Auer", "Matthias Dorn", "Stellplatz 2", "frei"} {
		if !strings.Contains(buildingBody, want) {
			t.Fatalf("building page missing %q", want)
		}
	}
	if strings.Index(buildingBody, "Top 3") > strings.Index(buildingBody, "Stellplatz 2") {
		t.Fatal("parking must follow residential units")
	}

	for _, email := range []string{"vera.verwalter@musterstadt.example", "hedwig.beirat@musterstadt.example"} {
		page := authedRequest(t, a, email, "/demo/app/kontakte")
		if page.Code != 200 || !strings.Contains(page.Body.String(), "Bewohner je Einheit") || !strings.Contains(page.Body.String(), "Stellplatz 2") {
			t.Fatalf("contacts occupancy for %s = %d %s", email, page.Code, page.Body.String())
		}
	}
	page := authedRequest(t, a, "matthias.mieter@musterstadt.example", "/demo/app/kontakte")
	if page.Code != 200 || strings.Contains(page.Body.String(), "Bewohner je Einheit") {
		t.Fatalf("renter contacts leaked occupancy: %d %s", page.Code, page.Body.String())
	}
}

func newHausv622App(t *testing.T) *app {
	t.Helper()
	a := newTestPortalApp(t, userProfile{
		Email: "vera.verwalter@musterstadt.example", FirstName: "Vera", LastName: "Verwalter",
		Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})
	tenant := a.tenants["demo"]
	tenant.Name = "Janusbergweg 123"
	a.tenants["demo"] = tenant
	a.profiles["alina.eigentuemer@musterstadt.example"] = userProfile{Email: "alina.eigentuemer@musterstadt.example", FirstName: "Alina", LastName: "Auer", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["matthias.mieter@musterstadt.example"] = userProfile{Email: "matthias.mieter@musterstadt.example", FirstName: "Matthias", LastName: "Dorn", Role: roleRenter, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["hedwig.beirat@musterstadt.example"] = userProfile{Email: "hedwig.beirat@musterstadt.example", FirstName: "Hedwig", LastName: "Eder", Role: roleBeirat, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{
		{ID: "top-1", Label: "Top 1", UnitType: unitTypeResidential, OwnerEmails: []string{"alina.eigentuemer@musterstadt.example"}},
		{ID: "top-3", Label: "Top 3", UnitType: unitTypeResidential, RenterEmails: []string{"matthias.mieter@musterstadt.example"}},
		{ID: "stellplatz-1", Label: "Stellplatz 1", UnitType: unitTypeParking, OwnerEmails: []string{"alina.eigentuemer@musterstadt.example"}},
		{ID: "stellplatz-2", Label: "Stellplatz 2", UnitType: unitTypeParking, RenterEmails: []string{"matthias.mieter@musterstadt.example"}},
	}); err != nil {
		t.Fatalf("seed units: %v", err)
	}
	return a
}

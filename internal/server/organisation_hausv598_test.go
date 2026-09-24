package server

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/config"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
)

// organisationTestApp is a Verwaltung with two houses and a bound organisation
// store, i.e. the shape HAUSV-598 introduces.
func organisationTestApp(t *testing.T, email string) *app {
	t.Helper()
	a := newTestPortalApp(t, userProfile{
		Email: email, Role: roleAdmin, Tenants: []string{"demo", "zweitehaus"}, AuthMethods: defaultAuthMethods(),
	})
	second := a.tenants["demo"]
	second.Slug = "zweitehaus"
	second.Name = "Zweites Haus"
	second.Organisation = "musterstadt"
	addTestTenant(a, second)
	first := a.tenants["demo"]
	first.Organisation = "musterstadt"
	a.tenants["demo"] = first
	a.organisations = map[string]config.OrganisationConfig{
		"musterstadt": {Key: "musterstadt", Name: "Hausverwaltung Musterstadt GmbH"},
	}
	database := dbtest.Open(t)
	a.organisationRepo = func(orgKey string) store.OrganisationRepository {
		return store.BindOrganisationRepository(database, orgKey)
	}
	return a
}

func TestOrganisationSyncSeedsTheConfiguredVerwaltungOnce(t *testing.T) {
	a := organisationTestApp(t, "verwaltung@example.com")
	ctx := context.Background()
	a.syncOrganisations(ctx)

	stored, ok, err := a.organisationRepo("musterstadt").Get(ctx)
	if err != nil || !ok {
		t.Fatalf("organisation was not seeded: ok=%v err=%v", ok, err)
	}
	if stored.Name != "Hausverwaltung Musterstadt GmbH" {
		t.Fatalf("seeded name = %q", stored.Name)
	}
	if len(stored.Houses) != 2 {
		t.Fatalf("seeded houses = %v, want both configured houses", stored.Houses)
	}

	// A second boot must not overwrite what someone edited in the app.
	edited := stored
	edited.Name = "Hausverwaltung Musterstadt & Partner"
	edited.ContactName = "Vera Verwalter"
	if err := a.organisationRepo("musterstadt").Save(ctx, edited); err != nil {
		t.Fatal(err)
	}
	a.syncOrganisations(ctx)
	again, _, err := a.organisationRepo("musterstadt").Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if again.Name != edited.Name || again.ContactName != "Vera Verwalter" {
		t.Fatalf("boot overwrote edited organisation data: %+v", again)
	}
}

func TestOrganisationSyncReconcilesHousesFromConfiguration(t *testing.T) {
	a := organisationTestApp(t, "verwaltung@example.com")
	ctx := context.Background()
	a.syncOrganisations(ctx)

	// A house leaves the Verwaltung in configuration; the next boot follows.
	tenant := a.tenants["zweitehaus"]
	tenant.Organisation = ""
	a.tenants["zweitehaus"] = tenant
	a.syncOrganisations(ctx)

	stored, _, err := a.organisationRepo("musterstadt").Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Houses) != 1 || stored.Houses[0] != "demo" {
		t.Fatalf("houses after reconcile = %v, want only demo", stored.Houses)
	}
}

func TestPortfolioReadsTheHouseListFromTheOrganisation(t *testing.T) {
	const email = "verwaltung@example.com"
	a := organisationTestApp(t, email)
	a.syncOrganisations(context.Background())

	page := authedRequest(t, a, email, "/demo/app/verwaltung")
	if page.Code != http.StatusOK {
		t.Fatalf("portfolio status = %d", page.Code)
	}
	body := page.Body.String()
	if !strings.Contains(body, "Hausverwaltung Musterstadt GmbH") {
		t.Fatal("portfolio does not name the stored organisation")
	}
	if !strings.Contains(body, "Zweites Haus") {
		t.Fatal("portfolio is missing the organisation's second house")
	}

	// A house the person does not administer stays out, even when the
	// organisation lists it: membership decides, the organisation only orders.
	if err := a.organisationRepo("musterstadt").SetHouses(context.Background(),
		[]string{"demo", "zweitehaus", "fremdes-haus"}); err != nil {
		t.Fatal(err)
	}
	again := authedRequest(t, a, email, "/demo/app/verwaltung").Body.String()
	if strings.Contains(again, "fremdes-haus") {
		t.Fatal("a house outside the person's membership leaked into the portfolio")
	}
}

func TestOrganisationNameFollowsTheStoreNotTheConfiguration(t *testing.T) {
	const email = "verwaltung@example.com"
	a := organisationTestApp(t, email)
	ctx := context.Background()
	a.syncOrganisations(ctx)
	if err := a.organisationRepo("musterstadt").Save(ctx, store.Organisation{
		Key: "musterstadt", Name: "Hausverwaltung Musterstadt & Partner", Houses: []string{"demo", "zweitehaus"},
	}); err != nil {
		t.Fatal(err)
	}

	page := authedRequest(t, a, email, "/demo/app/verwaltung")
	body := page.Body.String()
	if !strings.Contains(body, "Hausverwaltung Musterstadt &amp; Partner") {
		t.Fatal("the stored organisation name is not what the page shows")
	}
	if strings.Contains(body, "Musterstadt GmbH") {
		t.Fatal("the configured name still wins over the stored one")
	}
}

func TestSingleHouseWithoutOrganisationIsUnchanged(t *testing.T) {
	const email = "verwalter@example.com"
	a := newTestPortalApp(t, userProfile{Email: email, Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	database := dbtest.Open(t)
	a.organisationRepo = func(orgKey string) store.OrganisationRepository {
		return store.BindOrganisationRepository(database, orgKey)
	}
	a.syncOrganisations(context.Background())

	page := authedRequest(t, a, email, "/demo/app")
	if page.Code != http.StatusOK {
		t.Fatalf("house portal status = %d", page.Code)
	}
	if record, ok := a.organisationRecordFor(context.Background(), &authCtx{
		email: email, role: roleManager, tenant: a.tenants["demo"], tenantRef: testTenantRef("demo"),
	}); ok {
		t.Fatalf("a house without organisation resolved one: %+v", record)
	}
}

func TestOrganisationContactIsEditableBySettings(t *testing.T) {
	const email = "verwaltung@example.com"
	a := organisationTestApp(t, email)
	ctx := context.Background()
	a.syncOrganisations(ctx)
	database := dbtest.Open(t)
	a.orgSettings = func(orgKey string) store.OrgSettingsRepository {
		return store.BindOrgSettingsRepository(database, orgKey)
	}

	form := url.Values{
		"threshold":         {"90"},
		"organisation_name": {"Hausverwaltung Musterstadt GmbH"},
		"contact_name":      {"Vera Verwalter"},
		"contact_address":   {" Musterstraße 12, 8010 Graz "},
		"contact_email":     {"buero@musterstadt.example"},
		"contact_phone":     {"+43 316 123456"},
	}
	for _, category := range store.IntakeCategories() {
		form.Set("trust_"+category.Key, "propose")
	}
	saved := authedFormRequest(t, a, email, "/demo/app/verwaltung/einstellungen", form)
	if saved.Code != http.StatusSeeOther {
		t.Fatalf("settings save = %d, want redirect:\n%s", saved.Code, saved.Body.String())
	}

	stored, ok, err := a.organisationRepo("musterstadt").Get(ctx)
	if err != nil || !ok {
		t.Fatalf("organisation missing after save: ok=%v err=%v", ok, err)
	}
	if stored.ContactName != "Vera Verwalter" || stored.ContactEmail != "buero@musterstadt.example" || stored.ContactPhone != "+43 316 123456" {
		t.Fatalf("contact not stored: %+v", stored)
	}
	if len(stored.Houses) != 2 {
		t.Fatalf("saving the contact changed the house set: %v", stored.Houses)
	}
	settings, err := a.orgSettings("musterstadt").Get(ctx)
	if err != nil || settings.ContactAddress != "Musterstraße 12, 8010 Graz" {
		t.Fatal("address not persisted in organisation JSON", err)
	}
	foreign, err := a.orgSettings("other").Get(ctx)
	if err != nil || foreign.ContactAddress != "" {
		t.Fatal("address escaped organisation scope", err)
	}

	page := authedRequest(t, a, email, "/demo/app/verwaltung/einstellungen")
	if !strings.Contains(page.Body.String(), "Vera Verwalter") || !strings.Contains(page.Body.String(), `value="Musterstraße 12, 8010 Graz"`) {
		t.Fatal("the settings page does not show the stored contact")
	}
	form.Set("contact_address", strings.Repeat("ä", 501))
	if got := authedFormRequest(t, a, email, "/demo/app/verwaltung/einstellungen", form); got.Code != http.StatusBadRequest {
		t.Fatal("overlong address accepted", got.Code)
	}
	form.Del("contact_address")
	if got := authedFormRequest(t, a, email, "/demo/app/verwaltung/einstellungen", form); got.Code != http.StatusSeeOther {
		t.Fatal("legacy settings save failed", got.Code)
	}
	settings, _ = a.orgSettings("musterstadt").Get(ctx)
	if settings.ContactAddress != "Musterstraße 12, 8010 Graz" {
		t.Fatal("omitted field erased address")
	}
	form.Set("contact_address", "")
	if got := authedFormRequest(t, a, email, "/demo/app/verwaltung/einstellungen", form); got.Code != http.StatusSeeOther {
		t.Fatal("address clear failed", got.Code)
	}
	settings, _ = a.orgSettings("musterstadt").Get(ctx)
	if settings.ContactAddress != "" {
		t.Fatal("address was not cleared")
	}
}

func TestOrganisationNameCannotBeEmptied(t *testing.T) {
	const email = "verwaltung@example.com"
	a := organisationTestApp(t, email)
	a.syncOrganisations(context.Background())
	database := dbtest.Open(t)
	a.orgSettings = func(orgKey string) store.OrgSettingsRepository {
		return store.BindOrgSettingsRepository(database, orgKey)
	}

	form := url.Values{"threshold": {"90"}, "organisation_name": {"   "}}
	for _, category := range store.IntakeCategories() {
		form.Set("trust_"+category.Key, "propose")
	}
	got := authedFormRequest(t, a, email, "/demo/app/verwaltung/einstellungen", form)
	if got.Code != http.StatusBadRequest {
		t.Fatalf("empty organisation name = %d, want 400", got.Code)
	}
}

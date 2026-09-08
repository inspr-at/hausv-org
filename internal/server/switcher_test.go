package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/config"
	"github.com/inspr-at/hausv-org/internal/web"
)

func switcherGET(t *testing.T, a *app, path string) *httptest.ResponseRecorder {
	t.Helper()
	token, _, err := a.sessions.PutSession("multi@example.com", "demo", authMethodEmail, roleOwner, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo"+path, nil)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	return rr
}

func TestSwitcherSearchTenantRoleBoundariesAndLimit(t *testing.T) {
	a := newPortalContextTestApp(t)
	addTestTenant(a, tenantConfig{Slug: "secret-house", Name: "Fremde Liegenschaft", Address: "Nicht freigegeben"})
	rr := switcherGET(t, a, "/app/liegenschaften/suche?format=json&q=Fremde")
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), `"entries":[]`) {
		t.Fatalf("foreign search: %d %s", rr.Code, rr.Body.String())
	}
	rr = switcherGET(t, a, "/app/liegenschaften/suche?format=json&q=Nebenweg")
	var result struct {
		Entries []web.LiegenschaftEntry `json:"entries"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 1 || result.Entries[0].Tenant != "haus-b" || result.Entries[0].Role != roleRenter {
		t.Fatalf("scoped address search: %+v", result)
	}
	// Kennzeichen is the stable tenant slug, searched independently of the label.
	rr = switcherGET(t, a, "/app/liegenschaften/suche?format=json&q=haus-b")
	if !strings.Contains(rr.Body.String(), `"name":"Haus B"`) {
		t.Fatal("slug search missing")
	}
	rr = switcherGET(t, a, "/app/liegenschaften/suche?format=json&recent=secret-house%7CAdmin&recent=haus-b%7CMieter&recent=haus-b%7CMieter")
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 1 || result.Entries[0].Tenant != "haus-b" || result.Entries[0].Role != roleRenter {
		t.Fatal("recent IDs bypassed tenant/role boundary or were not deduplicated")
	}
	// Each generated tenant has explicit membership and an owner assignment.
	for i := 0; i < 12; i++ {
		slug := fmt.Sprintf("switch-%02d", i)
		profile := a.profiles["multi@example.com"]
		profile.Tenants = append(profile.Tenants, slug)
		profile.TenantMemberships[slug] = tenantMembership{Role: roleOwner}
		a.profiles["multi@example.com"] = profile
		addTestTenant(a, tenantConfig{Slug: slug, Name: "Suchobjekt " + strconv.Itoa(i), Address: "Wien"})
		if err := testUnitRepository(t, a, slug).SetUnits([]unit{{ID: "top", Label: "Top 1", OwnerEmails: []string{"multi@example.com"}}}); err != nil {
			t.Fatal(err)
		}
	}
	rr = switcherGET(t, a, "/app/liegenschaften/suche?format=json&q=Suchobjekt")
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if rr.Code != 200 || len(result.Entries) != 8 {
		t.Fatalf("limit: %d entries %d", rr.Code, len(result.Entries))
	}
	if !strings.Contains(rr.Header().Get("Cache-Control"), "no-store") {
		t.Fatal("private directory must not be cached")
	}
	rr = switcherGET(t, a, "/app/liegenschaften/suche?q=Suchobjekt")
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), "Meine Liegenschaften") || !strings.Contains(rr.Body.String(), `action="/demo/app/context"`) {
		t.Fatalf("no-JS GET: %d", rr.Code)
	}
}

func TestScopeContextSharedAssembly(t *testing.T) {
	a := newPortalContextTestApp(t)
	tenant := a.tenants["demo"]
	tenant.PortalType = config.PortalTypeCommunity
	a.tenants["demo"] = tenant
	tenantB := a.tenants["haus-b"]
	tenantB.PortalType = config.PortalTypeCommunity
	a.tenants["haus-b"] = tenantB
	ac := authCtx{email: "multi@example.com", role: roleOwner, realRole: roleOwner, tenant: tenant}
	scope := a.scopeContext(&ac, "", false)
	if !scope.Ready || len(scope.Segments) != 1 || scope.Segments[0] != houseDisplayName(tenant) || scope.Switcher.Count != 2 {
		t.Fatalf("personal scope: %+v", scope)
	}
	if scope.Switcher.Current == nil || scope.Switcher.Current.Role != roleOwner {
		t.Fatal("active role lost")
	}
	ac.role = roleRenter
	resident := a.scopeContext(&ac, "", false)
	if resident.Switcher.Count != 2 || resident.Switcher.Current == nil || resident.Switcher.Current.Role != roleRenter {
		t.Fatal("resident with two properties lost the active role")
	}
	overview := a.scopeContext(&ac, "Musterverwaltung", true)
	if strings.Join(overview.Segments, " › ") != "Musterverwaltung › Alle Liegenschaften" || overview.Switcher.Current != nil {
		t.Fatalf("overview: %+v", overview)
	}
	if strings.Contains(scope.Switcher.StorageKey, ac.email) {
		t.Fatal("storage namespace must not contain identity")
	}
}

func portfolio15000() []portfolioHouseInput {
	rows := make([]portfolioHouseInput, 15000)
	for i := range rows {
		rows[i] = portfolioHouseInput{Slug: fmt.Sprintf("objekt-%05d", i), Name: fmt.Sprintf("Liegenschaft %05d", i), Address: "Graz", UnitCount: 2}
	}
	return rows
}
func TestPortfolio15000PaginationAndBoundedRender(t *testing.T) {
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	data := buildPortfolio(now, "Verwaltung", "Vera", "name", portfolio15000())
	for _, page := range []int{1, 2, 300} {
		paged := pagePortfolio(data, url.Values{"page": {strconv.Itoa(page)}, "sort": {"name"}})
		if paged.HouseCount != 15000 || paged.UnitCount != 30000 || len(paged.QuietHouses) != 50 || paged.Page != page || paged.Pages != 300 {
			t.Fatalf("page %d: rows=%d page=%d pages=%d", page, len(paged.QuietHouses), paged.Page, paged.Pages)
		}
		if paged.QuietHouses[0].Slug != fmt.Sprintf("objekt-%05d", (page-1)*50) {
			t.Fatal("page boundary incorrect")
		}
		var body bytes.Buffer
		if err := web.PortfolioPage(web.PortalPageData{}, paged).Render(context.Background(), &body); err != nil {
			t.Fatal(err)
		}
		if n := strings.Count(body.String(), `class="portfolio-row-form"`); n != 50 {
			t.Fatalf("rendered %d rows", n)
		}
		if body.Len() > 200000 {
			t.Fatalf("unbounded shell/page: %d bytes", body.Len())
		}
	}
	for _, raw := range []string{"-1", "bad", "99999999999999999999999999"} {
		if pagePortfolio(data, url.Values{"page": {raw}}).Page != 1 {
			t.Fatalf("bad page %s", raw)
		}
	}
	query := pagePortfolio(data, url.Values{"q": {"objekt-14999"}, "sort": {"name"}})
	if query.ResultCount != 1 || query.QuietHouses[0].Slug != "objekt-14999" {
		t.Fatal("search before paging failed")
	}
	if pagePortfolio(data, url.Values{"filter": {"need"}}).ResultCount != 0 {
		t.Fatal("need filter returned quiet rows")
	}
	if pagePortfolio(data, url.Values{"filter": {"quiet"}}).ResultCount != 15000 {
		t.Fatal("quiet filter missing rows")
	}
}
func BenchmarkPortfolio15000Page(b *testing.B) {
	houses := portfolio15000()
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	query := url.Values{"page": {"150"}, "sort": {"name"}}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		data := pagePortfolio(buildPortfolio(now, "Verwaltung", "Vera", "name", houses), query)
		if len(data.QuietHouses) != 50 {
			b.Fatal("unbounded page")
		}
	}
}

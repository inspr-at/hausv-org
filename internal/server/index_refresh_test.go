package server

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
)

type indexFixtureClient struct {
	calls int
	fail  bool
}

func (c *indexFixtureClient) Do(r *http.Request) (*http.Response, error) {
	c.calls++
	status := 200
	body := "C-VPIZR-0;C-VPICOICOP18_5-0;F-VPIMZBM\nVPIZR-202609;VPICOICOP18-0;133,2\n"
	if strings.HasSuffix(r.URL.Path, "_C-VPIZR-0.csv") {
		body = "code;name\nVPIZR-202609;Sep.26 (vorl.)\n"
	}
	if c.fail {
		status = 503
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}, nil
}

func TestIndexRefreshAdminRouteAndOfflineFailure(t *testing.T) {
	a, _, _ := newArchiveDemoApp(t)
	admin := "index-admin@example.com"
	a.profiles[admin] = userProfile{Email: admin, Role: roleAdmin, Tenants: []string{archiveDemoTenant}, AuthMethods: defaultAuthMethods()}
	client := &indexFixtureClient{}
	a.indexRefreshClient = client
	path := "/app/verwaltung/wertsicherung"
	page := archiveDemoRequest(t, a, admin, "GET", path, nil)
	if page.Code != 200 || !strings.Contains(page.Body.String(), "VPI aktualisieren") || (!strings.Contains(page.Body.String(), "Statistik Austria") || !strings.Contains(page.Body.String(), "CC BY 4.0")) {
		t.Fatalf("page %d", page.Code)
	}
	for _, role := range []string{roleManager, roleOwner, roleResident, roleBeirat, roleServiceProvider} {
		email := strings.ToLower(role) + "-index@example.com"
		a.profiles[email] = userProfile{Email: email, Role: role, Tenants: []string{archiveDemoTenant}, AuthMethods: defaultAuthMethods()}
		res := archiveDemoRequest(t, a, email, "POST", path+"/refresh", nil)
		if res.Code != 403 {
			t.Fatalf("role %s: %d", role, res.Code)
		}
	}
	if client.calls != 0 {
		t.Fatal("denied action fetched data")
	}
	res := archiveDemoRequest(t, a, admin, "POST", path+"/refresh", nil)
	if res.Code != 303 || !strings.Contains(res.Header().Get("Location"), "index_refresh=ok") {
		t.Fatalf("refresh %d %s", res.Code, res.Body.String())
	}
	imports, err := store.NewIndexReferenceStore(a.tenantDB).Recent(a.tenantIdentities[archiveDemoTenant].Ref())
	if err != nil || len(imports) != 10 {
		t.Fatalf("imports %d %v", len(imports), err)
	}
	page = archiveDemoRequest(t, a, admin, "GET", path, nil)
	if !strings.Contains(page.Body.String(), "Letzter Import:") || !strings.Contains(page.Body.String(), "2026-09") || !strings.Contains(page.Body.String(), "1 neu") {
		t.Fatal("import result missing")
	}
	client.fail = true
	res = archiveDemoRequest(t, a, admin, "POST", path+"/refresh", nil)
	if res.Code != 303 || !strings.Contains(res.Header().Get("Location"), "index_refresh=failed") {
		t.Fatal("failure not shown")
	}
	remaining, err := store.NewIndexReferenceStore(a.tenantDB).Recent(a.tenantIdentities[archiveDemoTenant].Ref())
	if err != nil || len(remaining) != len(imports) {
		t.Fatal("network failure modified imports", err)
	}
	a.indexRefreshBusy.Store(true)
	res = archiveDemoRequest(t, a, admin, "POST", path+"/refresh", nil)
	if res.Code != 409 {
		t.Fatal("concurrent refresh not rejected")
	}
}

func TestIndexRefreshWorkerDisabledAndCancellable(t *testing.T) {
	a, _, _ := newArchiveDemoApp(t)
	client := &indexFixtureClient{}
	a.indexRefreshClient = client
	stop := a.StartIndexRefreshWorker()
	stop()
	if client.calls != 0 {
		t.Fatal("default worker fetched")
	}
	a.indexRefreshEnabled = true
	stop = a.StartIndexRefreshWorker()
	stop()
	if client.calls != 0 {
		t.Fatal("startup performed synchronous fetch")
	}
}

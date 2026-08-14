package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestHomeSetupSessionIsCryptographicallySeparatedFromPortalSession(t *testing.T) {
	secret := []byte(strings.Repeat("s", 32))
	portalSessions := newSessionStore(secret)
	homeSessions := newSessionStore(homeSetupSecret(secret))
	portalToken, _, err := portalSessions.Put("owner@example.com", "mein-zuhause", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, ok := homeSessions.Get(portalToken); ok {
		t.Fatal("portal session was accepted as a home setup session")
	}
	homeToken, _, err := homeSessions.Put("owner@example.com", "mein-zuhause", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, ok := portalSessions.Get(homeToken); ok {
		t.Fatal("home setup session was accepted as a portal session")
	}
}

func TestHomeStartReservationConfirmationAndConnectorBoundary(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	mailer := &recordingMailer{}
	a.mailer = mailer
	handler := a.handler()

	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "http://hausv.org/start", nil))
	if page.Code != http.StatusOK {
		t.Fatalf("start page status = %d", page.Code)
	}
	for _, want := range []string{"Zuhause zuerst sicher anlegen", "hausv.org/", "Keine Home-Assistant-Zugangsdaten", `action="/start"`} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("start page missing %q", want)
		}
	}
	if page.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("start cache control = %q", page.Header().Get("Cache-Control"))
	}

	values := url.Values{
		"household_name": {"Zuhause am Stadtpark"},
		"slug":           {"stadtpark-7"},
		"email":          {"owner@example.com"},
		"authority":      {"1"},
	}
	created := postHomeStartHAUSV469(t, handler, values)
	if created.Code != http.StatusSeeOther || created.Header().Get("Location") != "/start?sent=1" {
		t.Fatalf("create = %d location %q", created.Code, created.Header().Get("Location"))
	}
	if strings.Contains(created.Body.String(), "owner@example.com") {
		t.Fatal("public create response exposed the owner email")
	}
	links := waitForMagicLinks(t, mailer, 1)
	confirmationURL, err := url.Parse(links[0].Link)
	if err != nil {
		t.Fatal(err)
	}
	secret := confirmationURL.Query().Get("token")
	if secret == "" {
		t.Fatal("confirmation email missing one-time token")
	}

	verify := httptest.NewRecorder()
	handler.ServeHTTP(verify, httptest.NewRequest(http.MethodGet, "http://hausv.org"+confirmationURL.RequestURI(), nil))
	if verify.Code != http.StatusSeeOther || verify.Header().Get("Location") != "/start/connector" {
		t.Fatalf("verify = %d location %q", verify.Code, verify.Header().Get("Location"))
	}
	if strings.Contains(verify.Body.String(), secret) || strings.Contains(verify.Header().Get("Location"), secret) {
		t.Fatal("confirmation token survived the redirect")
	}
	var setupCookie *http.Cookie
	for _, cookie := range verify.Result().Cookies() {
		if cookie.Name == homeSetupCookieName {
			setupCookie = cookie
		}
	}
	if setupCookie == nil || !setupCookie.HttpOnly || setupCookie.Path != "/start" {
		t.Fatalf("setup cookie = %+v", setupCookie)
	}

	connectorRequest := httptest.NewRequest(http.MethodGet, "http://hausv.org/start/connector", nil)
	connectorRequest.AddCookie(setupCookie)
	connector := httptest.NewRecorder()
	handler.ServeHTTP(connector, connectorRequest)
	if connector.Code != http.StatusOK {
		t.Fatalf("connector page status = %d", connector.Code)
	}
	connectorBody := connector.Body.String()
	for _, want := range []string{"Zuhause am Stadtpark ist reserviert", "hausv.org/stadtpark-7", "keinen Token", "Nur-Lese"} {
		if !strings.Contains(connectorBody, want) {
			t.Fatalf("connector page missing %q", want)
		}
	}
	for _, forbidden := range []string{"owner@example.com", secret, `name="token"`, `name="base_url"`, `name="ha_`} {
		if strings.Contains(connectorBody, forbidden) {
			t.Fatalf("connector page exposed %q", forbidden)
		}
	}

	reused := httptest.NewRecorder()
	handler.ServeHTTP(reused, httptest.NewRequest(http.MethodGet, "http://hausv.org"+confirmationURL.RequestURI(), nil))
	if reused.Code != http.StatusSeeOther || reused.Header().Get("Location") != "/start?link=expired" {
		t.Fatalf("reused link = %d location %q", reused.Code, reused.Header().Get("Location"))
	}
}

func TestHomeStartRejectsReservedInvalidAndForeignPathsWithoutEnumeration(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	mailer := &recordingMailer{}
	a.mailer = mailer
	handler := a.handler()

	valid := url.Values{"household_name": {"Erstes Zuhause"}, "slug": {"bereits-reserviert"}, "email": {"first@example.com"}, "authority": {"1"}}
	if response := postHomeStartHAUSV469(t, handler, valid); response.Code != http.StatusSeeOther {
		t.Fatalf("initial reservation status = %d", response.Code)
	}
	waitForMagicLinks(t, mailer, 1)

	cases := []url.Values{
		{"household_name": {"Fremdes Zuhause"}, "slug": {"bereits-reserviert"}, "email": {"other@example.com"}, "authority": {"1"}},
		{"household_name": {"Konfiguriert"}, "slug": {"demo"}, "email": {"other@example.com"}, "authority": {"1"}},
		{"household_name": {"System"}, "slug": {"auth"}, "email": {"other@example.com"}, "authority": {"1"}},
		{"household_name": {"Ungültig"}, "slug": {"../falsch"}, "email": {"other@example.com"}, "authority": {"1"}},
		{"household_name": {"Ohne Nachweis"}, "slug": {"ohne-nachweis"}, "email": {"other@example.com"}},
		{"household_name": {"Zeilen\numbruch"}, "slug": {"mail-injection"}, "email": {"other@example.com"}, "authority": {"1"}},
	}
	for _, values := range cases {
		response := postHomeStartHAUSV469(t, handler, values)
		if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/start?sent=1" {
			t.Fatalf("neutral rejection = %d location %q", response.Code, response.Header().Get("Location"))
		}
	}
	time.Sleep(20 * time.Millisecond)
	if got := len(mailer.recordedMagicLinks()); got != 1 {
		t.Fatalf("rejected reservations sent %d links, want one initial link", got)
	}

	tenantPage := httptest.NewRecorder()
	handler.ServeHTTP(tenantPage, httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/start", nil))
	if tenantPage.Code != http.StatusNotFound {
		t.Fatalf("tenant start page status = %d", tenantPage.Code)
	}
}

func TestHomeStartLinkExpirySessionScopeAndRateLimit(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.mailer = &recordingMailer{}
	handler := a.handler()

	if _, err := a.homeReservations.Reserve(store.HomeReservation{
		Slug: "ablauf", HouseholdName: "Ablauf", OwnerEmail: "owner@example.com", AuthorizationConfirmed: true,
	}, time.Now()); err != nil {
		t.Fatal(err)
	}
	a.homeSetupTokens.Put("expired-home-token", "owner@example.com", "ablauf", -time.Second)
	expired := httptest.NewRecorder()
	handler.ServeHTTP(expired, httptest.NewRequest(http.MethodGet, "http://hausv.org/start/verify?token=expired-home-token", nil))
	if expired.Header().Get("Location") != "/start?link=expired" {
		t.Fatalf("expired location = %q", expired.Header().Get("Location"))
	}

	foreignToken, _, err := a.homeSetupSessions.Put("other@example.com", "ablauf", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	foreignRequest := httptest.NewRequest(http.MethodGet, "http://hausv.org/start/connector", nil)
	foreignRequest.AddCookie(&http.Cookie{Name: homeSetupCookieName, Value: foreignToken})
	foreign := httptest.NewRecorder()
	handler.ServeHTTP(foreign, foreignRequest)
	if foreign.Header().Get("Location") != "/start" {
		t.Fatalf("foreign setup location = %q", foreign.Header().Get("Location"))
	}

	values := url.Values{"household_name": {"Rate Home"}, "slug": {"rate-home"}, "email": {"rate@example.com"}, "authority": {"1"}}
	for i := 0; i < homeStartAccountPolicy.limit; i++ {
		if response := postHomeStartHAUSV469(t, handler, values); response.Code != http.StatusSeeOther {
			t.Fatalf("allowed request %d status = %d", i, response.Code)
		}
	}
	limited := postHomeStartHAUSV469(t, handler, values)
	if limited.Code != http.StatusTooManyRequests || limited.Header().Get("Retry-After") == "" || strings.Contains(limited.Body.String(), "rate@example.com") {
		t.Fatalf("rate limit = %d retry=%q body=%q", limited.Code, limited.Header().Get("Retry-After"), limited.Body.String())
	}
}

func TestHomeStartPrivacyExplainsReservationDataAndRetention(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	response := httptest.NewRecorder()
	a.handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://hausv.org/datenschutz", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("privacy status = %d", response.Code)
	}
	for _, want := range []string{
		"gewünschte Pfad", "Eigentümer-E-Mail", "keine Home-Assistant-Adresse", "nach 24 Stunden zur Löschung fällig", "nächsten stündlichen Bereinigungslauf", "Reservierungslinks: 15 Minuten",
		"abgeleitete Hashes des Einmal-Codes", "keine Entity-IDs, Messwerte oder Gerätezustände", "Ein Widerruf beendet den Zugang sofort",
	} {
		if !strings.Contains(response.Body.String(), want) {
			t.Fatalf("privacy page missing %q", want)
		}
	}
}

func postHomeStartHAUSV469(t *testing.T, handler http.Handler, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "http://hausv.org/start", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "http://hausv.org")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

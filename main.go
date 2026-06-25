package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed assets/*
var assets embed.FS

const (
	roleAdmin         = "Admin"
	roleResident      = "Bewohner"
	permissionParking = "parking"
)

type app struct {
	baseURL               string
	addr                  string
	rootDomain            string
	defaultTenant         string
	tenants               map[string]tenantConfig
	sessionSecure         bool
	allowed               map[string]struct{}
	admins                map[string]struct{}
	profiles              map[string]userProfile
	localDevLogin         bool
	tokens                *tokenStore
	sessions              *sessionStore
	mailer                mailer
	templates             *template.Template
	parkingStore          *parkingStore
	parkingSampleInterval time.Duration
}

type tokenStore struct {
	mu     sync.Mutex
	secret []byte
	items  map[string]loginToken
}

type loginToken struct {
	email      string
	tenantSlug string
	expiresAt  time.Time
	used       bool
}

type sessionStore struct {
	mu    sync.Mutex
	items map[string]session
}

type session struct {
	email      string
	role       string
	tenantSlug string
	expiresAt  time.Time
}

type mailer interface {
	SendMagicLink(to string, link string) error
	Configured() bool
}

type smtpMailer struct {
	host string
	port string
	user string
	pass string
	from string
}

type tenantConfig struct {
	Slug    string              `json:"slug"`
	Name    string              `json:"name"`
	Address string              `json:"address"`
	Host    string              `json:"host"`
	HA      homeAssistantConfig `json:"-"`
}

type homeAssistantConfig struct {
	baseURL           string
	token             string
	meterEnergyEntity string
	powerEntity       string
	priceEntity       string
}

type haState struct {
	EntityID   string         `json:"entity_id"`
	State      string         `json:"state"`
	Attributes map[string]any `json:"attributes"`
}

type haHistoryState struct {
	EntityID    string         `json:"entity_id"`
	State       string         `json:"state"`
	LastChanged time.Time      `json:"last_changed"`
	LastUpdated time.Time      `json:"last_updated"`
	Attributes  map[string]any `json:"attributes"`
}

type parkingTelemetry struct {
	Configured bool
	Connected  bool
	Message    string
	Metrics    []parkingMetric
	Entities   []parkingEntityRef
}

type parkingMetric struct {
	Label  string
	Value  string
	Detail string
}

type parkingEntityRef struct {
	Label    string
	EntityID string
}

type parkingStore struct {
	mu   sync.Mutex
	path string
	data parkingStoreData
}

type parkingStoreData struct {
	Tenants map[string]parkingTenantData `json:"tenants"`
}

type parkingTenantData struct {
	Settings      parkingSettings              `json:"settings"`
	Months        map[string]parkingMonthState `json:"months"`
	EnergySamples []parkingNumericSample       `json:"energy_samples"`
	PriceSamples  []parkingNumericSample       `json:"price_samples"`
	Samples       []parkingStoredSample        `json:"samples,omitempty"`
}

type parkingSettings struct {
	GridFeeEURPerKWh float64 `json:"grid_fee_eur_per_kwh"`
}

type parkingMonthState struct {
	Paid bool `json:"paid"`
}

type parkingStoredSample struct {
	At             time.Time `json:"at"`
	EnergyKWh      float64   `json:"energy_kwh"`
	PriceEURPerKWh float64   `json:"price_eur_per_kwh"`
}

type parkingNumericSample struct {
	At    time.Time `json:"at"`
	Value float64   `json:"value"`
}

type parkingAccountingView struct {
	Message          string
	GridFeeValue     string
	GridFeeLabel     string
	Months           []parkingMonthView
	HasMonths        bool
	LastSampleLabel  string
	HistoryAvailable bool
}

type parkingMonthView struct {
	Month           string
	MonthLabel      string
	PeriodLabel     string
	KWh             string
	EnergyCost      string
	GridCost        string
	TotalCost       string
	AveragePrice    string
	Paid            bool
	PaidLabel       string
	TogglePaidValue string
	ToggleLabel     string
	ChartPercent    int
	Partial         bool
	SampleCount     int
	HourCount       int
}

func main() {
	a, err := newApp()
	if err != nil {
		log.Fatal(err)
	}
	stopSampler := a.startParkingSampler()
	defer stopSampler()

	mux := http.NewServeMux()
	mux.Handle("GET /assets/", http.FileServerFS(assets))
	mux.HandleFunc("GET /healthz", a.health)
	mux.HandleFunc("GET /", a.home)
	mux.HandleFunc("POST /auth/request", a.requestLogin)
	mux.HandleFunc("GET /auth/verify", a.verifyLogin)
	mux.HandleFunc("POST /auth/logout", a.logout)
	mux.HandleFunc("GET /app", a.portal)
	mux.HandleFunc("GET /app/parking", a.parking)
	mux.HandleFunc("POST /app/parking/settings", a.updateParkingSettings)
	mux.HandleFunc("POST /app/parking/month", a.updateParkingMonth)
	mux.HandleFunc("GET /app/settings/users", a.userSettings)
	mux.HandleFunc("GET /{tenant}", a.tenantPathRedirect)
	mux.HandleFunc("GET /{tenant}/{rest...}", a.tenantPathRedirect)

	server := &http.Server{
		Addr:              a.addr,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("weg-portal listening on %s", a.addr)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func newApp() (*app, error) {
	if err := loadLocalEnv(".env.local"); err != nil {
		return nil, err
	}

	baseURL := strings.TrimRight(env("BASE_URL", "http://localhost:8080"), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("BASE_URL must be an absolute URL")
	}

	publicURL := !isLocalHost(parsed.Hostname())
	secret, err := sessionSecret(publicURL)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("pages").Parse(pageTemplates)
	if err != nil {
		return nil, err
	}

	allowed := parseAllowed(env("INVITE_EMAILS", ""))
	admins := parseAllowed(env("ADMIN_EMAILS", ""))
	rootDomain := normalizeHost(env("ROOT_DOMAIN", "hausv.org"))
	defaultTenant := env("DEFAULT_TENANT", "jhw22")
	tenants, err := parseTenants(env("WEG_TENANTS_JSON", ""), rootDomain, defaultTenant, newHomeAssistantConfig())
	if err != nil {
		return nil, err
	}
	profiles, err := parseUserProfiles(env("WEG_USERS_JSON", ""), allowed, admins, defaultTenant)
	if err != nil {
		return nil, err
	}
	localDevLogin := parseBool(env("LOCAL_DEV_LOGIN", "false")) && isLocalHost(parsed.Hostname())

	mailTransport := smtpMailer{
		host: env("SMTP_HOST", ""),
		port: env("SMTP_PORT", "587"),
		user: env("SMTP_USER", ""),
		pass: env("SMTP_PASS", ""),
		from: env("MAIL_FROM", "WEG Portal <noreply@example.invalid>"),
	}
	if err := mailTransport.Validate(); err != nil {
		return nil, err
	}
	if publicURL && !mailTransport.Configured() {
		return nil, fmt.Errorf("SMTP_HOST and MAIL_FROM are required when BASE_URL is public")
	}

	parkingDataPath := env("PARKING_DATA_PATH", "tmp/parking.json")
	parkingStore, err := newParkingStore(parkingDataPath)
	if err != nil {
		return nil, err
	}
	parkingSampleInterval, err := parseDuration(env("PARKING_SAMPLE_INTERVAL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid PARKING_SAMPLE_INTERVAL")
	}

	return &app{
		baseURL:       baseURL,
		addr:          env("ADDR", ":8080"),
		rootDomain:    rootDomain,
		defaultTenant: defaultTenant,
		tenants:       tenants,
		sessionSecure: parsed.Scheme == "https",
		allowed:       allowed,
		admins:        admins,
		profiles:      profiles,
		localDevLogin: localDevLogin,
		tokens: &tokenStore{
			secret: secret,
			items:  map[string]loginToken{},
		},
		sessions:              &sessionStore{items: map[string]session{}},
		mailer:                mailTransport,
		templates:             tmpl,
		parkingStore:          parkingStore,
		parkingSampleInterval: parkingSampleInterval,
	}, nil
}

func (a *app) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"service":"weg-portal","status":"ok"}`)
}

func (a *app) home(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, _, tenantSlug, ok := a.currentUser(r)
	if ok && tenantSlug == tenant.Slug {
		http.Redirect(w, r, "/app", http.StatusSeeOther)
		return
	}
	a.render(w, "home", map[string]any{
		"Title":          "WEG Portal " + tenant.Address,
		"Tenant":         tenant,
		"Email":          email,
		"Sent":           r.URL.Query().Get("sent") == "1",
		"MailConfigured": a.mailer.Configured(),
		"DevLoginLink":   "",
		"Denied":         r.URL.Query().Get("denied") == "1",
	})
}

func (a *app) requestLogin(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	email := normalizeEmail(r.FormValue("email"))
	if _, err := mail.ParseAddress(email); err != nil {
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}
	if !a.isAllowed(email, tenant.Slug) {
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}

	token, err := randomToken(32)
	if err != nil {
		http.Error(w, "Could not create login link", http.StatusInternalServerError)
		return
	}
	a.tokens.Put(token, email, tenant.Slug, 15*time.Minute)

	link := a.publicBaseURL(r, tenant) + "/auth/verify?token=" + url.QueryEscape(token)
	if a.localDevLogin && !a.mailer.Configured() {
		a.render(w, "home", map[string]any{
			"Title":          "WEG Portal " + tenant.Address,
			"Tenant":         tenant,
			"Email":          email,
			"Sent":           true,
			"MailConfigured": false,
			"DevLoginLink":   link,
			"Denied":         false,
		})
		return
	}

	if err := a.mailer.SendMagicLink(email, link); err != nil {
		log.Printf("magic link delivery failed for %s: %v", redactedEmail(email), err)
		http.Redirect(w, r, "/?sent=1", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/?sent=1", http.StatusSeeOther)
}

func (a *app) verifyLogin(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	email, tenantSlug, ok := a.tokens.Consume(token)
	if !ok {
		http.Error(w, "Dieser Anmeldelink ist abgelaufen oder wurde bereits verwendet.", http.StatusUnauthorized)
		return
	}

	sid, err := randomToken(32)
	if err != nil {
		http.Error(w, "Could not create session", http.StatusInternalServerError)
		return
	}
	role := a.roleFor(email, tenantSlug)
	a.sessions.Put(sid, email, role, tenantSlug, 12*time.Hour)

	http.SetCookie(w, &http.Cookie{
		Name:     "weg_session",
		Value:    sid,
		Path:     "/",
		Expires:  time.Now().Add(12 * time.Hour),
		HttpOnly: true,
		Secure:   a.sessionSecure,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/app", http.StatusSeeOther)
}

func (a *app) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("weg_session"); err == nil {
		a.sessions.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "weg_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.sessionSecure,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *app) portal(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	profile := a.profileFor(email)
	isAdmin := role == roleAdmin
	a.render(w, "portal", map[string]any{
		"Title":         "WEG Portal",
		"Tenant":        tenant,
		"Email":         email,
		"DisplayName":   profile.DisplayName(),
		"Role":          role,
		"IsAdmin":       isAdmin,
		"CanSeeParking": isAdmin || profile.HasPermission(permissionParking),
	})
}

func (a *app) parking(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	profile := a.profileFor(email)
	if role != roleAdmin && !profile.HasPermission(permissionParking) {
		http.NotFound(w, r)
		return
	}
	isAdmin := role == roleAdmin
	telemetry := a.parkingTelemetry(r.Context(), tenant)
	a.render(w, "parking", map[string]any{
		"Title":       "Parkplatznutzung",
		"Tenant":      tenant,
		"Email":       email,
		"DisplayName": profile.DisplayName(),
		"Role":        role,
		"IsAdmin":     isAdmin,
		"Telemetry":   telemetry,
		"Accounting":  a.parkingAccounting(r.Context(), tenant),
	})
}

func (a *app) updateParkingSettings(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	_, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if role != roleAdmin {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	gridFee, err := parseDecimal(r.FormValue("grid_fee_eur_per_kwh"))
	if err != nil || gridFee < 0 || gridFee > 5 {
		http.Redirect(w, r, "/app/parking?settings=invalid", http.StatusSeeOther)
		return
	}
	if err := a.parkingStore.SetGridFee(tenant.Slug, gridFee); err != nil {
		log.Printf("parking settings save failed for %s: %v", tenant.Slug, err)
		http.Error(w, "Could not save parking settings", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/app/parking?settings=saved", http.StatusSeeOther)
}

func (a *app) updateParkingMonth(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	_, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if role != roleAdmin {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	month := strings.TrimSpace(r.FormValue("month"))
	if _, err := time.Parse("2006-01", month); err != nil {
		http.Redirect(w, r, "/app/parking?month=invalid", http.StatusSeeOther)
		return
	}
	paid := parseBool(r.FormValue("paid"))
	if err := a.parkingStore.SetMonthPaid(tenant.Slug, month, paid); err != nil {
		log.Printf("parking month save failed for %s: %v", tenant.Slug, err)
		http.Error(w, "Could not save parking month", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/app/parking?month=saved", http.StatusSeeOther)
}

func (a *app) userSettings(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if role != roleAdmin {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	profile := a.profileFor(email)
	a.render(w, "userSettings", map[string]any{
		"Title":       "Benutzer & Rechte",
		"Tenant":      tenant,
		"Email":       email,
		"DisplayName": profile.DisplayName(),
		"Role":        role,
		"Users":       a.userRows(tenant.Slug),
	})
}

func (a *app) render(w http.ResponseWriter, name string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render %s failed: %v", name, err)
	}
}

func (a *app) tenantPathRedirect(w http.ResponseWriter, r *http.Request) {
	slug := normalizeSlug(r.PathValue("tenant"))
	tenant, ok := a.tenantBySlug(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	rest := strings.TrimLeft(r.PathValue("rest"), "/")
	targetPath := "/"
	if rest != "" {
		targetPath += rest
	}
	target := tenant.PublicURL(targetPath)
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, target, http.StatusMovedPermanently)
}

func (a *app) tenantForRequest(r *http.Request) tenantConfig {
	host := normalizeHost(r.Host)
	for _, tenant := range a.tenants {
		if tenant.Host != "" && tenant.Host == host {
			return tenant
		}
	}
	if a.rootDomain != "" && strings.HasSuffix(host, "."+a.rootDomain) {
		slug := normalizeSlug(strings.TrimSuffix(host, "."+a.rootDomain))
		if tenant, ok := a.tenantBySlug(slug); ok {
			return tenant
		}
	}
	tenant, _ := a.tenantBySlug(a.defaultTenant)
	return tenant
}

func (a *app) tenantBySlug(slug string) (tenantConfig, bool) {
	tenant, ok := a.tenants[normalizeSlug(slug)]
	return tenant, ok
}

func (a *app) publicBaseURL(r *http.Request, tenant tenantConfig) string {
	host := normalizeHost(r.Host)
	if isLocalHost(host) {
		return a.baseURL
	}
	if tenant.Host != "" {
		return "https://" + tenant.Host
	}
	return a.baseURL
}

func (t tenantConfig) PublicURL(path string) string {
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if t.Host == "" {
		return path
	}
	return "https://" + t.Host + path
}

func (a *app) currentUser(r *http.Request) (string, string, string, bool) {
	c, err := r.Cookie("weg_session")
	if err != nil {
		return "", "", "", false
	}
	return a.sessions.Get(c.Value)
}

func (a *app) isAllowed(email string, tenantSlug string) bool {
	if profile, ok := a.profiles[email]; ok && profile.HasTenant(tenantSlug) {
		return true
	}
	if _, ok := a.admins[email]; ok {
		return true
	}
	_, ok := a.allowed[email]
	return ok
}

func (a *app) roleFor(email string, tenantSlug string) string {
	if profile, ok := a.profiles[email]; ok && profile.Role != "" {
		return profile.Role
	}
	if _, ok := a.admins[email]; ok {
		return roleAdmin
	}
	return roleResident
}

func (a *app) profileFor(email string) userProfile {
	email = normalizeEmail(email)
	if profile, ok := a.profiles[email]; ok {
		return profile
	}
	role := a.roleFor(email, a.defaultTenant)
	return userProfile{
		Email:   email,
		Role:    role,
		Status:  "Eingeladen",
		Tenants: []string{a.defaultTenant},
	}
}

func (a *app) userRows(tenantSlug string) []userRow {
	seen := map[string]struct{}{}
	rows := make([]userRow, 0, len(a.profiles)+len(a.admins)+len(a.allowed))
	for email, profile := range a.profiles {
		if !profile.HasTenant(tenantSlug) {
			continue
		}
		rows = append(rows, profile.UserRow())
		seen[email] = struct{}{}
	}
	for email := range a.admins {
		if _, ok := seen[email]; ok {
			continue
		}
		rows = append(rows, userProfile{Email: email, Role: roleAdmin, Status: "Aktiv", Tenants: []string{tenantSlug}}.UserRow())
		seen[email] = struct{}{}
	}
	for email := range a.allowed {
		if _, ok := seen[email]; ok {
			continue
		}
		rows = append(rows, userProfile{Email: email, Role: roleResident, Status: "Eingeladen", Tenants: []string{tenantSlug}}.UserRow())
	}
	if len(rows) == 0 {
		rows = append(rows, userProfile{Email: "Noch keine Einladungen", Role: roleResident, Status: "Offen", Tenants: []string{tenantSlug}}.UserRow())
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Role != rows[j].Role {
			return rows[i].Role == roleAdmin
		}
		if rows[i].LastName != rows[j].LastName {
			return rows[i].LastName < rows[j].LastName
		}
		return rows[i].Email < rows[j].Email
	})
	return rows
}

func (a *app) parkingTelemetry(ctx context.Context, tenant tenantConfig) parkingTelemetry {
	ha := tenant.HA
	telemetry := parkingTelemetry{
		Configured: ha.baseURL != "",
		Entities: []parkingEntityRef{
			{Label: "Zählerstand", EntityID: ha.meterEnergyEntity},
			{Label: "Leistung", EntityID: ha.powerEntity},
			{Label: "aWATTar Preis", EntityID: ha.priceEntity},
		},
	}
	if ha.baseURL == "" {
		telemetry.Message = "Home Assistant ist lokal noch nicht konfiguriert."
		return telemetry
	}
	if ha.token == "" {
		telemetry.Message = "Home Assistant ist vorbereitet, aber lokal fehlt noch ein HA_TOKEN."
		return telemetry
	}

	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	var liveEnergy float64
	var livePrice float64
	var hasLiveEnergy bool
	var hasLivePrice bool
	states := []struct {
		label  string
		entity string
	}{
		{label: "Zählerstand", entity: ha.meterEnergyEntity},
		{label: "Aktuelle Leistung", entity: ha.powerEntity},
		{label: "aWATTar Gesamtpreis", entity: ha.priceEntity},
	}
	for _, item := range states {
		state, err := ha.State(ctx, item.entity)
		if err != nil {
			telemetry.Message = "Home Assistant konnte gerade nicht gelesen werden."
			return telemetry
		}
		telemetry.Metrics = append(telemetry.Metrics, parkingMetric{
			Label:  item.label,
			Value:  formatHAValue(state),
			Detail: item.entity,
		})
		switch item.entity {
		case ha.meterEnergyEntity:
			if value, err := parseHAFloat(state.State); err == nil {
				liveEnergy = value
				hasLiveEnergy = true
			}
		case ha.priceEntity:
			if value, err := parseHAFloat(state.State); err == nil {
				livePrice = value
				hasLivePrice = true
			}
		}
	}
	if hasLiveEnergy && hasLivePrice {
		if err := a.parkingStore.AppendSamples(tenant.Slug, []parkingStoredSample{{
			At:             time.Now().UTC(),
			EnergyKWh:      liveEnergy,
			PriceEURPerKWh: livePrice,
		}}); err != nil {
			log.Printf("parking live sample save failed for %s: %v", tenant.Slug, err)
		}
	}
	telemetry.Connected = true
	telemetry.Message = "Live aus Home Assistant gelesen."
	return telemetry
}

func (a *app) parkingAccounting(ctx context.Context, tenant tenantConfig) parkingAccountingView {
	seedCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	if err := a.seedParkingHistory(seedCtx, tenant, 35*24*time.Hour); err != nil && tenant.HA.baseURL != "" && tenant.HA.token != "" {
		log.Printf("parking history seed failed for %s: %v", tenant.Slug, err)
	}

	data := a.parkingStore.TenantData(tenant.Slug)
	months := calculateParkingMonths(data, time.Now(), time.Local)
	view := parkingAccountingView{
		GridFeeValue:     formatInputFloat(data.Settings.GridFeeEURPerKWh),
		GridFeeLabel:     formatEURPerKWh(data.Settings.GridFeeEURPerKWh),
		Months:           months,
		HasMonths:        len(months) > 0,
		HistoryAvailable: len(data.EnergySamples) >= 2 && len(data.PriceSamples) > 0,
	}
	if len(data.EnergySamples) > 0 {
		last := data.EnergySamples[len(data.EnergySamples)-1].At.In(time.Local)
		view.LastSampleLabel = last.Format("02.01.2006 15:04")
	}
	if len(months) == 0 {
		view.Message = "Noch nicht genug Messpunkte für eine Monatsabrechnung. Die App sammelt ab jetzt eigene Messpunkte und liest zusätzlich verfügbare Home-Assistant-Historie ein."
	} else {
		view.Message = "Kosten werden stündlich aus Zählerdifferenz, aWATTar-Preis und Netzbetreibergebühren berechnet."
	}
	return view
}

func (a *app) seedParkingHistory(ctx context.Context, tenant tenantConfig, lookback time.Duration) error {
	ha := tenant.HA
	if ha.baseURL == "" || ha.token == "" || ha.meterEnergyEntity == "" || ha.priceEntity == "" {
		return nil
	}
	end := time.Now()
	start := end.Add(-lookback)
	history, err := ha.History(ctx, start, end, []string{ha.meterEnergyEntity, ha.priceEntity})
	if err != nil {
		return err
	}
	energySamples := samplesFromHistory(history[ha.meterEnergyEntity])
	priceSamples := samplesFromHistory(history[ha.priceEntity])
	if len(energySamples) == 0 && len(priceSamples) == 0 {
		return nil
	}
	return a.parkingStore.AppendReadings(tenant.Slug, energySamples, priceSamples)
}

func (a *app) startParkingSampler() func() {
	if a.parkingSampleInterval <= 0 {
		log.Printf("parking sampler disabled")
		return func() {}
	}
	configuredTenants := 0
	for _, tenant := range a.tenants {
		if tenant.HA.baseURL != "" && tenant.HA.token != "" {
			configuredTenants++
		}
	}
	if configuredTenants == 0 {
		log.Printf("parking sampler disabled: no configured Home Assistant tenants")
		return func() {}
	}
	log.Printf("parking sampler enabled for %d tenant(s), interval %s", configuredTenants, a.parkingSampleInterval)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		a.sampleParkingTenants(ctx)
		ticker := time.NewTicker(a.parkingSampleInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.sampleParkingTenants(ctx)
			}
		}
	}()
	return cancel
}

func (a *app) sampleParkingTenants(ctx context.Context) {
	for _, tenant := range a.tenants {
		if tenant.HA.baseURL == "" || tenant.HA.token == "" {
			continue
		}
		if err := a.sampleParkingTenant(ctx, tenant); err != nil {
			log.Printf("parking sample failed for %s: %v", tenant.Slug, err)
			continue
		}
		log.Printf("parking sample saved for %s", tenant.Slug)
	}
}

func (a *app) sampleParkingTenant(ctx context.Context, tenant tenantConfig) error {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	energy, err := tenant.HA.State(ctx, tenant.HA.meterEnergyEntity)
	if err != nil {
		return err
	}
	price, err := tenant.HA.State(ctx, tenant.HA.priceEntity)
	if err != nil {
		return err
	}
	energyValue, err := parseHAFloat(energy.State)
	if err != nil {
		return err
	}
	priceValue, err := parseHAFloat(price.State)
	if err != nil {
		return err
	}
	return a.parkingStore.AppendSamples(tenant.Slug, []parkingStoredSample{{
		At:             time.Now().UTC(),
		EnergyKWh:      energyValue,
		PriceEURPerKWh: priceValue,
	}})
}

func newParkingStore(path string) (*parkingStore, error) {
	store := &parkingStore{
		path: path,
		data: parkingStoreData{Tenants: map[string]parkingTenantData{}},
	}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read parking data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid parking data")
	}
	if store.data.Tenants == nil {
		store.data.Tenants = map[string]parkingTenantData{}
	}
	return store, nil
}

func (s *parkingStore) TenantData(tenantSlug string) parkingTenantData {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.tenantLocked(tenantSlug)
	data.Months = copyMonthStates(data.Months)
	data.EnergySamples = append([]parkingNumericSample(nil), data.EnergySamples...)
	data.PriceSamples = append([]parkingNumericSample(nil), data.PriceSamples...)
	data.Samples = append([]parkingStoredSample(nil), data.Samples...)
	return data
}

func (s *parkingStore) SetGridFee(tenantSlug string, gridFee float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.tenantLocked(tenantSlug)
	data.Settings.GridFeeEURPerKWh = gridFee
	s.data.Tenants[normalizeSlug(tenantSlug)] = data
	return s.saveLocked()
}

func (s *parkingStore) SetMonthPaid(tenantSlug string, month string, paid bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.tenantLocked(tenantSlug)
	if data.Months == nil {
		data.Months = map[string]parkingMonthState{}
	}
	data.Months[month] = parkingMonthState{Paid: paid}
	s.data.Tenants[normalizeSlug(tenantSlug)] = data
	return s.saveLocked()
}

func (s *parkingStore) AppendSamples(tenantSlug string, samples []parkingStoredSample) error {
	if len(samples) == 0 {
		return nil
	}
	energy := make([]parkingNumericSample, 0, len(samples))
	prices := make([]parkingNumericSample, 0, len(samples))
	for _, sample := range samples {
		energy = append(energy, parkingNumericSample{At: sample.At, Value: sample.EnergyKWh})
		prices = append(prices, parkingNumericSample{At: sample.At, Value: sample.PriceEURPerKWh})
	}
	return s.AppendReadings(tenantSlug, energy, prices)
}

func (s *parkingStore) AppendReadings(tenantSlug string, energySamples []parkingNumericSample, priceSamples []parkingNumericSample) error {
	if len(energySamples) == 0 && len(priceSamples) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.tenantLocked(tenantSlug)
	for _, sample := range energySamples {
		if sample.At.IsZero() || sample.Value < 0 || sample.Value > 1000000 {
			continue
		}
		data.EnergySamples = append(data.EnergySamples, parkingNumericSample{
			At:    sample.At.UTC(),
			Value: sample.Value,
		})
	}
	for _, sample := range priceSamples {
		if sample.At.IsZero() || sample.Value < -5 || sample.Value > 5 {
			continue
		}
		data.PriceSamples = append(data.PriceSamples, parkingNumericSample{
			At:    sample.At.UTC(),
			Value: sample.Value,
		})
	}
	keepAfter := time.Now().AddDate(-1, -1, 0)
	data.EnergySamples = normalizeNumericSamples(data.EnergySamples, keepAfter)
	data.PriceSamples = normalizeNumericSamples(data.PriceSamples, keepAfter)
	data.Samples = nil
	s.data.Tenants[normalizeSlug(tenantSlug)] = data
	return s.saveLocked()
}

func (s *parkingStore) tenantLocked(tenantSlug string) parkingTenantData {
	tenantSlug = normalizeSlug(tenantSlug)
	if tenantSlug == "" {
		tenantSlug = "default"
	}
	if s.data.Tenants == nil {
		s.data.Tenants = map[string]parkingTenantData{}
	}
	data, ok := s.data.Tenants[tenantSlug]
	if !ok {
		data = defaultParkingTenantData()
	}
	if data.Months == nil {
		data.Months = map[string]parkingMonthState{}
	}
	if len(data.Samples) > 0 {
		for _, sample := range data.Samples {
			data.EnergySamples = append(data.EnergySamples, parkingNumericSample{At: sample.At, Value: sample.EnergyKWh})
			data.PriceSamples = append(data.PriceSamples, parkingNumericSample{At: sample.At, Value: sample.PriceEURPerKWh})
		}
		data.Samples = nil
	}
	keepAfter := time.Now().AddDate(-1, -1, 0)
	data.EnergySamples = normalizeNumericSamples(data.EnergySamples, keepAfter)
	data.PriceSamples = normalizeNumericSamples(data.PriceSamples, keepAfter)
	s.data.Tenants[tenantSlug] = data
	return data
}

func (s *parkingStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("could not create parking data directory")
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode parking data")
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("could not write parking data")
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("could not replace parking data")
	}
	return nil
}

func defaultParkingTenantData() parkingTenantData {
	return parkingTenantData{
		Settings: parkingSettings{GridFeeEURPerKWh: 0.10},
		Months:   map[string]parkingMonthState{},
	}
}

func copyMonthStates(in map[string]parkingMonthState) map[string]parkingMonthState {
	out := map[string]parkingMonthState{}
	for month, state := range in {
		out[month] = state
	}
	return out
}

func normalizeNumericSamples(samples []parkingNumericSample, keepAfter time.Time) []parkingNumericSample {
	out := make([]parkingNumericSample, 0, len(samples))
	for _, sample := range samples {
		if sample.At.IsZero() || sample.At.Before(keepAfter) {
			continue
		}
		out = append(out, parkingNumericSample{At: sample.At.UTC().Truncate(time.Second), Value: sample.Value})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At)
	})
	deduped := out[:0]
	for _, sample := range out {
		if len(deduped) > 0 && deduped[len(deduped)-1].At.Equal(sample.At) {
			deduped[len(deduped)-1] = sample
			continue
		}
		deduped = append(deduped, sample)
	}
	return deduped
}

func samplesFromHistory(history []haHistoryState) []parkingNumericSample {
	out := make([]parkingNumericSample, 0, len(history))
	for _, item := range history {
		value, err := parseHAFloat(item.State)
		if err != nil {
			continue
		}
		at := item.timestamp()
		if at.IsZero() {
			continue
		}
		out = append(out, parkingNumericSample{At: at.UTC(), Value: value})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At)
	})
	return out
}

func priceAt(samples []parkingNumericSample, at time.Time) (float64, bool) {
	if len(samples) == 0 {
		return 0, false
	}
	index := sort.Search(len(samples), func(i int) bool {
		return samples[i].At.After(at)
	})
	if index == 0 {
		return samples[0].Value, true
	}
	return samples[index-1].Value, true
}

type parkingHourUsage struct {
	At         time.Time
	KWh        float64
	PriceEUR   float64
	EnergyCost float64
	GridCost   float64
}

func calculateParkingHourlyUsage(energySamples []parkingNumericSample, priceSamples []parkingNumericSample, gridFeeEURPerKWh float64, now time.Time) []parkingHourUsage {
	keepAfter := now.AddDate(-1, -1, 0)
	energySamples = normalizeNumericSamples(append([]parkingNumericSample(nil), energySamples...), keepAfter)
	priceSamples = normalizeNumericSamples(append([]parkingNumericSample(nil), priceSamples...), keepAfter)
	if len(energySamples) < 2 || len(priceSamples) == 0 {
		return nil
	}
	var out []parkingHourUsage
	for i := 1; i < len(energySamples); i++ {
		prev := energySamples[i-1]
		curr := energySamples[i]
		if !curr.At.After(prev.At) {
			continue
		}
		delta := curr.Value - prev.Value
		if delta <= 0 || delta > 500 {
			continue
		}
		totalSeconds := curr.At.Sub(prev.At).Seconds()
		if totalSeconds <= 0 {
			continue
		}
		cursor := prev.At
		for cursor.Before(curr.At) {
			nextHour := cursor.Truncate(time.Hour).Add(time.Hour)
			segmentEnd := nextHour
			if segmentEnd.After(curr.At) {
				segmentEnd = curr.At
			}
			seconds := segmentEnd.Sub(cursor).Seconds()
			if seconds <= 0 {
				break
			}
			kWh := delta * (seconds / totalSeconds)
			price, ok := priceAt(priceSamples, cursor)
			if !ok {
				cursor = segmentEnd
				continue
			}
			hour := cursor.Truncate(time.Hour)
			out = append(out, parkingHourUsage{
				At:         hour,
				KWh:        kWh,
				PriceEUR:   price,
				EnergyCost: price * kWh,
				GridCost:   gridFeeEURPerKWh * kWh,
			})
			cursor = segmentEnd
		}
	}
	if len(out) == 0 {
		return nil
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At)
	})
	merged := out[:0]
	for _, item := range out {
		if len(merged) > 0 && merged[len(merged)-1].At.Equal(item.At) {
			merged[len(merged)-1].KWh += item.KWh
			merged[len(merged)-1].EnergyCost += item.EnergyCost
			merged[len(merged)-1].GridCost += item.GridCost
			continue
		}
		merged = append(merged, item)
	}
	return merged
}

func calculateParkingMonths(data parkingTenantData, now time.Time, loc *time.Location) []parkingMonthView {
	if loc == nil {
		loc = time.Local
	}
	hours := calculateParkingHourlyUsage(data.EnergySamples, data.PriceSamples, data.Settings.GridFeeEURPerKWh, now)
	if len(hours) == 0 {
		return nil
	}
	type aggregate struct {
		month      string
		kWh        float64
		energyCost float64
		gridCost   float64
		first      time.Time
		last       time.Time
		hourCount  int
	}
	aggregates := map[string]*aggregate{}
	for _, hour := range hours {
		month := hour.At.In(loc).Format("2006-01")
		hourEnd := hour.At.Add(time.Hour)
		agg := aggregates[month]
		if agg == nil {
			agg = &aggregate{month: month, first: hour.At, last: hourEnd}
			aggregates[month] = agg
		}
		if hour.At.Before(agg.first) {
			agg.first = hour.At
		}
		if hourEnd.After(agg.last) {
			agg.last = hourEnd
		}
		agg.kWh += hour.KWh
		agg.energyCost += hour.EnergyCost
		agg.gridCost += hour.GridCost
		agg.hourCount++
	}
	for month := range data.Months {
		if _, ok := aggregates[month]; !ok {
			parsed, err := time.ParseInLocation("2006-01", month, loc)
			if err != nil {
				continue
			}
			aggregates[month] = &aggregate{month: month, first: parsed, last: parsed}
		}
	}
	if len(aggregates) == 0 {
		return nil
	}
	maxTotal := 0.0
	months := make([]string, 0, len(aggregates))
	for month, agg := range aggregates {
		months = append(months, month)
		total := agg.energyCost + agg.gridCost
		if total > maxTotal {
			maxTotal = total
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(months)))
	out := make([]parkingMonthView, 0, len(months))
	for _, month := range months {
		agg := aggregates[month]
		total := agg.energyCost + agg.gridCost
		averagePrice := 0.0
		if agg.kWh > 0 {
			averagePrice = total / agg.kWh
		}
		chartPercent := 0
		if maxTotal > 0 {
			chartPercent = int(total / maxTotal * 100)
			if chartPercent < 3 && total > 0 {
				chartPercent = 3
			}
		}
		paid := data.Months[month].Paid
		firstOfMonth, _ := time.ParseInLocation("2006-01", month, loc)
		out = append(out, parkingMonthView{
			Month:           month,
			MonthLabel:      formatMonthLabel(month, loc),
			PeriodLabel:     formatPeriodLabel(agg.first, agg.last, loc),
			KWh:             formatKWh(agg.kWh),
			EnergyCost:      formatEUR(agg.energyCost),
			GridCost:        formatEUR(agg.gridCost),
			TotalCost:       formatEUR(total),
			AveragePrice:    formatEURPerKWh(averagePrice),
			Paid:            paid,
			PaidLabel:       paidLabel(paid),
			TogglePaidValue: boolFormValue(!paid),
			ToggleLabel:     togglePaidLabel(paid),
			ChartPercent:    chartPercent,
			Partial:         agg.hourCount == 0 || agg.first.In(loc).After(firstOfMonth.Add(24*time.Hour)),
			SampleCount:     len(data.EnergySamples),
			HourCount:       agg.hourCount,
		})
	}
	return out
}

type userProfile struct {
	Email       string   `json:"email"`
	Title       string   `json:"title"`
	FirstName   string   `json:"first_name"`
	LastName    string   `json:"last_name"`
	Role        string   `json:"role"`
	Status      string   `json:"status"`
	Tenants     []string `json:"tenants"`
	Permissions []string `json:"permissions"`
}

func (p userProfile) DisplayName() string {
	parts := []string{}
	for _, part := range []string{p.Title, p.FirstName, p.LastName} {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	return p.Email
}

func (p userProfile) HasPermission(permission string) bool {
	permission = strings.ToLower(strings.TrimSpace(permission))
	for _, item := range p.Permissions {
		if strings.ToLower(strings.TrimSpace(item)) == permission {
			return true
		}
	}
	return false
}

func (p userProfile) HasTenant(tenantSlug string) bool {
	tenantSlug = normalizeSlug(tenantSlug)
	for _, item := range p.Tenants {
		if normalizeSlug(item) == tenantSlug {
			return true
		}
	}
	return false
}

func (p userProfile) UserRow() userRow {
	if p.Role == "" {
		p.Role = roleResident
	}
	if p.Status == "" {
		p.Status = "Eingeladen"
	}
	return userRow{
		Email:           p.Email,
		Title:           p.Title,
		FirstName:       p.FirstName,
		LastName:        p.LastName,
		DisplayName:     p.DisplayName(),
		Role:            p.Role,
		Status:          p.Status,
		Tenants:         strings.Join(p.Tenants, ", "),
		PermissionLabel: permissionLabel(p.Permissions),
	}
}

type userRow struct {
	Email           string
	Title           string
	FirstName       string
	LastName        string
	DisplayName     string
	Role            string
	Status          string
	Tenants         string
	PermissionLabel string
}

func (s *tokenStore) Put(token string, email string, tenantSlug string, ttl time.Duration) {
	key := s.digest(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = loginToken{email: email, tenantSlug: tenantSlug, expiresAt: time.Now().Add(ttl)}
}

func (s *tokenStore) Consume(token string) (string, string, bool) {
	if token == "" {
		return "", "", false
	}
	key := s.digest(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[key]
	if !ok || item.used || time.Now().After(item.expiresAt) {
		delete(s.items, key)
		return "", "", false
	}
	item.used = true
	s.items[key] = item
	return item.email, item.tenantSlug, true
}

func (s *tokenStore) digest(token string) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(token))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *sessionStore) Put(id string, email string, role string, tenantSlug string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[id] = session{email: email, role: role, tenantSlug: tenantSlug, expiresAt: time.Now().Add(ttl)}
}

func (s *sessionStore) Get(id string) (string, string, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok || time.Now().After(item.expiresAt) {
		delete(s.items, id)
		return "", "", "", false
	}
	return item.email, item.role, item.tenantSlug, true
}

func (s *sessionStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, id)
}

func (m smtpMailer) Configured() bool {
	return m.host != "" && m.port != "" && m.from != ""
}

func (m smtpMailer) Validate() error {
	if !m.Configured() {
		return nil
	}
	if _, err := mail.ParseAddress(m.from); err != nil {
		return fmt.Errorf("invalid MAIL_FROM")
	}
	if (m.user == "") != (m.pass == "") {
		return fmt.Errorf("SMTP_USER and SMTP_PASS must be set together")
	}
	if _, err := strconv.Atoi(m.port); err != nil {
		return fmt.Errorf("SMTP_PORT must be numeric")
	}
	return nil
}

func (m smtpMailer) auth() smtp.Auth {
	if m.user == "" && m.pass == "" {
		return nil
	}
	return smtp.PlainAuth("", m.user, m.pass, m.host)
}

func (m smtpMailer) SendMagicLink(to string, link string) error {
	if !m.Configured() {
		return errors.New("smtp not configured")
	}

	addr := net.JoinHostPort(m.host, m.port)
	fromAddr, err := mail.ParseAddress(m.from)
	if err != nil {
		return fmt.Errorf("invalid MAIL_FROM")
	}

	msg := strings.Join([]string{
		"From: " + m.from,
		"To: " + to,
		"Subject: Ihr Zugang zum WEG Portal",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		"Hallo,",
		"",
		"hier ist Ihr Anmeldelink für das WEG Portal:",
		link,
		"",
		"Der Link ist 15 Minuten gültig und kann nur einmal verwendet werden.",
		"",
		"Freundliche Grüße",
		"WEG Portal",
	}, "\r\n")

	return smtp.SendMail(addr, m.auth(), fromAddr.Address, []string{to}, []byte(msg))
}

func newHomeAssistantConfig() homeAssistantConfig {
	return homeAssistantConfig{
		baseURL:           strings.TrimRight(env("HA_BASE_URL", ""), "/"),
		token:             strings.TrimSpace(os.Getenv("HA_TOKEN")),
		meterEnergyEntity: env("PARKING_METER_ENERGY_ENTITY", "sensor.kws_306wf_energy_meter_energy"),
		powerEntity:       env("PARKING_POWER_ENTITY", "sensor.kws360_power"),
		priceEntity:       env("PARKING_PRICE_ENTITY", "sensor.epex_spot_data_total_price"),
	}
}

func (c homeAssistantConfig) State(ctx context.Context, entityID string) (haState, error) {
	if c.baseURL == "" || c.token == "" || entityID == "" {
		return haState{}, errors.New("home assistant not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/states/"+url.PathEscape(entityID), nil)
	if err != nil {
		return haState{}, errors.New("could not build home assistant request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return haState{}, errors.New("home assistant request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return haState{}, fmt.Errorf("home assistant returned %d", resp.StatusCode)
	}

	var state haState
	dec := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	if err := dec.Decode(&state); err != nil {
		return haState{}, errors.New("home assistant returned invalid json")
	}
	return state, nil
}

func (c homeAssistantConfig) History(ctx context.Context, start time.Time, end time.Time, entityIDs []string) (map[string][]haHistoryState, error) {
	if c.baseURL == "" || c.token == "" || len(entityIDs) == 0 {
		return nil, errors.New("home assistant not configured")
	}
	endpoint := c.baseURL + "/api/history/period/" + url.PathEscape(start.UTC().Format(time.RFC3339))
	q := url.Values{}
	q.Set("end_time", end.UTC().Format(time.RFC3339))
	q.Set("filter_entity_id", strings.Join(entityIDs, ","))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, errors.New("could not build home assistant history request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.New("home assistant history request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("home assistant history returned %d", resp.StatusCode)
	}

	var groups [][]haHistoryState
	dec := json.NewDecoder(io.LimitReader(resp.Body, 8<<20))
	if err := dec.Decode(&groups); err != nil {
		return nil, errors.New("home assistant history returned invalid json")
	}
	out := map[string][]haHistoryState{}
	for _, group := range groups {
		for _, item := range group {
			if item.EntityID == "" {
				continue
			}
			out[item.EntityID] = append(out[item.EntityID], item)
		}
	}
	for entityID := range out {
		sort.Slice(out[entityID], func(i, j int) bool {
			return out[entityID][i].timestamp().Before(out[entityID][j].timestamp())
		})
	}
	return out, nil
}

func (s haHistoryState) timestamp() time.Time {
	if !s.LastChanged.IsZero() {
		return s.LastChanged
	}
	return s.LastUpdated
}

func formatHAValue(state haState) string {
	unit, _ := state.Attributes["unit_of_measurement"].(string)
	value := strings.TrimSpace(state.State)
	if value == "" {
		value = "unbekannt"
	}
	if unit == "" {
		return value
	}
	return value + " " + unit
}

func parseHAFloat(raw string) (float64, error) {
	value := strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	if value == "" || strings.EqualFold(value, "unknown") || strings.EqualFold(value, "unavailable") {
		return 0, errors.New("state is not numeric")
	}
	return strconv.ParseFloat(value, 64)
}

func parseDecimal(raw string) (float64, error) {
	value := strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	if value == "" {
		return 0, errors.New("empty decimal")
	}
	return strconv.ParseFloat(value, 64)
}

func parseDuration(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	if duration, err := time.ParseDuration(raw); err == nil {
		return duration, nil
	}
	minutes, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	return time.Duration(minutes) * time.Minute, nil
}

func formatInputFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 3, 64)
}

func formatEUR(value float64) string {
	return strings.Replace(fmt.Sprintf("%.2f €", value), ".", ",", 1)
}

func formatEURPerKWh(value float64) string {
	return strings.Replace(fmt.Sprintf("%.3f €/kWh", value), ".", ",", 1)
}

func formatKWh(value float64) string {
	return strings.Replace(fmt.Sprintf("%.2f kWh", value), ".", ",", 1)
}

func formatMonthLabel(month string, loc *time.Location) string {
	t, err := time.ParseInLocation("2006-01", month, loc)
	if err != nil {
		return month
	}
	names := []string{"Jänner", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"}
	return names[int(t.Month())-1] + " " + strconv.Itoa(t.Year())
}

func formatPeriodLabel(first time.Time, last time.Time, loc *time.Location) string {
	if first.IsZero() || last.IsZero() {
		return "Noch keine Messwerte"
	}
	if loc == nil {
		loc = time.Local
	}
	return first.In(loc).Format("02.01. 15:04") + " bis " + last.In(loc).Format("02.01. 15:04")
}

func paidLabel(paid bool) string {
	if paid {
		return "Bezahlt"
	}
	return "Offen"
}

func togglePaidLabel(paid bool) string {
	if paid {
		return "Als offen markieren"
	}
	return "Als bezahlt markieren"
}

func boolFormValue(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self'; style-src 'self' 'unsafe-inline'; form-action 'self'; base-uri 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func env(key string, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func loadLocalEnv(path string) error {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("could not read local env file")
	}

	for i, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("invalid local env line %d", i+1)
		}
		key = strings.TrimSpace(key)
		if key == "" || strings.ContainsAny(key, " \t") {
			return fmt.Errorf("invalid local env key on line %d", i+1)
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, trimEnvQuotes(strings.TrimSpace(value)))
	}
	return nil
}

func trimEnvQuotes(value string) string {
	if len(value) < 2 {
		return value
	}
	if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}

func parseBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func isLocalHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func parseAllowed(raw string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, item := range strings.Split(raw, ",") {
		email := normalizeEmail(item)
		if email != "" {
			out[email] = struct{}{}
		}
	}
	return out
}

func parseTenants(raw string, rootDomain string, defaultTenant string, defaultHA homeAssistantConfig) (map[string]tenantConfig, error) {
	out := map[string]tenantConfig{}
	raw = strings.TrimSpace(raw)
	if raw != "" {
		var tenants []tenantConfig
		if err := json.Unmarshal([]byte(raw), &tenants); err != nil {
			return nil, fmt.Errorf("invalid WEG_TENANTS_JSON")
		}
		for _, tenant := range tenants {
			tenant.Slug = normalizeSlug(tenant.Slug)
			if tenant.Slug == "" {
				return nil, fmt.Errorf("tenant is missing slug")
			}
			if tenant.Name == "" {
				tenant.Name = "WEG Portal"
			}
			if tenant.Address == "" {
				tenant.Address = tenant.Slug
			}
			tenant.Host = normalizeHost(tenant.Host)
			if tenant.Host == "" && rootDomain != "" {
				tenant.Host = tenant.Slug + "." + rootDomain
			}
			if tenant.HA.baseURL == "" && tenant.Slug == normalizeSlug(defaultTenant) {
				tenant.HA = defaultHA
			}
			out[tenant.Slug] = tenant
		}
	}

	defaultTenant = normalizeSlug(defaultTenant)
	if _, ok := out[defaultTenant]; !ok {
		host := ""
		if rootDomain != "" {
			host = defaultTenant + "." + rootDomain
		}
		out[defaultTenant] = tenantConfig{
			Slug:    defaultTenant,
			Name:    "WEG Portal",
			Address: "Janischhofweg 22",
			Host:    host,
			HA:      defaultHA,
		}
	}
	return out, nil
}

func parseUserProfiles(raw string, allowed map[string]struct{}, admins map[string]struct{}, defaultTenant string) (map[string]userProfile, error) {
	out := map[string]userProfile{}
	defaultTenant = normalizeSlug(defaultTenant)
	raw = strings.TrimSpace(raw)
	if raw != "" {
		var profiles []userProfile
		if err := json.Unmarshal([]byte(raw), &profiles); err != nil {
			return nil, fmt.Errorf("invalid WEG_USERS_JSON")
		}
		for _, profile := range profiles {
			email := normalizeEmail(profile.Email)
			if email == "" {
				return nil, fmt.Errorf("user profile is missing email")
			}
			if _, err := mail.ParseAddress(email); err != nil {
				return nil, fmt.Errorf("user profile has invalid email")
			}
			profile.Email = email
			profile.Title = strings.TrimSpace(profile.Title)
			profile.FirstName = strings.TrimSpace(profile.FirstName)
			profile.LastName = strings.TrimSpace(profile.LastName)
			profile.Role = normalizeRole(profile.Role)
			if profile.Role == "" {
				if _, ok := admins[email]; ok {
					profile.Role = roleAdmin
				} else {
					profile.Role = roleResident
				}
			}
			if profile.Status == "" {
				profile.Status = "Eingeladen"
			}
			profile.Tenants = normalizeTenants(profile.Tenants, defaultTenant)
			profile.Permissions = normalizePermissions(profile.Permissions)
			out[email] = profile
		}
	}

	for email := range admins {
		if _, ok := out[email]; ok {
			continue
		}
		out[email] = userProfile{Email: email, Role: roleAdmin, Status: "Aktiv", Tenants: []string{defaultTenant}}
	}
	for email := range allowed {
		if _, ok := out[email]; ok {
			continue
		}
		out[email] = userProfile{Email: email, Role: roleResident, Status: "Eingeladen", Tenants: []string{defaultTenant}}
	}
	return out, nil
}

func normalizeRole(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "admin":
		return roleAdmin
	case "bewohner", "resident":
		return roleResident
	default:
		return strings.TrimSpace(raw)
	}
}

func normalizePermissions(raw []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, item := range raw {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func normalizeTenants(raw []string, fallback string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, item := range raw {
		item = normalizeSlug(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	if len(out) == 0 && fallback != "" {
		out = append(out, normalizeSlug(fallback))
	}
	sort.Strings(out)
	return out
}

func normalizeSlug(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, "_", "-")
	return raw
}

func normalizeHost(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(raw)
	if err == nil {
		raw = host
	}
	return strings.TrimSuffix(raw, ".")
}

func permissionLabel(permissions []string) string {
	labels := []string{}
	for _, permission := range normalizePermissions(permissions) {
		switch permission {
		case permissionParking:
			labels = append(labels, "Parkplatznutzung")
		default:
			labels = append(labels, permission)
		}
	}
	if len(labels) == 0 {
		return "Standard"
	}
	return strings.Join(labels, ", ")
}

func normalizeEmail(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func randomToken(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func sessionSecret(requireConfigured bool) ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv("SESSION_KEY"))
	if raw == "" {
		if requireConfigured {
			return nil, fmt.Errorf("SESSION_KEY is required when BASE_URL is public")
		}
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return nil, err
		}
		return secret, nil
	}
	if n, err := strconv.Atoi(raw); err == nil && n == 0 {
		return nil, fmt.Errorf("SESSION_KEY must not be empty")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err == nil && len(decoded) >= 32 {
		return decoded, nil
	}
	if len(raw) < 32 {
		return nil, fmt.Errorf("SESSION_KEY must be at least 32 bytes or base64url-encoded 32 bytes")
	}
	return []byte(raw), nil
}

func redactedEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "<redacted>"
	}
	name := parts[0]
	if len(name) > 1 {
		name = name[:1] + "***"
	} else {
		name = "***"
	}
	return name + "@" + parts[1]
}

const pageTemplates = `
{{define "home"}}
<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
    :root {
      color-scheme: light;
      --ink: #17201d;
      --muted: #5f6f67;
      --line: rgba(23, 32, 29, .14);
      --paper: #f8faf7;
      --leaf: #276447;
      --leaf-dark: #173f2d;
      --sky: #dbeef5;
      --sun: #f3cc73;
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      color: var(--ink);
      background: #edf3ec;
    }
    .hero {
      position: relative;
      min-height: 100vh;
      overflow: hidden;
      display: grid;
      grid-template-rows: auto 1fr auto;
    }
    .hero::before {
      content: "";
      position: absolute;
      inset: -24px;
      background: url('/assets/jhw22-hero.jpg') center / cover no-repeat;
      filter: blur(14px) saturate(.94) brightness(.86);
      transform: scale(1.04);
      z-index: -2;
    }
    .hero::after {
      content: "";
      position: absolute;
      inset: 0;
      background:
        linear-gradient(90deg, rgba(248,250,247,.96) 0%, rgba(248,250,247,.88) 34%, rgba(248,250,247,.48) 62%, rgba(248,250,247,.2) 100%),
        linear-gradient(180deg, rgba(23,32,29,.1), rgba(23,32,29,.42));
      z-index: -1;
    }
    header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 24px;
      padding: 28px clamp(20px, 5vw, 72px);
    }
    .brand {
      display: inline-flex;
      align-items: center;
      gap: 12px;
      font-weight: 750;
      letter-spacing: 0;
      color: var(--ink);
      text-decoration: none;
    }
    .mark {
      min-width: 44px;
      height: 38px;
      border-radius: 8px;
      background: linear-gradient(135deg, var(--leaf), #77a979);
      display: grid;
      place-items: center;
      padding: 0 9px;
      color: white;
      font-weight: 800;
    }
    nav {
      display: flex;
      gap: 18px;
      color: var(--leaf-dark);
      font-size: 14px;
      font-weight: 650;
    }
    main {
      display: grid;
      grid-template-columns: minmax(0, 1.1fr) minmax(320px, 440px);
      gap: clamp(28px, 6vw, 92px);
      align-items: center;
      padding: 32px clamp(20px, 5vw, 72px) 48px;
    }
    .copy {
      max-width: 760px;
      padding-bottom: 8vh;
    }
    .eyebrow {
      color: var(--leaf-dark);
      font-size: 13px;
      font-weight: 800;
      text-transform: uppercase;
      letter-spacing: .12em;
      margin-bottom: 18px;
    }
    h1 {
      margin: 0;
      font-size: clamp(48px, 7vw, 104px);
      line-height: .92;
      letter-spacing: 0;
      max-width: 860px;
    }
    .lead {
      max-width: 620px;
      margin: 26px 0 0;
      font-size: clamp(18px, 2vw, 23px);
      line-height: 1.45;
      color: #2d3c36;
    }
    .login {
      background: rgba(248,250,247,.9);
      border: 1px solid rgba(255,255,255,.68);
      box-shadow: 0 24px 70px rgba(22, 44, 32, .2);
      border-radius: 8px;
      padding: 28px;
      backdrop-filter: blur(20px);
    }
    .login h2 {
      margin: 0;
      font-size: 24px;
      letter-spacing: 0;
    }
    .login p {
      color: var(--muted);
      line-height: 1.5;
      margin: 10px 0 22px;
    }
    label {
      display: block;
      font-size: 13px;
      font-weight: 750;
      color: var(--leaf-dark);
      margin-bottom: 8px;
    }
    input {
      width: 100%;
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 14px 15px;
      font: inherit;
      background: white;
      color: var(--ink);
    }
    button {
      width: 100%;
      border: 0;
      border-radius: 8px;
      padding: 14px 16px;
      margin-top: 14px;
      font: inherit;
      font-weight: 800;
      color: white;
      background: var(--leaf);
      cursor: pointer;
    }
    button:hover { background: var(--leaf-dark); }
    .notice {
      border: 1px solid rgba(39, 100, 71, .18);
      background: rgba(39, 100, 71, .08);
      color: var(--leaf-dark);
      border-radius: 8px;
      padding: 12px 14px;
      font-size: 14px;
      line-height: 1.4;
      margin-bottom: 16px;
    }
    .notice.warn {
      border-color: rgba(173, 92, 27, .2);
      background: rgba(243, 204, 115, .18);
      color: #6c491a;
    }
    .dev-link {
      display: block;
      border: 1px solid rgba(39, 100, 71, .26);
      background: white;
      color: var(--leaf-dark);
      border-radius: 8px;
      padding: 12px 14px;
      margin: -4px 0 16px;
      text-align: center;
      text-decoration: none;
      font-size: 14px;
      font-weight: 800;
    }
    .dev-link:hover { border-color: rgba(39, 100, 71, .5); }
    .meta {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 1px;
      margin-top: 32px;
      max-width: 640px;
      border: 1px solid rgba(23, 32, 29, .08);
      background: rgba(23, 32, 29, .08);
    }
    .meta div {
      background: rgba(248,250,247,.74);
      padding: 18px;
      min-height: 92px;
    }
    .meta strong {
      display: block;
      font-size: 24px;
      margin-bottom: 4px;
    }
    .meta span {
      color: var(--muted);
      font-size: 14px;
      line-height: 1.35;
    }
    footer {
      padding: 20px clamp(20px, 5vw, 72px) 26px;
      color: rgba(255,255,255,.92);
      font-weight: 650;
      text-shadow: 0 1px 12px rgba(0,0,0,.28);
    }
    @media (max-width: 860px) {
      header { align-items: flex-start; }
      nav { display: none; }
      main {
        grid-template-columns: 1fr;
        align-items: start;
        padding-top: 12px;
      }
      .copy { padding-bottom: 0; }
      .login { max-width: 560px; }
      .meta { display: none; }
      h1 { font-size: clamp(40px, 13vw, 64px); }
      .lead { font-size: 17px; }
    }
  </style>
</head>
<body>
  <section class="hero">
    <header>
      <a class="brand" href="/" aria-label="WEG Portal Startseite"><span class="mark">WEG</span><span>{{.Tenant.Address}}</span></a>
      <nav aria-label="Portalbereiche">
        <span>Aushang</span>
        <span>Dokumente</span>
        <span>Anliegen</span>
      </nav>
    </header>
    <main>
      <div class="copy">
        <div class="eyebrow">WEG Portal</div>
        <h1>Alles rund um unser gemeinsames Haus.</h1>
        <p class="lead">Alle Neuigkeiten, Unterlagen, Anliegen und Abstimmungen, erreichbar per persönlichem E-Mail-Zugang.</p>
        <div class="meta" aria-label="Portalüberblick">
          <div><strong>12</strong><span>Wohneinheiten, ein gemeinsamer digitaler Eingang.</span></div>
          <div><strong>15 min</strong><span>Gültigkeit für jeden E-Mail-Anmeldelink.</span></div>
          <div><strong>1</strong><span>Ort für Aushang, Dokumente und Kontakt.</span></div>
        </div>
      </div>
      <section class="login" aria-label="Anmeldung">
        <h2>Anmelden</h2>
        <p>Geben Sie Ihre E-Mail-Adresse ein. Wenn sie eingeladen ist, schicken wir einen einmaligen Anmeldelink.</p>
        {{if .Sent}}
          <div class="notice">Wenn die Adresse eingeladen ist, wurde ein Link verschickt. Bitte Posteingang prüfen.</div>
          {{if not .MailConfigured}}<div class="notice warn">Mailversand ist lokal noch nicht konfiguriert. In Produktion kommt SMTP aus agenix.</div>{{end}}
          {{if .DevLoginLink}}<a class="dev-link" href="{{.DevLoginLink}}">Lokalen Dev-Login öffnen</a>{{end}}
        {{end}}
        {{if .Denied}}<div class="notice warn">Diese Adresse ist noch nicht eingeladen.</div>{{end}}
        <form method="post" action="/auth/request">
          <label for="email">E-Mail-Adresse</label>
          <input id="email" name="email" type="email" inputmode="email" autocomplete="email" required placeholder="name@example.com">
          <button type="submit">Anmeldelink senden</button>
        </form>
      </section>
    </main>
    <footer>{{.Tenant.Address}} · Privat für die Hausgemeinschaft</footer>
  </section>
</body>
</html>
{{end}}

{{define "portal"}}
<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
    :root {
      color-scheme: light;
      --ink: #17201d;
      --muted: #60716a;
      --line: #dfe7df;
      --paper: #f7faf6;
      --leaf: #276447;
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    * { box-sizing: border-box; }
    body { margin: 0; background: var(--paper); color: var(--ink); }
    header {
      position: sticky;
      top: 0;
      background: rgba(247,250,246,.9);
      backdrop-filter: blur(16px);
      border-bottom: 1px solid var(--line);
      padding: 16px clamp(18px, 4vw, 52px);
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 16px;
      z-index: 1;
    }
    .brand {
      color: var(--ink);
      font-weight: 800;
      text-decoration: none;
    }
    .user { color: var(--muted); font-size: 14px; }
    .top-actions {
      display: flex;
      align-items: center;
      gap: 10px;
      flex-wrap: wrap;
    }
    .link-button {
      border: 1px solid var(--line);
      background: white;
      border-radius: 8px;
      color: var(--ink);
      display: inline-flex;
      align-items: center;
      min-height: 41px;
      padding: 9px 12px;
      text-decoration: none;
      font-weight: 700;
    }
    form { margin: 0; }
    button {
      border: 1px solid var(--line);
      background: white;
      border-radius: 8px;
      padding: 10px 12px;
      font: inherit;
      font-weight: 700;
      cursor: pointer;
    }
    main { padding: 34px clamp(18px, 4vw, 52px) 60px; }
    .grid {
      display: grid;
      grid-template-columns: 1.4fr .9fr;
      gap: 24px;
      align-items: start;
    }
    section {
      background: white;
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 22px;
    }
    .wide { grid-column: 1 / -1; }
    h1 { margin: 0 0 18px; font-size: clamp(32px, 5vw, 56px); letter-spacing: 0; }
    h2 { margin: 0 0 14px; font-size: 20px; letter-spacing: 0; }
    .muted { color: var(--muted); line-height: 1.5; }
    .list { display: grid; gap: 12px; margin-top: 16px; }
    .item {
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 14px;
      display: grid;
      gap: 4px;
      color: inherit;
      text-decoration: none;
    }
    a.item:hover { border-color: rgba(39, 100, 71, .45); }
    .item strong { font-size: 16px; }
    .item span { color: var(--muted); font-size: 14px; line-height: 1.4; }
    .actions {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 12px;
      margin-top: 18px;
    }
    .action {
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 16px;
      min-height: 100px;
      background: #fbfdfb;
    }
    @media (max-width: 860px) {
      header { align-items: flex-start; flex-direction: column; }
      .grid, .actions { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <header>
    <div>
      <a class="brand" href="/app">WEG Portal · {{.Tenant.Address}}</a>
      <div class="user">Angemeldet als {{.DisplayName}} · Rolle: {{.Role}}</div>
    </div>
    <div class="top-actions">
      {{if .CanSeeParking}}<a class="link-button" href="/app/parking">Parkplatznutzung</a>{{end}}
      {{if .IsAdmin}}<a class="link-button" href="/app/settings/users">Benutzer & Rechte</a>{{end}}
      <form method="post" action="/auth/logout"><button type="submit">Abmelden</button></form>
    </div>
  </header>
  <main>
    <h1>Hausüberblick</h1>
    <div class="grid">
      <section>
        <h2>Aktueller Aushang</h2>
        <p class="muted">Hier landen später offizielle Informationen der Hausgemeinschaft, Termine und kurze Updates.</p>
        <div class="list">
          <div class="item"><strong>Willkommen im Prototyp</strong><span>Der Zugang funktioniert bereits per E-Mail-Link. Inhalte sind noch Beispielmodule.</span></div>
          <div class="item"><strong>Nächste Ausbaustufe</strong><span>Einladungen, Bewohnerliste, Dokumentenablage und Anliegenverwaltung.</span></div>
        </div>
      </section>
      <section>
        <h2>Schnellzugriff</h2>
        <div class="list">
          <div class="item"><strong>Dokumente</strong><span>Protokolle, Abrechnungen, Regeln und Pläne.</span></div>
          <div class="item"><strong>Anliegen</strong><span>Reparaturen, Fragen, Vorschläge und Rückmeldungen.</span></div>
          <div class="item"><strong>Abstimmungen</strong><span>Vorbereitete Entscheidungen für die Hausgemeinschaft.</span></div>
          {{if .CanSeeParking}}<a class="item" href="/app/parking"><strong>Parkplatznutzung</strong><span>Privater Bereich für die abgestimmte Nutzung des Stellplatzes.</span></a>{{end}}
          {{if .IsAdmin}}<a class="item" href="/app/settings/users"><strong>Benutzer & Rechte</strong><span>Einladungen, Rollen und Zugriff der Hausgemeinschaft verwalten.</span></a>{{end}}
        </div>
      </section>
    </div>
    <div class="actions">
      <div class="action"><strong>Einladungssystem</strong><p class="muted">Zugriff nur für freigeschaltete E-Mail-Adressen.</p></div>
      <div class="action"><strong>E-Mail-Faktor</strong><p class="muted">Einmalige Links, 15 Minuten gültig.</p></div>
      <div class="action"><strong>Web-App</strong><p class="muted">Responsive, ohne Installation, bereit für Homescreen-Pinning.</p></div>
    </div>
  </main>
</body>
</html>
{{end}}

{{define "parking"}}
<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
    :root {
      color-scheme: light;
      --ink: #17201d;
      --muted: #60716a;
      --line: #dfe7df;
      --paper: #f7faf6;
      --leaf: #276447;
      --soft: #eef5ee;
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    * { box-sizing: border-box; }
    body { margin: 0; background: var(--paper); color: var(--ink); }
    header {
      position: sticky;
      top: 0;
      background: rgba(247,250,246,.92);
      backdrop-filter: blur(16px);
      border-bottom: 1px solid var(--line);
      padding: 16px clamp(18px, 4vw, 52px);
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 16px;
      z-index: 1;
    }
    .brand {
      color: var(--ink);
      font-weight: 800;
      text-decoration: none;
    }
    .user { color: var(--muted); font-size: 14px; }
    .top-actions {
      display: flex;
      align-items: center;
      gap: 10px;
      flex-wrap: wrap;
    }
    form { margin: 0; }
    button, .link-button {
      border: 1px solid var(--line);
      background: white;
      border-radius: 8px;
      color: var(--ink);
      min-height: 41px;
      padding: 9px 12px;
      font: inherit;
      font-weight: 700;
      text-decoration: none;
      cursor: pointer;
    }
    main {
      padding: 34px clamp(18px, 4vw, 52px) 60px;
      display: grid;
      gap: 22px;
    }
    h1 { margin: 0; font-size: clamp(34px, 5vw, 58px); letter-spacing: 0; }
    h2 { margin: 0 0 14px; font-size: 20px; letter-spacing: 0; }
    p { margin: 0; }
    .muted { color: var(--muted); line-height: 1.5; }
    .layout {
      display: grid;
      grid-template-columns: minmax(0, 1.1fr) minmax(280px, .9fr);
      gap: 22px;
      align-items: start;
    }
    section {
      background: white;
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 22px;
    }
    .list {
      display: grid;
      gap: 12px;
      margin-top: 16px;
    }
    .item {
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 14px;
      display: grid;
      gap: 4px;
      background: #fbfdfb;
    }
    .item strong { font-size: 16px; }
    .item span { color: var(--muted); font-size: 14px; line-height: 1.4; }
    .note {
      border: 1px solid rgba(39, 100, 71, .18);
      background: var(--soft);
      color: #244333;
      border-radius: 8px;
      padding: 16px;
      line-height: 1.5;
    }
    .metric-grid {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 12px;
      margin-top: 16px;
    }
    .metric {
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 16px;
      background: #fbfdfb;
      min-height: 118px;
    }
    .metric span {
      display: block;
      color: var(--muted);
      font-size: 12px;
      font-weight: 800;
      text-transform: uppercase;
      letter-spacing: .08em;
      margin-bottom: 10px;
    }
    .metric strong {
      display: block;
      font-size: clamp(22px, 4vw, 34px);
      line-height: 1.05;
      margin-bottom: 8px;
    }
    .metric code, .entity-ref code {
      color: var(--muted);
      font-size: 12px;
      white-space: normal;
      overflow-wrap: anywhere;
    }
    .status {
      display: inline-flex;
      align-items: center;
      border-radius: 999px;
      background: var(--soft);
      color: var(--leaf);
      min-height: 30px;
      padding: 5px 10px;
      font-size: 13px;
      font-weight: 800;
      margin-top: 12px;
    }
    .settings-form {
      display: grid;
      grid-template-columns: minmax(180px, 1fr) auto;
      gap: 10px;
      margin: 16px 0;
      align-items: end;
    }
    label {
      display: grid;
      gap: 6px;
      color: var(--muted);
      font-size: 13px;
      font-weight: 800;
    }
    input {
      border: 1px solid var(--line);
      border-radius: 8px;
      min-height: 41px;
      padding: 9px 10px;
      font: inherit;
      color: var(--ink);
      background: #fff;
    }
    .table-wrap { overflow-x: auto; margin-top: 16px; }
    table { width: 100%; border-collapse: collapse; min-width: 760px; }
    th, td {
      border-bottom: 1px solid var(--line);
      padding: 11px 8px;
      text-align: left;
      vertical-align: middle;
      font-size: 14px;
    }
    th {
      color: var(--muted);
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: .06em;
    }
    .amount { font-weight: 800; }
    .pill {
      display: inline-flex;
      align-items: center;
      min-height: 28px;
      border-radius: 999px;
      padding: 4px 9px;
      font-size: 12px;
      font-weight: 800;
      background: #f7efe4;
      color: #744719;
      white-space: nowrap;
    }
    .pill.ok {
      background: var(--soft);
      color: var(--leaf);
    }
    .mini {
      color: var(--muted);
      font-size: 12px;
      display: block;
      margin-top: 3px;
    }
    .chart {
      display: grid;
      gap: 10px;
      margin-top: 16px;
    }
    .chart-row {
      display: grid;
      grid-template-columns: 120px minmax(120px, 1fr) 90px;
      gap: 10px;
      align-items: center;
      font-size: 13px;
    }
    .bar {
      height: 14px;
      border-radius: 999px;
      background: #e9eee8;
      overflow: hidden;
    }
    .bar span {
      display: block;
      height: 100%;
      border-radius: inherit;
      background: var(--leaf);
      min-width: 2px;
    }
    .plain-button {
      min-height: 34px;
      padding: 7px 10px;
      font-size: 12px;
    }
    @media (max-width: 860px) {
      header { align-items: flex-start; flex-direction: column; }
      .layout, .metric-grid, .settings-form { grid-template-columns: 1fr; }
      .chart-row { grid-template-columns: 90px minmax(100px, 1fr) 72px; }
    }
  </style>
</head>
<body>
  <header>
    <div>
      <a class="brand" href="/app">WEG Portal · {{.Tenant.Address}}</a>
      <div class="user">Angemeldet als {{.DisplayName}} · Rolle: {{.Role}}</div>
    </div>
    <div class="top-actions">
      <a class="link-button" href="/app">Zurück</a>
      <form method="post" action="/auth/logout"><button type="submit">Abmelden</button></form>
    </div>
  </header>
  <main>
    <div>
      <h1>Parkplatznutzung</h1>
      <p class="muted">Privater Bereich für die persönlich abgestimmte Nutzung des Stellplatzes.</p>
    </div>
    <div class="layout">
      <section>
        <h2>Vereinbarung</h2>
        <div class="list">
          <div class="item"><strong>Rahmen</strong><span>Nutzung nur nach persönlicher Absprache zwischen den berechtigten Personen.</span></div>
          <div class="item"><strong>Dokumentation</strong><span>Hier können später kurze Notizen, Zeiträume und Absprachen hinterlegt werden.</span></div>
          <div class="item"><strong>Ladestation</strong><span>Nutzung der vorhandenen Infrastruktur nach vorheriger Abstimmung.</span></div>
        </div>
      </section>
      <section>
        <h2>Zähler & Preis</h2>
        {{if .Telemetry.Connected}}
          <p class="muted">{{.Telemetry.Message}}</p>
          <div class="metric-grid">
            {{range .Telemetry.Metrics}}
              <div class="metric">
                <span>{{.Label}}</span>
                <strong>{{.Value}}</strong>
                <code>{{.Detail}}</code>
              </div>
            {{end}}
          </div>
        {{else}}
          <p class="note">{{.Telemetry.Message}}</p>
          <div class="list">
            {{range .Telemetry.Entities}}
              <div class="item entity-ref"><strong>{{.Label}}</strong><code>{{.EntityID}}</code></div>
            {{end}}
          </div>
        {{end}}
        {{if .Telemetry.Configured}}<div class="status">Home Assistant vorbereitet</div>{{end}}
      </section>
      <section class="wide">
        <h2>Monatsabrechnung</h2>
        <p class="muted">{{.Accounting.Message}}</p>
        {{if .IsAdmin}}
          <form class="settings-form" method="post" action="/app/parking/settings">
            <label>Netzbetreibergebühren je kWh
              <input type="number" min="0" max="5" step="0.001" name="grid_fee_eur_per_kwh" value="{{.Accounting.GridFeeValue}}">
            </label>
            <button type="submit">Speichern</button>
          </form>
        {{else}}
          <p class="mini">Aktuelles Delta: {{.Accounting.GridFeeLabel}}</p>
        {{end}}
        {{if .Accounting.LastSampleLabel}}<p class="mini">Letzter Zählerwert: {{.Accounting.LastSampleLabel}}</p>{{end}}
        {{if .Accounting.HasMonths}}
          <div class="chart">
            {{range .Accounting.Months}}
              <div class="chart-row">
                <strong>{{.MonthLabel}}</strong>
                <div class="bar"><span style="width: {{.ChartPercent}}%;"></span></div>
                <span class="amount">{{.TotalCost}}</span>
              </div>
            {{end}}
          </div>
          <div class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Monat</th>
                  <th>Zeitraum</th>
                  <th>Verbrauch</th>
                  <th>aWATTar + Delta</th>
                  <th>Strom</th>
                  <th>Delta</th>
                  <th>Summe</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {{range .Accounting.Months}}
                  <tr>
                    <td><strong>{{.MonthLabel}}</strong>{{if .Partial}}<span class="mini">Teilmonat</span>{{end}}</td>
                    <td>{{.PeriodLabel}}<span class="mini">{{.HourCount}} Stunden-Buckets</span></td>
                    <td>{{.KWh}}</td>
                    <td>{{.AveragePrice}}</td>
                    <td>{{.EnergyCost}}</td>
                    <td>{{.GridCost}}</td>
                    <td class="amount">{{.TotalCost}}</td>
                    <td>
                      <span class="pill {{if .Paid}}ok{{end}}">{{.PaidLabel}}</span>
                      {{if $.IsAdmin}}
                        <form method="post" action="/app/parking/month" style="margin-top: 8px;">
                          <input type="hidden" name="month" value="{{.Month}}">
                          <input type="hidden" name="paid" value="{{.TogglePaidValue}}">
                          <button class="plain-button" type="submit">{{.ToggleLabel}}</button>
                        </form>
                      {{end}}
                    </td>
                  </tr>
                {{end}}
              </tbody>
            </table>
          </div>
        {{else}}
          <p class="note">Noch keine Monatswerte. Sobald zwei Zählerstände und mindestens ein aWATTar-Preis vorliegen, erscheint hier die erste Abrechnung.</p>
        {{end}}
      </section>
      <section>
        <h2>Privatsphäre</h2>
        <p class="note">Dieser Bereich ist nicht Teil der allgemeinen Hausgemeinschaftsansicht. Der Zugriff wird serverseitig geprüft und ist nur für ausdrücklich berechtigte Benutzer sichtbar.</p>
      </section>
    </div>
  </main>
</body>
</html>
{{end}}

{{define "userSettings"}}
<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
    :root {
      color-scheme: light;
      --ink: #17201d;
      --muted: #60716a;
      --line: #dfe7df;
      --paper: #f7faf6;
      --leaf: #276447;
      --soft: #eef5ee;
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    * { box-sizing: border-box; }
    body { margin: 0; background: var(--paper); color: var(--ink); }
    header {
      position: sticky;
      top: 0;
      background: rgba(247,250,246,.92);
      backdrop-filter: blur(16px);
      border-bottom: 1px solid var(--line);
      padding: 16px clamp(18px, 4vw, 52px);
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 16px;
      z-index: 1;
    }
    .brand {
      color: var(--ink);
      font-weight: 800;
      text-decoration: none;
    }
    .user { color: var(--muted); font-size: 14px; }
    .top-actions {
      display: flex;
      align-items: center;
      gap: 10px;
      flex-wrap: wrap;
    }
    form { margin: 0; }
    button, .link-button {
      border: 1px solid var(--line);
      background: white;
      border-radius: 8px;
      color: var(--ink);
      min-height: 41px;
      padding: 9px 12px;
      font: inherit;
      font-weight: 700;
      text-decoration: none;
      cursor: pointer;
    }
    button:disabled {
      color: var(--muted);
      cursor: default;
      background: #f4f7f3;
    }
    main {
      padding: 34px clamp(18px, 4vw, 52px) 60px;
      display: grid;
      gap: 22px;
    }
    h1 { margin: 0; font-size: clamp(32px, 5vw, 54px); letter-spacing: 0; }
    h2 { margin: 0 0 14px; font-size: 20px; letter-spacing: 0; }
    p { margin: 0; }
    .muted { color: var(--muted); line-height: 1.5; }
    .layout {
      display: grid;
      grid-template-columns: minmax(0, 1.35fr) minmax(280px, .65fr);
      gap: 22px;
      align-items: start;
    }
    section {
      background: white;
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 22px;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      margin-top: 16px;
      font-size: 15px;
    }
    th, td {
      border-bottom: 1px solid var(--line);
      padding: 13px 10px;
      text-align: left;
      vertical-align: middle;
    }
    th {
      color: var(--muted);
      font-size: 12px;
      font-weight: 800;
      text-transform: uppercase;
      letter-spacing: .08em;
    }
    td:first-child, th:first-child { padding-left: 0; }
    td:last-child, th:last-child { padding-right: 0; }
    .pill {
      display: inline-flex;
      align-items: center;
      border-radius: 999px;
      background: var(--soft);
      color: var(--leaf);
      min-height: 28px;
      padding: 5px 10px;
      font-size: 13px;
      font-weight: 800;
    }
    .permissions {
      display: grid;
      gap: 10px;
      margin-top: 14px;
    }
    .permission {
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 14px;
      background: #fbfdfb;
    }
    .permission strong { display: block; margin-bottom: 4px; }
    .invite {
      display: grid;
      gap: 10px;
      margin-top: 14px;
    }
    input, select {
      width: 100%;
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 12px;
      font: inherit;
      background: white;
      color: var(--ink);
    }
    input:disabled, select:disabled { color: var(--muted); background: #f4f7f3; }
    @media (max-width: 860px) {
      header { align-items: flex-start; flex-direction: column; }
      .layout { grid-template-columns: 1fr; }
      table { display: block; overflow-x: auto; white-space: nowrap; }
    }
  </style>
</head>
<body>
  <header>
    <div>
      <a class="brand" href="/app">WEG Portal · {{.Tenant.Address}}</a>
      <div class="user">Angemeldet als {{.DisplayName}} · Rolle: {{.Role}}</div>
    </div>
    <div class="top-actions">
      <a class="link-button" href="/app">Zurück</a>
      <form method="post" action="/auth/logout"><button type="submit">Abmelden</button></form>
    </div>
  </header>
  <main>
    <div>
      <h1>Benutzer & Rechte</h1>
      <p class="muted">Lokale Verwaltung der eingeladenen E-Mail-Adressen und ihrer Rollen.</p>
    </div>
    <div class="layout">
      <section>
        <h2>Zugänge</h2>
        <p class="muted">Aktuell kommt diese Liste aus der lokalen Umgebungskonfiguration.</p>
        <table aria-label="Benutzerliste">
          <thead>
            <tr>
              <th>Name</th>
              <th>Titel</th>
              <th>Vorname</th>
              <th>Nachname</th>
              <th>E-Mail</th>
              <th>Rolle</th>
              <th>Rechte</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {{range .Users}}
            <tr>
              <td>{{.DisplayName}}</td>
              <td>{{.Title}}</td>
              <td>{{.FirstName}}</td>
              <td>{{.LastName}}</td>
              <td>{{.Email}}</td>
              <td><span class="pill">{{.Role}}</span></td>
              <td>{{.PermissionLabel}}</td>
              <td>{{.Status}}</td>
            </tr>
            {{end}}
          </tbody>
        </table>
      </section>
      <div class="permissions">
        <section>
          <h2>Rechtegruppen</h2>
          <div class="permissions">
            <div class="permission"><strong>Admin</strong><p class="muted">Zugänge verwalten, Rollen setzen und Portalbereiche vorbereiten.</p></div>
            <div class="permission"><strong>Bewohner</strong><p class="muted">Aushang, Dokumente, Anliegen und Abstimmungen nutzen.</p></div>
            <div class="permission"><strong>Parkplatznutzung</strong><p class="muted">Separates Sonderrecht für einen privaten, abgestimmten Bereich.</p></div>
            <div class="permission"><strong>Beirat</strong><p class="muted">Vorgemerkt für spätere Moderation und Freigaben.</p></div>
          </div>
        </section>
        <section>
          <h2>Einladung vorbereiten</h2>
          <p class="muted">Noch ohne Speichern, damit der lokale Prototyp keine falschen Versprechen macht.</p>
          <form class="invite">
            <input type="text" placeholder="Titel" disabled>
            <input type="text" placeholder="Vorname" disabled>
            <input type="text" placeholder="Nachname" disabled>
            <input type="email" placeholder="name@example.com" disabled>
            <select disabled>
              <option>Bewohner</option>
              <option>Admin</option>
              <option>Beirat</option>
            </select>
            <button type="button" disabled>Einladung vorbereiten</button>
          </form>
        </section>
      </div>
    </div>
  </main>
</body>
</html>
{{end}}
`

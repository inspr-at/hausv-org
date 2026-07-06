package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
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

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/websocket"
	"golang.org/x/oauth2"
)

//go:embed assets/*
var assets embed.FS

const (
	roleAdmin         = "Admin"
	roleManager       = "Verwalter"
	roleOwner         = "Eigentümer"
	roleRenter        = "Mieter"
	roleBeirat        = "Beirat"
	roleResident      = "Bewohner"
	permissionParking = "parking"
	authMethodEmail   = "email"
	authMethodOIDC    = "oidc"
)

type capability string

const (
	capabilityPlatformAdmin       capability = "platform-admin"
	capabilityManageUsers         capability = "manage-users"
	capabilityManageParking       capability = "manage-parking"
	capabilityManageAnnouncements capability = "manage-announcements"
	capabilityOwnerDocuments      capability = "owner-documents"
	capabilityVote                capability = "vote"
	capabilityOversight           capability = "oversight"
)

var (
	appVersion = "0.1.0"
	gitCommit  = "dev"
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
	sessionTTL            time.Duration
	tokens                *tokenStore
	sessions              *sessionStore
	oidc                  *oidcLogin
	oidcFlows             *oidcFlowStore
	mailer                mailer
	templates             *template.Template
	announcementStore     *announcementStore
	announcementReadStore *announcementReadStore
	inviteStore           *inviteStore
	activityStore         *activityStore
	parkingStore          *parkingStore
	parkingSampleInterval time.Duration
	parkingHistoryStart   time.Time
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
	mu      sync.Mutex
	secret  []byte
	revoked map[string]time.Time
}

type session struct {
	Email      string `json:"email"`
	TenantSlug string `json:"tenant_slug"`
	AuthMethod string `json:"auth_method"`
	ExpiresAt  int64  `json:"expires_at"`
}

type oidcLogin struct {
	mu           sync.Mutex
	providerName string
	issuer       string
	clientID     string
	clientSecret string
	redirectURL  string
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
}

type oidcFlowStore struct {
	mu    sync.Mutex
	items map[string]oidcFlow
}

type oidcFlow struct {
	tenantSlug   string
	nonce        string
	codeVerifier string
	expiresAt    time.Time
	used         bool
}

type oidcUserClaims struct {
	Email         string `json:"email"`
	EmailVerified *bool  `json:"email_verified"`
}

type mailer interface {
	SendMagicLink(to string, link string) error
	SendInvite(to string, loginURL string, address string) error
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

type haStatistic struct {
	Start json.RawMessage `json:"start"`
	End   json.RawMessage `json:"end"`
	State *float64        `json:"state"`
	Sum   *float64        `json:"sum"`
	Mean  *float64        `json:"mean"`
	Min   *float64        `json:"min"`
	Max   *float64        `json:"max"`
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

type announcementStore struct {
	mu   sync.Mutex
	path string
	data announcementStoreData
}

type announcementStoreData struct {
	Announcements []announcement `json:"announcements"`
}

type announcementReadStore struct {
	mu   sync.Mutex
	path string
	data announcementReadStoreData
}

type announcementReadStoreData struct {
	Seen map[string]map[string]time.Time `json:"seen"`
}

type announcement struct {
	ID          string     `json:"id"`
	TenantSlug  string     `json:"tenant"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	Category    string     `json:"category"`
	Pinned      bool       `json:"pinned"`
	PublishedAt time.Time  `json:"published_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	AuthorEmail string     `json:"author_email"`
	AuthorName  string     `json:"author_name"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type announcementView struct {
	ID                 string
	Title              string
	Body               string
	BodyHTML           template.HTML
	Category           string
	CategoryClass      string
	Pinned             bool
	PinnedChecked      bool
	PublishedAt        string
	PublishedAtInput   string
	ExpiresAt          string
	ExpiresAtInput     string
	HasExpiresAt       bool
	Author             string
	Status             string
	Published          bool
	Expired            bool
	Unread             bool
	EditDialogID       string
	DeleteConfirmLabel string
}

type announcementFilterView struct {
	Label  string
	URL    string
	Active bool
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
	DetailPath      string
	PeriodLabel     string
	KWh             string
	EnergyCost      string
	GridCost        string
	TotalCost       string
	AverageAwattar  string
	EffectivePrice  string
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

type parkingMonthDetailView struct {
	Month           string
	MonthLabel      string
	BackPath        string
	Message         string
	GridFeeLabel    string
	LastSampleLabel string
	Summary         parkingMonthView
	Hours           []parkingHourView
	HasHours        bool
}

type parkingHourView struct {
	AtLabel             string
	AtTitle             string
	KWh                 string
	KWhTitle            string
	AverageAwattar      string
	AverageAwattarTitle string
	EnergyCost          string
	EnergyCostTitle     string
	GridCost            string
	GridCostTitle       string
	TotalCost           string
	TotalCostTitle      string
	WeightTitle         string
	ChartPercent        int
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		target := "http://127.0.0.1:8080/healthz"
		if len(os.Args) > 2 {
			target = os.Args[2]
		}
		if err := runHealthcheck(target); err != nil {
			log.Printf("healthcheck failed: %v", err)
			os.Exit(1)
		}
		return
	}

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
	mux.HandleFunc("GET /auth/oidc/start", a.startOIDCLogin)
	mux.HandleFunc("GET /auth/oidc/callback", a.finishOIDCLogin)
	mux.HandleFunc("POST /auth/logout", a.logout)
	mux.HandleFunc("GET /app", a.portal)
	mux.HandleFunc("GET /app/announcements", a.announcements)
	mux.HandleFunc("POST /app/announcements", a.createAnnouncement)
	mux.HandleFunc("POST /app/announcements/edit", a.editAnnouncement)
	mux.HandleFunc("POST /app/announcements/delete", a.deleteAnnouncement)
	mux.HandleFunc("GET /app/parking", a.parking)
	mux.HandleFunc("GET /app/parking/settings", a.parkingSettings)
	mux.HandleFunc("GET /app/parking/month/{month}", a.parkingMonth)
	mux.HandleFunc("POST /app/parking/settings", a.updateParkingSettings)
	mux.HandleFunc("POST /app/parking/month", a.updateParkingMonth)
	mux.HandleFunc("GET /app/settings", a.settingsHub)
	mux.HandleFunc("GET /app/settings/users", a.userSettings)
	mux.HandleFunc("POST /app/settings/users", a.createInvite)
	mux.HandleFunc("POST /app/settings/users/edit", a.editInvite)
	mux.HandleFunc("POST /app/settings/users/delete", a.deleteInvite)
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

func runHealthcheck(target string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}
	var payload struct {
		Service string `json:"service"`
		Status  string `json:"status"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 512)).Decode(&payload); err != nil {
		return fmt.Errorf("invalid health response: %w", err)
	}
	if payload.Service != "weg-portal" || payload.Status != "ok" {
		return fmt.Errorf("unexpected health response service=%q status=%q", payload.Service, payload.Status)
	}
	return nil
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
	oidcCtx, cancelOIDC := context.WithTimeout(context.Background(), 10*time.Second)
	oidcLogin, err := newOIDCLogin(
		oidcCtx,
		env("OIDC_ISSUER", ""),
		env("OIDC_CLIENT_ID", ""),
		strings.TrimSpace(os.Getenv("OIDC_CLIENT_SECRET")),
		env("OIDC_REDIRECT_URL", ""),
		env("OIDC_PROVIDER_NAME", "Zitadel"),
	)
	cancelOIDC()
	if err != nil {
		return nil, err
	}
	if publicURL && !mailTransport.Configured() && !oidcLogin.Configured() {
		return nil, fmt.Errorf("SMTP or OIDC login is required when BASE_URL is public")
	}

	announcementDataPath := env("ANNOUNCE_DATA_PATH", "tmp/announcements.json")
	announcements, err := newAnnouncementStore(announcementDataPath)
	if err != nil {
		return nil, err
	}
	announcementReadDataPath := env("ANNOUNCE_READ_DATA_PATH", "tmp/announcement_reads.json")
	announcementReads, err := newAnnouncementReadStore(announcementReadDataPath)
	if err != nil {
		return nil, err
	}
	inviteDataPath := env("INVITE_DATA_PATH", "tmp/invites.json")
	invites, err := newInviteStore(inviteDataPath)
	if err != nil {
		return nil, err
	}
	activityDataPath := env("ACTIVITY_DATA_PATH", "tmp/activity.json")
	activity, err := newActivityStore(activityDataPath)
	if err != nil {
		return nil, err
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
	parkingHistoryStart, err := parseHistoryStart(env("PARKING_HISTORY_START", ""), time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid PARKING_HISTORY_START")
	}
	sessionTTL, err := parseDuration(env("SESSION_TTL", "720h"))
	if err != nil || sessionTTL <= 0 {
		return nil, fmt.Errorf("invalid SESSION_TTL")
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
		sessionTTL:    sessionTTL,
		tokens: &tokenStore{
			secret: secret,
			items:  map[string]loginToken{},
		},
		sessions:              newSessionStore(secret),
		oidc:                  oidcLogin,
		oidcFlows:             &oidcFlowStore{items: map[string]oidcFlow{}},
		mailer:                mailTransport,
		templates:             tmpl,
		announcementStore:     announcements,
		announcementReadStore: announcementReads,
		inviteStore:           invites,
		activityStore:         activity,
		parkingStore:          parkingStore,
		parkingSampleInterval: parkingSampleInterval,
		parkingHistoryStart:   parkingHistoryStart,
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
		"Title":               "WEG Portal " + tenant.Address,
		"Tenant":              tenant,
		"Email":               email,
		"Sent":                r.URL.Query().Get("sent") == "1",
		"MailConfigured":      a.mailer.Configured(),
		"DevLoginLink":        "",
		"Denied":              r.URL.Query().Get("denied") == "1",
		"OIDCConfigured":      a.oidc.Configured(),
		"OIDCProviderName":    a.oidc.ProviderName(),
		"EmailLoginAvailable": a.emailLoginAvailable(),
	})
}

func (a *app) requestLogin(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	if !a.emailLoginAvailable() {
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	email := normalizeEmail(r.FormValue("email"))
	if _, err := mail.ParseAddress(email); err != nil {
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}
	if !a.isAllowed(email, tenant.Slug) || !a.isAuthMethodAllowed(email, tenant.Slug, authMethodEmail) {
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
			"Title":               "WEG Portal " + tenant.Address,
			"Tenant":              tenant,
			"Email":               email,
			"Sent":                true,
			"MailConfigured":      false,
			"DevLoginLink":        link,
			"Denied":              false,
			"OIDCConfigured":      a.oidc.Configured(),
			"OIDCProviderName":    a.oidc.ProviderName(),
			"EmailLoginAvailable": a.emailLoginAvailable(),
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

	if err := a.startSession(w, email, tenantSlug, authMethodEmail); err != nil {
		http.Error(w, "Could not create session", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/app", http.StatusSeeOther)
}

func (a *app) startOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if !a.oidc.Configured() {
		http.NotFound(w, r)
		return
	}
	tenant := a.tenantForRequest(r)
	if err := a.oidc.EnsureProvider(r.Context()); err != nil {
		log.Printf("oidc discovery failed during login start: %v", err)
		http.Error(w, "SSO ist gerade nicht erreichbar. Bitte später erneut versuchen oder den E-Mail-Link verwenden.", http.StatusServiceUnavailable)
		return
	}
	state, err := randomToken(32)
	if err != nil {
		http.Error(w, "Could not start SSO login", http.StatusInternalServerError)
		return
	}
	nonce, err := randomToken(32)
	if err != nil {
		http.Error(w, "Could not start SSO login", http.StatusInternalServerError)
		return
	}
	codeVerifier, err := randomToken(32)
	if err != nil {
		http.Error(w, "Could not start SSO login", http.StatusInternalServerError)
		return
	}
	a.oidcFlows.Put(state, oidcFlow{
		tenantSlug:   tenant.Slug,
		nonce:        nonce,
		codeVerifier: codeVerifier,
	}, 10*time.Minute)

	redirectURL := a.oidc.RedirectURL(r, tenant, a.publicBaseURL(r, tenant))
	oauthConfig := a.oidc.OAuthConfig(redirectURL)
	authCodeURL := oauthConfig.AuthCodeURL(
		state,
		oidc.Nonce(nonce),
		oauth2.SetAuthURLParam("code_challenge", pkceChallenge(codeVerifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	http.Redirect(w, r, authCodeURL, http.StatusSeeOther)
}

func (a *app) finishOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if !a.oidc.Configured() {
		http.NotFound(w, r)
		return
	}
	if err := a.oidc.EnsureProvider(r.Context()); err != nil {
		log.Printf("oidc discovery failed during login callback: %v", err)
		http.Error(w, "SSO ist gerade nicht erreichbar. Bitte später erneut versuchen.", http.StatusServiceUnavailable)
		return
	}
	if errText := strings.TrimSpace(r.URL.Query().Get("error")); errText != "" {
		log.Printf("oidc login failed: %s", errText)
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}
	flow, ok := a.oidcFlows.Consume(r.URL.Query().Get("state"))
	if !ok {
		http.Error(w, "Diese SSO-Anmeldung ist abgelaufen. Bitte erneut anmelden.", http.StatusUnauthorized)
		return
	}
	tenant, ok := a.tenantBySlug(flow.tenantSlug)
	if !ok {
		http.Error(w, "Unknown tenant", http.StatusUnauthorized)
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		http.Error(w, "SSO-Anmeldung ohne Code.", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	oauthConfig := a.oidc.OAuthConfig(a.oidc.RedirectURL(r, tenant, a.publicBaseURL(r, tenant)))
	token, err := oauthConfig.Exchange(
		ctx,
		code,
		oauth2.SetAuthURLParam("code_verifier", flow.codeVerifier),
	)
	if err != nil {
		log.Printf("oidc token exchange failed: %v", err)
		http.Error(w, "SSO-Anmeldung konnte nicht abgeschlossen werden.", http.StatusUnauthorized)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		log.Printf("oidc token exchange returned no id_token")
		http.Error(w, "SSO-Anmeldung konnte nicht geprüft werden.", http.StatusUnauthorized)
		return
	}
	idToken, err := a.oidc.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Printf("oidc id_token verification failed: %v", err)
		http.Error(w, "SSO-Anmeldung konnte nicht geprüft werden.", http.StatusUnauthorized)
		return
	}
	if idToken.Nonce != flow.nonce {
		log.Printf("oidc nonce mismatch")
		http.Error(w, "SSO-Anmeldung konnte nicht geprüft werden.", http.StatusUnauthorized)
		return
	}

	claims := oidcUserClaims{}
	if err := idToken.Claims(&claims); err != nil {
		log.Printf("oidc claims decode failed: %v", err)
		http.Error(w, "SSO-Anmeldung konnte nicht gelesen werden.", http.StatusUnauthorized)
		return
	}
	if claims.Email == "" || claims.EmailVerified == nil {
		userInfo, err := a.oidc.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
		if err != nil {
			log.Printf("oidc userinfo failed: %v", err)
		} else {
			var extra oidcUserClaims
			if err := userInfo.Claims(&extra); err == nil {
				claims.Merge(extra)
			}
		}
	}
	email := normalizeEmail(claims.Email)
	if email == "" || claims.EmailVerified == nil || !*claims.EmailVerified {
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}
	if !a.isAllowed(email, tenant.Slug) || !a.isAuthMethodAllowed(email, tenant.Slug, authMethodOIDC) {
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}
	if err := a.startSession(w, email, tenant.Slug, authMethodOIDC); err != nil {
		http.Error(w, "Could not create session", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/app", http.StatusSeeOther)
}

func (a *app) startSession(w http.ResponseWriter, email string, tenantSlug string, authMethod string) error {
	token, expiresAt, err := a.sessions.Put(email, tenantSlug, authMethod, a.sessionTTL)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "weg_session",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   a.sessionSecure,
		SameSite: http.SameSiteLaxMode,
	})
	if a.activityStore != nil {
		if err := a.activityStore.Touch(email, time.Now(), authMethod); err != nil {
			log.Printf("activity record failed for %s: %v", redactedEmail(email), err)
		}
	}
	return nil
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

func (a *app) announcements(w http.ResponseWriter, r *http.Request) {
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
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	canManage := canManageAnnouncements(role)
	now := time.Now()
	selectedCategory := selectedAnnouncementCategory(r.URL.Query().Get("category"))
	searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	lastSeen := time.Time{}
	if a.announcementReadStore != nil {
		lastSeen = a.announcementReadStore.LastSeen(tenant.Slug, email)
	}
	archive := []announcement{}
	filtered := []announcement{}
	all := []announcementView{}
	if a.announcementStore != nil {
		archive = a.announcementStore.Archive(tenant.Slug, now)
		filtered = filterAnnouncements(archive, selectedCategory, searchQuery)
		if canManage {
			all = announcementViewsWithReadState(a.announcementStore.ListTenant(tenant.Slug), now, true, lastSeen)
		}
	}
	a.render(w, "announcements", map[string]any{
		"Title":                  "Aushang",
		"Tenant":                 tenant,
		"Email":                  email,
		"DisplayName":            profile.DisplayName(),
		"Initials":               profile.Initials(),
		"Role":                   role,
		"IsAdmin":                isAdmin,
		"CanSeeParking":          isAdmin || profile.HasPermission(permissionParking),
		"CanManageAnnouncements": canManage,
		"ActivePage":             "announcements",
		"Announcements":          announcementViewsWithReadState(filtered, now, true, lastSeen),
		"HasAnnouncements":       len(filtered) > 0,
		"HasAnyAnnouncements":    len(archive) > 0,
		"AllAnnouncements":       all,
		"HasAllAnnouncements":    len(all) > 0,
		"AnnounceMsg":            announcementMessage(r.URL.Query().Get("announce")),
		"NowInput":               now.In(time.Local).Format("2006-01-02T15:04"),
		"SearchQuery":            searchQuery,
		"SelectedCategory":       selectedCategory,
		"CategoryFilters":        announcementFilterViews(searchQuery, selectedCategory),
		"UnreadAnnouncements":    0,
		"HasUnreadAnnouncements": false,
	})
	if a.announcementReadStore != nil {
		if err := a.announcementReadStore.MarkSeen(tenant.Slug, email, now); err != nil {
			log.Printf("announcement read mark failed for %s/%s: %v", tenant.Slug, redactedEmail(email), err)
		}
	}
}

func (a *app) createAnnouncement(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !canManageAnnouncements(role) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	profile := a.profileFor(email)
	item, err := announcementFromForm(r, tenant.Slug, profile, time.Now())
	if err != nil {
		http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
		return
	}
	if _, err := a.announcementStore.Create(item); err != nil {
		log.Printf("announcement create failed for %s: %v", tenant.Slug, err)
		http.Redirect(w, r, "/app/announcements?announce=error", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/announcements?announce=created", http.StatusSeeOther)
}

func (a *app) editAnnouncement(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !canManageAnnouncements(role) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	profile := a.profileFor(email)
	item, err := announcementFromForm(r, tenant.Slug, profile, time.Now())
	if err != nil {
		http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
		return
	}
	ok, err = a.announcementStore.Update(id, item)
	if err != nil {
		log.Printf("announcement update failed for %s: %v", tenant.Slug, err)
		http.Redirect(w, r, "/app/announcements?announce=error", http.StatusSeeOther)
		return
	}
	if !ok {
		http.Redirect(w, r, "/app/announcements?announce=missing", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/announcements?announce=updated", http.StatusSeeOther)
}

func (a *app) deleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	_, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !canManageAnnouncements(role) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	removed, err := a.announcementStore.Delete(tenant.Slug, strings.TrimSpace(r.FormValue("id")))
	if err != nil {
		log.Printf("announcement delete failed for %s: %v", tenant.Slug, err)
		http.Redirect(w, r, "/app/announcements?announce=error", http.StatusSeeOther)
		return
	}
	if !removed {
		http.Redirect(w, r, "/app/announcements?announce=missing", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/announcements?announce=deleted", http.StatusSeeOther)
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
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	canManage := canManageAnnouncements(role)
	announcements := []announcementView{}
	now := time.Now()
	lastSeen := time.Time{}
	if a.announcementReadStore != nil {
		lastSeen = a.announcementReadStore.LastSeen(tenant.Slug, email)
	}
	if a.announcementStore != nil {
		announcements = announcementViewsWithReadState(a.announcementStore.Visible(tenant.Slug, now), now, false, lastSeen)
		if len(announcements) > 3 {
			announcements = announcements[:3]
		}
	}
	a.render(w, "portal", map[string]any{
		"Title":                  "WEG Portal",
		"Tenant":                 tenant,
		"Email":                  email,
		"DisplayName":            profile.DisplayName(),
		"Initials":               profile.Initials(),
		"Role":                   role,
		"IsAdmin":                isAdmin,
		"CanSeeParking":          isAdmin || profile.HasPermission(permissionParking),
		"CanManageAnnouncements": canManage,
		"ActivePage":             "home",
		"Announcements":          announcements,
		"HasAnnouncements":       len(announcements) > 0,
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
	if !hasCapability(role, capabilityPlatformAdmin) && !profile.HasPermission(permissionParking) {
		http.NotFound(w, r)
		return
	}
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	telemetry := a.parkingTelemetry(r.Context(), tenant)
	a.render(w, "parking", map[string]any{
		"Title":         "Parkplatznutzung",
		"Tenant":        tenant,
		"Email":         email,
		"DisplayName":   profile.DisplayName(),
		"Initials":      profile.Initials(),
		"Role":          role,
		"IsAdmin":       isAdmin,
		"CanSeeParking": true,
		"ActivePage":    "parking",
		"Telemetry":     telemetry,
		"Accounting":    a.parkingAccounting(r.Context(), tenant),
	})
}

func (a *app) parkingSettings(w http.ResponseWriter, r *http.Request) {
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
	if !hasCapability(role, capabilityManageParking) {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	settingsMsg, settingsOK := parkingSettingsMessage(r.URL.Query().Get("settings"))
	a.render(w, "parkingSettings", map[string]any{
		"Title":         "Parkplatz-Abrechnung",
		"Tenant":        tenant,
		"Email":         email,
		"DisplayName":   profile.DisplayName(),
		"Initials":      profile.Initials(),
		"Role":          role,
		"IsAdmin":       true,
		"CanSeeParking": true,
		"ActivePage":    "settings",
		"Accounting":    a.parkingAccounting(r.Context(), tenant),
		"SettingsMsg":   settingsMsg,
		"SettingsOK":    settingsOK,
	})
}

func (a *app) parkingMonth(w http.ResponseWriter, r *http.Request) {
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
	if !hasCapability(role, capabilityPlatformAdmin) && !profile.HasPermission(permissionParking) {
		http.NotFound(w, r)
		return
	}
	month := strings.TrimSpace(r.PathValue("month"))
	if _, err := time.Parse("2006-01", month); err != nil {
		http.NotFound(w, r)
		return
	}
	view := a.parkingMonthDetails(r.Context(), tenant, month)
	if !view.HasHours && view.Summary.Month == "" {
		http.NotFound(w, r)
		return
	}
	a.render(w, "parkingMonth", map[string]any{
		"Title":         "Parkplatznutzung · " + view.MonthLabel,
		"Tenant":        tenant,
		"Email":         email,
		"DisplayName":   profile.DisplayName(),
		"Initials":      profile.Initials(),
		"Role":          role,
		"IsAdmin":       hasCapability(role, capabilityPlatformAdmin),
		"CanSeeParking": true,
		"ActivePage":    "parking",
		"Detail":        view,
	})
}

func (a *app) updateParkingSettings(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	_, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !hasCapability(role, capabilityManageParking) {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	gridFee, err := parseDecimal(r.FormValue("grid_fee_eur_per_kwh"))
	if err != nil || gridFee < 0 || gridFee > 5 {
		http.Redirect(w, r, "/app/parking/settings?settings=invalid", http.StatusSeeOther)
		return
	}
	if err := a.parkingStore.SetGridFee(tenant.Slug, gridFee); err != nil {
		log.Printf("parking settings save failed for %s: %v", tenant.Slug, err)
		http.Error(w, "Could not save parking settings", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/app/parking/settings?settings=saved", http.StatusSeeOther)
}

func (a *app) updateParkingMonth(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	_, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !hasCapability(role, capabilityManageParking) {
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

func parkingSettingsMessage(status string) (string, bool) {
	switch status {
	case "saved":
		return "Parkplatz-Abrechnung gespeichert.", true
	case "invalid":
		return "Bitte eine gültige Netzgebühr zwischen 0 und 5 €/kWh eingeben.", false
	default:
		return "", false
	}
}

func announcementMessage(status string) string {
	switch status {
	case "created":
		return "Aushang gespeichert."
	case "updated":
		return "Aushang aktualisiert."
	case "deleted":
		return "Aushang gelöscht."
	case "invalid":
		return "Bitte Titel, Text und Veröffentlichungsdatum prüfen."
	case "missing":
		return "Dieser Aushang wurde nicht gefunden."
	case "error":
		return "Der Aushang konnte nicht gespeichert werden."
	default:
		return ""
	}
}

func canManageAnnouncements(role string) bool {
	return hasCapability(role, capabilityManageAnnouncements)
}

func hasCapability(role string, action capability) bool {
	role = normalizeRole(role)
	if role == roleAdmin {
		return true
	}
	switch action {
	case capabilityPlatformAdmin, capabilityManageUsers, capabilityManageParking:
		return false
	case capabilityManageAnnouncements:
		return role == roleManager
	case capabilityOwnerDocuments, capabilityVote:
		return role == roleOwner
	case capabilityOversight:
		return role == roleBeirat || role == roleManager
	default:
		return false
	}
}

func roleCapabilityLabels(role string) []string {
	role = normalizeRole(role)
	switch role {
	case roleAdmin:
		return []string{"Plattformverwaltung", "Alle Bereiche"}
	case roleManager:
		return []string{"Aushang verwalten", "Übersicht"}
	case roleOwner:
		return []string{"Eigentümer-Dokumente", "Abstimmungen"}
	case roleRenter:
		return []string{"Bewohnerbereich"}
	case roleBeirat:
		return []string{"Übersicht", "Leserechte"}
	case roleResident:
		return []string{"Bewohnerbereich"}
	default:
		if role == "" {
			return []string{"Bewohnerbereich"}
		}
		return []string{role}
	}
}

func roleSortRank(role string) int {
	switch normalizeRole(role) {
	case roleAdmin:
		return 0
	case roleManager:
		return 1
	case roleBeirat:
		return 2
	case roleOwner:
		return 3
	case roleRenter:
		return 4
	case roleResident:
		return 5
	default:
		return 6
	}
}

func roleClass(role string) string {
	switch normalizeRole(role) {
	case roleAdmin:
		return "role-admin"
	case roleManager:
		return "role-manager"
	case roleOwner:
		return "role-owner"
	case roleRenter:
		return "role-renter"
	case roleBeirat:
		return "role-beirat"
	case roleResident:
		return "role-resident"
	default:
		return "role-resident"
	}
}

func sameOriginPost(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return true
	}
	host := normalizeHost(r.Host)
	if host == "" {
		return false
	}
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		parsed, err := url.Parse(origin)
		return err == nil && normalizeHost(parsed.Host) == host
	}
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
		parsed, err := url.Parse(referer)
		return err == nil && normalizeHost(parsed.Host) == host
	}
	return true
}

func announcementFromForm(r *http.Request, tenantSlug string, author userProfile, now time.Time) (announcement, error) {
	title := strings.TrimSpace(r.FormValue("title"))
	body := strings.TrimSpace(r.FormValue("body"))
	if title == "" || body == "" {
		return announcement{}, fmt.Errorf("title and body are required")
	}
	if len([]rune(title)) > 140 {
		return announcement{}, fmt.Errorf("title too long")
	}
	if len([]rune(body)) > 5000 {
		return announcement{}, fmt.Errorf("body too long")
	}
	publishedAt, err := parseOptionalLocalDateTime(r.FormValue("published_at"), now)
	if err != nil {
		return announcement{}, err
	}
	expiresAt, err := parseOptionalExpiry(r.FormValue("expires_at"))
	if err != nil {
		return announcement{}, err
	}
	item := announcement{
		TenantSlug:  normalizeSlug(tenantSlug),
		Title:       title,
		Body:        body,
		Category:    normalizeAnnouncementCategory(r.FormValue("category")),
		Pinned:      parseBool(r.FormValue("pinned")),
		PublishedAt: publishedAt.UTC(),
		ExpiresAt:   expiresAt,
		AuthorEmail: normalizeEmail(author.Email),
		AuthorName:  author.DisplayName(),
	}
	if item.TenantSlug == "" {
		return announcement{}, fmt.Errorf("tenant is required")
	}
	return item, nil
}

func parseOptionalLocalDateTime(raw string, fallback time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t, nil
		}
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid datetime")
}

func parseOptionalExpiry(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	t, err := parseOptionalLocalDateTime(raw, time.Time{})
	if err != nil {
		return nil, err
	}
	t = t.UTC()
	return &t, nil
}

func normalizeAnnouncementCategory(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "dringend", "urgent":
		return "Dringend"
	case "termin", "date", "event":
		return "Termin"
	case "wartung", "maintenance":
		return "Wartung"
	default:
		return "Info"
	}
}

func selectedAnnouncementCategory(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "all") || strings.EqualFold(raw, "alle") {
		return ""
	}
	return normalizeAnnouncementCategory(raw)
}

func filterAnnouncements(items []announcement, category string, query string) []announcement {
	category = selectedAnnouncementCategory(category)
	query = strings.ToLower(strings.TrimSpace(query))
	if category == "" && query == "" {
		return items
	}
	out := []announcement{}
	for _, item := range items {
		if category != "" && item.Category != category {
			continue
		}
		if query != "" {
			haystack := strings.ToLower(item.Title + "\n" + item.Body + "\n" + item.Category)
			if !strings.Contains(haystack, query) {
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

func announcementFilterViews(query string, selectedCategory string) []announcementFilterView {
	selectedCategory = selectedAnnouncementCategory(selectedCategory)
	categories := []string{"", "Info", "Termin", "Wartung", "Dringend"}
	labels := map[string]string{"": "Alle"}
	out := make([]announcementFilterView, 0, len(categories))
	for _, category := range categories {
		label := labels[category]
		if label == "" {
			label = category
		}
		values := url.Values{}
		if strings.TrimSpace(query) != "" {
			values.Set("q", strings.TrimSpace(query))
		}
		if category != "" {
			values.Set("category", category)
		}
		filterURL := "/app/announcements"
		if encoded := values.Encode(); encoded != "" {
			filterURL += "?" + encoded
		}
		out = append(out, announcementFilterView{
			Label:  label,
			URL:    filterURL,
			Active: category == selectedCategory,
		})
	}
	return out
}

func unreadAnnouncementCount(items []announcement, lastSeen time.Time, now time.Time) int {
	count := 0
	for _, item := range items {
		if announcementUnread(item, lastSeen, now) {
			count++
		}
	}
	return count
}

func announcementUnread(item announcement, lastSeen time.Time, now time.Time) bool {
	if item.PublishedAt.After(now) {
		return false
	}
	if item.ExpiresAt != nil && !item.ExpiresAt.After(now) {
		return false
	}
	if lastSeen.IsZero() {
		return true
	}
	return item.PublishedAt.After(lastSeen)
}

func announcementViews(items []announcement, now time.Time, includeStatus bool) []announcementView {
	return announcementViewsWithReadState(items, now, includeStatus, time.Time{})
}

func announcementViewsWithReadState(items []announcement, now time.Time, includeStatus bool, lastSeen time.Time) []announcementView {
	views := make([]announcementView, 0, len(items))
	for _, item := range items {
		views = append(views, announcementViewFrom(item, now, includeStatus, lastSeen))
	}
	return views
}

func announcementViewFrom(item announcement, now time.Time, includeStatus bool, lastSeen time.Time) announcementView {
	published := !item.PublishedAt.After(now)
	expired := item.ExpiresAt != nil && !item.ExpiresAt.After(now)
	status := ""
	switch {
	case expired:
		status = "Abgelaufen"
	case !published:
		status = "Geplant"
	case item.Pinned:
		status = "Fixiert"
	default:
		status = "Veröffentlicht"
	}
	if !includeStatus {
		status = ""
	}
	expiresAt := ""
	expiresAtInput := ""
	if item.ExpiresAt != nil {
		expiresAt = item.ExpiresAt.In(time.Local).Format("02.01.2006 15:04")
		expiresAtInput = item.ExpiresAt.In(time.Local).Format("2006-01-02T15:04")
	}
	author := strings.TrimSpace(item.AuthorName)
	if author == "" {
		author = item.AuthorEmail
	}
	return announcementView{
		ID:                 item.ID,
		Title:              item.Title,
		Body:               item.Body,
		BodyHTML:           plainTextHTML(item.Body),
		Category:           item.Category,
		CategoryClass:      strings.ToLower(normalizeSlug(item.Category)),
		Pinned:             item.Pinned,
		PinnedChecked:      item.Pinned,
		PublishedAt:        item.PublishedAt.In(time.Local).Format("02.01.2006 15:04"),
		PublishedAtInput:   item.PublishedAt.In(time.Local).Format("2006-01-02T15:04"),
		ExpiresAt:          expiresAt,
		ExpiresAtInput:     expiresAtInput,
		HasExpiresAt:       item.ExpiresAt != nil,
		Author:             author,
		Status:             status,
		Published:          published,
		Expired:            expired,
		Unread:             announcementUnread(item, lastSeen, now),
		EditDialogID:       "announcement-edit-" + item.ID,
		DeleteConfirmLabel: "Aushang \"" + item.Title + "\" wirklich löschen?",
	}
}

func plainTextHTML(body string) template.HTML {
	escaped := template.HTMLEscapeString(strings.TrimSpace(body))
	escaped = strings.ReplaceAll(escaped, "\r\n", "\n")
	escaped = strings.ReplaceAll(escaped, "\n\n", "<br><br>")
	escaped = strings.ReplaceAll(escaped, "\n", "<br>")
	return template.HTML(escaped)
}

func (a *app) settingsHub(w http.ResponseWriter, r *http.Request) {
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
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	a.render(w, "settingsHub", map[string]any{
		"Title":         "Einstellungen",
		"Tenant":        tenant,
		"Email":         email,
		"DisplayName":   profile.DisplayName(),
		"Initials":      profile.Initials(),
		"Role":          role,
		"IsAdmin":       isAdmin,
		"CanSeeParking": isAdmin || profile.HasPermission(permissionParking),
		"ActivePage":    "settings",
	})
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
	if !hasCapability(role, capabilityManageUsers) {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	profile := a.profileFor(email)
	inviteMsg, inviteOK := inviteMessage(r.URL.Query().Get("invite"))
	a.render(w, "userSettings", map[string]any{
		"Title":         "Benutzer & Rechte",
		"Tenant":        tenant,
		"Email":         email,
		"DisplayName":   profile.DisplayName(),
		"Initials":      profile.Initials(),
		"Role":          role,
		"Users":         a.userRows(tenant.Slug),
		"InviteMsg":     inviteMsg,
		"InviteOK":      inviteOK,
		"IsAdmin":       hasCapability(role, capabilityPlatformAdmin),
		"CanSeeParking": hasCapability(role, capabilityPlatformAdmin) || profile.HasPermission(permissionParking),
		"ActivePage":    "users",
	})
}

func inviteMessage(status string) (string, bool) {
	switch status {
	case "invited":
		return "Einladung gespeichert und per E-Mail verschickt.", true
	case "saved_no_mail":
		return "Einladung gespeichert. Die E-Mail konnte nicht zugestellt werden.", false
	case "exists":
		return "Diese E-Mail-Adresse ist bereits eingetragen.", false
	case "invalid_email":
		return "Bitte eine gültige E-Mail-Adresse angeben.", false
	case "error":
		return "Die Einladung konnte nicht gespeichert werden.", false
	case "updated":
		return "Änderungen gespeichert.", true
	case "deleted":
		return "Zugang gelöscht.", true
	case "not_editable":
		return "Dieser Eintrag kommt aus der Konfiguration und kann hier nicht geändert werden.", false
	default:
		return "", false
	}
}

func (a *app) createInvite(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	_, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !hasCapability(role, capabilityManageUsers) {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	inviteEmail := normalizeEmail(r.FormValue("email"))
	if _, err := mail.ParseAddress(inviteEmail); err != nil {
		a.redirectInvite(w, r, "invalid_email")
		return
	}
	if _, exists := a.profiles[inviteEmail]; exists {
		a.redirectInvite(w, r, "exists")
		return
	}

	inviteRole := normalizeRole(r.FormValue("role"))
	if inviteRole == "" {
		inviteRole = roleResident
	}
	profile := userProfile{
		Email:       inviteEmail,
		Title:       strings.TrimSpace(r.FormValue("title")),
		FirstName:   strings.TrimSpace(r.FormValue("first_name")),
		LastName:    strings.TrimSpace(r.FormValue("last_name")),
		Role:        inviteRole,
		Status:      "Eingeladen",
		Tenants:     []string{tenant.Slug},
		AuthMethods: defaultAuthMethods(),
	}

	added, err := a.inviteStore.Add(profile)
	if err != nil {
		log.Printf("invite persistence failed for %s: %v", redactedEmail(inviteEmail), err)
		a.redirectInvite(w, r, "error")
		return
	}
	if !added {
		a.redirectInvite(w, r, "exists")
		return
	}

	loginURL := a.publicBaseURL(r, tenant) + "/"
	if err := a.mailer.SendInvite(inviteEmail, loginURL, tenant.Address); err != nil {
		log.Printf("invite email delivery failed for %s: %v", redactedEmail(inviteEmail), err)
		a.redirectInvite(w, r, "saved_no_mail")
		return
	}
	a.redirectInvite(w, r, "invited")
}

func (a *app) redirectInvite(w http.ResponseWriter, r *http.Request, status string) {
	http.Redirect(w, r, "/app/settings/users?invite="+url.QueryEscape(status), http.StatusSeeOther)
}

func (a *app) editInvite(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	_, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !hasCapability(role, capabilityManageUsers) {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	orig := normalizeEmail(r.FormValue("orig_email"))
	existing, isInvite := a.inviteStore.Get(orig)
	if !isInvite {
		// Only persisted invites are editable; env-config users are read-only.
		a.redirectInvite(w, r, "not_editable")
		return
	}

	newEmail := normalizeEmail(r.FormValue("email"))
	if _, err := mail.ParseAddress(newEmail); err != nil {
		a.redirectInvite(w, r, "invalid_email")
		return
	}
	if newEmail != orig {
		if _, inEnv := a.profiles[newEmail]; inEnv {
			a.redirectInvite(w, r, "exists")
			return
		}
	}

	newRole := normalizeRole(r.FormValue("role"))
	if newRole == "" {
		newRole = roleResident
	}
	updated := existing
	updated.Email = newEmail
	updated.Title = strings.TrimSpace(r.FormValue("title"))
	updated.FirstName = strings.TrimSpace(r.FormValue("first_name"))
	updated.LastName = strings.TrimSpace(r.FormValue("last_name"))
	updated.Role = newRole
	if len(updated.Tenants) == 0 {
		updated.Tenants = []string{tenant.Slug}
	}
	if len(updated.AuthMethods) == 0 {
		updated.AuthMethods = defaultAuthMethods()
	}

	changed, err := a.inviteStore.Update(orig, updated)
	if err != nil {
		a.redirectInvite(w, r, "exists")
		return
	}
	if !changed {
		a.redirectInvite(w, r, "not_editable")
		return
	}
	a.redirectInvite(w, r, "updated")
}

func (a *app) deleteInvite(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	_, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !hasCapability(role, capabilityManageUsers) {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	removed, err := a.inviteStore.Delete(normalizeEmail(r.FormValue("email")))
	if err != nil {
		a.redirectInvite(w, r, "error")
		return
	}
	if !removed {
		a.redirectInvite(w, r, "not_editable")
		return
	}
	a.redirectInvite(w, r, "deleted")
}

func (a *app) render(w http.ResponseWriter, name string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	if _, ok := data["AppVersion"]; !ok {
		data["AppVersion"] = buildLabel()
	}
	a.enrichUnreadAnnouncementData(data)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render %s failed: %v", name, err)
	}
}

func (a *app) enrichUnreadAnnouncementData(data map[string]any) {
	if _, ok := data["UnreadAnnouncements"]; ok {
		if _, hasFlag := data["HasUnreadAnnouncements"]; !hasFlag {
			if count, ok := data["UnreadAnnouncements"].(int); ok {
				data["HasUnreadAnnouncements"] = count > 0
			}
		}
		return
	}
	tenant, ok := data["Tenant"].(tenantConfig)
	if !ok || tenant.Slug == "" || a.announcementStore == nil || a.announcementReadStore == nil {
		data["UnreadAnnouncements"] = 0
		data["HasUnreadAnnouncements"] = false
		return
	}
	email, ok := data["Email"].(string)
	if !ok || strings.TrimSpace(email) == "" {
		data["UnreadAnnouncements"] = 0
		data["HasUnreadAnnouncements"] = false
		return
	}
	now := time.Now()
	lastSeen := a.announcementReadStore.LastSeen(tenant.Slug, email)
	count := unreadAnnouncementCount(a.announcementStore.Visible(tenant.Slug, now), lastSeen, now)
	data["UnreadAnnouncements"] = count
	data["HasUnreadAnnouncements"] = count > 0
}

func buildLabel() string {
	version := strings.TrimPrefix(strings.TrimSpace(appVersion), "v")
	if version == "" {
		version = "0.1.0"
	}
	commit := strings.TrimSpace(gitCommit)
	if commit == "" {
		commit = "dev"
	}
	return fmt.Sprintf("%s (%s)", version, commit)
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
	email, tenantSlug, authMethod, ok := a.sessions.Get(c.Value)
	if !ok {
		return "", "", "", false
	}
	if !a.isAllowed(email, tenantSlug) || !a.isAuthMethodAllowed(email, tenantSlug, authMethod) {
		return "", "", "", false
	}
	return email, a.roleFor(email, tenantSlug), tenantSlug, true
}

// directoryProfile resolves a profile from the env directory first, then falls
// back to persisted invites. Env is authoritative: an invite can never override
// or escalate an env-defined user, so existing logins are unaffected.
func (a *app) directoryProfile(email string) (userProfile, bool) {
	email = normalizeEmail(email)
	if profile, ok := a.profiles[email]; ok {
		return profile, true
	}
	if a.inviteStore != nil {
		if profile, ok := a.inviteStore.Get(email); ok {
			return profile, true
		}
	}
	return userProfile{}, false
}

func (a *app) isAllowed(email string, tenantSlug string) bool {
	if profile, ok := a.directoryProfile(email); ok && profile.HasTenant(tenantSlug) {
		return true
	}
	if _, ok := a.admins[email]; ok {
		return true
	}
	_, ok := a.allowed[email]
	return ok
}

func (a *app) isAuthMethodAllowed(email string, tenantSlug string, authMethod string) bool {
	authMethod = normalizeAuthMethod(authMethod)
	if authMethod == "" {
		return false
	}
	profile, ok := a.directoryProfile(email)
	if !ok || !profile.HasTenant(tenantSlug) {
		return false
	}
	return profile.AllowsAuthMethod(authMethod)
}

func (a *app) emailLoginAvailable() bool {
	return a.mailer.Configured() || a.localDevLogin
}

func (a *app) roleFor(email string, tenantSlug string) string {
	if profile, ok := a.directoryProfile(email); ok && profile.Role != "" {
		return normalizeRole(profile.Role)
	}
	if _, ok := a.admins[email]; ok {
		return roleAdmin
	}
	return roleResident
}

func (a *app) profileFor(email string) userProfile {
	email = normalizeEmail(email)
	if profile, ok := a.directoryProfile(email); ok {
		return profile
	}
	role := a.roleFor(email, a.defaultTenant)
	return userProfile{
		Email:       email,
		Role:        role,
		Status:      "Eingeladen",
		Tenants:     []string{a.defaultTenant},
		AuthMethods: defaultAuthMethods(),
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
		seen[email] = struct{}{}
	}
	if a.inviteStore != nil {
		for _, profile := range a.inviteStore.List() {
			email := normalizeEmail(profile.Email)
			if _, ok := seen[email]; ok {
				continue
			}
			if !profile.HasTenant(tenantSlug) {
				continue
			}
			row := profile.UserRow()
			row.Editable = true
			rows = append(rows, row)
			seen[email] = struct{}{}
		}
	}
	if len(rows) == 0 {
		rows = append(rows, userProfile{Email: "Noch keine Einladungen", Role: roleResident, Status: "Offen", Tenants: []string{tenantSlug}}.UserRow())
	}
	if a.activityStore != nil {
		for i := range rows {
			if !strings.Contains(rows[i].Email, "@") {
				continue
			}
			if rec, ok := a.activityStore.Get(rows[i].Email); ok {
				rows[i].Status = "Aktiv"
				rows[i].LastSeen = "zuletzt angemeldet: " + rec.LastLogin.In(time.Local).Format("02.01.2006")
			} else if rows[i].Status == "Eingeladen" {
				rows[i].LastSeen = "noch nie angemeldet"
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Role != rows[j].Role {
			return roleSortRank(rows[i].Role) < roleSortRank(rows[j].Role)
		}
		if rows[i].LastName != rows[j].LastName {
			return rows[i].LastName < rows[j].LastName
		}
		return rows[i].Email < rows[j].Email
	})
	return rows
}

func newAnnouncementStore(path string) (*announcementStore, error) {
	store := &announcementStore{path: path, data: announcementStoreData{Announcements: []announcement{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read announcement data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid announcement data")
	}
	return store, nil
}

func newAnnouncementReadStore(path string) (*announcementReadStore, error) {
	store := &announcementReadStore{path: path, data: announcementReadStoreData{Seen: map[string]map[string]time.Time{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read announcement read data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid announcement read data")
	}
	if store.data.Seen == nil {
		store.data.Seen = map[string]map[string]time.Time{}
	}
	return store, nil
}

func (s *announcementReadStore) LastSeen(tenantSlug string, email string) time.Time {
	if s == nil {
		return time.Time{}
	}
	tenantSlug = normalizeSlug(tenantSlug)
	email = normalizeEmail(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	if tenantSlug == "" || email == "" {
		return time.Time{}
	}
	return s.data.Seen[tenantSlug][email]
}

func (s *announcementReadStore) MarkSeen(tenantSlug string, email string, seenAt time.Time) error {
	if s == nil {
		return nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	email = normalizeEmail(email)
	if tenantSlug == "" || email == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Seen == nil {
		s.data.Seen = map[string]map[string]time.Time{}
	}
	if s.data.Seen[tenantSlug] == nil {
		s.data.Seen[tenantSlug] = map[string]time.Time{}
	}
	s.data.Seen[tenantSlug][email] = seenAt.UTC()
	return s.saveLocked()
}

func (s *announcementReadStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("could not create announcement read data directory")
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode announcement read data")
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("could not write announcement read data")
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("could not replace announcement read data")
	}
	return nil
}

func (s *announcementStore) Create(item announcement) (announcement, error) {
	now := time.Now().UTC()
	item.ID = ""
	item.CreatedAt = now
	item.UpdatedAt = now
	if item.PublishedAt.IsZero() {
		item.PublishedAt = now
	}
	if item.Category == "" {
		item.Category = "Info"
	}
	id, err := randomToken(12)
	if err != nil {
		return announcement{}, err
	}
	item.ID = id
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Announcements = append(s.data.Announcements, item)
	if err := s.saveLocked(); err != nil {
		return announcement{}, err
	}
	return item, nil
}

func (s *announcementStore) Update(id string, updated announcement) (bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Announcements {
		if existing.ID != id || normalizeSlug(existing.TenantSlug) != normalizeSlug(updated.TenantSlug) {
			continue
		}
		updated.ID = existing.ID
		updated.CreatedAt = existing.CreatedAt
		updated.UpdatedAt = time.Now().UTC()
		if updated.PublishedAt.IsZero() {
			updated.PublishedAt = existing.PublishedAt
		}
		if updated.AuthorEmail == "" {
			updated.AuthorEmail = existing.AuthorEmail
		}
		if updated.AuthorName == "" {
			updated.AuthorName = existing.AuthorName
		}
		s.data.Announcements[i] = updated
		if err := s.saveLocked(); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (s *announcementStore) Delete(tenantSlug string, id string) (bool, error) {
	tenantSlug = normalizeSlug(tenantSlug)
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.data.Announcements[:0]
	removed := false
	for _, item := range s.data.Announcements {
		if item.ID == id && normalizeSlug(item.TenantSlug) == tenantSlug {
			removed = true
			continue
		}
		kept = append(kept, item)
	}
	if !removed {
		return false, nil
	}
	s.data.Announcements = kept
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *announcementStore) Visible(tenantSlug string, now time.Time) []announcement {
	tenantSlug = normalizeSlug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []announcement{}
	for _, item := range s.data.Announcements {
		if normalizeSlug(item.TenantSlug) != tenantSlug {
			continue
		}
		if item.PublishedAt.After(now) {
			continue
		}
		if item.ExpiresAt != nil && !item.ExpiresAt.After(now) {
			continue
		}
		out = append(out, item)
	}
	sortAnnouncements(out)
	return out
}

func (s *announcementStore) Archive(tenantSlug string, now time.Time) []announcement {
	tenantSlug = normalizeSlug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []announcement{}
	for _, item := range s.data.Announcements {
		if normalizeSlug(item.TenantSlug) != tenantSlug {
			continue
		}
		if item.PublishedAt.After(now) {
			continue
		}
		out = append(out, item)
	}
	sortAnnouncements(out)
	return out
}

func (s *announcementStore) ListTenant(tenantSlug string) []announcement {
	tenantSlug = normalizeSlug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []announcement{}
	for _, item := range s.data.Announcements {
		if normalizeSlug(item.TenantSlug) == tenantSlug {
			out = append(out, item)
		}
	}
	sortAnnouncements(out)
	return out
}

func (s *announcementStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("could not create announcement data directory")
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode announcement data")
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("could not write announcement data")
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("could not replace announcement data")
	}
	return nil
}

func sortAnnouncements(items []announcement) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Pinned != items[j].Pinned {
			return items[i].Pinned
		}
		if !items[i].PublishedAt.Equal(items[j].PublishedAt) {
			return items[i].PublishedAt.After(items[j].PublishedAt)
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
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
	seedCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	if err := a.seedParkingHistory(seedCtx, tenant, a.parkingHistoryStart, time.Now()); err != nil && tenant.HA.baseURL != "" && tenant.HA.token != "" {
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
		view.Message = "Kosten werden stündlich aus Zählerdifferenz, aWATTar-Preis und Netzbetreibergebühren berechnet. Historie wird ab " + a.parkingHistoryStart.In(time.Local).Format("02.01.2006") + " aus Home Assistant nachgezogen, soweit dort Statistikdaten vorhanden sind."
	}
	return view
}

func (a *app) parkingMonthDetails(ctx context.Context, tenant tenantConfig, month string) parkingMonthDetailView {
	seedCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	if err := a.seedParkingHistory(seedCtx, tenant, a.parkingHistoryStart, time.Now()); err != nil && tenant.HA.baseURL != "" && tenant.HA.token != "" {
		log.Printf("parking history seed failed for %s: %v", tenant.Slug, err)
	}

	data := a.parkingStore.TenantData(tenant.Slug)
	view := calculateParkingMonthDetails(data, month, time.Now(), time.Local)
	view.BackPath = "/app/parking"
	view.GridFeeLabel = formatEURPerKWh(data.Settings.GridFeeEURPerKWh)
	view.Message = "Stundenwerte aus Zählerdifferenz und dem in dieser Stunde gültigen aWATTar-Preis."
	if len(data.EnergySamples) > 0 {
		last := data.EnergySamples[len(data.EnergySamples)-1].At.In(time.Local)
		view.LastSampleLabel = last.Format("02.01.2006 15:04")
	}
	return view
}

func (a *app) seedParkingHistory(ctx context.Context, tenant tenantConfig, start time.Time, end time.Time) error {
	ha := tenant.HA
	if ha.baseURL == "" || ha.token == "" || ha.meterEnergyEntity == "" || ha.priceEntity == "" {
		return nil
	}
	energySamples, priceSamples, err := ha.Statistics(ctx, start, end)
	if err != nil {
		log.Printf("parking statistics backfill failed for %s: %v", tenant.Slug, err)
	}
	restStart := end.Add(-35 * 24 * time.Hour)
	if restStart.Before(start) {
		restStart = start
	}
	history, err := ha.History(ctx, restStart, end, []string{ha.meterEnergyEntity, ha.priceEntity})
	if err != nil {
		if len(energySamples) == 0 && len(priceSamples) == 0 {
			return err
		}
		log.Printf("parking REST history fallback failed for %s: %v", tenant.Slug, err)
		return a.parkingStore.AppendReadings(tenant.Slug, energySamples, priceSamples)
	}
	energySamples = append(energySamples, samplesFromHistory(history[ha.meterEnergyEntity])...)
	priceSamples = append(priceSamples, samplesFromHistory(history[ha.priceEntity])...)
	if len(energySamples) == 0 && len(priceSamples) == 0 {
		return nil
	}
	log.Printf("parking history seed found %d energy sample(s) and %d price sample(s) for %s", len(energySamples), len(priceSamples), tenant.Slug)
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
		a.backfillParkingTenants(ctx)
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

func (a *app) backfillParkingTenants(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for _, tenant := range a.tenants {
		if tenant.HA.baseURL == "" || tenant.HA.token == "" {
			continue
		}
		if err := a.seedParkingHistory(ctx, tenant, a.parkingHistoryStart, time.Now()); err != nil {
			log.Printf("parking startup history seed failed for %s: %v", tenant.Slug, err)
			continue
		}
		log.Printf("parking startup history seed completed for %s", tenant.Slug)
	}
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

func samplesFromStatistics(stats []haStatistic, fields ...string) []parkingNumericSample {
	out := make([]parkingNumericSample, 0, len(stats))
	for _, stat := range stats {
		at, err := parseStatisticTime(stat.Start)
		if err != nil || at.IsZero() {
			continue
		}
		value, ok := statisticValue(stat, fields...)
		if !ok {
			continue
		}
		out = append(out, parkingNumericSample{At: at.UTC(), Value: value})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At)
	})
	return out
}

func statisticValue(stat haStatistic, fields ...string) (float64, bool) {
	for _, field := range fields {
		switch field {
		case "state":
			if stat.State != nil {
				return *stat.State, true
			}
		case "sum":
			if stat.Sum != nil {
				return *stat.Sum, true
			}
		case "mean":
			if stat.Mean != nil {
				return *stat.Mean, true
			}
		case "min":
			if stat.Min != nil {
				return *stat.Min, true
			}
		case "max":
			if stat.Max != nil {
				return *stat.Max, true
			}
		}
	}
	return 0, false
}

func parseStatisticTime(raw json.RawMessage) (time.Time, error) {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 || string(raw) == "null" {
		return time.Time{}, errors.New("missing statistic time")
	}
	if raw[0] == '"' {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return time.Time{}, err
		}
		return time.Parse(time.RFC3339, value)
	}
	var value float64
	if err := json.Unmarshal(raw, &value); err != nil {
		return time.Time{}, err
	}
	if value > 100000000000 {
		return time.UnixMilli(int64(value)), nil
	}
	return time.Unix(int64(value), 0), nil
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
		averageAwattar := 0.0
		effectivePrice := 0.0
		if agg.kWh > 0 {
			averageAwattar = agg.energyCost / agg.kWh
			effectivePrice = total / agg.kWh
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
			DetailPath:      "/app/parking/month/" + month,
			PeriodLabel:     formatPeriodLabel(agg.first, agg.last, loc),
			KWh:             formatKWh(agg.kWh),
			EnergyCost:      formatEUR(agg.energyCost),
			GridCost:        formatEUR(agg.gridCost),
			TotalCost:       formatEUR(total),
			AverageAwattar:  formatEURPerKWh(averageAwattar),
			EffectivePrice:  formatEURPerKWh(effectivePrice),
			AveragePrice:    formatEURPerKWh(effectivePrice),
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

func calculateParkingMonthDetails(data parkingTenantData, month string, now time.Time, loc *time.Location) parkingMonthDetailView {
	if loc == nil {
		loc = time.Local
	}
	view := parkingMonthDetailView{
		Month:      month,
		MonthLabel: formatMonthLabel(month, loc),
	}
	for _, summary := range calculateParkingMonths(data, now, loc) {
		if summary.Month == month {
			view.Summary = summary
			break
		}
	}
	hours := calculateParkingHourlyUsage(data.EnergySamples, data.PriceSamples, data.Settings.GridFeeEURPerKWh, now)
	if len(hours) == 0 {
		return view
	}
	maxTotal := 0.0
	var monthHours []parkingHourUsage
	for _, hour := range hours {
		if hour.At.In(loc).Format("2006-01") != month {
			continue
		}
		monthHours = append(monthHours, hour)
		total := hour.EnergyCost + hour.GridCost
		if total > maxTotal {
			maxTotal = total
		}
	}
	if len(monthHours) == 0 {
		return view
	}
	sort.Slice(monthHours, func(i, j int) bool {
		return monthHours[i].At.Before(monthHours[j].At)
	})
	view.Hours = make([]parkingHourView, 0, len(monthHours))
	for _, hour := range monthHours {
		total := hour.EnergyCost + hour.GridCost
		averageAwattar := 0.0
		if hour.KWh > 0 {
			averageAwattar = hour.EnergyCost / hour.KWh
		}
		chartPercent := 0
		if maxTotal > 0 {
			chartPercent = int(total / maxTotal * 100)
			if chartPercent < 2 && total > 0 {
				chartPercent = 2
			}
		}
		view.Hours = append(view.Hours, parkingHourView{
			AtLabel:             hour.At.In(loc).Format("02.01. 15:04"),
			AtTitle:             hour.At.In(loc).Format("02.01.2006 15:04") + " bis " + hour.At.Add(time.Hour).In(loc).Format("15:04"),
			KWh:                 formatKWh(hour.KWh),
			KWhTitle:            "Verbrauch: " + formatPreciseKWh(hour.KWh),
			AverageAwattar:      formatEURPerKWh(averageAwattar),
			AverageAwattarTitle: "aWATTar Preis dieser Stunde: " + formatPreciseEURPerKWh(averageAwattar),
			EnergyCost:          formatEUR(hour.EnergyCost),
			EnergyCostTitle:     "Stromkosten: " + formatPreciseEUR(hour.EnergyCost) + " = " + formatPreciseKWh(hour.KWh) + " × " + formatPreciseEURPerKWh(averageAwattar),
			GridCost:            formatEUR(hour.GridCost),
			GridCostTitle:       "Netzgebühr: " + formatPreciseEUR(hour.GridCost) + " = " + formatPreciseKWh(hour.KWh) + " × " + formatPreciseEURPerKWh(data.Settings.GridFeeEURPerKWh),
			TotalCost:           formatEUR(total),
			TotalCostTitle:      "Summe: " + formatPreciseEUR(total) + " = Strom " + formatPreciseEUR(hour.EnergyCost) + " + Netzgebühr " + formatPreciseEUR(hour.GridCost),
			WeightTitle:         "Relative Höhe der Stundensumme. 100% entspricht der teuersten Stunde dieses Monats.",
			ChartPercent:        chartPercent,
		})
	}
	view.HasHours = len(view.Hours) > 0
	return view
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
	AuthMethods []string `json:"auth_methods"`
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

// Initials returns up to two uppercase letters for the avatar badge, derived
// from first+last name, falling back to the first glyph of the display name.
func (p userProfile) Initials() string {
	first := initialLetter(p.FirstName)
	last := initialLetter(p.LastName)
	if first == "" && last == "" {
		return strings.ToUpper(initialLetter(p.DisplayName()))
	}
	return strings.ToUpper(first + last)
}

func initialLetter(s string) string {
	for _, r := range strings.TrimSpace(s) {
		return string(r)
	}
	return ""
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

func (p userProfile) AllowsAuthMethod(method string) bool {
	method = normalizeAuthMethod(method)
	if method == "" {
		return false
	}
	methods := p.AuthMethods
	if len(methods) == 0 {
		methods = defaultAuthMethods()
	}
	for _, item := range methods {
		if normalizeAuthMethod(item) == method {
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
	p.Role = normalizeRole(p.Role)
	if p.Status == "" {
		p.Status = "Eingeladen"
	}
	return userRow{
		Email:            p.Email,
		Title:            p.Title,
		FirstName:        p.FirstName,
		LastName:         p.LastName,
		DisplayName:      p.DisplayName(),
		Initials:         p.Initials(),
		Role:             p.Role,
		RoleClass:        roleClass(p.Role),
		RoleCapabilities: roleCapabilityLabels(p.Role),
		Status:           p.Status,
		Tenants:          strings.Join(p.Tenants, ", "),
		PermissionLabel:  permissionLabel(p.Permissions),
		PermissionList:   permissionLabelList(p.Permissions),
		AuthLabel:        authMethodsLabel(p.AuthMethods),
		AuthList:         authMethodsLabelList(p.AuthMethods),
	}
}

type userRow struct {
	Email            string
	Title            string
	FirstName        string
	LastName         string
	DisplayName      string
	Initials         string
	Role             string
	RoleClass        string
	RoleCapabilities []string
	Status           string
	Tenants          string
	PermissionLabel  string
	PermissionList   []string
	AuthLabel        string
	AuthList         []string
	Editable         bool
	LastSeen         string
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

func newSessionStore(secret []byte) *sessionStore {
	sum := sha256.Sum256(append([]byte("weg-session-aead-v1."), secret...))
	key := make([]byte, len(sum))
	copy(key, sum[:])
	return &sessionStore{
		secret:  key,
		revoked: map[string]time.Time{},
	}
}

func (s *sessionStore) Put(email string, tenantSlug string, authMethod string, ttl time.Duration) (string, time.Time, error) {
	expiresAt := time.Now().Add(ttl)
	item := session{
		Email:      normalizeEmail(email),
		TenantSlug: normalizeSlug(tenantSlug),
		AuthMethod: normalizeAuthMethod(authMethod),
		ExpiresAt:  expiresAt.Unix(),
	}
	if item.Email == "" || item.TenantSlug == "" || item.AuthMethod == "" {
		return "", time.Time{}, fmt.Errorf("invalid session")
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return "", time.Time{}, err
	}
	block, err := aes.NewCipher(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", time.Time{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", time.Time{}, err
	}
	ciphertext := gcm.Seal(nil, nonce, payload, []byte("weg-session-v1"))
	token := "v1." + base64.RawURLEncoding.EncodeToString(nonce) + "." + base64.RawURLEncoding.EncodeToString(ciphertext)
	return token, expiresAt, nil
}

func (s *sessionStore) Get(token string) (string, string, string, bool) {
	item, ok := s.verify(token)
	if !ok {
		return "", "", "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
	if _, revoked := s.revoked[s.revocationKey(token)]; revoked {
		return "", "", "", false
	}
	return item.Email, item.TenantSlug, item.AuthMethod, true
}

func (s *sessionStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
	if item, ok := s.verify(token); ok {
		s.revoked[s.revocationKey(token)] = time.Unix(item.ExpiresAt, 0)
	}
}

func (s *sessionStore) verify(token string) (session, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "v1" {
		return session{}, false
	}
	nonce, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return session{}, false
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return session{}, false
	}
	block, err := aes.NewCipher(s.secret)
	if err != nil {
		return session{}, false
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(nonce) != gcm.NonceSize() {
		return session{}, false
	}
	payload, err := gcm.Open(nil, nonce, ciphertext, []byte("weg-session-v1"))
	if err != nil {
		return session{}, false
	}
	var item session
	if err := json.Unmarshal(payload, &item); err != nil {
		return session{}, false
	}
	item.Email = normalizeEmail(item.Email)
	item.TenantSlug = normalizeSlug(item.TenantSlug)
	item.AuthMethod = normalizeAuthMethod(item.AuthMethod)
	if item.Email == "" || item.TenantSlug == "" || item.AuthMethod == "" || item.ExpiresAt <= time.Now().Unix() {
		return session{}, false
	}
	return item, true
}

func (s *sessionStore) revocationKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (s *sessionStore) cleanupLocked() {
	now := time.Now()
	for key, expiresAt := range s.revoked {
		if now.After(expiresAt) {
			delete(s.revoked, key)
		}
	}
}

func newOIDCLogin(ctx context.Context, issuer string, clientID string, clientSecret string, redirectURL string, providerName string) (*oidcLogin, error) {
	issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	redirectURL = strings.TrimSpace(redirectURL)
	providerName = strings.TrimSpace(providerName)
	if providerName == "" {
		providerName = "Zitadel"
	}
	if issuer == "" && clientID == "" && clientSecret == "" && redirectURL == "" {
		return &oidcLogin{providerName: providerName}, nil
	}
	if issuer == "" || clientID == "" {
		return nil, fmt.Errorf("OIDC_ISSUER and OIDC_CLIENT_ID are required when OIDC is configured")
	}
	if redirectURL != "" {
		parsed, err := url.Parse(redirectURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return nil, fmt.Errorf("OIDC_REDIRECT_URL must be an absolute URL")
		}
	}
	login := &oidcLogin{
		providerName: providerName,
		issuer:       issuer,
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
	}
	if err := login.EnsureProvider(ctx); err != nil {
		log.Printf("oidc discovery unavailable at startup, will retry on login: %v", err)
	}
	return login, nil
}

func (o *oidcLogin) Configured() bool {
	return o != nil && o.issuer != "" && o.clientID != ""
}

func (o *oidcLogin) ProviderName() string {
	if o == nil || o.providerName == "" {
		return "SSO"
	}
	return o.providerName
}

func (o *oidcLogin) RedirectURL(_ *http.Request, _ tenantConfig, baseURL string) string {
	if o.redirectURL != "" {
		return o.redirectURL
	}
	return strings.TrimRight(baseURL, "/") + "/auth/oidc/callback"
}

func (o *oidcLogin) OAuthConfig(redirectURL string) oauth2.Config {
	return oauth2.Config{
		ClientID:     o.clientID,
		ClientSecret: o.clientSecret,
		Endpoint:     o.provider.Endpoint(),
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "email", "profile"},
	}
}

func (o *oidcLogin) EnsureProvider(ctx context.Context) error {
	if !o.Configured() {
		return fmt.Errorf("OIDC is not configured")
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.provider != nil && o.verifier != nil {
		return nil
	}
	provider, err := oidc.NewProvider(ctx, o.issuer)
	if err != nil {
		return fmt.Errorf("OIDC discovery failed")
	}
	o.provider = provider
	o.verifier = provider.Verifier(&oidc.Config{ClientID: o.clientID})
	return nil
}

func (s *oidcFlowStore) Put(state string, flow oidcFlow, ttl time.Duration) {
	flow.expiresAt = time.Now().Add(ttl)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
	s.items[state] = flow
}

func (s *oidcFlowStore) Consume(state string) (oidcFlow, bool) {
	if state == "" {
		return oidcFlow{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
	flow, ok := s.items[state]
	if !ok || flow.used || time.Now().After(flow.expiresAt) {
		delete(s.items, state)
		return oidcFlow{}, false
	}
	flow.used = true
	delete(s.items, state)
	return flow, true
}

func (s *oidcFlowStore) cleanupLocked() {
	now := time.Now()
	for state, flow := range s.items {
		if now.After(flow.expiresAt) {
			delete(s.items, state)
		}
	}
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (c *oidcUserClaims) Merge(other oidcUserClaims) {
	if c.Email == "" {
		c.Email = other.Email
	}
	if c.EmailVerified == nil {
		c.EmailVerified = other.EmailVerified
	}
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

func (m smtpMailer) SendInvite(to string, loginURL string, address string) error {
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
		"Subject: Einladung zum WEG Portal " + address,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		"Hallo,",
		"",
		"Sie wurden zum WEG Portal \"" + address + "\" eingeladen.",
		"Melden Sie sich mit dieser E-Mail-Adresse an:",
		loginURL,
		"",
		"Beim Anmelden erhalten Sie einen einmaligen Login-Link per E-Mail",
		"oder nutzen Ihren SSO-Zugang.",
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

func (c homeAssistantConfig) Statistics(ctx context.Context, start time.Time, end time.Time) ([]parkingNumericSample, []parkingNumericSample, error) {
	if c.baseURL == "" || c.token == "" || c.meterEnergyEntity == "" || c.priceEntity == "" {
		return nil, nil, errors.New("home assistant not configured")
	}
	wsURL, err := c.websocketURL()
	if err != nil {
		return nil, nil, err
	}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, nil, errors.New("home assistant websocket connection failed")
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetReadDeadline(deadline)
		_ = conn.SetWriteDeadline(deadline)
	}

	var authMessage struct {
		Type string `json:"type"`
	}
	if err := conn.ReadJSON(&authMessage); err != nil {
		return nil, nil, errors.New("home assistant websocket auth start failed")
	}
	if authMessage.Type == "auth_required" {
		if err := conn.WriteJSON(map[string]string{
			"type":         "auth",
			"access_token": c.token,
		}); err != nil {
			return nil, nil, errors.New("home assistant websocket auth failed")
		}
		if err := conn.ReadJSON(&authMessage); err != nil {
			return nil, nil, errors.New("home assistant websocket auth response failed")
		}
	}
	if authMessage.Type != "auth_ok" {
		return nil, nil, errors.New("home assistant websocket auth rejected")
	}

	const requestID = 1
	if err := conn.WriteJSON(map[string]any{
		"id":            requestID,
		"type":          "recorder/statistics_during_period",
		"start_time":    start.UTC().Format(time.RFC3339),
		"end_time":      end.UTC().Format(time.RFC3339),
		"statistic_ids": []string{c.meterEnergyEntity, c.priceEntity},
		"period":        "hour",
		"types":         []string{"state", "sum", "mean"},
	}); err != nil {
		return nil, nil, errors.New("home assistant websocket statistics request failed")
	}

	for {
		var response struct {
			ID      int                      `json:"id"`
			Type    string                   `json:"type"`
			Success bool                     `json:"success"`
			Error   map[string]any           `json:"error"`
			Result  map[string][]haStatistic `json:"result"`
		}
		if err := conn.ReadJSON(&response); err != nil {
			return nil, nil, errors.New("home assistant websocket statistics response failed")
		}
		if response.ID != requestID {
			continue
		}
		if response.Type != "result" || !response.Success {
			return nil, nil, errors.New("home assistant websocket statistics rejected")
		}
		energySamples := samplesFromStatistics(response.Result[c.meterEnergyEntity], "state", "sum")
		priceSamples := samplesFromStatistics(response.Result[c.priceEntity], "state", "mean")
		return energySamples, priceSamples, nil
	}
}

func (c homeAssistantConfig) websocketURL() (string, error) {
	parsed, err := url.Parse(c.baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("invalid home assistant base url")
	}
	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	default:
		return "", errors.New("unsupported home assistant websocket scheme")
	}
	parsed.Path = "/api/websocket"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
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
	if number, err := parseHAFloat(value); err == nil {
		switch unit {
		case "kWh":
			return formatDecimal(number, 2) + " kWh"
		case "W":
			return formatDecimal(number, 1) + " W"
		case "€/kWh", "EUR/kWh":
			return formatDecimal(number, 6) + " €/kWh"
		case "€", "EUR":
			return formatDecimal(number, 2) + " €"
		}
		if unit != "" {
			return formatDecimal(number, 2) + " " + unit
		}
		return formatDecimal(number, 2)
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

func parseHistoryStart(raw string, now time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.Local), nil
	}
	if t, err := time.ParseInLocation("2006-01-02", raw, time.Local); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, raw)
}

func formatInputFloat(value float64) string {
	return formatDecimal(value, 3)
}

func formatEUR(value float64) string {
	return formatDecimal(value, 2) + " €"
}

func formatEURPerKWh(value float64) string {
	return formatDecimal(value, 3) + " €/kWh"
}

func formatKWh(value float64) string {
	return formatDecimal(value, 2) + " kWh"
}

func formatPreciseEUR(value float64) string {
	return formatDecimal(value, 6) + " €"
}

func formatPreciseEURPerKWh(value float64) string {
	return formatDecimal(value, 6) + " €/kWh"
}

func formatPreciseKWh(value float64) string {
	return formatDecimal(value, 6) + " kWh"
}

func formatDecimal(value float64, decimals int) string {
	if decimals < 0 {
		decimals = 0
	}
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	raw := fmt.Sprintf("%.*f", decimals, value)
	parts := strings.SplitN(raw, ".", 2)
	intPart := parts[0]
	for i := len(intPart) - 3; i > 0; i -= 3 {
		intPart = intPart[:i] + "." + intPart[i:]
	}
	if decimals == 0 || len(parts) == 1 {
		return sign + intPart
	}
	return sign + intPart + "," + parts[1]
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
		return "BEZAHLT"
	}
	return "OFFEN"
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
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; connect-src 'self' https://fonts.googleapis.com https://fonts.gstatic.com; form-action 'self'; base-uri 'self'; frame-ancestors 'none'")
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
			authMethods, err := normalizeAuthMethods(profile.AuthMethods)
			if err != nil {
				return nil, err
			}
			profile.AuthMethods = authMethods
			out[email] = profile
		}
	}

	for email := range admins {
		if _, ok := out[email]; ok {
			continue
		}
		out[email] = userProfile{Email: email, Role: roleAdmin, Status: "Aktiv", Tenants: []string{defaultTenant}, AuthMethods: defaultAuthMethods()}
	}
	for email := range allowed {
		if _, ok := out[email]; ok {
			continue
		}
		out[email] = userProfile{Email: email, Role: roleResident, Status: "Eingeladen", Tenants: []string{defaultTenant}, AuthMethods: defaultAuthMethods()}
	}
	return out, nil
}

type inviteStore struct {
	path string
	mu   sync.Mutex
	data inviteStoreData
}

type inviteStoreData struct {
	Invites []userProfile `json:"invites"`
}

func newInviteStore(path string) (*inviteStore, error) {
	store := &inviteStore{path: path, data: inviteStoreData{Invites: []userProfile{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read invite data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid invite data")
	}
	return store, nil
}

func (s *inviteStore) Get(email string) (userProfile, bool) {
	email = normalizeEmail(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, profile := range s.data.Invites {
		if normalizeEmail(profile.Email) == email {
			return profile, true
		}
	}
	return userProfile{}, false
}

func (s *inviteStore) List() []userProfile {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]userProfile(nil), s.data.Invites...)
}

// Add persists a new invite. Returns false (no error) when the email is already
// invited. Callers must ensure the email is not already in the env directory.
func (s *inviteStore) Add(profile userProfile) (bool, error) {
	profile.Email = normalizeEmail(profile.Email)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.data.Invites {
		if normalizeEmail(existing.Email) == profile.Email {
			return false, nil
		}
	}
	s.data.Invites = append(s.data.Invites, profile)
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *inviteStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("could not create invite data directory")
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode invite data")
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("could not write invite data")
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("could not replace invite data")
	}
	return nil
}

// Update replaces the invite keyed by oldEmail with updated. Returns false (no
// error) when oldEmail is not a persisted invite. When the email changes it
// must not collide with another invite (caller also checks the env directory).
func (s *inviteStore) Update(oldEmail string, updated userProfile) (bool, error) {
	oldEmail = normalizeEmail(oldEmail)
	updated.Email = normalizeEmail(updated.Email)
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i, existing := range s.data.Invites {
		if normalizeEmail(existing.Email) == oldEmail {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false, nil
	}
	if updated.Email != oldEmail {
		for i, existing := range s.data.Invites {
			if i != idx && normalizeEmail(existing.Email) == updated.Email {
				return false, fmt.Errorf("email already invited")
			}
		}
	}
	s.data.Invites[idx] = updated
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

// Delete removes the invite for email. Returns whether one was removed.
func (s *inviteStore) Delete(email string) (bool, error) {
	email = normalizeEmail(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.data.Invites[:0]
	removed := false
	for _, existing := range s.data.Invites {
		if normalizeEmail(existing.Email) == email {
			removed = true
			continue
		}
		kept = append(kept, existing)
	}
	if !removed {
		return false, nil
	}
	s.data.Invites = kept
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

type activityStore struct {
	path string
	mu   sync.Mutex
	data map[string]activityRecord
}

type activityRecord struct {
	LastLogin  time.Time `json:"last_login"`
	AuthMethod string    `json:"auth_method,omitempty"`
}

func newActivityStore(path string) (*activityStore, error) {
	store := &activityStore{path: path, data: map[string]activityRecord{}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read activity data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid activity data")
	}
	if store.data == nil {
		store.data = map[string]activityRecord{}
	}
	return store, nil
}

// Touch records a successful login. Best-effort: callers log failures but do
// not block login on a persistence error.
func (s *activityStore) Touch(email string, at time.Time, authMethod string) error {
	email = normalizeEmail(email)
	if email == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[email] = activityRecord{LastLogin: at.UTC(), AuthMethod: authMethod}
	return s.saveLocked()
}

func (s *activityStore) Get(email string) (activityRecord, bool) {
	email = normalizeEmail(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.data[email]
	return rec, ok
}

func (s *activityStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("could not create activity data directory")
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode activity data")
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("could not write activity data")
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("could not replace activity data")
	}
	return nil
}

func normalizeRole(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "admin", "administrator", "platform-admin", "platform_admin":
		return roleAdmin
	case "verwalter", "verwaltung", "hausverwaltung", "manager", "property-manager", "property_manager", "property manager":
		return roleManager
	case "eigentuemer", "eigentümer", "wohnungseigentuemer", "wohnungseigentümer", "owner", "homeowner", "property-owner", "property_owner":
		return roleOwner
	case "mieter", "tenant", "renter", "lessee":
		return roleRenter
	case "beirat", "board", "advisory-board", "advisory_board", "committee":
		return roleBeirat
	case "bewohner", "resident", "user":
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

func defaultAuthMethods() []string {
	return []string{authMethodEmail, authMethodOIDC}
}

func normalizeAuthMethods(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return defaultAuthMethods(), nil
	}
	seen := map[string]struct{}{}
	out := []string{}
	for _, item := range raw {
		method := normalizeAuthMethod(item)
		if method == "" {
			return nil, fmt.Errorf("invalid auth method %q", item)
		}
		if _, ok := seen[method]; ok {
			continue
		}
		seen[method] = struct{}{}
		out = append(out, method)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one auth method is required")
	}
	sort.Strings(out)
	return out, nil
}

func normalizeAuthMethod(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case authMethodEmail, "mail", "magic", "magic-link", "magic_link":
		return authMethodEmail
	case authMethodOIDC, "sso", "zitadel", "citatel":
		return authMethodOIDC
	default:
		return ""
	}
}

func authMethodsLabel(methods []string) string {
	return strings.Join(authMethodsLabelList(methods), ", ")
}

func authMethodsLabelList(methods []string) []string {
	normalized, err := normalizeAuthMethods(methods)
	if err != nil {
		return []string{"Ungültig"}
	}
	labels := make([]string, 0, len(normalized))
	for _, method := range normalized {
		switch method {
		case authMethodEmail:
			labels = append(labels, "E-Mail-Link")
		case authMethodOIDC:
			labels = append(labels, "Zitadel SSO")
		default:
			labels = append(labels, method)
		}
	}
	return labels
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
	return strings.Join(permissionLabelList(permissions), ", ")
}

func permissionLabelList(permissions []string) []string {
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
		return []string{"Standard"}
	}
	return labels
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
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=Spectral:wght@400;500;600;700&display=swap" rel="stylesheet">
  <style>
    :root {
      color-scheme: light;
      --ink:#20251f; --muted:#6b6f63; --soft:#9a9485;
      --line:#e7e0d2; --paper:#f7f3ea; --panel:#fffefb;
      --gold:#c8993f; --gold-ink:#8a7b3f; --gold-light:#e7c574;
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    * { box-sizing: border-box; }
    body { margin: 0; color: var(--ink); background: #10160f; }
    .hero { position: relative; min-height: 100vh; overflow: hidden; display: grid; grid-template-rows: auto 1fr auto; }
    .hero::before {
      content: ""; position: absolute; inset: -16px;
      background: url('/assets/jhw22-hero.jpg') center 42% / cover no-repeat;
      filter: blur(3px) brightness(.74) saturate(.95); transform: scale(1.05); z-index: -2;
    }
    .hero::after {
      content: ""; position: absolute; inset: 0;
      background: linear-gradient(180deg, rgba(16,22,16,.52) 0%, rgba(16,22,16,.3) 34%, rgba(16,22,16,.6) 76%, rgba(16,22,16,.9) 100%);
      z-index: -1;
    }
    header { display: flex; justify-content: space-between; align-items: center; gap: 24px; padding: 28px clamp(20px,5vw,72px); color: #fff; }
    .brand { display: inline-flex; align-items: center; gap: 12px; text-decoration: none; color: #fff; }
    .mark { min-width: 46px; height: 40px; border-radius: 8px; background: rgba(255,255,255,.16); border: 1px solid rgba(255,255,255,.4); backdrop-filter: blur(6px); display: grid; place-items: center; padding: 0 9px; color: #fff; font-weight: 700; font-size: 13px; }
    .brand .name { font-family: Spectral, serif; font-weight: 600; font-size: 17px; }
    nav { display: flex; gap: 24px; color: rgba(255,255,255,.92); font-size: 14px; font-weight: 600; }
    nav a { color: inherit; text-decoration: none; padding-bottom: 4px; border-bottom: 1px solid rgba(231,197,116,.65); }
    nav a:hover { color: #fff; border-bottom-color: var(--gold-light); }
    main { display: grid; grid-template-columns: minmax(0,1.1fr) minmax(320px,420px); gap: clamp(28px,6vw,64px); align-items: end; padding: 0 clamp(20px,5vw,72px) clamp(40px,8vh,72px); }
    .copy { max-width: 760px; color: #fff; }
    .eyebrow { font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: .2em; color: var(--gold-light); margin-bottom: 18px; }
    h1 { margin: 0; font-family: Spectral, serif; font-weight: 500; font-size: clamp(46px,7vw,72px); line-height: 1.0; letter-spacing: -.01em; text-shadow: 0 2px 30px rgba(0,0,0,.3); }
    .lead { max-width: 440px; margin: 24px 0 0; font-size: clamp(17px,2vw,19px); line-height: 1.55; color: rgba(255,255,255,.9); }
    .meta { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 18px; margin-top: 34px; max-width: 620px; }
    .meta div { border-left: 1px solid rgba(231,197,116,.58); padding-left: 16px; min-width: 0; }
    .meta strong { display: block; font-weight: 700; font-size: 14px; color: #fff; margin-bottom: 6px; }
    .meta span { color: rgba(255,255,255,.78); font-size: 13px; line-height: 1.4; }
    .login { background: var(--paper); border-radius: 14px; padding: 30px; box-shadow: 0 28px 70px rgba(0,0,0,.42); }
    .login h2 { margin: 0; font-family: Spectral, serif; font-weight: 600; font-size: 26px; }
    .login p { color: var(--muted); line-height: 1.5; margin: 11px 0 22px; font-size: 14.5px; }
    label { display: block; font-size: 11.5px; font-weight: 700; text-transform: uppercase; letter-spacing: .07em; color: var(--gold-ink); margin-bottom: 8px; }
    input { width: 100%; border: 1px solid #e2dac9; border-radius: 10px; padding: 14px 15px; font: inherit; background: #fffefb; color: var(--ink); }
    button { width: 100%; border: 0; border-radius: 10px; padding: 15px 16px; margin-top: 13px; font: inherit; font-weight: 700; color: #fff; background: var(--ink); cursor: pointer; }
    button:hover { background: #000; }
    .notice { border: 1px solid rgba(32,37,31,.16); background: rgba(200,153,63,.1); color: #6a5320; border-radius: 10px; padding: 12px 14px; font-size: 14px; line-height: 1.4; margin-bottom: 16px; }
    .notice.warn { border-color: rgba(173,92,27,.22); background: rgba(231,197,116,.2); color: #6c491a; }
    .dev-link { display: block; border: 1px solid var(--line); background: #fffefb; color: var(--ink); border-radius: 10px; padding: 12px 14px; margin: -4px 0 16px; text-align: center; text-decoration: none; font-size: 14px; font-weight: 700; }
    .dev-link:hover { border-color: var(--gold); }
    .sso-button { display: flex; align-items: center; justify-content: center; min-height: 48px; border-radius: 10px; background: var(--ink); color: #fff; text-decoration: none; font-weight: 700; margin-bottom: 14px; }
    .sso-button:hover { background: #000; }
    .divider { display: grid; grid-template-columns: 1fr auto 1fr; align-items: center; gap: 10px; color: var(--muted); font-size: 13px; margin: 12px 0; }
    .divider::before, .divider::after { content: ""; height: 1px; background: var(--line); }
    .foot-note { margin: 16px 0 0; font-size: 13px; line-height: 1.4; color: var(--soft); }
    footer { padding: 20px clamp(20px,5vw,72px) 26px; color: rgba(255,255,255,.85); font-weight: 500; font-size: 14px; }
    footer .version { margin-left: 8px; color: rgba(255,255,255,.54); font-size: 12px; }
    @media (max-width: 860px) {
      nav { display: none; }
      main { grid-template-columns: 1fr; align-items: start; gap: 28px; }
      .meta { display: none; }
      h1 { font-size: clamp(40px,12vw,56px); }
    }
  </style>
</head>
<body>
  <section class="hero">
    <header>
      <a class="brand" href="/" aria-label="WEG Portal Startseite"><span class="mark">WEG</span><span class="name">{{.Tenant.Address}}</span></a>
      <nav aria-label="Seitennavigation">
        <a href="#login">Anmelden</a>
      </nav>
    </header>
    <main>
      <div class="copy">
        <div class="eyebrow">WEG Portal</div>
        <h1>Alles rund um unser gemeinsames Haus.</h1>
        <p class="lead">Der private digitale Eingang für die Hausgemeinschaft, erreichbar per persönlichem E-Mail-Zugang oder SSO.</p>
        <div class="meta" aria-label="Portalüberblick">
          <div><strong>Eingeladen</strong><span>Zugang nur für freigegebene E-Mail-Adressen der Hausgemeinschaft.</span></div>
          <div><strong>Einmalig</strong><span>Anmeldung per SSO oder zeitlich begrenztem E-Mail-Link.</span></div>
          <div><strong>Parkplatz</strong><span>Verbrauch und Abrechnung bleiben im geschützten Portal.</span></div>
        </div>
      </div>
      <section id="login" class="login" aria-label="Anmeldung">
        <h2>Anmelden</h2>
        <p>{{if .OIDCConfigured}}Melden Sie sich per SSO an oder verwenden Sie einen einmaligen E-Mail-Link.{{else}}Geben Sie Ihre E-Mail-Adresse ein. Wenn sie eingeladen ist, schicken wir einen einmaligen Anmeldelink.{{end}}</p>
        {{if .OIDCConfigured}}<a class="sso-button" href="/auth/oidc/start">Mit {{.OIDCProviderName}} anmelden</a>{{end}}
        {{if and .OIDCConfigured .EmailLoginAvailable}}<div class="divider"><span>oder</span></div>{{end}}
        {{if .Sent}}
          <div class="notice">Wenn die Adresse eingeladen ist, wurde ein Link verschickt. Bitte Posteingang prüfen.</div>
          {{if not .MailConfigured}}<div class="notice warn">Mailversand ist lokal noch nicht konfiguriert. In Produktion kommt SMTP aus agenix.</div>{{end}}
          {{if .DevLoginLink}}<a class="dev-link" href="{{.DevLoginLink}}">Lokalen Dev-Login öffnen</a>{{end}}
        {{end}}
        {{if .Denied}}<div class="notice warn">Diese Adresse ist noch nicht eingeladen.</div>{{end}}
        {{if .EmailLoginAvailable}}
          <form method="post" action="/auth/request">
            <label for="email">E-Mail-Adresse</label>
            <input id="email" name="email" type="email" inputmode="email" autocomplete="email" required placeholder="name@example.com">
            <button type="submit">Anmeldelink senden</button>
          </form>
          <p class="foot-note">Link 15 Minuten gültig · privat für die Hausgemeinschaft</p>
        {{else}}
          <div class="notice">E-Mail-Anmeldelinks sind nicht aktiv. Bitte SSO verwenden.</div>
        {{end}}
      </section>
    </main>
    <footer>{{.Tenant.Address}} · Privat für die Hausgemeinschaft <span class="version">{{.AppVersion}}</span></footer>
  </section>
</body>
</html>
{{end}}

{{define "appStyles"}}
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=Spectral:wght@400;500;600;700&display=swap" rel="stylesheet">
  <style>
    :root {
      color-scheme: light;
      --ink:#20251f; --muted:#6b6f63; --soft:#9a9485;
      --line:#e7e0d2; --paper:#f7f3ea; --panel:#fffefb;
      --panel-soft:#fbf8f0; --gold:#c8993f; --gold-ink:#8a7b3f;
      --gold-light:#e7c574; --leaf:#2f6b4a; --nav:#172019; --nav-2:#20291f;
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    * { box-sizing: border-box; }
    body { margin: 0; background: var(--paper); color: var(--ink); }
    a { color: inherit; }
    button, input { font: inherit; }
    .app-shell { min-height: 100vh; display: grid; grid-template-columns: 264px minmax(0,1fr); background: var(--paper); }
    .sidebar { position: sticky; top: 0; height: 100vh; display: flex; flex-direction: column; gap: 24px; padding: 22px 16px 18px; color: rgba(255,255,255,.86); background: radial-gradient(circle at 20% 0%, rgba(255,255,255,.08), transparent 28%), var(--nav); border-right: 1px solid rgba(255,255,255,.08); }
    .side-brand { display: grid; grid-template-columns: 50px 1fr; gap: 14px; align-items: center; padding: 0 8px 12px; }
    .side-mark { width: 48px; height: 48px; border-radius: 8px; display: grid; place-items: center; color: #fff; font-weight: 800; font-size: 13px; border: 1px solid rgba(255,255,255,.43); background: rgba(255,255,255,.08); }
    .side-title { display: block; font-family: Spectral, serif; font-size: 18px; font-weight: 600; line-height: 1.1; color: #fff; text-decoration: none; }
    .side-sub { display: block; margin-top: 5px; font-size: 14px; color: rgba(255,255,255,.72); }
    .side-nav { display: grid; gap: 7px; }
    .nav-item { position: relative; min-height: 46px; display: flex; align-items: center; gap: 12px; padding: 10px 12px; border-radius: 7px; color: rgba(255,255,255,.78); text-decoration: none; font-size: 15px; font-weight: 600; }
    .nav-item:hover { color: #fff; background: rgba(255,255,255,.06); }
    .nav-item.active { color: #fff; background: rgba(255,255,255,.08); }
    .nav-item.active::before { content: ""; position: absolute; left: -16px; top: 0; bottom: 0; width: 4px; background: var(--gold); }
    .nav-item.disabled { color: rgba(255,255,255,.38); cursor: default; }
    .nav-item.disabled:hover { background: transparent; }
    .nav-icon { width: 23px; height: 23px; display: grid; place-items: center; flex: 0 0 auto; color: currentColor; }
    .nav-icon svg { width: 22px; height: 22px; stroke: currentColor; stroke-width: 1.9; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .nav-label { min-width: 0; }
    .nav-badge { margin-left: auto; min-width: 25px; height: 22px; display: inline-flex; align-items: center; justify-content: center; border-radius: 999px; padding: 0 7px; background: var(--gold); color: #172019; font-size: 11px; font-weight: 900; line-height: 1; }
    .side-foot { margin-top: auto; border-top: 1px solid rgba(255,255,255,.16); padding: 18px 8px 0; display: grid; gap: 14px; }
    .side-user { display: grid; grid-template-columns: 42px 1fr; gap: 12px; align-items: center; }
    .avatar { width: 42px; height: 42px; border-radius: 50%; display: grid; place-items: center; background: var(--gold); color: #fff; font-weight: 800; border: 1px solid rgba(255,255,255,.25); }
    .side-user strong { display: block; color: #fff; font-size: 14px; }
    .side-user span, .side-version { color: rgba(255,255,255,.64); font-size: 13px; }
    .logout-form { margin: 0; }
    .logout-button { width: 100%; min-height: 42px; display: inline-flex; align-items: center; justify-content: center; gap: 10px; border: 1px solid rgba(255,255,255,.24); border-radius: 7px; color: rgba(255,255,255,.92); background: transparent; font-weight: 700; cursor: pointer; }
    .logout-button:hover { border-color: var(--gold); color: #fff; }
    .app-main { min-width: 0; padding-bottom: 58px; }
    .content-top { height: 64px; display: flex; align-items: center; justify-content: space-between; gap: 14px; padding: 0 clamp(28px,4vw,44px); border-bottom: 1px solid var(--line); background: rgba(255,254,251,.72); }
    .crumb { display: inline-flex; align-items: center; gap: 10px; color: var(--muted); font-size: 14px; }
    .crumb svg, .action svg { width: 18px; height: 18px; stroke: currentColor; fill: none; stroke-width: 1.9; stroke-linecap: round; stroke-linejoin: round; }
    .page-actions { display: flex; align-items: center; gap: 10px; }
    .page { width: min(1220px,100%); margin: 0 auto; padding: 34px clamp(28px,4vw,44px) 0; display: grid; gap: 24px; }
    .page.wide { width: min(1280px,100%); }
    h1 { margin: 0; font-family: Spectral, serif; font-weight: 500; font-size: clamp(42px,5vw,54px); line-height: 1; }
    h2 { margin: 0; font-family: Spectral, serif; font-weight: 600; font-size: 23px; line-height: 1.1; }
    h3 { margin: 0; font-family: Spectral, serif; font-weight: 600; font-size: 20px; line-height: 1.2; }
    p { margin: 0; }
    .lede { margin-top: 14px; color: var(--muted); font-size: 16px; line-height: 1.55; }
    .muted { color: var(--muted); line-height: 1.5; }
    .subtle-note { margin-top: 6px; max-width: 760px; font-size: 14.5px; }
    .kicker { color: var(--gold-ink); font-size: 12px; font-weight: 800; letter-spacing: .12em; text-transform: uppercase; border-bottom: 2px solid var(--ink); padding-bottom: 11px; margin-bottom: 20px; }
    .panel { background: var(--panel); border: 1px solid var(--line); border-radius: 8px; padding: 24px; box-shadow: 0 12px 30px rgba(32,37,31,.04); }
    .panel.compact { padding: 18px; }
    .button, button.action { min-height: 38px; display: inline-flex; align-items: center; justify-content: center; gap: 8px; border: 1px solid var(--line); background: var(--panel); border-radius: 7px; color: var(--ink); padding: 8px 13px; font-weight: 700; line-height: 1.15; text-decoration: none; cursor: pointer; white-space: nowrap; }
    .button:hover, button.action:hover { border-color: var(--gold); }
    .button.primary, button.primary { background: var(--ink); border-color: var(--ink); color: #fff; }
    .button.small, button.small { min-height: 31px; padding: 6px 10px; font-size: 12px; }
    .button.ghost { background: transparent; }
    .banner { position: relative; height: 128px; overflow: hidden; border-bottom: 1px solid var(--line); background: #e9e4d7; }
    .banner::before { content: ""; position: absolute; inset: 0; background: url('/assets/jhw22-hero.jpg') center 47% / cover no-repeat; }
    .banner::after { content: ""; position: absolute; inset: 0; background: linear-gradient(90deg, rgba(23,32,25,.1), rgba(247,243,234,.72) 76%, rgba(247,243,234,.92)); }
    .banner-kicker { position: absolute; left: clamp(28px,4vw,44px); bottom: 18px; color: var(--gold-ink); font-size: 12px; font-weight: 800; letter-spacing: .18em; text-transform: uppercase; }
    .home-grid { display: grid; grid-template-columns: minmax(0,1.35fr) minmax(340px,.85fr); gap: 22px; align-items: start; }
    .entries { display: grid; gap: 22px; }
    .entry + .entry { border-top: 1px solid var(--line); padding-top: 22px; }
    .entry-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; }
    .entry h3 a { color: inherit; text-decoration: none; }
    .entry h3 a:hover { color: var(--gold-ink); }
    .entry p, .entry-body { margin-top: 8px; color: #5c5f54; line-height: 1.6; }
    .entry-meta { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin-top: 10px; color: var(--soft); font-size: 12.5px; }
    .entry-actions { display: flex; gap: 8px; flex-wrap: wrap; justify-content: flex-end; }
    .archive-tools { display: grid; gap: 12px; margin: -4px 0 20px; }
    .filter-form { display: grid; grid-template-columns: minmax(220px,1fr) auto; gap: 10px; align-items: end; }
    .filter-form label { margin: 0; }
    .filter-tabs { display: flex; gap: 8px; flex-wrap: wrap; }
    .filter-tab { min-height: 34px; display: inline-flex; align-items: center; justify-content: center; border: 1px solid var(--line); border-radius: 999px; padding: 6px 12px; color: var(--muted); background: var(--panel); text-decoration: none; font-size: 13px; font-weight: 800; }
    .filter-tab.active, .filter-tab:hover { border-color: var(--gold); color: var(--ink); background: rgba(200,153,63,.14); }
    .quick-list { display: grid; }
    .quick-row { display: grid; grid-template-columns: 30px minmax(0,1fr) auto; gap: 12px; align-items: center; padding: 13px 0; border-bottom: 1px solid var(--line); color: inherit; text-decoration: none; }
    button.quick-row { width: 100%; border: 0; border-bottom: 1px solid var(--line); background: transparent; font: inherit; text-align: left; cursor: pointer; }
    button.quick-row:hover { color: var(--gold-ink); }
    .quick-row:last-child { border-bottom: 0; }
    .quick-row > div { min-width: 0; }
    .quick-row svg, .info-icon svg { width: 24px; height: 24px; stroke: currentColor; stroke-width: 1.8; fill: none; stroke-linecap: round; stroke-linejoin: round; color: var(--ink); }
    .quick-row h3 { font-size: 18px; }
    .quick-row p { margin-top: 3px; color: var(--soft); font-size: 13.5px; line-height: 1.35; overflow-wrap: anywhere; }
    .quick-row.disabled { cursor: default; }
    .quick-row.disabled h3, .quick-row.disabled svg { color: var(--muted); }
    .quick-row .pill { justify-self: end; }
    .quick-arrow { color: var(--gold-ink); font-size: 24px; line-height: 1; }
    .info-row { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 18px; }
    .info-card { display: grid; grid-template-columns: 66px 1fr auto; gap: 16px; align-items: center; padding: 18px 20px; }
    .info-icon { width: 64px; height: 64px; display: grid; place-items: center; border-radius: 8px; background: #f3eee5; }
    .info-card p { margin-top: 5px; color: var(--soft); font-size: 13.5px; line-height: 1.35; }
    .status-strip { display: grid; gap: 14px; }
    .rule { display: flex; align-items: center; justify-content: space-between; gap: 14px; flex-wrap: wrap; color: var(--ink); line-height: 1.5; }
    .rule p { flex: 1 1 640px; min-width: 0; }
    .rule .pill { margin-left: auto; }
    .pill { display: inline-flex; align-items: center; min-height: 26px; border-radius: 999px; padding: 3px 10px; font-size: 12px; font-weight: 800; background: rgba(200,153,63,.16); color: #8a6a1f; white-space: nowrap; }
    .pill.ok { background: rgba(47,107,74,.12); color: var(--leaf); }
    .pill.ok::before { content: ""; width: 8px; height: 8px; border-radius: 50%; background: currentColor; margin-right: 8px; }
    .pill.dringend { background: rgba(158,42,43,.12); color: #9e2a2b; }
    .pill.termin { background: rgba(47,107,74,.12); color: var(--leaf); }
    .pill.wartung { background: rgba(173,92,27,.14); color: #8a551f; }
    .pill.info { background: rgba(200,153,63,.16); color: #8a6a1f; }
    .pill.unread { background: var(--gold); color: #172019; }
    .dialog { border: 1px solid var(--line); border-radius: 10px; padding: 0; width: min(680px, calc(100vw - 28px)); color: var(--ink); background: var(--panel); box-shadow: 0 28px 70px rgba(0,0,0,.34); }
    .dialog::backdrop { background: rgba(23,32,25,.42); }
    .dialog form { margin: 0; }
    .dialog-head { display: flex; justify-content: space-between; align-items: center; gap: 14px; padding: 20px 22px; border-bottom: 1px solid var(--line); }
    .dialog-head h2 { font-size: 25px; }
    .dialog-close { width: 34px; height: 34px; border: 1px solid var(--line); border-radius: 7px; background: var(--panel-soft); color: var(--ink); font-size: 22px; line-height: 1; cursor: pointer; }
    .dialog-body { display: grid; gap: 14px; padding: 20px 22px 22px; }
    .dialog-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; }
    .dialog-grid .full { grid-column: 1 / -1; }
    textarea { width: 100%; border: 1px solid #e2dac9; border-radius: 7px; min-height: 150px; padding: 10px 12px; color: var(--ink); background: #fffefb; resize: vertical; font: inherit; line-height: 1.45; }
    select { width: 100%; border: 1px solid #e2dac9; border-radius: 7px; min-height: 42px; padding: 9px 12px; color: var(--ink); background: #fffefb; font: inherit; }
    .check-row { display: inline-flex; align-items: center; gap: 8px; min-height: 42px; color: var(--ink); font-size: 14px; font-weight: 700; letter-spacing: 0; text-transform: none; }
    .check-row input { width: auto; min-height: 0; }
    .metric-grid { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 16px; }
    .metric-card, .month-card { min-width: 0; background: var(--panel-soft); border: 1px solid var(--line); border-radius: 8px; padding: 16px; }
    .metric-label, .field-label, th { color: var(--gold-ink); font-size: 11px; font-weight: 800; text-transform: uppercase; letter-spacing: .06em; }
    .metric-value { display: block; margin-top: 7px; font-family: Spectral, serif; font-size: 26px; font-weight: 600; line-height: 1.08; }
    code, .mini { color: var(--soft); font-size: 12px; line-height: 1.35; overflow-wrap: anywhere; }
    .accounting { display: grid; gap: 16px; }
    .section-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; flex-wrap: wrap; }
    .month-strip { display: grid; grid-template-columns: repeat(7,minmax(120px,1fr)); gap: 10px; }
    .month-card { display: grid; gap: 10px; color: inherit; text-decoration: none; }
    .month-card:hover { border-color: var(--gold); }
    .month-card strong { font-family: Spectral, serif; font-size: 16px; }
    .bar { height: 9px; border-radius: 999px; background: #ece5d6; overflow: hidden; }
    .bar span { display: block; height: 100%; min-width: 2px; border-radius: inherit; background: var(--gold); }
    .amount { font-weight: 800; font-variant-numeric: tabular-nums; }
    .table-wrap { overflow-x: auto; border: 1px solid var(--line); border-radius: 8px; background: var(--panel); }
    table { width: 100%; border-collapse: collapse; min-width: 980px; }
    th, td { padding: 13px 16px; border-bottom: 1px solid var(--line); vertical-align: middle; }
    th { text-align: left; background: rgba(251,248,240,.7); }
    td { font-size: 14px; }
    tbody tr:last-child td { border-bottom: 0; }
    .num { text-align: right; font-variant-numeric: tabular-nums; white-space: nowrap; }
    .month-cell strong { display: block; font-family: Spectral, serif; font-size: 17px; }
    .month-cell a { text-decoration: none; }
    .month-cell a:hover { color: var(--gold-ink); }
    .month-cell span { display: block; margin-top: 3px; color: var(--soft); font-size: 12px; }
    .row-actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
    .empty { border: 1px solid var(--line); background: var(--panel-soft); color: #5c5f54; border-radius: 8px; padding: 14px; line-height: 1.5; }
    .settings-card { max-width: 620px; display: grid; gap: 16px; }
    .form-grid { display: grid; gap: 12px; }
    label { display: grid; gap: 7px; color: var(--gold-ink); font-size: 12px; font-weight: 800; text-transform: uppercase; letter-spacing: .06em; }
    input { width: 100%; border: 1px solid #e2dac9; border-radius: 7px; min-height: 42px; padding: 9px 12px; color: var(--ink); background: #fffefb; }
    .flash { padding: 10px 13px; border-radius: 7px; font-size: 13.5px; font-weight: 700; border: 1px solid rgba(200,153,63,.28); background: rgba(200,153,63,.14); color: #8a6a1f; }
    .flash.ok { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.25); }
    .legend { border: 1px solid var(--line); border-radius: 8px; background: var(--panel-soft); padding: 14px; display: grid; grid-template-columns: repeat(auto-fit,minmax(210px,1fr)); gap: 12px; }
    .legend strong { display: block; font-family: Spectral, serif; margin-bottom: 3px; }
    .legend span { display: block; color: var(--muted); font-size: 13px; line-height: 1.4; }
    .bar-cell { min-width: 150px; }
    @media (max-width: 1120px) {
      .app-shell { grid-template-columns: 1fr; }
      .sidebar { position: relative; height: auto; padding: 16px; }
      .side-nav { grid-template-columns: repeat(auto-fit,minmax(170px,1fr)); }
      .side-foot { margin-top: 4px; grid-template-columns: 1fr auto; align-items: center; }
      .logout-form { justify-self: end; min-width: 160px; }
      .home-grid, .metric-grid, .info-row { grid-template-columns: 1fr; }
      .month-strip { grid-template-columns: repeat(auto-fit,minmax(150px,1fr)); }
    }
    @media (max-width: 680px) {
      .side-brand, .side-user { grid-template-columns: auto 1fr; }
      .side-foot { grid-template-columns: 1fr; }
      .logout-form { justify-self: stretch; }
      .content-top { height: auto; min-height: 58px; flex-direction: column; align-items: flex-start; padding-top: 12px; padding-bottom: 12px; }
      .page { padding-left: 18px; padding-right: 18px; }
      h1 { font-size: clamp(36px,12vw,48px); }
      .metric-grid { grid-template-columns: 1fr; }
      .info-card { grid-template-columns: 52px 1fr; }
      .quick-row { grid-template-columns: 28px minmax(0,1fr); }
      .quick-row .pill { grid-column: 2; justify-self: start; }
      .filter-form { grid-template-columns: 1fr; }
      .dialog-grid { grid-template-columns: 1fr; }
      .quick-arrow, .info-card .quick-arrow { display: none; }
    }
  </style>
{{end}}

{{define "sidebar"}}
  <aside class="sidebar" aria-label="Portalnavigation">
    <div class="side-brand">
      <a class="side-mark" href="/app">WEG</a>
      <div>
        <a class="side-title" href="/app">WEG Portal</a>
        <span class="side-sub">{{.Tenant.Address}}</span>
      </div>
    </div>
    <nav class="side-nav">
      <a class="nav-item {{if eq .ActivePage "home"}}active{{end}}" href="/app"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg></span><span class="nav-label">Hausüberblick</span></a>
      <a class="nav-item {{if eq .ActivePage "announcements"}}active{{end}}" href="/app/announcements"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M4 5h16v13H7l-3 3z"/><path d="M8 9h8M8 13h6"/></svg></span><span class="nav-label">Aushang</span>{{if .HasUnreadAnnouncements}}<span class="nav-badge">{{.UnreadAnnouncements}}</span>{{end}}</a>
      {{if .CanSeeParking}}<a class="nav-item {{if eq .ActivePage "parking"}}active{{end}}" href="/app/parking"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 16h14"/><path d="m7 16 1.5-5h7L17 16"/><path d="M7 16v3M17 16v3"/><path d="M7 19h1M16 19h1"/></svg></span>Parkplatznutzung</a>{{end}}
      <span class="nav-item disabled"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/><path d="M9 13h6M9 17h6"/></svg></span>Dokumente</span>
      <span class="nav-item disabled"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 18.5V6.5A2.5 2.5 0 0 1 7.5 4h9A2.5 2.5 0 0 1 19 6.5v6A2.5 2.5 0 0 1 16.5 15H10l-5 3.5z"/></svg></span>Anliegen</span>
      <span class="nav-item disabled"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 19V9M12 19V5M19 19v-7"/><path d="M3.5 19h17"/></svg></span>Abstimmungen</span>
      {{if .IsAdmin}}<a class="nav-item {{if eq .ActivePage "users"}}active{{end}}" href="/app/settings/users"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"/><path d="M3.5 20a5 5 0 0 1 10 0"/><path d="M16 11.5a2.5 2.5 0 1 0 0-5"/><path d="M17 15a4 4 0 0 1 3.5 4"/></svg></span>Benutzer &amp; Rechte</a>{{end}}
      <a class="nav-item {{if eq .ActivePage "settings"}}active{{end}}" href="/app/settings"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M12 8.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7z"/><path d="M19 12a7 7 0 0 0-.1-1l2-1.5-2-3.5-2.4 1a7 7 0 0 0-1.8-1L14.4 3h-4.8L9.3 6a7 7 0 0 0-1.8 1l-2.4-1-2 3.5 2 1.5A7 7 0 0 0 5 12a7 7 0 0 0 .1 1l-2 1.5 2 3.5 2.4-1a7 7 0 0 0 1.8 1l.3 3h4.8l.3-3a7 7 0 0 0 1.8-1l2.4 1 2-3.5-2-1.5a7 7 0 0 0 .1-1z"/></svg></span>Einstellungen</a>
    </nav>
    <div class="side-foot">
      <div class="side-user">
        <span class="avatar">{{.Initials}}</span>
        <div><strong>{{.DisplayName}}</strong><span>{{.Role}}</span></div>
      </div>
      <span class="side-version">{{.AppVersion}}</span>
      <form class="logout-form" method="post" action="/auth/logout"><button class="logout-button" type="submit">Abmelden</button></form>
    </div>
  </aside>
{{end}}

{{define "appOpen"}}
<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  {{template "appStyles" .}}
</head>
<body>
  <div class="app-shell">
    {{template "sidebar" .}}
{{end}}

{{define "appClose"}}
  </div>
</body>
</html>
{{end}}

{{define "portal"}}
{{template "appOpen" .}}
    <main class="app-main">
      <div class="content-top"><span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg>Hausüberblick</span></div>
      <div class="banner"><span class="banner-kicker">WEG Portal · {{.Tenant.Address}}</span></div>
      <section class="page">
        <div>
          <h1>Hausüberblick</h1>
          <p class="lede">Aktuelle Informationen der Hausgemeinschaft und direkte Wege zu den freigeschalteten Bereichen.</p>
        </div>
        <div class="home-grid">
          <section class="panel">
            <div class="section-head">
              <div class="kicker">Aktueller Aushang</div>
              {{if .HasUnreadAnnouncements}}<span class="pill unread">{{.UnreadAnnouncements}} neu</span>{{end}}
              <a class="button small" href="/app/announcements">Archiv öffnen</a>
            </div>
            {{if .HasAnnouncements}}
              <div class="entries">
                {{range .Announcements}}
                  <article class="entry">
                    <div class="entry-head">
                      <div>
                        <h3><a href="/app/announcements">{{.Title}}</a></h3>
                        <div class="entry-meta">
                          <span class="pill {{.CategoryClass}}">{{.Category}}</span>
                          {{if .Unread}}<span class="pill unread">neu</span>{{end}}
                          {{if .Pinned}}<span class="pill">Fixiert</span>{{end}}
                          <span>{{.PublishedAt}}</span>
                        </div>
                      </div>
                    </div>
                    <div class="entry-body">{{.BodyHTML}}</div>
                  </article>
                {{end}}
              </div>
            {{else}}
              <p class="empty">Noch keine Beiträge. Sobald die Verwaltung einen Aushang veröffentlicht, erscheint er hier.</p>
            {{end}}
          </section>
          <section class="panel">
            <div class="kicker">Schnellzugriff</div>
            <div class="quick-list">
              <a class="quick-row" href="/app/announcements"><svg viewBox="0 0 24 24"><path d="M4 5h16v13H7l-3 3z"/><path d="M8 9h8M8 13h6"/></svg><div><h3>Aushang</h3><p>Offizielle Informationen, Termine und Hinweise der Hausgemeinschaft.</p></div>{{if .HasUnreadAnnouncements}}<span class="pill unread">{{.UnreadAnnouncements}} neu</span>{{else}}<span class="quick-arrow">›</span>{{end}}</a>
              {{if .CanSeeParking}}<a class="quick-row" href="/app/parking"><svg viewBox="0 0 24 24"><path d="M5 16h14"/><path d="m7 16 1.5-5h7L17 16"/><path d="M7 16v3M17 16v3"/><path d="M7 19h1M16 19h1"/></svg><div><h3>Parkplatznutzung</h3><p>Privater Bereich für die abgestimmte Nutzung des Stellplatzes.</p></div><span class="quick-arrow">›</span></a>{{end}}
              {{if .IsAdmin}}<a class="quick-row" href="/app/settings/users"><svg viewBox="0 0 24 24"><path d="M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"/><path d="M3.5 20a5 5 0 0 1 10 0"/><path d="M16 11.5a2.5 2.5 0 1 0 0-5"/><path d="M17 15a4 4 0 0 1 3.5 4"/></svg><div><h3>Benutzer &amp; Rechte</h3><p>Einladungen, Rollen und Zugriff der Hausgemeinschaft verwalten.</p></div><span class="quick-arrow">›</span></a>{{end}}
              {{if .CanManageAnnouncements}}<a class="quick-row" href="/app/announcements"><svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg><div><h3>Aushang verwalten</h3><p>Beiträge verfassen, fixieren, planen und löschen.</p></div><span class="quick-arrow">›</span></a>{{end}}
            </div>
          </section>
        </div>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "announcements"}}
{{template "appOpen" .}}
    <script src="/assets/announcements.js" defer></script>
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M4 5h16v13H7l-3 3z"/><path d="M8 9h8M8 13h6"/></svg><span>/</span><span>Aushang</span></span>
        {{if .CanManageAnnouncements}}<div class="page-actions"><button class="button primary" type="button" data-dialog="announcement-create"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 5v14M5 12h14" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Neu verfassen</button></div>{{end}}
      </div>
      <section class="page">
        <div>
          <h1>Aushang</h1>
          <p class="lede">Offizielle Informationen, Termine und Hinweise der Hausgemeinschaft.</p>
        </div>
        {{if .AnnounceMsg}}<p class="flash ok">{{.AnnounceMsg}}</p>{{end}}
        <div class="home-grid">
          <section class="panel">
            <div class="section-head">
              <div class="kicker">Archiv</div>
              {{if .HasAnnouncements}}<span class="pill">{{len .Announcements}} Treffer</span>{{end}}
            </div>
            <div class="archive-tools">
              <form class="filter-form" method="get" action="/app/announcements">
                {{if .SelectedCategory}}<input type="hidden" name="category" value="{{.SelectedCategory}}">{{end}}
                <label for="announcement-search">Suche<input id="announcement-search" name="q" value="{{.SearchQuery}}" placeholder="Titel, Text oder Kategorie"></label>
                <button class="button" type="submit">Suchen</button>
              </form>
              <div class="filter-tabs" aria-label="Aushang-Kategorien">
                {{range .CategoryFilters}}<a class="filter-tab {{if .Active}}active{{end}}" href="{{.URL}}">{{.Label}}</a>{{end}}
              </div>
            </div>
            {{if .HasAnnouncements}}
              <div class="entries">
                {{range .Announcements}}
                  <article class="entry">
                    <div class="entry-head">
                      <div>
                        <h3>{{.Title}}</h3>
                        <div class="entry-meta">
                          <span class="pill {{.CategoryClass}}">{{.Category}}</span>
                          {{if .Unread}}<span class="pill unread">neu</span>{{end}}
                          {{if .Pinned}}<span class="pill">Fixiert</span>{{end}}
                          {{if .Status}}<span class="pill">{{.Status}}</span>{{end}}
                          <span>{{.PublishedAt}}</span>
                          {{if .HasExpiresAt}}<span>bis {{.ExpiresAt}}</span>{{end}}
                        </div>
                      </div>
                    </div>
                    <div class="entry-body">{{.BodyHTML}}</div>
                  </article>
                {{end}}
              </div>
            {{else}}
              {{if .HasAnyAnnouncements}}
                <p class="empty">Keine Beiträge für diese Suche oder Kategorie.</p>
              {{else}}
                <p class="empty">Noch keine Beiträge.</p>
              {{end}}
            {{end}}
          </section>

          <section class="panel">
            <div class="kicker">Aushang verwalten</div>
            {{if .CanManageAnnouncements}}
              <div class="quick-list">
                <button class="quick-row" type="button" data-dialog="announcement-create">
                  <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>
                  <div><h3>Neu verfassen</h3><p>Kategorie, Fixierung, Veröffentlichung und Ablaufdatum setzen.</p></div>
                  <span class="quick-arrow">›</span>
                </button>
                {{if .HasAllAnnouncements}}
                  {{range .AllAnnouncements}}
                    <div class="quick-row">
                      <svg viewBox="0 0 24 24"><path d="M4 5h16v13H7l-3 3z"/><path d="M8 9h8M8 13h6"/></svg>
                      <div><h3>{{.Title}}</h3><p>{{.Category}} · {{.PublishedAt}}{{if .Status}} · {{.Status}}{{end}}</p></div>
                      <span class="entry-actions">
                        <button class="button small" type="button" data-dialog="{{.EditDialogID}}">Bearbeiten</button>
                        <form method="post" action="/app/announcements/delete" data-confirm="{{.DeleteConfirmLabel}}">
                          <input type="hidden" name="id" value="{{.ID}}">
                          <button class="button small" type="submit">Löschen</button>
                        </form>
                      </span>
                    </div>
                    <dialog id="{{.EditDialogID}}" class="dialog">
                      <form method="post" action="/app/announcements/edit">
                        <input type="hidden" name="id" value="{{.ID}}">
                        <div class="dialog-head">
                          <h2>Aushang bearbeiten</h2>
                          <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
                        </div>
                        <div class="dialog-body">
                          <div class="dialog-grid">
                            <label class="full" for="title-{{.ID}}">Titel<input id="title-{{.ID}}" name="title" value="{{.Title}}" required maxlength="140"></label>
                            <label for="category-{{.ID}}">Kategorie<select id="category-{{.ID}}" name="category">
                              <option value="Info"{{if eq .Category "Info"}} selected{{end}}>Info</option>
                              <option value="Termin"{{if eq .Category "Termin"}} selected{{end}}>Termin</option>
                              <option value="Wartung"{{if eq .Category "Wartung"}} selected{{end}}>Wartung</option>
                              <option value="Dringend"{{if eq .Category "Dringend"}} selected{{end}}>Dringend</option>
                            </select></label>
                            <label for="published-{{.ID}}">Veröffentlichen<input id="published-{{.ID}}" type="datetime-local" name="published_at" value="{{.PublishedAtInput}}"></label>
                            <label for="expires-{{.ID}}">Ablauf optional<input id="expires-{{.ID}}" type="datetime-local" name="expires_at" value="{{.ExpiresAtInput}}"></label>
                            <label class="check-row"><input type="checkbox" name="pinned" value="true"{{if .PinnedChecked}} checked{{end}}> oben fixieren</label>
                            <label class="full" for="body-{{.ID}}">Text<textarea id="body-{{.ID}}" name="body" required>{{.Body}}</textarea></label>
                          </div>
                          <button class="button primary" type="submit">Speichern</button>
                        </div>
                      </form>
                    </dialog>
                  {{end}}
                {{else}}
                  <p class="empty">Noch kein Aushang gespeichert.</p>
                {{end}}
              </div>
            {{else}}
              <p class="empty">Veröffentlichen und Bearbeiten ist der Verwaltung vorbehalten.</p>
            {{end}}
          </section>
        </div>
      </section>

      {{if .CanManageAnnouncements}}
      <dialog id="announcement-create" class="dialog">
        <form method="post" action="/app/announcements">
          <div class="dialog-head">
            <h2>Neu verfassen</h2>
            <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
          </div>
          <div class="dialog-body">
            <div class="dialog-grid">
              <label class="full" for="announcement-title">Titel<input id="announcement-title" name="title" required maxlength="140"></label>
              <label for="announcement-category">Kategorie<select id="announcement-category" name="category">
                <option value="Info">Info</option>
                <option value="Termin">Termin</option>
                <option value="Wartung">Wartung</option>
                <option value="Dringend">Dringend</option>
              </select></label>
              <label for="announcement-published">Veröffentlichen<input id="announcement-published" type="datetime-local" name="published_at" value="{{.NowInput}}"></label>
              <label for="announcement-expires">Ablauf optional<input id="announcement-expires" type="datetime-local" name="expires_at"></label>
              <label class="check-row"><input type="checkbox" name="pinned" value="true"> oben fixieren</label>
              <label class="full" for="announcement-body">Text<textarea id="announcement-body" name="body" required></textarea></label>
            </div>
            <button class="button primary" type="submit">Veröffentlichen</button>
          </div>
        </form>
      </dialog>
      {{end}}
    </main>
{{template "appClose" .}}
{{end}}

{{define "parking"}}
{{template "appOpen" .}}
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><span>Parkplatznutzung</span></span>
        <div class="page-actions">
          <a class="button" href="/app/parking"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M4 4v6h6" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"/><path d="M20 20v-6h-6" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"/><path d="M5 10a7 7 0 0 1 12-3M19 14a7 7 0 0 1-12 3" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Aktualisieren</a>
          {{if .IsAdmin}}<a class="button" href="/app/parking/settings"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 8.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7z" fill="none" stroke="currentColor" stroke-width="1.9"/><path d="M19 12a7 7 0 0 0-.1-1l2-1.5-2-3.5-2.4 1a7 7 0 0 0-1.8-1L14.4 3h-4.8L9.3 6a7 7 0 0 0-1.8 1l-2.4-1-2 3.5 2 1.5A7 7 0 0 0 5 12a7 7 0 0 0 .1 1l-2 1.5 2 3.5 2.4-1a7 7 0 0 0 1.8 1l.3 3h4.8l.3-3a7 7 0 0 0 1.8-1l2.4 1 2-3.5-2-1.5a7 7 0 0 0 .1-1z" fill="none" stroke="currentColor" stroke-width="1.9"/></svg>Abrechnung konfigurieren</a>{{end}}
        </div>
      </div>
      <section class="page wide">
        <div>
          <h1>Parkplatznutzung</h1>
          <p class="lede">Private Lade- und Stellplatzabrechnung für die persönlich abgestimmte Nutzung.</p>
          <p class="muted subtle-note">Sichtbar nur für berechtigte Personen und gedacht für die private Abstimmung der Stellplatz- und Lade-Nutzung.</p>
        </div>

        <section class="panel status-strip">
          <div class="rule">
            <p>Nutzung nur nach persönlicher Absprache. Die Monatswerte berechnen sich stündlich aus Zählerdifferenz, aWATTar-Preis und Netzgebühr.</p>
            {{if .Telemetry.Configured}}<span class="pill ok">Home Assistant aktiv</span>{{end}}
          </div>
          {{if .Telemetry.Connected}}
            <div class="metric-grid">
              {{range .Telemetry.Metrics}}
                <div class="metric-card">
                  <span class="metric-label">{{.Label}}</span>
                  <strong class="metric-value">{{.Value}}</strong>
                  <code>{{.Detail}}</code>
                </div>
              {{end}}
            </div>
          {{else}}
            <p class="empty">{{.Telemetry.Message}}</p>
          {{end}}
        </section>

        <section class="panel accounting">
          <div class="section-head">
            <div>
              <h2>Monatsabrechnung</h2>
              <p class="muted">{{.Accounting.Message}}</p>
            </div>
          </div>
          {{if .Accounting.HasMonths}}
            <div class="month-strip">
              {{range .Accounting.Months}}
                <a class="month-card" href="{{.DetailPath}}">
                  <strong>{{.MonthLabel}}</strong>
                  <div class="bar"><span style="width: {{.ChartPercent}}%;"></span></div>
                  <span class="amount">{{.TotalCost}}</span>
                </a>
              {{end}}
            </div>
            <div class="table-wrap">
              <table aria-label="Monatsabrechnung Parkplatznutzung">
                <thead>
                  <tr>
                    <th>Monat</th>
                    <th class="num">Verbrauch</th>
                    <th class="num">Ø aWATTar</th>
                    <th class="num">Ø effektiv</th>
                    <th class="num">Strom</th>
                    <th class="num">Netzgeb.</th>
                    <th class="num">Summe</th>
                    <th>Status</th>
                    <th>Aktionen</th>
                  </tr>
                </thead>
                <tbody>
                  {{range .Accounting.Months}}
                    <tr>
                      <td class="month-cell"><a href="{{.DetailPath}}"><strong>{{.MonthLabel}}</strong></a>{{if .Partial}}<span>Teilmonat</span>{{end}}<span>{{.HourCount}} Stunden</span></td>
                      <td class="num">{{.KWh}}</td>
                      <td class="num">{{.AverageAwattar}}</td>
                      <td class="num">{{.EffectivePrice}}</td>
                      <td class="num">{{.EnergyCost}}</td>
                      <td class="num">{{.GridCost}}</td>
                      <td class="num amount">{{.TotalCost}}</td>
                      <td><span class="pill {{if .Paid}}ok{{end}}">{{.PaidLabel}}</span></td>
                      <td>
                        <div class="row-actions">
                          <a class="button small" href="{{.DetailPath}}">Details</a>
                          {{if $.IsAdmin}}
                            <form method="post" action="/app/parking/month">
                              <input type="hidden" name="month" value="{{.Month}}">
                              <input type="hidden" name="paid" value="{{.TogglePaidValue}}">
                              <button class="button small" type="submit">{{.ToggleLabel}}</button>
                            </form>
                          {{end}}
                        </div>
                      </td>
                    </tr>
                  {{end}}
                </tbody>
              </table>
            </div>
          {{else}}
            <p class="empty">Noch keine Monatswerte. Sobald zwei Zählerstände und mindestens ein aWATTar-Preis vorliegen, erscheint hier die erste Abrechnung.</p>
          {{end}}
        </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "parkingMonth"}}
{{template "appOpen" .}}
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/parking">Parkplatznutzung</a><span>/</span><span>{{.Detail.MonthLabel}}</span></span>
        <div class="page-actions"><a class="button" href="{{.Detail.BackPath}}">Monate</a></div>
      </div>
      <section class="page wide">
        <div>
          <h1>{{.Detail.MonthLabel}}</h1>
          <p class="lede">{{.Detail.Message}}</p>
        </div>
        <section class="panel status-strip">
          <div class="rule">
            <span class="pill">Netzgebühr {{.Detail.GridFeeLabel}}</span>
            {{if .Detail.LastSampleLabel}}<span class="mini">Letzter Zählerwert: {{.Detail.LastSampleLabel}}</span>{{end}}
          </div>
          {{if .Detail.Summary.Month}}
            <div class="metric-grid">
              <div class="metric-card"><span class="metric-label">Verbrauch</span><strong class="metric-value">{{.Detail.Summary.KWh}}</strong></div>
              <div class="metric-card"><span class="metric-label">Ø aWATTar</span><strong class="metric-value">{{.Detail.Summary.AverageAwattar}}</strong></div>
              <div class="metric-card"><span class="metric-label">Ø effektiv</span><strong class="metric-value">{{.Detail.Summary.EffectivePrice}}</strong></div>
              <div class="metric-card"><span class="metric-label">Strom</span><strong class="metric-value">{{.Detail.Summary.EnergyCost}}</strong></div>
              <div class="metric-card"><span class="metric-label">Netzgeb.</span><strong class="metric-value">{{.Detail.Summary.GridCost}}</strong></div>
              <div class="metric-card"><span class="metric-label">Summe</span><strong class="metric-value">{{.Detail.Summary.TotalCost}}</strong></div>
            </div>
          {{end}}
        </section>
        <section class="panel accounting">
          <h2>Stundenwerte</h2>
          <div class="legend" aria-label="Legende für Stundenwerte">
            <div><strong>Stunde</strong><span>Beginn der Abrechnungsstunde; jede Zeile umfasst diese Stunde.</span></div>
            <div><strong>Verbrauch</strong><span>Geschätzte kWh aus der Differenz der Zählerstände innerhalb dieser Stunde.</span></div>
            <div><strong>Ø aWATTar</strong><span>Stündlicher aWATTar-Arbeitspreis ohne Netzgebühr.</span></div>
            <div><strong>Strom</strong><span>Verbrauch × aWATTar-Preis.</span></div>
            <div><strong>Netzgeb.</strong><span>Verbrauch × eingestellte Netzgebühr.</span></div>
            <div><strong>Summe</strong><span>Strom plus Netzgebühr; dieser Wert fließt in den Monatsbetrag.</span></div>
            <div><strong>Gewichtung</strong><span>Relative Balkenlänge im Vergleich zur teuersten Stunde des Monats.</span></div>
          </div>
          {{if .Detail.HasHours}}
            <div class="table-wrap">
              <table aria-label="Stundenwerte Parkplatznutzung">
                <thead>
                  <tr>
                    <th title="Beginn der Abrechnungsstunde; jede Zeile umfasst diese Stunde.">Stunde</th>
                    <th class="num" title="Geschätzte kWh aus der Differenz der Zählerstände innerhalb dieser Stunde.">Verbrauch</th>
                    <th class="num" title="Stündlicher aWATTar-Arbeitspreis ohne Netzgebühr.">Ø aWATTar</th>
                    <th class="num" title="Verbrauch × aWATTar-Preis.">Strom</th>
                    <th class="num" title="Verbrauch × eingestellte Netzgebühr.">Netzgeb.</th>
                    <th class="num" title="Strom plus Netzgebühr; dieser Wert fließt in den Monatsbetrag.">Summe</th>
                    <th title="Relative Balkenlänge im Vergleich zur teuersten Stunde des Monats.">Gewichtung</th>
                  </tr>
                </thead>
                <tbody>
                  {{range .Detail.Hours}}
                    <tr>
                      <td title="{{.AtTitle}}">{{.AtLabel}}</td>
                      <td class="num" title="{{.KWhTitle}}">{{.KWh}}</td>
                      <td class="num" title="{{.AverageAwattarTitle}}">{{.AverageAwattar}}</td>
                      <td class="num" title="{{.EnergyCostTitle}}">{{.EnergyCost}}</td>
                      <td class="num" title="{{.GridCostTitle}}">{{.GridCost}}</td>
                      <td class="num amount" title="{{.TotalCostTitle}}">{{.TotalCost}}</td>
                      <td class="bar-cell" title="{{.WeightTitle}}"><div class="bar"><span style="width: {{.ChartPercent}}%;"></span></div></td>
                    </tr>
                  {{end}}
                </tbody>
              </table>
            </div>
          {{else}}
            <p class="empty">Für diesen Monat sind noch keine Stundenwerte gespeichert.</p>
          {{end}}
        </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "settingsHub"}}
{{template "appOpen" .}}
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><span>Einstellungen</span></span>
      </div>
      <section class="page">
        <div>
          <h1>Einstellungen</h1>
          <p class="lede">Persönliche Einstellungen und Verwaltungsbereiche für {{.Tenant.Address}}.</p>
        </div>
        <div class="home-grid">
          <section class="panel">
            <div class="kicker">Konto</div>
            <div class="quick-list">
              <div class="quick-row disabled">
                <svg viewBox="0 0 24 24"><path d="M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8z"/><path d="M4.5 21a7.5 7.5 0 0 1 15 0"/></svg>
                <div><h3>Profil</h3><p>{{.Email}} · {{.Role}}</p></div>
                <span class="pill">Aktiv</span>
              </div>
              <div class="quick-row disabled">
                <svg viewBox="0 0 24 24"><path d="M18 8a6 6 0 1 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9"/><path d="M10 21h4"/></svg>
                <div><h3>Benachrichtigungen</h3><p>E-Mail-Präferenzen werden hier gebündelt, sobald Ereignisbenachrichtigungen aktiv sind.</p></div>
                <span class="pill">Vorbereitet</span>
              </div>
            </div>
          </section>
          <section class="panel">
            <div class="kicker">Verwaltung</div>
            {{if .IsAdmin}}
              <div class="quick-list">
                <a class="quick-row" href="/app/settings/users">
                  <svg viewBox="0 0 24 24"><path d="M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"/><path d="M3.5 20a5 5 0 0 1 10 0"/><path d="M16 11.5a2.5 2.5 0 1 0 0-5"/><path d="M17 15a4 4 0 0 1 3.5 4"/></svg>
                  <div><h3>Benutzer &amp; Rechte</h3><p>Einladungen, Rollen und Zugriff der Hausgemeinschaft verwalten.</p></div>
                  <span class="quick-arrow">›</span>
                </a>
                <a class="quick-row" href="/app/parking/settings">
                  <svg viewBox="0 0 24 24"><path d="M5 16h14"/><path d="m7 16 1.5-5h7L17 16"/><path d="M7 16v3M17 16v3"/><path d="M7 19h1M16 19h1"/></svg>
                  <div><h3>Parkplatz-Abrechnung</h3><p>Netzgebühr und Abrechnungswerte für die private Parkplatznutzung.</p></div>
                  <span class="quick-arrow">›</span>
                </a>
                <div class="quick-row disabled">
                  <svg viewBox="0 0 24 24"><path d="M4 21V8l8-5 8 5v13"/><path d="M9 21v-7h6v7"/><path d="M8 10h.01M16 10h.01"/></svg>
                  <div><h3>Gebäude</h3><p>Adresse, Kontakte, Einheiten und Hausdaten werden hier zusammengeführt.</p></div>
                  <span class="pill">Vorbereitet</span>
                </div>
              </div>
            {{else}}
              <p class="empty">Verwaltungsbereiche sind nur für berechtigte Personen sichtbar.</p>
            {{end}}
          </section>
        </div>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "parkingSettings"}}
{{template "appOpen" .}}
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Parkplatz-Abrechnung</span></span>
        <div class="page-actions"><a class="button" href="/app/settings">Zurück zu Einstellungen</a><a class="button" href="/app/parking">Zur Parkplatznutzung</a></div>
      </div>
      <section class="page">
        <div>
          <h1>Parkplatz-Abrechnung</h1>
          <p class="lede">Abrechnungswerte für die private Parkplatznutzung.</p>
        </div>
        <section class="panel settings-card">
          <div>
            <h2>Netzgebühr</h2>
            <p class="muted">Kurzer Aufschlag je kWh für Netzbetreibergebühren und lokale Basisanteile. Dieser Wert fließt in die Monatsabrechnung ein.</p>
          </div>
          {{if .SettingsMsg}}<p class="flash {{if .SettingsOK}}ok{{end}}">{{.SettingsMsg}}</p>{{end}}
          <form class="form-grid" method="post" action="/app/parking/settings">
            <label for="grid_fee_eur_per_kwh">Netzgebühr je kWh</label>
            <input id="grid_fee_eur_per_kwh" type="text" inputmode="decimal" name="grid_fee_eur_per_kwh" value="{{.Accounting.GridFeeValue}}" autocomplete="off">
            <button class="button primary" type="submit">Speichern</button>
            {{if .Accounting.LastSampleLabel}}<span class="mini">Letzter Zählerwert: {{.Accounting.LastSampleLabel}}</span>{{end}}
          </form>
        </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "userSettings"}}
{{template "appOpen" .}}
    <style>
      .users .panel { background: var(--panel); border: 1px solid var(--line); border-radius: 12px; padding: 22px; }
      .users .stack { display: grid; gap: 14px; }
      .users .panel-head { display: flex; align-items: baseline; justify-content: space-between; gap: 16px; border-bottom: 2px solid var(--ink); padding-bottom: 10px; margin-bottom: 14px; }
      .users .panel-head .kicker { border: 0; padding: 0; margin: 0; font-size: 12px; font-weight: 700; letter-spacing: .14em; text-transform: uppercase; color: var(--gold-ink); }
      .users .count { color: var(--soft); font-size: 12px; font-weight: 700; letter-spacing: .04em; font-variant-numeric: tabular-nums; }
      .users .muted { color: var(--muted); line-height: 1.55; }
      .users .roster-intro { margin: 2px 0 4px; }
      .users .disclosure { border: 1px solid var(--line); border-radius: 11px; background: var(--panel-soft); }
      .users .disclosure > summary { list-style: none; cursor: pointer; display: flex; align-items: center; gap: 11px; padding: 13px 16px; font-weight: 700; font-size: 14px; color: var(--ink); user-select: none; border-radius: 10px; }
      .users .disclosure > summary::-webkit-details-marker { display: none; }
      .users .disclosure > summary:hover { color: var(--gold-ink); }
      .users .disclosure > summary:focus-visible { outline: 2px solid var(--gold); outline-offset: -2px; }
      .users .disclosure[open] > summary { border-radius: 10px 10px 0 0; }
      .users .invite-plus { flex: 0 0 auto; width: 22px; height: 22px; border-radius: 6px; display: grid; place-items: center; background: var(--ink); color: #fff; }
      .users .summary-sub { margin-left: auto; font-weight: 600; font-size: 12.5px; color: var(--soft); }
      .users .disclosure-body { padding: 4px 16px 18px; }
      .users .invite-form { display: grid; grid-template-columns: repeat(12, 1fr); gap: 10px; }
      .users .invite-form .f-titel { grid-column: span 2; }
      .users .invite-form .f-vorname { grid-column: span 3; }
      .users .invite-form .f-nachname { grid-column: span 3; }
      .users .invite-form .f-email { grid-column: span 4; }
      .users .invite-form .f-role { grid-column: span 5; }
      .users .invite-form .f-submit { grid-column: span 7; }
      .users input, .users select { width: 100%; border: 1px solid #e2dac9; border-radius: 9px; padding: 12px; font: inherit; background: #fffefb; color: var(--ink); }
      .users .invite-form button { border: 1px solid var(--ink); background: var(--ink); border-radius: 10px; color: #fff; min-height: 44px; padding: 10px 13px; font: inherit; font-weight: 700; cursor: pointer; }
      .users .invite-form button:hover { background: #2c3329; }
      .users .invite-flash { margin: 0 0 12px; padding: 10px 13px; border-radius: 9px; font-size: 13.5px; font-weight: 600; border: 1px solid transparent; }
      .users .invite-flash.ok { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.25); }
      .users .invite-flash.warn { background: rgba(200,153,63,.14); color: #93701d; border-color: rgba(200,153,63,.3); }
      .users .table-wrap { overflow: visible; }
      .users table { width: 100%; border-collapse: collapse; font-size: 15px; }
      .users thead th { color: var(--gold-ink); font-size: 11px; font-weight: 800; text-transform: uppercase; letter-spacing: .06em; text-align: left; padding: 4px 14px 12px; border-bottom: 2px solid var(--line); white-space: nowrap; }
      .users tbody td { padding: 15px 14px; border-bottom: 1px solid var(--line); vertical-align: middle; }
      .users tbody tr:last-child td { border-bottom: 0; }
      .users tbody tr { transition: background .12s ease; }
      .users tbody tr:hover { background: #faf6ec; }
      .users th:first-child, .users td:first-child { padding-left: 4px; }
      .users th:last-child, .users td:last-child { padding-right: 4px; }
      .users .person { display: flex; align-items: center; gap: 13px; min-width: 220px; }
      .users .avatar { flex: 0 0 auto; width: 40px; height: 40px; border-radius: 50%; display: grid; place-items: center; font-size: 14px; font-weight: 700; color: var(--gold-ink); background: rgba(200,153,63,.15); border: 1px solid rgba(200,153,63,.32); }
      .users .person-name { font-weight: 600; line-height: 1.25; }
      .users .person-mail { color: var(--muted); font-size: 13px; margin-top: 2px; word-break: break-word; }
      .users .chips { display: flex; flex-wrap: wrap; gap: 6px; }
      .users .chip { display: inline-flex; align-items: center; gap: 6px; border: 1px solid var(--line); background: var(--panel-soft); color: #6f6a5c; border-radius: 8px; padding: 4px 10px; font-size: 12.5px; font-weight: 600; white-space: nowrap; }
      .users .chip.plain { color: var(--soft); }
      .users .role-caps { display: flex; flex-wrap: wrap; gap: 5px; margin-top: 6px; }
      .users .role-cap { display: inline-flex; align-items: center; min-height: 22px; border: 1px solid var(--line); border-radius: 999px; padding: 2px 8px; color: var(--muted); background: var(--panel-soft); font-size: 11.5px; font-weight: 700; white-space: nowrap; }
      .users .pill { display: inline-flex; align-items: center; gap: 7px; border-radius: 999px; min-height: 28px; padding: 4px 12px; font-size: 13px; font-weight: 700; white-space: nowrap; border: 1px solid transparent; }
      .users .pill .dot { width: 7px; height: 7px; border-radius: 50%; background: currentColor; opacity: .9; }
      .users .pill.role-admin { background: rgba(200,153,63,.16); color: #8a6a1f; border-color: rgba(200,153,63,.28); }
      .users .pill.role-manager { background: rgba(32,37,31,.08); color: var(--ink); border-color: rgba(32,37,31,.15); }
      .users .pill.role-owner { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.22); }
      .users .pill.role-renter { background: rgba(76,103,138,.11); color: #365475; border-color: rgba(76,103,138,.22); }
      .users .pill.role-resident { background: rgba(47,107,74,.11); color: var(--leaf); border-color: rgba(47,107,74,.2); }
      .users .pill.role-beirat { background: rgba(32,37,31,.06); color: #4b4f45; border-color: rgba(32,37,31,.12); }
      .users .pill.status-active { background: rgba(47,107,74,.12); color: var(--leaf); }
      .users .pill.status-pending { background: rgba(200,153,63,.14); color: #93701d; }
      .users .last-seen { display: block; color: var(--soft); font-size: 11.5px; margin-top: 5px; white-space: nowrap; }
      .users th.col-role, .users td.col-role, .users th.col-status, .users td.col-status { white-space: nowrap; }
      .users .th-label { display: inline-flex; align-items: center; gap: 6px; }
      .users .info { position: relative; display: inline-flex; }
      .users .info-btn { width: 17px; height: 17px; border-radius: 50%; border: 1px solid var(--gold-ink); background: transparent; color: var(--gold-ink); display: grid; place-items: center; padding: 0; cursor: help; }
      .users .info-btn:hover, .users .info-btn:focus-visible { background: var(--gold-ink); color: #fff; outline: none; }
      .users .info-btn:focus-visible { box-shadow: 0 0 0 2px rgba(200,153,63,.4); }
      .users .popup { position: absolute; top: calc(100% + 11px); left: -12px; width: min(480px, 88vw); background: var(--panel); border: 1px solid var(--line); border-radius: 14px; box-shadow: 0 20px 46px rgba(32,37,31,.17), 0 3px 9px rgba(32,37,31,.05); padding: 16px 19px 18px; z-index: 8; opacity: 0; visibility: hidden; transform: translateY(-6px); transition: opacity .16s ease, transform .16s ease; text-transform: none; letter-spacing: normal; }
      .users .popup::before { content: ""; position: absolute; top: -6px; left: 19px; width: 12px; height: 12px; background: var(--panel); border-left: 1px solid var(--line); border-top: 1px solid var(--line); border-radius: 3px 0 0 0; transform: rotate(45deg); }
      .users .popup::after { content: ""; position: absolute; top: -15px; left: 0; right: 0; height: 15px; }
      .users .info:hover .popup, .users .info:focus-within .popup { opacity: 1; visibility: visible; transform: translateY(0); }
      .users .popup-title { display: block; font-family: Spectral, serif; font-weight: 600; font-size: 15px; color: var(--ink); padding-bottom: 11px; border-bottom: 1px solid var(--line); }
      .users .popup-grid { display: grid; grid-template-columns: minmax(0,1fr) minmax(0,1fr); column-gap: 26px; }
      .users .popup .permission { display: block; padding: 12px 0; }
      .users .popup-grid .permission:nth-child(1), .users .popup-grid .permission:nth-child(2) { padding-top: 14px; }
      .users .popup-grid .permission:nth-child(n+3) { border-top: 1px solid var(--line); }
      .users .popup .permission strong { display: block; font-family: Spectral, serif; font-weight: 600; font-size: 13.5px; color: var(--ink); margin-bottom: 3px; }
      .users .popup .permission .muted { display: block; font-size: 12.5px; font-weight: 400; color: var(--muted); line-height: 1.5; overflow-wrap: break-word; }
      .users .rdot { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 9px; vertical-align: middle; }
      .users .rdot.admin { background: var(--gold); }
      .users .rdot.manager { background: var(--ink); }
      .users .rdot.owner { background: var(--leaf); }
      .users .rdot.renter { background: #365475; }
      .users .rdot.resident { background: var(--leaf); }
      .users .rdot.beirat { background: #8a8d80; }
      .users .rdot.right { background: var(--gold-light); box-shadow: inset 0 0 0 1px var(--gold); }
      .users .col-actions { width: 44px; }
      .users td.col-actions { text-align: right; }
      .users .row-edit { border: 1px solid transparent; background: transparent; border-radius: 8px; width: 32px; height: 32px; display: inline-grid; place-items: center; color: var(--soft); cursor: pointer; padding: 0; }
      .users .row-edit:hover { border-color: var(--line); background: var(--panel-soft); color: var(--gold-ink); }
      .users .row-edit svg { stroke: currentColor; fill: none; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
      .users .edit-dialog { position: relative; width: min(440px, 92vw); border: 1px solid var(--line); border-radius: 14px; padding: 22px; background: var(--panel); color: var(--ink); box-shadow: 0 30px 80px rgba(32,37,31,.32); }
      .users .edit-dialog::backdrop { background: rgba(32,37,31,.42); }
      .users .edit-dialog h2 { margin: 0 0 4px; font-family: Spectral, serif; font-weight: 600; font-size: 19px; }
      .users .edit-dialog .dlg-sub { color: var(--muted); font-size: 13px; margin: 0 0 16px; word-break: break-word; }
      .users .dlg-x { position: absolute; top: 12px; right: 12px; }
      .users .dlg-x button { border: 0; background: transparent; font-size: 22px; line-height: 1; color: var(--soft); cursor: pointer; padding: 2px 6px; }
      .users .dlg-x button:hover { color: var(--ink); }
      .users .dlg-form { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
      .users .dlg-form input, .users .dlg-form select, .users .dlg-form button { grid-column: 1 / -1; }
      .users .dlg-form .f-vorname, .users .dlg-form .f-nachname { grid-column: span 1; }
      .users .dlg-form button { border: 1px solid var(--ink); background: var(--ink); color: #fff; border-radius: 10px; min-height: 44px; padding: 10px 13px; font: inherit; font-weight: 700; cursor: pointer; }
      .users .dlg-form button:hover { background: #2c3329; }
      .users .dlg-delete { margin-top: 16px; padding-top: 14px; border-top: 1px solid var(--line); display: flex; justify-content: space-between; align-items: center; gap: 12px; }
      .users .dlg-delete span { color: var(--muted); font-size: 12.5px; }
      .users .dlg-delete .danger { border: 1px solid rgba(150,40,40,.32); background: rgba(150,40,40,.07); color: #9a2b2b; border-radius: 10px; min-height: 40px; padding: 8px 15px; font: inherit; font-weight: 700; cursor: pointer; }
      .users .dlg-delete .danger:hover { background: rgba(150,40,40,.14); }
      @media (max-width: 760px) {
        .users .table-wrap { overflow: visible; }
        .users table, .users thead, .users tbody, .users tr, .users td { display: block; width: 100%; }
        .users thead { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); }
        .users tbody tr { border: 1px solid var(--line); border-radius: 11px; padding: 14px; margin-bottom: 12px; background: var(--panel); }
        .users tbody tr:hover { background: var(--panel); }
        .users tbody td { border: 0; padding: 0; }
        .users tbody td.col-person { margin-bottom: 12px; }
        .users tbody td[data-label]:not(.col-person) { display: grid; grid-template-columns: 96px 1fr; align-items: start; gap: 10px; padding: 7px 0; border-top: 1px dashed var(--line); }
        .users tbody td[data-label]:not(.col-person)::before { content: attr(data-label); color: var(--gold-ink); font-size: 10.5px; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; padding-top: 5px; }
        .users .invite-form > * { grid-column: 1 / -1 !important; }
      }
      @media (max-width: 560px) { .users .popup-grid { grid-template-columns: 1fr; } .users .popup-grid .permission:nth-child(n+2) { border-top: 1px solid var(--line); padding-top: 12px; } }
    </style>
    <script src="/assets/users.js" defer></script>
    <main class="app-main">
      <div class="content-top"><span class="crumb"><svg viewBox="0 0 24 24"><path d="M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"/><path d="M3.5 20a5 5 0 0 1 10 0"/><path d="M16 11.5a2.5 2.5 0 1 0 0-5"/><path d="M17 15a4 4 0 0 1 3.5 4"/></svg>Benutzer &amp; Rechte</span></div>
      <section class="page users">
        <div>
          <h1>Benutzer &amp; Rechte</h1>
          <p class="lede">Lokale Verwaltung der eingeladenen E-Mail-Adressen und ihrer Rollen.</p>
        </div>
    <section class="panel stack">
      <div class="panel-head">
        <span class="kicker">Zugänge</span>
        <span class="count">{{len .Users}} {{if eq (len .Users) 1}}Person{{else}}Personen{{end}}</span>
      </div>

      <details class="disclosure invite-bar"{{if .InviteMsg}} open{{end}}>
        <summary>
          <span class="invite-plus"><svg viewBox="0 0 24 24" width="13" height="13" aria-hidden="true"><path d="M12 5.5v13M5.5 12h13" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"/></svg></span>
          Person einladen
          <span class="summary-sub">Speichert &amp; lädt per E-Mail ein</span>
        </summary>
        <div class="disclosure-body">
          {{if .InviteMsg}}<p class="invite-flash{{if .InviteOK}} ok{{else}} warn{{end}}">{{.InviteMsg}}</p>{{end}}
          <p class="muted" style="margin-bottom:12px">Die eingeladene Person wird gespeichert und erhält eine E-Mail mit dem Anmelde-Link. Sie kann sich danach mit dieser Adresse anmelden.</p>
          <form class="invite-form" method="post" action="/app/settings/users">
            <input class="f-titel" type="text" name="title" placeholder="Titel">
            <input class="f-vorname" type="text" name="first_name" placeholder="Vorname">
            <input class="f-nachname" type="text" name="last_name" placeholder="Nachname">
            <input class="f-email" type="email" name="email" placeholder="name@example.com" required>
            <select class="f-role" name="role">
              <option value="Mieter">Mieter</option>
              <option value="Eigentümer">Eigentümer</option>
              <option value="Beirat">Beirat</option>
              <option value="Verwalter">Verwalter</option>
              <option value="Admin">Admin</option>
              <option value="Bewohner">Bewohner</option>
            </select>
            <button class="f-submit" type="submit">Einladung senden</button>
          </form>
        </div>
      </details>

      <p class="muted roster-intro">Diese Liste kommt aus der Umgebungskonfiguration und den hier gespeicherten Einladungen.</p>

      <div class="table-wrap">
        <table aria-label="Benutzerliste">
          <thead>
            <tr>
              <th class="col-person">Person</th>
              <th class="col-role">
                <span class="th-label">Rolle
                  <span class="info">
                    <button type="button" class="info-btn" aria-label="Rollen und Rechte erklärt"><svg viewBox="0 0 16 16" width="10" height="10" aria-hidden="true"><circle cx="8" cy="3.5" r="1.15" fill="currentColor"/><rect x="6.9" y="6.3" width="2.2" height="6.3" rx="1.1" fill="currentColor"/></svg></button>
                    <span class="popup" role="tooltip">
                      <span class="popup-title">Rollen &amp; Rechte</span>
                      <span class="popup-grid">
                        <span class="permission"><strong><span class="rdot admin"></span>Admin</strong><span class="muted">Zugänge verwalten, Rollen setzen und Portalbereiche vorbereiten.</span></span>
                        <span class="permission"><strong><span class="rdot manager"></span>Verwalter</strong><span class="muted">Aushang verwalten und übergreifende Leserechte vorbereiten.</span></span>
                        <span class="permission"><strong><span class="rdot owner"></span>Eigentümer</strong><span class="muted">Bewohnerbereich plus Eigentümer-Dokumente und Abstimmungen.</span></span>
                        <span class="permission"><strong><span class="rdot renter"></span>Mieter</strong><span class="muted">Bewohnerbereich ohne Eigentümer-Abstimmungen.</span></span>
                        <span class="permission"><strong><span class="rdot beirat"></span>Beirat</strong><span class="muted">Bewohnerbereich plus lesende Übersicht.</span></span>
                        <span class="permission"><strong><span class="rdot right"></span>Parkplatznutzung</strong><span class="muted">Separates Sonderrecht für den privaten Parkplatzbereich.</span></span>
                      </span>
                    </span>
                  </span>
                </span>
              </th>
              <th>Rechte</th>
              <th>Anmeldung</th>
              <th class="col-status">Status</th>
              <th class="col-actions" aria-label="Aktionen"></th>
            </tr>
          </thead>
          <tbody>
            {{range .Users}}
            <tr>
              <td class="col-person" data-label="Person">
                <div class="person">
                  <span class="avatar">{{.Initials}}</span>
                  <div>
                    <div class="person-name">{{.DisplayName}}</div>
                    <div class="person-mail">{{.Email}}</div>
                  </div>
                </div>
              </td>
              <td class="col-role" data-label="Rolle"><span class="pill {{.RoleClass}}"><span class="dot"></span>{{.Role}}</span><div class="role-caps">{{range .RoleCapabilities}}<span class="role-cap">{{.}}</span>{{end}}</div></td>
              <td data-label="Rechte"><div class="chips">{{range .PermissionList}}<span class="chip{{if eq . "Standard"}} plain{{end}}">{{.}}</span>{{end}}</div></td>
              <td data-label="Anmeldung"><div class="chips">{{range .AuthList}}<span class="chip">{{.}}</span>{{end}}</div></td>
              <td class="col-status" data-label="Status"><span class="pill {{if eq .Status "Aktiv"}}status-active{{else}}status-pending{{end}}"><span class="dot"></span>{{.Status}}</span>{{if .LastSeen}}<span class="last-seen">{{.LastSeen}}</span>{{end}}</td>
              <td class="col-actions" data-label="">
                {{if .Editable}}
                <button type="button" class="row-edit" data-edit="{{.Email}}" aria-label="Bearbeiten"><svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true"><path d="M4 20h4L18.5 9.5a2 2 0 0 0-2.83-2.83L5 17.2z"/><path d="M13.5 6.5 17 10"/></svg></button>
                <dialog id="edit-{{.Email}}" class="edit-dialog">
                  <div class="dlg-x"><form method="dialog"><button aria-label="Schließen">&times;</button></form></div>
                  <h2>Zugang bearbeiten</h2>
                  <p class="dlg-sub">{{.Email}}</p>
                  <form method="post" action="/app/settings/users/edit" class="dlg-form">
                    <input type="hidden" name="orig_email" value="{{.Email}}">
                    <input class="f-titel" type="text" name="title" value="{{.Title}}" placeholder="Titel">
                    <input class="f-vorname" type="text" name="first_name" value="{{.FirstName}}" placeholder="Vorname">
                    <input class="f-nachname" type="text" name="last_name" value="{{.LastName}}" placeholder="Nachname">
                    <input class="f-email" type="email" name="email" value="{{.Email}}" required>
                    <select class="f-role" name="role">
                      <option value="Mieter"{{if eq .Role "Mieter"}} selected{{end}}>Mieter</option>
                      <option value="Eigentümer"{{if eq .Role "Eigentümer"}} selected{{end}}>Eigentümer</option>
                      <option value="Beirat"{{if eq .Role "Beirat"}} selected{{end}}>Beirat</option>
                      <option value="Verwalter"{{if eq .Role "Verwalter"}} selected{{end}}>Verwalter</option>
                      <option value="Admin"{{if eq .Role "Admin"}} selected{{end}}>Admin</option>
                      <option value="Bewohner"{{if eq .Role "Bewohner"}} selected{{end}}>Bewohner</option>
                    </select>
                    <button type="submit">Speichern</button>
                  </form>
                  <div class="dlg-delete">
                    <span>Dauerhaft entfernen</span>
                    <form method="post" action="/app/settings/users/delete">
                      <input type="hidden" name="email" value="{{.Email}}">
                      <button type="submit" class="danger">Löschen</button>
                    </form>
                  </div>
                </dialog>
                {{end}}
              </td>
            </tr>
            {{end}}
          </tbody>
        </table>
      </div>
    </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}
`

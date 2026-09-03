// Package auth owns login state: magic-link tokens, signed session cookies and
// the OIDC flow.
//
// Sessions are AES-GCM signed and verified here; nothing above this package
// should be minting or parsing them by hand.
package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/textutil"
)

type TokenStore struct {
	mu     sync.Mutex
	secret []byte
	items  map[string]LoginToken
}

type LoginToken struct {
	email        string
	tenantSlug   string
	redirectPath string
	expiresAt    time.Time
	used         bool
}

type SessionStore struct {
	mu      sync.Mutex
	secret  []byte
	revoked map[string]time.Time
}

type Session struct {
	Email            string `json:"email"`
	TenantSlug       string `json:"tenant_slug"`
	AuthMethod       string `json:"auth_method"`
	Role             string `json:"role,omitempty"`
	ExpiresAt        int64  `json:"expires_at"`
	PreviewRole      string `json:"preview_role,omitempty"`
	PreviewStartedAt int64  `json:"preview_started_at,omitempty"`
	PreviewExpiresAt int64  `json:"preview_expires_at,omitempty"`
}

type OidcLogin struct {
	mu           sync.Mutex
	providerName string
	issuer       string
	clientID     string
	clientSecret string
	redirectURL  string
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
}

type OidcFlowStore struct {
	mu    sync.Mutex
	items map[string]OidcFlow
}

type OidcFlow struct {
	tenantSlug   string
	nonce        string
	codeVerifier string
	expiresAt    time.Time
	used         bool
}

type OidcUserClaims struct {
	Email         string `json:"email"`
	EmailVerified *bool  `json:"email_verified"`
}

func (s *TokenStore) Put(token string, email string, tenantSlug string, ttl time.Duration) {
	s.PutWithRedirect(token, email, tenantSlug, ttl, "")
}

func (s *TokenStore) PutWithRedirect(token string, email string, tenantSlug string, ttl time.Duration, redirectPath string) {
	key := s.digest(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = LoginToken{email: email, tenantSlug: tenantSlug, redirectPath: SafeInternalRedirectPath(redirectPath), expiresAt: time.Now().Add(ttl)}
}

func (s *TokenStore) Consume(token string) (string, string, string, bool) {
	if token == "" {
		return "", "", "", false
	}
	key := s.digest(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[key]
	if !ok || item.used || time.Now().After(item.expiresAt) {
		delete(s.items, key)
		return "", "", "", false
	}
	item.used = true
	s.items[key] = item
	return item.email, item.tenantSlug, item.redirectPath, true
}

// Invalidate removes a one-time login token without revealing whether it was
// present. Delivery failures and bounded queue shutdown use it so a link that
// was never safely delivered cannot become usable later.
func (s *TokenStore) Invalidate(token string) {
	if s == nil || token == "" {
		return
	}
	key := s.digest(token)
	s.mu.Lock()
	delete(s.items, key)
	s.mu.Unlock()
}

// Peek returns the login identity without marking the one-time token used. It
// exists so request/account admission checks can run before Consume; callers
// must still call Consume immediately afterwards and treat its result as the
// authoritative single-use decision.
func (s *TokenStore) Peek(token string) (string, string, string, bool) {
	if token == "" {
		return "", "", "", false
	}
	key := s.digest(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[key]
	if !ok || item.used || time.Now().After(item.expiresAt) {
		delete(s.items, key)
		return "", "", "", false
	}
	return item.email, item.tenantSlug, item.redirectPath, true
}

func SafeInternalRedirectPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.Contains(raw, "://") {
		return ""
	}
	if raw != "/app" && !strings.HasPrefix(raw, "/app/") {
		return ""
	}
	return raw
}

func (s *TokenStore) digest(token string) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(token))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func NewSessionStore(secret []byte) *SessionStore {
	sum := sha256.Sum256(append([]byte("weg-session-aead-v1."), secret...))
	key := make([]byte, len(sum))
	copy(key, sum[:])
	return &SessionStore{
		secret:  key,
		revoked: map[string]time.Time{},
	}
}

func (s *SessionStore) Put(email string, tenantSlug string, authMethod string, ttl time.Duration) (string, time.Time, error) {
	return s.PutSession(email, tenantSlug, authMethod, "", time.Now().Add(ttl))
}

// PutSession issues a signed session for an already authenticated identity and
// an explicitly selected tenant/role context. Callers must validate that the
// identity owns the requested context before calling it. An existing expiry can
// be retained during a context switch so switching never extends a login.
func (s *SessionStore) PutSession(email string, tenantSlug string, authMethod string, role string, expiresAt time.Time) (string, time.Time, error) {
	item := Session{
		Email:      textutil.Email(email),
		TenantSlug: textutil.Slug(tenantSlug),
		AuthMethod: store.NormalizeAuthMethod(authMethod),
		Role:       store.NormalizeRole(role),
		ExpiresAt:  expiresAt.Unix(),
	}
	return s.putSession(item)
}

// PutRolePreview narrows an existing administrative session to one of the two
// resident-facing roles. The parent login expiry remains authoritative.
func (s *SessionStore) PutRolePreview(parent Session, previewRole string, startedAt time.Time, previewExpiresAt time.Time) (string, time.Time, error) {
	parent.PreviewRole = store.NormalizeRole(previewRole)
	parent.PreviewStartedAt = startedAt.UTC().Truncate(time.Second).Unix()
	parent.PreviewExpiresAt = previewExpiresAt.UTC().Truncate(time.Second).Unix()
	return s.putSession(parent)
}

// EndRolePreview issues a fresh signed copy of the real administrative
// context without extending the parent login.
func (s *SessionStore) EndRolePreview(preview Session) (string, time.Time, error) {
	if preview.PreviewRole == "" {
		return "", time.Time{}, fmt.Errorf("invalid role preview")
	}
	preview.PreviewRole = ""
	preview.PreviewStartedAt = 0
	preview.PreviewExpiresAt = 0
	return s.putSession(preview)
}

func (s *SessionStore) putSession(item Session) (string, time.Time, error) {
	item.Email = textutil.Email(item.Email)
	item.TenantSlug = textutil.Slug(item.TenantSlug)
	item.AuthMethod = store.NormalizeAuthMethod(item.AuthMethod)
	item.Role = store.NormalizeRole(item.Role)
	item.PreviewRole = store.NormalizeRole(item.PreviewRole)
	if item.Email == "" || item.TenantSlug == "" || item.AuthMethod == "" || !validSessionRole(item.Role) || item.ExpiresAt <= time.Now().Unix() || !validRolePreview(item) {
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
	return token, time.Unix(item.ExpiresAt, 0), nil
}

func (s *SessionStore) Get(token string) (string, string, string, bool) {
	item, ok := s.GetSession(token)
	if !ok {
		return "", "", "", false
	}
	return item.Email, item.TenantSlug, item.AuthMethod, true
}

// GetSession returns the complete signed context. The legacy Get method stays
// available for callers that only need identity, tenant and login method.
func (s *SessionStore) GetSession(token string) (Session, bool) {
	item, ok := s.verify(token)
	if !ok {
		return Session{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
	if _, revoked := s.revoked[s.revocationKey(token)]; revoked {
		return Session{}, false
	}
	return item, true
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
	if item, ok := s.verify(token); ok {
		s.revoked[s.revocationKey(token)] = time.Unix(item.ExpiresAt, 0)
	}
}

func (s *SessionStore) verify(token string) (Session, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "v1" {
		return Session{}, false
	}
	nonce, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Session{}, false
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Session{}, false
	}
	block, err := aes.NewCipher(s.secret)
	if err != nil {
		return Session{}, false
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(nonce) != gcm.NonceSize() {
		return Session{}, false
	}
	payload, err := gcm.Open(nil, nonce, ciphertext, []byte("weg-session-v1"))
	if err != nil {
		return Session{}, false
	}
	var item Session
	if err := json.Unmarshal(payload, &item); err != nil {
		return Session{}, false
	}
	item.Email = textutil.Email(item.Email)
	item.TenantSlug = textutil.Slug(item.TenantSlug)
	item.AuthMethod = store.NormalizeAuthMethod(item.AuthMethod)
	item.Role = store.NormalizeRole(item.Role)
	item.PreviewRole = store.NormalizeRole(item.PreviewRole)
	if item.Email == "" || item.TenantSlug == "" || item.AuthMethod == "" || !validSessionRole(item.Role) || item.ExpiresAt <= time.Now().Unix() || !validRolePreview(item) {
		return Session{}, false
	}
	return item, true
}

func validRolePreview(item Session) bool {
	if item.PreviewRole == "" {
		return item.PreviewStartedAt == 0 && item.PreviewExpiresAt == 0
	}
	if item.Role != store.RoleAdmin || (item.PreviewRole != store.RoleOwner && item.PreviewRole != store.RoleResident) {
		return false
	}
	return item.PreviewStartedAt > 0 && item.PreviewExpiresAt > item.PreviewStartedAt && item.PreviewExpiresAt <= item.ExpiresAt
}

func validSessionRole(role string) bool {
	switch role {
	case "", store.RoleAdmin, store.RoleManager, store.RoleOwner, store.RoleRenter, store.RoleBeirat, store.RoleResident, store.RoleServiceProvider:
		return true
	default:
		return false
	}
}

func (s *SessionStore) revocationKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (s *SessionStore) cleanupLocked() {
	now := time.Now()
	for key, expiresAt := range s.revoked {
		if now.After(expiresAt) {
			delete(s.revoked, key)
		}
	}
}

func NewOIDCLogin(ctx context.Context, issuer string, clientID string, clientSecret string, redirectURL string, providerName string) (*OidcLogin, error) {
	issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	redirectURL = strings.TrimSpace(redirectURL)
	providerName = strings.TrimSpace(providerName)
	if providerName == "" {
		providerName = "Zitadel"
	}
	if issuer == "" && clientID == "" && clientSecret == "" && redirectURL == "" {
		return &OidcLogin{providerName: providerName}, nil
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
	login := &OidcLogin{
		providerName: providerName,
		issuer:       issuer,
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
	}
	if err := login.EnsureProvider(ctx); err != nil {
		slog.Warn("OIDC discovery unavailable at startup", "retry", "on_login", "error", err)
	}
	return login, nil
}

func (o *OidcLogin) Configured() bool {
	return o != nil && o.issuer != "" && o.clientID != ""
}

func (o *OidcLogin) ProviderName() string {
	if o == nil || o.providerName == "" {
		return "SSO"
	}
	return o.providerName
}

func (o *OidcLogin) RedirectURL(baseURL string) string {
	if o.redirectURL != "" {
		return o.redirectURL
	}
	return strings.TrimRight(baseURL, "/") + "/auth/oidc/callback"
}

func (o *OidcLogin) OAuthConfig(redirectURL string) oauth2.Config {
	return oauth2.Config{
		ClientID:     o.clientID,
		ClientSecret: o.clientSecret,
		Endpoint:     o.provider.Endpoint(),
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "email", "profile"},
	}
}

func (o *OidcLogin) EnsureProvider(ctx context.Context) error {
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

func (s *OidcFlowStore) Put(state string, flow OidcFlow, ttl time.Duration) {
	flow.expiresAt = time.Now().Add(ttl)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
	s.items[state] = flow
}

func (s *OidcFlowStore) Consume(state string) (OidcFlow, bool) {
	if state == "" {
		return OidcFlow{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
	flow, ok := s.items[state]
	if !ok || flow.used || time.Now().After(flow.expiresAt) {
		delete(s.items, state)
		return OidcFlow{}, false
	}
	flow.used = true
	delete(s.items, state)
	return flow, true
}

func (s *OidcFlowStore) cleanupLocked() {
	now := time.Now()
	for state, flow := range s.items {
		if now.After(flow.expiresAt) {
			delete(s.items, state)
		}
	}
}

func PkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (c *OidcUserClaims) Merge(other OidcUserClaims) {
	if c.Email == "" {
		c.Email = other.Email
	}
	if c.EmailVerified == nil {
		c.EmailVerified = other.EmailVerified
	}
}

func RandomToken(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// Constructors. main used to build these with struct literals over unexported
// fields — legal only while they shared a package. Requiring a constructor also
// makes it impossible to create a TokenStore without its signing secret.

func NewTokenStore(secret []byte) *TokenStore {
	return &TokenStore{secret: secret, items: map[string]LoginToken{}}
}

func NewOIDCFlowStore() *OidcFlowStore {
	return &OidcFlowStore{items: map[string]OidcFlow{}}
}

// NewOIDCFlow carries the per-login state through the OIDC redirect.
func NewOIDCFlow(tenantSlug, nonce, codeVerifier string) OidcFlow {
	return OidcFlow{tenantSlug: tenantSlug, nonce: nonce, codeVerifier: codeVerifier}
}

// Accessors for the flow state consumed by the OIDC callback handler.
func (f OidcFlow) TenantSlug() string   { return f.tenantSlug }
func (f OidcFlow) Nonce() string        { return f.nonce }
func (f OidcFlow) CodeVerifier() string { return f.codeVerifier }

func (o *OidcLogin) Verifier() *oidc.IDTokenVerifier { return o.verifier }
func (o *OidcLogin) Provider() *oidc.Provider        { return o.provider }

// Sign returns an HMAC-SHA256 of value under the session secret. main used to
// reach into sessions.secret and do this itself (to sign calendar-feed tokens);
// the key never needs to leave this package, so it doesn't.
//
// Behaviour preserved verbatim: empty secret is an error, otherwise
// hmac.New(sha256.New, secret) over the raw value.
func (s *SessionStore) Sign(value string) ([]byte, error) {
	if s == nil || len(s.secret) == 0 {
		return nil, fmt.Errorf("signing secret unavailable")
	}
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil), nil
}

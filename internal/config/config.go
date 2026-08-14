// Package config parses the process environment into typed configuration:
// tenants, user profiles, durations, secrets.
//
// This is the ONLY package that reads os.Getenv. Everything below it takes
// explicit values, which is what makes the rest testable.
package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/homeassistant"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/textutil"
)

type TenantConfig struct {
	Slug              string                   `json:"slug"`
	Name              string                   `json:"name"`
	Address           string                   `json:"address"`
	PortalType        string                   `json:"portal_type,omitempty"`
	BrandIcon         string                   `json:"brand_icon,omitempty"`
	BrandAbbreviation string                   `json:"brand_abbreviation,omitempty"`
	MapLatitude       float64                  `json:"map_latitude,omitempty"`
	MapLongitude      float64                  `json:"map_longitude,omitempty"`
	MapZoom           int                      `json:"map_zoom,omitempty"`
	ContactName       string                   `json:"contact_name,omitempty"`
	ContactAddress    string                   `json:"contact_address,omitempty"`
	ContactEmail      string                   `json:"contact_email,omitempty"`
	ContactPhone      string                   `json:"contact_phone,omitempty"`
	EmergencyName     string                   `json:"emergency_name,omitempty"`
	EmergencyPhone    string                   `json:"emergency_phone,omitempty"`
	CaretakerName     string                   `json:"caretaker_name,omitempty"`
	CaretakerEmail    string                   `json:"caretaker_email,omitempty"`
	CaretakerPhone    string                   `json:"caretaker_phone,omitempty"`
	HeroImageURL      string                   `json:"hero_image_url,omitempty"`
	HA                homeassistant.Config     `json:"-"`
	HAConnectors      *HomeAssistantConnectors `json:"-"`
}

const (
	PortalTypeCommunity = "community"
	PortalTypeApartment = "apartment"
	PortalTypeHouse     = "house"
)

// HomeAssistantConnector is the non-secret, tenant-scoped connector
// declaration. Tokens are referenced by file or environment-variable name;
// inline token material is intentionally not part of the schema.
type HomeAssistantConnector struct {
	TenantSlug  string `json:"tenant_slug"`
	HomeKey     string `json:"home_key,omitempty"`
	BaseURL     string `json:"base_url"`
	TokenFile   string `json:"token_file,omitempty"`
	TokenEnv    string `json:"token_env,omitempty"`
	MeterEnergy string `json:"meter_energy_entity,omitempty"`
	Power       string `json:"power_entity,omitempty"`
	Price       string `json:"price_entity,omitempty"`
	PlugSwitch  string `json:"plug_switch_entity,omitempty"`
	BatterySOC  string `json:"battery_soc_entity,omitempty"`
	GridFeedIn  string `json:"grid_feed_in_entity,omitempty"`
}

type HomeAssistantConnectors map[string]homeassistant.Config

func (t TenantConfig) PublicURL(path string) string {
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return "/" + t.Slug + path
}

func ParseDuration(raw string) (time.Duration, error) {
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

func ParseHistoryStart(raw string, now time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.Local), nil
	}
	if t, err := time.ParseInLocation("2006-01-02", raw, time.Local); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, raw)
}

func Env(key string, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func LoadLocalEnv(path string) error {
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
		_ = os.Setenv(key, TrimEnvQuotes(strings.TrimSpace(value)))
	}
	return nil
}

func TrimEnvQuotes(value string) string {
	if len(value) < 2 {
		return value
	}
	if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}

func ParseBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func IsLocalHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func ParseAllowed(raw string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, item := range strings.Split(raw, ",") {
		email := textutil.Email(item)
		if email != "" {
			out[email] = struct{}{}
		}
	}
	return out
}

func ParseTenants(raw string, defaultTenant string, defaultHA homeassistant.Config) (map[string]TenantConfig, error) {
	out := map[string]TenantConfig{}
	raw = strings.TrimSpace(raw)
	if raw != "" {
		var tenants []TenantConfig
		if err := json.Unmarshal([]byte(raw), &tenants); err != nil {
			return nil, fmt.Errorf("invalid WEG_TENANTS_JSON")
		}
		for _, tenant := range tenants {
			tenant.Slug = textutil.Slug(tenant.Slug)
			if tenant.Slug == "" {
				return nil, fmt.Errorf("tenant is missing slug")
			}
			if tenant.Address == "" {
				tenant.Address = tenant.Slug
			}
			if tenant.Name == "" {
				tenant.Name = tenant.Address
			}
			tenant.PortalType = strings.ToLower(strings.TrimSpace(tenant.PortalType))
			if tenant.PortalType == "" {
				tenant.PortalType = PortalTypeCommunity
			}
			switch tenant.PortalType {
			case PortalTypeCommunity, PortalTypeApartment, PortalTypeHouse:
			default:
				return nil, fmt.Errorf("tenant %s has invalid portal_type", tenant.Slug)
			}
			tenant.ContactName = strings.TrimSpace(tenant.ContactName)
			tenant.ContactAddress = strings.TrimSpace(tenant.ContactAddress)
			tenant.ContactEmail = textutil.Email(tenant.ContactEmail)
			tenant.ContactPhone = strings.TrimSpace(tenant.ContactPhone)
			tenant.EmergencyName = strings.TrimSpace(tenant.EmergencyName)
			tenant.EmergencyPhone = strings.TrimSpace(tenant.EmergencyPhone)
			tenant.CaretakerName = strings.TrimSpace(tenant.CaretakerName)
			tenant.CaretakerEmail = textutil.Email(tenant.CaretakerEmail)
			tenant.CaretakerPhone = strings.TrimSpace(tenant.CaretakerPhone)
			if tenant.HeroImageURL == "" {
				tenant.HeroImageURL = store.DefaultTenantHeroImageURL
			}
			if tenant.HA.BaseURL() == "" && tenant.Slug == textutil.Slug(defaultTenant) {
				tenant.HA = defaultHA
			}
			out[tenant.Slug] = tenant
		}
	}

	defaultTenant = textutil.Slug(defaultTenant)
	if _, ok := out[defaultTenant]; !ok {
		out[defaultTenant] = TenantConfig{
			Slug:         defaultTenant,
			Name:         "Musterweg 1",
			Address:      "Musterweg 1",
			PortalType:   PortalTypeCommunity,
			MapLatitude:  48.2082,
			MapLongitude: 16.3738,
			MapZoom:      17,
			HeroImageURL: store.DefaultTenantHeroImageURL,
			HA:           defaultHA,
		}
	}
	return out, nil
}

// ApplyHomeAssistantConnectors adds tenant-scoped Home Assistant connections
// after ParseTenants. The JSON is safe to keep in declarative host config:
// credentials are only loaded from an agenix-mounted file or a named env var.
func ApplyHomeAssistantConnectors(raw string, tenants map[string]TenantConfig) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var connectors []HomeAssistantConnector
	if err := json.Unmarshal([]byte(raw), &connectors); err != nil {
		return fmt.Errorf("invalid HA_CONNECTORS_JSON")
	}
	seen := map[string]struct{}{}
	for _, connector := range connectors {
		slug := textutil.Slug(connector.TenantSlug)
		homeKey := NormalizeHomeKey(connector.HomeKey)
		tenant, ok := tenants[slug]
		if slug == "" || !ok {
			return fmt.Errorf("home assistant connector references unknown tenant")
		}
		connectorKey := slug + "\x00" + homeKey
		if _, duplicate := seen[connectorKey]; duplicate {
			return fmt.Errorf("duplicate home assistant connector for tenant %s and home %s", slug, homeKey)
		}
		seen[connectorKey] = struct{}{}
		baseURL := strings.TrimRight(strings.TrimSpace(connector.BaseURL), "/")
		parsed, err := url.Parse(baseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return fmt.Errorf("invalid home assistant base URL for tenant %s", slug)
		}
		token, err := connectorToken(connector)
		if err != nil {
			return fmt.Errorf("home assistant credential unavailable for tenant %s", slug)
		}
		ha := homeassistant.NewConfig(
			baseURL,
			token,
			strings.TrimSpace(connector.MeterEnergy),
			strings.TrimSpace(connector.Power),
			strings.TrimSpace(connector.Price),
		).WithChargingEntities(
			strings.TrimSpace(connector.PlugSwitch),
			strings.TrimSpace(connector.BatterySOC),
			strings.TrimSpace(connector.GridFeedIn),
		)
		if tenant.HAConnectors == nil {
			connectors := HomeAssistantConnectors{}
			tenant.HAConnectors = &connectors
		}
		(*tenant.HAConnectors)[homeKey] = ha
		if homeKey == DefaultHomeKey {
			tenant.HA = ha
		}
		tenants[slug] = tenant
	}
	return nil
}

const DefaultHomeKey = "default"

func NormalizeHomeKey(raw string) string {
	if key := textutil.Slug(raw); key != "" {
		return key
	}
	return DefaultHomeKey
}

func (t TenantConfig) HomeAssistant(homeKey string) homeassistant.Config {
	key := NormalizeHomeKey(homeKey)
	if t.HAConnectors != nil {
		if connector, ok := (*t.HAConnectors)[key]; ok {
			return connector
		}
	}
	if key == DefaultHomeKey {
		return t.HA
	}
	return homeassistant.Config{}
}

func (t TenantConfig) HomeAssistantHomeKeys() []string {
	keys := []string{}
	if t.HAConnectors != nil {
		for key, connector := range *t.HAConnectors {
			if connector.Configured() {
				keys = append(keys, NormalizeHomeKey(key))
			}
		}
	}
	if t.HA.Configured() {
		found := false
		for _, key := range keys {
			found = found || key == DefaultHomeKey
		}
		if !found {
			keys = append(keys, DefaultHomeKey)
		}
	}
	sort.Strings(keys)
	return keys
}

func connectorToken(connector HomeAssistantConnector) (string, error) {
	tokenFile := strings.TrimSpace(connector.TokenFile)
	tokenEnv := strings.TrimSpace(connector.TokenEnv)
	if (tokenFile == "") == (tokenEnv == "") {
		return "", errors.New("exactly one token source is required")
	}
	if tokenFile != "" {
		raw, err := os.ReadFile(tokenFile)
		if err != nil {
			return "", errors.New("token file unavailable")
		}
		token := strings.TrimSpace(string(raw))
		if token == "" {
			return "", errors.New("token file empty")
		}
		return token, nil
	}
	if strings.ContainsAny(tokenEnv, " \t\r\n=") {
		return "", errors.New("invalid token env name")
	}
	token := strings.TrimSpace(os.Getenv(tokenEnv))
	if token == "" {
		return "", errors.New("token env unavailable")
	}
	return token, nil
}

func ParseUserProfiles(raw string, allowed map[string]struct{}, admins map[string]struct{}, defaultTenant string) (map[string]store.UserProfile, error) {
	out := map[string]store.UserProfile{}
	defaultTenant = textutil.Slug(defaultTenant)
	raw = strings.TrimSpace(raw)
	if raw != "" {
		var profiles []store.UserProfile
		if err := json.Unmarshal([]byte(raw), &profiles); err != nil {
			return nil, fmt.Errorf("invalid WEG_USERS_JSON")
		}
		for _, profile := range profiles {
			email := textutil.Email(profile.Email)
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
			profile.Phone = strings.TrimSpace(profile.Phone)
			profile.Role = store.NormalizeRole(profile.Role)
			if profile.Role == "" {
				if _, ok := admins[email]; ok {
					profile.Role = store.RoleAdmin
				} else {
					profile.Role = store.RoleResident
				}
			}
			if profile.Status == "" {
				profile.Status = "Eingeladen"
			}
			profile.TenantMemberships = store.NormalizeTenantMemberships(profile.TenantMemberships)
			profile.Tenants = store.NormalizeTenants(append(profile.Tenants, store.TenantMembershipSlugs(profile.TenantMemberships)...), defaultTenant)
			profile.Permissions = store.NormalizePermissions(profile.Permissions)
			authMethods, err := store.NormalizeAuthMethods(profile.AuthMethods)
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
		out[email] = store.UserProfile{Email: email, Role: store.RoleAdmin, Status: "Aktiv", Tenants: []string{defaultTenant}, AuthMethods: store.DefaultAuthMethods()}
	}
	for email := range allowed {
		if _, ok := out[email]; ok {
			continue
		}
		out[email] = store.UserProfile{Email: email, Role: store.RoleResident, Status: "Eingeladen", Tenants: []string{defaultTenant}, AuthMethods: store.DefaultAuthMethods()}
	}
	return out, nil
}

func NormalizeHost(raw string) string {
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

func SessionSecret(requireConfigured bool) ([]byte, error) {
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

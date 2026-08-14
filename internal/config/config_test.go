package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/inspr-at/hausv-org/internal/homeassistant"
)

func TestApplyHomeAssistantConnectorsUsesReferencedSecret(t *testing.T) {
	secretPath := filepath.Join(t.TempDir(), "ha-token")
	if err := os.WriteFile(secretPath, []byte("not-a-real-secret\n"), 0o600); err != nil {
		t.Fatalf("write token fixture: %v", err)
	}
	tenants := map[string]TenantConfig{
		"home": {Slug: "home", Name: "Zuhause"},
	}
	raw := `[{
		"tenant_slug":"home",
		"base_url":"https://ha.example.test/",
		"token_file":` + quoteJSON(secretPath) + `,
		"power_entity":"sensor.grid_power"
	}]`
	if err := ApplyHomeAssistantConnectors(raw, tenants); err != nil {
		t.Fatalf("ApplyHomeAssistantConnectors: %v", err)
	}
	if !tenants["home"].HA.Configured() {
		t.Fatal("connector not configured")
	}
	if got := tenants["home"].HA.BaseURL(); got != "https://ha.example.test" {
		t.Fatalf("base URL = %q", got)
	}
	if got := tenants["home"].HA.PowerEntity(); got != "sensor.grid_power" {
		t.Fatalf("power entity = %q", got)
	}
}

func TestApplyHomeAssistantConnectorsSupportsMultipleHomesPerTenant(t *testing.T) {
	t.Setenv("HA_TEST_TOKEN", "fixture")
	tenants := map[string]TenantConfig{"home": {Slug: "home"}}
	err := ApplyHomeAssistantConnectors(`[
		{"tenant_slug":"home","home_key":"einheit-12","base_url":"https://unit12.example.test","token_env":"HA_TEST_TOKEN"},
		{"tenant_slug":"home","home_key":"top-12","base_url":"https://top12.example.test","token_env":"HA_TEST_TOKEN"}
	]`, tenants)
	if err != nil {
		t.Fatalf("ApplyHomeAssistantConnectors: %v", err)
	}
	for homeKey, wantURL := range map[string]string{
		"einheit-12": "https://unit12.example.test",
		"top-12":     "https://top12.example.test",
	} {
		connector := tenants["home"].HomeAssistant(homeKey)
		if !connector.Configured() || connector.BaseURL() != wantURL {
			t.Fatalf("connector %q = configured:%v url:%q, want %q", homeKey, connector.Configured(), connector.BaseURL(), wantURL)
		}
	}
	if tenants["home"].HA.Configured() {
		t.Fatal("named home connectors must not silently replace the legacy default connector")
	}
}

func TestApplyHomeAssistantConnectorsRejectsMissingAndDuplicateTenant(t *testing.T) {
	tenants := map[string]TenantConfig{"home": {Slug: "home"}}
	t.Setenv("HA_TEST_TOKEN", "fixture")
	if err := ApplyHomeAssistantConnectors(`[{"tenant_slug":"other","base_url":"https://ha.example.test","token_env":"HA_TEST_TOKEN"}]`, tenants); err == nil {
		t.Fatal("expected unknown tenant error")
	}
	if err := ApplyHomeAssistantConnectors(`[
		{"tenant_slug":"home","base_url":"https://one.example.test","token_env":"HA_TEST_TOKEN"},
		{"tenant_slug":"home","base_url":"https://two.example.test","token_env":"HA_TEST_TOKEN"}
	]`, tenants); err == nil {
		t.Fatal("expected duplicate tenant error")
	}
}

func TestApplyHomeAssistantConnectorsRequiresExternalTokenReference(t *testing.T) {
	tenants := map[string]TenantConfig{"home": {Slug: "home"}}
	if err := ApplyHomeAssistantConnectors(`[{"tenant_slug":"home","base_url":"https://ha.example.test"}]`, tenants); err == nil {
		t.Fatal("expected missing token reference error")
	}
}

func TestParseTenantsKeepsDefaultConnectorFallback(t *testing.T) {
	defaultHA := homeassistant.NewConfig("https://ha.example.test", "fixture", "", "", "")
	tenants, err := ParseTenants("", "home", defaultHA)
	if err != nil {
		t.Fatalf("ParseTenants: %v", err)
	}
	if !tenants["home"].HA.Configured() {
		t.Fatal("default connector not preserved")
	}
	if tenants["home"].MapLatitude == 0 || tenants["home"].MapLongitude == 0 || tenants["home"].MapZoom != 17 {
		t.Fatalf("default tenant map coordinates = %#v", tenants["home"])
	}
	if tenants["home"].PortalType != PortalTypeCommunity {
		t.Fatalf("default portal type = %q", tenants["home"].PortalType)
	}
	if tenants["home"].Name != "Musterweg 1" {
		t.Fatalf("default tenant name = %q", tenants["home"].Name)
	}
}

func TestParseTenantsValidatesIndependentPortalType(t *testing.T) {
	tenants, err := ParseTenants(
		`[{"slug":"private-home","portal_type":"house"},{"slug":"weg","portal_type":"community"}]`,
		"weg",
		homeassistant.Config{},
	)
	if err != nil {
		t.Fatalf("ParseTenants: %v", err)
	}
	if tenants["private-home"].PortalType != PortalTypeHouse || tenants["weg"].PortalType != PortalTypeCommunity {
		t.Fatalf("portal types = %#v", tenants)
	}
	if _, err := ParseTenants(
		`[{"slug":"invalid","portal_type":"mixed-up"}]`,
		"invalid",
		homeassistant.Config{},
	); err == nil {
		t.Fatal("expected invalid portal_type error")
	}
}

func TestParseTenantsUsesAddressAsNeutralNameFallback(t *testing.T) {
	tenants, err := ParseTenants(
		`[{"slug":"private-home","address":"Musterweg 4","portal_type":"house"},{"slug":"slug-only","portal_type":"apartment"}]`,
		"private-home",
		homeassistant.Config{},
	)
	if err != nil {
		t.Fatalf("ParseTenants: %v", err)
	}
	if got := tenants["private-home"].Name; got != "Musterweg 4" {
		t.Fatalf("address-backed tenant name = %q", got)
	}
	if got := tenants["slug-only"].Name; got != "slug-only" {
		t.Fatalf("slug-backed tenant name = %q", got)
	}
}

func quoteJSON(value string) string {
	out := `"`
	for _, r := range value {
		switch r {
		case '\\', '"':
			out += `\` + string(r)
		default:
			out += string(r)
		}
	}
	return out + `"`
}

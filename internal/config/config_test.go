package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/markus-barta/hausv-org/internal/homeassistant"
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
	tenants, err := ParseTenants("", "hausv.org", "home", defaultHA)
	if err != nil {
		t.Fatalf("ParseTenants: %v", err)
	}
	if !tenants["home"].HA.Configured() {
		t.Fatal("default connector not preserved")
	}
	if tenants["home"].MapLatitude == 0 || tenants["home"].MapLongitude == 0 || tenants["home"].MapZoom != 17 {
		t.Fatalf("default tenant map coordinates = %#v", tenants["home"])
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

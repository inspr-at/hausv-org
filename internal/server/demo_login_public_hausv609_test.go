package server

import (
	"path/filepath"
	"strings"
	"testing"
)

// The demo bundle (HAUSV-609) runs behind a public BASE_URL with neither SMTP
// nor OIDC: the access-code login is its only login path, so the boot guard
// must accept it — and must still refuse a public instance with no login at all.
func TestPublicBaseURLAcceptsDemoLoginAsOnlyLoginPath(t *testing.T) {
	setPublicDemoEnv := func(t *testing.T) {
		t.Helper()
		t.Setenv("DB_PATH", filepath.Join(t.TempDir(), "boot.db"))
		t.Setenv("PARKING_DATA_PATH", filepath.Join(t.TempDir(), "parking.json"))
		t.Setenv("BASE_URL", "https://hausv.example")
		t.Setenv("TRUSTED_PROXY_CIDRS", "127.0.0.1/32")
		t.Setenv("SESSION_KEY", strings.Repeat("ab", 32))
		t.Setenv("SMTP_HOST", "")
		t.Setenv("OIDC_ISSUER", "")
	}

	t.Run("demo login satisfies the guard", func(t *testing.T) {
		setPublicDemoEnv(t)
		t.Setenv("DEMO_LOGIN_ENABLED", "true")
		t.Setenv("DEMO_LOGIN_ACCESS_CODE", "musterstadt-2026")
		a, err := newApp()
		if err != nil {
			t.Fatalf("public demo instance must boot with the access-code login alone: %v", err)
		}
		if !a.demoLogin || a.localDevLogin {
			t.Fatalf("demoLogin=%v localDevLogin=%v; want demo login on and dev login off", a.demoLogin, a.localDevLogin)
		}
	})

	t.Run("no login path still refuses to boot", func(t *testing.T) {
		setPublicDemoEnv(t)
		t.Setenv("DEMO_LOGIN_ENABLED", "false")
		t.Setenv("DEMO_LOGIN_ACCESS_CODE", "")
		if _, err := newApp(); err == nil || !strings.Contains(err.Error(), "demo login") {
			t.Fatalf("public instance without SMTP, OIDC or demo login must refuse to boot naming all three, got: %v", err)
		}
	})

	t.Run("enabled without a code is not a login path", func(t *testing.T) {
		setPublicDemoEnv(t)
		t.Setenv("DEMO_LOGIN_ENABLED", "true")
		t.Setenv("DEMO_LOGIN_ACCESS_CODE", "")
		if _, err := newApp(); err == nil {
			t.Fatal("DEMO_LOGIN_ENABLED without an access code must not count as a login path")
		}
	})
}

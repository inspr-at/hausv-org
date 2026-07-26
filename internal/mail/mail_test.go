package mail

import "testing"

func TestSMTPMailerAllowsInternalRelayWithoutAuth(t *testing.T) {
	m := NewSMTP("smtp", "25", "", "", "WEG Portal <noreply@hausv.org>")

	if !m.Configured() {
		t.Fatal("mailer should be configured with host, port, and from")
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("relay mailer should validate without auth: %v", err)
	}
	if auth := m.auth(); auth != nil {
		t.Fatal("relay mailer without user/pass should not create smtp auth")
	}
}

func TestSMTPMailerRequiresPairedCredentials(t *testing.T) {
	m := NewSMTP("smtp", "25", "user", "", "WEG Portal <noreply@hausv.org>")

	if err := m.Validate(); err == nil {
		t.Fatal("mailer should reject partial smtp credentials")
	}
}

func TestPrivacyURLUsesSamePortalOrigin(t *testing.T) {
	got := privacyURL("https://jhw22.hausv.org/auth/verify?token=secret")
	if got != "https://jhw22.hausv.org/datenschutz" {
		t.Fatalf("privacy URL = %q", got)
	}
}

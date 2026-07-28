package mail

import (
	"strings"
	"testing"
)

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

func TestMagicLinkMessageUsesHouseLanguageWithoutProviderJargon(t *testing.T) {
	msg := magicLinkMessage(
		"hausv.org <noreply@hausv.org>",
		"max@example.com",
		"https://jhw22.hausv.org/auth/verify?token=secret",
		"Janischhofweg 22, 8043 Graz",
	)
	for _, want := range []string{
		"Subject: Ihr Anmeldelink für Janischhofweg 22, 8043 Graz",
		"im Hausportal für Janischhofweg 22, 8043 Graz",
		"Der Link ist 15 Minuten gültig",
		"https://jhw22.hausv.org/datenschutz",
		"Hausportal Janischhofweg 22, 8043 Graz",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("magic-link message missing %q:\n%s", want, msg)
		}
	}
	for _, forbidden := range []string{"WEG Portal", "Zitadel", "SSO"} {
		if strings.Contains(msg, forbidden) {
			t.Fatalf("magic-link message exposes %q:\n%s", forbidden, msg)
		}
	}
}

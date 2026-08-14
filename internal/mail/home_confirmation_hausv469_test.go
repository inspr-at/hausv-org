package mail

import (
	"strings"
	"testing"
)

func TestHomeConfirmationMessageDescribesReservationNotLogin(t *testing.T) {
	message := homeConfirmationMessage(
		"HAUSV <hello@hausv.org>",
		"owner@example.com",
		"https://hausv.org/start/verify?token=secret-value",
		"Zuhause am Park",
	)
	for _, want := range []string{
		"Subject: HAUSV Home Reservierung bestätigen",
		"Reservierung für Zuhause am Park",
		"noch kein Portal aktiviert",
		"Datenschutzinformationen: https://hausv.org/datenschutz",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("message missing %q:\n%s", want, message)
		}
	}
	if strings.Contains(message, "/start/datenschutz") || strings.Contains(message, "im Hausportal anmelden") {
		t.Fatalf("reservation message uses login wording:\n%s", message)
	}
}

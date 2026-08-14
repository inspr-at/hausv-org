package server

import (
	"testing"
	"time"
)

// HAUSV-147: a fresh calendar token parses; a very old one is rejected.
func TestCalendarFeedTokenExpiry(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	tok, err := a.calendarFeedToken("owner@example.com", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := a.parseCalendarFeedToken(tok); !ok {
		t.Fatal("fresh token should parse")
	}

	// Hand-mint a token issued beyond the max age and confirm it's refused.
	old := a.signedCalendarFeedTokenAt("owner@example.com", "demo", time.Now().Add(-maxCalendarFeedAge-24*time.Hour))
	if _, ok := a.parseCalendarFeedToken(old); ok {
		t.Fatal("token older than maxCalendarFeedAge must be rejected")
	}
}

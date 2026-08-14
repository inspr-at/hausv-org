package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestExistingConfigUserCanLogIntoConfirmedAdditionalHome(t *testing.T) {
	const (
		email = "existing@example.com"
		slug  = "elternhaus"
	)
	a := newTestPortalApp(t, userProfile{
		Email:       email,
		Role:        roleResident,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	now := time.Date(2026, time.August, 14, 15, 0, 0, 0, time.UTC)
	if _, err := a.homeReservations.Reserve(store.HomeReservation{
		Slug:                   slug,
		HouseholdName:          "Elternhaus",
		OwnerEmail:             email,
		AuthorizationConfirmed: true,
	}, now); err != nil {
		t.Fatal(err)
	}
	if _, confirmed, err := a.homeReservations.Confirm(slug, email, now.Add(time.Minute)); err != nil || !confirmed {
		t.Fatalf("confirm confirmed=%v err=%v", confirmed, err)
	}
	if _, created, err := a.homePortals.Activate(slug, email, now.Add(2*time.Minute)); err != nil || !created {
		t.Fatalf("activate created=%v err=%v", created, err)
	}

	// The env profile remains authoritative for its original house. Only the
	// atomically activated home receives the additional owner authorization.
	profile, ok := a.directoryProfile(email)
	if !ok || !profile.HasTenant("demo") || profile.HasTenant(slug) || normalizeRole(profile.Role) != roleResident {
		t.Fatalf("env profile was replaced or escalated: %+v ok=%v", profile, ok)
	}
	if !a.isAllowed(email, slug) || !a.isAuthMethodAllowed(email, slug, authMethodEmail) {
		t.Fatal("confirmed home owner cannot use email login")
	}
	if got := a.roleFor(email, slug); got != roleOwner {
		t.Fatalf("additional home role = %q, want %q", got, roleOwner)
	}
	if scoped := a.profileForTenant(email, slug); !scoped.HasTenant(slug) || normalizeRole(scoped.Role) != roleOwner {
		t.Fatalf("additional home profile = %+v", scoped)
	}

	mailer := &recordingMailer{}
	a.mailer = mailer
	values := url.Values{"email": {email}}
	req := httptest.NewRequest(http.MethodPost, "http://hausv.org/"+slug+"/auth/request", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://hausv.org/"+slug)
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/"+slug+"/?sent=1" {
		t.Fatalf("login request status=%d location=%q", rr.Code, rr.Header().Get("Location"))
	}
	links := waitForMagicLinks(t, mailer, 1)
	if links[0].To != email || !strings.Contains(links[0].Link, "/"+slug+"/auth/verify?") {
		t.Fatalf("magic link = %+v", links[0])
	}
}

func TestPlainPersistedMembershipStillCannotExtendConfigUser(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "existing@example.com",
		Role:        roleResident,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	if _, err := a.inviteStore.Add(userProfile{
		Email:       "existing@example.com",
		Role:        roleAdmin,
		Tenants:     []string{"unverified-home"},
		AuthMethods: defaultAuthMethods(),
	}); err != nil {
		t.Fatal(err)
	}
	if a.isAllowed("existing@example.com", "unverified-home") || a.isAuthMethodAllowed("existing@example.com", "unverified-home", authMethodEmail) {
		t.Fatal("plain persisted membership extended an env-backed identity")
	}
}

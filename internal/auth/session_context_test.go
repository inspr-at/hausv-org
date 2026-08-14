package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestSessionContextRoundTripKeepsRoleAndExpiry(t *testing.T) {
	sessions := NewSessionStore([]byte(strings.Repeat("s", 32)))
	expiresAt := time.Now().Add(45 * time.Minute).Truncate(time.Second)
	token, gotExpiry, err := sessions.PutSession("Owner@Example.com", "Haus-B", store.AuthMethodOIDC, store.RoleRenter, expiresAt)
	if err != nil {
		t.Fatalf("PutSession: %v", err)
	}
	if !gotExpiry.Equal(expiresAt) {
		t.Fatalf("expiry = %s, want %s", gotExpiry, expiresAt)
	}
	context, ok := sessions.GetSession(token)
	if !ok {
		t.Fatal("signed context should verify")
	}
	if context.Email != "owner@example.com" || context.TenantSlug != "haus-b" || context.AuthMethod != store.AuthMethodOIDC || context.Role != store.RoleRenter || context.ExpiresAt != expiresAt.Unix() {
		t.Fatalf("context = %+v", context)
	}
}

func TestSessionContextRejectsUnknownRole(t *testing.T) {
	sessions := NewSessionStore([]byte(strings.Repeat("s", 32)))
	if _, _, err := sessions.PutSession("owner@example.com", "demo", store.AuthMethodEmail, "Superuser", time.Now().Add(time.Hour)); err == nil {
		t.Fatal("unknown session role must be rejected")
	}
}

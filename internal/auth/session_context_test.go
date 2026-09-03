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

func TestRolePreviewRoundTripAndEndKeepRealIdentityAndParentExpiry(t *testing.T) {
	sessions := NewSessionStore([]byte(strings.Repeat("s", 32)))
	parentExpiry := time.Now().Add(time.Hour).Truncate(time.Second)
	parentToken, _, err := sessions.PutSession("Admin@Example.com", "Haus-B", store.AuthMethodOIDC, store.RoleAdmin, parentExpiry)
	if err != nil {
		t.Fatalf("PutSession: %v", err)
	}
	parent, ok := sessions.GetSession(parentToken)
	if !ok {
		t.Fatal("parent session should verify")
	}
	startedAt := time.Now().UTC().Truncate(time.Second)
	previewExpiry := startedAt.Add(15 * time.Minute)
	previewToken, gotExpiry, err := sessions.PutRolePreview(parent, store.RoleOwner, startedAt, previewExpiry)
	if err != nil {
		t.Fatalf("PutRolePreview: %v", err)
	}
	if !gotExpiry.Equal(parentExpiry) {
		t.Fatalf("cookie expiry = %s, want parent expiry %s", gotExpiry, parentExpiry)
	}
	preview, ok := sessions.GetSession(previewToken)
	if !ok {
		t.Fatal("preview session should verify")
	}
	if preview.Email != "admin@example.com" || preview.Role != store.RoleAdmin || preview.PreviewRole != store.RoleOwner {
		t.Fatalf("preview identity = %+v", preview)
	}
	if preview.PreviewStartedAt != startedAt.Unix() || preview.PreviewExpiresAt != previewExpiry.Unix() || preview.ExpiresAt != parentExpiry.Unix() {
		t.Fatalf("preview timestamps = %+v", preview)
	}

	restoredToken, restoredExpiry, err := sessions.EndRolePreview(preview)
	if err != nil {
		t.Fatalf("EndRolePreview: %v", err)
	}
	restored, ok := sessions.GetSession(restoredToken)
	if !ok || restored.PreviewRole != "" || restored.Role != store.RoleAdmin || !restoredExpiry.Equal(parentExpiry) {
		t.Fatalf("restored session = %+v ok=%v expiry=%s", restored, ok, restoredExpiry)
	}
}

func TestRolePreviewValidationRejectsWideningAndExpiryExtension(t *testing.T) {
	sessions := NewSessionStore([]byte(strings.Repeat("s", 32)))
	parentExpiry := time.Now().Add(time.Hour).Truncate(time.Second)
	startedAt := time.Now().UTC().Truncate(time.Second)

	for _, test := range []struct {
		name        string
		parentRole  string
		previewRole string
		expiresAt   time.Time
	}{
		{name: "non admin actor", parentRole: store.RoleManager, previewRole: store.RoleOwner, expiresAt: startedAt.Add(15 * time.Minute)},
		{name: "manager preview", parentRole: store.RoleAdmin, previewRole: store.RoleManager, expiresAt: startedAt.Add(15 * time.Minute)},
		{name: "past parent", parentRole: store.RoleAdmin, previewRole: store.RoleResident, expiresAt: parentExpiry.Add(time.Second)},
	} {
		t.Run(test.name, func(t *testing.T) {
			parent := Session{Email: "admin@example.com", TenantSlug: "demo", AuthMethod: store.AuthMethodEmail, Role: test.parentRole, ExpiresAt: parentExpiry.Unix()}
			if _, _, err := sessions.PutRolePreview(parent, test.previewRole, startedAt, test.expiresAt); err == nil {
				t.Fatal("invalid preview must be rejected")
			}
		})
	}
}

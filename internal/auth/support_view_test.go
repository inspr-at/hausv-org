package auth

import (
	"github.com/inspr-at/hausv-org/internal/store"
	"testing"
	"time"
)

func TestSupportSessionRejectsMixedAndUnboundedContexts(t *testing.T) {
	now := time.Now().Unix()
	base := Session{Email: "admin@example.com", TenantSlug: "demo", AuthMethod: "email", Role: store.RoleAdmin, ExpiresAt: now + 3600, SupportTargetEmail: "resident@example.com", SupportTargetRole: store.RoleResident, SupportStartedAt: now, SupportExpiresAt: now + 900}
	if !validSupportSession(base) {
		t.Fatal("valid bounded support context rejected")
	}
	for name, mutate := range map[string]func(*Session){
		"preview":   func(s *Session) { s.PreviewRole = store.RoleOwner },
		"manager":   func(s *Session) { s.Role = store.RoleManager },
		"self":      func(s *Session) { s.SupportTargetEmail = s.Email },
		"unbounded": func(s *Session) { s.SupportExpiresAt++ },
		"parent":    func(s *Session) { s.ExpiresAt = now + 800 },
		"future":    func(s *Session) { s.SupportStartedAt = now + 10 },
		"partial":   func(s *Session) { s.SupportTargetRole = "" },
	} {
		t.Run(name, func(t *testing.T) {
			s := base
			mutate(&s)
			if validSupportSession(s) {
				t.Fatal("unsafe context accepted")
			}
		})
	}
}

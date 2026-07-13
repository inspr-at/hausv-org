package store

// UserProfile is PERSISTED — inviteStore stores []UserProfile — so it lives
// here rather than in config. Putting it in config would force config and store
// to import each other.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/markus-barta/hausv-org/internal/textutil"
)

type UserProfile struct {
	Email             string                      `json:"email"`
	Title             string                      `json:"title"`
	FirstName         string                      `json:"first_name"`
	LastName          string                      `json:"last_name"`
	Phone             string                      `json:"phone"`
	DirectoryOptIn    bool                        `json:"directory_opt_in,omitempty"`
	Role              string                      `json:"role"`
	Status            string                      `json:"status"`
	Tenants           []string                    `json:"tenants"`
	Permissions       []string                    `json:"permissions"`
	TenantMemberships map[string]TenantMembership `json:"tenant_memberships,omitempty"`
	AuthMethods       []string                    `json:"auth_methods"`
}

type TenantMembership struct {
	Role        string   `json:"role,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

func (p UserProfile) DisplayName() string {
	parts := []string{}
	for _, part := range []string{p.Title, p.FirstName, p.LastName} {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	return p.Email
}

// Initials returns up to two uppercase letters for the avatar badge, derived
// from first+last name, falling back to the first glyph of the display name.
func (p UserProfile) Initials() string {
	first := InitialLetter(p.FirstName)
	last := InitialLetter(p.LastName)
	if first == "" && last == "" {
		return strings.ToUpper(InitialLetter(p.DisplayName()))
	}
	return strings.ToUpper(first + last)
}

func InitialLetter(s string) string {
	for _, r := range strings.TrimSpace(s) {
		return string(r)
	}
	return ""
}

func (p UserProfile) HasPermission(permission string) bool {
	permission = strings.ToLower(strings.TrimSpace(permission))
	for _, item := range p.Permissions {
		if strings.ToLower(strings.TrimSpace(item)) == permission {
			return true
		}
	}
	return false
}

func (p UserProfile) ForTenant(tenantSlug string) UserProfile {
	tenantSlug = textutil.Slug(tenantSlug)
	out := p
	out.Email = textutil.Email(out.Email)
	out.Role = NormalizeRole(out.Role)
	out.Tenants = NormalizeTenants(out.Tenants, "")
	out.Permissions = NormalizePermissions(out.Permissions)
	out.AuthMethods = append([]string(nil), out.AuthMethods...)
	if membership, ok := p.membershipForTenant(tenantSlug); ok {
		out.Tenants = NormalizeTenants(append(out.Tenants, tenantSlug), "")
		if role := NormalizeRole(membership.Role); role != "" {
			out.Role = role
		}
		if membership.Permissions != nil {
			out.Permissions = NormalizePermissions(membership.Permissions)
		}
	}
	return out
}

func (p UserProfile) membershipForTenant(tenantSlug string) (TenantMembership, bool) {
	tenantSlug = textutil.Slug(tenantSlug)
	if tenantSlug == "" {
		return TenantMembership{}, false
	}
	for rawSlug, membership := range p.TenantMemberships {
		if textutil.Slug(rawSlug) == tenantSlug {
			return membership, true
		}
	}
	return TenantMembership{}, false
}

func (p UserProfile) AllowsAuthMethod(method string) bool {
	method = NormalizeAuthMethod(method)
	if method == "" {
		return false
	}
	methods := p.AuthMethods
	if len(methods) == 0 {
		methods = DefaultAuthMethods()
	}
	for _, item := range methods {
		if NormalizeAuthMethod(item) == method {
			return true
		}
	}
	return false
}

func (p UserProfile) HasTenant(tenantSlug string) bool {
	tenantSlug = textutil.Slug(tenantSlug)
	for _, item := range p.Tenants {
		if textutil.Slug(item) == tenantSlug {
			return true
		}
	}
	_, ok := p.membershipForTenant(tenantSlug)
	if ok {
		return true
	}
	return false
}

type InviteStore struct {
	path string
	mu   sync.Mutex
	data InviteStoreData
}

type InviteStoreData struct {
	Invites []UserProfile `json:"invites"`
}

func NewInviteStore(path string) (*InviteStore, error) {
	store := &InviteStore{path: path, data: InviteStoreData{Invites: []UserProfile{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read invite data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid invite data")
	}
	return store, nil
}

func (s *InviteStore) Get(email string) (UserProfile, bool) {
	email = textutil.Email(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, profile := range s.data.Invites {
		if textutil.Email(profile.Email) == email {
			return profile, true
		}
	}
	return UserProfile{}, false
}

func (s *InviteStore) List() []UserProfile {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]UserProfile(nil), s.data.Invites...)
}

// Add persists a new invite. Returns false (no error) when the email is already
// invited. Callers must ensure the email is not already in the env directory.
func (s *InviteStore) Add(profile UserProfile) (bool, error) {
	profile.Email = textutil.Email(profile.Email)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.data.Invites {
		if textutil.Email(existing.Email) == profile.Email {
			return false, nil
		}
	}
	s.data.Invites = append(s.data.Invites, profile)
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *InviteStore) saveLocked() error {
	return SaveJSONAtomic(s.path, s.data, "invite")
}

// Update replaces the invite keyed by oldEmail with updated. Returns false (no
// error) when oldEmail is not a persisted invite. When the email changes it
// must not collide with another invite (caller also checks the env directory).
func (s *InviteStore) Update(oldEmail string, updated UserProfile) (bool, error) {
	oldEmail = textutil.Email(oldEmail)
	updated.Email = textutil.Email(updated.Email)
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i, existing := range s.data.Invites {
		if textutil.Email(existing.Email) == oldEmail {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false, nil
	}
	if updated.Email != oldEmail {
		for i, existing := range s.data.Invites {
			if i != idx && textutil.Email(existing.Email) == updated.Email {
				return false, fmt.Errorf("email already invited")
			}
		}
	}
	s.data.Invites[idx] = updated
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

// Delete removes the invite for email. Returns whether one was removed.
func (s *InviteStore) Delete(email string) (bool, error) {
	email = textutil.Email(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.data.Invites[:0]
	removed := false
	for _, existing := range s.data.Invites {
		if textutil.Email(existing.Email) == email {
			removed = true
			continue
		}
		kept = append(kept, existing)
	}
	if !removed {
		return false, nil
	}
	s.data.Invites = kept
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func NormalizePermissions(raw []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, item := range raw {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func NormalizeAuthMethods(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return DefaultAuthMethods(), nil
	}
	seen := map[string]struct{}{}
	out := []string{}
	for _, item := range raw {
		method := NormalizeAuthMethod(item)
		if method == "" {
			return nil, fmt.Errorf("invalid auth method %q", item)
		}
		if _, ok := seen[method]; ok {
			continue
		}
		seen[method] = struct{}{}
		out = append(out, method)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one auth method is required")
	}
	sort.Strings(out)
	return out, nil
}

func NormalizeAuthMethod(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case AuthMethodEmail, "mail", "magic", "magic-link", "magic_link":
		return AuthMethodEmail
	case AuthMethodOIDC, "sso", "zitadel", "citatel":
		return AuthMethodOIDC
	default:
		return ""
	}
}

func NormalizeTenants(raw []string, fallback string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, item := range raw {
		item = textutil.Slug(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	if len(out) == 0 && fallback != "" {
		out = append(out, textutil.Slug(fallback))
	}
	sort.Strings(out)
	return out
}

func DefaultAuthMethods() []string {
	return []string{AuthMethodEmail, AuthMethodOIDC}
}

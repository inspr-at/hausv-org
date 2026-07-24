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
	// Adopted marks a store record as an admin-sanctioned override of a
	// config-sourced (env) user. Only records with this flag are allowed to win
	// over the env directory in directoryProfile; a plain or legacy store record
	// must never escalate an env user (HAUSV-135 anti-escalation, HAUSV-163
	// adopt-on-edit).
	Adopted bool `json:"adopted,omitempty"`
	// Deactivated suspends a user without deleting their record: they can no
	// longer sign in via any method. Break-glass ADMIN_EMAILS admins are exempt
	// (they always retain login) (HAUSV-163).
	Deactivated bool `json:"deactivated,omitempty"`
}

type TenantMembership struct {
	Role        string   `json:"role,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	// DirectoryOptIn overrides the person-wide directory visibility for THIS
	// house. nil means inherit, so a person who never set it per house keeps
	// behaving exactly as before (HAUSV-178).
	DirectoryOptIn *bool `json:"directory_opt_in,omitempty"`
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
		if membership.DirectoryOptIn != nil {
			out.DirectoryOptIn = *membership.DirectoryOptIn
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
// Mutate applies fn to a stored invite under ONE lock acquisition, so a
// per-field change (e.g. toggling one permission) can't be clobbered by a
// concurrent whole-profile write from another admin (HAUSV-145). Returns the
// updated profile and found=false if no invite matches.
func (s *InviteStore) Mutate(email string, fn func(*UserProfile)) (UserProfile, bool, error) {
	email = textutil.Email(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Invites {
		if textutil.Email(s.data.Invites[i].Email) == email {
			fn(&s.data.Invites[i])
			if err := s.saveLocked(); err != nil {
				return UserProfile{}, false, err
			}
			return s.data.Invites[i], true, nil
		}
	}
	return UserProfile{}, false, nil
}

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

func NormalizeTenantMemberships(raw map[string]TenantMembership) map[string]TenantMembership {
	if len(raw) == 0 {
		return nil
	}
	out := map[string]TenantMembership{}
	for slug, membership := range raw {
		slug = textutil.Slug(slug)
		if slug == "" {
			continue
		}
		membership.Role = NormalizeRole(membership.Role)
		if membership.Permissions != nil {
			membership.Permissions = NormalizePermissions(membership.Permissions)
		}
		out[slug] = membership
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func TenantMembershipSlugs(memberships map[string]TenantMembership) []string {
	slugs := []string{}
	for slug := range memberships {
		slug = textutil.Slug(slug)
		if slug != "" {
			slugs = append(slugs, slug)
		}
	}
	return slugs
}

// SetTenantMembership writes role and permissions for ONE house only, leaving
// the global identity and every other house's membership untouched. This is the
// operation a house admin is entitled to perform (HAUSV-135/169).
//
// The old path wrote the whole profile, so the top-level Role leaked into every
// house that had no explicit membership entry — a manager of house A could
// silently change someone's role in house B.
func (s *InviteStore) SetTenantMembership(email string, tenantSlug string, role string, permissions []string) (UserProfile, bool, error) {
	email = textutil.Email(email)
	tenantSlug = textutil.Slug(tenantSlug)
	if email == "" || tenantSlug == "" {
		return UserProfile{}, false, fmt.Errorf("invalid membership target")
	}
	return s.Mutate(email, func(p *UserProfile) {
		if p.TenantMemberships == nil {
			p.TenantMemberships = map[string]TenantMembership{}
		}
		// Pin every OTHER house to its current effective values first. Without
		// this the top-level role stays load-bearing for houses that have no
		// explicit entry, so changing it here would silently change the person's
		// role there too — the exact leak this method exists to prevent.
		materializeMemberships(p)
		// Normalize the key so a differently-cased slug cannot create a duplicate.
		for existing := range p.TenantMemberships {
			if textutil.Slug(existing) == tenantSlug && existing != tenantSlug {
				delete(p.TenantMemberships, existing)
			}
		}
		// Preserve this house's directory visibility: a role/permission edit
		// must not silently reset it (HAUSV-178).
		var directoryOptIn *bool
		if existing, ok := p.TenantMemberships[tenantSlug]; ok {
			directoryOptIn = existing.DirectoryOptIn
		}
		p.TenantMemberships[tenantSlug] = TenantMembership{
			Role:           NormalizeRole(role),
			Permissions:    NormalizePermissions(permissions),
			DirectoryOptIn: directoryOptIn,
		}
		p.Tenants = NormalizeTenants(append(p.Tenants, tenantSlug), "")
		syncProfileDefaults(p)
	})
}

// materializeMemberships gives every house the person belongs to an explicit
// membership, resolved from what is effective right now. Afterwards the flat
// top-level fields are only a default for houses that do not exist yet, and can
// never leak into an existing one.
func materializeMemberships(p *UserProfile) {
	if p.TenantMemberships == nil {
		p.TenantMemberships = map[string]TenantMembership{}
	}
	for _, tenant := range p.Tenants {
		slug := textutil.Slug(tenant)
		if slug == "" {
			continue
		}
		if _, ok := p.TenantMemberships[slug]; ok {
			continue
		}
		effective := p.ForTenant(slug)
		p.TenantMemberships[slug] = TenantMembership{
			Role:        NormalizeRole(effective.Role),
			Permissions: NormalizePermissions(effective.Permissions),
		}
	}
}

// syncProfileDefaults keeps the top-level Role/Permissions in step with the
// alphabetically first membership, mirroring how the SQLite model derives them
// from person+memberships. Without this the flat fields go stale after a
// house-scoped edit: ForTenant would still resolve correctly, but any code
// reading the raw profile would see a permission that was just revoked.
func syncProfileDefaults(p *UserProfile) {
	if len(p.TenantMemberships) == 0 {
		return
	}
	slugs := make([]string, 0, len(p.TenantMemberships))
	for slug := range p.TenantMemberships {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	first := p.TenantMemberships[slugs[0]]
	p.Role = NormalizeRole(first.Role)
	p.Permissions = NormalizePermissions(first.Permissions)
}

// RemoveTenant detaches a person from ONE house. The profile itself only
// disappears once no house is left, which keeps single-house behaviour
// identical to the previous whole-profile delete (HAUSV-135/169).
func (s *InviteStore) RemoveTenant(email string, tenantSlug string) (removedProfile bool, found bool, err error) {
	email = textutil.Email(email)
	tenantSlug = textutil.Slug(tenantSlug)
	if email == "" || tenantSlug == "" {
		return false, false, fmt.Errorf("invalid membership target")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Invites {
		if textutil.Email(s.data.Invites[i].Email) != email {
			continue
		}
		profile := s.data.Invites[i]
		// Same reason as in SetTenantMembership: pin the remaining houses before
		// changing anything, so none of them depends on the flat defaults.
		materializeMemberships(&profile)
		kept := []string{}
		for _, tenant := range profile.Tenants {
			if textutil.Slug(tenant) != tenantSlug {
				kept = append(kept, tenant)
			}
		}
		for key := range profile.TenantMemberships {
			if textutil.Slug(key) == tenantSlug {
				delete(profile.TenantMemberships, key)
			}
		}
		profile.Tenants = NormalizeTenants(kept, "")
		if len(profile.Tenants) == 0 && len(profile.TenantMemberships) == 0 {
			s.data.Invites = append(s.data.Invites[:i], s.data.Invites[i+1:]...)
			if err := s.saveLocked(); err != nil {
				return false, true, err
			}
			return true, true, nil
		}
		syncProfileDefaults(&profile)
		s.data.Invites[i] = profile
		if err := s.saveLocked(); err != nil {
			return false, true, err
		}
		return false, true, nil
	}
	return false, false, nil
}

// SetTenantDirectoryOptIn sets contact-directory visibility for ONE house
// (HAUSV-178).
func (s *InviteStore) SetTenantDirectoryOptIn(email string, tenantSlug string, optIn bool) (bool, error) {
	email = textutil.Email(email)
	tenantSlug = textutil.Slug(tenantSlug)
	if email == "" || tenantSlug == "" {
		return false, fmt.Errorf("invalid directory target")
	}
	_, found, err := s.Mutate(email, func(p *UserProfile) {
		materializeMemberships(p)
		membership := p.TenantMemberships[tenantSlug]
		value := optIn
		membership.DirectoryOptIn = &value
		p.TenantMemberships[tenantSlug] = membership
	})
	return found, err
}

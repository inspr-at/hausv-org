package store

import (
	"fmt"
	"time"

	"github.com/markus-barta/hausv-org/internal/textutil"
)

// ProfileStorage is the surface the server uses for app-managed user records.
// Both the JSON InviteStore and the SQLite identity model satisfy it, so the
// backend is swappable exactly like every other store (HAUSV-169).
//
// The last two methods are the point of the N:N rework: they are scoped to ONE
// house, so a house admin can no longer rewrite or delete a person globally.
type ProfileStorage interface {
	Get(email string) (UserProfile, bool)
	List() []UserProfile
	Add(profile UserProfile) (bool, error)
	Update(oldEmail string, updated UserProfile) (bool, error)
	Delete(email string) (bool, error)
	Mutate(email string, fn func(*UserProfile)) (UserProfile, bool, error)
	SetTenantMembership(email string, tenantSlug string, role string, permissions []string) (UserProfile, bool, error)
	RemoveTenant(email string, tenantSlug string) (removedProfile bool, found bool, err error)
	// SetTenantDirectoryOptIn sets contact-directory visibility for ONE house.
	// The directory is rendered per house, so the switch belongs there
	// (HAUSV-178).
	SetTenantDirectoryOptIn(email string, tenantSlug string, optIn bool) (bool, error)
}

var (
	_ ProfileStorage = (*InviteStore)(nil)
	_ ProfileStorage = (*SQLIdentityStore)(nil)
)

// profileFromPerson rebuilds the flat UserProfile the rest of the app speaks
// from a person plus their memberships. Memberships are ordered by house, so
// the derived top-level defaults are deterministic; per-house resolution still
// happens through UserProfile.ForTenant.
func (s *SQLIdentityStore) profileFromPerson(p Person) UserProfile {
	profile := UserProfile{
		Email:       p.Email,
		Title:       p.Title,
		FirstName:   p.FirstName,
		LastName:    p.LastName,
		AuthMethods: p.AuthMethods,
		Deactivated: p.Deactivated,
		Adopted:     p.Adopted,
	}
	memberships := s.MembershipsForPerson(p.ID)
	if len(memberships) == 0 {
		return profile
	}
	profile.TenantMemberships = map[string]TenantMembership{}
	for _, m := range memberships {
		profile.Tenants = append(profile.Tenants, m.TenantSlug)
		profile.TenantMemberships[m.TenantSlug] = TenantMembership{Role: m.Role, Permissions: m.Permissions, DirectoryOptIn: m.DirectoryOptIn}
	}
	// Top-level role/permissions/status act as the default for houses without an
	// explicit entry; the first membership supplies them.
	profile.Role = memberships[0].Role
	profile.Permissions = memberships[0].Permissions
	profile.Status = memberships[0].Status
	return profile
}

func (s *SQLIdentityStore) Get(email string) (UserProfile, bool) {
	if s == nil {
		return UserProfile{}, false
	}
	person, ok := s.PersonByEmail(email)
	if !ok {
		return UserProfile{}, false
	}
	return s.profileFromPerson(person), true
}

func (s *SQLIdentityStore) List() []UserProfile {
	if s == nil {
		return nil
	}
	out := []UserProfile{}
	for _, person := range s.ListPersons() {
		out = append(out, s.profileFromPerson(person))
	}
	return out
}

// Add persists a new person plus their memberships. Returns false (no error)
// when the email already exists, matching InviteStore.Add.
func (s *SQLIdentityStore) Add(profile UserProfile) (bool, error) {
	if s == nil {
		return false, nil
	}
	email := textutil.Email(profile.Email)
	if email == "" {
		return false, nil
	}
	if _, exists := s.PersonByEmail(email); exists {
		return false, nil
	}
	if err := s.writeProfile(profile, time.Now()); err != nil {
		return false, err
	}
	return true, nil
}

// writeProfile upserts the person and reconciles their memberships to exactly
// the houses named by the profile. Used by the whole-profile paths (Add/Update/
// Mutate); the house-scoped paths never call it.
func (s *SQLIdentityStore) writeProfile(profile UserProfile, at time.Time) error {
	person, err := s.UpsertPerson(Person{
		Email:       profile.Email,
		Title:       profile.Title,
		FirstName:   profile.FirstName,
		LastName:    profile.LastName,
		AuthMethods: profile.AuthMethods,
		Deactivated: profile.Deactivated,
		Adopted:     profile.Adopted,
	}, at)
	if err != nil {
		return err
	}
	wanted := map[string]struct{}{}
	for _, tenant := range tenantsOfProfile(profile) {
		wanted[tenant] = struct{}{}
		resolved := profile.ForTenant(tenant)
		var directoryOptIn *bool
		if m, ok := profile.TenantMemberships[tenant]; ok {
			directoryOptIn = m.DirectoryOptIn
		}
		if _, err := s.SetMembership(HouseMembership{
			PersonID:       person.ID,
			TenantSlug:     tenant,
			Role:           resolved.Role,
			Permissions:    resolved.Permissions,
			Status:         profile.Status,
			DirectoryOptIn: directoryOptIn,
		}, at); err != nil {
			return err
		}
	}
	for _, existing := range s.MembershipsForPerson(person.ID) {
		if _, keep := wanted[existing.TenantSlug]; !keep {
			if _, err := s.RemoveMembership(person.ID, existing.TenantSlug); err != nil {
				return err
			}
		}
	}
	return nil
}

// Update replaces the record keyed by oldEmail. This is the GLOBAL path — it is
// reachable from the platform-admin workflow, not from a house admin, whose
// edits go through SetTenantMembership instead.
func (s *SQLIdentityStore) Update(oldEmail string, updated UserProfile) (bool, error) {
	if s == nil {
		return false, nil
	}
	oldEmail = textutil.Email(oldEmail)
	newEmail := textutil.Email(updated.Email)
	person, ok := s.PersonByEmail(oldEmail)
	if !ok {
		return false, nil
	}
	if newEmail != oldEmail {
		if other, exists := s.PersonByEmail(newEmail); exists && other.ID != person.ID {
			return false, fmt.Errorf("email already invited")
		}
		if _, err := s.ChangePersonEmail(person.ID, newEmail, time.Now()); err != nil {
			return false, err
		}
	}
	if err := s.writeProfile(updated, time.Now()); err != nil {
		return false, err
	}
	return true, nil
}

// Delete removes the person and, by cascade, every membership. The house-scoped
// counterpart is RemoveTenant.
func (s *SQLIdentityStore) Delete(email string) (bool, error) {
	if s == nil {
		return false, nil
	}
	person, ok := s.PersonByEmail(email)
	if !ok {
		return false, nil
	}
	return s.DeletePerson(person.ID)
}

func (s *SQLIdentityStore) Mutate(email string, fn func(*UserProfile)) (UserProfile, bool, error) {
	if s == nil {
		return UserProfile{}, false, nil
	}
	person, ok := s.PersonByEmail(email)
	if !ok {
		return UserProfile{}, false, nil
	}
	profile := s.profileFromPerson(person)
	fn(&profile)
	if err := s.writeProfile(profile, time.Now()); err != nil {
		return UserProfile{}, false, err
	}
	fresh, _ := s.PersonByEmail(textutil.Email(profile.Email))
	return s.profileFromPerson(fresh), true, nil
}

// SetTenantMembership writes role and permissions for ONE house. The person's
// global identity and every other membership are untouched — enforced by the
// schema, not by convention (HAUSV-135/169).
func (s *SQLIdentityStore) SetTenantMembership(email string, tenantSlug string, role string, permissions []string) (UserProfile, bool, error) {
	if s == nil {
		return UserProfile{}, false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	if textutil.Email(email) == "" || tenantSlug == "" {
		return UserProfile{}, false, fmt.Errorf("invalid membership target")
	}
	person, ok := s.PersonByEmail(email)
	if !ok {
		return UserProfile{}, false, nil
	}
	status := ""
	var directoryOptIn *bool
	if existing, had := s.Membership(person.ID, tenantSlug); had {
		status = existing.Status
		directoryOptIn = existing.DirectoryOptIn
	}
	if _, err := s.SetMembership(HouseMembership{
		PersonID:       person.ID,
		TenantSlug:     tenantSlug,
		Role:           role,
		Permissions:    permissions,
		Status:         status,
		DirectoryOptIn: directoryOptIn,
	}, time.Now()); err != nil {
		return UserProfile{}, false, err
	}
	fresh, _ := s.PersonByEmail(email)
	return s.profileFromPerson(fresh), true, nil
}

// RemoveTenant detaches the person from ONE house; the person disappears only
// when no house is left, so single-house behaviour matches the old delete.
func (s *SQLIdentityStore) RemoveTenant(email string, tenantSlug string) (bool, bool, error) {
	if s == nil {
		return false, false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	if textutil.Email(email) == "" || tenantSlug == "" {
		return false, false, fmt.Errorf("invalid membership target")
	}
	person, ok := s.PersonByEmail(email)
	if !ok {
		return false, false, nil
	}
	if _, err := s.RemoveMembership(person.ID, tenantSlug); err != nil {
		return false, true, err
	}
	if len(s.MembershipsForPerson(person.ID)) == 0 {
		removed, err := s.DeletePerson(person.ID)
		return removed, true, err
	}
	return false, true, nil
}

func (s *SQLIdentityStore) SetTenantDirectoryOptIn(email string, tenantSlug string, optIn bool) (bool, error) {
	if s == nil {
		return false, nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	if textutil.Email(email) == "" || tenantSlug == "" {
		return false, fmt.Errorf("invalid directory target")
	}
	person, ok := s.PersonByEmail(email)
	if !ok {
		return false, nil
	}
	existing, had := s.Membership(person.ID, tenantSlug)
	if !had {
		return false, nil
	}
	existing.DirectoryOptIn = &optIn
	if _, err := s.SetMembership(existing, time.Now()); err != nil {
		return false, err
	}
	return true, nil
}

package server

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

// organisationRecord is what a page renders: the stored organisation where one
// exists, otherwise the configured name so a deployment without the table (or
// without an organisation at all) reads exactly as it did before.
type organisationRecord struct {
	Key            string
	Name           string
	ContactName    string
	ContactAddress string
	ContactEmail   string
	ContactPhone   string
	// Houses are the organisation's tenant slugs. Authorization never reads
	// this: a person still sees only houses they are a member of.
	Houses []string
	Stored bool
}

// syncOrganisations mirrors the configured organisations into the store at boot
// and reconciles which houses each one administers.
//
// Configuration stays the deployment's truth for which houses exist, so the
// house set is rewritten on every boot. The organisation's own fields are
// written only when the row is new — otherwise a boot would silently undo
// contact data someone edited in the app.
func (a *app) syncOrganisations(ctx context.Context) {
	if a == nil || a.organisationRepo == nil {
		return
	}
	for key, organisation := range a.organisations {
		key = normalizeSlug(key)
		if key == "" {
			continue
		}
		repo := a.organisationRepo(key)
		if repo == nil {
			continue
		}
		houses := a.configuredOrganisationHouses(key)
		stored, ok, err := repo.Get(ctx)
		if err != nil {
			logError("organisation could not be read at boot", err, "organisation", key)
			continue
		}
		if !ok {
			name := strings.TrimSpace(organisation.Name)
			if name == "" {
				name = key
			}
			if err := repo.Save(ctx, store.Organisation{Key: key, Name: name, Houses: houses}); err != nil {
				logError("organisation could not be seeded", err, "organisation", key)
			}
			continue
		}
		if sameSlugs(stored.Houses, houses) {
			continue
		}
		if err := repo.SetHouses(ctx, houses); err != nil {
			logError("organisation houses could not be reconciled", err, "organisation", key)
		}
	}
}

func (a *app) configuredOrganisationHouses(orgKey string) []string {
	orgKey = normalizeSlug(orgKey)
	if a == nil || orgKey == "" {
		return nil
	}
	houses := make([]string, 0, len(a.tenants))
	for slug, tenant := range a.tenants {
		if normalizeSlug(tenant.Organisation) != orgKey {
			continue
		}
		houses = append(houses, normalizeSlug(slug))
	}
	return store.NormalizeHouseSlugs(houses)
}

func sameSlugs(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// organisationRecordFor resolves the organisation for this request. It prefers
// the stored row and falls back to configuration, so an unmigrated or
// organisation-less deployment behaves as before.
func (a *app) organisationRecordFor(ctx context.Context, ac *authCtx) (organisationRecord, bool) {
	configured, ok := a.organisationFor(ac)
	if !ok {
		return organisationRecord{}, false
	}
	record := organisationRecord{
		Key:    normalizeSlug(configured.Key),
		Name:   strings.TrimSpace(configured.Name),
		Houses: a.configuredOrganisationHouses(configured.Key),
	}
	if record.Name == "" {
		record.Name = record.Key
	}
	if a.orgSettings != nil {
		if repo := a.orgSettings(record.Key); repo != nil {
			settings, err := repo.Get(ctx)
			if err != nil {
				logError("organisation address could not be read", err, "organisation", record.Key)
			} else {
				record.ContactAddress = settings.ContactAddress
			}
		}
	}
	if a.organisationRepo == nil || record.Key == "" {
		return record, true
	}
	repo := a.organisationRepo(record.Key)
	if repo == nil {
		return record, true
	}
	stored, found, err := repo.Get(ctx)
	if err != nil {
		logError("organisation could not be read", err, "organisation", record.Key)
		return record, true
	}
	if !found {
		return record, true
	}
	record.Name = stored.Name
	record.ContactName = stored.ContactName
	record.ContactEmail = stored.ContactEmail
	record.ContactPhone = stored.ContactPhone
	record.Stored = true
	if len(stored.Houses) > 0 {
		record.Houses = stored.Houses
	}
	return record, true
}

// organisationManagedTenants lists the houses of the organisation, in the
// organisation's order, restricted to the ones this person actually manages.
// The restriction is the point: the organisation says which houses belong to
// the Verwaltung, membership still says which of them a person may open.
func (a *app) organisationManagedTenants(ctx context.Context, ac *authCtx) []managedTenant {
	managed := a.managedTenants(ac)
	record, ok := a.organisationRecordFor(ctx, ac)
	if !ok || len(record.Houses) == 0 {
		return managed
	}
	position := make(map[string]int, len(record.Houses))
	for index, slug := range record.Houses {
		position[slug] = index
	}
	inOrganisation := make([]managedTenant, 0, len(managed))
	others := make([]managedTenant, 0, len(managed))
	for _, tenant := range managed {
		if _, ok := position[normalizeSlug(tenant.Ref.Slug)]; ok {
			inOrganisation = append(inOrganisation, tenant)
			continue
		}
		others = append(others, tenant)
	}
	sort.SliceStable(inOrganisation, func(i, j int) bool {
		return position[normalizeSlug(inOrganisation[i].Ref.Slug)] < position[normalizeSlug(inOrganisation[j].Ref.Slug)]
	})
	// A house someone manages outside the organisation stays visible; dropping
	// it would hide work rather than tidy a list.
	return append(inOrganisation, others...)
}

func (a *app) saveOrganisationContact(ctx context.Context, record organisationRecord) error {
	if a == nil || a.organisationRepo == nil {
		return nil
	}
	repo := a.organisationRepo(record.Key)
	if repo == nil {
		return nil
	}
	houses := record.Houses
	if len(houses) == 0 {
		houses = a.configuredOrganisationHouses(record.Key)
	}
	return repo.Save(ctx, store.Organisation{
		Key:          record.Key,
		Name:         record.Name,
		ContactName:  record.ContactName,
		ContactEmail: record.ContactEmail,
		ContactPhone: record.ContactPhone,
		Houses:       houses,
		UpdatedAt:    time.Now().UTC(),
	})
}

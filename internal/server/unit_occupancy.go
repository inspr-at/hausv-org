package server

import (
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

// unitOccupancy resolves the recorded parties for a displayed unit label. The
// unit repository, rather than the message sender, remains the authority for
// the assignment shown in the portal.
func (a *app) unitOccupancy(tenantSlug, unitLabel string) web.UnitOccupancy {
	occupancy := web.UnitOccupancy{UnitLabel: strings.TrimSpace(unitLabel)}
	if a == nil || occupancy.UnitLabel == "" {
		return occupancy
	}
	identity, ok := a.tenantIdentity(tenantSlug)
	if !ok {
		return occupancy
	}
	units := a.repositoriesFor(identity.Ref()).units
	if units == nil {
		return occupancy
	}
	want := store.NormalizeUnitID(occupancy.UnitLabel)
	for _, unit := range units.List() {
		if store.NormalizeUnitID(unit.Label) != want && store.NormalizeUnitID(unit.ID) != want {
			continue
		}
		occupancy.Found = true
		occupancy.UnitLabel = unit.Label
		occupancy.UnitType = unitTypeLabel(unit.UnitType)
		occupancy.Owners = a.occupancyPeople(tenantSlug, unit.OwnerEmails)
		occupancy.Renters = a.occupancyPeople(tenantSlug, unit.RenterEmails)
		addOccupancyPartyDates(occupancy.Owners, unit)
		addOccupancyPartyDates(occupancy.Renters, unit)
		return occupancy
	}
	return occupancy
}

func (a *app) occupancyPeople(tenantSlug string, emails []string) []web.Person {
	people := make([]web.Person, 0, len(emails))
	seen := map[string]struct{}{}
	for _, raw := range emails {
		email := normalizeEmail(raw)
		if email == "" {
			continue
		}
		if _, exists := seen[email]; exists {
			continue
		}
		seen[email] = struct{}{}
		profile := a.profileForTenant(email, tenantSlug)
		name := strings.TrimSpace(profile.DisplayName())
		if name == "" || normalizeEmail(name) == email {
			name = occupancyFallbackName(email)
		}
		people = append(people, web.Person{Email: email, Name: name, Title: strings.TrimSpace(profile.Title)})
	}
	return people
}

func occupancyFallbackName(email string) string {
	local, _, _ := strings.Cut(normalizeEmail(email), "@")
	words := strings.FieldsFunc(local, func(r rune) bool {
		return r == '.' || r == '_' || r == '-'
	})
	for index, word := range words {
		letters := []rune(strings.ToLower(word))
		if len(letters) > 0 {
			letters[0] = unicode.ToUpper(letters[0])
		}
		words[index] = string(letters)
	}
	return strings.Join(words, " ")
}

func occupancyLabel(occupancy web.UnitOccupancy) string {
	if len(occupancy.Renters) > 0 {
		return occupancyRole("Mieter", occupancy.Renters[0]) + " " + occupancyPersonLabel(occupancy.Renters)
	}
	if len(occupancy.Owners) > 0 {
		return occupancyRole("Eigentümer", occupancy.Owners[0]) + " " + occupancyPersonLabel(occupancy.Owners)
	}
	if occupancy.Found {
		return "unbewohnt"
	}
	return "nicht zugeordnet"
}

func occupancyRole(role string, person web.Person) string {
	if strings.Contains(strings.ToLower(person.Title), "frau") {
		return role + "in"
	}
	return role
}

func occupancyPersonLabel(people []web.Person) string {
	if len(people) == 0 {
		return ""
	}
	if len(people) == 1 {
		return people[0].Name
	}
	return people[0].Name + " +" + strconv.Itoa(len(people)-1)
}

func (a *app) unitOccupancies(tenant store.TenantRef) ([]web.UnitOccupancy, error) {
	if a == nil {
		return nil, nil
	}
	units := a.repositoriesFor(tenant).units
	if units == nil {
		return nil, nil
	}
	items, err := units.ListChecked()
	if err != nil {
		return nil, err
	}
	occupancies := make([]web.UnitOccupancy, 0, len(items))
	for _, unit := range items {
		occupancies = append(occupancies, a.unitOccupancy(tenant.Slug, unit.Label))
	}
	sort.SliceStable(occupancies, func(i, j int) bool {
		leftParking := normalizeUnitType(occupancies[i].UnitType) == unitTypeParking
		rightParking := normalizeUnitType(occupancies[j].UnitType) == unitTypeParking
		if leftParking != rightParking {
			return !leftParking
		}
		return store.UnitLabelLess(occupancies[i].UnitLabel, occupancies[j].UnitLabel)
	})
	return occupancies, nil
}

func (a *app) buildingUnitViewsWithOccupancy(repositories requestRepositories, tenant store.TenantRef, units []unit) []buildingUnitView {
	views := a.buildingUnitViewsWithPayments(repositories, tenant, units)
	for index := range views {
		occupancy := a.unitOccupancy(tenant.Slug, views[index].Label)
		views[index].OwnerSummary = occupancyPeopleSummary(occupancy.Owners, "nicht zugeordnet")
		views[index].RenterSummary = occupancyPeopleSummary(occupancy.Renters, "frei")
	}
	sort.SliceStable(views, func(i, j int) bool {
		leftParking := normalizeUnitType(views[i].UnitType) == unitTypeParking
		rightParking := normalizeUnitType(views[j].UnitType) == unitTypeParking
		if leftParking != rightParking {
			return !leftParking
		}
		return store.UnitLabelLess(views[i].Label, views[j].Label)
	})
	return views
}

func occupancyPeopleSummary(people []web.Person, empty string) string {
	if len(people) == 0 {
		return empty
	}
	names := make([]string, 0, len(people))
	for _, person := range people {
		names = append(names, person.Name)
	}
	return strings.Join(names, ", ")
}

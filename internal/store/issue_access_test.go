package store

import (
	"testing"
)

func TestIssueAccessPrivateUnitsAndCommonAreas(t *testing.T) {
	tenant := testTenantRef("demo")
	owner := UnitMembership{Unit: Unit{ID: "top-2", Label: "Top 2"}, Relation: RoleOwner}
	renter := UnitMembership{Unit: Unit{ID: "top-3", Label: "Top 3"}, Relation: RoleRenter}
	roles := []struct {
		name   string
		access IssueAccess
		want   []bool
	}{
		{"board-owner", IssueAccess{Email: "board@example.com", Board: true, Units: []UnitMembership{owner}}, []bool{true, true, true, false, false, false, true, false}},
		{"board-renter", IssueAccess{Email: "board@example.com", Board: true, Units: []UnitMembership{renter}}, []bool{true, true, false, true, false, false, true, false}},
		{"board-no-unit", IssueAccess{Email: "board@example.com", Board: true}, []bool{true, true, false, false, false, false, true, false}},
		{"owner", IssueAccess{Email: "owner@example.com", Common: true, Units: []UnitMembership{owner}}, []bool{true, true, false, false, false, false, false, false}},
		{"resident", IssueAccess{Email: "resident@example.com"}, []bool{false, false, false, false, false, false, false, false}},
		{"author", IssueAccess{Email: "author@example.com"}, []bool{true, true, true, true, true, true, false, true}},
		{"manager", IssueAccess{Email: "manager@example.com", Manage: true}, []bool{true, true, true, true, true, true, true, true}},
	}
	issues := []ResidentIssue{
		{LocationType: IssueLocationCommon},
		{LocationType: IssueLocationUnit, LocationDetail: "Allgemeinflächen"},
		{LocationType: IssueLocationUnit, LocationDetail: "Top 2"},
		{LocationType: IssueLocationUnit, UnitID: "top-3", LocationDetail: "Badezimmer"},
		{LocationType: IssueLocationUnit, LocationDetail: "Top 20"},
		{LocationType: IssueLocationUnit},
		{LocationType: IssueLocationUnit, AuthorEmail: "board@example.com"},
		{LocationType: IssueLocationUnit, UnitID: "top-9", LocationDetail: "Top 2"},
	}
	for _, role := range roles {
		t.Run(role.name, func(t *testing.T) {
			for i, item := range issues {
				item.TenantSlug = tenant.Slug
				if item.AuthorEmail == "" {
					item.AuthorEmail = "author@example.com"
				}
				if got := role.access.CanView(tenant, item); got != role.want[i] {
					t.Errorf("issue %d visible=%v want=%v", i, got, role.want[i])
				}
				item.TenantSlug = "another-house"
				if role.access.CanView(tenant, item) {
					t.Errorf("issue %d crossed tenant boundary", i)
				}
			}
		})
	}
}

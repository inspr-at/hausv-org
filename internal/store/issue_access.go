package store

import (
	"strings"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// IssueAccess carries server-resolved permissions and tenant-bound unit assignments.
// Lists, details, attachments and counts must use this single visibility policy.
type IssueAccess struct {
	Email           string
	Manage          bool
	Board           bool
	Common          bool
	ServiceProvider bool
	Units           []UnitMembership
}

func (a IssueAccess) CanView(tenant TenantRef, item ResidentIssue) bool {
	email := textutil.Email(a.Email)
	if tenant.Slug == "" || textutil.Slug(item.TenantSlug) != tenant.Slug || email == "" {
		return false
	}
	if a.Manage {
		return true
	}
	if a.ServiceProvider {
		switch NormalizeIssueStatus(item.Status) {
		case IssueStatusDone, IssueStatusRejected, IssueStatusDuplicate:
			return false
		}
		return textutil.Email(item.AssigneeEmail) == email
	}
	if textutil.Email(item.AuthorEmail) == email {
		return true
	}
	if IssueIsCommon(item) {
		return a.Board || a.Common
	}
	if a.Board {
		for _, membership := range a.Units {
			if item.UnitID != "" {
				if NormalizeUnitID(item.UnitID) == NormalizeUnitID(membership.Unit.ID) {
					return true
				}
			} else if IssueLocationMatchesUnit(item.LocationDetail, membership.Unit) {
				return true
			}
		}
	}
	return false
}

// Legacy intake records use own-unit even for explicit Allgemeinflächen.
// An empty private location is ambiguous and must never become public.
func IssueIsCommon(item ResidentIssue) bool {
	if item.UnitID != "" {
		return false
	}
	return NormalizeIssueLocation(item.LocationType) == IssueLocationCommon || strings.EqualFold(strings.TrimSpace(item.LocationDetail), "Allgemeinflächen")
}

func IssueLocationMatchesUnit(location string, unit Unit) bool {
	location = strings.TrimSpace(location)
	return location != "" && (strings.EqualFold(location, strings.TrimSpace(unit.Label)) || strings.EqualFold(location, unit.ID))
}

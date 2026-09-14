package store

import "strings"

// Domain vocabulary. These are the values that get PERSISTED and validated, so
// they belong to the store rather than to the UI. main aliases them, so call
// sites read unchanged.
//
// Values are verbatim from the original const block — a changed value here is a
// silent data-format change, not a rename.
const (
	RoleAdmin           = "Admin"
	RoleManager         = "Verwalter"
	RoleOwner           = "Eigentümer"
	RoleRenter          = "Mieter"
	RoleBeirat          = "Beirat"
	RoleResident        = "Bewohner"
	RoleServiceProvider = "Dienstleister"

	PermissionParking         = "parking"
	PermissionEnergyView      = "energy-view"
	PermissionEnergyConfigure = "energy-configure"
	PermissionEnergyControl   = "energy-control"
	PermissionSupportView     = "support-view"
	// PermissionEnergyCaretaker is the legacy combined view/configure grant.
	// It remains readable so existing profiles do not lose access.
	PermissionEnergyCaretaker = "energy-caretaker"

	AuthMethodEmail = "email"
	AuthMethodOIDC  = "oidc"

	IssueStatusNew       = "Neu"
	IssueStatusAccepted  = "Angenommen"
	IssueStatusScheduled = "Termin vereinbart"
	IssueStatusProgress  = "In Bearbeitung"
	IssueStatusDone      = "Erledigt"
	IssueStatusRejected  = "Abgelehnt"
	IssueStatusDuplicate = "Duplikat"
	IssueStatusOpen      = IssueStatusNew

	IssueCommentKindNeutral     = ""
	IssueCommentKindInformation = "information"
	IssueCommentKindQuestion    = "question"
	IssueCommentKindAnswer      = "answer"

	IssuePriorityLow    = "Niedrig"
	IssuePriorityNorm   = "Mittel"
	IssuePriorityHigh   = "Hoch"
	IssuePriorityUrgent = "Dringend"

	IssueLocationUnit   = "own-unit"
	IssueLocationCommon = "common"
)

func NormalizeIssueCommentKind(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case IssueCommentKindInformation:
		return IssueCommentKindInformation
	case IssueCommentKindQuestion:
		return IssueCommentKindQuestion
	case IssueCommentKindAnswer:
		return IssueCommentKindAnswer
	default:
		return IssueCommentKindNeutral
	}
}

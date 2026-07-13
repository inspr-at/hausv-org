package store

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

	PermissionParking = "parking"

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

	IssuePriorityLow    = "Niedrig"
	IssuePriorityNorm   = "Mittel"
	IssuePriorityHigh   = "Hoch"
	IssuePriorityUrgent = "Dringend"

	IssueLocationUnit   = "own-unit"
	IssueLocationCommon = "common"
)

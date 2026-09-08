package authz

import "github.com/inspr-at/hausv-org/internal/store"

// MatrixAction is one of the five operations shown in the portal rights
// matrix. The labels deliberately match the German, user-facing vocabulary.
type MatrixAction string

const (
	MatrixActionView    MatrixAction = "sehen"
	MatrixActionCreate  MatrixAction = "anlegen"
	MatrixActionChange  MatrixAction = "ändern"
	MatrixActionApprove MatrixAction = "freigeben"
	MatrixActionDelete  MatrixAction = "löschen"
)

// MatrixFamily describes a product-facing role family. RepresentativeRole is
// used only for capability-backed decisions; Roles documents every internal
// role that is grouped under the family in the portal.
type MatrixFamily struct {
	Key                string
	Label              string
	RepresentativeRole string
	Roles              []string
}

// MatrixArea is one product area displayed in the rights matrix.
type MatrixArea struct {
	Key   string
	Label string
}

// MatrixGrant is an allowed operation in one matrix cell. Capability is set
// whenever authz.Can is the policy source. An empty Capability marks an
// explicit product rule for an area that has no dedicated capability yet; it
// must not be interpreted as a new authorization grant by callers.
type MatrixGrant struct {
	Action     MatrixAction
	Capability Capability
	ActorRole  string
	Note       string
}

// MatrixCell is the single source rendered by the portal rights page. Missing
// grants are denied; consumers must never use this presentation matrix as an
// authorization check.
type MatrixCell struct {
	FamilyKey string
	AreaKey   string
	Grants    []MatrixGrant
	Note      string
}

// MatrixSpecialRole documents narrow rights which are not role capabilities.
// They remain enforced by their assignment- or permission-specific policies.
type MatrixSpecialRole struct {
	Label   string
	Area    string
	Actions []MatrixAction
	Scope   string
}

var AllCapabilities = []Capability{
	CapabilityPlatformAdmin,
	CapabilityManageUsers,
	CapabilityManageParking,
	CapabilityManageAnnouncements,
	CapabilityManageDocuments,
	CapabilityManageIssues,
	CapabilityManageVotes,
	CapabilityManageBuilding,
	CapabilityOwnerDocuments,
	CapabilityVote,
	CapabilityOversight,
	CapabilityManageEnergy,
	CapabilityControlEnergy,
}

var AllRoles = []string{
	store.RoleAdmin,
	store.RoleManager,
	store.RoleOwner,
	store.RoleBeirat,
	store.RoleRenter,
	store.RoleResident,
	store.RoleServiceProvider,
}

var RoleFamilies = []MatrixFamily{
	{Key: "verwaltung", Label: "Hausverwaltung", RepresentativeRole: store.RoleManager, Roles: []string{store.RoleAdmin, store.RoleManager}},
	{Key: "eigentuemer", Label: "Eigentümer", RepresentativeRole: store.RoleOwner, Roles: []string{store.RoleOwner, store.RoleBeirat}},
	{Key: "bewohner", Label: "Bewohner", RepresentativeRole: store.RoleResident, Roles: []string{store.RoleRenter, store.RoleResident}},
}

var RoleAreas = []MatrixArea{
	{Key: "portfolio", Label: "Portfolio"},
	{Key: "inbox", Label: "Posteingang"},
	{Key: "settings", Label: "Einstellungen"},
	{Key: "snippets", Label: "Textbausteine"},
	{Key: "overview", Label: "Hausüberblick"},
	{Key: "issues", Label: "Anliegen"},
	{Key: "documents", Label: "Dokumente"},
	{Key: "announcements", Label: "Ankündigungen"},
	{Key: "events", Label: "Termine"},
	{Key: "votes", Label: "Abstimmungen"},
	{Key: "handovers", Label: "Übergaben"},
	{Key: "contacts", Label: "Kontakte"},
	{Key: "units-payments", Label: "Einheiten/Zahlungen"},
	{Key: "energy", Label: "Energie"},
	{Key: "account", Label: "Konto"},
}

var RoleMatrix = []MatrixCell{
	{FamilyKey: "verwaltung", AreaKey: "portfolio", Grants: capabilityGrants(CapabilityManageIssues, store.RoleManager, MatrixActionView)},
	{FamilyKey: "verwaltung", AreaKey: "inbox", Grants: capabilityGrants(CapabilityManageIssues, store.RoleManager, MatrixActionView, MatrixActionCreate, MatrixActionChange, MatrixActionApprove)},
	{FamilyKey: "verwaltung", AreaKey: "settings", Grants: capabilityGrants(CapabilityPlatformAdmin, store.RoleAdmin, MatrixActionView, MatrixActionChange), Note: "nur Organisationsadministration"},
	{FamilyKey: "verwaltung", AreaKey: "snippets", Note: "noch keine eigene Capability und kein eigener Portalbereich"},
	{FamilyKey: "verwaltung", AreaKey: "overview", Grants: explicitGrants("Mandantschaft und Liegenschaftszuordnung", MatrixActionView)},
	{FamilyKey: "verwaltung", AreaKey: "issues", Grants: capabilityGrants(CapabilityManageIssues, store.RoleManager, MatrixActionView, MatrixActionCreate, MatrixActionChange, MatrixActionApprove)},
	{FamilyKey: "verwaltung", AreaKey: "documents", Grants: capabilityGrants(CapabilityManageDocuments, store.RoleManager, MatrixActionView, MatrixActionCreate, MatrixActionChange)},
	{FamilyKey: "verwaltung", AreaKey: "announcements", Grants: capabilityGrants(CapabilityManageAnnouncements, store.RoleManager, MatrixActionView, MatrixActionCreate, MatrixActionChange, MatrixActionApprove, MatrixActionDelete)},
	{FamilyKey: "verwaltung", AreaKey: "events", Grants: capabilityGrants(CapabilityManageAnnouncements, store.RoleManager, MatrixActionView, MatrixActionCreate, MatrixActionChange, MatrixActionApprove, MatrixActionDelete)},
	{FamilyKey: "verwaltung", AreaKey: "votes", Grants: capabilityGrants(CapabilityManageVotes, store.RoleManager, MatrixActionView, MatrixActionCreate, MatrixActionChange, MatrixActionApprove)},
	{FamilyKey: "verwaltung", AreaKey: "handovers", Grants: capabilityGrants(CapabilityManageDocuments, store.RoleManager, MatrixActionView, MatrixActionCreate, MatrixActionChange, MatrixActionApprove)},
	{FamilyKey: "verwaltung", AreaKey: "contacts", Grants: capabilityGrants(CapabilityManageUsers, store.RoleManager, MatrixActionView, MatrixActionCreate, MatrixActionChange, MatrixActionDelete)},
	{FamilyKey: "verwaltung", AreaKey: "units-payments", Grants: capabilityGrants(CapabilityManageBuilding, store.RoleManager, MatrixActionView, MatrixActionCreate, MatrixActionChange, MatrixActionDelete)},
	{FamilyKey: "verwaltung", AreaKey: "energy", Grants: []MatrixGrant{
		{Action: MatrixActionView, Capability: CapabilityManageEnergy, ActorRole: store.RoleManager},
		{Action: MatrixActionChange, Capability: CapabilityManageEnergy, ActorRole: store.RoleManager},
		{Action: MatrixActionApprove, Capability: CapabilityControlEnergy, ActorRole: store.RoleManager},
	}},
	{FamilyKey: "verwaltung", AreaKey: "account", Grants: explicitGrants("eigenes Konto", MatrixActionView, MatrixActionChange)},

	{FamilyKey: "eigentuemer", AreaKey: "portfolio"},
	{FamilyKey: "eigentuemer", AreaKey: "inbox"},
	{FamilyKey: "eigentuemer", AreaKey: "settings"},
	{FamilyKey: "eigentuemer", AreaKey: "snippets"},
	{FamilyKey: "eigentuemer", AreaKey: "overview", Grants: explicitGrants("Mandantschaft und Liegenschaftszuordnung", MatrixActionView)},
	{FamilyKey: "eigentuemer", AreaKey: "issues", Grants: explicitGrants("Bewohnerregel der Liegenschaft", MatrixActionView, MatrixActionCreate)},
	{FamilyKey: "eigentuemer", AreaKey: "documents", Grants: []MatrixGrant{{Action: MatrixActionView, Capability: CapabilityOwnerDocuments, ActorRole: store.RoleOwner, Note: "Eigentümerdokumente"}}, Note: "Beirat sieht nur für ihn freigegebene Dokumente"},
	{FamilyKey: "eigentuemer", AreaKey: "announcements", Grants: explicitGrants("veröffentlichte Inhalte der Liegenschaft", MatrixActionView)},
	{FamilyKey: "eigentuemer", AreaKey: "events", Grants: explicitGrants("veröffentlichte Inhalte der Liegenschaft", MatrixActionView)},
	{FamilyKey: "eigentuemer", AreaKey: "votes", Grants: []MatrixGrant{
		{Action: MatrixActionView, Note: "veröffentlichte Abstimmungen"},
		{Action: MatrixActionChange, Capability: CapabilityVote, ActorRole: store.RoleOwner, Note: "Stimme abgeben"},
	}, Note: "Beirat ohne Eigentümerrolle hat kein Stimmrecht"},
	{FamilyKey: "eigentuemer", AreaKey: "handovers"},
	{FamilyKey: "eigentuemer", AreaKey: "contacts", Grants: explicitGrants("freigegebene Kontakte der Liegenschaft", MatrixActionView)},
	{FamilyKey: "eigentuemer", AreaKey: "units-payments", Grants: explicitGrants("eigene Einheit und eigene Zahlungen", MatrixActionView)},
	{FamilyKey: "eigentuemer", AreaKey: "energy", Grants: []MatrixGrant{
		{Action: MatrixActionView, Capability: CapabilityManageEnergy, ActorRole: store.RoleOwner},
		{Action: MatrixActionChange, Capability: CapabilityManageEnergy, ActorRole: store.RoleOwner},
		{Action: MatrixActionApprove, Capability: CapabilityControlEnergy, ActorRole: store.RoleOwner, Note: "Haus-Schalter"},
	}, Note: "Eigentümer; Beirat nur mit eigener Liegenschafts- oder Einheitenzuordnung"},
	{FamilyKey: "eigentuemer", AreaKey: "account", Grants: explicitGrants("eigenes Konto", MatrixActionView, MatrixActionChange)},

	{FamilyKey: "bewohner", AreaKey: "portfolio"},
	{FamilyKey: "bewohner", AreaKey: "inbox"},
	{FamilyKey: "bewohner", AreaKey: "settings"},
	{FamilyKey: "bewohner", AreaKey: "snippets"},
	{FamilyKey: "bewohner", AreaKey: "overview", Grants: explicitGrants("Mandantschaft und Liegenschaftszuordnung", MatrixActionView)},
	{FamilyKey: "bewohner", AreaKey: "issues", Grants: explicitGrants("Bewohnerregel der Liegenschaft", MatrixActionView, MatrixActionCreate)},
	{FamilyKey: "bewohner", AreaKey: "documents", Grants: explicitGrants("für die Person freigegebene Dokumente", MatrixActionView)},
	{FamilyKey: "bewohner", AreaKey: "announcements", Grants: explicitGrants("veröffentlichte Inhalte der Liegenschaft", MatrixActionView)},
	{FamilyKey: "bewohner", AreaKey: "events", Grants: explicitGrants("veröffentlichte Inhalte der Liegenschaft", MatrixActionView)},
	{FamilyKey: "bewohner", AreaKey: "votes", Grants: explicitGrants("veröffentlichte Abstimmungen ohne Stimmrecht", MatrixActionView)},
	{FamilyKey: "bewohner", AreaKey: "handovers"},
	{FamilyKey: "bewohner", AreaKey: "contacts", Grants: explicitGrants("freigegebene Kontakte der Liegenschaft", MatrixActionView)},
	{FamilyKey: "bewohner", AreaKey: "units-payments", Grants: explicitGrants("eigene Einheit und eigene Zahlungen", MatrixActionView)},
	{FamilyKey: "bewohner", AreaKey: "energy", Grants: explicitGrants("eigene Einheit oder ausdrücklich erteilte Freigabe", MatrixActionView)},
	{FamilyKey: "bewohner", AreaKey: "account", Grants: explicitGrants("eigenes Konto", MatrixActionView, MatrixActionChange)},
}

var MatrixSpecialRoles = []MatrixSpecialRole{
	{Label: "Dienstleister", Area: "Anliegen", Actions: []MatrixAction{MatrixActionView, MatrixActionChange}, Scope: "Nur zugewiesene Anliegen; Status nur entlang des Dienstleister-Ablaufs."},
	{Label: "Technische Vertrauensperson", Area: "Energie", Actions: []MatrixAction{MatrixActionView, MatrixActionChange}, Scope: "Nur ausdrücklich freigegebener Zugriff auf die Liegenschaft; kein Haus-Schalter."},
}

func capabilityGrants(capability Capability, actorRole string, actions ...MatrixAction) []MatrixGrant {
	grants := make([]MatrixGrant, 0, len(actions))
	for _, action := range actions {
		grants = append(grants, MatrixGrant{Action: action, Capability: capability, ActorRole: actorRole})
	}
	return grants
}

func explicitGrants(note string, actions ...MatrixAction) []MatrixGrant {
	grants := make([]MatrixGrant, 0, len(actions))
	for _, action := range actions {
		grants = append(grants, MatrixGrant{Action: action, Note: note})
	}
	return grants
}

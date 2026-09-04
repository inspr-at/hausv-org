package store

import (
	"context"
	"time"
)

type IntakeSource string

const (
	IntakeSourcePortal IntakeSource = "portal"
	IntakeSourceEmail  IntakeSource = "email"
	IntakeSourcePhone  IntakeSource = "phone"
)

type IntakeStatus string

const (
	IntakeStatusOpen     IntakeStatus = "open"
	IntakeStatusProposed IntakeStatus = "proposed"
	IntakeStatusApproved IntakeStatus = "approved"
	IntakeStatusEdited   IntakeStatus = "edited"
	IntakeStatusRejected IntakeStatus = "rejected"
	IntakeStatusAuto     IntakeStatus = "auto"
)

type IntakeItem struct {
	ID           string            `json:"id"`
	Organisation string            `json:"organisation"`
	TenantSlug   string            `json:"tenant_slug,omitempty"`
	Unit         string            `json:"unit,omitempty"`
	Source       IntakeSource      `json:"source"`
	ExternalRef  string            `json:"external_ref,omitempty"`
	FromName     string            `json:"from_name,omitempty"`
	FromEmail    string            `json:"from_email,omitempty"`
	FromPhone    string            `json:"from_phone,omitempty"`
	Subject      string            `json:"subject"`
	Body         string            `json:"body"`
	ReceivedAt   time.Time         `json:"received_at"`
	DueAt        time.Time         `json:"due_at,omitempty"`
	Status       IntakeStatus      `json:"status"`
	Suggestion   *IntakeSuggestion `json:"suggestion,omitempty"`
	Handling     *IntakeHandling   `json:"handling,omitempty"`
	IssueID      string            `json:"issue_id,omitempty"`
	Truth        *IntakeTruth      `json:"truth,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type IntakeSuggestion struct {
	Source      string             `json:"source"`
	Provider    string             `json:"provider,omitempty"`
	Model       string             `json:"model,omitempty"`
	PromptHash  string             `json:"prompt_hash,omitempty"`
	Category    string             `json:"category,omitempty"`
	Priority    string             `json:"priority,omitempty"`
	TenantSlug  string             `json:"tenant_slug,omitempty"`
	Unit        string             `json:"unit,omitempty"`
	Assignee    string             `json:"assignee,omitempty"`
	TemplateKey string             `json:"template_key,omitempty"`
	Reply       string             `json:"reply,omitempty"`
	Unfilled    []string           `json:"unfilled,omitempty"`
	Actions     []string           `json:"actions,omitempty"`
	Confidence  map[string]float64 `json:"confidence,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
}

type IntakeHandling struct {
	Action  string    `json:"action"`
	ByEmail string    `json:"by_email,omitempty"`
	ByName  string    `json:"by_name,omitempty"`
	At      time.Time `json:"at"`
	Note    string    `json:"note,omitempty"`
}

type IntakeTruth struct {
	Category    string `json:"category,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Assignee    string `json:"assignee,omitempty"`
	TemplateKey string `json:"template_key,omitempty"`
}

type IntakeFilter struct {
	Statuses          []IntakeStatus
	Sources           []IntakeSource
	TenantSlug        string
	TenantSlugs       []string
	Unassigned        bool
	IncludeUnassigned bool
	Assignee          string
	Since             time.Time
	Sort              string
	Limit             int
	Offset            int
}

type IntakeRepository interface {
	Create(context.Context, IntakeItem) error
	Get(context.Context, string) (IntakeItem, error)
	List(context.Context, IntakeFilter) ([]IntakeItem, error)
	Count(context.Context, IntakeFilter) (int, error)
	UpdateSuggestion(context.Context, string, IntakeSuggestion, IntakeStatus) error
	UpdateHandling(context.Context, string, IntakeHandling, IntakeStatus, string) error
	Assign(context.Context, string, string, string) error
}

type IntakeCategory struct {
	Key   string
	Label string
}

const (
	IntakeCategoryRepair         = "reparatur"
	IntakeCategoryHouseRules     = "hausordnung"
	IntakeCategoryOperatingCosts = "betriebskosten"
	IntakeCategoryApproval       = "freigabe"
	IntakeCategoryKeys           = "schluessel"
	IntakeCategoryReceipt        = "beleg"
	IntakeCategoryAppointment    = "termin"
	IntakeCategoryInsurance      = "versicherung"
	IntakeCategoryHandover       = "uebergabe"
	IntakeCategoryMasterData     = "stammdaten"
	IntakeCategoryWinterGarden   = "winterdienst_garten"
	IntakeCategoryParking        = "parkplatz"
	IntakeCategoryOther          = "sonstiges"
)

func IntakeCategories() []IntakeCategory {
	return []IntakeCategory{
		{IntakeCategoryRepair, "Reparatur/Mangel"},
		{IntakeCategoryHouseRules, "Hausordnung/Nachbarschaft"},
		{IntakeCategoryOperatingCosts, "Betriebskosten/Vorschreibung"},
		{IntakeCategoryApproval, "Freigabe/Umbau"},
		{IntakeCategoryKeys, "Schlüssel/Zutritt"},
		{IntakeCategoryReceipt, "Beleg/Rechnung"},
		{IntakeCategoryAppointment, "Termin"},
		{IntakeCategoryInsurance, "Versicherung/Schaden"},
		{IntakeCategoryHandover, "Übergabe"},
		{IntakeCategoryMasterData, "Stammdaten"},
		{IntakeCategoryWinterGarden, "Winterdienst/Garten/Reinigung"},
		{IntakeCategoryParking, "Parkplatz"},
		{IntakeCategoryOther, "Sonstiges"},
	}
}

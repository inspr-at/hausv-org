// Package view holds the presentation model: the structs the templates render
// and the formatters that build them.
//
// It is pure: given domain records it returns display data. No stores, no HTTP,
// no *app — all the German labels, EUR/kWh formatting and CSS classes live here
// rather than being scattered through handlers.
package view

import (
	"fmt"
	"html/template"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/config"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/textutil"
)

// Values copied verbatim from main's mixed const blocks (the mover will not
// split a group that also holds unrelated constants).
const (
	HTMLDateTimeLocalLayout = "2006-01-02T15:04"
	DeATDateLayout          = "02.01.2006"
	DeATDateTimeLayout      = "02.01.2006 15:04"
	DeATTimeLayout          = "15:04"
	DeATShortDateTimeLayout = "02.01. 15:04"
	TenantBrandCommunity    = "community"
	TenantBrandSingleHome   = "single-home"
	TenantBrandMultiTenant  = "multi-tenant"
	TenantBrandMixedUse     = "mixed-use"
	TenantBrandAddressPlate = "address-plaque"
	TenantBrandParking      = "parking"
)

// BreakAfterSlashes adds a safe line-break opportunity without allowing
// labels to split arbitrarily in the middle of a word.
func BreakAfterSlashes(label string) string {
	return strings.ReplaceAll(label, "/", "/\u200b")
}

type NotificationEventOption struct {
	Key         string
	Label       string
	Description string
	Checked     bool
}

type AnnouncementView struct {
	ID                 string
	Title              string
	Body               string
	BodyHTML           template.HTML
	Category           string
	CategoryClass      string
	Pinned             bool
	PinnedChecked      bool
	PublishedAt        string
	PublishedAtInput   string
	ExpiresAt          string
	ExpiresAtInput     string
	HasExpiresAt       bool
	Author             string
	Status             string
	Published          bool
	Expired            bool
	Unread             bool
	CanManage          bool
	Attachments        []AttachmentView
	HasAttachments     bool
	EditDialogID       string
	DeleteConfirmLabel string
}

type HouseEventView struct {
	ID                 string
	Title              string
	Body               string
	BodyHTML           template.HTML
	HasBody            bool
	Category           string
	CategoryClass      string
	Location           string
	HasLocation        bool
	StartsAt           string
	StartsAtInput      string
	DateLabel          string
	EndsAt             string
	EndsAtInput        string
	HasEndsAt          bool
	DateBadgeDay       string
	DateBadgeMonth     string
	TimeRange          string
	Status             string
	Past               bool
	IsNext             bool
	CanManage          bool
	Author             string
	Attachments        []AttachmentView
	HasAttachments     bool
	EditDialogID       string
	DeleteConfirmLabel string
}

type IssueCommentView struct {
	ID             string
	Author         string
	Body           string
	Kind           string
	KindLabel      string
	IsQuestion     bool
	IsAnswer       bool
	CreatedAt      string
	CanDelete      bool
	DeleteURL      string
	Attachments    []AttachmentView
	HasAttachments bool
}

type AttachmentView struct {
	ID             string
	Filename       string
	Size           string
	ContentType    string
	UploadedBy     string
	CreatedAt      string
	URL            string
	PreviewURL     string
	ThumbURL       string
	IsImage        bool
	IsPDF          bool
	CanDelete      bool
	DeleteURL      string
	DeleteRedirect string
}

type AttachmentGroup struct {
	Attachments    []AttachmentView
	HasAttachments bool
}

type SelectOption struct {
	Value    string
	Label    string
	Selected bool
}

type DocumentView struct {
	Archived          bool
	ArchiveParty      string
	ArchivePartyEmail string
	ID                string
	Title             string
	Category          string
	Visibility        string
	VisibilityClass   string
	UnitLabel         string
	HasUnit           bool
	Filename          string
	FileKind          string
	Size              string
	ContentType       string
	UploadedBy        string
	UploadedAt        string
	UploadedDate      string
	DownloadURL       string
	PreviewURL        string
	CanPreview        bool
	IsImage           bool
	IsPDF             bool
	VersionLabel      string
	ReplaceDialogID   string
	Versions          []DocumentVersionView
	HasVersions       bool
}

type DocumentVersionView struct {
	ID          string
	Version     string
	Filename    string
	Size        string
	UploadedAt  string
	DownloadURL string
}

type DocumentCategoryView struct {
	Category     string
	Documents    []DocumentView
	HasDocuments bool
	EmptyMessage string
}

type BallotView struct {
	ID                  string
	Title               string
	Description         string
	HasDescription      bool
	Type                string
	Weighting           string
	Quorum              string
	HasQuorum           bool
	ReminderLabel       string
	Status              string
	StatusClass         string
	IsDraft             bool
	IsOpen              bool
	IsClosed            bool
	OpensAt             string
	HasOpensAt          bool
	ClosesAt            string
	HasClosesAt         bool
	CreatedAt           string
	UpdatedAt           string
	Options             []BallotOptionView
	CanVote             bool
	CanManage           bool
	CanOpen             bool
	CanClose            bool
	HasVote             bool
	NeedsVote           bool
	VoteOption          string
	VoteWeight          string
	VotedAt             string
	ReadOnlyMessage     string
	HasResults          bool
	TotalVotes          int
	TotalWeight         int
	TotalWeightLabel    string
	EligibleWeightLabel string
	Participation       string
	QuorumStatus        string
	QuorumClass         string
	WinnerLabel         string
	HasWinner           bool
	ProtocolURL         string
	HasProtocol         bool
	Attachments         []AttachmentView
	HasAttachments      bool
	EditDialogID        string
}

type BallotOptionView struct {
	Value        string
	Label        string
	Selected     bool
	VoteCount    int
	Weight       int
	WeightLabel  string
	Percent      int
	PercentStyle string
}

type IssueView struct {
	ID                      string
	Title                   string
	Body                    string
	Description             string
	Author                  string
	AuthorEmail             string
	Category                string
	Status                  string
	StatusClass             string
	NextStep                string
	DetailURL               string
	DetailAction            string
	ResidentState           string
	HasOpenQuestion         bool
	OpenQuestion            IssueCommentView
	ResolutionConfirmed     bool
	Priority                string
	AssigneeEmail           string
	AssigneeName            string
	HasAssignee             bool
	Location                string
	LocationType            string
	LocationDetail          string
	CreatedAt               string
	Age                     string
	CanComment              bool
	CanClose                bool
	CanReopen               bool
	CanServiceUpdate        bool
	ServiceProposal         string
	HasServiceProposal      bool
	ServiceAppointment      string
	HasServiceAppointment   bool
	ServiceStartInput       string
	ServiceEndInput         string
	CanEditEstimate         bool
	EstimateAmount          string
	EstimateAmountValue     string
	EstimateNote            string
	HasEstimate             bool
	EstimateAttachments     []AttachmentView
	HasEstimateAttachments  bool
	EstimateAttachmentGroup AttachmentGroup
	PhotoCount              int
	HasPhotos               bool
	Attachments             []AttachmentView
	HasAttachments          bool
	Comments                []IssueCommentView
	HasComments             bool
	StatusOptions           []SelectOption
	ServiceStatusOptions    []SelectOption
	PriorityOptions         []SelectOption
	// StatusLabel is the short German word shown to people (e.g. "In Arbeit").
	// Status stays the stored value ("In Bearbeitung") used in URLs, forms and
	// filter comparisons — never rename that one.
	StatusLabel string
}

type IssueBoardFilterView struct {
	Status          string
	Priority        string
	Category        string
	Assignee        string
	Sort            string
	StatusOptions   []SelectOption
	PriorityOptions []SelectOption
	CategoryOptions []SelectOption
	SortOptions     []SelectOption
	HasActive       bool
}

type ProfileUnitView struct {
	Label    string
	Relation string
	Share    string
}

type UnitPaymentStatusView struct {
	UnitID        string
	UnitLabel     string
	UnitTypeLabel string
	Relation      string
	Status        string
	StatusValue   string
	StatusClass   string
	Detail        string
	UpdatedAt     string
	UpdatedBy     string
	HasUpdatedAt  bool
	StatusOptions []SelectOption
}

type BuildingUnitView struct {
	ID                 string
	Label              string
	HomeDisplayName    string
	HasHomeDisplayName bool
	UnitType           string
	UnitTypeLabel      string
	TypeOptions        []SelectOption
	BillableLabel      string
	Share              string
	ShareValue         string
	OwnerEmails        string
	RenterEmails       string
	OwnerSummary       string
	RenterSummary      string
	HasOwners          bool
	HasRenters         bool
	MembersLabel       string
	PaymentStatus      string
	PaymentStatusClass string
	PaymentDetail      string
	PaymentUpdatedAt   string
	PaymentHasUpdated  bool
	PaymentOptions     []SelectOption
	DeleteConfirmLabel string
}

type HomeIdentityView struct {
	DisplayName    string
	UnitLabel      string
	AriaLabel      string
	HasDisplayName bool
	HasUnit        bool
}

type ContactCardView struct {
	Name        string
	Role        string
	Description string
	Email       string
	Phone       string
	HasEmail    bool
	HasPhone    bool
}

type ManagedContactView struct {
	ID                 string
	Kind               string
	KindOptions        []SelectOption
	Name               string
	Company            string
	DisplayName        string
	Description        string
	Email              string
	Phone              string
	Notes              string
	ServiceRegion      string
	Qualification      string
	EnergyCapabilities []string
	EnergySummary      string
	HasEnergyProfile   bool
	Active             bool
	CanEdit            bool
	StatusLabel        string
	HasEmail           bool
	HasPhone           bool
	EditDialogID       string
	DeleteConfirmLabel string
}

type ContactOptionView struct {
	Email string
	Label string
}

type EmptyStateView struct {
	Title       string
	Message     string
	ActionURL   string
	ActionLabel string
	HasAction   bool
}

type DashboardDigestItem struct {
	Kind        string
	Title       string
	Detail      string
	URL         string
	ActionLabel string
	Actionable  bool
	// SourceKind and SourceID name the record this item was built from
	// ("event", "issue"). The overview uses them to keep the cards below the
	// daily focus from repeating an entry that already stands at the top.
	SourceKind string
	SourceID   string
}

type AuditEventView struct {
	At             string
	AtDate         string
	AtTime         string
	AtISO          string
	DateHeader     string
	ShowDateHeader bool
	Action         string
	ActionText     string
	ActionTone     string
	ToneLabel      string
	Actor          string
	ActorRole      string
	Target         string
	TargetType     string
	HasTarget      bool
	Summary        string
	DisplayTitle   string
	Context        string
	HasContext     bool
	// ActorLabel and ObjectLabel carry the same two facts as Context, but kept
	// apart so the audit table can align them in their own columns. Context
	// stays the single-line fallback for narrow screens.
	ActorLabel  string
	ObjectLabel string
	HasObject   bool
	Details     []AuditDetailView
	HasDetails  bool
}

type AuditDetailView struct {
	Key   string
	Value string
}

type AuditStatsView struct {
	TotalEvents      int
	ActorCount       int
	TodayCount       int
	FilterSummary    string
	ActiveFilters    []AuditFilterChipView
	HasActiveFilters bool
}

type AuditFilterChipView struct {
	Label string
	Value string
}

type AnnouncementFilterView struct {
	Label  string
	URL    string
	Active bool
}

type ParkingAccountingView struct {
	Message            string
	GridFeeValue       string
	GridFeeLabel       string
	BaseFeeValue       string
	BaseFeeLabel       string
	EffectiveFrom      string
	EffectiveFromLabel string
	Tariffs            []ParkingTariffView
	HasTariffs         bool
	Months             []ParkingMonthView
	HasMonths          bool
	OutstandingValue   float64
	Outstanding        string
	HasOutstanding     bool
	OverdueValue       float64
	Overdue            string
	HasOverdue         bool
	LastSampleLabel    string
	HistoryAvailable   bool
}

type ParkingTariffView struct {
	EffectiveFrom      string
	EffectiveFromInput string
	GridFee            string
	BaseFee            string
}

type ParkingMonthView struct {
	Month            string
	MonthLabel       string
	DetailPath       string
	PeriodLabel      string
	KWhValue         float64
	EnergyCostValue  float64
	GridCostValue    float64
	BaseFeeValue     float64
	TotalCostValue   float64
	SurplusKWhValue  float64
	SurplusCostValue float64
	NormalKWhValue   float64
	KWh              string
	EnergyCost       string
	GridCost         string
	BaseFee          string
	TotalCost        string
	SurplusKWh       string
	SurplusCost      string
	NormalKWh        string
	HasSurplus       bool
	AverageAwattar   string
	EffectivePrice   string
	AveragePrice     string
	Paid             bool
	PaidLabel        string
	PaidAtInput      string
	PaidAtLabel      string
	PaidBy           string
	PaymentMethod    string
	PaymentReference string
	PaymentDetails   string
	Outstanding      bool
	Overdue          bool
	TogglePaidValue  string
	ToggleLabel      string
	ChartPercent     int
	Partial          bool
	SampleCount      int
	HourCount        int
	Attachments      []AttachmentView
	HasAttachments   bool
}

type ParkingMonthDetailView struct {
	Month           string
	MonthLabel      string
	BackPath        string
	Message         string
	GridFeeLabel    string
	LastSampleLabel string
	Summary         ParkingMonthView
	Hours           []ParkingHourView
	HasHours        bool
	Sessions        []ChargingSessionView
	HasSessions     bool
}

// ParkingLiveView is the "Jetzt" card on /app/parking: current charging
// source, battery context and the surplus/normal split for today and the
// running month.
type ParkingLiveView struct {
	Available        bool
	Enabled          bool
	ShadowMode       bool
	StaleData        bool
	Mode             string // surplus | manual | off | idle
	ModeLabel        string
	ModeClass        string
	ModeDetail       string
	RateLabel        string
	PlugOn           bool
	PowerLabel       string
	PowerKW          float64
	PowerEntity      string
	EnergyEntity     string
	PowerSourceState string
	PowerLastUpdated time.Time
	FeedInLabel      string
	BatterySOCLabel  string
	BatteryClass     string // full | partial | low
	BatteryHint      string
	SessionSince     string
	SessionKWh       string
	SessionCost      string
	HasSession       bool
	TodaySplit       ParkingSplitView
	MonthSplit       ParkingSplitView
	CanToggle        bool
	ToggleOn         bool
	AutoPaused       bool
	Sessions         []ChargingSessionView
	HasSessions      bool
	Admin            ParkingChargingAdminStrip
}

// ParkingSplitView is one two-tone surplus/normal bar.
type ParkingSplitView struct {
	Label       string
	SurplusKWh  string
	NormalKWh   string
	SurplusCost string
	NormalCost  string
	SurplusPct  int
	HasAny      bool
}

type ChargingSessionView struct {
	StartLabel    string
	DurationLabel string
	KWh           string
	ModeLabel     string
	ModeClass     string
	Cost          string
	Active        bool
}

// ParkingChargingAdminStrip is the inline diagnostics row admins see on the
// live card.
type ParkingChargingAdminStrip struct {
	Show        bool
	PhaseLabel  string
	SinceLabel  string
	LastReason  string
	PollLabel   string
	StalePill   bool
	ShadowPill  bool
	ErrorDetail string
}

// ParkingChargingAdminView backs the settings-page panels.
type ParkingChargingAdminView struct {
	Enabled          bool
	ShadowMode       bool
	StartSocValue    string
	StopSocValue     string
	StartFeedInValue string
	StopFeedInValue  string
	StopDelayValue   string
	MinOnValue       string
	MinOffValue      string
	SurplusRateValue string
	State            ParkingChargingAdminStrip
	Events           []ChargingEventView
	HasEvents        bool
	Telegram         TelegramStatusView
}

type ChargingEventView struct {
	AtLabel   string
	KindLabel string
	KindClass string
	Detail    string
	Shadow    bool
}

type TelegramStatusView struct {
	Configured  bool
	Chats       []TelegramChatView
	HasChats    bool
	PendingCode string
	CodeEmail   string
	LinkOptions []TelegramLinkOption
}

type TelegramChatView struct {
	ChatID      int64
	DisplayName string
	Email       string
	LinkedAt    string
}

type TelegramLinkOption struct {
	Email string
	Label string
}

type ParkingHourView struct {
	AtLabel             string
	AtTitle             string
	KWh                 string
	KWhTitle            string
	SurplusKWh          string
	SurplusKWhTitle     string
	HasSurplus          bool
	AverageAwattar      string
	AverageAwattarTitle string
	EnergyCost          string
	EnergyCostTitle     string
	GridCost            string
	GridCostTitle       string
	TotalCost           string
	TotalCostTitle      string
	WeightTitle         string
	ChartPercent        int
}

type BallotResultCount struct {
	Count   int
	Weight  int
	Percent int
}

type BallotResultSummary struct {
	Options             map[string]BallotResultCount
	TotalVotes          int
	TotalWeight         int
	TotalWeightLabel    string
	EligibleWeight      int
	EligibleWeightLabel string
	ParticipationPPM    int
	ParticipationLabel  string
	QuorumReached       bool
	QuorumStatus        string
	QuorumClass         string
	WinnerLabel         string
}

func BallotWinnerLabel(options map[string]BallotResultCount) string {
	if len(options) == 0 {
		return ""
	}
	maxWeight := 0
	for _, result := range options {
		if result.Weight > maxWeight {
			maxWeight = result.Weight
		}
	}
	if maxWeight <= 0 {
		return ""
	}
	winners := []string{}
	for option, result := range options {
		if result.Weight == maxWeight {
			winners = append(winners, option)
		}
	}
	sort.Strings(winners)
	return strings.Join(winners, ", ")
}

func FormatBallotWeight(weight int) string {
	if weight <= 0 {
		return ""
	}
	if weight == 1 {
		return "1 Stimme"
	}
	return FormatBallotShareWeight(weight)
}

func FormatBallotResultWeight(weighting string, weight int) string {
	if weight <= 0 {
		return "0"
	}
	if store.NormalizeBallotWeighting(weighting) == store.BallotWeightingPerHead {
		if weight == 1 {
			return "1 Stimme"
		}
		return strconv.Itoa(weight) + " Stimmen"
	}
	return FormatBallotShareWeight(weight)
}

func FormatPPMPercent(ppm int) string {
	if ppm < 0 {
		ppm = 0
	}
	if ppm > 1_000_000 {
		ppm = 1_000_000
	}
	return FormatDecimal(float64(ppm)/10_000, 1) + " %"
}

func FormatBallotShareWeight(ppm int) string {
	if ppm <= 0 {
		return "0"
	}
	percent := FormatBallotSharePercent(ppm)
	if ppm < 1000 {
		return strconv.Itoa(ppm) + " Anteile (" + percent + ")"
	}
	return percent + " Miteigentumsanteil"
}

func FormatBallotSharePercent(ppm int) string {
	if ppm < 0 {
		ppm = 0
	}
	if ppm > 1_000_000 {
		ppm = 1_000_000
	}
	decimals := 1
	if ppm > 0 && ppm < 1000 {
		decimals = 3
	} else if ppm > 0 && ppm < 10000 {
		decimals = 2
	}
	return FormatDecimal(float64(ppm)/10_000, decimals) + " %"
}

func FormatBallotReminder(minutes int) string {
	if minutes <= 0 {
		minutes = store.DefaultBallotReminderBeforeMinutes
	}
	if minutes%60 == 0 {
		hours := minutes / 60
		if hours == 1 {
			return "1 Stunde vorher"
		}
		return strconv.Itoa(hours) + " Stunden vorher"
	}
	if minutes == 1 {
		return "1 Minute vorher"
	}
	return strconv.Itoa(minutes) + " Minuten vorher"
}

func ManagedContactViewFrom(item store.ManagedContact) ManagedContactView {
	displayName := store.ManagedContactDisplayName(item)
	description := strings.TrimSpace(item.Company)
	if description == "" || strings.EqualFold(description, displayName) {
		description = strings.TrimSpace(item.Notes)
	}
	capabilityLabels := map[string]string{
		"metering": "Leistungsmessung", "smart-meter": "Smart Meter", "home-assistant": "Home Assistant",
		"pv": "PV", "battery": "Speicher", "wallbox": "Wallbox", "heat-pump": "Wärmepumpe", "electrical": "Elektro-Fachnachweis",
	}
	labels := []string{}
	for _, capability := range item.EnergyCapabilities {
		if label := capabilityLabels[capability]; label != "" {
			labels = append(labels, label)
		}
	}
	summaryParts := []string{}
	if item.ServiceRegion != "" {
		summaryParts = append(summaryParts, item.ServiceRegion)
	}
	if len(labels) > 0 {
		summaryParts = append(summaryParts, strings.Join(labels, ", "))
	}
	return ManagedContactView{
		ID:                 item.ID,
		Kind:               item.Kind,
		KindOptions:        ContactKindOptions(item.Kind),
		Name:               item.Name,
		Company:            item.Company,
		DisplayName:        displayName,
		Description:        description,
		Email:              item.Email,
		Phone:              item.Phone,
		Notes:              item.Notes,
		ServiceRegion:      item.ServiceRegion,
		Qualification:      item.Qualification,
		EnergyCapabilities: append([]string(nil), item.EnergyCapabilities...),
		EnergySummary:      strings.Join(summaryParts, " · "),
		HasEnergyProfile:   item.ServiceRegion != "" || item.Qualification != "" || len(labels) > 0,
		Active:             item.Active,
		StatusLabel:        ContactStatusLabel(item.Active),
		HasEmail:           textutil.Email(item.Email) != "",
		HasPhone:           strings.TrimSpace(item.Phone) != "",
		EditDialogID:       "contact-edit-" + item.ID,
		DeleteConfirmLabel: "Kontakt \"" + displayName + "\" deaktivieren?",
	}
}

func ContactStatusLabel(active bool) string {
	if active {
		return "Aktiv"
	}
	return "Inaktiv"
}

type ParkingStatementView struct {
	Tenant       config.TenantConfig
	User         store.UserProfile
	Year         int
	GeneratedAt  string
	GridFeeLabel string
	Months       []ParkingMonthView
	HasMonths    bool
	TotalKWh     string
	EnergyCost   string
	GridCost     string
	BaseFee      string
	TotalCost    string
	SurplusKWh   string
	SurplusCost  string
	NormalKWh    string
	HasSurplus   bool
}

func ParkingStatementTariffLabel(settings store.ParkingSettings) string {
	settings = store.NormalizeParkingSettings(settings)
	if len(settings.Tariffs) == 1 {
		tariff := settings.Tariffs[0]
		return FormatEURPerKWh(tariff.GridFeeEURPerKWh) + ", Basis " + FormatEUR(tariff.BaseFeeEUR)
	}
	return "laut Tarifhistorie"
}

func RoleClass(role string) string {
	switch store.NormalizeRole(role) {
	case store.RoleAdmin:
		return "role-admin"
	case store.RoleManager:
		return "role-manager"
	case store.RoleOwner:
		return "role-owner"
	case store.RoleRenter:
		return "role-renter"
	case store.RoleBeirat:
		return "role-beirat"
	case store.RoleResident:
		return "role-resident"
	case store.RoleServiceProvider:
		return "role-service"
	default:
		return "role-resident"
	}
}

func EventCategoryClass(raw string) string {
	switch store.NormalizeEventCategory(raw) {
	case "Eigentümerversammlung":
		return "versammlung"
	case "Reinigung":
		return "reinigung"
	case "Wartung":
		return "wartung"
	case "Ablesung":
		return "ablesung"
	case "Frist":
		return "frist"
	default:
		return "sonstiges"
	}
}

func AnnouncementViewFrom(item store.Announcement, now time.Time, includeStatus bool, lastSeen time.Time) AnnouncementView {
	published := !item.PublishedAt.After(now)
	expired := item.ExpiresAt != nil && !item.ExpiresAt.After(now)
	status := ""
	switch {
	case expired:
		status = "Abgelaufen"
	case !published:
		status = "Geplant"
	case item.Pinned:
		status = "Fixiert"
	default:
		status = "Veröffentlicht"
	}
	if !includeStatus {
		status = ""
	}
	expiresAt := ""
	expiresAtInput := ""
	if item.ExpiresAt != nil {
		expiresAt = FormatLocalDateTime(*item.ExpiresAt)
		expiresAtInput = FormatLocalDateTimeInput(*item.ExpiresAt)
	}
	author := strings.TrimSpace(item.AuthorName)
	if author == "" {
		author = item.AuthorEmail
	}
	return AnnouncementView{
		ID:                 item.ID,
		Title:              item.Title,
		Body:               item.Body,
		BodyHTML:           PlainTextHTML(item.Body),
		Category:           item.Category,
		CategoryClass:      strings.ToLower(textutil.Slug(item.Category)),
		Pinned:             item.Pinned,
		PinnedChecked:      item.Pinned,
		PublishedAt:        FormatLocalDateTime(item.PublishedAt),
		PublishedAtInput:   FormatLocalDateTimeInput(item.PublishedAt),
		ExpiresAt:          expiresAt,
		ExpiresAtInput:     expiresAtInput,
		HasExpiresAt:       item.ExpiresAt != nil,
		Author:             author,
		Status:             status,
		Published:          published,
		Expired:            expired,
		Unread:             AnnouncementUnread(item, lastSeen, now),
		EditDialogID:       "store.Announcement-edit-" + item.ID,
		DeleteConfirmLabel: "Aushang \"" + item.Title + "\" wirklich löschen?",
	}
}

func EventViewFrom(item store.HouseEvent, now time.Time) HouseEventView {
	startLocal := item.StartsAt.In(time.Local)
	past := !store.EventRollsOffAt(item).After(now)
	status := "Geplant"
	if past {
		status = "Vergangen"
	} else if SameLocalDate(startLocal, now.In(time.Local)) {
		status = "Heute"
	}
	endsAt := ""
	endsAtInput := ""
	if item.EndsAt != nil {
		endsLocal := item.EndsAt.In(time.Local)
		endsAt = FormatLocalDateTime(endsLocal)
		endsAtInput = FormatLocalDateTimeInput(endsLocal)
	}
	author := strings.TrimSpace(item.AuthorName)
	if author == "" {
		author = item.AuthorEmail
	}
	body := strings.TrimSpace(item.Body)
	location := strings.TrimSpace(item.Location)
	return HouseEventView{
		ID:                 item.ID,
		Title:              item.Title,
		Body:               item.Body,
		BodyHTML:           PlainTextHTML(item.Body),
		HasBody:            body != "",
		Category:           item.Category,
		CategoryClass:      EventCategoryClass(item.Category),
		Location:           location,
		HasLocation:        location != "",
		StartsAt:           FormatLocalDateTime(startLocal),
		StartsAtInput:      FormatLocalDateTimeInput(startLocal),
		DateLabel:          startLocal.Format("02.01.2006"),
		EndsAt:             endsAt,
		EndsAtInput:        endsAtInput,
		HasEndsAt:          item.EndsAt != nil,
		DateBadgeDay:       startLocal.Format("02"),
		DateBadgeMonth:     GermanMonthShort(startLocal),
		TimeRange:          EventTimeRange(item),
		Status:             status,
		Past:               past,
		Author:             author,
		EditDialogID:       "event-edit-" + item.ID,
		DeleteConfirmLabel: "Termin \"" + item.Title + "\" wirklich löschen?",
	}
}

func IssueSelectOptions(values []string, selected string) []SelectOption {
	options := make([]SelectOption, 0, len(values))
	for _, value := range values {
		options = append(options, SelectOption{Value: value, Label: value, Selected: value == selected})
	}
	return options
}

func IssueLocationLabel(locationType string, detail string) string {
	label := "Gemeinschaft"
	if store.NormalizeIssueLocation(locationType) == store.IssueLocationUnit {
		label = "Eigene Einheit"
	}
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return label
	}
	return label + " · " + detail
}

// TruncateIssueDescription truncates body text for the Beschreibung column,
// keeping approximately 100 chars and breaking on word boundaries.
func TruncateIssueDescription(body string) string {
	clean := strings.Join(strings.Fields(body), " ")
	runes := []rune(clean)
	if len(runes) <= 100 {
		return clean
	}

	// Find a word boundary near position 100
	cutPoint := 100
	for i := cutPoint; i > 60 && i < len(runes); i-- {
		if runes[i] == ' ' {
			cutPoint = i
			break
		}
	}

	return strings.TrimSpace(string(runes[:cutPoint])) + "…"
}

func IssueStatusClass(status string) string {
	switch store.NormalizeIssueStatus(status) {
	case store.IssueStatusProgress, store.IssueStatusAccepted, store.IssueStatusScheduled:
		return "status-progress"
	case store.IssueStatusDone:
		return "status-done"
	case store.IssueStatusRejected, store.IssueStatusDuplicate:
		return "status-closed"
	default:
		return "status-open"
	}
}

// IssueStatusLabel renders the short German word shown to people. It is the
// display counterpart to store.IssueStatus*: the stored values keep their
// verbatim text (URLs, form values, DB rows), this only shortens what a
// person reads. store.IssueStatusProgress ("In Bearbeitung") shows as "In
// Arbeit" — one line in a fixed-width status column; every other status
// already reads as a short, conventional German word and passes through
// unchanged.
func IssueStatusLabel(status string) string {
	normalized := store.NormalizeIssueStatus(status)
	if normalized == store.IssueStatusProgress {
		return "In Arbeit"
	}
	return normalized
}

func FormatIssueEstimateAmount(cents int64) string {
	if cents <= 0 {
		return ""
	}
	return FormatEUR(float64(cents) / 100)
}

func FormatIssueEstimateInput(cents int64) string {
	if cents <= 0 {
		return ""
	}
	return FormatDecimal(float64(cents)/100, 2)
}

func IssueBoardFilterOptions(filters IssueBoardFilterView) IssueBoardFilterView {
	filters.StatusOptions = IssueStatusFilterOptions(filters.Status, "Alle Status")
	filters.PriorityOptions = IssueFilterOptions(IssuePriorities(), filters.Priority, "Alle Prioritäten")
	filters.CategoryOptions = IssueFilterOptions(IssueCategories(), filters.Category, "Alle Kategorien")
	filters.SortOptions = []SelectOption{
		{Value: "updated", Label: "Zuletzt aktualisiert", Selected: filters.Sort == "updated"},
		{Value: "age", Label: "Älteste zuerst", Selected: filters.Sort == "age"},
		{Value: "priority", Label: "Priorität", Selected: filters.Sort == "priority"},
		{Value: "status", Label: "Status", Selected: filters.Sort == "status"},
		{Value: "category", Label: "Kategorie", Selected: filters.Sort == "category"},
		{Value: "assignee", Label: "Zuständigkeit", Selected: filters.Sort == "assignee"},
	}
	return filters
}

func IssueFilterOptions(values []string, selected string, allLabel string) []SelectOption {
	options := []SelectOption{{Value: "", Label: allLabel, Selected: selected == ""}}
	for _, value := range values {
		options = append(options, SelectOption{Value: value, Label: value, Selected: value == selected})
	}
	return options
}

// IssueStatusFilterOptions is IssueFilterOptions for issue statuses: the
// option Value stays the stored value (it becomes the "status" query param),
// but the Label people read is the short display word from IssueStatusLabel.
func IssueStatusFilterOptions(selected string, allLabel string) []SelectOption {
	options := []SelectOption{{Value: "", Label: allLabel, Selected: selected == ""}}
	for _, value := range IssueStatuses() {
		options = append(options, SelectOption{Value: value, Label: IssueStatusLabel(value), Selected: value == selected})
	}
	return options
}

// IssueStatusSelectOptions is IssueSelectOptions for issue statuses: same
// Value/Selected contract, but the Label is the short display word rather
// than the stored value itself.
func IssueStatusSelectOptions(values []string, selected string) []SelectOption {
	options := make([]SelectOption, 0, len(values))
	for _, value := range values {
		options = append(options, SelectOption{Value: value, Label: IssueStatusLabel(value), Selected: value == selected})
	}
	return options
}

func AttachmentViewFromRecord(item store.AttachmentRecord, canDelete bool) AttachmentView {
	escapedID := url.PathEscape(item.ID)
	contentType := strings.ToLower(strings.TrimSpace(item.ContentType))
	return AttachmentView{
		ID:          item.ID,
		Filename:    item.Filename,
		Size:        FormatBytes(item.Size),
		ContentType: contentType,
		UploadedBy:  item.UploadedBy,
		CreatedAt:   FormatLocalDateTime(item.CreatedAt),
		URL:         "/app/attachments/" + escapedID,
		PreviewURL:  "/app/attachments/" + escapedID + "/preview",
		ThumbURL:    "/app/attachments/" + escapedID + "/thumb",
		IsImage:     store.IsImageContentType(contentType),
		IsPDF:       strings.Split(contentType, ";")[0] == "application/pdf",
		CanDelete:   canDelete,
		DeleteURL:   "/app/attachments/delete",
	}
}

func AuditEventViewFrom(event store.AuditEvent) AuditEventView {
	details := make([]AuditDetailView, 0, len(event.Details))
	keys := make([]string, 0, len(event.Details))
	for key := range event.Details {
		if key == "target_label" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		details = append(details, AuditDetailView{Key: AuditDetailLabel(key), Value: event.Details[key]})
	}
	targetID := event.TargetID
	if label := strings.TrimSpace(event.Details["target_label"]); label != "" {
		targetID = label
	}
	target := AuditTargetLabel(event.TargetType, targetID)
	actionText := AuditActionLabel(event.Action)
	displayTitle := strings.TrimSpace(event.Summary)
	if displayTitle == "" || strings.EqualFold(displayTitle, actionText) {
		displayTitle = actionText
	}
	contextParts := make([]string, 0, 2)
	actor := strings.TrimSpace(event.ActorEmail)
	if role := strings.TrimSpace(event.ActorRole); actor != "" && role != "" {
		actor += " · " + role
	}
	if actor != "" {
		contextParts = append(contextParts, actor)
	}
	object := ""
	if target != "" && AuditTargetTypeLabel(event.TargetType) != "Sitzung" {
		object = target
		contextParts = append(contextParts, target)
	}
	context := strings.Join(contextParts, " · ")
	return AuditEventView{
		At:           FormatLocalDateTime(event.At),
		AtDate:       FormatLocalDate(event.At),
		AtTime:       FormatLocalTime(event.At),
		AtISO:        event.At.Format(time.RFC3339),
		Action:       event.Action,
		ActionText:   actionText,
		ActionTone:   AuditActionTone(event.Action),
		ToneLabel:    AuditToneLabel(event.Action),
		Actor:        event.ActorEmail,
		ActorRole:    event.ActorRole,
		Target:       target,
		TargetType:   AuditTargetTypeLabel(event.TargetType),
		HasTarget:    target != "",
		Summary:      event.Summary,
		DisplayTitle: displayTitle,
		Context:      context,
		HasContext:   context != "",
		ActorLabel:   actor,
		ObjectLabel:  object,
		HasObject:    object != "",
		Details:      details,
		HasDetails:   len(details) > 0,
	}
}

func AuditActionOptions(selected string) []SelectOption {
	options := []SelectOption{{Value: "", Label: "Alle Aktionen", Selected: selected == ""}}
	for _, action := range []string{
		store.AuditActionSupportViewStart, store.AuditActionSupportViewEnd,
		store.AuditActionLogin,
		store.AuditActionInviteCreate,
		store.AuditActionInviteUpdate,
		store.AuditActionInviteDelete,
		store.AuditActionBuildingUpdate,
		store.AuditActionPortalModulesUpdate,
		store.AuditActionHeroUpdate,
		store.AuditActionUnitSave,
		store.AuditActionUnitDelete,
		store.AuditActionUnitPayment,
		store.AuditActionDocumentUpload,
		store.AuditActionDocumentDownload,
		store.AuditActionDocumentReplace,
		store.AuditActionAttachmentView,
		store.AuditActionAttachmentDelete,
		store.AuditActionIntegrationImport,
		store.AuditActionHandoverCreate,
		store.AuditActionHandoverConfirm,
		store.AuditActionHandoverFile,
		store.AuditActionVoteCreate,
		store.AuditActionVoteOpen,
		store.AuditActionVoteClose,
		store.AuditActionVoteCast,
		store.AuditActionVoteReminder,
		store.AuditActionParkingSettings,
		store.AuditActionParkingMonth,
		store.AuditActionParkingReminder,
		store.AuditActionIssueWorkflow,
		store.AuditActionIssueEstimate,
		store.AuditActionIssueServiceAdd,
		store.AuditActionIssueServiceDrop,
		store.AuditActionIssueComment,
		store.AuditActionIssueCommentDelete,
		store.AuditActionEventCreate,
		store.AuditActionEventUpdate,
		store.AuditActionEventDelete,
		store.AuditActionContactSave,
		store.AuditActionContactDelete,
		store.AuditActionIssueAISuggest,
		store.AuditActionIssueAIAccept,
		store.AuditActionIssueAIEdit,
		store.AuditActionIssueAIReject,
		store.AuditActionIssueAIAuto,
		store.AuditActionIssueAIRestore,
		store.AuditActionIntakePhoneNote,
		store.AuditActionIntakeAssign,
		store.AuditActionVerwaltungSettings,
		store.AuditActionRolePreviewStart,
		store.AuditActionRolePreviewEnd,
	} {
		options = append(options, SelectOption{Value: action, Label: AuditActionLabel(action), Selected: selected == action})
	}
	return options
}

func AuditActionLabel(action string) string {
	switch store.NormalizeAuditAction(action) {
	case store.AuditActionCapabilityOverride:
		return "Rollenrecht geändert"
	case store.AuditActionCapabilityReset:
		return "Rollenrechte zurückgesetzt"
	case store.AuditActionUserCapability:
		return "Eigenes Benutzerrecht geändert"
	case store.AuditActionCapabilityProfile:
		return "Berechtigungsprofil gespeichert"
	case store.AuditActionSupportViewStart:
		return "Supportansicht gestartet"
	case store.AuditActionSupportViewEnd:
		return "Supportansicht beendet"
	case store.AuditActionLogin:
		return "Anmeldung"
	case store.AuditActionContextSwitch:
		return "Portal gewechselt"
	case store.AuditActionInviteCreate:
		return "Einladung angelegt"
	case store.AuditActionInviteUpdate:
		return "Einladung geändert"
	case store.AuditActionInviteDelete:
		return "Einladung gelöscht"
	case store.AuditActionBuildingUpdate:
		return "Gebäude geändert"
	case store.AuditActionPortalModulesUpdate:
		return "Portalbereiche geändert"
	case store.AuditActionHeroUpdate:
		return "Hero-Bild geändert"
	case store.AuditActionProfilePicture:
		return "Profilbild geändert"
	case store.AuditActionUnitSave:
		return "Einheit gespeichert"
	case store.AuditActionUnitDelete:
		return "Einheit gelöscht"
	case store.AuditActionUnitPayment:
		return "Zahlungsstatus geändert"
	case store.AuditActionDocumentUpload:
		return "Dokument hochgeladen"
	case store.AuditActionDocumentDownload:
		return "Dokument heruntergeladen"
	case store.AuditActionDocumentReplace:
		return "Dokument ersetzt"
	case store.AuditActionAttachmentView:
		return "Anhang angesehen"
	case store.AuditActionAttachmentDelete:
		return "Anhang entfernt"
	case store.AuditActionIntegrationImport:
		return "Integration importiert"
	case store.AuditActionIntegrationExport:
		return "Rohdaten exportiert"
	case store.AuditActionHandoverCreate:
		return "Übergabe angelegt"
	case store.AuditActionHandoverConfirm:
		return "Übergabe bestätigt"
	case store.AuditActionHandoverFile:
		return "Übergabe abgelegt"
	case store.AuditActionVoteCreate:
		return "Abstimmung angelegt"
	case store.AuditActionVoteOpen:
		return "Abstimmung geöffnet"
	case store.AuditActionVoteClose:
		return "Abstimmung geschlossen"
	case store.AuditActionVoteCast:
		return "Stimme gespeichert"
	case store.AuditActionVoteReminder:
		return "Abstimmungs-Erinnerung gesendet"
	case store.AuditActionParkingSettings:
		return "Parkplatz-Abrechnung geändert"
	case store.AuditActionParkingMonth:
		return "Monatsstatus geändert"
	case store.AuditActionParkingReminder:
		return "Zahlungserinnerung gesendet"
	case store.AuditActionChargingSettings:
		return "Ladeeinstellungen geändert"
	case store.AuditActionChargingManual:
		return "Ladevorgang manuell erfasst"
	case store.AuditActionChargingSession:
		return "Ladevorgang gespeichert"
	case store.AuditActionIssueWorkflow:
		return "Anliegen bearbeitet"
	case store.AuditActionIssueEstimate:
		return "Kostenvoranschlag aktualisiert"
	case store.AuditActionIssueServiceAdd:
		return "Dienstleister eingeladen"
	case store.AuditActionIssueServiceDrop:
		return "Dienstleister-Zugriff entzogen"
	case store.AuditActionIssueComment:
		return "Kommentar hinzugefügt"
	case store.AuditActionIssueCommentDelete:
		return "Kommentar gelöscht"
	case store.AuditActionEventCreate:
		return "Termin angelegt"
	case store.AuditActionEventUpdate:
		return "Termin geändert"
	case store.AuditActionEventDelete:
		return "Termin gelöscht"
	case store.AuditActionContactSave:
		return "Kontakt gespeichert"
	case store.AuditActionContactDelete:
		return "Kontakt deaktiviert"
	case store.AuditActionEnergyOnboarding:
		return "Energie-Einrichtung geändert"
	case store.AuditActionEnergyMode:
		return "Energiemodus geändert"
	case store.AuditActionEnergyImport:
		return "Smart-Meter-Daten importiert"
	case store.AuditActionEnergyTarget:
		return "Energieziel geändert"
	case store.AuditActionEnergyRecommend:
		return "Energieempfehlung aktualisiert"
	case store.AuditActionEnergyMeasureAdd:
		return "Energiemaßnahme angelegt"
	case store.AuditActionEnergyMeasureEdit:
		return "Energiemaßnahme geändert"
	case store.AuditActionEnergyCaretaker:
		return "Energiezugriff der Hausbetreuung geändert"
	case store.AuditActionEnergyInvite:
		return "Hausbetreuung zu Energie eingeladen"
	case store.AuditActionEnergyMaintSave:
		return "Wartungsplan gespeichert"
	case store.AuditActionEnergyMaintDone:
		return "Wartung abgeschlossen"
	case store.AuditActionEnergyTariff:
		return "Energietarif bewertet"
	case store.AuditActionEnergyExport:
		return "Energiedaten exportiert"
	case store.AuditActionEnergyIdentity:
		return "Zuhause-Darstellung geändert"
	case store.AuditActionEnergyHistoryDelete:
		return "Energie-Messverlauf gelöscht"
	case store.AuditActionEnergyProfileDelete:
		return "Energieprofil gelöscht"
	case store.AuditActionAnnualPeriodSave:
		return "Abrechnungsperiode gespeichert"
	case store.AuditActionAnnualPartiesImport:
		return "Parteien aus Tabelle übernommen"
	case store.AuditActionAnnualCostTypeSave:
		return "Kostenart gespeichert"
	case store.AuditActionAnnualBasesSave:
		return "Verteilerbasis je Einheit gespeichert"
	case store.AuditActionAnnualReceiptCreate:
		return "Beleg erfasst"
	case store.AuditActionAnnualReceiptAmount:
		return "Belegbetrag korrigiert"
	case store.AuditActionAnnualReceiptDelete:
		return "Belegzuordnung entfernt"
	case store.AuditActionAnnualRunCreate:
		return "Abrechnungslauf berechnet"
	case store.AuditActionAnnualRunSend:
		return "Jahresabrechnung per E-Mail versendet"
	case store.AuditActionAnnualRunArchive:
		return "Jahresabrechnung im Archiv abgelegt"
	case store.AuditActionAnnualPrepaymentSave:
		return "Vorauszahlung gespeichert"
	case store.AuditActionIssueAISuggest:
		return "KI-Vorschlag erstellt"
	case store.AuditActionIssueAIAccept:
		return "KI-Vorschlag freigegeben"
	case store.AuditActionIssueAIEdit:
		return "KI-Vorschlag geändert und freigegeben"
	case store.AuditActionIssueAIReject:
		return "KI-Vorschlag verworfen"
	case store.AuditActionIssueAIAuto:
		return "Automatisch erledigt"
	case store.AuditActionIssueAIRestore:
		return "Zurück in den Eingang"
	case store.AuditActionIntakePhoneNote:
		return "Telefonnotiz erfasst"
	case store.AuditActionIntakeAssign:
		return "Eingang zugeordnet"
	case store.AuditActionVerwaltungSettings:
		return "Verwaltungseinstellungen geändert"
	case store.AuditActionDemoReset:
		return "Demodaten initialisiert"
	case store.AuditActionTextbausteinChanged:
		return "Textbaustein geändert"
	case store.AuditActionRolePreviewStart:
		return "Ansicht als Rolle gestartet"
	case store.AuditActionRolePreviewEnd:
		return "Ansicht als Rolle beendet"
	default:
		return "Aktivität"
	}
}

func AuditActionTone(action string) string {
	switch store.NormalizeAuditAction(action) {
	case store.AuditActionInviteCreate, store.AuditActionUnitSave, store.AuditActionDocumentUpload, store.AuditActionHandoverCreate, store.AuditActionHandoverConfirm, store.AuditActionVoteCreate, store.AuditActionVoteOpen, store.AuditActionVoteCast, store.AuditActionVoteReminder, store.AuditActionParkingReminder, store.AuditActionIssueServiceAdd, store.AuditActionEventCreate, store.AuditActionContactSave, store.AuditActionLogin:
		return "add"
	case store.AuditActionInviteDelete, store.AuditActionUnitDelete, store.AuditActionDocumentReplace, store.AuditActionAttachmentDelete, store.AuditActionVoteClose, store.AuditActionIssueServiceDrop, store.AuditActionEventDelete, store.AuditActionContactDelete:
		return "danger"
	default:
		return "change"
	}
}

func AuditToneLabel(action string) string {
	switch AuditActionTone(action) {
	case "add":
		return "Hinzugefügt"
	case "danger":
		return "Kritisch"
	default:
		return "Geändert"
	}
}

func AuditTargetLabel(targetType string, targetID string) string {
	targetType = AuditTargetTypeLabel(targetType)
	targetID = strings.TrimSpace(targetID)
	if targetType == "" {
		return targetID
	}
	if targetID == "" {
		return targetType
	}
	return targetType + ": " + targetID
}

func AuditTargetTypeLabel(targetType string) string {
	switch strings.TrimSpace(targetType) {
	case "user":
		return "Person"
	case "session":
		return "Sitzung"
	case "building":
		return "Gebäude"
	case "hero":
		return "Hero-Bild"
	case "profile_picture":
		return "Profilbild"
	case "store.Unit", "unit":
		return "Einheit"
	case "parking":
		return "Parkplatz"
	case "issue":
		return "Anliegen"
	case "document":
		return "Dokument"
	case "store.Ballot", "ballot":
		return "Abstimmung"
	case "attachment":
		return "Anhang"
	case "integration":
		return "Integration"
	case "handover":
		return "Übergabe"
	case "event":
		return "Termin"
	default:
		return strings.TrimSpace(targetType)
	}
}

func AuditDetailLabel(key string) string {
	switch key {
	case "family":
		return "Rollenfamilie"
	case "area":
		return "Bereich"
	case "capability":
		return "Berechtigung"
	case "before":
		return "Vorher"
	case "after":
		return "Nachher"
	case "auth_method":
		return "Anmeldung"
	case "role_from":
		return "Rolle vorher"
	case "role_to":
		return "Rolle neu"
	case "permissions_from":
		return "Rechte vorher"
	case "permissions_to":
		return "Rechte neu"
	case "mail_status":
		return "E-Mail"
	case "changed_fields":
		return "Geänderte Felder"
	case "grid_fee":
		return "Netzgebühr"
	case "base_fee":
		return "Basisgebühr"
	case "effective_from":
		return "Gültig ab"
	case "month":
		return "Monat"
	case "paid":
		return "Status"
	case "paid_at":
		return "Bezahlt am"
	case "paid_by":
		return "Erfasst von"
	case "payment_method":
		return "Zahlungsart"
	case "payment_reference":
		return "Referenz"
	case "estimate_amount":
		return "Kostenschätzung"
	case "file_count":
		return "Dateien"
	case "has_file":
		return "Datei"
	case "balance":
		return "Offener Betrag"
	case "status":
		return "Status"
	case "priority":
		return "Priorität"
	case "unit_label":
		return "Einheit"
	case "share":
		return "Anteil"
	case "title":
		return "Titel"
	case "category":
		return "Kategorie"
	case "visibility":
		return "Sichtbarkeit"
	case "size":
		return "Größe"
	case "content_type":
		return "Dateityp"
	case "store.Unit":
		return "Einheit"
	case "version":
		return "Version"
	case "previous":
		return "Vorherige Version"
	case "previous_id":
		return "Vorherige ID"
	case "type":
		return "Typ"
	case "weighting":
		return "Gewichtung"
	case "quorum":
		return "Quorum"
	case "reminder":
		return "Erinnerung"
	case "weight":
		return "Stimmgewicht"
	case "cast_at":
		return "Stimmabgabe"
	case "recipients":
		return "Empfänger"
	case "deadline":
		return "Frist"
	case "document_id":
		return "Dokument"
	case "entity_type":
		return "Bereich"
	case "entity_id":
		return "Vorgang"
	case "target_id":
		return "Technische ID"
	case "access":
		return "Zugriff"
	case "source":
		return "Quelle"
	case "format":
		return "Format"
	case "assigned":
		return "Zugeordnet"
	case "unclear":
		return "Unklar"
	case "rejected":
		return "Abgelehnt"
	default:
		return strings.ReplaceAll(key, "_", " ")
	}
}

func UnitPaymentStatusViewFromUnit(item store.Unit, relation string, record store.UnitPaymentStatus, hasRecord bool) UnitPaymentStatusView {
	status := store.UnitPaymentStatusOpen
	if hasRecord {
		status = record.Status
	}
	view := UnitPaymentStatusView{
		UnitID:        item.ID,
		UnitLabel:     item.Label,
		UnitTypeLabel: UnitTypeLabel(item.UnitType),
		Relation:      UnitPaymentRelationLabel(relation),
		Status:        UnitPaymentStatusLabel(status),
		StatusValue:   store.NormalizeUnitPaymentStatus(status),
		StatusClass:   UnitPaymentStatusClass(status),
		Detail:        UnitPaymentStatusDetail(status, hasRecord),
		StatusOptions: UnitPaymentStatusOptions(status),
	}
	if hasRecord && !record.UpdatedAt.IsZero() {
		view.UpdatedAt = FormatDateTimeIn(record.UpdatedAt, time.Local, DeATShortDateTimeLayout)
		view.HasUpdatedAt = true
	}
	if hasRecord {
		view.UpdatedBy = record.UpdatedBy
	}
	return view
}

func UnitPaymentRelationLabel(relation string) string {
	switch store.NormalizeRole(relation) {
	case store.RoleOwner:
		return "Eigentümer"
	case store.RoleRenter:
		return "Mieter"
	default:
		return strings.TrimSpace(relation)
	}
}

func UnitTypeOptions(selected string) []SelectOption {
	selected = store.NormalizeUnitType(selected)
	options := []SelectOption{
		{Value: store.UnitTypeResidential, Label: "Wohnung"},
		{Value: store.UnitTypeCommercial, Label: "Geschäftslokal"},
		{Value: store.UnitTypeParking, Label: "Stellplatz"},
		{Value: store.UnitTypeStorage, Label: "Keller / Lager"},
		{Value: store.UnitTypeOther, Label: "Sonstiges"},
	}
	for i := range options {
		options[i].Selected = options[i].Value == selected
	}
	return options
}

func UnitTypeLabel(unitType string) string {
	switch store.NormalizeUnitType(unitType) {
	case store.UnitTypeResidential:
		return "Wohnung"
	case store.UnitTypeCommercial:
		return "Geschäftslokal"
	case store.UnitTypeParking:
		return "Stellplatz"
	case store.UnitTypeStorage:
		return "Keller / Lager"
	case store.UnitTypeOther:
		return "Sonstiges"
	default:
		return "Einheit"
	}
}

func UnitBillableLabel(weight int) string {
	if weight <= 0 {
		return "zählt nicht als WE"
	}
	return "zählt als " + FormatBillableUnitWeight(weight) + " WE"
}

func UnitPaymentStatusLabel(status string) string {
	switch store.NormalizeUnitPaymentStatus(status) {
	case store.UnitPaymentStatusPaid:
		return "Bezahlt"
	case store.UnitPaymentStatusPartial:
		return "Teilbezahlt"
	case store.UnitPaymentStatusOverdue:
		return "Überfällig"
	default:
		return "Offen"
	}
}

func UnitPaymentStatusClass(status string) string {
	switch store.NormalizeUnitPaymentStatus(status) {
	case store.UnitPaymentStatusPaid:
		return "ok"
	case store.UnitPaymentStatusPartial:
		return "info"
	case store.UnitPaymentStatusOverdue:
		return "dringend"
	default:
		return ""
	}
}

func UnitPaymentStatusOptions(selected string) []SelectOption {
	selected = store.NormalizeUnitPaymentStatus(selected)
	options := []SelectOption{
		{Value: store.UnitPaymentStatusOpen, Label: UnitPaymentStatusLabel(store.UnitPaymentStatusOpen)},
		{Value: store.UnitPaymentStatusPaid, Label: UnitPaymentStatusLabel(store.UnitPaymentStatusPaid)},
		{Value: store.UnitPaymentStatusPartial, Label: UnitPaymentStatusLabel(store.UnitPaymentStatusPartial)},
		{Value: store.UnitPaymentStatusOverdue, Label: UnitPaymentStatusLabel(store.UnitPaymentStatusOverdue)},
	}
	for i := range options {
		options[i].Selected = options[i].Value == selected
	}
	return options
}

func FormatMiteigentumsanteil(ppm int) string {
	if ppm <= 0 {
		return "ohne Anteil"
	}
	return FormatDecimal(float64(ppm), 0) + " / 1.000.000"
}

func NotificationEventOptions(prefs store.NotificationPreferences) []NotificationEventOption {
	prefs = store.MergeNotificationPreferences(prefs)
	out := NotificationEventCatalog()
	for i := range out {
		out[i].Checked = prefs.Email[out[i].Key]
	}
	return out
}

func EmptyState(title string, message string) EmptyStateView {
	return EmptyStateView{Title: title, Message: message}
}

func NormalizeTenantBrandIcon(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", TenantBrandCommunity, "weg":
		return TenantBrandCommunity
	case TenantBrandSingleHome, "single", "home", "house":
		return TenantBrandSingleHome
	case TenantBrandMultiTenant, "multi", "building", "apartment":
		return TenantBrandMultiTenant
	case TenantBrandMixedUse, "mixed", "business":
		return TenantBrandMixedUse
	case TenantBrandAddressPlate, "address", "plaque", "plate":
		return TenantBrandAddressPlate
	case TenantBrandParking, "garage":
		return TenantBrandParking
	default:
		return ""
	}
}

func TenantBrandIconLabel(icon string) string {
	switch NormalizeTenantBrandIcon(icon) {
	case TenantBrandSingleHome:
		return "Einfamilienhaus"
	case TenantBrandMultiTenant:
		return "Mehrparteienhaus"
	case TenantBrandMixedUse:
		return "Gemischt genutzt"
	case TenantBrandAddressPlate:
		return "Adressschild"
	case TenantBrandParking:
		return "Parkplatz / Ladeplatz"
	default:
		return "Hausgemeinschaft"
	}
}

func TenantBrandIconOptions(selected string) []SelectOption {
	selected = NormalizeTenantBrandIcon(selected)
	options := []SelectOption{
		{Value: TenantBrandCommunity, Label: TenantBrandIconLabel(TenantBrandCommunity)},
		{Value: TenantBrandMultiTenant, Label: TenantBrandIconLabel(TenantBrandMultiTenant)},
		{Value: TenantBrandSingleHome, Label: TenantBrandIconLabel(TenantBrandSingleHome)},
		{Value: TenantBrandMixedUse, Label: TenantBrandIconLabel(TenantBrandMixedUse)},
		{Value: TenantBrandAddressPlate, Label: TenantBrandIconLabel(TenantBrandAddressPlate)},
		{Value: TenantBrandParking, Label: TenantBrandIconLabel(TenantBrandParking)},
	}
	for i := range options {
		options[i].Selected = options[i].Value == selected
	}
	return options
}

func ContactKindOptions(selected string) []SelectOption {
	selected = store.NormalizeContactKind(selected)
	kinds := []string{"Hausmeister", "Notdienst", "Verwaltung", "Dienstleister", "Energie-Fachbetrieb", "Sonstiges"}
	options := make([]SelectOption, 0, len(kinds))
	for _, kind := range kinds {
		options = append(options, SelectOption{Value: kind, Label: kind, Selected: selected == kind})
	}
	return options
}

func DocumentCategoryOptions(selected string) []SelectOption {
	selected = strings.TrimSpace(selected)
	if selected != "" {
		selected = store.NormalizeDocumentCategory(selected)
	}
	options := make([]SelectOption, 0, len(DocumentCategories())+1)
	options = append(options, SelectOption{Value: "", Label: "Kategorie wählen", Selected: selected == ""})
	for _, category := range DocumentCategories() {
		options = append(options, SelectOption{Value: category, Label: category, Selected: selected == category})
	}
	return options
}

func DocumentUnitOptions(units []store.Unit, selected string) []SelectOption {
	selected = store.NormalizeUnitID(selected)
	options := []SelectOption{{Value: "", Label: "Gesamte Liegenschaft", Selected: selected == ""}}
	for _, item := range units {
		id := store.NormalizeUnitID(item.ID)
		label := strings.TrimSpace(item.Label)
		if id == "" || label == "" {
			continue
		}
		options = append(options, SelectOption{Value: id, Label: label, Selected: selected == id})
	}
	return options
}

func DocumentUnitLabel(unitID string) string {
	unitID = store.NormalizeUnitID(unitID)
	if unitID == "" {
		return ""
	}
	return unitID
}

func DocumentUnitAuditLabel(unitID string) string {
	unitID = store.NormalizeUnitID(unitID)
	if unitID == "" {
		return ""
	}
	return unitID
}

func DocumentVisibilityOptions(selected string) []SelectOption {
	selected = store.NormalizeDocumentVisibility(selected)
	values := []string{store.DocumentVisibilityAllResidents, store.DocumentVisibilityOwnersOnly, store.DocumentVisibilityBoardOnly, store.DocumentVisibilityManagerOnly}
	options := make([]SelectOption, 0, len(values))
	for _, value := range values {
		options = append(options, SelectOption{Value: value, Label: DocumentVisibilityLabel(value), Selected: selected == value})
	}
	return options
}

func DocumentVisibilityLabel(visibility string) string {
	switch store.NormalizeDocumentVisibility(visibility) {
	case store.DocumentVisibilityAllResidents:
		return "Alle Bewohner"
	case store.DocumentVisibilityOwnersOnly:
		return "Nur Eigentümer"
	case store.DocumentVisibilityBoardOnly:
		return "Nur Beirat und Verwaltung"
	case store.DocumentVisibilityManagerOnly:
		return "Nur Verwaltung"
	default:
		return ""
	}
}

func DocumentVisibilityClass(visibility string) string {
	switch store.NormalizeDocumentVisibility(visibility) {
	case store.DocumentVisibilityAllResidents:
		return "ok"
	case store.DocumentVisibilityOwnersOnly, store.DocumentVisibilityBoardOnly:
		return "unread"
	case store.DocumentVisibilityManagerOnly:
		return "role-admin"
	default:
		return ""
	}
}

func DocumentViewFrom(item store.DocumentRecord) DocumentView {
	contentType := strings.ToLower(strings.TrimSpace(item.ContentType))
	canPreview := DocumentCanPreview(contentType)
	archiveParty, archiveEmail := "", ""
	if item.AnnualStatementArchive != nil {
		archiveEmail = item.AnnualStatementArchive.PartyID
		if archiveEmail != "" {
			archiveParty = archiveEmail
		}
	}
	return DocumentView{
		Archived:          item.AnnualStatementArchive != nil,
		ArchiveParty:      archiveParty,
		ArchivePartyEmail: archiveEmail,
		ID:                item.ID,
		Title:             item.Title,
		Category:          item.Category,
		Visibility:        DocumentVisibilityLabel(item.Visibility),
		VisibilityClass:   DocumentVisibilityClass(item.Visibility),
		UnitLabel:         DocumentUnitLabel(item.UnitID),
		HasUnit:           store.NormalizeUnitID(item.UnitID) != "",
		Filename:          item.Filename,
		FileKind:          DocumentFileKind(item),
		Size:              FormatBytes(item.Size),
		ContentType:       item.ContentType,
		UploadedBy:        item.UploadedBy,
		UploadedAt:        FormatLocalDateTime(item.UploadedAt),
		UploadedDate:      FormatLocalDate(item.UploadedAt),
		DownloadURL:       "/app/dokumente/" + url.PathEscape(item.ID) + "/download",
		PreviewURL:        "/app/dokumente/" + url.PathEscape(item.ID) + "/preview",
		CanPreview:        canPreview,
		IsImage:           store.IsImageContentType(contentType),
		IsPDF:             strings.Split(contentType, ";")[0] == "application/pdf",
		VersionLabel:      DocumentVersionLabel(item.Version),
		ReplaceDialogID:   "document-replace-" + item.ID,
	}
}

func DocumentVersionLabel(version int) string {
	if version <= 0 {
		version = 1
	}
	return "Version " + strconv.Itoa(version)
}

func DocumentSortOptions(selected string) []SelectOption {
	selected = SelectedDocumentSort(selected)
	options := []SelectOption{
		{Value: "newest", Label: "Neueste", Selected: selected == "newest"},
		{Value: "oldest", Label: "Älteste", Selected: selected == "oldest"},
		{Value: "title", Label: "Titel A–Z", Selected: selected == "title"},
	}
	return options
}

func FormatBytes(size int64) string {
	if size < 0 {
		size = 0
	}
	const kb = 1024
	const mb = 1024 * kb
	switch {
	case size >= mb:
		return FormatDecimal(float64(size)/float64(mb), 1) + " MB"
	case size >= kb:
		return FormatDecimal(float64(size)/float64(kb), 1) + " KB"
	default:
		return strconv.FormatInt(size, 10) + " B"
	}
}

func BallotWeightingLabel(weighting string) string {
	switch store.NormalizeBallotWeighting(weighting) {
	case store.BallotWeightingPerHead:
		return "pro Kopf"
	case store.BallotWeightingPerShare:
		return "nach Miteigentumsanteil"
	default:
		return ""
	}
}

func FormatParkingTariffDate(raw string) string {
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return raw
	}
	return FormatLocalDate(t)
}

type ParkingBalanceView struct {
	Outstanding float64
	Overdue     float64
}

// UserUnitAssignment preserves the stored unit type for grouped access presentation.
type UserUnitAssignment struct {
	Label string
	Kind  string
}

type UserRow struct {
	Email                  string
	Title                  string
	FirstName              string
	LastName               string
	Phone                  string
	DirectoryOptIn         bool
	DisplayName            string
	Initials               string
	Role                   string
	RoleClass              string
	RoleCapabilities       []string
	Status                 string
	Tenants                string
	PermissionLabel        string
	PermissionList         []string
	ParkingChecked         bool
	EnergyCaretakerChecked bool
	SupportViewChecked     bool
	OutstandingBalance     string
	HasOutstanding         bool
	AuthLabel              string
	AuthList               []string
	EmailAuthChecked       bool
	OIDCAuthChecked        bool
	UnitAssignments        []UserUnitAssignment
	UnitList               []string
	HasUnits               bool
	Editable               bool
	IsConfig               bool
	Protected              bool
	Deactivated            bool
	LastSeen               string
}

// User-facing formatting convention: de-AT copy uses local time, dot-grouped
// thousands and comma decimals. HTML control values use browser-native layouts.
func FormatLocalDate(t time.Time) string {
	return FormatDateTimeIn(t, time.Local, DeATDateLayout)
}

func FormatLocalDateTime(t time.Time) string {
	return FormatDateTimeIn(t, time.Local, DeATDateTimeLayout)
}

func FormatLocalShortDateTime(t time.Time) string {
	return FormatDateTimeIn(t, time.Local, DeATShortDateTimeLayout)
}

func FormatLocalTime(t time.Time) string {
	return FormatDateTimeIn(t, time.Local, DeATTimeLayout)
}

func FormatLocalDateTimeInput(t time.Time) string {
	return FormatDateTimeIn(t, time.Local, HTMLDateTimeLocalLayout)
}

func FormatDateTimeIn(t time.Time, loc *time.Location, layout string) string {
	if loc == nil {
		loc = time.Local
	}
	return t.In(loc).Format(layout)
}

func FormatInputFloat(value float64) string {
	return FormatDecimal(value, 3)
}

// FormatEURCents shares the statement receipt formatter with PDF exports.
// Integer cents must not lose precision through a float64 conversion.
func FormatEURCents(cents int64) string {
	raw := strconv.FormatInt(cents, 10)
	sign := ""
	if strings.HasPrefix(raw, "-") {
		sign, raw = "-", raw[1:]
	}
	for len(raw) < 3 {
		raw = "0" + raw
	}
	digits, fraction := raw[:len(raw)-2], raw[len(raw)-2:]
	for index := len(digits) - 3; index > 0; index -= 3 {
		digits = digits[:index] + "." + digits[index:]
	}
	return sign + digits + "," + fraction + " €"
}

func FormatEUR(value float64) string {
	return FormatDecimal(value, 2) + " €"
}

func FormatEURPerKWh(value float64) string {
	return FormatDecimal(value, 3) + " €/kWh"
}

func FormatKWh(value float64) string {
	return FormatDecimal(value, 2) + " kWh"
}

func FormatPreciseEUR(value float64) string {
	return FormatDecimal(value, 6) + " €"
}

func FormatPreciseEURPerKWh(value float64) string {
	return FormatDecimal(value, 6) + " €/kWh"
}

func FormatPreciseKWh(value float64) string {
	return FormatDecimal(value, 6) + " kWh"
}

func FormatDecimal(value float64, decimals int) string {
	if decimals < 0 {
		decimals = 0
	}
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	raw := fmt.Sprintf("%.*f", decimals, value)
	parts := strings.SplitN(raw, ".", 2)
	intPart := parts[0]
	for i := len(intPart) - 3; i > 0; i -= 3 {
		intPart = intPart[:i] + "." + intPart[i:]
	}
	if decimals == 0 || len(parts) == 1 {
		return sign + intPart
	}
	return sign + intPart + "," + parts[1]
}

func FormatMonthLabel(month string, loc *time.Location) string {
	t, err := time.ParseInLocation("2006-01", month, loc)
	if err != nil {
		return month
	}
	names := []string{"Jänner", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"}
	return names[int(t.Month())-1] + " " + strconv.Itoa(t.Year())
}

func UnitCountLabel(count int) string {
	if count == 1 {
		return "Wohneinheit"
	}
	return "Wohneinheiten"
}

func BillableUnitCountLabel(weight int) string {
	if weight == store.UnitBillableFullPPM {
		return "Wohneinheit"
	}
	return "Wohneinheiten"
}

func FormatBillableUnitWeight(weight int) string {
	if weight <= 0 {
		return "0"
	}
	value := float64(weight) / float64(store.UnitBillableFullPPM)
	if weight%store.UnitBillableFullPPM == 0 {
		return FormatDecimal(value, 0)
	}
	return strings.TrimRight(strings.TrimRight(FormatDecimal(value, 2), "0"), ",")
}

func FormatPeriodLabel(first time.Time, last time.Time, loc *time.Location) string {
	if first.IsZero() || last.IsZero() {
		return "Noch keine Messwerte"
	}
	if loc == nil {
		loc = time.Local
	}
	return FormatDateTimeIn(first, loc, DeATShortDateTimeLayout) + " bis " + FormatDateTimeIn(last, loc, DeATShortDateTimeLayout)
}

func PaidLabel(paid bool) string {
	if paid {
		return "BEZAHLT"
	}
	return "OFFEN"
}

func TogglePaidLabel(paid bool) string {
	if paid {
		return "Als offen markieren"
	}
	return "Als bezahlt markieren"
}

func AuthMethodsLabel(methods []string) string {
	return strings.Join(AuthMethodsLabelList(methods), ", ")
}

func PermissionLabel(permissions []string) string {
	return strings.Join(PermissionLabelList(permissions), ", ")
}

func AnnouncementUnread(item store.Announcement, lastSeen time.Time, now time.Time) bool {
	if item.PublishedAt.After(now) {
		return false
	}
	if item.ExpiresAt != nil && !item.ExpiresAt.After(now) {
		return false
	}
	if lastSeen.IsZero() {
		return true
	}
	return item.PublishedAt.After(lastSeen)
}

func SameLocalDate(a time.Time, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func PlainTextHTML(body string) template.HTML {
	escaped := template.HTMLEscapeString(strings.TrimSpace(body))
	escaped = strings.ReplaceAll(escaped, "\r\n", "\n")
	escaped = strings.ReplaceAll(escaped, "\n\n", "<br><br>")
	escaped = strings.ReplaceAll(escaped, "\n", "<br>")
	return template.HTML(escaped)
}

func EventTimeRange(item store.HouseEvent) string {
	startLocal := item.StartsAt.In(time.Local)
	if item.EndsAt == nil {
		return FormatLocalTime(startLocal)
	}
	endLocal := item.EndsAt.In(time.Local)
	if SameLocalDate(startLocal, endLocal) {
		return FormatLocalTime(startLocal) + " bis " + FormatLocalTime(endLocal)
	}
	return FormatLocalShortDateTime(startLocal) + " bis " + FormatLocalShortDateTime(endLocal)
}

func GermanMonthShort(t time.Time) string {
	months := [...]string{"Jan", "Feb", "Mär", "Apr", "Mai", "Jun", "Jul", "Aug", "Sep", "Okt", "Nov", "Dez"}
	month := int(t.Month())
	if month < 1 || month > len(months) {
		return ""
	}
	return months[month-1]
}

// GermanDateLong renders a spoken-language date such as
// "Freitag, 1. August 2026". The portal uses it to date the daily focus, so it
// stays in the presentation layer next to the other German labels.
func GermanDateLong(t time.Time) string {
	weekdays := [...]string{"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"}
	months := [...]string{"Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"}
	month := int(t.Month())
	if month < 1 || month > len(months) {
		return ""
	}
	return weekdays[int(t.Weekday())%len(weekdays)] + ", " + strconv.Itoa(t.Day()) + ". " + months[month-1] + " " + strconv.Itoa(t.Year())
}

// GermanDateShort renders "Do, 17.09.2026": the abbreviated German weekday
// plus the numeric date. Go's layout has no German weekday token, so callers
// used to write a literal "Mo," that was wrong six days a week.
func GermanDateShort(t time.Time) string {
	weekdays := [...]string{"So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"}
	return weekdays[int(t.Weekday())%len(weekdays)] + ", " + t.Format("02.01.2006")
}

func IssueStatuses() []string {
	return []string{store.IssueStatusNew, store.IssueStatusAccepted, store.IssueStatusScheduled, store.IssueStatusProgress, store.IssueStatusDone, store.IssueStatusRejected, store.IssueStatusDuplicate}
}

func IssuePriorities() []string {
	return []string{store.IssuePriorityLow, store.IssuePriorityNorm, store.IssuePriorityHigh, store.IssuePriorityUrgent}
}

func IssueCategories() []string {
	return []string{"Reparatur", "Frage", "Vorschlag", "Sonstiges"}
}

// notificationEventCatalog pairs the store's event vocabulary with the German
// labels shown in the UI. The keys belong to the store; the words belong here.
func NotificationEventCatalog() []NotificationEventOption {
	labels := map[string][2]string{
		store.NotificationEventAnnouncement: {"Aushänge & Bekanntmachungen", "Neue und wichtige Informationen zur Liegenschaft"},
		store.NotificationEventIssue:        {"Anliegen & Status", "Kommentare und Änderungen bei Anliegen"},
		store.NotificationEventVote:         {"Abstimmungen", "Neue Abstimmungen und Erinnerungen"},
		store.NotificationEventDocument:     {"Dokumente", "Neu bereitgestellte Unterlagen"},
		store.NotificationEventPayment:      {"Zahlungen", "Fällige oder überfällige Zahlungen"},
		store.NotificationEventCharging:     {"Parkplatz-Laden", "Start und Ende eigener Ladevorgänge"},
	}
	out := make([]NotificationEventOption, 0, len(store.NotificationEvents))
	for _, key := range store.NotificationEvents {
		out = append(out, NotificationEventOption{Key: key, Label: labels[key][0], Description: labels[key][1]})
	}
	return out
}

func UnitPaymentStatusDetail(status string, hasRecord bool) string {
	if !hasRecord {
		return "Noch nicht gesetzt."
	}
	switch store.NormalizeUnitPaymentStatus(status) {
	case store.UnitPaymentStatusPaid:
		return "Als bezahlt markiert."
	case store.UnitPaymentStatusPartial:
		return "Teilzahlung vorgemerkt."
	case store.UnitPaymentStatusOverdue:
		return "Bitte zeitnah prüfen."
	default:
		return "Offen vorgemerkt."
	}
}

func DocumentCategories() []string {
	return []string{
		store.DocumentCategoryProtocol,
		store.DocumentCategoryBilling,
		store.DocumentCategoryRules,
		store.DocumentCategoryContract,
		store.DocumentCategoryPlan,
		store.DocumentCategoryOther,
	}
}

func DocumentCanPreview(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return store.IsImageContentType(contentType) || contentType == "application/pdf"
}

func DocumentFileKind(item store.DocumentRecord) string {
	contentType := strings.ToLower(strings.TrimSpace(item.ContentType))
	if strings.Contains(contentType, "pdf") {
		return "PDF"
	}
	if strings.HasPrefix(contentType, "image/") {
		return "Bild"
	}
	ext := strings.TrimPrefix(strings.ToUpper(filepath.Ext(item.Filename)), ".")
	if ext != "" && len([]rune(ext)) <= 5 {
		return ext
	}
	return "Datei"
}

func SelectedDocumentSort(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "oldest", "title":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "newest"
	}
}

func AuthMethodsLabelList(methods []string) []string {
	normalized, err := store.NormalizeAuthMethods(methods)
	if err != nil {
		return []string{"Ungültig"}
	}
	labels := make([]string, 0, len(normalized))
	for _, method := range normalized {
		switch method {
		case store.AuthMethodEmail:
			labels = append(labels, "E-Mail-Link")
		case store.AuthMethodOIDC:
			labels = append(labels, "Sichere Anmeldung")
		default:
			labels = append(labels, method)
		}
	}
	return labels
}

func PermissionLabelList(permissions []string) []string {
	labels := []string{}
	for _, permission := range store.NormalizePermissions(permissions) {
		switch permission {
		case store.PermissionSupportView:
			labels = append(labels, "Supportansicht")
		case store.PermissionParking:
			labels = append(labels, "Parkplatznutzung")
		case store.PermissionEnergyCaretaker:
			labels = append(labels, "Technische Vertrauensperson")
		default:
			labels = append(labels, permission)
		}
	}
	if len(labels) == 0 {
		return []string{"Standard"}
	}
	return labels
}

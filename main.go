package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/websocket"
	"golang.org/x/oauth2"
)

//go:embed assets/*
var assets embed.FS

const (
	roleAdmin                     = "Admin"
	roleManager                   = "Verwalter"
	roleOwner                     = "Eigentümer"
	roleRenter                    = "Mieter"
	roleBeirat                    = "Beirat"
	roleResident                  = "Bewohner"
	permissionParking             = "parking"
	authMethodEmail               = "email"
	authMethodOIDC                = "oidc"
	issueStatusNew                = "Neu"
	issueStatusProgress           = "In Bearbeitung"
	issueStatusDone               = "Erledigt"
	issueStatusRejected           = "Abgelehnt"
	issueStatusDuplicate          = "Duplikat"
	issueStatusOpen               = issueStatusNew
	issuePriorityLow              = "Niedrig"
	issuePriorityNorm             = "Mittel"
	issuePriorityHigh             = "Hoch"
	issuePriorityUrgent           = "Dringend"
	issueLocationUnit             = "own-unit"
	issueLocationCommon           = "common"
	notificationEventAnnouncement = "announcement"
	notificationEventIssue        = "issue"
	notificationEventVote         = "vote"
	notificationEventDocument     = "document"
	notificationEventPayment      = "payment"
)

const (
	defaultTenantHeroImageURL = "/assets/jhw22-hero.jpg"
	maxIssuePhotoBytes        = 5 << 20
	maxIssueFormBytes         = maxIssuePhotoBytes + (1 << 20)
	maxTenantHeroBytes        = 5 << 20
	maxTenantHeroFormBytes    = maxTenantHeroBytes + (1 << 20)
	maxDocumentBytes          = 20 << 20
	maxDocumentFormBytes      = maxDocumentBytes + (1 << 20)
)

const (
	auditActionLogin            = "login"
	auditActionInviteCreate     = "invite.create"
	auditActionInviteUpdate     = "invite.update"
	auditActionInviteDelete     = "invite.delete"
	auditActionBuildingUpdate   = "building.update"
	auditActionHeroUpdate       = "building.hero"
	auditActionUnitSave         = "building.unit.save"
	auditActionUnitDelete       = "building.unit.delete"
	auditActionParkingSettings  = "parking.settings"
	auditActionParkingMonth     = "parking.month"
	auditActionIssueWorkflow    = "issue.workflow"
	auditActionDocumentUpload   = "document.upload"
	auditActionDocumentDownload = "document.download"
	auditActionDocumentReplace  = "document.replace"
	auditActionVoteCreate       = "vote.create"
	auditActionVoteOpen         = "vote.open"
	auditActionVoteClose        = "vote.close"
	auditActionVoteCast         = "vote.cast"
	auditActionVoteReminder     = "vote.reminder"
)

const (
	documentCategoryProtocol = "Protokoll"
	documentCategoryBilling  = "Abrechnung"
	documentCategoryRules    = "Hausordnung"
	documentCategoryContract = "Vertrag"
	documentCategoryPlan     = "Plan"
	documentCategoryOther    = "Sonstiges"

	documentVisibilityAllResidents = "all-residents"
	documentVisibilityOwnersOnly   = "owners-only"
	documentVisibilityManagerOnly  = "verwalter-only"
)

const (
	ballotTypeMeeting  = "Versammlung"
	ballotTypeCircular = "Umlaufbeschluss"

	ballotWeightingPerShare = "per-share"
	ballotWeightingPerHead  = "per-head"

	ballotStatusDraft  = "Entwurf"
	ballotStatusOpen   = "Offen"
	ballotStatusClosed = "Geschlossen"

	defaultBallotReminderBeforeMinutes = 24 * 60
	maxBallotReminderBeforeMinutes     = 30 * 24 * 60
)

type capability string

const (
	capabilityPlatformAdmin       capability = "platform-admin"
	capabilityManageUsers         capability = "manage-users"
	capabilityManageParking       capability = "manage-parking"
	capabilityManageAnnouncements capability = "manage-announcements"
	capabilityManageDocuments     capability = "manage-documents"
	capabilityManageIssues        capability = "manage-issues"
	capabilityManageVotes         capability = "manage-votes"
	capabilityManageBuilding      capability = "manage-building"
	capabilityOwnerDocuments      capability = "owner-documents"
	capabilityVote                capability = "vote"
	capabilityOversight           capability = "oversight"
)

var (
	appVersion = "0.1.0"
	gitCommit  = "dev"
)

type app struct {
	baseURL               string
	addr                  string
	rootDomain            string
	defaultTenant         string
	tenants               map[string]tenantConfig
	sessionSecure         bool
	allowed               map[string]struct{}
	admins                map[string]struct{}
	profiles              map[string]userProfile
	localDevLogin         bool
	sessionTTL            time.Duration
	tokens                *tokenStore
	sessions              *sessionStore
	oidc                  *oidcLogin
	oidcFlows             *oidcFlowStore
	mailer                mailer
	templates             *template.Template
	announcementStore     *announcementStore
	announcementReadStore *announcementReadStore
	eventStore            *eventStore
	notificationPrefs     *notificationPrefStore
	profileOverlays       *profileOverlayStore
	tenantOverrides       *tenantOverrideStore
	tenantHeroDir         string
	inviteStore           *inviteStore
	activityStore         *activityStore
	unitStore             *unitStore
	issueStore            *issueStore
	auditStore            *auditStore
	documentStore         *documentStore
	voteStore             *voteStore
	voteReminderInterval  time.Duration
	parkingStore          *parkingStore
	parkingSampleInterval time.Duration
	parkingHistoryStart   time.Time
}

type tokenStore struct {
	mu     sync.Mutex
	secret []byte
	items  map[string]loginToken
}

type loginToken struct {
	email      string
	tenantSlug string
	expiresAt  time.Time
	used       bool
}

type sessionStore struct {
	mu      sync.Mutex
	secret  []byte
	revoked map[string]time.Time
}

type session struct {
	Email      string `json:"email"`
	TenantSlug string `json:"tenant_slug"`
	AuthMethod string `json:"auth_method"`
	ExpiresAt  int64  `json:"expires_at"`
}

type oidcLogin struct {
	mu           sync.Mutex
	providerName string
	issuer       string
	clientID     string
	clientSecret string
	redirectURL  string
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
}

type oidcFlowStore struct {
	mu    sync.Mutex
	items map[string]oidcFlow
}

type oidcFlow struct {
	tenantSlug   string
	nonce        string
	codeVerifier string
	expiresAt    time.Time
	used         bool
}

type oidcUserClaims struct {
	Email         string `json:"email"`
	EmailVerified *bool  `json:"email_verified"`
}

type mailer interface {
	SendMagicLink(to string, link string) error
	SendInvite(to string, loginURL string, address string) error
	SendNotification(to string, subject string, body string) error
	Configured() bool
}

type smtpMailer struct {
	host string
	port string
	user string
	pass string
	from string
}

type tenantConfig struct {
	Slug           string              `json:"slug"`
	Name           string              `json:"name"`
	Address        string              `json:"address"`
	ContactName    string              `json:"contact_name,omitempty"`
	ContactEmail   string              `json:"contact_email,omitempty"`
	ContactPhone   string              `json:"contact_phone,omitempty"`
	EmergencyName  string              `json:"emergency_name,omitempty"`
	EmergencyPhone string              `json:"emergency_phone,omitempty"`
	CaretakerName  string              `json:"caretaker_name,omitempty"`
	CaretakerEmail string              `json:"caretaker_email,omitempty"`
	CaretakerPhone string              `json:"caretaker_phone,omitempty"`
	HeroImageURL   string              `json:"hero_image_url,omitempty"`
	Host           string              `json:"host"`
	HA             homeAssistantConfig `json:"-"`
}

type homeAssistantConfig struct {
	baseURL           string
	token             string
	meterEnergyEntity string
	powerEntity       string
	priceEntity       string
}

type haState struct {
	EntityID   string         `json:"entity_id"`
	State      string         `json:"state"`
	Attributes map[string]any `json:"attributes"`
}

type haHistoryState struct {
	EntityID    string         `json:"entity_id"`
	State       string         `json:"state"`
	LastChanged time.Time      `json:"last_changed"`
	LastUpdated time.Time      `json:"last_updated"`
	Attributes  map[string]any `json:"attributes"`
}

type haStatistic struct {
	Start json.RawMessage `json:"start"`
	End   json.RawMessage `json:"end"`
	State *float64        `json:"state"`
	Sum   *float64        `json:"sum"`
	Mean  *float64        `json:"mean"`
	Min   *float64        `json:"min"`
	Max   *float64        `json:"max"`
}

type parkingTelemetry struct {
	Configured bool
	Connected  bool
	Message    string
	Metrics    []parkingMetric
	Entities   []parkingEntityRef
}

type parkingMetric struct {
	Label  string
	Value  string
	Detail string
}

type parkingEntityRef struct {
	Label    string
	EntityID string
}

type announcementStore struct {
	mu   sync.Mutex
	path string
	data announcementStoreData
}

type announcementStoreData struct {
	Announcements []announcement `json:"announcements"`
}

type announcementReadStore struct {
	mu   sync.Mutex
	path string
	data announcementReadStoreData
}

type announcementReadStoreData struct {
	Seen map[string]map[string]time.Time `json:"seen"`
}

type eventStore struct {
	mu   sync.Mutex
	path string
	data eventStoreData
}

type eventStoreData struct {
	Events []houseEvent `json:"events"`
}

type notificationPrefStore struct {
	mu   sync.Mutex
	path string
	data notificationPrefStoreData
}

type notificationPrefStoreData struct {
	Users map[string]notificationPreferences `json:"users"`
}

type profileOverlayStore struct {
	mu   sync.Mutex
	path string
	data profileOverlayStoreData
}

type profileOverlayStoreData struct {
	Profiles map[string]profileOverlay `json:"profiles"`
}

type profileOverlay struct {
	Title          string    `json:"title"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	Phone          string    `json:"phone,omitempty"`
	DirectoryOptIn bool      `json:"directory_opt_in,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

type tenantOverrideStore struct {
	mu   sync.Mutex
	path string
	data tenantOverrideStoreData
}

type tenantOverrideStoreData struct {
	Tenants map[string]tenantOverride `json:"tenants"`
}

type tenantOverride struct {
	MetaSet        bool      `json:"meta_set,omitempty"`
	Name           string    `json:"name,omitempty"`
	Address        string    `json:"address,omitempty"`
	ContactName    string    `json:"contact_name,omitempty"`
	ContactEmail   string    `json:"contact_email,omitempty"`
	ContactPhone   string    `json:"contact_phone,omitempty"`
	EmergencyName  string    `json:"emergency_name,omitempty"`
	EmergencyPhone string    `json:"emergency_phone,omitempty"`
	CaretakerName  string    `json:"caretaker_name,omitempty"`
	CaretakerEmail string    `json:"caretaker_email,omitempty"`
	CaretakerPhone string    `json:"caretaker_phone,omitempty"`
	HeroImage      string    `json:"hero_image,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

type notificationPreferences struct {
	Email        map[string]bool `json:"email"`
	Unsubscribed bool            `json:"unsubscribed,omitempty"`
}

type notificationEventOption struct {
	Key         string
	Label       string
	Description string
	Checked     bool
}

type portalNotification struct {
	Event      string
	Tenant     tenantConfig
	Recipients []string
	ActorEmail string
	Subject    string
	Lines      []string
	ActionURL  string
	ActionText string
}

type announcement struct {
	ID          string     `json:"id"`
	TenantSlug  string     `json:"tenant"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	Category    string     `json:"category"`
	Pinned      bool       `json:"pinned"`
	PublishedAt time.Time  `json:"published_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	AuthorEmail string     `json:"author_email"`
	AuthorName  string     `json:"author_name"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type announcementView struct {
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
	EditDialogID       string
	DeleteConfirmLabel string
}

type houseEvent struct {
	ID          string     `json:"id"`
	TenantSlug  string     `json:"tenant"`
	Title       string     `json:"title"`
	Body        string     `json:"body,omitempty"`
	Category    string     `json:"category"`
	Location    string     `json:"location,omitempty"`
	StartsAt    time.Time  `json:"starts_at"`
	EndsAt      *time.Time `json:"ends_at,omitempty"`
	AuthorEmail string     `json:"author_email"`
	AuthorName  string     `json:"author_name"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type houseEventView struct {
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
	EndsAt             string
	EndsAtInput        string
	HasEndsAt          bool
	DateBadgeDay       string
	DateBadgeMonth     string
	TimeRange          string
	Status             string
	Past               bool
	Author             string
	EditDialogID       string
	DeleteConfirmLabel string
}

type issueStore struct {
	mu            sync.Mutex
	path          string
	attachmentDir string
	data          issueStoreData
}

type issueStoreData struct {
	Issues []residentIssue `json:"issues"`
}

type residentIssue struct {
	ID              string              `json:"id"`
	TenantSlug      string              `json:"tenant"`
	AuthorEmail     string              `json:"author_email"`
	AuthorName      string              `json:"author_name"`
	Category        string              `json:"category"`
	Title           string              `json:"title"`
	Body            string              `json:"body"`
	LocationType    string              `json:"location_type"`
	LocationDetail  string              `json:"location_detail"`
	PhotoPaths      []string            `json:"photo_paths"`
	Status          string              `json:"status"`
	Priority        string              `json:"priority"`
	AssigneeEmail   string              `json:"assignee_email,omitempty"`
	StatusChangedAt time.Time           `json:"status_changed_at,omitempty"`
	StatusChangedBy string              `json:"status_changed_by,omitempty"`
	StatusHistory   []issueStatusChange `json:"status_history,omitempty"`
	Comments        []issueComment      `json:"comments,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

type issueComment struct {
	ID          string    `json:"id"`
	AuthorEmail string    `json:"author_email"`
	AuthorName  string    `json:"author_name"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"created_at"`
}

type issueCommentView struct {
	Author    string
	Body      string
	CreatedAt string
}

type issueStatusChange struct {
	From       string    `json:"from"`
	To         string    `json:"to"`
	ActorEmail string    `json:"actor_email"`
	ActorName  string    `json:"actor_name"`
	ChangedAt  time.Time `json:"changed_at"`
}

type issueWorkflowUpdate struct {
	Status        string
	Priority      string
	AssigneeEmail string
	ActorEmail    string
	ActorName     string
	ChangedAt     time.Time
}

type selectOption struct {
	Value    string
	Label    string
	Selected bool
}

type documentStore struct {
	mu      sync.Mutex
	path    string
	fileDir string
	data    documentStoreData
}

type documentStoreData struct {
	Documents []documentRecord `json:"documents"`
}

type documentRecord struct {
	ID             string    `json:"id"`
	SeriesID       string    `json:"series_id,omitempty"`
	Version        int       `json:"version"`
	Current        bool      `json:"current"`
	SupersedesID   string    `json:"supersedes_id,omitempty"`
	ReplacedByID   string    `json:"replaced_by_id,omitempty"`
	TenantSlug     string    `json:"tenant"`
	Title          string    `json:"title"`
	Category       string    `json:"category"`
	Visibility     string    `json:"visibility"`
	UnitID         string    `json:"unit_id,omitempty"`
	Filename       string    `json:"filename"`
	StoredFilename string    `json:"stored_filename"`
	Size           int64     `json:"size"`
	ContentType    string    `json:"content_type"`
	UploadedBy     string    `json:"uploaded_by"`
	UploadedAt     time.Time `json:"uploaded_at"`
}

type documentView struct {
	ID              string
	Title           string
	Category        string
	Visibility      string
	VisibilityClass string
	UnitLabel       string
	HasUnit         bool
	Filename        string
	Size            string
	ContentType     string
	UploadedBy      string
	UploadedAt      string
	DownloadURL     string
	VersionLabel    string
	ReplaceDialogID string
	Versions        []documentVersionView
	HasVersions     bool
}

type documentVersionView struct {
	ID          string
	Version     string
	Filename    string
	Size        string
	UploadedAt  string
	DownloadURL string
}

type documentCategoryView struct {
	Category     string
	Documents    []documentView
	HasDocuments bool
	EmptyMessage string
}

type voteStore struct {
	mu   sync.Mutex
	path string
	data voteStoreData
}

type voteStoreData struct {
	Ballots []ballot `json:"ballots"`
}

type ballot struct {
	ID                    string                `json:"id"`
	TenantSlug            string                `json:"tenant"`
	Title                 string                `json:"title"`
	Description           string                `json:"description,omitempty"`
	Options               []string              `json:"options"`
	Type                  string                `json:"type"`
	Weighting             string                `json:"weighting"`
	QuorumPPM             int                   `json:"quorum_ppm"`
	OpensAt               time.Time             `json:"opens_at,omitempty"`
	ClosesAt              time.Time             `json:"closes_at,omitempty"`
	CreatedBy             string                `json:"created_by"`
	CreatedAt             time.Time             `json:"created_at"`
	UpdatedAt             time.Time             `json:"updated_at"`
	Status                string                `json:"status"`
	Votes                 map[string]ballotVote `json:"votes,omitempty"`
	ReminderBeforeMinutes int                   `json:"reminder_before_minutes,omitempty"`
	ReminderSentAt        map[string]time.Time  `json:"reminder_sent_at,omitempty"`
}

type ballotVote struct {
	Option string    `json:"option"`
	Weight int       `json:"weight"`
	At     time.Time `json:"at"`
}

type ballotView struct {
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
	OpensAt             string
	HasOpensAt          bool
	ClosesAt            string
	HasClosesAt         bool
	CreatedAt           string
	UpdatedAt           string
	Options             []ballotOptionView
	CanVote             bool
	CanManage           bool
	CanOpen             bool
	CanClose            bool
	HasVote             bool
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
	EditDialogID        string
}

type ballotOptionView struct {
	Value        string
	Label        string
	Selected     bool
	VoteCount    int
	Weight       int
	WeightLabel  string
	Percent      int
	PercentStyle string
}

type issueView struct {
	ID              string
	Title           string
	Body            string
	Author          string
	AuthorEmail     string
	Category        string
	Status          string
	StatusClass     string
	Priority        string
	AssigneeEmail   string
	HasAssignee     bool
	Location        string
	CreatedAt       string
	CanComment      bool
	CanClose        bool
	CanReopen       bool
	PhotoCount      int
	HasPhotos       bool
	Comments        []issueCommentView
	HasComments     bool
	StatusOptions   []selectOption
	PriorityOptions []selectOption
}

type issueBoardFilterView struct {
	Status          string
	Priority        string
	Category        string
	Assignee        string
	Sort            string
	StatusOptions   []selectOption
	PriorityOptions []selectOption
	CategoryOptions []selectOption
	SortOptions     []selectOption
	HasActive       bool
}

type unitStore struct {
	mu   sync.Mutex
	path string
	data unitStoreData
}

type unitStoreData struct {
	Units []unit `json:"units"`
}

type unit struct {
	ID                    string   `json:"id"`
	TenantSlug            string   `json:"tenant"`
	Label                 string   `json:"label"`
	MiteigentumsanteilPPM int      `json:"miteigentumsanteil"`
	OwnerEmails           []string `json:"owner_emails,omitempty"`
	RenterEmails          []string `json:"renter_emails,omitempty"`
}

type unitMembership struct {
	Unit     unit
	Relation string
}

type profileUnitView struct {
	Label    string
	Relation string
	Share    string
}

type buildingUnitView struct {
	ID                 string
	Label              string
	Share              string
	ShareValue         string
	OwnerEmails        string
	RenterEmails       string
	DeleteConfirmLabel string
}

type contactCardView struct {
	Name        string
	Role        string
	Description string
	Email       string
	Phone       string
	HasEmail    bool
	HasPhone    bool
}

type emptyStateView struct {
	Title       string
	Message     string
	ActionURL   string
	ActionLabel string
	HasAction   bool
}

type dashboardDigestItem struct {
	Title  string
	Detail string
	URL    string
	Badge  string
}

type auditEvent struct {
	At         time.Time         `json:"at"`
	TenantSlug string            `json:"tenant"`
	ActorEmail string            `json:"actor_email"`
	ActorRole  string            `json:"actor_role,omitempty"`
	Action     string            `json:"action"`
	TargetType string            `json:"target_type,omitempty"`
	TargetID   string            `json:"target_id,omitempty"`
	Summary    string            `json:"summary"`
	Details    map[string]string `json:"details,omitempty"`
}

type auditFilter struct {
	TenantSlug string
	Action     string
	Query      string
	Limit      int
}

type auditEventView struct {
	At         string
	Action     string
	ActionText string
	Actor      string
	ActorRole  string
	Target     string
	TargetType string
	Summary    string
	Details    []auditDetailView
	HasDetails bool
}

type auditDetailView struct {
	Key   string
	Value string
}

type unitMembers struct {
	Unit    unit
	Owners  []string
	Renters []string
	Found   bool
}

type announcementFilterView struct {
	Label  string
	URL    string
	Active bool
}

type parkingStore struct {
	mu   sync.Mutex
	path string
	data parkingStoreData
}

type parkingStoreData struct {
	Tenants map[string]parkingTenantData `json:"tenants"`
}

type parkingTenantData struct {
	Settings      parkingSettings              `json:"settings"`
	Months        map[string]parkingMonthState `json:"months"`
	EnergySamples []parkingNumericSample       `json:"energy_samples"`
	PriceSamples  []parkingNumericSample       `json:"price_samples"`
	Samples       []parkingStoredSample        `json:"samples,omitempty"`
}

type parkingSettings struct {
	GridFeeEURPerKWh float64 `json:"grid_fee_eur_per_kwh"`
}

type parkingMonthState struct {
	Paid bool `json:"paid"`
}

type parkingStoredSample struct {
	At             time.Time `json:"at"`
	EnergyKWh      float64   `json:"energy_kwh"`
	PriceEURPerKWh float64   `json:"price_eur_per_kwh"`
}

type parkingNumericSample struct {
	At    time.Time `json:"at"`
	Value float64   `json:"value"`
}

type parkingAccountingView struct {
	Message          string
	GridFeeValue     string
	GridFeeLabel     string
	Months           []parkingMonthView
	HasMonths        bool
	LastSampleLabel  string
	HistoryAvailable bool
}

type parkingMonthView struct {
	Month           string
	MonthLabel      string
	DetailPath      string
	PeriodLabel     string
	KWh             string
	EnergyCost      string
	GridCost        string
	TotalCost       string
	AverageAwattar  string
	EffectivePrice  string
	AveragePrice    string
	Paid            bool
	PaidLabel       string
	TogglePaidValue string
	ToggleLabel     string
	ChartPercent    int
	Partial         bool
	SampleCount     int
	HourCount       int
}

type parkingMonthDetailView struct {
	Month           string
	MonthLabel      string
	BackPath        string
	Message         string
	GridFeeLabel    string
	LastSampleLabel string
	Summary         parkingMonthView
	Hours           []parkingHourView
	HasHours        bool
}

type parkingHourView struct {
	AtLabel             string
	AtTitle             string
	KWh                 string
	KWhTitle            string
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

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		target := "http://127.0.0.1:8080/healthz"
		if len(os.Args) > 2 {
			target = os.Args[2]
		}
		if err := runHealthcheck(target); err != nil {
			log.Printf("healthcheck failed: %v", err)
			os.Exit(1)
		}
		return
	}

	a, err := newApp()
	if err != nil {
		log.Fatal(err)
	}
	stopSampler := a.startParkingSampler()
	defer stopSampler()
	stopVoteReminders := a.startVoteReminderWorker()
	defer stopVoteReminders()

	mux := http.NewServeMux()
	mux.Handle("GET /assets/", http.FileServerFS(assets))
	mux.HandleFunc("GET /tenant-hero/{tenant}", a.tenantHeroImage)
	mux.HandleFunc("GET /healthz", a.health)
	mux.HandleFunc("GET /", a.home)
	mux.HandleFunc("POST /auth/request", a.requestLogin)
	mux.HandleFunc("GET /auth/verify", a.verifyLogin)
	mux.HandleFunc("GET /auth/oidc/start", a.startOIDCLogin)
	mux.HandleFunc("GET /auth/oidc/callback", a.finishOIDCLogin)
	mux.HandleFunc("POST /auth/logout", a.logout)
	mux.HandleFunc("GET /app", a.portal)
	mux.HandleFunc("GET /app/announcements", a.announcements)
	mux.HandleFunc("POST /app/announcements", a.createAnnouncement)
	mux.HandleFunc("POST /app/announcements/edit", a.editAnnouncement)
	mux.HandleFunc("POST /app/announcements/delete", a.deleteAnnouncement)
	mux.HandleFunc("GET /app/events", a.events)
	mux.HandleFunc("POST /app/events", a.createEvent)
	mux.HandleFunc("POST /app/events/edit", a.editEvent)
	mux.HandleFunc("POST /app/events/delete", a.deleteEvent)
	mux.HandleFunc("GET /app/dokumente", a.documents)
	mux.HandleFunc("POST /app/dokumente", a.uploadDocument)
	mux.HandleFunc("POST /app/dokumente/replace", a.replaceDocument)
	mux.HandleFunc("GET /app/dokumente/{id}/download", a.downloadDocument)
	mux.HandleFunc("GET /app/abstimmungen", a.ballots)
	mux.HandleFunc("POST /app/abstimmungen", a.submitBallot)
	mux.HandleFunc("POST /app/abstimmungen/open", a.openBallot)
	mux.HandleFunc("POST /app/abstimmungen/close", a.closeBallot)
	mux.HandleFunc("GET /app/abstimmungen/{id}/protokoll", a.ballotProtocol)
	mux.HandleFunc("GET /app/kontakte", a.contacts)
	mux.HandleFunc("GET /app/anliegen", a.issues)
	mux.HandleFunc("GET /app/anliegen/board", a.issueBoard)
	mux.HandleFunc("POST /app/anliegen", a.createIssue)
	mux.HandleFunc("POST /app/anliegen/comment", a.addIssueComment)
	mux.HandleFunc("POST /app/anliegen/workflow", a.updateIssueWorkflow)
	mux.HandleFunc("GET /app/parking", a.parking)
	mux.HandleFunc("GET /app/parking/settings", a.parkingSettings)
	mux.HandleFunc("GET /app/parking/month/{month}", a.parkingMonth)
	mux.HandleFunc("POST /app/parking/settings", a.updateParkingSettings)
	mux.HandleFunc("POST /app/parking/month", a.updateParkingMonth)
	mux.HandleFunc("GET /app/audit", a.auditLog)
	mux.HandleFunc("GET /app/settings", a.settingsHub)
	mux.HandleFunc("GET /app/settings/building", a.buildingSettings)
	mux.HandleFunc("POST /app/settings/building", a.updateBuildingSettings)
	mux.HandleFunc("POST /app/settings/building/hero", a.updateBuildingHero)
	mux.HandleFunc("POST /app/settings/building/units", a.upsertBuildingUnit)
	mux.HandleFunc("POST /app/settings/building/units/delete", a.deleteBuildingUnit)
	mux.HandleFunc("GET /app/settings/profile", a.profileSettings)
	mux.HandleFunc("POST /app/settings/profile", a.updateProfileSettings)
	mux.HandleFunc("GET /app/settings/notifications", a.notificationSettings)
	mux.HandleFunc("POST /app/settings/notifications", a.updateNotificationSettings)
	mux.HandleFunc("GET /app/settings/users", a.userSettings)
	mux.HandleFunc("POST /app/settings/users", a.createInvite)
	mux.HandleFunc("POST /app/settings/users/edit", a.editInvite)
	mux.HandleFunc("POST /app/settings/users/delete", a.deleteInvite)
	mux.HandleFunc("GET /{tenant}", a.tenantPathRedirect)
	mux.HandleFunc("GET /{tenant}/{rest...}", a.tenantPathRedirect)

	server := &http.Server{
		Addr:              a.addr,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("weg-portal listening on %s", a.addr)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func runHealthcheck(target string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}
	var payload struct {
		Service string `json:"service"`
		Status  string `json:"status"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 512)).Decode(&payload); err != nil {
		return fmt.Errorf("invalid health response: %w", err)
	}
	if payload.Service != "weg-portal" || payload.Status != "ok" {
		return fmt.Errorf("unexpected health response service=%q status=%q", payload.Service, payload.Status)
	}
	return nil
}

func newApp() (*app, error) {
	if err := loadLocalEnv(".env.local"); err != nil {
		return nil, err
	}

	baseURL := strings.TrimRight(env("BASE_URL", "http://localhost:8080"), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("BASE_URL must be an absolute URL")
	}

	publicURL := !isLocalHost(parsed.Hostname())
	secret, err := sessionSecret(publicURL)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("pages").Parse(pageTemplates)
	if err != nil {
		return nil, err
	}

	allowed := parseAllowed(env("INVITE_EMAILS", ""))
	admins := parseAllowed(env("ADMIN_EMAILS", ""))
	rootDomain := normalizeHost(env("ROOT_DOMAIN", "hausv.org"))
	defaultTenant := env("DEFAULT_TENANT", "jhw22")
	tenants, err := parseTenants(env("WEG_TENANTS_JSON", ""), rootDomain, defaultTenant, newHomeAssistantConfig())
	if err != nil {
		return nil, err
	}
	profiles, err := parseUserProfiles(env("WEG_USERS_JSON", ""), allowed, admins, defaultTenant)
	if err != nil {
		return nil, err
	}
	localDevLogin := parseBool(env("LOCAL_DEV_LOGIN", "false")) && isLocalHost(parsed.Hostname())

	mailTransport := smtpMailer{
		host: env("SMTP_HOST", ""),
		port: env("SMTP_PORT", "587"),
		user: env("SMTP_USER", ""),
		pass: env("SMTP_PASS", ""),
		from: env("MAIL_FROM", "WEG Portal <noreply@example.invalid>"),
	}
	if err := mailTransport.Validate(); err != nil {
		return nil, err
	}
	oidcCtx, cancelOIDC := context.WithTimeout(context.Background(), 10*time.Second)
	oidcLogin, err := newOIDCLogin(
		oidcCtx,
		env("OIDC_ISSUER", ""),
		env("OIDC_CLIENT_ID", ""),
		strings.TrimSpace(os.Getenv("OIDC_CLIENT_SECRET")),
		env("OIDC_REDIRECT_URL", ""),
		env("OIDC_PROVIDER_NAME", "Zitadel"),
	)
	cancelOIDC()
	if err != nil {
		return nil, err
	}
	if publicURL && !mailTransport.Configured() && !oidcLogin.Configured() {
		return nil, fmt.Errorf("SMTP or OIDC login is required when BASE_URL is public")
	}

	announcementDataPath := env("ANNOUNCE_DATA_PATH", "tmp/announcements.json")
	announcements, err := newAnnouncementStore(announcementDataPath)
	if err != nil {
		return nil, err
	}
	announcementReadDataPath := env("ANNOUNCE_READ_DATA_PATH", "tmp/announcement_reads.json")
	announcementReads, err := newAnnouncementReadStore(announcementReadDataPath)
	if err != nil {
		return nil, err
	}
	eventDataPath := env("EVENT_DATA_PATH", "tmp/events.json")
	events, err := newEventStore(eventDataPath)
	if err != nil {
		return nil, err
	}
	notificationPrefDataPath := env("NOTIFICATION_PREF_DATA_PATH", "tmp/notification_prefs.json")
	notificationPrefs, err := newNotificationPrefStore(notificationPrefDataPath)
	if err != nil {
		return nil, err
	}
	profileDataPath := env("PROFILE_DATA_PATH", "tmp/profile_overlays.json")
	profileOverlays, err := newProfileOverlayStore(profileDataPath)
	if err != nil {
		return nil, err
	}
	tenantDataPath := env("TENANT_DATA_PATH", "tmp/tenant_overrides.json")
	tenantOverrides, err := newTenantOverrideStore(tenantDataPath)
	if err != nil {
		return nil, err
	}
	tenantHeroDir := env("TENANT_HERO_DIR", filepath.Join(filepath.Dir(tenantDataPath), "tenant-heroes"))
	inviteDataPath := env("INVITE_DATA_PATH", "tmp/invites.json")
	invites, err := newInviteStore(inviteDataPath)
	if err != nil {
		return nil, err
	}
	activityDataPath := env("ACTIVITY_DATA_PATH", "tmp/activity.json")
	activity, err := newActivityStore(activityDataPath)
	if err != nil {
		return nil, err
	}
	unitDataPath := env("UNIT_DATA_PATH", "tmp/units.json")
	units, err := newUnitStore(unitDataPath)
	if err != nil {
		return nil, err
	}
	issueDataPath := env("ISSUE_DATA_PATH", "tmp/issues.json")
	defaultIssueAttachmentDir := filepath.Join(filepath.Dir(issueDataPath), "issue-attachments")
	issues, err := newIssueStore(issueDataPath, env("ISSUE_ATTACHMENT_DIR", defaultIssueAttachmentDir))
	if err != nil {
		return nil, err
	}
	auditDataPath := env("AUDIT_DATA_PATH", "tmp/audit.jsonl")
	auditStore, err := newAuditStore(auditDataPath)
	if err != nil {
		return nil, err
	}
	documentDataPath := env("DOC_DATA_PATH", "tmp/documents.json")
	defaultDocumentFileDir := filepath.Join(filepath.Dir(documentDataPath), "documents")
	documents, err := newDocumentStore(documentDataPath, env("DOC_FILE_DIR", defaultDocumentFileDir))
	if err != nil {
		return nil, err
	}
	voteDataPath := env("VOTE_DATA_PATH", "tmp/votes.json")
	votes, err := newVoteStore(voteDataPath)
	if err != nil {
		return nil, err
	}
	parkingDataPath := env("PARKING_DATA_PATH", "tmp/parking.json")
	parkingStore, err := newParkingStore(parkingDataPath)
	if err != nil {
		return nil, err
	}
	parkingSampleInterval, err := parseDuration(env("PARKING_SAMPLE_INTERVAL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid PARKING_SAMPLE_INTERVAL")
	}
	voteReminderInterval, err := parseDuration(env("VOTE_REMINDER_INTERVAL", "1h"))
	if err != nil {
		return nil, fmt.Errorf("invalid VOTE_REMINDER_INTERVAL")
	}
	parkingHistoryStart, err := parseHistoryStart(env("PARKING_HISTORY_START", ""), time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid PARKING_HISTORY_START")
	}
	sessionTTL, err := parseDuration(env("SESSION_TTL", "720h"))
	if err != nil || sessionTTL <= 0 {
		return nil, fmt.Errorf("invalid SESSION_TTL")
	}

	return &app{
		baseURL:       baseURL,
		addr:          env("ADDR", ":8080"),
		rootDomain:    rootDomain,
		defaultTenant: defaultTenant,
		tenants:       tenants,
		sessionSecure: parsed.Scheme == "https",
		allowed:       allowed,
		admins:        admins,
		profiles:      profiles,
		localDevLogin: localDevLogin,
		sessionTTL:    sessionTTL,
		tokens: &tokenStore{
			secret: secret,
			items:  map[string]loginToken{},
		},
		sessions:              newSessionStore(secret),
		oidc:                  oidcLogin,
		oidcFlows:             &oidcFlowStore{items: map[string]oidcFlow{}},
		mailer:                mailTransport,
		templates:             tmpl,
		announcementStore:     announcements,
		announcementReadStore: announcementReads,
		eventStore:            events,
		notificationPrefs:     notificationPrefs,
		profileOverlays:       profileOverlays,
		tenantOverrides:       tenantOverrides,
		tenantHeroDir:         tenantHeroDir,
		inviteStore:           invites,
		activityStore:         activity,
		unitStore:             units,
		issueStore:            issues,
		auditStore:            auditStore,
		documentStore:         documents,
		voteStore:             votes,
		voteReminderInterval:  voteReminderInterval,
		parkingStore:          parkingStore,
		parkingSampleInterval: parkingSampleInterval,
		parkingHistoryStart:   parkingHistoryStart,
	}, nil
}

func (a *app) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"service":"weg-portal","status":"ok"}`)
}

func (a *app) home(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, _, tenantSlug, ok := a.currentUser(r)
	if ok && tenantSlug == tenant.Slug {
		http.Redirect(w, r, "/app", http.StatusSeeOther)
		return
	}
	unitCount := 0
	if a.unitStore != nil {
		unitCount = a.unitStore.UnitCount(tenant.Slug)
	}
	titleName := firstNonEmpty(tenant.Name, "WEG Portal")
	a.render(w, "home", map[string]any{
		"Title":               titleName + " " + tenant.Address,
		"Tenant":              tenant,
		"Email":               email,
		"UnitCount":           unitCount,
		"UnitCountLabel":      unitCountLabel(unitCount),
		"HasUnitCount":        unitCount > 0,
		"Sent":                r.URL.Query().Get("sent") == "1",
		"MailConfigured":      a.mailer.Configured(),
		"DevLoginLink":        "",
		"Denied":              r.URL.Query().Get("denied") == "1",
		"OIDCConfigured":      a.oidc.Configured(),
		"OIDCProviderName":    a.oidc.ProviderName(),
		"EmailLoginAvailable": a.emailLoginAvailable(),
	})
}

func (a *app) requestLogin(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	if !a.emailLoginAvailable() {
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	email := normalizeEmail(r.FormValue("email"))
	if _, err := mail.ParseAddress(email); err != nil {
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}
	if !a.isAllowed(email, tenant.Slug) || !a.isAuthMethodAllowed(email, tenant.Slug, authMethodEmail) {
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}

	token, err := randomToken(32)
	if err != nil {
		http.Error(w, "Could not create login link", http.StatusInternalServerError)
		return
	}
	a.tokens.Put(token, email, tenant.Slug, 15*time.Minute)

	link := a.publicBaseURL(r, tenant) + "/auth/verify?token=" + url.QueryEscape(token)
	if a.localDevLogin && !a.mailer.Configured() {
		a.render(w, "home", map[string]any{
			"Title":               "WEG Portal " + tenant.Address,
			"Tenant":              tenant,
			"Email":               email,
			"Sent":                true,
			"MailConfigured":      false,
			"DevLoginLink":        link,
			"Denied":              false,
			"OIDCConfigured":      a.oidc.Configured(),
			"OIDCProviderName":    a.oidc.ProviderName(),
			"EmailLoginAvailable": a.emailLoginAvailable(),
		})
		return
	}

	if err := a.mailer.SendMagicLink(email, link); err != nil {
		log.Printf("magic link delivery failed for %s: %v", redactedEmail(email), err)
		http.Redirect(w, r, "/?sent=1", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/?sent=1", http.StatusSeeOther)
}

func (a *app) verifyLogin(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	email, tenantSlug, ok := a.tokens.Consume(token)
	if !ok {
		http.Error(w, "Dieser Anmeldelink ist abgelaufen oder wurde bereits verwendet.", http.StatusUnauthorized)
		return
	}

	if err := a.startSession(w, email, tenantSlug, authMethodEmail); err != nil {
		http.Error(w, "Could not create session", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/app", http.StatusSeeOther)
}

func (a *app) startOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if !a.oidc.Configured() {
		http.NotFound(w, r)
		return
	}
	tenant := a.tenantForRequest(r)
	if err := a.oidc.EnsureProvider(r.Context()); err != nil {
		log.Printf("oidc discovery failed during login start: %v", err)
		http.Error(w, "SSO ist gerade nicht erreichbar. Bitte später erneut versuchen oder den E-Mail-Link verwenden.", http.StatusServiceUnavailable)
		return
	}
	state, err := randomToken(32)
	if err != nil {
		http.Error(w, "Could not start SSO login", http.StatusInternalServerError)
		return
	}
	nonce, err := randomToken(32)
	if err != nil {
		http.Error(w, "Could not start SSO login", http.StatusInternalServerError)
		return
	}
	codeVerifier, err := randomToken(32)
	if err != nil {
		http.Error(w, "Could not start SSO login", http.StatusInternalServerError)
		return
	}
	a.oidcFlows.Put(state, oidcFlow{
		tenantSlug:   tenant.Slug,
		nonce:        nonce,
		codeVerifier: codeVerifier,
	}, 10*time.Minute)

	redirectURL := a.oidc.RedirectURL(r, tenant, a.publicBaseURL(r, tenant))
	oauthConfig := a.oidc.OAuthConfig(redirectURL)
	authCodeURL := oauthConfig.AuthCodeURL(
		state,
		oidc.Nonce(nonce),
		oauth2.SetAuthURLParam("code_challenge", pkceChallenge(codeVerifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	http.Redirect(w, r, authCodeURL, http.StatusSeeOther)
}

func (a *app) finishOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if !a.oidc.Configured() {
		http.NotFound(w, r)
		return
	}
	if err := a.oidc.EnsureProvider(r.Context()); err != nil {
		log.Printf("oidc discovery failed during login callback: %v", err)
		http.Error(w, "SSO ist gerade nicht erreichbar. Bitte später erneut versuchen.", http.StatusServiceUnavailable)
		return
	}
	if errText := strings.TrimSpace(r.URL.Query().Get("error")); errText != "" {
		log.Printf("oidc login failed: %s", errText)
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}
	flow, ok := a.oidcFlows.Consume(r.URL.Query().Get("state"))
	if !ok {
		http.Error(w, "Diese SSO-Anmeldung ist abgelaufen. Bitte erneut anmelden.", http.StatusUnauthorized)
		return
	}
	tenant, ok := a.tenantBySlug(flow.tenantSlug)
	if !ok {
		http.Error(w, "Unknown tenant", http.StatusUnauthorized)
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		http.Error(w, "SSO-Anmeldung ohne Code.", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	oauthConfig := a.oidc.OAuthConfig(a.oidc.RedirectURL(r, tenant, a.publicBaseURL(r, tenant)))
	token, err := oauthConfig.Exchange(
		ctx,
		code,
		oauth2.SetAuthURLParam("code_verifier", flow.codeVerifier),
	)
	if err != nil {
		log.Printf("oidc token exchange failed: %v", err)
		http.Error(w, "SSO-Anmeldung konnte nicht abgeschlossen werden.", http.StatusUnauthorized)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		log.Printf("oidc token exchange returned no id_token")
		http.Error(w, "SSO-Anmeldung konnte nicht geprüft werden.", http.StatusUnauthorized)
		return
	}
	idToken, err := a.oidc.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Printf("oidc id_token verification failed: %v", err)
		http.Error(w, "SSO-Anmeldung konnte nicht geprüft werden.", http.StatusUnauthorized)
		return
	}
	if idToken.Nonce != flow.nonce {
		log.Printf("oidc nonce mismatch")
		http.Error(w, "SSO-Anmeldung konnte nicht geprüft werden.", http.StatusUnauthorized)
		return
	}

	claims := oidcUserClaims{}
	if err := idToken.Claims(&claims); err != nil {
		log.Printf("oidc claims decode failed: %v", err)
		http.Error(w, "SSO-Anmeldung konnte nicht gelesen werden.", http.StatusUnauthorized)
		return
	}
	if claims.Email == "" || claims.EmailVerified == nil {
		userInfo, err := a.oidc.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
		if err != nil {
			log.Printf("oidc userinfo failed: %v", err)
		} else {
			var extra oidcUserClaims
			if err := userInfo.Claims(&extra); err == nil {
				claims.Merge(extra)
			}
		}
	}
	email := normalizeEmail(claims.Email)
	if email == "" || claims.EmailVerified == nil || !*claims.EmailVerified {
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}
	if !a.isAllowed(email, tenant.Slug) || !a.isAuthMethodAllowed(email, tenant.Slug, authMethodOIDC) {
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}
	if err := a.startSession(w, email, tenant.Slug, authMethodOIDC); err != nil {
		http.Error(w, "Could not create session", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/app", http.StatusSeeOther)
}

func (a *app) startSession(w http.ResponseWriter, email string, tenantSlug string, authMethod string) error {
	token, expiresAt, err := a.sessions.Put(email, tenantSlug, authMethod, a.sessionTTL)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "weg_session",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   a.sessionSecure,
		SameSite: http.SameSiteLaxMode,
	})
	if a.activityStore != nil {
		if err := a.activityStore.Touch(email, time.Now(), authMethod); err != nil {
			log.Printf("activity record failed for %s: %v", redactedEmail(email), err)
		}
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenantSlug,
		ActorEmail: email,
		ActorRole:  a.roleFor(email, tenantSlug),
		Action:     auditActionLogin,
		TargetType: "session",
		TargetID:   email,
		Summary:    "Anmeldung erfolgreich",
		Details: map[string]string{
			"auth_method": strings.Join(authMethodsLabelList([]string{authMethod}), ", "),
		},
	})
	return nil
}

func (a *app) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("weg_session"); err == nil {
		a.sessions.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "weg_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.sessionSecure,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *app) announcements(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	canManage := canManageAnnouncements(role)
	now := time.Now()
	selectedCategory := selectedAnnouncementCategory(r.URL.Query().Get("category"))
	searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	lastSeen := time.Time{}
	if a.announcementReadStore != nil {
		lastSeen = a.announcementReadStore.LastSeen(tenant.Slug, email)
	}
	archive := []announcement{}
	filtered := []announcement{}
	all := []announcementView{}
	if a.announcementStore != nil {
		archive = a.announcementStore.Archive(tenant.Slug, now)
		filtered = filterAnnouncements(archive, selectedCategory, searchQuery)
		if canManage {
			all = announcementViewsWithReadState(a.announcementStore.ListTenant(tenant.Slug), now, true, lastSeen)
		}
	}
	a.render(w, "announcements", map[string]any{
		"Title":                  "Aushang",
		"Tenant":                 tenant,
		"Email":                  email,
		"DisplayName":            profile.DisplayName(),
		"Initials":               profile.Initials(),
		"Role":                   role,
		"IsAdmin":                isAdmin,
		"CanSeeParking":          isAdmin || profile.HasPermission(permissionParking),
		"CanManageAnnouncements": canManage,
		"ActivePage":             "announcements",
		"Announcements":          announcementViewsWithReadState(filtered, now, true, lastSeen),
		"HasAnnouncements":       len(filtered) > 0,
		"HasAnyAnnouncements":    len(archive) > 0,
		"AnnouncementsEmpty":     emptyState("Keine Beiträge", "Für diese Suche oder Kategorie gibt es keinen Aushang."),
		"AnnouncementsBlank":     emptyState("Noch keine Beiträge", "Sobald ein Aushang veröffentlicht ist, erscheint er hier."),
		"AllAnnouncements":       all,
		"HasAllAnnouncements":    len(all) > 0,
		"AllAnnouncementsEmpty":  emptyState("Noch kein Aushang gespeichert", "Neue Aushänge erscheinen hier nach dem Speichern."),
		"AnnounceMsg":            announcementMessage(r.URL.Query().Get("announce")),
		"NowInput":               formatLocalDateTimeInput(now),
		"SearchQuery":            searchQuery,
		"SelectedCategory":       selectedCategory,
		"CategoryFilters":        announcementFilterViews(searchQuery, selectedCategory),
		"UnreadAnnouncements":    0,
		"HasUnreadAnnouncements": false,
	})
	if a.announcementReadStore != nil {
		if err := a.announcementReadStore.MarkSeen(tenant.Slug, email, now); err != nil {
			log.Printf("announcement read mark failed for %s/%s: %v", tenant.Slug, redactedEmail(email), err)
		}
	}
}

func (a *app) createAnnouncement(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !canManageAnnouncements(role) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	item, err := announcementFromForm(r, tenant.Slug, profile, time.Now())
	if err != nil {
		http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
		return
	}
	created, err := a.announcementStore.Create(item)
	if err != nil {
		log.Printf("announcement create failed for %s: %v", tenant.Slug, err)
		http.Redirect(w, r, "/app/announcements?announce=error", http.StatusSeeOther)
		return
	}
	a.notifyAnnouncementPublished(tenant, created, email)
	http.Redirect(w, r, "/app/announcements?announce=created", http.StatusSeeOther)
}

func (a *app) editAnnouncement(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !canManageAnnouncements(role) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	profile := a.profileForTenant(email, tenant.Slug)
	item, err := announcementFromForm(r, tenant.Slug, profile, time.Now())
	if err != nil {
		http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
		return
	}
	ok, err = a.announcementStore.Update(id, item)
	if err != nil {
		log.Printf("announcement update failed for %s: %v", tenant.Slug, err)
		http.Redirect(w, r, "/app/announcements?announce=error", http.StatusSeeOther)
		return
	}
	if !ok {
		http.Redirect(w, r, "/app/announcements?announce=missing", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/announcements?announce=updated", http.StatusSeeOther)
}

func (a *app) deleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	_, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !canManageAnnouncements(role) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	removed, err := a.announcementStore.Delete(tenant.Slug, strings.TrimSpace(r.FormValue("id")))
	if err != nil {
		log.Printf("announcement delete failed for %s: %v", tenant.Slug, err)
		http.Redirect(w, r, "/app/announcements?announce=error", http.StatusSeeOther)
		return
	}
	if !removed {
		http.Redirect(w, r, "/app/announcements?announce=missing", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/announcements?announce=deleted", http.StatusSeeOther)
}

func (a *app) events(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	canManage := canManageEvents(role)
	now := time.Now()
	upcoming := []houseEvent{}
	all := []houseEvent{}
	if a.eventStore != nil {
		upcoming = a.eventStore.Upcoming(tenant.Slug, now)
		if canManage {
			all = a.eventStore.ListTenant(tenant.Slug)
		}
	}
	a.render(w, "events", map[string]any{
		"Title":                  "Termine",
		"Tenant":                 tenant,
		"Email":                  email,
		"DisplayName":            profile.DisplayName(),
		"Initials":               profile.Initials(),
		"Role":                   role,
		"IsAdmin":                isAdmin,
		"CanSeeParking":          isAdmin || profile.HasPermission(permissionParking),
		"CanManageAnnouncements": canManageAnnouncements(role),
		"CanManageEvents":        canManage,
		"ActivePage":             "events",
		"Events":                 eventViews(upcoming, now),
		"HasEvents":              len(upcoming) > 0,
		"EventsEmpty":            emptyState("Noch keine kommenden Termine", "Geplante Versammlungen, Wartungen und Fristen erscheinen hier."),
		"AllEvents":              eventViews(all, now),
		"HasAllEvents":           len(all) > 0,
		"AllEventsEmpty":         emptyState("Noch kein Termin gespeichert", "Neue Termine erscheinen hier nach dem Speichern."),
		"EventMsg":               eventMessage(r.URL.Query().Get("event")),
		"NowInput":               formatLocalDateTimeInput(now),
	})
}

func (a *app) createEvent(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !canManageEvents(role) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	item, err := eventFromForm(r, tenant.Slug, profile)
	if err != nil {
		http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
		return
	}
	if _, err := a.eventStore.Create(item); err != nil {
		log.Printf("event create failed for %s: %v", tenant.Slug, err)
		http.Redirect(w, r, "/app/events?event=error", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/events?event=created", http.StatusSeeOther)
}

func (a *app) editEvent(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !canManageEvents(role) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	profile := a.profileForTenant(email, tenant.Slug)
	item, err := eventFromForm(r, tenant.Slug, profile)
	if err != nil {
		http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
		return
	}
	ok, err = a.eventStore.Update(id, item)
	if err != nil {
		log.Printf("event update failed for %s: %v", tenant.Slug, err)
		http.Redirect(w, r, "/app/events?event=error", http.StatusSeeOther)
		return
	}
	if !ok {
		http.Redirect(w, r, "/app/events?event=missing", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/events?event=updated", http.StatusSeeOther)
}

func (a *app) deleteEvent(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	_, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !canManageEvents(role) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	removed, err := a.eventStore.Delete(tenant.Slug, strings.TrimSpace(r.FormValue("id")))
	if err != nil {
		log.Printf("event delete failed for %s: %v", tenant.Slug, err)
		http.Redirect(w, r, "/app/events?event=error", http.StatusSeeOther)
		return
	}
	if !removed {
		http.Redirect(w, r, "/app/events?event=missing", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/events?event=deleted", http.StatusSeeOther)
}

func (a *app) documents(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	canManage := hasCapability(role, capabilityManageDocuments)
	visible := []documentRecord{}
	if a.documentStore != nil {
		visible = a.visibleDocumentsForActor(tenant.Slug, email, role)
	}
	searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	sortMode := selectedDocumentSort(r.URL.Query().Get("sort"))
	documents := sortDocumentsForView(filterDocuments(visible, searchQuery), sortMode)
	documentMsg, documentOK := documentMessage(r.URL.Query().Get("doc"))
	a.render(w, "documents", map[string]any{
		"Title":              "Dokumente",
		"Tenant":             tenant,
		"Email":              email,
		"DisplayName":        profile.DisplayName(),
		"Initials":           profile.Initials(),
		"Role":               role,
		"IsAdmin":            isAdmin,
		"CanSeeParking":      isAdmin || profile.HasPermission(permissionParking),
		"CanManageDocuments": canManage,
		"ActivePage":         "documents",
		"Documents":          a.documentViewsForActor(tenant.Slug, email, role, documents),
		"DocumentSections":   a.documentCategorySectionsForActor(tenant.Slug, email, role, documents, true),
		"HasDocuments":       len(documents) > 0,
		"HasAnyDocuments":    len(visible) > 0,
		"DocumentsEmpty":     emptyState("Noch keine Dokumente", "Sobald die Verwaltung ein Dokument hochlädt, erscheint es hier nach Sichtbarkeit gefiltert."),
		"DocumentMsg":        documentMsg,
		"DocumentOK":         documentOK,
		"SearchQuery":        searchQuery,
		"SortMode":           sortMode,
		"SortOptions":        documentSortOptions(sortMode),
		"CategoryOptions":    documentCategoryOptions(""),
		"VisibilityOptions":  documentVisibilityOptions(""),
		"UnitOptions":        documentUnitOptions(a.unitStore.ListTenant(tenant.Slug), ""),
		"MaxDocumentSize":    formatBytes(maxDocumentBytes),
	})
}

func (a *app) uploadDocument(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !hasCapability(role, capabilityManageDocuments) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if a.documentStore == nil {
		http.Redirect(w, r, "/app/dokumente?doc=invalid", http.StatusSeeOther)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxDocumentFormBytes)
	if err := r.ParseMultipartForm(maxDocumentBytes); err != nil {
		http.Redirect(w, r, "/app/dokumente?doc=invalid", http.StatusSeeOther)
		return
	}
	header := documentFileHeader(r)
	if header == nil {
		http.Redirect(w, r, "/app/dokumente?doc=invalid", http.StatusSeeOther)
		return
	}
	created, err := a.documentStore.Create(documentRecord{
		TenantSlug: tenant.Slug,
		Title:      strings.TrimSpace(r.FormValue("title")),
		Category:   normalizeDocumentCategory(r.FormValue("category")),
		Visibility: normalizeDocumentVisibility(r.FormValue("visibility")),
		UnitID:     normalizeUnitID(r.FormValue("unit_id")),
		UploadedBy: email,
	}, header, time.Now())
	if err != nil {
		log.Printf("document upload failed for %s/%s: %v", tenant.Slug, redactedEmail(email), err)
		http.Redirect(w, r, "/app/dokumente?doc=invalid", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     auditActionDocumentUpload,
		TargetType: "document",
		TargetID:   created.ID,
		Summary:    "Dokument hochgeladen",
		Details: map[string]string{
			"title":        created.Title,
			"category":     created.Category,
			"visibility":   documentVisibilityLabel(created.Visibility),
			"unit":         documentUnitAuditLabel(created.UnitID),
			"size":         formatBytes(created.Size),
			"content_type": created.ContentType,
		},
	})
	http.Redirect(w, r, "/app/dokumente?doc=uploaded", http.StatusSeeOther)
}

func (a *app) replaceDocument(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !hasCapability(role, capabilityManageDocuments) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if a.documentStore == nil {
		http.Redirect(w, r, "/app/dokumente?doc=invalid", http.StatusSeeOther)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxDocumentFormBytes)
	if err := r.ParseMultipartForm(maxDocumentBytes); err != nil {
		http.Redirect(w, r, "/app/dokumente?doc=invalid", http.StatusSeeOther)
		return
	}
	header := documentFileHeader(r)
	if header == nil {
		http.Redirect(w, r, "/app/dokumente?doc=invalid", http.StatusSeeOther)
		return
	}
	replacement, replaced, err := a.documentStore.Replace(tenant.Slug, strings.TrimSpace(r.FormValue("id")), email, header, time.Now())
	if err != nil {
		log.Printf("document replace failed for %s/%s: %v", tenant.Slug, redactedEmail(email), err)
		http.Redirect(w, r, "/app/dokumente?doc=invalid", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     auditActionDocumentReplace,
		TargetType: "document",
		TargetID:   replacement.ID,
		Summary:    "Dokument ersetzt",
		Details: map[string]string{
			"title":        replacement.Title,
			"category":     replacement.Category,
			"visibility":   documentVisibilityLabel(replacement.Visibility),
			"version":      documentVersionLabel(replacement.Version),
			"previous":     documentVersionLabel(replaced.Version),
			"previous_id":  replaced.ID,
			"content_type": replacement.ContentType,
			"size":         formatBytes(replacement.Size),
		},
	})
	http.Redirect(w, r, "/app/dokumente?doc=replaced", http.StatusSeeOther)
}

func (a *app) downloadDocument(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if a.documentStore == nil {
		http.NotFound(w, r)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	item, found := a.documentStore.Get(tenant.Slug, id)
	if !found {
		http.NotFound(w, r)
		return
	}
	if !a.canViewDocument(tenant.Slug, item, email, role) {
		http.Error(w, "Dieses Dokument ist für diesen Zugang nicht freigegeben.", http.StatusForbidden)
		return
	}
	path, ok := a.documentStore.FilePath(item)
	if !ok {
		http.NotFound(w, r)
		return
	}
	file, err := os.Open(path)
	if err != nil {
		log.Printf("document file open failed for %s/%s: %v", tenant.Slug, item.ID, err)
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": item.Filename}))
	if item.ContentType != "" {
		w.Header().Set("Content-Type", item.ContentType)
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     auditActionDocumentDownload,
		TargetType: "document",
		TargetID:   item.ID,
		Summary:    "Dokument heruntergeladen",
		Details: map[string]string{
			"title":      item.Title,
			"category":   item.Category,
			"visibility": documentVisibilityLabel(item.Visibility),
			"unit":       documentUnitAuditLabel(item.UnitID),
			"version":    documentVersionLabel(item.Version),
		},
	})
	http.ServeContent(w, r, item.Filename, item.UploadedAt, file)
}

func (a *app) createBallot(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !hasCapability(role, capabilityManageVotes) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if a.voteStore == nil {
		http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	item, err := ballotFromForm(r, tenant.Slug, email)
	if err != nil {
		http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
		return
	}
	created, err := a.voteStore.Create(item)
	if err != nil {
		log.Printf("ballot create failed for %s/%s: %v", tenant.Slug, redactedEmail(email), err)
		http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     auditActionVoteCreate,
		TargetType: "ballot",
		TargetID:   created.ID,
		Summary:    "Abstimmung angelegt",
		Details: map[string]string{
			"title":     created.Title,
			"type":      created.Type,
			"weighting": ballotWeightingLabel(created.Weighting),
			"quorum":    formatMiteigentumsanteil(created.QuorumPPM),
			"reminder":  formatBallotReminder(created.ReminderBeforeMinutes),
		},
	})
	http.Redirect(w, r, "/app/abstimmungen?vote=created", http.StatusSeeOther)
}

func (a *app) openBallot(w http.ResponseWriter, r *http.Request) {
	a.updateBallotStatus(w, r, ballotStatusOpen)
}

func (a *app) closeBallot(w http.ResponseWriter, r *http.Request) {
	a.updateBallotStatus(w, r, ballotStatusClosed)
}

func (a *app) updateBallotStatus(w http.ResponseWriter, r *http.Request, status string) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !hasCapability(role, capabilityManageVotes) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if a.voteStore == nil {
		http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	var (
		updated ballot
		found   bool
		err     error
		action  string
		summary string
		statusQ string
	)
	switch status {
	case ballotStatusOpen:
		updated, found, err = a.voteStore.Open(tenant.Slug, id, time.Now())
		action = auditActionVoteOpen
		summary = "Abstimmung geöffnet"
		statusQ = "opened"
	case ballotStatusClosed:
		updated, found, err = a.voteStore.Close(tenant.Slug, id, time.Now())
		action = auditActionVoteClose
		summary = "Abstimmung geschlossen"
		statusQ = "closed"
	default:
		err = fmt.Errorf("invalid ballot status")
	}
	if err != nil {
		log.Printf("ballot status update failed for %s/%s: %v", tenant.Slug, id, err)
		http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
		return
	}
	if !found {
		http.Redirect(w, r, "/app/abstimmungen?vote=missing", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     action,
		TargetType: "ballot",
		TargetID:   updated.ID,
		Summary:    summary,
		Details: map[string]string{
			"title":  updated.Title,
			"status": updated.Status,
		},
	})
	http.Redirect(w, r, "/app/abstimmungen?vote="+statusQ, http.StatusSeeOther)
}

func ballotFromForm(r *http.Request, tenantSlug string, createdBy string) (ballot, error) {
	opensAt, err := parseOptionalLocalDateTime(r.FormValue("opens_at"), time.Time{})
	if err != nil {
		return ballot{}, err
	}
	closesAt, err := parseOptionalLocalDateTime(r.FormValue("closes_at"), time.Time{})
	if err != nil {
		return ballot{}, err
	}
	if !opensAt.IsZero() {
		opensAt = opensAt.UTC()
	}
	if !closesAt.IsZero() {
		closesAt = closesAt.UTC()
	}
	quorum, err := parseBallotQuorumPPM(r.FormValue("quorum_ppm"), r.FormValue("quorum_percent"))
	if err != nil {
		return ballot{}, err
	}
	reminderBefore, err := parseBallotReminderBeforeMinutes(r.FormValue("reminder_before_minutes"), r.FormValue("reminder_before_hours"))
	if err != nil {
		return ballot{}, err
	}
	item := ballot{
		TenantSlug:            tenantSlug,
		Title:                 strings.TrimSpace(r.FormValue("title")),
		Description:           strings.TrimSpace(r.FormValue("description")),
		Options:               ballotOptionsFromForm(r),
		Type:                  normalizeBallotType(r.FormValue("type")),
		Weighting:             normalizeBallotWeighting(r.FormValue("weighting")),
		QuorumPPM:             quorum,
		OpensAt:               opensAt,
		ClosesAt:              closesAt,
		CreatedBy:             createdBy,
		ReminderBeforeMinutes: reminderBefore,
	}
	item = normalizeBallot(item)
	if item.Title == "" || len(item.Options) < 2 || item.Type == "" || item.Weighting == "" || item.CreatedBy == "" {
		return ballot{}, fmt.Errorf("invalid ballot")
	}
	return item, nil
}

func ballotOptionsFromForm(r *http.Request) []string {
	out := append([]string(nil), r.Form["options"]...)
	if text := strings.TrimSpace(r.FormValue("options_text")); text != "" {
		for _, line := range strings.Split(text, "\n") {
			out = append(out, line)
		}
	}
	return out
}

func parseBallotQuorumPPM(rawPPM string, rawPercent string) (int, error) {
	rawPPM = strings.TrimSpace(rawPPM)
	if rawPPM != "" {
		value, err := strconv.Atoi(rawPPM)
		if err != nil || value < 0 || value > 1_000_000 {
			return 0, fmt.Errorf("invalid quorum")
		}
		return value, nil
	}
	rawPercent = strings.TrimSpace(rawPercent)
	if rawPercent == "" {
		return 0, nil
	}
	value, err := parseDecimal(rawPercent)
	if err != nil || value < 0 || value > 100 {
		return 0, fmt.Errorf("invalid quorum")
	}
	return int(value * 10_000), nil
}

func parseBallotReminderBeforeMinutes(rawMinutes string, rawHours string) (int, error) {
	rawMinutes = strings.TrimSpace(rawMinutes)
	if rawMinutes != "" {
		value, err := strconv.Atoi(rawMinutes)
		if err != nil || value <= 0 || value > maxBallotReminderBeforeMinutes {
			return 0, fmt.Errorf("invalid reminder")
		}
		return value, nil
	}
	rawHours = strings.TrimSpace(rawHours)
	if rawHours == "" {
		return defaultBallotReminderBeforeMinutes, nil
	}
	value, err := parseDecimal(rawHours)
	minutes := int(value * 60)
	if err != nil || minutes <= 0 || minutes > maxBallotReminderBeforeMinutes {
		return 0, fmt.Errorf("invalid reminder")
	}
	return minutes, nil
}

func documentFileHeader(r *http.Request) *multipart.FileHeader {
	if r == nil || r.MultipartForm == nil {
		return nil
	}
	for _, name := range []string{"document", "file"} {
		files := r.MultipartForm.File[name]
		if len(files) > 0 {
			return files[0]
		}
	}
	return nil
}

func (a *app) visibleDocumentsForActor(tenantSlug string, email string, role string) []documentRecord {
	if a == nil || a.documentStore == nil {
		return nil
	}
	all := a.documentStore.ListCurrentTenant(tenantSlug)
	out := make([]documentRecord, 0, len(all))
	for _, item := range all {
		if a.canViewDocument(tenantSlug, item, email, role) {
			out = append(out, item)
		}
	}
	return out
}

func (a *app) canViewDocument(tenantSlug string, item documentRecord, email string, role string) bool {
	if normalizeSlug(item.TenantSlug) != normalizeSlug(tenantSlug) {
		return false
	}
	if hasCapability(role, capabilityManageDocuments) {
		return true
	}
	switch normalizeDocumentVisibility(item.Visibility) {
	case documentVisibilityAllResidents:
		return true
	case documentVisibilityOwnersOnly:
		return a.isDocumentOwner(tenantSlug, email, role, item.UnitID)
	case documentVisibilityManagerOnly:
		return false
	default:
		return false
	}
}

func (a *app) isDocumentOwner(tenantSlug string, email string, role string, unitID string) bool {
	tenantSlug = normalizeSlug(tenantSlug)
	email = normalizeEmail(email)
	unitID = normalizeUnitID(unitID)
	if tenantSlug == "" || email == "" {
		return false
	}
	if a != nil && a.unitStore != nil {
		if unitID != "" {
			members := a.unitStore.MembersForUnit(tenantSlug, unitID)
			return members.Found && emailListContains(members.Owners, email)
		}
		for _, membership := range a.unitStore.UnitsForEmail(tenantSlug, email) {
			if membership.Relation == roleOwner {
				return true
			}
		}
	}
	return normalizeRole(role) == roleOwner
}

func (a *app) ballots(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	canManage := hasCapability(role, capabilityManageVotes)
	canOversight := hasCapability(role, capabilityOversight)
	now := time.Now()
	all := []ballot{}
	if a.voteStore != nil {
		if _, err := a.voteStore.CloseExpiredTenant(tenant.Slug, now); err != nil {
			log.Printf("ballot auto-close failed for %s: %v", tenant.Slug, err)
		}
		all = a.voteStore.ListTenant(tenant.Slug)
	}
	visible := make([]ballot, 0, len(all))
	for _, item := range all {
		switch normalizeBallotStatus(item.Status) {
		case ballotStatusOpen, ballotStatusClosed:
			visible = append(visible, item)
		}
	}
	msg, msgOK := voteMessage(r.URL.Query().Get("vote"))
	a.render(w, "ballots", map[string]any{
		"Title":              "Abstimmungen",
		"Tenant":             tenant,
		"Email":              email,
		"DisplayName":        profile.DisplayName(),
		"Initials":           profile.Initials(),
		"Role":               role,
		"IsAdmin":            isAdmin,
		"CanSeeParking":      isAdmin || profile.HasPermission(permissionParking),
		"CanManageVotes":     canManage,
		"CanVote":            hasCapability(role, capabilityVote),
		"CanOversightVotes":  canOversight,
		"ActivePage":         "abstimmungen",
		"Ballots":            a.ballotViewsForActor(tenant.Slug, email, role, visible, now, canManage || canOversight),
		"HasBallots":         len(visible) > 0,
		"BallotCountLabel":   pluralizeCount(len(visible), "Eintrag", "Einträge"),
		"BallotsEmpty":       emptyState("Keine Abstimmungen", "Geöffnete und abgeschlossene Beschlüsse erscheinen hier."),
		"ManageBallots":      a.ballotViewsForActor(tenant.Slug, email, role, all, now, true),
		"HasManageBallots":   len(all) > 0,
		"ManageBallotsEmpty": emptyState("Noch keine Abstimmung", "Neue Entwürfe werden hier angelegt und anschließend geöffnet."),
		"VoteMsg":            msg,
		"VoteOK":             msgOK,
		"NowInput":           formatLocalDateTimeInput(now),
	})
}

func (a *app) submitBallot(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(firstNonEmpty(r.FormValue("ballot_id"), r.FormValue("id"))) != "" || strings.TrimSpace(r.FormValue("option")) != "" {
		a.castVote(w, r)
		return
	}
	a.createBallot(w, r)
}

func (a *app) castVote(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if !hasCapability(role, capabilityVote) {
		http.Error(w, "Dieser Zugang ist für Abstimmungen lesend.", http.StatusForbidden)
		return
	}
	if a.voteStore == nil {
		http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
		return
	}
	id := strings.TrimSpace(firstNonEmpty(r.FormValue("ballot_id"), r.FormValue("id")))
	option := strings.TrimSpace(r.FormValue("option"))
	updated, found, err := a.castBallotVote(tenant.Slug, email, id, option, time.Now())
	if !found {
		http.Redirect(w, r, "/app/abstimmungen?vote=missing", http.StatusSeeOther)
		return
	}
	if err != nil {
		log.Printf("ballot vote failed for %s/%s/%s: %v", tenant.Slug, id, redactedEmail(email), err)
		http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
		return
	}
	if vote, ok := updated.Votes[normalizeEmail(email)]; ok {
		a.recordAudit(auditEvent{
			TenantSlug: tenant.Slug,
			ActorEmail: email,
			ActorRole:  role,
			Action:     auditActionVoteCast,
			TargetType: "ballot",
			TargetID:   updated.ID,
			Summary:    "Stimme gespeichert",
			Details: map[string]string{
				"title":   updated.Title,
				"weight":  formatBallotResultWeight(updated.Weighting, vote.Weight),
				"cast_at": formatLocalDateTime(vote.At),
			},
		})
	}
	http.Redirect(w, r, "/app/abstimmungen?vote=cast", http.StatusSeeOther)
}

func (a *app) ballotProtocol(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !hasCapability(role, capabilityVote) && !hasCapability(role, capabilityOversight) && !hasCapability(role, capabilityManageVotes) {
		http.Error(w, "Dieses Protokoll ist für diesen Zugang nicht freigegeben.", http.StatusForbidden)
		return
	}
	if a.voteStore == nil {
		http.NotFound(w, r)
		return
	}
	now := time.Now()
	if _, err := a.voteStore.CloseExpiredTenant(tenant.Slug, now); err != nil {
		log.Printf("ballot auto-close failed for %s: %v", tenant.Slug, err)
	}
	id := strings.TrimSpace(r.PathValue("id"))
	item, found := a.voteStore.Get(tenant.Slug, id)
	if !found {
		http.NotFound(w, r)
		return
	}
	if normalizeBallotStatus(item.Status) != ballotStatusClosed {
		http.Error(w, "Protokoll erst nach Abschluss verfügbar.", http.StatusConflict)
		return
	}
	view := a.ballotViewForActor(tenant.Slug, email, role, item, now, true)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": "abstimmung-" + item.ID + "-protokoll.html"}))
	if err := a.templates.ExecuteTemplate(w, "ballotProtocol", map[string]any{
		"Title":       "Abstimmungsprotokoll",
		"Tenant":      tenant,
		"Ballot":      view,
		"GeneratedAt": formatLocalDateTime(now),
		"AppVersion":  buildLabel(),
	}); err != nil {
		log.Printf("render ballotProtocol failed: %v", err)
	}
}

func (a *app) castBallotVote(tenantSlug string, email string, ballotID string, option string, at time.Time) (ballot, bool, error) {
	if a == nil || a.voteStore == nil {
		return ballot{}, false, nil
	}
	item, found := a.voteStore.Get(tenantSlug, ballotID)
	if !found {
		return ballot{}, false, nil
	}
	weight, eligible := a.ballotVoteWeight(tenantSlug, email, item)
	if !eligible {
		return ballot{}, true, fmt.Errorf("not eligible to vote")
	}
	return a.voteStore.CastVote(tenantSlug, ballotID, email, option, weight, at)
}

func (a *app) ballotVoteWeight(tenantSlug string, email string, item ballot) (int, bool) {
	if a == nil {
		return 0, false
	}
	tenantSlug = normalizeSlug(tenantSlug)
	email = normalizeEmail(email)
	if tenantSlug == "" || email == "" {
		return 0, false
	}
	ownerShare := 0
	if a != nil && a.unitStore != nil {
		for _, membership := range a.unitStore.UnitsForEmail(tenantSlug, email) {
			if membership.Relation == roleOwner {
				ownerShare += membership.Unit.MiteigentumsanteilPPM
			}
		}
	}
	isOwner := ownerShare > 0 || normalizeRole(a.roleFor(email, tenantSlug)) == roleOwner
	switch normalizeBallotWeighting(item.Weighting) {
	case ballotWeightingPerHead:
		if isOwner {
			return 1, true
		}
	case ballotWeightingPerShare:
		if ownerShare > 0 {
			return ownerShare, true
		}
	}
	return 0, false
}

func (a *app) ballotViewsForActor(tenantSlug string, email string, role string, items []ballot, now time.Time, includeResults bool) []ballotView {
	out := make([]ballotView, 0, len(items))
	for _, item := range items {
		out = append(out, a.ballotViewForActor(tenantSlug, email, role, item, now, includeResults))
	}
	return out
}

func (a *app) ballotViewForActor(tenantSlug string, email string, role string, item ballot, now time.Time, includeResults bool) ballotView {
	item = normalizeBallot(item)
	weight, eligible := a.ballotVoteWeight(tenantSlug, email, item)
	status, statusClass, active := ballotStatusForView(item, now)
	vote, hasVote := item.Votes[normalizeEmail(email)]
	rawStatus := normalizeBallotStatus(item.Status)
	tally := a.computeBallotTally(tenantSlug, item)
	view := ballotView{
		ID:                  item.ID,
		Title:               item.Title,
		Description:         item.Description,
		HasDescription:      item.Description != "",
		Type:                item.Type,
		Weighting:           ballotWeightingLabel(item.Weighting),
		Quorum:              formatPPMPercent(item.QuorumPPM),
		HasQuorum:           item.QuorumPPM > 0,
		ReminderLabel:       formatBallotReminder(item.ReminderBeforeMinutes),
		Status:              status,
		StatusClass:         statusClass,
		CreatedAt:           formatLocalDateTime(item.CreatedAt),
		UpdatedAt:           formatLocalDateTime(item.UpdatedAt),
		CanManage:           hasCapability(role, capabilityManageVotes),
		CanVote:             hasCapability(role, capabilityVote) && eligible && active,
		CanOpen:             rawStatus == ballotStatusDraft,
		CanClose:            rawStatus == ballotStatusOpen,
		HasVote:             hasVote,
		VoteOption:          vote.Option,
		VoteWeight:          formatBallotWeight(weight),
		ReadOnlyMessage:     ballotReadOnlyMessage(role, item, eligible, active, hasVote),
		TotalVotes:          tally.TotalVotes,
		TotalWeight:         tally.TotalWeight,
		TotalWeightLabel:    tally.TotalWeightLabel,
		EligibleWeightLabel: tally.EligibleWeightLabel,
		Participation:       tally.ParticipationLabel,
		QuorumStatus:        tally.QuorumStatus,
		QuorumClass:         tally.QuorumClass,
		WinnerLabel:         tally.WinnerLabel,
		HasWinner:           tally.WinnerLabel != "",
		ProtocolURL:         "/app/abstimmungen/" + url.PathEscape(item.ID) + "/protokoll",
		HasProtocol:         rawStatus == ballotStatusClosed && (hasCapability(role, capabilityVote) || hasCapability(role, capabilityOversight) || hasCapability(role, capabilityManageVotes)),
		EditDialogID:        "ballot-" + item.ID,
	}
	if !item.OpensAt.IsZero() {
		view.OpensAt = formatLocalDateTime(item.OpensAt)
		view.HasOpensAt = true
	}
	if !item.ClosesAt.IsZero() {
		view.ClosesAt = formatLocalDateTime(item.ClosesAt)
		view.HasClosesAt = true
	}
	if hasVote {
		view.VotedAt = formatLocalDateTime(vote.At)
		view.VoteWeight = formatBallotWeight(vote.Weight)
	}
	if includeResults || rawStatus == ballotStatusClosed {
		view.HasResults = true
	}
	for _, option := range item.Options {
		result := tally.Options[option]
		view.Options = append(view.Options, ballotOptionView{
			Value:        option,
			Label:        option,
			Selected:     hasVote && vote.Option == option,
			VoteCount:    result.Count,
			Weight:       result.Weight,
			WeightLabel:  formatBallotResultWeight(item.Weighting, result.Weight),
			Percent:      result.Percent,
			PercentStyle: strconv.Itoa(result.Percent),
		})
	}
	return view
}

type ballotResultCount struct {
	Count   int
	Weight  int
	Percent int
}

type ballotResultSummary struct {
	Options             map[string]ballotResultCount
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

func (a *app) computeBallotTally(tenantSlug string, item ballot) ballotResultSummary {
	out := ballotResultSummary{Options: map[string]ballotResultCount{}}
	item = normalizeBallot(item)
	for _, vote := range item.Votes {
		if !ballotHasOption(item, vote.Option) || vote.Weight <= 0 {
			continue
		}
		result := out.Options[vote.Option]
		result.Count++
		result.Weight += vote.Weight
		out.Options[vote.Option] = result
		out.TotalVotes++
		out.TotalWeight += vote.Weight
	}
	if out.TotalWeight > 0 {
		for option, result := range out.Options {
			result.Percent = (result.Weight*100 + out.TotalWeight/2) / out.TotalWeight
			out.Options[option] = result
		}
	}
	out.EligibleWeight = a.ballotEligibleWeightTotal(tenantSlug, item)
	if out.EligibleWeight < out.TotalWeight {
		out.EligibleWeight = out.TotalWeight
	}
	if out.EligibleWeight > 0 {
		out.ParticipationPPM = (out.TotalWeight*1_000_000 + out.EligibleWeight/2) / out.EligibleWeight
	}
	out.TotalWeightLabel = formatBallotResultWeight(item.Weighting, out.TotalWeight)
	out.EligibleWeightLabel = formatBallotResultWeight(item.Weighting, out.EligibleWeight)
	out.ParticipationLabel = formatPPMPercent(out.ParticipationPPM)
	out.QuorumReached = item.QuorumPPM == 0 || out.ParticipationPPM >= item.QuorumPPM
	if item.QuorumPPM == 0 {
		out.QuorumStatus = "Kein Quorum"
		out.QuorumClass = "ok"
	} else if out.QuorumReached {
		out.QuorumStatus = "Quorum erreicht"
		out.QuorumClass = "ok"
	} else {
		out.QuorumStatus = "Quorum offen"
		out.QuorumClass = "status-open"
	}
	out.WinnerLabel = ballotWinnerLabel(out.Options)
	return out
}

func (a *app) ballotEligibleWeightTotal(tenantSlug string, item ballot) int {
	total := 0
	for _, weight := range a.ballotEligibleWeights(tenantSlug, item) {
		total += weight
	}
	return total
}

func (a *app) ballotEligibleWeights(tenantSlug string, item ballot) map[string]int {
	weights := map[string]int{}
	tenantSlug = normalizeSlug(tenantSlug)
	switch normalizeBallotWeighting(item.Weighting) {
	case ballotWeightingPerHead:
		if a != nil && a.unitStore != nil {
			for _, unit := range a.unitStore.ListTenant(tenantSlug) {
				for _, email := range unit.OwnerEmails {
					if email = normalizeEmail(email); email != "" {
						weights[email] = 1
					}
				}
			}
		}
		if a != nil {
			for _, row := range a.userRows(tenantSlug) {
				email := normalizeEmail(row.Email)
				if email != "" && normalizeRole(row.Role) == roleOwner {
					weights[email] = 1
				}
			}
		}
	case ballotWeightingPerShare:
		if a != nil && a.unitStore != nil {
			for _, unit := range a.unitStore.ListTenant(tenantSlug) {
				if unit.MiteigentumsanteilPPM <= 0 {
					continue
				}
				for _, email := range unit.OwnerEmails {
					if email = normalizeEmail(email); email != "" {
						weights[email] += unit.MiteigentumsanteilPPM
					}
				}
			}
		}
	}
	return weights
}

func ballotWinnerLabel(options map[string]ballotResultCount) string {
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

func ballotStatusForView(item ballot, now time.Time) (string, string, bool) {
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()
	switch normalizeBallotStatus(item.Status) {
	case ballotStatusOpen:
		if !item.OpensAt.IsZero() && now.Before(item.OpensAt) {
			return "Geplant", "status-progress", false
		}
		if !item.ClosesAt.IsZero() && !now.Before(item.ClosesAt) {
			return "Frist abgelaufen", "status-closed", false
		}
		return ballotStatusOpen, "status-open", true
	case ballotStatusClosed:
		return ballotStatusClosed, "status-closed", false
	default:
		return ballotStatusDraft, "status-progress", false
	}
}

func ballotReadOnlyMessage(role string, item ballot, eligible bool, active bool, hasVote bool) string {
	if !active {
		switch normalizeBallotStatus(item.Status) {
		case ballotStatusDraft:
			return "Noch nicht geöffnet."
		case ballotStatusClosed:
			return "Abstimmung geschlossen."
		default:
			if !item.OpensAt.IsZero() && time.Now().UTC().Before(item.OpensAt) {
				return "Noch nicht gestartet."
			}
			if !item.ClosesAt.IsZero() && !time.Now().UTC().Before(item.ClosesAt) {
				return "Frist abgelaufen."
			}
		}
	}
	if hasCapability(role, capabilityVote) {
		if eligible {
			if hasVote {
				return "Stimme abgegeben; bis zur Schließung änderbar."
			}
			return ""
		}
		return "Kein Stimmgewicht für diesen Zugang hinterlegt."
	}
	if hasCapability(role, capabilityOversight) {
		return "Beirat: lesende Übersicht."
	}
	return "Nur Eigentümer können abstimmen."
}

func formatBallotWeight(weight int) string {
	if weight <= 0 {
		return ""
	}
	if weight == 1 {
		return "1 Stimme"
	}
	return formatMiteigentumsanteil(weight)
}

func formatBallotResultWeight(weighting string, weight int) string {
	if weight <= 0 {
		return "0"
	}
	if normalizeBallotWeighting(weighting) == ballotWeightingPerHead {
		if weight == 1 {
			return "1 Stimme"
		}
		return strconv.Itoa(weight) + " Stimmen"
	}
	return formatMiteigentumsanteil(weight)
}

func formatPPMPercent(ppm int) string {
	if ppm < 0 {
		ppm = 0
	}
	if ppm > 1_000_000 {
		ppm = 1_000_000
	}
	return formatDecimal(float64(ppm)/10_000, 1) + " %"
}

func formatBallotReminder(minutes int) string {
	if minutes <= 0 {
		minutes = defaultBallotReminderBeforeMinutes
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

func voteMessage(status string) (string, bool) {
	switch status {
	case "created":
		return "Abstimmung angelegt.", true
	case "opened":
		return "Abstimmung geöffnet.", true
	case "closed":
		return "Abstimmung geschlossen.", true
	case "cast":
		return "Stimme gespeichert.", true
	case "missing":
		return "Abstimmung nicht gefunden.", false
	case "invalid":
		return "Bitte Abstimmung, Option und Berechtigung prüfen.", false
	default:
		return "", false
	}
}

func documentMessage(status string) (string, bool) {
	switch status {
	case "uploaded":
		return "Dokument hochgeladen.", true
	case "replaced":
		return "Neue Version gespeichert.", true
	case "invalid":
		return "Bitte Titel, Kategorie, Sichtbarkeit und Datei prüfen. Erlaubt sind PDF, JPG, PNG oder WebP bis 20 MB.", false
	default:
		return "", false
	}
}

func (a *app) portal(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	canManage := canManageAnnouncements(role)
	announcements := []announcementView{}
	now := time.Now()
	lastSeen := time.Time{}
	if a.announcementReadStore != nil {
		lastSeen = a.announcementReadStore.LastSeen(tenant.Slug, email)
	}
	if a.announcementStore != nil {
		announcements = announcementViewsWithReadState(a.announcementStore.Visible(tenant.Slug, now), now, false, lastSeen)
		if len(announcements) > 3 {
			announcements = announcements[:3]
		}
	}
	events := []houseEventView{}
	if a.eventStore != nil {
		events = eventViews(a.eventStore.Upcoming(tenant.Slug, now), now)
		if len(events) > 4 {
			events = events[:4]
		}
	}
	digest := a.dashboardDigestItems(tenant.Slug, email, role, now, lastSeen)
	a.render(w, "portal", map[string]any{
		"Title":                  "WEG Portal",
		"Tenant":                 tenant,
		"Email":                  email,
		"DisplayName":            profile.DisplayName(),
		"Initials":               profile.Initials(),
		"Role":                   role,
		"IsAdmin":                isAdmin,
		"CanSeeParking":          isAdmin || profile.HasPermission(permissionParking),
		"CanManageAnnouncements": canManage,
		"CanManageEvents":        canManageEvents(role),
		"ActivePage":             "home",
		"Digest":                 digest,
		"HasDigest":              len(digest) > 0,
		"DigestEmpty":            emptyState("Nichts Neues", "Aktuell gibt es keine ungelesenen Aushänge, offenen Anliegen oder anstehenden Termine."),
		"Announcements":          announcements,
		"HasAnnouncements":       len(announcements) > 0,
		"AnnouncementsEmpty":     emptyStateAction("Noch keine Beiträge", "Sobald die Verwaltung einen Aushang veröffentlicht, erscheint er hier.", "/app/announcements", "Archiv öffnen"),
		"Events":                 events,
		"HasEvents":              len(events) > 0,
		"EventsEmpty":            emptyStateAction("Noch keine kommenden Termine", "Geplante Versammlungen, Wartungen und Fristen erscheinen hier.", "/app/events", "Termine öffnen"),
	})
}

func (a *app) dashboardDigestItems(tenantSlug string, email string, role string, now time.Time, lastSeen time.Time) []dashboardDigestItem {
	items := []dashboardDigestItem{}
	if a.announcementStore != nil {
		unread := unreadAnnouncementCount(a.announcementStore.Visible(tenantSlug, now), lastSeen, now)
		if unread > 0 {
			items = append(items, dashboardDigestItem{
				Title:  "Neue Aushänge",
				Detail: pluralizeCount(unread, "ungelesener Beitrag", "ungelesene Beiträge"),
				URL:    "/app/announcements",
				Badge:  strconv.Itoa(unread),
			})
		}
	}
	if a.issueStore != nil {
		open := issueOpenCount(a.visibleIssuesForActor(tenantSlug, email, role))
		if open > 0 {
			url := "/app/anliegen"
			title := "Offene Anliegen"
			detail := pluralizeCount(open, "offenes Anliegen", "offene Anliegen")
			if hasCapability(role, capabilityManageIssues) {
				url = "/app/anliegen/board"
				title = "Offene Anliegen im Haus"
			}
			items = append(items, dashboardDigestItem{
				Title:  title,
				Detail: detail,
				URL:    url,
				Badge:  strconv.Itoa(open),
			})
		}
	}
	if a.eventStore != nil {
		upcoming := a.eventStore.Upcoming(tenantSlug, now)
		if len(upcoming) > 0 {
			items = append(items, dashboardDigestItem{
				Title:  "Kommende Termine",
				Detail: pluralizeCount(len(upcoming), "Termin geplant", "Termine geplant"),
				URL:    "/app/events",
				Badge:  strconv.Itoa(len(upcoming)),
			})
		}
	}
	return items
}

func pluralizeCount(count int, singular string, plural string) string {
	if count == 1 {
		return "1 " + singular
	}
	return strconv.Itoa(count) + " " + plural
}

func (a *app) contacts(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	managerContacts := managerContactViews(tenant)
	emergencyContacts := emergencyContactViews(tenant)
	boardContacts := a.boardContactViews(tenant.Slug)
	residentContacts := a.residentDirectoryViews(tenant.Slug)
	a.render(w, "contacts", map[string]any{
		"Title":                "Kontakte",
		"Tenant":               tenant,
		"Email":                email,
		"DisplayName":          profile.DisplayName(),
		"Initials":             profile.Initials(),
		"Role":                 role,
		"IsAdmin":              isAdmin,
		"CanSeeParking":        isAdmin || profile.HasPermission(permissionParking),
		"ActivePage":           "contacts",
		"ManagerContacts":      managerContacts,
		"HasManagerContacts":   len(managerContacts) > 0,
		"ManagerEmpty":         emptyState("Kein Verwaltungskontakt", "Der Kontaktblock wird in den Gebäude-Einstellungen gepflegt."),
		"EmergencyContacts":    emergencyContacts,
		"HasEmergencyContacts": len(emergencyContacts) > 0,
		"EmergencyEmpty":       emptyState("Kein Notdienst hinterlegt", "Notdienst und Hausmeister werden in den Gebäude-Einstellungen gepflegt."),
		"BoardContacts":        boardContacts,
		"HasBoardContacts":     len(boardContacts) > 0,
		"BoardEmpty":           emptyState("Kein Beirat hinterlegt", "Beiräte erscheinen hier, sobald sie in Benutzer & Rechte die Beirat-Rolle haben."),
		"ResidentContacts":     residentContacts,
		"HasResidentContacts":  len(residentContacts) > 0,
		"ResidentEmpty":        emptyState("Keine freigegebenen Kontakte", "Kontakte aus der Hausgemeinschaft erscheinen nur nach ausdrücklicher Freigabe im Profil."),
	})
}

func managerContactViews(tenant tenantConfig) []contactCardView {
	contact := contactCardView{
		Name:        firstNonEmpty(tenant.ContactName, tenant.Name, "Hausverwaltung"),
		Role:        roleManager,
		Description: "Verwaltung",
		Email:       tenant.ContactEmail,
		Phone:       tenant.ContactPhone,
	}
	if contact.Email == "" && contact.Phone == "" && strings.TrimSpace(tenant.ContactName) == "" {
		return nil
	}
	contact.HasEmail = contact.Email != ""
	contact.HasPhone = contact.Phone != ""
	return []contactCardView{contact}
}

func emergencyContactViews(tenant tenantConfig) []contactCardView {
	contacts := []contactCardView{}
	if tenant.EmergencyName != "" || tenant.EmergencyPhone != "" {
		contacts = append(contacts, contactCardView{
			Name:        firstNonEmpty(tenant.EmergencyName, "Notdienst"),
			Role:        "Notdienst",
			Description: "Dringende Fälle außerhalb der regulären Verwaltung",
			Phone:       tenant.EmergencyPhone,
			HasPhone:    tenant.EmergencyPhone != "",
		})
	}
	if tenant.CaretakerName != "" || tenant.CaretakerEmail != "" || tenant.CaretakerPhone != "" {
		contacts = append(contacts, contactCardView{
			Name:        firstNonEmpty(tenant.CaretakerName, "Hausmeister"),
			Role:        "Hausmeister",
			Description: "Operativer Kontakt im Haus",
			Email:       tenant.CaretakerEmail,
			Phone:       tenant.CaretakerPhone,
			HasEmail:    tenant.CaretakerEmail != "",
			HasPhone:    tenant.CaretakerPhone != "",
		})
	}
	return contacts
}

func (a *app) boardContactViews(tenantSlug string) []contactCardView {
	contacts := []contactCardView{}
	for _, row := range a.userRows(tenantSlug) {
		if row.Role != roleBeirat || normalizeEmail(row.Email) == "" {
			continue
		}
		contacts = append(contacts, contactCardView{
			Name:        row.DisplayName,
			Role:        roleBeirat,
			Description: "Beirat",
			Email:       row.Email,
			Phone:       row.Phone,
			HasEmail:    row.Email != "",
			HasPhone:    row.Phone != "",
		})
	}
	return contacts
}

func (a *app) residentDirectoryViews(tenantSlug string) []contactCardView {
	contacts := []contactCardView{}
	for _, row := range a.userRows(tenantSlug) {
		if !row.DirectoryOptIn || !residentDirectoryRole(row.Role) || normalizeEmail(row.Email) == "" {
			continue
		}
		contacts = append(contacts, contactCardView{
			Name:        row.DisplayName,
			Role:        row.Role,
			Description: "Hausgemeinschaft",
			Email:       row.Email,
			Phone:       row.Phone,
			HasEmail:    row.Email != "",
			HasPhone:    row.Phone != "",
		})
	}
	return contacts
}

func residentDirectoryRole(role string) bool {
	switch normalizeRole(role) {
	case roleOwner, roleRenter, roleResident:
		return true
	default:
		return false
	}
}

func (a *app) issues(w http.ResponseWriter, r *http.Request) {
	a.renderIssuesPage(w, r, false)
}

func (a *app) issueBoard(w http.ResponseWriter, r *http.Request) {
	a.renderIssuesPage(w, r, true)
}

func (a *app) renderIssuesPage(w http.ResponseWriter, r *http.Request, boardOnly bool) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	issues := []issueView{}
	manageIssues := []issueView{}
	canManageIssues := hasCapability(role, capabilityManageIssues)
	canCreateIssue := !hasCapability(role, capabilityOversight) || canManageIssues
	if boardOnly && !canManageIssues {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	filters := issueBoardFiltersFromQuery(r.URL.Query())
	if a.issueStore != nil {
		if !boardOnly {
			if canManageIssues {
				issues = issueViewsForActor(a.issueStore.ListAuthor(tenant.Slug, email), role, email)
			} else {
				issues = issueViewsForActor(a.visibleIssuesForActor(tenant.Slug, email, role), role, email)
			}
		}
		if canManageIssues {
			manageIssues = issueViewsForActor(filterIssueBoard(a.issueStore.ListTenant(tenant.Slug), filters), role, email)
		}
	}
	msg, msgOK := issueMessage(r.URL.Query().Get("issue"))
	a.render(w, "issues", map[string]any{
		"Title":                  "Anliegen",
		"Tenant":                 tenant,
		"Email":                  email,
		"DisplayName":            profile.DisplayName(),
		"Initials":               profile.Initials(),
		"Role":                   role,
		"IsAdmin":                isAdmin,
		"CanSeeParking":          isAdmin || profile.HasPermission(permissionParking),
		"CanManageAnnouncements": canManageAnnouncements(role),
		"CanManageIssues":        canManageIssues,
		"CanCreateIssue":         canCreateIssue,
		"ActivePage":             "issues",
		"BoardOnly":              boardOnly,
		"BoardAction":            issueBoardAction(boardOnly),
		"BoardFilters":           issueBoardFilterOptions(filters),
		"Issues":                 issues,
		"HasIssues":              len(issues) > 0,
		"IssuesEmpty":            emptyState("Noch kein Anliegen", "Nach dem Absenden erscheint das Anliegen hier mit Status und Rückfragen."),
		"ManageIssues":           manageIssues,
		"HasManageIssues":        len(manageIssues) > 0,
		"ManageIssuesEmpty":      emptyState("Keine Anliegen im Haus", "Sobald ein Anliegen gemeldet wird, erscheint es hier für die Bearbeitung."),
		"IssueMsg":               msg,
		"IssueOK":                msgOK,
	})
}

func issueMessage(status string) (string, bool) {
	switch status {
	case "created":
		return "Anliegen gespeichert. Die Verwaltung sieht es im nächsten Bearbeitungsschritt.", true
	case "updated":
		return "Anliegen aktualisiert.", true
	case "invalid":
		return "Bitte Kategorie, Ort, Titel und Beschreibung prüfen.", false
	case "photo":
		return "Das Foto konnte nicht übernommen werden. Erlaubt sind JPG, PNG oder WebP bis 5 MB.", false
	case "missing":
		return "Dieses Anliegen wurde nicht gefunden.", false
	case "error":
		return "Das Anliegen konnte nicht gespeichert werden.", false
	default:
		return "", false
	}
}

func (a *app) createIssue(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, _, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxIssueFormBytes)
	if err := r.ParseMultipartForm(maxIssuePhotoBytes); err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	item, err := issueFromForm(r, tenant.Slug, profile, time.Now())
	if err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	if a.issueStore != nil {
		photo, hasPhoto := issuePhotoHeader(r)
		if hasPhoto {
			photoPath, err := a.issueStore.SavePhoto(tenant.Slug, item.ID, photo)
			if err != nil {
				http.Redirect(w, r, "/app/anliegen?issue=photo", http.StatusSeeOther)
				return
			}
			item.PhotoPaths = []string{photoPath}
		}
		created, err := a.issueStore.Create(item)
		if err != nil {
			log.Printf("issue create failed for %s/%s: %v", tenant.Slug, redactedEmail(email), err)
			http.Redirect(w, r, "/app/anliegen?issue=error", http.StatusSeeOther)
			return
		}
		a.notifyIssueCreated(tenant, created)
	}
	http.Redirect(w, r, "/app/anliegen?issue=created", http.StatusSeeOther)
}

func (a *app) addIssueComment(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	existing, found := a.issueStore.Get(tenant.Slug, id)
	if !found {
		http.Redirect(w, r, "/app/anliegen?issue=missing", http.StatusSeeOther)
		return
	}
	canManage := hasCapability(role, capabilityManageIssues)
	readOnly := hasCapability(role, capabilityOversight) && !canManage
	if !a.canViewIssueForActor(tenant.Slug, existing, email, role) {
		http.Error(w, "Dieser Kommentar ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	isOwner := normalizeEmail(existing.AuthorEmail) == normalizeEmail(email)
	if readOnly || (!canManage && !isOwner) {
		http.Error(w, "Dieser Kommentar ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	body := strings.TrimSpace(r.FormValue("body"))
	if body == "" || len([]rune(body)) > 3000 {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	updated, ok, err := a.issueStore.AddComment(tenant.Slug, id, issueComment{
		AuthorEmail: email,
		AuthorName:  profile.DisplayName(),
		Body:        body,
		CreatedAt:   time.Now(),
	})
	if err != nil {
		log.Printf("issue comment failed for %s/%s: %v", tenant.Slug, id, err)
		http.Redirect(w, r, "/app/anliegen?issue=error", http.StatusSeeOther)
		return
	}
	if !ok {
		http.Redirect(w, r, "/app/anliegen?issue=missing", http.StatusSeeOther)
		return
	}
	a.notifyIssueUpdated(tenant, updated, email, "Neuer Kommentar zu Anliegen \""+updated.Title+"\"")
	http.Redirect(w, r, "/app/anliegen?issue=updated", http.StatusSeeOther)
}

func (a *app) updateIssueWorkflow(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	existing, found := a.issueStore.Get(tenant.Slug, id)
	if !found {
		http.Redirect(w, r, "/app/anliegen?issue=missing", http.StatusSeeOther)
		return
	}

	canManage := hasCapability(role, capabilityManageIssues)
	readOnly := hasCapability(role, capabilityOversight) && !canManage
	if !a.canViewIssueForActor(tenant.Slug, existing, email, role) {
		http.Error(w, "Dieser Statuswechsel ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	isOwner := normalizeEmail(existing.AuthorEmail) == normalizeEmail(email)
	status := normalizeIssueStatus(r.FormValue("status"))
	if status == "" {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	priority := normalizeIssuePriority(r.FormValue("priority"))
	assignee := normalizeEmail(r.FormValue("assignee_email"))
	if !canManage {
		if readOnly || !isOwner || r.FormValue("priority") != "" || r.FormValue("assignee_email") != "" || !canResidentTransition(existing.Status, status) {
			http.Error(w, "Dieser Statuswechsel ist der Verwaltung vorbehalten.", http.StatusForbidden)
			return
		}
		priority = normalizeIssuePriority(existing.Priority)
		assignee = normalizeEmail(existing.AssigneeEmail)
	}
	if priority == "" {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	updated, _, err := a.issueStore.UpdateWorkflow(tenant.Slug, id, issueWorkflowUpdate{
		Status:        status,
		Priority:      priority,
		AssigneeEmail: assignee,
		ActorEmail:    email,
		ActorName:     profile.DisplayName(),
		ChangedAt:     time.Now(),
	})
	if err != nil {
		log.Printf("issue workflow update failed for %s/%s: %v", tenant.Slug, id, err)
		http.Redirect(w, r, "/app/anliegen?issue=error", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     auditActionIssueWorkflow,
		TargetType: "issue",
		TargetID:   updated.ID,
		Summary:    "Anliegen-Workflow geändert",
		Details: map[string]string{
			"status":   normalizeIssueStatus(updated.Status),
			"priority": normalizeIssuePriority(updated.Priority),
		},
	})
	a.notifyIssueUpdated(tenant, updated, email, "Anliegen \""+updated.Title+"\" aktualisiert")
	http.Redirect(w, r, "/app/anliegen?issue=updated", http.StatusSeeOther)
}

func (a *app) notifyIssueCreated(tenant tenantConfig, issue residentIssue) {
	recipients := a.issueManagerEmails(tenant.Slug)
	a.notify(portalNotification{
		Event:      notificationEventIssue,
		Tenant:     tenant,
		Recipients: recipients,
		ActorEmail: issue.AuthorEmail,
		Subject:    "Neues Anliegen: " + issue.Title,
		Lines: []string{
			"Es wurde ein neues Anliegen für " + tenant.Address + " erfasst.",
			"",
			issue.Title,
			issue.Category + " · " + issueLocationLabel(issue.LocationType, issue.LocationDetail),
			"",
			issue.Body,
		},
	})
}

func (a *app) notifyIssueUpdated(tenant tenantConfig, issue residentIssue, actorEmail string, subject string) {
	recipients := []string{issue.AuthorEmail, issue.AssigneeEmail}
	a.notify(portalNotification{
		Event:      notificationEventIssue,
		Tenant:     tenant,
		Recipients: recipients,
		ActorEmail: actorEmail,
		Subject:    subject,
		Lines: []string{
			"Ein Anliegen für " + tenant.Address + " wurde aktualisiert.",
			"",
			issue.Title,
			"Status: " + normalizeIssueStatus(issue.Status),
			"Priorität: " + normalizeIssuePriority(issue.Priority),
		},
	})
}

func (a *app) notifyAnnouncementPublished(tenant tenantConfig, item announcement, actorEmail string) {
	now := time.Now()
	if item.PublishedAt.After(now) || (item.ExpiresAt != nil && !item.ExpiresAt.After(now)) {
		return
	}
	a.notify(portalNotification{
		Event:      notificationEventAnnouncement,
		Tenant:     tenant,
		Recipients: a.tenantNotificationEmails(tenant.Slug),
		ActorEmail: actorEmail,
		Subject:    "Neuer Aushang: " + item.Title,
		Lines: []string{
			"Für " + tenant.Address + " wurde ein neuer Aushang veröffentlicht.",
			"",
			item.Title,
			"Kategorie: " + item.Category,
			"",
			item.Body,
		},
	})
}

func (a *app) startVoteReminderWorker() func() {
	if a.voteReminderInterval <= 0 || a.voteStore == nil {
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		a.sendDueBallotReminders(time.Now())
		ticker := time.NewTicker(a.voteReminderInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.sendDueBallotReminders(time.Now())
			}
		}
	}()
	return cancel
}

func (a *app) sendDueBallotReminders(now time.Time) int {
	if a == nil || a.voteStore == nil {
		return 0
	}
	if now.IsZero() {
		now = time.Now()
	}
	sentTotal := 0
	for _, tenant := range a.tenants {
		if _, err := a.voteStore.CloseExpiredTenant(tenant.Slug, now); err != nil {
			log.Printf("ballot auto-close failed for %s: %v", tenant.Slug, err)
		}
		for _, item := range a.voteStore.ListTenant(tenant.Slug) {
			if !ballotReminderDue(item, now) {
				continue
			}
			recipients := a.ballotReminderRecipients(tenant.Slug, item)
			if len(recipients) == 0 {
				continue
			}
			event := portalNotification{
				Event:      notificationEventVote,
				Tenant:     tenant,
				Recipients: recipients,
				Subject:    "Erinnerung: Abstimmung " + item.Title,
				ActionText: "Abstimmung öffnen",
				ActionURL:  tenant.PublicURL("/app/abstimmungen"),
				Lines: []string{
					"Für " + tenant.Address + " läuft eine Abstimmung demnächst ab.",
					"",
					item.Title,
					"Frist: " + formatLocalDateTime(item.ClosesAt),
				},
			}
			sent := a.notify(event)
			if len(sent) == 0 {
				continue
			}
			if _, _, err := a.voteStore.MarkReminderSent(tenant.Slug, item.ID, sent, now); err != nil {
				log.Printf("ballot reminder mark failed for %s/%s: %v", tenant.Slug, item.ID, err)
				continue
			}
			a.recordAudit(auditEvent{
				TenantSlug: tenant.Slug,
				ActorEmail: "system",
				ActorRole:  "System",
				Action:     auditActionVoteReminder,
				TargetType: "ballot",
				TargetID:   item.ID,
				Summary:    "Abstimmungs-Erinnerung gesendet",
				Details: map[string]string{
					"title":      item.Title,
					"recipients": strconv.Itoa(len(sent)),
					"deadline":   formatLocalDateTime(item.ClosesAt),
				},
			})
			sentTotal += len(sent)
		}
	}
	return sentTotal
}

func ballotReminderDue(item ballot, now time.Time) bool {
	item = normalizeBallot(item)
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()
	if item.Status != ballotStatusOpen || item.ClosesAt.IsZero() || !now.Before(item.ClosesAt) {
		return false
	}
	return item.ClosesAt.Sub(now) <= time.Duration(item.ReminderBeforeMinutes)*time.Minute
}

func (a *app) ballotReminderRecipients(tenantSlug string, item ballot) []string {
	weights := a.ballotEligibleWeights(tenantSlug, item)
	recipients := make([]string, 0, len(weights))
	for email := range weights {
		if _, voted := item.Votes[email]; voted {
			continue
		}
		if _, reminded := item.ReminderSentAt[email]; reminded {
			continue
		}
		recipients = append(recipients, email)
	}
	sort.Strings(recipients)
	return recipients
}

func (a *app) notify(event portalNotification) []string {
	if a.mailer == nil || !a.mailer.Configured() {
		return nil
	}
	event.Event = normalizeNotificationEvent(event.Event)
	if event.Event == "" {
		return nil
	}
	body := event.Body()
	sent := []string{}
	for _, recipient := range a.notificationRecipients(event) {
		if err := a.mailer.SendNotification(recipient, event.Subject, body); err != nil {
			log.Printf("notification delivery failed for %s: %v", redactedEmail(recipient), err)
			continue
		}
		sent = append(sent, recipient)
	}
	return sent
}

func (a *app) notificationRecipients(event portalNotification) []string {
	event.Event = normalizeNotificationEvent(event.Event)
	if event.Event == "" {
		return nil
	}
	out := []string{}
	for _, recipient := range excludeEmail(uniqueEmails(event.Recipients), event.ActorEmail) {
		if recipient == "" {
			continue
		}
		if a.notificationPrefs != nil && !a.notificationPrefs.EmailEnabled(recipient, event.Event) {
			continue
		}
		out = append(out, recipient)
	}
	return out
}

func (event portalNotification) Body() string {
	lines := append([]string(nil), event.Lines...)
	if event.ActionURL != "" {
		actionText := strings.TrimSpace(event.ActionText)
		if actionText == "" {
			actionText = "Öffnen"
		}
		lines = append(lines, "", actionText+": "+event.ActionURL)
	}
	return strings.Join(lines, "\n")
}

func (a *app) issueManagerEmails(tenantSlug string) []string {
	tenantSlug = normalizeSlug(tenantSlug)
	recipients := []string{}
	for email, profile := range a.profiles {
		if profile.HasTenant(tenantSlug) && hasCapability(profile.ForTenant(tenantSlug).Role, capabilityManageIssues) {
			recipients = append(recipients, email)
		}
	}
	for email := range a.admins {
		recipients = append(recipients, email)
	}
	if a.inviteStore != nil {
		for _, profile := range a.inviteStore.List() {
			if profile.HasTenant(tenantSlug) && hasCapability(profile.ForTenant(tenantSlug).Role, capabilityManageIssues) {
				recipients = append(recipients, profile.Email)
			}
		}
	}
	return uniqueEmails(recipients)
}

func (a *app) tenantNotificationEmails(tenantSlug string) []string {
	tenantSlug = normalizeSlug(tenantSlug)
	recipients := []string{}
	for email, profile := range a.profiles {
		if profile.HasTenant(tenantSlug) {
			recipients = append(recipients, email)
		}
	}
	for email := range a.admins {
		recipients = append(recipients, email)
	}
	if a.inviteStore != nil {
		for _, profile := range a.inviteStore.List() {
			if profile.HasTenant(tenantSlug) {
				recipients = append(recipients, profile.Email)
			}
		}
	}
	return uniqueEmails(recipients)
}

func (a *app) parking(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	if !hasCapability(role, capabilityPlatformAdmin) && !profile.HasPermission(permissionParking) {
		http.NotFound(w, r)
		return
	}
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	telemetry := a.parkingTelemetry(r.Context(), tenant)
	a.render(w, "parking", map[string]any{
		"Title":         "Parkplatznutzung",
		"Tenant":        tenant,
		"Email":         email,
		"DisplayName":   profile.DisplayName(),
		"Initials":      profile.Initials(),
		"Role":          role,
		"IsAdmin":       isAdmin,
		"CanSeeParking": true,
		"ActivePage":    "parking",
		"Telemetry":     telemetry,
		"Accounting":    a.parkingAccounting(r.Context(), tenant),
	})
}

func (a *app) parkingSettings(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	if !hasCapability(role, capabilityManageParking) {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	settingsMsg, settingsOK := parkingSettingsMessage(r.URL.Query().Get("settings"))
	a.render(w, "parkingSettings", map[string]any{
		"Title":         "Parkplatz-Abrechnung",
		"Tenant":        tenant,
		"Email":         email,
		"DisplayName":   profile.DisplayName(),
		"Initials":      profile.Initials(),
		"Role":          role,
		"IsAdmin":       true,
		"CanSeeParking": true,
		"ActivePage":    "settings",
		"Accounting":    a.parkingAccounting(r.Context(), tenant),
		"SettingsMsg":   settingsMsg,
		"SettingsOK":    settingsOK,
	})
}

func (a *app) parkingMonth(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	if !hasCapability(role, capabilityPlatformAdmin) && !profile.HasPermission(permissionParking) {
		http.NotFound(w, r)
		return
	}
	month := strings.TrimSpace(r.PathValue("month"))
	if _, err := time.Parse("2006-01", month); err != nil {
		http.NotFound(w, r)
		return
	}
	view := a.parkingMonthDetails(r.Context(), tenant, month)
	if !view.HasHours && view.Summary.Month == "" {
		http.NotFound(w, r)
		return
	}
	a.render(w, "parkingMonth", map[string]any{
		"Title":         "Parkplatznutzung · " + view.MonthLabel,
		"Tenant":        tenant,
		"Email":         email,
		"DisplayName":   profile.DisplayName(),
		"Initials":      profile.Initials(),
		"Role":          role,
		"IsAdmin":       hasCapability(role, capabilityPlatformAdmin),
		"CanSeeParking": true,
		"ActivePage":    "parking",
		"Detail":        view,
	})
}

func (a *app) updateParkingSettings(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	actorEmail, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !hasCapability(role, capabilityManageParking) {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	gridFee, err := parseDecimal(r.FormValue("grid_fee_eur_per_kwh"))
	if err != nil || gridFee < 0 || gridFee > 5 {
		http.Redirect(w, r, "/app/parking/settings?settings=invalid", http.StatusSeeOther)
		return
	}
	if err := a.parkingStore.SetGridFee(tenant.Slug, gridFee); err != nil {
		log.Printf("parking settings save failed for %s: %v", tenant.Slug, err)
		http.Error(w, "Could not save parking settings", http.StatusInternalServerError)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionParkingSettings,
		TargetType: "parking",
		TargetID:   tenant.Slug,
		Summary:    "Parkplatz-Abrechnung geändert",
		Details: map[string]string{
			"grid_fee": formatEURPerKWh(gridFee),
		},
	})
	http.Redirect(w, r, "/app/parking/settings?settings=saved", http.StatusSeeOther)
}

func (a *app) updateParkingMonth(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	actorEmail, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !hasCapability(role, capabilityManageParking) {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	month := strings.TrimSpace(r.FormValue("month"))
	if _, err := time.Parse("2006-01", month); err != nil {
		http.Redirect(w, r, "/app/parking?month=invalid", http.StatusSeeOther)
		return
	}
	paid := parseBool(r.FormValue("paid"))
	if err := a.parkingStore.SetMonthPaid(tenant.Slug, month, paid); err != nil {
		log.Printf("parking month save failed for %s: %v", tenant.Slug, err)
		http.Error(w, "Could not save parking month", http.StatusInternalServerError)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionParkingMonth,
		TargetType: "parking",
		TargetID:   month,
		Summary:    "Monatsstatus geändert",
		Details: map[string]string{
			"month": formatMonthLabel(month, time.Local),
			"paid":  paidLabel(paid),
		},
	})
	http.Redirect(w, r, "/app/parking?month=saved", http.StatusSeeOther)
}

func parkingSettingsMessage(status string) (string, bool) {
	switch status {
	case "saved":
		return "Parkplatz-Abrechnung gespeichert.", true
	case "invalid":
		return "Bitte eine gültige Netzgebühr zwischen 0 und 5 €/kWh eingeben.", false
	default:
		return "", false
	}
}

func announcementMessage(status string) string {
	switch status {
	case "created":
		return "Aushang gespeichert."
	case "updated":
		return "Aushang aktualisiert."
	case "deleted":
		return "Aushang gelöscht."
	case "invalid":
		return "Bitte Titel, Text und Veröffentlichungsdatum prüfen."
	case "missing":
		return "Dieser Aushang wurde nicht gefunden."
	case "error":
		return "Der Aushang konnte nicht gespeichert werden."
	default:
		return ""
	}
}

func canManageAnnouncements(role string) bool {
	return hasCapability(role, capabilityManageAnnouncements)
}

func canManageEvents(role string) bool {
	return hasCapability(role, capabilityManageAnnouncements)
}

func hasCapability(role string, action capability) bool {
	role = normalizeRole(role)
	if role == roleAdmin {
		return true
	}
	switch action {
	case capabilityPlatformAdmin, capabilityManageParking:
		return false
	case capabilityManageUsers, capabilityManageAnnouncements, capabilityManageDocuments, capabilityManageIssues, capabilityManageVotes, capabilityManageBuilding:
		return role == roleManager
	case capabilityOwnerDocuments, capabilityVote:
		return role == roleOwner
	case capabilityOversight:
		return role == roleBeirat || role == roleManager
	default:
		return false
	}
}

func roleCapabilityLabels(role string) []string {
	role = normalizeRole(role)
	switch role {
	case roleAdmin:
		return []string{"Plattformverwaltung", "Alle Bereiche"}
	case roleManager:
		return []string{"Aushang verwalten", "Dokumente verwalten", "Benutzer verwalten", "Gebäude verwalten"}
	case roleOwner:
		return []string{"Eigentümer-Dokumente", "Abstimmungen"}
	case roleRenter:
		return []string{"Bewohnerbereich"}
	case roleBeirat:
		return []string{"Übersicht", "Leserechte"}
	case roleResident:
		return []string{"Bewohnerbereich"}
	default:
		if role == "" {
			return []string{"Bewohnerbereich"}
		}
		return []string{role}
	}
}

func roleSortRank(role string) int {
	switch normalizeRole(role) {
	case roleAdmin:
		return 0
	case roleManager:
		return 1
	case roleBeirat:
		return 2
	case roleOwner:
		return 3
	case roleRenter:
		return 4
	case roleResident:
		return 5
	default:
		return 6
	}
}

func roleClass(role string) string {
	switch normalizeRole(role) {
	case roleAdmin:
		return "role-admin"
	case roleManager:
		return "role-manager"
	case roleOwner:
		return "role-owner"
	case roleRenter:
		return "role-renter"
	case roleBeirat:
		return "role-beirat"
	case roleResident:
		return "role-resident"
	default:
		return "role-resident"
	}
}

func sameOriginPost(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return true
	}
	host := normalizeHost(r.Host)
	if host == "" {
		return false
	}
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		parsed, err := url.Parse(origin)
		return err == nil && normalizeHost(parsed.Host) == host
	}
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
		parsed, err := url.Parse(referer)
		return err == nil && normalizeHost(parsed.Host) == host
	}
	return true
}

func announcementFromForm(r *http.Request, tenantSlug string, author userProfile, now time.Time) (announcement, error) {
	title := strings.TrimSpace(r.FormValue("title"))
	body := strings.TrimSpace(r.FormValue("body"))
	if title == "" || body == "" {
		return announcement{}, fmt.Errorf("title and body are required")
	}
	if len([]rune(title)) > 140 {
		return announcement{}, fmt.Errorf("title too long")
	}
	if len([]rune(body)) > 5000 {
		return announcement{}, fmt.Errorf("body too long")
	}
	publishedAt, err := parseOptionalLocalDateTime(r.FormValue("published_at"), now)
	if err != nil {
		return announcement{}, err
	}
	expiresAt, err := parseOptionalExpiry(r.FormValue("expires_at"))
	if err != nil {
		return announcement{}, err
	}
	item := announcement{
		TenantSlug:  normalizeSlug(tenantSlug),
		Title:       title,
		Body:        body,
		Category:    normalizeAnnouncementCategory(r.FormValue("category")),
		Pinned:      parseBool(r.FormValue("pinned")),
		PublishedAt: publishedAt.UTC(),
		ExpiresAt:   expiresAt,
		AuthorEmail: normalizeEmail(author.Email),
		AuthorName:  author.DisplayName(),
	}
	if item.TenantSlug == "" {
		return announcement{}, fmt.Errorf("tenant is required")
	}
	return item, nil
}

func eventMessage(status string) string {
	switch status {
	case "created":
		return "Termin gespeichert."
	case "updated":
		return "Termin aktualisiert."
	case "deleted":
		return "Termin gelöscht."
	case "invalid":
		return "Bitte Titel und Datum prüfen."
	case "missing":
		return "Dieser Termin wurde nicht gefunden."
	case "error":
		return "Der Termin konnte nicht gespeichert werden."
	default:
		return ""
	}
}

func eventFromForm(r *http.Request, tenantSlug string, author userProfile) (houseEvent, error) {
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		return houseEvent{}, fmt.Errorf("title is required")
	}
	if len([]rune(title)) > 140 {
		return houseEvent{}, fmt.Errorf("title too long")
	}
	body := strings.TrimSpace(r.FormValue("body"))
	if len([]rune(body)) > 3000 {
		return houseEvent{}, fmt.Errorf("body too long")
	}
	startsAt, err := parseRequiredLocalDateTime(r.FormValue("starts_at"))
	if err != nil {
		return houseEvent{}, err
	}
	endsAt, err := parseOptionalEventEnd(r.FormValue("ends_at"), startsAt)
	if err != nil {
		return houseEvent{}, err
	}
	item := houseEvent{
		TenantSlug:  normalizeSlug(tenantSlug),
		Title:       title,
		Body:        body,
		Category:    normalizeEventCategory(r.FormValue("category")),
		Location:    strings.TrimSpace(r.FormValue("location")),
		StartsAt:    startsAt.UTC(),
		EndsAt:      endsAt,
		AuthorEmail: normalizeEmail(author.Email),
		AuthorName:  author.DisplayName(),
	}
	if item.TenantSlug == "" {
		return houseEvent{}, fmt.Errorf("tenant is required")
	}
	return item, nil
}

func parseRequiredLocalDateTime(raw string) (time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}, fmt.Errorf("datetime is required")
	}
	return parseOptionalLocalDateTime(raw, time.Time{})
}

func parseOptionalEventEnd(raw string, startsAt time.Time) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	endsAt, err := parseOptionalLocalDateTime(raw, time.Time{})
	if err != nil {
		return nil, err
	}
	if !endsAt.After(startsAt) {
		return nil, fmt.Errorf("event end must be after start")
	}
	endsAt = endsAt.UTC()
	return &endsAt, nil
}

func parseOptionalLocalDateTime(raw string, fallback time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t, nil
		}
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid datetime")
}

func normalizeEventCategory(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "versammlung", "eigentuemerversammlung", "eigentümerversammlung", "versammlung der eigentümer", "meeting":
		return "Eigentümerversammlung"
	case "reinigung", "cleaning":
		return "Reinigung"
	case "wartung", "maintenance":
		return "Wartung"
	case "ablesung", "ablesetermin", "reading":
		return "Ablesung"
	case "frist", "deadline":
		return "Frist"
	default:
		return "Sonstiges"
	}
}

func eventCategoryClass(raw string) string {
	switch normalizeEventCategory(raw) {
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

func parseOptionalExpiry(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	t, err := parseOptionalLocalDateTime(raw, time.Time{})
	if err != nil {
		return nil, err
	}
	t = t.UTC()
	return &t, nil
}

func normalizeAnnouncementCategory(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "dringend", "urgent":
		return "Dringend"
	case "termin", "date", "event":
		return "Termin"
	case "wartung", "maintenance":
		return "Wartung"
	default:
		return "Info"
	}
}

func selectedAnnouncementCategory(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "all") || strings.EqualFold(raw, "alle") {
		return ""
	}
	return normalizeAnnouncementCategory(raw)
}

func filterAnnouncements(items []announcement, category string, query string) []announcement {
	category = selectedAnnouncementCategory(category)
	query = strings.ToLower(strings.TrimSpace(query))
	if category == "" && query == "" {
		return items
	}
	out := []announcement{}
	for _, item := range items {
		if category != "" && item.Category != category {
			continue
		}
		if query != "" {
			haystack := strings.ToLower(item.Title + "\n" + item.Body + "\n" + item.Category)
			if !strings.Contains(haystack, query) {
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

func announcementFilterViews(query string, selectedCategory string) []announcementFilterView {
	selectedCategory = selectedAnnouncementCategory(selectedCategory)
	categories := []string{"", "Info", "Termin", "Wartung", "Dringend"}
	labels := map[string]string{"": "Alle"}
	out := make([]announcementFilterView, 0, len(categories))
	for _, category := range categories {
		label := labels[category]
		if label == "" {
			label = category
		}
		values := url.Values{}
		if strings.TrimSpace(query) != "" {
			values.Set("q", strings.TrimSpace(query))
		}
		if category != "" {
			values.Set("category", category)
		}
		filterURL := "/app/announcements"
		if encoded := values.Encode(); encoded != "" {
			filterURL += "?" + encoded
		}
		out = append(out, announcementFilterView{
			Label:  label,
			URL:    filterURL,
			Active: category == selectedCategory,
		})
	}
	return out
}

func unreadAnnouncementCount(items []announcement, lastSeen time.Time, now time.Time) int {
	count := 0
	for _, item := range items {
		if announcementUnread(item, lastSeen, now) {
			count++
		}
	}
	return count
}

func announcementUnread(item announcement, lastSeen time.Time, now time.Time) bool {
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

func announcementViews(items []announcement, now time.Time, includeStatus bool) []announcementView {
	return announcementViewsWithReadState(items, now, includeStatus, time.Time{})
}

func announcementViewsWithReadState(items []announcement, now time.Time, includeStatus bool, lastSeen time.Time) []announcementView {
	views := make([]announcementView, 0, len(items))
	for _, item := range items {
		views = append(views, announcementViewFrom(item, now, includeStatus, lastSeen))
	}
	return views
}

func announcementViewFrom(item announcement, now time.Time, includeStatus bool, lastSeen time.Time) announcementView {
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
		expiresAt = formatLocalDateTime(*item.ExpiresAt)
		expiresAtInput = formatLocalDateTimeInput(*item.ExpiresAt)
	}
	author := strings.TrimSpace(item.AuthorName)
	if author == "" {
		author = item.AuthorEmail
	}
	return announcementView{
		ID:                 item.ID,
		Title:              item.Title,
		Body:               item.Body,
		BodyHTML:           plainTextHTML(item.Body),
		Category:           item.Category,
		CategoryClass:      strings.ToLower(normalizeSlug(item.Category)),
		Pinned:             item.Pinned,
		PinnedChecked:      item.Pinned,
		PublishedAt:        formatLocalDateTime(item.PublishedAt),
		PublishedAtInput:   formatLocalDateTimeInput(item.PublishedAt),
		ExpiresAt:          expiresAt,
		ExpiresAtInput:     expiresAtInput,
		HasExpiresAt:       item.ExpiresAt != nil,
		Author:             author,
		Status:             status,
		Published:          published,
		Expired:            expired,
		Unread:             announcementUnread(item, lastSeen, now),
		EditDialogID:       "announcement-edit-" + item.ID,
		DeleteConfirmLabel: "Aushang \"" + item.Title + "\" wirklich löschen?",
	}
}

func eventViews(items []houseEvent, now time.Time) []houseEventView {
	views := make([]houseEventView, 0, len(items))
	for _, item := range items {
		views = append(views, eventViewFrom(item, now))
	}
	return views
}

func eventViewFrom(item houseEvent, now time.Time) houseEventView {
	startLocal := item.StartsAt.In(time.Local)
	past := !eventRollsOffAt(item).After(now)
	status := "Geplant"
	if past {
		status = "Vergangen"
	} else if sameLocalDate(startLocal, now.In(time.Local)) {
		status = "Heute"
	}
	endsAt := ""
	endsAtInput := ""
	if item.EndsAt != nil {
		endsLocal := item.EndsAt.In(time.Local)
		endsAt = formatLocalDateTime(endsLocal)
		endsAtInput = formatLocalDateTimeInput(endsLocal)
	}
	author := strings.TrimSpace(item.AuthorName)
	if author == "" {
		author = item.AuthorEmail
	}
	body := strings.TrimSpace(item.Body)
	location := strings.TrimSpace(item.Location)
	return houseEventView{
		ID:                 item.ID,
		Title:              item.Title,
		Body:               item.Body,
		BodyHTML:           plainTextHTML(item.Body),
		HasBody:            body != "",
		Category:           item.Category,
		CategoryClass:      eventCategoryClass(item.Category),
		Location:           location,
		HasLocation:        location != "",
		StartsAt:           formatLocalDateTime(startLocal),
		StartsAtInput:      formatLocalDateTimeInput(startLocal),
		EndsAt:             endsAt,
		EndsAtInput:        endsAtInput,
		HasEndsAt:          item.EndsAt != nil,
		DateBadgeDay:       startLocal.Format("02"),
		DateBadgeMonth:     germanMonthShort(startLocal),
		TimeRange:          eventTimeRange(item),
		Status:             status,
		Past:               past,
		Author:             author,
		EditDialogID:       "event-edit-" + item.ID,
		DeleteConfirmLabel: "Termin \"" + item.Title + "\" wirklich löschen?",
	}
}

func eventTimeRange(item houseEvent) string {
	startLocal := item.StartsAt.In(time.Local)
	if item.EndsAt == nil {
		return formatLocalTime(startLocal)
	}
	endLocal := item.EndsAt.In(time.Local)
	if sameLocalDate(startLocal, endLocal) {
		return formatLocalTime(startLocal) + " bis " + formatLocalTime(endLocal)
	}
	return formatLocalShortDateTime(startLocal) + " bis " + formatLocalShortDateTime(endLocal)
}

func eventRollsOffAt(item houseEvent) time.Time {
	if item.EndsAt != nil {
		return *item.EndsAt
	}
	startLocal := item.StartsAt.In(time.Local)
	year, month, day := startLocal.Date()
	return time.Date(year, month, day, 23, 59, 59, 0, time.Local).UTC()
}

func sameLocalDate(a time.Time, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func germanMonthShort(t time.Time) string {
	months := [...]string{"Jan", "Feb", "Mär", "Apr", "Mai", "Jun", "Jul", "Aug", "Sep", "Okt", "Nov", "Dez"}
	month := int(t.Month())
	if month < 1 || month > len(months) {
		return ""
	}
	return months[month-1]
}

func plainTextHTML(body string) template.HTML {
	escaped := template.HTMLEscapeString(strings.TrimSpace(body))
	escaped = strings.ReplaceAll(escaped, "\r\n", "\n")
	escaped = strings.ReplaceAll(escaped, "\n\n", "<br><br>")
	escaped = strings.ReplaceAll(escaped, "\n", "<br>")
	return template.HTML(escaped)
}

func issueFromForm(r *http.Request, tenantSlug string, author userProfile, now time.Time) (residentIssue, error) {
	id, err := randomToken(12)
	if err != nil {
		return residentIssue{}, err
	}
	title := strings.TrimSpace(r.FormValue("title"))
	body := strings.TrimSpace(r.FormValue("body"))
	if title == "" || body == "" || len([]rune(title)) > 140 || len([]rune(body)) > 4000 {
		return residentIssue{}, fmt.Errorf("invalid issue text")
	}
	locationType := normalizeIssueLocation(r.FormValue("location_type"))
	if locationType == "" {
		return residentIssue{}, fmt.Errorf("invalid location")
	}
	locationDetail := strings.TrimSpace(r.FormValue("location_detail"))
	if len([]rune(locationDetail)) > 160 {
		return residentIssue{}, fmt.Errorf("location detail too long")
	}
	category := normalizeIssueCategory(r.FormValue("category"))
	if category == "" {
		return residentIssue{}, fmt.Errorf("invalid category")
	}
	return residentIssue{
		ID:             id,
		TenantSlug:     normalizeSlug(tenantSlug),
		AuthorEmail:    normalizeEmail(author.Email),
		AuthorName:     author.DisplayName(),
		Category:       category,
		Title:          title,
		Body:           body,
		LocationType:   locationType,
		LocationDetail: locationDetail,
		Status:         issueStatusOpen,
		Priority:       issuePriorityNorm,
		CreatedAt:      now.UTC(),
		UpdatedAt:      now.UTC(),
	}, nil
}

func normalizeIssueCategory(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "reparatur", "repair", "mangel", "mängel", "schaden":
		return "Reparatur"
	case "frage", "question":
		return "Frage"
	case "vorschlag", "idee", "suggestion":
		return "Vorschlag"
	case "sonstiges", "sonstige", "other":
		return "Sonstiges"
	default:
		return ""
	}
}

func normalizeIssueStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "neu", "offen", "new", "open":
		return issueStatusNew
	case "in bearbeitung", "bearbeitung", "in-arbeit", "progress", "in_progress":
		return issueStatusProgress
	case "erledigt", "geschlossen", "done", "closed":
		return issueStatusDone
	case "abgelehnt", "rejected":
		return issueStatusRejected
	case "duplikat", "duplicate":
		return issueStatusDuplicate
	default:
		return ""
	}
}

func issueStatuses() []string {
	return []string{issueStatusNew, issueStatusProgress, issueStatusDone, issueStatusRejected, issueStatusDuplicate}
}

func normalizeIssuePriority(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "niedrig", "low":
		return issuePriorityLow
	case "", "normal", "mittel", "medium":
		return issuePriorityNorm
	case "hoch", "high":
		return issuePriorityHigh
	case "dringend", "urgent":
		return issuePriorityUrgent
	default:
		return ""
	}
}

func issuePriorities() []string {
	return []string{issuePriorityLow, issuePriorityNorm, issuePriorityHigh, issuePriorityUrgent}
}

func issueSelectOptions(values []string, selected string) []selectOption {
	options := make([]selectOption, 0, len(values))
	for _, value := range values {
		options = append(options, selectOption{Value: value, Label: value, Selected: value == selected})
	}
	return options
}

func normalizeIssueLocation(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case issueLocationUnit, "eigene einheit", "wohnung", "unit":
		return issueLocationUnit
	case issueLocationCommon, "gemeinschaft", "allgemeinbereich":
		return issueLocationCommon
	default:
		return ""
	}
}

func issueLocationLabel(locationType string, detail string) string {
	label := "Gemeinschaft"
	if normalizeIssueLocation(locationType) == issueLocationUnit {
		label = "Eigene Einheit"
	}
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return label
	}
	return label + " · " + detail
}

func issueStatusClass(status string) string {
	switch normalizeIssueStatus(status) {
	case issueStatusProgress:
		return "status-progress"
	case issueStatusDone:
		return "status-done"
	case issueStatusRejected, issueStatusDuplicate:
		return "status-closed"
	default:
		return "status-open"
	}
}

func canResidentTransition(from string, to string) bool {
	from = normalizeIssueStatus(from)
	to = normalizeIssueStatus(to)
	switch to {
	case issueStatusDone:
		return from == issueStatusNew || from == issueStatusProgress
	case issueStatusNew:
		return from == issueStatusDone || from == issueStatusRejected || from == issueStatusDuplicate
	default:
		return false
	}
}

func issuePhotoHeader(r *http.Request) (*multipart.FileHeader, bool) {
	if r.MultipartForm == nil {
		return nil, false
	}
	files := r.MultipartForm.File["photo"]
	if len(files) == 0 {
		files = r.MultipartForm.File["photos"]
	}
	if len(files) == 0 || files[0] == nil || files[0].Filename == "" || files[0].Size == 0 {
		return nil, false
	}
	return files[0], true
}

func tenantHeroHeader(r *http.Request) (*multipart.FileHeader, bool) {
	if r.MultipartForm == nil {
		return nil, false
	}
	files := r.MultipartForm.File["hero_image"]
	if len(files) == 0 {
		files = r.MultipartForm.File["photo"]
	}
	if len(files) == 0 || files[0] == nil || files[0].Filename == "" || files[0].Size == 0 {
		return nil, false
	}
	return files[0], true
}

func issueBoardAction(boardOnly bool) string {
	if boardOnly {
		return "/app/anliegen/board"
	}
	return "/app/anliegen"
}

func issueBoardFiltersFromQuery(values url.Values) issueBoardFilterView {
	priority := ""
	if raw := strings.TrimSpace(values.Get("priority")); raw != "" {
		priority = normalizeIssuePriority(raw)
	}
	filters := issueBoardFilterView{
		Status:   normalizeIssueStatus(values.Get("status")),
		Priority: priority,
		Category: normalizeIssueCategory(values.Get("category")),
		Assignee: normalizeEmail(values.Get("assignee")),
		Sort:     normalizeIssueBoardSort(values.Get("sort")),
	}
	if filters.Sort == "" {
		filters.Sort = "updated"
	}
	filters.HasActive = filters.Status != "" || filters.Priority != "" || filters.Category != "" || filters.Assignee != "" || filters.Sort != "updated"
	return filters
}

func issueBoardFilterOptions(filters issueBoardFilterView) issueBoardFilterView {
	filters.StatusOptions = issueFilterOptions(issueStatuses(), filters.Status, "Alle Status")
	filters.PriorityOptions = issueFilterOptions(issuePriorities(), filters.Priority, "Alle Prioritäten")
	filters.CategoryOptions = issueFilterOptions(issueCategories(), filters.Category, "Alle Kategorien")
	filters.SortOptions = []selectOption{
		{Value: "updated", Label: "Zuletzt aktualisiert", Selected: filters.Sort == "updated"},
		{Value: "age", Label: "Älteste zuerst", Selected: filters.Sort == "age"},
		{Value: "priority", Label: "Priorität", Selected: filters.Sort == "priority"},
		{Value: "status", Label: "Status", Selected: filters.Sort == "status"},
		{Value: "category", Label: "Kategorie", Selected: filters.Sort == "category"},
		{Value: "assignee", Label: "Zuständigkeit", Selected: filters.Sort == "assignee"},
	}
	return filters
}

func issueFilterOptions(values []string, selected string, allLabel string) []selectOption {
	options := []selectOption{{Value: "", Label: allLabel, Selected: selected == ""}}
	for _, value := range values {
		options = append(options, selectOption{Value: value, Label: value, Selected: value == selected})
	}
	return options
}

func issueCategories() []string {
	return []string{"Reparatur", "Frage", "Vorschlag", "Sonstiges"}
}

func normalizeIssueBoardSort(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "updated", "aktualisiert":
		return "updated"
	case "age", "alter", "oldest":
		return "age"
	case "priority", "priorität":
		return "priority"
	case "status":
		return "status"
	case "category", "kategorie":
		return "category"
	case "assignee", "zuständig", "zustaendig":
		return "assignee"
	default:
		return ""
	}
}

func filterIssueBoard(items []residentIssue, filters issueBoardFilterView) []residentIssue {
	out := make([]residentIssue, 0, len(items))
	for _, item := range items {
		if filters.Status != "" && normalizeIssueStatus(item.Status) != filters.Status {
			continue
		}
		if filters.Priority != "" && normalizeIssuePriority(item.Priority) != filters.Priority {
			continue
		}
		if filters.Category != "" && normalizeIssueCategory(item.Category) != filters.Category {
			continue
		}
		if filters.Assignee != "" && normalizeEmail(item.AssigneeEmail) != filters.Assignee {
			continue
		}
		out = append(out, item)
	}
	sortIssueBoard(out, filters.Sort)
	return out
}

func sortIssueBoard(items []residentIssue, sortMode string) {
	sortMode = normalizeIssueBoardSort(sortMode)
	if sortMode == "" || sortMode == "updated" {
		sortIssues(items)
		return
	}
	sort.SliceStable(items, func(i, j int) bool {
		switch sortMode {
		case "age":
			if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
				return items[i].CreatedAt.Before(items[j].CreatedAt)
			}
		case "priority":
			if issuePriorityRank(items[i].Priority) != issuePriorityRank(items[j].Priority) {
				return issuePriorityRank(items[i].Priority) > issuePriorityRank(items[j].Priority)
			}
		case "status":
			if issueStatusRank(items[i].Status) != issueStatusRank(items[j].Status) {
				return issueStatusRank(items[i].Status) < issueStatusRank(items[j].Status)
			}
		case "category":
			if items[i].Category != items[j].Category {
				return items[i].Category < items[j].Category
			}
		case "assignee":
			left := normalizeEmail(items[i].AssigneeEmail)
			right := normalizeEmail(items[j].AssigneeEmail)
			if left != right {
				return left < right
			}
		}
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
}

func issuePriorityRank(priority string) int {
	switch normalizeIssuePriority(priority) {
	case issuePriorityUrgent:
		return 4
	case issuePriorityHigh:
		return 3
	case issuePriorityNorm:
		return 2
	case issuePriorityLow:
		return 1
	default:
		return 0
	}
}

func issueStatusRank(status string) int {
	switch normalizeIssueStatus(status) {
	case issueStatusNew:
		return 1
	case issueStatusProgress:
		return 2
	case issueStatusDone:
		return 3
	case issueStatusRejected:
		return 4
	case issueStatusDuplicate:
		return 5
	default:
		return 9
	}
}

func issueOpenCount(items []residentIssue) int {
	count := 0
	for _, item := range items {
		if issueIsOpen(item) {
			count++
		}
	}
	return count
}

func issueIsOpen(item residentIssue) bool {
	switch normalizeIssueStatus(item.Status) {
	case issueStatusDone, issueStatusRejected, issueStatusDuplicate:
		return false
	default:
		return true
	}
}

func (a *app) visibleIssuesForActor(tenantSlug string, email string, role string) []residentIssue {
	if a.issueStore == nil {
		return nil
	}
	all := a.issueStore.ListTenant(tenantSlug)
	out := make([]residentIssue, 0, len(all))
	for _, item := range all {
		if a.canViewIssueForActor(tenantSlug, item, email, role) {
			out = append(out, item)
		}
	}
	sortIssues(out)
	return out
}

func (a *app) canViewIssueForActor(tenantSlug string, item residentIssue, email string, role string) bool {
	tenantSlug = normalizeSlug(tenantSlug)
	email = normalizeEmail(email)
	if tenantSlug == "" || normalizeSlug(item.TenantSlug) != tenantSlug || email == "" {
		return false
	}
	if hasCapability(role, capabilityManageIssues) || hasCapability(role, capabilityOversight) {
		return true
	}
	if normalizeEmail(item.AuthorEmail) == email {
		return true
	}
	return normalizeIssueLocation(item.LocationType) == issueLocationCommon && a.actorCanSeeCommonIssues(tenantSlug, email, role)
}

func (a *app) actorCanSeeCommonIssues(tenantSlug string, email string, role string) bool {
	if normalizeRole(role) == roleOwner {
		return true
	}
	for _, membership := range a.unitStore.UnitsForEmail(tenantSlug, email) {
		if normalizeRole(membership.Relation) == roleOwner {
			return true
		}
	}
	return false
}

func issueViews(items []residentIssue) []issueView {
	return issueViewsForActor(items, "", "")
}

func issueViewsForActor(items []residentIssue, role string, actorEmail string) []issueView {
	views := make([]issueView, 0, len(items))
	actorEmail = normalizeEmail(actorEmail)
	canManage := hasCapability(role, capabilityManageIssues)
	readOnly := hasCapability(role, capabilityOversight) && !canManage
	for _, item := range items {
		photoCount := len(item.PhotoPaths)
		comments := issueCommentViews(item.Comments)
		status := normalizeIssueStatus(item.Status)
		if status == "" {
			status = issueStatusOpen
		}
		priority := normalizeIssuePriority(item.Priority)
		if priority == "" {
			priority = issuePriorityNorm
		}
		isOwner := normalizeEmail(item.AuthorEmail) == actorEmail
		author := strings.TrimSpace(item.AuthorName)
		if author == "" {
			author = item.AuthorEmail
		}
		canResidentAct := !canManage && !readOnly && isOwner
		views = append(views, issueView{
			ID:              item.ID,
			Title:           item.Title,
			Body:            item.Body,
			Author:          author,
			AuthorEmail:     item.AuthorEmail,
			Category:        item.Category,
			Status:          status,
			StatusClass:     issueStatusClass(status),
			Priority:        priority,
			AssigneeEmail:   item.AssigneeEmail,
			HasAssignee:     item.AssigneeEmail != "",
			Location:        issueLocationLabel(item.LocationType, item.LocationDetail),
			CreatedAt:       formatLocalDateTime(item.CreatedAt),
			CanComment:      canManage || canResidentAct,
			CanClose:        canResidentAct && canResidentTransition(status, issueStatusDone),
			CanReopen:       canResidentAct && canResidentTransition(status, issueStatusNew),
			PhotoCount:      photoCount,
			HasPhotos:       photoCount > 0,
			Comments:        comments,
			HasComments:     len(comments) > 0,
			StatusOptions:   issueSelectOptions(issueStatuses(), status),
			PriorityOptions: issueSelectOptions(issuePriorities(), priority),
		})
	}
	return views
}

func issueCommentViews(comments []issueComment) []issueCommentView {
	views := make([]issueCommentView, 0, len(comments))
	sort.SliceStable(comments, func(i, j int) bool {
		return comments[i].CreatedAt.Before(comments[j].CreatedAt)
	})
	for _, comment := range comments {
		author := strings.TrimSpace(comment.AuthorName)
		if author == "" {
			author = comment.AuthorEmail
		}
		views = append(views, issueCommentView{
			Author:    author,
			Body:      comment.Body,
			CreatedAt: formatLocalDateTime(comment.CreatedAt),
		})
	}
	return views
}

func (a *app) settingsHub(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	a.render(w, "settingsHub", map[string]any{
		"Title":         "Einstellungen",
		"Tenant":        tenant,
		"Email":         email,
		"DisplayName":   profile.DisplayName(),
		"Initials":      profile.Initials(),
		"Role":          role,
		"IsAdmin":       isAdmin,
		"CanSeeParking": isAdmin || profile.HasPermission(permissionParking),
		"ActivePage":    "settings",
	})
}

func (a *app) buildingSettings(w http.ResponseWriter, r *http.Request) {
	tenant, email, role, profile, ok := a.buildingSettingsContext(w, r)
	if !ok {
		return
	}
	buildingMsg, buildingOK := buildingSettingsMessage(r.URL.Query().Get("building"))
	heroMsg, heroOK := buildingHeroMessage(r.URL.Query().Get("hero"))
	unitMsg, unitOK := buildingUnitMessage(r.URL.Query().Get("unit"))
	a.render(w, "buildingSettings", map[string]any{
		"Title":         "Gebäude",
		"Tenant":        tenant,
		"Email":         email,
		"DisplayName":   profile.DisplayName(),
		"Initials":      profile.Initials(),
		"Role":          role,
		"IsAdmin":       hasCapability(role, capabilityPlatformAdmin),
		"CanSeeParking": hasCapability(role, capabilityPlatformAdmin) || profile.HasPermission(permissionParking),
		"ActivePage":    "settings",
		"BuildingMsg":   buildingMsg,
		"BuildingOK":    buildingOK,
		"HeroMsg":       heroMsg,
		"HeroOK":        heroOK,
		"UnitMsg":       unitMsg,
		"UnitOK":        unitOK,
		"Units":         buildingUnitViews(a.unitStore.ListTenant(tenant.Slug)),
		"UnitsEmpty":    emptyState("Noch keine Einheiten", "Angelegte Einheiten erscheinen hier mit Anteil und Kontaktlinks."),
	})
}

func (a *app) auditLog(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !canViewAudit(role) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	action := normalizeAuditAction(r.URL.Query().Get("action"))
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	events := []auditEvent{}
	if a.auditStore != nil {
		events = a.auditStore.List(auditFilter{
			TenantSlug: tenant.Slug,
			Action:     action,
			Query:      query,
			Limit:      200,
		})
	}
	a.render(w, "auditLog", map[string]any{
		"Title":         "Audit-Log",
		"Tenant":        tenant,
		"Email":         email,
		"DisplayName":   profile.DisplayName(),
		"Initials":      profile.Initials(),
		"Role":          role,
		"IsAdmin":       hasCapability(role, capabilityPlatformAdmin),
		"CanSeeParking": hasCapability(role, capabilityPlatformAdmin) || profile.HasPermission(permissionParking),
		"ActivePage":    "audit",
		"Events":        auditEventViews(events),
		"HasEvents":     len(events) > 0,
		"EventsEmpty":   emptyState("Noch keine Audit-Einträge", "Sensible Aktionen erscheinen hier, sobald sie im Portal ausgeführt werden."),
		"ActionOptions": auditActionOptions(action),
		"ActionFilter":  action,
		"SearchQuery":   query,
	})
}

func canViewAudit(role string) bool {
	role = normalizeRole(role)
	return role == roleAdmin || role == roleManager
}

func auditEventViews(events []auditEvent) []auditEventView {
	views := make([]auditEventView, 0, len(events))
	for _, event := range events {
		views = append(views, auditEventViewFrom(event))
	}
	return views
}

func auditEventViewFrom(event auditEvent) auditEventView {
	details := make([]auditDetailView, 0, len(event.Details))
	keys := make([]string, 0, len(event.Details))
	for key := range event.Details {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		details = append(details, auditDetailView{Key: auditDetailLabel(key), Value: event.Details[key]})
	}
	return auditEventView{
		At:         formatLocalDateTime(event.At),
		Action:     event.Action,
		ActionText: auditActionLabel(event.Action),
		Actor:      event.ActorEmail,
		ActorRole:  event.ActorRole,
		Target:     auditTargetLabel(event.TargetType, event.TargetID),
		TargetType: auditTargetTypeLabel(event.TargetType),
		Summary:    event.Summary,
		Details:    details,
		HasDetails: len(details) > 0,
	}
}

func auditActionOptions(selected string) []selectOption {
	options := []selectOption{{Value: "", Label: "Alle Aktionen", Selected: selected == ""}}
	for _, action := range []string{
		auditActionLogin,
		auditActionInviteCreate,
		auditActionInviteUpdate,
		auditActionInviteDelete,
		auditActionBuildingUpdate,
		auditActionHeroUpdate,
		auditActionUnitSave,
		auditActionUnitDelete,
		auditActionDocumentUpload,
		auditActionDocumentDownload,
		auditActionDocumentReplace,
		auditActionVoteCreate,
		auditActionVoteOpen,
		auditActionVoteClose,
		auditActionVoteCast,
		auditActionVoteReminder,
		auditActionParkingSettings,
		auditActionParkingMonth,
		auditActionIssueWorkflow,
	} {
		options = append(options, selectOption{Value: action, Label: auditActionLabel(action), Selected: selected == action})
	}
	return options
}

func auditActionLabel(action string) string {
	switch normalizeAuditAction(action) {
	case auditActionLogin:
		return "Anmeldung"
	case auditActionInviteCreate:
		return "Einladung angelegt"
	case auditActionInviteUpdate:
		return "Einladung geändert"
	case auditActionInviteDelete:
		return "Einladung gelöscht"
	case auditActionBuildingUpdate:
		return "Gebäude geändert"
	case auditActionHeroUpdate:
		return "Hero-Bild geändert"
	case auditActionUnitSave:
		return "Einheit gespeichert"
	case auditActionUnitDelete:
		return "Einheit gelöscht"
	case auditActionDocumentUpload:
		return "Dokument hochgeladen"
	case auditActionDocumentDownload:
		return "Dokument heruntergeladen"
	case auditActionDocumentReplace:
		return "Dokument ersetzt"
	case auditActionVoteCreate:
		return "Abstimmung angelegt"
	case auditActionVoteOpen:
		return "Abstimmung geöffnet"
	case auditActionVoteClose:
		return "Abstimmung geschlossen"
	case auditActionVoteCast:
		return "Stimme gespeichert"
	case auditActionVoteReminder:
		return "Abstimmungs-Erinnerung gesendet"
	case auditActionParkingSettings:
		return "Parkplatz-Abrechnung geändert"
	case auditActionParkingMonth:
		return "Monatsstatus geändert"
	case auditActionIssueWorkflow:
		return "Anliegen-Workflow geändert"
	default:
		return action
	}
}

func auditTargetLabel(targetType string, targetID string) string {
	targetType = auditTargetTypeLabel(targetType)
	targetID = strings.TrimSpace(targetID)
	if targetType == "" {
		return targetID
	}
	if targetID == "" {
		return targetType
	}
	return targetType + ": " + targetID
}

func auditTargetTypeLabel(targetType string) string {
	switch strings.TrimSpace(targetType) {
	case "user":
		return "Person"
	case "session":
		return "Sitzung"
	case "building":
		return "Gebäude"
	case "hero":
		return "Hero-Bild"
	case "unit":
		return "Einheit"
	case "parking":
		return "Parkplatz"
	case "issue":
		return "Anliegen"
	case "document":
		return "Dokument"
	case "ballot":
		return "Abstimmung"
	default:
		return strings.TrimSpace(targetType)
	}
}

func auditDetailLabel(key string) string {
	switch key {
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
	case "month":
		return "Monat"
	case "paid":
		return "Status"
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
	case "unit":
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
	default:
		return strings.ReplaceAll(key, "_", " ")
	}
}

func (a *app) recordAudit(event auditEvent) {
	if a == nil || a.auditStore == nil {
		return
	}
	if err := a.auditStore.Append(event); err != nil {
		log.Printf("audit record failed for %s: %v", event.Action, err)
	}
}

func (a *app) updateBuildingSettings(w http.ResponseWriter, r *http.Request) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, r)
	if !ok {
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	override, err := tenantOverrideFromForm(r.Form)
	if err != nil {
		http.Redirect(w, r, "/app/settings/building?building=invalid", http.StatusSeeOther)
		return
	}
	if a.tenantOverrides != nil {
		if err := a.tenantOverrides.SetMeta(tenant.Slug, override); err != nil {
			log.Printf("building settings save failed for %s: %v", tenant.Slug, err)
			http.Redirect(w, r, "/app/settings/building?building=error", http.StatusSeeOther)
			return
		}
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionBuildingUpdate,
		TargetType: "building",
		TargetID:   tenant.Slug,
		Summary:    "Gebäudedaten geändert",
		Details: map[string]string{
			"changed_fields": "Stammdaten, Kontaktblock, Notdienst, Hausmeister",
		},
	})
	http.Redirect(w, r, "/app/settings/building?building=saved", http.StatusSeeOther)
}

func (a *app) updateBuildingHero(w http.ResponseWriter, r *http.Request) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, r)
	if !ok {
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := r.ParseMultipartForm(maxTenantHeroFormBytes); err != nil {
		http.Redirect(w, r, "/app/settings/building?hero=invalid", http.StatusSeeOther)
		return
	}
	header, ok := tenantHeroHeader(r)
	if !ok {
		http.Redirect(w, r, "/app/settings/building?hero=invalid", http.StatusSeeOther)
		return
	}
	filename, err := a.saveTenantHeroImage(tenant.Slug, header)
	if err != nil {
		log.Printf("tenant hero upload failed for %s: %v", tenant.Slug, err)
		http.Redirect(w, r, "/app/settings/building?hero=invalid", http.StatusSeeOther)
		return
	}
	if a.tenantOverrides != nil {
		if err := a.tenantOverrides.SetHeroImage(tenant.Slug, filename); err != nil {
			log.Printf("tenant hero save failed for %s: %v", tenant.Slug, err)
			http.Redirect(w, r, "/app/settings/building?hero=error", http.StatusSeeOther)
			return
		}
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionHeroUpdate,
		TargetType: "hero",
		TargetID:   tenant.Slug,
		Summary:    "Hero-Bild geändert",
	})
	http.Redirect(w, r, "/app/settings/building?hero=saved", http.StatusSeeOther)
}

func (a *app) upsertBuildingUnit(w http.ResponseWriter, r *http.Request) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, r)
	if !ok {
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	item, err := buildingUnitFromForm(tenant.Slug, r.Form)
	if err != nil {
		http.Redirect(w, r, "/app/settings/building?unit=invalid", http.StatusSeeOther)
		return
	}
	origID := normalizeUnitID(r.FormValue("orig_id"))
	units := a.unitStore.ListTenant(tenant.Slug)
	replaced := false
	for i := range units {
		if normalizeUnitID(units[i].ID) == item.ID && (origID == "" || origID != item.ID) {
			http.Redirect(w, r, "/app/settings/building?unit=duplicate", http.StatusSeeOther)
			return
		}
	}
	if origID == "" {
		origID = item.ID
	}
	for i := range units {
		if normalizeUnitID(units[i].ID) == origID {
			units[i] = item
			replaced = true
			break
		}
	}
	if !replaced {
		units = append(units, item)
	}
	if err := a.unitStore.SetTenantUnits(tenant.Slug, units); err != nil {
		log.Printf("unit save failed for %s: %v", tenant.Slug, err)
		http.Redirect(w, r, "/app/settings/building?unit=error", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionUnitSave,
		TargetType: "unit",
		TargetID:   item.ID,
		Summary:    "Einheit gespeichert",
		Details: map[string]string{
			"unit_label": item.Label,
			"share":      formatMiteigentumsanteil(item.MiteigentumsanteilPPM),
		},
	})
	http.Redirect(w, r, "/app/settings/building?unit=saved", http.StatusSeeOther)
}

func (a *app) deleteBuildingUnit(w http.ResponseWriter, r *http.Request) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, r)
	if !ok {
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	deleteID := normalizeUnitID(r.FormValue("id"))
	if deleteID == "" {
		http.Redirect(w, r, "/app/settings/building?unit=invalid", http.StatusSeeOther)
		return
	}
	units := a.unitStore.ListTenant(tenant.Slug)
	kept := units[:0]
	removed := false
	removedUnit := unit{}
	for _, item := range units {
		if normalizeUnitID(item.ID) == deleteID {
			removed = true
			removedUnit = item
			continue
		}
		kept = append(kept, item)
	}
	if !removed {
		http.Redirect(w, r, "/app/settings/building?unit=missing", http.StatusSeeOther)
		return
	}
	if err := a.unitStore.SetTenantUnits(tenant.Slug, kept); err != nil {
		log.Printf("unit delete failed for %s: %v", tenant.Slug, err)
		http.Redirect(w, r, "/app/settings/building?unit=error", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionUnitDelete,
		TargetType: "unit",
		TargetID:   deleteID,
		Summary:    "Einheit gelöscht",
		Details: map[string]string{
			"unit_label": removedUnit.Label,
			"share":      formatMiteigentumsanteil(removedUnit.MiteigentumsanteilPPM),
		},
	})
	http.Redirect(w, r, "/app/settings/building?unit=deleted", http.StatusSeeOther)
}

func (a *app) buildingSettingsContext(w http.ResponseWriter, r *http.Request) (tenantConfig, string, string, userProfile, bool) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return tenantConfig{}, "", "", userProfile{}, false
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return tenantConfig{}, "", "", userProfile{}, false
	}
	if !hasCapability(role, capabilityManageBuilding) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return tenantConfig{}, "", "", userProfile{}, false
	}
	return tenant, email, role, a.profileForTenant(email, tenant.Slug), true
}

func tenantOverrideFromForm(values url.Values) (tenantOverride, error) {
	override := tenantOverride{
		MetaSet:        true,
		Name:           strings.TrimSpace(values.Get("name")),
		Address:        strings.TrimSpace(values.Get("address")),
		ContactName:    strings.TrimSpace(values.Get("contact_name")),
		ContactEmail:   normalizeEmail(values.Get("contact_email")),
		ContactPhone:   strings.TrimSpace(values.Get("contact_phone")),
		EmergencyName:  strings.TrimSpace(values.Get("emergency_name")),
		EmergencyPhone: strings.TrimSpace(values.Get("emergency_phone")),
		CaretakerName:  strings.TrimSpace(values.Get("caretaker_name")),
		CaretakerEmail: normalizeEmail(values.Get("caretaker_email")),
		CaretakerPhone: strings.TrimSpace(values.Get("caretaker_phone")),
	}
	if override.Name == "" || override.Address == "" {
		return tenantOverride{}, fmt.Errorf("building name and address are required")
	}
	if len([]rune(override.Name)) > 160 || len([]rune(override.Address)) > 500 || len([]rune(override.ContactName)) > 160 || len([]rune(override.ContactPhone)) > 80 || len([]rune(override.EmergencyName)) > 160 || len([]rune(override.EmergencyPhone)) > 80 || len([]rune(override.CaretakerName)) > 160 || len([]rune(override.CaretakerPhone)) > 80 {
		return tenantOverride{}, fmt.Errorf("building field too long")
	}
	if rawEmail := strings.TrimSpace(values.Get("contact_email")); rawEmail != "" {
		if _, err := mail.ParseAddress(rawEmail); err != nil || override.ContactEmail == "" {
			return tenantOverride{}, fmt.Errorf("invalid contact email")
		}
	}
	if rawEmail := strings.TrimSpace(values.Get("caretaker_email")); rawEmail != "" {
		if _, err := mail.ParseAddress(rawEmail); err != nil || override.CaretakerEmail == "" {
			return tenantOverride{}, fmt.Errorf("invalid caretaker email")
		}
	}
	return override, nil
}

func buildingUnitFromForm(tenantSlug string, values url.Values) (unit, error) {
	label := strings.TrimSpace(values.Get("label"))
	id := normalizeUnitID(values.Get("id"))
	if id == "" {
		id = normalizeUnitID(label)
	}
	if label == "" || id == "" {
		return unit{}, fmt.Errorf("unit label required")
	}
	share, err := strconv.Atoi(strings.TrimSpace(firstNonEmpty(values.Get("miteigentumsanteil"), "0")))
	if err != nil || share < 0 || share > 1000000 {
		return unit{}, fmt.Errorf("invalid miteigentumsanteil")
	}
	owners, err := emailListFromText(values.Get("owner_emails"))
	if err != nil {
		return unit{}, err
	}
	renters, err := emailListFromText(values.Get("renter_emails"))
	if err != nil {
		return unit{}, err
	}
	return unit{
		ID:                    id,
		TenantSlug:            normalizeSlug(tenantSlug),
		Label:                 label,
		MiteigentumsanteilPPM: share,
		OwnerEmails:           owners,
		RenterEmails:          renters,
	}, nil
}

func emailListFromText(raw string) ([]string, error) {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t'
	})
	emails := make([]string, 0, len(parts))
	for _, part := range parts {
		email := normalizeEmail(part)
		if email == "" {
			continue
		}
		if _, err := mail.ParseAddress(part); err != nil {
			return nil, fmt.Errorf("invalid email")
		}
		emails = append(emails, email)
	}
	return normalizeEmailList(emails), nil
}

func buildingUnitViews(units []unit) []buildingUnitView {
	views := make([]buildingUnitView, 0, len(units))
	for _, item := range units {
		views = append(views, buildingUnitView{
			ID:                 item.ID,
			Label:              item.Label,
			Share:              formatMiteigentumsanteil(item.MiteigentumsanteilPPM),
			ShareValue:         strconv.Itoa(item.MiteigentumsanteilPPM),
			OwnerEmails:        strings.Join(item.OwnerEmails, ", "),
			RenterEmails:       strings.Join(item.RenterEmails, ", "),
			DeleteConfirmLabel: "Einheit " + item.Label + " entfernen",
		})
	}
	return views
}

func buildingSettingsMessage(status string) (string, bool) {
	switch status {
	case "saved":
		return "Gebäudedaten gespeichert.", true
	case "invalid":
		return "Bitte Name, Adresse und Kontaktdaten prüfen.", false
	case "error":
		return "Die Gebäudedaten konnten nicht gespeichert werden.", false
	default:
		return "", false
	}
}

func buildingHeroMessage(status string) (string, bool) {
	switch status {
	case "saved":
		return "Hero-Bild gespeichert.", true
	case "invalid":
		return "Bitte ein JPG-, PNG- oder WebP-Bild bis 5 MB auswählen.", false
	case "error":
		return "Das Hero-Bild konnte nicht gespeichert werden.", false
	default:
		return "", false
	}
}

func buildingUnitMessage(status string) (string, bool) {
	switch status {
	case "saved":
		return "Einheit gespeichert.", true
	case "deleted":
		return "Einheit entfernt.", true
	case "invalid":
		return "Bitte Einheit, Anteil und E-Mail-Links prüfen.", false
	case "duplicate":
		return "Diese Einheit existiert bereits.", false
	case "missing":
		return "Diese Einheit wurde nicht gefunden.", false
	case "error":
		return "Die Einheiten konnten nicht gespeichert werden.", false
	default:
		return "", false
	}
}

func (a *app) profileSettings(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	units := profileUnitViews(a.unitStore.UnitsForEmail(tenant.Slug, email))
	profileMsg, profileOK := profileSettingsMessage(r.URL.Query().Get("profile"))
	a.render(w, "profileSettings", map[string]any{
		"Title":                  "Profil",
		"Tenant":                 tenant,
		"Email":                  email,
		"DisplayName":            profile.DisplayName(),
		"Initials":               profile.Initials(),
		"Role":                   role,
		"IsAdmin":                isAdmin,
		"CanSeeParking":          isAdmin || profile.HasPermission(permissionParking),
		"CanManageAnnouncements": canManageAnnouncements(role),
		"ActivePage":             "settings",
		"Profile":                profile,
		"ProfileMsg":             profileMsg,
		"ProfileOK":              profileOK,
		"PermissionList":         permissionLabelList(profile.Permissions),
		"AuthList":               authMethodsLabelList(profile.AuthMethods),
		"Units":                  units,
		"HasUnits":               len(units) > 0,
	})
}

func (a *app) updateProfileSettings(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, _, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	overlay, err := profileOverlayFromForm(r.Form)
	if err != nil {
		http.Redirect(w, r, "/app/settings/profile?profile=invalid", http.StatusSeeOther)
		return
	}
	if a.profileOverlays != nil {
		if err := a.profileOverlays.Set(email, overlay); err != nil {
			log.Printf("profile save failed for %s: %v", redactedEmail(email), err)
			http.Redirect(w, r, "/app/settings/profile?profile=error", http.StatusSeeOther)
			return
		}
	}
	http.Redirect(w, r, "/app/settings/profile?profile=saved", http.StatusSeeOther)
}

func profileOverlayFromForm(values url.Values) (profileOverlay, error) {
	overlay := profileOverlay{
		Title:          strings.TrimSpace(values.Get("title")),
		FirstName:      strings.TrimSpace(values.Get("first_name")),
		LastName:       strings.TrimSpace(values.Get("last_name")),
		Phone:          strings.TrimSpace(values.Get("phone")),
		DirectoryOptIn: values.Get("directory_opt_in") != "",
	}
	if len([]rune(overlay.Title)) > 40 || len([]rune(overlay.FirstName)) > 120 || len([]rune(overlay.LastName)) > 120 || len([]rune(overlay.Phone)) > 80 {
		return profileOverlay{}, fmt.Errorf("profile field too long")
	}
	return overlay, nil
}

func profileSettingsMessage(status string) (string, bool) {
	switch status {
	case "saved":
		return "Profil gespeichert.", true
	case "invalid":
		return "Bitte die Profildaten prüfen.", false
	case "error":
		return "Das Profil konnte nicht gespeichert werden.", false
	default:
		return "", false
	}
}

func profileUnitViews(units []unitMembership) []profileUnitView {
	views := make([]profileUnitView, 0, len(units))
	for _, membership := range units {
		relation := normalizeRole(membership.Relation)
		if relation == "" {
			relation = membership.Relation
		}
		views = append(views, profileUnitView{
			Label:    membership.Unit.Label,
			Relation: relation,
			Share:    formatMiteigentumsanteil(membership.Unit.MiteigentumsanteilPPM),
		})
	}
	return views
}

func formatMiteigentumsanteil(ppm int) string {
	if ppm <= 0 {
		return "ohne Anteil"
	}
	return formatDecimal(float64(ppm), 0) + " / 1.000.000"
}

func (a *app) notificationSettings(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	prefs := defaultNotificationPreferences()
	if a.notificationPrefs != nil {
		prefs = a.notificationPrefs.Get(email)
	}
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	notifyMsg, notifyOK := notificationSettingsMessage(r.URL.Query().Get("notify"))
	a.render(w, "notificationSettings", map[string]any{
		"Title":                     "Benachrichtigungen",
		"Tenant":                    tenant,
		"Email":                     email,
		"DisplayName":               profile.DisplayName(),
		"Initials":                  profile.Initials(),
		"Role":                      role,
		"IsAdmin":                   isAdmin,
		"CanSeeParking":             isAdmin || profile.HasPermission(permissionParking),
		"ActivePage":                "settings",
		"NotifyMsg":                 notifyMsg,
		"NotifyOK":                  notifyOK,
		"EmailNotificationsEnabled": !prefs.Unsubscribed,
		"NotificationEvents":        notificationEventOptions(prefs),
	})
}

func (a *app) updateNotificationSettings(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, _, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if a.notificationPrefs != nil {
		if err := a.notificationPrefs.Set(email, notificationPreferencesFromForm(r.Form)); err != nil {
			log.Printf("notification preference save failed for %s: %v", redactedEmail(email), err)
			http.Redirect(w, r, "/app/settings/notifications?notify=error", http.StatusSeeOther)
			return
		}
	}
	http.Redirect(w, r, "/app/settings/notifications?notify=saved", http.StatusSeeOther)
}

func notificationSettingsMessage(status string) (string, bool) {
	switch status {
	case "saved":
		return "Benachrichtigungen gespeichert.", true
	case "error":
		return "Benachrichtigungen konnten nicht gespeichert werden.", false
	default:
		return "", false
	}
}

func notificationEventCatalog() []notificationEventOption {
	return []notificationEventOption{
		{Key: notificationEventAnnouncement, Label: "Aushang", Description: "Neue veröffentlichte Aushänge"},
		{Key: notificationEventIssue, Label: "Anliegen", Description: "Neue Anliegen, Kommentare und Statusänderungen"},
		{Key: notificationEventVote, Label: "Abstimmungen", Description: "Neue Abstimmungen und Erinnerungen"},
		{Key: notificationEventDocument, Label: "Dokumente", Description: "Neu bereitgestellte Dokumente"},
		{Key: notificationEventPayment, Label: "Zahlungen", Description: "Fällige oder überfällige Zahlungen"},
	}
}

func notificationEventOptions(prefs notificationPreferences) []notificationEventOption {
	prefs = mergeNotificationPreferences(prefs)
	out := notificationEventCatalog()
	for i := range out {
		out[i].Checked = prefs.Email[out[i].Key]
	}
	return out
}

func notificationPreferencesFromForm(values url.Values) notificationPreferences {
	enabled := map[string]struct{}{}
	for _, raw := range values["events"] {
		event := normalizeNotificationEvent(raw)
		if event != "" {
			enabled[event] = struct{}{}
		}
	}
	prefs := notificationPreferences{
		Email:        map[string]bool{},
		Unsubscribed: values.Get("email_enabled") == "",
	}
	for _, event := range notificationEventCatalog() {
		_, ok := enabled[event.Key]
		prefs.Email[event.Key] = ok
	}
	return prefs
}

func defaultNotificationPreferences() notificationPreferences {
	prefs := notificationPreferences{Email: map[string]bool{}}
	for _, event := range notificationEventCatalog() {
		prefs.Email[event.Key] = true
	}
	return prefs
}

func mergeNotificationPreferences(prefs notificationPreferences) notificationPreferences {
	merged := defaultNotificationPreferences()
	merged.Unsubscribed = prefs.Unsubscribed
	for event, enabled := range prefs.Email {
		event = normalizeNotificationEvent(event)
		if event == "" {
			continue
		}
		merged.Email[event] = enabled
	}
	return merged
}

func normalizeNotificationPreferences(prefs notificationPreferences) notificationPreferences {
	normalized := notificationPreferences{
		Email:        map[string]bool{},
		Unsubscribed: prefs.Unsubscribed,
	}
	merged := mergeNotificationPreferences(prefs)
	for _, event := range notificationEventCatalog() {
		normalized.Email[event.Key] = merged.Email[event.Key]
	}
	return normalized
}

func normalizeNotificationEvent(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	for _, event := range notificationEventCatalog() {
		if raw == event.Key {
			return event.Key
		}
	}
	return ""
}

func (a *app) userSettings(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !hasCapability(role, capabilityManageUsers) {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	users := a.userRows(tenant.Slug)
	inviteMsg, inviteOK := inviteMessage(r.URL.Query().Get("invite"))
	a.render(w, "userSettings", map[string]any{
		"Title":         "Benutzer & Rechte",
		"Tenant":        tenant,
		"Email":         email,
		"DisplayName":   profile.DisplayName(),
		"Initials":      profile.Initials(),
		"Role":          role,
		"Users":         users,
		"HasUsers":      len(users) > 0,
		"UsersEmpty":    emptyStateAction("Noch keine Zugänge", "Sobald eine Person eingeladen ist, erscheint sie hier mit Rolle, Rechten und Anmeldestatus.", "/app/settings/users", "Person einladen"),
		"InviteMsg":     inviteMsg,
		"InviteOK":      inviteOK,
		"IsAdmin":       hasCapability(role, capabilityPlatformAdmin),
		"CanSeeParking": hasCapability(role, capabilityPlatformAdmin) || profile.HasPermission(permissionParking),
		"ActivePage":    "users",
	})
}

func inviteMessage(status string) (string, bool) {
	switch status {
	case "invited":
		return "Einladung gespeichert und per E-Mail verschickt.", true
	case "saved_no_mail":
		return "Einladung gespeichert. Die E-Mail konnte nicht zugestellt werden.", false
	case "exists":
		return "Diese E-Mail-Adresse ist bereits eingetragen.", false
	case "invalid_email":
		return "Bitte eine gültige E-Mail-Adresse angeben.", false
	case "error":
		return "Die Einladung konnte nicht gespeichert werden.", false
	case "updated":
		return "Änderungen gespeichert.", true
	case "deleted":
		return "Zugang gelöscht.", true
	case "not_editable":
		return "Dieser Eintrag kommt aus der Konfiguration und kann hier nicht geändert werden.", false
	case "forbidden_role":
		return "Nur Plattform-Admins können die Admin-Rolle vergeben.", false
	default:
		return "", false
	}
}

func canAssignUserRole(actorRole string, targetRole string) bool {
	targetRole = normalizeRole(targetRole)
	if targetRole == roleAdmin {
		return hasCapability(actorRole, capabilityPlatformAdmin)
	}
	return hasCapability(actorRole, capabilityManageUsers)
}

func auditChangedUserFields(before userProfile, after userProfile) []string {
	changed := []string{}
	if normalizeEmail(before.Email) != normalizeEmail(after.Email) {
		changed = append(changed, "E-Mail")
	}
	if strings.TrimSpace(before.Title) != strings.TrimSpace(after.Title) {
		changed = append(changed, "Titel")
	}
	if strings.TrimSpace(before.FirstName) != strings.TrimSpace(after.FirstName) || strings.TrimSpace(before.LastName) != strings.TrimSpace(after.LastName) {
		changed = append(changed, "Name")
	}
	if normalizeRole(before.Role) != normalizeRole(after.Role) {
		changed = append(changed, "Rolle")
	}
	if strings.Join(normalizePermissions(before.Permissions), ",") != strings.Join(normalizePermissions(after.Permissions), ",") {
		changed = append(changed, "Rechte")
	}
	if len(changed) == 0 {
		changed = append(changed, "Metadaten")
	}
	return changed
}

func (a *app) createInvite(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	actorEmail, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !hasCapability(role, capabilityManageUsers) {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	inviteEmail := normalizeEmail(r.FormValue("email"))
	if _, err := mail.ParseAddress(inviteEmail); err != nil {
		a.redirectInvite(w, r, "invalid_email")
		return
	}
	if _, exists := a.profiles[inviteEmail]; exists {
		a.redirectInvite(w, r, "exists")
		return
	}

	inviteRole := normalizeRole(r.FormValue("role"))
	if inviteRole == "" {
		inviteRole = roleResident
	}
	if !canAssignUserRole(role, inviteRole) {
		a.redirectInvite(w, r, "forbidden_role")
		return
	}
	profile := userProfile{
		Email:       inviteEmail,
		Title:       strings.TrimSpace(r.FormValue("title")),
		FirstName:   strings.TrimSpace(r.FormValue("first_name")),
		LastName:    strings.TrimSpace(r.FormValue("last_name")),
		Role:        inviteRole,
		Status:      "Eingeladen",
		Tenants:     []string{tenant.Slug},
		Permissions: parsePermissionForm(r.Form),
		AuthMethods: defaultAuthMethods(),
	}

	added, err := a.inviteStore.Add(profile)
	if err != nil {
		log.Printf("invite persistence failed for %s: %v", redactedEmail(inviteEmail), err)
		a.redirectInvite(w, r, "error")
		return
	}
	if !added {
		a.redirectInvite(w, r, "exists")
		return
	}

	loginURL := a.publicBaseURL(r, tenant) + "/"
	if err := a.mailer.SendInvite(inviteEmail, loginURL, tenant.Address); err != nil {
		log.Printf("invite email delivery failed for %s: %v", redactedEmail(inviteEmail), err)
		a.recordAudit(auditEvent{
			TenantSlug: tenant.Slug,
			ActorEmail: actorEmail,
			ActorRole:  role,
			Action:     auditActionInviteCreate,
			TargetType: "user",
			TargetID:   inviteEmail,
			Summary:    "Einladung gespeichert",
			Details: map[string]string{
				"role_to":        profile.Role,
				"permissions_to": strings.Join(permissionLabelList(profile.Permissions), ", "),
				"mail_status":    "nicht zugestellt",
			},
		})
		a.redirectInvite(w, r, "saved_no_mail")
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionInviteCreate,
		TargetType: "user",
		TargetID:   inviteEmail,
		Summary:    "Einladung gespeichert",
		Details: map[string]string{
			"role_to":        profile.Role,
			"permissions_to": strings.Join(permissionLabelList(profile.Permissions), ", "),
			"mail_status":    "verschickt",
		},
	})
	a.redirectInvite(w, r, "invited")
}

func (a *app) redirectInvite(w http.ResponseWriter, r *http.Request, status string) {
	http.Redirect(w, r, "/app/settings/users?invite="+url.QueryEscape(status), http.StatusSeeOther)
}

func (a *app) editInvite(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	actorEmail, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !hasCapability(role, capabilityManageUsers) {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	orig := normalizeEmail(r.FormValue("orig_email"))
	existing, isInvite := a.inviteStore.Get(orig)
	if !isInvite {
		// Only persisted invites are editable; env-config users are read-only.
		a.redirectInvite(w, r, "not_editable")
		return
	}
	if normalizeRole(existing.Role) == roleAdmin && !hasCapability(role, capabilityPlatformAdmin) {
		a.redirectInvite(w, r, "not_editable")
		return
	}

	newEmail := normalizeEmail(r.FormValue("email"))
	if _, err := mail.ParseAddress(newEmail); err != nil {
		a.redirectInvite(w, r, "invalid_email")
		return
	}
	if newEmail != orig {
		if _, inEnv := a.profiles[newEmail]; inEnv {
			a.redirectInvite(w, r, "exists")
			return
		}
	}

	newRole := normalizeRole(r.FormValue("role"))
	if newRole == "" {
		newRole = roleResident
	}
	if !canAssignUserRole(role, newRole) {
		a.redirectInvite(w, r, "forbidden_role")
		return
	}
	updated := existing
	updated.Email = newEmail
	updated.Title = strings.TrimSpace(r.FormValue("title"))
	updated.FirstName = strings.TrimSpace(r.FormValue("first_name"))
	updated.LastName = strings.TrimSpace(r.FormValue("last_name"))
	updated.Role = newRole
	updated.Permissions = parsePermissionForm(r.Form)
	if len(updated.Tenants) == 0 {
		updated.Tenants = []string{tenant.Slug}
	}
	if len(updated.AuthMethods) == 0 {
		updated.AuthMethods = defaultAuthMethods()
	}

	changed, err := a.inviteStore.Update(orig, updated)
	if err != nil {
		a.redirectInvite(w, r, "exists")
		return
	}
	if !changed {
		a.redirectInvite(w, r, "not_editable")
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionInviteUpdate,
		TargetType: "user",
		TargetID:   updated.Email,
		Summary:    "Einladung geändert",
		Details: map[string]string{
			"changed_fields":   strings.Join(auditChangedUserFields(existing, updated), ", "),
			"role_from":        normalizeRole(existing.Role),
			"role_to":          normalizeRole(updated.Role),
			"permissions_from": strings.Join(permissionLabelList(existing.Permissions), ", "),
			"permissions_to":   strings.Join(permissionLabelList(updated.Permissions), ", "),
		},
	})
	a.redirectInvite(w, r, "updated")
}

func (a *app) deleteInvite(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	actorEmail, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !hasCapability(role, capabilityManageUsers) {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	deleteEmail := normalizeEmail(r.FormValue("email"))
	existing, isInvite := a.inviteStore.Get(deleteEmail)
	if !isInvite {
		a.redirectInvite(w, r, "not_editable")
		return
	}
	if normalizeRole(existing.Role) == roleAdmin && !hasCapability(role, capabilityPlatformAdmin) {
		a.redirectInvite(w, r, "not_editable")
		return
	}
	removed, err := a.inviteStore.Delete(deleteEmail)
	if err != nil {
		a.redirectInvite(w, r, "error")
		return
	}
	if !removed {
		a.redirectInvite(w, r, "not_editable")
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionInviteDelete,
		TargetType: "user",
		TargetID:   deleteEmail,
		Summary:    "Einladung gelöscht",
		Details: map[string]string{
			"role_from":        normalizeRole(existing.Role),
			"permissions_from": strings.Join(permissionLabelList(existing.Permissions), ", "),
		},
	})
	a.redirectInvite(w, r, "deleted")
}

func (a *app) render(w http.ResponseWriter, name string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	if _, ok := data["AppVersion"]; !ok {
		data["AppVersion"] = buildLabel()
	}
	enrichCapabilityData(data)
	a.enrichUnreadAnnouncementData(data)
	a.enrichIssueData(data)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render %s failed: %v", name, err)
	}
}

func enrichCapabilityData(data map[string]any) {
	role, _ := data["Role"].(string)
	if role == "" {
		return
	}
	if _, ok := data["IsAdmin"]; !ok {
		data["IsAdmin"] = hasCapability(role, capabilityPlatformAdmin)
	}
	if _, ok := data["CanManageUsers"]; !ok {
		data["CanManageUsers"] = hasCapability(role, capabilityManageUsers)
	}
	if _, ok := data["CanManageAnnouncements"]; !ok {
		data["CanManageAnnouncements"] = hasCapability(role, capabilityManageAnnouncements)
	}
	if _, ok := data["CanManageDocuments"]; !ok {
		data["CanManageDocuments"] = hasCapability(role, capabilityManageDocuments)
	}
	if _, ok := data["CanManageBuilding"]; !ok {
		data["CanManageBuilding"] = hasCapability(role, capabilityManageBuilding)
	}
	if _, ok := data["CanViewAudit"]; !ok {
		data["CanViewAudit"] = canViewAudit(role)
	}
}

func (a *app) enrichUnreadAnnouncementData(data map[string]any) {
	if _, ok := data["UnreadAnnouncements"]; ok {
		if _, hasFlag := data["HasUnreadAnnouncements"]; !hasFlag {
			if count, ok := data["UnreadAnnouncements"].(int); ok {
				data["HasUnreadAnnouncements"] = count > 0
			}
		}
		return
	}
	tenant, ok := data["Tenant"].(tenantConfig)
	if !ok || tenant.Slug == "" || a.announcementStore == nil || a.announcementReadStore == nil {
		data["UnreadAnnouncements"] = 0
		data["HasUnreadAnnouncements"] = false
		return
	}
	email, ok := data["Email"].(string)
	if !ok || strings.TrimSpace(email) == "" {
		data["UnreadAnnouncements"] = 0
		data["HasUnreadAnnouncements"] = false
		return
	}
	now := time.Now()
	lastSeen := a.announcementReadStore.LastSeen(tenant.Slug, email)
	count := unreadAnnouncementCount(a.announcementStore.Visible(tenant.Slug, now), lastSeen, now)
	data["UnreadAnnouncements"] = count
	data["HasUnreadAnnouncements"] = count > 0
}

func (a *app) enrichIssueData(data map[string]any) {
	if _, ok := data["OpenIssues"]; ok {
		if _, hasFlag := data["HasOpenIssues"]; !hasFlag {
			if count, ok := data["OpenIssues"].(int); ok {
				data["HasOpenIssues"] = count > 0
			}
		}
		return
	}
	tenant, ok := data["Tenant"].(tenantConfig)
	if !ok || tenant.Slug == "" || a.issueStore == nil {
		data["OpenIssues"] = 0
		data["HasOpenIssues"] = false
		return
	}
	email, _ := data["Email"].(string)
	role, _ := data["Role"].(string)
	count := issueOpenCount(a.visibleIssuesForActor(tenant.Slug, email, role))
	data["OpenIssues"] = count
	data["HasOpenIssues"] = count > 0
}

func buildLabel() string {
	version := strings.TrimPrefix(strings.TrimSpace(appVersion), "v")
	if version == "" {
		version = "0.1.0"
	}
	commit := strings.TrimSpace(gitCommit)
	if commit == "" {
		commit = "dev"
	}
	return fmt.Sprintf("%s (%s)", version, commit)
}

func emptyState(title string, message string) emptyStateView {
	return emptyStateView{Title: title, Message: message}
}

func emptyStateAction(title string, message string, actionURL string, actionLabel string) emptyStateView {
	state := emptyState(title, message)
	state.ActionURL = actionURL
	state.ActionLabel = actionLabel
	state.HasAction = actionURL != "" && actionLabel != ""
	return state
}

func (a *app) tenantPathRedirect(w http.ResponseWriter, r *http.Request) {
	slug := normalizeSlug(r.PathValue("tenant"))
	tenant, ok := a.tenantBySlug(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	rest := strings.TrimLeft(r.PathValue("rest"), "/")
	targetPath := "/"
	if rest != "" {
		targetPath += rest
	}
	target := tenant.PublicURL(targetPath)
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, target, http.StatusMovedPermanently)
}

func (a *app) tenantHeroImage(w http.ResponseWriter, r *http.Request) {
	slug := normalizeSlug(r.PathValue("tenant"))
	if slug == "" {
		http.NotFound(w, r)
		return
	}
	if _, ok := a.tenants[slug]; !ok {
		http.NotFound(w, r)
		return
	}
	override, ok := a.tenantOverrides.Get(slug)
	if !ok || override.HeroImage == "" {
		http.NotFound(w, r)
		return
	}
	filename := filepath.Base(override.HeroImage)
	if filename == "" || filename == "." || filename != override.HeroImage {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(a.tenantHeroDir, filename))
}

func (a *app) tenantForRequest(r *http.Request) tenantConfig {
	host := normalizeHost(r.Host)
	for _, tenant := range a.tenants {
		if tenant.Host != "" && tenant.Host == host {
			return a.withTenantOverride(tenant)
		}
	}
	if a.rootDomain != "" && strings.HasSuffix(host, "."+a.rootDomain) {
		slug := normalizeSlug(strings.TrimSuffix(host, "."+a.rootDomain))
		if tenant, ok := a.tenantBySlug(slug); ok {
			return tenant
		}
	}
	tenant, _ := a.tenantBySlug(a.defaultTenant)
	return tenant
}

func (a *app) tenantBySlug(slug string) (tenantConfig, bool) {
	tenant, ok := a.tenants[normalizeSlug(slug)]
	if !ok {
		return tenantConfig{}, false
	}
	return a.withTenantOverride(tenant), true
}

func (a *app) withTenantOverride(tenant tenantConfig) tenantConfig {
	if strings.TrimSpace(tenant.HeroImageURL) == "" {
		tenant.HeroImageURL = defaultTenantHeroImageURL
	}
	if a.tenantOverrides == nil {
		return tenant
	}
	override, ok := a.tenantOverrides.Get(tenant.Slug)
	if !ok {
		return tenant
	}
	if override.MetaSet {
		if override.Name != "" {
			tenant.Name = override.Name
		}
		if override.Address != "" {
			tenant.Address = override.Address
		}
		tenant.ContactName = override.ContactName
		tenant.ContactEmail = override.ContactEmail
		tenant.ContactPhone = override.ContactPhone
		tenant.EmergencyName = override.EmergencyName
		tenant.EmergencyPhone = override.EmergencyPhone
		tenant.CaretakerName = override.CaretakerName
		tenant.CaretakerEmail = override.CaretakerEmail
		tenant.CaretakerPhone = override.CaretakerPhone
	}
	if override.HeroImage != "" {
		tenant.HeroImageURL = "/tenant-hero/" + tenant.Slug
	}
	return tenant
}

func (a *app) publicBaseURL(r *http.Request, tenant tenantConfig) string {
	host := normalizeHost(r.Host)
	if isLocalHost(host) {
		return a.baseURL
	}
	if tenant.Host != "" {
		return "https://" + tenant.Host
	}
	return a.baseURL
}

func (t tenantConfig) PublicURL(path string) string {
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if t.Host == "" {
		return path
	}
	return "https://" + t.Host + path
}

func (a *app) currentUser(r *http.Request) (string, string, string, bool) {
	c, err := r.Cookie("weg_session")
	if err != nil {
		return "", "", "", false
	}
	email, tenantSlug, authMethod, ok := a.sessions.Get(c.Value)
	if !ok {
		return "", "", "", false
	}
	if !a.isAllowed(email, tenantSlug) || !a.isAuthMethodAllowed(email, tenantSlug, authMethod) {
		return "", "", "", false
	}
	return email, a.roleFor(email, tenantSlug), tenantSlug, true
}

// directoryProfile resolves a profile from the env directory first, then falls
// back to persisted invites. Env is authoritative: an invite can never override
// or escalate an env-defined user, so existing logins are unaffected.
func (a *app) directoryProfile(email string) (userProfile, bool) {
	email = normalizeEmail(email)
	if profile, ok := a.profiles[email]; ok {
		return a.withProfileOverlay(profile), true
	}
	if a.inviteStore != nil {
		if profile, ok := a.inviteStore.Get(email); ok {
			return a.withProfileOverlay(profile), true
		}
	}
	return userProfile{}, false
}

func (a *app) withProfileOverlay(profile userProfile) userProfile {
	if a.profileOverlays == nil {
		return profile
	}
	overlay, ok := a.profileOverlays.Get(profile.Email)
	if !ok {
		return profile
	}
	profile.Title = overlay.Title
	profile.FirstName = overlay.FirstName
	profile.LastName = overlay.LastName
	profile.Phone = overlay.Phone
	profile.DirectoryOptIn = overlay.DirectoryOptIn
	return profile
}

func (a *app) isAllowed(email string, tenantSlug string) bool {
	if profile, ok := a.directoryProfile(email); ok && profile.HasTenant(tenantSlug) {
		return true
	}
	if _, ok := a.admins[email]; ok {
		return true
	}
	_, ok := a.allowed[email]
	return ok
}

func (a *app) isAuthMethodAllowed(email string, tenantSlug string, authMethod string) bool {
	authMethod = normalizeAuthMethod(authMethod)
	if authMethod == "" {
		return false
	}
	profile, ok := a.directoryProfile(email)
	if !ok || !profile.HasTenant(tenantSlug) {
		return false
	}
	return profile.AllowsAuthMethod(authMethod)
}

func (a *app) emailLoginAvailable() bool {
	return a.mailer.Configured() || a.localDevLogin
}

func (a *app) roleFor(email string, tenantSlug string) string {
	if profile, ok := a.directoryProfile(email); ok {
		profile = profile.ForTenant(tenantSlug)
		if profile.Role != "" {
			return normalizeRole(profile.Role)
		}
	}
	if _, ok := a.admins[email]; ok {
		return roleAdmin
	}
	return roleResident
}

func (a *app) profileFor(email string) userProfile {
	return a.profileForTenant(email, a.defaultTenant)
}

func (a *app) profileForTenant(email string, tenantSlug string) userProfile {
	email = normalizeEmail(email)
	tenantSlug = normalizeSlug(tenantSlug)
	if tenantSlug == "" {
		tenantSlug = a.defaultTenant
	}
	if profile, ok := a.directoryProfile(email); ok {
		return profile.ForTenant(tenantSlug)
	}
	role := a.roleFor(email, tenantSlug)
	return userProfile{
		Email:       email,
		Role:        role,
		Status:      "Eingeladen",
		Tenants:     []string{tenantSlug},
		AuthMethods: defaultAuthMethods(),
	}
}

func (a *app) userRows(tenantSlug string) []userRow {
	seen := map[string]struct{}{}
	rows := make([]userRow, 0, len(a.profiles)+len(a.admins)+len(a.allowed))
	for email, profile := range a.profiles {
		profile = a.withProfileOverlay(profile)
		if !profile.HasTenant(tenantSlug) {
			continue
		}
		rows = append(rows, profile.ForTenant(tenantSlug).UserRow())
		seen[email] = struct{}{}
	}
	for email := range a.admins {
		if _, ok := seen[email]; ok {
			continue
		}
		rows = append(rows, userProfile{Email: email, Role: roleAdmin, Status: "Aktiv", Tenants: []string{tenantSlug}}.UserRow())
		seen[email] = struct{}{}
	}
	for email := range a.allowed {
		if _, ok := seen[email]; ok {
			continue
		}
		rows = append(rows, userProfile{Email: email, Role: roleResident, Status: "Eingeladen", Tenants: []string{tenantSlug}}.UserRow())
		seen[email] = struct{}{}
	}
	if a.inviteStore != nil {
		for _, profile := range a.inviteStore.List() {
			profile = a.withProfileOverlay(profile)
			email := normalizeEmail(profile.Email)
			if _, ok := seen[email]; ok {
				continue
			}
			if !profile.HasTenant(tenantSlug) {
				continue
			}
			row := profile.ForTenant(tenantSlug).UserRow()
			row.Editable = true
			rows = append(rows, row)
			seen[email] = struct{}{}
		}
	}
	if a.activityStore != nil {
		for i := range rows {
			if !strings.Contains(rows[i].Email, "@") {
				continue
			}
			if rec, ok := a.activityStore.Get(rows[i].Email); ok {
				rows[i].Status = "Aktiv"
				rows[i].LastSeen = "zuletzt angemeldet: " + formatLocalDate(rec.LastLogin)
			} else if rows[i].Status == "Eingeladen" {
				rows[i].LastSeen = "noch nie angemeldet"
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Role != rows[j].Role {
			return roleSortRank(rows[i].Role) < roleSortRank(rows[j].Role)
		}
		if rows[i].LastName != rows[j].LastName {
			return rows[i].LastName < rows[j].LastName
		}
		return rows[i].Email < rows[j].Email
	})
	return rows
}

func newAnnouncementStore(path string) (*announcementStore, error) {
	store := &announcementStore{path: path, data: announcementStoreData{Announcements: []announcement{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read announcement data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid announcement data")
	}
	return store, nil
}

func newAnnouncementReadStore(path string) (*announcementReadStore, error) {
	store := &announcementReadStore{path: path, data: announcementReadStoreData{Seen: map[string]map[string]time.Time{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read announcement read data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid announcement read data")
	}
	if store.data.Seen == nil {
		store.data.Seen = map[string]map[string]time.Time{}
	}
	return store, nil
}

func newEventStore(path string) (*eventStore, error) {
	store := &eventStore{path: path, data: eventStoreData{Events: []houseEvent{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read event data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid event data")
	}
	events := make([]houseEvent, 0, len(store.data.Events))
	for _, item := range store.data.Events {
		normalized, ok := normalizeHouseEvent(item)
		if ok {
			events = append(events, normalized)
		}
	}
	store.data.Events = events
	sortEvents(store.data.Events)
	return store, nil
}

func (s *announcementReadStore) LastSeen(tenantSlug string, email string) time.Time {
	if s == nil {
		return time.Time{}
	}
	tenantSlug = normalizeSlug(tenantSlug)
	email = normalizeEmail(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	if tenantSlug == "" || email == "" {
		return time.Time{}
	}
	return s.data.Seen[tenantSlug][email]
}

func (s *announcementReadStore) MarkSeen(tenantSlug string, email string, seenAt time.Time) error {
	if s == nil {
		return nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	email = normalizeEmail(email)
	if tenantSlug == "" || email == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Seen == nil {
		s.data.Seen = map[string]map[string]time.Time{}
	}
	if s.data.Seen[tenantSlug] == nil {
		s.data.Seen[tenantSlug] = map[string]time.Time{}
	}
	s.data.Seen[tenantSlug][email] = seenAt.UTC()
	return s.saveLocked()
}

func (s *announcementReadStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("could not create announcement read data directory")
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode announcement read data")
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("could not write announcement read data")
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("could not replace announcement read data")
	}
	return nil
}

func newNotificationPrefStore(path string) (*notificationPrefStore, error) {
	store := &notificationPrefStore{path: path, data: notificationPrefStoreData{Users: map[string]notificationPreferences{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read notification preference data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid notification preference data")
	}
	if store.data.Users == nil {
		store.data.Users = map[string]notificationPreferences{}
	}
	normalized := map[string]notificationPreferences{}
	for email, prefs := range store.data.Users {
		email = normalizeEmail(email)
		if email == "" {
			continue
		}
		normalized[email] = normalizeNotificationPreferences(prefs)
	}
	store.data.Users = normalized
	return store, nil
}

func newProfileOverlayStore(path string) (*profileOverlayStore, error) {
	store := &profileOverlayStore{path: path, data: profileOverlayStoreData{Profiles: map[string]profileOverlay{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read profile data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid profile data")
	}
	if store.data.Profiles == nil {
		store.data.Profiles = map[string]profileOverlay{}
	}
	normalized := map[string]profileOverlay{}
	for email, overlay := range store.data.Profiles {
		email = normalizeEmail(email)
		if email == "" {
			continue
		}
		normalized[email] = normalizeProfileOverlay(overlay)
	}
	store.data.Profiles = normalized
	return store, nil
}

func (s *profileOverlayStore) Get(email string) (profileOverlay, bool) {
	if s == nil {
		return profileOverlay{}, false
	}
	email = normalizeEmail(email)
	if email == "" {
		return profileOverlay{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	overlay, ok := s.data.Profiles[email]
	return overlay, ok
}

func (s *profileOverlayStore) Set(email string, overlay profileOverlay) error {
	if s == nil {
		return nil
	}
	email = normalizeEmail(email)
	if email == "" {
		return fmt.Errorf("invalid profile email")
	}
	overlay = normalizeProfileOverlay(overlay)
	overlay.UpdatedAt = time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Profiles == nil {
		s.data.Profiles = map[string]profileOverlay{}
	}
	s.data.Profiles[email] = overlay
	return s.saveLocked()
}

func (s *profileOverlayStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("could not create profile data directory")
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode profile data")
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("could not write profile data")
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("could not replace profile data")
	}
	return nil
}

func normalizeProfileOverlay(overlay profileOverlay) profileOverlay {
	overlay.Title = strings.TrimSpace(overlay.Title)
	overlay.FirstName = strings.TrimSpace(overlay.FirstName)
	overlay.LastName = strings.TrimSpace(overlay.LastName)
	overlay.Phone = strings.TrimSpace(overlay.Phone)
	if !overlay.UpdatedAt.IsZero() {
		overlay.UpdatedAt = overlay.UpdatedAt.UTC()
	}
	return overlay
}

func newTenantOverrideStore(path string) (*tenantOverrideStore, error) {
	store := &tenantOverrideStore{path: path, data: tenantOverrideStoreData{Tenants: map[string]tenantOverride{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read tenant data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid tenant data")
	}
	if store.data.Tenants == nil {
		store.data.Tenants = map[string]tenantOverride{}
	}
	normalized := map[string]tenantOverride{}
	for slug, override := range store.data.Tenants {
		slug = normalizeSlug(slug)
		if slug == "" {
			continue
		}
		normalized[slug] = normalizeTenantOverride(override)
	}
	store.data.Tenants = normalized
	return store, nil
}

func (s *tenantOverrideStore) Get(tenantSlug string) (tenantOverride, bool) {
	if s == nil {
		return tenantOverride{}, false
	}
	tenantSlug = normalizeSlug(tenantSlug)
	if tenantSlug == "" {
		return tenantOverride{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	override, ok := s.data.Tenants[tenantSlug]
	return override, ok
}

func (s *tenantOverrideStore) Set(tenantSlug string, override tenantOverride) error {
	override.MetaSet = true
	return s.set(tenantSlug, override)
}

func (s *tenantOverrideStore) SetMeta(tenantSlug string, meta tenantOverride) error {
	if s == nil {
		return nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	if tenantSlug == "" {
		return fmt.Errorf("invalid tenant")
	}
	meta = normalizeTenantOverride(meta)
	meta.MetaSet = true
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Tenants == nil {
		s.data.Tenants = map[string]tenantOverride{}
	}
	current := normalizeTenantOverride(s.data.Tenants[tenantSlug])
	meta.HeroImage = current.HeroImage
	meta.UpdatedAt = time.Now().UTC()
	s.data.Tenants[tenantSlug] = meta
	return s.saveLocked()
}

func (s *tenantOverrideStore) SetHeroImage(tenantSlug string, filename string) error {
	if s == nil {
		return nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	filename = filepath.Base(strings.TrimSpace(filename))
	if tenantSlug == "" || filename == "." || filename == string(filepath.Separator) || filename == "" {
		return fmt.Errorf("invalid tenant hero image")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Tenants == nil {
		s.data.Tenants = map[string]tenantOverride{}
	}
	override := normalizeTenantOverride(s.data.Tenants[tenantSlug])
	override.HeroImage = filename
	override.UpdatedAt = time.Now().UTC()
	s.data.Tenants[tenantSlug] = override
	return s.saveLocked()
}

func (s *tenantOverrideStore) set(tenantSlug string, override tenantOverride) error {
	if s == nil {
		return nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	if tenantSlug == "" {
		return fmt.Errorf("invalid tenant")
	}
	override = normalizeTenantOverride(override)
	override.UpdatedAt = time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Tenants == nil {
		s.data.Tenants = map[string]tenantOverride{}
	}
	s.data.Tenants[tenantSlug] = override
	return s.saveLocked()
}

func (s *tenantOverrideStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("could not create tenant data directory")
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode tenant data")
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("could not write tenant data")
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("could not replace tenant data")
	}
	return nil
}

func normalizeTenantOverride(override tenantOverride) tenantOverride {
	override.Name = strings.TrimSpace(override.Name)
	override.Address = strings.TrimSpace(override.Address)
	override.ContactName = strings.TrimSpace(override.ContactName)
	override.ContactEmail = normalizeEmail(override.ContactEmail)
	override.ContactPhone = strings.TrimSpace(override.ContactPhone)
	override.EmergencyName = strings.TrimSpace(override.EmergencyName)
	override.EmergencyPhone = strings.TrimSpace(override.EmergencyPhone)
	override.CaretakerName = strings.TrimSpace(override.CaretakerName)
	override.CaretakerEmail = normalizeEmail(override.CaretakerEmail)
	override.CaretakerPhone = strings.TrimSpace(override.CaretakerPhone)
	override.HeroImage = filepath.Base(strings.TrimSpace(override.HeroImage))
	if override.HeroImage == "." || override.HeroImage == string(filepath.Separator) {
		override.HeroImage = ""
	}
	if !override.UpdatedAt.IsZero() {
		override.UpdatedAt = override.UpdatedAt.UTC()
	}
	return override
}

func (s *notificationPrefStore) Get(email string) notificationPreferences {
	prefs := defaultNotificationPreferences()
	if s == nil {
		return prefs
	}
	email = normalizeEmail(email)
	if email == "" {
		return prefs
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if stored, ok := s.data.Users[email]; ok {
		prefs = mergeNotificationPreferences(stored)
	}
	return prefs
}

func (s *notificationPrefStore) Set(email string, prefs notificationPreferences) error {
	if s == nil {
		return nil
	}
	email = normalizeEmail(email)
	if email == "" {
		return fmt.Errorf("invalid notification preference email")
	}
	prefs = normalizeNotificationPreferences(prefs)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Users == nil {
		s.data.Users = map[string]notificationPreferences{}
	}
	s.data.Users[email] = prefs
	return s.saveLocked()
}

func (s *notificationPrefStore) EmailEnabled(email string, event string) bool {
	event = normalizeNotificationEvent(event)
	if event == "" {
		return false
	}
	prefs := defaultNotificationPreferences()
	if s != nil {
		prefs = s.Get(email)
	}
	if prefs.Unsubscribed {
		return false
	}
	if prefs.Email == nil {
		return true
	}
	enabled, ok := prefs.Email[event]
	if !ok {
		return true
	}
	return enabled
}

func (s *notificationPrefStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("could not create notification preference data directory")
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode notification preference data")
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("could not write notification preference data")
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("could not replace notification preference data")
	}
	return nil
}

func newIssueStore(path string, attachmentDir string) (*issueStore, error) {
	if attachmentDir == "" && path != "" {
		attachmentDir = filepath.Join(filepath.Dir(path), "issue-attachments")
	}
	store := &issueStore{path: path, attachmentDir: attachmentDir, data: issueStoreData{Issues: []residentIssue{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read issue data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid issue data")
	}
	if store.data.Issues == nil {
		store.data.Issues = []residentIssue{}
	}
	return store, nil
}

func (s *issueStore) Create(item residentIssue) (residentIssue, error) {
	if s == nil {
		return item, nil
	}
	now := time.Now().UTC()
	if item.ID == "" {
		id, err := randomToken(12)
		if err != nil {
			return residentIssue{}, err
		}
		item.ID = id
	}
	item.TenantSlug = normalizeSlug(item.TenantSlug)
	item.AuthorEmail = normalizeEmail(item.AuthorEmail)
	item.Category = normalizeIssueCategory(item.Category)
	item.LocationType = normalizeIssueLocation(item.LocationType)
	item.Title = strings.TrimSpace(item.Title)
	item.Body = strings.TrimSpace(item.Body)
	item.LocationDetail = strings.TrimSpace(item.LocationDetail)
	if item.TenantSlug == "" || item.AuthorEmail == "" || item.Category == "" || item.LocationType == "" || item.Title == "" || item.Body == "" {
		return residentIssue{}, fmt.Errorf("invalid issue")
	}
	item.Status = normalizeIssueStatus(item.Status)
	if item.Status == "" {
		item.Status = issueStatusOpen
	}
	item.Priority = normalizeIssuePriority(item.Priority)
	if item.Priority == "" {
		item.Priority = issuePriorityNorm
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	} else {
		item.CreatedAt = item.CreatedAt.UTC()
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = item.CreatedAt
	} else {
		item.UpdatedAt = item.UpdatedAt.UTC()
	}
	if item.StatusChangedAt.IsZero() {
		item.StatusChangedAt = item.CreatedAt
	} else {
		item.StatusChangedAt = item.StatusChangedAt.UTC()
	}
	if item.StatusChangedBy == "" {
		item.StatusChangedBy = item.AuthorEmail
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Issues = append(s.data.Issues, item)
	if err := s.saveLocked(); err != nil {
		return residentIssue{}, err
	}
	return item, nil
}

func (s *issueStore) ListTenant(tenantSlug string) []residentIssue {
	if s == nil {
		return nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []residentIssue{}
	for _, item := range s.data.Issues {
		if normalizeSlug(item.TenantSlug) == tenantSlug {
			out = append(out, copyIssue(item))
		}
	}
	sortIssues(out)
	return out
}

func (s *issueStore) ListAuthor(tenantSlug string, email string) []residentIssue {
	if s == nil {
		return nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	email = normalizeEmail(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []residentIssue{}
	for _, item := range s.data.Issues {
		if normalizeSlug(item.TenantSlug) == tenantSlug && normalizeEmail(item.AuthorEmail) == email {
			out = append(out, copyIssue(item))
		}
	}
	sortIssues(out)
	return out
}

func (s *issueStore) Get(tenantSlug string, id string) (residentIssue, bool) {
	if s == nil {
		return residentIssue{}, false
	}
	tenantSlug = normalizeSlug(tenantSlug)
	id = strings.TrimSpace(id)
	if tenantSlug == "" || id == "" {
		return residentIssue{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.Issues {
		if normalizeSlug(item.TenantSlug) == tenantSlug && item.ID == id {
			return copyIssue(item), true
		}
	}
	return residentIssue{}, false
}

func (s *issueStore) UpdateWorkflow(tenantSlug string, id string, update issueWorkflowUpdate) (residentIssue, bool, error) {
	if s == nil {
		return residentIssue{}, false, nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	id = strings.TrimSpace(id)
	status := normalizeIssueStatus(update.Status)
	priority := normalizeIssuePriority(update.Priority)
	if tenantSlug == "" || id == "" || status == "" || priority == "" {
		return residentIssue{}, false, fmt.Errorf("invalid issue workflow update")
	}
	if update.ChangedAt.IsZero() {
		update.ChangedAt = time.Now()
	}
	changedAt := update.ChangedAt.UTC()
	actorEmail := normalizeEmail(update.ActorEmail)
	actorName := strings.TrimSpace(update.ActorName)
	assignee := normalizeEmail(update.AssigneeEmail)

	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Issues {
		if normalizeSlug(existing.TenantSlug) != tenantSlug || existing.ID != id {
			continue
		}
		updated := existing
		oldStatus := normalizeIssueStatus(updated.Status)
		if oldStatus == "" {
			oldStatus = issueStatusOpen
		}
		updated.Status = status
		updated.Priority = priority
		updated.AssigneeEmail = assignee
		updated.UpdatedAt = changedAt
		if oldStatus != status {
			updated.StatusChangedAt = changedAt
			updated.StatusChangedBy = actorEmail
			updated.StatusHistory = append(updated.StatusHistory, issueStatusChange{
				From:       oldStatus,
				To:         status,
				ActorEmail: actorEmail,
				ActorName:  actorName,
				ChangedAt:  changedAt,
			})
		}
		s.data.Issues[i] = updated
		if err := s.saveLocked(); err != nil {
			return residentIssue{}, false, err
		}
		return copyIssue(updated), true, nil
	}
	return residentIssue{}, false, nil
}

func (s *issueStore) AddComment(tenantSlug string, id string, comment issueComment) (residentIssue, bool, error) {
	if s == nil {
		return residentIssue{}, false, nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	id = strings.TrimSpace(id)
	comment.Body = strings.TrimSpace(comment.Body)
	comment.AuthorEmail = normalizeEmail(comment.AuthorEmail)
	comment.AuthorName = strings.TrimSpace(comment.AuthorName)
	if tenantSlug == "" || id == "" || comment.Body == "" || len([]rune(comment.Body)) > 3000 || comment.AuthorEmail == "" {
		return residentIssue{}, false, fmt.Errorf("invalid issue comment")
	}
	if comment.ID == "" {
		commentID, err := randomToken(10)
		if err != nil {
			return residentIssue{}, false, err
		}
		comment.ID = commentID
	}
	if comment.CreatedAt.IsZero() {
		comment.CreatedAt = time.Now().UTC()
	} else {
		comment.CreatedAt = comment.CreatedAt.UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Issues {
		if normalizeSlug(existing.TenantSlug) != tenantSlug || existing.ID != id {
			continue
		}
		updated := existing
		updated.Comments = append(updated.Comments, comment)
		updated.UpdatedAt = comment.CreatedAt
		s.data.Issues[i] = updated
		if err := s.saveLocked(); err != nil {
			return residentIssue{}, false, err
		}
		return copyIssue(updated), true, nil
	}
	return residentIssue{}, false, nil
}

func (a *app) saveTenantHeroImage(tenantSlug string, header *multipart.FileHeader) (string, error) {
	if header == nil || header.Filename == "" || header.Size == 0 {
		return "", fmt.Errorf("tenant hero image required")
	}
	if a.tenantHeroDir == "" {
		return "", fmt.Errorf("tenant hero image directory unavailable")
	}
	if header.Size > maxTenantHeroBytes {
		return "", fmt.Errorf("tenant hero image too large")
	}
	file, err := header.Open()
	if err != nil {
		return "", fmt.Errorf("could not open tenant hero image")
	}
	defer file.Close()

	sniff := make([]byte, 512)
	n, readErr := file.Read(sniff)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return "", fmt.Errorf("could not read tenant hero image")
	}
	contentType := http.DetectContentType(sniff[:n])
	ext, ok := issuePhotoExtension(contentType)
	if !ok {
		return "", fmt.Errorf("unsupported tenant hero image type")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("could not rewind tenant hero image")
	}

	tenantSlug = normalizeSlug(tenantSlug)
	if tenantSlug == "" {
		return "", fmt.Errorf("tenant required")
	}
	if err := os.MkdirAll(a.tenantHeroDir, 0o755); err != nil {
		return "", fmt.Errorf("could not create tenant hero image directory")
	}
	tmp, err := os.CreateTemp(a.tenantHeroDir, tenantSlug+"-hero-*.tmp")
	if err != nil {
		return "", fmt.Errorf("could not create tenant hero image")
	}
	tmpName := tmp.Name()
	written, copyErr := io.Copy(tmp, io.LimitReader(file, maxTenantHeroBytes+1))
	closeErr := tmp.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(tmpName)
		return "", fmt.Errorf("could not write tenant hero image")
	}
	if written > maxTenantHeroBytes {
		_ = os.Remove(tmpName)
		return "", fmt.Errorf("tenant hero image too large")
	}
	filename := tenantSlug + "-hero" + ext
	dest := filepath.Join(a.tenantHeroDir, filename)
	if err := os.Chmod(tmpName, 0o600); err != nil {
		_ = os.Remove(tmpName)
		return "", fmt.Errorf("could not protect tenant hero image")
	}
	if err := os.Rename(tmpName, dest); err != nil {
		_ = os.Remove(tmpName)
		return "", fmt.Errorf("could not replace tenant hero image")
	}
	return filename, nil
}

func (s *issueStore) SavePhoto(tenantSlug string, issueID string, header *multipart.FileHeader) (string, error) {
	if s == nil || header == nil || header.Filename == "" || header.Size == 0 {
		return "", nil
	}
	if s.attachmentDir == "" {
		return "", fmt.Errorf("issue attachment directory unavailable")
	}
	if header.Size > maxIssuePhotoBytes {
		return "", fmt.Errorf("issue photo too large")
	}
	file, err := header.Open()
	if err != nil {
		return "", fmt.Errorf("could not open issue photo")
	}
	defer file.Close()

	sniff := make([]byte, 512)
	n, readErr := file.Read(sniff)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return "", fmt.Errorf("could not read issue photo")
	}
	contentType := http.DetectContentType(sniff[:n])
	ext, ok := issuePhotoExtension(contentType)
	if !ok {
		return "", fmt.Errorf("unsupported issue photo type")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("could not rewind issue photo")
	}

	tenantSlug = normalizeSlug(tenantSlug)
	if tenantSlug == "" {
		tenantSlug = "tenant"
	}
	issueID = normalizeSlug(issueID)
	if issueID == "" {
		return "", fmt.Errorf("issue id required")
	}
	dir := filepath.Join(s.attachmentDir, tenantSlug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("could not create issue attachment directory")
	}
	filename := issueID + "-photo" + ext
	dest := filepath.Join(dir, filename)
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("could not create issue attachment")
	}
	written, copyErr := io.Copy(out, io.LimitReader(file, maxIssuePhotoBytes+1))
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(dest)
		return "", fmt.Errorf("could not write issue attachment")
	}
	if written > maxIssuePhotoBytes {
		_ = os.Remove(dest)
		return "", fmt.Errorf("issue photo too large")
	}
	return filepath.ToSlash(filepath.Join(filepath.Base(s.attachmentDir), tenantSlug, filename)), nil
}

func issuePhotoExtension(contentType string) (string, bool) {
	switch contentType {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}

func (s *issueStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("could not create issue data directory")
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode issue data")
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("could not write issue data")
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("could not replace issue data")
	}
	return nil
}

func copyIssue(item residentIssue) residentIssue {
	item.PhotoPaths = append([]string(nil), item.PhotoPaths...)
	item.StatusHistory = append([]issueStatusChange(nil), item.StatusHistory...)
	item.Comments = append([]issueComment(nil), item.Comments...)
	return item
}

func sortIssues(items []residentIssue) {
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
}

func newDocumentStore(path string, fileDir string) (*documentStore, error) {
	if fileDir == "" && path != "" {
		fileDir = filepath.Join(filepath.Dir(path), "documents")
	}
	store := &documentStore{path: path, fileDir: fileDir, data: documentStoreData{Documents: []documentRecord{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read document data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid document data")
	}
	store.data.Documents = normalizeDocuments(store.data.Documents)
	return store, nil
}

func (s *documentStore) Create(item documentRecord, header *multipart.FileHeader, now time.Time) (documentRecord, error) {
	if s == nil {
		return documentRecord{}, fmt.Errorf("document store unavailable")
	}
	if now.IsZero() {
		now = time.Now()
	}
	item.UploadedAt = now.UTC()
	fileSave, err := s.saveUploadedDocumentFile(item.TenantSlug, header)
	if err != nil {
		return documentRecord{}, err
	}
	item.ID = fileSave.ID
	item.SeriesID = fileSave.ID
	item.Version = 1
	item.Current = true
	item.Filename = fileSave.Filename
	item.StoredFilename = fileSave.StoredFilename
	item.Size = fileSave.Size
	item.ContentType = fileSave.ContentType
	item = normalizeDocumentRecord(item)
	if item.TenantSlug == "" || item.Title == "" || item.Category == "" || item.Visibility == "" || item.UploadedBy == "" {
		_ = os.Remove(fileSave.Path)
		return documentRecord{}, fmt.Errorf("invalid document metadata")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Documents = append(s.data.Documents, item)
	sortDocuments(s.data.Documents)
	if err := s.saveLocked(); err != nil {
		_ = os.Remove(fileSave.Path)
		return documentRecord{}, err
	}
	return copyDocument(item), nil
}

type documentFileSave struct {
	ID             string
	Filename       string
	StoredFilename string
	ContentType    string
	Size           int64
	Path           string
}

func (s *documentStore) saveUploadedDocumentFile(tenantSlug string, header *multipart.FileHeader) (documentFileSave, error) {
	if header == nil || header.Filename == "" || header.Size <= 0 {
		return documentFileSave{}, fmt.Errorf("document file required")
	}
	if s.fileDir == "" {
		return documentFileSave{}, fmt.Errorf("document file directory unavailable")
	}
	if header.Size > maxDocumentBytes {
		return documentFileSave{}, fmt.Errorf("document file too large")
	}
	file, err := header.Open()
	if err != nil {
		return documentFileSave{}, fmt.Errorf("could not open document")
	}
	defer file.Close()
	sniff := make([]byte, 512)
	n, readErr := file.Read(sniff)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return documentFileSave{}, fmt.Errorf("could not read document")
	}
	if n == 0 {
		return documentFileSave{}, fmt.Errorf("document file required")
	}
	contentType := http.DetectContentType(sniff[:n])
	ext, ok := documentExtension(contentType)
	if !ok {
		return documentFileSave{}, fmt.Errorf("unsupported document type")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return documentFileSave{}, fmt.Errorf("could not rewind document")
	}
	id, err := randomToken(12)
	if err != nil {
		return documentFileSave{}, err
	}
	storedFilename := id + ext
	storedPath, written, err := s.writeDocumentFile(tenantSlug, storedFilename, file)
	if err != nil {
		return documentFileSave{}, err
	}
	return documentFileSave{
		ID:             id,
		Filename:       sanitizeDocumentFilename(header.Filename),
		StoredFilename: storedFilename,
		ContentType:    contentType,
		Size:           written,
		Path:           storedPath,
	}, nil
}

func (s *documentStore) Replace(tenantSlug string, id string, uploadedBy string, header *multipart.FileHeader, now time.Time) (documentRecord, documentRecord, error) {
	if s == nil {
		return documentRecord{}, documentRecord{}, fmt.Errorf("document store unavailable")
	}
	tenantSlug = normalizeSlug(tenantSlug)
	id = strings.TrimSpace(id)
	uploadedBy = normalizeEmail(uploadedBy)
	if tenantSlug == "" || id == "" || uploadedBy == "" {
		return documentRecord{}, documentRecord{}, fmt.Errorf("invalid document replacement")
	}
	existing, found := s.Get(tenantSlug, id)
	if !found || !existing.Current {
		return documentRecord{}, documentRecord{}, fmt.Errorf("document not found")
	}
	if now.IsZero() {
		now = time.Now()
	}
	fileSave, err := s.saveUploadedDocumentFile(tenantSlug, header)
	if err != nil {
		return documentRecord{}, documentRecord{}, err
	}
	replacement := existing
	replacement.ID = fileSave.ID
	replacement.Version = existing.Version + 1
	replacement.Current = true
	replacement.SupersedesID = existing.ID
	replacement.ReplacedByID = ""
	replacement.Filename = fileSave.Filename
	replacement.StoredFilename = fileSave.StoredFilename
	replacement.Size = fileSave.Size
	replacement.ContentType = fileSave.ContentType
	replacement.UploadedBy = uploadedBy
	replacement.UploadedAt = now.UTC()
	replacement = normalizeDocumentRecord(replacement)

	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.data.Documents {
		if normalizeSlug(item.TenantSlug) != tenantSlug || item.ID != existing.ID || !item.Current {
			continue
		}
		item.Current = false
		item.ReplacedByID = replacement.ID
		replaced := normalizeDocumentRecord(item)
		s.data.Documents[i] = replaced
		s.data.Documents = append(s.data.Documents, replacement)
		sortDocuments(s.data.Documents)
		if err := s.saveLocked(); err != nil {
			_ = os.Remove(fileSave.Path)
			return documentRecord{}, documentRecord{}, err
		}
		return copyDocument(replacement), copyDocument(replaced), nil
	}
	_ = os.Remove(fileSave.Path)
	return documentRecord{}, documentRecord{}, fmt.Errorf("document not current")
}

func (s *documentStore) writeDocumentFile(tenantSlug string, storedFilename string, file multipart.File) (string, int64, error) {
	tenantSlug = normalizeSlug(tenantSlug)
	storedFilename = filepath.Base(storedFilename)
	if tenantSlug == "" || storedFilename == "" || storedFilename == "." || storedFilename == string(filepath.Separator) {
		return "", 0, fmt.Errorf("invalid document storage target")
	}
	dir := filepath.Join(s.fileDir, tenantSlug)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", 0, fmt.Errorf("could not create document directory")
	}
	tmp, err := os.CreateTemp(dir, storedFilename+".*.tmp")
	if err != nil {
		return "", 0, fmt.Errorf("could not create document file")
	}
	tmpName := tmp.Name()
	written, copyErr := io.Copy(tmp, io.LimitReader(file, maxDocumentBytes+1))
	chmodErr := tmp.Chmod(0o600)
	closeErr := tmp.Close()
	if copyErr != nil || chmodErr != nil || closeErr != nil {
		_ = os.Remove(tmpName)
		return "", 0, fmt.Errorf("could not write document file")
	}
	if written <= 0 || written > maxDocumentBytes {
		_ = os.Remove(tmpName)
		return "", 0, fmt.Errorf("document file too large")
	}
	dest := filepath.Join(dir, storedFilename)
	if err := os.Rename(tmpName, dest); err != nil {
		_ = os.Remove(tmpName)
		return "", 0, fmt.Errorf("could not store document file")
	}
	return dest, written, nil
}

func (s *documentStore) ListTenant(tenantSlug string) []documentRecord {
	if s == nil {
		return nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []documentRecord{}
	for _, item := range s.data.Documents {
		if normalizeSlug(item.TenantSlug) == tenantSlug {
			out = append(out, copyDocument(item))
		}
	}
	sortDocuments(out)
	return out
}

func (s *documentStore) ListCurrentTenant(tenantSlug string) []documentRecord {
	all := s.ListTenant(tenantSlug)
	out := []documentRecord{}
	for _, item := range all {
		if item.Current {
			out = append(out, item)
		}
	}
	sortDocuments(out)
	return out
}

func (s *documentStore) Versions(tenantSlug string, seriesID string) []documentRecord {
	if s == nil {
		return nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	seriesID = strings.TrimSpace(seriesID)
	if tenantSlug == "" || seriesID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []documentRecord{}
	for _, item := range s.data.Documents {
		if normalizeSlug(item.TenantSlug) == tenantSlug && item.SeriesID == seriesID {
			out = append(out, copyDocument(item))
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Version != out[j].Version {
			return out[i].Version > out[j].Version
		}
		return out[i].UploadedAt.After(out[j].UploadedAt)
	})
	return out
}

func (s *documentStore) Get(tenantSlug string, id string) (documentRecord, bool) {
	if s == nil {
		return documentRecord{}, false
	}
	tenantSlug = normalizeSlug(tenantSlug)
	id = strings.TrimSpace(id)
	if tenantSlug == "" || id == "" {
		return documentRecord{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.Documents {
		if normalizeSlug(item.TenantSlug) == tenantSlug && item.ID == id {
			return copyDocument(item), true
		}
	}
	return documentRecord{}, false
}

func (s *documentStore) FilePath(item documentRecord) (string, bool) {
	if s == nil || s.fileDir == "" {
		return "", false
	}
	tenantSlug := normalizeSlug(item.TenantSlug)
	storedFilename := filepath.Base(item.StoredFilename)
	if tenantSlug == "" || storedFilename == "" || storedFilename == "." || storedFilename == string(filepath.Separator) {
		return "", false
	}
	return filepath.Join(s.fileDir, tenantSlug, storedFilename), true
}

func (s *documentStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("could not create document data directory")
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode document data")
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("could not write document data")
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("could not replace document data")
	}
	return nil
}

func normalizeDocuments(items []documentRecord) []documentRecord {
	out := make([]documentRecord, 0, len(items))
	for _, item := range items {
		item = normalizeDocumentRecord(item)
		if item.ID == "" || item.TenantSlug == "" || item.Title == "" || item.StoredFilename == "" {
			continue
		}
		out = append(out, item)
	}
	sortDocuments(out)
	return out
}

func normalizeDocumentRecord(item documentRecord) documentRecord {
	item.ID = strings.TrimSpace(item.ID)
	item.SeriesID = strings.TrimSpace(item.SeriesID)
	if item.SeriesID == "" && item.ID != "" {
		item.SeriesID = item.ID
	}
	if item.Version <= 0 {
		item.Version = 1
	}
	item.SupersedesID = strings.TrimSpace(item.SupersedesID)
	item.ReplacedByID = strings.TrimSpace(item.ReplacedByID)
	if !item.Current && item.ReplacedByID == "" {
		item.Current = true
	}
	item.TenantSlug = normalizeSlug(item.TenantSlug)
	item.Title = truncateAuditValue(strings.TrimSpace(item.Title), 160)
	item.Category = normalizeDocumentCategory(item.Category)
	item.Visibility = normalizeDocumentVisibility(item.Visibility)
	item.UnitID = normalizeUnitID(item.UnitID)
	item.Filename = sanitizeDocumentFilename(item.Filename)
	item.StoredFilename = filepath.Base(strings.TrimSpace(item.StoredFilename))
	item.ContentType = strings.TrimSpace(item.ContentType)
	item.UploadedBy = normalizeEmail(item.UploadedBy)
	if item.Size < 0 {
		item.Size = 0
	}
	if item.UploadedAt.IsZero() {
		item.UploadedAt = time.Now()
	}
	item.UploadedAt = item.UploadedAt.UTC()
	return item
}

func copyDocument(item documentRecord) documentRecord {
	return item
}

func sortDocuments(items []documentRecord) {
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].UploadedAt.Equal(items[j].UploadedAt) {
			return items[i].UploadedAt.After(items[j].UploadedAt)
		}
		if items[i].Category != items[j].Category {
			return items[i].Category < items[j].Category
		}
		return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
	})
}

func documentExtension(contentType string) (string, bool) {
	switch contentType {
	case "application/pdf":
		return ".pdf", true
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}

func sanitizeDocumentFilename(raw string) string {
	name := filepath.Base(strings.TrimSpace(raw))
	if name == "." || name == string(filepath.Separator) {
		name = ""
	}
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || r == '/' || r == '\\' {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" {
		return "dokument"
	}
	if len([]rune(name)) > 120 {
		runes := []rune(name)
		name = string(runes[:120])
	}
	return name
}

func normalizeDocumentCategory(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case strings.ToLower(documentCategoryProtocol):
		return documentCategoryProtocol
	case strings.ToLower(documentCategoryBilling):
		return documentCategoryBilling
	case strings.ToLower(documentCategoryRules):
		return documentCategoryRules
	case strings.ToLower(documentCategoryContract):
		return documentCategoryContract
	case strings.ToLower(documentCategoryPlan), "pläne", "plaene":
		return documentCategoryPlan
	case "", strings.ToLower(documentCategoryOther):
		return documentCategoryOther
	default:
		return ""
	}
}

func documentCategories() []string {
	return []string{
		documentCategoryProtocol,
		documentCategoryBilling,
		documentCategoryRules,
		documentCategoryContract,
		documentCategoryPlan,
		documentCategoryOther,
	}
}

func documentCategoryOptions(selected string) []selectOption {
	selected = normalizeDocumentCategory(selected)
	options := make([]selectOption, 0, len(documentCategories()))
	for _, category := range documentCategories() {
		options = append(options, selectOption{Value: category, Label: category, Selected: selected == category})
	}
	return options
}

func documentUnitOptions(units []unit, selected string) []selectOption {
	selected = normalizeUnitID(selected)
	options := []selectOption{{Value: "", Label: "Gesamtes Haus", Selected: selected == ""}}
	for _, item := range units {
		id := normalizeUnitID(item.ID)
		label := strings.TrimSpace(item.Label)
		if id == "" || label == "" {
			continue
		}
		options = append(options, selectOption{Value: id, Label: label, Selected: selected == id})
	}
	return options
}

func documentUnitLabel(unitID string) string {
	unitID = normalizeUnitID(unitID)
	if unitID == "" {
		return ""
	}
	return unitID
}

func documentUnitAuditLabel(unitID string) string {
	unitID = normalizeUnitID(unitID)
	if unitID == "" {
		return ""
	}
	return unitID
}

func normalizeDocumentVisibility(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case documentVisibilityAllResidents, "all", "alle", "alle-bewohner":
		return documentVisibilityAllResidents
	case documentVisibilityOwnersOnly, "owner", "owners", "eigentuemer", "eigentümer":
		return documentVisibilityOwnersOnly
	case documentVisibilityManagerOnly, "manager", "verwalter", "verwaltung":
		return documentVisibilityManagerOnly
	default:
		return ""
	}
}

func documentVisibilityOptions(selected string) []selectOption {
	selected = normalizeDocumentVisibility(selected)
	values := []string{documentVisibilityAllResidents, documentVisibilityOwnersOnly, documentVisibilityManagerOnly}
	options := make([]selectOption, 0, len(values))
	for _, value := range values {
		options = append(options, selectOption{Value: value, Label: documentVisibilityLabel(value), Selected: selected == value})
	}
	return options
}

func documentVisibilityLabel(visibility string) string {
	switch normalizeDocumentVisibility(visibility) {
	case documentVisibilityAllResidents:
		return "Alle Bewohner"
	case documentVisibilityOwnersOnly:
		return "Nur Eigentümer"
	case documentVisibilityManagerOnly:
		return "Nur Verwaltung"
	default:
		return ""
	}
}

func documentVisibilityClass(visibility string) string {
	switch normalizeDocumentVisibility(visibility) {
	case documentVisibilityAllResidents:
		return "ok"
	case documentVisibilityOwnersOnly:
		return "unread"
	case documentVisibilityManagerOnly:
		return "role-admin"
	default:
		return ""
	}
}

func documentViews(items []documentRecord) []documentView {
	views := make([]documentView, 0, len(items))
	for _, item := range items {
		views = append(views, documentViewFrom(item))
	}
	return views
}

func (a *app) documentViewsForActor(tenantSlug string, email string, role string, items []documentRecord) []documentView {
	views := make([]documentView, 0, len(items))
	for _, item := range items {
		view := documentViewFrom(item)
		if a != nil && a.documentStore != nil {
			for _, version := range a.documentStore.Versions(tenantSlug, item.SeriesID) {
				if version.Current || !a.canViewDocument(tenantSlug, version, email, role) {
					continue
				}
				view.Versions = append(view.Versions, documentVersionView{
					ID:          version.ID,
					Version:     documentVersionLabel(version.Version),
					Filename:    version.Filename,
					Size:        formatBytes(version.Size),
					UploadedAt:  formatLocalDateTime(version.UploadedAt),
					DownloadURL: "/app/dokumente/" + url.PathEscape(version.ID) + "/download",
				})
			}
		}
		view.HasVersions = len(view.Versions) > 0
		views = append(views, view)
	}
	return views
}

func documentViewFrom(item documentRecord) documentView {
	return documentView{
		ID:              item.ID,
		Title:           item.Title,
		Category:        item.Category,
		Visibility:      documentVisibilityLabel(item.Visibility),
		VisibilityClass: documentVisibilityClass(item.Visibility),
		UnitLabel:       documentUnitLabel(item.UnitID),
		HasUnit:         normalizeUnitID(item.UnitID) != "",
		Filename:        item.Filename,
		Size:            formatBytes(item.Size),
		ContentType:     item.ContentType,
		UploadedBy:      item.UploadedBy,
		UploadedAt:      formatLocalDateTime(item.UploadedAt),
		DownloadURL:     "/app/dokumente/" + url.PathEscape(item.ID) + "/download",
		VersionLabel:    documentVersionLabel(item.Version),
		ReplaceDialogID: "document-replace-" + item.ID,
	}
}

func documentVersionLabel(version int) string {
	if version <= 0 {
		version = 1
	}
	return "Version " + strconv.Itoa(version)
}

func (a *app) documentCategorySectionsForActor(tenantSlug string, email string, role string, items []documentRecord, includeEmpty bool) []documentCategoryView {
	byCategory := map[string][]documentRecord{}
	for _, item := range items {
		byCategory[item.Category] = append(byCategory[item.Category], item)
	}
	sections := []documentCategoryView{}
	for _, category := range documentCategories() {
		docs := byCategory[category]
		if len(docs) == 0 && !includeEmpty {
			continue
		}
		sections = append(sections, documentCategoryView{
			Category:     category,
			Documents:    a.documentViewsForActor(tenantSlug, email, role, docs),
			HasDocuments: len(docs) > 0,
			EmptyMessage: "Keine passenden Dokumente in dieser Kategorie.",
		})
	}
	return sections
}

func filterDocuments(items []documentRecord, query string) []documentRecord {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		out := make([]documentRecord, 0, len(items))
		for _, item := range items {
			out = append(out, copyDocument(item))
		}
		return out
	}
	out := []documentRecord{}
	for _, item := range items {
		haystack := strings.ToLower(strings.Join([]string{
			item.Title,
			item.Category,
			documentVisibilityLabel(item.Visibility),
			item.Filename,
			item.UploadedBy,
			documentUnitLabel(item.UnitID),
		}, " "))
		if strings.Contains(haystack, query) {
			out = append(out, copyDocument(item))
		}
	}
	return out
}

func selectedDocumentSort(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "oldest", "title":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "newest"
	}
}

func documentSortOptions(selected string) []selectOption {
	selected = selectedDocumentSort(selected)
	options := []selectOption{
		{Value: "newest", Label: "Neueste zuerst", Selected: selected == "newest"},
		{Value: "oldest", Label: "Älteste zuerst", Selected: selected == "oldest"},
		{Value: "title", Label: "Titel A-Z", Selected: selected == "title"},
	}
	return options
}

func sortDocumentsForView(items []documentRecord, sortMode string) []documentRecord {
	out := make([]documentRecord, 0, len(items))
	for _, item := range items {
		out = append(out, copyDocument(item))
	}
	switch selectedDocumentSort(sortMode) {
	case "oldest":
		sort.SliceStable(out, func(i, j int) bool {
			if !out[i].UploadedAt.Equal(out[j].UploadedAt) {
				return out[i].UploadedAt.Before(out[j].UploadedAt)
			}
			return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title)
		})
	case "title":
		sort.SliceStable(out, func(i, j int) bool {
			return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title)
		})
	default:
		sortDocuments(out)
	}
	return out
}

func documentCategorySections(items []documentRecord, includeEmpty bool) []documentCategoryView {
	byCategory := map[string][]documentRecord{}
	for _, item := range items {
		byCategory[item.Category] = append(byCategory[item.Category], item)
	}
	sections := []documentCategoryView{}
	for _, category := range documentCategories() {
		docs := byCategory[category]
		if len(docs) == 0 && !includeEmpty {
			continue
		}
		sections = append(sections, documentCategoryView{
			Category:     category,
			Documents:    documentViews(docs),
			HasDocuments: len(docs) > 0,
			EmptyMessage: "Keine passenden Dokumente in dieser Kategorie.",
		})
	}
	return sections
}

func formatBytes(size int64) string {
	if size < 0 {
		size = 0
	}
	const kb = 1024
	const mb = 1024 * kb
	switch {
	case size >= mb:
		return formatDecimal(float64(size)/float64(mb), 1) + " MB"
	case size >= kb:
		return formatDecimal(float64(size)/float64(kb), 1) + " KB"
	default:
		return strconv.FormatInt(size, 10) + " B"
	}
}

func newVoteStore(path string) (*voteStore, error) {
	store := &voteStore{path: path, data: voteStoreData{Ballots: []ballot{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read vote data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid vote data")
	}
	store.data.Ballots = normalizeBallots(store.data.Ballots)
	return store, nil
}

func (s *voteStore) Create(item ballot) (ballot, error) {
	if s == nil {
		return ballot{}, fmt.Errorf("vote store unavailable")
	}
	now := time.Now().UTC()
	id, err := randomToken(12)
	if err != nil {
		return ballot{}, err
	}
	item.ID = id
	item.Status = ballotStatusDraft
	item.CreatedAt = now
	item.UpdatedAt = now
	item.Votes = nil
	item = normalizeBallot(item)
	if item.TenantSlug == "" || item.Title == "" || len(item.Options) < 2 || item.Type == "" || item.Weighting == "" || item.CreatedBy == "" {
		return ballot{}, fmt.Errorf("invalid ballot")
	}
	if !item.OpensAt.IsZero() && !item.ClosesAt.IsZero() && !item.ClosesAt.After(item.OpensAt) {
		return ballot{}, fmt.Errorf("ballot close must be after open")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Ballots = append(s.data.Ballots, item)
	sortBallots(s.data.Ballots)
	if err := s.saveLocked(); err != nil {
		return ballot{}, err
	}
	return copyBallot(item), nil
}

func (s *voteStore) Open(tenantSlug string, id string, at time.Time) (ballot, bool, error) {
	return s.setStatus(tenantSlug, id, ballotStatusOpen, at)
}

func (s *voteStore) Close(tenantSlug string, id string, at time.Time) (ballot, bool, error) {
	return s.setStatus(tenantSlug, id, ballotStatusClosed, at)
}

func (s *voteStore) CloseExpiredTenant(tenantSlug string, at time.Time) ([]ballot, error) {
	if s == nil {
		return nil, nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	if tenantSlug == "" {
		return nil, fmt.Errorf("invalid tenant")
	}
	if at.IsZero() {
		at = time.Now()
	}
	at = at.UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	closed := []ballot{}
	changed := false
	for i, item := range s.data.Ballots {
		item = normalizeBallot(item)
		if normalizeSlug(item.TenantSlug) != tenantSlug || item.Status != ballotStatusOpen || item.ClosesAt.IsZero() || at.Before(item.ClosesAt) {
			continue
		}
		item.Status = ballotStatusClosed
		item.UpdatedAt = at
		item = normalizeBallot(item)
		s.data.Ballots[i] = item
		closed = append(closed, copyBallot(item))
		changed = true
	}
	if !changed {
		return nil, nil
	}
	sortBallots(s.data.Ballots)
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return closed, nil
}

func (s *voteStore) setStatus(tenantSlug string, id string, status string, at time.Time) (ballot, bool, error) {
	if s == nil {
		return ballot{}, false, nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	id = strings.TrimSpace(id)
	status = normalizeBallotStatus(status)
	if tenantSlug == "" || id == "" || status == "" {
		return ballot{}, false, fmt.Errorf("invalid ballot status")
	}
	if at.IsZero() {
		at = time.Now()
	}
	at = at.UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.data.Ballots {
		if normalizeSlug(item.TenantSlug) != tenantSlug || item.ID != id {
			continue
		}
		if item.Status == ballotStatusClosed && status == ballotStatusOpen {
			return ballot{}, true, fmt.Errorf("closed ballot cannot reopen")
		}
		item.Status = status
		if status == ballotStatusOpen && item.OpensAt.IsZero() {
			item.OpensAt = at
		}
		if status == ballotStatusClosed && (item.ClosesAt.IsZero() || item.ClosesAt.After(at)) {
			item.ClosesAt = at
		}
		item.UpdatedAt = at
		item = normalizeBallot(item)
		s.data.Ballots[i] = item
		sortBallots(s.data.Ballots)
		if err := s.saveLocked(); err != nil {
			return ballot{}, true, err
		}
		return copyBallot(item), true, nil
	}
	return ballot{}, false, nil
}

func (s *voteStore) CastVote(tenantSlug string, id string, email string, option string, weight int, at time.Time) (ballot, bool, error) {
	if s == nil {
		return ballot{}, false, nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	id = strings.TrimSpace(id)
	email = normalizeEmail(email)
	option = strings.TrimSpace(option)
	if tenantSlug == "" || id == "" || email == "" || option == "" || weight <= 0 {
		return ballot{}, false, fmt.Errorf("invalid vote")
	}
	if at.IsZero() {
		at = time.Now()
	}
	at = at.UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.data.Ballots {
		if normalizeSlug(item.TenantSlug) != tenantSlug || item.ID != id {
			continue
		}
		item = normalizeBallot(item)
		if item.Status != ballotStatusOpen {
			return ballot{}, true, fmt.Errorf("ballot is not open")
		}
		if !item.OpensAt.IsZero() && at.Before(item.OpensAt) {
			return ballot{}, true, fmt.Errorf("ballot is not open yet")
		}
		if !item.ClosesAt.IsZero() && !at.Before(item.ClosesAt) {
			item.Status = ballotStatusClosed
			item.UpdatedAt = at
			item = normalizeBallot(item)
			s.data.Ballots[i] = item
			sortBallots(s.data.Ballots)
			if err := s.saveLocked(); err != nil {
				return ballot{}, true, err
			}
			return ballot{}, true, fmt.Errorf("ballot is closed")
		}
		if !ballotHasOption(item, option) {
			return ballot{}, true, fmt.Errorf("invalid vote option")
		}
		if item.Votes == nil {
			item.Votes = map[string]ballotVote{}
		}
		item.Votes[email] = ballotVote{Option: option, Weight: weight, At: at}
		item.UpdatedAt = at
		item = normalizeBallot(item)
		s.data.Ballots[i] = item
		sortBallots(s.data.Ballots)
		if err := s.saveLocked(); err != nil {
			return ballot{}, true, err
		}
		return copyBallot(item), true, nil
	}
	return ballot{}, false, nil
}

func (s *voteStore) MarkReminderSent(tenantSlug string, id string, recipients []string, at time.Time) (ballot, bool, error) {
	if s == nil {
		return ballot{}, false, nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	id = strings.TrimSpace(id)
	recipients = normalizeEmailList(recipients)
	if tenantSlug == "" || id == "" || len(recipients) == 0 {
		return ballot{}, false, fmt.Errorf("invalid reminder")
	}
	if at.IsZero() {
		at = time.Now()
	}
	at = at.UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.data.Ballots {
		if normalizeSlug(item.TenantSlug) != tenantSlug || item.ID != id {
			continue
		}
		item = normalizeBallot(item)
		if item.ReminderSentAt == nil {
			item.ReminderSentAt = map[string]time.Time{}
		}
		for _, recipient := range recipients {
			item.ReminderSentAt[recipient] = at
		}
		item.UpdatedAt = at
		item = normalizeBallot(item)
		s.data.Ballots[i] = item
		sortBallots(s.data.Ballots)
		if err := s.saveLocked(); err != nil {
			return ballot{}, true, err
		}
		return copyBallot(item), true, nil
	}
	return ballot{}, false, nil
}

func (s *voteStore) ListTenant(tenantSlug string) []ballot {
	if s == nil {
		return nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []ballot{}
	for _, item := range s.data.Ballots {
		if normalizeSlug(item.TenantSlug) == tenantSlug {
			out = append(out, copyBallot(item))
		}
	}
	sortBallots(out)
	return out
}

func (s *voteStore) Get(tenantSlug string, id string) (ballot, bool) {
	if s == nil {
		return ballot{}, false
	}
	tenantSlug = normalizeSlug(tenantSlug)
	id = strings.TrimSpace(id)
	if tenantSlug == "" || id == "" {
		return ballot{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.Ballots {
		if normalizeSlug(item.TenantSlug) == tenantSlug && item.ID == id {
			return copyBallot(normalizeBallot(item)), true
		}
	}
	return ballot{}, false
}

func (s *voteStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("could not create vote data directory")
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode vote data")
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("could not write vote data")
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("could not replace vote data")
	}
	return nil
}

func normalizeBallots(items []ballot) []ballot {
	out := make([]ballot, 0, len(items))
	for _, item := range items {
		item = normalizeBallot(item)
		if item.ID == "" || item.TenantSlug == "" || item.Title == "" || len(item.Options) < 2 || item.Type == "" || item.Weighting == "" {
			continue
		}
		out = append(out, item)
	}
	sortBallots(out)
	return out
}

func normalizeBallot(item ballot) ballot {
	item.ID = strings.TrimSpace(item.ID)
	item.TenantSlug = normalizeSlug(item.TenantSlug)
	item.Title = truncateAuditValue(strings.TrimSpace(item.Title), 160)
	item.Description = truncateAuditValue(strings.TrimSpace(item.Description), 5000)
	item.Options = normalizeBallotOptions(item.Options)
	item.Type = normalizeBallotType(item.Type)
	item.Weighting = normalizeBallotWeighting(item.Weighting)
	item.Status = normalizeBallotStatus(item.Status)
	item.CreatedBy = normalizeEmail(item.CreatedBy)
	if item.QuorumPPM < 0 {
		item.QuorumPPM = 0
	}
	if item.QuorumPPM > 1_000_000 {
		item.QuorumPPM = 1_000_000
	}
	if !item.OpensAt.IsZero() {
		item.OpensAt = item.OpensAt.UTC().Truncate(time.Second)
	}
	if !item.ClosesAt.IsZero() {
		item.ClosesAt = item.ClosesAt.UTC().Truncate(time.Second)
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	item.CreatedAt = item.CreatedAt.UTC().Truncate(time.Second)
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = item.CreatedAt
	}
	item.UpdatedAt = item.UpdatedAt.UTC().Truncate(time.Second)
	if item.ReminderBeforeMinutes <= 0 {
		item.ReminderBeforeMinutes = defaultBallotReminderBeforeMinutes
	}
	if item.ReminderBeforeMinutes > maxBallotReminderBeforeMinutes {
		item.ReminderBeforeMinutes = maxBallotReminderBeforeMinutes
	}
	if len(item.Votes) == 0 {
		item.Votes = nil
	} else {
		votes := map[string]ballotVote{}
		for email, vote := range item.Votes {
			email = normalizeEmail(email)
			vote.Option = strings.TrimSpace(vote.Option)
			if email == "" || vote.Option == "" || vote.Weight <= 0 || !ballotHasOption(item, vote.Option) {
				continue
			}
			if vote.At.IsZero() {
				vote.At = item.UpdatedAt
			}
			vote.At = vote.At.UTC().Truncate(time.Second)
			votes[email] = vote
		}
		item.Votes = votes
		if len(item.Votes) == 0 {
			item.Votes = nil
		}
	}
	if len(item.ReminderSentAt) == 0 {
		item.ReminderSentAt = nil
	} else {
		sent := map[string]time.Time{}
		for email, at := range item.ReminderSentAt {
			email = normalizeEmail(email)
			if email == "" {
				continue
			}
			if at.IsZero() {
				at = item.UpdatedAt
			}
			sent[email] = at.UTC().Truncate(time.Second)
		}
		item.ReminderSentAt = sent
		if len(item.ReminderSentAt) == 0 {
			item.ReminderSentAt = nil
		}
	}
	return item
}

func normalizeBallotOptions(raw []string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, option := range raw {
		option = truncateAuditValue(strings.TrimSpace(option), 120)
		if option == "" {
			continue
		}
		key := strings.ToLower(option)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, option)
		if len(out) >= 12 {
			break
		}
	}
	return out
}

func normalizeBallotType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "versammlung", "meeting", "eigentuemerversammlung", "eigentümerversammlung":
		return ballotTypeMeeting
	case "", "umlauf", "umlaufbeschluss", "circular", "resolution":
		return ballotTypeCircular
	default:
		return ""
	}
}

func normalizeBallotWeighting(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "per-head", "head", "kopf", "pro-kopf":
		return ballotWeightingPerHead
	case "", "per-share", "share", "anteil", "miteigentumsanteil":
		return ballotWeightingPerShare
	default:
		return ""
	}
}

func normalizeBallotStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "entwurf", "draft":
		return ballotStatusDraft
	case "offen", "open":
		return ballotStatusOpen
	case "geschlossen", "closed":
		return ballotStatusClosed
	default:
		return ""
	}
}

func ballotHasOption(item ballot, option string) bool {
	option = strings.TrimSpace(option)
	for _, existing := range item.Options {
		if existing == option {
			return true
		}
	}
	return false
}

func copyBallot(item ballot) ballot {
	item.Options = append([]string(nil), item.Options...)
	if len(item.Votes) > 0 {
		votes := map[string]ballotVote{}
		for email, vote := range item.Votes {
			votes[email] = vote
		}
		item.Votes = votes
	}
	if len(item.ReminderSentAt) > 0 {
		sent := map[string]time.Time{}
		for email, at := range item.ReminderSentAt {
			sent[email] = at
		}
		item.ReminderSentAt = sent
	}
	return item
}

func sortBallots(items []ballot) {
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
	})
}

func ballotWeightingLabel(weighting string) string {
	switch normalizeBallotWeighting(weighting) {
	case ballotWeightingPerHead:
		return "pro Kopf"
	case ballotWeightingPerShare:
		return "nach Miteigentumsanteil"
	default:
		return ""
	}
}

func newUnitStore(path string) (*unitStore, error) {
	store := &unitStore{path: path, data: unitStoreData{Units: []unit{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read unit data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid unit data")
	}
	store.data.Units = normalizeUnits(store.data.Units, "")
	return store, nil
}

func (s *unitStore) SetTenantUnits(tenantSlug string, units []unit) error {
	if s == nil {
		return nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	if tenantSlug == "" {
		return nil
	}
	normalized := normalizeUnits(units, tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.data.Units[:0]
	for _, existing := range s.data.Units {
		if normalizeSlug(existing.TenantSlug) != tenantSlug {
			kept = append(kept, existing)
		}
	}
	s.data.Units = append(kept, normalized...)
	sortUnits(s.data.Units)
	return s.saveLocked()
}

func (s *unitStore) ListTenant(tenantSlug string) []unit {
	if s == nil {
		return nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []unit{}
	for _, item := range s.data.Units {
		if normalizeSlug(item.TenantSlug) == tenantSlug {
			out = append(out, copyUnit(item))
		}
	}
	sortUnits(out)
	return out
}

func (s *unitStore) UnitCount(tenantSlug string) int {
	return len(s.ListTenant(tenantSlug))
}

func (s *unitStore) UnitsForEmail(tenantSlug string, email string) []unitMembership {
	if s == nil {
		return nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	email = normalizeEmail(email)
	if tenantSlug == "" || email == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []unitMembership{}
	for _, item := range s.data.Units {
		if normalizeSlug(item.TenantSlug) != tenantSlug {
			continue
		}
		relation := ""
		if emailListContains(item.OwnerEmails, email) {
			relation = roleOwner
		} else if emailListContains(item.RenterEmails, email) {
			relation = roleRenter
		}
		if relation != "" {
			out = append(out, unitMembership{Unit: copyUnit(item), Relation: relation})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return unitLess(out[i].Unit, out[j].Unit)
	})
	return out
}

func (s *unitStore) MembersForUnit(tenantSlug string, unitID string) unitMembers {
	if s == nil {
		return unitMembers{}
	}
	tenantSlug = normalizeSlug(tenantSlug)
	unitID = normalizeUnitID(unitID)
	if tenantSlug == "" || unitID == "" {
		return unitMembers{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.data.Units {
		if normalizeSlug(item.TenantSlug) == tenantSlug && normalizeSlug(item.ID) == unitID {
			item = copyUnit(item)
			return unitMembers{Unit: item, Owners: append([]string(nil), item.OwnerEmails...), Renters: append([]string(nil), item.RenterEmails...), Found: true}
		}
	}
	return unitMembers{}
}

func (s *unitStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("could not create unit data directory")
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode unit data")
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("could not write unit data")
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("could not replace unit data")
	}
	return nil
}

func (s *eventStore) Create(item houseEvent) (houseEvent, error) {
	if s == nil {
		return item, nil
	}
	now := time.Now().UTC()
	item.ID = ""
	item.CreatedAt = now
	item.UpdatedAt = now
	normalized, ok := normalizeHouseEvent(item)
	if !ok {
		return houseEvent{}, fmt.Errorf("invalid event")
	}
	id, err := randomToken(12)
	if err != nil {
		return houseEvent{}, err
	}
	normalized.ID = id
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Events = append(s.data.Events, normalized)
	sortEvents(s.data.Events)
	if err := s.saveLocked(); err != nil {
		return houseEvent{}, err
	}
	return normalized, nil
}

func (s *eventStore) Update(id string, updated houseEvent) (bool, error) {
	if s == nil {
		return false, nil
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	normalized, ok := normalizeHouseEvent(updated)
	if !ok {
		return false, fmt.Errorf("invalid event")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Events {
		if existing.ID != id || normalizeSlug(existing.TenantSlug) != normalizeSlug(normalized.TenantSlug) {
			continue
		}
		normalized.ID = existing.ID
		normalized.CreatedAt = existing.CreatedAt
		normalized.UpdatedAt = time.Now().UTC()
		if normalized.AuthorEmail == "" {
			normalized.AuthorEmail = existing.AuthorEmail
		}
		if normalized.AuthorName == "" {
			normalized.AuthorName = existing.AuthorName
		}
		s.data.Events[i] = normalized
		sortEvents(s.data.Events)
		if err := s.saveLocked(); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (s *eventStore) Delete(tenantSlug string, id string) (bool, error) {
	if s == nil {
		return false, nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	id = strings.TrimSpace(id)
	if tenantSlug == "" || id == "" {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.data.Events[:0]
	removed := false
	for _, item := range s.data.Events {
		if item.ID == id && normalizeSlug(item.TenantSlug) == tenantSlug {
			removed = true
			continue
		}
		kept = append(kept, item)
	}
	if !removed {
		return false, nil
	}
	s.data.Events = kept
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *eventStore) ListTenant(tenantSlug string) []houseEvent {
	if s == nil {
		return nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []houseEvent{}
	for _, item := range s.data.Events {
		if normalizeSlug(item.TenantSlug) == tenantSlug {
			out = append(out, copyEvent(item))
		}
	}
	sortEvents(out)
	return out
}

func (s *eventStore) Upcoming(tenantSlug string, now time.Time) []houseEvent {
	tenantSlug = normalizeSlug(tenantSlug)
	items := s.ListTenant(tenantSlug)
	out := []houseEvent{}
	for _, item := range items {
		if eventRollsOffAt(item).After(now) {
			out = append(out, item)
		}
	}
	sortEvents(out)
	return out
}

func (s *eventStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("could not create event data directory")
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode event data")
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("could not write event data")
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("could not replace event data")
	}
	return nil
}

func normalizeHouseEvent(item houseEvent) (houseEvent, bool) {
	item.ID = strings.TrimSpace(item.ID)
	item.TenantSlug = normalizeSlug(item.TenantSlug)
	item.Title = strings.TrimSpace(item.Title)
	item.Body = strings.TrimSpace(item.Body)
	item.Category = normalizeEventCategory(item.Category)
	item.Location = strings.TrimSpace(item.Location)
	item.AuthorEmail = normalizeEmail(item.AuthorEmail)
	item.AuthorName = strings.TrimSpace(item.AuthorName)
	item.StartsAt = item.StartsAt.UTC().Truncate(time.Second)
	if item.EndsAt != nil {
		endsAt := item.EndsAt.UTC().Truncate(time.Second)
		if !endsAt.After(item.StartsAt) {
			return houseEvent{}, false
		}
		item.EndsAt = &endsAt
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	} else {
		item.CreatedAt = item.CreatedAt.UTC().Truncate(time.Second)
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = item.CreatedAt
	} else {
		item.UpdatedAt = item.UpdatedAt.UTC().Truncate(time.Second)
	}
	if item.TenantSlug == "" || item.Title == "" || item.StartsAt.IsZero() {
		return houseEvent{}, false
	}
	return item, true
}

func copyEvent(item houseEvent) houseEvent {
	if item.EndsAt != nil {
		endsAt := *item.EndsAt
		item.EndsAt = &endsAt
	}
	return item
}

func sortEvents(items []houseEvent) {
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].StartsAt.Equal(items[j].StartsAt) {
			return items[i].StartsAt.Before(items[j].StartsAt)
		}
		if strings.ToLower(items[i].Title) != strings.ToLower(items[j].Title) {
			return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
		}
		return items[i].ID < items[j].ID
	})
}

func (s *announcementStore) Create(item announcement) (announcement, error) {
	now := time.Now().UTC()
	item.ID = ""
	item.CreatedAt = now
	item.UpdatedAt = now
	if item.PublishedAt.IsZero() {
		item.PublishedAt = now
	}
	if item.Category == "" {
		item.Category = "Info"
	}
	id, err := randomToken(12)
	if err != nil {
		return announcement{}, err
	}
	item.ID = id
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Announcements = append(s.data.Announcements, item)
	if err := s.saveLocked(); err != nil {
		return announcement{}, err
	}
	return item, nil
}

func (s *announcementStore) Update(id string, updated announcement) (bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Announcements {
		if existing.ID != id || normalizeSlug(existing.TenantSlug) != normalizeSlug(updated.TenantSlug) {
			continue
		}
		updated.ID = existing.ID
		updated.CreatedAt = existing.CreatedAt
		updated.UpdatedAt = time.Now().UTC()
		if updated.PublishedAt.IsZero() {
			updated.PublishedAt = existing.PublishedAt
		}
		if updated.AuthorEmail == "" {
			updated.AuthorEmail = existing.AuthorEmail
		}
		if updated.AuthorName == "" {
			updated.AuthorName = existing.AuthorName
		}
		s.data.Announcements[i] = updated
		if err := s.saveLocked(); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (s *announcementStore) Delete(tenantSlug string, id string) (bool, error) {
	tenantSlug = normalizeSlug(tenantSlug)
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.data.Announcements[:0]
	removed := false
	for _, item := range s.data.Announcements {
		if item.ID == id && normalizeSlug(item.TenantSlug) == tenantSlug {
			removed = true
			continue
		}
		kept = append(kept, item)
	}
	if !removed {
		return false, nil
	}
	s.data.Announcements = kept
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *announcementStore) Visible(tenantSlug string, now time.Time) []announcement {
	tenantSlug = normalizeSlug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []announcement{}
	for _, item := range s.data.Announcements {
		if normalizeSlug(item.TenantSlug) != tenantSlug {
			continue
		}
		if item.PublishedAt.After(now) {
			continue
		}
		if item.ExpiresAt != nil && !item.ExpiresAt.After(now) {
			continue
		}
		out = append(out, item)
	}
	sortAnnouncements(out)
	return out
}

func (s *announcementStore) Archive(tenantSlug string, now time.Time) []announcement {
	tenantSlug = normalizeSlug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []announcement{}
	for _, item := range s.data.Announcements {
		if normalizeSlug(item.TenantSlug) != tenantSlug {
			continue
		}
		if item.PublishedAt.After(now) {
			continue
		}
		out = append(out, item)
	}
	sortAnnouncements(out)
	return out
}

func (s *announcementStore) ListTenant(tenantSlug string) []announcement {
	tenantSlug = normalizeSlug(tenantSlug)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []announcement{}
	for _, item := range s.data.Announcements {
		if normalizeSlug(item.TenantSlug) == tenantSlug {
			out = append(out, item)
		}
	}
	sortAnnouncements(out)
	return out
}

func (s *announcementStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("could not create announcement data directory")
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode announcement data")
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("could not write announcement data")
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("could not replace announcement data")
	}
	return nil
}

func sortAnnouncements(items []announcement) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Pinned != items[j].Pinned {
			return items[i].Pinned
		}
		if !items[i].PublishedAt.Equal(items[j].PublishedAt) {
			return items[i].PublishedAt.After(items[j].PublishedAt)
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
}

func (a *app) parkingTelemetry(ctx context.Context, tenant tenantConfig) parkingTelemetry {
	ha := tenant.HA
	telemetry := parkingTelemetry{
		Configured: ha.baseURL != "",
		Entities: []parkingEntityRef{
			{Label: "Zählerstand", EntityID: ha.meterEnergyEntity},
			{Label: "Leistung", EntityID: ha.powerEntity},
			{Label: "aWATTar Preis", EntityID: ha.priceEntity},
		},
	}
	if ha.baseURL == "" {
		telemetry.Message = "Home Assistant ist lokal noch nicht konfiguriert."
		return telemetry
	}
	if ha.token == "" {
		telemetry.Message = "Home Assistant ist vorbereitet, aber lokal fehlt noch ein HA_TOKEN."
		return telemetry
	}

	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	var liveEnergy float64
	var livePrice float64
	var hasLiveEnergy bool
	var hasLivePrice bool
	states := []struct {
		label  string
		entity string
	}{
		{label: "Zählerstand", entity: ha.meterEnergyEntity},
		{label: "Aktuelle Leistung", entity: ha.powerEntity},
		{label: "aWATTar Gesamtpreis", entity: ha.priceEntity},
	}
	for _, item := range states {
		state, err := ha.State(ctx, item.entity)
		if err != nil {
			telemetry.Message = "Home Assistant konnte gerade nicht gelesen werden."
			return telemetry
		}
		telemetry.Metrics = append(telemetry.Metrics, parkingMetric{
			Label:  item.label,
			Value:  formatHAValue(state),
			Detail: item.entity,
		})
		switch item.entity {
		case ha.meterEnergyEntity:
			if value, err := parseHAFloat(state.State); err == nil {
				liveEnergy = value
				hasLiveEnergy = true
			}
		case ha.priceEntity:
			if value, err := parseHAFloat(state.State); err == nil {
				livePrice = value
				hasLivePrice = true
			}
		}
	}
	if hasLiveEnergy && hasLivePrice {
		if err := a.parkingStore.AppendSamples(tenant.Slug, []parkingStoredSample{{
			At:             time.Now().UTC(),
			EnergyKWh:      liveEnergy,
			PriceEURPerKWh: livePrice,
		}}); err != nil {
			log.Printf("parking live sample save failed for %s: %v", tenant.Slug, err)
		}
	}
	telemetry.Connected = true
	telemetry.Message = "Live aus Home Assistant gelesen."
	return telemetry
}

func (a *app) parkingAccounting(ctx context.Context, tenant tenantConfig) parkingAccountingView {
	seedCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	if err := a.seedParkingHistory(seedCtx, tenant, a.parkingHistoryStart, time.Now()); err != nil && tenant.HA.baseURL != "" && tenant.HA.token != "" {
		log.Printf("parking history seed failed for %s: %v", tenant.Slug, err)
	}

	data := a.parkingStore.TenantData(tenant.Slug)
	months := calculateParkingMonths(data, time.Now(), time.Local)
	view := parkingAccountingView{
		GridFeeValue:     formatInputFloat(data.Settings.GridFeeEURPerKWh),
		GridFeeLabel:     formatEURPerKWh(data.Settings.GridFeeEURPerKWh),
		Months:           months,
		HasMonths:        len(months) > 0,
		HistoryAvailable: len(data.EnergySamples) >= 2 && len(data.PriceSamples) > 0,
	}
	if len(data.EnergySamples) > 0 {
		last := data.EnergySamples[len(data.EnergySamples)-1].At.In(time.Local)
		view.LastSampleLabel = formatLocalDateTime(last)
	}
	if len(months) == 0 {
		view.Message = "Noch nicht genug Messpunkte für eine Monatsabrechnung. Die App sammelt ab jetzt eigene Messpunkte und liest zusätzlich verfügbare Home-Assistant-Historie ein."
	} else {
		view.Message = "Kosten werden stündlich aus Zählerdifferenz, aWATTar-Preis und Netzbetreibergebühren berechnet. Historie wird ab " + formatLocalDate(a.parkingHistoryStart) + " aus Home Assistant nachgezogen, soweit dort Statistikdaten vorhanden sind."
	}
	return view
}

func (a *app) parkingMonthDetails(ctx context.Context, tenant tenantConfig, month string) parkingMonthDetailView {
	seedCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	if err := a.seedParkingHistory(seedCtx, tenant, a.parkingHistoryStart, time.Now()); err != nil && tenant.HA.baseURL != "" && tenant.HA.token != "" {
		log.Printf("parking history seed failed for %s: %v", tenant.Slug, err)
	}

	data := a.parkingStore.TenantData(tenant.Slug)
	view := calculateParkingMonthDetails(data, month, time.Now(), time.Local)
	view.BackPath = "/app/parking"
	view.GridFeeLabel = formatEURPerKWh(data.Settings.GridFeeEURPerKWh)
	view.Message = "Stundenwerte aus Zählerdifferenz und dem in dieser Stunde gültigen aWATTar-Preis."
	if len(data.EnergySamples) > 0 {
		last := data.EnergySamples[len(data.EnergySamples)-1].At.In(time.Local)
		view.LastSampleLabel = formatLocalDateTime(last)
	}
	return view
}

func (a *app) seedParkingHistory(ctx context.Context, tenant tenantConfig, start time.Time, end time.Time) error {
	ha := tenant.HA
	if ha.baseURL == "" || ha.token == "" || ha.meterEnergyEntity == "" || ha.priceEntity == "" {
		return nil
	}
	energySamples, priceSamples, err := ha.Statistics(ctx, start, end)
	if err != nil {
		log.Printf("parking statistics backfill failed for %s: %v", tenant.Slug, err)
	}
	restStart := end.Add(-35 * 24 * time.Hour)
	if restStart.Before(start) {
		restStart = start
	}
	history, err := ha.History(ctx, restStart, end, []string{ha.meterEnergyEntity, ha.priceEntity})
	if err != nil {
		if len(energySamples) == 0 && len(priceSamples) == 0 {
			return err
		}
		log.Printf("parking REST history fallback failed for %s: %v", tenant.Slug, err)
		return a.parkingStore.AppendReadings(tenant.Slug, energySamples, priceSamples)
	}
	energySamples = append(energySamples, samplesFromHistory(history[ha.meterEnergyEntity])...)
	priceSamples = append(priceSamples, samplesFromHistory(history[ha.priceEntity])...)
	if len(energySamples) == 0 && len(priceSamples) == 0 {
		return nil
	}
	log.Printf("parking history seed found %d energy sample(s) and %d price sample(s) for %s", len(energySamples), len(priceSamples), tenant.Slug)
	return a.parkingStore.AppendReadings(tenant.Slug, energySamples, priceSamples)
}

func (a *app) startParkingSampler() func() {
	if a.parkingSampleInterval <= 0 {
		log.Printf("parking sampler disabled")
		return func() {}
	}
	configuredTenants := 0
	for _, tenant := range a.tenants {
		if tenant.HA.baseURL != "" && tenant.HA.token != "" {
			configuredTenants++
		}
	}
	if configuredTenants == 0 {
		log.Printf("parking sampler disabled: no configured Home Assistant tenants")
		return func() {}
	}
	log.Printf("parking sampler enabled for %d tenant(s), interval %s", configuredTenants, a.parkingSampleInterval)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		a.backfillParkingTenants(ctx)
		a.sampleParkingTenants(ctx)
		ticker := time.NewTicker(a.parkingSampleInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.sampleParkingTenants(ctx)
			}
		}
	}()
	return cancel
}

func (a *app) backfillParkingTenants(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for _, tenant := range a.tenants {
		if tenant.HA.baseURL == "" || tenant.HA.token == "" {
			continue
		}
		if err := a.seedParkingHistory(ctx, tenant, a.parkingHistoryStart, time.Now()); err != nil {
			log.Printf("parking startup history seed failed for %s: %v", tenant.Slug, err)
			continue
		}
		log.Printf("parking startup history seed completed for %s", tenant.Slug)
	}
}

func (a *app) sampleParkingTenants(ctx context.Context) {
	for _, tenant := range a.tenants {
		if tenant.HA.baseURL == "" || tenant.HA.token == "" {
			continue
		}
		if err := a.sampleParkingTenant(ctx, tenant); err != nil {
			log.Printf("parking sample failed for %s: %v", tenant.Slug, err)
			continue
		}
		log.Printf("parking sample saved for %s", tenant.Slug)
	}
}

func (a *app) sampleParkingTenant(ctx context.Context, tenant tenantConfig) error {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	energy, err := tenant.HA.State(ctx, tenant.HA.meterEnergyEntity)
	if err != nil {
		return err
	}
	price, err := tenant.HA.State(ctx, tenant.HA.priceEntity)
	if err != nil {
		return err
	}
	energyValue, err := parseHAFloat(energy.State)
	if err != nil {
		return err
	}
	priceValue, err := parseHAFloat(price.State)
	if err != nil {
		return err
	}
	return a.parkingStore.AppendSamples(tenant.Slug, []parkingStoredSample{{
		At:             time.Now().UTC(),
		EnergyKWh:      energyValue,
		PriceEURPerKWh: priceValue,
	}})
}

func newParkingStore(path string) (*parkingStore, error) {
	store := &parkingStore{
		path: path,
		data: parkingStoreData{Tenants: map[string]parkingTenantData{}},
	}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read parking data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid parking data")
	}
	if store.data.Tenants == nil {
		store.data.Tenants = map[string]parkingTenantData{}
	}
	return store, nil
}

func (s *parkingStore) TenantData(tenantSlug string) parkingTenantData {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.tenantLocked(tenantSlug)
	data.Months = copyMonthStates(data.Months)
	data.EnergySamples = append([]parkingNumericSample(nil), data.EnergySamples...)
	data.PriceSamples = append([]parkingNumericSample(nil), data.PriceSamples...)
	data.Samples = append([]parkingStoredSample(nil), data.Samples...)
	return data
}

func (s *parkingStore) SetGridFee(tenantSlug string, gridFee float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.tenantLocked(tenantSlug)
	data.Settings.GridFeeEURPerKWh = gridFee
	s.data.Tenants[normalizeSlug(tenantSlug)] = data
	return s.saveLocked()
}

func (s *parkingStore) SetMonthPaid(tenantSlug string, month string, paid bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.tenantLocked(tenantSlug)
	if data.Months == nil {
		data.Months = map[string]parkingMonthState{}
	}
	data.Months[month] = parkingMonthState{Paid: paid}
	s.data.Tenants[normalizeSlug(tenantSlug)] = data
	return s.saveLocked()
}

func (s *parkingStore) AppendSamples(tenantSlug string, samples []parkingStoredSample) error {
	if len(samples) == 0 {
		return nil
	}
	energy := make([]parkingNumericSample, 0, len(samples))
	prices := make([]parkingNumericSample, 0, len(samples))
	for _, sample := range samples {
		energy = append(energy, parkingNumericSample{At: sample.At, Value: sample.EnergyKWh})
		prices = append(prices, parkingNumericSample{At: sample.At, Value: sample.PriceEURPerKWh})
	}
	return s.AppendReadings(tenantSlug, energy, prices)
}

func (s *parkingStore) AppendReadings(tenantSlug string, energySamples []parkingNumericSample, priceSamples []parkingNumericSample) error {
	if len(energySamples) == 0 && len(priceSamples) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.tenantLocked(tenantSlug)
	for _, sample := range energySamples {
		if sample.At.IsZero() || sample.Value < 0 || sample.Value > 1000000 {
			continue
		}
		data.EnergySamples = append(data.EnergySamples, parkingNumericSample{
			At:    sample.At.UTC(),
			Value: sample.Value,
		})
	}
	for _, sample := range priceSamples {
		if sample.At.IsZero() || sample.Value < -5 || sample.Value > 5 {
			continue
		}
		data.PriceSamples = append(data.PriceSamples, parkingNumericSample{
			At:    sample.At.UTC(),
			Value: sample.Value,
		})
	}
	keepAfter := time.Now().AddDate(-1, -1, 0)
	data.EnergySamples = normalizeNumericSamples(data.EnergySamples, keepAfter)
	data.PriceSamples = normalizeNumericSamples(data.PriceSamples, keepAfter)
	data.Samples = nil
	s.data.Tenants[normalizeSlug(tenantSlug)] = data
	return s.saveLocked()
}

func (s *parkingStore) tenantLocked(tenantSlug string) parkingTenantData {
	tenantSlug = normalizeSlug(tenantSlug)
	if tenantSlug == "" {
		tenantSlug = "default"
	}
	if s.data.Tenants == nil {
		s.data.Tenants = map[string]parkingTenantData{}
	}
	data, ok := s.data.Tenants[tenantSlug]
	if !ok {
		data = defaultParkingTenantData()
	}
	if data.Months == nil {
		data.Months = map[string]parkingMonthState{}
	}
	if len(data.Samples) > 0 {
		for _, sample := range data.Samples {
			data.EnergySamples = append(data.EnergySamples, parkingNumericSample{At: sample.At, Value: sample.EnergyKWh})
			data.PriceSamples = append(data.PriceSamples, parkingNumericSample{At: sample.At, Value: sample.PriceEURPerKWh})
		}
		data.Samples = nil
	}
	keepAfter := time.Now().AddDate(-1, -1, 0)
	data.EnergySamples = normalizeNumericSamples(data.EnergySamples, keepAfter)
	data.PriceSamples = normalizeNumericSamples(data.PriceSamples, keepAfter)
	s.data.Tenants[tenantSlug] = data
	return data
}

func (s *parkingStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("could not create parking data directory")
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode parking data")
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("could not write parking data")
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("could not replace parking data")
	}
	return nil
}

func defaultParkingTenantData() parkingTenantData {
	return parkingTenantData{
		Settings: parkingSettings{GridFeeEURPerKWh: 0.10},
		Months:   map[string]parkingMonthState{},
	}
}

func copyMonthStates(in map[string]parkingMonthState) map[string]parkingMonthState {
	out := map[string]parkingMonthState{}
	for month, state := range in {
		out[month] = state
	}
	return out
}

func normalizeNumericSamples(samples []parkingNumericSample, keepAfter time.Time) []parkingNumericSample {
	out := make([]parkingNumericSample, 0, len(samples))
	for _, sample := range samples {
		if sample.At.IsZero() || sample.At.Before(keepAfter) {
			continue
		}
		out = append(out, parkingNumericSample{At: sample.At.UTC().Truncate(time.Second), Value: sample.Value})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At)
	})
	deduped := out[:0]
	for _, sample := range out {
		if len(deduped) > 0 && deduped[len(deduped)-1].At.Equal(sample.At) {
			deduped[len(deduped)-1] = sample
			continue
		}
		deduped = append(deduped, sample)
	}
	return deduped
}

func samplesFromHistory(history []haHistoryState) []parkingNumericSample {
	out := make([]parkingNumericSample, 0, len(history))
	for _, item := range history {
		value, err := parseHAFloat(item.State)
		if err != nil {
			continue
		}
		at := item.timestamp()
		if at.IsZero() {
			continue
		}
		out = append(out, parkingNumericSample{At: at.UTC(), Value: value})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At)
	})
	return out
}

func samplesFromStatistics(stats []haStatistic, fields ...string) []parkingNumericSample {
	out := make([]parkingNumericSample, 0, len(stats))
	for _, stat := range stats {
		at, err := parseStatisticTime(stat.Start)
		if err != nil || at.IsZero() {
			continue
		}
		value, ok := statisticValue(stat, fields...)
		if !ok {
			continue
		}
		out = append(out, parkingNumericSample{At: at.UTC(), Value: value})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At)
	})
	return out
}

func statisticValue(stat haStatistic, fields ...string) (float64, bool) {
	for _, field := range fields {
		switch field {
		case "state":
			if stat.State != nil {
				return *stat.State, true
			}
		case "sum":
			if stat.Sum != nil {
				return *stat.Sum, true
			}
		case "mean":
			if stat.Mean != nil {
				return *stat.Mean, true
			}
		case "min":
			if stat.Min != nil {
				return *stat.Min, true
			}
		case "max":
			if stat.Max != nil {
				return *stat.Max, true
			}
		}
	}
	return 0, false
}

func parseStatisticTime(raw json.RawMessage) (time.Time, error) {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 || string(raw) == "null" {
		return time.Time{}, errors.New("missing statistic time")
	}
	if raw[0] == '"' {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return time.Time{}, err
		}
		return time.Parse(time.RFC3339, value)
	}
	var value float64
	if err := json.Unmarshal(raw, &value); err != nil {
		return time.Time{}, err
	}
	if value > 100000000000 {
		return time.UnixMilli(int64(value)), nil
	}
	return time.Unix(int64(value), 0), nil
}

func priceAt(samples []parkingNumericSample, at time.Time) (float64, bool) {
	if len(samples) == 0 {
		return 0, false
	}
	index := sort.Search(len(samples), func(i int) bool {
		return samples[i].At.After(at)
	})
	if index == 0 {
		return samples[0].Value, true
	}
	return samples[index-1].Value, true
}

type parkingHourUsage struct {
	At         time.Time
	KWh        float64
	PriceEUR   float64
	EnergyCost float64
	GridCost   float64
}

func calculateParkingHourlyUsage(energySamples []parkingNumericSample, priceSamples []parkingNumericSample, gridFeeEURPerKWh float64, now time.Time) []parkingHourUsage {
	keepAfter := now.AddDate(-1, -1, 0)
	energySamples = normalizeNumericSamples(append([]parkingNumericSample(nil), energySamples...), keepAfter)
	priceSamples = normalizeNumericSamples(append([]parkingNumericSample(nil), priceSamples...), keepAfter)
	if len(energySamples) < 2 || len(priceSamples) == 0 {
		return nil
	}
	var out []parkingHourUsage
	for i := 1; i < len(energySamples); i++ {
		prev := energySamples[i-1]
		curr := energySamples[i]
		if !curr.At.After(prev.At) {
			continue
		}
		delta := curr.Value - prev.Value
		if delta <= 0 || delta > 500 {
			continue
		}
		totalSeconds := curr.At.Sub(prev.At).Seconds()
		if totalSeconds <= 0 {
			continue
		}
		cursor := prev.At
		for cursor.Before(curr.At) {
			nextHour := cursor.Truncate(time.Hour).Add(time.Hour)
			segmentEnd := nextHour
			if segmentEnd.After(curr.At) {
				segmentEnd = curr.At
			}
			seconds := segmentEnd.Sub(cursor).Seconds()
			if seconds <= 0 {
				break
			}
			kWh := delta * (seconds / totalSeconds)
			price, ok := priceAt(priceSamples, cursor)
			if !ok {
				cursor = segmentEnd
				continue
			}
			hour := cursor.Truncate(time.Hour)
			out = append(out, parkingHourUsage{
				At:         hour,
				KWh:        kWh,
				PriceEUR:   price,
				EnergyCost: price * kWh,
				GridCost:   gridFeeEURPerKWh * kWh,
			})
			cursor = segmentEnd
		}
	}
	if len(out) == 0 {
		return nil
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At)
	})
	merged := out[:0]
	for _, item := range out {
		if len(merged) > 0 && merged[len(merged)-1].At.Equal(item.At) {
			merged[len(merged)-1].KWh += item.KWh
			merged[len(merged)-1].EnergyCost += item.EnergyCost
			merged[len(merged)-1].GridCost += item.GridCost
			continue
		}
		merged = append(merged, item)
	}
	return merged
}

func calculateParkingMonths(data parkingTenantData, now time.Time, loc *time.Location) []parkingMonthView {
	if loc == nil {
		loc = time.Local
	}
	hours := calculateParkingHourlyUsage(data.EnergySamples, data.PriceSamples, data.Settings.GridFeeEURPerKWh, now)
	if len(hours) == 0 {
		return nil
	}
	type aggregate struct {
		month      string
		kWh        float64
		energyCost float64
		gridCost   float64
		first      time.Time
		last       time.Time
		hourCount  int
	}
	aggregates := map[string]*aggregate{}
	for _, hour := range hours {
		month := hour.At.In(loc).Format("2006-01")
		hourEnd := hour.At.Add(time.Hour)
		agg := aggregates[month]
		if agg == nil {
			agg = &aggregate{month: month, first: hour.At, last: hourEnd}
			aggregates[month] = agg
		}
		if hour.At.Before(agg.first) {
			agg.first = hour.At
		}
		if hourEnd.After(agg.last) {
			agg.last = hourEnd
		}
		agg.kWh += hour.KWh
		agg.energyCost += hour.EnergyCost
		agg.gridCost += hour.GridCost
		agg.hourCount++
	}
	for month := range data.Months {
		if _, ok := aggregates[month]; !ok {
			parsed, err := time.ParseInLocation("2006-01", month, loc)
			if err != nil {
				continue
			}
			aggregates[month] = &aggregate{month: month, first: parsed, last: parsed}
		}
	}
	if len(aggregates) == 0 {
		return nil
	}
	maxTotal := 0.0
	months := make([]string, 0, len(aggregates))
	for month, agg := range aggregates {
		months = append(months, month)
		total := agg.energyCost + agg.gridCost
		if total > maxTotal {
			maxTotal = total
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(months)))
	out := make([]parkingMonthView, 0, len(months))
	for _, month := range months {
		agg := aggregates[month]
		total := agg.energyCost + agg.gridCost
		averageAwattar := 0.0
		effectivePrice := 0.0
		if agg.kWh > 0 {
			averageAwattar = agg.energyCost / agg.kWh
			effectivePrice = total / agg.kWh
		}
		chartPercent := 0
		if maxTotal > 0 {
			chartPercent = int(total / maxTotal * 100)
			if chartPercent < 3 && total > 0 {
				chartPercent = 3
			}
		}
		paid := data.Months[month].Paid
		firstOfMonth, _ := time.ParseInLocation("2006-01", month, loc)
		out = append(out, parkingMonthView{
			Month:           month,
			MonthLabel:      formatMonthLabel(month, loc),
			DetailPath:      "/app/parking/month/" + month,
			PeriodLabel:     formatPeriodLabel(agg.first, agg.last, loc),
			KWh:             formatKWh(agg.kWh),
			EnergyCost:      formatEUR(agg.energyCost),
			GridCost:        formatEUR(agg.gridCost),
			TotalCost:       formatEUR(total),
			AverageAwattar:  formatEURPerKWh(averageAwattar),
			EffectivePrice:  formatEURPerKWh(effectivePrice),
			AveragePrice:    formatEURPerKWh(effectivePrice),
			Paid:            paid,
			PaidLabel:       paidLabel(paid),
			TogglePaidValue: boolFormValue(!paid),
			ToggleLabel:     togglePaidLabel(paid),
			ChartPercent:    chartPercent,
			Partial:         agg.hourCount == 0 || agg.first.In(loc).After(firstOfMonth.Add(24*time.Hour)),
			SampleCount:     len(data.EnergySamples),
			HourCount:       agg.hourCount,
		})
	}
	return out
}

func calculateParkingMonthDetails(data parkingTenantData, month string, now time.Time, loc *time.Location) parkingMonthDetailView {
	if loc == nil {
		loc = time.Local
	}
	view := parkingMonthDetailView{
		Month:      month,
		MonthLabel: formatMonthLabel(month, loc),
	}
	for _, summary := range calculateParkingMonths(data, now, loc) {
		if summary.Month == month {
			view.Summary = summary
			break
		}
	}
	hours := calculateParkingHourlyUsage(data.EnergySamples, data.PriceSamples, data.Settings.GridFeeEURPerKWh, now)
	if len(hours) == 0 {
		return view
	}
	maxTotal := 0.0
	var monthHours []parkingHourUsage
	for _, hour := range hours {
		if hour.At.In(loc).Format("2006-01") != month {
			continue
		}
		monthHours = append(monthHours, hour)
		total := hour.EnergyCost + hour.GridCost
		if total > maxTotal {
			maxTotal = total
		}
	}
	if len(monthHours) == 0 {
		return view
	}
	sort.Slice(monthHours, func(i, j int) bool {
		return monthHours[i].At.Before(monthHours[j].At)
	})
	view.Hours = make([]parkingHourView, 0, len(monthHours))
	for _, hour := range monthHours {
		total := hour.EnergyCost + hour.GridCost
		averageAwattar := 0.0
		if hour.KWh > 0 {
			averageAwattar = hour.EnergyCost / hour.KWh
		}
		chartPercent := 0
		if maxTotal > 0 {
			chartPercent = int(total / maxTotal * 100)
			if chartPercent < 2 && total > 0 {
				chartPercent = 2
			}
		}
		view.Hours = append(view.Hours, parkingHourView{
			AtLabel:             formatDateTimeIn(hour.At, loc, deATShortDateTimeLayout),
			AtTitle:             formatDateTimeIn(hour.At, loc, deATDateTimeLayout) + " bis " + formatDateTimeIn(hour.At.Add(time.Hour), loc, deATTimeLayout),
			KWh:                 formatKWh(hour.KWh),
			KWhTitle:            "Verbrauch: " + formatPreciseKWh(hour.KWh),
			AverageAwattar:      formatEURPerKWh(averageAwattar),
			AverageAwattarTitle: "aWATTar Preis dieser Stunde: " + formatPreciseEURPerKWh(averageAwattar),
			EnergyCost:          formatEUR(hour.EnergyCost),
			EnergyCostTitle:     "Stromkosten: " + formatPreciseEUR(hour.EnergyCost) + " = " + formatPreciseKWh(hour.KWh) + " × " + formatPreciseEURPerKWh(averageAwattar),
			GridCost:            formatEUR(hour.GridCost),
			GridCostTitle:       "Netzgebühr: " + formatPreciseEUR(hour.GridCost) + " = " + formatPreciseKWh(hour.KWh) + " × " + formatPreciseEURPerKWh(data.Settings.GridFeeEURPerKWh),
			TotalCost:           formatEUR(total),
			TotalCostTitle:      "Summe: " + formatPreciseEUR(total) + " = Strom " + formatPreciseEUR(hour.EnergyCost) + " + Netzgebühr " + formatPreciseEUR(hour.GridCost),
			WeightTitle:         "Relative Höhe der Stundensumme. 100% entspricht der teuersten Stunde dieses Monats.",
			ChartPercent:        chartPercent,
		})
	}
	view.HasHours = len(view.Hours) > 0
	return view
}

type userProfile struct {
	Email             string                      `json:"email"`
	Title             string                      `json:"title"`
	FirstName         string                      `json:"first_name"`
	LastName          string                      `json:"last_name"`
	Phone             string                      `json:"phone"`
	DirectoryOptIn    bool                        `json:"directory_opt_in,omitempty"`
	Role              string                      `json:"role"`
	Status            string                      `json:"status"`
	Tenants           []string                    `json:"tenants"`
	Permissions       []string                    `json:"permissions"`
	TenantMemberships map[string]tenantMembership `json:"tenant_memberships,omitempty"`
	AuthMethods       []string                    `json:"auth_methods"`
}

type tenantMembership struct {
	Role        string   `json:"role,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

func (p userProfile) DisplayName() string {
	parts := []string{}
	for _, part := range []string{p.Title, p.FirstName, p.LastName} {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	return p.Email
}

// Initials returns up to two uppercase letters for the avatar badge, derived
// from first+last name, falling back to the first glyph of the display name.
func (p userProfile) Initials() string {
	first := initialLetter(p.FirstName)
	last := initialLetter(p.LastName)
	if first == "" && last == "" {
		return strings.ToUpper(initialLetter(p.DisplayName()))
	}
	return strings.ToUpper(first + last)
}

func initialLetter(s string) string {
	for _, r := range strings.TrimSpace(s) {
		return string(r)
	}
	return ""
}

func (p userProfile) HasPermission(permission string) bool {
	permission = strings.ToLower(strings.TrimSpace(permission))
	for _, item := range p.Permissions {
		if strings.ToLower(strings.TrimSpace(item)) == permission {
			return true
		}
	}
	return false
}

func (p userProfile) ForTenant(tenantSlug string) userProfile {
	tenantSlug = normalizeSlug(tenantSlug)
	out := p
	out.Email = normalizeEmail(out.Email)
	out.Role = normalizeRole(out.Role)
	out.Tenants = normalizeTenants(out.Tenants, "")
	out.Permissions = normalizePermissions(out.Permissions)
	out.AuthMethods = append([]string(nil), out.AuthMethods...)
	if membership, ok := p.membershipForTenant(tenantSlug); ok {
		out.Tenants = normalizeTenants(append(out.Tenants, tenantSlug), "")
		if role := normalizeRole(membership.Role); role != "" {
			out.Role = role
		}
		if membership.Permissions != nil {
			out.Permissions = normalizePermissions(membership.Permissions)
		}
	}
	return out
}

func (p userProfile) membershipForTenant(tenantSlug string) (tenantMembership, bool) {
	tenantSlug = normalizeSlug(tenantSlug)
	if tenantSlug == "" {
		return tenantMembership{}, false
	}
	for rawSlug, membership := range p.TenantMemberships {
		if normalizeSlug(rawSlug) == tenantSlug {
			return membership, true
		}
	}
	return tenantMembership{}, false
}

func (p userProfile) AllowsAuthMethod(method string) bool {
	method = normalizeAuthMethod(method)
	if method == "" {
		return false
	}
	methods := p.AuthMethods
	if len(methods) == 0 {
		methods = defaultAuthMethods()
	}
	for _, item := range methods {
		if normalizeAuthMethod(item) == method {
			return true
		}
	}
	return false
}

func (p userProfile) HasTenant(tenantSlug string) bool {
	tenantSlug = normalizeSlug(tenantSlug)
	for _, item := range p.Tenants {
		if normalizeSlug(item) == tenantSlug {
			return true
		}
	}
	_, ok := p.membershipForTenant(tenantSlug)
	if ok {
		return true
	}
	return false
}

func (p userProfile) UserRow() userRow {
	if p.Role == "" {
		p.Role = roleResident
	}
	p.Role = normalizeRole(p.Role)
	if p.Status == "" {
		p.Status = "Eingeladen"
	}
	return userRow{
		Email:            p.Email,
		Title:            p.Title,
		FirstName:        p.FirstName,
		LastName:         p.LastName,
		Phone:            p.Phone,
		DirectoryOptIn:   p.DirectoryOptIn,
		DisplayName:      p.DisplayName(),
		Initials:         p.Initials(),
		Role:             p.Role,
		RoleClass:        roleClass(p.Role),
		RoleCapabilities: roleCapabilityLabels(p.Role),
		Status:           p.Status,
		Tenants:          strings.Join(p.Tenants, ", "),
		PermissionLabel:  permissionLabel(p.Permissions),
		PermissionList:   permissionLabelList(p.Permissions),
		ParkingChecked:   p.HasPermission(permissionParking),
		AuthLabel:        authMethodsLabel(p.AuthMethods),
		AuthList:         authMethodsLabelList(p.AuthMethods),
	}
}

type userRow struct {
	Email            string
	Title            string
	FirstName        string
	LastName         string
	Phone            string
	DirectoryOptIn   bool
	DisplayName      string
	Initials         string
	Role             string
	RoleClass        string
	RoleCapabilities []string
	Status           string
	Tenants          string
	PermissionLabel  string
	PermissionList   []string
	ParkingChecked   bool
	AuthLabel        string
	AuthList         []string
	Editable         bool
	LastSeen         string
}

func (s *tokenStore) Put(token string, email string, tenantSlug string, ttl time.Duration) {
	key := s.digest(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = loginToken{email: email, tenantSlug: tenantSlug, expiresAt: time.Now().Add(ttl)}
}

func (s *tokenStore) Consume(token string) (string, string, bool) {
	if token == "" {
		return "", "", false
	}
	key := s.digest(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[key]
	if !ok || item.used || time.Now().After(item.expiresAt) {
		delete(s.items, key)
		return "", "", false
	}
	item.used = true
	s.items[key] = item
	return item.email, item.tenantSlug, true
}

func (s *tokenStore) digest(token string) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(token))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func newSessionStore(secret []byte) *sessionStore {
	sum := sha256.Sum256(append([]byte("weg-session-aead-v1."), secret...))
	key := make([]byte, len(sum))
	copy(key, sum[:])
	return &sessionStore{
		secret:  key,
		revoked: map[string]time.Time{},
	}
}

func (s *sessionStore) Put(email string, tenantSlug string, authMethod string, ttl time.Duration) (string, time.Time, error) {
	expiresAt := time.Now().Add(ttl)
	item := session{
		Email:      normalizeEmail(email),
		TenantSlug: normalizeSlug(tenantSlug),
		AuthMethod: normalizeAuthMethod(authMethod),
		ExpiresAt:  expiresAt.Unix(),
	}
	if item.Email == "" || item.TenantSlug == "" || item.AuthMethod == "" {
		return "", time.Time{}, fmt.Errorf("invalid session")
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return "", time.Time{}, err
	}
	block, err := aes.NewCipher(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", time.Time{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", time.Time{}, err
	}
	ciphertext := gcm.Seal(nil, nonce, payload, []byte("weg-session-v1"))
	token := "v1." + base64.RawURLEncoding.EncodeToString(nonce) + "." + base64.RawURLEncoding.EncodeToString(ciphertext)
	return token, expiresAt, nil
}

func (s *sessionStore) Get(token string) (string, string, string, bool) {
	item, ok := s.verify(token)
	if !ok {
		return "", "", "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
	if _, revoked := s.revoked[s.revocationKey(token)]; revoked {
		return "", "", "", false
	}
	return item.Email, item.TenantSlug, item.AuthMethod, true
}

func (s *sessionStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
	if item, ok := s.verify(token); ok {
		s.revoked[s.revocationKey(token)] = time.Unix(item.ExpiresAt, 0)
	}
}

func (s *sessionStore) verify(token string) (session, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "v1" {
		return session{}, false
	}
	nonce, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return session{}, false
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return session{}, false
	}
	block, err := aes.NewCipher(s.secret)
	if err != nil {
		return session{}, false
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(nonce) != gcm.NonceSize() {
		return session{}, false
	}
	payload, err := gcm.Open(nil, nonce, ciphertext, []byte("weg-session-v1"))
	if err != nil {
		return session{}, false
	}
	var item session
	if err := json.Unmarshal(payload, &item); err != nil {
		return session{}, false
	}
	item.Email = normalizeEmail(item.Email)
	item.TenantSlug = normalizeSlug(item.TenantSlug)
	item.AuthMethod = normalizeAuthMethod(item.AuthMethod)
	if item.Email == "" || item.TenantSlug == "" || item.AuthMethod == "" || item.ExpiresAt <= time.Now().Unix() {
		return session{}, false
	}
	return item, true
}

func (s *sessionStore) revocationKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (s *sessionStore) cleanupLocked() {
	now := time.Now()
	for key, expiresAt := range s.revoked {
		if now.After(expiresAt) {
			delete(s.revoked, key)
		}
	}
}

func newOIDCLogin(ctx context.Context, issuer string, clientID string, clientSecret string, redirectURL string, providerName string) (*oidcLogin, error) {
	issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	redirectURL = strings.TrimSpace(redirectURL)
	providerName = strings.TrimSpace(providerName)
	if providerName == "" {
		providerName = "Zitadel"
	}
	if issuer == "" && clientID == "" && clientSecret == "" && redirectURL == "" {
		return &oidcLogin{providerName: providerName}, nil
	}
	if issuer == "" || clientID == "" {
		return nil, fmt.Errorf("OIDC_ISSUER and OIDC_CLIENT_ID are required when OIDC is configured")
	}
	if redirectURL != "" {
		parsed, err := url.Parse(redirectURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return nil, fmt.Errorf("OIDC_REDIRECT_URL must be an absolute URL")
		}
	}
	login := &oidcLogin{
		providerName: providerName,
		issuer:       issuer,
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
	}
	if err := login.EnsureProvider(ctx); err != nil {
		log.Printf("oidc discovery unavailable at startup, will retry on login: %v", err)
	}
	return login, nil
}

func (o *oidcLogin) Configured() bool {
	return o != nil && o.issuer != "" && o.clientID != ""
}

func (o *oidcLogin) ProviderName() string {
	if o == nil || o.providerName == "" {
		return "SSO"
	}
	return o.providerName
}

func (o *oidcLogin) RedirectURL(_ *http.Request, _ tenantConfig, baseURL string) string {
	if o.redirectURL != "" {
		return o.redirectURL
	}
	return strings.TrimRight(baseURL, "/") + "/auth/oidc/callback"
}

func (o *oidcLogin) OAuthConfig(redirectURL string) oauth2.Config {
	return oauth2.Config{
		ClientID:     o.clientID,
		ClientSecret: o.clientSecret,
		Endpoint:     o.provider.Endpoint(),
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "email", "profile"},
	}
}

func (o *oidcLogin) EnsureProvider(ctx context.Context) error {
	if !o.Configured() {
		return fmt.Errorf("OIDC is not configured")
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.provider != nil && o.verifier != nil {
		return nil
	}
	provider, err := oidc.NewProvider(ctx, o.issuer)
	if err != nil {
		return fmt.Errorf("OIDC discovery failed")
	}
	o.provider = provider
	o.verifier = provider.Verifier(&oidc.Config{ClientID: o.clientID})
	return nil
}

func (s *oidcFlowStore) Put(state string, flow oidcFlow, ttl time.Duration) {
	flow.expiresAt = time.Now().Add(ttl)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
	s.items[state] = flow
}

func (s *oidcFlowStore) Consume(state string) (oidcFlow, bool) {
	if state == "" {
		return oidcFlow{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
	flow, ok := s.items[state]
	if !ok || flow.used || time.Now().After(flow.expiresAt) {
		delete(s.items, state)
		return oidcFlow{}, false
	}
	flow.used = true
	delete(s.items, state)
	return flow, true
}

func (s *oidcFlowStore) cleanupLocked() {
	now := time.Now()
	for state, flow := range s.items {
		if now.After(flow.expiresAt) {
			delete(s.items, state)
		}
	}
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (c *oidcUserClaims) Merge(other oidcUserClaims) {
	if c.Email == "" {
		c.Email = other.Email
	}
	if c.EmailVerified == nil {
		c.EmailVerified = other.EmailVerified
	}
}

func (m smtpMailer) Configured() bool {
	return m.host != "" && m.port != "" && m.from != ""
}

func (m smtpMailer) Validate() error {
	if !m.Configured() {
		return nil
	}
	if _, err := mail.ParseAddress(m.from); err != nil {
		return fmt.Errorf("invalid MAIL_FROM")
	}
	if (m.user == "") != (m.pass == "") {
		return fmt.Errorf("SMTP_USER and SMTP_PASS must be set together")
	}
	if _, err := strconv.Atoi(m.port); err != nil {
		return fmt.Errorf("SMTP_PORT must be numeric")
	}
	return nil
}

func (m smtpMailer) auth() smtp.Auth {
	if m.user == "" && m.pass == "" {
		return nil
	}
	return smtp.PlainAuth("", m.user, m.pass, m.host)
}

func (m smtpMailer) SendMagicLink(to string, link string) error {
	if !m.Configured() {
		return errors.New("smtp not configured")
	}

	addr := net.JoinHostPort(m.host, m.port)
	fromAddr, err := mail.ParseAddress(m.from)
	if err != nil {
		return fmt.Errorf("invalid MAIL_FROM")
	}

	msg := strings.Join([]string{
		"From: " + m.from,
		"To: " + to,
		"Subject: Ihr Zugang zum WEG Portal",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		"Hallo,",
		"",
		"hier ist Ihr Anmeldelink für das WEG Portal:",
		link,
		"",
		"Der Link ist 15 Minuten gültig und kann nur einmal verwendet werden.",
		"",
		"Freundliche Grüße",
		"WEG Portal",
	}, "\r\n")

	return smtp.SendMail(addr, m.auth(), fromAddr.Address, []string{to}, []byte(msg))
}

func (m smtpMailer) SendInvite(to string, loginURL string, address string) error {
	if !m.Configured() {
		return errors.New("smtp not configured")
	}
	addr := net.JoinHostPort(m.host, m.port)
	fromAddr, err := mail.ParseAddress(m.from)
	if err != nil {
		return fmt.Errorf("invalid MAIL_FROM")
	}
	msg := strings.Join([]string{
		"From: " + m.from,
		"To: " + to,
		"Subject: Einladung zum WEG Portal " + address,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		"Hallo,",
		"",
		"Sie wurden zum WEG Portal \"" + address + "\" eingeladen.",
		"Melden Sie sich mit dieser E-Mail-Adresse an:",
		loginURL,
		"",
		"Beim Anmelden erhalten Sie einen einmaligen Login-Link per E-Mail",
		"oder nutzen Ihren SSO-Zugang.",
		"",
		"Freundliche Grüße",
		"WEG Portal",
	}, "\r\n")
	return smtp.SendMail(addr, m.auth(), fromAddr.Address, []string{to}, []byte(msg))
}

func (m smtpMailer) SendNotification(to string, subject string, body string) error {
	if !m.Configured() {
		return errors.New("smtp not configured")
	}
	addr := net.JoinHostPort(m.host, m.port)
	fromAddr, err := mail.ParseAddress(m.from)
	if err != nil {
		return fmt.Errorf("invalid MAIL_FROM")
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		subject = "WEG Portal Benachrichtigung"
	}
	body = strings.TrimSpace(body)
	if body == "" {
		body = "Es gibt eine neue Aktualisierung im WEG Portal."
	}
	msg := strings.Join([]string{
		"From: " + m.from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
		"",
		"Freundliche Grüße",
		"WEG Portal",
	}, "\r\n")
	return smtp.SendMail(addr, m.auth(), fromAddr.Address, []string{to}, []byte(msg))
}

func newHomeAssistantConfig() homeAssistantConfig {
	return homeAssistantConfig{
		baseURL:           strings.TrimRight(env("HA_BASE_URL", ""), "/"),
		token:             strings.TrimSpace(os.Getenv("HA_TOKEN")),
		meterEnergyEntity: env("PARKING_METER_ENERGY_ENTITY", "sensor.kws_306wf_energy_meter_energy"),
		powerEntity:       env("PARKING_POWER_ENTITY", "sensor.kws360_power"),
		priceEntity:       env("PARKING_PRICE_ENTITY", "sensor.epex_spot_data_total_price"),
	}
}

func (c homeAssistantConfig) State(ctx context.Context, entityID string) (haState, error) {
	if c.baseURL == "" || c.token == "" || entityID == "" {
		return haState{}, errors.New("home assistant not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/states/"+url.PathEscape(entityID), nil)
	if err != nil {
		return haState{}, errors.New("could not build home assistant request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return haState{}, errors.New("home assistant request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return haState{}, fmt.Errorf("home assistant returned %d", resp.StatusCode)
	}

	var state haState
	dec := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	if err := dec.Decode(&state); err != nil {
		return haState{}, errors.New("home assistant returned invalid json")
	}
	return state, nil
}

func (c homeAssistantConfig) History(ctx context.Context, start time.Time, end time.Time, entityIDs []string) (map[string][]haHistoryState, error) {
	if c.baseURL == "" || c.token == "" || len(entityIDs) == 0 {
		return nil, errors.New("home assistant not configured")
	}
	endpoint := c.baseURL + "/api/history/period/" + url.PathEscape(start.UTC().Format(time.RFC3339))
	q := url.Values{}
	q.Set("end_time", end.UTC().Format(time.RFC3339))
	q.Set("filter_entity_id", strings.Join(entityIDs, ","))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, errors.New("could not build home assistant history request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.New("home assistant history request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("home assistant history returned %d", resp.StatusCode)
	}

	var groups [][]haHistoryState
	dec := json.NewDecoder(io.LimitReader(resp.Body, 8<<20))
	if err := dec.Decode(&groups); err != nil {
		return nil, errors.New("home assistant history returned invalid json")
	}
	out := map[string][]haHistoryState{}
	for _, group := range groups {
		for _, item := range group {
			if item.EntityID == "" {
				continue
			}
			out[item.EntityID] = append(out[item.EntityID], item)
		}
	}
	for entityID := range out {
		sort.Slice(out[entityID], func(i, j int) bool {
			return out[entityID][i].timestamp().Before(out[entityID][j].timestamp())
		})
	}
	return out, nil
}

func (c homeAssistantConfig) Statistics(ctx context.Context, start time.Time, end time.Time) ([]parkingNumericSample, []parkingNumericSample, error) {
	if c.baseURL == "" || c.token == "" || c.meterEnergyEntity == "" || c.priceEntity == "" {
		return nil, nil, errors.New("home assistant not configured")
	}
	wsURL, err := c.websocketURL()
	if err != nil {
		return nil, nil, err
	}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, nil, errors.New("home assistant websocket connection failed")
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetReadDeadline(deadline)
		_ = conn.SetWriteDeadline(deadline)
	}

	var authMessage struct {
		Type string `json:"type"`
	}
	if err := conn.ReadJSON(&authMessage); err != nil {
		return nil, nil, errors.New("home assistant websocket auth start failed")
	}
	if authMessage.Type == "auth_required" {
		if err := conn.WriteJSON(map[string]string{
			"type":         "auth",
			"access_token": c.token,
		}); err != nil {
			return nil, nil, errors.New("home assistant websocket auth failed")
		}
		if err := conn.ReadJSON(&authMessage); err != nil {
			return nil, nil, errors.New("home assistant websocket auth response failed")
		}
	}
	if authMessage.Type != "auth_ok" {
		return nil, nil, errors.New("home assistant websocket auth rejected")
	}

	const requestID = 1
	if err := conn.WriteJSON(map[string]any{
		"id":            requestID,
		"type":          "recorder/statistics_during_period",
		"start_time":    start.UTC().Format(time.RFC3339),
		"end_time":      end.UTC().Format(time.RFC3339),
		"statistic_ids": []string{c.meterEnergyEntity, c.priceEntity},
		"period":        "hour",
		"types":         []string{"state", "sum", "mean"},
	}); err != nil {
		return nil, nil, errors.New("home assistant websocket statistics request failed")
	}

	for {
		var response struct {
			ID      int                      `json:"id"`
			Type    string                   `json:"type"`
			Success bool                     `json:"success"`
			Error   map[string]any           `json:"error"`
			Result  map[string][]haStatistic `json:"result"`
		}
		if err := conn.ReadJSON(&response); err != nil {
			return nil, nil, errors.New("home assistant websocket statistics response failed")
		}
		if response.ID != requestID {
			continue
		}
		if response.Type != "result" || !response.Success {
			return nil, nil, errors.New("home assistant websocket statistics rejected")
		}
		energySamples := samplesFromStatistics(response.Result[c.meterEnergyEntity], "state", "sum")
		priceSamples := samplesFromStatistics(response.Result[c.priceEntity], "state", "mean")
		return energySamples, priceSamples, nil
	}
}

func (c homeAssistantConfig) websocketURL() (string, error) {
	parsed, err := url.Parse(c.baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("invalid home assistant base url")
	}
	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	default:
		return "", errors.New("unsupported home assistant websocket scheme")
	}
	parsed.Path = "/api/websocket"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func (s haHistoryState) timestamp() time.Time {
	if !s.LastChanged.IsZero() {
		return s.LastChanged
	}
	return s.LastUpdated
}

func formatHAValue(state haState) string {
	unit, _ := state.Attributes["unit_of_measurement"].(string)
	value := strings.TrimSpace(state.State)
	if value == "" {
		value = "unbekannt"
	}
	if number, err := parseHAFloat(value); err == nil {
		switch unit {
		case "kWh":
			return formatDecimal(number, 2) + " kWh"
		case "W":
			return formatDecimal(number, 1) + " W"
		case "€/kWh", "EUR/kWh":
			return formatDecimal(number, 6) + " €/kWh"
		case "€", "EUR":
			return formatDecimal(number, 2) + " €"
		}
		if unit != "" {
			return formatDecimal(number, 2) + " " + unit
		}
		return formatDecimal(number, 2)
	}
	if unit == "" {
		return value
	}
	return value + " " + unit
}

func parseHAFloat(raw string) (float64, error) {
	value := strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	if value == "" || strings.EqualFold(value, "unknown") || strings.EqualFold(value, "unavailable") {
		return 0, errors.New("state is not numeric")
	}
	return strconv.ParseFloat(value, 64)
}

func parseDecimal(raw string) (float64, error) {
	value := strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	if value == "" {
		return 0, errors.New("empty decimal")
	}
	return strconv.ParseFloat(value, 64)
}

func parseDuration(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	if duration, err := time.ParseDuration(raw); err == nil {
		return duration, nil
	}
	minutes, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	return time.Duration(minutes) * time.Minute, nil
}

func parseHistoryStart(raw string, now time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.Local), nil
	}
	if t, err := time.ParseInLocation("2006-01-02", raw, time.Local); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, raw)
}

const (
	deATDateLayout          = "02.01.2006"
	deATDateTimeLayout      = "02.01.2006 15:04"
	deATShortDateTimeLayout = "02.01. 15:04"
	deATTimeLayout          = "15:04"
	htmlDateTimeLocalLayout = "2006-01-02T15:04"
)

// User-facing formatting convention: de-AT copy uses local time, dot-grouped
// thousands and comma decimals. HTML control values use browser-native layouts.
func formatLocalDate(t time.Time) string {
	return formatDateTimeIn(t, time.Local, deATDateLayout)
}

func formatLocalDateTime(t time.Time) string {
	return formatDateTimeIn(t, time.Local, deATDateTimeLayout)
}

func formatLocalShortDateTime(t time.Time) string {
	return formatDateTimeIn(t, time.Local, deATShortDateTimeLayout)
}

func formatLocalTime(t time.Time) string {
	return formatDateTimeIn(t, time.Local, deATTimeLayout)
}

func formatLocalDateTimeInput(t time.Time) string {
	return formatDateTimeIn(t, time.Local, htmlDateTimeLocalLayout)
}

func formatDateTimeIn(t time.Time, loc *time.Location, layout string) string {
	if loc == nil {
		loc = time.Local
	}
	return t.In(loc).Format(layout)
}

func formatInputFloat(value float64) string {
	return formatDecimal(value, 3)
}

func formatEUR(value float64) string {
	return formatDecimal(value, 2) + " €"
}

func formatEURPerKWh(value float64) string {
	return formatDecimal(value, 3) + " €/kWh"
}

func formatKWh(value float64) string {
	return formatDecimal(value, 2) + " kWh"
}

func formatPreciseEUR(value float64) string {
	return formatDecimal(value, 6) + " €"
}

func formatPreciseEURPerKWh(value float64) string {
	return formatDecimal(value, 6) + " €/kWh"
}

func formatPreciseKWh(value float64) string {
	return formatDecimal(value, 6) + " kWh"
}

func formatDecimal(value float64, decimals int) string {
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

func formatMonthLabel(month string, loc *time.Location) string {
	t, err := time.ParseInLocation("2006-01", month, loc)
	if err != nil {
		return month
	}
	names := []string{"Jänner", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"}
	return names[int(t.Month())-1] + " " + strconv.Itoa(t.Year())
}

func unitCountLabel(count int) string {
	if count == 1 {
		return "Wohneinheit"
	}
	return "Wohneinheiten"
}

func formatPeriodLabel(first time.Time, last time.Time, loc *time.Location) string {
	if first.IsZero() || last.IsZero() {
		return "Noch keine Messwerte"
	}
	if loc == nil {
		loc = time.Local
	}
	return formatDateTimeIn(first, loc, deATShortDateTimeLayout) + " bis " + formatDateTimeIn(last, loc, deATShortDateTimeLayout)
}

func paidLabel(paid bool) string {
	if paid {
		return "BEZAHLT"
	}
	return "OFFEN"
}

func togglePaidLabel(paid bool) string {
	if paid {
		return "Als offen markieren"
	}
	return "Als bezahlt markieren"
}

func boolFormValue(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; connect-src 'self' https://fonts.googleapis.com https://fonts.gstatic.com; form-action 'self'; base-uri 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func env(key string, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func loadLocalEnv(path string) error {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("could not read local env file")
	}

	for i, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("invalid local env line %d", i+1)
		}
		key = strings.TrimSpace(key)
		if key == "" || strings.ContainsAny(key, " \t") {
			return fmt.Errorf("invalid local env key on line %d", i+1)
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, trimEnvQuotes(strings.TrimSpace(value)))
	}
	return nil
}

func trimEnvQuotes(value string) string {
	if len(value) < 2 {
		return value
	}
	if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}

func parseBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func isLocalHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func parseAllowed(raw string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, item := range strings.Split(raw, ",") {
		email := normalizeEmail(item)
		if email != "" {
			out[email] = struct{}{}
		}
	}
	return out
}

func parseTenants(raw string, rootDomain string, defaultTenant string, defaultHA homeAssistantConfig) (map[string]tenantConfig, error) {
	out := map[string]tenantConfig{}
	raw = strings.TrimSpace(raw)
	if raw != "" {
		var tenants []tenantConfig
		if err := json.Unmarshal([]byte(raw), &tenants); err != nil {
			return nil, fmt.Errorf("invalid WEG_TENANTS_JSON")
		}
		for _, tenant := range tenants {
			tenant.Slug = normalizeSlug(tenant.Slug)
			if tenant.Slug == "" {
				return nil, fmt.Errorf("tenant is missing slug")
			}
			if tenant.Name == "" {
				tenant.Name = "WEG Portal"
			}
			if tenant.Address == "" {
				tenant.Address = tenant.Slug
			}
			tenant.ContactName = strings.TrimSpace(tenant.ContactName)
			tenant.ContactEmail = normalizeEmail(tenant.ContactEmail)
			tenant.ContactPhone = strings.TrimSpace(tenant.ContactPhone)
			tenant.EmergencyName = strings.TrimSpace(tenant.EmergencyName)
			tenant.EmergencyPhone = strings.TrimSpace(tenant.EmergencyPhone)
			tenant.CaretakerName = strings.TrimSpace(tenant.CaretakerName)
			tenant.CaretakerEmail = normalizeEmail(tenant.CaretakerEmail)
			tenant.CaretakerPhone = strings.TrimSpace(tenant.CaretakerPhone)
			if tenant.HeroImageURL == "" {
				tenant.HeroImageURL = defaultTenantHeroImageURL
			}
			tenant.Host = normalizeHost(tenant.Host)
			if tenant.Host == "" && rootDomain != "" {
				tenant.Host = tenant.Slug + "." + rootDomain
			}
			if tenant.HA.baseURL == "" && tenant.Slug == normalizeSlug(defaultTenant) {
				tenant.HA = defaultHA
			}
			out[tenant.Slug] = tenant
		}
	}

	defaultTenant = normalizeSlug(defaultTenant)
	if _, ok := out[defaultTenant]; !ok {
		host := ""
		if rootDomain != "" {
			host = defaultTenant + "." + rootDomain
		}
		out[defaultTenant] = tenantConfig{
			Slug:         defaultTenant,
			Name:         "WEG Portal",
			Address:      "Janischhofweg 22",
			HeroImageURL: defaultTenantHeroImageURL,
			Host:         host,
			HA:           defaultHA,
		}
	}
	return out, nil
}

func parseUserProfiles(raw string, allowed map[string]struct{}, admins map[string]struct{}, defaultTenant string) (map[string]userProfile, error) {
	out := map[string]userProfile{}
	defaultTenant = normalizeSlug(defaultTenant)
	raw = strings.TrimSpace(raw)
	if raw != "" {
		var profiles []userProfile
		if err := json.Unmarshal([]byte(raw), &profiles); err != nil {
			return nil, fmt.Errorf("invalid WEG_USERS_JSON")
		}
		for _, profile := range profiles {
			email := normalizeEmail(profile.Email)
			if email == "" {
				return nil, fmt.Errorf("user profile is missing email")
			}
			if _, err := mail.ParseAddress(email); err != nil {
				return nil, fmt.Errorf("user profile has invalid email")
			}
			profile.Email = email
			profile.Title = strings.TrimSpace(profile.Title)
			profile.FirstName = strings.TrimSpace(profile.FirstName)
			profile.LastName = strings.TrimSpace(profile.LastName)
			profile.Phone = strings.TrimSpace(profile.Phone)
			profile.Role = normalizeRole(profile.Role)
			if profile.Role == "" {
				if _, ok := admins[email]; ok {
					profile.Role = roleAdmin
				} else {
					profile.Role = roleResident
				}
			}
			if profile.Status == "" {
				profile.Status = "Eingeladen"
			}
			profile.TenantMemberships = normalizeTenantMemberships(profile.TenantMemberships)
			profile.Tenants = normalizeTenants(append(profile.Tenants, tenantMembershipSlugs(profile.TenantMemberships)...), defaultTenant)
			profile.Permissions = normalizePermissions(profile.Permissions)
			authMethods, err := normalizeAuthMethods(profile.AuthMethods)
			if err != nil {
				return nil, err
			}
			profile.AuthMethods = authMethods
			out[email] = profile
		}
	}

	for email := range admins {
		if _, ok := out[email]; ok {
			continue
		}
		out[email] = userProfile{Email: email, Role: roleAdmin, Status: "Aktiv", Tenants: []string{defaultTenant}, AuthMethods: defaultAuthMethods()}
	}
	for email := range allowed {
		if _, ok := out[email]; ok {
			continue
		}
		out[email] = userProfile{Email: email, Role: roleResident, Status: "Eingeladen", Tenants: []string{defaultTenant}, AuthMethods: defaultAuthMethods()}
	}
	return out, nil
}

type inviteStore struct {
	path string
	mu   sync.Mutex
	data inviteStoreData
}

type inviteStoreData struct {
	Invites []userProfile `json:"invites"`
}

func newInviteStore(path string) (*inviteStore, error) {
	store := &inviteStore{path: path, data: inviteStoreData{Invites: []userProfile{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read invite data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid invite data")
	}
	return store, nil
}

func (s *inviteStore) Get(email string) (userProfile, bool) {
	email = normalizeEmail(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, profile := range s.data.Invites {
		if normalizeEmail(profile.Email) == email {
			return profile, true
		}
	}
	return userProfile{}, false
}

func (s *inviteStore) List() []userProfile {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]userProfile(nil), s.data.Invites...)
}

// Add persists a new invite. Returns false (no error) when the email is already
// invited. Callers must ensure the email is not already in the env directory.
func (s *inviteStore) Add(profile userProfile) (bool, error) {
	profile.Email = normalizeEmail(profile.Email)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.data.Invites {
		if normalizeEmail(existing.Email) == profile.Email {
			return false, nil
		}
	}
	s.data.Invites = append(s.data.Invites, profile)
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *inviteStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("could not create invite data directory")
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode invite data")
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("could not write invite data")
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("could not replace invite data")
	}
	return nil
}

// Update replaces the invite keyed by oldEmail with updated. Returns false (no
// error) when oldEmail is not a persisted invite. When the email changes it
// must not collide with another invite (caller also checks the env directory).
func (s *inviteStore) Update(oldEmail string, updated userProfile) (bool, error) {
	oldEmail = normalizeEmail(oldEmail)
	updated.Email = normalizeEmail(updated.Email)
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i, existing := range s.data.Invites {
		if normalizeEmail(existing.Email) == oldEmail {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false, nil
	}
	if updated.Email != oldEmail {
		for i, existing := range s.data.Invites {
			if i != idx && normalizeEmail(existing.Email) == updated.Email {
				return false, fmt.Errorf("email already invited")
			}
		}
	}
	s.data.Invites[idx] = updated
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

// Delete removes the invite for email. Returns whether one was removed.
func (s *inviteStore) Delete(email string) (bool, error) {
	email = normalizeEmail(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.data.Invites[:0]
	removed := false
	for _, existing := range s.data.Invites {
		if normalizeEmail(existing.Email) == email {
			removed = true
			continue
		}
		kept = append(kept, existing)
	}
	if !removed {
		return false, nil
	}
	s.data.Invites = kept
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

type activityStore struct {
	path string
	mu   sync.Mutex
	data map[string]activityRecord
}

type activityRecord struct {
	LastLogin  time.Time `json:"last_login"`
	AuthMethod string    `json:"auth_method,omitempty"`
}

func newActivityStore(path string) (*activityStore, error) {
	store := &activityStore{path: path, data: map[string]activityRecord{}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read activity data")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(raw, &store.data); err != nil {
		return nil, fmt.Errorf("invalid activity data")
	}
	if store.data == nil {
		store.data = map[string]activityRecord{}
	}
	return store, nil
}

// Touch records a successful login. Best-effort: callers log failures but do
// not block login on a persistence error.
func (s *activityStore) Touch(email string, at time.Time, authMethod string) error {
	email = normalizeEmail(email)
	if email == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[email] = activityRecord{LastLogin: at.UTC(), AuthMethod: authMethod}
	return s.saveLocked()
}

func (s *activityStore) Get(email string) (activityRecord, bool) {
	email = normalizeEmail(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.data[email]
	return rec, ok
}

func (s *activityStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("could not create activity data directory")
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode activity data")
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("could not write activity data")
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("could not replace activity data")
	}
	return nil
}

type auditStore struct {
	path    string
	mu      sync.Mutex
	entries []auditEvent
}

func newAuditStore(path string) (*auditStore, error) {
	store := &auditStore{path: path, entries: []auditEvent{}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read audit data")
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return store, nil
	}
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &store.entries); err != nil {
			return nil, fmt.Errorf("invalid audit data")
		}
		for i := range store.entries {
			store.entries[i] = normalizeAuditEvent(store.entries[i])
		}
		return store, nil
	}
	for i, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event auditEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("invalid audit data on line %d", i+1)
		}
		store.entries = append(store.entries, normalizeAuditEvent(event))
	}
	return store, nil
}

func (s *auditStore) Append(event auditEvent) error {
	if s == nil {
		return nil
	}
	event = normalizeAuditEvent(event)
	if event.Action == "" {
		return nil
	}
	raw, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("could not encode audit event")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.path != "" {
		if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
			return fmt.Errorf("could not create audit data directory")
		}
		f, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return fmt.Errorf("could not open audit data")
		}
		if _, err := f.Write(append(raw, '\n')); err != nil {
			_ = f.Close()
			return fmt.Errorf("could not append audit data")
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("could not close audit data")
		}
	}
	s.entries = append(s.entries, copyAuditEvent(event))
	return nil
}

func (s *auditStore) List(filter auditFilter) []auditEvent {
	if s == nil {
		return nil
	}
	filter.TenantSlug = normalizeSlug(filter.TenantSlug)
	filter.Action = normalizeAuditAction(filter.Action)
	filter.Query = strings.ToLower(strings.TrimSpace(filter.Query))
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 200
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]auditEvent, 0, min(filter.Limit, len(s.entries)))
	for i := len(s.entries) - 1; i >= 0 && len(out) < filter.Limit; i-- {
		event := s.entries[i]
		if filter.TenantSlug != "" && normalizeSlug(event.TenantSlug) != filter.TenantSlug {
			continue
		}
		if filter.Action != "" && normalizeAuditAction(event.Action) != filter.Action {
			continue
		}
		if filter.Query != "" && !auditEventMatches(event, filter.Query) {
			continue
		}
		out = append(out, copyAuditEvent(event))
	}
	return out
}

func normalizeAuditEvent(event auditEvent) auditEvent {
	event.TenantSlug = normalizeSlug(event.TenantSlug)
	event.ActorEmail = normalizeEmail(event.ActorEmail)
	event.ActorRole = normalizeRole(event.ActorRole)
	event.Action = normalizeAuditAction(event.Action)
	event.TargetType = strings.TrimSpace(event.TargetType)
	event.TargetID = strings.TrimSpace(event.TargetID)
	event.Summary = truncateAuditValue(event.Summary, 220)
	event.Details = sanitizeAuditDetails(event.Details)
	if event.At.IsZero() {
		event.At = time.Now()
	}
	event.At = event.At.UTC().Truncate(time.Second)
	return event
}

func normalizeAuditAction(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	switch raw {
	case auditActionLogin, auditActionInviteCreate, auditActionInviteUpdate, auditActionInviteDelete,
		auditActionBuildingUpdate, auditActionHeroUpdate, auditActionUnitSave, auditActionUnitDelete,
		auditActionDocumentUpload, auditActionDocumentDownload, auditActionDocumentReplace,
		auditActionVoteCreate, auditActionVoteOpen, auditActionVoteClose, auditActionVoteCast, auditActionVoteReminder,
		auditActionParkingSettings, auditActionParkingMonth, auditActionIssueWorkflow:
		return raw
	default:
		return ""
	}
}

func sanitizeAuditDetails(details map[string]string) map[string]string {
	if len(details) == 0 {
		return nil
	}
	out := map[string]string{}
	for key, value := range details {
		key = strings.ToLower(strings.TrimSpace(key))
		key = strings.ReplaceAll(key, " ", "_")
		if key == "" || auditDetailKeySensitive(key) {
			continue
		}
		value = truncateAuditValue(value, 180)
		if value == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func auditDetailKeySensitive(key string) bool {
	key = strings.ToLower(key)
	for _, marker := range []string{"secret", "token", "password", "passwd", "private_key", "client_secret"} {
		if strings.Contains(key, marker) {
			return true
		}
	}
	return false
}

func truncateAuditValue(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "…"
}

func copyAuditEvent(event auditEvent) auditEvent {
	if event.Details != nil {
		details := make(map[string]string, len(event.Details))
		for key, value := range event.Details {
			details[key] = value
		}
		event.Details = details
	}
	return event
}

func auditEventMatches(event auditEvent, query string) bool {
	haystack := strings.ToLower(strings.Join([]string{
		event.ActorEmail,
		event.ActorRole,
		event.Action,
		event.TargetType,
		event.TargetID,
		event.Summary,
		strings.Join(auditDetailValues(event.Details), " "),
	}, " "))
	return strings.Contains(haystack, query)
}

func auditDetailValues(details map[string]string) []string {
	values := make([]string, 0, len(details))
	for key, value := range details {
		values = append(values, key, value)
	}
	return values
}

func normalizeRole(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "admin", "administrator", "platform-admin", "platform_admin":
		return roleAdmin
	case "verwalter", "verwaltung", "hausverwaltung", "manager", "property-manager", "property_manager", "property manager":
		return roleManager
	case "eigentuemer", "eigentümer", "wohnungseigentuemer", "wohnungseigentümer", "owner", "homeowner", "property-owner", "property_owner":
		return roleOwner
	case "mieter", "tenant", "renter", "lessee":
		return roleRenter
	case "beirat", "board", "advisory-board", "advisory_board", "committee":
		return roleBeirat
	case "bewohner", "resident", "user":
		return roleResident
	default:
		return strings.TrimSpace(raw)
	}
}

func normalizePermissions(raw []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, item := range raw {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func normalizeTenantMemberships(raw map[string]tenantMembership) map[string]tenantMembership {
	if len(raw) == 0 {
		return nil
	}
	out := map[string]tenantMembership{}
	for slug, membership := range raw {
		slug = normalizeSlug(slug)
		if slug == "" {
			continue
		}
		membership.Role = normalizeRole(membership.Role)
		if membership.Permissions != nil {
			membership.Permissions = normalizePermissions(membership.Permissions)
		}
		out[slug] = membership
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func tenantMembershipSlugs(memberships map[string]tenantMembership) []string {
	slugs := []string{}
	for slug := range memberships {
		slug = normalizeSlug(slug)
		if slug != "" {
			slugs = append(slugs, slug)
		}
	}
	return slugs
}

func defaultAuthMethods() []string {
	return []string{authMethodEmail, authMethodOIDC}
}

func normalizeAuthMethods(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return defaultAuthMethods(), nil
	}
	seen := map[string]struct{}{}
	out := []string{}
	for _, item := range raw {
		method := normalizeAuthMethod(item)
		if method == "" {
			return nil, fmt.Errorf("invalid auth method %q", item)
		}
		if _, ok := seen[method]; ok {
			continue
		}
		seen[method] = struct{}{}
		out = append(out, method)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one auth method is required")
	}
	sort.Strings(out)
	return out, nil
}

func normalizeAuthMethod(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case authMethodEmail, "mail", "magic", "magic-link", "magic_link":
		return authMethodEmail
	case authMethodOIDC, "sso", "zitadel", "citatel":
		return authMethodOIDC
	default:
		return ""
	}
}

func authMethodsLabel(methods []string) string {
	return strings.Join(authMethodsLabelList(methods), ", ")
}

func authMethodsLabelList(methods []string) []string {
	normalized, err := normalizeAuthMethods(methods)
	if err != nil {
		return []string{"Ungültig"}
	}
	labels := make([]string, 0, len(normalized))
	for _, method := range normalized {
		switch method {
		case authMethodEmail:
			labels = append(labels, "E-Mail-Link")
		case authMethodOIDC:
			labels = append(labels, "Zitadel SSO")
		default:
			labels = append(labels, method)
		}
	}
	return labels
}

func normalizeTenants(raw []string, fallback string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, item := range raw {
		item = normalizeSlug(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	if len(out) == 0 && fallback != "" {
		out = append(out, normalizeSlug(fallback))
	}
	sort.Strings(out)
	return out
}

func normalizeSlug(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, "_", "-")
	return raw
}

func normalizeUnitID(raw string) string {
	raw = normalizeSlug(raw)
	raw = strings.Join(strings.Fields(raw), "-")
	raw = strings.ReplaceAll(raw, "/", "-")
	return raw
}

func normalizeUnits(raw []unit, fallbackTenant string) []unit {
	out := make([]unit, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		item.TenantSlug = normalizeSlug(firstNonEmpty(item.TenantSlug, fallbackTenant))
		item.ID = normalizeUnitID(item.ID)
		item.Label = strings.TrimSpace(item.Label)
		if item.ID == "" && item.Label != "" {
			item.ID = normalizeUnitID(item.Label)
		}
		if item.Label == "" {
			item.Label = item.ID
		}
		if item.TenantSlug == "" || item.ID == "" {
			continue
		}
		if item.MiteigentumsanteilPPM < 0 {
			item.MiteigentumsanteilPPM = 0
		}
		item.OwnerEmails = normalizeEmailList(item.OwnerEmails)
		item.RenterEmails = normalizeEmailList(item.RenterEmails)
		key := item.TenantSlug + "/" + item.ID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	sortUnits(out)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizeEmailList(raw []string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, item := range raw {
		email := normalizeEmail(item)
		if email == "" {
			continue
		}
		if _, ok := seen[email]; ok {
			continue
		}
		seen[email] = struct{}{}
		out = append(out, email)
	}
	sort.Strings(out)
	return out
}

func uniqueEmails(raw []string) []string {
	return normalizeEmailList(raw)
}

func excludeEmail(raw []string, excluded string) []string {
	excluded = normalizeEmail(excluded)
	out := []string{}
	for _, email := range uniqueEmails(raw) {
		if email == "" || email == excluded {
			continue
		}
		out = append(out, email)
	}
	return out
}

func emailListContains(list []string, email string) bool {
	email = normalizeEmail(email)
	for _, item := range list {
		if normalizeEmail(item) == email {
			return true
		}
	}
	return false
}

func copyUnit(item unit) unit {
	item.OwnerEmails = append([]string(nil), item.OwnerEmails...)
	item.RenterEmails = append([]string(nil), item.RenterEmails...)
	return item
}

func sortUnits(units []unit) {
	sort.Slice(units, func(i, j int) bool {
		return unitLess(units[i], units[j])
	})
}

func unitLess(a unit, b unit) bool {
	if a.TenantSlug != b.TenantSlug {
		return a.TenantSlug < b.TenantSlug
	}
	if strings.ToLower(a.Label) != strings.ToLower(b.Label) {
		return strings.ToLower(a.Label) < strings.ToLower(b.Label)
	}
	return a.ID < b.ID
}

func normalizeHost(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(raw)
	if err == nil {
		raw = host
	}
	return strings.TrimSuffix(raw, ".")
}

func permissionLabel(permissions []string) string {
	return strings.Join(permissionLabelList(permissions), ", ")
}

func parsePermissionForm(values url.Values) []string {
	allowed := map[string]struct{}{
		permissionParking: {},
	}
	out := []string{}
	for _, permission := range normalizePermissions(values["permissions"]) {
		if _, ok := allowed[permission]; ok {
			out = append(out, permission)
		}
	}
	return out
}

func permissionLabelList(permissions []string) []string {
	labels := []string{}
	for _, permission := range normalizePermissions(permissions) {
		switch permission {
		case permissionParking:
			labels = append(labels, "Parkplatznutzung")
		default:
			labels = append(labels, permission)
		}
	}
	if len(labels) == 0 {
		return []string{"Standard"}
	}
	return labels
}

func normalizeEmail(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func randomToken(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func sessionSecret(requireConfigured bool) ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv("SESSION_KEY"))
	if raw == "" {
		if requireConfigured {
			return nil, fmt.Errorf("SESSION_KEY is required when BASE_URL is public")
		}
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return nil, err
		}
		return secret, nil
	}
	if n, err := strconv.Atoi(raw); err == nil && n == 0 {
		return nil, fmt.Errorf("SESSION_KEY must not be empty")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err == nil && len(decoded) >= 32 {
		return decoded, nil
	}
	if len(raw) < 32 {
		return nil, fmt.Errorf("SESSION_KEY must be at least 32 bytes or base64url-encoded 32 bytes")
	}
	return []byte(raw), nil
}

func redactedEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "<redacted>"
	}
	name := parts[0]
	if len(name) > 1 {
		name = name[:1] + "***"
	} else {
		name = "***"
	}
	return name + "@" + parts[1]
}

const pageTemplates = `
{{define "designTokens"}}
      /* Design tokens: colors, spacing, radius, shadows and typography used by shared components. */
      --ink:#20251f; --muted:#6b6f63; --soft:#9a9485;
      --line:#e7e0d2; --paper:#f7f3ea; --panel:#fffefb; --panel-soft:#fbf8f0;
      --gold:#c8993f; --gold-ink:#8a7b3f; --gold-light:#e7c574; --leaf:#2f6b4a;
      --nav:#172019; --nav-2:#20291f;
      --font-sans: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      --font-serif: Spectral, serif;
      --space-1:4px; --space-2:8px; --space-3:12px; --space-4:16px; --space-5:20px; --space-6:24px;
      --radius-xs:7px; --radius-sm:8px; --radius-md:10px; --radius-lg:12px; --radius-xl:14px; --radius-pill:999px;
      --shadow-panel:0 12px 30px rgba(32,37,31,.04);
      --shadow-dialog:0 28px 70px rgba(0,0,0,.34);
      --shadow-login:0 28px 70px rgba(0,0,0,.42);
      font-family: var(--font-sans);
{{end}}
{{define "home"}}
<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=Spectral:wght@400;500;600;700&display=swap" rel="stylesheet">
  <style>
    :root {
      color-scheme: light;
{{template "designTokens" .}}
    }
    * { box-sizing: border-box; }
    body { margin: 0; color: var(--ink); background: #10160f; }
    /* Accessibility convention: all keyboard-reachable controls keep a visible focus ring. */
    :where(a, button, input, select, textarea, summary, [tabindex]):focus-visible { outline: 3px solid var(--gold-light); outline-offset: 3px; }
    .hero { position: relative; min-height: 100vh; overflow: hidden; display: grid; grid-template-rows: auto 1fr auto; }
    .hero::before {
      content: ""; position: absolute; inset: -16px;
      background: url('{{.Tenant.HeroImageURL}}') center 42% / cover no-repeat;
      filter: blur(3px) brightness(.74) saturate(.95); transform: scale(1.05); z-index: -2;
    }
    .hero::after {
      content: ""; position: absolute; inset: 0;
      background: linear-gradient(180deg, rgba(16,22,16,.52) 0%, rgba(16,22,16,.3) 34%, rgba(16,22,16,.6) 76%, rgba(16,22,16,.9) 100%);
      z-index: -1;
    }
    header { display: flex; justify-content: space-between; align-items: center; gap: 24px; padding: 28px clamp(20px,5vw,72px); color: #fff; }
    .brand { display: inline-flex; align-items: center; gap: 12px; text-decoration: none; color: #fff; }
    .mark { min-width: 46px; height: 40px; border-radius: var(--radius-sm); background: rgba(255,255,255,.16); border: 1px solid rgba(255,255,255,.4); backdrop-filter: blur(6px); display: grid; place-items: center; padding: 0 9px; color: #fff; font-weight: 700; font-size: 13px; }
    .brand .name { font-family: var(--font-serif); font-weight: 600; font-size: 17px; }
    nav { display: flex; gap: 24px; color: rgba(255,255,255,.92); font-size: 14px; font-weight: 600; }
    nav a { color: inherit; text-decoration: none; padding-bottom: 4px; border-bottom: 1px solid rgba(231,197,116,.65); }
    nav a:hover { color: #fff; border-bottom-color: var(--gold-light); }
    main { display: grid; grid-template-columns: minmax(0,1.1fr) minmax(320px,420px); gap: clamp(28px,6vw,64px); align-items: end; padding: 0 clamp(20px,5vw,72px) clamp(40px,8vh,72px); }
    .copy { max-width: 760px; color: #fff; }
    .eyebrow { font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: .2em; color: var(--gold-light); margin-bottom: 18px; }
    h1 { margin: 0; font-family: var(--font-serif); font-weight: 500; font-size: clamp(46px,7vw,72px); line-height: 1.0; letter-spacing: -.01em; text-shadow: 0 2px 30px rgba(0,0,0,.3); }
    .lead { max-width: 440px; margin: 24px 0 0; font-size: clamp(17px,2vw,19px); line-height: 1.55; color: rgba(255,255,255,.9); }
    .meta { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 18px; margin-top: 34px; max-width: 620px; }
    .meta div { border-left: 1px solid rgba(231,197,116,.58); padding-left: 16px; min-width: 0; }
    .meta strong { display: block; font-weight: 700; font-size: 14px; color: #fff; margin-bottom: 6px; }
    .meta span { color: rgba(255,255,255,.78); font-size: 13px; line-height: 1.4; }
    .login { background: var(--paper); border-radius: var(--radius-xl); padding: 30px; box-shadow: var(--shadow-login); }
    .login h2 { margin: 0; font-family: var(--font-serif); font-weight: 600; font-size: 26px; }
    .login p { color: var(--muted); line-height: 1.5; margin: 11px 0 22px; font-size: 14.5px; }
    label { display: block; font-size: 11.5px; font-weight: 700; text-transform: uppercase; letter-spacing: .07em; color: var(--gold-ink); margin-bottom: 8px; }
    input { width: 100%; border: 1px solid #e2dac9; border-radius: 10px; padding: 14px 15px; font: inherit; background: #fffefb; color: var(--ink); }
    button { width: 100%; border: 0; border-radius: 10px; padding: 15px 16px; margin-top: 13px; font: inherit; font-weight: 700; color: #fff; background: var(--ink); cursor: pointer; }
    button:hover { background: #000; }
    .notice { border: 1px solid rgba(32,37,31,.16); background: rgba(200,153,63,.1); color: #6a5320; border-radius: 10px; padding: 12px 14px; font-size: 14px; line-height: 1.4; margin-bottom: 16px; }
    .notice.warn { border-color: rgba(173,92,27,.22); background: rgba(231,197,116,.2); color: #6c491a; }
    .dev-link { display: block; border: 1px solid var(--line); background: #fffefb; color: var(--ink); border-radius: 10px; padding: 12px 14px; margin: -4px 0 16px; text-align: center; text-decoration: none; font-size: 14px; font-weight: 700; }
    .dev-link:hover { border-color: var(--gold); }
    .sso-button { display: flex; align-items: center; justify-content: center; min-height: 48px; border-radius: 10px; background: var(--ink); color: #fff; text-decoration: none; font-weight: 700; margin-bottom: 14px; }
    .sso-button:hover { background: #000; }
    .divider { display: grid; grid-template-columns: 1fr auto 1fr; align-items: center; gap: 10px; color: var(--muted); font-size: 13px; margin: 12px 0; }
    .divider::before, .divider::after { content: ""; height: 1px; background: var(--line); }
    .foot-note { margin: 16px 0 0; font-size: 13px; line-height: 1.4; color: var(--soft); }
    footer { padding: 20px clamp(20px,5vw,72px) 26px; color: rgba(255,255,255,.85); font-weight: 500; font-size: 14px; }
    footer .version { margin-left: 8px; color: rgba(255,255,255,.54); font-size: 12px; }
    @media (max-width: 860px) {
      nav { display: none; }
      main { grid-template-columns: 1fr; align-items: start; gap: 28px; }
      .meta { grid-template-columns: 1fr; gap: 12px; max-width: 320px; margin-top: 22px; }
      .meta div:not(:first-child) { display: none; }
      h1 { font-size: clamp(40px,12vw,56px); }
    }
  </style>
</head>
<body>
  <section class="hero">
    <header>
      <a class="brand" href="/" aria-label="WEG Portal Startseite"><span class="mark">WEG</span><span class="name">{{.Tenant.Name}}</span></a>
      <nav aria-label="Seitennavigation">
        <a href="#login">Anmelden</a>
      </nav>
    </header>
    <main>
      <div class="copy">
        <div class="eyebrow">WEG Portal</div>
        <h1>Alles rund um unser gemeinsames Haus.</h1>
        <p class="lead">Der private digitale Eingang für die Hausgemeinschaft, erreichbar per persönlichem E-Mail-Zugang oder SSO.</p>
        <div class="meta" aria-label="Portalüberblick">
          {{if .HasUnitCount}}<div><strong>{{.UnitCount}}</strong><span>{{.UnitCountLabel}} im Haus, direkt aus den hinterlegten Einheiten.</span></div>{{else}}<div><strong>Eingeladen</strong><span>Zugang nur für freigegebene E-Mail-Adressen der Hausgemeinschaft.</span></div>{{end}}
          <div><strong>Einmalig</strong><span>Anmeldung per SSO oder zeitlich begrenztem E-Mail-Link.</span></div>
          <div><strong>Parkplatz</strong><span>Verbrauch und Abrechnung bleiben im geschützten Portal.</span></div>
        </div>
      </div>
      <section id="login" class="login" aria-label="Anmeldung">
        <h2>Anmelden</h2>
        <p>{{if .OIDCConfigured}}Melden Sie sich per SSO an oder verwenden Sie einen einmaligen E-Mail-Link.{{else}}Geben Sie Ihre E-Mail-Adresse ein. Wenn sie eingeladen ist, schicken wir einen einmaligen Anmeldelink.{{end}}</p>
        {{if .OIDCConfigured}}<a class="sso-button" href="/auth/oidc/start">Mit {{.OIDCProviderName}} anmelden</a>{{end}}
        {{if and .OIDCConfigured .EmailLoginAvailable}}<div class="divider"><span>oder</span></div>{{end}}
        {{if .Sent}}
          <div class="notice">Wenn die Adresse eingeladen ist, wurde ein Link verschickt. Bitte Posteingang prüfen.</div>
          {{if not .MailConfigured}}<div class="notice warn">Mailversand ist lokal noch nicht konfiguriert. In Produktion kommt SMTP aus agenix.</div>{{end}}
          {{if .DevLoginLink}}<a class="dev-link" href="{{.DevLoginLink}}">Lokalen Dev-Login öffnen</a>{{end}}
        {{end}}
        {{if .Denied}}<div class="notice warn">Diese Adresse ist noch nicht eingeladen.</div>{{end}}
        {{if .EmailLoginAvailable}}
          <form method="post" action="/auth/request">
            <label for="email">E-Mail-Adresse</label>
            <input id="email" name="email" type="email" inputmode="email" autocomplete="email" required placeholder="name@example.com">
            <button type="submit">Anmeldelink senden</button>
          </form>
          <p class="foot-note">Link 15 Minuten gültig · privat für die Hausgemeinschaft</p>
        {{else}}
          <div class="notice">E-Mail-Anmeldelinks sind nicht aktiv. Bitte SSO verwenden.</div>
        {{end}}
      </section>
    </main>
    <footer>{{.Tenant.Address}} · Privat für die Hausgemeinschaft <span class="version">{{.AppVersion}}</span></footer>
  </section>
</body>
</html>
{{end}}

{{define "appStyles"}}
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=Spectral:wght@400;500;600;700&display=swap" rel="stylesheet">
  <style>
    :root {
      color-scheme: light;
{{template "designTokens" .}}
    }
    * { box-sizing: border-box; }
    body { margin: 0; background: var(--paper); color: var(--ink); }
    a { color: inherit; }
    button, input { font: inherit; }
    /* Accessibility convention: all keyboard-reachable controls keep a visible focus ring. */
    :where(a, button, input, select, textarea, summary, [tabindex]):focus-visible { outline: 3px solid var(--gold); outline-offset: 3px; }
    .app-shell { min-height: 100vh; display: grid; grid-template-columns: 264px minmax(0,1fr); background: var(--paper); }
    .sidebar { position: sticky; top: 0; height: 100vh; display: flex; flex-direction: column; gap: 24px; padding: 22px 16px 18px; color: rgba(255,255,255,.86); background: radial-gradient(circle at 20% 0%, rgba(255,255,255,.08), transparent 28%), var(--nav); border-right: 1px solid rgba(255,255,255,.08); }
    .side-brand { display: grid; grid-template-columns: 50px 1fr; gap: 14px; align-items: center; padding: 0 8px 12px; }
    .side-mark { width: 48px; height: 48px; border-radius: var(--radius-sm); display: grid; place-items: center; color: #fff; font-weight: 800; font-size: 13px; border: 1px solid rgba(255,255,255,.43); background: rgba(255,255,255,.08); }
    .side-title { display: block; font-family: var(--font-serif); font-size: 18px; font-weight: 600; line-height: 1.1; color: #fff; text-decoration: none; }
    .side-sub { display: block; margin-top: 5px; font-size: 14px; color: rgba(255,255,255,.72); }
    .side-nav { display: grid; gap: 7px; }
    .nav-item { position: relative; min-height: 46px; display: flex; align-items: center; gap: 12px; padding: 10px 12px; border-radius: var(--radius-xs); color: rgba(255,255,255,.78); text-decoration: none; font-size: 15px; font-weight: 600; }
    .nav-item:hover { color: #fff; background: rgba(255,255,255,.06); }
    .nav-item.active { color: #fff; background: rgba(255,255,255,.08); }
    .nav-item.active::before { content: ""; position: absolute; left: -16px; top: 0; bottom: 0; width: 4px; background: var(--gold); }
    .nav-item.disabled { color: rgba(255,255,255,.38); cursor: default; }
    .nav-item.disabled:hover { background: transparent; }
    .nav-icon { width: 23px; height: 23px; display: grid; place-items: center; flex: 0 0 auto; color: currentColor; }
    .nav-icon svg { width: 22px; height: 22px; stroke: currentColor; stroke-width: 1.9; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .nav-label { min-width: 0; }
    .nav-badge { margin-left: auto; min-width: 25px; height: 22px; display: inline-flex; align-items: center; justify-content: center; border-radius: var(--radius-pill); padding: 0 7px; background: var(--gold); color: #172019; font-size: 11px; font-weight: 900; line-height: 1; }
    .side-foot { margin-top: auto; border-top: 1px solid rgba(255,255,255,.16); padding: 18px 8px 0; display: grid; gap: 14px; }
    .side-user { display: grid; grid-template-columns: 42px 1fr; gap: 12px; align-items: center; }
    .avatar { width: 42px; height: 42px; border-radius: 50%; display: grid; place-items: center; background: var(--gold); color: #fff; font-weight: 800; border: 1px solid rgba(255,255,255,.25); }
    .side-user strong { display: block; color: #fff; font-size: 14px; }
    .side-user span, .side-version { color: rgba(255,255,255,.64); font-size: 13px; }
    .logout-form { margin: 0; }
    .logout-button { width: 100%; min-height: 42px; display: inline-flex; align-items: center; justify-content: center; gap: 10px; border: 1px solid rgba(255,255,255,.24); border-radius: var(--radius-xs); color: rgba(255,255,255,.92); background: transparent; font-weight: 700; cursor: pointer; }
    .logout-button:hover { border-color: var(--gold); color: #fff; }
    .app-main { min-width: 0; padding-bottom: 58px; }
    .content-top { height: 64px; display: flex; align-items: center; justify-content: space-between; gap: 14px; padding: 0 clamp(28px,4vw,44px); border-bottom: 1px solid var(--line); background: rgba(255,254,251,.72); }
    .crumb { display: inline-flex; align-items: center; gap: 10px; color: var(--muted); font-size: 14px; }
    .crumb svg, .action svg { width: 18px; height: 18px; stroke: currentColor; fill: none; stroke-width: 1.9; stroke-linecap: round; stroke-linejoin: round; }
    .page-actions { display: flex; align-items: center; gap: 10px; }
    .page { width: min(1220px,100%); margin: 0 auto; padding: 34px clamp(28px,4vw,44px) 0; display: grid; gap: 24px; }
    .page.wide { width: min(1280px,100%); }
    h1 { margin: 0; font-family: var(--font-serif); font-weight: 500; font-size: clamp(42px,5vw,54px); line-height: 1; }
    h2 { margin: 0; font-family: var(--font-serif); font-weight: 600; font-size: 23px; line-height: 1.1; }
    h3 { margin: 0; font-family: var(--font-serif); font-weight: 600; font-size: 20px; line-height: 1.2; }
    p { margin: 0; }
    .lede { margin-top: 14px; color: var(--muted); font-size: 16px; line-height: 1.55; }
    .muted { color: var(--muted); line-height: 1.5; }
    .subtle-note { margin-top: 6px; max-width: 760px; font-size: 14.5px; }
    .kicker { color: var(--gold-ink); font-size: 12px; font-weight: 800; letter-spacing: .12em; text-transform: uppercase; border-bottom: 2px solid var(--ink); padding-bottom: 11px; margin-bottom: 20px; }
    /* Shared components: panel, button, pill, quick-row, table-wrap, dialog, flash and empty-state. */
    .panel { background: var(--panel); border: 1px solid var(--line); border-radius: var(--radius-sm); padding: var(--space-6); box-shadow: var(--shadow-panel); }
    .panel.compact { padding: 18px; }
    .button, button.action { min-height: 38px; display: inline-flex; align-items: center; justify-content: center; gap: var(--space-2); border: 1px solid var(--line); background: var(--panel); border-radius: var(--radius-xs); color: var(--ink); padding: 8px 13px; font-weight: 700; line-height: 1.15; text-decoration: none; cursor: pointer; white-space: nowrap; }
    .button:hover, button.action:hover { border-color: var(--gold); }
    .button.primary, button.primary { background: var(--ink); border-color: var(--ink); color: #fff; }
    .button.small, button.small { min-height: 31px; padding: 6px 10px; font-size: 12px; }
    .button.ghost { background: transparent; }
    .banner { position: relative; height: 128px; overflow: hidden; border-bottom: 1px solid var(--line); background: #e9e4d7; }
    .banner::before { content: ""; position: absolute; inset: 0; background: url('{{.Tenant.HeroImageURL}}') center 47% / cover no-repeat; }
    .banner::after { content: ""; position: absolute; inset: 0; background: linear-gradient(90deg, rgba(23,32,25,.1), rgba(247,243,234,.72) 76%, rgba(247,243,234,.92)); }
    .banner-kicker { position: absolute; left: clamp(28px,4vw,44px); bottom: 18px; color: var(--gold-ink); font-size: 12px; font-weight: 800; letter-spacing: .18em; text-transform: uppercase; }
    .home-grid { display: grid; grid-template-columns: minmax(0,1.35fr) minmax(340px,.85fr); gap: 22px; align-items: start; }
    .entries { display: grid; gap: 22px; }
    .entry + .entry { border-top: 1px solid var(--line); padding-top: 22px; }
    .entry-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; }
    .entry h3 a { color: inherit; text-decoration: none; }
    .entry h3 a:hover { color: var(--gold-ink); }
    .entry p, .entry-body { margin-top: 8px; color: #5c5f54; line-height: 1.6; }
    .entry-meta { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin-top: 10px; color: var(--soft); font-size: 12.5px; }
    .entry-actions { display: flex; gap: 8px; flex-wrap: wrap; justify-content: flex-end; }
    .archive-tools { display: grid; gap: 12px; margin: -4px 0 20px; }
    .filter-form { display: grid; grid-template-columns: minmax(220px,1fr) auto; gap: 10px; align-items: end; }
    .filter-form.audit-filter { grid-template-columns: minmax(180px,.55fr) minmax(240px,1fr) auto; margin-bottom: 16px; }
    .filter-form label { margin: 0; }
    .filter-tabs { display: flex; gap: 8px; flex-wrap: wrap; }
    .filter-tab { min-height: 34px; display: inline-flex; align-items: center; justify-content: center; border: 1px solid var(--line); border-radius: var(--radius-pill); padding: 6px 12px; color: var(--muted); background: var(--panel); text-decoration: none; font-size: 13px; font-weight: 800; }
    .filter-tab.active, .filter-tab:hover { border-color: var(--gold); color: var(--ink); background: rgba(200,153,63,.14); }
    .quick-list { display: grid; }
    .quick-row { display: grid; grid-template-columns: 30px minmax(0,1fr) auto; gap: var(--space-3); align-items: center; padding: 13px 0; border-bottom: 1px solid var(--line); color: inherit; text-decoration: none; }
    button.quick-row { width: 100%; border: 0; border-bottom: 1px solid var(--line); background: transparent; font: inherit; text-align: left; cursor: pointer; }
    button.quick-row:hover { color: var(--gold-ink); }
    .quick-row:last-child { border-bottom: 0; }
    .quick-row > div { min-width: 0; }
    .quick-row svg { width: 24px; height: 24px; stroke: currentColor; stroke-width: 1.8; fill: none; stroke-linecap: round; stroke-linejoin: round; color: var(--ink); }
    .quick-row h3 { font-size: 18px; }
    .quick-row p { margin-top: 3px; color: var(--soft); font-size: 13.5px; line-height: 1.35; overflow-wrap: anywhere; }
    .quick-row.disabled { cursor: default; }
    .quick-row.disabled h3, .quick-row.disabled svg { color: var(--muted); }
    .quick-row .pill { justify-self: end; }
    .quick-arrow { color: var(--gold-ink); font-size: 24px; line-height: 1; }
    .digest-panel { grid-column: 1 / -1; }
    .digest-panel .quick-list { grid-template-columns: repeat(3,minmax(0,1fr)); gap: 10px; }
    .digest-panel .quick-row { border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 13px; background: var(--panel-soft); }
    .digest-panel .quick-row:last-child { border-bottom: 1px solid var(--line); }
    .agenda-list { display: grid; gap: 10px; margin-bottom: 22px; }
    .event-card { display: grid; grid-template-columns: 58px minmax(0,1fr); gap: 13px; align-items: start; border: 1px solid var(--line); border-radius: var(--radius-sm); padding: var(--space-3); color: inherit; background: var(--panel-soft); text-decoration: none; }
    .event-card:hover { border-color: var(--gold); }
    .event-card.past { opacity: .68; }
    .date-badge { min-height: 58px; display: grid; place-items: center; align-content: center; gap: 2px; border-radius: var(--radius-sm); background: var(--ink); color: #fff; font-weight: 800; text-align: center; }
    .date-badge strong { font-family: var(--font-serif); font-size: 24px; line-height: .95; }
    .date-badge span { font-size: 11px; text-transform: uppercase; letter-spacing: .08em; }
    .event-info { min-width: 0; display: grid; gap: 7px; }
    .event-info h3 { font-size: 18px; overflow-wrap: anywhere; }
    .event-info p { color: var(--muted); line-height: 1.45; font-size: 13.5px; }
    .event-meta { display: flex; flex-wrap: wrap; align-items: center; gap: 7px; color: var(--soft); font-size: 12.5px; font-weight: 700; }
    .status-strip { display: grid; gap: 14px; }
    .rule { display: flex; align-items: center; justify-content: space-between; gap: 14px; flex-wrap: wrap; color: var(--ink); line-height: 1.5; }
    .rule p { flex: 1 1 640px; min-width: 0; }
    .rule .pill { margin-left: auto; }
    .pill { display: inline-flex; align-items: center; min-height: 26px; border-radius: var(--radius-pill); padding: 3px 10px; font-size: 12px; font-weight: 800; background: rgba(200,153,63,.16); color: #8a6a1f; white-space: nowrap; }
    .pill.ok { background: rgba(47,107,74,.12); color: var(--leaf); }
    .pill.ok::before { content: ""; width: 8px; height: 8px; border-radius: 50%; background: currentColor; margin-right: 8px; }
    .pill.dringend { background: rgba(158,42,43,.12); color: #9e2a2b; }
    .pill.termin { background: rgba(47,107,74,.12); color: var(--leaf); }
    .pill.wartung { background: rgba(173,92,27,.14); color: #8a551f; }
    .pill.info { background: rgba(200,153,63,.16); color: #8a6a1f; }
    .pill.versammlung { background: rgba(32,37,31,.08); color: var(--ink); }
    .pill.reinigung { background: rgba(76,103,138,.11); color: #365475; }
    .pill.ablesung { background: rgba(47,107,74,.12); color: var(--leaf); }
    .pill.frist { background: rgba(158,42,43,.12); color: #9e2a2b; }
    .pill.sonstiges { background: rgba(200,153,63,.16); color: #8a6a1f; }
    .pill.unread { background: var(--gold); color: #172019; }
    .pill.status-open { background: rgba(200,153,63,.16); color: #8a6a1f; }
    .pill.status-progress { background: rgba(32,37,31,.08); color: var(--ink); }
    .pill.status-done { background: rgba(47,107,74,.12); color: var(--leaf); }
    .pill.status-closed { background: rgba(158,42,43,.1); color: #9e2a2b; }
    .chips { display: flex; flex-wrap: wrap; gap: 6px; }
    .chip { display: inline-flex; align-items: center; gap: 5px; border: 1px solid var(--line); background: var(--panel-soft); color: #6f6a5c; border-radius: var(--radius-sm); padding: 4px 10px; font-size: 12.5px; font-weight: 600; white-space: nowrap; }
    .chip strong { color: var(--gold-ink); }
    .issue-layout { display: grid; grid-template-columns: minmax(0,1.45fr) minmax(280px,.8fr); gap: 22px; align-items: start; }
    .issue-form { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; }
    .issue-form label { display: grid; gap: 7px; color: var(--gold-ink); font-size: 11px; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; }
    .issue-form .full { grid-column: 1 / -1; }
    .issue-form input, .issue-form select, .issue-form textarea { width: 100%; border: 1px solid #e2dac9; border-radius: var(--radius-sm); padding: 12px; font: inherit; background: #fffefb; color: var(--ink); }
    .issue-form textarea { min-height: 148px; resize: vertical; line-height: 1.45; }
    .issue-form input[type=file] { padding: 10px; color: var(--muted); }
    .file-control { position: relative; min-height: 44px; display: flex; align-items: center; gap: 10px; border: 1px solid #e2dac9; border-radius: var(--radius-sm); padding: 10px 12px; background: #fffefb; color: var(--ink); overflow: hidden; }
    .file-control input[type=file] { position: absolute; inset: 0; opacity: 0; cursor: pointer; }
    .file-control span { pointer-events: none; font-size: 13px; font-weight: 800; letter-spacing: 0; text-transform: none; }
    .issue-form .hint { margin-top: 2px; color: var(--soft); font-size: 12px; font-weight: 600; letter-spacing: 0; text-transform: none; }
    .issue-form button { grid-column: 1 / -1; min-height: 46px; border: 1px solid var(--ink); border-radius: var(--radius-sm); background: var(--ink); color: #fff; font: inherit; font-weight: 800; cursor: pointer; }
    .issue-form button:hover { background: #2c3329; }
    .issue-flash { margin: 0 0 14px; padding: 11px 13px; border-radius: var(--radius-sm); font-size: 13.5px; font-weight: 700; border: 1px solid transparent; }
    .issue-flash.ok { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.25); }
    .issue-flash.warn { background: rgba(200,153,63,.14); color: #93701d; border-color: rgba(200,153,63,.3); }
    .issue-list { display: grid; gap: 10px; }
    .issue-card { border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 13px; background: #fffefb; display: grid; gap: 10px; }
    .issue-card h3 { font-size: 18px; }
    .issue-meta { display: flex; flex-wrap: wrap; gap: 7px; align-items: center; color: var(--soft); font-size: 12.5px; font-weight: 700; }
    .issue-location { color: var(--muted); font-size: 13px; line-height: 1.35; }
    .issue-actions { display: flex; flex-wrap: wrap; gap: 8px; align-items: end; }
    .issue-actions form { margin: 0; }
    .issue-actions label { display: grid; gap: 5px; min-width: 150px; color: var(--gold-ink); font-size: 10.5px; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; }
    .issue-actions label.assignee { flex: 1 1 240px; }
    .issue-actions input, .issue-actions select { width: 100%; border: 1px solid #e2dac9; border-radius: var(--radius-xs); min-height: 38px; padding: 8px 10px; color: var(--ink); background: #fffefb; font: inherit; font-size: 13px; }
    .issue-actions button { min-height: 38px; border: 1px solid var(--ink); border-radius: var(--radius-xs); padding: 8px 12px; background: var(--ink); color: #fff; font: inherit; font-size: 13px; font-weight: 800; cursor: pointer; }
    .issue-actions .ghost { background: transparent; color: var(--ink); border-color: var(--line); }
    .issue-board-filter { display: grid; grid-template-columns: repeat(12, minmax(0,1fr)); gap: 10px; margin-bottom: 14px; align-items: end; }
    .issue-board-filter label { display: grid; gap: 5px; color: var(--gold-ink); font-size: 10.5px; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; grid-column: span 2; }
    .issue-board-filter label.assignee { grid-column: span 3; }
    .issue-board-filter label.sort { grid-column: span 3; }
    .issue-board-filter input, .issue-board-filter select { width: 100%; border: 1px solid #e2dac9; border-radius: var(--radius-xs); min-height: 38px; padding: 8px 10px; color: var(--ink); background: #fffefb; font: inherit; font-size: 13px; }
    .issue-board-filter .board-filter-actions { grid-column: span 2; display: flex; gap: 8px; align-items: center; }
    .issue-board-filter button, .issue-board-filter a { min-height: 38px; border: 1px solid var(--ink); border-radius: var(--radius-xs); padding: 8px 12px; background: var(--ink); color: #fff; font: inherit; font-size: 13px; font-weight: 800; cursor: pointer; text-decoration: none; display: inline-flex; align-items: center; }
    .issue-board-filter a { background: transparent; color: var(--ink); border-color: var(--line); }
    .comment-thread { display: grid; gap: 8px; border-top: 1px dashed var(--line); padding-top: 10px; }
    .comment { display: grid; gap: 3px; border-left: 3px solid rgba(200,153,63,.35); padding-left: 9px; color: var(--ink); }
    .comment-meta { color: var(--soft); font-size: 12px; font-weight: 800; }
    .comment-form { display: grid; gap: 8px; }
    .comment-form textarea { min-height: 84px; }
    .comment-form button { justify-self: start; min-height: 38px; border: 1px solid var(--ink); border-radius: var(--radius-xs); padding: 8px 12px; background: var(--ink); color: #fff; font: inherit; font-size: 13px; font-weight: 800; cursor: pointer; }
    .dialog { border: 1px solid var(--line); border-radius: var(--radius-md); padding: 0; width: min(680px, calc(100vw - 28px)); color: var(--ink); background: var(--panel); box-shadow: var(--shadow-dialog); }
    .dialog::backdrop { background: rgba(23,32,25,.42); }
    .dialog form { margin: 0; }
    .dialog-head { display: flex; justify-content: space-between; align-items: center; gap: 14px; padding: 20px 22px; border-bottom: 1px solid var(--line); }
    .dialog-head h2 { font-size: 25px; }
    .dialog-close { width: 34px; height: 34px; border: 1px solid var(--line); border-radius: var(--radius-xs); background: var(--panel-soft); color: var(--ink); font-size: 22px; line-height: 1; cursor: pointer; }
    .dialog-body { display: grid; gap: 14px; padding: 20px 22px 22px; }
    .dialog-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; }
    .dialog-grid .full { grid-column: 1 / -1; }
    textarea { width: 100%; border: 1px solid #e2dac9; border-radius: var(--radius-xs); min-height: 150px; padding: 10px 12px; color: var(--ink); background: #fffefb; resize: vertical; font: inherit; line-height: 1.45; }
    select { width: 100%; border: 1px solid #e2dac9; border-radius: var(--radius-xs); min-height: 42px; padding: 9px 12px; color: var(--ink); background: #fffefb; font: inherit; }
    .check-row { display: inline-flex; align-items: center; gap: 8px; min-height: 42px; color: var(--ink); font-size: 14px; font-weight: 700; letter-spacing: 0; text-transform: none; }
    .check-row input { width: auto; min-height: 0; }
    .metric-grid { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 16px; }
    .metric-card, .month-card { min-width: 0; background: var(--panel-soft); border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 16px; }
    .metric-label, .field-label, th { color: var(--gold-ink); font-size: 11px; font-weight: 800; text-transform: uppercase; letter-spacing: .06em; }
    .metric-value { display: block; margin-top: 7px; font-family: var(--font-serif); font-size: 26px; font-weight: 600; line-height: 1.08; }
    code, .mini { color: var(--soft); font-size: 12px; line-height: 1.35; overflow-wrap: anywhere; }
    .accounting { display: grid; gap: 16px; }
    .section-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; flex-wrap: wrap; }
    .month-strip { display: grid; grid-template-columns: repeat(7,minmax(120px,1fr)); gap: 10px; }
    .month-card { display: grid; gap: 10px; color: inherit; text-decoration: none; }
    .month-card:hover { border-color: var(--gold); }
    .month-card strong { font-family: var(--font-serif); font-size: 16px; }
    .bar { height: 9px; border-radius: var(--radius-pill); background: #ece5d6; overflow: hidden; }
    .bar span { display: block; height: 100%; min-width: 2px; border-radius: inherit; background: var(--gold); }
    .amount { font-weight: 800; font-variant-numeric: tabular-nums; }
    .table-wrap { overflow-x: auto; border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel); }
    table { width: 100%; border-collapse: collapse; min-width: 980px; }
    th, td { padding: 13px 16px; border-bottom: 1px solid var(--line); vertical-align: middle; }
    th { text-align: left; background: rgba(251,248,240,.7); }
    td { font-size: 14px; }
    tbody tr:last-child td { border-bottom: 0; }
    .num { text-align: right; font-variant-numeric: tabular-nums; white-space: nowrap; }
    .month-cell strong { display: block; font-family: var(--font-serif); font-size: 17px; }
    .month-cell a { text-decoration: none; }
    .month-cell a:hover { color: var(--gold-ink); }
    .month-cell span { display: block; margin-top: 3px; color: var(--soft); font-size: 12px; }
    .row-actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
    .document-sections { display: grid; gap: 14px; }
    .filter-form.document-filter { grid-template-columns: minmax(280px,1fr) minmax(150px,.28fr) auto; align-items: end; margin-bottom: 12px; }
    .filter-form.document-filter button { width: auto; margin-top: 0; min-height: 42px; }
    .document-section { border-top: 1px solid var(--line); padding-top: 14px; display: grid; gap: 10px; }
    .document-section:first-child { border-top: 0; padding-top: 0; }
    .document-section h3 { font-size: 18px; }
    .document-list { display: grid; gap: 8px; }
    .document-row { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 14px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-xs); background: #fffefb; padding: 12px 14px; }
    .document-row strong { display: block; font-size: 15px; }
    .document-meta { margin-top: 5px; display: flex; flex-wrap: wrap; gap: 7px; align-items: center; color: var(--muted); font-size: 12.5px; }
    .document-file { color: var(--soft); overflow-wrap: anywhere; }
    .document-side { display: flex; align-items: center; justify-content: flex-end; gap: 8px; flex-wrap: wrap; }
    .document-versions { grid-column: 1 / -1; border-top: 1px solid var(--line); padding-top: 10px; }
    .document-versions summary { cursor: pointer; font-weight: 800; color: var(--gold-ink); }
    .version-list { margin-top: 8px; display: grid; gap: 7px; }
    .version-row { display: flex; align-items: center; justify-content: space-between; gap: 10px; flex-wrap: wrap; color: var(--muted); font-size: 12.5px; }
    .vote-list { display: grid; gap: 12px; }
    .vote-card { border: 1px solid var(--line); border-radius: var(--radius-sm); background: #fffefb; padding: 16px; display: grid; gap: 13px; }
    .vote-card-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; }
    .vote-card-head > .pill { justify-self: end; }
    .vote-card h3 { font-size: 20px; overflow-wrap: anywhere; }
    .vote-meta { display: flex; flex-wrap: wrap; align-items: center; gap: 7px; color: var(--soft); font-size: 12.5px; font-weight: 700; }
    .vote-options { display: grid; gap: 8px; }
    .vote-option { min-height: 42px; display: grid; grid-template-columns: auto minmax(0,1fr); gap: 10px; align-items: center; border: 1px solid var(--line); border-radius: var(--radius-xs); background: var(--panel-soft); padding: 10px 12px; color: var(--ink); font-size: 14px; font-weight: 700; }
    .vote-option input { width: auto; min-height: 0; }
    .vote-actions { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
    .vote-result { display: grid; gap: 6px; }
    .vote-result-row { display: grid; grid-template-columns: minmax(110px,.45fr) minmax(120px,1fr) auto; gap: 10px; align-items: center; color: var(--muted); font-size: 13px; }
    .vote-result-row strong { color: var(--ink); overflow-wrap: anywhere; }
    .vote-bar { height: 9px; border-radius: var(--radius-pill); background: #ece5d6; overflow: hidden; }
    .vote-bar span { display: block; height: 100%; min-width: 2px; border-radius: inherit; background: var(--gold); }
    .vote-manage-list { display: grid; gap: 8px; }
    .vote-manage-row { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 12px; align-items: center; border-bottom: 1px solid var(--line); padding: 10px 0; }
    .vote-manage-row:last-child { border-bottom: 0; }
    .vote-manage-actions { display: flex; gap: 8px; flex-wrap: wrap; justify-content: flex-end; }
    .vote-manage-actions form { margin: 0; }
    .empty { border: 1px solid var(--line); background: var(--panel-soft); color: #5c5f54; border-radius: var(--radius-sm); padding: 14px; line-height: 1.5; }
    .empty-state { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); padding: 18px; display: grid; grid-template-columns: 42px minmax(0,1fr) auto; gap: 14px; align-items: center; color: var(--ink); }
    .empty-state-icon { width: 42px; height: 42px; border-radius: var(--radius-sm); display: grid; place-items: center; background: rgba(200,153,63,.16); color: var(--gold-ink); }
    .empty-state-icon svg { width: 22px; height: 22px; stroke: currentColor; stroke-width: 1.9; fill: none; stroke-linecap: round; stroke-linejoin: round; }
    .empty-state h3 { font-size: 19px; }
    .empty-state p { margin-top: 4px; color: var(--muted); line-height: 1.45; font-size: 14px; }
    .settings-card { max-width: 620px; display: grid; gap: 16px; }
    .form-grid { display: grid; gap: 12px; }
    label { display: grid; gap: 7px; color: var(--gold-ink); font-size: 12px; font-weight: 800; text-transform: uppercase; letter-spacing: .06em; }
    input { width: 100%; border: 1px solid #e2dac9; border-radius: var(--radius-xs); min-height: 42px; padding: 9px 12px; color: var(--ink); background: #fffefb; }
    .flash { padding: 10px 13px; border-radius: var(--radius-xs); font-size: 13.5px; font-weight: 700; border: 1px solid rgba(200,153,63,.28); background: rgba(200,153,63,.14); color: #8a6a1f; }
    .flash.ok { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.25); }
    .legend { border: 1px solid var(--line); border-radius: var(--radius-sm); background: var(--panel-soft); padding: 14px; display: grid; grid-template-columns: repeat(auto-fit,minmax(210px,1fr)); gap: 12px; }
    .legend strong { display: block; font-family: var(--font-serif); margin-bottom: 3px; }
    .legend span { display: block; color: var(--muted); font-size: 13px; line-height: 1.4; }
    .bar-cell { min-width: 150px; }
    @media (max-width: 1120px) {
      .app-shell { grid-template-columns: 1fr; }
      .sidebar { position: relative; height: auto; padding: 16px; }
      .side-nav { grid-template-columns: repeat(auto-fit,minmax(170px,1fr)); }
      .side-foot { margin-top: 4px; grid-template-columns: 1fr auto; align-items: center; }
      .logout-form { justify-self: end; min-width: 160px; }
      .home-grid, .metric-grid, .issue-layout { grid-template-columns: 1fr; }
      .digest-panel .quick-list { grid-template-columns: 1fr; }
      .month-strip { grid-template-columns: repeat(auto-fit,minmax(150px,1fr)); }
    }
    @media (max-width: 680px) {
      .side-brand, .side-user { grid-template-columns: auto 1fr; }
      .side-foot { grid-template-columns: 1fr; }
      .logout-form { justify-self: stretch; }
      .content-top { height: auto; min-height: 58px; flex-direction: column; align-items: flex-start; padding-top: 12px; padding-bottom: 12px; }
      .page { padding-left: 18px; padding-right: 18px; }
      h1 { font-size: clamp(36px,12vw,48px); }
	      .metric-grid { grid-template-columns: 1fr; }
	      .issue-form { grid-template-columns: 1fr; }
	      .issue-board-filter { grid-template-columns: 1fr; }
	      .issue-board-filter label, .issue-board-filter label.assignee, .issue-board-filter label.sort, .issue-board-filter .board-filter-actions { grid-column: 1 / -1; }
	      .quick-row { grid-template-columns: 28px minmax(0,1fr); }
	      .document-row { grid-template-columns: 1fr; }
	      .document-side { justify-content: flex-start; }
	      .vote-card-head, .vote-actions, .vote-manage-row { display: grid; grid-template-columns: 1fr; }
	      .vote-card-head > .pill { justify-self: start; }
	      .vote-result-row { grid-template-columns: 1fr; }
	      .vote-manage-actions { justify-content: flex-start; }
      .quick-row .pill { grid-column: 2; justify-self: start; }
	      .filter-form { grid-template-columns: 1fr; }
	      .filter-form.document-filter { grid-template-columns: 1fr; }
      .dialog-grid { grid-template-columns: 1fr; }
      .empty-state { grid-template-columns: 1fr; }
      .quick-arrow { display: none; }
    }
  </style>
{{end}}

{{define "sidebar"}}
  <aside class="sidebar" aria-label="Portalnavigation">
    <div class="side-brand">
      <a class="side-mark" href="/app">WEG</a>
      <div>
        <a class="side-title" href="/app">{{.Tenant.Name}}</a>
        <span class="side-sub">{{.Tenant.Address}}</span>
      </div>
    </div>
    <nav class="side-nav">
      <a class="nav-item {{if eq .ActivePage "home"}}active{{end}}" href="/app"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg></span><span class="nav-label">Hausüberblick</span></a>
      <a class="nav-item {{if eq .ActivePage "announcements"}}active{{end}}" href="/app/announcements"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M4 5h16v13H7l-3 3z"/><path d="M8 9h8M8 13h6"/></svg></span><span class="nav-label">Aushang</span>{{if .HasUnreadAnnouncements}}<span class="nav-badge">{{.UnreadAnnouncements}}</span>{{end}}</a>
      <a class="nav-item {{if eq .ActivePage "events"}}active{{end}}" href="/app/events"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M7 3v4M17 3v4"/><path d="M4.5 6h15v14h-15z"/><path d="M4.5 10h15"/><path d="M8 14h.01M12 14h.01M16 14h.01"/></svg></span><span class="nav-label">Termine</span></a>
      <a class="nav-item {{if eq .ActivePage "contacts"}}active{{end}}" href="/app/kontakte"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M16 4h2.5A1.5 1.5 0 0 1 20 5.5v13A1.5 1.5 0 0 1 18.5 20h-13A1.5 1.5 0 0 1 4 18.5v-13A1.5 1.5 0 0 1 5.5 4H8"/><path d="M8.5 3.5h7v4h-7z"/><path d="M9 13a3 3 0 1 0 6 0"/><path d="M7.5 18a4.5 4.5 0 0 1 9 0"/></svg></span><span class="nav-label">Kontakte</span></a>
      {{if .CanSeeParking}}<a class="nav-item {{if eq .ActivePage "parking"}}active{{end}}" href="/app/parking"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 16h14"/><path d="m7 16 1.5-5h7L17 16"/><path d="M7 16v3M17 16v3"/><path d="M7 19h1M16 19h1"/></svg></span>Parkplatznutzung</a>{{end}}
      <a class="nav-item {{if eq .ActivePage "documents"}}active{{end}}" href="/app/dokumente"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/><path d="M9 13h6M9 17h6"/></svg></span>Dokumente</a>
      <a class="nav-item {{if eq .ActivePage "issues"}}active{{end}}" href="/app/anliegen"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 18.5V6.5A2.5 2.5 0 0 1 7.5 4h9A2.5 2.5 0 0 1 19 6.5v6A2.5 2.5 0 0 1 16.5 15H10l-5 3.5z"/></svg></span>Anliegen{{if .HasOpenIssues}}<span class="nav-badge">{{.OpenIssues}}</span>{{end}}</a>
      <a class="nav-item {{if eq .ActivePage "abstimmungen"}}active{{end}}" href="/app/abstimmungen"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 19V9M12 19V5M19 19v-7"/><path d="M3.5 19h17"/></svg></span>Abstimmungen</a>
      {{if .CanManageUsers}}<a class="nav-item {{if eq .ActivePage "users"}}active{{end}}" href="/app/settings/users"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"/><path d="M3.5 20a5 5 0 0 1 10 0"/><path d="M16 11.5a2.5 2.5 0 1 0 0-5"/><path d="M17 15a4 4 0 0 1 3.5 4"/></svg></span>Benutzer &amp; Rechte</a>{{end}}
      {{if .CanViewAudit}}<a class="nav-item {{if eq .ActivePage "audit"}}active{{end}}" href="/app/audit"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 4h14v16H5z"/><path d="M8 8h8M8 12h8M8 16h5"/></svg></span>Audit-Log</a>{{end}}
      <a class="nav-item {{if eq .ActivePage "settings"}}active{{end}}" href="/app/settings"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M12 8.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7z"/><path d="M19 12a7 7 0 0 0-.1-1l2-1.5-2-3.5-2.4 1a7 7 0 0 0-1.8-1L14.4 3h-4.8L9.3 6a7 7 0 0 0-1.8 1l-2.4-1-2 3.5 2 1.5A7 7 0 0 0 5 12a7 7 0 0 0 .1 1l-2 1.5 2 3.5 2.4-1a7 7 0 0 0 1.8 1l.3 3h4.8l.3-3a7 7 0 0 0 1.8-1l2.4 1 2-3.5-2-1.5a7 7 0 0 0 .1-1z"/></svg></span>Einstellungen</a>
    </nav>
    <div class="side-foot">
      <div class="side-user">
        <span class="avatar">{{.Initials}}</span>
        <div><strong>{{.DisplayName}}</strong><span>{{.Role}}</span></div>
      </div>
      <span class="side-version">{{.AppVersion}}</span>
      <form class="logout-form" method="post" action="/auth/logout"><button class="logout-button" type="submit">Abmelden</button></form>
    </div>
  </aside>
{{end}}

{{define "appOpen"}}
<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  {{template "appStyles" .}}
</head>
<body>
  <div class="app-shell">
    {{template "sidebar" .}}
{{end}}

{{define "appClose"}}
  </div>
</body>
</html>
{{end}}

{{define "emptyState"}}
  <div class="empty-state">
    <span class="empty-state-icon"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 5h14v14H5z"/><path d="M8 9h8M8 13h5"/></svg></span>
    <div><h3>{{.Title}}</h3><p>{{.Message}}</p></div>
    {{if .HasAction}}<a class="button" href="{{.ActionURL}}">{{.ActionLabel}}</a>{{end}}
  </div>
{{end}}

{{define "portal"}}
{{template "appOpen" .}}
    <main class="app-main">
      <div class="content-top"><span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg>Hausüberblick</span></div>
      <div class="banner"><span class="banner-kicker">WEG Portal · {{.Tenant.Address}}</span></div>
      <section class="page">
        <div>
          <h1>Hausüberblick</h1>
          <p class="lede">Aktuelle Informationen der Hausgemeinschaft und direkte Wege zu den freigeschalteten Bereichen.</p>
        </div>
        <div class="home-grid">
          <section class="panel digest-panel">
            <div class="section-head">
              <div class="kicker">Was ist neu</div>
            </div>
            {{if .HasDigest}}
              <div class="quick-list">
                {{range .Digest}}
                  <a class="quick-row" href="{{.URL}}"><svg viewBox="0 0 24 24"><path d="M5 12h14"/><path d="m13 6 6 6-6 6"/></svg><div><h3>{{.Title}}</h3><p>{{.Detail}}</p></div><span class="pill unread">{{.Badge}}</span></a>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .DigestEmpty}}
            {{end}}
          </section>
          <section class="panel">
            <div class="section-head">
              <div class="kicker">Aktueller Aushang</div>
              {{if .HasUnreadAnnouncements}}<span class="pill unread">{{.UnreadAnnouncements}} neu</span>{{end}}
              <a class="button small" href="/app/announcements">Archiv öffnen</a>
            </div>
            {{if .HasAnnouncements}}
              <div class="entries">
                {{range .Announcements}}
                  <article class="entry">
                    <div class="entry-head">
                      <div>
                        <h3><a href="/app/announcements">{{.Title}}</a></h3>
                        <div class="entry-meta">
                          <span class="pill {{.CategoryClass}}">{{.Category}}</span>
                          {{if .Unread}}<span class="pill unread">neu</span>{{end}}
                          {{if .Pinned}}<span class="pill">Fixiert</span>{{end}}
                          <span>{{.PublishedAt}}</span>
                        </div>
                      </div>
                    </div>
                    <div class="entry-body">{{.BodyHTML}}</div>
                  </article>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .AnnouncementsEmpty}}
            {{end}}
          </section>
          <section class="panel">
            <div class="kicker">Nächste Termine</div>
            {{if .HasEvents}}
              <div class="agenda-list">
                {{range .Events}}
                  <a class="event-card" href="/app/events">
                    <span class="date-badge"><strong>{{.DateBadgeDay}}</strong><span>{{.DateBadgeMonth}}</span></span>
                    <span class="event-info">
                      <h3>{{.Title}}</h3>
                      <span class="event-meta"><span class="pill {{.CategoryClass}}">{{.Category}}</span><span>{{.TimeRange}}</span>{{if .HasLocation}}<span>{{.Location}}</span>{{end}}</span>
                    </span>
                  </a>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .EventsEmpty}}
            {{end}}
            <div class="kicker">Schnellzugriff</div>
            <div class="quick-list">
              <a class="quick-row" href="/app/announcements"><svg viewBox="0 0 24 24"><path d="M4 5h16v13H7l-3 3z"/><path d="M8 9h8M8 13h6"/></svg><div><h3>Aushang</h3><p>Offizielle Informationen, Termine und Hinweise der Hausgemeinschaft.</p></div>{{if .HasUnreadAnnouncements}}<span class="pill unread">{{.UnreadAnnouncements}} neu</span>{{else}}<span class="quick-arrow">›</span>{{end}}</a>
              <a class="quick-row" href="/app/events"><svg viewBox="0 0 24 24"><path d="M7 3v4M17 3v4"/><path d="M4.5 6h15v14h-15z"/><path d="M4.5 10h15"/><path d="M8 14h.01M12 14h.01M16 14h.01"/></svg><div><h3>Termine</h3><p>Versammlungen, Wartungen, Fristen und gemeinsame Hausereignisse.</p></div><span class="quick-arrow">›</span></a>
              <a class="quick-row" href="/app/kontakte"><svg viewBox="0 0 24 24"><path d="M16 4h2.5A1.5 1.5 0 0 1 20 5.5v13A1.5 1.5 0 0 1 18.5 20h-13A1.5 1.5 0 0 1 4 18.5v-13A1.5 1.5 0 0 1 5.5 4H8"/><path d="M8.5 3.5h7v4h-7z"/><path d="M9 13a3 3 0 1 0 6 0"/><path d="M7.5 18a4.5 4.5 0 0 1 9 0"/></svg><div><h3>Kontakte</h3><p>Verwaltung, Notdienst, Beirat und freigegebene Kontakte.</p></div><span class="quick-arrow">›</span></a>
              <a class="quick-row" href="/app/dokumente"><svg viewBox="0 0 24 24"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/><path d="M9 13h6M9 17h6"/></svg><div><h3>Dokumente</h3><p>Protokolle, Abrechnungen und Unterlagen nach Berechtigung.</p></div><span class="quick-arrow">›</span></a>
              <a class="quick-row" href="/app/anliegen"><svg viewBox="0 0 24 24"><path d="M5 18.5V6.5A2.5 2.5 0 0 1 7.5 4h9A2.5 2.5 0 0 1 19 6.5v6A2.5 2.5 0 0 1 16.5 15H10l-5 3.5z"/></svg><div><h3>Anliegen</h3><p>Mängel, Fragen und Vorschläge direkt an die Verwaltung melden.</p></div>{{if .HasOpenIssues}}<span class="pill unread">{{.OpenIssues}} offen</span>{{else}}<span class="quick-arrow">›</span>{{end}}</a>
              {{if .CanSeeParking}}<a class="quick-row" href="/app/parking"><svg viewBox="0 0 24 24"><path d="M5 16h14"/><path d="m7 16 1.5-5h7L17 16"/><path d="M7 16v3M17 16v3"/><path d="M7 19h1M16 19h1"/></svg><div><h3>Parkplatznutzung</h3><p>Privater Bereich für die abgestimmte Nutzung des Stellplatzes.</p></div><span class="quick-arrow">›</span></a>{{end}}
              {{if .CanManageUsers}}<a class="quick-row" href="/app/settings/users"><svg viewBox="0 0 24 24"><path d="M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"/><path d="M3.5 20a5 5 0 0 1 10 0"/><path d="M16 11.5a2.5 2.5 0 1 0 0-5"/><path d="M17 15a4 4 0 0 1 3.5 4"/></svg><div><h3>Benutzer &amp; Rechte</h3><p>Einladungen, Rollen und Zugriff der Hausgemeinschaft verwalten.</p></div><span class="quick-arrow">›</span></a>{{end}}
              {{if .CanManageAnnouncements}}<a class="quick-row" href="/app/announcements"><svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg><div><h3>Aushang verwalten</h3><p>Beiträge verfassen, fixieren, planen und löschen.</p></div><span class="quick-arrow">›</span></a>{{end}}
            </div>
          </section>
        </div>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "contacts"}}
{{template "appOpen" .}}
    <style>
      .contacts .contact-grid { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 22px; align-items: start; }
      .contacts .contact-section { display: grid; gap: 14px; }
      .contacts .contact-list { display: grid; gap: 10px; }
      .contacts .contact-card { border: 1px solid var(--line); border-radius: 8px; padding: 14px; background: var(--panel-soft); display: grid; gap: 10px; }
      .contacts .contact-head { display: flex; justify-content: space-between; gap: 12px; align-items: flex-start; }
      .contacts .contact-head strong { font-family: Spectral, serif; font-size: 20px; line-height: 1.12; overflow-wrap: anywhere; }
      .contacts .contact-lines { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; color: var(--muted); font-size: 13.5px; font-weight: 700; }
      .contacts .contact-lines a { color: inherit; text-decoration: none; border-bottom: 1px solid rgba(200,153,63,.5); }
      .contacts .contact-lines a:hover { color: var(--gold-ink); }
      .contacts .directory-panel { grid-column: 1 / -1; }
      @media (max-width: 900px) { .contacts .contact-grid { grid-template-columns: 1fr; } .contacts .directory-panel { grid-column: 1; } }
    </style>
    <main class="app-main contacts">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M16 4h2.5A1.5 1.5 0 0 1 20 5.5v13A1.5 1.5 0 0 1 18.5 20h-13A1.5 1.5 0 0 1 4 18.5v-13A1.5 1.5 0 0 1 5.5 4H8"/><path d="M8.5 3.5h7v4h-7z"/><path d="M9 13a3 3 0 1 0 6 0"/><path d="M7.5 18a4.5 4.5 0 0 1 9 0"/></svg><span>/</span><span>Kontakte</span></span>
      </div>
      <section class="page wide">
        <div>
          <h1>Kontakte</h1>
          <p class="lede">Verwaltung, Notdienst, Hausmeister, Beirat und freigegebene Kontakte für {{.Tenant.Address}}.</p>
        </div>
        <div class="contact-grid">
          <section class="panel contact-section">
            <div>
              <div class="kicker">Verwaltung</div>
              <h2>Hausverwaltung</h2>
            </div>
            {{if .HasManagerContacts}}
              <div class="contact-list">
                {{range .ManagerContacts}}
                  <article class="contact-card">
                    <div class="contact-head"><strong>{{.Name}}</strong><span class="pill">{{.Role}}</span></div>
                    <p class="muted">{{.Description}}</p>
                    <div class="contact-lines">{{if .HasEmail}}<a href="mailto:{{.Email}}">{{.Email}}</a>{{end}}{{if .HasPhone}}<a href="tel:{{.Phone}}">{{.Phone}}</a>{{end}}</div>
                  </article>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .ManagerEmpty}}
            {{end}}
          </section>

          <section class="panel contact-section">
            <div>
              <div class="kicker">Notfall</div>
              <h2>Notdienst &amp; Hausmeister</h2>
            </div>
            {{if .HasEmergencyContacts}}
              <div class="contact-list">
                {{range .EmergencyContacts}}
                  <article class="contact-card">
                    <div class="contact-head"><strong>{{.Name}}</strong><span class="pill">{{.Role}}</span></div>
                    <p class="muted">{{.Description}}</p>
                    <div class="contact-lines">{{if .HasEmail}}<a href="mailto:{{.Email}}">{{.Email}}</a>{{end}}{{if .HasPhone}}<a href="tel:{{.Phone}}">{{.Phone}}</a>{{end}}</div>
                  </article>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .EmergencyEmpty}}
            {{end}}
          </section>

          <section class="panel contact-section">
            <div>
              <div class="kicker">Beirat</div>
              <h2>Beirat</h2>
            </div>
            {{if .HasBoardContacts}}
              <div class="contact-list">
                {{range .BoardContacts}}
                  <article class="contact-card">
                    <div class="contact-head"><strong>{{.Name}}</strong><span class="pill">{{.Role}}</span></div>
                    <p class="muted">{{.Description}}</p>
                    <div class="contact-lines">{{if .HasEmail}}<a href="mailto:{{.Email}}">{{.Email}}</a>{{end}}{{if .HasPhone}}<a href="tel:{{.Phone}}">{{.Phone}}</a>{{end}}</div>
                  </article>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .BoardEmpty}}
            {{end}}
          </section>

          <section class="panel contact-section directory-panel">
            <div>
              <div class="kicker">Hausgemeinschaft</div>
              <h2>Freigegebene Kontakte</h2>
            </div>
            {{if .HasResidentContacts}}
              <div class="contact-list">
                {{range .ResidentContacts}}
                  <article class="contact-card">
                    <div class="contact-head"><strong>{{.Name}}</strong><span class="pill">{{.Role}}</span></div>
                    <p class="muted">{{.Description}}</p>
                    <div class="contact-lines">{{if .HasEmail}}<a href="mailto:{{.Email}}">{{.Email}}</a>{{end}}{{if .HasPhone}}<a href="tel:{{.Phone}}">{{.Phone}}</a>{{end}}</div>
                  </article>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .ResidentEmpty}}
            {{end}}
          </section>
        </div>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "issues"}}
{{template "appOpen" .}}
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M5 18.5V6.5A2.5 2.5 0 0 1 7.5 4h9A2.5 2.5 0 0 1 19 6.5v6A2.5 2.5 0 0 1 16.5 15H10l-5 3.5z"/></svg><span>/</span><span>Anliegen</span></span>
        {{if .CanManageIssues}}<div class="page-actions">{{if .BoardOnly}}<a class="button" href="/app/anliegen">Zurück zu Anliegen</a>{{else}}<a class="button" href="/app/anliegen/board">Triage-Board</a>{{end}}</div>{{end}}
      </div>
      <section class="page">
        <div>
          <h1>Anliegen</h1>
          <p class="lede">Mängel, Fragen und Vorschläge direkt an die Verwaltung melden.</p>
        </div>
        {{if not .BoardOnly}}<div class="issue-layout">
          {{if .CanCreateIssue}}
          <section class="panel">
            <div class="section-head">
              <div>
                <div class="kicker">Neues Anliegen</div>
                <p class="muted">Bitte so konkret wie möglich beschreiben. Ein Foto ist optional.</p>
              </div>
            </div>
            {{if .IssueMsg}}<p class="issue-flash{{if .IssueOK}} ok{{else}} warn{{end}}">{{.IssueMsg}}</p>{{end}}
            <form class="issue-form" method="post" action="/app/anliegen" enctype="multipart/form-data">
              <label>Kategorie
                <select name="category" required>
                  <option value="Reparatur">Reparatur</option>
                  <option value="Frage">Frage</option>
                  <option value="Vorschlag">Vorschlag</option>
                  <option value="Sonstiges">Sonstiges</option>
                </select>
              </label>
              <label>Ort
                <select name="location_type" required>
                  <option value="common">Gemeinschaftsbereich</option>
                  <option value="own-unit">Eigene Einheit</option>
                </select>
              </label>
              <label class="full">Titel
                <input type="text" name="title" maxlength="140" required placeholder="Kurz zusammenfassen">
              </label>
              <label class="full">Beschreibung
                <textarea name="body" maxlength="4000" required placeholder="Was ist passiert? Seit wann? Gibt es eine Dringlichkeit?"></textarea>
              </label>
              <label class="full">Details zum Ort
                <input type="text" name="location_detail" maxlength="160" placeholder="z. B. Stiegenhaus, Garage, Top 3">
              </label>
              <label class="full">Foto
                <span class="file-control"><input type="file" name="photo" accept="image/jpeg,image/png,image/webp"><span>Foto auswählen</span></span>
                <span class="hint">Optional, JPG/PNG/WebP bis 5 MB.</span>
              </label>
              <button type="submit">Anliegen senden</button>
            </form>
          </section>
          {{end}}
          <aside class="panel">
            <div class="section-head">
              <div>
                <div class="kicker">{{if and .CanManageIssues (not .BoardOnly)}}Meine eigenen Anliegen{{else if eq .Role "Beirat"}}Anliegen im Haus{{else}}Meine letzten Anliegen{{end}}</div>
                <p class="muted">Status und Rückfragen werden hier zusammengeführt.</p>
              </div>
            </div>
            {{if .HasIssues}}
              <div class="issue-list">
                {{range .Issues}}
                  <article class="issue-card">
                    <div class="issue-meta">
                      <span class="pill {{.StatusClass}}">{{.Status}}</span>
                      <span class="pill">{{.Category}}</span>
                      <span class="pill">{{.Priority}}</span>
                      <span>{{.CreatedAt}}</span>
                    </div>
	                    <h3>{{.Title}}</h3>
	                    <p class="issue-location">{{.Location}}{{if .HasAssignee}} · Zuständig: {{.AssigneeEmail}}{{end}}{{if .HasPhotos}} · {{.PhotoCount}} Foto{{if ne .PhotoCount 1}}s{{end}}{{end}}</p>
                    <div class="comment-thread">
                      {{if .HasComments}}
                        {{range .Comments}}<div class="comment"><span class="comment-meta">{{.Author}} · {{.CreatedAt}}</span><p>{{.Body}}</p></div>{{end}}
                      {{else}}
                        <p class="empty">Noch keine Kommentare.</p>
                      {{end}}
                    </div>
	                    {{if .CanComment}}<form class="comment-form" method="post" action="/app/anliegen/comment">
	                      <input type="hidden" name="id" value="{{.ID}}">
	                      <textarea name="body" maxlength="3000" required placeholder="Kommentar oder Ergänzung schreiben" aria-label="Kommentar oder Ergänzung"></textarea>
	                      <button type="submit">Kommentar senden</button>
	                    </form>{{end}}
                    {{if or .CanClose .CanReopen}}
                      <div class="issue-actions">
                        {{if .CanClose}}<form method="post" action="/app/anliegen/workflow"><input type="hidden" name="id" value="{{.ID}}"><input type="hidden" name="status" value="Erledigt"><button class="ghost" type="submit">Erledigt melden</button></form>{{end}}
                        {{if .CanReopen}}<form method="post" action="/app/anliegen/workflow"><input type="hidden" name="id" value="{{.ID}}"><input type="hidden" name="status" value="Neu"><button class="ghost" type="submit">Wieder öffnen</button></form>{{end}}
                      </div>
                    {{end}}
                  </article>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .IssuesEmpty}}
            {{end}}
          </aside>
        </div>{{end}}
        {{if .CanManageIssues}}
          <section class="panel">
            <div class="section-head">
              <div>
                <div class="kicker">Anliegen verwalten</div>
                <p class="muted">Status, Priorität und Zuständigkeit innerhalb der Hausverwaltung setzen.</p>
              </div>
            </div>
            <form class="issue-board-filter" method="get" action="{{.BoardAction}}">
              <label>Status
                <select name="status">
                  {{range .BoardFilters.StatusOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
                </select>
              </label>
              <label>Priorität
                <select name="priority">
                  {{range .BoardFilters.PriorityOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
                </select>
              </label>
              <label>Kategorie
                <select name="category">
                  {{range .BoardFilters.CategoryOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
                </select>
              </label>
              <label class="assignee">Zuständig
                <input type="email" name="assignee" value="{{.BoardFilters.Assignee}}" placeholder="name@example.com">
              </label>
              <label class="sort">Sortierung
                <select name="sort">
                  {{range .BoardFilters.SortOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
                </select>
              </label>
              <div class="board-filter-actions">
                <button type="submit">Filtern</button>
                {{if .BoardFilters.HasActive}}<a href="{{.BoardAction}}">Zurücksetzen</a>{{end}}
              </div>
            </form>
            {{if .HasManageIssues}}
              <div class="issue-list">
                {{range .ManageIssues}}
                  <article class="issue-card">
                    <div class="issue-meta">
                      <span class="pill {{.StatusClass}}">{{.Status}}</span>
	                      <span class="pill">{{.Category}}</span>
	                      <span class="pill">{{.Priority}}</span>
	                      <span>{{.Author}}</span>
	                      <span>{{.CreatedAt}}</span>
	                    </div>
                    <h3>{{.Title}}</h3>
                    <p class="issue-location">{{.Location}}{{if .HasAssignee}} · Zuständig: {{.AssigneeEmail}}{{end}}{{if .HasPhotos}} · {{.PhotoCount}} Foto{{if ne .PhotoCount 1}}s{{end}}{{end}}</p>
                    <div class="comment-thread">
                      {{if .HasComments}}
                        {{range .Comments}}<div class="comment"><span class="comment-meta">{{.Author}} · {{.CreatedAt}}</span><p>{{.Body}}</p></div>{{end}}
                      {{else}}
                        <p class="empty">Noch keine Kommentare.</p>
                      {{end}}
                    </div>
	                    {{if .CanComment}}<form class="comment-form" method="post" action="/app/anliegen/comment">
	                      <input type="hidden" name="id" value="{{.ID}}">
	                      <textarea name="body" maxlength="3000" required placeholder="Kommentar oder Rückfrage schreiben" aria-label="Kommentar oder Rückfrage"></textarea>
	                      <button type="submit">Kommentar senden</button>
	                    </form>{{end}}
                    <form class="issue-actions" method="post" action="/app/anliegen/workflow">
                      <input type="hidden" name="id" value="{{.ID}}">
                      <label>Status
                        <select name="status">
                          {{range .StatusOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
                        </select>
                      </label>
                      <label>Priorität
                        <select name="priority">
                          {{range .PriorityOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
                        </select>
                      </label>
                      <label class="assignee">Zuständig
                        <input type="email" name="assignee_email" value="{{.AssigneeEmail}}" placeholder="name@example.com">
                      </label>
                      <button type="submit">Aktualisieren</button>
                    </form>
                  </article>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .ManageIssuesEmpty}}
            {{end}}
          </section>
        {{end}}
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "announcements"}}
{{template "appOpen" .}}
    <script src="/assets/announcements.js" defer></script>
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M4 5h16v13H7l-3 3z"/><path d="M8 9h8M8 13h6"/></svg><span>/</span><span>Aushang</span></span>
        {{if .CanManageAnnouncements}}<div class="page-actions"><button class="button primary" type="button" data-dialog="announcement-create" aria-haspopup="dialog" aria-controls="announcement-create"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 5v14M5 12h14" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Neu verfassen</button></div>{{end}}
      </div>
      <section class="page">
        <div>
          <h1>Aushang</h1>
          <p class="lede">Offizielle Informationen, Termine und Hinweise der Hausgemeinschaft.</p>
        </div>
        {{if .AnnounceMsg}}<p class="flash ok">{{.AnnounceMsg}}</p>{{end}}
        <div class="home-grid">
          <section class="panel">
            <div class="section-head">
              <div class="kicker">Archiv</div>
              {{if .HasAnnouncements}}<span class="pill">{{len .Announcements}} Treffer</span>{{end}}
            </div>
            <div class="archive-tools">
              <form class="filter-form" method="get" action="/app/announcements">
                {{if .SelectedCategory}}<input type="hidden" name="category" value="{{.SelectedCategory}}">{{end}}
                <label for="announcement-search">Suche<input id="announcement-search" name="q" value="{{.SearchQuery}}" placeholder="Titel, Text oder Kategorie"></label>
                <button class="button" type="submit">Suchen</button>
              </form>
              <div class="filter-tabs" aria-label="Aushang-Kategorien">
                {{range .CategoryFilters}}<a class="filter-tab {{if .Active}}active{{end}}" href="{{.URL}}">{{.Label}}</a>{{end}}
              </div>
            </div>
            {{if .HasAnnouncements}}
              <div class="entries">
                {{range .Announcements}}
                  <article class="entry">
                    <div class="entry-head">
                      <div>
                        <h3>{{.Title}}</h3>
                        <div class="entry-meta">
                          <span class="pill {{.CategoryClass}}">{{.Category}}</span>
                          {{if .Unread}}<span class="pill unread">neu</span>{{end}}
                          {{if .Pinned}}<span class="pill">Fixiert</span>{{end}}
                          {{if .Status}}<span class="pill">{{.Status}}</span>{{end}}
                          <span>{{.PublishedAt}}</span>
                          {{if .HasExpiresAt}}<span>bis {{.ExpiresAt}}</span>{{end}}
                        </div>
                      </div>
                    </div>
                    <div class="entry-body">{{.BodyHTML}}</div>
                  </article>
                {{end}}
              </div>
            {{else}}
              {{if .HasAnyAnnouncements}}
                {{template "emptyState" .AnnouncementsEmpty}}
              {{else}}
                {{template "emptyState" .AnnouncementsBlank}}
              {{end}}
            {{end}}
          </section>

          <section class="panel">
            <div class="kicker">Aushang verwalten</div>
            {{if .CanManageAnnouncements}}
              <div class="quick-list">
                <button class="quick-row" type="button" data-dialog="announcement-create" aria-haspopup="dialog" aria-controls="announcement-create">
                  <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>
                  <div><h3>Neu verfassen</h3><p>Kategorie, Fixierung, Veröffentlichung und Ablaufdatum setzen.</p></div>
                  <span class="quick-arrow">›</span>
                </button>
                {{if .HasAllAnnouncements}}
                  {{range .AllAnnouncements}}
                    <div class="quick-row">
                      <svg viewBox="0 0 24 24"><path d="M4 5h16v13H7l-3 3z"/><path d="M8 9h8M8 13h6"/></svg>
                      <div><h3>{{.Title}}</h3><p>{{.Category}} · {{.PublishedAt}}{{if .Status}} · {{.Status}}{{end}}</p></div>
                      <span class="entry-actions">
                        <button class="button small" type="button" data-dialog="{{.EditDialogID}}" aria-haspopup="dialog" aria-controls="{{.EditDialogID}}">Bearbeiten</button>
                        <form method="post" action="/app/announcements/delete" data-confirm="{{.DeleteConfirmLabel}}">
                          <input type="hidden" name="id" value="{{.ID}}">
                          <button class="button small" type="submit">Löschen</button>
                        </form>
                      </span>
                    </div>
                    <dialog id="{{.EditDialogID}}" class="dialog" aria-labelledby="{{.EditDialogID}}-title">
                      <form method="post" action="/app/announcements/edit">
                        <input type="hidden" name="id" value="{{.ID}}">
                        <div class="dialog-head">
                          <h2 id="{{.EditDialogID}}-title">Aushang bearbeiten</h2>
                          <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
                        </div>
                        <div class="dialog-body">
                          <div class="dialog-grid">
                            <label class="full" for="title-{{.ID}}">Titel<input id="title-{{.ID}}" name="title" value="{{.Title}}" required maxlength="140"></label>
                            <label for="category-{{.ID}}">Kategorie<select id="category-{{.ID}}" name="category">
                              <option value="Info"{{if eq .Category "Info"}} selected{{end}}>Info</option>
                              <option value="Termin"{{if eq .Category "Termin"}} selected{{end}}>Termin</option>
                              <option value="Wartung"{{if eq .Category "Wartung"}} selected{{end}}>Wartung</option>
                              <option value="Dringend"{{if eq .Category "Dringend"}} selected{{end}}>Dringend</option>
                            </select></label>
                            <label for="published-{{.ID}}">Veröffentlichen<input id="published-{{.ID}}" type="datetime-local" name="published_at" value="{{.PublishedAtInput}}"></label>
                            <label for="expires-{{.ID}}">Ablauf optional<input id="expires-{{.ID}}" type="datetime-local" name="expires_at" value="{{.ExpiresAtInput}}"></label>
                            <label class="check-row"><input type="checkbox" name="pinned" value="true"{{if .PinnedChecked}} checked{{end}}> oben fixieren</label>
                            <label class="full" for="body-{{.ID}}">Text<textarea id="body-{{.ID}}" name="body" required>{{.Body}}</textarea></label>
                          </div>
                          <button class="button primary" type="submit">Speichern</button>
                        </div>
                      </form>
                    </dialog>
                  {{end}}
            {{else}}
                  {{template "emptyState" .AllAnnouncementsEmpty}}
            {{end}}
              </div>
            {{else}}
              <p class="empty">Veröffentlichen und Bearbeiten ist der Verwaltung vorbehalten.</p>
            {{end}}
          </section>
        </div>
      </section>

      {{if .CanManageAnnouncements}}
      <dialog id="announcement-create" class="dialog" aria-labelledby="announcement-create-title">
        <form method="post" action="/app/announcements">
          <div class="dialog-head">
            <h2 id="announcement-create-title">Neu verfassen</h2>
            <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
          </div>
          <div class="dialog-body">
            <div class="dialog-grid">
              <label class="full" for="announcement-title">Titel<input id="announcement-title" name="title" required maxlength="140"></label>
              <label for="announcement-category">Kategorie<select id="announcement-category" name="category">
                <option value="Info">Info</option>
                <option value="Termin">Termin</option>
                <option value="Wartung">Wartung</option>
                <option value="Dringend">Dringend</option>
              </select></label>
              <label for="announcement-published">Veröffentlichen<input id="announcement-published" type="datetime-local" name="published_at" value="{{.NowInput}}"></label>
              <label for="announcement-expires">Ablauf optional<input id="announcement-expires" type="datetime-local" name="expires_at"></label>
              <label class="check-row"><input type="checkbox" name="pinned" value="true"> oben fixieren</label>
              <label class="full" for="announcement-body">Text<textarea id="announcement-body" name="body" required></textarea></label>
            </div>
            <button class="button primary" type="submit">Veröffentlichen</button>
          </div>
        </form>
      </dialog>
      {{end}}
    </main>
{{template "appClose" .}}
{{end}}

{{define "events"}}
{{template "appOpen" .}}
    <script src="/assets/announcements.js" defer></script>
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M7 3v4M17 3v4"/><path d="M4.5 6h15v14h-15z"/><path d="M4.5 10h15"/><path d="M8 14h.01M12 14h.01M16 14h.01"/></svg><span>/</span><span>Termine</span></span>
        {{if .CanManageEvents}}<div class="page-actions"><button class="button primary" type="button" data-dialog="event-create" aria-haspopup="dialog" aria-controls="event-create"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 5v14M5 12h14" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Termin anlegen</button></div>{{end}}
      </div>
      <section class="page">
        <div>
          <h1>Termine</h1>
          <p class="lede">Versammlungen, Wartungen, Fristen und gemeinsame Ereignisse im Haus.</p>
        </div>
        {{if .EventMsg}}<p class="flash ok">{{.EventMsg}}</p>{{end}}
        <div class="home-grid">
          <section class="panel">
            <div class="section-head">
              <div class="kicker">Kommende Termine</div>
              {{if .HasEvents}}<span class="pill">{{len .Events}} geplant</span>{{end}}
            </div>
            {{if .HasEvents}}
              <div class="agenda-list">
                {{range .Events}}
                  <article class="event-card">
                    <span class="date-badge"><strong>{{.DateBadgeDay}}</strong><span>{{.DateBadgeMonth}}</span></span>
                    <div class="event-info">
                      <h3>{{.Title}}</h3>
                      <div class="event-meta">
                        <span class="pill {{.CategoryClass}}">{{.Category}}</span>
                        <span>{{.StartsAt}}</span>
                        <span>{{.TimeRange}}</span>
                        {{if .HasLocation}}<span>{{.Location}}</span>{{end}}
                        {{if .Status}}<span class="pill">{{.Status}}</span>{{end}}
                      </div>
                      {{if .HasBody}}<div class="entry-body">{{.BodyHTML}}</div>{{end}}
                    </div>
                  </article>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .EventsEmpty}}
            {{end}}
          </section>

          <section class="panel">
            <div class="kicker">Termine verwalten</div>
            {{if .CanManageEvents}}
              <div class="quick-list">
                <button class="quick-row" type="button" data-dialog="event-create" aria-haspopup="dialog" aria-controls="event-create">
                  <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>
                  <div><h3>Termin anlegen</h3><p>Kategorie, Zeitpunkt, Ort und Details speichern.</p></div>
                  <span class="quick-arrow">›</span>
                </button>
                {{if .HasAllEvents}}
                  {{range .AllEvents}}
                    <div class="quick-row">
                      <svg viewBox="0 0 24 24"><path d="M7 3v4M17 3v4"/><path d="M4.5 6h15v14h-15z"/><path d="M4.5 10h15"/><path d="M8 14h.01M12 14h.01M16 14h.01"/></svg>
                      <div><h3>{{.Title}}</h3><p>{{.Category}} · {{.StartsAt}}{{if .Past}} · vergangen{{end}}</p></div>
                      <span class="entry-actions">
                        <button class="button small" type="button" data-dialog="{{.EditDialogID}}" aria-haspopup="dialog" aria-controls="{{.EditDialogID}}">Bearbeiten</button>
                        <form method="post" action="/app/events/delete" data-confirm="{{.DeleteConfirmLabel}}">
                          <input type="hidden" name="id" value="{{.ID}}">
                          <button class="button small" type="submit">Löschen</button>
                        </form>
                      </span>
                    </div>
                    <dialog id="{{.EditDialogID}}" class="dialog" aria-labelledby="{{.EditDialogID}}-title">
                      <form method="post" action="/app/events/edit">
                        <input type="hidden" name="id" value="{{.ID}}">
                        <div class="dialog-head">
                          <h2 id="{{.EditDialogID}}-title">Termin bearbeiten</h2>
                          <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
                        </div>
                        <div class="dialog-body">
                          <div class="dialog-grid">
                            <label class="full" for="event-title-{{.ID}}">Titel<input id="event-title-{{.ID}}" name="title" value="{{.Title}}" required maxlength="140"></label>
                            <label for="event-category-{{.ID}}">Kategorie<select id="event-category-{{.ID}}" name="category">
                              <option value="Eigentümerversammlung"{{if eq .Category "Eigentümerversammlung"}} selected{{end}}>Eigentümerversammlung</option>
                              <option value="Reinigung"{{if eq .Category "Reinigung"}} selected{{end}}>Reinigung</option>
                              <option value="Wartung"{{if eq .Category "Wartung"}} selected{{end}}>Wartung</option>
                              <option value="Ablesung"{{if eq .Category "Ablesung"}} selected{{end}}>Ablesung</option>
                              <option value="Frist"{{if eq .Category "Frist"}} selected{{end}}>Frist</option>
                              <option value="Sonstiges"{{if eq .Category "Sonstiges"}} selected{{end}}>Sonstiges</option>
                            </select></label>
                            <label for="event-start-{{.ID}}">Beginn<input id="event-start-{{.ID}}" type="datetime-local" name="starts_at" value="{{.StartsAtInput}}" required></label>
                            <label for="event-end-{{.ID}}">Ende optional<input id="event-end-{{.ID}}" type="datetime-local" name="ends_at" value="{{.EndsAtInput}}"></label>
                            <label class="full" for="event-location-{{.ID}}">Ort<input id="event-location-{{.ID}}" name="location" value="{{.Location}}" maxlength="160"></label>
                            <label class="full" for="event-body-{{.ID}}">Details<textarea id="event-body-{{.ID}}" name="body">{{.Body}}</textarea></label>
                          </div>
                          <button class="button primary" type="submit">Speichern</button>
                        </div>
                      </form>
                    </dialog>
                  {{end}}
            {{else}}
                  {{template "emptyState" .AllEventsEmpty}}
            {{end}}
              </div>
            {{else}}
              <p class="empty">Anlegen und Bearbeiten ist der Verwaltung vorbehalten.</p>
            {{end}}
          </section>
        </div>
      </section>

      {{if .CanManageEvents}}
      <dialog id="event-create" class="dialog" aria-labelledby="event-create-title">
        <form method="post" action="/app/events">
          <div class="dialog-head">
            <h2 id="event-create-title">Termin anlegen</h2>
            <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
          </div>
          <div class="dialog-body">
            <div class="dialog-grid">
              <label class="full" for="event-title">Titel<input id="event-title" name="title" required maxlength="140"></label>
              <label for="event-category">Kategorie<select id="event-category" name="category">
                <option value="Eigentümerversammlung">Eigentümerversammlung</option>
                <option value="Reinigung">Reinigung</option>
                <option value="Wartung">Wartung</option>
                <option value="Ablesung">Ablesung</option>
                <option value="Frist">Frist</option>
                <option value="Sonstiges">Sonstiges</option>
              </select></label>
              <label for="event-start">Beginn<input id="event-start" type="datetime-local" name="starts_at" value="{{.NowInput}}" required></label>
              <label for="event-end">Ende optional<input id="event-end" type="datetime-local" name="ends_at"></label>
              <label class="full" for="event-location">Ort<input id="event-location" name="location" maxlength="160"></label>
              <label class="full" for="event-body">Details<textarea id="event-body" name="body"></textarea></label>
            </div>
            <button class="button primary" type="submit">Speichern</button>
          </div>
        </form>
      </dialog>
      {{end}}
    </main>
{{template "appClose" .}}
{{end}}

{{define "documents"}}
{{template "appOpen" .}}
    <script src="/assets/announcements.js" defer></script>
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/><path d="M9 13h6M9 17h6"/></svg><span>/</span><span>Dokumente</span></span>
        {{if .CanManageDocuments}}<div class="page-actions"><button class="button primary" type="button" data-dialog="document-upload" aria-haspopup="dialog" aria-controls="document-upload"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 5v14M5 12h14" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Dokument hochladen</button></div>{{end}}
      </div>
      <section class="page">
        <div>
          <h1>Dokumente</h1>
          <p class="lede">Protokolle, Abrechnungen, Hausordnung und Unterlagen nach Berechtigung der jeweiligen Person.</p>
        </div>
        {{if .DocumentMsg}}<p class="flash {{if .DocumentOK}}ok{{end}}">{{.DocumentMsg}}</p>{{end}}
        <div class="home-grid">
          <section class="panel">
            <div class="section-head">
              <div class="kicker">Ablage</div>
              {{if .HasDocuments}}<span class="pill">{{len .Documents}} Treffer</span>{{else if .HasAnyDocuments}}<span class="pill">0 Treffer</span>{{else}}<span class="pill">Noch leer</span>{{end}}
            </div>
            <form class="filter-form document-filter" method="get" action="/app/dokumente">
              <label for="document-search">Suchen<input id="document-search" name="q" value="{{.SearchQuery}}" placeholder="Titel, Kategorie, Datei oder Person"></label>
              <label for="document-sort">Sortierung<select id="document-sort" name="sort">
                {{range .SortOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
              </select></label>
              <button class="button" type="submit">Suchen</button>
            </form>
            <div class="document-sections">
              {{range .DocumentSections}}
                <section class="document-section">
                  <h3>{{.Category}}</h3>
                  {{if .HasDocuments}}
                    <div class="document-list">
                      {{range .Documents}}
                        <article class="document-row">
                          <div>
                            <strong>{{.Title}}</strong>
                            <div class="document-meta">
                              <span class="pill {{.VisibilityClass}}">{{.Visibility}}</span>
                              {{if .HasUnit}}<span class="pill">Einheit {{.UnitLabel}}</span>{{end}}
                              <span>{{.VersionLabel}}</span>
                              <span>{{.UploadedAt}}</span>
                              <span>{{.Size}}</span>
                              <span class="document-file">{{.Filename}}</span>
                            </div>
                          </div>
                          <div class="document-side">
                            <span class="pill">{{.Category}}</span>
                            <a class="button small" href="{{.DownloadURL}}">Herunterladen</a>
                            {{if $.CanManageDocuments}}<button class="button small" type="button" data-dialog="{{.ReplaceDialogID}}" aria-haspopup="dialog" aria-controls="{{.ReplaceDialogID}}">Ersetzen</button>{{end}}
                          </div>
                          {{if .HasVersions}}
                            <details class="document-versions">
                              <summary>Ältere Versionen</summary>
                              <div class="version-list">
                                {{range .Versions}}
                                  <div class="version-row">
                                    <span>{{.Version}} · {{.UploadedAt}} · {{.Size}} · {{.Filename}}</span>
                                    <a class="button small" href="{{.DownloadURL}}">Herunterladen</a>
                                  </div>
                                {{end}}
                              </div>
                            </details>
                          {{end}}
                        </article>
                        {{if $.CanManageDocuments}}
                        <dialog id="{{.ReplaceDialogID}}" class="dialog" aria-labelledby="{{.ReplaceDialogID}}-title">
                          <form method="post" action="/app/dokumente/replace" enctype="multipart/form-data">
                            <input type="hidden" name="id" value="{{.ID}}">
                            <div class="dialog-head">
                              <h2 id="{{.ReplaceDialogID}}-title">Neue Version hochladen</h2>
                              <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
                            </div>
                            <div class="dialog-body">
                              <p class="mini">{{.Title}} · aktuell {{.VersionLabel}}</p>
                              <div class="dialog-grid">
                                <label class="full" for="{{.ReplaceDialogID}}-file">Datei<input id="{{.ReplaceDialogID}}-file" type="file" name="document" accept="application/pdf,image/jpeg,image/png,image/webp" required></label>
                              </div>
                              <p class="mini">Kategorie, Sichtbarkeit und Einheit bleiben unverändert; die bisherige Version bleibt im Verlauf abrufbar.</p>
                              <button class="button primary" type="submit">Version speichern</button>
                            </div>
                          </form>
                        </dialog>
                        {{end}}
                      {{end}}
                    </div>
                  {{else}}
                    <p class="empty">{{.EmptyMessage}}</p>
                  {{end}}
                </section>
              {{end}}
            </div>
          </section>

          <section class="panel">
            <div class="kicker">Dokumentenverwaltung</div>
            {{if .CanManageDocuments}}
              <div class="quick-list">
                <button class="quick-row" type="button" data-dialog="document-upload" aria-haspopup="dialog" aria-controls="document-upload">
                  <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>
                  <div><h3>Dokument hochladen</h3><p>Kategorie, Sichtbarkeit und Datei bis {{.MaxDocumentSize}} speichern.</p></div>
                  <span class="quick-arrow">›</span>
                </button>
                <div class="legend" aria-label="Sichtbarkeiten für Dokumente">
                  <div><strong>Alle Bewohner</strong><span>Allgemeine Informationen wie Hausordnung oder Hinweise.</span></div>
                  <div><strong>Nur Eigentümer</strong><span>Unterlagen für Eigentümer, etwa Protokolle oder Abrechnungen.</span></div>
                  <div><strong>Nur Verwaltung</strong><span>Interne Arbeitsdokumente der Verwaltung.</span></div>
                </div>
              </div>
            {{else}}
              <p class="empty">Hochladen und Sichtbarkeit setzen ist der Verwaltung vorbehalten.</p>
            {{end}}
          </section>
        </div>
      </section>

      {{if .CanManageDocuments}}
      <dialog id="document-upload" class="dialog" aria-labelledby="document-upload-title">
        <form method="post" action="/app/dokumente" enctype="multipart/form-data">
          <div class="dialog-head">
            <h2 id="document-upload-title">Dokument hochladen</h2>
            <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
          </div>
          <div class="dialog-body">
            <div class="dialog-grid">
              <label class="full" for="document-title">Titel<input id="document-title" name="title" required maxlength="160" autocomplete="off"></label>
              <label for="document-category">Kategorie<select id="document-category" name="category" required>
                {{range .CategoryOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
              </select></label>
              <label for="document-visibility">Sichtbarkeit<select id="document-visibility" name="visibility" required>
                {{range .VisibilityOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
              </select></label>
              <label for="document-unit">Einheit optional<select id="document-unit" name="unit_id">
                {{range .UnitOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
              </select></label>
              <label class="full" for="document-file">Datei<input id="document-file" type="file" name="document" accept="application/pdf,image/jpeg,image/png,image/webp" required></label>
            </div>
            <p class="mini">Erlaubt sind PDF, JPG, PNG oder WebP bis {{.MaxDocumentSize}}. Dateien werden nicht öffentlich ausgeliefert.</p>
            <button class="button primary" type="submit">Hochladen</button>
          </div>
        </form>
      </dialog>
      {{end}}
    </main>
{{template "appClose" .}}
{{end}}

{{define "ballotProtocol"}}
<!doctype html>
<html lang="de">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
    * { box-sizing: border-box; }
    body { margin: 0; background: #f7f3ea; color: #20251f; font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    main { width: min(920px,100%); margin: 0 auto; padding: 42px 28px; }
    h1, h2, h3 { font-family: Spectral, serif; margin: 0; }
    h1 { font-size: 42px; line-height: 1; }
    h2 { font-size: 24px; margin-top: 28px; }
    h3 { font-size: 18px; }
    p { margin: 0; line-height: 1.5; }
    .protocol-head { display: grid; gap: 10px; border-bottom: 2px solid #20251f; padding-bottom: 22px; }
    .meta { display: flex; flex-wrap: wrap; gap: 8px; color: #6b6f63; font-size: 13px; font-weight: 700; }
    .pill { display: inline-flex; align-items: center; min-height: 26px; border-radius: 999px; padding: 3px 10px; font-size: 12px; font-weight: 800; background: rgba(200,153,63,.16); color: #8a6a1f; }
    .pill.ok { background: rgba(47,107,74,.12); color: #2f6b4a; }
    .summary { margin-top: 24px; display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 12px; }
    .box { border: 1px solid #e7e0d2; border-radius: 8px; background: #fffefb; padding: 14px; }
    .box span { display: block; color: #8a7b3f; font-size: 11px; font-weight: 800; text-transform: uppercase; letter-spacing: .06em; }
    .box strong { display: block; margin-top: 6px; font-family: Spectral, serif; font-size: 22px; }
    table { width: 100%; margin-top: 14px; border-collapse: collapse; background: #fffefb; border: 1px solid #e7e0d2; }
    th, td { padding: 12px 14px; border-bottom: 1px solid #e7e0d2; text-align: left; }
    th { color: #8a7b3f; font-size: 11px; text-transform: uppercase; letter-spacing: .06em; }
    tr:last-child td { border-bottom: 0; }
    .num { text-align: right; font-variant-numeric: tabular-nums; }
    footer { margin-top: 28px; color: #6b6f63; font-size: 12px; }
    @media print {
      body { background: #fff; }
      main { padding: 24px 0; }
      .box, table { break-inside: avoid; }
    }
    @media (max-width: 680px) {
      main { padding: 28px 18px; }
      h1 { font-size: 34px; }
      .summary { grid-template-columns: 1fr; }
      table { font-size: 13px; }
    }
  </style>
</head>
<body>
  <main>
    <section class="protocol-head">
      <div class="meta"><span>{{.Tenant.Name}}</span><span>{{.Tenant.Address}}</span><span>Erstellt {{.GeneratedAt}}</span></div>
      <h1>Abstimmungsprotokoll</h1>
      <h2>{{.Ballot.Title}}</h2>
      <div class="meta">
        <span class="pill {{.Ballot.StatusClass}}">{{.Ballot.Status}}</span>
        <span>{{.Ballot.Type}}</span>
        <span>{{.Ballot.Weighting}}</span>
        {{if .Ballot.HasQuorum}}<span>Quorum {{.Ballot.Quorum}}</span>{{end}}
        {{if .Ballot.HasClosesAt}}<span>Frist {{.Ballot.ClosesAt}}</span>{{end}}
      </div>
      {{if .Ballot.HasDescription}}<p>{{.Ballot.Description}}</p>{{end}}
    </section>

    <section class="summary" aria-label="Zusammenfassung">
      <div class="box"><span>Teilnahme</span><strong>{{.Ballot.Participation}}</strong></div>
      <div class="box"><span>Stimmgewicht</span><strong>{{.Ballot.TotalWeightLabel}}</strong></div>
      <div class="box"><span>Quorum</span><strong>{{.Ballot.QuorumStatus}}</strong></div>
      <div class="box"><span>Stimmberechtigt</span><strong>{{.Ballot.EligibleWeightLabel}}</strong></div>
      <div class="box"><span>Stimmen</span><strong>{{.Ballot.TotalVotes}}</strong></div>
      <div class="box"><span>Ergebnis</span><strong>{{if .Ballot.HasWinner}}{{.Ballot.WinnerLabel}}{{else}}Keine Stimmen{{end}}</strong></div>
    </section>

    <section>
      <h2>Auszählung</h2>
      <table aria-label="Auszählung">
        <thead><tr><th>Option</th><th class="num">Gewicht</th><th class="num">Stimmen</th><th class="num">Anteil</th></tr></thead>
        <tbody>
          {{range .Ballot.Options}}
            <tr><td><strong>{{.Label}}</strong></td><td class="num">{{.WeightLabel}}</td><td class="num">{{.VoteCount}}</td><td class="num">{{.Percent}} %</td></tr>
          {{end}}
        </tbody>
      </table>
    </section>
    <footer>{{.AppVersion}}</footer>
  </main>
</body>
</html>
{{end}}

{{define "ballots"}}
{{template "appOpen" .}}
    <script src="/assets/announcements.js" defer></script>
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M5 19V9M12 19V5M19 19v-7"/><path d="M3.5 19h17"/></svg><span>/</span><span>Abstimmungen</span></span>
        {{if .CanManageVotes}}<div class="page-actions"><button class="button primary" type="button" data-dialog="ballot-create" aria-haspopup="dialog" aria-controls="ballot-create"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 5v14M5 12h14" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Abstimmung anlegen</button></div>{{end}}
      </div>
      <section class="page">
        <div>
          <h1>Abstimmungen</h1>
          <p class="lede">Beschlüsse, Umlaufbeschlüsse und Stimmabgabe für Eigentümer der Gemeinschaft.</p>
        </div>
        {{if .VoteMsg}}<p class="flash {{if .VoteOK}}ok{{end}}">{{.VoteMsg}}</p>{{end}}
        <div class="home-grid">
          <section class="panel">
            <div class="section-head">
              <div class="kicker">Abstimmungen</div>
              {{if .HasBallots}}<span class="pill">{{.BallotCountLabel}}</span>{{else}}<span class="pill">Noch leer</span>{{end}}
            </div>
            {{if .HasBallots}}
              <div class="vote-list">
                {{range .Ballots}}
                  <article class="vote-card">
                    <div class="vote-card-head">
                      <div>
                        <h3>{{.Title}}</h3>
                        <div class="vote-meta">
                          <span class="pill {{.StatusClass}}">{{.Status}}</span>
                          <span>{{.Type}}</span>
                          <span>{{.Weighting}}</span>
                          {{if .HasQuorum}}<span>Quorum {{.Quorum}}</span>{{end}}
                          {{if .HasClosesAt}}<span>Erinnerung {{.ReminderLabel}}</span>{{end}}
                          {{if .HasOpensAt}}<span>ab {{.OpensAt}}</span>{{end}}
                          {{if .HasClosesAt}}<span>bis {{.ClosesAt}}</span>{{end}}
                        </div>
                      </div>
                      {{if .HasVote}}<span class="pill ok">Stimme gespeichert</span>{{end}}
                    </div>
                    {{if .HasDescription}}<p class="muted">{{.Description}}</p>{{end}}
                    {{if .CanVote}}
                      <form method="post" action="/app/abstimmungen">
                        <input type="hidden" name="ballot_id" value="{{.ID}}">
                        <div class="vote-options">
                          {{range .Options}}
                            <label class="vote-option"><input type="radio" name="option" value="{{.Value}}" required{{if .Selected}} checked{{end}}> <span>{{.Label}}</span></label>
                          {{end}}
                        </div>
                        <div class="vote-actions">
                          <span class="mini">{{if .HasVote}}Aktuell: {{.VoteOption}} · {{.VoteWeight}} · {{.VotedAt}}{{else}}Stimmgewicht: {{.VoteWeight}}{{end}}</span>
                          <button class="button primary" type="submit">{{if .HasVote}}Stimme ändern{{else}}Stimme speichern{{end}}</button>
                        </div>
                      </form>
                    {{else}}
                      <div class="vote-options">
                        {{range .Options}}<div class="vote-option"><span></span><span>{{.Label}}</span></div>{{end}}
                      </div>
                      {{if .ReadOnlyMessage}}<p class="empty">{{.ReadOnlyMessage}}</p>{{end}}
                    {{end}}
                    {{if .HasResults}}
                      <div class="vote-result" aria-label="Abstimmungsergebnis">
                        <div class="vote-meta">
                          <span>{{.TotalVotes}} Stimmen</span>
                          <span>{{.TotalWeightLabel}} von {{.EligibleWeightLabel}}</span>
                          <span>Teilnahme {{.Participation}}</span>
                          <span class="pill {{.QuorumClass}}">{{.QuorumStatus}}</span>
                          {{if .HasWinner}}<span>Ergebnis {{.WinnerLabel}}</span>{{end}}
                          {{if .HasProtocol}}<a class="button small" href="{{.ProtocolURL}}">Protokoll</a>{{end}}
                        </div>
                        {{range .Options}}
                          <div class="vote-result-row">
                            <strong>{{.Label}}</strong>
                            <span class="vote-bar"><span style="width: {{.PercentStyle}}%;"></span></span>
                            <span>{{.WeightLabel}} · {{.VoteCount}} Stimmen</span>
                          </div>
                        {{end}}
                      </div>
                    {{end}}
                  </article>
                {{end}}
              </div>
            {{else}}
              {{template "emptyState" .BallotsEmpty}}
            {{end}}
          </section>

          <section class="panel">
            <div class="kicker">Verwaltung</div>
            {{if .CanManageVotes}}
              <div class="quick-list">
                <button class="quick-row" type="button" data-dialog="ballot-create" aria-haspopup="dialog" aria-controls="ballot-create">
                  <svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>
                  <div><h3>Abstimmung anlegen</h3><p>Optionen, Frist, Quorum und Gewichtung festlegen.</p></div>
                  <span class="quick-arrow">›</span>
                </button>
              </div>
              {{if .HasManageBallots}}
                <div class="vote-manage-list">
                  {{range .ManageBallots}}
                    <div class="vote-manage-row">
                      <div>
                        <strong>{{.Title}}</strong>
                        <div class="vote-meta"><span class="pill {{.StatusClass}}">{{.Status}}</span><span>{{.Type}}</span><span>{{.Weighting}}</span>{{if .HasClosesAt}}<span>Erinnerung {{.ReminderLabel}}</span>{{end}}<span>{{.UpdatedAt}}</span></div>
                      </div>
                      <div class="vote-manage-actions">
                        {{if .CanOpen}}<form method="post" action="/app/abstimmungen/open"><input type="hidden" name="id" value="{{.ID}}"><button class="button small" type="submit">Öffnen</button></form>{{end}}
                        {{if .CanClose}}<form method="post" action="/app/abstimmungen/close"><input type="hidden" name="id" value="{{.ID}}"><button class="button small" type="submit">Schließen</button></form>{{end}}
                        {{if .HasProtocol}}<a class="button small" href="{{.ProtocolURL}}">Protokoll</a>{{end}}
                      </div>
                    </div>
                  {{end}}
                </div>
              {{else}}
                {{template "emptyState" .ManageBallotsEmpty}}
              {{end}}
            {{else if .CanOversightVotes}}
              <p class="empty">Beirat sieht offene Abstimmungen und Ergebnisse lesend.</p>
            {{else}}
              <p class="empty">Abstimmungen werden von der Verwaltung angelegt.</p>
            {{end}}
          </section>
        </div>
      </section>

      {{if .CanManageVotes}}
      <dialog id="ballot-create" class="dialog" aria-labelledby="ballot-create-title">
        <form method="post" action="/app/abstimmungen">
          <div class="dialog-head">
            <h2 id="ballot-create-title">Abstimmung anlegen</h2>
            <button class="dialog-close" type="button" data-close-dialog aria-label="Schließen">&times;</button>
          </div>
          <div class="dialog-body">
            <div class="dialog-grid">
              <label class="full" for="ballot-title">Titel<input id="ballot-title" name="title" required maxlength="160" autocomplete="off"></label>
              <label for="ballot-type">Typ<select id="ballot-type" name="type" required>
                <option value="Umlaufbeschluss">Umlaufbeschluss</option>
                <option value="Versammlung">Versammlung</option>
              </select></label>
              <label for="ballot-weighting">Gewichtung<select id="ballot-weighting" name="weighting" required>
                <option value="per-share">nach Miteigentumsanteil</option>
                <option value="per-head">pro Kopf</option>
              </select></label>
              <label for="ballot-opens">Öffnen optional<input id="ballot-opens" type="datetime-local" name="opens_at" value="{{.NowInput}}"></label>
              <label for="ballot-closes">Frist optional<input id="ballot-closes" type="datetime-local" name="closes_at"></label>
              <label for="ballot-quorum">Quorum in %<input id="ballot-quorum" name="quorum_percent" inputmode="decimal" placeholder="50"></label>
              <label for="ballot-reminder">Erinnerung vor Frist (h)<input id="ballot-reminder" name="reminder_before_hours" inputmode="decimal" value="24"></label>
              <label class="full" for="ballot-options">Optionen<textarea id="ballot-options" name="options_text" required placeholder="Ja&#10;Nein&#10;Enthaltung"></textarea></label>
              <label class="full" for="ballot-description">Beschreibung<textarea id="ballot-description" name="description"></textarea></label>
            </div>
            <button class="button primary" type="submit">Anlegen</button>
          </div>
        </form>
      </dialog>
      {{end}}
    </main>
{{template "appClose" .}}
{{end}}

{{define "parking"}}
{{template "appOpen" .}}
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><span>Parkplatznutzung</span></span>
        <div class="page-actions">
          <a class="button" href="/app/parking"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M4 4v6h6" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"/><path d="M20 20v-6h-6" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"/><path d="M5 10a7 7 0 0 1 12-3M19 14a7 7 0 0 1-12 3" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/></svg>Aktualisieren</a>
          {{if .IsAdmin}}<a class="button" href="/app/parking/settings"><svg viewBox="0 0 24 24" width="17" height="17" aria-hidden="true"><path d="M12 8.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7z" fill="none" stroke="currentColor" stroke-width="1.9"/><path d="M19 12a7 7 0 0 0-.1-1l2-1.5-2-3.5-2.4 1a7 7 0 0 0-1.8-1L14.4 3h-4.8L9.3 6a7 7 0 0 0-1.8 1l-2.4-1-2 3.5 2 1.5A7 7 0 0 0 5 12a7 7 0 0 0 .1 1l-2 1.5 2 3.5 2.4-1a7 7 0 0 0 1.8 1l.3 3h4.8l.3-3a7 7 0 0 0 1.8-1l2.4 1 2-3.5-2-1.5a7 7 0 0 0 .1-1z" fill="none" stroke="currentColor" stroke-width="1.9"/></svg>Abrechnung konfigurieren</a>{{end}}
        </div>
      </div>
      <section class="page wide">
        <div>
          <h1>Parkplatznutzung</h1>
          <p class="lede">Private Lade- und Stellplatzabrechnung für die persönlich abgestimmte Nutzung.</p>
          <p class="muted subtle-note">Sichtbar nur für berechtigte Personen und gedacht für die private Abstimmung der Stellplatz- und Lade-Nutzung.</p>
        </div>

        <section class="panel status-strip">
          <div class="rule">
            <p>Nutzung nur nach persönlicher Absprache. Die Monatswerte berechnen sich stündlich aus Zählerdifferenz, aWATTar-Preis und Netzgebühr.</p>
            {{if .Telemetry.Configured}}<span class="pill ok">Home Assistant aktiv</span>{{end}}
          </div>
          {{if .Telemetry.Connected}}
            <div class="metric-grid">
              {{range .Telemetry.Metrics}}
                <div class="metric-card">
                  <span class="metric-label">{{.Label}}</span>
                  <strong class="metric-value">{{.Value}}</strong>
                  <code>{{.Detail}}</code>
                </div>
              {{end}}
            </div>
          {{else}}
            <p class="empty">{{.Telemetry.Message}}</p>
          {{end}}
        </section>

        <section class="panel accounting">
          <div class="section-head">
            <div>
              <h2>Monatsabrechnung</h2>
              <p class="muted">{{.Accounting.Message}}</p>
            </div>
          </div>
          {{if .Accounting.HasMonths}}
            <div class="month-strip">
              {{range .Accounting.Months}}
                <a class="month-card" href="{{.DetailPath}}">
                  <strong>{{.MonthLabel}}</strong>
                  <div class="bar"><span style="width: {{.ChartPercent}}%;"></span></div>
                  <span class="amount">{{.TotalCost}}</span>
                </a>
              {{end}}
            </div>
            <div class="table-wrap">
              <table aria-label="Monatsabrechnung Parkplatznutzung">
                <thead>
                  <tr>
                    <th>Monat</th>
                    <th class="num">Verbrauch</th>
                    <th class="num">Ø aWATTar</th>
                    <th class="num">Ø effektiv</th>
                    <th class="num">Strom</th>
                    <th class="num">Netzgeb.</th>
                    <th class="num">Summe</th>
                    <th>Status</th>
                    <th>Aktionen</th>
                  </tr>
                </thead>
                <tbody>
                  {{range .Accounting.Months}}
                    <tr>
                      <td class="month-cell"><a href="{{.DetailPath}}"><strong>{{.MonthLabel}}</strong></a>{{if .Partial}}<span>Teilmonat</span>{{end}}<span>{{.HourCount}} Stunden</span></td>
                      <td class="num">{{.KWh}}</td>
                      <td class="num">{{.AverageAwattar}}</td>
                      <td class="num">{{.EffectivePrice}}</td>
                      <td class="num">{{.EnergyCost}}</td>
                      <td class="num">{{.GridCost}}</td>
                      <td class="num amount">{{.TotalCost}}</td>
                      <td><span class="pill {{if .Paid}}ok{{end}}">{{.PaidLabel}}</span></td>
                      <td>
                        <div class="row-actions">
                          <a class="button small" href="{{.DetailPath}}">Details</a>
                          {{if $.IsAdmin}}
                            <form method="post" action="/app/parking/month">
                              <input type="hidden" name="month" value="{{.Month}}">
                              <input type="hidden" name="paid" value="{{.TogglePaidValue}}">
                              <button class="button small" type="submit">{{.ToggleLabel}}</button>
                            </form>
                          {{end}}
                        </div>
                      </td>
                    </tr>
                  {{end}}
                </tbody>
              </table>
            </div>
          {{else}}
            <p class="empty">Noch keine Monatswerte. Sobald zwei Zählerstände und mindestens ein aWATTar-Preis vorliegen, erscheint hier die erste Abrechnung.</p>
          {{end}}
        </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "parkingMonth"}}
{{template "appOpen" .}}
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/parking">Parkplatznutzung</a><span>/</span><span>{{.Detail.MonthLabel}}</span></span>
        <div class="page-actions"><a class="button" href="{{.Detail.BackPath}}">Monate</a></div>
      </div>
      <section class="page wide">
        <div>
          <h1>{{.Detail.MonthLabel}}</h1>
          <p class="lede">{{.Detail.Message}}</p>
        </div>
        <section class="panel status-strip">
          <div class="rule">
            <span class="pill">Netzgebühr {{.Detail.GridFeeLabel}}</span>
            {{if .Detail.LastSampleLabel}}<span class="mini">Letzter Zählerwert: {{.Detail.LastSampleLabel}}</span>{{end}}
          </div>
          {{if .Detail.Summary.Month}}
            <div class="metric-grid">
              <div class="metric-card"><span class="metric-label">Verbrauch</span><strong class="metric-value">{{.Detail.Summary.KWh}}</strong></div>
              <div class="metric-card"><span class="metric-label">Ø aWATTar</span><strong class="metric-value">{{.Detail.Summary.AverageAwattar}}</strong></div>
              <div class="metric-card"><span class="metric-label">Ø effektiv</span><strong class="metric-value">{{.Detail.Summary.EffectivePrice}}</strong></div>
              <div class="metric-card"><span class="metric-label">Strom</span><strong class="metric-value">{{.Detail.Summary.EnergyCost}}</strong></div>
              <div class="metric-card"><span class="metric-label">Netzgeb.</span><strong class="metric-value">{{.Detail.Summary.GridCost}}</strong></div>
              <div class="metric-card"><span class="metric-label">Summe</span><strong class="metric-value">{{.Detail.Summary.TotalCost}}</strong></div>
            </div>
          {{end}}
        </section>
        <section class="panel accounting">
          <h2>Stundenwerte</h2>
          <div class="legend" aria-label="Legende für Stundenwerte">
            <div><strong>Stunde</strong><span>Beginn der Abrechnungsstunde; jede Zeile umfasst diese Stunde.</span></div>
            <div><strong>Verbrauch</strong><span>Geschätzte kWh aus der Differenz der Zählerstände innerhalb dieser Stunde.</span></div>
            <div><strong>Ø aWATTar</strong><span>Stündlicher aWATTar-Arbeitspreis ohne Netzgebühr.</span></div>
            <div><strong>Strom</strong><span>Verbrauch × aWATTar-Preis.</span></div>
            <div><strong>Netzgeb.</strong><span>Verbrauch × eingestellte Netzgebühr.</span></div>
            <div><strong>Summe</strong><span>Strom plus Netzgebühr; dieser Wert fließt in den Monatsbetrag.</span></div>
            <div><strong>Gewichtung</strong><span>Relative Balkenlänge im Vergleich zur teuersten Stunde des Monats.</span></div>
          </div>
          {{if .Detail.HasHours}}
            <div class="table-wrap">
              <table aria-label="Stundenwerte Parkplatznutzung">
                <thead>
                  <tr>
                    <th title="Beginn der Abrechnungsstunde; jede Zeile umfasst diese Stunde.">Stunde</th>
                    <th class="num" title="Geschätzte kWh aus der Differenz der Zählerstände innerhalb dieser Stunde.">Verbrauch</th>
                    <th class="num" title="Stündlicher aWATTar-Arbeitspreis ohne Netzgebühr.">Ø aWATTar</th>
                    <th class="num" title="Verbrauch × aWATTar-Preis.">Strom</th>
                    <th class="num" title="Verbrauch × eingestellte Netzgebühr.">Netzgeb.</th>
                    <th class="num" title="Strom plus Netzgebühr; dieser Wert fließt in den Monatsbetrag.">Summe</th>
                    <th title="Relative Balkenlänge im Vergleich zur teuersten Stunde des Monats.">Gewichtung</th>
                  </tr>
                </thead>
                <tbody>
                  {{range .Detail.Hours}}
                    <tr>
                      <td title="{{.AtTitle}}">{{.AtLabel}}</td>
                      <td class="num" title="{{.KWhTitle}}">{{.KWh}}</td>
                      <td class="num" title="{{.AverageAwattarTitle}}">{{.AverageAwattar}}</td>
                      <td class="num" title="{{.EnergyCostTitle}}">{{.EnergyCost}}</td>
                      <td class="num" title="{{.GridCostTitle}}">{{.GridCost}}</td>
                      <td class="num amount" title="{{.TotalCostTitle}}">{{.TotalCost}}</td>
                      <td class="bar-cell" title="{{.WeightTitle}}"><div class="bar"><span style="width: {{.ChartPercent}}%;"></span></div></td>
                    </tr>
                  {{end}}
                </tbody>
              </table>
            </div>
          {{else}}
            <p class="empty">Für diesen Monat sind noch keine Stundenwerte gespeichert.</p>
          {{end}}
        </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "settingsHub"}}
{{template "appOpen" .}}
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><span>Einstellungen</span></span>
      </div>
      <section class="page">
        <div>
          <h1>Einstellungen</h1>
          <p class="lede">Persönliche Einstellungen und Verwaltungsbereiche für {{.Tenant.Address}}.</p>
        </div>
        <div class="home-grid">
          <section class="panel">
            <div class="kicker">Konto</div>
            <div class="quick-list">
              <a class="quick-row" href="/app/settings/profile">
                <svg viewBox="0 0 24 24"><path d="M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8z"/><path d="M4.5 21a7.5 7.5 0 0 1 15 0"/></svg>
                <div><h3>Profil</h3><p>{{.Email}} · {{.Role}}</p></div>
                <span class="quick-arrow">›</span>
              </a>
              <a class="quick-row" href="/app/settings/notifications">
                <svg viewBox="0 0 24 24"><path d="M18 8a6 6 0 1 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9"/><path d="M10 21h4"/></svg>
                <div><h3>Benachrichtigungen</h3><p>E-Mail-Ereignisse pro Bereich steuern.</p></div>
                <span class="quick-arrow">›</span>
              </a>
            </div>
          </section>
          <section class="panel">
            <div class="kicker">Verwaltung</div>
            {{if or .CanManageUsers .CanManageBuilding .CanManageDocuments .IsAdmin}}
              <div class="quick-list">
                {{if .CanManageBuilding}}<a class="quick-row" href="/app/settings/building">
                  <svg viewBox="0 0 24 24"><path d="M4 21V8l8-5 8 5v13"/><path d="M9 21v-7h6v7"/><path d="M8 10h.01M16 10h.01"/></svg>
                  <div><h3>Gebäude</h3><p>Adresse, Kontakt, Hero-Bild und Einheiten verwalten.</p></div>
                  <span class="quick-arrow">›</span>
                </a>{{end}}
                {{if .CanManageUsers}}
                <a class="quick-row" href="/app/settings/users">
                  <svg viewBox="0 0 24 24"><path d="M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"/><path d="M3.5 20a5 5 0 0 1 10 0"/><path d="M16 11.5a2.5 2.5 0 1 0 0-5"/><path d="M17 15a4 4 0 0 1 3.5 4"/></svg>
                  <div><h3>Benutzer &amp; Rechte</h3><p>Einladungen, Rollen und Zugriff der Hausgemeinschaft verwalten.</p></div>
                  <span class="quick-arrow">›</span>
                </a>
                {{end}}
                {{if .CanManageDocuments}}
                <a class="quick-row" href="/app/dokumente">
                  <svg viewBox="0 0 24 24"><path d="M7 3h7l3 3v15H7z"/><path d="M14 3v4h4"/><path d="M9 13h6M9 17h6"/></svg>
                  <div><h3>Dokumente</h3><p>Unterlagen hochladen, kategorisieren und Sichtbarkeit setzen.</p></div>
                  <span class="quick-arrow">›</span>
                </a>
                {{end}}
                {{if .CanViewAudit}}
                <a class="quick-row" href="/app/audit">
                  <svg viewBox="0 0 24 24"><path d="M5 4h14v16H5z"/><path d="M8 8h8M8 12h8M8 16h5"/></svg>
                  <div><h3>Audit-Log</h3><p>Sensible Aktionen und Änderungen im Portal nachvollziehen.</p></div>
                  <span class="quick-arrow">›</span>
                </a>
                {{end}}
                {{if .IsAdmin}}<a class="quick-row" href="/app/parking/settings">
                  <svg viewBox="0 0 24 24"><path d="M5 16h14"/><path d="m7 16 1.5-5h7L17 16"/><path d="M7 16v3M17 16v3"/><path d="M7 19h1M16 19h1"/></svg>
                  <div><h3>Parkplatz-Abrechnung</h3><p>Netzgebühr und Abrechnungswerte für die private Parkplatznutzung.</p></div>
                  <span class="quick-arrow">›</span>
                </a>{{end}}
              </div>
            {{else}}
              <p class="empty">Verwaltungsbereiche sind nur für berechtigte Personen sichtbar.</p>
            {{end}}
          </section>
        </div>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "auditLog"}}
{{template "appOpen" .}}
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M5 4h14v16H5z"/><path d="M8 8h8M8 12h8M8 16h5"/></svg><span>/</span><span>Audit-Log</span></span>
        <div class="page-actions"><a class="button" href="/app/settings">Einstellungen</a></div>
      </div>
      <section class="page wide">
        <div>
          <h1>Audit-Log</h1>
          <p class="lede">Sensible Aktionen im Portal, begrenzt auf {{.Tenant.Address}}.</p>
        </div>
        <section class="panel accounting">
          <form class="filter-form audit-filter" method="get" action="/app/audit">
            <label for="audit-action">Aktion
              <select id="audit-action" name="action">
                {{range .ActionOptions}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>{{end}}
              </select>
            </label>
            <label for="audit-search">Suche
              <input id="audit-search" type="search" name="q" value="{{.SearchQuery}}" placeholder="Person, Ziel oder Aktion">
            </label>
            <button class="button" type="submit">Filtern</button>
          </form>
          {{if .HasEvents}}
            <div class="table-wrap">
              <table aria-label="Audit-Log">
                <thead>
                  <tr>
                    <th>Zeitpunkt</th>
                    <th>Aktion</th>
                    <th>Wer</th>
                    <th>Ziel</th>
                    <th>Details</th>
                  </tr>
                </thead>
                <tbody>
                  {{range .Events}}
                    <tr>
                      <td class="month-cell"><strong>{{.At}}</strong></td>
                      <td><span class="pill">{{.ActionText}}</span><span class="mini">{{.Summary}}</span></td>
                      <td><strong>{{.Actor}}</strong>{{if .ActorRole}}<span class="mini">{{.ActorRole}}</span>{{end}}</td>
                      <td>{{if .Target}}<strong>{{.Target}}</strong>{{else}}<span class="mini">-</span>{{end}}</td>
                      <td>
                        {{if .HasDetails}}
                          <div class="chips">{{range .Details}}<span class="chip"><strong>{{.Key}}:</strong> {{.Value}}</span>{{end}}</div>
                        {{else}}
                          <span class="mini">Keine weiteren Details</span>
                        {{end}}
                      </td>
                    </tr>
                  {{end}}
                </tbody>
              </table>
            </div>
          {{else}}
            {{template "emptyState" .EventsEmpty}}
          {{end}}
        </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "buildingSettings"}}
{{template "appOpen" .}}
    <style>
      .building .building-grid { display: grid; grid-template-columns: minmax(0,1.15fr) minmax(320px,.85fr); gap: 22px; align-items: start; }
      .building .settings-card { max-width: none; }
      .building .meta-form { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; }
      .building .meta-form .full, .building .unit-form .full { grid-column: 1 / -1; }
      .building .meta-form .f-actions { grid-column: 1 / -1; display: flex; justify-content: flex-end; }
      .building textarea { min-height: 92px; }
      .building .hero-preview { width: 100%; aspect-ratio: 16 / 9; object-fit: cover; border: 1px solid var(--line); border-radius: 8px; background: var(--panel-soft); }
      .building .hero-form { display: grid; gap: 12px; }
      .building .unit-panel { display: grid; gap: 18px; }
      .building .unit-add, .building .unit-editor { border: 1px solid var(--line); border-radius: 8px; padding: 16px; background: var(--panel-soft); }
      .building .unit-list { display: grid; gap: 12px; }
      .building .unit-form { display: grid; grid-template-columns: repeat(12,minmax(0,1fr)); gap: 10px; align-items: end; }
      .building .unit-form .f-label { grid-column: span 4; }
      .building .unit-form .f-share { grid-column: span 3; }
      .building .unit-form .f-owners, .building .unit-form .f-renters { grid-column: span 6; }
      .building .unit-form .f-actions { grid-column: span 5; display: flex; gap: 8px; align-items: center; justify-content: flex-end; flex-wrap: wrap; }
      .building .unit-delete { display: inline; margin: 0; }
      .building .unit-summary { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; margin-bottom: 12px; }
      .building .unit-summary strong { font-family: Spectral, serif; font-size: 20px; }
      @media (max-width: 960px) { .building .building-grid { grid-template-columns: 1fr; } }
      @media (max-width: 760px) {
        .building .meta-form, .building .unit-form { grid-template-columns: 1fr; }
        .building .unit-form .f-label, .building .unit-form .f-share, .building .unit-form .f-owners, .building .unit-form .f-renters, .building .unit-form .f-actions { grid-column: 1 / -1; }
        .building .meta-form .f-actions, .building .unit-form .f-actions { justify-content: stretch; }
        .building .unit-form .f-actions .button { flex: 1 1 auto; }
      }
    </style>
    <main class="app-main building">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Gebäude</span></span>
        <div class="page-actions"><a class="button" href="/app/settings">Zurück zu Einstellungen</a></div>
      </div>
      <section class="page wide">
        <div>
          <h1>Gebäude</h1>
          <p class="lede">Stammdaten, Kontaktblock, Titelbild und Einheiten für {{.Tenant.Address}}.</p>
        </div>
        <div class="building-grid">
          <section class="panel settings-card">
            <div>
              <h2>Stammdaten</h2>
              <p class="muted">Diese Angaben überschreiben die Umgebungswerte für diesen Tenant.</p>
            </div>
            {{if .BuildingMsg}}<p class="flash {{if .BuildingOK}}ok{{end}}">{{.BuildingMsg}}</p>{{end}}
            <form class="meta-form" method="post" action="/app/settings/building">
              <label class="full" for="building-name">Name
                <input id="building-name" type="text" name="name" value="{{.Tenant.Name}}" maxlength="160" required>
              </label>
              <label class="full" for="building-address">Adresse
                <textarea id="building-address" name="address" maxlength="500" required>{{.Tenant.Address}}</textarea>
              </label>
              <label for="contact-name">Verwalter Kontakt
                <input id="contact-name" type="text" name="contact_name" value="{{.Tenant.ContactName}}" maxlength="160" placeholder="Name oder Firma">
              </label>
              <label for="contact-email">Kontakt-E-Mail
                <input id="contact-email" type="email" name="contact_email" value="{{.Tenant.ContactEmail}}" maxlength="160" autocomplete="email">
              </label>
              <label for="contact-phone">Kontakt-Telefon
                <input id="contact-phone" type="tel" name="contact_phone" value="{{.Tenant.ContactPhone}}" maxlength="80" autocomplete="tel">
              </label>
              <label for="emergency-name">Notdienst
                <input id="emergency-name" type="text" name="emergency_name" value="{{.Tenant.EmergencyName}}" maxlength="160" placeholder="Notdienst">
              </label>
              <label for="emergency-phone">Notdienst-Telefon
                <input id="emergency-phone" type="tel" name="emergency_phone" value="{{.Tenant.EmergencyPhone}}" maxlength="80" autocomplete="tel">
              </label>
              <label for="caretaker-name">Hausmeister
                <input id="caretaker-name" type="text" name="caretaker_name" value="{{.Tenant.CaretakerName}}" maxlength="160">
              </label>
              <label for="caretaker-email">Hausmeister-E-Mail
                <input id="caretaker-email" type="email" name="caretaker_email" value="{{.Tenant.CaretakerEmail}}" maxlength="160" autocomplete="email">
              </label>
              <label for="caretaker-phone">Hausmeister-Telefon
                <input id="caretaker-phone" type="tel" name="caretaker_phone" value="{{.Tenant.CaretakerPhone}}" maxlength="80" autocomplete="tel">
              </label>
              <div class="f-actions"><button class="button primary" type="submit">Stammdaten speichern</button></div>
            </form>
          </section>

          <aside class="panel settings-card">
            <div>
              <h2>Hero-Bild</h2>
              <p class="muted">Das Bild erscheint auf der Startseite und im App-Banner.</p>
            </div>
            <img class="hero-preview" src="{{.Tenant.HeroImageURL}}" alt="">
            {{if .HeroMsg}}<p class="flash {{if .HeroOK}}ok{{end}}">{{.HeroMsg}}</p>{{end}}
            <form class="hero-form" method="post" action="/app/settings/building/hero" enctype="multipart/form-data">
              <label for="hero-image">Bilddatei
                <span class="file-control"><input id="hero-image" type="file" name="hero_image" accept="image/jpeg,image/png,image/webp" required><span>Bild auswählen</span></span>
              </label>
              <button class="button primary" type="submit">Hero-Bild speichern</button>
              <span class="mini">JPG, PNG oder WebP bis 5 MB.</span>
            </form>
          </aside>
        </div>

        <section class="panel unit-panel">
          <div class="section-head">
            <div>
              <h2>Einheiten</h2>
              <p class="muted">Wohneinheiten, Miteigentumsanteile und Eigentümer/Mieter-Links je Tenant.</p>
            </div>
          </div>
          {{if .UnitMsg}}<p class="flash {{if .UnitOK}}ok{{end}}">{{.UnitMsg}}</p>{{end}}
          <div class="unit-add">
            <div class="unit-summary"><strong>Neue Einheit</strong><span class="pill">Anlegen</span></div>
            <form class="unit-form" method="post" action="/app/settings/building/units">
              <label class="f-label">Einheit
                <input type="text" name="label" maxlength="120" required placeholder="Top 1">
              </label>
              <label class="f-share">Miteigentumsanteil
                <input type="number" name="miteigentumsanteil" min="0" max="1000000" step="1" value="0" inputmode="numeric">
              </label>
              <label class="f-owners">Eigentümer E-Mails
                <input type="text" name="owner_emails" placeholder="name@example.com, zweite@example.com">
              </label>
              <label class="f-renters">Mieter E-Mails
                <input type="text" name="renter_emails" placeholder="name@example.com">
              </label>
              <div class="f-actions"><button class="button primary" type="submit">Einheit anlegen</button></div>
            </form>
          </div>
          {{if .Units}}
            <div class="unit-list">
              {{range .Units}}
                <article class="unit-editor">
                  <div class="unit-summary"><strong>{{.Label}}</strong><span class="pill">{{.Share}}</span></div>
                  <form class="unit-form" method="post" action="/app/settings/building/units">
                    <input type="hidden" name="orig_id" value="{{.ID}}">
                    <input type="hidden" name="id" value="{{.ID}}">
                    <label class="f-label">Einheit
                      <input type="text" name="label" value="{{.Label}}" maxlength="120" required>
                    </label>
                    <label class="f-share">Miteigentumsanteil
                      <input type="number" name="miteigentumsanteil" min="0" max="1000000" step="1" value="{{.ShareValue}}" inputmode="numeric">
                    </label>
                    <label class="f-owners">Eigentümer E-Mails
                      <input type="text" name="owner_emails" value="{{.OwnerEmails}}">
                    </label>
                    <label class="f-renters">Mieter E-Mails
                      <input type="text" name="renter_emails" value="{{.RenterEmails}}">
                    </label>
                    <div class="f-actions">
                      <button class="button primary" type="submit">Speichern</button>
                    </div>
                  </form>
                  <form class="unit-delete" method="post" action="/app/settings/building/units/delete">
                    <input type="hidden" name="id" value="{{.ID}}">
                    <button class="button small ghost" type="submit" aria-label="{{.DeleteConfirmLabel}}">Entfernen</button>
                  </form>
                </article>
              {{end}}
            </div>
          {{else}}
            {{template "emptyState" .UnitsEmpty}}
          {{end}}
        </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "profileSettings"}}
{{template "appOpen" .}}
    <style>
      .profile .settings-card { max-width: 820px; display: grid; gap: 18px; }
      .profile .profile-flash { margin: 0; padding: 10px 13px; border-radius: 9px; font-size: 13.5px; font-weight: 600; border: 1px solid transparent; }
      .profile .profile-flash.ok { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.25); }
      .profile .profile-flash.warn { background: rgba(150,40,40,.08); color: #9a2b2b; border-color: rgba(150,40,40,.22); }
      .profile .profile-form { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; }
      .profile .profile-form .short { grid-column: span 1; }
      .profile .profile-form .full { grid-column: 1 / -1; }
      .profile .directory-check { grid-column: 1 / -1; min-height: 42px; display: flex; align-items: center; gap: 10px; color: var(--ink); font-size: 14px; font-weight: 700; letter-spacing: 0; text-transform: none; }
      .profile .directory-check input { width: auto; min-height: 0; }
      .profile .readonly-grid { display: grid; grid-template-columns: repeat(auto-fit,minmax(220px,1fr)); gap: 10px; }
      .profile .readonly-box { border: 1px solid var(--line); border-radius: 8px; padding: 13px; background: var(--panel-soft); display: grid; gap: 8px; }
      .profile .readonly-box strong { font-family: Spectral, serif; font-size: 18px; }
      .profile .chips { display: flex; flex-wrap: wrap; gap: 6px; }
      .profile .chip { display: inline-flex; align-items: center; border: 1px solid var(--line); background: var(--panel); color: #6f6a5c; border-radius: 8px; padding: 4px 10px; font-size: 12.5px; font-weight: 700; }
      .profile .unit-list { display: grid; gap: 8px; }
      .profile .unit-row { display: grid; grid-template-columns: minmax(0,1fr) auto; gap: 10px; align-items: center; border-top: 1px solid var(--line); padding-top: 8px; }
      .profile .unit-row:first-child { border-top: 0; padding-top: 0; }
      @media (max-width: 680px) { .profile .profile-form { grid-template-columns: 1fr; } .profile .profile-form .short { grid-column: 1 / -1; } }
    </style>
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Profil</span></span>
        <div class="page-actions"><a class="button" href="/app/settings">Zurück zu Einstellungen</a></div>
      </div>
      <section class="page profile">
        <div>
          <h1>Profil</h1>
          <p class="lede">{{.Email}}</p>
        </div>
        <section class="panel settings-card">
          {{if .ProfileMsg}}<p class="profile-flash{{if .ProfileOK}} ok{{else}} warn{{end}}">{{.ProfileMsg}}</p>{{end}}
          <form class="profile-form" method="post" action="/app/settings/profile">
            <label class="short" for="profile-title">Titel<input id="profile-title" name="title" value="{{.Profile.Title}}" maxlength="40" autocomplete="honorific-prefix"></label>
            <label class="short" for="profile-phone">Telefon optional<input id="profile-phone" name="phone" value="{{.Profile.Phone}}" maxlength="80" autocomplete="tel"></label>
            <label for="profile-first">Vorname<input id="profile-first" name="first_name" value="{{.Profile.FirstName}}" maxlength="120" autocomplete="given-name"></label>
            <label for="profile-last">Nachname<input id="profile-last" name="last_name" value="{{.Profile.LastName}}" maxlength="120" autocomplete="family-name"></label>
            <label class="directory-check" for="profile-directory"><input id="profile-directory" type="checkbox" name="directory_opt_in"{{if .Profile.DirectoryOptIn}} checked{{end}}>Im Kontakte-Verzeichnis anzeigen</label>
            <div class="full row-actions">
              <button class="button primary" type="submit">Speichern</button>
              <a class="button" href="/app/settings">Abbrechen</a>
            </div>
          </form>
          <div class="readonly-grid">
            <div class="readonly-box">
              <span class="field-label">Rolle</span>
              <strong>{{.Role}}</strong>
              <div class="chips">{{range .PermissionList}}<span class="chip">{{.}}</span>{{end}}</div>
            </div>
            <div class="readonly-box">
              <span class="field-label">Anmeldung</span>
              <div class="chips">{{range .AuthList}}<span class="chip">{{.}}</span>{{end}}</div>
            </div>
            <div class="readonly-box">
              <span class="field-label">Einheiten</span>
              {{if .HasUnits}}
                <div class="unit-list">
                  {{range .Units}}
                    <div class="unit-row"><span><strong>{{.Label}}</strong><span class="mini">{{.Relation}}</span></span><span class="chip">{{.Share}}</span></div>
                  {{end}}
                </div>
              {{else}}
                <p class="muted">Keine Einheit verknüpft.</p>
              {{end}}
            </div>
          </div>
        </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "notificationSettings"}}
{{template "appOpen" .}}
    <style>
      .notifications .panel { background: var(--panel); border: 1px solid var(--line); border-radius: 12px; padding: 22px; }
      .notifications .settings-card { max-width: 760px; display: grid; gap: 16px; }
      .notifications .notify-flash { margin: 0; padding: 10px 13px; border-radius: 9px; font-size: 13.5px; font-weight: 600; border: 1px solid transparent; }
      .notifications .notify-flash.ok { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.25); }
      .notifications .notify-flash.warn { background: rgba(150,40,40,.08); color: #9a2b2b; border-color: rgba(150,40,40,.22); }
      .notifications .toggle-list { display: grid; gap: 10px; }
      .notifications .toggle-row { display: grid; grid-template-columns: auto minmax(0,1fr); gap: 11px; align-items: start; border: 1px solid var(--line); border-radius: 9px; padding: 13px; background: var(--panel-soft); color: var(--ink); }
      .notifications .toggle-row input { width: 18px; height: 18px; margin-top: 2px; accent-color: var(--gold); }
      .notifications .toggle-row strong { display: block; font-size: 14px; }
      .notifications .toggle-row span { display: block; color: var(--muted); font-size: 13px; line-height: 1.45; margin-top: 2px; }
      .notifications .actions { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; }
    </style>
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Benachrichtigungen</span></span>
        <div class="page-actions"><a class="button" href="/app/settings">Zurück zu Einstellungen</a></div>
      </div>
      <section class="page notifications">
        <div>
          <h1>Benachrichtigungen</h1>
          <p class="lede">{{.Email}}</p>
        </div>
        <section class="panel settings-card">
          {{if .NotifyMsg}}<p class="notify-flash{{if .NotifyOK}} ok{{else}} warn{{end}}">{{.NotifyMsg}}</p>{{end}}
          <form class="form-grid" method="post" action="/app/settings/notifications">
            <div class="toggle-list full">
              <label class="toggle-row">
                <input type="checkbox" name="email_enabled" value="on"{{if .EmailNotificationsEnabled}} checked{{end}}>
                <span><strong>E-Mail-Benachrichtigungen</strong><span>Globale Zustellung für dieses Konto.</span></span>
              </label>
              {{range .NotificationEvents}}
              <label class="toggle-row">
                <input type="checkbox" name="events" value="{{.Key}}"{{if .Checked}} checked{{end}}>
                <span><strong>{{.Label}}</strong><span>{{.Description}}</span></span>
              </label>
              {{end}}
            </div>
            <div class="actions full">
              <button class="button primary" type="submit">Speichern</button>
              <a class="button" href="/app/settings">Abbrechen</a>
            </div>
          </form>
        </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "parkingSettings"}}
{{template "appOpen" .}}
    <main class="app-main">
      <div class="content-top">
        <span class="crumb"><svg viewBox="0 0 24 24"><path d="M3 11.5 12 4l9 7.5"/><path d="M5.5 10.5V20h13v-9.5"/><path d="M9.5 20v-5h5v5"/></svg><span>/</span><a href="/app/settings">Einstellungen</a><span>/</span><span>Parkplatz-Abrechnung</span></span>
        <div class="page-actions"><a class="button" href="/app/settings">Zurück zu Einstellungen</a><a class="button" href="/app/parking">Zur Parkplatznutzung</a></div>
      </div>
      <section class="page">
        <div>
          <h1>Parkplatz-Abrechnung</h1>
          <p class="lede">Abrechnungswerte für die private Parkplatznutzung.</p>
        </div>
        <section class="panel settings-card">
          <div>
            <h2>Netzgebühr</h2>
            <p class="muted">Kurzer Aufschlag je kWh für Netzbetreibergebühren und lokale Basisanteile. Dieser Wert fließt in die Monatsabrechnung ein.</p>
          </div>
          {{if .SettingsMsg}}<p class="flash {{if .SettingsOK}}ok{{end}}">{{.SettingsMsg}}</p>{{end}}
          <form class="form-grid" method="post" action="/app/parking/settings">
            <label for="grid_fee_eur_per_kwh">Netzgebühr je kWh</label>
            <input id="grid_fee_eur_per_kwh" type="text" inputmode="decimal" name="grid_fee_eur_per_kwh" value="{{.Accounting.GridFeeValue}}" autocomplete="off">
            <button class="button primary" type="submit">Speichern</button>
            {{if .Accounting.LastSampleLabel}}<span class="mini">Letzter Zählerwert: {{.Accounting.LastSampleLabel}}</span>{{end}}
          </form>
        </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}

{{define "userSettings"}}
{{template "appOpen" .}}
    <style>
      .users .panel { background: var(--panel); border: 1px solid var(--line); border-radius: 12px; padding: 22px; }
      .users .stack { display: grid; gap: 14px; }
      .users .panel-head { display: flex; align-items: baseline; justify-content: space-between; gap: 16px; border-bottom: 2px solid var(--ink); padding-bottom: 10px; margin-bottom: 14px; }
      .users .panel-head .kicker { border: 0; padding: 0; margin: 0; font-size: 12px; font-weight: 700; letter-spacing: .14em; text-transform: uppercase; color: var(--gold-ink); }
      .users .count { color: var(--soft); font-size: 12px; font-weight: 700; letter-spacing: .04em; font-variant-numeric: tabular-nums; }
      .users .muted { color: var(--muted); line-height: 1.55; }
      .users .roster-intro { margin: 2px 0 4px; }
      .users .disclosure { border: 1px solid var(--line); border-radius: 11px; background: var(--panel-soft); }
      .users .disclosure > summary { list-style: none; cursor: pointer; display: flex; align-items: center; gap: 11px; padding: 13px 16px; font-weight: 700; font-size: 14px; color: var(--ink); user-select: none; border-radius: 10px; }
      .users .disclosure > summary::-webkit-details-marker { display: none; }
      .users .disclosure > summary:hover { color: var(--gold-ink); }
      .users .disclosure > summary:focus-visible { outline: 2px solid var(--gold); outline-offset: -2px; }
      .users .disclosure[open] > summary { border-radius: 10px 10px 0 0; }
      .users .invite-plus { flex: 0 0 auto; width: 22px; height: 22px; border-radius: 6px; display: grid; place-items: center; background: var(--ink); color: #fff; }
      .users .summary-sub { margin-left: auto; font-weight: 600; font-size: 12.5px; color: var(--soft); }
      .users .disclosure-body { padding: 4px 16px 18px; }
      .users .invite-form { display: grid; grid-template-columns: repeat(12, 1fr); gap: 10px; }
      .users .invite-form .f-titel { grid-column: span 2; }
      .users .invite-form .f-vorname { grid-column: span 3; }
      .users .invite-form .f-nachname { grid-column: span 3; }
      .users .invite-form .f-email { grid-column: span 4; }
      .users .invite-form .f-role { grid-column: span 5; }
      .users .invite-form .f-submit { grid-column: span 7; }
      .users .invite-form .f-permissions, .users .dlg-form .f-permissions { grid-column: 1 / -1; }
      .users input, .users select { width: 100%; border: 1px solid #e2dac9; border-radius: 9px; padding: 12px; font: inherit; background: #fffefb; color: var(--ink); }
      .users .permission-fieldset { border: 1px solid var(--line); border-radius: 9px; padding: 12px; background: #fffefb; display: grid; gap: 10px; }
      .users .permission-fieldset legend { padding: 0 6px; color: var(--gold-ink); font-size: 11px; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; }
      .users .preset-label { display: inline-flex; width: fit-content; min-height: 24px; align-items: center; border: 1px solid rgba(200,153,63,.28); border-radius: 999px; padding: 3px 9px; background: rgba(200,153,63,.12); color: #8a6a1f; font-size: 11.5px; font-weight: 800; }
      .users .permission-grid { display: grid; grid-template-columns: repeat(auto-fit,minmax(220px,1fr)); gap: 8px; }
      .users .permission-check { display: grid; grid-template-columns: auto minmax(0,1fr); gap: 9px; align-items: start; border: 1px solid var(--line); border-radius: 8px; padding: 10px; color: var(--ink); background: var(--panel-soft); text-transform: none; letter-spacing: 0; font-size: 13px; font-weight: 600; }
      .users .permission-check input, .users .dlg-form .permission-check input { width: auto; min-height: 0; margin: 2px 0 0; accent-color: var(--gold); grid-row: 1 / span 2; }
      .users .permission-check span { display: block; color: var(--muted); font-size: 12px; font-weight: 500; line-height: 1.35; margin-top: 3px; grid-column: 2; }
      .users .invite-form button { border: 1px solid var(--ink); background: var(--ink); border-radius: 10px; color: #fff; min-height: 44px; padding: 10px 13px; font: inherit; font-weight: 700; cursor: pointer; }
      .users .invite-form button:hover { background: #2c3329; }
      .users .invite-flash { margin: 0 0 12px; padding: 10px 13px; border-radius: 9px; font-size: 13.5px; font-weight: 600; border: 1px solid transparent; }
      .users .invite-flash.ok { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.25); }
      .users .invite-flash.warn { background: rgba(200,153,63,.14); color: #93701d; border-color: rgba(200,153,63,.3); }
      .users .table-wrap { overflow: visible; }
      .users table { width: 100%; border-collapse: collapse; font-size: 15px; }
      .users thead th { color: var(--gold-ink); font-size: 11px; font-weight: 800; text-transform: uppercase; letter-spacing: .06em; text-align: left; padding: 4px 14px 12px; border-bottom: 2px solid var(--line); white-space: nowrap; }
      .users tbody td { padding: 15px 14px; border-bottom: 1px solid var(--line); vertical-align: middle; }
      .users tbody tr:last-child td { border-bottom: 0; }
      .users tbody tr { transition: background .12s ease; }
      .users tbody tr:hover { background: #faf6ec; }
      .users th:first-child, .users td:first-child { padding-left: 4px; }
      .users th:last-child, .users td:last-child { padding-right: 4px; }
      .users .person { display: flex; align-items: center; gap: 13px; min-width: 220px; }
      .users .avatar { flex: 0 0 auto; width: 40px; height: 40px; border-radius: 50%; display: grid; place-items: center; font-size: 14px; font-weight: 700; color: var(--gold-ink); background: rgba(200,153,63,.15); border: 1px solid rgba(200,153,63,.32); }
      .users .person-name { font-weight: 600; line-height: 1.25; }
      .users .person-mail { color: var(--muted); font-size: 13px; margin-top: 2px; word-break: break-word; }
      .users .chips { display: flex; flex-wrap: wrap; gap: 6px; }
      .users .chip { display: inline-flex; align-items: center; gap: 6px; border: 1px solid var(--line); background: var(--panel-soft); color: #6f6a5c; border-radius: 8px; padding: 4px 10px; font-size: 12.5px; font-weight: 600; white-space: nowrap; }
      .users .chip.plain { color: var(--soft); }
      .users .role-caps { display: flex; flex-wrap: wrap; gap: 5px; margin-top: 6px; }
      .users .role-cap { display: inline-flex; align-items: center; min-height: 22px; border: 1px solid var(--line); border-radius: 999px; padding: 2px 8px; color: var(--muted); background: var(--panel-soft); font-size: 11.5px; font-weight: 700; white-space: nowrap; }
      .users .pill { display: inline-flex; align-items: center; gap: 7px; border-radius: 999px; min-height: 28px; padding: 4px 12px; font-size: 13px; font-weight: 700; white-space: nowrap; border: 1px solid transparent; }
      .users .pill .dot { width: 7px; height: 7px; border-radius: 50%; background: currentColor; opacity: .9; }
      .users .pill.role-admin { background: rgba(200,153,63,.16); color: #8a6a1f; border-color: rgba(200,153,63,.28); }
      .users .pill.role-manager { background: rgba(32,37,31,.08); color: var(--ink); border-color: rgba(32,37,31,.15); }
      .users .pill.role-owner { background: rgba(47,107,74,.12); color: var(--leaf); border-color: rgba(47,107,74,.22); }
      .users .pill.role-renter { background: rgba(76,103,138,.11); color: #365475; border-color: rgba(76,103,138,.22); }
      .users .pill.role-resident { background: rgba(47,107,74,.11); color: var(--leaf); border-color: rgba(47,107,74,.2); }
      .users .pill.role-beirat { background: rgba(32,37,31,.06); color: #4b4f45; border-color: rgba(32,37,31,.12); }
      .users .pill.status-active { background: rgba(47,107,74,.12); color: var(--leaf); }
      .users .pill.status-pending { background: rgba(200,153,63,.14); color: #93701d; }
      .users .last-seen { display: block; color: var(--soft); font-size: 11.5px; margin-top: 5px; white-space: nowrap; }
      .users th.col-role, .users td.col-role, .users th.col-status, .users td.col-status { white-space: nowrap; }
      .users .th-label { display: inline-flex; align-items: center; gap: 6px; }
      .users .info { position: relative; display: inline-flex; }
      .users .info-btn { width: 17px; height: 17px; border-radius: 50%; border: 1px solid var(--gold-ink); background: transparent; color: var(--gold-ink); display: grid; place-items: center; padding: 0; cursor: help; }
      .users .info-btn:hover, .users .info-btn:focus-visible { background: var(--gold-ink); color: #fff; outline: none; }
      .users .info-btn:focus-visible { box-shadow: 0 0 0 2px rgba(200,153,63,.4); }
      .users .popup { position: absolute; top: calc(100% + 11px); left: -12px; width: min(480px, 88vw); background: var(--panel); border: 1px solid var(--line); border-radius: 14px; box-shadow: 0 20px 46px rgba(32,37,31,.17), 0 3px 9px rgba(32,37,31,.05); padding: 16px 19px 18px; z-index: 8; opacity: 0; visibility: hidden; transform: translateY(-6px); transition: opacity .16s ease, transform .16s ease; text-transform: none; letter-spacing: normal; }
      .users .popup::before { content: ""; position: absolute; top: -6px; left: 19px; width: 12px; height: 12px; background: var(--panel); border-left: 1px solid var(--line); border-top: 1px solid var(--line); border-radius: 3px 0 0 0; transform: rotate(45deg); }
      .users .popup::after { content: ""; position: absolute; top: -15px; left: 0; right: 0; height: 15px; }
      .users .info:hover .popup, .users .info:focus-within .popup { opacity: 1; visibility: visible; transform: translateY(0); }
      .users .popup-title { display: block; font-family: Spectral, serif; font-weight: 600; font-size: 15px; color: var(--ink); padding-bottom: 11px; border-bottom: 1px solid var(--line); }
      .users .popup-grid { display: grid; grid-template-columns: minmax(0,1fr) minmax(0,1fr); column-gap: 26px; }
      .users .popup .permission { display: block; padding: 12px 0; }
      .users .popup-grid .permission:nth-child(1), .users .popup-grid .permission:nth-child(2) { padding-top: 14px; }
      .users .popup-grid .permission:nth-child(n+3) { border-top: 1px solid var(--line); }
      .users .popup .permission strong { display: block; font-family: Spectral, serif; font-weight: 600; font-size: 13.5px; color: var(--ink); margin-bottom: 3px; }
      .users .popup .permission .muted { display: block; font-size: 12.5px; font-weight: 400; color: var(--muted); line-height: 1.5; overflow-wrap: break-word; }
      .users .rdot { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 9px; vertical-align: middle; }
      .users .rdot.admin { background: var(--gold); }
      .users .rdot.manager { background: var(--ink); }
      .users .rdot.owner { background: var(--leaf); }
      .users .rdot.renter { background: #365475; }
      .users .rdot.resident { background: var(--leaf); }
      .users .rdot.beirat { background: #8a8d80; }
      .users .rdot.right { background: var(--gold-light); box-shadow: inset 0 0 0 1px var(--gold); }
      .users .col-actions { width: 44px; }
      .users td.col-actions { text-align: right; }
      .users .row-edit { border: 1px solid transparent; background: transparent; border-radius: 8px; width: 32px; height: 32px; display: inline-grid; place-items: center; color: var(--soft); cursor: pointer; padding: 0; }
      .users .row-edit:hover { border-color: var(--line); background: var(--panel-soft); color: var(--gold-ink); }
      .users .row-edit svg { stroke: currentColor; fill: none; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
      .users .edit-dialog { position: relative; width: min(440px, 92vw); border: 1px solid var(--line); border-radius: 14px; padding: 22px; background: var(--panel); color: var(--ink); box-shadow: 0 30px 80px rgba(32,37,31,.32); }
      .users .edit-dialog::backdrop { background: rgba(32,37,31,.42); }
      .users .edit-dialog h2 { margin: 0 0 4px; font-family: Spectral, serif; font-weight: 600; font-size: 19px; }
      .users .edit-dialog .dlg-sub { color: var(--muted); font-size: 13px; margin: 0 0 16px; word-break: break-word; }
      .users .dlg-x { position: absolute; top: 12px; right: 12px; }
      .users .dlg-x button { border: 0; background: transparent; font-size: 22px; line-height: 1; color: var(--soft); cursor: pointer; padding: 2px 6px; }
      .users .dlg-x button:hover { color: var(--ink); }
      .users .dlg-form { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
      .users .dlg-form input, .users .dlg-form select, .users .dlg-form button { grid-column: 1 / -1; }
      .users .dlg-form .f-vorname, .users .dlg-form .f-nachname { grid-column: span 1; }
      .users .dlg-form button { border: 1px solid var(--ink); background: var(--ink); color: #fff; border-radius: 10px; min-height: 44px; padding: 10px 13px; font: inherit; font-weight: 700; cursor: pointer; }
      .users .dlg-form button:hover { background: #2c3329; }
      .users .dlg-delete { margin-top: 16px; padding-top: 14px; border-top: 1px solid var(--line); display: flex; justify-content: space-between; align-items: center; gap: 12px; }
      .users .dlg-delete span { color: var(--muted); font-size: 12.5px; }
      .users .dlg-delete .danger { border: 1px solid rgba(150,40,40,.32); background: rgba(150,40,40,.07); color: #9a2b2b; border-radius: 10px; min-height: 40px; padding: 8px 15px; font: inherit; font-weight: 700; cursor: pointer; }
      .users .dlg-delete .danger:hover { background: rgba(150,40,40,.14); }
      @media (max-width: 760px) {
        .users .table-wrap { overflow: visible; }
        .users table, .users thead, .users tbody, .users tr, .users td { display: block; width: 100%; }
        .users thead { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); }
        .users tbody tr { border: 1px solid var(--line); border-radius: 11px; padding: 14px; margin-bottom: 12px; background: var(--panel); }
        .users tbody tr:hover { background: var(--panel); }
        .users tbody td { border: 0; padding: 0; }
        .users tbody td.col-person { margin-bottom: 12px; }
        .users tbody td[data-label]:not(.col-person) { display: grid; grid-template-columns: 96px 1fr; align-items: start; gap: 10px; padding: 7px 0; border-top: 1px dashed var(--line); }
        .users tbody td[data-label]:not(.col-person)::before { content: attr(data-label); color: var(--gold-ink); font-size: 10.5px; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; padding-top: 5px; }
        .users .invite-form > * { grid-column: 1 / -1 !important; }
      }
      @media (max-width: 560px) { .users .popup-grid { grid-template-columns: 1fr; } .users .popup-grid .permission:nth-child(n+2) { border-top: 1px solid var(--line); padding-top: 12px; } }
    </style>
    <script src="/assets/users.js" defer></script>
    <main class="app-main">
      <div class="content-top"><span class="crumb"><svg viewBox="0 0 24 24"><path d="M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"/><path d="M3.5 20a5 5 0 0 1 10 0"/><path d="M16 11.5a2.5 2.5 0 1 0 0-5"/><path d="M17 15a4 4 0 0 1 3.5 4"/></svg>Benutzer &amp; Rechte</span></div>
      <section class="page users">
        <div>
          <h1>Benutzer &amp; Rechte</h1>
          <p class="lede">Lokale Verwaltung der eingeladenen E-Mail-Adressen und ihrer Rollen.</p>
        </div>
    <section class="panel stack">
      <div class="panel-head">
        <span class="kicker">Zugänge</span>
        <span class="count">{{len .Users}} {{if eq (len .Users) 1}}Person{{else}}Personen{{end}}</span>
      </div>

      <details class="disclosure invite-bar"{{if .InviteMsg}} open{{end}}>
        <summary>
          <span class="invite-plus"><svg viewBox="0 0 24 24" width="13" height="13" aria-hidden="true"><path d="M12 5.5v13M5.5 12h13" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"/></svg></span>
          Person einladen
          <span class="summary-sub">Speichert &amp; lädt per E-Mail ein</span>
        </summary>
        <div class="disclosure-body">
          {{if .InviteMsg}}<p class="invite-flash{{if .InviteOK}} ok{{else}} warn{{end}}">{{.InviteMsg}}</p>{{end}}
          <p class="muted" style="margin-bottom:12px">Die eingeladene Person wird gespeichert und erhält eine E-Mail mit dem Anmelde-Link. Sie kann sich danach mit dieser Adresse anmelden.</p>
          <form class="invite-form" method="post" action="/app/settings/users">
            <input class="f-titel" type="text" name="title" placeholder="Titel" aria-label="Titel">
            <input class="f-vorname" type="text" name="first_name" placeholder="Vorname" aria-label="Vorname">
            <input class="f-nachname" type="text" name="last_name" placeholder="Nachname" aria-label="Nachname">
            <input class="f-email" type="email" name="email" placeholder="name@example.com" aria-label="E-Mail-Adresse" autocomplete="email" required>
            <select class="f-role" name="role" aria-label="Rolle">
              <option value="Mieter" data-preset-label="Standardzugriff" data-preset-permissions="">Mieter</option>
              <option value="Eigentümer" data-preset-label="Eigentümerzugriff" data-preset-permissions="">Eigentümer</option>
              <option value="Beirat" data-preset-label="Beiratszugriff" data-preset-permissions="">Beirat</option>
              <option value="Verwalter" data-preset-label="Verwalterzugriff" data-preset-permissions="">Verwalter</option>
              {{if .IsAdmin}}<option value="Admin" data-preset-label="Adminzugriff" data-preset-permissions="parking">Admin</option>{{end}}
              <option value="Bewohner" data-preset-label="Bewohnerzugriff" data-preset-permissions="">Bewohner</option>
            </select>
            <fieldset class="permission-fieldset f-permissions">
              <legend>Sonderrechte</legend>
              <span class="preset-label" data-preset-label>Standardzugriff</span>
              <div class="permission-grid">
                <label class="permission-check"><input type="checkbox" name="permissions" value="parking" data-permission="parking"><strong>Parkplatznutzung</strong><span>Privater Bereich für Stellplatz- und Ladeabrechnung.</span></label>
              </div>
            </fieldset>
            <button class="f-submit" type="submit">Einladung senden</button>
          </form>
        </div>
      </details>

      <p class="muted roster-intro">Diese Liste kommt aus der Umgebungskonfiguration und den hier gespeicherten Einladungen.</p>

      {{if .HasUsers}}<div class="table-wrap">
        <table aria-label="Benutzerliste">
          <thead>
            <tr>
              <th class="col-person">Person</th>
              <th class="col-role">
                <span class="th-label">Rolle
                  <span class="info">
                    <button type="button" class="info-btn" aria-label="Rollen und Rechte erklärt" aria-describedby="role-help"><svg viewBox="0 0 16 16" width="10" height="10" aria-hidden="true"><circle cx="8" cy="3.5" r="1.15" fill="currentColor"/><rect x="6.9" y="6.3" width="2.2" height="6.3" rx="1.1" fill="currentColor"/></svg></button>
                    <span id="role-help" class="popup" role="tooltip">
                      <span class="popup-title">Rollen &amp; Rechte</span>
                      <span class="popup-grid">
                        <span class="permission"><strong><span class="rdot admin"></span>Admin</strong><span class="muted">Zugänge verwalten, Rollen setzen und Portalbereiche vorbereiten.</span></span>
                        <span class="permission"><strong><span class="rdot manager"></span>Verwalter</strong><span class="muted">Tenant-Verwaltung ohne Plattform- oder Parkplatzkonfiguration.</span></span>
                        <span class="permission"><strong><span class="rdot owner"></span>Eigentümer</strong><span class="muted">Bewohnerbereich plus Eigentümer-Dokumente und Abstimmungen.</span></span>
                        <span class="permission"><strong><span class="rdot renter"></span>Mieter</strong><span class="muted">Bewohnerbereich ohne Eigentümer-Abstimmungen.</span></span>
                        <span class="permission"><strong><span class="rdot beirat"></span>Beirat</strong><span class="muted">Bewohnerbereich plus lesende Übersicht.</span></span>
                        <span class="permission"><strong><span class="rdot right"></span>Parkplatznutzung</strong><span class="muted">Separates Sonderrecht für den privaten Parkplatzbereich.</span></span>
                      </span>
                    </span>
                  </span>
                </span>
              </th>
              <th>Rechte</th>
              <th>Anmeldung</th>
              <th class="col-status">Status</th>
              <th class="col-actions" aria-label="Aktionen"></th>
            </tr>
          </thead>
          <tbody>
            {{range .Users}}
            <tr>
              <td class="col-person" data-label="Person">
                <div class="person">
                  <span class="avatar">{{.Initials}}</span>
                  <div>
                    <div class="person-name">{{.DisplayName}}</div>
                    <div class="person-mail">{{.Email}}</div>
                    {{if .Phone}}<div class="person-mail">{{.Phone}}</div>{{end}}
                  </div>
                </div>
              </td>
              <td class="col-role" data-label="Rolle"><span class="pill {{.RoleClass}}"><span class="dot"></span>{{.Role}}</span><div class="role-caps">{{range .RoleCapabilities}}<span class="role-cap">{{.}}</span>{{end}}</div></td>
              <td data-label="Rechte"><div class="chips">{{range .PermissionList}}<span class="chip{{if eq . "Standard"}} plain{{end}}">{{.}}</span>{{end}}</div></td>
              <td data-label="Anmeldung"><div class="chips">{{range .AuthList}}<span class="chip">{{.}}</span>{{end}}</div></td>
              <td class="col-status" data-label="Status"><span class="pill {{if eq .Status "Aktiv"}}status-active{{else}}status-pending{{end}}"><span class="dot"></span>{{.Status}}</span>{{if .LastSeen}}<span class="last-seen">{{.LastSeen}}</span>{{end}}</td>
              <td class="col-actions" data-label="">
                {{if .Editable}}
                <button type="button" class="row-edit" data-edit="{{.Email}}" aria-label="Bearbeiten" aria-haspopup="dialog" aria-controls="edit-{{.Email}}"><svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true"><path d="M4 20h4L18.5 9.5a2 2 0 0 0-2.83-2.83L5 17.2z"/><path d="M13.5 6.5 17 10"/></svg></button>
                <dialog id="edit-{{.Email}}" class="edit-dialog" aria-labelledby="edit-title-{{.Email}}">
                  <div class="dlg-x"><form method="dialog"><button aria-label="Schließen">&times;</button></form></div>
                  <h2 id="edit-title-{{.Email}}">Zugang bearbeiten</h2>
                  <p class="dlg-sub">{{.Email}}</p>
                  <form method="post" action="/app/settings/users/edit" class="dlg-form">
                    <input type="hidden" name="orig_email" value="{{.Email}}">
                    <input class="f-titel" type="text" name="title" value="{{.Title}}" placeholder="Titel" aria-label="Titel">
                    <input class="f-vorname" type="text" name="first_name" value="{{.FirstName}}" placeholder="Vorname" aria-label="Vorname">
                    <input class="f-nachname" type="text" name="last_name" value="{{.LastName}}" placeholder="Nachname" aria-label="Nachname">
                    <input class="f-email" type="email" name="email" value="{{.Email}}" aria-label="E-Mail-Adresse" autocomplete="email" required>
                    <select class="f-role" name="role" aria-label="Rolle">
                      <option value="Mieter" data-preset-label="Standardzugriff" data-preset-permissions=""{{if eq .Role "Mieter"}} selected{{end}}>Mieter</option>
                      <option value="Eigentümer" data-preset-label="Eigentümerzugriff" data-preset-permissions=""{{if eq .Role "Eigentümer"}} selected{{end}}>Eigentümer</option>
                      <option value="Beirat" data-preset-label="Beiratszugriff" data-preset-permissions=""{{if eq .Role "Beirat"}} selected{{end}}>Beirat</option>
                      <option value="Verwalter" data-preset-label="Verwalterzugriff" data-preset-permissions=""{{if eq .Role "Verwalter"}} selected{{end}}>Verwalter</option>
                      {{if $.IsAdmin}}<option value="Admin" data-preset-label="Adminzugriff" data-preset-permissions="parking"{{if eq .Role "Admin"}} selected{{end}}>Admin</option>{{end}}
                      <option value="Bewohner" data-preset-label="Bewohnerzugriff" data-preset-permissions=""{{if eq .Role "Bewohner"}} selected{{end}}>Bewohner</option>
                    </select>
                    <fieldset class="permission-fieldset f-permissions">
                      <legend>Sonderrechte</legend>
                      <span class="preset-label" data-preset-label>Gespeicherte Rechte</span>
                      <div class="permission-grid">
                        <label class="permission-check"><input type="checkbox" name="permissions" value="parking" data-permission="parking"{{if .ParkingChecked}} checked{{end}}><strong>Parkplatznutzung</strong><span>Privater Bereich für Stellplatz- und Ladeabrechnung.</span></label>
                      </div>
                    </fieldset>
                    <button type="submit">Speichern</button>
                  </form>
                  <div class="dlg-delete">
                    <span>Dauerhaft entfernen</span>
                    <form method="post" action="/app/settings/users/delete">
                      <input type="hidden" name="email" value="{{.Email}}">
                      <button type="submit" class="danger">Löschen</button>
                    </form>
                  </div>
                </dialog>
                {{end}}
              </td>
            </tr>
            {{end}}
          </tbody>
        </table>
      </div>{{else}}
        {{template "emptyState" .UsersEmpty}}
      {{end}}
    </section>
      </section>
    </main>
{{template "appClose" .}}
{{end}}
`

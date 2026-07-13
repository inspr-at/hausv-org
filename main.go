package main

import (
	"context"
	"crypto/hmac"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/markus-barta/hausv-org/internal/auth"
	"github.com/markus-barta/hausv-org/internal/authz"
	"github.com/markus-barta/hausv-org/internal/config"
	"github.com/markus-barta/hausv-org/internal/homeassistant"
	appmail "github.com/markus-barta/hausv-org/internal/mail"
	"github.com/markus-barta/hausv-org/internal/view"
	"github.com/markus-barta/hausv-org/internal/web"
	"html/template"
	_ "image/png"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/markus-barta/hausv-org/internal/integrations"
	"github.com/markus-barta/hausv-org/internal/store"
	"github.com/markus-barta/hausv-org/internal/textutil"
	"github.com/markus-barta/hausv-org/internal/version"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// ── extracted to view ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
type (
	announcementFilterView  = view.AnnouncementFilterView
	announcementView        = view.AnnouncementView
	attachmentGroup         = view.AttachmentGroup
	attachmentView          = view.AttachmentView
	auditDetailView         = view.AuditDetailView
	auditEventView          = view.AuditEventView
	auditFilterChipView     = view.AuditFilterChipView
	auditStatsView          = view.AuditStatsView
	ballotOptionView        = view.BallotOptionView
	ballotResultCount       = view.BallotResultCount
	ballotResultSummary     = view.BallotResultSummary
	ballotView              = view.BallotView
	buildingUnitView        = view.BuildingUnitView
	contactCardView         = view.ContactCardView
	contactOptionView       = view.ContactOptionView
	dashboardDigestItem     = view.DashboardDigestItem
	documentCategoryView    = view.DocumentCategoryView
	documentVersionView     = view.DocumentVersionView
	documentView            = view.DocumentView
	emptyStateView          = view.EmptyStateView
	houseEventView          = view.HouseEventView
	issueBoardFilterView    = view.IssueBoardFilterView
	issueCommentView        = view.IssueCommentView
	issueView               = view.IssueView
	managedContactView      = view.ManagedContactView
	notificationEventOption = view.NotificationEventOption
	parkingAccountingView   = view.ParkingAccountingView
	parkingBalanceView      = view.ParkingBalanceView
	parkingHourView         = view.ParkingHourView
	parkingMonthDetailView  = view.ParkingMonthDetailView
	parkingMonthView        = view.ParkingMonthView
	parkingStatementView    = view.ParkingStatementView
	parkingTariffView       = view.ParkingTariffView
	profileUnitView         = view.ProfileUnitView
	selectOption            = view.SelectOption
	unitPaymentStatusView   = view.UnitPaymentStatusView
	userRow                 = view.UserRow
)

var announcementViewFrom = view.AnnouncementViewFrom
var attachmentViewFromRecord = view.AttachmentViewFromRecord
var auditActionLabel = view.AuditActionLabel
var auditActionOptions = view.AuditActionOptions
var auditActionTone = view.AuditActionTone
var auditDetailLabel = view.AuditDetailLabel
var auditEventViewFrom = view.AuditEventViewFrom
var auditTargetLabel = view.AuditTargetLabel
var auditTargetTypeLabel = view.AuditTargetTypeLabel
var auditToneLabel = view.AuditToneLabel
var authMethodsLabel = view.AuthMethodsLabel
var ballotWeightingLabel = view.BallotWeightingLabel
var ballotWinnerLabel = view.BallotWinnerLabel
var billableUnitCountLabel = view.BillableUnitCountLabel
var contactKindOptions = view.ContactKindOptions
var contactStatusLabel = view.ContactStatusLabel
var documentCategoryOptions = view.DocumentCategoryOptions
var documentSortOptions = view.DocumentSortOptions
var documentUnitAuditLabel = view.DocumentUnitAuditLabel
var documentUnitLabel = view.DocumentUnitLabel
var documentUnitOptions = view.DocumentUnitOptions
var documentVersionLabel = view.DocumentVersionLabel
var documentViewFrom = view.DocumentViewFrom
var documentVisibilityClass = view.DocumentVisibilityClass
var documentVisibilityLabel = view.DocumentVisibilityLabel
var documentVisibilityOptions = view.DocumentVisibilityOptions
var emptyState = view.EmptyState
var eventCategoryClass = view.EventCategoryClass
var eventViewFrom = view.EventViewFrom
var formatBallotReminder = view.FormatBallotReminder
var formatBallotResultWeight = view.FormatBallotResultWeight
var formatBallotSharePercent = view.FormatBallotSharePercent
var formatBallotShareWeight = view.FormatBallotShareWeight
var formatBallotWeight = view.FormatBallotWeight
var formatBillableUnitWeight = view.FormatBillableUnitWeight
var formatBytes = view.FormatBytes
var formatDateTimeIn = view.FormatDateTimeIn
var formatDecimal = view.FormatDecimal
var formatEUR = view.FormatEUR
var formatEURPerKWh = view.FormatEURPerKWh
var formatInputFloat = view.FormatInputFloat
var formatIssueEstimateAmount = view.FormatIssueEstimateAmount
var formatIssueEstimateInput = view.FormatIssueEstimateInput
var formatKWh = view.FormatKWh
var formatLocalDate = view.FormatLocalDate
var formatLocalDateTime = view.FormatLocalDateTime
var formatLocalDateTimeInput = view.FormatLocalDateTimeInput
var formatLocalShortDateTime = view.FormatLocalShortDateTime
var formatLocalTime = view.FormatLocalTime
var formatMiteigentumsanteil = view.FormatMiteigentumsanteil
var formatMonthLabel = view.FormatMonthLabel
var formatPPMPercent = view.FormatPPMPercent
var formatParkingTariffDate = view.FormatParkingTariffDate
var formatPeriodLabel = view.FormatPeriodLabel
var formatPreciseEUR = view.FormatPreciseEUR
var formatPreciseEURPerKWh = view.FormatPreciseEURPerKWh
var formatPreciseKWh = view.FormatPreciseKWh
var issueBoardFilterOptions = view.IssueBoardFilterOptions
var issueFilterOptions = view.IssueFilterOptions
var issueLocationLabel = view.IssueLocationLabel
var issueSelectOptions = view.IssueSelectOptions
var issueStatusClass = view.IssueStatusClass
var managedContactViewFrom = view.ManagedContactViewFrom
var normalizeTenantBrandIcon = view.NormalizeTenantBrandIcon
var notificationEventOptions = view.NotificationEventOptions
var paidLabel = view.PaidLabel
var parkingStatementTariffLabel = view.ParkingStatementTariffLabel
var permissionLabel = view.PermissionLabel
var roleClass = view.RoleClass
var tenantBrandIconLabel = view.TenantBrandIconLabel
var tenantBrandIconOptions = view.TenantBrandIconOptions
var togglePaidLabel = view.TogglePaidLabel
var unitBillableLabel = view.UnitBillableLabel
var unitCountLabel = view.UnitCountLabel
var unitPaymentRelationLabel = view.UnitPaymentRelationLabel
var unitPaymentStatusClass = view.UnitPaymentStatusClass
var unitPaymentStatusLabel = view.UnitPaymentStatusLabel
var unitPaymentStatusOptions = view.UnitPaymentStatusOptions
var unitPaymentStatusViewFromUnit = view.UnitPaymentStatusViewFromUnit
var unitTypeLabel = view.UnitTypeLabel
var unitTypeOptions = view.UnitTypeOptions

// ── extracted to view ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
var authMethodsLabelList = view.AuthMethodsLabelList
var permissionLabelList = view.PermissionLabelList

// ── extracted to view ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
var documentCanPreview = view.DocumentCanPreview
var documentFileKind = view.DocumentFileKind
var selectedDocumentSort = view.SelectedDocumentSort

// ── extracted to view ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
var documentCategories = view.DocumentCategories

// ── extracted to view ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
var notificationEventCatalog = view.NotificationEventCatalog
var unitPaymentStatusDetail = view.UnitPaymentStatusDetail

// ── extracted to view ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
var eventTimeRange = view.EventTimeRange
var germanMonthShort = view.GermanMonthShort
var issueCategories = view.IssueCategories
var issuePriorities = view.IssuePriorities
var issueStatuses = view.IssueStatuses

// ── extracted to view ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
var announcementUnread = view.AnnouncementUnread
var plainTextHTML = view.PlainTextHTML
var sameLocalDate = view.SameLocalDate

// ── extracted to auth ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
type (
	loginToken     = auth.LoginToken
	oidcFlow       = auth.OidcFlow
	oidcFlowStore  = auth.OidcFlowStore
	oidcLogin      = auth.OidcLogin
	oidcUserClaims = auth.OidcUserClaims
	session        = auth.Session
	sessionStore   = auth.SessionStore
	tokenStore     = auth.TokenStore
)

var newOIDCLogin = auth.NewOIDCLogin
var newSessionStore = auth.NewSessionStore
var pkceChallenge = auth.PkceChallenge
var randomToken = auth.RandomToken
var safeInternalRedirectPath = auth.SafeInternalRedirectPath

// ── extracted to mail ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
type (
	mailer             = appmail.Mailer
	portalNotification = appmail.PortalNotification
	smtpMailer         = appmail.SmtpMailer
)

var redactedEmail = appmail.RedactedEmail

// ── extracted to config ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
type (
	tenantConfig = config.TenantConfig
)

var env = config.Env
var isLocalHost = config.IsLocalHost
var loadLocalEnv = config.LoadLocalEnv
var normalizeHost = config.NormalizeHost
var parseAllowed = config.ParseAllowed
var parseBool = config.ParseBool
var parseDuration = config.ParseDuration
var parseHistoryStart = config.ParseHistoryStart
var parseTenants = config.ParseTenants
var parseUserProfiles = config.ParseUserProfiles
var sessionSecret = config.SessionSecret
var trimEnvQuotes = config.TrimEnvQuotes

// ── extracted to store ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
var normalizeTenantMemberships = store.NormalizeTenantMemberships
var tenantMembershipSlugs = store.TenantMembershipSlugs

// ── extracted to homeassistant ──────────────────────────────────────────────
type homeAssistantConfig = homeassistant.Config

// newHomeAssistantConfig keeps env parsing in main (the composition root); the
// package itself takes explicit values.
func newHomeAssistantConfig() homeAssistantConfig {
	return homeassistant.NewConfig(
		env("HA_BASE_URL", ""),
		os.Getenv("HA_TOKEN"),
		env("PARKING_METER_ENERGY_ENTITY", "sensor.kws_306wf_energy_meter_energy"),
		env("PARKING_POWER_ENTITY", "sensor.kws360_power"),
		env("PARKING_PRICE_ENTITY", "sensor.epex_spot_data_total_price"),
	)
}

type haState = homeassistant.EntityState
type haStatistic = homeassistant.Statistic
type haHistoryState = homeassistant.HistoryState

var (
	samplesFromStatistics = homeassistant.SamplesFromStatistics
	samplesFromHistory    = homeassistant.SamplesFromHistory
	parseHAFloat          = homeassistant.ParseFloat
)

// Capabilities now owned by authz; aliased so call sites read unchanged.
const (
	capabilityPlatformAdmin       = authz.CapabilityPlatformAdmin
	capabilityManageUsers         = authz.CapabilityManageUsers
	capabilityManageParking       = authz.CapabilityManageParking
	capabilityManageAnnouncements = authz.CapabilityManageAnnouncements
	capabilityManageDocuments     = authz.CapabilityManageDocuments
	capabilityManageIssues        = authz.CapabilityManageIssues
	capabilityManageVotes         = authz.CapabilityManageVotes
	capabilityManageBuilding      = authz.CapabilityManageBuilding
	capabilityOwnerDocuments      = authz.CapabilityOwnerDocuments
	capabilityVote                = authz.CapabilityVote
	capabilityOversight           = authz.CapabilityOversight
)

// ── extracted to authz ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
type (
	capability = authz.Capability
)

var canAssignUserRole = authz.CanAssignUserRole
var canCreateResidentIssue = authz.CanCreateResidentIssue
var canManageAnnouncements = authz.CanManageAnnouncements
var canManageContacts = authz.CanManageContacts
var canManageEvents = authz.CanManageEvents
var canManageHandovers = authz.CanManageHandovers
var canResidentTransition = authz.CanResidentTransition
var canServiceProviderTransition = authz.CanServiceProviderTransition
var canUseResidentAreas = authz.CanUseResidentAreas
var canViewAudit = authz.CanViewAudit
var hasCapability = authz.HasCapability
var isServiceProviderRole = authz.IsServiceProviderRole

// ── extracted to store ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
var defaultAuthMethods = store.DefaultAuthMethods

// ── extracted to store ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
type (
	inviteStore      = store.InviteStore
	inviteStoreData  = store.InviteStoreData
	tenantMembership = store.TenantMembership
	userProfile      = store.UserProfile
)

var initialLetter = store.InitialLetter
var newInviteStore = store.NewInviteStore
var normalizeAuthMethod = store.NormalizeAuthMethod
var normalizeAuthMethods = store.NormalizeAuthMethods
var normalizePermissions = store.NormalizePermissions
var normalizeTenants = store.NormalizeTenants

// Limits and document vocabulary now owned by the store; aliased for call sites.
const (
	defaultTenantHeroImageURL      = store.DefaultTenantHeroImageURL
	maxIssuePhotoBytes             = store.MaxIssuePhotoBytes
	maxIssueFormBytes              = store.MaxIssueFormBytes
	maxAttachmentBytes             = store.MaxAttachmentBytes
	maxIssueAttachmentCount        = store.MaxIssueAttachmentCount
	maxIssueAttachmentFormBytes    = store.MaxIssueAttachmentFormBytes
	attachmentPreviewMaxDimension  = store.AttachmentPreviewMaxDimension
	attachmentThumbMaxDimension    = store.AttachmentThumbMaxDimension
	maxTenantHeroBytes             = store.MaxTenantHeroBytes
	maxTenantHeroFormBytes         = store.MaxTenantHeroFormBytes
	maxDocumentBytes               = store.MaxDocumentBytes
	maxDocumentFormBytes           = store.MaxDocumentFormBytes
	documentCategoryProtocol       = store.DocumentCategoryProtocol
	documentCategoryBilling        = store.DocumentCategoryBilling
	documentCategoryRules          = store.DocumentCategoryRules
	documentCategoryContract       = store.DocumentCategoryContract
	documentCategoryPlan           = store.DocumentCategoryPlan
	documentCategoryOther          = store.DocumentCategoryOther
	documentVisibilityAllResidents = store.DocumentVisibilityAllResidents
	documentVisibilityOwnersOnly   = store.DocumentVisibilityOwnersOnly
	documentVisibilityManagerOnly  = store.DocumentVisibilityManagerOnly
)

// ── extracted to store ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
var cleanParkingPaymentField = store.CleanParkingPaymentField
var handoverTokenHash = store.HandoverTokenHash
var subtleConstantStringCompare = store.SubtleConstantStringCompare

// ── extracted to store ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
var defaultParkingTenantData = store.DefaultParkingTenantData
var normalizeNumericSamples = store.NormalizeNumericSamples

// ── extracted to store ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
var copyMonthStates = store.CopyMonthStates
var uniqueEmails = store.UniqueEmails

// ── extracted to store ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
type (
	attachmentFileSave    = store.AttachmentFileSave
	attachmentRecord      = store.AttachmentRecord
	attachmentStore       = store.AttachmentStore
	attachmentStoreData   = store.AttachmentStoreData
	documentFileSave      = store.DocumentFileSave
	documentRecord        = store.DocumentRecord
	documentStore         = store.DocumentStore
	documentStoreData     = store.DocumentStoreData
	handoverConfirmation  = store.HandoverConfirmation
	handoverKey           = store.HandoverKey
	handoverMeter         = store.HandoverMeter
	handoverRecord        = store.HandoverRecord
	handoverRoom          = store.HandoverRoom
	handoverStore         = store.HandoverStore
	handoverStoreData     = store.HandoverStoreData
	handoverTokenDelivery = store.HandoverTokenDelivery
	issueComment          = store.IssueComment
	issueStatusChange     = store.IssueStatusChange
	issueStore            = store.IssueStore
	issueStoreData        = store.IssueStoreData
	issueWorkflowUpdate   = store.IssueWorkflowUpdate
	parkingMonthState     = store.ParkingMonthState
	parkingNumericSample  = store.ParkingNumericSample
	parkingSettings       = store.ParkingSettings
	parkingStore          = store.ParkingStore
	parkingStoreData      = store.ParkingStoreData
	parkingStoredSample   = store.ParkingStoredSample
	parkingTariff         = store.ParkingTariff
	parkingTenantData     = store.ParkingTenantData
	residentIssue         = store.ResidentIssue
	uploadedFile          = store.UploadedFile
)

var attachmentExtension = store.AttachmentExtension
var copyDocument = store.CopyDocument
var copyHandover = store.CopyHandover
var copyIssue = store.CopyIssue
var detectAttachmentContentType = store.DetectAttachmentContentType
var documentExtension = store.DocumentExtension
var isImageContentType = store.IsImageContentType
var issuePhotoExtension = store.IssuePhotoExtension
var newAttachmentStore = store.NewAttachmentStore
var newDocumentStore = store.NewDocumentStore
var newHandoverStore = store.NewHandoverStore
var newIssueStore = store.NewIssueStore
var newParkingStore = store.NewParkingStore
var normalizeAttachmentEntity = store.NormalizeAttachmentEntity
var normalizeDocumentCategory = store.NormalizeDocumentCategory
var normalizeDocumentRecord = store.NormalizeDocumentRecord
var normalizeDocumentVisibility = store.NormalizeDocumentVisibility
var normalizeDocuments = store.NormalizeDocuments
var normalizeHandover = store.NormalizeHandover
var normalizeHandoverConfirmations = store.NormalizeHandoverConfirmations
var normalizeHandoverKeys = store.NormalizeHandoverKeys
var normalizeHandoverMeters = store.NormalizeHandoverMeters
var normalizeHandoverRooms = store.NormalizeHandoverRooms
var normalizeHandoverType = store.NormalizeHandoverType
var normalizeHandovers = store.NormalizeHandovers
var normalizeIssueCategory = store.NormalizeIssueCategory
var normalizeIssueLocation = store.NormalizeIssueLocation
var normalizeIssuePriority = store.NormalizeIssuePriority
var normalizeIssueStatus = store.NormalizeIssueStatus
var normalizeParkingMonthState = store.NormalizeParkingMonthState
var normalizeParkingMonthStates = store.NormalizeParkingMonthStates
var normalizeParkingMonths = store.NormalizeParkingMonths
var normalizeParkingSettings = store.NormalizeParkingSettings
var normalizeParkingTariff = store.NormalizeParkingTariff
var normalizeParkingTariffDate = store.NormalizeParkingTariffDate
var rejectActiveAttachmentContent = store.RejectActiveAttachmentContent
var resizeImageNearest = store.ResizeImageNearest
var sanitizeDocumentFilename = store.SanitizeDocumentFilename
var sortDocuments = store.SortDocuments
var sortHandovers = store.SortHandovers
var sortIssues = store.SortIssues
var writeImageAttachmentVariant = store.WriteImageAttachmentVariant
var writePrivateFile = store.WritePrivateFile

// Vocabulary constants now owned by the store; aliased so call sites are unchanged.
const (
	auditActionBuildingUpdate          = store.AuditActionBuildingUpdate
	auditActionContactDelete           = store.AuditActionContactDelete
	auditActionContactSave             = store.AuditActionContactSave
	auditActionDocumentDownload        = store.AuditActionDocumentDownload
	auditActionDocumentReplace         = store.AuditActionDocumentReplace
	auditActionDocumentUpload          = store.AuditActionDocumentUpload
	auditActionHeroUpdate              = store.AuditActionHeroUpdate
	auditActionInviteCreate            = store.AuditActionInviteCreate
	auditActionInviteDelete            = store.AuditActionInviteDelete
	auditActionInviteUpdate            = store.AuditActionInviteUpdate
	auditActionIssueEstimate           = store.AuditActionIssueEstimate
	auditActionIssueServiceAdd         = store.AuditActionIssueServiceAdd
	auditActionIssueServiceDrop        = store.AuditActionIssueServiceDrop
	auditActionIssueWorkflow           = store.AuditActionIssueWorkflow
	auditActionLogin                   = store.AuditActionLogin
	auditActionParkingMonth            = store.AuditActionParkingMonth
	auditActionParkingReminder         = store.AuditActionParkingReminder
	auditActionParkingSettings         = store.AuditActionParkingSettings
	auditActionUnitDelete              = store.AuditActionUnitDelete
	auditActionUnitPayment             = store.AuditActionUnitPayment
	auditActionUnitSave                = store.AuditActionUnitSave
	auditActionVoteCast                = store.AuditActionVoteCast
	auditActionVoteClose               = store.AuditActionVoteClose
	auditActionVoteCreate              = store.AuditActionVoteCreate
	auditActionVoteOpen                = store.AuditActionVoteOpen
	auditActionVoteReminder            = store.AuditActionVoteReminder
	ballotStatusClosed                 = store.BallotStatusClosed
	ballotStatusDraft                  = store.BallotStatusDraft
	ballotStatusOpen                   = store.BallotStatusOpen
	ballotTypeCircular                 = store.BallotTypeCircular
	ballotTypeMeeting                  = store.BallotTypeMeeting
	ballotWeightingPerHead             = store.BallotWeightingPerHead
	ballotWeightingPerShare            = store.BallotWeightingPerShare
	defaultBallotReminderBeforeMinutes = store.DefaultBallotReminderBeforeMinutes
	maxBallotReminderBeforeMinutes     = store.MaxBallotReminderBeforeMinutes
	unitBillableFullPPM                = store.UnitBillableFullPPM
	unitTypeCommercial                 = store.UnitTypeCommercial
	unitTypeOther                      = store.UnitTypeOther
	unitTypeParking                    = store.UnitTypeParking
	unitTypeResidential                = store.UnitTypeResidential
	unitTypeStorage                    = store.UnitTypeStorage
)

// ── extracted to store ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
var defaultUnitBillableWeight = store.DefaultUnitBillableWeight

// ── extracted to store ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
type (
	announcement          = store.Announcement
	announcementStore     = store.AnnouncementStore
	announcementStoreData = store.AnnouncementStoreData
	auditEvent            = store.AuditEvent
	auditFilter           = store.AuditFilter
	auditStore            = store.AuditStore
	ballot                = store.Ballot
	ballotVote            = store.BallotVote
	eventStore            = store.EventStore
	eventStoreData        = store.EventStoreData
	houseEvent            = store.HouseEvent
	unit                  = store.Unit
	unitMembers           = store.UnitMembers
	unitMembership        = store.UnitMembership
	unitStore             = store.UnitStore
	unitStoreData         = store.UnitStoreData
	voteStore             = store.VoteStore
	voteStoreData         = store.VoteStoreData
)

var auditDetailValues = store.AuditDetailValues
var auditEventMatches = store.AuditEventMatches
var ballotHasOption = store.BallotHasOption
var billableUnitWeight = store.BillableUnitWeight
var copyAuditEvent = store.CopyAuditEvent
var copyBallot = store.CopyBallot
var copyEvent = store.CopyEvent
var copyUnit = store.CopyUnit
var emailListContains = store.EmailListContains
var eventRollsOffAt = store.EventRollsOffAt
var newAnnouncementStore = store.NewAnnouncementStore
var newAuditStore = store.NewAuditStore
var newEventStore = store.NewEventStore
var newUnitStore = store.NewUnitStore
var newVoteStore = store.NewVoteStore
var normalizeAnnouncementCategory = store.NormalizeAnnouncementCategory
var normalizeAuditAction = store.NormalizeAuditAction
var normalizeAuditEvent = store.NormalizeAuditEvent
var normalizeBallot = store.NormalizeBallot
var normalizeBallotOptions = store.NormalizeBallotOptions
var normalizeBallotStatus = store.NormalizeBallotStatus
var normalizeBallotType = store.NormalizeBallotType
var normalizeBallotWeighting = store.NormalizeBallotWeighting
var normalizeBallots = store.NormalizeBallots
var normalizeEmailList = store.NormalizeEmailList
var normalizeEventCategory = store.NormalizeEventCategory
var normalizeHouseEvent = store.NormalizeHouseEvent
var normalizeRole = store.NormalizeRole
var normalizeUnitBillableWeight = store.NormalizeUnitBillableWeight
var normalizeUnitType = store.NormalizeUnitType
var normalizeUnits = store.NormalizeUnits
var sanitizeAuditDetails = store.SanitizeAuditDetails
var sortAnnouncements = store.SortAnnouncements
var sortBallots = store.SortBallots
var sortEvents = store.SortEvents
var sortUnits = store.SortUnits
var truncateAuditValue = store.TruncateAuditValue
var unitLess = store.UnitLess

const (
	unitPaymentStatusOpen    = store.UnitPaymentStatusOpen
	unitPaymentStatusPaid    = store.UnitPaymentStatusPaid
	unitPaymentStatusPartial = store.UnitPaymentStatusPartial
	unitPaymentStatusOverdue = store.UnitPaymentStatusOverdue
)

// ── extracted to store ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
type (
	activityRecord            = store.ActivityRecord
	activityStore             = store.ActivityStore
	announcementReadStore     = store.AnnouncementReadStore
	announcementReadStoreData = store.AnnouncementReadStoreData
	contactBookStore          = store.ContactBookStore
	contactBookStoreData      = store.ContactBookStoreData
	managedContact            = store.ManagedContact
	notificationPrefStore     = store.NotificationPrefStore
	notificationPrefStoreData = store.NotificationPrefStoreData
	notificationPreferences   = store.NotificationPreferences
	profileOverlay            = store.ProfileOverlay
	profileOverlayStore       = store.ProfileOverlayStore
	profileOverlayStoreData   = store.ProfileOverlayStoreData
	unitPaymentStatus         = store.UnitPaymentStatus
	unitPaymentStatusData     = store.UnitPaymentStatusData
	unitPaymentStatusStore    = store.UnitPaymentStatusStore
)

var defaultNotificationPreferences = store.DefaultNotificationPreferences
var managedContactDisplayName = store.ManagedContactDisplayName
var mergeNotificationPreferences = store.MergeNotificationPreferences
var newActivityStore = store.NewActivityStore
var newAnnouncementReadStore = store.NewAnnouncementReadStore
var newContactBookStore = store.NewContactBookStore
var newNotificationPrefStore = store.NewNotificationPrefStore
var newProfileOverlayStore = store.NewProfileOverlayStore
var newUnitPaymentStatusStore = store.NewUnitPaymentStatusStore
var normalizeContactKind = store.NormalizeContactKind
var normalizeManagedContact = store.NormalizeManagedContact
var normalizeNotificationEvent = store.NormalizeNotificationEvent
var normalizeNotificationPreferences = store.NormalizeNotificationPreferences
var normalizeProfileOverlay = store.NormalizeProfileOverlay
var normalizeUnitID = store.NormalizeUnitID
var normalizeUnitPaymentRecord = store.NormalizeUnitPaymentRecord
var normalizeUnitPaymentStatus = store.NormalizeUnitPaymentStatus
var saveJSONAtomic = store.SaveJSONAtomic
var sortManagedContacts = store.SortManagedContacts
var sortUnitPaymentStatuses = store.SortUnitPaymentStatuses

const (
	roleAdmin                     = store.RoleAdmin
	roleManager                   = store.RoleManager
	roleOwner                     = store.RoleOwner
	roleRenter                    = store.RoleRenter
	roleBeirat                    = store.RoleBeirat
	roleResident                  = store.RoleResident
	roleServiceProvider           = store.RoleServiceProvider
	permissionParking             = store.PermissionParking
	authMethodEmail               = store.AuthMethodEmail
	authMethodOIDC                = store.AuthMethodOIDC
	issueStatusNew                = store.IssueStatusNew
	issueStatusProgress           = store.IssueStatusProgress
	issueStatusDone               = store.IssueStatusDone
	issueStatusRejected           = store.IssueStatusRejected
	issueStatusDuplicate          = store.IssueStatusDuplicate
	issueStatusOpen               = store.IssueStatusOpen
	issuePriorityLow              = store.IssuePriorityLow
	issuePriorityNorm             = store.IssuePriorityNorm
	issuePriorityHigh             = store.IssuePriorityHigh
	issuePriorityUrgent           = store.IssuePriorityUrgent
	issueLocationUnit             = store.IssueLocationUnit
	issueLocationCommon           = store.IssueLocationCommon
	notificationEventAnnouncement = store.NotificationEventAnnouncement
	notificationEventIssue        = store.NotificationEventIssue
	notificationEventVote         = store.NotificationEventVote
	notificationEventDocument     = store.NotificationEventDocument
	notificationEventPayment      = store.NotificationEventPayment
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
	unitPaymentStore      *unitPaymentStatusStore
	issueStore            *issueStore
	attachmentStore       *attachmentStore
	contactStore          *contactBookStore
	auditStore            *auditStore
	documentStore         *documentStore
	handoverStore         *handoverStore
	voteStore             *voteStore
	voteReminderInterval  time.Duration
	parkingStore          *parkingStore
	parkingSampleInterval time.Duration
	parkingHistoryStart   time.Time
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

type tenantOverrideStore struct {
	mu   sync.Mutex
	path string
	data tenantOverrideStoreData
}

type tenantOverrideStoreData struct {
	Tenants map[string]tenantOverride `json:"tenants"`
}

type tenantOverride struct {
	MetaSet           bool      `json:"meta_set,omitempty"`
	Name              string    `json:"name,omitempty"`
	Address           string    `json:"address,omitempty"`
	BrandIcon         string    `json:"brand_icon,omitempty"`
	BrandAbbreviation string    `json:"brand_abbreviation,omitempty"`
	ContactName       string    `json:"contact_name,omitempty"`
	ContactEmail      string    `json:"contact_email,omitempty"`
	ContactPhone      string    `json:"contact_phone,omitempty"`
	EmergencyName     string    `json:"emergency_name,omitempty"`
	EmergencyPhone    string    `json:"emergency_phone,omitempty"`
	CaretakerName     string    `json:"caretaker_name,omitempty"`
	CaretakerEmail    string    `json:"caretaker_email,omitempty"`
	CaretakerPhone    string    `json:"caretaker_phone,omitempty"`
	HeroImage         string    `json:"hero_image,omitempty"`
	UpdatedAt         time.Time `json:"updated_at,omitempty"`
}

const (
	tenantBrandCommunity    = "community"
	tenantBrandSingleHome   = "single-home"
	tenantBrandMultiTenant  = "multi-tenant"
	tenantBrandMixedUse     = "mixed-use"
	tenantBrandAddressPlate = "address-plaque"
	tenantBrandParking      = "parking"
)

// routes builds the application's ServeMux. Extracted from main() so that
// tests exercise the real route patterns instead of calling handler methods
// directly — a test that fakes r.SetPathValue cannot catch a wrong pattern.
func (a *app) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /assets/", http.FileServerFS(web.Assets))
	mux.HandleFunc("GET /favicon.svg", favicon)
	mux.HandleFunc("GET /favicon.ico", favicon)
	mux.HandleFunc("GET /tenant-hero/{tenant}", a.tenantHeroImage)
	mux.HandleFunc("GET /healthz", a.health)
	mux.HandleFunc("GET /", a.home)
	mux.HandleFunc("POST /auth/request", a.requestLogin)
	mux.HandleFunc("GET /auth/verify", a.verifyLogin)
	mux.HandleFunc("GET /auth/oidc/start", a.startOIDCLogin)
	mux.HandleFunc("GET /auth/oidc/callback", a.finishOIDCLogin)
	mux.HandleFunc("POST /auth/logout", a.logout)
	mux.HandleFunc("GET /calendar/{token}", a.calendarFeed)
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
	mux.HandleFunc("GET /app/dokumente/{id}/preview", a.previewDocument)
	mux.HandleFunc("GET /app/dokumente/{id}/download", a.downloadDocument)
	mux.HandleFunc("GET /app/attachments/{id}", a.serveAttachment)
	mux.HandleFunc("GET /app/attachments/{id}/{variant}", a.serveAttachment)
	mux.HandleFunc("POST /app/attachments/delete", a.deleteAttachment)
	mux.HandleFunc("GET /app/abstimmungen", a.ballots)
	mux.HandleFunc("POST /app/abstimmungen", a.submitBallot)
	mux.HandleFunc("POST /app/abstimmungen/open", a.openBallot)
	mux.HandleFunc("POST /app/abstimmungen/close", a.closeBallot)
	mux.HandleFunc("GET /app/abstimmungen/{id}/protokoll", a.ballotProtocol)
	mux.HandleFunc("GET /app/uebergaben", a.handovers)
	mux.HandleFunc("POST /app/uebergaben", a.createHandover)
	mux.HandleFunc("POST /app/uebergaben/file", a.fileHandoverProtocol)
	mux.HandleFunc("GET /app/uebergaben/{id}/protokoll", a.handoverProtocol)
	mux.HandleFunc("GET /handover/{token}", a.handoverConfirmPage)
	mux.HandleFunc("POST /handover/{token}", a.confirmHandover)
	mux.HandleFunc("GET /app/kontakte", a.contacts)
	mux.HandleFunc("POST /app/kontakte", a.upsertManagedContact)
	mux.HandleFunc("POST /app/kontakte/delete", a.deactivateManagedContact)
	mux.HandleFunc("GET /app/anliegen", a.issues)
	mux.HandleFunc("GET /app/anliegen/board", a.issueBoard)
	mux.HandleFunc("GET /app/anliegen/{id}/photos/{index}", a.serveLegacyIssuePhoto)
	mux.HandleFunc("POST /app/anliegen", a.createIssue)
	mux.HandleFunc("POST /app/anliegen/comment", a.addIssueComment)
	mux.HandleFunc("POST /app/anliegen/comment/delete", a.deleteIssueComment)
	mux.HandleFunc("POST /app/anliegen/workflow", a.updateIssueWorkflow)
	mux.HandleFunc("GET /app/parking", a.parking)
	mux.HandleFunc("GET /app/parking/settings", a.parkingSettings)
	mux.HandleFunc("GET /app/parking/month/{month}", a.parkingMonth)
	mux.HandleFunc("GET /app/parking/export/{year}", a.parkingStatement)
	mux.HandleFunc("POST /app/parking/settings", a.updateParkingSettings)
	mux.HandleFunc("POST /app/parking/month", a.updateParkingMonth)
	mux.HandleFunc("POST /app/parking/reminders", a.sendParkingReminders)
	mux.HandleFunc("GET /app/audit", a.auditLog)
	mux.HandleFunc("GET /app/settings", a.settingsHub)
	mux.HandleFunc("GET /app/settings/building", a.buildingSettings)
	mux.HandleFunc("POST /app/settings/building", a.updateBuildingSettings)
	mux.HandleFunc("POST /app/settings/building/hero", a.updateBuildingHero)
	mux.HandleFunc("POST /app/settings/building/hero/delete", a.deleteBuildingHero)
	mux.HandleFunc("POST /app/settings/building/units", a.upsertBuildingUnit)
	mux.HandleFunc("POST /app/settings/building/units/delete", a.deleteBuildingUnit)
	mux.HandleFunc("POST /app/settings/building/payment-status", a.updateUnitPaymentStatus)
	mux.HandleFunc("GET /app/settings/profile", a.profileSettings)
	mux.HandleFunc("POST /app/settings/profile", a.updateProfileSettings)
	mux.HandleFunc("GET /app/settings/notifications", a.notificationSettings)
	mux.HandleFunc("POST /app/settings/notifications", a.updateNotificationSettings)
	mux.HandleFunc("GET /app/settings/parking-access", a.parkingAccessSettings)
	mux.HandleFunc("POST /app/settings/parking-access", a.updateParkingAccess)
	mux.HandleFunc("GET /app/settings/users", a.userSettings)
	mux.HandleFunc("POST /app/settings/users", a.createInvite)
	mux.HandleFunc("POST /app/settings/users/edit", a.editInvite)
	mux.HandleFunc("POST /app/settings/users/delete", a.deleteInvite)
	mux.HandleFunc("GET /{tenant}", a.tenantPathRedirect)
	mux.HandleFunc("GET /{tenant}/{rest...}", a.tenantPathRedirect)
	return mux
}

// handler is the fully wrapped HTTP handler, middleware included. This is
// what main() serves and what the tests drive.
func (a *app) handler() http.Handler {
	return securityHeaders(a.routes())
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

	server := &http.Server{
		Addr:              a.addr,
		Handler:           a.handler(),
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

	tmpl, err := template.New("pages").Parse(web.PageTemplates)
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

	mailTransport := appmail.NewSMTP(
		env("SMTP_HOST", ""),
		env("SMTP_PORT", "587"),
		env("SMTP_USER", ""),
		env("SMTP_PASS", ""),
		env("MAIL_FROM", "WEG Portal <noreply@example.invalid>"),
	)
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
	unitPaymentDataPath := env("UNIT_PAYMENT_STATUS_DATA_PATH", "tmp/unit_payment_status.json")
	unitPayments, err := newUnitPaymentStatusStore(unitPaymentDataPath)
	if err != nil {
		return nil, err
	}
	issueDataPath := env("ISSUE_DATA_PATH", "tmp/issues.json")
	defaultIssueAttachmentDir := filepath.Join(filepath.Dir(issueDataPath), "issue-attachments")
	issues, err := newIssueStore(issueDataPath, env("ISSUE_ATTACHMENT_DIR", defaultIssueAttachmentDir))
	if err != nil {
		return nil, err
	}
	attachmentDataPath := env("ATTACHMENT_DATA_PATH", "tmp/attachments.json")
	defaultAttachmentFileDir := filepath.Join(filepath.Dir(attachmentDataPath), "attachments")
	attachments, err := newAttachmentStore(attachmentDataPath, env("ATTACHMENT_FILE_DIR", defaultAttachmentFileDir))
	if err != nil {
		return nil, err
	}
	contactDataPath := env("CONTACT_DATA_PATH", "tmp/contacts.json")
	contacts, err := newContactBookStore(contactDataPath)
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
	handoverDataPath := env("HANDOVER_DATA_PATH", "tmp/handovers.json")
	handovers, err := newHandoverStore(handoverDataPath)
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
		baseURL:               baseURL,
		addr:                  env("ADDR", ":8080"),
		rootDomain:            rootDomain,
		defaultTenant:         defaultTenant,
		tenants:               tenants,
		sessionSecure:         parsed.Scheme == "https",
		allowed:               allowed,
		admins:                admins,
		profiles:              profiles,
		localDevLogin:         localDevLogin,
		sessionTTL:            sessionTTL,
		tokens:                auth.NewTokenStore(secret),
		sessions:              newSessionStore(secret),
		oidc:                  oidcLogin,
		oidcFlows:             auth.NewOIDCFlowStore(),
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
		unitPaymentStore:      unitPayments,
		issueStore:            issues,
		attachmentStore:       attachments,
		contactStore:          contacts,
		auditStore:            auditStore,
		documentStore:         documents,
		handoverStore:         handovers,
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

func favicon(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = io.WriteString(w, web.FaviconSVG)
}

func (a *app) home(w http.ResponseWriter, r *http.Request) {
	if a.isMarketingHost(r) {
		a.marketingLanding(w, r)
		return
	}
	tenant := a.tenantForRequest(r)
	email, _, tenantSlug, ok := a.currentUser(r)
	if ok && tenantSlug == tenant.Slug {
		http.Redirect(w, r, "/app", http.StatusSeeOther)
		return
	}
	unitWeight := 0
	if a.unitStore != nil {
		unitWeight = a.unitStore.BillableUnitWeight(tenant.Slug)
	}
	titleName := firstNonEmpty(tenant.Name, "WEG Portal")
	a.render(w, "home", map[string]any{
		"Title":               titleName + " " + tenant.Address,
		"Tenant":              tenant,
		"Email":               email,
		"UnitCount":           formatBillableUnitWeight(unitWeight),
		"UnitCountLabel":      billableUnitCountLabel(unitWeight),
		"HasUnitCount":        unitWeight > 0,
		"Sent":                r.URL.Query().Get("sent") == "1",
		"MailConfigured":      a.mailer.Configured(),
		"DevLoginLink":        "",
		"Denied":              r.URL.Query().Get("denied") == "1",
		"OIDCConfigured":      a.oidc.Configured(),
		"OIDCProviderName":    a.oidc.ProviderName(),
		"EmailLoginAvailable": a.emailLoginAvailable(),
	})
}

func (a *app) marketingLanding(w http.ResponseWriter, r *http.Request) {
	a.render(w, "landing", map[string]any{
		"Title":          "hausv.org - kostenlose Hausverwaltung",
		"ContactLocal":   "hello",
		"ContactDomain":  "hausv.org",
		"ContactDisplay": "hello [at] hausv [dot] org",
		"PrimaryAppURL":  "https://jhw22.hausv.org/",
		"RequestedHost":  normalizeHost(r.Host),
		"LandingHeroURL": "/assets/hausv-landing-hero.png",
	})
}

func (a *app) requestLogin(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	if !a.emailLoginAvailable() {
		http.Redirect(w, r, "/?denied=1", http.StatusSeeOther)
		return
	}
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
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
	email, tenantSlug, redirectPath, ok := a.tokens.Consume(token)
	if !ok {
		http.Error(w, "Dieser Anmeldelink ist abgelaufen oder wurde bereits verwendet.", http.StatusUnauthorized)
		return
	}

	if err := a.startSession(w, email, tenantSlug, authMethodEmail); err != nil {
		http.Error(w, "Could not create session", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, firstNonEmpty(redirectPath, "/app"), http.StatusSeeOther)
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
	a.oidcFlows.Put(state, auth.NewOIDCFlow(tenant.Slug, nonce, codeVerifier), 10*time.Minute)

	redirectURL := a.oidc.RedirectURL(a.publicBaseURL(r, tenant))
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
	tenant, ok := a.tenantBySlug(flow.TenantSlug())
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
	oauthConfig := a.oidc.OAuthConfig(a.oidc.RedirectURL(a.publicBaseURL(r, tenant)))
	token, err := oauthConfig.Exchange(
		ctx,
		code,
		oauth2.SetAuthURLParam("code_verifier", flow.CodeVerifier()),
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
	idToken, err := a.oidc.Verifier().Verify(ctx, rawIDToken)
	if err != nil {
		log.Printf("oidc id_token verification failed: %v", err)
		http.Error(w, "SSO-Anmeldung konnte nicht geprüft werden.", http.StatusUnauthorized)
		return
	}
	if idToken.Nonce != flow.Nonce() {
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
		userInfo, err := a.oidc.Provider().UserInfo(ctx, oauth2.StaticTokenSource(token))
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
	if denyServiceProviderArea(w, role) {
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
			all = a.announcementViewsWithReadState(tenant.Slug, a.announcementStore.ListTenant(tenant.Slug), now, true, lastSeen, email, role)
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
		"Announcements":          a.announcementViewsWithReadState(tenant.Slug, filtered, now, true, lastSeen, email, role),
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
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	item, err := announcementFromForm(r, tenant.Slug, profile, time.Now())
	if err != nil {
		http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
		return
	}
	attachmentHeaders, err := attachmentFormHeaders(r, maxIssueAttachmentCount, "attachments")
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
	if len(attachmentHeaders) > 0 {
		if a.attachmentStore == nil {
			_, _ = a.announcementStore.Delete(tenant.Slug, created.ID)
			http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
			return
		}
		if _, err := a.attachmentStore.CreateUploaded(tenant.Slug, "announcement", created.ID, email, uploadedFilesFromHeaders(attachmentHeaders), time.Now()); err != nil {
			_, _ = a.announcementStore.Delete(tenant.Slug, created.ID)
			http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
			return
		}
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
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
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
	attachmentHeaders, err := attachmentFormHeaders(r, maxIssueAttachmentCount, "attachments")
	if err != nil {
		http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
		return
	}
	var uploaded []attachmentRecord
	if len(attachmentHeaders) > 0 {
		if a.attachmentStore == nil {
			http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
			return
		}
		uploaded, err = a.attachmentStore.CreateUploaded(tenant.Slug, "announcement", id, email, uploadedFilesFromHeaders(attachmentHeaders), time.Now())
		if err != nil {
			http.Redirect(w, r, "/app/announcements?announce=invalid", http.StatusSeeOther)
			return
		}
	}
	ok, err = a.announcementStore.Update(id, item)
	if err != nil {
		for _, attachment := range uploaded {
			_, _, _ = a.attachmentStore.Delete(tenant.Slug, attachment.ID, time.Now())
		}
		log.Printf("announcement update failed for %s: %v", tenant.Slug, err)
		http.Redirect(w, r, "/app/announcements?announce=error", http.StatusSeeOther)
		return
	}
	if !ok {
		for _, attachment := range uploaded {
			_, _, _ = a.attachmentStore.Delete(tenant.Slug, attachment.ID, time.Now())
		}
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
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
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
	if denyServiceProviderArea(w, role) {
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
	calendarFeedURL := ""
	if token, err := a.calendarFeedToken(email, tenant.Slug); err == nil {
		calendarFeedURL = a.publicBaseURL(r, tenant) + "/calendar/" + url.PathEscape(token) + ".ics"
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
		"Events":                 a.eventViews(tenant.Slug, upcoming, now, email, role),
		"HasEvents":              len(upcoming) > 0,
		"EventsEmpty":            emptyState("Noch keine kommenden Termine", "Geplante Versammlungen, Wartungen und Fristen erscheinen hier."),
		"CalendarFeedURL":        calendarFeedURL,
		"HasCalendarFeedURL":     calendarFeedURL != "",
		"AllEvents":              a.eventViews(tenant.Slug, all, now, email, role),
		"HasAllEvents":           len(all) > 0,
		"AllEventsEmpty":         emptyState("Noch kein Termin gespeichert", "Neue Termine erscheinen hier nach dem Speichern."),
		"EventMsg":               eventMessage(r.URL.Query().Get("event")),
		"NowInput":               formatLocalDateTimeInput(now),
	})
}

type calendarFeedPayload struct {
	Email      string `json:"email"`
	TenantSlug string `json:"tenant"`
	IssuedAt   int64  `json:"iat"`
}

func (a *app) calendarFeed(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.PathValue("token"))
	token = strings.TrimSuffix(token, ".ics")
	payload, ok := a.parseCalendarFeedToken(token)
	if !ok {
		http.NotFound(w, r)
		return
	}
	tenant, ok := a.tenantBySlug(payload.TenantSlug)
	if !ok || !a.isAllowed(payload.Email, tenant.Slug) {
		http.NotFound(w, r)
		return
	}
	role := a.roleFor(payload.Email, tenant.Slug)
	profile := a.profileForTenant(payload.Email, tenant.Slug)
	body := a.renderCalendarFeed(tenant, profile, role, time.Now().UTC())
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.Header().Set("Content-Disposition", `inline; filename="hausv-`+tenant.Slug+`.ics"`)
	_, _ = io.WriteString(w, body)
}

func (a *app) calendarFeedToken(email string, tenantSlug string) (string, error) {
	email = normalizeEmail(email)
	tenantSlug = normalizeSlug(tenantSlug)
	if email == "" || tenantSlug == "" {
		return "", fmt.Errorf("calendar feed identity required")
	}
	payload := calendarFeedPayload{
		Email:      email,
		TenantSlug: tenantSlug,
		IssuedAt:   time.Now().Unix(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(raw)
	signedPart := "v1." + encodedPayload
	signature, err := a.signCalendarFeed(signedPart)
	if err != nil {
		return "", err
	}
	return signedPart + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (a *app) parseCalendarFeedToken(token string) (calendarFeedPayload, bool) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 || parts[0] != "v1" {
		return calendarFeedPayload{}, false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return calendarFeedPayload{}, false
	}
	expected, err := a.signCalendarFeed(parts[0] + "." + parts[1])
	if err != nil || !hmac.Equal(signature, expected) {
		return calendarFeedPayload{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return calendarFeedPayload{}, false
	}
	var payload calendarFeedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return calendarFeedPayload{}, false
	}
	payload.Email = normalizeEmail(payload.Email)
	payload.TenantSlug = normalizeSlug(payload.TenantSlug)
	if payload.Email == "" || payload.TenantSlug == "" {
		return calendarFeedPayload{}, false
	}
	return payload, true
}

func (a *app) signCalendarFeed(value string) ([]byte, error) {
	if a == nil || a.sessions == nil {
		return nil, fmt.Errorf("calendar feed signing secret unavailable")
	}
	return a.sessions.Sign(value)
}

func (a *app) renderCalendarFeed(tenant tenantConfig, profile userProfile, role string, now time.Time) string {
	email := normalizeEmail(profile.Email)
	tenantSlug := normalizeSlug(tenant.Slug)
	var b strings.Builder
	calendarLine(&b, "BEGIN", "VCALENDAR")
	calendarLine(&b, "VERSION", "2.0")
	calendarLine(&b, "PRODID", "-//hausv.org//Portal//DE")
	calendarLine(&b, "CALSCALE", "GREGORIAN")
	calendarLine(&b, "METHOD", "PUBLISH")
	calendarLine(&b, "X-WR-CALNAME", "hausv.org "+tenant.Name)
	calendarLine(&b, "X-WR-CALDESC", "Termine und freigegebene Vorgänge für "+tenant.Address)
	if canUseResidentAreas(role) && a != nil && a.eventStore != nil {
		for _, item := range a.eventStore.Upcoming(tenantSlug, now) {
			a.writeCalendarEvent(&b, tenant, item, now)
		}
	}
	if a != nil && a.issueStore != nil {
		for _, item := range a.issueStore.ListTenant(tenantSlug) {
			if strings.TrimSpace(item.ServiceProposal) == "" || !a.canViewIssueForActor(tenantSlug, item, email, role) {
				continue
			}
			writeCalendarIssueProposal(&b, tenant, item, now)
		}
	}
	calendarLine(&b, "END", "VCALENDAR")
	return b.String()
}

func (a *app) writeCalendarEvent(b *strings.Builder, tenant tenantConfig, item houseEvent, now time.Time) {
	end := item.StartsAt.Add(time.Hour)
	if item.EndsAt != nil && item.EndsAt.After(item.StartsAt) {
		end = *item.EndsAt
	}
	calendarLine(b, "BEGIN", "VEVENT")
	calendarLine(b, "UID", "event-"+item.ID+"@"+tenant.Slug+".hausv.org")
	calendarLine(b, "DTSTAMP", calendarDateTime(now))
	calendarLine(b, "DTSTART", calendarDateTime(item.StartsAt))
	calendarLine(b, "DTEND", calendarDateTime(end))
	calendarLine(b, "SUMMARY", item.Title)
	if location := strings.TrimSpace(item.Location); location != "" {
		calendarLine(b, "LOCATION", location)
	}
	description := strings.TrimSpace(item.Body)
	if category := strings.TrimSpace(item.Category); category != "" {
		if description != "" {
			description += "\n\n"
		}
		description += "Kategorie: " + category
	}
	if description != "" {
		calendarLine(b, "DESCRIPTION", description)
	}
	if category := strings.TrimSpace(item.Category); category != "" {
		calendarLine(b, "CATEGORIES", category)
	}
	calendarLine(b, "END", "VEVENT")
}

func writeCalendarIssueProposal(b *strings.Builder, tenant tenantConfig, item residentIssue, now time.Time) {
	calendarLine(b, "BEGIN", "VTODO")
	calendarLine(b, "UID", "issue-proposal-"+item.ID+"@"+tenant.Slug+".hausv.org")
	calendarLine(b, "DTSTAMP", calendarDateTime(now))
	calendarLine(b, "SUMMARY", "Terminvorschlag: "+item.Title)
	description := strings.TrimSpace(item.ServiceProposal)
	if description != "" {
		description += "\n\n"
	}
	description += "Anliegen: " + item.Title
	if status := strings.TrimSpace(item.Status); status != "" {
		description += "\nStatus: " + status
	}
	calendarLine(b, "DESCRIPTION", description)
	calendarLine(b, "STATUS", "NEEDS-ACTION")
	calendarLine(b, "END", "VTODO")
}

func calendarDateTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

func calendarLine(b *strings.Builder, name string, value string) {
	b.WriteString(name)
	b.WriteByte(':')
	b.WriteString(calendarEscapeText(value))
	b.WriteString("\r\n")
}

func calendarEscapeText(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, ";", `\;`)
	value = strings.ReplaceAll(value, ",", `\,`)
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.ReplaceAll(value, "\n", `\n`)
	return value
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
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	item, err := eventFromForm(r, tenant.Slug, profile)
	if err != nil {
		http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
		return
	}
	attachmentHeaders, err := attachmentFormHeaders(r, maxIssueAttachmentCount, "attachments")
	if err != nil {
		http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
		return
	}
	created, err := a.eventStore.Create(item)
	if err != nil {
		log.Printf("event create failed for %s: %v", tenant.Slug, err)
		http.Redirect(w, r, "/app/events?event=error", http.StatusSeeOther)
		return
	}
	if len(attachmentHeaders) > 0 {
		if a.attachmentStore == nil {
			_, _ = a.eventStore.Delete(tenant.Slug, created.ID)
			http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
			return
		}
		if _, err := a.attachmentStore.CreateUploaded(tenant.Slug, "event", created.ID, email, uploadedFilesFromHeaders(attachmentHeaders), time.Now()); err != nil {
			_, _ = a.eventStore.Delete(tenant.Slug, created.ID)
			http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
			return
		}
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
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
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
	attachmentHeaders, err := attachmentFormHeaders(r, maxIssueAttachmentCount, "attachments")
	if err != nil {
		http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
		return
	}
	var uploaded []attachmentRecord
	if len(attachmentHeaders) > 0 {
		if a.attachmentStore == nil {
			http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
			return
		}
		uploaded, err = a.attachmentStore.CreateUploaded(tenant.Slug, "event", id, email, uploadedFilesFromHeaders(attachmentHeaders), time.Now())
		if err != nil {
			http.Redirect(w, r, "/app/events?event=invalid", http.StatusSeeOther)
			return
		}
	}
	ok, err = a.eventStore.Update(id, item)
	if err != nil {
		for _, attachment := range uploaded {
			_, _, _ = a.attachmentStore.Delete(tenant.Slug, attachment.ID, time.Now())
		}
		log.Printf("event update failed for %s: %v", tenant.Slug, err)
		http.Redirect(w, r, "/app/events?event=error", http.StatusSeeOther)
		return
	}
	if !ok {
		for _, attachment := range uploaded {
			_, _, _ = a.attachmentStore.Delete(tenant.Slug, attachment.ID, time.Now())
		}
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
	if denyServiceProviderArea(w, role) {
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
	}, uploadedFileFromHeader(header), time.Now())
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
	replacement, replaced, err := a.documentStore.Replace(tenant.Slug, strings.TrimSpace(r.FormValue("id")), email, uploadedFileFromHeader(header), time.Now())
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

func (a *app) previewDocument(w http.ResponseWriter, r *http.Request) {
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
	if !documentCanPreview(item.ContentType) {
		http.NotFound(w, r)
		return
	}
	path, ok := a.documentStore.FilePath(item)
	if !ok {
		http.NotFound(w, r)
		return
	}
	file, err := os.Open(path)
	if err != nil {
		log.Printf("document preview open failed for %s/%s: %v", tenant.Slug, item.ID, err)
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'; sandbox")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": item.Filename}))
	if item.ContentType != "" {
		w.Header().Set("Content-Type", item.ContentType)
	}
	http.ServeContent(w, r, item.Filename, item.UploadedAt, file)
}

func (a *app) serveLegacyIssuePhoto(w http.ResponseWriter, r *http.Request) {
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
	if a.issueStore == nil {
		http.NotFound(w, r)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	index, err := strconv.Atoi(strings.TrimSpace(r.PathValue("index")))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	issue, found := a.issueStore.Get(tenant.Slug, id)
	if !found || !a.canViewIssueForActor(tenant.Slug, issue, email, role) {
		http.NotFound(w, r)
		return
	}
	path, filename, ok := a.legacyIssuePhotoPath(tenant.Slug, issue, index)
	if !ok {
		http.NotFound(w, r)
		return
	}
	file, err := os.Open(path)
	if err != nil {
		log.Printf("legacy issue photo open failed for %s/%s/%d: %v", tenant.Slug, issue.ID, index, err)
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
	if contentType == "" {
		sniff := make([]byte, 512)
		n, readErr := file.Read(sniff)
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			http.NotFound(w, r)
			return
		}
		contentType = http.DetectContentType(sniff[:n])
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			http.NotFound(w, r)
			return
		}
	}
	if !isImageContentType(contentType) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'; sandbox")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": filename}))
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=300")
	http.ServeContent(w, r, filename, issue.UpdatedAt, file)
}

func (a *app) serveAttachment(w http.ResponseWriter, r *http.Request) {
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
	if a.attachmentStore == nil {
		http.NotFound(w, r)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	variant := strings.ToLower(strings.TrimSpace(r.PathValue("variant")))
	if variant != "" && variant != "preview" && variant != "thumb" && variant != "thumbnail" {
		http.NotFound(w, r)
		return
	}
	item, found := a.attachmentStore.Get(tenant.Slug, id)
	if !found {
		http.NotFound(w, r)
		return
	}
	if !a.canViewAttachment(tenant.Slug, item, email, role) {
		http.Error(w, "Dieser Anhang ist für diesen Zugang nicht freigegeben.", http.StatusForbidden)
		return
	}
	path, contentType, _, ok := a.attachmentStore.FilePath(item, variant)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if _, err := os.Stat(path); err != nil {
		log.Printf("attachment file open failed for %s/%s/%s: %v", tenant.Slug, item.EntityType, item.ID, err)
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'; sandbox")
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	disposition := attachmentDisposition(contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": item.Filename}))
	w.Header().Set("Cache-Control", "private, max-age=300")
	http.ServeFile(w, r, path)
}

func (a *app) deleteAttachment(w http.ResponseWriter, r *http.Request) {
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
	if a.attachmentStore == nil {
		http.Redirect(w, r, "/app/anliegen?issue=missing", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	item, found := a.attachmentStore.Get(tenant.Slug, id)
	if !found {
		http.Redirect(w, r, redirectAfterAttachmentChange(r, "/app/anliegen?issue=missing"), http.StatusSeeOther)
		return
	}
	if !a.canDeleteAttachment(tenant.Slug, item, email, role) {
		http.Error(w, "Dieser Anhang kann nur von Verwaltung oder Ersteller entfernt werden.", http.StatusForbidden)
		return
	}
	if _, removed, err := a.attachmentStore.Delete(tenant.Slug, id, time.Now()); err != nil {
		log.Printf("attachment delete failed for %s/%s: %v", tenant.Slug, id, err)
		http.Redirect(w, r, redirectAfterAttachmentChange(r, "/app/anliegen?issue=error"), http.StatusSeeOther)
		return
	} else if !removed {
		http.Redirect(w, r, redirectAfterAttachmentChange(r, "/app/anliegen?issue=missing"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, redirectAfterAttachmentChange(r, "/app/anliegen?issue=updated"), http.StatusSeeOther)
}

func attachmentDisposition(contentType string) string {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if isImageContentType(contentType) || contentType == "application/pdf" {
		return "inline"
	}
	return "attachment"
}

func redirectAfterAttachmentChange(r *http.Request, fallback string) string {
	if fallback == "" {
		fallback = "/app"
	}
	for _, raw := range []string{r.FormValue("redirect"), r.Header.Get("Referer")} {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if strings.HasPrefix(raw, "/app/") || raw == "/app" {
			return raw
		}
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			continue
		}
		if normalizeHost(parsed.Host) != normalizeHost(r.Host) {
			continue
		}
		if parsed.Path == "/app" || strings.HasPrefix(parsed.Path, "/app/") {
			if parsed.RawQuery != "" {
				return parsed.Path + "?" + parsed.RawQuery
			}
			return parsed.Path
		}
	}
	return fallback
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
	attachmentHeaders, err := attachmentFormHeaders(r, maxIssueAttachmentCount, "attachments")
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
	if len(attachmentHeaders) > 0 {
		if a.attachmentStore == nil {
			_, _ = a.voteStore.Delete(tenant.Slug, created.ID)
			http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
			return
		}
		if _, err := a.attachmentStore.CreateUploaded(tenant.Slug, "ballot", created.ID, email, uploadedFilesFromHeaders(attachmentHeaders), time.Now()); err != nil {
			_, _ = a.voteStore.Delete(tenant.Slug, created.ID)
			http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
			return
		}
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
	if isServiceProviderRole(role) {
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
	if denyServiceProviderArea(w, role) {
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
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
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
		"AppVersion":  version.BuildLabel(),
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
	attachments := a.attachmentViewsForEntity(tenantSlug, "ballot", item.ID, email, role)
	if len(attachments) > 0 {
		view.Attachments = attachments
		view.HasAttachments = true
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
	if isServiceProviderRole(role) {
		http.Redirect(w, r, "/app/anliegen", http.StatusSeeOther)
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
	unreadAnnouncements := 0
	if a.announcementStore != nil {
		visible := a.announcementStore.Visible(tenant.Slug, now)
		unreadAnnouncements = unreadAnnouncementCount(visible, lastSeen, now)
		announcements = a.announcementViewsWithReadState(tenant.Slug, visible, now, false, lastSeen, email, role)
		if len(announcements) > 3 {
			announcements = announcements[:3]
		}
	}
	eventCount := 0
	events := []houseEventView{}
	if a.eventStore != nil {
		upcoming := a.eventStore.Upcoming(tenant.Slug, now)
		eventCount = len(upcoming)
		events = a.eventViews(tenant.Slug, upcoming, now, email, role)
		if len(events) > 4 {
			events = events[:4]
		}
	}
	issueURL := "/app/anliegen"
	issueTitle := "Offene Anliegen"
	if hasCapability(role, capabilityManageIssues) {
		issueURL = "/app/anliegen/board"
		issueTitle = "Offene Anliegen im Haus"
	}
	openIssues := []residentIssue{}
	if a.issueStore != nil {
		for _, item := range a.visibleIssuesForActor(tenant.Slug, email, role) {
			if issueIsOpen(item) {
				openIssues = append(openIssues, item)
			}
		}
	}
	issuePreviews := a.issueViewsForActor(tenant.Slug, openIssues, role, email)
	if len(issuePreviews) > 2 {
		issuePreviews = issuePreviews[:2]
	}
	documents := []documentView{}
	documentCount := 0
	if a.documentStore != nil {
		visible := sortDocumentsForView(a.visibleDocumentsForActor(tenant.Slug, email, role), "newest")
		documentCount = len(visible)
		if len(visible) > 3 {
			visible = visible[:3]
		}
		documents = a.documentViewsForActor(tenant.Slug, email, role, visible)
	}
	unitPaymentStatuses := a.unitPaymentStatusViewsForEmail(tenant.Slug, email)
	canSeeParking := isAdmin || profile.HasPermission(permissionParking)
	parkingTitle := "Alles erledigt"
	parkingDetail := "keine offenen Posten"
	parkingSummaryDetail := "Für alle Stellplätze sind keine offenen Meldungen oder Zahlungsrückstände vorhanden."
	parkingPillClass := "ok"
	if canSeeParking {
		balance := a.parkingBalance(tenant.Slug)
		if balance.Outstanding > 0 {
			parkingTitle = "Offen " + formatEUR(balance.Outstanding)
			parkingDetail = "für die Stellplatznutzung"
			parkingSummaryDetail = "Offene Beträge für die Stellplatznutzung sind vorhanden."
			parkingPillClass = "info"
		}
		if balance.Overdue > 0 {
			parkingDetail = "Überfällig " + formatEUR(balance.Overdue)
			parkingSummaryDetail = "Überfällige Beträge sollten geprüft und zugeordnet werden."
			parkingPillClass = "dringend"
		}
	}
	a.render(w, "portal", map[string]any{
		"Title":                     "WEG Portal",
		"Tenant":                    tenant,
		"Email":                     email,
		"DisplayName":               profile.DisplayName(),
		"GreetingName":              firstNonEmpty(profile.FirstName, profile.DisplayName()),
		"Initials":                  profile.Initials(),
		"Role":                      role,
		"IsAdmin":                   isAdmin,
		"CanSeeParking":             canSeeParking,
		"CanManageAnnouncements":    canManage,
		"CanManageEvents":           canManageEvents(role),
		"ActivePage":                "home",
		"UnreadAnnouncements":       unreadAnnouncements,
		"AnnouncementSummaryDetail": pluralizeCount(unreadAnnouncements, "ungelesener Beitrag", "ungelesene Beiträge"),
		"EventCount":                eventCount,
		"EventSummaryDetail":        pluralizeCount(eventCount, "Termin geplant", "Termine geplant"),
		"IssueCount":                len(openIssues),
		"IssueSummaryTitle":         issueTitle,
		"IssueSummaryURL":           issueURL,
		"IssueSummaryDetail":        pluralizeCount(len(openIssues), "offenes Anliegen", "offene Anliegen"),
		"DashboardIssues":           issuePreviews,
		"HasDashboardIssues":        len(issuePreviews) > 0,
		"DashboardIssuesEmpty":      emptyState("Alles erledigt", "Aktuell sind keine offenen Anliegen sichtbar."),
		"DashboardDocuments":        documents,
		"HasDashboardDocuments":     len(documents) > 0,
		"DocumentCount":             documentCount,
		"DocumentSummaryDetail":     pluralizeCount(documentCount, "Dokument sichtbar", "Dokumente sichtbar"),
		"DashboardDocumentsEmpty":   emptyState("Noch keine Dokumente", "Sichtbare Unterlagen erscheinen hier nach Rolle und Berechtigung."),
		"UnitPaymentStatuses":       unitPaymentStatuses,
		"HasUnitPaymentStatuses":    len(unitPaymentStatuses) > 0,
		"ParkingStatusTitle":        parkingTitle,
		"ParkingStatusValue":        parkingTitle,
		"ParkingStatusDetail":       parkingDetail,
		"ParkingSummaryDetail":      parkingSummaryDetail,
		"ParkingStatusClass":        parkingPillClass,
		"Announcements":             announcements,
		"HasAnnouncements":          len(announcements) > 0,
		"AnnouncementsEmpty":        emptyState("Noch keine Beiträge", "Sobald die Verwaltung einen Aushang veröffentlicht, erscheint er hier."),
		"Events":                    events,
		"HasEvents":                 len(events) > 0,
		"EventsEmpty":               emptyState("Noch keine kommenden Termine", "Geplante Versammlungen, Wartungen und Fristen erscheinen hier."),
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
	if denyServiceProviderArea(w, role) {
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	canManageContacts := canManageContacts(role)
	managerContacts := managerContactViews(tenant)
	emergencyContacts := emergencyContactViews(tenant)
	managedContacts := a.managedContactViews(tenant.Slug, canManageContacts)
	boardContacts := a.boardContactViews(tenant.Slug)
	residentContacts := a.residentDirectoryViews(tenant.Slug)
	contactMsg, contactOK := contactMessage(r.URL.Query().Get("contact"))
	a.render(w, "contacts", map[string]any{
		"Title":                "Kontakte",
		"Tenant":               tenant,
		"Email":                email,
		"DisplayName":          profile.DisplayName(),
		"Initials":             profile.Initials(),
		"Role":                 role,
		"IsAdmin":              isAdmin,
		"CanSeeParking":        isAdmin || profile.HasPermission(permissionParking),
		"CanManageContacts":    canManageContacts,
		"ActivePage":           "contacts",
		"ContactMsg":           contactMsg,
		"ContactOK":            contactOK,
		"ManagerContacts":      managerContacts,
		"HasManagerContacts":   len(managerContacts) > 0,
		"ManagerEmpty":         emptyState("Kein Verwaltungskontakt", "Der Kontaktblock wird in den Gebäude-Einstellungen gepflegt."),
		"EmergencyContacts":    emergencyContacts,
		"HasEmergencyContacts": len(emergencyContacts) > 0,
		"EmergencyEmpty":       emptyState("Kein Notdienst hinterlegt", "Notdienst und Hausmeister werden in den Gebäude-Einstellungen gepflegt."),
		"ManagedContacts":      managedContacts,
		"HasManagedContacts":   len(managedContacts) > 0,
		"ManagedEmpty":         emptyState("Noch kein Adressbucheintrag", "Dienstleister, Hausmeister und Notdienste können hier zentral hinterlegt werden."),
		"ContactKindOptions":   contactKindOptions(""),
		"BoardContacts":        boardContacts,
		"HasBoardContacts":     len(boardContacts) > 0,
		"BoardEmpty":           emptyState("Kein Beirat hinterlegt", "Beiräte erscheinen hier, sobald sie in Benutzer & Rechte die Beirat-Rolle haben."),
		"ResidentContacts":     residentContacts,
		"HasResidentContacts":  len(residentContacts) > 0,
		"ResidentEmpty":        emptyState("Keine freigegebenen Kontakte", "Kontakte aus der Hausgemeinschaft erscheinen nur nach ausdrücklicher Freigabe im Profil."),
	})
}

func (a *app) upsertManagedContact(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	actorEmail, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !canManageContacts(role) {
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
	item, err := managedContactFromForm(tenant.Slug, r.Form)
	if err != nil {
		http.Redirect(w, r, "/app/kontakte?contact=invalid", http.StatusSeeOther)
		return
	}
	saved, created, err := a.contactStore.Upsert(item)
	if err != nil {
		log.Printf("contact save failed for %s/%s: %v", tenant.Slug, redactedEmail(item.Email), err)
		http.Redirect(w, r, "/app/kontakte?contact=error", http.StatusSeeOther)
		return
	}
	action := "aktualisiert"
	if created {
		action = "angelegt"
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionContactSave,
		TargetType: "contact",
		TargetID:   saved.ID,
		Summary:    "Adressbuch-Kontakt " + action,
		Details: map[string]string{
			"type":   saved.Kind,
			"status": contactStatusLabel(saved.Active),
		},
	})
	http.Redirect(w, r, "/app/kontakte?contact=saved", http.StatusSeeOther)
}

func (a *app) deactivateManagedContact(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	actorEmail, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !canManageContacts(role) {
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
	removed, err := a.contactStore.Deactivate(tenant.Slug, id, time.Now())
	if err != nil {
		http.Redirect(w, r, "/app/kontakte?contact=error", http.StatusSeeOther)
		return
	}
	if removed.ID != "" {
		a.recordAudit(auditEvent{
			TenantSlug: tenant.Slug,
			ActorEmail: actorEmail,
			ActorRole:  role,
			Action:     auditActionContactDelete,
			TargetType: "contact",
			TargetID:   removed.ID,
			Summary:    "Adressbuch-Kontakt deaktiviert",
			Details: map[string]string{
				"type":   removed.Kind,
				"status": contactStatusLabel(removed.Active),
			},
		})
	}
	http.Redirect(w, r, "/app/kontakte?contact=deleted", http.StatusSeeOther)
}

func contactMessage(status string) (string, bool) {
	switch status {
	case "saved":
		return "Kontakt gespeichert.", true
	case "deleted":
		return "Kontakt deaktiviert.", true
	case "invalid":
		return "Bitte Art, Name/Firma und Kontaktdaten prüfen.", false
	case "error":
		return "Der Kontakt konnte nicht gespeichert werden.", false
	default:
		return "", false
	}
}

func (a *app) managedContactViews(tenantSlug string, includeInactive bool) []managedContactView {
	if a == nil || a.contactStore == nil {
		return nil
	}
	items := a.contactStore.ListTenant(tenantSlug, includeInactive)
	views := make([]managedContactView, 0, len(items))
	for _, item := range items {
		views = append(views, managedContactViewFrom(item))
	}
	return views
}

func (a *app) serviceContactOptions(tenantSlug string) []contactOptionView {
	if a == nil || a.contactStore == nil {
		return nil
	}
	items := a.contactStore.ListTenant(tenantSlug, false)
	options := []contactOptionView{}
	for _, item := range items {
		email := normalizeEmail(item.Email)
		if email == "" {
			continue
		}
		label := managedContactDisplayName(item)
		if item.Kind != "" {
			label += " · " + item.Kind
		}
		options = append(options, contactOptionView{Email: email, Label: label})
	}
	return options
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
	manageIssuePreview := []issueView{}
	canManageIssues := hasCapability(role, capabilityManageIssues)
	canCreateIssue := canCreateResidentIssue(role)
	totalIssueCount := 0
	openIssueCount := 0
	urgentIssueCount := 0
	if boardOnly && !canManageIssues {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	filters := issueBoardFiltersFromQuery(r.URL.Query())
	if a.issueStore != nil {
		allTenantIssues := a.issueStore.ListTenant(tenant.Slug)
		totalIssueCount = len(allTenantIssues)
		openIssueCount = issueOpenCount(allTenantIssues)
		urgentIssueCount = issuePriorityCount(allTenantIssues, issuePriorityUrgent)
		if !boardOnly {
			if canManageIssues {
				issues = a.issueViewsForActor(tenant.Slug, a.issueStore.ListAuthor(tenant.Slug, email), role, email)
			} else {
				issues = a.issueViewsForActor(tenant.Slug, a.visibleIssuesForActor(tenant.Slug, email, role), role, email)
			}
		}
		if canManageIssues {
			filteredIssues := filterIssueBoard(allTenantIssues, filters)
			if boardOnly {
				manageIssues = a.issueViewsForActor(tenant.Slug, filteredIssues, role, email)
			}
			previewIssues := filterIssueBoard(allTenantIssues, issueBoardFilterView{Sort: "updated"})
			if len(previewIssues) > 3 {
				previewIssues = previewIssues[:3]
			}
			manageIssuePreview = a.issueViewsForActor(tenant.Slug, previewIssues, role, email)
		}
	}
	msg, msgOK := issueMessage(r.URL.Query().Get("issue"))
	calendarFeedURL := ""
	if token, err := a.calendarFeedToken(email, tenant.Slug); err == nil {
		calendarFeedURL = a.publicBaseURL(r, tenant) + "/calendar/" + url.PathEscape(token) + ".ics"
	}
	serviceContacts := a.serviceContactOptions(tenant.Slug)
	a.render(w, "issues", map[string]any{
		"Title":                      "Anliegen",
		"Tenant":                     tenant,
		"Email":                      email,
		"DisplayName":                profile.DisplayName(),
		"Initials":                   profile.Initials(),
		"Role":                       role,
		"IsAdmin":                    isAdmin,
		"CanSeeParking":              isAdmin || profile.HasPermission(permissionParking),
		"CanManageAnnouncements":     canManageAnnouncements(role),
		"CanManageIssues":            canManageIssues,
		"CanCreateIssue":             canCreateIssue,
		"ActivePage":                 "issues",
		"BoardOnly":                  boardOnly,
		"BoardAction":                issueBoardAction(boardOnly),
		"CalendarFeedURL":            calendarFeedURL,
		"HasCalendarFeedURL":         calendarFeedURL != "",
		"ServiceProviderContacts":    serviceContacts,
		"HasServiceProviderContacts": len(serviceContacts) > 0,
		"BoardFilters":               issueBoardFilterOptions(filters),
		"Issues":                     issues,
		"HasIssues":                  len(issues) > 0,
		"IssueCount":                 len(issues),
		"TotalIssueCount":            totalIssueCount,
		"OpenIssueCount":             openIssueCount,
		"UrgentIssueCount":           urgentIssueCount,
		"IssuesEmpty":                emptyState("Noch kein Anliegen", "Nach dem Absenden erscheint das Anliegen hier mit Status und Rückfragen."),
		"ManageIssues":               manageIssues,
		"HasManageIssues":            len(manageIssues) > 0,
		"ManageIssuePreview":         manageIssuePreview,
		"HasManageIssuePreview":      len(manageIssuePreview) > 0,
		"ManageIssuePreviewCount":    len(manageIssuePreview),
		"ManageIssuesEmpty":          emptyState("Keine Anliegen im Haus", "Sobald ein Anliegen gemeldet wird, erscheint es hier für die Bearbeitung."),
		"IssueMsg":                   msg,
		"IssueOK":                    msgOK,
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
		return "Anhänge konnten nicht übernommen werden. Erlaubt sind Bilddateien oder PDF bis 10 MB, maximal 10 Dateien.", false
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
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !canCreateResidentIssue(role) {
		http.Error(w, "Dieser Zugang kann keine neuen Anliegen anlegen.", http.StatusForbidden)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxIssueAttachmentFormBytes)
	if err := r.ParseMultipartForm(maxAttachmentBytes); err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	now := time.Now()
	item, err := issueFromForm(r, tenant.Slug, profile, now)
	if err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	attachmentHeaders, err := issueAttachmentHeaders(r)
	if err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=photo", http.StatusSeeOther)
		return
	}
	var uploaded []attachmentRecord
	if len(attachmentHeaders) > 0 {
		if a.attachmentStore == nil {
			http.Redirect(w, r, "/app/anliegen?issue=photo", http.StatusSeeOther)
			return
		}
		uploaded, err = a.attachmentStore.CreateUploaded(tenant.Slug, "issue", item.ID, email, uploadedFilesFromHeaders(attachmentHeaders), now)
		if err != nil {
			http.Redirect(w, r, "/app/anliegen?issue=photo", http.StatusSeeOther)
			return
		}
	}
	if a.issueStore != nil {
		created, err := a.issueStore.Create(item)
		if err != nil {
			for _, attachment := range uploaded {
				_, _, _ = a.attachmentStore.Delete(tenant.Slug, attachment.ID, time.Now())
			}
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
	r.Body = http.MaxBytesReader(w, r.Body, maxIssueAttachmentFormBytes)
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))), "multipart/form-data") {
		if err := r.ParseMultipartForm(maxAttachmentBytes); err != nil {
			http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
			return
		}
	} else if err := r.ParseForm(); err != nil {
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
	canServiceComment := isServiceProviderRole(role) && issueAssignedToActor(existing, email) && issueIsOpen(existing)
	if readOnly || (!canManage && !isOwner && !canServiceComment) {
		http.Error(w, "Dieser Kommentar ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	body := strings.TrimSpace(r.FormValue("body"))
	if body == "" || len([]rune(body)) > 3000 {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	attachmentHeaders, err := issueAttachmentHeaders(r)
	if err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=photo", http.StatusSeeOther)
		return
	}
	commentID, err := randomToken(10)
	if err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=error", http.StatusSeeOther)
		return
	}
	var uploaded []attachmentRecord
	if len(attachmentHeaders) > 0 {
		if a.attachmentStore == nil {
			http.Redirect(w, r, "/app/anliegen?issue=photo", http.StatusSeeOther)
			return
		}
		uploaded, err = a.attachmentStore.CreateUploaded(tenant.Slug, "issue-comment", commentID, email, uploadedFilesFromHeaders(attachmentHeaders), time.Now())
		if err != nil {
			http.Redirect(w, r, "/app/anliegen?issue=photo", http.StatusSeeOther)
			return
		}
	}
	profile := a.profileForTenant(email, tenant.Slug)
	updated, ok, err := a.issueStore.AddComment(tenant.Slug, id, issueComment{
		ID:          commentID,
		AuthorEmail: email,
		AuthorName:  profile.DisplayName(),
		Body:        body,
		CreatedAt:   time.Now(),
	})
	if err != nil {
		for _, attachment := range uploaded {
			_, _, _ = a.attachmentStore.Delete(tenant.Slug, attachment.ID, time.Now())
		}
		log.Printf("issue comment failed for %s/%s: %v", tenant.Slug, id, err)
		http.Redirect(w, r, "/app/anliegen?issue=error", http.StatusSeeOther)
		return
	}
	if !ok {
		for _, attachment := range uploaded {
			_, _, _ = a.attachmentStore.Delete(tenant.Slug, attachment.ID, time.Now())
		}
		http.Redirect(w, r, "/app/anliegen?issue=missing", http.StatusSeeOther)
		return
	}
	a.notifyIssueUpdated(tenant, updated, email, "Neuer Kommentar zu Anliegen \""+updated.Title+"\"")
	http.Redirect(w, r, "/app/anliegen?issue=updated", http.StatusSeeOther)
}

func (a *app) deleteIssueComment(w http.ResponseWriter, r *http.Request) {
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
	commentID := strings.TrimSpace(r.FormValue("comment_id"))
	issue, comment, found := a.issueCommentTarget(tenant.Slug, commentID)
	if !found {
		http.Redirect(w, r, "/app/anliegen?issue=missing", http.StatusSeeOther)
		return
	}
	if !a.canDeleteIssueComment(tenant.Slug, issue, comment, email, role) {
		http.Error(w, "Dieser Kommentar kann mit diesem Zugang nicht gelöscht werden.", http.StatusForbidden)
		return
	}
	updated, deleted, err := a.issueStore.DeleteComment(tenant.Slug, issue.ID, comment.ID, time.Now())
	if err != nil {
		log.Printf("issue comment delete failed for %s/%s/%s: %v", tenant.Slug, issue.ID, comment.ID, err)
		http.Redirect(w, r, "/app/anliegen?issue=error", http.StatusSeeOther)
		return
	}
	if !deleted {
		http.Redirect(w, r, "/app/anliegen?issue=missing", http.StatusSeeOther)
		return
	}
	if a.attachmentStore != nil {
		for _, attachment := range a.attachmentStore.ListEntity(tenant.Slug, "issue-comment", comment.ID) {
			_, _, _ = a.attachmentStore.Delete(tenant.Slug, attachment.ID, time.Now())
		}
	}
	a.notifyIssueUpdated(tenant, updated, email, "Kommentar zu Anliegen \""+updated.Title+"\" gelöscht")
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
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
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
	serviceProposal, serviceProposalProvided, err := issueServiceProposalFromForm(r.Form)
	if err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	estimateAmount, estimateNote, estimateProvided, err := issueEstimateFromForm(r.Form)
	if err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	estimateHeaders, err := attachmentFormHeaders(r, 1, "estimate_attachment")
	if err != nil {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	if len(estimateHeaders) > 0 {
		estimateProvided = true
		if a.attachmentStore == nil {
			http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
			return
		}
	}
	if !canManage {
		canServiceAct := isServiceProviderRole(role) && issueAssignedToActor(existing, email) && issueIsOpen(existing)
		if canServiceAct {
			existingStatus := normalizeIssueStatus(existing.Status)
			if existingStatus == "" {
				existingStatus = issueStatusOpen
			}
			statusUnchanged := existingStatus == status
			if readOnly || r.FormValue("priority") != "" || r.FormValue("assignee_email") != "" || (!statusUnchanged && !canServiceProviderTransition(existing.Status, status)) {
				http.Error(w, "Dieser Statuswechsel ist der Verwaltung vorbehalten.", http.StatusForbidden)
				return
			}
		} else {
			if readOnly || !isOwner || r.FormValue("priority") != "" || r.FormValue("assignee_email") != "" || serviceProposalProvided || estimateProvided || !canResidentTransition(existing.Status, status) {
				http.Error(w, "Dieser Statuswechsel ist der Verwaltung vorbehalten.", http.StatusForbidden)
				return
			}
		}
		priority = normalizeIssuePriority(existing.Priority)
		assignee = normalizeEmail(existing.AssigneeEmail)
	}
	if priority == "" {
		http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	var uploadedEstimates []attachmentRecord
	if len(estimateHeaders) > 0 {
		uploadedEstimates, err = a.attachmentStore.CreateUploaded(tenant.Slug, "issue-estimate", id, email, uploadedFilesFromHeaders(estimateHeaders), time.Now())
		if err != nil {
			log.Printf("issue estimate upload failed for %s/%s: %v", tenant.Slug, id, err)
			http.Redirect(w, r, "/app/anliegen?issue=invalid", http.StatusSeeOther)
			return
		}
	}
	updated, _, err := a.issueStore.UpdateWorkflow(tenant.Slug, id, issueWorkflowUpdate{
		Status:                status,
		Priority:              priority,
		AssigneeEmail:         assignee,
		ServiceProposal:       serviceProposal,
		UpdateServiceProposal: serviceProposalProvided,
		EstimateAmountCents:   estimateAmount,
		EstimateNote:          estimateNote,
		UpdateEstimate:        estimateProvided,
		ActorEmail:            email,
		ActorName:             profile.DisplayName(),
		ChangedAt:             time.Now(),
	})
	if err != nil {
		for _, attachment := range uploadedEstimates {
			_, _, _ = a.attachmentStore.Delete(tenant.Slug, attachment.ID, time.Now())
		}
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
	if estimateProvided {
		details := map[string]string{
			"has_file": strconv.FormatBool(len(uploadedEstimates) > 0),
		}
		if updated.EstimateAmountCents > 0 {
			details["estimate_amount"] = formatIssueEstimateAmount(updated.EstimateAmountCents)
		}
		if len(uploadedEstimates) > 0 {
			details["file_count"] = strconv.Itoa(len(uploadedEstimates))
		}
		a.recordAudit(auditEvent{
			TenantSlug: tenant.Slug,
			ActorEmail: email,
			ActorRole:  role,
			Action:     auditActionIssueEstimate,
			TargetType: "issue",
			TargetID:   updated.ID,
			Summary:    "Kostenvoranschlag aktualisiert",
			Details:    details,
		})
	}
	a.handleIssueServiceAssignmentChange(r, tenant, existing, updated, email, role)
	a.notifyIssueUpdated(tenant, updated, email, "Anliegen \""+updated.Title+"\" aktualisiert")
	http.Redirect(w, r, "/app/anliegen?issue=updated", http.StatusSeeOther)
}

func (a *app) handleIssueServiceAssignmentChange(r *http.Request, tenant tenantConfig, before residentIssue, after residentIssue, actorEmail string, actorRole string) {
	oldAssignee := normalizeEmail(before.AssigneeEmail)
	newAssignee := normalizeEmail(after.AssigneeEmail)
	if oldAssignee != "" && oldAssignee != newAssignee && a.isServiceProviderPrincipal(tenant.Slug, oldAssignee) {
		a.recordAudit(auditEvent{
			TenantSlug: tenant.Slug,
			ActorEmail: actorEmail,
			ActorRole:  actorRole,
			Action:     auditActionIssueServiceDrop,
			TargetType: "issue",
			TargetID:   after.ID,
			Summary:    "Dienstleister-Zugriff entzogen",
			Details: map[string]string{
				"service_email": oldAssignee,
				"issue_title":   after.Title,
			},
		})
	}
	if newAssignee == "" || newAssignee == oldAssignee {
		return
	}
	if !a.shouldInviteServiceProvider(tenant.Slug, newAssignee) {
		return
	}
	createdInvite, err := a.ensureServiceProviderInvite(tenant.Slug, newAssignee)
	if err != nil {
		log.Printf("service provider invite persistence failed for %s/%s: %v", tenant.Slug, redactedEmail(newAssignee), err)
		a.recordIssueServiceInviteAudit(tenant, after, actorEmail, actorRole, newAssignee, createdInvite, "nicht gespeichert")
		return
	}
	mailStatus := "verschickt"
	if err := a.sendServiceProviderMagicLink(r, tenant, after, newAssignee); err != nil {
		log.Printf("service provider magic link failed for %s/%s: %v", tenant.Slug, redactedEmail(newAssignee), err)
		mailStatus = "nicht zugestellt"
	}
	a.recordIssueServiceInviteAudit(tenant, after, actorEmail, actorRole, newAssignee, createdInvite, mailStatus)
}

func (a *app) recordIssueServiceInviteAudit(tenant tenantConfig, issue residentIssue, actorEmail string, actorRole string, serviceEmail string, createdInvite bool, mailStatus string) {
	inviteStatus := "bestehend"
	if createdInvite {
		inviteStatus = "neu"
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  actorRole,
		Action:     auditActionIssueServiceAdd,
		TargetType: "issue",
		TargetID:   issue.ID,
		Summary:    "Dienstleister eingeladen",
		Details: map[string]string{
			"service_email": serviceEmail,
			"issue_title":   issue.Title,
			"invite":        inviteStatus,
			"mail_status":   mailStatus,
		},
	})
}

func (a *app) shouldInviteServiceProvider(tenantSlug string, email string) bool {
	if email == "" {
		return false
	}
	if profile, ok := a.directoryProfile(email); ok && profile.HasTenant(tenantSlug) {
		return isServiceProviderRole(profile.ForTenant(tenantSlug).Role)
	}
	return true
}

func (a *app) isServiceProviderPrincipal(tenantSlug string, email string) bool {
	if profile, ok := a.directoryProfile(email); ok && profile.HasTenant(tenantSlug) {
		return isServiceProviderRole(profile.ForTenant(tenantSlug).Role)
	}
	return false
}

func (a *app) ensureServiceProviderInvite(tenantSlug string, email string) (bool, error) {
	if profile, ok := a.directoryProfile(email); ok && profile.HasTenant(tenantSlug) {
		return false, nil
	}
	if a.inviteStore == nil {
		return false, fmt.Errorf("invite store not configured")
	}
	profile := userProfile{
		Email:       email,
		Role:        roleServiceProvider,
		Status:      "Eingeladen",
		Tenants:     []string{tenantSlug},
		AuthMethods: defaultAuthMethods(),
	}
	added, err := a.inviteStore.Add(profile)
	if err != nil {
		return false, err
	}
	if added {
		return true, nil
	}
	existing, ok := a.inviteStore.Get(email)
	if ok && existing.HasTenant(tenantSlug) && isServiceProviderRole(existing.ForTenant(tenantSlug).Role) {
		return false, nil
	}
	return false, nil
}

func (a *app) sendServiceProviderMagicLink(r *http.Request, tenant tenantConfig, issue residentIssue, email string) error {
	if a == nil || a.tokens == nil {
		return fmt.Errorf("login tokens not configured")
	}
	token, err := randomToken(32)
	if err != nil {
		return err
	}
	redirectPath := "/app/anliegen#issue-" + url.PathEscape(issue.ID)
	a.tokens.PutWithRedirect(token, email, tenant.Slug, 15*time.Minute, redirectPath)
	link := a.publicBaseURL(r, tenant) + "/auth/verify?token=" + url.QueryEscape(token)
	return a.mailer.SendMagicLink(email, link)
}

func (a *app) notifyIssueCreated(tenant tenantConfig, issue residentIssue) {
	recipients := a.issueManagerEmails(tenant.Slug)
	a.notify(portalNotification{
		Event:      notificationEventIssue,
		Tenant:     tenant,
		Recipients: recipients,
		ActorEmail: issue.AuthorEmail,
		Subject:    "Neues Anliegen: " + issue.Title,
		ActionText: "Anliegen öffnen",
		ActionURL:  tenant.PublicURL("/app/anliegen#issue-" + url.PathEscape(issue.ID)),
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
		ActionText: "Anliegen öffnen",
		ActionURL:  tenant.PublicURL("/app/anliegen#issue-" + url.PathEscape(issue.ID)),
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
		ActionText: "Aushang öffnen",
		ActionURL:  tenant.PublicURL("/app/announcements#announcement-" + url.PathEscape(item.ID)),
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
				ActionURL:  tenant.PublicURL("/app/abstimmungen#ballot-" + url.PathEscape(item.ID)),
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
		if profile.HasTenant(tenantSlug) && !isServiceProviderRole(profile.ForTenant(tenantSlug).Role) {
			recipients = append(recipients, email)
		}
	}
	for email := range a.admins {
		recipients = append(recipients, email)
	}
	if a.inviteStore != nil {
		for _, profile := range a.inviteStore.List() {
			if profile.HasTenant(tenantSlug) && !isServiceProviderRole(profile.ForTenant(tenantSlug).Role) {
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
	if denyServiceProviderArea(w, role) {
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	if !hasCapability(role, capabilityPlatformAdmin) && !profile.HasPermission(permissionParking) {
		http.NotFound(w, r)
		return
	}
	isAdmin := hasCapability(role, capabilityPlatformAdmin)
	telemetry := a.parkingTelemetry(r.Context(), tenant)
	parkingMsg, parkingOK := parkingMessage(r.URL.Query().Get("month"), r.URL.Query().Get("reminder"))
	accounting := a.parkingAccounting(r.Context(), tenant)
	accounting.Months = a.hydrateParkingMonths(tenant.Slug, email, role, accounting.Months)
	a.render(w, "parking", map[string]any{
		"Title":                    "Parkplatznutzung",
		"Tenant":                   tenant,
		"Email":                    email,
		"DisplayName":              profile.DisplayName(),
		"Initials":                 profile.Initials(),
		"Role":                     role,
		"IsAdmin":                  isAdmin,
		"CanManageParkingPayments": hasCapability(role, capabilityManageUsers) || hasCapability(role, capabilityManageParking),
		"CanMarkParkingPayment":    isAdmin || hasCapability(role, capabilityManageUsers) || hasCapability(role, capabilityManageParking) || profile.HasPermission(permissionParking),
		"CanSeeParking":            true,
		"ActivePage":               "parking",
		"Telemetry":                telemetry,
		"Accounting":               accounting,
		"ParkingMsg":               parkingMsg,
		"ParkingOK":                parkingOK,
		"TodayInput":               time.Now().In(time.Local).Format("2006-01-02"),
		"StatementYear":            time.Now().In(time.Local).Year(),
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
	view.Summary = a.hydrateParkingMonth(tenant.Slug, email, role, view.Summary)
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

func (a *app) hydrateParkingMonths(tenantSlug string, email string, role string, months []parkingMonthView) []parkingMonthView {
	for i := range months {
		months[i] = a.hydrateParkingMonth(tenantSlug, email, role, months[i])
	}
	return months
}

func (a *app) hydrateParkingMonth(tenantSlug string, email string, role string, month parkingMonthView) parkingMonthView {
	if a == nil || a.attachmentStore == nil || month.Month == "" {
		return month
	}
	attachments := a.attachmentViewsForEntity(tenantSlug, "parking", month.Month, email, role)
	if len(attachments) == 0 {
		return month
	}
	month.Attachments = attachments
	month.HasAttachments = true
	return month
}

func (a *app) parkingStatement(w http.ResponseWriter, r *http.Request) {
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
	year, ok := parkingStatementYear(r.PathValue("year"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	actor := a.profileForTenant(email, tenant.Slug)
	targetEmail := normalizeEmail(r.URL.Query().Get("user"))
	if targetEmail == "" {
		targetEmail = email
	}
	target, ok := a.parkingStatementTarget(tenant.Slug, email, role, actor, targetEmail)
	if !ok {
		http.NotFound(w, r)
		return
	}
	statement := a.buildParkingStatement(r.Context(), tenant, target, year)
	filename := "parkplatzabrechnung-" + strconv.Itoa(year) + "-" + safeFilenamePart(target.Email) + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	if err := writeParkingStatementCSV(w, statement); err != nil {
		log.Printf("parking statement export failed for %s/%d: %v", tenant.Slug, year, err)
	}
}

func parkingStatementYear(raw string) (int, bool) {
	year, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || year < 2000 || year > 2100 {
		return 0, false
	}
	return year, true
}

func (a *app) parkingStatementTarget(tenantSlug string, actorEmail string, actorRole string, actor userProfile, targetEmail string) (userProfile, bool) {
	tenantSlug = normalizeSlug(tenantSlug)
	targetEmail = normalizeEmail(targetEmail)
	if tenantSlug == "" || targetEmail == "" {
		return userProfile{}, false
	}
	isManager := hasCapability(actorRole, capabilityManageUsers) || hasCapability(actorRole, capabilityPlatformAdmin)
	if targetEmail != normalizeEmail(actorEmail) && !isManager {
		return userProfile{}, false
	}
	target := a.profileForTenant(targetEmail, tenantSlug)
	if !target.HasTenant(tenantSlug) {
		return userProfile{}, false
	}
	targetIsAdmin := hasCapability(target.Role, capabilityPlatformAdmin)
	targetCanPark := targetIsAdmin || target.HasPermission(permissionParking)
	if !targetCanPark {
		return userProfile{}, false
	}
	if !isManager {
		actorCanPark := hasCapability(actorRole, capabilityPlatformAdmin) || actor.HasPermission(permissionParking)
		if !actorCanPark || targetEmail != normalizeEmail(actorEmail) {
			return userProfile{}, false
		}
	}
	return target, true
}

func (a *app) buildParkingStatement(ctx context.Context, tenant tenantConfig, user userProfile, year int) parkingStatementView {
	seedCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	if err := a.seedParkingHistory(seedCtx, tenant, a.parkingHistoryStart, time.Now()); err != nil && tenant.HA.Configured() {
		log.Printf("parking history seed failed for %s: %v", tenant.Slug, err)
	}
	data := a.parkingStore.TenantData(tenant.Slug)
	months := []parkingMonthView{}
	totalKWh := 0.0
	energyCost := 0.0
	gridCost := 0.0
	baseFee := 0.0
	totalCost := 0.0
	for _, month := range calculateParkingMonths(data, time.Now(), time.Local) {
		if !strings.HasPrefix(month.Month, strconv.Itoa(year)+"-") {
			continue
		}
		months = append(months, month)
		totalKWh += month.KWhValue
		energyCost += month.EnergyCostValue
		gridCost += month.GridCostValue
		baseFee += month.BaseFeeValue
		totalCost += month.TotalCostValue
	}
	return parkingStatementView{
		Tenant:       tenant,
		User:         user,
		Year:         year,
		GeneratedAt:  formatLocalDateTime(time.Now()),
		GridFeeLabel: parkingStatementTariffLabel(data.Settings),
		Months:       months,
		HasMonths:    len(months) > 0,
		TotalKWh:     formatKWh(totalKWh),
		EnergyCost:   formatEUR(energyCost),
		GridCost:     formatEUR(gridCost),
		BaseFee:      formatEUR(baseFee),
		TotalCost:    formatEUR(totalCost),
	}
}

func writeParkingStatementCSV(w io.Writer, statement parkingStatementView) error {
	writer := csv.NewWriter(w)
	writer.Comma = ';'
	rows := [][]string{
		{"WEG Portal Parkplatzabrechnung"},
		{"Gebäude", statement.Tenant.Name},
		{"Adresse", statement.Tenant.Address},
		{"Person", statement.User.DisplayName()},
		{"E-Mail", statement.User.Email},
		{"Jahr", strconv.Itoa(statement.Year)},
		{"Erstellt", statement.GeneratedAt},
		{"Tarif", statement.GridFeeLabel},
		{},
		{"Monat", "Zeitraum", "kWh", "aWATTar Ø", "Effektivpreis", "Strom", "Netzgeb.", "Basis", "Summe", "Status"},
	}
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	for _, month := range statement.Months {
		if err := writer.Write([]string{month.MonthLabel, month.PeriodLabel, month.KWh, month.AverageAwattar, month.EffectivePrice, month.EnergyCost, month.GridCost, month.BaseFee, month.TotalCost, month.PaidLabel}); err != nil {
			return err
		}
	}
	if err := writer.Write([]string{"Gesamt", "", statement.TotalKWh, "", "", statement.EnergyCost, statement.GridCost, statement.BaseFee, statement.TotalCost, ""}); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

func safeFilenamePart(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}
	out := strings.Trim(b.String(), "-.")
	if out == "" {
		return "person"
	}
	return out
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
	tariff, err := parkingTariffFromForm(r.Form)
	if err != nil {
		http.Redirect(w, r, "/app/parking/settings?settings=invalid", http.StatusSeeOther)
		return
	}
	if err := a.parkingStore.UpsertTariff(tenant.Slug, tariff); err != nil {
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
			"effective_from": formatParkingTariffDate(tariff.EffectiveFrom),
			"grid_fee":       formatEURPerKWh(tariff.GridFeeEURPerKWh),
			"base_fee":       formatEUR(tariff.BaseFeeEUR),
		},
	})
	http.Redirect(w, r, "/app/parking/settings?settings=saved", http.StatusSeeOther)
}

func parkingTariffFromForm(values url.Values) (parkingTariff, error) {
	effectiveFrom := normalizeParkingTariffDate(values.Get("effective_from"))
	if effectiveFrom == "" {
		effectiveFrom = time.Now().In(time.Local).Format("2006-01-02")
	}
	gridFee, err := parseDecimal(values.Get("grid_fee_eur_per_kwh"))
	if err != nil || gridFee < 0 || gridFee > 5 {
		return parkingTariff{}, fmt.Errorf("invalid grid fee")
	}
	baseFee := 0.0
	if strings.TrimSpace(values.Get("base_fee_eur")) != "" {
		baseFee, err = parseDecimal(values.Get("base_fee_eur"))
		if err != nil || baseFee < 0 || baseFee > 5000 {
			return parkingTariff{}, fmt.Errorf("invalid base fee")
		}
	}
	return normalizeParkingTariff(parkingTariff{
		EffectiveFrom:    effectiveFrom,
		GridFeeEURPerKWh: gridFee,
		BaseFeeEUR:       baseFee,
	}), nil
}

func (a *app) updateParkingMonth(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	actorEmail, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	actor := a.profileForTenant(actorEmail, tenant.Slug)
	canManagePayment := hasCapability(role, capabilityManageUsers) || hasCapability(role, capabilityManageParking) || hasCapability(role, capabilityPlatformAdmin)
	canMarkPayment := canManagePayment || actor.HasPermission(permissionParking)
	if !canMarkPayment {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	month := strings.TrimSpace(r.FormValue("month"))
	if _, err := time.Parse("2006-01", month); err != nil {
		http.Redirect(w, r, "/app/parking?month=invalid", http.StatusSeeOther)
		return
	}
	payment, err := parkingPaymentFromForm(r.Form, actorEmail)
	if err != nil {
		http.Redirect(w, r, "/app/parking?month=invalid", http.StatusSeeOther)
		return
	}
	if !payment.Paid && !canManagePayment {
		http.Error(w, "Dieser Bereich ist Admins vorbehalten.", http.StatusForbidden)
		return
	}
	attachmentHeaders, err := attachmentFormHeaders(r, maxIssueAttachmentCount, "attachments")
	if err != nil {
		http.Redirect(w, r, "/app/parking?month=invalid", http.StatusSeeOther)
		return
	}
	var uploaded []attachmentRecord
	if len(attachmentHeaders) > 0 {
		if a.attachmentStore == nil {
			http.Redirect(w, r, "/app/parking?month=invalid", http.StatusSeeOther)
			return
		}
		uploaded, err = a.attachmentStore.CreateUploaded(tenant.Slug, "parking", month, actorEmail, uploadedFilesFromHeaders(attachmentHeaders), time.Now())
		if err != nil {
			log.Printf("parking attachment upload failed for %s/%s/%s: %v", tenant.Slug, month, redactedEmail(actorEmail), err)
			http.Redirect(w, r, "/app/parking?month=invalid", http.StatusSeeOther)
			return
		}
	}
	if err := a.parkingStore.SetMonthPayment(tenant.Slug, month, payment); err != nil {
		for _, attachment := range uploaded {
			_, _, _ = a.attachmentStore.Delete(tenant.Slug, attachment.ID, time.Now())
		}
		log.Printf("parking month save failed for %s: %v", tenant.Slug, err)
		http.Error(w, "Could not save parking month", http.StatusInternalServerError)
		return
	}
	details := map[string]string{
		"month": formatMonthLabel(month, time.Local),
		"paid":  paidLabel(payment.Paid),
	}
	if payment.Paid {
		if !payment.PaidAt.IsZero() {
			details["paid_at"] = formatLocalDate(payment.PaidAt.In(time.Local))
		}
		if payment.PaidBy != "" {
			details["paid_by"] = payment.PaidBy
		}
		if payment.PaymentMethod != "" {
			details["payment_method"] = payment.PaymentMethod
		}
		if payment.PaymentReference != "" {
			details["payment_reference"] = payment.PaymentReference
		}
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionParkingMonth,
		TargetType: "parking",
		TargetID:   month,
		Summary:    "Monatsstatus geändert",
		Details:    details,
	})
	http.Redirect(w, r, "/app/parking?month=saved", http.StatusSeeOther)
}

func (a *app) sendParkingReminders(w http.ResponseWriter, r *http.Request) {
	tenant := a.tenantForRequest(r)
	actorEmail, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !hasCapability(role, capabilityManageUsers) && !hasCapability(role, capabilityManageParking) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	returnToAccess := r.FormValue("return_to") == "parking_access"
	sent := a.sendParkingPaymentReminders(tenant, actorEmail, role, time.Now())
	status := "none"
	if sent > 0 {
		status = "sent"
	}
	if returnToAccess {
		http.Redirect(w, r, "/app/settings/parking-access?parking_access=reminder_"+status, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/parking?reminder="+status, http.StatusSeeOther)
}

func (a *app) sendParkingPaymentReminders(tenant tenantConfig, actorEmail string, actorRole string, now time.Time) int {
	if a == nil || a.parkingStore == nil {
		return 0
	}
	if now.IsZero() {
		now = time.Now()
	}
	data := a.parkingStore.TenantData(tenant.Slug)
	months := calculateParkingMonths(data, now, time.Local)
	if len(months) == 0 {
		return 0
	}
	sentTotal := 0
	for _, row := range a.userRows(tenant.Slug) {
		if !row.ParkingChecked {
			continue
		}
		recipientMonths := parkingReminderMonths(months, data.Months, row.Email)
		if len(recipientMonths) == 0 {
			continue
		}
		monthIDs := make([]string, 0, len(recipientMonths))
		balance := 0.0
		lines := []string{
			"Für " + tenant.Address + " sind Parkplatz-Abrechnungen überfällig.",
			"",
			"Überfällige Monate:",
		}
		for _, month := range recipientMonths {
			monthIDs = append(monthIDs, month.Month)
			balance += month.TotalCostValue
			lines = append(lines, month.MonthLabel+": "+month.TotalCost)
		}
		lines = append(lines, "", "Offener Betrag: "+formatEUR(balance))
		actionURL := tenant.PublicURL("/app/parking")
		if len(recipientMonths) > 0 {
			actionURL = tenant.PublicURL("/app/parking#parking-month-" + url.PathEscape(recipientMonths[0].Month))
		}
		sent := a.notify(portalNotification{
			Event:      notificationEventPayment,
			Tenant:     tenant,
			Recipients: []string{row.Email},
			ActorEmail: actorEmail,
			Subject:    "Zahlungserinnerung Parkplatznutzung",
			ActionText: "Parkplatzabrechnung öffnen",
			ActionURL:  actionURL,
			Lines:      lines,
		})
		if len(sent) == 0 {
			continue
		}
		if err := a.parkingStore.MarkPaymentReminderSent(tenant.Slug, monthIDs, sent, now); err != nil {
			log.Printf("parking reminder mark failed for %s/%s: %v", tenant.Slug, redactedEmail(row.Email), err)
			continue
		}
		a.recordAudit(auditEvent{
			TenantSlug: tenant.Slug,
			ActorEmail: actorEmail,
			ActorRole:  actorRole,
			Action:     auditActionParkingReminder,
			TargetType: "parking",
			TargetID:   row.Email,
			Summary:    "Zahlungserinnerung gesendet",
			Details: map[string]string{
				"recipients": strconv.Itoa(len(sent)),
				"balance":    formatEUR(balance),
				"month":      strings.Join(monthIDs, ", "),
			},
		})
		sentTotal += len(sent)
	}
	return sentTotal
}

func parkingReminderMonths(months []parkingMonthView, states map[string]parkingMonthState, email string) []parkingMonthView {
	email = normalizeEmail(email)
	if email == "" {
		return nil
	}
	out := []parkingMonthView{}
	for _, month := range months {
		if !month.Overdue {
			continue
		}
		state := normalizeParkingMonthState(states[month.Month])
		if _, ok := state.ReminderSentAt[email]; ok {
			continue
		}
		out = append(out, month)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Month < out[j].Month
	})
	return out
}

func parkingPaymentFromForm(values url.Values, actorEmail string) (parkingMonthState, error) {
	paid := parseBool(values.Get("paid"))
	state := parkingMonthState{Paid: paid}
	if !paid {
		return state, nil
	}
	paidAt := time.Now().In(time.Local)
	if raw := strings.TrimSpace(values.Get("paid_at")); raw != "" {
		parsed, err := parseParkingPaidAt(raw)
		if err != nil {
			return parkingMonthState{}, err
		}
		paidAt = parsed
	}
	state.PaidAt = paidAt.UTC()
	state.PaidBy = normalizeEmail(actorEmail)
	state.PaymentMethod = cleanParkingPaymentField(values.Get("payment_method"))
	state.PaymentReference = cleanParkingPaymentField(values.Get("payment_reference"))
	return state, nil
}

func parseParkingPaidAt(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("missing paid date")
	}
	if t, err := time.ParseInLocation("2006-01-02", raw, time.Local); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04", raw, time.Local); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid paid date")
}

func parkingMessage(monthStatus string, reminderStatus string) (string, bool) {
	switch reminderStatus {
	case "sent":
		return "Zahlungserinnerungen gesendet.", true
	case "none":
		return "Keine überfälligen offenen Parkplatzbeträge mit aktiver Benachrichtigung gefunden.", false
	}
	switch monthStatus {
	case "saved":
		return "Zahlungsstatus gespeichert.", true
	case "invalid":
		return "Bitte Monat und Zahlungsdaten prüfen.", false
	default:
		return "", false
	}
}

func parkingSettingsMessage(status string) (string, bool) {
	switch status {
	case "saved":
		return "Parkplatz-Abrechnung gespeichert.", true
	case "invalid":
		return "Bitte Gültigkeitsdatum, Netzgebühr und Basisgebühr prüfen.", false
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

func denyServiceProviderArea(w http.ResponseWriter, role string) bool {
	if !isServiceProviderRole(role) {
		return false
	}
	http.Error(w, "Dieser Zugang ist nur für zugewiesene Anliegen freigeschaltet.", http.StatusForbidden)
	return true
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
	case roleServiceProvider:
		return []string{"Zugewiesene Anliegen"}
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
	case roleServiceProvider:
		return 6
	default:
		return 7
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

func (a *app) announcementViewsWithReadState(tenantSlug string, items []announcement, now time.Time, includeStatus bool, lastSeen time.Time, actorEmail string, role string) []announcementView {
	views := announcementViewsWithReadState(items, now, includeStatus, lastSeen)
	if a == nil || a.attachmentStore == nil {
		return views
	}
	for i := range views {
		attachments := a.attachmentViewsForEntity(tenantSlug, "announcement", views[i].ID, actorEmail, role)
		if len(attachments) == 0 {
			continue
		}
		views[i].Attachments = attachments
		views[i].HasAttachments = true
	}
	return views
}

func eventViews(items []houseEvent, now time.Time) []houseEventView {
	views := make([]houseEventView, 0, len(items))
	for _, item := range items {
		views = append(views, eventViewFrom(item, now))
	}
	return views
}

func (a *app) eventViews(tenantSlug string, items []houseEvent, now time.Time, actorEmail string, role string) []houseEventView {
	views := eventViews(items, now)
	if a == nil || a.attachmentStore == nil {
		return views
	}
	for i := range views {
		attachments := a.attachmentViewsForEntity(tenantSlug, "event", views[i].ID, actorEmail, role)
		if len(attachments) == 0 {
			continue
		}
		views[i].Attachments = attachments
		views[i].HasAttachments = true
	}
	return views
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

func serviceProviderIssueStatuses() []string {
	return []string{issueStatusProgress, issueStatusDone}
}

func issueServiceProposalFromForm(values url.Values) (string, bool, error) {
	if _, ok := values["service_proposal"]; !ok {
		return "", false, nil
	}
	proposal := strings.TrimSpace(values.Get("service_proposal"))
	if len([]rune(proposal)) > 180 {
		return "", true, fmt.Errorf("service proposal too long")
	}
	return proposal, true, nil
}

func issueEstimateFromForm(values url.Values) (int64, string, bool, error) {
	if values == nil {
		return 0, "", false, nil
	}
	_, amountProvided := values["estimate_amount"]
	_, noteProvided := values["estimate_note"]
	if !amountProvided && !noteProvided {
		return 0, "", false, nil
	}
	amount, err := parseIssueEstimateAmountCents(values.Get("estimate_amount"))
	if err != nil {
		return 0, "", true, err
	}
	note := strings.TrimSpace(values.Get("estimate_note"))
	if len([]rune(note)) > 240 {
		return 0, "", true, fmt.Errorf("estimate note too long")
	}
	return amount, note, true, nil
}

func parseIssueEstimateAmountCents(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	raw = strings.ReplaceAll(raw, " ", "")
	if strings.Count(raw, ",") == 1 && strings.Count(raw, ".") > 0 && strings.LastIndex(raw, ",") > strings.LastIndex(raw, ".") {
		raw = strings.ReplaceAll(raw, ".", "")
		raw = strings.ReplaceAll(raw, ",", ".")
	} else if strings.Count(raw, ",") == 1 && strings.Count(raw, ".") == 0 {
		raw = strings.ReplaceAll(raw, ",", ".")
	}
	return integrations.ParseDecimalCents(raw)
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

func issueAttachmentHeaders(r *http.Request) ([]*multipart.FileHeader, error) {
	return attachmentFormHeaders(r, maxIssueAttachmentCount, "attachments", "photos", "photo")
}

func attachmentFormHeaders(r *http.Request, maxCount int, names ...string) ([]*multipart.FileHeader, error) {
	if r.MultipartForm == nil {
		return nil, nil
	}
	out := []*multipart.FileHeader{}
	for _, name := range names {
		for _, header := range r.MultipartForm.File[name] {
			if header == nil || strings.TrimSpace(header.Filename) == "" || header.Size == 0 {
				continue
			}
			out = append(out, header)
			if maxCount > 0 && len(out) > maxCount {
				return nil, fmt.Errorf("too many attachments")
			}
		}
	}
	return out, nil
}

func parseMaybeMultipartForm(w http.ResponseWriter, r *http.Request, maxBodyBytes int64, maxMemoryBytes int64) error {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))), "multipart/form-data") {
		if r.MultipartForm != nil {
			return nil
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		return r.ParseMultipartForm(maxMemoryBytes)
	}
	if r.Form != nil {
		return nil
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	return r.ParseForm()
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

func issuePriorityCount(items []residentIssue, priority string) int {
	priority = normalizeIssuePriority(priority)
	count := 0
	for _, item := range items {
		if normalizeIssuePriority(item.Priority) == priority {
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
	if isServiceProviderRole(role) {
		return issueIsOpen(item) && issueAssignedToActor(item, email)
	}
	if normalizeEmail(item.AuthorEmail) == email {
		return true
	}
	return normalizeIssueLocation(item.LocationType) == issueLocationCommon && a.actorCanSeeCommonIssues(tenantSlug, email, role)
}

func issueAssignedToActor(item residentIssue, email string) bool {
	return normalizeEmail(item.AssigneeEmail) != "" && normalizeEmail(item.AssigneeEmail) == normalizeEmail(email)
}

func (a *app) canViewAttachment(tenantSlug string, item attachmentRecord, email string, role string) bool {
	tenantSlug = normalizeSlug(tenantSlug)
	if tenantSlug == "" || normalizeSlug(item.TenantSlug) != tenantSlug || normalizeEmail(email) == "" {
		return false
	}
	entityType := normalizeAttachmentEntity(item.EntityType)
	if isServiceProviderRole(role) && entityType != "issue" && entityType != "issue-comment" && entityType != "issue-estimate" {
		return false
	}
	switch entityType {
	case "issue", "issue-estimate":
		if a.issueStore == nil {
			return false
		}
		issue, found := a.issueStore.Get(tenantSlug, item.EntityID)
		return found && a.canViewIssueForActor(tenantSlug, issue, email, role)
	case "issue-comment":
		issue, _, found := a.issueCommentTarget(tenantSlug, item.EntityID)
		return found && a.canViewIssueForActor(tenantSlug, issue, email, role)
	case "document":
		if a.documentStore == nil {
			return false
		}
		doc, found := a.documentStore.Get(tenantSlug, item.EntityID)
		return found && a.canViewDocument(tenantSlug, doc, email, role)
	case "announcement", "event", "building":
		return true
	case "ballot":
		return hasCapability(role, capabilityManageVotes) || hasCapability(role, capabilityVote) || hasCapability(role, capabilityOversight)
	case "handover":
		return canManageHandovers(role)
	case "parking":
		return hasCapability(role, capabilityManageParking) || hasCapability(role, capabilityPlatformAdmin) || a.profileForTenant(email, tenantSlug).HasPermission(permissionParking)
	default:
		return false
	}
}

func (a *app) canDeleteAttachment(tenantSlug string, item attachmentRecord, email string, role string) bool {
	tenantSlug = normalizeSlug(tenantSlug)
	email = normalizeEmail(email)
	if tenantSlug == "" || normalizeSlug(item.TenantSlug) != tenantSlug || email == "" {
		return false
	}
	if hasCapability(role, capabilityPlatformAdmin) || normalizeEmail(item.UploadedBy) == email {
		return true
	}
	switch normalizeAttachmentEntity(item.EntityType) {
	case "issue", "issue-estimate":
		if hasCapability(role, capabilityManageIssues) {
			return true
		}
		if a.issueStore == nil {
			return false
		}
		issue, found := a.issueStore.Get(tenantSlug, item.EntityID)
		return found && normalizeEmail(issue.AuthorEmail) == email
	case "issue-comment":
		if hasCapability(role, capabilityManageIssues) {
			return true
		}
		issue, comment, found := a.issueCommentTarget(tenantSlug, item.EntityID)
		return found && (normalizeEmail(issue.AuthorEmail) == email || normalizeEmail(comment.AuthorEmail) == email)
	case "document":
		return hasCapability(role, capabilityManageDocuments)
	case "announcement":
		return canManageAnnouncements(role)
	case "event":
		return canManageEvents(role)
	case "ballot":
		return hasCapability(role, capabilityManageVotes)
	case "handover":
		return canManageHandovers(role)
	case "parking":
		return hasCapability(role, capabilityManageParking)
	case "building":
		return hasCapability(role, capabilityManageBuilding)
	default:
		return false
	}
}

func (a *app) canDeleteIssueComment(tenantSlug string, issue residentIssue, comment issueComment, email string, role string) bool {
	email = normalizeEmail(email)
	if email == "" || !a.canViewIssueForActor(tenantSlug, issue, email, role) {
		return false
	}
	if hasCapability(role, capabilityManageIssues) || hasCapability(role, capabilityPlatformAdmin) {
		return true
	}
	return normalizeEmail(comment.AuthorEmail) == email
}

func (a *app) issueCommentTarget(tenantSlug string, commentID string) (residentIssue, issueComment, bool) {
	if a == nil || a.issueStore == nil {
		return residentIssue{}, issueComment{}, false
	}
	commentID = strings.TrimSpace(commentID)
	if commentID == "" {
		return residentIssue{}, issueComment{}, false
	}
	for _, issue := range a.issueStore.ListTenant(tenantSlug) {
		for _, comment := range issue.Comments {
			if comment.ID == commentID {
				return issue, comment, true
			}
		}
	}
	return residentIssue{}, issueComment{}, false
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

func (a *app) issueViewsForActor(tenantSlug string, items []residentIssue, role string, actorEmail string) []issueView {
	views := issueViewsForActor(items, role, actorEmail)
	if a == nil {
		return views
	}
	for i := range views {
		attachments := a.legacyIssuePhotoViews(tenantSlug, items[i])
		attachments = append(attachments, a.attachmentViewsForEntity(tenantSlug, "issue", views[i].ID, actorEmail, role)...)
		photoCount := 0
		for _, attachment := range attachments {
			if attachment.IsImage {
				photoCount++
			}
		}
		if len(attachments) > 0 {
			views[i].Attachments = attachments
			views[i].HasAttachments = true
		}
		estimateAttachments := a.attachmentViewsForEntity(tenantSlug, "issue-estimate", views[i].ID, actorEmail, role)
		if len(estimateAttachments) > 0 {
			views[i].EstimateAttachments = estimateAttachments
			views[i].HasEstimateAttachments = true
			views[i].EstimateAttachmentGroup = attachmentGroup{Attachments: estimateAttachments, HasAttachments: true}
			views[i].HasEstimate = true
		}
		views[i].PhotoCount = photoCount
		views[i].HasPhotos = photoCount > 0
		for j := range views[i].Comments {
			comment, found := issueCommentByID(items[i].Comments, views[i].Comments[j].ID)
			if found && a.canDeleteIssueComment(tenantSlug, items[i], comment, actorEmail, role) {
				views[i].Comments[j].CanDelete = true
				views[i].Comments[j].DeleteURL = "/app/anliegen/comment/delete"
			}
			commentAttachments := a.attachmentViewsForEntity(tenantSlug, "issue-comment", views[i].Comments[j].ID, actorEmail, role)
			if len(commentAttachments) == 0 {
				continue
			}
			views[i].Comments[j].Attachments = commentAttachments
			views[i].Comments[j].HasAttachments = true
			for _, attachment := range commentAttachments {
				if attachment.IsImage {
					views[i].PhotoCount++
					views[i].HasPhotos = true
				}
			}
		}
	}
	return views
}

func issueCommentByID(comments []issueComment, id string) (issueComment, bool) {
	id = strings.TrimSpace(id)
	for _, comment := range comments {
		if comment.ID == id {
			return comment, true
		}
	}
	return issueComment{}, false
}

func (a *app) legacyIssuePhotoViews(tenantSlug string, item residentIssue) []attachmentView {
	if a == nil || a.issueStore == nil || len(item.PhotoPaths) == 0 {
		return nil
	}
	views := make([]attachmentView, 0, len(item.PhotoPaths))
	for i := range item.PhotoPaths {
		_, filename, ok := a.legacyIssuePhotoPath(tenantSlug, item, i)
		if !ok {
			continue
		}
		url := "/app/anliegen/" + url.PathEscape(item.ID) + "/photos/" + strconv.Itoa(i)
		views = append(views, attachmentView{
			ID:         "legacy-photo-" + strconv.Itoa(i),
			Filename:   filename,
			URL:        url,
			PreviewURL: url,
			ThumbURL:   url,
			IsImage:    true,
		})
	}
	return views
}

func (a *app) legacyIssuePhotoPath(tenantSlug string, item residentIssue, index int) (string, string, bool) {
	if a == nil || a.issueStore == nil || a.issueStore.AttachmentDir() == "" || index < 0 || index >= len(item.PhotoPaths) {
		return "", "", false
	}
	tenantSlug = normalizeSlug(tenantSlug)
	if tenantSlug == "" || normalizeSlug(item.TenantSlug) != tenantSlug {
		return "", "", false
	}
	filename := filepath.Base(strings.TrimSpace(item.PhotoPaths[index]))
	if filename == "" || filename == "." || filename == string(filepath.Separator) {
		return "", "", false
	}
	return filepath.Join(a.issueStore.AttachmentDir(), tenantSlug, filename), filename, true
}

func (a *app) attachmentViewsForEntity(tenantSlug string, entityType string, entityID string, actorEmail string, role string) []attachmentView {
	if a == nil || a.attachmentStore == nil {
		return nil
	}
	records := a.attachmentStore.ListEntity(tenantSlug, entityType, entityID)
	views := make([]attachmentView, 0, len(records))
	for _, record := range records {
		views = append(views, attachmentViewFromRecord(record, a.canDeleteAttachment(tenantSlug, record, actorEmail, role)))
	}
	return views
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
		canServiceAct := isServiceProviderRole(role) && issueAssignedToActor(item, actorEmail) && issueIsOpen(item)
		hasEstimate := item.EstimateAmountCents > 0 || strings.TrimSpace(item.EstimateNote) != ""
		views = append(views, issueView{
			ID:                   item.ID,
			Title:                item.Title,
			Body:                 item.Body,
			Author:               author,
			AuthorEmail:          item.AuthorEmail,
			Category:             item.Category,
			Status:               status,
			StatusClass:          issueStatusClass(status),
			Priority:             priority,
			AssigneeEmail:        item.AssigneeEmail,
			HasAssignee:          item.AssigneeEmail != "",
			Location:             issueLocationLabel(item.LocationType, item.LocationDetail),
			CreatedAt:            formatLocalDateTime(item.CreatedAt),
			CanComment:           canManage || canResidentAct || canServiceAct,
			CanClose:             canResidentAct && canResidentTransition(status, issueStatusDone),
			CanReopen:            canResidentAct && canResidentTransition(status, issueStatusNew),
			CanServiceUpdate:     canServiceAct,
			ServiceProposal:      item.ServiceProposal,
			HasServiceProposal:   strings.TrimSpace(item.ServiceProposal) != "",
			CanEditEstimate:      canManage || canServiceAct,
			EstimateAmount:       formatIssueEstimateAmount(item.EstimateAmountCents),
			EstimateAmountValue:  formatIssueEstimateInput(item.EstimateAmountCents),
			EstimateNote:         item.EstimateNote,
			HasEstimate:          hasEstimate,
			PhotoCount:           photoCount,
			HasPhotos:            photoCount > 0,
			Comments:             comments,
			HasComments:          len(comments) > 0,
			StatusOptions:        issueSelectOptions(issueStatuses(), status),
			ServiceStatusOptions: issueSelectOptions(serviceProviderIssueStatuses(), status),
			PriorityOptions:      issueSelectOptions(issuePriorities(), priority),
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
			ID:        comment.ID,
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
	if denyServiceProviderArea(w, role) {
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
	paymentMsg, paymentOK := unitPaymentStatusMessage(r.URL.Query().Get("payment"))
	units := a.unitStore.ListTenant(tenant.Slug)
	billableWeight := billableUnitWeight(units)
	a.render(w, "buildingSettings", map[string]any{
		"Title":            "Gebäude",
		"Tenant":           tenant,
		"Email":            email,
		"DisplayName":      profile.DisplayName(),
		"Initials":         profile.Initials(),
		"Role":             role,
		"IsAdmin":          hasCapability(role, capabilityPlatformAdmin),
		"CanSeeParking":    hasCapability(role, capabilityPlatformAdmin) || profile.HasPermission(permissionParking),
		"ActivePage":       "settings",
		"BuildingMsg":      buildingMsg,
		"BuildingOK":       buildingOK,
		"BrandIconOptions": tenantBrandIconOptions(tenant.BrandIcon),
		"BrandIconLabel":   tenantBrandIconLabel(tenant.BrandIcon),
		"HeroMsg":          heroMsg,
		"HeroOK":           heroOK,
		"HasCustomHero":    a.hasTenantHero(tenant.Slug),
		"UnitMsg":          unitMsg,
		"UnitOK":           unitOK,
		"Units":            buildingUnitViews(units),
		"UnitTotal":        len(units),
		"BillableUnits":    formatBillableUnitWeight(billableWeight),
		"BillableLabel":    billableUnitCountLabel(billableWeight),
		"UnitsEmpty":       emptyState("Noch keine Einheiten", "Angelegte Einheiten erscheinen hier mit Anteil und Kontaktlinks."),
		"PaymentMsg":       paymentMsg,
		"PaymentOK":        paymentOK,
		"PaymentRows":      a.unitPaymentStatusViewsForUnits(tenant.Slug, units),
		"HasPaymentRows":   len(units) > 0,
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
	eventViews := auditEventViews(events)
	stats := auditStats(events, action, query)
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
		"Events":        eventViews,
		"HasEvents":     len(eventViews) > 0,
		"EventsEmpty":   emptyState("Noch keine Audit-Einträge", "Sensible Aktionen erscheinen hier, sobald sie im Portal ausgeführt werden."),
		"ActionOptions": auditActionOptions(action),
		"ActionFilter":  action,
		"SearchQuery":   query,
		"AuditStats":    stats,
	})
}

func auditEventViews(events []auditEvent) []auditEventView {
	views := make([]auditEventView, 0, len(events))
	lastDate := ""
	for _, event := range events {
		view := auditEventViewFrom(event)
		if view.AtDate != lastDate {
			view.ShowDateHeader = true
			view.DateHeader = auditDateHeader(event.At)
			lastDate = view.AtDate
		}
		views = append(views, view)
	}
	return views
}

func auditStats(events []auditEvent, action string, query string) auditStatsView {
	stats := auditStatsView{
		TotalEvents:   len(events),
		FilterSummary: "Alle Aktionen",
	}
	actors := map[string]struct{}{}
	now := time.Now().In(time.Local)
	for _, event := range events {
		actor := normalizeEmail(event.ActorEmail)
		if actor != "" {
			actors[actor] = struct{}{}
		}
		if sameLocalDate(event.At.In(time.Local), now) {
			stats.TodayCount++
		}
	}
	stats.ActorCount = len(actors)
	chips := []auditFilterChipView{}
	if action != "" {
		chips = append(chips, auditFilterChipView{Label: "Aktion", Value: auditActionLabel(action)})
	}
	query = strings.TrimSpace(query)
	if query != "" {
		chips = append(chips, auditFilterChipView{Label: "Suche", Value: query})
	}
	if len(chips) > 0 {
		parts := make([]string, 0, len(chips))
		for _, chip := range chips {
			parts = append(parts, chip.Label+": "+chip.Value)
		}
		stats.FilterSummary = strings.Join(parts, " · ")
	}
	stats.ActiveFilters = chips
	stats.HasActiveFilters = len(chips) > 0
	return stats
}

func auditDateHeader(t time.Time) string {
	local := t.In(time.Local)
	today := time.Now().In(time.Local)
	if sameLocalDate(local, today) {
		return "Heute"
	}
	if sameLocalDate(local, today.AddDate(0, 0, -1)) {
		return "Gestern"
	}
	return formatLocalDate(t)
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
			"changed_fields":     "Stammdaten, Marke, Kontaktblock, Notdienst, Hausmeister",
			"brand_icon":         tenantBrandIconLabel(override.BrandIcon),
			"brand_abbreviation": override.BrandAbbreviation,
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
	previous := ""
	if a.tenantOverrides != nil {
		if override, ok := a.tenantOverrides.Get(tenant.Slug); ok {
			previous = override.HeroImage
		}
	}
	filename, err := a.saveTenantHeroImage(tenant.Slug, header)
	if err != nil {
		log.Printf("tenant hero upload failed for %s: %v", tenant.Slug, err)
		http.Redirect(w, r, "/app/settings/building?hero=invalid", http.StatusSeeOther)
		return
	}
	if a.tenantOverrides != nil {
		if err := a.tenantOverrides.SetHeroImage(tenant.Slug, filename); err != nil {
			_ = a.removeTenantHeroImage(filename)
			log.Printf("tenant hero save failed for %s: %v", tenant.Slug, err)
			http.Redirect(w, r, "/app/settings/building?hero=error", http.StatusSeeOther)
			return
		}
	}
	if previous != "" && previous != filename {
		if err := a.removeTenantHeroImage(previous); err != nil {
			log.Printf("tenant old hero cleanup failed for %s: %v", tenant.Slug, err)
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

func (a *app) deleteBuildingHero(w http.ResponseWriter, r *http.Request) {
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
	previous := ""
	if a.tenantOverrides != nil {
		var err error
		previous, err = a.tenantOverrides.ClearHeroImage(tenant.Slug)
		if err != nil {
			log.Printf("tenant hero reset failed for %s: %v", tenant.Slug, err)
			http.Redirect(w, r, "/app/settings/building?hero=error", http.StatusSeeOther)
			return
		}
	}
	if previous != "" {
		if err := a.removeTenantHeroImage(previous); err != nil {
			log.Printf("tenant hero remove failed for %s: %v", tenant.Slug, err)
		}
		a.recordAudit(auditEvent{
			TenantSlug: tenant.Slug,
			ActorEmail: actorEmail,
			ActorRole:  role,
			Action:     auditActionHeroUpdate,
			TargetType: "hero",
			TargetID:   tenant.Slug,
			Summary:    "Hero-Bild entfernt",
		})
	}
	http.Redirect(w, r, "/app/settings/building?hero=removed", http.StatusSeeOther)
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
			"unit_label":         item.Label,
			"unit_type":          unitTypeLabel(item.UnitType),
			"billable_weight":    unitBillableLabel(item.BillableWeightPPM),
			"miteigentumsanteil": formatMiteigentumsanteil(item.MiteigentumsanteilPPM),
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
			"unit_label":         removedUnit.Label,
			"unit_type":          unitTypeLabel(removedUnit.UnitType),
			"billable_weight":    unitBillableLabel(removedUnit.BillableWeightPPM),
			"miteigentumsanteil": formatMiteigentumsanteil(removedUnit.MiteigentumsanteilPPM),
		},
	})
	http.Redirect(w, r, "/app/settings/building?unit=deleted", http.StatusSeeOther)
}

func (a *app) updateUnitPaymentStatus(w http.ResponseWriter, r *http.Request) {
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
	unitID := normalizeUnitID(r.FormValue("unit_id"))
	status := normalizeUnitPaymentStatus(r.FormValue("status"))
	if unitID == "" || status == "" {
		http.Redirect(w, r, "/app/settings/building?payment=invalid", http.StatusSeeOther)
		return
	}
	members := a.unitStore.MembersForUnit(tenant.Slug, unitID)
	if !members.Found {
		http.Redirect(w, r, "/app/settings/building?payment=missing", http.StatusSeeOther)
		return
	}
	record, err := a.unitPaymentStore.Set(unitPaymentStatus{
		TenantSlug: tenant.Slug,
		UnitID:     unitID,
		Status:     status,
		UpdatedBy:  actorEmail,
	})
	if err != nil {
		log.Printf("unit payment status save failed for %s/%s: %v", tenant.Slug, unitID, err)
		http.Redirect(w, r, "/app/settings/building?payment=error", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionUnitPayment,
		TargetType: "unit",
		TargetID:   unitID,
		Summary:    "Zahlungsstatus geändert",
		Details: map[string]string{
			"unit_label": members.Unit.Label,
			"status":     unitPaymentStatusLabel(record.Status),
		},
	})
	http.Redirect(w, r, "/app/settings/building?payment=saved", http.StatusSeeOther)
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
		MetaSet:           true,
		Name:              strings.TrimSpace(values.Get("name")),
		Address:           strings.TrimSpace(values.Get("address")),
		BrandIcon:         normalizeTenantBrandIcon(values.Get("brand_icon")),
		BrandAbbreviation: normalizeTenantBrandAbbreviation(values.Get("brand_abbreviation")),
		ContactName:       strings.TrimSpace(values.Get("contact_name")),
		ContactEmail:      normalizeEmail(values.Get("contact_email")),
		ContactPhone:      strings.TrimSpace(values.Get("contact_phone")),
		EmergencyName:     strings.TrimSpace(values.Get("emergency_name")),
		EmergencyPhone:    strings.TrimSpace(values.Get("emergency_phone")),
		CaretakerName:     strings.TrimSpace(values.Get("caretaker_name")),
		CaretakerEmail:    normalizeEmail(values.Get("caretaker_email")),
		CaretakerPhone:    strings.TrimSpace(values.Get("caretaker_phone")),
	}
	if override.Name == "" || override.Address == "" {
		return tenantOverride{}, fmt.Errorf("building name and address are required")
	}
	if override.BrandIcon == "" {
		return tenantOverride{}, fmt.Errorf("invalid brand icon")
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
	unitType := normalizeUnitType(values.Get("unit_type"))
	if unitType == "" {
		return unit{}, fmt.Errorf("invalid unit type")
	}
	return unit{
		ID:                    id,
		TenantSlug:            normalizeSlug(tenantSlug),
		Label:                 label,
		UnitType:              unitType,
		BillableWeightPPM:     defaultUnitBillableWeight(unitType),
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
			UnitType:           item.UnitType,
			UnitTypeLabel:      unitTypeLabel(item.UnitType),
			TypeOptions:        unitTypeOptions(item.UnitType),
			BillableLabel:      unitBillableLabel(item.BillableWeightPPM),
			Share:              formatMiteigentumsanteil(item.MiteigentumsanteilPPM),
			ShareValue:         strconv.Itoa(item.MiteigentumsanteilPPM),
			OwnerEmails:        strings.Join(item.OwnerEmails, ", "),
			RenterEmails:       strings.Join(item.RenterEmails, ", "),
			DeleteConfirmLabel: "Einheit " + item.Label + " entfernen",
		})
	}
	return views
}

func (a *app) unitPaymentStatusViewsForUnits(tenantSlug string, units []unit) []unitPaymentStatusView {
	statuses := map[string]unitPaymentStatus{}
	if a != nil && a.unitPaymentStore != nil {
		for _, item := range a.unitPaymentStore.ListTenant(tenantSlug) {
			statuses[item.UnitID] = item
		}
	}
	views := make([]unitPaymentStatusView, 0, len(units))
	for _, item := range units {
		record, hasRecord := statuses[normalizeUnitID(item.ID)]
		views = append(views, unitPaymentStatusViewFromUnit(item, "", record, hasRecord))
	}
	return views
}

func (a *app) unitPaymentStatusViewsForEmail(tenantSlug string, email string) []unitPaymentStatusView {
	if a == nil || a.unitStore == nil || a.unitPaymentStore == nil {
		return nil
	}
	memberships := a.unitStore.UnitsForEmail(tenantSlug, email)
	views := make([]unitPaymentStatusView, 0, len(memberships))
	for _, membership := range memberships {
		record, hasRecord := a.unitPaymentStore.Get(tenantSlug, membership.Unit.ID)
		if !hasRecord {
			continue
		}
		views = append(views, unitPaymentStatusViewFromUnit(membership.Unit, membership.Relation, record, true))
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
	case "removed":
		return "Hero-Bild entfernt. Das Standardbild ist wieder aktiv.", true
	case "invalid":
		return "Bitte ein JPG-, PNG- oder WebP-Bild bis 5 MB auswählen.", false
	case "error":
		return "Das Hero-Bild konnte nicht geändert werden.", false
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

func unitPaymentStatusMessage(status string) (string, bool) {
	switch status {
	case "saved":
		return "Zahlungsstatus gespeichert.", true
	case "invalid":
		return "Bitte Einheit und Zahlungsstatus prüfen.", false
	case "missing":
		return "Diese Einheit wurde nicht gefunden.", false
	case "error":
		return "Der Zahlungsstatus konnte nicht gespeichert werden.", false
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
	if denyServiceProviderArea(w, role) {
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
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if denyServiceProviderArea(w, role) {
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
	if denyServiceProviderArea(w, role) {
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
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if denyServiceProviderArea(w, role) {
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

func (a *app) parkingAccessSettings(w http.ResponseWriter, r *http.Request) {
	tenant, email, role, profile, ok := a.parkingAccessContext(w, r)
	if !ok {
		return
	}
	accessMsg, accessOK := parkingAccessMessage(r.URL.Query().Get("parking_access"))
	rows := a.parkingAccessRows(tenant.Slug)
	a.render(w, "parkingAccessSettings", map[string]any{
		"Title":           "Parkplatz-Zugriff",
		"Tenant":          tenant,
		"Email":           email,
		"DisplayName":     profile.DisplayName(),
		"Initials":        profile.Initials(),
		"Role":            role,
		"IsAdmin":         hasCapability(role, capabilityPlatformAdmin),
		"CanSeeParking":   hasCapability(role, capabilityPlatformAdmin) || profile.HasPermission(permissionParking),
		"ActivePage":      "settings",
		"AccessRows":      rows,
		"HasAccessRows":   len(rows) > 0,
		"AccessRowsEmpty": emptyState("Noch keine Zugänge", "Sobald Personen eingeladen sind, kann der Parkplatz-Zugriff hier gepflegt werden."),
		"AccessMsg":       accessMsg,
		"AccessOK":        accessOK,
		"StatementYear":   time.Now().In(time.Local).Year(),
	})
}

func (a *app) updateParkingAccess(w http.ResponseWriter, r *http.Request) {
	tenant, actorEmail, role, _, ok := a.parkingAccessContext(w, r)
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
	targetEmail := normalizeEmail(r.FormValue("email"))
	enabled := r.FormValue("parking") == "1"
	if a.inviteStore == nil {
		http.Redirect(w, r, "/app/settings/parking-access?parking_access=error", http.StatusSeeOther)
		return
	}
	existing, isInvite := a.inviteStore.Get(targetEmail)
	if !isInvite {
		http.Redirect(w, r, "/app/settings/parking-access?parking_access=not_editable", http.StatusSeeOther)
		return
	}
	if !existing.HasTenant(tenant.Slug) {
		http.Redirect(w, r, "/app/settings/parking-access?parking_access=invalid", http.StatusSeeOther)
		return
	}
	if normalizeRole(existing.Role) == roleAdmin && !hasCapability(role, capabilityPlatformAdmin) {
		http.Redirect(w, r, "/app/settings/parking-access?parking_access=not_editable", http.StatusSeeOther)
		return
	}
	updated := existing
	updated.Permissions = setPermission(updated.Permissions, permissionParking, enabled)
	if len(updated.Tenants) == 0 {
		updated.Tenants = []string{tenant.Slug}
	}
	if len(updated.AuthMethods) == 0 {
		updated.AuthMethods = defaultAuthMethods()
	}
	changed, err := a.inviteStore.Update(targetEmail, updated)
	if err != nil {
		log.Printf("parking access update failed for %s: %v", redactedEmail(targetEmail), err)
		http.Redirect(w, r, "/app/settings/parking-access?parking_access=error", http.StatusSeeOther)
		return
	}
	if !changed {
		http.Redirect(w, r, "/app/settings/parking-access?parking_access=not_editable", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionInviteUpdate,
		TargetType: "user",
		TargetID:   updated.Email,
		Summary:    "Parkplatz-Zugriff geändert",
		Details: map[string]string{
			"changed_fields":   "Parkplatz-Zugriff",
			"permissions_from": strings.Join(permissionLabelList(existing.Permissions), ", "),
			"permissions_to":   strings.Join(permissionLabelList(updated.Permissions), ", "),
		},
	})
	status := "revoked"
	if enabled {
		status = "granted"
	}
	http.Redirect(w, r, "/app/settings/parking-access?parking_access="+status, http.StatusSeeOther)
}

func (a *app) parkingAccessContext(w http.ResponseWriter, r *http.Request) (tenantConfig, string, string, userProfile, bool) {
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
	if !hasCapability(role, capabilityManageUsers) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return tenantConfig{}, "", "", userProfile{}, false
	}
	return tenant, email, role, a.profileForTenant(email, tenant.Slug), true
}

func parkingAccessMessage(status string) (string, bool) {
	switch status {
	case "granted":
		return "Parkplatz-Zugriff freigegeben.", true
	case "revoked":
		return "Parkplatz-Zugriff entzogen.", true
	case "reminder_sent":
		return "Zahlungserinnerungen gesendet.", true
	case "reminder_none":
		return "Keine überfälligen offenen Parkplatzbeträge mit aktiver Benachrichtigung gefunden.", false
	case "not_editable":
		return "Dieser Eintrag kommt aus der Konfiguration und kann hier nicht geändert werden.", false
	case "invalid":
		return "Bitte den Zugang prüfen.", false
	case "error":
		return "Der Parkplatz-Zugriff konnte nicht gespeichert werden.", false
	default:
		return "", false
	}
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

// render prepares page data and writes the template. The data preparation
// reaches into stores (unread announcements, open issues), so this stays in the
// HTTP layer — see executeTemplate for the part that does not.
func (a *app) render(w http.ResponseWriter, name string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	if _, ok := data["AppVersion"]; !ok {
		data["AppVersion"] = version.BuildLabel()
	}
	if _, ok := data["DisplayVersion"]; !ok {
		data["DisplayVersion"] = version.DisplayVersion(version.Version)
	}
	if _, ok := data["AssetVersion"]; !ok {
		data["AssetVersion"] = version.AssetVersion()
	}
	if _, ok := data["ReleaseNotes"]; !ok {
		notes := version.Notes()
		data["ReleaseNotes"] = notes
		data["HasReleaseNotes"] = len(notes) > 0
	}
	enrichCapabilityData(data)
	// These two read from the announcement and issue stores. They are the reason
	// render() cannot itself live in a pure rendering package.
	a.enrichUnreadAnnouncementData(data)
	a.enrichIssueData(data)
	a.executeTemplate(w, name, data)
}

// executeTemplate is the pure rendering step: no store access, no business
// logic — just template + data → HTML. This is the piece that becomes
// internal/web.Renderer.
func (a *app) executeTemplate(w http.ResponseWriter, name string, data map[string]any) {
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
	if _, ok := data["IsServiceProvider"]; !ok {
		data["IsServiceProvider"] = isServiceProviderRole(role)
	}
	if _, ok := data["CanUseResidentAreas"]; !ok {
		data["CanUseResidentAreas"] = canUseResidentAreas(role)
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
	if _, ok := data["CanManageHandovers"]; !ok {
		data["CanManageHandovers"] = canManageHandovers(role)
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

func (a *app) isMarketingHost(r *http.Request) bool {
	if a == nil || r == nil {
		return false
	}
	host := normalizeHost(r.Host)
	root := normalizeHost(a.rootDomain)
	if host == "" || root == "" {
		return false
	}
	return host == root || host == "www."+root
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
	tenant.BrandIcon = normalizeTenantBrandIcon(tenant.BrandIcon)
	if tenant.BrandIcon == "" {
		tenant.BrandIcon = tenantBrandCommunity
	}
	tenant.BrandAbbreviation = normalizeTenantBrandAbbreviation(tenant.BrandAbbreviation)
	if a.tenantOverrides == nil {
		if tenant.BrandAbbreviation == "" {
			tenant.BrandAbbreviation = generatedTenantBrandAbbreviation(tenant)
		}
		return tenant
	}
	override, ok := a.tenantOverrides.Get(tenant.Slug)
	if !ok {
		if tenant.BrandAbbreviation == "" {
			tenant.BrandAbbreviation = generatedTenantBrandAbbreviation(tenant)
		}
		return tenant
	}
	if override.MetaSet {
		if override.Name != "" {
			tenant.Name = override.Name
		}
		if override.Address != "" {
			tenant.Address = override.Address
		}
		if override.BrandIcon != "" {
			tenant.BrandIcon = override.BrandIcon
		}
		tenant.BrandAbbreviation = firstNonEmpty(override.BrandAbbreviation, tenant.BrandAbbreviation)
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
	tenant.BrandIcon = normalizeTenantBrandIcon(tenant.BrandIcon)
	if tenant.BrandIcon == "" {
		tenant.BrandIcon = tenantBrandCommunity
	}
	tenant.BrandAbbreviation = normalizeTenantBrandAbbreviation(tenant.BrandAbbreviation)
	if tenant.BrandAbbreviation == "" {
		tenant.BrandAbbreviation = generatedTenantBrandAbbreviation(tenant)
	}
	return tenant
}

func (a *app) hasTenantHero(tenantSlug string) bool {
	if a == nil || a.tenantOverrides == nil {
		return false
	}
	override, ok := a.tenantOverrides.Get(tenantSlug)
	return ok && override.HeroImage != ""
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
		rows = append(rows, userRowFrom(profile.ForTenant(tenantSlug)))
		seen[email] = struct{}{}
	}
	for email := range a.admins {
		if _, ok := seen[email]; ok {
			continue
		}
		rows = append(rows, userRowFrom(userProfile{Email: email, Role: roleAdmin, Status: "Aktiv", Tenants: []string{tenantSlug}}))
		seen[email] = struct{}{}
	}
	for email := range a.allowed {
		if _, ok := seen[email]; ok {
			continue
		}
		rows = append(rows, userRowFrom(userProfile{Email: email, Role: roleResident, Status: "Eingeladen", Tenants: []string{tenantSlug}}))
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
			row := userRowFrom(profile.ForTenant(tenantSlug))
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

func (a *app) parkingAccessRows(tenantSlug string) []userRow {
	rows := a.userRows(tenantSlug)
	balance := a.parkingBalance(tenantSlug)
	for i := range rows {
		if !rows[i].ParkingChecked || balance.Outstanding <= 0 {
			continue
		}
		rows[i].OutstandingBalance = formatEUR(balance.Outstanding)
		rows[i].HasOutstanding = true
	}
	return rows
}

func (a *app) parkingBalance(tenantSlug string) parkingBalanceView {
	if a == nil || a.parkingStore == nil {
		return parkingBalanceView{}
	}
	data := a.parkingStore.TenantData(tenantSlug)
	return parkingBalanceSummary(calculateParkingMonths(data, time.Now(), time.Local))
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

func (s *tenantOverrideStore) ClearHeroImage(tenantSlug string) (string, error) {
	if s == nil {
		return "", nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	if tenantSlug == "" {
		return "", fmt.Errorf("invalid tenant")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Tenants == nil {
		s.data.Tenants = map[string]tenantOverride{}
	}
	override := normalizeTenantOverride(s.data.Tenants[tenantSlug])
	previous := override.HeroImage
	if previous == "" {
		return "", nil
	}
	override.HeroImage = ""
	override.UpdatedAt = time.Now().UTC()
	s.data.Tenants[tenantSlug] = override
	return previous, s.saveLocked()
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
	return saveJSONAtomic(s.path, s.data, "tenant")
}

func normalizeTenantOverride(override tenantOverride) tenantOverride {
	override.Name = strings.TrimSpace(override.Name)
	override.Address = strings.TrimSpace(override.Address)
	override.BrandIcon = normalizeTenantBrandIcon(override.BrandIcon)
	override.BrandAbbreviation = normalizeTenantBrandAbbreviation(override.BrandAbbreviation)
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

func normalizeTenantBrandAbbreviation(raw string) string {
	raw = strings.ToUpper(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '.':
			b.WriteRune(r)
		case r == ' ' || r == '_' || r == '/':
			b.WriteRune('-')
		}
		if b.Len() >= 12 {
			break
		}
	}
	return strings.Trim(b.String(), "-.")
}

func generatedTenantBrandAbbreviation(tenant tenantConfig) string {
	if value := normalizeTenantBrandAbbreviation(tenant.Slug); value != "" {
		return value
	}
	if value := normalizeTenantBrandAbbreviation(tenant.Host); value != "" {
		return value
	}
	return "HAUS"
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
	suffix, err := randomToken(8)
	if err != nil {
		_ = os.Remove(tmpName)
		return "", fmt.Errorf("could not name tenant hero image")
	}
	filename := tenantSlug + "-hero-" + suffix + ext
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

func (a *app) removeTenantHeroImage(filename string) error {
	filename = filepath.Base(strings.TrimSpace(filename))
	if a == nil || a.tenantHeroDir == "" || filename == "" || filename == "." || filename == string(filepath.Separator) {
		return nil
	}
	err := os.Remove(filepath.Join(a.tenantHeroDir, filename))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// uploadedFileFromHeader adapts an HTTP multipart upload at the handler
// boundary — the only place the two representations meet.
func uploadedFileFromHeader(h *multipart.FileHeader) uploadedFile {
	return uploadedFile{
		Filename:     h.Filename,
		Size:         h.Size,
		DeclaredType: h.Header.Get("Content-Type"),
		Open:         func() (io.ReadSeekCloser, error) { return h.Open() },
	}
}

func uploadedFilesFromHeaders(hs []*multipart.FileHeader) []uploadedFile {
	out := make([]uploadedFile, 0, len(hs))
	for _, h := range hs {
		out = append(out, uploadedFileFromHeader(h))
	}
	return out
}

func managedContactFromForm(tenantSlug string, values url.Values) (managedContact, error) {
	return normalizeManagedContact(managedContact{
		ID:         strings.TrimSpace(values.Get("id")),
		TenantSlug: tenantSlug,
		Kind:       values.Get("kind"),
		Name:       values.Get("name"),
		Company:    values.Get("company"),
		Email:      values.Get("email"),
		Phone:      values.Get("phone"),
		Notes:      values.Get("notes"),
		Active:     values.Get("active") != "false",
	})
}

func storeEBInterfaceInvoiceDocument(store *documentStore, invoice integrations.Invoice, uploadedBy string, data []byte, now time.Time) (documentRecord, error) {
	if store == nil {
		return documentRecord{}, fmt.Errorf("document store unavailable")
	}
	title := "E-Rechnung"
	if strings.TrimSpace(invoice.InvoiceNumber) != "" {
		title += " " + strings.TrimSpace(invoice.InvoiceNumber)
	}
	if strings.TrimSpace(invoice.IssuerName) != "" {
		title += " - " + strings.TrimSpace(invoice.IssuerName)
	}
	filenameToken := integrations.SanitizeFilenameToken(firstNonEmpty(invoice.InvoiceNumber, invoice.ExternalID, "rechnung"))
	return store.CreateGenerated(documentRecord{
		TenantSlug: invoice.TenantSlug,
		Title:      title,
		Category:   documentCategoryBilling,
		Visibility: documentVisibilityManagerOnly,
		UploadedBy: uploadedBy,
	}, "ebinterface-"+filenameToken+".xml", "application/xml", data, now)
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

func (a *app) parkingTelemetry(ctx context.Context, tenant tenantConfig) parkingTelemetry {
	ha := tenant.HA
	telemetry := parkingTelemetry{
		Configured: ha.BaseURL() != "",
		Entities: []parkingEntityRef{
			{Label: "Zählerstand", EntityID: ha.MeterEnergyEntity()},
			{Label: "Leistung", EntityID: ha.PowerEntity()},
			{Label: "aWATTar Preis", EntityID: ha.PriceEntity()},
		},
	}
	if ha.BaseURL() == "" {
		telemetry.Message = "Home Assistant ist lokal noch nicht konfiguriert."
		return telemetry
	}
	if ha.Token() == "" {
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
		{label: "Zählerstand", entity: ha.MeterEnergyEntity()},
		{label: "Aktuelle Leistung", entity: ha.PowerEntity()},
		{label: "aWATTar Gesamtpreis", entity: ha.PriceEntity()},
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
		case ha.MeterEnergyEntity():
			if value, err := homeassistant.ParseFloat(state.State); err == nil {
				liveEnergy = value
				hasLiveEnergy = true
			}
		case ha.PriceEntity():
			if value, err := homeassistant.ParseFloat(state.State); err == nil {
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
	if err := a.seedParkingHistory(seedCtx, tenant, a.parkingHistoryStart, time.Now()); err != nil && tenant.HA.Configured() {
		log.Printf("parking history seed failed for %s: %v", tenant.Slug, err)
	}

	data := a.parkingStore.TenantData(tenant.Slug)
	months := calculateParkingMonths(data, time.Now(), time.Local)
	balance := parkingBalanceSummary(months)
	currentTariff := parkingTariffAt(data.Settings, time.Now(), time.Local)
	tariffs := parkingTariffViews(data.Settings)
	view := parkingAccountingView{
		GridFeeValue:     formatInputFloat(currentTariff.GridFeeEURPerKWh),
		GridFeeLabel:     formatEURPerKWh(currentTariff.GridFeeEURPerKWh),
		BaseFeeValue:     formatInputFloat(currentTariff.BaseFeeEUR),
		BaseFeeLabel:     formatEUR(currentTariff.BaseFeeEUR),
		EffectiveFrom:    currentTariff.EffectiveFrom,
		Tariffs:          tariffs,
		HasTariffs:       len(tariffs) > 0,
		Months:           months,
		HasMonths:        len(months) > 0,
		OutstandingValue: balance.Outstanding,
		Outstanding:      formatEUR(balance.Outstanding),
		HasOutstanding:   balance.Outstanding > 0,
		OverdueValue:     balance.Overdue,
		Overdue:          formatEUR(balance.Overdue),
		HasOverdue:       balance.Overdue > 0,
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
	if err := a.seedParkingHistory(seedCtx, tenant, a.parkingHistoryStart, time.Now()); err != nil && tenant.HA.Configured() {
		log.Printf("parking history seed failed for %s: %v", tenant.Slug, err)
	}

	data := a.parkingStore.TenantData(tenant.Slug)
	view := calculateParkingMonthDetails(data, month, time.Now(), time.Local)
	view.BackPath = "/app/parking"
	view.GridFeeLabel = formatEURPerKWh(parkingTariffForMonth(data.Settings, month, time.Local).GridFeeEURPerKWh)
	view.Message = "Stundenwerte aus Zählerdifferenz, aWATTar-Preis und stündlicher Netzgebühr; die Monatsbasis steht in der Zusammenfassung."
	if len(data.EnergySamples) > 0 {
		last := data.EnergySamples[len(data.EnergySamples)-1].At.In(time.Local)
		view.LastSampleLabel = formatLocalDateTime(last)
	}
	return view
}

func (a *app) seedParkingHistory(ctx context.Context, tenant tenantConfig, start time.Time, end time.Time) error {
	ha := tenant.HA
	if ha.BaseURL() == "" || ha.Token() == "" || ha.MeterEnergyEntity() == "" || ha.PriceEntity() == "" {
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
	history, err := ha.History(ctx, restStart, end, []string{ha.MeterEnergyEntity(), ha.PriceEntity()})
	if err != nil {
		if len(energySamples) == 0 && len(priceSamples) == 0 {
			return err
		}
		log.Printf("parking REST history fallback failed for %s: %v", tenant.Slug, err)
		return a.parkingStore.AppendReadings(tenant.Slug, energySamples, priceSamples)
	}
	energySamples = append(energySamples, homeassistant.SamplesFromHistory(history[ha.MeterEnergyEntity()])...)
	priceSamples = append(priceSamples, homeassistant.SamplesFromHistory(history[ha.PriceEntity()])...)
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
		if tenant.HA.Configured() {
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
		if tenant.HA.BaseURL() == "" || tenant.HA.Token() == "" {
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
		if tenant.HA.BaseURL() == "" || tenant.HA.Token() == "" {
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
	energy, err := tenant.HA.State(ctx, tenant.HA.MeterEnergyEntity())
	if err != nil {
		return err
	}
	price, err := tenant.HA.State(ctx, tenant.HA.PriceEntity())
	if err != nil {
		return err
	}
	energyValue, err := homeassistant.ParseFloat(energy.State)
	if err != nil {
		return err
	}
	priceValue, err := homeassistant.ParseFloat(price.State)
	if err != nil {
		return err
	}
	return a.parkingStore.AppendSamples(tenant.Slug, []parkingStoredSample{{
		At:             time.Now().UTC(),
		EnergyKWh:      energyValue,
		PriceEURPerKWh: priceValue,
	}})
}

func parkingTariffAt(settings parkingSettings, at time.Time, loc *time.Location) parkingTariff {
	settings = normalizeParkingSettings(settings)
	if loc == nil {
		loc = time.Local
	}
	if at.IsZero() {
		at = time.Now()
	}
	day := at.In(loc).Format("2006-01-02")
	selected := settings.Tariffs[0]
	for _, tariff := range settings.Tariffs {
		if tariff.EffectiveFrom <= day {
			selected = tariff
			continue
		}
		break
	}
	return selected
}

func parkingTariffForMonth(settings parkingSettings, month string, loc *time.Location) parkingTariff {
	if loc == nil {
		loc = time.Local
	}
	first, err := time.ParseInLocation("2006-01", month, loc)
	if err != nil {
		return parkingTariffAt(settings, time.Now(), loc)
	}
	return parkingTariffAt(settings, first, loc)
}

func parkingTariffViews(settings parkingSettings) []parkingTariffView {
	settings = normalizeParkingSettings(settings)
	views := make([]parkingTariffView, 0, len(settings.Tariffs))
	for i := len(settings.Tariffs) - 1; i >= 0; i-- {
		tariff := settings.Tariffs[i]
		views = append(views, parkingTariffView{
			EffectiveFrom:      formatParkingTariffDate(tariff.EffectiveFrom),
			EffectiveFromInput: tariff.EffectiveFrom,
			GridFee:            formatEURPerKWh(tariff.GridFeeEURPerKWh),
			BaseFee:            formatEUR(tariff.BaseFeeEUR),
		})
	}
	return views
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

func parkingBalanceSummary(months []parkingMonthView) parkingBalanceView {
	var summary parkingBalanceView
	for _, month := range months {
		if month.Outstanding {
			summary.Outstanding += month.TotalCostValue
		}
		if month.Overdue {
			summary.Overdue += month.TotalCostValue
		}
	}
	return summary
}

func calculateParkingHourlyUsage(energySamples []parkingNumericSample, priceSamples []parkingNumericSample, gridFeeEURPerKWh float64, now time.Time) []parkingHourUsage {
	return calculateParkingHourlyUsageWithSettings(energySamples, priceSamples, parkingSettings{GridFeeEURPerKWh: gridFeeEURPerKWh}, now, time.Local)
}

func calculateParkingHourlyUsageWithSettings(energySamples []parkingNumericSample, priceSamples []parkingNumericSample, settings parkingSettings, now time.Time, loc *time.Location) []parkingHourUsage {
	if loc == nil {
		loc = time.Local
	}
	settings = normalizeParkingSettings(settings)
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
			tariff := parkingTariffAt(settings, cursor, loc)
			hour := cursor.Truncate(time.Hour)
			out = append(out, parkingHourUsage{
				At:         hour,
				KWh:        kWh,
				PriceEUR:   price,
				EnergyCost: price * kWh,
				GridCost:   tariff.GridFeeEURPerKWh * kWh,
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
	if now.IsZero() {
		now = time.Now()
	}
	data.Settings = normalizeParkingSettings(data.Settings)
	data.Months = normalizeParkingMonthStates(data.Months)
	hours := calculateParkingHourlyUsageWithSettings(data.EnergySamples, data.PriceSamples, data.Settings, now, loc)
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
		tariff := parkingTariffForMonth(data.Settings, month, loc)
		total := agg.energyCost + agg.gridCost + tariff.BaseFeeEUR
		if total > maxTotal {
			maxTotal = total
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(months)))
	out := make([]parkingMonthView, 0, len(months))
	currentMonth := now.In(loc).Format("2006-01")
	for _, month := range months {
		agg := aggregates[month]
		tariff := parkingTariffForMonth(data.Settings, month, loc)
		gridCost := agg.gridCost
		total := agg.energyCost + gridCost + tariff.BaseFeeEUR
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
		state := normalizeParkingMonthState(data.Months[month])
		paid := state.Paid
		firstOfMonth, _ := time.ParseInLocation("2006-01", month, loc)
		paidAtInput := ""
		paidAtLabel := ""
		if !state.PaidAt.IsZero() {
			paidAtInput = formatDateTimeIn(state.PaidAt, loc, "2006-01-02")
			paidAtLabel = formatDateTimeIn(state.PaidAt, loc, deATDateLayout)
		}
		outstanding := !paid && total > 0
		overdue := outstanding && month < currentMonth
		out = append(out, parkingMonthView{
			Month:            month,
			MonthLabel:       formatMonthLabel(month, loc),
			DetailPath:       "/app/parking/month/" + month,
			PeriodLabel:      formatPeriodLabel(agg.first, agg.last, loc),
			KWhValue:         agg.kWh,
			EnergyCostValue:  agg.energyCost,
			GridCostValue:    gridCost,
			BaseFeeValue:     tariff.BaseFeeEUR,
			TotalCostValue:   total,
			KWh:              formatKWh(agg.kWh),
			EnergyCost:       formatEUR(agg.energyCost),
			GridCost:         formatEUR(gridCost),
			BaseFee:          formatEUR(tariff.BaseFeeEUR),
			TotalCost:        formatEUR(total),
			AverageAwattar:   formatEURPerKWh(averageAwattar),
			EffectivePrice:   formatEURPerKWh(effectivePrice),
			AveragePrice:     formatEURPerKWh(effectivePrice),
			Paid:             paid,
			PaidLabel:        paidLabel(paid),
			PaidAtInput:      paidAtInput,
			PaidAtLabel:      paidAtLabel,
			PaidBy:           state.PaidBy,
			PaymentMethod:    state.PaymentMethod,
			PaymentReference: state.PaymentReference,
			PaymentDetails:   parkingPaymentDetails(state, loc),
			Outstanding:      outstanding,
			Overdue:          overdue,
			TogglePaidValue:  boolFormValue(!paid),
			ToggleLabel:      togglePaidLabel(paid),
			ChartPercent:     chartPercent,
			Partial:          agg.hourCount == 0 || agg.first.In(loc).After(firstOfMonth.Add(24*time.Hour)),
			SampleCount:      len(data.EnergySamples),
			HourCount:        agg.hourCount,
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
	data.Settings = normalizeParkingSettings(data.Settings)
	hours := calculateParkingHourlyUsageWithSettings(data.EnergySamples, data.PriceSamples, data.Settings, now, loc)
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
		tariff := parkingTariffAt(data.Settings, hour.At, loc)
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
			GridCostTitle:       "Netzgebühr: " + formatPreciseEUR(hour.GridCost) + " = " + formatPreciseKWh(hour.KWh) + " × " + formatPreciseEURPerKWh(tariff.GridFeeEURPerKWh),
			TotalCost:           formatEUR(total),
			TotalCostTitle:      "Summe: " + formatPreciseEUR(total) + " = Strom " + formatPreciseEUR(hour.EnergyCost) + " + Netzgebühr " + formatPreciseEUR(hour.GridCost),
			WeightTitle:         "Relative Höhe der Stundensumme. 100% entspricht der teuersten Stunde dieses Monats.",
			ChartPercent:        chartPercent,
		})
	}
	view.HasHours = len(view.Hours) > 0
	return view
}

func userRowFrom(p userProfile) userRow {
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

func formatHAValue(state haState) string {
	unit, _ := state.Attributes["unit_of_measurement"].(string)
	value := strings.TrimSpace(state.State)
	if value == "" {
		value = "unbekannt"
	}
	if number, err := homeassistant.ParseFloat(value); err == nil {
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

func parseDecimal(raw string) (float64, error) {
	value := strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	if value == "" {
		return 0, errors.New("empty decimal")
	}
	return strconv.ParseFloat(value, 64)
}

const (
	deATDateLayout          = "02.01.2006"
	deATDateTimeLayout      = "02.01.2006 15:04"
	deATShortDateTimeLayout = "02.01. 15:04"
	deATTimeLayout          = "15:04"
	htmlDateTimeLocalLayout = "2006-01-02T15:04"
)

func parkingPaymentDetails(state parkingMonthState, loc *time.Location) string {
	state = normalizeParkingMonthState(state)
	if !state.Paid {
		return ""
	}
	if loc == nil {
		loc = time.Local
	}
	parts := []string{}
	if !state.PaidAt.IsZero() {
		parts = append(parts, "bezahlt am "+formatDateTimeIn(state.PaidAt, loc, deATDateLayout))
	}
	if state.PaymentMethod != "" {
		parts = append(parts, state.PaymentMethod)
	}
	if state.PaymentReference != "" {
		parts = append(parts, "Ref. "+state.PaymentReference)
	}
	if state.PaidBy != "" {
		parts = append(parts, "erfasst von "+state.PaidBy)
	}
	return strings.Join(parts, " · ")
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

func truncateRunes(value string, limit int) string { return textutil.Truncate(value, limit) }

func normalizeSlug(raw string) string { return textutil.Slug(raw) }

func firstNonEmpty(values ...string) string { return textutil.FirstNonEmpty(values...) }

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

func setPermission(raw []string, permission string, enabled bool) []string {
	permission = strings.ToLower(strings.TrimSpace(permission))
	out := []string{}
	for _, item := range normalizePermissions(raw) {
		if item == permission {
			continue
		}
		out = append(out, item)
	}
	if enabled && permission != "" {
		out = append(out, permission)
	}
	return normalizePermissions(out)
}

func normalizeEmail(v string) string { return textutil.Email(v) }

package server

import (
	"context"
	"crypto/hmac"
	"database/sql"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	_ "image/png"
	"io"
	"log"
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

	"github.com/markus-barta/hausv-org/internal/auth"
	"github.com/markus-barta/hausv-org/internal/authz"
	"github.com/markus-barta/hausv-org/internal/config"
	"github.com/markus-barta/hausv-org/internal/db"
	"github.com/markus-barta/hausv-org/internal/homeassistant"
	appmail "github.com/markus-barta/hausv-org/internal/mail"
	"github.com/markus-barta/hausv-org/internal/view"
	"github.com/markus-barta/hausv-org/internal/web"

	"github.com/markus-barta/hausv-org/internal/integrations"
	"github.com/markus-barta/hausv-org/internal/store"
	"github.com/markus-barta/hausv-org/internal/telegram"
	"github.com/markus-barta/hausv-org/internal/textutil"
	"github.com/markus-barta/hausv-org/internal/version"
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
		env("HA_TOKEN", ""),
		env("PARKING_METER_ENERGY_ENTITY", "sensor.kws_306wf_energy_meter_energy"),
		env("PARKING_POWER_ENTITY", "sensor.kws360_power"),
		env("PARKING_PRICE_ENTITY", "sensor.epex_spot_data_total_price"),
	).WithChargingEntities(
		env("CHARGING_PLUG_SWITCH_ENTITY", "switch.kws_306wf_energy_meter"),
		env("CHARGING_BATTERY_SOC_ENTITY", "sensor.sonnenbatterie_260365_state_charge_user"),
		env("CHARGING_GRID_FEEDIN_ENTITY", "sensor.sonnenbatterie_260365_state_grid_output"),
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
	chargingSession       = store.ChargingSession
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
var normalizeChargingSessions = store.NormalizeChargingSessions
var surplusRate = store.SurplusRate
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
	auditActionIssueComment            = store.AuditActionIssueComment
	auditActionIssueCommentDelete      = store.AuditActionIssueCommentDelete
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
	fairUseFreeUnits                   = store.FairUseFreeUnits
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
	activityStorage           = store.ActivityStorage
	profileOverlayStorage     = store.ProfileOverlayStorage
	notificationPrefStorage   = store.NotificationPrefStorage
	unitPaymentStatusStorage  = store.UnitPaymentStatusStorage
	contactBookStorage        = store.ContactBookStorage
	announcementReadStorage   = store.AnnouncementReadStorage
	announcementStorage       = store.AnnouncementStorage
	eventStorage              = store.EventStorage
	handoverStorage           = store.HandoverStorage
	documentStorage           = store.DocumentStorage
	protocolFiler             = store.ProtocolFiler
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
var newSQLActivityStore = store.NewSQLActivityStore
var newSQLProfileOverlayStore = store.NewSQLProfileOverlayStore
var newSQLNotificationPrefStore = store.NewSQLNotificationPrefStore
var newSQLUnitPaymentStatusStore = store.NewSQLUnitPaymentStatusStore
var newSQLContactBookStore = store.NewSQLContactBookStore
var newSQLAnnouncementReadStore = store.NewSQLAnnouncementReadStore
var newSQLAnnouncementStore = store.NewSQLAnnouncementStore
var newSQLEventStore = store.NewSQLEventStore
var newSQLHandoverStore = store.NewSQLHandoverStore
var newSQLDocumentStore = store.NewSQLDocumentStore
var newSQLProtocolFiler = store.NewSQLProtocolFiler
var newSequentialProtocolFiler = store.NewSequentialProtocolFiler
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
	issueStatusAccepted           = store.IssueStatusAccepted
	issueStatusScheduled          = store.IssueStatusScheduled
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

const serviceProviderAccessClosedMessage = "Datenschutzprüfung offen: Dienstleister-Zugänge sind noch nicht freigeschaltet."

var errServiceProviderAccessClosed = errors.New("service provider access is disabled")

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
	serviceAccessEnabled  bool
	sessionTTL            time.Duration
	tokens                *tokenStore
	sessions              *sessionStore
	oidc                  *oidcLogin
	oidcFlows             *oidcFlowStore
	mailer                mailer
	templates             *template.Template
	db                    *sql.DB
	announcementStore     announcementStorage
	announcementReadStore announcementReadStorage
	eventStore            eventStorage
	notificationPrefs     notificationPrefStorage
	profileOverlays       profileOverlayStorage
	tenantOverrides       *tenantOverrideStore
	tenantHeroDir         string
	inviteStore           *inviteStore
	activityStore         activityStorage
	unitStore             *unitStore
	unitPaymentStore      unitPaymentStatusStorage
	issueStore            *issueStore
	attachmentStore       *attachmentStore
	contactStore          contactBookStorage
	auditStore            *auditStore
	documentStore         documentStorage
	handoverStore         handoverStorage
	protocolFiler         protocolFiler
	voteStore             *voteStore
	voteReminderInterval  time.Duration
	parkingStore          *parkingStore
	parkingSampleInterval time.Duration
	parkingHistoryStart   time.Time

	chargingTickInterval   time.Duration
	chargingStaleAfter     time.Duration
	chargingConfirmTimeout time.Duration
	chargingHAFailLimit    int
	chargingMu             sync.Mutex
	chargingHAFails        map[string]int
	chargingShadow         map[string]chargingControllerState
	chargingShadowPlug     map[string]bool
	chargingEvents         *chargingEventRing
	chargingLastPoll       map[string]time.Time

	telegram                  telegramAPI
	telegramStore             *telegramStore
	telegramPollTimeout       time.Duration
	chargingTelegramMu        sync.Mutex
	chargingTelegramTimes     []time.Time
	chargingTelegramThrottled bool
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
	mux.HandleFunc("GET /app", a.page(a.portal))
	mux.HandleFunc("GET /app/announcements", a.page(a.announcements))
	mux.HandleFunc("POST /app/announcements", a.action(a.createAnnouncement))
	mux.HandleFunc("POST /app/announcements/edit", a.action(a.editAnnouncement))
	mux.HandleFunc("POST /app/announcements/delete", a.action(a.deleteAnnouncement))
	mux.HandleFunc("GET /app/events", a.page(a.events))
	mux.HandleFunc("POST /app/events", a.action(a.createEvent))
	mux.HandleFunc("POST /app/events/edit", a.action(a.editEvent))
	mux.HandleFunc("POST /app/events/delete", a.action(a.deleteEvent))
	mux.HandleFunc("GET /app/dokumente", a.page(a.documents))
	mux.HandleFunc("POST /app/dokumente", a.authedAction(capabilityManageDocuments, a.uploadDocument))
	mux.HandleFunc("POST /app/dokumente/replace", a.authedAction(capabilityManageDocuments, a.replaceDocument))
	mux.HandleFunc("GET /app/dokumente/{id}/preview", a.page(a.previewDocument))
	mux.HandleFunc("GET /app/dokumente/{id}/download", a.page(a.downloadDocument))
	mux.HandleFunc("GET /app/attachments/{id}", a.page(a.serveAttachment))
	mux.HandleFunc("GET /app/attachments/{id}/{variant}", a.page(a.serveAttachment))
	mux.HandleFunc("POST /app/attachments/delete", a.action(a.deleteAttachment))
	mux.HandleFunc("GET /app/abstimmungen", a.page(a.ballots))
	mux.HandleFunc("POST /app/abstimmungen", a.action(a.submitBallot))
	mux.HandleFunc("POST /app/abstimmungen/open", a.action(a.openBallot))
	mux.HandleFunc("POST /app/abstimmungen/close", a.action(a.closeBallot))
	mux.HandleFunc("GET /app/abstimmungen/{id}/protokoll", a.page(a.ballotProtocol))
	mux.HandleFunc("GET /app/uebergaben", a.page(a.handovers))
	mux.HandleFunc("POST /app/uebergaben", a.action(a.createHandover))
	mux.HandleFunc("POST /app/uebergaben/file", a.action(a.fileHandoverProtocol))
	mux.HandleFunc("GET /app/uebergaben/{id}/protokoll", a.page(a.handoverProtocol))
	mux.HandleFunc("GET /handover/{token}", a.handoverConfirmPage)
	mux.HandleFunc("POST /handover/{token}", a.confirmHandover)
	mux.HandleFunc("GET /app/kontakte", a.page(a.contacts))
	mux.HandleFunc("POST /app/kontakte", a.action(a.upsertManagedContact))
	mux.HandleFunc("POST /app/kontakte/delete", a.action(a.deactivateManagedContact))
	mux.HandleFunc("GET /app/anliegen", a.page(a.issues))
	mux.HandleFunc("GET /app/anliegen/board", a.page(a.issueBoard))
	mux.HandleFunc("GET /app/anliegen/{id}/photos/{index}", a.page(a.serveLegacyIssuePhoto))
	mux.HandleFunc("POST /app/anliegen", a.action(a.createIssue))
	mux.HandleFunc("POST /app/anliegen/comment", a.action(a.addIssueComment))
	mux.HandleFunc("POST /app/anliegen/comment/delete", a.action(a.deleteIssueComment))
	mux.HandleFunc("POST /app/anliegen/workflow", a.action(a.updateIssueWorkflow))
	mux.HandleFunc("GET /app/parking", a.page(a.parking))
	mux.HandleFunc("GET /app/parking/settings", a.authed(capabilityManageParking, a.parkingSettings))
	mux.HandleFunc("GET /app/parking/month/{month}", a.page(a.parkingMonth))
	mux.HandleFunc("GET /app/parking/month/{month}/export", a.page(a.parkingMonthExport))
	mux.HandleFunc("GET /app/parking/export/{year}", a.page(a.parkingStatement))
	mux.HandleFunc("POST /app/parking/settings", a.authedAction(capabilityManageParking, a.updateParkingSettings))
	mux.HandleFunc("POST /app/parking/month", a.action(a.updateParkingMonth))
	mux.HandleFunc("POST /app/parking/reminders", a.action(a.sendParkingReminders))
	mux.HandleFunc("GET /app/parking/charging/status", a.page(a.chargingStatus))
	mux.HandleFunc("POST /app/parking/charging/on", a.action(a.chargingOnAction))
	mux.HandleFunc("POST /app/parking/charging/off", a.action(a.chargingOffAction))
	mux.HandleFunc("POST /app/parking/charging/auto", a.action(a.chargingAutoAction))
	mux.HandleFunc("POST /app/parking/charging/settings", a.authedAction(capabilityManageParking, a.updateChargingSettings))
	mux.HandleFunc("POST /app/parking/charging/telegram/link", a.authedAction(capabilityManageParking, a.createTelegramLinkCode))
	mux.HandleFunc("POST /app/parking/charging/telegram/unlink", a.authedAction(capabilityManageParking, a.unlinkTelegramChat))
	mux.HandleFunc("GET /app/audit", a.page(a.auditLog))
	mux.HandleFunc("GET /app/settings", a.page(a.settingsHub))
	mux.HandleFunc("GET /app/settings/building", a.page(a.buildingSettings))
	mux.HandleFunc("POST /app/settings/building", a.action(a.updateBuildingSettings))
	mux.HandleFunc("POST /app/settings/building/hero", a.action(a.updateBuildingHero))
	mux.HandleFunc("POST /app/settings/building/hero/delete", a.action(a.deleteBuildingHero))
	mux.HandleFunc("POST /app/settings/building/units", a.action(a.upsertBuildingUnit))
	mux.HandleFunc("POST /app/settings/building/units/delete", a.action(a.deleteBuildingUnit))
	mux.HandleFunc("POST /app/settings/building/payment-status", a.action(a.updateUnitPaymentStatus))
	mux.HandleFunc("GET /app/settings/profile", a.page(a.profileSettings))
	mux.HandleFunc("POST /app/settings/profile", a.action(a.updateProfileSettings))
	mux.HandleFunc("GET /app/settings/notifications", a.page(a.notificationSettings))
	mux.HandleFunc("POST /app/settings/notifications", a.action(a.updateNotificationSettings))
	mux.HandleFunc("GET /app/settings/parking-access", a.page(a.parkingAccessSettings))
	mux.HandleFunc("POST /app/settings/parking-access", a.action(a.updateParkingAccess))
	mux.HandleFunc("GET /app/settings/users", a.authed(capabilityManageUsers, a.userSettings))
	mux.HandleFunc("POST /app/settings/users", a.authedAction(capabilityManageUsers, a.createInvite))
	mux.HandleFunc("POST /app/settings/users/edit", a.authedAction(capabilityManageUsers, a.editInvite))
	mux.HandleFunc("POST /app/settings/users/delete", a.authedAction(capabilityManageUsers, a.deleteInvite))
	mux.HandleFunc("GET /{tenant}", a.tenantPathRedirect)
	mux.HandleFunc("GET /{tenant}/{rest...}", a.tenantPathRedirect)
	return mux
}

// handler is the fully wrapped HTTP handler, middleware included. This is
// what main() serves and what the tests drive.
func (a *app) handler() http.Handler {
	// recoverAndLog is outermost so it captures panics and the final status from
	// every inner layer, including securityHeaders (HAUSV-141).
	return recoverAndLog(securityHeaders(a.routes()))
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
	if payload.Service != "hausv-org" || payload.Status != "ok" {
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
		env("OIDC_CLIENT_SECRET", ""),
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
	documentFileDir := env("DOC_FILE_DIR", defaultDocumentFileDir)
	documents, err := newDocumentStore(documentDataPath, documentFileDir)
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
	chargingTickInterval, err := parseDuration(env("CHARGING_TICK_INTERVAL", "30s"))
	if err != nil {
		return nil, fmt.Errorf("invalid CHARGING_TICK_INTERVAL")
	}
	chargingStaleAfter, err := parseDuration(env("CHARGING_STALE_AFTER", "10m"))
	if err != nil {
		return nil, fmt.Errorf("invalid CHARGING_STALE_AFTER")
	}
	chargingConfirmTimeout, err := parseDuration(env("CHARGING_CONFIRM_TIMEOUT", "2m"))
	if err != nil {
		return nil, fmt.Errorf("invalid CHARGING_CONFIRM_TIMEOUT")
	}
	chargingHAFailLimit, err := strconv.Atoi(env("CHARGING_HA_FAIL_LIMIT", "5"))
	if err != nil || chargingHAFailLimit < 1 {
		return nil, fmt.Errorf("invalid CHARGING_HA_FAIL_LIMIT")
	}
	telegramPollTimeout, err := parseDuration(env("TELEGRAM_POLL_TIMEOUT", "50s"))
	if err != nil {
		return nil, fmt.Errorf("invalid TELEGRAM_POLL_TIMEOUT")
	}
	telegramStore, err := newTelegramStore(env("TELEGRAM_DATA_PATH", "tmp/telegram.json"))
	if err != nil {
		return nil, err
	}

	// SQLite lives beside the JSON stores in the same data dir. The path is
	// derived from the JSON stores' committed data dir (PARKING_DATA_PATH) so no
	// new env/compose entry is needed — in prod that's /data/hausv.db, locally
	// tmp/hausv.db. DB_PATH overrides if ever set. It runs its migrations on boot.
	//
	// Non-fatal ON PURPOSE: if SQLite can't open, each migrated store falls back
	// to its JSON backend below, so boot can never brick on a DB problem
	// (HAUSV-170).
	dbPath := env("DB_PATH", "")
	if dbPath == "" {
		dbPath = filepath.Join(filepath.Dir(parkingDataPath), "hausv.db")
	}
	database, err := db.Open(dbPath)
	if err != nil {
		log.Printf("sqlite unavailable, using json stores: %v", err)
		database = nil
	}

	// Migrated stores prefer SQLite, importing existing JSON data once
	// (clobber-safe), and fall back to their JSON backend if SQLite is
	// unavailable. Low-stakes stores lead the migration (HAUSV-168/170).
	var activityBackend activityStorage = activity
	var profileBackend profileOverlayStorage = profileOverlays
	var notificationBackend notificationPrefStorage = notificationPrefs
	var unitPaymentBackend unitPaymentStatusStorage = unitPayments
	var contactBackend contactBookStorage = contacts
	var annReadBackend announcementReadStorage = announcementReads
	var annBackend announcementStorage = announcements
	var eventBackend eventStorage = events
	var handoverBackend handoverStorage = handovers
	var documentBackend documentStorage = documents
	// Kept concrete: the atomic protocol filer needs both SQL stores and only
	// works when they share one database (HAUSV-148).
	var sqlDocumentStore *store.SQLDocumentStore
	var sqlHandoverStore *store.SQLHandoverStore
	if database != nil {
		sqlActivity := newSQLActivityStore(database)
		if err := sqlActivity.ImportActivity(activity); err != nil {
			log.Printf("activity import to sqlite failed, keeping json: %v", err)
		} else {
			activityBackend = sqlActivity
		}
		sqlProfile := newSQLProfileOverlayStore(database)
		if err := sqlProfile.ImportOverlays(profileOverlays); err != nil {
			log.Printf("profile-overlay import to sqlite failed, keeping json: %v", err)
		} else {
			profileBackend = sqlProfile
		}
		sqlNotification := newSQLNotificationPrefStore(database)
		if err := sqlNotification.ImportPrefs(notificationPrefs); err != nil {
			log.Printf("notification-pref import to sqlite failed, keeping json: %v", err)
		} else {
			notificationBackend = sqlNotification
		}
		sqlUnitPayment := newSQLUnitPaymentStatusStore(database)
		if err := sqlUnitPayment.ImportStatuses(unitPayments); err != nil {
			log.Printf("unit-payment-status import to sqlite failed, keeping json: %v", err)
		} else {
			unitPaymentBackend = sqlUnitPayment
		}
		sqlContacts := newSQLContactBookStore(database)
		if err := sqlContacts.ImportContacts(contacts); err != nil {
			log.Printf("contact import to sqlite failed, keeping json: %v", err)
		} else {
			contactBackend = sqlContacts
		}
		sqlAnnRead := newSQLAnnouncementReadStore(database)
		if err := sqlAnnRead.ImportReads(announcementReads); err != nil {
			log.Printf("announcement-read import to sqlite failed, keeping json: %v", err)
		} else {
			annReadBackend = sqlAnnRead
		}
		sqlAnn := newSQLAnnouncementStore(database)
		if err := sqlAnn.ImportAnnouncements(announcements); err != nil {
			log.Printf("announcement import to sqlite failed, keeping json: %v", err)
		} else {
			annBackend = sqlAnn
		}
		sqlEvent := newSQLEventStore(database)
		if err := sqlEvent.ImportEvents(events); err != nil {
			log.Printf("event import to sqlite failed, keeping json: %v", err)
		} else {
			eventBackend = sqlEvent
		}
		sqlHandover := newSQLHandoverStore(database)
		if err := sqlHandover.ImportHandovers(handovers); err != nil {
			log.Printf("handover import to sqlite failed, keeping json: %v", err)
		} else {
			handoverBackend = sqlHandover
			sqlHandoverStore = sqlHandover
		}
		// Metadata only: the files themselves stay on disk in documentFileDir.
		sqlDocument := newSQLDocumentStore(database, documentFileDir)
		if err := sqlDocument.ImportDocuments(documents); err != nil {
			log.Printf("document import to sqlite failed, keeping json: %v", err)
		} else {
			documentBackend = sqlDocument
			sqlDocumentStore = sqlDocument
		}
	}

	// Filing a handover protocol writes a document AND the link on the handover.
	// With both stores in one database that happens in a single transaction;
	// otherwise fall back to the sequential (idempotent, non-atomic) path.
	var filer protocolFiler = newSequentialProtocolFiler(documentBackend, handoverBackend)
	if atomic := newSQLProtocolFiler(sqlDocumentStore, sqlHandoverStore); atomic != nil {
		filer = atomic
	} else {
		log.Printf("handover filing is not atomic: document/handover stores are not sharing sqlite")
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
		serviceAccessEnabled:  serviceProviderAccessEnabled(),
		sessionTTL:            sessionTTL,
		tokens:                auth.NewTokenStore(secret),
		sessions:              newSessionStore(secret),
		oidc:                  oidcLogin,
		oidcFlows:             auth.NewOIDCFlowStore(),
		mailer:                mailTransport,
		templates:             tmpl,
		db:                    database,
		announcementStore:     annBackend,
		announcementReadStore: annReadBackend,
		eventStore:            eventBackend,
		notificationPrefs:     notificationBackend,
		profileOverlays:       profileBackend,
		tenantOverrides:       tenantOverrides,
		tenantHeroDir:         tenantHeroDir,
		inviteStore:           invites,
		activityStore:         activityBackend,
		unitStore:             units,
		unitPaymentStore:      unitPaymentBackend,
		issueStore:            issues,
		attachmentStore:       attachments,
		contactStore:          contactBackend,
		auditStore:            auditStore,
		documentStore:         documentBackend,
		handoverStore:         handoverBackend,
		protocolFiler:         filer,
		voteStore:             votes,
		voteReminderInterval:  voteReminderInterval,
		parkingStore:          parkingStore,
		parkingSampleInterval: parkingSampleInterval,
		parkingHistoryStart:   parkingHistoryStart,

		chargingTickInterval:   chargingTickInterval,
		chargingStaleAfter:     chargingStaleAfter,
		chargingConfirmTimeout: chargingConfirmTimeout,
		chargingHAFailLimit:    chargingHAFailLimit,
		chargingHAFails:        map[string]int{},
		chargingShadow:         map[string]chargingControllerState{},
		chargingShadowPlug:     map[string]bool{},
		chargingEvents:         &chargingEventRing{},
		chargingLastPoll:       map[string]time.Time{},

		telegram:            telegram.New(env("TELEGRAM_API_BASE_URL", ""), env("TELEGRAM_BOT_TOKEN", "")),
		telegramStore:       telegramStore,
		telegramPollTimeout: telegramPollTimeout,
	}, nil
}

// serviceProviderAccessEnabled is the single runtime launch gate for external
// Dienstleister. Missing, empty and unrecognized values stay closed.
func serviceProviderAccessEnabled() bool {
	return parseBool(env("SERVICE_PROVIDER_ACCESS_ENABLED", "false"))
}

func (a *app) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"service":"hausv-org","status":"ok"}`)
}

func favicon(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = io.WriteString(w, web.FaviconSVG)
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
	// Bound a leaked feed URL's lifetime (HAUSV-147). Calendar clients poll a
	// subscription for months, so the cap is generous rather than short — a
	// re-subscribe (new URL) refreshes it. IssuedAt==0 is a pre-expiry token; we
	// accept it rather than lock existing subscribers out.
	if payload.IssuedAt > 0 && time.Since(time.Unix(payload.IssuedAt, 0)) > maxCalendarFeedAge {
		return calendarFeedPayload{}, false
	}
	return payload, true
}

// maxCalendarFeedAge caps how long a signed calendar-feed URL stays valid.
const maxCalendarFeedAge = 400 * 24 * time.Hour

// signedCalendarFeedTokenAt mints a feed token as if issued at `at`. Test helper
// for the expiry check (HAUSV-147); not used in production.
func (a *app) signedCalendarFeedTokenAt(email, tenantSlug string, at time.Time) string {
	payload := calendarFeedPayload{Email: normalizeEmail(email), TenantSlug: normalizeSlug(tenantSlug), IssuedAt: at.Unix()}
	raw, _ := json.Marshal(payload)
	signedPart := "v1." + base64.RawURLEncoding.EncodeToString(raw)
	sig, _ := a.signCalendarFeed(signedPart)
	return signedPart + "." + base64.RawURLEncoding.EncodeToString(sig)
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
			if (strings.TrimSpace(item.ServiceProposal) == "" && item.ServiceProposedStart.IsZero()) || !a.canViewIssueForActor(tenantSlug, item, email, role) {
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
	description := strings.TrimSpace(item.ServiceProposal)
	if description != "" {
		description += "\n\n"
	}
	description += "Anliegen: " + item.Title
	if status := strings.TrimSpace(item.Status); status != "" {
		description += "\nStatus: " + status
	}

	// A structured appointment becomes a real dated VEVENT; a legacy free-text
	// proposal stays a dateless VTODO (HAUSV-128).
	if !item.ServiceProposedStart.IsZero() {
		end := item.ServiceProposedEnd
		if end.IsZero() {
			end = item.ServiceProposedStart.Add(time.Hour)
		}
		calendarLine(b, "BEGIN", "VEVENT")
		calendarLine(b, "UID", "issue-proposal-"+item.ID+"@"+tenant.Slug+".hausv.org")
		calendarLine(b, "DTSTAMP", calendarDateTime(now))
		calendarLine(b, "DTSTART", calendarDateTime(item.ServiceProposedStart))
		calendarLine(b, "DTEND", calendarDateTime(end))
		calendarLine(b, "SUMMARY", "Termin: "+item.Title)
		calendarLine(b, "DESCRIPTION", description)
		calendarLine(b, "END", "VEVENT")
		return
	}

	calendarLine(b, "BEGIN", "VTODO")
	calendarLine(b, "UID", "issue-proposal-"+item.ID+"@"+tenant.Slug+".hausv.org")
	calendarLine(b, "DTSTAMP", calendarDateTime(now))
	calendarLine(b, "SUMMARY", "Terminvorschlag: "+item.Title)
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

func (a *app) portal(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
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

func residentDirectoryRole(role string) bool {
	switch normalizeRole(role) {
	case roleOwner, roleRenter, roleResident:
		return true
	default:
		return false
	}
}

func (a *app) handleIssueServiceAssignmentChange(r *http.Request, tenant tenantConfig, before residentIssue, after residentIssue, actorEmail string, actorRole string) {
	oldAssignee := normalizeEmail(before.AssigneeEmail)
	newAssignee := normalizeEmail(after.AssigneeEmail)
	if !a.serviceAccessEnabled && newAssignee != oldAssignee && a.shouldInviteServiceProvider(tenant.Slug, newAssignee) {
		return
	}
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

func (a *app) serviceProviderAccessClosedFor(tenantSlug string, email string) bool {
	return a != nil && !a.serviceAccessEnabled && a.isServiceProviderPrincipal(tenantSlug, email)
}

func (a *app) ensureServiceProviderInvite(tenantSlug string, email string) (bool, error) {
	if a == nil || !a.serviceAccessEnabled {
		return false, errServiceProviderAccessClosed
	}
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
	if a == nil || !a.serviceAccessEnabled {
		return errServiceProviderAccessClosed
	}
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
	recipients := []string{issue.AuthorEmail}
	if a.serviceAccessEnabled || !a.shouldInviteServiceProvider(tenant.Slug, normalizeEmail(issue.AssigneeEmail)) {
		recipients = append(recipients, issue.AssigneeEmail)
	}
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
		if a.serviceProviderAccessClosedFor(event.Tenant.Slug, recipient) {
			continue
		}
		if a.notificationPrefs != nil && !a.notificationPrefs.EmailEnabled(recipient, event.Event) {
			continue
		}
		out = append(out, recipient)
	}
	return out
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
		{"Monat", "Zeitraum", "kWh gesamt", "kWh Überschuss", "kWh Normal", "aWATTar Ø", "Effektivpreis", "Strom (Normal)", "Überschuss", "Netzgeb.", "Basis", "Summe", "Status"},
	}
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	for _, month := range statement.Months {
		if err := writer.Write([]string{month.MonthLabel, month.PeriodLabel, month.KWh, month.SurplusKWh, month.NormalKWh, month.AverageAwattar, month.EffectivePrice, month.EnergyCost, month.SurplusCost, month.GridCost, month.BaseFee, month.TotalCost, month.PaidLabel}); err != nil {
			return err
		}
	}
	if err := writer.Write([]string{"Gesamt", "", statement.TotalKWh, statement.SurplusKWh, statement.NormalKWh, "", "", statement.EnergyCost, statement.SurplusCost, statement.GridCost, statement.BaseFee, statement.TotalCost, ""}); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

// writeParkingMonthCSV renders one month's settlement: the monthly summary plus
// the hour-by-hour detail that mirrors the on-screen month view, so an exported
// total always matches what the page shows (HAUSV-164).
func writeParkingMonthCSV(w io.Writer, tenant tenantConfig, view parkingMonthDetailView) error {
	writer := csv.NewWriter(w)
	writer.Comma = ';'
	s := view.Summary
	status := "offen"
	if s.Paid {
		status = "bezahlt"
		if s.PaidAtLabel != "" {
			status += " am " + s.PaidAtLabel
		}
		if s.PaidBy != "" {
			status += " von " + s.PaidBy
		}
	}
	rows := [][]string{
		{"WEG Portal Parkplatzabrechnung – Monat"},
		{"Gebäude", tenant.Name},
		{"Adresse", tenant.Address},
		{"Monat", view.MonthLabel},
		{"Tarif", view.GridFeeLabel},
		{"Erstellt", formatLocalDateTime(time.Now())},
		{"Status", status},
	}
	if s.Partial {
		rows = append(rows, []string{"Hinweis", "Messdaten unvollständig – die Abdeckung beginnt nicht am Monatsanfang; die Summe kann Lücken enthalten."})
	}
	rows = append(rows,
		[]string{},
		[]string{"Monatssumme", "kWh gesamt", "kWh Überschuss", "kWh Normal", "Strom", "Überschuss", "Netzgeb.", "Basis", "Summe"},
		[]string{"", s.KWh, s.SurplusKWh, s.NormalKWh, s.EnergyCost, s.SurplusCost, s.GridCost, s.BaseFee, s.TotalCost},
		[]string{},
		[]string{"Stunde", "kWh", "Überschuss-kWh", "aWATTar Ø", "Strom", "Netzgeb.", "Summe"},
	)
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	for _, h := range view.Hours {
		surplus := "—"
		if h.HasSurplus {
			surplus = h.SurplusKWh
		}
		if err := writer.Write([]string{h.AtLabel, h.KWh, surplus, h.AverageAwattar, h.EnergyCost, h.GridCost, h.TotalCost}); err != nil {
			return err
		}
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
	// Fail closed (HAUSV-143): a POST with neither Origin nor Referer is not a
	// normal same-origin browser form submit — modern browsers always send
	// Origin on POST. SameSite=Lax already blocks the cross-site case, so this
	// is defense in depth.
	return false
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

func unreadAnnouncementCount(items []announcement, lastSeen time.Time, now time.Time) int {
	count := 0
	for _, item := range items {
		if announcementUnread(item, lastSeen, now) {
			count++
		}
	}
	return count
}

func serviceProviderIssueStatuses() []string {
	return []string{issueStatusAccepted, issueStatusScheduled, issueStatusProgress, issueStatusDone}
}

// issueAppointmentInput formats a proposed appointment time for a datetime-local
// form input, or "" when unset.
func issueAppointmentInput(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return formatLocalDateTimeInput(t)
}

// formatIssueAppointment renders the proposed appointment for display, or "" when
// no start is set.
func formatIssueAppointment(start, end time.Time) string {
	if start.IsZero() {
		return ""
	}
	out := formatLocalDateTime(start)
	if !end.IsZero() {
		out += " – " + formatLocalDateTime(end)
	}
	return out
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

func (a *app) settingsHub(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
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

func (a *app) buildingSettings(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role, profile, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	buildingMsg, buildingOK := buildingSettingsMessage(r.URL.Query().Get("building"))
	heroMsg, heroOK := buildingHeroMessage(r.URL.Query().Get("hero"))
	unitMsg, unitOK := buildingUnitMessage(r.URL.Query().Get("unit"))
	paymentMsg, paymentOK := unitPaymentStatusMessage(r.URL.Query().Get("payment"))
	units := a.unitStore.ListTenant(tenant.Slug)
	billableWeight := billableUnitWeight(units)
	fairUseExceeded := billableWeight > fairUseFreeUnits*unitBillableFullPPM
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
		"FairUseFreeUnits": fairUseFreeUnits,
		"FairUseExceeded":  fairUseExceeded,
		"UnitsEmpty":       emptyState("Noch keine Einheiten", "Angelegte Einheiten erscheinen hier mit Anteil und Kontaktlinks."),
		"PaymentMsg":       paymentMsg,
		"PaymentOK":        paymentOK,
		"PaymentRows":      a.unitPaymentStatusViewsForUnits(tenant.Slug, units),
		"HasPaymentRows":   len(units) > 0,
	})
}

func (a *app) auditLog(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
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

func (a *app) updateBuildingSettings(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
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

func (a *app) updateBuildingHero(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
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

func (a *app) deleteBuildingHero(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
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

func (a *app) upsertBuildingUnit(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
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
	// Add/replace under one lock so a concurrent unit add/delete isn't lost to a
	// whole-slice overwrite (HAUSV-145).
	duplicate, err := a.unitStore.UpsertUnit(tenant.Slug, origID, item)
	if err != nil {
		log.Printf("unit save failed for %s: %v", tenant.Slug, err)
		http.Redirect(w, r, "/app/settings/building?unit=error", http.StatusSeeOther)
		return
	}
	if duplicate {
		http.Redirect(w, r, "/app/settings/building?unit=duplicate", http.StatusSeeOther)
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

func (a *app) deleteBuildingUnit(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
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
	// Remove under one lock (HAUSV-145).
	removed, removedUnit, err := a.unitStore.DeleteUnit(tenant.Slug, deleteID)
	if err != nil {
		log.Printf("unit delete failed for %s: %v", tenant.Slug, err)
		http.Redirect(w, r, "/app/settings/building?unit=error", http.StatusSeeOther)
		return
	}
	if !removed {
		http.Redirect(w, r, "/app/settings/building?unit=missing", http.StatusSeeOther)
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

func (a *app) updateUnitPaymentStatus(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
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

func (a *app) buildingSettingsContext(w http.ResponseWriter, ac authCtx) (tenantConfig, string, string, userProfile, bool) {
	if !hasCapability(ac.role, capabilityManageBuilding) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return tenantConfig{}, "", "", userProfile{}, false
	}
	return ac.tenant, ac.email, ac.role, a.profileForTenant(ac.email, ac.tenant.Slug), true
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

func (a *app) profileSettings(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
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

func (a *app) updateProfileSettings(w http.ResponseWriter, r *http.Request, ac authCtx) {
	email, role := ac.email, ac.role
	if denyServiceProviderArea(w, role) {
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

func (a *app) notificationSettings(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
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

func (a *app) updateNotificationSettings(w http.ResponseWriter, r *http.Request, ac authCtx) {
	email, role := ac.email, ac.role
	if denyServiceProviderArea(w, role) {
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

func (a *app) userSettings(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
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
	case "reverted":
		return "Zugang auf die Konfiguration zurückgesetzt.", true
	case "not_editable":
		return "Dieser Eintrag kann hier nicht geändert werden.", false
	case "protected":
		return "Notfall-Admins (ADMIN_EMAILS) sind geschützt und können hier nicht geändert werden.", false
	case "last_admin":
		return "Die letzte Administrator-Rolle kann nicht entfernt werden.", false
	case "self_lockout":
		return "Sie können sich nicht selbst die Administrator-Rolle entziehen.", false
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
	beforeAuth, _ := normalizeAuthMethods(before.AuthMethods)
	afterAuth, _ := normalizeAuthMethods(after.AuthMethods)
	if strings.Join(beforeAuth, ",") != strings.Join(afterAuth, ",") {
		changed = append(changed, "Anmeldung")
	}
	if before.Deactivated != after.Deactivated {
		changed = append(changed, "Aktivierung")
	}
	if len(changed) == 0 {
		changed = append(changed, "Metadaten")
	}
	return changed
}

func (a *app) createInvite(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role := ac.tenant, ac.email, ac.role
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
	if !a.serviceAccessEnabled && isServiceProviderRole(inviteRole) {
		http.Error(w, serviceProviderAccessClosedMessage, http.StatusForbidden)
		return
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
		AuthMethods: parseAuthMethodForm(r.Form),
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

func (a *app) editInvite(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role := ac.tenant, ac.email, ac.role
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	orig := normalizeEmail(r.FormValue("orig_email"))
	existing, isInvite := a.inviteStore.Get(orig)
	envProfile, isEnv := a.profiles[orig]
	if !isInvite && !isEnv {
		a.redirectInvite(w, r, "not_editable")
		return
	}
	// Break-glass ADMIN_EMAILS admins are read-only here: they are the recovery
	// anchor and must never be editable or lockable-out from the app (HAUSV-163).
	if _, isBreakGlass := a.admins[orig]; isBreakGlass {
		a.redirectInvite(w, r, "protected")
		return
	}

	// Resolve the current effective profile for this tenant, preferring an
	// existing app override, otherwise the env directory record. InviteStore.Get
	// is keyed by email across ALL tenants, so the HasTenant guard stops a manager
	// of tenant A from editing (or, via rename, minting) a tenant-B record
	// (HAUSV-135).
	var effectiveProfile userProfile
	if isInvite {
		if !existing.HasTenant(tenant.Slug) {
			a.redirectInvite(w, r, "not_editable")
			return
		}
		effectiveProfile = existing.ForTenant(tenant.Slug)
	} else {
		env := a.withProfileOverlay(envProfile)
		if !env.HasTenant(tenant.Slug) {
			a.redirectInvite(w, r, "not_editable")
			return
		}
		effectiveProfile = env.ForTenant(tenant.Slug)
	}
	if normalizeRole(effectiveProfile.Role) == roleAdmin && !hasCapability(role, capabilityPlatformAdmin) {
		a.redirectInvite(w, r, "not_editable")
		return
	}

	// Config-sourced users keep their configured email as a fixed identity; only
	// pure app invites may be renamed.
	newEmail := orig
	if !isEnv {
		newEmail = normalizeEmail(r.FormValue("email"))
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
	}

	newRole := normalizeRole(r.FormValue("role"))
	if newRole == "" {
		newRole = roleResident
	}
	if !a.serviceAccessEnabled && (isServiceProviderRole(effectiveProfile.Role) || isServiceProviderRole(newRole)) {
		http.Error(w, serviceProviderAccessClosedMessage, http.StatusForbidden)
		return
	}
	if !canAssignUserRole(role, newRole) {
		a.redirectInvite(w, r, "forbidden_role")
		return
	}

	newDeactivated := r.FormValue("deactivated") != ""
	wasAdmin := normalizeRole(effectiveProfile.Role) == roleAdmin
	self := orig == normalizeEmail(actorEmail)

	// Self-lockout: an admin cannot demote themselves, and nobody can deactivate
	// their own account mid-session (HAUSV-163).
	if self && ((wasAdmin && newRole != roleAdmin) || (newDeactivated && !effectiveProfile.Deactivated)) {
		a.redirectInvite(w, r, "self_lockout")
		return
	}
	// Last-admin guard: never let the final active admin lose admin access, whether
	// by demotion or deactivation.
	if wasAdmin && (newRole != roleAdmin || newDeactivated) {
		remaining := a.adminEmails(tenant.Slug)
		delete(remaining, orig)
		if len(remaining) == 0 {
			a.redirectInvite(w, r, "last_admin")
			return
		}
	}

	// Seed the persisted record: an existing override keeps its stored fields; a
	// freshly adopted config user is seeded from its env record so tenants and
	// memberships survive (HAUSV-163 adopt-on-edit).
	updated := existing
	if !isInvite {
		updated = envProfile
	}
	updated.Email = newEmail
	updated.Title = strings.TrimSpace(r.FormValue("title"))
	updated.FirstName = strings.TrimSpace(r.FormValue("first_name"))
	updated.LastName = strings.TrimSpace(r.FormValue("last_name"))
	updated.Role = newRole
	updated.Permissions = parsePermissionForm(r.Form)
	updated.AuthMethods = parseAuthMethodForm(r.Form)
	updated.Deactivated = newDeactivated
	if len(updated.Tenants) == 0 {
		updated.Tenants = []string{tenant.Slug}
	}
	if isEnv {
		// Mark the override so it wins over the env record in directoryProfile.
		updated.Adopted = true
	}

	if isInvite {
		changed, err := a.inviteStore.Update(orig, updated)
		if err != nil {
			a.redirectInvite(w, r, "exists")
			return
		}
		if !changed {
			a.redirectInvite(w, r, "not_editable")
			return
		}
	} else {
		added, err := a.inviteStore.Add(updated)
		if err != nil {
			log.Printf("adopt persistence failed for %s: %v", redactedEmail(orig), err)
			a.redirectInvite(w, r, "error")
			return
		}
		if !added {
			// Raced with a concurrent adopt; apply as an update instead.
			if _, err := a.inviteStore.Update(orig, updated); err != nil {
				a.redirectInvite(w, r, "error")
				return
			}
		}
	}

	source := "App"
	if !isInvite {
		source = "Konfiguration (übernommen)"
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionInviteUpdate,
		TargetType: "user",
		TargetID:   updated.Email,
		Summary:    "Zugang geändert",
		Details: map[string]string{
			"source":           source,
			"changed_fields":   strings.Join(auditChangedUserFields(effectiveProfile, updated), ", "),
			"role_from":        normalizeRole(effectiveProfile.Role),
			"role_to":          normalizeRole(updated.Role),
			"permissions_from": strings.Join(permissionLabelList(effectiveProfile.Permissions), ", "),
			"permissions_to":   strings.Join(permissionLabelList(updated.Permissions), ", "),
			"auth_from":        strings.Join(authMethodsLabelList(effectiveProfile.AuthMethods), ", "),
			"auth_to":          strings.Join(authMethodsLabelList(updated.AuthMethods), ", "),
		},
	})
	a.redirectInvite(w, r, "updated")
}

func (a *app) deleteInvite(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role := ac.tenant, ac.email, ac.role
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
	// Cross-tenant guard: Get is email-keyed across all tenants (HAUSV-135).
	if !existing.HasTenant(tenant.Slug) {
		a.redirectInvite(w, r, "not_editable")
		return
	}
	effectiveProfile := existing.ForTenant(tenant.Slug)
	if !a.serviceAccessEnabled && isServiceProviderRole(effectiveProfile.Role) {
		http.Error(w, serviceProviderAccessClosedMessage, http.StatusForbidden)
		return
	}
	if normalizeRole(effectiveProfile.Role) == roleAdmin && !hasCapability(role, capabilityPlatformAdmin) {
		a.redirectInvite(w, r, "not_editable")
		return
	}
	// Break-glass admins are never deletable from the app (HAUSV-163). They are
	// not adopted into the store, so this is defence-in-depth over the isInvite
	// check above.
	if _, isBreakGlass := a.admins[deleteEmail]; isBreakGlass {
		a.redirectInvite(w, r, "protected")
		return
	}
	// Last-admin / self-lockout guard: never remove the final Admin, and never let
	// an admin delete their own admin access (HAUSV-163).
	if normalizeRole(effectiveProfile.Role) == roleAdmin {
		if deleteEmail == normalizeEmail(actorEmail) {
			a.redirectInvite(w, r, "self_lockout")
			return
		}
		remaining := a.adminEmails(tenant.Slug)
		delete(remaining, deleteEmail)
		if len(remaining) == 0 {
			a.redirectInvite(w, r, "last_admin")
			return
		}
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
	// Deleting an override on a config-sourced user reverts it to the env record
	// rather than removing the person entirely.
	status := "deleted"
	summary := "Einladung gelöscht"
	if _, isEnv := a.profiles[deleteEmail]; isEnv {
		status = "reverted"
		summary = "Auf Konfiguration zurückgesetzt"
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: actorEmail,
		ActorRole:  role,
		Action:     auditActionInviteDelete,
		TargetType: "user",
		TargetID:   deleteEmail,
		Summary:    summary,
		Details: map[string]string{
			"role_from":        normalizeRole(existing.Role),
			"permissions_from": strings.Join(permissionLabelList(existing.Permissions), ", "),
		},
	})
	a.redirectInvite(w, r, status)
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
	if _, ok := data["ServiceProviderAccessEnabled"]; !ok {
		data["ServiceProviderAccessEnabled"] = a.serviceAccessEnabled
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
	var invite userProfile
	hasInvite := false
	if a.inviteStore != nil {
		invite, hasInvite = a.inviteStore.Get(email)
	}
	envProfile, hasEnv := a.profiles[email]
	switch {
	case hasInvite && hasEnv:
		// Both a store record and an env record exist. The store record wins only
		// when it is an admin-sanctioned adoption (HAUSV-163 adopt-on-edit);
		// otherwise the env directory stays authoritative so a plain or legacy
		// store record can never escalate an env user (HAUSV-135).
		if invite.Adopted {
			return a.withProfileOverlay(invite), true
		}
		return a.withProfileOverlay(envProfile), true
	case hasInvite:
		return a.withProfileOverlay(invite), true
	case hasEnv:
		return a.withProfileOverlay(envProfile), true
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
	email = normalizeEmail(email)
	// Break-glass: an ADMIN_EMAILS bootstrap admin can always sign in, even if an
	// app-managed override would otherwise deny it. This is the documented
	// recovery path — env config stays authoritative for login (HAUSV-163).
	if _, ok := a.admins[email]; ok {
		return true
	}
	if profile, ok := a.directoryProfile(email); ok && profile.HasTenant(tenantSlug) {
		if profile.Deactivated {
			return false
		}
		if !a.serviceAccessEnabled && isServiceProviderRole(profile.ForTenant(tenantSlug).Role) {
			return false
		}
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
	if profile.Deactivated {
		// Break-glass ADMIN_EMAILS admins can never be deactivated in-app, but
		// exempt them here too so a manual/legacy override can't strand recovery.
		if _, isBreakGlass := a.admins[normalizeEmail(email)]; !isBreakGlass {
			return false
		}
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
	// App-managed (invite store) rows come first so an adopted config user shows
	// its editable override rather than the read-only env row (HAUSV-163).
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
			if _, isEnv := a.profiles[email]; isEnv && !profile.Adopted {
				// A non-adopted store record never shadows an env user; the env
				// loop renders the authoritative row instead (HAUSV-135).
				continue
			}
			row := userRowFrom(profile.ForTenant(tenantSlug))
			row.Editable = true
			if _, isEnv := a.profiles[email]; isEnv {
				// Adopted config user: an app override layered over the env record.
				row.IsConfig = true
			}
			rows = append(rows, row)
			seen[email] = struct{}{}
		}
	}
	for email, profile := range a.profiles {
		if _, ok := seen[email]; ok {
			continue
		}
		profile = a.withProfileOverlay(profile)
		if !profile.HasTenant(tenantSlug) {
			continue
		}
		row := userRowFrom(profile.ForTenant(tenantSlug))
		// Config-sourced rows stay Editable=false so pages that only mutate the
		// invite store (e.g. parking access) keep them read-only. The Benutzer &
		// Rechte page opens the edit dialog for non-protected config users via
		// adopt-on-edit (HAUSV-163).
		row.IsConfig = true
		if _, isBreakGlass := a.admins[email]; isBreakGlass {
			// ADMIN_EMAILS bootstrap admins are the break-glass anchor and must
			// never be lockable-out from the app.
			row.Protected = true
		}
		rows = append(rows, row)
		seen[email] = struct{}{}
	}
	for email := range a.admins {
		if _, ok := seen[email]; ok {
			continue
		}
		row := userRowFrom(userProfile{Email: email, Role: roleAdmin, Status: "Aktiv", Tenants: []string{tenantSlug}})
		row.IsConfig = true
		row.Protected = true
		rows = append(rows, row)
		seen[email] = struct{}{}
	}
	for email := range a.allowed {
		if _, ok := seen[email]; ok {
			continue
		}
		row := userRowFrom(userProfile{Email: email, Role: roleResident, Status: "Eingeladen", Tenants: []string{tenantSlug}})
		row.IsConfig = true
		rows = append(rows, row)
		seen[email] = struct{}{}
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
	// Deactivation wins over the activity-derived status.
	for i := range rows {
		if rows[i].Deactivated {
			rows[i].Status = "Deaktiviert"
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
	At          time.Time
	KWh         float64
	PriceEUR    float64
	EnergyCost  float64
	GridCost    float64
	SurplusKWh  float64
	SurplusCost float64
}

func calculateParkingHourlyUsage(energySamples []parkingNumericSample, priceSamples []parkingNumericSample, gridFeeEURPerKWh float64, now time.Time) []parkingHourUsage {
	return calculateParkingHourlyUsageWithSettings(energySamples, priceSamples, nil, parkingSettings{GridFeeEURPerKWh: gridFeeEURPerKWh}, now, time.Local)
}

// surplusOverlapFraction returns which fraction of [from, to) lies inside a
// surplus charging session. Sessions must be surplus-mode only; an open
// session (zero End) counts up to `now`. The controller writes a meter sample
// at every session boundary, so for controller-created intervals this is
// exactly 0 or 1 — the proportional path only smooths foreign intervals
// (e.g. a boundary snapshot that failed while HA was down).
func surplusOverlapFraction(sessions []chargingSession, from, to, now time.Time) float64 {
	total := to.Sub(from).Seconds()
	if total <= 0 || len(sessions) == 0 {
		return 0
	}
	overlap := 0.0
	for _, session := range sessions {
		start := session.Start
		end := session.End
		if end.IsZero() {
			end = now
		}
		if !start.Before(to) || !end.After(from) {
			continue
		}
		if start.Before(from) {
			start = from
		}
		if end.After(to) {
			end = to
		}
		if seconds := end.Sub(start).Seconds(); seconds > 0 {
			overlap += seconds
		}
	}
	if overlap <= 0 {
		return 0
	}
	if frac := overlap / total; frac < 1 {
		return frac
	}
	return 1
}

func calculateParkingHourlyUsageWithSettings(energySamples []parkingNumericSample, priceSamples []parkingNumericSample, sessions []chargingSession, settings parkingSettings, now time.Time, loc *time.Location) []parkingHourUsage {
	if loc == nil {
		loc = time.Local
	}
	settings = normalizeParkingSettings(settings)
	keepAfter := now.AddDate(-1, -1, 0)
	energySamples = normalizeNumericSamples(append([]parkingNumericSample(nil), energySamples...), keepAfter)
	priceSamples = normalizeNumericSamples(append([]parkingNumericSample(nil), priceSamples...), keepAfter)
	surplusSessions := make([]chargingSession, 0, len(sessions))
	for _, session := range normalizeChargingSessions(append([]chargingSession(nil), sessions...), keepAfter) {
		if session.Mode == store.ChargingModeSurplus {
			surplusSessions = append(surplusSessions, session)
		}
	}
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
			surplusKWh := kWh * surplusOverlapFraction(surplusSessions, cursor, segmentEnd, now)
			normalKWh := kWh - surplusKWh
			hour := cursor.Truncate(time.Hour)
			out = append(out, parkingHourUsage{
				At:          hour,
				KWh:         kWh,
				PriceEUR:    price,
				EnergyCost:  price * normalKWh,
				GridCost:    tariff.GridFeeEURPerKWh * normalKWh,
				SurplusKWh:  surplusKWh,
				SurplusCost: surplusRate(tariff) * surplusKWh,
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
			merged[len(merged)-1].SurplusKWh += item.SurplusKWh
			merged[len(merged)-1].SurplusCost += item.SurplusCost
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
	hours := calculateParkingHourlyUsageWithSettings(data.EnergySamples, data.PriceSamples, data.ChargingSessions, data.Settings, now, loc)
	if len(hours) == 0 {
		return nil
	}
	type aggregate struct {
		month       string
		kWh         float64
		energyCost  float64
		gridCost    float64
		surplusKWh  float64
		surplusCost float64
		first       time.Time
		last        time.Time
		hourCount   int
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
		agg.surplusKWh += hour.SurplusKWh
		agg.surplusCost += hour.SurplusCost
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
		total := agg.energyCost + agg.gridCost + agg.surplusCost + tariff.BaseFeeEUR
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
		total := agg.energyCost + gridCost + agg.surplusCost + tariff.BaseFeeEUR
		normalKWh := agg.kWh - agg.surplusKWh
		averageAwattar := 0.0
		effectivePrice := 0.0
		if normalKWh > 0 {
			averageAwattar = agg.energyCost / normalKWh
		}
		if agg.kWh > 0 {
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
			SurplusKWhValue:  agg.surplusKWh,
			SurplusCostValue: agg.surplusCost,
			NormalKWhValue:   normalKWh,
			KWh:              formatKWh(agg.kWh),
			EnergyCost:       formatEUR(agg.energyCost),
			GridCost:         formatEUR(gridCost),
			BaseFee:          formatEUR(tariff.BaseFeeEUR),
			TotalCost:        formatEUR(total),
			SurplusKWh:       formatKWh(agg.surplusKWh),
			SurplusCost:      formatEUR(agg.surplusCost),
			NormalKWh:        formatKWh(normalKWh),
			HasSurplus:       agg.surplusKWh > 0,
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
	hours := calculateParkingHourlyUsageWithSettings(data.EnergySamples, data.PriceSamples, data.ChargingSessions, data.Settings, now, loc)
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
		total := hour.EnergyCost + hour.GridCost + hour.SurplusCost
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
		total := hour.EnergyCost + hour.GridCost + hour.SurplusCost
		normalKWh := hour.KWh - hour.SurplusKWh
		averageAwattar := 0.0
		if normalKWh > 0 {
			averageAwattar = hour.EnergyCost / normalKWh
		}
		tariff := parkingTariffAt(data.Settings, hour.At, loc)
		chartPercent := 0
		if maxTotal > 0 {
			chartPercent = int(total / maxTotal * 100)
			if chartPercent < 2 && total > 0 {
				chartPercent = 2
			}
		}
		totalTitle := "Summe: " + formatPreciseEUR(total) + " = Strom " + formatPreciseEUR(hour.EnergyCost) + " + Netzgebühr " + formatPreciseEUR(hour.GridCost)
		if hour.SurplusKWh > 0 {
			totalTitle += " + Überschuss " + formatPreciseEUR(hour.SurplusCost)
		}
		view.Hours = append(view.Hours, parkingHourView{
			AtLabel:             formatDateTimeIn(hour.At, loc, deATShortDateTimeLayout),
			AtTitle:             formatDateTimeIn(hour.At, loc, deATDateTimeLayout) + " bis " + formatDateTimeIn(hour.At.Add(time.Hour), loc, deATTimeLayout),
			KWh:                 formatKWh(hour.KWh),
			KWhTitle:            "Verbrauch: " + formatPreciseKWh(hour.KWh),
			SurplusKWh:          formatKWh(hour.SurplusKWh),
			SurplusKWhTitle:     "PV-Überschuss: " + formatPreciseKWh(hour.SurplusKWh) + " × " + formatPreciseEURPerKWh(surplusRate(tariff)) + " = " + formatPreciseEUR(hour.SurplusCost),
			HasSurplus:          hour.SurplusKWh > 0,
			AverageAwattar:      formatEURPerKWh(averageAwattar),
			AverageAwattarTitle: "aWATTar Preis dieser Stunde: " + formatPreciseEURPerKWh(averageAwattar),
			EnergyCost:          formatEUR(hour.EnergyCost),
			EnergyCostTitle:     "Stromkosten: " + formatPreciseEUR(hour.EnergyCost) + " = " + formatPreciseKWh(normalKWh) + " × " + formatPreciseEURPerKWh(averageAwattar),
			GridCost:            formatEUR(hour.GridCost),
			GridCostTitle:       "Netzgebühr: " + formatPreciseEUR(hour.GridCost) + " = " + formatPreciseKWh(normalKWh) + " × " + formatPreciseEURPerKWh(tariff.GridFeeEURPerKWh),
			TotalCost:           formatEUR(total),
			TotalCostTitle:      totalTitle,
			WeightTitle:         "Relative Höhe der Stundensumme. 100% entspricht der teuersten Stunde dieses Monats.",
			ChartPercent:        chartPercent,
		})
	}
	view.HasHours = len(view.Hours) > 0
	view.Sessions = chargingSessionViews(data, 0, month)
	view.HasSessions = len(view.Sessions) > 0
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
		EmailAuthChecked: p.AllowsAuthMethod(authMethodEmail),
		OIDCAuthChecked:  p.AllowsAuthMethod(authMethodOIDC),
		Deactivated:      p.Deactivated,
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

// parseAuthMethodForm reads the login-method checkboxes. An empty selection
// falls back to both methods rather than persisting zero methods, so a save can
// never leave a user with no way to sign in (HAUSV-163).
func parseAuthMethodForm(values url.Values) []string {
	methods, err := normalizeAuthMethods(values["auth_methods"])
	if err != nil {
		return defaultAuthMethods()
	}
	return methods
}

// adminEmails returns the set of emails that currently resolve to the Admin role
// for the tenant, across env config and app-managed overrides. It backs the
// last-admin / self-lockout guard so the portal can never be left adminless
// (HAUSV-163).
func (a *app) adminEmails(tenantSlug string) map[string]struct{} {
	admins := map[string]struct{}{}
	for _, row := range a.userRows(tenantSlug) {
		if row.Deactivated {
			continue
		}
		if normalizeRole(row.Role) == roleAdmin {
			admins[normalizeEmail(row.Email)] = struct{}{}
		}
	}
	return admins
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

// ── package API for cmd/ ────────────────────────────────────────────────────
// The app type stays unexported: the composition root only needs to build it,
// start its workers and serve its handler.

// New builds the application from the environment.
func New() (*app, error) { return newApp() }

// Handler is the fully wrapped HTTP handler, middleware included.
func (a *app) Handler() http.Handler { return a.handler() }

// Addr is the listen address.
func (a *app) Addr() string { return a.addr }

// Close releases process-lifetime resources. Currently the SQLite handle; the
// JSON stores hold no OS handles between writes.
func (a *app) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

// RunHealthcheck is the container health probe.
func RunHealthcheck(target string) error { return runHealthcheck(target) }

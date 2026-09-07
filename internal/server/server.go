package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inspr-at/hausv-org/internal/ai"
	"github.com/inspr-at/hausv-org/internal/auth"
	"github.com/inspr-at/hausv-org/internal/authz"
	"github.com/inspr-at/hausv-org/internal/config"
	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/demo"
	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/homeassistant"
	appmail "github.com/inspr-at/hausv-org/internal/mail"
	"github.com/inspr-at/hausv-org/internal/view"
	"github.com/inspr-at/hausv-org/internal/web"

	"github.com/inspr-at/hausv-org/internal/integrations"
	"github.com/inspr-at/hausv-org/internal/mailintake"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/telegram"
	"github.com/inspr-at/hausv-org/internal/textutil"
	"github.com/inspr-at/hausv-org/internal/version"
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
	homeIdentityView        = view.HomeIdentityView
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
var notificationEventOptions = view.NotificationEventOptions
var paidLabel = view.PaidLabel
var parkingStatementTariffLabel = view.ParkingStatementTariffLabel
var permissionLabel = view.PermissionLabel
var roleClass = view.RoleClass
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
var germanDateLong = view.GermanDateLong
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
var parseOrganisations = config.ParseOrganisations
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
	capabilityManageEnergy        = authz.CapabilityManageEnergy
	capabilityControlEnergy       = authz.CapabilityControlEnergy
)

// ── extracted to authz ──────────────────────────────────────────────
// Aliases so the move needs zero call-site changes. Delete as callers migrate.
type (
	authorizationActor    = authz.Actor
	authorizationResource = authz.Resource
	capability            = authz.Capability
)

var canAssignUserRole = authz.CanAssignUserRole
var canCreateResidentIssue = authz.CanCreateResidentIssue
var canManageAnnouncements = authz.CanManageAnnouncements
var canManageContacts = authz.CanManageContacts
var canManageEvents = authz.CanManageEvents
var canManageHandovers = authz.CanManageHandovers
var canResidentTransition = authz.CanResidentTransition
var canServiceProviderTransition = authz.CanServiceProviderTransition
var canViewAudit = authz.CanViewAudit
var canViewFullAudit = authz.CanViewFullAudit
var can = authz.Can
var roleHasCapability = authz.RoleHasCapability
var isServiceProviderRole = authz.IsServiceProviderRole
var roleCanUseResidentAreas = authz.RoleCanUseResidentAreas

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
var normalizeIssueCommentKind = store.NormalizeIssueCommentKind
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
var sortHandovers = store.SortHandovers
var sortIssues = store.SortIssues
var writeImageAttachmentVariant = store.WriteImageAttachmentVariant
var writePrivateFile = store.WritePrivateFile

// Vocabulary constants now owned by the store; aliased so call sites are unchanged.
const (
	auditActionBuildingUpdate          = store.AuditActionBuildingUpdate
	auditActionPortalModulesUpdate     = store.AuditActionPortalModulesUpdate
	auditActionContactDelete           = store.AuditActionContactDelete
	auditActionContactSave             = store.AuditActionContactSave
	auditActionDocumentDownload        = store.AuditActionDocumentDownload
	auditActionDocumentReplace         = store.AuditActionDocumentReplace
	auditActionDocumentUpload          = store.AuditActionDocumentUpload
	auditActionAttachmentView          = store.AuditActionAttachmentView
	auditActionAttachmentDelete        = store.AuditActionAttachmentDelete
	auditActionIntegrationImport       = store.AuditActionIntegrationImport
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
	auditActionEventCreate             = store.AuditActionEventCreate
	auditActionEventUpdate             = store.AuditActionEventUpdate
	auditActionEventDelete             = store.AuditActionEventDelete
	auditActionLogin                   = store.AuditActionLogin
	auditActionContextSwitch           = store.AuditActionContextSwitch
	auditActionParkingMonth            = store.AuditActionParkingMonth
	auditActionParkingReminder         = store.AuditActionParkingReminder
	auditActionParkingSettings         = store.AuditActionParkingSettings
	auditActionUnitDelete              = store.AuditActionUnitDelete
	auditActionUnitPayment             = store.AuditActionUnitPayment
	auditActionUnitSave                = store.AuditActionUnitSave
	auditActionAnnualPeriodSave        = store.AuditActionAnnualPeriodSave
	auditActionAnnualPartiesImport     = store.AuditActionAnnualPartiesImport
	auditActionAnnualCostTypeSave      = store.AuditActionAnnualCostTypeSave
	auditActionAnnualBasesSave         = store.AuditActionAnnualBasesSave
	auditActionAnnualReceiptCreate     = store.AuditActionAnnualReceiptCreate
	auditActionAnnualReceiptAmount     = store.AuditActionAnnualReceiptAmount
	auditActionAnnualReceiptDelete     = store.AuditActionAnnualReceiptDelete
	auditActionAnnualPrepaymentSave    = store.AuditActionAnnualPrepaymentSave
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
	activityRecord             = store.ActivityRecord
	activityStore              = store.ActivityStore
	activityStorage            = store.ActivityStorage
	announcementRepository     = store.AnnouncementRepository
	profileOverlayStorage      = store.ProfileOverlayStorage
	notificationPrefStorage    = store.NotificationPrefStorage
	unitPaymentRepository      = store.UnitPaymentStatusRepository
	unitPaymentStatusStorage   = store.UnitPaymentStatusStorage
	contactBookRepository      = store.ContactBookRepository
	contactBookStorage         = store.ContactBookStorage
	announcementReadRepository = store.AnnouncementReadRepository
	announcementReadStorage    = store.AnnouncementReadStorage
	announcementStorage        = store.AnnouncementStorage
	eventRepository            = store.EventRepository
	eventStorage               = store.EventStorage
	handoverRepository         = store.HandoverRepository
	handoverStorage            = store.HandoverStorage
	identityRepository         = store.IdentityRepository
	documentStorage            = store.DocumentStorage
	protocolFiler              = store.ProtocolFiler
	attachmentStorage          = store.AttachmentStorage
	telegramStorage            = store.TelegramStorage
	unitStorage                = store.UnitStorage
	unitRepository             = store.UnitRepository
	voteRepository             = store.VoteRepository
	voteStorage                = store.VoteStorage
	issueStorage               = store.IssueStorage
	profileStorage             = store.ProfileStorage
	announcementReadStore      = store.AnnouncementReadStore
	announcementReadStoreData  = store.AnnouncementReadStoreData
	contactBookStore           = store.ContactBookStore
	contactBookStoreData       = store.ContactBookStoreData
	managedContact             = store.ManagedContact
	notificationPrefStore      = store.NotificationPrefStore
	notificationPrefStoreData  = store.NotificationPrefStoreData
	notificationPreferences    = store.NotificationPreferences
	profileOverlay             = store.ProfileOverlay
	profileOverlayStore        = store.ProfileOverlayStore
	profileOverlayStoreData    = store.ProfileOverlayStoreData
	unitPaymentStatus          = store.UnitPaymentStatus
	unitPaymentStatusData      = store.UnitPaymentStatusData
	unitPaymentStatusStore     = store.UnitPaymentStatusStore
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
var newSQLAttachmentStore = store.NewSQLAttachmentStore
var newSQLTelegramStore = store.NewSQLTelegramStore
var newSQLUnitStore = store.NewSQLUnitStore
var newSQLVoteStore = store.NewSQLVoteStore
var newSQLIssueStore = store.NewSQLIssueStore
var newSQLIdentityStore = store.NewSQLIdentityStore
var migrateLegacyIssuePhotos = store.MigrateLegacyIssuePhotos
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
	permissionEnergyView          = store.PermissionEnergyView
	permissionEnergyConfigure     = store.PermissionEnergyConfigure
	permissionEnergyControl       = store.PermissionEnergyControl
	permissionEnergyCaretaker     = store.PermissionEnergyCaretaker
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
	issueCommentKindNeutral       = store.IssueCommentKindNeutral
	issueCommentKindInformation   = store.IssueCommentKindInformation
	issueCommentKindQuestion      = store.IssueCommentKindQuestion
	issueCommentKindAnswer        = store.IssueCommentKindAnswer
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

const (
	serviceProviderAccessClosedMessage = "Dienstleister-Zugänge sind derzeit nicht verfügbar."
	serviceProviderAssessmentVersion   = "2026-07-26"
)

var errServiceProviderAccessClosed = errors.New("service provider access is disabled")

type app struct {
	baseURL                 string
	addr                    string
	rootDomain              string
	defaultTenant           string
	tenants                 map[string]tenantConfig
	organisations           map[string]config.OrganisationConfig
	tenantIdentities        map[string]store.TenantIdentity
	sessionSecure           bool
	allowed                 map[string]struct{}
	admins                  map[string]struct{}
	profiles                map[string]userProfile
	localDevLogin           bool
	demoLogin               bool
	demoLoginCode           string
	demoReset               func(context.Context, time.Time, io.Writer) (demo.SeedResult, error)
	serviceAccessEnabled    bool
	templExampleEnabled     bool
	sessionTTL              time.Duration
	tokens                  *tokenStore
	sessions                *sessionStore
	homeSetupTokens         *tokenStore
	homeSetupSessions       *sessionStore
	homeConnectorHashKey    []byte
	oidc                    *oidcLogin
	oidcFlows               *oidcFlowStore
	mailer                  mailer
	trustedProxies          trustedProxyAllowlist
	magicLinkDeliveryMu     sync.Mutex
	magicLinkDelivery       *magicLinkDeliveryQueue
	magicLinkDeliveryClosed bool
	authLimiterMu           sync.Mutex
	authLimiter             *authRateLimiter
	templates               *template.Template
	dataDir                 string
	announcementStore       announcementStorage
	announcementReadStore   announcementReadStorage
	eventStore              eventStorage
	notificationPrefs       notificationPrefStorage
	profileOverlays         profileOverlayStorage
	tenantOverrides         *tenantOverrideStore
	tenantHeroDir           string
	tenantHeroSeedDir       string
	// inviteStore serves app-managed user records. Backed by the person/house
	// N:N model when SQLite is available, otherwise by the JSON store
	// (HAUSV-169).
	inviteStore               profileStorage
	identityStore             *store.SQLIdentityStore
	activityStore             activityStorage
	annualStatementCostTypes  store.AnnualStatementCostTypeStorage
	annualStatementPeriods    store.AnnualStatementPeriodStorage
	annualStatementAkontos    store.AnnualStatementPrepaymentStorage
	annualStatementReceipts   store.AnnualStatementReceiptStorage
	annualConsumption         store.AnnualStatementConsumptionStorage
	annualStatementRuns       store.AnnualStatementRunStorage
	annualStatementDeliveries *store.SQLAnnualStatementDeliveryStore
	unitStore                 unitStorage
	unitPaymentStore          unitPaymentStatusStorage
	issueStore                issueStorage
	attachmentStore           attachmentStorage
	contactStore              contactBookStorage
	auditStore                *auditStore
	documentStore             documentStorage
	handoverStore             handoverStorage
	protocolFiler             protocolFiler
	voteStore                 voteStorage
	voteReminderInterval      time.Duration
	parkingStore              *parkingStore
	parkingSampleInterval     time.Duration
	parkingHistoryStart       time.Time
	energyStore               energy.Storage
	homeReservations          store.HomeReservationStorage
	homePortals               store.HomePortalStorage
	homeConnectors            store.HomeConnectorStorage
	homeConnectorReadings     store.HomeConnectorReadingStorage
	homeConnectorDownloadDir  string
	energySampleInterval      time.Duration
	energySamplerMu           sync.Mutex
	energySamplers            map[string]*energySamplerState
	retentionFailure          atomic.Bool
	energyLifecycleLocks      sync.Map
	energyChartMu             sync.Mutex
	energyChartCache          map[string]energyChartCacheEntry
	mapTileMu                 sync.Mutex
	mapTileBaseURL            string
	geocoder                  addressGeocoder
	mapPreviewMu              sync.Mutex
	mapPreviewTiles           map[string]map[mapTileKey]time.Time

	annualStatementReceiptSuggester annualStatementReceiptSuggester

	// Organisation-level stores and the AI triage provider (HAUSV-593 slice).
	intake                 func(orgKey string) store.IntakeRepository
	orgSettings            func(orgKey string) store.OrgSettingsRepository
	organisationRepo       func(orgKey string) store.OrganisationRepository
	organisationMemberRepo func(orgKey string) store.OrganisationMemberRepository
	// Mail intake: one mailbox per organisation, polled in the background.
	mailIntakeConfigs map[string]mailintake.Config
	mailIntake        mailIntakeState
	intakeMailSeen    func(orgKey string) store.IntakeMailSeenRepository
	mailIntakeDir     string
	textbausteine     func(orgKey string) store.TextbausteinRepository
	triage            ai.TriageSuggester
	triageProvidersMu sync.Mutex
	triageProviders   map[string]triageProviderCacheEntry
	// Suggestion jobs are intentionally process-local. Production runs one app
	// replica, so cancellation and polling share this single in-memory table.
	inboxSuggestTimeout time.Duration
	suggestJobsMu       sync.Mutex
	suggestJobs         map[string]*suggestJob

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
	telegramStore             telegramStorage
	telegramPollTimeout       time.Duration
	chargingTelegramMu        sync.Mutex
	chargingTelegramTimes     []time.Time
	chargingTelegramThrottled bool

	paymentImportMu       sync.Mutex
	paymentImportPreviews map[string]camtImportPreview

	ebInterfaceImportMu       sync.Mutex
	ebInterfaceImportPreviews map[string]ebInterfaceImportPreview

	structuredExportMu       sync.Mutex
	structuredExportPreviews map[string]structuredExportPreview

	// tenantDB is the scoped seam: the object every SQL store asks for a database
	// handle, and the only way those stores can reach the database at all. The
	// app keeps it because stores are constructed from it at boot.
	tenantDB *store.TenantDB
	// scopedDB owns the lane pools tenantDB hands out. The app holds it purely to
	// close them on shutdown: TenantDB deliberately has no Close, because its
	// method set is what stops a store from reaching the database unscoped.
	scopedDB *db.Scoped
	// pool is what the app keeps of the process connection pool: its LIFECYCLE
	// and nothing else. It is typed as the two methods the app calls — PingContext
	// for the health probe, Close for shutdown — and not as *sql.DB, so no request
	// handler can run a statement on it. Four ledger statements used to hide here
	// as a.db.QueryRow / a.db.Exec: outside every lane, and under a fail-closed
	// policy they would have seen and written nothing. They now go through
	// tenantDB like every store, and this type is what stops the next one from
	// taking the same shortcut. Boot-time maintenance (EnsureTenantIdentities)
	// still takes the pool as a local in newApp; it never reaches the app
	// struct.
	pool processPool
}

// processPool is the lifecycle surface of the process connection pool: what
// app.pool may do with it, and all it may do with it.
type processPool interface {
	PingContext(context.Context) error
	Close() error
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
	MapSet            bool      `json:"map_set,omitempty"`
	MapLatitude       float64   `json:"map_latitude,omitempty"`
	MapLongitude      float64   `json:"map_longitude,omitempty"`
	MapZoom           int       `json:"map_zoom,omitempty"`
	BrandIcon         string    `json:"brand_icon,omitempty"`
	BrandAbbreviation string    `json:"brand_abbreviation,omitempty"`
	ContactName       string    `json:"contact_name,omitempty"`
	ContactAddress    string    `json:"contact_address,omitempty"`
	ContactEmail      string    `json:"contact_email,omitempty"`
	ContactPhone      string    `json:"contact_phone,omitempty"`
	EmergencyName     string    `json:"emergency_name,omitempty"`
	EmergencyPhone    string    `json:"emergency_phone,omitempty"`
	CaretakerName     string    `json:"caretaker_name,omitempty"`
	CaretakerEmail    string    `json:"caretaker_email,omitempty"`
	CaretakerPhone    string    `json:"caretaker_phone,omitempty"`
	HeroImage         string    `json:"hero_image,omitempty"`
	DisabledModules   []string  `json:"disabled_modules,omitempty"`
	UpdatedAt         time.Time `json:"updated_at,omitempty"`
}

const (
	tenantBrandCommunity    = "community"
	tenantBrandSingleHome   = "single-home"
	tenantBrandMultiTenant  = "multi-tenant"
	tenantBrandMixedUse     = "mixed-use"
	tenantBrandAddressPlate = "address-plaque"
	tenantBrandParking      = "parking"
	tenantBrandLucidePrefix = "lucide:"
)

func normalizeTenantBrandIcon(raw string) string {
	if preset := view.NormalizeTenantBrandIcon(raw); preset != "" {
		return preset
	}
	raw = strings.ToLower(strings.TrimSpace(raw))
	if !strings.HasPrefix(raw, tenantBrandLucidePrefix) {
		return ""
	}
	name := strings.TrimSpace(strings.TrimPrefix(raw, tenantBrandLucidePrefix))
	if !web.IsLucideIcon(name) {
		return ""
	}
	return tenantBrandLucidePrefix + name
}

func tenantBrandLucideName(icon string) string {
	icon = normalizeTenantBrandIcon(icon)
	if !strings.HasPrefix(icon, tenantBrandLucidePrefix) {
		return ""
	}
	return strings.TrimPrefix(icon, tenantBrandLucidePrefix)
}

func tenantBrandIconLabel(icon string) string {
	if name := tenantBrandLucideName(icon); name != "" {
		return name
	}
	return view.TenantBrandIconLabel(icon)
}

func tenantBrandLucideSVG(icon string) template.HTML {
	name := tenantBrandLucideName(icon)
	if name == "" {
		return ""
	}
	contents, err := web.Assets.ReadFile("assets/icons/lucide/" + name + ".svg")
	if err != nil {
		return ""
	}
	svg := string(contents)
	svg = strings.Replace(svg, "<svg", `<svg aria-hidden="true" focusable="false"`, 1)
	svg = strings.Replace(svg, `class="`, `class="hausv-mark tenant-brand-mark `, 1)
	return template.HTML(svg) // #nosec G203 -- only a whitelisted, vendored Lucide SVG can reach this branch.
}

// tenantBrandMarkSVG returns the brand mark SVG for templ portal (string, not template.HTML).
func tenantBrandMarkSVG(icon string) string {
	if lucideSVG := tenantBrandLucideSVG(icon); lucideSVG != "" {
		return string(lucideSVG)
	}
	// Fall back to built-in SVGs based on icon type
	icon = normalizeTenantBrandIcon(icon)
	switch icon {
	case "single-home":
		return `<svg class="hausv-mark tenant-brand-mark" viewBox="0 0 64 48" aria-hidden="true" focusable="false"><path d="M12 35h40"/><path d="M16 35V22.5L32 11l16 11.5V35"/><path d="M26.5 35v-9h11v9"/><path d="M21.5 27.5h5M37.5 27.5h5"/></svg>`
	case "multi-tenant":
		return `<svg class="hausv-mark tenant-brand-mark" viewBox="0 0 64 48" aria-hidden="true" focusable="false"><path d="M13 36h38"/><path d="M17 36V18h30v18"/><path d="M23 36v-7h6v7M35 36v-7h6v7"/><path d="M22 23h5M37 23h5M22 28h5M37 28h5"/><path d="M19 18l13-8 13 8"/></svg>`
	case "mixed-use":
		return `<svg class="hausv-mark tenant-brand-mark" viewBox="0 0 64 48" aria-hidden="true" focusable="false"><path d="M14 36h36"/><path d="M18 36V17h28v19"/><path d="M18 24h28"/><path d="M22 36v-7h8v7M35 36v-7h7v7"/><path d="M22 21h5M36 21h5"/><path d="M16 17l16-7 16 7"/></svg>`
	case "address-plaque":
		return `<svg class="hausv-mark tenant-brand-mark" viewBox="0 0 64 48" aria-hidden="true" focusable="false"><path d="M15 12h34a4 4 0 0 1 4 4v20a4 4 0 0 1-4 4H15a4 4 0 0 1-4-4V16a4 4 0 0 1 4-4z"/><path d="M20 21h24M20 28h18"/><path d="M46 28h.01"/></svg>`
	case "parking":
		return `<svg class="hausv-mark tenant-brand-mark" viewBox="0 0 64 48" aria-hidden="true" focusable="false"><path d="M17 36h30"/><path d="M20 36l2.5-12h19L44 36"/><path d="M23 36v4M41 36v4"/><path d="M24 29h16"/><path d="M28 20h8a5 5 0 0 1 0 10h-8V16"/></svg>`
	default:
		// Default community icon
		return `<svg class="hausv-mark tenant-brand-mark" viewBox="0 0 64 48" aria-hidden="true" focusable="false"><path d="M13 34h38"/><path d="M14.5 34v-9l7-5.5 7 5.5v9"/><path d="M35.5 34v-9l7-5.5 7 5.5v9"/><path d="M25 34V20.5L32 15l7 5.5V34"/><path d="M29 34v-7h6v7"/><path d="M18 28h3.5M42.5 28H46"/></svg>`
	}
}

// routes builds the application's ServeMux. Extracted from main() so that
// tests exercise the real route patterns instead of calling handler methods
// directly — a test that fakes r.SetPathValue cannot catch a wrong pattern.
func (a *app) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /assets/", http.FileServerFS(web.Assets))
	mux.HandleFunc("GET /favicon.svg", favicon)
	mux.HandleFunc("GET /favicon.ico", favicon)
	mux.HandleFunc("GET /tenant-hero/{tenant}", a.tenantHeroImage)
	mux.HandleFunc("GET /map-tiles/{z}/{x}/{tile}", a.mapTile)
	mux.HandleFunc("GET /healthz", a.health)
	if a.templExampleEnabled {
		mux.HandleFunc("GET /_templ/example", a.templExample)
	}
	mux.HandleFunc("GET /datenschutz", a.privacyNotice)
	mux.HandleFunc("GET /impressum", a.imprintPage)
	mux.HandleFunc("GET /start", a.homeStartPage)
	mux.HandleFunc("POST /start", a.requestHomeStart)
	mux.HandleFunc("GET /start/verify", a.verifyHomeStart)
	mux.HandleFunc("GET /start/connector", a.homeConnectorStart)
	mux.HandleFunc("POST /start/activate", a.activateHomePortal)
	mux.HandleFunc("POST /start/connector/pairing", a.startHomeConnectorPairing)
	mux.HandleFunc("POST /start/connector/revoke", a.revokeHomeConnector)
	mux.HandleFunc("POST /api/home-connectors/pair", a.pairHomeConnector)
	mux.HandleFunc("POST /api/home-connectors/heartbeat", a.heartbeatHomeConnector)
	mux.HandleFunc("GET /downloads/{filename}", a.downloadHomeConnector)
	mux.HandleFunc("GET /", a.home)
	mux.HandleFunc("POST /auth/request", a.requestLogin)
	mux.HandleFunc("GET /auth/verify", a.publicPage(a.verifyLogin))
	mux.HandleFunc("GET /auth/oidc/start", a.publicPage(a.startOIDCLogin))
	mux.HandleFunc("GET /auth/oidc/callback", a.publicPage(a.finishOIDCLogin))
	mux.HandleFunc("POST /auth/logout", a.logout)
	mux.HandleFunc("GET /calendar/{token}", a.calendarFeed)
	mux.HandleFunc("GET /app", a.page(a.portal))
	mux.HandleFunc("POST /app/context", a.action(a.switchPortalContext))
	mux.HandleFunc("GET /app/verwaltung", a.page(a.requireVerwaltung(a.portfolioPage)))
	mux.HandleFunc("GET /app/verwaltung/posteingang", a.page(a.requireVerwaltung(a.inboxPage)))
	mux.HandleFunc("GET /app/verwaltung/posteingang/{id}", a.page(a.requireVerwaltung(a.inboxCasePage)))
	mux.HandleFunc("GET /app/verwaltung/posteingang/{id}/vorschlag", a.page(a.requireVerwaltung(a.inboxSuggestionPartial)))
	mux.HandleFunc("POST /app/verwaltung/posteingang/{id}", a.action(a.requireVerwaltung(a.inboxCaseAction)))
	mux.HandleFunc("POST /app/verwaltung/telefonnotiz", a.action(a.requireVerwaltung(a.phoneNoteAction)))
	mux.HandleFunc("GET /app/verwaltung/einstellungen", a.page(a.requireVerwaltung(a.verwaltungSettingsPage)))
	mux.HandleFunc("POST /app/verwaltung/einstellungen", a.action(a.requireVerwaltung(a.verwaltungSettingsAction)))
	mux.HandleFunc("GET /app/verwaltung/rechte", a.page(a.requireVerwaltung(a.rechtePage)))
	mux.HandleFunc("GET /app/verwaltung/einstellungen/demo", a.page(a.requireVerwaltung(a.verwaltungDemoResetPage)))
	mux.HandleFunc("POST /app/verwaltung/einstellungen/demo", a.action(a.requireVerwaltung(a.verwaltungDemoResetAction)))
	mux.HandleFunc("POST /app/verwaltung/einstellungen/ki-test", a.action(a.requireVerwaltung(a.verwaltungAITestAction)))
	mux.HandleFunc("POST /app/verwaltung/einstellungen/mitarbeiter", a.action(a.requireVerwaltung(a.verwaltungMemberAdd)))
	mux.HandleFunc("POST /app/verwaltung/einstellungen/mitarbeiter/entfernen", a.action(a.requireVerwaltung(a.verwaltungMemberRemove)))
	mux.HandleFunc("GET /app/verwaltung/textbausteine", a.page(a.requireVerwaltung(a.textbausteinListPage)))
	mux.HandleFunc("GET /app/verwaltung/textbausteine/{key}", a.page(a.requireVerwaltung(a.textbausteinFormPage)))
	mux.HandleFunc("POST /app/verwaltung/textbausteine/{key}", a.action(a.requireVerwaltung(a.textbausteinSaveAction)))
	mux.HandleFunc("POST /app/verwaltung/textbausteine/{key}/deaktivieren", a.action(a.requireVerwaltung(a.textbausteinDeactivateAction)))
	mux.HandleFunc("POST /app/verwaltung/textbausteine/{key}/aktivieren", a.action(a.requireVerwaltung(a.textbausteinActivateAction)))
	mux.HandleFunc("POST /app/ansicht/start", a.action(a.rolePreviewStart))
	mux.HandleFunc("POST /app/ansicht/ende", a.action(a.rolePreviewEnd))
	mux.HandleFunc("GET /app/hilfe", a.page(a.helpPage))
	mux.HandleFunc("POST /app/hilfe/connector/pairing", a.action(a.startAppHomeConnectorPairing))
	mux.HandleFunc("POST /app/hilfe/connector/revoke", a.action(a.revokeAppHomeConnector))
	mux.HandleFunc("GET /app/zuhause/onboarding", a.page(a.withEnergyLifecycleOperation(a.homeOnboarding)))
	mux.HandleFunc("POST /app/zuhause/onboarding", a.action(a.withEnergyLifecycleOperation(a.updateHomeOnboarding)))
	mux.HandleFunc("GET /app/energie", a.page(a.withEnergyLifecycleOperation(a.energyCockpit)))
	mux.HandleFunc("GET /app/energie/live", a.page(a.withEnergyLifecycleOperation(a.energyLiveRefresh)))
	mux.HandleFunc("POST /app/energie/mode", a.action(a.withClaimedEnergyLifecycleOperation(a.updateEnergyMode)))
	mux.HandleFunc("POST /app/energie/mappings", a.action(a.withClaimedEnergyLifecycleOperation(a.updateEnergyMappings)))
	mux.HandleFunc("POST /app/energie/assets", a.action(a.withClaimedEnergyLifecycleOperation(a.updateEnergyAssets)))
	mux.HandleFunc("POST /app/energie/smart-meter", a.action(a.withClaimedEnergyLifecycleOperation(a.importSmartMeter)))
	mux.HandleFunc("POST /app/energie/target", a.action(a.withClaimedEnergyLifecycleOperation(a.updateEnergyTarget)))
	mux.HandleFunc("POST /app/energie/anschlussleistung", a.action(a.withClaimedEnergyLifecycleOperation(a.updateEnergyAgreedPower)))
	mux.HandleFunc("POST /app/energie/verbraucher", a.action(a.withClaimedEnergyLifecycleOperation(a.addEnergyConsumer)))
	mux.HandleFunc("GET /app/energie/verbraucher/messwerte", a.page(a.withEnergyLifecycleOperation(a.energyConsumerMeasurementOptions)))
	mux.HandleFunc("POST /app/energie/verbraucher/entfernen", a.action(a.withClaimedEnergyLifecycleOperation(a.deleteEnergyConsumer)))
	mux.HandleFunc("POST /app/energie/verbraucher/reihenfolge", a.action(a.withClaimedEnergyLifecycleOperation(a.reorderEnergyConsumers)))
	mux.HandleFunc("POST /app/energie/recommendation", a.action(a.withClaimedEnergyLifecycleOperation(a.updateEnergyRecommendation)))
	mux.HandleFunc("POST /app/energie/measure", a.action(a.withClaimedEnergyLifecycleOperation(a.createEnergyMeasure)))
	mux.HandleFunc("POST /app/energie/measure/update", a.action(a.withClaimedEnergyLifecycleOperation(a.updateEnergyMeasure)))
	mux.HandleFunc("POST /app/energie/caretaker", a.action(a.withClaimedEnergyLifecycleOperation(a.updateEnergyCaretaker)))
	mux.HandleFunc("POST /app/energie/caretaker/invite", a.action(a.withClaimedEnergyLifecycleOperation(a.inviteEnergyCaretaker)))
	mux.HandleFunc("POST /app/energie/maintenance", a.action(a.withClaimedEnergyLifecycleOperation(a.upsertEnergyMaintenance)))
	mux.HandleFunc("POST /app/energie/maintenance/complete", a.action(a.withClaimedEnergyLifecycleOperation(a.completeEnergyMaintenance)))
	mux.HandleFunc("POST /app/energie/tariff/assessment", a.action(a.withClaimedEnergyLifecycleOperation(a.saveEnergyTariffAssessment)))
	mux.HandleFunc("GET /app/settings/energy-data", a.page(a.withEnergyLifecycleOperation(a.energyDataPage)))
	mux.HandleFunc("POST /app/settings/energy-data/export", a.action(a.withEnergyLifecycleOperation(a.exportEnergyData)))
	mux.HandleFunc("POST /app/settings/energy-data/history/delete", a.action(a.withEnergyLifecycleReset(a.deleteEnergyMeasurementData)))
	mux.HandleFunc("POST /app/settings/energy-data/profile/delete", a.action(a.withEnergyLifecycleReset(a.deleteEnergyProfile)))
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
	mux.HandleFunc("GET /app/dokumente/rechnungen/import", a.authed(capabilityManageDocuments, a.ebInterfaceImportPage))
	mux.HandleFunc("POST /app/dokumente/rechnungen/import/preview", a.authedAction(capabilityManageDocuments, a.previewEBInterfaceImport))
	mux.HandleFunc("POST /app/dokumente/rechnungen/import/store", a.authedAction(capabilityManageDocuments, a.storeEBInterfaceImport))
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
	mux.HandleFunc("POST /app/uebergaben/attachments", a.action(a.addHandoverAttachments))
	mux.HandleFunc("POST /app/uebergaben/file", a.action(a.fileHandoverProtocol))
	mux.HandleFunc("GET /app/uebergaben/{id}/protokoll", a.page(a.handoverProtocol))
	mux.HandleFunc("GET /handover/{token}", a.publicPage(a.handoverConfirmPage))
	mux.HandleFunc("GET /handover/{token}/attachments/{id}", a.publicPage(a.handoverAttachment))
	mux.HandleFunc("GET /handover/{token}/attachments/{id}/{variant}", a.publicPage(a.handoverAttachment))
	mux.HandleFunc("POST /handover/{token}", a.confirmHandover)
	mux.HandleFunc("GET /app/kontakte", a.page(a.contacts))
	mux.HandleFunc("POST /app/kontakte", a.action(a.upsertManagedContact))
	mux.HandleFunc("POST /app/kontakte/delete", a.action(a.deactivateManagedContact))
	mux.HandleFunc("GET /app/anliegen", a.page(a.issues))
	mux.HandleFunc("GET /app/anliegen/board", a.page(a.issueBoard))
	mux.HandleFunc("GET /app/anliegen/board/{id}", a.page(a.issueTriage))
	mux.HandleFunc("GET /app/anliegen/{id}", a.page(a.issueResidentDetail))
	mux.HandleFunc("POST /app/anliegen", a.action(a.createIssue))
	mux.HandleFunc("POST /app/anliegen/comment", a.action(a.addIssueComment))
	mux.HandleFunc("POST /app/anliegen/comment/delete", a.action(a.deleteIssueComment))
	mux.HandleFunc("POST /app/anliegen/resolution", a.action(a.confirmIssueResolution))
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
	mux.HandleFunc("GET /app/settings/annual-statement", a.page(a.annualStatementPage))
	mux.HandleFunc("POST /app/settings/annual-statement/cost-types", a.action(a.saveAnnualStatementCostType))
	mux.HandleFunc("POST /app/settings/annual-statement/periods", a.action(a.saveAnnualStatementPeriod))
	mux.HandleFunc("POST /app/settings/annual-statement/periods/next", a.action(a.cloneNextAnnualStatementPeriod))
	mux.HandleFunc("POST /app/settings/annual-statement/parties/import", a.action(a.importAnnualStatementParties))
	mux.HandleFunc("POST /app/settings/annual-statement/allocation-bases", a.action(a.saveAnnualStatementAllocationBases))
	mux.HandleFunc("POST /app/settings/annual-statement/runs", a.action(a.createAnnualStatementRun))
	mux.HandleFunc("POST /app/settings/annual-statement/runs/{runID}/archive", a.action(a.archiveAnnualStatementRun))
	mux.HandleFunc("POST /app/settings/annual-statement/runs/{runID}/send", a.action(a.sendAnnualStatementRun))
	mux.HandleFunc("GET /app/settings/annual-statement/runs/{runID}/pdf", a.page(a.downloadAnnualStatementPDF))
	mux.HandleFunc("POST /app/settings/annual-statement/prepayments", a.action(a.saveAnnualStatementPrepayment))
	mux.HandleFunc("POST /app/settings/annual-statement/receipts/suggest", a.action(a.suggestAnnualStatementReceipt))
	mux.HandleFunc("POST /app/settings/annual-statement/receipts/confirm", a.action(a.confirmAnnualStatementReceiptSuggestion))
	mux.HandleFunc("POST /app/settings/annual-statement/receipts", a.action(a.createAnnualStatementReceipt))
	mux.HandleFunc("POST /app/settings/annual-statement/receipts/update-amount", a.action(a.updateAnnualStatementReceiptAmount))
	mux.HandleFunc("POST /app/settings/annual-statement/receipts/delete", a.action(a.deleteAnnualStatementReceipt))
	mux.HandleFunc("GET /app/settings/modules", a.authed(capabilityManageBuilding, a.portalModuleSettings))
	mux.HandleFunc("POST /app/settings/modules", a.authedAction(capabilityManageBuilding, a.updatePortalModules))
	mux.HandleFunc("GET /app/settings/home", a.page(a.withEnergyLifecycleOperation(a.homeIdentitySettings)))
	mux.HandleFunc("POST /app/settings/home", a.action(a.withEnergyLifecycleOperation(a.updateHomeIdentity)))
	mux.HandleFunc("GET /app/settings/building", a.page(a.buildingSettings))
	mux.HandleFunc("POST /app/settings/building", a.action(a.updateBuildingSettings))
	mux.HandleFunc("POST /app/settings/building/contacts", a.action(a.updateBuildingContacts))
	mux.HandleFunc("POST /app/settings/building/appearance", a.action(a.updateBuildingAppearance))
	mux.HandleFunc("POST /app/settings/building/geocode", a.authedAction(capabilityManageBuilding, a.geocodeBuildingAddress))
	mux.HandleFunc("POST /app/settings/building/hero", a.action(a.updateBuildingHero))
	mux.HandleFunc("POST /app/settings/building/hero/delete", a.action(a.deleteBuildingHero))
	mux.HandleFunc("POST /app/settings/building/units", a.action(a.upsertBuildingUnit))
	mux.HandleFunc("POST /app/settings/building/units/delete", a.action(a.deleteBuildingUnit))
	mux.HandleFunc("POST /app/settings/building/payment-status", a.action(a.updateUnitPaymentStatus))
	mux.HandleFunc("GET /app/settings/payments/import", a.page(a.paymentImportPage))
	mux.HandleFunc("POST /app/settings/payments/import/preview", a.action(a.previewPaymentImport))
	mux.HandleFunc("POST /app/settings/payments/import/apply", a.action(a.applyPaymentImport))
	mux.HandleFunc("GET /app/settings/data-export", a.authed(capabilityManageBuilding, a.structuredExportPage))
	mux.HandleFunc("POST /app/settings/data-export/preview", a.authedAction(capabilityManageBuilding, a.previewStructuredExport))
	mux.HandleFunc("POST /app/settings/data-export/download", a.authedAction(capabilityManageBuilding, a.downloadStructuredExport))
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
	// Catch-all for every unmatched GET path, so a mistyped or stale URL ends on
	// the branded 404 instead of the standard library's plain-text one.
	mux.HandleFunc("GET /{tenant}", a.publicPage(a.tenantPathRedirect))
	mux.HandleFunc("GET /{tenant}/{rest...}", a.publicPage(a.tenantPathRedirect))
	return mux
}

// handler is the fully wrapped HTTP handler, middleware included. This is
// what main() serves and what the tests drive.
func (a *app) handler() http.Handler {
	// recoverAndLog is outermost so it captures panics and the final status from
	// every inner layer, including securityHeaders (HAUSV-141).
	return a.recoverAndLog(a.securityHeaders(a.canonicalHost(a.tenantPaths(a.routes()))))
}

type tenantPathContextKey struct{}

// requestRepositories contains only repositories bound to the tenant resolved
// for this request. It is deliberately unexported and can only be constructed
// together with a resolvedTenantRequest by tenantPaths.
type requestRepositories struct {
	annualStatementCostTypes  store.AnnualStatementCostTypeRepository
	annualStatementPeriods    store.AnnualStatementPeriodRepository
	annualStatementAkontos    store.AnnualStatementPrepaymentRepository
	annualStatementReceipts   store.AnnualStatementReceiptRepository
	annualConsumption         store.AnnualStatementConsumptionRepository
	annualStatementRuns       store.AnnualStatementRunRepository
	annualStatementDeliveries store.AnnualStatementDeliveryRepository
	announcementReads         store.AnnouncementReadRepository
	announcements             store.AnnouncementRepository
	attachments               store.AttachmentRepository
	contacts                  store.ContactBookRepository
	documents                 store.DocumentRepository
	events                    store.EventRepository
	handovers                 store.HandoverRepository
	identity                  store.IdentityRepository
	issues                    store.IssueRepository
	unitPayments              store.UnitPaymentStatusRepository
	units                     store.UnitRepository
	votes                     store.VoteRepository
}

type resolvedTenantRequest struct {
	tenant       tenantConfig
	tenantRef    store.TenantRef
	repositories requestRepositories
	pathPrefixed bool
}

func (a *app) repositoriesForTenant(tenant store.TenantRef) requestRepositories {
	repositories := requestRepositories{}
	if a == nil {
		return repositories
	}
	if a.annualStatementCostTypes != nil {
		repositories.annualStatementCostTypes, _ = store.BindAnnualStatementCostTypeRepository(a.annualStatementCostTypes, tenant)
	}
	if a.annualStatementPeriods != nil {
		repositories.annualStatementPeriods, _ = store.BindAnnualStatementPeriodRepository(a.annualStatementPeriods, tenant)
	}
	if a.annualStatementAkontos != nil {
		repositories.annualStatementAkontos, _ = store.BindAnnualStatementPrepaymentRepository(a.annualStatementAkontos, tenant)
	}
	if a.annualStatementDeliveries != nil {
		repositories.annualStatementDeliveries, _ = store.BindAnnualStatementDeliveryRepository(a.annualStatementDeliveries, tenant)
	}
	if a.annualStatementRuns != nil {
		repositories.annualStatementRuns, _ = store.BindAnnualStatementRunRepository(a.annualStatementRuns, tenant)
	}
	if a.annualConsumption != nil {
		repositories.annualConsumption, _ = store.BindAnnualStatementConsumptionRepository(a.annualConsumption, tenant)
	}
	if a.annualStatementReceipts != nil {
		repositories.annualStatementReceipts, _ = store.BindAnnualStatementReceiptRepository(a.annualStatementReceipts, tenant)
	}
	if a.announcementReadStore != nil {
		repositories.announcementReads, _ = store.BindAnnouncementReadRepository(a.announcementReadStore, tenant)
	}
	if a.announcementStore != nil {
		repositories.announcements, _ = store.BindAnnouncementRepository(a.announcementStore, tenant)
	}
	if a.attachmentStore != nil {
		repositories.attachments, _ = store.BindAttachmentRepository(a.attachmentStore, tenant)
	}
	if a.contactStore != nil {
		repositories.contacts, _ = store.BindContactBookRepository(a.contactStore, tenant)
	}
	if a.documentStore != nil {
		repositories.documents, _ = store.BindDocumentRepository(a.documentStore, tenant)
	}
	if a.eventStore != nil {
		repositories.events, _ = store.BindEventRepository(a.eventStore, tenant)
	}
	if a.handoverStore != nil {
		repositories.handovers, _ = store.BindHandoverRepository(a.handoverStore, tenant)
	}
	if a.identityStore != nil {
		repositories.identity, _ = store.BindIdentityRepository(a.identityStore, tenant)
	}
	if a.issueStore != nil {
		repositories.issues, _ = store.BindIssueRepository(a.issueStore, tenant)
	}
	if a.unitPaymentStore != nil {
		repositories.unitPayments, _ = store.BindUnitPaymentStatusRepository(a.unitPaymentStore, tenant)
	}
	if a.unitStore != nil {
		repositories.units, _ = store.BindUnitRepository(a.unitStore, tenant)
	}
	if a.voteStore != nil {
		repositories.votes, _ = store.BindVoteRepository(a.voteStore, tenant)
	}
	return repositories
}

func (a *app) tenantIdentity(slug string) (store.TenantIdentity, bool) {
	if a == nil {
		return store.TenantIdentity{}, false
	}
	slug = normalizeSlug(slug)
	identity, ok := a.tenantIdentities[slug]
	if ok && identity.Ref().Valid() {
		return identity, true
	}
	if a.homePortals == nil {
		return store.TenantIdentity{}, false
	}
	portal, found, err := a.homePortals.Get(slug)
	if err != nil || !found {
		return store.TenantIdentity{}, false
	}
	identity = store.TenantIdentity{ID: portal.TenantID, Slug: portal.Slug, Name: portal.HouseholdName}
	return identity, identity.Ref().Valid()

}

func (a *app) resolveTenantRequest(tenant tenantConfig, pathPrefixed bool) (resolvedTenantRequest, bool) {
	identity, ok := a.tenantIdentity(tenant.Slug)
	if !ok {
		return resolvedTenantRequest{}, false
	}
	tenantRef := identity.Ref()
	return resolvedTenantRequest{
		tenant:       tenant,
		tenantRef:    tenantRef,
		repositories: a.repositoriesForTenant(tenantRef),
		pathPrefixed: pathPrefixed,
	}, true
}

func resolvedTenantFromContext(ctx context.Context) (resolvedTenantRequest, bool) {
	if ctx == nil {
		return resolvedTenantRequest{}, false
	}
	resolved, ok := ctx.Value(tenantPathContextKey{}).(resolvedTenantRequest)
	return resolved, ok && resolved.tenant.Slug != ""
}

// tenantPaths resolves /<tenant>/... before the standard mux sees the request
// and keeps redirects inside the same tenant prefix.
func (a *app) tenantPaths(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		first, rest, _ := strings.Cut(path, "/")
		tenant, ok := a.tenantBySlug(first)
		if !ok {
			// Unprefixed routes keep their established default-tenant behaviour,
			// but the resolution still happens here rather than in handlers.
			tenant, ok = a.tenantBySlug(a.defaultTenant)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			resolved, ok := a.resolveTenantRequest(tenant, false)
			if !ok {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			ctx := context.WithValue(r.Context(), tenantPathContextKey{}, resolved)
			*r = *r.WithContext(ctx)
			next.ServeHTTP(w, r)
			return
		}

		resolved, ok := a.resolveTenantRequest(tenant, true)
		if !ok {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		ctx := context.WithValue(r.Context(), tenantPathContextKey{}, resolved)
		*r = *r.WithContext(ctx)
		cloned := r.Clone(ctx)
		cloned.URL.Path = "/" + rest
		if rest == "" {
			cloned.URL.Path = "/"
		}
		writer := &tenantLocationWriter{ResponseWriter: w, prefix: "/" + tenant.Slug}
		next.ServeHTTP(writer, cloned)
		r.Pattern = cloned.Pattern
	})
}

type tenantLocationWriter struct {
	http.ResponseWriter
	prefix string
}

func (w *tenantLocationWriter) WriteHeader(status int) {
	location := w.Header().Get("Location")
	if strings.HasPrefix(location, "/") && !strings.HasPrefix(location, "//") &&
		!strings.HasPrefix(location, w.prefix+"/") && location != w.prefix {
		w.Header().Set("Location", w.prefix+location)
	}
	w.ResponseWriter.WriteHeader(status)
}

// canonicalHost permanently redirects the www alias to the bare root domain so
// the public site has exactly one canonical address (HAUSV-435). The target is
// always https: the app is only publicly reachable behind TLS.
func (a *app) canonicalHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		root := normalizeHost(a.rootDomain)
		if root != "" && normalizeHost(r.Host) == "www."+root {
			http.Redirect(w, r, "https://"+root+r.URL.RequestURI(), http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
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
	trustedProxies, err := parseTrustedProxyAllowlist(env("TRUSTED_PROXY_CIDRS", ""), publicURL)
	if err != nil {
		return nil, err
	}
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
	defaultTenant := env("DEFAULT_TENANT", "demo")
	tenants, err := parseTenants(env("WEG_TENANTS_JSON", ""), defaultTenant, newHomeAssistantConfig())
	if err != nil {
		return nil, err
	}
	organisations, err := parseOrganisations(env("WEG_ORGANISATIONS_JSON", ""))
	if err != nil {
		return nil, err
	}
	if err := config.ApplyHomeAssistantConnectors(env("HA_CONNECTORS_JSON", ""), tenants); err != nil {
		return nil, err
	}
	profiles, err := parseUserProfiles(env("WEG_USERS_JSON", ""), allowed, admins, defaultTenant)
	if err != nil {
		return nil, err
	}
	localDevLogin := parseBool(env("LOCAL_DEV_LOGIN", "false")) && isLocalHost(parsed.Hostname())
	// Demo login (HAUSV-609): a fixture-only instance on a public host may render the
	// magic link inline when the visitor knows the shared access code. Never enable
	// this on an instance that holds real data.
	demoLoginCode := strings.TrimSpace(env("DEMO_LOGIN_ACCESS_CODE", ""))
	demoLoginEnabled := parseBool(env("DEMO_LOGIN_ENABLED", "false"))
	demoLogin := demoLoginEnabled && demoLoginCode != ""
	if demoLoginEnabled && localDevLogin {
		logInfo("demo login takes precedence over LOCAL_DEV_LOGIN")
		localDevLogin = false
	}
	if demoLogin {
		logInfo("demo login enabled: fixture-only instance expected", "host", parsed.Hostname())
	}

	smtpHost := env("SMTP_HOST", "")
	mailOutboxDir := strings.TrimSpace(env("MAIL_OUTBOX_DIR", ""))
	mailTransport := appmail.NewSMTP(
		smtpHost,
		env("SMTP_PORT", "587"),
		env("SMTP_USER", ""),
		env("SMTP_PASS", ""),
		env("MAIL_FROM", "hausv.org <noreply@example.invalid>"),
	).WithOutbox(mailOutboxDir)
	if err := mailTransport.Validate(); err != nil {
		return nil, err
	}
	switch {
	case mailOutboxDir != "":
		logInfo("mail: outbox mode " + mailOutboxDir)
	case mailTransport.Configured():
		logInfo("mail: smtp " + smtpHost)
	default:
		logInfo("mail: not configured")
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
	// The demo login (HAUSV-609) is the third public login path: fixture-only
	// instances behind an access code, never real data.
	if publicURL && !mailTransport.Delivers() && !oidcLogin.Configured() && !demoLogin {
		return nil, fmt.Errorf("SMTP, OIDC or demo login is required when BASE_URL is public")
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
	tenantHeroSeedDir := env("TENANT_HERO_SEED_DIR", "")
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
	issueAttachmentDir := env("ISSUE_ATTACHMENT_DIR", defaultIssueAttachmentDir)
	issues, err := newIssueStore(issueDataPath, issueAttachmentDir)
	if err != nil {
		return nil, err
	}
	attachmentDataPath := env("ATTACHMENT_DATA_PATH", "tmp/attachments.json")
	defaultAttachmentFileDir := filepath.Join(filepath.Dir(attachmentDataPath), "attachments")
	attachmentFileDir := env("ATTACHMENT_FILE_DIR", defaultAttachmentFileDir)
	attachments, err := newAttachmentStore(attachmentDataPath, attachmentFileDir)
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
	// 30 s ergeben 30 Abtastungen je Viertelstunde. Dichter zu messen belastet
	// Home Assistant ohne erkennbaren Gewinn, dünner zu messen macht jede
	// ausgefallene Abtastung zu einem sichtbaren Loch in der Abdeckung.
	energySampleInterval, err := parseDuration(env("ENERGY_SAMPLE_INTERVAL", "30s"))
	if err != nil {
		return nil, fmt.Errorf("invalid ENERGY_SAMPLE_INTERVAL")
	}
	parkingHistoryStart, err := parseHistoryStart(env("PARKING_HISTORY_START", ""), time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid PARKING_HISTORY_START")
	}
	sessionTTL, err := parseDuration(env("SESSION_TTL", "720h"))
	if err != nil || sessionTTL <= 0 {
		return nil, fmt.Errorf("invalid SESSION_TTL")
	}
	inboxSuggestTimeout, err := parseDuration(env("AI_TIMEOUT", "45s"))
	if err != nil || inboxSuggestTimeout <= 0 {
		return nil, fmt.Errorf("invalid AI_TIMEOUT")
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
	// SQLite is REQUIRED (HAUSV-173). The JSON fallback is gone on purpose: a
	// database problem must fail the boot loudly instead of silently serving a
	// stale second copy of the data. The JSON files stay on disk as frozen
	// pre-cutover backups, and rolling back is one image tag away.
	dbPath := env("DB_PATH", "")
	if dbPath == "" {
		dbPath = filepath.Join(filepath.Dir(parkingDataPath), "hausv.db")
	}
	// Backend selection lives here and nowhere else: stores take a lane from the
	// scoped seam and do not know which engine served it. DB_BACKEND unset keeps
	// SQLite, so an existing deployment is unchanged by this being wired at all.
	backend := db.Backend(strings.ToLower(strings.TrimSpace(env("DB_BACKEND", ""))))
	dbDSN := dbPath
	if backend == db.BackendPostgres {
		dbDSN = strings.TrimSpace(env("DATABASE_URL", ""))
		if dbDSN == "" {
			return nil, fmt.Errorf("DB_BACKEND=postgres requires DATABASE_URL")
		}
	}
	// The lane plan is stated before anything opens: how many tenant-pinned
	// pools may exist at once, and how wide each one is. Both are knobs because
	// the number that fits depends on the server, and a plan that does not fit
	// has to stop the boot rather than turn into connection refusals under load.
	laneCap, err := strconv.Atoi(env("DB_LANE_CAP", "24"))
	if err != nil || laneCap < 1 {
		return nil, fmt.Errorf("invalid DB_LANE_CAP")
	}
	laneMaxConns, err := strconv.Atoi(env("DB_LANE_MAX_CONNS", "3"))
	if err != nil || laneMaxConns < 1 {
		return nil, fmt.Errorf("invalid DB_LANE_MAX_CONNS")
	}
	dbConfig := db.Config{Backend: backend, DSN: dbDSN, LaneCap: laneCap, LaneMaxConns: laneMaxConns}
	database, err := db.OpenConfig(context.Background(), dbConfig)
	if err != nil {
		if backend == db.BackendPostgres {
			// Never echo the DSN: it carries the role password.
			return nil, fmt.Errorf("open postgres: %w", err)
		}
		return nil, fmt.Errorf("open sqlite at %s: %w", dbPath, err)
	}
	// The scoped seam. Every SQL store below takes this instead of the process
	// pool, so each of their statements names a tenant lane or a declared
	// cross-tenant one. VerifyBudget runs before any store exists, so a lane plan
	// the server cannot serve stops the boot rather than turning into connection
	// refusals under load.
	scoped, err := db.NewScoped(dbConfig, database)
	if err != nil {
		_ = database.Close()
		return nil, err
	}
	if err := scoped.VerifyBudget(context.Background()); err != nil {
		_ = database.Close()
		return nil, err
	}
	tenantDB := store.NewTenantDB(scoped)
	configuredIdentities := make([]store.TenantIdentity, 0, len(tenants))
	for _, tenant := range tenants {
		configuredIdentities = append(configuredIdentities, store.TenantIdentity{Slug: tenant.Slug, Name: tenant.Name})
	}
	tenantIdentities, err := store.EnsureTenantIdentities(context.Background(), database, configuredIdentities)
	if err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("ensure tenant identities: %w", err)
	}

	// Every migrated store is served from SQLite. Imports stay in place because
	// they are idempotent and clobber-safe: on an already-migrated database they
	// are a no-op, and they are what migrates a fresh environment. A failing
	// import aborts the boot rather than quietly starting on an empty store.
	sqlActivity := newSQLActivityStore(tenantDB)
	sqlAnnualStatementCostTypes := store.NewSQLAnnualStatementCostTypeStore(tenantDB)
	sqlAnnualStatementPeriods := store.NewSQLAnnualStatementPeriodStore(tenantDB)
	sqlAnnualStatementPrepayments := store.NewSQLAnnualStatementPrepaymentStore(tenantDB)
	sqlAnnualStatementReceipts := store.NewSQLAnnualStatementReceiptStore(tenantDB)
	sqlProfileOverlay := newSQLProfileOverlayStore(tenantDB)
	sqlNotification := newSQLNotificationPrefStore(tenantDB)
	sqlUnitPayment := newSQLUnitPaymentStatusStore(tenantDB)
	sqlContacts := newSQLContactBookStore(tenantDB)
	sqlAnnRead := newSQLAnnouncementReadStore(tenantDB)
	sqlAnn := newSQLAnnouncementStore(tenantDB)
	sqlEvent := newSQLEventStore(tenantDB)
	sqlHandover := newSQLHandoverStore(tenantDB)
	sqlDocument := newSQLDocumentStore(tenantDB, documentFileDir)
	sqlAttachment := newSQLAttachmentStore(tenantDB, attachmentFileDir)
	sqlTelegram := newSQLTelegramStore(tenantDB)
	sqlUnits := newSQLUnitStore(tenantDB)
	sqlVotes := newSQLVoteStore(tenantDB)
	sqlIssues := newSQLIssueStore(tenantDB, issueAttachmentDir)
	identity := newSQLIdentityStore(tenantDB)
	energyBackend := energy.NewSQLStore(tenantDB)
	homeReservationBackend := store.NewSQLHomeReservationStore(tenantDB)
	homePortalBackend := store.NewSQLHomePortalStore(tenantDB)
	homeConnectorBackend := store.NewSQLHomeConnectorStore(tenantDB)
	homeConnectorReadingBackend := store.NewSQLHomeConnectorReadingStore(tenantDB)
	if removed, err := homeReservationBackend.PurgePendingBefore(time.Now().Add(-homePendingRetention)); err != nil {
		return nil, fmt.Errorf("expired home reservation purge failed: %w", err)
	} else if removed > 0 {
		logInfo("expired home reservations purged", "count", removed)
	}
	knownEnergyTenants := make(map[string]struct{}, len(tenants))
	for slug := range tenants {
		knownEnergyTenants[slug] = struct{}{}
	}
	if err := energy.ApplyProfileSeeds(energyBackend, env("HOME_PROFILE_SEEDS_JSON", ""), knownEnergyTenants, time.Now()); err != nil {
		return nil, err
	}
	rawBefore, intervalBefore, assessmentBefore := energyRetentionCutoffs(time.Now())
	if summary, err := energyBackend.PurgeExpired(rawBefore, intervalBefore, assessmentBefore); err != nil {
		return nil, fmt.Errorf("expired energy data purge failed: %w", err)
	} else if summary.Imports+summary.Intervals+summary.TariffAssessments > 0 {
		logInfo("expired energy data purged",
			"raw_imports", summary.Imports,
			"intervals", summary.Intervals,
			"assessments", summary.TariffAssessments,
		)
	}

	for _, step := range []struct {
		name    string
		import_ func() error
	}{
		{"activity", func() error { return sqlActivity.ImportActivity(activity) }},
		{"profile-overlay", func() error { return sqlProfileOverlay.ImportOverlays(profileOverlays) }},
		{"notification-pref", func() error { return sqlNotification.ImportPrefs(notificationPrefs) }},
		{"unit-payment-status", func() error { return sqlUnitPayment.ImportStatuses(unitPayments) }},
		{"contact", func() error { return sqlContacts.ImportContacts(contacts) }},
		{"announcement-read", func() error { return sqlAnnRead.ImportReads(announcementReads) }},
		{"announcement", func() error { return sqlAnn.ImportAnnouncements(announcements) }},
		{"event", func() error { return sqlEvent.ImportEvents(events) }},
		{"handover", func() error { return sqlHandover.ImportHandovers(handovers) }},
		{"document", func() error { return sqlDocument.ImportDocuments(documents) }},
		{"attachment", func() error { return sqlAttachment.ImportAttachments(attachments) }},
		{"telegram", func() error { return sqlTelegram.ImportTelegram(telegramStore) }},
		{"unit", func() error { return sqlUnits.ImportUnits(units) }},
		{"vote", func() error { return sqlVotes.ImportBallots(votes) }},
		{"issue", func() error { return sqlIssues.ImportIssues(issues) }},
		{"identity", func() error { return identity.ImportProfiles(invites, time.Now()) }},
	} {
		if err := step.import_(); err != nil {
			return nil, fmt.Errorf("%s import to sqlite: %w", step.name, err)
		}
	}

	var activityBackend activityStorage = sqlActivity
	var annualStatementCostTypeBackend store.AnnualStatementCostTypeStorage = sqlAnnualStatementCostTypes
	var annualStatementPeriodBackend store.AnnualStatementPeriodStorage = sqlAnnualStatementPeriods
	var annualStatementPrepaymentBackend store.AnnualStatementPrepaymentStorage = sqlAnnualStatementPrepayments
	var annualStatementReceiptBackend store.AnnualStatementReceiptStorage = sqlAnnualStatementReceipts
	var profileBackend profileOverlayStorage = sqlProfileOverlay
	var notificationBackend notificationPrefStorage = sqlNotification
	var unitPaymentBackend unitPaymentStatusStorage = sqlUnitPayment
	var contactBackend contactBookStorage = sqlContacts
	var annReadBackend announcementReadStorage = sqlAnnRead
	var annBackend announcementStorage = sqlAnn
	var eventBackend eventStorage = sqlEvent
	var handoverBackend handoverStorage = sqlHandover
	var documentBackend documentStorage = sqlDocument
	var attachmentBackend attachmentStorage = sqlAttachment
	var telegramBackend telegramStorage = sqlTelegram
	var unitBackend unitStorage = sqlUnits
	var voteBackend voteStorage = sqlVotes
	var issueBackend issueStorage = sqlIssues
	var inviteBackend profileStorage = identity

	// Bound the one table that would otherwise grow monotonically: soft-deleted
	// attachment records whose files are long gone (HAUSV-146). The audit log is
	// the durable record of a deletion, so dropping the tombstone loses nothing.
	if n, err := sqlAttachment.PurgeDeletedBefore(time.Now().Add(-store.AttachmentTombstoneRetention)); err != nil {
		logError("attachment tombstone purge failed", err)
	} else if n > 0 {
		logInfo("expired attachment tombstones purged", "count", n)
	}

	// One-off: move legacy issue photos (ResidentIssue.PhotoPaths, written by the
	// long-removed SavePhoto) into the attachment store (HAUSV-175). Idempotent —
	// clearing the field is what marks an issue done — so it becomes a no-op once
	// it has run. Deliberately NON-fatal: the legacy read path is still in place,
	// so a failure here degrades to "photo still served the old way" rather than
	// blocking boot.
	tenantRefs := make([]store.TenantRef, 0, len(tenants))
	for _, t := range tenants {
		identity, ok := tenantIdentities[t.Slug]
		if !ok || !identity.Ref().Valid() {
			_ = database.Close()
			return nil, fmt.Errorf("tenant identity unavailable for %s", t.Slug)
		}
		tenantRefs = append(tenantRefs, identity.Ref())
	}
	if n, err := migrateLegacyIssuePhotos(issueBackend, attachmentBackend, issueAttachmentDir, tenantRefs, time.Now()); err != nil {
		logError("legacy issue photo migration incomplete", err, "fallback", "legacy_path")
	} else if n > 0 {
		logInfo("legacy issue photos migrated", "count", n)
	}

	// Filing a handover protocol writes a document AND the link on the handover
	// in one transaction (HAUSV-148). Both stores now always share the database,
	// so the atomic filer is always available; the guard stays as an assertion.
	// Check the concrete pointer, NOT the interface: a nil *SQLProtocolFiler
	// wrapped in an interface is itself non-nil, so an interface nil-check here
	// would never fire.
	sqlFiler := newSQLProtocolFiler(sqlDocument, sqlHandover)
	if sqlFiler == nil {
		return nil, fmt.Errorf("handover filing would not be atomic: document/handover stores are not sharing sqlite")
	}
	var filer protocolFiler = sqlFiler

	a := &app{
		baseURL:                   baseURL,
		addr:                      env("ADDR", ":8080"),
		rootDomain:                rootDomain,
		defaultTenant:             defaultTenant,
		tenants:                   tenants,
		organisations:             organisations,
		tenantIdentities:          tenantIdentities,
		sessionSecure:             parsed.Scheme == "https",
		allowed:                   allowed,
		admins:                    admins,
		profiles:                  profiles,
		localDevLogin:             localDevLogin,
		demoLogin:                 demoLogin,
		demoLoginCode:             demoLoginCode,
		serviceAccessEnabled:      serviceProviderAccessEnabled(),
		templExampleEnabled:       parseBool(env("TEMPL_EXAMPLE_ENABLED", "false")),
		sessionTTL:                sessionTTL,
		tokens:                    auth.NewTokenStore(secret),
		sessions:                  newSessionStore(secret),
		homeSetupTokens:           auth.NewTokenStore(homeSetupSecret(secret)),
		homeSetupSessions:         newSessionStore(homeSetupSecret(secret)),
		homeConnectorHashKey:      homeConnectorSecret(secret),
		oidc:                      oidcLogin,
		oidcFlows:                 auth.NewOIDCFlowStore(),
		mailer:                    mailTransport,
		trustedProxies:            trustedProxies,
		templates:                 tmpl,
		pool:                      database,
		tenantDB:                  tenantDB,
		scopedDB:                  scoped,
		dataDir:                   filepath.Dir(dbPath),
		announcementStore:         annBackend,
		announcementReadStore:     annReadBackend,
		eventStore:                eventBackend,
		notificationPrefs:         notificationBackend,
		profileOverlays:           profileBackend,
		tenantOverrides:           tenantOverrides,
		tenantHeroDir:             tenantHeroDir,
		tenantHeroSeedDir:         tenantHeroSeedDir,
		inviteStore:               inviteBackend,
		identityStore:             identity,
		activityStore:             activityBackend,
		annualStatementCostTypes:  annualStatementCostTypeBackend,
		annualStatementPeriods:    annualStatementPeriodBackend,
		annualStatementAkontos:    annualStatementPrepaymentBackend,
		annualStatementReceipts:   annualStatementReceiptBackend,
		annualConsumption:         store.NewSQLAnnualStatementConsumptionStore(tenantDB),
		annualStatementRuns:       store.NewSQLAnnualStatementRunStore(tenantDB, documentBackend),
		annualStatementDeliveries: store.NewSQLAnnualStatementDeliveryStore(tenantDB),
		unitStore:                 unitBackend,
		unitPaymentStore:          unitPaymentBackend,
		issueStore:                issueBackend,
		attachmentStore:           attachmentBackend,
		contactStore:              contactBackend,
		auditStore:                auditStore,
		documentStore:             documentBackend,
		handoverStore:             handoverBackend,
		protocolFiler:             filer,
		voteStore:                 voteBackend,
		voteReminderInterval:      voteReminderInterval,
		parkingStore:              parkingStore,
		parkingSampleInterval:     parkingSampleInterval,
		parkingHistoryStart:       parkingHistoryStart,
		energyStore:               energyBackend,
		homeReservations:          homeReservationBackend,
		homePortals:               homePortalBackend,
		homeConnectors:            homeConnectorBackend,
		homeConnectorReadings:     homeConnectorReadingBackend,
		homeConnectorDownloadDir:  env("HOME_CONNECTOR_DOWNLOAD_DIR", "/connector-downloads"),
		energySampleInterval:      energySampleInterval,
		energySamplers:            map[string]*energySamplerState{},
		mapTileBaseURL:            env("MAP_TILE_BASE_URL", ""),
		geocoder:                  newNominatimGeocoder(env("GEOCODING_BASE_URL", "")),
		mapPreviewTiles:           map[string]map[mapTileKey]time.Time{},
		inboxSuggestTimeout:       inboxSuggestTimeout,
		suggestJobs:               map[string]*suggestJob{},
		triageProviders:           map[string]triageProviderCacheEntry{},

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
		telegramStore:       telegramBackend,
		telegramPollTimeout: telegramPollTimeout,
	}
	a.intake = func(orgKey string) store.IntakeRepository { return store.BindIntakeRepository(database, orgKey) }
	a.orgSettings = func(orgKey string) store.OrgSettingsRepository {
		return store.BindOrgSettingsRepository(database, orgKey)
	}
	a.organisationRepo = func(orgKey string) store.OrganisationRepository {
		return store.BindOrganisationRepository(database, orgKey)
	}
	a.organisationMemberRepo = func(orgKey string) store.OrganisationMemberRepository {
		return store.BindOrganisationMemberRepository(database, orgKey)
	}
	a.intakeMailSeen = func(orgKey string) store.IntakeMailSeenRepository {
		return store.BindIntakeMailSeenRepository(database, orgKey)
	}
	// A mailbox per organisation. Broken configuration stops the boot: a
	// Verwaltung that believes its mail is being read must not be wrong.
	mailIntakeConfigs, err := mailintake.ParseConfigs(env("INTAKE_MAIL_JSON", ""))
	if err != nil {
		return nil, err
	}
	for key := range mailIntakeConfigs {
		if _, ok := a.organisations[key]; !ok {
			return nil, fmt.Errorf("INTAKE_MAIL_JSON names unknown organisation %q", key)
		}
	}
	a.mailIntakeConfigs = mailIntakeConfigs
	a.mailIntakeDir = env("INTAKE_MAIL_DIR", filepath.Join(attachmentFileDir, "intake-mail"))
	// The configured organisations become rows here, once, so every read path
	// below can take the Verwaltung from the store instead of from the map.
	a.syncOrganisations(context.Background())
	a.textbausteine = func(orgKey string) store.TextbausteinRepository {
		return store.BindTextbausteinRepository(database, orgKey)
	}
	if seedDir := strings.TrimSpace(os.Getenv("DEMO_SEED_DIR")); a.demoLogin && seedDir != "" {
		a.demoReset = func(ctx context.Context, anchor time.Time, out io.Writer) (demo.SeedResult, error) {
			options := demo.SeedOptions{Reset: true, DiscardAnnualStatements: true, Stats: true, Out: out, Anchor: anchor, DocumentDir: documentFileDir}
			if units, ok := a.unitStore.(store.UnitSink); ok {
				options.Units = units
			}
			return demo.Load(ctx, database, seedDir, options)
		}
	}
	if suggester, err := ai.NewFromEnv(os.Getenv); err != nil {
		logError("ai triage provider not configured", err)
	} else if suggester != nil {
		a.triage = suggester
		logInfo("ai triage provider configured", "label", suggester.Label())
	}
	return a, nil
}

// serviceProviderAccessEnabled is the runtime launch gate for external
// Dienstleister. The boolean alone is deliberately insufficient: an operator
// must also attest to the exact documented self-assessment revision. A future
// assessment change therefore closes old deployments until the operator has
// consciously reviewed and adopted it.
func serviceProviderAccessEnabled() bool {
	return parseBool(env("SERVICE_PROVIDER_ACCESS_ENABLED", "false")) &&
		strings.TrimSpace(env("SERVICE_PROVIDER_ASSESSMENT_VERSION", "")) == serviceProviderAssessmentVersion
}

func (a *app) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if a == nil || a.pool == nil || a.pool.PingContext(ctx) != nil || probeWritableDir(a.dataDir) != nil || a.retentionFailure.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"service":"hausv-org","status":"unhealthy"}`)
		return
	}
	_, _ = io.WriteString(w, `{"service":"hausv-org","status":"ok"}`)
}

func probeWritableDir(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return fmt.Errorf("data directory unavailable")
	}
	f, err := os.CreateTemp(dir, ".hausv-health-*")
	if err != nil {
		return err
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	return os.Remove(name)
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
	if !ok || !a.isAllowed(payload.Email, tenant.Slug) || !a.portalModulesFor(tenant.Slug).Events {
		http.NotFound(w, r)
		return
	}
	role := a.roleFor(payload.Email, tenant.Slug)
	profile := a.profileForTenant(payload.Email, tenant.Slug)
	identity, ok := a.tenantIdentity(tenant.Slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	tenantRef := identity.Ref()
	body := a.renderCalendarFeed(tenant, tenantRef, profile, role, time.Now().UTC(), a.repositoriesForTenant(tenantRef).events)
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

func (a *app) renderCalendarFeed(tenant tenantConfig, tenantRef store.TenantRef, profile userProfile, role string, now time.Time, events eventRepository) string {
	email := normalizeEmail(profile.Email)
	var b strings.Builder
	calendarLine(&b, "BEGIN", "VCALENDAR")
	calendarLine(&b, "VERSION", "2.0")
	calendarLine(&b, "PRODID", "-//hausv.org//Portal//DE")
	calendarLine(&b, "CALSCALE", "GREGORIAN")
	calendarLine(&b, "METHOD", "PUBLISH")
	calendarLine(&b, "X-WR-CALNAME", "hausv.org "+tenant.Name)
	calendarLine(&b, "X-WR-CALDESC", "Termine und freigegebene Vorgänge für "+tenant.Address)
	if roleCanUseResidentAreas(role) && events != nil {
		for _, item := range events.Upcoming(now) {
			a.writeCalendarEvent(&b, tenant, item, now)
		}
	}
	issues, _ := store.BindIssueRepository(a.issueStore, tenantRef)
	if issues != nil {
		for _, item := range issues.List() {
			if (strings.TrimSpace(item.ServiceProposal) == "" && item.ServiceProposedStart.IsZero()) || !a.canViewIssueForActor(tenantRef, item, email, role) {
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
	calendarLine(b, "UID", "event-"+item.ID+"+"+tenant.Slug+"@hausv.org")
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
		calendarLine(b, "UID", "issue-proposal-"+item.ID+"+"+tenant.Slug+"@hausv.org")
		calendarLine(b, "DTSTAMP", calendarDateTime(now))
		calendarLine(b, "DTSTART", calendarDateTime(item.ServiceProposedStart))
		calendarLine(b, "DTEND", calendarDateTime(end))
		calendarLine(b, "SUMMARY", "Termin: "+item.Title)
		calendarLine(b, "DESCRIPTION", description)
		calendarLine(b, "END", "VEVENT")
		return
	}

	calendarLine(b, "BEGIN", "VTODO")
	calendarLine(b, "UID", "issue-proposal-"+item.ID+"+"+tenant.Slug+"@hausv.org")
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
		raw = internalTenantPath(r, raw)
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
		parsed.Path = internalTenantPath(r, parsed.Path)
		if parsed.Path == "/app" || strings.HasPrefix(parsed.Path, "/app/") {
			if parsed.RawQuery != "" {
				return parsed.Path + "?" + parsed.RawQuery
			}
			return parsed.Path
		}
	}
	return fallback
}

func internalTenantPath(r *http.Request, raw string) string {
	if r == nil {
		return raw
	}
	resolved, _ := resolvedTenantFromContext(r.Context())
	return stripTenantPath(raw, resolved.tenant.Slug)
}

func stripTenantPath(raw string, tenantSlug string) string {
	prefix := "/" + normalizeSlug(tenantSlug)
	if prefix == "/" {
		return raw
	}
	if raw == prefix {
		return "/"
	}
	if strings.HasPrefix(raw, prefix+"/") {
		return strings.TrimPrefix(raw, prefix)
	}
	return raw
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

// portalIsDense decides which composition the Hausüberblick uses.
//
// HAUSV-527: density follows content as well as role. Managing roles get the
// dense composition because they come to the screen to work, but an empty house
// gives them nothing to be dense about — the dense layout then renders tall
// empty cards, which reads worse than the calm one and loses the reassurance the
// previous page carried. Residents never get it: a wall of maintenance tickets
// that are not theirs is intimidating, not useful.
func portalIsDense(role string, openIssues, upcomingEvents, unreadAnnouncements int) bool {
	if role != roleManager && role != roleAdmin {
		return false
	}
	return openIssues > 0 || upcomingEvents > 0 || unreadAnnouncements > 0
}

func (a *app) portal(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	modules := a.portalModulesFor(tenant.Slug)
	if isServiceProviderRole(role) {
		if !modules.Issues {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/app/anliegen", http.StatusSeeOther)
		return
	}
	profile := a.profileForTenant(email, tenant.Slug)
	now := time.Now()
	lastSeen := time.Time{}
	announcementReads := ac.repositories.announcementReads
	if announcementReads != nil {
		lastSeen = announcementReads.LastSeen(email)
	}
	signals := a.portalSignals(ac.repositories, ac.tenantRef, email, role, now, lastSeen, modules)
	digest := a.dashboardDigestItems(ac.repositories, ac.tenantRef, email, role, now, lastSeen, signals, modules)
	var primary dashboardDigestItem
	hasPrimary := false
	followUps := make([]dashboardDigestItem, 0, 3)
	for _, item := range digest {
		if !hasPrimary && item.Actionable {
			primary = item
			hasPrimary = true
			continue
		}
		if len(followUps) < 3 {
			followUps = append(followUps, item)
		}
	}
	canSeeParking := modules.Parking && (ac.can(capabilityPlatformAdmin) || profile.HasPermission(permissionParking))
	canManagePortalHandovers := modules.Handovers && canManageHandovers(ac.actor(), ac.resource())
	canManagePortalUsers := modules.Users && ac.can(capabilityManageUsers)
	hasHomeUtilities := canSeeParking || canManagePortalHandovers || canManagePortalUsers
	canResidentAreas := roleCanUseResidentAreas(role)
	canManageIssueBoard := ac.can(capabilityManageIssues)
	// Ein Eintrag, ein Platz: was der Tagesfokus schon beim Namen nennt, lassen
	// die Karten darunter weg. Die Zahlen in den Kartenköpfen bleiben trotzdem
	// die echten Gesamtwerte und sagen das auch.
	surfaced := map[string]bool{}
	if hasPrimary {
		markDigestSource(surfaced, primary)
	}
	for _, item := range followUps {
		markDigestSource(surfaced, item)
	}
	remainingEvents := make([]houseEvent, 0, len(signals.events))
	for _, item := range signals.events {
		if !surfaced["event:"+item.ID] {
			remainingEvents = append(remainingEvents, item)
		}
	}
	remainingIssues := make([]residentIssue, 0, len(signals.openIssues))
	for _, item := range signals.openIssues {
		if !surfaced["issue:"+item.ID] {
			remainingIssues = append(remainingIssues, item)
		}
	}
	boardEvents := eventViews(firstN(remainingEvents, 3), now)
	boardAnnouncements := announcementViewsWithReadState(firstN(signals.announcements, 3), now, false, lastSeen)
	boardIssues := issueViewsForActor(tenant.Slug, firstN(remainingIssues, 3), role, email)
	energyCard, hasEnergyCard := portalEnergyView{}, false
	if modules.Energy {
		energyCard, hasEnergyCard = a.portalEnergyCard(ac, now)
	}
	openBallots := 0
	if canResidentAreas && modules.Votes {
		openBallots = openBallotCount(ac.repositories.votes, now)
	}
	issuesURL := "/app/anliegen"
	if canManageIssueBoard {
		issuesURL = "/app/anliegen/board"
	}
	data := map[string]any{
		"Title":                  houseDisplayName(tenant),
		"GreetingName":           firstNonEmpty(profile.FirstName, profile.DisplayName()),
		"CanSeeParking":          canSeeParking,
		"ActivePage":             "home",
		"PortalToday":            germanDateLong(now.In(time.Local)),
		"DashboardPrimary":       primary,
		"HasDashboardPrimary":    hasPrimary,
		"DashboardFollowUps":     followUps,
		"HasDashboardFollowUps":  len(followUps) > 0,
		"HasHomeUtilities":       hasHomeUtilities,
		"PortalEvents":           boardEvents,
		"HasPortalEvents":        len(boardEvents) > 0,
		"PortalEventTotalLabel":  portalEventTotalLabel(len(signals.events), len(boardEvents)),
		"PortalEventsInFocus":    len(boardEvents) == 0 && len(signals.events) > 0,
		"PortalAnnouncements":    boardAnnouncements,
		"HasPortalAnnouncements": len(boardAnnouncements) > 0,
		"PortalUnreadLabel":      portalUnreadLabel(signals.unreadAnnouncements),
		"PortalIssues":           boardIssues,
		"HasPortalIssues":        len(boardIssues) > 0,
		"HasPortalOpenIssues":    len(signals.openIssues) > 0,
		"PortalOpenIssueLabel":   portalOpenIssueLabel(len(signals.openIssues), len(boardIssues)),
		"PortalIssuesInFocus":    len(boardIssues) == 0 && len(signals.openIssues) > 0,
		"PortalIssuesURL":        issuesURL,
		"CanCreateResidentIssue": canCreateResidentIssue(ac.actor(), ac.resource()),
		"CanManageEvents":        canManageEvents(ac.actor(), ac.resource()),
		"CanManageAnnouncements": ac.can(capabilityManageAnnouncements),
		"PortalEnergy":           energyCard,
		"HasPortalEnergy":        hasEnergyCard,
		"PortalAreas":            portalAreaViews(modules, canResidentAreas, canSeeParking, canManagePortalHandovers, canManagePortalUsers, openBallots),
	}
	// The sidebar badges read the same numbers. Passing them explicitly keeps
	// render() from re-reading announcements and issues for this page.
	if ac.repositories.announcements != nil && announcementReads != nil && strings.TrimSpace(email) != "" {
		data["UnreadAnnouncements"] = signals.unreadAnnouncements
		data["HasUnreadAnnouncements"] = signals.unreadAnnouncements > 0
	}
	if a.issueStore != nil {
		data["OpenIssues"] = len(signals.openIssues)
		data["HasOpenIssues"] = len(signals.openIssues) > 0
	}
	areas := portalAreaViews(modules, canResidentAreas, canSeeParking, canManagePortalHandovers, canManagePortalUsers, openBallots)
	portalAreas := make([]web.PortalArea, 0, len(areas))
	for _, area := range areas {
		portalAreas = append(portalAreas, web.PortalArea{Label: area.Label, URL: area.URL})
	}
	energy := web.PortalEnergy{
		Ready:       energyCard.Ready,
		HomeName:    energyCard.HomeName,
		Message:     energyCard.Message,
		ActionLabel: energyCard.ActionLabel,
		ActionURL:   energyCard.ActionURL,
	}
	if !hasEnergyCard {
		energy = web.PortalEnergy{}
	}
	portalIssues := issueViewsForActor(tenant.Slug, signals.openIssues, role, email)
	for i := range portalIssues {
		portalIssues[i].AssigneeName = portalIssues[i].AssigneeEmail
		if profile, ok := a.directoryProfile(portalIssues[i].AssigneeEmail); ok {
			portalIssues[i].AssigneeName = profile.DisplayName()
		}
	}
	portalEvents := eventViews(signals.events, now)
	portalAnnouncements := announcementViewsWithReadState(signals.announcements, now, false, lastSeen)
	// HAUSV-527: density follows content as well as role. An empty house gives a
	// manager nothing to be dense about — the dense composition then renders tall
	// empty cards, which reads worse than the calm one and loses the reassurance
	// the old page carried. Calm for everyone when nothing is waiting.
	portalDense := portalIsDense(role, len(portalIssues), len(portalEvents), signals.unreadAnnouncements)
	shell := a.portalShellData(&ac)

	a.renderPortalTempl(w, r, web.PortalPageData{
		Title:                  "Hausüberblick · " + houseDisplayName(tenant) + " · " + role,
		TenantSlug:             tenant.Slug,
		HouseName:              houseDisplayName(tenant),
		Address:                tenant.Address,
		MapURL:                 tenantMapURL(tenant.Address),
		HeroImageURL:           tenant.HeroImageURL,
		BrandIcon:              tenant.BrandIcon,
		BrandMarkSVG:           tenantBrandMarkSVG(tenant.BrandIcon),
		Map:                    portalMapForTenant(tenant),
		GreetingName:           rolePreviewGreetingName(&ac, firstNonEmpty(profile.FirstName, profile.DisplayName())),
		Today:                  germanDateLong(now.In(time.Local)),
		DisplayName:            profile.DisplayName(),
		Initials:               profile.Initials(),
		Role:                   role,
		DisplayVersion:         version.DisplayVersion(version.Version),
		Dense:                  portalDense,
		Modules:                web.PortalModules{Energy: modules.Energy, Announcements: modules.Announcements, Events: modules.Events, Contacts: modules.Contacts, Documents: modules.Documents, Issues: modules.Issues, Votes: modules.Votes, Parking: modules.Parking, Handovers: modules.Handovers, Users: modules.Users, Audit: modules.Audit, Help: modules.Help},
		CanUseResidentAreas:    canResidentAreas,
		CanViewEnergy:          modules.Energy && a.canViewEnergy(ac),
		CanManageIssues:        canManageIssueBoard,
		CanCreateResidentIssue: canCreateResidentIssue(ac.actor(), ac.resource()),
		CanSeeParking:          canSeeParking,
		CanManageHandovers:     canManagePortalHandovers,
		CanManageUsers:         canManagePortalUsers,
		CanViewAudit:           modules.Audit && canViewAudit(ac.actor(), ac.resource()),
		ShowVerwaltungNav:      shell.IsOrganisationMember,
		ShowInboxNav:           shell.ShowInboxNav,
		InboxOpenCount:         shell.InboxOpenCount,
		RolePreview:            rolePreviewPortalData(&ac),
		RolePreviewChoices:     a.rolePreviewChoices(&ac),
		Flash:                  rolePreviewFlash(r),
		HomeIdentity:           a.homeIdentityForActor(ac, modules.Energy && a.canViewEnergy(ac)),
		HasPrimary:             hasPrimary,
		Primary:                primary,
		Issues:                 portalIssues,
		Events:                 portalEvents,
		Announcements:          portalAnnouncements,
		UnreadAnnouncements:    signals.unreadAnnouncements,
		Energy:                 energy,
		HasEnergy:              hasEnergyCard,
		Areas:                  portalAreas,
		Contexts:               shell.PortalContexts,
		Shell:                  shell,
		ReleaseNotes:           version.Notes(),
	})
}

func rolePreviewFlash(r *http.Request) string {
	if r != nil && r.URL.Query().Get("flash") == "Wird gebaut" {
		return "Wird gebaut"
	}
	return ""
}

func (a *app) renderPortalTempl(w http.ResponseWriter, r *http.Request, data web.PortalPageData) {
	var rendered bytes.Buffer
	if err := web.PortalPage(data).Render(r.Context(), &rendered); err != nil {
		logError("templ portal render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), data.TenantSlug))
}

// portalAreaView is one entry-point tile on the overview. Detail says what the
// area is for; Note carries a live number only when there is one.
type portalAreaView struct {
	Icon       string
	Label      string
	Detail     string
	URL        string
	Note       string
	HasNote    bool
	Management bool
}

type portalEnergyStat struct {
	Label string
	Value string
	Muted bool
}

type portalEnergyView struct {
	Ready       bool
	HomeName    string
	ModeLabel   string
	ModeActive  bool
	Message     string
	Stats       []portalEnergyStat
	ActionLabel string
	ActionURL   string
	Footnote    string
}

// portalSignals is the tenant state the overview screen works from: the visible
// announcements, the upcoming events and the issues this actor may see. It is
// read once per request and then shared by the daily focus, the board cards and
// the sidebar badges, so opening the portal stays a single pass per store.
type portalSignals struct {
	announcements       []announcement
	unreadAnnouncements int
	events              []houseEvent
	issues              []residentIssue
	openIssues          []residentIssue
}

func (a *app) portalSignals(repositories requestRepositories, tenant store.TenantRef, email string, role string, now time.Time, lastSeen time.Time, modules portalModuleFlags) portalSignals {
	signals := portalSignals{}
	if modules.Announcements && repositories.announcements != nil {
		signals.announcements = repositories.announcements.Visible(now)
		signals.unreadAnnouncements = unreadAnnouncementCount(signals.announcements, lastSeen, now)
	}
	if modules.Events && repositories.events != nil {
		signals.events = repositories.events.Upcoming(now)
	}
	if modules.Issues && a.issueStore != nil {
		signals.issues = a.visibleIssuesForActor(tenant, email, role)
		signals.openIssues = make([]residentIssue, 0, len(signals.issues))
		for _, item := range signals.issues {
			if issueIsOpen(item) {
				signals.openIssues = append(signals.openIssues, item)
			}
		}
	}
	return signals
}

func firstN[T any](items []T, limit int) []T {
	if limit < 0 || len(items) <= limit {
		return items
	}
	return items[:limit]
}

// portalUnreadLabel is the short badge on the Aushang card. The daily focus
// already spells the count out, so the card stays terse.
func portalUnreadLabel(unread int) string {
	if unread <= 0 {
		return ""
	}
	return strconv.Itoa(unread) + " ungelesen"
}

// markDigestSource remembers which record a focus item was built from so the
// board cards can leave it out instead of saying the same thing twice.
func markDigestSource(surfaced map[string]bool, item dashboardDigestItem) {
	if item.SourceKind == "" || item.SourceID == "" {
		return
	}
	surfaced[item.SourceKind+":"+item.SourceID] = true
}

// portalEventTotalLabel explains a shortened list: it only appears when the
// card shows fewer rows than the house actually has ahead, and then it names
// the real total.
func portalEventTotalLabel(total int, shown int) string {
	if shown == 0 || total <= shown {
		return ""
	}
	return pluralizeCount(total, "Termin insgesamt", "Termine insgesamt")
}

// portalOpenIssueLabel keeps the Anliegen card honest: the number is always the
// full count of open items, and it says "insgesamt" as soon as one of them is
// already shown in the daily focus above.
func portalOpenIssueLabel(total int, shown int) string {
	if total <= 0 {
		return ""
	}
	if shown < total {
		return pluralizeCount(total, "offenes Anliegen insgesamt", "offene Anliegen insgesamt")
	}
	return pluralizeCount(total, "offenes Anliegen", "offene Anliegen")
}

func openBallotCount(votes voteRepository, now time.Time) int {
	if votes == nil {
		return 0
	}
	count := 0
	for _, item := range votes.List() {
		if _, _, active := ballotStatusForView(item, now); active {
			count++
		}
	}
	return count
}

// portalAreaViews lists the areas that do not already have their own card on
// the overview — the energy card links to "Mein Zuhause", so that area is
// deliberately absent here. Each entry is role-scoped: a tile only appears when
// the actor may actually open it.
func portalAreaViews(modules portalModuleFlags, canResidentAreas bool, canSeeParking bool, canManageHandovers bool, canManageUsers bool, openBallots int) []portalAreaView {
	areas := make([]portalAreaView, 0, 7)
	if canResidentAreas {
		if modules.Documents {
			areas = append(areas, portalAreaView{Icon: "document", Label: "Dokumente", Detail: "Protokolle, Verträge und Nachweise.", URL: "/app/dokumente"})
		}
		if modules.Votes {
			ballots := portalAreaView{Icon: "vote", Label: "Abstimmungen", Detail: "Beschlüsse und laufende Entscheidungen.", URL: "/app/abstimmungen"}
			if openBallots > 0 {
				ballots.Note = pluralizeCount(openBallots, "Abstimmung läuft", "Abstimmungen laufen")
				ballots.HasNote = true
			}
			areas = append(areas, ballots)
		}
		if modules.Contacts {
			areas = append(areas, portalAreaView{Icon: "contact", Label: "Kontakte", Detail: "Verwaltung, Beirat und Dienstleister.", URL: "/app/kontakte"})
		}
	}
	if canSeeParking {
		areas = append(areas, portalAreaView{Icon: "parking", Label: "Parkplatznutzung", Detail: "Verbrauch und Abrechnung.", URL: "/app/parking", Management: true})
	}
	if canManageHandovers {
		areas = append(areas, portalAreaView{Icon: "handover", Label: "Übergaben", Detail: "Termine und Protokolle.", URL: "/app/uebergaben", Management: true})
	}
	if canManageUsers {
		areas = append(areas, portalAreaView{Icon: "users", Label: "Benutzer & Rechte", Detail: "Zugänge verwalten.", URL: "/app/settings/users", Management: true})
	}
	if canResidentAreas {
		areas = append(areas, portalAreaView{Icon: "settings", Label: "Einstellungen", Detail: "Profil, Haus und Benachrichtigungen.", URL: "/app/settings"})
	}
	return areas
}

// portalEnergyCard is the compact energy signal on the overview. It never
// invents a number: without a completed setup it offers the setup, and without
// measured intervals it says so plainly instead of showing a zero.
func (a *app) portalEnergyCard(ac authCtx, now time.Time) (portalEnergyView, bool) {
	if a.energyStore == nil || !a.canViewEnergy(ac) {
		return portalEnergyView{}, false
	}
	card := portalEnergyView{
		HomeName:    "Mein Zuhause",
		ActionLabel: "Einrichtung starten",
		ActionURL:   "/app/zuhause/onboarding",
		Footnote:    "Im geplanten Leistungstarif zählt die höchste Viertelstunde eines Monats.",
	}
	profile, exists, err := a.energyFor(ac).Profile(ac.tenant.Slug)
	if err != nil {
		card.Message = "Die Energiedaten sind gerade nicht abrufbar. Bitte später erneut ansehen."
		card.ActionLabel = "Zuhause öffnen"
		card.ActionURL = "/app/energie"
		return card, true
	}
	if !exists || !profile.OnboardingComplete {
		card.Message = "HAUSV liest zuerst nur mit und schaltet nichts. Die Einrichtung erfasst, welche Verbraucher es gibt und welche Messwerte bereits vorliegen."
		return card, true
	}
	card.Ready = true
	card.ActionLabel = "Zuhause öffnen"
	card.ActionURL = "/app/energie"
	if name := strings.TrimSpace(profile.HouseholdName); name != "" {
		card.HomeName = name
	}
	card.ModeActive = profile.OperatingMode != energy.ModeObserve
	card.ModeLabel = "Nur beobachten"
	if card.ModeActive {
		card.ModeLabel = "Aktive Steuerung"
	}
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	intervals, _ := a.energyFor(ac).ListIntervals(ac.tenant.Slug, monthStart.UTC(), time.Time{})
	peak := energy.PeakForMonth(intervals, now, time.Local)
	peakStat := portalEnergyStat{Label: "Spitze diesen Monat", Value: "Noch nicht gemessen", Muted: true}
	if peak > 0 {
		peakStat = portalEnergyStat{Label: "Spitze diesen Monat", Value: formatEnergyCompact(peak, 1) + " kW"}
	} else {
		card.Message = "Für diesen Monat liegen noch keine Messwerte vor. Solange nichts gemessen ist, nennt HAUSV keine Spitze."
	}
	card.Stats = []portalEnergyStat{
		peakStat,
		portalEnergyStatValue("Zielwert", profile.TargetPeakKW, "Nicht gesetzt"),
		portalEnergyStatValue("Vereinbarte Leistung", profile.AgreedPowerKW, "Nicht erfasst"),
	}
	return card, true
}

func portalEnergyStatValue(label string, value *float64, fallback string) portalEnergyStat {
	if value == nil || *value <= 0 {
		return portalEnergyStat{Label: label, Value: fallback, Muted: true}
	}
	return portalEnergyStat{Label: label, Value: formatEnergyCompact(*value, 1) + " kW"}
}

func (a *app) dashboardDigestItems(repositories requestRepositories, tenant store.TenantRef, email string, role string, now time.Time, lastSeen time.Time, signals portalSignals, modules portalModuleFlags) []dashboardDigestItem {
	tenantSlug := tenant.Slug
	actor := actorFor(email, tenantSlug, role)
	resource := resourceFor(tenantSlug)
	var paymentItem *dashboardDigestItem
	var announcementItem *dashboardDigestItem
	var issueItem *dashboardDigestItem
	var eventItem *dashboardDigestItem
	if modules.Contacts {
		for _, status := range unitPaymentStatusViewsForEmail(repositories.units, repositories.unitPayments, email) {
			if status.StatusValue == unitPaymentStatusPaid {
				continue
			}
			title := "Zahlungsstatus prüfen"
			if status.StatusValue == unitPaymentStatusOverdue {
				title = "Offenen Zahlungsstatus klären"
			}
			item := dashboardDigestItem{
				Kind:        "Zahlungsstatus",
				Title:       title,
				Detail:      status.UnitLabel + " · " + status.Status,
				URL:         "/app/kontakte",
				ActionLabel: "Verwaltung kontaktieren",
				Actionable:  true,
			}
			paymentItem = &item
			break
		}
	}
	unread := signals.unreadAnnouncements
	if unread > 0 {
		title := "Neue Aushänge lesen"
		if unread == 1 {
			title = "Neuen Aushang lesen"
		}
		item := dashboardDigestItem{
			Kind:        "Aushang",
			Title:       title,
			Detail:      pluralizeCount(unread, "ungelesener Beitrag", "ungelesene Beiträge"),
			URL:         "/app/announcements",
			ActionLabel: "Jetzt lesen",
			Actionable:  true,
		}
		announcementItem = &item
	}
	if a.issueStore != nil {
		visible := signals.issues
		if can(actor, capabilityManageIssues, resource) {
			openIssues := signals.openIssues
			if len(openIssues) > 0 {
				views := a.issueViewsForActor(tenant, openIssues, role, email)
				first := views[0]
				// Der Fokus nennt den Fall beim Namen; die Gesamtzahl steht im
				// Kopf der Anliegen-Karte. Sonst stünden dieselbe Zahl und
				// dasselbe Verb zweimal auf dem Schirm.
				item := dashboardDigestItem{
					Kind:        "Anliegen",
					Title:       first.Title,
					Detail:      first.NextStep,
					URL:         first.DetailURL,
					ActionLabel: first.DetailAction,
					Actionable:  true,
					SourceKind:  "issue",
					SourceID:    first.ID,
				}
				issueItem = &item
			}
		} else {
			views := a.issueViewsForActor(tenant, visible, role, email)
			var waiting *issueView
			for i := range views {
				item := &views[i]
				switch item.ResidentState {
				case "question":
					issueItem = &dashboardDigestItem{
						Kind:        "Ihr Anliegen",
						Title:       "Rückfrage beantworten",
						Detail:      item.Title + " · " + item.NextStep,
						URL:         item.DetailURL,
						ActionLabel: item.DetailAction,
						Actionable:  true,
						SourceKind:  "issue",
						SourceID:    item.ID,
					}
				case "resolution":
					issueItem = &dashboardDigestItem{
						Kind:        "Ihr Anliegen",
						Title:       "Lösung prüfen",
						Detail:      item.Title + " · " + item.NextStep,
						URL:         item.DetailURL,
						ActionLabel: item.DetailAction,
						Actionable:  true,
						SourceKind:  "issue",
						SourceID:    item.ID,
					}
				case "waiting":
					if waiting == nil && issueIsOpen(visible[i]) {
						waiting = item
					}
				}
				if issueItem != nil {
					break
				}
			}
			if issueItem == nil && waiting != nil {
				issueItem = &dashboardDigestItem{
					Kind:        "Ihr Anliegen",
					Title:       "Anliegen bleibt im Blick",
					Detail:      waiting.Title + " · " + waiting.NextStep,
					URL:         waiting.DetailURL,
					ActionLabel: "Status ansehen",
					Actionable:  false,
					SourceKind:  "issue",
					SourceID:    waiting.ID,
				}
			}
		}
	}
	upcoming := signals.events
	if len(upcoming) > 0 {
		views := a.eventViews(tenant, upcoming[:1], now, email, role)
		if len(views) > 0 {
			item := dashboardDigestItem{
				Kind:        "Termin",
				Title:       "Nächster Termin: " + views[0].Title,
				Detail:      views[0].StartsAt,
				URL:         "/app/events",
				ActionLabel: "Termine ansehen",
				Actionable:  false,
				SourceKind:  "event",
				SourceID:    views[0].ID,
			}
			eventItem = &item
		}
	}
	items := []dashboardDigestItem{}
	appendItem := func(item *dashboardDigestItem) {
		if item != nil {
			items = append(items, *item)
		}
	}
	if can(actor, capabilityManageIssues, resource) {
		appendItem(issueItem)
		appendItem(announcementItem)
	} else if paymentItem != nil {
		appendItem(paymentItem)
		appendItem(issueItem)
		appendItem(announcementItem)
	} else if issueItem != nil && issueItem.Actionable {
		appendItem(issueItem)
		appendItem(announcementItem)
	} else {
		appendItem(announcementItem)
		appendItem(issueItem)
	}
	appendItem(eventItem)
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
		logError("service provider invite persistence failed", err, "tenant", tenant.Slug, "recipient", redactedEmail(newAssignee))
		a.recordIssueServiceInviteAudit(tenant, after, actorEmail, actorRole, newAssignee, createdInvite, "nicht gespeichert")
		return
	}
	mailStatus := "verschickt"
	if err := a.sendServiceProviderMagicLink(r, tenant, after, newAssignee); err != nil {
		logWarn("service provider magic link delivery failed",
			"tenant", tenant.Slug,
			"error_type", fmt.Sprintf("%T", err),
		)
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
	if err := sendMagicLinkWithContext(r.Context(), a.mailer, email, link, tenant.Address); err != nil {
		a.tokens.Invalidate(token)
		return err
	}
	return nil
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
	if strings.HasPrefix(event.ActionURL, "/") && !strings.HasPrefix(event.ActionURL, "//") {
		event.ActionURL = strings.TrimRight(a.baseURL, "/") + event.ActionURL
	}
	body := event.Body()
	sent := []string{}
	for _, recipient := range a.notificationRecipients(event) {
		if err := a.mailer.SendNotification(recipient, event.Subject, body); err != nil {
			logWarn("notification delivery failed",
				"tenant", event.Tenant.Slug,
				"error_type", fmt.Sprintf("%T", err),
			)
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

const (
	parkingStatementCSVTitle = "Parkplatzabrechnung"
	parkingMonthCSVTitle     = "Parkplatzabrechnung – Monat"
)

func writeParkingStatementCSV(w io.Writer, statement parkingStatementView) error {
	writer := csv.NewWriter(w)
	writer.Comma = ';'
	rows := [][]string{
		{parkingStatementCSVTitle},
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
		{parkingMonthCSVTitle},
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

func (a *app) actorCanSeeCommonIssues(tenant store.TenantRef, email string, role string) bool {
	if normalizeRole(role) == roleOwner {
		return true
	}
	units, ok := store.BindUnitRepository(a.unitStore, tenant)
	if !ok {
		return false
	}
	for _, membership := range units.UnitsForEmail(email) {
		if normalizeRole(membership.Relation) == roleOwner {
			return true
		}
	}
	return false
}

func (a *app) settingsHub(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	modules := a.portalModulesFor(tenant.Slug)
	if denyServiceProviderArea(w, role) {
		return
	}
	calendarFeedURL := ""
	if modules.Events {
		if token, err := a.calendarFeedToken(email, tenant.Slug); err == nil {
			calendarFeedURL = a.publicBaseURL(r, tenant) + "/calendar/" + url.PathEscape(token) + ".ics"
		}
	}
	profile := a.profileForTenant(email, tenant.Slug)
	prefs := defaultNotificationPreferences()
	if a.notificationPrefs != nil {
		prefs = a.notificationPrefs.Get(email)
	}
	enabledNotifications := 0
	notificationOptions := notificationOptionsForModules(notificationEventOptions(prefs), modules)
	for _, option := range notificationOptions {
		if option.Checked {
			enabledNotifications++
		}
	}
	notificationSummary := fmt.Sprintf("%d von %d Themen aktiv", enabledNotifications, len(notificationOptions))
	if prefs.Unsubscribed {
		notificationSummary = "E-Mails pausiert"
	}
	homeURL := "/app/zuhause/onboarding"
	homeProfile, homeProfileExists, homeProfileErr := a.energyFor(ac).Profile(tenant.Slug)
	if homeProfileErr == nil && homeProfileExists && homeProfile.OnboardingComplete {
		homeURL = "/app/settings/home"
	}
	canManageEnergyData := modules.Energy && homeProfileErr == nil &&
		homeProfileExists &&
		!energyProfileUnclaimed(homeProfile) &&
		a.canManageHomeIdentityProfile(ac, homeProfile, homeProfileExists)
	pageData := map[string]any{
		"Title":                       "Einstellungen",
		"ActivePage":                  "settings",
		"CalendarFeedURL":             calendarFeedURL,
		"HasCalendarFeedURL":          calendarFeedURL != "",
		"SettingsDisplayName":         profile.DisplayName(),
		"SettingsNotificationSummary": notificationSummary,
		"SettingsHomeURL":             homeURL,
		"SettingsCanManageEnergyData": canManageEnergyData,
	}
	a.renderSettingsHubTempl(w, r, ac, pageData)
}

func (a *app) buildingSettings(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, _, _, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	buildingMsg, buildingOK := buildingSettingsMessage(r.URL.Query().Get("building"))
	section := normalizeBuildingSettingsSection(r.URL.Query().Get("section"))
	heroMsg, heroOK := buildingHeroMessage(r.URL.Query().Get("hero"))
	unitMsg, unitOK := buildingUnitMessage(r.URL.Query().Get("unit"))
	paymentMsg, paymentOK := unitPaymentStatusMessage(r.URL.Query().Get("payment"))
	units := ac.repositories.units.List()
	billableWeight := billableUnitWeight(units)
	fairUseExceeded := billableWeight > fairUseFreeUnits*unitBillableFullPPM
	homeProfile, hasHomeProfile, profileErr := a.energyFor(ac).Profile(tenant.Slug)
	if profileErr != nil || !homeProfile.OnboardingComplete {
		hasHomeProfile = false
	}
	homeProfileUnitLabel, hasHomeProfileUnit := a.energyHomeUnitLabel(ac.tenantRef, homeProfile)
	homeProfileScopeLabel := "gesamte Liegenschaft"
	if homeProfile.HomeType == energy.HomeApartment {
		homeProfileScopeLabel = "noch nicht zugeordnet"
		if hasHomeProfileUnit {
			homeProfileScopeLabel = homeProfileUnitLabel
		}
	}
	lucideIconNamesJSON, err := json.Marshal(web.LucideIconNames())
	if err != nil {
		lucideIconNamesJSON = []byte("[]")
	}
	pageData := map[string]any{
		"Title":                 "Gebäude & Einheiten",
		"ActivePage":            "settings",
		"BuildingMsg":           buildingMsg,
		"BuildingOK":            buildingOK,
		"BuildingSection":       section,
		"BrandIconOptions":      tenantBrandIconOptions(tenant.BrandIcon),
		"BrandIconLabel":        tenantBrandIconLabel(tenant.BrandIcon),
		"BrandIconIsLucide":     tenantBrandLucideName(tenant.BrandIcon) != "",
		"LucideIconNamesJSON":   template.JS(lucideIconNamesJSON),
		"HeroMsg":               heroMsg,
		"HeroOK":                heroOK,
		"HasCustomHero":         a.hasTenantHero(tenant.Slug),
		"UnitMsg":               unitMsg,
		"UnitOK":                unitOK,
		"Units":                 a.buildingUnitViewsWithOccupancy(ac.repositories, ac.tenantRef, units),
		"NewUnitTypeOptions":    unitTypeOptions(unitTypeResidential),
		"NewUnitPaymentOptions": unitPaymentStatusOptions(""),
		"UnitTotal":             len(units),
		"BillableUnits":         formatBillableUnitWeight(billableWeight),
		"BillableLabel":         billableUnitCountLabel(billableWeight),
		"FairUseFreeUnits":      fairUseFreeUnits,
		"FairUseExceeded":       fairUseExceeded,
		"UnitsEmpty":            emptyState("Noch keine Einheiten", "Angelegte Einheiten erscheinen hier mit Anteil und Kontaktlinks."),
		"PaymentMsg":            paymentMsg,
		"PaymentOK":             paymentOK,
		"HomeProfile":           homeProfile,
		"HasHomeProfile":        hasHomeProfile,
		"HomeTypeLabel":         energyHomeTypeLabel(homeProfile.HomeType),
		"HomeProfileUnitLabel":  homeProfileUnitLabel,
		"HasHomeProfileUnit":    hasHomeProfileUnit,
		"HomeProfileScopeLabel": homeProfileScopeLabel,
		"HomeProfileSaved":      r.URL.Query().Get("home") == "saved",
	}
	a.renderBuildingSettingsTempl(w, r, ac, pageData)
}

func normalizeBuildingSettingsSection(section string) string {
	switch strings.TrimSpace(strings.ToLower(section)) {
	case "contacts", "units", "appearance":
		return strings.TrimSpace(strings.ToLower(section))
	default:
		return "overview"
	}
}

func (a *app) auditLog(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, role := ac.tenant, ac.role
	if !canViewAudit(ac.actor(), ac.resource()) {
		http.Error(w, "Dieser Bereich ist für diesen Zugang nicht freigegeben.", http.StatusForbidden)
		return
	}
	action := normalizeAuditAction(r.URL.Query().Get("action"))
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	events := []auditEvent{}
	if a.auditStore != nil {
		events = a.auditStore.List(auditFilter{
			TenantSlug: tenant.Slug,
			Limit:      500,
		})
	}
	fullAudit := canViewFullAudit(ac.actor(), ac.resource())
	if !fullAudit {
		events = a.scopedAuditEvents(ac, events)
	}
	availableEvents := events
	events = filterAuditEvents(events, action, query)
	eventViews := auditEventViews(a.auditEventsForView(ac.repositories, ac.tenantRef, events, fullAudit))
	stats := auditStats(events, action, query)
	auditTitle := "Mein Verlauf"
	auditLede := "Was in Ihrem Konto und bei freigegebenen Vorgängen passiert ist. Interne Verwaltungsdetails bleiben geschützt."
	if fullAudit {
		auditTitle = "Aktivitätsverlauf"
		auditLede = "Änderungen und Zugriffe für " + tenant.Address + "."
	} else if normalizeRole(role) == roleServiceProvider {
		auditTitle = "Freigegebener Verlauf"
		auditLede = "Änderungen bei den Vorgängen, auf die Sie aktuell Zugriff haben. Interne Verwaltungsdetails bleiben geschützt."
	}
	a.renderAuditTempl(w, r, web.AuditPageData{
		Portal:         a.auditPortalContext(ac, auditTitle),
		Events:         eventViews,
		HasEvents:      len(eventViews) > 0,
		HasAnyEvents:   len(availableEvents) > 0,
		EventsEmpty:    emptyState("Noch nichts im Verlauf", "Relevante Änderungen an Ihrem Zugang und Ihren Vorgängen erscheinen hier."),
		ActionOptions:  auditActionOptionsForEvents(action, availableEvents),
		SearchQuery:    query,
		AuditStats:     stats,
		AuditPageTitle: auditTitle,
		AuditLede:      auditLede,
		AuditIsFull:    fullAudit,
		CanManageUsers: ac.can(capabilityManageUsers),
	})
}

func (a *app) auditEventsForView(repositories requestRepositories, tenant store.TenantRef, events []auditEvent, includeTechnicalID bool) []auditEvent {
	out := make([]auditEvent, 0, len(events))
	for _, event := range events {
		event = copyAuditEvent(event)
		label := a.auditTargetTitle(repositories, tenant, event.TargetType, event.TargetID)
		if label != "" {
			if event.Details == nil {
				event.Details = map[string]string{}
			}
			event.Details["target_label"] = label
			if includeTechnicalID {
				event.Details["target_id"] = event.TargetID
			}
		}
		out = append(out, event)
	}
	return out
}

func (a *app) auditTargetTitle(repositories requestRepositories, tenant store.TenantRef, targetType string, targetID string) string {
	targetID = strings.TrimSpace(targetID)
	if targetID == "" {
		return ""
	}
	attachments, _ := store.BindAttachmentRepository(a.attachmentStore, tenant)
	documents, _ := store.BindDocumentRepository(a.documentStore, tenant)
	issues, _ := store.BindIssueRepository(a.issueStore, tenant)
	switch strings.TrimSpace(targetType) {
	case "attachment":
		if attachments != nil {
			if item, found := attachments.Get(targetID); found {
				switch normalizeAttachmentEntity(item.EntityType) {
				case "issue", "issue-estimate":
					if title := a.auditTargetTitle(repositories, tenant, "issue", item.EntityID); title != "" {
						return title
					}
				case "issue-comment":
					if issue, _, found := a.issueCommentTarget(tenant, item.EntityID); found {
						return strings.TrimSpace(issue.Title)
					}
				case "document", "ballot", "handover", "event":
					if title := a.auditTargetTitle(repositories, tenant, item.EntityType, item.EntityID); title != "" {
						return title
					}
				}
				return strings.TrimSpace(item.Filename)
			}
		}
	case "issue":
		if issues != nil {
			if item, found := issues.Get(targetID); found {
				return strings.TrimSpace(item.Title)
			}
		}
	case "document":
		if documents != nil {
			if item, found := documents.Get(targetID); found {
				return strings.TrimSpace(item.Title)
			}
		}
	case "store.Ballot", "ballot":
		if repositories.votes != nil {
			if item, found := repositories.votes.Get(targetID); found {
				return strings.TrimSpace(item.Title)
			}
		}
	case "handover":
		if repositories.handovers != nil {
			if item, found := repositories.handovers.Get(targetID); found {
				return strings.TrimSpace(item.Title)
			}
		}
	case "event":
		if repositories.events != nil {
			for _, item := range repositories.events.List() {
				if item.ID == targetID {
					return strings.TrimSpace(item.Title)
				}
			}
		}
	case "store.Unit", "unit":
		if repositories.units != nil {
			for _, item := range repositories.units.List() {
				if normalizeUnitID(item.ID) == normalizeUnitID(targetID) {
					return strings.TrimSpace(item.Label)
				}
			}
		}
	}
	return ""
}

func auditActionOptionsForEvents(selected string, events []auditEvent) []selectOption {
	selected = normalizeAuditAction(selected)
	actions := map[string]struct{}{}
	for _, event := range events {
		if action := normalizeAuditAction(event.Action); action != "" {
			actions[action] = struct{}{}
		}
	}
	if selected != "" {
		actions[selected] = struct{}{}
	}
	keys := make([]string, 0, len(actions))
	for action := range actions {
		keys = append(keys, action)
	}
	sort.Slice(keys, func(i, j int) bool {
		return auditActionLabel(keys[i]) < auditActionLabel(keys[j])
	})
	options := []selectOption{{Value: "", Label: "Alle Arten", Selected: selected == ""}}
	for _, action := range keys {
		options = append(options, selectOption{
			Value:    action,
			Label:    auditActionLabel(action),
			Selected: selected == action,
		})
	}
	return options
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
		logError("audit record failed", err, "action", event.Action, "tenant", event.TenantSlug)
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
	override, err := tenantBuildingOverrideFromForm(tenant, r.Form)
	// Keep the original combined form contract for older clients and bookmarks.
	// The current UI uses separate, safer forms for each tab.
	if r.Form.Has("brand_icon") {
		override, err = tenantOverrideFromForm(r.Form)
	} else if r.Form.Has("contact_name") || r.Form.Has("emergency_name") || r.Form.Has("caretaker_name") {
		var contacts tenantOverride
		contacts, err = tenantContactsOverrideFromForm(tenant, r.Form)
		if err == nil {
			contacts.Name, contacts.Address = override.Name, override.Address
			contacts.MapSet, contacts.MapLatitude, contacts.MapLongitude, contacts.MapZoom = override.MapSet, override.MapLatitude, override.MapLongitude, override.MapZoom
			override = contacts
		}
	}
	if err != nil {
		http.Redirect(w, r, "/app/settings/building?section=overview&building=invalid", http.StatusSeeOther)
		return
	}
	if a.tenantOverrides != nil {
		if err := a.tenantOverrides.SetMeta(tenant.Slug, override); err != nil {
			logError("building settings save failed", err, "tenant", tenant.Slug)
			http.Redirect(w, r, "/app/settings/building?section=overview&building=error", http.StatusSeeOther)
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
		Details:    map[string]string{"changed_fields": "Stammdaten, Kartenposition"},
	})
	http.Redirect(w, r, "/app/settings/building?section=overview&building=saved", http.StatusSeeOther)
}

func (a *app) updateBuildingContacts(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	override, err := tenantContactsOverrideFromForm(tenant, r.Form)
	if err != nil {
		http.Redirect(w, r, "/app/settings/building?section=contacts&building=invalid", http.StatusSeeOther)
		return
	}
	if a.tenantOverrides != nil {
		if err := a.tenantOverrides.SetMeta(tenant.Slug, override); err != nil {
			logError("building contacts save failed", err, "tenant", tenant.Slug)
			http.Redirect(w, r, "/app/settings/building?section=contacts&building=error", http.StatusSeeOther)
			return
		}
	}
	a.recordAudit(auditEvent{TenantSlug: tenant.Slug, ActorEmail: actorEmail, ActorRole: role, Action: auditActionBuildingUpdate, TargetType: "building", TargetID: tenant.Slug, Summary: "Hauskontakte geändert", Details: map[string]string{"changed_fields": "Verwaltung, Notdienst, Hausmeister"}})
	http.Redirect(w, r, "/app/settings/building?section=contacts&building=saved", http.StatusSeeOther)
}

func (a *app) updateBuildingAppearance(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	override, err := tenantAppearanceOverrideFromForm(tenant, r.Form)
	if err != nil {
		http.Redirect(w, r, "/app/settings/building?section=appearance&building=invalid", http.StatusSeeOther)
		return
	}
	if a.tenantOverrides != nil {
		if err := a.tenantOverrides.SetMeta(tenant.Slug, override); err != nil {
			logError("building appearance save failed", err, "tenant", tenant.Slug)
			http.Redirect(w, r, "/app/settings/building?section=appearance&building=error", http.StatusSeeOther)
			return
		}
	}
	a.recordAudit(auditEvent{TenantSlug: tenant.Slug, ActorEmail: actorEmail, ActorRole: role, Action: auditActionBuildingUpdate, TargetType: "building", TargetID: tenant.Slug, Summary: "Erscheinungsbild geändert", Details: map[string]string{"changed_fields": "Portal-Symbol, Kurzkennung"}})
	http.Redirect(w, r, "/app/settings/building?section=appearance&building=saved", http.StatusSeeOther)
}

func (a *app) updateBuildingHero(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(maxTenantHeroFormBytes); err != nil {
		http.Redirect(w, r, "/app/settings/building?section=appearance&hero=invalid", http.StatusSeeOther)
		return
	}
	header, ok := tenantHeroHeader(r)
	if !ok {
		http.Redirect(w, r, "/app/settings/building?section=appearance&hero=invalid", http.StatusSeeOther)
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
		logError("tenant hero upload failed", err, "tenant", tenant.Slug)
		http.Redirect(w, r, "/app/settings/building?section=appearance&hero=invalid", http.StatusSeeOther)
		return
	}
	if a.tenantOverrides != nil {
		if err := a.tenantOverrides.SetHeroImage(tenant.Slug, filename); err != nil {
			_ = a.removeTenantHeroImage(filename)
			logError("tenant hero save failed", err, "tenant", tenant.Slug)
			http.Redirect(w, r, "/app/settings/building?section=appearance&hero=error", http.StatusSeeOther)
			return
		}
	}
	if previous != "" && previous != filename {
		if err := a.removeTenantHeroImage(previous); err != nil {
			logError("tenant old hero cleanup failed", err, "tenant", tenant.Slug)
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
	http.Redirect(w, r, "/app/settings/building?section=appearance&hero=saved", http.StatusSeeOther)
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
			logError("tenant hero reset failed", err, "tenant", tenant.Slug)
			http.Redirect(w, r, "/app/settings/building?section=appearance&hero=error", http.StatusSeeOther)
			return
		}
	}
	if previous != "" {
		if err := a.removeTenantHeroImage(previous); err != nil {
			logError("tenant hero remove failed", err, "tenant", tenant.Slug)
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
	http.Redirect(w, r, "/app/settings/building?section=appearance&hero=removed", http.StatusSeeOther)
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
	origID := normalizeUnitID(r.FormValue("orig_id"))
	dialogTarget := "#unit-add"
	if origID != "" {
		dialogTarget = "#unit-" + origID
	}
	item, err := buildingUnitFromForm(tenant.Slug, r.Form)
	if err != nil {
		http.Redirect(w, r, "/app/settings/building?section=units&unit=invalid"+dialogTarget, http.StatusSeeOther)
		return
	}
	paymentStatus := ""
	if r.Form.Has("payment_status") {
		paymentStatus = normalizeUnitPaymentStatus(r.FormValue("payment_status"))
		if paymentStatus == "" {
			http.Redirect(w, r, "/app/settings/building?section=units&payment=invalid"+dialogTarget, http.StatusSeeOther)
			return
		}
	}
	if profile, exists, profileErr := a.energyFor(ac).Profile(tenant.Slug); profileErr != nil {
		http.Redirect(w, r, "/app/settings/building?section=units&unit=error"+dialogTarget, http.StatusSeeOther)
		return
	} else if exists && profile.HomeType == energy.HomeApartment {
		linkedID := normalizeUnitID(profile.UnitID)
		itemID := normalizeUnitID(item.ID)
		changesLinkedIdentity := linkedID != "" &&
			((origID == linkedID && (itemID != linkedID || normalizeUnitType(item.UnitType) != unitTypeResidential)) ||
				(itemID == linkedID && normalizeUnitType(item.UnitType) != unitTypeResidential))
		if changesLinkedIdentity {
			http.Redirect(w, r, "/app/settings/building?section=units&unit=home-linked"+dialogTarget, http.StatusSeeOther)
			return
		}
	}
	// The unit form knows nothing about the annual-statement bases
	// (HAUSV-577); an edit must carry them over instead of wiping them.
	if origID != "" {
		for _, existing := range ac.repositories.units.List() {
			if normalizeUnitID(existing.ID) == origID {
				item = store.CarryAllocationBases(existing, item)
				break
			}
		}
	}
	// Add/replace under one lock so a concurrent unit add/delete isn't lost to a
	// whole-slice overwrite (HAUSV-145).
	duplicate, err := ac.repositories.units.UpsertUnit(origID, item)
	if err != nil {
		logError("unit save failed", err, "tenant", tenant.Slug, "unit_id", item.ID)
		http.Redirect(w, r, "/app/settings/building?section=units&unit=error"+dialogTarget, http.StatusSeeOther)
		return
	}
	if duplicate {
		http.Redirect(w, r, "/app/settings/building?section=units&unit=duplicate"+dialogTarget, http.StatusSeeOther)
		return
	}
	if paymentStatus != "" {
		if _, err := ac.repositories.unitPayments.Set(unitPaymentStatus{TenantSlug: tenant.Slug, UnitID: item.ID, Status: paymentStatus, UpdatedBy: actorEmail}); err != nil {
			logError("unit payment status save failed", err, "tenant", tenant.Slug, "unit_id", item.ID)
			http.Redirect(w, r, "/app/settings/building?section=units&payment=error"+dialogTarget, http.StatusSeeOther)
			return
		}
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
	http.Redirect(w, r, "/app/settings/building?section=units&unit=saved", http.StatusSeeOther)
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
		http.Redirect(w, r, "/app/settings/building?section=units&unit=invalid", http.StatusSeeOther)
		return
	}
	if profile, exists, profileErr := a.energyFor(ac).Profile(tenant.Slug); profileErr != nil {
		http.Redirect(w, r, "/app/settings/building?section=units&unit=error", http.StatusSeeOther)
		return
	} else if exists && profile.HomeType == energy.HomeApartment {
		linkedID := normalizeUnitID(profile.UnitID)
		if linkedID == "" {
			if linked, ok := a.effectiveEnergyUnit(ac.tenantRef, profile); ok {
				linkedID = normalizeUnitID(linked.ID)
			}
		}
		if linkedID == deleteID {
			http.Redirect(w, r, "/app/settings/building?section=units&unit=home-linked#unit-"+deleteID, http.StatusSeeOther)
			return
		}
	}
	// Remove under one lock (HAUSV-145).
	removed, removedUnit, err := ac.repositories.units.DeleteUnit(deleteID)
	if err != nil {
		logError("unit delete failed", err, "tenant", tenant.Slug, "unit_id", deleteID)
		http.Redirect(w, r, "/app/settings/building?section=units&unit=error#unit-"+deleteID, http.StatusSeeOther)
		return
	}
	if !removed {
		http.Redirect(w, r, "/app/settings/building?section=units&unit=missing", http.StatusSeeOther)
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
	http.Redirect(w, r, "/app/settings/building?section=units&unit=deleted", http.StatusSeeOther)
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
		http.Redirect(w, r, "/app/settings/building?section=units&payment=invalid", http.StatusSeeOther)
		return
	}
	members := ac.repositories.units.MembersForUnit(unitID)
	if !members.Found {
		http.Redirect(w, r, "/app/settings/building?section=units&payment=missing", http.StatusSeeOther)
		return
	}
	record, err := ac.repositories.unitPayments.Set(unitPaymentStatus{
		TenantSlug: tenant.Slug,
		UnitID:     unitID,
		Status:     status,
		UpdatedBy:  actorEmail,
	})
	if err != nil {
		logError("unit payment status save failed", err, "tenant", tenant.Slug, "unit_id", unitID)
		http.Redirect(w, r, "/app/settings/building?section=units&payment=error#unit-"+unitID, http.StatusSeeOther)
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
	http.Redirect(w, r, "/app/settings/building?section=units&payment=saved", http.StatusSeeOther)
}

func (a *app) buildingSettingsContext(w http.ResponseWriter, ac authCtx) (tenantConfig, string, string, userProfile, bool) {
	if !ac.can(capabilityManageBuilding) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return tenantConfig{}, "", "", userProfile{}, false
	}
	return ac.tenant, ac.email, ac.role, a.profileForTenant(ac.email, ac.tenant.Slug), true
}

func tenantOverrideFromTenant(tenant tenantConfig) tenantOverride {
	return tenantOverride{
		MetaSet: true, Name: strings.TrimSpace(tenant.Name), Address: strings.TrimSpace(tenant.Address),
		MapSet: true, MapLatitude: tenant.MapLatitude, MapLongitude: tenant.MapLongitude, MapZoom: tenant.MapZoom,
		BrandIcon: normalizeTenantBrandIcon(tenant.BrandIcon), BrandAbbreviation: normalizeTenantBrandAbbreviation(tenant.BrandAbbreviation),
		ContactName: tenant.ContactName, ContactAddress: tenant.ContactAddress, ContactEmail: tenant.ContactEmail, ContactPhone: tenant.ContactPhone,
		EmergencyName: tenant.EmergencyName, EmergencyPhone: tenant.EmergencyPhone,
		CaretakerName: tenant.CaretakerName, CaretakerEmail: tenant.CaretakerEmail, CaretakerPhone: tenant.CaretakerPhone,
	}
}

func tenantBuildingOverrideFromForm(tenant tenantConfig, values url.Values) (tenantOverride, error) {
	override := tenantOverrideFromTenant(tenant)
	override.Name = strings.TrimSpace(values.Get("name"))
	override.Address = strings.TrimSpace(values.Get("address"))
	latitude, longitude, zoom, err := mapPositionFromForm(values.Get("map_position"))
	if err != nil {
		return tenantOverride{}, err
	}
	// The form always submits this field. An empty value deliberately stores a
	// zero position so an inherited/configured map can be removed as well.
	override.MapSet = values.Has("map_position")
	override.MapLatitude, override.MapLongitude, override.MapZoom = latitude, longitude, zoom
	if override.Name == "" || override.Address == "" || len([]rune(override.Name)) > 160 || len([]rune(override.Address)) > 500 {
		return tenantOverride{}, fmt.Errorf("invalid building metadata")
	}
	return override, nil
}

func tenantContactsOverrideFromForm(tenant tenantConfig, values url.Values) (tenantOverride, error) {
	override := tenantOverrideFromTenant(tenant)
	override.ContactName = strings.TrimSpace(values.Get("contact_name"))
	override.ContactAddress = strings.TrimSpace(values.Get("contact_address"))
	override.ContactEmail = normalizeEmail(values.Get("contact_email"))
	override.ContactPhone = strings.TrimSpace(values.Get("contact_phone"))
	override.EmergencyName = strings.TrimSpace(values.Get("emergency_name"))
	override.EmergencyPhone = strings.TrimSpace(values.Get("emergency_phone"))
	override.CaretakerName = strings.TrimSpace(values.Get("caretaker_name"))
	override.CaretakerEmail = normalizeEmail(values.Get("caretaker_email"))
	override.CaretakerPhone = strings.TrimSpace(values.Get("caretaker_phone"))
	if len([]rune(override.ContactName)) > 160 || len([]rune(override.ContactAddress)) > 500 || len([]rune(override.ContactPhone)) > 80 || len([]rune(override.EmergencyName)) > 160 || len([]rune(override.EmergencyPhone)) > 80 || len([]rune(override.CaretakerName)) > 160 || len([]rune(override.CaretakerPhone)) > 80 {
		return tenantOverride{}, fmt.Errorf("building contact too long")
	}
	for _, candidate := range []struct{ raw, normalized string }{{values.Get("contact_email"), override.ContactEmail}, {values.Get("caretaker_email"), override.CaretakerEmail}} {
		if strings.TrimSpace(candidate.raw) != "" {
			if _, err := mail.ParseAddress(candidate.raw); err != nil || candidate.normalized == "" {
				return tenantOverride{}, fmt.Errorf("invalid contact email")
			}
		}
	}
	return override, nil
}

func tenantAppearanceOverrideFromForm(tenant tenantConfig, values url.Values) (tenantOverride, error) {
	override := tenantOverrideFromTenant(tenant)
	override.BrandIcon = normalizeTenantBrandIcon(values.Get("brand_icon"))
	override.BrandAbbreviation = normalizeTenantBrandAbbreviation(values.Get("brand_abbreviation"))
	if override.BrandIcon == "" || override.BrandAbbreviation == "" {
		return tenantOverride{}, fmt.Errorf("invalid building appearance")
	}
	return override, nil
}

func tenantOverrideFromForm(values url.Values) (tenantOverride, error) {
	mapLatitude, mapLongitude, mapZoom, err := mapPositionFromForm(values.Get("map_position"))
	if err != nil {
		return tenantOverride{}, err
	}
	override := tenantOverride{
		MetaSet:           true,
		Name:              strings.TrimSpace(values.Get("name")),
		Address:           strings.TrimSpace(values.Get("address")),
		MapSet:            values.Has("map_position"),
		MapLatitude:       mapLatitude,
		MapLongitude:      mapLongitude,
		MapZoom:           mapZoom,
		BrandIcon:         normalizeTenantBrandIcon(values.Get("brand_icon")),
		BrandAbbreviation: normalizeTenantBrandAbbreviation(values.Get("brand_abbreviation")),
		ContactName:       strings.TrimSpace(values.Get("contact_name")),
		ContactAddress:    strings.TrimSpace(values.Get("contact_address")),
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
	if len([]rune(override.Name)) > 160 || len([]rune(override.Address)) > 500 || len([]rune(override.ContactName)) > 160 || len([]rune(override.ContactAddress)) > 500 || len([]rune(override.ContactPhone)) > 80 || len([]rune(override.EmergencyName)) > 160 || len([]rune(override.EmergencyPhone)) > 80 || len([]rune(override.CaretakerName)) > 160 || len([]rune(override.CaretakerPhone)) > 80 {
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

func mapPositionFromForm(raw string) (float64, float64, int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, 0, 0, nil
	}
	parts := strings.Split(raw, ",")
	if len(parts) != 2 {
		return 0, 0, 0, fmt.Errorf("map position must contain latitude and longitude")
	}
	latitude, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid map latitude")
	}
	longitude, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid map longitude")
	}
	if _, _, _, ok := tenantMapCoordinates(tenantConfig{MapLatitude: latitude, MapLongitude: longitude, MapZoom: defaultMapZoom}); !ok {
		return 0, 0, 0, fmt.Errorf("invalid map position")
	}
	return latitude, longitude, defaultMapZoom, nil
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
			OwnerSummary:       unitAssignmentSummary(item.OwnerEmails),
			RenterSummary:      unitAssignmentSummary(item.RenterEmails),
			HasOwners:          len(item.OwnerEmails) > 0,
			HasRenters:         len(item.RenterEmails) > 0,
			MembersLabel:       unitMembersLabel(len(item.OwnerEmails), len(item.RenterEmails)),
			DeleteConfirmLabel: "Einheit " + item.Label + " entfernen",
		})
	}
	return views
}

func unitAssignmentSummary(emails []string) string {
	if len(emails) == 0 {
		return "Nicht zugeordnet"
	}
	if len(emails) == 1 {
		return emails[0]
	}
	return emails[0] + " +" + strconv.Itoa(len(emails)-1)
}

func (a *app) buildingUnitViewsWithPayments(repositories requestRepositories, tenant store.TenantRef, units []unit) []buildingUnitView {
	tenantSlug := tenant.Slug
	views := buildingUnitViews(units)
	payments := unitPaymentStatusViewsForUnits(repositories.unitPayments, units)
	for i := range views {
		if i >= len(payments) {
			break
		}
		views[i].PaymentStatus = payments[i].Status
		views[i].PaymentStatusClass = payments[i].StatusClass
		views[i].PaymentDetail = payments[i].Detail
		views[i].PaymentUpdatedAt = payments[i].UpdatedAt
		views[i].PaymentHasUpdated = payments[i].HasUpdatedAt
		views[i].PaymentOptions = payments[i].StatusOptions
	}
	if profile, exists, err := a.energyStore.Profile(tenantSlug); err == nil && exists && profile.OnboardingComplete {
		if linked, ok := a.effectiveEnergyUnit(tenant, profile); ok {
			displayName := strings.TrimSpace(profile.HouseholdName)
			for i := range views {
				if displayName != "" && normalizeUnitID(views[i].ID) == normalizeUnitID(linked.ID) {
					views[i].HomeDisplayName = displayName
					views[i].HasHomeDisplayName = true
					break
				}
			}
		}
	}
	return views
}

func unitMembersLabel(ownerCount, renterCount int) string {
	parts := make([]string, 0, 2)
	if ownerCount > 0 {
		parts = append(parts, strconv.Itoa(ownerCount)+" Eigentümer")
	}
	if renterCount > 0 {
		parts = append(parts, strconv.Itoa(renterCount)+" Mieter")
	}
	if len(parts) == 0 {
		return "Noch keine Personen verknüpft"
	}
	return strings.Join(parts, " · ")
}

func unitPaymentStatusViewsForUnits(payments unitPaymentRepository, units []unit) []unitPaymentStatusView {
	statuses := map[string]unitPaymentStatus{}
	if payments != nil {
		for _, item := range payments.List() {
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

func unitPaymentStatusViewsForEmail(units unitRepository, payments unitPaymentRepository, email string) []unitPaymentStatusView {
	if units == nil || payments == nil {
		return nil
	}
	memberships := units.UnitsForEmail(email)
	views := make([]unitPaymentStatusView, 0, len(memberships))
	for _, membership := range memberships {
		record, hasRecord := payments.Get(membership.Unit.ID)
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
	case "home-linked":
		return "Diese Einheit gehört zu „Mein Zuhause“. Kennung und Art bleiben deshalb geschützt; auch Entfernen ist erst nach einer bewussten Neuordnung möglich.", false
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
	units := profileUnitViews(ac.repositories.units.UnitsForEmail(email))
	profileMsg, profileOK := profileSettingsMessage(r.URL.Query().Get("profile"))
	pageData := map[string]any{
		"Title":                  "Profil",
		"CanManageAnnouncements": canManageAnnouncements(ac.actor(), ac.resource()),
		"ActivePage":             "settings",
		"Profile":                profile,
		"ProfileMsg":             profileMsg,
		"ProfileOK":              profileOK,
		"PermissionList":         permissionLabelList(profile.Permissions),
		"AuthList":               authMethodsLabelList(profile.AuthMethods),
		"Units":                  units,
		"HasUnits":               len(units) > 0,
	}
	a.renderProfileSettingsTempl(w, r, ac, pageData)
}

func (a *app) updateProfileSettings(w http.ResponseWriter, r *http.Request, ac authCtx) {
	email, role, tenant := ac.email, ac.role, ac.tenant
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
			logError("profile save failed", err, "actor", redactedEmail(email))
			http.Redirect(w, r, "/app/settings/profile?profile=error", http.StatusSeeOther)
			return
		}
	}
	// The contact directory is rendered PER HOUSE, so the visibility choice is
	// recorded on this house's membership. Until someone saves it here it stays
	// unset and keeps inheriting the person-wide value, which is why existing
	// users see no change (HAUSV-178).
	if a.inviteStore != nil {
		if _, err := a.inviteStore.SetTenantDirectoryOptIn(email, tenant.Slug, overlay.DirectoryOptIn); err != nil {
			logError("directory visibility save failed", err, "actor", redactedEmail(email), "tenant", tenant.Slug)
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
	email, role := ac.email, ac.role
	modules := a.portalModulesFor(ac.tenant.Slug)
	if denyServiceProviderArea(w, role) {
		return
	}
	prefs := defaultNotificationPreferences()
	if a.notificationPrefs != nil {
		prefs = a.notificationPrefs.Get(email)
	}
	notifyMsg, notifyOK := notificationSettingsMessage(r.URL.Query().Get("notify"))
	events := notificationOptionsForModules(notificationEventOptions(prefs), modules)
	enabledCount := 0
	for _, event := range events {
		if event.Checked {
			enabledCount++
		}
	}
	pageData := map[string]any{
		"Title":                     "Benachrichtigungen",
		"ActivePage":                "settings",
		"NotifyMsg":                 notifyMsg,
		"NotifyOK":                  notifyOK,
		"EmailNotificationsEnabled": !prefs.Unsubscribed,
		"NotificationEvents":        events,
		"NotificationEnabledCount":  enabledCount,
		"NotificationEventCount":    len(events),
	}
	a.renderNotificationSettingsTempl(w, r, ac, pageData)
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
			logError("notification preference save failed", err, "actor", redactedEmail(email))
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
	users := a.userRows(ac.tenantRef)
	activeUsers, invitedUsers, deactivatedUsers := userStatusCounts(users)
	inviteMsg, inviteOK := inviteMessage(r.URL.Query().Get("invite"))
	pageData := map[string]any{
		"Title":             "Benutzer & Rechte",
		"Users":             users,
		"HasUsers":          len(users) > 0,
		"UserCount":         len(users),
		"ActiveUserCount":   activeUsers,
		"InvitedUserCount":  invitedUsers,
		"DisabledUserCount": deactivatedUsers,
		"UsersEmpty":        emptyState("Noch keine Zugänge", "Sobald eine Person eingeladen ist, erscheint sie hier mit Rolle und Zugangsstatus."),
		"InviteMsg":         inviteMsg,
		"InviteOK":          inviteOK,
		"ActivePage":        "users",
	}
	a.renderUserSettingsTempl(w, r, ac, pageData)
}

func userStatusCounts(users []userRow) (active int, invited int, deactivated int) {
	for _, user := range users {
		switch user.Status {
		case "Deaktiviert":
			deactivated++
		case "Eingeladen":
			invited++
		default:
			active++
		}
	}
	return active, invited, deactivated
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
	if !canAssignUserRole(ac.actor(), inviteRole, ac.resource()) {
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
		logError("invite persistence failed", err, "recipient", redactedEmail(inviteEmail), "tenant", tenant.Slug)
		a.redirectInvite(w, r, "error")
		return
	}
	if !added {
		a.redirectInvite(w, r, "exists")
		return
	}

	loginURL := a.publicBaseURL(r, tenant) + "/"
	if err := a.mailer.SendInvite(inviteEmail, loginURL, tenant.Address); err != nil {
		logWarn("invite email delivery failed",
			"tenant", tenant.Slug,
			"error_type", fmt.Sprintf("%T", err),
		)
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
	anchor := "access-list"
	switch status {
	case "invalid_email", "exists", "error":
		anchor = "invite"
	}
	http.Redirect(w, r, "/app/settings/users?invite="+url.QueryEscape(status)+"#"+anchor, http.StatusSeeOther)
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
	if normalizeRole(effectiveProfile.Role) == roleAdmin && !ac.can(capabilityPlatformAdmin) {
		a.redirectInvite(w, r, "not_editable")
		return
	}

	// Global identity (email, title, name) belongs to the PERSON, not to a house.
	// A house admin manages only their own membership; changing identity is a
	// separate, explicitly authorized platform-admin workflow (HAUSV-169 AC8).
	canEditIdentity := ac.can(capabilityPlatformAdmin)

	// Config-sourced users keep their configured email as a fixed identity; only
	// pure app invites may be renamed.
	newEmail := orig
	if !isEnv && canEditIdentity {
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
	if !canAssignUserRole(ac.actor(), newRole, ac.resource()) {
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
		remaining := a.adminEmails(ac.tenantRef)
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
	if canEditIdentity {
		updated.Title = strings.TrimSpace(r.FormValue("title"))
		updated.FirstName = strings.TrimSpace(r.FormValue("first_name"))
		updated.LastName = strings.TrimSpace(r.FormValue("last_name"))
	}
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
		// Role and permissions are HOUSE-scoped: they land on this tenant's
		// membership only. Writing the whole profile here is what let a manager of
		// house A change someone's role in house B (HAUSV-135).
		if _, found, err := a.inviteStore.SetTenantMembership(orig, tenant.Slug, newRole, updated.Permissions); err != nil {
			logError("membership update failed", err, "actor", redactedEmail(orig), "tenant", tenant.Slug)
			a.redirectInvite(w, r, "error")
			return
		} else if !found {
			a.redirectInvite(w, r, "not_editable")
			return
		}
		// Login-level fields stay global; identity fields move only for a
		// platform admin.
		if _, _, err := a.inviteStore.Mutate(orig, func(p *userProfile) {
			p.AuthMethods = updated.AuthMethods
			p.Deactivated = updated.Deactivated
			if canEditIdentity {
				p.Title = updated.Title
				p.FirstName = updated.FirstName
				p.LastName = updated.LastName
			}
		}); err != nil {
			a.redirectInvite(w, r, "error")
			return
		}
		// A rename is a global identity change, hence platform-admin only.
		if canEditIdentity && newEmail != orig {
			renamed, ok := a.inviteStore.Get(orig)
			if !ok {
				a.redirectInvite(w, r, "not_editable")
				return
			}
			renamed.Email = newEmail
			if _, err := a.inviteStore.Update(orig, renamed); err != nil {
				a.redirectInvite(w, r, "exists")
				return
			}
		}
	} else {
		// Adopting a config user: the new role applies to THIS house only, so it
		// goes in as a membership while other configured houses keep theirs.
		if updated.TenantMemberships == nil {
			updated.TenantMemberships = map[string]tenantMembership{}
		}
		updated.TenantMemberships[tenant.Slug] = tenantMembership{Role: newRole, Permissions: updated.Permissions}
		updated.Role = normalizeRole(envProfile.Role)
		added, err := a.inviteStore.Add(updated)
		if err != nil {
			logError("adopt persistence failed", err, "actor", redactedEmail(orig), "tenant", tenant.Slug)
			a.redirectInvite(w, r, "error")
			return
		}
		if !added {
			// Raced with a concurrent adopt; apply the house-scoped change instead.
			if _, _, err := a.inviteStore.SetTenantMembership(orig, tenant.Slug, newRole, updated.Permissions); err != nil {
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
	if normalizeRole(effectiveProfile.Role) == roleAdmin && !ac.can(capabilityPlatformAdmin) {
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
		remaining := a.adminEmails(ac.tenantRef)
		delete(remaining, deleteEmail)
		if len(remaining) == 0 {
			a.redirectInvite(w, r, "last_admin")
			return
		}
	}
	// House-scoped removal: detach this person from THIS house. The person and
	// any other house they belong to survive; the record only disappears once no
	// house is left, which keeps single-house behaviour identical (HAUSV-135).
	removedProfile, found, err := a.inviteStore.RemoveTenant(deleteEmail, tenant.Slug)
	if err != nil {
		a.redirectInvite(w, r, "error")
		return
	}
	if !found {
		a.redirectInvite(w, r, "not_editable")
		return
	}
	_ = removedProfile
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
	if tenant, ok := data["Tenant"].(tenantConfig); ok {
		data["TenantBrandLucideSVG"] = tenantBrandLucideSVG(tenant.BrandIcon)
	}
	if _, ok := data["ReleaseNotes"]; !ok {
		notes := version.Notes()
		data["ReleaseNotes"] = notes
		data["HasReleaseNotes"] = len(notes) > 0
	}
	enrichCapabilityData(data)
	// These two read from the announcement and issue stores. They are the reason
	// render() cannot itself live in a pure rendering package.
	a.enrichUnreadAnnouncementData(data, nil, nil)
	a.enrichIssueData(data)
	a.executeTemplate(w, name, data)
}

// baseContext centralises the identity and navigation data shared by every
// authenticated page. Page-specific data is layered on top by withBase so
// intentional overrides remain visible at the call site.
func (a *app) baseContext(ac authCtx) map[string]any {
	profile := a.profileForTenant(ac.email, ac.tenant.Slug)
	modules := a.portalModulesFor(ac.tenant.Slug)
	isAdmin := ac.can(capabilityPlatformAdmin)
	canViewEnergy := modules.Energy && a.canViewEnergy(ac)
	portalContexts := a.portalContextsFor(ac.email, ac.tenant.Slug, ac.role)
	return map[string]any{
		"Tenant":                 ac.tenant,
		"HouseName":              houseDisplayName(ac.tenant),
		"SidebarAddress":         sidebarAddressForTenant(ac.tenant),
		"MapURL":                 tenantMapURL(ac.tenant.Address),
		"SidebarMap":             sidebarMapForTenant(ac.tenant),
		"Email":                  ac.email,
		"Role":                   ac.role,
		"PortalContexts":         portalContexts,
		"CanSwitchPortalContext": len(portalContexts) > 1,
		"IsAdmin":                isAdmin,
		"DisplayName":            profile.DisplayName(),
		"Initials":               profile.Initials(),
		"PortalModules":          modules,
		"CanSeeParking":          modules.Parking && (isAdmin || profile.HasPermission(permissionParking)),
		"CanViewEnergy":          canViewEnergy,
		"CanManageEnergy":        modules.Energy && a.canManageEnergy(ac),
		"CanManageHomeIdentity":  modules.Energy && a.canManageHomeIdentity(ac),
		"CanControlEnergy":       modules.Energy && a.canControlEnergy(ac),
		"HomeIdentity":           a.homeIdentityForActor(ac, canViewEnergy),
	}
}

func (a *app) homeIdentityForActor(ac authCtx, canViewEnergy bool) homeIdentityView {
	identity := defaultHomeIdentityView()
	if !canViewEnergy || a.energyStore == nil {
		return identity
	}
	profile, exists, err := a.energyFor(ac).Profile(ac.tenant.Slug)
	if err != nil || !exists || (!profile.OnboardingComplete && profile.OnboardingStep < 3) {
		return identity
	}
	return a.homeIdentityFromProfile(ac.tenantRef, profile)
}

func defaultHomeIdentityView() homeIdentityView {
	return homeIdentityView{
		DisplayName: "Mein Zuhause",
		AriaLabel:   "Mein Zuhause",
	}
}

func (a *app) homeIdentityFromProfile(tenant store.TenantRef, profile energy.HomeProfile) homeIdentityView {
	identity := defaultHomeIdentityView()
	displayName := strings.TrimSpace(profile.HouseholdName)
	if displayName == "" {
		return identity
	}
	identity.DisplayName = displayName
	identity.AriaLabel = displayName
	identity.HasDisplayName = true
	if unitLabel, ok := a.energyHomeUnitLabel(tenant, profile); ok {
		identity.UnitLabel = unitLabel
		identity.HasUnit = true
		identity.AriaLabel = displayName + ", offizielle Einheit " + unitLabel
	}
	return identity
}

func (a *app) withBase(ac authCtx, pageData map[string]any) map[string]any {
	data := a.baseContext(ac)
	for key, value := range pageData {
		data[key] = value
	}
	a.enrichUnreadAnnouncementData(data, ac.repositories.announcements, ac.repositories.announcementReads)
	return data
}

// executeTemplate is the pure rendering step: no store access, no business
// logic — just template + data → HTML. This is the piece that becomes
// internal/web.Renderer.
func (a *app) executeTemplate(w http.ResponseWriter, name string, data map[string]any) {
	viewData := data
	tenantSlug := ""
	if tenant, ok := data["Tenant"].(tenantConfig); ok {
		tenantSlug = tenant.Slug
		viewData = make(map[string]any, len(data))
		for key, value := range data {
			viewData[key] = value
		}
		viewData["Tenant"] = tenantTemplateViewFrom(tenant)
	}
	var rendered bytes.Buffer
	if err := a.templates.ExecuteTemplate(&rendered, name, viewData); err != nil {
		logError("template render failed", err, "template", name)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	body := rendered.String()
	if tenantSlug != "" {
		body = prefixTenantHTMLPaths(body, tenantSlug)
	}
	_, _ = io.WriteString(w, body)
}

func prefixTenantHTMLPaths(body string, tenantSlug string) string {
	prefix := "/" + normalizeSlug(tenantSlug)
	if prefix == "/" {
		return body
	}
	for _, attribute := range []string{"href", "action", "formaction", "src", "data-src", "data-glass-src", "data-map-tile", "value"} {
		needle := attribute + `="/`
		body = strings.ReplaceAll(body, needle+tenantSlug+"/", "\x00HAUSV_TENANT_PATH\x00")
		body = strings.ReplaceAll(body, needle, attribute+`="`+prefix+"/")
		body = strings.ReplaceAll(body, "\x00HAUSV_TENANT_PATH\x00", needle+tenantSlug+"/")
	}
	body = strings.ReplaceAll(body, "url(/", "url("+prefix+"/")
	body = strings.ReplaceAll(body, "url('/", "url('"+prefix+"/")
	body = strings.ReplaceAll(body, `url("/`, `url("`+prefix+`/`)
	return body
}

// tenantTemplateView keeps the configuration model free of html/template
// types while allowing the server-owned hero route to remain an absolute URL
// inside CSS url() values. html/template otherwise percent-encodes its slashes,
// turning it into a different request path. Only same-origin server-owned
// routes are trusted; other configured URLs retain the default escaping.
type tenantTemplateView struct {
	tenantConfig
	HeroImageURL any
}

func tenantTemplateViewFrom(tenant tenantConfig) tenantTemplateView {
	heroURL := any(tenant.HeroImageURL)
	if tenantHeroURLIsServerOwned(tenant) {
		heroURL = template.URL(tenant.HeroImageURL)
	}
	return tenantTemplateView{
		tenantConfig: tenant,
		HeroImageURL: heroURL,
	}
}

func tenantHeroURLIsServerOwned(tenant tenantConfig) bool {
	raw := strings.TrimSpace(tenant.HeroImageURL)
	if raw == "/tenant-hero/"+normalizeSlug(tenant.Slug) {
		return true
	}
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Scheme != "" || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != raw {
		return false
	}
	return strings.HasPrefix(parsed.Path, "/assets/") && path.Clean(parsed.Path) == parsed.Path
}

func enrichCapabilityData(data map[string]any) {
	role, _ := data["Role"].(string)
	if role == "" {
		return
	}
	tenant, _ := data["Tenant"].(tenantConfig)
	person, _ := data["Email"].(string)
	actor := actorFor(person, tenant.Slug, role)
	resource := resourceFor(tenant.Slug)
	if _, ok := data["IsAdmin"]; !ok {
		data["IsAdmin"] = can(actor, capabilityPlatformAdmin, resource)
	}
	if _, ok := data["IsServiceProvider"]; !ok {
		data["IsServiceProvider"] = isServiceProviderRole(role)
	}
	if _, ok := data["CanUseResidentAreas"]; !ok {
		data["CanUseResidentAreas"] = roleCanUseResidentAreas(role)
	}
	if _, ok := data["CanManageUsers"]; !ok {
		data["CanManageUsers"] = can(actor, capabilityManageUsers, resource)
	}
	if _, ok := data["CanManageAnnouncements"]; !ok {
		data["CanManageAnnouncements"] = can(actor, capabilityManageAnnouncements, resource)
	}
	if _, ok := data["CanManageIssues"]; !ok {
		data["CanManageIssues"] = can(actor, capabilityManageIssues, resource)
	}
	if _, ok := data["CanManageDocuments"]; !ok {
		data["CanManageDocuments"] = can(actor, capabilityManageDocuments, resource)
	}
	if _, ok := data["CanManageBuilding"]; !ok {
		data["CanManageBuilding"] = can(actor, capabilityManageBuilding, resource)
	}
	if _, ok := data["CanManageHandovers"]; !ok {
		data["CanManageHandovers"] = canManageHandovers(actor, resource)
	}
	if _, ok := data["CanViewAudit"]; !ok {
		data["CanViewAudit"] = canViewAudit(actor, resource)
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
	identity, identityOK := a.tenantIdentity(tenant.Slug)
	if !identityOK {
		data["OpenIssues"] = 0
		data["HasOpenIssues"] = false
		return
	}
	count := issueOpenCount(a.visibleIssuesForActor(identity.Ref(), email, role))
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
	tenant, ok := a.tenantBySlug(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	filename := ""
	directory := a.tenantHeroDir
	if override, ok := a.tenantOverrides.Get(slug); ok {
		filename = override.HeroImage
	}
	if filename == "" {
		filename = tenant.HeroImageFile
		directory = a.tenantHeroSeedDir
	}
	filename = filepath.Base(strings.TrimSpace(filename))
	if directory == "" || filename == "" || filename == "." {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(directory, filename))
}

func (a *app) tenantForRequest(r *http.Request) tenantConfig {
	if r != nil {
		if resolved, ok := resolvedTenantFromContext(r.Context()); ok {
			return resolved.tenant
		}
		// Direct handler unit tests and non-routed helper calls predate the
		// middleware stack. This compatibility path does not mint repositories;
		// authenticated handlers still require resolvedTenantRequest above.
		path := strings.TrimPrefix(r.URL.Path, "/")
		if first, _, _ := strings.Cut(path, "/"); first != "" {
			if tenant, ok := a.tenantBySlug(first); ok {
				return tenant
			}
		}
	}
	tenant, _ := a.tenantBySlug(a.defaultTenant)
	return tenant
}

func (a *app) isMarketingHost(r *http.Request) bool {
	if a == nil || r == nil {
		return false
	}
	if resolved, ok := resolvedTenantFromContext(r.Context()); ok && resolved.pathPrefixed {
		return false
	}
	if _, ok := resolvedTenantFromContext(r.Context()); !ok {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if first, _, _ := strings.Cut(path, "/"); first != "" {
			if _, found := a.tenantBySlug(first); found {
				return false
			}
		}
	}
	host := normalizeHost(r.Host)
	root := normalizeHost(a.rootDomain)
	if host == "" || root == "" {
		return false
	}
	return host == root || host == "www."+root
}

func (a *app) tenantBySlug(slug string) (tenantConfig, bool) {
	slug = normalizeSlug(slug)
	if tenant, ok := a.tenants[slug]; ok {
		return a.withTenantOverride(tenant), true
	}
	if a.homePortals == nil {
		return tenantConfig{}, false
	}
	portal, ok, err := a.homePortals.Get(slug)
	if err != nil || !ok {
		return tenantConfig{}, false
	}
	return a.withTenantOverride(tenantConfig{
		Slug:       portal.Slug,
		Name:       portal.HouseholdName,
		Address:    portal.HouseholdName,
		PortalType: config.PortalTypeHouse,
		BrandIcon:  tenantBrandSingleHome,
	}), true
}

func (a *app) withTenantOverride(tenant tenantConfig) tenantConfig {
	if strings.TrimSpace(tenant.HeroImageURL) == "" {
		tenant.HeroImageURL = defaultTenantHeroImageURL
	}
	if tenant.HeroImageFile != "" && a.tenantHeroSeedDir != "" {
		tenant.HeroImageURL = "/tenant-hero/" + tenant.Slug
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
		if override.MapSet {
			tenant.MapLatitude = override.MapLatitude
			tenant.MapLongitude = override.MapLongitude
			tenant.MapZoom = override.MapZoom
		}
		if override.BrandIcon != "" {
			tenant.BrandIcon = override.BrandIcon
		}
		tenant.BrandAbbreviation = firstNonEmpty(override.BrandAbbreviation, tenant.BrandAbbreviation)
		tenant.ContactName = override.ContactName
		tenant.ContactAddress = override.ContactAddress
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
	return strings.TrimRight(a.baseURL, "/") + "/" + tenant.Slug
}

func (a *app) currentUser(r *http.Request) (string, string, string, bool) {
	c, err := r.Cookie("weg_session")
	if err != nil {
		return "", "", "", false
	}
	session, ok := a.sessions.GetSession(c.Value)
	if !ok {
		return "", "", "", false
	}
	if !a.isAllowed(session.Email, session.TenantSlug) || !a.isAuthMethodAllowed(session.Email, session.TenantSlug, session.AuthMethod) {
		return "", "", "", false
	}
	role := a.roleFor(session.Email, session.TenantSlug)
	if session.Role != "" {
		if !a.ownPortalContextAllowed(session.Email, session.TenantSlug, session.Role, session.AuthMethod) {
			return "", "", "", false
		}
		role = session.Role
	}
	return session.Email, role, session.TenantSlug, true
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
	// A self-service home activation is a separate, verified authorization
	// boundary. It may add exactly that home's confirmed owner to an existing
	// env-backed identity, but must not let an arbitrary persisted profile
	// override env roles or memberships (HAUSV-478).
	if a.isConfirmedHomePortalOwner(email, tenantSlug) {
		if profile, ok := a.directoryProfile(email); ok && profile.Deactivated {
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
	confirmedHomeOwner := a.isConfirmedHomePortalOwner(email, tenantSlug)
	if !ok {
		return confirmedHomeOwner && userProfile{AuthMethods: defaultAuthMethods()}.AllowsAuthMethod(authMethod)
	}
	if !profile.HasTenant(tenantSlug) && !confirmedHomeOwner {
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

// isConfirmedHomePortalOwner trusts only the owner recorded by the atomic
// self-service activation transaction. A plain invite/profile record is not
// sufficient, preserving the anti-escalation boundary for env-backed users.
func (a *app) isConfirmedHomePortalOwner(email string, tenantSlug string) bool {
	if a == nil || a.homePortals == nil {
		return false
	}
	email = normalizeEmail(email)
	tenantSlug = normalizeSlug(tenantSlug)
	if email == "" || tenantSlug == "" {
		return false
	}
	portal, found, err := a.homePortals.Get(tenantSlug)
	return err == nil && found && normalizeEmail(portal.OwnerEmail) == email
}

func (a *app) emailLoginAvailable() bool {
	return a.mailer.Configured() || a.localDevLogin || a.demoLogin
}

func (a *app) roleFor(email string, tenantSlug string) string {
	if profile, ok := a.directoryProfile(email); ok && profile.HasTenant(tenantSlug) {
		profile = profile.ForTenant(tenantSlug)
		if profile.Role != "" {
			return normalizeRole(profile.Role)
		}
	}
	if _, ok := a.admins[email]; ok {
		return roleAdmin
	}
	if a.isConfirmedHomePortalOwner(email, tenantSlug) {
		return roleOwner
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
		if !profile.HasTenant(tenantSlug) && a.isConfirmedHomePortalOwner(email, tenantSlug) {
			if _, isBreakGlass := a.admins[email]; !isBreakGlass {
				profile.Role = roleOwner
				profile.Status = "Aktiv"
				profile.Tenants = append(profile.Tenants, tenantSlug)
				profile.Permissions = nil
			}
		}
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

func (a *app) userRows(tenant store.TenantRef) []userRow {
	tenantSlug := tenant.Slug
	seen := map[string]struct{}{}
	rows := make([]userRow, 0, len(a.profiles)+len(a.admins)+len(a.allowed))
	units, _ := store.BindUnitRepository(a.unitStore, tenant)
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
		// A directory profile is authoritative for house membership. Do not
		// synthesize a row for the active house merely because the same address is
		// also present in a process-wide bootstrap allowlist.
		if _, known := a.directoryProfile(email); known {
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
		if _, known := a.directoryProfile(email); known {
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
		if units != nil {
			for _, membership := range units.UnitsForEmail(rows[i].Email) {
				label := strings.TrimSpace(membership.Unit.Label)
				if relation := unitPaymentRelationLabel(membership.Relation); relation != "" {
					label += " · " + relation
				}
				rows[i].UnitList = append(rows[i].UnitList, label)
			}
			rows[i].HasUnits = len(rows[i].UnitList) > 0
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if userStatusSortRank(rows[i].Status) != userStatusSortRank(rows[j].Status) {
			return userStatusSortRank(rows[i].Status) < userStatusSortRank(rows[j].Status)
		}
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

func userStatusSortRank(status string) int {
	switch status {
	case "Deaktiviert":
		return 0
	case "Eingeladen":
		return 1
	default:
		return 2
	}
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
	meta.DisabledModules = append([]string(nil), current.DisabledModules...)
	meta.UpdatedAt = time.Now().UTC()
	s.data.Tenants[tenantSlug] = meta
	return s.saveLocked()
}

func (s *tenantOverrideStore) SetDisabledModules(tenantSlug string, disabled []string) error {
	if s == nil {
		return nil
	}
	tenantSlug = normalizeSlug(tenantSlug)
	if tenantSlug == "" {
		return fmt.Errorf("invalid tenant")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Tenants == nil {
		s.data.Tenants = map[string]tenantOverride{}
	}
	override := normalizeTenantOverride(s.data.Tenants[tenantSlug])
	override.DisabledModules = normalizeDisabledPortalModules(disabled)
	override.UpdatedAt = time.Now().UTC()
	s.data.Tenants[tenantSlug] = override
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
	if override.MapSet {
		if _, _, _, ok := tenantMapCoordinates(tenantConfig{MapLatitude: override.MapLatitude, MapLongitude: override.MapLongitude, MapZoom: override.MapZoom}); !ok {
			override.MapLatitude = 0
			override.MapLongitude = 0
			override.MapZoom = 0
		}
	}
	override.BrandIcon = normalizeTenantBrandIcon(override.BrandIcon)
	override.BrandAbbreviation = normalizeTenantBrandAbbreviation(override.BrandAbbreviation)
	override.ContactName = strings.TrimSpace(override.ContactName)
	override.ContactAddress = strings.TrimSpace(override.ContactAddress)
	override.ContactEmail = normalizeEmail(override.ContactEmail)
	override.ContactPhone = strings.TrimSpace(override.ContactPhone)
	override.EmergencyName = strings.TrimSpace(override.EmergencyName)
	override.EmergencyPhone = strings.TrimSpace(override.EmergencyPhone)
	override.CaretakerName = strings.TrimSpace(override.CaretakerName)
	override.CaretakerEmail = normalizeEmail(override.CaretakerEmail)
	override.CaretakerPhone = strings.TrimSpace(override.CaretakerPhone)
	override.HeroImage = filepath.Base(strings.TrimSpace(override.HeroImage))
	override.DisabledModules = normalizeDisabledPortalModules(override.DisabledModules)
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

func storeEBInterfaceInvoiceDocument(storage documentStorage, tenant store.TenantRef, invoice integrations.Invoice, uploadedBy string, data []byte, now time.Time) (documentRecord, error) {
	documents, ok := store.BindDocumentRepository(storage, tenant)
	if !ok {
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
	return documents.CreateGenerated(documentRecord{
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
		Email:                  p.Email,
		Title:                  p.Title,
		FirstName:              p.FirstName,
		LastName:               p.LastName,
		Phone:                  p.Phone,
		DirectoryOptIn:         p.DirectoryOptIn,
		DisplayName:            p.DisplayName(),
		Initials:               p.Initials(),
		Role:                   p.Role,
		RoleClass:              roleClass(p.Role),
		RoleCapabilities:       roleCapabilityLabels(p.Role),
		Status:                 p.Status,
		Tenants:                strings.Join(p.Tenants, ", "),
		PermissionLabel:        permissionLabel(p.Permissions),
		PermissionList:         permissionLabelList(p.Permissions),
		ParkingChecked:         p.HasPermission(permissionParking),
		EnergyCaretakerChecked: p.HasPermission(permissionEnergyCaretaker),
		AuthLabel:              authMethodsLabel(p.AuthMethods),
		AuthList:               authMethodsLabelList(p.AuthMethods),
		EmailAuthChecked:       p.AllowsAuthMethod(authMethodEmail),
		OIDCAuthChecked:        p.AllowsAuthMethod(authMethodOIDC),
		Deactivated:            p.Deactivated,
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
		permissionParking:         {},
		permissionEnergyView:      {},
		permissionEnergyConfigure: {},
		permissionEnergyControl:   {},
		permissionEnergyCaretaker: {},
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
func (a *app) adminEmails(tenant store.TenantRef) map[string]struct{} {
	admins := map[string]struct{}{}
	for _, row := range a.userRows(tenant) {
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

// Close releases process-lifetime resources. Login mail receives a bounded
// drain/cancellation window before SQLite closes; JSON stores hold no OS
// handles between writes. Tenant lanes go before the process pool, since they
// are the ones holding server connections.
func (a *app) Close() error {
	a.closeMagicLinkDelivery()
	var firstErr error
	if a.scopedDB != nil {
		firstErr = a.scopedDB.Close()
	}
	if a.pool != nil {
		if err := a.pool.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// StartEnergyRetentionWorker enforces the published maximum energy-data
// retention while a process remains up between deployments.
func (a *app) StartEnergyRetentionWorker() func() { return a.startEnergyRetentionWorker() }

// RunHealthcheck is the container health probe.
func RunHealthcheck(target string) error { return runHealthcheck(target) }

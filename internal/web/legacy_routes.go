package web

import "github.com/inspr-at/hausv-org/internal/view"

type PaymentImportUnitView struct {
	Label     string
	Reference string
}

type PaymentImportRowView struct {
	Decision      string
	DecisionClass string
	Reference     string
	UnitLabel     string
	Status        string
	Amount        string
	Reason        string
}

type PaymentImportPreviewView struct {
	Token          string
	Filename       string
	SourceVersion  string
	CreatedAt      string
	Assigned       int
	Unclear        int
	Rejected       int
	Rows           []PaymentImportRowView
	CanApply       bool
	AlreadyApplied bool
	Changed        bool
}

type EbInterfaceImportPreviewView struct {
	Token         string
	Filename      string
	SourceVersion string
	CreatedAt     string
	InvoiceNumber string
	IssuerName    string
	RecipientName string
	Amount        string
	IssueDate     string
	DueDate       string
	ServicePeriod string
	ErrorLabels   []string
	CanStore      bool
	AlreadyStored bool
}

type StructuredExportSourceView struct {
	Value       string
	Title       string
	Description string
	Count       int
}

type StructuredExportPreviewView struct {
	Token       string
	CreatedAt   string
	Sources     []StructuredExportSourceView
	Accepted    int
	Rejected    int
	CSVChecksum string
	Filename    string
}

type EnergyDataImportView struct {
	Filename string
	Date     string
	Format   string
	Size     string
}

type PortalModuleOptionView struct {
	ID          string
	Label       string
	Description string
	Icon        string
	Enabled     bool
}

type IssueResidentDetailPageData struct {
	Portal       PortalPageData
	Issue        view.IssueView
	IssueCreated bool
}

type ParkingSettingsPageData struct {
	Portal                 PortalPageData
	Accounting             view.ParkingAccountingView
	SettingsMsg            string
	SettingsOK             bool
	ChargingMsg            string
	ChargingOK             bool
	Charging               view.ParkingChargingAdminView
	ParkingSection         string
	SectionAccounting      bool
	SectionCharging        bool
	SectionTelegram        bool
	CanManageParkingConfig bool
	CanManageUsers         bool
}

type ParkingMonthPageData struct {
	Portal                   PortalPageData
	Detail                   view.ParkingMonthDetailView
	ParkingMsg               string
	ParkingOK                bool
	CanManageParkingPayments bool
	CanMarkParkingPayment    bool
}

type ParkingAccessSettingsPageData struct {
	Portal                       PortalPageData
	ParkingSection               string
	CanManageParkingConfig       bool
	CanManageUsers               bool
	AccessRows                   []view.UserRow
	HasAccessRows                bool
	AccessRowsEmpty              view.EmptyStateView
	AccessMsg                    string
	AccessOK                     bool
	ServiceProviderAccessEnabled bool
	IsAdmin                      bool
}

type PortalModuleSettingsPageData struct {
	Portal              PortalPageData
	HouseName           string
	PortalModuleOptions []PortalModuleOptionView
	PortalModulesSaved  bool
	PortalModulesError  bool
}

type StructuredExportPageData struct {
	Portal                  PortalPageData
	StructuredExportSources []StructuredExportSourceView
	StructuredExportPreview *StructuredExportPreviewView
	StructuredExportMsg     string
	StructuredExportOK      bool
}

type EnergyDataPageData struct {
	Portal            PortalPageData
	Imports           []EnergyDataImportView
	HasImports        bool
	ImportCount       int
	IntervalCount     int
	AssessmentCount   int
	AssetCount        int
	MappingCount      int
	CanControlEnergy  bool
	IsActiveMode      bool
	IsShadowMode      bool
	HistoryDeleted    bool
	ExportUnavailable bool
}

type EbInterfaceImportPageData struct {
	Portal                   PortalPageData
	EBInterfacePreview       *EbInterfaceImportPreviewView
	EBInterfaceImportMsg     string
	EBInterfaceImportOK      bool
	MaxEBInterfaceImportSize string
}

type PaymentImportPageData struct {
	Portal                  PortalPageData
	PaymentImportPeriod     string
	PaymentImportReferences []PaymentImportUnitView
	HasPaymentImportUnits   bool
	PaymentImportPreview    *PaymentImportPreviewView
	PaymentImportMsg        string
	PaymentImportOK         bool
	MaxCAMTImportSize       string
}

// legacyRouteChoice keeps conditional attribute text escaped by templ.
func legacyRouteChoice(condition bool, yes, no string) string {
	if condition {
		return yes
	}
	return no
}

// legacyPaidLabel and legacyPaymentLabel render the settled state as single
// text nodes, exactly like the retired template did ("Bezahlt am 03.09.2026",
// "Überweisung · QA-ZAHLUNG"), so screen readers and oracles see one phrase.
func legacyPaidLabel(paidAt string) string {
	if paidAt == "" {
		return "Bezahlt"
	}
	return "Bezahlt am " + paidAt
}

func legacyPaymentLabel(method, reference string) string {
	if reference == "" {
		return method
	}
	return method + " · " + reference
}

package store

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/indexation"
	"github.com/inspr-at/hausv-org/internal/textutil"
	"github.com/inspr-at/hausv-org/internal/ulid"
)

const (
	LeaseStatusDraft  = "draft"
	LeaseStatusActive = "active"
	LeaseStatusEnded  = "ended"

	LeaseKindHauptmiete = "hauptmiete"
	LeaseKindUntermiete = "untermiete"

	UseKindWohnung   = "wohnung"
	UseKindGeschaeft = "geschaeft"
	UseKindGarage    = "garage"
	UseKindSonstiges = "sonstiges"

	MRGVoll     = "voll"
	MRGTeil     = "teil"
	MRGAusnahme = "ausnahme"
	MRGWGG      = "wgg"

	RentRegimeRichtwert  = "richtwert"
	RentRegimeKategorie  = "kategorie"
	RentRegimeAngemessen = "angemessen"
	RentRegimeFrei       = "frei"
	RentRegimeSonstig    = "sonstig"

	PartyHauptmieter = "hauptmieter"
	PartyMitmieter   = "mitmieter"

	ComponentHMZ        = "hmz"
	ComponentBKAkonto   = "bk_akonto"
	ComponentHeizAkonto = "heiz_akonto"
	ComponentLift       = "lift"
	ComponentMoebel     = "moebel"
	ComponentStellplatz = "stellplatz"
	ComponentSonstiges  = "sonstiges"

	ClauseMieWeG       = "mieweg_model"
	ClauseVPIThreshold = "vpi_threshold"
	ClauseVPIPeriodic  = "vpi_periodic"
	ClauseStaffel      = "staffel"
	ClauseNone         = "none"

	ReviewUnreviewed = "unreviewed"
	ReviewOK         = "ok"
	ReviewDoubtful   = "doubtful"
	ReviewInvalid    = "invalid"

	OriginManual = "manual"
	OriginImport = "import"

	openEnded = "9999-12-31"
)

var (
	ErrLeaseOverlap  = errors.New("overlapping hauptmiete")
	ErrLeaseNotFound = errors.New("lease not found")
	ErrUnitNotFound  = errors.New("unit not found")
	ErrLeaseInvalid  = errors.New("invalid lease")
)

// Lease is one contract for a unit. At most one Hauptmiete may cover a date.
type Lease struct {
	ID                 string
	UnitID             string
	Status             string
	ConcludedOn        string
	StartsOn           string
	EndsOn             string
	LeaseKind          string
	UseKind            string
	MRGScope           string
	RentRegime         string
	PriceRestricted    bool
	PriceRestrictedSet bool
	MaxHMZCents        *int64
	LandlordIsBusiness bool
	TenantIsConsumer   bool
	ZinsterminDay      int
	WirksamwerdenMode  string // Empty inherits the organisation default.
	VATOpted           bool
	Notes              string
	UpdatedAt          time.Time
	UpdatedBy          string
	Parties            []LeaseParty
	Components         []RentComponent
	Clauses            []IndexClause
}

type LeaseParty struct {
	ID, LeaseID, Name, Address, Email, Role, ValidFrom, ValidTo string
}

type RentComponent struct {
	ID, LeaseID, Kind, ValidFrom, Origin string
	NetCents                             int64
	VATRateBP                            int
	CreatedAt                            time.Time
}

type IndexClause struct {
	StaffelSteps                           []indexation.StaffelStep
	ID, LeaseID, ComponentKind, ClauseType string
	Series, BasePeriod, BaseValue          string
	ThresholdKind, ThresholdValue          string
	ThresholdInclusive                     bool
	FullChangeOnTrigger                    bool
	TwoWay                                 bool
	PctRounding                            string
	PeriodicMonth                          int
	ReferenceMonthOffset                   *int
	ClauseText, ReviewStatus, ReviewNote   string
	ValidFrom                              string
	State                                  *ValorisationState
}

// ValorisationState is derived. Slice 2a only seeds it from an import.
type ValorisationState struct {
	ClauseID                                             string
	ContractValue, ContractBasePeriod, ContractBaseValue string
	CapValue, CapAnchorPeriod                            string
	CapLastYear                                          int
	LastEffectiveOn, LastRunItemID                       string
}

// LeaseClassification is the display answer for the unit tab. It does not
// compute a new rent.
type LeaseClassification struct {
	MieWeG       bool
	MieWeGReason string
	SpecialCap   bool
	CapReason    string
}

type LeaseRepository interface {
	Create(Lease) (Lease, error)
	Update(Lease) (Lease, error)
	End(id, endsOn, actor string) (Lease, error)
	Get(id string) (Lease, bool, error)
	List() ([]Lease, error)
	History(unitID string) ([]Lease, error)
	ReplaceParties(leaseID string, parties []LeaseParty) error
	AddRentComponent(leaseID string, component RentComponent) (RentComponent, error)
	SetClause(clause IndexClause) (IndexClause, error)
	ReviewClause(id, status, note string) (IndexClause, error)
	SetValorisation(state ValorisationState) error
	Import(drafts []LeaseImportDraft, units []Unit, commit bool, actor string) (LeaseImportReport, error)
}

type LeaseStorage interface{ leaseStorage() }

type leaseBackend interface {
	createLease(TenantRef, Lease) (Lease, error)
	updateLease(TenantRef, Lease) (Lease, error)
	endLease(TenantRef, string, string, string) (Lease, error)
	getLease(TenantRef, string) (Lease, bool, error)
	listLeases(TenantRef) ([]Lease, error)
	leaseHistory(TenantRef, string) ([]Lease, error)
	replaceLeaseParties(TenantRef, string, []LeaseParty) error
	addRentComponent(TenantRef, string, RentComponent) (RentComponent, error)
	setLeaseClause(TenantRef, IndexClause) (IndexClause, error)
	reviewLeaseClause(TenantRef, string, string, string) (IndexClause, error)
	setValorisation(TenantRef, ValorisationState) error
	importLeases(TenantRef, []LeaseImportDraft, []Unit, bool, string) (LeaseImportReport, error)
}

type boundLeaseRepository struct {
	storage leaseBackend
	tenant  TenantRef
}

func BindLeaseRepository(storage LeaseStorage, tenant TenantRef) (LeaseRepository, bool) {
	backend, ok := storage.(leaseBackend)
	resolved, tenantOK := validTenantRef(tenant)
	if !ok || !tenantOK {
		return nil, false
	}
	return &boundLeaseRepository{storage: backend, tenant: resolved}, true
}

func (r *boundLeaseRepository) Create(lease Lease) (Lease, error) {
	return r.storage.createLease(r.tenant, lease)
}
func (r *boundLeaseRepository) Update(lease Lease) (Lease, error) {
	return r.storage.updateLease(r.tenant, lease)
}
func (r *boundLeaseRepository) End(id, endsOn, actor string) (Lease, error) {
	return r.storage.endLease(r.tenant, id, endsOn, actor)
}
func (r *boundLeaseRepository) Get(id string) (Lease, bool, error) {
	return r.storage.getLease(r.tenant, id)
}
func (r *boundLeaseRepository) List() ([]Lease, error) { return r.storage.listLeases(r.tenant) }
func (r *boundLeaseRepository) History(unitID string) ([]Lease, error) {
	return r.storage.leaseHistory(r.tenant, unitID)
}
func (r *boundLeaseRepository) ReplaceParties(leaseID string, parties []LeaseParty) error {
	return r.storage.replaceLeaseParties(r.tenant, leaseID, parties)
}
func (r *boundLeaseRepository) AddRentComponent(leaseID string, component RentComponent) (RentComponent, error) {
	return r.storage.addRentComponent(r.tenant, leaseID, component)
}
func (r *boundLeaseRepository) SetClause(clause IndexClause) (IndexClause, error) {
	return r.storage.setLeaseClause(r.tenant, clause)
}
func (r *boundLeaseRepository) ReviewClause(id, status, note string) (IndexClause, error) {
	return r.storage.reviewLeaseClause(r.tenant, id, status, note)
}
func (r *boundLeaseRepository) SetValorisation(state ValorisationState) error {
	return r.storage.setValorisation(r.tenant, state)
}
func (r *boundLeaseRepository) Import(drafts []LeaseImportDraft, units []Unit, commit bool, actor string) (LeaseImportReport, error) {
	return r.storage.importLeases(r.tenant, drafts, units, commit, actor)
}

func newLeaseID() (string, error) {
	id, err := ulid.New()
	if err != nil {
		return "", err
	}
	return strings.ToLower(id), nil
}

func normalizeLease(lease Lease) (Lease, error) {
	if strings.TrimSpace(lease.ID) == "" {
		id, err := newLeaseID()
		if err != nil {
			return Lease{}, err
		}
		lease.ID = id
	}
	lease.UnitID = textutil.UnitID(lease.UnitID)
	lease.Status = strings.ToLower(strings.TrimSpace(lease.Status))
	if lease.Status == "" {
		lease.Status = LeaseStatusActive
	}
	lease.LeaseKind = strings.ToLower(strings.TrimSpace(lease.LeaseKind))
	if lease.LeaseKind == "" {
		lease.LeaseKind = LeaseKindHauptmiete
	}
	lease.UseKind = normalizeUseKind(lease.UseKind)
	lease.MRGScope = strings.ToLower(strings.TrimSpace(lease.MRGScope))
	lease.RentRegime = strings.ToLower(strings.TrimSpace(lease.RentRegime))
	lease.ConcludedOn = strings.TrimSpace(lease.ConcludedOn)
	lease.StartsOn = strings.TrimSpace(lease.StartsOn)
	lease.EndsOn = strings.TrimSpace(lease.EndsOn)
	lease.Notes = strings.TrimSpace(lease.Notes)
	lease.UpdatedBy = textutil.Email(lease.UpdatedBy)
	if lease.ZinsterminDay == 0 {
		lease.ZinsterminDay = 5
	}
	if !lease.PriceRestrictedSet {
		lease.PriceRestricted = DerivedPriceRestricted(lease.MRGScope, lease.RentRegime)
	}
	if err := validateLease(lease); err != nil {
		return Lease{}, err
	}
	parties := make([]LeaseParty, 0, len(lease.Parties))
	for _, party := range lease.Parties {
		normalized, err := normalizeParty(lease.ID, party)
		if err != nil {
			return Lease{}, err
		}
		parties = append(parties, normalized)
	}
	lease.Parties = parties
	components := make([]RentComponent, 0, len(lease.Components))
	for _, component := range lease.Components {
		normalized, err := normalizeComponent(lease.ID, component)
		if err != nil {
			return Lease{}, err
		}
		components = append(components, normalized)
	}
	lease.Components = components
	clauses := make([]IndexClause, 0, len(lease.Clauses))
	for _, clause := range lease.Clauses {
		normalized, err := normalizeClause(lease.ID, clause)
		if err != nil {
			return Lease{}, err
		}
		clauses = append(clauses, normalized)
	}
	lease.Clauses = clauses
	return lease, nil
}

func validateLease(lease Lease) error {
	if lease.UnitID == "" {
		return fmt.Errorf("%w: unit", ErrLeaseInvalid)
	}
	switch lease.Status {
	case LeaseStatusDraft, LeaseStatusActive, LeaseStatusEnded:
	default:
		return fmt.Errorf("%w: status", ErrLeaseInvalid)
	}
	switch lease.LeaseKind {
	case LeaseKindHauptmiete, LeaseKindUntermiete:
	default:
		return fmt.Errorf("%w: lease kind", ErrLeaseInvalid)
	}
	switch lease.UseKind {
	case UseKindWohnung, UseKindGeschaeft, UseKindGarage, UseKindSonstiges:
	default:
		return fmt.Errorf("%w: use", ErrLeaseInvalid)
	}
	switch lease.MRGScope {
	case MRGVoll, MRGTeil, MRGAusnahme, MRGWGG:
	default:
		return fmt.Errorf("%w: mrg", ErrLeaseInvalid)
	}
	switch lease.RentRegime {
	case RentRegimeRichtwert, RentRegimeKategorie, RentRegimeAngemessen, RentRegimeFrei, RentRegimeSonstig:
	default:
		return fmt.Errorf("%w: regime", ErrLeaseInvalid)
	}
	if _, err := time.Parse("2006-01-02", lease.ConcludedOn); err != nil {
		return fmt.Errorf("%w: concluded_on", ErrLeaseInvalid)
	}
	start, err := time.Parse("2006-01-02", lease.StartsOn)
	if err != nil {
		return fmt.Errorf("%w: starts_on", ErrLeaseInvalid)
	}
	if lease.EndsOn != "" {
		end, err := time.Parse("2006-01-02", lease.EndsOn)
		if err != nil || !end.After(start) {
			return fmt.Errorf("%w: ends_on", ErrLeaseInvalid)
		}
	}
	if lease.Status == LeaseStatusEnded && lease.EndsOn == "" {
		return fmt.Errorf("%w: ended lease needs ends_on", ErrLeaseInvalid)
	}
	switch lease.WirksamwerdenMode {
	case "", "wko", "oevi", "contract":
	default:
		return fmt.Errorf("%w: wirksamwerden mode", ErrLeaseInvalid)
	}
	if lease.ZinsterminDay < 1 || lease.ZinsterminDay > 28 {
		return fmt.Errorf("%w: zinstermin", ErrLeaseInvalid)
	}
	if lease.MaxHMZCents != nil && *lease.MaxHMZCents < 0 {
		return fmt.Errorf("%w: max hmz", ErrLeaseInvalid)
	}
	if len([]rune(lease.Notes)) > 4000 {
		return fmt.Errorf("%w: notes", ErrLeaseInvalid)
	}
	return nil
}

func normalizeParty(leaseID string, party LeaseParty) (LeaseParty, error) {
	if strings.TrimSpace(party.ID) == "" {
		id, err := newLeaseID()
		if err != nil {
			return LeaseParty{}, err
		}
		party.ID = id
	}
	party.LeaseID = leaseID
	party.Name = strings.TrimSpace(party.Name)
	party.Address = strings.TrimSpace(party.Address)
	party.Email = textutil.Email(party.Email)
	party.Role = strings.ToLower(strings.TrimSpace(party.Role))
	if party.Role == "" {
		party.Role = PartyHauptmieter
	}
	party.ValidFrom = strings.TrimSpace(party.ValidFrom)
	party.ValidTo = strings.TrimSpace(party.ValidTo)
	if party.Name == "" {
		return LeaseParty{}, fmt.Errorf("%w: party name", ErrLeaseInvalid)
	}
	if party.Role != PartyHauptmieter && party.Role != PartyMitmieter {
		return LeaseParty{}, fmt.Errorf("%w: party role", ErrLeaseInvalid)
	}
	if _, err := time.Parse("2006-01-02", party.ValidFrom); err != nil {
		return LeaseParty{}, fmt.Errorf("%w: party valid_from", ErrLeaseInvalid)
	}
	if party.ValidTo != "" {
		if _, err := time.Parse("2006-01-02", party.ValidTo); err != nil {
			return LeaseParty{}, fmt.Errorf("%w: party valid_to", ErrLeaseInvalid)
		}
	}
	return party, nil
}

func normalizeComponent(leaseID string, component RentComponent) (RentComponent, error) {
	if strings.TrimSpace(component.ID) == "" {
		id, err := newLeaseID()
		if err != nil {
			return RentComponent{}, err
		}
		component.ID = id
	}
	component.LeaseID = leaseID
	component.Kind = strings.ToLower(strings.TrimSpace(component.Kind))
	component.ValidFrom = strings.TrimSpace(component.ValidFrom)
	component.Origin = strings.TrimSpace(component.Origin)
	if component.Origin == "" {
		component.Origin = OriginManual
	}
	if component.CreatedAt.IsZero() {
		component.CreatedAt = time.Now().UTC()
	}
	switch component.Kind {
	case ComponentHMZ, ComponentBKAkonto, ComponentHeizAkonto, ComponentLift, ComponentMoebel, ComponentStellplatz, ComponentSonstiges:
	default:
		return RentComponent{}, fmt.Errorf("%w: component kind", ErrLeaseInvalid)
	}
	if component.NetCents < 0 || component.VATRateBP < 0 {
		return RentComponent{}, fmt.Errorf("%w: component amount", ErrLeaseInvalid)
	}
	if _, err := time.Parse("2006-01-02", component.ValidFrom); err != nil {
		return RentComponent{}, fmt.Errorf("%w: component valid_from", ErrLeaseInvalid)
	}
	return component, nil
}

func normalizeClause(leaseID string, clause IndexClause) (IndexClause, error) {
	if strings.TrimSpace(clause.ID) == "" {
		id, err := newLeaseID()
		if err != nil {
			return IndexClause{}, err
		}
		clause.ID = id
	}
	clause.LeaseID = leaseID
	clause.ComponentKind = strings.ToLower(strings.TrimSpace(clause.ComponentKind))
	if clause.ComponentKind == "" {
		clause.ComponentKind = ComponentHMZ
	}
	clause.ClauseType = normalizeClauseType(clause.ClauseType)
	clause.Series = strings.ToLower(strings.TrimSpace(clause.Series))
	clause.BasePeriod = strings.TrimSpace(clause.BasePeriod)
	clause.BaseValue = strings.TrimSpace(clause.BaseValue)
	clause.ThresholdKind = strings.ToLower(strings.TrimSpace(clause.ThresholdKind))
	clause.ThresholdValue = strings.TrimSpace(clause.ThresholdValue)
	clause.PctRounding = strings.ToLower(strings.TrimSpace(clause.PctRounding))
	if clause.PctRounding == "" {
		clause.PctRounding = "none"
	}
	clause.ReviewStatus = strings.ToLower(strings.TrimSpace(clause.ReviewStatus))
	if clause.ReviewStatus == "" {
		clause.ReviewStatus = ReviewUnreviewed
	}
	clause.ReviewNote = strings.TrimSpace(clause.ReviewNote)
	clause.ClauseText = strings.TrimSpace(clause.ClauseText)
	clause.ValidFrom = strings.TrimSpace(clause.ValidFrom)
	if clause.ClauseType == "" {
		clause.ClauseType = ClauseNone
	}
	switch clause.ClauseType {
	case ClauseMieWeG, ClauseVPIThreshold, ClauseVPIPeriodic, ClauseStaffel, ClauseNone:
	default:
		return IndexClause{}, fmt.Errorf("%w: clause type", ErrLeaseInvalid)
	}
	switch clause.ReviewStatus {
	case ReviewUnreviewed, ReviewOK, ReviewDoubtful, ReviewInvalid:
	default:
		return IndexClause{}, fmt.Errorf("%w: review", ErrLeaseInvalid)
	}
	if clause.PctRounding != "none" && clause.PctRounding != "one_decimal" {
		return IndexClause{}, fmt.Errorf("%w: rounding", ErrLeaseInvalid)
	}
	if clause.ThresholdKind != "" && clause.ThresholdKind != "percent" && clause.ThresholdKind != "points" {
		return IndexClause{}, fmt.Errorf("%w: threshold kind", ErrLeaseInvalid)
	}
	if clause.BaseValue != "" {
		value, err := NormalizeIndexDecimal(clause.BaseValue)
		if err != nil {
			return IndexClause{}, fmt.Errorf("%w: base value", ErrLeaseInvalid)
		}
		clause.BaseValue = value
	}
	if clause.ThresholdValue != "" {
		value, err := NormalizeIndexDecimal(clause.ThresholdValue)
		if err != nil {
			return IndexClause{}, fmt.Errorf("%w: threshold", ErrLeaseInvalid)
		}
		clause.ThresholdValue = value
	}
	if clause.BasePeriod != "" {
		if _, err := time.Parse("2006-01", clause.BasePeriod); err != nil {
			return IndexClause{}, fmt.Errorf("%w: base period", ErrLeaseInvalid)
		}
	}
	if clause.ValidFrom == "" {
		return IndexClause{}, fmt.Errorf("%w: clause valid_from", ErrLeaseInvalid)
	}
	if _, err := time.Parse("2006-01-02", clause.ValidFrom); err != nil {
		return IndexClause{}, fmt.Errorf("%w: clause valid_from", ErrLeaseInvalid)
	}
	if clause.PeriodicMonth < 0 || clause.PeriodicMonth > 12 {
		return IndexClause{}, fmt.Errorf("%w: periodic month", ErrLeaseInvalid)
	}
	if clause.State != nil {
		clause.State.ClauseID = clause.ID
		if err := normalizeValorisation(clause.State); err != nil {
			return IndexClause{}, err
		}
	}
	if clause.ClauseType == ClauseStaffel {
		if err := indexation.ValidateStaffelSteps(clause.StaffelSteps); err != nil {
			return IndexClause{}, fmt.Errorf("%w: %s", ErrLeaseInvalid, err)
		}
		if clause.StaffelSteps[0].EffectiveOn <= clause.ValidFrom {
			return IndexClause{}, fmt.Errorf("%w: Staffel muss nach Klauselbeginn liegen", ErrLeaseInvalid)
		}
	} else if len(clause.StaffelSteps) != 0 {
		return IndexClause{}, fmt.Errorf("%w: Staffeln nur für Staffelklauseln", ErrLeaseInvalid)
	}
	return clause, nil
}

func normalizeValorisation(state *ValorisationState) error {
	if state == nil {
		return nil
	}
	var err error
	if state.ContractValue, err = optionalIndexDecimal(state.ContractValue); err != nil {
		return fmt.Errorf("%w: contract value", ErrLeaseInvalid)
	}
	if state.ContractBaseValue, err = optionalIndexDecimal(state.ContractBaseValue); err != nil {
		return fmt.Errorf("%w: contract base", ErrLeaseInvalid)
	}
	if state.CapValue, err = optionalIndexDecimal(state.CapValue); err != nil {
		return fmt.Errorf("%w: cap value", ErrLeaseInvalid)
	}
	state.ContractBasePeriod = strings.TrimSpace(state.ContractBasePeriod)
	state.CapAnchorPeriod = strings.TrimSpace(state.CapAnchorPeriod)
	for _, period := range []string{state.ContractBasePeriod, state.CapAnchorPeriod} {
		if period == "" {
			continue
		}
		if _, err := time.Parse("2006-01", period); err != nil {
			return fmt.Errorf("%w: valorisation period", ErrLeaseInvalid)
		}
	}
	state.LastEffectiveOn = strings.TrimSpace(state.LastEffectiveOn)
	if state.LastEffectiveOn != "" {
		if _, err := time.Parse("2006-01-02", state.LastEffectiveOn); err != nil {
			return fmt.Errorf("%w: last effective", ErrLeaseInvalid)
		}
	}
	state.LastRunItemID = strings.TrimSpace(state.LastRunItemID)
	return nil
}

func optionalIndexDecimal(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	return NormalizeIndexDecimal(raw)
}

// NormalizeIndexDecimal keeps a published index figure as exact decimal text.
// A comma is the Austrian decimal separator and becomes a dot. Floats are not used.
func NormalizeIndexDecimal(raw string) (string, error) {
	raw = strings.ReplaceAll(strings.TrimSpace(raw), " ", "")
	if raw == "" {
		return "", fmt.Errorf("empty index value")
	}
	if strings.Contains(raw, ",") {
		raw = strings.ReplaceAll(raw, ".", "")
		raw = strings.ReplaceAll(raw, ",", ".")
	}
	if strings.HasPrefix(raw, "+") {
		raw = raw[1:]
	}
	parts := strings.Split(raw, ".")
	if len(parts) > 2 || parts[0] == "" || strings.ContainsAny(parts[0], "-") && parts[0] != strings.TrimLeft(parts[0], "-") {
		return "", fmt.Errorf("index value")
	}
	digits := parts[0]
	if strings.HasPrefix(digits, "-") {
		digits = digits[1:]
	}
	if digits == "" || strings.Trim(digits, "0123456789") != "" {
		return "", fmt.Errorf("index value")
	}
	if len(parts) == 2 && (parts[1] == "" || strings.Trim(parts[1], "0123456789") != "") {
		return "", fmt.Errorf("index value")
	}
	return raw, nil
}

func DerivedPriceRestricted(scope, regime string) bool {
	if scope != MRGVoll {
		return false
	}
	switch regime {
	case RentRegimeRichtwert, RentRegimeKategorie, RentRegimeAngemessen:
		return true
	default:
		return false
	}
}

// ClassifyLease reports whether the MieWeG and the special cap apply, with the reason.
func ClassifyLease(lease Lease) LeaseClassification {
	out := LeaseClassification{}
	switch {
	case lease.UseKind != UseKindWohnung:
		out.MieWeGReason = "Keine Wohnung, daher kein MieWeG."
	case lease.MRGScope == MRGAusnahme:
		out.MieWeGReason = "Volle Ausnahme vom MRG, daher kein MieWeG."
	case lease.MRGScope == MRGWGG:
		out.MieWeGReason = "WGG, daher kein MieWeG."
	case lease.LeaseKind != LeaseKindHauptmiete && lease.LeaseKind != LeaseKindUntermiete:
		out.MieWeGReason = "Kein Haupt- oder Untermietvertrag."
	case lease.MRGScope == MRGVoll:
		out.MieWeG = true
		out.MieWeGReason = "Wohnung in Vollanwendung."
	case lease.MRGScope == MRGTeil:
		out.MieWeG = true
		out.MieWeGReason = "Wohnung in Teilanwendung."
	default:
		out.MieWeGReason = "MieWeG gilt nicht."
	}
	if !out.MieWeG {
		out.CapReason = "Kein Sonderdeckel, weil das MieWeG nicht gilt."
		return out
	}
	if lease.PriceRestricted {
		out.SpecialCap = true
		out.CapReason = "Preisbildung nach Richtwert, Kategorie oder angemessenem Mietzins."
		return out
	}
	out.CapReason = "Freier Mietzins, kein Sonderdeckel."
	return out
}

func normalizeUseKind(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "wohnung", "wohnen", "apartment":
		return UseKindWohnung
	case "geschaeft", "geschäft", "commercial", "lokal":
		return UseKindGeschaeft
	case "garage", "stellplatz", "parking":
		return UseKindGarage
	default:
		return UseKindSonstiges
	}
}

func normalizeClauseType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "none", "keine", "ohne":
		return ClauseNone
	case ClauseMieWeG, "mieweg", "wertgesichert":
		return ClauseMieWeG
	case ClauseVPIThreshold, "schwelle", "vpi":
		return ClauseVPIThreshold
	case ClauseVPIPeriodic, "periodisch":
		return ClauseVPIPeriodic
	case ClauseStaffel, "staffelmietzins":
		return ClauseStaffel
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func leaseEndExclusive(endsOn string) string {
	if strings.TrimSpace(endsOn) == "" {
		return openEnded
	}
	return endsOn
}

func rangesOverlap(startA, endA, startB, endB string) bool {
	return startA < leaseEndExclusive(endB) && startB < leaseEndExclusive(endA)
}

func hauptmieteOccupies(lease Lease) bool {
	return lease.LeaseKind == LeaseKindHauptmiete
}

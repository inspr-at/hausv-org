package energy

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	MeasureDraft     = "draft"
	MeasureRequested = "requested"
	MeasureAssigned  = "assigned"
	MeasureScheduled = "scheduled"
	MeasureCompleted = "completed"
	MeasureCancelled = "cancelled"
)

type MaintenancePlan struct {
	ID              string
	TenantSlug      string
	HomeKey         string
	AssetID         string
	Title           string
	IntervalMonths  int
	LastCompletedAt *time.Time
	NextDueAt       time.Time
	ContactID       string
	DocumentID      string
	IssueID         string
	EvidenceNote    string
	Active          bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NormalizeMaintenancePlan(plan MaintenancePlan, now time.Time) (MaintenancePlan, error) {
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()
	plan.ID = strings.TrimSpace(plan.ID)
	if plan.ID == "" {
		plan.ID = NewID("maintenance")
	}
	plan.TenantSlug = normalizeSlug(plan.TenantSlug)
	plan.HomeKey = NormalizeHomeKey(plan.HomeKey)
	plan.AssetID = strings.TrimSpace(plan.AssetID)
	plan.Title = strings.TrimSpace(plan.Title)
	if plan.Title == "" {
		plan.Title = "Wartung prüfen"
	}
	if plan.IntervalMonths < 1 || plan.IntervalMonths > 120 {
		return MaintenancePlan{}, fmt.Errorf("energy: maintenance interval must be between 1 and 120 months")
	}
	plan.ContactID = strings.TrimSpace(plan.ContactID)
	plan.DocumentID = strings.TrimSpace(plan.DocumentID)
	plan.IssueID = strings.TrimSpace(plan.IssueID)
	plan.EvidenceNote = strings.TrimSpace(plan.EvidenceNote)
	if plan.TenantSlug == "" || plan.AssetID == "" {
		return MaintenancePlan{}, fmt.Errorf("energy: maintenance tenant and asset required")
	}
	if plan.LastCompletedAt != nil {
		completed := plan.LastCompletedAt.UTC()
		plan.LastCompletedAt = &completed
		if plan.NextDueAt.IsZero() {
			plan.NextDueAt = completed.AddDate(0, plan.IntervalMonths, 0)
		}
	}
	if plan.NextDueAt.IsZero() {
		plan.NextDueAt = now.AddDate(0, plan.IntervalMonths, 0)
	}
	plan.NextDueAt = plan.NextDueAt.UTC()
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = now
	}
	plan.CreatedAt = plan.CreatedAt.UTC()
	plan.UpdatedAt = now
	return plan, nil
}

func MaintenanceRecommendation(now time.Time, plans []MaintenancePlan) (Recommendation, bool) {
	if now.IsZero() {
		now = time.Now()
	}
	active := make([]MaintenancePlan, 0, len(plans))
	for _, plan := range plans {
		if plan.Active && !plan.NextDueAt.IsZero() {
			active = append(active, plan)
		}
	}
	if len(active) == 0 {
		return Recommendation{}, false
	}
	sort.Slice(active, func(i, j int) bool { return active[i].NextDueAt.Before(active[j].NextDueAt) })
	next := active[0]
	days := int(math.Ceil(next.NextDueAt.Sub(now).Hours() / 24))
	if days > 30 {
		return Recommendation{}, false
	}
	reason := "Die nächste Wartung ist bald fällig."
	impact := "Fällig in " + fmt.Sprintf("%d", days) + " Tagen"
	if days <= 0 {
		reason = "Die Wartung ist fällig. Ein Nachweis hält den Hauszustand nachvollziehbar."
		impact = "Jetzt fällig"
	}
	return Recommendation{
		ID:           "maintenance-" + next.ID,
		Title:        next.Title,
		Reason:       reason,
		Benefit:      "Zuverlässiger Betrieb",
		Prerequisite: "Keine",
		Effort:       "Termin abstimmen",
		ImpactRange:  impact,
		State:        "now",
	}, true
}

type TariffAssessment struct {
	ID              string
	TenantSlug      string
	HomeKey         string
	AssessmentMonth string
	ProfileID       string
	ProfileVersion  string
	ProfileStatus   string
	SourceURL       string
	PeakKW          float64
	BilledKW        float64
	AnnualPowerEUR  float64
	DataQuality     string
	CreatedAt       time.Time
}

func NormalizeTariffAssessment(item TariffAssessment, now time.Time) (TariffAssessment, error) {
	if now.IsZero() {
		now = time.Now()
	}
	item.ID = strings.TrimSpace(item.ID)
	if item.ID == "" {
		item.ID = NewID("tariff")
	}
	item.TenantSlug = normalizeSlug(item.TenantSlug)
	item.HomeKey = NormalizeHomeKey(item.HomeKey)
	item.AssessmentMonth = strings.TrimSpace(item.AssessmentMonth)
	item.ProfileID = strings.TrimSpace(item.ProfileID)
	item.ProfileVersion = strings.TrimSpace(item.ProfileVersion)
	item.ProfileStatus = strings.TrimSpace(item.ProfileStatus)
	item.SourceURL = strings.TrimSpace(item.SourceURL)
	item.DataQuality = normalizeToken(item.DataQuality, QualityUnavailable)
	item.PeakKW = round2(math.Max(0, item.PeakKW))
	item.BilledKW = round2(math.Max(0, item.BilledKW))
	item.AnnualPowerEUR = round2(math.Max(0, item.AnnualPowerEUR))
	if item.TenantSlug == "" || item.AssessmentMonth == "" || item.ProfileID == "" || item.ProfileVersion == "" {
		return TariffAssessment{}, fmt.Errorf("energy: tariff assessment identity required")
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now.UTC()
	}
	item.CreatedAt = item.CreatedAt.UTC()
	return item, nil
}

type Measure struct {
	ID               string
	TenantSlug       string
	HomeKey          string
	IssueID          string
	RecommendationID string
	Title            string
	Status           string
	ContactID        string
	SharedFields     []string
	OfferNote        string
	AppointmentAt    *time.Time
	WorkNote         string
	CompletedAt      *time.Time
	EvidenceNote     string
	BeforeFrom       *time.Time
	BeforeTo         *time.Time
	AfterFrom        *time.Time
	AfterTo          *time.Time
	BeforePeakKW     *float64
	AfterPeakKW      *float64
	BeforeQuality    string
	AfterQuality     string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NormalizeMeasure(item Measure, now time.Time) (Measure, error) {
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()
	item.ID = strings.TrimSpace(item.ID)
	if item.ID == "" {
		item.ID = NewID("measure")
	}
	item.TenantSlug = normalizeSlug(item.TenantSlug)
	item.HomeKey = NormalizeHomeKey(item.HomeKey)
	item.IssueID = strings.TrimSpace(item.IssueID)
	item.RecommendationID = normalizeToken(item.RecommendationID, "other")
	item.Title = strings.TrimSpace(item.Title)
	if item.Title == "" {
		item.Title = "Energiemaßnahme"
	}
	switch strings.ToLower(strings.TrimSpace(item.Status)) {
	case MeasureRequested, MeasureAssigned, MeasureScheduled, MeasureCompleted, MeasureCancelled:
		item.Status = strings.ToLower(strings.TrimSpace(item.Status))
	default:
		item.Status = MeasureDraft
	}
	item.ContactID = strings.TrimSpace(item.ContactID)
	item.SharedFields = uniqueTokens(item.SharedFields, 16)
	item.OfferNote = strings.TrimSpace(item.OfferNote)
	item.WorkNote = strings.TrimSpace(item.WorkNote)
	item.EvidenceNote = strings.TrimSpace(item.EvidenceNote)
	item.AppointmentAt = utcTime(item.AppointmentAt)
	item.CompletedAt = utcTime(item.CompletedAt)
	item.BeforeFrom = utcTime(item.BeforeFrom)
	item.BeforeTo = utcTime(item.BeforeTo)
	item.AfterFrom = utcTime(item.AfterFrom)
	item.AfterTo = utcTime(item.AfterTo)
	item.BeforePeakKW = roundedPointer(item.BeforePeakKW)
	item.AfterPeakKW = roundedPointer(item.AfterPeakKW)
	item.BeforeQuality = normalizeToken(item.BeforeQuality, "")
	item.AfterQuality = normalizeToken(item.AfterQuality, "")
	if item.TenantSlug == "" || item.IssueID == "" {
		return Measure{}, fmt.Errorf("energy: measure tenant and issue required")
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.CreatedAt = item.CreatedAt.UTC()
	item.UpdatedAt = now
	return item, nil
}

func PeakForRange(intervals []Interval) (*float64, string) {
	var peak float64
	found := false
	quality := QualityMeasured
	for _, interval := range intervals {
		if interval.Quality == QualityGap || interval.Quality == QualityConflict || interval.Quality == QualityUnavailable {
			quality = interval.Quality
			continue
		}
		if interval.Quality != QualityMeasured && interval.Quality != QualityEstimated {
			continue
		}
		if !found || interval.AverageKW > peak {
			peak = interval.AverageKW
			found = true
		}
		if interval.Quality == QualityEstimated && quality == QualityMeasured {
			quality = QualityEstimated
		}
	}
	if !found {
		return nil, QualityUnavailable
	}
	peak = round2(peak)
	return &peak, quality
}

func utcTime(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

func roundedPointer(value *float64) *float64 {
	if value == nil {
		return nil
	}
	rounded := round2(math.Max(0, *value))
	return &rounded
}

func uniqueTokens(values []string, limit int) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = normalizeToken(value, "")
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
		if len(out) >= limit {
			break
		}
	}
	sort.Strings(out)
	return out
}

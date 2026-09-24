package store

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

const (
	ImportWarningIndexUnchecked = "index_check_unavailable"
	ImportWarningBaseMismatch   = "base_value_mismatch"
)

// LeaseImportDraft is one CSV row before it becomes a lease.
type LeaseImportDraft struct {
	Line                                         int
	Unit, TenantName, TenantEmail, TenantAddress string
	ConcludedOn, StartsOn, EndsOn                string
	UseKind, MRGScope, RentRegime                string
	HMZCents, BKCents, HeatCents                 int64
	HasBK, HasHeat                               bool
	VATRateBP                                    int
	ClauseType, Series, BasePeriod, BaseValue    string
	Threshold, ThresholdKind                     string
	ThresholdInclusive                           bool
	LastMonth, LastValue, HMZAfter               string
	ClauseText                                   string
	Staffel                                      string
}

type LeaseImportRow struct {
	Line     int
	Unit     string
	Errors   []string
	Warnings []string
}

type LeaseImportReport struct {
	Rows      []LeaseImportRow
	Committed int
}

func (r LeaseImportReport) HasErrors() bool {
	for _, row := range r.Rows {
		if len(row.Errors) > 0 {
			return true
		}
	}
	return false
}

// ParseLeaseCSV reads the slice-2 lease table. A missing column or an unreadable
// file is an error. A bad cell becomes a row error later, during Import.
func ParseLeaseCSV(raw []byte) ([]LeaseImportDraft, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty csv")
	}
	reader := csv.NewReader(bytes.NewReader(raw))
	if first := strings.SplitN(string(raw), "\n", 2)[0]; strings.Count(first, ";") > strings.Count(first, ",") {
		reader.Comma = ';'
	}
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil || len(rows) < 2 {
		return nil, fmt.Errorf("invalid csv")
	}
	columns := map[string]int{}
	for index, value := range rows[0] {
		columns[strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "\ufeff")))] = index
	}
	required := []string{"einheit", "vertragsabschluss", "beginn", "nutzung", "mrg", "regime", "hmz_netto"}
	for _, name := range required {
		if _, ok := columns[name]; !ok {
			return nil, fmt.Errorf("missing column %s", name)
		}
	}
	cell := func(row []string, name string) string {
		index, ok := columns[name]
		if !ok || index >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[index])
	}
	drafts := []LeaseImportDraft{}
	for line, row := range rows[1:] {
		if strings.TrimSpace(strings.Join(row, "")) == "" {
			continue
		}
		draft := LeaseImportDraft{
			Line: line + 2, Unit: cell(row, "einheit"),
			TenantName: cell(row, "mieter_name"), TenantEmail: cell(row, "mieter_email"), TenantAddress: cell(row, "mieter_adresse"),
			ConcludedOn: cell(row, "vertragsabschluss"), StartsOn: cell(row, "beginn"), EndsOn: cell(row, "ende"),
			UseKind: cell(row, "nutzung"), MRGScope: cell(row, "mrg"), RentRegime: cell(row, "regime"),
			ClauseType: cell(row, "klausel_typ"), Series: cell(row, "index"), BasePeriod: cell(row, "basis_monat"),
			BaseValue: cell(row, "basis_wert"), Threshold: cell(row, "schwelle"), ThresholdKind: cell(row, "schwelle_art"),
			LastMonth: cell(row, "letzte_valorisierung_monat"), LastValue: cell(row, "letzte_valorisierung_wert"),
			HMZAfter: cell(row, "hmz_nach_letzter_valorisierung"), ClauseText: cell(row, "klausel_text"),
			Staffel: cell(row, "staffel"),
		}
		var err error
		if draft.HMZCents, err = parseEuroCents(cell(row, "hmz_netto"), true); err != nil {
			draft.HMZCents = -1
		}
		if value := cell(row, "bk_akonto"); value != "" {
			draft.HasBK = true
			if draft.BKCents, err = parseEuroCents(value, true); err != nil {
				draft.BKCents = -1
			}
		}
		if value := cell(row, "heiz_akonto"); value != "" {
			draft.HasHeat = true
			if draft.HeatCents, err = parseEuroCents(value, true); err != nil {
				draft.HeatCents = -1
			}
		}
		draft.VATRateBP = parseVATBasisPoints(cell(row, "ust_satz"), draft.UseKind)
		draft.ThresholdInclusive = parseYes(cell(row, "schwelle_inkl"))
		draft.ConcludedOn = normalizeImportDate(draft.ConcludedOn)
		draft.StartsOn = normalizeImportDate(draft.StartsOn)
		draft.EndsOn = normalizeImportDate(draft.EndsOn)
		draft.BasePeriod = normalizeImportMonth(draft.BasePeriod)
		draft.LastMonth = normalizeImportMonth(draft.LastMonth)
		draft.MRGScope = strings.ToLower(draft.MRGScope)
		draft.RentRegime = strings.ToLower(draft.RentRegime)
		switch draft.ThresholdKind {
		case "prozent", "percent", "%":
			draft.ThresholdKind = "percent"
		case "punkte", "points":
			draft.ThresholdKind = "points"
		}
		drafts = append(drafts, draft)
	}
	if len(drafts) == 0 {
		return nil, fmt.Errorf("no rows")
	}
	return drafts, nil
}

func parseYes(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "ja", "yes", "true", "j":
		return true
	default:
		return false
	}
}

func parseVATBasisPoints(raw, useKind string) int {
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "%"))
	raw = strings.ReplaceAll(raw, ",", ".")
	if raw == "" {
		switch normalizeUseKind(useKind) {
		case UseKindGarage:
			return 2000
		case UseKindGeschaeft:
			return 0
		default:
			return 1000
		}
	}
	switch raw {
	case "0":
		return 0
	case "10":
		return 1000
	case "20":
		return 2000
	default:
		return -1
	}
}

// ParseMoneyCents reads an Austrian or plain euro amount into integer cents.
func ParseMoneyCents(raw string) (int64, error) {
	return parseEuroCents(raw, true)
}

func parseEuroCents(raw string, required bool) (int64, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, "€")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if required {
			return 0, fmt.Errorf("amount")
		}
		return 0, nil
	}
	negative := strings.HasPrefix(raw, "-")
	raw = strings.TrimPrefix(raw, "-")
	if strings.Contains(raw, ",") {
		raw = strings.ReplaceAll(raw, ".", "")
		raw = strings.ReplaceAll(raw, ",", ".")
	} else if strings.Count(raw, ".") == 1 {
		parts := strings.Split(raw, ".")
		if len(parts[1]) == 3 {
			raw = parts[0] + parts[1]
		}
	}
	parts := strings.Split(raw, ".")
	if len(parts) > 2 || parts[0] == "" || strings.Trim(parts[0], "0123456789") != "" {
		return 0, fmt.Errorf("amount")
	}
	euros, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	frac := int64(0)
	if len(parts) == 2 {
		fraction := parts[1]
		if fraction == "" || len(fraction) > 2 || strings.Trim(fraction, "0123456789") != "" {
			return 0, fmt.Errorf("amount")
		}
		if len(fraction) == 1 {
			fraction += "0"
		}
		frac, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, err
		}
	}
	cents := euros*100 + frac
	if negative {
		cents = -cents
	}
	return cents, nil
}

func normalizeImportDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	for _, layout := range []string{"2006-01-02", "02.01.2006", "2.1.2006"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.Format("2006-01-02")
		}
	}
	return raw
}

func normalizeImportMonth(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	for _, layout := range []string{"2006-01", "01.2006", "1.2006", "2006-01-02"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.Format("2006-01")
		}
	}
	return raw
}

func matchImportUnit(units []Unit, raw string) (string, bool) {
	key := textutil.UnitID(raw)
	for _, unit := range units {
		if textutil.UnitID(unit.ID) == key || textutil.UnitID(unit.Label) == key {
			return unit.ID, true
		}
	}
	return "", false
}

func prepareLeaseImport(drafts []LeaseImportDraft, units []Unit) (LeaseImportReport, []Lease) {
	report := LeaseImportReport{Rows: make([]LeaseImportRow, 0, len(drafts))}
	accepted := make([]Lease, 0, len(drafts))
	for _, draft := range drafts {
		unitID, _ := matchImportUnit(units, draft.Unit)
		lease, row := leaseFromDraft(draft, unitID)
		if len(row.Errors) == 0 && hauptmieteOccupies(lease) {
			for _, prior := range accepted {
				if prior.UnitID == lease.UnitID && hauptmieteOccupies(prior) && rangesOverlap(prior.StartsOn, prior.EndsOn, lease.StartsOn, lease.EndsOn) {
					row.Errors = append(row.Errors, "overlap")
					break
				}
			}
		}
		report.Rows = append(report.Rows, row)
		if len(row.Errors) == 0 {
			accepted = append(accepted, lease)
		}
	}
	return report, accepted
}

func leaseFromDraft(draft LeaseImportDraft, unitID string) (Lease, LeaseImportRow) {
	row := LeaseImportRow{Line: draft.Line, Unit: draft.Unit}
	if unitID == "" {
		row.Errors = append(row.Errors, "unknown_unit")
		return Lease{}, row
	}
	if draft.HMZCents < 0 || draft.BKCents < 0 || draft.HeatCents < 0 || draft.VATRateBP < 0 {
		row.Errors = append(row.Errors, "invalid_amount")
	}
	lease := Lease{
		UnitID: unitID, Status: LeaseStatusActive, ConcludedOn: draft.ConcludedOn, StartsOn: draft.StartsOn, EndsOn: draft.EndsOn,
		LeaseKind: LeaseKindHauptmiete, UseKind: normalizeUseKind(draft.UseKind), MRGScope: draft.MRGScope, RentRegime: draft.RentRegime,
		LandlordIsBusiness: true, TenantIsConsumer: normalizeUseKind(draft.UseKind) != UseKindGeschaeft,
		VATOpted: draft.VATRateBP > 0, ZinsterminDay: 5,
	}
	if _, err := normalizeLease(leaseWithoutChildren(lease)); err != nil {
		row.Errors = append(row.Errors, "invalid_lease")
	}
	if draft.TenantName != "" {
		lease.Parties = []LeaseParty{{
			Name: draft.TenantName, Email: draft.TenantEmail, Address: draft.TenantAddress,
			Role: PartyHauptmieter, ValidFrom: draft.StartsOn,
		}}
	}
	if draft.HMZCents >= 0 {
		lease.Components = append(lease.Components, RentComponent{Kind: ComponentHMZ, NetCents: draft.HMZCents, VATRateBP: draft.VATRateBP, ValidFrom: draft.StartsOn, Origin: OriginImport})
	}
	if draft.HasBK && draft.BKCents >= 0 {
		lease.Components = append(lease.Components, RentComponent{Kind: ComponentBKAkonto, NetCents: draft.BKCents, VATRateBP: draft.VATRateBP, ValidFrom: draft.StartsOn, Origin: OriginImport})
	}
	if draft.HasHeat && draft.HeatCents >= 0 {
		lease.Components = append(lease.Components, RentComponent{Kind: ComponentHeizAkonto, NetCents: draft.HeatCents, VATRateBP: draft.VATRateBP, ValidFrom: draft.StartsOn, Origin: OriginImport})
	}
	clause := IndexClause{
		ClauseType: normalizeClauseType(draft.ClauseType), Series: draft.Series, BasePeriod: draft.BasePeriod, BaseValue: draft.BaseValue,
		ThresholdKind: draft.ThresholdKind, ThresholdValue: draft.Threshold, ThresholdInclusive: draft.ThresholdInclusive,
		FullChangeOnTrigger: true, TwoWay: true, ClauseText: draft.ClauseText, ReviewStatus: ReviewUnreviewed, ValidFrom: draft.StartsOn,
	}
	if clause.ClauseType == ClauseStaffel {
		var err error
		clause.StaffelSteps, err = ParseStaffel(draft.Staffel)
		if err != nil {
			row.Errors = append(row.Errors, "invalid_staffel")
		}
	} else if draft.Staffel != "" {
		row.Errors = append(row.Errors, "invalid_staffel")
	}
	if clause.ClauseType != ClauseNone || clause.ClauseText != "" || clause.BaseValue != "" {
		if draft.LastMonth != "" || draft.LastValue != "" || draft.HMZAfter != "" {
			clause.State = &ValorisationState{
				ContractBasePeriod: draft.LastMonth, ContractBaseValue: draft.LastValue, CapAnchorPeriod: draft.LastMonth,
				ContractValue: draft.HMZAfter, CapValue: draft.HMZAfter,
			}
		}
		lease.Clauses = []IndexClause{clause}
	}
	if clause.BaseValue != "" && clause.Series != "" && clause.BasePeriod != "" {
		published, checked := PublishedIndexValue(clause.Series, clause.BasePeriod)
		if !checked {
			row.Warnings = append(row.Warnings, ImportWarningIndexUnchecked)
		} else if published != clause.BaseValue && published != strings.ReplaceAll(clause.BaseValue, ",", ".") {
			row.Warnings = append(row.Warnings, ImportWarningBaseMismatch)
		}
	}
	if len(row.Errors) > 0 {
		return Lease{}, row
	}
	normalized, err := normalizeLease(lease)
	if err != nil {
		row.Errors = append(row.Errors, "invalid_lease")
		return Lease{}, row
	}
	return normalized, row
}

func leaseWithoutChildren(lease Lease) Lease {
	lease.Parties, lease.Components, lease.Clauses = nil, nil, nil
	return lease
}

// ReadLeaseCSV is the io.Reader form used by the upload handler.
func ReadLeaseCSV(r io.Reader, limit int64) ([]LeaseImportDraft, error) {
	raw, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil || int64(len(raw)) > limit {
		return nil, fmt.Errorf("invalid csv")
	}
	return ParseLeaseCSV(raw)
}

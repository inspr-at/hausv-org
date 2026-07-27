package server

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/markus-barta/hausv-org/internal/integrations"
)

type unitPaymentReferenceCandidate struct {
	TenantSlug     string
	UnitID         string
	UnitLabel      string
	Reference      string
	ExpectedAmount integrations.MoneyAmount
}

type unitPaymentImportDecision string

const (
	unitPaymentImportAssigned unitPaymentImportDecision = "zugeordnet"
	unitPaymentImportUnclear  unitPaymentImportDecision = "unklar"
	unitPaymentImportRejected unitPaymentImportDecision = "abgelehnt"
)

type unitPaymentImportRow struct {
	Decision       unitPaymentImportDecision
	PaymentID      string
	Reference      string
	UnitID         string
	UnitLabel      string
	Status         string
	Reason         string
	Amount         integrations.MoneyAmount
	ExpectedAmount integrations.MoneyAmount
}

type unitPaymentImportReport struct {
	Assigned int
	Changed  int
	Unclear  int
	Rejected int
	Rows     []unitPaymentImportRow
}

type paymentImportAuditMeta struct {
	TargetID      string
	SourceVersion string
	FileDigest    string
}

func unitPaymentReferenceCandidates(tenantSlug string, period string, units []unit, expected map[string]integrations.MoneyAmount) ([]unitPaymentReferenceCandidate, error) {
	tenantSlug = normalizeSlug(tenantSlug)
	if tenantSlug == "" {
		return nil, fmt.Errorf("tenant required")
	}
	seen := map[string]struct{}{}
	candidates := make([]unitPaymentReferenceCandidate, 0, len(units))
	for _, item := range units {
		unitID := normalizeUnitID(item.ID)
		if unitID == "" {
			continue
		}
		reference, err := integrations.GeneratePaymentReference(integrations.PaymentReferenceInput{
			TenantSlug: tenantSlug,
			Scope:      "unit-payment-status",
			SubjectID:  unitID,
			Period:     period,
		}, seen)
		if err != nil {
			return nil, err
		}
		seen[reference] = struct{}{}
		candidates = append(candidates, unitPaymentReferenceCandidate{
			TenantSlug:     tenantSlug,
			UnitID:         unitID,
			UnitLabel:      item.Label,
			Reference:      reference,
			ExpectedAmount: expected[unitID],
		})
	}
	return candidates, nil
}

func (a *app) applyImportedPaymentsToUnitStatuses(payments []integrations.Payment, candidates []unitPaymentReferenceCandidate, actorEmail string, actorRole string, meta paymentImportAuditMeta) (unitPaymentImportReport, error) {
	report := reconcileImportedPaymentsWithUnitStatus(payments, candidates)
	if a == nil || a.unitPaymentStore == nil {
		return report, fmt.Errorf("unit payment status store not configured")
	}
	for _, row := range report.Rows {
		if row.Decision != unitPaymentImportAssigned {
			continue
		}
		tenantSlug := candidateTenant(candidates, row.Reference)
		if current, ok := a.unitPaymentStore.Get(tenantSlug, row.UnitID); ok && normalizeUnitPaymentStatus(current.Status) == normalizeUnitPaymentStatus(row.Status) {
			continue
		}
		record, err := a.unitPaymentStore.Set(unitPaymentStatus{
			TenantSlug: tenantSlug,
			UnitID:     row.UnitID,
			Status:     row.Status,
			UpdatedBy:  actorEmail,
		})
		if err != nil {
			return report, err
		}
		a.recordAudit(auditEvent{
			TenantSlug: record.TenantSlug,
			ActorEmail: actorEmail,
			ActorRole:  actorRole,
			Action:     auditActionUnitPayment,
			TargetType: "unit",
			TargetID:   record.UnitID,
			Summary:    "Zahlungsstatus aus Import übernommen",
			Details: map[string]string{
				"unit_label": row.UnitLabel,
				"status":     unitPaymentStatusLabel(record.Status),
				"source":     string(integrations.FormatCAMT053),
			},
		})
		report.Changed++
	}
	if tenantSlug := candidatesTenant(candidates); tenantSlug != "" {
		event := auditEvent{
			TenantSlug: tenantSlug,
			ActorEmail: actorEmail,
			ActorRole:  actorRole,
			Action:     auditActionIntegrationImport,
			TargetType: "integration",
			TargetID:   firstNonEmpty(strings.TrimSpace(meta.TargetID), string(integrations.FormatCAMT053)),
			Summary:    "Zahlungsstatus-Import verarbeitet",
			Details: map[string]string{
				"format":         string(integrations.FormatCAMT053),
				"source_version": strings.TrimSpace(meta.SourceVersion),
				"file_digest":    shortImportDigest(meta.FileDigest),
				"assigned":       strconv.Itoa(report.Assigned),
				"changed":        strconv.Itoa(report.Changed),
				"unclear":        strconv.Itoa(report.Unclear),
				"rejected":       strconv.Itoa(report.Rejected),
			},
		}
		if a.auditStore == nil {
			return report, fmt.Errorf("audit store not configured")
		}
		if err := a.auditStore.Append(event); err != nil {
			return report, err
		}
	}
	return report, nil
}

func shortImportDigest(digest string) string {
	digest = strings.TrimSpace(digest)
	if len(digest) > 12 {
		return digest[:12]
	}
	return digest
}

func reconcileImportedPaymentsWithUnitStatus(payments []integrations.Payment, candidates []unitPaymentReferenceCandidate) unitPaymentImportReport {
	byReference := map[string][]unitPaymentReferenceCandidate{}
	for _, candidate := range candidates {
		normalized := integrations.NormalizePaymentReference(candidate.Reference)
		if integrations.ValidatePaymentReference(normalized) != nil {
			continue
		}
		candidate.Reference = normalized
		candidate.TenantSlug = normalizeSlug(candidate.TenantSlug)
		candidate.UnitID = normalizeUnitID(candidate.UnitID)
		byReference[normalized] = append(byReference[normalized], candidate)
	}
	report := unitPaymentImportReport{}
	for _, payment := range payments {
		row := reconcileImportedPayment(payment, byReference)
		report.Rows = append(report.Rows, row)
		switch row.Decision {
		case unitPaymentImportAssigned:
			report.Assigned++
		case unitPaymentImportUnclear:
			report.Unclear++
		default:
			report.Rejected++
		}
	}
	return report
}

func reconcileImportedPayment(payment integrations.Payment, candidates map[string][]unitPaymentReferenceCandidate) unitPaymentImportRow {
	reference := integrations.NormalizePaymentReference(payment.Reference)
	row := unitPaymentImportRow{
		PaymentID: strings.TrimSpace(payment.ExternalID),
		Reference: reference,
		Amount:    payment.Amount,
	}
	if reference == "" {
		row.Decision = unitPaymentImportRejected
		row.Reason = "Zahlungsreferenz fehlt."
		return row
	}
	if integrations.ValidatePaymentReference(reference) != nil {
		row.Decision = unitPaymentImportRejected
		row.Reason = "Zahlungsreferenz ist ungültig."
		return row
	}
	matches := candidates[reference]
	if len(matches) == 0 {
		row.Decision = unitPaymentImportRejected
		row.Reason = "Zahlungsreferenz ist keiner Einheit bekannt."
		return row
	}
	if len(matches) > 1 {
		row.Decision = unitPaymentImportUnclear
		row.Reason = "Zahlungsreferenz ist nicht eindeutig."
		return row
	}
	candidate := matches[0]
	row.UnitID = candidate.UnitID
	row.UnitLabel = candidate.UnitLabel
	row.ExpectedAmount = candidate.ExpectedAmount
	paymentTenant := normalizeSlug(payment.TenantSlug)
	if paymentTenant != "" && candidate.TenantSlug != "" && paymentTenant != candidate.TenantSlug {
		row.Decision = unitPaymentImportRejected
		row.Reason = "Zahlung gehört zu einem anderen Tenant."
		return row
	}
	status, reason, ok := importedPaymentStatusSuggestion(payment.Amount, candidate.ExpectedAmount)
	row.Status = status
	row.Reason = reason
	if !ok {
		row.Decision = unitPaymentImportUnclear
		return row
	}
	row.Decision = unitPaymentImportAssigned
	return row
}

func importedPaymentStatusSuggestion(amount integrations.MoneyAmount, expected integrations.MoneyAmount) (string, string, bool) {
	if err := amount.Validate(); err != nil || amount.Cents <= 0 {
		return "", "Zahlungsbetrag ist ungültig.", false
	}
	if expected.Currency == "" && expected.Cents == 0 {
		return unitPaymentStatusPaid, "Referenz eindeutig, kein Sollbetrag hinterlegt.", true
	}
	if err := expected.Validate(); err != nil || expected.Cents <= 0 {
		return "", "Erwarteter Betrag ist ungültig.", false
	}
	if strings.ToUpper(amount.Currency) != strings.ToUpper(expected.Currency) {
		return "", "Währung stimmt nicht überein.", false
	}
	if amount.Cents >= expected.Cents {
		return unitPaymentStatusPaid, "Betrag deckt den erwarteten Wert.", true
	}
	return unitPaymentStatusPartial, "Betrag ist niedriger als der erwartete Wert.", true
}

func candidateTenant(candidates []unitPaymentReferenceCandidate, reference string) string {
	reference = integrations.NormalizePaymentReference(reference)
	for _, candidate := range candidates {
		if integrations.NormalizePaymentReference(candidate.Reference) == reference {
			return normalizeSlug(candidate.TenantSlug)
		}
	}
	return ""
}

func candidatesTenant(candidates []unitPaymentReferenceCandidate) string {
	tenantSlug := ""
	for _, candidate := range candidates {
		candidateTenantSlug := normalizeSlug(candidate.TenantSlug)
		if candidateTenantSlug == "" {
			continue
		}
		if tenantSlug != "" && candidateTenantSlug != tenantSlug {
			return ""
		}
		tenantSlug = candidateTenantSlug
	}
	return tenantSlug
}

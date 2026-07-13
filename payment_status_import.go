package main

import (
	"fmt"
	"strings"
)

type unitPaymentReferenceCandidate struct {
	TenantSlug     string
	UnitID         string
	UnitLabel      string
	Reference      string
	ExpectedAmount moneyAmount
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
	Amount         moneyAmount
	ExpectedAmount moneyAmount
}

type unitPaymentImportReport struct {
	Assigned int
	Unclear  int
	Rejected int
	Rows     []unitPaymentImportRow
}

func unitPaymentReferenceCandidates(tenantSlug string, period string, units []unit, expected map[string]moneyAmount) ([]unitPaymentReferenceCandidate, error) {
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
		reference, err := generatePaymentReference(paymentReferenceInput{
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

func (a *app) applyImportedPaymentsToUnitStatuses(payments []canonicalPayment, candidates []unitPaymentReferenceCandidate, actorEmail string, actorRole string) (unitPaymentImportReport, error) {
	report := reconcileImportedPaymentsWithUnitStatus(payments, candidates)
	if a == nil || a.unitPaymentStore == nil {
		return report, fmt.Errorf("unit payment status store not configured")
	}
	for _, row := range report.Rows {
		if row.Decision != unitPaymentImportAssigned {
			continue
		}
		record, err := a.unitPaymentStore.Set(unitPaymentStatus{
			TenantSlug: candidateTenant(candidates, row.Reference),
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
				"source":     string(integrationFormatCAMT053),
			},
		})
	}
	return report, nil
}

func reconcileImportedPaymentsWithUnitStatus(payments []canonicalPayment, candidates []unitPaymentReferenceCandidate) unitPaymentImportReport {
	byReference := map[string][]unitPaymentReferenceCandidate{}
	for _, candidate := range candidates {
		normalized := normalizePaymentReference(candidate.Reference)
		if validatePaymentReference(normalized) != nil {
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

func reconcileImportedPayment(payment canonicalPayment, candidates map[string][]unitPaymentReferenceCandidate) unitPaymentImportRow {
	reference := normalizePaymentReference(payment.Reference)
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
	if validatePaymentReference(reference) != nil {
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

func importedPaymentStatusSuggestion(amount moneyAmount, expected moneyAmount) (string, string, bool) {
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
	reference = normalizePaymentReference(reference)
	for _, candidate := range candidates {
		if normalizePaymentReference(candidate.Reference) == reference {
			return normalizeSlug(candidate.TenantSlug)
		}
	}
	return ""
}

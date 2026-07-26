package server

import (
	"strings"
	"testing"

	"github.com/markus-barta/hausv-org/internal/integrations"
)

func TestApplyImportedPaymentsToUnitStatusesReportsAndAppliesOnlyClearMatches(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	units := []unit{
		{ID: "top-1", TenantSlug: "jhw22", Label: "Top 1", UnitType: unitTypeResidential},
		{ID: "top-2", TenantSlug: "jhw22", Label: "Top 2", UnitType: unitTypeResidential},
		{ID: "top-3", TenantSlug: "jhw22", Label: "Top 3", UnitType: unitTypeResidential},
	}
	if err := a.unitStore.SetTenantUnits("jhw22", units); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	candidates, err := unitPaymentReferenceCandidates("jhw22", "2026-07", units, map[string]integrations.MoneyAmount{
		"top-1": {Currency: "EUR", Cents: 10000},
		"top-2": {Currency: "EUR", Cents: 10000},
		"top-3": {Currency: "EUR", Cents: 10000},
	})
	if err != nil {
		t.Fatalf("unitPaymentReferenceCandidates: %v", err)
	}
	duplicateReference := candidates[2].Reference
	candidates = append(candidates, unitPaymentReferenceCandidate{
		TenantSlug:     "jhw22",
		UnitID:         "top-duplicate",
		UnitLabel:      "Top Duplicate",
		Reference:      duplicateReference,
		ExpectedAmount: integrations.MoneyAmount{Currency: "EUR", Cents: 10000},
	})
	payments := []integrations.Payment{
		{TenantSlug: "jhw22", ExternalID: "p-1", Reference: candidates[0].Reference, Amount: integrations.MoneyAmount{Currency: "EUR", Cents: 10000}},
		{TenantSlug: "jhw22", ExternalID: "p-2", Reference: candidates[1].Reference, Amount: integrations.MoneyAmount{Currency: "EUR", Cents: 6000}},
		{TenantSlug: "jhw22", ExternalID: "p-3", Reference: duplicateReference, Amount: integrations.MoneyAmount{Currency: "EUR", Cents: 10000}},
		{TenantSlug: "jhw22", ExternalID: "p-4", Amount: integrations.MoneyAmount{Currency: "EUR", Cents: 10000}},
		{TenantSlug: "jhw22", ExternalID: "p-5", Reference: "HV-JHW22-UNKNOWN-123", Amount: integrations.MoneyAmount{Currency: "EUR", Cents: 10000}},
	}

	report, err := a.applyImportedPaymentsToUnitStatuses(payments, candidates, "manager@example.com", roleManager)
	if err != nil {
		t.Fatalf("applyImportedPaymentsToUnitStatuses: %v", err)
	}
	if report.Assigned != 2 || report.Unclear != 1 || report.Rejected != 2 || len(report.Rows) != 5 {
		t.Fatalf("report = %+v", report)
	}
	if report.Rows[0].Status != unitPaymentStatusPaid || report.Rows[1].Status != unitPaymentStatusPartial || report.Rows[2].Decision != unitPaymentImportUnclear {
		t.Fatalf("unexpected report rows = %+v", report.Rows)
	}
	for _, row := range report.Rows {
		if strings.Contains(row.Reason, "Debtor") || strings.Contains(row.Reason, "IBAN") {
			t.Fatalf("report row leaks debtor data: %+v", row)
		}
	}

	top1, ok := a.unitPaymentStore.Get("jhw22", "top-1")
	if !ok || top1.Status != unitPaymentStatusPaid {
		t.Fatalf("top-1 status = %+v ok=%v", top1, ok)
	}
	top2, ok := a.unitPaymentStore.Get("jhw22", "top-2")
	if !ok || top2.Status != unitPaymentStatusPartial {
		t.Fatalf("top-2 status = %+v ok=%v", top2, ok)
	}
	if _, ok := a.unitPaymentStore.Get("jhw22", "top-3"); ok {
		t.Fatalf("unclear duplicate match should not update top-3")
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "jhw22", Action: auditActionUnitPayment, Limit: 10})
	if len(events) != 2 {
		t.Fatalf("unit payment import audit events = %+v", events)
	}
	importEvents := a.auditStore.List(auditFilter{TenantSlug: "jhw22", Action: auditActionIntegrationImport, Limit: 10})
	if len(importEvents) != 1 {
		t.Fatalf("integration import audit events = %+v", importEvents)
	}
	importEvent := importEvents[0]
	if importEvent.Details["assigned"] != "2" || importEvent.Details["unclear"] != "1" || importEvent.Details["rejected"] != "2" {
		t.Fatalf("integration import counts = %+v", importEvent)
	}
	haystack := strings.Join(append([]string{importEvent.TargetID, importEvent.Summary}, auditDetailValues(importEvent.Details)...), " ")
	for _, forbidden := range []string{"p-1", candidates[0].Reference, "10000", "6000"} {
		if strings.Contains(haystack, forbidden) {
			t.Fatalf("integration audit leaks payment data %q: %+v", forbidden, importEvent)
		}
	}
}

package main

import (
	"strings"
	"testing"
)

func TestGeneratePaymentReferenceIsReadableValidAndShort(t *testing.T) {
	reference, err := generatePaymentReference(paymentReferenceInput{
		TenantSlug: "Jänischhofweg 22 / Graz",
		Scope:      "parking",
		SubjectID:  "top-11",
		Period:     "2026-06",
	}, nil)
	if err != nil {
		t.Fatalf("generatePaymentReference: %v", err)
	}
	if err := validatePaymentReference(reference); err != nil {
		t.Fatalf("validatePaymentReference(%q): %v", reference, err)
	}
	if len(reference) > paymentReferenceMaxLength {
		t.Fatalf("reference len = %d, want <= %d: %q", len(reference), paymentReferenceMaxLength, reference)
	}
	if !strings.HasPrefix(reference, "HV-JAENISCHHO-202606-") {
		t.Fatalf("reference = %q, want readable tenant and period", reference)
	}
}

func TestValidatePaymentReferenceRejectsSpecialCharactersAndLength(t *testing.T) {
	for _, bad := range []string{
		"",
		"hv-JHW22-202606-ABC",
		"HV-JHW22-202606-AB C",
		"HV-JHW22-202606-ÄBC",
		"HV-JHW22--202606-ABC",
		"HV-JHW22-202606-ABCDEFGHIJKLMNOPQRSTUVWXYZ",
	} {
		if err := validatePaymentReference(bad); err == nil {
			t.Fatalf("validatePaymentReference(%q) succeeded, want error", bad)
		}
	}
	for _, good := range []string{
		"HV-JHW22-202606-ABC123",
		"HV-HAUS10-GEN-Z9Y8X7W6V5",
	} {
		if err := validatePaymentReference(good); err != nil {
			t.Fatalf("validatePaymentReference(%q): %v", good, err)
		}
	}
}

func TestGeneratePaymentReferenceAvoidsTenantCollisions(t *testing.T) {
	input := paymentReferenceInput{
		TenantSlug: "jhw22",
		Scope:      "parking",
		SubjectID:  "top-11",
		Period:     "2026-06",
	}
	first, err := generatePaymentReference(input, nil)
	if err != nil {
		t.Fatalf("first reference: %v", err)
	}
	second, err := generatePaymentReference(input, map[string]struct{}{first: {}})
	if err != nil {
		t.Fatalf("second reference: %v", err)
	}
	if first == second {
		t.Fatalf("collision not avoided: %q", first)
	}
	if err := validatePaymentReference(second); err != nil {
		t.Fatalf("second reference invalid: %v", err)
	}
}

func TestPaymentReferenceNormalizesWhitespaceOnlyForLookup(t *testing.T) {
	if got := normalizePaymentReference(" hv-jhw22-202606-abc123 "); got != "HV-JHW22-202606-ABC123" {
		t.Fatalf("normalizePaymentReference = %q", got)
	}
	if err := validatePaymentReference(normalizePaymentReference(" hv-jhw22-202606-abc123 ")); err != nil {
		t.Fatalf("normalized reference should validate: %v", err)
	}
}

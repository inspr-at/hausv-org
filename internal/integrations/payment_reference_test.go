package integrations

import (
	"strings"
	"testing"
)

func TestGeneratePaymentReferenceIsReadableValidAndShort(t *testing.T) {
	reference, err := GeneratePaymentReference(PaymentReferenceInput{
		TenantSlug: "Musterweg 1 / Wien",
		Scope:      "parking",
		SubjectID:  "einheit-12",
		Period:     "2026-06",
	}, nil)
	if err != nil {
		t.Fatalf("GeneratePaymentReference: %v", err)
	}
	if err := ValidatePaymentReference(reference); err != nil {
		t.Fatalf("ValidatePaymentReference(%q): %v", reference, err)
	}
	if len(reference) > paymentReferenceMaxLength {
		t.Fatalf("reference len = %d, want <= %d: %q", len(reference), paymentReferenceMaxLength, reference)
	}
	if !strings.HasPrefix(reference, "HV-MUSTERWEG1-202606-") {
		t.Fatalf("reference = %q, want readable tenant and period", reference)
	}
}

func TestValidatePaymentReferenceRejectsSpecialCharactersAndLength(t *testing.T) {
	for _, bad := range []string{
		"",
		"hv-DEMO-202606-ABC",
		"HV-DEMO-202606-AB C",
		"HV-DEMO-202606-ÄBC",
		"HV-DEMO--202606-ABC",
		"HV-DEMO-202606-ABCDEFGHIJKLMNOPQRSTUVWXYZ",
	} {
		if err := ValidatePaymentReference(bad); err == nil {
			t.Fatalf("ValidatePaymentReference(%q) succeeded, want error", bad)
		}
	}
	for _, good := range []string{
		"HV-DEMO-202606-ABC123",
		"HV-HAUS10-GEN-Z9Y8X7W6V5",
	} {
		if err := ValidatePaymentReference(good); err != nil {
			t.Fatalf("ValidatePaymentReference(%q): %v", good, err)
		}
	}
}

func TestGeneratePaymentReferenceAvoidsTenantCollisions(t *testing.T) {
	input := PaymentReferenceInput{
		TenantSlug: "demo",
		Scope:      "parking",
		SubjectID:  "einheit-12",
		Period:     "2026-06",
	}
	first, err := GeneratePaymentReference(input, nil)
	if err != nil {
		t.Fatalf("first reference: %v", err)
	}
	second, err := GeneratePaymentReference(input, map[string]struct{}{first: {}})
	if err != nil {
		t.Fatalf("second reference: %v", err)
	}
	if first == second {
		t.Fatalf("collision not avoided: %q", first)
	}
	if err := ValidatePaymentReference(second); err != nil {
		t.Fatalf("second reference invalid: %v", err)
	}
}

func TestPaymentReferenceNormalizesWhitespaceOnlyForLookup(t *testing.T) {
	if got := NormalizePaymentReference(" hv-demo-202606-abc123 "); got != "HV-DEMO-202606-ABC123" {
		t.Fatalf("NormalizePaymentReference = %q", got)
	}
	if err := ValidatePaymentReference(NormalizePaymentReference(" hv-demo-202606-abc123 ")); err != nil {
		t.Fatalf("normalized reference should validate: %v", err)
	}
}

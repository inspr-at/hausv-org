package main

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"strings"
)

const (
	paymentReferencePrefix    = "HV"
	paymentReferenceMaxLength = 35
	paymentReferenceHashLen   = 10
)

type paymentReferenceInput struct {
	TenantSlug string
	Scope      string
	SubjectID  string
	Period     string
}

func generatePaymentReference(input paymentReferenceInput, existing map[string]struct{}) (string, error) {
	tenantToken, err := paymentReferenceTenantToken(input.TenantSlug)
	if err != nil {
		return "", err
	}
	periodToken := paymentReferencePeriodToken(input.Period)
	normalizedExisting := map[string]struct{}{}
	for reference := range existing {
		normalized := normalizePaymentReference(reference)
		if normalized != "" {
			normalizedExisting[normalized] = struct{}{}
		}
	}
	for counter := 0; counter < 1024; counter++ {
		hashToken := paymentReferenceHash(input, counter)
		reference := paymentReferencePrefix + "-" + tenantToken + "-" + periodToken + "-" + hashToken
		if err := validatePaymentReference(reference); err != nil {
			return "", err
		}
		if _, exists := normalizedExisting[reference]; !exists {
			return reference, nil
		}
	}
	return "", fmt.Errorf("could not generate unique payment reference")
}

func validatePaymentReference(reference string) error {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return fmt.Errorf("payment reference required")
	}
	if len(reference) > paymentReferenceMaxLength {
		return fmt.Errorf("payment reference exceeds %d characters", paymentReferenceMaxLength)
	}
	if reference != strings.ToUpper(reference) {
		return fmt.Errorf("payment reference must be uppercase")
	}
	if strings.HasPrefix(reference, "-") || strings.HasSuffix(reference, "-") || strings.Contains(reference, "--") {
		return fmt.Errorf("payment reference has invalid separators")
	}
	for _, r := range reference {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return fmt.Errorf("payment reference contains unsupported character %q", r)
	}
	return nil
}

func normalizePaymentReference(reference string) string {
	reference = strings.ToUpper(strings.TrimSpace(reference))
	reference = strings.Join(strings.Fields(reference), "")
	return reference
}

func paymentReferenceTenantToken(tenantSlug string) (string, error) {
	token := paymentReferenceASCII(tenantSlug)
	if token == "" {
		return "", fmt.Errorf("tenant token required")
	}
	if len(token) > 10 {
		token = token[:10]
	}
	return token, nil
}

func paymentReferencePeriodToken(period string) string {
	token := paymentReferenceASCII(period)
	if token == "" {
		return "GEN"
	}
	if len(token) > 8 {
		token = token[:8]
	}
	return token
}

func paymentReferenceASCII(raw string) string {
	replacer := strings.NewReplacer(
		"ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss",
		"Ä", "AE", "Ö", "OE", "Ü", "UE",
	)
	raw = strings.ToUpper(replacer.Replace(strings.TrimSpace(raw)))
	var b strings.Builder
	for _, r := range raw {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func paymentReferenceHash(input paymentReferenceInput, counter int) string {
	raw := strings.Join([]string{
		normalizeSlug(input.TenantSlug),
		strings.ToLower(strings.TrimSpace(input.Scope)),
		strings.ToLower(strings.TrimSpace(input.SubjectID)),
		strings.ToLower(strings.TrimSpace(input.Period)),
		fmt.Sprintf("%d", counter),
	}, "\x1f")
	sum := sha256.Sum256([]byte(raw))
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:])
	if len(encoded) > paymentReferenceHashLen {
		return encoded[:paymentReferenceHashLen]
	}
	return encoded
}

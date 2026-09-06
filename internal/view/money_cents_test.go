package view

import "testing"

func TestFormatEURCentsPreservesIntegerPrecision(t *testing.T) {
	for cents, want := range map[int64]string{0: "0,00 €", 1: "0,01 €", -1: "-0,01 €", 123456: "1.234,56 €", 9223372036854775807: "92.233.720.368.547.758,07 €", -9223372036854775808: "-92.233.720.368.547.758,08 €"} {
		if got := FormatEURCents(cents); got != want {
			t.Errorf("%d: got %s want %s", cents, got, want)
		}
	}
}

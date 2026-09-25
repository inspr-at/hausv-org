package store

import (
	"sort"
	"testing"
)

func TestUnitLabelTextKeepsTheNumberWithItsName(t *testing.T) {
	if got := UnitLabelText("Top 3"); got != "Top\u00a03" {
		t.Fatalf("Top 3 = %q", got)
	}
	if got := UnitLabelText("Stellplatz 11"); got != "Stellplatz\u00a011" {
		t.Fatalf("Stellplatz 11 = %q", got)
	}
	if got := UnitLabelText("Keller"); got != "Keller" {
		t.Fatalf("Keller = %q", got)
	}
}

func TestUnitLabelLessOrdersNumbersLikeAPersonReadsThem(t *testing.T) {
	labels := []string{"Top 10", "Top 2", "Stellplatz 11", "Top 1", "Stellplatz 2", "Top 21", "Keller"}
	sort.Slice(labels, func(i, j int) bool { return UnitLabelLess(labels[i], labels[j]) })
	want := []string{"Keller", "Stellplatz 2", "Stellplatz 11", "Top 1", "Top 2", "Top 10", "Top 21"}
	for i := range want {
		if labels[i] != want[i] {
			t.Fatalf("sorted %v, want %v", labels, want)
		}
	}
}

func TestUnitLabelLessHandlesEdgeCases(t *testing.T) {
	cases := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"leading zeros do not change the number", "Top 007", "Top 8", true},
		{"equal numbers fall back to the rest", "Top 3a", "Top 3b", true},
		{"case is ignored", "top 2", "Top 10", true},
		{"prefix is shorter", "Top", "Top 1", true},
		{"long numbers stay numeric", "Top 9999999999999999999", "Top 10000000000000000000", true},
		{"identical labels are not less", "Top 1", "Top 1", false},
		{"empty label sorts first", "", "Top 1", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := UnitLabelLess(tc.a, tc.b); got != tc.want {
				t.Fatalf("UnitLabelLess(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
			if tc.a != tc.b && UnitLabelLess(tc.b, tc.a) == tc.want {
				t.Fatalf("UnitLabelLess is not antisymmetric for %q and %q", tc.a, tc.b)
			}
		})
	}
}

func TestUnitLessUsesNaturalLabelOrder(t *testing.T) {
	units := []Unit{
		{TenantSlug: "haus", Label: "Top 10", ID: "u10"},
		{TenantSlug: "haus", Label: "Top 2", ID: "u2"},
		{TenantSlug: "haus", Label: "Top 1", ID: "u1"},
	}
	sort.Slice(units, func(i, j int) bool { return UnitLess(units[i], units[j]) })
	want := []string{"Top 1", "Top 2", "Top 10"}
	for i := range want {
		if units[i].Label != want[i] {
			t.Fatalf("got %q at %d, want %q", units[i].Label, i, want[i])
		}
	}
}

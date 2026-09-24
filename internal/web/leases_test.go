package web

import "testing"

func TestLeaseIndexLine(t *testing.T) {
	got := LeaseIndexLine("vpi2020", "2024-09", "123.6")
	want := "VPI 2020 · Basis September 2024 = 123,6"
	if got != want {
		t.Fatalf("index line %q, want %q", got, want)
	}
	if LeaseSeriesLabel("vpi1996") != "VPI 1996" {
		t.Fatal("series label")
	}
	if LeaseMonthName("2024-01") != "Jänner 2024" {
		t.Fatal("month name")
	}
	if LeaseGrossCents(100000, 1000) != 110000 {
		t.Fatal("gross")
	}
}

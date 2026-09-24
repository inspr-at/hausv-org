package indexation

import (
	"fmt"
	"strings"
	"testing"
)

func TestImportOfficialFormats(t *testing.T) {
	for _, dimension := range []string{"C-VPI1-0", "C-VPI5-0", "C-VPI5NEU-0", "C-VPICOICOP18_5-0"} {
		code := "VPI-0"
		if strings.Contains(dimension, "COICOP18") {
			code = "VPICOICOP18-0"
		}
		// Official column names, period syntax and decimal-comma values.
		csv := fmt.Sprintf("\ufeffC-VPIZR-0;%s;F-VPIMZBM;F-VPIMZVM\r\nVPIZR-2025;%s;128,20000;0,00000\r\nVPIZR-202607;%s;132,20000;132,20000\r\nVPIZR-202608;%s;132,90000;132,20000\r\nVPIZR-202608;VPI-01;200,00000;1,00000\r\n", dimension, code, code, code)
		got, err := ImportOGD(strings.NewReader(csv), ImportOptions{Series: VPI2020, Source: "https://data.statistik.gv.at/test.csv", FetchedAt: date("2026-09-24"), FinalThrough: "2026-07"})
		if err != nil || len(got.Values) != 2 || len(got.Annual) != 1 {
			t.Fatalf("format %s: %+v %v", dimension, got, err)
		}
		if got.Values[0].Preliminary || !got.Values[1].Preliminary || got.Values[1].Value != dec(t, "132.9") || got.Annual[0].Value != dec(t, "128.2") {
			t.Fatalf("incorrect import: %+v", got)
		}
	}
}

func TestImportFailsClosed(t *testing.T) {
	header := "C-VPIZR-0;C-VPI1-0;F-VPIMZBM\n"
	opts := ImportOptions{Series: VPI2020, Source: "official", FetchedAt: date("2026-09-24"), FinalThrough: "2026-07"}
	for _, csv := range []string{
		"", "period;value\n2026-01;100\n", header,
		header + "VPIZR-202613;VPI-0;100\n",
		header + "VPIZR-202601;VPI-0;NaN\n",
		header + "VPIZR-202601;VPI-0;0\n",
		header + "VPIZR-202601;VPI-0;100\nVPIZR-202601;VPI-0;101\n",
		header + "VPIZR-202601;VPI-0\n",
		header + "VPIZR-202610;VPI-0;100\n",
		header + "VPIZR-2026Q1;VPI-0;100\n",
		header + "202601;VPI-0;100\n",
	} {
		if _, err := ImportOGD(strings.NewReader(csv), opts); err == nil {
			t.Fatalf("malformed CSV accepted: %q", csv)
		}
	}
	opts.FinalThrough = ""
	if _, err := ImportOGD(strings.NewReader(header+"VPIZR-202601;VPI-0;100\n"), opts); err == nil {
		t.Fatal("unknown finality accepted")
	}
}

func TestSnapshotCoverageProvenanceAndFinality(t *testing.T) {
	s, err := LoadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if s.Manifest.Version != "260924101853.0.0" || s.Manifest.License != "CC BY 4.0" || s.Manifest.FinalThrough != "2026-07" || len(s.Manifest.Sources) != 10 {
		t.Fatalf("manifest: %+v", s.Manifest)
	}
	if len(s.Data.Values()) != 3092 || len(s.Annual) != 251 {
		t.Fatalf("snapshot row counts: %d monthly, %d annual", len(s.Data.Values()), len(s.Annual))
	}
	for _, tc := range []struct {
		series Series
		start  Month
		august string
	}{
		{VPI1966, "1967-01", "723.4"}, {VPI1976, "1977-01", "412.2"}, {VPI1986, "1987-01", "265.2"}, {VPI1996, "1997-01", "202.8"}, {VPI2000, "2001-01", "192.7"},
		{VPI2005, "2006-01", "174.4"}, {VPI2010, "2011-01", "159.2"}, {VPI2015, "2016-01", "143.8"}, {VPI2020, "2021-01", "132.9"}, {VPI2025, "2026-01", "103.7"},
	} {
		for month := tc.start; month <= "2026-08"; month = month.next() {
			v, ok, err := s.Data.Lookup(tc.series, month)
			if err != nil || !ok || v.ChainSource != "" || v.Source == "" || v.FetchedAt.IsZero() {
				t.Fatalf("missing published %s %s: %+v %v", tc.series, month, v, err)
			}
			if v.Preliminary != (month == "2026-08") {
				t.Fatalf("incorrect finality: %+v", v)
			}
			if month == "2026-08" && v.Value != dec(t, tc.august) {
				t.Fatalf("August %s: %v", tc.series, v.Value)
			}
		}
	}
	values := s.Data.Values()
	values[0].Value = 1
	copy, err := LoadSnapshot()
	if err != nil || copy.Data.Values()[0].Value == 1 {
		t.Fatal("snapshot shared mutable observations")
	}
}

func TestPublishedChainingAndExplicitFallback(t *testing.T) {
	s, err := LoadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	jan, _, _ := s.Data.Lookup(VPI2025, "2026-01")
	july, _, _ := s.Data.Lookup(VPI2025, "2026-07")
	d, err := NewDataset([]IndexValue{jan, july}, s.Manifest.Factors)
	if err != nil {
		t.Fatal(err)
	}
	chained, ok, err := d.Lookup(VPI2010, "2026-07")
	// Official factor 1.535 * 103.1 = 158.2585 -> 158.3.
	if err != nil || !ok || chained.Value != dec(t, "158.3") || chained.ChainSource == "" {
		t.Fatalf("chained: %+v %v", chained, err)
	}
	if _, ok, err := d.Lookup(VPI2010, "2025-12"); err != nil || ok {
		t.Fatal("2025 factors applied retroactively")
	}
	// Direct published values retain priority, even if preliminary. Do not mask
	// revisions by replacing them with a mathematically derived observation.
	published := IndexValue{Series: VPI2010, Month: "2026-07", Value: dec(t, "158.4"), Preliminary: true}
	d, _ = NewDataset([]IndexValue{july, published}, s.Manifest.Factors)
	v, _, _ := d.Lookup(VPI2010, "2026-07")
	if v.Value != published.Value || !v.Preliminary || v.ChainSource != "" {
		t.Fatal("published value was overwritten")
	}
	var values []IndexValue
	for _, v := range s.Data.Values() {
		if v.Series == VPI2025 {
			values = append(values, v)
		}
	}
	d, _ = NewDataset(values, s.Manifest.Factors)
	base, _, _ := d.Lookup(VPI2010, "2026-01")
	clause := Clause{Series: VPI2010, BaseMonth: "2026-01", BaseValue: base.Value, AmountCents: 100000, ThresholdKind: PercentThreshold, Threshold: Unit}
	if _, err := Evaluate(clause, d, "2026-07"); err == nil {
		t.Fatal("derived index silently authorized")
	}
	clause.AllowDerived = true
	result, err := Evaluate(clause, d, "2026-07")
	if err != nil || !result.Crossed || result.Explanation[0].ChainSource == "" {
		t.Fatalf("explicit chain: %+v %v", result, err)
	}
	if _, err := NewDataset([]IndexValue{jan, jan}, nil); err == nil {
		t.Fatal("duplicate accepted")
	}
}

func TestStatutoryRates(t *testing.T) {
	states := []FederalState{Burgenland, Carinthia, LowerAustria, UpperAustria, Salzburg, Styria, Tyrol, Vorarlberg, Vienna}
	for _, tc := range []struct {
		on   string
		want []int64
	}{
		{"2023-04-01", []int64{609, 781, 685, 723, 922, 921, 814, 1025, 667}},
		{"2024-04-01", []int64{609, 781, 685, 723, 922, 921, 814, 1025, 667}},
		{"2025-04-01", []int64{609, 781, 685, 723, 922, 921, 814, 1025, 667}},
		{"2026-04-01", []int64{615, 789, 692, 730, 931, 930, 822, 1035, 674}},
	} {
		for i, state := range states {
			got, err := Richtwert(state, date(tc.on))
			if err != nil || got.CentsPerM2 != tc.want[i] || got.Source == "" {
				t.Fatalf("%s on %s: %+v %v", state, tc.on, got, err)
			}
		}
	}
	for _, tc := range []struct {
		on   string
		want []int64
	}{
		{"2023-04-01", []int64{423, 318, 212, 212, 106}},
		{"2023-07-01", []int64{447, 335, 223, 223, 112}},
		{"2024-04-01", []int64{447, 335, 223, 223, 112}},
		{"2025-04-01", []int64{447, 335, 223, 223, 112}},
		{"2026-04-01", []int64{451, 338, 225, 225, 113}},
	} {
		for i, category := range []Category{CategoryA, CategoryB, CategoryC, CategoryDUsable, CategoryDUnusable} {
			got, err := CategoryAmount(category, date(tc.on))
			if err != nil || got.CentsPerM2 != tc.want[i] {
				t.Fatalf("%s on %s: %+v %v", category, tc.on, got, err)
			}
		}
	}
	if _, err := Richtwert(Vienna, date("2027-04-01")); err == nil {
		t.Fatal("unverified future rate accepted")
	}
	if _, err := CategoryAmount("D", date("2026-04-01")); err == nil {
		t.Fatal("ambiguous category D accepted")
	}
}

package indexation

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type fixtureHTTP func(*http.Request) (*http.Response, error)

func (f fixtureHTTP) Do(r *http.Request) (*http.Response, error) { return f(r) }

const releaseCSV = "C-VPIZR-0;C-VPICOICOP18_5-0;F-VPIMZBM\nVPIZR-202607;VPICOICOP18-0;132,2\nVPIZR-202608;VPICOICOP18-0;132,9\nVPIZR-2025;VPICOICOP18-0;128,1\n"
const releaseLabels = "code;name;;en_name;de_desc;de_link;en_desc;en_link;de_syn;en_syn\nVPIZR-202607;Jul.26;;July 2026;;;;;;\nVPIZR-202608;Aug.26 (vorl.);;August 2026 (prel.);;;;;;\nVPIZR-2025;Jahresdurchschnitt 2025;;annual average 2025;;;;;;\n"

func TestFetchOGDUsesBoundedClientAndOfficialStatus(t *testing.T) {
	source := SnapshotSource{Series: VPI2020, URL: "https://data.statistik.gv.at/data/fixture.csv"}
	calls := 0
	client := fixtureHTTP(func(r *http.Request) (*http.Response, error) {
		calls++
		if _, ok := r.Context().Deadline(); !ok {
			t.Fatal("missing timeout")
		}
		body := releaseCSV
		if r.URL.String() == PeriodsURL(source.URL) {
			body = releaseLabels
		} else if r.URL.String() != source.URL {
			t.Fatalf("unexpected URL %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	got, err := FetchOGD(t.Context(), client, source, time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC))
	if err != nil || calls != 3 {
		t.Fatalf("fetch: %v calls=%d", err, calls)
	}
	if got.Imported.Values[0].Preliminary || !got.Imported.Values[1].Preliminary || got.Imported.Annual[0].Preliminary {
		t.Fatalf("wrong statuses: %+v", got.Imported)
	}
	final, err := ParseOGDRelease([]byte(releaseCSV), []byte(strings.ReplaceAll(releaseLabels, " (vorl.)", "")), source, got.FetchedAt)
	if err != nil || final.SHA256 == got.SHA256 || final.DataSHA256 != got.DataSHA256 || final.Imported.Values[1].Preliminary {
		t.Fatal("status-only release was not distinguished", err)
	}
}

func TestOGDReleaseFailsClosed(t *testing.T) {
	source := SnapshotSource{Series: VPI2020, URL: "https://data.statistik.gv.at/data/fixture.csv"}
	now := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	for _, labels := range []string{
		"code;name\nVPIZR-202607;Jul.26\n",
		strings.ReplaceAll(releaseLabels, "Aug.26 (vorl.)", "unbekannt"),
		strings.ReplaceAll(releaseLabels, "Jul.26", "Jun.26"),
		releaseLabels + "VPIZR-202607;Jul.26;;;;;;;;\n",
	} {
		if _, err := ParseOGDRelease([]byte(releaseCSV), []byte(labels), source, now); err == nil {
			t.Fatal("accepted unknown, missing, mismatched or duplicate status")
		}
	}
	if _, err := ParseOGDRelease([]byte(releaseCSV+"VPIZR-202608;VPICOICOP18-0;132,9\n"), []byte(releaseLabels), source, now); err == nil {
		t.Fatal("accepted duplicate value")
	}
	for _, tc := range []struct {
		name   string
		client fixtureHTTP
	}{
		{"http", func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader("unavailable"))}, nil
		}},
		{"size_header", func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, ContentLength: MaxOGDBytes + 1, Body: io.NopCloser(strings.NewReader(""))}, nil
		}},
		{"size_stream", func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, ContentLength: -1, Body: io.NopCloser(io.LimitReader(repeatingReader{}, MaxOGDBytes+1))}, nil
		}},
		{"network", func(*http.Request) (*http.Response, error) { return nil, fmt.Errorf("offline") }},
		{"cancel", func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			if tc.name == "cancel" {
				cancel()
			}
			defer cancel()
			if _, err := FetchOGD(ctx, tc.client, source, now); err == nil {
				t.Fatal("accepted failed fetch")
			}
		})
	}
}

type repeatingReader struct{}

func (repeatingReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	return len(p), nil
}

func TestFetchOGDRejectsChangingStatusPublication(t *testing.T) {
	calls := 0
	client := fixtureHTTP(func(r *http.Request) (*http.Response, error) {
		calls++
		body := releaseCSV
		if strings.HasSuffix(r.URL.Path, "_C-VPIZR-0.csv") {
			body = releaseLabels
			if calls == 3 {
				body = strings.ReplaceAll(body, " (vorl.)", "")
			}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	_, err := FetchOGD(t.Context(), client, SnapshotSource{Series: VPI2020, URL: "https://data.statistik.gv.at/data/fixture.csv"}, time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC))
	if err == nil || !strings.Contains(err.Error(), "status changed") {
		t.Fatal("mixed publication accepted", err)
	}
}

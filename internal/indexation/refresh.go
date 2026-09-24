package indexation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const MaxOGDBytes = 32 << 20

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// FetchedOGD binds the values and the period labels into one content identity.
// The data CSV alone does not encode finality: identical numbers can become final.
type FetchedOGD struct {
	URL, SHA256, DataSHA256, StatusSHA256 string
	FetchedAt                             time.Time
	Imported                              Imported
}

func PeriodsURL(source string) string {
	return strings.TrimSuffix(source, ".csv") + "_C-VPIZR-0.csv"
}

func FetchOGD(ctx context.Context, client HTTPDoer, source SnapshotSource, now time.Time) (FetchedOGD, error) {
	// Sources come from the pinned manifest, never from a submitted URL.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	labels, err := fetchOGDBytes(ctx, client, PeriodsURL(source.URL))
	if err != nil {
		return FetchedOGD{}, err
	}
	data, err := fetchOGDBytes(ctx, client, source.URL)
	if err != nil {
		return FetchedOGD{}, err
	}
	// Refuse a status publication that changed while its numbers were fetched.
	// Otherwise a release boundary could mark the old preliminary number final.
	confirmedLabels, err := fetchOGDBytes(ctx, client, PeriodsURL(source.URL))
	if err != nil {
		return FetchedOGD{}, err
	}
	if !bytes.Equal(labels, confirmedLabels) {
		return FetchedOGD{}, fmt.Errorf("OGD status changed during fetch; retry required")
	}
	return ParseOGDRelease(data, labels, source, now)
}

func fetchOGDBytes(ctx context.Context, client HTTPDoer, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OGD fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OGD HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > MaxOGDBytes {
		return nil, fmt.Errorf("OGD exceeds size limit")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxOGDBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxOGDBytes {
		return nil, fmt.Errorf("OGD exceeds size limit")
	}
	return data, nil
}

var monthlyLabel = regexp.MustCompile(`^(Jän|Feb|Mär|Apr|Mai|Jun|Jul|Aug|Sep|Okt|Nov|Dez)\.[0-9]{2}$`)

// ParseOGDRelease fails closed on missing/unknown status labels and validates
// the label against its period. No status is inferred from the wall clock.
func ParseOGDRelease(data, labels []byte, source SnapshotSource, now time.Time) (FetchedOGD, error) {
	r := csv.NewReader(strings.NewReader(string(labels)))
	r.Comma = ';'
	header, err := r.Read()
	if err != nil || len(header) < 2 || strings.TrimPrefix(header[0], "\ufeff") != "code" || header[1] != "name" {
		return FetchedOGD{}, fmt.Errorf("unsupported OGD period labels")
	}
	statuses := map[string]bool{}
	months := []string{"Jän", "Feb", "Mär", "Apr", "Mai", "Jun", "Jul", "Aug", "Sep", "Okt", "Nov", "Dez"}
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return FetchedOGD{}, fmt.Errorf("OGD period labels: %w", err)
		}
		code, label := row[0], row[1]
		if _, ok := statuses[code]; ok {
			return FetchedOGD{}, fmt.Errorf("duplicate OGD period label")
		}
		preliminary := strings.HasSuffix(label, " (vorl.)")
		label = strings.TrimSuffix(label, " (vorl.)")
		valid := false
		if strings.HasPrefix(code, "VPIZR-") {
			period := strings.TrimPrefix(code, "VPIZR-")
			if len(period) == 4 {
				valid = label == "Jahresdurchschnitt "+period
			} else if len(period) == 6 && monthlyLabel.MatchString(label) {
				date, e := time.Parse("200601", period)
				valid = e == nil && label == months[int(date.Month())-1]+"."+period[2:4]
			}
		}
		if !valid {
			return FetchedOGD{}, fmt.Errorf("unknown OGD period label for %s", code)
		}
		statuses[code] = preliminary
	}
	values, err := ImportOGD(strings.NewReader(string(data)), ImportOptions{Series: source.Series, Source: source.URL, FetchedAt: now, FinalThrough: "0001-01"})
	if err != nil {
		return FetchedOGD{}, err
	}
	for i := range values.Values {
		v := &values.Values[i]
		p, ok := statuses["VPIZR-"+strings.ReplaceAll(string(v.Month), "-", "")]
		if !ok {
			return FetchedOGD{}, fmt.Errorf("missing monthly OGD status: %s", v.Month)
		}
		v.Preliminary = p
	}
	for i := range values.Annual {
		v := &values.Annual[i]
		p, ok := statuses[fmt.Sprintf("VPIZR-%d", v.Year)]
		if !ok {
			return FetchedOGD{}, fmt.Errorf("missing annual OGD status: %d", v.Year)
		}
		v.Preliminary = p
	}
	dataSHA := fmt.Sprintf("%x", sha256.Sum256(data))
	statusSHA := fmt.Sprintf("%x", sha256.Sum256(labels))
	return FetchedOGD{URL: source.URL, SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(dataSHA+":"+statusSHA))), DataSHA256: dataSHA, StatusSHA256: statusSHA, FetchedAt: now.UTC(), Imported: values}, nil
}

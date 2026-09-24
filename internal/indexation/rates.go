package indexation

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

//go:embed data/statutory-rates.csv
var statutoryRatesCSV string

type RateKind string

const (
	RichtwertRate RateKind = "richtwert"
	CategoryRate  RateKind = "category"
)

type FederalState string

const (
	Burgenland   FederalState = "AT-1"
	Carinthia    FederalState = "AT-2"
	LowerAustria FederalState = "AT-3"
	UpperAustria FederalState = "AT-4"
	Salzburg     FederalState = "AT-5"
	Styria       FederalState = "AT-6"
	Tyrol        FederalState = "AT-7"
	Vorarlberg   FederalState = "AT-8"
	Vienna       FederalState = "AT-9"
)

type Category string

const (
	CategoryA         Category = "A"
	CategoryB         Category = "B"
	CategoryC         Category = "C"
	CategoryDUsable   Category = "D_usable"
	CategoryDUnusable Category = "D_unusable"
)

type StatutoryRate struct {
	Kind          RateKind
	Key           string
	CentsPerM2    int64
	EffectiveFrom time.Time
	ValidUntil    time.Time // Exclusive; next publication requires a new snapshot.
	Source        string
}

// StatutoryRates contains effective intervals, so the unchanged 2024 and 2025
// amounts do not masquerade as new valorisations. Category D's two legal amounts
// are explicit. These are published rates, not reconstructed percentages.
func StatutoryRates() ([]StatutoryRate, error) {
	r := csv.NewReader(strings.NewReader(statutoryRatesCSV))
	header, err := r.Read()
	if err != nil || strings.Join(header, ",") != "kind,key,cents_per_m2,effective_from,valid_until,source" {
		return nil, fmt.Errorf("invalid statutory rate header")
	}
	var rates []StatutoryRate
	for {
		row, err := r.Read()
		if err == io.EOF {
			return rates, nil
		}
		if err != nil {
			return nil, err
		}
		cents, err := strconv.ParseInt(row[2], 10, 64)
		if err != nil || cents <= 0 {
			return nil, fmt.Errorf("invalid statutory amount")
		}
		from, e1 := time.Parse(time.DateOnly, row[3])
		until, e2 := time.Parse(time.DateOnly, row[4])
		if e1 != nil || e2 != nil || !until.After(from) || row[5] == "" {
			return nil, fmt.Errorf("invalid statutory interval or source")
		}
		rates = append(rates, StatutoryRate{RateKind(row[0]), row[1], cents, from, until, row[5]})
	}
}

func Richtwert(state FederalState, asOf time.Time) (StatutoryRate, error) {
	return lookupRate(RichtwertRate, string(state), asOf)
}

func CategoryAmount(category Category, asOf time.Time) (StatutoryRate, error) {
	return lookupRate(CategoryRate, string(category), asOf)
}

func lookupRate(kind RateKind, key string, asOf time.Time) (StatutoryRate, error) {
	rates, err := StatutoryRates()
	if err != nil {
		return StatutoryRate{}, err
	}
	date := civilDate(asOf)
	for _, rate := range rates {
		if rate.Kind == kind && rate.Key == key && !date.Before(rate.EffectiveFrom) && date.Before(rate.ValidUntil) {
			return rate, nil
		}
	}
	return StatutoryRate{}, fmt.Errorf("no verified %s rate for %q on %s", kind, key, date.Format(time.DateOnly))
}

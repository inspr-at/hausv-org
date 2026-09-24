package indexation

import (
	"fmt"
	"sort"
	"time"
)

type Series string

const (
	VPI2025 Series = "VPI2025"
	VPI2020 Series = "VPI2020"
	VPI2015 Series = "VPI2015"
	VPI2010 Series = "VPI2010"
	VPI2005 Series = "VPI2005"
	VPI2000 Series = "VPI2000"
	VPI1996 Series = "VPI1996"
	VPI1986 Series = "VPI1986"
	VPI1976 Series = "VPI1976"
	VPI1966 Series = "VPI1966"
)

func (s Series) Valid() bool {
	switch s {
	case VPI2025, VPI2020, VPI2015, VPI2010, VPI2005, VPI2000, VPI1996, VPI1986, VPI1976, VPI1966:
		return true
	}
	return false
}

type IndexValue struct {
	Series      Series
	Month       Month
	Value       Decimal
	Preliminary bool
	Source      string
	FetchedAt   time.Time
	ChainSource string // Nonempty only when derived through an official factor.
}

type AnnualValue struct {
	Series      Series
	Year        int
	Value       Decimal
	Preliminary bool
	Source      string
	FetchedAt   time.Time
}

type ChainFactor struct {
	From          Series  `json:"from"`
	To            Series  `json:"to"`
	EffectiveFrom Month   `json:"effective_from"`
	Factor        Decimal `json:"factor_millionths"`
	Source        string  `json:"source"`
}

// Dataset owns immutable copies of observations and factors. Published values
// take precedence over calculated chains, including their preliminary flag.
type Dataset struct {
	values  map[Series]map[Month]IndexValue
	factors []ChainFactor
}

func NewDataset(values []IndexValue, factors []ChainFactor) (Dataset, error) {
	d := Dataset{values: make(map[Series]map[Month]IndexValue), factors: append([]ChainFactor(nil), factors...)}
	for _, v := range values {
		if !v.Series.Valid() || !v.Month.Valid() || v.Value <= 0 {
			return Dataset{}, fmt.Errorf("invalid observation: %s %s", v.Series, v.Month)
		}
		if d.values[v.Series] == nil {
			d.values[v.Series] = make(map[Month]IndexValue)
		}
		if _, exists := d.values[v.Series][v.Month]; exists {
			return Dataset{}, fmt.Errorf("duplicate observation: %s %s", v.Series, v.Month)
		}
		d.values[v.Series][v.Month] = v
	}
	seen := make(map[Series]bool)
	for _, f := range factors {
		// Only the published direct 2025 -> predecessor factors are supported.
		// Ratios of rounded historical indices are not chaining factors.
		if f.From != VPI2025 || !f.To.Valid() || f.To == f.From || f.Factor <= 0 || !f.EffectiveFrom.Valid() || f.Source == "" || seen[f.To] {
			return Dataset{}, fmt.Errorf("invalid or duplicate chain to %s", f.To)
		}
		seen[f.To] = true
	}
	return d, nil
}

func (d Dataset) Values() []IndexValue {
	var result []IndexValue
	for _, series := range d.values {
		for _, v := range series {
			result = append(result, v)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Series != result[j].Series {
			return result[i].Series < result[j].Series
		}
		return result[i].Month < result[j].Month
	})
	return result
}

func (d Dataset) Lookup(series Series, month Month) (IndexValue, bool, error) {
	if !series.Valid() || !month.Valid() {
		return IndexValue{}, false, fmt.Errorf("invalid series or month")
	}
	if v, ok := d.values[series][month]; ok {
		return v, true, nil
	}
	for _, f := range d.factors {
		if f.To != series || month < f.EffectiveFrom {
			continue
		}
		v, ok := d.values[f.From][month]
		if !ok {
			continue
		}
		// Round the product directly to one decimal, not in two stages.
		n, err := roundedRatio(int64(v.Value), int64(f.Factor), int64(Unit)*int64(Unit/10), true)
		if err != nil || n > int64(^uint64(0)>>1)/int64(Unit/10) {
			return IndexValue{}, false, fmt.Errorf("chaining overflow")
		}
		v.Value, v.Series, v.ChainSource = Decimal(n)*(Unit/10), series, f.Source
		return v, true, nil
	}
	return IndexValue{}, false, nil
}

func (d Dataset) latest(series Series) Month {
	var last Month
	for m := range d.values[series] {
		if m > last {
			last = m
		}
	}
	for _, f := range d.factors {
		if f.To == series {
			for m := range d.values[f.From] {
				if m >= f.EffectiveFrom && m > last {
					last = m
				}
			}
		}
	}
	return last
}

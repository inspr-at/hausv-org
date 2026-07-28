package energy

import (
	"math"
	"sort"
	"time"
)

const (
	QualityMeasured    = "measured"
	QualityEstimated   = "estimated"
	QualityGap         = "gap"
	QualityConflict    = "conflict"
	QualityStale       = "stale"
	QualityUnavailable = "unavailable"
)

type PowerSample struct {
	At time.Time
	KW float64
}

type EnergySample struct {
	At  time.Time
	KWh float64
}

type QualityAssessment struct {
	Status     string
	Label      string
	Effect     string
	NextAction string
}

type SourceComparison struct {
	ReferenceSource string
	ComparedSource  string
	ReferencePeakKW float64
	ComparedPeakKW  float64
	DeltaKW         float64
	DeltaPercent    float64
	Status          string
}

// CompareMonthlyPeaks compares completed quarter-hour peaks without merging
// sources. Smart-meter data can therefore remain a local reference while Home
// Assistant remains the live operational source.
func CompareMonthlyPeaks(intervals []Interval, at time.Time, location *time.Location, referenceSource, comparedSource string) (SourceComparison, bool) {
	reference := []Interval{}
	compared := []Interval{}
	for _, interval := range intervals {
		switch interval.Source {
		case referenceSource:
			reference = append(reference, interval)
		case comparedSource:
			compared = append(compared, interval)
		}
	}
	referencePeak := PeakForMonth(reference, at, location)
	comparedPeak := PeakForMonth(compared, at, location)
	if referencePeak <= 0 || comparedPeak <= 0 {
		return SourceComparison{}, false
	}
	delta := comparedPeak - referencePeak
	percent := math.Abs(delta) / referencePeak * 100
	status := "plausible"
	if percent > 10 {
		status = "check"
	}
	return SourceComparison{
		ReferenceSource: referenceSource,
		ComparedSource:  comparedSource,
		ReferencePeakKW: round2(referencePeak),
		ComparedPeakKW:  round2(comparedPeak),
		DeltaKW:         round2(delta),
		DeltaPercent:    round2(percent),
		Status:          status,
	}, true
}

// AggregatePowerIntervals integrates a piecewise-constant power series into
// exact settlement quarters. Iterating in absolute 15-minute steps preserves
// both repeated autumn quarters and the skipped spring hour.
func AggregatePowerIntervals(tenantSlug string, samples []PowerSample, from, to time.Time, location *time.Location, source string) []Interval {
	if location == nil {
		location = time.Local
	}
	if !to.After(from) {
		return nil
	}
	samples = append([]PowerSample(nil), samples...)
	sort.Slice(samples, func(i, j int) bool { return samples[i].At.Before(samples[j].At) })
	start := QuarterStart(from, location)
	if start.Before(from) {
		start = start.Add(15 * time.Minute)
	}
	out := []Interval{}
	for windowStart := start; windowStart.Add(15*time.Minute).Compare(to) <= 0; windowStart = windowStart.Add(15 * time.Minute) {
		windowEnd := windowStart.Add(15 * time.Minute)
		average, quality, ok := integratePowerWindow(samples, windowStart, windowEnd)
		if !ok {
			out = append(out, Interval{
				TenantSlug: tenantSlug, StartsAt: windowStart.UTC(), Duration: 15 * time.Minute,
				Quality: QualityGap, Source: source,
			})
			continue
		}
		out = append(out, Interval{
			TenantSlug: tenantSlug, StartsAt: windowStart.UTC(), Duration: 15 * time.Minute,
			ImportKWh: average / 4, AverageKW: average, Quality: quality, Source: source,
		})
	}
	return out
}

func integratePowerWindow(samples []PowerSample, start, end time.Time) (float64, string, bool) {
	index := sort.Search(len(samples), func(i int) bool { return samples[i].At.After(start) }) - 1
	if index < 0 {
		return 0, QualityGap, false
	}
	current := samples[index]
	if math.IsNaN(current.KW) || math.IsInf(current.KW, 0) || current.KW < 0 {
		return 0, QualityConflict, false
	}
	cursor := start
	energy := 0.0
	quality := QualityMeasured
	for i := index + 1; i < len(samples) && samples[i].At.Before(end); i++ {
		next := samples[i]
		if next.At.After(cursor) {
			energy += current.KW * next.At.Sub(cursor).Hours()
			cursor = next.At
		}
		if math.IsNaN(next.KW) || math.IsInf(next.KW, 0) || next.KW < 0 {
			quality = QualityConflict
			continue
		}
		current = next
	}
	if cursor.Before(end) {
		energy += current.KW * end.Sub(cursor).Hours()
	}
	return energy / end.Sub(start).Hours(), quality, true
}

// AggregateEnergyIntervals derives quarters from a cumulative import meter.
// Linear boundary interpolation is deterministic and makes late-arriving
// samples idempotent; counter resets are surfaced as conflicts.
func AggregateEnergyIntervals(tenantSlug string, samples []EnergySample, from, to time.Time, location *time.Location, source string) []Interval {
	if location == nil {
		location = time.Local
	}
	samples = append([]EnergySample(nil), samples...)
	sort.Slice(samples, func(i, j int) bool { return samples[i].At.Before(samples[j].At) })
	start := QuarterStart(from, location)
	if start.Before(from) {
		start = start.Add(15 * time.Minute)
	}
	out := []Interval{}
	for windowStart := start; windowStart.Add(15*time.Minute).Compare(to) <= 0; windowStart = windowStart.Add(15 * time.Minute) {
		windowEnd := windowStart.Add(15 * time.Minute)
		left, leftOK := cumulativeAt(samples, windowStart)
		right, rightOK := cumulativeAt(samples, windowEnd)
		interval := Interval{
			TenantSlug: tenantSlug, StartsAt: windowStart.UTC(), Duration: 15 * time.Minute,
			Source: source, Quality: QualityMeasured,
		}
		if !leftOK || !rightOK {
			interval.Quality = QualityGap
		} else if right < left {
			interval.Quality = QualityConflict
		} else {
			interval.ImportKWh = right - left
			interval.AverageKW = AveragePowerKW(interval.ImportKWh, interval.Duration)
		}
		out = append(out, interval)
	}
	return out
}

func cumulativeAt(samples []EnergySample, at time.Time) (float64, bool) {
	right := sort.Search(len(samples), func(i int) bool { return !samples[i].At.Before(at) })
	if right < len(samples) && samples[right].At.Equal(at) {
		return samples[right].KWh, finiteNonNegative(samples[right].KWh)
	}
	if right == 0 || right >= len(samples) {
		return 0, false
	}
	left := samples[right-1]
	next := samples[right]
	if !finiteNonNegative(left.KWh) || !finiteNonNegative(next.KWh) || !next.At.After(left.At) {
		return 0, false
	}
	ratio := at.Sub(left.At).Seconds() / next.At.Sub(left.At).Seconds()
	return left.KWh + (next.KWh-left.KWh)*ratio, true
}

func ProjectQuarterAverage(start, now time.Time, energySoFarKWh, currentPowerKW float64) (float64, bool) {
	end := start.Add(15 * time.Minute)
	if now.Before(start) || !now.Before(end) || energySoFarKWh < 0 || currentPowerKW < 0 {
		return 0, false
	}
	projectedEnergy := energySoFarKWh + currentPowerKW*end.Sub(now).Hours()
	return AveragePowerKW(projectedEnergy, 15*time.Minute), true
}

func AssessQuality(now, lastSeen time.Time, gaps, conflicts int) QualityAssessment {
	switch {
	case conflicts > 0:
		return QualityAssessment{Status: QualityConflict, Label: "Werte widersprechen sich", Effect: "Empfehlungen sind vorerst unklar.", NextAction: "Messpunkt und Einheit prüfen."}
	case gaps > 0:
		return QualityAssessment{Status: QualityGap, Label: "Messlücke erkannt", Effect: "Die Monatsspitze kann unvollständig sein.", NextAction: "Verbindung und Zählerverlauf prüfen."}
	case lastSeen.IsZero():
		return QualityAssessment{Status: QualityUnavailable, Label: "Noch kein Messwert", Effect: "Es wird keine Spitze geschätzt.", NextAction: "Einen Netzbezugswert verbinden."}
	case now.Sub(lastSeen) > 10*time.Minute:
		return QualityAssessment{Status: QualityStale, Label: "Messwert nicht aktuell", Effect: "HAUSV bleibt im Beobachtungsmodus.", NextAction: "Home Assistant prüfen."}
	default:
		return QualityAssessment{Status: QualityMeasured, Label: "Messwerte aktuell", Effect: "Auswertung ist nachvollziehbar.", NextAction: "Keine Aktion nötig."}
	}
}

func finiteNonNegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}

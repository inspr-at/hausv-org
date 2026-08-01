package energy

import (
	"sort"
	"time"
)

const (
	// QuarterDuration ist die Abrechnungseinheit des Leistungstarifs: bemessen
	// wird die höchste abgeschlossene Viertelstunde eines Kalendermonats.
	QuarterDuration = 15 * time.Minute

	// Quellen. Sie stehen bewusst nebeneinander und werden nie vermischt: der
	// Smart-Meter-Export bleibt die lokale Referenz, Home Assistant die laufende
	// Betriebsquelle. CompareMonthlyPeaks stellt beide gegenüber.
	SourceSmartMeter    = "smart-meter"
	SourceHomeAssistant = "home-assistant"

	defaultSampleInterval = 30 * time.Second
	defaultMinCoverage    = 0.9
)

// SamplingPolicy beschreibt, unter welchen Bedingungen aus Momentanleistungen
// eine Viertelstunde werden darf.
//
// Der entscheidende Wert ist MaxHold. Eine Momentanleistung sagt nur etwas über
// den Augenblick ihrer Messung aus; sie über eine beliebig lange Lücke
// fortzuschreiben wäre eine Erfindung. MaxHold begrenzt, wie lange ein Messwert
// als gültig gilt — alles darüber zählt als unbelegte Zeit und senkt die
// Abdeckung, statt stillschweigend über das Loch zu mitteln.
type SamplingPolicy struct {
	// Interval ist der angestrebte Abtastabstand.
	Interval time.Duration
	// MaxHold ist die längste Zeit, die ein einzelner Messwert fortgeschrieben
	// werden darf.
	MaxHold time.Duration
	// MinCoverage ist der Anteil der Viertelstunde, der belegt sein muss, bevor
	// überhaupt ein Wert festgeschrieben wird.
	MinCoverage float64
}

// DefaultSamplingPolicy leitet die Regel aus dem Abtastabstand ab: ein Messwert
// gilt höchstens zwei Abtastschritte weiter (eine ausgefallene Abtastung wird
// also verziehen, zwei nicht mehr), und mindestens 90 % der Viertelstunde
// müssen belegt sein.
func DefaultSamplingPolicy(interval time.Duration) SamplingPolicy {
	if interval <= 0 {
		interval = defaultSampleInterval
	}
	return SamplingPolicy{Interval: interval, MaxHold: 2 * interval, MinCoverage: defaultMinCoverage}
}

func (p SamplingPolicy) normalized() SamplingPolicy {
	if p.Interval <= 0 {
		p.Interval = defaultSampleInterval
	}
	if p.MaxHold <= 0 {
		p.MaxHold = 2 * p.Interval
	}
	if p.MinCoverage <= 0 || p.MinCoverage > 1 {
		p.MinCoverage = defaultMinCoverage
	}
	return p
}

// QuarterInput beschreibt genau eine Viertelstunde und was für sie beobachtet
// wurde. StaleReadings zählt Abrufe, die zwar geantwortet haben, deren Wert
// aber eingefroren war — das ist etwas anderes als gar keine Antwort und wird
// auch anders benannt.
type QuarterInput struct {
	TenantSlug    string
	Source        string
	Start         time.Time
	Now           time.Time
	Location      *time.Location
	Samples       []PowerSample
	StaleReadings int
	Policy        SamplingPolicy
}

// QuarterFromPowerSamples verdichtet Momentanleistungen zu genau einer
// Viertelstunde. Das zweite Ergebnis ist false, solange die Viertelstunde noch
// läuft: eine angefangene Viertelstunde darf nie geschrieben werden, weil das
// Cockpit ausdrücklich von der „abgeschlossenen Viertelstunde" spricht.
//
// Die Qualität ist höchstens QualityEstimated. Eine aus Momentanwerten
// verdichtete Viertelstunde ist keine Zählerablesung: zwischen zwei Abtastungen
// bleibt die Leistung angenommen, nicht gemessen. QualityMeasured wäre eine
// stärkere Aussage, als die Daten hergeben.
func QuarterFromPowerSamples(in QuarterInput) (Interval, bool) {
	policy := in.Policy.normalized()
	location := in.Location
	if location == nil {
		location = time.Local
	}
	now := in.Now
	if now.IsZero() {
		now = time.Now()
	}
	start := QuarterStart(in.Start, location)
	end := start.Add(QuarterDuration)
	if end.After(now) {
		return Interval{}, false
	}
	source := in.Source
	if source == "" {
		source = SourceHomeAssistant
	}

	inWindow := 0
	for _, sample := range in.Samples {
		if !sample.At.Before(start) && sample.At.Before(end) {
			inWindow++
		}
	}
	degraded := Interval{
		TenantSlug: in.TenantSlug,
		StartsAt:   start.UTC(),
		Duration:   QuarterDuration,
		Quality:    QualityGap,
		Source:     source,
	}
	// Eingefrorene Werte sind eine andere Diagnose als eine tote Verbindung:
	// Home Assistant hat geantwortet, die Messstelle aber nicht mehr geliefert.
	if inWindow == 0 && in.StaleReadings > 0 {
		degraded.Quality = QualityStale
	}
	if PowerSampleCoverage(in.Samples, start, end, policy.MaxHold) < policy.MinCoverage-1e-9 {
		return degraded, true
	}
	quarters := AggregatePowerIntervals(in.TenantSlug, in.Samples, start, end, location, source)
	if len(quarters) != 1 {
		return degraded, true
	}
	quarter := quarters[0]
	switch quarter.Quality {
	case QualityMeasured:
		quarter.Quality = QualityEstimated
	case QualityGap, QualityConflict:
		// Ohne belastbare Datenlage wird kein Wert veröffentlicht. Eine Zahl aus
		// widersprüchlichen Abtastungen wäre schlimmer als gar keine, weil sie in
		// die Monatsspitze und damit in die Tarifschätzung einginge.
		quarter.ImportKWh, quarter.AverageKW = 0, 0
	}
	return quarter, true
}

// PowerSampleCoverage ist der Anteil von [start,end), für den tatsächlich eine
// Messung vorliegt. Jeder Messwert belegt die Zeit bis zum nächsten Messwert,
// höchstens aber maxHold lang. Ein Messwert unmittelbar vor start belegt den
// Anfang des Fensters mit — das ist derselbe Vorhalt, mit dem
// AggregatePowerIntervals die Leistung integriert.
func PowerSampleCoverage(samples []PowerSample, start, end time.Time, maxHold time.Duration) float64 {
	total := end.Sub(start)
	if total <= 0 || maxHold <= 0 || len(samples) == 0 {
		return 0
	}
	ordered := append([]PowerSample(nil), samples...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].At.Before(ordered[j].At) })
	covered := time.Duration(0)
	for index, sample := range ordered {
		from := sample.At
		if from.Before(start) {
			from = start
		}
		if !from.Before(end) {
			break
		}
		to := sample.At.Add(maxHold)
		if index+1 < len(ordered) && ordered[index+1].At.Before(to) {
			to = ordered[index+1].At
		}
		if to.After(end) {
			to = end
		}
		if to.After(from) {
			covered += to.Sub(from)
		}
	}
	if covered > total {
		covered = total
	}
	return float64(covered) / float64(total)
}

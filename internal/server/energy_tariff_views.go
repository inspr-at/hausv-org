package server

import (
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/homeassistant"
)

type energyPeakView struct {
	Source string
	Value  string
}

type energyComparisonView struct {
	Tone    string
	Title   string
	Details string
}

type energyTariffView struct {
	ID          string
	Version     string
	Status      string
	SourceTitle string
	SourceURL   string
	Rule        string
	HasEstimate bool
	Disclaimer  string
	// Verrechnete Leistung und ihre Aufteilung an der Staffel. Getrennt
	// ausgewiesen, damit sichtbar wird, wo Kappen doppelt so viel bringt.
	PeakKW   string
	BilledKW string
	// PeakMeterPercent visualisiert die gemessene Spitze relativ zur
	// verrechneten Leistung. Der Wert ist serverseitig auf 0–100 begrenzt,
	// damit die ruhige Vergleichsgrafik nie eine andere Aussage als die Zahlen
	// daneben trifft.
	PeakMeterPercent int
	BelowKW          string
	AboveKW          string
	HasTier          bool
	MinimumReason    string
	AgreedKW         string
	HasAgreed        bool
	AgreedHint       string
	TierHint         string
	// MonthLabel und Basis benennen, woraus die Spitze stammt. Eine Kennzahl
	// ohne ihre Grundlage ist auf diesem Bildschirm wertlos: der Leistungstarif
	// bemisst je Kalendermonat, und ein halber Monat sieht aus wie ein ganzer.
	MonthLabel string
	// CoverageLabel is the compact visible statement. Basis keeps the full
	// month, source and quality explanation for progressive disclosure.
	CoverageLabel string
	Basis         string
	// PeakTime nennt den Zeitpunkt der teuersten Viertelstunde. Ohne ihn bleibt
	// die Spitze eine Zahl, mit ihm wird sie ein Ereignis, das man wiedererkennt.
	PeakTime    string
	HasPeakTime bool
	// AnnualPowerEUR ist ausschließlich der Leistungsanteil des Netztarifs.
	// Getrennt von einem Label geführt, weil genau diese Zahl unter einer
	// Überschrift über den Tarif 2027 als ganze Jahresrechnung missverstanden
	// wird — und ein Haushalt, dem eine Ersparnis versprochen wird, die nie
	// eintritt, ist dauerhaft verloren.
	AnnualPowerEUR string
	// Die Sätze der Modellrechnung kommen aus dem Regelprofil und nicht aus der
	// Vorlage: sonst behauptet die Oberfläche weiter 33,82 €, wenn im Profil
	// längst ein anderer Satz steht.
	BelowRateEUR string
	AboveRateEUR string
	ThresholdKW  string
	// MissingIsWaiting unterscheidet Warten von Handeln: liegt der Netzbezug
	// zugeordnet vor, kommt die Zahl von selbst.
	MissingIsWaiting bool
	// MissingReason erklärt im Leerzustand, warum keine Spitze dasteht. Eine
	// leere Kachel oder eine 0 wäre beides falsch.
	MissingReason string
}

type energyScenarioView struct {
	Title       string
	PeakBand    string
	EffectBand  string
	Uncertainty string
	Assumptions string
	// BilledBand übersetzt das Spitzenband in die verrechnete Leistung. Nur sie
	// steht auf der Rechnung: unterhalb der Mindestbemessung senkt eine
	// niedrigere Spitze nichts mehr, und genau dort entstünde sonst ein
	// Einsparversprechen, das nie eintritt.
	BilledBand    string
	HasBilledBand bool
	// BaselineNote nennt den Ausgangswert. Ein Band ohne seinen Ausgangspunkt
	// lädt dazu ein, die Differenz zu irgendeiner Zahl im Kopf zu bilden.
	BaselineNote string
	// FloorNote warnt, wenn die optimistische Seite des Bandes unter die
	// verrechenbare Untergrenze fällt. Die Spitze sinkt dort real weiter, der
	// verrechnete Betrag aber nicht — ohne den Hinweis liest sich das Szenario
	// als Ersparnis, die so nicht eintritt.
	FloorNote string
}

type energyTariffAssessmentView struct {
	ID            string
	Month         string
	Profile       string
	Peak          string
	Annual        string
	Quality       string
	Created       string
	SourceURL     string
	ProfileStatus string
}

func energyComparisonForView(intervals []energy.Interval, at time.Time) (energyComparisonView, bool) {
	comparison, ok := energy.CompareMonthlyPeaks(intervals, at, time.Local, "smart-meter", "home-assistant")
	if !ok {
		return energyComparisonView{}, false
	}
	if comparison.Status == "check" {
		return energyComparisonView{
			Tone:  "warning",
			Title: "Messquellen weichen sichtbar ab",
			Details: "Home Assistant liegt bei der Monatsspitze um " +
				formatEnergyValueUnit(formatEnergyNumber(comparison.DeltaPercent), "%") + " neben der Smart-Meter-Referenz. Zähler, Einheit und Vorzeichen prüfen.",
		}, true
	}
	return energyComparisonView{
		Tone:    "good",
		Title:   "Messquellen sind plausibel",
		Details: "Die Monatsspitzen liegen innerhalb von 10 % beieinander.",
	}, true
}

func buildEnergyTariffAssessmentViews(items []energy.TariffAssessment) []energyTariffAssessmentView {
	out := make([]energyTariffAssessmentView, 0, len(items))
	for _, item := range items {
		month := item.AssessmentMonth
		if parsed, err := time.Parse("2006-01", item.AssessmentMonth); err == nil {
			month = parsed.Format("01/2006")
		}
		out = append(out, energyTariffAssessmentView{
			ID: item.ID, Month: month, Profile: item.ProfileID + " · " + item.ProfileVersion,
			Peak: formatEnergyValueUnit(formatEnergyNumber(item.PeakKW), "kW"), Annual: formatEnergyValueUnit(formatEnergyNumber(item.AnnualPowerEUR), "€") + " Modellwert/Jahr",
			Quality: energyQualityLabel(item.DataQuality), Created: item.CreatedAt.In(time.Local).Format("02.01.2006 15:04"),
			SourceURL: item.SourceURL, ProfileStatus: item.ProfileStatus,
		})
	}
	return out
}

// updateEnergyAgreedPower speichert die mit dem Netzbetreiber vereinbarte
// Anschlussleistung. Sie ist keine Zielgröße, sondern eine Vertragstatsache:
// ab 2027 bemisst der Entwurf mindestens 20 % davon, auch in einem Monat ohne
// jede Spitze.
func (a *app) updateEnergyAgreedPower(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	profile, exists, err := a.energyFor(ac).Profile(ac.tenant.Slug)
	if err != nil || !exists {
		http.Error(w, "Hausprofil fehlt.", http.StatusBadRequest)
		return
	}
	before := ""
	if profile.AgreedPowerKW != nil {
		before = formatEnergyNumber(*profile.AgreedPowerKW)
	}
	raw := strings.TrimSpace(r.FormValue("agreed_power_kw"))
	if raw == "" {
		// Leeren heißt "nicht erfasst" — die Mindestbemessung ruht dann wieder,
		// statt gegen einen alten Wert weiterzurechnen.
		profile.AgreedPowerKW = nil
	} else {
		value, parseErr := homeassistant.ParseFloat(raw)
		if parseErr != nil || !energy.ValidPowerKW(value, 1000) {
			http.Error(w, "Vereinbarte Anschlussleistung muss eine positive kW-Zahl sein.", http.StatusBadRequest)
			return
		}
		profile.AgreedPowerKW = &value
	}
	if err := a.energyFor(ac).SaveProfile(profile); err != nil {
		http.Error(w, "Anschlussleistung konnte nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	after := ""
	if profile.AgreedPowerKW != nil {
		after = formatEnergyNumber(*profile.AgreedPowerKW)
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     "energy.agreed_power.change",
		TargetType: "home-profile",
		TargetID:   ac.tenant.Slug,
		Summary:    "Vereinbarte Anschlussleistung geändert",
		Details: map[string]string{
			"agreed_from_kw": before,
			"agreed_to_kw":   after,
		},
	})
	http.Redirect(w, r, "/app/energie?agreed=1#tarif", http.StatusSeeOther)
}

func (a *app) saveEnergyTariffAssessment(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	intervals, err := a.energyFor(ac).ListIntervals(ac.tenant.Slug, monthStart.UTC(), monthStart.AddDate(0, 1, 0).UTC())
	if err != nil {
		http.Error(w, "Messwerte konnten nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	peak := energy.PeakForMonth(intervals, now, time.Local)
	if peak <= 0 {
		http.Redirect(w, r, "/app/energie?assessment=no_data#tarif", http.StatusSeeOther)
		return
	}
	rules := energy.AustrianDraft2027()
	// Die vereinbarte Anschlussleistung entscheidet über die Mindestbemessung.
	// Fehlt das Profil, bleibt sie 0 und nur der 2-kW-Sockel greift.
	assessedProfile, _, _ := a.energyFor(ac).Profile(ac.tenant.Slug)
	estimate := rules.Estimate(peak, agreedPowerKW(assessedProfile))
	gaps, conflicts := intervalQualityCounts(intervals)
	quality := energy.QualityMeasured
	if gaps > 0 || conflicts > 0 {
		quality = energy.QualityGap
	}
	item := energy.TariffAssessment{
		ID:         energy.NewID("tariff"),
		TenantSlug: ac.tenant.Slug, AssessmentMonth: monthStart.Format("2006-01"),
		ProfileID: rules.ID, ProfileVersion: rules.Version, ProfileStatus: rules.Status, SourceURL: rules.SourceURL,
		PeakKW: peak, BilledKW: estimate.BilledKW, AnnualPowerEUR: estimate.AnnualPowerEUR, DataQuality: quality,
	}
	if err := a.energyFor(ac).SaveTariffAssessment(item); err != nil {
		http.Error(w, "Tarifbewertung konnte nicht festgehalten werden.", http.StatusInternalServerError)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role,
		Action: "energy.tariff.assessment", TargetType: "energy-tariff-assessment", TargetID: item.ID,
		Summary: "Tarifbewertung unveränderlich festgehalten", Details: map[string]string{"profile_id": rules.ID, "profile_version": rules.Version},
	})
	http.Redirect(w, r, "/app/energie?assessment=saved#tarif", http.StatusSeeOther)
}

func energyPeakViews(intervals []energy.Interval, at time.Time) []energyPeakView {
	bySource := map[string][]energy.Interval{}
	for _, interval := range intervals {
		if interval.Quality == energy.QualityMeasured || interval.Quality == energy.QualityEstimated {
			bySource[interval.Source] = append(bySource[interval.Source], interval)
		}
	}
	sources := make([]string, 0, len(bySource))
	for source := range bySource {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	out := make([]energyPeakView, 0, len(sources))
	for _, source := range sources {
		label := source
		switch source {
		case "smart-meter":
			label = "Smart Meter"
		case "home-assistant":
			label = "Home Assistant"
		}
		out = append(out, energyPeakView{
			Source: label,
			Value:  formatEnergyValueUnit(formatEnergyNumber(energy.PeakForMonth(bySource[source], at, time.Local)), "kW"),
		})
	}
	return out
}

// recordsItself sagt, ob ein bestaetigter Netzbezug vorliegt. Nur dann fuellt
// sich die Karte von selbst — der Leerzustand ist dann eine Wartezeit und
// keine Aufforderung, eine Datei zu suchen.
func buildEnergyTariffView(profile energy.HomeProfile, intervals []energy.Interval, recordsItself bool) energyTariffView {
	now := time.Now()
	rules := energy.AustrianDraft2027()
	view := energyTariffView{
		ID:          rules.ID,
		Version:     rules.Version,
		Status:      "Entwurf · nicht verbindlich",
		SourceTitle: rules.SourceTitle,
		SourceURL:   rules.SourceURL,
		Rule:        "Höchste abgeschlossene Viertelstunde des Monats; Entwurfsannahme mindestens 2 kW und 20 % der vereinbarten Leistung.",
		Disclaimer:  "Keine Tarif- oder Einspargarantie. Neue Verordnungsversionen können ausgetauscht werden, ohne Messdaten zu verändern.",
		MonthLabel:  energyMonthLabel(now),

		BelowRateEUR: formatEnergyValueUnit(formatEnergyCompact(rules.AnnualBelowEURPerKW, 2), "€"),
		AboveRateEUR: formatEnergyValueUnit(formatEnergyCompact(rules.AnnualAboveEURPerKW, 2), "€"),
		ThresholdKW:  formatEnergyValueUnit(formatEnergyCompact(rules.TierThresholdKW, 0), "kW"),
	}
	peak := energy.PeakForMonth(intervals, now, time.Local)
	if peak <= 0 {
		if recordsItself {
			view.MissingReason = "Die erste vollständige Viertelstunde wird gerade aufgezeichnet und erscheint hier, sobald sie zu Ende ist."
			view.MissingIsWaiting = true
		} else {
			view.MissingReason = "Sobald der Netzbezug einem Messwert zugeordnet ist, zeichnet HAUSV die Viertelstunden selbst auf."
		}
		return view
	}
	estimate := rules.Estimate(peak, agreedPowerKW(profile))
	view.AnnualPowerEUR = formatEnergyValueUnit(formatEnergyNumber(estimate.AnnualPowerEUR), "€")
	view.HasEstimate = true
	coverage := buildEnergyTariffCoverageView(intervals, now)
	view.CoverageLabel = coverage.Label
	view.Basis = coverage.Basis
	if at, ok := peakQuarterOfMonth(intervals, now); ok {
		view.PeakTime = at.In(time.Local).Format("02.01. um 15:04")
		view.HasPeakTime = true
	}
	view.PeakKW = formatEnergyValueUnit(formatEnergyCompact(peak, 1), "kW")
	view.BilledKW = formatEnergyValueUnit(formatEnergyCompact(estimate.BilledKW, 1), "kW")
	if estimate.BilledKW > 0 {
		view.PeakMeterPercent = int(math.Round(math.Max(0, math.Min(1, peak/estimate.BilledKW)) * 100))
	}
	view.MinimumReason = estimate.MinimumReason
	if estimate.AboveKW > 0 {
		view.HasTier = true
		view.BelowKW = formatEnergyValueUnit(formatEnergyCompact(estimate.BelowKW, 1), "kW")
		view.AboveKW = formatEnergyValueUnit(formatEnergyCompact(estimate.AboveKW, 1), "kW")
		view.TierHint = "Der Anteil über " + formatEnergyValueUnit(formatEnergyCompact(rules.TierThresholdKW, 0), "kW") + " wird im Entwurf mit dem höheren Satz bemessen. Dort wirkt Kappen etwa doppelt so stark."
	}
	if agreed := agreedPowerKW(profile); agreed > 0 {
		view.HasAgreed = true
		view.AgreedKW = formatEnergyValueUnit(formatEnergyCompact(agreed, 1), "kW")
	} else {
		view.AgreedHint = "Ohne vereinbarte Anschlussleistung rechnet die Schätzung nur mit dem 2-kW-Sockel. Der Wert steht auf Ihrer Netzrechnung."
	}
	return view
}

// energyMonthLabel schreibt den Kalendermonat aus. Der Leistungstarif bemisst
// je Kalendermonat; „08/2026“ oder gar nichts wäre auf diesem Bildschirm die
// schlechtere Antwort.
func energyMonthLabel(at time.Time) string {
	names := []string{"Jänner", "Februar", "März", "April", "Mai", "Juni",
		"Juli", "August", "September", "Oktober", "November", "Dezember"}
	local := at.In(time.Local)
	return names[int(local.Month())-1] + " " + strconv.Itoa(local.Year())
}

type energyTariffCoverageView struct {
	Label     string
	Basis     string
	Measured  int
	Estimated int
}

// buildEnergyTariffCoverageView benennt, worauf die Monatsspitze beruht: wie
// viele abgeschlossene Viertelstunden aus welcher Quelle und wie viel des
// Monats damit abgedeckt ist. Label bleibt kurz genug fuer die Karte; Basis
// traegt Monat, Quelle und die gemessen/geschaetzt-Unterscheidung fuer die
// progressive Offenlegung.
func buildEnergyTariffCoverageView(intervals []energy.Interval, at time.Time) energyTariffCoverageView {
	local := at.In(time.Local)
	monthStart := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, time.Local)
	monthIntervals := make([]energy.Interval, 0, len(intervals))
	sources := map[string]struct{}{}
	for _, interval := range intervals {
		start := interval.StartsAt.In(time.Local)
		if start.Before(monthStart) || !start.Before(monthStart.AddDate(0, 1, 0)) {
			continue
		}
		monthIntervals = append(monthIntervals, interval)
		if interval.Quality == energy.QualityMeasured || interval.Quality == energy.QualityEstimated {
			sources[interval.Source] = struct{}{}
		}
	}
	counts := energy.CountUsableQuarters(monthIntervals)
	if counts.Total == 0 {
		return energyTariffCoverageView{}
	}
	elapsed := int(local.Sub(monthStart) / (15 * time.Minute))
	if elapsed < counts.Total {
		elapsed = counts.Total
	}
	label := strconv.Itoa(counts.Total) + " von " + strconv.Itoa(elapsed) + " Viertelstunden abgedeckt"
	basis := strconv.Itoa(counts.Total) + " von " + strconv.Itoa(elapsed) +
		" bisherigen Viertelstunden im " + energyMonthLabel(local)
	quality := make([]string, 0, 2)
	if counts.Measured > 0 {
		quality = append(quality, strconv.Itoa(counts.Measured)+" direkt gemessen")
	}
	if counts.Estimated > 0 {
		quality = append(quality, strconv.Itoa(counts.Estimated)+" aus Momentanwerten geschätzt")
	}
	if len(quality) > 0 {
		basis += " · " + strings.Join(quality, ", ")
	}
	if names := energySourceLabels(sources); names != "" {
		basis += " · Quelle: " + names
	}
	return energyTariffCoverageView{
		Label:     label,
		Basis:     basis,
		Measured:  counts.Measured,
		Estimated: counts.Estimated,
	}
}

func energyTariffBasis(intervals []energy.Interval, at time.Time) string {
	return buildEnergyTariffCoverageView(intervals, at).Basis
}

// energySourceLabels übersetzt die internen Quellenschlüssel in Klartext und
// hält die Reihenfolge stabil, damit zwei Aufrufe nicht unterschiedlich lauten.
func energySourceLabels(sources map[string]struct{}) string {
	keys := make([]string, 0, len(sources))
	for key := range sources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	labels := make([]string, 0, len(keys))
	for _, key := range keys {
		switch key {
		case "smart-meter":
			labels = append(labels, "Smart-Meter-Export")
		case "home-assistant":
			labels = append(labels, "Home Assistant")
		default:
			labels = append(labels, key)
		}
	}
	return strings.Join(labels, " und ")
}

// peakQuarterOfMonth liefert den Beginn der teuersten Viertelstunde des
// laufenden Monats. PeakForMonth gibt nur den Wert zurück; für die Oberfläche
// zählt aber, wann er entstanden ist — daran erkennt ein Haushalt die eigene
// Gewohnheit wieder.
func peakQuarterOfMonth(intervals []energy.Interval, at time.Time) (time.Time, bool) {
	local := at.In(time.Local)
	monthStart := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, time.Local)
	monthEnd := monthStart.AddDate(0, 1, 0)
	best := time.Time{}
	bestKW := 0.0
	found := false
	for _, interval := range intervals {
		if interval.Quality != energy.QualityMeasured && interval.Quality != energy.QualityEstimated {
			continue
		}
		start := interval.StartsAt.In(time.Local)
		if start.Before(monthStart) || !start.Before(monthEnd) {
			continue
		}
		if !found || interval.AverageKW > bestKW {
			best, bestKW, found = start, interval.AverageKW, true
		}
	}
	return best, found
}

// agreedPowerKW liefert 0, solange die vereinbarte Anschlussleistung nicht
// erfasst ist. Estimate lässt die 20-%-Mindestbemessung dann bewusst aus,
// statt sie gegen einen geratenen Wert zu rechnen.
func agreedPowerKW(profile energy.HomeProfile) float64 {
	if profile.AgreedPowerKW == nil {
		return 0
	}
	return *profile.AgreedPowerKW
}

// assetFlexContribution liefert Leistung und Flexibilität eines Verbrauchers.
//
// Reihenfolge ist Absicht: gesetzte Werte am Asset schlagen die Vorbelegung
// aus der Kategorie. Damit bleibt die Kategorie eine Vorlage und wird nicht
// zur Verhaltensregel — das ist der Kern von HAUSV-422.
func assetFlexContribution(asset energy.Asset) (float64, string) {
	// Erzeugung verschiebt die Bezugsspitze nicht. Die Nennleistung einer
	// PV-Anlage bleibt erfasst — sie beschreibt die Anlagengröße —, darf aber
	// nicht als abschaltbare Last verrechnet werden.
	if asset.Kind == "pv" {
		return 0, energy.FlexUnknown
	}
	kw := defaultAssetPowerKW(asset.Kind)
	if asset.RatedPowerKW != nil {
		kw = *asset.RatedPowerKW
	}
	flexibility := asset.Flexibility
	if flexibility == "" || flexibility == energy.FlexUnknown {
		if asset.Source == energyCustomAssetSource {
			// Frei angelegt und ohne Angabe: nichts versprechen. Der
			// Verbraucher zählt im Verbrauch, aber nicht in der Peak-Wirkung.
			return 0, energy.FlexUnknown
		}
		flexibility = defaultAssetFlexibility(asset.Kind)
	}
	return kw, flexibility
}

// defaultAssetPowerKW ist die Vorbelegung einer Vorlage ohne eigene Angabe.
// Bewusst zurückhaltend: es sind Modellannahmen, keine Messwerte. PV liefert
// 0, weil Erzeugung die Bezugsspitze nicht verschiebt.
func defaultAssetPowerKW(kind string) float64 {
	switch kind {
	case "ev", "wallbox", "battery":
		return 3
	case "hot-water", "heat-pump":
		return 1
	default:
		return 0
	}
}

func assetFlexAssumption(asset energy.Asset, suffix string) string {
	name := strings.TrimSpace(asset.Name)
	if name == "" {
		name = energy.AssetKindLabel(asset.Kind)
	}
	return name + " " + suffix
}

func isChargingPreset(asset energy.Asset) bool {
	return asset.Source != energyCustomAssetSource && (asset.Kind == "ev" || asset.Kind == "wallbox")
}

// chargingPresetIndex wählt aus den Vorlagen für die Ladelast die
// aussagekräftigste aus: eine erfasste Nennleistung schlägt die Vorbelegung,
// sonst die höhere Leistung. Ohne diese Wahl entschiede die Sortierung — die
// Liste kommt nach Art sortiert, also gewänne immer das E-Auto mit seinen
// vorbelegten 3 kW, selbst neben einer erfassten 22-kW-Wallbox.
func chargingPresetIndex(assets []energy.Asset) int {
	best := -1
	for index, asset := range assets {
		if !isChargingPreset(asset) {
			continue
		}
		if best < 0 {
			best = index
			continue
		}
		current, previous := assets[index], assets[best]
		if (current.RatedPowerKW != nil) != (previous.RatedPowerKW != nil) {
			if current.RatedPowerKW != nil {
				best = index
			}
			continue
		}
		currentKW, _ := assetFlexContribution(current)
		previousKW, _ := assetFlexContribution(previous)
		if currentKW > previousKW {
			best = index
		}
	}
	return best
}

func buildEnergyScenarioViews(profile energy.HomeProfile, assets []energy.Asset, intervals []energy.Interval) []energyScenarioView {
	baseline := energy.PeakForMonth(intervals, time.Now(), time.Local)
	if baseline <= 0 {
		return nil
	}
	shiftable, throttle, battery := 0.0, 0.0, 0.0
	assumptions := []string{}
	// Je Asset gerechnet, nicht je Kategorie. Die Kategorie liefert nur
	// Vorbelegungen; eine gesetzte Nennleistung und eine gesetzte Flexibilität
	// gewinnen immer. Vorher verschmolz `has[kind]` zwei gleichartige
	// Verbraucher zu einem — zwei Wallboxen ergaben dieselben 3 kW wie eine.
	chargingPreset := chargingPresetIndex(assets)
	for index, asset := range assets {
		kw, flexibility := assetFlexContribution(asset)
		if kw <= 0 {
			continue
		}
		// E-Auto und Wallbox beschreiben als Vorlagen dieselbe Ladelast; beide
		// anzurechnen würde die Flexibilität erfinden. Frei angelegte
		// Verbraucher zählen dagegen einzeln — sie wurden bewusst so benannt.
		if isChargingPreset(asset) && index != chargingPreset {
			continue
		}
		switch {
		case asset.Kind == "battery" && flexibility != energy.FlexFixed:
			// Ein Speicher ist keine drosselbare Last, sondern eine eigene
			// Rolle mit eigenem Gewicht in der Simulation. Wer ihn ausdrücklich
			// als fest erklärt, bekommt dieses Gewicht nicht.
			battery += kw
			assumptions = append(assumptions, assetFlexAssumption(asset, "Speicherleistung"))
		case flexibility == energy.FlexShift:
			shiftable += kw
			assumptions = append(assumptions, assetFlexAssumption(asset, "zeitlich verschiebbar"))
		case flexibility == energy.FlexThrottle:
			throttle += kw
			assumptions = append(assumptions, assetFlexAssumption(asset, "kurz begrenzbar"))
		}
	}
	if shiftable+throttle+battery == 0 {
		return nil
	}
	counts := energy.CountUsableQuarters(intervals)
	quality := energy.QualityUnavailable
	if counts.Measured > 0 {
		quality = energy.QualityMeasured
	}
	// A mixed basis stays conservative: one estimated quarter means the
	// scenario must not present the complete baseline as directly measured.
	if counts.Estimated > 0 {
		quality = energy.QualityEstimated
	}
	gaps, conflicts := intervalQualityCounts(intervals)
	if gaps > 0 || conflicts > 0 {
		quality = energy.QualityGap
	}
	result := energy.SimulateScenario(energy.ScenarioInput{
		Name: "Bestehende Flexibilität nutzen", BaselinePeakKW: baseline,
		ShiftableKW: shiftable, ThrottleKW: throttle, BatteryKW: battery, DataQuality: quality,
	})
	view := energyScenarioView{
		Title: result.Name,
		BaselineNote: "Ausgangswert: die gemessene Monatsspitze von " +
			formatEnergyValueUnit(formatEnergyCompact(baseline, 1), "kW") + " im " + energyMonthLabel(time.Now()) + ".",
		PeakBand:    formatEnergyValueUnit(formatEnergyNumber(result.ExpectedPeakLowKW)+"–"+formatEnergyNumber(result.ExpectedPeakHighKW), "kW"),
		EffectBand:  formatEnergyValueUnit(formatEnergyNumber(result.PeakEffectLowKW)+"–"+formatEnergyNumber(result.PeakEffectHighKW), "kW") + " mögliche Peak-Wirkung",
		Uncertainty: result.Uncertainty,
		Assumptions: strings.Join(assumptions, " · "),
	}
	// Estimate(0, agreed) liefert genau die Untergrenze: den 2-kW-Sockel oder
	// die 20 % der vereinbarten Leistung, je nachdem was höher ist.
	rules := energy.AustrianDraft2027()
	agreed := agreedPowerKW(profile)
	billedLow := rules.Estimate(result.ExpectedPeakLowKW, agreed).BilledKW
	billedHigh := rules.Estimate(result.ExpectedPeakHighKW, agreed).BilledKW
	// Nur zeigen, wenn die Mindestbemessung das Band tatsächlich anhebt. Sonst
	// stünde zweimal dieselbe Zahl da und die Aussage ginge im Rauschen unter.
	if billedLow > result.ExpectedPeakLowKW+0.05 || billedHigh > result.ExpectedPeakHighKW+0.05 {
		view.BilledBand = formatEnergyValueUnit(formatEnergyCompact(billedLow, 1)+"–"+formatEnergyCompact(billedHigh, 1), "kW")
		if billedLow == billedHigh {
			view.BilledBand = formatEnergyValueUnit(formatEnergyCompact(billedLow, 1), "kW")
		}
		view.HasBilledBand = true
	}
	if floor := rules.Estimate(0, agreed).BilledKW; result.ExpectedPeakLowKW < floor {
		view.FloorNote = "Unter " + formatEnergyValueUnit(formatEnergyCompact(floor, 1), "kW") + " sinkt der verrechnete Betrag nicht weiter: so weit reicht die Mindestbemessung. Weiteres Kappen senkt die Spitze, nicht die Rechnung."
	}
	return []energyScenarioView{view}
}

// defaultAssetFlexibility hält die Vorbelegung an einer Stelle: im
// energy-Paket, das sie auch beim Seeding speichert.
func defaultAssetFlexibility(kind string) string {
	return energy.DefaultAssetFlexibility(kind)
}

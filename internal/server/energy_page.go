package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/web"
)

// energyPortalContext keeps the cockpit on the shared portal shell. The title
// stays the household name, exactly as the legacy template rendered it, so the
// browser tab does not change meaning when the templ switch is flipped.
func (a *app) energyPortalContext(ac authCtx, title string) web.PortalPageData {
	data := a.auditPortalContext(ac, title)
	data.Title = title
	data.ActivePage = "energy"
	return data
}

func (a *app) renderEnergyTempl(w http.ResponseWriter, r *http.Request, data web.EnergyPageData) {
	var rendered bytes.Buffer
	if err := web.EnergyPage(data).Render(r.Context(), &rendered); err != nil {
		logError("templ energy render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), data.Portal.TenantSlug))
}

func energyWebMetric(metric energyMetricView) web.EnergyMetricView {
	return web.EnergyMetricView{Label: metric.Label, Value: metric.Value, Detail: metric.Detail}
}

func energyWebMetrics(metrics []energyMetricView) []web.EnergyMetricView {
	out := make([]web.EnergyMetricView, 0, len(metrics))
	for _, metric := range metrics {
		out = append(out, energyWebMetric(metric))
	}
	return out
}

func energyWebLive(live energyLiveView) web.EnergyLiveView {
	return web.EnergyLiveView{
		Main:             energyWebMetric(live.Main),
		HasMain:          live.HasMain,
		Flows:            energyWebMetrics(live.Flows),
		Battery:          energyWebMetric(live.Battery),
		HasBattery:       live.HasBattery,
		BatterySOC:       energyWebMetric(live.BatterySOC),
		HasBatterySOC:    live.HasBatterySOC,
		Additional:       energyWebMetrics(live.Additional),
		HasAdditional:    live.HasAdditional,
		AdditionalCount:  live.AdditionalCount,
		AdditionalTopics: live.AdditionalTopics,
	}
}

func energyWebChart(chart energyChartView) web.EnergyChartView {
	series := make([]web.EnergyChartSeriesView, 0, len(chart.Series))
	for _, item := range chart.Series {
		series = append(series, web.EnergyChartSeriesView{
			Key: item.Key, Label: item.Label,
			Path: item.Path, MobilePath: item.MobilePath,
			AreaPath: item.AreaPath, MobileAreaPath: item.MobileAreaPath,
			Latest: item.Latest,
		})
	}
	ticks := func(source []energyChartTickView) []web.EnergyChartTickView {
		out := make([]web.EnergyChartTickView, 0, len(source))
		for _, tick := range source {
			out = append(out, web.EnergyChartTickView{Position: tick.Position, MobilePosition: tick.MobilePosition, Label: tick.Label})
		}
		return out
	}
	samples := make([]web.EnergyChartSampleView, 0, len(chart.Samples))
	for _, sample := range chart.Samples {
		values := make([]web.EnergyChartSampleValueView, 0, len(sample.Values))
		for _, value := range sample.Values {
			values = append(values, web.EnergyChartSampleValueView{
				Key: value.Key, Label: value.Label, Value: value.Value,
				Position: value.Position, MobilePosition: value.MobilePosition,
			})
		}
		samples = append(samples, web.EnergyChartSampleView{
			Index: sample.Index, Time: sample.Time,
			Position: sample.Position, MobilePosition: sample.MobilePosition,
			HitPosition: sample.HitPosition, HitWidth: sample.HitWidth,
			MobileHit: sample.MobileHit, MobileHitWidth: sample.MobileHitWidth,
			Values: values,
		})
	}
	return web.EnergyChartView{
		HasData: chart.HasData, IsToday: chart.IsToday,
		Title: chart.Title, DialogTitle: chart.DialogTitle,
		Summary: chart.Summary, Detail: chart.Detail, Status: chart.Status, Range: chart.Range,
		Series: series, XTicks: ticks(chart.XTicks), YTicks: ticks(chart.YTicks), Samples: samples,
		HasThreshold:   chart.HasThreshold,
		ThresholdLabel: chart.ThresholdLabel, ThresholdValue: chart.ThresholdValue,
		ThresholdPosition: chart.ThresholdPosition, ThresholdMobilePosition: chart.ThresholdMobilePosition,
	}
}

func energyWebTariff(tariff energyTariffView) web.EnergyTariffView {
	return web.EnergyTariffView{
		ID: tariff.ID, Version: tariff.Version, Status: tariff.Status,
		SourceTitle: tariff.SourceTitle, SourceURL: tariff.SourceURL,
		HasEstimate: tariff.HasEstimate, Disclaimer: tariff.Disclaimer,
		PeakKW: tariff.PeakKW, BilledKW: tariff.BilledKW,
		PeakMeterPercent: tariff.PeakMeterPercent,
		BelowKW:          tariff.BelowKW, AboveKW: tariff.AboveKW, HasTier: tariff.HasTier,
		MinimumReason: tariff.MinimumReason,
		HasAgreed:     tariff.HasAgreed, AgreedHint: tariff.AgreedHint, TierHint: tariff.TierHint,
		MonthLabel: tariff.MonthLabel, CoverageLabel: tariff.CoverageLabel, Basis: tariff.Basis,
		PeakTime: tariff.PeakTime, HasPeakTime: tariff.HasPeakTime,
		AnnualPowerEUR: tariff.AnnualPowerEUR,
		BelowRateEUR:   tariff.BelowRateEUR, AboveRateEUR: tariff.AboveRateEUR, ThresholdKW: tariff.ThresholdKW,
		MissingIsWaiting: tariff.MissingIsWaiting, MissingReason: tariff.MissingReason,
	}
}

func energyWebOptions(options []energyOption) []web.EnergyOption {
	out := make([]web.EnergyOption, 0, len(options))
	for _, option := range options {
		out = append(out, web.EnergyOption{Value: option.Value, Label: option.Label})
	}
	return out
}

func energyWebCoverage(items []energyCoverageView) []web.EnergyCoverageView {
	out := make([]web.EnergyCoverageView, 0, len(items))
	for _, item := range items {
		out = append(out, web.EnergyCoverageView{Label: item.Label, Status: item.Status, Detail: item.Detail, Tone: item.Tone})
	}
	return out
}

func energyWebRoadmap(steps []energyRoadmapStep) []web.EnergyRoadmapStep {
	out := make([]web.EnergyRoadmapStep, 0, len(steps))
	for _, step := range steps {
		out = append(out, web.EnergyRoadmapStep{
			Number: step.Number, Title: step.Title, Detail: step.Detail, State: step.State,
			Action: step.Action, URL: step.URL, Current: step.Current,
		})
	}
	return out
}

func energyWebSystemAssets(assets []energy.Asset) []web.EnergySystemAsset {
	out := make([]web.EnergySystemAsset, 0, len(assets))
	for _, asset := range assets {
		out = append(out, web.EnergySystemAsset{Name: asset.Name, IsPV: asset.Kind == "pv"})
	}
	return out
}

func energyWebAssets(assets []energy.Asset) []web.EnergyAssetOption {
	out := make([]web.EnergyAssetOption, 0, len(assets))
	for _, asset := range assets {
		out = append(out, web.EnergyAssetOption{ID: asset.ID, Name: asset.Name})
	}
	return out
}

func energyWebMaintenance(plans []energyMaintenanceView) []web.EnergyMaintenanceView {
	out := make([]web.EnergyMaintenanceView, 0, len(plans))
	for _, plan := range plans {
		out = append(out, web.EnergyMaintenanceView{
			ID: plan.ID, AssetID: plan.AssetID, AssetName: plan.AssetName, Title: plan.Title,
			IntervalMonths: plan.IntervalMonths, NextDueValue: plan.NextDueValue,
			DueLabel: plan.DueLabel, Tone: plan.Tone, LastCompleted: plan.LastCompleted,
			ContactID: plan.ContactID, ContactName: plan.ContactName,
			DocumentID: plan.DocumentID, DocumentTitle: plan.DocumentTitle,
			IssueID: plan.IssueID, IssueTitle: plan.IssueTitle,
			EvidenceNote: plan.EvidenceNote,
		})
	}
	return out
}

func energyWebMeasures(measures []energyMeasureView) []web.EnergyMeasureView {
	out := make([]web.EnergyMeasureView, 0, len(measures))
	for _, measure := range measures {
		out = append(out, web.EnergyMeasureView{
			ID: measure.ID, IssueID: measure.IssueID, Title: measure.Title,
			Status: measure.Status, StatusLabel: measure.StatusLabel,
			ContactID: measure.ContactID, ContactName: measure.ContactName,
			OfferNote: measure.OfferNote, AppointmentValue: measure.AppointmentValue, Appointment: measure.Appointment,
			WorkNote: measure.WorkNote, EvidenceNote: measure.EvidenceNote,
			BeforeFrom: measure.BeforeFrom, BeforeTo: measure.BeforeTo,
			AfterFrom: measure.AfterFrom, AfterTo: measure.AfterTo,
			BeforePeak: measure.BeforePeak, AfterPeak: measure.AfterPeak,
			BeforeQuality: measure.BeforeQuality, AfterQuality: measure.AfterQuality,
			HasComparison: measure.HasComparison,
		})
	}
	return out
}

func energyWebScenarios(scenarios []energyScenarioView) []web.EnergyScenarioView {
	out := make([]web.EnergyScenarioView, 0, len(scenarios))
	for _, scenario := range scenarios {
		out = append(out, web.EnergyScenarioView{
			Title: scenario.Title, PeakBand: scenario.PeakBand, EffectBand: scenario.EffectBand,
			Uncertainty: scenario.Uncertainty, Assumptions: scenario.Assumptions,
			BilledBand: scenario.BilledBand, HasBilledBand: scenario.HasBilledBand,
			BaselineNote: scenario.BaselineNote, FloorNote: scenario.FloorNote,
		})
	}
	return out
}

func energyWebCaretakers(caretakers []energyCaretakerView) []web.EnergyCaretakerView {
	out := make([]web.EnergyCaretakerView, 0, len(caretakers))
	for _, caretaker := range caretakers {
		out = append(out, web.EnergyCaretakerView{
			Email: caretaker.Email, Name: caretaker.Name,
			CanView: caretaker.CanView, CanConfigure: caretaker.CanConfigure,
			Editable: caretaker.Editable,
		})
	}
	return out
}

func energyWebAssessments(assessments []energyTariffAssessmentView) []web.EnergyTariffAssessmentView {
	out := make([]web.EnergyTariffAssessmentView, 0, len(assessments))
	for _, assessment := range assessments {
		out = append(out, web.EnergyTariffAssessmentView{
			Month: assessment.Month, Profile: assessment.Profile, Peak: assessment.Peak,
			Annual: assessment.Annual, Quality: assessment.Quality, Created: assessment.Created,
		})
	}
	return out
}

func energyWebImports(imports []energyImportView) []web.EnergyImportView {
	out := make([]web.EnergyImportView, 0, len(imports))
	for _, item := range imports {
		out = append(out, web.EnergyImportView{Filename: item.Filename, Date: item.Date})
	}
	return out
}

func energyWebPeaks(peaks []energyPeakView) []web.EnergyPeakView {
	out := make([]web.EnergyPeakView, 0, len(peaks))
	for _, peak := range peaks {
		out = append(out, web.EnergyPeakView{Source: peak.Source, Value: peak.Value})
	}
	return out
}

func energyWebJSON(raw []byte, fallback string) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(fallback)
	}
	return json.RawMessage(raw)
}

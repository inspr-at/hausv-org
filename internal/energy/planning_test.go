package energy

import (
	"testing"
	"time"
)

func TestAustrianDraftTariffIsVersionedAndNeverGuaranteed(t *testing.T) {
	profile := AustrianDraft2027()
	if profile.Status != "draft" || profile.Version == "" || profile.SourceURL == "" || profile.ValidFrom.Year() != 2027 {
		t.Fatalf("profile = %+v", profile)
	}
	estimate := profile.Estimate(14, 20)
	if estimate.ProfileID != profile.ID || estimate.Guaranteed || estimate.BilledKW != 14 {
		t.Fatalf("estimate = %+v", estimate)
	}
	if estimate.AnnualPowerEUR <= 0 || estimate.AssumptionLabel == "" {
		t.Fatalf("estimate must carry draft assumptions: %+v", estimate)
	}
}

func TestDecideControlFailsClosedAndSeparatesStages(t *testing.T) {
	now := time.Now()
	base := ControlInput{
		Mode: ModeObserve, Stage: StageRecommend, Now: now, LastMeasurementAt: now,
		ProjectedKW: 12, TargetKW: 8,
		Capabilities: []DeviceCapability{{ID: "wallbox-limit", Kind: CapabilityLimit}},
	}
	if decision := DecideControl(base); decision.Action != DecisionRecommend {
		t.Fatalf("recommend decision = %+v", decision)
	}
	base.Stage = StageShadow
	if decision := DecideControl(base); decision.Action != DecisionShadow {
		t.Fatalf("shadow decision = %+v", decision)
	}
	base.Mode, base.Stage = ModeActive, StageActive
	if decision := DecideControl(base); decision.Action != DecisionApply {
		t.Fatalf("active decision = %+v", decision)
	}
	base.LastMeasurementAt = now.Add(-11 * time.Minute)
	if decision := DecideControl(base); decision.Action != DecisionNoop || !decision.Safe {
		t.Fatalf("stale decision = %+v", decision)
	}
}

func TestScenarioShowsBandAndNoGuarantee(t *testing.T) {
	result := SimulateScenario(ScenarioInput{
		Name: "Ohne Speicher", BaselinePeakKW: 14, ShiftableKW: 6, ThrottleKW: 2, DataQuality: QualityMeasured,
	})
	if result.Guaranteed || result.ExpectedPeakLowKW > result.ExpectedPeakHighKW || result.PeakEffectHighKW <= 0 {
		t.Fatalf("scenario = %+v", result)
	}
	if result.ComfortAssumption == "" || result.EnergyAssumption == "" || result.Uncertainty == "" {
		t.Fatalf("scenario explanations missing: %+v", result)
	}
}

func TestNextRecommendationDoesNotRequireNewHardware(t *testing.T) {
	profile := DefaultProfile("home", time.Now())
	recommendation := NextRecommendation(profile, []Asset{{Kind: "ev"}}, nil, nil)
	if recommendation.ID != "measure" || recommendation.Prerequisite == "" || recommendation.ImpactRange == "" {
		t.Fatalf("recommendation = %+v", recommendation)
	}
}

func TestPP20AdapterKeepsSpecificsOutsideCore(t *testing.T) {
	capabilities := (PP20Adapter{PlugSwitchEntity: "switch.pp20"}).Capabilities()
	if len(capabilities) != 3 || capabilities[2].Adapter != "pp20" || !capabilities[2].ManualOverride {
		t.Fatalf("capabilities = %+v", capabilities)
	}
}

func TestMeasurePackageStartsDataSparseAndMarketplaceClosed(t *testing.T) {
	recommendation := Recommendation{ID: "measure", Title: "Netzbezug messen", Reason: "Daten fehlen", Benefit: "Peak sichtbar"}
	pkg := NewMeasurePackage("home", recommendation)
	if len(pkg.SharedFields) != 0 || pkg.Status != "draft" || pkg.MarketplaceGate != "closed-until-pilot-evidence" {
		t.Fatalf("package = %+v", pkg)
	}
}

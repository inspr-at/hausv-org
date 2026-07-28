package energy

import (
	"testing"
	"time"
)

func TestApplyProfileSeedsCreatesThreeObserveOnlyPilotsAndNeverOverwrites(t *testing.T) {
	storage := NewMemoryStore()
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	known := map[string]struct{}{"jhw22": {}, "eltern": {}, "schwiegereltern": {}}
	raw := `[
		{"tenant_slug":"jhw22","household_name":"Wohnung Barta","home_type":"apartment","assets":["ev"],"complete":true},
		{"tenant_slug":"eltern","household_name":"Haus Eltern","home_type":"house","assets":["pv","ev","hot-water"]},
		{"tenant_slug":"schwiegereltern","household_name":"Haus Schwiegereltern","home_type":"house","assets":["pv","battery","ev"]}
	]`
	if err := ApplyProfileSeeds(storage, raw, known, now); err != nil {
		t.Fatalf("ApplyProfileSeeds: %v", err)
	}
	for slug := range known {
		profile, ok, err := storage.Profile(slug)
		if err != nil || !ok {
			t.Fatalf("Profile(%s): ok=%v err=%v", slug, ok, err)
		}
		if profile.OperatingMode != ModeObserve || profile.AutomationStage != StageObserve {
			t.Fatalf("%s mode = %s/%s", slug, profile.OperatingMode, profile.AutomationStage)
		}
	}
	jhw22, _, _ := storage.Profile("jhw22")
	jhw22.HouseholdName = "Vom Nutzer geändert"
	if err := storage.SaveProfile(jhw22); err != nil {
		t.Fatal(err)
	}
	if err := ApplyProfileSeeds(storage, raw, known, now.Add(time.Hour)); err != nil {
		t.Fatalf("second ApplyProfileSeeds: %v", err)
	}
	jhw22, _, _ = storage.Profile("jhw22")
	if jhw22.HouseholdName != "Vom Nutzer geändert" {
		t.Fatalf("seed overwrote profile: %+v", jhw22)
	}
}

func TestApplyProfileSeedsRejectsUnknownTenant(t *testing.T) {
	err := ApplyProfileSeeds(NewMemoryStore(), `[{"tenant_slug":"foreign","household_name":"Nope"}]`, map[string]struct{}{"jhw22": {}}, time.Now())
	if err == nil {
		t.Fatal("expected unknown tenant error")
	}
}

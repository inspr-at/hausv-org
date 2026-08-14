package energy

import (
	"testing"
	"time"
)

func TestApplyProfileSeedsCreatesThreeObserveOnlyPilotsAndNeverOverwrites(t *testing.T) {
	storage := NewMemoryStore()
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	known := map[string]struct{}{"demo": {}, "haus-a": {}, "haus-b": {}}
	raw := `[
		{"tenant_slug":"demo","household_name":"Demo-Wohnung","home_type":"apartment","assets":["ev"],"complete":true},
		{"tenant_slug":"haus-a","household_name":"Haus A","home_type":"house","assets":["pv","ev","hot-water"]},
		{"tenant_slug":"haus-b","household_name":"Haus B","home_type":"house","assets":["pv","battery","ev"]}
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
	demo, _, _ := storage.Profile("demo")
	demo.HouseholdName = "Vom Nutzer geändert"
	if err := storage.SaveProfile(demo); err != nil {
		t.Fatal(err)
	}
	if err := ApplyProfileSeeds(storage, raw, known, now.Add(time.Hour)); err != nil {
		t.Fatalf("second ApplyProfileSeeds: %v", err)
	}
	demo, _, _ = storage.Profile("demo")
	if demo.HouseholdName != "Vom Nutzer geändert" {
		t.Fatalf("seed overwrote profile: %+v", demo)
	}
}

func TestApplyProfileSeedsRejectsUnknownTenant(t *testing.T) {
	err := ApplyProfileSeeds(NewMemoryStore(), `[{"tenant_slug":"foreign","household_name":"Nope"}]`, map[string]struct{}{"demo": {}}, time.Now())
	if err == nil {
		t.Fatal("expected unknown tenant error")
	}
}

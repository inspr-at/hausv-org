package version

import (
	"testing"
	"time"
)

func TestCalendarGrammarAndDates(t *testing.T) {
	for _, v := range []string{"260914170935.0.0", "240229235959.0.0", "100101000000.0.0", "991231235959.0.0"} {
		if _, err := ParseCalendar(v); err != nil {
			t.Errorf("valid %s: %v", v, err)
		}
	}
	for _, v := range []string{"260229120000.0.0", "260431120000.0.0", "260914240000.0.0", "260914126000.0.0", "260914120060.0.0", "261314120000.0.0", "260914120000.1.0", "1.11.0", "26.09.14", "v260914120000.0.0", "260914120000.0.0-demo", "260914120000.0.0+abc", "260914120000.0.0\n", "090101000000.0.0"} {
		if _, err := ParseCalendar(v); err == nil {
			t.Errorf("accepted %q", v)
		}
	}
}
func TestReservationRejectsSameSecondAndClockRollback(t *testing.T) {
	now := time.Date(2026, 9, 14, 19, 9, 35, 0, time.FixedZone("Vienna", 7200))
	v, err := Reserve(now, "")
	if err != nil || v != FirstCalendarVersion {
		t.Fatalf("UTC reservation %s %v", v, err)
	}
	for _, at := range []time.Time{now, now.Add(-time.Second)} {
		if _, err := Reserve(at, v); err == nil {
			t.Fatal("collision/rollback accepted")
		}
	}
	if next, err := Reserve(now.Add(time.Second), v); err != nil || next <= v {
		t.Fatal("later second rejected")
	}
}
func TestMixedEraRequiresExplicitSchemeAndAnchor(t *testing.T) {
	legacy := ReleaseIdentity{VersionScheme: "legacy", Version: LastLegacyVersion, ReleaseChannel: "production"}
	calendar := ReleaseIdentity{VersionScheme: Scheme, Version: FirstCalendarVersion, ReleaseChannel: "production", ReleaseSequence: 1}
	if n, err := Compare(legacy, calendar); err != nil || n != -1 {
		t.Fatal("migration anchor order", n, err)
	}
	if n, err := Compare(calendar, legacy); err != nil || n != 1 {
		t.Fatal("reverse anchor order", n, err)
	}
	for _, scheme := range []string{"", "semver", "inspr-calendar-v1", "unknown"} {
		v := calendar
		v.VersionScheme = scheme
		if _, err := Compare(v, calendar); err == nil {
			t.Errorf("accepted scheme %q", scheme)
		}
	}
	v := legacy
	v.Version = "1.10.0"
	if _, err := Compare(v, calendar); err == nil {
		t.Fatal("invented cross-era order")
	}
	v = calendar
	v.ReleaseChannel = "demo"
	if _, err := Compare(v, calendar); err == nil {
		t.Fatal("compared different channels")
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

var categories = map[string]float64{
	"reparatur": 30, "hausordnung": 12, "betriebskosten": 12, "freigabe": 8,
	"schluessel": 6, "beleg": 8, "termin": 6, "versicherung": 4,
	"uebergabe": 4, "stammdaten": 3, "winterdienst_garten": 4, "parkplatz": 2, "sonstiges": 1,
}

func TestCommittedSeed(t *testing.T) {
	seedDir := filepath.Join("..", "seed")
	var houses []house
	readJSON(t, filepath.Join(seedDir, "houses.json"), &houses)
	var persons []person
	readJSON(t, filepath.Join(seedDir, "persons.json"), &persons)
	var items []intakeItem
	readJSON(t, filepath.Join(seedDir, "intake.json"), &items)
	var templates []textTemplate
	readJSON(t, filepath.Join(seedDir, "textbausteine.json"), &templates)
	var events []event
	readJSON(t, filepath.Join(seedDir, "events.json"), &events)
	var announcements []announcement
	readJSON(t, filepath.Join(seedDir, "announcements.json"), &announcements)
	var org map[string]any
	readJSON(t, filepath.Join(seedDir, "org.json"), &org)

	if len(items) < 350 || len(items) > 450 {
		t.Fatalf("intake count = %d, want 350..450", len(items))
	}
	if len(houses) != 12 {
		t.Fatalf("house count = %d, want 12", len(houses))
	}
	houseSet := map[string]bool{}
	for index, h := range houses {
		houseSet[h.Slug] = true
		if len(h.Units) != houseSpecs[index].count+parkingCount(houseSpecs[index].count) {
			t.Errorf("house %s has %d units", h.Slug, len(h.Units))
		}
	}
	templateSet := map[string]bool{}
	for _, template := range templates {
		templateSet[template.Key] = true
	}
	for category := range categories {
		covered := false
		for _, template := range templates {
			covered = covered || template.Category == category
		}
		if !covered {
			t.Errorf("category %s has no text template", category)
		}
	}

	ids := map[string]bool{}
	bodies := map[string]bool{}
	counts := map[string]int{}
	precomputedCount, autoDone, today, unassigned, older, olderApproved := 0, 0, 0, 0, 0, 0
	sources := map[string]int{}
	for _, item := range items {
		if ids[item.ID] {
			t.Errorf("duplicate intake id %s", item.ID)
		}
		ids[item.ID] = true
		bodies[item.Body] = true
		if item.House != "" && !houseSet[item.House] {
			t.Errorf("item %s has unknown house %s", item.ID, item.House)
		}
		if _, ok := categories[item.Truth.Category]; !ok {
			t.Errorf("item %s has unknown category %s", item.ID, item.Truth.Category)
		}
		counts[item.Truth.Category]++
		sources[item.Source]++
		if item.House == "" {
			unassigned++
		}
		if !templateSet[item.Truth.TemplateKey] {
			t.Errorf("item %s has unknown template %s", item.ID, item.Truth.TemplateKey)
		}
		received, err := time.Parse(time.RFC3339, item.ReceivedAt)
		if err != nil || !strings.HasSuffix(item.ReceivedAt, "+02:00") {
			t.Errorf("item %s has invalid received_at %q", item.ID, item.ReceivedAt)
		}
		if received.Format("2006-01-02") == "2026-09-09" {
			today++
		}
		if received.Before(time.Date(2026, 9, 7, 0, 0, 0, 0, received.Location())) {
			older++
			if item.StatusHint == "approved" {
				olderApproved++
			}
		}
		if item.Precomputed != nil {
			precomputedCount++
			if !templateSet[item.Precomputed.TemplateKey] || strings.Contains(item.Precomputed.Reply, "{{") {
				t.Errorf("item %s has an invalid rendered reply", item.ID)
			}
		}
		if item.StatusHint == "auto_done" {
			autoDone++
			if item.Truth.Category != "beleg" && item.Truth.Category != "termin" {
				t.Errorf("item %s auto-completed category %s", item.ID, item.Truth.Category)
			}
			if item.Precomputed == nil || item.Precomputed.Confidence.Overall < 0.9 {
				t.Errorf("item %s auto-completed without confident precomputation", item.ID)
			}
		}
	}
	if len(bodies) < 120 {
		t.Errorf("unique request bodies = %d, want at least 120", len(bodies))
	}
	for category, want := range categories {
		got := float64(counts[category]) * 100 / float64(len(items))
		if got < want-5 || got > want+5 {
			t.Errorf("category %s weight %.1f%%, want %.1f%% ±5", category, got, want)
		}
	}
	precomputedPercent := float64(precomputedCount) * 100 / float64(len(items))
	if precomputedPercent < 75 || precomputedPercent > 85 {
		t.Errorf("precomputed = %.1f%%, want 75..85", precomputedPercent)
	}
	if autoDone < 25 {
		t.Errorf("auto_done = %d, want at least 25", autoDone)
	}
	if today < 35 || today > 45 {
		t.Errorf("today items = %d, want about 40", today)
	}
	if sources["email"] != 240 || sources["phone"] != 100 || sources["portal"] != 60 {
		t.Errorf("source mix = %#v, want email 240, phone 100, portal 60", sources)
	}
	if unassigned != 20 {
		t.Errorf("unassigned = %d, want 20", unassigned)
	}
	if older == 0 || olderApproved*2 <= older {
		t.Errorf("older approved = %d/%d, want a majority", olderApproved, older)
	}

	assertExampleEmails(t, seedDir)
	assertUnitMemberships(t, houses, persons)
}

func TestGeneratorIsDeterministic(t *testing.T) {
	out := t.TempDir()
	if err := generate(out); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"houses.json", "persons.json", "intake.json", "textbausteine.json", "events.json", "announcements.json", "org.json", "annual-statement.json"} {
		got, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join("..", "seed", name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("committed %s differs from generated output", name)
		}
	}
}

func readJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, value); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
}

func assertExampleEmails(t *testing.T, seedDir string) {
	t.Helper()
	emailPattern := regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+`)
	for _, name := range []string{"houses.json", "persons.json", "intake.json", "org.json"} {
		raw, err := os.ReadFile(filepath.Join(seedDir, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, email := range emailPattern.FindAllString(string(raw), -1) {
			if !strings.HasSuffix(strings.ToLower(email), ".example") {
				t.Errorf("%s contains non-example email %s", name, fmt.Sprintf("%q", email))
			}
		}
	}
}

func TestFixtureEmailDoesNotDoubleExampleSuffix(t *testing.T) {
	for input, want := range map[string]string{
		"alina@example.com":         "alina@example.example",
		"alina@musterstadt.example": "alina@musterstadt.example",
		"ALINA@MUSTERSTADT.EXAMPLE": "ALINA@MUSTERSTADT.EXAMPLE",
	} {
		if got := fixtureEmail(input); got != want {
			t.Errorf("fixtureEmail(%q) = %q, want %q", input, got, want)
		}
	}
}

func assertUnitMemberships(t *testing.T, houses []house, persons []person) {
	t.Helper()
	if len(houses) == 0 {
		t.Fatal("missing houses")
	}
	first := houses[0]
	var top1, top3 unit
	parking := 0
	for _, item := range first.Units {
		switch item.Label {
		case "Top 1":
			top1 = item
		case "Top 3":
			top3 = item
		}
		if item.UnitType == "Stellplatz" {
			parking++
		}
	}
	if top1.OwnerEmail != "alina.eigentuemer@musterstadt.example" || top3.TenantEmail != "matthias.mieter@musterstadt.example" {
		t.Fatalf("top memberships = %#v %#v", top1, top3)
	}
	if parking < 4 || parking > 12 {
		t.Fatalf("parking spaces = %d", parking)
	}
	for _, item := range first.Units {
		if item.OwnerEmail == "" && item.TenantEmail == "" {
			continue
		}
		found := false
		for _, person := range persons {
			if person.Email != item.OwnerEmail && person.Email != item.TenantEmail {
				continue
			}
			for _, membership := range person.Memberships {
				found = found || (membership.House == first.Slug && containsUnit(membership.Units, item.Label))
			}
		}
		if !found {
			t.Errorf("%s parties do not come from a membership", item.Label)
		}
	}
}

func TestCommittedDocumentsMatchDeterministicGenerator(t *testing.T) {
	houses, _ := buildHousesAndPersons()
	want, err := json.MarshalIndent(buildDocuments(houses), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join("..", "seed", "documents.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(got)) != string(want) {
		t.Fatal("documents.json differs from deterministic generator")
	}
}

// HAUSV-729: mirrors the loader rule in internal/demo/seed.go (every manual
// item and every fourth approved/edited item becomes an open issue) so the
// committed seed cannot regress to two open issues of one house with the same
// subject. The behaviour itself is covered by the loader test in internal/demo.
func TestCommittedSeedOpensDistinctSubjectsPerHouse(t *testing.T) {
	var items []intakeItem
	readJSON(t, filepath.Join("..", "seed", "intake.json"), &items)
	seen := map[string]map[string]string{}
	opened := 0
	for index, item := range items {
		open := item.StatusHint == "manual" || ((item.StatusHint == "approved" || item.StatusHint == "edited") && index%4 == 0)
		if !open || item.House == "" {
			continue
		}
		opened++
		if seen[item.House] == nil {
			seen[item.House] = map[string]string{}
		}
		if other, dup := seen[item.House][item.Subject]; dup {
			t.Errorf("%s: %s and %s open with the same subject %q", item.House, other, item.ID, item.Subject)
		}
		seen[item.House][item.Subject] = item.ID
	}
	if opened < 40 {
		t.Fatalf("only %d items open as issues, want at least 40", opened)
	}
}

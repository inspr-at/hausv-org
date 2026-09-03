package store

import (
	"context"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

func TestIntakeRepositoryRoundTripFiltersAndIsolation(t *testing.T) {
	database := dbtest.Open(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 3, 8, 0, 0, 0, time.UTC)
	a := BindIntakeRepository(database, "org-a")
	b := BindIntakeRepository(database, "org-b")
	items := []IntakeItem{
		{ID: "one", Source: IntakeSourceEmail, Status: IntakeStatusOpen, TenantSlug: "haus-a", Subject: "A", Body: "B", ReceivedAt: now},
		{ID: "two", Source: IntakeSourcePhone, Status: IntakeStatusProposed, Subject: "C", Body: "D", ReceivedAt: now.Add(time.Hour), Suggestion: &IntakeSuggestion{Source: "seed", Assignee: "vera"}},
	}
	for _, item := range items {
		if err := a.Create(ctx, item); err != nil {
			t.Fatal(err)
		}
	}
	if err := b.Create(ctx, IntakeItem{ID: "one", Source: IntakeSourcePortal, Status: IntakeStatusAuto, Subject: "other", Body: "other", ReceivedAt: now}); err != nil {
		t.Fatal(err)
	}
	got, err := a.Get(ctx, "one")
	if err != nil || got.Organisation != "org-a" || got.TenantSlug != "haus-a" {
		t.Fatalf("round trip = %#v, %v", got, err)
	}
	if count, err := a.Count(ctx, IntakeFilter{Statuses: []IntakeStatus{IntakeStatusProposed}, Unassigned: true, Assignee: "vera"}); err != nil || count != 1 {
		t.Fatalf("filtered count = %d, %v", count, err)
	}
	if list, err := a.List(ctx, IntakeFilter{Sources: []IntakeSource{IntakeSourceEmail}, TenantSlug: "haus-a"}); err != nil || len(list) != 1 || list[0].ID != "one" {
		t.Fatalf("filtered list = %#v, %v", list, err)
	}
	if list, err := b.List(ctx, IntakeFilter{}); err != nil || len(list) != 1 || list[0].Subject != "other" {
		t.Fatalf("org B leaked = %#v, %v", list, err)
	}
	if err := a.UpdateSuggestion(ctx, "one", IntakeSuggestion{Source: "model", Category: IntakeCategoryRepair}, IntakeStatusProposed); err != nil {
		t.Fatal(err)
	}
	if err := a.UpdateHandling(ctx, "one", IntakeHandling{Action: "approved", ByName: "Vera"}, IntakeStatusApproved, "issue-1"); err != nil {
		t.Fatal(err)
	}
	if err := a.Assign(ctx, "two", "haus-b", "Top 2"); err != nil {
		t.Fatal(err)
	}
	got, _ = a.Get(ctx, "one")
	if got.Status != IntakeStatusApproved || got.IssueID != "issue-1" || got.Suggestion == nil || got.Handling == nil {
		t.Fatalf("updates not persisted: %#v", got)
	}
}

func TestOrgSettingsAndTextbausteine(t *testing.T) {
	database := dbtest.Open(t)
	ctx := context.Background()
	settings := BindOrgSettingsRepository(database, "musterstadt")
	got, err := settings.Get(ctx)
	if err != nil || got.AutoThreshold != 0.9 || got.AutoEnabled || len(got.TrustLevels) != len(IntakeCategories()) {
		t.Fatalf("defaults = %#v, %v", got, err)
	}
	got.Name = "Musterstadt"
	got.TrustLevels[IntakeCategoryReceipt] = "auto"
	got.AutoEnabled = true
	if err := settings.Save(ctx, got); err != nil {
		t.Fatal(err)
	}
	if err := settings.Increment(ctx, "approved"); err != nil {
		t.Fatal(err)
	}
	got, _ = settings.Get(ctx)
	if got.Counters.Approved != 1 || got.TrustLevels[IntakeCategoryReceipt] != "auto" {
		t.Fatalf("saved settings = %#v", got)
	}
	templates := BindTextbausteinRepository(database, "musterstadt")
	if err := templates.Upsert(ctx, Textbaustein{Key: "hello", Category: IntakeCategoryOther, Title: "Hallo", Body: "Hallo {{Name}} / {{Fehlt}}", Placeholders: []string{"Name"}, Active: true}); err != nil {
		t.Fatal(err)
	}
	item, err := templates.Get(ctx, "hello")
	if err != nil || RenderTextbaustein(item.Body, map[string]string{"Name": "Rita"}) != "Hallo Rita / " {
		t.Fatalf("template = %#v, %v", item, err)
	}
}

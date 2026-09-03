package store

import (
	"testing"
	"time"
)

func TestIssueIntakeFieldsRoundTrip(t *testing.T) {
	_, lanes := testLanes(t)
	tenant := testTenantRef("demo")
	repo, _ := BindIssueRepository(NewSQLIssueStore(lanes, ""), tenant)
	due := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	saved, err := repo.Create(ResidentIssue{
		ID: "stable", Source: "email", DueAt: due, IntakeID: "in-1", AuthorEmail: "rita@example.com",
		Category: IntakeCategoryRepair, Title: "Wasser", Body: "Fleck", LocationType: IssueLocationUnit,
		Status: IssueStatusNew, Priority: IssuePriorityHigh,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, ok := repo.Get(saved.ID)
	if !ok || got.Source != "email" || got.IntakeID != "in-1" || !got.DueAt.Equal(due) {
		t.Fatalf("issue intake fields = %#v", got)
	}
}

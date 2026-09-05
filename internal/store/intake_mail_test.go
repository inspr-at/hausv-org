package store

import (
	"context"
	"testing"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

func TestIntakeMailSeenIsIdempotentPerOrganisation(t *testing.T) {
	database := dbtest.Open(t)
	ctx := context.Background()
	musterstadt := BindIntakeMailSeenRepository(database, "musterstadt")
	seestadt := BindIntakeMailSeenRepository(database, "seestadt")

	seen, err := musterstadt.Seen(ctx, "<abc@example.com>")
	if err != nil || seen {
		t.Fatalf("fresh id reported seen=%v err=%v", seen, err)
	}
	if err := musterstadt.Record(ctx, "<abc@example.com>", "in-0001"); err != nil {
		t.Fatal(err)
	}
	// Angle brackets and whitespace are presentation, not identity.
	seen, err = musterstadt.Seen(ctx, "  abc@example.com ")
	if err != nil || !seen {
		t.Fatalf("recorded id reported seen=%v err=%v", seen, err)
	}
	if err := musterstadt.Record(ctx, "abc@example.com", "in-0002"); err != nil {
		t.Fatalf("second record of the same id must be a no-op, got %v", err)
	}
	count, err := musterstadt.Count(ctx)
	if err != nil || count != 1 {
		t.Fatalf("count = %d err=%v, want 1", count, err)
	}

	seen, err = seestadt.Seen(ctx, "abc@example.com")
	if err != nil || seen {
		t.Fatalf("another organisation sees the row: seen=%v err=%v", seen, err)
	}
}

func TestIntakeMailSeenRejectsEmptyIDs(t *testing.T) {
	database := dbtest.Open(t)
	ctx := context.Background()
	repo := BindIntakeMailSeenRepository(database, "musterstadt")
	if _, err := repo.Seen(ctx, " <> "); err == nil {
		t.Fatal("empty message id accepted by Seen")
	}
	if err := repo.Record(ctx, "", "in-0001"); err == nil {
		t.Fatal("empty message id accepted by Record")
	}
}

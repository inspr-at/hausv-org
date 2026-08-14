package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
)

func sampleBallot() Ballot {
	return Ballot{
		TenantSlug: "demo",
		Title:      "Neue Heizung",
		Options:    []string{"Ja", "Nein"},
		Type:       BallotTypeCircular,
		Weighting:  BallotWeightingPerHead,
		CreatedBy:  "admin@example.com",
	}
}

func TestVoteStorageParity(t *testing.T) {
	backends := map[string]func(t *testing.T) VoteStorage{
		"json": func(t *testing.T) VoteStorage {
			s, err := NewVoteStore(filepath.Join(t.TempDir(), "votes.json"))
			if err != nil {
				t.Fatalf("json store: %v", err)
			}
			return s
		},
		"sqlite": func(t *testing.T) VoteStorage {
			database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatalf("db open: %v", err)
			}
			t.Cleanup(func() { database.Close() })
			return NewSQLVoteStore(database)
		},
	}

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			s := build(t)

			// Needs at least two options.
			bad := sampleBallot()
			bad.Options = []string{"Nur eine"}
			if _, err := s.Create(bad); err == nil {
				t.Fatal("a ballot with one option must error")
			}

			created, err := s.Create(sampleBallot())
			if err != nil {
				t.Fatalf("create: %v", err)
			}
			if created.ID == "" || created.Status != BallotStatusDraft {
				t.Fatalf("created bad: %+v", created)
			}
			id := created.ID

			// A draft does not accept votes.
			if _, _, err := s.CastVote("demo", id, "a@example.com", "Ja", 1, now); err == nil {
				t.Fatal("voting on a draft must error")
			}

			// Open, then vote.
			opened, ok, err := s.Open("demo", id, now)
			if err != nil || !ok || opened.Status != BallotStatusOpen {
				t.Fatalf("open: err=%v ok=%v status=%q", err, ok, opened.Status)
			}
			voted, ok, err := s.CastVote("demo", id, "a@example.com", "Ja", 1, now.Add(time.Minute))
			if err != nil || !ok {
				t.Fatalf("cast: err=%v ok=%v", err, ok)
			}
			if v, exists := voted.Votes["a@example.com"]; !exists || v.Option != "Ja" || v.Weight != 1 {
				t.Fatalf("vote not recorded: %+v", voted.Votes)
			}
			// Re-voting replaces the previous choice.
			revoted, _, err := s.CastVote("demo", id, "a@example.com", "Nein", 2, now.Add(2*time.Minute))
			if err != nil {
				t.Fatalf("re-cast: %v", err)
			}
			if len(revoted.Votes) != 1 || revoted.Votes["a@example.com"].Option != "Nein" {
				t.Fatalf("re-vote must replace: %+v", revoted.Votes)
			}
			// Unknown option is rejected.
			if _, _, err := s.CastVote("demo", id, "b@example.com", "Vielleicht", 1, now.Add(3*time.Minute)); err == nil {
				t.Fatal("unknown option must error")
			}
			// Zero weight is rejected.
			if _, _, err := s.CastVote("demo", id, "b@example.com", "Ja", 0, now.Add(3*time.Minute)); err == nil {
				t.Fatal("non-positive weight must error")
			}

			// Reminders.
			reminded, ok, err := s.MarkReminderSent("demo", id, []string{"a@example.com", "b@example.com"}, now.Add(4*time.Minute))
			if err != nil || !ok || len(reminded.ReminderSentAt) != 2 {
				t.Fatalf("reminder: err=%v ok=%v map=%+v", err, ok, reminded.ReminderSentAt)
			}

			// Close, and a closed ballot cannot reopen or accept votes.
			closed, ok, err := s.Close("demo", id, now.Add(5*time.Minute))
			if err != nil || !ok || closed.Status != BallotStatusClosed {
				t.Fatalf("close: err=%v ok=%v status=%q", err, ok, closed.Status)
			}
			if _, _, err := s.Open("demo", id, now.Add(6*time.Minute)); err == nil {
				t.Fatal("a closed ballot must not reopen")
			}
			if _, _, err := s.CastVote("demo", id, "c@example.com", "Ja", 1, now.Add(6*time.Minute)); err == nil {
				t.Fatal("voting on a closed ballot must error")
			}

			// Unknown ballot: found=false, no error.
			if _, ok, err := s.Close("demo", "nope", now); ok || err != nil {
				t.Fatalf("closing unknown: ok=%v err=%v", ok, err)
			}

			if got := s.ListTenant("demo"); len(got) != 1 {
				t.Fatalf("list = %+v", got)
			}
			if got, ok := s.Get("demo", id); !ok || got.ID != id {
				t.Fatalf("get = %+v ok=%v", got, ok)
			}

			// Delete.
			if removed, err := s.Delete("demo", id); err != nil || !removed {
				t.Fatalf("delete: removed=%v err=%v", removed, err)
			}
			if removed, err := s.Delete("demo", id); err != nil || removed {
				t.Fatalf("second delete: removed=%v err=%v", removed, err)
			}
			if _, ok := s.Get("demo", id); ok {
				t.Fatal("deleted ballot must be gone")
			}
		})
	}
}

// A ballot past its closing time auto-closes, and that closure must be
// persisted even though the call reports an error.
func TestVoteCastAfterCloseTimePersistsClosureParity(t *testing.T) {
	backends := map[string]func(t *testing.T) VoteStorage{
		"json": func(t *testing.T) VoteStorage {
			s, err := NewVoteStore(filepath.Join(t.TempDir(), "votes.json"))
			if err != nil {
				t.Fatalf("json store: %v", err)
			}
			return s
		},
		"sqlite": func(t *testing.T) VoteStorage {
			database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatalf("db open: %v", err)
			}
			t.Cleanup(func() { database.Close() })
			return NewSQLVoteStore(database)
		},
	}

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			s := build(t)
			item := sampleBallot()
			item.ClosesAt = now.Add(time.Hour)
			created, err := s.Create(item)
			if err != nil {
				t.Fatalf("create: %v", err)
			}
			if _, _, err := s.Open("demo", created.ID, now); err != nil {
				t.Fatalf("open: %v", err)
			}
			// Vote after the closing time.
			if _, ok, err := s.CastVote("demo", created.ID, "a@example.com", "Ja", 1, now.Add(2*time.Hour)); err == nil || !ok {
				t.Fatalf("voting past ClosesAt must error but report found: ok=%v err=%v", ok, err)
			}
			got, ok := s.Get("demo", created.ID)
			if !ok || got.Status != BallotStatusClosed {
				t.Fatalf("auto-close must be persisted, got status %q", got.Status)
			}
		})
	}
}

func TestVoteCloseExpiredTenantParity(t *testing.T) {
	backends := map[string]func(t *testing.T) VoteStorage{
		"json": func(t *testing.T) VoteStorage {
			s, err := NewVoteStore(filepath.Join(t.TempDir(), "votes.json"))
			if err != nil {
				t.Fatalf("json store: %v", err)
			}
			return s
		},
		"sqlite": func(t *testing.T) VoteStorage {
			database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatalf("db open: %v", err)
			}
			t.Cleanup(func() { database.Close() })
			return NewSQLVoteStore(database)
		},
	}

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			s := build(t)

			expiring := sampleBallot()
			expiring.Title = "Läuft ab"
			expiring.ClosesAt = now.Add(time.Hour)
			a, err := s.Create(expiring)
			if err != nil {
				t.Fatalf("create a: %v", err)
			}
			if _, _, err := s.Open("demo", a.ID, now); err != nil {
				t.Fatalf("open a: %v", err)
			}

			lasting := sampleBallot()
			lasting.Title = "Läuft weiter"
			lasting.ClosesAt = now.Add(48 * time.Hour)
			b, err := s.Create(lasting)
			if err != nil {
				t.Fatalf("create b: %v", err)
			}
			if _, _, err := s.Open("demo", b.ID, now); err != nil {
				t.Fatalf("open b: %v", err)
			}

			// Nothing expired yet.
			if got, err := s.CloseExpiredTenant("demo", now.Add(time.Minute)); err != nil || len(got) != 0 {
				t.Fatalf("premature close: got=%+v err=%v", got, err)
			}
			// Only the expired one closes.
			closed, err := s.CloseExpiredTenant("demo", now.Add(2*time.Hour))
			if err != nil || len(closed) != 1 || closed[0].ID != a.ID {
				t.Fatalf("close expired: closed=%+v err=%v", closed, err)
			}
			if got, _ := s.Get("demo", a.ID); got.Status != BallotStatusClosed {
				t.Fatalf("expired ballot status = %q", got.Status)
			}
			if got, _ := s.Get("demo", b.ID); got.Status != BallotStatusOpen {
				t.Fatalf("unexpired ballot must stay open, got %q", got.Status)
			}
			// Running again closes nothing new.
			if got, err := s.CloseExpiredTenant("demo", now.Add(2*time.Hour)); err != nil || len(got) != 0 {
				t.Fatalf("second run: got=%+v err=%v", got, err)
			}
		})
	}
}

func TestSQLVoteImportFromJSON(t *testing.T) {
	jsonStore, err := NewVoteStore(filepath.Join(t.TempDir(), "votes.json"))
	if err != nil {
		t.Fatalf("json store: %v", err)
	}
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	created, err := jsonStore.Create(sampleBallot())
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, _, err := jsonStore.Open("demo", created.ID, now); err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, _, err := jsonStore.CastVote("demo", created.ID, "a@example.com", "Ja", 3, now.Add(time.Minute)); err != nil {
		t.Fatalf("cast: %v", err)
	}

	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	defer database.Close()
	sqlStore := NewSQLVoteStore(database)

	for i := 0; i < 2; i++ {
		if err := sqlStore.ImportBallots(jsonStore); err != nil {
			t.Fatalf("import %d: %v", i, err)
		}
	}
	if got := sqlStore.ListTenant("demo"); len(got) != 1 {
		t.Fatalf("imported %d ballots, want 1", len(got))
	}
	// Cast votes survive the import, weight included.
	got, ok := sqlStore.Get("demo", created.ID)
	if !ok || got.Status != BallotStatusOpen {
		t.Fatalf("imported ballot = %+v ok=%v", got, ok)
	}
	if v, exists := got.Votes["a@example.com"]; !exists || v.Option != "Ja" || v.Weight != 3 {
		t.Fatalf("votes lost in import: %+v", got.Votes)
	}
}

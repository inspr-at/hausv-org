package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// HAUSV-140: the stores are the app's source of truth and had zero tests. These
// run under `go test -race` in CI. They hammer a persisted store from many
// goroutines and assert (a) the race detector stays quiet and (b) the file on
// disk is always valid JSON — never a torn write.
func TestAnnouncementStoreConcurrentCreateListDeletePersist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "announcements.json")
	s, err := NewAnnouncementStore(path)
	if err != nil {
		t.Fatal(err)
	}
	repo, _ := BindAnnouncementRepository(s, "demo")

	const workers = 16
	const perWorker = 40
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				created, err := repo.Create(Announcement{
					TenantSlug: "demo",
					Title:      fmt.Sprintf("w%d-%d", w, i),
					Body:       "x",
					AuthorName: "t",
				})
				if err != nil {
					t.Errorf("create: %v", err)
					return
				}
				_ = repo.List()
				if i%3 == 0 {
					if _, err := repo.Delete(created.ID); err != nil {
						t.Errorf("delete: %v", err)
						return
					}
				}
			}
		}(w)
	}
	wg.Wait()

	// The file must always be complete, parseable JSON — proof no torn write
	// ever became visible.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var data AnnouncementStoreData
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("persisted store is not valid JSON after concurrent writes: %v", err)
	}

	// Reloading yields the same count the in-memory store reports.
	reloaded, err := NewAnnouncementStore(path)
	if err != nil {
		t.Fatalf("reload after concurrent writes: %v", err)
	}
	reloadedRepo, _ := BindAnnouncementRepository(reloaded, "demo")
	if got, want := len(reloadedRepo.List()), len(repo.List()); got != want {
		t.Fatalf("reloaded count %d != in-memory count %d", got, want)
	}
}

// A second store type, exercising an upsert-shaped API under contention.
func TestUnitPaymentStatusStoreConcurrentSet(t *testing.T) {
	dir := t.TempDir()
	s, err := NewUnitPaymentStatusStore(filepath.Join(dir, "ups.json"))
	if err != nil {
		t.Fatal(err)
	}
	repository, _ := BindUnitPaymentStatusRepository(s, "demo")
	var wg sync.WaitGroup
	for w := 0; w < 16; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				if _, err := repository.Set(UnitPaymentStatus{
					TenantSlug: "demo",
					UnitID:     fmt.Sprintf("unit-%d", w),
					Status:     UnitPaymentStatusPaid,
				}); err != nil {
					t.Errorf("set: %v", err)
					return
				}
				_ = repository.List()
			}
		}(w)
	}
	wg.Wait()
	if got := len(repository.List()); got != 16 {
		t.Fatalf("want 16 distinct units, got %d", got)
	}
}

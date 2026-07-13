package store

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

// HAUSV-145: concurrent adds of DIFFERENT units must all survive — the whole-
// slice overwrite lost some; UpsertUnit under one lock must not.
func TestUnitStoreConcurrentUpsertKeepsAll(t *testing.T) {
	s, err := NewUnitStore(filepath.Join(t.TempDir(), "units.json"))
	if err != nil {
		t.Fatal(err)
	}
	const n = 50
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			dup, err := s.UpsertUnit("jhw22", "", Unit{ID: fmt.Sprintf("u%02d", i), TenantSlug: "jhw22", Label: fmt.Sprintf("Unit %d", i)})
			if err != nil || dup {
				t.Errorf("upsert u%02d: dup=%v err=%v", i, dup, err)
			}
		}(i)
	}
	wg.Wait()
	if got := len(s.ListTenant("jhw22")); got != n {
		t.Fatalf("concurrent adds lost updates: got %d units, want %d", got, n)
	}
}

// Duplicate semantics preserved: a second create with the same ID is rejected.
func TestUnitStoreUpsertDuplicateDetection(t *testing.T) {
	s, _ := NewUnitStore(filepath.Join(t.TempDir(), "units.json"))
	if dup, _ := s.UpsertUnit("jhw22", "", Unit{ID: "a", TenantSlug: "jhw22", Label: "A"}); dup {
		t.Fatal("first create must not be a duplicate")
	}
	if dup, _ := s.UpsertUnit("jhw22", "", Unit{ID: "a", TenantSlug: "jhw22", Label: "A2"}); !dup {
		t.Fatal("second create of same ID must be a duplicate")
	}
	// Editing the same unit in place is NOT a duplicate.
	if dup, _ := s.UpsertUnit("jhw22", "a", Unit{ID: "a", TenantSlug: "jhw22", Label: "A3"}); dup {
		t.Fatal("in-place edit must not be a duplicate")
	}
}

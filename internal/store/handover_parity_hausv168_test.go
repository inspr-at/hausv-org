package store

import (
	"path/filepath"
	"testing"
	"time"
)

func sampleHandover(id string) HandoverRecord {
	return HandoverRecord{
		ID:         id,
		TenantSlug: "demo",
		Title:      "Übergabe " + id,
		CreatedBy:  "admin@example.com",
		Confirmations: []HandoverConfirmation{
			{Role: "incoming", Name: "Neu Mieter", Email: "neu@example.com", TokenHash: HandoverTokenHash("token-" + id)},
		},
	}
}

func TestHandoverStorageParity(t *testing.T) {
	backends := map[string]func(t *testing.T) HandoverStorage{
		"json": func(t *testing.T) HandoverStorage {
			s, err := NewHandoverStore(filepath.Join(t.TempDir(), "handovers.json"))
			if err != nil {
				t.Fatalf("json store: %v", err)
			}
			return s
		},
		"sqlite": func(t *testing.T) HandoverStorage {
			database, lanes := testLanes(t)
			t.Cleanup(func() { database.Close() })
			return NewSQLHandoverStore(lanes)
		},
	}

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			storage := build(t)
			s, ok := BindHandoverRepository(storage, testTenantRef("demo"))
			if !ok {
				t.Fatal("bind handover repository")
			}

			// Invalid: missing title.
			if _, err := s.Create(HandoverRecord{ID: "x", TenantSlug: "demo", CreatedBy: "a@example.com"}); err == nil {
				t.Fatal("handover without title must error")
			}

			created, err := s.Create(sampleHandover("h1"))
			if err != nil {
				t.Fatalf("create: %v", err)
			}
			if created.ID != "h1" || created.CreatedAt.IsZero() {
				t.Fatalf("created bad: %+v", created)
			}

			// Duplicate (tenant, id) rejected.
			if _, err := s.Create(sampleHandover("h1")); err == nil {
				t.Fatal("duplicate handover must error")
			}

			if got := s.List(); len(got) != 1 || got[0].ID != "h1" {
				t.Fatalf("list = %+v", got)
			}

			if got, ok := s.Get("h1"); !ok || got.Title != "Übergabe h1" {
				t.Fatalf("get = %+v ok=%v", got, ok)
			}
			if _, ok := s.Get("nope"); ok {
				t.Fatal("get unknown must be false")
			}

			// Global token lookup.
			if got, idx, ok := storage.GetByToken("token-h1"); !ok || idx != 0 || got.ID != "h1" {
				t.Fatalf("GetByToken = %+v idx=%d ok=%v", got, idx, ok)
			}
			if _, _, ok := storage.GetByToken("wrong"); ok {
				t.Fatal("GetByToken unknown must be false")
			}

			// Confirm via token: sets ConfirmedAt + ConfirmedBy from name.
			rec, conf, ok, err := storage.ConfirmByToken("token-h1", "Max Muster", "alles gut", now)
			if err != nil || !ok {
				t.Fatalf("confirm: err=%v ok=%v", err, ok)
			}
			if conf.ConfirmedAt.IsZero() || conf.ConfirmedBy != "Max Muster" {
				t.Fatalf("confirmation not set: %+v", conf)
			}
			if len(rec.Confirmations) != 1 || rec.Confirmations[0].ConfirmedAt.IsZero() {
				t.Fatalf("record confirmation not persisted: %+v", rec)
			}

			// Idempotent: confirming again is a no-op that still reports ok, no error,
			// and does not overwrite ConfirmedBy.
			rec2, _, ok2, err2 := storage.ConfirmByToken("token-h1", "Someone Else", "neu", now.Add(time.Hour))
			if err2 != nil || !ok2 {
				t.Fatalf("re-confirm: err=%v ok=%v", err2, ok2)
			}
			if rec2.Confirmations[0].ConfirmedBy != "Max Muster" {
				t.Fatalf("re-confirm must not overwrite: %+v", rec2.Confirmations[0])
			}

			// Unknown token: not found.
			if _, _, ok, _ := storage.ConfirmByToken("no-such", "", "", now); ok {
				t.Fatal("confirm unknown token must be false")
			}

			// File a document id.
			filed, ok, err := s.SetFiledDocument("h1", "doc-99", now)
			if err != nil || !ok || filed.FiledDocumentID != "doc-99" {
				t.Fatalf("SetFiledDocument: err=%v ok=%v filed=%+v", err, ok, filed)
			}
			if got, _ := s.Get("h1"); got.FiledDocumentID != "doc-99" {
				t.Fatalf("filed doc not persisted: %+v", got)
			}
			if _, ok, _ := s.SetFiledDocument("missing", "d", now); ok {
				t.Fatal("SetFiledDocument unknown must be false")
			}
		})
	}
}

func TestSQLHandoverImportFromJSON(t *testing.T) {
	jsonStore, err := NewHandoverStore(filepath.Join(t.TempDir(), "handovers.json"))
	if err != nil {
		t.Fatalf("json store: %v", err)
	}
	jsonRepo, _ := BindHandoverRepository(jsonStore, testTenantRef("demo"))
	if _, err := jsonRepo.Create(sampleHandover("h1")); err != nil {
		t.Fatalf("seed h1: %v", err)
	}
	if _, err := jsonRepo.Create(sampleHandover("h2")); err != nil {
		t.Fatalf("seed h2: %v", err)
	}

	database, lanes := testLanes(t)
	defer database.Close()
	sqlStore := NewSQLHandoverStore(lanes)

	for i := 0; i < 2; i++ {
		if err := sqlStore.ImportHandovers(jsonStore); err != nil {
			t.Fatalf("import %d: %v", i, err)
		}
	}
	sqlRepo, _ := BindHandoverRepository(sqlStore, testTenantRef("demo"))
	if got := sqlRepo.List(); len(got) != 2 {
		t.Fatalf("imported %d, want 2: %+v", len(got), got)
	}
	// Token lookup works on imported data.
	if _, _, ok := sqlStore.GetByToken("token-h2"); !ok {
		t.Fatal("GetByToken on imported handover failed")
	}
}

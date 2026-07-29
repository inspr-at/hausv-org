package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAuditStoreListRetainedDeduplicatesArchiveOverlapAndFiltersTenant(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	at := time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC)
	event := func(offset int, tenant, summary string) AuditEvent {
		return NormalizeAuditEvent(AuditEvent{
			At:         at.Add(time.Duration(offset) * time.Minute),
			TenantSlug: tenant,
			ActorEmail: "admin@example.test",
			ActorRole:  RoleAdmin,
			Action:     AuditActionEnergyImport,
			TargetType: "energy_import",
			TargetID:   summary,
			Summary:    summary,
		})
	}

	first := event(1, "jhw22", "first")
	expired := first
	expired.At = time.Date(2020, 7, 29, 8, 0, 0, 0, time.UTC)
	expired.TargetID = "expired"
	expired.Summary = "expired"
	otherTenant := event(2, "other-house", "other tenant")
	firstBoundary := event(3, "jhw22", "first boundary")
	legitimateDuplicate := event(4, "jhw22", "legitimate duplicate")
	secondBoundary := event(5, "jhw22", "second boundary")
	liveOnly := event(6, "jhw22", "live only")

	// Older deployments may have JSON-array audit files. The next archive is
	// JSONL and starts with the tail retained from this source.
	writeAuditJSONArchive(t, path+".100", []AuditEvent{
		expired,
		first,
		otherTenant,
		firstBoundary,
	})
	writeAuditJSONLArchive(t, path+".200", []AuditEvent{
		firstBoundary,
		legitimateDuplicate,
		legitimateDuplicate, // Repeated inside one source is a real second event.
		secondBoundary,
	})
	writeAuditJSONLArchive(t, path, []AuditEvent{
		secondBoundary,
		liveOnly,
	})

	auditStore, err := NewAuditStore(path)
	if err != nil {
		t.Fatalf("NewAuditStore() error = %v", err)
	}

	got, err := auditStore.ListRetained(" JHW22 ")
	if err != nil {
		t.Fatalf("ListRetained() error = %v", err)
	}
	want := []AuditEvent{
		first,
		firstBoundary,
		legitimateDuplicate,
		legitimateDuplicate,
		secondBoundary,
		liveOnly,
	}
	if len(got) != len(want) {
		t.Fatalf("ListRetained() returned %d events, want %d: %#v", len(got), len(want), got)
	}
	for index := range want {
		assertAuditEventEqual(t, got[index], want[index])
	}

	other, err := auditStore.ListRetained("other-house")
	if err != nil {
		t.Fatalf("ListRetained(other-house) error = %v", err)
	}
	if len(other) != 1 {
		t.Fatalf("ListRetained(other-house) returned %d events, want 1: %#v", len(other), other)
	}
	assertAuditEventEqual(t, other[0], otherTenant)
}

func TestAuditStorePurgeExpiredRetriesAfterLiveRewriteFailure(t *testing.T) {
	oldRetention := auditArchiveRetention
	auditArchiveRetention = 30 * 24 * time.Hour
	defer func() { auditArchiveRetention = oldRetention }()

	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	expired := NormalizeAuditEvent(AuditEvent{
		At: now.Add(-31 * 24 * time.Hour), TenantSlug: "jhw22",
		Action: AuditActionLogin, Summary: "expired",
	})
	recent := NormalizeAuditEvent(AuditEvent{
		At: now.Add(-24 * time.Hour), TenantSlug: "jhw22",
		Action: AuditActionLogin, Summary: "recent",
	})
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	auditStore := &AuditStore{path: path, entries: []AuditEvent{expired, recent}}

	if err := auditStore.PurgeExpired(now); err == nil {
		t.Fatal("expected live rewrite failure")
	}
	if len(auditStore.entries) != 2 {
		t.Fatalf("failed purge advanced in-memory state: %+v", auditStore.entries)
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	writeAuditJSONLArchive(t, path, []AuditEvent{expired, recent})
	if err := auditStore.PurgeExpired(now); err != nil {
		t.Fatalf("retry purge: %v", err)
	}
	if len(auditStore.entries) != 1 || auditStore.entries[0].Summary != "recent" {
		t.Fatalf("retry did not reconcile in-memory state: %+v", auditStore.entries)
	}
	onDisk, err := readAuditEventFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(onDisk) != 1 || onDisk[0].Summary != "recent" {
		t.Fatalf("retry did not reconcile live file: %+v", onDisk)
	}
}

func TestAuditStorePurgeUsesEventTimeBeforeOldArchiveMtime(t *testing.T) {
	oldRetention := auditArchiveRetention
	auditArchiveRetention = 30 * 24 * time.Hour
	defer func() { auditArchiveRetention = oldRetention }()

	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	expired := NormalizeAuditEvent(AuditEvent{
		At: now.Add(-31 * 24 * time.Hour), TenantSlug: "jhw22",
		Action: AuditActionLogin, Summary: "expired",
	})
	recent := NormalizeAuditEvent(AuditEvent{
		At: now.Add(-24 * time.Hour), TenantSlug: "jhw22",
		Action: AuditActionLogin, Summary: "recent",
	})
	legacyWithoutTimestamp := AuditEvent{
		TenantSlug: "jhw22", Action: AuditActionLogin, Summary: "legacy expired by mtime",
	}
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	archive := path + ".100"
	writeAuditJSONLArchive(t, archive, []AuditEvent{expired, recent, legacyWithoutTimestamp})
	oldArchiveTime := now.Add(-60 * 24 * time.Hour)
	if err := os.Chtimes(archive, oldArchiveTime, oldArchiveTime); err != nil {
		t.Fatal(err)
	}

	auditStore := &AuditStore{path: path}
	if err := auditStore.PurgeExpired(now); err != nil {
		t.Fatalf("PurgeExpired() error = %v", err)
	}

	onDisk, err := readAuditEventFile(archive)
	if err != nil {
		t.Fatalf("read retained archive: %v", err)
	}
	if len(onDisk) != 1 || onDisk[0].Summary != "recent" {
		t.Fatalf("old archive mtime removed a recent event or retained expired data: %+v", onDisk)
	}
}

func TestAuditStorePersistsLegacyLiveTimestampFallbackBeforePruning(t *testing.T) {
	oldRetention := auditArchiveRetention
	auditArchiveRetention = 30 * 24 * time.Hour
	defer func() { auditArchiveRetention = oldRetention }()

	now := time.Now().UTC().Truncate(time.Second)
	expired := NormalizeAuditEvent(AuditEvent{
		At: now.Add(-31 * 24 * time.Hour), TenantSlug: "jhw22",
		Action: AuditActionLogin, Summary: "expired",
	})
	recent := NormalizeAuditEvent(AuditEvent{
		At: now.Add(-24 * time.Hour), TenantSlug: "jhw22",
		Action: AuditActionLogin, Summary: "recent",
	})
	legacyWithoutTimestamp := AuditEvent{
		TenantSlug: "jhw22", Action: AuditActionLogin, Summary: "legacy expired by mtime",
	}
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	writeAuditJSONLArchive(t, path, []AuditEvent{expired, recent, legacyWithoutTimestamp})
	oldFileTime := now.Add(-60 * 24 * time.Hour)
	if err := os.Chtimes(path, oldFileTime, oldFileTime); err != nil {
		t.Fatal(err)
	}

	first, err := NewAuditStore(path)
	if err != nil {
		t.Fatalf("first NewAuditStore() error = %v", err)
	}
	got := first.List(AuditFilter{TenantSlug: "jhw22", Limit: 10})
	if len(got) != 1 || got[0].Summary != "recent" {
		t.Fatalf("legacy live retention after first load = %+v", got)
	}
	rawOnDisk, err := readRawAuditEventFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rawOnDisk) != 1 || rawOnDisk[0].At.IsZero() || rawOnDisk[0].Summary != "recent" {
		t.Fatalf("live fallback was not durably reconciled: %+v", rawOnDisk)
	}

	second, err := NewAuditStore(path)
	if err != nil {
		t.Fatalf("second NewAuditStore() error = %v", err)
	}
	reloaded := second.List(AuditFilter{TenantSlug: "jhw22", Limit: 10})
	if len(reloaded) != 1 || reloaded[0].Summary != "recent" {
		t.Fatalf("restart extended or removed retained live data: %+v", reloaded)
	}
}

func writeAuditJSONArchive(t *testing.T, path string, events []AuditEvent) {
	t.Helper()
	raw, err := json.Marshal(events)
	if err != nil {
		t.Fatalf("json.Marshal(%s) error = %v", path, err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("os.WriteFile(%s) error = %v", path, err)
	}
}

func writeAuditJSONLArchive(t *testing.T, path string, events []AuditEvent) {
	t.Helper()
	lines := make([]string, 0, len(events))
	for _, event := range events {
		raw, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("json.Marshal(%s) error = %v", path, err)
		}
		lines = append(lines, string(raw))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%s) error = %v", path, err)
	}
}

func assertAuditEventEqual(t *testing.T, got, want AuditEvent) {
	t.Helper()
	gotJSON, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json.Marshal(got) error = %v", err)
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("json.Marshal(want) error = %v", err)
	}
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("audit event = %s, want %s", gotJSON, wantJSON)
	}
}

package demo

import (
	"context"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

// A reseed must replace the unit inventory, not add to it: the label is the
// identity the portal resolves occupants by, and a stale row from an earlier
// fixture made it read an address that no longer exists (measured on the demo).
func TestResetClearsUnitsBeforeSeeding(t *testing.T) {
	ctx := context.Background()
	database := dbtest.Open(t)
	if _, err := Load(ctx, database, "../../scripts/demo/seed", SeedOptions{Reset: true}); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO units (id, tenant_id, tenant_slug, data) VALUES ('veraltet', (SELECT tenant_id FROM units WHERE tenant_slug='janusbergweg-123' LIMIT 1), 'janusbergweg-123', '{"id":"veraltet","tenant":"janusbergweg-123","label":"Top 1","owner_emails":["alt@example.example"]}')`); err != nil {
		t.Fatalf("insert stale unit: %v", err)
	}
	if _, err := Load(ctx, database, "../../scripts/demo/seed", SeedOptions{Reset: true}); err != nil {
		t.Fatalf("reseed: %v", err)
	}
	var stale int
	if err := database.QueryRowContext(ctx, `SELECT count(*) FROM units WHERE id='veraltet'`).Scan(&stale); err != nil {
		t.Fatalf("count stale: %v", err)
	}
	if stale != 0 {
		t.Fatalf("stale unit survived the reseed")
	}
	var total int
	if err := database.QueryRowContext(ctx, `SELECT count(*) FROM units WHERE tenant_slug='janusbergweg-123'`).Scan(&total); err != nil {
		t.Fatalf("count units: %v", err)
	}
	if total == 0 {
		t.Fatalf("reseed wrote no units")
	}
	var data string
	if err := database.QueryRowContext(ctx, `SELECT data FROM units WHERE tenant_slug='janusbergweg-123' AND data LIKE '%"label":"Top 1"%' LIMIT 1`).Scan(&data); err != nil {
		t.Fatalf("read Top 1: %v", err)
	}
	if want := "alina.eigentuemer@musterstadt.example"; !strings.Contains(data, want) {
		t.Fatalf("Top 1 owner = %s, want %s", data, want)
	}
}

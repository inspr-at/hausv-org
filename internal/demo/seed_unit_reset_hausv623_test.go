package demo

import (
	"context"
	"fmt"
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
	// PostgreSQL enforces per-tenant RLS; the maintenance lane is how the
	// fixture writes across tenants (same switch the seed itself uses).
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if strings.Contains(fmt.Sprintf("%T", database.Driver()), "stdlib") {
		if _, err := tx.ExecContext(ctx, `SET LOCAL hausv.cross_tenant = 'on'`); err != nil {
			t.Fatalf("maintenance lane: %v", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO units (id, tenant_id, tenant_slug, data) VALUES ('veraltet', (SELECT tenant_id FROM units WHERE tenant_slug='janusbergweg-123' LIMIT 1), 'janusbergweg-123', '{"id":"veraltet","tenant":"janusbergweg-123","label":"Top 1","owner_emails":["alt@example.example"]}')`); err != nil {
		t.Fatalf("insert stale unit: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if _, err := Load(ctx, database, "../../scripts/demo/seed", SeedOptions{Reset: true}); err != nil {
		t.Fatalf("reseed: %v", err)
	}
	read := func(query string, args ...any) string {
		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("begin read: %v", err)
		}
		defer tx.Rollback()
		if strings.Contains(fmt.Sprintf("%T", database.Driver()), "stdlib") {
			if _, err := tx.ExecContext(ctx, `SET LOCAL hausv.cross_tenant = 'on'`); err != nil {
				t.Fatalf("maintenance lane: %v", err)
			}
		}
		var value string
		if err := tx.QueryRowContext(ctx, query, args...).Scan(&value); err != nil {
			t.Fatalf("query %q: %v", query, err)
		}
		return value
	}
	if stale := read(`SELECT count(*) FROM units WHERE id='veraltet'`); stale != "0" {
		t.Fatalf("stale unit survived the reseed (count=%s)", stale)
	}
	if total := read(`SELECT count(*) FROM units WHERE tenant_slug='janusbergweg-123'`); total == "0" {
		t.Fatalf("reseed wrote no units")
	}
	data := read(`SELECT data FROM units WHERE tenant_slug='janusbergweg-123' AND data LIKE '%"label":"Top 1"%' LIMIT 1`)
	if want := "alina.eigentuemer@musterstadt.example"; !strings.Contains(data, want) {
		t.Fatalf("Top 1 owner = %s, want %s", data, want)
	}
}

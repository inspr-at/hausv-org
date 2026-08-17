// Package tenantid resolves a tenant's renameable slug to its immutable
// identity.
//
// It is its own package because two persistence layers need the same answer —
// internal/store and internal/energy — and neither should depend on the other to
// get it. Both write tenant_id on every insert now, so a row without an identity
// is a row the query layer can no longer see, and there is exactly one piece of
// code that decides what that identity is.
package tenantid

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
	"github.com/inspr-at/hausv-org/internal/ulid"
)

// Querier is the subset of *sql.DB and *sql.Tx needed here, so a resolution can
// happen inside the same transaction as the write it is for.
type Querier interface {
	QueryRow(query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
}

// Lookup returns the identity of an existing tenant. Absence is not an error:
// onboarding rows legitimately precede the identity.
func Lookup(q Querier, slug string) (string, bool) {
	slug = textutil.Slug(slug)
	if q == nil || slug == "" {
		return "", false
	}
	var id string
	if err := q.QueryRow(`SELECT tenant_id FROM tenant WHERE slug=$1`, slug).Scan(&id); err != nil {
		return "", false
	}
	if !ulid.Valid(id) {
		return "", false
	}
	return id, true
}

// Ensure resolves a slug to its identity, minting one if the slug has none.
//
// Minting here is not a way around the boot-time identity pass. A tenant can
// reach a write path without ever having been in the configured list: a house
// activated through the home portal, a slug dropped from configuration while its
// rows remain, or a JSON import replaying historical data. Every one of those
// must end with a row that carries an identity.
func Ensure(q Querier, slug string) (string, error) {
	slug = textutil.Slug(slug)
	if q == nil {
		return "", fmt.Errorf("tenantid: resolution requires a database")
	}
	if slug == "" {
		return "", fmt.Errorf("tenantid: a slug is required")
	}
	if id, ok := Lookup(q, slug); ok {
		return id, nil
	}
	id, err := ulid.New()
	if err != nil {
		return "", fmt.Errorf("tenantid: mint id for %s: %w", slug, err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	// ON CONFLICT DO NOTHING rather than a bare INSERT, because a concurrent boot
	// or request may have won this race and slug is UNIQUE. Re-reading the
	// winner's row is the resolution rather than a retry loop — but only if the
	// insert never RAISES: this runs inside the boot-time backfill's
	// transaction, and on PostgreSQL a statement that fails poisons the whole
	// transaction, so the re-read that was supposed to recover fails with it and
	// the replica does not start.
	result, err := q.Exec(
		`INSERT INTO tenant(tenant_id,slug,name,created_at,updated_at) VALUES($1,$2,$3,$4,$5)
		 ON CONFLICT(slug) DO NOTHING`,
		id, slug, "", now, now,
	)
	if err != nil {
		if existing, ok := Lookup(q, slug); ok {
			return existing, nil
		}
		return "", fmt.Errorf("tenantid: mint id for %s: %w", slug, err)
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 1 {
		return id, nil
	}
	// Nothing was inserted, so somebody else's row is there. Its id is the
	// answer: minting a second one for the same slug is the thing that must
	// never happen, because every row already written points at the first.
	if existing, ok := Lookup(q, slug); ok {
		return existing, nil
	}
	return "", fmt.Errorf("tenantid: mint id for %s: the insert matched nothing and no tenant row exists", slug)
}

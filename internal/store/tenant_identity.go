package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
	"github.com/inspr-at/hausv-org/internal/ulid"
)

// TenantIdentity is the stable identity of a tenant. The slug is a label that
// may change; the ID never does.
type TenantIdentity struct {
	ID   string
	Slug string
	Name string
}

// TenantRef is the complete tenant identity required at every repository
// boundary. Keeping both halves in one value makes swapping ID and slug a
// compile-time error while the query layer continues to use Slug during the
// tenant_id rollback window.
type TenantRef struct {
	ID   string
	Slug string
}

// Ref drops display-only identity data before entering the repository layer.
func (t TenantIdentity) Ref() TenantRef {
	return TenantRef{ID: t.ID, Slug: t.Slug}
}

// Valid reports whether both identity halves are present and well-formed.
func (t TenantRef) Valid() bool {
	_, ok := validTenantRef(t)
	return ok
}

func validTenantRef(tenant TenantRef) (TenantRef, bool) {
	tenant.Slug = textutil.Slug(tenant.Slug)
	if tenant.Slug == "" || tenant.ID == "" || !ulid.Valid(tenant.ID) {
		return TenantRef{}, false
	}
	return tenant, true
}

// EnsureTenantIdentities mints an ID for every configured slug that does not
// have one yet and returns the full mapping.
//
// Idempotent by construction: an existing slug keeps its ID, which is the entire
// point — if a boot re-minted IDs, every row referencing the old one would be
// orphaned. Renaming is therefore an UPDATE of name only; the slug column is
// what a later rename story will change, and the ID stays put.
func EnsureTenantIdentities(ctx context.Context, database *sql.DB, configured []TenantIdentity) (map[string]TenantIdentity, error) {
	if database == nil {
		return nil, fmt.Errorf("store: tenant identity requires a database")
	}
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("store: begin tenant identity: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, tenant := range configured {
		// Canonicalise here as well as in configuration. Every other resolver —
		// tenantid.Ensure, the backfill, validTenantRef — keys on textutil.Slug,
		// and a `tenant` row whose slug is not canonical is a row none of them
		// can ever find.
		tenant.Slug = textutil.Slug(tenant.Slug)
		if tenant.Slug == "" {
			return nil, fmt.Errorf("store: tenant identity requires a slug")
		}
		// The mint is tenantid.Ensure, not a SELECT-then-INSERT written here.
		// This loop used to do its own, with a bare INSERT and no recovery: two
		// replicas booting against one PostgreSQL both found no row, both
		// inserted, and the loser took a unique violation on tenant.slug that
		// aborted the whole boot. Eight concurrent boots reproduced it one time
		// in eight. tenantid.Ensure inserts with ON CONFLICT DO NOTHING and
		// answers a lost race by re-reading the winner's row, which is the
		// resolution rather than a retry — and the row it re-reads is the one
		// every already-written tenant_id points at.
		id, err := ensureTenantID(tx, tenant.Slug)
		if err != nil {
			return nil, fmt.Errorf("store: mint tenant id for %s: %w", tenant.Slug, err)
		}
		// The display name is the only thing configuration owns. The ID never
		// moves — if a boot re-minted it, every row referencing the old one
		// would be orphaned.
		if _, err := tx.ExecContext(ctx,
			`UPDATE tenant SET name=$1, updated_at=$2 WHERE tenant_id=$3`,
			tenant.Name, now, id); err != nil {
			return nil, fmt.Errorf("store: update tenant %s: %w", tenant.Slug, err)
		}
	}

	rows, err := tx.QueryContext(ctx, `SELECT tenant_id, slug, name FROM tenant`)
	if err != nil {
		return nil, fmt.Errorf("store: read tenant identities: %w", err)
	}
	defer rows.Close()
	out := make(map[string]TenantIdentity)
	for rows.Next() {
		var item TenantIdentity
		if err := rows.Scan(&item.ID, &item.Slug, &item.Name); err != nil {
			return nil, fmt.Errorf("store: scan tenant identity: %w", err)
		}
		if !ulid.Valid(item.ID) {
			// A stored ID that fails the check means the row predates the
			// constraint or was written by something else. Fail loudly: silently
			// tolerating it would let it reach PostgreSQL and be rejected there.
			return nil, fmt.Errorf("store: tenant %s has a malformed id", item.Slug)
		}
		out[item.Slug] = item
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate tenant identities: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: commit tenant identity: %w", err)
	}
	// Only now can the rows be linked: migration 0033 ran before a single
	// identity existed, so its backfill resolved to NULL everywhere. Doing this
	// here rather than leaving it to the caller means the order — mint, link,
	// verify — is not something a boot sequence can get wrong.
	if err := BackfillTenantIDs(ctx, database); err != nil {
		return nil, err
	}
	return out, nil
}

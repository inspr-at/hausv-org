package store

import (
	"github.com/inspr-at/hausv-org/internal/db"
)

// TenantDB is how a store asks for a database handle, and the only way it can.
//
// Its shape is the enforcement. It has For and Unscoped and NOTHING ELSE — no
// Query, no QueryRow, no Exec, no Begin — so a store cannot reach the database
// without first saying, in the call itself, which tenant it is acting for or why
// it is deliberately crossing tenants. That absence is a compile error at every
// site that forgets, which is the only mechanism that scales to the 130-odd
// database calls the store and energy packages make today.
//
// What comes back is a db.Handle: the exact four non-context methods the stores
// already call, which *sql.DB satisfies unchanged. So adopting this is a change
// of where the handle comes from, not of how it is used.
type TenantDB struct {
	scoped *db.Scoped
}

// NewTenantDB wraps an open scoped-access factory.
func NewTenantDB(scoped *db.Scoped) *TenantDB {
	if scoped == nil {
		panic("store: tenant database requires scoped access")
	}
	return &TenantDB{scoped: scoped}
}

// For returns a handle bound to one tenant. An unusable reference is not an
// error here: it produces a scope no row can match, so a caller that lost its
// tenant reads nothing rather than reading everything.
func (t *TenantDB) For(tenant TenantRef) db.Handle {
	ref, ok := validTenantRef(tenant)
	if !ok {
		return t.scoped.For("")
	}
	return t.scoped.For(ref.ID)
}

// Unscoped returns the declared cross-tenant handle. The reason is written down
// at the call site on purpose: every use of this is a decision someone has to be
// able to find later.
func (t *TenantDB) Unscoped(reason string) db.Handle {
	return t.scoped.Unscoped(reason)
}

// Query, QueryRow, Exec and Begin are deliberately ABSENT, and adding them —
// even "just for the migration runner", even as a passthrough — removes the only
// thing that makes the scope non-optional. Two tests fail immediately if they
// come back: one asserts TenantDB does not satisfy db.Handle, the other pins the
// method set to exactly these two accessors.

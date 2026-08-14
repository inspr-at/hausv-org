package server

import (
	"net/http"
	"sync"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// energyLifecycleLock returns an app-local, tenant-scoped barrier between
// ordinary energy work and destructive lifecycle resets. Ordinary handlers
// share the lock; history/profile deletion is exclusive. This ordering
// guarantees that:
//
//   - a write already in flight finishes before deletion and is deleted with it;
//   - a write waiting behind deletion re-runs its handler authorization against
//     the deliberately unclaimed profile; and
//   - a Home Assistant history fetch cannot repopulate the chart cache after the
//     deleting handler has cleared it.
//
// Each app has a bounded, configured tenant set, so retaining one mutex per
// normalized tenant for the app lifetime is intentional.
func (a *app) energyLifecycleLock(tenantSlug string) *sync.RWMutex {
	key := textutil.Slug(tenantSlug)
	value, _ := a.energyLifecycleLocks.LoadOrStore(key, &sync.RWMutex{})
	return value.(*sync.RWMutex)
}

func (a *app) withEnergyLifecycleOperation(next authedHandler) authedHandler {
	return func(w http.ResponseWriter, r *http.Request, ac authCtx) {
		lock := a.energyLifecycleLock(ac.tenant.Slug)
		lock.RLock()
		defer lock.RUnlock()
		next(w, r, ac)
	}
}

// withClaimedEnergyLifecycleOperation adds the claimed-profile invariant used
// by normal energy mutations. The check deliberately happens after acquiring
// the shared barrier: a request queued behind full deletion must observe the
// unclaimed placeholder and may not recreate data with an otherwise valid
// owner/admin capability.
func (a *app) withClaimedEnergyLifecycleOperation(next authedHandler) authedHandler {
	return a.withEnergyLifecycleOperation(func(w http.ResponseWriter, r *http.Request, ac authCtx) {
		profile, exists, err := a.energyStore.Profile(ac.tenant.Slug)
		if err != nil {
			http.Error(w, "Energieprofil konnte nicht geprüft werden.", http.StatusInternalServerError)
			return
		}
		if !exists || energyProfileUnclaimed(profile) {
			http.Error(w, "Das Energieprofil muss zuerst eingerichtet werden.", http.StatusConflict)
			return
		}
		next(w, r, ac)
	})
}

func (a *app) withEnergyLifecycleReset(next authedHandler) authedHandler {
	return func(w http.ResponseWriter, r *http.Request, ac authCtx) {
		lock := a.energyLifecycleLock(ac.tenant.Slug)
		lock.Lock()
		defer lock.Unlock()
		next(w, r, ac)
	}
}

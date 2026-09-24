package server

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func canRefreshIndices(ac authCtx) bool {
	return normalizeRole(ac.role) == roleAdmin && ac.preview == nil && ac.supportView == nil
}

func (a *app) refreshIndices(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !canRefreshIndices(ac) {
		http.Error(w, "VPI-Aktualisierung ist der Administration vorbehalten.", http.StatusForbidden)
		return
	}
	if a.tenantDB == nil {
		http.Error(w, "Indexdaten nicht verfügbar.", http.StatusServiceUnavailable)
		return
	}
	if !a.indexRefreshBusy.CompareAndSwap(false, true) {
		http.Error(w, "Eine VPI-Aktualisierung läuft bereits.", http.StatusConflict)
		return
	}
	defer a.indexRefreshBusy.Store(false)
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	results, err := store.NewIndexReferenceStore(a.tenantDB).Refresh(ctx, a.indexRefreshClient, ac.email, time.Now())
	status := "ok"
	if err != nil {
		status = "partial"
		if len(results) == 0 {
			status = "failed"
		}
		slog.Warn("VPI refresh incomplete", "error", err)
	}
	// Preserve the tenant prefix; callers cannot supply a redirect target.
	path := strings.TrimSuffix(r.URL.Path, "/refresh")
	http.Redirect(w, r, path+"?index_refresh="+status, http.StatusSeeOther)
}

// StartIndexRefreshWorker follows the server's cancellable periodic workers.
// Refresh is opt-in and never runs synchronously on startup. Jitter spreads
// requests across installations; a failed fetch is retried the following day.
func (a *app) StartIndexRefreshWorker() func() {
	if !a.indexRefreshEnabled || a.tenantDB == nil {
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		timer := time.NewTimer(time.Minute + time.Duration(rand.Int64N(int64(29*time.Minute))))
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				if a.indexRefreshBusy.CompareAndSwap(false, true) {
					refreshCtx, stop := context.WithTimeout(ctx, 5*time.Minute)
					results, err := store.NewIndexReferenceStore(a.tenantDB).Refresh(refreshCtx, a.indexRefreshClient, "system:index-refresh", time.Now())
					stop()
					a.indexRefreshBusy.Store(false)
					if err != nil {
						slog.Warn("scheduled VPI refresh incomplete; retry next day", "error", err)
					} else {
						slog.Info("scheduled VPI refresh completed", "sources", len(results))
					}
				}
				timer.Reset(24*time.Hour + time.Duration(rand.Int64N(int64(30*time.Minute))))
			}
		}
	}()
	return func() { cancel(); <-done }
}

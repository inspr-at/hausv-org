package server

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/auth"
	"github.com/inspr-at/hausv-org/internal/authz"
	"github.com/inspr-at/hausv-org/internal/store"
)

// recoverAndLog is the outermost middleware. It recovers panics (so a nil-map or
// index bug in one of the ~71 handlers returns a 500 instead of killing the
// request with a bare stack trace) and emits one structured log line per
// request with a request id, so production is observable at all (HAUSV-141).
func (a *app) recoverAndLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := newRequestID()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		defer func() {
			tenantSlug := ""
			if a != nil {
				tenantSlug = normalizeSlug(a.defaultTenant)
			}
			if resolved, ok := resolvedTenantFromContext(r.Context()); ok {
				tenantSlug = resolved.tenant.Slug
			}
			if rec := recover(); rec != nil {
				// If nothing has been written yet, send a clean 500. If a partial
				// response already went out, we can only log.
				if !sw.wrote {
					sw.Header().Set("Cache-Control", "no-store")
					http.Error(sw, "Internal Server Error", http.StatusInternalServerError)
				}
				slog.Error("panic recovered",
					"request_id", reqID,
					"method", r.Method,
					"route", requestLogRoute(r),
					"tenant", tenantSlug,
					"panic", rec,
					"stack", string(debug.Stack()),
				)
			}
			slog.Info("request",
				"request_id", reqID,
				"method", r.Method,
				"route", requestLogRoute(r),
				"tenant", tenantSlug,
				"status", sw.status,
				"bytes", sw.bytes,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		}()

		next.ServeHTTP(sw, r)
	})
}

// requestLogRoute deliberately records the registered route pattern, not the
// concrete URL path. Public calendar and handover URLs contain bearer tokens in
// their path; object routes also contain identifiers that are unnecessary for
// request-level operations. The matched pattern keeps logs useful for
// filtering without retaining either. Unmatched attacker-controlled paths are
// collapsed to one stable value.
func requestLogRoute(r *http.Request) string {
	if r != nil && r.Pattern != "" {
		return r.Pattern
	}
	return "unmatched"
}

// authCtx carries the authenticated identity for a request. It can be minted
// ONLY by authenticate() below (via the page/action combinators), so a handler
// that takes an authCtx is guaranteed to have passed the session + tenant-match
// guard. This is the invariant HAUSV-135 violated when a copy-pasted
// tenantSlug != tenant.Slug line was forgotten: with the check hoisted into the
// wrapper, a handler can no longer be written without it.
type authCtx struct {
	policy       *authz.Policy
	email        string
	role         string
	realRole     string
	preview      *rolePreviewContext
	tenant       tenantConfig
	tenantRef    store.TenantRef
	repositories requestRepositories
}

func (ac authCtx) actor() authorizationActor {
	return authorizationActor{Person: ac.email, Tenant: ac.tenant.Slug, Role: ac.role, Organisation: rightsOrganisation(ac.tenant), Policy: ac.policy}
}

func (ac authCtx) resource() authorizationResource {
	return authorizationResource{Tenant: ac.tenant.Slug}
}

func (ac authCtx) can(action capability) bool {
	return can(ac.actor(), action, ac.resource())
}

func actorFor(person string, tenantSlug string, role string) authorizationActor {
	return authorizationActor{Person: person, Tenant: tenantSlug, Role: role}
}

func resourceFor(tenantSlug string) authorizationResource {
	return authorizationResource{Tenant: tenantSlug}
}

// authedHandler is a handler that requires an authenticated request. Its authCtx
// parameter is only obtainable from the combinators, so the type itself enforces
// that the route went through the guard.
type authedHandler func(http.ResponseWriter, *http.Request, authCtx)

// authenticate runs the session + tenant-match guard that every /app handler
// used to copy verbatim: resolve the request tenant, look up the session user,
// and require the session's tenant to match. On failure it redirects to "/" (the
// exact behaviour of the old inline guard) and reports false.
func (a *app) authenticate(w http.ResponseWriter, r *http.Request) (authCtx, bool) {
	if a.closedServiceProviderSession(r) {
		http.Error(w, serviceProviderAccessClosedMessage, http.StatusForbidden)
		return authCtx{}, false
	}
	resolved, resolvedOK := resolvedTenantFromContext(r.Context())
	if !resolvedOK {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return authCtx{}, false
	}
	tenant := resolved.tenant
	if session, ok := a.rolePreviewSessionForRequest(r); ok && session.PreviewRole != "" {
		if session.TenantSlug != tenant.Slug {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return authCtx{}, false
		}
		if reason := a.rolePreviewSessionEndReason(session); reason != "" {
			a.terminateRolePreview(w, r, session, reason)
			return authCtx{}, false
		}
		return authCtx{
			policy: a.capabilityPolicy(r.Context(), tenant, true), email: session.Email, role: session.PreviewRole, realRole: session.Role,
			preview: &rolePreviewContext{Role: session.PreviewRole, StartedAt: time.Unix(session.PreviewStartedAt, 0), ExpiresAt: time.Unix(session.PreviewExpiresAt, 0)},
			tenant:  tenant, tenantRef: resolved.tenantRef, repositories: resolved.repositories,
		}, true
	}
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return authCtx{}, false
	}
	return authCtx{policy: a.capabilityPolicy(r.Context(), tenant, false), email: email, role: role, realRole: role, tenant: tenant, tenantRef: resolved.tenantRef, repositories: resolved.repositories}, true
}

func (a *app) rolePreviewSessionForRequest(r *http.Request) (auth.Session, bool) {
	if a == nil || a.sessions == nil || r == nil {
		return auth.Session{}, false
	}
	cookie, err := r.Cookie("weg_session")
	if err != nil {
		return auth.Session{}, false
	}
	return a.sessions.GetSession(cookie.Value)
}

func (a *app) closedServiceProviderSession(r *http.Request) bool {
	if a == nil || a.serviceAccessEnabled || a.sessions == nil {
		return false
	}
	cookie, err := r.Cookie("weg_session")
	if err != nil {
		return false
	}
	email, tenantSlug, _, ok := a.sessions.Get(cookie.Value)
	return ok && a.serviceProviderAccessClosedFor(tenantSlug, email)
}

// page wraps a GET handler with the authenticate guard. Capability and
// service-provider checks stay in the handler body: they are heterogeneous
// (different messages, per-entity logic) and are genuine authorization, not the
// uniform session boilerplate this hoists out.
//
// It is also where a refused page becomes a page. Those capability checks all
// bail out with http.Error, which the browser renders as an unstyled <pre>;
// interceptErrorPage catches that response and re-renders it with the house
// branding, the handler's own wording and a way onward — once, here, instead of
// at every call site. See errorpage.go for the mechanics.
func (a *app) page(h authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ew := a.interceptErrorPage(w, r)
		ac, ok := a.authenticate(ew, r)
		if !ok {
			// A closed service-provider session is refused before there is an
			// identity to build navigation from, so that page stays anonymous.
			ew.finish(authCtx{}, false)
			return
		}
		if module, managed := portalModuleForPath(r.URL.Path); managed && !a.portalModulesFor(ac.tenant.Slug).Enabled(module) {
			http.NotFound(ew, r)
			ew.finish(ac, true)
			return
		}
		h(ew, r, ac)
		ew.finish(ac, true)
	}
}

// publicPage is page() without a session: the same branded error page for the
// GET routes a person can reach while signed out — an expired handover link, a
// mistyped address, a login callback that could not be completed. Machine
// endpoints on the same mux (the calendar feed, map tiles, hero images, the
// health probe, static assets) are deliberately NOT wrapped; their callers want
// a status code, not a page.
func (a *app) publicPage(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ew := a.interceptErrorPage(w, r)
		h(ew, r)
		ew.finishForVisitor()
	}
}

// action wraps a POST handler: the authenticate guard plus the same-origin
// (CSRF) check that every mutating handler used to repeat. Capability checks
// stay in the body for the same reason as page().
func (a *app) action(h authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ac, ok := a.authenticate(w, r)
		if !ok {
			return
		}
		if ac.preview != nil && !strings.HasSuffix(r.URL.Path, "/app/ansicht/ende") {
			http.Error(w, rolePreviewReadOnlyMessage, http.StatusForbidden)
			return
		}
		if module, managed := portalModuleForPath(r.URL.Path); managed && !a.portalModulesFor(ac.tenant.Slug).Enabled(module) {
			http.NotFound(w, r)
			return
		}
		if !sameOriginPost(r) {
			http.Error(w, "Bad request", http.StatusForbidden)
			return
		}
		h(w, r, ac)
	}
}

// authed wraps a GET handler with the authenticate guard AND a single required
// capability, for routes whose only authorization is one uniform capability
// check. Heterogeneous checks — "uploader or admin", permission-based access, or
// a dispatcher that needs different capabilities per branch (see submitBallot) —
// stay in the handler body on purpose (HAUSV-138).
func (a *app) authed(cap capability, h authedHandler) http.HandlerFunc {
	return a.page(func(w http.ResponseWriter, r *http.Request, ac authCtx) {
		if !ac.can(cap) {
			http.Error(w, capabilityForbiddenMessage(cap), http.StatusForbidden)
			return
		}
		h(w, r, ac)
	})
}

// authedAction is authed for POST: the authenticate guard, then the same-origin
// (CSRF) check, then the capability check — the exact order the hoisted handlers
// applied inline.
func (a *app) authedAction(cap capability, h authedHandler) http.HandlerFunc {
	return a.action(func(w http.ResponseWriter, r *http.Request, ac authCtx) {
		if !ac.can(cap) {
			http.Error(w, capabilityForbiddenMessage(cap), http.StatusForbidden)
			return
		}
		h(w, r, ac)
	})
}

// capabilityForbiddenMessage centralises the 403 text so the wrapper and any
// remaining in-handler capability check read identically.
func capabilityForbiddenMessage(cap capability) string {
	if cap == capabilityManageDocuments {
		return "Dieser Bereich ist der Verwaltung vorbehalten."
	}
	return "Dieser Bereich ist Admins vorbehalten."
}

func newRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "req-unknown"
	}
	return hex.EncodeToString(b[:])
}

// statusWriter records the status code and byte count without altering the
// response. It stays transparent for streaming (Flush) and for anything using
// http.ResponseController (Unwrap) — asset serving and file downloads rely on
// the underlying writer's behaviour.
type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
	wrote  bool
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.wrote = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	w.wrote = true
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// Unwrap lets http.ResponseController reach the real writer (SetReadDeadline,
// Flush, etc.) used by streaming responses.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

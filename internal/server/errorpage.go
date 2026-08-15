package server

import (
	"bytes"
	"net/http"
	"strings"
)

// This file holds the seam that turns a failed page request into a page.
//
// Every permission and lookup failure in a GET handler is written with
// http.Error / http.NotFound. That produces a plain-text body, which the
// browser shows as an unstyled <pre>: no branding, no navigation, no way back —
// and, because a <pre> does not wrap, the only horizontal overflow in the
// product (a 964px line inside a 390px viewport).
//
// Fixing that at the ~40 call sites would be noisy and would not stay fixed:
// the next handler would reintroduce it. So the fix sits in the two GET
// combinators in middleware.go — page() and publicPage(). They wrap the
// ResponseWriter, and the wrapper recognises the exact signature http.Error
// leaves behind (Content-Type text/plain, then WriteHeader with a 4xx/5xx),
// swallows that response and re-renders the same status with the same message
// as the branded error page.
//
// The seam is deliberately one-sided. action() — every form POST — is NOT
// wrapped, so mutating routes keep their plain-text contract, and so does
// anything that sets its own content type first (PDF, CSV, ICS, JSON, HTML
// fragments). Sub-resource and programmatic GETs opt out too; see
// wantsHTMLErrorPage.

// plainTextErrorContentType is what http.Error sets on the response before it
// calls WriteHeader. Matching on it is what distinguishes "a handler bailed out
// with http.Error" from "a handler is deliberately writing a 4xx body of its
// own kind".
const plainTextErrorContentType = "text/plain; charset=utf-8"

// errorPageWriter intercepts plain-text error responses on a page route.
//
// It stays completely transparent for everything else: the first write that is
// not a plain-text 4xx/5xx disarms the interception permanently, so a handler
// that streams a PDF or renders HTML never pays for this.
type errorPageWriter struct {
	http.ResponseWriter
	app      *app
	req      *http.Request
	armed    bool
	captured bool
	status   int
	message  strings.Builder
}

// interceptErrorPage wraps w for the duration of one GET page request.
func (a *app) interceptErrorPage(w http.ResponseWriter, r *http.Request) *errorPageWriter {
	return &errorPageWriter{
		ResponseWriter: w,
		app:            a,
		req:            r,
		armed:          a != nil && a.templates != nil && wantsHTMLErrorPage(r),
	}
}

func (w *errorPageWriter) WriteHeader(code int) {
	if w.armed && !w.captured && code >= 400 && w.Header().Get("Content-Type") == plainTextErrorContentType {
		w.captured = true
		w.status = code
		return
	}
	w.armed = false
	w.ResponseWriter.WriteHeader(code)
}

func (w *errorPageWriter) Write(b []byte) (int, error) {
	if w.captured {
		// http.Error writes the message and nothing else; keep it verbatim so the
		// page can show the handler's own wording.
		w.message.Write(b)
		return len(b), nil
	}
	w.armed = false
	return w.ResponseWriter.Write(b)
}

// Unwrap keeps http.ResponseController working for handlers that stream
// (attachment and document downloads reach for the real writer).
func (w *errorPageWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *errorPageWriter) Flush() {
	if w.captured {
		return
	}
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// finish renders the branded page for a swallowed error. It is a no-op when the
// handler succeeded, which is the overwhelmingly common case.
func (w *errorPageWriter) finish(ac authCtx, authenticated bool) {
	if w == nil || !w.captured {
		return
	}
	w.app.writeErrorPage(w.ResponseWriter, w.req, ac, authenticated, w.status, w.message.String())
}

// finishForVisitor is finish for a route that does not require a session. A
// mistyped URL is just as likely to be typed by someone who is signed in, and
// they should get their own navigation rather than a link back to the sign-in
// screen — so the identity is resolved here, but only if there is a page to
// render.
func (w *errorPageWriter) finishForVisitor() {
	if w == nil || !w.captured {
		return
	}
	ac, ok := w.app.sessionActor(w.req)
	w.finish(ac, ok)
}

// sessionActor resolves the signed-in identity without enforcing it. It mirrors
// authenticate()'s tenant-match rule so a session from another house never
// shapes this house's page, and it refuses a closed service-provider session,
// whose owner must not be offered app navigation.
func (a *app) sessionActor(r *http.Request) (authCtx, bool) {
	if a == nil || r == nil || a.closedServiceProviderSession(r) {
		return authCtx{}, false
	}
	tenant := a.tenantForRequest(r)
	email, role, tenantSlug, ok := a.currentUser(r)
	if !ok || tenantSlug != tenant.Slug {
		return authCtx{}, false
	}
	return authCtx{email: email, role: role, tenant: tenant}, true
}

// wantsHTMLErrorPage reports whether this request is a person navigating to a
// page — the only case where an HTML error page is the right answer.
//
// Fetch Metadata is the precise signal: a browser sends Sec-Fetch-Dest:
// document only for a top-level navigation, and something else for an <img>
// source, a fetch() or an XHR. Clients that send no Fetch Metadata at all
// (older browsers, curl, the Go test suite) fall back to the Accept header, and
// a client that asks for JSON keeps the plain body it asked for.
func wantsHTMLErrorPage(r *http.Request) bool {
	if r == nil || r.Method != http.MethodGet {
		return false
	}
	if r.URL != nil && r.URL.Query().Get("format") == "json" {
		return false
	}
	switch r.Header.Get("Sec-Fetch-Dest") {
	case "", "document", "iframe", "frame":
	default:
		return false
	}
	accept := r.Header.Get("Accept")
	if accept == "" {
		return true
	}
	return strings.Contains(accept, "text/html") || strings.Contains(accept, "*/*")
}

// stockErrorMessages are the strings the standard library and the transport
// layer put in front of users today. They are English, technical, or both, and
// carry no information the person can act on, so the page replaces them with
// German copy for the status. Anything a handler wrote on purpose — "Dieser
// Bereich ist der Verwaltung vorbehalten.", "Dieses Dokument ist für diesen
// Zugang nicht freigegeben." — survives verbatim, because that specificity is
// the whole value of the message.
var stockErrorMessages = map[string]bool{
	"404 page not found":       true,
	"Bad request":              true,
	"Forbidden":                true,
	"Internal Server Error":    true,
	"Method Not Allowed":       true,
	"Unauthorized":             true,
	"Unknown tenant":           true,
	"Could not create session": true,
}

// errorPageCopy is the calm German wording for a status, before the handler's
// own message (if it has one) is layered on top.
func errorPageCopy(status int, raw string) (headline string, message string, advice string) {
	switch status {
	case http.StatusBadRequest:
		headline = "Anfrage nicht verstanden"
		message = "Diese Anfrage konnte nicht verarbeitet werden."
		advice = "Bitte laden Sie die Seite neu und versuchen Sie es noch einmal."
	case http.StatusUnauthorized:
		headline = "Anmeldung erforderlich"
		message = "Diese Seite lässt sich nur mit einer gültigen Anmeldung öffnen."
		advice = "Bitte melden Sie sich erneut an."
	case http.StatusForbidden:
		headline = "Kein Zugriff"
		message = "Dieser Bereich ist für diesen Zugang nicht freigegeben."
		// The advice must read correctly after any handler message, so it carries
		// no pronoun back to one.
		advice = "Wenden Sie sich bitte an die Hausverwaltung, wenn Sie Zugang benötigen."
	case http.StatusNotFound:
		headline = "Nicht gefunden"
		message = "Unter dieser Adresse gibt es nichts zu sehen."
		advice = "Möglicherweise ist der Link unvollständig oder der Eintrag wurde entfernt."
	case http.StatusMethodNotAllowed:
		headline = "Nicht möglich"
		message = "Diese Seite lässt sich so nicht öffnen."
		advice = "Bitte rufen Sie den Bereich über die Navigation auf."
	case http.StatusTooManyRequests:
		headline = "Zu viele Versuche"
		message = "Es wurden in kurzer Zeit zu viele Anfragen gestellt."
		advice = "Bitte warten Sie einen Moment und versuchen Sie es dann erneut."
	case http.StatusServiceUnavailable:
		headline = "Gerade nicht erreichbar"
		message = "Dieser Dienst ist im Moment nicht verfügbar."
		advice = "Bitte versuchen Sie es in einigen Minuten noch einmal."
	case http.StatusInternalServerError:
		headline = "Etwas ist schiefgelaufen"
		message = "Der Vorgang konnte nicht abgeschlossen werden."
		advice = "Bitte versuchen Sie es in einigen Minuten noch einmal. Die Störung wurde protokolliert."
	default:
		headline = "Das hat nicht geklappt"
		message = "Diese Seite konnte nicht geladen werden."
		advice = "Bitte versuchen Sie es später noch einmal."
	}
	if specific := strings.TrimSpace(raw); specific != "" && !stockErrorMessages[specific] {
		message = specific
	}
	return headline, message, advice
}

// errorPageLink is one onward route offered on the error page.
type errorPageLink struct {
	URL   string
	Label string
	Hint  string
}

// errorPageOnwardLinks lists only areas this person can actually open. It is
// built from the same capability flags the sidebar uses, so the page never
// advertises something they lack — naming a locked area would both frustrate
// and leak.
func errorPageOnwardLinks(role string, canSeeParking bool) []errorPageLink {
	links := []errorPageLink{}
	if roleCanUseResidentAreas(role) {
		links = append(links,
			errorPageLink{URL: "/app/announcements", Label: "Aushang", Hint: "Mitteilungen des Hauses"},
			errorPageLink{URL: "/app/events", Label: "Termine", Hint: "Was als Nächstes ansteht"},
			errorPageLink{URL: "/app/dokumente", Label: "Dokumente", Hint: "Freigegebene Unterlagen"},
		)
	}
	if roleHasCapability(role, capabilityManageIssues) {
		links = append(links, errorPageLink{URL: "/app/anliegen/board", Label: "Anliegen", Hint: "Meldungen bearbeiten"})
	} else {
		links = append(links, errorPageLink{URL: "/app/anliegen", Label: "Anliegen", Hint: "Melden und nachverfolgen"})
	}
	if roleCanUseResidentAreas(role) {
		links = append(links,
			errorPageLink{URL: "/app/abstimmungen", Label: "Abstimmungen", Hint: "Beschlüsse und laufende Entscheidungen"},
			errorPageLink{URL: "/app/kontakte", Label: "Kontakte", Hint: "Verwaltung, Beirat und Dienstleister"},
		)
	}
	if canSeeParking {
		links = append(links, errorPageLink{URL: "/app/parking", Label: "Parkplatznutzung", Hint: "Verbrauch und Abrechnung"})
	}
	if roleCanUseResidentAreas(role) {
		links = append(links, errorPageLink{URL: "/app/settings", Label: "Einstellungen", Hint: "Profil und Benachrichtigungen"})
	}
	return links
}

// bufferedPage collects a rendered page so the error page can reuse render()
// — which owns the shared page context — and still send its own status code.
// render writes a 200 implicitly; buffering lets the status stay 403.
type bufferedPage struct {
	head http.Header
	body bytes.Buffer
}

func (b *bufferedPage) Header() http.Header {
	if b.head == nil {
		b.head = http.Header{}
	}
	return b.head
}

func (b *bufferedPage) Write(p []byte) (int, error) { return b.body.Write(p) }

func (b *bufferedPage) WriteHeader(int) {}

// writeErrorPage renders the branded page onto the real writer, keeping the
// status the handler chose. w is the unwrapped writer, so nothing here is
// intercepted again.
func (a *app) writeErrorPage(w http.ResponseWriter, r *http.Request, ac authCtx, authenticated bool, status int, raw string) {
	headline, message, advice := errorPageCopy(status, raw)
	tenant := ac.tenant
	if !authenticated {
		// Branding for a visitor without a session. The host decides the house;
		// the page shows only its name and address, never anything private.
		tenant = a.tenantForRequest(r)
	}

	data := map[string]any{
		"Title":         headline + " · " + houseDisplayName(tenant),
		"Tenant":        tenant,
		"ErrorStatus":   status,
		"ErrorHeadline": headline,
		"ErrorMessage":  message,
		"ErrorAdvice":   advice,
	}
	if authenticated {
		profile := a.profileForTenant(ac.email, ac.tenant.Slug)
		canSeeParking := ac.can(capabilityPlatformAdmin) || profile.HasPermission(permissionParking)
		links := errorPageOnwardLinks(ac.role, canSeeParking)
		data["ErrorLinks"] = links
		data["HasErrorLinks"] = len(links) > 0
		if roleCanUseResidentAreas(ac.role) {
			data["ErrorPrimaryURL"] = "/app"
			data["ErrorPrimaryLabel"] = "Zum Hausüberblick"
		} else {
			// Service-provider accounts have no Hausüberblick — /app redirects them
			// to Anliegen, so send them straight there.
			data["ErrorPrimaryURL"] = "/app/anliegen"
			data["ErrorPrimaryLabel"] = "Zu den Anliegen"
		}
	} else {
		data["HasErrorLinks"] = false
		data["ErrorPrimaryURL"] = "/"
		data["ErrorPrimaryLabel"] = "Zur Anmeldung"
	}

	buffered := &bufferedPage{}
	a.executeTemplate(buffered, "errorPage", data)
	if buffered.body.Len() == 0 {
		// The template is the fallback's fallback; never leave the request without
		// a body just because rendering failed.
		http.Error(w, message, status)
		return
	}

	header := w.Header()
	header.Set("Content-Type", "text/html; charset=utf-8")
	header.Set("Cache-Control", "no-store")
	header.Del("Content-Length")
	w.WriteHeader(status)
	_, _ = w.Write(buffered.body.Bytes())
}

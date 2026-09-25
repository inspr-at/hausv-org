package server

import (
	"net/url"
	"path"
	"regexp"
	"strings"
)

// OrganisationHousePath is the only builder for a house destination rendered
// inside an organisation page. It never prefixes a tenant slug: the HTML
// prefixer would add the session tenant and produce a double prefix such as
// /janusbergweg-123/musterstrasse-12/app/.... A foreign house is opened by
// posting the returned path as next on /app/context. foreign is true when the
// destination house is not the session house.
func OrganisationHousePath(sessionSlug, houseSlug, appPath string) (string, bool) {
	safe, ok := safePortalReturnPath(appPath)
	if !ok {
		safe = "/app"
	}
	return safe, normalizeSlug(sessionSlug) != normalizeSlug(houseSlug)
}

var (
	portalReturnToken    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,80}$`)
	portalReturnFragment = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,80}$`)
	portalMenuPaths      = map[string]struct{}{
		"/app": {}, "/app/energie": {}, "/app/announcements": {}, "/app/events": {},
		"/app/kontakte": {}, "/app/dokumente": {}, "/app/anliegen": {}, "/app/anliegen/board": {},
		"/app/abstimmungen": {}, "/app/parking": {}, "/app/uebergaben": {},
		"/app/settings/users": {}, "/app/audit": {}, "/app/settings": {}, "/app/hilfe": {},
	}
)

// portalContextNext keeps a context switch inside the destination house.
// Anything that is not a relative same-house path falls back to the house home.
func portalContextNext(path string) string {
	if safe, ok := safePortalReturnPath(path); ok {
		return safe
	}
	return "/app"
}

func safePortalReturnPath(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, "\\\t\r\n") || strings.Contains(raw, "://") || strings.HasPrefix(raw, "//") {
		return "", false
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.User != nil {
		return "", false
	}
	cleanPath := parsed.Path
	unescaped, err := url.PathUnescape(cleanPath)
	if err != nil || unescaped != cleanPath || path.Clean(cleanPath) != cleanPath {
		return "", false
	}
	if cleanPath != "/app" && !strings.HasPrefix(cleanPath, "/app/") {
		return "", false
	}
	if !portalReturnPathAllowed(cleanPath) {
		return "", false
	}
	query := ""
	if parsed.RawQuery != "" {
		encoded, ok := portalReturnQuery(cleanPath, parsed.Query())
		if !ok {
			return "", false
		}
		query = encoded
	}
	if parsed.Fragment != "" && !portalReturnFragment.MatchString(parsed.Fragment) {
		return "", false
	}
	out := cleanPath
	if query != "" {
		out += "?" + query
	}
	if parsed.Fragment != "" {
		out += "#" + parsed.Fragment
	}
	return out, true
}

func portalReturnPathAllowed(cleanPath string) bool {
	if _, ok := portalMenuPaths[cleanPath]; ok {
		return true
	}
	switch cleanPath {
	case "/app/settings/valorisation", "/app/settings/annual-statement":
		return true
	}
	if rest, ok := strings.CutPrefix(cleanPath, "/app/settings/building/units/"); ok {
		id, lease, found := strings.Cut(rest, "/")
		return found && lease == "lease" && portalReturnToken.MatchString(id)
	}
	if id, ok := strings.CutPrefix(cleanPath, "/app/verwaltung/posteingang/"); ok {
		return !strings.Contains(id, "/") && portalReturnToken.MatchString(id)
	}
	return false
}

func portalReturnQuery(cleanPath string, values url.Values) (string, bool) {
	if cleanPath != "/app/settings/annual-statement" {
		return "", false
	}
	year := values.Get("year")
	if len(year) != 4 || strings.Trim(year, "0123456789") != "" {
		return "", false
	}
	allowed := url.Values{"year": {year}}
	if run := values.Get("run"); run != "" {
		if !portalReturnToken.MatchString(run) {
			return "", false
		}
		allowed.Set("run", run)
	}
	for key := range values {
		if key != "year" && key != "run" {
			return "", false
		}
	}
	return allowed.Encode(), true
}

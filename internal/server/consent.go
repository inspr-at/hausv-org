package server

import (
	"encoding/json"
	"html/template"
	"net/url"
	"strings"
)

// consentManifest is the consent-gate manifest for the public pages of this
// instance (HAUSV-753). The gate itself is the vendored inspr-modules
// primitive (internal/web/assets/consent-gate.js); this manifest declares the
// controller, the host scope, the one optional service — the Google Ads tag —
// with the storage it creates and the Consent Mode keys it may ever set, and
// the German texts. It is only built when GOOGLE_ADS_TAG_ID is configured.
func (a *app) consentManifest() template.JS {
	if a == nil || a.googleAdsTagID == "" {
		return ""
	}
	scope := "localhost"
	if u, err := url.Parse(a.baseURL); err == nil && u.Hostname() != "" {
		scope = strings.ToLower(u.Hostname())
	}
	destination := map[string]any{
		"type":                 "gtag",
		"tagId":                a.googleAdsTagID,
		"consent":              []string{"ad_storage", "ad_user_data"},
		"linkerAcceptIncoming": true,
	}
	if a.googleAdsLeadConversion != "" {
		destination["conversion"] = map[string]any{"sendTo": a.googleAdsLeadConversion, "value": 1.0, "currency": "EUR"}
	}
	manifest := map[string]any{
		"version":        1,
		"revision":       1,
		"textVersion":    1,
		"controller":     platformOperatorName(),
		"scope":          scope,
		"cookieName":     "hausv_consent",
		"permissionDays": 180,
		"refusalMonths":  6,
		"language":       "de",
		"privacyUrl":     "/datenschutz",
		"categories":     []map[string]any{{"id": "necessary", "required": true}, {"id": "marketing"}},
		"services": []map[string]any{{
			"id":       "google-ads",
			"category": "marketing",
			"provider": "Google Ireland Limited",
			"purposes": []string{"Conversion-Messung", "Zuordnung der Anzeige zur Anfrage"},
			// Google may set its cookies host-only or on the domain; both
			// forms are declared so withdrawal clears both (scope = this host).
			"storage": []map[string]any{
				{"kind": "cookie", "name": "_gcl_*", "days": 90},
				{"kind": "cookie", "name": "_gcl_*", "days": 90, "domain": scope, "path": "/"},
				{"kind": "local", "name": "_gcl_ls"},
			},
			"destination": destination,
		}},
		"text": map[string]any{"de": map[string]any{
			"bar": map[string]string{
				"text":     "Wir verwenden notwendige Cookies. Mit Ihrer Einwilligung nutzen wir zusätzlich Google Ads, um zu messen, welche Anzeige zu einer Anfrage geführt hat.",
				"link":     "Datenschutz",
				"accept":   "Akzeptieren",
				"reject":   "Ablehnen",
				"settings": "Einstellungen",
			},
			"sheet": map[string]string{
				"title":  "Privatsphäre",
				"intro":  "Notwendige Cookies halten Sie angemeldet und schützen Formulare. Alles Weitere bleibt aus, bis Sie es hier einschalten.",
				"save":   "Speichern",
				"cancel": "Abbrechen",
				"link":   "Details und Widerruf: Datenschutz",
			},
			"embed":   map[string]string{"loadOnce": "Diesmal laden", "always": "Immer erlauben", "open": "oder öffnen auf"},
			"control": "Privatsphäre",
			"categories": map[string]map[string]string{
				"necessary": {"label": "Notwendig", "description": "Anmeldung, Formulare, diese Einstellung. Immer aktiv."},
				"marketing": {"label": "Marketing (Google Ads)", "description": "Google Ireland Ltd. misst, ob eine Anzeige zu einer Anfrage geführt hat; Daten können in die USA übertragen werden. Cookies bis zu 90 Tage, Ihre Wahl 6 Monate."},
			},
		}},
	}
	raw, err := json.Marshal(manifest) // html-safe: < > & are escaped
	if err != nil {
		return ""
	}
	return template.JS(raw)
}

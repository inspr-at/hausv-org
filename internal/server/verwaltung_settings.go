package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/ai"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

var newSettingsAISuggester = ai.NewFromEnv

type aiSettingsConfig struct {
	Provider   string
	Label      string
	BaseURL    string
	Host       string
	Model      string
	Timeout    string
	Configured bool
}

func effectiveAIConfig(getenv func(string) string, settings store.OrgSettings) aiSettingsConfig {
	value := func(key string) string {
		if getenv == nil {
			return ""
		}
		return strings.TrimSpace(getenv(key))
	}
	baseURL := value("AI_BASE_URL")
	if settings.AIBaseURL != "" {
		baseURL = strings.TrimSpace(settings.AIBaseURL)
	}
	model := value("AI_MODEL")
	if settings.AIModel != "" {
		model = strings.TrimSpace(settings.AIModel)
	}
	provider := strings.TrimSpace(settings.AIProvider)
	parsed, _ := url.Parse(baseURL)
	if provider == "" {
		provider = "environment"
		if parsed != nil && strings.Contains(strings.ToLower(parsed.Hostname()), "openrouter") {
			provider = "cloud"
		}
	}
	label := value("AI_PROVIDER_LABEL")
	if settings.AIProvider != "" || label == "" {
		switch provider {
		case "cloud":
			label = "Cloud (OpenRouter)"
		case "local":
			label = "Lokal (OpenAI-kompatibel)"
		default:
			label = "Umgebung"
		}
	}
	host := ""
	if parsed != nil && parsed.Host != "" {
		host = parsed.Host
	}
	timeout := value("AI_TIMEOUT")
	if timeout == "" {
		timeout = "45s"
	}
	return aiSettingsConfig{Provider: provider, Label: label, BaseURL: baseURL, Host: host, Model: model, Timeout: timeout, Configured: baseURL != "" && model != ""}
}

func aiSettingsGetenv(getenv func(string) string, settings store.OrgSettings) func(string) string {
	return func(key string) string {
		switch key {
		case "AI_BASE_URL":
			if settings.AIBaseURL != "" {
				return strings.TrimSpace(settings.AIBaseURL)
			}
		case "AI_MODEL":
			if settings.AIModel != "" {
				return strings.TrimSpace(settings.AIModel)
			}
		}
		if getenv == nil {
			return ""
		}
		return getenv(key)
	}
}

func validateAIBaseURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("Die KI-Basis-URL muss eine vollständige HTTP- oder HTTPS-Adresse sein.")
	}
	if parsed.User != nil {
		return fmt.Errorf("Die KI-Basis-URL darf keine Zugangsdaten enthalten.")
	}
	return nil
}

func (a *app) verwaltungSettingsPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.isOrganisationAdmin(&ac) {
		http.Error(w, "Dieser Bereich ist Organisationsadministratoren vorbehalten.", http.StatusForbidden)
		return
	}
	orgKey, ok := a.inboxOrganisationKey(&ac)
	if !ok || a.orgSettings == nil {
		http.Error(w, "Einstellungen nicht verfügbar.", http.StatusServiceUnavailable)
		return
	}
	settings, err := a.orgSettings(orgKey).Get(r.Context())
	if err != nil {
		a.inboxError(w, err)
		return
	}
	a.renderVerwaltungSettings(w, r, ac, settings, r.URL.Query().Get("flash"), "", false)
}

func (a *app) verwaltungSettingsAction(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.isOrganisationAdmin(&ac) {
		http.Error(w, "Dieser Bereich ist Organisationsadministratoren vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	orgKey, ok := a.inboxOrganisationKey(&ac)
	if !ok || a.orgSettings == nil {
		http.Error(w, "Einstellungen nicht verfügbar.", http.StatusServiceUnavailable)
		return
	}
	repo := a.orgSettings(orgKey)
	before, err := repo.Get(r.Context())
	if err != nil {
		a.inboxError(w, err)
		return
	}
	after := before
	after.TrustLevels = map[string]string{}
	for _, category := range store.IntakeCategories() {
		level := r.FormValue("trust_" + category.Key)
		if level != "manual" && level != "propose" && level != "auto" {
			level = "propose"
		}
		after.TrustLevels[category.Key] = level
	}
	threshold, err := strconv.Atoi(r.FormValue("threshold"))
	if err != nil || threshold < 50 || threshold > 100 {
		http.Error(w, "Schwellwert muss zwischen 50 und 100 liegen.", http.StatusBadRequest)
		return
	}
	after.AutoThreshold = float64(threshold) / 100
	after.AutoEnabled = r.FormValue("auto_enabled") == "1"
	if r.Form.Has("ai_provider") || r.Form.Has("ai_base_url") || r.Form.Has("ai_model") {
		provider := strings.TrimSpace(r.FormValue("ai_provider"))
		if provider != "environment" && provider != "cloud" && provider != "local" {
			http.Error(w, "Bitte einen KI-Anbieter auswählen.", http.StatusBadRequest)
			return
		}
		baseURL := strings.TrimSpace(r.FormValue("ai_base_url"))
		if err := validateAIBaseURL(baseURL); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if provider == "environment" {
			after.AIProvider = ""
			after.AIBaseURL = ""
			after.AIModel = ""
		} else {
			after.AIProvider = provider
			after.AIBaseURL = baseURL
			after.AIModel = strings.TrimSpace(r.FormValue("ai_model"))
		}
	}
	if err := repo.Save(r.Context(), after); err != nil {
		a.inboxError(w, err)
		return
	}
	summary := settingsDiff(before, after)
	if before.AIProvider != after.AIProvider || before.AIBaseURL != after.AIBaseURL || before.AIModel != after.AIModel {
		summary += "; KI-Anbieter aktualisiert"
	}
	for _, tenant := range a.managedTenants(&ac) {
		a.recordAudit(store.AuditEvent{TenantSlug: tenant.Ref.Slug, ActorEmail: ac.email, ActorRole: tenant.Role, Action: store.AuditActionVerwaltungSettings, TargetType: "organisation", TargetID: orgKey, Summary: "Verwaltungseinstellungen gespeichert", Details: map[string]string{"diff": strings.TrimPrefix(summary, "; ")}})
	}
	http.Redirect(w, r, "/app/verwaltung/einstellungen?flash="+url.QueryEscape("Einstellungen gespeichert"), http.StatusSeeOther)
}

func (a *app) renderVerwaltungSettings(w http.ResponseWriter, r *http.Request, ac authCtx, settings store.OrgSettings, flash, aiResult string, aiOK bool) {
	total := settings.Counters.Approved + settings.Counters.Edited
	share := 0
	if total > 0 {
		share = settings.Counters.Approved * 100 / total
	}
	config := effectiveAIConfig(os.Getenv, settings)
	data := web.VerwaltungSettingsData{
		Threshold: int(settings.AutoThreshold*100 + 0.5), AutoEnabled: settings.AutoEnabled,
		ProviderLabel: config.Label, AIHost: config.Host, AIModel: config.Model, AITimeout: config.Timeout,
		AIProvider: config.Provider, AIConfigured: config.Configured, AIBaseURLOverride: settings.AIBaseURL, AIModelOverride: settings.AIModel,
		Approved: settings.Counters.Approved, Edited: settings.Counters.Edited, Rejected: settings.Counters.Rejected,
		Auto: settings.Counters.Auto, UnchangedShare: share, Flash: flash, AITestResult: aiResult, AITestOK: aiOK,
		DemoResetAvailable: a.demoReset != nil,
	}
	for _, category := range store.IntakeCategories() {
		data.Categories = append(data.Categories, web.VerwaltungSettingsCategory{Key: category.Key, Label: category.Label, Level: settings.TrustLevels[category.Key]})
	}
	var rendered bytes.Buffer
	if err := web.VerwaltungSettingsPage(a.verwaltungShell(&ac, "settings"), data).Render(r.Context(), &rendered); err != nil {
		a.inboxError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), ac.tenant.Slug))
}

func (a *app) verwaltungAITestAction(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.isOrganisationAdmin(&ac) {
		http.Error(w, "Dieser Bereich ist Organisationsadministratoren vorbehalten.", http.StatusForbidden)
		return
	}
	orgKey, ok := a.inboxOrganisationKey(&ac)
	if !ok || a.orgSettings == nil {
		http.Error(w, "Einstellungen nicht verfügbar.", http.StatusServiceUnavailable)
		return
	}
	settings, err := a.orgSettings(orgKey).Get(r.Context())
	if err != nil {
		a.inboxError(w, err)
		return
	}
	getenv := aiSettingsGetenv(os.Getenv, settings)
	suggester, err := newSettingsAISuggester(getenv)
	if err == nil && suggester == nil {
		err = ai.ErrUnavailable
	}
	started := time.Now()
	var suggestion ai.TriageSuggestion
	if err == nil {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		categories := make([]ai.CategoryRule, 0, len(store.IntakeCategories()))
		for _, category := range store.IntakeCategories() {
			categories = append(categories, ai.CategoryRule{Key: category.Key, Label: category.Label, Description: category.Label, DefaultPriority: store.IssuePriorityNorm})
		}
		suggestion, err = suggester.Suggest(ctx, ai.TriageInput{Organisation: orgKey, Source: "settings-test", Subject: "Verbindungstest", Body: "Im Stiegenhaus ist eine Lampe ausgefallen.", ReceivedAt: time.Now(), Categories: categories})
	}
	elapsed := time.Since(started).Milliseconds()
	if err != nil {
		message := err.Error()
		if apiKey := strings.TrimSpace(os.Getenv("AI_API_KEY")); apiKey != "" {
			message = strings.ReplaceAll(message, apiKey, "[geschützt]")
		}
		a.renderVerwaltungSettings(w, r, ac, settings, "", "Verbindung fehlgeschlagen · "+message, false)
		return
	}
	model := strings.TrimSpace(suggestion.Model)
	if model == "" {
		model = effectiveAIConfig(os.Getenv, settings).Model
	}
	a.renderVerwaltungSettings(w, r, ac, settings, "", fmt.Sprintf("Verbindung ok · %s · %d ms", model, elapsed), true)
}

func (a *app) verwaltungDemoResetPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if a.demoReset == nil {
		http.NotFound(w, r)
		return
	}
	if !a.isOrganisationAdmin(&ac) {
		http.Error(w, "Dieser Bereich ist Organisationsadministratoren vorbehalten.", http.StatusForbidden)
		return
	}
	a.renderVerwaltungDemoReset(w, r, ac, web.VerwaltungDemoResetData{Step: 1})
}

func (a *app) verwaltungDemoResetAction(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if a.demoReset == nil {
		http.NotFound(w, r)
		return
	}
	if !a.isOrganisationAdmin(&ac) {
		http.Error(w, "Dieser Bereich ist Organisationsadministratoren vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	switch r.FormValue("step") {
	case "1":
		if r.FormValue("confirm_word") != "ZURÜCKSETZEN" || r.FormValue("understood") != "1" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			a.renderVerwaltungDemoReset(w, r, ac, web.VerwaltungDemoResetData{Step: 1, Error: "Bitte ZURÜCKSETZEN eingeben und die Bestätigung aktivieren."})
			return
		}
		a.renderVerwaltungDemoReset(w, r, ac, web.VerwaltungDemoResetData{Step: 2})
	case "2":
		anchor := demoResetAnchor(time.Now())
		started := time.Now()
		var stats bytes.Buffer
		result, err := a.demoReset(r.Context(), anchor, &stats)
		if err != nil {
			logError("demo reset failed", err)
			http.Error(w, "Die Demo konnte nicht zurückgesetzt werden.", http.StatusInternalServerError)
			return
		}
		orgKey, _ := a.inboxOrganisationKey(&ac)
		for _, tenant := range a.managedTenants(&ac) {
			a.recordAudit(store.AuditEvent{TenantSlug: tenant.Ref.Slug, ActorEmail: ac.email, ActorRole: tenant.Role, Action: store.AuditActionDemoReset, TargetType: "organisation", TargetID: orgKey, Summary: "Demo zurückgesetzt"})
		}
		data := web.VerwaltungDemoResetData{Step: 3, Anchor: anchor.Format("02.01.2006"), Duration: time.Since(started).Round(time.Millisecond).String()}
		statusKeys := make([]string, 0, len(result.Statuses))
		for key := range result.Statuses {
			statusKeys = append(statusKeys, string(key))
		}
		sort.Strings(statusKeys)
		for _, key := range statusKeys {
			data.Statuses = append(data.Statuses, web.VerwaltungDemoResetCount{Label: key, Count: result.Statuses[store.IntakeStatus(key)]})
		}
		categoryLabels := map[string]string{}
		for _, category := range store.IntakeCategories() {
			categoryLabels[category.Key] = category.Label
		}
		categoryKeys := make([]string, 0, len(result.Categories))
		for key := range result.Categories {
			categoryKeys = append(categoryKeys, key)
		}
		sort.Strings(categoryKeys)
		for _, key := range categoryKeys {
			label := categoryLabels[key]
			if label == "" {
				label = key
			}
			data.Categories = append(data.Categories, web.VerwaltungDemoResetCount{Label: label, Count: result.Categories[key]})
		}
		a.renderVerwaltungDemoReset(w, r, ac, data)
	default:
		http.Error(w, "Ungültiger Bestätigungsschritt.", http.StatusBadRequest)
	}
}

func demoResetAnchor(now time.Time) time.Time {
	location, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		location = time.Local
	}
	local := now.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), 12, 0, 0, 0, location)
}

func (a *app) renderVerwaltungDemoReset(w http.ResponseWriter, r *http.Request, ac authCtx, data web.VerwaltungDemoResetData) {
	var rendered bytes.Buffer
	if err := web.VerwaltungDemoResetPage(a.verwaltungShell(&ac, "settings"), data).Render(r.Context(), &rendered); err != nil {
		a.inboxError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), ac.tenant.Slug))
}

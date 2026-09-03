package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

func (a *app) textbausteinListPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	repo, ok := a.textbausteinAdminRepository(w, r, &ac)
	if !ok {
		return
	}
	items, err := repo.List(r.Context())
	if err != nil {
		a.inboxError(w, err)
		return
	}
	data := web.TextbausteinListData{Count: len(items), Flash: strings.TrimSpace(r.URL.Query().Get("flash"))}
	byCategory := make(map[string][]web.TextbausteinRow)
	for _, item := range items {
		status := "Inaktiv"
		if item.Active {
			status = "Aktiv"
		}
		updated := "Noch nicht geändert"
		if !item.UpdatedAt.IsZero() {
			updated = item.UpdatedAt.Local().Format("02.01.2006, 15:04")
		}
		byCategory[item.Category] = append(byCategory[item.Category], web.TextbausteinRow{
			Key: item.Key, Title: item.Title, Status: status, Active: item.Active, Updated: updated,
			EditURL: "/app/verwaltung/textbausteine/" + url.PathEscape(item.Key),
		})
	}
	known := make(map[string]bool)
	for _, category := range store.IntakeCategories() {
		known[category.Key] = true
		if rows := byCategory[category.Key]; len(rows) > 0 {
			data.Groups = append(data.Groups, web.TextbausteinGroup{Label: category.Label, Items: rows})
		}
	}
	unknown := make([]string, 0)
	for category := range byCategory {
		if !known[category] {
			unknown = append(unknown, category)
		}
	}
	sort.Strings(unknown)
	for _, category := range unknown {
		data.Groups = append(data.Groups, web.TextbausteinGroup{Label: category, Items: byCategory[category]})
	}
	a.renderTextbausteinList(w, r, ac, data)
}

func (a *app) textbausteinFormPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	repo, ok := a.textbausteinAdminRepository(w, r, &ac)
	if !ok {
		return
	}
	key := strings.TrimSpace(r.PathValue("key"))
	if key == "neu" {
		a.renderTextbausteinForm(w, r, ac, textbausteinFormData(store.Textbaustein{Category: store.IntakeCategoryOther, Active: true}, true, ""), http.StatusOK)
		return
	}
	item, err := repo.Get(r.Context(), key)
	if errors.Is(err, store.ErrIntakeNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		a.inboxError(w, err)
		return
	}
	a.renderTextbausteinForm(w, r, ac, textbausteinFormData(item, false, ""), http.StatusOK)
}

func (a *app) textbausteinSaveAction(w http.ResponseWriter, r *http.Request, ac authCtx) {
	repo, ok := a.textbausteinAdminRepository(w, r, &ac)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Die Eingaben konnten nicht gelesen werden.", http.StatusBadRequest)
		return
	}
	pathKey := strings.TrimSpace(r.PathValue("key"))
	isNew := pathKey == "neu"
	item := store.Textbaustein{
		Key: pathKey, Title: strings.TrimSpace(r.FormValue("title")), Category: strings.TrimSpace(r.FormValue("category")),
		Body: strings.TrimSpace(r.FormValue("body")), Active: r.FormValue("active") == "1", Placeholders: textbausteinPlaceholderNames(),
	}
	if item.Title == "" || item.Body == "" || !isTextbausteinCategory(item.Category) {
		if isNew {
			item.Key = ""
		}
		a.renderTextbausteinForm(w, r, ac, textbausteinFormData(item, isNew, "Bitte Titel, Kategorie und Text vollständig ausfüllen."), http.StatusBadRequest)
		return
	}
	if isNew {
		items, err := repo.List(r.Context())
		if err != nil {
			a.inboxError(w, err)
			return
		}
		item.Key = uniqueTextbausteinKey(item.Title, items)
	} else {
		before, err := repo.Get(r.Context(), pathKey)
		if errors.Is(err, store.ErrIntakeNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			a.inboxError(w, err)
			return
		}
		item.Key = before.Key
	}
	if err := repo.Upsert(r.Context(), item); err != nil {
		a.inboxError(w, err)
		return
	}
	a.auditTextbausteinChange(&ac, item.Key, "Textbaustein gespeichert", map[string]string{"title": item.Title, "active": fmt.Sprint(item.Active)})
	http.Redirect(w, r, "/app/verwaltung/textbausteine?flash="+url.QueryEscape("Textbaustein gespeichert"), http.StatusSeeOther)
}

func (a *app) textbausteinDeactivateAction(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.textbausteinSetActive(w, r, ac, false)
}

func (a *app) textbausteinActivateAction(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.textbausteinSetActive(w, r, ac, true)
}

func (a *app) textbausteinSetActive(w http.ResponseWriter, r *http.Request, ac authCtx, active bool) {
	repo, ok := a.textbausteinAdminRepository(w, r, &ac)
	if !ok {
		return
	}
	key := strings.TrimSpace(r.PathValue("key"))
	item, err := repo.Get(r.Context(), key)
	if errors.Is(err, store.ErrIntakeNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		a.inboxError(w, err)
		return
	}
	item.Active = active
	if err := repo.Upsert(r.Context(), item); err != nil {
		a.inboxError(w, err)
		return
	}
	summary := "Textbaustein deaktiviert"
	if active {
		summary = "Textbaustein aktiviert"
	}
	a.auditTextbausteinChange(&ac, item.Key, summary, map[string]string{"active": fmt.Sprint(active)})
	http.Redirect(w, r, "/app/verwaltung/textbausteine?flash="+url.QueryEscape(summary), http.StatusSeeOther)
}

func (a *app) textbausteinAdminRepository(w http.ResponseWriter, _ *http.Request, ac *authCtx) (store.TextbausteinRepository, bool) {
	if !a.isOrganisationAdmin(ac) {
		http.Error(w, "Dieser Bereich ist Organisationsadministratoren vorbehalten.", http.StatusForbidden)
		return nil, false
	}
	orgKey, ok := a.inboxOrganisationKey(ac)
	if !ok || a.textbausteine == nil {
		http.Error(w, "Textbausteine sind nicht verfügbar.", http.StatusServiceUnavailable)
		return nil, false
	}
	return a.textbausteine(orgKey), true
}

func (a *app) auditTextbausteinChange(ac *authCtx, key, summary string, details map[string]string) {
	orgKey, _ := a.inboxOrganisationKey(ac)
	for _, tenant := range a.managedTenants(ac) {
		a.recordAudit(store.AuditEvent{
			TenantSlug: tenant.Ref.Slug, ActorEmail: ac.email, ActorRole: tenant.Role,
			Action: store.AuditActionTextbausteinChanged, TargetType: "textbaustein", TargetID: key,
			Summary: summary, Details: map[string]string{"organisation": orgKey, "title": details["title"], "active": details["active"]},
		})
	}
}

func (a *app) renderTextbausteinList(w http.ResponseWriter, r *http.Request, ac authCtx, data web.TextbausteinListData) {
	var rendered bytes.Buffer
	if err := web.TextbausteinListPage(a.verwaltungShell(&ac, "textbausteine"), data).Render(r.Context(), &rendered); err != nil {
		a.inboxError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), ac.tenant.Slug))
}

func (a *app) renderTextbausteinForm(w http.ResponseWriter, r *http.Request, ac authCtx, data web.TextbausteinFormData, status int) {
	var rendered bytes.Buffer
	if err := web.TextbausteinFormPage(a.verwaltungShell(&ac, "textbausteine"), data).Render(r.Context(), &rendered); err != nil {
		a.inboxError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), ac.tenant.Slug))
}

func textbausteinFormData(item store.Textbaustein, isNew bool, formError string) web.TextbausteinFormData {
	data := web.TextbausteinFormData{
		Key: item.Key, Title: item.Title, Body: item.Body, Active: item.Active, New: isNew, Error: formError,
		Action: "/app/verwaltung/textbausteine/" + url.PathEscape(item.Key), Placeholders: store.TextbausteinPlaceholders(),
	}
	if isNew {
		data.Action = "/app/verwaltung/textbausteine/neu"
	}
	for _, category := range store.IntakeCategories() {
		data.Categories = append(data.Categories, web.TextbausteinCategoryOption{Key: category.Key, Label: category.Label, Selected: item.Category == category.Key})
	}
	return data
}

func isTextbausteinCategory(key string) bool {
	for _, category := range store.IntakeCategories() {
		if key == category.Key {
			return true
		}
	}
	return false
}

func textbausteinPlaceholderNames() []string {
	placeholders := store.TextbausteinPlaceholders()
	for index := range placeholders {
		placeholders[index] = strings.TrimSuffix(strings.TrimPrefix(placeholders[index], "{{"), "}}")
	}
	return placeholders
}

func uniqueTextbausteinKey(title string, items []store.Textbaustein) string {
	base := textbausteinKey(title)
	if base == "" {
		base = "textbaustein"
	}
	taken := map[string]bool{"neu": true}
	for _, item := range items {
		taken[item.Key] = true
	}
	if !taken[base] {
		return base
	}
	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s-%d", base, suffix)
		if !taken[candidate] {
			return candidate
		}
	}
}

func textbausteinKey(title string) string {
	replacer := strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss")
	title = replacer.Replace(strings.ToLower(strings.TrimSpace(title)))
	var key strings.Builder
	dash := false
	for _, r := range title {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			if dash && key.Len() > 0 {
				key.WriteByte('-')
			}
			key.WriteRune(r)
			dash = false
			continue
		}
		dash = true
	}
	return key.String()
}

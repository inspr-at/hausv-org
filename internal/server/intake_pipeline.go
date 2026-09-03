package server

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/ai"
	"github.com/inspr-at/hausv-org/internal/store"
)

// processIntake applies the organisation's explicit trust policy. It is
// deliberately fail-closed: automatic handling additionally requires a live
// suggester, even when an imported item carries a precomputed suggestion.
func (a *app) processIntake(ctx context.Context, orgKey string, item store.IntakeItem, actor string) (store.IntakeItem, error) {
	repo, settingsRepo, err := a.intakeStores(orgKey)
	if err != nil {
		return item, err
	}
	settings, err := settingsRepo.Get(ctx)
	if err != nil {
		return item, err
	}

	if item.Suggestion == nil {
		if a.triage == nil {
			return item, nil
		}
		suggestion, suggestErr := a.suggestIntake(ctx, orgKey, item)
		if ctxErr := ctx.Err(); ctxErr != nil {
			item.Suggestion = nil
			item.Status = store.IntakeStatusOpen
			return item, ctxErr
		}
		if errors.Is(suggestErr, context.Canceled) || errors.Is(suggestErr, context.DeadlineExceeded) {
			// A cancelled or timed-out request must never leave a partial model
			// response behind. The caller keeps the intake item open.
			item.Suggestion = nil
			item.Status = store.IntakeStatusOpen
			return item, suggestErr
		}
		if suggestErr != nil && (suggestion.Category == "" || !errors.Is(suggestErr, ai.ErrUncertain)) {
			return item, suggestErr
		}
		item.Suggestion = &suggestion
	}
	if item.Suggestion == nil {
		return item, nil
	}

	level := settings.TrustLevels[item.Suggestion.Category]
	if level == "manual" {
		item.Suggestion = nil
		item.Status = store.IntakeStatusOpen
		item.UpdatedAt = time.Now().UTC()
		if err := repo.Create(ctx, item); err != nil {
			return item, err
		}
		return repo.Get(ctx, item.ID)
	}
	if level == "auto" && a.triage != nil && settings.AutoEnabled && item.Suggestion.Confidence["overall"] >= settings.AutoThreshold {
		if err := a.handleIntake(ctx, orgKey, item, intakeHandleOptions{status: store.IntakeStatusAuto, action: "auto", actorName: "System (KI)", auditAction: store.AuditActionIssueAIAuto, counter: "auto"}); err != nil {
			return item, err
		}
		return repo.Get(ctx, item.ID)
	}
	if err := repo.UpdateSuggestion(ctx, item.ID, *item.Suggestion, store.IntakeStatusProposed); err != nil {
		return item, err
	}
	return repo.Get(ctx, item.ID)
}

// processOpenIntake processes only unhandled items without a suggestion. A
// second call therefore does not repeat already acknowledged side effects.
func (a *app) processOpenIntake(ctx context.Context, orgKey string, limit int) (n int, err error) {
	repo, _, err := a.intakeStores(orgKey)
	if err != nil {
		return 0, err
	}
	items, err := repo.List(ctx, store.IntakeFilter{Statuses: []store.IntakeStatus{store.IntakeStatusOpen}, Limit: limit})
	if err != nil {
		return 0, err
	}
	for _, item := range items {
		if item.Suggestion != nil {
			continue
		}
		if _, err := a.processIntake(ctx, orgKey, item, "System (KI)"); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (a *app) intakeStores(orgKey string) (store.IntakeRepository, store.OrgSettingsRepository, error) {
	orgKey = normalizeSlug(orgKey)
	if a == nil || orgKey == "" || a.intake == nil || a.orgSettings == nil {
		return nil, nil, fmt.Errorf("intake stores unavailable")
	}
	return a.intake(orgKey), a.orgSettings(orgKey), nil
}

func (a *app) suggestIntake(ctx context.Context, orgKey string, item store.IntakeItem) (store.IntakeSuggestion, error) {
	if a == nil || a.triage == nil {
		return store.IntakeSuggestion{}, ai.ErrUnavailable
	}
	templates := []store.Textbaustein{}
	if a.textbausteine != nil {
		var err error
		templates, err = a.textbausteine(orgKey).List(ctx)
		if err != nil {
			return store.IntakeSuggestion{}, err
		}
	}
	houses := a.organisationHouseHints(orgKey)
	categories := make([]ai.CategoryRule, 0, len(store.IntakeCategories()))
	for _, category := range store.IntakeCategories() {
		categories = append(categories, ai.CategoryRule{Key: category.Key, Label: category.Label, DefaultPriority: store.IssuePriorityNorm})
	}
	templateHints := make([]ai.TemplateHint, 0, len(templates))
	for _, item := range templates {
		if item.Active {
			templateHints = append(templateHints, ai.TemplateHint{Key: item.Key, Category: item.Category, Title: item.Title, Body: item.Body})
		}
	}
	assignedSlug := normalizeSlug(item.TenantSlug)
	assignedName := ""
	if tenant, _, ok := a.organisationTenant(orgKey, assignedSlug); ok {
		assignedName = houseDisplayName(tenant)
	}
	input := ai.TriageInput{Organisation: orgKey, AssignedHouseSlug: assignedSlug, AssignedHouseName: assignedName, AssignedUnit: item.Unit, Source: string(item.Source), Subject: item.Subject, Body: item.Body, FromName: item.FromName, FromEmail: item.FromEmail, FromPhone: item.FromPhone, ReceivedAt: item.ReceivedAt, Houses: houses, Categories: categories, Templates: templateHints, Assignees: a.organisationAssigneeHints(orgKey)}
	suggestion, err := a.triage.Suggest(ctx, input)
	confidence := make(map[string]float64, len(suggestion.Confidence))
	for key, value := range suggestion.Confidence {
		confidence[key] = value
	}
	stored := store.IntakeSuggestion{Source: "model", Model: suggestion.Model, PromptHash: suggestion.PromptHash, Category: suggestion.Category, Priority: suggestion.Priority, TenantSlug: suggestion.HouseSlug, Unit: suggestion.Unit, Assignee: suggestion.Assignee, TemplateKey: suggestion.TemplateKey, Reply: suggestion.Reply, Actions: append([]string(nil), suggestion.Actions...), Confidence: confidence, CreatedAt: time.Now().UTC()}
	if stored.TenantSlug == "" {
		stored.TenantSlug = item.TenantSlug
	}
	if stored.Unit == "" {
		stored.Unit = item.Unit
	}
	stored.Reply, stored.Unfilled = store.FillReply(stored.Reply, a.intakeReplyValues(orgKey, item, stored))
	stored.Reply = store.TidyReply(stored.Reply)
	if len(stored.Unfilled) > 0 && stored.Confidence["overall"] > 0.5 {
		stored.Confidence["overall"] = 0.5
	}
	if stored.Category != "" {
		a.recordIntakeAudit(stored.TenantSlug, "System (KI)", store.AuditActionIssueAISuggest, item, stored, "KI-Vorschlag erstellt")
	}
	return stored, err
}

// intakeReplyValues supplies the labels visible in the case view. Model text
// never determines the replacement values themselves.
func (a *app) intakeReplyValues(orgKey string, item store.IntakeItem, suggestion store.IntakeSuggestion) map[string]string {
	houseSlug := normalizeSlug(firstNonEmpty(suggestion.TenantSlug, item.TenantSlug))
	houseName := ""
	if tenant, _, ok := a.organisationTenant(orgKey, houseSlug); ok {
		houseName = houseDisplayName(tenant)
	}
	assigneeName := ""
	for _, hint := range a.organisationAssigneeHints(orgKey) {
		if hint.Key == suggestion.Assignee {
			assigneeName = hint.Name
			break
		}
	}
	return map[string]string{
		"Name":      item.FromName,
		"Haus":      houseName,
		"Einheit":   firstNonEmpty(suggestion.Unit, item.Unit),
		"Nummer":    item.ID,
		"Zuständig": assigneeName,
		// The seed templates say "innerhalb von {{Frist}}", so Frist is a
		// duration derived from the priority (the same ladder as the due
		// date), not a calendar date.
		"Frist": intakeFristPhrase(suggestion.Priority),
		// A contractor is never known at triage time; the generic phrase the
		// fixture uses is a value, not a gap to flag.
		"Handwerker": "unseren zuständigen Fachbetrieb",
	}
}

// intakeFristPhrase words the response deadline the way the Textbausteine
// expect it ("innerhalb von …"), following the due-date ladder per priority.
func intakeFristPhrase(priority string) string {
	switch priority {
	case store.IssuePriorityUrgent:
		return "vier Stunden"
	case store.IssuePriorityHigh:
		return "24 Stunden"
	case store.IssuePriorityLow:
		return "zwei Wochen"
	default:
		return "vier Werktagen"
	}
}

func (a *app) organisationHouseHints(orgKey string) []ai.HouseHint {
	orgKey = normalizeSlug(orgKey)
	items := []ai.HouseHint{}
	for slug, tenant := range a.tenants {
		if normalizeSlug(tenant.Organisation) != orgKey && !(tenant.Organisation == "" && normalizeSlug(slug) == orgKey) {
			continue
		}
		hint := ai.HouseHint{Slug: tenant.Slug, Name: houseDisplayName(tenant), Address: tenant.Address}
		if identity, ok := a.tenantIdentity(tenant.Slug); ok {
			if units := a.repositoriesFor(identity.Ref()).units; units != nil {
				for _, unit := range units.List() {
					hint.Units = append(hint.Units, unit.Label)
				}
			}
		}
		items = append(items, hint)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func (a *app) organisationAssigneeHints(orgKey string) []ai.AssigneeHint {
	houses := map[string]struct{}{}
	for _, house := range a.organisationHouseHints(orgKey) {
		houses[house.Slug] = struct{}{}
	}
	items := []ai.AssigneeHint{}
	for email, profile := range a.profiles {
		managed := []string{}
		for slug := range houses {
			if role := normalizeRole(profile.ForTenant(slug).Role); profile.HasTenant(slug) && (role == roleManager || role == roleAdmin) {
				managed = append(managed, slug)
			}
		}
		if len(managed) == 0 {
			continue
		}
		sort.Strings(managed)
		key := strings.Split(normalizeEmail(email), "@")[0]
		items = append(items, ai.AssigneeHint{Key: key, Name: profile.DisplayName(), Houses: managed})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

type intakeHandleOptions struct {
	status      store.IntakeStatus
	action      string
	actorEmail  string
	actorName   string
	auditAction string
	counter     string
}

func (a *app) handleIntake(ctx context.Context, orgKey string, item store.IntakeItem, options intakeHandleOptions) error {
	if item.Suggestion == nil {
		return fmt.Errorf("intake suggestion required")
	}
	tenantSlug := normalizeSlug(firstNonEmpty(item.Suggestion.TenantSlug, item.TenantSlug))
	tenant, ref, ok := a.organisationTenant(orgKey, tenantSlug)
	if !ok {
		return fmt.Errorf("assigned house not found")
	}
	repositories := a.repositoriesFor(ref)
	if repositories.issues == nil {
		return fmt.Errorf("issue repository unavailable")
	}
	priority := store.NormalizeIssuePriority(item.Suggestion.Priority)
	if priority == "" {
		priority = store.IssuePriorityNorm
	}
	category := intakeIssueCategory(item.Suggestion.Category)
	dueAt := item.DueAt
	if dueAt.IsZero() {
		dueAt = intakeDueAt(time.Now(), priority)
	}
	actorEmail := normalizeEmail(options.actorEmail)
	if actorEmail == "" {
		actorEmail = resolveIntakeActorEmail(item, orgKey)
	}
	actorName := strings.TrimSpace(options.actorName)
	if actorName == "" {
		actorName = actorEmail
	}
	authorEmail := normalizeEmail(item.FromEmail)
	if authorEmail == "" {
		authorEmail = actorEmail
	}
	if authorEmail == "" {
		authorEmail = "inbox@hausv.invalid"
	}
	authorName := strings.TrimSpace(item.FromName)
	if authorName == "" {
		authorName = "Telefonnotiz"
	}
	assignee := a.resolveOrganisationAssignee(orgKey, item.Suggestion.Assignee, tenantSlug)
	body := "Kategorie: " + intakeCategoryLabel(item.Suggestion.Category) + "\n\n" + strings.TrimSpace(item.Body)
	issue, found := a.findIntakeIssue(repositories.issues, item)
	if !found {
		created, err := repositories.issues.Create(store.ResidentIssue{TenantSlug: tenantSlug, Source: string(item.Source), DueAt: dueAt, IntakeID: item.ID, AuthorEmail: authorEmail, AuthorName: authorName, Category: category, Title: item.Subject, Body: body, LocationType: store.IssueLocationUnit, LocationDetail: firstNonEmpty(item.Suggestion.Unit, item.Unit), Status: store.IssueStatusNew, Priority: priority, AssigneeEmail: assignee})
		if err != nil {
			return err
		}
		issue = created
		a.notifyIssueCreated(tenant, issue)
	} else {
		updated, exists, err := repositories.issues.UpdateWorkflow(issue.ID, store.IssueWorkflowUpdate{Status: store.IssueStatusNew, Priority: priority, AssigneeEmail: assignee, Body: body, LocationType: store.IssueLocationUnit, LocationDetail: firstNonEmpty(item.Suggestion.Unit, item.Unit), UpdateDetails: true, ActorEmail: actorEmail, ActorName: actorName, ChangedAt: time.Now()})
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("existing issue disappeared")
		}
		issue = updated
	}
	if options.status != store.IntakeStatusRejected && strings.TrimSpace(item.Suggestion.Reply) != "" && !hasIntakeReply(issue, item.ID) {
		updated, exists, err := repositories.issues.AddComment(issue.ID, store.IssueComment{ID: "intake-reply-" + item.ID, AuthorEmail: actorEmail, AuthorName: actorName, Body: item.Suggestion.Reply, Kind: store.IssueCommentKindInformation, CreatedAt: time.Now()})
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("issue comment target missing")
		}
		issue = updated
		a.recordAudit(store.AuditEvent{TenantSlug: tenantSlug, ActorEmail: actorEmail, ActorRole: roleManager, Action: store.AuditActionIssueComment, TargetType: "issue", TargetID: issue.ID, Summary: "Kommentar zu Anliegen hinzugefügt", Details: map[string]string{"comment_id": "intake-reply-" + item.ID, "message_type": store.IssueCommentKindInformation, "has_file": "false", "file_count": "0"}})
		a.notifyIssueUpdated(tenant, issue, actorEmail, "Neue Antwort zu Anliegen \""+issue.Title+"\"")
	}
	repo := a.intake(orgKey)
	if err := repo.UpdateHandling(ctx, item.ID, store.IntakeHandling{Action: options.action, ByEmail: options.actorEmail, ByName: actorName, At: time.Now().UTC()}, options.status, issue.ID); err != nil {
		return err
	}
	a.recordIntakeAudit(tenantSlug, actorEmail, options.auditAction, item, *item.Suggestion, "KI-Einordnung bearbeitet")
	if options.counter != "" {
		if err := a.orgSettings(orgKey).Increment(ctx, options.counter); err != nil {
			return err
		}
	}
	return nil
}

func (a *app) findIntakeIssue(repo store.IssueRepository, item store.IntakeItem) (store.ResidentIssue, bool) {
	if item.IssueID != "" {
		if issue, ok := repo.Get(item.IssueID); ok {
			return issue, true
		}
	}
	for _, issue := range repo.List() {
		if issue.IntakeID == item.ID {
			return issue, true
		}
	}
	return store.ResidentIssue{}, false
}

func hasIntakeReply(issue store.ResidentIssue, intakeID string) bool {
	for _, comment := range issue.Comments {
		if comment.ID == "intake-reply-"+intakeID {
			return true
		}
	}
	return false
}

func (a *app) organisationTenant(orgKey, slug string) (tenantConfig, store.TenantRef, bool) {
	tenant, ok := a.tenantBySlug(slug)
	if !ok || (normalizeSlug(tenant.Organisation) != normalizeSlug(orgKey) && !(tenant.Organisation == "" && normalizeSlug(orgKey) == normalizeSlug(slug))) {
		return tenantConfig{}, store.TenantRef{}, false
	}
	identity, ok := a.tenantIdentity(slug)
	if !ok {
		return tenantConfig{}, store.TenantRef{}, false
	}
	return tenant, identity.Ref(), true
}

func (a *app) resolveOrganisationAssignee(orgKey, value, tenantSlug string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	for email, profile := range a.profiles {
		normalized := normalizeEmail(email)
		if !profile.HasTenant(tenantSlug) || normalizeRole(profile.ForTenant(tenantSlug).Role) != roleManager {
			continue
		}
		if value == normalized || value == strings.Split(normalized, "@")[0] || value == normalizeSlug(profile.DisplayName()) {
			return normalized
		}
	}
	return ""
}

func resolveIntakeActorEmail(item store.IntakeItem, orgKey string) string {
	if strings.Contains(item.ExternalRef, "@") {
		if cut := strings.Index(item.ExternalRef, "|"); cut > 0 {
			return normalizeEmail(item.ExternalRef[:cut])
		}
		return normalizeEmail(item.ExternalRef)
	}
	return normalizeSlug(orgKey) + "@hausv.invalid"
}

func intakeIssueCategory(category string) string {
	if category == store.IntakeCategoryRepair {
		return "Reparatur"
	}
	if category == store.IntakeCategoryHouseRules || category == store.IntakeCategoryOther {
		return "Sonstiges"
	}
	return "Frage"
}
func intakeCategoryLabel(key string) string {
	for _, category := range store.IntakeCategories() {
		if category.Key == key {
			return category.Label
		}
	}
	return "Sonstiges"
}

func intakeDueAt(now time.Time, priority string) time.Time {
	days := 3
	normalized := store.NormalizeIssuePriority(priority)
	switch normalized {
	case store.IssuePriorityUrgent:
		days = 0
	case store.IssuePriorityHigh:
		due := now.AddDate(0, 0, 1)
		return time.Date(due.Year(), due.Month(), due.Day(), 17, 0, 0, 0, due.Location()).UTC()
	case store.IssuePriorityLow:
		days = 10
	}
	due := now
	for added := 0; added < days; {
		due = due.AddDate(0, 0, 1)
		if due.Weekday() != time.Saturday && due.Weekday() != time.Sunday {
			added++
		}
	}
	return time.Date(due.Year(), due.Month(), due.Day(), 17, 0, 0, 0, due.Location()).UTC()
}

func (a *app) recordIntakeAudit(tenantSlug, actor, action string, item store.IntakeItem, suggestion store.IntakeSuggestion, summary string) {
	if tenantSlug == "" || action == "" {
		return
	}
	a.recordAudit(store.AuditEvent{TenantSlug: tenantSlug, ActorEmail: normalizeEmail(actor), ActorRole: roleManager, Action: action, TargetType: "intake", TargetID: item.ID, Summary: summary, Details: map[string]string{"intake_id": item.ID, "category": suggestion.Category, "confidence": fmt.Sprintf("%.2f", suggestion.Confidence["overall"])}})
}

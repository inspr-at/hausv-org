package server

import "strings"

// scopedAuditEvents applies the current authorization state before presenting
// history to non-management roles. This deliberately favors current access:
// once an issue is closed or a provider assignment is revoked, its shared
// history disappears from that provider's view as well.
func (a *app) scopedAuditEvents(ac authCtx, events []auditEvent) []auditEvent {
	out := make([]auditEvent, 0, len(events))
	for _, event := range events {
		if !a.canViewScopedAuditEvent(ac, event) {
			continue
		}
		out = append(out, scopedAuditEvent(event, ac.email))
	}
	return out
}

func (a *app) canViewScopedAuditEvent(ac authCtx, event auditEvent) bool {
	if normalizeEmail(event.ActorEmail) == normalizeEmail(ac.email) {
		return true
	}
	switch strings.TrimSpace(event.TargetType) {
	case "issue":
		return a.canViewAuditIssue(ac, event.TargetID)
	case "unit", "store.Unit":
		return a.canViewAuditUnit(ac, event.TargetID)
	case "attachment":
		entityType := normalizeAttachmentEntity(event.Details["entity_type"])
		entityID := strings.TrimSpace(event.Details["entity_id"])
		switch entityType {
		case "issue", "issue-estimate":
			return a.canViewAuditIssue(ac, entityID)
		case "issue-comment":
			issue, _, found := a.issueCommentTarget(ac.tenant.Slug, entityID)
			return found && a.canViewIssueForActor(ac.tenant.Slug, issue, ac.email, ac.role)
		}
	}
	return false
}

func (a *app) canViewAuditIssue(ac authCtx, issueID string) bool {
	if a == nil || a.issueStore == nil {
		return false
	}
	issue, found := a.issueStore.Get(ac.tenant.Slug, strings.TrimSpace(issueID))
	return found && a.canViewIssueForActor(ac.tenant.Slug, issue, ac.email, ac.role)
}

func (a *app) canViewAuditUnit(ac authCtx, unitID string) bool {
	if a == nil || a.unitStore == nil {
		return false
	}
	members := ac.repositories.units.MembersForUnit(unitID)
	return members.Found && (emailListContains(members.Owners, ac.email) || emailListContains(members.Renters, ac.email))
}

func scopedAuditEvent(event auditEvent, selfEmail string) auditEvent {
	event = copyAuditEvent(event)
	event.Summary = scopedAuditSummary(event, selfEmail)
	switch {
	case normalizeEmail(event.ActorEmail) == normalizeEmail(selfEmail):
		event.ActorEmail = "Sie"
	case normalizeRole(event.ActorRole) == roleAdmin || normalizeRole(event.ActorRole) == roleManager:
		event.ActorEmail = "Verwaltung"
	case normalizeRole(event.ActorRole) == roleServiceProvider:
		event.ActorEmail = "Dienstleister"
	default:
		event.ActorEmail = "Hausgemeinschaft"
	}
	event.ActorRole = ""
	event.Details = scopedAuditDetails(event.Details)
	return event
}

func scopedAuditSummary(event auditEvent, selfEmail string) string {
	byCurrentUser := normalizeEmail(event.ActorEmail) == normalizeEmail(selfEmail)
	switch normalizeAuditAction(event.Action) {
	case auditActionIssueComment:
		switch strings.TrimSpace(event.Details["message_type"]) {
		case issueCommentKindQuestion:
			if !byCurrentUser {
				return "Rückfrage erhalten"
			}
			return "Rückfrage gestellt"
		case issueCommentKindAnswer:
			if !byCurrentUser {
				return "Antwort erhalten"
			}
			return "Rückfrage beantwortet"
		default:
			return "Nachricht hinzugefügt"
		}
	case auditActionIssueCommentDelete:
		return "Nachricht entfernt"
	case auditActionIssueWorkflow:
		return "Anliegen bearbeitet"
	default:
		return auditActionLabel(event.Action)
	}
}

func scopedAuditDetails(details map[string]string) map[string]string {
	if len(details) == 0 {
		return nil
	}
	allowed := map[string]bool{
		"access":      true,
		"assigned":    true,
		"auth_method": true,
		"entity_type": true,
		"file_count":  true,
		"format":      true,
		"has_file":    true,
		"priority":    true,
		"rejected":    true,
		"source":      true,
		"status":      true,
		"unclear":     true,
	}
	out := map[string]string{}
	for key, value := range details {
		if allowed[key] {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func filterAuditEvents(events []auditEvent, action string, query string) []auditEvent {
	action = normalizeAuditAction(action)
	query = strings.ToLower(strings.TrimSpace(query))
	if action == "" && query == "" {
		return events
	}
	out := make([]auditEvent, 0, len(events))
	for _, event := range events {
		if action != "" && normalizeAuditAction(event.Action) != action {
			continue
		}
		if query != "" && !auditEventMatches(event, query) {
			continue
		}
		out = append(out, event)
	}
	return out
}

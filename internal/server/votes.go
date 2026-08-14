package server

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/version"
)

func (a *app) createBallot(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if !hasCapability(role, capabilityManageVotes) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if a.voteStore == nil {
		http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	item, err := ballotFromForm(r, tenant.Slug, email)
	if err != nil {
		http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
		return
	}
	attachmentHeaders, err := attachmentFormHeaders(r, maxIssueAttachmentCount, "attachments")
	if err != nil {
		http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
		return
	}
	created, err := a.voteStore.Create(item)
	if err != nil {
		logError("ballot create failed", err, "tenant", tenant.Slug, "actor", redactedEmail(email))
		http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
		return
	}
	if len(attachmentHeaders) > 0 {
		if a.attachmentStore == nil {
			_, _ = a.voteStore.Delete(tenant.Slug, created.ID)
			http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
			return
		}
		if _, err := a.attachmentStore.CreateUploaded(tenant.Slug, "ballot", created.ID, email, uploadedFilesFromHeaders(attachmentHeaders), time.Now()); err != nil {
			_, _ = a.voteStore.Delete(tenant.Slug, created.ID)
			http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
			return
		}
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     auditActionVoteCreate,
		TargetType: "ballot",
		TargetID:   created.ID,
		Summary:    "Abstimmung angelegt",
		Details: map[string]string{
			"title":     created.Title,
			"type":      created.Type,
			"weighting": ballotWeightingLabel(created.Weighting),
			"quorum":    formatMiteigentumsanteil(created.QuorumPPM),
			"reminder":  formatBallotReminder(created.ReminderBeforeMinutes),
		},
	})
	http.Redirect(w, r, "/app/abstimmungen?vote=created#ballot-"+url.PathEscape(created.ID), http.StatusSeeOther)
}

func (a *app) openBallot(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.updateBallotStatus(w, r, ac, ballotStatusOpen)
}

func (a *app) closeBallot(w http.ResponseWriter, r *http.Request, ac authCtx) {
	a.updateBallotStatus(w, r, ac, ballotStatusClosed)
}

func (a *app) updateBallotStatus(w http.ResponseWriter, r *http.Request, ac authCtx, status string) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if !hasCapability(role, capabilityManageVotes) {
		http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if a.voteStore == nil {
		http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	var (
		updated ballot
		found   bool
		err     error
		action  string
		summary string
		statusQ string
	)
	switch status {
	case ballotStatusOpen:
		updated, found, err = a.voteStore.Open(tenant.Slug, id, time.Now())
		action = auditActionVoteOpen
		summary = "Abstimmung geöffnet"
		statusQ = "opened"
	case ballotStatusClosed:
		updated, found, err = a.voteStore.Close(tenant.Slug, id, time.Now())
		action = auditActionVoteClose
		summary = "Abstimmung geschlossen"
		statusQ = "closed"
	default:
		err = fmt.Errorf("invalid ballot status")
	}
	if err != nil {
		logError("ballot status update failed", err, "tenant", tenant.Slug, "ballot_id", id)
		http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
		return
	}
	if !found {
		http.Redirect(w, r, "/app/abstimmungen?vote=missing", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug,
		ActorEmail: email,
		ActorRole:  role,
		Action:     action,
		TargetType: "ballot",
		TargetID:   updated.ID,
		Summary:    summary,
		Details: map[string]string{
			"title":  updated.Title,
			"status": updated.Status,
		},
	})
	http.Redirect(w, r, "/app/abstimmungen?vote="+statusQ+"#ballot-"+url.PathEscape(updated.ID), http.StatusSeeOther)
}

func ballotFromForm(r *http.Request, tenantSlug string, createdBy string) (ballot, error) {
	opensAt, err := parseOptionalLocalDateTime(r.FormValue("opens_at"), time.Time{})
	if err != nil {
		return ballot{}, err
	}
	closesAt, err := parseOptionalLocalDateTime(r.FormValue("closes_at"), time.Time{})
	if err != nil {
		return ballot{}, err
	}
	if !opensAt.IsZero() {
		opensAt = opensAt.UTC()
	}
	if !closesAt.IsZero() {
		closesAt = closesAt.UTC()
	}
	quorum, err := parseBallotQuorumPPM(r.FormValue("quorum_ppm"), r.FormValue("quorum_percent"))
	if err != nil {
		return ballot{}, err
	}
	reminderBefore, err := parseBallotReminderBeforeMinutes(r.FormValue("reminder_before_minutes"), r.FormValue("reminder_before_hours"))
	if err != nil {
		return ballot{}, err
	}
	item := ballot{
		TenantSlug:            tenantSlug,
		Title:                 strings.TrimSpace(r.FormValue("title")),
		Description:           strings.TrimSpace(r.FormValue("description")),
		Options:               ballotOptionsFromForm(r),
		Type:                  normalizeBallotType(r.FormValue("type")),
		Weighting:             normalizeBallotWeighting(r.FormValue("weighting")),
		QuorumPPM:             quorum,
		OpensAt:               opensAt,
		ClosesAt:              closesAt,
		CreatedBy:             createdBy,
		ReminderBeforeMinutes: reminderBefore,
	}
	item = normalizeBallot(item)
	if item.Title == "" || len(item.Options) < 2 || item.Type == "" || item.Weighting == "" || item.CreatedBy == "" {
		return ballot{}, fmt.Errorf("invalid ballot")
	}
	return item, nil
}

func ballotOptionsFromForm(r *http.Request) []string {
	out := append([]string(nil), r.Form["options"]...)
	if text := strings.TrimSpace(r.FormValue("options_text")); text != "" {
		for _, line := range strings.Split(text, "\n") {
			out = append(out, line)
		}
	}
	return out
}

func (a *app) ballots(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if denyServiceProviderArea(w, role) {
		return
	}
	canManage := hasCapability(role, capabilityManageVotes)
	canOversight := hasCapability(role, capabilityOversight)
	now := time.Now()
	all := []ballot{}
	if a.voteStore != nil {
		if _, err := a.voteStore.CloseExpiredTenant(tenant.Slug, now); err != nil {
			logError("ballot auto-close failed", err, "tenant", tenant.Slug)
		}
		all = a.voteStore.ListTenant(tenant.Slug)
	}
	visible := make([]ballot, 0, len(all))
	for _, item := range all {
		switch normalizeBallotStatus(item.Status) {
		case ballotStatusOpen, ballotStatusClosed:
			visible = append(visible, item)
		}
	}
	pageItems := visible
	if canManage {
		pageItems = all
	}
	views := a.ballotViewsForActor(tenant.Slug, email, role, pageItems, now, canManage || canOversight)
	pendingCount := 0
	draftCount := 0
	openCount := 0
	openWithVoteCount := 0
	openReadOnlyCount := 0
	for _, view := range views {
		if view.NeedsVote {
			pendingCount++
		}
		if view.IsDraft {
			draftCount++
		}
		if view.IsOpen {
			openCount++
			if view.HasVote {
				openWithVoteCount++
			} else if !view.CanVote {
				openReadOnlyCount++
			}
		}
	}
	overviewTitle := "Gerade nichts zu tun"
	overviewText := "Neue und abgeschlossene Abstimmungen erscheinen hier."
	overviewClass := "ok"
	if canManage {
		overviewTitle = "Noch keine Abstimmung"
		overviewText = "Legen Sie einen Entwurf an, sobald eine Entscheidung ansteht."
		overviewClass = "action"
		switch {
		case draftCount == 1:
			overviewTitle = "Ein Entwurf wartet auf Öffnung"
			overviewText = "Prüfen Sie Frage und Regeln und geben Sie die Abstimmung anschließend frei."
			overviewClass = "action"
		case draftCount > 1:
			overviewTitle = fmt.Sprintf("%d Entwürfe warten auf Öffnung", draftCount)
			overviewText = "Prüfen Sie Frage und Regeln und geben Sie die Abstimmungen anschließend frei."
			overviewClass = "action"
		case openCount == 1:
			overviewTitle = "Eine Abstimmung läuft"
			overviewText = "Frist und Teilnahme bleiben direkt bei der Abstimmung sichtbar."
			overviewClass = "ok"
		case openCount > 1:
			overviewTitle = fmt.Sprintf("%d Abstimmungen laufen", openCount)
			overviewText = "Fristen und Teilnahme bleiben direkt bei den Abstimmungen sichtbar."
			overviewClass = "ok"
		case len(views) > 0:
			overviewTitle = "Ergebnisse verfügbar"
			overviewText = "Abgeschlossene Abstimmungen und Protokolle bleiben direkt darunter erreichbar."
			overviewClass = "ok"
		}
	} else {
		switch {
		case pendingCount == 1:
			overviewTitle = "Ihre Stimme ist gefragt"
			overviewText = "Eine offene Abstimmung wartet auf Ihre Entscheidung."
			overviewClass = "action"
		case pendingCount > 1:
			overviewTitle = "Ihre Stimme ist gefragt"
			overviewText = fmt.Sprintf("%d offene Abstimmungen warten auf Ihre Entscheidung.", pendingCount)
			overviewClass = "action"
		case openCount > 0 && openWithVoteCount == openCount:
			overviewTitle = "Alles erledigt"
			overviewText = "Ihre Stimme ist gespeichert und kann bis zur Schließung geändert werden."
		case openReadOnlyCount > 0:
			overviewTitle = "Offene Abstimmung zur Information"
			overviewText = "Für diesen Zugang ist keine Stimmabgabe hinterlegt. Sie können die Abstimmung und später das Ergebnis verfolgen."
		case len(views) > 0:
			overviewTitle = "Ergebnisse verfügbar"
			overviewText = "Abgeschlossene Abstimmungen und Protokolle finden Sie direkt darunter."
		}
	}
	msg, msgOK := voteMessage(r.URL.Query().Get("vote"))
	a.render(w, "ballots", a.withBase(ac, map[string]any{
		"Title":             "Abstimmungen",
		"CanManageVotes":    canManage,
		"CanVote":           hasCapability(role, capabilityVote),
		"CanOversightVotes": canOversight,
		"ActivePage":        "abstimmungen",
		"Ballots":           views,
		"HasBallots":        len(pageItems) > 0,
		"BallotCountLabel":  pluralizeCount(len(pageItems), "Abstimmung", "Abstimmungen"),
		"VoteOverviewTitle": overviewTitle,
		"VoteOverviewText":  overviewText,
		"VoteOverviewClass": overviewClass,
		"VoteMsg":           msg,
		"VoteOK":            msgOK,
		"NowInput":          formatLocalDateTimeInput(now),
	}))
}

func (a *app) submitBallot(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if err := parseMaybeMultipartForm(w, r, maxIssueAttachmentFormBytes, maxAttachmentBytes); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(firstNonEmpty(r.FormValue("ballot_id"), r.FormValue("id"))) != "" || strings.TrimSpace(r.FormValue("option")) != "" {
		a.castVote(w, r, ac)
		return
	}
	a.createBallot(w, r, ac)
}

func (a *app) castVote(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if !hasCapability(role, capabilityVote) {
		http.Error(w, "Dieser Zugang ist für Abstimmungen lesend.", http.StatusForbidden)
		return
	}
	if a.voteStore == nil {
		http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
		return
	}
	id := strings.TrimSpace(firstNonEmpty(r.FormValue("ballot_id"), r.FormValue("id")))
	option := strings.TrimSpace(r.FormValue("option"))
	updated, found, err := a.castBallotVote(tenant.Slug, email, id, option, time.Now())
	if !found {
		http.Redirect(w, r, "/app/abstimmungen?vote=missing", http.StatusSeeOther)
		return
	}
	if err != nil {
		logError("ballot vote failed", err, "tenant", tenant.Slug, "ballot_id", id, "actor", redactedEmail(email))
		http.Redirect(w, r, "/app/abstimmungen?vote=invalid", http.StatusSeeOther)
		return
	}
	if vote, ok := updated.Votes[normalizeEmail(email)]; ok {
		a.recordAudit(auditEvent{
			TenantSlug: tenant.Slug,
			ActorEmail: email,
			ActorRole:  role,
			Action:     auditActionVoteCast,
			TargetType: "ballot",
			TargetID:   updated.ID,
			Summary:    "Stimme gespeichert",
			Details: map[string]string{
				"title":   updated.Title,
				"weight":  formatBallotResultWeight(updated.Weighting, vote.Weight),
				"cast_at": formatLocalDateTime(vote.At),
			},
		})
	}
	http.Redirect(w, r, "/app/abstimmungen?vote=cast#ballot-"+url.PathEscape(updated.ID), http.StatusSeeOther)
}

func (a *app) ballotProtocol(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, email, role := ac.tenant, ac.email, ac.role
	if !hasCapability(role, capabilityVote) && !hasCapability(role, capabilityOversight) && !hasCapability(role, capabilityManageVotes) {
		http.Error(w, "Dieses Protokoll ist für diesen Zugang nicht freigegeben.", http.StatusForbidden)
		return
	}
	if a.voteStore == nil {
		http.NotFound(w, r)
		return
	}
	now := time.Now()
	if _, err := a.voteStore.CloseExpiredTenant(tenant.Slug, now); err != nil {
		logError("ballot auto-close failed", err, "tenant", tenant.Slug)
	}
	id := strings.TrimSpace(r.PathValue("id"))
	item, found := a.voteStore.Get(tenant.Slug, id)
	if !found {
		http.NotFound(w, r)
		return
	}
	if normalizeBallotStatus(item.Status) != ballotStatusClosed {
		http.Error(w, "Protokoll erst nach Abschluss verfügbar.", http.StatusConflict)
		return
	}
	view := a.ballotViewForActor(tenant.Slug, email, role, item, now, true)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": "abstimmung-" + item.ID + "-protokoll.html"}))
	a.executeTemplate(w, "ballotProtocol", map[string]any{
		"Title":       "Abstimmungsprotokoll",
		"Tenant":      tenant,
		"Ballot":      view,
		"GeneratedAt": formatLocalDateTime(now),
		"AppVersion":  version.BuildLabel(),
	})
}

func (a *app) castBallotVote(tenantSlug string, email string, ballotID string, option string, at time.Time) (ballot, bool, error) {
	if a == nil || a.voteStore == nil {
		return ballot{}, false, nil
	}
	item, found := a.voteStore.Get(tenantSlug, ballotID)
	if !found {
		return ballot{}, false, nil
	}
	weight, eligible := a.ballotVoteWeight(tenantSlug, email, item)
	if !eligible {
		return ballot{}, true, fmt.Errorf("not eligible to vote")
	}
	return a.voteStore.CastVote(tenantSlug, ballotID, email, option, weight, at)
}

func (a *app) ballotVoteWeight(tenantSlug string, email string, item ballot) (int, bool) {
	if a == nil {
		return 0, false
	}
	tenantSlug = normalizeSlug(tenantSlug)
	email = normalizeEmail(email)
	if tenantSlug == "" || email == "" {
		return 0, false
	}
	ownerShare := 0
	if a != nil && a.unitStore != nil {
		for _, membership := range a.unitStore.UnitsForEmail(tenantSlug, email) {
			if membership.Relation == roleOwner {
				ownerShare += membership.Unit.MiteigentumsanteilPPM
			}
		}
	}
	isOwner := ownerShare > 0 || normalizeRole(a.roleFor(email, tenantSlug)) == roleOwner
	switch normalizeBallotWeighting(item.Weighting) {
	case ballotWeightingPerHead:
		if isOwner {
			return 1, true
		}
	case ballotWeightingPerShare:
		if ownerShare > 0 {
			return ownerShare, true
		}
	}
	return 0, false
}

func (a *app) ballotViewsForActor(tenantSlug string, email string, role string, items []ballot, now time.Time, includeResults bool) []ballotView {
	out := make([]ballotView, 0, len(items))
	for _, item := range items {
		out = append(out, a.ballotViewForActor(tenantSlug, email, role, item, now, includeResults))
	}
	return out
}

func (a *app) ballotViewForActor(tenantSlug string, email string, role string, item ballot, now time.Time, includeResults bool) ballotView {
	item = normalizeBallot(item)
	weight, eligible := a.ballotVoteWeight(tenantSlug, email, item)
	status, statusClass, active := ballotStatusForView(item, now)
	vote, hasVote := item.Votes[normalizeEmail(email)]
	rawStatus := normalizeBallotStatus(item.Status)
	tally := a.computeBallotTally(tenantSlug, item)
	view := ballotView{
		ID:                  item.ID,
		Title:               item.Title,
		Description:         item.Description,
		HasDescription:      item.Description != "",
		Type:                item.Type,
		Weighting:           ballotWeightingLabel(item.Weighting),
		Quorum:              formatPPMPercent(item.QuorumPPM),
		HasQuorum:           item.QuorumPPM > 0,
		ReminderLabel:       formatBallotReminder(item.ReminderBeforeMinutes),
		Status:              status,
		StatusClass:         statusClass,
		IsDraft:             rawStatus == ballotStatusDraft,
		IsOpen:              rawStatus == ballotStatusOpen && active,
		IsClosed:            rawStatus == ballotStatusClosed,
		CreatedAt:           formatLocalDateTime(item.CreatedAt),
		UpdatedAt:           formatLocalDateTime(item.UpdatedAt),
		CanManage:           hasCapability(role, capabilityManageVotes),
		CanVote:             hasCapability(role, capabilityVote) && eligible && active,
		CanOpen:             rawStatus == ballotStatusDraft,
		CanClose:            rawStatus == ballotStatusOpen,
		HasVote:             hasVote,
		VoteOption:          vote.Option,
		VoteWeight:          formatBallotWeight(weight),
		ReadOnlyMessage:     ballotReadOnlyMessage(role, item, eligible, active, hasVote),
		TotalVotes:          tally.TotalVotes,
		TotalWeight:         tally.TotalWeight,
		TotalWeightLabel:    tally.TotalWeightLabel,
		EligibleWeightLabel: tally.EligibleWeightLabel,
		Participation:       tally.ParticipationLabel,
		QuorumStatus:        tally.QuorumStatus,
		QuorumClass:         tally.QuorumClass,
		WinnerLabel:         tally.WinnerLabel,
		HasWinner:           tally.WinnerLabel != "",
		ProtocolURL:         "/app/abstimmungen/" + url.PathEscape(item.ID) + "/protokoll",
		HasProtocol:         rawStatus == ballotStatusClosed && (hasCapability(role, capabilityVote) || hasCapability(role, capabilityOversight) || hasCapability(role, capabilityManageVotes)),
		EditDialogID:        "ballot-" + item.ID,
	}
	view.NeedsVote = view.CanVote && !view.HasVote
	if !item.OpensAt.IsZero() {
		view.OpensAt = formatLocalDateTime(item.OpensAt)
		view.HasOpensAt = true
	}
	if !item.ClosesAt.IsZero() {
		view.ClosesAt = formatLocalDateTime(item.ClosesAt)
		view.HasClosesAt = true
	}
	if hasVote {
		view.VotedAt = formatLocalDateTime(vote.At)
		view.VoteWeight = formatBallotWeight(vote.Weight)
	}
	if includeResults || rawStatus == ballotStatusClosed {
		view.HasResults = true
	}
	attachments := a.attachmentViewsForEntity(tenantSlug, "ballot", item.ID, email, role)
	if len(attachments) > 0 {
		view.Attachments = attachments
		view.HasAttachments = true
	}
	for _, option := range item.Options {
		result := tally.Options[option]
		view.Options = append(view.Options, ballotOptionView{
			Value:        option,
			Label:        option,
			Selected:     hasVote && vote.Option == option,
			VoteCount:    result.Count,
			Weight:       result.Weight,
			WeightLabel:  formatBallotResultWeight(item.Weighting, result.Weight),
			Percent:      result.Percent,
			PercentStyle: strconv.Itoa(result.Percent),
		})
	}
	return view
}

func (a *app) computeBallotTally(tenantSlug string, item ballot) ballotResultSummary {
	out := ballotResultSummary{Options: map[string]ballotResultCount{}}
	item = normalizeBallot(item)
	for _, vote := range item.Votes {
		if !ballotHasOption(item, vote.Option) || vote.Weight <= 0 {
			continue
		}
		result := out.Options[vote.Option]
		result.Count++
		result.Weight += vote.Weight
		out.Options[vote.Option] = result
		out.TotalVotes++
		out.TotalWeight += vote.Weight
	}
	if out.TotalWeight > 0 {
		for option, result := range out.Options {
			result.Percent = (result.Weight*100 + out.TotalWeight/2) / out.TotalWeight
			out.Options[option] = result
		}
	}
	out.EligibleWeight = a.ballotEligibleWeightTotal(tenantSlug, item)
	if out.EligibleWeight < out.TotalWeight {
		out.EligibleWeight = out.TotalWeight
	}
	if out.EligibleWeight > 0 {
		out.ParticipationPPM = (out.TotalWeight*1_000_000 + out.EligibleWeight/2) / out.EligibleWeight
	}
	out.TotalWeightLabel = formatBallotResultWeight(item.Weighting, out.TotalWeight)
	out.EligibleWeightLabel = formatBallotResultWeight(item.Weighting, out.EligibleWeight)
	out.ParticipationLabel = formatPPMPercent(out.ParticipationPPM)
	out.QuorumReached = item.QuorumPPM == 0 || out.ParticipationPPM >= item.QuorumPPM
	if item.QuorumPPM == 0 {
		out.QuorumStatus = "Kein Quorum"
		out.QuorumClass = "ok"
	} else if out.QuorumReached {
		out.QuorumStatus = "Quorum erreicht"
		out.QuorumClass = "ok"
	} else {
		out.QuorumStatus = "Quorum offen"
		out.QuorumClass = "status-open"
	}
	out.WinnerLabel = ballotWinnerLabel(out.Options)
	return out
}

func (a *app) ballotEligibleWeightTotal(tenantSlug string, item ballot) int {
	total := 0
	for _, weight := range a.ballotEligibleWeights(tenantSlug, item) {
		total += weight
	}
	return total
}

func (a *app) ballotEligibleWeights(tenantSlug string, item ballot) map[string]int {
	weights := map[string]int{}
	tenantSlug = normalizeSlug(tenantSlug)
	switch normalizeBallotWeighting(item.Weighting) {
	case ballotWeightingPerHead:
		if a != nil && a.unitStore != nil {
			for _, unit := range a.unitStore.ListTenant(tenantSlug) {
				for _, email := range unit.OwnerEmails {
					if email = normalizeEmail(email); email != "" {
						weights[email] = 1
					}
				}
			}
		}
		if a != nil {
			for _, row := range a.userRows(tenantSlug) {
				email := normalizeEmail(row.Email)
				if email != "" && normalizeRole(row.Role) == roleOwner {
					weights[email] = 1
				}
			}
		}
	case ballotWeightingPerShare:
		if a != nil && a.unitStore != nil {
			for _, unit := range a.unitStore.ListTenant(tenantSlug) {
				if unit.MiteigentumsanteilPPM <= 0 {
					continue
				}
				for _, email := range unit.OwnerEmails {
					if email = normalizeEmail(email); email != "" {
						weights[email] += unit.MiteigentumsanteilPPM
					}
				}
			}
		}
	}
	return weights
}

func ballotStatusForView(item ballot, now time.Time) (string, string, bool) {
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()
	switch normalizeBallotStatus(item.Status) {
	case ballotStatusOpen:
		if !item.OpensAt.IsZero() && now.Before(item.OpensAt) {
			return "Geplant", "status-progress", false
		}
		if !item.ClosesAt.IsZero() && !now.Before(item.ClosesAt) {
			return "Frist abgelaufen", "status-closed", false
		}
		return ballotStatusOpen, "status-open", true
	case ballotStatusClosed:
		return ballotStatusClosed, "status-closed", false
	default:
		return ballotStatusDraft, "status-progress", false
	}
}

func ballotReadOnlyMessage(role string, item ballot, eligible bool, active bool, hasVote bool) string {
	if !active {
		switch normalizeBallotStatus(item.Status) {
		case ballotStatusDraft:
			return "Noch nicht geöffnet."
		case ballotStatusClosed:
			return "Abstimmung geschlossen."
		default:
			if !item.OpensAt.IsZero() && time.Now().UTC().Before(item.OpensAt) {
				return "Noch nicht gestartet."
			}
			if !item.ClosesAt.IsZero() && !time.Now().UTC().Before(item.ClosesAt) {
				return "Frist abgelaufen."
			}
		}
	}
	if hasCapability(role, capabilityVote) {
		if eligible {
			if hasVote {
				return "Stimme abgegeben; bis zur Schließung änderbar."
			}
			return ""
		}
		return "Kein Stimmgewicht für diesen Zugang hinterlegt."
	}
	if hasCapability(role, capabilityOversight) {
		return "Beirat: lesende Übersicht."
	}
	return "Nur Eigentümer können abstimmen."
}

func voteMessage(status string) (string, bool) {
	switch status {
	case "created":
		return "Abstimmung angelegt.", true
	case "opened":
		return "Abstimmung geöffnet.", true
	case "closed":
		return "Abstimmung geschlossen.", true
	case "cast":
		return "Stimme gespeichert.", true
	case "missing":
		return "Abstimmung nicht gefunden.", false
	case "invalid":
		return "Bitte Abstimmung, Option und Berechtigung prüfen.", false
	default:
		return "", false
	}
}

func (a *app) startVoteReminderWorker() func() {
	if a.voteReminderInterval <= 0 || a.voteStore == nil {
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		a.sendDueBallotReminders(time.Now())
		ticker := time.NewTicker(a.voteReminderInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.sendDueBallotReminders(time.Now())
			}
		}
	}()
	return cancel
}

func (a *app) sendDueBallotReminders(now time.Time) int {
	if a == nil || a.voteStore == nil {
		return 0
	}
	if now.IsZero() {
		now = time.Now()
	}
	sentTotal := 0
	for _, tenant := range a.tenants {
		if _, err := a.voteStore.CloseExpiredTenant(tenant.Slug, now); err != nil {
			logError("ballot auto-close failed", err, "tenant", tenant.Slug)
		}
		for _, item := range a.voteStore.ListTenant(tenant.Slug) {
			if !ballotReminderDue(item, now) {
				continue
			}
			recipients := a.ballotReminderRecipients(tenant.Slug, item)
			if len(recipients) == 0 {
				continue
			}
			event := portalNotification{
				Event:      notificationEventVote,
				Tenant:     tenant,
				Recipients: recipients,
				Subject:    "Erinnerung: Abstimmung " + item.Title,
				ActionText: "Abstimmung öffnen",
				ActionURL:  tenant.PublicURL("/app/abstimmungen#ballot-" + url.PathEscape(item.ID)),
				Lines: []string{
					"Für " + tenant.Address + " läuft eine Abstimmung demnächst ab.",
					"",
					item.Title,
					"Frist: " + formatLocalDateTime(item.ClosesAt),
				},
			}
			sent := a.notify(event)
			if len(sent) == 0 {
				continue
			}
			if _, _, err := a.voteStore.MarkReminderSent(tenant.Slug, item.ID, sent, now); err != nil {
				logError("ballot reminder mark failed", err, "tenant", tenant.Slug, "ballot_id", item.ID)
				continue
			}
			a.recordAudit(auditEvent{
				TenantSlug: tenant.Slug,
				ActorEmail: "system",
				ActorRole:  "System",
				Action:     auditActionVoteReminder,
				TargetType: "ballot",
				TargetID:   item.ID,
				Summary:    "Abstimmungs-Erinnerung gesendet",
				Details: map[string]string{
					"title":      item.Title,
					"recipients": strconv.Itoa(len(sent)),
					"deadline":   formatLocalDateTime(item.ClosesAt),
				},
			})
			sentTotal += len(sent)
		}
	}
	return sentTotal
}

func ballotReminderDue(item ballot, now time.Time) bool {
	item = normalizeBallot(item)
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()
	if item.Status != ballotStatusOpen || item.ClosesAt.IsZero() || !now.Before(item.ClosesAt) {
		return false
	}
	return item.ClosesAt.Sub(now) <= time.Duration(item.ReminderBeforeMinutes)*time.Minute
}

func (a *app) ballotReminderRecipients(tenantSlug string, item ballot) []string {
	weights := a.ballotEligibleWeights(tenantSlug, item)
	recipients := make([]string, 0, len(weights))
	for email := range weights {
		if _, voted := item.Votes[email]; voted {
			continue
		}
		if _, reminded := item.ReminderSentAt[email]; reminded {
			continue
		}
		recipients = append(recipients, email)
	}
	sort.Strings(recipients)
	return recipients
}

// StartVoteReminderWorker starts the ballot-reminder worker; the returned func
// stops it.
func (a *app) StartVoteReminderWorker() func() { return a.startVoteReminderWorker() }

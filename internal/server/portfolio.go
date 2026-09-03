package server

import (
	"bytes"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/view"
	"github.com/inspr-at/hausv-org/internal/web"
)

type portfolioHouseInput struct {
	Slug        string
	Role        string
	Name        string
	Address     string
	UnitCount   int
	Issues      []store.ResidentIssue
	Events      []store.HouseEvent
	AuditEvents []store.AuditEvent
	PeopleNames map[string]string
}

func (a *app) portfolioPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	shell := a.verwaltungShell(&ac, "portfolio")
	profile := a.profileForTenant(ac.email, ac.tenant.Slug)
	managed := a.managedTenants(&ac)
	houses := make([]portfolioHouseInput, 0, len(managed))
	for _, tenant := range managed {
		repositories := a.repositoriesFor(tenant.Ref)
		house := portfolioHouseInput{
			Slug:        tenant.Config.Slug,
			Role:        tenant.Role,
			Name:        houseDisplayName(tenant.Config),
			Address:     tenant.Config.Address,
			PeopleNames: map[string]string{},
		}
		if repositories.issues != nil {
			house.Issues = repositories.issues.List()
			for _, issue := range house.Issues {
				addPortfolioPersonName(a, tenant.Config.Slug, house.PeopleNames, issue.AssigneeEmail)
			}
		}
		if repositories.events != nil {
			house.Events = repositories.events.List()
		}
		if repositories.units != nil {
			house.UnitCount = repositories.units.UnitCount()
		}
		if a.auditStore != nil {
			house.AuditEvents = a.auditStore.List(auditFilter{TenantSlug: tenant.Ref.Slug, Limit: 6})
			for _, event := range house.AuditEvents {
				addPortfolioPersonName(a, tenant.Config.Slug, house.PeopleNames, event.ActorEmail)
			}
		}
		houses = append(houses, house)
	}

	firstName := strings.TrimSpace(profile.FirstName)
	if firstName == "" {
		firstName = profile.DisplayName()
	}
	data := buildPortfolio(time.Now(), shell.OrganisationName, firstName, r.URL.Query().Get("sort"), houses)
	var rendered bytes.Buffer
	if err := web.PortfolioPage(shell, data).Render(r.Context(), &rendered); err != nil {
		logError("templ portfolio render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(prefixTenantHTMLPaths(rendered.String(), ac.tenant.Slug)))
}

func addPortfolioPersonName(a *app, tenantSlug string, names map[string]string, email string) {
	email = normalizeEmail(email)
	if email == "" {
		return
	}
	profile := a.profileForTenant(email, tenantSlug)
	name := strings.TrimSpace(profile.DisplayName())
	if name == "" {
		name = email
	}
	names[email] = name
}

// buildPortfolio converts already tenant-scoped data into the complete page
// model. Keeping repository access outside makes ranking and KPI policy
// deterministic and independently testable.
func buildPortfolio(now time.Time, organisationName, firstName, sortMode string, houses []portfolioHouseInput) web.PortfolioData {
	nowLocal := now.In(time.Local)
	sortMode = strings.ToLower(strings.TrimSpace(sortMode))
	if sortMode != "name" {
		sortMode = "need"
	}
	data := web.PortfolioData{
		OrganisationName: strings.TrimSpace(organisationName),
		Eyebrow:          strings.ToUpper(strings.TrimSpace(organisationName)) + " · PORTFOLIO",
		Greeting:         portfolioGreeting(nowLocal, firstName),
		Today:            view.GermanDateLong(nowLocal),
		Sort:             sortMode,
	}
	type pendingAudit struct {
		event store.AuditEvent
		house string
		actor string
	}
	pendingAudits := make([]pendingAudit, 0, len(houses)*6)

	for _, house := range houses {
		row := web.PortfolioHouse{
			Slug:        house.Slug,
			Role:        house.Role,
			Name:        house.Name,
			Address:     house.Address,
			Assignee:    "—",
			Oldest:      "—",
			NextDate:    "—",
			NextTitle:   "—",
			ActionClass: "green",
		}
		data.HouseCount++
		data.UnitCount += house.UnitCount
		oldestDays := -1
		assignees := map[string]int{}
		for _, issue := range house.Issues {
			if !portfolioIssueIsOpen(issue) {
				continue
			}
			row.Open++
			data.OpenIssues++
			ageDays := portfolioAgeDays(now, issue.CreatedAt)
			if ageDays > oldestDays {
				oldestDays = ageDays
			}
			if portfolioIssueOverdue(now, issue) {
				row.Overdue++
				data.Overdue++
			}
			if portfolioSameDate(issue.DueAt, nowLocal) {
				row.DueToday++
				data.DueToday++
			}
			if !issue.CreatedAt.IsZero() && !issue.CreatedAt.After(now) && !issue.CreatedAt.Before(now.Add(-24*time.Hour)) {
				data.NewSinceYesterday++
			}
			priority := store.NormalizeIssuePriority(issue.Priority)
			if priority == store.IssuePriorityHigh || priority == store.IssuePriorityUrgent {
				row.HighPriority++
			}
			if assignee := normalizeEmail(issue.AssigneeEmail); assignee != "" {
				assignees[assignee]++
			}
		}
		row.Score = row.Overdue*3 + row.HighPriority*2 + row.Open + row.DueToday*2
		if row.Open > 0 {
			row.ActionClass = "gold"
			data.ActionHouseCount++
			if oldestDays >= 0 {
				row.Oldest = portfolioDayCount(oldestDays)
			}
			row.Assignee = portfolioLeadingAssignee(assignees, house.PeopleNames)
		} else {
			data.QuietHouseCount++
		}
		if next, ok := portfolioNextEvent(now, house.Events); ok {
			row.NextDate = next.StartsAt.In(time.Local).Format("02.01.")
			row.NextTitle = next.Title
		}
		if row.Open > 0 {
			data.Houses = append(data.Houses, row)
		} else {
			data.QuietHouses = append(data.QuietHouses, row)
		}

		for _, event := range house.Events {
			start := event.StartsAt.In(time.Local)
			if !portfolioTodayOrTomorrow(nowLocal, start) {
				continue
			}
			data.Appointments = append(data.Appointments, web.PortfolioAppointment{
				House: house.Name,
				Title: event.Title,
				Day:   start.Format("02"),
				Month: view.GermanMonthShort(start),
				Time:  start.Format("15:04"),
				At:    event.StartsAt,
			})
		}
		for _, event := range house.AuditEvents {
			actor := strings.TrimSpace(house.PeopleNames[normalizeEmail(event.ActorEmail)])
			if actor == "" {
				actor = strings.TrimSpace(event.ActorEmail)
			}
			if actor == "" {
				actor = "System"
			}
			pendingAudits = append(pendingAudits, pendingAudit{event: event, house: house.Name, actor: actor})
		}
	}

	lessByName := func(left, right web.PortfolioHouse) bool {
		leftName, rightName := strings.ToLower(left.Name), strings.ToLower(right.Name)
		if leftName == rightName {
			return left.Slug < right.Slug
		}
		return leftName < rightName
	}
	sort.SliceStable(data.Houses, func(i, j int) bool {
		if sortMode != "name" && data.Houses[i].Score != data.Houses[j].Score {
			return data.Houses[i].Score > data.Houses[j].Score
		}
		return lessByName(data.Houses[i], data.Houses[j])
	})
	sort.SliceStable(data.QuietHouses, func(i, j int) bool { return lessByName(data.QuietHouses[i], data.QuietHouses[j]) })
	sort.SliceStable(data.Appointments, func(i, j int) bool {
		if !data.Appointments[i].At.Equal(data.Appointments[j].At) {
			return data.Appointments[i].At.Before(data.Appointments[j].At)
		}
		return strings.ToLower(data.Appointments[i].House) < strings.ToLower(data.Appointments[j].House)
	})
	sort.SliceStable(pendingAudits, func(i, j int) bool { return pendingAudits[i].event.At.After(pendingAudits[j].event.At) })
	for _, item := range pendingAudits[:min(6, len(pendingAudits))] {
		data.AuditEvents = append(data.AuditEvents, web.PortfolioAuditEvent{
			Actor:    item.actor,
			Action:   view.AuditActionLabel(item.event.Action),
			House:    item.house,
			Relative: portfolioRelativeTime(now, item.event.At),
			At:       item.event.At,
		})
	}
	data.TodayLine = fmt.Sprintf("%s · %s · %s", data.Today, portfolioHouseCount(data.HouseCount), portfolioUnitCount(data.UnitCount))
	return data
}

func portfolioGreeting(now time.Time, firstName string) string {
	greeting := "Guten Abend"
	if now.Hour() <= 11 {
		greeting = "Guten Morgen"
	} else if now.Hour() <= 17 {
		greeting = "Guten Tag"
	}
	firstName = strings.TrimSpace(firstName)
	if firstName == "" {
		return greeting + "."
	}
	return greeting + ", " + firstName + "."
}

func portfolioIssueIsOpen(issue store.ResidentIssue) bool {
	switch store.NormalizeIssueStatus(issue.Status) {
	case store.IssueStatusNew, store.IssueStatusAccepted, store.IssueStatusScheduled, store.IssueStatusProgress:
		return true
	default:
		return false
	}
}

func portfolioIssueOverdue(now time.Time, issue store.ResidentIssue) bool {
	if !issue.DueAt.IsZero() {
		return issue.DueAt.Before(now)
	}
	if issue.CreatedAt.IsZero() || issue.CreatedAt.After(now) {
		return false
	}
	limit := 14 * 24 * time.Hour
	priority := store.NormalizeIssuePriority(issue.Priority)
	if priority == store.IssuePriorityHigh || priority == store.IssuePriorityUrgent {
		limit = 7 * 24 * time.Hour
	}
	return now.Sub(issue.CreatedAt) > limit
}

func portfolioAgeDays(now, created time.Time) int {
	if created.IsZero() || created.After(now) {
		return 0
	}
	return int(now.Sub(created) / (24 * time.Hour))
}

func portfolioSameDate(value, reference time.Time) bool {
	if value.IsZero() {
		return false
	}
	value = value.In(time.Local)
	reference = reference.In(time.Local)
	vy, vm, vd := value.Date()
	ry, rm, rd := reference.Date()
	return vy == ry && vm == rm && vd == rd
}

func portfolioTodayOrTomorrow(now, event time.Time) bool {
	year, month, day := now.In(time.Local).Date()
	start := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	event = event.In(time.Local)
	return !event.Before(start) && event.Before(start.AddDate(0, 0, 2))
}

func portfolioNextEvent(now time.Time, events []store.HouseEvent) (store.HouseEvent, bool) {
	var next store.HouseEvent
	found := false
	for _, event := range events {
		if !store.EventRollsOffAt(event).After(now) {
			continue
		}
		if !found || event.StartsAt.Before(next.StartsAt) || (event.StartsAt.Equal(next.StartsAt) && strings.ToLower(event.Title) < strings.ToLower(next.Title)) {
			next = event
			found = true
		}
	}
	return next, found
}

func portfolioLeadingAssignee(counts map[string]int, names map[string]string) string {
	bestEmail, bestName, bestCount := "", "", 0
	for email, count := range counts {
		name := strings.TrimSpace(names[email])
		if name == "" {
			name = email
		}
		if count > bestCount || (count == bestCount && strings.ToLower(name) < strings.ToLower(bestName)) {
			bestEmail, bestName, bestCount = email, name, count
		}
	}
	if bestEmail == "" {
		return "—"
	}
	return bestName
}

func portfolioRelativeTime(now, at time.Time) string {
	if at.IsZero() {
		return "—"
	}
	age := now.Sub(at)
	if age < time.Minute {
		return "gerade eben"
	}
	if age < time.Hour {
		return fmt.Sprintf("vor %d Min.", int(age/time.Minute))
	}
	if age < 24*time.Hour {
		return fmt.Sprintf("vor %d Std.", int(age/time.Hour))
	}
	days := int(age / (24 * time.Hour))
	return "vor " + portfolioDayCount(days)
}

func portfolioDayCount(days int) string {
	if days == 1 {
		return "1 Tag"
	}
	return fmt.Sprintf("%d Tage", days)
}

func portfolioHouseCount(count int) string {
	if count == 1 {
		return "1 Haus"
	}
	return fmt.Sprintf("%d Häuser", count)
}

func portfolioUnitCount(count int) string {
	if count == 1 {
		return "1 Einheit"
	}
	return fmt.Sprintf("%d Einheiten", count)
}

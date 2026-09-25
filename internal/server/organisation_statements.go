package server

import (
	"bytes"
	"net/http"
	"strconv"

	"github.com/inspr-at/hausv-org/internal/web"
)

func (a *app) organisationStatementsPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	access := a.organisationAccessFor(&ac)
	rows := make([]web.OrganisationStatementRow, 0, len(access.Houses))
	for _, house := range access.Houses {
		if !access.managesHouse(house.Ref.Slug) {
			continue
		}
		repos := a.repositoriesFor(house.Ref)
		year, runID, detail := organisationStatementTarget(repos)
		openPath := "/app/settings/annual-statement"
		if year > 0 {
			openPath += "?year=" + strconv.Itoa(year)
			if runID != "" {
				openPath += "&run=" + runID
			}
		}
		path, foreign := OrganisationHousePath(ac.tenant.Slug, house.Config.Slug, openPath)
		rows = append(rows, web.OrganisationStatementRow{
			Name: houseDisplayName(house.Config), Address: house.Config.Address,
			Tenant: house.Config.Slug, Role: house.Role, OpenPath: path, Foreign: foreign, Detail: detail,
		})
	}
	var rendered bytes.Buffer
	if err := web.OrganisationStatementsPage(a.verwaltungShell(r.Context(), &ac, "statements"), rowsPage(rows)).Render(r.Context(), &rendered); err != nil {
		logError("organisation statements render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(prefixTenantHTMLPaths(rendered.String(), ac.tenant.Slug)))
}

func rowsPage(rows []web.OrganisationStatementRow) web.OrganisationStatementsData {
	return web.OrganisationStatementsData{Rows: rows}
}

func organisationStatementTarget(repos requestRepositories) (year int, runID, detail string) {
	detail = "Jahresabrechnung öffnen"
	if repos.annualStatementPeriods != nil {
		for _, period := range repos.annualStatementPeriods.List() {
			if period.Year > year {
				year = period.Year
			}
		}
	}
	if year == 0 || repos.annualStatementRuns == nil {
		return year, "", detail
	}
	runs, err := repos.annualStatementRuns.List(year)
	if err != nil || len(runs) == 0 {
		return year, "", detail
	}
	run := runs[0]
	for _, candidate := range runs[1:] {
		if candidate.Revision > run.Revision {
			run = candidate
		}
	}
	return year, run.ID, "Abrechnung " + strconv.Itoa(year)
}

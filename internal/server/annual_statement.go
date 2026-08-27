package server

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

const maxAnnualStatementPartyImportBytes = 1 << 20

func (a *app) annualStatementPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, _, _, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if ac.repositories.annualStatementPeriods == nil || ac.repositories.units == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	periods := ac.repositories.annualStatementPeriods.List()
	selectedYear, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("year")))
	if selectedYear == 0 && len(periods) > 0 {
		selectedYear = periods[0].Year
	}
	formYear := selectedYear
	if formYear == 0 {
		formYear = time.Now().Year()
	}
	startsOn, endsOn := "", ""
	periodViews := make([]web.AnnualStatementPeriodView, 0, len(periods))
	for _, period := range periods {
		selected := period.Year == selectedYear
		if selected {
			startsOn, endsOn = period.StartsOn, period.EndsOn
		}
		periodViews = append(periodViews, web.AnnualStatementPeriodView{
			Year: period.Year, StartsOn: period.StartsOn, EndsOn: period.EndsOn,
			DateRange: annualStatementDateRange(period.StartsOn, period.EndsOn), Selected: selected,
		})
	}
	units := ac.repositories.units.List()
	unitViews := make([]web.AnnualStatementUnitView, 0, len(units))
	for _, unit := range units {
		unitViews = append(unitViews, web.AnnualStatementUnitView{
			ID: unit.ID, Label: unit.Label, TypeLabel: unitTypeLabel(unit.UnitType),
			OwnerSummary:  annualStatementPartySummary(unit.OwnerEmails, "Nicht zugeordnet"),
			RenterSummary: annualStatementPartySummary(unit.RenterEmails, "Kein Mietverhältnis hinterlegt"),
		})
	}
	periodMsg, periodOK := annualStatementPeriodMessage(r.URL.Query().Get("period"))
	importMsg, importOK := annualStatementImportMessage(r.URL.Query().Get("import"), r.URL.Query().Get("count"))
	a.renderSettingsComponent(w, r, tenant.Slug, web.AnnualStatementPage(web.AnnualStatementPageData{
		Portal:     a.settingsPortalContext(ac, "Jahresabrechnung", "settings"),
		EstateName: tenant.Name, EstateAddress: tenant.Address,
		Periods: periodViews, HasPeriods: len(periodViews) > 0,
		Year: formYear, StartsOn: startsOn, EndsOn: endsOn,
		PeriodMsg: periodMsg, PeriodOK: periodOK, ImportMsg: importMsg, ImportOK: importOK,
		Units: unitViews, HasUnits: len(unitViews) > 0,
	}))
}

func (a *app) saveAnnualStatementPeriod(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	year, err := strconv.Atoi(strings.TrimSpace(r.FormValue("year")))
	period := store.AnnualStatementPeriod{
		Year: year, StartsOn: r.FormValue("starts_on"), EndsOn: r.FormValue("ends_on"),
		UpdatedAt: time.Now().UTC(), UpdatedBy: actorEmail,
	}
	if err != nil || ac.repositories.annualStatementPeriods == nil {
		http.Redirect(w, r, "/app/settings/annual-statement?period=invalid", http.StatusSeeOther)
		return
	}
	if _, err := ac.repositories.annualStatementPeriods.Save(period); err != nil {
		http.Redirect(w, r, "/app/settings/annual-statement?period=invalid", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug, ActorEmail: actorEmail, ActorRole: role,
		Action: auditActionAnnualPeriodSave, TargetType: "annual-statement-period", TargetID: strconv.Itoa(year),
		Summary: "Abrechnungsperiode gespeichert",
		Details: map[string]string{"year": strconv.Itoa(year), "starts_on": period.StartsOn, "ends_on": period.EndsOn},
	})
	http.Redirect(w, r, "/app/settings/annual-statement?year="+strconv.Itoa(year)+"&period=saved", http.StatusSeeOther)
}

func (a *app) importAnnualStatementParties(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAnnualStatementPartyImportBytes+(64<<10))
	if ac.repositories.units == nil || r.ParseMultipartForm(maxAnnualStatementPartyImportBytes) != nil {
		http.Redirect(w, r, "/app/settings/annual-statement?import=invalid", http.StatusSeeOther)
		return
	}
	file, header, err := r.FormFile("parties_file")
	if err != nil || header == nil || header.Size <= 0 || header.Size > maxAnnualStatementPartyImportBytes || strings.ToLower(filepath.Ext(header.Filename)) != ".csv" {
		http.Redirect(w, r, "/app/settings/annual-statement?import=invalid", http.StatusSeeOther)
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxAnnualStatementPartyImportBytes+1))
	if err != nil || len(raw) > maxAnnualStatementPartyImportBytes {
		http.Redirect(w, r, "/app/settings/annual-statement?import=invalid", http.StatusSeeOther)
		return
	}
	assignments, err := parseAnnualStatementPartyCSV(raw)
	if err != nil {
		http.Redirect(w, r, "/app/settings/annual-statement?import=invalid", http.StatusSeeOther)
		return
	}
	units := ac.repositories.units.List()
	updates, assigned, err := annualStatementPartyUpdates(units, assignments)
	if err != nil {
		http.Redirect(w, r, "/app/settings/annual-statement?import=unknown-unit", http.StatusSeeOther)
		return
	}
	unknownUnit, err := ac.repositories.units.UpdateParties(updates)
	if unknownUnit {
		http.Redirect(w, r, "/app/settings/annual-statement?import=unknown-unit", http.StatusSeeOther)
		return
	}
	if err != nil {
		logError("annual statement party import failed", err, "tenant", tenant.Slug)
		http.Redirect(w, r, "/app/settings/annual-statement?import=error", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug, ActorEmail: actorEmail, ActorRole: role,
		Action: auditActionAnnualPartiesImport, TargetType: "annual-statement-parties", TargetID: tenant.Slug,
		Summary: "Parteien aus Tabelle übernommen", Details: map[string]string{"assignments": strconv.Itoa(assigned)},
	})
	http.Redirect(w, r, "/app/settings/annual-statement?import=saved&count="+strconv.Itoa(assigned), http.StatusSeeOther)
}

type annualStatementPartyAssignment struct {
	unit  string
	role  string
	email string
}

func parseAnnualStatementPartyCSV(raw []byte) ([]annualStatementPartyAssignment, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty csv")
	}
	reader := csv.NewReader(bytes.NewReader(raw))
	if firstLine := strings.SplitN(string(raw), "\n", 2)[0]; strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		reader.Comma = ';'
	}
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil || len(rows) < 2 {
		return nil, fmt.Errorf("invalid csv")
	}
	columns := map[string]int{}
	for index, value := range rows[0] {
		columns[strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "\ufeff")))] = index
	}
	unitColumn, unitOK := columns["einheit"]
	roleColumn, roleOK := columns["rolle"]
	emailColumn, emailOK := columns["e-mail"]
	if !emailOK {
		emailColumn, emailOK = columns["email"]
	}
	if !unitOK || !roleOK || !emailOK {
		return nil, fmt.Errorf("missing columns")
	}
	maxColumn := max(unitColumn, roleColumn, emailColumn)
	assignments := []annualStatementPartyAssignment{}
	for _, row := range rows[1:] {
		if len(row) <= maxColumn {
			return nil, fmt.Errorf("short row")
		}
		unitValue := strings.TrimSpace(row[unitColumn])
		roleValue := strings.ToLower(strings.TrimSpace(row[roleColumn]))
		rawEmail := strings.TrimSpace(row[emailColumn])
		if unitValue == "" && roleValue == "" && rawEmail == "" {
			continue
		}
		parsedEmail, err := mail.ParseAddress(rawEmail)
		if err != nil || normalizeEmail(parsedEmail.Address) == "" {
			return nil, fmt.Errorf("invalid email")
		}
		normalizedRole := normalizeRole(roleValue)
		switch roleValue {
		case "mietverhältnis", "mietverhaeltnis":
			normalizedRole = roleRenter
		}
		if normalizedRole != roleOwner && normalizedRole != roleRenter {
			return nil, fmt.Errorf("invalid role")
		}
		assignments = append(assignments, annualStatementPartyAssignment{unit: unitValue, role: normalizedRole, email: normalizeEmail(parsedEmail.Address)})
	}
	if len(assignments) == 0 {
		return nil, fmt.Errorf("no assignments")
	}
	return assignments, nil
}

func annualStatementPartyUpdates(units []unit, assignments []annualStatementPartyAssignment) ([]store.UnitPartyUpdate, int, error) {
	byID := map[string]int{}
	byLabel := map[string]int{}
	for index, item := range units {
		byID[normalizeUnitID(item.ID)] = index
		labelKey := normalizeUnitID(item.Label)
		if prior, exists := byLabel[labelKey]; !exists || prior == index {
			byLabel[labelKey] = index
		} else {
			byLabel[labelKey] = -1
		}
	}
	type importedParties struct {
		owners, renters       []string
		hasOwners, hasRenters bool
	}
	imports := map[int]importedParties{}
	for _, assignment := range assignments {
		key := normalizeUnitID(assignment.unit)
		index, found := byID[key]
		if !found {
			index, found = byLabel[key]
		}
		if !found || index < 0 {
			return nil, 0, fmt.Errorf("unknown unit")
		}
		parties := imports[index]
		if assignment.role == roleOwner {
			parties.hasOwners = true
			parties.owners = append(parties.owners, assignment.email)
		} else {
			parties.hasRenters = true
			parties.renters = append(parties.renters, assignment.email)
		}
		imports[index] = parties
	}
	updates := make([]store.UnitPartyUpdate, 0, len(imports))
	for index, parties := range imports {
		updates = append(updates, store.UnitPartyUpdate{
			UnitID: units[index].ID, OwnerEmails: parties.owners, RenterEmails: parties.renters,
			SetOwners: parties.hasOwners, SetRenters: parties.hasRenters,
		})
	}
	return updates, len(assignments), nil
}

func annualStatementPartySummary(emails []string, empty string) string {
	if len(emails) == 0 {
		return empty
	}
	return strings.Join(emails, ", ")
}

func annualStatementDateRange(startsOn, endsOn string) string {
	format := func(raw string) string {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return raw
		}
		return parsed.Format("02.01.2006")
	}
	return format(startsOn) + " – " + format(endsOn)
}

func annualStatementPeriodMessage(status string) (string, bool) {
	switch status {
	case "saved":
		return "Die Abrechnungsperiode wurde gespeichert.", true
	case "invalid":
		return "Bitte Abrechnungsjahr und Zeitraum vollständig und chronologisch eingeben.", false
	default:
		return "", false
	}
}

func annualStatementImportMessage(status, count string) (string, bool) {
	switch status {
	case "saved":
		return count + " Zuordnungen wurden übernommen und können in weiteren Abrechnungsjahren wiederverwendet werden.", true
	case "unknown-unit":
		return "Der Import wurde nicht übernommen: Mindestens eine Einheit ist nicht vorhanden. Es wurden keine Einheiten angelegt.", false
	case "invalid":
		return "Die CSV-Datei konnte nicht übernommen werden. Prüfen Sie die Spalten Einheit, Rolle und E-Mail.", false
	case "error":
		return "Die Zuordnungen konnten nicht gespeichert werden.", false
	default:
		return "", false
	}
}

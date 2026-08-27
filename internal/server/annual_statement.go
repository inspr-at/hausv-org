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
	a.renderAnnualStatementPage(w, r, ac, web.AnnualStatementReceiptSuggestionView{}, "", false)
}

func (a *app) renderAnnualStatementPage(w http.ResponseWriter, r *http.Request, ac authCtx, receiptSuggestion web.AnnualStatementReceiptSuggestionView, receiptMsg string, receiptOK bool) {
	tenant, actorEmail, _, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if ac.repositories.annualStatementCostTypes == nil || ac.repositories.annualStatementPeriods == nil || ac.repositories.units == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := ac.repositories.annualStatementCostTypes.EnsureDefaults(actorEmail); err != nil {
		logError("annual statement cost type defaults failed", err, "tenant", tenant.Slug)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	costTypes := ac.repositories.annualStatementCostTypes.List()
	costTypeViews := make([]web.AnnualStatementCostTypeView, 0, len(costTypes))
	allocatableCount := 0
	for _, costType := range costTypes {
		if costType.Allocatable {
			allocatableCount++
		}
		costTypeViews = append(costTypeViews, web.AnnualStatementCostTypeView{
			Key: costType.Key, Name: costType.Name, Allocatable: costType.Allocatable,
			AllocationKey: costType.AllocationKey, AllocationKeyLabel: annualStatementAllocationKeyLabel(costType.AllocationKey),
		})
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
	allocation := annualStatementAllocationView(costTypes, units)
	basesMsg, basesOK := annualStatementBasesMessage(r.URL.Query().Get("bases"))
	periodMsg, periodOK := annualStatementPeriodMessage(r.URL.Query().Get("period"))
	importMsg, importOK := annualStatementImportMessage(r.URL.Query().Get("import"), r.URL.Query().Get("count"))
	costTypeMsg, costTypeOK := annualStatementCostTypeMessage(r.URL.Query().Get("cost-type"))
	receiptDocuments := []web.AnnualStatementReceiptDocumentView{}
	if ac.repositories.documents != nil {
		for _, document := range ac.repositories.documents.ListCurrent() {
			if annualStatementReceiptContentTypeSupported(document.ContentType) {
				receiptDocuments = append(receiptDocuments, web.AnnualStatementReceiptDocumentView{ID: document.ID, Label: document.Title + " · " + document.Filename})
			}
		}
	}
	if receiptMsg == "" {
		receiptMsg, receiptOK = annualStatementReceiptMessage(r.URL.Query().Get("receipt"))
	}
	a.renderSettingsComponent(w, r, tenant.Slug, web.AnnualStatementPage(web.AnnualStatementPageData{
		Portal:     a.settingsPortalContext(ac, "Jahresabrechnung", "settings"),
		EstateName: tenant.Name, EstateAddress: tenant.Address,
		Periods: periodViews, HasPeriods: len(periodViews) > 0,
		Year: formYear, StartsOn: startsOn, EndsOn: endsOn,
		PeriodMsg: periodMsg, PeriodOK: periodOK, ImportMsg: importMsg, ImportOK: importOK,
		CostTypes: costTypeViews, CostTypeCount: len(costTypeViews), AllocatableCostTypeCount: allocatableCount,
		CostTypeMsg: costTypeMsg, CostTypeOK: costTypeOK,
		ReceiptDocuments: receiptDocuments, HasReceiptDocuments: len(receiptDocuments) > 0,
		ReceiptSuggesterReady: a.annualStatementReceiptSuggester != nil,
		ReceiptMsg:            receiptMsg, ReceiptOK: receiptOK,
		ReceiptSuggestion: receiptSuggestion, HasReceiptSuggestion: receiptSuggestion.DocumentID != "",
		Units: unitViews, HasUnits: len(unitViews) > 0,
		Allocation: allocation, BasesMsg: basesMsg, BasesOK: basesOK,
	}))
}

func (a *app) saveAnnualStatementCostType(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	allocation := strings.TrimSpace(r.FormValue("allocation"))
	if ac.repositories.annualStatementCostTypes == nil || (allocation != "allocatable" && allocation != "not_allocatable") {
		http.Redirect(w, r, "/app/settings/annual-statement?cost-type=invalid", http.StatusSeeOther)
		return
	}
	// The store rejects an allocatable cost type without a valid key, so a
	// missing or tampered allocation_key lands on the same invalid redirect.
	costType := store.AnnualStatementCostType{
		Key: r.FormValue("key"), Name: r.FormValue("name"), Allocatable: allocation == "allocatable",
		AllocationKey: r.FormValue("allocation_key"),
		UpdatedAt:     time.Now().UTC(), UpdatedBy: actorEmail,
	}
	saved, err := ac.repositories.annualStatementCostTypes.Save(costType)
	if err != nil {
		http.Redirect(w, r, "/app/settings/annual-statement?cost-type=invalid", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug, ActorEmail: actorEmail, ActorRole: role,
		Action: auditActionAnnualCostTypeSave, TargetType: "annual-statement-cost-type", TargetID: saved.Key,
		Summary: "Kostenart gespeichert",
		Details: map[string]string{"key": saved.Key, "name": saved.Name, "allocatable": strconv.FormatBool(saved.Allocatable), "allocation_key": saved.AllocationKey},
	})
	http.Redirect(w, r, "/app/settings/annual-statement?cost-type=saved", http.StatusSeeOther)
}

// saveAnnualStatementAllocationBases records Nutzfläche and Personen for every
// unit in one submit. Nutzwert stays on the unit record itself and is edited in
// the building settings; nothing here invents a Miteigentumsanteil.
func (a *app) saveAnnualStatementAllocationBases(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil || ac.repositories.units == nil {
		http.Redirect(w, r, "/app/settings/annual-statement?bases=invalid", http.StatusSeeOther)
		return
	}
	updates := make([]store.UnitAllocationBasisUpdate, 0, len(r.Form["unit_id"]))
	for index, unitID := range r.Form["unit_id"] {
		area, areaOK := parseAnnualStatementArea(formValueAt(r.Form["usable_area_m2"], index))
		persons, personsOK := parseAnnualStatementCount(formValueAt(r.Form["persons"], index))
		if !areaOK || !personsOK {
			http.Redirect(w, r, "/app/settings/annual-statement?bases=invalid", http.StatusSeeOther)
			return
		}
		updates = append(updates, store.UnitAllocationBasisUpdate{UnitID: unitID, UsableAreaM2Hundredths: area, Persons: persons})
	}
	if len(updates) == 0 {
		http.Redirect(w, r, "/app/settings/annual-statement?bases=invalid", http.StatusSeeOther)
		return
	}
	unknownUnit, err := ac.repositories.units.UpdateAllocationBases(updates)
	if unknownUnit {
		http.Redirect(w, r, "/app/settings/annual-statement?bases=unknown-unit", http.StatusSeeOther)
		return
	}
	if err != nil {
		logError("annual statement allocation bases save failed", err, "tenant", tenant.Slug)
		http.Redirect(w, r, "/app/settings/annual-statement?bases=error", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug, ActorEmail: actorEmail, ActorRole: role,
		Action: auditActionAnnualBasesSave, TargetType: "annual-statement-allocation-bases", TargetID: tenant.Slug,
		Summary: "Verteilerbasis je Einheit gespeichert", Details: map[string]string{"units": strconv.Itoa(len(updates))},
	})
	http.Redirect(w, r, "/app/settings/annual-statement?bases=saved#verteilerschluessel", http.StatusSeeOther)
}

func formValueAt(values []string, index int) string {
	if index < len(values) {
		return values[index]
	}
	return ""
}

// parseAnnualStatementArea accepts "72,5", "72.50" or "" (not recorded) and
// returns hundredths of a square metre. More than two decimals, negatives and
// implausible sizes are rejected rather than rounded.
func parseAnnualStatementArea(raw string) (int, bool) {
	raw = strings.ReplaceAll(strings.TrimSpace(raw), ",", ".")
	if raw == "" {
		return 0, true
	}
	whole, fraction, _ := strings.Cut(raw, ".")
	if whole == "" {
		whole = "0"
	}
	if len(fraction) > 2 {
		return 0, false
	}
	fraction += strings.Repeat("0", 2-len(fraction))
	metres, err := strconv.Atoi(whole)
	if err != nil || metres < 0 || metres > 100_000 {
		return 0, false
	}
	hundredths, err := strconv.Atoi(fraction)
	if err != nil || hundredths < 0 {
		return 0, false
	}
	return metres*100 + hundredths, true
}

func parseAnnualStatementCount(raw string) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, true
	}
	count, err := strconv.Atoi(raw)
	if err != nil || count < 0 || count > 10_000 {
		return 0, false
	}
	return count, true
}

func annualStatementAllocationKeyLabel(key string) string {
	switch key {
	case store.AllocationKeyNutzwert:
		return "Nutzwert"
	case store.AllocationKeyFlaeche:
		return "Nutzfläche"
	case store.AllocationKeyPersonen:
		return "Personen"
	case store.AllocationKeyVerbrauch:
		return "Verbrauch"
	default:
		return "Kein Schlüssel"
	}
}

func annualStatementAllocationKeyOptions() []web.AnnualStatementAllocationKeyOption {
	out := make([]web.AnnualStatementAllocationKeyOption, 0, len(store.AllocationKeys))
	for _, key := range store.AllocationKeys {
		out = append(out, web.AnnualStatementAllocationKeyOption{Key: key, Label: annualStatementAllocationKeyLabel(key)})
	}
	return out
}

func annualStatementAllocationView(costTypes []store.AnnualStatementCostType, units []store.Unit) web.AnnualStatementAllocationView {
	names := map[string]string{}
	for _, costType := range costTypes {
		names[costType.Key] = costType.Name
	}
	view := web.AnnualStatementAllocationView{KeyOptions: annualStatementAllocationKeyOptions()}
	for _, unit := range units {
		view.Bases = append(view.Bases, web.AnnualStatementUnitBasisView{
			ID: unit.ID, Label: unit.Label, Share: formatMiteigentumsanteil(unit.MiteigentumsanteilPPM),
			UsableArea: formatAnnualStatementArea(unit.UsableAreaM2Hundredths), Persons: formatAnnualStatementCount(unit.Persons),
		})
	}
	for _, preview := range store.AnnualStatementAllocationPreviews(costTypes, units) {
		previewView := web.AnnualStatementAllocationPreviewView{
			Key: preview.Key, Label: annualStatementAllocationKeyLabel(preview.Key), Blocked: preview.Blocked,
			UnmappedUnits: strings.Join(preview.UnmappedUnits, ", "), Sourceless: preview.Key == store.AllocationKeyVerbrauch,
		}
		costTypeNames := make([]string, 0, len(preview.CostTypeKeys))
		for _, key := range preview.CostTypeKeys {
			costTypeNames = append(costTypeNames, names[key])
		}
		previewView.CostTypes = strings.Join(costTypeNames, ", ")
		for _, share := range preview.Shares {
			previewView.Shares = append(previewView.Shares, web.AnnualStatementUnitShareView{
				Label: share.Label, Basis: formatAnnualStatementBasis(preview.Key, share.Basis), Share: formatAnnualStatementShare(share.SharePPM), Mapped: share.Mapped,
			})
		}
		view.Previews = append(view.Previews, previewView)
		if preview.Blocked {
			view.BlockedCount++
		}
	}
	view.RunReady = len(view.Previews) > 0 && view.BlockedCount == 0
	return view
}

func formatAnnualStatementArea(hundredths int) string {
	if hundredths <= 0 {
		return ""
	}
	return strings.Replace(fmt.Sprintf("%d.%02d", hundredths/100, hundredths%100), ".", ",", 1)
}

func formatAnnualStatementCount(count int) string {
	if count <= 0 {
		return ""
	}
	return strconv.Itoa(count)
}

func formatAnnualStatementBasis(key string, basis int) string {
	if basis <= 0 {
		return "fehlt"
	}
	switch key {
	case store.AllocationKeyNutzwert:
		return formatMiteigentumsanteil(basis)
	case store.AllocationKeyFlaeche:
		return formatAnnualStatementArea(basis) + " m²"
	case store.AllocationKeyPersonen:
		return strconv.Itoa(basis)
	default:
		return "fehlt"
	}
}

// formatAnnualStatementShare renders parts per million as a percentage with
// two decimals, e.g. 333334 → "33,33 %".
func formatAnnualStatementShare(ppm int) string {
	if ppm <= 0 {
		return "–"
	}
	return strings.Replace(fmt.Sprintf("%d.%02d %%", ppm/10_000, (ppm%10_000)/100), ".", ",", 1)
}

func annualStatementBasesMessage(status string) (string, bool) {
	switch strings.TrimSpace(status) {
	case "saved":
		return "Verteilerbasis je Einheit gespeichert.", true
	case "unknown-unit":
		return "Eine Einheit ist nicht mehr vorhanden. Es wurde nichts gespeichert.", false
	case "invalid":
		return "Nutzfläche (m², max. zwei Nachkommastellen) und Personen (ganze Zahl) prüfen. Es wurde nichts gespeichert.", false
	case "error":
		return "Die Verteilerbasis konnte nicht gespeichert werden.", false
	default:
		return "", false
	}
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

func annualStatementCostTypeMessage(status string) (string, bool) {
	switch status {
	case "saved":
		return "Die Kostenart wurde gespeichert.", true
	case "invalid":
		return "Bitte Bezeichnung, Schlüssel und Umlagefähigkeit vollständig angeben. Der Schlüssel darf Kleinbuchstaben, Ziffern, Bindestriche und Unterstriche enthalten.", false
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

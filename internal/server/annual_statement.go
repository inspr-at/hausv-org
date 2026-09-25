package server

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/view"
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
	if ac.repositories.annualStatementCostTypes == nil || ac.repositories.annualStatementPeriods == nil || ac.repositories.annualStatementAkontos == nil || ac.repositories.annualStatementReceipts == nil || ac.repositories.units == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	costTypes := ac.repositories.annualStatementCostTypes.List()
	if len(costTypes) == 0 {
		costTypes = store.AnnualStatementDefaultCostTypes(actorEmail)
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
	selectedPeriodFound := false
	var selectedPeriod store.AnnualStatementPeriod
	followup := web.AnnualStatementFollowupView{}
	periodViews := make([]web.AnnualStatementPeriodView, 0, len(periods))
	for _, period := range periods {
		selected := period.Year == selectedYear
		if selected {
			selectedPeriodFound = true
			selectedPeriod = period
			startsOn, endsOn = period.StartsOn, period.EndsOn
			if next, nextOK := annualStatementFollowupPeriod(period, actorEmail); nextOK {
				followup = web.AnnualStatementFollowupView{
					SourceYear: period.Year, Year: next.Year, DateRange: annualStatementDateRange(next.StartsOn, next.EndsOn),
					Deadline: annualStatementDeadline(next.EndsOn), Available: true,
				}
			}
		}
		periodLegal, _ := ac.repositories.annualStatementPeriods.Structure(period.Year)
		periodViews = append(periodViews, web.AnnualStatementPeriodView{
			Year: period.Year, StartsOn: period.StartsOn, EndsOn: period.EndsOn,
			DateRange: annualStatementDateRange(period.StartsOn, period.EndsOn), Deadline: annualLegalDeadline(period, periodLegal.Legal), Selected: selected,
		})
	}
	units, unitNotice := unitsForPage(ac.repositories.units)
	structureYear := 0
	var periodBases []store.AnnualStatementPeriodUnitBasis
	legal := store.DefaultAnnualStatementLegalSettings()
	if selectedPeriodFound {
		structureYear = selectedYear
		structure, found := ac.repositories.annualStatementPeriods.Structure(selectedYear)
		if !found {
			logError("annual statement period structure unavailable", fmt.Errorf("missing period snapshot"), "tenant", tenant.Slug, "year", selectedYear)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		legal = structure.Legal
		periodBases = structure.UnitBases
		for i := range periodViews {
			if periodViews[i].Selected {
				periodViews[i].Deadline = annualLegalDeadline(selectedPeriod, legal)
			}
		}
		if next, ok := annualStatementFollowupPeriod(selectedPeriod, actorEmail); ok {
			followup.Deadline = annualLegalDeadline(next, legal)
		}
		costTypes = structure.CostTypes
		units = annualStatementUnitsWithPeriodBases(units, structure.UnitBases)
	}
	costTypeViews := make([]web.AnnualStatementCostTypeView, 0, len(costTypes))
	allocatableCount := 0
	for _, costType := range costTypes {
		if costType.Allocatable {
			allocatableCount++
		}
		costTypeViews = append(costTypeViews, web.AnnualStatementCostTypeView{
			Key: costType.Key, Name: costType.Name, Allocatable: costType.Allocatable,
			AllocationKey: costType.AllocationKey, VATRatePercent: costType.VATRatePercent,
		})
	}
	unitViews := make([]web.AnnualStatementUnitView, 0, len(units))
	for _, unit := range units {
		unitViews = append(unitViews, web.AnnualStatementUnitView{
			ID: unit.ID, Label: unit.Label, TypeLabel: unitTypeLabel(unit.UnitType),
			OwnerSummary:  annualStatementPartySummary(unit.OwnerEmails, "Nicht zugeordnet"),
			RenterSummary: annualStatementPartySummary(unit.RenterEmails, "Kein Mietverhältnis hinterlegt"),
		})
	}
	consumption := annualStatementConsumption(ac.repositories.annualConsumption, selectedPeriod, units, costTypes)
	allocation := annualStatementAllocationView(costTypes, units, consumption, periodBases, legal.AgreedShares)
	basesMsg, basesOK := annualStatementBasesMessage(r.URL.Query().Get("bases"))
	periodMsg, periodOK := annualStatementPeriodMessage(r.URL.Query().Get("period"))
	importMsg, importOK := annualStatementImportMessage(r.URL.Query().Get("import"), r.URL.Query().Get("count"))
	costTypeMsg, costTypeOK := annualStatementCostTypeMessage(r.URL.Query().Get("cost-type"))
	receiptDocuments := []web.AnnualStatementReceiptDocumentView{}
	documentByID := map[string]store.DocumentRecord{}
	if ac.repositories.documents != nil {
		usedDocuments := map[string]bool{}
		for _, receipt := range ac.repositories.annualStatementReceipts.List() {
			usedDocuments[receipt.DocumentID] = true
		}
		for _, document := range ac.repositories.documents.List() {
			documentByID[document.ID] = document
			if document.Current && annualStatementReceiptContentTypeSupported(document.ContentType) && !usedDocuments[document.ID] {
				receiptDocuments = append(receiptDocuments, web.AnnualStatementReceiptDocumentView{ID: document.ID, Label: document.Title + " · " + document.Filename})
			}
		}
	}
	costTypeNames := map[string]string{}
	for _, costType := range costTypes {
		costTypeNames[costType.Key] = costType.Name
	}
	receiptViews := []web.AnnualStatementReceiptView{}
	selectedReceipts := []store.AnnualStatementReceipt{}
	if selectedYear != 0 {
		selectedReceipts = ac.repositories.annualStatementReceipts.ListByPeriod(selectedYear)
		for _, receipt := range selectedReceipts {
			documentTitle := "Dokument " + receipt.DocumentID
			if document, found := documentByID[receipt.DocumentID]; found {
				documentTitle = document.Title + " · " + document.Filename
			}
			invoiceDate := receipt.InvoiceDate
			if parsed, err := time.Parse("2006-01-02", receipt.InvoiceDate); err == nil {
				invoiceDate = parsed.Format("02.01.2006")
			}
			receiptViews = append(receiptViews, web.AnnualStatementReceiptView{
				ID: receipt.ID, DocumentTitle: documentTitle, Supplier: receipt.Supplier, HeatingCategory: receipt.HeatingCategory,
				Amount: formatAnnualStatementReceiptAmount(receipt.AmountCents), AmountValue: formatAnnualStatementReceiptAmountValue(receipt.AmountCents),
				InvoiceDate: invoiceDate, CostTypeName: costTypeNames[receipt.CostTypeKey],
			})
		}
	}
	if receiptMsg == "" {
		receiptMsg, receiptOK = annualStatementReceiptMessage(r.URL.Query().Get("receipt"))
	}
	prepayments := map[string]store.AnnualStatementPrepayment{}
	if selectedYear != 0 {
		for _, item := range ac.repositories.annualStatementAkontos.ListByPeriod(selectedYear) {
			prepayments[item.UnitID] = item
		}
	}
	settlement, settlementReady := store.AnnualStatementSettlementPreviewWithAgreed(costTypes, selectedReceipts, units, consumption.Vectors, legal.AgreedShares)
	if legal.HeizKGApplies && ac.repositories.annualStatementRuns != nil {
		_, result, err := ac.repositories.annualStatementRuns.Preview(selectedYear, consumption.Vectors)
		settlement = nil
		settlementReady = err == nil
		if err == nil {
			for _, u := range result.Units {
				settlement = append(settlement, store.AnnualStatementSettlementUnit{UnitID: u.UnitID, AllocatedCents: u.AllocatedCents})
			}
		}
	}
	allocatedByUnit := map[string]int64{}
	for _, item := range settlement {
		allocatedByUnit[item.UnitID] = item.AllocatedCents
	}
	prefill := annualStatementPrepaymentPrefill(ac.repositories.annualStatementRuns, selectedYear)
	prepaymentViews := make([]web.AnnualStatementPrepaymentView, 0, len(units))
	for _, unit := range units {
		item, recorded := prepayments[unit.ID]
		allocated := allocatedByUnit[unit.ID]
		view := web.AnnualStatementPrepaymentView{
			UnitID: unit.ID, UnitLabel: unit.Label, Recorded: recorded,
			Paid: "nicht erfasst", Allocated: formatAnnualStatementMoney(allocated),
			Balance: formatAnnualStatementBalance(item.AmountCents - allocated),
		}
		if recorded {
			view.AmountValue = formatAnnualStatementReceiptAmountValue(item.AmountCents)
			view.Paid = formatAnnualStatementMoney(item.AmountCents)
		}
		if amount, ok := prefill[unit.ID]; ok && !recorded {
			view.AmountValue = formatAnnualStatementReceiptAmountValue(amount)
			view.Prefilled = true
		}
		prepaymentViews = append(prepaymentViews, view)
	}
	prepaymentMsg, prepaymentOK := annualStatementPrepaymentMessage(r.URL.Query().Get("prepayment"))
	runView := a.annualStatementDeliveryView(annualStatementRunView(ac.repositories.annualStatementRuns, ac.repositories.documents, selectedYear, r.URL.Query().Get("run"), r.URL.Query().Get("run-status"), consumption.Vectors), ac.repositories, r.URL.Query())
	runView.CreatedBy = a.profileForTenant(runView.CreatedBy, ac.tenant.Slug).DisplayName()
	if runView.ID == "" {
		address := tenant.ContactAddress
		if org, found := a.organisationRecordFor(r.Context(), &ac); found {
			address = firstNonEmpty(org.ContactAddress, address)
		}
		runView.ManagementAddressMissing = strings.TrimSpace(address) == ""
	}
	reserveMsg, reserveOK := annualStatementReserveMessage(r.URL.Query().Get("reserve"))
	reserveDocuments := []web.AnnualStatementReceiptDocumentView{}
	if ac.repositories.documents != nil {
		for _, document := range ac.repositories.documents.List() {
			if document.Current && annualStatementReceiptContentTypeSupported(document.ContentType) {
				reserveDocuments = append(reserveDocuments, web.AnnualStatementReceiptDocumentView{ID: document.ID, Label: document.Title + " · " + document.Filename})
			}
		}
	}
	showReserve := legal.Regime == "weg" && structureYear > 0 && ac.repositories.annualStatementReserve != nil
	reserveView := web.AnnualStatementReserveView{}
	if showReserve {
		titles := map[string]string{}
		for _, document := range reserveDocuments {
			titles[document.ID] = document.Label
		}
		reserveView, showReserve = annualStatementReserveView(ac.repositories.annualStatementReserve, selectedYear, selectedPeriod, units, reserveDocuments, titles)
	}
	a.renderSettingsComponent(w, r, tenant.Slug, web.AnnualStatementPage(web.AnnualStatementPageData{
		Portal:     a.settingsPortalContext(ac, "Jahresabrechnung", "settings"),
		EstateName: tenant.Name, EstateAddress: tenant.Address,
		Legal:   legal,
		Periods: periodViews, HasPeriods: len(periodViews) > 0,
		Year: formYear, StructureYear: structureYear, StartsOn: startsOn, EndsOn: endsOn,
		PeriodMsg: periodMsg, PeriodOK: periodOK, Followup: followup, ImportMsg: importMsg, ImportOK: importOK,
		CostTypes: costTypeViews, CostTypeCount: len(costTypeViews), AllocatableCostTypeCount: allocatableCount,
		CostTypeMsg: costTypeMsg, CostTypeOK: costTypeOK,
		ReceiptDocuments: receiptDocuments, HasReceiptDocuments: len(receiptDocuments) > 0,
		ReceiptSuggesterReady: a.annualStatementReceiptSuggester != nil,
		ReceiptMsg:            receiptMsg, ReceiptOK: receiptOK,
		ReceiptSuggestion: receiptSuggestion, HasReceiptSuggestion: receiptSuggestion.DocumentID != "",
		Receipts: receiptViews, HasReceipts: len(receiptViews) > 0,
		Units: unitViews, HasUnits: len(unitViews) > 0, UnitDataNotice: unitNotice,
		Allocation: allocation, Consumption: consumption.View, BasesMsg: basesMsg, BasesOK: basesOK,
		Run:              runView,
		TenantStatements: a.tenantStatementPanel(ac, runView.ID, r.URL.Query().Get("tenant-message")),
		Prepayments:      prepaymentViews, PrepaymentMsg: prepaymentMsg, PrepaymentOK: prepaymentOK, SettlementReady: settlementReady,
		ShowReserve: showReserve, Reserve: reserveView, ReserveMsg: reserveMsg, ReserveOK: reserveOK,
	}))
}

func (a *app) saveAnnualStatementPrepayment(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil || ac.repositories.annualStatementAkontos == nil || ac.repositories.annualStatementPeriods == nil {
		http.Redirect(w, r, annualStatementPrepaymentRedirect(0, "invalid"), http.StatusSeeOther)
		return
	}
	year, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("year")))
	redirectYear := annualStatementPrepaymentDisplayYear(year, ac.repositories.annualStatementPeriods.List())
	amount, amountOK := parseAnnualStatementPrepaymentAmount(r.FormValue("amount"))
	if !amountOK {
		http.Redirect(w, r, annualStatementPrepaymentRedirect(redirectYear, "invalid"), http.StatusSeeOther)
		return
	}
	saved, previous, err := ac.repositories.annualStatementAkontos.Save(store.AnnualStatementPrepayment{
		PeriodYear: year, UnitID: r.FormValue("unit_id"), AmountCents: amount, UpdatedBy: actorEmail,
	})
	if err != nil {
		logError("annual statement prepayment save failed", err, "tenant", tenant.Slug, "year", year)
		http.Redirect(w, r, annualStatementPrepaymentRedirect(redirectYear, "invalid"), http.StatusSeeOther)
		return
	}
	previousAmount := ""
	if previous != nil {
		previousAmount = strconv.FormatInt(previous.AmountCents, 10)
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug, ActorEmail: actorEmail, ActorRole: role,
		Action: auditActionAnnualPrepaymentSave, TargetType: "annual-statement-prepayment", TargetID: strconv.Itoa(saved.PeriodYear) + ":" + saved.UnitID,
		Summary: "Vorauszahlung gespeichert", Details: map[string]string{
			"period_year": strconv.Itoa(saved.PeriodYear), "unit_id": saved.UnitID,
			"previous_amount_cents": previousAmount, "new_amount_cents": strconv.FormatInt(saved.AmountCents, 10),
		},
	})
	http.Redirect(w, r, annualStatementPrepaymentRedirect(saved.PeriodYear, "saved"), http.StatusSeeOther)
}

func parseAnnualStatementPrepaymentAmount(raw string) (int64, bool) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	parts := strings.Split(raw, ".")
	if len(parts) != 2 || parts[0] == "" || len(parts[1]) != 2 {
		return 0, false
	}
	// ParseInt accepts signs, including negative zero and signed fractional
	// components. Money input consists only of unsigned decimal digits.
	for _, part := range parts {
		for _, digit := range part {
			if digit < '0' || digit > '9' {
				return 0, false
			}
		}
	}
	euros, eurosErr := strconv.ParseInt(parts[0], 10, 64)
	cents, centsErr := strconv.ParseInt(parts[1], 10, 64)
	if eurosErr != nil || centsErr != nil || euros < 0 || cents < 0 || cents > 99 || euros > (int64(^uint64(0)>>1)-cents)/100 {
		return 0, false
	}
	return euros*100 + cents, true
}

func annualStatementPrepaymentDisplayYear(requested int, periods []store.AnnualStatementPeriod) int {
	for _, period := range periods {
		if period.Year == requested {
			return requested
		}
	}
	if len(periods) > 0 {
		return periods[0].Year
	}
	return 0
}

func annualStatementPrepaymentRedirect(year int, status string) string {
	return "/app/settings/annual-statement?year=" + strconv.Itoa(year) + "&prepayment=" + status + "#vorauszahlungen"
}

func annualStatementPrepaymentMessage(status string) (string, bool) {
	switch strings.TrimSpace(status) {
	case "saved":
		return "Vorauszahlung gespeichert.", true
	case "invalid":
		return "Vorauszahlung konnte nicht gespeichert werden. Bitte Abrechnungsjahr, Einheit und Betrag prüfen.", false
	default:
		return "", false
	}
}

func formatAnnualStatementMoney(cents int64) string {
	negative := cents < 0
	if negative {
		cents = -cents
	}
	formatted := formatAnnualStatementReceiptAmount(cents)
	if negative {
		return "−" + formatted
	}
	return formatted
}

func formatAnnualStatementBalance(cents int64) string {
	if cents < 0 {
		return "Nachzahlung " + formatAnnualStatementMoney(-cents)
	}
	if cents > 0 {
		return "Guthaben " + formatAnnualStatementMoney(cents)
	}
	return "Ausgeglichen"
}

func annualStatementVATRate(key, raw string) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return store.DefaultAnnualStatementVATPercent(strings.ToLower(strings.TrimSpace(key))), true
	}
	rate, err := strconv.Atoi(raw)
	if err != nil || !store.ValidAnnualStatementVATPercent(rate) {
		return 0, false
	}
	return rate, true
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
	vatRate, vatOK := annualStatementVATRate(r.FormValue("key"), r.FormValue("vat_rate"))
	if !vatOK {
		http.Redirect(w, r, "/app/settings/annual-statement?cost-type=invalid", http.StatusSeeOther)
		return
	}
	costType := store.AnnualStatementCostType{
		Key: r.FormValue("key"), Name: r.FormValue("name"), Allocatable: allocation == "allocatable",
		AllocationKey: r.FormValue("allocation_key"), VATRatePercent: vatRate,
		UpdatedAt: time.Now().UTC(), UpdatedBy: actorEmail,
	}
	periodYear, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("period_year")))
	var saved store.AnnualStatementCostType
	var err error
	if periodYear > 0 && ac.repositories.annualStatementPeriods != nil {
		saved, err = ac.repositories.annualStatementPeriods.SaveStructureCostType(periodYear, costType)
	} else {
		if err = ac.repositories.annualStatementCostTypes.EnsureDefaults(actorEmail); err == nil {
			saved, err = ac.repositories.annualStatementCostTypes.Save(costType)
		}
	}
	if err != nil {
		http.Redirect(w, r, annualStatementStructureRedirect(periodYear, "cost-type", "invalid", ""), http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug, ActorEmail: actorEmail, ActorRole: role,
		Action: auditActionAnnualCostTypeSave, TargetType: "annual-statement-cost-type", TargetID: saved.Key,
		Summary: "Kostenart gespeichert",
		Details: map[string]string{"period_year": strconv.Itoa(periodYear), "key": saved.Key, "name": saved.Name, "allocatable": strconv.FormatBool(saved.Allocatable), "allocation_key": saved.AllocationKey},
	})
	http.Redirect(w, r, annualStatementStructureRedirect(periodYear, "cost-type", "saved", ""), http.StatusSeeOther)
}

// saveAnnualStatementAllocationBases records Nutzfläche and Personen for every
// unit in one submit. For a selected period all three bases, including
// Nutzwert, stay in that period's snapshot; nothing here invents a share.
func (a *app) saveAnnualStatementAllocationBases(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil || ac.repositories.units == nil {
		http.Redirect(w, r, "/app/settings/annual-statement?bases=invalid", http.StatusSeeOther)
		return
	}
	periodYear, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("period_year")))
	// All or nothing: the three parallel arrays must line up, every row must
	// name a unit, every value must parse. Any defect rejects the whole
	// submit before the store is touched, so nothing is written or audited.
	updates, ok := annualStatementBasisUpdatesFromForm(r.Form)
	if !ok {
		http.Redirect(w, r, "/app/settings/annual-statement?bases=invalid", http.StatusSeeOther)
		return
	}
	// The page submits every unit of the tenant in one form. A submit whose
	// unit set differs from the current one comes from a stale page (a unit
	// was added, renamed or removed meanwhile) and is rejected as a whole;
	// the store's partial update is deliberately not exposed here.
	if !annualStatementBasisUpdatesCoverUnits(updates, ac.repositories.units.List()) {
		http.Redirect(w, r, "/app/settings/annual-statement?bases=stale", http.StatusSeeOther)
		return
	}
	var err error
	if periodYear > 0 && ac.repositories.annualStatementPeriods != nil {
		structure, found := ac.repositories.annualStatementPeriods.Structure(periodYear)
		if !found {
			http.Redirect(w, r, annualStatementStructureRedirect(periodYear, "bases", "stale", "verteilerschluessel"), http.StatusSeeOther)
			return
		}
		basisByUnit := make(map[string]store.AnnualStatementPeriodUnitBasis, len(structure.UnitBases))
		for _, basis := range structure.UnitBases {
			basisByUnit[normalizeUnitID(basis.UnitID)] = basis
		}
		canonicalByUnit := map[string]store.Unit{}
		for _, unit := range ac.repositories.units.List() {
			canonicalByUnit[normalizeUnitID(unit.ID)] = unit
		}
		bases := make([]store.AnnualStatementPeriodUnitBasis, 0, len(updates))
		for _, update := range updates {
			basis, exists := basisByUnit[update.UnitID]
			if !exists {
				basis = store.AnnualStatementPeriodUnitBasis{UnitID: update.UnitID, MiteigentumsanteilPPM: canonicalByUnit[update.UnitID].MiteigentumsanteilPPM}
			}
			basis.UsableAreaM2Hundredths = update.UsableAreaM2Hundredths
			basis.UsableAreaRecorded = update.UsableAreaRecorded
			basis.Persons = update.Persons
			basis.PersonsRecorded = update.PersonsRecorded
			basis.VacantFrom = update.VacantFrom
			basis.VacantTo = update.VacantTo
			bases = append(bases, basis)
		}
		err = ac.repositories.annualStatementPeriods.SaveStructureUnitBases(periodYear, bases)
	} else {
		var unknownUnit bool
		unknownUnit, err = ac.repositories.units.UpdateAllocationBases(updates)
		if unknownUnit {
			http.Redirect(w, r, annualStatementStructureRedirect(periodYear, "bases", "stale", "verteilerschluessel"), http.StatusSeeOther)
			return
		}
	}
	if err != nil {
		logError("annual statement allocation bases save failed", err, "tenant", tenant.Slug)
		http.Redirect(w, r, annualStatementStructureRedirect(periodYear, "bases", "error", "verteilerschluessel"), http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug, ActorEmail: actorEmail, ActorRole: role,
		Action: auditActionAnnualBasesSave, TargetType: "annual-statement-allocation-bases", TargetID: tenant.Slug,
		Summary: "Verteilerbasis je Einheit gespeichert", Details: map[string]string{"period_year": strconv.Itoa(periodYear), "units": strconv.Itoa(len(updates))},
	})
	http.Redirect(w, r, annualStatementStructureRedirect(periodYear, "bases", "saved", "verteilerschluessel"), http.StatusSeeOther)
}

func annualStatementStructureRedirect(year int, key, status, fragment string) string {
	query := url.Values{key: {status}}
	if year > 0 {
		query.Set("year", strconv.Itoa(year))
	}
	target := "/app/settings/annual-statement?" + query.Encode()
	if fragment != "" {
		target += "#" + fragment
	}
	return target
}

func annualStatementBasisUpdatesFromForm(values url.Values) ([]store.UnitAllocationBasisUpdate, bool) {
	unitIDs, areas, persons := values["unit_id"], values["usable_area_m2"], values["persons"]
	vacantFrom, vacantTo := values["vacant_from"], values["vacant_to"]
	if len(unitIDs) == 0 || len(areas) != len(unitIDs) || len(persons) != len(unitIDs) {
		return nil, false
	}
	if len(vacantFrom) == 0 && len(vacantTo) == 0 {
		vacantFrom = make([]string, len(unitIDs))
		vacantTo = make([]string, len(unitIDs))
	}
	if len(vacantFrom) != len(unitIDs) || len(vacantTo) != len(unitIDs) {
		return nil, false
	}
	updates := make([]store.UnitAllocationBasisUpdate, 0, len(unitIDs))
	seen := map[string]struct{}{}
	for index, rawID := range unitIDs {
		unitID := normalizeUnitID(rawID)
		if _, duplicate := seen[unitID]; unitID == "" || duplicate {
			return nil, false
		}
		seen[unitID] = struct{}{}
		area, areaRecorded, areaOK := parseAnnualStatementArea(areas[index])
		count, countRecorded, countOK := parseAnnualStatementCount(persons[index])
		from, to, vacancyOK := parseAnnualStatementVacancy(vacantFrom[index], vacantTo[index])
		if !areaOK || !countOK || !vacancyOK {
			return nil, false
		}
		updates = append(updates, store.UnitAllocationBasisUpdate{
			UnitID: unitID, UsableAreaM2Hundredths: area, UsableAreaRecorded: areaRecorded, Persons: count, PersonsRecorded: countRecorded,
			VacantFrom: from, VacantTo: to,
		})
	}
	return updates, true
}

// annualStatementBasisUpdatesCoverUnits is true when the submitted rows name
// exactly the tenant's current units — no extra, no missing.
func annualStatementBasisUpdatesCoverUnits(updates []store.UnitAllocationBasisUpdate, units []store.Unit) bool {
	if len(updates) != len(units) {
		return false
	}
	current := make(map[string]struct{}, len(units))
	for _, item := range units {
		current[normalizeUnitID(item.ID)] = struct{}{}
	}
	for _, update := range updates {
		if _, found := current[update.UnitID]; !found {
			return false
		}
	}
	return true
}

// parseAnnualStatementArea accepts "72,5", "72.50", "0" (recorded, no
// Nutzfläche) or "" (not recorded) and returns hundredths of a square metre.
// More than two decimals, negatives and implausible sizes are rejected rather
// than rounded.
func parseAnnualStatementArea(raw string) (hundredths int, recorded bool, ok bool) {
	raw = strings.ReplaceAll(strings.TrimSpace(raw), ",", ".")
	if raw == "" {
		return 0, false, true
	}
	whole, fraction, _ := strings.Cut(raw, ".")
	if whole == "" {
		whole = "0"
	}
	if len(fraction) > 2 {
		return 0, false, false
	}
	fraction += strings.Repeat("0", 2-len(fraction))
	metres, err := strconv.Atoi(whole)
	if err != nil || metres < 0 || metres > 99_999 {
		return 0, false, false
	}
	cents, err := strconv.Atoi(fraction)
	if err != nil || cents < 0 {
		return 0, false, false
	}
	return metres*100 + cents, true, true
}

// parseAnnualStatementCount distinguishes "" (not recorded) from "0" (recorded,
// no persons — a Stellplatz or a vacant flat still takes part in the key).
func parseAnnualStatementCount(raw string) (count int, recorded bool, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false, true
	}
	count, err := strconv.Atoi(raw)
	if err != nil || count < 0 || count > 10_000 {
		return 0, false, false
	}
	return count, true, true
}

func parseAnnualStatementVacancy(from, to string) (string, string, bool) {
	from, to, err := store.NormalizeAnnualStatementVacancy(from, to)
	return from, to, err == nil
}

func annualStatementAllocationKeyLabel(key string) string {
	switch key {
	case store.AllocationKeyAgreed:
		return "Vereinbarte Anteile"
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

func annualStatementUnitsWithPeriodBases(units []store.Unit, bases []store.AnnualStatementPeriodUnitBasis) []store.Unit {
	byUnit := make(map[string]store.AnnualStatementPeriodUnitBasis, len(bases))
	for _, basis := range bases {
		byUnit[normalizeUnitID(basis.UnitID)] = basis
	}
	out := append([]store.Unit(nil), units...)
	for index := range out {
		basis, found := byUnit[normalizeUnitID(out[index].ID)]
		if !found {
			out[index].MiteigentumsanteilPPM = 0
			out[index].UsableAreaM2Hundredths = 0
			out[index].UsableAreaRecorded = false
			out[index].Persons = 0
			out[index].PersonsRecorded = false
			continue
		}
		out[index].MiteigentumsanteilPPM = basis.MiteigentumsanteilPPM
		out[index].UsableAreaM2Hundredths = basis.UsableAreaM2Hundredths
		out[index].UsableAreaRecorded = basis.UsableAreaRecorded
		out[index].Persons = basis.Persons
		out[index].PersonsRecorded = basis.PersonsRecorded
	}
	return out
}

func annualStatementAllocationView(costTypes []store.AnnualStatementCostType, units []store.Unit, consumptionData annualStatementConsumptionData, bases []store.AnnualStatementPeriodUnitBasis, agreed ...map[string]map[string]int) web.AnnualStatementAllocationView {
	names := map[string]string{}
	for _, costType := range costTypes {
		names[costType.Key] = costType.Name
	}
	vacancy := map[string]store.AnnualStatementPeriodUnitBasis{}
	for _, basis := range bases {
		vacancy[normalizeUnitID(basis.UnitID)] = basis
	}
	out := web.AnnualStatementAllocationView{KeyOptions: annualStatementAllocationKeyOptions()}
	for _, unit := range units {
		basis := vacancy[normalizeUnitID(unit.ID)]
		out.Bases = append(out.Bases, web.AnnualStatementUnitBasisView{
			ID: unit.ID, Label: unit.Label, Share: formatMiteigentumsanteil(unit.MiteigentumsanteilPPM),
			UsableArea: formatAnnualStatementArea(unit.UsableAreaM2Hundredths, unit.UsableAreaRecorded), Persons: formatAnnualStatementCount(unit.Persons, unit.PersonsRecorded),
			VacantFrom: basis.VacantFrom, VacantTo: basis.VacantTo,
		})
	}
	keylessNames := []string{}
	for _, key := range store.AnnualStatementCostTypesWithoutKey(costTypes) {
		keylessNames = append(keylessNames, names[key])
	}
	out.KeylessCostTypes = strings.Join(keylessNames, ", ")
	for _, preview := range store.AnnualStatementAllocationPreviews(costTypes, units, agreed...) {
		if preview.Key == store.AllocationKeyVerbrauch {
			continue
		}
		previewView := web.AnnualStatementAllocationPreviewView{
			Key: preview.Key, Label: annualStatementAllocationKeyLabel(preview.Key), Blocked: preview.Blocked,
			UnmappedUnits: strings.Join(preview.UnmappedUnits, ", "), Sourceless: preview.Key == store.AllocationKeyVerbrauch,
		}
		if preview.Key == store.AllocationKeyNutzwert && !preview.Blocked && preview.BasisTotal != store.MiteigentumsanteilTotalPPM {
			previewView.TotalNotice = "Die Miteigentumsanteile summieren sich auf " + view.FormatDecimal(float64(preview.BasisTotal), 0) + " statt 1.000.000. Die Anteile beziehen sich auf diese Summe; prüfen Sie, ob eine Einheit fehlt oder ein Anteil noch nicht hinterlegt ist."
		}
		costTypeNames := make([]string, 0, len(preview.CostTypeKeys))
		for _, key := range preview.CostTypeKeys {
			costTypeNames = append(costTypeNames, names[key])
		}
		previewView.CostTypes = strings.Join(costTypeNames, ", ")
		for _, share := range preview.Shares {
			previewView.Shares = append(previewView.Shares, web.AnnualStatementUnitShareView{
				Label: share.Label, Basis: formatAnnualStatementBasis(preview.Key, share.Basis, share.Mapped), Share: formatAnnualStatementShare(share.SharePPM, share.Mapped && !preview.Blocked), Mapped: share.Mapped,
			})
		}
		out.Previews = append(out.Previews, previewView)
		if preview.Blocked {
			out.BlockedCount++
		}
	}
	for _, preview := range annualStatementConsumptionAllocationViews(costTypes, units, consumptionData) {
		out.Previews = append(out.Previews, preview)
		if preview.Blocked {
			out.BlockedCount++
		}
	}
	out.RunReady = len(out.Previews) > 0 && out.BlockedCount == 0 && out.KeylessCostTypes == ""
	return out
}

func formatAnnualStatementArea(hundredths int, recorded bool) string {
	if !recorded || hundredths < 0 {
		return ""
	}
	return strings.Replace(fmt.Sprintf("%d.%02d", hundredths/100, hundredths%100), ".", ",", 1)
}

func formatAnnualStatementCount(count int, recorded bool) string {
	if !recorded || count < 0 {
		return ""
	}
	return strconv.Itoa(count)
}

func formatAnnualStatementBasis(key string, basis int, mapped bool) string {
	if !mapped {
		return "fehlt"
	}
	switch key {
	case store.AllocationKeyNutzwert:
		return formatMiteigentumsanteil(basis)
	case store.AllocationKeyFlaeche:
		return formatAnnualStatementArea(basis, true) + " m²"
	case store.AllocationKeyAgreed:
		return strconv.Itoa(basis) + " PPM"
	case store.AllocationKeyPersonen:
		return strconv.Itoa(basis)
	default:
		return "fehlt"
	}
}

// formatAnnualStatementShare renders parts per million as a percentage
// rounded to the nearest hundredth of a percent (half up). The underlying
// ppm allocation stays exact; only its visible representation is rounded.
// A mapped unit with a zero basis (0 Personen) is a real "0,00 %"; an
// unmapped or blocked one shows no share.
func formatAnnualStatementShare(ppm int, mapped bool) string {
	if !mapped || ppm < 0 {
		return "–"
	}
	// One hundredth of a percent is 100 ppm. Adding half that unit before
	// integer division gives the required half-up rule and carries naturally.
	hundredths := (ppm + 50) / 100
	return strings.Replace(fmt.Sprintf("%d.%02d %%", hundredths/100, hundredths%100), ".", ",", 1)
}

func annualStatementBasesMessage(status string) (string, bool) {
	switch strings.TrimSpace(status) {
	case "saved":
		return "Verteilerbasis je Einheit gespeichert.", true
	case "stale":
		return "Die Einheiten haben sich seit dem Laden der Seite geändert. Bitte die Seite neu laden; es wurde nichts gespeichert.", false
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
	if ac.repositories.annualStatementCostTypes == nil || ac.repositories.units == nil {
		http.Redirect(w, r, "/app/settings/annual-statement?period=invalid", http.StatusSeeOther)
		return
	}
	if err := ac.repositories.annualStatementCostTypes.EnsureDefaults(actorEmail); err != nil {
		http.Redirect(w, r, "/app/settings/annual-statement?period=invalid", http.StatusSeeOther)
		return
	}
	if _, err := ac.repositories.annualStatementPeriods.SaveWithStructure(period, ac.repositories.annualStatementCostTypes.List(), ac.repositories.units.List()); err != nil {
		logError("annual statement period structure save failed", err, "tenant", tenant.Slug, "year", year)
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

func (a *app) cloneNextAnnualStatementPeriod(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actorEmail, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil || ac.repositories.annualStatementPeriods == nil || ac.repositories.annualStatementCostTypes == nil || ac.repositories.units == nil {
		http.Redirect(w, r, "/app/settings/annual-statement?period=invalid#perioden", http.StatusSeeOther)
		return
	}
	sourceYear, err := strconv.Atoi(strings.TrimSpace(r.FormValue("source_year")))
	periods := ac.repositories.annualStatementPeriods.List()
	var source store.AnnualStatementPeriod
	found := false
	for _, period := range periods {
		if period.Year == sourceYear {
			source, found = period, true
			break
		}
	}
	target, valid := annualStatementFollowupPeriod(source, actorEmail)
	if err != nil || !found || !valid {
		http.Redirect(w, r, "/app/settings/annual-statement?period=invalid#perioden", http.StatusSeeOther)
		return
	}
	if err := ac.repositories.annualStatementCostTypes.EnsureDefaults(actorEmail); err != nil {
		logError("annual statement source defaults unavailable", err, "tenant", tenant.Slug, "source_year", sourceYear)
		http.Redirect(w, r, "/app/settings/annual-statement?year="+strconv.Itoa(sourceYear)+"&period=error#perioden", http.StatusSeeOther)
		return
	}
	if err := ac.repositories.annualStatementPeriods.EnsureStructure(sourceYear, ac.repositories.annualStatementCostTypes.List(), ac.repositories.units.List(), actorEmail); err != nil {
		logError("annual statement source structure unavailable", err, "tenant", tenant.Slug, "source_year", sourceYear)
		http.Redirect(w, r, "/app/settings/annual-statement?year="+strconv.Itoa(sourceYear)+"&period=error#perioden", http.StatusSeeOther)
		return
	}
	saved, created, err := ac.repositories.annualStatementPeriods.CloneStructure(sourceYear, target)
	if err != nil {
		logError("annual statement follow-up period save failed", err, "tenant", tenant.Slug, "source_year", sourceYear, "target_year", target.Year)
		http.Redirect(w, r, "/app/settings/annual-statement?year="+strconv.Itoa(sourceYear)+"&period=error#perioden", http.StatusSeeOther)
		return
	}
	if !created {
		http.Redirect(w, r, "/app/settings/annual-statement?year="+strconv.Itoa(target.Year)+"&period=exists#perioden", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug, ActorEmail: actorEmail, ActorRole: role,
		Action: auditActionAnnualPeriodSave, TargetType: "annual-statement-period", TargetID: strconv.Itoa(saved.Year),
		Summary: "Folgeperiode aus Vorlage erstellt",
		Details: map[string]string{
			"source_year": strconv.Itoa(sourceYear), "year": strconv.Itoa(saved.Year),
			"starts_on": saved.StartsOn, "ends_on": saved.EndsOn, "copied_amounts": "false",
		},
	})
	http.Redirect(w, r, "/app/settings/annual-statement?year="+strconv.Itoa(saved.Year)+"&period=cloned#perioden", http.StatusSeeOther)
}

func annualStatementFollowupPeriod(source store.AnnualStatementPeriod, updatedBy string) (store.AnnualStatementPeriod, bool) {
	if source.Year < 1 || source.Year >= 9999 {
		return store.AnnualStatementPeriod{}, false
	}
	startsOn, startsOK := annualStatementShiftDate(source.StartsOn, 12)
	endsOn, endsOK := annualStatementShiftDate(source.EndsOn, 12)
	if !startsOK || !endsOK {
		return store.AnnualStatementPeriod{}, false
	}
	return store.AnnualStatementPeriod{
		Year: source.Year + 1, StartsOn: startsOn, EndsOn: endsOn,
		UpdatedAt: time.Now().UTC(), UpdatedBy: updatedBy,
	}, true
}

func annualStatementShiftDate(raw string, months int) (string, bool) {
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return "", false
	}
	totalMonths := int(parsed.Month()) - 1 + months
	year := parsed.Year() + totalMonths/12
	monthIndex := totalMonths % 12
	if monthIndex < 0 {
		year--
		monthIndex += 12
	}
	month := time.Month(monthIndex + 1)
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	day := parsed.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Format("2006-01-02"), true
}

func annualStatementDeadline(endsOn string) string {
	deadline, ok := annualStatementShiftDate(endsOn, 6)
	if !ok {
		return "–"
	}
	parsed, _ := time.Parse("2006-01-02", deadline)
	return parsed.Format("02.01.2006")
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
	unit       string
	role       string
	email      string
	name       string
	address    string
	nameSet    bool
	addressSet bool
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
		optional := func(key string) (string, bool) {
			index, found := columns[key]
			if !found || index >= len(row) {
				// A row shorter than its header has not stated the value: it
				// must not clear what an earlier import recorded.
				return "", false
			}
			return strings.TrimSpace(row[index]), true
		}
		name, nameSet := optional("name")
		address, addressSet := optional("anschrift")
		if name == "" {
			name = parsedEmail.Name
		}
		assignments = append(assignments, annualStatementPartyAssignment{unit: unitValue, role: normalizedRole, email: normalizeEmail(parsedEmail.Address), name: name, address: address, nameSet: nameSet || name != "", addressSet: addressSet})
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
		contacts              []store.UnitPartyContact
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
		if assignment.nameSet || assignment.addressSet {
			contact := store.UnitPartyContact{Email: assignment.email}
			for _, existing := range append(append([]store.UnitPartyContact(nil), units[index].PartyContacts...), parties.contacts...) {
				if existing.Email == assignment.email {
					contact = existing
				}
			}
			if assignment.nameSet {
				contact.Name = assignment.name
			}
			if assignment.addressSet {
				contact.Address = assignment.address
			}
			parties.contacts = append(parties.contacts, contact)
		}
		imports[index] = parties
	}
	updates := make([]store.UnitPartyUpdate, 0, len(imports))
	for index, parties := range imports {
		updates = append(updates, store.UnitPartyUpdate{
			UnitID: units[index].ID, OwnerEmails: parties.owners, RenterEmails: parties.renters,
			SetOwners: parties.hasOwners, SetRenters: parties.hasRenters, Contacts: parties.contacts,
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
	case "cloned":
		return "Vorlage übernommen. Belege, Beträge und Akonto wurden nicht kopiert.", true
	case "exists":
		return "Das Folgejahr ist bereits vorhanden und wurde nicht überschrieben.", false
	case "invalid":
		return "Bitte Abrechnungsjahr und Zeitraum vollständig und chronologisch eingeben.", false
	case "error":
		return "Das Folgejahr konnte nicht angelegt werden.", false
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

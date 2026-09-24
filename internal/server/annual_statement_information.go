package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/inspr-at/hausv-org/internal/store"
)

func (a *app) saveAnnualStatementAgreedShares(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actor, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if r.ParseForm() != nil || ac.repositories.annualStatementPeriods == nil || ac.repositories.units == nil {
		http.Error(w, "Ungültige Eingabe.", 400)
		return
	}
	year, err := strconv.Atoi(r.FormValue("year"))
	structure, found := ac.repositories.annualStatementPeriods.Structure(year)
	key := r.FormValue("cost_type")
	known := false
	for _, cost := range structure.CostTypes {
		if cost.Key == key && cost.Allocatable && cost.AllocationKey == store.AllocationKeyAgreed {
			known = true
		}
	}
	units, unitErr := ac.repositories.units.ListChecked()
	if err != nil || !found || !known || unitErr != nil {
		http.Error(w, "Periode und Kostenart prüfen.", 400)
		return
	}
	shares := map[string]int{}
	for _, unit := range units {
		n, e := strconv.Atoi(r.FormValue("share_" + unit.ID))
		if e != nil || n < 0 || n > 1_000_000 {
			http.Error(w, "Jede Einheit benötigt einen Anteil zwischen 0 und 1.000.000 PPM.", 400)
			return
		}
		shares[unit.ID] = n
	}
	if store.AnnualStatementAgreedPreview(key, units, shares).Blocked {
		http.Error(w, "Die vereinbarten Anteile müssen genau 1.000.000 PPM ergeben.", 400)
		return
	}
	if structure.Legal.AgreedShares == nil {
		structure.Legal.AgreedShares = map[string]map[string]int{}
	}
	structure.Legal.AgreedShares[key] = shares
	if err := ac.repositories.annualStatementPeriods.SaveLegal(year, structure.Legal); err != nil {
		http.Error(w, "Anteile konnten nicht gespeichert werden.", 400)
		return
	}
	a.recordAudit(auditEvent{TenantSlug: tenant.Slug, ActorEmail: actor, ActorRole: role, Action: store.AuditActionAnnualPeriodSave, TargetType: "annual-statement-period", TargetID: strconv.Itoa(year), Summary: "Vereinbarte Anteile gespeichert", Details: map[string]string{"cost_type": key}})
	http.Redirect(w, r, "/app/settings/annual-statement?year="+strconv.Itoa(year)+"#vereinbarte-anteile", http.StatusSeeOther)
}

// Decimal EUR and quantities support six places without float conversion.
func parseAnnualInformationMicros(raw string) (int64, bool) {
	raw = strings.ReplaceAll(strings.TrimSpace(raw), ",", ".")
	parts := strings.Split(raw, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, false
	}
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
		if len(frac) == 0 || len(frac) > 6 {
			return 0, false
		}
	}
	for _, c := range parts[0] + frac {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	n, err := strconv.ParseInt(parts[0]+frac+strings.Repeat("0", 6-len(frac)), 10, 64)
	return n, err == nil
}

func (a *app) saveAnnualStatementHeatingInformation(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actor, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if r.ParseForm() != nil || ac.repositories.annualStatementPeriods == nil {
		http.Error(w, "Ungültige Eingabe.", 400)
		return
	}
	year, err := strconv.Atoi(r.FormValue("year"))
	structure, found := ac.repositories.annualStatementPeriods.Structure(year)
	if err != nil || !found {
		http.Error(w, "Periode fehlt.", 400)
		return
	}
	info := store.AnnualStatementHeatingInformation{TaxesNote: strings.TrimSpace(r.FormValue("taxes_note")), DistrictHeatingOver20MW: r.FormValue("district_over_20mw") == "on", FuelMix: strings.TrimSpace(r.FormValue("fuel_mix")), Emissions: strings.TrimSpace(r.FormValue("emissions")), MeteringCostsNote: strings.TrimSpace(r.FormValue("metering_costs_note")), OperatingCostsNote: strings.TrimSpace(r.FormValue("operating_costs_note")), RemoteMeters: r.FormValue("remote_meters"), MonthlyInformation: strings.TrimSpace(r.FormValue("monthly_information")), ClimateSource: strings.TrimSpace(r.FormValue("climate_source")), ComplaintContact: strings.TrimSpace(r.FormValue("complaint_contact"))}
	for name, target := range map[string]*int{"climate_current_ppm": &info.ClimateCurrentPPM, "climate_previous_ppm": &info.ClimatePreviousPPM} {
		if raw := r.FormValue(name); raw != "" {
			n, e := strconv.Atoi(raw)
			if e != nil {
				http.Error(w, "Ungültiger Klimafaktor.", 400)
				return
			}
			*target = n
		}
	}
	rows := r.Form["purchase_row"]
	if len(rows) > 100 {
		http.Error(w, "Zu viele Energiebezüge.", 400)
		return
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if seen[row] {
			http.Error(w, "Doppelte Energiebezugszeile.", 400)
			return
		}
		seen[row] = true
		field := func(name string) string { return strings.TrimSpace(r.FormValue("purchase_" + row + "_" + name)) }
		if field("supplier") == "" && field("carrier") == "" && field("quantity") == "" && field("price") == "" && field("note") == "" {
			continue
		}
		quantity, qOK := parseAnnualInformationMicros(field("quantity"))
		price, pOK := parseAnnualInformationMicros(field("price"))
		if !qOK || !pOK {
			http.Error(w, "Menge und Preis mit höchstens sechs Nachkommastellen eingeben.", 400)
			return
		}
		info.Purchases = append(info.Purchases, store.AnnualStatementEnergyPurchase{CostTypeKey: field("cost"), Supplier: field("supplier"), Carrier: field("carrier"), QuantityMicros: quantity, Unit: field("unit"), PriceMicros: price, PriceNote: field("note")})
	}
	structure.Legal.HeatingInformation = &info
	if err := ac.repositories.annualStatementPeriods.SaveLegal(year, structure.Legal); err != nil {
		http.Error(w, "Bitte Energiebezüge, Preisstand, Einheiten und Klimafaktoren mit Quelle prüfen.", 400)
		return
	}
	a.recordAudit(auditEvent{TenantSlug: tenant.Slug, ActorEmail: actor, ActorRole: role, Action: store.AuditActionAnnualPeriodSave, TargetType: "annual-statement-period", TargetID: strconv.Itoa(year), Summary: "HeizKG-Abrechnungsinformationen gespeichert"})
	http.Redirect(w, r, "/app/settings/annual-statement?year="+strconv.Itoa(year)+"#heizinformationen", http.StatusSeeOther)
}

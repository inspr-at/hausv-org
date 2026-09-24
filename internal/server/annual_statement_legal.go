package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func (a *app) saveAnnualStatementLegal(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actor, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if r.ParseForm() != nil || ac.repositories.annualStatementPeriods == nil {
		http.Error(w, "Ungültige Eingabe.", 400)
		return
	}
	year, err := strconv.Atoi(r.FormValue("year"))
	legal := store.AnnualStatementLegalSettings{Regime: r.FormValue("regime"), HeizKGApplies: r.FormValue("heizkg_applies") == "on"}
	if err != nil || ac.repositories.annualStatementPeriods.SaveLegal(year, legal) != nil {
		http.Error(w, "Bitte Rechtsgrundlage und Periode prüfen.", 400)
		return
	}
	a.recordAudit(auditEvent{TenantSlug: tenant.Slug, ActorEmail: actor, ActorRole: role, Action: store.AuditActionAnnualPeriodSave, TargetType: "annual-statement-period", TargetID: strconv.Itoa(year), Summary: "Rechtsgrundlage der Periode gespeichert"})
	http.Redirect(w, r, "/app/settings/annual-statement?year="+strconv.Itoa(year)+"#rechtsgrundlage", http.StatusSeeOther)
}
func annualLegalDeadline(period store.AnnualStatementPeriod, legal store.AnnualStatementLegalSettings) string {
	raw := legal.Deadline(period)
	if raw == "" {
		return "Laut Vertrag"
	}
	at, _ := time.Parse("2006-01-02", raw)
	return at.Format("02.01.2006")
}

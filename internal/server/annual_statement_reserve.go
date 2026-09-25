package server

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	display "github.com/inspr-at/hausv-org/internal/view"
	"github.com/inspr-at/hausv-org/internal/web"
)

func (a *app) saveAnnualStatementReserve(w http.ResponseWriter, r *http.Request, ac authCtx) {
	tenant, actor, role, _, ok := a.buildingSettingsContext(w, ac)
	if !ok {
		return
	}
	if r.ParseForm() != nil || ac.repositories.annualStatementReserve == nil {
		http.Error(w, "Ungültige Eingabe.", http.StatusBadRequest)
		return
	}
	year, err := strconv.Atoi(r.FormValue("year"))
	amount, amountOK := parseAnnualStatementReserveAmount(r.FormValue("amount"))
	if err != nil || !amountOK {
		http.Redirect(w, r, "/app/settings/annual-statement?year="+strconv.Itoa(year)+"&reserve=invalid#ruecklage", http.StatusSeeOther)
		return
	}
	entry := store.AnnualStatementReserveEntry{
		PeriodYear: year, Kind: r.FormValue("kind"), EntryDate: r.FormValue("entry_date"),
		AmountCents: amount, DocumentID: r.FormValue("document_id"), Note: r.FormValue("note"),
		CreatedAt: time.Now().UTC(), CreatedBy: actor,
	}
	saved, err := ac.repositories.annualStatementReserve.Add(entry)
	if err != nil {
		http.Redirect(w, r, "/app/settings/annual-statement?year="+strconv.Itoa(year)+"&reserve=invalid#ruecklage", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: tenant.Slug, ActorEmail: actor, ActorRole: role,
		Action: store.AuditActionAnnualReserveAdd, TargetType: "annual-statement-reserve", TargetID: saved.ID,
		Summary: "Rücklage gebucht",
		Details: map[string]string{"kind": saved.Kind, "amount_cents": strconv.FormatInt(saved.AmountCents, 10), "actor": actor},
	})
	http.Redirect(w, r, "/app/settings/annual-statement?year="+strconv.Itoa(year)+"&reserve=saved#ruecklage", http.StatusSeeOther)
}

func parseAnnualStatementReserveAmount(raw string) (int64, bool) {
	raw = strings.TrimSpace(raw)
	sign := int64(1)
	if strings.HasPrefix(raw, "-") {
		sign = -1
		raw = strings.TrimSpace(raw[1:])
	}
	cents, ok := parseAnnualStatementPrepaymentAmount(raw)
	if !ok || cents == 0 {
		return 0, false
	}
	return sign * cents, true
}

func annualStatementReserveView(repo store.AnnualStatementReserveRepository, year int, period store.AnnualStatementPeriod, units []store.Unit, documents []web.AnnualStatementReceiptDocumentView, documentTitles map[string]string) (web.AnnualStatementReserveView, bool) {
	if repo == nil || year == 0 {
		return web.AnnualStatementReserveView{}, false
	}
	entries := repo.ListByPeriod(year)
	// Presentation order only: preserve the repository/snapshot ordering used
	// by historical runs, but show opening balances before same-day bookings.
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].EntryDate != entries[j].EntryDate {
			return entries[i].EntryDate < entries[j].EntryDate
		}
		return entries[i].Kind == store.ReserveKindOpening && entries[j].Kind != store.ReserveKindOpening
	})
	balance, ok := store.AnnualStatementReserveBalance(entries, period, units)
	if !ok {
		balance = store.AnnualStatementReserveResult{}
	}
	view := web.AnnualStatementReserveView{
		Opening: formatAnnualStatementMoney(balance.OpeningCents), Contributions: formatAnnualStatementMoney(balance.ContributionCents),
		Withdrawals: formatAnnualStatementMoney(balance.WithdrawalCents), Interest: formatAnnualStatementMoney(balance.InterestCents),
		Closing: formatAnnualStatementMoney(balance.ClosingCents), Documents: documents,
	}
	if !ok || balance.MinimumUnavailable {
		view.MinimumHint = "Die Mindest-Rücklage konnte für diesen Zeitraum nicht vollständig geprüft werden."
	} else if balance.AreaIncomplete {
		view.MinimumHint = "Die Mindest-Rücklage konnte nicht geprüft werden, weil die Nutzfläche unvollständig ist."
	} else if len(balance.MinimumRates) == 0 {
		view.MinimumHint = "Vor 01.07.2022 gab es keinen gesetzlichen Mindestbetrag je m². Eine angemessene Rücklage war dennoch zu bilden."
	} else if balance.MinimumWarning {
		view.MinimumHint = "Die Zuführungen liegen unter der Mindest-Rücklage: " + display.AnnualStatementReserveRateLabel(balance.MinimumRates) + " (WEG 2002 § 31)."
	} else if len(entries) > 0 {
		view.MinimumHint = "Die Zuführungen erreichen die Mindest-Rücklage: " + display.AnnualStatementReserveRateLabel(balance.MinimumRates) + "."
	}
	if balance.ClosingMismatch {
		view.MinimumHint = "Eine Endstand-Kontrolle weicht vom errechneten Endstand ab. " + view.MinimumHint
	}
	for _, entry := range entries {
		date := entry.EntryDate
		if parsed, err := time.Parse("2006-01-02", entry.EntryDate); err == nil {
			date = parsed.Format("02.01.2006")
		}
		note := ""
		if entry.Note != "" {
			note = " · " + entry.Note
		}
		document := ""
		if entry.DocumentID != "" {
			document = " · " + entry.DocumentID
			if title := documentTitles[entry.DocumentID]; title != "" {
				document = " · " + title
			}
		}
		view.Entries = append(view.Entries, web.AnnualStatementReserveEntryView{
			Date: date, Kind: annualStatementReserveKindLabel(entry.Kind), Amount: formatAnnualStatementMoney(entry.AmountCents), Note: note, Document: document,
		})
	}
	return view, true
}

func annualStatementReserveKindLabel(kind string) string {
	switch kind {
	case store.ReserveKindOpening:
		return "Anfangsstand"
	case store.ReserveKindContribution:
		return "Zuführung"
	case store.ReserveKindWithdrawal:
		return "Entnahme"
	case store.ReserveKindInterest:
		return "Zinsen"
	case store.ReserveKindClosingCheck:
		return "Endstand-Kontrolle"
	default:
		return kind
	}
}

func annualStatementReserveMessage(code string) (string, bool) {
	switch code {
	case "saved":
		return "Buchung hinzugefügt.", true
	case "invalid":
		return "Die Buchung konnte nicht gespeichert werden. Entnahmen brauchen einen Beleg, und die Rücklage gilt nur für WEG.", false
	default:
		return "", false
	}
}

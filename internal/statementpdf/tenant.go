package statementpdf

import (
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/pdf"
	"github.com/inspr-at/hausv-org/internal/store"
)

// TenantDocument uses the same letter layout as the owner statement, with its
// own legal terms. A tenant never receives the owner's complete WEG statement.
func TenantDocument(s store.TenantStatement, accountID string) (Document, error) {
	var account *store.TenantStatementAccount
	for i := range s.Accounts {
		if s.Accounts[i].ID == accountID {
			account = &s.Accounts[i]
		}
	}
	if account == nil {
		return Document{}, ErrNotFound
	}
	a := *account
	run := s.Source
	run.Approval = s.Approval
	run.CreatedAt, _ = time.Parse(time.DateOnly, s.StatementOn)
	d := letterDocument(run, s.UnitLabel)
	d.UnitID, d.PartyID = s.UnitID, a.ID
	d.Title = fmt.Sprintf("Mieter-Betriebskostenabrechnung %d", run.PeriodYear)
	d.Reference = "Ref. " + s.ID
	d.Basis = nil
	d.Address = []string{"Mietpartei"}
	for _, p := range a.Parties {
		d.Address = append(d.Address, p.Name)
		if p.Address != "" {
			d.Address = append(d.Address, p.Address)
		}
	}
	if a.Landlord {
		d.Address = []string{s.Owner.Name, s.Owner.Address}
		d.Basis = append(d.Basis, "Leerstand: Ausgleich verbleibt beim Eigentümer.")
	}
	d.Info = nil
	estate := nonempty(run.Input.Presentation.EstateName, run.Input.Presentation.EstateAddress)
	if len(estate) == 2 && strings.HasPrefix(estate[1], estate[0]) {
		estate = estate[1:] // The address already starts with the estate name.
	}
	d.Info = append(d.Info, InfoField{Label: "Liegenschaft", Value: strings.Join(estate, "\n")}, InfoField{Label: "Im Namen und auf Rechnung von", Value: s.Owner.Name}, InfoField{Label: "Einheit", Value: s.UnitLabel}, InfoField{Label: "Abrechnungsperiode", Value: date(run.Input.Period.StartsOn) + " bis " + date(run.Input.Period.EndsOn)}, InfoField{Label: "Abrechnungsdatum", Value: date(s.StatementOn)})
	for _, line := range a.Lines {
		row := CostRow{Name: line.Name, Total: money(line.SourceGrossCents), Key: "WEG-Ableitung", Share: "siehe Hinweise", Amount: money(line.GrossCents), Net: money(line.NetCents), Rate: fmt.Sprintf("%d %%", line.VATRate), VAT: money(line.VATCents), Gross: money(line.GrossCents)}
		if line.Heating {
			for _, unit := range run.Result.Units {
				if unit.UnitID != s.UnitID {
					continue
				}
				for _, cost := range unit.Costs {
					if cost.CostTypeKey != line.Key {
						continue
					}
					details := heatingDetails(run, unit, cost, 0, false)
					// The source unit's measurement and area pools are the basis;
					// only this tenant's lease Akonto belongs in the settlement.
					row.Measurements = append(row.Measurements, details[:7]...)
					for _, interval := range a.Intervals {
						row.Measurements = append(row.Measurements, "Mietzeitraum: "+date(interval.From)+" bis "+date(lastDay(interval.To)))
					}
					row.Measurements = append(row.Measurements, "Abgeleiteter Heizkostenanteil dieser Mietpartei: "+money(line.GrossCents))
					row.Measurements = append(row.Measurements, details[8:]...)
				}
			}
		}
		d.Costs = append(d.Costs, row)
		d.ShowVAT = true
	}
	d.Total = money(a.OperatingCents + a.HeatingCents)
	d.Prepaid = money(a.OperatingPrepaidCents + a.HeatingPrepaidCents)
	d.Balance = settlement(a.BalanceCents())
	d.PaymentTerms = []string{fmt.Sprintf("Betriebskosten %s abzüglich BK-Akonto laut Mietvertrag %s: %s", money(a.OperatingCents), money(a.OperatingPrepaidCents), settlement(a.OperatingCents-a.OperatingPrepaidCents)), fmt.Sprintf("Heizkosten %s abzüglich Heizkosten-Akonto laut Mietvertrag %s: %s", money(a.HeatingCents), money(a.HeatingPrepaidCents), settlement(a.HeatingCents-a.HeatingPrepaidCents))}
	if a.Scope == store.MRGVoll || a.Landlord {
		d.PaymentTerms = append(d.PaymentTerms, fmt.Sprintf("Betriebskosten gemäß § 21 Abs. 3 MRG: Abrechnung bis 30. Juni %d. Guthaben oder Nachzahlung zum übernächsten Zinstermin, am %s. Maßgeblich ist die Mietpartei am Fälligkeitstag; keine zeitanteilige BK-Aufteilung bei Mieterwechsel.", run.PeriodYear+1, date(a.OperatingDueOn)))
		d.InspectionBasis = "§ 21 Abs. 3 MRG"
	} else {
		d.PaymentTerms = append(d.PaymentTerms, "Betriebskosten laut vereinbartem Kostenkatalog. Vertragliche Fälligkeit: "+date(a.OperatingDueOn)+". Die gesetzliche MRG-Frist 30. Juni gilt in Teilanwendung/Ausnahme nicht.")
		d.InspectionBasis = "Vertrag / ABGB"
	}
	if run.Input.Structure.Legal.HeizKGApplies {
		d.PaymentTerms = append(d.PaymentTerms, "HeizKG: Guthaben und Nachzahlung binnen zwei Monaten, bis "+date(a.HeatingDueOn)+". Abnehmerwechsel gemäß § 23 HeizKG: gleiche Monatsanteile ohne Zwischenermittlung; Teilmonate nach Tagen im jeweiligen Monat.", "Einwendungen gegen die Heizkostenabrechnung binnen sechs Monaten (§ 24 HeizKG). Die HeizKG-Abrechnung und Verbrauchsnachweise liegen zur Belegeinsicht bereit.")
	}
	legal := run.Input.Structure.Legal
	d.Inspection = []string{legal.InspectionPlace, legal.InspectionPeriod, legal.InspectionContact, "Auflage der Abrechnung im Haus; Belegkopien auf Verlangen gegen Kostenersatz."}
	d.Excluded = []string{"Rücklage wird nicht an die Mietpartei überwälzt."}
	for _, c := range s.Excluded {
		d.Excluded = append(d.Excluded, c.Name+": beim Eigentümer verbleibend "+money(c.AmountCents))
	}
	d.PaymentTerms = append(d.PaymentTerms, "Abgleich mit WEG: überwälzter Quellbetrag "+money(s.SourcePassedCents)+" + beim Eigentümer verbleibend "+money(s.SourceRetainedCents)+". Umsatzsteuer wird nach Nutzung und Option des Mietvertrags berechnet.")
	return d, nil
}

func RenderTenant(s store.TenantStatement, accountID string) ([]byte, error) {
	var pages []pdf.Page
	for _, a := range s.Accounts {
		if accountID != "" && a.ID != accountID {
			continue
		}
		if len(a.Lines) == 0 && a.OperatingPrepaidCents == 0 && a.HeatingPrepaidCents == 0 {
			continue
		}
		d, err := TenantDocument(s, a.ID)
		if err != nil {
			return nil, err
		}
		pages = append(pages, d.Pages()...)
	}
	if len(pages) == 0 {
		return nil, ErrNotFound
	}
	return pdf.Pages(pages, statementPalette), nil
}

// settlement names a tenant balance instead of printing a signed saldo.
func settlement(cents int64) string {
	switch {
	case cents > 0:
		return "Nachzahlung " + money(cents)
	case cents < 0:
		return "Guthaben " + money(-cents)
	default:
		return "Ausgeglichen " + money(0)
	}
}

// lastDay turns a stored exclusive interval end into the last included day.
func lastDay(exclusive string) string {
	at, err := time.Parse(time.DateOnly, exclusive)
	if err != nil {
		return exclusive
	}
	return at.AddDate(0, 0, -1).Format(time.DateOnly)
}

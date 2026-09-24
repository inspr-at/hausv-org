package statementpdf

import (
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/pdf"
	"github.com/inspr-at/hausv-org/internal/store"
)

type InfoField struct{ Label, Value string }

var statementPalette = pdf.Palette{Paper: [3]uint8{255, 255, 255}, Ink: [3]uint8{32, 37, 31}, Accent: [3]uint8{170, 146, 93}}

func letterDocument(run store.AnnualStatementRun, unit string) Document {
	p := run.Input.Presentation
	org := p.Organisation
	if strings.TrimSpace(org) == "" {
		org = "Hausverwaltung: Angabe fehlt"
	}
	address := p.ContactAddress
	if strings.TrimSpace(address) == "" {
		address = "Verwaltungsanschrift fehlt"
	}
	d := Document{
		UnitLabel: unit,
		Title:     fmt.Sprintf("Jahresabrechnung %d — %s", run.PeriodYear, unit),
		Sender:    nonempty(org, address, p.ContactName, strings.Join(nonempty(p.ContactPhone, p.ContactEmail), " · ")),
		Info: []InfoField{
			{"Liegenschaft", strings.Join(nonempty(p.EstateName, p.EstateAddress), "\n")},
			{"Einheit / Top", unit},
			{"Abrechnungsperiode", date(run.Input.Period.StartsOn) + " bis " + date(run.Input.Period.EndsOn)},
			{"Abrechnungsdatum", statementDate(run).Format("02.01.2006")},
		},
		Contact:   strings.Join(nonempty(org, p.ContactPhone, p.ContactEmail), " · "),
		Reference: fmt.Sprintf("Ref. %s · Revision %d · Erstellt: %s", run.ID, run.Revision, timestamp(run.CreatedAt)),
	}
	if basis := run.Input.Structure.Legal.Basis(); run.Input.Structure.Legal.Regime != "" {
		d.Info = append(d.Info, InfoField{"Rechtsgrundlage", basis})
	}
	// References label the existing inspection instructions; they do not change
	// the stored place, period or contact (RIS WEG §34, MRG §21, HeizKG §19).
	var inspectionRefs []string
	switch run.Input.Structure.Legal.Regime {
	case "weg":
		inspectionRefs = append(inspectionRefs, "§ 34 Abs. 1 WEG")
	case "mrg_voll":
		inspectionRefs = append(inspectionRefs, "§ 21 Abs. 3 MRG")
	}
	if run.Input.Structure.Legal.HeizKGApplies {
		inspectionRefs = append(inspectionRefs, "§ 19 Abs. 3 HeizKG")
	}
	d.InspectionBasis = strings.Join(inspectionRefs, "; ")
	if run.Approval != nil {
		role := "Verwaltung"
		if run.Approval.Role == store.RoleAdmin {
			role = "Administration"
		}
		d.ApprovalNotice = "Freigegeben: " + timestamp(run.Approval.ApprovedAt) + " · " + role
	}
	return d
}

// The short summary reflects the existing terms, including separate deadlines
// for MRG operating costs and heating. The complete wording follows in Hinweise.
func balanceTiming(run store.AnnualStatementRun, unit store.AnnualStatementRunUnit) string {
	legal, at := run.Input.Structure.Legal, statementDate(run)
	var timing []string
	switch legal.Regime {
	case "weg":
		if unit.BalanceCents < 0 {
			timing = append(timing, "WEG: Anrechnung auf künftige Vorauszahlungen")
		} else if unit.BalanceCents > 0 {
			timing = append(timing, "WEG: fällig bis "+store.ShiftStatementDate(at, 2).Format("02.01.2006"))
		}
	case "mrg_voll":
		months := time.Month(1)
		if at.Day() >= 5 {
			months = 2
		}
		due := time.Date(at.Year(), at.Month()+months, 5, 0, 0, 0, 0, at.Location())
		timing = append(timing, "Betriebskosten: Ausgleich am "+due.Format("02.01.2006"))
	case "mrg_teil", "ausnahme":
		timing = append(timing, "Betriebskosten: Fälligkeit laut Vertrag")
	}
	if legal.HeizKGApplies {
		timing = append(timing, "HeizKG-Anteil: Ausgleich bis "+store.ShiftStatementDate(at, 2).Format("02.01.2006"))
	}
	return strings.Join(timing, " · ")
}

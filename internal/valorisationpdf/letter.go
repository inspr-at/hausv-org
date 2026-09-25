// Package valorisationpdf renders the five variants from a frozen run.
package valorisationpdf

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/pdf"
	"github.com/inspr-at/hausv-org/internal/store"
)

func Money(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	s := fmt.Sprint(cents / 100)
	var b strings.Builder
	for j, c := range s {
		if j > 0 && (len(s)-j)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return sign + b.String() + fmt.Sprintf(",%02d €", cents%100)
}
func Date(raw string) string {
	d, err := time.Parse(time.DateOnly, raw)
	if err != nil {
		return raw
	}
	return d.Format("02.01.2006")
}
func Variant(item store.ValorisationItem) string {
	if item.NewCents < item.OldCents {
		return "Verminderung"
	}
	if item.MieWeG {
		if item.Lease.MRGScope == store.MRGVoll {
			return "MieWeG · Vollanwendung"
		}
		return "MieWeG · Teilanwendung"
	}
	if item.Lease.MRGScope == store.MRGVoll {
		return "Vertrag · Vollanwendung"
	}
	return "Vertrag · Teilanwendung / Ausnahme"
}

// Letterhead geometry is shared with annual statements; calculation tables
// and their pagination remain specific to this letter.
type layout struct {
	pages []pdf.Page
	y     float64
	run   store.ValorisationRun
	item  store.ValorisationItem
}

func (l *layout) text(x, y, size float64, style pdf.Style, text string) {
	p := &l.pages[len(l.pages)-1]
	p.Lines = append(p.Lines, pdf.Line{Text: text, Style: style, X: x, Y: y, Size: size})
}
func (l *layout) newPage() {
	l.pages = append(l.pages, pdf.Page{Positioned: true})
	l.y = 750
	l.text(62, 790, 9, pdf.Strong, l.run.Input.Organisation)
	l.text(62, 770, 8, pdf.Body, "Wertsicherung · "+l.item.Label()+" · "+Date(l.run.EffectiveOn))
}
func (l *layout) room(height float64) {
	if l.y-height < 108 {
		l.newPage()
	}
}
func (l *layout) paragraph(text string, style pdf.Style) {
	lines := pdf.WrapWidth(text, style, 10, 450)
	for _, line := range lines {
		l.room(14)
		l.text(62, l.y, 10, style, line)
		l.y -= 14
	}
	l.y -= 6
}
func (l *layout) heading(text string) { l.room(50); l.y -= 6; l.paragraph(text, pdf.Strong) }
func (l *layout) row(label, old, next string) {
	lines := pdf.WrapWidth(label, pdf.Body, 9, 255)
	l.room(float64(len(lines))*13 + 7)
	y := l.y
	for _, line := range lines {
		l.text(62, l.y, 9, pdf.Body, line)
		l.y -= 13
	}
	l.text(410-pdf.TextWidth(old, pdf.Body, 9), y, 9, pdf.Body, old)
	l.text(533-pdf.TextWidth(next, pdf.Body, 9), y, 9, pdf.Body, next)
	l.y -= 5
}
func Render(run store.ValorisationRun, item store.ValorisationItem, letterDate time.Time) ([]byte, error) {
	if item.NewCents <= 0 || item.WirksamOn == "" || len(item.Recipients) == 0 {
		return nil, fmt.Errorf("Schreiben benötigt berechnete Anpassung und Empfänger")
	}
	if run.Status != "draft" && letterDate.Format(time.DateOnly) < item.WirksamOn {
		return nil, fmt.Errorf("Schreiben vor Wirksamkeit unzulässig")
	}
	l := layout{run: run, item: item}
	l.newPage()
	l.pages[0].Lines = nil
	sender := run.Input.Settings.LetterSenderText
	if sender == "" {
		sender = run.Input.Organisation + " · im Auftrag des Vermieters"
	}
	address := []string{item.Recipients[0].Name, item.Recipients[0].Address}
	info := []pdf.LetterInfoField{
		{Label: "Datum", Value: letterDate.Format("02.01.2006")},
		{Label: "Einheit / Hauptmieter", Value: item.Label()},
		{Label: "Vertragsabschluss", Value: Date(item.Lease.ConcludedOn)},
	}
	var overflow []string
	l.y, overflow = pdf.Letterhead(&l.pages[0], []string{sender}, address, info)
	title := "Anpassung des Hauptmietzinses"
	if item.NewCents < item.OldCents {
		title = "Verminderung des Hauptmietzinses"
	}
	l.text(62, l.y, 20, pdf.Heading, title)
	l.y -= 25
	house := run.Input.House
	if run.Input.Address != "" {
		if house == "" || strings.Contains(run.Input.Address, house) {
			house = run.Input.Address
		} else {
			house += " · " + run.Input.Address
		}
	}
	l.paragraph("Wertsicherung · "+house, pdf.Body)
	if run.Status == "draft" {
		l.paragraph("Entwurf zur Prüfung", pdf.Strong)
	}
	l.paragraph("Sehr geehrte Damen und Herren,", pdf.Body)
	l.paragraph("auf Grundlage Ihrer vertraglichen Wertsicherung passen wir Ihren monatlichen Hauptmietzins wie folgt an.", pdf.Body)
	l.room(85)
	page := &l.pages[len(l.pages)-1]
	page.Shapes = append(page.Shapes, pdf.Shape{X: 62, Y: l.y - 67, Width: 471, Height: 76, Color: [3]uint8{242, 246, 243}})
	l.text(76, l.y-9, 9, pdf.Body, "Hauptmietzins netto pro Monat")
	l.text(76, l.y-36, 18, pdf.Strong, Money(item.OldCents)+"  auf  "+Money(item.NewCents))
	l.text(76, l.y-55, 9, pdf.Body, "Wirksam ab "+Date(item.WirksamOn)+" · erstmals fällig am "+Date(item.CollectableFrom))
	l.y -= 87
	if len(item.Recipients) > 1 {
		l.heading("Vertragsparteien")
		for _, party := range item.Recipients {
			l.paragraph(party.Name+" · "+party.Address, pdf.Body)
		}
	}
	l.heading("Vertragliche Grundlage")
	if item.Clause.ClauseType == store.ClauseStaffel {
		l.paragraph("Staffelmietzins laut Vertrag", pdf.Strong)
		for _, line := range item.Explanation {
			if strings.HasPrefix(line, "Staffelmietzins laut Vertrag") || strings.HasPrefix(line, "MieWeG begrenzt die Staffel") {
				l.paragraph(line, pdf.Body)
			}
		}
	}
	l.paragraph(item.Clause.ClauseText, pdf.Body)
	if item.Clause.BasePeriod != "" {
		series := strings.Replace(strings.ToUpper(item.Clause.Series), "VPI", "VPI ", 1)
		l.paragraph(fmt.Sprintf("%s, Basis %s: %s · Schwelle %s %s", series, monthName(item.Clause.BasePeriod), strings.ReplaceAll(item.Clause.BaseValue, ".", ","), store.ValorisationNumber(item.Clause.ThresholdValue, 2), thresholdKind(item.Clause.ThresholdKind)), pdf.Body)
	}
	if item.Contract.TriggerMonth != "" {
		l.paragraph(fmt.Sprintf("Auslösemonat %s: endgültiger Index %s; Änderung %s. Vertraglicher Betrag: %s.", monthName(string(item.Contract.TriggerMonth)), displayRatio(item.Contract.NewBase.String()), displayPercent(item.Contract.ChangePercent.String(), false), Money(item.ContractCents)), pdf.Body)
	}
	if item.MieWeG {
		l.heading("Gesetzliche Vergleichsrechnung (MieWeG)")
		l.paragraph("Anker: "+monthName(string(item.CapAnchor))+". Über 3 % wird die weitere Jahresveränderung zur Hälfte berücksichtigt. Im ersten Teiljahr zählen nur volle Monate nach dem Ankermonat.", pdf.Body)
		if item.SpecialCap {
			l.paragraph("Sonderdeckel: höchstens 1 % für 2026 und 2 % für 2027, jeweils vor der Aliquotierung.", pdf.Body)
		}
		l.row("Jahr · Jahresmittel alt / neu", "Monate / 12", "Kurvenwert (gerundet)")
		for _, step := range item.Ceiling.Years {
			l.row(fmt.Sprintf("%d · %s / %s", step.Year, displayRatio(step.PreviousAverage.String()), displayRatio(step.CurrentAverage.String())), fmt.Sprintf("%d / 12", step.FullMonths), store.ValorisationExactMoney(step.ExactAmountCents))
			l.paragraph("Änderung "+displayPercent(step.RawRatePercent, true)+"; begrenzt "+displayPercent(step.LimitedRatePercent, true)+".", pdf.Body)
		}
		l.row("Vertragskurve / gesetzliche Kurve", Money(item.ContractCents), Money(item.CapCents))
		l.paragraph("Maßgeblich ist der niedrigere Betrag (§ 1 Abs 4 MieWeG). Ein halber Cent wird abgerundet; die Kurven werden ohne Zwischenrundung fortgeführt.", pdf.Body)
	} else {
		l.paragraph("MieWeG ist nicht anwendbar. Die vertragliche Anpassung wird am "+Date(item.WirksamOn)+" wirksam.", pdf.Body)
	}
	total := item.NewGrossCents
	latest := map[string]store.RentComponent{}
	for _, c := range item.Lease.Components {
		if c.Kind != store.ComponentHMZ && c.ValidFrom <= run.EffectiveOn && c.ValidFrom >= latest[c.Kind].ValidFrom {
			latest[c.Kind] = c
		}
	}
	l.room(150 + float64(len(latest))*18)
	l.heading("Monatliche Vorschreibung")
	l.row("Bestandteil", "Bisher", "Neu")
	l.row("Hauptmietzins netto", Money(item.OldCents), Money(item.NewCents))
	l.row(fmt.Sprintf("Umsatzsteuer Hauptmietzins (%s %%)", displayRatio(fmt.Sprintf("%d/100", item.VATRateBP))), Money(item.OldVATCents), Money(item.NewVATCents))
	l.row("Hauptmietzins brutto", Money(item.OldGrossCents), Money(item.NewGrossCents))
	for _, kind := range []string{store.ComponentBKAkonto, store.ComponentHeizAkonto, store.ComponentLift, store.ComponentMoebel, store.ComponentStellplatz, store.ComponentSonstiges} {
		if c, ok := latest[kind]; ok {
			gross := c.NetCents + store.ValorisationVAT(c.NetCents, c.VATRateBP)
			total += gross
			l.row(componentLabel(kind)+" (unverändert, brutto)", Money(gross), Money(gross))
		}
	}
	l.row("Monatlich gesamt brutto", "", Money(total))
	l.paragraph("Änderung des Hauptmietzinses netto: "+Money(item.NewCents-item.OldCents)+" ("+displayPercent(fmt.Sprintf("%d/%d", (item.NewCents-item.OldCents)*100, item.OldCents), true)+").", pdf.Body)
	l.room(170)
	l.heading("Termine und Hinweise")
	if item.RequiresMRGNotice {
		l.paragraph("Der angepasste Hauptmietzins wird gemäß § 16 Abs 9 MRG erstmals am "+Date(item.CollectableFrom)+" fällig, sofern Ihnen dieses Schreiben spätestens am "+Date(item.NoticeDeadline)+" zugeht (mindestens 14 Tage vor dem Zinstermin). Bei späterem Zugang verschiebt sich die Fälligkeit auf den nächsten zulässigen Zinstermin.", pdf.Body)
	} else {
		l.paragraph("Die Anpassung ist ab "+Date(item.WirksamOn)+" wirksam. Erster Zinstermin: "+Date(item.CollectableFrom)+".", pdf.Body)
	}
	if item.OverrideCents != nil {
		l.paragraph("Manuelle Entscheidung: "+item.OverrideReason+". Freigegeben durch "+item.OverrideBy+".", pdf.Strong)
	}
	l.paragraph("Betriebskosten- und Heizungsakonti werden durch diese Wertsicherung nicht erhöht.", pdf.Body)
	if run.Input.Contact != "" {
		l.paragraph("Für Rückfragen: "+run.Input.Contact, pdf.Body)
	} else {
		l.paragraph("Für Rückfragen wenden Sie sich bitte an Ihre Hausverwaltung.", pdf.Body)
	}
	l.paragraph("Mit freundlichen Grüßen", pdf.Body)
	l.paragraph(run.Input.Organisation+" · im Auftrag des Vermieters", pdf.Body)
	l.paragraph("Datenquelle: Statistik Austria · data.statistik.gv.at (CC BY 4.0). Berechnung: HAUSV. Datenstand "+run.IndexVersion+".", pdf.Body)
	if len(overflow) > 0 {
		l.heading("Ergänzende Adress- und Verwaltungsangaben")
		for _, line := range overflow {
			l.paragraph(line, pdf.Body)
		}
	}
	for j := range l.pages {
		l.pages[j].Footer = []string{fmt.Sprintf("%s · Wertsicherung · %s · Seite %d von %d", run.Input.Organisation, item.Label(), j+1, len(l.pages))}
	}
	return pdf.Pages(l.pages, pdf.Palette{Paper: [3]uint8{255, 255, 255}, Ink: [3]uint8{32, 43, 39}, Accent: [3]uint8{97, 118, 107}}), nil
}
func displayRatio(raw string) string { return store.ValorisationNumber(raw, 5) }

// Letter formatting never changes the exact values saved with the run.
func displayPercent(raw string, signed bool) string {
	r, ok := new(big.Rat).SetString(raw)
	if !ok {
		return "–"
	}
	value := strings.ReplaceAll(r.FloatString(2), ".", ",")
	if signed && r.Sign() > 0 {
		value = "+" + value
	}
	return value + " %"
}

func monthName(raw string) string {
	d, err := time.Parse("2006-01", raw)
	if err != nil {
		return raw
	}
	months := [...]string{"", "Jänner", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"}
	return months[d.Month()] + " " + d.Format("2006")
}
func componentLabel(kind string) string {
	switch kind {
	case store.ComponentBKAkonto:
		return "BK-Akonto"
	case store.ComponentHeizAkonto:
		return "Heizungsakonto"
	case store.ComponentStellplatz:
		return "Stellplatz"
	case store.ComponentMoebel:
		return "Möbel"
	case store.ComponentLift:
		return "Lift"
	}
	return "Sonstiges"
}

func thresholdKind(kind string) string {
	if kind == "points" {
		return "Punkte"
	}
	return "%"
}

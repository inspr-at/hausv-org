package statementpdf

import (
	"fmt"
	"strings"

	"github.com/inspr-at/hausv-org/internal/pdf"
)

const (
	left    = 62.0
	measure = 471.0 // 166 mm, with equal A4 side margins.
	bottom  = 104.0
	leading = 12.0
)

type letterLayout struct {
	d        Document
	pages    []pdf.Page
	y        float64
	overflow []string
}

func (l *letterLayout) page() *pdf.Page { return &l.pages[len(l.pages)-1] }
func (l *letterLayout) text(text string, style pdf.Style, size, x, y float64) {
	l.page().Lines = append(l.page().Lines, pdf.Line{Text: text, Style: style, Size: size, X: x, Y: y})
}
func (l *letterLayout) rule(y float64) {
	l.page().Shapes = append(l.page().Shapes, pdf.Shape{X: left, Y: y, Width: measure, Height: .5, Color: [3]uint8{216, 214, 207}})
}
func (l *letterLayout) newPage(first bool) {
	l.pages = append(l.pages, pdf.Page{Positioned: true})
	if l.d.ApprovalNotice == "" {
		l.page().Shapes = append(l.page().Shapes, pdf.Shape{X: left, Y: 814, Width: measure, Height: 17, Color: [3]uint8{244, 235, 215}})
		l.text("ENTWURF · zur Prüfung", pdf.Strong, 8, left+8, 819)
	}
	if !first {
		if len(l.d.Sender) > 0 {
			l.text(pdf.WrapWidth(l.d.Sender[0], pdf.Strong, 9, measure)[0], pdf.Strong, 9, left, 797)
		}
		l.y = 768
		// Bound the repeated header even when supplied labels span pages.
		title := pdf.WrapWidth(l.d.Title+" · Fortsetzung", pdf.Heading, 16, measure)
		for _, line := range title[:min(2, len(title))] {
			l.text(line, pdf.Heading, 16, left, l.y)
			l.y -= 19
		}
		l.rule(l.y)
		l.y -= 22
		return
	}
	l.y, l.overflow = pdf.Letterhead(l.page(), l.d.Sender, l.d.Address, l.d.Info)
	l.paragraph(l.d.Title, pdf.Heading, 22, 25)
	l.y -= 6
}

func (l *letterLayout) ensure(height float64) {
	if l.y-height < bottom {
		l.newPage(false)
	}
}
func (l *letterLayout) paragraph(text string, style pdf.Style, size, gap float64) {
	for _, line := range pdf.WrapWidth(text, style, size, measure) {
		l.ensure(gap)
		l.text(line, style, size, left, l.y)
		l.y -= gap
	}
}
func (l *letterLayout) section(title string) {
	l.ensure(46)
	l.y -= 8
	l.paragraph(title, pdf.Strong, 11, 14)
	l.y -= 5
}

func (d Document) Pages() []pdf.Page {
	l := &letterLayout{d: d}
	l.d.Sender, l.d.Address = nonempty(d.Sender...), nonempty(d.Address...)
	l.newPage(true)
	l.summary()
	l.costTable()
	if len(d.Basis) > 0 {
		l.y -= 5
		l.paragraph(strings.Join(d.Basis, " · "), pdf.Body, 8.5, 11)
	}
	if len(d.Reserve) > 0 {
		l.section("Rücklage")
		for _, text := range d.Reserve {
			l.paragraph(text, pdf.Body, 9.5, 13)
		}
	}
	var objections []string
	for _, row := range d.Costs {
		for _, text := range row.Measurements {
			if strings.HasPrefix(text, "Einwendungen sind") && !contains(objections, text) {
				objections = append(objections, text)
			}
		}
	}
	if len(d.PaymentTerms)+len(d.Inspection)+len(objections) > 0 {
		l.section("Hinweise")
		for _, text := range d.PaymentTerms {
			if strings.HasPrefix(text, "Abrechnungsdatum:") {
				continue
			}
			l.paragraph(text, pdf.Body, 9, leading)
			l.y -= 5
		}
		if len(d.Inspection) > 0 {
			label := "Einsicht in die Belege"
			if d.InspectionBasis != "" {
				label += " (" + d.InspectionBasis + ")"
			}
			l.paragraph(label+" · "+strings.Join(d.Inspection, " · "), pdf.Body, 9, leading)
			l.y -= 5
		}
		for _, text := range objections {
			l.paragraph(text, pdf.Body, 9, leading)
			l.y -= 5
		}
	}
	annex := false
	for _, row := range d.Costs {
		if len(row.Measurements) == 0 {
			continue
		}
		if !annex {
			l.newPage(false)
			annex = true
		}
		l.section(row.Name + " · Messnachweis")
		for _, text := range row.Measurements {
			if strings.HasPrefix(text, "Einwendungen sind") {
				continue
			} // Already in Hinweise.
			l.paragraph(text, pdf.Body, 9.5, 13)
			l.y -= 4
		}
	}
	if len(d.Excluded) > 0 {
		l.section("Hinweis: Nicht umlagefähige Kosten")
		for _, text := range d.Excluded {
			l.paragraph(text, pdf.Body, 9.5, 13)
		}
	}
	if len(d.Receipts) > 0 {
		l.section("Belegverzeichnis")
		for _, text := range d.Receipts {
			l.paragraph(text, pdf.Body, 9.5, 13)
			l.y -= 6
		}
	}
	if len(l.overflow) > 0 {
		l.section("Ergänzende Adress- und Verwaltungsangaben")
		for _, text := range l.overflow {
			l.paragraph(text, pdf.Body, 9, leading)
		}
	}
	notice := d.ApprovalNotice
	if notice == "" {
		notice = DraftNotice
	}
	contact := nonempty(pdf.WrapWidth(d.Contact, pdf.Body, 7, measure)...)
	if len(contact) > 2 {
		contact = []string{"Verwaltung: Kontaktdaten siehe Briefkopf und ergänzende Verwaltungsangaben."}
	}
	for i := range l.pages {
		footer := append([]string{notice}, contact...)
		ref := pdf.WrapWidth(d.Reference, pdf.Body, 7, measure-80)
		reference := ""
		if len(ref) > 0 {
			reference = ref[0] + " · "
		}
		footer = append(footer, fmt.Sprintf("%sSeite %d / %d", reference, i+1, len(l.pages)))
		l.pages[i].Footer = footer
	}
	return l.pages
}

func contains(values []string, value string) bool {
	for _, existing := range values {
		if existing == value {
			return true
		}
	}
	return false
}

func (l *letterLayout) summary() {
	d := l.d
	type entry struct {
		text      string
		style     pdf.Style
		size, gap float64
	}
	entries := []entry{{"Kurzfassung", pdf.Strong, 11, 21}}
	label := "Gesamtkosten Einheit: "
	if d.PartyID == "" {
		label = "Gesamtkosten Liegenschaft: "
	}
	entries = append(entries, entry{label + d.Total, pdf.Body, 10, 16})
	if d.Prepaid != "" {
		entries = append(entries, entry{"Geleistete Vorauszahlungen: " + d.Prepaid, pdf.Body, 10, 19})
	}
	if d.Balance != "" {
		entries = append(entries, entry{"Ergebnis · " + d.Balance, pdf.Strong, 14, 20})
	}
	if d.Timing != "" {
		entries = append(entries, entry{d.Timing, pdf.Body, 8.5, 12})
	}
	for _, text := range d.Proposals {
		entries = append(entries, entry{text, pdf.Body, 9, 12})
	}
	l.ensure(110)
	start, page := l.y+13, len(l.pages)-1
	shade := func() {
		l.pages[page].Shapes = append(l.pages[page].Shapes, pdf.Shape{X: left, Y: l.y - 1, Width: measure, Height: start - l.y + 1, Color: [3]uint8{247, 245, 239}})
	}
	for _, entry := range entries {
		for _, line := range pdf.WrapWidth(entry.text, entry.style, entry.size, measure-24) {
			if l.y-entry.gap < bottom {
				shade()
				l.newPage(false)
				start, page = l.y+13, len(l.pages)-1
			}
			l.text(line, entry.style, entry.size, left+12, l.y)
			l.y -= entry.gap
		}
	}
	shade()
	l.y -= 24
}

func (l *letterLayout) costTable() {
	if len(l.d.Costs) == 0 {
		return
	}
	widths := []float64{153, 87, 83, 65, 83}
	headings := []string{"Kostenart", "Gesamtkosten", "Schlüssel", "Anteil", "Betrag"}
	if l.d.ShowVAT {
		widths = []float64{151, 80, 70, 80, 90}
		headings = []string{"Kostenart", "Netto", "USt-Satz", "USt", "Brutto"}
	} else if l.d.PartyID == "" {
		widths, headings = []float64{360, 111}, []string{"Kostenart", "Betrag"}
	}
	header := func() {
		x := left
		for i, label := range headings {
			xText := x + 5
			if l.d.ShowVAT && i > 0 || !l.d.ShowVAT && (i == 1 || i >= 3) {
				xText = x + widths[i] - 5 - pdf.TextWidth(label, pdf.Strong, 9)
			}
			l.text(label, pdf.Strong, 9, xText, l.y)
			x += widths[i]
		}
		l.rule(l.y - 7)
		l.y -= 23
	}
	l.ensure(52)
	header()
	for _, row := range l.d.Costs {
		cells := []string{row.Name, row.Total, row.Key, row.Share, row.Amount}
		if l.d.ShowVAT {
			cells = []string{row.Name, row.Net, row.Rate, row.VAT, row.Gross}
		} else if l.d.PartyID == "" {
			cells = []string{row.Name, row.Amount}
		}
		wrapped, count := make([][]string, len(cells)), 1
		for i, cell := range cells {
			wrapped[i] = pdf.WrapWidth(cell, pdf.Body, 9, widths[i]-10)
			count = max(count, len(wrapped[i]))
		}
		height := float64(count)*12 + 10
		if l.y-height < bottom && height < 570 {
			l.newPage(false)
			header()
		}
		for n := 0; n < count; n++ {
			if l.y-22 < bottom {
				l.newPage(false)
				header()
			}
			x := left
			for i, lines := range wrapped {
				if n < len(lines) {
					xText := x + 5
					if l.d.ShowVAT && i > 0 || !l.d.ShowVAT && (i == 1 || i >= 3) {
						xText = x + widths[i] - 5 - pdf.TextWidth(lines[n], pdf.Body, 9)
					}
					l.text(lines[n], pdf.Body, 9, xText, l.y)
				}
				x += widths[i]
			}
			l.y -= 12
		}
		l.rule(l.y + 3)
		l.y -= 10
	}
	if l.y-30 < bottom {
		l.newPage(false)
		header()
	}
	l.text("Summe", pdf.Strong, 10, left+5, l.y)
	for _, line := range pdf.WrapWidth(l.d.Total, pdf.Strong, 10, measure-110) {
		l.text(line, pdf.Strong, 10, left+measure-5-pdf.TextWidth(line, pdf.Strong, 10), l.y)
		l.y -= 13
	}
	for _, line := range l.d.VATSummary {
		l.ensure(16)
		l.text(line, pdf.Body, 9, left+5, l.y)
		l.y -= 14
	}
	l.y -= 8
}

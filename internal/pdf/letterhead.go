package pdf

import "strings"

// LetterInfoField is shared by annual statements and rent adjustment letters.
type LetterInfoField struct{ Label, Value string }

type letterheadLayout struct {
	page     *Page
	overflow []string
}

func (l *letterheadLayout) text(text string, style Style, size, x, y float64) {
	l.page.Lines = append(l.page.Lines, Line{Text: text, Style: style, Size: size, X: x, Y: y})
}

// Letterhead uses the reviewed letter-window grid. Long supplied content is
// returned for a paginated annex instead of colliding with the body or footer.
func Letterhead(page *Page, sender, address []string, info []LetterInfoField) (bodyY float64, overflow []string) {
	const left = 62.0
	l := &letterheadLayout{page: page}
	// The recipient starts 49 mm below the page edge, within a letter window.
	l.fixedBlock(sender, left, 797, 257, 9, 11, 6)
	addressBottom := l.fixedBlock(address, left, 703, 250, 10, 13, 8)
	infoY := 796.0
	for _, field := range info {
		if strings.TrimSpace(field.Value) == "" {
			continue
		}
		if infoY < 612 {
			l.overflow = append(l.overflow, field.Label+": "+field.Value)
			continue
		}
		l.text(field.Label, Strong, 7.5, 347, infoY)
		infoY -= 11
		for _, line := range WrapWidth(field.Value, Body, 8.5, 186) {
			if infoY < 600 {
				l.overflow = append(l.overflow, line)
				continue
			}
			l.text(line, Body, 8.5, 347, infoY)
			infoY -= 11
		}
		infoY -= 6
	}
	return min(640, addressBottom-20, infoY-13), l.overflow
}

func (l *letterheadLayout) fixedBlock(values []string, x, y, width, size, gap float64, maxLines int) float64 {
	count := 0
	for _, value := range values {
		for _, line := range WrapWidth(value, Body, size, width) {
			if count >= maxLines {
				l.overflow = append(l.overflow, line)
				continue
			}
			l.text(line, Body, size, x, y)
			y -= gap
			count++
		}
	}
	return y
}

package pdf

import (
	"fmt"
	"strings"
)

type Style int

const (
	Body Style = iota
	Heading
	Strong
	Table
)

type Line struct {
	Text  string
	Style Style
}
type Page struct {
	Lines  []Line
	Footer []string
}
type Palette struct{ Paper, Ink, Accent [3]uint8 }

// Pages renders explicit A4 pages with 16-point line spacing. Callers paginate
// at most 43 lines per page and supply their own repeated headers and footers.
func Pages(pages []Page, palette Palette) []byte {
	if len(pages) == 0 {
		pages = []Page{{}}
	}
	streams := make([]string, len(pages))
	color := func(c [3]uint8) string {
		return fmt.Sprintf("%.4f %.4f %.4f", float64(c[0])/255, float64(c[1])/255, float64(c[2])/255)
	}
	for i, page := range pages {
		var b strings.Builder
		fmt.Fprintf(&b, "%s rg\n0 0 595 842 re f\n%s RG\n0.7 w\n50 806 m 545 806 l S\n50 78 m 545 78 l S\n", color(palette.Paper), color(palette.Accent))
		for j, line := range page.Lines {
			font, size := "F1", 10
			switch line.Style {
			case Heading:
				font, size = "F4", 20
			case Strong:
				font = "F2"
			case Table:
				font, size = "F3", 9
			}
			fmt.Fprintf(&b, "%s rg\nBT /%s %d Tf 1 0 0 1 50 %d Tm (%s) Tj ET\n", color(palette.Ink), font, size, 784-j*16, escapeASCII(line.Text))
		}
		for j, line := range page.Footer {
			fmt.Fprintf(&b, "%s rg\nBT /F1 8 Tf 1 0 0 1 50 %d Tm (%s) Tj ET\n", color(palette.Ink), 63-j*11, escapeASCII(line))
		}
		streams[i] = b.String()
	}
	return writeStreams(streams)
}

// Columns wraps and aligns cells for the fixed-width Table font. Positive
// widths align left, negative widths align right; widths include padding.
func Columns(cells []string, widths []int) []Line {
	if len(cells) != len(widths) {
		panic("pdf: cell/width mismatch")
	}
	wrapped := make([][]string, len(cells))
	count := 1
	for i, cell := range cells {
		width := widths[i]
		if width < 0 {
			width = -width
		}
		if width < 2 {
			panic("pdf: column width must be at least 2")
		}
		wrapped[i] = WrapText(cell, width-1)
		count = max(count, len(wrapped[i]))
	}
	lines := make([]Line, count)
	for row := range lines {
		var b strings.Builder
		for col, width := range widths {
			value := ""
			if row < len(wrapped[col]) {
				value = wrapped[col][row]
			}
			left := width > 0
			if width < 0 {
				width = -width
			}
			padding := strings.Repeat(" ", width-1-len([]rune(value)))
			if left {
				b.WriteString(value + padding + " ")
			} else {
				b.WriteString(padding + value + " ")
			}
		}
		lines[row] = Line{Text: strings.TrimRight(b.String(), " "), Style: Table}
	}
	return lines
}

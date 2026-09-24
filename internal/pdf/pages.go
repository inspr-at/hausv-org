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
	Text       string
	Style      Style
	X, Y, Size float64 // PDF points, used by positioned pages.
}
type Page struct {
	Lines      []Line
	Footer     []string
	Positioned bool
	Shapes     []Shape
}

// Shape is a filled rectangle or horizontal rule, in PDF points.
type Shape struct {
	X, Y, Width, Height float64
	Color               [3]uint8
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
		if page.Positioned {
			fmt.Fprintf(&b, "%s rg\n0 0 595 842 re f\n", color(palette.Paper))
			for _, shape := range page.Shapes {
				fmt.Fprintf(&b, "%s rg\n%.2f %.2f %.2f %.2f re f\n", color(shape.Color), shape.X, shape.Y, shape.Width, shape.Height)
			}
			for _, line := range page.Lines {
				fmt.Fprintf(&b, "%s rg\nBT /%s %.2f Tf 1 0 0 1 %.2f %.2f Tm (%s) Tj ET\n", color(palette.Ink), fontName(line.Style), line.Size, line.X, line.Y, escapeASCII(line.Text))
			}
			fmt.Fprintf(&b, "%s RG\n0.5 w\n62 83 m 533 83 l S\n", color(palette.Accent))
			for j, line := range page.Footer {
				fmt.Fprintf(&b, "%s rg\nBT /F1 7 Tf 1 0 0 1 62 %d Tm (%s) Tj ET\n", color(palette.Ink), 69-j*10, escapeASCII(line))
			}
			streams[i] = b.String()
			continue
		}
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

func fontName(style Style) string {
	switch style {
	case Strong:
		return "F2"
	case Heading:
		return "F4"
	case Table:
		return "F3"
	default:
		return "F1"
	}
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

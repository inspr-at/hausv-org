package pdf

import "strings"

// TextWidth measures exactly the WinAnsi glyphs emitted by the PDF writer.
func TextWidth(text string, style Style, size float64) float64 {
	width := 0
	for _, r := range text {
		c := byte('?')
		switch {
		case r >= 32 && r <= 126, r >= 160 && r <= 255:
			c = byte(r)
		case r == '\n' || r == '\r' || r == '\t':
			c = ' '
		default:
			if special, ok := winAnsiSpecial[r]; ok {
				c = special
			}
		}
		switch style {
		case Strong:
			width += helveticaBoldWidths[c]
		case Heading:
			width += timesWidths[c]
		case Table:
			width += 600
		default:
			width += helveticaWidths[c]
		}
	}
	return float64(width) * size / 1000
}

// WrapWidth wraps by rendered point width, retaining explicit address lines and
// splitting an overlong token without losing any glyphs.
func WrapWidth(text string, style Style, size, width float64) []string {
	if width <= 0 || size <= 0 {
		panic("pdf: invalid text measure")
	}
	var lines []string
	for _, paragraph := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line := ""
		for _, word := range strings.Fields(paragraph) {
			candidate := word
			if line != "" {
				candidate = line + " " + word
			}
			if TextWidth(candidate, style, size) <= width {
				line = candidate
				continue
			}
			if line != "" {
				lines = append(lines, line)
				line = ""
			}
			for _, r := range word {
				candidate = line + string(r)
				if line != "" && TextWidth(candidate, style, size) > width {
					lines = append(lines, line)
					line = ""
				}
				line += string(r)
			}
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

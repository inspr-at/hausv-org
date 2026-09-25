package store

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// UnitLabelLess orders unit labels the way a person reads them: "Top 2" comes
// before "Top 10", and "Stellplatz 9" before "Stellplatz 11". Plain string
// comparison sorted the demo's register as Top 1, Top 10, Top 11, …, Top 2,
// which reads as a defect on every register a Hausverwaltung looks at.
//
// Digit runs compare numerically (by length first, so arbitrarily long numbers
// are safe), everything else case-insensitively. Labels that differ only in
// case fall back to the raw comparison so the order stays deterministic.
func UnitLabelLess(a string, b string) bool {
	left, right := strings.ToLower(a), strings.ToLower(b)
	if left != right {
		return naturalLess(left, right)
	}
	return a < b
}

func naturalLess(a string, b string) bool {
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if isDigit(a[i]) && isDigit(b[j]) {
			ai, aj := digitRun(a, i), digitRun(b, j)
			left := strings.TrimLeft(a[i:ai], "0")
			right := strings.TrimLeft(b[j:aj], "0")
			if len(left) != len(right) {
				return len(left) < len(right)
			}
			if left != right {
				return left < right
			}
			i, j = ai, aj
			continue
		}
		if a[i] != b[j] {
			return a[i] < b[j]
		}
		i++
		j++
	}
	return len(a)-i < len(b)-j
}

// UnitLabelText keeps a unit name and its number on one line. Stored labels
// stay ordinary spaces; only the displayed text uses a non-breaking space,
// so "Top" never wraps away from "3".
func UnitLabelText(label string) string {
	if !strings.Contains(label, " ") {
		return label
	}
	var b strings.Builder
	b.Grow(len(label))
	for i := 0; i < len(label); {
		r, size := utf8.DecodeRuneInString(label[i:])
		if r == ' ' && unitLabelGlue(label, i) {
			b.WriteRune('\u00a0')
		} else {
			b.WriteRune(r)
		}
		i += size
	}
	return b.String()
}

func unitLabelGlue(label string, space int) bool {
	prev, _ := utf8.DecodeLastRuneInString(label[:space])
	next, _ := utf8.DecodeRuneInString(label[space+1:])
	return unicode.IsLetter(prev) && unicode.IsDigit(next)
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func digitRun(s string, start int) int {
	end := start
	for end < len(s) && isDigit(s[end]) {
		end++
	}
	return end
}

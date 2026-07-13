// Package pdf is a minimal PDF writer: text lines in, PDF bytes out.
//
// It deliberately knows nothing about the domain — no handovers, no tenants.
// Document layout is the caller's job; this package only lays glyphs on pages.
package pdf

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

func WrapText(text string, max int) []string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	if text == "" {
		return nil
	}
	if max <= 0 {
		max = 92
	}
	out := []string{}
	for _, paragraph := range strings.Split(text, "\n") {
		words := strings.Fields(paragraph)
		line := ""
		for _, word := range words {
			if line == "" {
				line = word
				continue
			}
			if len([]rune(line))+1+len([]rune(word)) > max {
				out = append(out, line)
				line = word
				continue
			}
			line += " " + word
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func Simple(lines []string) []byte {
	const maxLinesPerPage = 42
	pages := [][]string{}
	for len(lines) > 0 {
		n := maxLinesPerPage
		if len(lines) < n {
			n = len(lines)
		}
		pages = append(pages, append([]string(nil), lines[:n]...))
		lines = lines[n:]
	}
	if len(pages) == 0 {
		pages = [][]string{{"Protokoll"}}
	}
	var objects []string
	objects = append(objects, "<< /Type /Catalog /Pages 2 0 R >>")
	kids := []string{}
	for i := range pages {
		pageObj := 3 + i*2
		kids = append(kids, strconv.Itoa(pageObj)+" 0 R")
	}
	objects = append(objects, fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(pages)))
	for i, pageLines := range pages {
		pageObj := 3 + i*2
		contentObj := pageObj + 1
		stream := contentStream(pageLines)
		objects = append(objects, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> /F2 << /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold >> >> >> /Contents %d 0 R >>", contentObj))
		objects = append(objects, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
	}
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	return buf.Bytes()
}

func contentStream(lines []string) string {
	var b strings.Builder
	b.WriteString("BT\n/F2 18 Tf\n50 790 Td\n")
	for i, line := range lines {
		if i == 1 {
			b.WriteString("/F1 11 Tf\n")
		}
		if i == 3 {
			b.WriteString("/F1 10 Tf\n")
		}
		if i > 0 {
			b.WriteString("0 -17 Td\n")
		}
		b.WriteString("(")
		b.WriteString(escapeASCII(line))
		b.WriteString(") Tj\n")
	}
	b.WriteString("ET")
	return b.String()
}

func escapeASCII(raw string) string {
	raw = strings.NewReplacer(
		"Ä", "Ae", "Ö", "Oe", "Ü", "Ue", "ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss", "€", "EUR",
		"–", "-", "—", "-", "·", "-", "„", "\"", "“", "\"", "”", "\"", "’", "'",
	).Replace(raw)
	var b strings.Builder
	for _, r := range raw {
		switch r {
		case '\\', '(', ')':
			b.WriteRune('\\')
			b.WriteRune(r)
		case '\t', '\n', '\r':
			b.WriteRune(' ')
		default:
			if r >= 32 && r <= 126 {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

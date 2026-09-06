package pdf

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestGermanEncodingAndPDFStructure(t *testing.T) {
	raw := "ÄÖÜ äöü ß € — Fläche (Prüfung) \\"
	encoded := escapeASCII(raw)
	for _, want := range []string{`\304`, `\326`, `\334`, `\344`, `\366`, `\374`, `\337`, `\200`, `\227`, `\(`, `\)`, `\\`} {
		if !strings.Contains(encoded, want) {
			t.Fatalf("missing %q in %s", want, encoded)
		}
	}
	data := Simple([]string{"Prüfung", raw})
	if !bytes.Contains(data, []byte("/Encoding /WinAnsiEncoding")) {
		t.Fatal("encoding missing")
	}
	start := strings.LastIndex(string(data), "startxref\n")
	offset, err := strconv.Atoi(strings.Split(string(data[start+10:]), "\n")[0])
	if err != nil || !bytes.HasPrefix(data[offset:], []byte("xref\n")) {
		t.Fatal("invalid xref")
	}
	xref := strings.Split(string(data[offset:]), "\n")
	var zero, count int
	fmt.Sscanf(xref[1], "%d %d", &zero, &count)
	for i := 1; i < count; i++ {
		var pos int
		fmt.Sscanf(xref[2+i], "%d", &pos)
		if !bytes.HasPrefix(data[pos:], []byte(fmt.Sprintf("%d 0 obj", i))) {
			t.Fatalf("object %d offset invalid", i)
		}
	}
	if got := escapeASCII("漢\n字"); got != "? ?" {
		t.Fatalf("unsupported glyph silently lost: %q", got)
	}
}

func TestColumnsWrapAndAlignUnicode(t *testing.T) {
	rows := Columns([]string{"Fläche sehrlangeswort", "25,00 €"}, []int{9, -10})
	if len(rows) < 2 || !strings.HasSuffix(rows[0].Text, "25,00 €") {
		t.Fatalf("rows=%+v", rows)
	}
	for _, row := range rows {
		if len([]rune(row.Text)) > 18 || row.Style != Table {
			t.Fatalf("overflow=%+v", row)
		}
	}
	if got := WrapText("äöüßäöüß", 3); strings.Join(got, "") != "äöüßäöüß" || len(got) != 3 {
		t.Fatal(got)
	}
}

func TestExplicitPagesDeterministic(t *testing.T) {
	pages := []Page{{Lines: []Line{{Text: "Erste", Style: Heading}}, Footer: []string{"Fußzeile"}}, {Lines: []Line{{Text: "Zweite"}}}}
	palette := Palette{Paper: [3]uint8{255, 255, 255}}
	data := Pages(pages, palette)
	if !bytes.Equal(data, Pages(pages, palette)) || !bytes.Contains(data, []byte("/Count 2")) || !bytes.Contains(data, []byte(`Fu\337zeile`)) {
		t.Fatal("page count, encoding or determinism")
	}
}

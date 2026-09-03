package store

import (
	"reflect"
	"testing"
)

func TestHausv616FillReplyUsesValuesAndNeutralFallbacks(t *testing.T) {
	text, unfilled := FillReply("Sehr geehrte{{Anrede}} {{Name}}, für {{Haus}}, {{Einheit}} unter {{Nummer}}: {{Zuständig}}; {{Handwerker}}; {{Frist}}.", map[string]string{
		"Name": "Rita Beispiel", "Haus": "Grazbachgasse 14", "Einheit": "Top 7", "Nummer": "in-0002",
	})
	want := "Sehr geehrte Rita Beispiel, für Grazbachgasse 14, Top 7 unter in-0002: die zuständige Person; einem Fachbetrieb; in Kürze."
	if text != want {
		t.Fatalf("FillReply = %q, want %q", text, want)
	}
	if want := []string{"Zuständig", "Handwerker", "Frist"}; !reflect.DeepEqual(unfilled, want) {
		t.Fatalf("unfilled = %#v, want %#v", unfilled, want)
	}
}

func TestHausv616FillReplyUsesNeutralSalutationWithoutName(t *testing.T) {
	text, unfilled := FillReply("Sehr geehrte{{Anrede}} Frau {{Name}},\n\nwir melden uns.", nil)
	if text != "Sehr geehrte Damen und Herren,\n\nwir melden uns." {
		t.Fatalf("FillReply = %q", text)
	}
	if want := []string{"Name"}; !reflect.DeepEqual(unfilled, want) {
		t.Fatalf("unfilled = %#v, want %#v", unfilled, want)
	}
}

func TestHausv616TidyReplyRemovesModelMadeEmptySlots(t *testing.T) {
	for _, test := range []struct{ in, want string }{
		{"Ihre Änderung für , Top 7, wurde aufgenommen.", "Ihre Änderung für Top 7 wurde aufgenommen."},
		{"wir prüfen Ihre Rückfrage zur Vorschreibung für , , unter  und melden uns mit einer nachvollziehbaren Aufstellung.", "wir prüfen Ihre Rückfrage zur Vorschreibung und melden uns mit einer nachvollziehbaren Aufstellung."},
	} {
		if got := TidyReply(test.in); got != test.want {
			t.Errorf("TidyReply(%q) = %q, want %q", test.in, got, test.want)
		}
	}
}

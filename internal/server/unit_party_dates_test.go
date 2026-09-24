package server

import (
	"net/url"
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestUnitPartyDateFormPreservesAddressAndValidates(t *testing.T) {
	unit := store.Unit{RenterEmails: []string{"old@example.com", "new@example.com"}, PartyContacts: []store.UnitPartyContact{{Email: "old@example.com", Name: "Bisherige Partei", Address: "Gasse 1"}}}
	values := url.Values{"party_email": {"old@example.com", "new@example.com"}, "party_valid_from": {"", "2025-07-01"}, "party_valid_to": {"2025-06-30", ""}}
	if err := applyUnitPartyDates(&unit, values); err != nil {
		t.Fatal(err)
	}
	if unit.PartyContacts[0].Name != "Bisherige Partei" || unit.PartyContacts[0].Address != "Gasse 1" || unit.PartyContacts[1].ValidFrom != "2025-07-01" {
		t.Fatal(unit)
	}
	for _, invalid := range []url.Values{
		{"party_email": {"foreign@example.com"}, "party_valid_from": {"2025-07-01"}, "party_valid_to": {""}},
		{"party_email": {"old@example.com"}, "party_valid_from": {"2025-07-01"}, "party_valid_to": {"2025-06-30"}},
		{"party_email": {"old@example.com"}, "party_valid_from": {"2025-02-30"}, "party_valid_to": {""}},
		{"party_email": {"old@example.com"}},
	} {
		if err := applyUnitPartyDates(&unit, invalid); err == nil {
			t.Fatal("invalid form accepted", invalid)
		}
	}
}

func TestPartyDateLabelOpenBounds(t *testing.T) {
	for _, tc := range []struct{ from, to, want string }{
		{"", "2025-06-30", "bis 30.06.2025"},
		{"2025-07-01", "", "ab 01.07.2025"},
		{"", "", "unbefristet"},
		{"2025-01-01", "2025-06-30", "01.01.2025 – 30.06.2025"},
	} {
		if got := partyDateLabel(tc.from, tc.to); got != tc.want {
			t.Fatalf("got %q, want %q", got, tc.want)
		}
	}
}

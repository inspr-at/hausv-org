package server

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/view"
	"github.com/inspr-at/hausv-org/internal/web"
)

func unitPartyDateViews(unit store.Unit) []view.UnitPartyDateView {
	var out []view.UnitPartyDateView
	for _, email := range store.NormalizeEmailList(append(append([]string(nil), unit.OwnerEmails...), unit.RenterEmails...)) {
		p := view.UnitPartyDateView{Email: email}
		for _, contact := range unit.PartyContacts {
			if contact.Email == email {
				p.From, p.To = contact.ValidFrom, contact.ValidTo
			}
		}
		out = append(out, p)
	}
	return out
}
func applyUnitPartyDates(unit *store.Unit, values url.Values) error {
	if !values.Has("party_email") {
		return nil
	}
	emails, from, to := values["party_email"], values["party_valid_from"], values["party_valid_to"]
	if len(emails) != len(from) || len(emails) != len(to) {
		return fmt.Errorf("incomplete party dates")
	}
	contacts := append([]store.UnitPartyContact(nil), unit.PartyContacts...)
	seen := map[string]bool{}
	for i, raw := range emails {
		email := normalizeEmail(raw)
		if email == "" && from[i] == "" && to[i] == "" {
			continue
		}
		if !store.EmailListContains(unit.OwnerEmails, email) && !store.EmailListContains(unit.RenterEmails, email) && from[i] == "" && to[i] == "" {
			continue
		}
		if seen[email] || (!store.EmailListContains(unit.OwnerEmails, email) && !store.EmailListContains(unit.RenterEmails, email)) {
			return fmt.Errorf("unknown or duplicate party")
		}
		seen[email] = true
		contact := store.UnitPartyContact{Email: email}
		index := -1
		for j, p := range contacts {
			if p.Email == email {
				contact = p
				index = j
				break
			}
		}
		contact.ValidFrom, contact.ValidTo = strings.TrimSpace(from[i]), strings.TrimSpace(to[i])
		if index >= 0 {
			contacts[index] = contact
		} else {
			contacts = append(contacts, contact)
		}
	}
	if err := store.ValidateUnitPartyContacts(contacts); err != nil {
		return err
	}
	unit.PartyContacts = contacts
	return nil
}
func partyDateLabel(from, to string) string {
	date := func(raw, empty string) string {
		if raw == "" {
			return empty
		}
		at, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return raw
		}
		return at.Format("02.01.2006")
	}
	return date(from, "Beginn offen") + " – " + date(to, "unbefristet")
}
func addOccupancyPartyDates(people []web.Person, unit store.Unit) {
	for i, p := range people {
		for _, c := range unit.PartyContacts {
			if p.Email == c.Email && (c.ValidFrom != "" || c.ValidTo != "") {
				people[i].Validity = partyDateLabel(c.ValidFrom, c.ValidTo)
			}
		}
	}
}

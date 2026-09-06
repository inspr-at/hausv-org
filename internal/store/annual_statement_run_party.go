package store

import (
	"sort"
	"strings"
)

// UnitPartyContact is explicitly supplied addressing data, never inferred from
// a unit's address or from an owner's/renter's role.
type UnitPartyContact struct {
	Email   string `json:"email"`
	Name    string `json:"name,omitempty"`
	Address string `json:"address,omitempty"`
}

type AnnualStatementRunPresentation struct {
	Organisation   string `json:"organisation"`
	EstateSlug     string `json:"estate_slug"`
	EstateName     string `json:"estate_name"`
	EstateAddress  string `json:"estate_address"`
	ContactName    string `json:"contact_name,omitempty"`
	ContactAddress string `json:"contact_address,omitempty"`
	ContactEmail   string `json:"contact_email,omitempty"`
	ContactPhone   string `json:"contact_phone,omitempty"`
}

type AnnualStatementRunParty struct {
	UnitID  string `json:"unit_id"`
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Address string `json:"address,omitempty"`
	Owner   bool   `json:"owner"`
	Renter  bool   `json:"renter"`
}

func mergeUnitPartyContacts(unit Unit, updates []UnitPartyContact) []UnitPartyContact {
	byEmail := map[string]UnitPartyContact{}
	for _, contact := range append(append([]UnitPartyContact(nil), unit.PartyContacts...), updates...) {
		contact.Email = strings.ToLower(strings.TrimSpace(contact.Email))
		if !EmailListContains(unit.OwnerEmails, contact.Email) && !EmailListContains(unit.RenterEmails, contact.Email) {
			continue
		}
		contact.Name = strings.TrimSpace(contact.Name)
		contact.Address = strings.TrimSpace(contact.Address)
		byEmail[contact.Email] = contact
	}
	out := make([]UnitPartyContact, 0, len(byEmail))
	for _, contact := range byEmail {
		out = append(out, contact)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Email < out[j].Email })
	return out
}

func annualStatementRunParties(unit Unit) []AnnualStatementRunParty {
	contacts := map[string]UnitPartyContact{}
	for _, contact := range unit.PartyContacts {
		contacts[contact.Email] = contact
	}
	emails := NormalizeEmailList(append(append([]string(nil), unit.OwnerEmails...), unit.RenterEmails...))
	sort.Strings(emails)
	out := make([]AnnualStatementRunParty, 0, len(emails))
	for _, email := range emails {
		contact := contacts[email]
		out = append(out, AnnualStatementRunParty{UnitID: unit.ID, ID: email, Name: contact.Name, Address: contact.Address, Owner: EmailListContains(unit.OwnerEmails, email), Renter: EmailListContains(unit.RenterEmails, email)})
	}
	return out
}

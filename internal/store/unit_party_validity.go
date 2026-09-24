package store

import (
	"encoding/json"
	"fmt"
	"time"
)

// Bounds are inclusive civil dates. One contiguous interval is recorded per
// party identity; blank dates preserve the existing unbounded assignment.
type unitPartyBounds struct {
	From string `json:"valid_from,omitempty"`
	To   string `json:"valid_to,omitempty"`
}

func ValidateUnitPartyContacts(contacts []UnitPartyContact) error {
	for _, p := range contacts {
		for _, date := range []string{p.ValidFrom, p.ValidTo} {
			if date != "" {
				if _, err := time.Parse("2006-01-02", date); err != nil {
					return fmt.Errorf("invalid party date")
				}
			}
		}
		if p.ValidFrom != "" && p.ValidTo != "" && p.ValidTo < p.ValidFrom {
			return fmt.Errorf("party end before start")
		}
	}
	return nil
}

func partyActive(from, to, on string) bool {
	return (from == "" || from <= on) && (to == "" || to >= on)
}

// UnitPartyActive prevents historical assignments from granting current unit
// membership. Billing snapshots deliberately retain those past assignments.
func UnitPartyActive(unit Unit, email string, now time.Time) bool {
	loc, _ := time.LoadLocation("Europe/Vienna")
	if loc != nil {
		now = now.In(loc)
	}
	for _, p := range unit.PartyContacts {
		if p.Email == email {
			return partyActive(p.ValidFrom, p.ValidTo, now.Format("2006-01-02"))
		}
	}
	return true
}

// EncodeUnitWithValidity keeps bounds in an additive column on the already tenant-scoped units
// table. The existing data JSON remains the source for names and assignments.
func EncodeUnitWithValidity(item Unit) (string, string, error) {
	if err := ValidateUnitPartyContacts(item.PartyContacts); err != nil {
		return "", "", err
	}
	item = CopyUnit(item)
	bounds := map[string]unitPartyBounds{}
	for i, p := range item.PartyContacts {
		if p.ValidFrom != "" || p.ValidTo != "" {
			bounds[p.Email] = unitPartyBounds{p.ValidFrom, p.ValidTo}
		}
		item.PartyContacts[i].ValidFrom, item.PartyContacts[i].ValidTo = "", ""
	}
	data, err := json.Marshal(item)
	if err != nil {
		return "", "", err
	}
	dates, err := json.Marshal(bounds)
	return string(data), string(dates), err
}

func decodeUnitValidity(item *Unit, raw string) error {
	bounds := map[string]unitPartyBounds{}
	if err := json.Unmarshal([]byte(raw), &bounds); err != nil {
		return err
	}
	byEmail := map[string]bool{}
	for i, p := range item.PartyContacts {
		b := bounds[p.Email]
		item.PartyContacts[i].ValidFrom, item.PartyContacts[i].ValidTo = b.From, b.To
		byEmail[p.Email] = true
	}
	for email, b := range bounds {
		if !byEmail[email] {
			item.PartyContacts = append(item.PartyContacts, UnitPartyContact{Email: email, ValidFrom: b.From, ValidTo: b.To})
		}
	}
	item.PartyContacts = mergeUnitPartyContacts(*item, nil)
	return ValidateUnitPartyContacts(item.PartyContacts)
}

func activeUnitMembers(item Unit) UnitMembers {
	out := UnitMembers{Unit: CopyUnit(item), Found: true}
	now := time.Now()
	for _, email := range item.OwnerEmails {
		if UnitPartyActive(item, email, now) {
			out.Owners = append(out.Owners, email)
		}
	}
	for _, email := range item.RenterEmails {
		if UnitPartyActive(item, email, now) {
			out.Renters = append(out.Renters, email)
		}
	}
	return out
}

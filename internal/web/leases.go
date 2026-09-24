package web

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/view"
)

// LeasePageData is the Mietvertrag tab for one unit.
type LeasePageData struct {
	Portal    PortalPageData
	UnitID    string
	UnitLabel string
	HasLease  bool
	Lease     LeaseDetail
	History   []LeaseDetail
	Form      LeaseForm
	Message   string
	OK        bool
	CanEnd    bool
}

type LeaseDetail struct {
	ID, Status, Kind, Use, MRG, Regime  string
	Concluded, Starts, Ends, Zinstermin string
	Notes                               string
	MieWeG, SpecialCap                  string
	MieWeGReason, CapReason             string
	Parties                             []LeasePartyView
	Components                          []LeaseMoneyView
	HasClause                           bool
	Clause                              LeaseClauseView
	Anchor                              string
}

type LeasePartyView struct {
	Name, Email, Address, Role, From, To string
}

type LeaseMoneyView struct {
	Kind, From, Net, VAT, Origin string
}

type LeaseClauseView struct {
	Type, Series, Base, Threshold, Text, Review, Note string
}

type LeaseForm struct {
	ID, Concluded, Starts, Ends, Notes                             string
	PartyName, PartyEmail, PartyAddress, PartyFrom, PartyTo        string
	ComponentNet, ComponentFrom                                    string
	ClauseID, Series, BasePeriod, BaseValue, Threshold, ClauseText string
	Zinstermin                                                     string
	PriceRestricted, Consumer, Business, TwoWay, Inclusive         bool
	Kinds, Uses, Scopes, Regimes, PartyRoles                       []view.SelectOption
	ComponentKinds, VATRates, ClauseTypes, ThresholdKinds          []view.SelectOption
	Reviews                                                        []view.SelectOption
}

type LeaseImportPageData struct {
	Portal    PortalPageData
	Rows      []LeaseImportRowView
	HasRows   bool
	Committed int
	Message   string
	OK        bool
}

type LeaseImportRowView struct {
	Line        int
	Unit        string
	Errors      string
	Warnings    string
	HasErrors   bool
	HasWarnings bool
}

func LeaseMoney(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "−"
		cents = -cents
	}
	raw := strconv.FormatInt(cents/100, 10)
	grouped := make([]byte, 0, len(raw)+2)
	for i := 0; i < len(raw); i++ {
		if i > 0 && (len(raw)-i)%3 == 0 {
			grouped = append(grouped, '.')
		}
		grouped = append(grouped, raw[i])
	}
	return sign + string(grouped) + "," + fmt.Sprintf("%02d", cents%100) + " €"
}

func LeaseDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "offen"
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return raw
	}
	return parsed.Format("02.01.2006")
}

func LeaseMonth(raw string) string {
	raw = strings.TrimSpace(raw)
	parsed, err := time.Parse("2006-01", raw)
	if err != nil {
		return raw
	}
	return parsed.Format("01.2006")
}

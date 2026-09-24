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
	Subtitle  string
	Editing   bool
	Ending    bool
	HasLease  bool
	Lease     LeaseDetail
	History   []LeaseDetail
	Form      LeaseForm
	Message   string
	OK        bool
	CanEnd    bool
}

type LeaseValorisationView struct{ URL, PDFURL, Date, Amount, Due, Status string }

type LeaseDetail struct {
	Valorisation                         []LeaseValorisationView
	ID, Status, Kind, Use, MRG, Regime   string
	Concluded, Starts, Ends, Zinstermin  string
	Notes                                string
	MieWeG, SpecialCap                   string
	MieWeGReason, CapReason              string
	Parties                              []LeasePartyView
	Components                           []LeaseMoneyView
	MonthlyNet, MonthlyVAT, MonthlyGross string
	HasMonthly                           bool
	ShowOrigin                           bool
	HasClause                            bool
	Clause                               LeaseClauseView
	Anchor                               string
}

type LeasePartyView struct {
	Name, Email, Address, Role, From, To string
}

type LeaseMoneyView struct {
	Kind, From, Net, VAT, Gross, Origin string
}

type LeaseClauseView struct {
	Type, Series, Base, Threshold, Text, Review, ReviewClass, Line, Note string
	Staffel                                                              []string
}

type LeaseStaffelRow struct {
	Date, Value string
	Percent     bool
}

type LeaseForm struct {
	Staffel                                                        []LeaseStaffelRow
	ID, Concluded, Starts, Ends, Notes                             string
	PartyName, PartyEmail, PartyAddress, PartyFrom, PartyTo        string
	ComponentNet, ComponentFrom                                    string
	ClauseID, Series, BasePeriod, BaseValue, Threshold, ClauseText string
	Zinstermin                                                     string
	PriceRestricted, Consumer, Business, TwoWay, Inclusive         bool
	Kinds, Uses, Scopes, Regimes, PartyRoles                       []view.SelectOption
	ComponentKinds, VATRates, ClauseTypes, ThresholdKinds          []view.SelectOption
	SeriesOptions, Reviews                                         []view.SelectOption
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

var leaseMonths = [...]string{"", "Jänner", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"}

// LeaseSeriesLabel turns a stored index code into the name shown on the page.
func LeaseSeriesLabel(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if strings.HasPrefix(raw, "vpi") && len(raw) > 3 {
		return "VPI " + raw[3:]
	}
	return raw
}

// LeaseMonthName renders a stored YYYY-MM as an Austrian month and year.
func LeaseMonthName(raw string) string {
	raw = strings.TrimSpace(raw)
	parsed, err := time.Parse("2006-01", raw)
	if err != nil {
		return raw
	}
	return leaseMonths[parsed.Month()] + " " + parsed.Format("2006")
}

// LeaseIndexLine is the read-view sentence for a clause anchor.
func LeaseIndexLine(series, period, value string) string {
	label := LeaseSeriesLabel(series)
	if label == "" {
		return ""
	}
	month := LeaseMonthName(period)
	value = strings.ReplaceAll(strings.TrimSpace(value), ".", ",")
	if month == "" || value == "" {
		return label
	}
	return label + " · Basis " + month + " = " + value
}

// LeaseGrossCents is net plus VAT. VAT is stored in basis points.
func LeaseGrossCents(net int64, vatBP int) int64 {
	if vatBP <= 0 {
		return net
	}
	return net + net*int64(vatBP)/10000
}

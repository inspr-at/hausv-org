package store

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"
)

// RentalManagement is an explicit mandate for one owner's unit. ContractCosts
// is an agreed catalogue per lease, never inferred from WEG allocatability.
type RentalManagement struct {
	UnitID                  string              `json:"unit_id"`
	OwnerEmail              string              `json:"owner_email"`
	Active                  bool                `json:"active"`
	HeatingMonthlyConfirmed bool                `json:"heating_monthly_confirmed"`
	Passable                map[string]bool     `json:"passable"`
	ContractCosts           map[string][]string `json:"contract_costs"`
	UpdatedBy               string              `json:"updated_by"`
	UpdatedAt               time.Time           `json:"updated_at"`
}

func DefaultTenantPassable(key string) bool {
	switch key {
	case "grundsteuer", "muellabfuhr", "abfall", "reinigung", "hausreinigung", "hausbetreuung", "wasser", "kaltwasser", "kanal", "abwasser", "wasser_abwasser", "rauchfangkehrer", "schaedlingsbekaempfung", "allgemeinstrom", "beleuchtung":
		return true
	}
	// Combined insurance, administration and special facilities require review
	// of their statutory conditions/caps before explicitly enabling them.
	return false
}

func tenantReserveCost(key string) bool {
	k := strings.ToLower(key)
	return strings.Contains(k, "ruecklag") || strings.Contains(k, "rücklag") || strings.Contains(k, "reserve")
}

// TenantStatementOwnerChangeIssue is shared by the preview and the write gate.
func TenantStatementOwnerChangeIssue(run AnnualStatementRun, unitID, ownerID string) string {
	if _, split := AnnualStatementPartyAllocation(run, unitID, ownerID); split {
		for _, party := range run.Input.Parties {
			if party.UnitID == unitID && party.Owner && party.ID != ownerID {
				return "Bei Eigentümerwechsel benötigt die Mieterabrechnung eine gesonderte Zuordnung der Vermieterzeiträume und Akontos."
			}
		}
	}
	return ""
}

const TenantStatementVATIssue = "Für USt-pflichtige Vermietung benötigt der WEG-Lauf eine ausgewiesene Netto-/USt-Basis."

func tenantLeaseDescription(l Lease) string {
	var names []string
	for _, p := range l.Parties {
		if name := strings.TrimSpace(p.Name); name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "Mietvertrag dieser Einheit"
	}
	return "Mietvertrag von " + strings.Join(names, ", ")
}

type TenantStatementLine struct {
	Key, Name                                        string
	Heating                                          bool
	SourceGrossCents, NetCents, VATCents, GrossCents int64
	VATRate                                          int
}

type TenantStatementAccount struct {
	ID, LeaseID, Scope, From, To               string
	Parties                                    []LeaseParty
	Intervals                                  []TenantStatementInterval
	Lines                                      []TenantStatementLine
	OperatingPrepaidCents, HeatingPrepaidCents int64
	OperatingCents, HeatingCents               int64
	OperatingDueOn, HeatingDueOn               string
	// An unlet interval remains with the landlord, not another tenant.
	Landlord bool
}

type TenantStatementInterval struct{ From, To string }

func (a TenantStatementAccount) BalanceCents() int64 {
	return a.OperatingCents + a.HeatingCents - a.OperatingPrepaidCents - a.HeatingPrepaidCents
}

type TenantStatement struct {
	ID, RunID, UnitID, UnitLabel, StatementOn, CreatedBy string
	CreatedAt                                            time.Time
	Revision                                             int
	Approval                                             *AnnualStatementRunApproval
	Owner                                                AnnualStatementRunParty
	Source                                               AnnualStatementRun
	Management                                           RentalManagement
	Leases                                               []Lease
	Accounts                                             []TenantStatementAccount
	// SourcePassed + SourceRetained = the owner's WEG unit share. Tenant VAT
	// is shown separately and can differ from WEG VAT. Rücklage stays separate.
	SourcePassedCents, SourceRetainedCents int64
	Excluded                               []AnnualStatementRunCost
}

func TenantOperatingDueOn(at time.Time) string {
	months := 1
	if at.Day() >= 5 {
		months = 2
	}
	return time.Date(at.Year(), at.Month()+time.Month(months), 5, 0, 0, 0, 0, time.UTC).Format(time.DateOnly)
}

// DeriveTenantStatement is pure. All money is integer cents; split remainders
// conserve the original WEG amount, including vacant intervals.
func DeriveTenantStatement(run AnnualStatementRun, management RentalManagement, leases []Lease, statementOn, contractDueOn string) (TenantStatement, error) {
	out := TenantStatement{RunID: run.ID, Source: run, UnitID: management.UnitID, Management: management, StatementOn: statementOn, Revision: 1}
	fail := func(message string) (TenantStatement, error) { return TenantStatement{}, fmt.Errorf("%s", message) }
	if run.Approval == nil || run.Input.Structure.Legal.Regime != "weg" {
		return fail("Nur freigegebene WEG-Läufe können abgeleitet werden.")
	}
	if !management.Active {
		return fail("Mietverwaltung ist für diese Eigentümer-Einheit nicht aktiv.")
	}
	at, err := time.Parse(time.DateOnly, statementOn)
	if err != nil || statementOn <= run.Input.Period.EndsOn {
		return fail("Abrechnungsdatum muss nach der Periode liegen.")
	}
	start, e1 := time.Parse(time.DateOnly, run.Input.Period.StartsOn)
	end, e2 := time.Parse(time.DateOnly, run.Input.Period.EndsOn)
	if e1 != nil || e2 != nil || start.Month() != 1 || start.Day() != 1 || end.Month() != 12 || end.Day() != 31 || start.Year() != end.Year() {
		return fail("Mieterabrechnung v1 benötigt eine Kalenderjahresperiode.")
	}
	end = end.AddDate(0, 0, 1)
	var unit *AnnualStatementRunUnit
	for i := range run.Result.Units {
		if run.Result.Units[i].UnitID == management.UnitID {
			unit = &run.Result.Units[i]
		}
	}
	if unit == nil {
		return fail("Einheit fehlt im WEG-Lauf.")
	}
	out.UnitLabel = unit.Label
	for _, p := range run.Input.Parties {
		if p.UnitID == management.UnitID && p.Owner && p.ID == management.OwnerEmail {
			out.Owner = p
		}
	}
	if out.Owner.ID == "" || out.Owner.Name == "" {
		return fail("Eigentümer mit Name muss im WEG-Lauf hinterlegt sein.")
	}
	// v4/v5 can split a unit between successive WEG owners. This v1 tenant
	// workflow has no landlord periods for lease Akontos, so using the whole
	// unit (or just scaling its costs) would settle against the wrong owner.
	if issue := TenantStatementOwnerChangeIssue(run, management.UnitID, management.OwnerEmail); issue != "" {
		return fail(issue)
	}
	legal := run.Input.Structure.Legal
	if legal.InspectionPlace == "" || legal.InspectionPeriod == "" || legal.InspectionContact == "" {
		return fail("Ort, Zeitraum und Kontakt für die Belegeinsicht fehlen.")
	}
	for _, l := range leases {
		if l.UnitID != management.UnitID || l.LeaseKind != LeaseKindHauptmiete || l.Status == LeaseStatusDraft {
			continue
		}
		if l.MRGScope == MRGWGG {
			return fail("WGG-Mietverträge sind in dieser Ableitung nicht unterstützt.")
		}
		if l.UseKind != UseKindWohnung && l.UseKind != UseKindGarage && l.UseKind != UseKindGeschaeft {
			return fail("Nutzung für die Umsatzsteuer bitte klären.")
		}
		out.Leases = append(out.Leases, l)
	}
	sort.Slice(out.Leases, func(i, j int) bool { return out.Leases[i].StartsOn < out.Leases[j].StartsOn })
	if len(out.Leases) == 0 {
		return fail("Kein Hauptmietvertrag für die Einheit vorhanden.")
	}
	for i, l := range out.Leases {
		for _, other := range out.Leases[:i] {
			if rangesOverlap(l.StartsOn, l.EndsOn, other.StartsOn, other.EndsOn) {
				return fail("Überlappende Mietverträge bitte klären.")
			}
		}
	}
	// Partition at each lease/party boundary: co-tenants receive copies of one
	// joint account, not multiple charges for the same unit.
	boundaries := []string{start.Format(time.DateOnly), end.Format(time.DateOnly)}
	for _, l := range out.Leases {
		for _, d := range []string{l.StartsOn, l.EndsOn} {
			if d > boundaries[0] && d < end.Format(time.DateOnly) {
				boundaries = append(boundaries, d)
			}
		}
		for _, p := range l.Parties {
			for _, d := range []string{p.ValidFrom, p.ValidTo} {
				if d > start.Format(time.DateOnly) && d < end.Format(time.DateOnly) {
					boundaries = append(boundaries, d)
				}
			}
		}
	}
	sort.Strings(boundaries)
	unique := boundaries[:0]
	for _, d := range boundaries {
		if len(unique) == 0 || unique[len(unique)-1] != d {
			unique = append(unique, d)
		}
	}
	accountAt := func(day string) (Lease, []LeaseParty) {
		for _, l := range out.Leases {
			if l.StartsOn <= day && day < leaseEndExclusive(l.EndsOn) {
				var parties []LeaseParty
				for _, p := range l.Parties {
					if p.ValidFrom <= day && day < leaseEndExclusive(p.ValidTo) {
						parties = append(parties, p)
					}
				}
				sort.Slice(parties, func(i, j int) bool { return parties[i].ID < parties[j].ID })
				return l, parties
			}
		}
		return Lease{}, nil
	}
	accountKey := func(l Lease, parties []LeaseParty) string {
		key := l.ID
		for _, p := range parties {
			key += "/" + p.ID
		}
		if key == "" {
			key = "landlord"
		}
		return key
	}
	indexes := map[string]int{}
	weights := []int64{}
	ensure := func(l Lease, parties []LeaseParty, from, to string) (int, error) {
		if l.ID != "" && len(parties) == 0 {
			return 0, fmt.Errorf("Mietparteien fehlen für %s.", from)
		}
		key := accountKey(l, parties)
		if i, ok := indexes[key]; ok {
			if from < to {
				out.Accounts[i].Intervals = append(out.Accounts[i].Intervals, TenantStatementInterval{from, to})
			}
			if from < to && from < out.Accounts[i].From {
				out.Accounts[i].From = from
			}
			if from < to && to > out.Accounts[i].To {
				out.Accounts[i].To = to
			}
			return i, nil
		}
		due := TenantOperatingDueOn(at)
		if l.ID != "" && l.MRGScope != MRGVoll {
			d, e := time.Parse(time.DateOnly, contractDueOn)
			if e != nil || d.Before(at) {
				return 0, fmt.Errorf("Vertragliche Fälligkeit für Teilanwendung/Ausnahme erforderlich.")
			}
			due = contractDueOn
			if _, ok := management.ContractCosts[l.ID]; !ok {
				return 0, fmt.Errorf("Im %s fehlt der vereinbarte Kostenkatalog.", tenantLeaseDescription(l))
			}
		}
		i := len(out.Accounts)
		indexes[key] = i
		weights = append(weights, 0)
		out.Accounts = append(out.Accounts, TenantStatementAccount{ID: fmt.Sprintf("account-%d", i+1), LeaseID: l.ID, Scope: l.MRGScope, Parties: parties, From: from, To: to, Landlord: l.ID == "", OperatingDueOn: due, HeatingDueOn: ShiftStatementDate(at, 2).Format(time.DateOnly)})
		if from < to {
			out.Accounts[i].Intervals = []TenantStatementInterval{{from, to}}
		}
		return i, nil
	}
	for i := 0; i < len(unique)-1; i++ {
		from, to := unique[i], unique[i+1]
		l, ps := accountAt(from)
		idx, e := ensure(l, ps, from, to)
		if e != nil {
			return fail(e.Error())
		}
		f, _ := time.Parse(time.DateOnly, from)
		t, _ := time.Parse(time.DateOnly, to)
		weights[idx] += tenantMonthWeight(f, t)
	}
	// MRG annual lump sum belongs to the due-date tenant even after a move.
	due := TenantOperatingDueOn(at)
	dueLease, dueParties := accountAt(due)
	dueIdx, e := ensure(dueLease, dueParties, due, due)
	if e != nil {
		return fail(e.Error())
	}
	leaseByID := map[string]Lease{}
	for _, l := range out.Leases {
		leaseByID[l.ID] = l
	}
	allFull := true
	for _, a := range out.Accounts {
		if !a.Landlord && a.Scope != MRGVoll {
			allFull = false
		}
	}
	if !allFull {
		// A catalogue alone does not define settlement on a contractual move.
		// v1 accepts a single unchanged contract; mixed regimes need review.
		count := 0
		for i, a := range out.Accounts {
			if weights[i] > 0 && !a.Landlord {
				count++
			}
		}
		if count != 1 || dueLease.ID == "" {
			return fail("Mieterwechsel außerhalb der MRG-Vollanwendung benötigt eine vertragliche Einzelabrechnung.")
		}
	}
	var total int64
	if legal.HeizKGApplies && !management.HeatingMonthlyConfirmed {
		return fail("HeizKG: Bitte bestätigen, dass keine Zwischenermittlung vorliegt. Vorhandene Zwischenablesungen benötigen eine gesonderte Heizkostenaufteilung.")
	}
	for _, c := range unit.Costs {
		if c.AmountCents < 0 || c.AmountCents > 1_000_000_000_000 {
			return fail("WEG-Betrag außerhalb des sicheren Bereichs.")
		}
		total += c.AmountCents
		heating := IsAnnualHeatingCost(c.CostTypeKey)
		if heating && !legal.HeizKGApplies {
			return fail("Heizkosten benötigen eine freigegebene HeizKG-Verteilung.")
		}
		costWeights := append([]int64(nil), weights...)
		if !heating && allFull {
			for i := range costWeights {
				costWeights[i] = 0
			}
			costWeights[dueIdx] = 1
		}
		shares := distributeLargestRemainder(c.AmountCents, costWeights)
		net := c.AmountCents
		if legal.ShowVAT {
			if c.NetCents < 0 || c.VATCents < 0 || c.NetCents > c.AmountCents || c.NetCents+c.VATCents != c.AmountCents {
				return fail("WEG-Umsatzsteuerbasis ist unvollständig.")
			}
			net = c.NetCents
		}
		nets := distributeLargestRemainder(net, shares)
		var passed int64
		for i, share := range shares {
			if share == 0 {
				continue
			}
			a := &out.Accounts[i]
			l := leaseByID[a.LeaseID]
			allowed := heating
			if !heating {
				allowed = DefaultTenantPassable(c.CostTypeKey)
				if value, ok := management.Passable[c.CostTypeKey]; ok {
					allowed = value
				}
				if !a.Landlord && l.MRGScope != MRGVoll {
					allowed = false
					for _, key := range management.ContractCosts[l.ID] {
						if key == c.CostTypeKey {
							allowed = true
						}
					}
				}
			}
			if tenantReserveCost(c.CostTypeKey) || tenantReserveCost(c.Name) || !allowed {
				continue
			}
			if l.VATOpted && !legal.ShowVAT {
				return fail(TenantStatementVATIssue)
			}
			line := TenantStatementLine{Key: c.CostTypeKey, Name: c.Name, Heating: heating, SourceGrossCents: share, NetCents: share, GrossCents: share}
			if l.VATOpted {
				line.NetCents = nets[i]
				line.VATRate = 10
				if heating || l.UseKind != UseKindWohnung {
					line.VATRate = 20
				}
				line.VATCents = (line.NetCents*int64(line.VATRate) + 50) / 100
				line.GrossCents = line.NetCents + line.VATCents
			}
			a.Lines = append(a.Lines, line)
			if !a.Landlord {
				passed += share
			}
			if heating {
				a.HeatingCents += line.GrossCents
			} else {
				a.OperatingCents += line.GrossCents
			}
		}
		out.SourcePassedCents += passed
		if passed < c.AmountCents {
			excluded := c
			excluded.AmountCents -= passed
			out.Excluded = append(out.Excluded, excluded)
		}
	}
	if total != unit.AllocatedCents {
		return fail("WEG-Kosten stimmen nicht mit dem Eigentümeranteil überein.")
	}
	out.SourceRetainedCents = unit.AllocatedCents - out.SourcePassedCents
	// Rent components are dated monthly Akonto amounts, not the owner's WEG
	// prepayment. Aggregate annual BK for the MRG due-date account.
	var operatingPrepaid int64
	for i := range out.Accounts {
		a := &out.Accounts[i]
		if a.Landlord {
			continue
		}
		l := leaseByID[a.LeaseID]
		if len(a.Intervals) == 0 {
			continue
		}
		bk, e := tenantPrepayments(l, ComponentBKAkonto, a.Intervals)
		if e != nil {
			return fail(e.Error())
		}
		var heat int64
		if legal.HeizKGApplies {
			heat, e = tenantPrepayments(l, ComponentHeizAkonto, a.Intervals)
			if e != nil {
				return fail(e.Error())
			}
		}
		a.HeatingPrepaidCents = heat
		if allFull {
			operatingPrepaid += bk
		} else {
			a.OperatingPrepaidCents = bk
		}
	}
	if allFull {
		out.Accounts[dueIdx].OperatingPrepaidCents = operatingPrepaid
	}
	return out, nil
}

// One equal share per month. Within a partial month use its actual day count.
// Use the exact LCM of 28, 29, 30 and 31 (377580).
func tenantMonthWeight(from, to time.Time) int64 {
	var weight int64
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		days := time.Date(d.Year(), d.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
		weight += 377580 / int64(days)
	}
	return weight
}

func tenantPrepayments(l Lease, kind string, intervals []TenantStatementInterval) (int64, error) {
	total := new(big.Rat)
	for _, interval := range intervals {
		from, _ := time.Parse(time.DateOnly, interval.From)
		to, _ := time.Parse(time.DateOnly, interval.To)
		for day := from; day.Before(to); day = day.AddDate(0, 0, 1) {
			var chosen *RentComponent
			for i := range l.Components {
				c := &l.Components[i]
				if c.Kind == kind && c.ValidFrom <= day.Format(time.DateOnly) && (chosen == nil || c.ValidFrom > chosen.ValidFrom) {
					chosen = c
				}
			}
			if chosen == nil {
				label := "Betriebskosten-Akonto"
				if kind == ComponentHeizAkonto {
					label = "Heizkosten-Akonto"
				}
				return 0, fmt.Errorf("Im %s fehlt das %s. Auch 0,00 € ist einzutragen.", tenantLeaseDescription(l), label)
			}
			if chosen.NetCents < 0 || chosen.NetCents > 1_000_000_000_000 || chosen.VATRateBP < 0 || chosen.VATRateBP > 10000 {
				return 0, fmt.Errorf("Ungültiges Mietakonto.")
			}
			gross := chosen.NetCents + (chosen.NetCents*int64(chosen.VATRateBP)+5000)/10000
			days := time.Date(day.Year(), day.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
			total.Add(total, new(big.Rat).SetFrac64(gross, int64(days)))
		}
	}
	q, r := new(big.Int), new(big.Int)
	q.QuoRem(total.Num(), total.Denom(), r)
	if r.Lsh(r, 1).Cmp(total.Denom()) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	if !q.IsInt64() {
		return 0, fmt.Errorf("Akonto zu groß.")
	}
	return q.Int64(), nil
}

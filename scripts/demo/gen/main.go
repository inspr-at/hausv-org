// Command gen rebuilds the committed, deterministic Hausverwaltung demo data.
// Run from the repository root with:
//
//	go run ./scripts/demo/gen -out scripts/demo/seed
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type unit struct {
	Label       string `json:"label"`
	Floor       string `json:"floor"`
	UnitType    string `json:"unit_type"`
	OwnerEmail  string `json:"owner_email"`
	TenantEmail string `json:"tenant_email"`
}

type house struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Address      string `json:"address"`
	Organisation string `json:"organisation"`
	Units        []unit `json:"units"`
}

type membership struct {
	House string   `json:"house"`
	Role  string   `json:"role"`
	Units []string `json:"units"`
}

type person struct {
	Email       string       `json:"email"`
	Name        string       `json:"name"`
	Title       string       `json:"title"`
	Phone       string       `json:"phone"`
	Memberships []membership `json:"memberships"`
}

type truth struct {
	Category    string `json:"category"`
	Priority    string `json:"priority"`
	Assignee    string `json:"assignee"`
	TemplateKey string `json:"template_key"`
}

type confidence struct {
	Category float64 `json:"category"`
	Priority float64 `json:"priority"`
	House    float64 `json:"house"`
	Unit     float64 `json:"unit"`
	Assignee float64 `json:"assignee"`
	Overall  float64 `json:"overall"`
}

type precomputed struct {
	Category    string     `json:"category"`
	Priority    string     `json:"priority"`
	House       string     `json:"house"`
	Unit        string     `json:"unit"`
	Assignee    string     `json:"assignee"`
	TemplateKey string     `json:"template_key"`
	Reply       string     `json:"reply"`
	Actions     []string   `json:"actions"`
	Confidence  confidence `json:"confidence"`
}

type intakeItem struct {
	ID          string       `json:"id"`
	Source      string       `json:"source"`
	ReceivedAt  string       `json:"received_at"`
	House       string       `json:"house"`
	Unit        string       `json:"unit"`
	FromName    string       `json:"from_name"`
	FromEmail   string       `json:"from_email"`
	FromPhone   string       `json:"from_phone"`
	Subject     string       `json:"subject"`
	Body        string       `json:"body"`
	Truth       truth        `json:"truth"`
	Precomputed *precomputed `json:"precomputed"`
	StatusHint  string       `json:"status_hint"`
}

type textTemplate struct {
	Key          string   `json:"key"`
	Category     string   `json:"category"`
	Title        string   `json:"title"`
	Body         string   `json:"body"`
	Placeholders []string `json:"placeholders"`
}

type event struct {
	ID          string `json:"id"`
	House       string `json:"house"`
	Title       string `json:"title"`
	StartsAt    string `json:"starts_at"`
	EndsAt      string `json:"ends_at"`
	Location    string `json:"location"`
	Description string `json:"description"`
}

type announcement struct {
	ID          string `json:"id"`
	House       string `json:"house"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	Category    string `json:"category"`
	PublishedAt string `json:"published_at"`
}

var houseSpecs = []struct {
	slug, name, address string
	count               int
}{
	{"janusbergweg-123", "Janusbergweg 123", "Janusbergweg 123, 8010 Graz", 18},
	{"grazbachgasse-14", "Grazbachgasse 14", "Grazbachgasse 14, 8010 Graz", 24},
	{"annenstrasse-71", "Annenstraße 71", "Annenstraße 71, 8020 Graz", 30},
	{"muenzgrabenstrasse-9", "Münzgrabenstraße 9", "Münzgrabenstraße 9, 8010 Graz", 12},
	{"schoergelgasse-25", "Schörgelgasse 25", "Schörgelgasse 25, 8010 Graz", 16},
	{"leonhardstrasse-3", "Leonhardstraße 3", "Leonhardstraße 3, 8010 Graz", 10},
	{"koerblergasse-40", "Körblergasse 40", "Körblergasse 40, 8010 Graz", 36},
	{"elisabethstrasse-12", "Elisabethstraße 12", "Elisabethstraße 12, 8010 Graz", 20},
	{"sparbersbachgasse-58", "Sparbersbachgasse 58", "Sparbersbachgasse 58, 8010 Graz", 14},
	{"mariatroster-strasse-101", "Mariatroster Straße 101", "Mariatroster Straße 101, 8043 Graz", 28},
	{"petersgasse-7", "Petersgasse 7", "Petersgasse 7, 8010 Graz", 8},
	{"kaiserfeldgasse-19", "Kaiserfeldgasse 19", "Kaiserfeldgasse 19, 8010 Graz", 22},
}

var residentNames = [][3]string{
	{"Alina Auer", "alina.eigentuemer@musterstadt.example", "+43 664 310 20 01"}, {"Matthias Dorn", "matthias.mieter@musterstadt.example", "+43 676 310 20 02"}, {"Sophie Berger", "sophie.bewohner@musterstadt.example", "+43 660 310 20 03"},
	{"Theresa Fink", "theresa.fink@example.com", "+43 664 310 20 04"}, {"Gregor Haas", "gregor.haas@example.com", "+43 676 310 20 05"}, {"Nora Illek", "nora.illek@example.com", "+43 660 310 20 06"},
	{"Jasmin Kern", "jasmin.kern@example.com", "+43 664 310 20 07"}, {"Lorenz Leitner", "lorenz.leitner@example.com", "+43 676 310 20 08"}, {"Mira Moser", "mira.moser@example.com", "+43 660 310 20 09"},
	{"Daniel Novak", "daniel.novak@example.com", "+43 664 310 20 10"}, {"Petra Ortner", "petra.ortner@example.com", "+43 676 310 20 11"}, {"Ramin Pichler", "ramin.pichler@example.com", "+43 660 310 20 12"},
	{"Elisa Rauch", "elisa.rauch@example.com", "+43 664 310 20 13"}, {"Simon Schober", "simon.schober@example.com", "+43 676 310 20 14"}, {"Leyla Tas", "leyla.tas@example.com", "+43 660 310 20 15"},
	{"Valentin Unger", "valentin.unger@example.com", "+43 664 310 20 16"}, {"Carina Wolf", "carina.wolf@example.com", "+43 676 310 20 17"}, {"Hannes Zeller", "hannes.zeller@example.com", "+43 660 310 20 18"},
	{"Bettina Almer", "bettina.almer@example.com", "+43 664 310 20 19"}, {"Emir Basic", "emir.basic@example.com", "+43 676 310 20 20"}, {"Klara Cerny", "klara.cerny@example.com", "+43 660 310 20 21"},
	{"David Ebner", "david.ebner@example.com", "+43 664 310 20 22"}, {"Fatma Gül", "fatma.guel@example.com", "+43 676 310 20 23"}, {"Josef Hofer", "josef.hofer@example.com", "+43 660 310 20 24"},
	{"Iris Jauk", "iris.jauk@example.com", "+43 664 310 20 25"}, {"Kemal Kaya", "kemal.kaya@example.com", "+43 676 310 20 26"}, {"Lena Lenz", "lena.lenz@example.com", "+43 660 310 20 27"},
	{"Moritz Maier", "moritz.maier@example.com", "+43 664 310 20 28"}, {"Nadine Oswald", "nadine.oswald@example.com", "+43 676 310 20 29"}, {"Peter Reiter", "peter.reiter@example.com", "+43 660 310 20 30"},
	{"Selma Sari", "selma.sari@example.com", "+43 664 310 20 31"}, {"Tobias Thaler", "tobias.thaler@example.com", "+43 676 310 20 32"}, {"Ulrike Weiss", "ulrike.weiss@example.com", "+43 660 310 20 33"},
	{"Florian Zechner", "florian.zechner@example.com", "+43 664 310 20 34"}, {"Gerlinde Binder", "gerlinde.binder@example.com", "+43 676 310 20 35"}, {"Armin Cakir", "armin.cakir@example.com", "+43 660 310 20 36"},
	{"Hedwig Eder", "hedwig.beirat@musterstadt.example", "+43 664 310 20 37"}, {"Ivan Gruber", "ivan.gruber@example.com", "+43 676 310 20 38"}, {"Marlene Karner", "marlene.karner@example.com", "+43 660 310 20 39"},
	{"Oskar Lind", "oskar.lind@example.com", "+43 664 310 20 40"},
}

var categoryCounts = []struct {
	key      string
	count    int
	priority string
	template string
}{
	{"reparatur", 120, "Hoch", "reparatur-beauftragt"}, {"hausordnung", 48, "Mittel", "hausordnung-rueckmeldung"},
	{"betriebskosten", 48, "Mittel", "betriebskosten-pruefung"}, {"freigabe", 32, "Mittel", "freigabe-unterlagen"},
	{"schluessel", 24, "Mittel", "schluessel-bestellung"}, {"beleg", 32, "Niedrig", "beleg-erfasst"},
	{"termin", 24, "Niedrig", "termin-bestaetigt"}, {"versicherung", 16, "Hoch", "versicherung-schaden"},
	{"uebergabe", 16, "Mittel", "uebergabe-termin"}, {"stammdaten", 12, "Niedrig", "stammdaten-geaendert"},
	{"winterdienst_garten", 16, "Mittel", "aussenanlage-auftrag"}, {"parkplatz", 8, "Mittel", "parkplatz-pruefung"},
	{"sonstiges", 4, "Mittel", "sonstiges-rueckfrage"},
}

func main() {
	out := flag.String("out", "scripts/demo/seed", "output directory")
	flag.Parse()
	if err := generate(*out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(out string) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	houses, persons := buildHousesAndPersons()
	templates := demoTemplates()
	items, err := buildIntake(houses, persons, templates)
	if err != nil {
		return err
	}
	files := map[string]any{
		"houses.json": houses, "persons.json": persons, "intake.json": items,
		"textbausteine.json": templates, "events.json": buildEvents(houses),
		"announcements.json": buildAnnouncements(houses), "org.json": buildOrg(),
	}
	for name, value := range files {
		if err := writeJSON(filepath.Join(out, name), value); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	return nil
}

func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func buildHousesAndPersons() ([]house, []person) {
	people := []person{
		{Email: "vera.verwalter@musterstadt.example", Name: "Vera Verwalter", Phone: "+43 316 555 100", Memberships: []membership{}},
		{Email: "paul.verwalter@musterstadt.example", Name: "Paul Sommer", Phone: "+43 316 555 101", Memberships: []membership{}},
	}
	houses := make([]house, 0, len(houseSpecs))
	for hi, spec := range houseSpecs {
		people[0].Memberships = append(people[0].Memberships, membership{House: spec.slug, Role: "Admin", Units: []string{}})
		people[1].Memberships = append(people[1].Memberships, membership{House: spec.slug, Role: "Verwalter", Units: []string{}})
		base := hi * 3
		roles := []string{"Eigentümer", "Mieter", "Bewohner"}
		labels := []string{"Top 1", "Top 3", "Top 7"}
		residentStart := len(people)
		for pi := 0; pi < 3; pi++ {
			n := residentNames[base+pi]
			people = append(people, person{Email: fixtureEmail(n[1]), Name: n[0], Phone: n[2], Memberships: []membership{{House: spec.slug, Role: roles[pi], Units: []string{labels[pi]}}}})
		}
		if hi < 4 {
			n := residentNames[36+hi]
			people = append(people, person{Email: fixtureEmail(n[1]), Name: n[0], Phone: n[2], Memberships: []membership{{House: spec.slug, Role: "Beirat", Units: []string{"Top 2"}}}})
		}
		units := make([]unit, 0, spec.count+parkingCount(spec.count))
		for u := 1; u <= spec.count; u++ {
			floor := fmt.Sprintf("%d. OG", (u-1)/4+1)
			if u <= 4 {
				floor = "EG"
			}
			units = append(units, unit{Label: fmt.Sprintf("Top %d", u), Floor: floor, UnitType: "Wohnung"})
		}
		for parking := 1; parking <= parkingCount(spec.count); parking++ {
			label := fmt.Sprintf("Stellplatz %d", parking)
			// Every fourth parking space stays free. The other spaces are assigned
			// round-robin to the people whose memberships belong to this house.
			if parking%4 != 0 {
				people[residentStart+(parking-1)%3].Memberships[0].Units = append(people[residentStart+(parking-1)%3].Memberships[0].Units, label)
			}
			units = append(units, unit{Label: label, Floor: "Garage", UnitType: "Stellplatz"})
		}
		for index := range units {
			units[index].OwnerEmail, units[index].TenantEmail = unitParties(people, spec.slug, units[index].Label)
		}
		houses = append(houses, house{Slug: spec.slug, Name: spec.name, Address: spec.address, Organisation: "musterstadt", Units: units})
	}
	return houses, people
}

func fixtureEmail(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasSuffix(strings.ToLower(value), ".example") {
		return value
	}
	return strings.TrimSuffix(value, ".com") + ".example"
}

func parkingCount(unitCount int) int {
	return 4 + (unitCount-8)/4
}

func unitParties(people []person, house, label string) (owner, tenant string) {
	for _, person := range people {
		for _, membership := range person.Memberships {
			if membership.House != house || !containsUnit(membership.Units, label) {
				continue
			}
			switch membership.Role {
			case "Eigentümer":
				owner = person.Email
			case "Mieter", "Bewohner":
				tenant = person.Email
			}
		}
	}
	return owner, tenant
}

func containsUnit(units []string, label string) bool {
	for _, unit := range units {
		if unit == label {
			return true
		}
	}
	return false
}

func buildIntake(houses []house, persons []person, templates []textTemplate) ([]intakeItem, error) {
	templateByKey := map[string]textTemplate{}
	for _, item := range templates {
		templateByKey[item.Key] = item
	}
	materialByCategory := map[string][]requestMaterial{}
	for _, item := range requestMaterials {
		materialByCategory[item.Category] = append(materialByCategory[item.Category], item)
	}
	items := make([]intakeItem, 0, 400)
	autoDone, todayOther, approved, olderOther := 0, 0, 0, 0
	index := 0
	for round := 0; ; round++ {
		added := false
		for ci, spec := range categoryCounts {
			if round >= spec.count {
				continue
			}
			added = true
			index++
			houseIndex := (index*7 + ci*3) % len(houses)
			h := houses[houseIndex]
			resident := residentForHouse(persons, h.Slug, index)
			unitLabel := resident.Memberships[0].Units[0]
			unassigned := index%20 == 0
			itemHouse := h.Slug
			if unassigned {
				itemHouse, unitLabel = "", ""
			}
			isAuto := (spec.key == "beleg" || spec.key == "termin") && autoDone < 30
			dayOffset := 1 + index%2
			status := "new"
			if isAuto {
				dayOffset, status = 0, "auto_done"
				autoDone++
			} else if todayOther < 10 {
				dayOffset = 0
				todayOther++
			} else if approved < 150 {
				dayOffset, status = 3+(index*5)%12, "approved"
				approved++
			} else if olderOther < 40 {
				dayOffset = 3 + (index*5)%12
				olderOther++
				if index%3 == 0 {
					status = "manual"
				}
			} else if index%9 == 0 {
				status = "manual"
			}
			// Concentrate open work on six houses so the portfolio shows both
			// "Handlungsbedarf" and "ruhig"; older, handled items spread over all twelve.
			if status == "new" || status == "manual" || (status == "approved" && (index-1)%4 == 0) {
				h = houses[houseIndex%6]
				resident = residentForHouse(persons, h.Slug, index)
				unitLabel = resident.Memberships[0].Units[0]
				itemHouse = h.Slug
				if unassigned {
					itemHouse, unitLabel = "", ""
				}
			}
			received := time.Date(2026, 9, 9-dayOffset, 7+(index*3)%11, (index*17)%60, 0, 0, time.FixedZone("CEST", 2*60*60))
			material := materialByCategory[spec.key][round%len(materialByCategory[spec.key])]
			body := expand(material.Body, h, unitLabel, resident.Name, index)
			subject := expand(material.Subject, h, unitLabel, resident.Name, index)
			assignee := "vera.verwalter"
			if index%3 == 0 {
				assignee = "paul.sommer"
			}
			tr := truth{Category: spec.key, Priority: spec.priority, Assignee: assignee, TemplateKey: spec.template}
			item := intakeItem{
				ID: fmt.Sprintf("in-%04d", index), Source: sourceFor(index), ReceivedAt: received.Format(time.RFC3339), House: itemHouse,
				Unit: unitLabel, FromName: resident.Name, FromEmail: resident.Email, Subject: subject, Body: body, Truth: tr, StatusHint: status,
			}
			if item.Source == "phone" {
				item.FromPhone = resident.Phone
			}
			if index%5 != 0 || isAuto {
				overall := 0.88 + float64(index%8)/100
				if status == "auto_done" {
					overall = 0.94
				}
				t := templateByKey[spec.template]
				item.Precomputed = &precomputed{
					Category: spec.key, Priority: spec.priority, House: itemHouse, Unit: unitLabel, Assignee: assignee, TemplateKey: spec.template,
					Reply: renderReply(t.Body, resident.Name, h, unitLabel, assignee), Actions: actionsFor(spec.key),
					Confidence: confidence{Category: 0.92, Priority: 0.86, House: confidenceFor(itemHouse), Unit: confidenceFor(unitLabel), Assignee: 0.89, Overall: overall},
				}
			}
			items = append(items, item)
		}
		if !added {
			break
		}
	}
	return items, nil
}

func residentForHouse(persons []person, houseSlug string, n int) person {
	var matches []person
	for _, p := range persons {
		if len(p.Memberships) == 1 && p.Memberships[0].House == houseSlug && len(p.Memberships[0].Units) > 0 {
			matches = append(matches, p)
		}
	}
	return matches[n%len(matches)]
}

func sourceFor(index int) string {
	switch index % 20 {
	case 0, 1, 2, 3, 4:
		return "phone"
	case 5, 6, 7:
		return "portal"
	default:
		return "email"
	}
}

func confidenceFor(value string) float64 {
	if value == "" {
		return 0.42
	}
	return 0.96
}

func expand(value string, h house, unitLabel, name string, n int) string {
	amount := []string{"184,20", "76,80", "1.248,00", "329,50"}[n%4]
	replacer := strings.NewReplacer("{{Haus}}", h.Name, "{{Adresse}}", h.Address, "{{Einheit}}", unitLabel, "{{Name}}", name, "{{Betrag}}", amount, "{{Datum}}", fmt.Sprintf("%02d.09.2026", 10+n%8))
	return replacer.Replace(value)
}

func renderReply(body, name string, h house, unitLabel, assignee string) string {
	parts := strings.Fields(name)
	last := parts[len(parts)-1]
	zustaendig := "Vera Verwalter"
	if assignee == "paul.sommer" {
		zustaendig = "Paul Sommer"
	}
	replacer := strings.NewReplacer(
		"{{Anrede}}", salutationFor(parts[0]), "{{Name}}", last, "{{Haus}}", h.Name, "{{Einheit}}", unitLabel,
		"{{Nummer}}", "HV-2026-09", "{{Handwerker}}", "unseren zuständigen Fachbetrieb", "{{Frist}}", "zwei Werktagen", "{{Zuständig}}", zustaendig,
	)
	return replacer.Replace(body)
}

func actionsFor(category string) []string {
	switch category {
	case "reparatur":
		return []string{"Anliegen anlegen", "Fachbetrieb verständigen"}
	case "beleg":
		return []string{"Beleg erfassen", "Kostenstelle zuordnen"}
	case "termin":
		return []string{"Termin eintragen", "Bestätigung senden"}
	default:
		return []string{"Anliegen anlegen", "Zuständige Person informieren"}
	}
}

func buildEvents(houses []house) []event {
	var out []event
	for hi, h := range houses {
		for j := 0; j < 3; j++ {
			day := 7 + (hi+j*4)%12
			start := time.Date(2026, 9, day, 9+j*4, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
			titles := []string{"Eigentümerversammlung", "Liftwartung", "Begehung der Allgemeinflächen"}
			locations := []string{"Gemeinschaftsraum", "Stiegenhaus", "Hauseingang"}
			out = append(out, event{ID: fmt.Sprintf("ev-%03d", len(out)+1), House: h.Slug, Title: titles[j], StartsAt: start.Format(time.RFC3339), EndsAt: start.Add(time.Duration(2-j/2) * time.Hour).Format(time.RFC3339), Location: locations[j], Description: "Termin für " + h.Name + ". Bitte Aushang und Zugang beachten."})
		}
	}
	return out
}

func buildAnnouncements(houses []house) []announcement {
	var out []announcement
	for hi, h := range houses {
		entries := []struct{ title, body, category string }{
			{"Wasserabschaltung am 11.09., 9–12 Uhr", "Wegen Arbeiten an der Steigleitung wird das Wasser vorübergehend abgestellt. Bitte halten Sie die Absperrhähne frei.", "Wartung"},
			{"Hausbegehung in der Demo-Woche", "Die Hausverwaltung prüft die Allgemeinflächen. Hinweise können vorab über das Portal gemeldet werden.", "Termin"},
		}
		if hi%2 == 0 {
			entries = append(entries, struct{ title, body, category string }{"Bitte Fluchtwege freihalten", "Kinderwägen, Fahrräder und Kartons dürfen nicht im Stiegenhaus abgestellt werden.", "Info"})
		}
		for _, e := range entries {
			out = append(out, announcement{ID: fmt.Sprintf("an-%03d", len(out)+1), House: h.Slug, Title: e.title, Body: e.body, Category: e.category, PublishedAt: "2026-09-08T08:00:00+02:00"})
		}
	}
	return out
}

func buildOrg() map[string]any {
	trust := map[string]string{}
	for _, c := range categoryCounts {
		trust[c.key] = "propose"
	}
	trust["beleg"], trust["termin"] = "auto", "auto"
	return map[string]any{
		"key": "musterstadt", "name": "Hausverwaltung Musterstadt GmbH", "trust_levels": trust,
		"auto_threshold": 0.9, "auto_enabled": true,
		"assignees": []map[string]string{{"key": "vera.verwalter", "email": "vera.verwalter@musterstadt.example", "name": "Vera Verwalter"}, {"key": "paul.sommer", "email": "paul.verwalter@musterstadt.example", "name": "Paul Sommer"}},
	}
}

// salutationFor completes "Sehr geehrte{{Anrede}}" for the fixture persons:
// " Frau" for the female first names in the material, "r Herr" otherwise.
func salutationFor(firstName string) string {
	female := map[string]bool{"Anna": true, "Maria": true, "Rita": true, "Nora": true, "Clara": true, "Bianca": true, "Sabine": true, "Petra": true, "Eva": true, "Julia": true, "Lena": true, "Sophie": true, "Katharina": true, "Ines": true, "Elisabeth": true, "Barbara": true, "Monika": true, "Andrea": true, "Christine": true, "Ursula": true, "Gerlinde": true, "Helga": true, "Renate": true, "Brigitte": true, "Claudia": true, "Martina": true, "Sandra": true, "Nicole": true, "Verena": true, "Tanja": true, "Laura": true, "Sarah": true, "Lisa": true, "Marlene": true, "Theresa": true, "Johanna": true, "Vera": true, "Hanna": true, "Emma": true, "Mia": true, "Lea": true, "Nina": true, "Silvia": true, "Margit": true, "Karin": true, "Doris": true, "Gabriele": true, "Birgit": true, "Ingrid": true, "Susanne": true, "Michaela": true, "Daniela": true, "Bettina": true, "Kerstin": true, "Alexandra": true, "Angelika": true, "Waltraud": true, "Hermine": true, "Leonie": true, "Valentina": true, "Magdalena": true, "Franziska": true, "Carina": true, "Melanie": true, "Stefanie": true, "Jasmin": true, "Simone": true, "Manuela": true, "Elke": true, "Astrid": true, "Iris": true, "Sonja": true, "Alina": true, "Elisa": true, "Fatma": true, "Hedwig": true, "Klara": true, "Leyla": true, "Mira": true, "Nadine": true, "Selma": true, "Sibel": true, "Ulrike": true}
	if female[firstName] {
		return " Frau"
	}
	return "r Herr"
}

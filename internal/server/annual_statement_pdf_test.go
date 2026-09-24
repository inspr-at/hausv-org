package server

import (
	"bytes"
	"encoding/json"
	"mime"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestAnnualStatementPDFDownloadPermissionsSnapshotAndNoWrites(t *testing.T) {
	const manager = "manager@example.com"
	a := newTestPortalApp(t, userProfile{Email: manager, Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	repos := testRepositories(a, "demo")
	units := []store.Unit{{ID: "a", Label: "Top Ä 1", MiteigentumsanteilPPM: 250000, OwnerEmails: []string{"owner@example.com", "coowner@example.com"}, RenterEmails: []string{"tenant@example.com"}, PartyContacts: []store.UnitPartyContact{{Email: "owner@example.com", Name: "Anna Eigentümer", Address: "Gasse 1\n8010 Graz"}}}, {ID: "b", Label: "Top 2", MiteigentumsanteilPPM: 750000, OwnerEmails: []string{"other@example.com"}}}
	if err := repos.units.SetUnits(units); err != nil {
		t.Fatal(err)
	}
	period := store.AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31", UpdatedBy: manager}
	costs := []store.AnnualStatementCostType{{Key: "water", Name: "Wasser", Allocatable: true, AllocationKey: store.AllocationKeyNutzwert, UpdatedBy: manager}}
	if _, err := repos.annualStatementPeriods.SaveWithStructure(period, costs, units); err != nil {
		t.Fatal(err)
	}
	doc, err := repos.documents.CreateGenerated(store.DocumentRecord{Title: "Wasser", Category: store.DocumentCategoryBilling, Visibility: store.DocumentVisibilityManagerOnly, UploadedBy: manager}, "water.pdf", "application/pdf", []byte("%PDF-1.4 original"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := repos.annualStatementReceipts.Create(store.AnnualStatementReceipt{DocumentID: doc.ID, PeriodYear: 2025, CostTypeKey: "water", AmountCents: 10000, InvoiceDate: "2025-02-01", CreatedBy: manager})
	if err != nil {
		t.Fatal(err)
	}
	for _, unit := range units {
		if _, _, err := repos.annualStatementAkontos.Save(store.AnnualStatementPrepayment{PeriodYear: 2025, UnitID: unit.ID, AmountCents: 3000, UpdatedBy: manager}); err != nil {
			t.Fatal(err)
		}
	}
	created := authedFormRequest(t, a, manager, "/demo/app/settings/annual-statement/runs", url.Values{"year": {"2025"}})
	if created.Code != http.StatusSeeOther {
		t.Fatal(created.Code)
	}
	runs, _ := repos.annualStatementRuns.List(2025)
	if len(runs) != 1 || len(runs[0].Input.Parties) != 4 || runs[0].Input.Presentation.EstateName == "" {
		t.Fatalf("missing snapshot: %+v", runs)
	}
	run := runs[0]
	route := "/demo" + annualStatementPDFURL(run.ID, "a", "owner@example.com")
	before := authedRequest(t, a, manager, route)
	if before.Code != 200 || before.Header().Get("Content-Type") != "application/pdf" || !bytes.HasPrefix(before.Body.Bytes(), []byte("%PDF-")) {
		t.Fatalf("download=%d %s", before.Code, before.Body.String())
	}
	kind, params, err := mime.ParseMediaType(before.Header().Get("Content-Disposition"))
	if err != nil || kind != "inline" || !strings.HasSuffix(params["filename"], ".pdf") {
		t.Fatal(params, err)
	}
	attachment := authedRequest(t, a, manager, route+"&download=1")
	attachmentKind, attachmentParams, attachmentErr := mime.ParseMediaType(attachment.Header().Get("Content-Disposition"))
	if attachment.Code != http.StatusOK || attachmentErr != nil || attachmentKind != "attachment" || attachmentParams["filename"] != params["filename"] || !bytes.Equal(before.Body.Bytes(), attachment.Body.Bytes()) {
		t.Fatal("explicit download must attach the same PDF with the same filename")
	}
	for _, c := range params["filename"] {
		if c > 127 || c == '/' || c == '\\' || c == '\r' || c == '\n' {
			t.Fatal("unsafe filename")
		}
	}
	for _, tc := range []struct {
		actor, path string
		code        int
	}{{"resident@example.com", route, 403}, {manager, "/demo" + annualStatementPDFURL("missing", "a", "owner@example.com"), 404}, {manager, "/demo" + annualStatementPDFURL(run.ID, "missing", "owner@example.com"), 404}, {manager, "/demo" + annualStatementPDFURL(run.ID, "a", "other@example.com"), 404}, {manager, "/demo" + annualStatementPDFURL(run.ID, "", "") + "?unit=a", 404}, {manager, "/demo" + annualStatementPDFURL(run.ID, "", "") + "?unit=&party=", 404}, {manager, "/demo" + annualStatementPDFURL(run.ID, "", "") + "?unit=a&unit=b&party=owner%40example.com", 404}} {
		if response := authedRequest(t, a, tc.actor, tc.path); response.Code != tc.code {
			t.Errorf("%s got %d want %d", tc.path, response.Code, tc.code)
		}
	}
	foreign := testRepositories(a, "other")
	if got, found, err := foreign.annualStatementRuns.Get(run.ID); err != nil || found || got.ID != "" {
		t.Fatal("foreign run visible", err)
	}
	if _, err := repos.annualStatementReceipts.UpdateAmount(receipt.ID, 90000, manager); err != nil {
		t.Fatal(err)
	}
	if _, err := repos.units.UpdateParties([]store.UnitPartyUpdate{{UnitID: "a", SetOwners: true, OwnerEmails: []string{"new@example.com"}, Contacts: []store.UnitPartyContact{{Email: "new@example.com", Address: "Neue Adresse"}}}}); err != nil {
		t.Fatal(err)
	}
	tenant := a.tenants["demo"]
	tenant.Name = "Neue Liegenschaft"
	tenant.ContactName = "Andere Verwaltung"
	a.tenants["demo"] = tenant
	// Snapshot all mutable repositories and audit after the deliberate source edits.
	state := func() string {
		runs, _ := repos.annualStatementRuns.List(2025)
		value := []any{runs, repos.units.List(), repos.annualStatementPeriods.List(), repos.annualStatementReceipts.ListByPeriod(2025), repos.annualStatementAkontos.ListByPeriod(2025), repos.documents.List(), a.auditStore.List(auditFilter{TenantSlug: "demo", Limit: 100})}
		raw, _ := json.Marshal(value)
		return string(raw)
	}
	saved := state()
	after := authedRequest(t, a, manager, route)
	if after.Code != 200 || !bytes.Equal(before.Body.Bytes(), after.Body.Bytes()) {
		t.Fatal("source edit changed stored PDF")
	}
	all := authedRequest(t, a, manager, "/demo"+annualStatementPDFURL(run.ID, "", ""))
	if all.Code != 200 || !bytes.Contains(all.Body.Bytes(), []byte("/Count 4")) {
		t.Fatal("bulk PDF missing party pages", all.Code)
	}
	if !reflect.DeepEqual(saved, state()) {
		t.Fatal("GET wrote data or audit")
	}
	page := authedRequest(t, a, manager, "/demo/app/settings/annual-statement?year=2025&run="+run.ID)
	for _, want := range []string{"Alle Dokumente (PDF)", "PDF für Anna Eigentümer", "PDF für coowner@example.com", "PDF für tenant@example.com"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Errorf("missing link %s", want)
		}
	}
}

func TestAnnualStatementCSVStoresOptionalAddressWithoutInventingIt(t *testing.T) {
	rows, err := parseAnnualStatementPartyCSV([]byte("Einheit;Rolle;E-Mail;Name;Anschrift\nTop 1;Wohnungseigentümer;anna@example.com;Anna Groß;Gasse 2, 8010 Graz\nTop 1;Mietverhältnis;mieter@example.com;;\n"))
	if err != nil {
		t.Fatal(err)
	}
	updates, count, err := annualStatementPartyUpdates([]unit{{ID: "top-1", Label: "Top 1"}}, rows)
	if err != nil || count != 2 || len(updates) != 1 || len(updates[0].Contacts) != 2 {
		t.Fatal(updates, count, err)
	}
	if updates[0].Contacts[0].Name != "Anna Groß" || updates[0].Contacts[0].Address != "Gasse 2, 8010 Graz" || updates[0].Contacts[1].Address != "" {
		t.Fatal(updates)
	}
}

func TestAnnualStatementCSVPartialContactColumnsPreserveAddress(t *testing.T) {
	rows, err := parseAnnualStatementPartyCSV([]byte("Einheit;Rolle;E-Mail;Name\nTop 1;Wohnungseigentümer;anna@example.com;Neuer Name\n"))
	if err != nil {
		t.Fatal(err)
	}
	units := []unit{{ID: "top-1", Label: "Top 1", PartyContacts: []store.UnitPartyContact{{Email: "anna@example.com", Name: "Alt", Address: "Bestehende Anschrift"}}}}
	updates, _, err := annualStatementPartyUpdates(units, rows)
	if err != nil || updates[0].Contacts[0].Address != "Bestehende Anschrift" || updates[0].Contacts[0].Name != "Neuer Name" {
		t.Fatal(updates, err)
	}
}

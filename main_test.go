package main

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"math"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

type sentNotification struct {
	To      string
	Subject string
	Body    string
}

type recordingMailer struct {
	notifications []sentNotification
}

func (m *recordingMailer) SendMagicLink(string, string) error      { return nil }
func (m *recordingMailer) SendInvite(string, string, string) error { return nil }
func (m *recordingMailer) Configured() bool                        { return true }
func (m *recordingMailer) SendNotification(to string, subject string, body string) error {
	m.notifications = append(m.notifications, sentNotification{To: to, Subject: subject, Body: body})
	return nil
}

func TestSMTPMailerAllowsInternalRelayWithoutAuth(t *testing.T) {
	m := smtpMailer{
		host: "smtp",
		port: "25",
		from: "WEG Portal <noreply@hausv.org>",
	}

	if !m.Configured() {
		t.Fatal("mailer should be configured with host, port, and from")
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("relay mailer should validate without auth: %v", err)
	}
	if auth := m.auth(); auth != nil {
		t.Fatal("relay mailer without user/pass should not create smtp auth")
	}
}

func TestSMTPMailerRequiresPairedCredentials(t *testing.T) {
	m := smtpMailer{
		host: "smtp",
		port: "25",
		user: "user",
		from: "WEG Portal <noreply@hausv.org>",
	}

	if err := m.Validate(); err == nil {
		t.Fatal("mailer should reject partial smtp credentials")
	}
}

func TestPageTemplatesConsolidateDesignTokensAndComponents(t *testing.T) {
	wants := []string{
		`{{define "designTokens"}}`,
		`{{template "designTokens" .}}`,
		`--font-sans:`,
		`--space-6:24px`,
		`--radius-sm:8px`,
		`--shadow-panel:0 12px 30px rgba(32,37,31,.04)`,
		`Shared components: panel, button, pill, quick-row, table-wrap, dialog, flash and empty-state.`,
		`.panel { background: var(--panel); border: 1px solid var(--line); border-radius: var(--radius-sm); padding: var(--space-6); box-shadow: var(--shadow-panel); }`,
		`.empty-state { border: 1px solid var(--line); border-radius: var(--radius-sm);`,
		`.parking-access .access-table, .parking-access .access-table tbody, .parking-access .access-table tr, .parking-access .access-table td { display: block; width: 100%; min-width: 0; }`,
	}
	for _, want := range wants {
		if !strings.Contains(pageTemplates, want) {
			t.Fatalf("pageTemplates missing shared design-system marker %q", want)
		}
	}
	if got := strings.Count(pageTemplates, "--ink:#20251f"); got != 1 {
		t.Fatalf("color token block is duplicated %d times, want once", got)
	}
	if got := strings.Count(pageTemplates, `{{template "designTokens" .}}`); got != 2 {
		t.Fatalf("design token partial is used %d times, want home and app styles", got)
	}
}

func TestPageTemplatesExposeAccessibilityConventions(t *testing.T) {
	wants := []string{
		`:where(a, button, input, select, textarea, summary, [tabindex]):focus-visible`,
		`aria-haspopup="dialog" aria-controls="announcement-create"`,
		`<dialog id="announcement-create" class="dialog" aria-labelledby="announcement-create-title">`,
		`aria-haspopup="dialog" aria-controls="event-create"`,
		`<dialog id="event-create" class="dialog" aria-labelledby="event-create-title">`,
		`aria-describedby="role-help"`,
		`<span id="role-help" class="popup" role="tooltip">`,
		`aria-label="E-Mail-Adresse" autocomplete="email" required`,
		`aria-label="Kommentar oder Rückfrage"`,
		`role="region" aria-label="Monatsabrechnung Parkplatznutzung"`,
		`<caption class="sr-only">Monatsabrechnung Parkplatznutzung`,
		`<th scope="row" class="month-cell">`,
		`role="region" aria-label="Stundenwerte Parkplatznutzung"`,
		`<caption class="sr-only">Stundenwerte Parkplatznutzung`,
	}
	for _, want := range wants {
		if !strings.Contains(pageTemplates, want) {
			t.Fatalf("pageTemplates missing accessibility convention %q", want)
		}
	}
	for _, path := range []string{"assets/announcements.js", "assets/users.js"} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(body)
		for _, want := range []string{"dialogTriggers", `aria-expanded`, "focusFirstDialogField"} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing dialog focus convention %q", path, want)
			}
		}
	}
}

func TestFormattingUsesDeATConventions(t *testing.T) {
	at := time.Date(2026, 7, 6, 9, 5, 0, 0, time.Local)
	tests := map[string]string{
		"date":        formatLocalDate(at),
		"datetime":    formatLocalDateTime(at),
		"input":       formatLocalDateTimeInput(at),
		"decimal":     formatDecimal(1234567.89, 2),
		"negative":    formatDecimal(-1234.5, 1),
		"eur_per_kwh": formatEURPerKWh(0.1234),
		"share":       formatMiteigentumsanteil(12345),
		"month":       formatMonthLabel("2026-01", time.Local),
	}
	wants := map[string]string{
		"date":        "06.07.2026",
		"datetime":    "06.07.2026 09:05",
		"input":       "2026-07-06T09:05",
		"decimal":     "1.234.567,89",
		"negative":    "-1.234,5",
		"eur_per_kwh": "0,123 €/kWh",
		"share":       "12.345 / 1.000.000",
		"month":       "Jänner 2026",
	}
	for name, got := range tests {
		if got != wants[name] {
			t.Fatalf("%s = %q, want %q", name, got, wants[name])
		}
	}
}

func TestInviteStoreAddDedupeGetList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invites.json")
	store, err := newInviteStore(path)
	if err != nil {
		t.Fatalf("newInviteStore: %v", err)
	}
	p := userProfile{Email: "New.Person@example.com", FirstName: "New", LastName: "Person", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	added, err := store.Add(p)
	if err != nil || !added {
		t.Fatalf("first Add: added=%v err=%v", added, err)
	}
	// dedupe is case-insensitive and must not overwrite the stored profile
	again, err := store.Add(userProfile{Email: "new.person@example.com", Role: roleAdmin})
	if err != nil {
		t.Fatalf("dedupe Add err: %v", err)
	}
	if again {
		t.Fatal("dedupe Add should return false for an existing email")
	}
	got, ok := store.Get("NEW.PERSON@example.com")
	if !ok || got.LastName != "Person" || got.Role != roleResident {
		t.Fatalf("Get returned %+v ok=%v; dedupe must not have overwritten the role", got, ok)
	}
	if n := len(store.List()); n != 1 {
		t.Fatalf("List len = %d, want 1", n)
	}
	reopened, err := newInviteStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if _, ok := reopened.Get("new.person@example.com"); !ok {
		t.Fatal("invite did not persist across reopen")
	}
}

func TestUnitStoreSetListResolvePersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "units.json")
	store, err := newUnitStore(path)
	if err != nil {
		t.Fatalf("newUnitStore: %v", err)
	}
	if err := store.SetTenantUnits("jhw22", []unit{
		{
			ID:                    "Top_2",
			Label:                 "Top 2",
			MiteigentumsanteilPPM: 12345,
			OwnerEmails:           []string{"Owner@Example.com", "owner@example.com"},
			RenterEmails:          []string{"Renter@Example.com"},
		},
		{
			Label:                 "Top 1",
			MiteigentumsanteilPPM: 22222,
			OwnerEmails:           []string{"second-owner@example.com"},
		},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}

	units := store.ListTenant("JHW22")
	if len(units) != 2 {
		t.Fatalf("ListTenant len = %d, want 2", len(units))
	}
	if units[0].ID != "top-1" || units[0].Label != "Top 1" || units[1].ID != "top-2" {
		t.Fatalf("units not normalized/sorted: %+v", units)
	}
	if got := units[1].OwnerEmails; len(got) != 1 || got[0] != "owner@example.com" {
		t.Fatalf("owners not normalized/deduped: %+v", got)
	}
	if units[1].MiteigentumsanteilPPM != 12345 {
		t.Fatalf("share = %d, want 12345", units[1].MiteigentumsanteilPPM)
	}

	memberships := store.UnitsForEmail("jhw22", "OWNER@example.com")
	if len(memberships) != 1 || memberships[0].Unit.ID != "top-2" || memberships[0].Relation != roleOwner {
		t.Fatalf("owner memberships = %+v", memberships)
	}
	renterMemberships := store.UnitsForEmail("jhw22", "renter@example.com")
	if len(renterMemberships) != 1 || renterMemberships[0].Relation != roleRenter {
		t.Fatalf("renter memberships = %+v", renterMemberships)
	}
	members := store.MembersForUnit("jhw22", "top-2")
	if !members.Found || len(members.Owners) != 1 || members.Owners[0] != "owner@example.com" || len(members.Renters) != 1 || members.Renters[0] != "renter@example.com" {
		t.Fatalf("members = %+v", members)
	}

	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("store file mode = %v err=%v, want 0600", info.Mode().Perm(), err)
	}
	reopened, err := newUnitStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if got := reopened.UnitCount("jhw22"); got != 2 {
		t.Fatalf("reopened UnitCount = %d, want 2", got)
	}
}

func TestVoteStoreCreateOpenCastClosePersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "votes.json")
	store, err := newVoteStore(path)
	if err != nil {
		t.Fatalf("newVoteStore: %v", err)
	}
	created, err := store.Create(ballot{
		TenantSlug:  "JHW22",
		Title:       "Ladestation beschließen",
		Description: "Soll eine Wallbox angeschafft werden?",
		Options:     []string{"Ja", "Nein", "Ja"},
		Type:        ballotTypeCircular,
		Weighting:   ballotWeightingPerShare,
		QuorumPPM:   500000,
		ClosesAt:    time.Now().Add(24 * time.Hour),
		CreatedBy:   "Manager@Example.com",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" || created.Status != ballotStatusDraft || created.TenantSlug != "jhw22" || len(created.Options) != 2 || created.CreatedBy != "manager@example.com" {
		t.Fatalf("created ballot = %+v", created)
	}
	opened, found, err := store.Open("jhw22", created.ID, time.Now())
	if err != nil || !found || opened.Status != ballotStatusOpen {
		t.Fatalf("Open = %+v found=%v err=%v", opened, found, err)
	}
	voted, found, err := store.CastVote("jhw22", created.ID, "Owner@Example.com", "Ja", 12345, time.Now())
	if err != nil || !found {
		t.Fatalf("CastVote first found=%v err=%v", found, err)
	}
	if vote := voted.Votes["owner@example.com"]; vote.Option != "Ja" || vote.Weight != 12345 {
		t.Fatalf("first vote = %+v", vote)
	}
	voted, found, err = store.CastVote("jhw22", created.ID, "owner@example.com", "Nein", 12345, time.Now())
	if err != nil || !found || len(voted.Votes) != 1 || voted.Votes["owner@example.com"].Option != "Nein" {
		t.Fatalf("mutable vote = %+v found=%v err=%v", voted.Votes, found, err)
	}
	closed, found, err := store.Close("jhw22", created.ID, time.Now())
	if err != nil || !found || closed.Status != ballotStatusClosed {
		t.Fatalf("Close = %+v found=%v err=%v", closed, found, err)
	}
	if _, _, err := store.CastVote("jhw22", created.ID, "owner@example.com", "Ja", 12345, time.Now()); err == nil {
		t.Fatal("closed ballot should reject votes")
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("vote store file mode = %v err=%v, want 0600", info.Mode().Perm(), err)
	}
	reopened, err := newVoteStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	loaded, ok := reopened.Get("jhw22", created.ID)
	if !ok || loaded.Status != ballotStatusClosed || loaded.Votes["owner@example.com"].Option != "Nein" {
		t.Fatalf("loaded ballot = %+v ok=%v", loaded, ok)
	}
}

func TestBallotVoteWeightUsesOwnerUnits(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["owner@example.com"] = userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	a.profiles["renter@example.com"] = userProfile{Email: "renter@example.com", Role: roleRenter, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	if err := a.unitStore.SetTenantUnits("jhw22", []unit{
		{ID: "top-1", TenantSlug: "jhw22", Label: "Top 1", MiteigentumsanteilPPM: 12345, OwnerEmails: []string{"owner@example.com"}, RenterEmails: []string{"renter@example.com"}},
		{ID: "top-2", TenantSlug: "jhw22", Label: "Top 2", MiteigentumsanteilPPM: 22222, OwnerEmails: []string{"owner@example.com"}},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	shareBallot, err := a.voteStore.Create(ballot{
		TenantSlug: "jhw22",
		Title:      "Sanierung",
		Options:    []string{"Ja", "Nein"},
		Type:       ballotTypeCircular,
		Weighting:  ballotWeightingPerShare,
		CreatedBy:  "manager@example.com",
	})
	if err != nil {
		t.Fatalf("Create share ballot: %v", err)
	}
	if _, _, err := a.voteStore.Open("jhw22", shareBallot.ID, time.Now()); err != nil {
		t.Fatalf("Open share ballot: %v", err)
	}
	updated, found, err := a.castBallotVote("jhw22", "owner@example.com", shareBallot.ID, "Ja", time.Now())
	if err != nil || !found {
		t.Fatalf("owner cast share vote found=%v err=%v", found, err)
	}
	if got := updated.Votes["owner@example.com"].Weight; got != 34567 {
		t.Fatalf("owner share vote weight = %d, want 34567", got)
	}
	if _, _, err := a.castBallotVote("jhw22", "renter@example.com", shareBallot.ID, "Ja", time.Now()); err == nil {
		t.Fatal("renter should not be eligible for owner ballot")
	}
	headBallot, err := a.voteStore.Create(ballot{
		TenantSlug: "jhw22",
		Title:      "Pro Kopf",
		Options:    []string{"Ja", "Nein"},
		Type:       ballotTypeMeeting,
		Weighting:  ballotWeightingPerHead,
		CreatedBy:  "manager@example.com",
	})
	if err != nil {
		t.Fatalf("Create head ballot: %v", err)
	}
	if _, _, err := a.voteStore.Open("jhw22", headBallot.ID, time.Now()); err != nil {
		t.Fatalf("Open head ballot: %v", err)
	}
	updated, found, err = a.castBallotVote("jhw22", "owner@example.com", headBallot.ID, "Nein", time.Now())
	if err != nil || !found || updated.Votes["owner@example.com"].Weight != 1 {
		t.Fatalf("owner cast head vote = %+v found=%v err=%v", updated.Votes["owner@example.com"], found, err)
	}
}

func TestBallotsPageOwnerVotingAndReadOnlyPersonas(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["renter@example.com"] = userProfile{Email: "renter@example.com", Role: roleRenter, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	a.profiles["beirat@example.com"] = userProfile{Email: "beirat@example.com", Role: roleBeirat, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	if err := a.unitStore.SetTenantUnits("jhw22", []unit{
		{ID: "top-1", TenantSlug: "jhw22", Label: "Top 1", MiteigentumsanteilPPM: 400000, OwnerEmails: []string{"owner@example.com"}, RenterEmails: []string{"renter@example.com"}},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	created, err := a.voteStore.Create(ballot{
		TenantSlug: "jhw22",
		Title:      "Dachsanierung",
		Options:    []string{"Ja", "Nein"},
		Type:       ballotTypeCircular,
		Weighting:  ballotWeightingPerShare,
		CreatedBy:  "manager@example.com",
	})
	if err != nil {
		t.Fatalf("Create ballot: %v", err)
	}
	if _, _, err := a.voteStore.Open("jhw22", created.ID, time.Now()); err != nil {
		t.Fatalf("Open ballot: %v", err)
	}

	ownerPage := authedRequest(t, a, "owner@example.com", "/app/abstimmungen", a.ballots)
	if ownerPage.Code != http.StatusOK {
		t.Fatalf("owner ballots status = %d", ownerPage.Code)
	}
	ownerBody := ownerPage.Body.String()
	for _, want := range []string{`href="/app/abstimmungen"`, "nav-item active", "Dachsanierung", `name="option"`, "Stimmgewicht: " + formatMiteigentumsanteil(400000)} {
		if !strings.Contains(ownerBody, want) {
			t.Fatalf("owner ballots page missing %q:\n%s", want, ownerBody)
		}
	}
	if strings.Contains(ownerBody, `class="nav-item disabled"`) && strings.Contains(ownerBody, "Abstimmungen") {
		t.Fatalf("Abstimmungen nav item must be a live link:\n%s", ownerBody)
	}

	vote := authedFormRequest(t, a, "owner@example.com", "/app/abstimmungen", url.Values{
		"ballot_id": {created.ID},
		"option":    {"Ja"},
	}, a.submitBallot)
	if vote.Code != http.StatusSeeOther {
		t.Fatalf("owner vote status = %d, want redirect", vote.Code)
	}
	stored, _ := a.voteStore.Get("jhw22", created.ID)
	if got := stored.Votes["owner@example.com"].Weight; got != 400000 {
		t.Fatalf("owner vote weight = %d, want 400000", got)
	}
	voteEvents := a.auditStore.List(auditFilter{TenantSlug: "jhw22", Action: auditActionVoteCast, Limit: 10})
	if len(voteEvents) != 1 || voteEvents[0].TargetID != created.ID || voteEvents[0].Details["weight"] == "" || voteEvents[0].Details["cast_at"] == "" {
		t.Fatalf("vote audit event should record timestamp/weight without option: %+v", voteEvents)
	}
	if _, ok := voteEvents[0].Details["option"]; ok {
		t.Fatalf("vote audit event should record timestamp/weight without option: %+v", voteEvents)
	}

	for _, persona := range []struct {
		email   string
		want    string
		handler http.HandlerFunc
	}{
		{"renter@example.com", "Nur Eigentümer können abstimmen.", a.submitBallot},
		{"beirat@example.com", "Beirat: lesende Übersicht.", a.submitBallot},
	} {
		page := authedRequest(t, a, persona.email, "/app/abstimmungen", a.ballots)
		if page.Code != http.StatusOK {
			t.Fatalf("%s ballots status = %d", persona.email, page.Code)
		}
		body := page.Body.String()
		if !strings.Contains(body, "Dachsanierung") || !strings.Contains(body, persona.want) {
			t.Fatalf("%s read-only page missing expected text:\n%s", persona.email, body)
		}
		if strings.Contains(body, `name="option"`) {
			t.Fatalf("%s must not see vote inputs:\n%s", persona.email, body)
		}
		post := authedFormRequest(t, a, persona.email, "/app/abstimmungen", url.Values{
			"ballot_id": {created.ID},
			"option":    {"Nein"},
		}, persona.handler)
		if post.Code != http.StatusForbidden {
			t.Fatalf("%s vote status = %d, want 403", persona.email, post.Code)
		}
	}

	board := authedRequest(t, a, "beirat@example.com", "/app/abstimmungen", a.ballots).Body.String()
	if !strings.Contains(board, "Teilnahme 100,0 %") || !strings.Contains(board, formatMiteigentumsanteil(400000)+" · 1 Stimmen") {
		t.Fatalf("beirat oversight should show weighted aggregate:\n%s", board)
	}
}

func TestBallotTallyQuorumAutoCloseAndProtocol(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "owner1@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["owner2@example.com"] = userProfile{Email: "owner2@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	a.profiles["beirat@example.com"] = userProfile{Email: "beirat@example.com", Role: roleBeirat, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	a.profiles["renter@example.com"] = userProfile{Email: "renter@example.com", Role: roleRenter, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	if err := a.unitStore.SetTenantUnits("jhw22", []unit{
		{ID: "top-1", TenantSlug: "jhw22", Label: "Top 1", MiteigentumsanteilPPM: 400000, OwnerEmails: []string{"owner1@example.com"}, RenterEmails: []string{"renter@example.com"}},
		{ID: "top-2", TenantSlug: "jhw22", Label: "Top 2", MiteigentumsanteilPPM: 600000, OwnerEmails: []string{"owner2@example.com"}},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	deadline := time.Now().Add(time.Hour)
	created, err := a.voteStore.Create(ballot{
		TenantSlug: "jhw22",
		Title:      "Fassade",
		Options:    []string{"Ja", "Nein"},
		Type:       ballotTypeCircular,
		Weighting:  ballotWeightingPerShare,
		QuorumPPM:  500000,
		ClosesAt:   deadline,
		CreatedBy:  "manager@example.com",
	})
	if err != nil {
		t.Fatalf("Create ballot: %v", err)
	}
	if _, _, err := a.voteStore.Open("jhw22", created.ID, time.Now()); err != nil {
		t.Fatalf("Open ballot: %v", err)
	}
	if _, _, err := a.castBallotVote("jhw22", "owner1@example.com", created.ID, "Ja", time.Now()); err != nil {
		t.Fatalf("owner1 vote: %v", err)
	}
	item, _ := a.voteStore.Get("jhw22", created.ID)
	view := a.ballotViewForActor("jhw22", "beirat@example.com", roleBeirat, item, time.Now(), true)
	if view.TotalWeightLabel != formatMiteigentumsanteil(400000) || view.EligibleWeightLabel != formatMiteigentumsanteil(1000000) || view.Participation != "40,0 %" || view.QuorumStatus != "Quorum offen" || view.WinnerLabel != "Ja" {
		t.Fatalf("single-vote tally = %+v", view)
	}
	if _, _, err := a.castBallotVote("jhw22", "owner2@example.com", created.ID, "Nein", time.Now()); err != nil {
		t.Fatalf("owner2 vote: %v", err)
	}
	item, _ = a.voteStore.Get("jhw22", created.ID)
	view = a.ballotViewForActor("jhw22", "beirat@example.com", roleBeirat, item, time.Now(), true)
	if view.TotalWeightLabel != formatMiteigentumsanteil(1000000) || view.Participation != "100,0 %" || view.QuorumStatus != "Quorum erreicht" || view.WinnerLabel != "Nein" {
		t.Fatalf("full tally = %+v", view)
	}

	openProtocol := authedPathValueRequest(t, a, "owner1@example.com", "/app/abstimmungen/"+created.ID+"/protokoll", map[string]string{"id": created.ID}, a.ballotProtocol)
	if openProtocol.Code != http.StatusConflict {
		t.Fatalf("open protocol status = %d, want 409", openProtocol.Code)
	}
	closed, err := a.voteStore.CloseExpiredTenant("jhw22", deadline.Add(time.Minute))
	if err != nil || len(closed) != 1 || closed[0].Status != ballotStatusClosed {
		t.Fatalf("CloseExpiredTenant = %+v err=%v", closed, err)
	}
	protocol := authedPathValueRequest(t, a, "owner1@example.com", "/app/abstimmungen/"+created.ID+"/protokoll", map[string]string{"id": created.ID}, a.ballotProtocol)
	if protocol.Code != http.StatusOK {
		t.Fatalf("protocol status = %d", protocol.Code)
	}
	if got := protocol.Header().Get("Content-Disposition"); !strings.Contains(got, "attachment") || !strings.Contains(got, "protokoll") {
		t.Fatalf("protocol content disposition = %q", got)
	}
	body := protocol.Body.String()
	for _, want := range []string{"Abstimmungsprotokoll", "Fassade", "Quorum erreicht", "Nein", "100,0 %"} {
		if !strings.Contains(body, want) {
			t.Fatalf("protocol missing %q:\n%s", want, body)
		}
	}
	renterProtocol := authedPathValueRequest(t, a, "renter@example.com", "/app/abstimmungen/"+created.ID+"/protokoll", map[string]string{"id": created.ID}, a.ballotProtocol)
	if renterProtocol.Code != http.StatusForbidden {
		t.Fatalf("renter protocol status = %d, want 403", renterProtocol.Code)
	}
}

func TestBallotVoteAfterDeadlineAutoCloses(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if err := a.unitStore.SetTenantUnits("jhw22", []unit{
		{ID: "top-1", TenantSlug: "jhw22", Label: "Top 1", MiteigentumsanteilPPM: 1000000, OwnerEmails: []string{"owner@example.com"}},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	now := time.Now()
	created, err := a.voteStore.Create(ballot{
		TenantSlug: "jhw22",
		Title:      "Deadline",
		Options:    []string{"Ja", "Nein"},
		Type:       ballotTypeCircular,
		Weighting:  ballotWeightingPerShare,
		OpensAt:    now.Add(-2 * time.Hour),
		ClosesAt:   now.Add(-time.Hour),
		CreatedBy:  "manager@example.com",
	})
	if err != nil {
		t.Fatalf("Create ballot: %v", err)
	}
	if _, _, err := a.voteStore.Open("jhw22", created.ID, now.Add(-2*time.Hour)); err != nil {
		t.Fatalf("Open ballot: %v", err)
	}
	if _, _, err := a.castBallotVote("jhw22", "owner@example.com", created.ID, "Ja", now); err == nil {
		t.Fatal("vote after deadline should fail")
	}
	closed, _ := a.voteStore.Get("jhw22", created.ID)
	if closed.Status != ballotStatusClosed {
		t.Fatalf("deadline vote should auto-close ballot: %+v", closed)
	}
	page := authedRequest(t, a, "owner@example.com", "/app/abstimmungen", a.ballots).Body.String()
	if strings.Contains(page, `name="option"`) || !strings.Contains(page, "Abstimmung geschlossen.") {
		t.Fatalf("closed ballot should render read-only:\n%s", page)
	}
}

func TestBallotReminderEmailsOnlyNonVotersAndHonorsPrefs(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["owner1@example.com"] = userProfile{Email: "owner1@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	a.profiles["owner2@example.com"] = userProfile{Email: "owner2@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	a.profiles["owner3@example.com"] = userProfile{Email: "owner3@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	if err := a.unitStore.SetTenantUnits("jhw22", []unit{
		{ID: "top-1", TenantSlug: "jhw22", Label: "Top 1", MiteigentumsanteilPPM: 300000, OwnerEmails: []string{"owner1@example.com"}},
		{ID: "top-2", TenantSlug: "jhw22", Label: "Top 2", MiteigentumsanteilPPM: 300000, OwnerEmails: []string{"owner2@example.com"}},
		{ID: "top-3", TenantSlug: "jhw22", Label: "Top 3", MiteigentumsanteilPPM: 400000, OwnerEmails: []string{"owner3@example.com"}},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	mailer := &recordingMailer{}
	a.mailer = mailer
	if err := a.notificationPrefs.Set("owner3@example.com", notificationPreferences{Email: map[string]bool{notificationEventVote: false}}); err != nil {
		t.Fatalf("Set owner3 prefs: %v", err)
	}
	now := time.Now()
	created, err := a.voteStore.Create(ballot{
		TenantSlug:            "jhw22",
		Title:                 "Reminder",
		Options:               []string{"Ja", "Nein"},
		Type:                  ballotTypeCircular,
		Weighting:             ballotWeightingPerShare,
		ClosesAt:              now.Add(2 * time.Hour),
		CreatedBy:             "manager@example.com",
		ReminderBeforeMinutes: 180,
	})
	if err != nil {
		t.Fatalf("Create ballot: %v", err)
	}
	if _, _, err := a.voteStore.Open("jhw22", created.ID, now); err != nil {
		t.Fatalf("Open ballot: %v", err)
	}
	if _, _, err := a.castBallotVote("jhw22", "owner1@example.com", created.ID, "Ja", now); err != nil {
		t.Fatalf("owner1 vote: %v", err)
	}

	if sent := a.sendDueBallotReminders(now.Add(30 * time.Minute)); sent != 1 {
		t.Fatalf("sendDueBallotReminders sent = %d, want 1", sent)
	}
	if len(mailer.notifications) != 1 || mailer.notifications[0].To != "owner2@example.com" || !strings.Contains(mailer.notifications[0].Subject, "Reminder") {
		t.Fatalf("reminder notifications = %+v", mailer.notifications)
	}
	updated, _ := a.voteStore.Get("jhw22", created.ID)
	if _, ok := updated.ReminderSentAt["owner2@example.com"]; !ok {
		t.Fatalf("owner2 should be marked reminded: %+v", updated.ReminderSentAt)
	}
	if _, ok := updated.ReminderSentAt["owner1@example.com"]; ok {
		t.Fatalf("owner1 voted and should not be marked reminded: %+v", updated.ReminderSentAt)
	}
	if _, ok := updated.ReminderSentAt["owner3@example.com"]; ok {
		t.Fatalf("reminder sent state = %+v", updated.ReminderSentAt)
	}
	if sent := a.sendDueBallotReminders(now.Add(time.Hour)); sent != 0 || len(mailer.notifications) != 1 {
		t.Fatalf("duplicate reminders sent=%d notifications=%+v", sent, mailer.notifications)
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "jhw22", Action: auditActionVoteReminder, Limit: 10})
	if len(events) != 1 || events[0].Details["recipients"] != "1" || events[0].Details["title"] != "Reminder" {
		t.Fatalf("reminder audit events = %+v", events)
	}
}

func TestDocumentStoreCreatePersistAndValidate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "documents.json")
	fileDir := filepath.Join(dir, "documents")
	store, err := newDocumentStore(path, fileDir)
	if err != nil {
		t.Fatalf("newDocumentStore: %v", err)
	}
	created, err := store.Create(documentRecord{
		TenantSlug: "JHW22",
		Title:      "Hausordnung",
		Category:   documentCategoryRules,
		Visibility: documentVisibilityAllResidents,
		UploadedBy: "Manager@Example.com",
	}, testMultipartHeader(t, "document", "../Hausordnung.pdf", []byte("%PDF-1.4\n% weg portal test\n")), time.Date(2026, 7, 7, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" || created.TenantSlug != "jhw22" || created.Filename != "Hausordnung.pdf" || created.StoredFilename == created.Filename {
		t.Fatalf("created document not normalized/private: %+v", created)
	}
	if created.ContentType != "application/pdf" || created.UploadedBy != "manager@example.com" || created.Size <= 0 {
		t.Fatalf("created document metadata = %+v", created)
	}
	storedPath, ok := store.FilePath(created)
	if !ok {
		t.Fatal("FilePath not available")
	}
	if info, err := os.Stat(storedPath); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("stored file mode = %v err=%v, want 0600", info.Mode().Perm(), err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("metadata file mode = %v err=%v, want 0600", info.Mode().Perm(), err)
	}
	reopened, err := newDocumentStore(path, fileDir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	docs := reopened.ListTenant("JHW22")
	if len(docs) != 1 || docs[0].Title != "Hausordnung" {
		t.Fatalf("reopened docs = %+v", docs)
	}
	if _, err := store.Create(documentRecord{
		TenantSlug: "jhw22",
		Title:      "Script",
		Category:   documentCategoryOther,
		Visibility: documentVisibilityAllResidents,
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "script.txt", []byte("<script>alert(1)</script>")), time.Now()); err == nil {
		t.Fatal("unsupported document type should be rejected")
	}
}

func TestDocumentStoreReplaceKeepsVersionHistory(t *testing.T) {
	dir := t.TempDir()
	store, err := newDocumentStore(filepath.Join(dir, "documents.json"), filepath.Join(dir, "documents"))
	if err != nil {
		t.Fatalf("newDocumentStore: %v", err)
	}
	created, err := store.Create(documentRecord{
		TenantSlug: "jhw22",
		Title:      "Hausordnung",
		Category:   documentCategoryRules,
		Visibility: documentVisibilityAllResidents,
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "hausordnung-v1.pdf", []byte("%PDF-1.4\nv1\n")), time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	replacement, replaced, err := store.Replace("jhw22", created.ID, "manager@example.com", testMultipartHeader(t, "document", "hausordnung-v2.pdf", []byte("%PDF-1.4\nv2\n")), time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}
	if !replacement.Current || replacement.Version != 2 || replacement.SeriesID != created.SeriesID || replacement.SupersedesID != created.ID {
		t.Fatalf("replacement metadata = %+v", replacement)
	}
	if replaced.Current || replaced.ReplacedByID != replacement.ID {
		t.Fatalf("replaced metadata = %+v", replaced)
	}
	current := store.ListCurrentTenant("jhw22")
	if len(current) != 1 || current[0].ID != replacement.ID {
		t.Fatalf("current docs = %+v", current)
	}
	versions := store.Versions("jhw22", created.SeriesID)
	if len(versions) != 2 || versions[0].Version != 2 || versions[1].Version != 1 {
		t.Fatalf("versions = %+v", versions)
	}
	if oldPath, ok := store.FilePath(replaced); !ok {
		t.Fatal("old version path missing")
	} else if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("old version file missing: %v", err)
	}
	if newPath, ok := store.FilePath(replacement); !ok {
		t.Fatal("new version path missing")
	} else if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("new version file missing: %v", err)
	}
}

func TestIssueStoreCreateListPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issues.json")
	store, err := newIssueStore(path, filepath.Join(t.TempDir(), "issue-attachments"))
	if err != nil {
		t.Fatalf("newIssueStore: %v", err)
	}
	created, err := store.Create(residentIssue{
		TenantSlug:     "JHW22",
		AuthorEmail:    "Resident@Example.com",
		AuthorName:     "Resident",
		Category:       "Reparatur",
		Title:          "Licht im Stiegenhaus",
		Body:           "Das Licht flackert.",
		LocationType:   issueLocationCommon,
		LocationDetail: "Stiege 1",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" || created.Status != issueStatusOpen || created.Priority != issuePriorityNorm {
		t.Fatalf("created issue defaults = %+v", created)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("issue store file mode = %v err=%v, want 0600", info.Mode().Perm(), err)
	}
	reopened, err := newIssueStore(path, "")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	byAuthor := reopened.ListAuthor("jhw22", "resident@example.com")
	if len(byAuthor) != 1 || byAuthor[0].Title != "Licht im Stiegenhaus" || byAuthor[0].TenantSlug != "jhw22" {
		t.Fatalf("ListAuthor = %+v", byAuthor)
	}
}

func TestDirectoryProfileEnvWinsAndInviteGrantsLogin(t *testing.T) {
	store, err := newInviteStore("")
	if err != nil {
		t.Fatalf("newInviteStore: %v", err)
	}
	// an invite trying to claim admin for an email that is an env resident
	if _, err := store.Add(userProfile{Email: "resident@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	// a brand-new invited-only user
	if _, err := store.Add(userProfile{Email: "invited@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	a := &app{
		defaultTenant: "jhw22",
		profiles: map[string]userProfile{
			"resident@example.com": {Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()},
		},
		inviteStore: store,
	}
	// env is authoritative: the invite must NOT escalate the env resident to admin
	if p, ok := a.directoryProfile("resident@example.com"); !ok || p.Role != roleResident {
		t.Fatalf("env must win: got role %q ok=%v", p.Role, ok)
	}
	// the invited-only user resolves from the store and may log into the tenant
	if p, ok := a.directoryProfile("invited@example.com"); !ok || p.Role != roleResident {
		t.Fatalf("invited user should resolve: got %+v ok=%v", p, ok)
	}
	if !a.isAllowed("invited@example.com", "jhw22") {
		t.Fatal("invited user should be allowed for jhw22")
	}
	// an unknown email is still denied
	if a.isAllowed("stranger@example.com", "jhw22") {
		t.Fatal("stranger must not be allowed")
	}
}

func TestInviteStoreUpdateRekeyAndDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invites.json")
	store, err := newInviteStore(path)
	if err != nil {
		t.Fatalf("newInviteStore: %v", err)
	}
	mustAdd := func(email, first string) {
		if _, err := store.Add(userProfile{Email: email, FirstName: first, Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}); err != nil {
			t.Fatalf("Add %s: %v", email, err)
		}
	}
	mustAdd("old@example.com", "Old")
	mustAdd("other@example.com", "Other")

	// updating a non-invite -> false, no error
	if ok, err := store.Update("ghost@example.com", userProfile{Email: "ghost@example.com"}); ok || err != nil {
		t.Fatalf("Update of non-invite: ok=%v err=%v (want false,nil)", ok, err)
	}

	// re-key old -> new + role change (case-insensitive key)
	ok, err := store.Update("old@example.com", userProfile{Email: "New@example.com", FirstName: "New", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if err != nil || !ok {
		t.Fatalf("Update rekey: ok=%v err=%v", ok, err)
	}
	if _, found := store.Get("old@example.com"); found {
		t.Fatal("old email should be gone after re-key")
	}
	got, found := store.Get("new@example.com")
	if !found || got.Role != roleAdmin || got.FirstName != "New" {
		t.Fatalf("re-keyed entry = %+v found=%v", got, found)
	}

	// re-keying onto an existing invite -> error, entry unchanged
	if ok, err := store.Update("new@example.com", userProfile{Email: "other@example.com"}); ok || err == nil {
		t.Fatalf("Update onto existing email: ok=%v err=%v (want false,err)", ok, err)
	}
	if _, found := store.Get("new@example.com"); !found {
		t.Fatal("entry must survive a rejected re-key")
	}

	// delete
	if ok, err := store.Delete("new@example.com"); !ok || err != nil {
		t.Fatalf("Delete: ok=%v err=%v", ok, err)
	}
	if _, found := store.Get("new@example.com"); found {
		t.Fatal("entry should be gone after delete")
	}
	if ok, _ := store.Delete("new@example.com"); ok {
		t.Fatal("second delete should report false")
	}
	if n := len(store.List()); n != 1 {
		t.Fatalf("List len = %d, want 1 (other@ remains)", n)
	}
}

func TestActivityStoreTouchGetPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "activity.json")
	store, err := newActivityStore(path)
	if err != nil {
		t.Fatalf("newActivityStore: %v", err)
	}
	if _, ok := store.Get("nobody@example.com"); ok {
		t.Fatal("empty store should have no records")
	}
	when := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	if err := store.Touch("Person@Example.com", when, authMethodEmail); err != nil {
		t.Fatalf("Touch: %v", err)
	}
	rec, ok := store.Get("person@example.com") // case-insensitive key
	if !ok || !rec.LastLogin.Equal(when) {
		t.Fatalf("Get = %+v ok=%v", rec, ok)
	}
	reopened, err := newActivityStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if _, ok := reopened.Get("person@example.com"); !ok {
		t.Fatal("activity did not persist across reopen")
	}
}

func TestUserRowsDeriveStatusFromActivity(t *testing.T) {
	act, _ := newActivityStore("")
	_ = act.Touch("loggedin@example.com", time.Date(2026, 7, 6, 9, 0, 0, 0, time.UTC), authMethodOIDC)
	inv, _ := newInviteStore("")
	_, _ = inv.Add(userProfile{Email: "loggedin@example.com", Role: roleResident, Status: "Eingeladen", Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	_, _ = inv.Add(userProfile{Email: "never@example.com", Role: roleResident, Status: "Eingeladen", Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a := &app{defaultTenant: "jhw22", profiles: map[string]userProfile{}, inviteStore: inv, activityStore: act}

	byEmail := map[string]userRow{}
	for _, r := range a.userRows("jhw22") {
		byEmail[r.Email] = r
	}
	if got := byEmail["loggedin@example.com"]; got.Status != "Aktiv" || !strings.Contains(got.LastSeen, "zuletzt angemeldet") {
		t.Fatalf("logged-in invite: status=%q lastseen=%q (want Aktiv + zuletzt)", got.Status, got.LastSeen)
	}
	if got := byEmail["never@example.com"]; got.Status != "Eingeladen" || got.LastSeen != "noch nie angemeldet" {
		t.Fatalf("never-logged-in invite: status=%q lastseen=%q", got.Status, got.LastSeen)
	}
}

func TestRunHealthcheckAcceptsExpectedPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service":"weg-portal","status":"ok"}`))
	}))
	defer server.Close()

	if err := runHealthcheck(server.URL); err != nil {
		t.Fatalf("healthcheck should accept expected payload: %v", err)
	}
}

func TestRunHealthcheckRejectsWrongPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service":"other","status":"ok"}`))
	}))
	defer server.Close()

	if err := runHealthcheck(server.URL); err == nil {
		t.Fatal("healthcheck should reject unexpected payload")
	}
}

func TestBuildLabelUsesSemverAndCommit(t *testing.T) {
	origVersion := appVersion
	origCommit := gitCommit
	t.Cleanup(func() {
		appVersion = origVersion
		gitCommit = origCommit
	})

	appVersion = "v0.1.0"
	gitCommit = "abc1234"

	if got := buildLabel(); got != "0.1.0 (abc1234)" {
		t.Fatalf("build label = %q, want semver and commit", got)
	}
}

func TestHomeUsesUnitCountFromStore(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if err := a.unitStore.SetTenantUnits("jhw22", []unit{
		{ID: "top-1", Label: "Top 1", OwnerEmails: []string{"owner1@example.com"}},
		{ID: "top-2", Label: "Top 2", OwnerEmails: []string{"owner2@example.com"}},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://jhw22.hausv.org/", nil)
	rr := httptest.NewRecorder()
	a.home(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("home status = %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "<strong>2</strong>") || !strings.Contains(body, "Wohneinheiten im Haus") {
		t.Fatalf("home should render real unit count, body: %s", body)
	}
	if strings.Contains(body, "12 Wohneinheiten") {
		t.Fatal("home must not render the old hardcoded unit count")
	}
}

func TestSessionSecretRequiredForPublicBaseURL(t *testing.T) {
	t.Setenv("SESSION_KEY", "")

	if _, err := sessionSecret(true); err == nil {
		t.Fatal("public deployment should require SESSION_KEY")
	}
}

func TestSessionSecretCanBeEphemeralForLocalDev(t *testing.T) {
	t.Setenv("SESSION_KEY", "")

	secret, err := sessionSecret(false)
	if err != nil {
		t.Fatalf("local development should allow generated session secret: %v", err)
	}
	if len(secret) != 32 {
		t.Fatalf("generated session secret length = %d, want 32", len(secret))
	}
}

func TestSignedSessionRoundTripSurvivesNewStore(t *testing.T) {
	secret := []byte(strings.Repeat("s", 32))
	first := newSessionStore(secret)

	token, _, err := first.Put("Markus@Barta.com", "JHW22", authMethodOIDC, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}

	second := newSessionStore(secret)
	email, tenantSlug, authMethod, ok := second.Get(token)
	if !ok {
		t.Fatal("session should verify in a new store with the same secret")
	}
	if email != "markus@barta.com" || tenantSlug != "jhw22" || authMethod != authMethodOIDC {
		t.Fatalf("unexpected session claims: %s %s %s", email, tenantSlug, authMethod)
	}

	other := newSessionStore([]byte(strings.Repeat("x", 32)))
	if _, _, _, ok := other.Get(token); ok {
		t.Fatal("session should not verify with a different secret")
	}
}

func TestParseUserProfilesNormalizesAuthMethods(t *testing.T) {
	raw := `[{"email":"joerg.lehner@gmx.at","first_name":"Jörg","last_name":"Lehner","tenants":["jhw22"],"auth_methods":["zitadel"]}]`

	profiles, err := parseUserProfiles(raw, map[string]struct{}{}, map[string]struct{}{}, "jhw22")
	if err != nil {
		t.Fatalf("parse profiles: %v", err)
	}
	profile := profiles["joerg.lehner@gmx.at"]
	if !profile.AllowsAuthMethod(authMethodOIDC) {
		t.Fatal("profile should allow OIDC")
	}
	if profile.AllowsAuthMethod(authMethodEmail) {
		t.Fatal("profile should not allow email login")
	}
	if got := profile.UserRow().AuthLabel; got != "Zitadel SSO" {
		t.Fatalf("auth label = %q", got)
	}
}

func TestParseUserProfilesNormalizesTenantMemberships(t *testing.T) {
	raw := `[{"email":"multi@example.com","role":"resident","permissions":["parking"],"tenant_memberships":{"JHW22":{"role":"Hausverwaltung"},"Haus-B":{"role":"Mieter","permissions":[]}}}]`

	profiles, err := parseUserProfiles(raw, map[string]struct{}{}, map[string]struct{}{}, "jhw22")
	if err != nil {
		t.Fatalf("parse profiles: %v", err)
	}
	profile := profiles["multi@example.com"]
	if !profile.HasTenant("jhw22") || !profile.HasTenant("haus-b") {
		t.Fatalf("tenant membership keys should grant tenant membership: %+v", profile)
	}

	jhwProfile := profile.ForTenant("jhw22")
	if jhwProfile.Role != roleManager || !jhwProfile.HasPermission(permissionParking) {
		t.Fatalf("jhw22 profile = %+v, want manager inheriting parking permission", jhwProfile)
	}

	otherProfile := profile.ForTenant("haus-b")
	if otherProfile.Role != roleRenter || otherProfile.HasPermission(permissionParking) {
		t.Fatalf("haus-b profile = %+v, want renter with parking explicitly cleared", otherProfile)
	}
}

func TestRoleForUsesTenantMembershipOverride(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "multi@example.com",
		FirstName:   "Multi",
		LastName:    "Tenant",
		Role:        roleResident,
		Tenants:     []string{"jhw22", "haus-b"},
		Permissions: []string{permissionParking},
		TenantMemberships: map[string]tenantMembership{
			"jhw22":  {Role: roleManager},
			"haus-b": {Role: roleResident, Permissions: []string{}},
		},
		AuthMethods: defaultAuthMethods(),
	})
	a.tenants["haus-b"] = tenantConfig{Slug: "haus-b", Name: "Haus B", Address: "Haus B", Host: "haus-b.hausv.org"}

	if got := a.roleFor("multi@example.com", "jhw22"); got != roleManager {
		t.Fatalf("roleFor jhw22 = %q, want %q", got, roleManager)
	}
	if got := a.roleFor("multi@example.com", "haus-b"); got != roleResident {
		t.Fatalf("roleFor haus-b = %q, want %q", got, roleResident)
	}

	jhwProfile := a.profileForTenant("multi@example.com", "jhw22")
	if jhwProfile.Role != roleManager || !jhwProfile.HasPermission(permissionParking) {
		t.Fatalf("jhw22 profile = %+v, want manager with parking", jhwProfile)
	}
	otherProfile := a.profileForTenant("multi@example.com", "haus-b")
	if otherProfile.Role != roleResident || otherProfile.HasPermission(permissionParking) {
		t.Fatalf("haus-b profile = %+v, want resident without parking", otherProfile)
	}

	token, _, err := a.sessions.Put("multi@example.com", "jhw22", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://jhw22.hausv.org/app", nil)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	_, role, tenantSlug, ok := a.currentUser(req)
	if !ok || tenantSlug != "jhw22" || role != roleManager {
		t.Fatalf("currentUser ok=%v tenant=%q role=%q, want jhw22 manager", ok, tenantSlug, role)
	}

	jhwRow := userRowForEmail(t, a.userRows("jhw22"), "multi@example.com")
	if jhwRow.Role != roleManager || !jhwRow.ParkingChecked {
		t.Fatalf("jhw22 row = %+v, want manager with parking checked", jhwRow)
	}
	otherRow := userRowForEmail(t, a.userRows("haus-b"), "multi@example.com")
	if otherRow.Role != roleResident || otherRow.ParkingChecked {
		t.Fatalf("haus-b row = %+v, want resident without parking", otherRow)
	}
}

func TestRoleForBackwardsCompatibleDefault(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"jhw22"},
		Permissions: []string{permissionParking},
		AuthMethods: defaultAuthMethods(),
	})

	if got := a.roleFor("owner@example.com", "jhw22"); got != roleOwner {
		t.Fatalf("roleFor default tenant = %q, want %q", got, roleOwner)
	}
	profile := a.profileForTenant("owner@example.com", "jhw22")
	if profile.Role != roleOwner || !profile.HasPermission(permissionParking) {
		t.Fatalf("profileForTenant = %+v, want owner retaining global parking permission", profile)
	}
	defaultProfile := a.profileFor("owner@example.com")
	if defaultProfile.Role != roleOwner || !defaultProfile.HasPermission(permissionParking) {
		t.Fatalf("profileFor default = %+v, want backwards-compatible global role/permission", defaultProfile)
	}
}

func TestOIDCLoginDefersUnavailableDiscovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	login, err := newOIDCLogin(ctx, "https://auth.invalid.example", "client-id", "", "", "Zitadel")
	if err != nil {
		t.Fatalf("new OIDC login should not fail hard when discovery is unavailable: %v", err)
	}
	if !login.Configured() {
		t.Fatal("OIDC should remain configured so discovery can be retried later")
	}
	if login.provider != nil || login.verifier != nil {
		t.Fatal("provider should not be initialized after canceled discovery")
	}
	if err := login.EnsureProvider(ctx); err == nil {
		t.Fatal("retry with canceled context should still report discovery failure")
	}
}

func TestNormalizeRoleAliasesAndCapabilityMatrix(t *testing.T) {
	aliases := map[string]string{
		"admin":            roleAdmin,
		"property-manager": roleManager,
		"Hausverwaltung":   roleManager,
		"Eigentuemer":      roleOwner,
		"eigentümer":       roleOwner,
		"owner":            roleOwner,
		"tenant":           roleRenter,
		"Mieter":           roleRenter,
		"advisory-board":   roleBeirat,
		"Beirat":           roleBeirat,
		"resident":         roleResident,
		"Bewohner":         roleResident,
	}
	for raw, want := range aliases {
		if got := normalizeRole(raw); got != want {
			t.Fatalf("normalizeRole(%q) = %q, want %q", raw, got, want)
		}
	}

	cases := []struct {
		role string
		cap  capability
		want bool
	}{
		{roleAdmin, capabilityManageUsers, true},
		{roleAdmin, capabilityManageParking, true},
		{roleManager, capabilityManageAnnouncements, true},
		{roleManager, capabilityManageUsers, true},
		{roleManager, capabilityManageParking, false},
		{roleManager, capabilityPlatformAdmin, false},
		{roleOwner, capabilityVote, true},
		{roleOwner, capabilityOwnerDocuments, true},
		{roleRenter, capabilityVote, false},
		{roleBeirat, capabilityOversight, true},
		{roleBeirat, capabilityManageAnnouncements, false},
		{roleResident, capabilityOversight, false},
	}
	for _, tc := range cases {
		if got := hasCapability(tc.role, tc.cap); got != tc.want {
			t.Fatalf("hasCapability(%q, %q) = %v, want %v", tc.role, tc.cap, got, tc.want)
		}
	}
}

func TestSettingsHubVisibleToResidentWithoutAdminSections(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})

	rr := authedRequest(t, a, "resident@example.com", "/app/settings", a.settingsHub)
	if rr.Code != http.StatusOK {
		t.Fatalf("settings hub status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{`href="/app/settings"`, "Profil", "Benachrichtigungen"} {
		if !strings.Contains(body, want) {
			t.Fatalf("settings hub should contain %q", want)
		}
	}
	if strings.Contains(body, `href="/app/parking/settings"`) {
		t.Fatal("resident settings hub must not expose parking settings")
	}
	if strings.Contains(body, `href="/app/settings/parking-access"`) {
		t.Fatal("resident settings hub must not expose parking access management")
	}
}

func TestNotificationSettingsPersistAndRender(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})

	save := authedFormRequest(t, a, "resident@example.com", "/app/settings/notifications", url.Values{
		"email_enabled": {"on"},
		"events":        {notificationEventAnnouncement, notificationEventDocument},
	}, a.updateNotificationSettings)
	if save.Code != http.StatusSeeOther {
		t.Fatalf("notification settings save status = %d", save.Code)
	}
	prefs := a.notificationPrefs.Get("resident@example.com")
	if prefs.Unsubscribed || !prefs.Email[notificationEventAnnouncement] || !prefs.Email[notificationEventDocument] || prefs.Email[notificationEventIssue] {
		t.Fatalf("saved notification prefs = %+v", prefs)
	}

	page := authedRequest(t, a, "resident@example.com", "/app/settings/notifications", a.notificationSettings)
	if page.Code != http.StatusOK {
		t.Fatalf("notification settings status = %d", page.Code)
	}
	body := page.Body.String()
	if !strings.Contains(body, `href="/app/settings"`) || !strings.Contains(body, `value="announcement" checked`) || strings.Contains(body, `value="issue" checked`) {
		t.Fatalf("notification settings page did not reflect saved prefs:\n%s", body)
	}
}

func TestProfileOverlayStorePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.json")
	store, err := newProfileOverlayStore(path)
	if err != nil {
		t.Fatalf("newProfileOverlayStore: %v", err)
	}
	if err := store.Set("Resident@Example.com", profileOverlay{Title: "Dr.", FirstName: "Resi", LastName: "Dent", Phone: "+43 1 234"}); err != nil {
		t.Fatalf("set overlay: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat tenant override store: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("store file mode = %v, want 0600", info.Mode().Perm())
	}
	reopened, err := newProfileOverlayStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	overlay, ok := reopened.Get("resident@example.com")
	if !ok || overlay.FirstName != "Resi" || overlay.Phone != "+43 1 234" {
		t.Fatalf("overlay = %+v ok=%v", overlay, ok)
	}
}

func TestTenantOverrideStoreLayersOverEnvDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tenants.json")
	store, err := newTenantOverrideStore(path)
	if err != nil {
		t.Fatalf("newTenantOverrideStore: %v", err)
	}
	if err := store.Set("JHW22", tenantOverride{
		Name:         "WEG Sonneneck",
		Address:      "Neue Gasse 7",
		ContactName:  "Hausverwaltung Nord",
		ContactEmail: "Office@Example.com",
		ContactPhone: "+43 1 999",
	}); err != nil {
		t.Fatalf("set tenant override: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat tenant override store: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("store file mode = %v, want 0600", info.Mode().Perm())
	}

	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.tenants["jhw22"] = tenantConfig{
		Slug:         "jhw22",
		Name:         "Env Name",
		Address:      "Env Address",
		ContactName:  "Env Contact",
		ContactEmail: "env@example.com",
		ContactPhone: "+43 1 111",
		HeroImageURL: "/assets/env.jpg",
		Host:         "jhw22.hausv.org",
	}
	a.tenantOverrides = store

	tenant, ok := a.tenantBySlug("jhw22")
	if !ok {
		t.Fatal("tenant not found")
	}
	if tenant.Name != "WEG Sonneneck" || tenant.Address != "Neue Gasse 7" || tenant.ContactEmail != "office@example.com" || tenant.HeroImageURL != "/assets/env.jpg" {
		t.Fatalf("tenant override = %+v", tenant)
	}
	if err := store.SetHeroImage("jhw22", "jhw22-hero.png"); err != nil {
		t.Fatalf("set hero image: %v", err)
	}
	tenant, _ = a.tenantBySlug("jhw22")
	if tenant.HeroImageURL != "/tenant-hero/jhw22" || tenant.ContactName != "Hausverwaltung Nord" {
		t.Fatalf("tenant after hero override = %+v", tenant)
	}
}

func TestProfileSettingsPersistOverlayWithoutAuthzEscalation(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if err := a.unitStore.SetTenantUnits("jhw22", []unit{{
		ID:                    "top-1",
		Label:                 "Top 1",
		MiteigentumsanteilPPM: 12345,
		RenterEmails:          []string{"resident@example.com"},
	}}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}

	save := authedFormRequest(t, a, "resident@example.com", "/app/settings/profile", url.Values{
		"title":            {"Dr."},
		"first_name":       {"Resi"},
		"last_name":        {"Dent"},
		"phone":            {"+43 1 234"},
		"directory_opt_in": {"on"},
		"role":             {"Admin"},
		"permissions":      {permissionParking},
	}, a.updateProfileSettings)
	if save.Code != http.StatusSeeOther {
		t.Fatalf("profile save status = %d, want redirect", save.Code)
	}
	profile := a.profileForTenant("resident@example.com", "jhw22")
	if profile.DisplayName() != "Dr. Resi Dent" || profile.Phone != "+43 1 234" {
		t.Fatalf("profile overlay not applied: %+v", profile)
	}
	if profile.Role != roleResident || profile.HasPermission(permissionParking) {
		t.Fatalf("profile self-edit escalated authz: %+v", profile)
	}
	if !profile.DirectoryOptIn {
		t.Fatalf("profile directory opt-in not applied: %+v", profile)
	}
	if parking := authedRequest(t, a, "resident@example.com", "/app/parking", a.parking); parking.Code != http.StatusNotFound {
		t.Fatalf("parking status = %d, want 404 without parking permission", parking.Code)
	}

	settings := authedRequest(t, a, "resident@example.com", "/app/settings", a.settingsHub)
	if !strings.Contains(settings.Body.String(), `href="/app/settings/profile"`) {
		t.Fatalf("settings hub should link to profile:\n%s", settings.Body.String())
	}
	page := authedRequest(t, a, "resident@example.com", "/app/settings/profile", a.profileSettings)
	if page.Code != http.StatusOK {
		t.Fatalf("profile page status = %d", page.Code)
	}
	body := page.Body.String()
	for _, want := range []string{`value="Dr."`, `value="Resi"`, `value="Dent"`, "43 1 234", `name="directory_opt_in" checked`, roleResident, "Top 1", "12.345 / 1.000.000"} {
		if !strings.Contains(body, want) {
			t.Fatalf("profile page should contain %q", want)
		}
	}
	row := userRowForEmail(t, a.userRows("jhw22"), "resident@example.com")
	if row.DisplayName != "Dr. Resi Dent" || row.Phone != "+43 1 234" || !row.DirectoryOptIn || row.Role != roleResident || row.ParkingChecked {
		t.Fatalf("roster row = %+v, want overlay display without authz escalation", row)
	}
}

func TestContactsPageShowsBuildingBoardAndOptInDirectory(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", FirstName: "Resi", LastName: "Dent", Phone: "+43 1 234", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	a.profiles["private@example.com"] = userProfile{Email: "private@example.com", FirstName: "Privat", LastName: "Person", Phone: "+43 1 555", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	a.profiles["board@example.com"] = userProfile{Email: "board@example.com", FirstName: "Berta", LastName: "Beirat", Phone: "+43 1 777", Role: roleBeirat, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}

	saveMeta := authedFormRequest(t, a, "manager@example.com", "/app/settings/building", url.Values{
		"name":            {"WEG Portal"},
		"address":         {"Janischhofweg 22"},
		"contact_name":    {"Hausverwaltung Nord"},
		"contact_email":   {"office@example.com"},
		"contact_phone":   {"+43 1 999"},
		"emergency_name":  {"Notdienst 24"},
		"emergency_phone": {"144"},
		"caretaker_name":  {"Hausmeister Max"},
		"caretaker_email": {"hausmeister@example.com"},
		"caretaker_phone": {"+43 1 888"},
	}, a.updateBuildingSettings)
	if saveMeta.Code != http.StatusSeeOther {
		t.Fatalf("building settings save status = %d", saveMeta.Code)
	}

	page := authedRequest(t, a, "resident@example.com", "/app/kontakte", a.contacts)
	if page.Code != http.StatusOK {
		t.Fatalf("contacts status = %d", page.Code)
	}
	body := page.Body.String()
	for _, want := range []string{`href="/app/kontakte"`, "nav-item active", "Hausverwaltung Nord", "office@example.com", "Notdienst 24", "144", "Hausmeister Max", "hausmeister@example.com", "Berta Beirat", "board@example.com", "43 1 777"} {
		if !strings.Contains(body, want) {
			t.Fatalf("contacts page should contain %q", want)
		}
	}
	for _, forbidden := range []string{"resident@example.com", "43 1 234", "private@example.com", "43 1 555"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("contacts page exposed non-opt-in contact %q", forbidden)
		}
	}

	optIn := authedFormRequest(t, a, "resident@example.com", "/app/settings/profile", url.Values{
		"first_name":       {"Resi"},
		"last_name":        {"Dent"},
		"phone":            {"+43 1 234"},
		"directory_opt_in": {"on"},
	}, a.updateProfileSettings)
	if optIn.Code != http.StatusSeeOther {
		t.Fatalf("profile opt-in status = %d", optIn.Code)
	}
	page = authedRequest(t, a, "resident@example.com", "/app/kontakte", a.contacts)
	body = page.Body.String()
	for _, want := range []string{"Resi Dent", "resident@example.com", "43 1 234"} {
		if !strings.Contains(body, want) {
			t.Fatalf("contacts page after opt-in should contain %q", want)
		}
	}
	if strings.Contains(body, "private@example.com") || strings.Contains(body, "43 1 555") {
		t.Fatal("contacts page must keep non-opt-in residents hidden")
	}
}

func TestBuildingSettingsManagerUpdatesMetaHeroAndUnits(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}

	resident := authedRequest(t, a, "resident@example.com", "/app/settings/building", a.buildingSettings)
	if resident.Code != http.StatusForbidden {
		t.Fatalf("resident building settings status = %d, want 403", resident.Code)
	}
	hub := authedRequest(t, a, "manager@example.com", "/app/settings", a.settingsHub)
	if !strings.Contains(hub.Body.String(), `href="/app/settings/building"`) {
		t.Fatalf("manager settings hub should link building settings:\n%s", hub.Body.String())
	}

	saveMeta := authedFormRequest(t, a, "manager@example.com", "/app/settings/building", url.Values{
		"name":            {"WEG Sonneneck"},
		"address":         {"Neue Gasse 7"},
		"contact_name":    {"Hausverwaltung Nord"},
		"contact_email":   {"office@example.com"},
		"contact_phone":   {"+43 1 999"},
		"emergency_name":  {"Notdienst 24"},
		"emergency_phone": {"144"},
		"caretaker_name":  {"Hausmeister Max"},
		"caretaker_email": {"hausmeister@example.com"},
		"caretaker_phone": {"+43 1 888"},
	}, a.updateBuildingSettings)
	if saveMeta.Code != http.StatusSeeOther {
		t.Fatalf("building meta save status = %d", saveMeta.Code)
	}
	tenant, _ := a.tenantBySlug("jhw22")
	if tenant.Name != "WEG Sonneneck" || tenant.Address != "Neue Gasse 7" || tenant.ContactName != "Hausverwaltung Nord" || tenant.ContactEmail != "office@example.com" || tenant.ContactPhone != "+43 1 999" || tenant.EmergencyName != "Notdienst 24" || tenant.EmergencyPhone != "144" || tenant.CaretakerName != "Hausmeister Max" || tenant.CaretakerEmail != "hausmeister@example.com" || tenant.CaretakerPhone != "+43 1 888" {
		t.Fatalf("tenant after meta save = %+v", tenant)
	}
	homeReq := httptest.NewRequest(http.MethodGet, "http://jhw22.hausv.org/", nil)
	home := httptest.NewRecorder()
	a.home(home, homeReq)
	if !strings.Contains(home.Body.String(), "WEG Sonneneck") || !strings.Contains(home.Body.String(), "Neue Gasse 7") || !strings.Contains(home.Body.String(), defaultTenantHeroImageURL) {
		t.Fatalf("home should render layered name, address and default hero:\n%s", home.Body.String())
	}

	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89,
	}
	uploadHero := authedMultipartFileRequest(t, a, "manager@example.com", "/app/settings/building/hero", nil, "hero_image", "hero.png", png, a.updateBuildingHero)
	if uploadHero.Code != http.StatusSeeOther {
		t.Fatalf("hero upload status = %d", uploadHero.Code)
	}
	tenant, _ = a.tenantBySlug("jhw22")
	if tenant.HeroImageURL != "/tenant-hero/jhw22" {
		t.Fatalf("tenant hero url = %q", tenant.HeroImageURL)
	}
	heroPath := filepath.Join(a.tenantHeroDir, "jhw22-hero.png")
	heroInfo, err := os.Stat(heroPath)
	if err != nil {
		t.Fatalf("stat hero file: %v", err)
	}
	if heroInfo.Mode().Perm() != 0o600 {
		t.Fatalf("hero file mode = %v, want 0600", heroInfo.Mode().Perm())
	}
	heroReq := httptest.NewRequest(http.MethodGet, "http://jhw22.hausv.org/tenant-hero/jhw22", nil)
	heroReq.SetPathValue("tenant", "jhw22")
	hero := httptest.NewRecorder()
	a.tenantHeroImage(hero, heroReq)
	if hero.Code != http.StatusOK || !strings.HasPrefix(hero.Header().Get("Content-Type"), "image/png") {
		t.Fatalf("hero response status=%d content-type=%q", hero.Code, hero.Header().Get("Content-Type"))
	}
	homeAfterHeroReq := httptest.NewRequest(http.MethodGet, "http://jhw22.hausv.org/", nil)
	homeAfterHero := httptest.NewRecorder()
	a.home(homeAfterHero, homeAfterHeroReq)
	if !strings.Contains(homeAfterHero.Body.String(), "/tenant-hero/jhw22") {
		t.Fatalf("home should render uploaded hero path:\n%s", homeAfterHero.Body.String())
	}
	portal := authedRequest(t, a, "manager@example.com", "/app", a.portal)
	if !strings.Contains(portal.Body.String(), "/tenant-hero/jhw22") {
		t.Fatalf("portal should render uploaded hero path:\n%s", portal.Body.String())
	}

	addUnit := authedFormRequest(t, a, "manager@example.com", "/app/settings/building/units", url.Values{
		"label":              {"Top 1"},
		"miteigentumsanteil": {"12345"},
		"owner_emails":       {"owner@example.com; second@example.com"},
		"renter_emails":      {"resident@example.com"},
	}, a.upsertBuildingUnit)
	if addUnit.Code != http.StatusSeeOther {
		t.Fatalf("unit add status = %d", addUnit.Code)
	}
	units := a.unitStore.ListTenant("jhw22")
	if len(units) != 1 || units[0].ID != "top-1" || units[0].MiteigentumsanteilPPM != 12345 || len(units[0].OwnerEmails) != 2 || units[0].RenterEmails[0] != "resident@example.com" {
		t.Fatalf("units after add = %+v", units)
	}
	page := authedRequest(t, a, "manager@example.com", "/app/settings/building", a.buildingSettings)
	for _, want := range []string{"WEG Sonneneck", "Neue Gasse 7", "Top 1", "12.345 / 1.000.000", `value="owner@example.com, second@example.com"`} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("building page should contain %q", want)
		}
	}

	editUnit := authedFormRequest(t, a, "manager@example.com", "/app/settings/building/units", url.Values{
		"orig_id":            {"top-1"},
		"id":                 {"top-1"},
		"label":              {"Top 1A"},
		"miteigentumsanteil": {"23456"},
		"owner_emails":       {"owner@example.com"},
		"renter_emails":      {""},
	}, a.upsertBuildingUnit)
	if editUnit.Code != http.StatusSeeOther {
		t.Fatalf("unit edit status = %d", editUnit.Code)
	}
	units = a.unitStore.ListTenant("jhw22")
	if len(units) != 1 || units[0].Label != "Top 1A" || units[0].MiteigentumsanteilPPM != 23456 || len(units[0].RenterEmails) != 0 {
		t.Fatalf("units after edit = %+v", units)
	}

	deleteUnit := authedFormRequest(t, a, "manager@example.com", "/app/settings/building/units/delete", url.Values{"id": {"top-1"}}, a.deleteBuildingUnit)
	if deleteUnit.Code != http.StatusSeeOther {
		t.Fatalf("unit delete status = %d", deleteUnit.Code)
	}
	if units := a.unitStore.ListTenant("jhw22"); len(units) != 0 {
		t.Fatalf("units after delete = %+v", units)
	}
}

func TestSettingsHubAdminLinksManagementSections(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})

	rr := authedRequest(t, a, "admin@example.com", "/app/settings", a.settingsHub)
	if rr.Code != http.StatusOK {
		t.Fatalf("settings hub status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{`href="/app/settings/building"`, `href="/app/settings/users"`, `href="/app/settings/parking-access"`, `href="/app/parking/settings"`, "Parkplatz-Abrechnung", "Parkplatz-Zugriff", "Gebäude"} {
		if !strings.Contains(body, want) {
			t.Fatalf("admin settings hub should contain %q", want)
		}
	}
}

func TestSettingsHubManagerLinksTenantManagementOnly(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})

	rr := authedRequest(t, a, "manager@example.com", "/app/settings", a.settingsHub)
	if rr.Code != http.StatusOK {
		t.Fatalf("settings hub status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{`href="/app/settings/building"`, `href="/app/settings/users"`, `href="/app/settings/parking-access"`, "Benutzer &amp; Rechte", "Parkplatz-Zugriff", "Gebäude"} {
		if !strings.Contains(body, want) {
			t.Fatalf("manager settings hub should contain %q", want)
		}
	}
	if strings.Contains(body, `href="/app/parking/settings"`) {
		t.Fatal("manager settings hub must not expose parking accounting config")
	}
}

func TestUserRowsDoesNotInjectSyntheticEmptyRow(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles = map[string]userProfile{}
	a.allowed = map[string]struct{}{}
	a.admins = map[string]struct{}{}

	rows := a.userRows("jhw22")
	if len(rows) != 0 {
		t.Fatalf("empty userRows = %+v, want no synthetic rows", rows)
	}
}

func TestRoleManagementUIOffersAllEffectiveRoles(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	for _, profile := range []userProfile{
		{Email: "owner@example.com", FirstName: "Eva", LastName: "Owner", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()},
		{Email: "renter@example.com", FirstName: "Max", LastName: "Renter", Role: roleRenter, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()},
		{Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()},
		{Email: "board@example.com", FirstName: "Berta", LastName: "Beirat", Role: roleBeirat, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()},
	} {
		a.profiles[profile.Email] = profile
	}

	rr := authedRequest(t, a, "admin@example.com", "/app/settings/users", a.userSettings)
	if rr.Code != http.StatusOK {
		t.Fatalf("user settings status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{
		`value="Mieter"`, `value="Eigentümer"`, `value="Beirat"`, `value="Verwalter"`, `value="Admin"`,
		"role-owner", "role-renter", "role-manager", "role-beirat",
		"Eigentümer-Dokumente", "Abstimmungen", "Aushang verwalten", "Gebäude verwalten", "Leserechte",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("role management UI should contain %q", want)
		}
	}
}

func TestRoleManagementUIOffersPermissionCheckboxesAndPresets(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "parker@example.com", FirstName: "Pat", LastName: "Parker", Role: roleRenter, Tenants: []string{"jhw22"}, Permissions: []string{permissionParking}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("Add invite: %v", err)
	}

	rr := authedRequest(t, a, "admin@example.com", "/app/settings/users", a.userSettings)
	if rr.Code != http.StatusOK {
		t.Fatalf("user settings status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{
		`name="permissions" value="parking"`,
		`data-permission="parking"`,
		`data-preset-permissions="parking"`,
		"Parkplatznutzung",
		`checked><strong>Parkplatznutzung`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("permission UI should contain %q", want)
		}
	}
}

func TestInviteCreatedWithParkingPermissionGrantsParkingAccess(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	values := url.Values{
		"email":       {"parker@example.com"},
		"first_name":  {"Pat"},
		"last_name":   {"Parker"},
		"role":        {"Mieter"},
		"permissions": {permissionParking},
	}
	create := authedFormRequest(t, a, "admin@example.com", "/app/settings/users", values, a.createInvite)
	if create.Code != http.StatusSeeOther {
		t.Fatalf("create invite status = %d, want redirect", create.Code)
	}
	profile, ok := a.inviteStore.Get("parker@example.com")
	if !ok || !profile.HasPermission(permissionParking) {
		t.Fatalf("stored invite = %+v ok=%v, want parking permission", profile, ok)
	}
	parking := authedRequest(t, a, "parker@example.com", "/app/parking", a.parking)
	if parking.Code != http.StatusOK {
		t.Fatalf("parking status = %d, want 200 for invited parking user", parking.Code)
	}
}

func TestEditInviteCanRevokeParkingPermission(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "parker@example.com", FirstName: "Pat", LastName: "Parker", Role: roleRenter, Tenants: []string{"jhw22"}, Permissions: []string{permissionParking}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("Add invite: %v", err)
	}
	values := url.Values{
		"orig_email": {"parker@example.com"},
		"email":      {"parker@example.com"},
		"first_name": {"Pat"},
		"last_name":  {"Parker"},
		"role":       {"Mieter"},
	}
	edit := authedFormRequest(t, a, "admin@example.com", "/app/settings/users/edit", values, a.editInvite)
	if edit.Code != http.StatusSeeOther {
		t.Fatalf("edit invite status = %d, want redirect", edit.Code)
	}
	profile, ok := a.inviteStore.Get("parker@example.com")
	if !ok || profile.HasPermission(permissionParking) {
		t.Fatalf("stored invite = %+v ok=%v, want parking revoked", profile, ok)
	}
	parking := authedRequest(t, a, "parker@example.com", "/app/parking", a.parking)
	if parking.Code != http.StatusNotFound {
		t.Fatalf("parking status = %d, want 404 after parking revoke", parking.Code)
	}
}

func TestParkingAccessPageGrantsAndRevokesInvitePermission(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "parker@example.com", FirstName: "Pat", LastName: "Parker", Role: roleRenter, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("Add invite: %v", err)
	}

	page := authedRequest(t, a, "manager@example.com", "/app/settings/parking-access", a.parkingAccessSettings)
	if page.Code != http.StatusOK {
		t.Fatalf("parking access status = %d, want 200", page.Code)
	}
	body := page.Body.String()
	for _, want := range []string{"Parkplatz-Zugriff", "parker@example.com", "Freigeben", "Portal"} {
		if !strings.Contains(body, want) {
			t.Fatalf("parking access page missing %q:\n%s", want, body)
		}
	}

	grant := authedFormRequest(t, a, "manager@example.com", "/app/settings/parking-access", url.Values{
		"email":   {"parker@example.com"},
		"parking": {"1"},
	}, a.updateParkingAccess)
	if grant.Code != http.StatusSeeOther {
		t.Fatalf("grant status = %d, want redirect", grant.Code)
	}
	profile, ok := a.inviteStore.Get("parker@example.com")
	if !ok || !profile.HasPermission(permissionParking) {
		t.Fatalf("stored profile after grant = %+v ok=%v", profile, ok)
	}
	if parking := authedRequest(t, a, "parker@example.com", "/app/parking", a.parking); parking.Code != http.StatusOK {
		t.Fatalf("parking status after grant = %d, want 200", parking.Code)
	}

	revoke := authedFormRequest(t, a, "manager@example.com", "/app/settings/parking-access", url.Values{
		"email":   {"parker@example.com"},
		"parking": {"0"},
	}, a.updateParkingAccess)
	if revoke.Code != http.StatusSeeOther {
		t.Fatalf("revoke status = %d, want redirect", revoke.Code)
	}
	profile, ok = a.inviteStore.Get("parker@example.com")
	if !ok || profile.HasPermission(permissionParking) {
		t.Fatalf("stored profile after revoke = %+v ok=%v", profile, ok)
	}
	if parking := authedRequest(t, a, "parker@example.com", "/app/parking", a.parking); parking.Code != http.StatusNotFound {
		t.Fatalf("parking status after revoke = %d, want 404", parking.Code)
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "jhw22", Action: auditActionInviteUpdate, Query: "parker", Limit: 10})
	if len(events) != 2 || events[0].Summary != "Parkplatz-Zugriff geändert" {
		t.Fatalf("parking access audit events = %+v", events)
	}
}

func TestParkingAccessPageKeepsEnvUsersReadOnly(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["env-parker@example.com"] = userProfile{Email: "env-parker@example.com", FirstName: "Env", LastName: "Parker", Role: roleRenter, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}

	page := authedRequest(t, a, "manager@example.com", "/app/settings/parking-access", a.parkingAccessSettings)
	if page.Code != http.StatusOK {
		t.Fatalf("parking access status = %d, want 200", page.Code)
	}
	body := page.Body.String()
	if !strings.Contains(body, "env-parker@example.com") || !strings.Contains(body, "Konfiguration") || !strings.Contains(body, "Schreibgeschützt") {
		t.Fatalf("env user should render read-only:\n%s", body)
	}

	grant := authedFormRequest(t, a, "manager@example.com", "/app/settings/parking-access", url.Values{
		"email":   {"env-parker@example.com"},
		"parking": {"1"},
	}, a.updateParkingAccess)
	if grant.Code != http.StatusSeeOther || !strings.Contains(grant.Header().Get("Location"), "not_editable") {
		t.Fatalf("env grant redirect = %d %q", grant.Code, grant.Header().Get("Location"))
	}
	if a.profiles["env-parker@example.com"].HasPermission(permissionParking) {
		t.Fatal("env profile must not be mutated by parking access page")
	}
	if parking := authedRequest(t, a, "env-parker@example.com", "/app/parking", a.parking); parking.Code != http.StatusNotFound {
		t.Fatalf("env parking status = %d, want 404", parking.Code)
	}
}

func TestNavigationActionsStayScopedToRelevantPages(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})

	portal := authedRequest(t, a, "admin@example.com", "/app", a.portal)
	if portal.Code != http.StatusOK {
		t.Fatalf("portal status = %d, want 200", portal.Code)
	}
	if strings.Contains(portal.Body.String(), "Aushang verwalten") {
		t.Fatal("portal should not duplicate the Aushang destination with an admin-only quick link")
	}

	access := authedRequest(t, a, "admin@example.com", "/app/settings/parking-access", a.parkingAccessSettings)
	if access.Code != http.StatusOK {
		t.Fatalf("parking access status = %d, want 200", access.Code)
	}
	accessBody := access.Body.String()
	for _, want := range []string{`action="/app/parking/reminders"`, `name="return_to" value="parking_access"`} {
		if !strings.Contains(accessBody, want) {
			t.Fatalf("parking access page missing %q", want)
		}
	}

	for _, tc := range []struct {
		name string
		path string
		fn   http.HandlerFunc
	}{
		{name: "building", path: "/app/settings/building", fn: a.buildingSettings},
		{name: "profile", path: "/app/settings/profile", fn: a.profileSettings},
	} {
		rr := authedRequest(t, a, "admin@example.com", tc.path, tc.fn)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", tc.name, rr.Code)
		}
		if strings.Contains(rr.Body.String(), `action="/app/parking/reminders"`) {
			t.Fatalf("%s page should not expose parking reminders", tc.name)
		}
	}
}

func TestAuditStoreAppendListFilterAndSanitize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	store, err := newAuditStore(path)
	if err != nil {
		t.Fatalf("newAuditStore: %v", err)
	}
	if err := store.Append(auditEvent{
		TenantSlug: "jhw22",
		ActorEmail: "Admin@Example.com",
		ActorRole:  roleAdmin,
		Action:     auditActionInviteCreate,
		TargetType: "user",
		TargetID:   "resident@example.com",
		Summary:    "Einladung gespeichert",
		Details: map[string]string{
			"role_to":     roleResident,
			"secret_note": "must-not-persist",
		},
	}); err != nil {
		t.Fatalf("append invite audit: %v", err)
	}
	if err := store.Append(auditEvent{
		TenantSlug: "other",
		ActorEmail: "admin@example.com",
		Action:     auditActionLogin,
		TargetType: "session",
		TargetID:   "admin@example.com",
		Summary:    "Anmeldung erfolgreich",
	}); err != nil {
		t.Fatalf("append login audit: %v", err)
	}
	reopened, err := newAuditStore(path)
	if err != nil {
		t.Fatalf("reopen audit store: %v", err)
	}
	events := reopened.List(auditFilter{TenantSlug: "jhw22", Action: auditActionInviteCreate, Query: "resident", Limit: 10})
	if len(events) != 1 {
		t.Fatalf("filtered audit events = %+v, want one", events)
	}
	if events[0].ActorEmail != "admin@example.com" || events[0].Details["secret_note"] != "" {
		t.Fatalf("audit event not normalized/sanitized: %+v", events[0])
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit file: %v", err)
	}
	if lines := strings.Count(strings.TrimSpace(string(raw)), "\n") + 1; lines != 2 {
		t.Fatalf("audit file lines = %d, want 2", lines)
	}
}

func TestAuditLogRecordsInviteAndGatesAccess(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	values := url.Values{
		"email":       {"new.resident@example.com"},
		"first_name":  {"New"},
		"last_name":   {"Resident"},
		"role":        {"Mieter"},
		"permissions": {permissionParking},
	}
	create := authedFormRequest(t, a, "manager@example.com", "/app/settings/users", values, a.createInvite)
	if create.Code != http.StatusSeeOther {
		t.Fatalf("create invite status = %d, want redirect", create.Code)
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "jhw22", Action: auditActionInviteCreate, Query: "new.resident", Limit: 10})
	if len(events) != 1 || events[0].ActorEmail != "manager@example.com" || events[0].TargetID != "new.resident@example.com" {
		t.Fatalf("invite audit events = %+v", events)
	}
	body := authedRequest(t, a, "manager@example.com", "/app/audit", a.auditLog)
	if body.Code != http.StatusOK || !strings.Contains(body.Body.String(), "Einladung angelegt") || !strings.Contains(body.Body.String(), "new.resident@example.com") {
		t.Fatalf("manager audit page status/body = %d\n%s", body.Code, body.Body.String())
	}
	resident := authedRequest(t, a, "resident@example.com", "/app/audit", a.auditLog)
	if resident.Code != http.StatusForbidden {
		t.Fatalf("resident audit status = %d, want 403", resident.Code)
	}
}

func TestDocumentUploadRecordsMetadataAndAudit(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	fields := map[string]string{
		"title":      "Hausordnung",
		"category":   documentCategoryRules,
		"visibility": documentVisibilityAllResidents,
	}
	upload := authedMultipartFileRequest(t, a, "manager@example.com", "/app/dokumente", fields, "document", "hausordnung.pdf", []byte("%PDF-1.4\n% weg portal test\n"), a.uploadDocument)
	if upload.Code != http.StatusSeeOther {
		t.Fatalf("upload status = %d, want redirect", upload.Code)
	}
	docs := a.documentStore.ListTenant("jhw22")
	if len(docs) != 1 || docs[0].Title != "Hausordnung" || docs[0].Visibility != documentVisibilityAllResidents {
		t.Fatalf("stored docs = %+v", docs)
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "jhw22", Action: auditActionDocumentUpload, Limit: 10})
	if len(events) != 1 || events[0].TargetID != docs[0].ID || events[0].Details["title"] != "Hausordnung" {
		t.Fatalf("audit events = %+v", events)
	}
	page := authedRequest(t, a, "resident@example.com", "/app/dokumente", a.documents)
	if page.Code != http.StatusOK {
		t.Fatalf("resident documents status = %d, want 200", page.Code)
	}
	body := page.Body.String()
	for _, want := range []string{`href="/app/dokumente"`, "Hausordnung", "Alle Bewohner"} {
		if !strings.Contains(body, want) {
			t.Fatalf("documents page missing %q", want)
		}
	}
	residentUpload := authedMultipartFileRequest(t, a, "resident@example.com", "/app/dokumente", fields, "document", "resident.pdf", []byte("%PDF-1.4\n% weg portal test\n"), a.uploadDocument)
	if residentUpload.Code != http.StatusForbidden {
		t.Fatalf("resident upload status = %d, want 403", residentUpload.Code)
	}
}

func TestDocumentsPageFiltersManagerOnlyMetadata(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	if _, err := a.documentStore.Create(documentRecord{
		TenantSlug: "jhw22",
		Title:      "Internes Protokoll",
		Category:   documentCategoryProtocol,
		Visibility: documentVisibilityManagerOnly,
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "intern.pdf", []byte("%PDF-1.4\n% intern\n")), time.Now()); err != nil {
		t.Fatalf("create manager-only doc: %v", err)
	}
	if _, err := a.documentStore.Create(documentRecord{
		TenantSlug: "jhw22",
		Title:      "Hausordnung",
		Category:   documentCategoryRules,
		Visibility: documentVisibilityAllResidents,
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "hausordnung.pdf", []byte("%PDF-1.4\n% public\n")), time.Now()); err != nil {
		t.Fatalf("create public doc: %v", err)
	}
	residentPage := authedRequest(t, a, "resident@example.com", "/app/dokumente", a.documents)
	if residentPage.Code != http.StatusOK {
		t.Fatalf("resident documents status = %d, want 200", residentPage.Code)
	}
	body := residentPage.Body.String()
	if strings.Contains(body, "Internes Protokoll") {
		t.Fatal("resident page must not expose manager-only document metadata")
	}
	if !strings.Contains(body, "Hausordnung") {
		t.Fatal("resident page should show all-residents document")
	}
	managerPage := authedRequest(t, a, "manager@example.com", "/app/dokumente", a.documents)
	if !strings.Contains(managerPage.Body.String(), "Internes Protokoll") {
		t.Fatal("manager page should show manager-only document")
	}
}

func TestDocumentsPageSearchSortAndCategoryEmptyStates(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	oldDoc, err := a.documentStore.Create(documentRecord{
		TenantSlug: "jhw22",
		Title:      "Abrechnung 2025",
		Category:   documentCategoryBilling,
		Visibility: documentVisibilityAllResidents,
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "abrechnung-2025.pdf", []byte("%PDF-1.4\nold\n")), time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("create old doc: %v", err)
	}
	newDoc, err := a.documentStore.Create(documentRecord{
		TenantSlug: "jhw22",
		Title:      "Abrechnung 2026",
		Category:   documentCategoryBilling,
		Visibility: documentVisibilityAllResidents,
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "abrechnung-2026.pdf", []byte("%PDF-1.4\nnew\n")), time.Date(2026, 2, 5, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("create new doc: %v", err)
	}
	page := authedRequest(t, a, "manager@example.com", "/app/dokumente?sort=oldest", a.documents)
	body := page.Body.String()
	oldIndex := strings.Index(body, oldDoc.Title)
	newIndex := strings.Index(body, newDoc.Title)
	if oldIndex < 0 || newIndex < 0 || oldIndex > newIndex {
		t.Fatalf("oldest sort order not reflected: old=%d new=%d", oldIndex, newIndex)
	}
	if !strings.Contains(body, "Keine passenden Dokumente in dieser Kategorie.") {
		t.Fatal("category empty state missing")
	}
	filtered := authedRequest(t, a, "manager@example.com", "/app/dokumente?q=2025", a.documents)
	filteredBody := filtered.Body.String()
	if !strings.Contains(filteredBody, oldDoc.Title) || strings.Contains(filteredBody, newDoc.Title) {
		t.Fatalf("search filtering body = %s", filteredBody)
	}
}

func TestDocumentDownloadEnforcesVisibilityAndAudits(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	a.profiles["owner@example.com"] = userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	a.profiles["renter@example.com"] = userProfile{Email: "renter@example.com", Role: roleRenter, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	a.profiles["beirat@example.com"] = userProfile{Email: "beirat@example.com", Role: roleBeirat, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	if err := a.unitStore.SetTenantUnits("jhw22", []unit{
		{ID: "top-1", TenantSlug: "jhw22", Label: "Top 1", MiteigentumsanteilPPM: 10000, OwnerEmails: []string{"owner@example.com"}, RenterEmails: []string{"renter@example.com"}},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	publicDoc, err := a.documentStore.Create(documentRecord{
		TenantSlug: "jhw22",
		Title:      "Hausordnung",
		Category:   documentCategoryRules,
		Visibility: documentVisibilityAllResidents,
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "hausordnung.pdf", []byte("%PDF-1.4\npublic\n")), time.Now())
	if err != nil {
		t.Fatalf("create public doc: %v", err)
	}
	ownerDoc, err := a.documentStore.Create(documentRecord{
		TenantSlug: "jhw22",
		Title:      "Top 1 Abrechnung",
		Category:   documentCategoryBilling,
		Visibility: documentVisibilityOwnersOnly,
		UnitID:     "top-1",
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "top-1.pdf", []byte("%PDF-1.4\nowner\n")), time.Now())
	if err != nil {
		t.Fatalf("create owner doc: %v", err)
	}
	managerDoc, err := a.documentStore.Create(documentRecord{
		TenantSlug: "jhw22",
		Title:      "Interne Notiz",
		Category:   documentCategoryOther,
		Visibility: documentVisibilityManagerOnly,
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "intern.pdf", []byte("%PDF-1.4\nmanager\n")), time.Now())
	if err != nil {
		t.Fatalf("create manager doc: %v", err)
	}

	cases := []struct {
		email string
		doc   documentRecord
		want  int
	}{
		{"resident@example.com", publicDoc, http.StatusOK},
		{"resident@example.com", ownerDoc, http.StatusForbidden},
		{"resident@example.com", managerDoc, http.StatusForbidden},
		{"owner@example.com", publicDoc, http.StatusOK},
		{"owner@example.com", ownerDoc, http.StatusOK},
		{"owner@example.com", managerDoc, http.StatusForbidden},
		{"renter@example.com", publicDoc, http.StatusOK},
		{"renter@example.com", ownerDoc, http.StatusForbidden},
		{"beirat@example.com", publicDoc, http.StatusOK},
		{"manager@example.com", publicDoc, http.StatusOK},
		{"manager@example.com", ownerDoc, http.StatusOK},
		{"manager@example.com", managerDoc, http.StatusOK},
	}
	authorized := 0
	for _, tc := range cases {
		rr := authedPathValueRequest(t, a, tc.email, "/app/dokumente/"+tc.doc.ID+"/download", map[string]string{"id": tc.doc.ID}, a.downloadDocument)
		if rr.Code != tc.want {
			t.Fatalf("%s downloading %s status = %d, want %d", tc.email, tc.doc.Title, rr.Code, tc.want)
		}
		if tc.want == http.StatusOK {
			authorized++
			if !strings.Contains(rr.Header().Get("Content-Disposition"), "attachment") {
				t.Fatalf("download missing attachment disposition: %q", rr.Header().Get("Content-Disposition"))
			}
			if !strings.Contains(rr.Body.String(), "%PDF-1.4") {
				t.Fatalf("download body missing file content for %s", tc.doc.Title)
			}
		}
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "jhw22", Action: auditActionDocumentDownload, Limit: 50})
	if len(events) != authorized {
		t.Fatalf("download audit count = %d, want %d: %+v", len(events), authorized, events)
	}
}

func TestDocumentReplaceShowsHistoryAndDownloadAuditVersion(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	created, err := a.documentStore.Create(documentRecord{
		TenantSlug: "jhw22",
		Title:      "Hausordnung",
		Category:   documentCategoryRules,
		Visibility: documentVisibilityAllResidents,
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "hausordnung-v1.pdf", []byte("%PDF-1.4\nv1\n")), time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("create doc: %v", err)
	}
	replace := authedMultipartFileRequest(t, a, "manager@example.com", "/app/dokumente/replace", map[string]string{"id": created.ID}, "document", "hausordnung-v2.pdf", []byte("%PDF-1.4\nv2\n"), a.replaceDocument)
	if replace.Code != http.StatusSeeOther {
		t.Fatalf("replace status = %d, want redirect", replace.Code)
	}
	current := a.documentStore.ListCurrentTenant("jhw22")
	if len(current) != 1 || current[0].Version != 2 {
		t.Fatalf("current docs = %+v", current)
	}
	page := authedRequest(t, a, "manager@example.com", "/app/dokumente", a.documents)
	body := page.Body.String()
	for _, want := range []string{"Version 2", "Ältere Versionen", "Version 1", "Ersetzen"} {
		if !strings.Contains(body, want) {
			t.Fatalf("documents page missing %q", want)
		}
	}
	downloadOld := authedPathValueRequest(t, a, "manager@example.com", "/app/dokumente/"+created.ID+"/download", map[string]string{"id": created.ID}, a.downloadDocument)
	if downloadOld.Code != http.StatusOK || !strings.Contains(downloadOld.Body.String(), "v1") {
		t.Fatalf("old version download status/body = %d %q", downloadOld.Code, downloadOld.Body.String())
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "jhw22", Action: auditActionDocumentDownload, Limit: 10})
	if len(events) != 1 || events[0].TargetID != created.ID || events[0].Details["version"] != "Version 1" {
		t.Fatalf("download audit events = %+v", events)
	}
	replaceEvents := a.auditStore.List(auditFilter{TenantSlug: "jhw22", Action: auditActionDocumentReplace, Limit: 10})
	if len(replaceEvents) != 1 || replaceEvents[0].Details["version"] != "Version 2" || replaceEvents[0].Details["previous"] != "Version 1" {
		t.Fatalf("replace audit events = %+v", replaceEvents)
	}
}

func TestManagerCanManageTenantSurfacesButNotPlatformSettings(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})

	users := authedRequest(t, a, "manager@example.com", "/app/settings/users", a.userSettings)
	if users.Code != http.StatusOK {
		t.Fatalf("manager user settings status = %d, want 200", users.Code)
	}
	if !strings.Contains(users.Body.String(), "Benutzer &amp; Rechte") {
		t.Fatal("manager should see user management page")
	}
	if strings.Contains(users.Body.String(), `<option value="Admin"`) {
		t.Fatal("manager user management must not offer Admin role assignment")
	}
	adminInvite := authedFormRequest(t, a, "manager@example.com", "/app/settings/users", url.Values{
		"email": {"new-admin@example.com"},
		"role":  {"Admin"},
	}, a.createInvite)
	if adminInvite.Code != http.StatusSeeOther {
		t.Fatalf("manager admin invite status = %d, want redirect", adminInvite.Code)
	}
	if _, ok := a.inviteStore.Get("new-admin@example.com"); ok {
		t.Fatal("manager must not be able to create an Admin invite")
	}
	if _, err := a.inviteStore.Add(userProfile{Email: "persisted-admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("Add admin invite: %v", err)
	}
	deleteAdmin := authedFormRequest(t, a, "manager@example.com", "/app/settings/users/delete", url.Values{
		"email": {"persisted-admin@example.com"},
	}, a.deleteInvite)
	if deleteAdmin.Code != http.StatusSeeOther {
		t.Fatalf("manager admin delete status = %d, want redirect", deleteAdmin.Code)
	}
	if _, ok := a.inviteStore.Get("persisted-admin@example.com"); !ok {
		t.Fatal("manager must not be able to delete an Admin invite")
	}
	values := url.Values{
		"title": {"Manager post"},
		"body":  {"Allowed"},
	}
	write := authedFormRequest(t, a, "manager@example.com", "/app/announcements", values, a.createAnnouncement)
	if write.Code != http.StatusSeeOther {
		t.Fatalf("manager announcement create status = %d, want redirect", write.Code)
	}
	ballotDeadline := time.Now().Add(48 * time.Hour).In(time.Local).Format("2006-01-02T15:04")
	ballotCreate := authedFormRequest(t, a, "manager@example.com", "/app/abstimmungen", url.Values{
		"title":                 {"Dachsanierung"},
		"description":           {"Beschluss zur Beauftragung"},
		"options_text":          {"Ja\nNein\nEnthaltung"},
		"type":                  {ballotTypeCircular},
		"weighting":             {ballotWeightingPerShare},
		"quorum_percent":        {"50"},
		"closes_at":             {ballotDeadline},
		"reminder_before_hours": {"12"},
	}, a.createBallot)
	if ballotCreate.Code != http.StatusSeeOther {
		t.Fatalf("manager ballot create status = %d, want redirect", ballotCreate.Code)
	}
	ballots := a.voteStore.ListTenant("jhw22")
	if len(ballots) != 1 || ballots[0].Title != "Dachsanierung" || ballots[0].QuorumPPM != 500000 || ballots[0].ReminderBeforeMinutes != 720 {
		t.Fatalf("created ballots = %+v", ballots)
	}
	open := authedFormRequest(t, a, "manager@example.com", "/app/abstimmungen/open", url.Values{"id": {ballots[0].ID}}, a.openBallot)
	if open.Code != http.StatusSeeOther {
		t.Fatalf("manager ballot open status = %d, want redirect", open.Code)
	}
	close := authedFormRequest(t, a, "manager@example.com", "/app/abstimmungen/close", url.Values{"id": {ballots[0].ID}}, a.closeBallot)
	if close.Code != http.StatusSeeOther {
		t.Fatalf("manager ballot close status = %d, want redirect", close.Code)
	}
	closed, _ := a.voteStore.Get("jhw22", ballots[0].ID)
	if closed.Status != ballotStatusClosed {
		t.Fatalf("closed ballot = %+v", closed)
	}
	for _, action := range []string{auditActionVoteCreate, auditActionVoteOpen, auditActionVoteClose} {
		events := a.auditStore.List(auditFilter{TenantSlug: "jhw22", Action: action, Limit: 10})
		if len(events) != 1 {
			t.Fatalf("audit events for %s = %+v", action, events)
		}
	}
	parkingSettings := authedRequest(t, a, "manager@example.com", "/app/parking/settings", a.parkingSettings)
	if parkingSettings.Code != http.StatusForbidden {
		t.Fatalf("manager parking settings status = %d, want 403", parkingSettings.Code)
	}
}

func TestResidentCannotManageTenantUsers(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})

	users := authedRequest(t, a, "resident@example.com", "/app/settings/users", a.userSettings)
	if users.Code != http.StatusForbidden {
		t.Fatalf("resident user settings status = %d, want 403", users.Code)
	}
	ballotCreate := authedFormRequest(t, a, "resident@example.com", "/app/abstimmungen", url.Values{
		"title":        {"Nicht erlaubt"},
		"options_text": {"Ja\nNein"},
	}, a.createBallot)
	if ballotCreate.Code != http.StatusForbidden {
		t.Fatalf("resident ballot create status = %d, want 403", ballotCreate.Code)
	}
}

func TestParkingSettingsIsParkingSpecificNotGlobalSettings(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})

	rr := authedRequest(t, a, "admin@example.com", "/app/parking/settings", a.parkingSettings)
	if rr.Code != http.StatusOK {
		t.Fatalf("parking settings status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{"Parkplatz-Abrechnung", `href="/app/settings"`, "Zurück zu Einstellungen"} {
		if !strings.Contains(body, want) {
			t.Fatalf("parking settings should contain %q", want)
		}
	}
	if strings.Contains(body, "<h1>Einstellungen</h1>") {
		t.Fatal("parking settings page must not use generic Einstellungen heading")
	}
}

func TestParkingSettingsSavesEffectiveTariffHistory(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})

	save := authedFormRequest(t, a, "admin@example.com", "/app/parking/settings", url.Values{
		"effective_from":       {"2026-07-01"},
		"grid_fee_eur_per_kwh": {"0.12"},
		"base_fee_eur":         {"5.50"},
	}, a.updateParkingSettings)
	if save.Code != http.StatusSeeOther {
		t.Fatalf("parking tariff save status = %d, want redirect", save.Code)
	}
	data := a.parkingStore.TenantData("jhw22")
	tariff := parkingTariffAt(data.Settings, time.Date(2026, 7, 15, 0, 0, 0, 0, time.Local), time.Local)
	if tariff.EffectiveFrom != "2026-07-01" {
		t.Fatalf("saved tariff = %+v", tariff)
	}
	assertClose(t, tariff.GridFeeEURPerKWh, 0.12)
	assertClose(t, tariff.BaseFeeEUR, 5.50)
	events := a.auditStore.List(auditFilter{TenantSlug: "jhw22", Action: auditActionParkingSettings, Limit: 10})
	if len(events) != 1 || events[0].Details["base_fee"] != "5,50 €" || events[0].Details["effective_from"] == "" {
		t.Fatalf("parking tariff audit events = %+v", events)
	}
	page := authedRequest(t, a, "admin@example.com", "/app/parking/settings", a.parkingSettings)
	body := page.Body.String()
	for _, want := range []string{"Tarif", "2026-07-01", "0,120 €/kWh", "Basis 5,50 €"} {
		if !strings.Contains(body, want) {
			t.Fatalf("parking settings page missing %q:\n%s", want, body)
		}
	}
}

func TestAnnouncementStoreCRUDVisibleSortPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "announcements.json")
	store, err := newAnnouncementStore(path)
	if err != nil {
		t.Fatalf("newAnnouncementStore: %v", err)
	}
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	expiredAt := now.Add(-time.Hour)
	future, err := store.Create(announcement{TenantSlug: "jhw22", Title: "Future", Body: "Later", Category: "Info", PublishedAt: now.Add(time.Hour)})
	if err != nil || future.ID == "" {
		t.Fatalf("create future: item=%+v err=%v", future, err)
	}
	expired, _ := store.Create(announcement{TenantSlug: "jhw22", Title: "Expired", Body: "Old", Category: "Info", PublishedAt: now.Add(-2 * time.Hour), ExpiresAt: &expiredAt})
	normal, _ := store.Create(announcement{TenantSlug: "jhw22", Title: "Normal", Body: "Visible", Category: "Termin", PublishedAt: now.Add(-30 * time.Minute)})
	pinned, _ := store.Create(announcement{TenantSlug: "jhw22", Title: "Pinned", Body: "Top", Category: "Dringend", Pinned: true, PublishedAt: now.Add(-2 * time.Hour)})
	_, _ = store.Create(announcement{TenantSlug: "other", Title: "Other", Body: "Hidden", Category: "Info", PublishedAt: now.Add(-time.Hour)})

	visible := store.Visible("jhw22", now)
	if len(visible) != 2 {
		t.Fatalf("visible len = %d, want 2 (future=%s expired=%s normal=%s pinned=%s)", len(visible), future.ID, expired.ID, normal.ID, pinned.ID)
	}
	if visible[0].Title != "Pinned" || visible[1].Title != "Normal" {
		t.Fatalf("visible order = %q, %q; want pinned first then recent", visible[0].Title, visible[1].Title)
	}

	updated := normal
	updated.Title = "Updated"
	if ok, err := store.Update(normal.ID, updated); !ok || err != nil {
		t.Fatalf("update: ok=%v err=%v", ok, err)
	}
	if removed, err := store.Delete("jhw22", pinned.ID); !removed || err != nil {
		t.Fatalf("delete: removed=%v err=%v", removed, err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("store file mode = %v err=%v, want 0600", info.Mode().Perm(), err)
	}
	reopened, err := newAnnouncementStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	all := reopened.ListTenant("jhw22")
	if len(all) != 3 {
		t.Fatalf("reopened list len = %d, want 3 after delete", len(all))
	}
	foundUpdated := false
	for _, item := range all {
		if item.ID == normal.ID && item.Title == "Updated" {
			foundUpdated = true
		}
	}
	if !foundUpdated {
		t.Fatal("updated announcement did not persist")
	}
}

func TestAnnouncementReadStorePersistsSeenState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "announcement_reads.json")
	store, err := newAnnouncementReadStore(path)
	if err != nil {
		t.Fatalf("newAnnouncementReadStore: %v", err)
	}
	seenAt := time.Date(2026, 7, 6, 12, 30, 0, 0, time.UTC)
	if err := store.MarkSeen("jhw22", "Resident@Example.com", seenAt); err != nil {
		t.Fatalf("mark seen: %v", err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("read store file mode = %v err=%v, want 0600", info.Mode().Perm(), err)
	}
	reopened, err := newAnnouncementReadStore(path)
	if err != nil {
		t.Fatalf("reopen read store: %v", err)
	}
	if got := reopened.LastSeen("jhw22", "resident@example.com"); !got.Equal(seenAt) {
		t.Fatalf("last seen = %v, want %v", got, seenAt)
	}
}

func TestEventStoreCRUDUpcomingPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.json")
	store, err := newEventStore(path)
	if err != nil {
		t.Fatalf("newEventStore: %v", err)
	}
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	past, err := store.Create(houseEvent{TenantSlug: "jhw22", Title: "Alte Reinigung", Category: "Reinigung", StartsAt: now.AddDate(0, 0, -2)})
	if err != nil {
		t.Fatalf("create past: %v", err)
	}
	future, err := store.Create(houseEvent{TenantSlug: "jhw22", Title: "Eigentümerversammlung", Category: "meeting", Location: "Hof", StartsAt: now.Add(48 * time.Hour)})
	if err != nil {
		t.Fatalf("create future: %v", err)
	}
	_, _ = store.Create(houseEvent{TenantSlug: "other", Title: "Other", Category: "Wartung", StartsAt: now.Add(24 * time.Hour)})

	upcoming := store.Upcoming("jhw22", now)
	if len(upcoming) != 1 || upcoming[0].ID != future.ID || upcoming[0].Category != "Eigentümerversammlung" {
		t.Fatalf("upcoming = %+v, want only normalized future event", upcoming)
	}

	updated := future
	updated.Title = "Versammlung aktualisiert"
	updated.StartsAt = now.Add(72 * time.Hour)
	if ok, err := store.Update(future.ID, updated); !ok || err != nil {
		t.Fatalf("update: ok=%v err=%v", ok, err)
	}
	if removed, err := store.Delete("jhw22", past.ID); !removed || err != nil {
		t.Fatalf("delete past: removed=%v err=%v", removed, err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("store file mode = %v err=%v, want 0600", info.Mode().Perm(), err)
	}

	reopened, err := newEventStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	events := reopened.ListTenant("jhw22")
	if len(events) != 1 || events[0].Title != "Versammlung aktualisiert" {
		t.Fatalf("reopened events = %+v", events)
	}
}

func TestPortalUsesAnnouncementEmptyStateWithoutPrototypeCopy(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})

	rr := authedRequest(t, a, "resident@example.com", "/app", a.portal)
	if rr.Code != http.StatusOK {
		t.Fatalf("portal status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, forbidden := range []string{"Willkommen im Prototyp", "Beispielmodule", "Nächste Ausbaustufe"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("portal must not contain placeholder copy %q", forbidden)
		}
	}
	if !strings.Contains(body, "Noch keine Beiträge") || !strings.Contains(body, `href="/app/announcements"`) {
		t.Fatal("portal should show announcement empty state and real archive link")
	}
}

func TestPortalDigestAggregatesRoleScopedAttentionItems(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", FirstName: "Resi", LastName: "Dent", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["manager@example.com"] = userProfile{Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	a.profiles["other@example.com"] = userProfile{Email: "other@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	now := time.Now()
	_, _ = a.announcementStore.Create(announcement{TenantSlug: "jhw22", Title: "Liftwartung", Body: "Lift Freitag", Category: "Wartung", PublishedAt: now.Add(-time.Hour)})
	_, _ = a.eventStore.Create(houseEvent{TenantSlug: "jhw22", Title: "Versammlung", Category: "Eigentümerversammlung", StartsAt: now.Add(48 * time.Hour)})
	_, _ = a.issueStore.Create(residentIssue{TenantSlug: "jhw22", AuthorEmail: "resident@example.com", AuthorName: "Resi Dent", Category: "Reparatur", Title: "Eigenes Anliegen", Body: "Offen", LocationType: issueLocationUnit, Status: issueStatusNew, Priority: issuePriorityNorm})
	_, _ = a.issueStore.Create(residentIssue{TenantSlug: "jhw22", AuthorEmail: "other@example.com", AuthorName: "Other", Category: "Reparatur", Title: "Privates Anliegen", Body: "Offen", LocationType: issueLocationUnit, Status: issueStatusNew, Priority: issuePriorityNorm})

	resident := authedRequest(t, a, "resident@example.com", "/app", a.portal).Body.String()
	for _, want := range []string{"Was ist neu", "Neue Aushänge", "1 ungelesener Beitrag", "Offene Anliegen", "1 offenes Anliegen", `href="/app/anliegen"`, "Kommende Termine", "1 Termin geplant"} {
		if !strings.Contains(resident, want) {
			t.Fatalf("resident digest should contain %q", want)
		}
	}
	if strings.Contains(resident, "2 offene Anliegen") || strings.Contains(resident, `href="/app/anliegen/board"`) {
		t.Fatalf("resident digest must be role-scoped:\n%s", resident)
	}

	manager := authedRequest(t, a, "manager@example.com", "/app", a.portal).Body.String()
	for _, want := range []string{"Offene Anliegen im Haus", "2 offene Anliegen", `href="/app/anliegen/board"`} {
		if !strings.Contains(manager, want) {
			t.Fatalf("manager digest should contain %q", want)
		}
	}
}

func TestPortalDashboardShowsRoleScopedDocumentsAndParking(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["parker@example.com"] = userProfile{Email: "parker@example.com", Role: roleRenter, Tenants: []string{"jhw22"}, Permissions: []string{permissionParking}, AuthMethods: defaultAuthMethods()}
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}

	upload := authedMultipartFileRequest(t, a, "manager@example.com", "/app/dokumente", map[string]string{
		"title":      "Hausordnung",
		"category":   documentCategoryRules,
		"visibility": documentVisibilityAllResidents,
	}, "document", "hausordnung.pdf", []byte("%PDF-1.4\n% weg portal test\n"), a.uploadDocument)
	if upload.Code != http.StatusSeeOther {
		t.Fatalf("upload status = %d, want redirect", upload.Code)
	}

	parker := authedRequest(t, a, "parker@example.com", "/app", a.portal).Body.String()
	for _, want := range []string{"Dokumente", "Hausordnung", "Parkplatznutzung", "Alles erledigt"} {
		if !strings.Contains(parker, want) {
			t.Fatalf("parking user dashboard should contain %q", want)
		}
	}

	resident := authedRequest(t, a, "resident@example.com", "/app", a.portal).Body.String()
	if !strings.Contains(resident, "Hausordnung") {
		t.Fatal("resident dashboard should show all-resident documents")
	}
	if strings.Contains(resident, "Parkplatznutzung") || strings.Contains(resident, `href="/app/parking"`) {
		t.Fatal("resident without parking permission must not see parking dashboard links")
	}
}

func TestEventsPageCRUDAndDashboardAgenda(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", FirstName: "Resi", LastName: "Dent", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	start := time.Now().Add(48 * time.Hour).In(time.Local).Format("2006-01-02T15:04")

	create := authedFormRequest(t, a, "manager@example.com", "/app/events", url.Values{
		"title":     {"Liftwartung"},
		"category":  {"Wartung"},
		"starts_at": {start},
		"location":  {"Stiegenhaus"},
		"body":      {"Lift außer Betrieb."},
	}, a.createEvent)
	if create.Code != http.StatusSeeOther {
		t.Fatalf("create event status = %d, want redirect", create.Code)
	}
	events := a.eventStore.ListTenant("jhw22")
	if len(events) != 1 || events[0].Title != "Liftwartung" || events[0].Category != "Wartung" {
		t.Fatalf("stored events = %+v", events)
	}

	page := authedRequest(t, a, "resident@example.com", "/app/events", a.events)
	if page.Code != http.StatusOK {
		t.Fatalf("resident events status = %d", page.Code)
	}
	body := page.Body.String()
	for _, want := range []string{"Liftwartung", "Stiegenhaus", "Lift außer Betrieb.", `href="/app/events"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("events page should contain %q", want)
		}
	}
	for _, forbidden := range []string{`data-dialog="event-create"`, `action="/app/events/delete"`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("resident events page must not contain management control %q", forbidden)
		}
	}
	dashboard := authedRequest(t, a, "resident@example.com", "/app", a.portal)
	if dashboard.Code != http.StatusOK || !strings.Contains(dashboard.Body.String(), "Liftwartung") {
		t.Fatalf("dashboard should show upcoming event, status=%d body=\n%s", dashboard.Code, dashboard.Body.String())
	}

	editedStart := time.Now().Add(72 * time.Hour).In(time.Local).Format("2006-01-02T15:04")
	edit := authedFormRequest(t, a, "manager@example.com", "/app/events/edit", url.Values{
		"id":        {events[0].ID},
		"title":     {"Hofreinigung"},
		"category":  {"Reinigung"},
		"starts_at": {editedStart},
		"location":  {"Hof"},
	}, a.editEvent)
	if edit.Code != http.StatusSeeOther {
		t.Fatalf("edit event status = %d, want redirect", edit.Code)
	}
	events = a.eventStore.ListTenant("jhw22")
	if len(events) != 1 || events[0].Title != "Hofreinigung" || events[0].Category != "Reinigung" || events[0].Location != "Hof" {
		t.Fatalf("edited events = %+v", events)
	}

	deleteResp := authedFormRequest(t, a, "manager@example.com", "/app/events/delete", url.Values{"id": {events[0].ID}}, a.deleteEvent)
	if deleteResp.Code != http.StatusSeeOther {
		t.Fatalf("delete event status = %d, want redirect", deleteResp.Code)
	}
	if got := a.eventStore.Upcoming("jhw22", time.Now()); len(got) != 0 {
		t.Fatalf("events after delete = %+v, want none", got)
	}
	empty := authedRequest(t, a, "resident@example.com", "/app/events", a.events)
	if !strings.Contains(empty.Body.String(), "Noch keine kommenden Termine") {
		t.Fatalf("empty events page should show empty state:\n%s", empty.Body.String())
	}
}

func TestPortalListsRealAnnouncementsPinnedFirstWithoutDeadTiles(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	now := time.Now().Add(-2 * time.Hour)
	expiredAt := now.Add(time.Hour)
	_, _ = a.announcementStore.Create(announcement{TenantSlug: "jhw22", Title: "Normaler Hinweis", Body: "Aktuell", Category: "Info", PublishedAt: now.Add(time.Hour)})
	_, _ = a.announcementStore.Create(announcement{TenantSlug: "jhw22", Title: "Fixierter Hinweis", Body: "Wichtig", Category: "Dringend", Pinned: true, PublishedAt: now})
	_, _ = a.announcementStore.Create(announcement{TenantSlug: "jhw22", Title: "Alter Hinweis", Body: "Abgelaufen", Category: "Info", PublishedAt: now.Add(-time.Hour), ExpiresAt: &expiredAt})
	_, _ = a.announcementStore.Create(announcement{TenantSlug: "jhw22", Title: "Geplanter Hinweis", Body: "Zukunft", Category: "Info", PublishedAt: time.Now().Add(time.Hour)})

	rr := authedRequest(t, a, "resident@example.com", "/app", a.portal)
	if rr.Code != http.StatusOK {
		t.Fatalf("portal status = %d", rr.Code)
	}
	body := rr.Body.String()
	pinnedIndex := strings.Index(body, "Fixierter Hinweis")
	normalIndex := strings.Index(body, "Normaler Hinweis")
	if pinnedIndex < 0 || normalIndex < 0 || pinnedIndex > normalIndex {
		t.Fatalf("portal should render pinned current announcement before normal current announcement:\n%s", body)
	}
	for _, forbidden := range []string{"Alter Hinweis", "Geplanter Hinweis", "info-card", `class="quick-row disabled"`, "Schnellzugriff", `class="quick-row" href="/app/announcements"`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("portal must not contain %q", forbidden)
		}
	}
	if !strings.Contains(body, `href="/app/announcements"`) || !strings.Contains(body, "Aktueller Aushang") {
		t.Fatal("portal should keep the announcement card and archive link")
	}
}

func TestIssuesPageRendersResidentFormAndNav(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})

	rr := authedRequest(t, a, "resident@example.com", "/app/anliegen", a.issues)
	if rr.Code != http.StatusOK {
		t.Fatalf("issues status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{`href="/app/anliegen"`, "Anliegen", "Neues Anliegen", `enctype="multipart/form-data"`, `name="category"`, `name="location_type"`, `name="photo"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("issues page should contain %q", want)
		}
	}
	if strings.Contains(body, `class="nav-item disabled"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 18.5`) {
		t.Fatal("Anliegen nav item must be a live link, not a disabled placeholder")
	}
}

func TestResidentCanSubmitIssueWithPhoto(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", FirstName: "Resi", LastName: "Dent", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	attachmentDir := filepath.Join(t.TempDir(), "issue-attachments")
	store, err := newIssueStore(filepath.Join(t.TempDir(), "issues.json"), attachmentDir)
	if err != nil {
		t.Fatalf("newIssueStore: %v", err)
	}
	a.issueStore = store

	rr := authedMultipartRequest(t, a, "resident@example.com", "/app/anliegen", map[string]string{
		"category":        "Reparatur",
		"location_type":   issueLocationCommon,
		"location_detail": "Stiegenhaus",
		"title":           "Licht flackert",
		"body":            "Das Licht im Stiegenhaus flackert seit gestern.",
	}, "licht.png", minimalPNG(), a.createIssue)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("submit issue status = %d, want redirect", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/app/anliegen?issue=created" {
		t.Fatalf("redirect = %q", loc)
	}
	issues := store.ListAuthor("jhw22", "resident@example.com")
	if len(issues) != 1 {
		t.Fatalf("stored issues = %+v", issues)
	}
	issue := issues[0]
	if issue.Category != "Reparatur" || issue.LocationType != issueLocationCommon || issue.LocationDetail != "Stiegenhaus" || issue.Status != issueStatusOpen {
		t.Fatalf("stored issue fields = %+v", issue)
	}
	if len(issue.PhotoPaths) != 1 {
		t.Fatalf("photo paths = %+v, want one", issue.PhotoPaths)
	}
	photoPath := filepath.Join(attachmentDir, "jhw22", filepath.Base(issue.PhotoPaths[0]))
	info, err := os.Stat(photoPath)
	if err != nil {
		t.Fatalf("stat photo: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("photo mode = %v, want 0600", info.Mode().Perm())
	}

	page := authedRequest(t, a, "resident@example.com", "/app/anliegen", a.issues)
	if !strings.Contains(page.Body.String(), "Licht flackert") || !strings.Contains(page.Body.String(), "1 Foto") {
		t.Fatalf("issues page should show submitted issue with photo count:\n%s", page.Body.String())
	}
}

func TestIssueSubmitRejectsInvalidPhotoType(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	store, err := newIssueStore(filepath.Join(t.TempDir(), "issues.json"), filepath.Join(t.TempDir(), "issue-attachments"))
	if err != nil {
		t.Fatalf("newIssueStore: %v", err)
	}
	a.issueStore = store

	rr := authedMultipartRequest(t, a, "resident@example.com", "/app/anliegen", map[string]string{
		"category":      "Frage",
		"location_type": issueLocationUnit,
		"title":         "Dokument hochladen",
		"body":          "Wo soll ich das melden?",
	}, "not-a-photo.txt", []byte("plain text"), a.createIssue)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("invalid photo status = %d, want redirect", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/app/anliegen?issue=photo" {
		t.Fatalf("redirect = %q", loc)
	}
	if got := store.ListAuthor("jhw22", "resident@example.com"); len(got) != 0 {
		t.Fatalf("invalid photo must not create issue, got %+v", got)
	}
}

func TestManagerCanUpdateIssueWorkflow(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	issue, err := a.issueStore.Create(residentIssue{
		TenantSlug:   "jhw22",
		AuthorEmail:  "resident@example.com",
		AuthorName:   "Resident",
		Category:     "Reparatur",
		Title:        "Tür schließt nicht",
		Body:         "Die Haustür bleibt offen.",
		LocationType: issueLocationCommon,
	})
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	update := authedFormRequest(t, a, "manager@example.com", "/app/anliegen/workflow", url.Values{
		"id":             {issue.ID},
		"status":         {issueStatusProgress},
		"priority":       {issuePriorityUrgent},
		"assignee_email": {"manager@example.com"},
	}, a.updateIssueWorkflow)
	if update.Code != http.StatusSeeOther {
		t.Fatalf("manager workflow status = %d, want redirect", update.Code)
	}
	updated, ok := a.issueStore.Get("jhw22", issue.ID)
	if !ok {
		t.Fatal("updated issue not found")
	}
	if updated.Status != issueStatusProgress || updated.Priority != issuePriorityUrgent || updated.AssigneeEmail != "manager@example.com" {
		t.Fatalf("updated issue = %+v", updated)
	}
	if len(updated.StatusHistory) != 1 || updated.StatusHistory[0].ActorEmail != "manager@example.com" || updated.StatusHistory[0].From != issueStatusNew || updated.StatusHistory[0].To != issueStatusProgress {
		t.Fatalf("status history = %+v", updated.StatusHistory)
	}

	page := authedRequest(t, a, "manager@example.com", "/app/anliegen", a.issues)
	body := page.Body.String()
	for _, want := range []string{"Anliegen verwalten", "Tür schließt nicht", issuePriorityUrgent, `name="assignee_email"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("manager issues page should contain %q", want)
		}
	}
}

func TestResidentCanCloseAndReopenOwnIssueOnly(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	own, err := a.issueStore.Create(residentIssue{
		TenantSlug:   "jhw22",
		AuthorEmail:  "resident@example.com",
		AuthorName:   "Resident",
		Category:     "Frage",
		Title:        "Eigene Frage",
		Body:         "Bitte prüfen.",
		LocationType: issueLocationUnit,
	})
	if err != nil {
		t.Fatalf("Create own issue: %v", err)
	}
	other, err := a.issueStore.Create(residentIssue{
		TenantSlug:   "jhw22",
		AuthorEmail:  "other@example.com",
		AuthorName:   "Other",
		Category:     "Frage",
		Title:        "Andere Frage",
		Body:         "Nicht meine.",
		LocationType: issueLocationCommon,
	})
	if err != nil {
		t.Fatalf("Create other issue: %v", err)
	}

	closeOwn := authedFormRequest(t, a, "resident@example.com", "/app/anliegen/workflow", url.Values{
		"id":     {own.ID},
		"status": {issueStatusDone},
	}, a.updateIssueWorkflow)
	if closeOwn.Code != http.StatusSeeOther {
		t.Fatalf("resident close status = %d, want redirect", closeOwn.Code)
	}
	closed, _ := a.issueStore.Get("jhw22", own.ID)
	if closed.Status != issueStatusDone {
		t.Fatalf("closed status = %q", closed.Status)
	}
	reopenOwn := authedFormRequest(t, a, "resident@example.com", "/app/anliegen/workflow", url.Values{
		"id":     {own.ID},
		"status": {issueStatusNew},
	}, a.updateIssueWorkflow)
	if reopenOwn.Code != http.StatusSeeOther {
		t.Fatalf("resident reopen status = %d, want redirect", reopenOwn.Code)
	}
	reopened, _ := a.issueStore.Get("jhw22", own.ID)
	if reopened.Status != issueStatusNew {
		t.Fatalf("reopened status = %q", reopened.Status)
	}

	otherUpdate := authedFormRequest(t, a, "resident@example.com", "/app/anliegen/workflow", url.Values{
		"id":     {other.ID},
		"status": {issueStatusDone},
	}, a.updateIssueWorkflow)
	if otherUpdate.Code != http.StatusForbidden {
		t.Fatalf("resident other issue status = %d, want 403", otherUpdate.Code)
	}
	priorityUpdate := authedFormRequest(t, a, "resident@example.com", "/app/anliegen/workflow", url.Values{
		"id":       {own.ID},
		"status":   {issueStatusDone},
		"priority": {issuePriorityUrgent},
	}, a.updateIssueWorkflow)
	if priorityUpdate.Code != http.StatusForbidden {
		t.Fatalf("resident priority update status = %d, want 403", priorityUpdate.Code)
	}
}

func TestIssueCommentsRenderAndNotify(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", FirstName: "Resi", LastName: "Dent", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["manager@example.com"] = userProfile{Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	mailer := &recordingMailer{}
	a.mailer = mailer

	create := authedMultipartRequest(t, a, "resident@example.com", "/app/anliegen", map[string]string{
		"category":      "Frage",
		"location_type": issueLocationCommon,
		"title":         "Kommentar Test",
		"body":          "Bitte um Rückmeldung.",
	}, "", nil, a.createIssue)
	if create.Code != http.StatusSeeOther {
		t.Fatalf("create status = %d", create.Code)
	}
	issues := a.issueStore.ListAuthor("jhw22", "resident@example.com")
	if len(issues) != 1 {
		t.Fatalf("issues = %+v", issues)
	}
	if len(mailer.notifications) != 1 || mailer.notifications[0].To != "manager@example.com" || !strings.Contains(mailer.notifications[0].Subject, "Neues Anliegen") {
		t.Fatalf("new issue notifications = %+v", mailer.notifications)
	}

	managerComment := authedFormRequest(t, a, "manager@example.com", "/app/anliegen/comment", url.Values{
		"id":   {issues[0].ID},
		"body": {"Ich prüfe das und melde mich."},
	}, a.addIssueComment)
	if managerComment.Code != http.StatusSeeOther {
		t.Fatalf("manager comment status = %d", managerComment.Code)
	}
	updated, _ := a.issueStore.Get("jhw22", issues[0].ID)
	if len(updated.Comments) != 1 || updated.Comments[0].AuthorEmail != "manager@example.com" {
		t.Fatalf("comments after manager = %+v", updated.Comments)
	}
	if len(mailer.notifications) != 2 || mailer.notifications[1].To != "resident@example.com" || !strings.Contains(mailer.notifications[1].Subject, "Neuer Kommentar") {
		t.Fatalf("comment notifications = %+v", mailer.notifications)
	}

	residentComment := authedFormRequest(t, a, "resident@example.com", "/app/anliegen/comment", url.Values{
		"id":   {issues[0].ID},
		"body": {"Danke, ich ergänze ein Foto später."},
	}, a.addIssueComment)
	if residentComment.Code != http.StatusSeeOther {
		t.Fatalf("resident comment status = %d", residentComment.Code)
	}
	page := authedRequest(t, a, "resident@example.com", "/app/anliegen", a.issues)
	body := page.Body.String()
	first := strings.Index(body, "Ich prüfe das")
	second := strings.Index(body, "Danke, ich ergänze")
	if first < 0 || second < 0 || first > second {
		t.Fatalf("comments should render chronologically:\n%s", body)
	}
}

func TestResidentCannotCommentOnOtherIssue(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	other, err := a.issueStore.Create(residentIssue{
		TenantSlug:   "jhw22",
		AuthorEmail:  "other@example.com",
		AuthorName:   "Other",
		Category:     "Frage",
		Title:        "Nicht meine",
		Body:         "Privat.",
		LocationType: issueLocationCommon,
	})
	if err != nil {
		t.Fatalf("Create other issue: %v", err)
	}

	comment := authedFormRequest(t, a, "resident@example.com", "/app/anliegen/comment", url.Values{
		"id":   {other.ID},
		"body": {"Kann ich nicht sehen."},
	}, a.addIssueComment)
	if comment.Code != http.StatusForbidden {
		t.Fatalf("other comment status = %d, want 403", comment.Code)
	}
	unchanged, _ := a.issueStore.Get("jhw22", other.ID)
	if len(unchanged.Comments) != 0 {
		t.Fatalf("other issue comments = %+v", unchanged.Comments)
	}
}

func TestIssueVisibilityByPersona(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "renter@example.com", Role: roleRenter, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["owner@example.com"] = userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	a.profiles["board@example.com"] = userProfile{Email: "board@example.com", Role: roleBeirat, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	a.profiles["manager@example.com"] = userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	for _, item := range []residentIssue{
		{TenantSlug: "jhw22", AuthorEmail: "renter@example.com", AuthorName: "Renter", Category: "Frage", Title: "Renter private", Body: "Own unit.", LocationType: issueLocationUnit},
		{TenantSlug: "jhw22", AuthorEmail: "owner@example.com", AuthorName: "Owner", Category: "Frage", Title: "Owner private", Body: "Own unit.", LocationType: issueLocationUnit},
		{TenantSlug: "jhw22", AuthorEmail: "other@example.com", AuthorName: "Other", Category: "Reparatur", Title: "Common roof", Body: "Shared.", LocationType: issueLocationCommon},
		{TenantSlug: "jhw22", AuthorEmail: "other@example.com", AuthorName: "Other", Category: "Reparatur", Title: "Other private", Body: "Hidden.", LocationType: issueLocationUnit},
	} {
		if _, err := a.issueStore.Create(item); err != nil {
			t.Fatalf("Create issue %q: %v", item.Title, err)
		}
	}

	renter := authedRequest(t, a, "renter@example.com", "/app/anliegen", a.issues).Body.String()
	if !strings.Contains(renter, "Renter private") || strings.Contains(renter, "Common roof") || strings.Contains(renter, "Other private") {
		t.Fatalf("renter visibility wrong:\n%s", renter)
	}
	owner := authedRequest(t, a, "owner@example.com", "/app/anliegen", a.issues).Body.String()
	if !strings.Contains(owner, "Owner private") || !strings.Contains(owner, "Common roof") || strings.Contains(owner, "Other private") || strings.Contains(owner, "Renter private") {
		t.Fatalf("owner visibility wrong:\n%s", owner)
	}
	board := authedRequest(t, a, "board@example.com", "/app/anliegen", a.issues).Body.String()
	for _, want := range []string{"Renter private", "Owner private", "Common roof", "Other private"} {
		if !strings.Contains(board, want) {
			t.Fatalf("beirat view missing %q:\n%s", want, board)
		}
	}
	if strings.Contains(board, "Kommentar senden") || strings.Contains(board, "Anliegen senden") {
		t.Fatalf("beirat view must be read-only:\n%s", board)
	}

	boardComment := authedFormRequest(t, a, "board@example.com", "/app/anliegen/comment", url.Values{
		"id":   {a.issueStore.ListAuthor("jhw22", "owner@example.com")[0].ID},
		"body": {"Read-only should fail."},
	}, a.addIssueComment)
	if boardComment.Code != http.StatusForbidden {
		t.Fatalf("beirat comment status = %d, want 403", boardComment.Code)
	}
}

func TestIssueTriageBoardFiltersAndOpenCounts(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	old := time.Now().Add(-48 * time.Hour)
	for _, item := range []residentIssue{
		{TenantSlug: "jhw22", AuthorEmail: "resident@example.com", AuthorName: "Resident", Category: "Reparatur", Title: "Urgent repair", Body: "Broken.", LocationType: issueLocationCommon, Status: issueStatusProgress, Priority: issuePriorityUrgent, AssigneeEmail: "manager@example.com", CreatedAt: old, UpdatedAt: old},
		{TenantSlug: "jhw22", AuthorEmail: "resident@example.com", AuthorName: "Resident", Category: "Frage", Title: "Regular question", Body: "Question.", LocationType: issueLocationCommon, Status: issueStatusNew, Priority: issuePriorityNorm},
		{TenantSlug: "jhw22", AuthorEmail: "resident@example.com", AuthorName: "Resident", Category: "Vorschlag", Title: "Closed suggestion", Body: "Done.", LocationType: issueLocationCommon, Status: issueStatusDone, Priority: issuePriorityLow},
	} {
		if _, err := a.issueStore.Create(item); err != nil {
			t.Fatalf("Create issue %q: %v", item.Title, err)
		}
	}

	residentBoard := authedRequest(t, a, "resident@example.com", "/app/anliegen/board", a.issueBoard)
	if residentBoard.Code != http.StatusForbidden {
		t.Fatalf("resident board status = %d, want 403", residentBoard.Code)
	}
	board := authedRequest(t, a, "manager@example.com", "/app/anliegen/board?status=In+Bearbeitung&priority=Dringend&category=Reparatur&assignee=manager@example.com&sort=age", a.issueBoard)
	if board.Code != http.StatusOK {
		t.Fatalf("manager board status = %d", board.Code)
	}
	body := board.Body.String()
	for _, want := range []string{"Anliegen verwalten", "Urgent repair", `name="assignee"`, "Zurücksetzen"} {
		if !strings.Contains(body, want) {
			t.Fatalf("triage board should contain %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "Regular question") || strings.Contains(body, "Closed suggestion") {
		t.Fatalf("triage board filter leaked other issues:\n%s", body)
	}

	dashboard := authedRequest(t, a, "manager@example.com", "/app", a.portal).Body.String()
	if !strings.Contains(dashboard, "2 offen") || !strings.Contains(dashboard, "nav-badge") {
		t.Fatalf("dashboard should surface open issue count:\n%s", dashboard)
	}
}

func TestNotificationFrameworkDedupesRecipientsAndHonorsPrefs(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "actor@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["enabled@example.com"] = userProfile{Email: "enabled@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	a.profiles["muted@example.com"] = userProfile{Email: "muted@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	a.profiles["unsubscribed@example.com"] = userProfile{Email: "unsubscribed@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	mailer := &recordingMailer{}
	a.mailer = mailer
	if err := a.notificationPrefs.Set("muted@example.com", notificationPreferences{Email: map[string]bool{notificationEventIssue: false}}); err != nil {
		t.Fatalf("Set muted prefs: %v", err)
	}
	if err := a.notificationPrefs.Set("unsubscribed@example.com", notificationPreferences{Unsubscribed: true}); err != nil {
		t.Fatalf("Set unsubscribed prefs: %v", err)
	}

	a.notify(portalNotification{
		Event:      notificationEventIssue,
		Tenant:     a.tenants["jhw22"],
		Recipients: []string{"enabled@example.com", "ENABLED@example.com", "muted@example.com", "unsubscribed@example.com", "actor@example.com"},
		ActorEmail: "actor@example.com",
		Subject:    "Anliegen aktualisiert",
		Lines:      []string{"Status geändert."},
	})

	if len(mailer.notifications) != 1 || mailer.notifications[0].To != "enabled@example.com" || !strings.Contains(mailer.notifications[0].Body, "Status geändert.") {
		t.Fatalf("notifications = %+v", mailer.notifications)
	}
}

func TestPublishedAnnouncementNotifiesTenantRecipients(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	mailer := &recordingMailer{}
	a.mailer = mailer

	create := authedFormRequest(t, a, "manager@example.com", "/app/announcements", url.Values{
		"title":    {"Liftwartung"},
		"body":     {"Lift am Freitag außer Betrieb."},
		"category": {"Wartung"},
	}, a.createAnnouncement)
	if create.Code != http.StatusSeeOther {
		t.Fatalf("announcement create status = %d", create.Code)
	}
	if len(mailer.notifications) != 1 || mailer.notifications[0].To != "resident@example.com" || !strings.Contains(mailer.notifications[0].Subject, "Neuer Aushang") {
		t.Fatalf("announcement notifications = %+v", mailer.notifications)
	}
}

func minimalPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00,
	}
}

func TestAnnouncementArchiveFiltersSearchesAndIncludesPast(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	now := time.Now().Add(-2 * time.Hour)
	expiredAt := time.Now().Add(-time.Hour)
	_, _ = a.announcementStore.Create(announcement{TenantSlug: "jhw22", Title: "Liftwartung", Body: "Lift Freitag", Category: "Wartung", PublishedAt: now})
	_, _ = a.announcementStore.Create(announcement{TenantSlug: "jhw22", Title: "Hausfest", Body: "Sommertermin", Category: "Termin", PublishedAt: now})
	_, _ = a.announcementStore.Create(announcement{TenantSlug: "jhw22", Title: "Alter Hinweis", Body: "Vergangen", Category: "Info", PublishedAt: now.Add(-time.Hour), ExpiresAt: &expiredAt})
	_, _ = a.announcementStore.Create(announcement{TenantSlug: "jhw22", Title: "Geplant", Body: "Noch nicht sichtbar", Category: "Info", PublishedAt: time.Now().Add(time.Hour)})

	all := authedRequest(t, a, "resident@example.com", "/app/announcements", a.announcements)
	if all.Code != http.StatusOK {
		t.Fatalf("archive status = %d", all.Code)
	}
	body := all.Body.String()
	for _, want := range []string{"Liftwartung", "Hausfest", "Alter Hinweis", "Abgelaufen", "Suche", "Wartung"} {
		if !strings.Contains(body, want) {
			t.Fatalf("archive should contain %q", want)
		}
	}
	if strings.Contains(body, "Geplant") {
		t.Fatal("archive must not expose future announcements to residents")
	}

	filtered := authedRequest(t, a, "resident@example.com", "/app/announcements?category=Wartung&q=Lift", a.announcements)
	if filtered.Code != http.StatusOK {
		t.Fatalf("filtered archive status = %d", filtered.Code)
	}
	body = filtered.Body.String()
	if !strings.Contains(body, "Liftwartung") || strings.Contains(body, "Hausfest") || strings.Contains(body, "Alter Hinweis") {
		t.Fatalf("filtered archive body did not match expected search/category result:\n%s", body)
	}
}

func TestAnnouncementUnreadBadgeClearsAfterArchiveView(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	_, _ = a.announcementStore.Create(announcement{TenantSlug: "jhw22", Title: "Neue Wartung", Body: "Heute", Category: "Wartung", PublishedAt: time.Now().Add(-time.Hour)})

	before := authedRequest(t, a, "resident@example.com", "/app", a.portal)
	if before.Code != http.StatusOK {
		t.Fatalf("portal before status = %d", before.Code)
	}
	if body := before.Body.String(); !strings.Contains(body, "Neue Wartung") || !strings.Contains(body, `pill unread">neu`) || !strings.Contains(body, `<span class="nav-badge">1</span>`) {
		t.Fatalf("portal should show unread announcement and nav badge before archive view:\n%s", body)
	}

	archive := authedRequest(t, a, "resident@example.com", "/app/announcements", a.announcements)
	if archive.Code != http.StatusOK {
		t.Fatalf("archive status = %d", archive.Code)
	}
	if got := a.announcementReadStore.LastSeen("jhw22", "resident@example.com"); got.IsZero() {
		t.Fatal("archive view should mark announcements as seen")
	}

	after := authedRequest(t, a, "resident@example.com", "/app", a.portal)
	if after.Code != http.StatusOK {
		t.Fatalf("portal after status = %d", after.Code)
	}
	body := after.Body.String()
	if strings.Contains(body, `<span class="nav-badge">`) || strings.Contains(body, `pill unread">neu`) {
		t.Fatalf("portal should clear unread badges after archive view:\n%s", body)
	}
}

func TestAnnouncementRoutesGateWritesAndAllowResidentRead(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})

	read := authedRequest(t, a, "resident@example.com", "/app/announcements", a.announcements)
	if read.Code != http.StatusOK {
		t.Fatalf("resident announcement read status = %d", read.Code)
	}
	write := authedFormRequest(t, a, "resident@example.com", "/app/announcements", url.Values{
		"title": {"Resident post"},
		"body":  {"Nope"},
	}, a.createAnnouncement)
	if write.Code != http.StatusForbidden {
		t.Fatalf("resident create status = %d, want 403", write.Code)
	}
}

func TestAnnouncementCreateRejectsCrossOriginAndPersistsSameOrigin(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	values := url.Values{
		"title":        {"Liftwartung"},
		"body":         {"Der Lift ist am Freitag vormittags außer Betrieb."},
		"category":     {"Wartung"},
		"published_at": {"2026-07-06T12:30"},
		"pinned":       {"true"},
	}

	cross := authedFormRequestWithOrigin(t, a, "admin@example.com", "/app/announcements", values, "https://evil.example", a.createAnnouncement)
	if cross.Code != http.StatusForbidden {
		t.Fatalf("cross-origin create status = %d, want 403", cross.Code)
	}

	same := authedFormRequest(t, a, "admin@example.com", "/app/announcements", values, a.createAnnouncement)
	if same.Code != http.StatusSeeOther {
		t.Fatalf("same-origin create status = %d, want redirect", same.Code)
	}
	visible := a.announcementStore.Visible("jhw22", time.Date(2026, 7, 6, 13, 0, 0, 0, time.Local))
	if len(visible) != 1 || visible[0].Title != "Liftwartung" || !visible[0].Pinned {
		t.Fatalf("created visible announcement = %+v", visible)
	}
}

func newTestPortalApp(t *testing.T, profile userProfile) *app {
	t.Helper()
	tmpl, err := template.New("pages").Parse(pageTemplates)
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	profile.Email = normalizeEmail(profile.Email)
	profile.Role = normalizeRole(profile.Role)
	profile.Permissions = normalizePermissions(profile.Permissions)
	profile.TenantMemberships = normalizeTenantMemberships(profile.TenantMemberships)
	profile.Tenants = normalizeTenants(append(profile.Tenants, tenantMembershipSlugs(profile.TenantMemberships)...), "jhw22")
	authMethods, err := normalizeAuthMethods(profile.AuthMethods)
	if err != nil {
		t.Fatalf("normalize auth methods: %v", err)
	}
	profile.AuthMethods = authMethods
	parkingStore, err := newParkingStore("")
	if err != nil {
		t.Fatalf("parking store: %v", err)
	}
	announcementStore, err := newAnnouncementStore("")
	if err != nil {
		t.Fatalf("announcement store: %v", err)
	}
	announcementReadStore, err := newAnnouncementReadStore("")
	if err != nil {
		t.Fatalf("announcement read store: %v", err)
	}
	eventStore, err := newEventStore("")
	if err != nil {
		t.Fatalf("event store: %v", err)
	}
	notificationPrefStore, err := newNotificationPrefStore("")
	if err != nil {
		t.Fatalf("notification pref store: %v", err)
	}
	profileOverlayStore, err := newProfileOverlayStore("")
	if err != nil {
		t.Fatalf("profile overlay store: %v", err)
	}
	tenantOverrideStore, err := newTenantOverrideStore("")
	if err != nil {
		t.Fatalf("tenant override store: %v", err)
	}
	inviteStore, err := newInviteStore("")
	if err != nil {
		t.Fatalf("invite store: %v", err)
	}
	activityStore, err := newActivityStore("")
	if err != nil {
		t.Fatalf("activity store: %v", err)
	}
	unitStore, err := newUnitStore("")
	if err != nil {
		t.Fatalf("unit store: %v", err)
	}
	issueStore, err := newIssueStore("", "")
	if err != nil {
		t.Fatalf("issue store: %v", err)
	}
	auditStore, err := newAuditStore("")
	if err != nil {
		t.Fatalf("audit store: %v", err)
	}
	documentStore, err := newDocumentStore("", filepath.Join(t.TempDir(), "documents"))
	if err != nil {
		t.Fatalf("document store: %v", err)
	}
	voteStore, err := newVoteStore("")
	if err != nil {
		t.Fatalf("vote store: %v", err)
	}
	return &app{
		baseURL:       "http://localhost:8080",
		rootDomain:    "hausv.org",
		defaultTenant: "jhw22",
		tenants: map[string]tenantConfig{
			"jhw22": {Slug: "jhw22", Name: "WEG Portal", Address: "Janischhofweg 22", HeroImageURL: defaultTenantHeroImageURL, Host: "jhw22.hausv.org"},
		},
		profiles: map[string]userProfile{
			profile.Email: profile,
		},
		allowed:               map[string]struct{}{},
		admins:                map[string]struct{}{},
		sessionTTL:            time.Hour,
		sessions:              newSessionStore([]byte(strings.Repeat("s", 32))),
		oidc:                  &oidcLogin{},
		mailer:                smtpMailer{},
		templates:             tmpl,
		announcementStore:     announcementStore,
		announcementReadStore: announcementReadStore,
		eventStore:            eventStore,
		notificationPrefs:     notificationPrefStore,
		profileOverlays:       profileOverlayStore,
		tenantOverrides:       tenantOverrideStore,
		tenantHeroDir:         filepath.Join(t.TempDir(), "tenant-heroes"),
		inviteStore:           inviteStore,
		activityStore:         activityStore,
		unitStore:             unitStore,
		issueStore:            issueStore,
		auditStore:            auditStore,
		documentStore:         documentStore,
		voteStore:             voteStore,
		parkingStore:          parkingStore,
	}
}

func userRowForEmail(t *testing.T, rows []userRow, email string) userRow {
	t.Helper()
	email = normalizeEmail(email)
	for _, row := range rows {
		if normalizeEmail(row.Email) == email {
			return row
		}
	}
	t.Fatalf("row for %s not found in %+v", email, rows)
	return userRow{}
}

func authedRequest(t *testing.T, a *app, email string, path string, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	token, _, err := a.sessions.Put(email, "jhw22", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://jhw22.hausv.org"+path, nil)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

func authedPathValueRequest(t *testing.T, a *app, email string, path string, values map[string]string, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	token, _, err := a.sessions.Put(email, "jhw22", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://jhw22.hausv.org"+path, nil)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	for key, value := range values {
		req.SetPathValue(key, value)
	}
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

func authedFormRequest(t *testing.T, a *app, email string, path string, values url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	return authedFormRequestWithOrigin(t, a, email, path, values, "http://jhw22.hausv.org", handler)
}

func authedMultipartRequest(t *testing.T, a *app, email string, path string, fields map[string]string, filename string, fileBody []byte, handler http.HandlerFunc) *httptest.ResponseRecorder {
	return authedMultipartFileRequest(t, a, email, path, fields, "photo", filename, fileBody, handler)
}

func testMultipartHeader(t *testing.T, field string, filename string, fileBody []byte) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(fileBody); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("multipart close: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://jhw22.hausv.org/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(maxDocumentBytes); err != nil {
		t.Fatalf("ParseMultipartForm: %v", err)
	}
	files := req.MultipartForm.File[field]
	if len(files) != 1 {
		t.Fatalf("multipart files for %s = %d, want 1", field, len(files))
	}
	return files[0]
}

func authedMultipartFileRequest(t *testing.T, a *app, email string, path string, fields map[string]string, fileField string, filename string, fileBody []byte, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("WriteField %s: %v", key, err)
		}
	}
	if filename != "" {
		part, err := writer.CreateFormFile(fileField, filename)
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}
		if _, err := part.Write(fileBody); err != nil {
			t.Fatalf("write photo: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("multipart close: %v", err)
	}
	token, _, err := a.sessions.Put(email, "jhw22", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://jhw22.hausv.org"+path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", "http://jhw22.hausv.org")
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

func authedFormRequestWithOrigin(t *testing.T, a *app, email string, path string, values url.Values, origin string, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	token, _, err := a.sessions.Put(email, "jhw22", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://jhw22.hausv.org"+path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", origin)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

func TestParkingHourlyUsageAppliesHourlyAwattarPrices(t *testing.T) {
	base := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	energy := []parkingNumericSample{
		{At: base, Value: 100},
		{At: base.Add(2 * time.Hour), Value: 102},
	}
	prices := []parkingNumericSample{
		{At: base, Value: 0.20},
		{At: base.Add(time.Hour), Value: 0.40},
	}

	hours := calculateParkingHourlyUsage(energy, prices, 0.10, base.Add(3*time.Hour))
	if len(hours) != 2 {
		t.Fatalf("hour buckets = %d, want 2", len(hours))
	}
	assertClose(t, hours[0].KWh, 1)
	assertClose(t, hours[0].EnergyCost, 0.20)
	assertClose(t, hours[0].GridCost, 0.10)
	assertClose(t, hours[1].KWh, 1)
	assertClose(t, hours[1].EnergyCost, 0.40)
	assertClose(t, hours[1].GridCost, 0.10)
}

func TestParkingMonthsExposeCostsAndPaidFlag(t *testing.T) {
	base := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	data := parkingTenantData{
		Settings: parkingSettings{GridFeeEURPerKWh: 0.10},
		Months: map[string]parkingMonthState{
			"2026-06": {Paid: true},
		},
		EnergySamples: []parkingNumericSample{
			{At: base, Value: 100},
			{At: base.Add(2 * time.Hour), Value: 102},
		},
		PriceSamples: []parkingNumericSample{
			{At: base, Value: 0.20},
			{At: base.Add(time.Hour), Value: 0.40},
		},
	}

	months := calculateParkingMonths(data, base.Add(3*time.Hour), time.UTC)
	if len(months) != 1 {
		t.Fatalf("months = %d, want 1", len(months))
	}
	month := months[0]
	if month.Month != "2026-06" || !month.Paid {
		t.Fatalf("month state = %+v, want paid 2026-06", month)
	}
	if month.KWh != "2,00 kWh" || month.EnergyCost != "0,60 €" || month.GridCost != "0,20 €" || month.TotalCost != "0,80 €" {
		t.Fatalf("unexpected formatted costs: %+v", month)
	}
	if month.AverageAwattar != "0,300 €/kWh" || month.EffectivePrice != "0,400 €/kWh" {
		t.Fatalf("unexpected average prices: %+v", month)
	}
	if month.DetailPath != "/app/parking/month/2026-06" {
		t.Fatalf("detail path = %q", month.DetailPath)
	}
	if month.HourCount != 2 {
		t.Fatalf("hour count = %d, want 2", month.HourCount)
	}
}

func TestParkingTariffHistoryAppliesPerHourAndMonthlyBaseFee(t *testing.T) {
	base := time.Date(2026, 6, 30, 23, 0, 0, 0, time.UTC)
	data := parkingTenantData{
		Settings: parkingSettings{Tariffs: []parkingTariff{
			{EffectiveFrom: "2026-06-01", GridFeeEURPerKWh: 0.10, BaseFeeEUR: 1},
			{EffectiveFrom: "2026-07-01", GridFeeEURPerKWh: 0.20, BaseFeeEUR: 2},
		}},
		EnergySamples: []parkingNumericSample{
			{At: base, Value: 100},
			{At: base.Add(2 * time.Hour), Value: 102},
		},
		PriceSamples: []parkingNumericSample{
			{At: base, Value: 0.30},
		},
	}

	months := calculateParkingMonths(data, base.Add(3*time.Hour), time.UTC)
	if len(months) != 2 {
		t.Fatalf("months = %+v, want two tariff periods", months)
	}
	july := months[0]
	june := months[1]
	if july.Month != "2026-07" || july.GridCost != "0,20 €" || july.BaseFee != "2,00 €" || july.TotalCost != "2,50 €" || july.BaseFeeValue != 2 {
		t.Fatalf("july tariff result = %+v", july)
	}
	if june.Month != "2026-06" || june.GridCost != "0,10 €" || june.BaseFee != "1,00 €" || june.TotalCost != "1,40 €" || june.BaseFeeValue != 1 {
		t.Fatalf("june tariff result = %+v", june)
	}
	detail := calculateParkingMonthDetails(data, "2026-07", base.Add(3*time.Hour), time.UTC)
	if !detail.HasHours || detail.Summary.TotalCost != "2,50 €" || !strings.Contains(detail.Hours[0].GridCostTitle, "0,200000 €/kWh") {
		t.Fatalf("july detail tariff result = %+v", detail)
	}
}

func TestParkingPaymentMetadataAndOutstandingVisibility(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "parker@example.com", FirstName: "Pat", LastName: "Parker", Role: roleRenter, Tenants: []string{"jhw22"}, Permissions: []string{permissionParking}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("Add invite: %v", err)
	}
	base := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	if err := a.parkingStore.AppendReadings("jhw22", []parkingNumericSample{
		{At: base, Value: 100},
		{At: base.Add(2 * time.Hour), Value: 102},
	}, []parkingNumericSample{
		{At: base, Value: 0.20},
		{At: base.Add(time.Hour), Value: 0.40},
	}); err != nil {
		t.Fatalf("AppendReadings: %v", err)
	}

	parkerPage := authedRequest(t, a, "parker@example.com", "/app/parking", a.parking)
	if parkerPage.Code != http.StatusOK || !strings.Contains(parkerPage.Body.String(), "Offen 0,80 €") {
		t.Fatalf("parker outstanding page = %d\n%s", parkerPage.Code, parkerPage.Body.String())
	}
	accessPage := authedRequest(t, a, "admin@example.com", "/app/settings/parking-access", a.parkingAccessSettings)
	if accessPage.Code != http.StatusOK || !strings.Contains(accessPage.Body.String(), "parker@example.com") || !strings.Contains(accessPage.Body.String(), "0,80 €") {
		t.Fatalf("access outstanding page = %d\n%s", accessPage.Code, accessPage.Body.String())
	}

	save := authedFormRequest(t, a, "admin@example.com", "/app/parking/month", url.Values{
		"month":             {"2026-06"},
		"paid":              {"true"},
		"paid_at":           {"2026-07-05"},
		"payment_method":    {"Überweisung"},
		"payment_reference": {"ABC-123"},
	}, a.updateParkingMonth)
	if save.Code != http.StatusSeeOther {
		t.Fatalf("payment save status = %d, want redirect", save.Code)
	}
	state := a.parkingStore.TenantData("jhw22").Months["2026-06"]
	if !state.Paid || state.PaidBy != "admin@example.com" || state.PaymentMethod != "Überweisung" || state.PaymentReference != "ABC-123" || formatLocalDate(state.PaidAt) != "05.07.2026" {
		t.Fatalf("stored payment state = %+v", state)
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "jhw22", Action: auditActionParkingMonth, Limit: 10})
	if len(events) != 1 || events[0].Details["paid_by"] != "admin@example.com" || events[0].Details["payment_reference"] != "ABC-123" {
		t.Fatalf("payment audit events = %+v", events)
	}
	paidPage := authedRequest(t, a, "parker@example.com", "/app/parking", a.parking)
	body := paidPage.Body.String()
	for _, want := range []string{"BEZAHLT", "bezahlt am 05.07.2026", "Überweisung", "Ref. ABC-123"} {
		if !strings.Contains(body, want) {
			t.Fatalf("paid page missing %q:\n%s", want, body)
		}
	}
}

func TestParkingPaymentRemindersRespectPreferencesAndDedupe(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	for _, profile := range []userProfile{
		{Email: "parker@example.com", FirstName: "Pat", LastName: "Parker", Role: roleRenter, Tenants: []string{"jhw22"}, Permissions: []string{permissionParking}, AuthMethods: defaultAuthMethods()},
		{Email: "muted@example.com", FirstName: "Mute", LastName: "User", Role: roleRenter, Tenants: []string{"jhw22"}, Permissions: []string{permissionParking}, AuthMethods: defaultAuthMethods()},
	} {
		if _, err := a.inviteStore.Add(profile); err != nil {
			t.Fatalf("Add invite %s: %v", profile.Email, err)
		}
	}
	mailer := &recordingMailer{}
	a.mailer = mailer
	if err := a.notificationPrefs.Set("muted@example.com", notificationPreferences{Email: map[string]bool{notificationEventPayment: false}}); err != nil {
		t.Fatalf("Set prefs: %v", err)
	}
	base := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	if err := a.parkingStore.AppendReadings("jhw22", []parkingNumericSample{
		{At: base, Value: 100},
		{At: base.Add(2 * time.Hour), Value: 102},
	}, []parkingNumericSample{
		{At: base, Value: 0.20},
		{At: base.Add(time.Hour), Value: 0.40},
	}); err != nil {
		t.Fatalf("AppendReadings: %v", err)
	}

	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.Local)
	sent := a.sendParkingPaymentReminders(a.tenants["jhw22"], "manager@example.com", roleManager, now)
	if sent != 1 || len(mailer.notifications) != 1 {
		t.Fatalf("reminders sent=%d notifications=%+v", sent, mailer.notifications)
	}
	notification := mailer.notifications[0]
	if notification.To != "parker@example.com" || !strings.Contains(notification.Subject, "Zahlungserinnerung") || !strings.Contains(notification.Body, "Juni 2026") || !strings.Contains(notification.Body, "0,80 €") {
		t.Fatalf("payment reminder notification = %+v", notification)
	}
	state := a.parkingStore.TenantData("jhw22").Months["2026-06"]
	if _, ok := state.ReminderSentAt["parker@example.com"]; !ok {
		t.Fatalf("reminder timestamp not stored: %+v", state)
	}
	if _, ok := state.ReminderSentAt["muted@example.com"]; ok {
		t.Fatalf("muted recipient should not be marked reminded: %+v", state)
	}
	again := a.sendParkingPaymentReminders(a.tenants["jhw22"], "manager@example.com", roleManager, now.Add(time.Hour))
	if again != 0 || len(mailer.notifications) != 1 {
		t.Fatalf("duplicate reminders sent=%d notifications=%+v", again, mailer.notifications)
	}
}

func TestParkingMonthDetailsExposeHourlyRows(t *testing.T) {
	base := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	data := parkingTenantData{
		Settings: parkingSettings{GridFeeEURPerKWh: 0.10},
		EnergySamples: []parkingNumericSample{
			{At: base, Value: 100},
			{At: base.Add(2 * time.Hour), Value: 102},
		},
		PriceSamples: []parkingNumericSample{
			{At: base, Value: 0.20},
			{At: base.Add(time.Hour), Value: 0.40},
		},
	}

	detail := calculateParkingMonthDetails(data, "2026-06", base.Add(3*time.Hour), time.UTC)
	if !detail.HasHours || len(detail.Hours) != 2 {
		t.Fatalf("detail hours = %d, has=%v; want 2 true", len(detail.Hours), detail.HasHours)
	}
	if detail.Summary.TotalCost != "0,80 €" || detail.Summary.AverageAwattar != "0,300 €/kWh" {
		t.Fatalf("unexpected detail summary: %+v", detail.Summary)
	}
	if detail.Hours[0].AtLabel != "25.06. 10:00" || detail.Hours[0].TotalCost != "0,30 €" {
		t.Fatalf("unexpected first hour: %+v", detail.Hours[0])
	}
	if detail.Hours[0].KWhTitle != "Verbrauch: 1,000000 kWh" {
		t.Fatalf("unexpected kWh title: %q", detail.Hours[0].KWhTitle)
	}
	if detail.Hours[0].EnergyCostTitle != "Stromkosten: 0,200000 € = 1,000000 kWh × 0,200000 €/kWh" {
		t.Fatalf("unexpected energy title: %q", detail.Hours[0].EnergyCostTitle)
	}
	if detail.Hours[0].TotalCostTitle != "Summe: 0,300000 € = Strom 0,200000 € + Netzgebühr 0,100000 €" {
		t.Fatalf("unexpected total title: %q", detail.Hours[0].TotalCostTitle)
	}
	if detail.Hours[1].AtLabel != "25.06. 11:00" || detail.Hours[1].TotalCost != "0,50 €" {
		t.Fatalf("unexpected second hour: %+v", detail.Hours[1])
	}
}

func TestParkingStatementCSVAccessAndFigures(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "parker@example.com", FirstName: "Pat", LastName: "Parker", Role: roleRenter, Tenants: []string{"jhw22"}, Permissions: []string{permissionParking}, AuthMethods: defaultAuthMethods()})
	a.profiles["manager@example.com"] = userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	a.profiles["other@example.com"] = userProfile{Email: "other@example.com", Role: roleRenter, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	base := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	if err := a.parkingStore.AppendReadings("jhw22", []parkingNumericSample{
		{At: base, Value: 100},
		{At: base.Add(2 * time.Hour), Value: 102},
	}, []parkingNumericSample{
		{At: base, Value: 0.20},
		{At: base.Add(time.Hour), Value: 0.40},
	}); err != nil {
		t.Fatalf("AppendReadings: %v", err)
	}
	if err := a.parkingStore.SetMonthPaid("jhw22", "2026-06", true); err != nil {
		t.Fatalf("SetMonthPaid: %v", err)
	}

	resident := authedPathValueRequest(t, a, "parker@example.com", "/app/parking/export/2026", map[string]string{"year": "2026"}, a.parkingStatement)
	if resident.Code != http.StatusOK {
		t.Fatalf("resident export status = %d", resident.Code)
	}
	if got := resident.Header().Get("Content-Disposition"); !strings.Contains(got, "attachment") || !strings.Contains(got, "parkplatzabrechnung-2026-parker-example.com.csv") {
		t.Fatalf("content disposition = %q", got)
	}
	body := resident.Body.String()
	for _, want := range []string{"WEG Portal Parkplatzabrechnung", "Pat Parker", "parker@example.com", "Juni 2026", "2,00 kWh", "0,60 €", "0,20 €", "0,00 €", "0,80 €", "BEZAHLT", "Gesamt"} {
		if !strings.Contains(body, want) {
			t.Fatalf("statement CSV missing %q:\n%s", want, body)
		}
	}

	manager := authedPathValueRequest(t, a, "manager@example.com", "/app/parking/export/2026?user=parker@example.com", map[string]string{"year": "2026"}, a.parkingStatement)
	if manager.Code != http.StatusOK || !strings.Contains(manager.Body.String(), "parker@example.com") {
		t.Fatalf("manager export status/body = %d\n%s", manager.Code, manager.Body.String())
	}
	other := authedPathValueRequest(t, a, "other@example.com", "/app/parking/export/2026?user=parker@example.com", map[string]string{"year": "2026"}, a.parkingStatement)
	if other.Code != http.StatusNotFound {
		t.Fatalf("other resident export status = %d, want 404", other.Code)
	}
	noParking := authedPathValueRequest(t, a, "other@example.com", "/app/parking/export/2026", map[string]string{"year": "2026"}, a.parkingStatement)
	if noParking.Code != http.StatusNotFound {
		t.Fatalf("resident without parking export status = %d, want 404", noParking.Code)
	}
}

func TestParkingStorePersistsPaidFlagAndGridFee(t *testing.T) {
	path := filepath.Join(t.TempDir(), "parking.json")
	store, err := newParkingStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if err := store.SetGridFee("jhw22", 0.123); err != nil {
		t.Fatalf("set grid fee: %v", err)
	}
	if err := store.SetMonthPaid("jhw22", "2026-06", true); err != nil {
		t.Fatalf("set paid flag: %v", err)
	}

	loaded, err := newParkingStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	data := loaded.TenantData("jhw22")
	assertClose(t, data.Settings.GridFeeEURPerKWh, 0.123)
	if !data.Months["2026-06"].Paid {
		t.Fatal("paid flag was not persisted")
	}
}

func TestSamplesFromStatisticsUsesMillisecondsAndPreferredFields(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	state := 123.45
	sum := 456.78
	mean := 0.31
	stats := []haStatistic{
		{Start: json.RawMessage(strconvFormatInt(start.UnixMilli())), State: &state, Sum: &sum, Mean: &mean},
	}

	energy := samplesFromStatistics(stats, "state", "sum")
	if len(energy) != 1 {
		t.Fatalf("energy samples = %d, want 1", len(energy))
	}
	if !energy[0].At.Equal(start) {
		t.Fatalf("sample time = %s, want %s", energy[0].At, start)
	}
	assertClose(t, energy[0].Value, state)

	price := samplesFromStatistics(stats, "mean")
	if len(price) != 1 {
		t.Fatalf("price samples = %d, want 1", len(price))
	}
	assertClose(t, price[0].Value, mean)
}

func TestSamplesFromStatisticsFallsBackWhenColumnsAreOmitted(t *testing.T) {
	start := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	sum := 44.5
	stats := []haStatistic{
		{Start: json.RawMessage(`"` + start.Format(time.RFC3339) + `"`), Sum: &sum},
	}

	samples := samplesFromStatistics(stats, "state", "sum")
	if len(samples) != 1 {
		t.Fatalf("samples = %d, want 1", len(samples))
	}
	assertClose(t, samples[0].Value, sum)
}

func TestParseHistoryStartDefaultsToCurrentYear(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	start, err := parseHistoryStart("", now)
	if err != nil {
		t.Fatalf("parse history start: %v", err)
	}
	if start.Year() != 2026 || start.Month() != time.January || start.Day() != 1 {
		t.Fatalf("history start = %s, want first day of 2026", start)
	}
}

func assertClose(t *testing.T, got float64, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.000001 {
		t.Fatalf("got %.6f, want %.6f", got, want)
	}
}

func strconvFormatInt(value int64) string {
	return strconv.FormatInt(value, 10)
}

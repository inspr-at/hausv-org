package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/png"
	"math"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/auth"
	"github.com/inspr-at/hausv-org/internal/energy"
	storepkg "github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/version"
	"github.com/inspr-at/hausv-org/internal/web"
)

func testRepositories(a *app, tenantSlug string) requestRepositories {
	return a.repositoriesForTenant(testTenantRef(tenantSlug))
}

// Every tenant used to be handed the SAME hard-coded ULID. That was harmless
// while queries filtered on the slug and actively misleading once they filter on
// the identity: "demo", "other" and "haus-b" would collapse into one tenant, and
// every cross-tenant test in this package would quietly stop testing isolation
// while still passing.
//
// So each slug gets its own identity, derived from the slug so it is stable
// across runs and legible in a failure message. Most of this package's tests run
// against the in-memory stores, which key on the slug and never see the id — the
// distinctness matters for the handful that reach SQL, and for not lying to the
// next person who reads this helper.
func testTenantRef(tenantSlug string) storepkg.TenantRef {
	return storepkg.TenantRef{ID: testTenantID(tenantSlug), Slug: tenantSlug}
}

const testTenantAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

func testTenantID(tenantSlug string) string {
	sum := sha256.Sum256([]byte("hausv-test-tenant:" + tenantSlug))
	out := make([]byte, 26)
	out[0] = testTenantAlphabet[int(sum[0])%8]
	for i := 1; i < 26; i++ {
		out[i] = testTenantAlphabet[int(sum[i])%len(testTenantAlphabet)]
	}
	return string(out)
}

func addTestTenant(a *app, tenant tenantConfig) {
	if a.tenants == nil {
		a.tenants = map[string]tenantConfig{}
	}
	if a.tenantIdentities == nil {
		a.tenantIdentities = map[string]storepkg.TenantIdentity{}
	}
	a.tenants[tenant.Slug] = tenant
	a.tenantIdentities[tenant.Slug] = storepkg.TenantIdentity{ID: testTenantRef(tenant.Slug).ID, Slug: tenant.Slug, Name: tenant.Name}
}

type sentNotification struct {
	To      string
	Subject string
	Body    string
}

type sentMagicLink struct {
	To   string
	Link string
}

type recordingMailer struct {
	mu            sync.Mutex
	magicLinks    []sentMagicLink
	invites       []string
	notifications []sentNotification
}

func testUnitRepository(t testing.TB, a *app, tenantSlug string) unitRepository {
	t.Helper()
	repository, ok := storepkg.BindUnitRepository(a.unitStore, testTenantRef(tenantSlug))
	if !ok {
		t.Fatalf("bind unit repository for %q", tenantSlug)
	}
	return repository
}

func testUnitPaymentRepository(t testing.TB, a *app, tenantSlug string) unitPaymentRepository {
	t.Helper()
	repository, ok := storepkg.BindUnitPaymentStatusRepository(a.unitPaymentStore, testTenantRef(tenantSlug))
	if !ok {
		t.Fatalf("bind unit payment repository for %q", tenantSlug)
	}
	return repository
}

func testVoteRepository(t testing.TB, a *app, tenantSlug string) voteRepository {
	t.Helper()
	repository, ok := storepkg.BindVoteRepository(a.voteStore, testTenantRef(tenantSlug))
	if !ok {
		t.Fatalf("bind vote repository for %q", tenantSlug)
	}
	return repository
}

func testRequestRepositories(t testing.TB, a *app, tenantSlug string) requestRepositories {
	t.Helper()
	tenant, ok := a.tenants[tenantSlug]
	if !ok {
		t.Fatalf("tenant %q not configured", tenantSlug)
	}
	return a.repositoriesForTenant(testTenantRef(tenant.Slug))
}

func (m *recordingMailer) SendMagicLink(to string, link string, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.magicLinks = append(m.magicLinks, sentMagicLink{To: to, Link: link})
	return nil
}
func (m *recordingMailer) SendInvite(to string, _ string, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invites = append(m.invites, to)
	return nil
}
func (m *recordingMailer) Configured() bool { return true }

func (m *recordingMailer) Delivers() bool { return m.Configured() }
func (m *recordingMailer) SendNotification(to string, subject string, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = append(m.notifications, sentNotification{To: to, Subject: subject, Body: body})
	return nil
}

func (m *recordingMailer) recordedMagicLinks() []sentMagicLink {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]sentMagicLink(nil), m.magicLinks...)
}

func TestFaviconUsesStrippedLogo(t *testing.T) {
	rr := httptest.NewRecorder()
	favicon(rr, httptest.NewRequest(http.MethodGet, "http://www.hausv.org/favicon.svg", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("favicon status = %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "image/svg+xml; charset=utf-8" {
		t.Fatalf("favicon content type = %q", got)
	}
	body := rr.Body.String()
	for _, want := range []string{`<svg`, `viewBox="0 0 64 64"`, `stroke="#e7c574"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("favicon missing %q", want)
		}
	}
	for _, forbidden := range []string{`hausv.org`, `>22<`, `mark-frame`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("favicon should not contain %q", forbidden)
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
	p := userProfile{Email: "New.Person@example.com", FirstName: "New", LastName: "Person", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
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
	unitStore, err := newUnitStore(path)
	if err != nil {
		t.Fatalf("newUnitStore: %v", err)
	}
	repository, _ := storepkg.BindUnitRepository(unitStore, testTenantRef("demo"))
	if err := repository.SetUnits([]unit{
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
		{
			Label:    "Stellplatz 1",
			UnitType: "parking",
		},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}

	units := repository.List()
	if len(units) != 3 {
		t.Fatalf("ListTenant len = %d, want 3", len(units))
	}
	if units[0].ID != "stellplatz-1" || units[0].UnitType != unitTypeParking || units[0].BillableWeightPPM != 0 {
		t.Fatalf("parking unit not normalized as non-billable: %+v", units[0])
	}
	if units[1].ID != "top-1" || units[1].Label != "Top 1" || units[2].ID != "top-2" {
		t.Fatalf("units not normalized/sorted: %+v", units)
	}
	if got := units[2].OwnerEmails; len(got) != 1 || got[0] != "owner@example.com" {
		t.Fatalf("owners not normalized/deduped: %+v", got)
	}
	if units[2].MiteigentumsanteilPPM != 12345 {
		t.Fatalf("share = %d, want 12345", units[2].MiteigentumsanteilPPM)
	}
	if units[1].UnitType != unitTypeResidential || units[1].BillableWeightPPM != unitBillableFullPPM {
		t.Fatalf("legacy unit should default to residential/full billable: %+v", units[1])
	}

	memberships := repository.UnitsForEmail("OWNER@example.com")
	if len(memberships) != 1 || memberships[0].Unit.ID != "top-2" || memberships[0].Relation != roleOwner {
		t.Fatalf("owner memberships = %+v", memberships)
	}
	renterMemberships := repository.UnitsForEmail("renter@example.com")
	if len(renterMemberships) != 1 || renterMemberships[0].Relation != roleRenter {
		t.Fatalf("renter memberships = %+v", renterMemberships)
	}
	members := repository.MembersForUnit("top-2")
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
	reopenedRepository, _ := storepkg.BindUnitRepository(reopened, testTenantRef("demo"))
	if got := reopenedRepository.UnitCount(); got != 3 {
		t.Fatalf("reopened UnitCount = %d, want 3", got)
	}
	if got := reopenedRepository.BillableUnitWeight(); got != 2*unitBillableFullPPM {
		t.Fatalf("reopened BillableUnitWeight = %d, want %d", got, 2*unitBillableFullPPM)
	}
}

func TestVoteStoreCreateOpenCastClosePersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "votes.json")
	voteStore, err := newVoteStore(path)
	if err != nil {
		t.Fatalf("newVoteStore: %v", err)
	}
	repository, _ := storepkg.BindVoteRepository(voteStore, testTenantRef("demo"))
	created, err := repository.Create(ballot{
		TenantSlug:  "DEMO",
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
	if created.ID == "" || created.Status != ballotStatusDraft || created.TenantSlug != "demo" || len(created.Options) != 2 || created.CreatedBy != "manager@example.com" {
		t.Fatalf("created ballot = %+v", created)
	}
	opened, found, err := repository.Open(created.ID, time.Now())
	if err != nil || !found || opened.Status != ballotStatusOpen {
		t.Fatalf("Open = %+v found=%v err=%v", opened, found, err)
	}
	voted, found, err := repository.CastVote(created.ID, "Owner@Example.com", "Ja", 12345, time.Now())
	if err != nil || !found {
		t.Fatalf("CastVote first found=%v err=%v", found, err)
	}
	if vote := voted.Votes["owner@example.com"]; vote.Option != "Ja" || vote.Weight != 12345 {
		t.Fatalf("first vote = %+v", vote)
	}
	voted, found, err = repository.CastVote(created.ID, "owner@example.com", "Nein", 12345, time.Now())
	if err != nil || !found || len(voted.Votes) != 1 || voted.Votes["owner@example.com"].Option != "Nein" {
		t.Fatalf("mutable vote = %+v found=%v err=%v", voted.Votes, found, err)
	}
	closed, found, err := repository.Close(created.ID, time.Now())
	if err != nil || !found || closed.Status != ballotStatusClosed {
		t.Fatalf("Close = %+v found=%v err=%v", closed, found, err)
	}
	if _, _, err := repository.CastVote(created.ID, "owner@example.com", "Ja", 12345, time.Now()); err == nil {
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
	reopenedRepository, _ := storepkg.BindVoteRepository(reopened, testTenantRef("demo"))
	loaded, ok := reopenedRepository.Get(created.ID)
	if !ok || loaded.Status != ballotStatusClosed || loaded.Votes["owner@example.com"].Option != "Nein" {
		t.Fatalf("loaded ballot = %+v ok=%v", loaded, ok)
	}
}

func TestBallotVoteWeightUsesOwnerUnits(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["owner@example.com"] = userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["renter@example.com"] = userProfile{Email: "renter@example.com", Role: roleRenter, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{
		{ID: "top-1", TenantSlug: "demo", Label: "Top 1", MiteigentumsanteilPPM: 12345, OwnerEmails: []string{"owner@example.com"}, RenterEmails: []string{"renter@example.com"}},
		{ID: "top-2", TenantSlug: "demo", Label: "Top 2", MiteigentumsanteilPPM: 22222, OwnerEmails: []string{"owner@example.com"}},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	shareBallot, err := testVoteRepository(t, a, "demo").Create(ballot{
		TenantSlug: "demo",
		Title:      "Sanierung",
		Options:    []string{"Ja", "Nein"},
		Type:       ballotTypeCircular,
		Weighting:  ballotWeightingPerShare,
		CreatedBy:  "manager@example.com",
	})
	if err != nil {
		t.Fatalf("Create share ballot: %v", err)
	}
	if _, _, err := testVoteRepository(t, a, "demo").Open(shareBallot.ID, time.Now()); err != nil {
		t.Fatalf("Open share ballot: %v", err)
	}
	updated, found, err := a.castBallotVote(testRequestRepositories(t, a, "demo"), "demo", "owner@example.com", shareBallot.ID, "Ja", time.Now())
	if err != nil || !found {
		t.Fatalf("owner cast share vote found=%v err=%v", found, err)
	}
	if got := updated.Votes["owner@example.com"].Weight; got != 34567 {
		t.Fatalf("owner share vote weight = %d, want 34567", got)
	}
	if _, _, err := a.castBallotVote(testRequestRepositories(t, a, "demo"), "demo", "renter@example.com", shareBallot.ID, "Ja", time.Now()); err == nil {
		t.Fatal("renter should not be eligible for owner ballot")
	}
	headBallot, err := testVoteRepository(t, a, "demo").Create(ballot{
		TenantSlug: "demo",
		Title:      "Pro Kopf",
		Options:    []string{"Ja", "Nein"},
		Type:       ballotTypeMeeting,
		Weighting:  ballotWeightingPerHead,
		CreatedBy:  "manager@example.com",
	})
	if err != nil {
		t.Fatalf("Create head ballot: %v", err)
	}
	if _, _, err := testVoteRepository(t, a, "demo").Open(headBallot.ID, time.Now()); err != nil {
		t.Fatalf("Open head ballot: %v", err)
	}
	updated, found, err = a.castBallotVote(testRequestRepositories(t, a, "demo"), "demo", "owner@example.com", headBallot.ID, "Nein", time.Now())
	if err != nil || !found || updated.Votes["owner@example.com"].Weight != 1 {
		t.Fatalf("owner cast head vote = %+v found=%v err=%v", updated.Votes["owner@example.com"], found, err)
	}
}

func TestBallotsPageOwnerVotingAndReadOnlyPersonas(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["renter@example.com"] = userProfile{Email: "renter@example.com", Role: roleRenter, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["beirat@example.com"] = userProfile{Email: "beirat@example.com", Role: roleBeirat, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{
		{ID: "top-1", TenantSlug: "demo", Label: "Top 1", MiteigentumsanteilPPM: 400000, OwnerEmails: []string{"owner@example.com"}, RenterEmails: []string{"renter@example.com"}},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	created, err := testVoteRepository(t, a, "demo").Create(ballot{
		TenantSlug: "demo",
		Title:      "Dachsanierung",
		Options:    []string{"Ja", "Nein"},
		Type:       ballotTypeCircular,
		Weighting:  ballotWeightingPerShare,
		CreatedBy:  "manager@example.com",
	})
	if err != nil {
		t.Fatalf("Create ballot: %v", err)
	}
	if _, _, err := testVoteRepository(t, a, "demo").Open(created.ID, time.Now()); err != nil {
		t.Fatalf("Open ballot: %v", err)
	}

	ownerPage := authedRequest(t, a, "owner@example.com", "/demo/app/abstimmungen")
	if ownerPage.Code != http.StatusOK {
		t.Fatalf("owner ballots status = %d", ownerPage.Code)
	}
	ownerBody := ownerPage.Body.String()
	for _, want := range []string{`href="/demo/app/abstimmungen"`, "nav-item active", "Dachsanierung", `name="option"`, "Ihre Stimme zählt: " + formatBallotWeight(400000), "Ihre Stimme ist gefragt", "Details zur Abstimmung"} {
		if !strings.Contains(ownerBody, want) {
			t.Fatalf("owner ballots page missing %q:\n%s", want, ownerBody)
		}
	}
	if strings.Contains(ownerBody, `class="kicker">Verwaltung`) {
		t.Fatalf("owner ballots page should not render an irrelevant administration panel:\n%s", ownerBody)
	}
	if strings.Contains(ownerBody, `class="nav-item disabled"`) && strings.Contains(ownerBody, "Abstimmungen") {
		t.Fatalf("Abstimmungen nav item must be a live link:\n%s", ownerBody)
	}

	vote := authedFormRequest(t, a, "owner@example.com", "/demo/app/abstimmungen", url.Values{
		"ballot_id": {created.ID},
		"option":    {"Ja"},
	})
	if vote.Code != http.StatusSeeOther {
		t.Fatalf("owner vote status = %d, want redirect", vote.Code)
	}
	if location := vote.Header().Get("Location"); location != "/demo/app/abstimmungen?vote=cast#ballot-"+created.ID {
		t.Fatalf("owner vote redirect = %q, want ballot anchor", location)
	}
	stored, _ := testVoteRepository(t, a, "demo").Get(created.ID)
	if got := stored.Votes["owner@example.com"].Weight; got != 400000 {
		t.Fatalf("owner vote weight = %d, want 400000", got)
	}
	voteEvents := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionVoteCast, Limit: 10})
	if len(voteEvents) != 1 || voteEvents[0].TargetID != created.ID || voteEvents[0].Details["weight"] == "" || voteEvents[0].Details["cast_at"] == "" {
		t.Fatalf("vote audit event should record timestamp/weight without option: %+v", voteEvents)
	}
	if _, ok := voteEvents[0].Details["option"]; ok {
		t.Fatalf("vote audit event should record timestamp/weight without option: %+v", voteEvents)
	}

	for _, persona := range []struct {
		email string
		want  string
	}{
		{"renter@example.com", "Nur Eigentümer können abstimmen."},
		{"beirat@example.com", "Beirat: lesende Übersicht."},
	} {
		page := authedRequest(t, a, persona.email, "/demo/app/abstimmungen")
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
		post := authedFormRequest(t, a, persona.email, "/demo/app/abstimmungen", url.Values{
			"ballot_id": {created.ID},
			"option":    {"Nein"},
		})
		if post.Code != http.StatusForbidden {
			t.Fatalf("%s vote status = %d, want 403", persona.email, post.Code)
		}
	}

	board := authedRequest(t, a, "beirat@example.com", "/demo/app/abstimmungen").Body.String()
	if !strings.Contains(board, "Teilnahme") || !strings.Contains(board, "100,0 %") || !strings.Contains(board, formatBallotResultWeight(ballotWeightingPerShare, 400000)+" · 1 Stimmen") {
		t.Fatalf("beirat oversight should show weighted aggregate:\n%s", board)
	}
}

func TestBallotCreateShowsAttachmentPreview(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	create := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/abstimmungen", map[string]string{
		"title":        "Dachsanierung",
		"description":  "Unterlagen beachten.",
		"options_text": "Ja\nNein",
		"type":         ballotTypeCircular,
		"weighting":    ballotWeightingPerShare,
	}, "attachments", "angebot.pdf", []byte("%PDF-1.4\n% angebot\n"))
	if create.Code != http.StatusSeeOther {
		t.Fatalf("ballot create status = %d, want redirect", create.Code)
	}
	ballots := testVoteRepository(t, a, "demo").List()
	if len(ballots) != 1 {
		t.Fatalf("ballots = %+v", ballots)
	}
	attachments := attachmentRepositoryForTest(a, "demo").ListEntity("ballot", ballots[0].ID)
	if len(attachments) != 1 || attachments[0].ContentType != "application/pdf" {
		t.Fatalf("ballot attachments = %+v", attachments)
	}
	page := authedRequest(t, a, "manager@example.com", "/demo/app/abstimmungen")
	body := page.Body.String()
	if !strings.Contains(body, "angebot.pdf") || !strings.Contains(body, "Dachsanierung") {
		t.Fatalf("ballots page should render attachment:\n%s", body)
	}
	for _, want := range []string{"Vor Veröffentlichung prüfen", "Nächster Schritt: Abstimmung öffnen", "Details zur Abstimmung", "Entwurf anlegen"} {
		if !strings.Contains(body, want) {
			t.Fatalf("manager draft page missing %q:\n%s", want, body)
		}
	}
	if got := strings.Count(body, `data-dialog="ballot-create"`); got != 1 {
		t.Fatalf("manager draft page should have one create trigger, got %d", got)
	}
}

func TestBallotTallyQuorumAutoCloseAndProtocol(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "owner1@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["owner2@example.com"] = userProfile{Email: "owner2@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["beirat@example.com"] = userProfile{Email: "beirat@example.com", Role: roleBeirat, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["renter@example.com"] = userProfile{Email: "renter@example.com", Role: roleRenter, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{
		{ID: "top-1", TenantSlug: "demo", Label: "Top 1", MiteigentumsanteilPPM: 400000, OwnerEmails: []string{"owner1@example.com"}, RenterEmails: []string{"renter@example.com"}},
		{ID: "top-2", TenantSlug: "demo", Label: "Top 2", MiteigentumsanteilPPM: 600000, OwnerEmails: []string{"owner2@example.com"}},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	deadline := time.Now().Add(time.Hour)
	created, err := testVoteRepository(t, a, "demo").Create(ballot{
		TenantSlug: "demo",
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
	if _, _, err := testVoteRepository(t, a, "demo").Open(created.ID, time.Now()); err != nil {
		t.Fatalf("Open ballot: %v", err)
	}
	if _, _, err := a.castBallotVote(testRequestRepositories(t, a, "demo"), "demo", "owner1@example.com", created.ID, "Ja", time.Now()); err != nil {
		t.Fatalf("owner1 vote: %v", err)
	}
	item, _ := testVoteRepository(t, a, "demo").Get(created.ID)
	view := a.ballotViewForActor(testRequestRepositories(t, a, "demo"), testTenantRef("demo"), "beirat@example.com", roleBeirat, item, time.Now(), true)
	if view.TotalWeightLabel != formatBallotResultWeight(ballotWeightingPerShare, 400000) || view.EligibleWeightLabel != formatBallotResultWeight(ballotWeightingPerShare, 1000000) || view.Participation != "40,0 %" || view.QuorumStatus != "Quorum offen" || view.WinnerLabel != "Ja" {
		t.Fatalf("single-vote tally = %+v", view)
	}
	if _, _, err := a.castBallotVote(testRequestRepositories(t, a, "demo"), "demo", "owner2@example.com", created.ID, "Nein", time.Now()); err != nil {
		t.Fatalf("owner2 vote: %v", err)
	}
	item, _ = testVoteRepository(t, a, "demo").Get(created.ID)
	view = a.ballotViewForActor(testRequestRepositories(t, a, "demo"), testTenantRef("demo"), "beirat@example.com", roleBeirat, item, time.Now(), true)
	if view.TotalWeightLabel != formatBallotResultWeight(ballotWeightingPerShare, 1000000) || view.Participation != "100,0 %" || view.QuorumStatus != "Quorum erreicht" || view.WinnerLabel != "Nein" {
		t.Fatalf("full tally = %+v", view)
	}

	openProtocol := authedRequest(t, a, "owner1@example.com", "/demo/app/abstimmungen/"+created.ID+"/protokoll")
	if openProtocol.Code != http.StatusConflict {
		t.Fatalf("open protocol status = %d, want 409", openProtocol.Code)
	}
	closed, err := testVoteRepository(t, a, "demo").CloseExpired(deadline.Add(time.Minute))
	if err != nil || len(closed) != 1 || closed[0].Status != ballotStatusClosed {
		t.Fatalf("CloseExpiredTenant = %+v err=%v", closed, err)
	}
	protocol := authedRequest(t, a, "owner1@example.com", "/demo/app/abstimmungen/"+created.ID+"/protokoll")
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
	renterProtocol := authedRequest(t, a, "renter@example.com", "/demo/app/abstimmungen/"+created.ID+"/protokoll")
	if renterProtocol.Code != http.StatusForbidden {
		t.Fatalf("renter protocol status = %d, want 403", renterProtocol.Code)
	}
}

func TestBallotVoteAfterDeadlineAutoCloses(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{
		{ID: "top-1", TenantSlug: "demo", Label: "Top 1", MiteigentumsanteilPPM: 1000000, OwnerEmails: []string{"owner@example.com"}},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	now := time.Now()
	created, err := testVoteRepository(t, a, "demo").Create(ballot{
		TenantSlug: "demo",
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
	if _, _, err := testVoteRepository(t, a, "demo").Open(created.ID, now.Add(-2*time.Hour)); err != nil {
		t.Fatalf("Open ballot: %v", err)
	}
	if _, _, err := a.castBallotVote(testRequestRepositories(t, a, "demo"), "demo", "owner@example.com", created.ID, "Ja", now); err == nil {
		t.Fatal("vote after deadline should fail")
	}
	closed, _ := testVoteRepository(t, a, "demo").Get(created.ID)
	if closed.Status != ballotStatusClosed {
		t.Fatalf("deadline vote should auto-close ballot: %+v", closed)
	}
	page := authedRequest(t, a, "owner@example.com", "/demo/app/abstimmungen").Body.String()
	if strings.Contains(page, `name="option"`) || !strings.Contains(page, `aria-label="Abstimmungsergebnis"`) || strings.Contains(page, `class="vote-option vote-option-static"`) {
		t.Fatalf("closed ballot should render read-only:\n%s", page)
	}
}

func TestBallotReminderEmailsOnlyNonVotersAndHonorsPrefs(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["owner1@example.com"] = userProfile{Email: "owner1@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["owner2@example.com"] = userProfile{Email: "owner2@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["owner3@example.com"] = userProfile{Email: "owner3@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{
		{ID: "top-1", TenantSlug: "demo", Label: "Top 1", MiteigentumsanteilPPM: 300000, OwnerEmails: []string{"owner1@example.com"}},
		{ID: "top-2", TenantSlug: "demo", Label: "Top 2", MiteigentumsanteilPPM: 300000, OwnerEmails: []string{"owner2@example.com"}},
		{ID: "top-3", TenantSlug: "demo", Label: "Top 3", MiteigentumsanteilPPM: 400000, OwnerEmails: []string{"owner3@example.com"}},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	mailer := &recordingMailer{}
	a.mailer = mailer
	if err := a.notificationPrefs.Set("owner3@example.com", notificationPreferences{Email: map[string]bool{notificationEventVote: false}}); err != nil {
		t.Fatalf("Set owner3 prefs: %v", err)
	}
	now := time.Now()
	created, err := testVoteRepository(t, a, "demo").Create(ballot{
		TenantSlug:            "demo",
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
	if _, _, err := testVoteRepository(t, a, "demo").Open(created.ID, now); err != nil {
		t.Fatalf("Open ballot: %v", err)
	}
	if _, _, err := a.castBallotVote(testRequestRepositories(t, a, "demo"), "demo", "owner1@example.com", created.ID, "Ja", now); err != nil {
		t.Fatalf("owner1 vote: %v", err)
	}

	if sent := a.sendDueBallotReminders(now.Add(30 * time.Minute)); sent != 1 {
		t.Fatalf("sendDueBallotReminders sent = %d, want 1", sent)
	}
	if len(mailer.notifications) != 1 || mailer.notifications[0].To != "owner2@example.com" || !strings.Contains(mailer.notifications[0].Subject, "Reminder") || !strings.Contains(mailer.notifications[0].Body, "/demo/app/abstimmungen#ballot-"+created.ID) {
		t.Fatalf("reminder notifications = %+v", mailer.notifications)
	}
	updated, _ := testVoteRepository(t, a, "demo").Get(created.ID)
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
	events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionVoteReminder, Limit: 10})
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
	documents := documentRepositoryForStorageTest(store, "demo")
	created, err := documents.Create(documentRecord{
		TenantSlug: "DEMO",
		Title:      "Hausordnung",
		Category:   documentCategoryRules,
		Visibility: documentVisibilityAllResidents,
		UploadedBy: "Manager@Example.com",
	}, testMultipartHeader(t, "document", "../Hausordnung.pdf", []byte("%PDF-1.4\n% weg portal test\n")), time.Date(2026, 7, 7, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" || created.TenantSlug != "demo" || created.Filename != "Hausordnung.pdf" || created.StoredFilename == created.Filename {
		t.Fatalf("created document not normalized/private: %+v", created)
	}
	if created.ContentType != "application/pdf" || created.UploadedBy != "manager@example.com" || created.Size <= 0 {
		t.Fatalf("created document metadata = %+v", created)
	}
	storedPath, ok := documents.FilePath(created)
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
	reopenedDocuments := documentRepositoryForStorageTest(reopened, "demo")
	docs := reopenedDocuments.List()
	if len(docs) != 1 || docs[0].Title != "Hausordnung" {
		t.Fatalf("reopened docs = %+v", docs)
	}
	if _, err := documents.Create(documentRecord{
		TenantSlug: "demo",
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
	documents := documentRepositoryForStorageTest(store, "demo")
	created, err := documents.Create(documentRecord{
		TenantSlug: "demo",
		Title:      "Hausordnung",
		Category:   documentCategoryRules,
		Visibility: documentVisibilityAllResidents,
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "hausordnung-v1.pdf", []byte("%PDF-1.4\nv1\n")), time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	replacement, replaced, err := documents.Replace(created.ID, "manager@example.com", testMultipartHeader(t, "document", "hausordnung-v2.pdf", []byte("%PDF-1.4\nv2\n")), time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}
	if !replacement.Current || replacement.Version != 2 || replacement.SeriesID != created.SeriesID || replacement.SupersedesID != created.ID {
		t.Fatalf("replacement metadata = %+v", replacement)
	}
	if replaced.Current || replaced.ReplacedByID != replacement.ID {
		t.Fatalf("replaced metadata = %+v", replaced)
	}
	current := documents.ListCurrent()
	if len(current) != 1 || current[0].ID != replacement.ID {
		t.Fatalf("current docs = %+v", current)
	}
	versions := documents.Versions(created.SeriesID)
	if len(versions) != 2 || versions[0].Version != 2 || versions[1].Version != 1 {
		t.Fatalf("versions = %+v", versions)
	}
	if oldPath, ok := documents.FilePath(replaced); !ok {
		t.Fatal("old version path missing")
	} else if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("old version file missing: %v", err)
	}
	if newPath, ok := documents.FilePath(replacement); !ok {
		t.Fatal("new version path missing")
	} else if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("new version file missing: %v", err)
	}
}

func TestIssueStoreCreateListPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issues.json")
	issueStorage, err := newIssueStore(path, filepath.Join(t.TempDir(), "issue-attachments"))
	if err != nil {
		t.Fatalf("newIssueStore: %v", err)
	}
	issues, _ := storepkg.BindIssueRepository(issueStorage, testTenantRef("demo"))
	created, err := issues.Create(residentIssue{
		TenantSlug:     "DEMO",
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
	reopenedIssues, _ := storepkg.BindIssueRepository(reopened, testTenantRef("demo"))
	byAuthor := reopenedIssues.ListAuthor("resident@example.com")
	if len(byAuthor) != 1 || byAuthor[0].Title != "Licht im Stiegenhaus" || byAuthor[0].TenantSlug != "demo" {
		t.Fatalf("ListAuthor = %+v", byAuthor)
	}
}

func TestDirectoryProfileEnvWinsAndInviteGrantsLogin(t *testing.T) {
	store, err := newInviteStore("")
	if err != nil {
		t.Fatalf("newInviteStore: %v", err)
	}
	// an invite trying to claim admin for an email that is an env resident
	if _, err := store.Add(userProfile{Email: "resident@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	// a brand-new invited-only user
	if _, err := store.Add(userProfile{Email: "invited@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	a := &app{
		defaultTenant: "demo",
		profiles: map[string]userProfile{
			"resident@example.com": {Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()},
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
	if !a.isAllowed("invited@example.com", "demo") {
		t.Fatal("invited user should be allowed for demo")
	}
	// an unknown email is still denied
	if a.isAllowed("stranger@example.com", "demo") {
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
		if _, err := store.Add(userProfile{Email: email, FirstName: first, Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}); err != nil {
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
	ok, err := store.Update("old@example.com", userProfile{Email: "New@example.com", FirstName: "New", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
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
	_, _ = inv.Add(userProfile{Email: "loggedin@example.com", Role: roleResident, Status: "Eingeladen", Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	_, _ = inv.Add(userProfile{Email: "never@example.com", Role: roleResident, Status: "Eingeladen", Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a := &app{defaultTenant: "demo", profiles: map[string]userProfile{}, inviteStore: inv, activityStore: act}

	byEmail := map[string]userRow{}
	for _, r := range a.userRows(testTenantRef("demo")) {
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
		_, _ = w.Write([]byte(`{"service":"hausv-org","status":"ok"}`))
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
	origVersion := version.Version
	origCommit := version.Commit
	t.Cleanup(func() {
		version.Version = origVersion
		version.Commit = origCommit
	})

	version.Version = "v0.1.0"
	version.Commit = "abc1234"

	if got := version.BuildLabel(); got != "0.1.0 (abc1234)" {
		t.Fatalf("build label = %q, want semver and commit", got)
	}

	version.Version = ""
	version.Commit = ""
	if got := version.BuildLabel(); got != "dev (dev)" {
		t.Fatalf("empty build identity should stay visibly non-production, got %q", got)
	}
}

func TestReleaseNotesMentionWohneinheitenPricing(t *testing.T) {
	notes := version.Notes()
	if len(notes) == 0 {
		t.Fatal("releaseNotes empty")
	}
	versionFile, err := os.ReadFile("../../VERSION")
	if err != nil {
		t.Fatalf("read VERSION: %v", err)
	}
	if want := strings.TrimSpace(string(versionFile)); notes[0].Version != want {
		t.Fatalf("newest release note = %q, want VERSION %q", notes[0].Version, want)
	}
	joined := ""
	for _, note := range notes {
		joined += note.Headline + " " + note.Intro + " "
		for _, item := range note.Items {
			joined += item.Label + " " + item.Text + " "
		}
	}
	if !strings.Contains(joined, "Wohneinheiten") {
		t.Fatal("release notes should mention Wohneinheiten pricing")
	}
	// Since 0.67.0 the public model is: open-source core stays free, managed
	// service via HAUSV Professional at 500/900 € base plus 1 € per unit (HAUSV-435).
	if !strings.Contains(joined, "500 €") || !strings.Contains(joined, "900 €") || !strings.Contains(joined, "HAUSV Professional") {
		t.Fatal("release notes should mention the HAUSV Professional service-fee model (500 €/900 € base)")
	}
	if strings.Contains(joined, "1 € pro Haus") || strings.Contains(joined, "pro Hausadresse") {
		t.Fatalf("release notes contain old house-based pricing wording: %s", joined)
	}
}

func TestHandoverCreateExportsAndFilesProtocol(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	mailer := &recordingMailer{}
	a.mailer = mailer
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{
		{ID: "einheit-12", Label: "Einheit 12", OwnerEmails: []string{"owner@example.com"}},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}

	create := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/uebergaben", map[string]string{
		"title":          "Übergabe Einheit 12",
		"unit_id":        "einheit-12",
		"handover_type":  "Nutzerwechsel",
		"scheduled_at":   "2026-07-08T14:30",
		"outgoing_name":  "Alte Nutzerin",
		"outgoing_email": "alt@example.com",
		"incoming_name":  "Neue Nutzerin",
		"incoming_email": "neu@example.com",
		"rooms_text":     "Wohnzimmer | gut | keine Mängel\nBad | sauber | Fuge prüfen",
		"meters_text":    "Strom | 12345,6 | kWh",
		"keys_text":      "Wohnung | 3",
		"notes":          "Fenstergriff im Bad nachziehen.",
	}, "attachments", "bad.png", minimalPNG())
	if create.Code != http.StatusSeeOther {
		t.Fatalf("handover create status = %d, want redirect", create.Code)
	}

	items := testRepositories(a, "demo").handovers.List()
	if len(items) != 1 {
		t.Fatalf("handovers = %+v", items)
	}
	item := items[0]
	if item.UnitID != "einheit-12" || len(item.Rooms) != 2 || len(item.Meters) != 1 || len(item.Keys) != 1 {
		t.Fatalf("handover content = %+v", item)
	}
	if got := handoverStatus(item); got != handoverStatusPending {
		t.Fatalf("handover status = %q, want pending", got)
	}
	if len(item.Confirmations) != 2 || item.Confirmations[0].TokenHash == "" || item.Confirmations[0].TokenHash == "alt@example.com" {
		t.Fatalf("confirmations not tokenized: %+v", item.Confirmations)
	}
	if len(mailer.notifications) != 2 || !strings.Contains(mailer.notifications[0].Body, "/handover/") || !strings.Contains(mailer.notifications[1].Body, "/handover/") {
		t.Fatalf("handover notifications = %+v", mailer.notifications)
	}

	attachments := attachmentRepositoryForTest(a, "demo").ListEntity("handover", item.ID)
	if len(attachments) != 1 || attachments[0].ContentType != "image/png" || attachments[0].ThumbFilename == "" {
		t.Fatalf("handover attachments = %+v", attachments)
	}
	page := authedRequest(t, a, "manager@example.com", "/demo/app/uebergaben")
	for _, want := range []string{
		"Übergabe Einheit 12",
		"bad.png",
		"data-lightbox-src",
		"PDF exportieren",
		"ohne Kautions- oder Schadenabrechnung",
		`name="redirect" value="/demo/app/uebergaben#handover-` + item.ID + `"`,
	} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("handover page missing %q:\n%s", want, page.Body.String())
		}
	}
	templPage := authedRequest(t, a, "manager@example.com", "/demo/app/uebergaben")
	for _, want := range []string{
		"data-templ-handovers",
		"Übergabe Einheit 12",
		"bad.png",
		"data-lightbox-src",
		"PDF exportieren",
		`action="/demo/app/uebergaben/attachments"`,
		`name="redirect" value="/demo/app/uebergaben#handover-` + item.ID + `"`,
	} {
		if !strings.Contains(templPage.Body.String(), want) {
			t.Fatalf("templ handover page missing %q:\n%s", want, templPage.Body.String())
		}
	}

	attachmentPath, _, _, ok := attachmentRepositoryForTest(a, "demo").FilePath(attachments[0], "")
	if !ok {
		t.Fatal("handover attachment file path missing")
	}
	deleted := authedFormRequest(t, a, "manager@example.com", "/demo/app/attachments/delete", url.Values{
		"id":       {attachments[0].ID},
		"redirect": {"/demo/app/uebergaben#handover-" + item.ID},
	})
	if deleted.Code != http.StatusSeeOther || deleted.Header().Get("Location") != "/demo/app/uebergaben#handover-"+item.ID {
		t.Fatalf("handover attachment delete = %d %q", deleted.Code, deleted.Header().Get("Location"))
	}
	if got := attachmentRepositoryForTest(a, "demo").ListEntity("handover", item.ID); len(got) != 0 {
		t.Fatalf("handover attachments after delete = %+v, want none", got)
	}
	if _, err := os.Stat(attachmentPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("deleted handover attachment stat err = %v, want not exist", err)
	}

	tokens := []string{}
	for _, notification := range mailer.notifications {
		for _, line := range strings.Split(notification.Body, "\n") {
			parsed, err := url.Parse(strings.TrimSpace(line))
			if err == nil && strings.HasPrefix(parsed.Path, "/demo/handover/") {
				tokens = append(tokens, strings.TrimPrefix(parsed.Path, "/demo/handover/"))
			}
		}
	}
	if len(tokens) != 2 {
		t.Fatalf("handover confirmation tokens = %q, want two", tokens)
	}
	addedFiles := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/uebergaben/attachments", map[string]string{
		"id": item.ID,
	}, "attachments", "nachtrag.png", minimalPNG())
	if addedFiles.Code != http.StatusSeeOther || !strings.Contains(addedFiles.Header().Get("Location"), "handover=attachments") {
		t.Fatalf("handover add files = %d %q", addedFiles.Code, addedFiles.Header().Get("Location"))
	}
	addedAttachments := attachmentRepositoryForTest(a, "demo").ListEntity("handover", item.ID)
	if len(addedAttachments) != 1 {
		t.Fatalf("handover attachments after add = %+v, want one", addedAttachments)
	}
	publicReview := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/handover/"+tokens[0], nil)
	publicReview.SetPathValue("token", tokens[0])
	publicReviewRR := httptest.NewRecorder()
	a.handoverConfirmPage(publicReviewRR, publicReview)
	publicAttachmentURL := "/handover/" + tokens[0] + "/attachments/" + addedAttachments[0].ID
	if publicReviewRR.Code != http.StatusOK ||
		!strings.Contains(publicReviewRR.Body.String(), "nachtrag.png") ||
		!strings.Contains(publicReviewRR.Body.String(), publicAttachmentURL) {
		t.Fatalf("public handover review does not expose its token-scoped evidence: %d/%s", publicReviewRR.Code, publicReviewRR.Body.String())
	}
	publicAttachment := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo"+publicAttachmentURL, nil)
	publicAttachment.SetPathValue("token", tokens[0])
	publicAttachment.SetPathValue("id", addedAttachments[0].ID)
	publicAttachmentRR := httptest.NewRecorder()
	a.handoverAttachment(publicAttachmentRR, publicAttachment)
	if publicAttachmentRR.Code != http.StatusOK || publicAttachmentRR.Header().Get("Content-Type") != "image/png" ||
		publicAttachmentRR.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("public handover attachment = %d %q %q", publicAttachmentRR.Code, publicAttachmentRR.Header().Get("Content-Type"), publicAttachmentRR.Header().Get("Cache-Control"))
	}
	wrongTokenAttachment := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/handover/wrong/attachments/"+addedAttachments[0].ID, nil)
	wrongTokenAttachment.SetPathValue("token", "wrong")
	wrongTokenAttachment.SetPathValue("id", addedAttachments[0].ID)
	wrongTokenAttachmentRR := httptest.NewRecorder()
	a.handoverAttachment(wrongTokenAttachmentRR, wrongTokenAttachment)
	if wrongTokenAttachmentRR.Code != http.StatusNotFound {
		t.Fatalf("wrong handover token opened attachment: %d", wrongTokenAttachmentRR.Code)
	}
	unrelatedAttachments, err := attachmentRepositoryForTest(a, "demo").CreateUploaded("handover", "another-handover", "manager@example.com", []uploadedFile{
		testMultipartHeader(t, "attachments", "fremd.png", minimalPNG()),
	}, time.Now())
	if err != nil || len(unrelatedAttachments) != 1 {
		t.Fatalf("unrelated handover attachment = %+v err=%v", unrelatedAttachments, err)
	}
	unrelated := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/handover/"+tokens[0]+"/attachments/"+unrelatedAttachments[0].ID, nil)
	unrelated.SetPathValue("token", tokens[0])
	unrelated.SetPathValue("id", unrelatedAttachments[0].ID)
	unrelatedRR := httptest.NewRecorder()
	a.handoverAttachment(unrelatedRR, unrelated)
	if unrelatedRR.Code != http.StatusNotFound {
		t.Fatalf("handover token opened unrelated attachment: %d", unrelatedRR.Code)
	}
	prematureFile := authedFormRequest(t, a, "manager@example.com", "/demo/app/uebergaben/file", url.Values{"id": {item.ID}})
	if prematureFile.Code != http.StatusSeeOther || !strings.Contains(prematureFile.Header().Get("Location"), "handover=pending") {
		t.Fatalf("pending handover file = %d %q", prematureFile.Code, prematureFile.Header().Get("Location"))
	}
	if docs := documentRepositoryForTest(a, "demo").List(); len(docs) != 0 {
		t.Fatalf("pending handover created documents: %+v", docs)
	}
	for idx, token := range tokens {
		form := url.Values{
			"confirm": {"yes"},
			"name":    {"Bestätigende Person " + strconv.Itoa(idx+1)},
		}
		confirm := httptest.NewRequest(http.MethodPost, "http://hausv.org/demo/handover/"+token, strings.NewReader(form.Encode()))
		confirm.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		confirm.Header.Set("Origin", "http://hausv.org/demo")
		confirmed := httptest.NewRecorder()
		a.handler().ServeHTTP(confirmed, confirm)
		if confirmed.Code != http.StatusSeeOther {
			t.Fatalf("handover confirmation %d status = %d, want redirect", idx, confirmed.Code)
		}
		if idx == 0 {
			lockedDelete := authedFormRequest(t, a, "manager@example.com", "/demo/app/attachments/delete", url.Values{
				"id": {addedAttachments[0].ID},
			})
			if lockedDelete.Code != http.StatusConflict {
				t.Fatalf("confirmed handover attachment delete = %d, want conflict", lockedDelete.Code)
			}
			if got := attachmentRepositoryForTest(a, "demo").ListEntity("handover", item.ID); len(got) != 1 {
				t.Fatalf("confirmed handover attachments = %+v, want immutable file", got)
			}
		}
	}
	afterConfirm, found := testRepositories(a, "demo").handovers.Get(item.ID)
	if !found || handoverStatus(afterConfirm) != handoverStatusConfirmed {
		t.Fatalf("handover after both confirmations = found %v item %+v", found, afterConfirm)
	}
	for _, confirmation := range afterConfirm.Confirmations {
		if confirmation.ConfirmedAt.IsZero() {
			t.Fatalf("handover confirmation remains open: %+v", afterConfirm.Confirmations)
		}
	}

	exported := authedRequest(t, a, "manager@example.com", "/demo/app/uebergaben/"+item.ID+"/protokoll")
	if exported.Code != http.StatusOK || !strings.HasPrefix(exported.Body.String(), "%PDF") {
		t.Fatalf("handover export status/body = %d/%q", exported.Code, exported.Body.String()[:min(12, exported.Body.Len())])
	}
	if ct := exported.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/pdf") {
		t.Fatalf("handover export content-type = %q", ct)
	}

	filed := authedFormRequest(t, a, "manager@example.com", "/demo/app/uebergaben/file", url.Values{"id": {item.ID}})
	if filed.Code != http.StatusSeeOther {
		t.Fatalf("handover file status = %d, want redirect", filed.Code)
	}
	updated, found := testRepositories(a, "demo").handovers.Get(item.ID)
	if !found || updated.FiledDocumentID == "" {
		t.Fatalf("handover not filed: found=%v item=%+v", found, updated)
	}
	docs := documentRepositoryForTest(a, "demo").List()
	if len(docs) != 1 || docs[0].Category != documentCategoryProtocol || docs[0].UnitID != "einheit-12" || docs[0].ContentType != "application/pdf" {
		t.Fatalf("filed document = %+v", docs)
	}
}

func TestHandoverPublicConfirmationToken(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	token := "confirm-secret-token"
	item, err := testRepositories(a, "demo").handovers.Create(handoverRecord{
		ID:           "handover-1",
		TenantSlug:   "demo",
		UnitID:       "top-1",
		Title:        "Übergabe Top 1",
		HandoverType: "Nutzerwechsel",
		Rooms:        []handoverRoom{{Name: "Wohnzimmer", Condition: "gut"}},
		Confirmations: []handoverConfirmation{{
			Role:      "Einziehend",
			Name:      "Neue Nutzerin",
			Email:     "neu@example.com",
			TokenHash: handoverTokenHash(token),
		}},
		CreatedBy: "manager@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("create handover: %v", err)
	}

	get := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/handover/"+token, nil)
	get.SetPathValue("token", token)
	getRR := httptest.NewRecorder()
	a.handoverConfirmPage(getRR, get)
	if getRR.Code != http.StatusOK ||
		!strings.Contains(getRR.Body.String(), "Übergabe Top 1") ||
		!strings.Contains(getRR.Body.String(), "keine Kautions-, Schaden- oder sonstige Abrechnung") ||
		!strings.Contains(getRR.Body.String(), "Wohnzimmer") ||
		!strings.Contains(getRR.Body.String(), `type="checkbox" name="confirm" value="yes" required`) ||
		!strings.Contains(getRR.Body.String(), "Protokoll bestätigen") {
		t.Fatalf("confirm page status/body = %d/%s", getRR.Code, getRR.Body.String())
	}

	form := url.Values{"confirm": {"yes"}, "name": {"Neue Nutzerin"}, "note": {"geprüft"}}
	post := httptest.NewRequest(http.MethodPost, "http://hausv.org/demo/handover/"+token, strings.NewReader(form.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.Header.Set("Origin", "http://hausv.org/demo")
	post.SetPathValue("token", token)
	postRR := httptest.NewRecorder()
	a.confirmHandover(postRR, post)
	if postRR.Code != http.StatusSeeOther {
		t.Fatalf("confirm post status = %d, want redirect", postRR.Code)
	}
	updated, found := testRepositories(a, "demo").handovers.Get(item.ID)
	if !found || len(updated.Confirmations) != 1 || updated.Confirmations[0].ConfirmedAt.IsZero() || updated.Confirmations[0].Note != "geprüft" {
		t.Fatalf("updated confirmation = found %v item %+v", found, updated)
	}
	confirmEventsBefore := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionHandoverConfirm, Limit: 20})
	repeatForm := url.Values{"confirm": {"yes"}, "name": {"Andere Person"}, "note": {"darf nicht ändern"}}
	repeat := httptest.NewRequest(http.MethodPost, "http://hausv.org/demo/handover/"+token, strings.NewReader(repeatForm.Encode()))
	repeat.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	repeat.Header.Set("Origin", "http://hausv.org/demo")
	repeat.SetPathValue("token", token)
	repeatRR := httptest.NewRecorder()
	a.confirmHandover(repeatRR, repeat)
	afterRepeat, _ := testRepositories(a, "demo").handovers.Get(item.ID)
	confirmEventsAfter := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionHandoverConfirm, Limit: 20})
	if repeatRR.Code != http.StatusSeeOther || afterRepeat.Confirmations[0].Note != "geprüft" ||
		len(confirmEventsAfter) != len(confirmEventsBefore) {
		t.Fatalf("repeated confirmation changed state: status=%d confirmation=%+v audit=%d→%d", repeatRR.Code, afterRepeat.Confirmations[0], len(confirmEventsBefore), len(confirmEventsAfter))
	}
	revisit := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/handover/"+token, nil)
	revisit.SetPathValue("token", token)
	revisitRR := httptest.NewRecorder()
	a.handoverConfirmPage(revisitRR, revisit)
	if revisitRR.Code != http.StatusOK || !strings.Contains(revisitRR.Body.String(), "Protokoll bestätigt") ||
		strings.Contains(revisitRR.Body.String(), `class="handover-confirm-form"`) {
		t.Fatalf("confirmed token is not an understandable read-only view: %d/%s", revisitRR.Code, revisitRR.Body.String())
	}
}

func TestTenantHomeUsesHouseLanguageAndPrivateMapLink(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.mailer = &recordingMailer{}
	profile := energy.DefaultProfile("demo", time.Now())
	profile.HomeType = energy.HomeCommunity
	profile.HouseholdName = "Musterweg 1"
	profile.OnboardingComplete = true
	profile.OnboardingStep = 5
	if err := a.energyStore.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/", nil)
	rr := httptest.NewRecorder()
	a.home(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("home status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{
		"Ihr Hausportal",
		"Alles Wichtige rund um unser Haus.",
		"Anmelden",
		"Anmeldelink senden",
		"Karte öffnen",
		"© OpenStreetMap",
		`class="location-map-tile" data-map-tile="/demo/map-tiles/17/`,
		"Privat für eingeladene Personen",
		"Impressum",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("home should render house-focused entry %q, body: %s", want, body)
		}
	}
	for _, forbidden := range []string{"WEG Portal", "Zitadel", "SSO", "Parkplatzabrechnung", "Wohneinheiten im Haus"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("home should not expose provider/product jargon %q, body: %s", forbidden, body)
		}
	}
}

func TestExpiredMagicLinkReturnsToFriendlyLoginWithoutToken(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	verify := httptest.NewRecorder()
	a.verifyLogin(verify, httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/auth/verify?token=expired-secret", nil))
	if verify.Code != http.StatusSeeOther {
		t.Fatalf("expired verify status = %d, want redirect", verify.Code)
	}
	location := verify.Header().Get("Location")
	if location != "/?login=expired#login" || strings.Contains(location, "token") || strings.Contains(location, "expired-secret") {
		t.Fatalf("expired verify location = %q", location)
	}

	page := httptest.NewRecorder()
	a.home(page, httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/?login=expired", nil))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Der bisherige Link ist abgelaufen") {
		t.Fatalf("friendly expired page = %d %q", page.Code, page.Body.String())
	}
	if strings.Contains(page.Body.String(), "expired-secret") {
		t.Fatal("expired page must not expose the consumed token")
	}
}

func TestRootDomainRendersMarketingLanding(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	rr := httptest.NewRecorder()
	a.home(rr, httptest.NewRequest(http.MethodGet, "http://hausv.org/", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("landing status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{
		"Ein Hausportal. Alles, was Menschen und Gebäude verbindet.",
		"HAUSV Free",
		"HAUSV Home",
		"HAUSV Professional",
		"Vertrauen verbindet alle drei Wege.",
		"Zentraler Hausüberblick",
		"Aushänge, die ankommen",
		"Kalender, der mitgeht",
		"Vom Mail-Eingang zur Lösung",
		"Geschützt und auffindbar",
		"Wohnungen digital übergeben",
		"Abstimmungen mit Verlauf",
		"Kontakte, Rollen und Rechte",
		"Live-Energie verständlich",
		"Jahresabrechnung bis zum Versand",
		"hello [at] hausv [dot] org",
		"wahlweise hosted oder selbst betrieben",
		"Quelloffen · selbst betrieben",
		"AGPL-3.0",
		"0&nbsp;€ für immer",
		"Quellcode ab Version 1.0",
		"Volle Rollen &amp; Rechte",
		"Dokumente &amp; Aushänge",
		"Anliegen mit Verlauf",
		"Energie &amp; Messwerte",
		"12 Monate kostenlos",
		"12&nbsp;€ pro Jahr",
		"<strong>0&nbsp;€ Grundgebühr</strong>",
		"<span>25 WE kostenlos · danach Verrechnung je&nbsp;WE&nbsp;/&nbsp;Monat</span>",
		"Home Assistant",
		"ebInterface und CAMT",
		"Impressum",
		"Betreiber laut Host-Konfiguration",
		"natürliche Person",
		`href="/impressum"`,
		"Rechtliches im Detail",
		"Impressum &amp; Infos",
		"Datensparsam",
		"KI nur mit Opt-in",
		"Was gerade entsteht",
		"Verrechnung: gemeinsam mit Friendly Customers",
		"Die Jahresabrechnung ist da: Lauf, PDF je Partei, Archiv und Versand.",
		"Buchhaltung, Mahnwesen und Zahlungsläufe folgen",
		"Wir bauen sie mit ausgewählten Verwaltungen statt am grünen Tisch.",
		"Energie: kontrolliert statt unbedacht",
		"Aktive Steuerung nach bewusster Freigabe",
		"Dienstleister-Zugänge bleiben rollenbasiert.",
		"Kein öffentlicher Marktplatz, kein Handel im Hintergrund.",
		// HAUSV-668: the media area fills the card to its hairline, and no card
		// carries a tinted border any more.
		"object-fit: cover; object-position: center;",
		"background: #fbf4e8;",
		`/assets/landing.js`,
		"/assets/hausv-landing-hero.png",
		"mark3d-stage",
		"mark3d-fallback",
		"Sicherheit & Datenschutz",
		"Home oder Professional?",
		"Gespräch anfragen",
		`href="/start"`,
		"HAUSV Home starten",
		"Eigenbetrieb vormerken",
		"Professional anfragen",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("landing page missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "Parken, Laden und anbinden") {
		t.Fatal("landing must not render the former lowercase integration heading")
	}
	// HAUSV-668 replaced the "what we leave out" disclosure with the roadmap the
	// product actually has; the old promises must not creep back.
	for _, stale := range []string{
		"Was bewusst nicht Teil des Portals ist",
		"Kein Verrechnungssystem",
		"Keine Jahresabrechnung oder Buchhaltung",
		"Kein Mahnwesen und keine Zahlungsaufträge",
		"Übergabe an bestehende Fachsysteme statt Nachbau",
		"Aktive Energiesteuerung nur nach bewusster Freigabe",
		"Dienstleister-Zugänge ausschließlich rollenbasiert",
		"Kein öffentlicher Marktplatz oder eigener Zahlungsfluss",
		"Parken, Laden und Anbinden",
		"Schäden sauber lösen",
		".feature-card:nth-child(1), .feature-card:nth-child(9)",
		"object-fit: contain",
	} {
		if strings.Contains(body, stale) {
			t.Fatalf("landing page still carries the pre-HAUSV-668 state %q", stale)
		}
	}
	for _, forbidden := range []string{"hallo@hausv.org", "hello@hausv.org", "Peak Shaving", "Bis 10 Häuser kostenlos", "Fair Use bis 10 Einheiten kostenlos", "Bis 25 Einheiten im Pilot kostenlos", "Richtwert für später", "Spenden", "3&nbsp;€ je Einheit", "500 €", "900 €", "Ladungsfähige Anschrift", "Musterweg 1", "Einfach kalkulierbar", "Klein starten. Erst mit dem Nutzen wachsen.", "Alle drei Produkte starten kostenlos", "25 Einheiten kostenlos", "0,12&nbsp;€ je Einheit / Monat", "Quellcode-Veröffentlichung mit Version 1.0", `id="preise"`, `href="#preise"`, `class="offer-card`, `class="open-source-note"`, `class="boundary-strip"`, "mark3d-top", "data-mark3d-top", "KI-first", `mailto:hallo`, `mailto:hello`, "Pilot verfügbar", "Betreiberfreigabe vorbereitet", "camt.053", "camt.054", "BMD/RZL", "[Name oder Firma", "[Straße und Hausnummer", "[Firmenbuchnummer", "Platzhalter"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("landing page should not expose/regress %q:\n%s", forbidden, body)
		}
	}
	if got := strings.Count(body, `class="product-path `); got != 3 {
		t.Fatalf("product path count = %d, want 3", got)
	}
	if got := strings.Count(body, `class="product-path-start`); got != 3 {
		t.Fatalf("product CTA count = %d, want 3", got)
	}
	if got := strings.Count(body, `class="feature-card"`); got != 10 {
		t.Fatalf("feature card count = %d, want 10", got)
	}
	if strings.Count(body, `class="shared-core"`) != 1 {
		t.Fatal("landing should show the shared trust core exactly once")
	}
	if strings.Contains(body, `class="roadmap-grid"`) || strings.Contains(body, `class="use-grid"`) {
		t.Fatal("landing should not render the old repetitive roadmap or role card grids")
	}
	if strings.Contains(body, `action="/auth/request"`) {
		t.Fatal("root-domain landing should not render the tenant login form")
	}
}

func TestWwwHostRedirectsPermanentlyToApex(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	handler := a.handler()

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "http://www.hausv.org/datenschutz?from=demo", nil))
	if rr.Code != http.StatusMovedPermanently {
		t.Fatalf("www redirect status = %d, want %d", rr.Code, http.StatusMovedPermanently)
	}
	if got, want := rr.Header().Get("Location"), "https://hausv.org/datenschutz?from=demo"; got != want {
		t.Fatalf("www redirect location = %q, want %q", got, want)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "http://hausv.org/", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("apex landing status = %d, want 200", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("tenant host status = %d, want 200 (must not be redirected)", rr.Code)
	}
}

func TestImprintPageCarriesLegalDetails(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "http://hausv.org/impressum", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("imprint status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{
		"Impressum &amp; Infos",
		"Medieninhaber / Betreiber",
		"Betreiber laut Host-Konfiguration",
		"Ladungsfähige Anschrift",
		"Anschrift laut Host-Konfiguration",
		"Blattlinie",
		"Professionelle Services",
		"HAUSV Professional",
		"GNU AGPL-3.0",
		"Betreiber-Selbstprüfung vom 14. August 2026",
		"§ 5 ECG",
		"§ 24 MedienG",
		"Keine externe Zertifizierung",
		"Datenschutzinformation",
		"Zurück zur Startseite",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("imprint page missing %q:\n%s", want, body)
		}
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

	token, _, err := first.Put("user@example.com", "DEMO", authMethodOIDC, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}

	second := newSessionStore(secret)
	email, tenantSlug, authMethod, ok := second.Get(token)
	if !ok {
		t.Fatal("session should verify in a new store with the same secret")
	}
	if email != "user@example.com" || tenantSlug != "demo" || authMethod != authMethodOIDC {
		t.Fatalf("unexpected session claims: %s %s %s", email, tenantSlug, authMethod)
	}

	other := newSessionStore([]byte(strings.Repeat("x", 32)))
	if _, _, _, ok := other.Get(token); ok {
		t.Fatal("session should not verify with a different secret")
	}
}

func TestParseUserProfilesNormalizesAuthMethods(t *testing.T) {
	raw := `[{"email":"joerg.lehner@gmx.net","first_name":"Jörg","last_name":"Lehner","tenants":["demo"],"auth_methods":["zitadel"]}]`

	profiles, err := parseUserProfiles(raw, map[string]struct{}{}, map[string]struct{}{}, "demo")
	if err != nil {
		t.Fatalf("parse profiles: %v", err)
	}
	profile := profiles["joerg.lehner@gmx.net"]
	if !profile.AllowsAuthMethod(authMethodOIDC) {
		t.Fatal("profile should allow OIDC")
	}
	if profile.AllowsAuthMethod(authMethodEmail) {
		t.Fatal("profile should not allow email login")
	}
	if got := userRowFrom(profile).AuthLabel; got != "Sichere Anmeldung" {
		t.Fatalf("auth label = %q", got)
	}
}

func TestParseUserProfilesNormalizesTenantMemberships(t *testing.T) {
	raw := `[{"email":"multi@example.com","role":"resident","permissions":["parking"],"tenant_memberships":{"DEMO":{"role":"Hausverwaltung"},"Haus-B":{"role":"Mieter","permissions":[]}}}]`

	profiles, err := parseUserProfiles(raw, map[string]struct{}{}, map[string]struct{}{}, "demo")
	if err != nil {
		t.Fatalf("parse profiles: %v", err)
	}
	profile := profiles["multi@example.com"]
	if !profile.HasTenant("demo") || !profile.HasTenant("haus-b") {
		t.Fatalf("tenant membership keys should grant tenant membership: %+v", profile)
	}

	demoProfile := profile.ForTenant("demo")
	if demoProfile.Role != roleManager || !demoProfile.HasPermission(permissionParking) {
		t.Fatalf("demo profile = %+v, want manager inheriting parking permission", demoProfile)
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
		Tenants:     []string{"demo", "haus-b"},
		Permissions: []string{permissionParking},
		TenantMemberships: map[string]tenantMembership{
			"demo":   {Role: roleManager},
			"haus-b": {Role: roleResident, Permissions: []string{}},
		},
		AuthMethods: defaultAuthMethods(),
	})
	addTestTenant(a, tenantConfig{Slug: "haus-b", Name: "Haus B", Address: "Haus B"})

	if got := a.roleFor("multi@example.com", "demo"); got != roleManager {
		t.Fatalf("roleFor demo = %q, want %q", got, roleManager)
	}
	if got := a.roleFor("multi@example.com", "haus-b"); got != roleResident {
		t.Fatalf("roleFor haus-b = %q, want %q", got, roleResident)
	}

	demoProfile := a.profileForTenant("multi@example.com", "demo")
	if demoProfile.Role != roleManager || !demoProfile.HasPermission(permissionParking) {
		t.Fatalf("demo profile = %+v, want manager with parking", demoProfile)
	}
	otherProfile := a.profileForTenant("multi@example.com", "haus-b")
	if otherProfile.Role != roleResident || otherProfile.HasPermission(permissionParking) {
		t.Fatalf("haus-b profile = %+v, want resident without parking", otherProfile)
	}

	token, _, err := a.sessions.Put("multi@example.com", "demo", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app", nil)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	_, role, tenantSlug, ok := a.currentUser(req)
	if !ok || tenantSlug != "demo" || role != roleManager {
		t.Fatalf("currentUser ok=%v tenant=%q role=%q, want demo manager", ok, tenantSlug, role)
	}

	demoRow := userRowForEmail(t, a.userRows(testTenantRef("demo")), "multi@example.com")
	if demoRow.Role != roleManager || !demoRow.ParkingChecked {
		t.Fatalf("demo row = %+v, want manager with parking checked", demoRow)
	}
	otherRow := userRowForEmail(t, a.userRows(testTenantRef("haus-b")), "multi@example.com")
	if otherRow.Role != roleResident || otherRow.ParkingChecked {
		t.Fatalf("haus-b row = %+v, want resident without parking", otherRow)
	}
}

func TestRoleForBackwardsCompatibleDefault(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"demo"},
		Permissions: []string{permissionParking},
		AuthMethods: defaultAuthMethods(),
	})

	if got := a.roleFor("owner@example.com", "demo"); got != roleOwner {
		t.Fatalf("roleFor default tenant = %q, want %q", got, roleOwner)
	}
	profile := a.profileForTenant("owner@example.com", "demo")
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
	if login.Provider() != nil || login.Verifier() != nil {
		t.Fatal("provider should not be initialized after canceled discovery")
	}
	if err := login.EnsureProvider(ctx); err == nil {
		t.Fatal("retry with canceled context should still report discovery failure")
	}
}

func TestOIDCRedirectURLUsesStablePlatformCallback(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "owner@example.com"})
	a.baseURL = "https://hausv.org"

	if got, want := a.oidcRedirectURL(), "https://hausv.org/auth/oidc/callback"; got != want {
		t.Fatalf("oidcRedirectURL() = %q, want %q", got, want)
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
		"Handwerker":       roleServiceProvider,
		"service-provider": roleServiceProvider,
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
		{roleServiceProvider, capabilityManageIssues, false},
		{roleServiceProvider, capabilityOwnerDocuments, false},
		{roleServiceProvider, capabilityVote, false},
	}
	for _, tc := range cases {
		if got := roleHasCapability(tc.role, tc.cap); got != tc.want {
			t.Fatalf("roleHasCapability(%q, %q) = %v, want %v", tc.role, tc.cap, got, tc.want)
		}
	}
}

func TestSettingsHubVisibleToResidentWithoutAdminSections(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	rr := authedRequest(t, a, "resident@example.com", "/demo/app/settings")
	if rr.Code != http.StatusOK {
		t.Fatalf("settings hub status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{`href="/demo/app/settings"`, "Profil", "Benachrichtigungen", "Kalender-Abo", "/calendar/", "Aktivitätsverlauf", "Änderungen nachvollziehen"} {
		if !strings.Contains(body, want) {
			t.Fatalf("settings hub should contain %q", want)
		}
	}
	if strings.Contains(body, "<h2>Verwaltung</h2>") {
		t.Fatal("resident settings hub must not label the personal audit area as management")
	}
	if strings.Contains(body, `href="/demo/app/parking/settings"`) {
		t.Fatal("resident settings hub must not expose parking settings")
	}
	if strings.Contains(body, `href="/demo/app/settings/parking-access"`) {
		t.Fatal("resident settings hub must not expose parking access management")
	}
}

func TestNotificationSettingsPersistAndRender(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	save := authedFormRequest(t, a, "resident@example.com", "/demo/app/settings/notifications", url.Values{
		"email_enabled": {"on"},
		"events":        {notificationEventAnnouncement, notificationEventDocument},
	})
	if save.Code != http.StatusSeeOther {
		t.Fatalf("notification settings save status = %d", save.Code)
	}
	prefs := a.notificationPrefs.Get("resident@example.com")
	if prefs.Unsubscribed || !prefs.Email[notificationEventAnnouncement] || !prefs.Email[notificationEventDocument] || prefs.Email[notificationEventIssue] {
		t.Fatalf("saved notification prefs = %+v", prefs)
	}

	page := authedRequest(t, a, "resident@example.com", "/demo/app/settings/notifications")
	if page.Code != http.StatusOK {
		t.Fatalf("notification settings status = %d", page.Code)
	}
	body := page.Body.String()
	if !strings.Contains(body, `href="/demo/app/settings"`) || !strings.Contains(body, `value="announcement" checked`) || strings.Contains(body, `value="issue" checked`) {
		t.Fatalf("notification settings page did not reflect saved prefs:\n%s", body)
	}
	for _, want := range []string{`data-notification-form`, "Liegenschaft &amp; Kommunikation", "Entscheidungen &amp; Unterlagen", "Zahlung &amp; Nutzung", "2 von 6 aktiv"} {
		if !strings.Contains(body, want) {
			t.Fatalf("notification settings page should contain %q", want)
		}
	}

	pause := authedFormRequest(t, a, "resident@example.com", "/demo/app/settings/notifications", url.Values{
		"events": {notificationEventAnnouncement, notificationEventDocument},
	})
	if pause.Code != http.StatusSeeOther {
		t.Fatalf("notification pause status = %d", pause.Code)
	}
	hub := authedRequest(t, a, "resident@example.com", "/demo/app/settings")
	if !strings.Contains(hub.Body.String(), "E-Mails pausiert") {
		t.Fatalf("settings hub should summarize paused notifications:\n%s", hub.Body.String())
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
	if err := store.Set("DEMO", tenantOverride{
		Name:              "WEG Sonneneck",
		Address:           "Neue Gasse 7",
		BrandIcon:         "multi-tenant",
		BrandAbbreviation: "sn eck<script>",
		ContactName:       "Hausverwaltung Nord",
		ContactEmail:      "Office@Example.com",
		ContactPhone:      "+43 1 999",
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

	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.tenants["demo"] = tenantConfig{
		Slug:         "demo",
		Name:         "Env Name",
		Address:      "Env Address",
		ContactName:  "Env Contact",
		ContactEmail: "env@example.com",
		ContactPhone: "+43 1 111",
		HeroImageURL: "/assets/env.jpg",
	}
	a.tenantOverrides = store

	tenant, ok := a.tenantBySlug("demo")
	if !ok {
		t.Fatal("tenant not found")
	}
	if tenant.Name != "WEG Sonneneck" || tenant.Address != "Neue Gasse 7" || tenant.BrandIcon != tenantBrandMultiTenant || tenant.BrandAbbreviation != "SN-ECKSCRIPT" || tenant.ContactEmail != "office@example.com" || tenant.HeroImageURL != "/assets/env.jpg" {
		t.Fatalf("tenant override = %+v", tenant)
	}
	if err := store.SetHeroImage("demo", "demo-hero.png"); err != nil {
		t.Fatalf("set hero image: %v", err)
	}
	tenant, _ = a.tenantBySlug("demo")
	if tenant.HeroImageURL != "/tenant-hero/demo" || tenant.ContactName != "Hausverwaltung Nord" || tenant.BrandIcon != tenantBrandMultiTenant {
		t.Fatalf("tenant after hero override = %+v", tenant)
	}
}

func TestProfileSettingsPersistOverlayWithoutAuthzEscalation(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{{
		ID:                    "top-1",
		Label:                 "Top 1",
		MiteigentumsanteilPPM: 12345,
		RenterEmails:          []string{"resident@example.com"},
	}}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}

	save := authedFormRequest(t, a, "resident@example.com", "/demo/app/settings/profile", url.Values{
		"title":            {"Dr."},
		"first_name":       {"Resi"},
		"last_name":        {"Dent"},
		"phone":            {"+43 1 234"},
		"directory_opt_in": {"on"},
		"role":             {"Admin"},
		"permissions":      {permissionParking},
	})
	if save.Code != http.StatusSeeOther {
		t.Fatalf("profile save status = %d, want redirect", save.Code)
	}
	profile := a.profileForTenant("resident@example.com", "demo")
	if profile.DisplayName() != "Dr. Resi Dent" || profile.Phone != "+43 1 234" {
		t.Fatalf("profile overlay not applied: %+v", profile)
	}
	if profile.Role != roleResident || profile.HasPermission(permissionParking) {
		t.Fatalf("profile self-edit escalated authz: %+v", profile)
	}
	if !profile.DirectoryOptIn {
		t.Fatalf("profile directory opt-in not applied: %+v", profile)
	}
	if parking := authedRequest(t, a, "resident@example.com", "/demo/app/parking"); parking.Code != http.StatusNotFound {
		t.Fatalf("parking status = %d, want 404 without parking permission", parking.Code)
	}

	settings := authedRequest(t, a, "resident@example.com", "/demo/app/settings")
	if !strings.Contains(settings.Body.String(), `href="/demo/app/settings/profile"`) {
		t.Fatalf("settings hub should link to profile:\n%s", settings.Body.String())
	}
	page := authedRequest(t, a, "resident@example.com", "/demo/app/settings/profile")
	if page.Code != http.StatusOK {
		t.Fatalf("profile page status = %d", page.Code)
	}
	body := page.Body.String()
	for _, want := range []string{`value="Dr."`, `value="Resi"`, `value="Dent"`, "43 1 234", `name="directory_opt_in" checked`, "Die Freigabe ist freiwillig.", "Konto &amp; Berechtigungen", roleResident, "Top 1", "12.345 / 1.000.000"} {
		if !strings.Contains(body, want) {
			t.Fatalf("profile page should contain %q", want)
		}
	}
	row := userRowForEmail(t, a.userRows(testTenantRef("demo")), "resident@example.com")
	if row.DisplayName != "Dr. Resi Dent" || row.Phone != "+43 1 234" || !row.DirectoryOptIn || row.Role != roleResident || row.ParkingChecked {
		t.Fatalf("roster row = %+v, want overlay display without authz escalation", row)
	}
}

func TestContactsPageShowsBuildingBoardAndOptInDirectory(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", FirstName: "Resi", LastName: "Dent", Phone: "+43 1 234", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["private@example.com"] = userProfile{Email: "private@example.com", FirstName: "Privat", LastName: "Person", Phone: "+43 1 555", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["board@example.com"] = userProfile{Email: "board@example.com", FirstName: "Berta", LastName: "Beirat", Phone: "+43 1 777", Role: roleBeirat, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}

	saveMeta := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/building", url.Values{
		"name":            {"WEG Portal"},
		"address":         {"Musterweg 1"},
		"contact_name":    {"Hausverwaltung Nord"},
		"contact_address": {"Verwaltergasse 10, 1010 Wien"},
		"contact_email":   {"office@example.com"},
		"contact_phone":   {"+43 1 999"},
		"emergency_name":  {"Notdienst 24"},
		"emergency_phone": {"144"},
		"caretaker_name":  {"Hausmeister Max"},
		"caretaker_email": {"hausmeister@example.com"},
		"caretaker_phone": {"+43 1 888"},
	})
	if saveMeta.Code != http.StatusSeeOther {
		t.Fatalf("building settings save status = %d", saveMeta.Code)
	}

	page := authedRequest(t, a, "resident@example.com", "/demo/app/kontakte")
	if page.Code != http.StatusOK {
		t.Fatalf("contacts status = %d", page.Code)
	}
	body := page.Body.String()
	for _, want := range []string{`href="/demo/app/kontakte"`, "nav-item active", "Hausverwaltung Nord", "office@example.com", "Notdienst 24", "144", "Hausmeister Max", "hausmeister@example.com", "Berta Beirat", "board@example.com", "43 1 777"} {
		if !strings.Contains(body, want) {
			t.Fatalf("contacts page should contain %q", want)
		}
	}
	for _, forbidden := range []string{"resident@example.com", "43 1 234", "private@example.com", "43 1 555"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("contacts page exposed non-opt-in contact %q", forbidden)
		}
	}

	optIn := authedFormRequest(t, a, "resident@example.com", "/demo/app/settings/profile", url.Values{
		"first_name":       {"Resi"},
		"last_name":        {"Dent"},
		"phone":            {"+43 1 234"},
		"directory_opt_in": {"on"},
	})
	if optIn.Code != http.StatusSeeOther {
		t.Fatalf("profile opt-in status = %d", optIn.Code)
	}
	page = authedRequest(t, a, "resident@example.com", "/demo/app/kontakte")
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
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}

	resident := authedRequest(t, a, "resident@example.com", "/demo/app/settings/building")
	if resident.Code != http.StatusForbidden {
		t.Fatalf("resident building settings status = %d, want 403", resident.Code)
	}
	hub := authedRequest(t, a, "manager@example.com", "/demo/app/settings")
	if !strings.Contains(hub.Body.String(), `href="/demo/app/settings/building"`) {
		t.Fatalf("manager settings hub should link building settings:\n%s", hub.Body.String())
	}

	saveMeta := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/building", url.Values{
		"name":               {"WEG Sonneneck"},
		"address":            {"Neue Gasse 7"},
		"brand_icon":         {"mixed-use"},
		"brand_abbreviation": {"sun/eck"},
		"contact_name":       {"Hausverwaltung Nord"},
		"contact_address":    {"Verwaltergasse 10, 1010 Wien"},
		"contact_email":      {"office@example.com"},
		"contact_phone":      {"+43 1 999"},
		"emergency_name":     {"Notdienst 24"},
		"emergency_phone":    {"144"},
		"caretaker_name":     {"Hausmeister Max"},
		"caretaker_email":    {"hausmeister@example.com"},
		"caretaker_phone":    {"+43 1 888"},
	})
	if saveMeta.Code != http.StatusSeeOther {
		t.Fatalf("building meta save status = %d", saveMeta.Code)
	}
	tenant, _ := a.tenantBySlug("demo")
	if tenant.Name != "WEG Sonneneck" || tenant.Address != "Neue Gasse 7" || tenant.BrandIcon != tenantBrandMixedUse || tenant.BrandAbbreviation != "SUN-ECK" || tenant.ContactName != "Hausverwaltung Nord" || tenant.ContactAddress != "Verwaltergasse 10, 1010 Wien" || tenant.ContactEmail != "office@example.com" || tenant.ContactPhone != "+43 1 999" || tenant.EmergencyName != "Notdienst 24" || tenant.EmergencyPhone != "144" || tenant.CaretakerName != "Hausmeister Max" || tenant.CaretakerEmail != "hausmeister@example.com" || tenant.CaretakerPhone != "+43 1 888" {
		t.Fatalf("tenant after meta save = %+v", tenant)
	}
	homeReq := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/", nil)
	home := httptest.NewRecorder()
	a.home(home, homeReq)
	if !strings.Contains(home.Body.String(), "WEG Sonneneck") || !strings.Contains(home.Body.String(), "Neue Gasse 7") || !strings.Contains(home.Body.String(), defaultTenantHeroImageURL) {
		t.Fatalf("home should render layered name, address and default hero:\n%s", home.Body.String())
	}
	appPage := authedRequest(t, a, "manager@example.com", "/demo/app")
	for _, want := range []string{`class="map `, `Neue Gasse 7 in OpenStreetMap öffnen`} {
		if !strings.Contains(appPage.Body.String(), want) {
			t.Fatalf("app sidebar should render selected brand marker %q:\n%s", want, appPage.Body.String())
		}
	}
	if strings.Contains(appPage.Body.String(), `class="side-code"`) || strings.Contains(appPage.Body.String(), ">SUN-ECK<") {
		t.Fatal("app sidebar should not repeat the internal brand abbreviation")
	}
	settingsPage := authedRequest(t, a, "manager@example.com", "/demo/app/settings/building?section=appearance")
	for _, want := range []string{`name="brand_icon"`, `value="mixed-use" checked`, `value="SUN-ECK"`, `Gemischt genutzt`, `Weitere Symbole aus Lucide`, `brand-lucide-icon-names`, `Live-Vorschau`} {
		if !strings.Contains(settingsPage.Body.String(), want) {
			t.Fatalf("building settings should render brand control %q:\n%s", want, settingsPage.Body.String())
		}
	}
	saveLucideBrand := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/building/appearance", url.Values{
		"brand_icon":         {"lucide:tree-pine"},
		"brand_abbreviation": {"WALD"},
	})
	if saveLucideBrand.Code != http.StatusSeeOther || saveLucideBrand.Result().Header.Get("Location") != "/demo/app/settings/building?section=appearance&building=saved" {
		t.Fatalf("Lucide brand save status=%d location=%q", saveLucideBrand.Code, saveLucideBrand.Result().Header.Get("Location"))
	}
	tenant, _ = a.tenantBySlug("demo")
	if tenant.BrandIcon != "lucide:tree-pine" || tenant.BrandAbbreviation != "WALD" {
		t.Fatalf("tenant after Lucide brand save = %+v", tenant)
	}
	appPage = authedRequest(t, a, "manager@example.com", "/demo/app")
	for _, want := range []string{`class="map `, `Neue Gasse 7 in OpenStreetMap öffnen`} {
		if !strings.Contains(appPage.Body.String(), want) {
			t.Fatalf("app sidebar should render selected Lucide marker %q:\n%s", want, appPage.Body.String())
		}
	}
	invalidBrand := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/building", url.Values{
		"name":       {"WEG Sonneneck"},
		"address":    {"Neue Gasse 7"},
		"brand_icon": {"<svg onload=alert(1)>"},
	})
	if invalidBrand.Code != http.StatusSeeOther || invalidBrand.Result().Header.Get("Location") != "/demo/app/settings/building?section=overview&building=invalid" {
		t.Fatalf("invalid brand save should redirect invalid, status=%d location=%q", invalidBrand.Code, invalidBrand.Result().Header.Get("Location"))
	}

	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89,
	}
	uploadHero := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/settings/building/hero", nil, "hero_image", "hero.png", png)
	if uploadHero.Code != http.StatusSeeOther {
		t.Fatalf("hero upload status = %d", uploadHero.Code)
	}
	tenant, _ = a.tenantBySlug("demo")
	if tenant.HeroImageURL != "/tenant-hero/demo" {
		t.Fatalf("tenant hero url = %q", tenant.HeroImageURL)
	}
	override, ok := a.tenantOverrides.Get("demo")
	if !ok || !strings.HasPrefix(override.HeroImage, "demo-hero-") || !strings.HasSuffix(override.HeroImage, ".png") {
		t.Fatalf("hero override filename = %#v, ok=%v", override.HeroImage, ok)
	}
	heroPath := filepath.Join(a.tenantHeroDir, override.HeroImage)
	heroInfo, err := os.Stat(heroPath)
	if err != nil {
		t.Fatalf("stat hero file: %v", err)
	}
	if heroInfo.Mode().Perm() != 0o600 {
		t.Fatalf("hero file mode = %v, want 0600", heroInfo.Mode().Perm())
	}
	heroReq := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/tenant-hero/demo", nil)
	heroReq.SetPathValue("tenant", "demo")
	hero := httptest.NewRecorder()
	a.tenantHeroImage(hero, heroReq)
	if hero.Code != http.StatusOK || !strings.HasPrefix(hero.Header().Get("Content-Type"), "image/png") {
		t.Fatalf("hero response status=%d content-type=%q", hero.Code, hero.Header().Get("Content-Type"))
	}
	homeAfterHeroReq := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/", nil)
	homeAfterHero := httptest.NewRecorder()
	a.home(homeAfterHero, homeAfterHeroReq)
	if !strings.Contains(homeAfterHero.Body.String(), "/tenant-hero/demo") {
		t.Fatalf("home should render uploaded hero path:\n%s", homeAfterHero.Body.String())
	}
	settingsAfterHero := authedRequest(t, a, "manager@example.com", "/demo/app/settings/building?section=appearance")
	if !strings.Contains(settingsAfterHero.Body.String(), `/demo/app/settings/building/hero/delete`) {
		t.Fatalf("building settings should offer hero reset after upload:\n%s", settingsAfterHero.Body.String())
	}
	deleteHero := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/building/hero/delete", url.Values{})
	if deleteHero.Code != http.StatusSeeOther {
		t.Fatalf("hero delete status = %d", deleteHero.Code)
	}
	tenant, _ = a.tenantBySlug("demo")
	if tenant.HeroImageURL != defaultTenantHeroImageURL {
		t.Fatalf("tenant hero after delete = %q", tenant.HeroImageURL)
	}
	override, _ = a.tenantOverrides.Get("demo")
	if override.HeroImage != "" {
		t.Fatalf("hero override after delete = %q", override.HeroImage)
	}
	if _, err := os.Stat(heroPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("hero file should be removed after delete, stat err=%v", err)
	}
	heroAfterDelete := httptest.NewRecorder()
	a.tenantHeroImage(heroAfterDelete, heroReq)
	if heroAfterDelete.Code != http.StatusNotFound {
		t.Fatalf("deleted tenant hero status = %d, want 404", heroAfterDelete.Code)
	}

	addUnit := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/building/units", url.Values{
		"label":              {"Top 1"},
		"unit_type":          {"residential"},
		"miteigentumsanteil": {"12345"},
		"owner_emails":       {"owner@example.com; second@example.com"},
		"renter_emails":      {"resident@example.com"},
		"payment_status":     {unitPaymentStatusOverdue},
	})
	if addUnit.Code != http.StatusSeeOther {
		t.Fatalf("unit add status = %d", addUnit.Code)
	}
	units := testUnitRepository(t, a, "demo").List()
	if len(units) != 1 || units[0].ID != "top-1" || units[0].UnitType != unitTypeResidential || units[0].BillableWeightPPM != unitBillableFullPPM || units[0].MiteigentumsanteilPPM != 12345 || len(units[0].OwnerEmails) != 2 || units[0].RenterEmails[0] != "resident@example.com" {
		t.Fatalf("units after add = %+v", units)
	}
	if payment, ok := testUnitPaymentRepository(t, a, "demo").Get("top-1"); !ok || payment.Status != unitPaymentStatusOverdue {
		t.Fatalf("payment status after add = %+v, found=%t", payment, ok)
	}
	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/building?section=units")
	for _, want := range []string{"WEG Sonneneck", "Neue Gasse 7", "Top 1", "Wohnung", "zählt als 1 WE", "1 von 25", "Wohneinheit im inkludierten Rahmen", "12.345 / 1.000.000", `value="owner@example.com, second@example.com"`} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("building page should contain %q", want)
		}
	}

	editUnit := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/building/units", url.Values{
		"orig_id":            {"top-1"},
		"id":                 {"top-1"},
		"label":              {"Top 1A"},
		"unit_type":          {"parking"},
		"miteigentumsanteil": {"23456"},
		"owner_emails":       {"owner@example.com"},
		"renter_emails":      {""},
	})
	if editUnit.Code != http.StatusSeeOther {
		t.Fatalf("unit edit status = %d", editUnit.Code)
	}
	units = testUnitRepository(t, a, "demo").List()
	if len(units) != 1 || units[0].Label != "Top 1A" || units[0].UnitType != unitTypeParking || units[0].BillableWeightPPM != 0 || units[0].MiteigentumsanteilPPM != 23456 || len(units[0].RenterEmails) != 0 {
		t.Fatalf("units after edit = %+v", units)
	}

	deleteUnit := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/building/units/delete", url.Values{"id": {"top-1"}})
	if deleteUnit.Code != http.StatusSeeOther {
		t.Fatalf("unit delete status = %d", deleteUnit.Code)
	}
	if units := testUnitRepository(t, a, "demo").List(); len(units) != 0 {
		t.Fatalf("units after delete = %+v", units)
	}
}

func TestSettingsHubAdminLinksManagementSections(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	rr := authedRequest(t, a, "admin@example.com", "/demo/app/settings")
	if rr.Code != http.StatusOK {
		t.Fatalf("settings hub status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{`href="/demo/app/settings/building"`, `href="/demo/app/settings/users"`, `href="/demo/app/settings/parking-access"`, `href="/demo/app/parking/settings"`, "Parkplatz-Abrechnung", "Parkplatz-Zugriff", "Gebäude"} {
		if !strings.Contains(body, want) {
			t.Fatalf("admin settings hub should contain %q", want)
		}
	}
}

func TestSettingsHubManagerLinksTenantManagementOnly(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	rr := authedRequest(t, a, "manager@example.com", "/demo/app/settings")
	if rr.Code != http.StatusOK {
		t.Fatalf("settings hub status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{`href="/demo/app/settings/building"`, `href="/demo/app/settings/users"`, `href="/demo/app/settings/parking-access"`, "Benutzer &amp; Rechte", "Parkplatz-Zugriff", "Gebäude"} {
		if !strings.Contains(body, want) {
			t.Fatalf("manager settings hub should contain %q", want)
		}
	}
	if strings.Contains(body, `href="/demo/app/parking/settings"`) {
		t.Fatal("manager settings hub must not expose parking accounting config")
	}
}

func TestUserRowsDoesNotInjectSyntheticEmptyRow(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles = map[string]userProfile{}
	a.allowed = map[string]struct{}{}
	a.admins = map[string]struct{}{}

	rows := a.userRows(testTenantRef("demo"))
	if len(rows) != 0 {
		t.Fatalf("empty userRows = %+v, want no synthetic rows", rows)
	}
}

func TestRoleManagementUIOffersAllEffectiveRoles(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	for _, profile := range []userProfile{
		{Email: "owner@example.com", FirstName: "Eva", LastName: "Owner", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()},
		{Email: "renter@example.com", FirstName: "Max", LastName: "Renter", Role: roleRenter, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()},
		{Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()},
		{Email: "board@example.com", FirstName: "Berta", LastName: "Beirat", Role: roleBeirat, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()},
	} {
		a.profiles[profile.Email] = profile
	}

	rr := authedRequest(t, a, "admin@example.com", "/demo/app/settings/users")
	if rr.Code != http.StatusOK {
		t.Fatalf("user settings status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{
		`value="Mieter"`, `value="Eigentümer"`, `value="Beirat"`, `value="Verwalter"`, `value="Admin"`,
		"role-owner", "role-renter", "role-manager", "role-beirat",
		"Eigentümerzugriff", "Abstimmungen", "Aushang", "Sonderrechte",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("role management UI should contain %q", want)
		}
	}
}

func TestRoleManagementUIOffersPermissionCheckboxesAndPresets(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "parker@example.com", FirstName: "Pat", LastName: "Parker", Role: roleRenter, Tenants: []string{"demo"}, Permissions: []string{permissionParking}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("Add invite: %v", err)
	}

	rr := authedRequest(t, a, "admin@example.com", "/demo/app/settings/users")
	if rr.Code != http.StatusOK {
		t.Fatalf("user settings status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{
		`name="permissions" value="parking"`,
		`data-permission="parking"`,
		`data-preset-permissions="parking"`,
		"Parkplatznutzung",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("permission UI should contain %q", want)
		}
	}
	parker := strings.Index(body, `name="orig_email" value="parker@example.com"`)
	if parker < 0 {
		t.Fatal("parker edit dialog should be present")
	}
	parking := strings.Index(body[parker:], `name="permissions" value="parking"`)
	if parking < 0 {
		t.Fatal("saved parking permission input should be present in the parker edit dialog")
	}
	parking += parker
	inputEnd := strings.Index(body[parking:], ">")
	if inputEnd < 0 || !strings.Contains(body[parking:parking+inputEnd], "checked") {
		t.Fatal("saved parking permission should remain checked")
	}
}

func TestInviteCreatedWithParkingPermissionGrantsParkingAccess(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	values := url.Values{
		"email":       {"parker@example.com"},
		"first_name":  {"Pat"},
		"last_name":   {"Parker"},
		"role":        {"Mieter"},
		"permissions": {permissionParking},
	}
	create := authedFormRequest(t, a, "admin@example.com", "/demo/app/settings/users", values)
	if create.Code != http.StatusSeeOther {
		t.Fatalf("create invite status = %d, want redirect", create.Code)
	}
	profile, ok := a.inviteStore.Get("parker@example.com")
	if !ok || !profile.HasPermission(permissionParking) {
		t.Fatalf("stored invite = %+v ok=%v, want parking permission", profile, ok)
	}
	parking := authedRequest(t, a, "parker@example.com", "/demo/app/parking")
	if parking.Code != http.StatusOK {
		t.Fatalf("parking status = %d, want 200 for invited parking user", parking.Code)
	}
}

func TestEditInviteCanRevokeParkingPermission(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "parker@example.com", FirstName: "Pat", LastName: "Parker", Role: roleRenter, Tenants: []string{"demo"}, Permissions: []string{permissionParking}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("Add invite: %v", err)
	}
	values := url.Values{
		"orig_email": {"parker@example.com"},
		"email":      {"parker@example.com"},
		"first_name": {"Pat"},
		"last_name":  {"Parker"},
		"role":       {"Mieter"},
	}
	edit := authedFormRequest(t, a, "admin@example.com", "/demo/app/settings/users/edit", values)
	if edit.Code != http.StatusSeeOther {
		t.Fatalf("edit invite status = %d, want redirect", edit.Code)
	}
	profile, ok := a.inviteStore.Get("parker@example.com")
	if !ok || profile.HasPermission(permissionParking) {
		t.Fatalf("stored invite = %+v ok=%v, want parking revoked", profile, ok)
	}
	parking := authedRequest(t, a, "parker@example.com", "/demo/app/parking")
	if parking.Code != http.StatusNotFound {
		t.Fatalf("parking status = %d, want 404 after parking revoke", parking.Code)
	}
}

func TestParkingAccessPageGrantsAndRevokesInvitePermission(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "parker@example.com", FirstName: "Pat", LastName: "Parker", Role: roleRenter, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("Add invite: %v", err)
	}

	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/parking-access")
	if page.Code != http.StatusOK {
		t.Fatalf("parking access status = %d, want 200", page.Code)
	}
	body := page.Body.String()
	for _, want := range []string{"Parkplatz verwalten", "parker@example.com", "Freigeben", "Wer darf den Parkplatz nutzen?"} {
		if !strings.Contains(body, want) {
			t.Fatalf("parking access page missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, `/demo/app/parking/settings?section=`) {
		t.Fatal("user manager must not see links to admin-only parking configuration")
	}

	grant := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/parking-access", url.Values{
		"email":   {"parker@example.com"},
		"parking": {"1"},
	})
	if grant.Code != http.StatusSeeOther {
		t.Fatalf("grant status = %d, want redirect", grant.Code)
	}
	profile, ok := a.inviteStore.Get("parker@example.com")
	if !ok || !profile.HasPermission(permissionParking) {
		t.Fatalf("stored profile after grant = %+v ok=%v", profile, ok)
	}
	if parking := authedRequest(t, a, "parker@example.com", "/demo/app/parking"); parking.Code != http.StatusOK {
		t.Fatalf("parking status after grant = %d, want 200", parking.Code)
	}

	revoke := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/parking-access", url.Values{
		"email":   {"parker@example.com"},
		"parking": {"0"},
	})
	if revoke.Code != http.StatusSeeOther {
		t.Fatalf("revoke status = %d, want redirect", revoke.Code)
	}
	profile, ok = a.inviteStore.Get("parker@example.com")
	if !ok || profile.HasPermission(permissionParking) {
		t.Fatalf("stored profile after revoke = %+v ok=%v", profile, ok)
	}
	if parking := authedRequest(t, a, "parker@example.com", "/demo/app/parking"); parking.Code != http.StatusNotFound {
		t.Fatalf("parking status after revoke = %d, want 404", parking.Code)
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionInviteUpdate, Query: "parker", Limit: 10})
	if len(events) != 2 || events[0].Summary != "Parkplatz-Zugriff geändert" {
		t.Fatalf("parking access audit events = %+v", events)
	}
}

func TestParkingAccessCandidatesStayTenantScopedAndPreferNames(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["local@example.com"] = userProfile{Email: "local@example.com", FirstName: "Lina", LastName: "Lokal", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["other@example.com"] = userProfile{Email: "other@example.com", FirstName: "Oskar", LastName: "Anderes Haus", Role: roleOwner, Tenants: []string{"other-house"}, AuthMethods: defaultAuthMethods()}
	a.allowed["local@example.com"] = struct{}{}
	a.allowed["other@example.com"] = struct{}{}

	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/parking-access")
	if page.Code != http.StatusOK {
		t.Fatalf("parking access status = %d, want 200", page.Code)
	}
	body := page.Body.String()
	if strings.Contains(body, "other@example.com") || strings.Contains(body, "Oskar") {
		t.Fatalf("parking access leaked another tenant's candidate:\n%s", body)
	}
	if !strings.Contains(body, `<strong>Lina Lokal</strong><span>local@example.com</span>`) {
		t.Fatalf("parking access must render a real name as the primary label:\n%s", body)
	}
	if strings.Contains(body, `<strong>local@example.com</strong>`) {
		t.Fatalf("parking access rendered email as the primary label despite a real name:\n%s", body)
	}
}

func TestClosedServiceProviderParkingAccessIsReadOnlyForEffectiveTenantRole(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{
		Email:       "service@example.com",
		FirstName:   "Multi",
		LastName:    "Service",
		Role:        roleResident,
		Tenants:     []string{"demo", "haus-b"},
		AuthMethods: defaultAuthMethods(),
		TenantMemberships: map[string]tenantMembership{
			"demo":   {Role: roleServiceProvider},
			"haus-b": {Role: roleResident},
		},
	}); err != nil {
		t.Fatalf("Add service invite: %v", err)
	}

	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/parking-access")
	if page.Code != http.StatusOK {
		t.Fatalf("parking access status = %d, want 200", page.Code)
	}
	body := page.Body.String()
	if !strings.Contains(body, "service@example.com") || !strings.Contains(body, "Schreibgeschützt") {
		t.Fatalf("closed service provider should render read-only:\n%s", body)
	}
	if strings.Contains(body, `<input type="hidden" name="email" value="service@example.com">`) {
		t.Fatalf("closed service provider still has a parking mutation form:\n%s", body)
	}

	update := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/parking-access", url.Values{
		"email":   {"service@example.com"},
		"parking": {"1"},
	})
	if update.Code != http.StatusForbidden || !strings.Contains(update.Body.String(), "Dienstleister-Zugänge sind derzeit nicht verfügbar") {
		t.Fatalf("closed service parking update = %d %q, want clear 403", update.Code, update.Body.String())
	}
	edit := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/users/edit", url.Values{
		"orig_email": {"service@example.com"},
		"email":      {"service@example.com"},
		"first_name": {"Changed"},
		"role":       {roleResident},
	})
	if edit.Code != http.StatusForbidden || !strings.Contains(edit.Body.String(), "Dienstleister-Zugänge sind derzeit nicht verfügbar") {
		t.Fatalf("closed effective service edit = %d %q, want clear 403", edit.Code, edit.Body.String())
	}
	deleteResponse := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/users/delete", url.Values{
		"email": {"service@example.com"},
	})
	if deleteResponse.Code != http.StatusForbidden || !strings.Contains(deleteResponse.Body.String(), "Dienstleister-Zugänge sind derzeit nicht verfügbar") {
		t.Fatalf("closed effective service delete = %d %q, want clear 403", deleteResponse.Code, deleteResponse.Body.String())
	}
	profile, ok := a.inviteStore.Get("service@example.com")
	if !ok || profile.FirstName != "Multi" || profile.HasPermission(permissionParking) || profile.ForTenant("demo").HasPermission(permissionParking) {
		t.Fatalf("rejected service mutations changed profile: %+v ok=%v", profile, ok)
	}
	if profile.ForTenant("haus-b").Role != roleResident {
		t.Fatalf("unrelated tenant membership changed: %+v", profile.ForTenant("haus-b"))
	}
	if events := a.auditStore.List(auditFilter{TenantSlug: "demo", Query: "service@example.com", Limit: 10}); len(events) != 0 {
		t.Fatalf("rejected service mutations created audit events: %+v", events)
	}

	users := authedRequest(t, a, "manager@example.com", "/demo/app/settings/users")
	if users.Code != http.StatusOK {
		t.Fatalf("users status = %d, want 200", users.Code)
	}
	usersBody := users.Body.String()
	if !strings.Contains(usersBody, "service@example.com") || !strings.Contains(usersBody, "Schreibgeschützt") {
		t.Fatalf("closed service provider should render read-only in user settings:\n%s", usersBody)
	}
	if strings.Contains(usersBody, `data-edit="service@example.com"`) || strings.Contains(usersBody, `<input type="hidden" name="email" value="service@example.com">`) {
		t.Fatalf("closed multi-tenant service provider still has an edit or global delete control:\n%s", usersBody)
	}
}

func TestParkingAccessPageKeepsEnvUsersReadOnly(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["env-parker@example.com"] = userProfile{Email: "env-parker@example.com", FirstName: "Env", LastName: "Parker", Role: roleRenter, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}

	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/parking-access")
	if page.Code != http.StatusOK {
		t.Fatalf("parking access status = %d, want 200", page.Code)
	}
	body := page.Body.String()
	if !strings.Contains(body, "env-parker@example.com") || !strings.Contains(body, "Konfiguration") || !strings.Contains(body, "Schreibgeschützt") {
		t.Fatalf("env user should render read-only:\n%s", body)
	}

	grant := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/parking-access", url.Values{
		"email":   {"env-parker@example.com"},
		"parking": {"1"},
	})
	if grant.Code != http.StatusSeeOther || !strings.Contains(grant.Header().Get("Location"), "not_editable") {
		t.Fatalf("env grant redirect = %d %q", grant.Code, grant.Header().Get("Location"))
	}
	if a.profiles["env-parker@example.com"].HasPermission(permissionParking) {
		t.Fatal("env profile must not be mutated by parking access page")
	}
	if parking := authedRequest(t, a, "env-parker@example.com", "/demo/app/parking"); parking.Code != http.StatusNotFound {
		t.Fatalf("env parking status = %d, want 404", parking.Code)
	}
}

func TestNavigationActionsStayScopedToRelevantPages(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	portal := authedRequest(t, a, "admin@example.com", "/demo/app")
	if portal.Code != http.StatusOK {
		t.Fatalf("portal status = %d, want 200", portal.Code)
	}
	if strings.Contains(portal.Body.String(), "Aushang verwalten") {
		t.Fatal("portal should not duplicate the Aushang destination with an admin-only quick link")
	}

	access := authedRequest(t, a, "admin@example.com", "/demo/app/settings/parking-access")
	if access.Code != http.StatusOK {
		t.Fatalf("parking access status = %d, want 200", access.Code)
	}
	if strings.Contains(access.Body.String(), `action="/demo/app/parking/reminders"`) {
		t.Fatal("parking access should keep payment reminders in the accounting flow")
	}
	accounting := authedRequest(t, a, "admin@example.com", "/demo/app/parking/settings?section=accounting")
	if accounting.Code != http.StatusOK || !strings.Contains(accounting.Body.String(), `action="/demo/app/parking/reminders"`) {
		t.Fatalf("parking accounting should own the reminder action, status=%d", accounting.Code)
	}

	for _, tc := range []struct {
		name string
		path string
	}{
		{name: "building", path: "/demo/app/settings/building"},
		{name: "profile", path: "/demo/app/settings/profile"},
	} {
		rr := authedRequest(t, a, "admin@example.com", tc.path)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", tc.name, rr.Code)
		}
		if strings.Contains(rr.Body.String(), `action="/demo/app/parking/reminders"`) {
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
		TenantSlug: "demo",
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
	events := reopened.List(auditFilter{TenantSlug: "demo", Action: auditActionInviteCreate, Query: "resident", Limit: 10})
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
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	values := url.Values{
		"email":       {"new.resident@example.com"},
		"first_name":  {"New"},
		"last_name":   {"Resident"},
		"role":        {"Mieter"},
		"permissions": {permissionParking},
	}
	create := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/users", values)
	if create.Code != http.StatusSeeOther {
		t.Fatalf("create invite status = %d, want redirect", create.Code)
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionInviteCreate, Query: "new.resident", Limit: 10})
	if len(events) != 1 || events[0].ActorEmail != "manager@example.com" || events[0].TargetID != "new.resident@example.com" {
		t.Fatalf("invite audit events = %+v", events)
	}
	body := authedRequest(t, a, "manager@example.com", "/demo/app/audit")
	auditBody := body.Body.String()
	for _, want := range []string{"Einladung angelegt", "new.resident@example.com", "Aktivitätsverlauf", "audit-stream", "filter-panel", "audit-event", "detail-list", "audit-add", "audit-time"} {
		if !strings.Contains(auditBody, want) {
			t.Fatalf("manager audit page missing %q status/body = %d\n%s", want, body.Code, auditBody)
		}
	}
	if body.Code != http.StatusOK {
		t.Fatalf("manager audit page status/body = %d\n%s", body.Code, body.Body.String())
	}
	resident := authedRequest(t, a, "resident@example.com", "/demo/app/audit")
	if resident.Code != http.StatusOK {
		t.Fatalf("resident audit status = %d, want 200", resident.Code)
	}
	residentBody := resident.Body.String()
	if strings.Contains(residentBody, "new.resident@example.com") || strings.Contains(residentBody, "manager@example.com") {
		t.Fatalf("resident audit leaks unrelated management event:\n%s", residentBody)
	}
	if !strings.Contains(residentBody, "Interne Verwaltungsdetails bleiben geschützt") {
		t.Fatalf("resident audit scope explanation missing:\n%s", residentBody)
	}
	// Ohne sichtbaren Vorgang trägt die Seite den gestalteten Leerzustand: er
	// erklärt, was später hier steht, statt eine leere Fläche zu zeigen.
	for _, want := range []string{"audit-blank", "Noch nichts im Verlauf", "Was festgehalten wird", "Ihre Anmeldungen"} {
		if !strings.Contains(residentBody, want) {
			t.Fatalf("resident audit empty state missing %q:\n%s", want, residentBody)
		}
	}
	if strings.Contains(residentBody, "Zugänge &amp; Rollen") {
		t.Fatalf("resident audit empty state must not describe management scope:\n%s", residentBody)
	}
}

func TestDocumentUploadRecordsMetadataAndAudit(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	fields := map[string]string{
		"title":      "Hausordnung",
		"category":   documentCategoryRules,
		"visibility": documentVisibilityAllResidents,
	}
	upload := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/dokumente", fields, "document", "hausordnung.pdf", []byte("%PDF-1.4\n% weg portal test\n"))
	if upload.Code != http.StatusSeeOther {
		t.Fatalf("upload status = %d, want redirect", upload.Code)
	}
	docs := documentRepositoryForTest(a, "demo").List()
	if len(docs) != 1 || docs[0].Title != "Hausordnung" || docs[0].Visibility != documentVisibilityAllResidents {
		t.Fatalf("stored docs = %+v", docs)
	}
	if location := upload.Header().Get("Location"); location != "/demo/app/dokumente?doc=uploaded#document-"+docs[0].ID {
		t.Fatalf("upload redirect = %q, want created document anchor", location)
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionDocumentUpload, Limit: 10})
	if len(events) != 1 || events[0].TargetID != docs[0].ID || events[0].Details["title"] != "Hausordnung" {
		t.Fatalf("audit events = %+v", events)
	}
	page := authedRequest(t, a, "resident@example.com", "/demo/app/dokumente")
	if page.Code != http.StatusOK {
		t.Fatalf("resident documents status = %d, want 200", page.Code)
	}
	body := page.Body.String()
	for _, want := range []string{`href="/demo/app/dokumente"`, "Hausordnung", "Alle Bewohner", "Vorschau", "/preview"} {
		if !strings.Contains(body, want) {
			t.Fatalf("documents page missing %q", want)
		}
	}
	preview := authedRequest(t, a, "resident@example.com", "/demo/app/dokumente/"+docs[0].ID+"/preview")
	if preview.Code != http.StatusOK {
		t.Fatalf("document preview status = %d, want 200", preview.Code)
	}
	if !strings.Contains(preview.Header().Get("Content-Disposition"), "inline") {
		t.Fatalf("preview disposition = %q, want inline", preview.Header().Get("Content-Disposition"))
	}
	residentUpload := authedMultipartFileRequest(t, a, "resident@example.com", "/demo/app/dokumente", fields, "document", "resident.pdf", []byte("%PDF-1.4\n% weg portal test\n"))
	if residentUpload.Code != http.StatusForbidden {
		t.Fatalf("resident upload status = %d, want 403", residentUpload.Code)
	}
}

func TestDocumentsPageFiltersManagerOnlyMetadata(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	if _, err := documentRepositoryForTest(a, "demo").Create(documentRecord{
		TenantSlug: "demo",
		Title:      "Internes Protokoll",
		Category:   documentCategoryProtocol,
		Visibility: documentVisibilityManagerOnly,
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "intern.pdf", []byte("%PDF-1.4\n% intern\n")), time.Now()); err != nil {
		t.Fatalf("create manager-only doc: %v", err)
	}
	if _, err := documentRepositoryForTest(a, "demo").Create(documentRecord{
		TenantSlug: "demo",
		Title:      "Hausordnung",
		Category:   documentCategoryRules,
		Visibility: documentVisibilityAllResidents,
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "hausordnung.pdf", []byte("%PDF-1.4\n% public\n")), time.Now()); err != nil {
		t.Fatalf("create public doc: %v", err)
	}
	residentPage := authedRequest(t, a, "resident@example.com", "/demo/app/dokumente")
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
	managerPage := authedRequest(t, a, "manager@example.com", "/demo/app/dokumente")
	if !strings.Contains(managerPage.Body.String(), "Internes Protokoll") {
		t.Fatal("manager page should show manager-only document")
	}
}

func TestDocumentsPageSearchSortAndCategoryEmptyStates(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	oldDoc, err := documentRepositoryForTest(a, "demo").Create(documentRecord{
		TenantSlug: "demo",
		Title:      "Abrechnung 2025",
		Category:   documentCategoryBilling,
		Visibility: documentVisibilityAllResidents,
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "abrechnung-2025.pdf", []byte("%PDF-1.4\nold\n")), time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("create old doc: %v", err)
	}
	newDoc, err := documentRepositoryForTest(a, "demo").Create(documentRecord{
		TenantSlug: "demo",
		Title:      "Abrechnung 2026",
		Category:   documentCategoryBilling,
		Visibility: documentVisibilityAllResidents,
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "abrechnung-2026.pdf", []byte("%PDF-1.4\nnew\n")), time.Date(2026, 2, 5, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("create new doc: %v", err)
	}
	page := authedRequest(t, a, "manager@example.com", "/demo/app/dokumente?sort=oldest")
	body := page.Body.String()
	oldIndex := strings.Index(body, oldDoc.Title)
	newIndex := strings.Index(body, newDoc.Title)
	if oldIndex < 0 || newIndex < 0 || oldIndex > newIndex {
		t.Fatalf("oldest sort order not reflected: old=%d new=%d", oldIndex, newIndex)
	}
	if strings.Contains(body, "Keine passenden Dokumente in dieser Kategorie.") || strings.Count(body, `class="document-section"`) != 1 {
		t.Fatal("default document view should hide empty category sections")
	}
	if !strings.Contains(body, `value="" selected disabled>Kategorie wählen`) && !strings.Contains(body, `value="" disabled selected>Kategorie wählen`) {
		t.Fatal("document upload should require an explicit category")
	}
	filtered := authedRequest(t, a, "manager@example.com", "/demo/app/dokumente?q=2025")
	filteredBody := filtered.Body.String()
	if !strings.Contains(filteredBody, oldDoc.Title) || strings.Contains(filteredBody, newDoc.Title) {
		t.Fatalf("search filtering body = %s", filteredBody)
	}
}

func TestEmptyLibrariesHideToolsThatHaveNothingToSearch(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["manager@example.com"] = userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}

	documents := authedRequest(t, a, "resident@example.com", "/demo/app/dokumente").Body.String()
	if !strings.Contains(documents, "Noch keine Dokumente") || strings.Contains(documents, `class="document-toolbar"`) {
		t.Fatalf("empty document library should explain itself without search tools:\n%s", documents)
	}

	announcements := authedRequest(t, a, "resident@example.com", "/demo/app/announcements").Body.String()
	if !strings.Contains(announcements, "Noch keine Beiträge") || strings.Contains(announcements, `class="archive-tools"`) {
		t.Fatalf("empty announcement archive should explain itself without search tools:\n%s", announcements)
	}

	audit := authedRequest(t, a, "resident@example.com", "/demo/app/audit").Body.String()
	if !strings.Contains(audit, "Noch nichts im Verlauf") || strings.Contains(audit, `class="audit-filter-panel"`) {
		t.Fatalf("empty resident history should explain itself without filters:\n%s", audit)
	}

	board := authedRequest(t, a, "manager@example.com", "/demo/app/anliegen/board").Body.String()
	if !strings.Contains(board, "Keine Anliegen in der Liegenschaft") || strings.Contains(board, `class="board-tools"`) {
		t.Fatalf("empty issue board should explain itself without filters:\n%s", board)
	}
}

func TestDocumentDownloadEnforcesVisibilityAndAudits(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["owner@example.com"] = userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["renter@example.com"] = userProfile{Email: "renter@example.com", Role: roleRenter, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["beirat@example.com"] = userProfile{Email: "beirat@example.com", Role: roleBeirat, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{
		{ID: "top-1", TenantSlug: "demo", Label: "Top 1", MiteigentumsanteilPPM: 10000, OwnerEmails: []string{"owner@example.com"}, RenterEmails: []string{"renter@example.com"}},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	publicDoc, err := documentRepositoryForTest(a, "demo").Create(documentRecord{
		TenantSlug: "demo",
		Title:      "Hausordnung",
		Category:   documentCategoryRules,
		Visibility: documentVisibilityAllResidents,
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "hausordnung.pdf", []byte("%PDF-1.4\npublic\n")), time.Now())
	if err != nil {
		t.Fatalf("create public doc: %v", err)
	}
	ownerDoc, err := documentRepositoryForTest(a, "demo").Create(documentRecord{
		TenantSlug: "demo",
		Title:      "Top 1 Abrechnung",
		Category:   documentCategoryBilling,
		Visibility: documentVisibilityOwnersOnly,
		UnitID:     "top-1",
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "top-1.pdf", []byte("%PDF-1.4\nowner\n")), time.Now())
	if err != nil {
		t.Fatalf("create owner doc: %v", err)
	}
	managerDoc, err := documentRepositoryForTest(a, "demo").Create(documentRecord{
		TenantSlug: "demo",
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
		rr := authedRequest(t, a, tc.email, "/demo/app/dokumente/"+tc.doc.ID+"/download")
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
	events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionDocumentDownload, Limit: 50})
	if len(events) != authorized {
		t.Fatalf("download audit count = %d, want %d: %+v", len(events), authorized, events)
	}
}

func TestDocumentReplaceShowsHistoryAndDownloadAuditVersion(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	created, err := documentRepositoryForTest(a, "demo").Create(documentRecord{
		TenantSlug: "demo",
		Title:      "Hausordnung",
		Category:   documentCategoryRules,
		Visibility: documentVisibilityAllResidents,
		UploadedBy: "manager@example.com",
	}, testMultipartHeader(t, "document", "hausordnung-v1.pdf", []byte("%PDF-1.4\nv1\n")), time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("create doc: %v", err)
	}
	replace := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/dokumente/replace", map[string]string{"id": created.ID}, "document", "hausordnung-v2.pdf", []byte("%PDF-1.4\nv2\n"))
	if replace.Code != http.StatusSeeOther {
		t.Fatalf("replace status = %d, want redirect", replace.Code)
	}
	current := documentRepositoryForTest(a, "demo").ListCurrent()
	if len(current) != 1 || current[0].Version != 2 {
		t.Fatalf("current docs = %+v", current)
	}
	if location := replace.Header().Get("Location"); location != "/demo/app/dokumente?doc=replaced#document-"+current[0].ID {
		t.Fatalf("replace redirect = %q, want current document anchor", location)
	}
	page := authedRequest(t, a, "manager@example.com", "/demo/app/dokumente")
	body := page.Body.String()
	for _, want := range []string{"Version 2", "Versionsverlauf", "Version 1", "Neue Version hochladen"} {
		if !strings.Contains(body, want) {
			t.Fatalf("documents page missing %q", want)
		}
	}
	downloadOld := authedRequest(t, a, "manager@example.com", "/demo/app/dokumente/"+created.ID+"/download")
	if downloadOld.Code != http.StatusOK || !strings.Contains(downloadOld.Body.String(), "v1") {
		t.Fatalf("old version download status/body = %d %q", downloadOld.Code, downloadOld.Body.String())
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionDocumentDownload, Limit: 10})
	if len(events) != 1 || events[0].TargetID != created.ID || events[0].Details["version"] != "Version 1" {
		t.Fatalf("download audit events = %+v", events)
	}
	replaceEvents := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionDocumentReplace, Limit: 10})
	if len(replaceEvents) != 1 || replaceEvents[0].Details["version"] != "Version 2" || replaceEvents[0].Details["previous"] != "Version 1" {
		t.Fatalf("replace audit events = %+v", replaceEvents)
	}
}

func TestManagerCanManageTenantSurfacesButNotPlatformSettings(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	users := authedRequest(t, a, "manager@example.com", "/demo/app/settings/users")
	if users.Code != http.StatusOK {
		t.Fatalf("manager user settings status = %d, want 200", users.Code)
	}
	if !strings.Contains(users.Body.String(), "Benutzer &amp; Rechte") {
		t.Fatal("manager should see user management page")
	}
	if strings.Contains(users.Body.String(), `<option value="Admin"`) {
		t.Fatal("manager user management must not offer Admin role assignment")
	}
	adminInvite := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/users", url.Values{
		"email": {"new-admin@example.com"},
		"role":  {"Admin"},
	})
	if adminInvite.Code != http.StatusSeeOther {
		t.Fatalf("manager admin invite status = %d, want redirect", adminInvite.Code)
	}
	if _, ok := a.inviteStore.Get("new-admin@example.com"); ok {
		t.Fatal("manager must not be able to create an Admin invite")
	}
	if _, err := a.inviteStore.Add(userProfile{Email: "persisted-admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("Add admin invite: %v", err)
	}
	deleteAdmin := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/users/delete", url.Values{
		"email": {"persisted-admin@example.com"},
	})
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
	write := authedFormRequest(t, a, "manager@example.com", "/demo/app/announcements", values)
	if write.Code != http.StatusSeeOther {
		t.Fatalf("manager announcement create status = %d, want redirect", write.Code)
	}
	ballotDeadline := time.Now().Add(48 * time.Hour).In(time.Local).Format("2006-01-02T15:04")
	ballotCreate := authedFormRequest(t, a, "manager@example.com", "/demo/app/abstimmungen", url.Values{
		"title":                 {"Dachsanierung"},
		"description":           {"Beschluss zur Beauftragung"},
		"options_text":          {"Ja\nNein\nEnthaltung"},
		"type":                  {ballotTypeCircular},
		"weighting":             {ballotWeightingPerShare},
		"quorum_percent":        {"50"},
		"closes_at":             {ballotDeadline},
		"reminder_before_hours": {"12"},
	})
	if ballotCreate.Code != http.StatusSeeOther {
		t.Fatalf("manager ballot create status = %d, want redirect", ballotCreate.Code)
	}
	ballots := testVoteRepository(t, a, "demo").List()
	if len(ballots) != 1 || ballots[0].Title != "Dachsanierung" || ballots[0].QuorumPPM != 500000 || ballots[0].ReminderBeforeMinutes != 720 {
		t.Fatalf("created ballots = %+v", ballots)
	}
	if location := ballotCreate.Header().Get("Location"); location != "/demo/app/abstimmungen?vote=created#ballot-"+ballots[0].ID {
		t.Fatalf("manager ballot create redirect = %q, want ballot anchor", location)
	}
	open := authedFormRequest(t, a, "manager@example.com", "/demo/app/abstimmungen/open", url.Values{"id": {ballots[0].ID}})
	if open.Code != http.StatusSeeOther {
		t.Fatalf("manager ballot open status = %d, want redirect", open.Code)
	}
	if location := open.Header().Get("Location"); location != "/demo/app/abstimmungen?vote=opened#ballot-"+ballots[0].ID {
		t.Fatalf("manager ballot open redirect = %q, want ballot anchor", location)
	}
	close := authedFormRequest(t, a, "manager@example.com", "/demo/app/abstimmungen/close", url.Values{"id": {ballots[0].ID}})
	if close.Code != http.StatusSeeOther {
		t.Fatalf("manager ballot close status = %d, want redirect", close.Code)
	}
	if location := close.Header().Get("Location"); location != "/demo/app/abstimmungen?vote=closed#ballot-"+ballots[0].ID {
		t.Fatalf("manager ballot close redirect = %q, want ballot anchor", location)
	}
	closed, _ := testVoteRepository(t, a, "demo").Get(ballots[0].ID)
	if closed.Status != ballotStatusClosed {
		t.Fatalf("closed ballot = %+v", closed)
	}
	for _, action := range []string{auditActionVoteCreate, auditActionVoteOpen, auditActionVoteClose} {
		events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: action, Limit: 10})
		if len(events) != 1 {
			t.Fatalf("audit events for %s = %+v", action, events)
		}
	}
	parkingSettings := authedRequest(t, a, "manager@example.com", "/demo/app/parking/settings")
	if parkingSettings.Code != http.StatusForbidden {
		t.Fatalf("manager parking settings status = %d, want 403", parkingSettings.Code)
	}
}

func TestResidentCannotManageTenantUsers(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	users := authedRequest(t, a, "resident@example.com", "/demo/app/settings/users")
	if users.Code != http.StatusForbidden {
		t.Fatalf("resident user settings status = %d, want 403", users.Code)
	}
	ballotCreate := authedFormRequest(t, a, "resident@example.com", "/demo/app/abstimmungen", url.Values{
		"title":        {"Nicht erlaubt"},
		"options_text": {"Ja\nNein"},
	})
	if ballotCreate.Code != http.StatusForbidden {
		t.Fatalf("resident ballot create status = %d, want 403", ballotCreate.Code)
	}
}

func TestParkingSettingsIsParkingSpecificNotGlobalSettings(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	rr := authedRequest(t, a, "admin@example.com", "/demo/app/parking/settings")
	if rr.Code != http.StatusOK {
		t.Fatalf("parking settings status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{"Parkplatz verwalten", `href="/demo/app/settings"`, "Einstellungen", "Tarif &amp; Gültigkeit"} {
		if !strings.Contains(body, want) {
			t.Fatalf("parking settings should contain %q", want)
		}
	}
	if strings.Contains(body, "<h1>Einstellungen</h1>") {
		t.Fatal("parking settings page must not use generic Einstellungen heading")
	}
}

func TestParkingSettingsSeparatesAdminTasksIntoSections(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	tests := []struct {
		path   string
		active string
		wants  []string
		hides  []string
	}{
		{
			path:   "/demo/app/parking/settings?section=accounting",
			active: "accounting",
			wants:  []string{`id="abrechnung"`, "Tarif &amp; Gültigkeit", "Zahlungsstände prüfen", "Offene Zahlungen erinnern"},
			hides:  []string{`id="laderegelung"`, `id="telegram"`},
		},
		{
			path:   "/demo/app/parking/settings?section=charging",
			active: "charging",
			wants:  []string{`id="laderegelung"`, "Erweiterte Grenzwerte", "Regler-Status &amp; Ereignisse", "Laderegeln speichern"},
			hides:  []string{`id="abrechnung"`, `id="telegram"`},
		},
		{
			path:   "/demo/app/parking/settings?section=telegram",
			active: "telegram",
			wants:  []string{`id="telegram"`, "Telegram verbinden", "Code erzeugen", "Noch niemand verbunden"},
			hides:  []string{`id="abrechnung"`, `id="laderegelung"`},
		},
	}
	for _, test := range tests {
		page := authedRequest(t, a, "admin@example.com", test.path)
		if page.Code != http.StatusOK {
			t.Fatalf("%s status = %d", test.active, page.Code)
		}
		body := page.Body.String()
		activeLink := `class="pk-nav-item active" href="/demo/app/parking/settings?section=` + test.active + `"`
		if !strings.Contains(body, activeLink) {
			t.Fatalf("%s section missing active navigation", test.active)
		}
		for _, want := range test.wants {
			if !strings.Contains(body, want) {
				t.Fatalf("%s section missing %q", test.active, want)
			}
		}
		for _, hidden := range test.hides {
			if strings.Contains(body, hidden) {
				t.Fatalf("%s section exposes unrelated panel %q", test.active, hidden)
			}
		}
	}
}

func TestParkingSettingsSavesEffectiveTariffHistory(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	save := authedFormRequest(t, a, "admin@example.com", "/demo/app/parking/settings", url.Values{
		"effective_from":       {"2026-07-01"},
		"grid_fee_eur_per_kwh": {"0.12"},
		"base_fee_eur":         {"5.50"},
	})
	if save.Code != http.StatusSeeOther {
		t.Fatalf("parking tariff save status = %d, want redirect", save.Code)
	}
	data := a.parkingStore.TenantData("demo")
	tariff := parkingTariffAt(data.Settings, time.Date(2026, 7, 15, 0, 0, 0, 0, time.Local), time.Local)
	if tariff.EffectiveFrom != "2026-07-01" {
		t.Fatalf("saved tariff = %+v", tariff)
	}
	assertClose(t, tariff.GridFeeEURPerKWh, 0.12)
	assertClose(t, tariff.BaseFeeEUR, 5.50)
	events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionParkingSettings, Limit: 10})
	if len(events) != 1 || events[0].Details["base_fee"] != "5,50 €" || events[0].Details["effective_from"] == "" {
		t.Fatalf("parking tariff audit events = %+v", events)
	}
	page := authedRequest(t, a, "admin@example.com", "/demo/app/parking/settings")
	body := page.Body.String()
	for _, want := range []string{"Tarif", "2026-07-01", "0,120 €/kWh", "Basis 5,50 €"} {
		if !strings.Contains(body, want) {
			t.Fatalf("parking settings page missing %q:\n%s", want, body)
		}
	}
}

func TestAnnouncementStoreCRUDVisibleSortPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "announcements.json")
	backend, err := newAnnouncementStore(path)
	if err != nil {
		t.Fatalf("newAnnouncementStore: %v", err)
	}
	repository, _ := storepkg.BindAnnouncementRepository(backend, testTenantRef("demo"))
	otherRepository, _ := storepkg.BindAnnouncementRepository(backend, testTenantRef("other"))
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	expiredAt := now.Add(-time.Hour)
	future, err := repository.Create(announcement{TenantSlug: "demo", Title: "Future", Body: "Later", Category: "Info", PublishedAt: now.Add(time.Hour)})
	if err != nil || future.ID == "" {
		t.Fatalf("create future: item=%+v err=%v", future, err)
	}
	expired, _ := repository.Create(announcement{TenantSlug: "demo", Title: "Expired", Body: "Old", Category: "Info", PublishedAt: now.Add(-2 * time.Hour), ExpiresAt: &expiredAt})
	normal, _ := repository.Create(announcement{TenantSlug: "demo", Title: "Normal", Body: "Visible", Category: "Termin", PublishedAt: now.Add(-30 * time.Minute)})
	pinned, _ := repository.Create(announcement{TenantSlug: "demo", Title: "Pinned", Body: "Top", Category: "Dringend", Pinned: true, PublishedAt: now.Add(-2 * time.Hour)})
	_, _ = otherRepository.Create(announcement{TenantSlug: "other", Title: "Other", Body: "Hidden", Category: "Info", PublishedAt: now.Add(-time.Hour)})

	visible := repository.Visible(now)
	if len(visible) != 2 {
		t.Fatalf("visible len = %d, want 2 (future=%s expired=%s normal=%s pinned=%s)", len(visible), future.ID, expired.ID, normal.ID, pinned.ID)
	}
	if visible[0].Title != "Pinned" || visible[1].Title != "Normal" {
		t.Fatalf("visible order = %q, %q; want pinned first then recent", visible[0].Title, visible[1].Title)
	}

	updated := normal
	updated.Title = "Updated"
	if ok, err := repository.Update(normal.ID, updated); !ok || err != nil {
		t.Fatalf("update: ok=%v err=%v", ok, err)
	}
	if removed, err := repository.Delete(pinned.ID); !removed || err != nil {
		t.Fatalf("delete: removed=%v err=%v", removed, err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("store file mode = %v err=%v, want 0600", info.Mode().Perm(), err)
	}
	reopened, err := newAnnouncementStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	reopenedRepository, _ := storepkg.BindAnnouncementRepository(reopened, testTenantRef("demo"))
	all := reopenedRepository.List()
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
	storage, err := newAnnouncementReadStore(path)
	if err != nil {
		t.Fatalf("newAnnouncementReadStore: %v", err)
	}
	repository, ok := storepkg.BindAnnouncementReadRepository(storage, testTenantRef("demo"))
	if !ok {
		t.Fatal("bind announcement read repository")
	}
	seenAt := time.Date(2026, 7, 6, 12, 30, 0, 0, time.UTC)
	if err := repository.MarkSeen("Resident@Example.com", seenAt); err != nil {
		t.Fatalf("mark seen: %v", err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("read store file mode = %v err=%v, want 0600", info.Mode().Perm(), err)
	}
	reopened, err := newAnnouncementReadStore(path)
	if err != nil {
		t.Fatalf("reopen read store: %v", err)
	}
	reopenedRepository, ok := storepkg.BindAnnouncementReadRepository(reopened, testTenantRef("demo"))
	if !ok {
		t.Fatal("bind reopened announcement read repository")
	}
	if got := reopenedRepository.LastSeen("resident@example.com"); !got.Equal(seenAt) {
		t.Fatalf("last seen = %v, want %v", got, seenAt)
	}
}

func TestEventStoreCRUDUpcomingPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.json")
	backend, err := newEventStore(path)
	if err != nil {
		t.Fatalf("newEventStore: %v", err)
	}
	repository, _ := storepkg.BindEventRepository(backend, testTenantRef("demo"))
	otherRepository, _ := storepkg.BindEventRepository(backend, testTenantRef("other"))
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	past, err := repository.Create(houseEvent{TenantSlug: "demo", Title: "Alte Reinigung", Category: "Reinigung", StartsAt: now.AddDate(0, 0, -2)})
	if err != nil {
		t.Fatalf("create past: %v", err)
	}
	future, err := repository.Create(houseEvent{TenantSlug: "demo", Title: "Eigentümerversammlung", Category: "meeting", Location: "Hof", StartsAt: now.Add(48 * time.Hour)})
	if err != nil {
		t.Fatalf("create future: %v", err)
	}
	_, _ = otherRepository.Create(houseEvent{TenantSlug: "other", Title: "Other", Category: "Wartung", StartsAt: now.Add(24 * time.Hour)})

	upcoming := repository.Upcoming(now)
	if len(upcoming) != 1 || upcoming[0].ID != future.ID || upcoming[0].Category != "Eigentümerversammlung" {
		t.Fatalf("upcoming = %+v, want only normalized future event", upcoming)
	}

	updated := future
	updated.Title = "Versammlung aktualisiert"
	updated.StartsAt = now.Add(72 * time.Hour)
	if ok, err := repository.Update(future.ID, updated); !ok || err != nil {
		t.Fatalf("update: ok=%v err=%v", ok, err)
	}
	if removed, err := repository.Delete(past.ID); !removed || err != nil {
		t.Fatalf("delete past: removed=%v err=%v", removed, err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("store file mode = %v err=%v, want 0600", info.Mode().Perm(), err)
	}

	reopened, err := newEventStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	reopenedRepository, _ := storepkg.BindEventRepository(reopened, testTenantRef("demo"))
	events := reopenedRepository.List()
	if len(events) != 1 || events[0].Title != "Versammlung aktualisiert" {
		t.Fatalf("reopened events = %+v", events)
	}
}

func TestPortalUsesOneCalmStateWithoutPrototypeCopy(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	rr := authedRequest(t, a, "resident@example.com", "/demo/app")
	if rr.Code != http.StatusOK {
		t.Fatalf("portal status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, forbidden := range []string{"Willkommen im Prototyp", "Beispielmodule", "Nächste Ausbaustufe"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("portal must not contain placeholder copy %q", forbidden)
		}
	}
	for _, want := range []string{"Heute wartet nichts auf Sie.", "Heute ist nichts zu erledigen", "Alles im Blick", `calm-main`, `data-portal-section-landing`, `href="/demo/app/anliegen?new=1#issue-new"`, germanDateLong(time.Now().In(time.Local))} {
		if !strings.Contains(body, want) {
			t.Fatalf("calm portal should contain %q", want)
		}
	}
	// Die Begrüßung trägt jetzt das Datum. Der frühere Untertitel stand direkt
	// über der Überschrift "Was ist als Nächstes zu tun?" und sagte dasselbe
	// noch einmal, nur unschärfer.
	if strings.Contains(body, "Hier steht, was jetzt wichtig ist") {
		t.Fatal("portal hero should carry the date instead of a filler subtitle")
	}
	if strings.Contains(body, `class="empty-state"`) || strings.Contains(body, "Noch keine Beiträge") {
		t.Fatal("calm portal should not stack empty states")
	}
	if !strings.Contains(body, `calm-main`) || strings.Contains(body, `class="banner"`) {
		t.Fatal("portal should use the calm templ overview instead of the old banner")
	}
	// The shared hero owns the desktop action; the fixed mobile quick action
	// keeps the same destination for thumb reach.
	if got := strings.Count(body, `href="/demo/app/anliegen?new=1#issue-new"`); got != 2 {
		t.Fatalf("desktop hero and mobile quick action should expose the create path, got %d", got)
	}
	// Shared portal-shell CSS selectors are now in the external portal-shell.css file (HAUSV-549)
	if !strings.Contains(body, `/assets/portal-shell.css?v=`) {
		t.Fatal("portal must link to external portal-shell.css (HAUSV-549)")
	}
	// Page-specific dashboard selectors remain inline in PortalStyles()
	for _, want := range []string{`.disclosures{`, `.utility-links{`} {
		if !strings.Contains(body, want) {
			t.Fatalf("portal stylesheet should contain focused dashboard selector %q", want)
		}
	}
	for _, forbidden := range []string{`.status-card span {`, `.home-list-row span {`, `.home-attention-item:first-child`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("portal stylesheet must not contain obsolete dashboard selector %q", forbidden)
		}
	}
}

func TestPortalDigestAggregatesRoleScopedAttentionItems(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", FirstName: "Resi", LastName: "Dent", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["manager@example.com"] = userProfile{Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["other@example.com"] = userProfile{Email: "other@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	now := time.Now()
	_, _ = testRepositories(a, "demo").announcements.Create(announcement{TenantSlug: "demo", Title: "Liftwartung", Body: "Lift Freitag", Category: "Wartung", PublishedAt: now.Add(-time.Hour)})
	_, _ = testRepositories(a, "demo").events.Create(houseEvent{TenantSlug: "demo", Title: "Versammlung", Category: "Eigentümerversammlung", StartsAt: now.Add(48 * time.Hour)})
	_, _ = testRepositories(a, "demo").issues.Create(residentIssue{TenantSlug: "demo", AuthorEmail: "resident@example.com", AuthorName: "Resi Dent", Category: "Reparatur", Title: "Eigenes Anliegen", Body: "Offen", LocationType: issueLocationUnit, Status: issueStatusNew, Priority: issuePriorityNorm})
	_, _ = testRepositories(a, "demo").issues.Create(residentIssue{TenantSlug: "demo", AuthorEmail: "other@example.com", AuthorName: "Other", Category: "Reparatur", Title: "Privates Anliegen", Body: "Offen", LocationType: issueLocationUnit, Status: issueStatusNew, Priority: issuePriorityNorm})

	resident := authedRequest(t, a, "resident@example.com", "/demo/app").Body.String()
	for _, want := range []string{"Drei Dinge warten auf Sie.", "Liftwartung", "Eigenes Anliegen", "Versammlung"} {
		if !strings.Contains(resident, want) {
			t.Fatalf("resident digest should contain %q", want)
		}
	}
	if strings.Contains(resident, `href="/demo/app/anliegen/board"`) {
		t.Fatalf("resident digest must be role-scoped:\n%s", resident)
	}

	manager := authedRequest(t, a, "manager@example.com", "/demo/app").Body.String()
	for _, want := range []string{"2", "offene Anliegen", `href="/demo/app/anliegen/board/`} {
		if !strings.Contains(manager, want) {
			t.Fatalf("manager digest should contain %q", want)
		}
	}
}

// Desktop and mobile shells intentionally render the same live source data.
func TestPortalResponsiveShellsShareFocusItems(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	now := time.Now()
	if _, err := testRepositories(a, "demo").events.Create(houseEvent{TenantSlug: "demo", Title: "Dachbegehung", Category: "Sonstiges", StartsAt: now.Add(48 * time.Hour)}); err != nil {
		t.Fatalf("create event: %v", err)
	}
	if _, err := issueRepositoryForTest(a, "demo").Create(residentIssue{TenantSlug: "demo", AuthorEmail: "resident@example.com", AuthorName: "Resi Dent", Category: "Reparatur", Title: "Wasserdruck im Bad zu niedrig", Body: "Kaum Druck", LocationType: issueLocationUnit, Status: issueStatusNew, Priority: issuePriorityNorm}); err != nil {
		t.Fatalf("create issue: %v", err)
	}

	body := authedRequest(t, a, "resident@example.com", "/demo/app").Body.String()
	for _, item := range []string{"Wasserdruck im Bad zu niedrig", "Dachbegehung"} {
		if count := strings.Count(body, item); count < 1 {
			t.Fatalf("portal should name %q, got %d:\n%s", item, count, body)
		}
	}
	for _, forbidden := range []string{"Kein Termin eingetragen", "Mängel, Fragen und Vorschläge gehen hier direkt an die Verwaltung."} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("portal must not claim the house is empty while the item stands above: %q", forbidden)
		}
	}
}

func TestPortalDashboardShowsRoleScopedDocumentsAndParking(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["parker@example.com"] = userProfile{Email: "parker@example.com", Role: roleRenter, Tenants: []string{"demo"}, Permissions: []string{permissionParking}, AuthMethods: defaultAuthMethods()}
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}

	upload := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/dokumente", map[string]string{
		"title":      "Hausordnung",
		"category":   documentCategoryRules,
		"visibility": documentVisibilityAllResidents,
	}, "document", "hausordnung.pdf", []byte("%PDF-1.4\n% weg portal test\n"))
	if upload.Code != http.StatusSeeOther {
		t.Fatalf("upload status = %d, want redirect", upload.Code)
	}

	parker := authedRequest(t, a, "parker@example.com", "/demo/app").Body.String()
	for _, want := range []string{"Dokumente", "Parkplatznutzung", "Weitere Bereiche", "Alles im Blick"} {
		if !strings.Contains(parker, want) {
			t.Fatalf("parking user dashboard should contain %q", want)
		}
	}
	if strings.Contains(parker, "Hausordnung") {
		t.Fatal("dashboard should not duplicate the document library without a real unread state")
	}

	resident := authedRequest(t, a, "resident@example.com", "/demo/app").Body.String()
	if !strings.Contains(resident, "Dokumente") || strings.Contains(resident, "Hausordnung") {
		t.Fatal("resident dashboard should keep documents in navigation without duplicating arbitrary files")
	}
	if !strings.Contains(resident, "<title>Hausüberblick · Musterweg 1 · Bewohner</title>") || strings.Contains(resident, "<title>WEG Portal</title>") {
		t.Fatal("dashboard browser title should use the house name instead of the legacy product name")
	}
	if strings.Contains(resident, `href="/demo/app/parking"`) || strings.Contains(resident, `href="/demo/app/parking#`) || strings.Contains(resident, `Parkplatz öffnen`) {
		t.Fatal("resident without parking permission must not see parking dashboard links")
	}
}

func TestEventsPageCRUDAndDashboardAgenda(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", FirstName: "Resi", LastName: "Dent", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	start := time.Now().Add(48 * time.Hour).In(time.Local).Format("2006-01-02T15:04")

	create := authedFormRequest(t, a, "manager@example.com", "/demo/app/events", url.Values{
		"title":     {"Liftwartung"},
		"category":  {"Wartung"},
		"starts_at": {start},
		"location":  {"Stiegenhaus"},
		"body":      {"Lift außer Betrieb."},
	})
	if create.Code != http.StatusSeeOther {
		t.Fatalf("create event status = %d, want redirect", create.Code)
	}
	events := testRepositories(a, "demo").events.List()
	if len(events) != 1 || events[0].Title != "Liftwartung" || events[0].Category != "Wartung" {
		t.Fatalf("stored events = %+v", events)
	}
	if loc := create.Header().Get("Location"); loc != "/demo/app/events?event=created#event-"+events[0].ID {
		t.Fatalf("create event redirect = %q", loc)
	}

	page := authedRequest(t, a, "resident@example.com", "/demo/app/events")
	if page.Code != http.StatusOK {
		t.Fatalf("resident events status = %d", page.Code)
	}
	body := page.Body.String()
	for _, want := range []string{"Liftwartung", "Stiegenhaus", "Lift außer Betrieb.", "Als Nächstes", "Kalender abonnieren", "Details", `href="/demo/app/events"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("events page should contain %q", want)
		}
	}
	for _, forbidden := range []string{`data-dialog="event-create"`, `action="/demo/app/events/delete"`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("resident events page must not contain management control %q", forbidden)
		}
	}
	dashboard := authedRequest(t, a, "resident@example.com", "/demo/app")
	if dashboard.Code != http.StatusOK || !strings.Contains(dashboard.Body.String(), "Liftwartung") {
		t.Fatalf("dashboard should show upcoming event, status=%d body=\n%s", dashboard.Code, dashboard.Body.String())
	}

	editedStart := time.Now().Add(72 * time.Hour).In(time.Local).Format("2006-01-02T15:04")
	edit := authedFormRequest(t, a, "manager@example.com", "/demo/app/events/edit", url.Values{
		"id":        {events[0].ID},
		"title":     {"Hofreinigung"},
		"category":  {"Reinigung"},
		"starts_at": {editedStart},
		"location":  {"Hof"},
	})
	if edit.Code != http.StatusSeeOther {
		t.Fatalf("edit event status = %d, want redirect", edit.Code)
	}
	events = testRepositories(a, "demo").events.List()
	if len(events) != 1 || events[0].Title != "Hofreinigung" || events[0].Category != "Reinigung" || events[0].Location != "Hof" {
		t.Fatalf("edited events = %+v", events)
	}
	if loc := edit.Header().Get("Location"); loc != "/demo/app/events?event=updated#event-"+events[0].ID {
		t.Fatalf("edit event redirect = %q", loc)
	}

	deleteResp := authedFormRequest(t, a, "manager@example.com", "/demo/app/events/delete", url.Values{"id": {events[0].ID}})
	if deleteResp.Code != http.StatusSeeOther {
		t.Fatalf("delete event status = %d, want redirect", deleteResp.Code)
	}
	if got := testRepositories(a, "demo").events.Upcoming(time.Now()); len(got) != 0 {
		t.Fatalf("events after delete = %+v, want none", got)
	}
	empty := authedRequest(t, a, "resident@example.com", "/demo/app/events")
	if !strings.Contains(empty.Body.String(), "Noch keine kommenden Termine") {
		t.Fatalf("empty events page should show empty state:\n%s", empty.Body.String())
	}
}

func TestEventsPageKeepsPastEventsProgressiveAndDashboardPreviewsUpcomingOnly(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	now := time.Now()
	pastEnd := now.Add(-47 * time.Hour)
	items := []houseEvent{
		{TenantSlug: "demo", Title: "Vergangene Begehung", Category: "Sonstiges", StartsAt: now.Add(-48 * time.Hour), EndsAt: &pastEnd},
		{TenantSlug: "demo", Title: "Nächste Ablesung", Category: "Ablesung", StartsAt: now.Add(24 * time.Hour)},
		{TenantSlug: "demo", Title: "Spätere Wartung", Category: "Wartung", StartsAt: now.Add(72 * time.Hour)},
	}
	for _, item := range items {
		if _, err := testRepositories(a, "demo").events.Create(item); err != nil {
			t.Fatalf("create event: %v", err)
		}
	}

	page := authedRequest(t, a, "resident@example.com", "/demo/app/events").Body.String()
	for _, want := range []string{"Nächste Ablesung", "Spätere Wartung", "Vergangene Termine · 1", "Vergangene Begehung", `class="history"`} {
		if !strings.Contains(page, want) {
			t.Fatalf("events page should contain %q", want)
		}
	}

	// Der Hausüberblick zeigt die nächsten Termine als kurze Vorschau. Sie ist
	// bewusst auf die kommenden Einträge begrenzt: Vergangenes gehört in den
	// Verlauf der Terminseite und nicht auf den Einstieg.
	dashboard := authedRequest(t, a, "resident@example.com", "/demo/app").Body.String()
	for _, want := range []string{"Nächste Termine", "Nächste Ablesung", "Spätere Wartung"} {
		if !strings.Contains(dashboard, want) {
			t.Fatalf("dashboard should preview upcoming events and contain %q:\n%s", want, dashboard)
		}
	}
	if strings.Contains(dashboard, "Vergangene Begehung") {
		t.Fatalf("dashboard must not preview past events:\n%s", dashboard)
	}
}

func TestEventCreateShowsAttachmentPreview(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	start := time.Now().Add(48 * time.Hour).In(time.Local).Format("2006-01-02T15:04")

	create := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/events", map[string]string{
		"title":     "Hofbegehung",
		"category":  "Sonstiges",
		"starts_at": start,
		"body":      "Bitte Foto beachten.",
	}, "attachments", "hof.png", minimalPNG())
	if create.Code != http.StatusSeeOther {
		t.Fatalf("event create status = %d, want redirect", create.Code)
	}
	events := testRepositories(a, "demo").events.List()
	if len(events) != 1 {
		t.Fatalf("events = %+v", events)
	}
	attachments := attachmentRepositoryForTest(a, "demo").ListEntity("event", events[0].ID)
	if len(attachments) != 1 || attachments[0].ContentType != "image/png" {
		t.Fatalf("event attachments = %+v", attachments)
	}
	page := authedRequest(t, a, "resident@example.com", "/demo/app/events")
	body := page.Body.String()
	for _, want := range []string{"hof.png", `data-lightbox-src`, "Hofbegehung"} {
		if !strings.Contains(body, want) {
			t.Fatalf("events page should contain %q:\n%s", want, body)
		}
	}
}

func TestPortalListsRealAnnouncementsPinnedFirstWithoutDeadTiles(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	now := time.Now().Add(-2 * time.Hour)
	expiredAt := now.Add(time.Hour)
	_, _ = testRepositories(a, "demo").announcements.Create(announcement{TenantSlug: "demo", Title: "Normaler Hinweis", Body: "Nur im Beitrag 4711", Category: "Info", PublishedAt: now.Add(time.Hour)})
	_, _ = testRepositories(a, "demo").announcements.Create(announcement{TenantSlug: "demo", Title: "Fixierter Hinweis", Body: "Nur im Beitrag 4712", Category: "Dringend", Pinned: true, PublishedAt: now})
	_, _ = testRepositories(a, "demo").announcements.Create(announcement{TenantSlug: "demo", Title: "Alter Hinweis", Body: "Abgelaufen", Category: "Info", PublishedAt: now.Add(-time.Hour), ExpiresAt: &expiredAt})
	_, _ = testRepositories(a, "demo").announcements.Create(announcement{TenantSlug: "demo", Title: "Geplanter Hinweis", Body: "Zukunft", Category: "Info", PublishedAt: time.Now().Add(time.Hour)})

	rr := authedRequest(t, a, "resident@example.com", "/demo/app")
	if rr.Code != http.StatusOK {
		t.Fatalf("portal status = %d", rr.Code)
	}
	body := rr.Body.String()
	// Der Überblick nennt die aktuellen Aushänge beim Namen, fixierte zuerst.
	// Das ist der Zweck der Vorschau: erkennen, worum es geht, ohne den
	// Beitragstext zu wiederholen.
	pinnedIndex, regularIndex := strings.Index(body, "Fixierter Hinweis"), strings.Index(body, "Normaler Hinweis")
	if pinnedIndex < 0 || regularIndex < 0 {
		t.Fatalf("portal should preview the visible announcements:\n%s", body)
	}
	if pinnedIndex > regularIndex {
		t.Fatalf("portal should list pinned announcements first:\n%s", body)
	}
	if strings.Contains(body, "Nur im Beitrag 4711") || strings.Contains(body, "Nur im Beitrag 4712") {
		t.Fatalf("portal should preview titles without repeating announcement bodies:\n%s", body)
	}
	for _, forbidden := range []string{"Alter Hinweis", "Geplanter Hinweis", "info-card", `class="quick-row disabled"`, `class="quick-row" href="/demo/app/announcements"`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("portal must not contain %q", forbidden)
		}
	}
	if !strings.Contains(body, `href="/demo/app/announcements"`) || !strings.Contains(body, "Neue Aushänge lesen") {
		t.Fatal("portal should keep one focused unread-announcement action")
	}
}

func TestAnnouncementCreateShowsAttachmentPreview(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}

	create := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/announcements", map[string]string{
		"title":    "Wirtschaftsplan",
		"body":     "Bitte beachten.",
		"category": "Info",
	}, "attachments", "wirtschaftsplan.pdf", []byte("%PDF-1.4\n% weg portal test\n"))
	if create.Code != http.StatusSeeOther {
		t.Fatalf("announcement create status = %d, want redirect", create.Code)
	}
	items := testRepositories(a, "demo").announcements.List()
	if len(items) != 1 {
		t.Fatalf("announcements = %+v", items)
	}
	attachments := attachmentRepositoryForTest(a, "demo").ListEntity("announcement", items[0].ID)
	if len(attachments) != 1 || attachments[0].ContentType != "application/pdf" {
		t.Fatalf("announcement attachments = %+v", attachments)
	}
	page := authedRequest(t, a, "resident@example.com", "/demo/app/announcements")
	body := page.Body.String()
	if !strings.Contains(body, "wirtschaftsplan.pdf") || strings.Contains(body, `action="/demo/app/attachments/delete"`) {
		t.Fatalf("resident announcement page attachment rendering mismatch:\n%s", body)
	}
}

func TestIssuesPageRendersResidentFormAndNav(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	rr := authedRequest(t, a, "resident@example.com", "/demo/app/anliegen")
	if rr.Code != http.StatusOK {
		t.Fatalf("issues status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{`href="/demo/app/anliegen"`, "Anliegen", "Erstes Anliegen melden", `enctype="multipart/form-data"`, `name="category"`, `name="location_type"`, `name="attachments"`, `multiple`} {
		if !strings.Contains(body, want) {
			t.Fatalf("issues page should contain %q", want)
		}
	}
	if strings.Contains(body, `class="nav-item disabled"><span class="nav-icon"><svg viewBox="0 0 24 24"><path d="M5 18.5`) {
		t.Fatal("Anliegen nav item must be a live link, not a disabled placeholder")
	}
	for _, duplicate := range []string{`class="issue-stats"`, `class="issue-tabs"`, "Kalender abonnieren"} {
		if strings.Contains(body, duplicate) {
			t.Fatalf("resident issue flow should not render duplicate or unrelated control %q", duplicate)
		}
	}
}

func TestIssueTitleFallbackIsServerOwnedAndBounded(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "first sentence", body: "  Licht im Keller defekt. Bitte prüfen.  ", want: "Licht im Keller defekt"},
		{name: "question", body: "Warum läuft die Lüftung? Bitte um Rückmeldung.", want: "Warum läuft die Lüftung"},
		{name: "spaced abbreviation", body: "z. B. flackert das Licht im Keller. Bitte prüfen.", want: "z. B. flackert das Licht im Keller"},
		{name: "compact abbreviation", body: "z.B. bleibt das Licht im Keller an. Bitte prüfen.", want: "z.B. bleibt das Licht im Keller an"},
		{name: "normalised whitespace", body: "Tür\n  im   Hof klemmt", want: "Tür im Hof klemmt"},
		{name: "unicode hard cut", body: strings.Repeat("ä", 90), want: strings.Repeat("ä", 75) + "…"},
		{name: "empty", body: " \n\t ", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := issueTitleFromBody(test.body); got != test.want {
				t.Fatalf("issueTitleFromBody() = %q, want %q", got, test.want)
			}
			if got := len([]rune(issueTitleFromBody(test.body))); got > 76 {
				t.Fatalf("derived title has %d runes, want at most 76", got)
			}
		})
	}

	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	created := authedMultipartFilesRequest(t, a, "resident@example.com", "/demo/app/anliegen", map[string]string{
		"category":        "Frage",
		"location_type":   issueLocationUnit,
		"location_detail": "Vorraum",
		"body":            "Warum läuft die Lüftung? Bitte um Rückmeldung.",
	}, nil)
	if created.Code != http.StatusSeeOther {
		t.Fatalf("create without title status = %d", created.Code)
	}
	issues := issueRepositoryForTest(a, "demo").ListAuthor("resident@example.com")
	if len(issues) != 1 || issues[0].Title != "Warum läuft die Lüftung" {
		t.Fatalf("server-derived issue = %+v", issues)
	}
	if loc := created.Header().Get("Location"); loc != "/demo/app/anliegen/"+issues[0].ID+"?created=1" {
		t.Fatalf("direct detail redirect = %q", loc)
	}

	limitTitle := strings.Repeat("T", 140)
	limits := authedMultipartFilesRequest(t, a, "resident@example.com", "/demo/app/anliegen", map[string]string{
		"category":      "Vorschlag",
		"location_type": issueLocationUnit,
		"title":         limitTitle,
		"body":          strings.Repeat("B", 4000),
	}, nil)
	if limits.Code != http.StatusSeeOther {
		t.Fatalf("exact issue limits status = %d", limits.Code)
	}
	foundLimit := false
	for _, issue := range issueRepositoryForTest(a, "demo").ListAuthor("resident@example.com") {
		if issue.Title == limitTitle && len([]rune(issue.Body)) == 4000 {
			foundLimit = true
			break
		}
	}
	if !foundLimit {
		t.Fatal("exact 140-rune title and 4000-rune body were not preserved")
	}

	invalid := authedMultipartFilesRequest(t, a, "resident@example.com", "/demo/app/anliegen", map[string]string{
		"category":      "Frage",
		"location_type": issueLocationUnit,
		"body":          strings.Repeat("x", 4001),
	}, nil)
	if invalid.Code != http.StatusSeeOther || invalid.Header().Get("Location") != "/demo/app/anliegen?issue=invalid" {
		t.Fatalf("overlong body response = %d %q", invalid.Code, invalid.Header().Get("Location"))
	}
}

func TestResidentCanSubmitIssueWithPhoto(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", FirstName: "Resi", LastName: "Dent", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	attachmentDir := filepath.Join(t.TempDir(), "issue-attachments")
	store, err := newIssueStore(filepath.Join(t.TempDir(), "issues.json"), attachmentDir)
	if err != nil {
		t.Fatalf("newIssueStore: %v", err)
	}
	a.issueStore = store
	managedAttachmentDir := filepath.Join(t.TempDir(), "attachments")
	managedAttachments, err := newAttachmentStore(filepath.Join(t.TempDir(), "attachments.json"), managedAttachmentDir)
	if err != nil {
		t.Fatalf("newAttachmentStore: %v", err)
	}
	a.attachmentStore = managedAttachments

	rr := authedMultipartRequest(t, a, "resident@example.com", "/demo/app/anliegen", map[string]string{
		"category":        "Reparatur",
		"location_type":   issueLocationCommon,
		"location_detail": "Stiegenhaus",
		"title":           "Licht flackert",
		"body":            "Das Licht im Stiegenhaus flackert seit gestern.",
	}, "licht.png", minimalPNG())
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("submit issue status = %d, want redirect", rr.Code)
	}
	if loc := rr.Header().Get("Location"); !strings.HasPrefix(loc, "/demo/app/anliegen/") || !strings.HasSuffix(loc, "?created=1") {
		t.Fatalf("redirect = %q", loc)
	}
	issueRepository, _ := storepkg.BindIssueRepository(store, testTenantRef("demo"))
	issues := issueRepository.ListAuthor("resident@example.com")
	if len(issues) != 1 {
		t.Fatalf("stored issues = %+v", issues)
	}
	issue := issues[0]
	if loc := rr.Header().Get("Location"); loc != "/demo/app/anliegen/"+issue.ID+"?created=1" {
		t.Fatalf("redirect = %q, want direct resident detail", loc)
	}
	if issue.Category != "Reparatur" || issue.LocationType != issueLocationCommon || issue.LocationDetail != "Stiegenhaus" || issue.Status != issueStatusOpen {
		t.Fatalf("stored issue fields = %+v", issue)
	}
	if len(issue.PhotoPaths) != 0 {
		t.Fatalf("legacy photo paths = %+v, want none", issue.PhotoPaths)
	}
	attachmentRepository, _ := storepkg.BindAttachmentRepository(managedAttachments, testTenantRef("demo"))
	attachments := attachmentRepository.ListEntity("issue", issue.ID)
	if len(attachments) != 1 {
		t.Fatalf("managed attachments = %+v, want one", attachments)
	}
	if attachments[0].ContentType != "image/png" || attachments[0].ThumbFilename == "" || attachments[0].PreviewFilename == "" {
		t.Fatalf("attachment metadata = %+v", attachments[0])
	}
	photoPath, _, _, ok := attachmentRepository.FilePath(attachments[0], "")
	if !ok {
		t.Fatal("managed attachment file path missing")
	}
	info, err := os.Stat(photoPath)
	if err != nil {
		t.Fatalf("stat photo: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("photo mode = %v, want 0600", info.Mode().Perm())
	}
	thumbPath, _, _, ok := attachmentRepository.FilePath(attachments[0], "thumb")
	if !ok {
		t.Fatal("managed thumbnail file path missing")
	}
	if _, err := os.Stat(thumbPath); err != nil {
		t.Fatalf("stat thumbnail: %v", err)
	}
	servedThumb := authedRequest(t, a, "resident@example.com", "/demo/app/attachments/"+attachments[0].ID+"/thumb")
	if servedThumb.Code != http.StatusOK {
		t.Fatalf("serve thumbnail status = %d", servedThumb.Code)
	}
	if ct := servedThumb.Header().Get("Content-Type"); !strings.HasPrefix(ct, "image/jpeg") {
		t.Fatalf("thumbnail content type = %q", ct)
	}

	page := authedRequest(t, a, "resident@example.com", "/demo/app/anliegen/"+issue.ID+"?created=1")
	if !strings.Contains(page.Body.String(), "Anliegen gemeldet") || !strings.Contains(page.Body.String(), "Ihre Meldung") || !strings.Contains(page.Body.String(), "Licht flackert") || !strings.Contains(page.Body.String(), "1 Foto") || !strings.Contains(page.Body.String(), `data-lightbox-src`) {
		t.Fatalf("issues page should show submitted issue with photo count:\n%s", page.Body.String())
	}
	if strings.Contains(page.Body.String(), "Neuigkeiten zum Anliegen") {
		t.Fatal("a new issue must not render an empty updates disclosure")
	}
}

// HAUSV-175: a photo that predates the attachment store must still be visible
// on its issue AFTER the migration moves it. This replaces the old test for the
// legacy render path, which is gone along with that path — what matters to a
// user is unchanged: the photo is still there.
func TestMigratedLegacyIssuePhotoStillRendersOnTheIssue(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	dir := t.TempDir()
	legacyDir := filepath.Join(dir, "issue-attachments")
	issues, err := newIssueStore(filepath.Join(dir, "issues.json"), legacyDir)
	if err != nil {
		t.Fatalf("newIssueStore: %v", err)
	}
	a.issueStore = issues
	attachments, err := newAttachmentStore(filepath.Join(dir, "attachments.json"), filepath.Join(dir, "files"))
	if err != nil {
		t.Fatalf("newAttachmentStore: %v", err)
	}
	a.attachmentStore = attachments

	if err := os.MkdirAll(filepath.Join(legacyDir, "demo"), 0o755); err != nil {
		t.Fatalf("mkdir legacy photo dir: %v", err)
	}
	legacyFilename := "legacy-1-photo.png"
	if err := os.WriteFile(filepath.Join(legacyDir, "demo", legacyFilename), minimalPNG(), 0o600); err != nil {
		t.Fatalf("write legacy photo: %v", err)
	}
	issueRepository, _ := storepkg.BindIssueRepository(issues, testTenantRef("demo"))
	if _, err := issueRepository.Create(residentIssue{
		ID:           "legacy-1",
		TenantSlug:   "demo",
		AuthorEmail:  "resident@example.com",
		Category:     "Reparatur",
		Title:        "Altes Foto",
		Body:         "Wurde vor der Attachment-Migration angelegt.",
		LocationType: issueLocationCommon,
		Status:       issueStatusNew,
		Priority:     issuePriorityNorm,
		PhotoPaths:   []string{filepath.ToSlash(filepath.Join(filepath.Base(legacyDir), "demo", legacyFilename))},
	}); err != nil {
		t.Fatalf("create legacy issue: %v", err)
	}

	n, err := migrateLegacyIssuePhotos(issues, attachments, legacyDir, []storepkg.TenantRef{testTenantRef("demo")}, time.Now())
	if err != nil || n != 1 {
		t.Fatalf("migrate: n=%d err=%v", n, err)
	}

	page := authedRequest(t, a, "resident@example.com", "/demo/app/anliegen/legacy-1")
	body := page.Body.String()
	for _, want := range []string{"Altes Foto", "1 Foto", legacyFilename, "data-lightbox-src="} {
		if !strings.Contains(body, want) {
			t.Fatalf("the migrated photo should still render, missing %q:\n%s", want, body)
		}
	}
	// And the legacy route is gone.
	if gone := authedRequest(t, a, "resident@example.com", "/demo/app/anliegen/legacy-1/photos/0"); gone.Code == http.StatusOK {
		t.Fatal("the legacy photo route should no longer exist")
	}
}
func TestResidentCanSubmitIssueWithMultipleAttachments(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	rr := authedMultipartFilesRequest(t, a, "resident@example.com", "/demo/app/anliegen", map[string]string{
		"category":      "Reparatur",
		"location_type": issueLocationCommon,
		"title":         "Wasser im Keller",
		"body":          "Im Keller steht Wasser.",
	}, []multipartTestFile{
		{Field: "attachments", Filename: "keller.png", Body: minimalPNG()},
		{Field: "attachments", Filename: "notiz.pdf", Body: []byte("%PDF-1.4\n% weg portal test\n")},
	})
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("submit issue status = %d, want redirect", rr.Code)
	}
	issues := issueRepositoryForTest(a, "demo").ListAuthor("resident@example.com")
	if len(issues) != 1 {
		t.Fatalf("stored issues = %+v", issues)
	}
	attachments := attachmentRepositoryForTest(a, "demo").ListEntity("issue", issues[0].ID)
	if len(attachments) != 2 {
		t.Fatalf("attachments = %+v, want two", attachments)
	}
	if attachments[0].ContentType != "image/png" || attachments[1].ContentType != "application/pdf" {
		t.Fatalf("attachment content types = %+v", attachments)
	}
	page := authedRequest(t, a, "resident@example.com", "/demo/app/anliegen/"+issues[0].ID)
	body := page.Body.String()
	for _, want := range []string{"keller.png", "notiz.pdf", "1 Foto", `data-lightbox-src`} {
		if !strings.Contains(body, want) {
			t.Fatalf("issues page should contain %q:\n%s", want, body)
		}
	}
}

func TestIssueAttachmentCreatorCanDelete(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	create := authedMultipartRequest(t, a, "resident@example.com", "/demo/app/anliegen", map[string]string{
		"category":      "Reparatur",
		"location_type": issueLocationCommon,
		"title":         "Tür klemmt",
		"body":          "Die Kellertür klemmt.",
	}, "tuer.png", minimalPNG())
	if create.Code != http.StatusSeeOther {
		t.Fatalf("create status = %d", create.Code)
	}
	issues := issueRepositoryForTest(a, "demo").ListAuthor("resident@example.com")
	if len(issues) != 1 {
		t.Fatalf("issues = %+v", issues)
	}
	attachments := attachmentRepositoryForTest(a, "demo").ListEntity("issue", issues[0].ID)
	if len(attachments) != 1 {
		t.Fatalf("attachments = %+v", attachments)
	}
	path, _, _, ok := attachmentRepositoryForTest(a, "demo").FilePath(attachments[0], "")
	if !ok {
		t.Fatal("attachment file path missing")
	}
	thumb := authedRequest(t, a, "resident@example.com", "/demo/app/attachments/"+attachments[0].ID+"/thumb")
	if thumb.Code != http.StatusOK {
		t.Fatalf("thumb status = %d", thumb.Code)
	}
	if events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionAttachmentView}); len(events) != 0 {
		t.Fatalf("thumbnail request must not create audit noise: %+v", events)
	}
	view := authedRequest(t, a, "resident@example.com", "/demo/app/attachments/"+attachments[0].ID)
	if view.Code != http.StatusOK {
		t.Fatalf("attachment view status = %d", view.Code)
	}
	delete := authedFormRequest(t, a, "resident@example.com", "/demo/app/attachments/delete", url.Values{"id": {attachments[0].ID}})
	if delete.Code != http.StatusSeeOther {
		t.Fatalf("delete status = %d", delete.Code)
	}
	if got := attachmentRepositoryForTest(a, "demo").ListEntity("issue", issues[0].ID); len(got) != 0 {
		t.Fatalf("attachments after delete = %+v, want none", got)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("deleted attachment file stat err = %v, want not exist", err)
	}
	viewEvents := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionAttachmentView})
	deleteEvents := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionAttachmentDelete})
	if len(viewEvents) != 1 || len(deleteEvents) != 1 {
		t.Fatalf("attachment audit events view=%+v delete=%+v", viewEvents, deleteEvents)
	}
	for _, event := range append(viewEvents, deleteEvents...) {
		if event.TargetID != attachments[0].ID || event.Details["entity_id"] != issues[0].ID || event.Details["entity_type"] != "issue" {
			t.Fatalf("attachment audit target = %+v", event)
		}
		if strings.Contains(strings.Join(auditDetailValues(event.Details), " "), "tuer.png") {
			t.Fatalf("attachment audit must not retain filename: %+v", event)
		}
	}
}

func TestIssueSubmitRejectsInvalidPhotoType(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	store, err := newIssueStore(filepath.Join(t.TempDir(), "issues.json"), filepath.Join(t.TempDir(), "issue-attachments"))
	if err != nil {
		t.Fatalf("newIssueStore: %v", err)
	}
	a.issueStore = store

	rr := authedMultipartRequest(t, a, "resident@example.com", "/demo/app/anliegen", map[string]string{
		"category":      "Frage",
		"location_type": issueLocationUnit,
		"title":         "Dokument hochladen",
		"body":          "Wo soll ich das melden?",
	}, "not-a-photo.txt", []byte("plain text"))
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("invalid photo status = %d, want redirect", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/demo/app/anliegen?issue=photo" {
		t.Fatalf("redirect = %q", loc)
	}
	issueRepository, _ := storepkg.BindIssueRepository(store, testTenantRef("demo"))
	if got := issueRepository.ListAuthor("resident@example.com"); len(got) != 0 {
		t.Fatalf("invalid photo must not create issue, got %+v", got)
	}
}

func TestManagerCanUpdateIssueWorkflow(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:   "demo",
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

	update := authedFormRequest(t, a, "manager@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":             {issue.ID},
		"status":         {issueStatusProgress},
		"priority":       {issuePriorityUrgent},
		"assignee_email": {"manager@example.com"},
	})
	if update.Code != http.StatusSeeOther {
		t.Fatalf("manager workflow status = %d, want redirect", update.Code)
	}
	if loc := update.Header().Get("Location"); loc != "/demo/app/anliegen/board?issue=updated#issue-"+issue.ID {
		t.Fatalf("manager workflow redirect = %q", loc)
	}
	updated, ok := issueRepositoryForTest(a, "demo").Get(issue.ID)
	if !ok {
		t.Fatal("updated issue not found")
	}
	if updated.Status != issueStatusProgress || updated.Priority != issuePriorityUrgent || updated.AssigneeEmail != "manager@example.com" {
		t.Fatalf("updated issue = %+v", updated)
	}
	if len(updated.StatusHistory) != 1 || updated.StatusHistory[0].ActorEmail != "manager@example.com" || updated.StatusHistory[0].From != issueStatusNew || updated.StatusHistory[0].To != issueStatusProgress {
		t.Fatalf("status history = %+v", updated.StatusHistory)
	}

	page := authedRequest(t, a, "manager@example.com", "/demo/app/anliegen")
	body := page.Body.String()
	for _, want := range []string{"Anliegen", "Triage-Board"} {
		if !strings.Contains(body, want) {
			t.Fatalf("manager issues page should contain %q", want)
		}
	}
	if !strings.Contains(body, "Tür schließt nicht") || !strings.Contains(body, `class="issue-summary-row"`) {
		t.Fatal("manager overview should show the open issue in its compact summary")
	}
	if strings.Contains(body, `name="assignee_email"`) {
		t.Fatalf("manager overview should link to board instead of rendering workflow form")
	}

	board := authedRequest(t, a, "manager@example.com", "/demo/app/anliegen/board")
	boardBody := board.Body.String()
	for _, want := range []string{"Anliegen bearbeiten", "Tür schließt nicht", issuePriorityUrgent, `href="/demo/app/anliegen/board/` + issue.ID + `"`} {
		if !strings.Contains(boardBody, want) {
			t.Fatalf("manager issue board should contain %q", want)
		}
	}
	// HAUSV-706 adds a status-only move menu. Assignment and priority remain
	// hidden snapshots; the full editor still lives in the focused triage.
	assertIssueBoardStatusMenus(t, boardBody)
	if strings.Contains(boardBody, "Bearbeitung aktualisieren") {
		t.Fatal("manager issue board must keep the full editor in focused triage")
	}
	if !strings.Contains(boardBody, `<details class="board-tools">`) ||
		strings.Contains(boardBody, `<details class="board-tools" open>`) {
		t.Fatalf("inactive issue filters should be collapsed")
	}
	triage := authedRequest(t, a, "manager@example.com", "/demo/app/anliegen/board/"+issue.ID+"?step=1").Body.String()
	for _, want := range []string{"Schritt 1 von 2", "Wie dringend ist das Anliegen?", "Heute kümmern", "Diese Woche", "Kann warten"} {
		if !strings.Contains(triage, want) {
			t.Fatalf("focused issue triage should contain %q", want)
		}
	}
}

func TestManagerIssueTriageKeepsTheIssueContextAcrossBothSteps(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:   "demo",
		AuthorEmail:  "resident@example.com",
		AuthorName:   "Resident",
		Category:     "Reparatur",
		Title:        "Licht im Stiegenhaus",
		Body:         "Das Licht fällt immer wieder aus.",
		LocationType: issueLocationCommon,
		Status:       issueStatusNew,
		Priority:     issuePriorityNorm,
	})
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	resident := authedRequest(t, a, "resident@example.com", "/demo/app/anliegen/board/"+issue.ID)
	if resident.Code != http.StatusForbidden {
		t.Fatalf("resident triage status = %d, want forbidden", resident.Code)
	}

	stepOneRedirect := "/demo/app/anliegen/board/" + issue.ID + "?step=2"
	stepOne := authedFormRequest(t, a, "manager@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":             {issue.ID},
		"status":         {issueStatusNew},
		"priority":       {issuePriorityHigh},
		"assignee_email": {""},
		"redirect":       {stepOneRedirect},
	})
	if stepOne.Code != http.StatusSeeOther || stepOne.Header().Get("Location") != stepOneRedirect {
		t.Fatalf("triage step one redirect = %d %q", stepOne.Code, stepOne.Header().Get("Location"))
	}
	stepTwoPage := authedRequest(t, a, "manager@example.com", stepOneRedirect)
	for _, want := range []string{"Schritt 2 von 2", "Wer kümmert sich als Nächstes?", "Ich übernehme", "Noch offen lassen"} {
		if !strings.Contains(stepTwoPage.Body.String(), want) {
			t.Fatalf("triage step two should contain %q", want)
		}
	}

	doneRedirect := "/demo/app/anliegen/board/" + issue.ID + "?step=done"
	stepTwo := authedFormRequest(t, a, "manager@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":             {issue.ID},
		"status":         {issueStatusProgress},
		"priority":       {issuePriorityHigh},
		"assignee_email": {"manager@example.com"},
		"redirect":       {doneRedirect},
	})
	if stepTwo.Code != http.StatusSeeOther || stepTwo.Header().Get("Location") != doneRedirect {
		t.Fatalf("triage step two redirect = %d %q", stepTwo.Code, stepTwo.Header().Get("Location"))
	}
	updated, found := issueRepositoryForTest(a, "demo").Get(issue.ID)
	if !found || updated.Status != issueStatusProgress || updated.Priority != issuePriorityHigh || updated.AssigneeEmail != "manager@example.com" {
		t.Fatalf("triaged issue = %+v found=%v", updated, found)
	}
	done := authedRequest(t, a, "manager@example.com", doneRedirect)
	for _, want := range []string{"Der nächste Schritt ist festgelegt.", issuePriorityHigh, "manager@example.com", "Bewohner kontaktieren"} {
		if !strings.Contains(done.Body.String(), want) {
			t.Fatalf("triage confirmation should contain %q", want)
		}
	}
}

func TestIssueQuestionCreatesExactlyOneResidentAnswerTaskAndAuditKind(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", FirstName: "Resi", LastName: "Dent", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:    "demo",
		AuthorEmail:   "resident@example.com",
		AuthorName:    "Resi Dent",
		Category:      "Reparatur",
		Title:         "Kellerlicht defekt",
		Body:          "Das Licht ist ausgefallen.",
		LocationType:  issueLocationCommon,
		Status:        issueStatusProgress,
		Priority:      issuePriorityHigh,
		AssigneeEmail: "manager@example.com",
	})
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	managerPage := authedRequest(t, a, "manager@example.com", "/demo/app/anliegen/board/"+issue.ID)
	for _, want := range []string{"Was soll der Bewohner wissen?", "Information senden", "Rückfrage stellen", "Lösung zur Prüfung senden"} {
		if !strings.Contains(managerPage.Body.String(), want) {
			t.Fatalf("manager message step missing %q", want)
		}
	}

	sentRedirect := "/demo/app/anliegen/board/" + issue.ID + "?step=sent"
	question := authedFormRequest(t, a, "manager@example.com", "/demo/app/anliegen/comment", url.Values{
		"id":           {issue.ID},
		"body":         {"In welchem Stockwerk ist das Licht ausgefallen?"},
		"message_type": {issueCommentKindQuestion},
		"redirect":     {sentRedirect},
	})
	if question.Code != http.StatusSeeOther || question.Header().Get("Location") != sentRedirect {
		t.Fatalf("question redirect = %d %q", question.Code, question.Header().Get("Location"))
	}
	stored, found := issueRepositoryForTest(a, "demo").Get(issue.ID)
	if !found || len(stored.Comments) != 1 || stored.Comments[0].Kind != issueCommentKindQuestion {
		t.Fatalf("stored question = %+v found=%v", stored.Comments, found)
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionIssueComment, Limit: 10})
	if len(events) != 1 || events[0].Details["message_type"] != issueCommentKindQuestion {
		t.Fatalf("question audit = %+v", events)
	}
	confirmation := authedRequest(t, a, "manager@example.com", sentRedirect).Body.String()
	if !strings.Contains(confirmation, "Der Bewohner sieht jetzt „Antworten“.") {
		t.Fatalf("question confirmation missing resident state")
	}

	overview := authedRequest(t, a, "resident@example.com", "/demo/app/anliegen").Body.String()
	for _, want := range []string{"Die Verwaltung braucht Ihre Antwort.", `href="/demo/app/anliegen/` + issue.ID + `"`, "Antworten"} {
		if !strings.Contains(overview, want) {
			t.Fatalf("resident overview missing %q", want)
		}
	}
	for _, unwanted := range []string{"Kommentar senden", "Als erledigt melden"} {
		if strings.Contains(overview, unwanted) {
			t.Fatalf("resident overview must not show generic action %q", unwanted)
		}
	}
	dashboard := authedRequest(t, a, "resident@example.com", "/demo/app").Body.String()
	for _, want := range []string{"Rückfrage beantworten", "Kellerlicht defekt", "Die Verwaltung braucht Ihre Antwort.", `href="/demo/app/anliegen/` + issue.ID + `"`, "Antworten"} {
		if !strings.Contains(dashboard, want) {
			t.Fatalf("resident dashboard missing focused answer task %q", want)
		}
	}
	detailURL := "/demo/app/anliegen/" + issue.ID
	detail := authedRequest(t, a, "resident@example.com", detailURL).Body.String()
	for _, want := range []string{"Rückfrage der Verwaltung", "In welchem Stockwerk ist das Licht ausgefallen?", `aria-label="Ihre Antwort"`, "Antwort senden", "Neuigkeiten zum Anliegen"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("resident answer task missing %q", want)
		}
	}

	answer := authedFormRequest(t, a, "resident@example.com", "/demo/app/anliegen/comment", url.Values{
		"id":           {issue.ID},
		"body":         {"Im zweiten Stock."},
		"message_type": {issueCommentKindQuestion},
		"redirect":     {detailURL},
	})
	if answer.Code != http.StatusSeeOther || answer.Header().Get("Location") != detailURL {
		t.Fatalf("answer redirect = %d %q", answer.Code, answer.Header().Get("Location"))
	}
	stored, _ = issueRepositoryForTest(a, "demo").Get(issue.ID)
	if len(stored.Comments) != 2 || stored.Comments[1].Kind != issueCommentKindAnswer {
		t.Fatalf("stored answer kind = %+v", stored.Comments)
	}
	afterAnswer := authedRequest(t, a, "resident@example.com", detailURL).Body.String()
	if strings.Contains(afterAnswer, `aria-label="Ihre Antwort"`) || !strings.Contains(afterAnswer, "Sie müssen im Moment nichts tun.") {
		t.Fatalf("answered question should return to waiting state")
	}

	info := authedFormRequest(t, a, "manager@example.com", "/demo/app/anliegen/comment", url.Values{
		"id":           {issue.ID},
		"body":         {"Der Elektriker ist informiert."},
		"message_type": {issueCommentKindInformation},
		"redirect":     {sentRedirect},
	})
	if info.Code != http.StatusSeeOther {
		t.Fatalf("information status = %d", info.Code)
	}
	stored, _ = issueRepositoryForTest(a, "demo").Get(issue.ID)
	if got := stored.Comments[len(stored.Comments)-1].Kind; got != issueCommentKindInformation {
		t.Fatalf("stored information kind = %q", got)
	}
	infoConfirmation := authedRequest(t, a, "manager@example.com", sentRedirect).Body.String()
	if !strings.Contains(infoConfirmation, "Der Bewohner muss darauf nicht reagieren.") {
		t.Fatalf("information confirmation should not request resident action")
	}
}

func TestResidentConfirmsOrRejectsProposedIssueResolution(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["other@example.com"] = userProfile{Email: "other@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	createDone := func(title string) residentIssue {
		item, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
			TenantSlug:   "demo",
			AuthorEmail:  "resident@example.com",
			AuthorName:   "Resident",
			Category:     "Reparatur",
			Title:        title,
			Body:         "Bitte prüfen.",
			LocationType: issueLocationUnit,
			Status:       issueStatusDone,
			Priority:     issuePriorityNorm,
		})
		if err != nil {
			t.Fatalf("Create issue: %v", err)
		}
		return item
	}

	confirmedIssue := createDone("Gelöste Tür")
	detailURL := "/demo/app/anliegen/" + confirmedIssue.ID
	detail := authedRequest(t, a, "resident@example.com", detailURL).Body.String()
	for _, want := range []string{"Ist das Anliegen für Sie erledigt?", "Ja, erledigt", "Nein, noch offen"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("resolution task missing %q", want)
		}
	}
	forbidden := authedFormRequest(t, a, "other@example.com", "/demo/app/anliegen/resolution", url.Values{
		"id":       {confirmedIssue.ID},
		"resolved": {"yes"},
	})
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("other resident resolution status = %d, want 403", forbidden.Code)
	}
	confirm := authedFormRequest(t, a, "resident@example.com", "/demo/app/anliegen/resolution", url.Values{
		"id":       {confirmedIssue.ID},
		"resolved": {"yes"},
	})
	if confirm.Code != http.StatusSeeOther || confirm.Header().Get("Location") != detailURL {
		t.Fatalf("confirm redirect = %d %q", confirm.Code, confirm.Header().Get("Location"))
	}
	confirmed, _ := issueRepositoryForTest(a, "demo").Get(confirmedIssue.ID)
	if confirmed.Status != issueStatusDone || confirmed.ResolutionConfirmedAt.IsZero() || confirmed.ResolutionConfirmedBy != "resident@example.com" {
		t.Fatalf("confirmed resolution = %+v", confirmed)
	}
	confirmedPage := authedRequest(t, a, "resident@example.com", detailURL).Body.String()
	if !strings.Contains(confirmedPage, "Sie haben die Lösung bestätigt.") || strings.Contains(confirmedPage, `value="no"`) {
		t.Fatalf("confirmed resolution should have no remaining decision")
	}

	reopenedIssue := createDone("Noch klemmende Tür")
	reopen := authedFormRequest(t, a, "resident@example.com", "/demo/app/anliegen/resolution", url.Values{
		"id":       {reopenedIssue.ID},
		"resolved": {"no"},
	})
	if reopen.Code != http.StatusSeeOther {
		t.Fatalf("reopen status = %d", reopen.Code)
	}
	reopened, _ := issueRepositoryForTest(a, "demo").Get(reopenedIssue.ID)
	if reopened.Status != issueStatusNew || !reopened.ResolutionConfirmedAt.IsZero() {
		t.Fatalf("reopened resolution = %+v", reopened)
	}
	reopenedPage := authedRequest(t, a, "resident@example.com", "/demo/app/anliegen/"+reopenedIssue.ID).Body.String()
	if !strings.Contains(reopenedPage, "Als Nächstes prüft die Verwaltung Ihre Meldung.") {
		t.Fatalf("reopened issue should return to waiting state")
	}
}

func TestResidentCanCloseAndReopenOwnIssueOnly(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	own, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:   "demo",
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
	other, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:   "demo",
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

	closeOwn := authedFormRequest(t, a, "resident@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":     {own.ID},
		"status": {issueStatusDone},
	})
	if closeOwn.Code != http.StatusSeeOther {
		t.Fatalf("resident close status = %d, want redirect", closeOwn.Code)
	}
	closed, _ := issueRepositoryForTest(a, "demo").Get(own.ID)
	if closed.Status != issueStatusDone {
		t.Fatalf("closed status = %q", closed.Status)
	}
	reopenOwn := authedFormRequest(t, a, "resident@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":     {own.ID},
		"status": {issueStatusNew},
	})
	if reopenOwn.Code != http.StatusSeeOther {
		t.Fatalf("resident reopen status = %d, want redirect", reopenOwn.Code)
	}
	reopened, _ := issueRepositoryForTest(a, "demo").Get(own.ID)
	if reopened.Status != issueStatusNew {
		t.Fatalf("reopened status = %q", reopened.Status)
	}

	otherUpdate := authedFormRequest(t, a, "resident@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":     {other.ID},
		"status": {issueStatusDone},
	})
	if otherUpdate.Code != http.StatusForbidden {
		t.Fatalf("resident other issue status = %d, want 403", otherUpdate.Code)
	}
	priorityUpdate := authedFormRequest(t, a, "resident@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":       {own.ID},
		"status":   {issueStatusDone},
		"priority": {issuePriorityUrgent},
	})
	if priorityUpdate.Code != http.StatusForbidden {
		t.Fatalf("resident priority update status = %d, want 403", priorityUpdate.Code)
	}
}

func TestIssueCommentsRenderAndNotify(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", FirstName: "Resi", LastName: "Dent", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["manager@example.com"] = userProfile{Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	mailer := &recordingMailer{}
	a.mailer = mailer

	create := authedMultipartRequest(t, a, "resident@example.com", "/demo/app/anliegen", map[string]string{
		"category":      "Frage",
		"location_type": issueLocationCommon,
		"title":         "Kommentar Test",
		"body":          "Bitte um Rückmeldung.",
	}, "", nil)
	if create.Code != http.StatusSeeOther {
		t.Fatalf("create status = %d", create.Code)
	}
	issues := issueRepositoryForTest(a, "demo").ListAuthor("resident@example.com")
	if len(issues) != 1 {
		t.Fatalf("issues = %+v", issues)
	}
	if len(mailer.notifications) != 1 || mailer.notifications[0].To != "manager@example.com" || !strings.Contains(mailer.notifications[0].Subject, "Neues Anliegen") || !strings.Contains(mailer.notifications[0].Body, "/demo/app/anliegen#issue-"+issues[0].ID) {
		t.Fatalf("new issue notifications = %+v", mailer.notifications)
	}

	managerComment := authedFormRequest(t, a, "manager@example.com", "/demo/app/anliegen/comment", url.Values{
		"id":   {issues[0].ID},
		"body": {"Ich prüfe das und melde mich."},
	})
	if managerComment.Code != http.StatusSeeOther {
		t.Fatalf("manager comment status = %d", managerComment.Code)
	}
	updated, _ := issueRepositoryForTest(a, "demo").Get(issues[0].ID)
	if len(updated.Comments) != 1 || updated.Comments[0].AuthorEmail != "manager@example.com" {
		t.Fatalf("comments after manager = %+v", updated.Comments)
	}
	if len(mailer.notifications) != 2 || mailer.notifications[1].To != "resident@example.com" || !strings.Contains(mailer.notifications[1].Subject, "Neuer Kommentar") || !strings.Contains(mailer.notifications[1].Body, "/demo/app/anliegen#issue-"+issues[0].ID) {
		t.Fatalf("comment notifications = %+v", mailer.notifications)
	}

	residentComment := authedFormRequest(t, a, "resident@example.com", "/demo/app/anliegen/comment", url.Values{
		"id":   {issues[0].ID},
		"body": {"Danke, ich ergänze ein Foto später."},
	})
	if residentComment.Code != http.StatusSeeOther {
		t.Fatalf("resident comment status = %d", residentComment.Code)
	}
	updated, _ = issueRepositoryForTest(a, "demo").Get(issues[0].ID)
	if len(updated.Comments) != 2 {
		t.Fatalf("comments after resident = %+v", updated.Comments)
	}
	managerCommentID := updated.Comments[0].ID
	residentCommentID := updated.Comments[1].ID
	page := authedRequest(t, a, "resident@example.com", "/demo/app/anliegen/"+issues[0].ID)
	body := page.Body.String()
	first := strings.Index(body, "Ich prüfe das")
	second := strings.Index(body, "Danke, ich ergänze")
	if first < 0 || second < 0 || first > second {
		t.Fatalf("comments should render chronologically:\n%s", body)
	}
	if !strings.Contains(body, `id="issue-`+issues[0].ID+`"`) {
		t.Fatalf("issue page missing deeplink anchor:\n%s", body)
	}
	if !strings.Contains(body, `id="comment-`+residentCommentID+`"`) || !strings.Contains(body, `name="comment_id" value="`+residentCommentID+`"`) {
		t.Fatalf("comment creator should see delete control for own comment:\n%s", body)
	}
	if strings.Contains(body, `name="comment_id" value="`+managerCommentID+`"`) {
		t.Fatalf("resident must not see delete control for manager comment:\n%s", body)
	}
	deleteManagerAsResident := authedFormRequest(t, a, "resident@example.com", "/demo/app/anliegen/comment/delete", url.Values{
		"comment_id": {managerCommentID},
	})
	if deleteManagerAsResident.Code != http.StatusForbidden {
		t.Fatalf("resident delete manager comment status = %d, want 403", deleteManagerAsResident.Code)
	}
	unchanged, _ := issueRepositoryForTest(a, "demo").Get(issues[0].ID)
	if len(unchanged.Comments) != 2 {
		t.Fatalf("forbidden delete changed comments: %+v", unchanged.Comments)
	}
	deleteOwn := authedFormRequest(t, a, "resident@example.com", "/demo/app/anliegen/comment/delete", url.Values{
		"comment_id": {residentCommentID},
	})
	if deleteOwn.Code != http.StatusSeeOther {
		t.Fatalf("resident delete own comment status = %d", deleteOwn.Code)
	}
	afterOwnDelete, _ := issueRepositoryForTest(a, "demo").Get(issues[0].ID)
	if len(afterOwnDelete.Comments) != 1 || afterOwnDelete.Comments[0].ID != managerCommentID {
		t.Fatalf("own comment delete result = %+v", afterOwnDelete.Comments)
	}
	deleteManager := authedFormRequest(t, a, "manager@example.com", "/demo/app/anliegen/comment/delete", url.Values{
		"comment_id": {managerCommentID},
	})
	if deleteManager.Code != http.StatusSeeOther {
		t.Fatalf("manager delete comment status = %d", deleteManager.Code)
	}
	afterManagerDelete, _ := issueRepositoryForTest(a, "demo").Get(issues[0].ID)
	if len(afterManagerDelete.Comments) != 0 {
		t.Fatalf("manager delete result = %+v", afterManagerDelete.Comments)
	}
	if len(mailer.notifications) < 3 || !strings.Contains(mailer.notifications[len(mailer.notifications)-1].Subject, "gelöscht") || !strings.Contains(mailer.notifications[len(mailer.notifications)-1].Body, "/demo/app/anliegen#issue-"+issues[0].ID) {
		t.Fatalf("delete notification missing deeplink = %+v", mailer.notifications)
	}
}

func TestResidentCannotCommentOnOtherIssue(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	other, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:   "demo",
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

	comment := authedFormRequest(t, a, "resident@example.com", "/demo/app/anliegen/comment", url.Values{
		"id":   {other.ID},
		"body": {"Kann ich nicht sehen."},
	})
	if comment.Code != http.StatusForbidden {
		t.Fatalf("other comment status = %d, want 403", comment.Code)
	}
	unchanged, _ := issueRepositoryForTest(a, "demo").Get(other.ID)
	if len(unchanged.Comments) != 0 {
		t.Fatalf("other issue comments = %+v", unchanged.Comments)
	}
}

func TestIssueVisibilityByPersona(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "renter@example.com", Role: roleRenter, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["owner@example.com"] = userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["board@example.com"] = userProfile{Email: "board@example.com", Role: roleBeirat, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["manager@example.com"] = userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	for _, item := range []residentIssue{
		{TenantSlug: "demo", AuthorEmail: "renter@example.com", AuthorName: "Renter", Category: "Frage", Title: "Renter private", Body: "Own unit.", LocationType: issueLocationUnit},
		{TenantSlug: "demo", AuthorEmail: "owner@example.com", AuthorName: "Owner", Category: "Frage", Title: "Owner private", Body: "Own unit.", LocationType: issueLocationUnit},
		{TenantSlug: "demo", AuthorEmail: "other@example.com", AuthorName: "Other", Category: "Reparatur", Title: "Common roof", Body: "Shared.", LocationType: issueLocationCommon},
		{TenantSlug: "demo", AuthorEmail: "other@example.com", AuthorName: "Other", Category: "Reparatur", Title: "Other private", Body: "Hidden.", LocationType: issueLocationUnit},
	} {
		if _, err := issueRepositoryForTest(a, "demo").Create(item); err != nil {
			t.Fatalf("Create issue %q: %v", item.Title, err)
		}
	}

	renter := authedRequest(t, a, "renter@example.com", "/demo/app/anliegen").Body.String()
	if !strings.Contains(renter, "Renter private") || strings.Contains(renter, "Common roof") || strings.Contains(renter, "Other private") {
		t.Fatalf("renter visibility wrong:\n%s", renter)
	}
	owner := authedRequest(t, a, "owner@example.com", "/demo/app/anliegen").Body.String()
	if !strings.Contains(owner, "Owner private") || !strings.Contains(owner, "Common roof") || strings.Contains(owner, "Other private") || strings.Contains(owner, "Renter private") {
		t.Fatalf("owner visibility wrong:\n%s", owner)
	}
	board := authedRequest(t, a, "board@example.com", "/demo/app/anliegen").Body.String()
	for _, want := range []string{"Renter private", "Owner private", "Common roof", "Other private"} {
		if !strings.Contains(board, want) {
			t.Fatalf("beirat view missing %q:\n%s", want, board)
		}
	}
	if strings.Contains(board, "Kommentar senden") || strings.Contains(board, "Anliegen senden") {
		t.Fatalf("beirat view must be read-only:\n%s", board)
	}

	boardComment := authedFormRequest(t, a, "board@example.com", "/demo/app/anliegen/comment", url.Values{
		"id":   {issueRepositoryForTest(a, "demo").ListAuthor("owner@example.com")[0].ID},
		"body": {"Read-only should fail."},
	})
	if boardComment.Code != http.StatusForbidden {
		t.Fatalf("beirat comment status = %d, want 403", boardComment.Code)
	}
}

func TestServiceProviderOnlySeesAssignedIssues(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "service@example.com", Role: roleServiceProvider, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.serviceAccessEnabled = true
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}

	assigned, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:    "demo",
		AuthorEmail:   "resident@example.com",
		AuthorName:    "Resident",
		Category:      "Reparatur",
		Title:         "Heizung prüfen",
		Body:          "Bitte vor Ort prüfen.",
		LocationType:  issueLocationCommon,
		AssigneeEmail: "service@example.com",
	})
	if err != nil {
		t.Fatalf("Create assigned issue: %v", err)
	}
	unassigned, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:   "demo",
		AuthorEmail:  "resident@example.com",
		AuthorName:   "Resident",
		Category:     "Reparatur",
		Title:        "Dachrinne reinigen",
		Body:         "Nicht extern zugewiesen.",
		LocationType: issueLocationCommon,
	})
	if err != nil {
		t.Fatalf("Create unassigned issue: %v", err)
	}

	page := authedRequest(t, a, "service@example.com", "/demo/app/anliegen")
	if page.Code != http.StatusOK {
		t.Fatalf("service issue page status = %d", page.Code)
	}
	body := page.Body.String()
	if !strings.Contains(body, "Heizung prüfen") || strings.Contains(body, "Dachrinne reinigen") {
		t.Fatalf("service provider issue visibility wrong:\n%s", body)
	}
	for _, hidden := range []string{`href="/demo/app/announcements"`, `href="/demo/app/events"`, `href="/demo/app/kontakte"`, `href="/demo/app/dokumente"`, `href="/demo/app/abstimmungen"`, `href="/demo/app/settings"`, "Anliegen senden"} {
		if strings.Contains(body, hidden) {
			t.Fatalf("service provider page should not expose %q:\n%s", hidden, body)
		}
	}
	if !a.canViewIssueForActor(testTenantRef("demo"), assigned, "service@example.com", roleServiceProvider) {
		t.Fatal("service provider should be able to view assigned issue")
	}
	if a.canViewIssueForActor(testTenantRef("demo"), unassigned, "service@example.com", roleServiceProvider) {
		t.Fatal("service provider should not be able to view unassigned issue")
	}
	if emailListContains(a.tenantNotificationEmails("demo"), "service@example.com") {
		t.Fatal("service provider should not receive broad tenant notifications")
	}
	if !emailListContains(a.notificationRecipients(portalNotification{
		Event:      notificationEventIssue,
		Recipients: []string{"service@example.com"},
		ActorEmail: "manager@example.com",
	}), "service@example.com") {
		t.Fatal("service provider should still receive explicitly assigned issue notifications")
	}

	comment := authedFormRequest(t, a, "service@example.com", "/demo/app/anliegen/comment", url.Values{
		"id":   {unassigned.ID},
		"body": {"Bitte ansehen."},
	})
	if comment.Code != http.StatusForbidden {
		t.Fatalf("service provider comment on unassigned issue status = %d, want 403", comment.Code)
	}

	create := authedFormRequest(t, a, "service@example.com", "/demo/app/anliegen", url.Values{
		"category":      {"Frage"},
		"location_type": {issueLocationCommon},
		"title":         {"Neues Anliegen"},
		"body":          {"Darf nicht angelegt werden."},
	})
	if create.Code != http.StatusForbidden {
		t.Fatalf("service provider create issue status = %d, want 403", create.Code)
	}

	portal := authedRequest(t, a, "service@example.com", "/demo/app")
	if portal.Code != http.StatusSeeOther || portal.Header().Get("Location") != "/demo/app/anliegen" {
		t.Fatalf("service provider portal redirect = %d %q, want /demo/app/anliegen", portal.Code, portal.Header().Get("Location"))
	}
	for name, path := range map[string]string{
		"announcements": "/demo/app/announcements",
		"contacts":      "/demo/app/kontakte",
		"documents":     "/demo/app/dokumente",
		"settings":      "/demo/app/settings",
		"users":         "/demo/app/settings/users",
	} {
		rr := authedRequest(t, a, "service@example.com", path)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("%s status = %d, want 403", name, rr.Code)
		}
	}
}

func TestServiceProviderAccessDefaultsClosedAndRejectsWritesAtomically(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if a.serviceAccessEnabled {
		t.Fatal("service-provider access must default to closed")
	}
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	mailer := &recordingMailer{}
	a.mailer = mailer

	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:   "demo",
		AuthorEmail:  "resident@example.com",
		AuthorName:   "Resident",
		Category:     "Reparatur",
		Title:        "Fenster undicht",
		Body:         "Bitte prüfen.",
		LocationType: issueLocationCommon,
		Status:       issueStatusOpen,
		Priority:     issuePriorityNorm,
	})
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}
	assign := authedFormRequest(t, a, "manager@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":             {issue.ID},
		"status":         {issueStatusProgress},
		"priority":       {issuePriorityHigh},
		"assignee_email": {"external@example.com"},
	})
	if assign.Code != http.StatusForbidden || !strings.Contains(assign.Body.String(), "Dienstleister-Zugänge sind derzeit nicht verfügbar") {
		t.Fatalf("closed assignment = %d %q, want clear 403", assign.Code, assign.Body.String())
	}
	unchanged, ok := issueRepositoryForTest(a, "demo").Get(issue.ID)
	if !ok || unchanged.Status != issueStatusOpen || unchanged.Priority != issuePriorityNorm || unchanged.AssigneeEmail != "" || len(unchanged.StatusHistory) != 0 {
		t.Fatalf("rejected assignment changed issue: %+v ok=%v", unchanged, ok)
	}
	if _, ok := a.inviteStore.Get("external@example.com"); ok {
		t.Fatal("rejected assignment created a service-provider profile")
	}
	if len(mailer.magicLinks) != 0 || len(mailer.invites) != 0 || len(mailer.notifications) != 0 || assign.Header().Get("Set-Cookie") != "" {
		t.Fatalf("rejected assignment caused side effects: magic=%+v invites=%+v notifications=%+v cookie=%q", mailer.magicLinks, mailer.invites, mailer.notifications, assign.Header().Get("Set-Cookie"))
	}
	if events := a.auditStore.List(auditFilter{TenantSlug: "demo", Limit: 20}); len(events) != 0 {
		t.Fatalf("rejected assignment created audit events: %+v", events)
	}

	createContact := authedFormRequest(t, a, "manager@example.com", "/demo/app/kontakte", url.Values{
		"kind": {"Dienstleister"}, "name": {"Extern GmbH"}, "email": {"contact@example.com"}, "active": {"true"},
	})
	if createContact.Code != http.StatusForbidden || len(testRepositories(a, "demo").contacts.List(true)) != 0 {
		t.Fatalf("closed contact create = %d contacts=%+v", createContact.Code, testRepositories(a, "demo").contacts.List(true))
	}
	existingContact, _, err := testRepositories(a, "demo").contacts.Upsert(managedContact{TenantSlug: "demo", Kind: roleServiceProvider, Name: "Alt GmbH", Email: "old-contact@example.com", Active: true})
	if err != nil {
		t.Fatalf("seed service contact: %v", err)
	}
	editContact := authedFormRequest(t, a, "manager@example.com", "/demo/app/kontakte", url.Values{
		"id": {existingContact.ID}, "kind": {"Hausmeister"}, "name": {"Neu GmbH"}, "email": {"new-contact@example.com"}, "active": {"true"},
	})
	contacts := testRepositories(a, "demo").contacts.List(true)
	if editContact.Code != http.StatusForbidden || len(contacts) != 1 || contacts[0].Name != "Alt GmbH" || contacts[0].Email != "old-contact@example.com" {
		t.Fatalf("closed contact edit = %d contacts=%+v", editContact.Code, contacts)
	}

	createInvite := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/users", url.Values{
		"email": {"invite@example.com"}, "role": {"Handwerker"},
	})
	if createInvite.Code != http.StatusForbidden {
		t.Fatalf("closed service invite create status = %d, want 403", createInvite.Code)
	}
	if _, ok := a.inviteStore.Get("invite@example.com"); ok || len(mailer.invites) != 0 {
		t.Fatalf("closed service invite persisted or mailed: ok=%v mail=%+v", ok, mailer.invites)
	}
	if _, err := a.inviteStore.Add(userProfile{Email: "existing-service@example.com", FirstName: "Alt", Role: roleServiceProvider, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("seed service invite: %v", err)
	}
	editInvite := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/users/edit", url.Values{
		"orig_email": {"existing-service@example.com"}, "email": {"existing-service@example.com"}, "first_name": {"Neu"}, "role": {roleResident},
	})
	persisted, ok := a.inviteStore.Get("existing-service@example.com")
	if editInvite.Code != http.StatusForbidden || !ok || persisted.FirstName != "Alt" || !isServiceProviderRole(persisted.Role) {
		t.Fatalf("closed service invite edit = %d profile=%+v ok=%v", editInvite.Code, persisted, ok)
	}
	if events := a.auditStore.List(auditFilter{TenantSlug: "demo", Limit: 20}); len(events) != 0 {
		t.Fatalf("rejected contact/invite writes created audit events: %+v", events)
	}

	board := authedRequest(t, a, "manager@example.com", "/demo/app/anliegen/board").Body.String()
	// The board may preserve an existing assignment in a hidden move field,
	// but must not expose an editable assignment control (HAUSV-706).
	assertIssueBoardStatusMenus(t, board)
	if strings.Contains(board, "Betreiberfreigabe offen") || strings.Contains(board, "datalist id=\"service-provider-contacts\"") {
		t.Fatalf("closed issue board should not expose service assignment controls or internal gate language:\n%s", board)
	}
	contactPage := authedRequest(t, a, "manager@example.com", "/demo/app/kontakte").Body.String()
	if strings.Contains(contactPage, "Dienstleister-Zugänge sind derzeit nicht verfügbar") || strings.Contains(contactPage, "<option value=\"Dienstleister\"") {
		t.Fatalf("closed contact UI exposes service-provider creation or internal gate copy:\n%s", contactPage)
	}
	usersPage := authedRequest(t, a, "manager@example.com", "/demo/app/settings/users").Body.String()
	if !strings.Contains(usersPage, "Dienstleister-Zugänge können noch nicht") || strings.Contains(usersPage, "<option value=\"Dienstleister\"") {
		t.Fatalf("closed user UI still offers service-provider roles:\n%s", usersPage)
	}
}

func TestPublicPrivacyNoticeMatchesActualDependencies(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	tenant := a.tenants["demo"]
	tenant.ContactName = "Hausverwaltung Beispiel GmbH"
	tenant.ContactAddress = "Verwaltergasse 10, 1010 Wien"
	tenant.ContactEmail = "haus@example.com"
	a.tenants["demo"] = tenant

	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/datenschutz", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("privacy status = %d, body=%q", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{
		"Betreiber-Selbstprüfung",
		"Dienstleister-Zugang: geschlossen",
		"Hausverwaltung Beispiel GmbH",
		"Verwaltergasse 10, 1010 Wien",
		"Betroffene Liegenschaft",
		"haus@example.com",
		"Betreiber laut Host-Konfiguration",
		"Anschrift laut Host-Konfiguration",
		"hello@hausv.org",
		"Transaktionsmails werden über den vom Betreiber dokumentierten Maildienst versendet",
		"öffentliche Webzugriff wird über die vom Betreiber dokumentierte Infrastruktur",
		"Sicherungen werden verschlüsselt",
		"Auditdaten werden nach drei Jahren zur Löschung fällig",
		"Österreichische Datenschutzbehörde",
		"§ 5 ECG",
		"§ 24 MedienG",
		"keine externe Zertifizierung",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("privacy page missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "fonts.googleapis.com") || strings.Contains(body, "fonts.gstatic.com") {
		t.Fatalf("privacy page loads an external font dependency:\n%s", body)
	}
	csp := rr.Header().Get("Content-Security-Policy")
	if strings.Contains(csp, "googleapis.com") || strings.Contains(csp, "gstatic.com") {
		t.Fatalf("CSP still permits external font hosts: %q", csp)
	}
	if !strings.Contains(csp, "img-src 'self' blob:") {
		t.Fatalf("CSP blocks local selected-image previews: %q", csp)
	}
}

func TestServiceProviderAccessConfigRequiresExplicitOptIn(t *testing.T) {
	t.Setenv("SERVICE_PROVIDER_ACCESS_ENABLED", "")
	t.Setenv("SERVICE_PROVIDER_ASSESSMENT_VERSION", "")
	if serviceProviderAccessEnabled() {
		t.Fatal("empty SERVICE_PROVIDER_ACCESS_ENABLED must stay closed")
	}
	t.Setenv("SERVICE_PROVIDER_ACCESS_ENABLED", "true")
	if serviceProviderAccessEnabled() {
		t.Fatal("boolean opt-in without assessment revision must stay closed")
	}
	t.Setenv("SERVICE_PROVIDER_ASSESSMENT_VERSION", "outdated")
	if serviceProviderAccessEnabled() {
		t.Fatal("outdated assessment revision must stay closed")
	}
	t.Setenv("SERVICE_PROVIDER_ASSESSMENT_VERSION", serviceProviderAssessmentVersion)
	if !serviceProviderAccessEnabled() {
		t.Fatal("boolean opt-in plus current assessment revision should open the tested flow")
	}
}

func TestClosedServiceProviderAuthenticationAndNotificationsAreRejected(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "service@example.com", Role: roleServiceProvider, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	mailer := &recordingMailer{}
	a.mailer = mailer

	loginValues := url.Values{"email": {"service@example.com"}}
	loginReq := httptest.NewRequest(http.MethodPost, "http://hausv.org/demo/auth/request", strings.NewReader(loginValues.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginReq.Header.Set("Origin", "http://hausv.org/demo")
	login := httptest.NewRecorder()
	a.handler().ServeHTTP(login, loginReq)
	if login.Code != http.StatusSeeOther || login.Header().Get("Location") != "/demo/?sent=1" || len(mailer.magicLinks) != 0 {
		t.Fatalf("closed login request = %d %q magic=%+v", login.Code, login.Body.String(), mailer.magicLinks)
	}

	a.tokens.Put("existing-magic-token", "service@example.com", "demo", 15*time.Minute)
	verify := httptest.NewRecorder()
	verifyReq := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/auth/verify?token=existing-magic-token", nil)
	a.handler().ServeHTTP(verify, verifyReq)
	if verify.Code != http.StatusForbidden || !strings.Contains(verify.Body.String(), "Dienstleister-Zugänge sind derzeit nicht verfügbar") || verify.Header().Get("Set-Cookie") != "" {
		t.Fatalf("closed magic-link verification = %d %q cookie=%q", verify.Code, verify.Body.String(), verify.Header().Get("Set-Cookie"))
	}

	sessionToken, _, err := a.sessions.Put("service@example.com", "demo", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("seed existing session: %v", err)
	}
	sessionReq := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app/anliegen", nil)
	sessionReq.AddCookie(&http.Cookie{Name: "weg_session", Value: sessionToken})
	session := httptest.NewRecorder()
	a.handler().ServeHTTP(session, sessionReq)
	if session.Code != http.StatusForbidden || !strings.Contains(session.Body.String(), "Dienstleister-Zugänge sind derzeit nicht verfügbar") {
		t.Fatalf("closed existing session = %d %q", session.Code, session.Body.String())
	}

	a.notify(portalNotification{
		Event:      notificationEventIssue,
		Tenant:     a.tenants["demo"],
		Recipients: []string{"service@example.com", "resident@example.com"},
		Subject:    "Test",
	})
	if len(mailer.notifications) != 1 || mailer.notifications[0].To != "resident@example.com" {
		t.Fatalf("closed provider received notification: %+v", mailer.notifications)
	}
	a.notifyIssueUpdated(a.tenants["demo"], residentIssue{
		AuthorEmail:   "resident@example.com",
		AssigneeEmail: "unknown-external@example.com",
		Title:         "Bestehende Zuordnung",
		Status:        issueStatusOpen,
		Priority:      issuePriorityNorm,
	}, "manager@example.com", "Test")
	for _, sent := range mailer.notifications {
		if sent.To != "resident@example.com" {
			t.Fatalf("closed unknown provider received notification: %+v", mailer.notifications)
		}
	}
	if events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionLogin, Limit: 20}); len(events) != 0 {
		t.Fatalf("rejected authentication created login audit events: %+v", events)
	}
}

func TestExplicitlyEnabledServiceProviderAccessSupportsInviteAndRevoke(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.serviceAccessEnabled = true
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	mailer := &recordingMailer{}
	a.mailer = mailer

	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:   "demo",
		AuthorEmail:  "resident@example.com",
		AuthorName:   "Resident",
		Category:     "Reparatur",
		Title:        "Fenster undicht",
		Body:         "Bitte prüfen.",
		LocationType: issueLocationCommon,
	})
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	assign := authedFormRequest(t, a, "manager@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":             {issue.ID},
		"status":         {issueStatusProgress},
		"priority":       {issuePriorityHigh},
		"assignee_email": {"service@example.com"},
	})
	if assign.Code != http.StatusSeeOther {
		t.Fatalf("assign service provider status = %d, want redirect", assign.Code)
	}
	invite, ok := a.inviteStore.Get("service@example.com")
	if !ok || normalizeRole(invite.Role) != roleServiceProvider || !invite.HasTenant("demo") {
		t.Fatalf("service invite = %+v ok=%v", invite, ok)
	}
	if len(mailer.magicLinks) != 1 || mailer.magicLinks[0].To != "service@example.com" {
		t.Fatalf("magic links = %+v", mailer.magicLinks)
	}

	verify := httptest.NewRecorder()
	verifyReq := httptest.NewRequest(http.MethodGet, mailer.magicLinks[0].Link, nil)
	a.handler().ServeHTTP(verify, verifyReq)
	if verify.Code != http.StatusSeeOther || verify.Header().Get("Location") != "/demo/app/anliegen#issue-"+issue.ID {
		t.Fatalf("verify redirect = %d %q", verify.Code, verify.Header().Get("Location"))
	}

	servicePage := authedRequest(t, a, "service@example.com", "/demo/app/anliegen").Body.String()
	if !strings.Contains(servicePage, "Fenster undicht") {
		t.Fatalf("assigned issue missing for service provider:\n%s", servicePage)
	}

	closeIssue := authedFormRequest(t, a, "manager@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":             {issue.ID},
		"status":         {issueStatusDone},
		"priority":       {issuePriorityHigh},
		"assignee_email": {"service@example.com"},
	})
	if closeIssue.Code != http.StatusSeeOther {
		t.Fatalf("close issue status = %d, want redirect", closeIssue.Code)
	}
	closed, ok := issueRepositoryForTest(a, "demo").Get(issue.ID)
	if !ok {
		t.Fatal("closed issue not found")
	}
	if a.canViewIssueForActor(testTenantRef("demo"), closed, "service@example.com", roleServiceProvider) {
		t.Fatal("closed issue should lock service provider out")
	}

	revoke := authedFormRequest(t, a, "manager@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":             {issue.ID},
		"status":         {issueStatusNew},
		"priority":       {issuePriorityHigh},
		"assignee_email": {""},
	})
	if revoke.Code != http.StatusSeeOther {
		t.Fatalf("revoke service provider status = %d, want redirect", revoke.Code)
	}
	revoked, ok := issueRepositoryForTest(a, "demo").Get(issue.ID)
	if !ok {
		t.Fatal("revoked issue not found")
	}
	if a.canViewIssueForActor(testTenantRef("demo"), revoked, "service@example.com", roleServiceProvider) {
		t.Fatal("unassigned issue should lock service provider out")
	}

	events := a.auditStore.List(auditFilter{TenantSlug: "demo", Limit: 50})
	for _, action := range []string{auditActionIssueServiceAdd, auditActionIssueServiceDrop, auditActionLogin} {
		found := false
		for _, event := range events {
			if event.Action == action {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("audit action %q missing in %+v", action, events)
		}
	}
}

func TestServiceProviderCanWorkAssignedIssueWithCommentPhotoAndProposal(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "service@example.com", Role: roleServiceProvider, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.serviceAccessEnabled = true
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}

	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:    "demo",
		AuthorEmail:   "resident@example.com",
		AuthorName:    "Resident",
		Category:      "Reparatur",
		Title:         "Kellerlicht defekt",
		Body:          "Bitte tauschen.",
		LocationType:  issueLocationCommon,
		AssigneeEmail: "service@example.com",
	})
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	page := authedRequest(t, a, "service@example.com", "/demo/app/anliegen").Body.String()
	for _, want := range []string{"Kellerlicht defekt", "Kommentar senden", `name="service_proposal"`, "Status senden"} {
		if !strings.Contains(page, want) {
			t.Fatalf("service work view missing %q:\n%s", want, page)
		}
	}

	comment := authedMultipartFilesRequest(t, a, "service@example.com", "/demo/app/anliegen/comment", map[string]string{
		"id":   issue.ID,
		"body": "Leuchte ist bestellt, Foto vom Bestand angehängt.",
	}, []multipartTestFile{{Field: "attachments", Filename: "bestand.png", Body: minimalPNG()}})
	if comment.Code != http.StatusSeeOther {
		t.Fatalf("service comment status = %d, want redirect", comment.Code)
	}
	updated, ok := issueRepositoryForTest(a, "demo").Get(issue.ID)
	if !ok || len(updated.Comments) != 1 || updated.Comments[0].AuthorEmail != "service@example.com" {
		t.Fatalf("service comment not saved: %+v ok=%v", updated.Comments, ok)
	}
	if attachments := attachmentRepositoryForTest(a, "demo").ListEntity("issue-comment", updated.Comments[0].ID); len(attachments) != 1 {
		t.Fatalf("service comment attachments = %+v, want one", attachments)
	}

	forbidden := authedFormRequest(t, a, "service@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":             {issue.ID},
		"status":         {issueStatusProgress},
		"priority":       {issuePriorityUrgent},
		"assignee_email": {"other@example.com"},
	})
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("service forbidden workflow status = %d, want 403", forbidden.Code)
	}

	proposal := authedFormRequest(t, a, "service@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":               {issue.ID},
		"status":           {issueStatusProgress},
		"service_proposal": {"Dienstag, 14. Juli, 9-11 Uhr"},
	})
	if proposal.Code != http.StatusSeeOther {
		t.Fatalf("service proposal status = %d, want redirect", proposal.Code)
	}
	updated, ok = issueRepositoryForTest(a, "demo").Get(issue.ID)
	if !ok || updated.Status != issueStatusProgress || updated.Priority != issuePriorityNorm || updated.AssigneeEmail != "service@example.com" || updated.ServiceProposal != "Dienstag, 14. Juli, 9-11 Uhr" {
		t.Fatalf("service workflow update = %+v ok=%v", updated, ok)
	}
	residentPage := authedRequest(t, a, "resident@example.com", "/demo/app/anliegen/"+issue.ID).Body.String()
	if !strings.Contains(residentPage, "Hinweis:") || !strings.Contains(residentPage, "Dienstag, 14. Juli, 9-11 Uhr") || !strings.Contains(residentPage, "Leuchte ist bestellt") {
		t.Fatalf("resident should see service proposal and comment:\n%s", residentPage)
	}

	done := authedFormRequest(t, a, "service@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":               {issue.ID},
		"status":           {issueStatusDone},
		"service_proposal": {"Erledigt am Dienstagvormittag"},
	})
	if done.Code != http.StatusSeeOther {
		t.Fatalf("service done status = %d, want redirect", done.Code)
	}
	closed, ok := issueRepositoryForTest(a, "demo").Get(issue.ID)
	if !ok || closed.Status != issueStatusDone || closed.ServiceProposal != "Erledigt am Dienstagvormittag" {
		t.Fatalf("service done update = %+v ok=%v", closed, ok)
	}
	if a.canViewIssueForActor(testTenantRef("demo"), closed, "service@example.com", roleServiceProvider) {
		t.Fatal("service provider should lose access after marking issue done")
	}
}

// HAUSV-128 AC-b: a service provider's own actions on an assigned issue —
// status changes AND uploads — must be visible in the Verwaltung audit-log.
// Before this, comment/photo uploads left no trace.
func TestServiceProviderCommentPhotoAndStatusAreAudited(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "service@example.com", Role: roleServiceProvider, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.serviceAccessEnabled = true

	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:    "demo",
		AuthorEmail:   "resident@example.com",
		AuthorName:    "Resident",
		Category:      "Reparatur",
		Title:         "Treppenhauslicht",
		Body:          "Bitte prüfen.",
		LocationType:  issueLocationCommon,
		AssigneeEmail: "service@example.com",
	})
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	comment := authedMultipartFilesRequest(t, a, "service@example.com", "/demo/app/anliegen/comment", map[string]string{
		"id":   issue.ID,
		"body": "Foto vom Schaden angehängt.",
	}, []multipartTestFile{{Field: "attachments", Filename: "schaden.png", Body: minimalPNG()}})
	if comment.Code != http.StatusSeeOther {
		t.Fatalf("service comment status = %d, want redirect", comment.Code)
	}

	status := authedFormRequest(t, a, "service@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":     {issue.ID},
		"status": {issueStatusProgress},
	})
	if status.Code != http.StatusSeeOther {
		t.Fatalf("service status change = %d, want redirect", status.Code)
	}

	commentEvents := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionIssueComment, Limit: 10})
	if len(commentEvents) == 0 {
		t.Fatal("provider comment produced no issue.comment audit event")
	}
	if commentEvents[0].ActorEmail != "service@example.com" {
		t.Fatalf("comment audit actor = %q, want the provider", commentEvents[0].ActorEmail)
	}
	if commentEvents[0].Details["has_file"] != "true" {
		t.Fatalf("comment audit has_file = %q, want true (photo upload must be traceable)", commentEvents[0].Details["has_file"])
	}

	workflowEvents := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionIssueWorkflow, Limit: 10})
	if len(workflowEvents) == 0 || workflowEvents[0].ActorEmail != "service@example.com" {
		t.Fatalf("provider status change not audited with provider as actor: %+v", workflowEvents)
	}
}

// HAUSV-128: a comment may carry a photo with no text (before/after
// documentation), but a truly-empty submit is still rejected.
func TestIssueCommentAllowsPhotoOnlyButRejectsEmpty(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "service@example.com", Role: roleServiceProvider, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.serviceAccessEnabled = true
	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug: "demo", AuthorEmail: "resident@example.com", AuthorName: "Resident",
		Category: "Reparatur", Title: "Foto-only", Body: "Bitte prüfen.", LocationType: issueLocationCommon,
		AssigneeEmail: "service@example.com",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	photoOnly := authedMultipartFilesRequest(t, a, "service@example.com", "/demo/app/anliegen/comment", map[string]string{
		"id": issue.ID,
	}, []multipartTestFile{{Field: "attachments", Filename: "foto.png", Body: minimalPNG()}})
	if photoOnly.Code != http.StatusSeeOther || photoOnly.Header().Get("Location") != "/demo/app/anliegen?issue=updated" {
		t.Fatalf("photo-only comment = %d loc=%q, want redirect to updated", photoOnly.Code, photoOnly.Header().Get("Location"))
	}
	updated, ok := issueRepositoryForTest(a, "demo").Get(issue.ID)
	if !ok || len(updated.Comments) != 1 || updated.Comments[0].Body != "" {
		t.Fatalf("photo-only comment not saved with empty body: %+v", updated.Comments)
	}
	if atts := attachmentRepositoryForTest(a, "demo").ListEntity("issue-comment", updated.Comments[0].ID); len(atts) != 1 {
		t.Fatalf("photo-only attachment = %+v, want one", atts)
	}

	empty := authedFormRequest(t, a, "service@example.com", "/demo/app/anliegen/comment", url.Values{"id": {issue.ID}})
	if empty.Code != http.StatusSeeOther || empty.Header().Get("Location") != "/demo/app/anliegen?issue=invalid" {
		t.Fatalf("empty comment = %d loc=%q, want redirect to invalid", empty.Code, empty.Header().Get("Location"))
	}
	if again, _ := issueRepositoryForTest(a, "demo").Get(issue.ID); len(again.Comments) != 1 {
		t.Fatalf("empty comment must not be saved: now %d comments", len(again.Comments))
	}
}

// HAUSV-128: a provider can advance an assigned issue into the new states
// (Angenommen) and keeps access; reopening to Neu is blocked.
func TestServiceProviderCanAcceptButNotReopen(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "service@example.com", Role: roleServiceProvider, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.serviceAccessEnabled = true
	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug: "demo", AuthorEmail: "resident@example.com", AuthorName: "Resident",
		Category: "Reparatur", Title: "Annehmen", Body: "Bitte prüfen.", LocationType: issueLocationCommon,
		AssigneeEmail: "service@example.com",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	accept := authedFormRequest(t, a, "service@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":     {issue.ID},
		"status": {issueStatusAccepted},
	})
	if accept.Code != http.StatusSeeOther {
		t.Fatalf("accept status = %d, want redirect", accept.Code)
	}
	updated, ok := issueRepositoryForTest(a, "demo").Get(issue.ID)
	if !ok || updated.Status != issueStatusAccepted {
		t.Fatalf("status after accept = %q, want %q", updated.Status, issueStatusAccepted)
	}
	if !a.canViewIssueForActor(testTenantRef("demo"), updated, "service@example.com", roleServiceProvider) {
		t.Fatal("Angenommen must keep the issue open and accessible to the provider")
	}

	reopen := authedFormRequest(t, a, "service@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":     {issue.ID},
		"status": {issueStatusNew},
	})
	if reopen.Code != http.StatusForbidden {
		t.Fatalf("provider reopen status = %d, want 403", reopen.Code)
	}
}

// HAUSV-128: "Termin vereinbart" requires a structured date, and a structured
// appointment renders as a real dated VEVENT (not the old dateless VTODO).
func TestServiceProviderScheduledRequiresDateAndEmitsEvent(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "service@example.com", Role: roleServiceProvider, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.serviceAccessEnabled = true
	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug: "demo", AuthorEmail: "resident@example.com", AuthorName: "Resident",
		Category: "Reparatur", Title: "Termin", Body: "Bitte prüfen.", LocationType: issueLocationCommon,
		AssigneeEmail: "service@example.com",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	noDate := authedFormRequest(t, a, "service@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":     {issue.ID},
		"status": {issueStatusScheduled},
	})
	if noDate.Code != http.StatusSeeOther || noDate.Header().Get("Location") != "/demo/app/anliegen?issue=termin" {
		t.Fatalf("scheduled without date = %d loc=%q, want redirect to termin", noDate.Code, noDate.Header().Get("Location"))
	}
	if u, _ := issueRepositoryForTest(a, "demo").Get(issue.ID); u.Status == issueStatusScheduled {
		t.Fatal("status must not become Termin vereinbart without a date")
	}

	withDate := authedFormRequest(t, a, "service@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":            {issue.ID},
		"status":        {issueStatusScheduled},
		"service_start": {"2026-07-14T09:00"},
		"service_end":   {"2026-07-14T11:00"},
	})
	if withDate.Code != http.StatusSeeOther || withDate.Header().Get("Location") != "/demo/app/anliegen?issue=updated" {
		t.Fatalf("scheduled with date = %d loc=%q, want redirect to updated", withDate.Code, withDate.Header().Get("Location"))
	}
	updated, ok := issueRepositoryForTest(a, "demo").Get(issue.ID)
	if !ok || updated.Status != issueStatusScheduled || updated.ServiceProposedStart.IsZero() {
		t.Fatalf("scheduled issue = %+v, want status scheduled + start set", updated)
	}

	var b strings.Builder
	writeCalendarIssueProposal(&b, tenantConfig{Slug: "demo"}, updated, time.Now())
	if ics := b.String(); !strings.Contains(ics, "BEGIN:VEVENT") || !strings.Contains(ics, "DTSTART:") {
		t.Fatalf("structured appointment must emit a dated VEVENT:\n%s", ics)
	}
}

// HAUSV-125: the fair-use free-unit rule is a visible soft guardrail on the
// building-settings page (informational, not enforced), and a Stellplatz does
// not count toward the billable total.
func TestFairUseIndicatorOnBuildingSettings(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	mkUnits := func(n int) []unit {
		out := make([]unit, 0, n)
		for i := 0; i < n; i++ {
			out = append(out, unit{ID: fmt.Sprintf("u%03d", i), TenantSlug: "demo", Label: fmt.Sprintf("Top %d", i), UnitType: unitTypeResidential})
		}
		return out
	}

	if err := testUnitRepository(t, a, "demo").SetUnits(mkUnits(25)); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	page := authedRequest(t, a, "admin@example.com", "/demo/app/settings/building?section=units").Body.String()
	if !strings.Contains(page, "von 25") {
		t.Fatalf("building settings should show 'von 25':\n%s", page)
	}
	if strings.Contains(page, "Über dem inkludierten Rahmen") {
		t.Fatal("25 billable units must not trigger the over-limit hint")
	}

	withParking := append(mkUnits(25), unit{ID: "p1", TenantSlug: "demo", Label: "Stellplatz", UnitType: unitTypeParking})
	if err := testUnitRepository(t, a, "demo").SetUnits(withParking); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	if page = authedRequest(t, a, "admin@example.com", "/demo/app/settings/building?section=units").Body.String(); strings.Contains(page, "Über dem inkludierten Rahmen") {
		t.Fatal("a Stellplatz must not push the billable count over the fair-use limit")
	}

	if err := testUnitRepository(t, a, "demo").SetUnits(mkUnits(26)); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	if page = authedRequest(t, a, "admin@example.com", "/demo/app/settings/building?section=units").Body.String(); !strings.Contains(page, "Über dem inkludierten Rahmen") {
		t.Fatalf("26 billable units should trigger the over-limit hint:\n%s", page)
	}
}

func TestCalendarFeedTokenScopesEventsAndServiceProposals(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.serviceAccessEnabled = true
	a.profiles["owner@example.com"] = userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["service@example.com"] = userProfile{Email: "service@example.com", Role: roleServiceProvider, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["other-service@example.com"] = userProfile{Email: "other-service@example.com", Role: roleServiceProvider, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}

	start := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	if _, err := testRepositories(a, "demo").events.Create(houseEvent{
		TenantSlug:  "demo",
		Title:       "Hausversammlung",
		Body:        "Beschlüsse und offene Punkte.",
		Category:    "Eigentümerversammlung",
		Location:    "Gemeinschaftsraum",
		StartsAt:    start,
		AuthorEmail: "manager@example.com",
		AuthorName:  "Verwaltung",
	}); err != nil {
		t.Fatalf("create event: %v", err)
	}
	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:      "demo",
		AuthorEmail:     "owner@example.com",
		AuthorName:      "Owner",
		Category:        "Reparatur",
		Title:           "Kellerlicht prüfen",
		Body:            "Nicht öffentlich für andere Bewohner.",
		LocationType:    issueLocationUnit,
		LocationDetail:  "Top 2",
		Status:          issueStatusProgress,
		Priority:        issuePriorityNorm,
		AssigneeEmail:   "service@example.com",
		ServiceProposal: "Dienstag 9-11 Uhr",
	})
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}

	residentToken, err := a.calendarFeedToken("resident@example.com", "demo")
	if err != nil {
		t.Fatalf("resident token: %v", err)
	}
	resident := calendarFeedRequest(t, a, residentToken)
	if resident.Code != http.StatusOK {
		t.Fatalf("resident feed status = %d", resident.Code)
	}
	residentBody := resident.Body.String()
	for _, want := range []string{"BEGIN:VCALENDAR", "BEGIN:VEVENT", "Hausversammlung", "Gemeinschaftsraum"} {
		if !strings.Contains(residentBody, want) {
			t.Fatalf("resident feed missing %q:\n%s", want, residentBody)
		}
	}
	for _, hidden := range []string{"Kellerlicht prüfen", "Dienstag 9-11 Uhr", "BEGIN:VTODO"} {
		if strings.Contains(residentBody, hidden) {
			t.Fatalf("resident feed should not expose %q:\n%s", hidden, residentBody)
		}
	}
	if got := resident.Header().Get("Content-Type"); !strings.Contains(got, "text/calendar") {
		t.Fatalf("feed content-type = %q", got)
	}

	serviceToken, err := a.calendarFeedToken("service@example.com", "demo")
	if err != nil {
		t.Fatalf("service token: %v", err)
	}
	service := calendarFeedRequest(t, a, serviceToken)
	if service.Code != http.StatusOK {
		t.Fatalf("service feed status = %d", service.Code)
	}
	serviceBody := service.Body.String()
	for _, want := range []string{"BEGIN:VCALENDAR", "BEGIN:VTODO", "Terminvorschlag: Kellerlicht prüfen", "Dienstag 9-11 Uhr", "NEEDS-ACTION"} {
		if !strings.Contains(serviceBody, want) {
			t.Fatalf("service feed missing %q:\n%s", want, serviceBody)
		}
	}
	if strings.Contains(serviceBody, "Hausversammlung") || strings.Contains(serviceBody, "BEGIN:VEVENT") {
		t.Fatalf("service provider feed should not expose house events:\n%s", serviceBody)
	}

	otherToken, err := a.calendarFeedToken("other-service@example.com", "demo")
	if err != nil {
		t.Fatalf("other service token: %v", err)
	}
	other := calendarFeedRequest(t, a, otherToken)
	if strings.Contains(other.Body.String(), issue.Title) {
		t.Fatalf("unassigned service provider feed exposed issue:\n%s", other.Body.String())
	}

	invalid := calendarFeedRequest(t, a, serviceToken+"x")
	if invalid.Code != http.StatusNotFound {
		t.Fatalf("invalid feed token status = %d, want 404", invalid.Code)
	}

	eventPage := authedRequest(t, a, "resident@example.com", "/demo/app/events").Body.String()
	if !strings.Contains(eventPage, "Kalender abonnieren") || !strings.Contains(eventPage, "/calendar/") {
		t.Fatalf("events page should expose calendar subscription link:\n%s", eventPage)
	}
	issuePage := authedRequest(t, a, "service@example.com", "/demo/app/anliegen").Body.String()
	if !strings.Contains(issuePage, "Kalender abonnieren") || !strings.Contains(issuePage, "/calendar/") {
		t.Fatalf("service issue page should expose calendar subscription link:\n%s", issuePage)
	}
}

func calendarFeedRequest(t *testing.T, a *app, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/calendar/"+token+".ics", nil)
	req.SetPathValue("token", token+".ics")
	rr := httptest.NewRecorder()
	a.calendarFeed(rr, req)
	return rr
}

func TestContactBookCRUDTenantVisibilityAndServiceProviderDatalist(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.serviceAccessEnabled = true
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	addTestTenant(a, tenantConfig{Slug: "other", Name: "Other Portal", Address: "Andere Gasse 1"})

	save := authedFormRequest(t, a, "manager@example.com", "/demo/app/kontakte", url.Values{
		"kind":    {"Dienstleister"},
		"name":    {"Eva Elektrik"},
		"company": {"Elektro Eva GmbH"},
		"email":   {"eva@example.com"},
		"phone":   {"+43 316 123"},
		"notes":   {"Elektrik und Licht"},
		"active":  {"true"},
	})
	if save.Code != http.StatusSeeOther {
		t.Fatalf("contact save status = %d, want redirect", save.Code)
	}
	contacts := testRepositories(a, "demo").contacts.List(true)
	if len(contacts) != 1 || contacts[0].Kind != "Dienstleister" || contacts[0].Email != "eva@example.com" || !contacts[0].Active {
		t.Fatalf("saved contacts = %+v", contacts)
	}
	if events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionContactSave}); len(events) != 1 {
		t.Fatalf("contact save audit events = %+v", events)
	}

	if _, _, err := testRepositories(a, "other").contacts.Upsert(managedContact{
		TenantSlug: "other",
		Kind:       "Notdienst",
		Name:       "Fremder Notdienst",
		Email:      "other@example.com",
		Active:     true,
	}); err != nil {
		t.Fatalf("other tenant contact: %v", err)
	}

	residentPage := authedRequest(t, a, "resident@example.com", "/demo/app/kontakte").Body.String()
	if !strings.Contains(residentPage, "Eva Elektrik") || !strings.Contains(residentPage, "eva@example.com") {
		t.Fatalf("resident contacts missing active address book entry:\n%s", residentPage)
	}
	if strings.Contains(residentPage, "Fremder Notdienst") || strings.Contains(residentPage, "Inaktiv") || strings.Contains(residentPage, "Kontakt speichern") {
		t.Fatalf("resident contacts leaked other tenant, inactive, or management UI:\n%s", residentPage)
	}

	if _, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:   "demo",
		AuthorEmail:  "resident@example.com",
		AuthorName:   "Resident",
		Category:     "Reparatur",
		Title:        "Licht prüfen",
		Body:         "Das Licht flackert.",
		LocationType: issueLocationCommon,
		Status:       issueStatusOpen,
		Priority:     issuePriorityNorm,
	}); err != nil {
		t.Fatalf("create issue for service-provider chooser: %v", err)
	}
	board := authedRequest(t, a, "manager@example.com", "/demo/app/anliegen/board").Body.String()
	if !strings.Contains(board, `datalist id="service-provider-contacts"`) || !strings.Contains(board, `value="eva@example.com"`) || !strings.Contains(board, `list="service-provider-contacts"`) {
		t.Fatalf("issue board missing service-provider contact chooser:\n%s", board)
	}

	forbidden := authedFormRequest(t, a, "resident@example.com", "/demo/app/kontakte", url.Values{
		"kind":   {"Dienstleister"},
		"name":   {"Should Fail"},
		"email":  {"fail@example.com"},
		"active": {"true"},
	})
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("resident contact save status = %d, want 403", forbidden.Code)
	}

	deleteReq := authedFormRequest(t, a, "manager@example.com", "/demo/app/kontakte/delete", url.Values{"id": {contacts[0].ID}})
	if deleteReq.Code != http.StatusSeeOther {
		t.Fatalf("contact deactivate status = %d, want redirect", deleteReq.Code)
	}
	contacts = testRepositories(a, "demo").contacts.List(true)
	if len(contacts) != 1 || contacts[0].Active {
		t.Fatalf("deactivated contacts = %+v", contacts)
	}
	residentAfterDelete := authedRequest(t, a, "resident@example.com", "/demo/app/kontakte").Body.String()
	if strings.Contains(residentAfterDelete, "Eva Elektrik") {
		t.Fatalf("resident contacts should hide inactive entries:\n%s", residentAfterDelete)
	}
	managerPage := authedRequest(t, a, "manager@example.com", "/demo/app/kontakte").Body.String()
	if !strings.Contains(managerPage, "Eva Elektrik") || !strings.Contains(managerPage, "Inaktiv") {
		t.Fatalf("manager contacts should show inactive entries:\n%s", managerPage)
	}
	if events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionContactDelete}); len(events) != 1 {
		t.Fatalf("contact deactivate audit events = %+v", events)
	}
}

func TestManualUnitPaymentStatusVisibilityAndAudit(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["owner@example.com"] = userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["other@example.com"] = userProfile{Email: "other@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{
		{ID: "top-1", TenantSlug: "demo", Label: "Top 1", UnitType: unitTypeResidential, MiteigentumsanteilPPM: 100000, OwnerEmails: []string{"owner@example.com"}},
		{ID: "top-2", TenantSlug: "demo", Label: "Top 2", UnitType: unitTypeResidential, MiteigentumsanteilPPM: 100000, OwnerEmails: []string{"other@example.com"}},
	}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}

	forbidden := authedFormRequest(t, a, "owner@example.com", "/demo/app/settings/building/payment-status", url.Values{
		"unit_id": {"top-1"},
		"status":  {unitPaymentStatusPaid},
	})
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("resident payment status update status = %d, want 403", forbidden.Code)
	}

	saveTop1 := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/building/payment-status", url.Values{
		"unit_id": {"top-1"},
		"status":  {"überfällig"},
	})
	if saveTop1.Code != http.StatusSeeOther {
		t.Fatalf("top-1 payment status update = %d, want redirect", saveTop1.Code)
	}
	saveTop2 := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/building/payment-status", url.Values{
		"unit_id": {"top-2"},
		"status":  {unitPaymentStatusPaid},
	})
	if saveTop2.Code != http.StatusSeeOther {
		t.Fatalf("top-2 payment status update = %d, want redirect", saveTop2.Code)
	}

	records := testUnitPaymentRepository(t, a, "demo").List()
	if len(records) != 2 || records[0].UnitID != "top-1" || records[0].Status != unitPaymentStatusOverdue || records[1].Status != unitPaymentStatusPaid {
		t.Fatalf("payment status records = %+v", records)
	}

	// The release-notes panel is part of every portal page and may legitimately
	// name a unit label, which would otherwise read as a leak below.
	ownerPage := withoutReleaseNotes(authedRequest(t, a, "owner@example.com", "/demo/app").Body.String())
	if !strings.Contains(ownerPage, "Offenen Zahlungsstatus klären") || !strings.Contains(ownerPage, "Top 1") || !strings.Contains(ownerPage, "Überfällig") {
		t.Fatalf("owner page missing own payment status:\n%s", ownerPage)
	}
	if strings.Contains(ownerPage, "Top 2") || strings.Contains(ownerPage, "Bezahlt") {
		t.Fatalf("owner page leaked other unit payment status:\n%s", ownerPage)
	}

	buildingPage := authedRequest(t, a, "manager@example.com", "/demo/app/settings/building?section=units").Body.String()
	if !strings.Contains(buildingPage, "Zahlungsstatus") || !strings.Contains(buildingPage, "Top 1") || !strings.Contains(buildingPage, "Top 2") || !strings.Contains(buildingPage, "Überfällig") || !strings.Contains(buildingPage, "Bezahlt") {
		t.Fatalf("manager building settings missing payment status overview:\n%s", buildingPage)
	}

	events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionUnitPayment, Limit: 10})
	if len(events) != 2 {
		t.Fatalf("payment audit events = %+v, want 2", events)
	}
	if events[0].Details["status"] != "Bezahlt" || events[1].Details["status"] != "Überfällig" {
		t.Fatalf("payment audit status details = %+v", events)
	}
}

func TestServiceProviderAttachmentAccessIsBoundToAssignedOpenIssue(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "service@example.com", Role: roleServiceProvider, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.serviceAccessEnabled = true
	a.profiles["other-service@example.com"] = userProfile{Email: "other-service@example.com", Role: roleServiceProvider, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}

	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:    "demo",
		AuthorEmail:   "resident@example.com",
		AuthorName:    "Resident",
		Category:      "Reparatur",
		Title:         "Wasserschaden Keller",
		Body:          "Bitte rasch ansehen.",
		LocationType:  issueLocationCommon,
		AssigneeEmail: "service@example.com",
	})
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}
	issueAttachments, err := attachmentRepositoryForTest(a, "demo").CreateUploaded("issue", issue.ID, "resident@example.com", []uploadedFile{
		testMultipartHeader(t, "attachments", "schaden.png", minimalPNG()),
	}, time.Now())
	if err != nil || len(issueAttachments) != 1 {
		t.Fatalf("issue attachment = %+v err=%v", issueAttachments, err)
	}
	commented, found, err := issueRepositoryForTest(a, "demo").AddComment(issue.ID, issueComment{
		AuthorEmail: "service@example.com",
		AuthorName:  "Dienstleister",
		Body:        "Foto nach Erstbesichtigung.",
	})
	if err != nil || !found || len(commented.Comments) != 1 {
		t.Fatalf("AddComment issue=%+v found=%v err=%v", commented, found, err)
	}
	commentAttachments, err := attachmentRepositoryForTest(a, "demo").CreateUploaded("issue-comment", commented.Comments[0].ID, "service@example.com", []uploadedFile{
		testMultipartHeader(t, "attachments", "bestand.png", minimalPNG()),
	}, time.Now())
	if err != nil || len(commentAttachments) != 1 {
		t.Fatalf("comment attachment = %+v err=%v", commentAttachments, err)
	}

	for _, item := range []attachmentRecord{issueAttachments[0], commentAttachments[0]} {
		assigned := authedRequest(t, a, "service@example.com", "/demo/app/attachments/"+item.ID+"/thumb")
		if assigned.Code != http.StatusOK {
			t.Fatalf("assigned service attachment %s status = %d, want 200", item.EntityType, assigned.Code)
		}
		unassigned := authedRequest(t, a, "other-service@example.com", "/demo/app/attachments/"+item.ID+"/thumb")
		if unassigned.Code != http.StatusForbidden {
			t.Fatalf("unassigned service attachment %s status = %d, want 403", item.EntityType, unassigned.Code)
		}
		anonymousReq := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app/attachments/"+item.ID+"/thumb", nil)
		anonymous := httptest.NewRecorder()
		a.handler().ServeHTTP(anonymous, anonymousReq)
		if anonymous.Code != http.StatusSeeOther || anonymous.Header().Get("Location") != "/demo/" {
			t.Fatalf("anonymous attachment %s status = %d location=%q, want redirect to tenant login", item.EntityType, anonymous.Code, anonymous.Header().Get("Location"))
		}
	}

	_, _, err = issueRepositoryForTest(a, "demo").UpdateWorkflow(issue.ID, issueWorkflowUpdate{
		ActorEmail: "manager@example.com",
		Status:     issueStatusDone,
		Priority:   issuePriorityNorm,
	})
	if err != nil {
		t.Fatalf("close issue: %v", err)
	}
	closed := authedRequest(t, a, "service@example.com", "/demo/app/attachments/"+issueAttachments[0].ID+"/thumb")
	if closed.Code != http.StatusForbidden {
		t.Fatalf("closed issue attachment status = %d, want 403", closed.Code)
	}
}

func TestIssueEstimateMetadataAndAttachmentUseProtectedRoutes(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "service@example.com", Role: roleServiceProvider, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.serviceAccessEnabled = true
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}

	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:    "demo",
		AuthorEmail:   "resident@example.com",
		AuthorName:    "Resident",
		Category:      "Reparatur",
		Title:         "Türschließer defekt",
		Body:          "Bitte Angebot einholen.",
		LocationType:  issueLocationCommon,
		AssigneeEmail: "service@example.com",
	})
	if err != nil {
		t.Fatalf("Create issue: %v", err)
	}

	save := authedMultipartFilesRequest(t, a, "service@example.com", "/demo/app/anliegen/workflow", map[string]string{
		"id":              issue.ID,
		"status":          issueStatusOpen,
		"estimate_amount": "240,50",
		"estimate_note":   "Material und Anfahrt grob geschätzt.",
	}, []multipartTestFile{{Field: "estimate_attachment", Filename: "kostenvoranschlag.pdf", Body: []byte("%PDF-1.4\n% angebot\n")}})
	if save.Code != http.StatusSeeOther {
		t.Fatalf("estimate save status = %d, want redirect", save.Code)
	}
	if loc := save.Header().Get("Location"); loc != "/demo/app/anliegen?issue=updated" {
		t.Fatalf("estimate save redirect = %q", loc)
	}
	updated, ok := issueRepositoryForTest(a, "demo").Get(issue.ID)
	if !ok || updated.EstimateAmountCents != 24050 || updated.EstimateNote != "Material und Anfahrt grob geschätzt." || updated.EstimateUpdatedBy != "service@example.com" {
		t.Fatalf("stored estimate = %+v ok=%v", updated, ok)
	}
	attachments := attachmentRepositoryForTest(a, "demo").ListEntity("issue-estimate", issue.ID)
	if len(attachments) != 1 || attachments[0].ContentType != "application/pdf" {
		t.Fatalf("estimate attachments = %+v", attachments)
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionIssueEstimate, Limit: 10})
	if len(events) != 1 || events[0].Details["estimate_amount"] != "240,50 €" || events[0].Details["file_count"] != "1" || events[0].Details["has_file"] != "true" {
		t.Fatalf("estimate audit events = %+v", events)
	}
	if _, ok := events[0].Details["estimate_note"]; ok {
		t.Fatalf("estimate note should not be persisted in audit details: %+v", events[0])
	}
	for _, value := range events[0].Details {
		if strings.Contains(value, "kostenvoranschlag.pdf") {
			t.Fatalf("estimate filename should not be persisted in audit details: %+v", events[0])
		}
	}
	residentPage := authedRequest(t, a, "resident@example.com", "/demo/app/anliegen/"+issue.ID).Body.String()
	for _, want := range []string{"Kostenvoranschlag", "240,50 €", "Material und Anfahrt grob geschätzt.", "kostenvoranschlag.pdf", "keine Rechnung und kein Zahlungsstatus"} {
		if !strings.Contains(residentPage, want) {
			t.Fatalf("resident estimate view missing %q:\n%s", want, residentPage)
		}
	}
	served := authedRequest(t, a, "resident@example.com", "/demo/app/attachments/"+attachments[0].ID)
	if served.Code != http.StatusOK {
		t.Fatalf("resident estimate attachment status = %d, want 200", served.Code)
	}
	if got := served.Header().Get("Content-Disposition"); !strings.Contains(got, "inline") || !strings.Contains(got, "kostenvoranschlag.pdf") {
		t.Fatalf("estimate attachment disposition = %q", got)
	}

	forbidden := authedFormRequest(t, a, "resident@example.com", "/demo/app/anliegen/workflow", url.Values{
		"id":              {issue.ID},
		"status":          {issueStatusDone},
		"estimate_amount": {"1,00"},
	})
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("resident estimate update status = %d, want 403", forbidden.Code)
	}
}

func TestIssueTriageBoardFiltersAndOpenCounts(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	old := time.Now().Add(-48 * time.Hour)
	for _, item := range []residentIssue{
		{TenantSlug: "demo", AuthorEmail: "resident@example.com", AuthorName: "Resident", Category: "Reparatur", Title: "Urgent repair", Body: "Broken.", LocationType: issueLocationCommon, Status: issueStatusProgress, Priority: issuePriorityUrgent, AssigneeEmail: "manager@example.com", CreatedAt: old, UpdatedAt: old},
		{TenantSlug: "demo", AuthorEmail: "resident@example.com", AuthorName: "Resident", Category: "Frage", Title: "Regular question", Body: "Question.", LocationType: issueLocationCommon, Status: issueStatusNew, Priority: issuePriorityNorm},
		{TenantSlug: "demo", AuthorEmail: "resident@example.com", AuthorName: "Resident", Category: "Vorschlag", Title: "Closed suggestion", Body: "Done.", LocationType: issueLocationCommon, Status: issueStatusDone, Priority: issuePriorityLow},
	} {
		if _, err := issueRepositoryForTest(a, "demo").Create(item); err != nil {
			t.Fatalf("Create issue %q: %v", item.Title, err)
		}
	}

	residentBoard := authedRequest(t, a, "resident@example.com", "/demo/app/anliegen/board")
	if residentBoard.Code != http.StatusForbidden {
		t.Fatalf("resident board status = %d, want 403", residentBoard.Code)
	}
	board := authedRequest(t, a, "manager@example.com", "/demo/app/anliegen/board?status=In+Bearbeitung&priority=Dringend&category=Reparatur&assignee=manager@example.com&sort=age")
	if board.Code != http.StatusOK {
		t.Fatalf("manager board status = %d", board.Code)
	}
	body := board.Body.String()
	for _, want := range []string{"Anliegen bearbeiten", "Urgent repair", `name="assignee"`, "Zurücksetzen"} {
		if !strings.Contains(body, want) {
			t.Fatalf("triage board should contain %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "Regular question") || strings.Contains(body, "Closed suggestion") {
		t.Fatalf("triage board filter leaked other issues:\n%s", body)
	}

	dashboard := authedRequest(t, a, "manager@example.com", "/demo/app").Body.String()
	if !strings.Contains(dashboard, ">2</strong><span>offene Anliegen") || !strings.Contains(dashboard, "nav-badge") {
		t.Fatalf("dashboard should surface open issue count:\n%s", dashboard)
	}
}

func TestNotificationFrameworkDedupesRecipientsAndHonorsPrefs(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "actor@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["enabled@example.com"] = userProfile{Email: "enabled@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["muted@example.com"] = userProfile{Email: "muted@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["unsubscribed@example.com"] = userProfile{Email: "unsubscribed@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
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
		Tenant:     a.tenants["demo"],
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
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	mailer := &recordingMailer{}
	a.mailer = mailer

	create := authedFormRequest(t, a, "manager@example.com", "/demo/app/announcements", url.Values{
		"title":    {"Liftwartung"},
		"body":     {"Lift am Freitag außer Betrieb."},
		"category": {"Wartung"},
	})
	if create.Code != http.StatusSeeOther {
		t.Fatalf("announcement create status = %d", create.Code)
	}
	items := testRepositories(a, "demo").announcements.List()
	if len(items) != 1 {
		t.Fatalf("announcements = %+v", items)
	}
	if len(mailer.notifications) != 1 || mailer.notifications[0].To != "resident@example.com" || !strings.Contains(mailer.notifications[0].Subject, "Neuer Aushang") || !strings.Contains(mailer.notifications[0].Body, "/demo/app/announcements#announcement-"+items[0].ID) {
		t.Fatalf("announcement notifications = %+v", mailer.notifications)
	}
	page := authedRequest(t, a, "resident@example.com", "/demo/app/announcements")
	if !strings.Contains(page.Body.String(), `id="announcement-`+items[0].ID+`"`) {
		t.Fatalf("announcement page missing deeplink anchor:\n%s", page.Body.String())
	}
}

func minimalPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		panic(err)
	}
	return out.Bytes()
}

func TestAnnouncementArchiveFiltersSearchesAndIncludesPast(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	now := time.Now().Add(-2 * time.Hour)
	expiredAt := time.Now().Add(-time.Hour)
	_, _ = testRepositories(a, "demo").announcements.Create(announcement{TenantSlug: "demo", Title: "Liftwartung", Body: "Lift Freitag", Category: "Wartung", PublishedAt: now})
	_, _ = testRepositories(a, "demo").announcements.Create(announcement{TenantSlug: "demo", Title: "Hausfest", Body: "Sommertermin", Category: "Termin", PublishedAt: now})
	_, _ = testRepositories(a, "demo").announcements.Create(announcement{TenantSlug: "demo", Title: "Alter Hinweis", Body: "Vergangen", Category: "Info", PublishedAt: now.Add(-time.Hour), ExpiresAt: &expiredAt})
	_, _ = testRepositories(a, "demo").announcements.Create(announcement{TenantSlug: "demo", Title: "Geplant", Body: "Noch nicht sichtbar", Category: "Info", PublishedAt: time.Now().Add(time.Hour)})

	all := authedRequest(t, a, "resident@example.com", "/demo/app/announcements")
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

	filtered := authedRequest(t, a, "resident@example.com", "/demo/app/announcements?category=Wartung&q=Lift")
	if filtered.Code != http.StatusOK {
		t.Fatalf("filtered archive status = %d", filtered.Code)
	}
	body = filtered.Body.String()
	if !strings.Contains(body, "Liftwartung") || strings.Contains(body, "Hausfest") || strings.Contains(body, "Alter Hinweis") {
		t.Fatalf("filtered archive body did not match expected search/category result:\n%s", body)
	}
}

func TestAnnouncementUnreadBadgeClearsAfterArchiveView(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	_, _ = testRepositories(a, "demo").announcements.Create(announcement{TenantSlug: "demo", Title: "Neue Wartung", Body: "Heute", Category: "Wartung", PublishedAt: time.Now().Add(-time.Hour)})

	before := authedRequest(t, a, "resident@example.com", "/demo/app")
	if before.Code != http.StatusOK {
		t.Fatalf("portal before status = %d", before.Code)
	}
	if body := before.Body.String(); !strings.Contains(body, "Neuen Aushang lesen") || !strings.Contains(body, "1 ungelesener Beitrag") || !strings.Contains(body, `<span class="nav-badge">1</span>`) {
		t.Fatalf("portal should show unread announcement and nav badge before archive view:\n%s", body)
	}

	archive := authedRequest(t, a, "resident@example.com", "/demo/app/announcements")
	if archive.Code != http.StatusOK {
		t.Fatalf("archive status = %d", archive.Code)
	}
	repository, ok := storepkg.BindAnnouncementReadRepository(a.announcementReadStore, testTenantRef("demo"))
	if !ok {
		t.Fatal("bind announcement read repository")
	}
	if got := repository.LastSeen("resident@example.com"); got.IsZero() {
		t.Fatal("archive view should mark announcements as seen")
	}

	after := authedRequest(t, a, "resident@example.com", "/demo/app")
	if after.Code != http.StatusOK {
		t.Fatalf("portal after status = %d", after.Code)
	}
	body := after.Body.String()
	if strings.Contains(body, `<span class="nav-badge">`) || strings.Contains(body, "ungelesener Beitrag") {
		t.Fatalf("portal should clear unread badges after archive view:\n%s", body)
	}
}

func TestAnnouncementRoutesGateWritesAndAllowResidentRead(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	read := authedRequest(t, a, "resident@example.com", "/demo/app/announcements")
	if read.Code != http.StatusOK {
		t.Fatalf("resident announcement read status = %d", read.Code)
	}
	write := authedFormRequest(t, a, "resident@example.com", "/demo/app/announcements", url.Values{
		"title": {"Resident post"},
		"body":  {"Nope"},
	})
	if write.Code != http.StatusForbidden {
		t.Fatalf("resident create status = %d, want 403", write.Code)
	}
}

func TestAnnouncementCreateRejectsCrossOriginAndPersistsSameOrigin(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	values := url.Values{
		"title":        {"Liftwartung"},
		"body":         {"Der Lift ist am Freitag vormittags außer Betrieb."},
		"category":     {"Wartung"},
		"published_at": {"2026-07-06T12:30"},
		"pinned":       {"true"},
	}

	cross := authedFormRequestWithOrigin(t, a, "admin@example.com", "/demo/app/announcements", values, "https://evil.example")
	if cross.Code != http.StatusForbidden {
		t.Fatalf("cross-origin create status = %d, want 403", cross.Code)
	}

	same := authedFormRequest(t, a, "admin@example.com", "/demo/app/announcements", values)
	if same.Code != http.StatusSeeOther {
		t.Fatalf("same-origin create status = %d, want redirect", same.Code)
	}
	visible := testRepositories(a, "demo").announcements.Visible(time.Date(2026, 7, 6, 13, 0, 0, 0, time.Local))
	if len(visible) != 1 || visible[0].Title != "Liftwartung" || !visible[0].Pinned {
		t.Fatalf("created visible announcement = %+v", visible)
	}
}

func newTestPortalApp(t *testing.T, profile userProfile) *app {
	t.Helper()
	tmpl, err := template.New("pages").Parse(web.PageTemplates)
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	profile.Email = normalizeEmail(profile.Email)
	profile.Role = normalizeRole(profile.Role)
	profile.Permissions = normalizePermissions(profile.Permissions)
	profile.TenantMemberships = normalizeTenantMemberships(profile.TenantMemberships)
	profile.Tenants = normalizeTenants(append(profile.Tenants, tenantMembershipSlugs(profile.TenantMemberships)...), "demo")
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
	annualStatementPeriods := storepkg.NewMemoryAnnualStatementPeriodStore()
	annualStatementCostTypes := storepkg.NewMemoryAnnualStatementCostTypeStore()
	unitStore, err := newUnitStore("")
	if err != nil {
		t.Fatalf("unit store: %v", err)
	}
	unitPaymentStore, err := newUnitPaymentStatusStore("")
	if err != nil {
		t.Fatalf("unit payment status store: %v", err)
	}
	issueStore, err := newIssueStore("", "")
	if err != nil {
		t.Fatalf("issue store: %v", err)
	}
	attachmentStore, err := newAttachmentStore("", filepath.Join(t.TempDir(), "attachments"))
	if err != nil {
		t.Fatalf("attachment store: %v", err)
	}
	contactStore, err := newContactBookStore("")
	if err != nil {
		t.Fatalf("contact store: %v", err)
	}
	auditStore, err := newAuditStore("")
	if err != nil {
		t.Fatalf("audit store: %v", err)
	}
	documentStore, err := newDocumentStore("", filepath.Join(t.TempDir(), "documents"))
	if err != nil {
		t.Fatalf("document store: %v", err)
	}
	annualStatementReceipts := storepkg.NewMemoryAnnualStatementReceiptStore(annualStatementPeriods, annualStatementCostTypes, documentStore)
	annualStatementAkontos := storepkg.NewMemoryAnnualStatementPrepaymentStore(annualStatementPeriods, unitStore)
	annualConsumption := storepkg.NewMemoryAnnualStatementConsumptionStore()
	annualStatementRuns := storepkg.NewMemoryAnnualStatementRunStore(storepkg.AnnualStatementRunSources{Periods: annualStatementPeriods, Units: unitStore, Receipts: annualStatementReceipts, Prepayments: annualStatementAkontos, Consumption: annualConsumption, Documents: documentStore})
	handoverStore, err := newHandoverStore("")
	if err != nil {
		t.Fatalf("handover store: %v", err)
	}
	voteStore, err := newVoteStore("")
	if err != nil {
		t.Fatalf("vote store: %v", err)
	}
	homeReservations := storepkg.NewMemoryHomeReservationStore()
	a := &app{
		baseURL:       "http://localhost:8080",
		rootDomain:    "hausv.org",
		defaultTenant: "demo",
		tenants: map[string]tenantConfig{
			"demo": {Slug: "demo", Name: "WEG Portal", Address: "Musterweg 1", MapLatitude: 48.2082, MapLongitude: 16.3738, MapZoom: 17, HeroImageURL: defaultTenantHeroImageURL},
		},
		tenantIdentities: map[string]storepkg.TenantIdentity{
			"demo": {ID: testTenantRef("demo").ID, Slug: "demo", Name: "WEG Portal"},
		},
		profiles: map[string]userProfile{
			profile.Email: profile,
		},
		allowed:                  map[string]struct{}{},
		admins:                   map[string]struct{}{},
		sessionTTL:               time.Hour,
		tokens:                   auth.NewTokenStore([]byte(strings.Repeat("t", 32))),
		sessions:                 newSessionStore([]byte(strings.Repeat("s", 32))),
		homeSetupTokens:          auth.NewTokenStore([]byte(strings.Repeat("h", 32))),
		homeSetupSessions:        newSessionStore([]byte(strings.Repeat("u", 32))),
		homeConnectorHashKey:     []byte(strings.Repeat("c", 32)),
		oidc:                     &oidcLogin{},
		mailer:                   smtpMailer{},
		templates:                tmpl,
		announcementStore:        announcementStore,
		announcementReadStore:    announcementReadStore,
		eventStore:               eventStore,
		notificationPrefs:        notificationPrefStore,
		profileOverlays:          profileOverlayStore,
		tenantOverrides:          tenantOverrideStore,
		tenantHeroDir:            filepath.Join(t.TempDir(), "tenant-heroes"),
		inviteStore:              inviteStore,
		personAvatars:            storepkg.NewMemoryPersonAvatarStore(),
		activityStore:            activityStore,
		annualStatementCostTypes: annualStatementCostTypes,
		annualStatementPeriods:   annualStatementPeriods,
		annualStatementAkontos:   annualStatementAkontos,
		annualStatementReceipts:  annualStatementReceipts,
		annualConsumption:        annualConsumption,
		annualStatementRuns:      annualStatementRuns,
		unitStore:                unitStore,
		unitPaymentStore:         unitPaymentStore,
		issueStore:               issueStore,
		attachmentStore:          attachmentStore,
		contactStore:             contactStore,
		auditStore:               auditStore,
		documentStore:            documentStore,
		handoverStore:            handoverStore,
		voteStore:                voteStore,
		parkingStore:             parkingStore,
		energyStore:              energy.NewMemoryStore(),
		homeReservations:         homeReservations,
		homePortals:              storepkg.NewMemoryHomePortalStore(homeReservations, inviteStore),
		homeConnectors:           storepkg.NewMemoryHomeConnectorStore(),
		homeConnectorReadings:    storepkg.NewMemoryHomeConnectorReadingStore(),
		homeConnectorDownloadDir: t.TempDir(),
	}
	t.Cleanup(a.closeMagicLinkDelivery)
	return a
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

func authedRequest(t *testing.T, a *app, email string, path string) *httptest.ResponseRecorder {
	t.Helper()
	token, _, err := a.sessions.Put(email, "demo", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://hausv.org"+path, nil)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	return rr
}

func authedFormRequest(t *testing.T, a *app, email string, path string, values url.Values) *httptest.ResponseRecorder {
	return authedFormRequestWithOrigin(t, a, email, path, values, "http://hausv.org/demo")
}

func authedMultipartRequest(t *testing.T, a *app, email string, path string, fields map[string]string, filename string, fileBody []byte) *httptest.ResponseRecorder {
	return authedMultipartFileRequest(t, a, email, path, fields, "photo", filename, fileBody)
}

type multipartTestFile struct {
	Field    string
	Filename string
	Body     []byte
}

func authedMultipartFilesRequest(t *testing.T, a *app, email string, path string, fields map[string]string, files []multipartTestFile) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("WriteField %s: %v", key, err)
		}
	}
	for _, file := range files {
		part, err := writer.CreateFormFile(file.Field, file.Filename)
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}
		if _, err := part.Write(file.Body); err != nil {
			t.Fatalf("write multipart file: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("multipart close: %v", err)
	}
	token, _, err := a.sessions.Put(email, "demo", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://hausv.org"+path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", "http://hausv.org/demo")
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	return rr
}

func testMultipartHeader(t *testing.T, field string, filename string, fileBody []byte) uploadedFile {
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
	req := httptest.NewRequest(http.MethodPost, "http://hausv.org/demo/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(maxDocumentBytes); err != nil {
		t.Fatalf("ParseMultipartForm: %v", err)
	}
	files := req.MultipartForm.File[field]
	if len(files) != 1 {
		t.Fatalf("multipart files for %s = %d, want 1", field, len(files))
	}
	return uploadedFileFromHeader(files[0])
}

func authedMultipartFileRequest(t *testing.T, a *app, email string, path string, fields map[string]string, fileField string, filename string, fileBody []byte) *httptest.ResponseRecorder {
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
	token, _, err := a.sessions.Put(email, "demo", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://hausv.org"+path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", "http://hausv.org/demo")
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	return rr
}

func authedFormRequestWithOrigin(t *testing.T, a *app, email string, path string, values url.Values, origin string) *httptest.ResponseRecorder {
	t.Helper()
	token, _, err := a.sessions.Put(email, "demo", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://hausv.org"+path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", origin)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
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

func TestParkingEmptyStateGuidesSetupWithoutPaymentControls(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "parker@example.com", FirstName: "Pat", LastName: "Parker", Role: roleRenter, Tenants: []string{"demo"}, Permissions: []string{permissionParking}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("Add invite: %v", err)
	}

	adminPage := authedRequest(t, a, "admin@example.com", "/demo/app/parking")
	adminBody := adminPage.Body.String()
	for _, want := range []string{"parking-empty", "Bereit für die erste Abrechnung", "Noch keine Monatswerte", "Abrechnung konfigurieren", "Zugriff verwalten", "Tarif", "Messwerte", "Nur Nachweis", "Keine Buchung"} {
		if !strings.Contains(adminBody, want) {
			t.Fatalf("admin empty parking page missing %q:\n%s", want, adminBody)
		}
	}
	for _, hidden := range []string{`class="parking-workspace"`, `class="parking-month-queue"`, `class="pay-form"`, "Zahlungsart", "Belege ablegen"} {
		if strings.Contains(adminBody, hidden) {
			t.Fatalf("admin empty parking page should not render %q:\n%s", hidden, adminBody)
		}
	}

	residentPage := authedRequest(t, a, "parker@example.com", "/demo/app/parking")
	residentBody := residentPage.Body.String()
	for _, want := range []string{"parking-empty", "Hausüberblick öffnen", "Nur Nachweis", "Keine Buchung"} {
		if !strings.Contains(residentBody, want) {
			t.Fatalf("resident empty parking page missing %q:\n%s", want, residentBody)
		}
	}
	for _, hidden := range []string{`href="/demo/app/parking/settings"`, `href="/demo/app/settings/parking-access"`, "Zugriff verwalten"} {
		if strings.Contains(residentBody, hidden) {
			t.Fatalf("resident empty parking page should not expose %q:\n%s", hidden, residentBody)
		}
	}
}

func TestParkingPaymentMetadataAndOutstandingVisibility(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "parker@example.com", FirstName: "Pat", LastName: "Parker", Role: roleRenter, Tenants: []string{"demo"}, Permissions: []string{permissionParking}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("Add invite: %v", err)
	}
	base := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	if err := a.parkingStore.AppendReadings("demo", []parkingNumericSample{
		{At: base, Value: 100},
		{At: base.Add(2 * time.Hour), Value: 102},
	}, []parkingNumericSample{
		{At: base, Value: 0.20},
		{At: base.Add(time.Hour), Value: 0.40},
	}); err != nil {
		t.Fatalf("AppendReadings: %v", err)
	}

	parkerPage := authedRequest(t, a, "parker@example.com", "/demo/app/parking")
	if parkerPage.Code != http.StatusOK || !strings.Contains(parkerPage.Body.String(), "0,80 €") || !strings.Contains(parkerPage.Body.String(), "OFFEN") {
		t.Fatalf("parker outstanding page = %d\n%s", parkerPage.Code, parkerPage.Body.String())
	}
	adminParking := authedRequest(t, a, "admin@example.com", "/demo/app/parking")
	adminBody := adminParking.Body.String()
	// The overview keeps one current-month summary and moves payment details to
	// the dedicated month page instead of rendering the same figures twice.
	for _, want := range []string{"parking-months-panel", "parking-current-month", "Neuester Monat", `/demo/app/parking/month/2026-06`, "Wie wird gerechnet?"} {
		if !strings.Contains(adminBody, want) {
			t.Fatalf("parking overview redesign missing %q:\n%s", want, adminBody)
		}
	}
	for _, old := range []string{`class="parking-workspace"`, `id="parking-detail-2026-06"`, `class="parking-month-table"`, `<table>`, `class="month-strip"`, `class="pay-form"`, `name="attachments"`, "Belege ablegen"} {
		if strings.Contains(adminBody, old) {
			t.Fatalf("parking overview should not render old dense/payment UI %q:\n%s", old, adminBody)
		}
	}
	adminMonth := authedRequest(t, a, "admin@example.com", "/demo/app/parking/month/2026-06")
	for _, want := range []string{"parking-month-summary", "Bezahlung erhalten", "Details (optional)", `name="payment_reference"`, "Kosten aufschlüsseln", "Stundenwerte"} {
		if !strings.Contains(adminMonth.Body.String(), want) {
			t.Fatalf("parking month redesign missing %q:\n%s", want, adminMonth.Body.String())
		}
	}
	residentMark := authedFormRequest(t, a, "parker@example.com", "/demo/app/parking/month", url.Values{
		"month": {"2026-06"},
		"paid":  {"true"},
	})
	if residentMark.Code != http.StatusSeeOther {
		t.Fatalf("resident payment mark status = %d, want redirect", residentMark.Code)
	}
	residentState := a.parkingStore.TenantData("demo").Months["2026-06"]
	if !residentState.Paid || residentState.PaidBy != "parker@example.com" {
		t.Fatalf("resident payment state = %+v", residentState)
	}
	if err := a.parkingStore.SetMonthPaid("demo", "2026-06", false); err != nil {
		t.Fatalf("reset resident payment: %v", err)
	}
	residentParking := authedRequest(t, a, "parker@example.com", "/demo/app/parking/month/2026-06")
	if residentParking.Code != http.StatusOK || !strings.Contains(residentParking.Body.String(), "Als bezahlt markieren") || strings.Contains(residentParking.Body.String(), "Bezahlung erhalten") {
		t.Fatalf("resident parking action mismatch = %d\n%s", residentParking.Code, residentParking.Body.String())
	}
	accessPage := authedRequest(t, a, "admin@example.com", "/demo/app/settings/parking-access")
	if accessPage.Code != http.StatusOK || !strings.Contains(accessPage.Body.String(), "parker@example.com") || strings.Contains(accessPage.Body.String(), "0,80 €") {
		t.Fatalf("access page should show permissions without mixing in balances = %d\n%s", accessPage.Code, accessPage.Body.String())
	}
	accountingPage := authedRequest(t, a, "admin@example.com", "/demo/app/parking/settings?section=accounting")
	for _, want := range []string{"Zahlungsstände prüfen", `/demo/app/parking?view=months#monate`, "Offene Zahlungen erinnern"} {
		if accountingPage.Code != http.StatusOK || !strings.Contains(accountingPage.Body.String(), want) {
			t.Fatalf("accounting flow missing %q = %d\n%s", want, accountingPage.Code, accountingPage.Body.String())
		}
	}

	save := authedMultipartFileRequest(t, a, "admin@example.com", "/demo/app/parking/month", map[string]string{
		"month":             "2026-06",
		"paid":              "true",
		"paid_at":           "2026-07-05",
		"payment_method":    "Überweisung",
		"payment_reference": "ABC-123",
	}, "attachments", "zahlungsbeleg.pdf", []byte("%PDF-1.4\n% beleg\n"))
	if save.Code != http.StatusSeeOther {
		t.Fatalf("payment save status = %d, want redirect", save.Code)
	}
	if loc := save.Header().Get("Location"); loc != "/demo/app/parking/month/2026-06?month=saved" {
		t.Fatalf("payment save redirect = %q", loc)
	}
	state := a.parkingStore.TenantData("demo").Months["2026-06"]
	if !state.Paid || state.PaidBy != "admin@example.com" || state.PaymentMethod != "Überweisung" || state.PaymentReference != "ABC-123" || formatLocalDate(state.PaidAt) != "05.07.2026" {
		t.Fatalf("stored payment state = %+v", state)
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionParkingMonth, Limit: 10})
	var adminPaymentEvent *auditEvent
	for i := range events {
		if events[i].ActorEmail == "admin@example.com" && events[i].Details["paid_by"] == "admin@example.com" {
			adminPaymentEvent = &events[i]
			break
		}
	}
	if adminPaymentEvent == nil || adminPaymentEvent.Details["payment_reference"] != "ABC-123" {
		t.Fatalf("payment audit events = %+v", events)
	}
	attachments := attachmentRepositoryForTest(a, "demo").ListEntity("parking", "2026-06")
	if len(attachments) != 1 || attachments[0].ContentType != "application/pdf" {
		t.Fatalf("parking attachments = %+v", attachments)
	}
	paidPage := authedRequest(t, a, "parker@example.com", "/demo/app/parking/month/2026-06")
	body := paidPage.Body.String()
	for _, want := range []string{"BEZAHLT", "Zahlung ist markiert."} {
		if !strings.Contains(body, want) {
			t.Fatalf("paid page missing %q:\n%s", want, body)
		}
	}
	for _, hidden := range []string{"Überweisung", "Ref. ABC-123", "zahlungsbeleg.pdf", "Belege ablegen", `name="payment_method"`, `name="attachments"`} {
		if strings.Contains(body, hidden) {
			t.Fatalf("paid page should hide payment metadata/control %q:\n%s", hidden, body)
		}
	}
}

func TestParkingPaymentRemindersRespectPreferencesAndDedupe(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	for _, profile := range []userProfile{
		{Email: "parker@example.com", FirstName: "Pat", LastName: "Parker", Role: roleRenter, Tenants: []string{"demo"}, Permissions: []string{permissionParking}, AuthMethods: defaultAuthMethods()},
		{Email: "muted@example.com", FirstName: "Mute", LastName: "User", Role: roleRenter, Tenants: []string{"demo"}, Permissions: []string{permissionParking}, AuthMethods: defaultAuthMethods()},
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
	if err := a.parkingStore.AppendReadings("demo", []parkingNumericSample{
		{At: base, Value: 100},
		{At: base.Add(2 * time.Hour), Value: 102},
	}, []parkingNumericSample{
		{At: base, Value: 0.20},
		{At: base.Add(time.Hour), Value: 0.40},
	}); err != nil {
		t.Fatalf("AppendReadings: %v", err)
	}

	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.Local)
	sent := a.sendParkingPaymentReminders(a.tenants["demo"], testTenantRef("demo"), "manager@example.com", roleManager, now)
	if sent != 1 || len(mailer.notifications) != 1 {
		t.Fatalf("reminders sent=%d notifications=%+v", sent, mailer.notifications)
	}
	notification := mailer.notifications[0]
	if notification.To != "parker@example.com" || !strings.Contains(notification.Subject, "Zahlungserinnerung") || !strings.Contains(notification.Body, "Juni 2026") || !strings.Contains(notification.Body, "0,80 €") || !strings.Contains(notification.Body, "/demo/app/parking#parking-month-2026-06") {
		t.Fatalf("payment reminder notification = %+v", notification)
	}
	state := a.parkingStore.TenantData("demo").Months["2026-06"]
	if _, ok := state.ReminderSentAt["parker@example.com"]; !ok {
		t.Fatalf("reminder timestamp not stored: %+v", state)
	}
	if _, ok := state.ReminderSentAt["muted@example.com"]; ok {
		t.Fatalf("muted recipient should not be marked reminded: %+v", state)
	}
	again := a.sendParkingPaymentReminders(a.tenants["demo"], testTenantRef("demo"), "manager@example.com", roleManager, now.Add(time.Hour))
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
	a := newTestPortalApp(t, userProfile{Email: "parker@example.com", FirstName: "Pat", LastName: "Parker", Role: roleRenter, Tenants: []string{"demo"}, Permissions: []string{permissionParking}, AuthMethods: defaultAuthMethods()})
	a.profiles["manager@example.com"] = userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["other@example.com"] = userProfile{Email: "other@example.com", Role: roleRenter, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	base := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	if err := a.parkingStore.AppendReadings("demo", []parkingNumericSample{
		{At: base, Value: 100},
		{At: base.Add(2 * time.Hour), Value: 102},
	}, []parkingNumericSample{
		{At: base, Value: 0.20},
		{At: base.Add(time.Hour), Value: 0.40},
	}); err != nil {
		t.Fatalf("AppendReadings: %v", err)
	}
	if err := a.parkingStore.SetMonthPaid("demo", "2026-06", true); err != nil {
		t.Fatalf("SetMonthPaid: %v", err)
	}

	resident := authedRequest(t, a, "parker@example.com", "/demo/app/parking/export/2026")
	if resident.Code != http.StatusOK {
		t.Fatalf("resident export status = %d", resident.Code)
	}
	if got := resident.Header().Get("Content-Disposition"); !strings.Contains(got, "attachment") || !strings.Contains(got, "parkplatzabrechnung-2026-parker-example.com.csv") {
		t.Fatalf("content disposition = %q", got)
	}
	body := resident.Body.String()
	for _, want := range []string{parkingStatementCSVTitle, "Pat Parker", "parker@example.com", "Juni 2026", "2,00 kWh", "0,60 €", "0,20 €", "0,00 €", "0,80 €", "BEZAHLT", "Gesamt"} {
		if !strings.Contains(body, want) {
			t.Fatalf("statement CSV missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "WEG Portal Parkplatzabrechnung") {
		t.Fatalf("statement CSV contains obsolete export title:\n%s", body)
	}

	manager := authedRequest(t, a, "manager@example.com", "/demo/app/parking/export/2026?user=parker@example.com")
	if manager.Code != http.StatusOK || !strings.Contains(manager.Body.String(), "parker@example.com") {
		t.Fatalf("manager export status/body = %d\n%s", manager.Code, manager.Body.String())
	}
	other := authedRequest(t, a, "other@example.com", "/demo/app/parking/export/2026?user=parker@example.com")
	if other.Code != http.StatusNotFound {
		t.Fatalf("other resident export status = %d, want 404", other.Code)
	}
	noParking := authedRequest(t, a, "other@example.com", "/demo/app/parking/export/2026")
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
	if err := store.SetGridFee("demo", 0.123); err != nil {
		t.Fatalf("set grid fee: %v", err)
	}
	if err := store.SetMonthPaid("demo", "2026-06", true); err != nil {
		t.Fatalf("set paid flag: %v", err)
	}

	loaded, err := newParkingStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	data := loaded.TenantData("demo")
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

// withoutReleaseNotes removes the release-notes panel from a rendered page so
// page-wide assertions are not disturbed by release copy.
func withoutReleaseNotes(page string) string {
	start := strings.Index(page, "<dialog id=\"release-history\"")
	if start < 0 {
		return page
	}
	end := strings.Index(page[start:], "</dialog>")
	if end < 0 {
		return page
	}
	return page[:start] + page[start+end+len("</dialog>"):]
}

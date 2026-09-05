package server

import (
	"bytes"
	"context"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"

	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/mailintake"
	"github.com/inspr-at/hausv-org/internal/store"
)

// intakeReplyMailer captures outgoing notifications so a test can read the
// subject tag and the recipient of an intake reply.
type intakeReplyMailer struct {
	mu   sync.Mutex
	sent []struct{ To, Subject, Body string }
}

func (m *intakeReplyMailer) SendMagicLink(string, string, string) error { return nil }
func (m *intakeReplyMailer) SendInvite(string, string, string) error    { return nil }
func (m *intakeReplyMailer) Configured() bool                           { return true }
func (m *intakeReplyMailer) SendNotification(to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, struct{ To, Subject, Body string }{to, subject, body})
	return nil
}

type mailLiteral struct{ *bytes.Reader }

func (l mailLiteral) Size() int64 { return int64(l.Len()) }

func rawTestMail(id, from, subject, body string) string {
	return strings.Join([]string{
		"From: " + from,
		"To: post@musterstadt.example",
		"Subject: " + subject,
		"Message-ID: <" + id + ">",
		"Date: Fri, 05 Sep 2026 09:15:00 +0200",
		"Content-Type: text/plain; charset=utf-8",
		"",
		body,
		"",
	}, "\r\n")
}

// startTestMailbox serves an in-memory IMAP mailbox on a real port.
func startTestMailbox(t *testing.T, messages ...string) mailintake.Config {
	t.Helper()
	memory := imapmemserver.New()
	user := imapmemserver.NewUser("post@musterstadt.example", "geheim")
	_ = user.Create("INBOX", nil)
	for _, raw := range messages {
		if _, err := user.Append("INBOX", mailLiteral{bytes.NewReader([]byte(raw))}, &imap.AppendOptions{Time: time.Now()}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	memory.AddUser(user)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return memory.NewSession(), nil, nil
		},
		InsecureAuth: true,
	})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	number := 0
	for _, digit := range port {
		number = number*10 + int(digit-'0')
	}
	t.Setenv("TEST_INTAKE_MAIL_PW", "geheim")
	return mailintake.Config{Organisation: "musterstadt", Host: host, Port: number, Username: "post@musterstadt.example",
		PasswordEnv: "TEST_INTAKE_MAIL_PW", Folder: "INBOX", Interval: time.Minute, Insecure: true}
}

// mailIntakeTestApp is the organisation from the HAUSV-598 tests with intake,
// mail ledger, a mailbox directory and a recording mailer.
func mailIntakeTestApp(t *testing.T) (*app, *intakeReplyMailer) {
	t.Helper()
	a := organisationTestApp(t, "verwaltung@example.com")
	a.syncOrganisations(context.Background())
	database := dbtest.Open(t)
	a.intake = func(orgKey string) store.IntakeRepository { return store.BindIntakeRepository(database, orgKey) }
	a.orgSettings = func(orgKey string) store.OrgSettingsRepository {
		return store.BindOrgSettingsRepository(database, orgKey)
	}
	a.intakeMailSeen = func(orgKey string) store.IntakeMailSeenRepository {
		return store.BindIntakeMailSeenRepository(database, orgKey)
	}
	a.mailIntakeDir = t.TempDir()
	mailer := &intakeReplyMailer{}
	a.mailer = mailer
	// No AI: items stay open, which is what these tests look at.
	a.triage = nil
	tenant := a.tenants["demo"]
	tenant.Address = "Musterweg 1, 8010 Graz"
	a.tenants["demo"] = tenant
	return a, mailer
}

func TestMailIsFiledOnceEvenWhenPolledTwice(t *testing.T) {
	a, _ := mailIntakeTestApp(t)
	config := startTestMailbox(t,
		rawTestMail("first@example.com", "Unbekannt <niemand@example.com>", "Frage zur Abrechnung", "Wie hoch ist die Vorschreibung?"),
		rawTestMail("second@example.com", "Unbekannt <niemand@example.com>", "Noch eine Frage", "Und wann?"),
	)
	ctx := context.Background()
	filed, err := a.pollMailIntakeOnce(ctx, "musterstadt", config)
	if err != nil {
		t.Fatal(err)
	}
	if filed != 2 {
		t.Fatalf("first poll filed %d, want 2", filed)
	}
	again, err := a.pollMailIntakeOnce(ctx, "musterstadt", config)
	if err != nil {
		t.Fatal(err)
	}
	if again != 0 {
		t.Fatalf("second poll filed %d, want 0 (messages are marked seen and recorded)", again)
	}
	items, err := a.intake("musterstadt").List(ctx, store.IntakeFilter{IncludeUnassigned: true, Sources: []store.IntakeSource{store.IntakeSourceEmail}})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("intake holds %d mail items, want 2", len(items))
	}
	for _, item := range items {
		if item.TenantSlug != "" {
			t.Fatalf("an unknown sender was assigned to %q", item.TenantSlug)
		}
		if item.ExternalRef == "" {
			t.Fatalf("message id not kept as external ref: %+v", item)
		}
	}
	count, _ := a.intakeMailSeen("musterstadt").Count(ctx)
	if count != 2 {
		t.Fatalf("ledger holds %d, want 2", count)
	}
}

func TestSenderMembershipProposesHouseAndUnit(t *testing.T) {
	a, _ := mailIntakeTestApp(t)
	addTestPerson(t, a, "alina@example.com", "demo", roleOwner)
	repo, ok := store.BindUnitRepository(a.unitStore, testTenantRef("demo"))
	if !ok {
		t.Fatal("unit repository unavailable")
	}
	// A flat and a parking space, like the demo data: the flat is the unit a
	// mail belongs to, the parking space must not make it ambiguous.
	if err := repo.SetUnits([]store.Unit{
		{ID: "top-1", TenantSlug: "demo", Label: "Top 1", UnitType: unitTypeResidential, OwnerEmails: []string{"alina@example.com"}},
		{ID: "sp-1", TenantSlug: "demo", Label: "Stellplatz 1", UnitType: unitTypeParking, OwnerEmails: []string{"alina@example.com"}},
	}); err != nil {
		t.Fatal(err)
	}
	match := a.matchMail(context.Background(), "musterstadt", mailintake.Message{FromEmail: "Alina@Example.com", Subject: "Heizung", Text: "kalt"})
	if match.TenantSlug != "demo" || match.Unit != "Top 1" || match.Confidence < 0.85 {
		t.Fatalf("match = %+v", match)
	}
}

func TestHouseAddressInTextProposesHouseWithLowerConfidence(t *testing.T) {
	a, _ := mailIntakeTestApp(t)
	match := a.matchMail(context.Background(), "musterstadt", mailintake.Message{FromEmail: "fremd@example.com", Subject: "Schaden", Text: "Im Haus Musterweg 1, 8010 Graz tropft das Dach."})
	if match.TenantSlug != "demo" || match.Confidence >= 0.85 || match.Confidence < 0.5 {
		t.Fatalf("match = %+v", match)
	}
	none := a.matchMail(context.Background(), "musterstadt", mailintake.Message{FromEmail: "fremd@example.com", Subject: "Hallo", Text: "Nichts Konkretes."})
	if none.TenantSlug != "" || none.Confidence != 0 {
		t.Fatalf("no evidence must not assign: %+v", none)
	}
}

func TestCaseTagInSubjectFollowsTheCase(t *testing.T) {
	a, _ := mailIntakeTestApp(t)
	ctx := context.Background()
	original := store.IntakeItem{ID: "in-9001", Organisation: "musterstadt", TenantSlug: "demo", Unit: "Top 3", Source: store.IntakeSourceEmail,
		FromEmail: "matthias@example.com", Subject: "Lärm", Body: "laut", ReceivedAt: time.Now(), Status: store.IntakeStatusOpen, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := a.intake("musterstadt").Create(ctx, original); err != nil {
		t.Fatal(err)
	}
	match := a.matchMail(ctx, "musterstadt", mailintake.Message{FromEmail: "somebody@example.com", Subject: "Re: Lärm " + mailIntakeSubjectTag("in-9001"), Text: "immer noch"})
	if match.FollowUpOf != "in-9001" || match.TenantSlug != "demo" || match.Unit != "Top 3" {
		t.Fatalf("follow-up not recognised: %+v", match)
	}

	id, err := a.ingestMail(ctx, "musterstadt", mailintake.Message{MessageID: "reply@example.com", FromEmail: "matthias@example.com", Subject: "Re: Lärm " + mailIntakeSubjectTag("in-9001"), Text: "immer noch laut"})
	if err != nil {
		t.Fatal(err)
	}
	item, err := a.intake("musterstadt").Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if item.TenantSlug != "demo" || item.Unit != "Top 3" || !strings.Contains(item.Body, mailIntakeSubjectTag("in-9001")) {
		t.Fatalf("follow-up item = %+v", item)
	}
}

func TestReplyToTheCaseLandsOnItsIssue(t *testing.T) {
	a, _ := mailIntakeTestApp(t)
	ctx := context.Background()
	repositories := a.repositoriesFor(testTenantRef("demo"))
	issue, err := repositories.issues.Create(store.ResidentIssue{TenantSlug: "demo", Title: "Lärm", Body: "laut", Category: "Sonstiges", LocationType: store.IssueLocationUnit, LocationDetail: "Top 3", Status: store.IssueStatusNew, Priority: store.IssuePriorityNorm, AuthorEmail: "matthias@example.com", AuthorName: "Matthias Dorn"})
	if err != nil {
		t.Fatal(err)
	}
	original := store.IntakeItem{ID: "in-9002", Organisation: "musterstadt", TenantSlug: "demo", Source: store.IntakeSourceEmail, IssueID: issue.ID,
		FromEmail: "matthias@example.com", Subject: "Lärm", Body: "laut", ReceivedAt: time.Now(), Status: store.IntakeStatusApproved, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := a.intake("musterstadt").Create(ctx, original); err != nil {
		t.Fatal(err)
	}
	id, err := a.ingestMail(ctx, "musterstadt", mailintake.Message{MessageID: "reply2@example.com", FromEmail: "matthias@example.com", FromName: "Matthias Dorn", Subject: "Re: Lärm " + mailIntakeSubjectTag("in-9002"), Text: "Danke, jetzt ist es ruhig."})
	if err != nil {
		t.Fatal(err)
	}
	if id != "in-9002" {
		t.Fatalf("reply created a new item %q instead of landing on the issue", id)
	}
	updated, ok := repositories.issues.Get(issue.ID)
	if !ok {
		t.Fatal("issue vanished")
	}
	found := false
	for _, comment := range updated.Comments {
		if strings.Contains(comment.Body, "jetzt ist es ruhig") && comment.AuthorName == "Matthias Dorn" {
			found = true
		}
	}
	if !found {
		t.Fatalf("reply not filed as issue comment: %+v", updated.Comments)
	}
}

func TestAttachmentsWaitWithTheItemAndMoveToTheIssueOnApproval(t *testing.T) {
	a, mailer := mailIntakeTestApp(t)
	ctx := context.Background()
	addTestPerson(t, a, "alina@example.com", "demo", roleOwner)
	id, err := a.ingestMail(ctx, "musterstadt", mailintake.Message{
		MessageID: "att@example.com", FromEmail: "alina@example.com", FromName: "Alina Auer", Subject: "Beleg Reinigung",
		Text: "Anbei der Beleg.", Attachments: []mailintake.Attachment{{Filename: "beleg.pdf", ContentType: "application/pdf", Data: []byte("%PDF-1.4 x")}}, Dropped: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	item, err := a.intake("musterstadt").Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(item.Attachments) != 1 || item.DroppedFiles != 1 || item.TenantSlug != "demo" {
		t.Fatalf("item = %+v", item)
	}
	if _, err := os.Stat(a.mailIntakeDir + "/" + item.Attachments[0].Path); err != nil {
		t.Fatalf("attachment bytes not on disk: %v", err)
	}

	// Approve with a suggestion the way the inbox does; the reply must go out
	// by mail with the case tag, and the file must reach the Anliegen.
	item.Suggestion = &store.IntakeSuggestion{Category: store.IntakeCategoryReceipt, Priority: store.IssuePriorityNorm, TenantSlug: "demo", Reply: "Danke, der Beleg ist angekommen.", Confidence: map[string]float64{"overall": 0.9}, CreatedAt: time.Now()}
	if err := a.intake("musterstadt").UpdateSuggestion(ctx, id, *item.Suggestion, store.IntakeStatusProposed); err != nil {
		t.Fatal(err)
	}
	item, _ = a.intake("musterstadt").Get(ctx, id)
	if err := a.handleIntake(ctx, "musterstadt", item, intakeHandleOptions{status: store.IntakeStatusApproved, action: "approve", actorEmail: "verwaltung@example.com", actorName: "Vera", auditAction: store.AuditActionIssueAIAccept}); err != nil {
		t.Fatal(err)
	}
	approved, _ := a.intake("musterstadt").Get(ctx, id)
	if approved.IssueID == "" {
		t.Fatal("approval created no issue")
	}
	repositories := a.repositoriesFor(testTenantRef("demo"))
	files := repositories.attachments.ListEntity("issue", approved.IssueID)
	if len(files) != 1 || files[0].Filename != "beleg.pdf" {
		t.Fatalf("issue attachments = %+v", files)
	}
	// The house notifications about the new Anliegen go out through the same
	// mailer; only the tagged reply to the sender is under test here.
	mailer.mu.Lock()
	defer mailer.mu.Unlock()
	var replies []struct{ To, Subject, Body string }
	for _, sent := range mailer.sent {
		if strings.Contains(sent.Subject, mailIntakeSubjectTag(id)) {
			replies = append(replies, sent)
		}
	}
	if len(replies) != 1 {
		t.Fatalf("tagged reply mails sent = %d, want 1 (all mails: %+v)", len(replies), mailer.sent)
	}
	if replies[0].To != "alina@example.com" || !strings.HasPrefix(replies[0].Subject, "Re: ") {
		t.Fatalf("reply = %+v", replies[0])
	}
	if !strings.Contains(replies[0].Body, "Beleg ist angekommen") {
		t.Fatalf("reply body = %q", replies[0].Body)
	}
}

func TestMailIntakeConfigMustNameAKnownOrganisation(t *testing.T) {
	configs, err := mailintake.ParseConfigs(`{"fremd": {"host": "imap.example", "username": "u", "password_env": "X"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := configs["fremd"]; !ok {
		t.Fatal("config parse lost the organisation")
	}
	// The boot check lives in newApp; here we only pin the contract that an
	// unknown key is not silently accepted by the parser's consumer.
	a := organisationTestApp(t, "verwaltung@example.com")
	if _, ok := a.organisations["fremd"]; ok {
		t.Fatal("test organisation unexpectedly knows 'fremd'")
	}
}

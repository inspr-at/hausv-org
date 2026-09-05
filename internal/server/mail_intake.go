package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/inspr-at/hausv-org/internal/mailintake"
	"github.com/inspr-at/hausv-org/internal/store"
)

// StartMailIntake is the boot hook main calls beside the other workers.
func (a *app) StartMailIntake() func() { return a.startMailIntake() }

// Mail intake: a Verwaltung's mailbox becomes Posteingang items.
//
// The mailbox is read, never trusted. A sender address proposes a house and a
// person, it does not prove them; the proposal carries a confidence and a human
// (or the trust level the organisation chose) decides. Nothing in a mail is
// executed, attachments are bytes within the portal's own upload limits.

const (
	mailIntakeBatch        = 25
	mailIntakeMaxFiles     = 10
	mailIntakeMaxTextBytes = 64 << 10
	// mailIntakeCaseTag is how an Anliegen number travels in a subject line:
	// outgoing replies carry it, incoming replies are recognised by it.
	mailIntakeCaseTag = "HV-"
)

var mailIntakeCaseRef = regexp.MustCompile(`\[` + mailIntakeCaseTag + `([A-Za-z0-9_-]{4,64})\]`)

// mailIntakeStatus is what the settings page shows per organisation.
type mailIntakeStatus struct {
	Configured bool
	Mailbox    string
	Interval   time.Duration
	LastRun    time.Time
	LastError  string
	LastCount  int
	Total      int
}

type mailIntakeState struct {
	mu     sync.Mutex
	status map[string]mailIntakeStatus
}

func (s *mailIntakeState) set(orgKey string, update func(*mailIntakeStatus)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status == nil {
		s.status = map[string]mailIntakeStatus{}
	}
	status := s.status[orgKey]
	update(&status)
	s.status[orgKey] = status
}

func (s *mailIntakeState) get(orgKey string) (mailIntakeStatus, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	status, ok := s.status[orgKey]
	return status, ok
}

// mailIntakeSubjectTag is the marker an outgoing reply carries so an answer to
// it finds its way back to the same Anliegen.
func mailIntakeSubjectTag(intakeID string) string {
	return "[" + mailIntakeCaseTag + strings.TrimSpace(intakeID) + "]"
}

// mailIntakeCaseReference extracts the intake id from a subject, if present.
func mailIntakeCaseReference(subject string) string {
	match := mailIntakeCaseRef.FindStringSubmatch(subject)
	if match == nil {
		return ""
	}
	return match[1]
}

func mailIntakeLimits() mailintake.Limits {
	return mailintake.Limits{MaxAttachmentBytes: maxAttachmentBytes, MaxAttachments: mailIntakeMaxFiles, MaxTextBytes: mailIntakeMaxTextBytes}
}

// startMailIntake polls every configured mailbox on its own interval. It
// returns the stop function; a deployment without INTAKE_MAIL_JSON gets a no-op.
func (a *app) startMailIntake() func() {
	if a == nil || len(a.mailIntakeConfigs) == 0 {
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	for _, orgKey := range mailintake.Keys(a.mailIntakeConfigs) {
		config := a.mailIntakeConfigs[orgKey]
		a.mailIntake.set(orgKey, func(s *mailIntakeStatus) {
			s.Configured = true
			s.Mailbox = config.Username + " @ " + config.Host
			s.Interval = config.Interval
		})
		go func(orgKey string, config mailintake.Config) {
			a.pollMailIntake(ctx, orgKey, config)
			ticker := time.NewTicker(config.Interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					a.pollMailIntake(ctx, orgKey, config)
				}
			}
		}(orgKey, config)
	}
	logInfo("mail intake enabled", "organisations", len(a.mailIntakeConfigs))
	return cancel
}

// pollMailIntake reads one mailbox once. Every message is stored before it is
// flagged read, and the seen ledger is checked before anything is created, so
// neither a crash nor a repeated fetch files a mail twice.
func (a *app) pollMailIntake(ctx context.Context, orgKey string, config mailintake.Config) {
	started := time.Now()
	count, err := a.pollMailIntakeOnce(ctx, orgKey, config)
	a.mailIntake.set(orgKey, func(s *mailIntakeStatus) {
		s.LastRun = started
		s.LastCount = count
		s.Total += count
		s.LastError = ""
		if err != nil {
			s.LastError = err.Error()
		}
	})
	if err != nil {
		logError("mail intake poll failed", err, "organisation", orgKey)
	}
}

func (a *app) pollMailIntakeOnce(ctx context.Context, orgKey string, config mailintake.Config) (int, error) {
	if a.intakeMailSeen == nil || a.intake == nil {
		return 0, errors.New("mail intake stores unavailable")
	}
	password, err := config.Password(os.Getenv)
	if err != nil {
		return 0, err
	}
	fetcher := mailintake.Fetcher{Config: config, Password: password, Limits: mailIntakeLimits()}
	messages, err := fetcher.Unread(ctx, mailIntakeBatch)
	if err != nil {
		return 0, err
	}
	seenRepo := a.intakeMailSeen(orgKey)
	var done []uint32
	filed := 0
	for _, message := range messages {
		messageID := message.MessageID
		if messageID == "" {
			// Without a Message-ID the ledger cannot recognise the mail again;
			// the UID is stable within the mailbox and serves as the key.
			messageID = fmt.Sprintf("uid-%d@%s", message.UID, config.Host)
		}
		seen, err := seenRepo.Seen(ctx, messageID)
		if err != nil {
			return filed, err
		}
		if seen {
			done = append(done, message.UID)
			continue
		}
		intakeID, err := a.ingestMail(ctx, orgKey, message)
		if err != nil {
			logError("mail intake could not file a message", err, "organisation", orgKey, "uid", message.UID)
			continue
		}
		if err := seenRepo.Record(ctx, messageID, intakeID); err != nil {
			return filed, err
		}
		done = append(done, message.UID)
		filed++
	}
	if err := fetcher.MarkSeen(ctx, done); err != nil {
		return filed, err
	}
	return filed, nil
}

// mailMatch is what the sender and the text say about where a mail belongs.
type mailMatch struct {
	TenantSlug string
	Unit       string
	Confidence float64
	Reason     string
	// FollowUpOf is the intake item an answer refers to, by subject tag.
	FollowUpOf string
}

// matchMail proposes a house, a unit and the case a mail continues. Order of
// evidence: the case tag in the subject beats the sender's membership, which
// beats a house address quoted in the text.
func (a *app) matchMail(ctx context.Context, orgKey string, message mailintake.Message) mailMatch {
	houses := a.configuredOrganisationHouses(orgKey)
	if a.organisationRepo != nil {
		if stored, ok, err := a.organisationRepo(orgKey).Get(ctx); err == nil && ok && len(stored.Houses) > 0 {
			houses = stored.Houses
		}
	}
	inOrganisation := make(map[string]bool, len(houses))
	for _, slug := range houses {
		inOrganisation[slug] = true
	}

	if ref := mailIntakeCaseReference(message.Subject); ref != "" && a.intake != nil {
		if previous, err := a.intake(orgKey).Get(ctx, ref); err == nil {
			return mailMatch{TenantSlug: previous.TenantSlug, Unit: previous.Unit, Confidence: 0.98, Reason: "Antwort auf " + mailIntakeSubjectTag(ref), FollowUpOf: previous.ID}
		}
	}

	sender := normalizeEmail(message.FromEmail)
	if sender != "" {
		if profile, ok := a.directoryProfile(sender); ok {
			candidates := make([]string, 0, 2)
			for _, slug := range profile.Tenants {
				slug = normalizeSlug(slug)
				if inOrganisation[slug] {
					candidates = append(candidates, slug)
				}
			}
			sort.Strings(candidates)
			if len(candidates) == 1 {
				return mailMatch{TenantSlug: candidates[0], Unit: a.unitLabelForEmail(candidates[0], sender), Confidence: 0.9, Reason: "Absender ist im Haus bekannt"}
			}
			if len(candidates) > 1 {
				if slug := a.houseMentionedIn(message.Subject+"\n"+message.Text, candidates); slug != "" {
					return mailMatch{TenantSlug: slug, Unit: a.unitLabelForEmail(slug, sender), Confidence: 0.85, Reason: "Absender in mehreren Häusern, Haus im Text genannt"}
				}
				return mailMatch{Confidence: 0.4, Reason: "Absender in mehreren Häusern, keines im Text genannt"}
			}
		}
	}

	if slug := a.houseMentionedIn(message.Subject+"\n"+message.Text, houses); slug != "" {
		return mailMatch{TenantSlug: slug, Confidence: 0.7, Reason: "Hausadresse im Text genannt"}
	}
	return mailMatch{Confidence: 0, Reason: "keine Zuordnung möglich"}
}

// houseMentionedIn looks for a house's name or address in free text.
func (a *app) houseMentionedIn(text string, slugs []string) string {
	haystack := strings.ToLower(text)
	for _, slug := range slugs {
		tenant, ok := a.tenantBySlug(slug)
		if !ok {
			continue
		}
		for _, needle := range []string{houseDisplayName(tenant), tenant.Address} {
			needle = strings.ToLower(strings.TrimSpace(needle))
			if len(needle) >= 6 && strings.Contains(haystack, needle) {
				return slug
			}
		}
	}
	return ""
}

// unitLabelForEmail returns the unit a person owns or rents in a house, when
// there is exactly one to name.
func (a *app) unitLabelForEmail(slug string, email string) string {
	identity, ok := a.tenantIdentity(slug)
	if !ok || a.unitStore == nil {
		return ""
	}
	units, ok := store.BindUnitRepository(a.unitStore, identity.Ref())
	if !ok {
		return ""
	}
	// A person usually holds a flat and a parking space; the flat is where a
	// mail about "my apartment" belongs. Name it only when it is unambiguous.
	residential := make([]string, 0, 2)
	for _, membership := range units.UnitsForEmail(email) {
		if normalizeUnitType(membership.Unit.UnitType) == unitTypeParking {
			continue
		}
		residential = append(residential, membership.Unit.Label)
	}
	if len(residential) != 1 {
		return ""
	}
	return residential[0]
}

// ingestMail files one mail: as a comment on the Anliegen it answers, or as a
// new intake item that then goes through the same triage as a phone note.
func (a *app) ingestMail(ctx context.Context, orgKey string, message mailintake.Message) (string, error) {
	match := a.matchMail(ctx, orgKey, message)
	now := time.Now().UTC()
	received := message.Date
	if received.IsZero() {
		received = now
	}

	if match.FollowUpOf != "" {
		previous, err := a.intake(orgKey).Get(ctx, match.FollowUpOf)
		if err == nil && previous.IssueID != "" && previous.TenantSlug != "" {
			if filed, err := a.fileMailAsIssueComment(previous, message); err == nil && filed {
				return previous.ID, nil
			}
		}
	}

	id, err := randomToken(12)
	if err != nil {
		return "", err
	}
	item := store.IntakeItem{
		ID: id, Organisation: orgKey, TenantSlug: match.TenantSlug, Unit: match.Unit,
		Source: store.IntakeSourceEmail, ExternalRef: message.MessageID,
		FromName: strings.TrimSpace(message.FromName), FromEmail: normalizeEmail(message.FromEmail),
		Subject: strings.TrimSpace(message.Subject), Body: message.Text,
		ReceivedAt: received, Status: store.IntakeStatusOpen, CreatedAt: now, UpdatedAt: now,
		DroppedFiles: message.Dropped,
	}
	if item.Body == "" {
		item.Body = "(kein Text)"
	}
	if match.FollowUpOf != "" {
		item.Body = "Antwort auf " + mailIntakeSubjectTag(match.FollowUpOf) + "\n\n" + item.Body
	}
	attachments, err := a.storeMailAttachments(orgKey, id, message.Attachments)
	if err != nil {
		return "", err
	}
	item.Attachments = attachments
	if err := a.intake(orgKey).Create(ctx, item); err != nil {
		return "", err
	}
	a.recordIntakeAudit(match.TenantSlug, "mail-intake", store.AuditActionIntakeMail, item, store.IntakeSuggestion{}, "E-Mail eingegangen: "+match.Reason)
	if _, err := a.processIntake(ctx, orgKey, item, "System (E-Mail)"); err != nil {
		logError("mail intake triage unavailable", err, "intake_id", id)
	}
	return id, nil
}

// fileMailAsIssueComment attaches an answer to the Anliegen it belongs to.
func (a *app) fileMailAsIssueComment(previous store.IntakeItem, message mailintake.Message) (bool, error) {
	identity, ok := a.tenantIdentity(previous.TenantSlug)
	if !ok {
		return false, nil
	}
	repositories := a.repositoriesFor(identity.Ref())
	if repositories.issues == nil {
		return false, nil
	}
	author := strings.TrimSpace(message.FromName)
	if author == "" {
		author = normalizeEmail(message.FromEmail)
	}
	body := message.Text
	if body == "" {
		body = "(kein Text)"
	}
	if message.Dropped > 0 || len(message.Attachments) > 0 {
		body += fmt.Sprintf("\n\n(%d Anhänge per E-Mail, %d nicht übernommen)", len(message.Attachments), message.Dropped)
	}
	_, exists, err := repositories.issues.AddComment(previous.IssueID, store.IssueComment{
		ID: "mail-" + strings.ReplaceAll(strings.TrimSpace(message.MessageID), "@", "-"), AuthorEmail: normalizeEmail(message.FromEmail), AuthorName: author,
		Body: body, Kind: store.IssueCommentKindInformation, CreatedAt: time.Now(),
	})
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	a.recordAudit(store.AuditEvent{TenantSlug: previous.TenantSlug, ActorEmail: normalizeEmail(message.FromEmail), ActorRole: roleResident,
		Action: store.AuditActionIntakeMail, TargetType: "issue", TargetID: previous.IssueID, Summary: "E-Mail-Antwort zum Anliegen abgelegt"})
	return true, nil
}

// storeMailAttachments keeps the bytes beside the intake item. The portal's
// attachment store takes over when the case becomes an Anliegen.
func (a *app) storeMailAttachments(orgKey string, intakeID string, parts []mailintake.Attachment) ([]store.IntakeAttachment, error) {
	if len(parts) == 0 {
		return nil, nil
	}
	if a.mailIntakeDir == "" {
		return nil, errors.New("mail intake directory not configured")
	}
	dir := filepath.Join(a.mailIntakeDir, normalizeSlug(orgKey), intakeID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	out := make([]store.IntakeAttachment, 0, len(parts))
	for index, part := range parts {
		name := fmt.Sprintf("%02d-%s", index+1, safeAttachmentName(part.Filename))
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, part.Data, 0o640); err != nil {
			return nil, err
		}
		out = append(out, store.IntakeAttachment{
			Filename: part.Filename, ContentType: part.ContentType, Size: int64(len(part.Data)),
			Path: filepath.ToSlash(filepath.Join(normalizeSlug(orgKey), intakeID, name)),
		})
	}
	return out, nil
}

var unsafeAttachmentChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func safeAttachmentName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = unsafeAttachmentChars.ReplaceAllString(name, "_")
	name = strings.Trim(name, "._")
	if name == "" {
		return "anhang"
	}
	if len(name) > 80 {
		name = name[len(name)-80:]
	}
	return name
}

// handMailAttachmentsToIssue copies the waiting files onto the Anliegen through
// the regular attachment store, with its limits and its sniffing.
func (a *app) handMailAttachmentsToIssue(repositories requestRepositories, item store.IntakeItem, issueID string, actorEmail string) {
	if len(item.Attachments) == 0 || repositories.attachments == nil || a.mailIntakeDir == "" {
		return
	}
	uploads := make([]store.UploadedFile, 0, len(item.Attachments))
	for _, attachment := range item.Attachments {
		path := filepath.Join(a.mailIntakeDir, filepath.FromSlash(attachment.Path))
		uploads = append(uploads, store.UploadedFile{
			Filename: attachment.Filename, Size: attachment.Size, DeclaredType: attachment.ContentType,
			Open: func() (io.ReadSeekCloser, error) { return os.Open(path) },
		})
	}
	if _, err := repositories.attachments.CreateUploaded("issue", issueID, actorEmail, uploads, time.Now()); err != nil {
		logError("mail attachments could not be handed to the issue", err, "intake_id", item.ID, "issue_id", issueID)
	}
}

// sendMailIntakeReply answers an e-mail case through the configured mailer,
// tagged so that the reply to the reply finds its case again.
func (a *app) sendMailIntakeReply(item store.IntakeItem, reply string) error {
	if a == nil || a.mailer == nil || !a.mailer.Configured() {
		return errors.New("mailer not configured")
	}
	to := normalizeEmail(item.FromEmail)
	if item.Source != store.IntakeSourceEmail || to == "" {
		return errors.New("not an e-mail case")
	}
	subject := strings.TrimSpace(item.Subject)
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}
	if mailIntakeCaseReference(subject) == "" {
		subject += " " + mailIntakeSubjectTag(item.ID)
	}
	return a.mailer.SendNotification(to, subject, strings.TrimSpace(reply))
}

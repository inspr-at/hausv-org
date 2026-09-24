package demo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/mailintake"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/textutil"
)

type SeedOptions struct {
	// DocumentDir is the same original-file directory used by the portal.
	DocumentDir string
	Reset       bool
	// DiscardAnnualStatements additionally drops the tenant's stored annual
	// statement runs, their archive documents and the delivery log. Reset alone
	// is an input operation and keeps them (they are immutable records); the
	// demo's "Demodaten initialisieren" wants the clean demo day (HAUSV-663).
	DiscardAnnualStatements bool
	Stats                   bool
	Out                     io.Writer
	// Units receives the fixture's tenant-scoped unit inventory when supplied.
	// It is optional so database-only consumers keep their existing behavior.
	Units store.UnitSink
	// Anchor shifts every seed date so that the fixture's demo day (2026-09-09)
	// lands on Anchor's calendar day; zero keeps the committed dates.
	Anchor time.Time
}

// seedDemoDay is the calendar day the committed fixture is written for.
var seedDemoDay = time.Date(2026, 9, 9, 0, 0, 0, 0, time.FixedZone("CEST", 2*60*60))

type SeedResult struct {
	Statuses   map[store.IntakeStatus]int
	Categories map[string]int
}

type seedOrg struct {
	Key            string            `json:"key"`
	Name           string            `json:"name"`
	ContactName    string            `json:"contact_name,omitempty"`
	ContactAddress string            `json:"contact_address,omitempty"`
	ContactEmail   string            `json:"contact_email,omitempty"`
	ContactPhone   string            `json:"contact_phone,omitempty"`
	TrustLevels    map[string]string `json:"trust_levels"`
	AutoThreshold  float64           `json:"auto_threshold"`
	AutoEnabled    bool              `json:"auto_enabled"`
	Assignees      []seedAssignee    `json:"assignees"`
	Members        []seedMember      `json:"members"`
}

type seedMember struct {
	Email   string            `json:"email"`
	Role    string            `json:"role"`
	Granted map[string]string `json:"granted"`
}

type seedAssignee struct {
	Key   string `json:"key"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type seedHouse struct {
	Slug         string                 `json:"slug"`
	Name         string                 `json:"name"`
	Address      string                 `json:"address"`
	Organisation string                 `json:"organisation"`
	Units        []seedUnit             `json:"units"`
	Contacts     []store.ManagedContact `json:"contacts"`
	Ballots      []store.Ballot         `json:"ballots"`
}

type seedUnit struct {
	StatementBasis *seedStatementBasis      `json:"-"`
	PartyContacts  []store.UnitPartyContact `json:"-"`
	Label          string                   `json:"label"`
	Floor          string                   `json:"floor"`
	UnitType       string                   `json:"unit_type"`
	OwnerEmail     string                   `json:"owner_email"`
	TenantEmail    string                   `json:"tenant_email"`
}

type seedIntake struct {
	ID           string             `json:"id"`
	Source       store.IntakeSource `json:"source"`
	ReceivedAt   time.Time          `json:"received_at"`
	House        string             `json:"house"`
	Unit         string             `json:"unit"`
	FromName     string             `json:"from_name"`
	FromEmail    string             `json:"from_email"`
	FromPhone    string             `json:"from_phone"`
	Subject      string             `json:"subject"`
	Body         string             `json:"body"`
	Truth        *store.IntakeTruth `json:"truth"`
	Precomputed  *seedSuggestion    `json:"precomputed"`
	StatusHint   string             `json:"status_hint"`
	HandledReply string             `json:"handled_reply,omitempty"`
}

type seedSuggestion struct {
	Category    string             `json:"category"`
	Priority    string             `json:"priority"`
	House       string             `json:"house"`
	Unit        string             `json:"unit"`
	Assignee    string             `json:"assignee"`
	TemplateKey string             `json:"template_key"`
	Reply       string             `json:"reply"`
	Actions     []string           `json:"actions"`
	Confidence  map[string]float64 `json:"confidence"`
}

type seedEvent struct {
	ID          string    `json:"id"`
	House       string    `json:"house"`
	Title       string    `json:"title"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	Location    string    `json:"location"`
	Description string    `json:"description"`
}

type seedAnnouncement struct {
	ID          string    `json:"id"`
	House       string    `json:"house"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	Category    string    `json:"category"`
	PublishedAt time.Time `json:"published_at"`
}

func Load(ctx context.Context, database *sql.DB, dir string, options SeedOptions) (SeedResult, error) {
	if database == nil {
		return SeedResult{}, fmt.Errorf("demo seed database is required")
	}
	var org seedOrg
	if err := readJSON(filepath.Join(dir, "org.json"), &org); err != nil {
		return SeedResult{}, err
	}
	org.Key = textutil.Slug(org.Key)
	if org.Key == "" {
		return SeedResult{}, fmt.Errorf("demo seed organisation key is required")
	}
	var houses []seedHouse
	var intake []seedIntake
	var templates []store.Textbaustein
	var events []seedEvent
	var announcements []seedAnnouncement
	for name, target := range map[string]any{
		"houses.json": &houses, "intake.json": &intake, "textbausteine.json": &templates,
		"events.json": &events, "announcements.json": &announcements,
	} {
		if err := readJSON(filepath.Join(dir, name), target); err != nil {
			return SeedResult{}, err
		}
	}

	documents, err := loadDocumentFixture(dir, options.DocumentDir, houses)
	if err != nil {
		return SeedResult{}, err
	}
	statement, err := loadStatementFixture(dir, houses, options.DocumentDir)
	if err != nil {
		return SeedResult{}, err
	}
	configured := make([]store.TenantIdentity, 0, len(houses))
	for _, house := range houses {
		configured = append(configured, store.TenantIdentity{Slug: house.Slug, Name: house.Name})
	}
	identities, err := store.EnsureTenantIdentities(ctx, database, configured)
	if err != nil {
		return SeedResult{}, err
	}
	if !options.Anchor.IsZero() {
		shiftSeedDates(options.Anchor, intake, events, announcements)
		shiftHouseDates(options.Anchor, houses)
	}
	if options.Reset {
		mail, err := loadMailboxFixture(dir)
		if err != nil {
			return SeedResult{}, err
		}
		if err := reset(ctx, database, org.Key, houses, intake, events, announcements, mail, options.DocumentDir, options.DiscardAnnualStatements); err != nil {
			return SeedResult{}, err
		}
	}

	settings := store.BindOrgSettingsRepository(database, org.Key)
	if err := settings.Save(ctx, store.OrgSettings{
		Organisation: org.Key, Name: org.Name, TrustLevels: org.TrustLevels,
		ContactAddress: org.ContactAddress,
		AutoThreshold:  org.AutoThreshold, AutoEnabled: org.AutoEnabled, Counters: seedCounters(intake),
	}); err != nil {
		return SeedResult{}, err
	}
	// The organisation's own contact data (shown in the sidebar and in
	// resident replies) come from the fixture too; the houses stay as they are.
	if org.ContactName != "" || org.ContactEmail != "" || org.ContactPhone != "" {
		orgRepo := store.BindOrganisationRepository(database, org.Key)
		current, found, err := orgRepo.Get(ctx)
		if err != nil {
			return SeedResult{}, err
		}
		if !found {
			current = store.Organisation{Key: org.Key, Name: org.Name}
		}
		current.ContactName, current.ContactEmail, current.ContactPhone = org.ContactName, org.ContactEmail, org.ContactPhone
		if err := orgRepo.Save(ctx, current); err != nil {
			return SeedResult{}, err
		}
	}
	memberRepo := store.BindOrganisationMemberRepository(database, org.Key)
	for _, member := range org.Members {
		// Preserve the house roles already supplied by persons.json in the
		// grant record, so removing a demo employee restores those roles.
		if err := memberRepo.Save(ctx, store.OrganisationMember{Email: member.Email, Role: member.Role, Granted: member.Granted}); err != nil {
			return SeedResult{}, fmt.Errorf("seed member %s: %w", member.Email, err)
		}
	}

	templateRepo := store.BindTextbausteinRepository(database, org.Key)
	for _, item := range templates {
		item.Active = true
		if err := templateRepo.Upsert(ctx, item); err != nil {
			return SeedResult{}, err
		}
	}

	if err := upsertHouseFixtures(ctx, database, houses, identities, intake, events, announcements, org); err != nil {
		return SeedResult{}, err
	}
	if err := seedLeases(ctx, database, identities, dir); err != nil {
		return SeedResult{}, err
	}
	if err := seedDocuments(ctx, database, documents, identities, options.DocumentDir); err != nil {
		return SeedResult{}, err
	}
	if statement != nil {
		if err := seedAnnualStatement(ctx, database, statement, identities[statement.House], options.DocumentDir, options.Reset); err != nil {
			return SeedResult{}, err
		}
	}
	if options.Units != nil {
		for _, house := range houses {
			identity, ok := identities[textutil.Slug(house.Slug)]
			if !ok {
				return SeedResult{}, fmt.Errorf("missing tenant identity for %s", house.Slug)
			}
			if err := options.Units.ReplaceTenantUnits(ctx, identity.Slug, fixtureUnits(identity.Slug, house.Units)); err != nil {
				return SeedResult{}, fmt.Errorf("seed units for %s: %w", identity.Slug, err)
			}
		}
	}
	intakeRepo := store.BindIntakeRepository(database, org.Key)
	result := SeedResult{Statuses: map[store.IntakeStatus]int{}, Categories: map[string]int{}}
	for index, raw := range intake {
		item := intakeItem(raw, org.Key, index)
		if err := intakeRepo.Create(ctx, item); err != nil {
			return SeedResult{}, fmt.Errorf("seed intake %s: %w", raw.ID, err)
		}
		result.Statuses[item.Status]++
		category := ""
		if item.Suggestion != nil {
			category = item.Suggestion.Category
		} else if item.Truth != nil {
			category = item.Truth.Category
		}
		result.Categories[category]++
	}
	if options.Stats && options.Out != nil {
		printStats(options.Out, result)
	}
	return result, nil
}

func intakeItem(raw seedIntake, orgKey string, index int) store.IntakeItem {
	status := store.IntakeStatusOpen
	if raw.Precomputed != nil {
		status = store.IntakeStatusProposed
	}
	var handling *store.IntakeHandling
	issueID := ""
	switch raw.StatusHint {
	case "auto_done":
		status = store.IntakeStatusAuto
		handling = &store.IntakeHandling{Action: "auto", ByName: "System", At: raw.ReceivedAt}
		issueID = "issue-" + raw.ID
	case "approved", "edited":
		status = store.IntakeStatusApproved
		handling = &store.IntakeHandling{Action: raw.StatusHint, ByName: "Demo", At: raw.ReceivedAt}
		if raw.StatusHint == "edited" {
			status = store.IntakeStatusEdited
		}
		issueID = "issue-" + raw.ID
	case "manual":
		status = store.IntakeStatusRejected
		handling = &store.IntakeHandling{Action: "rejected", ByName: "Demo", At: raw.ReceivedAt}
		issueID = "issue-" + raw.ID
	}
	var suggestion *store.IntakeSuggestion
	if raw.Precomputed != nil {
		suggestion = &store.IntakeSuggestion{
			Source: "seed", Category: raw.Precomputed.Category, Priority: raw.Precomputed.Priority,
			TenantSlug: raw.Precomputed.House, Unit: raw.Precomputed.Unit, Assignee: raw.Precomputed.Assignee,
			TemplateKey: raw.Precomputed.TemplateKey, Reply: raw.Precomputed.Reply,
			Actions: raw.Precomputed.Actions, Confidence: raw.Precomputed.Confidence, CreatedAt: raw.ReceivedAt,
		}
	}
	if suggestion != nil && raw.StatusHint == "edited" {
		suggestion.Reply = raw.HandledReply
	}
	return store.IntakeItem{
		ID: raw.ID, Organisation: orgKey, TenantSlug: raw.House, Unit: raw.Unit, Source: raw.Source,
		FromName: raw.FromName, FromEmail: raw.FromEmail, FromPhone: raw.FromPhone,
		Subject: raw.Subject, Body: raw.Body, ReceivedAt: raw.ReceivedAt, DueAt: dueFor(raw.ReceivedAt, seedPriority(raw)), Status: status,
		Suggestion: suggestion, Handling: handling, IssueID: issueID, Truth: raw.Truth,
		CreatedAt: raw.ReceivedAt, UpdatedAt: raw.ReceivedAt.Add(time.Duration(index) * time.Nanosecond),
	}
}

func upsertHouseFixtures(ctx context.Context, database *sql.DB, houses []seedHouse, identities map[string]store.TenantIdentity, intake []seedIntake, events []seedEvent, announcements []seedAnnouncement, org seedOrg) error {
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if strings.Contains(fmt.Sprintf("%T", database.Driver()), "stdlib") {
		if _, err := tx.ExecContext(ctx, `SET LOCAL hausv.cross_tenant = 'on'`); err != nil {
			return err
		}
	}
	for _, house := range houses {
		identity, ok := identities[textutil.Slug(house.Slug)]
		if !ok {
			return fmt.Errorf("missing tenant identity for %s", house.Slug)
		}
		for _, raw := range house.Contacts {
			raw.TenantSlug = identity.Slug
			item, err := store.NormalizeManagedContact(raw)
			if err != nil || item.ID == "" {
				return fmt.Errorf("invalid seed contact %s in %s: %v", raw.ID, identity.Slug, err)
			}
			if err := upsertJSON(ctx, tx, "contacts", identity, item.ID, item); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE contacts SET active=$1 WHERE tenant_slug=$2 AND id=$3`, item.Active, identity.Slug, item.ID); err != nil {
				return err
			}
		}
		for _, item := range house.Ballots {
			item.TenantSlug = identity.Slug
			item = store.NormalizeBallot(item)
			if item.ID == "" || item.Title == "" || len(item.Options) < 2 {
				return fmt.Errorf("invalid seed ballot in %s", identity.Slug)
			}
			if err := upsertJSON(ctx, tx, "ballots", identity, item.ID, item); err != nil {
				return err
			}
		}
		for _, unit := range fixtureUnits(identity.Slug, house.Units) {
			if err := upsertJSON(ctx, tx, "units", identity, unit.ID, unit); err != nil {
				return err
			}
		}
	}
	assigneeEmail := map[string]string{}
	for _, assignee := range org.Assignees {
		assigneeEmail[assignee.Key] = assignee.Email
	}
	for index, raw := range intake {
		if raw.StatusHint != "auto_done" && raw.StatusHint != "approved" && raw.StatusHint != "edited" && raw.StatusHint != "manual" {
			continue
		}
		identity, ok := identities[textutil.Slug(raw.House)]
		if !ok {
			continue
		}
		category, priority, assignee := store.IntakeCategoryOther, store.IssuePriorityNorm, ""
		if raw.Truth != nil {
			category, priority, assignee = raw.Truth.Category, raw.Truth.Priority, raw.Truth.Assignee
		} else if raw.Precomputed != nil {
			category, priority, assignee = raw.Precomputed.Category, raw.Precomputed.Priority, raw.Precomputed.Assignee
		}
		status := store.IssueStatusNew
		if raw.StatusHint == "auto_done" || ((raw.StatusHint == "approved" || raw.StatusHint == "edited") && index%4 != 0) {
			status = store.IssueStatusDone
		} else if raw.StatusHint == "approved" || raw.StatusHint == "edited" {
			status = store.IssueStatusProgress
		} else if raw.StatusHint == "manual" && assigneeEmail[assignee] != "" {
			// HAUSV-669: manual work is a mix of new, accepted and in-progress items;
			// only assigned items may leave "Neu" (HAUSV-716 rule).
			switch (index / 3) % 3 {
			case 1:
				status = store.IssueStatusAccepted
			case 2:
				status = store.IssueStatusProgress
			}
		}
		email := raw.FromEmail
		if email == "" {
			email = "demo@" + org.Key + ".example"
		}
		location := store.IssueLocationCommon
		if raw.Unit != "" {
			location = store.IssueLocationUnit
		}
		issue := store.ResidentIssue{
			ID: "issue-" + raw.ID, TenantSlug: identity.Slug, Source: string(raw.Source), IntakeID: raw.ID,
			AuthorEmail: email, AuthorName: raw.FromName, Category: category, Title: raw.Subject, Body: raw.Body,
			LocationType: location, LocationDetail: raw.Unit, Status: status, Priority: priority,
			AssigneeEmail: assigneeEmail[assignee], StatusChangedAt: raw.ReceivedAt, StatusChangedBy: "system",
			CreatedAt: raw.ReceivedAt, UpdatedAt: raw.ReceivedAt, DueAt: dueFor(raw.ReceivedAt, priority),
		}
		if raw.StatusHint == "edited" && raw.HandledReply != "" {
			issue.Comments = []store.IssueComment{{ID: "intake-reply-" + raw.ID, AuthorEmail: assigneeEmail[assignee], AuthorName: "Demo", Body: raw.HandledReply, Kind: store.IssueCommentKindInformation, CreatedAt: raw.ReceivedAt}}
		}
		if issue.Title == "" {
			issue.Title = "Demo-Anliegen"
		}
		if issue.Body == "" {
			issue.Body = issue.Title
		}
		if err := upsertJSON(ctx, tx, "issues", identity, issue.ID, issue); err != nil {
			return err
		}
	}
	for _, raw := range events {
		identity, ok := identities[textutil.Slug(raw.House)]
		if !ok {
			continue
		}
		end := raw.EndsAt
		item := store.HouseEvent{ID: raw.ID, TenantSlug: identity.Slug, Title: raw.Title, Body: raw.Description, Category: "Termin", Location: raw.Location, StartsAt: raw.StartsAt, EndsAt: &end, AuthorName: "System", CreatedAt: raw.StartsAt, UpdatedAt: raw.StartsAt}
		if err := upsertJSON(ctx, tx, "events", identity, raw.ID, item); err != nil {
			return err
		}
	}
	for _, raw := range announcements {
		identity, ok := identities[textutil.Slug(raw.House)]
		if !ok {
			continue
		}
		item := store.Announcement{ID: raw.ID, TenantSlug: identity.Slug, Title: raw.Title, Body: raw.Body, Category: raw.Category, PublishedAt: raw.PublishedAt, AuthorName: "System", CreatedAt: raw.PublishedAt, UpdatedAt: raw.PublishedAt}
		if err := upsertJSON(ctx, tx, "announcements", identity, raw.ID, item); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func fixtureUnits(tenantSlug string, rawUnits []seedUnit) []store.Unit {
	units := make([]store.Unit, 0, len(rawUnits))
	for _, raw := range rawUnits {
		unit := store.Unit{
			ID:         store.NormalizeUnitID(raw.Label),
			TenantSlug: tenantSlug,
			Label:      raw.Label,
			UnitType:   store.NormalizeUnitType(raw.UnitType),
		}
		if unit.UnitType == "" {
			unit.UnitType = store.UnitTypeResidential
		}
		unit.PartyContacts = append([]store.UnitPartyContact(nil), raw.PartyContacts...)
		if basis := raw.StatementBasis; basis != nil {
			unit.MiteigentumsanteilPPM = basis.PPM
			unit.UsableAreaM2Hundredths = basis.Area
			unit.UsableAreaRecorded = true
			unit.Persons = basis.Persons
			unit.PersonsRecorded = true
		}
		if raw.OwnerEmail != "" {
			unit.OwnerEmails = []string{raw.OwnerEmail}
		}
		if raw.TenantEmail != "" {
			unit.RenterEmails = []string{raw.TenantEmail}
		}
		units = append(units, unit)
	}
	return store.NormalizeUnits(units, tenantSlug)
}

func upsertJSON(ctx context.Context, tx *sql.Tx, table string, tenant store.TenantIdentity, id string, value any) error {
	blob, err := json.Marshal(value)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE `+table+` SET tenant_id=$1,tenant_slug=$2,data=$3 WHERE tenant_slug=$2 AND id=$4`, tenant.ID, tenant.Slug, string(blob), id)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil || updated > 0 {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO `+table+`(tenant_id,tenant_slug,id,data) VALUES($1,$2,$3,$4)`, tenant.ID, tenant.Slug, id, string(blob))
	return err
}

func reset(ctx context.Context, database *sql.DB, orgKey string, houses []seedHouse, intake []seedIntake, events []seedEvent, announcements []seedAnnouncement, mail []mailintake.Message, documentDir string, discardAnnualStatements bool) error {
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	postgres := strings.Contains(fmt.Sprintf("%T", database.Driver()), "stdlib")
	if postgres {
		if _, err := tx.ExecContext(ctx, `SET LOCAL hausv.cross_tenant = 'on'`); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `SELECT set_config('app.org_key',$1,true)`, orgKey); err != nil {
			return err
		}
	}
	if err := resetMailboxFixture(ctx, tx, orgKey, houses, mail); err != nil {
		return err
	}
	for _, raw := range intake {
		if _, err := tx.ExecContext(ctx, `DELETE FROM issues WHERE id=$1`, "issue-"+raw.ID); err != nil {
			return err
		}
	}
	for _, raw := range events {
		if _, err := tx.ExecContext(ctx, `DELETE FROM events WHERE id=$1`, raw.ID); err != nil {
			return err
		}
	}
	for _, raw := range announcements {
		if _, err := tx.ExecContext(ctx, `DELETE FROM announcements WHERE id=$1`, raw.ID); err != nil {
			return err
		}
	}
	// Units are keyed by a normalized label; a fixture change (a renamed house,
	// a corrected e-mail, a new normalization) would otherwise leave the old
	// rows behind and the portal would resolve occupants from stale records.
	for _, house := range houses {
		slug := textutil.Slug(house.Slug)
		for _, query := range []string{
			`DELETE FROM valorisation_state WHERE tenant_slug=$1`,
			`DELETE FROM index_clauses WHERE tenant_slug=$1`,
			`DELETE FROM rent_components WHERE tenant_slug=$1`,
			`DELETE FROM lease_parties WHERE tenant_slug=$1`,
			`DELETE FROM leases WHERE tenant_slug=$1`,
			`DELETE FROM units WHERE tenant_slug=$1`,
		} {
			if _, err := tx.ExecContext(ctx, query, slug); err != nil {
				return err
			}
		}
	}
	// A fresh demo day imports each mailbox fixture once again. The ledger
	// prevents duplicate imports until the next reset. Rights a demo visitor
	// configured (HAUSV-699) go back to the standard matrix as well (HAUSV-730).
	for _, table := range []string{"intake_items", "intake_mail_seen", "org_settings", "textbausteine", "organisation_capability_overrides", "user_capability_grants", "capability_profiles"} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE org_key=$1`, orgKey); err != nil {
			return err
		}
	}
	// Annual statement runs, their archive documents and the delivery log are
	// immutable records a plain reset keeps; only the demo's clean-day reset
	// discards them (HAUSV-663), so the customer never meets yesterday's
	// "Lauf 6". Archive files leave the disk after the commit; a file that is
	// already gone is not an error.
	var archived []string
	if !discardAnnualStatements {
		return tx.Commit()
	}
	for _, house := range houses {
		slug := textutil.Slug(house.Slug)
		paths, err := archivedDocumentPaths(ctx, tx, slug, documentDir)
		if err != nil {
			return err
		}
		archived = append(archived, paths...)
		for _, query := range []string{
			`DELETE FROM documents WHERE tenant_slug=$1 AND id LIKE 'annual-archive-%'`,
			`DELETE FROM annual_statement_deliveries WHERE tenant_slug=$1`,
		} {
			if _, err := tx.ExecContext(ctx, query, slug); err != nil {
				return err
			}
		}
	}
	// PostgreSQL guards stored runs with a trigger that rejects every UPDATE
	// and DELETE. The demo reset is the one legitimate way to discard them, so
	// the trigger is switched off for this transaction only; the DDL rolls
	// back with everything else if the reset fails.
	if postgres {
		if _, err := tx.ExecContext(ctx, `ALTER TABLE annual_statement_runs DISABLE TRIGGER annual_statement_run_immutable`); err != nil {
			return fmt.Errorf("demo reset needs to own annual_statement_runs to discard stored runs: %w", err)
		}
	}
	for _, house := range houses {
		if _, err := tx.ExecContext(ctx, `DELETE FROM annual_statement_runs WHERE tenant_slug=$1`, textutil.Slug(house.Slug)); err != nil {
			return err
		}
	}
	if postgres {
		if _, err := tx.ExecContext(ctx, `ALTER TABLE annual_statement_runs ENABLE TRIGGER annual_statement_run_immutable`); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	for _, path := range archived {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove archived demo document %s: %w", path, err)
		}
	}
	return nil
}

// archivedDocumentPaths lists the files behind a house's annual statement
// archive documents, following the store's layout (<dir>/<tenant>/<stored name>).
func archivedDocumentPaths(ctx context.Context, tx *sql.Tx, slug, documentDir string) ([]string, error) {
	if documentDir == "" {
		return nil, nil
	}
	rows, err := tx.QueryContext(ctx, `SELECT data FROM documents WHERE tenant_slug=$1 AND id LIKE 'annual-archive-%'`, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var record store.DocumentRecord
		if err := json.Unmarshal(raw, &record); err != nil {
			return nil, err
		}
		name := filepath.Base(record.StoredFilename)
		tenant := textutil.Slug(record.TenantSlug)
		if tenant == "" || name == "" || name == "." || name == string(filepath.Separator) {
			continue
		}
		paths = append(paths, filepath.Join(documentDir, tenant, name))
	}
	return paths, rows.Err()
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func printStats(out io.Writer, result SeedResult) {
	statuses := []store.IntakeStatus{store.IntakeStatusOpen, store.IntakeStatusProposed, store.IntakeStatusApproved, store.IntakeStatusEdited, store.IntakeStatusRejected, store.IntakeStatusAuto}
	for _, status := range statuses {
		fmt.Fprintf(out, "status %s: %d\n", status, result.Statuses[status])
	}
	keys := make([]string, 0, len(result.Categories))
	for key := range result.Categories {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(out, "category %s: %d\n", key, result.Categories[key])
	}
}

// dueFor derives the demo due date from the priority window used in the concept:
// Dringend heute, Hoch morgen, Mittel in vier Tagen, Niedrig in vierzehn Tagen.
func dueFor(received time.Time, priority string) time.Time {
	switch priority {
	case store.IssuePriorityUrgent:
		return received.Add(4 * time.Hour)
	case store.IssuePriorityHigh:
		return received.Add(24 * time.Hour)
	case store.IssuePriorityLow:
		return received.Add(14 * 24 * time.Hour)
	default:
		return received.Add(4 * 24 * time.Hour)
	}
}

func seedPriority(raw seedIntake) string {
	if raw.Truth != nil && raw.Truth.Priority != "" {
		return raw.Truth.Priority
	}
	if raw.Precomputed != nil && raw.Precomputed.Priority != "" {
		return raw.Precomputed.Priority
	}
	return store.IssuePriorityNorm
}

// shiftSeedDates moves all fixture timestamps by whole days so the demo day
// becomes the anchor's day; times of day are preserved.
func shiftSeedDates(anchor time.Time, intake []seedIntake, events []seedEvent, announcements []seedAnnouncement) {
	a := anchor.In(seedDemoDay.Location())
	target := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, seedDemoDay.Location())
	shift := target.Sub(seedDemoDay)
	if shift == 0 {
		return
	}
	clampToNow := a.Hour() != 0 || a.Minute() != 0
	for i := range intake {
		intake[i].ReceivedAt = intake[i].ReceivedAt.Add(shift)
		if clampToNow && intake[i].ReceivedAt.After(a) {
			intake[i].ReceivedAt = a.Add(-time.Duration(i%45+1) * time.Minute)
		}
	}
	for i := range events {
		events[i].StartsAt = events[i].StartsAt.Add(shift)
		events[i].EndsAt = events[i].EndsAt.Add(shift)
	}
	for i := range announcements {
		announcements[i].PublishedAt = announcements[i].PublishedAt.Add(shift)
	}
}

// seedCounters uses the same action names as the intake pipeline. Assigning
// totals rather than incrementing keeps repeated loads and resets stable.
func seedCounters(items []seedIntake) store.OrgCounters {
	var counters store.OrgCounters
	for _, item := range items {
		switch item.StatusHint {
		case "approved":
			counters.Approved++
		case "edited":
			counters.Edited++
		case "manual":
			counters.Rejected++
		case "auto_done":
			counters.Auto++
		}
	}
	return counters
}

func shiftHouseDates(anchor time.Time, houses []seedHouse) {
	a := anchor.In(seedDemoDay.Location())
	shift := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, seedDemoDay.Location()).Sub(seedDemoDay)
	move := func(at time.Time) time.Time {
		if at.IsZero() {
			return at
		}
		return at.Add(shift)
	}
	for i := range houses {
		for j := range houses[i].Contacts {
			item := &houses[i].Contacts[j]
			item.CreatedAt, item.UpdatedAt = move(item.CreatedAt), move(item.UpdatedAt)
		}
		for j := range houses[i].Ballots {
			item := &houses[i].Ballots[j]
			item.OpensAt, item.ClosesAt = move(item.OpensAt), move(item.ClosesAt)
			item.CreatedAt, item.UpdatedAt = move(item.CreatedAt), move(item.UpdatedAt)
			for email, vote := range item.Votes {
				vote.At = move(vote.At)
				item.Votes[email] = vote
			}
			for email, at := range item.ReminderSentAt {
				item.ReminderSentAt[email] = move(at)
			}
		}
	}
}

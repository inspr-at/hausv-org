package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestScopedAuditUsesCurrentAuthorizationAndRedactsPersonalData(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["manager@example.com"] = userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.profiles["service@example.com"] = userProfile{Email: "service@example.com", Role: roleServiceProvider, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	a.serviceAccessEnabled = true

	if err := a.unitStore.SetTenantUnits("demo", []unit{{
		ID:          "top-1",
		TenantSlug:  "demo",
		Label:       "Top 1",
		OwnerEmails: []string{"resident@example.com"},
	}}); err != nil {
		t.Fatalf("SetTenantUnits: %v", err)
	}
	ownIssue, err := a.issueStore.Create(residentIssue{
		TenantSlug:   "demo",
		AuthorEmail:  "resident@example.com",
		AuthorName:   "Resident",
		Category:     "Reparatur",
		Title:        "Eigener Vorgang",
		Body:         "Nur für die eigene Einheit.",
		LocationType: issueLocationUnit,
	})
	if err != nil {
		t.Fatalf("Create resident issue: %v", err)
	}
	providerIssue, err := a.issueStore.Create(residentIssue{
		TenantSlug:    "demo",
		AuthorEmail:   "resident@example.com",
		AuthorName:    "Resident",
		Category:      "Reparatur",
		Title:         "Zugewiesener Vorgang",
		Body:          "Nur für den Dienstleister.",
		LocationType:  issueLocationCommon,
		AssigneeEmail: "service@example.com",
	})
	if err != nil {
		t.Fatalf("Create provider issue: %v", err)
	}
	hiddenIssue, err := a.issueStore.Create(residentIssue{
		TenantSlug:   "demo",
		AuthorEmail:  "other@example.com",
		AuthorName:   "Other",
		Category:     "Reparatur",
		Title:        "Verborgener Vorgang",
		Body:         "Nicht freigegeben.",
		LocationType: issueLocationUnit,
	})
	if err != nil {
		t.Fatalf("Create hidden issue: %v", err)
	}

	for _, event := range []auditEvent{
		{
			TenantSlug: "demo", ActorEmail: "manager@example.com", ActorRole: roleManager,
			Action: auditActionIssueWorkflow, TargetType: "issue", TargetID: ownIssue.ID,
			Summary: "PERSONAL-SUMMARY-MUST-NOT-LEAK",
			Details: map[string]string{"status": "In Bearbeitung", "paid_by": "private@example.com", "payment_reference": "SECRET-REFERENCE"},
		},
		{
			TenantSlug: "demo", ActorEmail: "manager@example.com", ActorRole: roleManager,
			Action: auditActionIssueWorkflow, TargetType: "issue", TargetID: providerIssue.ID,
			Summary: "Provider workflow", Details: map[string]string{"status": "Angenommen"},
		},
		{
			TenantSlug: "demo", ActorEmail: "manager@example.com", ActorRole: roleManager,
			Action: auditActionIssueWorkflow, TargetType: "issue", TargetID: hiddenIssue.ID,
			Summary: "Hidden workflow",
		},
		{
			TenantSlug: "demo", ActorEmail: "manager@example.com", ActorRole: roleManager,
			Action: auditActionUnitPayment, TargetType: "unit", TargetID: "top-1",
			Summary: "Unit payment", Details: map[string]string{"status": "Bezahlt", "payment_reference": "UNIT-SECRET"},
		},
		{
			TenantSlug: "demo", ActorEmail: "manager@example.com", ActorRole: roleManager,
			Action: auditActionInviteCreate, TargetType: "user", TargetID: "other.person@example.com",
			Summary: "Unrelated invite",
		},
		{
			TenantSlug: "demo", ActorEmail: "resident@example.com", ActorRole: roleOwner,
			Action: auditActionLogin, TargetType: "session", TargetID: "resident@example.com",
			Summary: "Login with unsafe prose", Details: map[string]string{"auth_method": "E-Mail-Link", "secret_token": "must-not-exist"},
		},
		{
			TenantSlug: "other-house", ActorEmail: "resident@example.com", ActorRole: roleOwner,
			Action: auditActionLogin, TargetType: "session", TargetID: "cross-tenant@example.com",
			Summary: "Cross tenant",
		},
	} {
		if err := a.auditStore.Append(event); err != nil {
			t.Fatalf("append audit event: %v", err)
		}
	}

	residentPage := authedRequest(t, a, "resident@example.com", "/demo/app/audit")
	if residentPage.Code != http.StatusOK {
		t.Fatalf("resident audit status = %d", residentPage.Code)
	}
	residentBody := residentPage.Body.String()
	for _, want := range []string{"Verwaltung", "Sie", "Eigener Vorgang", "Top 1", "In Bearbeitung", "E-Mail-Link", "Anliegen bearbeitet"} {
		if !strings.Contains(residentBody, want) {
			t.Fatalf("resident audit missing %q:\n%s", want, residentBody)
		}
	}
	for _, forbidden := range []string{
		"manager@example.com", "private@example.com", "SECRET-REFERENCE", "UNIT-SECRET",
		"PERSONAL-SUMMARY-MUST-NOT-LEAK", ownIssue.ID, hiddenIssue.ID, "other.person@example.com", "cross-tenant@example.com",
	} {
		if strings.Contains(residentBody, forbidden) {
			t.Fatalf("resident audit leaks %q:\n%s", forbidden, residentBody)
		}
	}

	providerPage := authedRequest(t, a, "service@example.com", "/demo/app/audit")
	if providerPage.Code != http.StatusOK {
		t.Fatalf("provider audit status = %d", providerPage.Code)
	}
	providerBody := providerPage.Body.String()
	if !strings.Contains(providerBody, "Zugewiesener Vorgang") || !strings.Contains(providerBody, "Verwaltung") {
		t.Fatalf("provider audit missing assigned history:\n%s", providerBody)
	}
	for _, forbidden := range []string{providerIssue.ID, ownIssue.ID, hiddenIssue.ID, "manager@example.com", `href="/demo/app/settings"`} {
		if strings.Contains(providerBody, forbidden) {
			t.Fatalf("provider audit leaks or links forbidden value %q:\n%s", forbidden, providerBody)
		}
	}

	managerPage := authedRequest(t, a, "manager@example.com", "/demo/app/audit")
	if managerPage.Code != http.StatusOK {
		t.Fatalf("manager audit status = %d", managerPage.Code)
	}
	managerBody := managerPage.Body.String()
	for _, want := range []string{"manager@example.com", "private@example.com", "PERSONAL-SUMMARY-MUST-NOT-LEAK", hiddenIssue.ID} {
		if !strings.Contains(managerBody, want) {
			t.Fatalf("manager full audit missing %q:\n%s", want, managerBody)
		}
	}
	if strings.Contains(managerBody, "cross-tenant@example.com") {
		t.Fatalf("manager audit leaks another tenant:\n%s", managerBody)
	}
}

func TestAuditActionsExposeUnderstandableLabelsAndTones(t *testing.T) {
	cases := []struct {
		action string
		label  string
		tone   string
	}{
		{auditActionAttachmentView, "Anhang angesehen", "change"},
		{auditActionAttachmentDelete, "Anhang entfernt", "danger"},
		{auditActionIntegrationImport, "Integration importiert", "change"},
		{auditActionInviteCreate, "Einladung angelegt", "add"},
		{auditActionIssueWorkflow, "Anliegen bearbeitet", "change"},
	}
	for _, tc := range cases {
		if got := auditActionLabel(tc.action); got != tc.label {
			t.Errorf("auditActionLabel(%q) = %q, want %q", tc.action, got, tc.label)
		}
		if got := auditActionTone(tc.action); got != tc.tone {
			t.Errorf("auditActionTone(%q) = %q, want %q", tc.action, got, tc.tone)
		}
	}
	if got := auditTargetTypeLabel("unit"); got != "Einheit" {
		t.Errorf("unit target label = %q", got)
	}
	if got := auditTargetTypeLabel("ballot"); got != "Abstimmung" {
		t.Errorf("ballot target label = %q", got)
	}
}

func TestAuditEventViewUsesOneReadableTitleAndCompactContext(t *testing.T) {
	login := auditEventViewFrom(auditEvent{
		ActorEmail: "Sie",
		Action:     auditActionLogin,
		TargetType: "session",
		TargetID:   "resident@example.com",
		Summary:    "Anmeldung",
	})
	if login.DisplayTitle != "Anmeldung" || login.Context != "Sie" || strings.Contains(login.Context, "resident@example.com") {
		t.Fatalf("compact login view = %+v", login)
	}

	document := auditEventViewFrom(auditEvent{
		ActorEmail: "resident@example.com",
		ActorRole:  roleResident,
		Action:     auditActionDocumentDownload,
		TargetType: "document",
		TargetID:   "Hausordnung 2026",
		Summary:    "Hausordnung 2026 heruntergeladen",
	})
	if document.DisplayTitle != "Hausordnung 2026 heruntergeladen" || document.Context != "resident@example.com · Bewohner · Dokument: Hausordnung 2026" {
		t.Fatalf("compact document view = %+v", document)
	}
}

func TestAuditFilterOffersOnlyActionsVisibleInCurrentScope(t *testing.T) {
	options := auditActionOptionsForEvents(auditActionDocumentDownload, []auditEvent{
		{Action: auditActionLogin},
		{Action: auditActionDocumentDownload},
		{Action: auditActionLogin},
	})
	if len(options) != 3 {
		t.Fatalf("action options = %+v, want all plus two visible actions", options)
	}
	if options[0].Label != "Alle Arten" || !options[2].Selected {
		t.Fatalf("action option hierarchy = %+v", options)
	}
}

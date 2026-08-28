package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func newSupportViewTestApp(t *testing.T, supportPermission bool) *app {
	t.Helper()
	permissions := []string{}
	if supportPermission {
		permissions = append(permissions, permissionSupportView)
	}
	a := newTestPortalApp(t, userProfile{
		Email: "admin@example.com", FirstName: "Ada", LastName: "Admin", Role: roleAdmin,
		Tenants: []string{"demo"}, Permissions: permissions, AuthMethods: defaultAuthMethods(),
	})
	a.profiles["resident@example.com"] = userProfile{
		Email: "resident@example.com", FirstName: "Rita", LastName: "Resident", Role: roleResident,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(), Status: "Aktiv",
	}
	a.profiles["manager@example.com"] = userProfile{
		Email: "manager@example.com", FirstName: "Mara", LastName: "Manager", Role: roleManager,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(), Status: "Aktiv",
	}
	return a
}

func supportViewPost(t *testing.T, a *app, token string, path string, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "http://hausv.org/demo"+path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://hausv.org/demo")
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	return rr
}

func startSupportViewForResident(t *testing.T, a *app) (string, int64) {
	return startSupportViewForTarget(t, a, "resident@example.com", roleResident)
}

func startSupportViewForTarget(t *testing.T, a *app, targetEmail string, targetRole string) (string, int64) {
	t.Helper()
	parentExpiry := time.Now().Add(50 * time.Minute).Truncate(time.Second)
	oldToken, _, err := a.sessions.PutSession("admin@example.com", "demo", authMethodEmail, roleAdmin, parentExpiry)
	if err != nil {
		t.Fatal(err)
	}
	rr := supportViewPost(t, a, oldToken, "/app/support-view/start", url.Values{
		"tenant": {"demo"}, "target_email": {targetEmail}, "target_role": {targetRole},
	})
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("start status = %d: %s", rr.Code, rr.Body.String())
	}
	cookie := sessionCookieFrom(t, rr)
	if _, ok := a.sessions.GetSession(oldToken); ok {
		t.Fatal("starting support view must revoke the ordinary session")
	}
	return cookie.Value, parentExpiry.Unix()
}

func supportViewGet(t *testing.T, a *app, token string, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo"+path, nil)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	return rr
}

func assertSupportAndOrdinaryReadAudits(t *testing.T, events []auditEvent, ordinaryEmail string, ordinaryRole string) {
	t.Helper()
	if len(events) != 2 {
		t.Fatalf("read audit events = %+v, want support and ordinary access", events)
	}
	seenSupport := false
	seenOrdinary := false
	for _, event := range events {
		switch event.ActorEmail {
		case "admin@example.com":
			seenSupport = true
			if event.ActorRole != roleAdmin || event.Details["support_target_email"] != ordinaryEmail || event.Details["support_target_role"] != ordinaryRole {
				t.Fatalf("support read audit = %+v", event)
			}
		case ordinaryEmail:
			seenOrdinary = true
			if event.ActorRole != ordinaryRole || event.Details["support_target_email"] != "" || event.Details["support_target_role"] != "" {
				t.Fatalf("ordinary read audit = %+v", event)
			}
		default:
			t.Fatalf("read audit attributed to unexpected actor: %+v", event)
		}
	}
	if !seenSupport || !seenOrdinary {
		t.Fatalf("read audit actors support=%v ordinary=%v events=%+v", seenSupport, seenOrdinary, events)
	}
}

func TestSupportViewReadAuditsUseRealAdminAndTargetContext(t *testing.T) {
	t.Run("resident document and attachment", func(t *testing.T) {
		a := newSupportViewTestApp(t, true)
		document, err := documentRepositoryForTest(a, "demo").Create(documentRecord{
			TenantSlug: "demo",
			Title:      "Hausordnung",
			Category:   documentCategoryRules,
			Visibility: documentVisibilityAllResidents,
			UploadedBy: "admin@example.com",
		}, testMultipartHeader(t, "document", "hausordnung.pdf", []byte("%PDF-1.4\n% support audit\n")), time.Now())
		if err != nil {
			t.Fatalf("create document: %v", err)
		}
		issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
			TenantSlug:   "demo",
			AuthorEmail:  "resident@example.com",
			AuthorName:   "Rita Resident",
			Category:     "Reparatur",
			Title:        "Tür klemmt",
			Body:         "Bitte prüfen.",
			LocationType: issueLocationCommon,
		})
		if err != nil {
			t.Fatalf("create issue: %v", err)
		}
		attachments, err := attachmentRepositoryForTest(a, "demo").CreateUploaded("issue", issue.ID, "resident@example.com", []uploadedFile{
			testMultipartHeader(t, "attachments", "tuer.png", minimalPNG()),
		}, time.Now())
		if err != nil || len(attachments) != 1 {
			t.Fatalf("create attachment = %+v err=%v", attachments, err)
		}

		token, _ := startSupportViewForResident(t, a)
		if rr := supportViewGet(t, a, token, "/app/dokumente/"+document.ID+"/download"); rr.Code != http.StatusOK {
			t.Fatalf("support document download status=%d body=%s", rr.Code, rr.Body.String())
		}
		if rr := supportViewGet(t, a, token, "/app/attachments/"+attachments[0].ID+"/preview"); rr.Code != http.StatusOK {
			t.Fatalf("support attachment preview status=%d body=%s", rr.Code, rr.Body.String())
		}
		if rr := authedRequest(t, a, "resident@example.com", "/demo/app/dokumente/"+document.ID+"/download"); rr.Code != http.StatusOK {
			t.Fatalf("ordinary document download status=%d", rr.Code)
		}
		if rr := authedRequest(t, a, "resident@example.com", "/demo/app/attachments/"+attachments[0].ID+"/preview"); rr.Code != http.StatusOK {
			t.Fatalf("ordinary attachment preview status=%d", rr.Code)
		}

		assertSupportAndOrdinaryReadAudits(t, a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionDocumentDownload, Limit: 10}), "resident@example.com", roleResident)
		assertSupportAndOrdinaryReadAudits(t, a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionAttachmentView, Limit: 10}), "resident@example.com", roleResident)
	})

	t.Run("manager handover protocol", func(t *testing.T) {
		a := newSupportViewTestApp(t, true)
		item, err := testRepositories(a, "demo").handovers.Create(handoverRecord{
			ID:           "handover-support-audit",
			TenantSlug:   "demo",
			UnitID:       "top-1",
			Title:        "Übergabe Top 1",
			HandoverType: "Nutzerwechsel",
			CreatedBy:    "manager@example.com",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		})
		if err != nil {
			t.Fatalf("create handover: %v", err)
		}
		token, _ := startSupportViewForTarget(t, a, "manager@example.com", roleManager)
		if rr := supportViewGet(t, a, token, "/app/uebergaben/"+item.ID+"/protokoll"); rr.Code != http.StatusOK {
			t.Fatalf("support handover protocol status=%d body=%s", rr.Code, rr.Body.String())
		}
		if rr := authedRequest(t, a, "manager@example.com", "/demo/app/uebergaben/"+item.ID+"/protokoll"); rr.Code != http.StatusOK {
			t.Fatalf("ordinary handover protocol status=%d", rr.Code)
		}
		assertSupportAndOrdinaryReadAudits(t, a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionDocumentDownload, Limit: 10}), "manager@example.com", roleManager)
	})
}

func TestSupportViewRequiresSeparatePermissionAndExactTargetRole(t *testing.T) {
	withoutPermission := newSupportViewTestApp(t, false)
	token, _, err := withoutPermission.sessions.PutSession("admin@example.com", "demo", authMethodEmail, roleAdmin, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	denied := supportViewPost(t, withoutPermission, token, "/app/support-view/start", url.Values{
		"tenant": {"demo"}, "target_email": {"resident@example.com"}, "target_role": {roleResident},
	})
	if denied.Code != http.StatusForbidden || len(denied.Result().Cookies()) != 0 {
		t.Fatalf("missing permission status=%d cookies=%v", denied.Code, denied.Result().Cookies())
	}

	withPermission := newSupportViewTestApp(t, true)
	for name, values := range map[string]url.Values{
		"role elevation": {"tenant": {"demo"}, "target_email": {"resident@example.com"}, "target_role": {roleAdmin}},
		"tenant tamper":  {"tenant": {"other"}, "target_email": {"resident@example.com"}, "target_role": {roleResident}},
		"unknown user":   {"tenant": {"demo"}, "target_email": {"outsider@example.com"}, "target_role": {roleResident}},
	} {
		t.Run(name, func(t *testing.T) {
			token, _, err := withPermission.sessions.PutSession("admin@example.com", "demo", authMethodEmail, roleAdmin, time.Now().Add(time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			rr := supportViewPost(t, withPermission, token, "/app/support-view/start", values)
			if rr.Code != http.StatusForbidden || len(rr.Result().Cookies()) != 0 {
				t.Fatalf("status=%d cookies=%v body=%s", rr.Code, rr.Result().Cookies(), rr.Body.String())
			}
		})
	}
}

func TestSupportViewChooserIsVisibleOnlyWithExplicitPermission(t *testing.T) {
	withPermission := newSupportViewTestApp(t, true)
	page := authedRequest(t, withPermission, "admin@example.com", "/demo/app/settings/users")
	if page.Code != http.StatusOK {
		t.Fatalf("chooser page status=%d", page.Code)
	}
	for _, want := range []string{"data-support-view-start", "Rita Resident", `name="target_role" value="Bewohner"`, "15 Minuten · schreibgeschützt"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("chooser missing %q", want)
		}
	}

	withoutPermission := newSupportViewTestApp(t, false)
	page = authedRequest(t, withoutPermission, "admin@example.com", "/demo/app/settings/users")
	if page.Code != http.StatusOK || strings.Contains(page.Body.String(), "data-support-view-start") {
		t.Fatalf("chooser without permission status=%d visible=%v", page.Code, strings.Contains(page.Body.String(), "data-support-view-start"))
	}
}

func TestNonAdminCannotGrantOrSilentlyClearSupportViewPermission(t *testing.T) {
	a := newSupportViewTestApp(t, true)

	created := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/users", url.Values{
		"email":        {"new-resident@example.com"},
		"role":         {roleResident},
		"permissions":  {permissionParking, permissionSupportView},
		"auth_methods": {authMethodEmail},
	})
	if created.Code != http.StatusSeeOther {
		t.Fatalf("manager create status=%d body=%s", created.Code, created.Body.String())
	}
	createdProfile, ok := a.inviteStore.Get("new-resident@example.com")
	if !ok || createdProfile.ForTenant("demo").HasPermission(permissionSupportView) {
		t.Fatalf("manager granted support-view on create: %+v ok=%v", createdProfile, ok)
	}
	if !createdProfile.ForTenant("demo").HasPermission(permissionParking) {
		t.Fatalf("unrelated submitted permission was lost on create: %+v", createdProfile)
	}

	if _, err := a.inviteStore.Add(userProfile{
		Email: "existing-grant@example.com", Role: roleResident, Tenants: []string{"demo"},
		Permissions: []string{permissionSupportView}, AuthMethods: defaultAuthMethods(),
	}); err != nil {
		t.Fatal(err)
	}
	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/users")
	if page.Code != http.StatusOK || strings.Contains(page.Body.String(), `value="support-view"`) {
		t.Fatalf("manager page exposed support-view grant control: status=%d", page.Code)
	}
	edited := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/users/edit", url.Values{
		"orig_email":   {"existing-grant@example.com"},
		"email":        {"existing-grant@example.com"},
		"role":         {roleResident},
		"permissions":  {permissionParking},
		"auth_methods": {authMethodEmail},
	})
	if edited.Code != http.StatusSeeOther {
		t.Fatalf("manager edit status=%d body=%s", edited.Code, edited.Body.String())
	}
	editedProfile, ok := a.inviteStore.Get("existing-grant@example.com")
	if !ok || !editedProfile.ForTenant("demo").HasPermission(permissionSupportView) {
		t.Fatalf("hidden support-view grant was silently cleared: %+v ok=%v", editedProfile, ok)
	}
	if !editedProfile.ForTenant("demo").HasPermission(permissionParking) {
		t.Fatalf("ordinary manager edit did not retain submitted permission: %+v", editedProfile)
	}
}

func TestReloginEndsAndRevokesExistingSupportViewSession(t *testing.T) {
	a := newSupportViewTestApp(t, true)
	startedAt := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	parentExpiry := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	supportToken, _, err := a.sessions.PutSupportView(
		"admin@example.com", "demo", authMethodEmail, roleAdmin,
		"resident@example.com", roleResident, startedAt, time.Now().UTC().Add(10*time.Minute), parentExpiry,
	)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/auth/verify", nil)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: supportToken})
	result := httptest.NewRecorder()
	if err := a.startSession(result, req, "admin@example.com", "demo", authMethodEmail); err != nil {
		t.Fatal(err)
	}
	if _, ok := a.sessions.GetSession(supportToken); ok {
		t.Fatal("re-login left the replaced support token valid")
	}
	ordinaryCookie := sessionCookieFrom(t, result)
	ordinary, ok := a.sessions.GetSession(ordinaryCookie.Value)
	if !ok || ordinary.Email != "admin@example.com" || ordinary.Role != "" || ordinary.SupportTargetEmail != "" {
		t.Fatalf("re-login session=%+v ok=%v", ordinary, ok)
	}
	ends := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionSupportViewEnd, Limit: 10})
	if len(ends) != 1 || ends[0].ActorEmail != "admin@example.com" || ends[0].ActorRole != roleAdmin ||
		ends[0].TargetID != "resident@example.com" || ends[0].Details["target_role"] != roleResident ||
		ends[0].Details["reason"] != "relogin" {
		t.Fatalf("re-login support end audit=%+v", ends)
	}
}

func TestSupportViewLegacyIssueDetailLoadsBannerStylesWithoutPublicLeak(t *testing.T) {
	a := newSupportViewTestApp(t, true)
	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:   "demo",
		AuthorEmail:  "resident@example.com",
		AuthorName:   "Rita Resident",
		Category:     "Reparatur",
		Title:        "Tür klemmt",
		Body:         "Bitte prüfen.",
		LocationType: issueLocationCommon,
	})
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}
	token, _ := startSupportViewForResident(t, a)
	page := supportViewGet(t, a, token, "/app/anliegen/"+issue.ID)
	if page.Code != http.StatusOK {
		t.Fatalf("legacy support issue status=%d body=%s", page.Code, page.Body.String())
	}
	body := page.Body.String()
	stylesheet := `/assets/support-view.css?v=`
	for _, want := range []string{
		stylesheet,
		`class="support-view-banner legacy-support-view-banner"`,
		`data-support-view-banner`,
		`class="app-shell"`,
		`id="main-content" tabindex="-1" class="app-main"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("legacy support issue missing %q", want)
		}
	}
	if strings.Count(body, stylesheet) != 1 || strings.Index(body, stylesheet) > strings.Index(body, "</head>") {
		t.Fatalf("legacy support stylesheet must appear exactly once in head")
	}

	public := httptest.NewRecorder()
	a.handler().ServeHTTP(public, httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/", nil))
	if public.Code != http.StatusOK {
		t.Fatalf("public home status=%d body=%s", public.Code, public.Body.String())
	}
	if strings.Contains(public.Body.String(), stylesheet) {
		t.Fatal("public home must not load the authenticated support-view stylesheet")
	}
}

func TestSupportViewUsesTargetPermissionsBlocksWritesAndExitsSafely(t *testing.T) {
	a := newSupportViewTestApp(t, true)
	token, parentExpiry := startSupportViewForResident(t, a)
	session, ok := a.sessions.GetSession(token)
	if !ok || session.Email != "admin@example.com" || session.Role != roleAdmin || session.SupportTargetEmail != "resident@example.com" || session.SupportTargetRole != roleResident {
		t.Fatalf("support session = %+v ok=%v", session, ok)
	}
	if session.ExpiresAt != parentExpiry || session.SupportExpiresAt > time.Now().Add(supportViewTTL+time.Second).Unix() {
		t.Fatalf("support expiry=%d parent=%d", session.SupportExpiresAt, session.ExpiresAt)
	}

	pageReq := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app", nil)
	pageReq.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	page := httptest.NewRecorder()
	a.handler().ServeHTTP(page, pageReq)
	body := page.Body.String()
	for _, want := range []string{"Portal anzeigen als Rita Resident", roleResident, "Supportansicht beenden", "data-support-view-banner"} {
		if !strings.Contains(body, want) {
			t.Fatalf("support portal missing %q", want)
		}
	}
	if strings.Contains(body, ">Benutzer &amp; Rechte<") {
		t.Fatal("support portal leaked admin user-management navigation")
	}
	announcementsReq := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app/announcements", nil)
	announcementsReq.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	announcementsResult := httptest.NewRecorder()
	a.handler().ServeHTTP(announcementsResult, announcementsReq)
	if announcementsResult.Code != http.StatusOK {
		t.Fatalf("support announcements status=%d", announcementsResult.Code)
	}
	reads, _ := store.BindAnnouncementReadRepository(a.announcementReadStore, testTenantRef("demo"))
	if !reads.LastSeen("resident@example.com").IsZero() {
		t.Fatal("support view changed the target user's unread state")
	}

	before := len(testRepositories(a, "demo").announcements.List())
	write := supportViewPost(t, a, token, "/app/announcements", url.Values{"title": {"Forbidden"}, "body": {"No write"}})
	if write.Code != http.StatusForbidden || len(testRepositories(a, "demo").announcements.List()) != before {
		t.Fatalf("support write status=%d announcements=%d", write.Code, len(testRepositories(a, "demo").announcements.List()))
	}

	exit := supportViewPost(t, a, token, "/app/support-view/end", nil)
	if exit.Code != http.StatusSeeOther {
		t.Fatalf("exit status=%d body=%s", exit.Code, exit.Body.String())
	}
	restoredCookie := sessionCookieFrom(t, exit)
	restored, ok := a.sessions.GetSession(restoredCookie.Value)
	if !ok || restored.Email != "admin@example.com" || restored.Role != roleAdmin || restored.SupportTargetEmail != "" || restored.ExpiresAt != parentExpiry {
		t.Fatalf("restored session=%+v ok=%v", restored, ok)
	}
	if _, ok := a.sessions.GetSession(token); ok {
		t.Fatal("support token remains valid after exit")
	}
	starts := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionSupportViewStart, Limit: 10})
	ends := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionSupportViewEnd, Limit: 10})
	if len(starts) != 1 || len(ends) != 1 || starts[0].ActorEmail != "admin@example.com" || starts[0].TargetID != "resident@example.com" || ends[0].Details["reason"] != "explicit" {
		t.Fatalf("support audit starts=%+v ends=%+v", starts, ends)
	}
}

func TestExpiredSupportViewRestoresAdminAndAuditsEndBeforeAnyPage(t *testing.T) {
	a := newSupportViewTestApp(t, true)
	now := time.Now().UTC().Truncate(time.Second)
	parentExpiry := now.Add(time.Hour)
	token, _, err := a.sessions.PutSupportView("admin@example.com", "demo", authMethodEmail, roleAdmin, "resident@example.com", roleResident, now.Add(-2*time.Minute), now.Add(-time.Minute), parentExpiry)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app", nil)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expired support status=%d body=%s", rr.Code, rr.Body.String())
	}
	restoredCookie := sessionCookieFrom(t, rr)
	restored, ok := a.sessions.GetSession(restoredCookie.Value)
	if !ok || restored.Email != "admin@example.com" || restored.Role != roleAdmin || restored.SupportTargetEmail != "" || restored.ExpiresAt != parentExpiry.Unix() {
		t.Fatalf("expired restored session=%+v ok=%v", restored, ok)
	}
	if _, ok := a.sessions.GetSession(token); ok {
		t.Fatal("expired support token remains usable")
	}
	ends := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionSupportViewEnd, Limit: 10})
	if len(ends) != 1 || ends[0].Details["reason"] != "expired" || ends[0].ActorEmail != "admin@example.com" {
		t.Fatalf("expired support audit=%+v", ends)
	}
}

func TestInvalidSupportViewTerminatesTokenAndRestoresOnlyValidActorContext(t *testing.T) {
	tests := []struct {
		name        string
		invalidate  func(*app)
		wantReason  string
		wantRestore bool
	}{
		{
			name: "support permission revoked",
			invalidate: func(a *app) {
				profile := a.profiles["admin@example.com"]
				profile.Permissions = nil
				a.profiles[profile.Email] = profile
			},
			wantReason:  "actor_permission_revoked",
			wantRestore: true,
		},
		{
			name: "target role changed",
			invalidate: func(a *app) {
				profile := a.profiles["resident@example.com"]
				profile.Role = roleOwner
				a.profiles[profile.Email] = profile
			},
			wantReason:  "target_role_changed",
			wantRestore: true,
		},
		{
			name: "target deactivated",
			invalidate: func(a *app) {
				profile := a.profiles["resident@example.com"]
				profile.Deactivated = true
				a.profiles[profile.Email] = profile
			},
			wantReason:  "target_deactivated",
			wantRestore: true,
		},
		{
			name: "actor context invalid",
			invalidate: func(a *app) {
				profile := a.profiles["admin@example.com"]
				profile.Role = roleManager
				a.profiles[profile.Email] = profile
			},
			wantReason:  "actor_context_invalid",
			wantRestore: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := newSupportViewTestApp(t, true)
			token, parentExpiry := startSupportViewForResident(t, a)
			test.invalidate(a)

			result := supportViewGet(t, a, token, "/app")
			if result.Code != http.StatusSeeOther {
				t.Fatalf("invalid support status=%d body=%s", result.Code, result.Body.String())
			}
			if _, ok := a.sessions.GetSession(token); ok {
				t.Fatal("invalid support token remains usable")
			}
			cookies := result.Result().Cookies()
			if len(cookies) != 1 || cookies[0].Name != "weg_session" {
				t.Fatalf("invalid support cookies=%+v", cookies)
			}
			if test.wantRestore {
				restored, ok := a.sessions.GetSession(cookies[0].Value)
				if !ok || restored.Email != "admin@example.com" || restored.Role != roleAdmin || restored.SupportTargetEmail != "" || restored.ExpiresAt != parentExpiry {
					t.Fatalf("restored actor session=%+v ok=%v", restored, ok)
				}
				if test.wantReason == supportViewEndActorPermissionRevoked {
					reentry := supportViewPost(t, a, cookies[0].Value, "/app/support-view/start", url.Values{
						"tenant": {"demo"}, "target_email": {"resident@example.com"}, "target_role": {roleResident},
					})
					if reentry.Code != http.StatusForbidden || len(reentry.Result().Cookies()) != 0 {
						t.Fatalf("permission-revoked actor re-entered support view: status=%d cookies=%+v", reentry.Code, reentry.Result().Cookies())
					}
					if current, ok := a.sessions.GetSession(cookies[0].Value); !ok || current.SupportTargetEmail != "" {
						t.Fatalf("permission-revoked parent session changed after denied re-entry: %+v ok=%v", current, ok)
					}
				}
			} else {
				if cookies[0].Value != "" || cookies[0].MaxAge >= 0 {
					t.Fatalf("invalid actor context must clear cookie: %+v", cookies[0])
				}
			}
			ends := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionSupportViewEnd, Limit: 10})
			if len(ends) != 1 || ends[0].ActorEmail != "admin@example.com" || ends[0].ActorRole != roleAdmin ||
				ends[0].TargetID != "resident@example.com" || ends[0].Details["target_role"] != roleResident ||
				ends[0].Details["reason"] != test.wantReason {
				t.Fatalf("invalid support end audit=%+v", ends)
			}
		})
	}
}

func TestSupportViewCannotBeTransferredByURLOrManipulatedCookie(t *testing.T) {
	a := newSupportViewTestApp(t, true)
	token, _ := startSupportViewForResident(t, a)

	withoutCookie := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app", nil)
	withoutCookieResult := httptest.NewRecorder()
	a.handler().ServeHTTP(withoutCookieResult, withoutCookie)
	if withoutCookieResult.Code != http.StatusSeeOther {
		t.Fatalf("copied URL status=%d", withoutCookieResult.Code)
	}

	last := token[len(token)-1]
	replacement := byte('A')
	if last == replacement {
		replacement = 'B'
	}
	forged := token[:len(token)-1] + string(replacement)
	forgedRequest := httptest.NewRequest(http.MethodGet, "http://hausv.org/demo/app", nil)
	forgedRequest.AddCookie(&http.Cookie{Name: "weg_session", Value: forged})
	forgedResult := httptest.NewRecorder()
	a.handler().ServeHTTP(forgedResult, forgedRequest)
	if forgedResult.Code != http.StatusSeeOther {
		t.Fatalf("manipulated cookie status=%d", forgedResult.Code)
	}
}

func TestLogoutEndsSupportViewAndAuditsTheRealAdmin(t *testing.T) {
	a := newSupportViewTestApp(t, true)
	token, _ := startSupportViewForResident(t, a)
	logout := supportViewPost(t, a, token, "/auth/logout", nil)
	if logout.Code != http.StatusSeeOther {
		t.Fatalf("logout status=%d", logout.Code)
	}
	if _, ok := a.sessions.GetSession(token); ok {
		t.Fatal("support token remains valid after logout")
	}
	ends := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionSupportViewEnd, Limit: 10})
	if len(ends) != 1 || ends[0].ActorEmail != "admin@example.com" || ends[0].ActorRole != roleAdmin || ends[0].TargetID != "resident@example.com" || ends[0].Details["reason"] != "logout" {
		t.Fatalf("logout support audit=%+v", ends)
	}
}

package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// HAUSV-568: The Hausüberblick Anliegen table must show a Beschreibung column
// with truncated Body (~100 chars, word boundary) and Ort must show only
// IssueLocationLabel(type, detail), never the Body.
func TestPortalIssuesTableShowsBeschreibungNotBodyInOrt(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	longBody := "Die Haustür fällt nicht richtig ins Schloss. Man muss sie kräftig zuziehen, sonst bleibt sie offen stehen. Das ist besonders nachts ein Problem, weil dann die Tür nicht sicher verschlossen ist."
	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:     "demo",
		AuthorEmail:    "resident@example.com",
		AuthorName:     "Resident",
		Category:       "Reparatur",
		Title:          "Haustür defekt",
		Body:           longBody,
		LocationType:   issueLocationCommon,
		LocationDetail: "Eingangsbereich",
		Status:         issueStatusNew,
		Priority:       issuePriorityNorm,
	})
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}

	body := authedRequest(t, a, "manager@example.com", "/demo/app").Body.String()

	// The table must have a Beschreibung column header
	if !strings.Contains(body, "<th>Beschreibung</th>") {
		t.Fatal("portal issues table must have a Beschreibung column header")
	}

	// The Ort column must contain only the location label, not the Body
	if strings.Contains(body, `class="place"`) {
		// Find the place cell and verify it doesn't contain the long body text
		placeStart := strings.Index(body, `class="place"`)
		placeEnd := strings.Index(body[placeStart:], "</td>")
		placeCell := body[placeStart : placeStart+placeEnd]

		if strings.Contains(placeCell, "fällt nicht richtig ins Schloss") {
			t.Fatal("Ort column must not contain Body text")
		}
		if strings.Contains(placeCell, "kräftig zuziehen") {
			t.Fatal("Ort column must not contain Body text")
		}
		// The label should be "Gemeinschaft" with optional detail
		if !strings.Contains(placeCell, "Gemeinschaft") {
			t.Fatalf("Ort column must contain 'Gemeinschaft', got: %s", placeCell)
		}
	} else {
		t.Fatal("portal issues table must have place cells with class='place'")
	}

	// The Beschreibung column must contain truncated Body
	if !strings.Contains(body, `class="description"`) {
		t.Fatal("portal issues table must have description cells with class='description'")
	}
	descStart := strings.Index(body, `class="description"`)
	descEnd := strings.Index(body[descStart:], "</td>")
	descCell := body[descStart : descStart+descEnd]

	// Extract just the displayed text (between the last ">" and "</span>")
	spanStart := strings.Index(descCell, `<span class="description-text"`)
	if spanStart == -1 {
		t.Fatalf("Could not find description-text span in cell: %s", descCell)
	}
	textStart := strings.Index(descCell[spanStart:], `>`)
	textEnd := strings.Index(descCell[spanStart:], `</span>`)
	if textStart == -1 || textEnd == -1 {
		t.Fatalf("Could not parse description text in cell: %s", descCell)
	}
	displayedText := descCell[spanStart+textStart+1 : spanStart+textEnd]

	if !strings.Contains(displayedText, "Die Haustür fällt nicht richtig ins Schloss") {
		t.Fatalf("Beschreibung must contain start of Body text, got: %q", displayedText)
	}
	// Verify it's truncated: the full body ends with "nicht sicher verschlossen ist."
	// but the truncated display should not contain that
	if strings.Contains(displayedText, "nicht sicher verschlossen") {
		t.Fatalf("Beschreibung must be truncated at ~100 chars, got: %q", displayedText)
	}
	// Should have ellipsis for truncated text
	if !strings.Contains(displayedText, "…") {
		t.Fatalf("Truncated Beschreibung must end with ellipsis, got: %q", displayedText)
	}
	// The full body should be in the title attribute for hover display
	if !strings.Contains(descCell, `title="`+longBody+`"`) {
		t.Fatal("Beschreibung cell must have full Body in title attribute for hover display")
	}

	// Verify the detail link still goes to the correct issue
	// The link might be in the "Betreff" (subject) column
	if !strings.Contains(body, issue.ID) {
		t.Fatalf("portal issues table must contain link to issue %s", issue.ID)
	}
}

func TestManagementHouseOverviewNamesUnitWithoutOwnPrefix(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if _, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:     "demo",
		AuthorEmail:    "resident@example.com",
		AuthorName:     "Resident",
		Category:       "Reparatur",
		Title:          "Heizung kalt",
		Body:           "Seit gestern kalt.",
		LocationType:   issueLocationUnit,
		LocationDetail: "Top 7",
		Status:         issueStatusNew,
		Priority:       issuePriorityNorm,
	}); err != nil {
		t.Fatal(err)
	}
	body := authedRequest(t, a, "manager@example.com", "/demo/app").Body.String()
	if strings.Contains(body, "Eigene Einheit") {
		t.Fatal("Verwaltungsansicht zeigt noch das Präfix Eigene Einheit")
	}
	if !strings.Contains(body, ">Top 7<") && !strings.Contains(body, ">Top 7</span>") {
		t.Fatal("Verwaltungsansicht nennt die Einheit nicht")
	}
}

// HAUSV-568: Managers must be able to edit body, location_type, and
// location_detail from the triage page after an issue is created.
func TestManagerCanEditIssueDetailsFromTriagePage(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:     "demo",
		AuthorEmail:    "resident@example.com",
		AuthorName:     "Resident",
		Category:       "Reparatur",
		Title:          "Heizung kalt",
		Body:           "Die Heizung bleibt kalt.",
		LocationType:   issueLocationCommon,
		LocationDetail: "Stiegenhaus 2. Stock",
		Status:         issueStatusNew,
		Priority:       issuePriorityNorm,
	})
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}

	triageBody := authedRequest(t, a, "manager@example.com", "/demo/app/anliegen/board/"+issue.ID).Body.String()

	// Verify the edit form exists
	if !strings.Contains(triageBody, `<summary>Details bearbeiten<span class="disclosure-chevron" aria-hidden="true">`) {
		t.Fatal("triage page must have a 'Details bearbeiten' section")
	}

	// Verify form fields for body, location_type, location_detail
	for _, field := range []string{
		`name="body"`,
		`name="location_type"`,
		`name="location_detail"`,
		`name="update_details" value="1"`,
		`action="/demo/app/anliegen/workflow"`,
		`maxlength="4000"`,
		`maxlength="160"`,
		`placeholder="Zum Beispiel: Heizungsraum"`,
	} {
		if !strings.Contains(triageBody, field) {
			t.Fatalf("triage page edit form missing %q", field)
		}
	}

	// Verify current values are populated
	if !strings.Contains(triageBody, "Die Heizung bleibt kalt.") {
		t.Fatal("edit form must show current body value")
	}
	if !strings.Contains(triageBody, "Stiegenhaus 2. Stock") {
		t.Fatal("edit form must show current location_detail value")
	}

	// Verify the copy emphasizes place-only for location_detail
	if !strings.Contains(triageBody, "Nur Ort, weitere Details in die Beschreibung") {
		t.Fatal("location_detail hint must clarify it's place-only")
	}
}

// HAUSV-568: Validate body max 4000 and location_detail max 160 characters.
func TestIssueDetailsUpdateValidation(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		locationDetail string
		wantCode       int
		wantUpdated    bool
	}{
		{
			name:           "valid update within limits",
			body:           "Updated body within limit",
			locationDetail: "Updated detail",
			wantCode:       http.StatusSeeOther,
			wantUpdated:    true,
		},
		{
			name:           "body at 4000 chars is valid",
			body:           strings.Repeat("a", 4000),
			locationDetail: "Detail",
			wantCode:       http.StatusSeeOther,
			wantUpdated:    true,
		},
		{
			name:           "body over 4000 chars is rejected",
			body:           strings.Repeat("a", 4001),
			locationDetail: "Detail",
			wantCode:       http.StatusSeeOther, // Redirects to error
			wantUpdated:    false,
		},
		{
			name:           "location_detail at 160 chars is valid",
			body:           "Body",
			locationDetail: strings.Repeat("a", 160),
			wantCode:       http.StatusSeeOther,
			wantUpdated:    true,
		},
		{
			name:           "location_detail over 160 chars is rejected",
			body:           "Body",
			locationDetail: strings.Repeat("a", 161),
			wantCode:       http.StatusSeeOther,
			wantUpdated:    false,
		},
		{
			name:           "empty body is rejected",
			body:           "",
			locationDetail: "Detail",
			wantCode:       http.StatusSeeOther,
			wantUpdated:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh app and issue for each test case to avoid interference
			a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
			issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
				TenantSlug:     "demo",
				AuthorEmail:    "resident@example.com",
				AuthorName:     "Resident",
				Category:       "Reparatur",
				Title:          "Test Issue",
				Body:           "Original body",
				LocationType:   issueLocationCommon,
				LocationDetail: "Original detail",
				Status:         issueStatusNew,
				Priority:       issuePriorityNorm,
			})
			if err != nil {
				t.Fatalf("create issue: %v", err)
			}

			form := url.Values{
				"id":              {issue.ID},
				"status":          {issue.Status},
				"priority":        {issue.Priority},
				"assignee_email":  {""},
				"update_details":  {"1"},
				"body":            {tt.body},
				"location_type":   {string(issueLocationCommon)},
				"location_detail": {tt.locationDetail},
			}

			req := authedFormRequest(t, a, "manager@example.com", "/demo/app/anliegen/workflow", form)
			if req.Code != tt.wantCode {
				t.Fatalf("%s: got status %d, want %d", tt.name, req.Code, tt.wantCode)
			}

			// Verify the issue was (or wasn't) updated
			updated, found := issueRepositoryForTest(a, "demo").Get(issue.ID)
			if !found {
				t.Fatalf("%s: issue not found", tt.name)
			}

			if tt.wantUpdated {
				if updated.Body != tt.body {
					t.Errorf("%s: body not updated, got %q, want %q", tt.name, updated.Body, tt.body)
				}
				if updated.LocationDetail != tt.locationDetail {
					t.Errorf("%s: location_detail not updated, got %q, want %q", tt.name, updated.LocationDetail, tt.locationDetail)
				}
			} else {
				if updated.Body != "Original body" {
					t.Errorf("%s: body should not have been updated, got %q", tt.name, updated.Body)
				}
				if updated.LocationDetail != "Original detail" {
					t.Errorf("%s: location_detail should not have been updated, got %q", tt.name, updated.LocationDetail)
				}
			}
		})
	}
}

// HAUSV-568: Non-managers must not be able to edit issue details via the
// workflow endpoint.
func TestResidentCannotEditIssueDetails(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:     "demo",
		AuthorEmail:    "resident@example.com",
		AuthorName:     "Resident",
		Category:       "Reparatur",
		Title:          "Test Issue",
		Body:           "Original body",
		LocationType:   issueLocationCommon,
		LocationDetail: "Original detail",
		Status:         issueStatusNew,
		Priority:       issuePriorityNorm,
	})
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}

	form := url.Values{
		"id":              {issue.ID},
		"status":          {issue.Status},
		"priority":        {issue.Priority},
		"assignee_email":  {""},
		"update_details":  {"1"},
		"body":            {"Hacked body"},
		"location_type":   {string(issueLocationCommon)},
		"location_detail": {"Hacked detail"},
	}

	req := authedFormRequest(t, a, "resident@example.com", "/demo/app/anliegen/workflow", form)
	if req.Code != http.StatusForbidden {
		t.Fatalf("resident detail update should return 403, got %d", req.Code)
	}

	// Verify the issue was NOT updated
	unchanged, found := issueRepositoryForTest(a, "demo").Get(issue.ID)
	if !found {
		t.Fatal("issue not found")
	}
	if unchanged.Body != "Original body" {
		t.Errorf("resident must not be able to update body, got %q", unchanged.Body)
	}
	if unchanged.LocationDetail != "Original detail" {
		t.Errorf("resident must not be able to update location_detail, got %q", unchanged.LocationDetail)
	}
}

// HAUSV-568: The create form and edit form must use "Wo genau?" with
// place-only copy and a hint that further detail goes in Beschreibung.
func TestIssueLocationDetailCopyIsPlaceOnly(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:     "demo",
		AuthorEmail:    "resident@example.com",
		AuthorName:     "Resident",
		Category:       "Reparatur",
		Title:          "Test",
		Body:           "Body",
		LocationType:   issueLocationCommon,
		LocationDetail: "Detail",
		Status:         issueStatusNew,
		Priority:       issuePriorityNorm,
	})
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}

	triageBody := authedRequest(t, a, "manager@example.com", "/demo/app/anliegen/board/"+issue.ID).Body.String()

	// Verify placeholder example
	if !strings.Contains(triageBody, `placeholder="Zum Beispiel: Heizungsraum"`) {
		t.Fatal("location_detail must have a place-only placeholder example")
	}

	// Verify hint clarifying it's place-only
	if !strings.Contains(triageBody, "Nur Ort, weitere Details in die Beschreibung") {
		t.Fatal("location_detail must have a hint clarifying it's place-only")
	}

	// Verify max length hints
	if !strings.Contains(triageBody, "Max. 4000 Zeichen") {
		t.Fatal("body field must show max 4000 chars hint")
	}
	if !strings.Contains(triageBody, "Max. 160 Zeichen") {
		t.Fatal("location_detail field must show max 160 chars hint")
	}
}

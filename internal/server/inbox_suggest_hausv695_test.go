package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestHausv695PrefixAllHTMXMethods(t *testing.T) {
	for _, attribute := range []string{"hx-get", "hx-post", "hx-put", "hx-patch", "hx-delete"} {
		t.Run(attribute, func(t *testing.T) {
			for _, path := range []string{"/app/test?status=open&amp;sort=age", "/annenstrasse-71/app/test", "https://example.com/test", "relative", "#target"} {
				body := fmt.Sprintf(`<div %s="%s"></div>`, attribute, path)
				want := body
				if strings.HasPrefix(path, "/app/") {
					want = fmt.Sprintf(`<div %s="/annenstrasse-71%s"></div>`, attribute, path)
				}
				got := prefixTenantHTMLPaths(body, "annenstrasse-71")
				if got != want || prefixTenantHTMLPaths(got, "annenstrasse-71") != want {
					t.Fatalf("prefix %s: got %q, want %q (including repeated rendering)", attribute, got, want)
				}
				if prefixTenantHTMLPaths(body, "") != body {
					t.Fatal("empty tenant changed URL")
				}
			}
		})
	}
}

type hausv695IntakeError struct{ store.IntakeRepository }

func (hausv695IntakeError) Get(context.Context, string) (store.IntakeItem, error) {
	return store.IntakeItem{}, errors.New("private repository diagnostic")
}

type hausv695TemplatesError struct{ store.TextbausteinRepository }

func (hausv695TemplatesError) List(context.Context) ([]store.Textbaustein, error) {
	return nil, errors.New("private repository diagnostic")
}

func TestHausv695SuggestionPartialErrorsKeepRecoverableHTML(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		status int
	}{
		{"unavailable", http.StatusServiceUnavailable},
		{"missing organisation", http.StatusServiceUnavailable},
		{"not found", http.StatusNotFound},
		{"repository error", http.StatusInternalServerError},
		{"view error", http.StatusInternalServerError},
		{"forbidden", http.StatusForbidden},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			a, intake, _ := newInboxTestApp(t, roleManager)
			seedInboxItem(t, intake, "poll", store.IntakeStatusOpen, nil)
			id := "poll"
			switch scenario.name {
			case "unavailable":
				a.intake = nil
			case "missing organisation":
				tenant := a.tenants["demo"]
				tenant.Organisation = ""
				a.tenants["demo"] = tenant
			case "not found":
				id = "missing"
			case "repository error":
				a.intake = func(string) store.IntakeRepository { return hausv695IntakeError{intake} }
			case "view error":
				a.textbausteine = func(string) store.TextbausteinRepository { return hausv695TemplatesError{} }
			case "forbidden":
				if err := intake.Assign(t.Context(), id, "other-house", "Top 1"); err != nil {
					t.Fatal(err)
				}
			}
			response := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang/"+id+"/vorschlag?status=open")
			if response.Code != scenario.status || response.Header().Get("Location") != "" || response.Header().Get("Content-Type") != "text/html; charset=utf-8" {
				t.Fatalf("status=%d headers=%v", response.Code, response.Header())
			}
			body := response.Body.String()
			for _, want := range []string{
				`id="vorschlag"`, `hx-trigger="inbox:suggestion-retry"`,
				`hx-get="/demo/app/verwaltung/posteingang/` + id + `/vorschlag?status=open"`,
				"Vorschlagsstatus konnte nicht geladen werden", "Erneut versuchen", "im Hintergrund weiter", "Abbrechen",
			} {
				if !strings.Contains(body, want) {
					t.Fatalf("error partial missing %q: %s", want, body)
				}
			}
			for _, unwanted := range []string{"private repository diagnostic", "other-house", `hx-trigger="every`, `id="case-workflow"`, "<!doctype", "<html"} {
				if strings.Contains(body, unwanted) {
					t.Fatalf("error partial contains %q", unwanted)
				}
			}
		})
	}
}

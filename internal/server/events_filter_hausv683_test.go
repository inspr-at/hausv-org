package server

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestEventsSearchAndTimeSegmentsHAUSV683(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	now := time.Now()
	for _, item := range []houseEvent{
		{TenantSlug: "demo", Title: "Künftige Prüfung", Category: "Wartung", Body: "Öl & Filter", Location: "Technikraum", StartsAt: now.Add(24 * time.Hour)},
		{TenantSlug: "demo", Title: "Frühere Prüfung", Category: "Wartung", Body: "Öl & Filter", StartsAt: now.Add(-48 * time.Hour)},
		{TenantSlug: "demo", Title: "Hausversammlung", Category: "Sonstiges", StartsAt: now.Add(48 * time.Hour)},
	} {
		if _, err := testRepositories(a, "demo").events.Create(item); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		query        string
		want, absent []string
	}{
		{"?status=current", []string{"Künftige Prüfung", "Hausversammlung"}, []string{"Frühere Prüfung"}},
		{"?status=past", []string{"Frühere Prüfung", "Abgelaufene Termine"}, []string{"Künftige Prüfung", "Noch keine kommenden Termine"}},
		{"?status=all&q=%C3%96L+%26+FILTER", []string{"Künftige Prüfung", "Frühere Prüfung", `status=current`, `q=%C3%96L+%26+FILTER`}, []string{"Hausversammlung"}},
		{"?status=current&q=technikraum", []string{"Künftige Prüfung"}, []string{"Hausversammlung", "Frühere Prüfung"}},
		{"?status=past&q=kein-treffer", []string{"Keine passenden Termine"}, []string{"Künftige Prüfung", "Frühere Prüfung"}},
		{"?status=invalid", []string{"Künftige Prüfung", "Frühere Prüfung", "Hausversammlung"}, nil},
	} {
		t.Run(tc.query, func(t *testing.T) {
			page := authedRequest(t, a, "resident@example.com", "/demo/app/events"+tc.query)
			if page.Code != http.StatusOK {
				t.Fatalf("status = %d", page.Code)
			}
			body := page.Body.String()
			for _, want := range tc.want {
				if !strings.Contains(body, want) {
					t.Errorf("missing %q", want)
				}
			}
			for _, absent := range tc.absent {
				if strings.Contains(body, absent) {
					t.Errorf("unexpected %q", absent)
				}
			}
			if tc.query == "?status=all&q=%C3%96L+%26+FILTER" {
				for _, count := range []string{`>Aktuell<span>1</span>`, `>Abgelaufen<span>1</span>`, `>Alle<span>2</span>`} {
					if !strings.Contains(body, count) {
						t.Errorf("missing segment count %q", count)
					}
				}
			}
		})
	}
}

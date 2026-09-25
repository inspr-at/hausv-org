package server

import (
	"crypto/sha256"
	"fmt"
	appmail "github.com/inspr-at/hausv-org/internal/mail"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestValorisationRoutesAndDenials(t *testing.T) {
	a, _, _ := newArchiveDemoApp(t)
	a.leaseStore = store.NewSQLLeaseStore(a.tenantDB)
	a.profiles[archiveDemoManager] = userProfile{Email: archiveDemoManager, Role: roleAdmin, Tenants: []string{archiveDemoTenant}, AuthMethods: defaultAuthMethods()}
	path := "/app/settings/valorisation"
	page := archiveDemoRequest(t, a, archiveDemoManager, "GET", path, nil)
	if page.Code != 200 {
		t.Fatalf("page %d %s", page.Code, page.Body.String())
	}
	for _, want := range []string{`href="/` + archiveDemoTenant + `/app/hilfe#recht-wirksamwerden"`, "Wertsicherung", "Bereit", "Unverändert", "Ausnahmen", "1.040,28", "1.017,35", "21.04.2026", "05.05.2026"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Errorf("missing %s", want)
		}
	}
	docs := a.documentStore.(*store.SQLDocumentStore)
	repo, _ := store.BindValorisationRepository(a.tenantDB, docs, a.tenantIdentities[archiveDemoTenant].Ref())
	runs, err := repo.List()
	if err != nil {
		t.Fatal(err)
	}
	run := runs[0]
	blocked := archiveDemoRequest(t, a, archiveDemoManager, "POST", path+"/runs/"+run.ID+"/approve", nil)
	if blocked.Code != http.StatusSeeOther {
		t.Fatalf("blocked approval must redirect: %d", blocked.Code)
	}
	location := strings.TrimPrefix(blocked.Header().Get("Location"), "/"+archiveDemoTenant)
	location, _, _ = strings.Cut(location, "#") // Browsers never send fragments to the server.
	notice := archiveDemoRequest(t, a, archiveDemoManager, "GET", location, nil)
	for _, want := range []string{"Wertsicherung", "Ausnahmen zuerst bearbeiten", "Keine Wertsicherungsklausel vorhanden.", "3 Ausnahmen sind noch offen.", `class="button ghost"`, `disabled`, `role="status"`} {
		if notice.Code != http.StatusOK || !strings.Contains(notice.Body.String(), want) {
			t.Errorf("blocked approval notice missing %q: HTTP %d", want, notice.Code)
		}
	}
	if strings.Contains(notice.Body.String(), "clause_unreviewed") {
		t.Fatal("raw exception code in notice")
	}
	unchanged, _, err := repo.Get(run.ID)
	if err != nil || unchanged.Status != "draft" {
		t.Fatal("blocked approval changed the run", err)
	}
	for _, role := range []string{roleOwner, roleResident, roleServiceProvider, roleBeirat} {
		actor := strings.ToLower(role) + "@example.com"
		a.profiles[actor] = userProfile{Email: actor, Role: role, Tenants: []string{archiveDemoTenant}, AuthMethods: defaultAuthMethods()}
		for _, request := range []struct{ method, path string }{{"GET", path}, {"POST", path + "/runs"}, {"POST", path + "/runs/" + run.ID + "/approve"}, {"POST", path + "/runs/" + run.ID + "/send"}, {"POST", path + "/runs/" + run.ID + "/deliveries/test/receipt"}, {"GET", path + "/runs/" + run.ID + "/items/" + run.Items[0].ID + "/pdf"}} {
			denied := archiveDemoRequest(t, a, actor, request.method, request.path, url.Values{"effective_on": {"2026-04-01"}})
			if denied.Code != 403 {
				t.Fatalf("%s %s %s = %d", role, request.method, request.path, denied.Code)
			}
		}
	}
	manager := "manager@example.com"
	a.profiles[manager] = userProfile{Email: manager, Role: roleManager, Tenants: []string{archiveDemoTenant}, AuthMethods: defaultAuthMethods()}
	denied := archiveDemoRequest(t, a, manager, "POST", path+"/runs/"+run.ID+"/approve", nil)
	if denied.Code != 403 {
		t.Fatalf("manager approved: %d", denied.Code)
	}
	for _, item := range run.Items {
		if item.Group == "exception" {
			res := archiveDemoRequest(t, a, archiveDemoManager, "POST", path+"/runs/"+run.ID+"/items/"+item.ID, url.Values{"action": {"exclude"}, "reason": {"Gesonderte Vertragsprüfung"}})
			if res.Code != 303 {
				t.Fatal("exclude", res.Code, res.Body.String())
			}
		}
	}
	approved := archiveDemoRequest(t, a, archiveDemoManager, "POST", path+"/runs/"+run.ID+"/approve", nil)
	if approved.Code != 303 {
		t.Fatal("approve", approved.Code, approved.Body.String())
	}
	run, _, err = repo.Get(run.ID)
	if err != nil || run.Status != "approved" {
		t.Fatal("run not approved", err)
	}
	for _, item := range run.Items {
		if item.LetterDocumentID == "" {
			continue
		}
		response := archiveDemoRequest(t, a, archiveDemoManager, "GET", path+"/runs/"+run.ID+"/items/"+item.ID+"/pdf", nil)
		if response.Code != http.StatusOK || !strings.HasPrefix(response.Body.String(), "%PDF-") || strings.Contains(response.Body.String(), "Entwurf") {
			t.Fatal("final PDF", response.Code)
		}
	}
	lease := archiveDemoRequest(t, a, archiveDemoManager, "GET", "/app/settings/building/units/top-1/lease", nil)
	if lease.Code != 200 || !strings.Contains(lease.Body.String(), "Freigegeben") || !strings.Contains(lease.Body.String(), "Anpassungsschreiben PDF") {
		t.Fatal("history", lease.Code)
	}
	outbox := t.TempDir()
	sink := appmail.NewSMTP("", "", "", "", "verwaltung@example.com").WithOutbox(outbox)
	failure := &failingDocumentMailer{Mailer: sink}
	a.mailer = failure
	sendPath := path + "/runs/" + run.ID + "/send"
	if failed := archiveDemoRequest(t, a, archiveDemoManager, "POST", sendPath, nil); failed.Code != 303 || failure.calls == 0 {
		t.Fatal("failed send", failed.Code, failure.calls)
	}
	stillApproved, _, _ := repo.Get(run.ID)
	if stillApproved.Status != "approved" {
		t.Fatal("failed mail completed run")
	}
	a.mailer = sink
	if sent := archiveDemoRequest(t, a, archiveDemoManager, "POST", sendPath, nil); sent.Code != 303 {
		t.Fatal("send", sent.Code, sent.Body.String())
	}
	deliveryRepo, _ := store.BindValorisationDeliveryRepository(store.NewSQLValorisationDeliveryStore(a.tenantDB), a.tenantIdentities[archiveDemoTenant].Ref())
	rows, err := deliveryRepo.List(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	sentCount := 0
	for _, row := range rows {
		if row.Status != "sent" {
			continue
		}
		sentCount++
		doc, found := a.repositoriesForTenant(a.tenantIdentities[archiveDemoTenant].Ref()).documents.Get(row.DocumentID)
		if !found || doc.ValorisationArchive.SHA256 != row.SHA256 {
			t.Fatal("delivery/archive mismatch", row)
		}
		pdf, err := readValorisationDocument(a.repositoriesForTenant(a.tenantIdentities[archiveDemoTenant].Ref()).documents, doc)
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(pdf)) != row.SHA256 {
			t.Fatal("attachment hash", err)
		}
	}
	entries, err := os.ReadDir(outbox)
	if err != nil || len(entries) != sentCount || sentCount == 0 {
		t.Fatal("outbox", len(entries), sentCount, err)
	}
	if again := archiveDemoRequest(t, a, archiveDemoManager, "POST", sendPath, nil); again.Code != 303 {
		t.Fatal(again.Code)
	}
	after, _ := os.ReadDir(outbox)
	if len(after) != len(entries) {
		t.Fatal("duplicate mail")
	}

}

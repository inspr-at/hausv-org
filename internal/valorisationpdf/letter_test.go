package valorisationpdf

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/indexation"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestLetterVariants(t *testing.T) {
	tool, err := exec.LookPath("pdftotext")
	if err != nil {
		t.Skip("pdftotext required for PDF text extraction")
	}
	snapshot, err := indexation.LoadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	when, _ := time.Parse(time.DateOnly, "2026-04-01")
	for _, variant := range []string{"mieweg-full", "mieweg-partial", "contract-full", "contract-exempt", "decrease"} {
		t.Run(variant, func(t *testing.T) {
			lease := store.Lease{ID: "e1", UnitID: "Top 1", Status: store.LeaseStatusActive, ConcludedOn: "2024-09-01", StartsOn: "2024-09-01", LeaseKind: store.LeaseKindHauptmiete, UseKind: store.UseKindWohnung, MRGScope: store.MRGTeil, PriceRestrictedSet: true, ZinsterminDay: 5,
				Parties:    []store.LeaseParty{{ID: "eva", Name: "Eva Huber", Address: "Janusbergweg 123/1\n8010 Graz", Email: "eva.huber@musterstadt.example", Role: store.PartyHauptmieter, ValidFrom: "2024-09-01"}},
				Components: []store.RentComponent{{Kind: store.ComponentHMZ, NetCents: 100000, VATRateBP: 1000, ValidFrom: "2024-09-01"}, {Kind: store.ComponentBKAkonto, NetCents: 18000, VATRateBP: 1000, ValidFrom: "2024-09-01"}},
				Clauses:    []store.IndexClause{{ID: "clause", ComponentKind: store.ComponentHMZ, ClauseType: store.ClauseVPIThreshold, Series: "vpi2020", BasePeriod: "2024-09", BaseValue: "123.6", ThresholdKind: "percent", ThresholdValue: "5", TwoWay: true, FullChangeOnTrigger: true, ReviewStatus: store.ReviewOK, ValidFrom: "2024-09-01", ClauseText: "Der Hauptmietzins ist nach dem VPI 2020 wertgesichert. Die Schwelle von 5 Prozent gilt in beide Richtungen; nach Überschreitung wird die volle Änderung berücksichtigt."}},
			}
			switch variant {
			case "mieweg-full":
				lease.MRGScope = store.MRGVoll
				lease.PriceRestricted = true
			case "contract-full":
				lease.UseKind = store.UseKindGeschaeft
				lease.MRGScope = store.MRGVoll
			case "contract-exempt":
				lease.MRGScope = store.MRGAusnahme
			}
			run, err := store.PreviewValorisation(store.ValorisationInput{EffectiveOn: "2026-04-01", Leases: []store.Lease{lease}, House: "Janusbergweg 123", Address: "Janusbergweg 123, 8010 Graz", Organisation: "Hausverwaltung Musterstadt", Contact: "Vera Verwalter · vera.verwalter@musterstadt.example"}, snapshot, when)
			if err != nil {
				t.Fatal(err)
			}
			run.ID = "demo-valorisation-2026"
			run.Revision = 1
			run.Status = "approved"
			run.ApprovedBy = "vera.verwalter@musterstadt.example"
			item := run.Items[0]
			item.ID = "item-top-1"
			if item.Group != "ready" {
				t.Fatal(item.Exceptions)
			}
			if variant == "decrease" {
				item.NewCents = 95000
				item.NewVATCents = 9500
				item.NewGrossCents = 104500
				item.Outcome = "decrease"
				item.RequiresMRGNotice = false
				item.CollectableFrom = "2026-04-05"
				value := item.NewCents
				item.OverrideCents = &value
				item.OverrideReason = "Verminderung nach gesonderter Prüfung"
				item.OverrideBy = run.ApprovedBy
			}
			raw, err := Render(run, item, when)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			path := filepath.Join(dir, "letter.pdf")
			if err = os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			out, err := exec.Command(tool, "-layout", path, "-").Output()
			if err != nil {
				t.Fatal(err)
			}
			text := string(out)
			for _, want := range []string{"Eva Huber", "8010 Graz", "1.000,00 €", Money(item.NewCents), "Hauptmietzins", "BK-Akonto", "unverändert", "01.04.2026", "Seite 1 von", "Statistik Austria"} {
				if !strings.Contains(text, want) {
					t.Errorf("missing %q", want)
				}
			}
			if strings.Contains(text, "Entwurf") {
				t.Fatal("approved letter is draft")
			}
			if item.RequiresMRGNotice && !strings.Contains(text, "§ 16 Abs 9 MRG") {
				t.Fatal("MRG notice missing")
			}
			if item.MieWeG && !strings.Contains(text, "niedrigere Betrag") {
				t.Fatal("parallel decision missing")
			}
			if variant == "mieweg-full" && !strings.Contains(text, "Sonderdeckel") {
				t.Fatal("special cap missing")
			}
			if variant == "decrease" && (!strings.Contains(text, "Verminderung") || !strings.Contains(text, "Manuelle Entscheidung")) {
				t.Fatal("decrease / override missing")
			}
			if dir := os.Getenv("HAUSV_VALORISATION_PDF_DIR"); dir != "" {
				if err = os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(dir, variant+".pdf"), raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			run.Status = "draft"
			draft, err := Render(run, item, when)
			if err != nil || !strings.Contains(string(draft), "Entwurf") {
				t.Fatal("draft missing")
			}
		})
	}
}

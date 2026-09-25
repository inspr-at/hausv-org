package valorisationpdf

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	for _, variant := range []string{"mieweg-full", "mieweg-partial", "contract-full", "contract-exempt", "decrease", "staffel-mieweg-full", "staffel-mieweg-partial", "staffel-contract-full", "staffel-contract-exempt", "staffel-decrease"} {
		t.Run(variant, func(t *testing.T) {
			kind := strings.TrimPrefix(variant, "staffel-")
			lease := store.Lease{ID: "e1", UnitID: "Top 1", Status: store.LeaseStatusActive, ConcludedOn: "2024-09-01", StartsOn: "2024-09-01", LeaseKind: store.LeaseKindHauptmiete, UseKind: store.UseKindWohnung, MRGScope: store.MRGTeil, PriceRestrictedSet: true, ZinsterminDay: 5,
				Parties:    []store.LeaseParty{{ID: "eva", Name: "Eva Huber", Address: "Janusbergweg 123/1\n8010 Graz", Email: "eva.huber@musterstadt.example", Role: store.PartyHauptmieter, ValidFrom: "2024-09-01"}},
				Components: []store.RentComponent{{Kind: store.ComponentHMZ, NetCents: 100000, VATRateBP: 1000, ValidFrom: "2024-09-01"}, {Kind: store.ComponentBKAkonto, NetCents: 18000, VATRateBP: 1000, ValidFrom: "2024-09-01"}},
				Clauses:    []store.IndexClause{{ID: "clause", ComponentKind: store.ComponentHMZ, ClauseType: store.ClauseVPIThreshold, Series: "vpi2020", BasePeriod: "2024-09", BaseValue: "123.6", ThresholdKind: "percent", ThresholdValue: "5", TwoWay: true, FullChangeOnTrigger: true, ReviewStatus: store.ReviewOK, ValidFrom: "2024-09-01", ClauseText: "Der Hauptmietzins ist nach dem VPI 2020 wertgesichert. Die Schwelle von 5 Prozent gilt in beide Richtungen; nach Überschreitung wird die volle Änderung berücksichtigt."}},
			}
			if strings.HasPrefix(variant, "staffel-") {
				amount := int64(110000)
				lease.Clauses[0] = store.IndexClause{ID: "clause", ComponentKind: store.ComponentHMZ, ClauseType: store.ClauseStaffel, TwoWay: true, ReviewStatus: store.ReviewOK, ValidFrom: lease.StartsOn, StaffelSteps: []indexation.StaffelStep{{EffectiveOn: "2026-04-01", NetCents: &amount}}}
			}
			switch kind {
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
			if kind == "decrease" {
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
			for _, internal := range []string{"Lauf", "Referenz", "SHA-256", run.ID[:16], run.InputsSHA256[:16], run.IndexVersion} {
				if strings.Contains(text, internal) {
					t.Fatalf("internal audit data %q in tenant letter", internal)
				}
			}
			if strings.HasPrefix(variant, "staffel-") && !strings.Contains(text, "Staffelmietzins laut Vertrag") {
				t.Fatal("Staffel letter basis missing", text)
			}
			if regexp.MustCompile(`[0-9]+/[0-9]+ %|[0-9]+\.[0-9]+ %|[0-9]+,[0-9]{3,} %|percent|Kurve exakt|top-1|[0-9]{4}-[0-9]{2}`).MatchString(text) {
				t.Fatal("technical notation in letter", text)
			}
			if !strings.Contains(text, "Top 1 · Eva Huber") {
				t.Fatal("unit and tenant label missing", text)
			}
			for _, want := range []string{"Eva Huber", "8010 Graz", "1.000,00 €", Money(item.NewCents), "Hauptmietzins", "BK-Akonto", "unverändert", "01.04.2026", "Seite 1 von", "Indexwerte laut Statistik Austria, Stand 24.09.2026", "Guten Tag Eva Huber,", "Für Rückfragen:", "Mit freundlichen Grüßen"} {
				if !strings.Contains(text, want) {
					t.Errorf("missing %q", want)
				}
			}
			if strings.Contains(text, "Wertsicherung · Janusbergweg 123 · Janusbergweg 123") {
				t.Fatal("house repeated in subtitle")
			}
			if !strings.HasPrefix(variant, "staffel-") {
				for _, want := range []string{"VPI 2020, Basis September 2024: 123,6", "Auslösemonat Dezember 2025", "5,02 %"} {
					if !strings.Contains(text, want) {
						t.Errorf("letter formatting missing %q", want)
					}
				}
			}
			if strings.Contains(text, "Entwurf") {
				t.Fatal("approved letter is draft")
			}
			if item.RequiresMRGNotice && !strings.Contains(text, "§ 16 Abs 9 MRG") {
				t.Fatal("MRG notice missing")
			}
			if item.RequiresMRGNotice && (!strings.Contains(text, Date(item.NoticeDeadline)) || !strings.Contains(text, "zugeht") || !strings.Contains(text, "14 Tage")) {
				t.Fatal("receipt deadline missing")
			}
			letterText := strings.Join(strings.Fields(text), " ")
			noBackPayment := "Der höhere Hauptmietzins ist erst ab diesem Zinstermin zu bezahlen; für die davorliegenden Monate wird keine Nachzahlung verlangt."
			if strings.Contains(letterText, noBackPayment) != (item.RequiresMRGNotice && item.NewCents > item.OldCents && item.CollectableFrom > item.WirksamOn) {
				t.Fatal("no-back-payment notice does not match MRG timing")
			}
			if item.MieWeG && !strings.Contains(text, "niedrigere Betrag") {
				t.Fatal("parallel decision missing")
			}
			if kind == "mieweg-full" && !strings.Contains(text, "Sonderdeckel") {
				t.Fatal("special cap missing")
			}
			if kind == "decrease" && (!strings.Contains(text, "Verminderung") || !strings.Contains(text, "Manuelle Entscheidung")) {
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

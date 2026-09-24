package demo

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/pdf"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/textutil"
	"github.com/inspr-at/hausv-org/internal/view"
)

type seedStatement struct {
	Legal      store.AnnualStatementLegalSettings `json:"legal"`
	House      string                             `json:"house"`
	Year       int                                `json:"year"`
	StartsOn   string                             `json:"starts_on"`
	EndsOn     string                             `json:"ends_on"`
	RecordedAt time.Time                          `json:"recorded_at"`
	CostTypes  []seedStatementCost                `json:"cost_types"`
	UnitBases  []seedStatementBasis               `json:"unit_bases"`
}
type seedStatementCost struct {
	Supplier             string `json:"supplier"`
	HeatingCategory      string `json:"heating_category"`
	OperatingAmountCents int64  `json:"operating_amount_cents"`
	Key                  string `json:"key"`
	Name                 string `json:"name"`
	AllocationKey        string `json:"allocation_key"`
	AmountCents          int64  `json:"amount_cents"`
	InvoiceDate          string `json:"invoice_date"`
}
type seedStatementBasis struct {
	HeatingConsumptionKWh int64  `json:"heating_consumption_kwh"`
	UnitID                string `json:"unit_id"`
	PPM                   int    `json:"miteigentumsanteil_ppm"`
	Area                  int    `json:"usable_area_m2_hundredths"`
	Persons               int    `json:"persons"`
	PrepaidCents          int64  `json:"prepaid_cents"`
}
type seedPerson struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

func loadStatementFixture(dir string, houses []seedHouse, documentDir string) (*seedStatement, error) {
	path := filepath.Join(dir, "annual-statement.json")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	if strings.TrimSpace(documentDir) == "" {
		return nil, fmt.Errorf("demo statement fixture requires DocumentDir")
	}
	var statement seedStatement
	if err := readJSON(path, &statement); err != nil {
		return nil, err
	}
	if statement.Year != 2025 || statement.StartsOn != "2025-01-01" || statement.EndsOn != "2025-12-31" || statement.RecordedAt.IsZero() || len(statement.CostTypes) == 0 {
		return nil, fmt.Errorf("invalid demo statement period")
	}
	costKeys := map[string]bool{}
	for _, cost := range statement.CostTypes {
		if costKeys[cost.Key] || cost.Key == "" || cost.Key != textutil.Slug(cost.Key) || cost.AmountCents <= 0 || cost.Name == "" {
			return nil, fmt.Errorf("invalid demo cost type %s", cost.Key)
		}
		costKeys[cost.Key] = true
		if cost.AllocationKey != store.AllocationKeyNutzwert && cost.AllocationKey != store.AllocationKeyFlaeche && cost.AllocationKey != store.AllocationKeyPersonen && cost.AllocationKey != store.AllocationKeyVerbrauch {
			return nil, fmt.Errorf("unsupported demo allocation key %s", cost.AllocationKey)
		}
		if _, err := time.Parse("2006-01-02", cost.InvoiceDate); err != nil {
			return nil, err
		}
	}
	var persons []seedPerson
	if err := readJSON(filepath.Join(dir, "persons.json"), &persons); err != nil {
		return nil, err
	}
	contacts := map[string]store.UnitPartyContact{}
	for _, person := range persons {
		contacts[person.Email] = store.UnitPartyContact{Email: person.Email, Name: person.Name, Address: person.Address}
	}
	bases := map[string]seedStatementBasis{}
	total := 0
	for _, basis := range statement.UnitBases {
		if _, found := bases[basis.UnitID]; found || basis.UnitID == "" || basis.PPM < 0 || basis.Area < 0 || basis.Persons < 0 || basis.PrepaidCents < 0 {
			return nil, fmt.Errorf("invalid demo unit basis %s", basis.UnitID)
		}
		total += basis.PPM
		bases[basis.UnitID] = basis
	}
	if total != 1_000_000 {
		return nil, fmt.Errorf("demo PPM sum must be 1000000")
	}
	found := false
	for hi := range houses {
		house := &houses[hi]
		if house.Slug != statement.House {
			continue
		}
		found = true
		if len(house.Units) != len(bases) {
			return nil, fmt.Errorf("demo statement must cover every unit")
		}
		for ui := range house.Units {
			unit := &house.Units[ui]
			basis, ok := bases[store.NormalizeUnitID(unit.Label)]
			if !ok {
				return nil, fmt.Errorf("demo basis missing for %s", unit.Label)
			}
			unit.StatementBasis = &basis
			if unit.OwnerEmail == "" {
				return nil, fmt.Errorf("demo owner missing for %s", unit.Label)
			}
			for _, email := range []string{unit.OwnerEmail, unit.TenantEmail} {
				if email == "" {
					continue
				}
				contact, ok := contacts[email]
				if !ok {
					return nil, fmt.Errorf("demo party %s absent from persons.json", email)
				}
				unit.PartyContacts = append(unit.PartyContacts, contact)
			}
		}
	}
	if !found {
		return nil, fmt.Errorf("demo statement house not found")
	}
	return &statement, nil
}

func seedAnnualStatement(ctx context.Context, database *sql.DB, statement *seedStatement, identity store.TenantIdentity, documentDir string, reset bool) error {
	if statement == nil {
		return nil
	}
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
	// The demo loader owns a maintenance transaction, like upsertJSON in seed.go.
	// Insert missing fixture rows, then restore their values and tenant identity
	// explicitly, including existing rows with a missing tenant identity.
	year, at, actor := statement.Year, statement.RecordedAt.Format(time.RFC3339Nano), "vera.verwalter@musterstadt.example"
	if reset {
		// Stored runs are immutable evidence and survive a demo reset. Only the
		// fixture's editable input period is restored; other houses/years stay intact.
		for _, table := range []string{"annual_statement_period_cost_types", "annual_statement_period_unit_bases", "annual_statement_receipts", "annual_statement_prepayments"} {
			if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE tenant_id=$1 AND period_year=$2`, identity.ID, year); err != nil {
				return err
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO annual_statement_periods(tenant_id,tenant_slug,year,starts_on,ends_on,updated_at,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(tenant_slug,year) DO NOTHING`, identity.ID, identity.Slug, year, statement.StartsOn, statement.EndsOn, at, actor); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE annual_statement_periods SET tenant_id=$1,starts_on=$4,ends_on=$5,updated_at=$6,updated_by=$7 WHERE tenant_slug=$2 AND year=$3`, identity.ID, identity.Slug, year, statement.StartsOn, statement.EndsOn, at, actor); err != nil {
		return err
	}
	legal, err := json.Marshal(statement.Legal)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE annual_statement_periods SET legal_settings=$1 WHERE tenant_id=$2 AND year=$3`, string(legal), identity.ID, year); err != nil {
		return err
	}
	for _, cost := range statement.CostTypes {
		if _, err := tx.ExecContext(ctx, `INSERT INTO annual_statement_cost_types(tenant_id,tenant_slug,key,name,allocatable,allocation_key,updated_at,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(tenant_slug,key) DO NOTHING`, identity.ID, identity.Slug, cost.Key, cost.Name, true, cost.AllocationKey, at, actor); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE annual_statement_cost_types SET tenant_id=$1,name=$4,allocatable=$5,allocation_key=$6,updated_at=$7,updated_by=$8 WHERE tenant_slug=$2 AND key=$3`, identity.ID, identity.Slug, cost.Key, cost.Name, true, cost.AllocationKey, at, actor); err != nil {
			return err
		}
		vatRate := store.DefaultAnnualStatementVATPercent(cost.Key)
		if _, err := tx.ExecContext(ctx, `INSERT INTO annual_statement_period_cost_types(tenant_id,tenant_slug,period_year,key,name,allocatable,allocation_key,vat_rate_percent,updated_at,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(tenant_slug,period_year,key) DO NOTHING`, identity.ID, identity.Slug, year, cost.Key, cost.Name, true, cost.AllocationKey, vatRate, at, actor); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE annual_statement_period_cost_types SET tenant_id=$1,name=$5,allocatable=$6,allocation_key=$7,vat_rate_percent=$8,updated_at=$9,updated_by=$10 WHERE tenant_slug=$2 AND period_year=$3 AND key=$4`, identity.ID, identity.Slug, year, cost.Key, cost.Name, true, cost.AllocationKey, vatRate, at, actor); err != nil {
			return err
		}
		for receiptIndex, amount := range []int64{cost.AmountCents, cost.OperatingAmountCents} {
			if amount <= 0 {
				continue
			}
			cost := cost
			cost.AmountCents = amount
			if receiptIndex == 1 {
				cost.HeatingCategory = "sonstige_betriebskosten"
			}
			id := fmt.Sprintf("demo-annual-%d-%s", year, cost.Key)
			if receiptIndex == 1 {
				id += "-betrieb"
			}
			data := pdf.Simple([]string{"Demobeleg · " + cost.Name, "Hausverwaltung Musterstadt", "Janusbergweg 123, 8010 Graz", "Periode: 01.01.2025 bis 31.12.2025", "Rechnungsdatum: " + cost.InvoiceDate, "Betrag: " + view.FormatEURCents(cost.AmountCents), "Synthetischer Demobeleg, keine Zahlungsaufforderung."})
			filename := id + ".pdf"
			if err := os.MkdirAll(filepath.Join(documentDir, identity.Slug), 0700); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(documentDir, identity.Slug, filename), data, 0600); err != nil {
				return err
			}
			doc := store.DocumentRecord{ID: id, SeriesID: id, Version: 1, Current: true, TenantSlug: identity.Slug, Title: "Demobeleg 2025 · " + cost.Name, Category: store.DocumentCategoryBilling, Visibility: store.DocumentVisibilityManagerOnly, Filename: filename, StoredFilename: filename, Size: int64(len(data)), ContentType: "application/pdf", UploadedBy: actor, UploadedAt: statement.RecordedAt}
			if err := upsertJSON(ctx, tx, "documents", identity, id, doc); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO annual_statement_receipts(tenant_id,tenant_slug,id,document_id,period_year,cost_type_key,amount_cents,invoice_date,created_at,created_by,updated_at,updated_by) VALUES($1,$2,$3,$3,$4,$5,$6,$7,$8,$9,$8,$9) ON CONFLICT(tenant_slug,id) DO NOTHING`, identity.ID, identity.Slug, id, year, cost.Key, cost.AmountCents, cost.InvoiceDate, at, actor); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE annual_statement_receipts SET tenant_id=$1,document_id=$3,period_year=$4,amount_cents=$6,cost_type_key=$5,invoice_date=$7,updated_at=$8,updated_by=$9 WHERE tenant_slug=$2 AND id=$3`, identity.ID, identity.Slug, id, year, cost.Key, cost.AmountCents, cost.InvoiceDate, at, actor); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE annual_statement_receipts SET supplier=$1,heating_category=$2 WHERE tenant_id=$3 AND id=$4`, cost.Supplier, cost.HeatingCategory, identity.ID, id); err != nil {
				return err
			}

		}
	}
	for _, basis := range statement.UnitBases {
		if _, err := tx.ExecContext(ctx, `INSERT INTO annual_statement_period_unit_bases(tenant_id,tenant_slug,period_year,unit_id,miteigentumsanteil_ppm,usable_area_m2_hundredths,usable_area_recorded,persons,persons_recorded) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$7) ON CONFLICT(tenant_slug,period_year,unit_id) DO NOTHING`, identity.ID, identity.Slug, year, basis.UnitID, basis.PPM, basis.Area, true, basis.Persons); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE annual_statement_period_unit_bases SET tenant_id=$1,miteigentumsanteil_ppm=$5,usable_area_m2_hundredths=$6,usable_area_recorded=$7,persons=$8,persons_recorded=$7 WHERE tenant_slug=$2 AND period_year=$3 AND unit_id=$4`, identity.ID, identity.Slug, year, basis.UnitID, basis.PPM, basis.Area, true, basis.Persons); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO annual_statement_prepayments(tenant_id,tenant_slug,period_year,unit_id,amount_cents,updated_at,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(tenant_slug,period_year,unit_id) DO NOTHING`, identity.ID, identity.Slug, year, basis.UnitID, basis.PrepaidCents, at, actor); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE annual_statement_prepayments SET tenant_id=$1,amount_cents=$5,updated_at=$6,updated_by=$7 WHERE tenant_slug=$2 AND period_year=$3 AND unit_id=$4`, identity.ID, identity.Slug, year, basis.UnitID, basis.PrepaidCents, at, actor); err != nil {
			return err
		}
	}

	if statement.Legal.HeizKGApplies {
		loc, err := time.LoadLocation("Europe/Vienna")
		if err != nil {
			return err
		}
		start, err := time.ParseInLocation("2006-01-02", statement.StartsOn, loc)
		if err != nil {
			return err
		}
		end, err := time.ParseInLocation("2006-01-02", statement.EndsOn, loc)
		if err != nil {
			return err
		}
		end = end.AddDate(0, 0, 1)
		for _, basis := range statement.UnitBases {
			source := "demo.heizung." + basis.UnitID
			for i, at := range []time.Time{start.UTC(), end.UTC()} {
				digest := sha256.Sum256([]byte("entity\x00" + source + "\x00" + at.Format(time.RFC3339Nano)))
				key := fmt.Sprintf("%x", digest)
				value := int64(100000000000)
				if i == 1 {
					value += basis.HeatingConsumptionKWh * 1000000
				}
				if _, err := tx.ExecContext(ctx, `INSERT INTO annual_statement_consumption_evidence(tenant_id,tenant_slug,source_key,unit_id,cost_type_key,source_kind,source_id,measured_at_ns,value_micros,measurement_unit,received_at_ns) VALUES($1,$2,$3,$4,'heizung','entity',$5,$6,$7,'kWh',$8) ON CONFLICT(tenant_id,source_key) DO NOTHING`, identity.ID, identity.Slug, key, basis.UnitID, source, at.UnixNano(), value, statement.RecordedAt.UnixNano()); err != nil {
					return err
				}
			}
		}
	}
	return tx.Commit()
}

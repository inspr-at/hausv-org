package demo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/indexation"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/textutil"
)

type seedLease struct {
	StaffelSteps         []indexation.StaffelStep `json:"staffel_steps"`
	ID                   string                   `json:"id"`
	House                string                   `json:"house,omitempty"`
	Unit                 string                   `json:"unit"`
	TenantName           string                   `json:"tenant_name"`
	TenantEmail          string                   `json:"tenant_email"`
	TenantAddress        string                   `json:"tenant_address"`
	ConcludedOn          string                   `json:"concluded_on"`
	StartsOn             string                   `json:"starts_on"`
	EndsOn               string                   `json:"ends_on"`
	UseKind              string                   `json:"use_kind"`
	MRGScope             string                   `json:"mrg_scope"`
	RentRegime           string                   `json:"rent_regime"`
	HMZCents             int64                    `json:"hmz_cents"`
	BKCents              int64                    `json:"bk_cents"`
	HeatCents            int64                    `json:"heat_cents"`
	VATBP                int                      `json:"vat_bp"`
	ClauseType           string                   `json:"clause_type"`
	Series               string                   `json:"series"`
	BasePeriod           string                   `json:"base_period"`
	BaseValue            string                   `json:"base_value"`
	Threshold            string                   `json:"threshold"`
	ThresholdKind        string                   `json:"threshold_kind"`
	Inclusive            bool                     `json:"threshold_inclusive"`
	TwoWay               *bool                    `json:"two_way"`
	Review               string                   `json:"review_status"`
	ClauseText           string                   `json:"clause_text"`
	LastMonth            string                   `json:"last_month"`
	LastEffectiveOn      string                   `json:"last_effective_on"`
	LastValue            string                   `json:"last_value"`
	HMZAfter             string                   `json:"hmz_after"`
	Consumer             *bool                    `json:"tenant_is_consumer"`
	PeriodicMonth        int                      `json:"periodic_month"`
	ReferenceMonthOffset *int                     `json:"reference_month_offset"`
	Notes                string                   `json:"notes"`
}

func seedLeases(ctx context.Context, database *sql.DB, identities map[string]store.TenantIdentity, dir string) error {
	raw, err := os.ReadFile(filepath.Join(dir, "leases.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var items []seedLease
	if err := json.Unmarshal(raw, &items); err != nil {
		return fmt.Errorf("demo leases: %w", err)
	}
	extra, err := os.ReadFile(filepath.Join(dir, "leases-zinshaus.json"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		var more []seedLease
		if err := json.Unmarshal(extra, &more); err != nil {
			return fmt.Errorf("demo zinshaus leases: %w", err)
		}
		items = append(items, more...)
	}
	if len(items) == 0 {
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
	order := []string{}
	byHouse := map[string][]seedLease{}
	for _, item := range items {
		slug := item.House
		if slug == "" {
			slug = "janusbergweg-123"
		}
		slug = textutil.Slug(slug)
		if _, seen := byHouse[slug]; !seen {
			order = append(order, slug)
		}
		byHouse[slug] = append(byHouse[slug], item)
	}
	for _, slug := range order {
		identity, ok := identities[slug]
		if !ok {
			return fmt.Errorf("demo leases: missing %s", slug)
		}
		for _, item := range byHouse[slug] {
			if err := store.ReplaceLeaseGraph(tx, identity.Ref(), item.lease()); err != nil {
				return fmt.Errorf("demo lease %s: %w", item.ID, err)
			}
		}
		input := valorisationDraftInput(slug)
		if err := store.SeedValorisationDraft(tx, identity.Ref(), input, "musterstadt", time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func valorisationDraftInput(slug string) store.ValorisationInput {
	input := store.ValorisationInput{EffectiveOn: "2026-04-01", House: "Janusbergweg 123", Address: "Janusbergweg 123, 8010 Graz", Organisation: "Hausverwaltung Musterstadt", Contact: "vera.verwalter@musterstadt.example"}
	if slug == "musterstrasse-12" {
		input.House = "Musterstraße 12 · Zinshaus"
		input.Address = "Musterstraße 12, 8010 Graz"
	}
	return input
}

func (item seedLease) lease() store.Lease {
	twoWay := true
	if item.TwoWay != nil {
		twoWay = *item.TwoWay
	}
	consumer := true
	if item.Consumer != nil {
		consumer = *item.Consumer
	}
	lease := store.Lease{
		ID: item.ID, UnitID: item.Unit, Status: store.LeaseStatusActive,
		ConcludedOn: item.ConcludedOn, StartsOn: item.StartsOn, EndsOn: item.EndsOn,
		LeaseKind: store.LeaseKindHauptmiete, UseKind: item.UseKind, MRGScope: item.MRGScope, RentRegime: item.RentRegime,
		LandlordIsBusiness: true, TenantIsConsumer: consumer, VATOpted: item.VATBP > 0, ZinsterminDay: 5, Notes: item.Notes,
		Parties: []store.LeaseParty{{
			ID: item.ID + "-party", Name: item.TenantName, Email: item.TenantEmail, Address: item.TenantAddress,
			Role: store.PartyHauptmieter, ValidFrom: item.StartsOn,
		}},
	}
	if item.HMZCents > 0 {
		lease.Components = append(lease.Components, store.RentComponent{
			ID: item.ID + "-hmz", Kind: componentKind(item.UseKind), NetCents: item.HMZCents, VATRateBP: item.VATBP,
			ValidFrom: item.StartsOn, Origin: store.OriginImport,
		})
	}
	if item.BKCents > 0 {
		lease.Components = append(lease.Components, store.RentComponent{
			ID: item.ID + "-bk", Kind: store.ComponentBKAkonto, NetCents: item.BKCents, VATRateBP: item.VATBP,
			ValidFrom: item.StartsOn, Origin: store.OriginImport,
		})
	}
	if item.HeatCents > 0 {
		lease.Components = append(lease.Components, store.RentComponent{
			ID: item.ID + "-heat", Kind: store.ComponentHeizAkonto, NetCents: item.HeatCents, VATRateBP: item.VATBP,
			ValidFrom: item.StartsOn, Origin: store.OriginImport,
		})
	}
	clause := store.IndexClause{
		StaffelSteps: item.StaffelSteps,
		ID:           item.ID + "-clause", ClauseType: item.ClauseType, Series: item.Series, BasePeriod: item.BasePeriod, BaseValue: item.BaseValue,
		ThresholdKind: item.ThresholdKind, ThresholdValue: item.Threshold, ThresholdInclusive: item.Inclusive,
		FullChangeOnTrigger: true, TwoWay: twoWay, PeriodicMonth: item.PeriodicMonth, ReferenceMonthOffset: item.ReferenceMonthOffset, ClauseText: item.ClauseText,
		ReviewStatus: item.Review, ValidFrom: item.StartsOn,
	}
	if item.LastMonth != "" {
		clause.State = &store.ValorisationState{
			ContractValue: item.HMZAfter, ContractBasePeriod: item.LastMonth, ContractBaseValue: item.LastValue,
			CapValue: item.HMZAfter, CapAnchorPeriod: item.LastMonth,
			LastEffectiveOn: item.LastEffectiveOn,
		}
		if item.ClauseType == store.ClauseStaffel {
			// A fixed schedule has a cap anchor, but no contractual index base.
			clause.State.ContractBasePeriod, clause.State.ContractBaseValue = "", ""
		}
	}
	lease.Clauses = []store.IndexClause{clause}
	return lease
}

func componentKind(useKind string) string {
	if useKind == store.UseKindGarage {
		return store.ComponentStellplatz
	}
	return store.ComponentHMZ
}

package server

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/store"
)

// Connector ingestion preserves the sensor timestamp and its confirmed mapping.
// Receipt time is never substituted for a missing period-boundary measurement.
func (a *app) ingestAnnualStatementConsumption(slug string, readings []store.HomeConnectorReading, receivedAt time.Time) error {
	return a.ingestAnnualStatementHomeConsumption(slug, "", readings, receivedAt)
}

func (a *app) ingestAnnualStatementHomeConsumption(slug, sourceHome string, readings []store.HomeConnectorReading, receivedAt time.Time) error {
	if a.annualConsumption == nil || a.energyStore == nil || len(readings) == 0 {
		return nil
	}
	profiles, err := a.energyStore.ListProfiles(slug)
	if err != nil {
		return err
	}
	type target struct{ home, unit, cost string }
	targets := map[string][]target{}
	for _, profile := range profiles {
		if store.NormalizeUnitID(profile.UnitID) == "" {
			continue
		}
		home := a.energyStore.ForHome(profile.HomeKey)
		assets, err := home.ListAssets(slug)
		if err != nil {
			return err
		}
		mappings, err := home.ListMappings(slug)
		if err != nil {
			return err
		}
		costs := map[string]string{}
		for _, asset := range assets {
			if !asset.Confirmed {
				continue
			}
			switch asset.Kind {
			case "heat-pump":
				costs[asset.ID] = "heizung"
			case "hot-water", "instant-water-heater":
				costs[asset.ID] = "warmwasser"
			}
		}
		for _, mapping := range mappings {
			if !mapping.Confirmed || mapping.Metric != energy.MetricConsumerEnergy || costs[mapping.AssetID] == "" {
				continue
			}
			entity := strings.ToLower(strings.TrimSpace(mapping.EntityID))
			targets[entity] = append(targets[entity], target{profile.HomeKey, store.NormalizeUnitID(profile.UnitID), costs[mapping.AssetID]})
		}
	}
	if len(targets) == 0 {
		return nil
	}
	identity, ok := a.tenantIdentity(slug)
	if !ok {
		return fmt.Errorf("consumption tenant unavailable")
	}
	repository, ok := store.BindAnnualStatementConsumptionRepository(a.annualConsumption, identity.Ref())
	if !ok {
		return fmt.Errorf("consumption repository unavailable")
	}
	var appendErrors []error
	for _, reading := range readings {
		entity := strings.ToLower(strings.TrimSpace(reading.EntityID))
		mapped := targets[entity]
		// An entity shared between homes or cost types is not an allocation fact.
		if len(mapped) != 1 || (sourceHome != "" && mapped[0].home != sourceHome) || reading.DeviceClass != "energy" ||
			(reading.StateClass != "total" && reading.StateClass != "total_increasing") ||
			reading.LastUpdated.IsZero() || reading.LastUpdated.After(receivedAt) {
			continue
		}
		micros, valid := annualStatementEnergyMicros(reading.State, reading.Unit)
		if !valid {
			continue
		}
		sourceID := entity
		if sourceHome != "" {
			sourceID = "ha:" + sourceHome + ":" + entity
		}
		if _, _, err := repository.Append(store.AnnualStatementConsumptionEvidence{
			UnitID: mapped[0].unit, CostTypeKey: mapped[0].cost,
			SourceKind: store.ConsumptionSourceEntity, SourceID: sourceID,
			MeasuredAt: reading.LastUpdated, ReceivedAt: receivedAt,
			ValueMicros: micros, MeasurementUnit: "kWh",
		}); err != nil {
			// One conflicting historical fact must not discard later counters.
			logWarn("annual statement consumption reading not recorded", "tenant", slug, "entity", entity, "error_type", "store")
			appendErrors = append(appendErrors, err)
		}
	}
	return errors.Join(appendErrors...)
}

// Sample confirmed consumer counters independently of the grid-power sampler.
func (a *app) sampleAnnualStatementConsumption(ctx context.Context, tenant tenantConfig, homeKey string, now time.Time) {
	if a.annualConsumption == nil || a.energyStore == nil {
		return
	}
	mappings, err := a.energyStore.ForHome(homeKey).ListMappings(tenant.Slug)
	if err != nil {
		return
	}
	var readings []store.HomeConnectorReading
	for _, mapping := range mappings {
		if !mapping.Confirmed || mapping.Metric != energy.MetricConsumerEnergy {
			continue
		}
		readCtx, cancel := context.WithTimeout(ctx, energySampleReadTimeout)
		state, err := tenant.HA.State(readCtx, mapping.EntityID)
		cancel()
		if err != nil {
			logWarn("annual statement confirmed counter could not be read", "tenant", tenant.Slug, "home", homeKey, "entity", mapping.EntityID, "error_type", "read")
			if ctx.Err() != nil {
				break
			}
			continue
		}
		readings = append(readings, store.HomeConnectorReading{
			EntityID: mapping.EntityID, State: state.State, LastUpdated: state.LastUpdated,
			Unit:        haAttribute(state.Attributes, "unit_of_measurement"),
			DeviceClass: haAttribute(state.Attributes, "device_class"), StateClass: haAttribute(state.Attributes, "state_class"),
		})
	}
	if err := a.ingestAnnualStatementHomeConsumption(tenant.Slug, homeKey, readings, now); err != nil {
		logWarn("annual statement sampled consumption persist failed", "error_type", "store")
	}
}

var annualStatementEnergyDecimal = regexp.MustCompile(`^[0-9]+([.,][0-9]+)?$`)

// Convert exactly to micro-kWh; unrepresentable values remain missing evidence.
func annualStatementEnergyMicros(raw, unit string) (int64, bool) {
	raw = strings.TrimSpace(raw)
	if len(raw) > 48 || !annualStatementEnergyDecimal.MatchString(raw) {
		return 0, false
	}
	value, ok := new(big.Rat).SetString(strings.ReplaceAll(raw, ",", "."))
	if !ok {
		return 0, false
	}
	var factor int64
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "wh":
		factor = 1_000
	case "kwh":
		factor = 1_000_000
	case "mwh":
		factor = 1_000_000_000
	default:
		return 0, false
	}
	value.Mul(value, new(big.Rat).SetInt64(factor))
	if !value.IsInt() || !value.Num().IsInt64() {
		return 0, false
	}
	return value.Num().Int64(), true
}

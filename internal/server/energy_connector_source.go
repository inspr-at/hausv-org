package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/homeassistant"
	"github.com/inspr-at/hausv-org/internal/store"
)

// energyStates keeps the existing operator-managed HA integration as the
// first choice and falls back to the outbound-only Home connector for private
// self-service portals. The connector slug is the authorization boundary.
func (a *app) energyStates(ctx context.Context, tenant tenantConfig) ([]homeassistant.EntityState, bool, error) {
	if tenant.HA.Configured() {
		states, err := tenant.HA.States(ctx)
		return states, true, err
	}
	if a.homeConnectors == nil || a.homeConnectorReadings == nil {
		return nil, false, nil
	}
	connector, found, err := a.homeConnectors.Get(tenant.Slug)
	if err != nil {
		return nil, false, err
	}
	if !found || connector.Status != store.HomeConnectorConnected {
		return nil, false, nil
	}
	if connector.LastSeenAt == nil || time.Since(*connector.LastSeenAt) > homeConnectorFreshFor {
		return nil, true, fmt.Errorf("home connector is offline")
	}
	readings, err := a.homeConnectorReadings.List(tenant.Slug)
	if err != nil {
		return nil, true, err
	}
	states := make([]homeassistant.EntityState, 0, len(readings))
	for _, reading := range readings {
		states = append(states, homeassistant.EntityState{
			EntityID: reading.EntityID,
			State:    reading.State,
			Attributes: map[string]any{
				"friendly_name":       reading.DisplayName,
				"unit_of_measurement": reading.Unit,
				"device_class":        reading.DeviceClass,
				"state_class":         reading.StateClass,
			},
			LastChanged: reading.LastUpdated,
			LastUpdated: reading.LastUpdated,
		})
	}
	return states, true, nil
}

func energyStateByID(states []homeassistant.EntityState, entityID string) (homeassistant.EntityState, error) {
	entityID = strings.ToLower(strings.TrimSpace(entityID))
	for _, state := range states {
		if strings.EqualFold(strings.TrimSpace(state.EntityID), entityID) {
			return state, nil
		}
	}
	return homeassistant.EntityState{}, fmt.Errorf("energy entity unavailable")
}

package homeconnector

import "testing"

func TestConnectorAllowsOnlyExplicitVehicleSleepSignalsHAUSV551(t *testing.T) {
	for _, fixture := range []struct {
		entityID, state, name, unit, deviceClass string
		want                                     bool
	}{
		{entityID: "binary_sensor.vehicle_model_x_asleep", state: "on", name: "Vehicle Model X Asleep", want: true},
		{entityID: "sensor.auto_model_x_sleeping", state: "asleep", name: "Auto Model X Sleeping", want: true},
		{entityID: "binary_sensor.zoe_asleep", state: "on", name: "Zoe Asleep", want: false},
		{entityID: "binary_sensor.bedroom_sleeping", state: "on", name: "Bedroom sleeping", want: false},
		{entityID: "binary_sensor.front_door", state: "on", name: "Front door", want: false},
		{entityID: "sensor.model_x_state", state: "asleep", name: "Model X state", want: false},
		{entityID: "sensor.model_x_power", state: "1200", name: "Model X Power", unit: "W", deviceClass: "power", want: true},
	} {
		got := allowedEnergyReading(fixture.entityID, fixture.state, fixture.name, fixture.unit, fixture.deviceClass)
		if got != fixture.want {
			t.Errorf("allowedEnergyReading(%q, %q) = %t, want %t", fixture.entityID, fixture.state, got, fixture.want)
		}
	}
}

func TestSelectedHeartbeatKeepsVehicleSleepDiscoveryHAUSV551(t *testing.T) {
	heartbeat := Heartbeat{Readings: []Reading{
		{EntityID: "sensor.house_power", State: "1200", DisplayName: "House power", Unit: "W", DeviceClass: "power"},
		{EntityID: "binary_sensor.vehicle_model_x_asleep", State: "on", DisplayName: "Vehicle Model X Asleep"},
		{EntityID: "binary_sensor.bedroom_sleeping", State: "on", DisplayName: "Bedroom sleeping"},
	}}
	filtered := selectHeartbeatReadings(heartbeat, []string{"sensor.house_power"})
	if len(filtered.Readings) != 2 || filtered.Readings[0].EntityID != "sensor.house_power" ||
		filtered.Readings[1].EntityID != "binary_sensor.vehicle_model_x_asleep" {
		t.Fatalf("selected heartbeat did not retain only energy plus vehicle sleep: %+v", filtered.Readings)
	}
}

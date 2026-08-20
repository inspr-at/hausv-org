package homeconnector

import "testing"

func TestConnectorAllowsOnlyExplicitVehicleSleepSignalsHAUSV551(t *testing.T) {
	for _, fixture := range []struct {
		entityID, state, name, unit, deviceClass string
		want                                     bool
	}{
		{entityID: "binary_sensor.model_x_asleep", state: "on", name: "Model X Asleep", want: true},
		{entityID: "sensor.model_x_sleeping", state: "asleep", name: "Model X Sleeping", want: true},
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

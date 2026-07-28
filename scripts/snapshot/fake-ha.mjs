#!/usr/bin/env node
// Deterministic, read-only Home Assistant fixture for the energy onboarding QA.

import { createServer } from 'node:http';

const port = Number(process.argv[2] || 8122);
const updated = '2026-07-28T10:00:00Z';
const states = [
  state('sensor.grid_import_power', '2.4', 'Netzbezug', 'power', 'kW'),
  state('sensor.grid_export_power', '0.4', 'Netzeinspeisung', 'power', 'kW'),
  state('sensor.grid_import_energy', '42', 'Netzbezug Energie', 'energy', 'kWh'),
  state('sensor.pv_current_power', '3.1', 'PV Leistung', 'power', 'kW'),
  state('sensor.home_battery_soc', '78', 'Hausspeicher Ladestand', 'battery', '%'),
  state('sensor.home_consumption', '1.8', 'Hausverbrauch', 'power', 'kW'),
  state('sensor.battery_charge_power', '0.8', 'Batteriespeicher Ladeleistung', 'power', 'kW'),
  state('sensor.iphone_battery', '81', 'iPhone Battery', 'battery', '%'),
  state('sensor.robot_battery', '64', 'Saugroboter Battery', 'battery', '%'),
  state('sensor.pv_forecast_power', '4.4', 'PV Forecast Power', 'power', 'kW'),
  state('sensor.kettle_power', '1.9', 'Wasserkocher Leistung', 'power', 'kW'),
  state('number.battery_force_charge', '0', 'Battery Force Charge', 'battery', '%'),
  state('binary_sensor.lock_battery', 'off', 'Nuki Battery', 'battery', '%'),
  state('switch.wallbox', 'off', 'Wallbox', '', ''),
];

function state(entity_id, value, friendly_name, device_class, unit_of_measurement) {
  return {
    entity_id,
    state: value,
    attributes: {
      friendly_name,
      device_class,
      unit_of_measurement,
      state_class: 'measurement',
    },
    last_changed: updated,
    last_updated: updated,
  };
}

createServer((request, response) => {
  response.setHeader('Content-Type', 'application/json');
  if (request.method === 'GET' && request.url === '/api/states') {
    response.end(JSON.stringify(states));
    return;
  }
  if (request.method === 'GET' && request.url?.startsWith('/api/states/')) {
    const entityID = decodeURIComponent(request.url.slice('/api/states/'.length));
    const match = states.find((item) => item.entity_id === entityID);
    if (match) {
      response.end(JSON.stringify(match));
      return;
    }
    response.statusCode = 404;
    response.end('{"message":"not found"}');
    return;
  }
  response.statusCode = 404;
  response.end('{"message":"not found"}');
}).listen(port, '127.0.0.1', () => {
  process.stdout.write(`fake Home Assistant listening on ${port}\n`);
});

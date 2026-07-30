#!/usr/bin/env node
// Deterministic, read-only Home Assistant fixture for the energy onboarding QA.

import { createServer } from 'node:http';

const port = Number(process.argv[2] || 8122);
const updated = new Date().toISOString();
const mapTilePNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=',
  'base64',
);
const commonNoise = [
  state('sensor.iphone_battery', '81', 'iPhone Battery', 'battery', '%'),
  state('sensor.robot_battery', '64', 'Saugroboter Battery', 'battery', '%'),
  state('sensor.pv_forecast_power', '4.4', 'PV Forecast Power', 'power', 'kW'),
  state('sensor.kettle_power', '1.9', 'Wasserkocher Leistung', 'power', 'kW'),
  state('number.battery_force_charge', '0', 'Battery Force Charge', 'battery', '%'),
  state('binary_sensor.lock_battery', 'off', 'Nuki Battery', 'battery', '%'),
  state('switch.wallbox', 'off', 'Wallbox', '', ''),
];
const homes = {
  jhw22: [
  state('sensor.grid_import_power', '2.4', 'Netzbezug', 'power', 'kW'),
  state('sensor.grid_export_power', '0.4', 'Netzeinspeisung', 'power', 'kW'),
  state('sensor.grid_import_energy', '42', 'Netzbezug Energie', 'energy', 'kWh'),
  state('sensor.pv_current_power', '3.1', 'PV Leistung', 'power', 'kW'),
  state('sensor.home_battery_soc', '78', 'Hausspeicher Ladestand', 'battery', '%'),
  state('sensor.home_consumption', '1.8', 'Hausverbrauch', 'power', 'kW'),
  state('sensor.battery_charge_power', '0.8', 'Batteriespeicher Ladeleistung', 'power', 'kW'),
  state('sensor.battery_discharge_power', '0.2', 'Batteriespeicher Entladeleistung', 'power', 'kW'),
  ...commonNoise,
  ],
  eltern: [
    state('sensor.parents_grid_import_power', '3.2', 'Netzbezug Haus Eltern', 'power', 'kW'),
    state('sensor.parents_grid_import_energy', '73', 'Netzbezug Energie Haus Eltern', 'energy', 'kWh'),
    state('sensor.parents_grid_export_power', '1.1', 'Netzeinspeisung Haus Eltern', 'power', 'kW'),
    state('sensor.parents_pv_current_power', '5.6', 'PV Leistung Haus Eltern', 'power', 'kW'),
    state('sensor.parents_home_consumption', '2.5', 'Hausverbrauch Eltern', 'power', 'kW'),
    ...commonNoise,
  ],
  schwiegereltern: [
    state('sensor.inlaws_grid_import_power', '1.7', 'Netzbezug Haus Schwiegereltern', 'power', 'kW'),
    state('sensor.inlaws_grid_import_energy', '51', 'Netzbezug Energie Haus Schwiegereltern', 'energy', 'kWh'),
    state('sensor.inlaws_pv_current_power', '4.2', 'PV Leistung Haus Schwiegereltern', 'power', 'kW'),
    state('sensor.inlaws_home_battery_soc', '66', 'Hausspeicher Ladestand Schwiegereltern', 'battery', '%'),
    state('sensor.inlaws_battery_charge_power', '1.2', 'Batteriespeicher Ladeleistung Schwiegereltern', 'power', 'kW'),
    state('sensor.inlaws_home_consumption', '2.1', 'Hausverbrauch Schwiegereltern', 'power', 'kW'),
    ...commonNoise,
  ],
};

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
  const url = new URL(request.url || '/', 'http://127.0.0.1');
  if (request.method === 'GET' && /^\/map-tiles\/\d+\/\d+\/\d+\.png$/.test(url.pathname)) {
    response.setHeader('Content-Type', 'image/png');
    response.setHeader('Cache-Control', 'no-store');
    response.end(mapTilePNG);
    process.stdout.write(`fake map tile served ${url.pathname}\n`);
    return;
  }
  response.setHeader('Content-Type', 'application/json');
  const historyMatch = url.pathname.match(/^\/(jhw22|eltern|schwiegereltern)\/api\/history\/period(?:\/.*)?$/);
  if (request.method === 'GET' && historyMatch) {
    const states = homes[historyMatch[1]];
    const requested = (url.searchParams.get('filter_entity_id') || '').split(',').filter(Boolean);
    const startPart = url.pathname.split('/period/')[1];
    const start = startPart ? new Date(decodeURIComponent(startPart)) : new Date(Date.now() - 24 * 3600000);
    const requestedEnd = url.searchParams.get('end_time');
    const end = requestedEnd ? new Date(requestedEnd) : new Date();
    const pointCount = Math.max(1, Math.floor((end.getTime() - start.getTime()) / (15 * 60000)) + 1);
    const groups = requested.map((entityID) => {
      const source = states.find((candidate) => candidate.entity_id === entityID);
      if (!source) return [];
      return Array.from({ length: pointCount }, (_, index) => {
        const at = new Date(start.getTime() + index * 15 * 60000).toISOString();
        return {
          entity_id: index === 0 || index === pointCount - 1 ? entityID : undefined,
          state: String(historyValue(entityID, Number(source.state), index)),
          last_changed: at,
          last_updated: at,
          attributes: index === 0 ? source.attributes : undefined,
        };
      });
    }).filter((group) => group.length);
    response.end(JSON.stringify(groups));
    return;
  }
  const match = url.pathname.match(/^\/(jhw22|eltern|schwiegereltern)\/api\/states(?:\/(.*))?$/);
  if (request.method !== 'GET' || !match) {
    response.statusCode = 404;
    response.end('{"message":"not found"}');
    return;
  }
  const states = homes[match[1]];
  if (!match[2]) {
    response.end(JSON.stringify(states));
    return;
  }
  const entityID = decodeURIComponent(match[2]);
  const item = states.find((candidate) => candidate.entity_id === entityID);
  if (item) {
      response.end(JSON.stringify(item));
      return;
  }
  response.statusCode = 404;
  response.end('{"message":"not found"}');
}).listen(port, '127.0.0.1', () => {
  process.stdout.write(`fake Home Assistant listening on ${port}\n`);
});

function historyValue(entityID, current, index) {
  const phase = index / 96;
  const daylight = Math.max(0, Math.sin((phase - 0.25) * Math.PI * 2));
  const evening = Math.exp(-Math.pow((phase - 0.82) / 0.08, 2));
  const morning = Math.exp(-Math.pow((phase - 0.30) / 0.07, 2));
  if (entityID.includes('pv_current_power')) return Number((daylight * Math.max(current, 5.2)).toFixed(3));
  if (entityID.includes('home_consumption')) return Number((0.8 + morning * 1.5 + evening * 4.6 + phase * 0.25).toFixed(3));
  if (entityID.includes('grid_import_power')) return Number((0.15 + evening * 2.1).toFixed(3));
  if (entityID.includes('grid_export_power')) return Number((daylight * 1.7).toFixed(3));
  if (entityID.includes('battery_charge_power')) return Number((daylight * 1.4).toFixed(3));
  if (entityID.includes('battery_discharge_power')) return Number((evening * 1.8).toFixed(3));
  return current;
}

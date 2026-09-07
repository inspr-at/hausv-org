package server

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

// HAUSV-671: a NaN or Inf kilowatt value anywhere in the flow config made
// json.Marshal fail; the page shipped "null" and the client renderer left the
// no-JS fallback in place — the cockpit lost its chart and its icons. The
// config must always encode, with non-finite numerics reduced to 0.
func TestEnergyFlowConfigEncodesWithNonFiniteReadings(t *testing.T) {
	cfg := energyFlowConfig{
		Home:      energyFlowNodeConfig{ID: "home", KW: math.NaN()},
		Producers: []energyFlowNodeConfig{{ID: "pv", KW: math.Inf(1)}, {ID: "pv2", KW: 4.87}},
		Storage:   &energyFlowNodeConfig{ID: "battery", KW: math.Inf(-1)},
		Grid:      &energyFlowNodeConfig{ID: "grid", KW: 0.032},
		Consumers: []energyFlowConsumerConfig{{ID: "heatpump", KW: math.NaN()}, {ID: "wallbox", KW: 2.5}},
	}
	if _, err := json.Marshal(cfg); err == nil {
		t.Fatal("precondition: encoding/json must refuse NaN/Inf, or this test proves nothing")
	}
	sanitizeEnergyFlowConfig(&cfg)
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("sanitized config must encode: %v", err)
	}
	text := string(raw)
	for _, want := range []string{`"id":"home","kw":0`, `"id":"pv","kw":0`, `"id":"pv2","kw":4.87`, `"id":"battery","kw":0`, `"id":"grid","kw":0.032`, `"id":"heatpump"`, `"id":"wallbox"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %s", want, text)
		}
	}
	if cfg.Consumers[0].KW != 0 || cfg.Consumers[1].KW != 2.5 {
		t.Fatalf("consumer kw: %v %v", cfg.Consumers[0].KW, cfg.Consumers[1].KW)
	}
}

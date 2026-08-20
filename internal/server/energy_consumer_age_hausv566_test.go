package server

import (
	"testing"
	"time"
)

func TestEnergyConsumerAgeLabelUsesGermanDayInflectionHAUSV566(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		age  time.Duration
		want string
	}{
		{name: "one day", age: 24 * time.Hour, want: "Stand vor 1 Tag"},
		{name: "one full day", age: 47 * time.Hour, want: "Stand vor 1 Tag"},
		{name: "multiple days", age: 48 * time.Hour, want: "Stand vor 2 Tagen"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := energyConsumerAgeLabel(now, now.Add(-test.age)); got != test.want {
				t.Fatalf("energyConsumerAgeLabel() = %q, want %q", got, test.want)
			}
		})
	}
}

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/markus-barta/hausv-org/internal/store"
)

func runChargingMode(args []string) error {
	flags := flag.NewFlagSet("charging-mode", flag.ContinueOnError)
	tenant := flags.String("tenant", "", "tenant slug")
	mode := flags.String("mode", "", "live, shadow, or disabled")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *tenant == "" || flags.NArg() != 0 {
		return fmt.Errorf("--tenant is required")
	}

	parkingPath := os.Getenv("PARKING_DATA_PATH")
	if parkingPath == "" {
		return fmt.Errorf("PARKING_DATA_PATH is required")
	}
	parking, err := store.NewParkingStore(parkingPath)
	if err != nil {
		return err
	}
	settings := parking.TenantData(*tenant).Settings.Charging
	previousMode := chargingModeName(settings)
	switch *mode {
	case "live":
		settings.Enabled = true
		settings.ShadowMode = false
	case "shadow":
		settings.Enabled = true
		settings.ShadowMode = true
	case "disabled":
		settings.Enabled = false
		settings.ShadowMode = false
	default:
		return fmt.Errorf("--mode must be live, shadow, or disabled")
	}
	settings = store.NormalizeChargingControlSettings(settings)
	if err := parking.SetChargingControl(*tenant, settings); err != nil {
		return err
	}

	auditPath := os.Getenv("AUDIT_DATA_PATH")
	if auditPath != "" {
		audit, err := store.NewAuditStore(auditPath)
		if err != nil {
			return err
		}
		if err := audit.Append(store.AuditEvent{
			At:         time.Now().UTC(),
			TenantSlug: *tenant,
			ActorEmail: "operator@hausv.org",
			ActorRole:  "platform-admin",
			Action:     store.AuditActionChargingSettings,
			TargetType: "charging",
			TargetID:   *tenant,
			Summary:    "Laderegelung per Operator-Kommando geändert",
			Details: map[string]string{
				"previous_mode": previousMode,
				"mode":          *mode,
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

func chargingModeName(settings store.ChargingControlSettings) string {
	switch {
	case !settings.Enabled:
		return "disabled"
	case settings.ShadowMode:
		return "shadow"
	default:
		return "live"
	}
}

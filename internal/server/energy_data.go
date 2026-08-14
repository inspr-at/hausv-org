package server

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/store"
)

const (
	energyRawImportRetention = 30 * 24 * time.Hour
	energyRetentionSweep     = 6 * time.Hour
	maxEnergyExportBytes     = 64 << 20
)

type energyDataImportView struct {
	Filename string
	Date     string
	Format   string
	Size     string
}

type energyExportFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

func energyRetentionCutoffs(now time.Time) (time.Time, time.Time, time.Time) {
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()
	return now.Add(-energyRawImportRetention), now.AddDate(0, -13, 0), now.AddDate(-3, 0, 0)
}

func (a *app) purgeExpiredEnergyData(now time.Time) {
	failed := false
	if a.auditStore != nil {
		if err := a.auditStore.PurgeExpired(now); err != nil {
			logError("expired audit data purge failed", err)
			failed = true
		}
	}
	if a.energyStore != nil {
		rawBefore, intervalBefore, assessmentBefore := energyRetentionCutoffs(now)
		summary, err := a.energyStore.PurgeExpired(rawBefore, intervalBefore, assessmentBefore)
		if err != nil {
			logError("expired energy data purge failed", err)
			failed = true
		} else if summary.Imports+summary.Intervals+summary.TariffAssessments > 0 {
			logInfo("expired energy data purged",
				"raw_imports", summary.Imports,
				"intervals", summary.Intervals,
				"assessments", summary.TariffAssessments,
			)
		}
	}
	a.retentionFailure.Store(failed)
}

func (a *app) startEnergyRetentionWorker() func() {
	if a.energyStore == nil && a.auditStore == nil {
		return func() {}
	}
	a.purgeExpiredEnergyData(time.Now())
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(energyRetentionSweep)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				a.purgeExpiredEnergyData(now)
			}
		}
	}()
	return cancel
}

func (a *app) energyDataAccess(ac authCtx) (energy.HomeProfile, bool) {
	profile, exists, err := a.energyStore.Profile(ac.tenant.Slug)
	if err != nil || !exists || energyProfileUnclaimed(profile) || !a.canManageHomeIdentityProfile(ac, profile, exists) {
		return energy.HomeProfile{}, false
	}
	return profile, true
}

func (a *app) energyDataPage(w http.ResponseWriter, r *http.Request, ac authCtx) {
	profile, ok := a.energyDataAccess(ac)
	if !ok {
		http.Error(w, "Dieser Bereich ist Eigentümern und der Hausadministration vorbehalten.", http.StatusForbidden)
		return
	}
	imports, err := a.energyStore.ListImports(ac.tenant.Slug)
	if err != nil {
		http.Error(w, "Energiedaten konnten nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	importViews := make([]energyDataImportView, 0, len(imports))
	for _, item := range imports {
		importViews = append(importViews, energyDataImportView{
			Filename: item.Filename,
			Date:     item.ImportedAt.In(time.Local).Format("02.01.2006, 15:04"),
			Format:   item.Format,
			Size:     "nach 30 Tagen · Löschung spätestens 6 Stunden später",
		})
	}
	intervals, intervalsErr := a.energyStore.ListIntervals(ac.tenant.Slug, time.Time{}, time.Time{})
	assessments, assessmentsErr := a.energyStore.ListTariffAssessments(ac.tenant.Slug)
	assets, assetsErr := a.energyStore.ListAssets(ac.tenant.Slug)
	mappings, mappingsErr := a.energyStore.ListMappings(ac.tenant.Slug)
	if intervalsErr != nil || assessmentsErr != nil || assetsErr != nil || mappingsErr != nil {
		http.Error(w, "Energiedaten konnten nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	a.render(w, "energyData", a.withBase(ac, map[string]any{
		"Title":             "Energiedaten & Datenschutz",
		"ActivePage":        "settings",
		"Profile":           profile,
		"Imports":           importViews,
		"HasImports":        len(importViews) > 0,
		"ImportCount":       len(importViews),
		"IntervalCount":     len(intervals),
		"AssessmentCount":   len(assessments),
		"AssetCount":        len(assets),
		"MappingCount":      len(mappings),
		"CanControlEnergy":  a.canControlEnergy(ac),
		"IsObserveMode":     profile.OperatingMode == energy.ModeObserve,
		"IsActiveMode":      profile.OperatingMode == energy.ModeActive,
		"IsShadowMode":      profile.AutomationStage == energy.StageShadow,
		"HistoryDeleted":    r.URL.Query().Get("result") == "history-deleted",
		"ExportUnavailable": r.URL.Query().Get("result") == "export-too-large",
	}))
}

func (a *app) exportEnergyData(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if _, ok := a.energyDataAccess(ac); !ok {
		http.Error(w, "Dieser Export ist Eigentümern und der Hausadministration vorbehalten.", http.StatusForbidden)
		return
	}
	payload, filename, counts, err := a.buildEnergyDataPackage(ac, time.Now())
	if err != nil {
		if strings.Contains(err.Error(), "export too large") {
			http.Redirect(w, r, "/app/settings/energy-data?result=export-too-large", http.StatusSeeOther)
			return
		}
		logError("energy data export failed", err, "tenant", ac.tenant.Slug)
		http.Error(w, "Der Energieexport konnte nicht erstellt werden.", http.StatusInternalServerError)
		return
	}
	if a.auditStore == nil {
		http.Error(w, "Der Export konnte nicht protokolliert werden.", http.StatusInternalServerError)
		return
	}
	digest := sha256.Sum256(payload)
	if err := a.auditStore.Append(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     store.AuditActionEnergyExport,
		TargetType: "home-energy",
		TargetID:   ac.tenant.Slug,
		Summary:    "Eigene Energiedaten exportiert",
		Details: map[string]string{
			"assets":      strconv.Itoa(counts["assets"]),
			"mappings":    strconv.Itoa(counts["mappings"]),
			"intervals":   strconv.Itoa(counts["intervals"]),
			"raw_imports": strconv.Itoa(counts["raw_imports"]),
			"digest":      hex.EncodeToString(digest[:8]),
		},
	}); err != nil {
		http.Error(w, "Der Export konnte nicht protokolliert werden.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	_, _ = w.Write(payload)
}

func (a *app) deleteEnergyMeasurementData(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if _, ok := a.energyDataAccess(ac); !ok {
		http.Error(w, "Diese Löschung ist Eigentümern und der Hausadministration vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil || strings.TrimSpace(r.FormValue("confirmation")) != "MESSVERLAUF LÖSCHEN" {
		http.Error(w, "Bitte bestätigen Sie die Löschung mit dem angezeigten Text.", http.StatusBadRequest)
		return
	}
	if a.auditStore == nil || a.auditStore.Append(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     store.AuditActionEnergyHistoryDelete,
		TargetType: "home-energy-history",
		TargetID:   ac.tenant.Slug,
		Summary:    "Löschung des Energie-Messverlaufs bestätigt",
		Details:    map[string]string{"stage": "requested"},
	}) != nil {
		http.Error(w, "Die Löschung konnte nicht sicher protokolliert werden.", http.StatusInternalServerError)
		return
	}
	summary, err := a.energyStore.DeleteMeasurementData(ac.tenant.Slug)
	if err != nil {
		http.Error(w, "Der Messverlauf konnte nicht gelöscht werden.", http.StatusInternalServerError)
		return
	}
	if err := a.auditStore.Append(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     store.AuditActionEnergyHistoryDelete,
		TargetType: "home-energy-history",
		TargetID:   ac.tenant.Slug,
		Summary:    "Energie-Messverlauf gelöscht",
		Details: map[string]string{
			"intervals":           strconv.Itoa(summary.Intervals),
			"raw_imports":         strconv.Itoa(summary.Imports),
			"tariff_assessments":  strconv.Itoa(summary.TariffAssessments),
			"measure_comparisons": strconv.Itoa(summary.Measures),
			"stage":               "completed",
		},
	}); err != nil {
		logError("completed energy history deletion audit failed", err, "tenant", ac.tenant.Slug)
	}
	http.Redirect(w, r, "/app/settings/energy-data?result=history-deleted", http.StatusSeeOther)
}

func (a *app) deleteEnergyProfile(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if _, ok := a.energyDataAccess(ac); !ok {
		http.Error(w, "Diese Löschung ist Eigentümern und der Hausadministration vorbehalten.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil || strings.TrimSpace(r.FormValue("confirmation")) != "ENERGIEPROFIL LÖSCHEN" {
		http.Error(w, "Bitte bestätigen Sie die Löschung mit dem angezeigten Text.", http.StatusBadRequest)
		return
	}
	if a.auditStore == nil || a.auditStore.Append(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     store.AuditActionEnergyProfileDelete,
		TargetType: "home-energy",
		TargetID:   ac.tenant.Slug,
		Summary:    "Löschung des Energieprofils bestätigt",
		Details:    map[string]string{"stage": "requested"},
	}) != nil {
		http.Error(w, "Die Löschung konnte nicht sicher protokolliert werden.", http.StatusInternalServerError)
		return
	}
	revoked, err := a.revokeTechnicalEnergyAccess(ac.tenant.Slug)
	if err != nil {
		logError("technical grant revocation before energy profile deletion failed", err, "tenant", ac.tenant.Slug)
		http.Error(w, "Technische Freigaben konnten nicht sicher widerrufen werden; das Energieprofil wurde nicht gelöscht.", http.StatusInternalServerError)
		return
	}
	if a.homeConnectorReadings != nil {
		if err := a.homeConnectorReadings.Clear(ac.tenant.Slug); err != nil {
			logError("home connector readings before energy profile deletion failed", err, "tenant", ac.tenant.Slug)
			http.Error(w, "Lokale Messwerte konnten nicht sicher gelöscht werden; das Energieprofil wurde nicht gelöscht.", http.StatusInternalServerError)
			return
		}
	}
	summary, err := a.energyStore.DeleteProfile(ac.tenant.Slug)
	if err != nil {
		// Revoking delegated access before deleting is fail-safe: a later retry
		// may complete the deletion, while stale grants can never revive through
		// a re-onboarding that happened after a partial failure.
		http.Error(w, "Technische Freigaben wurden widerrufen, das Energieprofil konnte aber noch nicht gelöscht werden. Bitte erneut versuchen.", http.StatusInternalServerError)
		return
	}
	// Home Assistant histories are intentionally transient. Remove every
	// tenant-scoped in-memory chart immediately with the profile instead of
	// waiting for lazy cache expiry.
	a.clearEnergyChartCache(ac.tenant.Slug)
	if err := a.auditStore.Append(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     store.AuditActionEnergyProfileDelete,
		TargetType: "home-energy",
		TargetID:   ac.tenant.Slug,
		Summary:    "Energieprofil gelöscht",
		Details: map[string]string{
			"assets":             strconv.Itoa(summary.Assets),
			"mappings":           strconv.Itoa(summary.Mappings),
			"intervals":          strconv.Itoa(summary.Intervals),
			"raw_imports":        strconv.Itoa(summary.Imports),
			"maintenance":        strconv.Itoa(summary.Maintenance),
			"tariff_assessments": strconv.Itoa(summary.TariffAssessments),
			"measures":           strconv.Itoa(summary.Measures),
			"technical_grants":   strconv.Itoa(revoked),
			"stage":              "completed",
		},
	}); err != nil {
		logError("completed energy profile deletion audit failed", err, "tenant", ac.tenant.Slug)
	}
	http.Redirect(w, r, "/app/zuhause/onboarding?reset=1", http.StatusSeeOther)
}

func (a *app) revokeTechnicalEnergyAccess(tenantSlug string) (int, error) {
	if a.inviteStore == nil {
		return 0, nil
	}
	candidates := map[string]userProfile{}
	for email, profile := range a.profiles {
		candidates[normalizeEmail(email)] = profile
	}
	for _, profile := range a.inviteStore.List() {
		candidates[normalizeEmail(profile.Email)] = profile
	}
	revoked := 0
	for email := range candidates {
		effective, ok := a.directoryProfile(email)
		if !ok {
			continue
		}
		member := effective.ForTenant(tenantSlug)
		if !member.HasPermission(permissionEnergyView) &&
			!member.HasPermission(permissionEnergyConfigure) &&
			!member.HasPermission(permissionEnergyControl) &&
			!member.HasPermission(permissionEnergyCaretaker) {
			continue
		}
		if envProfile, isEnv := a.profiles[normalizeEmail(email)]; isEnv {
			stored, hasStored := a.inviteStore.Get(email)
			if !hasStored || !stored.Adopted {
				override := envProfile
				override.Adopted = true
				if hasStored {
					if _, err := a.inviteStore.Update(email, override); err != nil {
						return revoked, err
					}
				} else if _, err := a.inviteStore.Add(override); err != nil {
					return revoked, err
				}
			}
		}
		_, found, err := a.inviteStore.MutateTenantPermissions(email, tenantSlug, func(current []string) []string {
			out := make([]string, 0, len(current))
			for _, permission := range current {
				switch permission {
				case permissionEnergyView, permissionEnergyConfigure, permissionEnergyControl, permissionEnergyCaretaker:
					continue
				default:
					out = append(out, permission)
				}
			}
			return out
		})
		if err != nil {
			return revoked, err
		}
		if found {
			revoked++
		}
	}
	return revoked, nil
}

func (a *app) buildEnergyDataPackage(ac authCtx, generatedAt time.Time) ([]byte, string, map[string]int, error) {
	profile, exists, err := a.energyStore.Profile(ac.tenant.Slug)
	if err != nil || !exists {
		return nil, "", nil, fmt.Errorf("energy profile unavailable")
	}
	assets, err := a.energyStore.ListAssets(ac.tenant.Slug)
	if err != nil {
		return nil, "", nil, err
	}
	mappings, err := a.energyStore.ListMappings(ac.tenant.Slug)
	if err != nil {
		return nil, "", nil, err
	}
	intervals, err := a.energyStore.ListIntervals(ac.tenant.Slug, time.Time{}, time.Time{})
	if err != nil {
		return nil, "", nil, err
	}
	imports, err := a.energyStore.ListImportsForExport(ac.tenant.Slug)
	if err != nil {
		return nil, "", nil, err
	}
	maintenance, err := a.energyStore.ListMaintenance(ac.tenant.Slug)
	if err != nil {
		return nil, "", nil, err
	}
	assessments, err := a.energyStore.ListTariffAssessments(ac.tenant.Slug)
	if err != nil {
		return nil, "", nil, err
	}
	measures, err := a.energyStore.ListMeasures(ac.tenant.Slug)
	if err != nil {
		return nil, "", nil, err
	}
	connectorReadings := []store.HomeConnectorReading{}
	if a.homeConnectorReadings != nil {
		connectorReadings, err = a.homeConnectorReadings.List(ac.tenant.Slug)
		if err != nil {
			return nil, "", nil, err
		}
	}
	totalRawBytes := 0
	for _, item := range imports {
		totalRawBytes += len(item.Payload)
	}
	if totalRawBytes > maxEnergyExportBytes {
		return nil, "", nil, fmt.Errorf("export too large")
	}

	generatedAt = generatedAt.UTC().Truncate(time.Second)
	intervalCSV, err := energyIntervalsCSV(intervals)
	if err != nil {
		return nil, "", nil, err
	}
	rawFiles := make([]energyExportFile, 0, len(imports))
	importMetadata := make([]map[string]any, 0, len(imports))
	zipFiles := map[string][]byte{
		"viertelstundenwerte.csv": intervalCSV,
	}
	for index, item := range imports {
		name := safeFilenamePart(filepath.Base(item.Filename))
		if name == "person" {
			name = "smart-meter.csv"
		}
		path := fmt.Sprintf("rohimporte/%02d-%s", index+1, name)
		digest := sha256.Sum256(item.Payload)
		zipFiles[path] = append([]byte(nil), item.Payload...)
		rawFiles = append(rawFiles, energyExportFile{
			Path:   path,
			SHA256: hex.EncodeToString(digest[:]),
			Bytes:  len(item.Payload),
		})
		importMetadata = append(importMetadata, map[string]any{
			"id":                item.ID,
			"original_filename": item.Filename,
			"sha256":            item.SHA256,
			"format":            item.Format,
			"imported_at":       item.ImportedAt.UTC().Format(time.RFC3339),
			"raw_file":          path,
		})
	}

	energyEvents := []map[string]any{}
	if a.auditStore != nil {
		events, err := a.auditStore.ListRetained(ac.tenant.Slug)
		if err != nil {
			return nil, "", nil, err
		}
		for _, event := range events {
			if strings.HasPrefix(event.Action, "energy.") {
				actor := "Andere berechtigte Person"
				if normalizeEmail(event.ActorEmail) == normalizeEmail(ac.email) {
					actor = "Sie"
				}
				energyEvents = append(energyEvents, map[string]any{
					"at":          event.At.UTC().Format(time.RFC3339),
					"action":      event.Action,
					"actor":       actor,
					"summary":     event.Summary,
					"target_type": event.TargetType,
				})
			}
		}
		sort.Slice(energyEvents, func(i, j int) bool {
			left, _ := energyEvents[i]["at"].(string)
			right, _ := energyEvents[j]["at"].(string)
			return left < right
		})
	}
	recommendation := energy.NextRecommendation(profile, assets, mappings, intervals)
	if maintenanceRecommendation, ok := energy.MaintenanceRecommendation(generatedAt, maintenance); ok {
		recommendation = maintenanceRecommendation
	}
	_, recordsItself := a.confirmedGridImportMapping(ac.tenant.Slug)
	tariff := buildEnergyTariffView(profile, intervals, recordsItself)
	metadata := map[string]any{
		"schema":       "https://hausv.org/schemas/energy-export/v1",
		"generated_at": generatedAt.Format(time.RFC3339),
		"home": map[string]any{
			"tenant":                profile.TenantSlug,
			"official_unit_id":      profile.UnitID,
			"type":                  profile.HomeType,
			"display_name":          profile.HouseholdName,
			"operating_mode":        profile.OperatingMode,
			"automation_stage":      profile.AutomationStage,
			"onboarding_step":       profile.OnboardingStep,
			"onboarding_complete":   profile.OnboardingComplete,
			"target_peak_kw":        profile.TargetPeakKW,
			"agreed_power_kw":       profile.AgreedPowerKW,
			"recommendation_id":     profile.RecommendationID,
			"recommendation_status": profile.RecommendationStatus,
			"free_started_at":       exportTime(profile.FreeStartedAt),
			"created_at":            profile.CreatedAt.UTC().Format(time.RFC3339),
			"updated_at":            profile.UpdatedAt.UTC().Format(time.RFC3339),
		},
		"assets":                   exportEnergyAssets(assets),
		"home_assistant_mappings":  exportEnergyMappings(mappings),
		"connector_sensor_catalog": exportHomeConnectorReadings(connectorReadings),
		"smart_meter_imports":      importMetadata,
		"maintenance":              exportEnergyMaintenance(maintenance),
		"tariff_assessments":       exportEnergyAssessments(assessments),
		"measures":                 exportEnergyMeasures(measures),
		"technical_access":         exportEnergyCaretakers(a.energyCaretakerViews(ac)),
		"current_recommendation":   recommendation,
		"current_tariff_model": map[string]any{
			"id":         tariff.ID,
			"version":    tariff.Version,
			"status":     tariff.Status,
			"source_url": tariff.SourceURL,
			"rule":       tariff.Rule,
			"disclaimer": tariff.Disclaimer,
		},
		"energy_audit_live": energyEvents,
		"source_notes": []string{
			"Home Assistant wird nur gelesen. Der Export enthält den begrenzten Sensorkatalog des lokalen Connectors und für ausgewählte Sensoren den zuletzt empfangenen Wert; direkte Home-Assistant-Zustände und Verläufe bleiben transient.",
			"Viertelstundenwerte mit der Quelle \"home-assistant\" stammen aus laufend gelesenen Momentanwerten des bestätigten Netzbezugs. Sie sind aus Abtastungen verdichtet und deshalb höchstens mit der Güte \"estimated\" ausgewiesen; \"gap\" und \"stale\" benennen fehlende oder eingefrorene Messwerte, statt über sie hinwegzumitteln.",
			"Der Export enthält bestätigte Entity-Zuordnungen, aber weder Home-Assistant-Adresse noch Zugangstoken.",
			"Smart-Meter-Rohdateien stammen aus den vom Nutzer hochgeladenen Originalen.",
			"Empfehlungen sind Hinweise; HAUSV trifft keine ausschließlich automatisierte Entscheidung mit rechtlicher oder ähnlich erheblicher Wirkung.",
		},
	}
	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, "", nil, err
	}
	metadataJSON = append(metadataJSON, '\n')
	zipFiles["energiedaten.json"] = metadataJSON

	files := []energyExportFile{}
	for path, payload := range zipFiles {
		digest := sha256.Sum256(payload)
		files = append(files, energyExportFile{Path: path, SHA256: hex.EncodeToString(digest[:]), Bytes: len(payload)})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	manifest := map[string]any{
		"schema":       "https://hausv.org/schemas/energy-export-manifest/v1",
		"generated_at": generatedAt.Format(time.RFC3339),
		"tenant":       ac.tenant.Slug,
		"format":       "ZIP with JSON, CSV and retained source files",
		"retention": map[string]string{
			"smart_meter_originals":    "30 Tage",
			"quarter_hour_values":      "13 Monate",
			"derived_assessments":      "3 Jahre",
			"profile_and_mappings":     "bis zur Korrektur, Trennung oder Löschung des Energieprofils",
			"connector_sensor_catalog": "bis zum Widerruf der Verbindung oder zur Löschung des Energieprofils; ausgewählte Werte werden mit jeder Meldung ersetzt",
			"energy_audit":             "höchstens 3 Jahre; Sicherheitsnachweise bleiben von der Selbstlöschung getrennt",
		},
		"not_included": []string{
			"Home-Assistant-Endpunkt und Zugangstoken",
			"nur transient gelesene Home-Assistant-Zustände und -Historie",
			"verschlüsselte Sicherungskopien; diese laufen nach dem betrieblichen Backup-Zyklus aus",
		},
		"files":     files,
		"raw_files": rawFiles,
	}
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, "", nil, err
	}
	manifestJSON = append(manifestJSON, '\n')

	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	if err := writeEnergyZipFile(writer, "manifest.json", manifestJSON); err != nil {
		return nil, "", nil, err
	}
	paths := make([]string, 0, len(zipFiles))
	for path := range zipFiles {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if err := writeEnergyZipFile(writer, path, zipFiles[path]); err != nil {
			return nil, "", nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", nil, err
	}
	filename := "hausv-energiedaten-" + safeFilenamePart(ac.tenant.Slug) + "-" + generatedAt.Format("20060102") + ".zip"
	counts := map[string]int{
		"assets":             len(assets),
		"mappings":           len(mappings),
		"intervals":          len(intervals),
		"raw_imports":        len(imports),
		"connector_readings": len(connectorReadings),
	}
	return output.Bytes(), filename, counts, nil
}

func exportHomeConnectorReadings(items []store.HomeConnectorReading) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"entity_id": item.EntityID, "state": item.State, "display_name": item.DisplayName,
			"unit": item.Unit, "device_class": item.DeviceClass, "state_class": item.StateClass,
			"last_updated": item.LastUpdated.UTC().Format(time.RFC3339),
			"received_at":  item.ReceivedAt.UTC().Format(time.RFC3339),
		})
	}
	return out
}

func energyIntervalsCSV(items []energy.Interval) ([]byte, error) {
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	if err := writer.Write([]string{"starts_at", "duration_minutes", "import_kwh", "average_kw", "quality", "source"}); err != nil {
		return nil, err
	}
	for _, item := range items {
		if err := writer.Write([]string{
			item.StartsAt.UTC().Format(time.RFC3339),
			strconv.Itoa(int(item.Duration / time.Minute)),
			strconv.FormatFloat(item.ImportKWh, 'f', -1, 64),
			strconv.FormatFloat(item.AverageKW, 'f', -1, 64),
			item.Quality,
			item.Source,
		}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return output.Bytes(), writer.Error()
}

func writeEnergyZipFile(writer *zip.Writer, path string, payload []byte) error {
	header := &zip.FileHeader{Name: path, Method: zip.Deflate}
	header.SetModTime(time.Unix(0, 0).UTC())
	target, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = target.Write(payload)
	return err
}

func exportEnergyAssets(items []energy.Asset) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"id": item.ID, "kind": item.Kind, "name": item.Name, "rated_power_kw": item.RatedPowerKW,
			"flexibility": item.Flexibility, "source": item.Source, "confirmed": item.Confirmed,
			"metadata":   exportSafeEnergyAssetMetadata(item.Metadata),
			"created_at": item.CreatedAt.UTC().Format(time.RFC3339), "updated_at": item.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out
}

func exportSafeEnergyAssetMetadata(input map[string]string) map[string]string {
	// Metadata is intentionally opt-in. Future adapters may attach credentials
	// under names a denylist does not yet know; an export must never learn those
	// fields merely because their key is novel.
	allowed := map[string]struct{}{
		"manufacturer":      {},
		"model":             {},
		"model_name":        {},
		"serial_number":     {},
		"installation_date": {},
		"commissioned_at":   {},
	}
	out := map[string]string{}
	for rawKey, value := range input {
		key := strings.ToLower(strings.TrimSpace(rawKey))
		if _, ok := allowed[key]; !ok {
			continue
		}
		out[key] = value
	}
	return out
}

func exportEnergyMappings(items []energy.EntityMapping) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"id": item.ID, "entity_id": item.EntityID, "asset_id": item.AssetID, "metric": item.Metric,
			"display_name": item.DisplayName, "unit": item.Unit, "device_class": item.DeviceClass,
			"confirmed": item.Confirmed, "last_seen_at": exportTime(item.LastSeenAt),
			"created_at": item.CreatedAt.UTC().Format(time.RFC3339), "updated_at": item.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out
}

func exportEnergyMaintenance(items []energy.MaintenancePlan) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"id": item.ID, "asset_id": item.AssetID, "title": item.Title, "interval_months": item.IntervalMonths,
			"last_completed_at": exportTime(item.LastCompletedAt), "next_due_at": item.NextDueAt.UTC().Format(time.RFC3339),
			"contact_id": item.ContactID, "document_id": item.DocumentID, "issue_id": item.IssueID,
			"evidence_note": item.EvidenceNote, "active": item.Active,
		})
	}
	return out
}

func exportEnergyAssessments(items []energy.TariffAssessment) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"id": item.ID, "month": item.AssessmentMonth, "profile_id": item.ProfileID,
			"profile_version": item.ProfileVersion, "profile_status": item.ProfileStatus, "source_url": item.SourceURL,
			"peak_kw": item.PeakKW, "billed_kw": item.BilledKW, "annual_power_eur": item.AnnualPowerEUR,
			"data_quality": item.DataQuality, "created_at": item.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out
}

func exportEnergyMeasures(items []energy.Measure) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"id": item.ID, "issue_id": item.IssueID, "recommendation_id": item.RecommendationID, "title": item.Title,
			"status": item.Status, "contact_id": item.ContactID, "shared_fields": item.SharedFields,
			"offer_note": item.OfferNote, "appointment_at": exportTime(item.AppointmentAt), "work_note": item.WorkNote,
			"completed_at": exportTime(item.CompletedAt), "evidence_note": item.EvidenceNote,
			"before_from": exportTime(item.BeforeFrom), "before_to": exportTime(item.BeforeTo),
			"after_from": exportTime(item.AfterFrom), "after_to": exportTime(item.AfterTo),
			"before_peak_kw": item.BeforePeakKW, "after_peak_kw": item.AfterPeakKW,
			"before_quality": item.BeforeQuality, "after_quality": item.AfterQuality,
			"created_at": item.CreatedAt.UTC().Format(time.RFC3339), "updated_at": item.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out
}

func exportEnergyCaretakers(items []energyCaretakerView) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if !item.CanView && !item.CanConfigure {
			continue
		}
		out = append(out, map[string]any{
			"email": item.Email, "name": item.Name, "can_view": item.CanView, "can_configure": item.CanConfigure,
		})
	}
	return out
}

func exportTime(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}

package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/homeassistant"
	"github.com/inspr-at/hausv-org/internal/store"
)

type blockingEnergyImportStoreHAUSV410 struct {
	energy.Storage
	started chan struct{}
	release <-chan struct{}
	once    *sync.Once
}

func (s *blockingEnergyImportStoreHAUSV410) PutImport(record energy.ImportRecord, intervals []energy.Interval) (bool, error) {
	s.once.Do(func() { close(s.started) })
	<-s.release
	return s.Storage.PutImport(record, intervals)
}

// ForTenant and ForHome have to re-wrap, or scoping quietly UNWRAPS this
// double: the embedded Storage returns its own scoped copy, the handler holds
// that instead, and the block this whole test is built on never happens. The
// signals are shared rather than copied so a scoped clone still reports to the
// test that is waiting on them.
func (s *blockingEnergyImportStoreHAUSV410) ForTenant(tenant store.TenantRef) energy.Storage {
	return &blockingEnergyImportStoreHAUSV410{
		Storage: s.Storage.ForTenant(tenant), started: s.started, release: s.release, once: s.once,
	}
}

func (s *blockingEnergyImportStoreHAUSV410) ForHome(homeKey string) energy.Storage {
	return &blockingEnergyImportStoreHAUSV410{
		Storage: s.Storage.ForHome(homeKey), started: s.started, release: s.release, once: s.once,
	}
}

type blockingEnergyProfileDeleteStoreHAUSV410 struct {
	energy.Storage
	started chan struct{}
	release <-chan struct{}
	once    *sync.Once
}

func (s *blockingEnergyProfileDeleteStoreHAUSV410) DeleteProfile(tenantSlug string) (energy.DeleteSummary, error) {
	s.once.Do(func() { close(s.started) })
	<-s.release
	return s.Storage.DeleteProfile(tenantSlug)
}

func (s *blockingEnergyProfileDeleteStoreHAUSV410) ForTenant(tenant store.TenantRef) energy.Storage {
	return &blockingEnergyProfileDeleteStoreHAUSV410{
		Storage: s.Storage.ForTenant(tenant), started: s.started, release: s.release, once: s.once,
	}
}

func (s *blockingEnergyProfileDeleteStoreHAUSV410) ForHome(homeKey string) energy.Storage {
	return &blockingEnergyProfileDeleteStoreHAUSV410{
		Storage: s.Storage.ForHome(homeKey), started: s.started, release: s.release, once: s.once,
	}
}

func TestEnergyProfileDeletionWaitsForInFlightSmartMeterImport(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})
	saveClaimedEnergyProfileHAUSV410(t, a, energy.HomeHouse)

	importStarted := make(chan struct{})
	importRelease := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(importRelease) }) })
	a.energyStore = &blockingEnergyImportStoreHAUSV410{
		Storage: a.energyStore,
		started: importStarted,
		release: importRelease,
		once:    &sync.Once{},
	}

	importRequest := newAuthedEnergyMultipartRequestHAUSV410(
		t,
		a,
		"/demo/app/energie/smart-meter",
		"smart_meter_file",
		"smart-meter.csv",
		[]byte("timestamp;import_kwh\n2026-07-01T00:00:00+02:00;0,42\n2026-07-01T00:15:00+02:00;0,38\n"),
	)
	deleteRequest := newAuthedEnergyFormRequestHAUSV410(
		t,
		a,
		"/demo/app/settings/energy-data/profile/delete",
		url.Values{"confirmation": {"ENERGIEPROFIL LÖSCHEN"}},
	)
	handler := a.handler()

	importDone := serveEnergyRequestHAUSV410(handler, importRequest)
	awaitEnergySignalHAUSV410(t, importStarted, "Smart-Meter import did not reach storage")

	deleteDone := serveEnergyRequestHAUSV410(handler, deleteRequest)
	awaitEnergyLifecycleWriterHAUSV410(t, a, "demo")
	select {
	case result := <-deleteDone:
		t.Fatalf("profile deletion completed while Smart-Meter import was in flight: status=%d body=%s", result.Code, result.Body.String())
	default:
	}

	releaseOnce.Do(func() { close(importRelease) })
	importResult := awaitEnergyResponseHAUSV410(t, importDone, "Smart-Meter import")
	deleteResult := awaitEnergyResponseHAUSV410(t, deleteDone, "energy-profile deletion")
	if importResult.Code != http.StatusSeeOther {
		t.Fatalf("Smart-Meter import status = %d body=%s", importResult.Code, importResult.Body.String())
	}
	if deleteResult.Code != http.StatusSeeOther {
		t.Fatalf("profile deletion status = %d body=%s", deleteResult.Code, deleteResult.Body.String())
	}

	profile, exists, err := a.energyStore.Profile("demo")
	if err != nil || !exists || !energyProfileUnclaimed(profile) {
		t.Fatalf("profile after raced deletion: exists=%v profile=%+v err=%v", exists, profile, err)
	}
	imports, err := a.energyStore.ListImports("demo")
	if err != nil || len(imports) != 0 {
		t.Fatalf("Smart-Meter imports survived raced deletion: imports=%+v err=%v", imports, err)
	}
	intervals, err := a.energyStore.ListIntervals("demo", time.Time{}, time.Time{})
	if err != nil || len(intervals) != 0 {
		t.Fatalf("Smart-Meter intervals survived raced deletion: intervals=%+v err=%v", intervals, err)
	}
}

func TestEnergyMutationQueuedBehindProfileDeletionCannotRecreateData(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})
	saveClaimedEnergyProfileHAUSV410(t, a, energy.HomeHouse)

	deleteStarted := make(chan struct{})
	deleteRelease := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(deleteRelease) }) })
	a.energyStore = &blockingEnergyProfileDeleteStoreHAUSV410{
		Storage: a.energyStore,
		started: deleteStarted,
		release: deleteRelease,
		once:    &sync.Once{},
	}

	deleteRequest := newAuthedEnergyFormRequestHAUSV410(
		t,
		a,
		"/demo/app/settings/energy-data/profile/delete",
		url.Values{"confirmation": {"ENERGIEPROFIL LÖSCHEN"}},
	)
	importRequest := newAuthedEnergyMultipartRequestHAUSV410(
		t,
		a,
		"/demo/app/energie/smart-meter",
		"smart_meter_file",
		"queued-smart-meter.csv",
		[]byte("timestamp;import_kwh\n2026-07-01T00:00:00+02:00;0,42\n2026-07-01T00:15:00+02:00;0,38\n"),
	)
	handler := a.handler()

	deleteDone := serveEnergyRequestHAUSV410(handler, deleteRequest)
	awaitEnergySignalHAUSV410(t, deleteStarted, "profile deletion did not reach storage")

	importDone := serveEnergyRequestHAUSV410(handler, importRequest)
	select {
	case result := <-importDone:
		t.Fatalf("normal energy POST crossed an in-flight profile deletion: status=%d body=%s", result.Code, result.Body.String())
	case <-time.After(100 * time.Millisecond):
	}

	releaseOnce.Do(func() { close(deleteRelease) })
	deleteResult := awaitEnergyResponseHAUSV410(t, deleteDone, "energy-profile deletion")
	importResult := awaitEnergyResponseHAUSV410(t, importDone, "queued Smart-Meter import")
	if deleteResult.Code != http.StatusSeeOther {
		t.Fatalf("profile deletion status = %d body=%s", deleteResult.Code, deleteResult.Body.String())
	}
	if importResult.Code != http.StatusConflict {
		t.Fatalf("queued import status = %d, want %d after profile deletion; body=%s", importResult.Code, http.StatusConflict, importResult.Body.String())
	}

	profile, exists, err := a.energyStore.Profile("demo")
	if err != nil || !exists || !energyProfileUnclaimed(profile) {
		t.Fatalf("profile after queued write: exists=%v profile=%+v err=%v", exists, profile, err)
	}
	imports, err := a.energyStore.ListImports("demo")
	if err != nil || len(imports) != 0 {
		t.Fatalf("queued import recreated data: imports=%+v err=%v", imports, err)
	}
	intervals, err := a.energyStore.ListIntervals("demo", time.Time{}, time.Time{})
	if err != nil || len(intervals) != 0 {
		t.Fatalf("queued import recreated intervals: intervals=%+v err=%v", intervals, err)
	}
}

func TestEnergyProfileDeletionCannotBeFollowedByInFlightChartCacheRefill(t *testing.T) {
	historyStarted := make(chan struct{})
	historyRelease := make(chan struct{})
	var historyStartedOnce sync.Once
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(historyRelease) }) })

	now := time.Now().UTC()
	haServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/states/sensor.house_power":
			_ = json.NewEncoder(w).Encode(homeassistant.EntityState{
				EntityID: "sensor.house_power",
				State:    "1250",
				Attributes: map[string]any{
					"friendly_name":       "Hausverbrauch",
					"device_class":        "power",
					"unit_of_measurement": "W",
				},
				LastUpdated: now,
			})
		case strings.HasPrefix(r.URL.Path, "/api/history/period/"):
			historyStartedOnce.Do(func() { close(historyStarted) })
			<-historyRelease
			_ = json.NewEncoder(w).Encode([][]homeassistant.HistoryState{{
				{
					EntityID:    "sensor.house_power",
					State:       "900",
					LastUpdated: now.Add(-30 * time.Minute),
				},
				{
					EntityID:    "sensor.house_power",
					State:       "1250",
					LastUpdated: now.Add(-15 * time.Minute),
				},
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(haServer.Close)

	a := newTestPortalApp(t, userProfile{
		Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})
	saveClaimedEnergyProfileHAUSV410(t, a, energy.HomeHouse)
	if err := a.energyStore.UpsertMapping(energy.EntityMapping{
		ID:          "mapping-house-power",
		TenantSlug:  "demo",
		EntityID:    "sensor.house_power",
		Metric:      energy.MetricLoadPower,
		DisplayName: "Hausverbrauch",
		Unit:        "W",
		Confirmed:   true,
	}); err != nil {
		t.Fatal(err)
	}
	tenant := a.tenants["demo"]
	tenant.HA = homeassistant.NewConfig(haServer.URL, "fixture", "", "", "")
	a.tenants["demo"] = tenant

	cockpitRequest := newAuthedEnergyGETRequestHAUSV410(t, a, "/demo/app/energie")
	deleteRequest := newAuthedEnergyFormRequestHAUSV410(
		t,
		a,
		"/demo/app/settings/energy-data/profile/delete",
		url.Values{"confirmation": {"ENERGIEPROFIL LÖSCHEN"}},
	)
	handler := a.handler()

	cockpitDone := serveEnergyRequestHAUSV410(handler, cockpitRequest)
	awaitEnergySignalHAUSV410(t, historyStarted, "energy chart did not request Home Assistant history")

	deleteDone := serveEnergyRequestHAUSV410(handler, deleteRequest)
	awaitEnergyLifecycleWriterHAUSV410(t, a, "demo")
	releaseOnce.Do(func() { close(historyRelease) })

	cockpitResult := awaitEnergyResponseHAUSV410(t, cockpitDone, "energy cockpit")
	deleteResult := awaitEnergyResponseHAUSV410(t, deleteDone, "energy-profile deletion")
	if cockpitResult.Code != http.StatusOK {
		t.Fatalf("energy cockpit status = %d body=%s", cockpitResult.Code, cockpitResult.Body.String())
	}
	if deleteResult.Code != http.StatusSeeOther {
		t.Fatalf("profile deletion status = %d body=%s", deleteResult.Code, deleteResult.Body.String())
	}

	a.energyChartMu.Lock()
	defer a.energyChartMu.Unlock()
	for key := range a.energyChartCache {
		if strings.HasPrefix(key, "demo|") {
			t.Fatalf("in-flight Home Assistant history refilled deleted tenant cache: key=%q", key)
		}
	}
}

func TestEnergyLifecycleLocksAreAppScoped(t *testing.T) {
	first := &app{}
	second := &app{}
	if first.energyLifecycleLock("demo") == second.energyLifecycleLock("demo") {
		t.Fatal("independent app instances share a tenant lifecycle lock")
	}
}

func newAuthedEnergyGETRequestHAUSV410(t *testing.T, a *app, path string) *http.Request {
	t.Helper()
	token, _, err := a.sessions.Put("owner@example.com", "demo", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://hausv.org"+path, nil)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	return req
}

func newAuthedEnergyFormRequestHAUSV410(t *testing.T, a *app, path string, values url.Values) *http.Request {
	t.Helper()
	token, _, err := a.sessions.Put("owner@example.com", "demo", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://hausv.org"+path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://hausv.org/demo")
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	return req
}

func newAuthedEnergyMultipartRequestHAUSV410(t *testing.T, a *app, path, field, filename string, payload []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart body: %v", err)
	}
	token, _, err := a.sessions.Put("owner@example.com", "demo", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://hausv.org"+path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", "http://hausv.org/demo")
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	return req
}

func serveEnergyRequestHAUSV410(handler http.Handler, request *http.Request) <-chan *httptest.ResponseRecorder {
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		done <- recorder
	}()
	return done
}

func awaitEnergySignalHAUSV410(t *testing.T, signal <-chan struct{}, failure string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatal(failure)
	}
}

func awaitEnergyLifecycleWriterHAUSV410(t *testing.T, a *app, tenantSlug string) {
	t.Helper()
	lock := a.energyLifecycleLock(tenantSlug)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !lock.TryRLock() {
			return
		}
		lock.RUnlock()
		time.Sleep(time.Millisecond)
	}
	t.Fatal("destructive energy request did not enter the tenant lifecycle barrier")
}

func awaitEnergyResponseHAUSV410(t *testing.T, done <-chan *httptest.ResponseRecorder, label string) *httptest.ResponseRecorder {
	t.Helper()
	select {
	case result := <-done:
		return result
	case <-time.After(3 * time.Second):
		t.Fatalf("%s did not complete", label)
		return nil
	}
}

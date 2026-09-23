package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// ParkingStorage is the parking/charging backend. JSON remains for tests and
// the one-shot boot import; the product path is SQL (HAUSV-758).
type ParkingStorage interface {
	TenantData(tenantSlug string) ParkingTenantData
	SetGridFee(tenantSlug string, gridFee float64) error
	UpsertTariff(tenantSlug string, tariff ParkingTariff) error
	SetMonthPaid(tenantSlug string, month string, paid bool) error
	SetMonthPayment(tenantSlug string, month string, state ParkingMonthState) error
	MarkPaymentReminderSent(tenantSlug string, months []string, recipients []string, at time.Time) error
	AppendSamples(tenantSlug string, samples []ParkingStoredSample) error
	AppendReadings(tenantSlug string, energySamples []ParkingNumericSample, priceSamples []ParkingNumericSample) error
	StartChargingSession(tenantSlug string, session ChargingSession, state ChargingControllerState) (ChargingSession, error)
	EndChargingSession(tenantSlug, sessionID string, end time.Time, endKWh float64, endedBy, endReason string, state ChargingControllerState) (ChargingSession, bool, error)
	SetChargingState(tenantSlug string, state ChargingControllerState) error
	SetChargingControl(tenantSlug string, settings ChargingControlSettings) error
}

var (
	_ ParkingStorage = (*ParkingStore)(nil)
	_ ParkingStorage = (*SQLParkingStore)(nil)
)

// SQLParkingStore keeps one JSON document per tenant in table parking.
type SQLParkingStore struct {
	db *TenantDB
}

func NewSQLParkingStore(db *TenantDB) *SQLParkingStore {
	return &SQLParkingStore{db: db}
}

func parkingSlug(tenantSlug string) string {
	slug := textutil.Slug(tenantSlug)
	if slug == "" {
		return "default"
	}
	return slug
}

func (s *SQLParkingStore) tenantRef(slug string) (TenantRef, error) {
	if s == nil || s.db == nil {
		return TenantRef{}, fmt.Errorf("parking store unavailable")
	}
	return newTenantIDCache(s.db.Unscoped("slug-to-tenant_id resolution reads the tenant registry, before there is an identity to scope to")).ref(parkingSlug(slug))
}

func loadParkingTx(tx *sql.Tx, tenant TenantRef) (ParkingTenantData, error) {
	initial, err := json.Marshal(DefaultParkingTenantData())
	if err != nil {
		return ParkingTenantData{}, err
	}
	// Acquire the writer lock BEFORE reading the document. The no-op conflict
	// update returns the latest committed value after a competing writer ends;
	// inserting also serializes the first write when no row exists yet. Both
	// PostgreSQL and the SQLite test backend hold this lock until commit/rollback.
	// A process-local mutex cannot protect independent store instances.
	var raw string
	err = tx.QueryRow(`INSERT INTO parking(tenant_id, tenant_slug, data) VALUES($1, $2, $3)
		ON CONFLICT(tenant_slug) DO UPDATE SET data=parking.data,
			tenant_id=coalesce(parking.tenant_id, excluded.tenant_id)
		RETURNING data`, tenant.ID, tenant.Slug, string(initial)).Scan(&raw)
	if err != nil {
		return ParkingTenantData{}, err
	}
	var data ParkingTenantData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return ParkingTenantData{}, fmt.Errorf("invalid parking data")
	}
	return data, nil
}

func writeParkingTx(tx *sql.Tx, tenant TenantRef, data ParkingTenantData) error {
	blob, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO parking(tenant_id, tenant_slug, data) VALUES($1, $2, $3)
		 ON CONFLICT(tenant_slug) DO UPDATE SET data=excluded.data,
		   tenant_id=coalesce(parking.tenant_id, excluded.tenant_id)`,
		tenant.ID, tenant.Slug, string(blob),
	)
	return err
}

func (s *SQLParkingStore) TenantData(tenantSlug string) ParkingTenantData {
	if s == nil {
		return DefaultParkingTenantData()
	}
	ref, err := s.tenantRef(tenantSlug)
	if err != nil {
		return DefaultParkingTenantData()
	}
	var raw string
	err = s.db.For(ref).QueryRow(`SELECT data FROM parking WHERE tenant_id=$1`, ref.ID).Scan(&raw)
	if err != nil {
		return DefaultParkingTenantData()
	}
	mem := &ParkingStore{data: ParkingStoreData{Tenants: map[string]ParkingTenantData{}}}
	var data ParkingTenantData
	if json.Unmarshal([]byte(raw), &data) != nil {
		return DefaultParkingTenantData()
	}
	mem.data.Tenants[ref.Slug] = data
	return mem.TenantData(ref.Slug)
}

func (s *SQLParkingStore) mutate(tenantSlug string, fn func(*ParkingStore) error) error {
	ref, err := s.tenantRef(tenantSlug)
	if err != nil {
		return err
	}
	tx, err := s.db.For(ref).Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	current, err := loadParkingTx(tx, ref)
	if err != nil {
		return err
	}
	mem := &ParkingStore{data: ParkingStoreData{Tenants: map[string]ParkingTenantData{ref.Slug: current}}}
	if err := fn(mem); err != nil {
		return err
	}
	if err := writeParkingTx(tx, ref, mem.TenantData(ref.Slug)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLParkingStore) SetGridFee(tenantSlug string, gridFee float64) error {
	return s.mutate(tenantSlug, func(mem *ParkingStore) error {
		return mem.SetGridFee(tenantSlug, gridFee)
	})
}

func (s *SQLParkingStore) UpsertTariff(tenantSlug string, tariff ParkingTariff) error {
	return s.mutate(tenantSlug, func(mem *ParkingStore) error {
		return mem.UpsertTariff(tenantSlug, tariff)
	})
}

func (s *SQLParkingStore) SetMonthPaid(tenantSlug string, month string, paid bool) error {
	return s.SetMonthPayment(tenantSlug, month, ParkingMonthState{Paid: paid})
}

func (s *SQLParkingStore) SetMonthPayment(tenantSlug string, month string, state ParkingMonthState) error {
	return s.mutate(tenantSlug, func(mem *ParkingStore) error {
		return mem.SetMonthPayment(tenantSlug, month, state)
	})
}

func (s *SQLParkingStore) MarkPaymentReminderSent(tenantSlug string, months []string, recipients []string, at time.Time) error {
	return s.mutate(tenantSlug, func(mem *ParkingStore) error {
		return mem.MarkPaymentReminderSent(tenantSlug, months, recipients, at)
	})
}

func (s *SQLParkingStore) AppendSamples(tenantSlug string, samples []ParkingStoredSample) error {
	return s.mutate(tenantSlug, func(mem *ParkingStore) error {
		return mem.AppendSamples(tenantSlug, samples)
	})
}

func (s *SQLParkingStore) AppendReadings(tenantSlug string, energySamples []ParkingNumericSample, priceSamples []ParkingNumericSample) error {
	return s.mutate(tenantSlug, func(mem *ParkingStore) error {
		return mem.AppendReadings(tenantSlug, energySamples, priceSamples)
	})
}

func (s *SQLParkingStore) StartChargingSession(tenantSlug string, session ChargingSession, state ChargingControllerState) (ChargingSession, error) {
	var out ChargingSession
	err := s.mutate(tenantSlug, func(mem *ParkingStore) error {
		created, err := mem.StartChargingSession(tenantSlug, session, state)
		out = created
		return err
	})
	return out, err
}

func (s *SQLParkingStore) EndChargingSession(tenantSlug, sessionID string, end time.Time, endKWh float64, endedBy, endReason string, state ChargingControllerState) (ChargingSession, bool, error) {
	var out ChargingSession
	var found bool
	err := s.mutate(tenantSlug, func(mem *ParkingStore) error {
		closed, ok, err := mem.EndChargingSession(tenantSlug, sessionID, end, endKWh, endedBy, endReason, state)
		out, found = closed, ok
		return err
	})
	return out, found, err
}

func (s *SQLParkingStore) SetChargingState(tenantSlug string, state ChargingControllerState) error {
	return s.mutate(tenantSlug, func(mem *ParkingStore) error {
		return mem.SetChargingState(tenantSlug, state)
	})
}

func (s *SQLParkingStore) SetChargingControl(tenantSlug string, settings ChargingControlSettings) error {
	return s.mutate(tenantSlug, func(mem *ParkingStore) error {
		return mem.SetChargingControl(tenantSlug, settings)
	})
}

// ImportParking copies JSON parking tenants that are not yet in SQL.
func (s *SQLParkingStore) ImportParking(src *ParkingStore) error {
	if s == nil || src == nil {
		return nil
	}
	src.mu.Lock()
	snapshot := make(map[string]ParkingTenantData, len(src.data.Tenants))
	for slug, data := range src.data.Tenants {
		snapshot[slug] = data
	}
	src.mu.Unlock()
	imports := s.db.Unscoped("boot import replay of the JSON parking snapshot: it spans every tenant and runs before the first request")
	var n int
	if err := imports.QueryRow(`SELECT COUNT(*) FROM parking`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	tenants := newTenantIDCache(imports)
	for slug, data := range snapshot {
		ref, err := tenants.ref(slug)
		if err != nil {
			return err
		}
		blob, err := json.Marshal(data)
		if err != nil {
			return err
		}
		if _, err := imports.Exec(
			`INSERT INTO parking(tenant_id, tenant_slug, data) VALUES($1, $2, $3) ON CONFLICT(tenant_slug) DO NOTHING`,
			ref.ID, ref.Slug, string(blob),
		); err != nil {
			return err
		}
	}
	return nil
}

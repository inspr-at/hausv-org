package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

type OrgCounters struct {
	Approved int `json:"approved"`
	Edited   int `json:"edited"`
	Rejected int `json:"rejected"`
	Auto     int `json:"auto"`
}

type OrgSettings struct {
	Valorisation  ValorisationSettings `json:"valorisation"`
	Organisation  string               `json:"organisation"`
	Name          string               `json:"name"`
	TrustLevels   map[string]string    `json:"trust_levels"`
	AutoThreshold float64              `json:"auto_threshold"`
	AutoEnabled   bool                 `json:"auto_enabled"`
	AIProvider    string               `json:"ai_provider,omitempty"`
	AIBaseURL     string               `json:"ai_base_url,omitempty"`
	AIModel       string               `json:"ai_model,omitempty"`
	Counters      OrgCounters          `json:"counters"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

type OrgSettingsRepository interface {
	Get(context.Context) (OrgSettings, error)
	Save(context.Context, OrgSettings) error
	Increment(context.Context, string) error
}

type sqlOrgSettingsRepository struct {
	begin  func(context.Context, string) (*sql.Tx, error)
	orgKey string
}

func BindOrgSettingsRepository(database *sql.DB, orgKey string) OrgSettingsRepository {
	return &sqlOrgSettingsRepository{
		begin:  func(ctx context.Context, orgKey string) (*sql.Tx, error) { return beginOrgTx(ctx, database, orgKey) },
		orgKey: textutil.Slug(orgKey),
	}
}

func DefaultOrgSettings(orgKey string) OrgSettings {
	trust := make(map[string]string, len(IntakeCategories()))
	for _, category := range IntakeCategories() {
		trust[category.Key] = "propose"
	}
	return OrgSettings{Valorisation: DefaultValorisationSettings(), Organisation: textutil.Slug(orgKey), TrustLevels: trust, AutoThreshold: 0.9}
}

func normalizeOrgSettings(orgKey string, item OrgSettings) (OrgSettings, error) {
	defaults := DefaultOrgSettings(orgKey)
	item.Valorisation = item.Valorisation.Normalized()
	item.Organisation = defaults.Organisation
	item.Name = strings.TrimSpace(item.Name)
	item.AIProvider = strings.ToLower(strings.TrimSpace(item.AIProvider))
	if item.AIProvider != "cloud" && item.AIProvider != "local" {
		item.AIProvider = ""
	}
	item.AIBaseURL = strings.TrimSpace(item.AIBaseURL)
	item.AIModel = strings.TrimSpace(item.AIModel)
	for key, value := range item.TrustLevels {
		if value == "manual" || value == "propose" || value == "auto" {
			defaults.TrustLevels[key] = value
		}
	}
	item.TrustLevels = defaults.TrustLevels
	if item.AutoThreshold <= 0 || item.AutoThreshold > 1 {
		item.AutoThreshold = defaults.AutoThreshold
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = time.Now().UTC()
	} else {
		item.UpdatedAt = item.UpdatedAt.UTC()
	}
	if item.Organisation == "" {
		return OrgSettings{}, fmt.Errorf("organisation is required")
	}
	return item, nil
}

func (r *sqlOrgSettingsRepository) Get(ctx context.Context) (OrgSettings, error) {
	if r.begin == nil || r.orgKey == "" {
		return OrgSettings{}, fmt.Errorf("organisation settings repository is not bound")
	}
	var data string
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return OrgSettings{}, err
	}
	defer tx.Rollback()
	err = tx.QueryRowContext(ctx, `SELECT data FROM org_settings WHERE org_key=$1`, r.orgKey).Scan(&data)
	if err == sql.ErrNoRows {
		if err := tx.Commit(); err != nil {
			return OrgSettings{}, err
		}
		return DefaultOrgSettings(r.orgKey), nil
	}
	if err != nil {
		return OrgSettings{}, err
	}
	var item OrgSettings
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return OrgSettings{}, err
	}
	if err := tx.Commit(); err != nil {
		return OrgSettings{}, err
	}
	return normalizeOrgSettings(r.orgKey, item)
}

func (r *sqlOrgSettingsRepository) Save(ctx context.Context, item OrgSettings) error {
	if r.begin == nil || r.orgKey == "" {
		return fmt.Errorf("organisation settings repository is not bound")
	}
	item, err := normalizeOrgSettings(r.orgKey, item)
	if err != nil {
		return err
	}
	blob, err := json.Marshal(item)
	if err != nil {
		return err
	}
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO org_settings(org_key,data) VALUES($1,$2)
		ON CONFLICT(org_key) DO UPDATE SET data=excluded.data`, r.orgKey, string(blob))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *sqlOrgSettingsRepository) Increment(ctx context.Context, field string) error {
	item, err := r.Get(ctx)
	if err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "approved":
		item.Counters.Approved++
	case "edited":
		item.Counters.Edited++
	case "rejected":
		item.Counters.Rejected++
	case "auto":
		item.Counters.Auto++
	default:
		return fmt.Errorf("unknown organisation counter %q", field)
	}
	return r.Save(ctx, item)
}

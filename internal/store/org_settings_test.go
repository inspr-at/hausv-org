package store

import (
	"context"
	"testing"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

func TestOrgSettingsPersistsAIOverrides(t *testing.T) {
	database := dbtest.Open(t)
	repo := BindOrgSettingsRepository(database, "musterstadt")
	settings := DefaultOrgSettings("musterstadt")
	settings.AIProvider = "local"
	settings.AIBaseURL = "http://inference.internal/v1"
	settings.AIModel = "hausv-local"
	if err := repo.Save(context.Background(), settings); err != nil {
		t.Fatal(err)
	}

	got, err := repo.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.AIProvider != "local" || got.AIBaseURL != settings.AIBaseURL || got.AIModel != settings.AIModel {
		t.Fatalf("AI overrides = %#v", got)
	}
}

func TestOrgSettingsNormalizesUnknownAIProviderToEnvironment(t *testing.T) {
	settings, err := normalizeOrgSettings("musterstadt", OrgSettings{AIProvider: "unknown", AIBaseURL: " https://example.test/v1 ", AIModel: " model "})
	if err != nil {
		t.Fatal(err)
	}
	if settings.AIProvider != "" || settings.AIBaseURL != "https://example.test/v1" || settings.AIModel != "model" {
		t.Fatalf("normalized settings = %#v", settings)
	}
}

package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestWirksamwerdenSettingMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "timing.db")
	database := openBeforeMigration(t, path, "0065_wirksamwerden_modes.sql")
	for _, mode := range []string{"cautious", "contractual", "oevi"} {
		if _, err := database.Exec(`INSERT INTO org_settings(org_key,data) VALUES($1,json_object('name','Test','valorisation',json_object('wirksamwerden_mode',$1,'four_eyes',1)))`, mode); err != nil {
			t.Fatal(err)
		}
	}
	database.Close()
	database, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	for old, want := range map[string]string{"cautious": "wko", "contractual": "contract", "oevi": "oevi"} {
		var mode, name string
		var fourEyes int
		if err := database.QueryRow(`SELECT json_extract(data,'$.valorisation.wirksamwerden_mode'),json_extract(data,'$.name'),json_extract(data,'$.valorisation.four_eyes') FROM org_settings WHERE org_key=$1`, old).Scan(&mode, &name, &fourEyes); err != nil {
			t.Fatal(err)
		}
		if mode != want || name != "Test" || fourEyes != 1 {
			t.Fatalf("%s: %s %s %d", old, mode, name, fourEyes)
		}
	}
}

func TestPostgresWirksamwerdenSettingMigration(t *testing.T) {
	dsn := os.Getenv("HAUSV_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("disposable PostgreSQL required")
	}
	database, err := sql.Open("pgx", isolatedPostgresSchema(t, dsn))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	entries, err := postgresMigrationsFS.ReadDir("postgres/migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() >= "0038_wirksamwerden_modes.sql" {
			break
		}
		raw, err := postgresMigrationsFS.ReadFile("postgres/migrations/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err = database.Exec(string(raw)); err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
	}
	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`SET LOCAL hausv.cross_tenant='on'; INSERT INTO org_settings(org_key,data) VALUES('old-wko','{"name":"One","valorisation":{"wirksamwerden_mode":"cautious","four_eyes":true}}'),('old-contract','{"name":"Two","valorisation":{"wirksamwerden_mode":"contractual"}}')`); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	tx, err = database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := postgresMigrationsFS.ReadFile("postgres/migrations/0038_wirksamwerden_modes.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(string(raw)); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	tx, err = database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`SET LOCAL hausv.cross_tenant='on'`); err != nil {
		t.Fatal(err)
	}
	for org, want := range map[string]string{"old-wko": "wko", "old-contract": "contract"} {
		var mode string
		if err = tx.QueryRow(`SELECT data::jsonb #>> '{valorisation,wirksamwerden_mode}' FROM org_settings WHERE org_key=$1`, org).Scan(&mode); err != nil {
			t.Fatal(err)
		}
		if mode != want {
			t.Fatalf("%s: %s, want %s", org, mode, want)
		}
	}
}

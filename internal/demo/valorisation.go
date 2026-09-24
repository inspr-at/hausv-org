package demo

import (
	"context"
	"database/sql"
	"fmt"
)

// resetValorisation is limited to the explicit clean demo reset. Trigger DDL
// and record deletion share its transaction, so failure restores every guard.
func resetValorisation(ctx context.Context, tx *sql.Tx, houses []seedHouse, postgres bool) error {
	names := []struct{ table, name string }{{"valorisation_runs", "valorisation_run_frozen"}, {"valorisation_items", "valorisation_item_frozen"}}
	var restore []string
	if postgres {
		for _, n := range names {
			if _, err := tx.ExecContext(ctx, `ALTER TABLE `+n.table+` DISABLE TRIGGER `+n.name); err != nil {
				return err
			}
			restore = append(restore, `ALTER TABLE `+n.table+` ENABLE TRIGGER `+n.name)
		}
	} else {
		rows, err := tx.QueryContext(ctx, `SELECT name,sql FROM sqlite_master WHERE type='trigger' AND name IN ('valorisation_run_frozen','valorisation_item_frozen','valorisation_run_no_delete','valorisation_item_no_delete')`)
		if err != nil {
			return err
		}
		var triggers []string
		for rows.Next() {
			var name, sql string
			if err = rows.Scan(&name, &sql); err != nil {
				rows.Close()
				return err
			}
			triggers = append(triggers, name)
			restore = append(restore, sql)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		for _, name := range triggers {
			if _, err = tx.ExecContext(ctx, `DROP TRIGGER `+name); err != nil {
				return err
			}
		}
	}
	for _, house := range houses {
		for _, table := range []string{"valorisation_events", "valorisation_deliveries", "valorisation_items", "valorisation_runs"} {
			if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE tenant_slug=$1`, house.Slug); err != nil {
				return fmt.Errorf("reset demo valorisation: %w", err)
			}
		}
	}
	for _, query := range restore {
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	return nil
}

package postgres

import (
	"context"
	"fmt"

	"cloud-backend/internal/cloudsync/contracts"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EnsureCurrencyReferenceCatalog upserts canonical active ISO 4217 catalog into cloud_currency_reference.
func EnsureCurrencyReferenceCatalog(ctx context.Context, pool *pgxpool.Pool, profiles []contracts.CurrencyProfile) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE cloud_currency_reference SET active = FALSE, updated_at = now()`); err != nil {
		return err
	}

	for _, profile := range profiles {
		if _, err := tx.Exec(ctx, `
INSERT INTO cloud_currency_reference(
  currency_code,currency_alpha_code,minor_unit,currency_iso_name,currency_symbol,curr_basic_name,curr_add_name,show_add,show_currency_basic_name,active,updated_at
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,TRUE,now())
ON CONFLICT(currency_code) DO UPDATE SET
  currency_alpha_code = EXCLUDED.currency_alpha_code,
  minor_unit = EXCLUDED.minor_unit,
  currency_iso_name = EXCLUDED.currency_iso_name,
  currency_symbol = EXCLUDED.currency_symbol,
  curr_basic_name = EXCLUDED.curr_basic_name,
  curr_add_name = EXCLUDED.curr_add_name,
  show_add = EXCLUDED.show_add,
  show_currency_basic_name = EXCLUDED.show_currency_basic_name,
  active = TRUE,
  updated_at = now()
`, profile.CurrencyCode, profile.CurrencyAlphaCode, profile.MinorUnit, profile.CurrencyIsoName, profile.CurrencySymbol, profile.CurrBasicName, profile.CurrAddName, profile.ShowAdd, profile.ShowCurrencyBasicName); err != nil {
			return fmt.Errorf("upsert currency profile %s: %w", profile.CurrencyAlphaCode, err)
		}
	}

	return tx.Commit(ctx)
}

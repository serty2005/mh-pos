package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateCheck(ctx context.Context, v *domain.Check) error {
	snapshot := v.Snapshot
	if len(snapshot) == 0 {
		snapshot = json.RawMessage(`{}`)
	}
	if v.CurrencyCode == "" {
		v.CurrencyCode = "RUB"
	}
	if v.RemainingTotal == 0 && v.Total >= v.PaidTotal {
		v.RemainingTotal = v.Total - v.PaidTotal
	}
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO checks(id,order_id,status,currency_code,subtotal,discount_total,surcharge_total,tax_total,total,paid_total,remaining_total,business_date_local,closed_at,snapshot,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.OrderID, string(v.Status), v.CurrencyCode, v.Subtotal, v.DiscountTotal, v.SurchargeTotal, v.TaxTotal, v.Total, v.PaidTotal, v.RemainingTotal, v.BusinessDateLocal, dbTime(v.ClosedAt), string(snapshot), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetCheck(ctx context.Context, id string) (*domain.Check, error) {
	return r.scanCheck(r.queryer(ctx).QueryRowContext(ctx, checkSelectSQL+` WHERE id = ?`, id))
}

func (r *Repository) GetCheckByOrder(ctx context.Context, orderID string) (*domain.Check, error) {
	return r.scanCheck(r.queryer(ctx).QueryRowContext(ctx, checkSelectSQL+` WHERE order_id = ?`, orderID))
}

func (r *Repository) scanCheck(row *sql.Row) (*domain.Check, error) {
	var v domain.Check
	var status, closed, snapshot, created, updated string
	err := row.Scan(&v.ID, &v.OrderID, &status, &v.CurrencyCode, &v.Subtotal, &v.DiscountTotal, &v.SurchargeTotal, &v.TaxTotal, &v.Total, &v.PaidTotal, &v.RemainingTotal, &v.BusinessDateLocal, &closed, &snapshot, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Status = domain.CheckStatus(status)
	v.ClosedAt = parseTime(closed)
	v.Snapshot = json.RawMessage(snapshot)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

const checkSelectSQL = `SELECT id,order_id,status,currency_code,subtotal,discount_total,surcharge_total,tax_total,total,paid_total,remaining_total,business_date_local,closed_at,snapshot,created_at,updated_at FROM checks`

func (r *Repository) UpdateCheckPaidTotal(ctx context.Context, v *domain.Check) error {
	_, err := r.execer(ctx).ExecContext(ctx, `UPDATE checks SET status = ?, paid_total = ?, remaining_total = ?, updated_at = ? WHERE id = ?`,
		string(v.Status), v.PaidTotal, v.RemainingTotal, dbTime(v.UpdatedAt), v.ID)
	return normalizeErr(err)
}

func (r *Repository) GetPayment(ctx context.Context, id string) (*domain.Payment, error) {
	return scanPaymentRows(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,edge_payment_id,restaurant_id,device_id,shift_id,precheck_id,method,amount,currency,status,business_date_local,provider_name,provider_transaction_id,provider_reference,fingerprint_hash,created_at,updated_at FROM payments WHERE id = ?`, id))
}

func (r *Repository) CreatePayment(ctx context.Context, v *domain.Payment) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO payments(id,edge_payment_id,restaurant_id,device_id,shift_id,precheck_id,method,amount,currency,status,business_date_local,provider_name,provider_transaction_id,provider_reference,fingerprint_hash,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.EdgePaymentID, v.RestaurantID, v.DeviceID, v.ShiftID, v.PrecheckID, string(v.Method), v.Amount, v.Currency, string(v.Status), v.BusinessDateLocal, nullableString(v.ProviderName), nullableString(v.ProviderTransactionID), nullableString(v.ProviderReference), nullableString(v.FingerprintHash), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) CreatePaymentAttempt(ctx context.Context, v *domain.PaymentAttempt) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO payment_attempts(id,payment_id,attempt_no,method,amount,currency,status,provider_name,provider_transaction_id,provider_reference,fingerprint_hash,attempted_at,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.PaymentID, v.AttemptNo, string(v.Method), v.Amount, v.Currency, string(v.Status), nullableString(v.ProviderName), nullableString(v.ProviderTransactionID), nullableString(v.ProviderReference), nullableString(v.FingerprintHash), dbTime(v.AttemptedAt), dbTime(v.CreatedAt))
	return normalizeErr(err)
}

func (r *Repository) UpdatePaymentStatus(ctx context.Context, v *domain.Payment) error {
	_, err := r.execer(ctx).ExecContext(ctx, `UPDATE payments SET status = ?, updated_at = ? WHERE id = ?`,
		string(v.Status), dbTime(v.UpdatedAt), v.ID)
	return normalizeErr(err)
}

func (r *Repository) NextPaymentAttemptNo(ctx context.Context, paymentID string) (int, error) {
	var no int
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT COALESCE(MAX(attempt_no), 0) + 1 FROM payment_attempts WHERE payment_id = ?`, paymentID).Scan(&no)
	return no, normalizeErr(err)
}

func (r *Repository) ListPaymentsByPrecheck(ctx context.Context, precheckID string) ([]domain.Payment, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,edge_payment_id,restaurant_id,device_id,shift_id,precheck_id,method,amount,currency,status,business_date_local,provider_name,provider_transaction_id,provider_reference,fingerprint_hash,created_at,updated_at FROM payments WHERE precheck_id = ? ORDER BY created_at, id`, precheckID)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	var out []domain.Payment
	for rows.Next() {
		v, err := scanPaymentRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, normalizeErr(rows.Err())
}

type paymentScanner interface {
	Scan(...any) error
}

func scanPaymentRows(row paymentScanner) (*domain.Payment, error) {
	var v domain.Payment
	var method, status, created, updated string
	var providerName, providerTransactionID, providerReference, fingerprintHash sql.NullString
	if err := row.Scan(&v.ID, &v.EdgePaymentID, &v.RestaurantID, &v.DeviceID, &v.ShiftID, &v.PrecheckID, &method, &v.Amount, &v.Currency, &status, &v.BusinessDateLocal, &providerName, &providerTransactionID, &providerReference, &fingerprintHash, &created, &updated); err != nil {
		return nil, normalizeErr(err)
	}
	v.Method = domain.PaymentMethod(method)
	v.Status = domain.PaymentStatus(status)
	v.ProviderName = stringPtr(providerName)
	v.ProviderTransactionID = stringPtr(providerTransactionID)
	v.ProviderReference = stringPtr(providerReference)
	v.FingerprintHash = stringPtr(fingerprintHash)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

package sqlite

import (
	"context"
	"database/sql"
	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateCheck(ctx context.Context, v *domain.Check) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO checks(id,order_id,status,subtotal,discount_total,tax_total,total,paid_total,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.OrderID, string(v.Status), v.Subtotal, v.DiscountTotal, v.TaxTotal, v.Total, v.PaidTotal, dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetCheck(ctx context.Context, id string) (*domain.Check, error) {
	return r.scanCheck(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,order_id,status,subtotal,discount_total,tax_total,total,paid_total,created_at,updated_at FROM checks WHERE id = ?`, id))
}

func (r *Repository) GetCheckByOrder(ctx context.Context, orderID string) (*domain.Check, error) {
	return r.scanCheck(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,order_id,status,subtotal,discount_total,tax_total,total,paid_total,created_at,updated_at FROM checks WHERE order_id = ?`, orderID))
}

func (r *Repository) scanCheck(row *sql.Row) (*domain.Check, error) {
	var v domain.Check
	var status, created, updated string
	err := row.Scan(&v.ID, &v.OrderID, &status, &v.Subtotal, &v.DiscountTotal, &v.TaxTotal, &v.Total, &v.PaidTotal, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Status = domain.CheckStatus(status)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) UpdateCheckPaidTotal(ctx context.Context, v *domain.Check) error {
	_, err := r.execer(ctx).ExecContext(ctx, `UPDATE checks SET status = ?, paid_total = ?, updated_at = ? WHERE id = ?`,
		string(v.Status), v.PaidTotal, dbTime(v.UpdatedAt), v.ID)
	return normalizeErr(err)
}

func (r *Repository) CreatePayment(ctx context.Context, v *domain.Payment) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO payments(id,edge_payment_id,restaurant_id,device_id,shift_id,precheck_id,method,amount,currency,status,provider_name,provider_transaction_id,provider_reference,fingerprint_hash,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.EdgePaymentID, v.RestaurantID, v.DeviceID, v.ShiftID, v.PrecheckID, string(v.Method), v.Amount, v.Currency, string(v.Status), nullableString(v.ProviderName), nullableString(v.ProviderTransactionID), nullableString(v.ProviderReference), nullableString(v.FingerprintHash), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) CreatePaymentAttempt(ctx context.Context, v *domain.PaymentAttempt) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO payment_attempts(id,payment_id,attempt_no,method,amount,currency,status,provider_name,provider_transaction_id,provider_reference,fingerprint_hash,attempted_at,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.PaymentID, v.AttemptNo, string(v.Method), v.Amount, v.Currency, string(v.Status), nullableString(v.ProviderName), nullableString(v.ProviderTransactionID), nullableString(v.ProviderReference), nullableString(v.FingerprintHash), dbTime(v.AttemptedAt), dbTime(v.CreatedAt))
	return normalizeErr(err)
}

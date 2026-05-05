package sqlite

import (
	"context"

	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateLocalEvent(ctx context.Context, v *domain.LocalEvent) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO local_event_log(id,event_id,envelope_version,event_type,aggregate_type,aggregate_id,restaurant_id,device_id,shift_id,payload_json,occurred_at,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.EventID, v.EnvelopeVersion, v.EventType, v.AggregateType, v.AggregateID, nullableString(v.RestaurantID), v.DeviceID, nullableString(v.ShiftID), v.PayloadJSON, dbTime(v.OccurredAt), dbTime(v.CreatedAt))
	return normalizeErr(err)
}

func (r *Repository) CountLocalEvents(ctx context.Context) (int, error) {
	var n int
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT COUNT(1) FROM local_event_log`).Scan(&n)
	return n, normalizeErr(err)
}

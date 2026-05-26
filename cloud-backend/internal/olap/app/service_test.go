package app_test

import (
	"context"
	"testing"
	"time"

	"cloud-backend/internal/olap/app"
)

func TestListRawBusinessEventsAppliesBoundedLimit(t *testing.T) {
	repo := &rawRepo{}
	service := app.NewService(repo)

	if _, err := service.ListRawBusinessEvents(context.Background(), app.RawBusinessEventFilter{Limit: 1000}); err != nil {
		t.Fatal(err)
	}
	if repo.filter.Limit != 50 {
		t.Fatalf("expected oversized limit to fall back to 50, got %d", repo.filter.Limit)
	}
}

func TestListRawBusinessEventsRejectsInvalidRange(t *testing.T) {
	repo := &rawRepo{}
	service := app.NewService(repo)
	from := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)

	if _, err := service.ListRawBusinessEvents(context.Background(), app.RawBusinessEventFilter{OccurredFrom: &from, OccurredTo: &to}); err == nil {
		t.Fatal("expected invalid range error")
	}
}

type rawRepo struct {
	filter app.RawBusinessEventFilter
}

func (r *rawRepo) ListRawBusinessEvents(_ context.Context, filter app.RawBusinessEventFilter) ([]app.RawBusinessEvent, error) {
	r.filter = filter
	return nil, nil
}

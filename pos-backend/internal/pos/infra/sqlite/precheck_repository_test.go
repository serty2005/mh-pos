package sqlite_test

import (
	"testing"
	"time"

	"pos-backend/internal/pos/domain"
	possqlite "pos-backend/internal/pos/infra/sqlite"
)

func TestPrecheckRepositoryPersistsAndFindsActiveByOrder(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedFinancialForSchemaTests(t, ctx, db)
	repo := possqlite.NewRepository(db)
	now := time.Date(2026, 5, 4, 20, 0, 0, 0, time.UTC)
	precheck := &domain.Precheck{
		ID:            "precheck-1",
		OrderID:       "order-1",
		Status:        domain.PrecheckIssued,
		Subtotal:      100,
		DiscountTotal: 0,
		TaxTotal:      0,
		Total:         100,
		CreatedAt:     now,
		IssuedAt:      now,
	}

	if err := repo.CreatePrecheck(ctx, precheck); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetPrecheck(ctx, precheck.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != precheck.ID || got.OrderID != precheck.OrderID || got.Status != domain.PrecheckIssued || got.Total != 100 {
		t.Fatalf("unexpected precheck: %+v", got)
	}
	active, err := repo.GetActivePrecheckByOrder(ctx, precheck.OrderID)
	if err != nil {
		t.Fatal(err)
	}
	if active.ID != precheck.ID {
		t.Fatalf("expected active precheck %s, got %s", precheck.ID, active.ID)
	}
}

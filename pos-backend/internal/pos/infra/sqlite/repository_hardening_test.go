package sqlite_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	platformsqlite "pos-backend/internal/platform/sqlite"
	"pos-backend/internal/pos/domain"
	possqlite "pos-backend/internal/pos/infra/sqlite"
)

func TestRepositoryNullableScansRoundTrip(t *testing.T) {
	db, _ := newSchemaDB(t)
	ctx := t.Context()
	seedFinancialForSchemaTests(t, ctx, db)
	repo := possqlite.NewRepository(db)
	now := time.Date(2026, 5, 4, 20, 0, 0, 0, time.UTC)
	later := now.Add(30 * time.Minute)

	firstSnapshot := json.RawMessage(`{"lines":[{"id":"line-1","qty":1}],"marker":"round-trip"}`)
	precheck := &domain.Precheck{
		ID:                 "precheck-scan-open",
		OrderID:            "order-1",
		Status:             domain.PrecheckIssued,
		Version:            1,
		CurrencyCode:       "RUB",
		Subtotal:           100,
		Total:              100,
		RemainingTotal:     100,
		Snapshot:           firstSnapshot,
		CreatedAt:          now,
		IssuedAt:           now,
		ClosedAt:           nil,
		CancellationReason: nil,
	}
	if err := repo.CreatePrecheck(ctx, precheck); err != nil {
		t.Fatal(err)
	}
	gotPrecheck, err := repo.GetPrecheck(ctx, precheck.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotPrecheck.ClosedAt != nil || gotPrecheck.CancelledByEmployeeID != nil || gotPrecheck.CancellationReason != nil || gotPrecheck.SupersedesPrecheckID != nil {
		t.Fatalf("expected nil nullable precheck fields, got %+v", gotPrecheck)
	}
	if string(gotPrecheck.Snapshot) != string(firstSnapshot) {
		t.Fatalf("precheck snapshot changed: got %s", gotPrecheck.Snapshot)
	}

	supersedes := "precheck-scan-open"
	cancelledBy := "employee-1"
	reason := "operator confirmed"
	terminal := &domain.Precheck{
		ID:                    "precheck-scan-terminal",
		OrderID:               "order-1",
		Status:                domain.PrecheckSuperseded,
		Version:               2,
		SupersedesPrecheckID:  &supersedes,
		CurrencyCode:          "RUB",
		Subtotal:              100,
		Total:                 100,
		RemainingTotal:        100,
		Snapshot:              json.RawMessage(`{"terminal":true}`),
		CreatedAt:             later,
		IssuedAt:              later,
		ClosedAt:              &later,
		CancelledByEmployeeID: &cancelledBy,
		CancellationReason:    &reason,
	}
	if err := repo.CreatePrecheck(ctx, terminal); err != nil {
		t.Fatal(err)
	}
	gotTerminal, err := repo.GetPrecheck(ctx, terminal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotTerminal.SupersedesPrecheckID == nil || *gotTerminal.SupersedesPrecheckID != supersedes {
		t.Fatalf("expected supersedes pointer to round-trip, got %+v", gotTerminal.SupersedesPrecheckID)
	}
	if gotTerminal.ClosedAt == nil || !gotTerminal.ClosedAt.Equal(later) {
		t.Fatalf("expected closed_at pointer %s, got %+v", later, gotTerminal.ClosedAt)
	}
	if gotTerminal.CancelledByEmployeeID == nil || *gotTerminal.CancelledByEmployeeID != cancelledBy || gotTerminal.CancellationReason == nil || *gotTerminal.CancellationReason != reason {
		t.Fatalf("expected nullable audit fields to round-trip, got %+v", gotTerminal)
	}

	payment := domain.Payment{
		ID:                "payment-scan-null",
		EdgePaymentID:     "edge-payment-scan-null",
		RestaurantID:      "restaurant-1",
		DeviceID:          "device-1",
		ShiftID:           "shift-1",
		PrecheckID:        precheck.ID,
		Method:            domain.PaymentCash,
		Amount:            100,
		Currency:          "RUB",
		Status:            domain.PaymentCaptured,
		BusinessDateLocal: "2026-05-04",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := repo.CreatePayment(ctx, &payment); err != nil {
		t.Fatal(err)
	}
	gotPayment, err := repo.GetPayment(ctx, payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotPayment.ProviderName != nil || gotPayment.ProviderTransactionID != nil || gotPayment.ProviderReference != nil || gotPayment.FingerprintHash != nil {
		t.Fatalf("expected nil payment provider fields, got %+v", gotPayment)
	}
	if gotPayment.BusinessDateLocal != "2026-05-04" {
		t.Fatalf("unexpected business_date_local: %s", gotPayment.BusinessDateLocal)
	}

	providerName := "card"
	providerTransactionID := "txn-1"
	providerReference := "ref-1"
	fingerprintHash := "fingerprint-1"
	paymentWithProvider := payment
	paymentWithProvider.ID = "payment-scan-provider"
	paymentWithProvider.EdgePaymentID = "edge-payment-scan-provider"
	paymentWithProvider.Method = domain.PaymentCard
	paymentWithProvider.ProviderName = &providerName
	paymentWithProvider.ProviderTransactionID = &providerTransactionID
	paymentWithProvider.ProviderReference = &providerReference
	paymentWithProvider.FingerprintHash = &fingerprintHash
	if err := repo.CreatePayment(ctx, &paymentWithProvider); err != nil {
		t.Fatal(err)
	}
	gotPaymentWithProvider, err := repo.GetPayment(ctx, paymentWithProvider.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotPaymentWithProvider.ProviderName == nil || *gotPaymentWithProvider.ProviderName != providerName ||
		gotPaymentWithProvider.ProviderTransactionID == nil || *gotPaymentWithProvider.ProviderTransactionID != providerTransactionID ||
		gotPaymentWithProvider.ProviderReference == nil || *gotPaymentWithProvider.ProviderReference != providerReference ||
		gotPaymentWithProvider.FingerprintHash == nil || *gotPaymentWithProvider.FingerprintHash != fingerprintHash {
		t.Fatalf("expected payment provider pointers to round-trip, got %+v", gotPaymentWithProvider)
	}

	cashSession := domain.CashSession{
		ID:                 "cash-session-scan",
		EdgeCashSessionID:  "edge-cash-session-scan",
		RestaurantID:       "restaurant-1",
		DeviceID:           "device-1",
		SalesPointID:       "sales-point-1",
		ShiftID:            "shift-1",
		OpenedByEmployeeID: "employee-1",
		Status:             domain.CashSessionOpen,
		BusinessDateLocal:  "2026-05-04",
		OpeningCashAmount:  1000,
		OpenedAt:           now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := repo.CreateCashSession(ctx, &cashSession); err != nil {
		t.Fatal(err)
	}
	gotCashSession, err := repo.GetCashSession(ctx, cashSession.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotCashSession.ClosedByEmployeeID != nil || gotCashSession.ClosingCashAmount != nil || gotCashSession.ClosedAt != nil {
		t.Fatalf("expected nil cash close fields, got %+v", gotCashSession)
	}
	closingAmount := int64(1250)
	cashSession.Status = domain.CashSessionClosed
	cashSession.ClosedByEmployeeID = &cancelledBy
	cashSession.ClosingCashAmount = &closingAmount
	cashSession.ClosedAt = &later
	cashSession.UpdatedAt = later
	if err := repo.UpdateCashSessionClosed(ctx, &cashSession); err != nil {
		t.Fatal(err)
	}
	gotCashSession, err = repo.GetCashSession(ctx, cashSession.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotCashSession.ClosedByEmployeeID == nil || *gotCashSession.ClosedByEmployeeID != cancelledBy ||
		gotCashSession.ClosingCashAmount == nil || *gotCashSession.ClosingCashAmount != closingAmount ||
		gotCashSession.ClosedAt == nil || !gotCashSession.ClosedAt.Equal(later) {
		t.Fatalf("expected cash close fields to round-trip, got %+v", gotCashSession)
	}

	authSession := domain.AuthSession{
		ID:             "auth-session-scan",
		RestaurantID:   "restaurant-1",
		NodeDeviceID:   "device-1",
		ClientDeviceID: "client-1",
		EmployeeID:     "employee-1",
		Status:         domain.AuthSessionActive,
		StartedAt:      now,
		LastSeenAt:     now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repo.CreateAuthSession(ctx, &authSession); err != nil {
		t.Fatal(err)
	}
	gotAuthSession, err := repo.GetAuthSession(ctx, authSession.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotAuthSession.ExpiresAt != nil || gotAuthSession.RevokedAt != nil {
		t.Fatalf("expected nil auth session times, got %+v", gotAuthSession)
	}
	authSession.ID = "auth-session-scan-times"
	authSession.ExpiresAt = &later
	authSession.RevokedAt = &later
	authSession.Status = domain.AuthSessionRevoked
	if err := repo.CreateAuthSession(ctx, &authSession); err != nil {
		t.Fatal(err)
	}
	gotAuthSession, err = repo.GetAuthSession(ctx, authSession.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotAuthSession.ExpiresAt == nil || !gotAuthSession.ExpiresAt.Equal(later) || gotAuthSession.RevokedAt == nil || !gotAuthSession.RevokedAt.Equal(later) {
		t.Fatalf("expected auth session nullable times to round-trip, got %+v", gotAuthSession)
	}

	restaurantID := "restaurant-1"
	clientDeviceID := "client-1"
	actorID := "employee-1"
	sessionID := "auth-session-scan"
	nextRetryAt := later
	lockedAt := later
	lockedBy := "worker-1"
	lastError := "retry later"
	outbox := domain.OutboxMessage{
		ID:            "outbox-scan-null",
		CommandID:     "cmd-outbox-scan-null",
		SequenceNo:    10,
		Origin:        domain.OriginEdgeDevice,
		DeviceID:      "device-1",
		NodeDeviceID:  "device-1",
		AggregateType: "Order",
		AggregateID:   "order-1",
		CommandType:   "OrderCreated",
		SyncDirection: domain.SyncDirectionEdgeToCloud,
		PayloadJSON:   `{"safe":true}`,
		Status:        domain.OutboxPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repo.CreateOutboxMessage(ctx, &outbox); err != nil {
		t.Fatal(err)
	}
	gotOutbox, err := repo.GetOutboxByID(ctx, outbox.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotOutbox.RestaurantID != nil || gotOutbox.ClientDeviceID != nil || gotOutbox.ActorEmployeeID != nil || gotOutbox.SessionID != nil ||
		gotOutbox.NextRetryAt != nil || gotOutbox.LockedAt != nil || gotOutbox.LockedBy != nil || gotOutbox.SentAt != nil || gotOutbox.LastError != nil {
		t.Fatalf("expected nil outbox optional fields, got %+v", gotOutbox)
	}
	outboxWithOptional := outbox
	outboxWithOptional.ID = "outbox-scan-processing"
	outboxWithOptional.CommandID = "cmd-outbox-scan-processing"
	outboxWithOptional.SequenceNo = 11
	outboxWithOptional.RestaurantID = &restaurantID
	outboxWithOptional.ClientDeviceID = &clientDeviceID
	outboxWithOptional.ActorEmployeeID = &actorID
	outboxWithOptional.SessionID = &sessionID
	outboxWithOptional.Status = domain.OutboxProcessing
	outboxWithOptional.NextRetryAt = &nextRetryAt
	outboxWithOptional.LockedAt = &lockedAt
	outboxWithOptional.LockedBy = &lockedBy
	outboxWithOptional.LastError = &lastError
	if err := repo.CreateOutboxMessage(ctx, &outboxWithOptional); err != nil {
		t.Fatal(err)
	}
	gotOutbox, err = repo.GetOutboxByID(ctx, outboxWithOptional.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotOutbox.RestaurantID == nil || *gotOutbox.RestaurantID != restaurantID ||
		gotOutbox.ClientDeviceID == nil || *gotOutbox.ClientDeviceID != clientDeviceID ||
		gotOutbox.ActorEmployeeID == nil || *gotOutbox.ActorEmployeeID != actorID ||
		gotOutbox.SessionID == nil || *gotOutbox.SessionID != sessionID ||
		gotOutbox.NextRetryAt == nil || !gotOutbox.NextRetryAt.Equal(nextRetryAt) ||
		gotOutbox.LockedAt == nil || !gotOutbox.LockedAt.Equal(lockedAt) ||
		gotOutbox.LockedBy == nil || *gotOutbox.LockedBy != lockedBy ||
		gotOutbox.LastError == nil || *gotOutbox.LastError != lastError {
		t.Fatalf("expected outbox optional fields to round-trip, got %+v", gotOutbox)
	}

	localEvent := domain.LocalEvent{
		ID:              "local-event-scan-null",
		EventID:         "edge-event-scan-null",
		CommandID:       "cmd-local-event-scan-null",
		EnvelopeVersion: domain.SyncEnvelopeVersion,
		EventType:       "OrderCreated",
		AggregateType:   "Order",
		AggregateID:     "order-1",
		DeviceID:        "device-1",
		NodeDeviceID:    "device-1",
		PayloadJSON:     `{"raw":"unchanged"}`,
		OccurredAt:      now,
		CreatedAt:       now,
	}
	if err := repo.CreateLocalEvent(ctx, &localEvent); err != nil {
		t.Fatal(err)
	}
	localEventWithOptional := localEvent
	localEventWithOptional.ID = "local-event-scan-optional"
	localEventWithOptional.EventID = "edge-event-scan-optional"
	localEventWithOptional.CommandID = "cmd-local-event-scan-optional"
	localEventWithOptional.RestaurantID = &restaurantID
	localEventWithOptional.ClientDeviceID = &clientDeviceID
	localEventWithOptional.ShiftID = stringPtrForTest("shift-1")
	localEventWithOptional.ActorEmployeeID = &actorID
	localEventWithOptional.SessionID = &sessionID
	localEventWithOptional.CreatedAt = later
	if err := repo.CreateLocalEvent(ctx, &localEventWithOptional); err != nil {
		t.Fatal(err)
	}
	events, err := repo.ListLocalEvents(ctx, 10, "")
	if err != nil {
		t.Fatal(err)
	}
	localEventsByID := map[string]domain.LocalEvent{}
	for _, event := range events {
		localEventsByID[event.ID] = event
	}
	if got := localEventsByID[localEvent.ID]; got.RestaurantID != nil || got.ClientDeviceID != nil || got.ShiftID != nil || got.ActorEmployeeID != nil || got.SessionID != nil || got.PayloadJSON != localEvent.PayloadJSON {
		t.Fatalf("expected nil local event optional fields and unchanged payload, got %+v", got)
	}
	if got := localEventsByID[localEventWithOptional.ID]; got.RestaurantID == nil || *got.RestaurantID != restaurantID ||
		got.ClientDeviceID == nil || *got.ClientDeviceID != clientDeviceID ||
		got.ShiftID == nil || *got.ShiftID != "shift-1" ||
		got.ActorEmployeeID == nil || *got.ActorEmployeeID != actorID ||
		got.SessionID == nil || *got.SessionID != sessionID {
		t.Fatalf("expected local event optional fields to round-trip, got %+v", got)
	}

	operation := domain.FinancialOperation{
		ID:                   "financial-operation-scan",
		EdgeOperationID:      "edge-financial-operation-scan",
		RestaurantID:         "restaurant-1",
		DeviceID:             "device-1",
		ShiftID:              "shift-1",
		OriginalShiftID:      "shift-1",
		CheckID:              "check-1",
		PrecheckID:           precheck.ID,
		Type:                 domain.FinancialOperationRefund,
		Kind:                 domain.FinancialOperationPartial,
		Status:               domain.FinancialOperationRecorded,
		Amount:               50,
		Currency:             "RUB",
		BusinessDateLocal:    "2026-05-04",
		InventoryDisposition: domain.InventoryNoStockEffect,
		Reason:               "guest refund",
		CreatedByEmployeeID:  "employee-1",
		Snapshot:             json.RawMessage(`{"payment":"payment-scan-null"}`),
		CreatedAt:            now,
	}
	if err := repo.CreateFinancialOperation(ctx, &operation); err != nil {
		t.Fatal(err)
	}
	quantity := int64(2)
	items := []domain.FinancialOperationItem{
		{ID: "financial-operation-item-null", OperationID: operation.ID, Scope: domain.FinancialItemWholeCheck, Amount: 25, Currency: "RUB", Snapshot: json.RawMessage(`{"item":null}`), CreatedAt: now},
		{ID: "financial-operation-item-quantity", OperationID: operation.ID, Scope: domain.FinancialItemWholeCheck, Quantity: &quantity, Amount: 25, Currency: "RUB", TaxAmount: 5, Snapshot: json.RawMessage(`{"item":"quantity"}`), CreatedAt: later},
	}
	for i := range items {
		if err := repo.CreateFinancialOperationItem(ctx, &items[i]); err != nil {
			t.Fatal(err)
		}
	}
	operations, err := repo.ListFinancialOperations(ctx, domain.FinancialOperationListQuery{CheckID: "check-1", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 1 {
		t.Fatalf("expected one financial operation, got %+v", operations)
	}
	if operations[0].ApprovedByEmployeeID != nil || string(operations[0].Snapshot) != string(operation.Snapshot) {
		t.Fatalf("expected financial operation null approved_by and snapshot round-trip, got %+v", operations[0])
	}
	if len(operations[0].Items) != 2 {
		t.Fatalf("expected two financial operation items, got %+v", operations[0].Items)
	}
	if operations[0].Items[0].Quantity != nil || operations[0].Items[0].OrderLineID != nil || operations[0].Items[0].PaymentID != nil {
		t.Fatalf("expected nil item nullable fields, got %+v", operations[0].Items[0])
	}
	if operations[0].Items[1].Quantity == nil || *operations[0].Items[1].Quantity != quantity || string(operations[0].Items[1].Snapshot) != string(items[1].Snapshot) {
		t.Fatalf("expected item quantity pointer and snapshot round-trip, got %+v", operations[0].Items[1])
	}
}

func TestRepositoryUniqueConstraintsNormalizeAndPreserveRows(t *testing.T) {
	db, _ := newSchemaDB(t)
	ctx := t.Context()
	seedFinancialForSchemaTests(t, ctx, db)
	repo := possqlite.NewRepository(db)
	now := time.Date(2026, 5, 4, 20, 0, 0, 0, time.UTC)
	precheck := &domain.Precheck{
		ID:             "precheck-unique",
		OrderID:        "order-1",
		Status:         domain.PrecheckIssued,
		Version:        1,
		CurrencyCode:   "RUB",
		Subtotal:       100,
		Total:          100,
		RemainingTotal: 100,
		Snapshot:       json.RawMessage(`{"secret":"secret-pin-marker"}`),
		CreatedAt:      now,
		IssuedAt:       now,
	}
	if err := repo.CreatePrecheck(ctx, precheck); err != nil {
		t.Fatal(err)
	}
	assertDuplicateRepositoryError(t, repo.CreatePrecheck(ctx, &domain.Precheck{
		ID:             "precheck-unique-duplicate-version",
		OrderID:        "order-1",
		Status:         domain.PrecheckSuperseded,
		Version:        1,
		CurrencyCode:   "RUB",
		Subtotal:       100,
		Total:          100,
		RemainingTotal: 100,
		Snapshot:       json.RawMessage(`{"secret":"secret-pin-marker"}`),
		CreatedAt:      now,
		IssuedAt:       now,
		ClosedAt:       &now,
	}))

	payment := domain.Payment{
		ID:                "payment-unique",
		EdgePaymentID:     "edge-payment-unique",
		RestaurantID:      "restaurant-1",
		DeviceID:          "device-1",
		ShiftID:           "shift-1",
		PrecheckID:        precheck.ID,
		Method:            domain.PaymentCash,
		Amount:            100,
		Currency:          "RUB",
		Status:            domain.PaymentCaptured,
		BusinessDateLocal: "2026-05-04",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := repo.CreatePayment(ctx, &payment); err != nil {
		t.Fatal(err)
	}
	duplicatePayment := payment
	duplicatePayment.ID = "payment-unique-duplicate-edge"
	assertDuplicateRepositoryError(t, repo.CreatePayment(ctx, &duplicatePayment))
	gotPayment, err := repo.GetPayment(ctx, payment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotPayment.EdgePaymentID != payment.EdgePaymentID || gotPayment.Amount != payment.Amount {
		t.Fatalf("existing payment was modified: %+v", gotPayment)
	}

	outbox := domain.OutboxMessage{
		ID:            "outbox-unique",
		CommandID:     "cmd-outbox-unique",
		SequenceNo:    20,
		Origin:        domain.OriginEdgeDevice,
		DeviceID:      "device-1",
		NodeDeviceID:  "device-1",
		AggregateType: "Payment",
		AggregateID:   payment.ID,
		CommandType:   "PaymentCaptured",
		PayloadJSON:   `{"raw":"raw-sync-payload-marker"}`,
		Status:        domain.OutboxPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repo.CreateOutboxMessage(ctx, &outbox); err != nil {
		t.Fatal(err)
	}
	duplicateOutboxID := outbox
	duplicateOutboxID.CommandID = "cmd-outbox-unique-second"
	duplicateOutboxID.SequenceNo = 21
	assertDuplicateRepositoryError(t, repo.CreateOutboxMessage(ctx, &duplicateOutboxID))
	duplicateSequence := outbox
	duplicateSequence.ID = "outbox-unique-duplicate-sequence"
	duplicateSequence.CommandID = "cmd-outbox-unique-duplicate-sequence"
	assertDuplicateRepositoryError(t, repo.CreateOutboxMessage(ctx, &duplicateSequence))
	gotOutbox, err := repo.GetOutboxByID(ctx, outbox.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotOutbox.PayloadJSON != outbox.PayloadJSON || gotOutbox.SequenceNo != outbox.SequenceNo {
		t.Fatalf("existing outbox row was modified: %+v", gotOutbox)
	}

	operation := domain.FinancialOperation{
		ID:                   "financial-operation-unique",
		EdgeOperationID:      "edge-financial-operation-unique",
		RestaurantID:         "restaurant-1",
		DeviceID:             "device-1",
		ShiftID:              "shift-1",
		OriginalShiftID:      "shift-1",
		CheckID:              "check-1",
		PrecheckID:           precheck.ID,
		Type:                 domain.FinancialOperationRefund,
		Kind:                 domain.FinancialOperationPartial,
		Status:               domain.FinancialOperationRecorded,
		Amount:               25,
		Currency:             "RUB",
		BusinessDateLocal:    "2026-05-04",
		InventoryDisposition: domain.InventoryNoStockEffect,
		Reason:               "payment-sensitive-marker",
		CreatedByEmployeeID:  "employee-1",
		Snapshot:             json.RawMessage(`{"marker":"payment-sensitive-marker"}`),
		CreatedAt:            now,
	}
	if err := repo.CreateFinancialOperation(ctx, &operation); err != nil {
		t.Fatal(err)
	}
	duplicateOperation := operation
	duplicateOperation.ID = "financial-operation-unique-duplicate-edge"
	assertDuplicateRepositoryError(t, repo.CreateFinancialOperation(ctx, &duplicateOperation))
	operations, err := repo.ListFinancialOperations(ctx, domain.FinancialOperationListQuery{CheckID: "check-1", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 1 || operations[0].ID != operation.ID {
		t.Fatalf("existing financial operation was not preserved: %+v", operations)
	}
}

func TestRepositoryBoundedFiltersUseStableOrderingAndScopes(t *testing.T) {
	db, _ := newSchemaDB(t)
	ctx := t.Context()
	repo := possqlite.NewRepository(db)
	seedClosedOrderFixtureBase(t, ctx, db, "restaurant-1", "device-1", "employee-1", "shift-1", "table-1")
	seedClosedOrderFixtureBase(t, ctx, db, "restaurant-1", "device-2", "employee-2", "shift-2", "table-2")
	seedClosedOrderFixtureBase(t, ctx, db, "restaurant-2", "device-3", "employee-3", "shift-3", "table-3")

	insertClosedOrderGraph(t, ctx, db, archiveOrderGraph{
		Suffix:            "old-r1",
		RestaurantID:      "restaurant-1",
		DeviceID:          "device-1",
		ShiftID:           "shift-1",
		TableID:           "table-1",
		EmployeeID:        "employee-1",
		BusinessDateLocal: "2026-05-01",
		ClosedAt:          "2026-05-01T20:00:00Z",
	})
	insertClosedOrderGraph(t, ctx, db, archiveOrderGraph{
		Suffix:            "mid-r1",
		RestaurantID:      "restaurant-1",
		DeviceID:          "device-1",
		ShiftID:           "shift-1",
		TableID:           "table-1",
		EmployeeID:        "employee-1",
		BusinessDateLocal: "2026-05-02",
		ClosedAt:          "2026-05-02T20:00:00Z",
	})
	insertClosedOrderGraph(t, ctx, db, archiveOrderGraph{
		Suffix:            "new-r1-device2",
		RestaurantID:      "restaurant-1",
		DeviceID:          "device-2",
		ShiftID:           "shift-2",
		TableID:           "table-2",
		EmployeeID:        "employee-2",
		BusinessDateLocal: "2026-05-03",
		ClosedAt:          "2026-05-03T20:00:00Z",
	})
	insertClosedOrderGraph(t, ctx, db, archiveOrderGraph{
		Suffix:            "other-r2",
		RestaurantID:      "restaurant-2",
		DeviceID:          "device-3",
		ShiftID:           "shift-3",
		TableID:           "table-3",
		EmployeeID:        "employee-3",
		BusinessDateLocal: "2026-05-04",
		ClosedAt:          "2026-05-04T20:00:00Z",
	})

	page, err := repo.ListClosedOrders(ctx, domain.ClosedOrderListQuery{RestaurantID: "restaurant-1", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	assertOrderSummaryIDs(t, page, "order-new-r1-device2", "order-mid-r1")
	page, err = repo.ListClosedOrders(ctx, domain.ClosedOrderListQuery{RestaurantID: "restaurant-1", Limit: 2, Offset: 1})
	if err != nil {
		t.Fatal(err)
	}
	assertOrderSummaryIDs(t, page, "order-mid-r1", "order-old-r1")
	filtered, err := repo.ListClosedOrders(ctx, domain.ClosedOrderListQuery{RestaurantID: "restaurant-1", DeviceID: "device-1", FromBusinessDateLocal: "2026-05-02", ToBusinessDateLocal: "2026-05-03", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	assertOrderSummaryIDs(t, filtered, "order-mid-r1")
	byCheck, err := repo.ListClosedOrders(ctx, domain.ClosedOrderListQuery{CheckID: "check-old-r1", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	assertOrderSummaryIDs(t, byCheck, "order-old-r1")
	empty, err := repo.ListClosedOrders(ctx, domain.ClosedOrderListQuery{RestaurantID: "missing", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("expected empty non-nil closed order result, got %+v", empty)
	}

	for _, op := range []domain.FinancialOperation{
		financialOperationForTest("fo-filter-old", "edge-fo-filter-old", "restaurant-1", "device-1", "shift-1", "check-old-r1", "precheck-old-r1", domain.FinancialOperationCancellation, "2026-05-01", "2026-05-01T21:00:00Z"),
		financialOperationForTest("fo-filter-mid", "edge-fo-filter-mid", "restaurant-1", "device-1", "shift-1", "check-mid-r1", "precheck-mid-r1", domain.FinancialOperationCancellation, "2026-05-02", "2026-05-02T21:00:00Z"),
		financialOperationForTest("fo-filter-new", "edge-fo-filter-new", "restaurant-1", "device-2", "shift-2", "check-new-r1-device2", "precheck-new-r1-device2", domain.FinancialOperationCancellation, "2026-05-03", "2026-05-03T21:00:00Z"),
		financialOperationForTest("fo-filter-other", "edge-fo-filter-other", "restaurant-2", "device-3", "shift-3", "check-other-r2", "precheck-other-r2", domain.FinancialOperationCancellation, "2026-05-04", "2026-05-04T21:00:00Z"),
	} {
		operation := op
		if err := repo.CreateFinancialOperation(ctx, &operation); err != nil {
			t.Fatal(err)
		}
	}
	ops, err := repo.ListFinancialOperations(ctx, domain.FinancialOperationListQuery{RestaurantID: "restaurant-1", OperationType: domain.FinancialOperationCancellation, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	assertFinancialOperationIDs(t, ops, "fo-filter-new")
	ops, err = repo.ListFinancialOperations(ctx, domain.FinancialOperationListQuery{RestaurantID: "restaurant-1", OperationType: domain.FinancialOperationCancellation, Limit: 1, Offset: 1})
	if err != nil {
		t.Fatal(err)
	}
	assertFinancialOperationIDs(t, ops, "fo-filter-mid")
	ops, err = repo.ListFinancialOperations(ctx, domain.FinancialOperationListQuery{RestaurantID: "restaurant-1", ShiftID: "shift-1", BusinessDateFrom: "2026-05-02", BusinessDateTo: "2026-05-03", OperationType: domain.FinancialOperationCancellation, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	assertFinancialOperationIDs(t, ops, "fo-filter-mid")
}

func TestRepositoryWritesRespectTransactionRollbackAndCommit(t *testing.T) {
	db, _ := newSchemaDB(t)
	ctx := t.Context()
	seedFinancialForSchemaTests(t, ctx, db)
	repo := possqlite.NewRepository(db)
	tx := platformsqlite.NewTxManager(db)
	now := time.Date(2026, 5, 4, 20, 0, 0, 0, time.UTC)
	precheck := &domain.Precheck{
		ID:             "precheck-tx",
		OrderID:        "order-1",
		Status:         domain.PrecheckIssued,
		Version:        1,
		CurrencyCode:   "RUB",
		Subtotal:       100,
		Total:          100,
		RemainingTotal: 100,
		Snapshot:       json.RawMessage(`{}`),
		CreatedAt:      now,
		IssuedAt:       now,
	}
	if err := repo.CreatePrecheck(ctx, precheck); err != nil {
		t.Fatal(err)
	}

	rollbackErr := errors.New("force rollback")
	err := tx.WithinTx(ctx, func(txCtx context.Context) error {
		payment := paymentForTxTest("payment-tx-rollback", "edge-payment-tx-rollback", precheck.ID, now)
		if err := repo.CreatePayment(txCtx, &payment); err != nil {
			return err
		}
		attempt := paymentAttemptForTxTest("payment-attempt-tx-rollback", payment.ID, 1, now)
		if err := repo.CreatePaymentAttempt(txCtx, &attempt); err != nil {
			return err
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("expected rollback error, got %v", err)
	}
	assertTableCount(t, ctx, db, "payments", "id = 'payment-tx-rollback'", 0)
	assertTableCount(t, ctx, db, "payment_attempts", "id = 'payment-attempt-tx-rollback'", 0)

	err = tx.WithinTx(ctx, func(txCtx context.Context) error {
		payment := paymentForTxTest("payment-tx-commit", "edge-payment-tx-commit", precheck.ID, now)
		if err := repo.CreatePayment(txCtx, &payment); err != nil {
			return err
		}
		attempt := paymentAttemptForTxTest("payment-attempt-tx-commit", payment.ID, 1, now)
		return repo.CreatePaymentAttempt(txCtx, &attempt)
	})
	if err != nil {
		t.Fatal(err)
	}
	assertTableCount(t, ctx, db, "payments", "id = 'payment-tx-commit'", 1)
	assertTableCount(t, ctx, db, "payment_attempts", "id = 'payment-attempt-tx-commit'", 1)
}

func TestStorageArchiveRepositoryScopeAndDestructiveApplySafety(t *testing.T) {
	db, _ := newSchemaDB(t)
	ctx := t.Context()
	repo := possqlite.NewRepository(db)
	seedClosedOrderFixtureBase(t, ctx, db, "restaurant-archive", "device-archive", "employee-archive", "shift-archive-closed", "table-archive")
	insertClosedOrderGraph(t, ctx, db, archiveOrderGraph{
		Suffix:            "archive-eligible",
		RestaurantID:      "restaurant-archive",
		DeviceID:          "device-archive",
		ShiftID:           "shift-archive-closed",
		TableID:           "table-archive",
		EmployeeID:        "employee-archive",
		BusinessDateLocal: "2026-05-01",
		ClosedAt:          "2026-05-01T20:00:00Z",
		OutboxStatus:      domain.OutboxPending,
		PayloadMarker:     "raw-sync-payload-marker",
	})
	insertClosedOrderGraph(t, ctx, db, archiveOrderGraph{
		Suffix:            "archive-cutoff",
		RestaurantID:      "restaurant-archive",
		DeviceID:          "device-archive",
		ShiftID:           "shift-archive-closed",
		TableID:           "table-archive",
		EmployeeID:        "employee-archive",
		BusinessDateLocal: "2026-05-02",
		ClosedAt:          "2026-05-02T20:00:00Z",
		OutboxStatus:      domain.OutboxSent,
		PayloadMarker:     "raw-sync-payload-marker",
	})
	insertClosedOrderGraph(t, ctx, db, archiveOrderGraph{
		Suffix:            "archive-after",
		RestaurantID:      "restaurant-archive",
		DeviceID:          "device-archive",
		ShiftID:           "shift-archive-closed",
		TableID:           "table-archive",
		EmployeeID:        "employee-archive",
		BusinessDateLocal: "2026-05-03",
		ClosedAt:          "2026-05-03T20:00:00Z",
		OutboxStatus:      domain.OutboxSent,
		PayloadMarker:     "raw-sync-payload-marker",
	})

	scope, err := repo.BuildStorageArchiveExportScope(ctx, "2026-05-02")
	if err != nil {
		t.Fatal(err)
	}
	if !scope.Blocked || !containsString(scope.BlockReasons, "pending_edge_to_cloud_outbox") {
		t.Fatalf("expected pending outbox block, got blocked=%v reasons=%+v", scope.Blocked, scope.BlockReasons)
	}
	if scope.BusinessDateRange.Oldest != "2026-05-01" || scope.BusinessDateRange.Newest != "2026-05-01" {
		t.Fatalf("expected exclusive cutoff range for 2026-05-01, got %+v", scope.BusinessDateRange)
	}
	if scope.Counts.ClosedOrders != 1 || scope.Counts.Checks != 1 || scope.Counts.Prechecks != 1 || scope.Counts.Payments != 1 || scope.Counts.LocalEventReferences != 1 || scope.Counts.OutboxMessageReferences != 1 {
		t.Fatalf("unexpected archive scope counts: %+v", scope.Counts)
	}
	for _, row := range scope.Rows {
		if row.Table == "local_event_log_summary" || row.Table == "pos_sync_outbox_summary" {
			if _, ok := row.Row["payload_json"]; ok {
				t.Fatalf("archive summary %s exposed payload_json", row.Table)
			}
			if row.Row["payload_policy"] != "summary_without_payload" {
				t.Fatalf("archive summary %s has unexpected payload policy: %+v", row.Table, row.Row)
			}
			if strings.Contains(fmt.Sprint(row.Row), "raw-sync-payload-marker") {
				t.Fatalf("archive summary %s leaked payload marker", row.Table)
			}
		}
	}
	applyScope, err := repo.BuildStorageArchiveApplyRuntimeScope(ctx, "2026-05-02")
	if err != nil {
		t.Fatal(err)
	}
	if applyScope.BlockingOutboxMessages != 1 || applyScope.Counts.ClosedOrders != 1 {
		t.Fatalf("unexpected apply runtime scope: %+v", applyScope)
	}
	if _, err := repo.ApplyStorageArchiveDestructive(ctx, "2026-05-02"); err == nil {
		t.Fatal("expected destructive apply to be blocked by pending outbox")
	}
	assertTableCount(t, ctx, db, "orders", "id = 'order-archive-eligible'", 1)

	if err := repo.MarkOutboxSent(ctx, "outbox-archive-eligible", "2026-05-02T01:00:00Z"); err != nil {
		t.Fatal(err)
	}
	deleted, err := repo.ApplyStorageArchiveDestructive(ctx, "2026-05-02")
	if err != nil {
		t.Fatal(err)
	}
	if deleted.ClosedOrders != 1 || deleted.Checks != 1 || deleted.Prechecks != 1 || deleted.Payments != 1 || deleted.FinancialOperations != 1 {
		t.Fatalf("unexpected deleted counts: %+v", deleted)
	}
	assertTableCount(t, ctx, db, "orders", "id = 'order-archive-eligible'", 0)
	assertTableCount(t, ctx, db, "orders", "id = 'order-archive-cutoff'", 1)
	assertTableCount(t, ctx, db, "orders", "id = 'order-archive-after'", 1)
	assertTableCount(t, ctx, db, "local_event_log", "id = 'local-event-archive-eligible'", 0)
	assertTableCount(t, ctx, db, "pos_sync_outbox", "id = 'outbox-archive-eligible'", 0)
}

func TestStorageArchiveRepositoryOpenBoundariesBlockWithoutMutation(t *testing.T) {
	db, _ := newSchemaDB(t)
	ctx := t.Context()
	repo := possqlite.NewRepository(db)
	seedClosedOrderFixtureBase(t, ctx, db, "restaurant-boundary", "device-boundary", "employee-boundary", "shift-boundary-closed", "table-boundary")
	insertClosedOrderGraph(t, ctx, db, archiveOrderGraph{
		Suffix:            "boundary-eligible",
		RestaurantID:      "restaurant-boundary",
		DeviceID:          "device-boundary",
		ShiftID:           "shift-boundary-closed",
		TableID:           "table-boundary",
		EmployeeID:        "employee-boundary",
		BusinessDateLocal: "2026-05-01",
		ClosedAt:          "2026-05-01T20:00:00Z",
		OutboxStatus:      domain.OutboxSent,
	})
	execSchema(t, ctx, db, `INSERT INTO shifts(id,restaurant_id,device_id,opened_by_employee_id,status,business_date_local,opened_at,opening_cash_amount,created_at,updated_at) VALUES ('shift-boundary-open','restaurant-boundary','device-boundary','employee-boundary','open','2026-05-01',?,0,?,?)`, schemaTestTime, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO orders(id,edge_order_id,restaurant_id,device_id,shift_id,status,table_id,table_name,guest_count,opened_at,created_at,updated_at) VALUES ('order-boundary-open','edge-order-boundary-open','restaurant-boundary','device-boundary','shift-boundary-open','open','table-boundary','T',1,?,?,?)`, schemaTestTime, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO cash_sessions(id,edge_cash_session_id,restaurant_id,device_id,sales_point_id,shift_id,opened_by_employee_id,status,business_date_local,opening_cash_amount,opened_at,created_at,updated_at) VALUES ('cash-session-boundary-open','edge-cash-session-boundary-open','restaurant-boundary','device-boundary','sales-point-table-boundary','shift-boundary-open','employee-boundary','open','2026-05-01',0,?,?,?)`, schemaTestTime, schemaTestTime, schemaTestTime)

	scope, err := repo.BuildStorageArchiveApplyRuntimeScope(ctx, "2026-05-02")
	if err != nil {
		t.Fatal(err)
	}
	if scope.ActiveOrders != 1 || scope.OpenShifts != 1 || scope.OpenCashSessions != 1 {
		t.Fatalf("expected open boundaries to be counted, got %+v", scope)
	}
	if _, err := repo.ApplyStorageArchiveDestructive(ctx, "2026-05-02"); err == nil {
		t.Fatal("expected destructive apply to be blocked by open boundaries")
	}
	assertTableCount(t, ctx, db, "orders", "id = 'order-boundary-eligible'", 1)
}

func TestCriticalRepositorySchemaIndexesExist(t *testing.T) {
	db, _ := newSchemaDB(t)
	ctx := t.Context()
	for _, index := range []string{
		"checks_business_date_closed_at",
		"checks_order_id_closed_at",
		"orders_closed_restaurant_created_at",
		"payments_business_date_shift_created_at",
		"financial_operations_restaurant_business_date_type_created_at",
		"financial_operations_original_shift_created_at",
		"financial_operations_check_created_at",
		"local_event_log_occurred_at",
		"pos_sync_outbox_status_sequence_no",
		"pos_sync_outbox_pending_retry_sequence",
		"pos_sync_outbox_created_at",
	} {
		var n int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM sqlite_master WHERE type = 'index' AND name = ?`, index).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatalf("expected critical repository index %s to exist", index)
		}
	}
}

type archiveOrderGraph struct {
	Suffix            string
	RestaurantID      string
	DeviceID          string
	ShiftID           string
	TableID           string
	EmployeeID        string
	BusinessDateLocal string
	ClosedAt          string
	OutboxStatus      domain.OutboxStatus
	PayloadMarker     string
}

func seedClosedOrderFixtureBase(t *testing.T, ctx context.Context, db *sql.DB, restaurantID, deviceID, employeeID, shiftID, tableID string) {
	t.Helper()
	roleID := "role-" + employeeID
	hallID := "hall-" + tableID
	sectionID := "section-" + tableID
	salesPointID := "sales-point-" + tableID
	execSchema(t, ctx, db, `INSERT OR IGNORE INTO restaurants(id,name,timezone,currency,active,created_at,updated_at) VALUES (?,?, 'UTC','RUB',1,?,?)`, restaurantID, restaurantID, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO devices(id,restaurant_id,device_code,name,type,active,registered_at,created_at,updated_at) VALUES (?,?,?,?, 'windows',1,?,?,?)`, deviceID, restaurantID, deviceID, deviceID, schemaTestTime, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO roles(id,name,permissions_json,active,created_at,updated_at) VALUES (?,?, '{}',1,?,?)`, roleID, roleID, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO employees(id,restaurant_id,role_id,name,pin_hash,active,created_at,updated_at) VALUES (?,?,?,?, 'hash',1,?,?)`, employeeID, restaurantID, roleID, employeeID, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO halls(id,restaurant_id,name,active,created_at,updated_at) VALUES (?,?,?,1,?,?)`, hallID, restaurantID, hallID, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO restaurant_sections(id,restaurant_id,name,mode,hall_id,is_default,created_at,updated_at) VALUES (?,?,?,'hall_section',?,0,?,?)`, sectionID, restaurantID, sectionID, hallID, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO tables(id,restaurant_id,hall_id,section_id,name,seats,is_default,active,created_at,updated_at) VALUES (?,?,?,?,?,2,0,1,?,?)`, tableID, restaurantID, hallID, sectionID, tableID, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO sales_points(id,restaurant_id,name,analytics_tag,default_table_id,is_active,created_at,updated_at) VALUES (?,?,?,?,?,1,?,?)`, salesPointID, restaurantID, salesPointID, salesPointID, tableID, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO shifts(id,restaurant_id,device_id,opened_by_employee_id,closed_by_employee_id,status,business_date_local,opened_at,closed_at,opening_cash_amount,closing_cash_amount,created_at,updated_at) VALUES (?,?,?,?,?,'closed','2026-05-01',?,?,0,0,?,?)`, shiftID, restaurantID, deviceID, employeeID, employeeID, schemaTestTime, schemaTestTime, schemaTestTime, schemaTestTime)
}

func insertClosedOrderGraph(t *testing.T, ctx context.Context, db *sql.DB, g archiveOrderGraph) {
	t.Helper()
	if g.OutboxStatus == "" {
		g.OutboxStatus = domain.OutboxSent
	}
	if g.PayloadMarker == "" {
		g.PayloadMarker = "safe"
	}
	orderID := "order-" + g.Suffix
	checkID := "check-" + g.Suffix
	precheckID := "precheck-" + g.Suffix
	paymentID := "payment-" + g.Suffix
	attemptID := "payment-attempt-" + g.Suffix
	operationID := "financial-operation-" + g.Suffix
	itemID := "financial-operation-item-" + g.Suffix
	lineID := "order-line-" + g.Suffix
	catalogID := "catalog-" + g.Suffix
	menuID := "menu-" + g.Suffix
	eventID := "local-event-" + g.Suffix
	outboxID := "outbox-" + g.Suffix
	execSchema(t, ctx, db, `INSERT INTO catalog_items(id,type,name,sku,base_unit,active,created_at,updated_at) VALUES (?,'dish',?,?, 'portion',1,?,?)`, catalogID, "Dish "+g.Suffix, "SKU-"+g.Suffix, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO menu_items(id,catalog_item_id,name,price,currency,active,created_at,updated_at) VALUES (?,?,?,?, 'RUB',1,?,?)`, menuID, catalogID, "Dish "+g.Suffix, 100, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO orders(id,edge_order_id,restaurant_id,device_id,shift_id,status,table_id,table_name,guest_count,opened_at,closed_at,created_at,updated_at) VALUES (?,?,?,?,?,'closed',?,'T',1,?,?,?,?)`, orderID, "edge-"+orderID, g.RestaurantID, g.DeviceID, g.ShiftID, g.TableID, g.ClosedAt, g.ClosedAt, g.ClosedAt, g.ClosedAt)
	execSchema(t, ctx, db, `INSERT INTO order_lines(id,order_id,menu_item_id,catalog_item_id,name,quantity,unit_price,total_price,currency_code,status,created_at,updated_at) VALUES (?,?,?,?,?,1,100,100,'RUB','active',?,?)`, lineID, orderID, menuID, catalogID, "Dish "+g.Suffix, g.ClosedAt, g.ClosedAt)
	execSchema(t, ctx, db, `INSERT INTO prechecks(id,order_id,status,version,currency_code,subtotal,discount_total,surcharge_total,tax_total,total,paid_total,remaining_total,snapshot,created_at,issued_at,closed_at) VALUES (?,?,'closed',1,'RUB',100,0,0,0,100,100,0,?,?,?,?)`, precheckID, orderID, `{"precheck":"`+g.PayloadMarker+`"}`, g.ClosedAt, g.ClosedAt, g.ClosedAt)
	execSchema(t, ctx, db, `INSERT INTO payments(id,edge_payment_id,restaurant_id,device_id,shift_id,precheck_id,method,amount,currency,status,business_date_local,created_at,updated_at) VALUES (?,?,?,?,?,?,'cash',100,'RUB','captured',?,?,?)`, paymentID, "edge-"+paymentID, g.RestaurantID, g.DeviceID, g.ShiftID, precheckID, g.BusinessDateLocal, g.ClosedAt, g.ClosedAt)
	execSchema(t, ctx, db, `INSERT INTO payment_attempts(id,payment_id,attempt_no,method,amount,currency,status,attempted_at,created_at) VALUES (?,?,1,'cash',100,'RUB','captured',?,?)`, attemptID, paymentID, g.ClosedAt, g.ClosedAt)
	execSchema(t, ctx, db, `INSERT INTO checks(id,order_id,status,currency_code,subtotal,discount_total,surcharge_total,tax_total,total,paid_total,remaining_total,business_date_local,closed_at,snapshot,created_at,updated_at) VALUES (?,?,'paid','RUB',100,0,0,0,100,100,0,?,?,?,?,?)`, checkID, orderID, g.BusinessDateLocal, g.ClosedAt, `{"check":"`+g.PayloadMarker+`"}`, g.ClosedAt, g.ClosedAt)
	execSchema(t, ctx, db, `INSERT INTO financial_operations(id,edge_operation_id,restaurant_id,device_id,shift_id,original_shift_id,check_id,precheck_id,operation_type,operation_kind,status,amount,currency,business_date_local,inventory_disposition,reason,created_by_employee_id,snapshot,created_at) VALUES (?,?,?,?,?,?,?,?,'refund','partial','recorded',10,'RUB',?,'no_stock_effect','refund',?,'{}',?)`, operationID, "edge-"+operationID, g.RestaurantID, g.DeviceID, g.ShiftID, g.ShiftID, checkID, precheckID, g.BusinessDateLocal, g.EmployeeID, g.ClosedAt)
	execSchema(t, ctx, db, `INSERT INTO financial_operation_items(id,operation_id,scope,payment_id,amount,currency,tax_amount,snapshot,created_at) VALUES (?,?,'payment',?,10,'RUB',0,'{}',?)`, itemID, operationID, paymentID, g.ClosedAt)
	execSchema(t, ctx, db, `INSERT INTO local_event_log(id,event_id,command_id,envelope_version,event_type,aggregate_type,aggregate_id,restaurant_id,device_id,node_device_id,payload_json,occurred_at,created_at) VALUES (?,?,?,?, 'OrderClosed','Order',?,?,?,?,?,?,?)`, eventID, "edge-event-"+g.Suffix, "cmd-event-"+g.Suffix, domain.SyncEnvelopeVersion, orderID, g.RestaurantID, g.DeviceID, g.DeviceID, `{"payload":"`+g.PayloadMarker+`"}`, g.ClosedAt, g.ClosedAt)
	execSchema(t, ctx, db, `INSERT INTO pos_sync_outbox(id,command_id,sequence_no,origin,restaurant_id,device_id,node_device_id,aggregate_type,aggregate_id,command_type,sync_direction,payload_json,status,sent_at,created_at,updated_at) VALUES (?,?,(SELECT COALESCE(MAX(sequence_no),0)+1 FROM pos_sync_outbox),'edge_device',?,?,?,'Order',?,'OrderClosed','edge_to_cloud',?,?,CASE WHEN ? = 'sent' THEN ? ELSE NULL END,?,?)`, outboxID, "cmd-outbox-"+g.Suffix, g.RestaurantID, g.DeviceID, g.DeviceID, orderID, `{"payload":"`+g.PayloadMarker+`"}`, string(g.OutboxStatus), string(g.OutboxStatus), g.ClosedAt, g.ClosedAt, g.ClosedAt)
}

func financialOperationForTest(id, edgeID, restaurantID, deviceID, shiftID, checkID, precheckID string, typ domain.FinancialOperationType, businessDateLocal, createdAt string) domain.FinancialOperation {
	return domain.FinancialOperation{
		ID:                   id,
		EdgeOperationID:      edgeID,
		RestaurantID:         restaurantID,
		DeviceID:             deviceID,
		ShiftID:              shiftID,
		OriginalShiftID:      shiftID,
		CheckID:              checkID,
		PrecheckID:           precheckID,
		Type:                 typ,
		Kind:                 domain.FinancialOperationPartial,
		Status:               domain.FinancialOperationRecorded,
		Amount:               10,
		Currency:             "RUB",
		BusinessDateLocal:    businessDateLocal,
		InventoryDisposition: domain.InventoryNoStockEffect,
		Reason:               "filter test",
		CreatedByEmployeeID:  strings.TrimPrefix(strings.Replace(shiftID, "shift", "employee", 1), "employee-archive"),
		Snapshot:             json.RawMessage(`{}`),
		CreatedAt:            parseTestTime(createdAt),
	}
}

func paymentForTxTest(id, edgeID, precheckID string, now time.Time) domain.Payment {
	return domain.Payment{
		ID:                id,
		EdgePaymentID:     edgeID,
		RestaurantID:      "restaurant-1",
		DeviceID:          "device-1",
		ShiftID:           "shift-1",
		PrecheckID:        precheckID,
		Method:            domain.PaymentCash,
		Amount:            100,
		Currency:          "RUB",
		Status:            domain.PaymentCaptured,
		BusinessDateLocal: "2026-05-04",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func paymentAttemptForTxTest(id, paymentID string, attemptNo int, now time.Time) domain.PaymentAttempt {
	return domain.PaymentAttempt{
		ID:          id,
		PaymentID:   paymentID,
		AttemptNo:   attemptNo,
		Method:      domain.PaymentCash,
		Amount:      100,
		Currency:    "RUB",
		Status:      domain.PaymentCaptured,
		AttemptedAt: now,
		CreatedAt:   now,
	}
}

func assertDuplicateRepositoryError(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, domain.ErrDuplicate) {
		t.Fatalf("expected duplicate repository error, got %v", err)
	}
	for _, marker := range []string{"secret-pin-marker", "raw-sync-payload-marker", "payment-sensitive-marker", "INSERT ", "SELECT ", "\n"} {
		if strings.Contains(err.Error(), marker) {
			t.Fatalf("repository error leaked forbidden marker %q", marker)
		}
	}
}

func assertTableCount(t *testing.T, ctx context.Context, db *sql.DB, table, where string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM `+table+` WHERE `+where).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("expected %s rows matching %s = %d, got %d", table, where, want, got)
	}
}

func assertOrderSummaryIDs(t *testing.T, got []domain.OrderSummary, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected order summaries %v, got %+v", want, got)
	}
	for i := range want {
		if got[i].ID != want[i] {
			t.Fatalf("expected order summaries %v, got %+v", want, got)
		}
	}
}

func assertFinancialOperationIDs(t *testing.T, got []domain.FinancialOperation, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected financial operations %v, got %+v", want, got)
	}
	for i := range want {
		if got[i].ID != want[i] {
			t.Fatalf("expected financial operations %v, got %+v", want, got)
		}
	}
}

func stringPtrForTest(v string) *string {
	return &v
}

func parseTestTime(v string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		panic(err)
	}
	return t
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

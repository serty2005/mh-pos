package app_test

import (
	"errors"
	"testing"
	"time"

	platformsqlite "pos-backend/internal/platform/sqlite"
	"pos-backend/internal/pos/app"
	"pos-backend/internal/pos/domain"
)

// TestPrintConfirmationConfirmsWithinWaitWindowWhenWorkerProcessesConcurrently проверяет
// собственно bounded HTTP wait в CapturePayment: пока основная горутина ждёт
// print_confirmed_at, фоновая горутина крутит print worker и успешно печатает
// check_nonfiscal target — CapturePayment должен вернуть confirmed=true, не дожидаясь
// полного таймаута.
func TestPrintConfirmationConfirmsWithinWaitWindowWhenWorkerProcessesConcurrently(t *testing.T) {
	f := newFixture(t)
	clock := &printTestClock{now: fixedClock{}.Now()}
	sender := &printTestSender{}
	f.service = app.NewServiceWithOptions(f.repo, platformsqlite.NewTxManager(f.db), &testIDs{n: 52000}, clock, app.ServiceOptions{
		StorageArchiveDir:     f.archiveDir,
		PrintSender:           sender,
		PrintConfirmationWait: 1500 * time.Millisecond,
	})
	seedDefaultReceiptPrinters(t, f)
	seedDefaultPrintTemplates(t, f)
	f.openShift(t)
	f.openCashSession(t)

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				_, _ = f.service.ProcessNextPrintJob(f.ctx, "worker-confirm-bg")
			}
		}
	}()
	defer func() {
		close(stop)
		<-done
	}()

	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	payment, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	if payment.PrintConfirmation == nil || !payment.PrintConfirmation.Confirmed {
		t.Fatalf("expected print confirmation within wait window, got %+v", payment.PrintConfirmation)
	}

	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if check.PrintConfirmedAt == nil {
		t.Fatal("expected check.print_confirmed_at to be stamped by worker")
	}

	confirmation, err := f.service.GetPrintConfirmationAsOperator(f.ctx, check.ID, f.edgeMeta())
	if err != nil {
		t.Fatal(err)
	}
	if !confirmation.Confirmed {
		t.Fatalf("expected GetPrintConfirmationAsOperator to reflect confirmed state, got %+v", confirmation)
	}
}

// TestPrintConfirmationTimesOutWhenPrinterOffline проверяет, что при недоступном
// принтере CapturePayment не блокируется дольше настроенного таймаута и возвращает
// payment с confirmed=false, при этом сама оплата остаётся проведённой.
func TestPrintConfirmationTimesOutWhenPrinterOffline(t *testing.T) {
	f := newFixture(t)
	clock := &printTestClock{now: fixedClock{}.Now()}
	sender := &printTestSender{err: errors.New("printer offline")}
	f.service = app.NewServiceWithOptions(f.repo, platformsqlite.NewTxManager(f.db), &testIDs{n: 52200}, clock, app.ServiceOptions{
		StorageArchiveDir:     f.archiveDir,
		PrintSender:           sender,
		PrintConfirmationWait: 150 * time.Millisecond,
	})
	seedDefaultReceiptPrinters(t, f)
	seedDefaultPrintTemplates(t, f)
	f.createPaidOrder(t)

	paymentsBefore := countRows(t, f, "payments")
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	payment, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("expected bounded wait close to configured timeout, took %s", elapsed)
	}
	if payment.PrintConfirmation == nil || payment.PrintConfirmation.Confirmed {
		t.Fatalf("expected unconfirmed print after timeout, got %+v", payment.PrintConfirmation)
	}
	if got := countRows(t, f, "payments"); got != paymentsBefore+1 {
		t.Fatalf("expected payment to remain captured despite print timeout: before=%d after=%d", paymentsBefore, got)
	}
}

// TestRetryPrintConfirmationRebuildsTargetsFromCurrentRoutes проверяет, что job-level
// retry подтверждения печати пересобирает targets из ТЕКУЩИХ print_routes: если принтер
// физически заменили после серии неудач, следующий retry должен слать уже на новый.
func TestRetryPrintConfirmationRebuildsTargetsFromCurrentRoutes(t *testing.T) {
	f := newFixture(t)
	clock := &printTestClock{now: fixedClock{}.Now()}
	sender := &printTestSender{err: errors.New("printer offline")}
	enablePrintQueue(t, f, clock, sender)
	seedDefaultPrintTemplates(t, f)
	order, check := f.createPaidOrder(t)
	_ = order

	for i := 0; i < 5; i++ {
		if _, err := f.service.ProcessNextPrintJob(f.ctx, "worker-retry-setup"); err != nil {
			t.Fatal(err)
		}
		clock.Advance(20 * time.Second)
	}
	confirmationBefore, err := f.service.GetPrintConfirmationAsOperator(f.ctx, check.ID, f.edgeMeta())
	if err != nil {
		t.Fatal(err)
	}
	if confirmationBefore.Confirmed {
		t.Fatal("expected check to remain unconfirmed while printer is offline")
	}

	// "Заменили" неисправный принтер: перенастраиваем route check_nonfiscal на другой
	// принтер и чиним sender.
	sender.err = nil
	if _, err := f.db.ExecContext(f.ctx, `UPDATE print_routes SET printer_id = 'printer-ticket' WHERE document_type = 'check_nonfiscal'`); err != nil {
		t.Fatal(err)
	}

	if _, err := f.service.RetryPrintConfirmationAsOperator(f.ctx, check.ID, f.managerEdgeMetaCommand(t, "cmd-retry-confirmation")); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		processed, err := f.service.ProcessNextPrintJob(f.ctx, "worker-retry-after")
		if err != nil {
			t.Fatal(err)
		}
		if !processed {
			break
		}
	}
	confirmationAfter, err := f.service.GetPrintConfirmationAsOperator(f.ctx, check.ID, f.edgeMeta())
	if err != nil {
		t.Fatal(err)
	}
	if !confirmationAfter.Confirmed {
		t.Fatalf("expected confirmation after routing to a working printer, got %+v", confirmationAfter)
	}
	foundNewPrinter := false
	for _, target := range confirmationAfter.Targets {
		if target.PrinterID == "printer-ticket" && target.Status == "succeeded" {
			foundNewPrinter = true
		}
	}
	if !foundNewPrinter {
		t.Fatalf("expected a succeeded target on the newly assigned printer, got %+v", confirmationAfter.Targets)
	}
}

// TestCancelUnconfirmedOrderRefundsVoidsTicketsAndCancelsOrder проверяет полный
// cancel-unconfirmed flow: компенсирующая финансовая операция на всю сумму чека, void
// всех выпущенных билетов и soft-cancel заказа — без повторной выдачи билетов/оплаты.
func TestCancelUnconfirmedOrderRefundsVoidsTicketsAndCancelsOrder(t *testing.T) {
	f := newFixture(t)
	clock := &printTestClock{now: fixedClock{}.Now()}
	sender := &printTestSender{err: errors.New("printer offline")}
	f.service = app.NewServiceWithOptions(f.repo, platformsqlite.NewTxManager(f.db), &testIDs{n: 52400}, clock, app.ServiceOptions{
		StorageArchiveDir:     f.archiveDir,
		PrintSender:           sender,
		PrintMaxAttempts:      1,
		PrintConfirmationWait: 0,
	})
	seedDefaultReceiptPrinters(t, f)
	seedDefaultPrintTemplates(t, f)
	f.openShift(t)
	f.openCashSession(t)
	qr := f.seedQRMenuItem(t, "cancel-unconfirmed", "business_date", nil)
	order, check, _ := f.payQRSale(t, qr, "cmd-cancel-unconfirmed-pay")

	confirmation, err := f.service.GetPrintConfirmationAsOperator(f.ctx, check.ID, f.edgeMeta())
	if err != nil {
		t.Fatal(err)
	}
	if confirmation.Confirmed {
		t.Fatal("expected unconfirmed print before cancel-unconfirmed")
	}

	result, err := f.service.CancelUnconfirmedOrder(f.ctx, app.CancelUnconfirmedOrderCommand{
		CommandMeta: f.edgeMetaCommand("cmd-cancel-unconfirmed-1"),
		OrderID:     order.ID,
		ManagerPIN:  "2468",
		Reason:      "printer offline, guest left",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.OrderCancelled || result.CancelledAt == nil {
		t.Fatalf("expected order cancelled, got %+v", result)
	}

	operations, err := f.service.ListFinancialOperationsByCheckAsOperator(f.ctx, check.ID, f.managerEdgeMetaCommand(t, "cmd-list-ops"), 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 1 || operations[0].Type != domain.FinancialOperationCancellation || operations[0].Amount != check.Total {
		t.Fatalf("expected one full cancellation operation for check total, got %+v", operations)
	}

	var voided, active int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM ticket_units WHERE check_id = ? AND status = 'voided'`, check.ID).Scan(&voided); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM ticket_units WHERE check_id = ? AND status = 'active'`, check.ID).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if voided == 0 || active != 0 {
		t.Fatalf("expected all tickets of the check voided, voided=%d active=%d", voided, active)
	}

	activeOrders, err := f.repo.ListActiveOrdersByRestaurantAndHall(f.ctx, f.restaurant.ID, f.hall.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range activeOrders {
		if o.ID == order.ID {
			t.Fatalf("expected cancelled order to disappear from active orders list")
		}
	}

	// Повтор с тем же command_id не должен ничего мутировать повторно (идемпотентность).
	if _, err := f.service.CancelUnconfirmedOrder(f.ctx, app.CancelUnconfirmedOrderCommand{
		CommandMeta: f.edgeMetaCommand("cmd-cancel-unconfirmed-1"),
		OrderID:     order.ID,
		ManagerPIN:  "2468",
		Reason:      "printer offline, guest left",
	}); err == nil {
		t.Fatal("expected replay of the same command_id to be rejected")
	}
	if got := countRows(t, f, "financial_operations"); got != 1 {
		t.Fatalf("expected exactly one financial operation after replay attempt, got %d", got)
	}
}

// TestCancelUnconfirmedOrderUsesCashShiftWhenOrderWasOpenedByWaiter воспроизводит
// живой table-service контур: заказ открывает официант, а чек закрывает кассир с
// открытой кассовой сменой. Cancel-unconfirmed должен валидировать cash boundary,
// не личную смену официанта.
func TestCancelUnconfirmedOrderUsesCashShiftWhenOrderWasOpenedByWaiter(t *testing.T) {
	f := newFixture(t)
	clock := &printTestClock{now: fixedClock{}.Now()}
	sender := &printTestSender{err: errors.New("printer offline")}
	f.service = app.NewServiceWithOptions(f.repo, platformsqlite.NewTxManager(f.db), &testIDs{n: 52500}, clock, app.ServiceOptions{
		StorageArchiveDir:     f.archiveDir,
		PrintSender:           sender,
		PrintMaxAttempts:      1,
		PrintConfirmationWait: 0,
	})
	seedDefaultReceiptPrinters(t, f)
	seedDefaultPrintTemplates(t, f)

	cashier, err := f.service.CreateEmployee(f.ctx, app.CreateEmployeeCommand{
		CommandMeta:  seedMeta(f.device.ID),
		RestaurantID: f.restaurant.ID,
		RoleID:       f.employee.RoleID,
		Name:         "Nina",
		PINHash:      testPINHash(t, "3333", "cashier-3333-salt"),
	})
	if err != nil {
		t.Fatal(err)
	}
	login, err := f.service.PinLogin(f.ctx, app.PinLoginCommand{
		CommandMeta: app.CommandMeta{
			CommandID:      "cmd-login-cashier-3333",
			NodeDeviceID:   f.device.ID,
			DeviceID:       f.device.ID,
			ClientDeviceID: "client-device-cashier",
			Origin:         app.OriginEdgeDevice,
		},
		PIN: "3333",
	})
	if err != nil {
		t.Fatal(err)
	}
	cashierMeta := func(commandID string) app.CommandMeta {
		meta := edgeMeta(f.device.ID)
		meta.CommandID = commandID
		meta.ClientDeviceID = "client-device-cashier"
		meta.ActorEmployeeID = cashier.ID
		meta.SessionID = login.Session.ID
		return meta
	}

	waiterShift := f.openShift(t)
	cashierShift, err := f.service.OpenShift(f.ctx, app.OpenShiftCommand{
		CommandMeta:        cashierMeta("cmd-open-cashier-shift"),
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: cashier.ID,
		OpeningCashAmount:  0,
	})
	if err != nil {
		t.Fatal(err)
	}
	cashSession, err := f.service.OpenCashSession(f.ctx, app.OpenCashSessionCommand{
		CommandMeta:        cashierMeta("cmd-open-cashier-cash"),
		RestaurantID:       f.restaurant.ID,
		SalesPointID:       f.salesPointID,
		OpenedByEmployeeID: cashier.ID,
		OpeningCashAmount:  0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if waiterShift.ID == cashierShift.ID || cashSession.ShiftID != cashierShift.ID {
		t.Fatalf("test setup must use different waiter/cashier shifts, waiter=%s cashier=%s cash=%s", waiterShift.ID, cashierShift.ID, cashSession.ShiftID)
	}

	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMetaCommand("cmd-waiter-order"), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMetaCommand("cmd-waiter-line"), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMetaCommand("cmd-waiter-precheck"), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{
		CommandMeta: cashierMeta("cmd-cashier-payment"),
		PrecheckID:  precheck.ID,
		Method:      domain.PaymentCash,
		Amount:      precheck.Total,
		Currency:    "RUB",
	}); err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if check.PrintConfirmedAt != nil {
		t.Fatal("expected print to remain unconfirmed")
	}

	if _, err := f.service.CancelUnconfirmedOrder(f.ctx, app.CancelUnconfirmedOrderCommand{
		CommandMeta: f.managerEdgeMetaCommand(t, "cmd-cancel-unconfirmed-waiter-cashier"),
		OrderID:     order.ID,
		ManagerPIN:  "2468",
		Reason:      "printer offline, guest left",
	}); err != nil {
		t.Fatal(err)
	}

	operations, err := f.repo.ListFinancialOperationsByCheck(f.ctx, check.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 1 {
		t.Fatalf("expected one cancellation operation, got %+v", operations)
	}
	if operations[0].ShiftID != cashierShift.ID || operations[0].OriginalShiftID != waiterShift.ID {
		t.Fatalf("unexpected operation shift boundary: shift_id=%s original_shift_id=%s", operations[0].ShiftID, operations[0].OriginalShiftID)
	}
}

// TestCancelUnconfirmedOrderRejectsWrongPINAndConfirmedCheck проверяет RBAC и
// защитный gate: неверный/не-manager PIN и уже подтверждённая печать блокируют операцию.
func TestCancelUnconfirmedOrderRejectsWrongPINAndConfirmedCheck(t *testing.T) {
	f := newFixture(t)
	clock := &printTestClock{now: fixedClock{}.Now()}
	sender := &printTestSender{err: errors.New("printer offline")}
	f.service = app.NewServiceWithOptions(f.repo, platformsqlite.NewTxManager(f.db), &testIDs{n: 52600}, clock, app.ServiceOptions{
		StorageArchiveDir:     f.archiveDir,
		PrintSender:           sender,
		PrintConfirmationWait: 0,
	})
	seedDefaultReceiptPrinters(t, f)
	seedDefaultPrintTemplates(t, f)
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}

	if _, err := f.service.CancelUnconfirmedOrder(f.ctx, app.CancelUnconfirmedOrderCommand{
		CommandMeta: f.edgeMetaCommand("cmd-cancel-wrong-pin"),
		OrderID:     order.ID,
		ManagerPIN:  "0000",
		Reason:      "printer offline",
	}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden for wrong manager pin, got %v", err)
	}

	// Дождаться подтверждения печати вручную (успешный sender), затем убедиться, что
	// cancel-unconfirmed для уже подтверждённого чека отклоняется.
	sender.err = nil
	for i := 0; i < 5; i++ {
		processed, err := f.service.ProcessNextPrintJob(f.ctx, "worker-confirm-for-reject-test")
		if err != nil {
			t.Fatal(err)
		}
		if !processed {
			break
		}
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if check.PrintConfirmedAt == nil {
		t.Fatal("expected print to be confirmed for this sub-test setup")
	}
	if _, err := f.service.CancelUnconfirmedOrder(f.ctx, app.CancelUnconfirmedOrderCommand{
		CommandMeta: f.edgeMetaCommand("cmd-cancel-confirmed-check"),
		OrderID:     order.ID,
		ManagerPIN:  "2468",
		Reason:      "should not be allowed",
	}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict for already-confirmed check, got %v", err)
	}
}

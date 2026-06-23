package check

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	txmanager "pos-backend/internal/platform/tx"
	"pos-backend/internal/pos/app/shared"
	appticket "pos-backend/internal/pos/app/ticket"
	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/domain/order"
	"pos-backend/internal/pos/ports"
)

// ticketIssuer выпускает QR-билеты после закрытия final check внутри текущей транзакции.
type ticketIssuer interface {
	IssueForClosedCheck(ctx context.Context, in appticket.IssueInput) ([]domain.TicketUnit, error)
}

const (
	defaultClosedOrdersLimit = 50
	maxClosedOrdersLimit     = 100

	defaultFinancialOperationsLimit = 50
	maxFinancialOperationsLimit     = 200
)

type Service struct {
	repo    ports.Repository
	tx      txmanager.Manager
	ids     idgen.Generator
	clock   clock.Clock
	tickets ticketIssuer
}

func NewService(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock, tickets ticketIssuer) *Service {
	return &Service{repo: repo, tx: tx, ids: ids, clock: clock, tickets: tickets}
}

type CapturePaymentCommand struct {
	shared.CommandMeta
	PrecheckID            string               `json:"precheck_id"`
	Method                domain.PaymentMethod `json:"method"`
	Amount                int64                `json:"amount"`
	Currency              string               `json:"currency"`
	ProviderName          string               `json:"provider_name,omitempty"`
	ProviderTransactionID string               `json:"provider_transaction_id,omitempty"`
	ProviderReference     string               `json:"provider_reference,omitempty"`
	FingerprintHash       string               `json:"fingerprint_hash,omitempty"`
}

type ReprintCheckCommand struct {
	shared.CommandMeta
	CheckID string `json:"check_id"`
}

type RefundPaymentCommand struct {
	shared.CommandMeta
	PaymentID string `json:"payment_id"`
	Reason    string `json:"reason,omitempty"`
}

type ListClosedOrdersCommand struct {
	shared.CommandMeta
	BusinessDateLocal     string
	FromBusinessDateLocal string
	ToBusinessDateLocal   string
	ShiftID               string
	DeviceID              string
	CheckID               string
	Limit                 int
	Offset                int
}

// ListFinancialOperationsCommand задает read-only фильтры ledger history для POS UI/reporting.
type ListFinancialOperationsCommand struct {
	shared.CommandMeta
	CheckID          string
	BusinessDateFrom string
	BusinessDateTo   string
	OperationType    domain.FinancialOperationType
	ShiftID          string
	OriginalShiftID  string
	Limit            int
	Offset           int
}

func (s *Service) GetCheck(ctx context.Context, id string) (*domain.Check, error) {
	return s.repo.GetCheck(ctx, id)
}

// GetCheckAsOperator загружает final check для аутентифицированных операторских сценариев с проверкой RBAC.
func (s *Service) GetCheckAsOperator(ctx context.Context, id string, meta shared.CommandMeta) (*domain.Check, error) {
	if _, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionCheckView)); err != nil {
		return nil, err
	}
	return s.GetCheck(ctx, id)
}

// ListFinancialOperationsByCheckAsOperator возвращает append-only ledger операций по final check
// только внутри restaurant scope текущего оператора.
func (s *Service) ListFinancialOperationsByCheckAsOperator(ctx context.Context, checkID string, meta shared.CommandMeta, limit, offset int) ([]domain.FinancialOperation, error) {
	cmd := ListFinancialOperationsCommand{
		CommandMeta: meta,
		CheckID:     checkID,
		Limit:       limit,
		Offset:      offset,
	}
	return s.ListFinancialOperationsAsOperator(ctx, cmd)
}

// ListFinancialOperationsAsOperator возвращает bounded append-only ledger history в restaurant scope оператора.
func (s *Service) ListFinancialOperationsAsOperator(ctx context.Context, cmd ListFinancialOperationsCommand) ([]domain.FinancialOperation, error) {
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionCheckView))
	if err != nil {
		return nil, err
	}
	query, err := normalizeFinancialOperationListQuery(cmd, operator.Employee.RestaurantID)
	if err != nil {
		return nil, err
	}
	if query.CheckID != "" {
		check, err := s.repo.GetCheck(ctx, query.CheckID)
		if err != nil {
			return nil, err
		}
		order, err := s.repo.GetOrder(ctx, check.OrderID)
		if err != nil {
			return nil, err
		}
		if order.RestaurantID != operator.Employee.RestaurantID {
			return nil, fmt.Errorf("%w: check is outside operator restaurant", domain.ErrForbidden)
		}
	}
	return s.repo.ListFinancialOperations(ctx, query)
}

func normalizeFinancialOperationListQuery(cmd ListFinancialOperationsCommand, restaurantID string) (domain.FinancialOperationListQuery, error) {
	limit := cmd.Limit
	if limit <= 0 {
		limit = defaultFinancialOperationsLimit
	}
	if limit > maxFinancialOperationsLimit {
		limit = maxFinancialOperationsLimit
	}
	if cmd.Offset < 0 {
		return domain.FinancialOperationListQuery{}, fmt.Errorf("%w: financial operations offset must be non-negative", domain.ErrInvalid)
	}
	fromDate := strings.TrimSpace(cmd.BusinessDateFrom)
	toDate := strings.TrimSpace(cmd.BusinessDateTo)
	if fromDate != "" {
		if err := validateBusinessDateFilter(fromDate); err != nil {
			return domain.FinancialOperationListQuery{}, err
		}
	}
	if toDate != "" {
		if err := validateBusinessDateFilter(toDate); err != nil {
			return domain.FinancialOperationListQuery{}, err
		}
	}
	if fromDate != "" && toDate != "" && fromDate > toDate {
		return domain.FinancialOperationListQuery{}, fmt.Errorf("%w: business_date_from must be before business_date_to", domain.ErrInvalid)
	}
	if cmd.OperationType != "" && cmd.OperationType != domain.FinancialOperationCancellation && cmd.OperationType != domain.FinancialOperationRefund {
		return domain.FinancialOperationListQuery{}, fmt.Errorf("%w: operation_type must be cancellation or refund", domain.ErrInvalid)
	}
	return domain.FinancialOperationListQuery{
		RestaurantID:     strings.TrimSpace(restaurantID),
		CheckID:          strings.TrimSpace(cmd.CheckID),
		BusinessDateFrom: fromDate,
		BusinessDateTo:   toDate,
		OperationType:    cmd.OperationType,
		ShiftID:          strings.TrimSpace(cmd.ShiftID),
		OriginalShiftID:  strings.TrimSpace(cmd.OriginalShiftID),
		Limit:            limit,
		Offset:           cmd.Offset,
	}, nil
}

func (s *Service) ListClosedOrders(ctx context.Context, cmd ListClosedOrdersCommand) ([]order.OrderSummary, error) {
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionCheckView))
	if err != nil {
		return nil, err
	}
	query, err := normalizeClosedOrderListQuery(cmd, operator.Employee.RestaurantID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListClosedOrders(ctx, query)
}

func normalizeClosedOrderListQuery(cmd ListClosedOrdersCommand, restaurantID string) (order.ClosedOrderListQuery, error) {
	limit := cmd.Limit
	if limit <= 0 {
		limit = defaultClosedOrdersLimit
	}
	if limit > maxClosedOrdersLimit {
		limit = maxClosedOrdersLimit
	}
	if cmd.Offset < 0 {
		return order.ClosedOrderListQuery{}, fmt.Errorf("%w: closed orders offset must be non-negative", domain.ErrInvalid)
	}
	businessDate := strings.TrimSpace(cmd.BusinessDateLocal)
	fromDate := strings.TrimSpace(cmd.FromBusinessDateLocal)
	toDate := strings.TrimSpace(cmd.ToBusinessDateLocal)
	if businessDate != "" {
		if fromDate != "" || toDate != "" {
			return order.ClosedOrderListQuery{}, fmt.Errorf("%w: business_date_local cannot be combined with range filters", domain.ErrInvalid)
		}
		if err := validateBusinessDateFilter(businessDate); err != nil {
			return order.ClosedOrderListQuery{}, err
		}
	}
	if fromDate != "" {
		if err := validateBusinessDateFilter(fromDate); err != nil {
			return order.ClosedOrderListQuery{}, err
		}
	}
	if toDate != "" {
		if err := validateBusinessDateFilter(toDate); err != nil {
			return order.ClosedOrderListQuery{}, err
		}
	}
	if fromDate != "" && toDate != "" && fromDate > toDate {
		return order.ClosedOrderListQuery{}, fmt.Errorf("%w: from_business_date_local must be before to_business_date_local", domain.ErrInvalid)
	}
	return order.ClosedOrderListQuery{
		RestaurantID:          strings.TrimSpace(restaurantID),
		BusinessDateLocal:     businessDate,
		FromBusinessDateLocal: fromDate,
		ToBusinessDateLocal:   toDate,
		ShiftID:               strings.TrimSpace(cmd.ShiftID),
		DeviceID:              strings.TrimSpace(cmd.DeviceID),
		CheckID:               strings.TrimSpace(cmd.CheckID),
		Limit:                 limit,
		Offset:                cmd.Offset,
	}, nil
}

func validateBusinessDateFilter(value string) error {
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf("%w: business date filters must use YYYY-MM-DD", domain.ErrInvalid)
	}
	return nil
}

func (s *Service) CapturePayment(ctx context.Context, cmd CapturePaymentCommand) (*domain.Payment, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.PrecheckID) == "" || cmd.Amount <= 0 || strings.TrimSpace(cmd.Currency) == "" {
		return nil, fmt.Errorf("%w: precheck_id, positive amount and currency are required", domain.ErrInvalid)
	}
	currency, err := shared.ValidateCurrencyCode(cmd.Currency)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	if cmd.Method != domain.PaymentCash && cmd.Method != domain.PaymentCard && cmd.Method != domain.PaymentOther {
		return nil, fmt.Errorf("%w: unsupported payment method", domain.ErrInvalid)
	}
	if strings.TrimSpace(cmd.CommandID) == "" {
		cmd.CommandID = s.ids.NewID()
	}
	now := s.clock.Now()
	var payment *domain.Payment
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if _, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(requiredPaymentPermission(cmd.Method))); err != nil {
			return err
		}
		precheck, err := s.repo.GetPrecheck(ctx, cmd.PrecheckID)
		if err != nil {
			return err
		}
		order, err := s.repo.GetOrder(ctx, precheck.OrderID)
		if err != nil {
			return err
		}
		if order.DeviceID != cmd.DeviceID {
			return fmt.Errorf("%w: payment device does not match order device", domain.ErrConflict)
		}
		if precheck.CurrencyCode != "" && currency != precheck.CurrencyCode {
			return fmt.Errorf("%w: payment currency does not match precheck currency", domain.ErrConflict)
		}
		cashSession, err := s.repo.GetOpenCashSessionByDevice(ctx, cmd.DeviceID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return fmt.Errorf("%w: оплата требует открытую кассовую смену", domain.ErrConflict)
			}
			return err
		}
		if cashSession.RestaurantID != order.RestaurantID {
			return fmt.Errorf("%w: кассовая смена оплаты не совпадает с рестораном заказа", domain.ErrConflict)
		}
		// Оплата относится к текущей кассовой смене оператора, а заказ сохраняет исходную личную смену официанта.
		paymentShiftID := cashSession.ShiftID
		restaurant, err := s.repo.GetRestaurant(ctx, order.RestaurantID)
		if err != nil {
			return err
		}
		businessDate, err := shared.BusinessDateLocal(*restaurant, now)
		if err != nil {
			return err
		}
		if err := precheck.ApplyCapturedPayment(cmd.Amount, now); err != nil {
			return err
		}
		active, err := s.repo.GetActivePrecheckByOrder(ctx, order.ID)
		if err != nil {
			return err
		}
		if active.ID != precheck.ID {
			return fmt.Errorf("%w: precheck is not active for order", domain.ErrConflict)
		}
		payment = &domain.Payment{
			ID:                    s.ids.NewID(),
			EdgePaymentID:         s.ids.NewID(),
			RestaurantID:          order.RestaurantID,
			DeviceID:              order.DeviceID,
			ShiftID:               paymentShiftID,
			PrecheckID:            precheck.ID,
			Method:                cmd.Method,
			Amount:                cmd.Amount,
			Currency:              currency,
			Status:                domain.PaymentCaptured,
			BusinessDateLocal:     businessDate,
			ProviderName:          optionalString(cmd.ProviderName),
			ProviderTransactionID: optionalString(cmd.ProviderTransactionID),
			ProviderReference:     optionalString(cmd.ProviderReference),
			FingerprintHash:       optionalString(cmd.FingerprintHash),
			CreatedAt:             now,
			UpdatedAt:             now,
		}
		if err := s.repo.CreatePayment(ctx, payment); err != nil {
			return err
		}
		attempt := &domain.PaymentAttempt{
			ID:                    s.ids.NewID(),
			PaymentID:             payment.ID,
			AttemptNo:             1,
			Method:                payment.Method,
			Amount:                payment.Amount,
			Currency:              payment.Currency,
			Status:                domain.PaymentCaptured,
			ProviderName:          payment.ProviderName,
			ProviderTransactionID: payment.ProviderTransactionID,
			ProviderReference:     payment.ProviderReference,
			FingerprintHash:       payment.FingerprintHash,
			AttemptedAt:           now,
			CreatedAt:             now,
		}
		if err := s.repo.CreatePaymentAttempt(ctx, attempt); err != nil {
			return err
		}
		if err := s.repo.UpdatePrecheckPayment(ctx, precheck); err != nil {
			return err
		}
		paymentEvent := struct {
			Payment             *domain.Payment `json:"payment"`
			PrecheckID          string          `json:"precheck_id"`
			CurrencyCode        string          `json:"currency_code"`
			GrandTotalMinor     int64           `json:"grand_total_minor"`
			PaidTotalMinor      int64           `json:"paid_total_minor"`
			RemainingTotalMinor int64           `json:"remaining_total_minor"`
		}{
			Payment:             payment,
			PrecheckID:          precheck.ID,
			CurrencyCode:        precheck.CurrencyCode,
			GrandTotalMinor:     precheck.Total,
			PaidTotalMinor:      precheck.PaidTotal,
			RemainingTotalMinor: precheck.RemainingTotal,
		}
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, paymentShiftID, "Payment", payment.ID, "PaymentCaptured", paymentEvent); err != nil {
			return err
		}
		if !precheck.IsFullyPaid() {
			return nil
		}
		if _, err := s.repo.GetCheckByOrder(ctx, order.ID); err == nil {
			return fmt.Errorf("%w: order already has final check", domain.ErrConflict)
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		check := &domain.Check{
			ID:                s.ids.NewID(),
			OrderID:           order.ID,
			Status:            domain.CheckPaid,
			CurrencyCode:      precheck.CurrencyCode,
			Subtotal:          precheck.Subtotal,
			DiscountTotal:     precheck.DiscountTotal,
			SurchargeTotal:    precheck.SurchargeTotal,
			TaxTotal:          precheck.TaxTotal,
			Total:             precheck.Total,
			PaidTotal:         precheck.PaidTotal,
			RemainingTotal:    precheck.RemainingTotal,
			BusinessDateLocal: businessDate,
			ClosedAt:          now,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		payments, err := s.repo.ListPaymentsByPrecheck(ctx, precheck.ID)
		if err != nil {
			return err
		}
		snapshot, err := buildCheckSnapshot(order, precheck, payments, check, now)
		if err != nil {
			return err
		}
		check.Snapshot = snapshot
		if err := s.repo.CreateCheck(ctx, check); err != nil {
			return err
		}
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, paymentShiftID, "Check", check.ID, "CheckCreated", check); err != nil {
			return err
		}
		checkClosedEvent, err := buildCheckClosedEventFromSnapshot(check.Snapshot)
		if err != nil {
			return err
		}
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, paymentShiftID, "Check", check.ID, "CheckClosed", checkClosedEvent); err != nil {
			return err
		}
		order.Status = domain.OrderClosed
		order.ClosedAt = &now
		order.UpdatedAt = now
		if err := s.repo.UpdateOrderClosed(ctx, order); err != nil {
			return err
		}
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, order.ShiftID, "Order", order.ID, "OrderClosed", order); err != nil {
			return err
		}
		// POS-48: QR-билеты выпускаются в той же транзакции, что и закрытие check.
		// Граница financial transaction единая: при откате оплаты билеты не остаются.
		if s.tickets != nil {
			if _, err := s.tickets.IssueForClosedCheck(ctx, appticket.IssueInput{
				Meta:          cmd.CommandMeta,
				RestaurantID:  order.RestaurantID,
				DeviceID:      order.DeviceID,
				CashSessionID: cashSession.ID,
				ShiftID:       paymentShiftID,
				CheckID:       check.ID,
				OrderID:       order.ID,
				SaleDateLocal: businessDate,
				Timezone:      restaurant.Timezone,
				Now:           now,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	return payment, err
}

func (s *Service) ReprintCheck(ctx context.Context, cmd ReprintCheckCommand) (*domain.ReprintDocument, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.CheckID) == "" {
		return nil, fmt.Errorf("%w: check_id is required", domain.ErrInvalid)
	}
	if strings.TrimSpace(cmd.CommandID) == "" {
		cmd.CommandID = s.ids.NewID()
	}
	now := s.clock.Now()
	var document *domain.ReprintDocument
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if _, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionCheckReprint)); err != nil {
			return err
		}
		check, err := s.repo.GetCheck(ctx, cmd.CheckID)
		if err != nil {
			return err
		}
		order, err := s.repo.GetOrder(ctx, check.OrderID)
		if err != nil {
			return err
		}
		if order.DeviceID != cmd.DeviceID {
			return fmt.Errorf("%w: check device does not match command device", domain.ErrConflict)
		}
		if len(check.Snapshot) == 0 || !json.Valid(check.Snapshot) {
			return fmt.Errorf("%w: check snapshot is not available", domain.ErrConflict)
		}
		document = domain.NewReprintDocument("check", check.ID, check.Snapshot, cmd.ActorEmployeeID, now)
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, order.ShiftID, "Check", check.ID, "CheckReprinted", document)
	})
	return document, err
}

func (s *Service) RefundPayment(ctx context.Context, cmd RefundPaymentCommand) (*domain.Payment, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.PaymentID) == "" {
		return nil, fmt.Errorf("%w: payment_id is required", domain.ErrInvalid)
	}
	if strings.TrimSpace(cmd.CommandID) == "" {
		cmd.CommandID = s.ids.NewID()
	}
	payment, err := s.repo.GetPayment(ctx, cmd.PaymentID)
	if err != nil {
		return nil, err
	}
	if payment.Status != domain.PaymentCaptured {
		return nil, fmt.Errorf("%w: payment is not captured", domain.ErrConflict)
	}
	precheck, err := s.repo.GetPrecheck(ctx, payment.PrecheckID)
	if err != nil {
		return nil, err
	}
	check, err := s.repo.GetCheckByOrder(ctx, precheck.OrderID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("%w: refund requires finalized check", domain.ErrConflict)
		}
		return nil, err
	}
	kind := domain.FinancialOperationPartial
	refunded, err := s.repo.SumFinancialOperationAmountByCheck(ctx, check.ID, domain.FinancialOperationRefund)
	if err != nil {
		return nil, err
	}
	cancelled, err := s.repo.SumFinancialOperationAmountByCheck(ctx, check.ID, domain.FinancialOperationCancellation)
	if err != nil {
		return nil, err
	}
	if payment.Amount == check.Total-refunded-cancelled {
		kind = domain.FinancialOperationFull
	}
	reason := strings.TrimSpace(cmd.Reason)
	if reason == "" {
		reason = "legacy_payment_refund"
	}
	_, err = s.RecordRefund(ctx, RecordCheckRefundCommand{
		CommandMeta:          cmd.CommandMeta,
		CheckID:              check.ID,
		OperationKind:        kind,
		InventoryDisposition: domain.InventoryNoStockEffect,
		Reason:               reason,
		Items: []FinancialOperationItemCommand{{
			Scope:     domain.FinancialItemPayment,
			PaymentID: payment.ID,
			Amount:    payment.Amount,
			Currency:  payment.Currency,
		}},
	})
	if err != nil {
		return nil, err
	}
	return payment, nil
}

func optionalString(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func requiredPaymentPermission(method domain.PaymentMethod) shared.PermissionID {
	switch method {
	case domain.PaymentCash:
		return shared.PermissionPaymentCash
	case domain.PaymentCard:
		return shared.PermissionPaymentCardManual
	default:
		return shared.PermissionPaymentOther
	}
}

func requiredRefundPermission(method domain.PaymentMethod) shared.PermissionID {
	return shared.PermissionPaymentRefund
}

type checkSnapshot struct {
	DocumentType      string                 `json:"document_type"`
	CheckID           string                 `json:"check_id"`
	OrderID           string                 `json:"order_id"`
	PrecheckID        string                 `json:"precheck_id"`
	RestaurantID      string                 `json:"restaurant_id"`
	TableID           string                 `json:"table_id"`
	TableName         string                 `json:"table_name"`
	Subtotal          int64                  `json:"subtotal"`
	DiscountTotal     int64                  `json:"discount_total"`
	SurchargeTotal    int64                  `json:"surcharge_total"`
	TaxTotal          int64                  `json:"tax_total"`
	Total             int64                  `json:"total"`
	PaidTotal         int64                  `json:"paid_total"`
	RemainingTotal    int64                  `json:"remaining_total"`
	CurrencyCode      string                 `json:"currency_code"`
	BusinessDateLocal string                 `json:"business_date_local"`
	ClosedAt          string                 `json:"closed_at"`
	Payments          []checkSnapshotPayment `json:"payments"`
	PrecheckSnapshot  json.RawMessage        `json:"precheck_snapshot"`
}

type checkSnapshotPayment struct {
	ID                string `json:"id"`
	Method            string `json:"method"`
	Amount            int64  `json:"amount"`
	Currency          string `json:"currency"`
	BusinessDateLocal string `json:"business_date_local"`
	CapturedAt        string `json:"captured_at"`
}

func buildCheckSnapshot(order *domain.Order, precheck *domain.Precheck, payments []domain.Payment, check *domain.Check, now time.Time) (json.RawMessage, error) {
	snapshot := checkSnapshot{
		DocumentType:      "check",
		CheckID:           check.ID,
		OrderID:           order.ID,
		PrecheckID:        precheck.ID,
		RestaurantID:      order.RestaurantID,
		TableID:           order.TableID,
		TableName:         order.TableName,
		CurrencyCode:      check.CurrencyCode,
		Subtotal:          check.Subtotal,
		DiscountTotal:     check.DiscountTotal,
		SurchargeTotal:    check.SurchargeTotal,
		TaxTotal:          check.TaxTotal,
		Total:             check.Total,
		PaidTotal:         check.PaidTotal,
		RemainingTotal:    check.RemainingTotal,
		BusinessDateLocal: check.BusinessDateLocal,
		ClosedAt:          shared.DBTime(now),
		PrecheckSnapshot:  precheck.Snapshot,
	}
	for _, payment := range payments {
		snapshot.Payments = append(snapshot.Payments, checkSnapshotPayment{
			ID:                payment.ID,
			Method:            string(payment.Method),
			Amount:            payment.Amount,
			Currency:          payment.Currency,
			BusinessDateLocal: payment.BusinessDateLocal,
			CapturedAt:        shared.DBTime(payment.CreatedAt),
		})
	}
	body, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

type checkClosedEvent struct {
	CheckID           string                     `json:"check_id"`
	OrderID           string                     `json:"order_id"`
	PrecheckID        string                     `json:"precheck_id"`
	RestaurantID      string                     `json:"restaurant_id"`
	BusinessDateLocal string                     `json:"business_date_local"`
	ClosedAt          time.Time                  `json:"closed_at"`
	Items             []checkClosedInventoryItem `json:"items"`
}

type checkClosedInventoryItem struct {
	OrderLineID          string                         `json:"order_line_id"`
	CatalogItemID        string                         `json:"catalog_item_id"`
	Quantity             string                         `json:"quantity"`
	UnitCode             string                         `json:"unit_code"`
	RequiredForInventory bool                           `json:"required_for_inventory"`
	Modifiers            []checkClosedInventoryModifier `json:"modifiers,omitempty"`
}

type checkClosedInventoryModifier struct {
	ModifierGroupID  string `json:"modifier_group_id"`
	ModifierOptionID string `json:"modifier_option_id"`
	Name             string `json:"name,omitempty"`
	Quantity         string `json:"quantity"`
	UnitCode         string `json:"unit_code"`
}

type immutableCheckSnapshot struct {
	CheckID           string          `json:"check_id"`
	OrderID           string          `json:"order_id"`
	PrecheckID        string          `json:"precheck_id"`
	RestaurantID      string          `json:"restaurant_id"`
	BusinessDateLocal string          `json:"business_date_local"`
	ClosedAt          string          `json:"closed_at"`
	PrecheckSnapshot  json.RawMessage `json:"precheck_snapshot"`
}

type immutablePrecheckSnapshot struct {
	Lines []immutableCheckLine `json:"lines"`
}

type immutableCheckLine struct {
	OrderLineID   string                       `json:"order_line_id"`
	CatalogItemID string                       `json:"catalog_item_id"`
	Quantity      int64                        `json:"quantity"`
	Modifiers     []immutableCheckLineModifier `json:"modifiers,omitempty"`
}

type immutableCheckLineModifier struct {
	ModifierGroupID  string `json:"modifier_group_id"`
	ModifierOptionID string `json:"modifier_option_id"`
	Name             string `json:"name,omitempty"`
	Quantity         int64  `json:"quantity"`
}

func buildCheckClosedEventFromSnapshot(raw json.RawMessage) (checkClosedEvent, error) {
	if len(raw) == 0 || !json.Valid(raw) {
		return checkClosedEvent{}, fmt.Errorf("%w: check snapshot is not available", domain.ErrConflict)
	}
	var snapshot immutableCheckSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return checkClosedEvent{}, fmt.Errorf("%w: check snapshot is invalid", domain.ErrConflict)
	}
	closedAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(snapshot.ClosedAt))
	if err != nil {
		return checkClosedEvent{}, fmt.Errorf("%w: check snapshot closed_at is invalid", domain.ErrConflict)
	}
	var precheckSnapshot immutablePrecheckSnapshot
	if len(snapshot.PrecheckSnapshot) == 0 || !json.Valid(snapshot.PrecheckSnapshot) {
		return checkClosedEvent{}, fmt.Errorf("%w: precheck snapshot is not available", domain.ErrConflict)
	}
	if err := json.Unmarshal(snapshot.PrecheckSnapshot, &precheckSnapshot); err != nil {
		return checkClosedEvent{}, fmt.Errorf("%w: precheck snapshot is invalid", domain.ErrConflict)
	}
	event := checkClosedEvent{
		CheckID:           strings.TrimSpace(snapshot.CheckID),
		OrderID:           strings.TrimSpace(snapshot.OrderID),
		PrecheckID:        strings.TrimSpace(snapshot.PrecheckID),
		RestaurantID:      strings.TrimSpace(snapshot.RestaurantID),
		BusinessDateLocal: strings.TrimSpace(snapshot.BusinessDateLocal),
		ClosedAt:          closedAt,
		Items:             make([]checkClosedInventoryItem, 0, len(precheckSnapshot.Lines)),
	}
	for _, line := range precheckSnapshot.Lines {
		item := checkClosedInventoryItem{
			OrderLineID:          strings.TrimSpace(line.OrderLineID),
			CatalogItemID:        strings.TrimSpace(line.CatalogItemID),
			Quantity:             formatInventoryQuantity(line.Quantity),
			UnitCode:             "PC",
			RequiredForInventory: true,
		}
		for _, modifier := range line.Modifiers {
			item.Modifiers = append(item.Modifiers, checkClosedInventoryModifier{
				ModifierGroupID:  strings.TrimSpace(modifier.ModifierGroupID),
				ModifierOptionID: strings.TrimSpace(modifier.ModifierOptionID),
				Name:             strings.TrimSpace(modifier.Name),
				Quantity:         formatInventoryQuantity(modifier.Quantity),
				UnitCode:         "PC",
			})
		}
		event.Items = append(event.Items, item)
	}
	if event.CheckID == "" || event.OrderID == "" || event.PrecheckID == "" || event.RestaurantID == "" ||
		event.BusinessDateLocal == "" || len(event.Items) == 0 {
		return checkClosedEvent{}, fmt.Errorf("%w: check snapshot is incomplete", domain.ErrConflict)
	}
	return event, nil
}

func formatInventoryQuantity(quantity int64) string {
	return fmt.Sprintf("%.3f", float64(quantity))
}

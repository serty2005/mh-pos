package check

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	txmanager "pos-backend/internal/platform/tx"
	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/ports"
	"strings"
	"time"
)

type Service struct {
	repo  ports.Repository
	tx    txmanager.Manager
	ids   idgen.Generator
	clock clock.Clock
}

func NewService(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock) *Service {
	return &Service{repo: repo, tx: tx, ids: ids, clock: clock}
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
		cashSession, err := s.repo.GetOpenCashSessionByDevice(ctx, cmd.DeviceID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return fmt.Errorf("%w: оплата требует открытую кассовую смену", domain.ErrConflict)
			}
			return err
		}
		if cashSession.ShiftID != order.ShiftID || cashSession.RestaurantID != order.RestaurantID {
			return fmt.Errorf("%w: кассовая смена оплаты не совпадает с личной сменой заказа", domain.ErrConflict)
		}
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
			ShiftID:               order.ShiftID,
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
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, order.ShiftID, "Payment", payment.ID, "PaymentCaptured", payment); err != nil {
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
			Subtotal:          precheck.Subtotal,
			DiscountTotal:     precheck.DiscountTotal,
			TaxTotal:          precheck.TaxTotal,
			Total:             precheck.Total,
			PaidTotal:         precheck.PaidTotal,
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
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, order.ShiftID, "Check", check.ID, "CheckCreated", check); err != nil {
			return err
		}
		order.Status = domain.OrderClosed
		order.ClosedAt = &now
		order.UpdatedAt = now
		if err := s.repo.UpdateOrderClosed(ctx, order); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, order.ShiftID, "Order", order.ID, "OrderClosed", order)
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
	now := s.clock.Now()
	var payment *domain.Payment
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		var err error
		payment, err = s.repo.GetPayment(ctx, cmd.PaymentID)
		if err != nil {
			return err
		}
		if payment.Status != domain.PaymentCaptured {
			return fmt.Errorf("%w: payment is not captured", domain.ErrConflict)
		}
		precheck, err := s.repo.GetPrecheck(ctx, payment.PrecheckID)
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
		cashSession, err := s.repo.GetOpenCashSessionByDevice(ctx, cmd.DeviceID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return fmt.Errorf("%w: возврат оплаты требует открытую кассовую смену", domain.ErrConflict)
			}
			return err
		}
		if cashSession.ShiftID != order.ShiftID || cashSession.RestaurantID != order.RestaurantID {
			return fmt.Errorf("%w: кассовая смена оплаты не совпадает с личной сменой заказа", domain.ErrConflict)
		}
		if _, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(requiredRefundPermission(payment.Method))); err != nil {
			return err
		}
		if err := precheck.ApplyRefundedPayment(payment.Amount, now); err != nil {
			return err
		}
		if err := s.repo.UpdatePrecheckPayment(ctx, precheck); err != nil {
			return err
		}
		payment.Status = domain.PaymentRefunded
		payment.UpdatedAt = now
		if err := s.repo.UpdatePaymentStatus(ctx, payment); err != nil {
			return err
		}
		nextAttemptNo, err := s.repo.NextPaymentAttemptNo(ctx, payment.ID)
		if err != nil {
			return err
		}
		attempt := &domain.PaymentAttempt{
			ID:                    s.ids.NewID(),
			PaymentID:             payment.ID,
			AttemptNo:             nextAttemptNo,
			Method:                payment.Method,
			Amount:                payment.Amount,
			Currency:              payment.Currency,
			Status:                domain.PaymentRefunded,
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
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, order.ShiftID, "Payment", payment.ID, "PaymentRefunded", payment); err != nil {
			return err
		}
		check, err := s.repo.GetCheckByOrder(ctx, order.ID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		if check != nil {
			if err := check.ApplyRefund(payment.Amount, now); err != nil {
				return err
			}
			if err := s.repo.UpdateCheckPaidTotal(ctx, check); err != nil {
				return err
			}
			if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, order.ShiftID, "Check", check.ID, "CheckRefunded", check); err != nil {
				return err
			}
		}
		return nil
	})
	return payment, err
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
	switch method {
	case domain.PaymentCash:
		return shared.PermissionPaymentCash
	case domain.PaymentCard:
		return shared.PermissionPaymentCardManual
	default:
		return shared.PermissionPaymentRefund
	}
}

type checkSnapshot struct {
	DocumentType      string                 `json:"document_type"`
	CheckID           string                 `json:"check_id"`
	OrderID           string                 `json:"order_id"`
	PrecheckID        string                 `json:"precheck_id"`
	TableID           string                 `json:"table_id"`
	TableName         string                 `json:"table_name"`
	Subtotal          int64                  `json:"subtotal"`
	DiscountTotal     int64                  `json:"discount_total"`
	TaxTotal          int64                  `json:"tax_total"`
	Total             int64                  `json:"total"`
	PaidTotal         int64                  `json:"paid_total"`
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
		TableID:           order.TableID,
		TableName:         order.TableName,
		Subtotal:          check.Subtotal,
		DiscountTotal:     check.DiscountTotal,
		TaxTotal:          check.TaxTotal,
		Total:             check.Total,
		PaidTotal:         check.PaidTotal,
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

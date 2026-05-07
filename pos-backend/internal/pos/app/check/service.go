package check

import (
	"context"
	"errors"
	"fmt"
	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	txmanager "pos-backend/internal/platform/tx"
	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/ports"
	"strings"
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

func (s *Service) GetCheck(ctx context.Context, id string) (*domain.Check, error) {
	return s.repo.GetCheck(ctx, id)
}

func (s *Service) CapturePayment(ctx context.Context, cmd CapturePaymentCommand) (*domain.Payment, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.PrecheckID) == "" || cmd.Amount <= 0 || strings.TrimSpace(cmd.Currency) == "" {
		return nil, fmt.Errorf("%w: precheck_id, positive amount and currency are required", domain.ErrInvalid)
	}
	if cmd.Method != domain.PaymentCash && cmd.Method != domain.PaymentCard && cmd.Method != domain.PaymentOther {
		return nil, fmt.Errorf("%w: unsupported payment method", domain.ErrInvalid)
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
		if _, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionPaymentCapture)); err != nil {
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
		shift, err := s.repo.GetOpenShiftByDevice(ctx, cmd.DeviceID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return fmt.Errorf("%w: payment requires an active shift", domain.ErrConflict)
			}
			return err
		}
		if shift.ID != order.ShiftID || shift.RestaurantID != order.RestaurantID {
			return fmt.Errorf("%w: payment shift does not match order", domain.ErrConflict)
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
			Currency:              strings.ToUpper(cmd.Currency),
			Status:                domain.PaymentCaptured,
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
			ID:            s.ids.NewID(),
			OrderID:       order.ID,
			Status:        domain.CheckPaid,
			Subtotal:      precheck.Subtotal,
			DiscountTotal: precheck.DiscountTotal,
			TaxTotal:      precheck.TaxTotal,
			Total:         precheck.Total,
			PaidTotal:     precheck.PaidTotal,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
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

func optionalString(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

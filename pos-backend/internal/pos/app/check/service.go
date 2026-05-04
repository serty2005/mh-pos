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

type CreateCheckCommand struct {
	shared.CommandMeta
	OrderID       string `json:"order_id"`
	DiscountTotal int64  `json:"discount_total"`
	TaxTotal      int64  `json:"tax_total"`
}

type CapturePaymentCommand struct {
	shared.CommandMeta
	CheckID  string               `json:"check_id"`
	Method   domain.PaymentMethod `json:"method"`
	Amount   int64                `json:"amount"`
	Currency string               `json:"currency"`
}

func (s *Service) GetCheck(ctx context.Context, id string) (*domain.Check, error) {
	return s.repo.GetCheck(ctx, id)
}

func (s *Service) CreateCheck(ctx context.Context, cmd CreateCheckCommand) (*domain.Check, error) {
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.OrderID) == "" || cmd.DiscountTotal < 0 || cmd.TaxTotal < 0 {
		return nil, fmt.Errorf("%w: order_id, non-negative discount_total and tax_total are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var check *domain.Check
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		order, err := s.repo.GetOrder(ctx, cmd.OrderID)
		if err != nil {
			return err
		}
		if order.Status != domain.OrderOpen {
			return fmt.Errorf("%w: cannot create check for closed order", domain.ErrConflict)
		}
		if _, err := s.repo.GetCheckByOrder(ctx, order.ID); err == nil {
			return fmt.Errorf("%w: order already has a check", domain.ErrConflict)
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		lines, err := s.repo.ListOrderLines(ctx, order.ID)
		if err != nil {
			return err
		}
		var subtotal int64
		for _, line := range lines {
			if line.Status == domain.OrderLineActive {
				subtotal += line.TotalPrice
			}
		}
		total := subtotal - cmd.DiscountTotal + cmd.TaxTotal
		if total < 0 {
			return fmt.Errorf("%w: check total cannot be negative", domain.ErrInvalid)
		}
		check = &domain.Check{ID: s.ids.NewID(), OrderID: order.ID, Status: domain.CheckOpen, Subtotal: subtotal, DiscountTotal: cmd.DiscountTotal, TaxTotal: cmd.TaxTotal, Total: total, PaidTotal: 0, CreatedAt: now, UpdatedAt: now}
		if err := s.repo.CreateCheck(ctx, check); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, "Check", check.ID, "CheckCreated", check)
	})
	return check, err
}

func (s *Service) CapturePayment(ctx context.Context, cmd CapturePaymentCommand) (*domain.Payment, error) {
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.CheckID) == "" || cmd.Amount <= 0 || strings.TrimSpace(cmd.Currency) == "" {
		return nil, fmt.Errorf("%w: check_id, positive amount and currency are required", domain.ErrInvalid)
	}
	if cmd.Method != domain.PaymentCash && cmd.Method != domain.PaymentCard && cmd.Method != domain.PaymentOther {
		return nil, fmt.Errorf("%w: unsupported payment method", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var payment *domain.Payment
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		check, err := s.repo.GetCheck(ctx, cmd.CheckID)
		if err != nil {
			return err
		}
		if check.Status != domain.CheckOpen && check.Status != domain.CheckPaid {
			return fmt.Errorf("%w: check cannot accept payments", domain.ErrConflict)
		}
		if check.PaidTotal+cmd.Amount > check.Total {
			return fmt.Errorf("%w: check overpayment is not allowed", domain.ErrConflict)
		}
		order, err := s.repo.GetOrder(ctx, check.OrderID)
		if err != nil {
			return err
		}
		payment = &domain.Payment{ID: s.ids.NewID(), CheckID: check.ID, Method: cmd.Method, Amount: cmd.Amount, Currency: strings.ToUpper(cmd.Currency), Status: domain.PaymentCaptured, CreatedAt: now, UpdatedAt: now}
		if err := s.repo.CreatePayment(ctx, payment); err != nil {
			return err
		}
		check.PaidTotal += cmd.Amount
		if check.PaidTotal == check.Total {
			check.Status = domain.CheckPaid
		}
		check.UpdatedAt = now
		if err := s.repo.UpdateCheckPaidTotal(ctx, check); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, "Payment", payment.ID, "PaymentCaptured", payment)
	})
	return payment, err
}

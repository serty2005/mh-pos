// Package ticket реализует выпуск и controlled reprint проданных QR-билетных единиц.
// Выпуск выполняется транзакционно после закрытия final check; reprint использует тот же
// ticket number и QR без создания новой единицы. Lookup/use/revoke вне текущего объема.
package ticket

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
	"pos-backend/internal/pos/domain"
	domainticket "pos-backend/internal/pos/domain/ticket"
	"pos-backend/internal/pos/ports"
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

// IssueInput описывает контекст закрытого final check для выпуска билетов.
// Метод выполняется внутри уже открытой транзакции CapturePayment.
type IssueInput struct {
	Meta          shared.CommandMeta
	RestaurantID  string
	DeviceID      string
	CashSessionID string
	ShiftID       string
	CheckID       string
	OrderID       string
	SaleDateLocal string
	Timezone      string
	Now           time.Time
}

// ReprintTicketCommand запрашивает повторную печать ранее выпущенного билета.
type ReprintTicketCommand struct {
	shared.CommandMeta
	TicketID string `json:"ticket_id"`
}

// ticketSnapshot — immutable печатное представление билета. Хранится в ticket_units.snapshot
// и переносится в TicketIssued event; reprint строится строго поверх него.
type ticketSnapshot struct {
	DocumentType      string `json:"document_type"`
	TicketID          string `json:"ticket_id"`
	TicketNumber      string `json:"ticket_number"`
	RestaurantID      string `json:"restaurant_id"`
	DeviceID          string `json:"device_id"`
	CashSessionID     string `json:"cash_session_id"`
	ShiftID           string `json:"shift_id"`
	CheckID           string `json:"check_id"`
	OrderID           string `json:"order_id"`
	OrderLineID       string `json:"order_line_id"`
	CatalogItemID     string `json:"catalog_item_id"`
	MenuItemID        string `json:"menu_item_id"`
	Name              string `json:"name"`
	SaleDateLocal     string `json:"sale_date_local"`
	Timezone          string `json:"timezone"`
	ValidityMode      string `json:"validity_mode"`
	ValidityDateLocal string `json:"validity_date_local,omitempty"`
	CashShiftSequence int64  `json:"cash_shift_sequence"`
	QRPayload         string `json:"qr_payload"`
	IssuedAt          string `json:"issued_at"`
}

// IssueForClosedCheck создает по одной ticket unit на каждую QR-enabled order line (quantity 1).
// Идемпотентность: при наличии билета для line единица не создается повторно (replay-safe).
func (s *Service) IssueForClosedCheck(ctx context.Context, in IssueInput) ([]domain.TicketUnit, error) {
	lines, err := s.repo.ListOrderLines(ctx, in.OrderID)
	if err != nil {
		return nil, err
	}
	var issued []domain.TicketUnit
	for i := range lines {
		line := lines[i]
		if line.Status != domain.OrderLineActive {
			continue
		}
		item, err := s.repo.GetCatalogItem(ctx, line.CatalogItemID)
		if err != nil {
			return nil, err
		}
		if !item.QRConfirmationEnabled {
			continue
		}
		// Replay не создает второй ticket: единица для line уже могла быть выпущена.
		if _, err := s.repo.GetTicketUnitByOrderLine(ctx, line.ID); err == nil {
			continue
		} else if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		validityDate, err := resolveValidityDate(item, in.SaleDateLocal, in.Timezone)
		if err != nil {
			return nil, err
		}
		sequence, err := s.repo.NextTicketCashShiftSequence(ctx, in.CashSessionID)
		if err != nil {
			return nil, err
		}
		id := s.ids.NewID()
		unit := domain.TicketUnit{
			ID:                id,
			TicketNumber:      id,
			RestaurantID:      in.RestaurantID,
			DeviceID:          in.DeviceID,
			CashSessionID:     in.CashSessionID,
			ShiftID:           in.ShiftID,
			CheckID:           in.CheckID,
			OrderID:           in.OrderID,
			OrderLineID:       line.ID,
			CatalogItemID:     line.CatalogItemID,
			MenuItemID:        line.MenuItemID,
			Name:              line.Name,
			SaleDateLocal:     in.SaleDateLocal,
			Timezone:          in.Timezone,
			ValidityMode:      domainticket.ValidityMode(item.ValidityMode),
			ValidityDateLocal: validityDate,
			CashShiftSequence: sequence,
			QRPayload:         domainticket.BuildQRPayload(id),
			PrintStatus:       domainticket.PrintStatusPending,
			CreatedAt:         in.Now,
			UpdatedAt:         in.Now,
		}
		snapshot, err := buildTicketSnapshot(unit)
		if err != nil {
			return nil, err
		}
		unit.Snapshot = snapshot
		if err := s.repo.CreateTicketUnit(ctx, &unit); err != nil {
			return nil, err
		}
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, in.Meta, in.RestaurantID, in.ShiftID, "Ticket", unit.ID, "TicketIssued", json.RawMessage(snapshot)); err != nil {
			return nil, err
		}
		issued = append(issued, unit)
	}
	return issued, nil
}

// ListTicketUnitsByCheckAsOperator возвращает билеты закрытого check внутри restaurant scope оператора.
func (s *Service) ListTicketUnitsByCheckAsOperator(ctx context.Context, checkID string, meta shared.CommandMeta) ([]domain.TicketUnit, error) {
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionCheckView))
	if err != nil {
		return nil, err
	}
	checkID = strings.TrimSpace(checkID)
	if checkID == "" {
		return nil, fmt.Errorf("%w: check_id is required", domain.ErrInvalid)
	}
	tickets, err := s.repo.ListTicketUnitsByCheck(ctx, checkID)
	if err != nil {
		return nil, err
	}
	for _, t := range tickets {
		if t.RestaurantID != operator.Employee.RestaurantID {
			return nil, fmt.Errorf("%w: ticket is outside operator restaurant", domain.ErrForbidden)
		}
	}
	return tickets, nil
}

// ReprintTicket возвращает COPY-документ ранее выпущенного билета без создания новой единицы.
// Используется тот же ticket number и QR; повторная печать помечается COPY marker.
func (s *Service) ReprintTicket(ctx context.Context, cmd ReprintTicketCommand) (*domain.ReprintDocument, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.TicketID) == "" {
		return nil, fmt.Errorf("%w: ticket_id is required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var document *domain.ReprintDocument
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionCheckReprint))
		if err != nil {
			return err
		}
		unit, err := s.repo.GetTicketUnit(ctx, cmd.TicketID)
		if err != nil {
			return err
		}
		if unit.RestaurantID != operator.Employee.RestaurantID {
			return fmt.Errorf("%w: ticket is outside operator restaurant", domain.ErrForbidden)
		}
		if len(unit.Snapshot) == 0 || !json.Valid(unit.Snapshot) {
			return fmt.Errorf("%w: ticket snapshot is not available", domain.ErrConflict)
		}
		// Reprint строится поверх immutable snapshot: тот же ticket number и QR, COPY marker.
		document = domain.NewReprintDocument("ticket", unit.ID, unit.Snapshot, cmd.ActorEmployeeID, now)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return document, nil
}

// resolveValidityDate резолвит immutable дату действия билета по validity mode catalog item.
func resolveValidityDate(item *domain.CatalogItem, saleDateLocal, timezone string) (string, error) {
	mode := domainticket.ValidityMode(strings.TrimSpace(item.ValidityMode))
	if !mode.IsValid() {
		return "", fmt.Errorf("%w: QR service validity mode is not configured", domain.ErrConflict)
	}
	switch mode {
	case domainticket.ValidityCashSession:
		// Срок действия — кассовая смена продажи; конкретная дата в snapshot не фиксируется.
		return "", nil
	case domainticket.ValidityBusinessDate:
		return saleDateLocal, nil
	case domainticket.ValidityAbsoluteDate:
		if item.ValidityExpiresAt == nil {
			return "", fmt.Errorf("%w: absolute validity date is not configured", domain.ErrConflict)
		}
		loc, err := time.LoadLocation(strings.TrimSpace(timezone))
		if err != nil {
			loc = time.UTC
		}
		return item.ValidityExpiresAt.In(loc).Format("2006-01-02"), nil
	default:
		return "", fmt.Errorf("%w: QR service validity mode is not configured", domain.ErrConflict)
	}
}

func buildTicketSnapshot(unit domain.TicketUnit) (json.RawMessage, error) {
	snapshot := ticketSnapshot{
		DocumentType:      "ticket",
		TicketID:          unit.ID,
		TicketNumber:      unit.TicketNumber,
		RestaurantID:      unit.RestaurantID,
		DeviceID:          unit.DeviceID,
		CashSessionID:     unit.CashSessionID,
		ShiftID:           unit.ShiftID,
		CheckID:           unit.CheckID,
		OrderID:           unit.OrderID,
		OrderLineID:       unit.OrderLineID,
		CatalogItemID:     unit.CatalogItemID,
		MenuItemID:        unit.MenuItemID,
		Name:              unit.Name,
		SaleDateLocal:     unit.SaleDateLocal,
		Timezone:          unit.Timezone,
		ValidityMode:      string(unit.ValidityMode),
		ValidityDateLocal: unit.ValidityDateLocal,
		CashShiftSequence: unit.CashShiftSequence,
		QRPayload:         unit.QRPayload,
		IssuedAt:          shared.DBTime(unit.CreatedAt),
	}
	body, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

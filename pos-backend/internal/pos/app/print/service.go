// Package printqueue реализует локальную Edge очередь нефискальной печати.
package printqueue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"mh-pos-platform/receipt/engine"
	"mh-pos-platform/receipt/escpos"
	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	appcheck "pos-backend/internal/pos/app/check"
	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/domain/receipt"
	"pos-backend/internal/pos/ports"
)

const (
	DefaultMaxAttempts        = 3
	defaultPrinterClass       = "generic"
	errPrintRoutingNotFound   = "PRINT_ROUTING_NOT_CONFIGURED"
	errPrintRoutingInvalid    = "PRINT_ROUTING_INVALID"
	errPrintPayloadRenderFail = "PRINT_PAYLOAD_RENDER_FAILED"
)

var retryBackoff = []time.Duration{2 * time.Second, 5 * time.Second, 15 * time.Second}

// Sender отправляет уже отрендеренные ESC/POS bytes в физический или тестовый принтер.
type Sender interface {
	Send(context.Context, escpos.PrinterConfig, []byte) error
}

type rawSender struct{}

func (rawSender) Send(ctx context.Context, cfg escpos.PrinterConfig, payload []byte) error {
	return escpos.WriteRaw(ctx, cfg, payload)
}

// Options задает test hooks и retry defaults для очереди печати.
type Options struct {
	Sender      Sender
	MaxAttempts int
}

// Service управляет print_jobs и worker render/send pipeline.
type Service struct {
	repo        ports.Repository
	ids         idgen.Generator
	clock       clock.Clock
	sender      Sender
	maxAttempts int
}

func NewService(repo ports.Repository, ids idgen.Generator, clock clock.Clock, options Options) *Service {
	sender := options.Sender
	if sender == nil {
		sender = rawSender{}
	}
	maxAttempts := options.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = DefaultMaxAttempts
	}
	return &Service{repo: repo, ids: ids, clock: clock, sender: sender, maxAttempts: maxAttempts}
}

// EnqueueForClosedCheck ставит локальные print jobs после закрытия check.
// Метод не меняет payment/order/check/ticket state и безопасен для replay: repository
// игнорирует уже существующую job по document_type/source_id.
func (s *Service) EnqueueForClosedCheck(ctx context.Context, in appcheck.PrintInput) error {
	now := in.Now
	if now.IsZero() {
		now = s.clock.Now()
	}
	if strings.TrimSpace(in.RestaurantID) == "" || strings.TrimSpace(in.PrecheckID) == "" {
		return fmt.Errorf("%w: restaurant_id and precheck_id are required for print enqueue", domain.ErrInvalid)
	}
	if err := s.enqueue(ctx, now, in.RestaurantID, receipt.DocumentPrecheck, "precheck", in.PrecheckID); err != nil {
		return err
	}
	for _, ticketID := range in.TicketIDs {
		if strings.TrimSpace(ticketID) == "" {
			continue
		}
		if err := s.enqueue(ctx, now, in.RestaurantID, receipt.DocumentTicket, "ticket", ticketID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) enqueue(ctx context.Context, now time.Time, restaurantID string, documentType receipt.DocumentType, sourceKind, sourceID string) error {
	job := &receipt.PrintJob{
		ID:           s.ids.NewID(),
		RestaurantID: strings.TrimSpace(restaurantID),
		DocumentType: documentType,
		SourceKind:   strings.TrimSpace(sourceKind),
		SourceID:     strings.TrimSpace(sourceID),
		Status:       receipt.PrintJobPending,
		MaxAttempts:  s.maxAttempts,
		PrinterClass: defaultPrinterClass,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	return s.repo.EnqueuePrintJob(ctx, job)
}

func (s *Service) GetPrintJobAsOperator(ctx context.Context, id string, meta shared.CommandMeta) (*receipt.PrintJob, error) {
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionPrintStatus))
	if err != nil {
		return nil, err
	}
	job, err := s.repo.GetPrintJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if job.RestaurantID != operator.Employee.RestaurantID {
		return nil, fmt.Errorf("%w: print job is outside operator restaurant", domain.ErrForbidden)
	}
	return job, nil
}

func (s *Service) ListPrintJobsAsOperator(ctx context.Context, meta shared.CommandMeta, query receipt.PrintJobListQuery) ([]receipt.PrintJob, error) {
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionPrintStatus))
	if err != nil {
		return nil, err
	}
	query.RestaurantID = operator.Employee.RestaurantID
	return s.repo.ListPrintJobs(ctx, query)
}

func (s *Service) RetryPrintJobAsOperator(ctx context.Context, id string, meta shared.CommandMeta) (*receipt.PrintJob, error) {
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionPrintRetry))
	if err != nil {
		return nil, err
	}
	job, err := s.repo.GetPrintJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if job.RestaurantID != operator.Employee.RestaurantID {
		return nil, fmt.Errorf("%w: print job is outside operator restaurant", domain.ErrForbidden)
	}
	// Manual retry меняет только локальный print_jobs статус; payment/order/ticket state не трогается.
	reset, err := s.repo.ResetPrintJobForRetry(ctx, id, s.clock.Now())
	if err == nil {
		slog.InfoContext(ctx, "print job manual retry requested",
			"operation", "print.retry",
			"job_id", id,
			"document_type", job.DocumentType,
			"source_kind", job.SourceKind,
			"source_id", job.SourceID,
			"actor_employee_id", operator.Employee.ID,
		)
	}
	return reset, err
}

// ProcessNextPrintJob выполняет один worker step: claim -> render -> send -> status update.
func (s *Service) ProcessNextPrintJob(ctx context.Context, workerID string) (bool, error) {
	now := s.clock.Now()
	job, err := s.repo.ClaimDuePrintJob(ctx, workerID, now)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	if err := s.renderAndSend(ctx, job); err != nil {
		return true, s.recordAttemptFailure(ctx, job, err)
	}
	attempts := job.Attempts + 1
	if err := s.repo.MarkPrintJobSucceeded(ctx, job.ID, attempts, s.clock.Now()); err != nil {
		return true, err
	}
	slog.InfoContext(ctx, "print job succeeded",
		"operation", "print.worker",
		"job_id", job.ID,
		"document_type", job.DocumentType,
		"source_kind", job.SourceKind,
		"source_id", job.SourceID,
		"attempts", attempts,
	)
	return true, nil
}

func (s *Service) recordAttemptFailure(ctx context.Context, job *receipt.PrintJob, cause error) error {
	now := s.clock.Now()
	attempts := job.Attempts + 1
	status := receipt.PrintJobPending
	var next *time.Time
	if attempts >= job.MaxAttempts {
		status = receipt.PrintJobFailed
	} else {
		delay := retryBackoff[min(attempts-1, len(retryBackoff)-1)]
		t := now.Add(delay)
		next = &t
	}
	slog.WarnContext(ctx, "print job attempt failed",
		"operation", "print.worker",
		"job_id", job.ID,
		"document_type", job.DocumentType,
		"source_kind", job.SourceKind,
		"source_id", job.SourceID,
		"attempts", attempts,
		"max_attempts", job.MaxAttempts,
		"next_attempt_at", next,
		"status", status,
		"error", cause,
	)
	return s.repo.MarkPrintJobFailedAttempt(ctx, job.ID, attempts, status, next, cause.Error(), now)
}

func (s *Service) renderAndSend(ctx context.Context, job *receipt.PrintJob) error {
	printers, err := s.repo.ListReceiptPrinters(ctx, job.RestaurantID, job.DocumentType)
	if err != nil {
		return err
	}
	if len(printers) == 0 {
		return errors.New(errPrintRoutingNotFound)
	}
	tmpl, err := s.lookupTemplate(ctx, job.RestaurantID, job.DocumentType)
	if err != nil {
		return err
	}
	printContext, err := s.buildPrintContext(ctx, job, tmpl)
	if err != nil {
		return err
	}
	blocks, err := engine.Render(tmpl.Content, printContext)
	if err != nil {
		return errors.New(errPrintPayloadRenderFail)
	}
	for _, printer := range printers {
		cfg, err := printerConfig(printer, tmpl)
		if err != nil {
			return err
		}
		payload, err := escpos.Render(blocks, cfg.RenderOptions())
		if err != nil {
			return errors.New(errPrintPayloadRenderFail)
		}
		if err := s.sender.Send(ctx, cfg, payload); err != nil {
			return err
		}
	}
	return nil
}

func printerConfig(printer receipt.Printer, tmpl receipt.Template) (escpos.PrinterConfig, error) {
	if err := printer.Validate(); err != nil {
		return escpos.PrinterConfig{}, errors.New(errPrintRoutingInvalid)
	}
	if strings.EqualFold(strings.TrimSpace(printer.Type), escpos.PrinterTypeUSB) && strings.TrimSpace(printer.Address) == "" {
		return escpos.PrinterConfig{}, errors.New(errPrintRoutingInvalid)
	}
	port := 0
	if printer.Port != nil {
		port = *printer.Port
	}
	printerClass := strings.TrimSpace(tmpl.PrinterClass)
	if printerClass == "" {
		printerClass = defaultPrinterClass
	}
	return escpos.PrinterConfig{
		Type:         strings.ToLower(strings.TrimSpace(printer.Type)),
		Address:      strings.TrimSpace(printer.Address),
		Port:         port,
		CPL:          printer.CPL,
		PrinterClass: printerClass,
		Codepage:     strings.TrimSpace(strings.ToLower(printer.Codepage)),
		PaperCutType: strings.TrimSpace(strings.ToLower(printer.PaperCutType)),
	}, nil
}

func (s *Service) lookupTemplate(ctx context.Context, restaurantID string, documentType receipt.DocumentType) (receipt.Template, error) {
	templates, err := s.repo.ListReceiptTemplates(ctx)
	if err != nil {
		return receipt.Template{}, err
	}
	var tenantDefault *receipt.Template
	var anyDefault *receipt.Template
	for i := range templates {
		t := templates[i]
		if t.DocumentType != documentType || !t.IsDefault {
			continue
		}
		if t.RestaurantID == restaurantID {
			return t, nil
		}
		if strings.TrimSpace(t.RestaurantID) == "" && tenantDefault == nil {
			tenantDefault = &t
		}
		if anyDefault == nil {
			anyDefault = &t
		}
	}
	if tenantDefault != nil {
		return *tenantDefault, nil
	}
	if anyDefault != nil {
		return *anyDefault, nil
	}
	return receipt.Template{}, fmt.Errorf("%w: default receipt template is not available for %s", domain.ErrConflict, documentType)
}

func (s *Service) buildPrintContext(ctx context.Context, job *receipt.PrintJob, _ receipt.Template) (any, error) {
	switch job.DocumentType {
	case receipt.DocumentPrecheck:
		return s.buildPrecheckContext(ctx, job.SourceID)
	case receipt.DocumentTicket:
		return s.buildTicketContext(ctx, job.SourceID)
	case receipt.DocumentCheckNonfiscal:
		return s.buildCheckContext(ctx, job.SourceID)
	default:
		return nil, fmt.Errorf("%w: unsupported print job document type %s", domain.ErrInvalid, job.DocumentType)
	}
}

type storedPrecheckSnapshot struct {
	PrecheckID     string `json:"precheck_id"`
	Version        int    `json:"version"`
	CurrencyCode   string `json:"currency_code"`
	Subtotal       int64  `json:"subtotal"`
	DiscountTotal  int64  `json:"discount_total"`
	SurchargeTotal int64  `json:"surcharge_total"`
	TaxTotal       int64  `json:"tax_total"`
	Total          int64  `json:"total"`
	IssuedAt       string `json:"issued_at"`
	Breakdown      struct {
		Lines     []storedLine     `json:"lines"`
		Taxes     []storedTax      `json:"taxes"`
		Discounts []storedDiscount `json:"discounts"`
	} `json:"breakdown"`
}

type storedLine struct {
	Name           string           `json:"name"`
	Quantity       int64            `json:"quantity"`
	UnitPriceMinor int64            `json:"unit_price_minor"`
	TotalMinor     int64            `json:"total_minor"`
	Modifiers      []storedModifier `json:"modifiers"`
}

type storedModifier struct {
	Name       string `json:"name"`
	PriceMinor int64  `json:"unit_price_minor"`
	TotalMinor int64  `json:"total_minor"`
}

type storedTax struct {
	Name             string `json:"name"`
	TaxableBaseMinor int64  `json:"taxable_base_minor"`
	TaxAmountMinor   int64  `json:"tax_amount_minor"`
}

type storedDiscount struct {
	AmountMinor int64   `json:"amount_minor"`
	Reason      *string `json:"reason"`
}

func (s *Service) buildPrecheckContext(ctx context.Context, precheckID string) (engine.PrecheckPrintContext, error) {
	precheck, err := s.repo.GetPrecheck(ctx, precheckID)
	if err != nil {
		return engine.PrecheckPrintContext{}, err
	}
	order, err := s.repo.GetOrder(ctx, precheck.OrderID)
	if err != nil {
		return engine.PrecheckPrintContext{}, err
	}
	return s.projectPrecheck(ctx, precheck.Snapshot, order.RestaurantID, order.ShiftID, precheck.ID, false)
}

type storedCheckSnapshot struct {
	CheckID           string          `json:"check_id"`
	PrecheckID        string          `json:"precheck_id"`
	RestaurantID      string          `json:"restaurant_id"`
	BusinessDateLocal string          `json:"business_date_local"`
	PrecheckSnapshot  json.RawMessage `json:"precheck_snapshot"`
}

func (s *Service) buildCheckContext(ctx context.Context, checkID string) (engine.PrecheckPrintContext, error) {
	check, err := s.repo.GetCheck(ctx, checkID)
	if err != nil {
		return engine.PrecheckPrintContext{}, err
	}
	order, err := s.repo.GetOrder(ctx, check.OrderID)
	if err != nil {
		return engine.PrecheckPrintContext{}, err
	}
	var snapshot storedCheckSnapshot
	if err := json.Unmarshal(check.Snapshot, &snapshot); err != nil {
		return engine.PrecheckPrintContext{}, fmt.Errorf("%w: check snapshot is invalid", domain.ErrConflict)
	}
	return s.projectPrecheck(ctx, snapshot.PrecheckSnapshot, order.RestaurantID, order.ShiftID, check.ID, false)
}

func (s *Service) projectPrecheck(ctx context.Context, raw json.RawMessage, restaurantID, shiftID, displayID string, isCopy bool) (engine.PrecheckPrintContext, error) {
	if len(raw) == 0 || !json.Valid(raw) {
		return engine.PrecheckPrintContext{}, fmt.Errorf("%w: precheck snapshot is not available", domain.ErrConflict)
	}
	var stored storedPrecheckSnapshot
	if err := json.Unmarshal(raw, &stored); err != nil {
		return engine.PrecheckPrintContext{}, fmt.Errorf("%w: precheck snapshot is invalid", domain.ErrConflict)
	}
	restaurant, shift, cashierName, err := s.header(ctx, restaurantID, shiftID)
	if err != nil {
		return engine.PrecheckPrintContext{}, err
	}
	snapshot := engine.PrecheckSnapshot{
		DocumentType:        "precheck",
		PrecheckNumber:      firstNonEmpty(fmt.Sprintf("%d", stored.Version), displayID),
		Restaurant:          engine.RestaurantSnapshot{Name: restaurant.Name},
		Cashier:             engine.CashierSnapshot{Name: cashierName},
		BusinessDate:        shift.BusinessDateLocal,
		OpenedAt:            shared.DBTime(shift.OpenedAt),
		PrintedAt:           shared.DBTime(s.clock.Now()),
		ShiftNumber:         1,
		CurrencyCode:        firstNonEmpty(stored.CurrencyCode, restaurant.Currency),
		SubtotalMinor:       stored.Subtotal,
		DiscountTotalMinor:  stored.DiscountTotal,
		SurchargeTotalMinor: stored.SurchargeTotal,
		TaxTotalMinor:       stored.TaxTotal,
		TotalMinor:          stored.Total,
		IsCopy:              isCopy,
	}
	for _, line := range stored.Breakdown.Lines {
		out := engine.PrecheckSnapshotLine{
			Name:           line.Name,
			Quantity:       line.Quantity,
			UnitPriceMinor: line.UnitPriceMinor,
			TotalMinor:     line.TotalMinor,
		}
		for _, mod := range line.Modifiers {
			out.Modifiers = append(out.Modifiers, engine.PrecheckSnapshotModifier{Name: mod.Name, PriceMinor: firstNonZero(mod.TotalMinor, mod.PriceMinor)})
		}
		snapshot.Lines = append(snapshot.Lines, out)
	}
	for _, tax := range stored.Breakdown.Taxes {
		snapshot.Taxes = append(snapshot.Taxes, engine.TaxSnapshot{Name: tax.Name, BaseMinor: tax.TaxableBaseMinor, AmountMinor: tax.TaxAmountMinor})
	}
	for _, discount := range stored.Breakdown.Discounts {
		name := "Скидка"
		if discount.Reason != nil && strings.TrimSpace(*discount.Reason) != "" {
			name = strings.TrimSpace(*discount.Reason)
		}
		snapshot.Discounts = append(snapshot.Discounts, engine.DiscountSnapshot{Name: name, AmountMinor: discount.AmountMinor})
	}
	return engine.ProjectPrecheck(snapshot), nil
}

func (s *Service) buildTicketContext(ctx context.Context, ticketID string) (engine.ServiceTicketPrintContext, error) {
	unit, err := s.repo.GetTicketUnit(ctx, ticketID)
	if err != nil {
		return engine.ServiceTicketPrintContext{}, err
	}
	restaurant, shift, cashierName, err := s.header(ctx, unit.RestaurantID, unit.ShiftID)
	if err != nil {
		return engine.ServiceTicketPrintContext{}, err
	}
	menuItem, err := s.repo.GetMenuItem(ctx, unit.MenuItemID)
	if err != nil {
		return engine.ServiceTicketPrintContext{}, err
	}
	var validityDate *string
	if strings.TrimSpace(unit.ValidityDateLocal) != "" {
		value := strings.TrimSpace(unit.ValidityDateLocal)
		validityDate = &value
	}
	saleTime := ""
	if !unit.CreatedAt.IsZero() {
		loc, _ := time.LoadLocation(unit.Timezone)
		if loc == nil {
			loc = time.UTC
		}
		saleTime = unit.CreatedAt.In(loc).Format("15:04:05")
	}
	snapshot := engine.TicketSnapshot{
		DocumentType:        "ticket",
		TicketNumber:        unit.TicketNumber,
		TicketDisplayNumber: fmt.Sprintf("%d", unit.CashShiftSequence),
		QRPayload:           unit.QRPayload,
		ServiceName:         unit.Name,
		CategoryName:        menuItem.CategoryID,
		PriceMinor:          menuItem.Price,
		CurrencyCode:        menuItem.Currency,
		SaleDateLocal:       unit.SaleDateLocal,
		SaleTimeLocal:       saleTime,
		Timezone:            unit.Timezone,
		ValidityMode:        string(unit.ValidityMode),
		ValidityDateLocal:   validityDate,
		Restaurant:          engine.RestaurantSnapshot{Name: restaurant.Name},
		Cashier:             engine.CashierSnapshot{Name: cashierName},
		ShiftNumber:         1,
		BusinessDate:        shift.BusinessDateLocal,
	}
	return engine.ProjectTicket(snapshot), nil
}

func (s *Service) header(ctx context.Context, restaurantID, shiftID string) (*domain.Restaurant, *domain.Shift, string, error) {
	restaurant, err := s.repo.GetRestaurant(ctx, restaurantID)
	if err != nil {
		return nil, nil, "", err
	}
	shift, err := s.repo.GetShift(ctx, shiftID)
	if err != nil {
		return nil, nil, "", err
	}
	cashierName := ""
	if employee, err := s.repo.GetEmployee(ctx, shift.OpenedByEmployeeID); err == nil {
		cashierName = employee.Name
	}
	return restaurant, shift, cashierName, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonZero(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

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
	appprecheck "pos-backend/internal/pos/app/precheck"
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

type PrintRouteCommand struct {
	shared.CommandMeta
	DocumentType receipt.DocumentType `json:"document_type"`
	ScopeType    string               `json:"scope_type"`
	ScopeID      string               `json:"scope_id,omitempty"`
	PrinterID    string               `json:"printer_id"`
	IsRequired   *bool                `json:"is_required,omitempty"`
	SortOrder    int                  `json:"sort_order,omitempty"`
}

type UpdatePrintRouteCommand struct {
	shared.CommandMeta
	DocumentType *receipt.DocumentType `json:"document_type,omitempty"`
	ScopeType    *string               `json:"scope_type,omitempty"`
	ScopeID      *string               `json:"scope_id,omitempty"`
	PrinterID    *string               `json:"printer_id,omitempty"`
	IsRequired   *bool                 `json:"is_required,omitempty"`
	SortOrder    *int                  `json:"sort_order,omitempty"`
	IsActive     *bool                 `json:"is_active,omitempty"`
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

func (s *Service) EnqueuePrecheck(ctx context.Context, in appprecheck.PrecheckPrintInput) error {
	now := in.Now
	if now.IsZero() {
		now = s.clock.Now()
	}
	if strings.TrimSpace(in.RestaurantID) == "" || strings.TrimSpace(in.PrecheckID) == "" {
		return fmt.Errorf("%w: restaurant_id and precheck_id are required for print enqueue", domain.ErrInvalid)
	}
	return s.enqueue(ctx, now, in.RestaurantID, receipt.DocumentPrecheck, "precheck", in.PrecheckID, strings.TrimSpace(in.SectionID))
}

// EnqueueForClosedCheck ставит локальные print jobs после закрытия check.
// Метод не меняет payment/order/check/ticket state и безопасен для replay: repository
// игнорирует уже существующую job по document_type/source_id.
func (s *Service) EnqueueForClosedCheck(ctx context.Context, in appcheck.PrintInput) error {
	now := in.Now
	if now.IsZero() {
		now = s.clock.Now()
	}
	if strings.TrimSpace(in.RestaurantID) == "" || strings.TrimSpace(in.CheckID) == "" {
		return fmt.Errorf("%w: restaurant_id and check_id are required for print enqueue", domain.ErrInvalid)
	}
	if err := s.enqueue(ctx, now, in.RestaurantID, receipt.DocumentCheckNonfiscal, "check", in.CheckID, strings.TrimSpace(in.SalesPointID)); err != nil {
		return err
	}
	for _, ticketID := range in.TicketIDs {
		if strings.TrimSpace(ticketID) == "" {
			continue
		}
		if err := s.enqueue(ctx, now, in.RestaurantID, receipt.DocumentTicket, "ticket", ticketID, strings.TrimSpace(in.SectionID)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) enqueue(ctx context.Context, now time.Time, restaurantID string, documentType receipt.DocumentType, sourceKind, sourceID, scopeID string) error {
	scopeType, ok := receipt.RequiredScopeType(documentType)
	if !ok {
		return fmt.Errorf("%w: unsupported print document_type %s", domain.ErrInvalid, documentType)
	}
	var scopeIDPtr *string
	if strings.TrimSpace(scopeID) != "" {
		trimmed := strings.TrimSpace(scopeID)
		scopeIDPtr = &trimmed
	}
	job := &receipt.PrintJob{
		ID:           s.ids.NewID(),
		RestaurantID: strings.TrimSpace(restaurantID),
		DocumentType: documentType,
		ScopeID:      scopeIDPtr,
		SourceKind:   strings.TrimSpace(sourceKind),
		SourceID:     strings.TrimSpace(sourceID),
		Status:       receipt.PrintJobPending,
		MaxAttempts:  s.maxAttempts,
		PrinterClass: defaultPrinterClass,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	routes, err := s.repo.ListActivePrintRoutes(ctx, job.RestaurantID, documentType, scopeType, scopeIDPtr)
	if err != nil {
		return err
	}
	if len(routes) == 0 {
		errCode := errPrintRoutingNotFound
		job.Status = receipt.PrintJobFailed
		job.LastError = &errCode
		return s.repo.EnqueuePrintJobWithTargets(ctx, job, nil)
	}
	targets := s.targetsForRoutes(job, routes, now)
	return s.repo.EnqueuePrintJobWithTargets(ctx, job, targets)
}

func (s *Service) routesForJob(ctx context.Context, job *receipt.PrintJob) ([]receipt.PrintRoute, error) {
	scopeType, ok := receipt.RequiredScopeType(job.DocumentType)
	if !ok {
		return nil, fmt.Errorf("%w: unsupported print document_type %s", domain.ErrInvalid, job.DocumentType)
	}
	return s.repo.ListActivePrintRoutes(ctx, job.RestaurantID, job.DocumentType, scopeType, job.ScopeID)
}

func (s *Service) targetsForRoutes(job *receipt.PrintJob, routes []receipt.PrintRoute, now time.Time) []receipt.PrintJobTarget {
	targets := make([]receipt.PrintJobTarget, 0, len(routes))
	for _, route := range routes {
		targets = append(targets, receipt.PrintJobTarget{
			ID:           s.ids.NewID(),
			PrintJobID:   job.ID,
			RestaurantID: job.RestaurantID,
			PrinterID:    route.PrinterID,
			ScopeType:    route.ScopeType,
			ScopeID:      route.ScopeID,
			Status:       receipt.PrintJobPending,
			MaxAttempts:  s.maxAttempts,
			IsRequired:   route.IsRequired,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}
	return targets
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
	targets, err := s.repo.ListPrintJobTargets(ctx, job.ID)
	if err != nil {
		return nil, err
	}
	job.Targets = targets
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

func (s *Service) ListRoutingPrintersAsOperator(ctx context.Context, meta shared.CommandMeta) ([]receipt.Printer, error) {
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionPrintRoutingView))
	if err != nil {
		return nil, err
	}
	return s.repo.ListAllReceiptPrinters(ctx, operator.Employee.RestaurantID)
}

func (s *Service) ListRoutingSalesPointsAsOperator(ctx context.Context, meta shared.CommandMeta) ([]domain.SalesPoint, error) {
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionPrintRoutingView))
	if err != nil {
		return nil, err
	}
	return s.repo.ListSalesPoints(ctx, operator.Employee.RestaurantID)
}

func (s *Service) ListRoutingSectionsAsOperator(ctx context.Context, meta shared.CommandMeta) ([]domain.RestaurantSection, error) {
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionPrintRoutingView))
	if err != nil {
		return nil, err
	}
	return s.repo.ListRestaurantSections(ctx, operator.Employee.RestaurantID)
}

func (s *Service) ListPrintRoutesAsOperator(ctx context.Context, meta shared.CommandMeta) ([]receipt.PrintRoute, error) {
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionPrintRoutingView))
	if err != nil {
		return nil, err
	}
	return s.repo.ListPrintRoutes(ctx, operator.Employee.RestaurantID)
}

func (s *Service) CreatePrintRouteAsOperator(ctx context.Context, cmd PrintRouteCommand) (*receipt.PrintRoute, error) {
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionPrintRoutingManage))
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	required := true
	if cmd.IsRequired != nil {
		required = *cmd.IsRequired
	}
	route := receipt.PrintRoute{
		ID:           s.ids.NewID(),
		RestaurantID: operator.Employee.RestaurantID,
		DocumentType: cmd.DocumentType,
		ScopeType:    strings.TrimSpace(cmd.ScopeType),
		PrinterID:    strings.TrimSpace(cmd.PrinterID),
		IsRequired:   required,
		SortOrder:    cmd.SortOrder,
		Origin:       "edge_override",
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if scopeID := strings.TrimSpace(cmd.ScopeID); scopeID != "" {
		route.ScopeID = &scopeID
	}
	if route.ScopeType == receipt.ScopeRestaurant {
		route.ScopeID = nil
	}
	if err := s.validatePrintRoute(ctx, route); err != nil {
		return nil, err
	}
	return s.repo.CreatePrintRoute(ctx, route, s.ids.NewID(), operator.Employee.ID, now)
}

func (s *Service) UpdatePrintRouteAsOperator(ctx context.Context, id string, cmd UpdatePrintRouteCommand) (*receipt.PrintRoute, error) {
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionPrintRoutingManage))
	if err != nil {
		return nil, err
	}
	route, err := s.printRouteForOperator(ctx, operator.Employee.RestaurantID, id)
	if err != nil {
		return nil, err
	}
	if cmd.DocumentType != nil {
		route.DocumentType = *cmd.DocumentType
	}
	if cmd.ScopeType != nil {
		route.ScopeType = strings.TrimSpace(*cmd.ScopeType)
	}
	if cmd.ScopeID != nil {
		scopeID := strings.TrimSpace(*cmd.ScopeID)
		if scopeID == "" {
			route.ScopeID = nil
		} else {
			route.ScopeID = &scopeID
		}
	}
	if cmd.PrinterID != nil {
		route.PrinterID = strings.TrimSpace(*cmd.PrinterID)
	}
	if cmd.IsRequired != nil {
		route.IsRequired = *cmd.IsRequired
	}
	if cmd.SortOrder != nil {
		route.SortOrder = *cmd.SortOrder
	}
	if cmd.IsActive != nil {
		route.IsActive = *cmd.IsActive
	}
	if route.ScopeType == receipt.ScopeRestaurant {
		route.ScopeID = nil
	}
	route.Origin = "edge_override"
	route.UpdatedAt = s.clock.Now()
	if err := s.validatePrintRoute(ctx, route); err != nil {
		return nil, err
	}
	return s.repo.UpdatePrintRoute(ctx, route, s.ids.NewID(), operator.Employee.ID, route.UpdatedAt)
}

func (s *Service) DeactivatePrintRouteAsOperator(ctx context.Context, id string, meta shared.CommandMeta) (*receipt.PrintRoute, error) {
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionPrintRoutingManage))
	if err != nil {
		return nil, err
	}
	if _, err := s.printRouteForOperator(ctx, operator.Employee.RestaurantID, id); err != nil {
		return nil, err
	}
	return s.repo.DeactivatePrintRoute(ctx, strings.TrimSpace(id), s.ids.NewID(), operator.Employee.ID, s.clock.Now())
}

func (s *Service) RetryPrintJobTargetAsOperator(ctx context.Context, jobID, targetID string, meta shared.CommandMeta) (*receipt.PrintJob, error) {
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionPrintRetry))
	if err != nil {
		return nil, err
	}
	job, err := s.repo.GetPrintJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job.RestaurantID != operator.Employee.RestaurantID {
		return nil, fmt.Errorf("%w: print job is outside operator restaurant", domain.ErrForbidden)
	}
	reset, err := s.repo.RetryPrintJobTarget(ctx, job.ID, targetID, s.clock.Now())
	if err != nil {
		return nil, err
	}
	reset.Targets, err = s.repo.ListPrintJobTargets(ctx, reset.ID)
	if err != nil {
		return nil, err
	}
	return reset, nil
}

func (s *Service) printRouteForOperator(ctx context.Context, restaurantID, id string) (receipt.PrintRoute, error) {
	routes, err := s.repo.ListPrintRoutes(ctx, restaurantID)
	if err != nil {
		return receipt.PrintRoute{}, err
	}
	for _, route := range routes {
		if route.ID == strings.TrimSpace(id) {
			return route, nil
		}
	}
	return receipt.PrintRoute{}, domain.ErrNotFound
}

func (s *Service) validatePrintRoute(ctx context.Context, route receipt.PrintRoute) error {
	requiredScope, ok := receipt.RequiredScopeType(route.DocumentType)
	if !ok {
		return fmt.Errorf("%w: unsupported print route document_type %s", domain.ErrInvalid, route.DocumentType)
	}
	if route.ScopeType != requiredScope {
		return fmt.Errorf("%w: document_type %s requires scope_type %s", domain.ErrInvalid, route.DocumentType, requiredScope)
	}
	if strings.TrimSpace(route.PrinterID) == "" {
		return fmt.Errorf("%w: printer_id is required", domain.ErrInvalid)
	}
	printer, err := s.repo.GetReceiptPrinter(ctx, route.PrinterID)
	if err != nil {
		return err
	}
	if printer.RestaurantID != route.RestaurantID || !printer.IsActive {
		return fmt.Errorf("%w: printer is inactive or belongs to another restaurant", domain.ErrInvalid)
	}
	switch route.ScopeType {
	case receipt.ScopeRestaurant:
		route.ScopeID = nil
	case receipt.ScopeSalesPoint:
		if route.ScopeID == nil || strings.TrimSpace(*route.ScopeID) == "" {
			return fmt.Errorf("%w: sales_point scope_id is required", domain.ErrInvalid)
		}
		salesPoint, err := s.repo.GetSalesPoint(ctx, *route.ScopeID)
		if err != nil {
			return err
		}
		if salesPoint.RestaurantID != route.RestaurantID || !salesPoint.IsActive {
			return fmt.Errorf("%w: sales point is inactive or belongs to another restaurant", domain.ErrInvalid)
		}
	case receipt.ScopeSection:
		if route.ScopeID == nil || strings.TrimSpace(*route.ScopeID) == "" {
			return fmt.Errorf("%w: section scope_id is required", domain.ErrInvalid)
		}
		section, err := s.repo.GetRestaurantSection(ctx, *route.ScopeID)
		if err != nil {
			return err
		}
		if section.RestaurantID != route.RestaurantID || !section.IsActive {
			return fmt.Errorf("%w: section is inactive or belongs to another restaurant", domain.ErrInvalid)
		}
		if mode, ok := receipt.RequiredSectionMode(route.DocumentType); ok && string(section.Mode) != mode {
			return fmt.Errorf("%w: document_type %s requires section mode %s", domain.ErrInvalid, route.DocumentType, mode)
		}
	default:
		return fmt.Errorf("%w: unsupported print route scope_type %s", domain.ErrInvalid, route.ScopeType)
	}
	return nil
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
	now := s.clock.Now()
	routes, err := s.routesForJob(ctx, job)
	if err != nil {
		return nil, err
	}
	targets := s.targetsForRoutes(job, routes, now)
	// Manual retry пересобирает только локальные print targets; payment/order/ticket state не трогается.
	reset, err := s.repo.ResetPrintJobForRetryWithTargets(ctx, id, targets, now)
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
	job, target, err := s.repo.ClaimDuePrintJobTarget(ctx, workerID, now)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	if err := s.renderAndSend(ctx, job, target); err != nil {
		return true, s.recordTargetAttemptFailure(ctx, job, target, err)
	}
	attempts := target.Attempts + 1
	if err := s.repo.MarkPrintJobTargetSucceeded(ctx, target.ID, attempts, s.clock.Now()); err != nil {
		return true, err
	}
	if err := s.maybeConfirmCheckPrint(ctx, job.ID); err != nil {
		return true, err
	}
	slog.InfoContext(ctx, "print job target succeeded",
		"operation", "print.worker",
		"job_id", job.ID,
		"target_id", target.ID,
		"printer_id", target.PrinterID,
		"document_type", job.DocumentType,
		"source_kind", job.SourceKind,
		"source_id", job.SourceID,
		"attempts", attempts,
	)
	return true, nil
}

func (s *Service) maybeConfirmCheckPrint(ctx context.Context, jobID string) error {
	job, err := s.repo.GetPrintJob(ctx, jobID)
	if err != nil {
		return err
	}
	if job.Status != receipt.PrintJobSucceeded {
		return nil
	}
	var checkID string
	switch job.DocumentType {
	case receipt.DocumentCheckNonfiscal:
		checkID = job.SourceID
	case receipt.DocumentTicket:
		ticket, err := s.repo.GetTicketUnit(ctx, job.SourceID)
		if err != nil {
			return err
		}
		checkID = ticket.CheckID
	default:
		return nil
	}
	_, err = s.repo.MarkCheckPrintConfirmedIfReady(ctx, checkID, s.clock.Now())
	return err
}

func (s *Service) recordTargetAttemptFailure(ctx context.Context, job *receipt.PrintJob, target *receipt.PrintJobTarget, cause error) error {
	now := s.clock.Now()
	attempts := target.Attempts + 1
	status := receipt.PrintJobPending
	var next *time.Time
	if attempts >= target.MaxAttempts {
		status = receipt.PrintJobFailed
	} else {
		delay := retryBackoff[min(attempts-1, len(retryBackoff)-1)]
		t := now.Add(delay)
		next = &t
	}
	slog.WarnContext(ctx, "print job target attempt failed",
		"operation", "print.worker",
		"job_id", job.ID,
		"target_id", target.ID,
		"printer_id", target.PrinterID,
		"document_type", job.DocumentType,
		"source_kind", job.SourceKind,
		"source_id", job.SourceID,
		"attempts", attempts,
		"max_attempts", target.MaxAttempts,
		"next_attempt_at", next,
		"status", status,
		"error", cause,
	)
	return s.repo.MarkPrintJobTargetFailedAttempt(ctx, target.ID, attempts, status, next, cause.Error(), now)
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

func (s *Service) renderAndSend(ctx context.Context, job *receipt.PrintJob, target *receipt.PrintJobTarget) error {
	printer, err := s.repo.GetReceiptPrinter(ctx, target.PrinterID)
	if err != nil {
		return err
	}
	if !printer.IsActive || printer.RestaurantID != job.RestaurantID {
		return errors.New(errPrintRoutingInvalid)
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
	cfg, err := printerConfig(*printer, tmpl)
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

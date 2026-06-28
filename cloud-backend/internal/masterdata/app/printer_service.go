package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud-backend/internal/masterdata/domain"
)

// PrinterFilter описывает фильтры списка Cloud принтеров.
type PrinterFilter struct {
	OrgID        string
	RestaurantID string
	IsActive     *bool
}

// CreatePrinterCommand описывает создание Cloud-owned ESC/POS принтера.
type CreatePrinterCommand struct {
	OrgID         string   `json:"org_id,omitempty"`
	RestaurantID  string   `json:"restaurant_id"`
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	Address       string   `json:"address,omitempty"`
	Port          *int     `json:"port,omitempty"`
	DocumentTypes []string `json:"document_types"`
	Codepage      string   `json:"codepage,omitempty"`
	PaperCutType  string   `json:"paper_cut_type,omitempty"`
	CPL           int      `json:"cpl"`
}

// UpdatePrinterCommand описывает изменение принтера; version инкрементируется.
type UpdatePrinterCommand struct {
	Name          string   `json:"name,omitempty"`
	Type          string   `json:"type,omitempty"`
	Address       *string  `json:"address,omitempty"`
	Port          *int     `json:"port,omitempty"`
	DocumentTypes []string `json:"document_types,omitempty"`
	Codepage      *string  `json:"codepage,omitempty"`
	PaperCutType  *string  `json:"paper_cut_type,omitempty"`
	CPL           *int     `json:"cpl,omitempty"`
}

func parsePrinterDocumentTypes(raw []string) ([]domain.PrinterDocumentType, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: document_types must not be empty", domain.ErrInvalid)
	}
	out := make([]domain.PrinterDocumentType, 0, len(raw))
	seen := map[domain.PrinterDocumentType]struct{}{}
	for _, s := range raw {
		dt := domain.PrinterDocumentType(strings.TrimSpace(s))
		if err := domain.ValidatePrinterDocumentType(dt); err != nil {
			return nil, err
		}
		if _, dup := seen[dt]; dup {
			return nil, fmt.Errorf("%w: duplicate document_type %q", domain.ErrInvalid, dt)
		}
		seen[dt] = struct{}{}
		out = append(out, dt)
	}
	return out, nil
}

// CreatePrinter создает Cloud-authored принтер и публикует delivery refresh для ресторана.
func (s *Service) CreatePrinter(ctx context.Context, cmd CreatePrinterCommand) (domain.Printer, error) {
	restaurantID := strings.TrimSpace(cmd.RestaurantID)
	if restaurantID == "" {
		return domain.Printer{}, fmt.Errorf("%w: restaurant_id is required", domain.ErrInvalid)
	}
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return domain.Printer{}, fmt.Errorf("%w: name is required", domain.ErrInvalid)
	}
	printerType := domain.PrinterType(strings.TrimSpace(cmd.Type))
	if err := domain.ValidatePrinterType(printerType); err != nil {
		return domain.Printer{}, err
	}
	docTypes, err := parsePrinterDocumentTypes(cmd.DocumentTypes)
	if err != nil {
		return domain.Printer{}, err
	}
	codepage := domain.PrinterCodepage(strings.TrimSpace(cmd.Codepage))
	if err := domain.ValidatePrinterCodepage(codepage); err != nil {
		return domain.Printer{}, err
	}
	paperCut := domain.PaperCutPartial
	if strings.TrimSpace(cmd.PaperCutType) != "" {
		paperCut = domain.PaperCutType(strings.TrimSpace(cmd.PaperCutType))
	}
	if err := domain.ValidatePaperCutType(paperCut); err != nil {
		return domain.Printer{}, err
	}
	if err := domain.ValidatePrinterCPL(cmd.CPL); err != nil {
		return domain.Printer{}, err
	}
	// USB принтеры не используют сетевой адрес и порт.
	address := strings.TrimSpace(cmd.Address)
	port := cmd.Port
	if printerType == domain.PrinterTypeUSB {
		address = ""
		port = nil
	}
	now := s.clock.Now().UTC()
	printer := domain.Printer{
		ID:            s.ids.NewID(),
		OrgID:         strings.TrimSpace(cmd.OrgID),
		RestaurantID:  restaurantID,
		Name:          name,
		Type:          printerType,
		Address:       address,
		Port:          port,
		DocumentTypes: docTypes,
		Codepage:      codepage,
		PaperCutType:  paperCut,
		CPL:           cmd.CPL,
		IsActive:      true,
		Version:       1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	stored, err := s.repo.CreatePrinter(ctx, printer)
	return s.afterPrinterCommit(ctx, stored, err)
}

// UpdatePrinter изменяет принтер, инкрементирует version и публикует delivery refresh.
func (s *Service) UpdatePrinter(ctx context.Context, id string, cmd UpdatePrinterCommand) (domain.Printer, error) {
	printer, err := s.repo.GetPrinter(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Printer{}, err
	}
	if strings.TrimSpace(cmd.Name) != "" {
		printer.Name = strings.TrimSpace(cmd.Name)
	}
	if strings.TrimSpace(cmd.Type) != "" {
		pt := domain.PrinterType(strings.TrimSpace(cmd.Type))
		if err := domain.ValidatePrinterType(pt); err != nil {
			return domain.Printer{}, err
		}
		printer.Type = pt
	}
	if cmd.Address != nil {
		printer.Address = strings.TrimSpace(*cmd.Address)
	}
	if cmd.Port != nil {
		printer.Port = cmd.Port
	}
	if len(cmd.DocumentTypes) > 0 {
		docTypes, err := parsePrinterDocumentTypes(cmd.DocumentTypes)
		if err != nil {
			return domain.Printer{}, err
		}
		printer.DocumentTypes = docTypes
	}
	if cmd.Codepage != nil {
		cp := domain.PrinterCodepage(strings.TrimSpace(*cmd.Codepage))
		if err := domain.ValidatePrinterCodepage(cp); err != nil {
			return domain.Printer{}, err
		}
		printer.Codepage = cp
	}
	if cmd.PaperCutType != nil {
		pct := domain.PaperCutType(strings.TrimSpace(*cmd.PaperCutType))
		if err := domain.ValidatePaperCutType(pct); err != nil {
			return domain.Printer{}, err
		}
		printer.PaperCutType = pct
	}
	if cmd.CPL != nil {
		if err := domain.ValidatePrinterCPL(*cmd.CPL); err != nil {
			return domain.Printer{}, err
		}
		printer.CPL = *cmd.CPL
	}
	// USB принтеры не хранят сетевые параметры.
	if printer.Type == domain.PrinterTypeUSB {
		printer.Address = ""
		printer.Port = nil
	}
	printer.Version++
	printer.UpdatedAt = s.clock.Now().UTC()
	stored, err := s.repo.UpdatePrinter(ctx, printer)
	return s.afterPrinterCommit(ctx, stored, err)
}

// GetPrinter возвращает один Cloud-owned принтер.
func (s *Service) GetPrinter(ctx context.Context, id string) (domain.Printer, error) {
	return s.repo.GetPrinter(ctx, strings.TrimSpace(id))
}

// ListPrinters возвращает список принтеров с фильтром по restaurant_id/is_active.
func (s *Service) ListPrinters(ctx context.Context, filter PrinterFilter) ([]domain.Printer, error) {
	filter.OrgID = strings.TrimSpace(filter.OrgID)
	filter.RestaurantID = strings.TrimSpace(filter.RestaurantID)
	return s.repo.ListPrinters(ctx, filter)
}

// DeactivatePrinter выполняет soft-delete (is_active=FALSE, version++) и публикует delivery refresh.
func (s *Service) DeactivatePrinter(ctx context.Context, id string) (domain.Printer, error) {
	printer, err := s.repo.GetPrinter(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Printer{}, err
	}
	if !printer.IsActive {
		return printer, nil
	}
	printer.IsActive = false
	printer.Version++
	printer.UpdatedAt = s.clock.Now().UTC()
	stored, err := s.repo.UpdatePrinter(ctx, printer)
	return s.afterPrinterCommit(ctx, stored, err)
}

// afterPrinterCommit публикует delivery package для ресторана принтера.
func (s *Service) afterPrinterCommit(ctx context.Context, printer domain.Printer, err error) (domain.Printer, error) {
	if err != nil {
		return printer, err
	}
	return afterRestaurantCommit(s, ctx, printer.RestaurantID, printer, nil)
}

// edgePrinter описывает строку принтера в Cloud -> Edge stream printers.
type edgePrinter struct {
	ID            string   `json:"id"`
	RestaurantID  string   `json:"restaurant_id"`
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	Address       string   `json:"address,omitempty"`
	Port          *int     `json:"port,omitempty"`
	DocumentTypes []string `json:"document_types"`
	Codepage      string   `json:"codepage"`
	PaperCutType  string   `json:"paper_cut_type"`
	CPL           int      `json:"cpl"`
	Version       int      `json:"version"`
}

type printersPayload struct {
	NodeDeviceID   string        `json:"node_device_id,omitempty"`
	RestaurantID   string        `json:"restaurant_id"`
	SyncMode       string        `json:"sync_mode"`
	CheckpointToken string       `json:"checkpoint_token,omitempty"`
	CloudVersion   int64         `json:"cloud_version"`
	CloudUpdatedAt time.Time     `json:"cloud_updated_at"`
	Printers       []edgePrinter `json:"printers"`
}

// printersStream собирает Cloud -> Edge package принтеров для ресторана.
// Checkpoint фиксирует MAX(updated_at) и количество активных принтеров.
func (s *Service) printersStream(ctx context.Context, restaurantID, nodeDeviceID, syncMode string, cloudVersion int64, updatedAt time.Time) (StreamPackage, int, error) {
	rows, err := s.repo.ListActivePrintersForRestaurant(ctx, restaurantID)
	if err != nil {
		return StreamPackage{}, 0, err
	}
	maxUpdated := time.Unix(0, 0).UTC()
	printers := make([]edgePrinter, 0, len(rows))
	for _, row := range rows {
		if row.UpdatedAt.After(maxUpdated) {
			maxUpdated = row.UpdatedAt.UTC()
		}
		docTypes := make([]string, 0, len(row.DocumentTypes))
		for _, dt := range row.DocumentTypes {
			docTypes = append(docTypes, string(dt))
		}
		printers = append(printers, edgePrinter{
			ID:            row.ID,
			RestaurantID:  row.RestaurantID,
			Name:          row.Name,
			Type:          string(row.Type),
			Address:       row.Address,
			Port:          row.Port,
			DocumentTypes: docTypes,
			Codepage:      string(row.Codepage),
			PaperCutType:  string(row.PaperCutType),
			CPL:           row.CPL,
			Version:       row.Version,
		})
	}
	checkpoint := fmt.Sprintf("printers:%s:%d:%d", restaurantID, maxUpdated.UnixMilli(), len(printers))
	body, err := json.Marshal(printersPayload{
		NodeDeviceID:    nodeDeviceID,
		RestaurantID:    restaurantID,
		SyncMode:        syncMode,
		CheckpointToken: checkpoint,
		CloudVersion:    cloudVersion,
		CloudUpdatedAt:  updatedAt,
		Printers:        printers,
	})
	if err != nil {
		return StreamPackage{}, 0, err
	}
	return StreamPackage{
		StreamName:      "printers",
		NodeDeviceID:    nodeDeviceID,
		RestaurantID:    restaurantID,
		SyncMode:        syncMode,
		CloudVersion:    cloudVersion,
		CheckpointToken: checkpoint,
		CloudUpdatedAt:  updatedAt,
		PayloadJSON:     body,
	}, len(printers), nil
}

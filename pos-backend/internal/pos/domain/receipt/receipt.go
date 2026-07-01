// Package receipt описывает Cloud-owned read model шаблонов печати на Edge.
package receipt

import (
	"fmt"
	"strings"

	"pos-backend/internal/pos/domain/shared"
)

// DocumentType задает поддерживаемые нефискальные типы документов ReceiptLine.
type DocumentType string

const (
	DocumentPrecheck       DocumentType = "precheck"
	DocumentCheckNonfiscal DocumentType = "check_nonfiscal"
	DocumentTicket         DocumentType = "ticket"
	DocumentKitchenService DocumentType = "kitchen_service"
	DocumentReport         DocumentType = "report"
	DocumentCashInOut      DocumentType = "cash_in_out"
	DocumentAcceptance     DocumentType = "acceptance"
)

// Template — read-only проекция Cloud receipt_template на Edge.
// Edge хранит только эффективный (is_active) набор; restaurant_id "" = tenant-level default.
type Template struct {
	ID           string       `json:"id"`
	RestaurantID string       `json:"restaurant_id,omitempty"`
	DocumentType DocumentType `json:"document_type"`
	Name         string       `json:"name"`
	Content      string       `json:"content"`
	Level        int          `json:"level"`
	CPL          int          `json:"cpl"`
	PrinterClass string       `json:"printer_class"`
	IsDefault    bool         `json:"is_default"`
	Version      int          `json:"version"`
}

// Printer — Cloud-owned read-only конфигурация ESC/POS принтера на Edge.
// DocumentTypes может содержать несколько нефискальных типов документов; один тип
// документа может маршрутизироваться на несколько активных принтеров ресторана.
type Printer struct {
	ID            string         `json:"id"`
	RestaurantID  string         `json:"restaurant_id"`
	Name          string         `json:"name"`
	Type          string         `json:"type"`
	Address       string         `json:"address,omitempty"`
	Port          *int           `json:"port,omitempty"`
	DocumentTypes []DocumentType `json:"document_types"`
	Codepage      string         `json:"codepage"`
	PaperCutType  string         `json:"paper_cut_type"`
	CPL           int            `json:"cpl"`
	IsActive      bool           `json:"is_active"`
	CloudVersion  int            `json:"cloud_version"`
	Version       int            `json:"version,omitempty"`
}

// Validate проверяет нефискальный document_type, CPL и непустые поля шаблона.
func (t Template) Validate() error {
	if strings.TrimSpace(t.ID) == "" {
		return fmt.Errorf("%w: receipt template id is required", shared.ErrInvalid)
	}
	switch t.DocumentType {
	case DocumentPrecheck, DocumentCheckNonfiscal, DocumentTicket,
		DocumentKitchenService, DocumentCashInOut, DocumentAcceptance:
	default:
		return fmt.Errorf("%w: unsupported receipt template document_type %q", shared.ErrInvalid, t.DocumentType)
	}
	if strings.TrimSpace(t.Name) == "" || strings.TrimSpace(t.Content) == "" {
		return fmt.Errorf("%w: receipt template name and content are required", shared.ErrInvalid)
	}
	switch t.CPL {
	case 32, 40, 48, 58:
	default:
		return fmt.Errorf("%w: receipt template cpl must be one of 32, 40, 48, 58", shared.ErrInvalid)
	}
	if t.Level != 1 && t.Level != 2 {
		return fmt.Errorf("%w: receipt template level must be 1 or 2", shared.ErrInvalid)
	}
	return nil
}

// Validate проверяет Cloud-owned конфигурацию принтера перед записью в Edge read model.
func (p Printer) Validate() error {
	if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.RestaurantID) == "" || strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("%w: receipt printer id, restaurant_id and name are required", shared.ErrInvalid)
	}
	switch strings.TrimSpace(strings.ToLower(p.Type)) {
	case "tcp":
		if strings.TrimSpace(p.Address) == "" {
			return fmt.Errorf("%w: tcp receipt printer address is required", shared.ErrInvalid)
		}
		if p.Port == nil || *p.Port <= 0 {
			return fmt.Errorf("%w: tcp receipt printer port is required", shared.ErrInvalid)
		}
	case "usb":
	default:
		return fmt.Errorf("%w: receipt printer type must be tcp or usb", shared.ErrInvalid)
	}
	if len(p.DocumentTypes) == 0 {
		return fmt.Errorf("%w: receipt printer document_types are required", shared.ErrInvalid)
	}
	for _, documentType := range p.DocumentTypes {
		switch documentType {
		case DocumentPrecheck, DocumentCheckNonfiscal, DocumentTicket,
			DocumentKitchenService, DocumentCashInOut, DocumentAcceptance:
		default:
			return fmt.Errorf("%w: unsupported receipt printer document_type %q", shared.ErrInvalid, documentType)
		}
	}
	switch strings.TrimSpace(strings.ToLower(p.Codepage)) {
	case "", "cp437", "cp866":
	default:
		return fmt.Errorf("%w: receipt printer codepage must be empty, cp437 or cp866", shared.ErrInvalid)
	}
	switch strings.TrimSpace(strings.ToLower(p.PaperCutType)) {
	case "", "partial", "full":
	default:
		return fmt.Errorf("%w: receipt printer paper_cut_type must be partial or full", shared.ErrInvalid)
	}
	switch p.CPL {
	case 32, 42, 48, 56, 80:
	default:
		return fmt.Errorf("%w: receipt printer cpl must be one of 32, 42, 48, 56, 80", shared.ErrInvalid)
	}
	if p.CloudVersion < 0 {
		return fmt.Errorf("%w: receipt printer cloud_version must be non-negative", shared.ErrInvalid)
	}
	return nil
}

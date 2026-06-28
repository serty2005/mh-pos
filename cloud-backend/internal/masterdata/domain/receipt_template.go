package domain

import (
	"fmt"
	"strings"
	"time"
)

// PermissionReceiptTemplatesManage задает RBAC permission id для Cloud receipt template CRUD.
const PermissionReceiptTemplatesManage = "cloud.templates.manage"

// ReceiptTemplateDocumentType задает поддерживаемые типы документов ReceiptLine (нефискальные).
type ReceiptTemplateDocumentType string

const (
	ReceiptDocumentPrecheck       ReceiptTemplateDocumentType = "precheck"
	ReceiptDocumentCheckNonfiscal ReceiptTemplateDocumentType = "check_nonfiscal"
	ReceiptDocumentTicket         ReceiptTemplateDocumentType = "ticket"
	ReceiptDocumentKitchenService ReceiptTemplateDocumentType = "kitchen_service"
	ReceiptDocumentCashInOut      ReceiptTemplateDocumentType = "cash_in_out"
	ReceiptDocumentAcceptance     ReceiptTemplateDocumentType = "acceptance"
)

// ReceiptTemplate описывает Cloud-owned шаблон печати ReceiptLine.
// restaurant_id == "" (NULL в БД) обозначает tenant-level default.
type ReceiptTemplate struct {
	ID           string                      `json:"id"`
	OrgID        string                      `json:"org_id"`
	RestaurantID string                      `json:"restaurant_id"`
	DocumentType ReceiptTemplateDocumentType `json:"document_type"`
	Name         string                      `json:"name"`
	Description  string                      `json:"description"`
	Content      string                      `json:"content"`
	Level        int                         `json:"level"`
	CPL          int                         `json:"cpl"`
	PrinterClass string                      `json:"printer_class"`
	IsDefault    bool                        `json:"is_default"`
	Version      int                         `json:"version"`
	IsActive     bool                        `json:"is_active"`
	CreatedAt    time.Time                   `json:"created_at"`
	UpdatedAt    time.Time                   `json:"updated_at"`
}

// ValidateReceiptTemplateDocumentType проверяет нефискальный document_type.
func ValidateReceiptTemplateDocumentType(v ReceiptTemplateDocumentType) error {
	switch v {
	case ReceiptDocumentPrecheck, ReceiptDocumentCheckNonfiscal, ReceiptDocumentTicket,
		ReceiptDocumentKitchenService, ReceiptDocumentCashInOut, ReceiptDocumentAcceptance:
		return nil
	default:
		return fmt.Errorf("%w: unsupported receipt template document_type %q", ErrInvalid, v)
	}
}

// ValidateReceiptTemplateCPL проверяет допустимую плотность символов в строке.
func ValidateReceiptTemplateCPL(cpl int) error {
	switch cpl {
	case 32, 40, 48, 58:
		return nil
	default:
		return fmt.Errorf("%w: cpl must be one of 32, 40, 48, 58", ErrInvalid)
	}
}

// ValidateReceiptTemplateLevel проверяет уровень шаблона (1=ReceiptLine, 2=go text/template).
func ValidateReceiptTemplateLevel(level int) error {
	if level != 1 && level != 2 {
		return fmt.Errorf("%w: level must be 1 or 2", ErrInvalid)
	}
	return nil
}

// NormalizePrinterClass возвращает безопасный printer_class с дефолтом generic.
func NormalizePrinterClass(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "generic"
	}
	return value
}

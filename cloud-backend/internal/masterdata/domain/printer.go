package domain

import (
	"fmt"
	"time"
)

const PermissionPrintersManage = "organization.manage"

// PrinterType — протокол подключения принтера.
type PrinterType string

const (
	PrinterTypeTCP PrinterType = "tcp"
	PrinterTypeUSB PrinterType = "usb"
)

// PrinterDocumentType — тип документа, который принтер обслуживает.
type PrinterDocumentType string

const (
	PrinterDocumentPrecheck       PrinterDocumentType = "precheck"
	PrinterDocumentCheckNonfiscal PrinterDocumentType = "check_nonfiscal"
	PrinterDocumentTicket         PrinterDocumentType = "ticket"
	PrinterDocumentKitchenService PrinterDocumentType = "kitchen_service"
	PrinterDocumentCashInOut      PrinterDocumentType = "cash_in_out"
	PrinterDocumentAcceptance     PrinterDocumentType = "acceptance"
)

// PrinterCodepage — кодовая страница ESC/POS принтера.
type PrinterCodepage string

const (
	PrinterCodepageDefault PrinterCodepage = ""
	PrinterCodepageCP437   PrinterCodepage = "cp437"
	PrinterCodepageCP866   PrinterCodepage = "cp866"
)

// PaperCutType — режим отрезки бумаги.
type PaperCutType string

const (
	PaperCutFull    PaperCutType = "full"
	PaperCutPartial PaperCutType = "partial"
)

// Printer — Cloud-owned ESC/POS принтер, назначенный ресторану.
type Printer struct {
	ID            string                `json:"id"`
	OrgID         string                `json:"org_id"`
	RestaurantID  string                `json:"restaurant_id"`
	Name          string                `json:"name"`
	Type          PrinterType           `json:"type"`
	Address       string                `json:"address,omitempty"`
	Port          *int                  `json:"port,omitempty"`
	DocumentTypes []PrinterDocumentType `json:"document_types"`
	Codepage      PrinterCodepage       `json:"codepage"`
	PaperCutType  PaperCutType          `json:"paper_cut_type"`
	CPL           int                   `json:"cpl"`
	IsActive      bool                  `json:"is_active"`
	Version       int                   `json:"version"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

func ValidatePrinterType(v PrinterType) error {
	switch v {
	case PrinterTypeTCP, PrinterTypeUSB:
		return nil
	}
	return fmt.Errorf("%w: printer type must be tcp or usb", ErrInvalid)
}

func ValidatePrinterDocumentType(v PrinterDocumentType) error {
	switch v {
	case PrinterDocumentPrecheck, PrinterDocumentCheckNonfiscal, PrinterDocumentTicket, PrinterDocumentKitchenService,
		PrinterDocumentCashInOut, PrinterDocumentAcceptance:
		return nil
	}
	return fmt.Errorf("%w: unknown printer document type %q", ErrInvalid, v)
}

func ValidatePrinterCodepage(v PrinterCodepage) error {
	switch v {
	case PrinterCodepageDefault, PrinterCodepageCP437, PrinterCodepageCP866:
		return nil
	}
	return fmt.Errorf("%w: codepage must be '', 'cp437' or 'cp866'", ErrInvalid)
}

func ValidatePaperCutType(v PaperCutType) error {
	switch v {
	case PaperCutFull, PaperCutPartial:
		return nil
	}
	return fmt.Errorf("%w: paper_cut_type must be full or partial", ErrInvalid)
}

func ValidatePrinterCPL(cpl int) error {
	switch cpl {
	case 32, 42, 48, 56, 80:
		return nil
	}
	return fmt.Errorf("%w: cpl must be 32, 42, 48, 56 or 80", ErrInvalid)
}

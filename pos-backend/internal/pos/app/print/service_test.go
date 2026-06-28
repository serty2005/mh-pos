package printqueue

import (
	"testing"

	"mh-pos-platform/receipt/escpos"
	"pos-backend/internal/pos/domain"
)

func TestPrinterConfigFromReceiptPrinter(t *testing.T) {
	port := 9100
	valid := []domain.ReceiptPrinter{
		{ID: "printer-tcp", RestaurantID: "restaurant-1", Name: "Front", Type: escpos.PrinterTypeTCP, Address: "10.25.1.201", Port: &port, DocumentTypes: []domain.ReceiptDocumentType{domain.ReceiptDocumentPrecheck}, Codepage: "cp437", PaperCutType: "partial", CPL: 48, IsActive: true},
		{ID: "printer-usb", RestaurantID: "restaurant-1", Name: "Ticket", Type: escpos.PrinterTypeUSB, Address: `\\.\USB001`, DocumentTypes: []domain.ReceiptDocumentType{domain.ReceiptDocumentTicket}, PaperCutType: "partial", CPL: 48, IsActive: true},
	}
	for _, printer := range valid {
		cfg, err := printerConfig(printer, domain.ReceiptTemplate{PrinterClass: "front"})
		if err != nil {
			t.Fatalf("expected valid printer config for %+v, got %v", printer, err)
		}
		if cfg.Address != printer.Address || cfg.CPL != printer.CPL || cfg.PrinterClass != "front" {
			t.Fatalf("unexpected config from printer %+v: %+v", printer, cfg)
		}
	}

	invalid := domain.ReceiptPrinter{ID: "bad", RestaurantID: "restaurant-1", Name: "Bad", Type: escpos.PrinterTypeTCP, DocumentTypes: []domain.ReceiptDocumentType{domain.ReceiptDocumentPrecheck}, CPL: 48, IsActive: true}
	if _, err := printerConfig(invalid, domain.ReceiptTemplate{}); err == nil || err.Error() != "PRINT_ROUTING_INVALID" {
		t.Fatalf("expected safe invalid routing error, got %v", err)
	}
	usbWithoutPath := domain.ReceiptPrinter{ID: "usb-bad", RestaurantID: "restaurant-1", Name: "USB", Type: escpos.PrinterTypeUSB, DocumentTypes: []domain.ReceiptDocumentType{domain.ReceiptDocumentTicket}, CPL: 48, IsActive: true}
	if _, err := printerConfig(usbWithoutPath, domain.ReceiptTemplate{}); err == nil || err.Error() != "PRINT_ROUTING_INVALID" {
		t.Fatalf("expected safe invalid USB routing error, got %v", err)
	}
}

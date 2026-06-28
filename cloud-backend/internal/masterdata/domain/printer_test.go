package domain

import "testing"

func TestValidatePrinterDocumentTypeIncludesCheckNonfiscal(t *testing.T) {
	if err := ValidatePrinterDocumentType(PrinterDocumentCheckNonfiscal); err != nil {
		t.Fatalf("expected check_nonfiscal printer document type to be valid, got %v", err)
	}
}

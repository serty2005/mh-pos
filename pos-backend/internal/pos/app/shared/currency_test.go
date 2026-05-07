package shared_test

import (
	"testing"

	"pos-backend/internal/pos/app/shared"
)

func TestValidateCurrencyCode(t *testing.T) {
	normalized, err := shared.ValidateCurrencyCode(" rub ")
	if err != nil {
		t.Fatalf("expected RUB to be supported, got %v", err)
	}
	if normalized != "RUB" {
		t.Fatalf("expected RUB, got %s", normalized)
	}
	if _, err := shared.ValidateCurrencyCode("AAA"); err == nil {
		t.Fatal("expected unsupported currency code to be rejected")
	}
}

func TestCurrencyMinorUnit(t *testing.T) {
	rubMinor, err := shared.CurrencyMinorUnit("RUB")
	if err != nil {
		t.Fatalf("expected RUB minor unit to be available, got %v", err)
	}
	if rubMinor != 2 {
		t.Fatalf("expected RUB minor unit 2, got %d", rubMinor)
	}
	kwdMinor, err := shared.CurrencyMinorUnit("kwd")
	if err != nil {
		t.Fatalf("expected KWD minor unit to be available, got %v", err)
	}
	if kwdMinor != 3 {
		t.Fatalf("expected KWD minor unit 3, got %d", kwdMinor)
	}
}

package escpos

import "testing"

func TestCP866RoundtripRussian(t *testing.T) {
	input := "Привет, Ёж №1"
	raw, err := EncodeCP866(input)
	if err != nil {
		t.Fatal(err)
	}
	if got := DecodeCP866(raw); got != input {
		t.Fatalf("roundtrip mismatch: %q", got)
	}
}

func TestCP866RejectsUnsupportedRune(t *testing.T) {
	if _, err := EncodeCP866("🙂"); err == nil {
		t.Fatal("expected unsupported rune error")
	}
	if CanEncodeCP866("🙂") {
		t.Fatal("expected CanEncodeCP866=false")
	}
}

func TestCP866ReplacesReceiptSafeTypography(t *testing.T) {
	raw, err := EncodeCP866("Выставка «TechWorld 2026» — 500,00 ₽…")
	if err != nil {
		t.Fatal(err)
	}
	if got := DecodeCP866(raw); got != "Выставка \"TechWorld 2026\" - 500,00 RUB..." {
		t.Fatalf("normalized CP866 text = %q", got)
	}
	if !CanEncodeCP866("«₽—…") {
		t.Fatal("expected receipt-safe typography to be encodable")
	}
}

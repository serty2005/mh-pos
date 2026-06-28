package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// loadSnapshot читает JSON-фикстуру snapshot и анмаршалит её в v.
func loadSnapshot(t *testing.T, name string, v any) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("fixtures", name))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatalf("unmarshal %s: %v", name, err)
	}
}

func TestProjectPrecheckFixture(t *testing.T) {
	var snapshot PrecheckSnapshot
	loadSnapshot(t, "precheck_snapshot.json", &snapshot)

	got := ProjectPrecheck(snapshot)

	want := PrecheckPrintContext{
		RestaurantName:    "Exhibition TechWorld 2026",
		RestaurantAddress: "Jl. Panglima Polim No.1, Jakarta",
		CashierName:       "Ahmad Kasir",
		ShiftNumber:       42,
		BusinessDate:      "2026-06-24",
		OpenedAt:          "2026-06-24T10:15:00",
		PrecheckNumber:    "00042",
		Lines: []PrecheckLine{
			{
				Name:           "Standard Ticket",
				Quantity:       1,
				UnitPriceMinor: 50000,
				TotalMinor:     60000,
				Modifiers:      []PrecheckModifier{{Name: "VIP Zone", PriceMinor: 10000}},
				IsService:      true,
				QREnabled:      true,
			},
			{
				Name:           "Latte Coffee",
				Quantity:       2,
				UnitPriceMinor: 35000,
				TotalMinor:     70000,
				Modifiers:      []PrecheckModifier{},
				IsService:      false,
				QREnabled:      false,
			},
		},
		SubtotalMinor:       130000,
		DiscountTotalMinor:  5000,
		SurchargeTotalMinor: 0,
		TaxTotalMinor:       20833,
		TotalMinor:          125000,
		CurrencyCode:        "RUB",
		Taxes:               []TaxComponent{{Name: "VAT 20%", BaseMinor: 104167, AmountMinor: 20833}},
		Discounts:           []DiscountComponent{{Name: "Discount 10%", AmountMinor: 5000}},
		IsCopy:              false,
		CopyMarker:          "",
		PrintedAt:           "2026-06-24T10:16:00",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("precheck context mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestProjectTicketFixture(t *testing.T) {
	var snapshot TicketSnapshot
	loadSnapshot(t, "ticket_snapshot.json", &snapshot)

	got := ProjectTicket(snapshot)

	want := ServiceTicketPrintContext{
		TicketNumber:        "019044ab-0000-7000-0000-000000000001",
		TicketDisplayNumber: "0001",
		QRPayload:           "MHT1:019044ab-0000-7000-0000-000000000001",
		ServiceName:         "Standard Ticket",
		CategoryName:        "Events",
		PriceMinor:          50000,
		CurrencyCode:        "RUB",
		SaleDateLocal:       "2026-06-24",
		SaleTimeLocal:       "10:15:00",
		Timezone:            "Asia/Jakarta",
		ValidityMode:        "cash_session",
		ValidityDateLocal:   nil,
		RestaurantName:      "Exhibition TechWorld 2026",
		RestaurantAddress:   "Jl. Panglima Polim No.1, Jakarta",
		CashierName:         "Ahmad Kasir",
		ShiftNumber:         42,
		BusinessDate:        "2026-06-24",
		IsCopy:              false,
		CopyMarker:          "",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ticket context mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

// TestProjectorsDeriveCopyMarker проверяет правило reprint: is_copy=true → "COPY".
func TestProjectorsDeriveCopyMarker(t *testing.T) {
	precheck := ProjectPrecheck(PrecheckSnapshot{IsCopy: true})
	if precheck.CopyMarker != "COPY" {
		t.Fatalf("precheck copy marker = %q, want COPY", precheck.CopyMarker)
	}
	ticket := ProjectTicket(TicketSnapshot{IsCopy: true})
	if ticket.CopyMarker != "COPY" {
		t.Fatalf("ticket copy marker = %q, want COPY", ticket.CopyMarker)
	}
}

// TestProjectTicketKeepsAbsoluteValidityDate проверяет, что validity_date_local
// для absolute_date пробрасывается из snapshot без потери.
func TestProjectTicketKeepsAbsoluteValidityDate(t *testing.T) {
	date := "2026-12-31"
	got := ProjectTicket(TicketSnapshot{
		ValidityMode:      "absolute_date",
		ValidityDateLocal: &date,
	})
	if got.ValidityDateLocal == nil || *got.ValidityDateLocal != date {
		t.Fatalf("validity_date_local = %v, want %q", got.ValidityDateLocal, date)
	}
}

// TestProjectorsArePure фиксирует инвариант проекции: проекторы не имеют
// зависимостей кроме snapshot-значения (БД/каталог/runtime читать структурно
// нечем), детерминированы и не мутируют вход.
func TestProjectorsArePure(t *testing.T) {
	var precheck PrecheckSnapshot
	loadSnapshot(t, "precheck_snapshot.json", &precheck)
	before, err := json.Marshal(precheck)
	if err != nil {
		t.Fatal(err)
	}

	first := ProjectPrecheck(precheck)
	second := ProjectPrecheck(precheck)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("ProjectPrecheck не детерминирован")
	}

	after, err := json.Marshal(precheck)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("ProjectPrecheck мутировал входной snapshot")
	}
}

package engine

// Этот файл задаёт контракт immutable print snapshot (вход проектора) и сами
// проекторы snapshot -> print context.
//
// Инвариант проекции: ProjectPrecheck и ProjectTicket — чистые функции от
// переданного snapshot-значения. Они не читают БД, текущий каталог, текущие
// цены или runtime-состояние; единственный источник данных — immutable snapshot,
// собранный на момент issue/print. Отсутствие зависимостей в сигнатуре делает
// чтение внешних источников структурно невозможным (см. projector_test.go).
//
// Snapshot-типы ниже — print-ориентированное immutable представление документа.
// Маппинг доменных precheck/ticket snapshot из pos-backend в этот контракт
// выполняет очередь печати (POS-74) и здесь не реализуется.

// copyMarkerText — маркер копии, проставляемый при reprint.
const copyMarkerText = "COPY"

// PrecheckSnapshot — immutable snapshot предчека для проекции в print context.
type PrecheckSnapshot struct {
	DocumentType   string             `json:"document_type"` // "precheck"
	PrecheckNumber string             `json:"precheck_number"`
	Restaurant     RestaurantSnapshot `json:"restaurant"`
	Cashier        CashierSnapshot    `json:"cashier"`
	ShiftNumber    int                `json:"shift_number"`
	BusinessDate   string             `json:"business_date"`
	OpenedAt       string             `json:"opened_at"`
	PrintedAt      string             `json:"printed_at"` // момент печати/reprint, фиксируется в snapshot

	CurrencyCode        string `json:"currency_code"`
	SubtotalMinor       int64  `json:"subtotal_minor"`
	DiscountTotalMinor  int64  `json:"discount_total_minor"`
	SurchargeTotalMinor int64  `json:"surcharge_total_minor"`
	TaxTotalMinor       int64  `json:"tax_total_minor"`
	TotalMinor          int64  `json:"total_minor"`

	Lines     []PrecheckSnapshotLine `json:"lines"`
	Taxes     []TaxSnapshot          `json:"taxes"`
	Discounts []DiscountSnapshot     `json:"discounts"`

	IsCopy bool `json:"is_copy"`
}

// PrecheckSnapshotLine — строка заказа в immutable snapshot предчека.
type PrecheckSnapshotLine struct {
	Name           string                     `json:"name"`
	Quantity       int64                      `json:"quantity"`
	UnitPriceMinor int64                      `json:"unit_price_minor"`
	TotalMinor     int64                      `json:"total_minor"`
	Modifiers      []PrecheckSnapshotModifier `json:"modifiers"`
	IsService      bool                       `json:"is_service"`
	QREnabled      bool                       `json:"qr_enabled"`
}

// PrecheckSnapshotModifier — выбранный модификатор строки в snapshot.
type PrecheckSnapshotModifier struct {
	Name       string `json:"name"`
	PriceMinor int64  `json:"price_minor"`
}

// TaxSnapshot — налоговая компонента в immutable snapshot.
type TaxSnapshot struct {
	Name        string `json:"name"`
	BaseMinor   int64  `json:"base_minor"`
	AmountMinor int64  `json:"amount_minor"`
}

// DiscountSnapshot — применённая скидка в immutable snapshot.
type DiscountSnapshot struct {
	Name        string `json:"name"`
	AmountMinor int64  `json:"amount_minor"`
}

// TicketSnapshot — immutable snapshot сервисного билета для проекции.
type TicketSnapshot struct {
	DocumentType        string `json:"document_type"` // "ticket"
	TicketNumber        string `json:"ticket_number"`
	TicketDisplayNumber string `json:"ticket_display_number"`
	QRPayload           string `json:"qr_payload"`

	ServiceName  string `json:"service_name"`
	CategoryName string `json:"category_name"`
	PriceMinor   int64  `json:"price_minor"`
	CurrencyCode string `json:"currency_code"`

	SaleDateLocal string `json:"sale_date_local"`
	SaleTimeLocal string `json:"sale_time_local"`
	Timezone      string `json:"timezone"`

	ValidityMode      string  `json:"validity_mode"`
	ValidityDateLocal *string `json:"validity_date_local"`

	Restaurant RestaurantSnapshot `json:"restaurant"`
	Cashier    CashierSnapshot    `json:"cashier"`

	ShiftNumber  int    `json:"shift_number"`
	BusinessDate string `json:"business_date"`

	IsCopy bool `json:"is_copy"`
}

// RestaurantSnapshot — реквизиты ресторана/мероприятия в snapshot.
type RestaurantSnapshot struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// CashierSnapshot — реквизиты кассира в snapshot.
type CashierSnapshot struct {
	Name string `json:"name"`
}

// ProjectPrecheck проецирует immutable snapshot предчека в PrecheckPrintContext.
// Функция детерминирована и зависит только от snapshot.
func ProjectPrecheck(snapshot PrecheckSnapshot) PrecheckPrintContext {
	ctx := PrecheckPrintContext{
		RestaurantName:    snapshot.Restaurant.Name,
		RestaurantAddress: snapshot.Restaurant.Address,
		CashierName:       snapshot.Cashier.Name,
		ShiftNumber:       snapshot.ShiftNumber,
		BusinessDate:      snapshot.BusinessDate,
		OpenedAt:          snapshot.OpenedAt,
		PrecheckNumber:    snapshot.PrecheckNumber,

		SubtotalMinor:       snapshot.SubtotalMinor,
		DiscountTotalMinor:  snapshot.DiscountTotalMinor,
		SurchargeTotalMinor: snapshot.SurchargeTotalMinor,
		TaxTotalMinor:       snapshot.TaxTotalMinor,
		TotalMinor:          snapshot.TotalMinor,
		CurrencyCode:        snapshot.CurrencyCode,

		IsCopy:     snapshot.IsCopy,
		CopyMarker: copyMarker(snapshot.IsCopy),
		PrintedAt:  snapshot.PrintedAt,
	}

	ctx.Lines = make([]PrecheckLine, 0, len(snapshot.Lines))
	for _, line := range snapshot.Lines {
		projected := PrecheckLine{
			Name:           line.Name,
			Quantity:       line.Quantity,
			UnitPriceMinor: line.UnitPriceMinor,
			TotalMinor:     line.TotalMinor,
			IsService:      line.IsService,
			QREnabled:      line.QREnabled,
		}
		projected.Modifiers = make([]PrecheckModifier, 0, len(line.Modifiers))
		for _, mod := range line.Modifiers {
			projected.Modifiers = append(projected.Modifiers, PrecheckModifier{
				Name:       mod.Name,
				PriceMinor: mod.PriceMinor,
			})
		}
		ctx.Lines = append(ctx.Lines, projected)
	}

	ctx.Taxes = make([]TaxComponent, 0, len(snapshot.Taxes))
	for _, tax := range snapshot.Taxes {
		ctx.Taxes = append(ctx.Taxes, TaxComponent{
			Name:        tax.Name,
			BaseMinor:   tax.BaseMinor,
			AmountMinor: tax.AmountMinor,
		})
	}

	ctx.Discounts = make([]DiscountComponent, 0, len(snapshot.Discounts))
	for _, discount := range snapshot.Discounts {
		ctx.Discounts = append(ctx.Discounts, DiscountComponent{
			Name:        discount.Name,
			AmountMinor: discount.AmountMinor,
		})
	}

	return ctx
}

// ProjectTicket проецирует immutable snapshot билета в ServiceTicketPrintContext.
// Функция детерминирована и зависит только от snapshot.
func ProjectTicket(snapshot TicketSnapshot) ServiceTicketPrintContext {
	return ServiceTicketPrintContext{
		TicketNumber:        snapshot.TicketNumber,
		TicketDisplayNumber: snapshot.TicketDisplayNumber,
		QRPayload:           snapshot.QRPayload,

		ServiceName:  snapshot.ServiceName,
		CategoryName: snapshot.CategoryName,
		PriceMinor:   snapshot.PriceMinor,
		CurrencyCode: snapshot.CurrencyCode,

		SaleDateLocal: snapshot.SaleDateLocal,
		SaleTimeLocal: snapshot.SaleTimeLocal,
		Timezone:      snapshot.Timezone,

		ValidityMode:      snapshot.ValidityMode,
		ValidityDateLocal: snapshot.ValidityDateLocal,

		RestaurantName:    snapshot.Restaurant.Name,
		RestaurantAddress: snapshot.Restaurant.Address,

		CashierName:  snapshot.Cashier.Name,
		ShiftNumber:  snapshot.ShiftNumber,
		BusinessDate: snapshot.BusinessDate,

		IsCopy:     snapshot.IsCopy,
		CopyMarker: copyMarker(snapshot.IsCopy),
	}
}

// copyMarker возвращает "COPY" для reprint и пустую строку для оригинала.
func copyMarker(isCopy bool) string {
	if isCopy {
		return copyMarkerText
	}
	return ""
}

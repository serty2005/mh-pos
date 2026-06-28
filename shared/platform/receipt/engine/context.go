// Package engine содержит print context view-model и проекторы из immutable
// snapshot для нефискальной печати (precheck, service ticket).
//
// Print context — плоская стабильная View-model, потребляемая шаблонным движком
// (POS-73) и очередью печати (POS-74). Контракт зафиксирован в
// docs/backend/RECEIPT-PRINT-SPEC.md (§Print Context). Денежные значения в
// контексте остаются целыми minor units; человекочитаемое форматирование
// (см. FormatMoneyMinor) выполняется шаблоном на этапе рендера, а не проекцией.
package engine

// PrecheckPrintContext — view-model для document_type=precheck.
// Соответствует JSON-схеме PrecheckPrintContext из RECEIPT-PRINT-SPEC.md.
type PrecheckPrintContext struct {
	// Шапка
	RestaurantName    string `json:"restaurant_name"`
	RestaurantAddress string `json:"restaurant_address"`
	CashierName       string `json:"cashier_name"`
	ShiftNumber       int    `json:"shift_number"`    // порядковый номер кассовой смены
	BusinessDate      string `json:"business_date"`   // YYYY-MM-DD
	OpenedAt          string `json:"opened_at"`       // ISO 8601 без TZ (local)
	PrecheckNumber    string `json:"precheck_number"` // display number

	// Строки заказа
	Lines []PrecheckLine `json:"lines"`

	// Итоги (minor units)
	SubtotalMinor       int64  `json:"subtotal_minor"`
	DiscountTotalMinor  int64  `json:"discount_total_minor"`
	SurchargeTotalMinor int64  `json:"surcharge_total_minor"`
	TaxTotalMinor       int64  `json:"tax_total_minor"` // НДС включён в итог (inclusive)
	TotalMinor          int64  `json:"total_minor"`
	CurrencyCode        string `json:"currency_code"`

	// Налоговые компоненты (детализация)
	Taxes []TaxComponent `json:"taxes"`

	// Скидки (может быть пустым)
	Discounts []DiscountComponent `json:"discounts"`

	// Метаданные
	IsCopy     bool   `json:"is_copy"`     // true при reprint
	CopyMarker string `json:"copy_marker"` // "COPY" при reprint, иначе ""
	PrintedAt  string `json:"printed_at"`
}

// PrecheckLine — строка заказа в precheck context.
type PrecheckLine struct {
	Name           string             `json:"name"`
	Quantity       int64              `json:"quantity"`
	UnitPriceMinor int64              `json:"unit_price_minor"`
	TotalMinor     int64              `json:"total_minor"`
	Modifiers      []PrecheckModifier `json:"modifiers"`
	IsService      bool               `json:"is_service"`
	QREnabled      bool               `json:"qr_enabled"`
}

// PrecheckModifier — выбранный модификатор строки заказа.
type PrecheckModifier struct {
	Name       string `json:"name"`
	PriceMinor int64  `json:"price_minor"`
}

// TaxComponent — налоговая компонента для детализации в чеке.
type TaxComponent struct {
	Name        string `json:"name"`
	BaseMinor   int64  `json:"base_minor"`
	AmountMinor int64  `json:"amount_minor"`
}

// DiscountComponent — применённая скидка для детализации в чеке.
type DiscountComponent struct {
	Name        string `json:"name"`
	AmountMinor int64  `json:"amount_minor"`
}

// ServiceTicketPrintContext — view-model для document_type=ticket.
// Соответствует JSON-схеме ServiceTicketPrintContext из RECEIPT-PRINT-SPEC.md.
type ServiceTicketPrintContext struct {
	// Идентификация билета
	TicketNumber        string `json:"ticket_number"`         // UUIDv7 = ticket_number
	TicketDisplayNumber string `json:"ticket_display_number"` // порядковый в смене
	QRPayload           string `json:"qr_payload"`

	// Что продано
	ServiceName  string `json:"service_name"`
	CategoryName string `json:"category_name"` // из menu category
	PriceMinor   int64  `json:"price_minor"`
	CurrencyCode string `json:"currency_code"`

	// Когда продано
	SaleDateLocal string `json:"sale_date_local"` // YYYY-MM-DD
	SaleTimeLocal string `json:"sale_time_local"` // HH:MM:SS
	Timezone      string `json:"timezone"`

	// Срок действия
	ValidityMode      string  `json:"validity_mode"`       // cash_session|business_date|absolute_date
	ValidityDateLocal *string `json:"validity_date_local"` // YYYY-MM-DD или null для cash_session

	// Ресторан / мероприятие
	RestaurantName    string `json:"restaurant_name"`
	RestaurantAddress string `json:"restaurant_address"`

	// Кассир
	CashierName  string `json:"cashier_name"`
	ShiftNumber  int    `json:"shift_number"`
	BusinessDate string `json:"business_date"`

	// Метаданные
	IsCopy     bool   `json:"is_copy"`
	CopyMarker string `json:"copy_marker"`
}

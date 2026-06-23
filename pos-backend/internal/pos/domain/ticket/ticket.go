// Package ticket описывает доменную модель проданной QR-билетной единицы (ticket unit),
// которая выпускается ровно один раз после закрытия final check для каждой QR-enabled line.
package ticket

import (
	"encoding/json"
	"time"
)

// ValidityMode определяет, как резолвится срок действия билета в момент выпуска.
// Значения доставляются Cloud-ом в catalog item (POS-52) и фиксируются immutable в ticket unit.
type ValidityMode string

const (
	// ValidityCashSession — билет действует в рамках кассовой смены продажи.
	ValidityCashSession ValidityMode = "cash_session"
	// ValidityBusinessDate — билет действует в business_date_local продажи.
	ValidityBusinessDate ValidityMode = "business_date"
	// ValidityAbsoluteDate — билет действует до одной заданной локальной даты ресторана.
	ValidityAbsoluteDate ValidityMode = "absolute_date"
)

// IsValid сообщает, поддерживается ли validity mode до запуска.
func (m ValidityMode) IsValid() bool {
	switch m {
	case ValidityCashSession, ValidityBusinessDate, ValidityAbsoluteDate:
		return true
	default:
		return false
	}
}

const (
	// PrintStatusPending — билет выпущен, но физическая печать выполняется отдельной print subsystem (POS-64).
	PrintStatusPending = "pending"

	// QRPayloadPrefix — версионный префикс QR payload. Checker (post-deploy QR lookup) парсит
	// уникальный ticket number из payload; PIN/token/payment-sensitive данные не включаются.
	QRPayloadPrefix = "MHT1:"
)

// TicketUnit — неизменяемая единица проданного QR-билета. Выпускается один раз после final check;
// reprint использует тот же ticket number и QR, не создавая новую единицу.
type TicketUnit struct {
	ID                string          `json:"id"`
	TicketNumber      string          `json:"ticket_number"`
	RestaurantID      string          `json:"restaurant_id"`
	DeviceID          string          `json:"device_id"`
	CashSessionID     string          `json:"cash_session_id"`
	ShiftID           string          `json:"shift_id"`
	CheckID           string          `json:"check_id"`
	OrderID           string          `json:"order_id"`
	OrderLineID       string          `json:"order_line_id"`
	CatalogItemID     string          `json:"catalog_item_id"`
	MenuItemID        string          `json:"menu_item_id"`
	Name              string          `json:"name"`
	SaleDateLocal     string          `json:"sale_date_local"`
	Timezone          string          `json:"timezone"`
	ValidityMode      ValidityMode    `json:"validity_mode"`
	ValidityDateLocal string          `json:"validity_date_local,omitempty"`
	CashShiftSequence int64           `json:"cash_shift_sequence"`
	QRPayload         string          `json:"qr_payload"`
	PrintStatus       string          `json:"print_status"`
	Snapshot          json.RawMessage `json:"snapshot,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// BuildQRPayload строит безопасный версионный QR payload из уникального ticket number.
func BuildQRPayload(ticketNumber string) string {
	return QRPayloadPrefix + ticketNumber
}

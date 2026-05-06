package domain

import (
	"pos-backend/internal/pos/domain/cash"
	"pos-backend/internal/pos/domain/catalog"
	"pos-backend/internal/pos/domain/check"
	"pos-backend/internal/pos/domain/device"
	"pos-backend/internal/pos/domain/employee"
	"pos-backend/internal/pos/domain/inventory"
	"pos-backend/internal/pos/domain/menu"
	"pos-backend/internal/pos/domain/order"
	"pos-backend/internal/pos/domain/precheck"
	"pos-backend/internal/pos/domain/restaurant"
	"pos-backend/internal/pos/domain/shared"
	"pos-backend/internal/pos/domain/shift"
)

var (
	ErrInvalid          = shared.ErrInvalid
	ErrNotFound         = shared.ErrNotFound
	ErrConflict         = shared.ErrConflict
	ErrForbidden        = shared.ErrForbidden
	ErrDuplicate        = shared.ErrDuplicate
	ErrDuplicateCommand = shared.ErrDuplicateCommand
)

type Restaurant = restaurant.Restaurant

type Device = device.Device

type Role = employee.Role
type Employee = employee.Employee
type ManagerOverrideAudit = employee.ManagerOverrideAudit

type CatalogItemType = catalog.CatalogItemType
type CatalogItem = catalog.CatalogItem

const (
	CatalogItemIngredient = catalog.CatalogItemIngredient
	CatalogItemDish       = catalog.CatalogItemDish
	CatalogItemGood       = catalog.CatalogItemGood
)

type MenuItem = menu.MenuItem

type ShiftStatus = shift.ShiftStatus
type Shift = shift.Shift

const (
	ShiftOpen   = shift.ShiftOpen
	ShiftClosed = shift.ShiftClosed
)

type OrderStatus = order.OrderStatus
type Order = order.Order

const (
	OrderOpen      = order.OrderOpen
	OrderLocked    = order.OrderLocked
	OrderClosed    = order.OrderClosed
	OrderCancelled = order.OrderCancelled
)

type OrderLineStatus = order.OrderLineStatus
type OrderLine = order.OrderLine

const (
	OrderLineActive    = order.OrderLineActive
	OrderLineCancelled = order.OrderLineCancelled
)

type CheckStatus = check.CheckStatus
type Check = check.Check

const (
	CheckOpen     = check.CheckOpen
	CheckPaid     = check.CheckPaid
	CheckRefunded = check.CheckRefunded
	CheckVoided   = check.CheckVoided
)

type PrecheckStatus = precheck.PrecheckStatus
type Precheck = precheck.Precheck

const (
	PrecheckIssued     = precheck.PrecheckIssued
	PrecheckClosed     = precheck.PrecheckClosed
	PrecheckCancelled  = precheck.PrecheckCancelled
	PrecheckSuperseded = precheck.PrecheckSuperseded
)

type PaymentStatus = check.PaymentStatus
type PaymentMethod = check.PaymentMethod
type Payment = check.Payment
type PaymentAttempt = check.PaymentAttempt

const (
	PaymentCaptured = check.PaymentCaptured
	PaymentRefunded = check.PaymentRefunded
	PaymentFailed   = check.PaymentFailed
	PaymentCash     = check.PaymentCash
	PaymentCard     = check.PaymentCard
	PaymentOther    = check.PaymentOther
)

type CashSessionStatus = cash.CashSessionStatus
type CashDrawerEventType = cash.CashDrawerEventType
type CashSession = cash.CashSession
type CashDrawerEvent = cash.CashDrawerEvent

const (
	CashSessionOpen     = cash.CashSessionOpen
	CashSessionClosed   = cash.CashSessionClosed
	CashDrawerCashIn    = cash.CashDrawerCashIn
	CashDrawerCashOut   = cash.CashDrawerCashOut
	CashDrawerNoSale    = cash.CashDrawerNoSale
	CashDrawerCashCount = cash.CashDrawerCashCount
)

type OutboxStatus = shared.OutboxStatus
type CommandOrigin = shared.CommandOrigin
type OutboxMessage = shared.OutboxMessage
type SyncStatus = shared.SyncStatus
type LocalEvent = shared.LocalEvent
type SyncEnvelope = shared.SyncEnvelope

const (
	OutboxPending    = shared.OutboxPending
	OutboxProcessing = shared.OutboxProcessing
	OutboxSent       = shared.OutboxSent
	OutboxFailed     = shared.OutboxFailed
	OutboxSuspended  = shared.OutboxSuspended
	OriginEdgeDevice = shared.OriginEdgeDevice
	OriginCloudSync  = shared.OriginCloudSync
	OriginSystemSeed = shared.OriginSystemSeed

	SyncEnvelopeVersion = shared.SyncEnvelopeVersion
)

type RecipeVersionStatus = inventory.RecipeVersionStatus
type RecipeVersion = inventory.RecipeVersion
type RecipeLine = inventory.RecipeLine
type PurchaseReceiptStatus = inventory.PurchaseReceiptStatus
type PurchaseReceipt = inventory.PurchaseReceipt
type PurchaseReceiptLine = inventory.PurchaseReceiptLine
type StockDocumentType = inventory.StockDocumentType
type StockDocumentStatus = inventory.StockDocumentStatus
type StockDocument = inventory.StockDocument
type StockMoveType = inventory.StockMoveType
type StockMove = inventory.StockMove
type StockBalance = inventory.StockBalance
type ItemCostType = inventory.ItemCostType
type ItemCost = inventory.ItemCost

const (
	RecipeVersionDraft           = inventory.RecipeVersionDraft
	RecipeVersionActive          = inventory.RecipeVersionActive
	RecipeVersionArchived        = inventory.RecipeVersionArchived
	PurchaseReceiptDraft         = inventory.PurchaseReceiptDraft
	PurchaseReceiptPosted        = inventory.PurchaseReceiptPosted
	PurchaseReceiptCancelled     = inventory.PurchaseReceiptCancelled
	StockDocumentPurchaseReceipt = inventory.StockDocumentPurchaseReceipt
	StockDocumentAdjustment      = inventory.StockDocumentAdjustment
	StockDocumentTransfer        = inventory.StockDocumentTransfer
	StockDocumentWriteOff        = inventory.StockDocumentWriteOff
	StockDocumentProduction      = inventory.StockDocumentProduction
	StockDocumentDraft           = inventory.StockDocumentDraft
	StockDocumentPosted          = inventory.StockDocumentPosted
	StockDocumentCancelled       = inventory.StockDocumentCancelled
	StockMoveIn                  = inventory.StockMoveIn
	StockMoveOut                 = inventory.StockMoveOut
	StockMoveAdjustment          = inventory.StockMoveAdjustment
	ItemCostLastPurchase         = inventory.ItemCostLastPurchase
	ItemCostMovingAverage        = inventory.ItemCostMovingAverage
)

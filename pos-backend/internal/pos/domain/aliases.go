// Package domain сохраняет transitional facade над разнесенными domain packages.
// Новые bounded contexts не должны превращать этот пакет в общий god-domain.
package domain

import (
	"time"

	"pos-backend/internal/pos/domain/cash"
	"pos-backend/internal/pos/domain/catalog"
	"pos-backend/internal/pos/domain/check"
	"pos-backend/internal/pos/domain/device"
	"pos-backend/internal/pos/domain/employee"
	"pos-backend/internal/pos/domain/financial"
	"pos-backend/internal/pos/domain/floor"
	"pos-backend/internal/pos/domain/inventory"
	"pos-backend/internal/pos/domain/menu"
	"pos-backend/internal/pos/domain/order"
	"pos-backend/internal/pos/domain/precheck"
	"pos-backend/internal/pos/domain/pricing"
	"pos-backend/internal/pos/domain/restaurant"
	"pos-backend/internal/pos/domain/shared"
	"pos-backend/internal/pos/domain/shift"
	"pos-backend/internal/pos/domain/storage"
)

var (
	ErrInvalid          = shared.ErrInvalid
	ErrNotFound         = shared.ErrNotFound
	ErrConflict         = shared.ErrConflict
	ErrForbidden        = shared.ErrForbidden
	ErrTooManyRequests  = shared.ErrTooManyRequests
	ErrDuplicate        = shared.ErrDuplicate
	ErrDuplicateCommand = shared.ErrDuplicateCommand
)

type Restaurant = restaurant.Restaurant
type BusinessDayMode = restaurant.BusinessDayMode

const (
	BusinessDayStandard = restaurant.BusinessDayStandard
	BusinessDay24x7     = restaurant.BusinessDay24x7
)

type Device = device.Device
type EdgeNodeStatus = device.EdgeNodeStatus
type EdgeNodeIdentity = device.EdgeNodeIdentity
type ClientDeviceStatus = device.ClientDeviceStatus
type ClientDevice = device.ClientDevice
type PairingStatus = device.PairingStatus
type ProvisioningStatus = device.ProvisioningStatus
type EdgeProvisioningState = device.EdgeProvisioningState
type ProvisioningStatusView = device.ProvisioningStatusView

const (
	EdgeNodePaired                          = device.EdgeNodePaired
	ClientDeviceActive                      = device.ClientDeviceActive
	ProvisioningNotConfigured               = device.ProvisioningNotConfigured
	ProvisioningUnpairedRegistered          = device.ProvisioningUnpairedRegistered
	ProvisioningAssignedDownloadingSnapshot = device.ProvisioningAssignedDownloadingSnapshot
	ProvisioningPaired                      = device.ProvisioningPaired
	ProvisioningError                       = device.ProvisioningError
)

type Role = employee.Role
type Employee = employee.Employee
type ManagerOverrideAudit = employee.ManagerOverrideAudit
type AuthSessionStatus = employee.AuthSessionStatus
type AuthSession = employee.AuthSession
type ActorContext = employee.ActorContext
type PinLoginResult = employee.PinLoginResult

const (
	AuthSessionActive  = employee.AuthSessionActive
	AuthSessionRevoked = employee.AuthSessionRevoked
)

type Hall = floor.Hall
type Table = floor.Table

type CatalogItemType = catalog.CatalogItemType
type CatalogItem = catalog.CatalogItem

const (
	CatalogItemDish         = catalog.CatalogItemDish
	CatalogItemGood         = catalog.CatalogItemGood
	CatalogItemSemiFinished = catalog.CatalogItemSemiFinished
	CatalogItemService      = catalog.CatalogItemService
)

type CatalogFolder = catalog.CatalogFolder
type FolderParameter = catalog.FolderParameter
type CatalogTag = catalog.CatalogTag
type CatalogItemTag = catalog.CatalogItemTag
type ModifierTargetType = catalog.ModifierTargetType
type ModifierGroup = catalog.ModifierGroup
type ModifierOption = catalog.ModifierOption
type ModifierGroupBinding = catalog.ModifierGroupBinding
type MenuItemModifierGroup = catalog.MenuItemModifierGroup

const (
	ModifierTargetMenuItem    = catalog.ModifierTargetMenuItem
	ModifierTargetCatalogItem = catalog.ModifierTargetCatalogItem
	ModifierTargetFolder      = catalog.ModifierTargetFolder
	ModifierTargetTag         = catalog.ModifierTargetTag
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
type OrderSummary = order.OrderSummary
type ClosedOrderListQuery = order.ClosedOrderListQuery

const (
	OrderOpen      = order.OrderOpen
	OrderLocked    = order.OrderLocked
	OrderClosed    = order.OrderClosed
	OrderCancelled = order.OrderCancelled
)

type OrderLineStatus = order.OrderLineStatus
type OrderLine = order.OrderLine
type LineModifier = order.LineModifier

const (
	OrderLineActive    = order.OrderLineActive
	OrderLineCancelled = order.OrderLineCancelled
	OrderLineVoided    = order.OrderLineVoided
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

type FinancialOperationType = financial.OperationType
type FinancialOperationKind = financial.OperationKind
type FinancialOperationStatus = financial.OperationStatus
type InventoryDisposition = financial.InventoryDisposition
type FinancialOperationItemScope = financial.OperationItemScope
type FinancialOperation = financial.Operation
type FinancialOperationItem = financial.OperationItem
type FinancialOperationListQuery = financial.OperationListQuery

const (
	FinancialOperationCancellation = financial.OperationCancellation
	FinancialOperationRefund       = financial.OperationRefund
	FinancialOperationFull         = financial.OperationFull
	FinancialOperationPartial      = financial.OperationPartial
	FinancialOperationRecorded     = financial.OperationRecorded
	InventoryNoStockEffect         = financial.InventoryNoStockEffect
	InventoryReturnToStock         = financial.InventoryReturnToStock
	InventoryWriteOffWaste         = financial.InventoryWriteOffWaste
	InventoryManualReview          = financial.InventoryManualReview
	FinancialItemWholeCheck        = financial.ItemWholeCheck
	FinancialItemOrderLine         = financial.ItemOrderLine
	FinancialItemModifierLine      = financial.ItemModifierLine
	FinancialItemServiceCharge     = financial.ItemServiceCharge
	FinancialItemTip               = financial.ItemTip
	FinancialItemPayment           = financial.ItemPayment
)

type AmountKind = pricing.AmountKind
type DiscountScope = pricing.DiscountScope
type ModifierType = pricing.ModifierType
type SurchargeKind = pricing.SurchargeKind
type TaxMode = pricing.TaxMode
type TaxRuleKind = pricing.TaxRuleKind
type Discount = pricing.Discount
type Surcharge = pricing.Surcharge
type OrderDiscount = pricing.OrderDiscount
type OrderSurcharge = pricing.OrderSurcharge
type TaxProfile = pricing.TaxProfile
type TaxRule = pricing.TaxRule
type ServiceChargeRule = pricing.ServiceChargeRule
type PricingPolicyKind = pricing.PricingPolicyKind
type PricingPolicy = pricing.PricingPolicy
type CalculationModifier = pricing.CalculationModifier
type CalculationInput = pricing.CalculationInput
type CalculationResult = pricing.CalculationResult
type CalculationSnapshot = pricing.CalculationSnapshot
type LineBreakdown = pricing.LineBreakdown
type DiscountBreakdown = pricing.DiscountBreakdown
type SurchargeBreakdown = pricing.SurchargeBreakdown
type TaxComponent = pricing.TaxComponent
type TaxComponentBreakdown = pricing.TaxComponentBreakdown
type LineModifierInput = pricing.LineModifierInput

const (
	AmountPercentage = pricing.AmountPercentage
	AmountFixed      = pricing.AmountFixed

	DiscountScopeLine  = pricing.DiscountScopeLine
	DiscountScopeOrder = pricing.DiscountScopeOrder

	ModifierTypeDiscount  = pricing.ModifierTypeDiscount
	ModifierTypeSurcharge = pricing.ModifierTypeSurcharge

	PricingPolicyDiscount  = pricing.PricingPolicyDiscount
	PricingPolicySurcharge = pricing.PricingPolicySurcharge

	SurchargeServiceCharge = pricing.SurchargeServiceCharge
	SurchargePB1ServiceFee = pricing.SurchargePB1ServiceFee
	SurchargeManual        = pricing.SurchargeManual

	TaxModeExclusive = pricing.TaxModeExclusive
	TaxModeInclusive = pricing.TaxModeInclusive

	TaxRulePercentage = pricing.TaxRulePercentage
	TaxRuleFixed      = pricing.TaxRuleFixed
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
type DataOwner = shared.DataOwner
type SyncDirection = shared.SyncDirection
type SyncMode = shared.SyncMode
type MasterDataStream = shared.MasterDataStream
type MasterDataSyncState = shared.MasterDataSyncState
type MasterRecordSyncMeta = shared.MasterRecordSyncMeta
type SyncExchangeState = shared.SyncExchangeState
type SyncExchangeRequest = shared.SyncExchangeRequest
type SyncExchangeEdgeEvent = shared.SyncExchangeEdgeEvent
type SyncExchangeStreamRequest = shared.SyncExchangeStreamRequest
type CloudPackage = shared.CloudPackage
type OutboxMessage = shared.OutboxMessage
type SyncStatus = shared.SyncStatus
type LocalEvent = shared.LocalEvent
type SyncEnvelope = shared.SyncEnvelope
type ReprintDocument = shared.ReprintDocument
type SQLiteDatabaseStats = storage.SQLiteDatabaseStats
type StorageTableCounts = storage.TableCounts
type StorageBusinessDateRange = storage.BusinessDateRange
type ClosedOrdersBusinessDateCount = storage.ClosedOrdersBusinessDateCount
type StorageOutboxStatusCount = storage.OutboxStatusCount
type StorageRetentionCapability = storage.RetentionCapability
type StorageLifecycleStatus = storage.LifecycleStatus
type StorageRetentionEligibleCounts = storage.RetentionEligibleCounts
type StorageRetentionDryRunResult = storage.RetentionDryRunResult
type StorageArchiveExportCounts = storage.ArchiveExportCounts
type StorageArchiveSourceMetadata = storage.ArchiveSourceMetadata
type StorageArchiveMetadataRow = storage.ArchiveMetadataRow
type StorageArchiveTableManifest = storage.ArchiveTableManifest
type StorageArchiveManifest = storage.ArchiveManifest
type StorageArchiveExportResult = storage.ArchiveExportResult
type StorageArchivePlanProtectedFlags = storage.ArchivePlanProtectedFlags
type StorageArchivePlanTableManifest = storage.ArchivePlanTableManifest
type StorageArchivePlanManifest = storage.ArchivePlanManifest
type StorageArchiveExportPlan = storage.ArchiveExportPlan

const (
	OutboxPending    = shared.OutboxPending
	OutboxProcessing = shared.OutboxProcessing
	OutboxSent       = shared.OutboxSent
	OutboxFailed     = shared.OutboxFailed
	OutboxSuspended  = shared.OutboxSuspended
	OriginEdgeDevice = shared.OriginEdgeDevice
	OriginCloudSync  = shared.OriginCloudSync
	OriginSystemSeed = shared.OriginSystemSeed

	DataOwnerCloud = shared.DataOwnerCloud
	DataOwnerEdge  = shared.DataOwnerEdge
	DataOwnerMixed = shared.DataOwnerMixed

	SyncDirectionCloudToEdge = shared.SyncDirectionCloudToEdge
	SyncDirectionEdgeToCloud = shared.SyncDirectionEdgeToCloud
	SyncDirectionLocalOnly   = shared.SyncDirectionLocalOnly

	SyncModeFullSnapshot = shared.SyncModeFullSnapshot
	SyncModeIncremental  = shared.SyncModeIncremental

	SyncExchangeProtocolVersion = shared.SyncExchangeProtocolVersion
	SyncExchangeStatusAccepted  = shared.SyncExchangeStatusAccepted
	SyncExchangeStatusPartial   = shared.SyncExchangeStatusPartial

	MasterDataStreamRestaurants = shared.MasterDataStreamRestaurants
	MasterDataStreamDevices     = shared.MasterDataStreamDevices
	MasterDataStreamStaff       = shared.MasterDataStreamStaff
	MasterDataStreamFloor       = shared.MasterDataStreamFloor
	MasterDataStreamCatalog     = shared.MasterDataStreamCatalog
	MasterDataStreamMenu        = shared.MasterDataStreamMenu
	MasterDataStreamPricing     = shared.MasterDataStreamPricing
	MasterDataStreamRecipes     = shared.MasterDataStreamRecipes
	MasterDataStreamInventory   = shared.MasterDataStreamInventory

	SyncEnvelopeVersion = shared.SyncEnvelopeVersion
)

func IsEdgeToCloudOperationalEvent(eventType string) bool {
	return shared.IsEdgeToCloudOperationalEvent(eventType)
}

func IsCloudOwnedAggregate(aggregateType string) bool {
	return shared.IsCloudOwnedAggregate(aggregateType)
}

func DirectionForOutbox(origin CommandOrigin, aggregateType, eventType string) SyncDirection {
	return shared.DirectionForOutbox(origin, aggregateType, eventType)
}

func NewReprintDocument(documentType, sourceID string, snapshot []byte, actorEmployeeID string, reprintedAt time.Time) *ReprintDocument {
	return shared.NewReprintDocument(documentType, sourceID, snapshot, actorEmployeeID, reprintedAt)
}

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

package shared

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// PermissionID задает канонический backend-идентификатор права для app-layer проверки RBAC.
type PermissionID string

const (
	// Идентификаторы прав кассового runtime.
	PermissionEmployeeShiftOpen        PermissionID = "pos.employee_shift.open"
	PermissionEmployeeShiftClose       PermissionID = "pos.employee_shift.close"
	PermissionEmployeeShiftViewCurrent PermissionID = "pos.employee_shift.view_current"
	PermissionEmployeeShiftRecent      PermissionID = "pos.employee_shift.recent"
	PermissionCashSessionOpen          PermissionID = "pos.cash_session.open"
	PermissionCashSessionClose         PermissionID = "pos.cash_session.close"
	PermissionCashSessionViewCurrent   PermissionID = "pos.cash_session.view_current"
	PermissionCashDrawerEvent          PermissionID = "pos.cash_drawer.record_event"
	PermissionCatalogView              PermissionID = "pos.catalog.view"
	PermissionFloorView                PermissionID = "pos.floor.view"
	PermissionMenuView                 PermissionID = "pos.menu.view"
	PermissionOrderCreate              PermissionID = "pos.order.create"
	PermissionOrderView                PermissionID = "pos.order.view"
	PermissionOrderAddLine             PermissionID = "pos.order.add_line"
	PermissionOrderChangeQuantity      PermissionID = "pos.order.change_quantity"
	PermissionOrderVoidLine            PermissionID = "pos.order.void_line"
	PermissionOrderClose               PermissionID = "pos.order.close"
	PermissionPricingView              PermissionID = "pos.pricing.view"
	PermissionPricingDiscountApply     PermissionID = "pos.pricing.discount.apply"
	PermissionPricingSurchargeApply    PermissionID = "pos.pricing.surcharge.apply"
	PermissionPrecheckIssue            PermissionID = "pos.precheck.issue"
	PermissionPrecheckView             PermissionID = "pos.precheck.view"
	PermissionPrecheckReprint          PermissionID = "pos.precheck.reprint"
	// Идентификаторы прав manager override flow.
	PermissionPrecheckCancelRequest PermissionID = "pos.precheck.cancel.request"
	PermissionPrecheckCancel        PermissionID = "pos.precheck.cancel"
	PermissionPaymentCash           PermissionID = "pos.payment.cash"
	PermissionPaymentCardManual     PermissionID = "pos.payment.card.manual"
	PermissionPaymentOther          PermissionID = "pos.payment.other"
	PermissionPaymentRefund         PermissionID = "pos.payment.refund"
	PermissionCheckView             PermissionID = "pos.check.view"
	PermissionCheckReprint          PermissionID = "pos.check.reprint"
	// Идентификаторы прав KDS runtime.
	PermissionKitchenView                PermissionID = "pos.kitchen.view"
	PermissionKitchenStatusChange        PermissionID = "pos.kitchen.status.change"
	PermissionKitchenCatalogView         PermissionID = "pos.kitchen.catalog.view"
	PermissionKitchenRecipeView          PermissionID = "pos.kitchen.recipe.view"
	PermissionKitchenRecipeSuggest       PermissionID = "pos.kitchen.recipe.suggest"
	PermissionKitchenCatalogSuggest      PermissionID = "pos.kitchen.catalog.suggest"
	PermissionKitchenStockReceipt        PermissionID = "pos.kitchen.stock.receipt"
	PermissionKitchenStockInventoryCount PermissionID = "pos.kitchen.stock.inventory_count"
	PermissionKitchenStockWriteOff       PermissionID = "pos.kitchen.stock.write_off"
	PermissionKitchenProductionComplete  PermissionID = "pos.kitchen.production.complete"
	PermissionKitchenStopListUpdate      PermissionID = "pos.kitchen.stop_list.update"
	PermissionSyncView                   PermissionID = "pos.sync.view"
	// Идентификатор права manager/service sync operation.
	PermissionSyncRetryFailed PermissionID = "pos.sync.retry_failed"
)

var knownPermissionIDs = map[PermissionID]struct{}{
	PermissionEmployeeShiftOpen:          {},
	PermissionEmployeeShiftClose:         {},
	PermissionEmployeeShiftViewCurrent:   {},
	PermissionEmployeeShiftRecent:        {},
	PermissionCashSessionOpen:            {},
	PermissionCashSessionClose:           {},
	PermissionCashSessionViewCurrent:     {},
	PermissionCashDrawerEvent:            {},
	PermissionCatalogView:                {},
	PermissionFloorView:                  {},
	PermissionMenuView:                   {},
	PermissionOrderCreate:                {},
	PermissionOrderView:                  {},
	PermissionOrderAddLine:               {},
	PermissionOrderChangeQuantity:        {},
	PermissionOrderVoidLine:              {},
	PermissionOrderClose:                 {},
	PermissionPricingView:                {},
	PermissionPricingDiscountApply:       {},
	PermissionPricingSurchargeApply:      {},
	PermissionPrecheckIssue:              {},
	PermissionPrecheckView:               {},
	PermissionPrecheckReprint:            {},
	PermissionPrecheckCancelRequest:      {},
	PermissionPrecheckCancel:             {},
	PermissionPaymentCash:                {},
	PermissionPaymentCardManual:          {},
	PermissionPaymentOther:               {},
	PermissionPaymentRefund:              {},
	PermissionCheckView:                  {},
	PermissionCheckReprint:               {},
	PermissionKitchenView:                {},
	PermissionKitchenStatusChange:        {},
	PermissionKitchenCatalogView:         {},
	PermissionKitchenRecipeView:          {},
	PermissionKitchenRecipeSuggest:       {},
	PermissionKitchenCatalogSuggest:      {},
	PermissionKitchenStockReceipt:        {},
	PermissionKitchenStockInventoryCount: {},
	PermissionKitchenStockWriteOff:       {},
	PermissionKitchenProductionComplete:  {},
	PermissionKitchenStopListUpdate:      {},
	PermissionSyncView:                   {},
	PermissionSyncRetryFailed:            {},
}

// RoleName задает канонический идентификатор pilot-роли.
type RoleName string

const (
	RoleCashier       RoleName = "cashier"
	RoleSeniorCashier RoleName = "senior_cashier"
	RoleWaiter        RoleName = "waiter"
	RoleManager       RoleName = "manager"
	RoleKitchen       RoleName = "kitchen"
	RoleSupportAdmin  RoleName = "support_admin"
)

// RoleProfile описывает backend-права, выданные канонической pilot-роли.
type RoleProfile struct {
	Name        RoleName
	Permissions []PermissionID
}

var canonicalRoleProfiles = map[RoleName]RoleProfile{
	RoleCashier: {
		Name: RoleCashier,
		Permissions: []PermissionID{
			PermissionEmployeeShiftOpen,
			PermissionEmployeeShiftClose,
			PermissionEmployeeShiftViewCurrent,
			PermissionEmployeeShiftRecent,
			PermissionCashSessionOpen,
			PermissionCashSessionViewCurrent,
			PermissionCatalogView,
			PermissionFloorView,
			PermissionMenuView,
			PermissionOrderCreate,
			PermissionOrderView,
			PermissionOrderAddLine,
			PermissionOrderChangeQuantity,
			PermissionOrderVoidLine,
			PermissionOrderClose,
			PermissionPricingView,
			PermissionPricingDiscountApply,
			PermissionPricingSurchargeApply,
			PermissionPrecheckIssue,
			PermissionPrecheckView,
			PermissionPrecheckReprint,
			PermissionPaymentCash,
			PermissionPaymentCardManual,
			PermissionCheckView,
		},
	},
	RoleSeniorCashier: {
		Name: RoleSeniorCashier,
		Permissions: []PermissionID{
			PermissionEmployeeShiftOpen,
			PermissionEmployeeShiftClose,
			PermissionEmployeeShiftViewCurrent,
			PermissionEmployeeShiftRecent,
			PermissionCashSessionOpen,
			PermissionCashSessionClose,
			PermissionCashSessionViewCurrent,
			PermissionCatalogView,
			PermissionFloorView,
			PermissionMenuView,
			PermissionOrderCreate,
			PermissionOrderView,
			PermissionOrderAddLine,
			PermissionOrderChangeQuantity,
			PermissionOrderVoidLine,
			PermissionOrderClose,
			PermissionPricingView,
			PermissionPricingDiscountApply,
			PermissionPricingSurchargeApply,
			PermissionPrecheckIssue,
			PermissionPrecheckView,
			PermissionPrecheckReprint,
			PermissionPrecheckCancelRequest,
			PermissionPaymentCash,
			PermissionPaymentCardManual,
			PermissionPaymentRefund,
			PermissionCheckView,
			PermissionSyncView,
		},
	},
	RoleWaiter: {
		Name: RoleWaiter,
		Permissions: []PermissionID{
			PermissionEmployeeShiftOpen,
			PermissionEmployeeShiftClose,
			PermissionEmployeeShiftViewCurrent,
			PermissionEmployeeShiftRecent,
			PermissionCatalogView,
			PermissionFloorView,
			PermissionMenuView,
			PermissionOrderCreate,
			PermissionOrderView,
			PermissionOrderAddLine,
			PermissionOrderChangeQuantity,
			PermissionOrderVoidLine,
			PermissionOrderClose,
			PermissionPricingView,
			PermissionPrecheckIssue,
			PermissionPrecheckView,
			PermissionPrecheckReprint,
			PermissionCheckView,
		},
	},
	RoleManager: {
		Name: RoleManager,
		Permissions: []PermissionID{
			PermissionEmployeeShiftOpen,
			PermissionEmployeeShiftClose,
			PermissionEmployeeShiftViewCurrent,
			PermissionEmployeeShiftRecent,
			PermissionCashSessionOpen,
			PermissionCashSessionClose,
			PermissionCashSessionViewCurrent,
			PermissionCashDrawerEvent,
			PermissionCatalogView,
			PermissionFloorView,
			PermissionMenuView,
			PermissionOrderCreate,
			PermissionOrderView,
			PermissionOrderAddLine,
			PermissionOrderChangeQuantity,
			PermissionOrderVoidLine,
			PermissionOrderClose,
			PermissionPricingView,
			PermissionPricingDiscountApply,
			PermissionPricingSurchargeApply,
			PermissionPrecheckIssue,
			PermissionPrecheckView,
			PermissionPrecheckReprint,
			PermissionPrecheckCancelRequest,
			PermissionPrecheckCancel,
			PermissionPaymentCash,
			PermissionPaymentCardManual,
			PermissionPaymentOther,
			PermissionPaymentRefund,
			PermissionCheckView,
			PermissionCheckReprint,
			PermissionSyncView,
			PermissionSyncRetryFailed,
		},
	},
	RoleKitchen: {
		Name: RoleKitchen,
		Permissions: []PermissionID{
			PermissionEmployeeShiftViewCurrent,
			PermissionCatalogView,
			PermissionKitchenView,
			PermissionKitchenStatusChange,
			PermissionKitchenCatalogView,
			PermissionKitchenRecipeView,
			PermissionKitchenRecipeSuggest,
			PermissionKitchenCatalogSuggest,
			PermissionKitchenStockReceipt,
			PermissionKitchenStockInventoryCount,
			PermissionKitchenStockWriteOff,
			PermissionKitchenProductionComplete,
			PermissionKitchenStopListUpdate,
		},
	},
	RoleSupportAdmin: {
		Name: RoleSupportAdmin,
		Permissions: []PermissionID{
			PermissionSyncView,
			PermissionSyncRetryFailed,
		},
	},
}

// PermissionsJSON кодирует детерминированный JSON-объект прав для role seeds и тестов.
func PermissionsJSON(permissions ...PermissionID) string {
	if len(permissions) == 0 {
		return "{}"
	}
	seen := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		key := strings.TrimSpace(string(permission))
		if key == "" {
			continue
		}
		seen[key] = struct{}{}
	}
	if len(seen) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteByte('{')
	for i, key := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(key)
		b.WriteString(`":true`)
	}
	b.WriteByte('}')
	return b.String()
}

// CanonicalRoleProfiles возвращает детерминированные snapshots профилей прав pilot-ролей.
func CanonicalRoleProfiles() []RoleProfile {
	names := []string{
		string(RoleCashier),
		string(RoleSeniorCashier),
		string(RoleWaiter),
		string(RoleManager),
		string(RoleKitchen),
		string(RoleSupportAdmin),
	}
	out := make([]RoleProfile, 0, len(names))
	for _, name := range names {
		profile := canonicalRoleProfiles[RoleName(name)]
		profile.Permissions = slices.Clone(profile.Permissions)
		out = append(out, profile)
	}
	return out
}

// RoleProfileByName возвращает канонический профиль прав для pilot-роли.
func RoleProfileByName(name RoleName) (RoleProfile, bool) {
	profile, ok := canonicalRoleProfiles[name]
	if !ok {
		return RoleProfile{}, false
	}
	profile.Permissions = slices.Clone(profile.Permissions)
	return profile, true
}

// RolePermissionsJSON кодирует канонический профиль роли в permissions JSON.
func RolePermissionsJSON(name RoleName) string {
	profile, ok := RoleProfileByName(name)
	if !ok {
		return "{}"
	}
	return PermissionsJSON(profile.Permissions...)
}

// HasAnyPermission возвращает true, если role permissions JSON содержит хотя бы одно из прав.
func HasAnyPermission(body string, permissions ...PermissionID) bool {
	for _, permission := range permissions {
		if HasPermission(body, string(permission)) {
			return true
		}
	}
	return false
}

// IsKnownPermissionID сообщает, входит ли право в канонический backend-каталог.
func IsKnownPermissionID(permission PermissionID) bool {
	_, ok := knownPermissionIDs[permission]
	return ok
}

// ValidatePermissionsJSON проверяет, что все выданные права входят в канонический backend-каталог.
func ValidatePermissionsJSON(body string) error {
	for _, permission := range PermissionsFromJSON(body) {
		if !IsKnownPermissionID(PermissionID(permission)) {
			return fmt.Errorf("unknown permission id %q", permission)
		}
	}
	return nil
}

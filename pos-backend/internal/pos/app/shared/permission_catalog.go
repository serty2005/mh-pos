package shared

import (
	"sort"
	"strings"
)

// PermissionID is a canonical backend permission identifier used for app-layer RBAC enforcement.
type PermissionID string

const (
	// Cashier runtime permission ids.
	PermissionShiftOpen              PermissionID = "pos.shift.open"
	PermissionShiftClose             PermissionID = "pos.shift.close"
	PermissionShiftViewCurrent       PermissionID = "pos.shift.view_current"
	PermissionShiftRecent            PermissionID = "pos.shift.recent"
	PermissionCashSessionOpen        PermissionID = "pos.cash_session.open"
	PermissionCashSessionClose       PermissionID = "pos.cash_session.close"
	PermissionCashSessionViewCurrent PermissionID = "pos.cash_session.view_current"
	PermissionCashDrawerEvent        PermissionID = "pos.cash_drawer.record_event"
	PermissionOrderCreate            PermissionID = "pos.order.create"
	PermissionOrderView              PermissionID = "pos.order.view"
	PermissionOrderAddLine           PermissionID = "pos.order.add_line"
	PermissionOrderChangeQuantity    PermissionID = "pos.order.change_quantity"
	PermissionOrderVoidLine          PermissionID = "pos.order.void_line"
	PermissionPrecheckIssue          PermissionID = "pos.precheck.issue"
	PermissionPrecheckView           PermissionID = "pos.precheck.view"
	// Manager override flow permission ids.
	PermissionPrecheckCancelRequest PermissionID = "pos.precheck.cancel.request"
	PermissionPrecheckCancel        PermissionID = "pos.precheck.cancel"
	PermissionPaymentCapture        PermissionID = "pos.payment.capture"
	PermissionCheckView             PermissionID = "pos.check.view"
	PermissionSyncView              PermissionID = "pos.sync.view"
	// Manager/service sync operation permission id.
	PermissionSyncRetryFailed PermissionID = "pos.sync.retry_failed"
)

// PermissionsJSON encodes a deterministic permissions JSON object for role seeds and tests.
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

// HasAnyPermission returns true when at least one permission is granted by role permissions JSON.
func HasAnyPermission(body string, permissions ...PermissionID) bool {
	for _, permission := range permissions {
		if HasPermission(body, string(permission)) {
			return true
		}
	}
	return false
}

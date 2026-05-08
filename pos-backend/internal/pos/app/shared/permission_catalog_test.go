package shared_test

import (
	"testing"

	"pos-backend/internal/pos/app/shared"
)

func TestPermissionsJSONDeterministicAndDeduplicated(t *testing.T) {
	got := shared.PermissionsJSON(
		shared.PermissionOrderAddLine,
		shared.PermissionOrderCreate,
		shared.PermissionOrderAddLine,
	)
	want := `{"pos.order.add_line":true,"pos.order.create":true}`
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestHasAnyPermission(t *testing.T) {
	body := shared.PermissionsJSON(shared.PermissionEmployeeShiftOpen, shared.PermissionPrecheckIssue)
	if !shared.HasAnyPermission(body, shared.PermissionPrecheckIssue, shared.PermissionPaymentCash) {
		t.Fatal("expected one of requested permissions to be granted")
	}
	if shared.HasAnyPermission(body, shared.PermissionPaymentCash, shared.PermissionSyncRetryFailed) {
		t.Fatal("expected no requested permissions to be granted")
	}
}

func TestCanonicalRoleProfiles(t *testing.T) {
	profiles := shared.CanonicalRoleProfiles()
	if len(profiles) != 6 {
		t.Fatalf("expected six pilot role profiles, got %d", len(profiles))
	}
	cashier, ok := shared.RoleProfileByName(shared.RoleCashier)
	if !ok {
		t.Fatal("expected cashier role profile")
	}
	if !shared.HasPermission(shared.PermissionsJSON(cashier.Permissions...), string(shared.PermissionPaymentCash)) {
		t.Fatal("expected cashier to support cash payment")
	}
	if shared.HasPermission(shared.PermissionsJSON(cashier.Permissions...), string(shared.PermissionCashSessionClose)) {
		t.Fatal("expected cashier not to close cash session directly")
	}
	waiter, ok := shared.RoleProfileByName(shared.RoleWaiter)
	if !ok {
		t.Fatal("expected waiter role profile")
	}
	waiterPermissions := shared.PermissionsJSON(waiter.Permissions...)
	if shared.HasAnyPermission(waiterPermissions, shared.PermissionPaymentCash, shared.PermissionPaymentCardManual, shared.PermissionPaymentOther) {
		t.Fatal("expected waiter payment to stay out of scope")
	}
	if !shared.HasPermission(waiterPermissions, string(shared.PermissionPrecheckReprint)) {
		t.Fatal("expected waiter to reprint prechecks from immutable snapshot")
	}
	manager, ok := shared.RoleProfileByName(shared.RoleManager)
	if !ok {
		t.Fatal("expected manager role profile")
	}
	if !shared.HasPermission(shared.PermissionsJSON(manager.Permissions...), string(shared.PermissionPrecheckCancel)) {
		t.Fatal("expected manager to approve precheck cancel")
	}
	if !shared.HasPermission(shared.PermissionsJSON(manager.Permissions...), string(shared.PermissionCheckReprint)) {
		t.Fatal("expected manager to reprint final checks")
	}
	if !shared.HasPermission(shared.RolePermissionsJSON(shared.RoleSupportAdmin), string(shared.PermissionSyncRetryFailed)) {
		t.Fatal("expected support admin to retry failed syncs")
	}
}

func TestValidatePermissionsJSON(t *testing.T) {
	valid := shared.PermissionsJSON(shared.PermissionFloorView, shared.PermissionOrderCreate)
	if err := shared.ValidatePermissionsJSON(valid); err != nil {
		t.Fatalf("expected valid canonical permissions json, got %v", err)
	}
	invalid := `{"pos.order.create":true,"pos.unknown.permission":true}`
	if err := shared.ValidatePermissionsJSON(invalid); err == nil {
		t.Fatal("expected unknown permission to be rejected")
	}
}

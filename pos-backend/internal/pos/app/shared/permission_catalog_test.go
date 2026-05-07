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
	body := shared.PermissionsJSON(shared.PermissionShiftOpen, shared.PermissionPrecheckIssue)
	if !shared.HasAnyPermission(body, shared.PermissionPrecheckIssue, shared.PermissionPaymentCapture) {
		t.Fatal("expected one of requested permissions to be granted")
	}
	if shared.HasAnyPermission(body, shared.PermissionPaymentCapture, shared.PermissionSyncRetryFailed) {
		t.Fatal("expected no requested permissions to be granted")
	}
}

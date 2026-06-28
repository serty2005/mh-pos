package version

import "testing"

func TestDefaultProductVersionBumpsPOSRuntimeAfterBaselineChange(t *testing.T) {
	if DefaultProductVersion != "0.1.9" {
		t.Fatalf("expected POS runtime version 0.1.9 after receipt_printers baseline change, got %s", DefaultProductVersion)
	}
}

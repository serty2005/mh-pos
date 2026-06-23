package version

import "testing"

func TestDefaultProductVersionBumpsPOSRuntimeAfterBaselineChange(t *testing.T) {
	if DefaultProductVersion != "0.1.6" {
		t.Fatalf("expected POS runtime version 0.1.6 after ticket_units baseline change, got %s", DefaultProductVersion)
	}
}

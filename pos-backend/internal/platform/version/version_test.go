package version

import "testing"

func TestDefaultProductVersionBumpsPOSRuntimeAfterBaselineChange(t *testing.T) {
	if DefaultProductVersion != "0.1.4" {
		t.Fatalf("expected POS runtime version 0.1.4 after managed baseline checksum change, got %s", DefaultProductVersion)
	}
}

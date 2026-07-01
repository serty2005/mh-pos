package version

import "testing"

func TestDefaultProductVersionBumpsPOSRuntimeAfterBaselineChange(t *testing.T) {
	if DefaultProductVersion != "0.1.11" {
		t.Fatalf("expected POS runtime version 0.1.11 after POS-86 print routing baseline change, got %s", DefaultProductVersion)
	}
}

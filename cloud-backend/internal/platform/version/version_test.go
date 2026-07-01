package version

import "testing"

func TestDefaultProductVersionBumpsCloudRuntimeAfterBaselineChange(t *testing.T) {
	if DefaultProductVersion != "0.1.16" {
		t.Fatalf("expected cloud runtime version 0.1.16 after POS-86 master-data baseline change, got %s", DefaultProductVersion)
	}
}

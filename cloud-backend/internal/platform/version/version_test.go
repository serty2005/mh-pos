package version

import "testing"

func TestDefaultProductVersionBumpsCloudRuntimeAfterBaselineChange(t *testing.T) {
	if DefaultProductVersion != "0.1.6" {
		t.Fatalf("expected cloud runtime version 0.1.6 after managed baseline checksum change, got %s", DefaultProductVersion)
	}
}

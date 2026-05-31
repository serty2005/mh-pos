package version

import "testing"

func TestDefaultProductVersionBumpsCloudRuntimeAfterBaselineChange(t *testing.T) {
	if DefaultProductVersion != "0.1.7" {
		t.Fatalf("expected cloud runtime version 0.1.7 after managed baseline checksum change, got %s", DefaultProductVersion)
	}
}

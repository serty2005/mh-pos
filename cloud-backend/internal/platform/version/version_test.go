package version

import "testing"

func TestDefaultProductVersionBumpsCloudRuntimeAfterBaselineChange(t *testing.T) {
	if DefaultProductVersion != "0.1.9" {
		t.Fatalf("expected cloud runtime version 0.1.9 after pairing consume baseline change, got %s", DefaultProductVersion)
	}
}

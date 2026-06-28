package version

import "testing"

func TestDefaultProductVersionBumpsCloudRuntimeAfterBaselineChange(t *testing.T) {
	if DefaultProductVersion != "0.1.15" {
		t.Fatalf("expected cloud runtime version 0.1.15 after printers stream baseline change, got %s", DefaultProductVersion)
	}
}

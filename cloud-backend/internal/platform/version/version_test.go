package version

import "testing"

func TestDefaultProductVersionBumpsCloudRuntimeAfterBaselineChange(t *testing.T) {
	if DefaultProductVersion != "0.1.12" {
		t.Fatalf("expected cloud runtime version 0.1.12 after tenant catalog foundation baseline change, got %s", DefaultProductVersion)
	}
}

package version

import "testing"

func TestDefaultProductVersionBumpsCloudRuntimeAfterBaselineChange(t *testing.T) {
	if DefaultProductVersion != "0.1.8" {
		t.Fatalf("expected cloud runtime version 0.1.8 after assignment audit baseline change, got %s", DefaultProductVersion)
	}
}

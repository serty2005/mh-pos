package version

import "testing"

func TestDefaultProductVersionBumpsPOSRuntimeAfterBaselineChange(t *testing.T) {
	if DefaultProductVersion != "0.1.5" {
		t.Fatalf("expected POS runtime version 0.1.5 after proposal_feedback baseline change, got %s", DefaultProductVersion)
	}
}

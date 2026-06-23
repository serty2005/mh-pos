package version

import "testing"

func TestDefaultProductVersionBumpsCloudRuntimeAfterBaselineChange(t *testing.T) {
	if DefaultProductVersion != "0.1.13" {
		t.Fatalf("expected cloud runtime version 0.1.13 after TicketIssued event_type baseline change, got %s", DefaultProductVersion)
	}
}

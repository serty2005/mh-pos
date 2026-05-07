package postgres

import "testing"

func TestCompareModuleVersion(t *testing.T) {
	result, err := compareModuleVersion("0.1.0", "0.2.0")
	if err != nil {
		t.Fatalf("compare failed: %v", err)
	}
	if result >= 0 {
		t.Fatalf("expected 0.1.0 < 0.2.0, got %d", result)
	}
	if _, err := compareModuleVersion("broken", "0.2.0"); err == nil {
		t.Fatal("expected invalid semantic version to fail")
	}
}

func TestShouldUpgradeVersion(t *testing.T) {
	needsUpgrade, err := shouldUpgradeVersion("0.1.0", "0.1.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !needsUpgrade {
		t.Fatal("expected upgrade when runtime version is lower")
	}
	needsUpgrade, err = shouldUpgradeVersion("0.1.1", "0.1.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if needsUpgrade {
		t.Fatal("expected no upgrade when versions are equal")
	}
}

func TestSanitizeFilenameToken(t *testing.T) {
	if got := sanitizeFilenameToken(" cloud backend / 0.1.0 "); got != "cloud_backend___0.1.0" {
		t.Fatalf("unexpected sanitized token %q", got)
	}
}

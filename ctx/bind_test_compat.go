package ctx

import "testing"

// Enable standard library JSON for test compatibility
// This ensures that the bind tests pass with exact standard library behavior
func init() {
	setTestCompatibilityMode(true)
}

// TestJSONCompatibilityEnabled verifies that test compatibility mode is active
func TestJSONCompatibilityEnabled(t *testing.T) {
	if !useStandardJSONForTests {
		t.Fatal("Test compatibility mode should be enabled")
	}
}

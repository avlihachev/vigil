package switcher

import "testing"

func TestActivateSession_DoesNotPanic(t *testing.T) {
	// just verify it doesn't panic with various inputs
	// actual activation requires running apps, so we just check no crash
	_ = ActivateSession("VSCode", "/tmp/nonexistent")
	_ = ActivateSession("Terminal", "/tmp/nonexistent")
	_ = ActivateSession("Cursor", "/tmp/nonexistent")
}

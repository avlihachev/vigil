package switcher

import "testing"

func TestActivateSession_DoesNotPanic(t *testing.T) {
	// just verify it doesn't panic with various inputs
	// actual activation requires running apps, so we just check no crash
	_ = ActivateSession("VSCode", "/tmp/nonexistent", 0)
	_ = ActivateSession("Terminal", "/tmp/nonexistent", 0)
	_ = ActivateSession("Cursor", "/tmp/nonexistent", 0)
}

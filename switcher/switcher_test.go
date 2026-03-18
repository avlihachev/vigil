package switcher

import "testing"

func TestBuildAppleScript_VSCode(t *testing.T) {
	script := buildActivateScript("VSCode", "/Users/test/proj")
	if script == "" {
		t.Error("expected non-empty script")
	}
}

func TestBuildAppleScript_Terminal(t *testing.T) {
	script := buildActivateScript("Terminal", "/Users/test/proj")
	if script == "" {
		t.Error("expected non-empty script")
	}
}

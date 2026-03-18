package switcher

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func ActivateSession(source string, cwd string) error {
	folderName := filepath.Base(cwd)
	switch source {
	case "VSCode":
		// System Events process name is "Code"; app name for fallback activate is "Visual Studio Code"
		return activateApp("Code", "Visual Studio Code", folderName, func() error {
			return exec.Command("code", cwd).Run()
		})
	case "Cursor":
		return activateApp("Cursor", "Cursor", folderName, func() error {
			return exec.Command("cursor", cwd).Run()
		})
	default:
		// source is the terminal display name; proc name matches in System Events
		proc := terminalProcessName(source)
		return activateApp(proc, proc, folderName, nil)
	}
}

// activateApp raises the target app window. Strategy:
//  1. System Events AXRaise on the best matching window (needs Accessibility permission).
//  2. If permission denied → fall back to `tell application X to activate` (no permissions needed).
//  3. If app not running → call openFallback (e.g. open a new window).
func activateApp(procName, appName, folderName string, openFallback func() error) error {
	// AppleScript `is` comparison is case-insensitive by default,
	// so procName can be any case and will still match the running process.
	script := fmt.Sprintf(`
tell application "System Events"
	set targetProc to missing value
	repeat with p in (every process)
		if name of p is "%s" then
			set targetProc to p
			exit repeat
		end if
	end repeat
	if targetProc is missing value then
		return "not_running"
	end if
	tell targetProc
		set didRaise to false
		repeat with w in windows
			if name of w contains "%s" then
				perform action "AXRaise" of w
				set frontmost to true
				set didRaise to true
				exit repeat
			end if
		end repeat
		if not didRaise then
			set frontmost to true
			if (count of windows) > 0 then
				perform action "AXRaise" of window 1
			end if
		end if
	end tell
end tell
return "done"`, procName, folderName)

	out, err := exec.Command("osascript", "-e", script).Output()
	result := strings.TrimSpace(string(out))

	if err == nil && result == "done" {
		return nil
	}

	if result == "not_running" {
		if openFallback != nil {
			return openFallback()
		}
		return fmt.Errorf("%s is not running", appName)
	}

	// Accessibility permission denied or any other error — simple activate (always works)
	simpleScript := fmt.Sprintf(`tell application "%s" to activate`, appName)
	return exec.Command("osascript", "-e", simpleScript).Run()
}

// terminalProcessName maps display name → System Events process name (matches CFBundleName).
func terminalProcessName(source string) string {
	m := map[string]string{
		"Ghostty":   "Ghostty",
		"Terminal":  "Terminal",
		"iTerm2":    "iTerm2",
		"Warp":      "Warp",
		"Alacritty": "Alacritty",
		"Kitty":     "Kitty",
	}
	if p, ok := m[source]; ok {
		return p
	}
	return source
}

package switcher

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// cwdToFileURL converts "/Users/foo" to "file:///Users/foo/" for AXDocument matching
func cwdToFileURL(cwd string) string {
	u := url.URL{Scheme: "file", Path: cwd}
	s := u.String()
	if !strings.HasSuffix(s, "/") {
		s += "/"
	}
	return s
}

func findCodeCLI() string {
	paths := []string{
		"/usr/local/bin/code",
		"/opt/homebrew/bin/code",
		"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("code"); err == nil {
		return p
	}
	return ""
}

func findCursorCLI() string {
	paths := []string{
		"/usr/local/bin/cursor",
		"/opt/homebrew/bin/cursor",
		"/Applications/Cursor.app/Contents/Resources/app/bin/cursor",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("cursor"); err == nil {
		return p
	}
	return ""
}

func ActivateSession(source string, cwd string, pid int) error {
	folderName := filepath.Base(cwd)
	cwdURL := cwdToFileURL(cwd)

	switch source {
	case "VSCode":
		if activateByAX("Code", "Visual Studio Code", cwdURL, folderName) == nil {
			return nil
		}
		// try workspace name from .code-workspace file
		if wsName := findWorkspaceName(cwd); wsName != "" {
			if activateByAX("Code", "Visual Studio Code", "", wsName) == nil {
				return nil
			}
		}
		if codePath := findCodeCLI(); codePath != "" {
			exec.Command(codePath, cwd).Run()
			exec.Command("open", "-a", "Visual Studio Code").Run()
		}
		return nil
	case "Cursor":
		if activateByAX("Cursor", "Cursor", cwdURL, folderName) == nil {
			return nil
		}
		if wsName := findWorkspaceName(cwd); wsName != "" {
			if activateByAX("Cursor", "Cursor", "", wsName) == nil {
				return nil
			}
		}
		if cursorPath := findCursorCLI(); cursorPath != "" {
			exec.Command(cursorPath, cwd).Run()
			exec.Command("open", "-a", "Cursor").Run()
		}
		return nil
	default:
		proc := terminalProcessName(source)
		return activateByAX(proc, source, cwdURL, folderName)
	}
}

// activateByAX uses the Accessibility C API to find and raise a window.
// docMatch is checked against AXDocument; titleMatch against window title.
func activateByAX(procName, appName, docMatch, titleMatch string) error {
	appPID := findAppPID(procName)
	if appPID == 0 {
		return fmt.Errorf("%s is not running", appName)
	}

	result := raiseWindow(appPID, docMatch, titleMatch)
	if result > 0 {
		return nil
	}
	if result == 0 {
		return fmt.Errorf("window not found")
	}

	// result == -1: no accessibility — bring app to front at least
	exec.Command("open", "-a", appName).Run()
	return nil
}

// findWorkspaceName walks up from cwd looking for .code-workspace files
// and returns the workspace name (filename without extension)
func findWorkspaceName(cwd string) string {
	dir := cwd
	for {
		matches, _ := filepath.Glob(filepath.Join(dir, "*.code-workspace"))
		if len(matches) > 0 {
			base := filepath.Base(matches[0])
			return strings.TrimSuffix(base, ".code-workspace")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func isTerminalComm(comm string) bool {
	switch comm {
	case "ghostty", "iterm2", "terminal", "warp", "alacritty", "kitty":
		return true
	}
	return false
}

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

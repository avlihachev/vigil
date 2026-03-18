package monitor

import (
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// terminalForPID walks up the process tree from pid to find the terminal emulator.
// Typical chain: ghostty → zsh → claude
func terminalForPID(pid int) string {
	current := pid
	for range 6 {
		out, err := exec.Command("ps", "-p", strconv.Itoa(current), "-o", "ppid=,comm=").Output()
		if err != nil {
			break
		}
		fields := strings.Fields(strings.TrimSpace(string(out)))
		if len(fields) < 2 {
			break
		}
		ppid, _ := strconv.Atoi(fields[0])
		comm := strings.ToLower(filepath.Base(fields[1]))
		switch comm {
		case "ghostty":
			return "Ghostty"
		case "iterm2":
			return "iTerm2"
		case "terminal":
			return "Terminal"
		case "warp":
			return "Warp"
		case "alacritty":
			return "Alacritty"
		case "kitty":
			return "Kitty"
		}
		if ppid <= 1 {
			break
		}
		current = ppid
	}
	return "Terminal"
}

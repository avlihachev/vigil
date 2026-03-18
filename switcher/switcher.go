package switcher

import (
	"fmt"
	"os/exec"
)

func ActivateSession(source string, cwd string) error {
	script := buildActivateScript(source, cwd)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

func buildActivateScript(source string, cwd string) string {
	switch source {
	case "VSCode":
		return fmt.Sprintf(`tell application "Visual Studio Code" to activate`)
	case "Cursor":
		return fmt.Sprintf(`tell application "Cursor" to activate`)
	default:
		return fmt.Sprintf(`tell application "Terminal" to activate`)
	}
}

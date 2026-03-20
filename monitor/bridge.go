package monitor

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const bridgeScript = `#!/bin/bash
input=$(cat)
echo "$input" | jq '{
  five_hour: .rate_limits.five_hour,
  seven_day: .rate_limits.seven_day,
  updated_at: (now | todate)
}' > ~/.vigil/rate-limits.json 2>/dev/null
`

func InstallBridge(vigilDir string) error {
	path := filepath.Join(vigilDir, "statusline-bridge.sh")
	if err := os.WriteFile(path, []byte(bridgeScript), 0755); err != nil {
		return err
	}
	return nil
}

func EnableStatusLine(claudeDir, vigilDir string) error {
	settingsPath := filepath.Join(claudeDir, "settings.json")
	bridgePath := filepath.Join(vigilDir, "statusline-bridge.sh")

	settings := make(map[string]interface{})
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		json.Unmarshal(data, &settings)
	}

	settings["statusLine"] = map[string]interface{}{
		"type":    "command",
		"command": bridgePath,
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, out, 0644)
}

func DisableStatusLine(claudeDir string) error {
	settingsPath := filepath.Join(claudeDir, "settings.json")

	settings := make(map[string]interface{})
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil
	}
	json.Unmarshal(data, &settings)
	delete(settings, "statusLine")

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, out, 0644)
}

func IsBridgeInstalled(claudeDir, vigilDir string) bool {
	settingsPath := filepath.Join(claudeDir, "settings.json")
	bridgePath := filepath.Join(vigilDir, "statusline-bridge.sh")

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return false
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return false
	}

	sl, ok := settings["statusLine"].(map[string]interface{})
	if !ok {
		return false
	}
	cmd, _ := sl["command"].(string)
	return cmd == bridgePath
}

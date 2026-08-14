package installer

import (
	"os"
	"path/filepath"
	"strconv"
)

const codexOwner = "codex"

func BuildCodexMCPPlan(home string) (Plan, error) {
	executable, err := os.Executable()
	if err != nil {
		return Plan{}, err
	}
	path := filepath.Join(home, ".codex", "config.toml")
	before, _, _, err := readCurrent(path)
	if err != nil {
		return Plan{}, err
	}
	after := AppendMarkedBlock(before, codexOwner, []byte("[mcp_servers.agent_doctor]\ncommand = "+strconv.Quote(executable)+"\nargs = [\"mcp\", \"serve\"]"))
	detected, err := DetectClients(home, runtimeTargetOS())
	if err != nil {
		return Plan{}, err
	}
	plan, err := BuildPlan(home, detected, []Change{{Path: path, Before: before, After: after, Mode: 0o600}})
	if err != nil {
		return Plan{}, err
	}
	plan.Warnings = append(plan.Warnings, "Only the Codex MCP block is changed automatically; other adapters remain available as explicit templates and plugins.")
	return plan, nil
}

func UninstallCodexMCP(home string) error {
	return UninstallMarkedBlock(filepath.Join(home, ".codex", "config.toml"), codexOwner)
}

var runtimeTargetOS = func() string {
	if value := os.Getenv("AGENT_DOCTOR_TARGET_OS"); value != "" {
		return value
	}
	return runtimeOS()
}

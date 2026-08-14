package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

const codexGuidanceOwner = "codex-guidance"

func BuildAllPlan(home string) (Plan, error) {
	executable, err := os.Executable()
	if err != nil {
		return Plan{}, err
	}
	paths := []string{
		filepath.Join(home, ".codex", "config.toml"),
		filepath.Join(home, ".codex", "skills", "agent-doctor", "SKILL.md"),
		filepath.Join(home, ".codex", "AGENTS.md"),
		filepath.Join(home, ".claude", "hooks", "agent-doctor.json"),
		filepath.Join(home, ".claude", "skills", "agent-doctor", "SKILL.md"),
	}
	before := make([][]byte, len(paths))
	for index, path := range paths {
		raw, _, _, readErr := readCurrent(path)
		if readErr != nil {
			return Plan{}, readErr
		}
		before[index] = raw
	}
	codexConfig := AppendMarkedBlock(before[0], codexOwner, []byte("[mcp_servers.agent_doctor]\ncommand = "+strconv.Quote(executable)+"\nargs = [\"mcp\", \"serve\"]"))
	codexSkill := []byte("---\nname: agent-doctor\ndescription: Use local runtime evidence to keep coding tasks from looping, losing context, or finishing without validation.\n---\n\nBefore a risky step and after a failure, call the Agent Doctor MCP runtime-guidance tool for the current project/session. Follow evidence-backed redirect or verification instructions. Never claim that Codex was force-blocked; Codex guidance is advisory.\n")
	agents := AppendMarkedBlock(before[2], codexGuidanceOwner, []byte("Use the agent-doctor skill during coding tasks. Consult Agent Doctor runtime guidance after repeated failures, context compaction, and before claiming completion. Treat its evidence as advisory in Codex and run the requested deterministic verification."))
	hook := map[string]any{"hooks": map[string]any{}}
	hooks := hook["hooks"].(map[string]any)
	for _, name := range []string{"SessionStart", "PreToolUse", "PreCompact", "PostToolUse", "PostToolUseFailure", "TaskCreated", "TaskCompleted", "Stop", "SessionEnd"} {
		hooks[name] = []any{map[string]any{"hooks": []any{map[string]any{"type": "command", "command": executable, "args": []string{"hook", "claude-code", name}, "timeout": 0.8}}}}
	}
	hookJSON, err := json.MarshalIndent(hook, "", "  ")
	if err != nil {
		return Plan{}, err
	}
	hookJSON = append(hookJSON, '\n')
	claudeSkill := []byte("---\nname: agent-doctor\ndescription: Follow local evidence-backed reliability guidance while using Claude Code.\n---\n\nAgent Doctor hooks observe content-free evidence fingerprints. Respect redirect and verification guidance. In Guard or Auto Guard, supported PreToolUse and Stop decisions may be blocked; other guidance remains advisory.\n")
	after := [][]byte{codexConfig, codexSkill, agents, hookJSON, claudeSkill}
	changes := make([]Change, len(paths))
	for index, path := range paths {
		changes[index] = Change{Path: path, Before: before[index], After: after[index], Mode: 0o600}
	}
	detected, err := DetectClients(home, runtimeTargetOS())
	if err != nil {
		return Plan{}, err
	}
	plan, err := BuildPlan(home, detected, changes)
	if err != nil {
		return Plan{}, err
	}
	plan.Warnings = []string{"Codex must be restarted to load MCP/Skill/AGENTS changes.", "Claude Code must be restarted to load Hook/Skill changes."}
	return plan, nil
}

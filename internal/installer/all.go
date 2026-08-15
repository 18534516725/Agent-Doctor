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
	codexSkill := []byte("---\nname: agent-doctor\ndescription: Use local runtime evidence to keep coding tasks from looping, losing context, or finishing without validation.\n---\n\nAt the start of every coding turn, call the Agent Doctor MCP get_runtime_guidance tool with projectId set to the current project ID (normally the current working directory). Call again after a failed tool step, context compaction, and before completion. Follow evidence-backed advise, redirect, block, or verification instructions before continuing. When non-continue guidance changes the work, briefly tell the user `Agent Doctor intervened: <action taken>`. Treat continue as silence. Codex MCP guidance cannot force-block tool calls; never claim it did.\n")
	agents := AppendMarkedBlock(before[2], codexGuidanceOwner, []byte("Use the agent-doctor skill during coding tasks. At the start of every coding turn call get_runtime_guidance with projectId set to the current project ID; call it again after a failed tool step, context compaction, and before completion. Follow non-continue guidance before continuing and state `Agent Doctor intervened: <action taken>` when it changes the work. Continue results stay silent. Codex MCP advice cannot force-block tool calls; run requested deterministic verification before claiming completion."))
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

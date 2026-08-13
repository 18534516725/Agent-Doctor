package clineadapter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCrossPlatformHookWrappersAreFailOpen(t *testing.T) {
	events := []string{"TaskStart", "TaskResume", "TaskCancel", "TaskComplete", "UserPromptSubmit", "PreCompact", "PreToolUse", "PostToolUse"}
	for _, eventName := range events {
		unix := read(t, filepath.Join("hooks", eventName))
		if !strings.Contains(unix, "agent-doctor hook cline "+eventName) || !strings.Contains(unix, "exit 0") || strings.Contains(unix, "set -e") {
			t.Fatalf("unsafe unix hook %s: %s", eventName, unix)
		}
		windows := read(t, filepath.Join("hooks", eventName+".ps1"))
		if !strings.Contains(windows, "agent-doctor hook cline "+eventName) || !strings.Contains(windows, "exit 0") || !strings.Contains(windows, "$ErrorActionPreference = 'Continue'") {
			t.Fatalf("unsafe PowerShell hook %s: %s", eventName, windows)
		}
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

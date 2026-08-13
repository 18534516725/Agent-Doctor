package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Client struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Detected    bool     `json:"detected"`
	ConfigPaths []string `json:"configPaths"`
}

var knownClients = []struct{ id, name string }{
	{"codex", "Codex"},
	{"claude-code", "Claude Code"},
	{"cursor", "Cursor"},
	{"windsurf", "Windsurf"},
	{"cline", "Cline"},
	{"roo-code", "Roo Code"},
	{"continue", "Continue"},
	{"opencode", "OpenCode"},
	{"aider", "Aider"},
	{"cherry-studio", "Cherry Studio"},
}

func DetectClients(home, targetOS string) ([]Client, error) {
	paths, err := ClientConfigPaths(home, targetOS)
	if err != nil {
		return nil, err
	}
	result := make([]Client, 0, len(knownClients))
	for _, known := range knownClients {
		path := filepath.FromSlash(paths[known.id])
		result = append(result, Client{
			ID: known.id, Name: known.name, Detected: configOrClientDirectoryExists(path),
			ConfigPaths: []string{path},
		})
	}
	return result, nil
}

func ClientConfigPaths(home, targetOS string) (map[string]string, error) {
	if strings.TrimSpace(home) == "" {
		return nil, fmt.Errorf("home directory is required")
	}
	if targetOS != "darwin" && targetOS != "linux" && targetOS != "windows" {
		return nil, fmt.Errorf("unsupported operating system %q", targetOS)
	}
	codeRoot := portableJoin(home, ".config", "Code", "User", "globalStorage")
	cherryRoot := portableJoin(home, ".config", "CherryStudio")
	if targetOS == "darwin" {
		codeRoot = portableJoin(home, "Library", "Application Support", "Code", "User", "globalStorage")
		cherryRoot = portableJoin(home, "Library", "Application Support", "CherryStudio")
	} else if targetOS == "windows" {
		codeRoot = portableJoin(home, "AppData", "Roaming", "Code", "User", "globalStorage")
		cherryRoot = portableJoin(home, "AppData", "Roaming", "CherryStudio")
	}
	return map[string]string{
		"codex":         portableJoin(home, ".codex", "config.toml"),
		"claude-code":   portableJoin(home, ".claude", "settings.json"),
		"cursor":        portableJoin(home, ".cursor", "mcp.json"),
		"windsurf":      portableJoin(home, ".codeium", "windsurf", "mcp_config.json"),
		"cline":         portableJoin(codeRoot, "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"),
		"roo-code":      portableJoin(codeRoot, "rooveterinaryinc.roo-cline", "settings", "mcp_settings.json"),
		"continue":      portableJoin(home, ".continue", "config.yaml"),
		"opencode":      portableJoin(home, ".config", "opencode", "opencode.json"),
		"aider":         portableJoin(home, ".aider.conf.yml"),
		"cherry-studio": portableJoin(cherryRoot, "config.json"),
	}, nil
}

func configOrClientDirectoryExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	info, err := os.Stat(filepath.Dir(path))
	return err == nil && info.IsDir()
}

func portableJoin(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for index, part := range parts {
		part = strings.ReplaceAll(part, "\\", "/")
		if index == 0 {
			part = strings.TrimRight(part, "/")
		} else {
			part = strings.Trim(part, "/")
		}
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	return strings.Join(cleaned, "/")
}

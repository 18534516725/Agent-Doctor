package projects

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Kind string

const (
	KindNode   Kind = "node"
	KindPython Kind = "python"
	KindRust   Kind = "rust"
	KindGo     Kind = "go"
	KindMake   Kind = "make"
)

type CommandSuggestion struct {
	Label            string   `json:"label"`
	Argv             []string `json:"argv"`
	WorkingDirectory string   `json:"workingDirectory"`
	Approved         bool     `json:"approved"`
}

type Project struct {
	Root        string              `json:"root"`
	Kinds       []Kind              `json:"kinds"`
	Suggestions []CommandSuggestion `json:"suggestions"`
}

func Detect(root string) (Project, error) {
	canonicalRoot, err := canonicalDirectory(root)
	if err != nil {
		return Project{}, err
	}
	project := Project{Root: canonicalRoot}

	if exists(filepath.Join(canonicalRoot, "package.json")) {
		project.Kinds = append(project.Kinds, KindNode)
		suggestions, err := detectNodeSuggestions(canonicalRoot)
		if err != nil {
			return Project{}, err
		}
		project.Suggestions = append(project.Suggestions, suggestions...)
	}
	if exists(filepath.Join(canonicalRoot, "pyproject.toml")) {
		project.Kinds = append(project.Kinds, KindPython)
		project.Suggestions = append(project.Suggestions,
			suggestion("Run Python tests", canonicalRoot, "python", "-m", "pytest"),
		)
	}
	if exists(filepath.Join(canonicalRoot, "Cargo.toml")) {
		project.Kinds = append(project.Kinds, KindRust)
		project.Suggestions = append(project.Suggestions,
			suggestion("Run Rust tests", canonicalRoot, "cargo", "test"),
			suggestion("Check Rust project", canonicalRoot, "cargo", "check"),
		)
	}
	if exists(filepath.Join(canonicalRoot, "go.mod")) {
		project.Kinds = append(project.Kinds, KindGo)
		project.Suggestions = append(project.Suggestions,
			suggestion("Run Go tests", canonicalRoot, "go", "test", "./..."),
			suggestion("Vet Go project", canonicalRoot, "go", "vet", "./..."),
		)
	}
	if exists(filepath.Join(canonicalRoot, "Makefile")) {
		project.Kinds = append(project.Kinds, KindMake)
		project.Suggestions = append(project.Suggestions, detectMakeSuggestions(canonicalRoot)...)
	}
	return project, nil
}

func detectNodeSuggestions(root string) ([]CommandSuggestion, error) {
	raw, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return nil, fmt.Errorf("read package.json: %w", err)
	}
	var manifest struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}
	manager := nodePackageManager(root)
	labels := map[string]string{
		"test": "Run Node tests", "build": "Build Node project", "lint": "Lint Node project",
		"typecheck": "Type-check Node project", "check": "Check Node project",
	}
	order := []string{"test", "build", "lint", "typecheck", "check"}
	result := make([]CommandSuggestion, 0, len(order))
	for _, script := range order {
		if _, ok := manifest.Scripts[script]; !ok {
			continue
		}
		argv := []string{manager, "run", script}
		if script == "test" && (manager == "pnpm" || manager == "npm" || manager == "yarn") {
			argv = []string{manager, "test"}
		}
		result = append(result, suggestion(labels[script], root, argv...))
	}
	return result, nil
}

func nodePackageManager(root string) string {
	for _, candidate := range []struct{ file, command string }{
		{"pnpm-lock.yaml", "pnpm"}, {"yarn.lock", "yarn"}, {"package-lock.json", "npm"},
	} {
		if exists(filepath.Join(root, candidate.file)) {
			return candidate.command
		}
	}
	return "npm"
}

var makeTargetPattern = regexp.MustCompile(`(?m)^([A-Za-z0-9][A-Za-z0-9_.-]*):(?:\s|$)`)

func detectMakeSuggestions(root string) []CommandSuggestion {
	raw, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		return nil
	}
	allowedTargets := map[string]bool{"test": true, "build": true, "lint": true, "check": true, "verify": true}
	targets := make([]string, 0)
	for _, match := range makeTargetPattern.FindAllStringSubmatch(string(raw), -1) {
		if allowedTargets[match[1]] {
			targets = append(targets, match[1])
		}
	}
	sort.Strings(targets)
	result := make([]CommandSuggestion, 0, len(targets))
	for _, target := range targets {
		result = append(result, suggestion("Run make "+target, root, "make", target))
	}
	return result
}

func suggestion(label, root string, argv ...string) CommandSuggestion {
	return CommandSuggestion{Label: label, Argv: argv, WorkingDirectory: root, Approved: false}
}

func canonicalDirectory(root string) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve project directory: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve project directory links: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("inspect project directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project root is not a directory")
	}
	return filepath.Clean(resolved), nil
}

func exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && !strings.ContainsRune(path, '\x00')
}

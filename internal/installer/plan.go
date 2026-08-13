package installer

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Change struct {
	Path   string      `json:"path"`
	Before []byte      `json:"before"`
	After  []byte      `json:"after"`
	Mode   fs.FileMode `json:"mode"`
}

type Plan struct {
	Home     string   `json:"home"`
	Detected []Client `json:"detected"`
	Changes  []Change `json:"changes"`
	Warnings []string `json:"warnings"`
}

func BuildPlan(home string, detected []Client, changes []Change) (Plan, error) {
	canonicalHome, err := filepath.Abs(home)
	if err != nil {
		return Plan{}, fmt.Errorf("resolve installer home: %w", err)
	}
	canonicalHome, err = filepath.EvalSymlinks(filepath.Clean(canonicalHome))
	if err != nil {
		return Plan{}, fmt.Errorf("resolve installer home links: %w", err)
	}
	seen := make(map[string]bool, len(changes))
	normalized := make([]Change, len(changes))
	for index, change := range changes {
		path, err := filepath.Abs(change.Path)
		if err != nil {
			return Plan{}, fmt.Errorf("resolve change path: %w", err)
		}
		path, err = resolveExistingPrefix(filepath.Clean(path))
		if err != nil {
			return Plan{}, err
		}
		if !pathWithin(path, canonicalHome) {
			return Plan{}, fmt.Errorf("change path %q is outside the installer home", path)
		}
		if seen[path] {
			return Plan{}, fmt.Errorf("duplicate change path %q", path)
		}
		seen[path] = true
		if change.Mode == 0 {
			change.Mode = 0o600
		}
		change.Path = path
		change.Before = append([]byte(nil), change.Before...)
		change.After = append([]byte(nil), change.After...)
		normalized[index] = change
	}
	return Plan{Home: canonicalHome, Detected: append([]Client(nil), detected...), Changes: normalized}, nil
}

func resolveExistingPrefix(path string) (string, error) {
	current := path
	remainder := make([]string, 0)
	for {
		if _, err := os.Lstat(current); err == nil {
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", fmt.Errorf("resolve configuration path links: %w", err)
			}
			for index := len(remainder) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, remainder[index])
			}
			return filepath.Clean(resolved), nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("configuration path has no existing ancestor: %s", path)
		}
		remainder = append(remainder, filepath.Base(current))
		current = parent
	}
}

func (plan Plan) Diff() string {
	var builder strings.Builder
	for _, change := range plan.Changes {
		_, _ = fmt.Fprintf(&builder, "--- %s (before)\n+++ %s (after)\n", change.Path, change.Path)
		for _, line := range diffLines(change.Before, change.After) {
			builder.WriteString(line)
			builder.WriteByte('\n')
		}
	}
	return builder.String()
}

func diffLines(before, after []byte) []string {
	beforeLines := strings.Split(strings.TrimSuffix(string(before), "\n"), "\n")
	afterLines := strings.Split(strings.TrimSuffix(string(after), "\n"), "\n")
	beforeSet := make(map[string]int)
	for _, line := range beforeLines {
		beforeSet[line]++
	}
	result := make([]string, 0)
	for _, line := range beforeLines {
		if !strings.Contains(string(after), line) && line != "" {
			result = append(result, "-"+line)
		}
	}
	for _, line := range afterLines {
		if beforeSet[line] > 0 {
			beforeSet[line]--
			continue
		}
		if line != "" {
			result = append(result, "+"+line)
		}
	}
	sort.Strings(result)
	return result
}

func BeginMarker(owner string) string { return "# >>> agent-doctor:" + owner + " >>>" }
func EndMarker(owner string) string   { return "# <<< agent-doctor:" + owner + " <<<" }

func AppendMarkedBlock(existing []byte, owner string, body []byte) []byte {
	without, _ := removeMarkedBlock(existing, owner)
	result := bytes.TrimRight(without, "\n")
	if len(result) > 0 {
		result = append(result, '\n')
	}
	result = append(result, []byte(BeginMarker(owner)+"\n")...)
	result = append(result, bytes.TrimSpace(body)...)
	result = append(result, []byte("\n"+EndMarker(owner)+"\n")...)
	return result
}

func pathWithin(path, root string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

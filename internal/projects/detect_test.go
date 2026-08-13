package projects

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDetectNodeProjectSuggestsExistingPnpmScripts(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "package.json"), `{
		"scripts": {"test":"vitest", "build":"vite build", "dev":"vite"}
	}`)
	writeFixture(t, filepath.Join(root, "pnpm-lock.yaml"), "lockfileVersion: '9.0'\n")

	project, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(project.Kinds, KindNode) {
		t.Fatalf("kinds=%v", project.Kinds)
	}
	want := [][]string{{"pnpm", "test"}, {"pnpm", "run", "build"}}
	if got := argumentVectors(project.Suggestions); !reflect.DeepEqual(got, want) {
		t.Fatalf("suggestions=%v want=%v", got, want)
	}
	for _, suggestion := range project.Suggestions {
		if suggestion.Approved {
			t.Fatal("detection must never auto-approve a command")
		}
	}
}

func TestDetectPolyglotProjectUsesReadOnlyManifests(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "go.mod"), "module example.test/project\n\ngo 1.25\n")
	writeFixture(t, filepath.Join(root, "pyproject.toml"), "[project]\nname = 'demo'\n")
	writeFixture(t, filepath.Join(root, "Cargo.toml"), "[package]\nname='demo'\nversion='0.1.0'\n")

	project, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []Kind{KindGo, KindPython, KindRust} {
		if !contains(project.Kinds, kind) {
			t.Fatalf("kind %q missing from %v", kind, project.Kinds)
		}
	}
}

func TestDetectRejectsMissingDirectory(t *testing.T) {
	if _, err := Detect(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected missing directory error")
	}
}

func writeFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func contains[T comparable](values []T, want T) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func argumentVectors(suggestions []CommandSuggestion) [][]string {
	result := make([][]string, len(suggestions))
	for index, suggestion := range suggestions {
		result[index] = suggestion.Argv
	}
	return result
}

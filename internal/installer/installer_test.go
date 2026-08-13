package installer

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDetectClientsIsReadOnly(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, ".codex", "config.toml")
	writeInstallerFile(t, config, []byte("model = \"example\"\n"))
	before := treeDigest(t, home)
	clients, err := DetectClients(home, "darwin")
	if err != nil {
		t.Fatal(err)
	}
	after := treeDigest(t, home)
	if before != after {
		t.Fatal("detection modified the fake home")
	}
	if len(clients) == 0 || clients[0].ID != "codex" || !clients[0].Detected {
		t.Fatalf("clients=%+v", clients)
	}
}

func TestBuildPlanListsExactPathsAndContents(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".codex", "config.toml")
	before := []byte("existing = true\n")
	after := AppendMarkedBlock(before, "codex", []byte("[mcp_servers.agent_doctor]\ncommand = \"agent-doctor\""))
	plan, err := BuildPlan(home, []Client{{ID: "codex", Name: "Codex", Detected: true}}, []Change{{
		Path: path, Before: before, After: after, Mode: 0o600,
	}})
	if err != nil {
		t.Fatal(err)
	}
	canonicalHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		t.Fatal(err)
	}
	canonicalPath := filepath.Join(canonicalHome, ".codex", "config.toml")
	if len(plan.Changes) != 1 || plan.Changes[0].Path != canonicalPath || !reflect.DeepEqual(plan.Changes[0].After, after) {
		t.Fatalf("plan=%+v", plan)
	}
	if !strings.Contains(plan.Diff(), canonicalPath) || !strings.Contains(plan.Diff(), "mcp_servers.agent_doctor") {
		t.Fatalf("diff=%s", plan.Diff())
	}
}

func TestBuildPlanRejectsSymlinkEscape(t *testing.T) {
	home := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(home, "escaped")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, err := BuildPlan(home, nil, []Change{{
		Path: filepath.Join(link, "config.toml"), After: []byte("unsafe"), Mode: 0o600,
	}})
	if err == nil {
		t.Fatal("plan accepted a configuration path outside home through a symlink")
	}
}

func TestApplyIsIdempotentAndCreatesMarkedBlock(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".codex", "config.toml")
	before := []byte("existing = true\n")
	writeInstallerFile(t, path, before)
	after := AppendMarkedBlock(before, "codex", []byte("enabled = true"))
	plan, err := BuildPlan(home, nil, []Change{{Path: path, Before: before, After: after, Mode: 0o600}})
	if err != nil {
		t.Fatal(err)
	}
	first, err := Apply(plan)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Apply(plan)
	if err != nil {
		t.Fatal(err)
	}
	if first.Applied != 1 || second.Applied != 0 || second.Skipped != 1 {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	stored, _ := os.ReadFile(path)
	if strings.Count(string(stored), BeginMarker("codex")) != 1 {
		t.Fatalf("marker is not idempotent: %s", stored)
	}
	canonicalHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		t.Fatal(err)
	}
	if first.BackupDirectory == "" || !strings.HasPrefix(first.BackupDirectory, filepath.Join(canonicalHome, ".agent-doctor", "backups")) {
		t.Fatalf("backup directory=%q", first.BackupDirectory)
	}
}

func TestApplyFailureRollsBackEveryPriorChange(t *testing.T) {
	home := t.TempDir()
	firstPath := filepath.Join(home, "client-a", "config.toml")
	firstBefore := []byte("original\n")
	writeInstallerFile(t, firstPath, firstBefore)
	blockingParent := filepath.Join(home, "not-a-directory")
	writeInstallerFile(t, blockingParent, []byte("block"))
	secondPath := filepath.Join(blockingParent, "config.toml")
	plan, err := BuildPlan(home, nil, []Change{
		{Path: firstPath, Before: firstBefore, After: []byte("changed\n"), Mode: 0o600},
		{Path: secondPath, Before: nil, After: []byte("never\n"), Mode: 0o600},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(plan); err == nil {
		t.Fatal("expected transactional failure")
	}
	got, err := os.ReadFile(firstPath)
	if err != nil || !reflect.DeepEqual(got, firstBefore) {
		t.Fatalf("rollback failed: got=%q err=%v", got, err)
	}
}

func TestUninstallRemovesOnlyOwnedBlock(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, "config.toml")
	original := []byte("user_before = true\nuser_after = true\n")
	withBlock := AppendMarkedBlock([]byte("user_before = true\n"), "codex", []byte("owned = true"))
	withBlock = append(withBlock, []byte("user_after = true\n")...)
	writeInstallerFile(t, path, withBlock)
	if err := UninstallMarkedBlock(path, "codex"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || !reflect.DeepEqual(got, original) {
		t.Fatalf("uninstall changed user content: %q err=%v", got, err)
	}
}

func TestClientConfigPathFixtures(t *testing.T) {
	tests := []struct{ targetOS, home, codex string }{
		{"darwin", "/Users/demo", "/Users/demo/.codex/config.toml"},
		{"linux", "/home/demo", "/home/demo/.codex/config.toml"},
		{"windows", "C:/Users/demo", "C:/Users/demo/.codex/config.toml"},
	}
	for _, test := range tests {
		paths, err := ClientConfigPaths(test.home, test.targetOS)
		if err != nil {
			t.Fatal(err)
		}
		if paths["codex"] != test.codex {
			t.Fatalf("%s codex=%q", test.targetOS, paths["codex"])
		}
	}
}

func writeInstallerFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func treeDigest(t *testing.T, root string) string {
	t.Helper()
	hash := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(root, path)
		_, _ = fmt.Fprintf(hash, "%s:%t\n", relative, entry.IsDir())
		if !entry.IsDir() {
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			_, _ = hash.Write(contents)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

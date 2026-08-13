package replay

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutorRequiresConsentBoundToExactPlan(t *testing.T) {
	repository, sha := createRepository(t)
	plan := testPlan(repository, sha)
	executor := NewExecutor(t.TempDir())

	if _, err := executor.Execute(context.Background(), plan, Consent{}); !errors.Is(err, ErrConsentRequired) {
		t.Fatalf("expected consent error, got %v", err)
	}
	consent := Consent{PlanHash: plan.Hash()}
	plan.Model = "different-model"
	if _, err := executor.Execute(context.Background(), plan, consent); !errors.Is(err, ErrConsentRequired) {
		t.Fatalf("modified plan reused consent: %v", err)
	}
}

func TestExecutorRejectsUnknownBaseCommit(t *testing.T) {
	repository, _ := createRepository(t)
	plan := testPlan(repository, strings.Repeat("f", 40))
	_, err := NewExecutor(t.TempDir()).Execute(context.Background(), plan, Consent{PlanHash: plan.Hash()})
	if !errors.Is(err, ErrBaseCommitUnknown) {
		t.Fatalf("err=%v", err)
	}
}

func TestExecutorUsesDetachedExternalWorktreeAndCleansIt(t *testing.T) {
	repository, sha := createRepository(t)
	beforeBranch := gitOutput(t, repository, "branch", "--show-current")
	temporaryRoot := t.TempDir()
	plan := testPlan(repository, sha)
	result, err := NewExecutor(temporaryRoot).Execute(context.Background(), plan, Consent{PlanHash: plan.Hash()})
	if err != nil {
		t.Fatal(err)
	}
	if result.WorktreePath == "" || !strings.HasPrefix(result.WorktreePath, temporaryRoot+string(filepath.Separator)) {
		t.Fatalf("unexpected worktree path %q", result.WorktreePath)
	}
	if _, err := os.Stat(result.WorktreePath); !os.IsNotExist(err) {
		t.Fatalf("temporary worktree was not removed: %v", err)
	}
	if got := gitOutput(t, repository, "branch", "--show-current"); got != beforeBranch {
		t.Fatalf("current branch changed from %q to %q", beforeBranch, got)
	}
	if result.MaxCalls != plan.MaxCalls || result.MaxCostMicros != plan.MaxCostMicros || !result.CleanupCompleted {
		t.Fatalf("limits or cleanup missing: %+v", result)
	}
}

func TestExecutorRunsOnlyApprovedCommands(t *testing.T) {
	repository, sha := createRepository(t)
	plan := testPlan(repository, sha)
	plan.Commands = append(plan.Commands, Command{Argv: []string{"touch", "forbidden"}, Approved: false})
	_, err := NewExecutor(t.TempDir()).Execute(context.Background(), plan, Consent{PlanHash: plan.Hash()})
	if !errors.Is(err, ErrCommandNotApproved) {
		t.Fatalf("err=%v", err)
	}
	if _, statErr := os.Stat(filepath.Join(repository, "forbidden")); !os.IsNotExist(statErr) {
		t.Fatal("unapproved command modified current repository")
	}
}

func TestPreviewIncludesSafetyBoundary(t *testing.T) {
	repository, sha := createRepository(t)
	plan := testPlan(repository, sha)
	preview := plan.Preview()
	for _, expected := range []string{"Codex", "model-a", sha, "safe task", "go test ./...", "Max calls: 3", "Max cost: 250000 micros", "temporary detached worktree"} {
		if !strings.Contains(preview, expected) {
			t.Fatalf("preview missing %q:\n%s", expected, preview)
		}
	}
}

func testPlan(repository, sha string) Plan {
	return Plan{
		Repository:    repository,
		BaseSHA:       sha,
		Client:        "Codex",
		Model:         "model-a",
		SanitizedTask: "safe task",
		Commands:      []Command{{Argv: []string{"go", "test", "./..."}, Approved: true}},
		MaxCalls:      3,
		MaxCostMicros: 250000,
	}
}

func createRepository(t *testing.T) (string, string) {
	t.Helper()
	repository := t.TempDir()
	gitRun(t, repository, "init", "-b", "main")
	gitRun(t, repository, "config", "user.email", "test@example.invalid")
	gitRun(t, repository, "config", "user.name", "Agent Doctor Test")
	if err := os.WriteFile(filepath.Join(repository, "go.mod"), []byte("module replay-test\n\ngo 1.25\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "main.go"), []byte("package replaytest\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repository, "add", "go.mod", "main.go")
	gitRun(t, repository, "commit", "-m", "initial")
	return repository, gitOutput(t, repository, "rev-parse", "HEAD")
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(output))
}

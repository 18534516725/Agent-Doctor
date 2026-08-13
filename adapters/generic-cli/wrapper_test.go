package genericcli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBuildInvocationPreservesArgvWithoutShell(t *testing.T) {
	invocation, err := BuildInvocation([]string{"aider", "--message", "fix $(whoami); do not execute"})
	if err != nil {
		t.Fatal(err)
	}
	if invocation.Path != "aider" || len(invocation.Args) != 2 || invocation.Args[1] != "fix $(whoami); do not execute" || invocation.Shell {
		t.Fatalf("invocation=%+v", invocation)
	}
	if _, err := BuildInvocation(nil); err == nil {
		t.Fatal("expected missing argv to fail")
	}
}

func TestRunPreservesExitStatusAndCapturesOnlySnapshots(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable fixture uses sh")
	}
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "file.txt"), []byte("before\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("git", "init", directory).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	commit := exec.Command("git", "-C", directory, "-c", "user.name=Test", "-c", "user.email=test@example.com", "add", "file.txt")
	if output, err := commit.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, output)
	}
	commit = exec.Command("git", "-C", directory, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "initial")
	if output, err := commit.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v: %s", err, output)
	}
	result, err := Run(context.Background(), RunRequest{Directory: directory, Invocation: Invocation{Path: "sh", Args: []string{"-c", "printf after > file.txt; exit 7"}}})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 7 || result.BeforeSnapshot == "" || result.AfterSnapshot == "" || result.BeforeSnapshot == result.AfterSnapshot || result.CapturedTranscript {
		t.Fatalf("result=%+v", result)
	}
}

func TestRunWithoutGitReturnsLimitedReport(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable fixture uses sh")
	}
	directory := t.TempDir()
	result, err := Run(context.Background(), RunRequest{Directory: directory, Invocation: Invocation{Path: "sh", Args: []string{"-c", "exit 0"}}})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 || !result.Limited || result.BeforeSnapshot != "" || result.AfterSnapshot != "" {
		t.Fatalf("result=%+v", result)
	}
}

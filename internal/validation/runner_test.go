package validation

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestRunnerRejectsUnapprovedCommand(t *testing.T) {
	runner := NewRunner(NewAllowlist())
	_, err := runner.Run(context.Background(), Command{
		Argv:             []string{"go", "test", "./..."},
		WorkingDirectory: t.TempDir(),
	})
	if !errors.Is(err, ErrCommandNotAllowed) {
		t.Fatalf("err=%v", err)
	}
}

func TestRunnerRequiresExactArgvAndWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	allowlist := NewAllowlist()
	approved := Command{Argv: []string{"go", "test", "./..."}, WorkingDirectory: root}
	if err := allowlist.Approve(approved); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(allowlist)
	for _, rejected := range []Command{
		{Argv: []string{"go", "test", "./...", "-count=1"}, WorkingDirectory: root},
		{Argv: approved.Argv, WorkingDirectory: t.TempDir()},
	} {
		if _, err := runner.Run(context.Background(), rejected); !errors.Is(err, ErrCommandNotAllowed) {
			t.Fatalf("expected exact-match rejection, got %v", err)
		}
	}
}

func TestAllowlistRejectsDirectShellEvaluation(t *testing.T) {
	root := t.TempDir()
	for _, argv := range [][]string{
		{"sh", "-c", "echo unsafe"},
		{"bash", "-c", "echo unsafe"},
		{"cmd.exe", "/c", "echo unsafe"},
		{"powershell.exe", "-Command", "Write-Output unsafe"},
	} {
		if err := NewAllowlist().Approve(Command{Argv: argv, WorkingDirectory: root}); err == nil {
			t.Fatalf("shell evaluation was approved: %v", argv)
		}
	}
}

func TestRunnerUsesArgumentVectorWithoutShell(t *testing.T) {
	root := t.TempDir()
	literal := "$(touch should-not-run); echo still-literal"
	command := Command{Argv: []string{"synthetic-tool", literal}, WorkingDirectory: root}
	allowlist := NewAllowlist()
	if err := allowlist.Approve(command); err != nil {
		t.Fatal(err)
	}

	runner := NewRunner(allowlist)
	runner.commandFactory = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		helper := exec.CommandContext(ctx, os.Args[0], "-test.run=TestValidationHelperProcess", "--", name)
		helper.Args = append(helper.Args, args...)
		helper.Env = append(os.Environ(), "AGENT_DOCTOR_HELPER=1")
		return helper
	}
	result, err := runner.Run(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Stdout, literal) {
		t.Fatalf("literal argument missing: %q", result.Stdout)
	}
	if _, err := os.Stat(root + "/should-not-run"); !os.IsNotExist(err) {
		t.Fatal("shell metacharacters were executed")
	}
}

func TestRunnerHonorsContextCancellation(t *testing.T) {
	root := t.TempDir()
	command := Command{Argv: []string{"synthetic-sleep"}, WorkingDirectory: root}
	allowlist := NewAllowlist()
	if err := allowlist.Approve(command); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(allowlist)
	runner.commandFactory = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		helper := exec.CommandContext(ctx, os.Args[0], "-test.run=TestValidationHelperProcess", "--", "sleep")
		helper.Env = append(os.Environ(), "AGENT_DOCTOR_HELPER=1")
		return helper
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	result, err := runner.Run(ctx, command)
	if err == nil || !result.TimedOut {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestRunnerRedactsCapturedCredentials(t *testing.T) {
	root := t.TempDir()
	command := Command{Argv: []string{"synthetic-secret-output"}, WorkingDirectory: root}
	allowlist := NewAllowlist()
	if err := allowlist.Approve(command); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(allowlist)
	runner.commandFactory = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		helper := exec.CommandContext(ctx, os.Args[0], "-test.run=TestValidationHelperProcess", "--", "secret")
		helper.Env = append(os.Environ(), "AGENT_DOCTOR_HELPER=1")
		return helper
	}
	result, err := runner.Run(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.Stdout, "synthetic-secret") {
		t.Fatalf("credential survived command capture: %q", result.Stdout)
	}
}

func TestValidationHelperProcess(t *testing.T) {
	if os.Getenv("AGENT_DOCTOR_HELPER") != "1" {
		return
	}
	arguments := argsAfterDoubleDash(os.Args)
	if len(arguments) > 0 && arguments[0] == "sleep" {
		time.Sleep(time.Second)
		os.Exit(0)
	}
	if len(arguments) > 0 && arguments[0] == "secret" {
		_, _ = os.Stdout.WriteString("Authorization: Bearer synthetic-secret")
		os.Exit(0)
	}
	_, _ = os.Stdout.WriteString(strings.Join(arguments, "\n"))
	os.Exit(0)
}

func argsAfterDoubleDash(arguments []string) []string {
	for index, argument := range arguments {
		if argument == "--" {
			return arguments[index+1:]
		}
	}
	return nil
}

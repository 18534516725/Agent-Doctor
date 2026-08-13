package genericcli

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

type Invocation struct {
	Path  string
	Args  []string
	Shell bool
}

type RunRequest struct {
	Directory  string
	Invocation Invocation
}

type RunResult struct {
	ExitCode           int
	BeforeSnapshot     string
	AfterSnapshot      string
	Limited            bool
	CapturedTranscript bool
}

func BuildInvocation(argv []string) (Invocation, error) {
	if len(argv) == 0 || argv[0] == "" {
		return Invocation{}, fmt.Errorf("command argv is required")
	}
	return Invocation{Path: argv[0], Args: append([]string(nil), argv[1:]...)}, nil
}

// Run executes exactly the supplied argv. It intentionally never invokes a
// shell or captures stdout/stderr transcripts; only safe Git state hashes are
// retained for later comparison.
func Run(ctx context.Context, request RunRequest) (RunResult, error) {
	if request.Directory == "" || request.Invocation.Path == "" || request.Invocation.Shell {
		return RunResult{}, fmt.Errorf("run request is invalid")
	}
	before, hasGit := gitSnapshot(ctx, request.Directory)
	command := exec.CommandContext(ctx, request.Invocation.Path, request.Invocation.Args...)
	command.Dir = request.Directory
	err := command.Run()
	result := RunResult{BeforeSnapshot: before, Limited: !hasGit, CapturedTranscript: false}
	if err == nil {
		result.ExitCode = 0
	} else if exitError, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitError.ExitCode()
	} else {
		return RunResult{}, err
	}
	if hasGit {
		result.AfterSnapshot, _ = gitSnapshot(ctx, request.Directory)
	}
	return result, nil
}

func gitSnapshot(ctx context.Context, directory string) (string, bool) {
	command := exec.CommandContext(ctx, "git", "-C", directory, "rev-parse", "--is-inside-work-tree")
	if command.Run() != nil {
		return "", false
	}
	status := exec.CommandContext(ctx, "git", "-C", directory, "status", "--porcelain=v1")
	output, err := status.Output()
	if err != nil {
		return "", false
	}
	entries := string(output)
	paths := make([]string, 0)
	for _, line := range strings.Split(strings.TrimSpace(entries), "\n") {
		if line != "" {
			paths = append(paths, line)
		}
	}
	sort.Strings(paths)
	hash := sha256.Sum256([]byte(strings.Join(paths, "\n") + "\n"))
	return fmt.Sprintf("%x", hash[:]), true
}

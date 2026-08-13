package validation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/privacy"
)

const maxCapturedOutputBytes = 1024 * 1024

type Result struct {
	ExitCode  int           `json:"exitCode"`
	Stdout    string        `json:"stdout"`
	Stderr    string        `json:"stderr"`
	Duration  time.Duration `json:"duration"`
	Truncated bool          `json:"truncated"`
	TimedOut  bool          `json:"timedOut"`
}

type Runner struct {
	allowlist      *Allowlist
	commandFactory func(context.Context, string, ...string) *exec.Cmd
}

func NewRunner(allowlist *Allowlist) *Runner {
	return &Runner{allowlist: allowlist, commandFactory: exec.CommandContext}
}

func (runner *Runner) Run(ctx context.Context, requested Command) (Result, error) {
	canonical, err := canonicalCommand(requested)
	if err != nil {
		return Result{}, err
	}
	if runner.allowlist == nil || !runner.allowlist.Allows(canonical) {
		return Result{}, ErrCommandNotAllowed
	}

	command := runner.commandFactory(ctx, canonical.Argv[0], canonical.Argv[1:]...)
	command.Dir = canonical.WorkingDirectory
	stdout := &boundedBuffer{limit: maxCapturedOutputBytes}
	stderr := &boundedBuffer{limit: maxCapturedOutputBytes}
	command.Stdout = stdout
	command.Stderr = stderr

	started := time.Now()
	runErr := command.Run()
	result := Result{
		ExitCode:  exitCode(runErr),
		Stdout:    privacy.FilterText(stdout.String()),
		Stderr:    privacy.FilterText(stderr.String()),
		Duration:  time.Since(started),
		Truncated: stdout.truncated || stderr.truncated,
		TimedOut:  errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(ctx.Err(), context.Canceled),
	}
	if runErr != nil {
		return result, fmt.Errorf("validation command failed: %w", runErr)
	}
	return result, nil
}

type boundedBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (buffer *boundedBuffer) Write(data []byte) (int, error) {
	originalLength := len(data)
	remaining := buffer.limit - buffer.buffer.Len()
	if remaining <= 0 {
		buffer.truncated = buffer.truncated || originalLength > 0
		return originalLength, nil
	}
	if len(data) > remaining {
		data = data[:remaining]
		buffer.truncated = true
	}
	_, _ = buffer.buffer.Write(data)
	return originalLength, nil
}

func (buffer *boundedBuffer) String() string { return buffer.buffer.String() }

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode()
	}
	return -1
}

package replay

import (
	"context"
	"errors"
	"fmt"

	"github.com/18534516725/Agent-Doctor/internal/validation"
)

var (
	ErrConsentRequired    = errors.New("one-time consent bound to the exact replay plan is required")
	ErrCommandNotApproved = errors.New("replay command is not approved")
)

type CommandResult struct {
	Argv   []string          `json:"argv"`
	Result validation.Result `json:"result"`
}

type Result struct {
	WorktreePath     string          `json:"worktreePath"`
	Commands         []CommandResult `json:"commands"`
	MaxCalls         int             `json:"maxCalls"`
	MaxCostMicros    int64           `json:"maxCostMicros"`
	CleanupCompleted bool            `json:"cleanupCompleted"`
}

type Executor struct {
	temporaryRoot string
}

func NewExecutor(temporaryRoot string) *Executor {
	return &Executor{temporaryRoot: temporaryRoot}
}

func (executor *Executor) Execute(ctx context.Context, plan Plan, consent Consent) (result Result, err error) {
	if consent.PlanHash == "" || consent.PlanHash != plan.Hash() {
		return Result{}, ErrConsentRequired
	}
	for _, command := range plan.Commands {
		if !command.Approved {
			return Result{}, ErrCommandNotApproved
		}
	}
	item, err := createWorktree(plan.Repository, executor.temporaryRoot, plan.BaseSHA)
	if err != nil {
		return Result{}, err
	}
	result = Result{WorktreePath: item.path, MaxCalls: plan.MaxCalls, MaxCostMicros: plan.MaxCostMicros}
	defer func() {
		cleanupErr := item.cleanup()
		result.CleanupCompleted = cleanupErr == nil
		if cleanupErr != nil && err == nil {
			err = cleanupErr
		}
	}()

	allowlist := validation.NewAllowlist()
	commands := make([]validation.Command, 0, len(plan.Commands))
	for _, requested := range plan.Commands {
		command := validation.Command{Argv: append([]string(nil), requested.Argv...), WorkingDirectory: item.path}
		if approveErr := allowlist.Approve(command); approveErr != nil {
			return result, fmt.Errorf("approve replay command: %w", approveErr)
		}
		commands = append(commands, command)
	}
	runner := validation.NewRunner(allowlist)
	for _, command := range commands {
		commandResult, runErr := runner.Run(ctx, command)
		result.Commands = append(result.Commands, CommandResult{Argv: command.Argv, Result: commandResult})
		if runErr != nil {
			return result, runErr
		}
	}
	return result, nil
}

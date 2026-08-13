package replay

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrBaseCommitUnknown = errors.New("replay base commit is unknown")

type worktree struct {
	repository string
	parent     string
	path       string
}

func createWorktree(repository, temporaryRoot, baseSHA string) (worktree, error) {
	repository, err := filepath.Abs(repository)
	if err != nil {
		return worktree{}, fmt.Errorf("resolve repository: %w", err)
	}
	resolved, err := runGitOutput(repository, "rev-parse", "--verify", baseSHA+"^{commit}")
	if err != nil || strings.TrimSpace(resolved) == "" {
		return worktree{}, fmt.Errorf("%w: %s", ErrBaseCommitUnknown, baseSHA)
	}
	parent, err := os.MkdirTemp(temporaryRoot, "agent-doctor-replay-")
	if err != nil {
		return worktree{}, fmt.Errorf("create replay parent: %w", err)
	}
	target := filepath.Join(parent, "worktree")
	if inside(target, repository) {
		_ = os.Remove(parent)
		return worktree{}, fmt.Errorf("temporary worktree must be outside current repository")
	}
	command := exec.Command("git", "-C", repository, "worktree", "add", "--detach", target, strings.TrimSpace(resolved))
	if output, err := command.CombinedOutput(); err != nil {
		_ = os.Remove(parent)
		return worktree{}, fmt.Errorf("create detached worktree: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return worktree{repository: repository, parent: parent, path: target}, nil
}

func (item worktree) cleanup() error {
	command := exec.Command("git", "-C", item.repository, "worktree", "remove", "--force", item.path)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("remove replay worktree: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if err := os.Remove(item.parent); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove replay parent: %w", err)
	}
	return nil
}

func runGitOutput(repository string, arguments ...string) (string, error) {
	command := exec.Command("git", append([]string{"-C", repository}, arguments...)...)
	output, err := command.Output()
	return string(output), err
}

func inside(path, root string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

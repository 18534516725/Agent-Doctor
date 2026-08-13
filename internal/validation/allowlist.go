package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var ErrCommandNotAllowed = errors.New("validation command is not explicitly approved")

type Command struct {
	Argv             []string `json:"argv"`
	WorkingDirectory string   `json:"workingDirectory"`
}

type Allowlist struct {
	mu       sync.RWMutex
	approved map[string]Command
}

func NewAllowlist() *Allowlist {
	return &Allowlist{approved: make(map[string]Command)}
}

func (allowlist *Allowlist) Approve(command Command) error {
	canonical, err := canonicalCommand(command)
	if err != nil {
		return err
	}
	key, err := commandKey(canonical)
	if err != nil {
		return err
	}
	allowlist.mu.Lock()
	allowlist.approved[key] = canonical
	allowlist.mu.Unlock()
	return nil
}

func (allowlist *Allowlist) Allows(command Command) bool {
	canonical, err := canonicalCommand(command)
	if err != nil {
		return false
	}
	key, err := commandKey(canonical)
	if err != nil {
		return false
	}
	allowlist.mu.RLock()
	_, ok := allowlist.approved[key]
	allowlist.mu.RUnlock()
	return ok
}

func canonicalCommand(command Command) (Command, error) {
	if len(command.Argv) == 0 || strings.TrimSpace(command.Argv[0]) == "" {
		return Command{}, fmt.Errorf("command executable is required")
	}
	for _, argument := range command.Argv {
		if strings.ContainsRune(argument, '\x00') {
			return Command{}, fmt.Errorf("command arguments cannot contain NUL")
		}
	}
	if invokesCommandStringInterpreter(command.Argv) {
		return Command{}, fmt.Errorf("direct shell command evaluation is not allowed")
	}
	abs, err := filepath.Abs(command.WorkingDirectory)
	if err != nil {
		return Command{}, fmt.Errorf("resolve working directory: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return Command{}, fmt.Errorf("resolve working directory links: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return Command{}, fmt.Errorf("working directory must exist and be a directory")
	}
	argv := append([]string(nil), command.Argv...)
	return Command{Argv: argv, WorkingDirectory: filepath.Clean(resolved)}, nil
}

func invokesCommandStringInterpreter(argv []string) bool {
	executable := strings.ToLower(filepath.Base(argv[0]))
	for _, argument := range argv[1:] {
		option := strings.ToLower(argument)
		switch executable {
		case "sh", "bash", "zsh", "fish", "dash", "ksh":
			if option == "-c" {
				return true
			}
		case "cmd", "cmd.exe":
			if option == "/c" || option == "/k" {
				return true
			}
		case "powershell", "powershell.exe", "pwsh", "pwsh.exe":
			if option == "-command" || option == "-c" || option == "-encodedcommand" {
				return true
			}
		}
	}
	return false
}

func commandKey(command Command) (string, error) {
	raw, err := json.Marshal(command)
	if err != nil {
		return "", fmt.Errorf("encode approved command: %w", err)
	}
	return string(raw), nil
}

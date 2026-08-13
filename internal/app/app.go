package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/adapters/claudecode"
	"github.com/18534516725/Agent-Doctor/internal/adapters/cline"
	"github.com/18534516725/Agent-Doctor/internal/events"
	"github.com/18534516725/Agent-Doctor/internal/mcp"
	"github.com/18534516725/Agent-Doctor/internal/storage"
)

const developmentVersion = "dev"

// Run dispatches CLI commands and returns a process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	return RunWithInput(args, os.Stdin, stdout, stderr)
}

// RunWithInput is the testable command entrypoint. Production callers use Run,
// which wires the protocol to the process standard input.
func RunWithInput(args []string, input io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "version" {
		fmt.Fprintf(stdout, "agent-doctor %s\n", developmentVersion)
		return 0
	}
	if len(args) == 2 && args[0] == "mcp" && args[1] == "serve" {
		backend := mcp.ToolBackend(unavailableMCPBackend{})
		if database, err := openLocalStore(); err == nil {
			defer database.Close()
			backend = localMCPBackend{store: database}
		}
		if err := mcp.NewServer(developmentVersion, backend).Serve(context.Background(), input, stdout); err != nil {
			fmt.Fprintln(stderr, "agent-doctor MCP server failed")
			return 1
		}
		return 0
	}
	if len(args) == 3 && args[0] == "hook" && args[1] == "claude-code" {
		database, err := openLocalStore()
		if err != nil {
			fmt.Fprintln(stderr, "agent-doctor: local lifecycle event was not recorded; Claude Code will continue normally")
			return 0
		}
		defer database.Close()
		return runClaudeCodeHook(input, func(event events.Event) error {
			return database.InsertEvent(context.Background(), event)
		}, stderr, time.Now)
	}
	if len(args) == 3 && args[0] == "hook" && args[1] == "cline" {
		database, err := openLocalStore()
		if err != nil {
			fmt.Fprintln(stderr, "agent-doctor: local lifecycle event was not recorded; Cline will continue normally")
			return 0
		}
		defer database.Close()
		return runClineHook(input, func(event events.Event) error {
			return database.InsertEvent(context.Background(), event)
		}, stderr)
	}

	fmt.Fprintln(stderr, "usage: agent-doctor <command>")
	return 2
}

func runClaudeCodeHook(input io.Reader, insert func(events.Event) error, diagnostics io.Writer, now func() time.Time) int {
	return claudecode.IngestFailOpen(input, now(), insert, diagnostics)
}

func runClineHook(input io.Reader, insert func(events.Event) error, diagnostics io.Writer) int {
	raw, err := io.ReadAll(io.LimitReader(input, events.MaxPayloadBytes+1))
	if err == nil && len(raw) <= events.MaxPayloadBytes {
		var event events.Event
		event, err = cline.NormalizeHook(json.RawMessage(raw))
		if err == nil && insert != nil {
			err = insert(event)
		}
	}
	if err != nil || len(raw) > events.MaxPayloadBytes || insert == nil {
		fmt.Fprintln(diagnostics, "agent-doctor: local lifecycle event was not recorded; Cline will continue normally")
	}
	return 0
}

func openLocalStore() (*storage.DB, error) {
	configDirectory, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("resolve local data directory: %w", err)
	}
	return storage.Open(filepath.Join(configDirectory, "AgentDoctor", "doctor.db"))
}

// unavailableMCPBackend is intentionally conservative until the user has
// captured compatible local evidence. It keeps the installed MCP server useful
// and honest rather than fabricating diagnostics from missing telemetry.
type unavailableMCPBackend struct{}

func (unavailableMCPBackend) Execute(_ context.Context, tool string, _ map[string]any) (mcp.ToolEvidence, error) {
	return mcp.ToolEvidence{
		Summary:        "No compatible local evidence has been captured for this request yet.",
		Items:          []mcp.EvidenceItem{{Label: "Requested tool", Value: tool}},
		Provenance:     "local-evidence-unavailable",
		Precision:      "unavailable",
		DataLimitNotes: []string{"Install a supported adapter or capture a consented local task before requesting a diagnostic."},
	}, nil
}

type localEvidenceStore interface {
	ListSessionEvents(context.Context, string) ([]events.Event, error)
}

// localMCPBackend turns normalized local events into bounded, payload-free
// evidence. Tools for which no compatible telemetry exists remain unavailable.
type localMCPBackend struct{ store localEvidenceStore }

func (backend localMCPBackend) Execute(ctx context.Context, tool string, arguments map[string]any) (mcp.ToolEvidence, error) {
	if tool != "get_task_evidence" {
		return unavailableMCPBackend{}.Execute(ctx, tool, arguments)
	}
	sessionID, _ := arguments["sessionId"].(string)
	items, err := backend.store.ListSessionEvents(ctx, sessionID)
	if err != nil {
		return mcp.ToolEvidence{}, err
	}
	if len(items) == 0 {
		return mcp.ToolEvidence{
			Summary: "No compatible local lifecycle evidence was found for this task.", Provenance: "local-event-store", Precision: "unavailable",
			DataLimitNotes: []string{"No prompt, source code, command, file path, or credential was searched or inferred."},
		}, nil
	}
	precision := "exact"
	evidence := mcp.ToolEvidence{
		Summary:    fmt.Sprintf("%d sanitized lifecycle events were captured for this task.", len(items)),
		Provenance: "local-event-store", Precision: precision,
		Items:          make([]mcp.EvidenceItem, 0, len(items)),
		DataLimitNotes: []string{"Event payloads, prompts, source code, commands, file paths, and credentials are intentionally excluded."},
	}
	for _, event := range items {
		if event.Precision == events.PrecisionUnavailable {
			precision = "unavailable"
		} else if event.Precision == events.PrecisionEstimated && precision == "exact" {
			precision = "estimated"
		}
		evidence.Items = append(evidence.Items, mcp.EvidenceItem{
			Label: event.Timestamp.UTC().Format(time.RFC3339),
			Value: strings.TrimSpace(event.EventType),
		})
	}
	evidence.Precision = precision
	return evidence, nil
}

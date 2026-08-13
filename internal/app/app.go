package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/adapters/claudecode"
	"github.com/18534516725/Agent-Doctor/internal/adapters/cline"
	"github.com/18534516725/Agent-Doctor/internal/events"
	"github.com/18534516725/Agent-Doctor/internal/installer"
	"github.com/18534516725/Agent-Doctor/internal/mcp"
	localserver "github.com/18534516725/Agent-Doctor/internal/server"
	"github.com/18534516725/Agent-Doctor/internal/storage"
)

var Version = "dev"

// Run dispatches CLI commands and returns a process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	return RunWithInput(args, os.Stdin, stdout, stderr)
}

// RunWithInput is the testable command entrypoint. Production callers use Run,
// which wires the protocol to the process standard input.
func RunWithInput(args []string, input io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "version" {
		fmt.Fprintf(stdout, "agent-doctor %s\n", Version)
		return 0
	}
	if len(args) >= 1 && (args[0] == "start" || args[0] == "dashboard") {
		return runLocalDashboard(args[1:], stdout, stderr)
	}
	if len(args) == 2 && args[0] == "doctor" && args[1] == "--json" {
		return runDoctorJSON(stdout, stderr)
	}
	if len(args) == 2 && args[0] == "mcp" && args[1] == "serve" {
		backend := mcp.ToolBackend(unavailableMCPBackend{})
		if database, err := openLocalStore(); err == nil {
			defer database.Close()
			backend = localMCPBackend{store: database}
		}
		if err := mcp.NewServer(Version, backend).Serve(context.Background(), input, stdout); err != nil {
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

	printUsage(stderr)
	return 2
}

func runLocalDashboard(args []string, stdout, stderr io.Writer) int {
	once := len(args) == 1 && args[0] == "--once"
	if len(args) > 0 && !once {
		printUsage(stderr)
		return 2
	}
	database, err := openLocalStore()
	if err != nil {
		fmt.Fprintln(stderr, "agent-doctor: local database unavailable")
		return 1
	}
	defer database.Close()
	service, err := localserver.New(localserver.Config{Version: Version, Store: database})
	if err != nil {
		fmt.Fprintln(stderr, "agent-doctor: local dashboard unavailable")
		return 1
	}
	listener, err := service.Listen()
	if err != nil {
		fmt.Fprintln(stderr, "agent-doctor: loopback listener unavailable")
		return 1
	}
	url := "http://" + listener.Addr().String() + "/"
	if once {
		_ = listener.Close()
		fmt.Fprintln(stdout, url)
		return 0
	}
	fmt.Fprintln(stdout, "Agent Doctor is ready:", url)
	fmt.Fprintln(stdout, "Open this local URL in your browser. Press Ctrl+C to stop.")
	interrupt, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	serveErrors := make(chan error, 1)
	go func() { serveErrors <- service.Serve(listener) }()
	select {
	case <-interrupt.Done():
		shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := service.Shutdown(shutdownContext); err != nil {
			fmt.Fprintln(stderr, "agent-doctor: dashboard shutdown incomplete")
			return 1
		}
		return 0
	case err := <-serveErrors:
		if err != nil {
			fmt.Fprintln(stderr, "agent-doctor: dashboard server stopped unexpectedly")
			return 1
		}
		return 0
	}
}

func runDoctorJSON(stdout, stderr io.Writer) int {
	database, err := openLocalStore()
	if err != nil {
		fmt.Fprintln(stderr, "agent-doctor: local database unavailable")
		return 1
	}
	defer database.Close()
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(stderr, "agent-doctor: home directory unavailable")
		return 1
	}
	clients, err := installer.DetectClients(home, runtime.GOOS)
	if err != nil {
		fmt.Fprintln(stderr, "agent-doctor: client detection unavailable")
		return 1
	}
	detected := make([]string, 0)
	for _, client := range clients {
		if client.Detected {
			detected = append(detected, client.Name)
		}
	}
	result := map[string]any{"status": "ready", "version": Version, "database": map[string]any{"schemaVersion": database.SchemaVersion(), "readOnlyRecovery": database.ReadOnly()}, "detectedClients": detected}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		fmt.Fprintln(stderr, "agent-doctor: status encoding failed")
		return 1
	}
	return 0
}

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, "usage: agent-doctor <command>")
	fmt.Fprintln(writer, "commands: setup, start, dashboard, diagnose, compare, context, costs, doctor, pause, export, forget, run, uninstall, mcp serve, version")
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
	configDirectory := os.Getenv("AGENT_DOCTOR_CONFIG_DIR")
	if configDirectory == "" {
		var err error
		configDirectory, err = os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("resolve local data directory: %w", err)
		}
		configDirectory = filepath.Join(configDirectory, "AgentDoctor")
	}
	return storage.Open(filepath.Join(configDirectory, "doctor.db"))
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

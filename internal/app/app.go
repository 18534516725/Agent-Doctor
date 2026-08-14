package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
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
	localproxy "github.com/18534516725/Agent-Doctor/internal/proxy"
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
	if len(args) == 0 || (len(args) == 1 && (args[0] == "--help" || args[0] == "help")) {
		printUsage(stdout)
		return 0
	}
	if len(args) == 1 && args[0] == "version" {
		fmt.Fprintf(stdout, "agent-doctor %s\n", Version)
		return 0
	}
	if len(args) >= 1 && (args[0] == "start" || args[0] == "dashboard") {
		return runLocalDashboard(args[1:], stdout, stderr)
	}
	if len(args) >= 1 && args[0] == "doctor" && hasFlag(args[1:], "--json") {
		return runDoctorJSON(stdout, stderr)
	}
	if len(args) >= 1 && args[0] == "setup" {
		return runSetup(args[1:], stdout, stderr)
	}
	if len(args) >= 1 && args[0] == "uninstall" {
		return runUninstall(args[1:], stdout, stderr)
	}
	if len(args) >= 1 && args[0] == "run" {
		return runWrapped(args[1:], input, stdout, stderr)
	}
	if len(args) >= 1 && args[0] == "pause" && hasFlag(args[1:], "--json") {
		return runPause(args[1:], stdout, stderr)
	}
	if len(args) >= 1 && hasFlag(args[1:], "--json") {
		switch args[0] {
		case "diagnose", "compare", "context", "costs", "export":
			return runLocalDataCommand(args[0], stdout, stderr)
		case "forget":
			return runForget(args[1:], stdout, stderr)
		}
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
		if capturePaused() {
			return 0
		}
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
		if capturePaused() {
			return 0
		}
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
	once := hasFlag(args, "--once")
	for _, argument := range args {
		if argument != "--once" && argument != "--no-open" {
			printUsage(stderr)
			return 2
		}
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
	var proxyListener net.Listener
	var proxyHTTPServer *http.Server
	proxyURL := ""
	if upstreamURL := strings.TrimSpace(os.Getenv("AGENT_DOCTOR_UPSTREAM_URL")); upstreamURL != "" {
		proxyListener, err = (&net.ListenConfig{}).Listen(context.Background(), "tcp4", "127.0.0.1:0")
		if err != nil {
			_ = listener.Close()
			fmt.Fprintln(stderr, "agent-doctor: capture proxy listener unavailable")
			return 1
		}
		proxyHandler, proxyErr := localproxy.New(localproxy.Config{
			UpstreamURL: upstreamURL, ListenAddress: proxyListener.Addr().String(), Store: database,
			ClientName:    defaultString(os.Getenv("AGENT_DOCTOR_CLIENT"), "auto-detected"),
			ClientVersion: os.Getenv("AGENT_DOCTOR_CLIENT_VERSION"),
			ProjectID:     defaultString(os.Getenv("AGENT_DOCTOR_PROJECT_ID"), "local-project"),
		})
		if proxyErr != nil {
			_ = listener.Close()
			_ = proxyListener.Close()
			fmt.Fprintln(stderr, "agent-doctor: capture proxy configuration invalid")
			return 1
		}
		proxyHTTPServer = &http.Server{Handler: proxyHandler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 90 * time.Second}
		proxyURL = "http://" + proxyListener.Addr().String()
	}
	if once {
		_ = listener.Close()
		if proxyListener != nil {
			_ = proxyListener.Close()
		}
		fmt.Fprintln(stdout, url)
		if proxyURL != "" {
			fmt.Fprintln(stdout, "proxy:", proxyURL)
		}
		return 0
	}
	fmt.Fprintln(stdout, "Agent Doctor is ready:", url)
	if proxyURL != "" {
		fmt.Fprintln(stdout, "Live capture proxy:", proxyURL)
	}
	fmt.Fprintln(stdout, "Open this local URL in your browser. Press Ctrl+C to stop.")
	interrupt, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	serveErrors := make(chan error, 1)
	go func() { serveErrors <- service.Serve(listener) }()
	if proxyHTTPServer != nil {
		go func() {
			if serveErr := proxyHTTPServer.Serve(proxyListener); serveErr != nil && serveErr != http.ErrServerClosed {
				select {
				case serveErrors <- serveErr:
				default:
				}
			}
		}()
	}
	select {
	case <-interrupt.Done():
		shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := service.Shutdown(shutdownContext); err != nil {
			fmt.Fprintln(stderr, "agent-doctor: dashboard shutdown incomplete")
			return 1
		}
		if proxyHTTPServer != nil {
			_ = proxyHTTPServer.Shutdown(shutdownContext)
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

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
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

func runSetup(args []string, stdout, stderr io.Writer) int {
	if !hasFlag(args, "--json") {
		printUsage(stderr)
		return 2
	}
	home, err := agentDoctorHome()
	if err != nil {
		fmt.Fprintln(stderr, "agent-doctor: home directory unavailable")
		return 1
	}
	plan, err := installer.BuildCodexMCPPlan(home)
	if err != nil {
		fmt.Fprintln(stderr, "agent-doctor: setup plan unavailable")
		return 1
	}
	result := map[string]any{"status": "planned", "applied": false, "diff": plan.Diff(), "warnings": plan.Warnings, "detectedClients": plan.Detected}
	if hasFlag(args, "--yes") {
		applied, applyErr := installer.Apply(plan)
		if applyErr != nil {
			fmt.Fprintln(stderr, "agent-doctor: setup failed and prior configuration was restored")
			return 1
		}
		result["status"], result["applied"], result["result"] = "ready", true, applied
	}
	return encodeJSON(stdout, stderr, result)
}

func runUninstall(args []string, stdout, stderr io.Writer) int {
	if !hasFlag(args, "--json") {
		printUsage(stderr)
		return 2
	}
	if !hasFlag(args, "--yes") {
		return encodeJSON(stdout, stderr, map[string]any{"status": "planned", "removed": false, "message": "Pass --yes to remove only Agent Doctor-owned configuration blocks."})
	}
	home, err := agentDoctorHome()
	if err != nil || installer.UninstallCodexMCP(home) != nil {
		fmt.Fprintln(stderr, "agent-doctor: uninstall incomplete")
		return 1
	}
	return encodeJSON(stdout, stderr, map[string]any{"status": "ready", "removed": true})
}

func runLocalDataCommand(command string, stdout, stderr io.Writer) int {
	database, err := openLocalStore()
	if err != nil {
		fmt.Fprintln(stderr, "agent-doctor: local database unavailable")
		return 1
	}
	defer database.Close()
	summary, err := database.DashboardSummary(context.Background())
	if err != nil {
		fmt.Fprintln(stderr, "agent-doctor: local summary unavailable")
		return 1
	}
	snapshot, err := database.DashboardSnapshot(context.Background())
	if err != nil {
		fmt.Fprintln(stderr, "agent-doctor: local snapshot unavailable")
		return 1
	}
	var value any
	switch command {
	case "diagnose":
		value = map[string]any{"status": "ready", "evidenceEvents": summary.Events, "activeSessions": summary.ActiveSessions, "precision": summary.Precision, "limitations": []string{"A diagnosis requires a recorded task and never infers missing evidence."}}
	case "compare":
		value = map[string]any{"status": "ready", "comparisonCount": snapshot.ComparisonCount, "minimumComparableSamples": 15}
	case "context":
		value = map[string]any{"status": "ready", "memories": snapshot.Memories, "contentIncluded": false}
	case "costs":
		value = map[string]any{"status": "ready", "costs": snapshot.Costs}
	case "export":
		value = map[string]any{"status": "ready", "summary": summary, "snapshot": snapshot, "eventPayloadsIncluded": false}
	default:
		value = map[string]any{"status": "unavailable"}
	}
	return encodeJSON(stdout, stderr, value)
}

func runPause(args []string, stdout, stderr io.Writer) int {
	path, err := pauseStatePath()
	if err != nil {
		fmt.Fprintln(stderr, "agent-doctor: capture state unavailable")
		return 1
	}
	paused := !hasFlag(args, "--resume")
	if paused {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			fmt.Fprintln(stderr, "agent-doctor: capture state unavailable")
			return 1
		}
		if err := os.WriteFile(path, []byte("paused\n"), 0o600); err != nil {
			fmt.Fprintln(stderr, "agent-doctor: capture pause could not be persisted")
			return 1
		}
	} else if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		fmt.Fprintln(stderr, "agent-doctor: capture resume could not be persisted")
		return 1
	}
	return encodeJSON(stdout, stderr, map[string]any{"status": "ready", "paused": paused, "scope": "all Agent Doctor lifecycle capture on this device"})
}

func capturePaused() bool {
	path, err := pauseStatePath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func pauseStatePath() (string, error) {
	database, err := localDatabasePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(database), "capture.paused"), nil
}

func runForget(args []string, stdout, stderr io.Writer) int {
	if !hasFlag(args, "--yes") {
		return encodeJSON(stdout, stderr, map[string]any{"status": "planned", "forgotten": false, "message": "Pass --yes to delete the local Agent Doctor database."})
	}
	path, err := localDatabasePath()
	if err != nil {
		fmt.Fprintln(stderr, "agent-doctor: local database path unavailable")
		return 1
	}
	for _, target := range []string{path, path + "-wal", path + "-shm"} {
		if removeErr := os.Remove(target); removeErr != nil && !os.IsNotExist(removeErr) {
			fmt.Fprintln(stderr, "agent-doctor: local data removal incomplete")
			return 1
		}
	}
	return encodeJSON(stdout, stderr, map[string]any{"status": "ready", "forgotten": true})
}

func runWrapped(args []string, input io.Reader, stdout, stderr io.Writer) int {
	separator := -1
	for index, argument := range args {
		if argument == "--" {
			separator = index
			break
		}
	}
	if separator < 0 || separator+1 >= len(args) {
		fmt.Fprintln(stderr, "usage: agent-doctor run -- <command> [args...]")
		return 2
	}
	argv := args[separator+1:]
	command := exec.Command(argv[0], argv[1:]...)
	command.Stdin, command.Stdout, command.Stderr = input, stdout, stderr
	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return exitError.ExitCode()
		}
		fmt.Fprintln(stderr, "agent-doctor: wrapped command failed to start")
		return 1
	}
	return 0
}

func encodeJSON(stdout, stderr io.Writer, value any) int {
	if err := json.NewEncoder(stdout).Encode(value); err != nil {
		fmt.Fprintln(stderr, "agent-doctor: JSON encoding failed")
		return 1
	}
	return 0
}

func hasFlag(arguments []string, flag string) bool {
	for _, argument := range arguments {
		if argument == flag {
			return true
		}
	}
	return false
}

func agentDoctorHome() (string, error) {
	if value := os.Getenv("AGENT_DOCTOR_HOME"); value != "" {
		return value, nil
	}
	return os.UserHomeDir()
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
	path, err := localDatabasePath()
	if err != nil {
		return nil, err
	}
	return storage.Open(path)
}

func localDatabasePath() (string, error) {
	configDirectory := os.Getenv("AGENT_DOCTOR_CONFIG_DIR")
	if configDirectory == "" {
		var err error
		configDirectory, err = os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("resolve local data directory: %w", err)
		}
		configDirectory = filepath.Join(configDirectory, "AgentDoctor")
	}
	return filepath.Join(configDirectory, "doctor.db"), nil
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

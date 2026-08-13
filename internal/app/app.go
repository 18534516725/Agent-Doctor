package app

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/18534516725/Agent-Doctor/internal/mcp"
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
		if err := mcp.NewServer(developmentVersion, unavailableMCPBackend{}).Serve(context.Background(), input, stdout); err != nil {
			fmt.Fprintln(stderr, "agent-doctor MCP server failed")
			return 1
		}
		return 0
	}

	fmt.Fprintln(stderr, "usage: agent-doctor <command>")
	return 2
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

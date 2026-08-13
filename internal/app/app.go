package app

import (
	"fmt"
	"io"
)

const developmentVersion = "dev"

// Run dispatches CLI commands and returns a process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "version" {
		fmt.Fprintf(stdout, "agent-doctor %s\n", developmentVersion)
		return 0
	}

	fmt.Fprintln(stderr, "usage: agent-doctor <command>")
	return 2
}

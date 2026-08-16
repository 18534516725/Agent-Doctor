package handoff

import (
	"fmt"
	"strings"

	projectcontext "github.com/18534516725/Agent-Doctor/internal/context"
	"github.com/18534516725/Agent-Doctor/internal/privacy"
)

const capsuleProvenance = "local-sqlite-cross-client-handoff"

func Render(snapshot Snapshot, budget int) Capsule {
	if budget <= 0 || budget > DefaultBudget {
		budget = DefaultBudget
	}
	snapshot = sanitizeSnapshot(snapshot)
	lines := []string{"# Cross-client task handoff"}
	appendBounded := func(line string) bool {
		line = strings.TrimSpace(privacy.FilterText(line))
		if line == "" {
			return false
		}
		candidate := strings.Join(append(append([]string(nil), lines...), line), "\n")
		if projectcontext.EstimateTokens(candidate) > budget {
			return false
		}
		lines = append(lines, line)
		return true
	}

	appendBounded(fmt.Sprintf("Source: %s · session %s", fallback(snapshot.SourceClient, "unknown-client"), fallback(snapshot.SourceSessionID, "unknown-session")))
	if len(snapshot.Memories) > 0 {
		appendBounded("## Confirmed project memory")
		for _, memory := range snapshot.Memories {
			appendBounded(fmt.Sprintf("- %s (source: %s%s)", memory.Content, fallback(memory.SourceKind, "unknown"), sourceSuffix(memory.SourceID)))
		}
	}
	if strings.TrimSpace(snapshot.Goal) != "" {
		appendBounded("## Current task")
		appendBounded("Goal: " + snapshot.Goal)
	}
	if strings.TrimSpace(snapshot.LatestResult) != "" {
		appendBounded("Latest captured result: " + snapshot.LatestResult)
	}
	if len(snapshot.Limitations) > 0 {
		appendBounded("## Boundaries")
		for _, limitation := range snapshot.Limitations {
			appendBounded("- " + limitation)
		}
	}
	appendBounded("Continue from this state; verify it against the current workspace before changing files.")

	rendered := strings.Join(lines, "\n")
	return Capsule{
		Snapshot: snapshot, Rendered: rendered, TokenEstimate: projectcontext.EstimateTokens(rendered),
		Budget: budget, Provenance: capsuleProvenance,
	}
}

func sanitizeSnapshot(snapshot Snapshot) Snapshot {
	snapshot.SourceClient = sanitize(snapshot.SourceClient)
	snapshot.SourceSessionID = sanitize(snapshot.SourceSessionID)
	snapshot.Goal = sanitize(snapshot.Goal)
	snapshot.LatestResult = sanitize(snapshot.LatestResult)

	memories := make([]Memory, 0, len(snapshot.Memories))
	for _, memory := range snapshot.Memories {
		memory.Content = sanitize(memory.Content)
		memory.SourceKind = sanitize(memory.SourceKind)
		memory.SourceID = sanitize(memory.SourceID)
		memories = append(memories, memory)
	}
	snapshot.Memories = memories

	limitations := make([]string, 0, len(snapshot.Limitations))
	for _, limitation := range snapshot.Limitations {
		if limitation = sanitize(limitation); limitation != "" {
			limitations = append(limitations, limitation)
		}
	}
	snapshot.Limitations = limitations
	return snapshot
}

func sanitize(value string) string {
	return strings.TrimSpace(privacy.FilterText(value))
}

func fallback(value, replacement string) string {
	if strings.TrimSpace(value) == "" {
		return replacement
	}
	return strings.TrimSpace(value)
}

func sourceSuffix(sourceID string) string {
	if strings.TrimSpace(sourceID) == "" {
		return ""
	}
	return "; id: " + strings.TrimSpace(sourceID)
}

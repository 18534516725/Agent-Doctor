package app

import (
	"context"
	"fmt"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/events"
	"github.com/18534516725/Agent-Doctor/internal/guidance"
	"github.com/18534516725/Agent-Doctor/internal/mcp"
)

type localGuidanceStore interface {
	RuntimeGuidance(context.Context, string, time.Time) (guidance.Decision, error)
	LatestRuntimeGuidance(context.Context, string, time.Time) (guidance.Decision, error)
	GuidanceControlLevel(context.Context, string) (guidance.ControlLevel, error)
	RecordGuidanceDelivery(context.Context, guidance.DeliveryReceipt) error
	ListSessionEvents(context.Context, string) ([]events.Event, error)
}

func runtimeGuidanceEvidence(ctx context.Context, store localGuidanceStore, arguments map[string]any, now time.Time) (mcp.ToolEvidence, error) {
	sessionID, _ := arguments["sessionId"].(string)
	projectID, _ := arguments["projectId"].(string)
	var (
		decision guidance.Decision
		err      error
	)
	if sessionID != "" || projectID == "" {
		decision, err = store.RuntimeGuidance(ctx, sessionID, now)
	} else {
		decision, err = store.LatestRuntimeGuidance(ctx, projectID, now)
	}
	if err != nil {
		return mcp.ToolEvidence{}, err
	}
	if decision.ProjectID != "" {
		projectID = decision.ProjectID
	}
	level, err := store.GuidanceControlLevel(ctx, projectID)
	if err != nil {
		return mcp.ToolEvidence{}, err
	}

	summary := decision.Finding
	if decision.Kind == guidance.KindContinue || summary == "" {
		summary = "No evidence-backed intervention is required."
	}
	items := []mcp.EvidenceItem{
		{Label: "Decision", Value: string(decision.Kind)},
		{Label: "Severity", Value: string(decision.Severity)},
		{Label: "Control level", Value: string(level)},
	}
	if decision.Instruction != "" {
		items = append(items, mcp.EvidenceItem{Label: "Instruction", Value: decision.Instruction})
	}
	for _, eventID := range decision.Evidence {
		items = append(items, mcp.EvidenceItem{Label: "Evidence ID", Value: eventID})
	}
	for _, action := range decision.ProhibitedActions {
		items = append(items, mcp.EvidenceItem{Label: "Avoid", Value: action})
	}
	for _, check := range decision.Verification {
		items = append(items, mcp.EvidenceItem{Label: "Verification", Value: check})
	}
	receipt := guidance.DeliveryReceipt{
		SessionID: decision.SessionID, ProjectID: projectID, Client: "codex-mcp",
		DecisionID: decision.DecisionID, DecisionKind: decision.Kind,
		ControlLevel: level, DeliveredAt: now.UTC(),
	}
	if err := store.RecordGuidanceDelivery(ctx, receipt); err != nil {
		return mcp.ToolEvidence{}, err
	}
	items = append(items, mcp.EvidenceItem{Label: "Delivery receipt", Value: decision.DecisionID})

	precision := guidancePrecision(ctx, store, decision)
	return mcp.ToolEvidence{
		Summary:    summary,
		Items:      items,
		Provenance: "local-sqlite-deterministic-guidance",
		Precision:  precision,
		DataLimitNotes: []string{
			"Guidance is limited to normalized events supported by the connected client.",
			"Prompts, source code, commands, tool results, credentials, and absolute paths are not included.",
		},
	}, nil
}

func guidancePrecision(ctx context.Context, store localGuidanceStore, decision guidance.Decision) string {
	if len(decision.Evidence) == 0 {
		return "exact"
	}
	eventList, err := store.ListSessionEvents(ctx, decision.SessionID)
	if err != nil {
		return "unavailable"
	}
	wanted := make(map[string]bool, len(decision.Evidence))
	for _, eventID := range decision.Evidence {
		wanted[eventID] = true
	}
	found := 0
	precision := "exact"
	for _, event := range eventList {
		if !wanted[event.EventID] {
			continue
		}
		found++
		switch event.Precision {
		case events.PrecisionUnavailable:
			return "unavailable"
		case events.PrecisionEstimated:
			precision = "estimated"
		case events.PrecisionExact:
		default:
			return "unavailable"
		}
	}
	if found != len(wanted) {
		return "unavailable"
	}
	return precision
}

func guidanceUnavailable(tool string) mcp.ToolEvidence {
	return mcp.ToolEvidence{
		Summary:        "Runtime guidance is unavailable because the local store does not support it.",
		Items:          []mcp.EvidenceItem{{Label: "Requested tool", Value: tool}},
		Provenance:     "local-evidence-unavailable",
		Precision:      "unavailable",
		DataLimitNotes: []string{fmt.Sprintf("No compatible runtime guidance capability was found for %s.", tool)},
	}
}

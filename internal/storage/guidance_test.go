package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/conversations"
	"github.com/18534516725/Agent-Doctor/internal/events"
	"github.com/18534516725/Agent-Doctor/internal/guidance"
)

func TestRuntimeGuidancePersistsOneStableDecision(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	ctx := context.Background()
	started := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	for index := 0; index < 3; index++ {
		event := storageEvent()
		event.EventID = "failure-" + string(rune('1'+index))
		event.EventType = events.EventToolFailed
		event.Timestamp = started.Add(time.Duration(index) * time.Second)
		event.Payload = json.RawMessage(`{"hookEvent":"PostToolUseFailure","toolName":"Bash","toolInputFingerprint":"sha256:input","toolResultFingerprint":"sha256:result"}`)
		if err := database.InsertEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}

	first, err := database.RuntimeGuidance(ctx, "session-1", started.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	second, err := database.RuntimeGuidance(ctx, "session-1", started.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if first.Kind != guidance.KindRedirect || first.DecisionID == "" || second.DecisionID != first.DecisionID {
		t.Fatalf("unstable guidance: first=%+v second=%+v", first, second)
	}
	var count int
	if err := database.sql.QueryRowContext(ctx, "SELECT COUNT(*) FROM guidance_decisions").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("persisted decisions=%d want=1", count)
	}
}

func TestGuidanceControlLevelDefaultsPersistsAndValidates(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	ctx := context.Background()
	if err := database.InsertEvent(ctx, storageEvent()); err != nil {
		t.Fatal(err)
	}
	level, err := database.GuidanceControlLevel(ctx, "project-1")
	if err != nil || level != guidance.ControlGuide {
		t.Fatalf("default level=%q err=%v", level, err)
	}
	if err := database.SaveGuidanceControlLevel(ctx, "project-1", guidance.ControlGuard, time.Now()); err != nil {
		t.Fatal(err)
	}
	level, err = database.GuidanceControlLevel(ctx, "project-1")
	if err != nil || level != guidance.ControlGuard {
		t.Fatalf("saved level=%q err=%v", level, err)
	}
	if err := database.SaveGuidanceControlLevel(ctx, "project-1", guidance.ControlLevel("unbounded"), time.Now()); err == nil {
		t.Fatal("unsupported control level was accepted")
	}
}

func TestLatestRuntimeGuidanceEvaluatesNewestProjectSession(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	ctx := context.Background()
	now := time.Date(2026, 8, 15, 11, 0, 0, 0, time.UTC)
	for index := 0; index < 3; index++ {
		event := storageEvent()
		event.EventID = fmt.Sprintf("project-failure-%d", index)
		event.ProjectID = "project-current"
		event.SessionID = "session-current"
		event.EventType = events.EventToolFailed
		event.Timestamp = now.Add(time.Duration(index-3) * time.Second)
		event.Payload = json.RawMessage(`{"toolName":"exec","toolInputFingerprint":"sha256:same","toolResultFingerprint":"sha256:same-error"}`)
		if err := database.InsertEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}

	decision, err := database.LatestRuntimeGuidance(ctx, "project-current", now)
	if err != nil {
		t.Fatal(err)
	}
	if decision.ProjectID != "project-current" || decision.SessionID != "session-current" || decision.Kind != guidance.KindRedirect {
		t.Fatalf("project guidance resolved the wrong task: %+v", decision)
	}
}

func TestLatestRuntimeGuidanceIsQuietBeforeFirstProjectTool(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	now := time.Date(2026, 8, 15, 11, 0, 0, 0, time.UTC)
	record := conversations.Request{
		ID: "request-before-tool", SessionID: "session-before-tool", ProjectID: "project-before-tool",
		Client: events.ClientRef{Name: "codex", Version: "1"}, Model: events.ModelRef{DisplayName: "gpt-test"},
		Protocol: "codex-session-log", Method: "LOCAL", Path: "local-session", StartedAt: now.Add(-time.Second),
		Usage:    conversations.Usage{Precision: "unavailable", Provenance: "codex-session-log"},
		Cost:     conversations.Cost{Currency: "USD", Precision: "unavailable", Provenance: "codex-session-log"},
		Messages: []conversations.Message{{ID: "user-before-tool", Sequence: 0, Role: "user", CreatedAt: now.Add(-time.Second)}},
	}
	if err := database.SaveConversationRequest(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	decision, err := database.LatestRuntimeGuidance(context.Background(), "project-before-tool", now)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Kind != guidance.KindContinue || decision.ProjectID != "project-before-tool" || decision.SessionID != "session-before-tool" || decision.DecisionID == "" {
		t.Fatalf("quiet project decision=%+v", decision)
	}
}

func TestGuidanceDeliveryReceiptTracksReads(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	ctx := context.Background()
	event := storageEvent()
	if err := database.InsertEvent(ctx, event); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 15, 11, 0, 0, 0, time.UTC)
	receipt := guidance.DeliveryReceipt{
		SessionID: "session-1", ProjectID: "project-1", Client: "codex-mcp",
		DecisionID: "decision-1", DecisionKind: guidance.KindContinue,
		ControlLevel: guidance.ControlGuide, DeliveredAt: now,
	}
	if err := database.RecordGuidanceDelivery(ctx, receipt); err != nil {
		t.Fatal(err)
	}
	receipt.DeliveredAt = now.Add(time.Minute)
	if err := database.RecordGuidanceDelivery(ctx, receipt); err != nil {
		t.Fatal(err)
	}
	got, err := database.LatestGuidanceDelivery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.SessionID != receipt.SessionID || got.DeliveryCount != 2 || !got.DeliveredAt.Equal(receipt.DeliveredAt) {
		t.Fatalf("delivery receipt=%+v", got)
	}
}

func TestGuidanceStatusUsesLiveConversationActivity(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	now := time.Date(2026, 8, 15, 11, 0, 0, 0, time.UTC)
	record := conversations.Request{
		ID: "request-live", SessionID: "session-live", ProjectID: "project-live",
		Client: events.ClientRef{Name: "codex", Version: "1"}, Model: events.ModelRef{DisplayName: "gpt-test"},
		Protocol: "codex-session-log", Method: "LOCAL", Path: "local-session",
		StartedAt: now.Add(-time.Minute),
		Usage:     conversations.Usage{Precision: "unavailable", Provenance: "codex-session-log"},
		Cost:      conversations.Cost{Currency: "USD", Precision: "unavailable", Provenance: "codex-session-log"},
		Messages: []conversations.Message{{
			ID: "message-live", RequestID: "request-live", Sequence: 0, Role: "user", Content: "current user message", CreatedAt: now.Add(-time.Second),
		}},
	}
	if err := database.SaveConversationRequest(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	status, err := database.GuidanceStatus(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != guidance.StateObserving || status.Client != "codex" || status.LastEvidenceAt == nil || !status.LastEvidenceAt.Equal(now.Add(-time.Second)) {
		t.Fatalf("live guidance status=%+v", status)
	}
}

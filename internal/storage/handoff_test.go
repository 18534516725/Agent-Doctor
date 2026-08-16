package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/conversations"
	"github.com/18534516725/Agent-Doctor/internal/events"
	projectmemory "github.com/18534516725/Agent-Doctor/internal/memory"
)

func TestProjectHandoffSanitizesStructuredPreviewAndRenderedText(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	ctx := context.Background()
	now := time.Date(2026, 8, 16, 8, 0, 0, 0, time.UTC)
	projectID := "sanitized-project"
	secret := "sk-1234567890abcdefghijklmnop"

	created, err := database.CreateMemory(ctx, projectID, projectmemory.CreateInput{Content: "Authorization: Bearer 1234567890abcdefghijklmnop", SourceKind: "manual"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = database.UpdateMemory(ctx, projectID, created.ID, projectmemory.UpdateInput{State: "active"}, now); err != nil {
		t.Fatal(err)
	}
	completed := now.Add(time.Minute)
	if err = database.SaveConversationRequest(ctx, conversations.Request{
		ID: "request-safe", SessionID: "session-safe", ProjectID: projectID,
		Client: events.ClientRef{Name: "codex", Version: "1"}, Model: events.ModelRef{DisplayName: "test"},
		Protocol: "test", Method: "LOCAL", Path: "test", StatusCode: 200, StartedAt: now, CompletedAt: &completed,
		Usage: conversations.Usage{Precision: "unavailable"}, Cost: conversations.Cost{Precision: "unavailable"},
		Messages: []conversations.Message{
			{ID: "user-safe", Sequence: 0, Role: "user", Content: "repair login with " + secret, CreatedAt: now},
			{ID: "assistant-safe", Sequence: 1, Role: "assistant", Content: "result used " + secret, CreatedAt: completed},
		},
	}); err != nil {
		t.Fatal(err)
	}

	capsule, err := database.ProjectHandoff(ctx, []string{projectID}, 800, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(capsule)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), secret) || strings.Contains(string(payload), "1234567890abcdefghijklmnop") {
		t.Fatalf("structured handoff leaked credential-shaped content: %s", payload)
	}
	if !strings.Contains(string(payload), "[REDACTED:") {
		t.Fatalf("handoff did not preserve an explicit redaction marker: %s", payload)
	}
}

func TestProjectHandoffResolvesCrossClientIdentityAndUsesOnlyActiveMemory(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	ctx := context.Background()
	now := time.Date(2026, 8, 16, 9, 0, 0, 0, time.UTC)
	projectPath := "/work/shared-project"

	active, err := database.CreateMemory(ctx, projectPath, projectmemory.CreateInput{Content: "不得修改认证协议", SourceKind: "manual"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = database.UpdateMemory(ctx, projectPath, active.ID, projectmemory.UpdateInput{State: "active"}, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err = database.CreateMemory(ctx, projectPath, projectmemory.CreateInput{Content: "尚未确认，不能注入", SourceKind: "manual"}, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}

	completed := now.Add(4 * time.Minute)
	record := conversations.Request{
		ID: "codex-request-1", SessionID: "codex-session-1", ProjectID: projectPath,
		Client: events.ClientRef{Name: "codex", Version: "1"}, Model: events.ModelRef{DisplayName: "gpt-test"},
		Protocol: "codex-session-log", Method: "LOCAL", Path: "session-log", StatusCode: 200,
		StartedAt: now.Add(3 * time.Minute), CompletedAt: &completed,
		Usage: conversations.Usage{Precision: "unavailable", Provenance: "client-log"},
		Cost:  conversations.Cost{Currency: "USD", Precision: "unavailable", Provenance: "client-log"},
		Messages: []conversations.Message{
			{ID: "user-1", Sequence: 0, Role: "user", Content: "完成登录模块并修复剩余测试", CreatedAt: now.Add(3 * time.Minute)},
			{ID: "assistant-1", Sequence: 1, Role: "assistant", Content: "后端接口已完成，前端还有两个测试失败。", CreatedAt: completed},
		},
	}
	if err := database.SaveConversationRequest(ctx, record); err != nil {
		t.Fatal(err)
	}
	latestCompleted := now.Add(4*time.Minute + 40*time.Second)
	latest := record
	latest.ID = "codex-request-2"
	latest.StartedAt = now.Add(4*time.Minute + 30*time.Second)
	latest.CompletedAt = &latestCompleted
	latest.Messages = []conversations.Message{
		{ID: "user-2", Sequence: 0, Role: "user", Content: "<recommended_plugins>host-only metadata</recommended_plugins>", CreatedAt: latest.StartedAt},
		{ID: "assistant-2", Sequence: 1, Role: "assistant", Content: "刚完成接力实现，正在运行最终验证。", CreatedAt: latestCompleted},
	}
	if err := database.SaveConversationRequest(ctx, latest); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(projectPath))
	claudeProjectID := "sha256:" + hex.EncodeToString(digest[:])
	capsule, err := database.ProjectHandoff(ctx, []string{claudeProjectID, projectPath}, 800, now.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if capsule.ProjectID != projectPath || capsule.SourceClient != "codex" || capsule.SourceSessionID != "codex-session-1" {
		t.Fatalf("wrong source: %+v", capsule)
	}
	for _, expected := range []string{"完成登录模块", "刚完成接力实现", "不得修改认证协议"} {
		if !strings.Contains(capsule.Rendered, expected) {
			t.Fatalf("handoff missing %q:\n%s", expected, capsule.Rendered)
		}
	}
	if strings.Contains(capsule.Rendered, "尚未确认") || len(capsule.Memories) != 1 {
		t.Fatalf("candidate memory entered handoff: %+v", capsule)
	}
}

func TestHandoffDeliveryReceiptIsVisibleOnNextPreview(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	ctx := context.Background()
	now := time.Date(2026, 8, 16, 10, 0, 0, 0, time.UTC)
	created, err := database.CreateMemory(ctx, "project-1", projectmemory.CreateInput{Content: "confirmed project constraint", SourceKind: "manual"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.UpdateMemory(ctx, "project-1", created.ID, projectmemory.UpdateInput{State: "active"}, now); err != nil {
		t.Fatal(err)
	}
	capsule, err := database.ProjectHandoff(ctx, []string{"project-1"}, 800, now)
	if err != nil {
		t.Fatal(err)
	}
	if capsule.Memories == nil {
		t.Fatal("empty confirmed memory must serialize as an empty array, not null")
	}
	if err := database.RecordHandoffDelivery(ctx, capsule, "claude-code", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	preview, err := database.ProjectHandoff(ctx, []string{"project-1"}, 800, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if preview.LastDelivery == nil || preview.LastDelivery.TargetClient != "claude-code" || preview.LastDelivery.DeliveredAt != now.Add(time.Minute) {
		t.Fatalf("delivery missing: %+v", preview.LastDelivery)
	}
}

func TestProjectHandoffRejectsProjectWithoutTaskOrConfirmedMemory(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	ctx := context.Background()
	now := time.Date(2026, 8, 16, 11, 0, 0, 0, time.UTC)
	if _, err := database.CreateMemory(ctx, "empty-project", projectmemory.CreateInput{Content: "unconfirmed candidate", SourceKind: "manual"}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ProjectHandoff(ctx, []string{"empty-project"}, 800, now); err == nil {
		t.Fatal("project without a captured task or confirmed memory produced a fake handoff")
	}
}

func TestBoundedHandoffTextRemovesCodexHostContextBeforeUserGoal(t *testing.T) {
	input := `<recommended_plugins>
- GitHub (github@example)
</recommended_plugins># AGENTS.md instructions

<INSTRUCTIONS>
Always run guidance.
</INSTRUCTIONS>
<environment_context>
  <cwd>/work/private</cwd>
</environment_context>
那你这块可以全部实现了吗`

	if got := boundedHandoffText(input, 1600); got != "那你这块可以全部实现了吗" {
		t.Fatalf("host context polluted task goal: %q", got)
	}
}

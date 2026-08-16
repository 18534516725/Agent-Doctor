package handoff

import (
	"strings"
	"testing"
	"time"
)

func TestRenderBuildsBoundedProvenanceLabelledCrossClientHandoff(t *testing.T) {
	now := time.Date(2026, 8, 16, 9, 30, 0, 0, time.UTC)
	capsule := Render(Snapshot{
		ProjectID:       "/work/project",
		SourceClient:    "codex",
		SourceSessionID: "codex-session-1",
		Goal:            "完成登录模块并修复剩余测试",
		LatestResult:    "后端接口已经完成，前端仍有两个测试失败。 Authorization: Bearer synthetic-secret",
		Memories: []Memory{
			{Content: "不得修改认证协议", SourceKind: "manual", SourceID: "memory-1"},
			{Content: "完成前运行 go test ./...", SourceKind: "conversation-message", SourceID: "message-2"},
		},
		GeneratedAt: now,
		Limitations: []string{"仅包含已确认记忆和最近一次任务快照。"},
	}, 800)

	for _, expected := range []string{
		"Cross-client task handoff", "codex", "codex-session-1", "完成登录模块",
		"后端接口已经完成", "不得修改认证协议", "go test ./...", "仅包含已确认记忆",
	} {
		if !strings.Contains(capsule.Rendered, expected) {
			t.Fatalf("capsule missing %q:\n%s", expected, capsule.Rendered)
		}
	}
	if strings.Contains(capsule.Rendered, "synthetic-secret") {
		t.Fatalf("capsule leaked credential: %s", capsule.Rendered)
	}
	if strings.Contains(capsule.LatestResult, "synthetic-secret") {
		t.Fatalf("structured capsule leaked credential: %+v", capsule)
	}
	if capsule.TokenEstimate > 800 || capsule.Budget != 800 || capsule.Provenance != "local-sqlite-cross-client-handoff" {
		t.Fatalf("unexpected capsule metadata: %+v", capsule)
	}
}

func TestRenderDropsLowerPriorityContextBeforeConfirmedMemory(t *testing.T) {
	capsule := Render(Snapshot{
		SourceClient: "codex",
		Goal:         strings.Repeat("long goal ", 200),
		LatestResult: strings.Repeat("long result ", 200),
		Memories:     []Memory{{Content: "必须保留的项目约束", SourceKind: "manual", SourceID: "memory-1"}},
	}, 80)

	if capsule.TokenEstimate > 80 {
		t.Fatalf("token budget exceeded: %d", capsule.TokenEstimate)
	}
	if !strings.Contains(capsule.Rendered, "必须保留的项目约束") {
		t.Fatalf("confirmed memory was dropped:\n%s", capsule.Rendered)
	}
}

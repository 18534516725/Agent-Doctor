package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	projectmemory "github.com/18534516725/Agent-Doctor/internal/memory"
)

func TestMemoryWorkflowRequiresConfirmationBeforeActivation(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	ctx := context.Background()
	now := time.Now().UTC()
	created, err := database.CreateMemory(ctx, "project-memory", projectmemory.CreateInput{Content: "项目必须先运行测试", SourceKind: "manual"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if created.State != "candidate" {
		t.Fatalf("created=%+v", created)
	}
	active, err := database.UpdateMemory(ctx, "project-memory", created.ID, projectmemory.UpdateInput{State: "active"}, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if active.State != "active" {
		t.Fatalf("active=%+v", active)
	}
	disabled, err := database.UpdateMemory(ctx, "project-memory", created.ID, projectmemory.UpdateInput{Content: "完成前必须运行测试", State: "disabled"}, now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if disabled.State != "disabled" || disabled.Content != "完成前必须运行测试" {
		t.Fatalf("disabled=%+v", disabled)
	}
	items, err := database.ListMemories(ctx, "project-memory", "")
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if err := database.DeleteMemory(ctx, "project-memory", created.ID, now.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}
	items, err = database.ListMemories(ctx, "project-memory", "")
	if err != nil || len(items) != 0 {
		t.Fatalf("deleted items=%+v err=%v", items, err)
	}
}

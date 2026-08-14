package storage

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

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

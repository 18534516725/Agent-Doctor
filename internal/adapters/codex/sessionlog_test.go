package codex

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/conversations"
)

func TestParseSessionLogBuildsCompleteTurns(t *testing.T) {
	input := strings.Join([]string{
		`{"timestamp":"2026-08-14T10:00:00Z","type":"session_meta","payload":{"id":"session-1","cwd":"/work/project","cli_version":"1.2.3"}}`,
		`{"timestamp":"2026-08-14T10:00:01Z","type":"event_msg","payload":{"type":"task_started","turn_id":"turn-1","started_at":1786672801}}`,
		`{"timestamp":"2026-08-14T10:00:01Z","type":"turn_context","payload":{"turn_id":"turn-1","model":"gpt-test","cwd":"/work/project"}}`,
		`{"timestamp":"2026-08-14T10:00:02Z","type":"response_item","payload":{"type":"message","id":"user-1","role":"user","content":[{"type":"input_text","text":"请检查这个项目"}]}}`,
		`{"timestamp":"2026-08-14T10:00:03Z","type":"response_item","payload":{"type":"function_call","id":"tool-1","name":"read_file","arguments":"{\"path\":\"README.md\"}"}}`,
		`{"timestamp":"2026-08-14T10:00:04Z","type":"response_item","payload":{"type":"message","id":"assistant-1","role":"assistant","content":[{"type":"output_text","text":"已完成检查"}]}}`,
		`{"timestamp":"2026-08-14T10:00:05Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":120,"output_tokens":30,"cached_input_tokens":20,"reasoning_output_tokens":5}}}}`,
		`{"timestamp":"2026-08-14T10:00:06Z","type":"event_msg","payload":{"type":"task_complete","turn_id":"turn-1","completed_at":1786672806,"duration_ms":5000,"time_to_first_token_ms":850}}`,
	}, "\n") + "\n"

	requests, err := ParseSessionLog(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 {
		t.Fatalf("requests=%d", len(requests))
	}
	record := requests[0]
	if record.ID != "codex-turn-1" || record.SessionID != "session-1" || record.ProjectID != "/work/project" {
		t.Fatalf("identities=%+v", record)
	}
	if record.Model.DisplayName != "gpt-test" || record.Client.Name != "codex" || record.Client.Version != "1.2.3" {
		t.Fatalf("client/model=%+v/%+v", record.Client, record.Model)
	}
	if len(record.Messages) != 3 || record.Messages[0].Content != "请检查这个项目" || record.Messages[1].ToolName != "read_file" || record.Messages[2].Content != "已完成检查" {
		t.Fatalf("messages=%+v", record.Messages)
	}
	if record.Usage.InputTokens == nil || *record.Usage.InputTokens != 120 || record.Usage.OutputTokens == nil || *record.Usage.OutputTokens != 30 {
		t.Fatalf("usage=%+v", record.Usage)
	}
	if record.FirstByteMS != 850 || record.DurationMS != 5000 || record.CompletedAt == nil {
		t.Fatalf("timing=%+v", record)
	}
}

func TestParseCurrentSessionLogKeepsOnlyLatestTurn(t *testing.T) {
	input := strings.Join([]string{
		`{"timestamp":"2026-08-14T09:00:00Z","type":"session_meta","payload":{"id":"session-1","cwd":"/work/project","cli_version":"1"}}`,
		`{"timestamp":"2026-08-14T09:00:01Z","type":"event_msg","payload":{"type":"task_started","turn_id":"old-turn","started_at":"2026-08-14T09:00:01Z"}}`,
		`{"timestamp":"2026-08-14T09:00:02Z","type":"response_item","payload":{"type":"message","id":"old-user","role":"user","content":[{"type":"input_text","text":"旧消息"}]}}`,
		`{"timestamp":"2026-08-14T10:00:01Z","type":"event_msg","payload":{"type":"task_started","turn_id":"current-turn","started_at":"2026-08-14T10:00:01Z"}}`,
		`{"timestamp":"2026-08-14T10:00:02Z","type":"turn_context","payload":{"turn_id":"current-turn","model":"gpt-current","cwd":"/work/project"}}`,
		`{"timestamp":"2026-08-14T10:00:03Z","type":"response_item","payload":{"type":"message","id":"current-user","role":"user","content":[{"type":"input_text","text":"当前消息"}]}}`,
	}, "\n") + "\n"

	requests, err := ParseCurrentSessionLog(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 || requests[0].ID != "codex-current-turn" || requests[0].Messages[0].Content != "当前消息" {
		t.Fatalf("requests=%+v", requests)
	}
}

func TestTailHasTaskStartedAcceptsFormattedJSON(t *testing.T) {
	tail := []byte("{\"type\": \"event_msg\", \"payload\": {\"type\": \"task_started\", \"turn_id\": \"turn-1\"}}\n")
	if !tailHasTaskStarted(tail) {
		t.Fatal("formatted task_started event was not detected")
	}
}

func TestParseCurrentSessionFileReadsHeadMetadataAndLatestTail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	content := strings.Join([]string{
		`{"timestamp":"2026-08-14T09:00:00Z","type":"session_meta","payload":{"id":"session-file","cwd":"/work/file","cli_version":"2"}}`,
		`{"timestamp":"2026-08-14T09:00:01Z","type":"event_msg","payload":{"type":"task_started","turn_id":"old-file-turn","started_at":"2026-08-14T09:00:01Z"}}`,
		`{"timestamp":"2026-08-14T09:00:02Z","type":"response_item","payload":{"type":"message","id":"old-file-user","role":"user","content":[{"type":"input_text","text":"旧文件消息"}]}}`,
		`{"timestamp":"2026-08-14T10:00:01Z","type":"event_msg","payload":{"type":"task_started","turn_id":"latest-file-turn","started_at":"2026-08-14T10:00:01Z"}}`,
		`{"timestamp":"2026-08-14T10:00:02Z","type":"response_item","payload":{"type":"message","id":"latest-file-user","role":"user","content":[{"type":"input_text","text":"最新文件消息"}]}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	requests, err := ParseCurrentSessionFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 || requests[0].SessionID != "session-file" || requests[0].Messages[0].Content != "最新文件消息" {
		t.Fatalf("requests=%+v", requests)
	}
}

func TestSessionWatcherImportsCurrentFileAndFollowsAppends(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "2026", "08", "14", "rollout.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	initial := strings.Join([]string{
		`{"timestamp":"2026-08-14T10:00:00Z","type":"session_meta","payload":{"id":"session-live","cwd":"/work/live","cli_version":"1"}}`,
		`{"timestamp":"2026-08-14T10:00:01Z","type":"event_msg","payload":{"type":"task_started","turn_id":"turn-live","started_at":"2026-08-14T10:00:01Z"}}`,
		`{"timestamp":"2026-08-14T10:00:02Z","type":"response_item","payload":{"type":"message","id":"user-live","role":"user","content":[{"type":"input_text","text":"实时消息"}]}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &watcherStore{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watcher := NewSessionWatcher(SessionWatcherConfig{
		Root: root, Store: store, PollInterval: 10 * time.Millisecond, RecentWindow: time.Hour,
		Now: time.Now,
	})
	done := make(chan error, 1)
	go func() { done <- watcher.Run(ctx) }()
	waitForSavedRequest(t, store, 1)

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = file.WriteString(`{"timestamp":"2026-08-14T10:00:03Z","type":"response_item","payload":{"type":"message","id":"assistant-live","role":"assistant","content":[{"type":"output_text","text":"实时回复"}]}}` + "\n")
	_ = file.Close()
	if err != nil {
		t.Fatal(err)
	}
	waitForMessageCount(t, store, 2)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("watcher did not stop")
	}
	if connection := store.connection(); connection.State != "active" || connection.Key != "codex" {
		t.Fatalf("connection=%+v", connection)
	}
}

func TestSessionWatcherMarksCodexConnectedBeforeFirstMessage(t *testing.T) {
	root := t.TempDir()
	store := &watcherStore{}
	ctx, cancel := context.WithCancel(context.Background())
	watcher := NewSessionWatcher(SessionWatcherConfig{Root: root, Store: store, PollInterval: 10 * time.Millisecond})
	done := make(chan error, 1)
	go func() { done <- watcher.Run(ctx) }()
	deadline := time.Now().Add(time.Second)
	for store.connection().State == "" && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	connection := store.connection()
	if connection.State != "connected" || !strings.Contains(connection.Detail, "无需重启") {
		t.Fatalf("connection=%+v", connection)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestParseSessionLogCapturesCustomToolInputAndOutput(t *testing.T) {
	input := strings.Join([]string{
		`{"timestamp":"2026-08-14T10:00:00Z","type":"session_meta","payload":{"id":"session-tools"}}`,
		`{"timestamp":"2026-08-14T10:00:01Z","type":"event_msg","payload":{"type":"task_started","turn_id":"turn-tools","started_at":1786672801}}`,
		`{"timestamp":"2026-08-14T10:00:02Z","type":"response_item","payload":{"type":"custom_tool_call","call_id":"call-1","name":"apply_patch","input":"*** Begin Patch"}}`,
		`{"timestamp":"2026-08-14T10:00:03Z","type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"call-1","output":"Done"}}`,
	}, "\n") + "\n"
	requests, err := ParseSessionLog(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 || len(requests[0].Messages) != 2 {
		t.Fatalf("requests=%+v", requests)
	}
	if requests[0].Messages[0].ToolPayloadJSON != "*** Begin Patch" || requests[0].Messages[1].ToolPayloadJSON != "Done" {
		t.Fatalf("messages=%+v", requests[0].Messages)
	}
}

func TestParseSessionLogDerivesTimingFromEnvelopeTimestamps(t *testing.T) {
	input := strings.Join([]string{
		`{"timestamp":"2026-08-14T10:00:00Z","type":"session_meta","payload":{"id":"session-timing"}}`,
		`{"timestamp":"2026-08-14T10:00:01Z","type":"event_msg","payload":{"type":"task_started","turn_id":"turn-timing","started_at":1786672801}}`,
		`{"timestamp":"2026-08-14T10:00:03Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"first answer"}]}}`,
		`{"timestamp":"2026-08-14T10:00:06Z","type":"event_msg","payload":{"type":"task_complete","turn_id":"turn-timing","completed_at":1786672806}}`,
	}, "\n") + "\n"
	requests, err := ParseSessionLog(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 || requests[0].FirstByteMS != 2000 || requests[0].DurationMS != 5000 {
		t.Fatalf("requests=%+v", requests)
	}
}

func TestParseSessionLogAccumulatesPerCallTokenUsage(t *testing.T) {
	input := strings.Join([]string{
		`{"timestamp":"2026-08-14T10:00:00Z","type":"session_meta","payload":{"id":"session-usage"}}`,
		`{"timestamp":"2026-08-14T10:00:01Z","type":"event_msg","payload":{"type":"task_started","turn_id":"turn-usage"}}`,
		`{"timestamp":"2026-08-14T10:00:02Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}}`,
		`{"timestamp":"2026-08-14T10:00:03Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":10,"output_tokens":2,"cached_input_tokens":1,"reasoning_output_tokens":1},"total_token_usage":{"input_tokens":10,"output_tokens":2,"cached_input_tokens":1,"reasoning_output_tokens":1}}}}`,
		`{"timestamp":"2026-08-14T10:00:03.500Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":10,"output_tokens":2,"cached_input_tokens":1,"reasoning_output_tokens":1},"total_token_usage":{"input_tokens":10,"output_tokens":2,"cached_input_tokens":1,"reasoning_output_tokens":1}}}}`,
		`{"timestamp":"2026-08-14T10:00:04Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":20,"output_tokens":3,"cached_input_tokens":2,"reasoning_output_tokens":2},"total_token_usage":{"input_tokens":30,"output_tokens":5,"cached_input_tokens":3,"reasoning_output_tokens":3}}}}`,
	}, "\n") + "\n"
	requests, err := ParseSessionLog(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	usage := requests[0].Usage
	if usage.InputTokens == nil || *usage.InputTokens != 30 || *usage.OutputTokens != 5 || *usage.CachedTokens != 3 || *usage.ReasoningTokens != 3 {
		t.Fatalf("usage=%+v", usage)
	}
}

func TestTokenSnapshotBeforeCaptureCutoffIsOnlyABaseline(t *testing.T) {
	input := strings.Join([]string{
		`{"timestamp":"2026-08-14T10:00:00Z","type":"session_meta","payload":{"id":"session-cutoff-token"}}`,
		`{"timestamp":"2026-08-14T10:00:01Z","type":"event_msg","payload":{"type":"task_started","turn_id":"turn-cutoff-token"}}`,
		`{"timestamp":"2026-08-14T10:00:03Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":10,"output_tokens":2},"total_token_usage":{"input_tokens":10,"output_tokens":2}}}}`,
		`{"timestamp":"2026-08-14T10:00:06Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"resumed"}]}}`,
		`{"timestamp":"2026-08-14T10:00:06.500Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":10,"output_tokens":2},"total_token_usage":{"input_tokens":10,"output_tokens":2}}}}`,
		`{"timestamp":"2026-08-14T10:00:07Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":4,"output_tokens":1},"total_token_usage":{"input_tokens":14,"output_tokens":3}}}}`,
	}, "\n") + "\n"
	requests, err := parseCurrentSessionLogAfter(strings.NewReader(input), time.Date(2026, 8, 14, 10, 0, 5, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 || requests[0].Usage.InputTokens == nil || *requests[0].Usage.InputTokens != 4 || *requests[0].Usage.OutputTokens != 1 {
		t.Fatalf("requests=%+v", requests)
	}
}

func TestParseCurrentSessionFileAcceptsToolOutputLargerThanScannerLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.jsonl")
	largePayload := strings.Repeat("x", 17*1024*1024)
	content := strings.Join([]string{
		`{"timestamp":"2026-08-14T10:00:00Z","type":"session_meta","payload":{"id":"session-large"}}`,
		`{"timestamp":"2026-08-14T10:00:01Z","type":"event_msg","payload":{"type":"task_started","turn_id":"turn-large"}}`,
		`{"timestamp":"2026-08-14T10:00:02Z","type":"response_item","payload":{"type":"custom_tool_call","name":"large_tool","input":"` + largePayload + `"}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	requests, err := ParseCurrentSessionFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 || len(requests[0].Messages) != 1 || len(requests[0].Messages[0].ToolPayloadJSON) != len(largePayload) {
		t.Fatalf("large payload was not preserved")
	}
}

func TestSessionWatcherHonorsPausedCaptureAndPromptSetting(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "rollout.jsonl")
	content := strings.Join([]string{
		`{"timestamp":"2026-08-14T10:00:00Z","type":"session_meta","payload":{"id":"session-policy"}}`,
		`{"timestamp":"2026-08-14T10:00:01Z","type":"event_msg","payload":{"type":"task_started","turn_id":"turn-policy"}}`,
		`{"timestamp":"2026-08-14T10:00:02Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"private prompt"}]}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	pausedStore := &watcherStore{}
	paused := NewSessionWatcher(SessionWatcherConfig{Root: root, Store: pausedStore, CapturePolicy: func(context.Context) (bool, bool) { return false, false }})
	pausedCtx, pausedCancel := context.WithCancel(context.Background())
	pausedDone := make(chan error, 1)
	go func() { pausedDone <- paused.Run(pausedCtx) }()
	time.Sleep(30 * time.Millisecond)
	pausedCancel()
	if err := <-pausedDone; err != nil || pausedStore.count() != 0 {
		t.Fatalf("paused err=%v requests=%d", err, pausedStore.count())
	}

	metadataStore := &watcherStore{}
	metadata := NewSessionWatcher(SessionWatcherConfig{Root: root, Store: metadataStore, CapturePolicy: func(context.Context) (bool, bool) { return true, false }})
	metadataCtx, metadataCancel := context.WithCancel(context.Background())
	metadataDone := make(chan error, 1)
	go func() { metadataDone <- metadata.Run(metadataCtx) }()
	waitForSavedRequest(t, metadataStore, 1)
	metadataCancel()
	if err := <-metadataDone; err != nil || metadataStore.messageCount() != 0 {
		t.Fatalf("metadata err=%v messages=%d", err, metadataStore.messageCount())
	}
}

func TestSessionWatcherReturnsToConnectedAfterCompletedTurn(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "rollout.jsonl")
	content := strings.Join([]string{
		`{"timestamp":"2026-08-14T10:00:00Z","type":"session_meta","payload":{"id":"session-complete"}}`,
		`{"timestamp":"2026-08-14T10:00:01Z","type":"event_msg","payload":{"type":"task_started","turn_id":"turn-complete"}}`,
		`{"timestamp":"2026-08-14T10:00:02Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"done"}]}}`,
		`{"timestamp":"2026-08-14T10:00:03Z","type":"event_msg","payload":{"type":"task_complete","turn_id":"turn-complete"}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &watcherStore{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- NewSessionWatcher(SessionWatcherConfig{Root: root, Store: store}).Run(ctx) }()
	waitForSavedRequest(t, store, 1)
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if connection := store.connection(); connection.State != "connected" {
		t.Fatalf("connection=%+v", connection)
	}
}

func TestSessionWatcherKeepsActiveWhenAnyRecentFileHasOpenTurn(t *testing.T) {
	root := t.TempDir()
	completedPath := filepath.Join(root, "completed.jsonl")
	activePath := filepath.Join(root, "active.jsonl")
	completed := strings.Join([]string{
		`{"timestamp":"2026-08-14T10:00:00Z","type":"session_meta","payload":{"id":"session-completed"}}`,
		`{"timestamp":"2026-08-14T10:00:01Z","type":"event_msg","payload":{"type":"task_started","turn_id":"turn-completed"}}`,
		`{"timestamp":"2026-08-14T10:00:02Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"done"}]}}`,
		`{"timestamp":"2026-08-14T10:00:03Z","type":"event_msg","payload":{"type":"task_complete","turn_id":"turn-completed"}}`,
	}, "\n") + "\n"
	active := strings.Join([]string{
		`{"timestamp":"2026-08-14T10:01:00Z","type":"session_meta","payload":{"id":"session-active"}}`,
		`{"timestamp":"2026-08-14T10:01:01Z","type":"event_msg","payload":{"type":"task_started","turn_id":"turn-active"}}`,
		`{"timestamp":"2026-08-14T10:01:02Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"working"}]}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(completedPath, []byte(completed), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(activePath, []byte(active), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	_ = os.Chtimes(completedPath, now.Add(-time.Minute), now.Add(-time.Minute))
	_ = os.Chtimes(activePath, now, now)
	store := &watcherStore{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- NewSessionWatcher(SessionWatcherConfig{Root: root, Store: store}).Run(ctx) }()
	waitForSavedRequest(t, store, 2)
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if connection := store.connection(); connection.State != "active" {
		t.Fatalf("connection=%+v", connection)
	}
}

func TestSessionWatcherDoesNotBackfillMessagesCreatedWhilePaused(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "rollout.jsonl")
	initial := strings.Join([]string{
		`{"timestamp":"2026-08-14T10:00:00Z","type":"session_meta","payload":{"id":"session-resume"}}`,
		`{"timestamp":"2026-08-14T10:00:01Z","type":"event_msg","payload":{"type":"task_started","turn_id":"turn-resume"}}`,
		`{"timestamp":"2026-08-14T10:00:02Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"before pause"}]}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	var enabled atomic.Bool
	enabled.Store(true)
	var clock atomic.Int64
	clock.Store(time.Date(2026, 8, 14, 10, 0, 3, 0, time.UTC).UnixNano())
	store := &watcherStore{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watcher := NewSessionWatcher(SessionWatcherConfig{
		Root: root, Store: store, PollInterval: 10 * time.Millisecond, RecentWindow: 365 * 24 * time.Hour,
		Now:           func() time.Time { return time.Unix(0, clock.Load()).UTC() },
		CapturePolicy: func(context.Context) (bool, bool) { return enabled.Load(), true },
	})
	done := make(chan error, 1)
	go func() { done <- watcher.Run(ctx) }()
	waitForSavedRequest(t, store, 1)

	enabled.Store(false)
	clock.Store(time.Date(2026, 8, 14, 10, 0, 5, 0, time.UTC).UnixNano())
	appendSessionLine(t, path, `{"timestamp":"2026-08-14T10:00:04Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"secret while paused"}]}}`)
	appendSessionLine(t, path, `{"timestamp":"2026-08-14T10:00:04.500Z","type":"event_msg","payload":{"type":"task_complete","turn_id":"turn-resume"}}`)
	time.Sleep(40 * time.Millisecond)
	if connection := store.connection(); connection.State != "connected" || !strings.Contains(connection.Detail, "暂停") {
		t.Fatalf("paused connection=%+v", connection)
	}
	enabled.Store(true)
	clock.Store(time.Date(2026, 8, 14, 10, 0, 10, 0, time.UTC).UnixNano())
	time.Sleep(30 * time.Millisecond)
	if connection := store.connection(); connection.State != "connected" || strings.Contains(connection.Detail, "暂停") {
		t.Fatalf("resumed connection=%+v", connection)
	}
	appendSessionLine(t, path, `{"timestamp":"2026-08-14T10:00:10.500Z","type":"event_msg","payload":{"type":"task_started","turn_id":"turn-after-resume"}}`)
	appendSessionLine(t, path, `{"timestamp":"2026-08-14T10:00:11Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"after resume"}]}}`)
	deadline := time.Now().Add(time.Second)
	for !strings.Contains(store.messageText(), "after resume") && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	text := store.messageText()
	if strings.Contains(text, "secret while paused") || !strings.Contains(text, "after resume") {
		t.Fatalf("captured messages=%q", text)
	}
}

func appendSessionLine(t *testing.T, path, line string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := file.WriteString(line + "\n")
	closeErr := file.Close()
	if writeErr != nil {
		t.Fatal(writeErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}

func TestParseCurrentSessionFileFromLocalFixture(t *testing.T) {
	path := os.Getenv("AGENT_DOCTOR_CODEX_SESSION_FIXTURE")
	if path == "" {
		t.Skip("set AGENT_DOCTOR_CODEX_SESSION_FIXTURE for a local integration check")
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	started := time.Now()
	requests, err := ParseCurrentSessionFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) == 0 {
		t.Fatal("current Codex session produced no requests")
	}
	t.Logf("parsed %d current requests with %d messages in %s", len(requests), len(requests[0].Messages), time.Since(started))
}

type watcherStore struct {
	mu          sync.Mutex
	requests    map[string]conversations.Request
	connections []conversations.ClientConnection
}

func (store *watcherStore) SaveConversationRequest(_ context.Context, record conversations.Request) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.requests == nil {
		store.requests = map[string]conversations.Request{}
	}
	store.requests[record.ID] = record
	return nil
}

func (store *watcherStore) UpsertClientConnection(_ context.Context, connection conversations.ClientConnection) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.connections = append(store.connections, connection)
	return nil
}

func (store *watcherStore) count() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return len(store.requests)
}

func (store *watcherStore) messageCount() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	for _, record := range store.requests {
		return len(record.Messages)
	}
	return 0
}

func (store *watcherStore) connection() conversations.ClientConnection {
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.connections) == 0 {
		return conversations.ClientConnection{}
	}
	return store.connections[len(store.connections)-1]
}

func (store *watcherStore) messageText() string {
	store.mu.Lock()
	defer store.mu.Unlock()
	parts := []string{}
	for _, record := range store.requests {
		for _, message := range record.Messages {
			parts = append(parts, message.Content, message.ToolPayloadJSON)
		}
	}
	return strings.Join(parts, "\n")
}

func waitForSavedRequest(t *testing.T, store *watcherStore, count int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if store.count() == count {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("saved requests=%d want=%d", store.count(), count)
}

func waitForMessageCount(t *testing.T, store *watcherStore, count int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if store.messageCount() == count {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("message count=%d want=%d", store.messageCount(), count)
}

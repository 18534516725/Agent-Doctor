package storage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/conversations"
	"github.com/18534516725/Agent-Doctor/internal/events"
)

func storageEvent() events.Event {
	return events.Event{
		SchemaVersion: 1,
		EventID:       "event-1",
		SessionID:     "session-1",
		ProjectID:     "project-1",
		Timestamp:     time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC),
		Client:        events.ClientRef{Name: "codex", Version: "1.0.0"},
		Model:         events.ModelRef{DisplayName: "public-model"},
		EventType:     events.EventUserPrompted,
		Payload:       json.RawMessage(`{"prompt":"Authorization: Bearer synthetic-value"}`),
		Provenance:    "client-event",
		Precision:     events.PrecisionExact,
	}
}

func TestOpenMigratesAndPersistsFilteredEvent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doctor.db")
	database, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if got := database.SchemaVersion(); got != 5 {
		t.Fatalf("schema=%d want=5", got)
	}
	if database.ReadOnly() {
		t.Fatal("fresh database must be writable")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("database permissions=%#o want=0600", got)
	}
	backups, err := filepath.Glob(path + ".backup-*")
	if err != nil || len(backups) != 0 {
		t.Fatalf("fresh database created unnecessary backups=%v err=%v", backups, err)
	}
	if err := database.InsertEvent(context.Background(), storageEvent()); err != nil {
		t.Fatal(err)
	}

	stored, err := database.ListSessionEvents(context.Background(), "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 {
		t.Fatalf("events=%d", len(stored))
	}
	if strings.Contains(string(stored[0].Payload), "synthetic-value") {
		t.Fatalf("secret reached storage: %s", stored[0].Payload)
	}
}

func TestDuplicateEventIsIdempotent(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	event := storageEvent()
	if err := database.InsertEvent(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if err := database.InsertEvent(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	stored, err := database.ListSessionEvents(context.Background(), event.SessionID)
	if err != nil || len(stored) != 1 {
		t.Fatalf("events=%d err=%v", len(stored), err)
	}
}

func TestDashboardSummaryOnlyAggregatesLocalEventMetadata(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	exact := storageEvent()
	estimated := storageEvent()
	estimated.EventID = "event-2"
	estimated.SessionID = "session-2"
	estimated.ProjectID = "project-2"
	estimated.Precision = events.PrecisionEstimated
	if err := database.InsertEvent(context.Background(), exact); err != nil {
		t.Fatal(err)
	}
	if err := database.InsertEvent(context.Background(), estimated); err != nil {
		t.Fatal(err)
	}

	summary, err := database.DashboardSummary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if summary.Projects != 2 || summary.Sessions != 2 || summary.Events != 2 || summary.ActiveSessions != 2 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary.Precision.Exact != 1 || summary.Precision.Estimated != 1 || summary.Precision.Unavailable != 0 {
		t.Fatalf("unexpected precision: %+v", summary.Precision)
	}
}

func TestFailedMigrationCreatesBackupAndEntersReadOnlyRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doctor.db")
	database, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	fixedNow := func() time.Time {
		return time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	}
	recovered, err := openWithMigrations(path, append(defaultMigrations(), migration{
		version: 6,
		name:    "broken",
		sql:     "CREATE TABLE broken( INVALID SQL",
	}), fixedNow)
	if err != nil {
		t.Fatalf("recovery open should remain usable: %v", err)
	}
	t.Cleanup(func() { _ = recovered.Close() })

	if !recovered.ReadOnly() || recovered.RecoveryError() == nil {
		t.Fatal("failed migration must enter explicit read-only recovery")
	}
	wantBackup := path + ".backup-20260813T120000Z"
	if recovered.RecoveryBackupPath() != wantBackup {
		t.Fatalf("backup=%q want=%q", recovered.RecoveryBackupPath(), wantBackup)
	}
	if _, err := os.Stat(wantBackup); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
	if err := recovered.InsertEvent(context.Background(), storageEvent()); err == nil {
		t.Fatal("read-only recovery accepted a write")
	}
}

func TestInitialSchemaContainsEveryCoreTable(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	want := []string{
		"schema_migrations", "projects", "clients", "models", "sessions", "events",
		"git_snapshots", "validations", "usage_records", "cost_records", "quota_snapshots",
		"memories", "context_capsules", "diagnoses", "comparisons", "replays", "consents",
		"price_catalog_versions", "exchange_rate_versions",
		"model_requests", "conversation_messages", "client_connections", "analysis_snapshots", "privacy_settings",
	}
	for _, table := range want {
		var count int
		err := database.sql.QueryRowContext(context.Background(),
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&count)
		if err != nil || count != 1 {
			t.Fatalf("table %s missing: count=%d err=%v", table, count, err)
		}
	}
}

func TestPrivacySettingsPersistAcrossDatabaseReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doctor.db")
	database, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	want := conversations.PrivacySettings{CapturePrompts: true, CaptureFileContents: true, RetentionDays: 45}
	if err := database.SavePrivacySettings(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	_ = database.Close()
	database, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	got, err := database.PrivacySettings(context.Background())
	if err != nil || got != want {
		t.Fatalf("settings=%+v err=%v", got, err)
	}
}

func TestConversationRoundTripPreservesCompleteMessagesAndUsage(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	started := time.Date(2026, 8, 14, 1, 2, 3, 0, time.UTC)
	completed := started.Add(1450 * time.Millisecond)
	input, output, cached, reasoning, cost := int64(321), int64(89), int64(120), int64(14), int64(2870)
	record := conversations.Request{
		ID: "request-1", SessionID: "session-live-1", ProjectID: "project-live-1",
		Client:   events.ClientRef{Name: "codex", Version: "1.2.3"},
		Model:    events.ModelRef{DisplayName: "gpt-example"},
		Protocol: "openai", Method: "POST", Path: "/v1/responses", StatusCode: 200,
		StartedAt: started, CompletedAt: &completed, FirstByteMS: 180, DurationMS: 1450,
		Usage: conversations.Usage{InputTokens: &input, OutputTokens: &output, CachedTokens: &cached, ReasoningTokens: &reasoning, Precision: "exact", Provenance: "provider-response"},
		Cost:  conversations.Cost{AmountMicros: &cost, Currency: "USD", Precision: "exact", Provenance: "local-catalog"},
		Messages: []conversations.Message{
			{ID: "message-1", Sequence: 0, Role: "system", Content: "You are precise.", CreatedAt: started},
			{ID: "message-2", Sequence: 1, Role: "user", Content: "完整保留这段对话。", CreatedAt: started.Add(time.Millisecond)},
			{ID: "message-3", Sequence: 2, Role: "assistant", Content: "已经完整保存。", ToolName: "shell", ToolPayloadJSON: `{"command":"go test ./..."}`, CreatedAt: completed},
		},
	}
	if err := database.SaveConversationRequest(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	var analysisCount int
	if err := database.sql.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM analysis_snapshots WHERE session_id=?", record.SessionID).Scan(&analysisCount); err != nil || analysisCount != 1 {
		t.Fatalf("analysis snapshots=%d err=%v", analysisCount, err)
	}

	got, err := database.GetConversationRequest(context.Background(), record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != record.ID || got.Protocol != "openai" || got.DurationMS != 1450 || got.FirstByteMS != 180 {
		t.Fatalf("request metadata mismatch: %+v", got)
	}
	if got.Usage.InputTokens == nil || *got.Usage.InputTokens != input || got.Cost.AmountMicros == nil || *got.Cost.AmountMicros != cost {
		t.Fatalf("usage/cost mismatch: usage=%+v cost=%+v", got.Usage, got.Cost)
	}
	if len(got.Messages) != 3 || got.Messages[1].Content != "完整保留这段对话。" || got.Messages[2].ToolPayloadJSON != `{"command":"go test ./..."}` {
		t.Fatalf("messages not preserved in order: %+v", got.Messages)
	}
}

func TestConversationSchemaHasNoTransportCredentialColumns(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	for _, table := range []string{"model_requests", "conversation_messages"} {
		rows, err := database.sql.QueryContext(context.Background(), "PRAGMA table_info("+table+")")
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var cid, notNull, primaryKey int
			var name, columnType string
			var defaultValue any
			if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
				t.Fatal(err)
			}
			lower := strings.ToLower(name)
			if strings.Contains(lower, "authorization") || strings.Contains(lower, "cookie") || strings.Contains(lower, "api_key") || strings.Contains(lower, "header") {
				t.Fatalf("transport credential column %q must not exist in %s", name, table)
			}
		}
		_ = rows.Close()
	}
}

func TestClientConnectionUpsertAndList(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "doctor.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	now := time.Date(2026, 8, 14, 2, 0, 0, 0, time.UTC)
	connection := conversations.ClientConnection{Key: "codex", DisplayName: "Codex", Detected: true, State: "connected", Capability: "proxy", Detail: "loopback proxy", LastHeartbeatAt: &now, UpdatedAt: now}
	if err := database.UpsertClientConnection(context.Background(), connection); err != nil {
		t.Fatal(err)
	}
	connection.State = "active"
	connection.Detail = "capturing"
	if err := database.UpsertClientConnection(context.Background(), connection); err != nil {
		t.Fatal(err)
	}
	got, err := database.ListClientConnections(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].State != "active" || got[0].Detail != "capturing" || !got[0].Detected {
		t.Fatalf("unexpected connections: %+v", got)
	}
}

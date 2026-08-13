package storage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	if got := database.SchemaVersion(); got != 2 {
		t.Fatalf("schema=%d", got)
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
		version: 3,
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

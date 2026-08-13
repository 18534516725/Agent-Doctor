package storage

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/18534516725/Agent-Doctor/migrations"
)

type migration struct {
	version int
	name    string
	sql     string
}

func defaultMigrations() []migration {
	contents, err := migrations.Files.ReadFile("001_initial.sql")
	if err != nil {
		panic(fmt.Sprintf("embedded migration missing: %v", err))
	}
	return []migration{{version: 1, name: "initial", sql: string(contents)}}
}

func migrate(connection *sql.DB, path string, migrationSet []migration, now func() time.Time, shouldBackup bool) (int, string, error) {
	current := currentSchemaVersion(connection)
	sort.Slice(migrationSet, func(left, right int) bool { return migrationSet[left].version < migrationSet[right].version })

	pending := make([]migration, 0, len(migrationSet))
	for _, item := range migrationSet {
		if item.version > current {
			pending = append(pending, item)
		}
	}
	if len(pending) == 0 {
		return current, "", nil
	}

	backupPath := ""
	if shouldBackup {
		if _, err := connection.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
			return current, "", fmt.Errorf("checkpoint database before migration backup: %w", err)
		}
		var err error
		backupPath, err = backupIfPresent(path, now())
		if err != nil {
			return current, "", err
		}
	}
	for _, item := range pending {
		transaction, err := connection.Begin()
		if err != nil {
			return current, backupPath, fmt.Errorf("begin migration %d: %w", item.version, err)
		}
		if _, err := transaction.Exec(item.sql); err != nil {
			_ = transaction.Rollback()
			return current, backupPath, fmt.Errorf("apply migration %d (%s): %w", item.version, item.name, err)
		}
		if _, err := transaction.Exec(
			"INSERT INTO schema_migrations(version, name, applied_at) VALUES(?, ?, ?)",
			item.version, item.name, now().UTC().Format(time.RFC3339Nano),
		); err != nil {
			_ = transaction.Rollback()
			return current, backupPath, fmt.Errorf("record migration %d: %w", item.version, err)
		}
		if err := transaction.Commit(); err != nil {
			return current, backupPath, fmt.Errorf("commit migration %d: %w", item.version, err)
		}
		current = item.version
	}
	return current, backupPath, nil
}

func currentSchemaVersion(connection *sql.DB) int {
	var exists int
	if err := connection.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&exists); err != nil || exists == 0 {
		return 0
	}
	var version sql.NullInt64
	if err := connection.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil || !version.Valid {
		return 0
	}
	return int(version.Int64)
}

func backupIfPresent(path string, timestamp time.Time) (string, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) || (err == nil && info.Size() == 0) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("inspect database before migration: %w", err)
	}

	backupPath := path + ".backup-" + timestamp.UTC().Format("20060102T150405Z")
	source, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open migration backup source: %w", err)
	}
	defer source.Close()
	destination, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("create migration backup: %w", err)
	}
	if _, err := io.Copy(destination, source); err != nil {
		_ = destination.Close()
		return "", fmt.Errorf("copy migration backup: %w", err)
	}
	if err := destination.Close(); err != nil {
		return "", fmt.Errorf("close migration backup: %w", err)
	}
	return backupPath, nil
}

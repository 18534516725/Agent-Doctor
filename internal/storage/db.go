package storage

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	sql                *sql.DB
	schemaVersion      int
	readOnly           bool
	recoveryErr        error
	recoveryBackupPath string
}

func Open(path string) (*DB, error) {
	return openWithMigrations(path, defaultMigrations(), time.Now)
}

func openWithMigrations(path string, migrationSet []migration, now func() time.Time) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	shouldBackup, err := existingNonemptyFile(path)
	if err != nil {
		return nil, err
	}
	connection, err := openSQLite(path, false)
	if err != nil {
		return nil, err
	}

	database := &DB{sql: connection}
	version, backupPath, migrationErr := migrate(connection, path, migrationSet, now, shouldBackup)
	if migrationErr == nil {
		database.schemaVersion = version
		return database, nil
	}

	_ = connection.Close()
	recoveryConnection, recoveryOpenErr := openSQLite(path, true)
	if recoveryOpenErr != nil {
		return nil, fmt.Errorf("migration failed (%v) and recovery open failed: %w", migrationErr, recoveryOpenErr)
	}
	return &DB{
		sql:                recoveryConnection,
		schemaVersion:      currentSchemaVersion(recoveryConnection),
		readOnly:           true,
		recoveryErr:        migrationErr,
		recoveryBackupPath: backupPath,
	}, nil
}

func openSQLite(path string, readOnly bool) (*sql.DB, error) {
	dsn := sqliteDSN(path, readOnly)
	connection, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open SQLite: %w", err)
	}
	connection.SetMaxOpenConns(1)
	if err := connection.Ping(); err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("ping SQLite: %w", err)
	}
	if _, err := connection.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if !readOnly {
		if _, err := connection.Exec("PRAGMA journal_mode = WAL; PRAGMA busy_timeout = 5000"); err != nil {
			_ = connection.Close()
			return nil, fmt.Errorf("configure SQLite: %w", err)
		}
		if err := os.Chmod(path, 0o600); err != nil {
			_ = connection.Close()
			return nil, fmt.Errorf("secure SQLite file permissions: %w", err)
		}
	}
	return connection, nil
}

func sqliteDSN(path string, readOnly bool) string {
	normalized := strings.ReplaceAll(filepath.ToSlash(path), `\`, "/")
	if len(normalized) >= 2 && normalized[1] == ':' && ((normalized[0] >= 'A' && normalized[0] <= 'Z') || (normalized[0] >= 'a' && normalized[0] <= 'z')) {
		normalized = "/" + normalized
	}
	location := &url.URL{Scheme: "file", Path: normalized}
	if readOnly {
		query := location.Query()
		query.Set("mode", "ro")
		location.RawQuery = query.Encode()
	}
	return location.String()
}

func (database *DB) Close() error               { return database.sql.Close() }
func (database *DB) SchemaVersion() int         { return database.schemaVersion }
func (database *DB) ReadOnly() bool             { return database.readOnly }
func (database *DB) RecoveryError() error       { return database.recoveryErr }
func (database *DB) RecoveryBackupPath() string { return database.recoveryBackupPath }

func existingNonemptyFile(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect database: %w", err)
	}
	return info.Size() > 0, nil
}

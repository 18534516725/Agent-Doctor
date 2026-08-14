package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/conversations"
)

func (database *DB) PrivacySettings(ctx context.Context) (conversations.PrivacySettings, error) {
	var settings conversations.PrivacySettings
	var capturePrompts, captureFileContents int
	err := database.sql.QueryRowContext(ctx, `SELECT capture_prompts, capture_file_contents, retention_days FROM privacy_settings WHERE singleton=1`).Scan(
		&capturePrompts, &captureFileContents, &settings.RetentionDays)
	if err != nil {
		return conversations.PrivacySettings{}, fmt.Errorf("query privacy settings: %w", err)
	}
	settings.CapturePrompts = capturePrompts == 1
	settings.CaptureFileContents = captureFileContents == 1
	return settings, nil
}

func (database *DB) SavePrivacySettings(ctx context.Context, settings conversations.PrivacySettings) error {
	if database.readOnly {
		return ErrReadOnlyRecovery
	}
	if settings.RetentionDays < 1 || settings.RetentionDays > 3650 {
		return fmt.Errorf("retention days must be between 1 and 3650")
	}
	transaction, err := database.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin privacy settings transaction: %w", err)
	}
	defer transaction.Rollback()
	_, err = transaction.ExecContext(ctx, `UPDATE privacy_settings SET capture_prompts=?, capture_file_contents=?, retention_days=?, updated_at=? WHERE singleton=1`,
		boolInt(settings.CapturePrompts), boolInt(settings.CaptureFileContents), settings.RetentionDays, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save privacy settings: %w", err)
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -settings.RetentionDays).Format(time.RFC3339Nano)
	if _, err := transaction.ExecContext(ctx, `DELETE FROM sessions WHERE started_at < ?`, cutoff); err != nil {
		return fmt.Errorf("apply conversation retention: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit privacy settings: %w", err)
	}
	return nil
}

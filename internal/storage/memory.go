package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	projectmemory "github.com/18534516725/Agent-Doctor/internal/memory"
)

func (database *DB) CreateMemory(ctx context.Context, projectID string, input projectmemory.CreateInput, now time.Time) (projectmemory.Item, error) {
	content := strings.TrimSpace(input.Content)
	if projectID == "" || content == "" || len([]byte(content)) > 16*1024 {
		return projectmemory.Item{}, errors.New("project and memory content up to 16 KiB are required")
	}
	if input.SourceKind == "" {
		input.SourceKind = "manual"
	}
	timestamp := now.UTC().Format(time.RFC3339Nano)
	digest := sha256.Sum256([]byte(projectID + "\x00" + content + "\x00" + timestamp))
	id := "memory-" + hex.EncodeToString(digest[:12])
	tx, err := database.sql.BeginTx(ctx, nil)
	if err != nil {
		return projectmemory.Item{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO projects(id,first_seen_at,last_seen_at) VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET last_seen_at=excluded.last_seen_at`, projectID, timestamp, timestamp); err != nil {
		return projectmemory.Item{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO memories(id,project_id,kind,content,confidence,source_session_id,created_at,updated_at,source,observation_count,state,source_id) VALUES(?,?,?, ?,1,NULL,?,?, 'user-explicit',1,'candidate',?)`, id, projectID, input.SourceKind, content, timestamp, timestamp, input.SourceID)
	if err != nil {
		return projectmemory.Item{}, fmt.Errorf("create memory: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return projectmemory.Item{}, err
	}
	return database.memoryByID(ctx, projectID, id)
}

func (database *DB) ListMemories(ctx context.Context, projectID, state string) ([]projectmemory.Item, error) {
	query := `SELECT id,project_id,content,state,kind,source_id,created_at,updated_at FROM memories WHERE project_id=? AND state!='deleted'`
	args := []any{projectID}
	if state != "" {
		query += " AND state=?"
		args = append(args, state)
	}
	query += " ORDER BY updated_at DESC,id"
	rows, err := database.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []projectmemory.Item{}
	for rows.Next() {
		var item projectmemory.Item
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Content, &item.State, &item.SourceKind, &item.SourceID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (database *DB) UpdateMemory(ctx context.Context, projectID, memoryID string, input projectmemory.UpdateInput, now time.Time) (projectmemory.Item, error) {
	current, err := database.memoryByID(ctx, projectID, memoryID)
	if err != nil {
		return current, err
	}
	if strings.TrimSpace(input.Content) != "" {
		if len([]byte(input.Content)) > 16*1024 {
			return current, errors.New("memory content exceeds 16 KiB")
		}
		current.Content = strings.TrimSpace(input.Content)
	}
	if input.State != "" {
		switch input.State {
		case "candidate", "active", "disabled":
			current.State = input.State
		default:
			return current, errors.New("invalid memory state")
		}
	}
	_, err = database.sql.ExecContext(ctx, `UPDATE memories SET content=?,state=?,updated_at=? WHERE id=? AND project_id=? AND state!='deleted'`, current.Content, current.State, now.UTC().Format(time.RFC3339Nano), memoryID, projectID)
	if err != nil {
		return current, err
	}
	return database.memoryByID(ctx, projectID, memoryID)
}
func (database *DB) DeleteMemory(ctx context.Context, projectID, memoryID string, now time.Time) error {
	result, err := database.sql.ExecContext(ctx, `UPDATE memories SET state='deleted',content='',deleted_at=?,updated_at=? WHERE id=? AND project_id=? AND state!='deleted'`, now.UTC().Format(time.RFC3339Nano), now.UTC().Format(time.RFC3339Nano), memoryID, projectID)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return errors.New("memory not found")
	}
	return nil
}
func (database *DB) memoryByID(ctx context.Context, projectID, memoryID string) (projectmemory.Item, error) {
	var item projectmemory.Item
	err := database.sql.QueryRowContext(ctx, `SELECT id,project_id,content,state,kind,source_id,created_at,updated_at FROM memories WHERE id=? AND project_id=? AND state!='deleted'`, memoryID, projectID).Scan(&item.ID, &item.ProjectID, &item.Content, &item.State, &item.SourceKind, &item.SourceID, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, fmt.Errorf("memory not found: %w", err)
	}
	return item, nil
}

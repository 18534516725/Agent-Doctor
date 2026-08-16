package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/handoff"
)

var codexHostContextPrefix = regexp.MustCompile(`(?s)^\s*(?:<recommended_plugins>.*?</recommended_plugins>|# AGENTS\.md instructions\s*<INSTRUCTIONS>.*?</INSTRUCTIONS>|<environment_context>.*?</environment_context>)\s*`)

func (database *DB) ProjectHandoff(ctx context.Context, projectIDs []string, budget int, now time.Time) (handoff.Capsule, error) {
	projectIDs = uniqueNonempty(projectIDs)
	if len(projectIDs) == 0 {
		return handoff.Capsule{}, errors.New("at least one project identity is required")
	}
	projectID, _, err := database.resolveHandoffProject(ctx, projectIDs)
	if err != nil {
		return handoff.Capsule{}, err
	}
	snapshot := handoff.Snapshot{
		ProjectID: projectID, GeneratedAt: now.UTC(), Memories: []handoff.Memory{},
		Limitations: []string{"Only confirmed project memory and the latest captured task snapshot are included; verify against the current workspace."},
	}
	hasTask, err := database.populateHandoffTask(ctx, &snapshot)
	if err != nil {
		return handoff.Capsule{}, err
	}
	if !hasTask {
		snapshot.Limitations = append(snapshot.Limitations, "No captured task conversation is available for this project yet.")
	}

	items, err := database.ListMemories(ctx, projectID, "active")
	if err != nil {
		return handoff.Capsule{}, err
	}
	for _, item := range items {
		snapshot.Memories = append(snapshot.Memories, handoff.Memory{Content: item.Content, SourceKind: item.SourceKind, SourceID: item.SourceID})
	}
	if !hasTask && len(snapshot.Memories) == 0 {
		return handoff.Capsule{}, errors.New("no captured task or confirmed project memory is available")
	}
	capsule := handoff.Render(snapshot, budget)
	delivery, err := database.latestHandoffDelivery(ctx, projectID)
	if err != nil {
		return handoff.Capsule{}, err
	}
	capsule.LastDelivery = delivery
	return capsule, nil
}

func (database *DB) populateHandoffTask(ctx context.Context, snapshot *handoff.Snapshot) (bool, error) {
	rows, err := database.sql.QueryContext(ctx, `SELECT id FROM model_requests WHERE project_id=? ORDER BY started_at DESC,id DESC LIMIT 50`, snapshot.ProjectID)
	if err != nil {
		return false, err
	}
	requestIDs := make([]string, 0, 50)
	for rows.Next() {
		var requestID string
		if err := rows.Scan(&requestID); err != nil {
			_ = rows.Close()
			return false, err
		}
		requestIDs = append(requestIDs, requestID)
	}
	if err := rows.Close(); err != nil {
		return false, err
	}
	if err := rows.Err(); err != nil {
		return false, err
	}

	for _, requestID := range requestIDs {
		record, err := database.GetConversationRequest(ctx, requestID)
		if err != nil {
			return false, err
		}
		var goal, latestResult string
		for _, message := range record.Messages {
			switch message.Role {
			case "user":
				if text := boundedHandoffText(message.Content, 1600); text != "" {
					goal = text
				}
			case "assistant":
				if text := boundedHandoffText(message.Content, 2000); text != "" {
					latestResult = text
				}
			}
		}
		foundInRecord := false
		if snapshot.Goal == "" && goal != "" {
			snapshot.Goal = goal
			foundInRecord = true
		}
		if snapshot.LatestResult == "" && latestResult != "" {
			snapshot.LatestResult = latestResult
			foundInRecord = true
		}
		if foundInRecord && snapshot.SourceClient == "" {
			snapshot.SourceClient = record.Client.Name
			snapshot.SourceSessionID = record.SessionID
		}
		if snapshot.Goal != "" && snapshot.LatestResult != "" {
			break
		}
	}
	return snapshot.Goal != "" || snapshot.LatestResult != "", nil
}

func (database *DB) RecordHandoffDelivery(ctx context.Context, capsule handoff.Capsule, targetClient string, deliveredAt time.Time) error {
	if database.readOnly {
		return ErrReadOnlyRecovery
	}
	if capsule.ProjectID == "" || strings.TrimSpace(targetClient) == "" || deliveredAt.IsZero() {
		return errors.New("project, target client, and delivery time are required")
	}
	timestamp := deliveredAt.UTC().Format(time.RFC3339Nano)
	digest := sha256.Sum256([]byte(strings.Join([]string{capsule.ProjectID, capsule.SourceSessionID, targetClient, timestamp}, "\x00")))
	id := "handoff-" + hex.EncodeToString(digest[:12])
	var sourceSession any
	if capsule.SourceSessionID != "" {
		var exists int
		if err := database.sql.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE id=?", capsule.SourceSessionID).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			sourceSession = capsule.SourceSessionID
		}
	}
	tx, err := database.sql.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO context_capsules(id,project_id,session_id,content,token_estimate,created_at) VALUES(?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET content=excluded.content,token_estimate=excluded.token_estimate,created_at=excluded.created_at`, id, capsule.ProjectID, sourceSession, capsule.Rendered, capsule.TokenEstimate, timestamp); err != nil {
		return fmt.Errorf("persist handoff capsule: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO handoff_delivery_receipts(id,project_id,source_session_id,source_client,target_client,memory_count,delivered_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET memory_count=excluded.memory_count,delivered_at=excluded.delivered_at`, id, capsule.ProjectID, sourceSession, capsule.SourceClient, strings.TrimSpace(targetClient), len(capsule.Memories), timestamp); err != nil {
		return fmt.Errorf("record handoff delivery: %w", err)
	}
	return tx.Commit()
}

func (database *DB) resolveHandoffProject(ctx context.Context, projectIDs []string) (string, string, error) {
	placeholders := strings.TrimRight(strings.Repeat("?,", len(projectIDs)), ",")
	arguments := make([]any, len(projectIDs))
	for index, value := range projectIDs {
		arguments[index] = value
	}
	var projectID, requestID string
	err := database.sql.QueryRowContext(ctx, `SELECT project_id,id FROM model_requests WHERE project_id IN (`+placeholders+`) ORDER BY started_at DESC,id DESC LIMIT 1`, arguments...).Scan(&projectID, &requestID)
	if err == nil {
		return projectID, requestID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", "", err
	}
	for _, candidate := range projectIDs {
		var active int
		if err := database.sql.QueryRowContext(ctx, "SELECT COUNT(*) FROM memories WHERE project_id=? AND state='active'", candidate).Scan(&active); err != nil {
			return "", "", err
		}
		if active > 0 {
			return candidate, "", nil
		}
	}
	for _, candidate := range projectIDs {
		var exists int
		if err := database.sql.QueryRowContext(ctx, "SELECT COUNT(*) FROM projects WHERE id=?", candidate).Scan(&exists); err != nil {
			return "", "", err
		}
		if exists > 0 {
			return candidate, "", nil
		}
	}
	return "", "", fmt.Errorf("project handoff not found")
}

func (database *DB) latestHandoffDelivery(ctx context.Context, projectID string) (*handoff.Delivery, error) {
	var delivery handoff.Delivery
	var deliveredAt string
	err := database.sql.QueryRowContext(ctx, `SELECT project_id,COALESCE(source_session_id,''),source_client,target_client,memory_count,delivered_at FROM handoff_delivery_receipts WHERE project_id=? ORDER BY delivered_at DESC,id DESC LIMIT 1`, projectID).Scan(
		&delivery.ProjectID, &delivery.SourceSessionID, &delivery.SourceClient, &delivery.TargetClient, &delivery.MemoryCount, &deliveredAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	delivery.DeliveredAt, err = time.Parse(time.RFC3339Nano, deliveredAt)
	if err != nil {
		return nil, err
	}
	return &delivery, nil
}

func uniqueNonempty(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func boundedHandoffText(value string, limit int) string {
	value = strings.TrimSpace(value)
	for {
		cleaned := codexHostContextPrefix.ReplaceAllString(value, "")
		if cleaned == value {
			break
		}
		value = strings.TrimSpace(cleaned)
	}
	if len(value) > limit {
		value = value[:limit] + "…"
	}
	return value
}

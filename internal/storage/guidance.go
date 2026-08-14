package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/guidance"
)

func (database *DB) RuntimeGuidance(ctx context.Context, sessionID string, now time.Time) (guidance.Decision, error) {
	if now.IsZero() {
		return guidance.Decision{}, fmt.Errorf("guidance evaluation time is required")
	}
	if sessionID == "" {
		var err error
		sessionID, err = database.latestSessionID(ctx)
		if err != nil {
			return guidance.Decision{}, err
		}
	}
	eventList, err := database.ListSessionEvents(ctx, sessionID)
	if err != nil {
		return guidance.Decision{}, err
	}
	if len(eventList) == 0 {
		return guidance.Decision{}, fmt.Errorf("session %q has no evidence", sessionID)
	}
	decision := guidance.Evaluate(guidance.Project(eventList), now.UTC())
	if decision.Kind == guidance.KindContinue {
		return decision, nil
	}
	if database.readOnly {
		return guidance.Decision{}, ErrReadOnlyRecovery
	}
	payload, err := json.Marshal(decision)
	if err != nil {
		return guidance.Decision{}, fmt.Errorf("encode runtime guidance: %w", err)
	}
	_, err = database.sql.ExecContext(ctx, `
		INSERT INTO guidance_decisions(
			id, session_id, project_id, kind, severity, payload_json,
			evidence_fingerprint, expires_at, created_at
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id, evidence_fingerprint) DO NOTHING`,
		decision.DecisionID, decision.SessionID, decision.ProjectID, decision.Kind, decision.Severity,
		string(payload), decision.EvidenceFingerprint, formatGuidanceTime(decision.ExpiresAt), formatGuidanceTime(decision.CreatedAt),
	)
	if err != nil {
		return guidance.Decision{}, fmt.Errorf("persist runtime guidance: %w", err)
	}
	return database.guidanceByFingerprint(ctx, decision.SessionID, decision.EvidenceFingerprint)
}

func (database *DB) LatestRuntimeGuidance(ctx context.Context, projectID string, now time.Time) (guidance.Decision, error) {
	row := database.sql.QueryRowContext(ctx, `
		SELECT payload_json FROM guidance_decisions
		WHERE project_id=? AND expires_at>?
		ORDER BY created_at DESC LIMIT 1`, projectID, formatGuidanceTime(now))
	decision, err := scanGuidance(row)
	if errors.Is(err, sql.ErrNoRows) {
		return guidance.Evaluate(guidance.SessionState{ProjectID: projectID}, now.UTC()), nil
	}
	return decision, err
}

func (database *DB) ListActiveGuidance(ctx context.Context, now time.Time, limit int) ([]guidance.Decision, error) {
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := database.sql.QueryContext(ctx, `
		SELECT payload_json FROM guidance_decisions
		WHERE expires_at>?
		ORDER BY created_at DESC LIMIT ?`, formatGuidanceTime(now), limit)
	if err != nil {
		return nil, fmt.Errorf("query active runtime guidance: %w", err)
	}
	defer rows.Close()
	decisions := make([]guidance.Decision, 0)
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, fmt.Errorf("scan active runtime guidance: %w", err)
		}
		var decision guidance.Decision
		if err := json.Unmarshal([]byte(payload), &decision); err != nil {
			return nil, fmt.Errorf("decode active runtime guidance: %w", err)
		}
		decisions = append(decisions, decision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active runtime guidance: %w", err)
	}
	return decisions, nil
}

func (database *DB) GuidanceStatus(ctx context.Context, now time.Time) (guidance.Status, error) {
	var occurredAt, client string
	err := database.sql.QueryRowContext(ctx, `
		SELECT e.occurred_at, c.name
		FROM events e JOIN clients c ON c.id=e.client_id
		ORDER BY e.occurred_at DESC, e.id DESC LIMIT 1`).Scan(&occurredAt, &client)
	if errors.Is(err, sql.ErrNoRows) {
		return guidance.ResolveStatus(nil, "", false, now.UTC(), nil), nil
	}
	if err != nil {
		return guidance.Status{}, fmt.Errorf("query latest guidance evidence: %w", err)
	}
	last, err := time.Parse(time.RFC3339Nano, occurredAt)
	if err != nil {
		return guidance.Status{}, fmt.Errorf("parse latest guidance evidence: %w", err)
	}
	var active int
	if err := database.sql.QueryRowContext(ctx, "SELECT COUNT(*) FROM guidance_decisions WHERE expires_at>?", formatGuidanceTime(now)).Scan(&active); err != nil {
		return guidance.Status{}, fmt.Errorf("query active guidance status: %w", err)
	}
	return guidance.ResolveStatus(&last, client, active > 0, now.UTC(), nil), nil
}

func (database *DB) GuidanceControlLevel(ctx context.Context, projectID string) (guidance.ControlLevel, error) {
	var level guidance.ControlLevel
	err := database.sql.QueryRowContext(ctx,
		"SELECT control_level FROM project_guidance_settings WHERE project_id=?", projectID,
	).Scan(&level)
	if errors.Is(err, sql.ErrNoRows) {
		return guidance.ControlGuide, nil
	}
	if err != nil {
		return "", fmt.Errorf("query guidance control level: %w", err)
	}
	return level, nil
}

func (database *DB) SaveGuidanceControlLevel(ctx context.Context, projectID string, level guidance.ControlLevel, now time.Time) error {
	if database.readOnly {
		return ErrReadOnlyRecovery
	}
	if !validControlLevel(level) {
		return fmt.Errorf("unsupported guidance control level %q", level)
	}
	if projectID == "" || now.IsZero() {
		return fmt.Errorf("project and update time are required")
	}
	_, err := database.sql.ExecContext(ctx, `
		INSERT INTO project_guidance_settings(project_id, control_level, updated_at)
		VALUES(?, ?, ?)
		ON CONFLICT(project_id) DO UPDATE SET
			control_level=excluded.control_level,
			updated_at=excluded.updated_at`, projectID, level, formatGuidanceTime(now))
	if err != nil {
		return fmt.Errorf("save guidance control level: %w", err)
	}
	return nil
}

func (database *DB) latestSessionID(ctx context.Context) (string, error) {
	var sessionID string
	err := database.sql.QueryRowContext(ctx, `
		SELECT s.id FROM sessions s
		LEFT JOIN events e ON e.session_id=s.id
		GROUP BY s.id
		ORDER BY COALESCE(MAX(e.occurred_at), s.started_at) DESC, s.id DESC
		LIMIT 1`).Scan(&sessionID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("no session evidence is available")
	}
	if err != nil {
		return "", fmt.Errorf("resolve latest session: %w", err)
	}
	return sessionID, nil
}

func (database *DB) guidanceByFingerprint(ctx context.Context, sessionID, fingerprint string) (guidance.Decision, error) {
	return scanGuidance(database.sql.QueryRowContext(ctx, `
		SELECT payload_json FROM guidance_decisions
		WHERE session_id=? AND evidence_fingerprint=?`, sessionID, fingerprint))
}

type guidanceScanner interface {
	Scan(dest ...any) error
}

func scanGuidance(scanner guidanceScanner) (guidance.Decision, error) {
	var payload string
	if err := scanner.Scan(&payload); err != nil {
		return guidance.Decision{}, err
	}
	var decision guidance.Decision
	if err := json.Unmarshal([]byte(payload), &decision); err != nil {
		return guidance.Decision{}, fmt.Errorf("decode runtime guidance: %w", err)
	}
	return decision, nil
}

func validControlLevel(level guidance.ControlLevel) bool {
	switch level {
	case guidance.ControlObserve, guidance.ControlGuide, guidance.ControlGuard, guidance.ControlAutopilot:
		return true
	default:
		return false
	}
}

func formatGuidanceTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

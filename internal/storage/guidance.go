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
	sessionID, err := database.latestSessionIDForProject(ctx, projectID)
	if errors.Is(err, sql.ErrNoRows) {
		return guidance.Evaluate(guidance.SessionState{ProjectID: projectID}, now.UTC()), nil
	}
	if err != nil {
		return guidance.Decision{}, err
	}
	eventList, err := database.ListSessionEvents(ctx, sessionID)
	if err != nil {
		return guidance.Decision{}, err
	}
	if len(eventList) == 0 {
		return guidance.Evaluate(guidance.SessionState{SessionID: sessionID, ProjectID: projectID}, now.UTC()), nil
	}
	return database.RuntimeGuidance(ctx, sessionID, now)
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
		SELECT activity_at, client_name FROM (
			SELECT e.occurred_at AS activity_at, c.name AS client_name, e.id AS activity_id
			FROM events e JOIN clients c ON c.id=e.client_id
			UNION ALL
			SELECT m.created_at AS activity_at, c.name AS client_name, m.id AS activity_id
			FROM conversation_messages m
			JOIN model_requests r ON r.id=m.request_id
			JOIN clients c ON c.id=r.client_id
		)
		ORDER BY activity_at DESC, activity_id DESC LIMIT 1`).Scan(&occurredAt, &client)
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

func (database *DB) RecordGuidanceDelivery(ctx context.Context, receipt guidance.DeliveryReceipt) error {
	if database.readOnly {
		return ErrReadOnlyRecovery
	}
	if receipt.SessionID == "" || receipt.ProjectID == "" || receipt.Client == "" || receipt.DecisionID == "" || receipt.DeliveredAt.IsZero() {
		return fmt.Errorf("complete guidance delivery receipt is required")
	}
	if !validControlLevel(receipt.ControlLevel) {
		return fmt.Errorf("unsupported guidance delivery control level %q", receipt.ControlLevel)
	}
	switch receipt.DecisionKind {
	case guidance.KindContinue, guidance.KindAdvise, guidance.KindRedirect, guidance.KindAsk, guidance.KindBlock, guidance.KindVerify:
	default:
		return fmt.Errorf("unsupported guidance delivery decision %q", receipt.DecisionKind)
	}
	deliveredAt := formatGuidanceTime(receipt.DeliveredAt)
	_, err := database.sql.ExecContext(ctx, `
		INSERT INTO guidance_delivery_receipts(
			session_id, project_id, client, decision_id, decision_kind, control_level,
			delivery_count, first_delivered_at, delivered_at
		) VALUES(?, ?, ?, ?, ?, ?, 1, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET
			project_id=excluded.project_id,
			client=excluded.client,
			decision_id=excluded.decision_id,
			decision_kind=excluded.decision_kind,
			control_level=excluded.control_level,
			delivery_count=guidance_delivery_receipts.delivery_count+1,
			delivered_at=excluded.delivered_at`,
		receipt.SessionID, receipt.ProjectID, receipt.Client, receipt.DecisionID,
		receipt.DecisionKind, receipt.ControlLevel, deliveredAt, deliveredAt,
	)
	if err != nil {
		return fmt.Errorf("record guidance delivery: %w", err)
	}
	return nil
}

func (database *DB) LatestGuidanceDelivery(ctx context.Context) (guidance.DeliveryReceipt, error) {
	var receipt guidance.DeliveryReceipt
	var deliveredAt string
	err := database.sql.QueryRowContext(ctx, `
		SELECT session_id, project_id, client, decision_id, decision_kind,
			control_level, delivery_count, delivered_at
		FROM guidance_delivery_receipts
		ORDER BY delivered_at DESC, session_id DESC LIMIT 1`).Scan(
		&receipt.SessionID, &receipt.ProjectID, &receipt.Client, &receipt.DecisionID,
		&receipt.DecisionKind, &receipt.ControlLevel, &receipt.DeliveryCount, &deliveredAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return guidance.DeliveryReceipt{}, nil
	}
	if err != nil {
		return guidance.DeliveryReceipt{}, fmt.Errorf("query latest guidance delivery: %w", err)
	}
	receipt.DeliveredAt, err = time.Parse(time.RFC3339Nano, deliveredAt)
	if err != nil {
		return guidance.DeliveryReceipt{}, fmt.Errorf("parse guidance delivery time: %w", err)
	}
	return receipt, nil
}

func (database *DB) latestSessionID(ctx context.Context) (string, error) {
	return database.latestSessionIDForProject(ctx, "")
}

func (database *DB) latestSessionIDForProject(ctx context.Context, projectID string) (string, error) {
	var sessionID string
	err := database.sql.QueryRowContext(ctx, `
		SELECT s.id FROM sessions s
		WHERE (?='' OR s.project_id=?)
		ORDER BY MAX(
			s.started_at,
			COALESCE((SELECT MAX(e.occurred_at) FROM events e WHERE e.session_id=s.id), ''),
			COALESCE((SELECT MAX(r.started_at) FROM model_requests r WHERE r.session_id=s.id), ''),
			COALESCE((SELECT MAX(r.completed_at) FROM model_requests r WHERE r.session_id=s.id), ''),
			COALESCE((SELECT MAX(m.created_at) FROM conversation_messages m WHERE m.session_id=s.id), '')
		) DESC, s.id DESC
		LIMIT 1`, projectID, projectID).Scan(&sessionID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", sql.ErrNoRows
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

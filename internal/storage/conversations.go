package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/conversations"
)

func (database *DB) SaveConversationRequest(ctx context.Context, record conversations.Request) error {
	if database.readOnly {
		return ErrReadOnlyRecovery
	}
	if err := validateConversationRequest(record); err != nil {
		return err
	}

	transaction, err := database.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin conversation transaction: %w", err)
	}
	defer transaction.Rollback()

	startedAt := record.StartedAt.UTC().Format(time.RFC3339Nano)
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO projects(id, first_seen_at, last_seen_at) VALUES(?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET last_seen_at=excluded.last_seen_at`,
		record.ProjectID, startedAt, startedAt); err != nil {
		return fmt.Errorf("upsert conversation project: %w", err)
	}
	clientID, err := lookupOrCreateClient(ctx, transaction, record.Client)
	if err != nil {
		return err
	}
	modelID, err := lookupOrCreateModel(ctx, transaction, record.Model)
	if err != nil {
		return err
	}
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO sessions(id, project_id, client_id, model_id, started_at)
		VALUES(?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING`,
		record.SessionID, record.ProjectID, clientID, modelID, startedAt); err != nil {
		return fmt.Errorf("upsert conversation session: %w", err)
	}

	var completedAt any
	if record.CompletedAt != nil {
		completedAt = record.CompletedAt.UTC().Format(time.RFC3339Nano)
	}
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO model_requests(
			id, session_id, project_id, client_id, model_id, protocol, method, request_path,
			status_code, started_at, completed_at, first_byte_ms, duration_ms,
			input_tokens, output_tokens, cached_tokens, reasoning_tokens,
			usage_precision, usage_provenance, cost_amount_micros, cost_currency,
			cost_precision, cost_provenance)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status_code=excluded.status_code, completed_at=excluded.completed_at,
			first_byte_ms=excluded.first_byte_ms, duration_ms=excluded.duration_ms,
			input_tokens=excluded.input_tokens, output_tokens=excluded.output_tokens,
			cached_tokens=excluded.cached_tokens, reasoning_tokens=excluded.reasoning_tokens,
			usage_precision=excluded.usage_precision, usage_provenance=excluded.usage_provenance,
			cost_amount_micros=excluded.cost_amount_micros, cost_currency=excluded.cost_currency,
			cost_precision=excluded.cost_precision, cost_provenance=excluded.cost_provenance`,
		record.ID, record.SessionID, record.ProjectID, clientID, modelID, record.Protocol,
		record.Method, record.Path, record.StatusCode, startedAt, completedAt, record.FirstByteMS,
		record.DurationMS, record.Usage.InputTokens, record.Usage.OutputTokens, record.Usage.CachedTokens,
		record.Usage.ReasoningTokens, record.Usage.Precision, record.Usage.Provenance,
		record.Cost.AmountMicros, defaultString(record.Cost.Currency, "USD"), record.Cost.Precision,
		record.Cost.Provenance); err != nil {
		return fmt.Errorf("upsert model request: %w", err)
	}
	if _, err := transaction.ExecContext(ctx, "DELETE FROM conversation_messages WHERE request_id=?", record.ID); err != nil {
		return fmt.Errorf("replace conversation messages: %w", err)
	}
	for _, message := range record.Messages {
		if _, err := transaction.ExecContext(ctx, `
			INSERT INTO conversation_messages(id, request_id, session_id, sequence, role, content, tool_name, tool_payload_json, created_at)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`, message.ID, record.ID, record.SessionID,
			message.Sequence, message.Role, message.Content, message.ToolName, message.ToolPayloadJSON,
			message.CreatedAt.UTC().Format(time.RFC3339Nano)); err != nil {
			return fmt.Errorf("insert conversation message %q: %w", message.ID, err)
		}
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit conversation: %w", err)
	}
	return nil
}

func (database *DB) GetConversationRequest(ctx context.Context, requestID string) (conversations.Request, error) {
	var record conversations.Request
	var startedAt string
	var completedAt sql.NullString
	var input, output, cached, reasoning, amount sql.NullInt64
	err := database.sql.QueryRowContext(ctx, `
		SELECT r.id, r.session_id, r.project_id, c.name, c.version, m.display_name,
			r.protocol, r.method, r.request_path, r.status_code, r.started_at, r.completed_at,
			r.first_byte_ms, r.duration_ms, r.input_tokens, r.output_tokens, r.cached_tokens,
			r.reasoning_tokens, r.usage_precision, r.usage_provenance, r.cost_amount_micros,
			r.cost_currency, r.cost_precision, r.cost_provenance
		FROM model_requests r JOIN clients c ON c.id=r.client_id JOIN models m ON m.id=r.model_id
		WHERE r.id=?`, requestID).Scan(
		&record.ID, &record.SessionID, &record.ProjectID, &record.Client.Name, &record.Client.Version,
		&record.Model.DisplayName, &record.Protocol, &record.Method, &record.Path, &record.StatusCode,
		&startedAt, &completedAt, &record.FirstByteMS, &record.DurationMS, &input, &output, &cached,
		&reasoning, &record.Usage.Precision, &record.Usage.Provenance, &amount, &record.Cost.Currency,
		&record.Cost.Precision, &record.Cost.Provenance)
	if errors.Is(err, sql.ErrNoRows) {
		return conversations.Request{}, fmt.Errorf("conversation request %q not found", requestID)
	}
	if err != nil {
		return conversations.Request{}, fmt.Errorf("query conversation request: %w", err)
	}
	parsedStartedAt, err := time.Parse(time.RFC3339Nano, startedAt)
	if err != nil {
		return conversations.Request{}, fmt.Errorf("parse request start: %w", err)
	}
	record.StartedAt = parsedStartedAt
	if completedAt.Valid {
		parsed, err := time.Parse(time.RFC3339Nano, completedAt.String)
		if err != nil {
			return conversations.Request{}, fmt.Errorf("parse request completion: %w", err)
		}
		record.CompletedAt = &parsed
	}
	record.Usage.InputTokens = nullableInt64(input)
	record.Usage.OutputTokens = nullableInt64(output)
	record.Usage.CachedTokens = nullableInt64(cached)
	record.Usage.ReasoningTokens = nullableInt64(reasoning)
	record.Cost.AmountMicros = nullableInt64(amount)

	rows, err := database.sql.QueryContext(ctx, `
		SELECT id, request_id, sequence, role, content, tool_name, tool_payload_json, created_at
		FROM conversation_messages WHERE request_id=? ORDER BY sequence, id`, requestID)
	if err != nil {
		return conversations.Request{}, fmt.Errorf("query conversation messages: %w", err)
	}
	defer rows.Close()
	record.Messages = []conversations.Message{}
	for rows.Next() {
		var message conversations.Message
		var createdAt string
		if err := rows.Scan(&message.ID, &message.RequestID, &message.Sequence, &message.Role, &message.Content, &message.ToolName, &message.ToolPayloadJSON, &createdAt); err != nil {
			return conversations.Request{}, fmt.Errorf("scan conversation message: %w", err)
		}
		message.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return conversations.Request{}, fmt.Errorf("parse message timestamp: %w", err)
		}
		record.Messages = append(record.Messages, message)
	}
	if err := rows.Err(); err != nil {
		return conversations.Request{}, fmt.Errorf("iterate conversation messages: %w", err)
	}
	return record, nil
}

func (database *DB) UpsertClientConnection(ctx context.Context, connection conversations.ClientConnection) error {
	if database.readOnly {
		return ErrReadOnlyRecovery
	}
	if strings.TrimSpace(connection.Key) == "" || connection.UpdatedAt.IsZero() {
		return errors.New("client connection key and updated time are required")
	}
	var heartbeat any
	if connection.LastHeartbeatAt != nil {
		heartbeat = connection.LastHeartbeatAt.UTC().Format(time.RFC3339Nano)
	}
	_, err := database.sql.ExecContext(ctx, `
		INSERT INTO client_connections(client_key, display_name, detected, state, capability, detail, last_heartbeat_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(client_key) DO UPDATE SET display_name=excluded.display_name, detected=excluded.detected,
			state=excluded.state, capability=excluded.capability, detail=excluded.detail,
			last_heartbeat_at=excluded.last_heartbeat_at, updated_at=excluded.updated_at`,
		connection.Key, connection.DisplayName, boolInt(connection.Detected), connection.State,
		connection.Capability, connection.Detail, heartbeat, connection.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("upsert client connection: %w", err)
	}
	return nil
}

func (database *DB) ListClientConnections(ctx context.Context) ([]conversations.ClientConnection, error) {
	rows, err := database.sql.QueryContext(ctx, `SELECT client_key, display_name, detected, state, capability, detail, last_heartbeat_at, updated_at FROM client_connections ORDER BY display_name, client_key`)
	if err != nil {
		return nil, fmt.Errorf("query client connections: %w", err)
	}
	defer rows.Close()
	result := []conversations.ClientConnection{}
	for rows.Next() {
		var item conversations.ClientConnection
		var detected int
		var heartbeat sql.NullString
		var updated string
		if err := rows.Scan(&item.Key, &item.DisplayName, &detected, &item.State, &item.Capability, &item.Detail, &heartbeat, &updated); err != nil {
			return nil, fmt.Errorf("scan client connection: %w", err)
		}
		item.Detected = detected == 1
		item.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
		if err != nil {
			return nil, fmt.Errorf("parse connection update: %w", err)
		}
		if heartbeat.Valid {
			parsed, err := time.Parse(time.RFC3339Nano, heartbeat.String)
			if err != nil {
				return nil, fmt.Errorf("parse connection heartbeat: %w", err)
			}
			item.LastHeartbeatAt = &parsed
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func validateConversationRequest(record conversations.Request) error {
	if strings.TrimSpace(record.ID) == "" || strings.TrimSpace(record.SessionID) == "" || strings.TrimSpace(record.ProjectID) == "" {
		return errors.New("request, session and project IDs are required")
	}
	if record.StartedAt.IsZero() || record.Client.Name == "" || record.Model.DisplayName == "" {
		return errors.New("request start, client and model are required")
	}
	if !validPrecision(record.Usage.Precision) || !validPrecision(record.Cost.Precision) {
		return errors.New("usage and cost precision must be exact, estimated or unavailable")
	}
	for index, message := range record.Messages {
		if message.ID == "" || message.Sequence != index || message.CreatedAt.IsZero() {
			return fmt.Errorf("message %d has invalid identity, sequence or timestamp", index)
		}
		switch message.Role {
		case "system", "user", "assistant", "tool":
		default:
			return fmt.Errorf("message %d has invalid role %q", index, message.Role)
		}
	}
	return nil
}

func validPrecision(value string) bool {
	return value == "exact" || value == "estimated" || value == "unavailable"
}
func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
func nullableInt64(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	result := value.Int64
	return &result
}

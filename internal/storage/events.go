package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/events"
	"github.com/18534516725/Agent-Doctor/internal/privacy"
)

var ErrReadOnlyRecovery = errors.New("database is in read-only recovery mode")

func (database *DB) InsertEvent(ctx context.Context, event events.Event) error {
	if database.readOnly {
		return ErrReadOnlyRecovery
	}
	if err := events.Validate(event); err != nil {
		return fmt.Errorf("validate event: %w", err)
	}
	filteredPayload, err := privacy.FilterJSON(event.Payload)
	if err != nil {
		return fmt.Errorf("filter event payload: %w", err)
	}

	transaction, err := database.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin event transaction: %w", err)
	}
	defer transaction.Rollback()

	timestamp := event.Timestamp.UTC().Format(time.RFC3339Nano)
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO projects(id, first_seen_at, last_seen_at) VALUES(?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET last_seen_at=excluded.last_seen_at`,
		event.ProjectID, timestamp, timestamp,
	); err != nil {
		return fmt.Errorf("upsert project: %w", err)
	}
	clientID, err := lookupOrCreateClient(ctx, transaction, event.Client)
	if err != nil {
		return err
	}
	modelID, err := lookupOrCreateModel(ctx, transaction, event.Model)
	if err != nil {
		return err
	}
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO sessions(id, project_id, client_id, model_id, started_at)
		VALUES(?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING`,
		event.SessionID, event.ProjectID, clientID, modelID, timestamp,
	); err != nil {
		return fmt.Errorf("upsert session: %w", err)
	}
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO events(id, schema_version, session_id, project_id, client_id, model_id,
			occurred_at, event_type, payload_json, provenance, precision)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING`,
		event.EventID, event.SchemaVersion, event.SessionID, event.ProjectID, clientID, modelID,
		timestamp, event.EventType, string(filteredPayload), event.Provenance, string(event.Precision),
	); err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit event: %w", err)
	}
	return nil
}

func (database *DB) ListSessionEvents(ctx context.Context, sessionID string) ([]events.Event, error) {
	rows, err := database.sql.QueryContext(ctx, `
		SELECT e.schema_version, e.id, e.session_id, e.project_id, e.occurred_at,
			c.name, c.version, m.display_name, e.event_type, e.payload_json,
			e.provenance, e.precision
		FROM events e
		JOIN clients c ON c.id=e.client_id
		JOIN models m ON m.id=e.model_id
		WHERE e.session_id=? ORDER BY e.occurred_at, e.id`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query session events: %w", err)
	}
	defer rows.Close()

	result := make([]events.Event, 0)
	for rows.Next() {
		var event events.Event
		var timestamp, payload, precision string
		if err := rows.Scan(
			&event.SchemaVersion, &event.EventID, &event.SessionID, &event.ProjectID, &timestamp,
			&event.Client.Name, &event.Client.Version, &event.Model.DisplayName, &event.EventType,
			&payload, &event.Provenance, &precision,
		); err != nil {
			return nil, fmt.Errorf("scan session event: %w", err)
		}
		parsedTimestamp, err := time.Parse(time.RFC3339Nano, timestamp)
		if err != nil {
			return nil, fmt.Errorf("parse event timestamp: %w", err)
		}
		event.Timestamp = parsedTimestamp
		event.Payload = json.RawMessage(payload)
		event.Precision = events.Precision(precision)
		result = append(result, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session events: %w", err)
	}
	return result, nil
}

func lookupOrCreateClient(ctx context.Context, transaction *sql.Tx, client events.ClientRef) (int64, error) {
	if _, err := transaction.ExecContext(ctx,
		"INSERT INTO clients(name, version) VALUES(?, ?) ON CONFLICT(name, version) DO NOTHING",
		client.Name, client.Version,
	); err != nil {
		return 0, fmt.Errorf("upsert client: %w", err)
	}
	var id int64
	if err := transaction.QueryRowContext(ctx,
		"SELECT id FROM clients WHERE name=? AND version=?", client.Name, client.Version,
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("lookup client: %w", err)
	}
	return id, nil
}

func lookupOrCreateModel(ctx context.Context, transaction *sql.Tx, model events.ModelRef) (int64, error) {
	if _, err := transaction.ExecContext(ctx,
		"INSERT INTO models(display_name) VALUES(?) ON CONFLICT(display_name) DO NOTHING", model.DisplayName,
	); err != nil {
		return 0, fmt.Errorf("upsert model: %w", err)
	}
	var id int64
	if err := transaction.QueryRowContext(ctx,
		"SELECT id FROM models WHERE display_name=?", model.DisplayName,
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("lookup model: %w", err)
	}
	return id, nil
}

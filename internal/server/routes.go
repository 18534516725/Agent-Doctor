package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/18534516725/Agent-Doctor/internal/dashboard"
	"github.com/18534516725/Agent-Doctor/internal/events"
)

const maxEventRequestBytes = events.MaxPayloadBytes * 2

type EventStore interface {
	InsertEvent(context.Context, events.Event) error
	ListSessionEvents(context.Context, string) ([]events.Event, error)
	ReadOnly() bool
}

type privacySettings struct {
	CapturePrompts      bool `json:"capturePrompts"`
	CaptureFileContents bool `json:"captureFileContents"`
	RetentionDays       int  `json:"retentionDays"`
}

type settingsState struct {
	mu      sync.RWMutex
	privacy privacySettings
}

func (server *Server) routes() http.Handler {
	mux := http.NewServeMux()
	settings := &settingsState{privacy: privacySettings{RetentionDays: 30}}
	mux.HandleFunc("GET /", server.dashboardHome)
	mux.HandleFunc("GET /assets/", server.dashboardAsset)
	mux.HandleFunc("GET /health", server.health)
	mux.HandleFunc("POST /api/v1/events", server.ingestEvent)
	mux.HandleFunc("GET /api/v1/dashboard/summary", server.dashboardSummary)
	mux.HandleFunc("GET /api/v1/dashboard/snapshot", server.dashboardSnapshot)
	mux.HandleFunc("GET /api/v1/sessions/", server.sessionDetails)
	mux.HandleFunc("GET /api/v1/settings/privacy", settings.getPrivacy)
	mux.HandleFunc("PUT /api/v1/settings/privacy", settings.putPrivacy)
	return securityHeaders(mux)
}

func (server *Server) health(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, map[string]any{
		"status":           "ok",
		"apiVersion":       "v1",
		"version":          server.version,
		"readOnlyRecovery": server.store.ReadOnly(),
	})
}

func (server *Server) ingestEvent(response http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(response, request.Body, maxEventRequestBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var event events.Event
	if err := decoder.Decode(&event); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeError(response, http.StatusRequestEntityTooLarge, "event request is too large")
			return
		}
		writeError(response, http.StatusBadRequest, "invalid event request")
		return
	}
	if err := events.Validate(event); err != nil {
		if strings.Contains(err.Error(), "payload exceeds") {
			writeError(response, http.StatusRequestEntityTooLarge, "event payload is too large")
			return
		}
		writeError(response, http.StatusUnprocessableEntity, "event contract validation failed")
		return
	}
	if err := server.store.InsertEvent(request.Context(), event); err != nil {
		writeError(response, http.StatusServiceUnavailable, "event storage unavailable")
		return
	}
	writeJSON(response, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (server *Server) dashboardSummary(response http.ResponseWriter, _ *http.Request) {
	summary := dashboard.Summary{}
	if provider, ok := server.store.(dashboard.SummaryProvider); ok {
		var err error
		summary, err = provider.DashboardSummary(context.Background())
		if err != nil {
			writeError(response, http.StatusServiceUnavailable, "dashboard storage unavailable")
			return
		}
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"schemaVersion":    1,
		"status":           "ready",
		"readOnlyRecovery": server.store.ReadOnly(),
		"summary":          summary,
	})
}

func (server *Server) dashboardSnapshot(response http.ResponseWriter, request *http.Request) {
	snapshot := dashboard.Snapshot{Sessions: []dashboard.Session{}, Trends: []dashboard.TrendPoint{}}
	if provider, ok := server.store.(dashboard.SnapshotProvider); ok {
		var err error
		snapshot, err = provider.DashboardSnapshot(request.Context())
		if err != nil {
			writeError(response, http.StatusServiceUnavailable, "dashboard storage unavailable")
			return
		}
	}
	writeJSON(response, http.StatusOK, map[string]any{"schemaVersion": 1, "snapshot": snapshot})
}

func (server *Server) sessionDetails(response http.ResponseWriter, request *http.Request) {
	sessionID := strings.TrimPrefix(request.URL.Path, "/api/v1/sessions/")
	if sessionID == "" || strings.Contains(sessionID, "/") || len(sessionID) > 128 {
		writeError(response, http.StatusBadRequest, "invalid session id")
		return
	}
	stored, err := server.store.ListSessionEvents(request.Context(), sessionID)
	if err != nil {
		writeError(response, http.StatusServiceUnavailable, "session storage unavailable")
		return
	}
	type safeEvent struct {
		EventID    string           `json:"eventId"`
		Timestamp  string           `json:"timestamp"`
		EventType  string           `json:"eventType"`
		Provenance string           `json:"provenance"`
		Precision  events.Precision `json:"precision"`
	}
	evidence := make([]safeEvent, 0, len(stored))
	for _, event := range stored {
		evidence = append(evidence, safeEvent{
			EventID: event.EventID, Timestamp: event.Timestamp.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
			EventType: event.EventType, Provenance: event.Provenance, Precision: event.Precision,
		})
	}
	writeJSON(response, http.StatusOK, map[string]any{"sessionId": sessionID, "events": evidence})
}

func (state *settingsState) getPrivacy(response http.ResponseWriter, _ *http.Request) {
	state.mu.RLock()
	settings := state.privacy
	state.mu.RUnlock()
	writeJSON(response, http.StatusOK, settings)
}

func (state *settingsState) putPrivacy(response http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(response, request.Body, 16*1024)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var settings privacySettings
	if err := decoder.Decode(&settings); err != nil || settings.RetentionDays < 1 || settings.RetentionDays > 3650 {
		writeError(response, http.StatusBadRequest, "invalid privacy settings")
		return
	}
	state.mu.Lock()
	state.privacy = settings
	state.mu.Unlock()
	writeJSON(response, http.StatusOK, settings)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		response.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("X-Frame-Options", "DENY")
		response.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(response, request)
	})
}

func writeError(response http.ResponseWriter, status int, message string) {
	writeJSON(response, status, map[string]any{
		"error": map[string]string{"code": http.StatusText(status), "message": message},
	})
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	if err := json.NewEncoder(response).Encode(value); err != nil {
		_ = fmt.Errorf("encode local API response: %w", err)
	}
}

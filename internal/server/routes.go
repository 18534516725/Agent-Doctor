package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/conversations"
	"github.com/18534516725/Agent-Doctor/internal/dashboard"
	"github.com/18534516725/Agent-Doctor/internal/events"
	"github.com/18534516725/Agent-Doctor/internal/realtime"
)

const maxEventRequestBytes = events.MaxPayloadBytes * 2

type EventStore interface {
	InsertEvent(context.Context, events.Event) error
	ListSessionEvents(context.Context, string) ([]events.Event, error)
	ReadOnly() bool
}

type ConversationStore interface {
	ListConversationRequests(context.Context, int, string) ([]conversations.Request, error)
	GetConversationRequest(context.Context, string) (conversations.Request, error)
	ListClientConnections(context.Context) ([]conversations.ClientConnection, error)
	LiveConversationAnalysis(context.Context) (conversations.LiveAnalysis, error)
	DeleteConversationSession(context.Context, string) error
}

type privacyStore interface {
	PrivacySettings(context.Context) (conversations.PrivacySettings, error)
	SavePrivacySettings(context.Context, conversations.PrivacySettings) error
}

func (server *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", server.dashboardHome)
	mux.HandleFunc("GET /assets/", server.dashboardAsset)
	mux.HandleFunc("GET /health", server.health)
	mux.HandleFunc("POST /api/v1/events", server.ingestEvent)
	mux.HandleFunc("GET /api/v1/dashboard/summary", server.dashboardSummary)
	mux.HandleFunc("GET /api/v1/dashboard/snapshot", server.dashboardSnapshot)
	mux.HandleFunc("GET /api/v1/live", server.liveEvents)
	mux.HandleFunc("GET /api/v1/conversations", server.conversationList)
	mux.HandleFunc("GET /api/v1/conversations/{id}", server.conversationDetails)
	mux.HandleFunc("GET /api/v1/connections", server.connectionList)
	mux.HandleFunc("GET /api/v1/analysis/live", server.liveAnalysis)
	mux.HandleFunc("DELETE /api/v1/sessions/{id}", server.deleteConversationSession)
	mux.HandleFunc("GET /api/v1/sessions/", server.sessionDetails)
	mux.HandleFunc("GET /api/v1/settings/privacy", server.getPrivacy)
	mux.HandleFunc("PUT /api/v1/settings/privacy", server.putPrivacy)
	return securityHeaders(mux)
}

func (server *Server) conversationStore(response http.ResponseWriter) (ConversationStore, bool) {
	store, ok := server.store.(ConversationStore)
	if !ok {
		writeError(response, http.StatusNotImplemented, "live conversation storage unavailable")
	}
	return store, ok
}

func (server *Server) conversationList(response http.ResponseWriter, request *http.Request) {
	store, ok := server.conversationStore(response)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(request.URL.Query().Get("limit"))
	items, err := store.ListConversationRequests(request.Context(), limit, request.URL.Query().Get("before"))
	if err != nil {
		writeError(response, http.StatusServiceUnavailable, "conversation storage unavailable")
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"items": items})
}

func (server *Server) conversationDetails(response http.ResponseWriter, request *http.Request) {
	store, ok := server.conversationStore(response)
	if !ok {
		return
	}
	id := request.PathValue("id")
	if id == "" || len(id) > 128 {
		writeError(response, http.StatusBadRequest, "invalid conversation id")
		return
	}
	item, err := store.GetConversationRequest(request.Context(), id)
	if err != nil {
		writeError(response, http.StatusNotFound, "conversation not found")
		return
	}
	writeJSON(response, http.StatusOK, item)
}

func (server *Server) connectionList(response http.ResponseWriter, request *http.Request) {
	store, ok := server.conversationStore(response)
	if !ok {
		return
	}
	items, err := store.ListClientConnections(request.Context())
	if err != nil {
		writeError(response, http.StatusServiceUnavailable, "connection storage unavailable")
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"items": items})
}

func (server *Server) liveAnalysis(response http.ResponseWriter, request *http.Request) {
	store, ok := server.conversationStore(response)
	if !ok {
		return
	}
	analysis, err := store.LiveConversationAnalysis(request.Context())
	if err != nil {
		writeError(response, http.StatusServiceUnavailable, "analysis storage unavailable")
		return
	}
	writeJSON(response, http.StatusOK, analysis)
}

func (server *Server) deleteConversationSession(response http.ResponseWriter, request *http.Request) {
	store, ok := server.conversationStore(response)
	if !ok {
		return
	}
	id := request.PathValue("id")
	if id == "" || len(id) > 128 {
		writeError(response, http.StatusBadRequest, "invalid session id")
		return
	}
	if err := store.DeleteConversationSession(request.Context(), id); err != nil {
		writeError(response, http.StatusNotFound, "session not found")
		return
	}
	server.hub.Publish(realtime.Event{Kind: "conversation.deleted", SessionID: id})
	response.WriteHeader(http.StatusNoContent)
}

func (server *Server) liveEvents(response http.ResponseWriter, request *http.Request) {
	flusher, ok := response.(http.Flusher)
	if !ok {
		writeError(response, http.StatusNotImplemented, "streaming unavailable")
		return
	}
	afterID, _ := strconv.ParseUint(request.Header.Get("Last-Event-ID"), 10, 64)
	if queryID, err := strconv.ParseUint(request.URL.Query().Get("lastEventId"), 10, 64); err == nil && queryID > afterID {
		afterID = queryID
	}
	events, cancel := server.hub.Subscribe(afterID, 32)
	defer cancel()
	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("X-Accel-Buffering", "no")
	response.WriteHeader(http.StatusOK)
	flusher.Flush()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-request.Context().Done():
			return
		case event, open := <-events:
			if !open {
				return
			}
			payload, _ := json.Marshal(event)
			_, _ = fmt.Fprintf(response, "id: %d\nevent: %s\ndata: %s\n\n", event.ID, event.Kind, payload)
			flusher.Flush()
		case <-heartbeat.C:
			_, _ = io.WriteString(response, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
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

func (server *Server) getPrivacy(response http.ResponseWriter, request *http.Request) {
	store, ok := server.store.(privacyStore)
	if !ok {
		writeJSON(response, http.StatusOK, conversations.PrivacySettings{CapturePrompts: true, RetentionDays: 30})
		return
	}
	settings, err := store.PrivacySettings(request.Context())
	if err != nil {
		writeError(response, http.StatusServiceUnavailable, "privacy settings unavailable")
		return
	}
	writeJSON(response, http.StatusOK, settings)
}

func (server *Server) putPrivacy(response http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(response, request.Body, 16*1024)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var settings conversations.PrivacySettings
	if err := decoder.Decode(&settings); err != nil || settings.RetentionDays < 1 || settings.RetentionDays > 3650 {
		writeError(response, http.StatusBadRequest, "invalid privacy settings")
		return
	}
	store, ok := server.store.(privacyStore)
	if !ok {
		writeError(response, http.StatusNotImplemented, "privacy settings storage unavailable")
		return
	}
	if err := store.SavePrivacySettings(request.Context(), settings); err != nil {
		writeError(response, http.StatusServiceUnavailable, "privacy settings unavailable")
		return
	}
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

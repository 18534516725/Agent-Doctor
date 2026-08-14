package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/conversations"
	"github.com/18534516725/Agent-Doctor/internal/dashboard"
	"github.com/18534516725/Agent-Doctor/internal/events"
)

type memoryEventStore struct {
	mu          sync.Mutex
	events      []events.Event
	summary     dashboard.Summary
	snapshot    dashboard.Snapshot
	requests    []conversations.Request
	connections []conversations.ClientConnection
	analysis    conversations.LiveAnalysis
}

func (store *memoryEventStore) InsertEvent(_ context.Context, event events.Event) error {
	store.mu.Lock()
	store.events = append(store.events, event)
	store.mu.Unlock()
	return nil
}

func (store *memoryEventStore) ListSessionEvents(_ context.Context, sessionID string) ([]events.Event, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	result := make([]events.Event, 0)
	for _, event := range store.events {
		if event.SessionID == sessionID {
			result = append(result, event)
		}
	}
	return result, nil
}

func (store *memoryEventStore) ReadOnly() bool { return false }

func (store *memoryEventStore) ListConversationRequests(_ context.Context, limit int, _ string) ([]conversations.Request, error) {
	if limit <= 0 || limit > len(store.requests) {
		limit = len(store.requests)
	}
	return append([]conversations.Request(nil), store.requests[:limit]...), nil
}
func (store *memoryEventStore) GetConversationRequest(_ context.Context, id string) (conversations.Request, error) {
	for _, item := range store.requests {
		if item.ID == id {
			return item, nil
		}
	}
	return conversations.Request{}, errors.New("not found")
}
func (store *memoryEventStore) ListClientConnections(context.Context) ([]conversations.ClientConnection, error) {
	return store.connections, nil
}
func (store *memoryEventStore) LiveConversationAnalysis(context.Context) (conversations.LiveAnalysis, error) {
	return store.analysis, nil
}
func (store *memoryEventStore) DeleteConversationSession(_ context.Context, sessionID string) error {
	for index, item := range store.requests {
		if item.SessionID == sessionID {
			store.requests = append(store.requests[:index], store.requests[index+1:]...)
			return nil
		}
	}
	return errors.New("not found")
}

func (store *memoryEventStore) DashboardSummary(context.Context) (dashboard.Summary, error) {
	return store.summary, nil
}

func (store *memoryEventStore) DashboardSnapshot(context.Context) (dashboard.Snapshot, error) {
	return store.snapshot, nil
}

func TestListenBindsOnlyLoopback(t *testing.T) {
	service := newTestServer(t)
	listener, err := service.Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	host, _, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	if parsed := net.ParseIP(host); parsed == nil || !parsed.IsLoopback() {
		t.Fatalf("listener is not loopback: %s", listener.Addr())
	}
}

func TestAPIRoutesRequireLocalSessionToken(t *testing.T) {
	service := newTestServer(t)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	response := httptest.NewRecorder()
	service.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", response.Code)
	}
}

func TestForeignOriginIsRejected(t *testing.T) {
	service := newTestServer(t)
	request := authenticatedRequest(t, service, http.MethodGet, "/api/v1/dashboard/summary", nil)
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()
	service.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d", response.Code)
	}
}

func TestOversizedEventPayloadReturns413(t *testing.T) {
	service := newTestServer(t)
	event := validServerEvent()
	event.Payload = json.RawMessage(`{"text":"` + strings.Repeat("x", events.MaxPayloadBytes) + `"}`)
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	request := authenticatedRequest(t, service, http.MethodPost, "/api/v1/events", bytes.NewReader(raw))
	response := httptest.NewRecorder()
	service.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestHealthIsVersionedAndDoesNotLeakPaths(t *testing.T) {
	service := newTestServer(t)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	service.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
	body := response.Body.String()
	for _, forbidden := range []string{"doctor.db", "/Users/", "\\Users\\", "databasePath"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("health leaked %q: %s", forbidden, body)
		}
	}
	if !strings.Contains(body, `"apiVersion":"v1"`) || !strings.Contains(body, `"version":"0.1.0-dev"`) {
		t.Fatalf("unversioned health: %s", body)
	}
}

func TestAuthenticatedEventIsAccepted(t *testing.T) {
	store := &memoryEventStore{}
	service, err := New(Config{Version: "0.1.0-dev", Store: store})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(validServerEvent())
	request := authenticatedRequest(t, service, http.MethodPost, "/api/v1/events", bytes.NewReader(raw))
	request.Header.Set("Origin", "http://127.0.0.1:43210")
	response := httptest.NewRecorder()
	service.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || len(store.events) != 1 {
		t.Fatalf("status=%d events=%d body=%s", response.Code, len(store.events), response.Body.String())
	}
}

func TestDashboardSummaryReturnsOnlySafeAggregates(t *testing.T) {
	store := &memoryEventStore{summary: dashboard.Summary{
		Projects: 2, Sessions: 3, Events: 8, ActiveSessions: 1,
		Precision: dashboard.PrecisionCounts{Exact: 5, Estimated: 2, Unavailable: 1},
	}}
	service, err := New(Config{Version: "0.1.0-dev", Store: store})
	if err != nil {
		t.Fatal(err)
	}
	request := authenticatedRequest(t, service, http.MethodGet, "/api/v1/dashboard/summary", nil)
	response := httptest.NewRecorder()
	service.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{`"projects":2`, `"sessions":3`, `"events":8`, `"activeSessions":1`, `"estimated":2`} {
		if !strings.Contains(body, want) {
			t.Fatalf("summary missing %s: %s", want, body)
		}
	}
	for _, forbidden := range []string{"payload", "prompt", "credential", "source code"} {
		if strings.Contains(strings.ToLower(body), forbidden) {
			t.Fatalf("summary leaked %q: %s", forbidden, body)
		}
	}
}

func TestDashboardSnapshotPowersEveryPageWithoutSensitivePayloads(t *testing.T) {
	store := &memoryEventStore{snapshot: dashboard.Snapshot{
		Sessions: []dashboard.Session{{ID: "session-1", Client: "codex", Model: "model-a", Status: "active", EventCount: 4}},
		Costs:    dashboard.Costs{Currency: "USD", ExactMicros: 120000, EstimatedMicros: 30000, Unavailable: 1},
		Memories: dashboard.Memories{Active: 2, Candidate: 1, Disabled: 0},
		Trends:   []dashboard.TrendPoint{{Date: "2026-08-13", Sessions: 1, Events: 4}},
	}}
	service, err := New(Config{Version: "0.1.0-dev", Store: store})
	if err != nil {
		t.Fatal(err)
	}
	request := authenticatedRequest(t, service, http.MethodGet, "/api/v1/dashboard/snapshot", nil)
	response := httptest.NewRecorder()
	service.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, expected := range []string{`"session-1"`, `"exactMicros":120000`, `"active":2`, `"2026-08-13"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("snapshot missing %s: %s", expected, body)
		}
	}
	for _, forbidden := range []string{"payload", "prompt", "fileContents", "credential", "upstream"} {
		if strings.Contains(strings.ToLower(body), strings.ToLower(forbidden)) {
			t.Fatalf("snapshot leaked %q: %s", forbidden, body)
		}
	}
}

func TestSessionEvidenceNeverReturnsEventPayload(t *testing.T) {
	event := validServerEvent()
	event.Payload = json.RawMessage(`{"detail":"payload-secret-marker"}`)
	store := &memoryEventStore{events: []events.Event{event}}
	service, err := New(Config{Version: "0.1.0-dev", Store: store})
	if err != nil {
		t.Fatal(err)
	}
	request := authenticatedRequest(t, service, http.MethodGet, "/api/v1/sessions/session-server-1", nil)
	response := httptest.NewRecorder()
	service.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if strings.Contains(body, "payload-secret-marker") || strings.Contains(body, `"payload"`) {
		t.Fatalf("session evidence leaked payload: %s", body)
	}
	if !strings.Contains(body, `"eventType":"session.started"`) || !strings.Contains(body, `"precision":"exact"`) {
		t.Fatalf("safe event metadata missing: %s", body)
	}
}

func TestConversationAPIExposesCompleteLocalMessagesAndAnalysis(t *testing.T) {
	input := int64(42)
	store := &memoryEventStore{
		requests:    []conversations.Request{{ID: "request-1", SessionID: "session-1", Messages: []conversations.Message{{Role: "user", Content: "完整用户问题"}, {Role: "assistant", Content: "完整模型回复"}}, Usage: conversations.Usage{InputTokens: &input, Precision: "exact"}}},
		connections: []conversations.ClientConnection{{Key: "codex", DisplayName: "Codex", Detected: true, State: "active"}},
		analysis:    conversations.LiveAnalysis{Requests: 1, InputTokens: 42},
	}
	service, err := New(Config{Version: "0.1.0-dev", Store: store})
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"/api/v1/conversations", "/api/v1/conversations/request-1", "/api/v1/connections", "/api/v1/analysis/live"} {
		request := authenticatedRequest(t, service, http.MethodGet, target, nil)
		response := httptest.NewRecorder()
		service.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", target, response.Code, response.Body.String())
		}
	}
	request := authenticatedRequest(t, service, http.MethodGet, "/api/v1/conversations/request-1", nil)
	response := httptest.NewRecorder()
	service.Handler().ServeHTTP(response, request)
	if !strings.Contains(response.Body.String(), "完整用户问题") || !strings.Contains(response.Body.String(), "完整模型回复") {
		t.Fatalf("full messages missing: %s", response.Body.String())
	}
}

func TestDeleteConversationSessionRequiresAuthAndDeletesOnlyTarget(t *testing.T) {
	store := &memoryEventStore{requests: []conversations.Request{{ID: "r1", SessionID: "s1"}, {ID: "r2", SessionID: "s2"}}}
	service, _ := New(Config{Version: "0.1.0-dev", Store: store})
	unauthorized := httptest.NewRecorder()
	service.Handler().ServeHTTP(unauthorized, httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/s1", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d", unauthorized.Code)
	}
	request := authenticatedRequest(t, service, http.MethodDelete, "/api/v1/sessions/s1", nil)
	response := httptest.NewRecorder()
	service.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || len(store.requests) != 1 || store.requests[0].SessionID != "s2" {
		t.Fatalf("delete result status=%d records=%+v", response.Code, store.requests)
	}
}

func TestDaemonTokensAreRandom256BitValues(t *testing.T) {
	first := newTestServer(t).Token()
	second := newTestServer(t).Token()
	if first == second || len(first) < 43 || len(second) < 43 {
		t.Fatalf("tokens do not provide independent 256-bit material")
	}
}

func TestEmbeddedDashboardBootstrapsAProtectedLocalSession(t *testing.T) {
	service := newTestServer(t)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	service.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, `name="agent-doctor-token"`) || !strings.Contains(body, service.Token()) || !strings.Contains(body, `id="root"`) {
		t.Fatalf("dashboard bootstrap incomplete: %s", body)
	}
	if strings.Contains(response.Header().Get("Content-Security-Policy"), "https:") {
		t.Fatalf("dashboard CSP permits remote assets: %s", response.Header().Get("Content-Security-Policy"))
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	service, err := New(Config{Version: "0.1.0-dev", Store: &memoryEventStore{}})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func authenticatedRequest(t *testing.T, service *Server, method, target string, body *bytes.Reader) *http.Request {
	t.Helper()
	var request *http.Request
	if body == nil {
		request = httptest.NewRequest(method, target, nil)
	} else {
		request = httptest.NewRequest(method, target, body)
	}
	request.Header.Set("Authorization", "Bearer "+service.Token())
	request.Header.Set("Content-Type", "application/json")
	return request
}

func validServerEvent() events.Event {
	return events.Event{
		SchemaVersion: 1,
		EventID:       "event-server-1",
		SessionID:     "session-server-1",
		ProjectID:     "project-server-1",
		Timestamp:     time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC),
		Client:        events.ClientRef{Name: "codex", Version: "1.0.0"},
		Model:         events.ModelRef{DisplayName: "public-model"},
		EventType:     events.EventSessionStarted,
		Payload:       json.RawMessage(`{"source":"client-event"}`),
		Provenance:    "client-event",
		Precision:     events.PrecisionExact,
	}
}

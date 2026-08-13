package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/events"
)

type memoryEventStore struct {
	mu     sync.Mutex
	events []events.Event
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

func TestDaemonTokensAreRandom256BitValues(t *testing.T) {
	first := newTestServer(t).Token()
	second := newTestServer(t).Token()
	if first == second || len(first) < 43 || len(second) < 43 {
		t.Fatalf("tokens do not provide independent 256-bit material")
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

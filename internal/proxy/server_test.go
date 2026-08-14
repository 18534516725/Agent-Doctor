package proxy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/conversations"
)

type recordingStore struct {
	mu      sync.Mutex
	records []conversations.Request
}

type privateRecordingStore struct{ recordingStore }

func (*privateRecordingStore) PrivacySettings(context.Context) (conversations.PrivacySettings, error) {
	return conversations.PrivacySettings{CapturePrompts: false, RetentionDays: 30}, nil
}

func (store *recordingStore) SaveConversationRequest(_ context.Context, record conversations.Request) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.records = append(store.records, record)
	return nil
}

func TestProxyForwardsStreamingRequestAndPersistsConversationWithoutCredentials(t *testing.T) {
	const secret = "Bearer synthetic-secret-value"
	var receivedAuthorization, receivedPath, receivedQuery, receivedBody string
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		receivedAuthorization = request.Header.Get("Authorization")
		receivedPath, receivedQuery = request.URL.Path, request.URL.RawQuery
		body, _ := io.ReadAll(request.Body)
		receivedBody = string(body)
		response.Header().Set("Content-Type", "text/event-stream")
		response.WriteHeader(http.StatusOK)
		flusher := response.(http.Flusher)
		_, _ = io.WriteString(response, "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n")
		flusher.Flush()
		_, _ = io.WriteString(response, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":2}}\n\n")
		flusher.Flush()
		_, _ = io.WriteString(response, "data: [DONE]\n\n")
	}))
	defer upstream.Close()
	store := &recordingStore{}
	handler, err := New(Config{UpstreamURL: upstream.URL, Store: store, ClientName: "codex", ProjectID: "project", CaptureLimitBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	proxyServer := httptest.NewServer(handler)
	defer proxyServer.Close()

	body := `{"model":"gpt-test","messages":[{"role":"user","content":"完整消息"}]}`
	request, _ := http.NewRequest(http.MethodPost, proxyServer.URL+"/v1/chat/completions?feature=stream", strings.NewReader(body))
	request.Header.Set("Authorization", secret)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	gotBody, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if receivedAuthorization != secret || receivedPath != "/v1/chat/completions" || receivedQuery != "feature=stream" || receivedBody != body {
		t.Fatalf("forward mismatch auth=%q path=%q query=%q body=%q", receivedAuthorization, receivedPath, receivedQuery, receivedBody)
	}
	if !strings.Contains(string(gotBody), `"content":"hello"`) || !strings.Contains(string(gotBody), "[DONE]") {
		t.Fatalf("stream changed: %s", gotBody)
	}
	if len(store.records) != 1 {
		t.Fatalf("records=%d", len(store.records))
	}
	record := store.records[0]
	if len(record.Messages) != 2 || record.Messages[0].Content != "完整消息" || record.Messages[1].Content != "hello" || record.Usage.InputTokens == nil || *record.Usage.InputTokens != 4 {
		t.Fatalf("capture mismatch: %+v", record)
	}
	encoded, _ := json.Marshal(record)
	if strings.Contains(string(encoded), "synthetic-secret-value") || strings.Contains(strings.ToLower(string(encoded)), "authorization") {
		t.Fatalf("credential persisted: %s", encoded)
	}
}

func TestProxyPersistsNonStreamingJSONResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, `{"model":"gpt-test","choices":[{"message":{"role":"assistant","content":"完整普通响应"}}],"usage":{"prompt_tokens":9,"completion_tokens":5}}`)
	}))
	defer upstream.Close()
	store := &recordingStore{}
	handler, err := New(Config{UpstreamURL: upstream.URL, Store: store, ClientName: "cursor", ProjectID: "project"})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/v1/chat/completions", strings.NewReader(`{"model":"gpt-test","messages":[{"role":"user","content":"问题"}]}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "完整普通响应") {
		t.Fatalf("response changed: code=%d body=%s", response.Code, response.Body.String())
	}
	if len(store.records) != 1 {
		t.Fatalf("records=%d", len(store.records))
	}
	record := store.records[0]
	if len(record.Messages) != 2 || record.Messages[1].Content != "完整普通响应" {
		t.Fatalf("messages=%+v", record.Messages)
	}
	if record.Usage.OutputTokens == nil || *record.Usage.OutputTokens != 5 || record.Usage.Precision != "exact" {
		t.Fatalf("usage=%+v", record.Usage)
	}
}

func TestProxyCanDisableLocalMessagePersistenceWithoutBreakingUsageCapture(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, `{"choices":[{"message":{"content":"private answer"}}],"usage":{"prompt_tokens":2,"completion_tokens":1}}`)
	}))
	defer upstream.Close()
	store := &privateRecordingStore{}
	handler, err := New(Config{UpstreamURL: upstream.URL, Store: store})
	if err != nil {
		t.Fatal(err)
	}
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "http://127.0.0.1/v1/chat/completions", strings.NewReader(`{"messages":[{"role":"user","content":"private question"}]}`)))
	if len(store.records) != 1 || len(store.records[0].Messages) != 0 || store.records[0].Usage.InputTokens == nil {
		t.Fatalf("record=%+v", store.records)
	}
}

func TestProxyRejectsRecursiveOrNonHTTPUpstream(t *testing.T) {
	store := &recordingStore{}
	for _, raw := range []string{"file:///tmp/socket", "http://127.0.0.1:7777"} {
		_, err := New(Config{UpstreamURL: raw, Store: store, ListenAddress: "127.0.0.1:7777"})
		if err == nil {
			t.Fatalf("expected rejection for %s", raw)
		}
	}
}

func TestProxyPropagatesCallerCancellation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		close(started)
		select {
		case <-request.Context().Done():
		case <-release:
		}
	}))
	defer upstream.Close()
	handler, err := New(Config{UpstreamURL: upstream.URL, Store: &recordingStore{}, ProjectID: "project"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/v1/responses", strings.NewReader(`{"model":"gpt-test","input":"hello"}`)).WithContext(ctx)
	done := make(chan struct{})
	go func() { handler.ServeHTTP(httptest.NewRecorder(), request); close(done) }()
	<-started
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		close(release)
		t.Fatal("proxy did not stop the upstream request after caller cancellation")
	}
	close(release)
}

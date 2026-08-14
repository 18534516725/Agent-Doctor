package proxy

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/conversations"
	"github.com/18534516725/Agent-Doctor/internal/events"
)

const defaultCaptureLimit = 16 << 20

type ConversationStore interface {
	SaveConversationRequest(context.Context, conversations.Request) error
}

type connectionStore interface {
	UpsertClientConnection(context.Context, conversations.ClientConnection) error
}

type privacyStore interface {
	PrivacySettings(context.Context) (conversations.PrivacySettings, error)
}

type Config struct {
	UpstreamURL       string
	ListenAddress     string
	Store             ConversationStore
	ClientName        string
	ClientVersion     string
	ProjectID         string
	CaptureLimitBytes int
	HTTPClient        *http.Client
	OnCommitted       func(sessionID string)
}

type Server struct {
	upstream                             *url.URL
	store                                ConversationStore
	client                               *http.Client
	clientName, clientVersion, projectID string
	captureLimit                         int
	onCommitted                          func(string)
}

func New(config Config) (http.Handler, error) {
	if config.Store == nil {
		return nil, errors.New("conversation store is required")
	}
	upstream, err := url.Parse(config.UpstreamURL)
	if err != nil || upstream.Host == "" || (upstream.Scheme != "http" && upstream.Scheme != "https") {
		return nil, errors.New("upstream must be an absolute HTTP(S) URL")
	}
	if config.ListenAddress != "" && sameEndpoint(upstream.Host, config.ListenAddress) {
		return nil, errors.New("proxy upstream cannot point to its own listen address")
	}
	limit := config.CaptureLimitBytes
	if limit == 0 {
		limit = defaultCaptureLimit
	}
	if limit < 1 {
		return nil, errors.New("capture limit must be positive")
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 0}
	}
	clientName := config.ClientName
	if clientName == "" {
		clientName = "unknown-client"
	}
	projectID := config.ProjectID
	if projectID == "" {
		projectID = "local-project"
	}
	server := &Server{upstream: upstream, store: config.Store, client: httpClient, clientName: clientName, clientVersion: config.ClientVersion, projectID: projectID, captureLimit: limit, onCommitted: config.OnCommitted}
	return http.HandlerFunc(server.serveHTTP), nil
}

func sameEndpoint(left, right string) bool {
	normalize := func(value string) string {
		host, port, err := net.SplitHostPort(value)
		if err != nil {
			return strings.ToLower(value)
		}
		if host == "localhost" {
			host = "127.0.0.1"
		}
		return net.JoinHostPort(strings.ToLower(host), port)
	}
	return normalize(left) == normalize(right)
}

func (server *Server) serveHTTP(response http.ResponseWriter, incoming *http.Request) {
	if incoming.Method == http.MethodConnect {
		http.Error(response, "CONNECT is not supported", http.StatusMethodNotAllowed)
		return
	}
	requestBytes, err := readBounded(incoming.Body, server.captureLimit)
	if err != nil {
		http.Error(response, "request body exceeds capture limit", http.StatusRequestEntityTooLarge)
		return
	}
	protocol := detectProtocol(incoming.URL.Path)
	parsedRequest := conversations.ParsedConversation{Usage: conversations.Usage{Precision: "unavailable", Provenance: "request"}}
	if protocol == "anthropic" {
		parsedRequest, _ = conversations.ParseAnthropicRequest(requestBytes)
	} else {
		parsedRequest, _ = conversations.ParseOpenAIRequest(requestBytes)
	}

	target := *server.upstream
	target.Path = joinURLPath(server.upstream.Path, incoming.URL.Path)
	target.RawQuery = incoming.URL.RawQuery
	outgoing, err := http.NewRequestWithContext(incoming.Context(), incoming.Method, target.String(), bytes.NewReader(requestBytes))
	if err != nil {
		http.Error(response, "upstream request unavailable", http.StatusBadGateway)
		return
	}
	copyForwardHeaders(outgoing.Header, incoming.Header)
	started := time.Now().UTC()
	upstreamResponse, err := server.client.Do(outgoing)
	if err != nil {
		if incoming.Context().Err() == nil {
			http.Error(response, "upstream unavailable", http.StatusBadGateway)
		}
		return
	}
	defer upstreamResponse.Body.Close()
	firstByte := time.Now().UTC()
	copyResponseHeaders(response.Header(), upstreamResponse.Header)
	response.WriteHeader(upstreamResponse.StatusCode)

	isStream := strings.Contains(strings.ToLower(upstreamResponse.Header.Get("Content-Type")), "text/event-stream")
	var openAI *conversations.OpenAIStreamAssembler
	var anthropic *conversations.AnthropicStreamAssembler
	var responseCapture bytes.Buffer
	if isStream {
		if protocol == "anthropic" {
			anthropic = conversations.NewAnthropicStreamAssembler(server.captureLimit)
		} else {
			openAI = conversations.NewOpenAIStreamAssembler(server.captureLimit)
		}
	}
	captureEnabled := true
	buffer := make([]byte, 32*1024)
	for {
		count, readErr := upstreamResponse.Body.Read(buffer)
		if count > 0 {
			chunk := buffer[:count]
			if captureEnabled {
				if isStream {
					if protocol == "anthropic" {
						captureEnabled = anthropic.Add(chunk) == nil
					} else {
						captureEnabled = openAI.Add(chunk) == nil
					}
				} else if responseCapture.Len()+len(chunk) <= server.captureLimit {
					_, _ = responseCapture.Write(chunk)
				} else {
					captureEnabled = false
				}
			}
			if _, writeErr := response.Write(chunk); writeErr != nil {
				return
			}
			if flusher, ok := response.(http.Flusher); ok {
				flusher.Flush()
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				return
			}
			break
		}
	}
	completed := time.Now().UTC()
	parsedResponse := conversations.ParsedConversation{Usage: conversations.Usage{Precision: "unavailable", Provenance: "provider-response"}}
	if captureEnabled {
		if isStream {
			if protocol == "anthropic" {
				parsedResponse, _ = anthropic.Complete()
			} else {
				parsedResponse, _ = openAI.Complete()
			}
		} else if protocol == "anthropic" {
			parsedResponse, _ = conversations.ParseAnthropicResponse(responseCapture.Bytes())
		} else {
			parsedResponse, _ = conversations.ParseOpenAIResponse(responseCapture.Bytes())
		}
	}
	server.persist(incoming.Context(), parsedRequest, parsedResponse, protocol, incoming.Method, incoming.URL.Path, upstreamResponse.StatusCode, started, firstByte, completed)
}

func (server *Server) persist(ctx context.Context, requestPart, responsePart conversations.ParsedConversation, protocol, method, path string, status int, started, firstByte, completed time.Time) {
	requestID := randomID("req")
	sessionID := randomID("session")
	messages := append([]conversations.Message{}, requestPart.Messages...)
	messages = append(messages, responsePart.Messages...)
	if settings, ok := server.store.(privacyStore); ok {
		if privacy, err := settings.PrivacySettings(context.WithoutCancel(ctx)); err == nil && !privacy.CapturePrompts {
			messages = nil
		}
	}
	for index := range messages {
		messages[index].ID = randomID("msg")
		messages[index].RequestID = requestID
		messages[index].Sequence = index
		if messages[index].CreatedAt.IsZero() {
			if index < len(requestPart.Messages) {
				messages[index].CreatedAt = started
			} else {
				messages[index].CreatedAt = completed
			}
		}
	}
	model := requestPart.Model
	if model == "" {
		model = responsePart.Model
	}
	if model == "" {
		model = "unknown-model"
	}
	cost := conversations.Cost{Currency: "USD", Precision: "unavailable", Provenance: "no-price-catalog"}
	record := conversations.Request{ID: requestID, SessionID: sessionID, ProjectID: server.projectID,
		Client: events.ClientRef{Name: server.clientName, Version: server.clientVersion}, Model: events.ModelRef{DisplayName: model}, Protocol: protocol,
		Method: method, Path: path, StatusCode: status, StartedAt: started, CompletedAt: &completed,
		FirstByteMS: firstByte.Sub(started).Milliseconds(), DurationMS: completed.Sub(started).Milliseconds(),
		Usage: responsePart.Usage, Cost: cost, Messages: messages}
	if err := server.store.SaveConversationRequest(context.WithoutCancel(ctx), record); err == nil {
		if connections, ok := server.store.(connectionStore); ok {
			heartbeat := completed
			_ = connections.UpsertClientConnection(context.WithoutCancel(ctx), conversations.ClientConnection{
				Key: server.clientName, DisplayName: server.clientName, Detected: true, State: "active",
				Capability: "loopback-proxy", Detail: "正在实时采集模型调用", LastHeartbeatAt: &heartbeat, UpdatedAt: completed,
			})
		}
		if server.onCommitted != nil {
			server.onCommitted(sessionID)
		}
	}
}

func detectProtocol(path string) string {
	if strings.Contains(path, "/messages") {
		return "anthropic"
	}
	return "openai"
}
func joinURLPath(base, request string) string {
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(request, "/")
}
func readBounded(reader io.Reader, limit int) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, errors.New("body too large")
	}
	return data, nil
}
func randomID(prefix string) string {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(raw)
}

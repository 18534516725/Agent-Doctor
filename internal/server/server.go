package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/realtime"
)

type Config struct {
	Version string
	Store   EventStore
	Hub     *realtime.Hub
}

type Server struct {
	version    string
	store      EventStore
	token      string
	handler    http.Handler
	httpServer *http.Server
	hub        *realtime.Hub
}

func New(config Config) (*Server, error) {
	if config.Version == "" {
		return nil, fmt.Errorf("server version is required")
	}
	if config.Store == nil {
		return nil, fmt.Errorf("event store is required")
	}
	token, err := generateSessionToken()
	if err != nil {
		return nil, err
	}
	hub := config.Hub
	if hub == nil {
		hub = realtime.NewHub(256)
	}
	server := &Server{version: config.Version, store: config.Store, token: token, hub: hub}
	server.handler = server.enforceLocalBoundary(server.routes())
	server.httpServer = &http.Server{
		Handler:           server.handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      0, // SSE remains open until the local dashboard disconnects.
		IdleTimeout:       60 * time.Second,
	}
	return server, nil
}

func (server *Server) Handler() http.Handler { return server.handler }
func (server *Server) Token() string         { return server.token }
func (server *Server) PublishConversation(sessionID string) {
	server.hub.Publish(realtime.Event{Kind: "conversation.saved", SessionID: sessionID})
}

func (server *Server) Listen() (net.Listener, error) {
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen on loopback: %w", err)
	}
	return listener, nil
}

func (server *Server) Serve(listener net.Listener) error {
	if listener == nil {
		return fmt.Errorf("listener is required")
	}
	host, _, err := net.SplitHostPort(listener.Addr().String())
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return fmt.Errorf("refusing to serve on a non-loopback listener")
	}
	err = server.httpServer.Serve(listener)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (server *Server) Shutdown(ctx context.Context) error {
	return server.httpServer.Shutdown(ctx)
}

func generateSessionToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate local session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

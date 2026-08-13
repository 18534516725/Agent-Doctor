package server

import (
	"crypto/subtle"
	"net"
	"net/http"
	"net/url"
	"strings"
)

func (server *Server) enforceLocalBoundary(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if origin != "" {
			if !isLoopbackOrigin(origin) {
				writeError(response, http.StatusForbidden, "foreign origin rejected")
				return
			}
			response.Header().Set("Access-Control-Allow-Origin", origin)
			response.Header().Set("Vary", "Origin")
			response.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			response.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		}
		if request.Method == http.MethodOptions {
			response.WriteHeader(http.StatusNoContent)
			return
		}
		if request.URL.Path == "/health" {
			next.ServeHTTP(response, request)
			return
		}
		if !server.validToken(request.Header.Get("Authorization")) {
			response.Header().Set("WWW-Authenticate", "Bearer")
			writeError(response, http.StatusUnauthorized, "local session token required")
			return
		}
		next.ServeHTTP(response, request)
	})
}

func (server *Server) validToken(header string) bool {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	provided := strings.TrimPrefix(header, prefix)
	if len(provided) != len(server.token) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(server.token)) == 1
}

func isLoopbackOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme != "http" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	host := parsed.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

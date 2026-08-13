package server

import (
	"embed"
	"fmt"
	"mime"
	"net/http"
	"path"
	"strings"
)

//go:embed web/index.html web/assets/*
var dashboardFiles embed.FS

func (server *Server) dashboardHome(response http.ResponseWriter, _ *http.Request) {
	raw, err := dashboardFiles.ReadFile("web/index.html")
	if err != nil {
		writeError(response, http.StatusInternalServerError, "dashboard unavailable")
		return
	}
	bootstrap := fmt.Sprintf(`<meta name="agent-doctor-token" content="%s">`, server.token)
	html := strings.Replace(string(raw), "</head>", bootstrap+"</head>", 1)
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.WriteHeader(http.StatusOK)
	_, _ = response.Write([]byte(html))
}

func (server *Server) dashboardAsset(response http.ResponseWriter, request *http.Request) {
	name := strings.TrimPrefix(request.URL.Path, "/assets/")
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		http.NotFound(response, request)
		return
	}
	raw, err := dashboardFiles.ReadFile("web/assets/" + name)
	if err != nil {
		http.NotFound(response, request)
		return
	}
	contentType := mime.TypeByExtension(path.Ext(name))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	response.Header().Set("Content-Type", contentType)
	response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	response.WriteHeader(http.StatusOK)
	_, _ = response.Write(raw)
}

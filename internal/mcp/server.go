package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

const (
	MaxMessageBytes      = 1024 * 1024
	latestStableProtocol = "2025-11-25"
)

var supportedProtocols = map[string]bool{
	"2024-11-05": true,
	"2025-03-26": true,
	"2025-06-18": true,
	"2025-11-25": true,
}

type Server struct {
	version string
	backend ToolBackend
}

func NewServer(version string, backend ToolBackend) *Server {
	return &Server{version: version, backend: backend}
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (server *Server) Serve(ctx context.Context, input io.Reader, output io.Writer) error {
	if server.backend == nil {
		return fmt.Errorf("MCP backend is required")
	}
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 64*1024), MaxMessageBytes)
	encoder := json.NewEncoder(output)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var incoming request
		if err := json.Unmarshal(line, &incoming); err != nil || incoming.JSONRPC != "2.0" || incoming.Method == "" {
			if err := encoder.Encode(errorResponse(nil, -32700, "invalid JSON-RPC message")); err != nil {
				return err
			}
			continue
		}
		outgoing, shouldRespond := server.handle(ctx, incoming)
		if shouldRespond {
			if err := encoder.Encode(outgoing); err != nil {
				return fmt.Errorf("write MCP response: %w", err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		if err := encoder.Encode(errorResponse(nil, -32600, "MCP message exceeds the size limit")); err != nil {
			return err
		}
	}
	return nil
}

func (server *Server) handle(ctx context.Context, incoming request) (response, bool) {
	if len(incoming.ID) == 0 {
		return response{}, false
	}
	switch incoming.Method {
	case "initialize":
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(incoming.Params, &params); err != nil {
			return errorResponse(incoming.ID, -32602, "invalid initialize parameters"), true
		}
		protocol := latestStableProtocol
		if supportedProtocols[params.ProtocolVersion] {
			protocol = params.ProtocolVersion
		}
		return resultResponse(incoming.ID, map[string]any{
			"protocolVersion": protocol,
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]string{"name": "agent-doctor", "title": "Agent Doctor", "version": server.version},
			"instructions":    "All tools are read-only and return sanitized local evidence. Call get_runtime_guidance at task start, after repeated failure or context compaction, and before a final answer. Call get_project_analysis when a task completes, becomes slow, or may have unusual token or cost impact. Follow evidence-backed redirect or verify instructions, treat continue as silence, and never invent unavailable values.",
		}), true
	case "ping":
		return resultResponse(incoming.ID, map[string]any{}), true
	case "tools/list":
		tools := make([]toolDefinition, len(readOnlyTools))
		copy(tools, readOnlyTools)
		return resultResponse(incoming.ID, map[string]any{"tools": tools}), true
	case "tools/call":
		return server.callTool(ctx, incoming), true
	default:
		return errorResponse(incoming.ID, -32601, "method not found"), true
	}
}

func (server *Server) callTool(ctx context.Context, incoming request) response {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(incoming.Params, &params); err != nil {
		return errorResponse(incoming.ID, -32602, "invalid tool parameters")
	}
	tool, ok := lookupTool(params.Name)
	if !ok {
		return errorResponse(incoming.ID, -32602, "unknown read-only tool")
	}
	if params.Arguments == nil {
		params.Arguments = map[string]any{}
	}
	if err := validateToolArguments(tool, params.Arguments); err != nil {
		return errorResponse(incoming.ID, -32602, "invalid tool arguments")
	}
	evidence, err := server.backend.Execute(ctx, params.Name, params.Arguments)
	if err != nil {
		safe := sanitizeEvidence(ToolEvidence{
			Summary: "The local evidence request could not be completed.", Precision: "unavailable",
			Provenance: "local-tool-error", DataLimitNotes: []string{"The underlying error was not exposed."},
		})
		return resultResponse(incoming.ID, map[string]any{
			"content":           []map[string]string{{"type": "text", "text": evidenceText(safe)}},
			"structuredContent": safe,
			"isError":           true,
		})
	}
	evidence = sanitizeEvidence(evidence)
	return resultResponse(incoming.ID, map[string]any{
		"content":           []map[string]string{{"type": "text", "text": evidenceText(evidence)}},
		"structuredContent": evidence,
		"isError":           false,
	})
}

func resultResponse(id json.RawMessage, result any) response {
	return response{JSONRPC: "2.0", ID: id, Result: result}
}

func errorResponse(id json.RawMessage, code int, message string) response {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	return response{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}}
}

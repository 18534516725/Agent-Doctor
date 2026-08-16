package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type fakeBackend struct{}

func (fakeBackend) Execute(_ context.Context, name string, _ map[string]any) (ToolEvidence, error) {
	return ToolEvidence{
		Summary:        "Evidence for " + name + "; Authorization: Bearer synthetic-secret; /Users/example/doctor.db",
		Items:          []EvidenceItem{{Label: "status", Value: "available"}},
		Provenance:     "local-sanitized-events",
		Precision:      "exact",
		DataLimitNotes: []string{"Only explicitly captured local evidence is included."},
	}, nil
}

func TestStdioInitializeAndToolList(t *testing.T) {
	requests := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
	}, "\n") + "\n"
	responses := runProtocol(t, requests)
	if len(responses) != 2 {
		t.Fatalf("responses=%d", len(responses))
	}
	initialize := responseResult(t, responses[0])
	if initialize["protocolVersion"] != "2025-11-25" {
		t.Fatalf("initialize=%v", initialize)
	}
	listed := responseResult(t, responses[1])
	rawTools, ok := listed["tools"].([]any)
	if !ok {
		t.Fatalf("tools=%T", listed["tools"])
	}
	want := []string{
		"get_project_analysis", "get_runtime_guidance", "get_context_capsule", "diagnose_last_task", "get_task_evidence", "compare_clients",
		"compare_models", "get_cost_summary", "get_quota_status", "get_performance_history",
		"recommend_next_action",
	}
	if len(rawTools) != len(want) {
		t.Fatalf("tools=%d want=%d", len(rawTools), len(want))
	}
	for index, raw := range rawTools {
		tool := raw.(map[string]any)
		if tool["name"] != want[index] {
			t.Fatalf("tool[%d]=%v want=%s", index, tool["name"], want[index])
		}
		annotations := tool["annotations"].(map[string]any)
		wantReadOnly := want[index] != "get_runtime_guidance" && want[index] != "get_context_capsule"
		if annotations["readOnlyHint"] != wantReadOnly || annotations["destructiveHint"] != false {
			t.Fatalf("tool annotations are dishonest for %s: %v", want[index], annotations)
		}
	}
}

func TestInitializeTellsClientWhenToReadProjectAnalysis(t *testing.T) {
	request := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}` + "\n"
	result := responseResult(t, runProtocol(t, request)[0])
	instructions, _ := result["instructions"].(string)
	for _, required := range []string{"get_runtime_guidance", "every coding turn", "current project ID", "final answer", "failed tool step", "compaction", "delivery receipt", "cannot force-block"} {
		if !strings.Contains(instructions, required) {
			t.Fatalf("instructions missing %q: %s", required, instructions)
		}
	}
}

func TestToolCallReturnsSanitizedEvidenceMetadata(t *testing.T) {
	request := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_context_capsule","arguments":{"projectId":"project-1"}}}` + "\n"
	responses := runProtocol(t, request)
	result := responseResult(t, responses[0])
	encoded, _ := json.Marshal(result)
	text := string(encoded)
	for _, forbidden := range []string{"synthetic-secret", "/Users/example/doctor.db"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("MCP leaked %q: %s", forbidden, text)
		}
	}
	for _, required := range []string{"provenance", "precision", "dataLimitNotes", "local-sanitized-events"} {
		if !strings.Contains(text, required) {
			t.Fatalf("missing %q: %s", required, text)
		}
	}
}

func TestUnknownToolIsRejected(t *testing.T) {
	request := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"delete_everything","arguments":{}}}` + "\n"
	responses := runProtocol(t, request)
	if len(responses) != 1 || responses[0]["error"] == nil {
		t.Fatalf("unknown tool response=%v", responses)
	}
}

func TestOversizedInputIsRejected(t *testing.T) {
	request := strings.Repeat("x", MaxMessageBytes+1) + "\n"
	responses := runProtocol(t, request)
	if len(responses) != 1 || responses[0]["error"] == nil {
		t.Fatalf("oversized response=%v", responses)
	}
}

func runProtocol(t *testing.T, input string) []map[string]any {
	t.Helper()
	server := NewServer("0.1.0-dev", fakeBackend{})
	var output bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	responses := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		var response map[string]any
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("invalid response %q: %v", line, err)
		}
		responses = append(responses, response)
	}
	return responses
}

func responseResult(t *testing.T, response map[string]any) map[string]any {
	t.Helper()
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("response has no result: %v", response)
	}
	return result
}

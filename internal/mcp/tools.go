package mcp

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/18534516725/Agent-Doctor/internal/privacy"
)

type ToolBackend interface {
	Execute(context.Context, string, map[string]any) (ToolEvidence, error)
}

type EvidenceItem struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type ToolEvidence struct {
	Summary        string         `json:"summary"`
	Items          []EvidenceItem `json:"items"`
	Provenance     string         `json:"provenance"`
	Precision      string         `json:"precision"`
	DataLimitNotes []string       `json:"dataLimitNotes"`
}

type toolDefinition struct {
	Name        string         `json:"name"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Annotations map[string]any `json:"annotations"`
	required    []string
	properties  map[string]string
}

var readOnlyTools = []toolDefinition{
	newTool("get_project_analysis", "Project analysis", "Get the current local project health, risks, cost coverage, efficiency findings, and recommended next actions before answering the user.", map[string]string{}, nil),
	newTool("get_context_capsule", "Context capsule", "Get a token-bounded, provenance-labelled context capsule for the current project.", map[string]string{"projectId": "string", "currentFile": "string", "budget": "number"}, []string{"projectId"}),
	newTool("diagnose_last_task", "Diagnose last task", "Diagnose the latest captured task using sanitized local evidence.", map[string]string{"sessionId": "string"}, []string{"sessionId"}),
	newTool("get_task_evidence", "Task evidence", "Get the bounded evidence timeline for one captured task.", map[string]string{"sessionId": "string"}, []string{"sessionId"}),
	newTool("compare_clients", "Compare clients", "Compare observed outcomes across selected AI clients without exposing prompts or source code.", map[string]string{"projectId": "string", "clients": "array"}, []string{"projectId"}),
	newTool("compare_models", "Compare models", "Compare observed outcomes across selected public model names.", map[string]string{"projectId": "string", "models": "array"}, []string{"projectId"}),
	newTool("get_cost_summary", "Cost summary", "Get exact, estimated, or unavailable usage cost with source versions.", map[string]string{"projectId": "string", "sessionId": "string", "from": "string", "to": "string"}, []string{"projectId"}),
	newTool("get_quota_status", "Quota status", "Get locally observed quota status; unavailable fields are never guessed.", map[string]string{"client": "string"}, nil),
	newTool("get_performance_history", "Performance history", "Get bounded task outcome and latency history for a project.", map[string]string{"projectId": "string", "limit": "number"}, []string{"projectId"}),
	newTool("recommend_next_action", "Recommend next action", "Recommend a safe next action from current evidence and approved validations.", map[string]string{"sessionId": "string"}, []string{"sessionId"}),
}

func newTool(name, title, description string, properties map[string]string, required []string) toolDefinition {
	schemaProperties := make(map[string]any, len(properties))
	for key, kind := range properties {
		schema := map[string]any{"type": kind}
		switch kind {
		case "string":
			schema["maxLength"] = 256
		case "number":
			schema["minimum"] = 1
			schema["maximum"] = 2000
		case "array":
			schema["items"] = map[string]any{"type": "string", "maxLength": 128}
			schema["maxItems"] = 10
		}
		schemaProperties[key] = schema
	}
	schema := map[string]any{"type": "object", "properties": schemaProperties, "additionalProperties": false}
	if len(required) > 0 {
		schema["required"] = required
	}
	return toolDefinition{
		Name: name, Title: title, Description: description, InputSchema: schema,
		Annotations: map[string]any{
			"readOnlyHint": true, "destructiveHint": false, "idempotentHint": true, "openWorldHint": false,
		},
		required: required, properties: properties,
	}
}

func lookupTool(name string) (toolDefinition, bool) {
	for _, tool := range readOnlyTools {
		if tool.Name == name {
			return tool, true
		}
	}
	return toolDefinition{}, false
}

func validateToolArguments(tool toolDefinition, arguments map[string]any) error {
	for _, required := range tool.required {
		if value, ok := arguments[required]; !ok || value == "" {
			return fmt.Errorf("required argument %q is missing", required)
		}
	}
	for key, value := range arguments {
		kind, ok := tool.properties[key]
		if !ok {
			return fmt.Errorf("unknown argument %q", key)
		}
		switch kind {
		case "string":
			text, ok := value.(string)
			if !ok || len(text) > 256 {
				return fmt.Errorf("argument %q must be a bounded string", key)
			}
		case "number":
			number, ok := value.(float64)
			if !ok || number < 1 || number > 2000 {
				return fmt.Errorf("argument %q is outside its allowed range", key)
			}
		case "array":
			values, ok := value.([]any)
			if !ok || len(values) > 10 {
				return fmt.Errorf("argument %q must be a bounded array", key)
			}
			for _, item := range values {
				text, ok := item.(string)
				if !ok || len(text) > 128 {
					return fmt.Errorf("argument %q contains an invalid item", key)
				}
			}
		}
	}
	return nil
}

var (
	unixAbsolutePathPattern    = regexp.MustCompile(`(?:^|[\s(])/(?:[^/\s]+/)+[^\s),;]*`)
	windowsAbsolutePathPattern = regexp.MustCompile(`(?i)\b[A-Z]:\\(?:[^\\\r\n]+\\)*[^\s,;]*`)
)

func sanitizeEvidence(evidence ToolEvidence) ToolEvidence {
	evidence.Summary = boundedPublicText(evidence.Summary, 32*1024)
	if evidence.Provenance == "" {
		evidence.Provenance = "local-evidence-unavailable"
	}
	evidence.Provenance = boundedPublicText(evidence.Provenance, 256)
	switch evidence.Precision {
	case "exact", "estimated", "unavailable":
	default:
		evidence.Precision = "unavailable"
	}
	if len(evidence.Items) > 100 {
		evidence.Items = evidence.Items[:100]
	}
	for index := range evidence.Items {
		evidence.Items[index].Label = boundedPublicText(evidence.Items[index].Label, 128)
		evidence.Items[index].Value = boundedPublicText(evidence.Items[index].Value, 4096)
	}
	if len(evidence.DataLimitNotes) == 0 {
		evidence.DataLimitNotes = []string{"No compatible local telemetry was available; no value was guessed."}
	}
	if len(evidence.DataLimitNotes) > 20 {
		evidence.DataLimitNotes = evidence.DataLimitNotes[:20]
	}
	for index := range evidence.DataLimitNotes {
		evidence.DataLimitNotes[index] = boundedPublicText(evidence.DataLimitNotes[index], 1024)
	}
	return evidence
}

func boundedPublicText(text string, limit int) string {
	filtered := privacy.FilterText(text)
	filtered = unixAbsolutePathPattern.ReplaceAllString(filtered, " [LOCAL_PATH]")
	filtered = windowsAbsolutePathPattern.ReplaceAllString(filtered, "[LOCAL_PATH]")
	if len(filtered) > limit {
		filtered = filtered[:limit] + "…"
	}
	return strings.TrimSpace(filtered)
}

func evidenceText(evidence ToolEvidence) string {
	lines := []string{evidence.Summary}
	for _, item := range evidence.Items {
		lines = append(lines, "- "+item.Label+": "+item.Value)
	}
	lines = append(lines, "Provenance: "+evidence.Provenance, "Precision: "+evidence.Precision)
	notes := append([]string(nil), evidence.DataLimitNotes...)
	sort.Strings(notes)
	for _, note := range notes {
		lines = append(lines, "Data limit: "+note)
	}
	return strings.Join(lines, "\n")
}

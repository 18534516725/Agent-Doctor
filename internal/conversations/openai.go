package conversations

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func ParseOpenAIRequest(body []byte) (ParsedConversation, error) {
	var raw struct {
		Model        string           `json:"model"`
		Instructions any              `json:"instructions"`
		Input        any              `json:"input"`
		Messages     []map[string]any `json:"messages"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return ParsedConversation{}, fmt.Errorf("decode OpenAI request: %w", err)
	}
	result := ParsedConversation{Model: raw.Model, Messages: []Message{}, Usage: unavailableUsage("request")}
	if instructions := stringContent(raw.Instructions); instructions != "" {
		result.Messages = append(result.Messages, Message{Role: "system", Content: instructions})
	}
	for _, item := range raw.Messages {
		role, _ := item["role"].(string)
		message := Message{Role: role, Content: stringContent(item["content"])}
		if role == "tool" {
			message.ToolName, _ = item["name"].(string)
			if message.ToolName == "" {
				message.ToolName, _ = item["tool_call_id"].(string)
			}
		}
		result.Messages = append(result.Messages, message)
	}
	if len(raw.Messages) == 0 {
		appendOpenAIInput(&result.Messages, raw.Input)
	}
	return result, nil
}

func appendOpenAIInput(messages *[]Message, input any) {
	if text, ok := input.(string); ok {
		*messages = append(*messages, Message{Role: "user", Content: text})
		return
	}
	items, ok := input.([]any)
	if !ok {
		return
	}
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		role, _ := item["role"].(string)
		if role == "" {
			role = "user"
		}
		*messages = append(*messages, Message{Role: role, Content: stringContent(item["content"])})
	}
}

type openAITool struct{ name, arguments string }

type OpenAIStreamAssembler struct {
	decoder sseDecoder
	text    strings.Builder
	tools   map[int]*openAITool
	usage   Usage
}

func NewOpenAIStreamAssembler(limit int) *OpenAIStreamAssembler {
	return &OpenAIStreamAssembler{decoder: sseDecoder{limit: limit}, tools: map[int]*openAITool{}, usage: unavailableUsage("provider-stream")}
}

func (assembler *OpenAIStreamAssembler) Add(chunk []byte) error {
	return assembler.decoder.add(chunk, assembler.consume)
}

func (assembler *OpenAIStreamAssembler) consume(payload []byte) error {
	var event map[string]any
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("decode OpenAI stream event: %w", err)
	}
	if delta, ok := event["delta"].(string); ok {
		assembler.text.WriteString(delta)
	}
	if choices, ok := event["choices"].([]any); ok {
		for _, rawChoice := range choices {
			choice, _ := rawChoice.(map[string]any)
			delta, _ := choice["delta"].(map[string]any)
			if text, ok := delta["content"].(string); ok {
				assembler.text.WriteString(text)
			}
			calls, _ := delta["tool_calls"].([]any)
			for _, rawCall := range calls {
				call, _ := rawCall.(map[string]any)
				index := int(number(call["index"]))
				tool := assembler.tools[index]
				if tool == nil {
					tool = &openAITool{}
					assembler.tools[index] = tool
				}
				function, _ := call["function"].(map[string]any)
				if name, ok := function["name"].(string); ok {
					tool.name += name
				}
				if arguments, ok := function["arguments"].(string); ok {
					tool.arguments += arguments
				}
			}
		}
	}
	if usage, ok := event["usage"].(map[string]any); ok {
		assembler.captureUsage(usage)
	}
	if response, ok := event["response"].(map[string]any); ok {
		if usage, ok := response["usage"].(map[string]any); ok {
			assembler.captureUsage(usage)
		}
	}
	return nil
}

func (assembler *OpenAIStreamAssembler) captureUsage(raw map[string]any) {
	assembler.usage = Usage{Precision: "exact", Provenance: "provider-stream"}
	assembler.usage.InputTokens = intPointer(raw, "prompt_tokens", "input_tokens")
	assembler.usage.OutputTokens = intPointer(raw, "completion_tokens", "output_tokens")
	if detail, ok := raw["prompt_tokens_details"].(map[string]any); ok {
		assembler.usage.CachedTokens = intPointer(detail, "cached_tokens")
	}
	if assembler.usage.CachedTokens == nil {
		if detail, ok := raw["input_tokens_details"].(map[string]any); ok {
			assembler.usage.CachedTokens = intPointer(detail, "cached_tokens")
		}
	}
	if detail, ok := raw["completion_tokens_details"].(map[string]any); ok {
		assembler.usage.ReasoningTokens = intPointer(detail, "reasoning_tokens")
	}
	if assembler.usage.ReasoningTokens == nil {
		if detail, ok := raw["output_tokens_details"].(map[string]any); ok {
			assembler.usage.ReasoningTokens = intPointer(detail, "reasoning_tokens")
		}
	}
}

func (assembler *OpenAIStreamAssembler) Complete() (ParsedConversation, error) {
	if err := assembler.decoder.flush(assembler.consume); err != nil {
		return ParsedConversation{}, err
	}
	result := ParsedConversation{Messages: []Message{}, Usage: assembler.usage}
	if assembler.text.Len() > 0 {
		result.Messages = append(result.Messages, Message{Role: "assistant", Content: assembler.text.String()})
	}
	indexes := make([]int, 0, len(assembler.tools))
	for index := range assembler.tools {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	for _, index := range indexes {
		tool := assembler.tools[index]
		result.Messages = append(result.Messages, Message{Role: "tool", ToolName: tool.name, ToolPayloadJSON: tool.arguments})
	}
	return result, nil
}

func number(value any) int64 {
	if raw, ok := value.(float64); ok {
		return int64(raw)
	}
	return 0
}
func intPointer(raw map[string]any, keys ...string) *int64 {
	for _, key := range keys {
		if value, ok := raw[key].(float64); ok {
			result := int64(value)
			return &result
		}
	}
	return nil
}

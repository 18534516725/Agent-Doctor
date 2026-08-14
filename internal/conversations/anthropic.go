package conversations

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func ParseAnthropicRequest(body []byte) (ParsedConversation, error) {
	var raw struct {
		Model    string           `json:"model"`
		System   any              `json:"system"`
		Messages []map[string]any `json:"messages"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return ParsedConversation{}, fmt.Errorf("decode Anthropic request: %w", err)
	}
	result := ParsedConversation{Model: raw.Model, Messages: []Message{}, Usage: unavailableUsage("request")}
	if system := stringContent(raw.System); system != "" {
		result.Messages = append(result.Messages, Message{Role: "system", Content: system})
	}
	for _, item := range raw.Messages {
		role, _ := item["role"].(string)
		result.Messages = append(result.Messages, Message{Role: role, Content: stringContent(item["content"])})
	}
	return result, nil
}

type anthropicBlock struct {
	kind, name string
	text       strings.Builder
	input      strings.Builder
}
type AnthropicStreamAssembler struct {
	decoder sseDecoder
	blocks  map[int]*anthropicBlock
	usage   Usage
}

func NewAnthropicStreamAssembler(limit int) *AnthropicStreamAssembler {
	return &AnthropicStreamAssembler{decoder: sseDecoder{limit: limit}, blocks: map[int]*anthropicBlock{}, usage: unavailableUsage("provider-stream")}
}
func (assembler *AnthropicStreamAssembler) Add(chunk []byte) error {
	return assembler.decoder.add(chunk, assembler.consume)
}

func (assembler *AnthropicStreamAssembler) consume(payload []byte) error {
	var event map[string]any
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("decode Anthropic stream event: %w", err)
	}
	typeName, _ := event["type"].(string)
	index := int(number(event["index"]))
	switch typeName {
	case "message_start":
		if message, ok := event["message"].(map[string]any); ok {
			if usage, ok := message["usage"].(map[string]any); ok {
				assembler.captureUsage(usage)
			}
		}
	case "message_delta":
		if usage, ok := event["usage"].(map[string]any); ok {
			assembler.captureUsage(usage)
		}
	case "content_block_start":
		content, _ := event["content_block"].(map[string]any)
		block := &anthropicBlock{}
		block.kind, _ = content["type"].(string)
		block.name, _ = content["name"].(string)
		if text, ok := content["text"].(string); ok {
			block.text.WriteString(text)
		}
		if input, ok := content["input"]; ok {
			if encoded, err := json.Marshal(input); err == nil && string(encoded) != "{}" {
				block.input.Write(encoded)
			}
		}
		assembler.blocks[index] = block
	case "content_block_delta":
		block := assembler.blocks[index]
		if block == nil {
			block = &anthropicBlock{kind: "text"}
			assembler.blocks[index] = block
		}
		delta, _ := event["delta"].(map[string]any)
		if text, ok := delta["text"].(string); ok {
			block.text.WriteString(text)
		}
		if partial, ok := delta["partial_json"].(string); ok {
			block.input.WriteString(partial)
		}
	}
	return nil
}

func (assembler *AnthropicStreamAssembler) captureUsage(raw map[string]any) {
	if assembler.usage.Precision != "exact" {
		assembler.usage = Usage{Precision: "exact", Provenance: "provider-stream"}
	}
	if value := intPointer(raw, "input_tokens"); value != nil {
		assembler.usage.InputTokens = value
	}
	if value := intPointer(raw, "output_tokens"); value != nil {
		assembler.usage.OutputTokens = value
	}
	if value := intPointer(raw, "cache_read_input_tokens"); value != nil {
		assembler.usage.CachedTokens = value
	}
}

func (assembler *AnthropicStreamAssembler) Complete() (ParsedConversation, error) {
	if err := assembler.decoder.flush(assembler.consume); err != nil {
		return ParsedConversation{}, err
	}
	result := ParsedConversation{Messages: []Message{}, Usage: assembler.usage}
	indexes := make([]int, 0, len(assembler.blocks))
	for index := range assembler.blocks {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	for _, index := range indexes {
		block := assembler.blocks[index]
		if block.kind == "tool_use" {
			result.Messages = append(result.Messages, Message{Role: "tool", ToolName: block.name, ToolPayloadJSON: block.input.String()})
		} else {
			result.Messages = append(result.Messages, Message{Role: "assistant", Content: block.text.String()})
		}
	}
	return result, nil
}

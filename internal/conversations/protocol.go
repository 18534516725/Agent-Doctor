package conversations

import (
	"bytes"
	"errors"
	"strings"
)

type ParsedConversation struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Usage    Usage     `json:"usage"`
}

type sseDecoder struct {
	buffer bytes.Buffer
	total  int
	limit  int
}

func (decoder *sseDecoder) add(chunk []byte, consume func([]byte) error) error {
	if decoder.limit <= 0 {
		return errors.New("capture byte limit must be positive")
	}
	decoder.total += len(chunk)
	if decoder.total > decoder.limit {
		return errors.New("protocol capture exceeds configured byte limit")
	}
	_, _ = decoder.buffer.Write(chunk)
	for {
		data := decoder.buffer.Bytes()
		index := bytes.Index(data, []byte("\n\n"))
		separatorLength := 2
		if index < 0 {
			index = bytes.Index(data, []byte("\r\n\r\n"))
			separatorLength = 4
		}
		if index < 0 {
			return nil
		}
		block := append([]byte(nil), data[:index]...)
		decoder.buffer.Next(index + separatorLength)
		if payload := sseData(block); len(payload) > 0 && string(payload) != "[DONE]" {
			if err := consume(payload); err != nil {
				return err
			}
		}
	}
}

func (decoder *sseDecoder) flush(consume func([]byte) error) error {
	if decoder.buffer.Len() == 0 {
		return nil
	}
	block := append([]byte(nil), decoder.buffer.Bytes()...)
	decoder.buffer.Reset()
	payload := sseData(block)
	if len(payload) == 0 || string(payload) == "[DONE]" {
		return nil
	}
	return consume(payload)
}

func sseData(block []byte) []byte {
	lines := strings.Split(strings.ReplaceAll(string(block), "\r\n", "\n"), "\n")
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, "data:") {
			parts = append(parts, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	return []byte(strings.Join(parts, "\n"))
}

func unavailableUsage(provenance string) Usage {
	return Usage{Precision: "unavailable", Provenance: provenance}
}

func stringContent(value any) string {
	switch item := value.(type) {
	case string:
		return item
	case []any:
		var result strings.Builder
		for _, raw := range item {
			part, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			for _, key := range []string{"text", "content", "output_text"} {
				if text, ok := part[key].(string); ok {
					result.WriteString(text)
					break
				}
			}
		}
		return result.String()
	default:
		return ""
	}
}

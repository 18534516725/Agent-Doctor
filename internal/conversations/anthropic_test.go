package conversations

import "testing"

func TestParseAnthropicRequestPreservesSystemAndMessages(t *testing.T) {
	got, err := ParseAnthropicRequest([]byte(`{"model":"claude-test","system":"be careful","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.Model != "claude-test" || len(got.Messages) != 2 || got.Messages[0].Role != "system" || got.Messages[1].Content != "hello" {
		t.Fatalf("unexpected request: %+v", got)
	}
}

func TestAnthropicStreamAssemblerReconstructsTextToolAndUsage(t *testing.T) {
	assembler := NewAnthropicStreamAssembler(1 << 20)
	stream := "data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":9,\"cache_read_input_tokens\":3}}}\n\n" +
		"data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"hello \"}}\n\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"world\"}}\n\n" +
		"data: {\"type\":\"content_block_start\",\"index\":1,\"content_block\":{\"type\":\"tool_use\",\"name\":\"shell\",\"input\":{}}}\n\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"cmd\\\":\\\"pwd\\\"}\"}}\n\n" +
		"data: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":6}}\n\n"
	if err := assembler.Add([]byte(stream)); err != nil {
		t.Fatal(err)
	}
	got, err := assembler.Complete()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 2 || got.Messages[0].Content != "hello world" || got.Messages[1].ToolName != "shell" || got.Messages[1].ToolPayloadJSON != `{"cmd":"pwd"}` {
		t.Fatalf("unexpected messages: %+v", got.Messages)
	}
	if got.Usage.InputTokens == nil || *got.Usage.InputTokens != 9 || got.Usage.OutputTokens == nil || *got.Usage.OutputTokens != 6 || got.Usage.CachedTokens == nil || *got.Usage.CachedTokens != 3 {
		t.Fatalf("unexpected usage: %+v", got.Usage)
	}
}

func TestMissingUsageIsUnavailableNotZero(t *testing.T) {
	assembler := NewAnthropicStreamAssembler(1024)
	if err := assembler.Add([]byte("data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"ok\"}}\n\n")); err != nil {
		t.Fatal(err)
	}
	got, err := assembler.Complete()
	if err != nil {
		t.Fatal(err)
	}
	if got.Usage.Precision != "unavailable" || got.Usage.InputTokens != nil || got.Usage.OutputTokens != nil {
		t.Fatalf("unexpected usage: %+v", got.Usage)
	}
}

func TestParseAnthropicResponsePreservesCompleteNonStreamingMessage(t *testing.T) {
	got, err := ParseAnthropicResponse([]byte(`{"model":"claude-test","content":[{"type":"text","text":"完整回复"}],"usage":{"input_tokens":7,"output_tokens":4}}`))
	if err != nil || len(got.Messages) != 1 || got.Messages[0].Content != "完整回复" || got.Usage.OutputTokens == nil || *got.Usage.OutputTokens != 4 {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

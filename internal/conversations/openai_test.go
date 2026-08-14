package conversations

import "testing"

func TestParseOpenAIChatRequestPreservesMessagesAndTools(t *testing.T) {
	got, err := ParseOpenAIRequest([]byte(`{"model":"gpt-test","messages":[{"role":"system","content":"be exact"},{"role":"user","content":[{"type":"text","text":"hello"}]}],"tools":[{"type":"function","function":{"name":"lookup"}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.Model != "gpt-test" || len(got.Messages) != 2 || got.Messages[1].Content != "hello" {
		t.Fatalf("unexpected request: %+v", got)
	}
}

func TestOpenAIStreamAssemblerReconstructsTextToolsAndExactUsage(t *testing.T) {
	assembler := NewOpenAIStreamAssembler(1 << 20)
	chunks := []string{
		"data: {\"choices\":[{\"delta\":{\"role\":\"assistant\",\"content\":\"hello \"}}]}\n\n",
		"data: {\"choices\":[{\"delta\":{\"content\":\"world\",\"tool_calls\":[{\"index\":0,\"function\":{\"name\":\"lookup\",\"arguments\":\"{\\\"q\\\":\\\"\"}}]}}]}\n\n",
		"data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"x\\\"}\"}}]}}],\"usage\":{\"prompt_tokens\":12,\"completion_tokens\":7,\"prompt_tokens_details\":{\"cached_tokens\":4},\"completion_tokens_details\":{\"reasoning_tokens\":2}}}\n\n",
		"data: [DONE]\n\n",
	}
	for _, chunk := range chunks {
		if err := assembler.Add([]byte(chunk)); err != nil {
			t.Fatal(err)
		}
	}
	got, err := assembler.Complete()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 2 || got.Messages[0].Content != "hello world" || got.Messages[1].ToolName != "lookup" || got.Messages[1].ToolPayloadJSON != `{"q":"x"}` {
		t.Fatalf("unexpected messages: %+v", got.Messages)
	}
	if got.Usage.InputTokens == nil || *got.Usage.InputTokens != 12 || got.Usage.CachedTokens == nil || *got.Usage.CachedTokens != 4 || got.Usage.ReasoningTokens == nil || *got.Usage.ReasoningTokens != 2 || got.Usage.Precision != "exact" {
		t.Fatalf("unexpected usage: %+v", got.Usage)
	}
}

func TestOpenAIResponsesStreamPreservesOutputText(t *testing.T) {
	assembler := NewOpenAIStreamAssembler(1 << 20)
	stream := "event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"first \"}\n\n" +
		"data: {\"type\":\"response.output_text.delta\",\"delta\":\"second\"}\n\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":5,\"output_tokens\":3}}}\n\n"
	if err := assembler.Add([]byte(stream)); err != nil {
		t.Fatal(err)
	}
	got, err := assembler.Complete()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 1 || got.Messages[0].Content != "first second" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestOpenAIAssemblerRejectsOversizedCapture(t *testing.T) {
	assembler := NewOpenAIStreamAssembler(8)
	if err := assembler.Add([]byte("data: too-large\n\n")); err == nil {
		t.Fatal("expected bounded capture error")
	}
}

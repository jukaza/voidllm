package proxy

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestAnthropicRequestToOpenAI_TextMessage(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"model": "test-model",
		"max_tokens": 100,
		"system": "You are helpful.",
		"messages": [{"role": "user", "content": "hello"}]
	}`)

	openaiBody, envelope, err := anthropicRequestToOpenAI(body)
	if err != nil {
		t.Fatalf("anthropicRequestToOpenAI: %v", err)
	}
	if envelope.Model != "test-model" {
		t.Errorf("model = %q, want test-model", envelope.Model)
	}

	var doc map[string]any
	if err := json.Unmarshal(openaiBody, &doc); err != nil {
		t.Fatalf("unmarshal openai body: %v", err)
	}
	messages, ok := doc["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages = %v, want 2 entries", doc["messages"])
	}
	first, _ := messages[0].(map[string]any)
	if first["role"] != "system" {
		t.Errorf("messages[0].role = %v, want system", first["role"])
	}
}

func TestOpenAIResponseToAnthropic_Text(t *testing.T) {
	t.Parallel()

	openai := []byte(`{
		"id": "chatcmpl-1",
		"object": "chat.completion",
		"model": "test-model",
		"choices": [{
			"index": 0,
			"message": {"role": "assistant", "content": "hi there"},
			"finish_reason": "stop"
		}],
		"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
	}`)

	out, err := openAIResponseToAnthropic(openai, "test-model")
	if err != nil {
		t.Fatalf("openAIResponseToAnthropic: %v", err)
	}

	var resp anthropicSkinMessageResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("unmarshal anthropic response: %v", err)
	}
	if resp.Type != "message" || resp.Role != "assistant" {
		t.Errorf("type/role = %q/%q, want message/assistant", resp.Type, resp.Role)
	}
	if len(resp.Content) != 1 || resp.Content[0].Text != "hi there" {
		t.Errorf("content = %+v, want hi there", resp.Content)
	}
	if resp.StopReason != "end_turn" {
		t.Errorf("stop_reason = %q, want end_turn", resp.StopReason)
	}
	if resp.Usage.InputTokens != 10 || resp.Usage.OutputTokens != 5 {
		t.Errorf("usage = %+v, want 10/5", resp.Usage)
	}
}

func TestMessagesHandler_NonStreaming(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl-1",
			"object": "chat.completion",
			"model": "test-model",
			"choices": [{
				"index": 0,
				"message": {"role": "assistant", "content": "pong"},
				"finish_reason": "stop"
			}],
			"usage": {"prompt_tokens": 3, "completion_tokens": 2, "total_tokens": 5}
		}`))
	}))
	t.Cleanup(upstream.Close)

	handler := NewProxyHandler(testRegistry(t, upstream.URL), slog.New(slog.NewTextHandler(io.Discard, nil)))
	app := fiber.New()
	app.Post("/v1/messages", handler.MessagesHandler)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{
		"model": "test-model",
		"max_tokens": 50,
		"messages": [{"role": "user", "content": "ping"}]
	}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, testTimeout)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}

	var out anthropicSkinMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Type != "message" || len(out.Content) == 0 || out.Content[0].Text != "pong" {
		t.Errorf("unexpected response: %+v", out)
	}
}

func TestCountTokensHandler(t *testing.T) {
	t.Parallel()

	handler := NewProxyHandler(testRegistry(t, "http://127.0.0.1:1"), slog.New(slog.NewTextHandler(io.Discard, nil)))
	app := fiber.New()
	app.Post("/v1/messages/count_tokens", handler.CountTokensHandler)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", strings.NewReader(`{
		"model": "claude-sonnet",
		"messages": [{"role": "user", "content": "hello world"}]
	}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, testTimeout)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var out anthropicSkinCountResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.InputTokens <= 0 {
		t.Errorf("input_tokens = %d, want > 0", out.InputTokens)
	}
}

func TestMessagesHandler_Streaming(t *testing.T) {
	t.Parallel()

	// Mock upstream that speaks OpenAI chat/completions SSE.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		lines := []string{
			`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"test-model","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"test-model","choices":[{"index":0,"delta":{"content":"Hel"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"test-model","choices":[{"index":0,"delta":{"content":"lo"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"test-model","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
		}
		for _, l := range lines {
			_, _ = io.WriteString(w, l+"\n\n")
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	t.Cleanup(upstream.Close)

	handler := NewProxyHandler(testRegistry(t, upstream.URL), slog.New(slog.NewTextHandler(io.Discard, nil)))
	app := fiber.New()
	app.Post("/v1/messages", handler.MessagesHandler)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{
		"model": "test-model",
		"max_tokens": 50,
		"stream": true,
		"messages": [{"role": "user", "content": "ping"}]
	}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, testTimeout)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("content-type = %q, want text/event-stream", ct)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	body := string(raw)

	// The stream must be re-skinned into Anthropic Messages API SSE events.
	for _, want := range []string{
		"event: message_start",
		"event: content_block_start",
		`"type":"text_delta"`,
		`"text":"Hel"`,
		`"text":"lo"`,
		"event: content_block_stop",
		"event: message_delta",
		`"stop_reason":"end_turn"`,
		"event: message_stop",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("stream body missing %q\n---\n%s", want, body)
		}
	}
	// No OpenAI-shaped leakage.
	if strings.Contains(body, "chat.completion.chunk") {
		t.Errorf("stream body leaked OpenAI chunk shape:\n%s", body)
	}
	if strings.Contains(body, "[DONE]") {
		t.Errorf("stream body leaked OpenAI [DONE] sentinel:\n%s", body)
	}
}

func TestAnthropicStreamConverter_ToolCall(t *testing.T) {
	t.Parallel()

	conv := newAnthropicStreamConverter("test-model")
	var b strings.Builder
	write := func(lines [][]byte) {
		for _, l := range lines {
			if l == nil {
				b.WriteByte('\n')
				continue
			}
			b.Write(l)
			b.WriteByte('\n')
		}
	}

	write(conv.Convert([]byte(`data: {"choices":[{"index":0,"delta":{"role":"assistant"}}]}`)))
	write(conv.Convert([]byte(`data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"get_weather","arguments":""}}]}}]}`)))
	write(conv.Convert([]byte(`data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"city\":"}}]}}]}`)))
	write(conv.Convert([]byte(`data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"SF\"}"}}]}}]}`)))
	write(conv.Convert([]byte(`data: {"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`)))
	write(conv.Convert([]byte(`data: [DONE]`)))

	out := b.String()
	for _, want := range []string{
		`"type":"tool_use"`,
		`"name":"get_weather"`,
		`"type":"input_json_delta"`,
		`"stop_reason":"tool_use"`,
		"event: message_stop",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("tool-call stream missing %q\n---\n%s", want, out)
		}
	}
}


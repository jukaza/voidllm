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

func TestMessagesHandler_RejectsStreaming(t *testing.T) {
	t.Parallel()

	handler := NewProxyHandler(testRegistry(t, "http://127.0.0.1:1"), slog.New(slog.NewTextHandler(io.Discard, nil)))
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

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}


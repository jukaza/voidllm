package proxy

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/jukaza/tavo/internal/jsonx"
)

// randMessageID returns a short random hex string used to build Anthropic
// streaming message IDs (msg_...). On the extremely unlikely event that the
// system RNG fails, it falls back to a fixed marker rather than panicking.
func randMessageID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "proxygen0000"
	}
	return hex.EncodeToString(b[:])
}

// anthropicStreamConverter converts an OpenAI-shaped SSE event stream
// (data: {chat.completion.chunk} ... data: [DONE]) into the Anthropic
// Messages API SSE event stream expected by Anthropic clients such as
// Claude Code. It is stateful and must not be reused across requests.
//
// The proxy pipeline always speaks OpenAI chat/completions to upstream
// providers (MessagesHandler overrides the upstream path to
// "chat/completions"). When the response skin is Anthropic and the upstream
// streams, each OpenAI SSE line is fed through Convert to obtain zero or more
// Anthropic SSE lines to write to the client. Finish emits any trailing
// events (content_block_stop, message_delta, message_stop) that were not yet
// produced when the stream ended.
type anthropicStreamConverter struct {
	model string

	messageStarted bool // message_start emitted
	textOpen       bool // text content block (index 0) is open
	finished       bool // message_delta + message_stop emitted

	// nextIndex is the next Anthropic content-block index to assign.
	nextIndex int
	// textIndex is the content-block index of the text block (0 when present).
	textIndex int

	// toolBlocks maps an OpenAI tool-call array index to the Anthropic
	// content-block index assigned to it, and tracks whether that block is
	// currently open.
	toolBlocks map[int]*anthropicToolBlockState

	stopReason   string
	inputTokens  int
	outputTokens int
}

type anthropicToolBlockState struct {
	blockIndex int
	open       bool
}

func newAnthropicStreamConverter(model string) *anthropicStreamConverter {
	return &anthropicStreamConverter{
		model:      model,
		stopReason: "end_turn",
		toolBlocks: make(map[int]*anthropicToolBlockState),
	}
}

// openAIStreamChunk is the minimal shape of an OpenAI streaming chunk needed
// to drive the Anthropic conversion.
type openAIStreamChunk struct {
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content   string           `json:"content"`
			ToolCalls []openAIToolCall `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Convert accepts a single raw upstream SSE line (OpenAI-shaped) and returns
// the Anthropic SSE lines to write to the client. Each returned []byte is a
// complete line (no trailing newline). The caller writes each line followed by
// "\n"; a blank line between logical events is inserted by the caller's event
// framing (Convert returns "event: X" and "data: Y" pairs plus a nil separator
// where a blank line must be emitted).
//
// A nil element in the returned slice signals "write a blank line" (SSE event
// terminator).
func (s *anthropicStreamConverter) Convert(line []byte) [][]byte {
	trimmed := strings.TrimSpace(string(line))
	if trimmed == "" {
		return nil
	}
	if !strings.HasPrefix(trimmed, "data:") {
		return nil
	}
	payload := strings.TrimSpace(trimmed[len("data:"):])
	if payload == "" {
		return nil
	}
	if payload == "[DONE]" {
		return s.Finish()
	}

	var chunk openAIStreamChunk
	if err := jsonx.Unmarshal([]byte(payload), &chunk); err != nil {
		// Unparseable chunk — skip it rather than corrupt the stream.
		return nil
	}
	if chunk.Model != "" && s.model == "" {
		s.model = chunk.Model
	}

	var out [][]byte
	if !s.messageStarted {
		if chunk.Usage != nil {
			s.inputTokens = chunk.Usage.PromptTokens
		}
		out = append(out, s.emitMessageStart()...)
	}

	if chunk.Usage != nil {
		if chunk.Usage.PromptTokens > 0 {
			s.inputTokens = chunk.Usage.PromptTokens
		}
		if chunk.Usage.CompletionTokens > 0 {
			s.outputTokens = chunk.Usage.CompletionTokens
		}
	}

	for _, choice := range chunk.Choices {
		// Text delta.
		if choice.Delta.Content != "" {
			if !s.textOpen {
				out = append(out, s.emitTextBlockStart()...)
			}
			out = append(out, s.event("content_block_delta", map[string]any{
				"type":  "content_block_delta",
				"index": s.textIndex,
				"delta": map[string]any{"type": "text_delta", "text": choice.Delta.Content},
			})...)
		}

		// Tool-call deltas.
		for _, tc := range choice.Delta.ToolCalls {
			out = append(out, s.emitToolCallDelta(tc)...)
		}

		if choice.FinishReason != nil {
			s.stopReason = mapOpenAIFinishToAnthropic(*choice.FinishReason)
		}
	}

	return out
}

// Finish emits any pending closing events. It is idempotent.
func (s *anthropicStreamConverter) Finish() [][]byte {
	if s.finished {
		return nil
	}
	var out [][]byte
	if !s.messageStarted {
		out = append(out, s.emitMessageStart()...)
	}
	// Close text block if open.
	if s.textOpen {
		out = append(out, s.event("content_block_stop", map[string]any{
			"type":  "content_block_stop",
			"index": s.textIndex,
		})...)
		s.textOpen = false
	}
	// Close any open tool blocks.
	for _, tb := range s.toolBlocks {
		if tb.open {
			out = append(out, s.event("content_block_stop", map[string]any{
				"type":  "content_block_stop",
				"index": tb.blockIndex,
			})...)
			tb.open = false
		}
	}
	// message_delta with stop reason and output usage.
	out = append(out, s.event("message_delta", map[string]any{
		"type": "message_delta",
		"delta": map[string]any{
			"stop_reason":   s.stopReason,
			"stop_sequence": nil,
		},
		"usage": map[string]any{"output_tokens": s.outputTokens},
	})...)
	// message_stop.
	out = append(out, s.event("message_stop", map[string]any{
		"type": "message_stop",
	})...)
	s.finished = true
	return out
}

func (s *anthropicStreamConverter) emitMessageStart() [][]byte {
	s.messageStarted = true
	msgID := fmt.Sprintf("msg_%s", randMessageID())
	return s.event("message_start", map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":            msgID,
			"type":          "message",
			"role":          "assistant",
			"model":         s.model,
			"content":       []any{},
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]any{
				"input_tokens":  s.inputTokens,
				"output_tokens": 0,
			},
		},
	})
}

func (s *anthropicStreamConverter) emitTextBlockStart() [][]byte {
	s.textIndex = s.nextIndex
	s.nextIndex++
	s.textOpen = true
	return s.event("content_block_start", map[string]any{
		"type":          "content_block_start",
		"index":         s.textIndex,
		"content_block": map[string]any{"type": "text", "text": ""},
	})
}

func (s *anthropicStreamConverter) emitToolCallDelta(tc openAIToolCall) [][]byte {
	// The OpenAI array index disambiguates concurrent tool calls. When absent,
	// default to 0 (single tool call).
	oaIdx := 0
	if tc.Index != nil {
		oaIdx = *tc.Index
	}

	tb, ok := s.toolBlocks[oaIdx]
	var out [][]byte
	if !ok {
		// New tool block. Close the text block first if it is open (Anthropic
		// requires the current block to be stopped before a new one starts).
		if s.textOpen {
			out = append(out, s.event("content_block_stop", map[string]any{
				"type":  "content_block_stop",
				"index": s.textIndex,
			})...)
			s.textOpen = false
		}
		tb = &anthropicToolBlockState{blockIndex: s.nextIndex}
		s.nextIndex++
		s.toolBlocks[oaIdx] = tb
		out = append(out, s.event("content_block_start", map[string]any{
			"type":  "content_block_start",
			"index": tb.blockIndex,
			"content_block": map[string]any{
				"type":  "tool_use",
				"id":    tc.ID,
				"name":  tc.Function.Name,
				"input": map[string]any{},
			},
		})...)
		tb.open = true
	}

	if tc.Function.Arguments != "" {
		out = append(out, s.event("content_block_delta", map[string]any{
			"type":  "content_block_delta",
			"index": tb.blockIndex,
			"delta": map[string]any{
				"type":         "input_json_delta",
				"partial_json": tc.Function.Arguments,
			},
		})...)
	}
	return out
}

// event marshals an Anthropic SSE event and returns the two SSE lines plus a
// blank-line separator marker (nil). The caller writes each non-nil element
// followed by "\n", and a bare "\n" for nil elements.
func (s *anthropicStreamConverter) event(eventType string, data map[string]any) [][]byte {
	payload, err := jsonx.Marshal(data)
	if err != nil {
		return nil
	}
	return [][]byte{
		[]byte("event: " + eventType),
		append([]byte("data: "), payload...),
		nil, // blank line: SSE event terminator
	}
}

func mapOpenAIFinishToAnthropic(reason string) string {
	switch reason {
	case "length":
		return "max_tokens"
	case "tool_calls", "function_call":
		return "tool_use"
	case "content_filter":
		return "end_turn"
	case "stop":
		return "end_turn"
	default:
		return "end_turn"
	}
}

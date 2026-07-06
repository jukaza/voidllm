package tokens

import (
	"strings"

	"github.com/jukaza/tavo/internal/jsonx"
)

// InputTextFromRequest extracts billable input text from an OpenAI-compatible
// chat request body.
func InputTextFromRequest(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var doc struct {
		Messages []struct {
			Role    string           `json:"role"`
			Content jsonx.RawMessage `json:"content"`
		} `json:"messages"`
		Prompt string           `json:"prompt"`
		Input  jsonx.RawMessage `json:"input"`
		System jsonx.RawMessage `json:"system"`
		Tools  jsonx.RawMessage `json:"tools"`
	}
	if jsonx.Unmarshal(body, &doc) != nil {
		return string(body)
	}

	var b strings.Builder
	appendRawField(&b, doc.System)
	for _, msg := range doc.Messages {
		if msg.Role != "" {
			b.WriteString(msg.Role)
			b.WriteByte('\n')
		}
		appendContentValue(&b, msg.Content)
	}
	if doc.Prompt != "" {
		b.WriteString(doc.Prompt)
	}
	appendRawField(&b, doc.Input)
	appendRawField(&b, doc.Tools)

	out := b.String()
	if out == "" {
		return string(body)
	}
	return out
}

// OutputTextFromResponse extracts assistant text from a non-streaming
// OpenAI-shaped chat completion body.
func OutputTextFromResponse(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var resp struct {
		Choices []struct {
			Message struct {
				Content   jsonx.RawMessage `json:"content"`
				ToolCalls jsonx.RawMessage `json:"tool_calls"`
			} `json:"message"`
			Text string `json:"text"`
		} `json:"choices"`
		Output []struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if jsonx.Unmarshal(body, &resp) != nil {
		return ""
	}

	var b strings.Builder
	for _, ch := range resp.Choices {
		if ch.Text != "" {
			b.WriteString(ch.Text)
		}
		appendContentValue(&b, ch.Message.Content)
		appendRawField(&b, ch.Message.ToolCalls)
	}
	for _, out := range resp.Output {
		for _, part := range out.Content {
			b.WriteString(part.Text)
		}
	}
	return b.String()
}

// AppendStreamDeltaContent accumulates assistant output from one SSE data line.
func AppendStreamDeltaContent(line []byte, b *strings.Builder) {
	if b == nil || !strings.HasPrefix(string(line), "data: ") {
		return
	}
	data := line[6:]
	if len(data) == 0 || string(data) == "[DONE]" {
		return
	}
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
			} `json:"delta"`
			Text string `json:"text"`
		} `json:"choices"`
	}
	if jsonx.Unmarshal(data, &chunk) != nil {
		return
	}
	for _, ch := range chunk.Choices {
		b.WriteString(ch.Text)
		b.WriteString(ch.Delta.Content)
		b.WriteString(ch.Delta.ReasoningContent)
	}
}

func appendContentValue(b *strings.Builder, raw jsonx.RawMessage) {
	if len(raw) == 0 {
		return
	}
	var s string
	if jsonx.Unmarshal(raw, &s) == nil {
		b.WriteString(s)
		return
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if jsonx.Unmarshal(raw, &parts) == nil {
		for _, p := range parts {
			b.WriteString(p.Text)
		}
	}
}

func appendRawField(b *strings.Builder, raw jsonx.RawMessage) {
	if len(raw) == 0 {
		return
	}
	var s string
	if jsonx.Unmarshal(raw, &s) == nil {
		b.WriteString(s)
		return
	}
	b.Write(raw)
}